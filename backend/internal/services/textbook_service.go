package services

// textbook_service.go — 课本页面图片业务逻辑层
//
// 迭代7新增：课本图片上传+列表+详情+删除+OCR识别+共享

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrTextbookNotFound     = errors.New("课本页面不存在")
	ErrTextbookUnauthorized = errors.New("无权操作此课本页面")
	ErrTextbookFileInvalid  = errors.New("文件格式无效，仅支持JPG/PNG/WEBP图片")
	ErrTextbookFileTooLarge = errors.New("文件过大，最大支持10MB")
)

// ==================== 常量 ====================

const (
	// MaxTextbookFileSize 最大文件大小10MB
	MaxTextbookFileSize = 10 * 1024 * 1024
	// TextbookUploadDir 上传目录
	TextbookUploadDir = "/www/wwwroot/tedna/uploads/textbooks"
)

// 允许的MIME类型
var allowedMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
}

// MIME类型→扩展名映射
var mimeToExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ==================== 服务结构体 ====================

// TextbookService 课本页面服务
type TextbookService struct {
	cfg *config.Config
}

var tbLog = logger.WithModule("textbook")

// NewTextbookService 创建课本服务实例
func NewTextbookService(cfg *config.Config) *TextbookService {
	return &TextbookService{cfg: cfg}
}

// ==================== 上传图片 ====================

// UploadTextbookPage 上传课本页面图片
// 1. 校验文件格式和大小
// 2. 保存到本地目录
// 3. 写入数据库记录
func (s *TextbookService) UploadTextbookPage(ctx context.Context, file multipart.File, header *multipart.FileHeader, req *models.UploadTextbookRequest, callerID string) (*models.TextbookPage, error) {
	// 校验必填字段
	if strings.TrimSpace(req.Subject) == "" {
		return nil, errors.New("学科不能为空")
	}
	if strings.TrimSpace(req.GradeRange) == "" {
		return nil, errors.New("年级不能为空")
	}
	if strings.TrimSpace(req.TextbookName) == "" {
		return nil, errors.New("教材名称不能为空")
	}

	// 校验文件大小
	if header.Size > MaxTextbookFileSize {
		return nil, ErrTextbookFileTooLarge
	}

	// 校验MIME类型
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		// 从文件名推断
		ext := strings.ToLower(filepath.Ext(header.Filename))
		switch ext {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".webp":
			mimeType = "image/webp"
		default:
			return nil, ErrTextbookFileInvalid
		}
	}
	if !allowedMimeTypes[mimeType] {
		return nil, ErrTextbookFileInvalid
	}

	// 生成存储文件名：时间戳_原始文件名（去掉空格）
	ext := mimeToExt[mimeType]
	if ext == "" {
		ext = ".jpg"
	}
	safeOrigName := strings.ReplaceAll(header.Filename, " ", "_")
	storedName := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), strings.TrimSuffix(safeOrigName, filepath.Ext(safeOrigName)), ext)

	// 按学科+年级创建子目录
	subDir := fmt.Sprintf("%s/%s", strings.ReplaceAll(req.Subject, "/", "_"), strings.ReplaceAll(req.GradeRange, "/", "_"))
	fullDir := filepath.Join(TextbookUploadDir, subDir)
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return nil, fmt.Errorf("创建上传目录失败: %w", err)
	}

	// 保存文件
	fullPath := filepath.Join(fullDir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(fullPath) // 清理失败文件
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}

	// 设置默认scope
	scope := req.Scope
	if scope == "" {
		scope = models.TextbookScopePersonal
	}
	var scopeRefID *string
	if req.ScopeRefID != "" {
		scopeRefID = &req.ScopeRefID
	}

	// 存储路径使用相对路径（subDir/storedName）
	relativePath := filepath.Join(subDir, storedName)

	// 写入数据库
	page := &models.TextbookPage{
		Subject:      req.Subject,
		GradeRange:   req.GradeRange,
		TextbookName: req.TextbookName,
		Chapter:      req.Chapter,
		PageNumber:   req.PageNumber,
		FileName:     header.Filename,
		FilePath:     relativePath,
		FileSize:     written,
		MimeType:     mimeType,
		Description:  req.Description,
		Tags:         "[]",
		Scope:        scope,
		ScopeRefID:   scopeRefID,
		UploadedBy:   callerID,
	}

	if err := repository.CreateTextbookPage(ctx, page); err != nil {
		os.Remove(fullPath) // 数据库写入失败时清理文件
		return nil, err
	}

	tbLog.Info("课本图片上传成功",
		"id", page.ID,
		"textbook", req.TextbookName,
		"file", header.Filename,
		"size", written,
		"uploader", callerID,
	)
	return page, nil
}

// ==================== 查询 ====================

// GetTextbookPage 获取课本页面详情
func (s *TextbookService) GetTextbookPage(ctx context.Context, id string) (*models.TextbookDetailResponse, error) {
	page, err := repository.GetTextbookPageByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrTextbookNotFound) {
			return nil, ErrTextbookNotFound
		}
		return nil, err
	}

	// 查询上传者名称
	uploaderName := ""
	if user, err := repository.FindUserByID(ctx, page.UploadedBy); err == nil {
		uploaderName = user.DisplayName
	}

	return &models.TextbookDetailResponse{
		TextbookPage: *page,
		UploaderName: uploaderName,
		ImageURL:     "/uploads/textbooks/" + page.FilePath,
		HasOCR:       page.OCRText != "",
	}, nil
}

// ListTextbookPages 查询课本页面列表
func (s *TextbookService) ListTextbookPages(ctx context.Context, callerID string, subject string, gradeRange string, textbookName string, scope string, limit int, offset int) (*models.TextbookListResponse, error) {
	items, total, err := repository.ListTextbookPages(ctx, callerID, subject, gradeRange, textbookName, scope, limit, offset)
	if err != nil {
		return nil, err
	}
	return &models.TextbookListResponse{Pages: items, Total: total}, nil
}

// ==================== 更新 ====================

// UpdateTextbookPage 更新课本页面元数据（需验证所有权）
func (s *TextbookService) UpdateTextbookPage(ctx context.Context, id string, req *models.UpdateTextbookRequest, callerID string) error {
	page, err := repository.GetTextbookPageByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrTextbookNotFound) {
			return ErrTextbookNotFound
		}
		return err
	}
	if page.UploadedBy != callerID {
		return ErrTextbookUnauthorized
	}
	return repository.UpdateTextbookPage(ctx, id, req)
}

// ==================== 删除 ====================

// DeleteTextbookPage 删除课本页面（软删除，需验证所有权）
func (s *TextbookService) DeleteTextbookPage(ctx context.Context, id string, callerID string) error {
	page, err := repository.GetTextbookPageByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrTextbookNotFound) {
			return ErrTextbookNotFound
		}
		return err
	}
	if page.UploadedBy != callerID {
		return ErrTextbookUnauthorized
	}
	return repository.DeleteTextbookPage(ctx, id)
}

// ==================== OCR识别（调用AI Vision）====================

// RecognizeTextbookPage 调用AI识别课本图片中的文字内容
// 读取图片→base64编码→发送给AI Vision→回填OCR结果
func (s *TextbookService) RecognizeTextbookPage(ctx context.Context, id string, callerID string) (string, error) {
	page, err := repository.GetTextbookPageByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrTextbookNotFound) {
			return "", ErrTextbookNotFound
		}
		return "", err
	}

	// 读取图片文件
	fullPath := filepath.Join(TextbookUploadDir, page.FilePath)
	imageData, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("读取图片文件失败: %w", err)
	}

	// base64编码
	b64 := base64.StdEncoding.EncodeToString(imageData)
	mediaType := page.MimeType
	if mediaType == "" {
		mediaType = "image/jpeg"
	}
	dataURI := fmt.Sprintf("data:%s;base64,%s", mediaType, b64)

	// 获取AI配置（使用scanner场景，Haiku速度快成本低适合OCR）
	cfg, err := ai.GetEffectiveConfig(s.cfg.AESKey, "scanner", s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel)
	if err != nil {
		return "", fmt.Errorf("获取AI配置失败: %w", err)
	}

	// 构造多模态消息（OpenAI兼容格式）
	result, err := ai.CallAIMultimodal(cfg,
		"你是课本文字识别专家。请仔细识别图片中的所有文字内容，包括标题、正文、注释、图表中的文字、公式等。按原文排版顺序输出，保持段落结构。如果有表格，用Markdown表格格式输出。如果有公式，用LaTeX格式输出。",
		"请识别这张课本图片中的所有文字内容：",
		dataURI,
	)
	if err != nil {
		return "", fmt.Errorf("AI识别失败: %w", err)
	}

	// 回填OCR结果到数据库
	if err := repository.UpdateTextbookOCR(ctx, id, result.Content, result.ModelUsed); err != nil {
		tbLog.Warn("OCR结果回填失败", "id", id, "error", err)
	}

	tbLog.Info("课本OCR识别完成", "id", id, "model", result.ModelUsed, "text_len", len(result.Content))
	return result.Content, nil
}

// ==================== 迭代7B新增：构建课本上下文（供备课对话注入）====================

// BuildTextbookContext 从课本图片ID列表构建AI上下文文本
// 有OCR缓存的直接用文字，没有的标记"未识别"
// 返回格式化的课本内容文本，可直接拼入系统提示词
func (s *TextbookService) BuildTextbookContext(ctx context.Context, pageIDs []string) string {
	if len(pageIDs) == 0 {
		return ""
	}

	pages, err := repository.GetTextbookPagesByIDs(ctx, pageIDs)
	if err != nil || len(pages) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n== 课本原文参考 ==\n")
	sb.WriteString("以下是老师上传的课本真实内容，请严格参考课本原文进行教学设计：\n\n")

	for i, page := range pages {
		sb.WriteString(fmt.Sprintf("--- 课本第%d页（%s · %s）---\n", i+1, page.TextbookName, page.Chapter))
		if page.OCRText != "" {
			sb.WriteString(page.OCRText)
			sb.WriteString("\n")
		} else {
			sb.WriteString("[此页图片尚未识别文字，请提醒老师先进行AI识别]\n")
		}
		sb.WriteString("\n")

		// 递增使用计数（异步）
		go func(pid string) {
			_ = repository.IncrementTextbookUsage(context.Background(), pid)
		}(page.ID)
	}

	return sb.String()
}
