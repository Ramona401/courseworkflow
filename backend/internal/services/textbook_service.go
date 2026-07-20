package services

// textbook_service.go — 课本页面图片业务逻辑层
//
// 迭代7新增：课本图片上传+列表+详情+删除+OCR识别+共享
//
// v231新增：教材照片归档维度扩展
//   - UploadTextbookPage 落库时带 Semester(学期) + Unit(单元) 两字段
//   - ListTextbookPages 签名新增 semester/unit 两个筛选参数并透传给 repository
//
// 上下文15新增：K12课本模块统一教育域闸门
//   - 每次请求实时读取用户角色和确定性教育域，不信任JWT历史教育域或前端参数
//   - K12正常查询与操作
//   - vocational/adult/mixed/common/空值/非法值列表返回成功空数组
//   - 非K12详情与写操作明确返回403
//   - 教育域解析的数据库错误继续上抛为5xx，不能伪装为空列表
//   - OCR为同步执行，无独立任务或Worker，因此本上下文不新增数据库字段

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
	ErrTextbookK12Only      = errors.New("当前教育域不支持课本能力")
)

// ==================== 常量 ====================

const (
	// MaxTextbookFileSize 最大文件大小10MB
	MaxTextbookFileSize = 10 * 1024 * 1024
	// TextbookUploadDir 上传目录
	TextbookUploadDir = "/www/wwwroot/tedna/private/textbooks"
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

// textbookActorEducationContext 是课本模块运行时使用的最小教育域上下文。
//
// DomainConflict单独保留，防止“解析结果碰巧为k12但实际同时属于多个具体教育域”被误放行。
type textbookActorEducationContext struct {
	Domain         string
	DomainConflict bool
}

var tbLog = logger.WithModule("textbook")

// NewTextbookService 创建课本服务实例
func NewTextbookService(cfg *config.Config) *TextbookService {
	return &TextbookService{cfg: cfg}
}

// ==================== K12教育域统一解析 ====================

// resolveTextbookActorEducationContext 实时解析当前用户的确定性教育域。
//
// 解析顺序：
//  1. 从users表实时读取角色，避免JWT中的历史角色造成授权漂移；
//  2. 使用统一ResolveUserEducationContext解析任命、校籍和教研组关系；
//  3. 原样保留DomainConflict，由调用方fail-closed处理；
//  4. 不使用NormalizeEducationDomain，避免空值或非法值静默回退K12。
func resolveTextbookActorEducationContext(ctx context.Context, callerID string) (*textbookActorEducationContext, error) {
	if strings.TrimSpace(callerID) == "" {
		return nil, ErrTextbookUnauthorized
	}

	user, err := repository.FindUserByID(ctx, callerID)
	if err != nil {
		return nil, fmt.Errorf("读取课本操作者实时角色失败: %w", err)
	}

	educationContext, err := repository.ResolveUserEducationContext(ctx, callerID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("解析课本操作者教育域失败: %w", err)
	}
	if educationContext == nil {
		return nil, errors.New("解析课本操作者教育域失败: 返回空上下文")
	}

	return &textbookActorEducationContext{
		Domain:         strings.ToLower(strings.TrimSpace(educationContext.EducationDomain)),
		DomainConflict: educationContext.DomainConflict,
	}, nil
}

// textbookActorCanUseK12Module 判断当前实时教育域是否可以进入K12课本模块。
func textbookActorCanUseK12Module(actorContext *textbookActorEducationContext) bool {
	return actorContext != nil &&
		!actorContext.DomainConflict &&
		actorContext.Domain == models.EducationDomainK12
}

// AuthorizeK12TextbookWrite 为Handler在解析multipart大文件前提供前置权限检查。
//
// Service内的正式写方法仍会再次执行同一检查，前置检查只用于尽早拒绝无权请求，
// 不能替代业务层最终授权。
func (s *TextbookService) AuthorizeK12TextbookWrite(ctx context.Context, callerID string) error {
	actorContext, err := resolveTextbookActorEducationContext(ctx, callerID)
	if err != nil {
		return err
	}
	if !textbookActorCanUseK12Module(actorContext) {
		return ErrTextbookK12Only
	}
	return nil
}

// resolveK12TextbookWriteDomain 返回经过严格验证的K12教育域，供Repository显式域参数使用。
func resolveK12TextbookWriteDomain(ctx context.Context, callerID string) (string, error) {
	actorContext, err := resolveTextbookActorEducationContext(ctx, callerID)
	if err != nil {
		return "", err
	}
	if !textbookActorCanUseK12Module(actorContext) {
		return "", ErrTextbookK12Only
	}
	return models.EducationDomainK12, nil
}

// mapTextbookRepositoryError 把Repository教育域防线错误映射为Service统一错误。
func mapTextbookRepositoryError(err error) error {
	if errors.Is(err, repository.ErrTextbookEducationDomainUnsupported) {
		return ErrTextbookK12Only
	}
	if errors.Is(err, repository.ErrTextbookNotFound) {
		return ErrTextbookNotFound
	}
	return err
}

// ==================== 上传图片 ====================

// UploadTextbookPage 上传课本页面图片
//  1. 严格校验调用方必须为K12
//  2. 校验文件格式和大小
//  3. 保存到本地目录
//  4. 写入数据库记录
func (s *TextbookService) UploadTextbookPage(ctx context.Context, file multipart.File, header *multipart.FileHeader, req *models.UploadTextbookRequest, callerID string) (*models.TextbookPage, error) {
	educationDomain, err := resolveK12TextbookWriteDomain(ctx, callerID)
	if err != nil {
		return nil, err
	}

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
		_ = os.Remove(fullPath) // 清理失败文件
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

	// 写入数据库（v231：带上 Semester 学期 + Unit 单元 两个归档字段）
	page := &models.TextbookPage{
		Subject:      req.Subject,
		GradeRange:   req.GradeRange,
		Semester:     strings.TrimSpace(req.Semester),
		Unit:         strings.TrimSpace(req.Unit),
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

	if err := repository.CreateTextbookPageForEducationDomain(ctx, page, educationDomain); err != nil {
		_ = os.Remove(fullPath) // 数据库写入失败时清理文件
		return nil, mapTextbookRepositoryError(err)
	}

	tbLog.Info("课本图片上传成功",
		"id", page.ID,
		"textbook", req.TextbookName,
		"semester", req.Semester,
		"unit", req.Unit,
		"file", header.Filename,
		"size", written,
		"uploader", callerID,
		"education_domain", educationDomain,
	)
	return page, nil
}

// ==================== 查询 ====================

// GetTextbookPage 获取课本页面详情。
//
// 直接ID访问同样执行K12教育域校验，非K12不能通过猜测ID读取课本详情。
func (s *TextbookService) GetTextbookPage(ctx context.Context, id string, callerID string) (*models.TextbookDetailResponse, error) {
	educationDomain, err := resolveK12TextbookWriteDomain(ctx, callerID)
	if err != nil {
		return nil, err
	}

	page, err := repository.GetTextbookPageByIDForEducationDomain(ctx, id, educationDomain)
	if err != nil {
		return nil, mapTextbookRepositoryError(err)
	}

	// 查询上传者名称
	uploaderName := ""
	if user, err := repository.FindUserByID(ctx, page.UploadedBy); err == nil {
		uploaderName = user.DisplayName
	}

	return &models.TextbookDetailResponse{
		TextbookPage: *page,
		UploaderName: uploaderName,
		ImageURL:     "/api/v1/lesson-plans/textbooks/" + page.ID + "/image",
		HasOCR:       page.OCRText != "",
	}, nil
}

// OpenTextbookImage 为课本图片鉴权端点打开正式图片文件。
//
// 安全规则：
//   - 实时用户教育域必须仍然是确定性K12；
//   - 课本记录必须存在并且为active；
//   - 只接受数据库保存的相对路径；
//   - 清理路径后必须仍位于TextbookUploadDir私有目录内；
//   - 解析符号链接后的真实路径也必须位于私有目录内；
//   - 只返回普通文件，不返回目录或其它特殊文件。
//
// 返回的文件由Handler负责关闭。
func (s *TextbookService) OpenTextbookImage(
	ctx context.Context,
	id string,
	callerID string,
) (*os.File, os.FileInfo, string, error) {
	educationDomain, err :=
		resolveK12TextbookWriteDomain(
			ctx,
			callerID,
		)
	if err != nil {
		return nil, nil, "", err
	}

	page, err :=
		repository.GetTextbookPageByIDForEducationDomain(
			ctx,
			id,
			educationDomain,
		)
	if err != nil {
		return nil, nil, "",
			mapTextbookRepositoryError(err)
	}

	relativePath := filepath.Clean(
		strings.TrimSpace(page.FilePath),
	)

	separator := string(os.PathSeparator)

	if relativePath == "" ||
		relativePath == "." ||
		relativePath == ".." ||
		filepath.IsAbs(relativePath) ||
		strings.HasPrefix(
			relativePath,
			".."+separator,
		) {
		return nil, nil, "",
			ErrTextbookNotFound
	}

	basePath, err :=
		filepath.Abs(TextbookUploadDir)
	if err != nil {
		return nil, nil, "",
			fmt.Errorf(
				"解析课本私有目录失败: %w",
				err,
			)
	}

	fullPath, err :=
		filepath.Abs(
			filepath.Join(
				basePath,
				relativePath,
			),
		)
	if err != nil {
		return nil, nil, "",
			fmt.Errorf(
				"解析课本图片路径失败: %w",
				err,
			)
	}

	if fullPath == basePath ||
		!strings.HasPrefix(
			fullPath,
			basePath+separator,
		) {
		return nil, nil, "",
			ErrTextbookNotFound
	}

	// 再解析真实符号链接路径，防止目录内恶意链接指向私有目录外。
	resolvedBase, err :=
		filepath.EvalSymlinks(basePath)
	if err != nil {
		return nil, nil, "",
			fmt.Errorf(
				"解析课本私有目录真实路径失败: %w",
				err,
			)
	}

	resolvedPath, err :=
		filepath.EvalSymlinks(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, "",
				ErrTextbookNotFound
		}

		return nil, nil, "",
			fmt.Errorf(
				"解析课本图片真实路径失败: %w",
				err,
			)
	}

	if resolvedPath == resolvedBase ||
		!strings.HasPrefix(
			resolvedPath,
			resolvedBase+separator,
		) {
		return nil, nil, "",
			ErrTextbookNotFound
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, "",
				ErrTextbookNotFound
		}

		return nil, nil, "",
			fmt.Errorf(
				"打开课本图片失败: %w",
				err,
			)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()

		return nil, nil, "",
			fmt.Errorf(
				"读取课本图片信息失败: %w",
				err,
			)
	}

	if !fileInfo.Mode().IsRegular() {
		_ = file.Close()

		return nil, nil, "",
			ErrTextbookNotFound
	}

	mimeType := strings.ToLower(
		strings.TrimSpace(page.MimeType),
	)

	if !allowedMimeTypes[mimeType] {
		_ = file.Close()

		return nil, nil, "",
			ErrTextbookFileInvalid
	}

	return file,
		fileInfo,
		mimeType,
		nil
}

// ListTextbookPages 查询课本页面列表
// v231：新增 semester(学期) + unit(单元) 两个筛选参数，透传给 repository
//
// 非K12、mixed、common、空值、非法域或教育域冲突返回成功空数组；
// 只有数据库解析错误或K12真实查询错误才返回error。
func (s *TextbookService) ListTextbookPages(ctx context.Context, callerID string, subject string, gradeRange string, semester string, unit string, textbookName string, scope string, limit int, offset int) (*models.TextbookListResponse, error) {
	actorContext, err := resolveTextbookActorEducationContext(ctx, callerID)
	if err != nil {
		return nil, err
	}
	if !textbookActorCanUseK12Module(actorContext) {
		return &models.TextbookListResponse{
			Pages: []*models.TextbookListItem{},
			Total: 0,
		}, nil
	}

	items, total, err := repository.ListTextbookPagesForEducationDomain(
		ctx,
		callerID,
		actorContext.Domain,
		subject,
		gradeRange,
		semester,
		unit,
		textbookName,
		scope,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	// 上下文15：列表图片URL统一改为鉴权端点。
	//
	// 前端不能再依赖公开/uploads目录读取原文件；
	// 实际图片内容必须通过携带登录凭证的API Blob请求获得。
	for _, item := range items {
		if item == nil {
			continue
		}

		item.ImageURL =
			"/api/v1/lesson-plans/textbooks/" +
				item.ID +
				"/image"
	}

	return &models.TextbookListResponse{Pages: items, Total: total}, nil
}

// ==================== 更新 ====================

// UpdateTextbookPage 更新课本页面元数据（需验证K12教育域与所有权）
func (s *TextbookService) UpdateTextbookPage(ctx context.Context, id string, req *models.UpdateTextbookRequest, callerID string) error {
	educationDomain, err := resolveK12TextbookWriteDomain(ctx, callerID)
	if err != nil {
		return err
	}

	page, err := repository.GetTextbookPageByIDForEducationDomain(ctx, id, educationDomain)
	if err != nil {
		return mapTextbookRepositoryError(err)
	}
	if page.UploadedBy != callerID {
		return ErrTextbookUnauthorized
	}

	if err := repository.UpdateTextbookPageForEducationDomain(ctx, id, req, educationDomain); err != nil {
		return mapTextbookRepositoryError(err)
	}
	return nil
}

// ==================== 删除 ====================

// DeleteTextbookPage 删除课本页面（软删除，需验证K12教育域与所有权）
func (s *TextbookService) DeleteTextbookPage(ctx context.Context, id string, callerID string) error {
	educationDomain, err := resolveK12TextbookWriteDomain(ctx, callerID)
	if err != nil {
		return err
	}

	page, err := repository.GetTextbookPageByIDForEducationDomain(ctx, id, educationDomain)
	if err != nil {
		return mapTextbookRepositoryError(err)
	}
	if page.UploadedBy != callerID {
		return ErrTextbookUnauthorized
	}

	if err := repository.DeleteTextbookPageForEducationDomain(ctx, id, educationDomain); err != nil {
		return mapTextbookRepositoryError(err)
	}
	return nil
}

// ==================== OCR识别（调用AI Vision）====================

// RecognizeTextbookPage 调用AI识别课本图片中的文字内容
// 读取图片→base64编码→发送给AI Vision→回填OCR结果
//
// v200修复(P1-17)：识别结果用于注入备课上下文，原system prompt要求"公式用LaTeX输出"
// 导致教材里的分数(如½)被识别成 \frac{1}{2} 这类代码，前端renderMarkdown不解析LaTeX，
// 老师看到的就是"分数显示为代码"。本次改写OCR system prompt：
//   - 删除"公式用LaTeX输出"要求
//   - 分数一律用 a/b 纯文本(如 1/2)、带分数用 "1又3/4"，上下标用自然语言或纯文本
//   - 明令禁止输出 LaTeX、\frac、$...$、HTML标签、代码块
//
// 表格仍用Markdown表格(前端renderMarkdown支持/将支持，不冲突)。
func (s *TextbookService) RecognizeTextbookPage(ctx context.Context, id string, callerID string) (string, error) {
	educationDomain, err := resolveK12TextbookWriteDomain(ctx, callerID)
	if err != nil {
		return "", err
	}

	page, err := repository.GetTextbookPageByIDForEducationDomain(ctx, id, educationDomain)
	if err != nil {
		return "", mapTextbookRepositoryError(err)
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
	// v198：补操作者(callerID) UserID + 所属学校ID，供模型境内/境外分流判定（原 traceCtx=nil，无分流无埋点，一并补齐）
	tbUID := callerID
	tbSchoolID, _ := repository.GetSchoolIDByUserID(ctx, callerID)
	tbTraceCtx := &ai.TraceContext{SceneCode: "scanner", UserID: &tbUID, SchoolID: schoolIDPtr(tbSchoolID)}

	// v200(P1-17)：OCR system prompt — 严禁LaTeX/代码，分数走纯文本，避免注入备课上下文后显示为代码
	ocrSystemPrompt := "你是课本文字识别专家。请仔细识别图片中的所有文字内容，包括标题、正文、注释、图表中的文字等，按原文排版顺序输出，保持段落结构。\n" +
		"重要的输出格式约束（必须严格遵守）：\n" +
		"1. 分数一律用纯文本斜杠形式表达，例如二分之一写成 1/2，四分之三写成 3/4，带分数（如一又三分之四）写成 1又3/4。\n" +
		"2. 上标、下标、指数等用自然语言或纯文本表达，例如 x的平方 可写成 x^2 或“x的平方”，化学式中的下标如水写成 H2O，不要使用任何特殊排版语法。\n" +
		"3. 严禁输出 LaTeX 代码（如 \\frac、\\times、$...$、\\(...\\) 等）、HTML 标签、Markdown 代码块（```）。所有数学与科学符号都必须以中小学生能直接读懂的普通文本或常见符号（+ - × ÷ = / 等）呈现。\n" +
		"4. 如果有表格，可以用 Markdown 表格格式输出。\n" +
		"目标：识别结果将直接作为课本原文参考供老师备课使用，必须是干净、可读、不含任何代码或排版标记的纯文本。"

	result, err := ai.CallAIMultimodal(cfg,
		ocrSystemPrompt,
		"请识别这张课本图片中的所有文字内容：",
		dataURI,
		tbTraceCtx,
	)
	if err != nil {
		return "", fmt.Errorf("AI识别失败: %w", err)
	}

	// OCR执行完成后再次解析当前用户教育域。
	//
	// AI调用可能持续数秒，期间用户的组织任命、角色或教育域可能发生变化；
	// 因此不能继续沿用执行前取得的educationDomain。
	// 只有执行结束时仍然是确定性K12用户，才允许回填识别结果。
	executionDomain, err :=
		resolveK12TextbookWriteDomain(
			ctx,
			callerID,
		)
	if err != nil {
		return "", err
	}

	// 执行时重新读取课本正式记录。
	//
	// Repository的K12专用详情读取只允许active记录；
	// 页面若在AI执行期间被归档、删除或替换文件，识别结果不得写入。
	freshPage, err :=
		repository.GetTextbookPageByIDForEducationDomain(
			ctx,
			id,
			executionDomain,
		)
	if err != nil {
		return "",
			mapTextbookRepositoryError(err)
	}

	if freshPage == nil ||
		strings.TrimSpace(freshPage.Status) != "active" ||
		strings.TrimSpace(freshPage.FilePath) == "" ||
		filepath.Clean(freshPage.FilePath) !=
			filepath.Clean(page.FilePath) {
		return "", ErrTextbookNotFound
	}

	// OCR为同步链路：
	// HTTP请求成功返回前必须确认数据库回填成功。
	//
	// 数据库错误不能只记录Warn后继续返回识别成功，
	// 否则前端会显示“识别完成”，实际下一轮AI却读取不到OCR原文。
	if err :=
		repository.UpdateTextbookOCRForEducationDomain(
			ctx,
			id,
			result.Content,
			result.ModelUsed,
			executionDomain,
		); err != nil {
		return "", fmt.Errorf(
			"OCR结果回填失败: %w",
			mapTextbookRepositoryError(err),
		)
	}

	// 后续完成日志使用执行时重新确认的正式教育域。
	educationDomain = executionDomain

	tbLog.Info(
		"课本OCR识别完成",
		"id", id,
		"model", result.ModelUsed,
		"text_len", len(result.Content),
		"education_domain", educationDomain,
	)
	return result.Content, nil
}

// ==================== 迭代7B新增：构建课本上下文（供备课对话注入）====================

// BuildTextbookContext 从课本图片ID列表构建AI上下文文本
// 有OCR缓存的直接用文字，没有的标记"未识别"
// 返回格式化的课本内容文本，可直接拼入系统提示词
//
// 注意：本函数的运行时教案教育域复核将在上下文15后端第二批接入，
// 当前第一批先完成HTTP入口与挂载写入硬闸，避免一次修改超过5个文件。
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
