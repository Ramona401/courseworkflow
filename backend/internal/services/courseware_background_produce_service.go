package services

// courseware_background_produce_service.go — 课件背景图库生产入口服务（批次3新建）
//
// 职责（挂在 CoursewareBackgroundService 上，与列表/选择/秒换共用一个服务）：
//   1. GenerateSet：「AI生成一套背景」——按提示词调豆包出封面+内页两张16:9(2560×1440)，
//      内页提示词强制追加"浅色低对比、适合做底纹"约束；下载落盘→上OSS→建个人集→
//      （带courseware_id时）自动选中并秒换已生成页。traceCtx 走现有积分追踪钩子，
//      与课件配图生成共用 courseware_image_gen 场景（同一计费口径）。
//   2. UploadSet：「上传一套」——两张图(≤5MB, JPG/PNG/WEBP)落盘→上OSS→个人集→自动选中。
//   3. DeleteSet：个人集删除=归档(archived)，不删OSS对象——已选课件存的是URL快照，引用安全。
//   4. PromoteSet：admin把个人集升级为系统图库（scope→system, user_id→NULL）。
//
// 个人集配额：每人激活态上限 models.CWBgPersonalMaxSets(20)，超出拒绝并提示先归档旧集。
// 存储路径：/uploads/courseware-assets/backgrounds/{userID}/（复用既有Nginx映射与OSS上传链路）。
// 已知边界：AI生成两张为串行，第2张失败时第1张已落盘的本地文件成为孤儿（量极小，可定期清理）。

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 常量 ====================

const (
	// cwBgUploadDir 背景图物理存储根目录（复用 courseware-assets 的Nginx映射）
	cwBgUploadDir = "/www/wwwroot/tedna/uploads/courseware-assets/backgrounds"
	// cwBgURLPrefix 背景图本地URL前缀（/uploads/ 前缀保证 UploadAssetToOSS 可直接消费）
	cwBgURLPrefix = "/uploads/courseware-assets/backgrounds/"
	// cwBgMaxFileSize 手动上传单张图上限 5MB
	cwBgMaxFileSize = 5 * 1024 * 1024
	// cwBgImageSize AI生成背景图固定尺寸：16:9 高清（2560×1440=3,686,400像素，恰好满足豆包最低像素要求）
	cwBgImageSize = "2560x1440"
	// cwBgContentSuffix 内页提示词强制追加的底纹约束（保证内页文字可读性）
	cwBgContentSuffix = "。要求：浅色、低对比度、整体柔和留白，适合作为课件内页底纹背景，画面不包含任何文字。"
	// cwBgPersonalSortOrder 个人集默认排序值
	cwBgPersonalSortOrder = 500
)

// cwBgAllowedMime 手动上传允许的图片MIME → 扩展名
var cwBgAllowedMime = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

var cwBgLog = logger.WithModule("courseware_background")

// ==================== AI生成一套背景 ====================

// GenerateSet AI生成一套背景（封面+内页两张）→ 上OSS → 建个人集 → 可选自动选中
func (s *CoursewareBackgroundService) GenerateSet(ctx context.Context, userID string, req *models.GenerateBackgroundSetRequest) (*models.BackgroundSetProduceResult, error) {
	coverPrompt := strings.TrimSpace(req.CoverPrompt)
	contentPrompt := strings.TrimSpace(req.ContentPrompt)
	if coverPrompt == "" || contentPrompt == "" {
		return nil, fmt.Errorf("封面与内页提示词均不能为空")
	}
	// 1. 个人集配额校验
	if err := s.checkPersonalQuota(ctx, userID); err != nil {
		return nil, err
	}
	// 2. 加载豆包图片生成配置（与课件配图共用 courseware_image_gen 场景模型）
	imgCfg, err := ai.GetImageConfig(s.cfg.GetAESKey())
	if err != nil {
		return nil, fmt.Errorf("图片生成API未配置: %w", err)
	}
	// v198：解析操作者所属学校ID，供模型境内/境外分流判定（背景图生成，走豆包ai.GenerateImage不经文本分流；填SchoolID仅供消费按学校归属）
	bgSchoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_image_gen",
		UserID:    &userID,
		SchoolID:  schoolIDPtr(bgSchoolID),
	}
	// 3. 内页提示词强制追加底纹约束（不依赖前端传入，后端保底）
	contentPrompt = contentPrompt + cwBgContentSuffix

	// 4. 串行生成两张（避免并发打豆包API），逐张下载落盘
	coverRes, err := ai.GenerateImage(ctx, imgCfg, coverPrompt, cwBgImageSize, 1, "", traceCtx)
	if err != nil {
		return nil, fmt.Errorf("封面背景生成失败: %w", err)
	}
	if len(coverRes.URLs) == 0 {
		return nil, fmt.Errorf("封面背景生成未返回有效URL")
	}
	coverLocal, err := s.downloadBgImage(userID, coverRes.URLs[0], "cover")
	if err != nil {
		return nil, fmt.Errorf("下载封面背景失败: %w", err)
	}
	contentRes, err := ai.GenerateImage(ctx, imgCfg, contentPrompt, cwBgImageSize, 1, "", traceCtx)
	if err != nil {
		return nil, fmt.Errorf("内页背景生成失败: %w", err)
	}
	if len(contentRes.URLs) == 0 {
		return nil, fmt.Errorf("内页背景生成未返回有效URL")
	}
	contentLocal, err := s.downloadBgImage(userID, contentRes.URLs[0], "content")
	if err != nil {
		return nil, fmt.Errorf("下载内页背景失败: %w", err)
	}

	// 5. 默认集名 + 描述（取封面提示词前40字便于辨认）
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "AI背景·" + time.Now().Format("0102-1504")
	}
	descRunes := []rune(coverPrompt)
	if len(descRunes) > 40 {
		descRunes = descRunes[:40]
	}
	desc := "AI生成 · " + string(descRunes)

	// 6. 上OSS + 建个人集 + 可选自动选中（公共收尾）
	return s.finishProduceSet(ctx, userID, name, desc, coverLocal, contentLocal, req.CoursewareID)
}

// ==================== 上传一套背景 ====================

// UploadSet 手动上传一套背景（两张图）→ 上OSS → 建个人集 → 可选自动选中
func (s *CoursewareBackgroundService) UploadSet(
	ctx context.Context,
	userID string, name string, coursewareID string,
	coverFile multipart.File, coverHdr *multipart.FileHeader,
	contentFile multipart.File, contentHdr *multipart.FileHeader,
) (*models.BackgroundSetProduceResult, error) {
	// 1. 个人集配额校验
	if err := s.checkPersonalQuota(ctx, userID); err != nil {
		return nil, err
	}
	// 2. 两张图分别落盘（内部做大小与MIME校验）
	coverLocal, err := s.saveBgUpload(userID, coverFile, coverHdr, "cover")
	if err != nil {
		return nil, fmt.Errorf("封面图: %w", err)
	}
	contentLocal, err := s.saveBgUpload(userID, contentFile, contentHdr, "content")
	if err != nil {
		return nil, fmt.Errorf("内页图: %w", err)
	}
	// 3. 默认集名
	if strings.TrimSpace(name) == "" {
		name = "上传背景·" + time.Now().Format("0102-1504")
	}
	// 4. 上OSS + 建个人集 + 可选自动选中（公共收尾）
	return s.finishProduceSet(ctx, userID, strings.TrimSpace(name), "手动上传", coverLocal, contentLocal, coursewareID)
}

// ==================== 删除（归档）与升级系统图库 ====================

// DeleteSet 归档图集（删除语义）。权限：个人集=本人或admin；系统集=仅admin。
// 不删OSS对象——已选课件存URL快照，归档后照常显示。
func (s *CoursewareBackgroundService) DeleteSet(ctx context.Context, userID string, role string, setID string) error {
	set, err := repository.GetBackgroundSetByID(ctx, setID)
	if err != nil {
		return fmt.Errorf("背景图集不存在: %w", err)
	}
	if set.Scope == models.CWBgScopePersonal {
		isOwner := set.UserID != nil && *set.UserID == userID
		if !isOwner && role != "admin" {
			return fmt.Errorf("无权删除他人的个人背景图集")
		}
	} else {
		if role != "admin" {
			return fmt.Errorf("无权删除系统背景图集")
		}
	}
	if err := repository.ArchiveBackgroundSet(ctx, setID); err != nil {
		return err
	}
	cwBgLog.Info("背景图集已归档", "set_id", setID, "name", set.Name, "scope", set.Scope, "operator", userID)
	return nil
}

// PromoteSet admin把个人集升级为系统图库（幂等：已是system直接返回）
func (s *CoursewareBackgroundService) PromoteSet(ctx context.Context, setID string) (*models.CoursewareBackgroundSet, error) {
	set, err := repository.GetBackgroundSetByID(ctx, setID)
	if err != nil {
		return nil, fmt.Errorf("背景图集不存在: %w", err)
	}
	if set.Scope == models.CWBgScopeSystem {
		return set, nil // 已是系统图库，幂等返回
	}
	if set.Status != models.CWBgStatusActive {
		return nil, fmt.Errorf("已归档的图集不能升级为系统图库")
	}
	if err := repository.PromoteSetToSystem(ctx, setID); err != nil {
		return nil, err
	}
	updated, err := repository.GetBackgroundSetByID(ctx, setID)
	if err != nil {
		return nil, fmt.Errorf("升级成功但回读失败: %w", err)
	}
	cwBgLog.Info("个人背景图集已升级为系统图库", "set_id", setID, "name", updated.Name)
	return updated, nil
}

// ==================== 内部辅助 ====================

// checkPersonalQuota 个人集配额校验：激活态≥上限时拒绝
func (s *CoursewareBackgroundService) checkPersonalQuota(ctx context.Context, userID string) error {
	n, err := repository.CountActivePersonalSets(ctx, userID)
	if err != nil {
		return err
	}
	if n >= models.CWBgPersonalMaxSets {
		return fmt.Errorf("个人背景图集已达上限%d个，请先在图库中删除不再使用的个人图集", models.CWBgPersonalMaxSets)
	}
	return nil
}

// finishProduceSet 生产收尾公共段：两张本地图上OSS → 建个人集 → 可选自动选中课件
// 自动选中失败不回滚集创建（集已可用，老师可手动点选），仅记WARN
func (s *CoursewareBackgroundService) finishProduceSet(
	ctx context.Context,
	userID string, name string, desc string,
	coverLocal string, contentLocal string, coursewareID string,
) (*models.BackgroundSetProduceResult, error) {
	ossSvc := NewOSSService(s.cfg)
	coverPublic, err := ossSvc.UploadAssetToOSS(coverLocal)
	if err != nil {
		return nil, fmt.Errorf("封面背景上传云盘失败: %w", err)
	}
	contentPublic, err := ossSvc.UploadAssetToOSS(contentLocal)
	if err != nil {
		return nil, fmt.Errorf("内页背景上传云盘失败: %w", err)
	}

	set := &models.CoursewareBackgroundSet{
		Name:             name,
		Description:      desc,
		Scope:            models.CWBgScopePersonal,
		UserID:           &userID,
		CoverOssURL:      coverLocal,
		CoverPublicURL:   coverPublic,
		ContentOssURL:    contentLocal,
		ContentPublicURL: contentPublic,
		Status:           models.CWBgStatusActive,
		SortOrder:        cwBgPersonalSortOrder,
	}
	if err := repository.CreateBackgroundSet(ctx, set); err != nil {
		return nil, err
	}
	cwBgLog.Info("个人背景图集已创建", "set_id", set.ID, "name", set.Name, "user_id", userID)

	result := &models.BackgroundSetProduceResult{Set: set}
	// 带课件ID时自动选中（复用既有 SelectBackground：含归属校验+快照写入+秒换已生成页）
	if strings.TrimSpace(coursewareID) != "" {
		sel, selErr := s.SelectBackground(ctx, coursewareID, userID, set.ID)
		if selErr != nil {
			cwBgLog.Warn("图集创建成功但自动选中失败（老师可在图库中手动点选）",
				"set_id", set.ID, "courseware_id", coursewareID, "error", selErr)
		} else {
			result.Selection = sel
		}
	}
	return result, nil
}

// downloadBgImage 下载豆包返回的远程图片到本地背景目录，返回 /uploads/ 本地URL
func (s *CoursewareBackgroundService) downloadBgImage(userID string, imageURL string, slot string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("下载图片失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载图片HTTP错误: %d", resp.StatusCode)
	}
	// 按Content-Type推断扩展名，默认png
	contentType := resp.Header.Get("Content-Type")
	ext := ".png"
	if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
		ext = ".jpg"
	} else if strings.Contains(contentType, "webp") {
		ext = ".webp"
	}
	dir := filepath.Join(cwBgUploadDir, userID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建背景目录失败: %w", err)
	}
	storedName := fmt.Sprintf("%d_bg_%s%s", time.Now().UnixMilli(), slot, ext)
	fullPath := filepath.Join(dir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, resp.Body); err != nil {
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	return cwBgURLPrefix + userID + "/" + storedName, nil
}

// saveBgUpload 手动上传单张图落盘：大小≤5MB + MIME白名单(JPG/PNG/WEBP)，返回 /uploads/ 本地URL
func (s *CoursewareBackgroundService) saveBgUpload(userID string, file multipart.File, header *multipart.FileHeader, slot string) (string, error) {
	if header.Size > cwBgMaxFileSize {
		return "", fmt.Errorf("文件过大，最大支持5MB（当前%.1fMB）", float64(header.Size)/(1024*1024))
	}
	mimeType := header.Header.Get("Content-Type")
	ext, ok := cwBgAllowedMime[mimeType]
	if !ok {
		// Content-Type缺失或不识别时按扩展名兜底
		switch strings.ToLower(filepath.Ext(header.Filename)) {
		case ".jpg", ".jpeg":
			ext = ".jpg"
		case ".png":
			ext = ".png"
		case ".webp":
			ext = ".webp"
		default:
			return "", fmt.Errorf("不支持的图片格式，支持JPG/PNG/WEBP")
		}
	}
	dir := filepath.Join(cwBgUploadDir, userID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建背景目录失败: %w", err)
	}
	storedName := fmt.Sprintf("%d_bgup_%s%s", time.Now().UnixMilli(), slot, ext)
	fullPath := filepath.Join(dir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("保存文件失败: %w", err)
	}
	return cwBgURLPrefix + userID + "/" + storedName, nil
}
