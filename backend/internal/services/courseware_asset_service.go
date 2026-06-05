package services

// courseware_asset_service.go — 课件多媒体资产服务
//
// v0.42 多媒体：AI图片生成 + 手动上传 + 插入HTML + 列表 + 删除
// v0.42.1 新增：AI视频生成（异步提交+状态查询）
// 风格锚点轮2新增：
//   - GenerateImage 生成链路自动套用课件级风格锚点（refImageURL 图生图 + VAOCI 拼 prompt）
//   - DeleteAsset 删除锚点资产时连带清空课件锚点引用（契约②：避免悬空引用）
//   - resolveAssetPublicURL 取资产公网URL辅助（契约③：优先 public_oss_url，否则本地路径补域名前缀）
//   - SetStyleAnchor / ClearStyleAnchor 设/清锚点业务方法（一步式同步：取URL→提取VAOCI→落库）
//
// 功能：
//   - GenerateImage: 调用豆包API生成图片，下载保存到本地（已设锚点时自动套用风格DNA）
//   - GenerateVideo: 调用豆包API提交视频生成任务（异步，返回task_id）
//   - QueryVideoStatus: 查询视频生成任务状态，成功时下载保存到本地
//   - UploadAsset: 手动上传图片到本地磁盘
//   - InsertImageToPage: 将图片插入到页面HTML中（替换占位符或追加）
//   - ListPageAssets / ListCoursewareAssets: 查询图片/视频资产
//   - DeleteAsset: 删除图片/视频资产（磁盘+数据库+连带删OSS+连带清锚点）
//   - SetStyleAnchor / ClearStyleAnchor: 设/清课件级风格锚点
//
// 存储路径: /uploads/courseware-assets/{courseware_id}/p{num}/{timestamp}_{name}
// Nginx映射: /uploads/courseware-assets/ → 磁盘目录
// 后续扩展（v0.43发布桥）: 发布到edu平台时批量上传OSS并替换URL

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 常量 ====================

const (
	// CWAssetUploadDir 课件图片/视频物理存储根目录
	CWAssetUploadDir = "/www/wwwroot/tedna/uploads/courseware-assets"

	// CWAssetURLPrefix URL前缀（Nginx alias映射）
	CWAssetURLPrefix = "/uploads/courseware-assets/"

	// CWAssetMaxSize 单张图片最大5MB
	CWAssetMaxSize = 5 * 1024 * 1024

	// cwAssetPublicHost 本地资产转公网URL的域名前缀（供豆包API下载 / 多模态读图 / 图生图）
	cwAssetPublicHost = "https://workflow.pkuailab.com"
)

// 允许的图片MIME类型
var cwAssetAllowedMimeTypes = map[string]bool{
	"image/jpeg":    true,
	"image/jpg":     true,
	"image/png":     true,
	"image/webp":    true,
	"image/gif":     true,
	"image/svg+xml": true,
}

// MIME → 扩展名
var cwAssetMimeToExt = map[string]string{
	"image/jpeg":    ".jpg",
	"image/jpg":     ".jpg",
	"image/png":     ".png",
	"image/webp":    ".webp",
	"image/gif":     ".gif",
	"image/svg+xml": ".svg",
}

// 文件名安全化正则
var cwAssetSafeNameRe = regexp.MustCompile(`[^a-zA-Z0-9\p{Han}_-]`)

var cwAssetLog = logger.WithModule("courseware_asset")

// ==================== 服务定义 ====================

// CoursewareAssetService 课件多媒体资产服务
type CoursewareAssetService struct {
	cfg *config.Config
}

// NewCoursewareAssetService 创建课件多媒体资产服务
func NewCoursewareAssetService(cfg *config.Config) *CoursewareAssetService {
	return &CoursewareAssetService{cfg: cfg}
}

// ==================== 公网URL辅助（契约③）====================

// resolveAssetPublicURL 取资产的公网可访问URL
// 优先级：public_oss_url（已上云）> 本地 /uploads/ 路径补域名前缀 > 已是 http(s) 则原样
// 返回空字符串表示无法解析出公网URL（既无公网地址也非本地路径）
func resolveAssetPublicURL(asset *models.CoursewareAsset) string {
	if asset == nil {
		return ""
	}
	// 1. 已上云：直接用 OSS 公网URL
	if strings.TrimSpace(asset.PublicOSSURL) != "" {
		return strings.TrimSpace(asset.PublicOSSURL)
	}
	u := strings.TrimSpace(asset.OssURL)
	if u == "" {
		return ""
	}
	// 2. 已是公网地址：原样返回
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	// 3. 本地 /uploads/ 路径：补域名前缀转公网URL
	if strings.HasPrefix(u, "/uploads/") {
		return cwAssetPublicHost + u
	}
	return ""
}

// ==================== AI图片生成 ====================

// GenerateImageServiceRequest AI图片生成请求参数
type GenerateImageServiceRequest struct {
	CoursewareID  string // 课件ID
	PageNumber    int    // 页码
	PlaceholderID string // 占位符ID（可选）
	Prompt        string // 生成提示词
	Size          string // 图片尺寸（如 2560x1440, 1920x1920）
	RefImageURL   string // 参考图URL（图生图模式，可选）
	UserID        string // 操作者ID
}

// GenerateImageServiceResponse AI图片生成响应
type GenerateImageServiceResponse struct {
	AssetID       string   `json:"asset_id"`       // 资产记录ID
	URL           string   `json:"url"`            // 本地存储的图片URL
	OriginalURLs  []string `json:"original_urls"`  // 豆包返回的原始URL列表
	ModelUsed     string   `json:"model_used"`     // 使用的模型
	RevisedPrompt string   `json:"revised_prompt"` // 模型修改后的提示词
}

// GenerateImage 调用豆包API生成图片，下载保存到本地
//
// 风格锚点轮2：若课件已设风格锚点（StyleAnchorAssetID 非空），自动套用：
//   - refImageURL：调用方未显式传 RefImageURL 时，用锚点图公网URL做图生图（视觉一致 + 省token）
//   - prompt：把锚点 VAOCI 索引文本拼到提示词后做文字约束
//   - 锚点图自身重生不自我套用（PlaceholderID 命中锚点图时跳过，避免自我参考）
func (s *CoursewareAssetService) GenerateImage(
	ctx context.Context,
	req *GenerateImageServiceRequest,
) (*GenerateImageServiceResponse, error) {
	// 1. 校验课件和权限
	cw, err := repository.GetCoursewareByID(ctx, req.CoursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != req.UserID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 2. 校验页面
	page, err := repository.GetCoursewarePageByNumber(ctx, req.CoursewareID, req.PageNumber)
	if err != nil {
		return nil, fmt.Errorf("页面不存在: 课件=%s 页码=%d", req.CoursewareID, req.PageNumber)
	}

	// 3. 获取图片生成API配置（从AI配置中心的 courseware_image_gen 场景）
	imgCfg, err := ai.GetImageConfig(s.cfg.GetAESKey())
	if err != nil {
		return nil, fmt.Errorf("图片生成API未配置: %w", err)
	}

	// 4. 准备调用参数
	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_image_gen",
		UserID:    &req.UserID,
	}
	// 确定图片尺寸：用户指定 > 默认1920x1920
	imageSize := req.Size
	if imageSize == "" {
		imageSize = "1920x1920"
	}

	// 4.1 生成所用提示词（可能被锚点VAOCI增强）
	effectivePrompt := req.Prompt

	// 4.2 确定参考图URL：调用方显式传入优先；否则尝试套用风格锚点
	refURL := ""
	if req.RefImageURL != "" {
		// 调用方显式指定参考图（如锚点图重生场景，前端会主动传），尊重其意图
		if strings.HasPrefix(req.RefImageURL, "/uploads/") {
			refURL = cwAssetPublicHost + req.RefImageURL
		} else {
			refURL = req.RefImageURL
		}
	} else if cw.StyleAnchorAssetID != nil && *cw.StyleAnchorAssetID != "" {
		// 课件已设风格锚点，且调用方未显式传参考图 → 自动套用锚点风格DNA
		anchorURL, vaoci := s.resolveStyleAnchorForGen(ctx, cw, req.PlaceholderID)
		if anchorURL != "" {
			refURL = anchorURL
			// VAOCI 索引文本拼进提示词做文字约束（与图生图双重保证风格一致）
			if vaoci != "" {
				// 兼顾风格一致 + 人物/主体一致：VAOCI 已含风格DNA与角色固定形象，整体作为参考约束
				effectivePrompt = req.Prompt + "\n\n【风格与人物一致性约束】请严格保持与参考图一致的视觉风格，并保持画面中人物/主体角色的形象（发型、脸型、服装、配色等）与参考图一致：" + vaoci
			}
			cwAssetLog.Info("生成图片自动套用风格锚点",
				"courseware_id", req.CoursewareID,
				"page_number", req.PageNumber,
				"anchor_asset_id", *cw.StyleAnchorAssetID,
				"has_vaoci", vaoci != "",
			)
		}
	}

	// 5. 调用豆包API生成图片
	result, err := ai.GenerateImage(ctx, imgCfg, effectivePrompt, imageSize, 1, refURL, traceCtx)
	if err != nil {
		return nil, fmt.Errorf("图片生成失败: %w", err)
	}
	if len(result.URLs) == 0 {
		return nil, fmt.Errorf("图片生成未返回有效URL")
	}

	// 6. 下载第一张图片到本地存储
	imageURL := result.URLs[0]
	localURL, err := s.downloadAndSaveImage(ctx, req.CoursewareID, req.PageNumber, imageURL, req.Prompt)
	if err != nil {
		return nil, fmt.Errorf("下载生成图片失败: %w", err)
	}

	// 7. 写入数据库
	asset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PageID:           &page.ID,
		PlaceholderID:    req.PlaceholderID,
		AssetType:        models.CWAssetTypeImage,
		GenerationPrompt: req.Prompt,
		OssURL:           localURL,
		FileSize:         0,
		MimeType:         "image/png",
		Status:           models.CWAssetStatusUploaded,
	}
	if err := repository.CreateCWAsset(ctx, asset); err != nil {
		return nil, fmt.Errorf("记录图片资产失败: %w", err)
	}

	cwAssetLog.Info("AI图片生成并保存成功",
		"courseware_id", req.CoursewareID,
		"page_number", req.PageNumber,
		"asset_id", asset.ID,
		"model", result.ModelUsed,
		"prompt_len", len(req.Prompt),
	)

	return &GenerateImageServiceResponse{
		AssetID:       asset.ID,
		URL:           localURL,
		OriginalURLs:  result.URLs,
		ModelUsed:     result.ModelUsed,
		RevisedPrompt: result.RevisedPrompt,
	}, nil
}

// resolveStyleAnchorForGen 取课件风格锚点的参考图公网URL + VAOCI（供 GenerateImage 自动套用）
// 返回 ("","")的情形：锚点资产已被删/查不到、锚点图无法解析公网URL、
//
//	或当前生成目标占位符正是锚点图自身（锚点图重生不自我套用）。
func (s *CoursewareAssetService) resolveStyleAnchorForGen(ctx context.Context, cw *models.Courseware, currentPlaceholderID string) (string, string) {
	if cw.StyleAnchorAssetID == nil || *cw.StyleAnchorAssetID == "" {
		return "", ""
	}
	anchorAsset, err := repository.GetCWAssetByID(ctx, *cw.StyleAnchorAssetID)
	if err != nil {
		// 锚点资产查不到（可能被删但未清引用）——本次生成不套用，不报错
		cwAssetLog.Warn("风格锚点资产查询失败，本次生成跳过套用",
			"courseware_id", cw.ID,
			"anchor_asset_id", *cw.StyleAnchorAssetID,
			"error", err,
		)
		return "", ""
	}
	// 锚点图自身重生：若当前生成目标占位符与锚点图占位符一致，则不自我参考
	if currentPlaceholderID != "" && anchorAsset.PlaceholderID == currentPlaceholderID {
		return "", ""
	}
	anchorURL := resolveAssetPublicURL(anchorAsset)
	if anchorURL == "" {
		return "", ""
	}
	return anchorURL, strings.TrimSpace(cw.StyleAnchorVAOCI)
}

// downloadAndSaveImage 下载远程图片并保存到本地磁盘
func (s *CoursewareAssetService) downloadAndSaveImage(ctx context.Context, coursewareID string, pageNumber int, imageURL string, prompt string) (string, error) {
	// 下载图片
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("下载图片失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载图片HTTP错误: %d", resp.StatusCode)
	}

	// 检测MIME类型
	contentType := resp.Header.Get("Content-Type")
	ext := ".png"
	if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
		ext = ".jpg"
	} else if strings.Contains(contentType, "webp") {
		ext = ".webp"
	}

	// 生成文件名（用提示词前几个字符作为可读部分）
	nameHint := cwAssetSafeNameRe.ReplaceAllString(prompt, "_")
	// 按rune截断防止切断中文UTF8多字节序列
	nameRunes := []rune(nameHint)
	if len(nameRunes) > 20 {
		nameHint = string(nameRunes[:20])
	}
	for strings.Contains(nameHint, "__") {
		nameHint = strings.ReplaceAll(nameHint, "__", "_")
	}
	nameHint = strings.Trim(nameHint, "_")
	if nameHint == "" {
		nameHint = "ai_gen"
	}
	storedName := fmt.Sprintf("%d_ai_%s%s", time.Now().UnixMilli(), nameHint, ext)

	// 创建目录
	assetDir := filepath.Join(CWAssetUploadDir, coursewareID, fmt.Sprintf("p%d", pageNumber))
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return "", fmt.Errorf("创建图片目录失败: %w", err)
	}

	// 写入文件
	fullPath := filepath.Join(assetDir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, resp.Body); err != nil {
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	// 构建本地URL
	relativePath := filepath.Join(coursewareID, fmt.Sprintf("p%d", pageNumber), storedName)
	localURL := CWAssetURLPrefix + relativePath

	return localURL, nil
}

// ==================== 手动上传图片 ====================

// UploadAssetRequest 上传图片请求参数
type UploadAssetRequest struct {
	CoursewareID  string // 课件ID
	PageNumber    int    // 页码
	PlaceholderID string // 占位符ID（可选）
	UserID        string // 操作者ID
}

// UploadAssetResponse 上传图片响应
type UploadAssetResponse struct {
	AssetID  string `json:"asset_id"`  // 资产记录ID
	URL      string `json:"url"`       // 图片访问URL
	FileName string `json:"file_name"` // 存储文件名
	FileSize int64  `json:"file_size"` // 文件大小
	MimeType string `json:"mime_type"` // MIME类型
}

// UploadAsset 手动上传图片到本地磁盘并记录到数据库
func (s *CoursewareAssetService) UploadAsset(
	ctx context.Context,
	req *UploadAssetRequest,
	file multipart.File,
	header *multipart.FileHeader,
) (*UploadAssetResponse, error) {
	// 1. 校验课件和权限
	cw, err := repository.GetCoursewareByID(ctx, req.CoursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != req.UserID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 2. 校验文件大小
	if header.Size > CWAssetMaxSize {
		return nil, fmt.Errorf("图片文件过大，最大支持5MB（当前%.1fMB）", float64(header.Size)/(1024*1024))
	}

	// 3. 检测MIME类型
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		switch ext {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".webp":
			mimeType = "image/webp"
		case ".gif":
			mimeType = "image/gif"
		case ".svg":
			mimeType = "image/svg+xml"
		default:
			return nil, fmt.Errorf("不支持的图片格式，支持JPG/PNG/WEBP/GIF/SVG")
		}
	}
	if !cwAssetAllowedMimeTypes[mimeType] {
		return nil, fmt.Errorf("不支持的图片格式，支持JPG/PNG/WEBP/GIF/SVG")
	}

	// 4. 校验页面
	page, err := repository.GetCoursewarePageByNumber(ctx, req.CoursewareID, req.PageNumber)
	if err != nil {
		return nil, fmt.Errorf("页面不存在: 课件=%s 页码=%d", req.CoursewareID, req.PageNumber)
	}

	// 5. 生成安全文件名
	ext := cwAssetMimeToExt[mimeType]
	if ext == "" {
		ext = ".png"
	}
	baseName := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
	baseName = cwAssetSafeNameRe.ReplaceAllString(baseName, "_")
	for strings.Contains(baseName, "__") {
		baseName = strings.ReplaceAll(baseName, "__", "_")
	}
	baseName = strings.Trim(baseName, "_")
	// 按rune截断防止切断中文UTF8多字节序列
	baseRunes := []rune(baseName)
	if len(baseRunes) > 40 {
		baseName = string(baseRunes[:40])
	}
	if baseName == "" {
		baseName = "image"
	}
	storedName := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), baseName, ext)

	// 6. 创建目录并保存
	assetDir := filepath.Join(CWAssetUploadDir, req.CoursewareID, fmt.Sprintf("p%d", req.PageNumber))
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return nil, fmt.Errorf("创建图片目录失败: %w", err)
	}

	fullPath := filepath.Join(assetDir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}

	// 7. 构建URL并写入数据库
	relativePath := filepath.Join(req.CoursewareID, fmt.Sprintf("p%d", req.PageNumber), storedName)
	assetURL := CWAssetURLPrefix + relativePath

	asset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PageID:           &page.ID,
		PlaceholderID:    req.PlaceholderID,
		AssetType:        models.CWAssetTypeImage,
		GenerationPrompt: "",
		OssURL:           assetURL,
		FileSize:         written,
		MimeType:         mimeType,
		Status:           models.CWAssetStatusUploaded,
	}
	if err := repository.CreateCWAsset(ctx, asset); err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("记录图片资产失败: %w", err)
	}

	cwAssetLog.Info("课件图片上传成功",
		"courseware_id", req.CoursewareID,
		"page_number", req.PageNumber,
		"asset_id", asset.ID,
		"file", header.Filename,
		"size", written,
		"url", assetURL,
	)

	return &UploadAssetResponse{
		AssetID:  asset.ID,
		URL:      assetURL,
		FileName: storedName,
		FileSize: written,
		MimeType: mimeType,
	}, nil
}

// ==================== 列表查询 ====================

// ListPageAssets 获取指定页面的所有图片/视频资产
func (s *CoursewareAssetService) ListPageAssets(ctx context.Context, coursewareID string, pageNumber int, userID string) ([]*models.CoursewareAsset, error) {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	page, err := repository.GetCoursewarePageByNumber(ctx, coursewareID, pageNumber)
	if err != nil {
		return nil, fmt.Errorf("页面不存在: 课件=%s 页码=%d", coursewareID, pageNumber)
	}

	return repository.ListCWAssetsByPage(ctx, page.ID)
}

// ListCoursewareAssets 获取课件的全部图片/视频资产
func (s *CoursewareAssetService) ListCoursewareAssets(ctx context.Context, coursewareID string, userID string) ([]*models.CoursewareAsset, error) {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	return repository.ListCWAssetsByCourseware(ctx, coursewareID)
}

// ==================== 删除资产 ====================

// DeleteAsset 删除图片/视频资产（磁盘+数据库+连带删OSS+连带清锚点）
//
// 风格锚点轮2（契约②）：style_anchor_asset_id 无外键约束，删图不会报错但会留悬空引用。
// 因此删除前先判断：该资产若是所属课件的当前风格锚点，则先清空课件锚点引用，
// 否则后续生成图片取锚点图时会查不到 asset 而报错（resolveStyleAnchorForGen 已做兜底，但
// 在删除时主动清理更干净，避免无效的锚点状态残留在前端展示）。
func (s *CoursewareAssetService) DeleteAsset(ctx context.Context, assetID string, userID string) error {
	asset, err := repository.GetCWAssetByID(ctx, assetID)
	if err != nil {
		return fmt.Errorf("资产不存在: %w", err)
	}

	cw, err := repository.GetCoursewareByID(ctx, asset.CoursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}

	// 契约②：删的资产若是该课件的风格锚点，先清空锚点引用（避免悬空引用）
	if cw.StyleAnchorAssetID != nil && *cw.StyleAnchorAssetID == assetID {
		if clrErr := repository.ClearCoursewareStyleAnchor(ctx, cw.ID); clrErr != nil {
			// 清锚点失败仅记WARN不阻断删除（resolveStyleAnchorForGen 有兜底，不会因悬空引用报错）
			cwAssetLog.Warn("删除锚点资产时清空课件锚点引用失败(继续删除资产)",
				"asset_id", assetID,
				"courseware_id", cw.ID,
				"error", clrErr,
			)
		} else {
			cwAssetLog.Info("删除锚点资产，已连带清空课件锚点引用",
				"asset_id", assetID,
				"courseware_id", cw.ID,
			)
		}
	}

	// 删除物理文件
	if asset.OssURL != "" && strings.HasPrefix(asset.OssURL, CWAssetURLPrefix) {
		relativePath := asset.OssURL[len(CWAssetURLPrefix):]
		fullPath := filepath.Join(CWAssetUploadDir, relativePath)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			cwAssetLog.Warn("删除物理文件失败",
				"asset_id", assetID,
				"path", fullPath,
				"error", err,
			)
		}
	}

	// v0.42.11: 若该资产已上传到OSS云盘(public_oss_url非空),尽力删除云盘副本
	// 失败仅记WARN不阻断,避免OSS抖动导致本地删除失败;残留孤儿文件可后续清理
	if asset.PublicOSSURL != "" {
		ossSvc := NewOSSService(s.cfg)
		if delErr := ossSvc.DeleteObjectFromOSS(asset.PublicOSSURL); delErr != nil {
			cwAssetLog.Warn("删除OSS云盘副本失败(本地仍照常删除)",
				"asset_id", assetID,
				"public_oss_url", asset.PublicOSSURL,
				"error", delErr,
			)
		} else {
			cwAssetLog.Info("OSS云盘副本已删除", "asset_id", assetID, "public_oss_url", asset.PublicOSSURL)
		}
	}

	if err := repository.DeleteCWAsset(ctx, assetID); err != nil {
		return fmt.Errorf("删除资产记录失败: %w", err)
	}

	cwAssetLog.Info("课件资产删除成功", "asset_id", assetID, "asset_type", asset.AssetType, "courseware_id", asset.CoursewareID)
	return nil
}

// ==================== 风格锚点设/清（轮2新增，一步式同步）====================

// SetStyleAnchorResult 设置锚点的返回结果
type SetStyleAnchorResult struct {
	AssetID   string `json:"asset_id"`   // 锚点资产ID
	AnchorURL string `json:"anchor_url"` // 锚点图公网URL（供前端展示缩略图）
	VAOCI     string `json:"vaoci"`      // 提取出的VAOCI风格索引文本
}

// SetStyleAnchor 设置课件风格锚点（一步式同步）
//
// 流程：校验资产归属 → 取资产公网URL → 多模态读图提取VAOCI → 落库（asset_id + vaoci）。
// 仅图片资产可设为锚点（视频/音频不支持）。
// 提取VAOCI为多模态调用，耗时数秒到十几秒，由前端 loading 兜底。
func (s *CoursewareAssetService) SetStyleAnchor(ctx context.Context, coursewareID string, assetID string, userID string) (*SetStyleAnchorResult, error) {
	// 1. 校验课件归属
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 2. 校验资产存在 + 属于本课件 + 是图片
	asset, err := repository.GetCWAssetByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("资产不存在: %w", err)
	}
	if asset.CoursewareID != coursewareID {
		return nil, fmt.Errorf("资产不属于此课件")
	}
	if asset.AssetType != models.CWAssetTypeImage {
		return nil, fmt.Errorf("仅图片可设为风格锚点")
	}

	// 3. 自动上云（轮3增强）：锚点图若未上传OSS，先传一次，拿到稳定的OSS公网地址。
	//    这样：① 多模态读图/后续图生图用阿里云稳定地址，不依赖本服务器 /uploads/ 可达性；
	//          ② 锚点 asset 的 public_oss_url 被回写，前端任何页都能用它显示锚点缩略图（跨页缓存）。
	//    已上云（public_oss_url 非空）则跳过，幂等不重复传。
	if strings.TrimSpace(asset.PublicOSSURL) == "" && strings.HasPrefix(asset.OssURL, "/uploads/") {
		ossSvc := NewOSSService(s.cfg)
		publicURL, upErr := ossSvc.UploadAssetToOSS(asset.OssURL)
		if upErr != nil {
			return nil, fmt.Errorf("锚点图上传云盘失败（设锚点需稳定公网地址）: %w", upErr)
		}
		// 回写 public_oss_url 持久化（失败仅记WARN，不阻断——本次已拿到URL可用）
		if updErr := repository.UpdateCWAssetPublicURL(ctx, assetID, publicURL); updErr != nil {
			cwAssetLog.Warn("锚点图OSS地址回写失败(不阻断设锚点)", "asset_id", assetID, "error", updErr)
		}
		asset.PublicOSSURL = publicURL
		cwAssetLog.Info("设锚点：锚点图已自动上云", "asset_id", assetID, "public_oss_url", publicURL)
	}

	// 取资产公网URL（契约③：此时 public_oss_url 必有值，优先用它）
	anchorURL := resolveAssetPublicURL(asset)
	if anchorURL == "" {
		return nil, fmt.Errorf("无法解析锚点图的公网URL，请确认图片已正确保存")
	}

	// 4. 多模态读图提取VAOCI风格索引（一步式同步，失败直接报错）
	vaoci, err := s.ExtractVAOCIFromImageURL(ctx, anchorURL, userID)
	if err != nil {
		return nil, fmt.Errorf("提取风格索引失败: %w", err)
	}

	// 5. 落库
	if err := repository.UpdateCoursewareStyleAnchor(ctx, coursewareID, assetID, vaoci); err != nil {
		return nil, fmt.Errorf("保存风格锚点失败: %w", err)
	}

	cwAssetLog.Info("设置课件风格锚点成功",
		"courseware_id", coursewareID,
		"anchor_asset_id", assetID,
		"vaoci_len", len([]rune(vaoci)),
	)

	return &SetStyleAnchorResult{
		AssetID:   assetID,
		AnchorURL: anchorURL,
		VAOCI:     vaoci,
	}, nil
}

// ClearStyleAnchor 清除课件风格锚点（两字段置NULL）
func (s *CoursewareAssetService) ClearStyleAnchor(ctx context.Context, coursewareID string, userID string) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if err := repository.ClearCoursewareStyleAnchor(ctx, coursewareID); err != nil {
		return fmt.Errorf("清除风格锚点失败: %w", err)
	}
	cwAssetLog.Info("清除课件风格锚点成功", "courseware_id", coursewareID)
	return nil
}

// ==================== 插入图片到页面HTML ====================

// InsertImageToPage 将图片插入到页面HTML中
// 两种模式：
//  1. placeholderID非空 → 替换占位符div为<img>标签
//  2. placeholderID为空 → 在内容区末尾追加<img>标签
func (s *CoursewareAssetService) InsertImageToPage(ctx context.Context, coursewareID string, pageNumber int, assetID string, userID string) (string, error) {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return "", fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return "", fmt.Errorf("无权操作此课件")
	}

	asset, err := repository.GetCWAssetByID(ctx, assetID)
	if err != nil {
		return "", fmt.Errorf("资产不存在: %w", err)
	}
	if asset.CoursewareID != coursewareID {
		return "", fmt.Errorf("资产不属于此课件")
	}

	page, err := repository.GetCoursewarePageByNumber(ctx, coursewareID, pageNumber)
	if err != nil {
		return "", fmt.Errorf("页面不存在")
	}
	if page.HTMLContent == "" {
		return "", fmt.Errorf("页面尚未生成HTML，请先生成课件")
	}

	html := page.HTMLContent
	imgTag := fmt.Sprintf(`<img src="%s" alt="课件图片" style="max-width:100%%;height:auto;border-radius:var(--cw-radius,12px);margin:12px 0" />`, asset.OssURL)

	// 模式1: 替换占位符
	if asset.PlaceholderID != "" {
		placeholderPattern := fmt.Sprintf(`<div[^>]*data-placeholder-id="%s"[^>]*>[\s\S]*?</div>`, regexp.QuoteMeta(asset.PlaceholderID))
		re, err := regexp.Compile(placeholderPattern)
		if err == nil && re.MatchString(html) {
			html = re.ReplaceAllString(html, imgTag)
			cwAssetLog.Info("替换占位符为图片",
				"courseware_id", coursewareID,
				"page_number", pageNumber,
				"placeholder_id", asset.PlaceholderID,
			)
		} else {
			cwAssetLog.Warn("未找到占位符，降级为追加模式",
				"placeholder_id", asset.PlaceholderID,
				"page_number", pageNumber,
			)
			html = appendImageToHTML(html, imgTag)
		}
	} else {
		html = appendImageToHTML(html, imgTag)
	}

	// 写回数据库
	if err := repository.UpdateCWPageHTML(ctx, page.ID, html, page.PlaceholderMap, page.MatchedComponentIDs, page.Status); err != nil {
		return "", fmt.Errorf("更新页面HTML失败: %w", err)
	}

	_ = repository.UpdateCWAssetStatus(ctx, assetID, models.CWAssetStatusConfirmed)

	cwAssetLog.Info("图片已插入页面HTML",
		"courseware_id", coursewareID,
		"page_number", pageNumber,
		"asset_id", assetID,
	)

	return html, nil
}

// ==================== HTML操作辅助函数 ====================

// appendImageToHTML 在HTML内容区末尾插入图片
func appendImageToHTML(html string, imgTag string) string {
	wrappedImg := fmt.Sprintf(`<div style="text-align:center;padding:16px 40px">%s</div>`, imgTag)
	lastClose := strings.LastIndex(html, "</div>")
	if lastClose < 0 {
		return html + "\n" + wrappedImg
	}
	return html[:lastClose] + "\n" + wrappedImg + "\n" + html[lastClose:]
}
