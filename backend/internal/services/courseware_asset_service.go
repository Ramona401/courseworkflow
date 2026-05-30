package services

// courseware_asset_service.go — 课件多媒体资产服务
//
// v0.42 多媒体：AI图片生成 + 手动上传 + 插入HTML + 列表 + 删除
// v0.42.1 新增：AI视频生成（异步提交+状态查询）
//
// 功能：
//   - GenerateImage: 调用豆包API生成图片，下载保存到本地
//   - GenerateVideo: 调用豆包API提交视频生成任务（异步，返回task_id）
//   - QueryVideoStatus: 查询视频生成任务状态，成功时下载保存到本地
//   - UploadAsset: 手动上传图片到本地磁盘
//   - InsertImageToPage: 将图片插入到页面HTML中（替换占位符或追加）
//   - ListPageAssets / ListCoursewareAssets: 查询图片/视频资产
//   - DeleteAsset: 删除图片/视频资产（磁盘+数据库）
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

	// 4. 调用豆包API生成图片
	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_image_gen",
		UserID:    &req.UserID,
	}
	// 确定图片尺寸：用户指定 > 默认1920x1920
	imageSize := req.Size
	if imageSize == "" {
		imageSize = "1920x1920"
	}
	// 构建参考图完整URL（本地路径转公网URL供豆包API下载）
	refURL := ""
	if req.RefImageURL != "" {
		if strings.HasPrefix(req.RefImageURL, "/uploads/") {
			refURL = "https://workflow.pkuailab.com" + req.RefImageURL
		} else {
			refURL = req.RefImageURL
		}
	}
	result, err := ai.GenerateImage(ctx, imgCfg, req.Prompt, imageSize, 1, refURL, traceCtx)
	if err != nil {
		return nil, fmt.Errorf("图片生成失败: %w", err)
	}
	if len(result.URLs) == 0 {
		return nil, fmt.Errorf("图片生成未返回有效URL")
	}

	// 5. 下载第一张图片到本地存储
	imageURL := result.URLs[0]
	localURL, err := s.downloadAndSaveImage(ctx, req.CoursewareID, req.PageNumber, imageURL, req.Prompt)
	if err != nil {
		return nil, fmt.Errorf("下载生成图片失败: %w", err)
	}

	// 6. 写入数据库
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

// DeleteAsset 删除图片/视频资产（磁盘+数据库）
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

	if err := repository.DeleteCWAsset(ctx, assetID); err != nil {
		return fmt.Errorf("删除资产记录失败: %w", err)
	}

	cwAssetLog.Info("课件资产删除成功", "asset_id", assetID, "asset_type", asset.AssetType, "courseware_id", asset.CoursewareID)
	return nil
}

// ==================== 插入图片到页面HTML ====================

// InsertImageToPage 将图片插入到页面HTML中
// 两种模式：
//   1. placeholderID非空 → 替换占位符div为<img>标签
//   2. placeholderID为空 → 在内容区末尾追加<img>标签
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
