package services

// courseware_service.go — 课件工坊核心服务
//
// 课件CRUD + 状态流转 + 索引确认 + 风格保存(Phase 4A结构化) + Logo上传 + 导航栏模板保存(Phase 4C P0-1)
//
// v141 改进：log.Printf → cwServiceLog 结构化日志

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 常量 ====================

const (
	// CWLogoUploadDir Logo文件物理存储根目录
	CWLogoUploadDir = "/www/wwwroot/tedna/uploads/courseware-logos"

	// CWLogoURLPrefix Logo URL前缀（Nginx alias映射）
	CWLogoURLPrefix = "/uploads/courseware-logos/"

	// CWLogoMaxSize 单Logo最大2MB
	CWLogoMaxSize = 2 * 1024 * 1024
)

// 允许的Logo MIME类型
var cwLogoAllowedMimeTypes = map[string]bool{
	"image/jpeg":    true,
	"image/jpg":     true,
	"image/png":     true,
	"image/webp":    true,
	"image/svg+xml": true,
}

// MIME → 扩展名
var cwLogoMimeToExt = map[string]string{
	"image/jpeg":    ".jpg",
	"image/jpg":     ".jpg",
	"image/png":     ".png",
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
}

var cwServiceLog = logger.WithModule("courseware_service")

// CoursewareService 课件工坊服务
type CoursewareService struct{}

// NewCoursewareService 创建课件工坊服务
func NewCoursewareService() *CoursewareService {
	return &CoursewareService{}
}

// ==================== 课件CRUD ====================

// CreateCourseware 创建课件（从教案出发）
// 自动读取教案的标题、学科、年级信息
func (s *CoursewareService) CreateCourseware(ctx context.Context, userID string, req *models.CreateCoursewareRequest) (*models.Courseware, error) {
	if req.LessonPlanID == "" {
		return nil, fmt.Errorf("教案ID不能为空")
	}

	// 查询关联教案获取基本信息
	lp, err := repository.GetLessonPlanByID(ctx, req.LessonPlanID)
	if err != nil {
		return nil, fmt.Errorf("关联教案不存在: %w", err)
	}

	// 标题：优先使用请求中的标题，否则使用教案标题
	title := req.Title
	if title == "" {
		title = lp.Title
	}

	cw := &models.Courseware{
		LessonPlanID: &req.LessonPlanID,
		UserID:       userID,
		Title:        title,
		Subject:      lp.Subject,
		Grade:        lp.Grade,
		Status:       models.CoursewareStatusDraft,
		PageCount:    0,
	}

	if err := repository.CreateCourseware(ctx, cw); err != nil {
		return nil, fmt.Errorf("创建课件失败: %w", err)
	}
	return cw, nil
}

// GetCourseware 获取课件详情（含全部页面）
// Phase 4C: 新增NavTemplateHTML传递
func (s *CoursewareService) GetCourseware(ctx context.Context, id string) (*models.CoursewareDetailResponse, error) {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}

	// 查询全部页面
	pages, err := repository.ListCoursewarePages(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("查询课件页面失败: %w", err)
	}

	// 查询关联教案标题
	lpTitle := ""
	if cw.LessonPlanID != nil && *cw.LessonPlanID != "" {
		lp, lpErr := repository.GetLessonPlanByID(ctx, *cw.LessonPlanID)
		if lpErr == nil {
			lpTitle = lp.Title
		}
	}

	// 风格锚点（轮3）：若已设锚点，查锚点资产拿其公网URL（优先 public_oss_url），
	// 供前端任何页常驻显示锚点缩略图（无需前端跨页另发请求）。查不到不报错，留空即可。
	anchorURL := ""
	if cw.StyleAnchorAssetID != nil && *cw.StyleAnchorAssetID != "" {
		if anchorAsset, aErr := repository.GetCWAssetByID(ctx, *cw.StyleAnchorAssetID); aErr == nil {
			if anchorAsset.PublicOSSURL != "" {
				anchorURL = anchorAsset.PublicOSSURL
			} else if anchorAsset.OssURL != "" {
				anchorURL = anchorAsset.OssURL
			}
		}
	}

	resp := &models.CoursewareDetailResponse{
		ID:              cw.ID,
		LessonPlanID:    cw.LessonPlanID,
		LessonPlanTitle: lpTitle,
		UserID:          cw.UserID,
		Title:           cw.Title,
		Subject:         cw.Subject,
		Grade:           cw.Grade,
		Status:          cw.Status,
		StatusName:      models.CoursewareStatusNameMap[cw.Status],
		StyleConfig:     cw.StyleConfig,
		PageCount:       cw.PageCount,
		IndexOverview:   cw.IndexOverview,
		LogoURL:         cw.LogoURL,
		OrgName:         cw.OrgName,
		NavTemplateHTML: cw.NavTemplateHTML,
		PipelineID:      cw.PipelineID,
		SourceType:      cw.SourceType,
		SourceName:      models.CWSourceNameMap[cw.SourceType],
		// 风格锚点字段（轮1留的契约①：装配时补齐，否则前端查锚点永远是零值）
		StyleAnchorAssetID: cw.StyleAnchorAssetID,
		StyleAnchorVAOCI:   cw.StyleAnchorVAOCI,
		StyleAnchorURL:     anchorURL,
		KPCodes:         cw.KPCodes,
		Pages:           pages,
		CreatedAt:       cw.CreatedAt,
		UpdatedAt:       cw.UpdatedAt,
	}
	return resp, nil
}

// ListCoursewares 查询我的课件列表
func (s *CoursewareService) ListCoursewares(ctx context.Context, userID string, status string, subject string, limit int, offset int) (*models.CoursewareListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	items, total, err := repository.ListCoursewares(ctx, userID, status, subject, limit, offset)
	if err != nil {
		return nil, err
	}
	return &models.CoursewareListResponse{
		Coursewares: items,
		Total:       total,
	}, nil
}

// UpdateCoursewareTitle 更新课件标题
func (s *CoursewareService) UpdateCoursewareTitle(ctx context.Context, id string, userID string, title string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if title == "" {
		return fmt.Errorf("标题不能为空")
	}
	return repository.UpdateCoursewareTitle(ctx, id, title)
}

// DeleteCourseware 删除课件（除已提交审核 in_pipeline 外的任意状态均可删除，删除时由前端二次确认）
func (s *CoursewareService) DeleteCourseware(ctx context.Context, id string, userID string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	// 已提交审核（in_pipeline）的课件关联了审核流程，禁止直接删除；其余状态均允许删除
	if cw.Status == models.CoursewareStatusInPipeline {
		return fmt.Errorf("课件已提交审核，请先撤回审核后再删除")
	}
	return repository.DeleteCourseware(ctx, id)
}

// ==================== 状态流转 ====================

// ConfirmIndex 确认课件索引。
//
// 【状态保持修复（PRD 5.2 共性根因）】原实现无条件把状态设为 styling，假设"确认方案的下一步
// 永远是去选风格"。这对【首次】走流程的新课件（draft/indexing）是对的；但对【已经走到后面、
// 甚至生成过 HTML/媒体】的老课件——老师回 Step1 改了方案（如加一页）再点"确认方案"——这句话
// 会把课件一脚踹回 styling(序号2)，导致"确认导航栏/批量生成/确认提交"等后续步骤全部消失，
// 老师误以为辛苦做的图片视频丢了（实际数据都在，只是被状态机挡在后面步骤之外）。
//
// 修复策略（借用 CoursewareStatusOrder 序号判断，与 RollbackStatus 同一套思路）：
//   - 当前序号 ≤ styling(2)：首次流程，保持原行为 → 设为 styling，引导去选风格；
//   - 当前序号 ≥ generating(3)：老课件回头改方案，已选过风格/生成过页，不回退到 styling，
//     保持在 preview，让老师能直接继续往后走。新增/改动的页本就是 pending 状态，
//     到批量生成步骤会自动补生成，不影响已有内容。
//   这样新课件首次流程完全不变，老课件改方案不再被踢出后续步骤。
func (s *CoursewareService) ConfirmIndex(ctx context.Context, id string, userID string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	// draft或indexing状态都可以确认索引
	if cw.Status == models.CoursewareStatusInPipeline {
		return fmt.Errorf("已提交审核的课件不允许修改")
	}
	// 更新页数
	count, _ := repository.CountCoursewarePages(ctx, id)
	if count == 0 {
		return fmt.Errorf("课件没有任何页面，请先生成索引")
	}
	_ = repository.UpdateCoursewarePageCount(ctx, id, count)

	// 状态保持修复：依据当前状态在状态机中的序号决定确认后落到哪个状态。
	curOrder, ok := models.CoursewareStatusOrder[cw.Status]
	stylingOrder := models.CoursewareStatusOrder[models.CoursewareStatusStyling] // =2
	if ok && curOrder >= models.CoursewareStatusOrder[models.CoursewareStatusGenerating] {
		// 老课件（已走过风格、生成过页）回头改方案：不回退到 styling，保持在 preview，
		// 让老师能直接继续"确认导航栏/批量生成/确认提交"，已有媒体与已生成页不受影响。
		cwServiceLog.Info("确认方案：老课件保持后续可走状态（不回退styling）",
			"courseware_id", id, "from", cw.Status, "to", models.CoursewareStatusPreview)
		return repository.UpdateCoursewareStatus(ctx, id, models.CoursewareStatusPreview)
	}
	_ = stylingOrder // 保留语义可读性（首次流程落 styling，下一行即为该路径）
	// 首次流程（draft/indexing/或本就在styling）：保持原行为，引导去选风格。
	return repository.UpdateCoursewareStatus(ctx, id, models.CoursewareStatusStyling)
}

// SaveStyleFull Phase 4A: 保存完整风格配置（模板+Logo+机构名+自定义色）
// 将结构化数据序列化为JSON存入style_config，同时更新logo_url和org_name
func (s *CoursewareService) SaveStyleFull(ctx context.Context, id string, userID string, req *models.SaveStyleFullRequest) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	// styling状态下保存风格；也允许draft/indexing（用户可能跳步操作）
	if cw.Status == models.CoursewareStatusInPipeline {
		return fmt.Errorf("当前状态不允许保存风格: %s", cw.Status)
	}
	if req.TemplateID == "" {
		return fmt.Errorf("请选择一个风格模板")
	}

	// 序列化style_config为JSON
	styleMap := map[string]string{
		"template_id":          req.TemplateID,
		"logo_url":             req.LogoURL,
		"org_name":             req.OrgName,
		"custom_primary_color": req.CustomPrimaryColor,
	}
	styleJSON, _ := json.Marshal(styleMap)
	if err := repository.UpdateCoursewareStyle(ctx, id, string(styleJSON)); err != nil {
		return err
	}

	// 同步更新logo_url和org_name到课件主表（方便后续直接读取）
	if req.LogoURL != "" {
		_ = repository.UpdateCoursewareLogo(ctx, id, req.LogoURL)
	}
	if req.OrgName != "" {
		_ = repository.UpdateCoursewareOrgName(ctx, id, req.OrgName)
	}

	return nil
}

// SaveStyle 保存风格选择（兼容旧接口，直接存JSON字符串）
func (s *CoursewareService) SaveStyle(ctx context.Context, id string, userID string, styleConfig string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if cw.Status == models.CoursewareStatusInPipeline {
		return fmt.Errorf("当前状态不允许保存风格: %s", cw.Status)
	}
	if styleConfig == "" {
		return fmt.Errorf("风格配置不能为空")
	}
	return repository.UpdateCoursewareStyle(ctx, id, styleConfig)
}

// ConfirmStyle Phase 4C: 确认风格选择，状态从 styling → generating
func (s *CoursewareService) ConfirmStyle(ctx context.Context, id string, userID string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if cw.Status == models.CoursewareStatusInPipeline {
		return fmt.Errorf("已提交审核的课件不允许修改(当前状态:%s)", cw.Status)
	}
	// 检查是否已保存风格配置
	if cw.StyleConfig == "" {
		return fmt.Errorf("请先选择并保存风格配置")
	}
	// 状态推进到generating
	return repository.UpdateCoursewareStatus(ctx, id, models.CoursewareStatusGenerating)
}

// SaveNavTemplate Phase 4C P0-1: 保存用户确认的导航栏HTML模板
// P0-1改造：
//   - 如果前端传入nav_html为空字符串"auto"，则自动从封面页HTML中按标记提取
//   - 提取后自动将硬编码页码替换为 {{PAGE_NUM}} / {{TOTAL_PAGES}} 占位符
func (s *CoursewareService) SaveNavTemplate(ctx context.Context, id string, userID string, navHTML string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	// generating或preview状态下都允许保存导航栏模板
	if cw.Status != models.CoursewareStatusGenerating && cw.Status != models.CoursewareStatusPreview {
		return fmt.Errorf("当前状态不允许保存导航栏模板: %s", cw.Status)
	}

	// P0-1: 如果传入"auto"或空值，自动从封面页提取导航栏
	if navHTML == "" || navHTML == "auto" {
		cwServiceLog.Info("自动从封面页提取导航栏", "courseware_id", id)
		pages, pErr := repository.ListCoursewarePages(ctx, id)
		if pErr != nil || len(pages) == 0 {
			return fmt.Errorf("无法获取封面页用于提取导航栏")
		}
		// 找第1页（封面页）
		var coverPage *models.CoursewarePage
		for _, p := range pages {
			if p.PageNumber == 1 && p.HTMLContent != "" {
				coverPage = p
				break
			}
		}
		if coverPage == nil {
			return fmt.Errorf("封面页尚未生成，无法提取导航栏")
		}
		// 按标记提取导航栏
		extracted := ExtractNavByMarkers(coverPage.HTMLContent)
		if extracted == "" {
			return fmt.Errorf("无法从封面页中提取导航栏（未找到NAV_START/NAV_END标记）")
		}
		navHTML = extracted
	}

	if strings.TrimSpace(navHTML) == "" {
		return fmt.Errorf("导航栏HTML不能为空")
	}

	// P0-1: 自动将硬编码页码替换为占位符
	navHTML = ReplaceNavPageNumbers(navHTML)

	cwServiceLog.Info("保存导航栏模板", "courseware_id", id, "nav_len", len(navHTML))
	return repository.UpdateCoursewareNavTemplate(ctx, id, navHTML)
}

// ConfirmCourseware 确认全部页面，状态 → confirmed
func (s *CoursewareService) ConfirmCourseware(ctx context.Context, id string, userID string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if cw.Status != models.CoursewareStatusPreview {
		return fmt.Errorf("仅预览状态可确认: %s", cw.Status)
	}
	return repository.UpdateCoursewareStatus(ctx, id, models.CoursewareStatusConfirmed)
}

// ==================== Logo上传 ====================

// UploadLogo Phase 4A: 上传课件Logo图片
// 存储路径: /uploads/courseware-logos/{courseware_id}/{timestamp}_{name}
func (s *CoursewareService) UploadLogo(
	ctx context.Context,
	coursewareID string,
	file multipart.File,
	header *multipart.FileHeader,
	callerID string,
) (*models.UploadLogoResponse, error) {
	// 1. 校验课件存在 + 权限
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != callerID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 2. 校验文件大小
	if header.Size > CWLogoMaxSize {
		return nil, fmt.Errorf("Logo文件过大，最大支持2MB")
	}

	// 3. 校验MIME类型
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
		case ".svg":
			mimeType = "image/svg+xml"
		default:
			return nil, fmt.Errorf("不支持的Logo格式，支持JPG/PNG/WEBP/SVG")
		}
	}
	if !cwLogoAllowedMimeTypes[mimeType] {
		return nil, fmt.Errorf("不支持的Logo格式，支持JPG/PNG/WEBP/SVG")
	}

	// 4. 生成安全文件名
	ext := cwLogoMimeToExt[mimeType]
	if ext == "" {
		ext = ".png"
	}
	// 文件名安全化
	baseName := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
	baseName = strings.ReplaceAll(baseName, " ", "_")
	baseName = strings.ReplaceAll(baseName, "..", "_")
	baseName = strings.ReplaceAll(baseName, "/", "_")
	baseName = strings.ReplaceAll(baseName, "\\", "_")
	for _, ch := range []string{"&", "=", ",", "#", "?", "%", "+", "(", ")", "[", "]", "<", ">", "\"", "'", "`", ";", ":"} {
		baseName = strings.ReplaceAll(baseName, ch, "_")
	}
	for strings.Contains(baseName, "__") {
		baseName = strings.ReplaceAll(baseName, "__", "_")
	}
	baseName = strings.Trim(baseName, "_")
	if len(baseName) > 60 {
		baseName = baseName[:60]
	}
	if baseName == "" {
		baseName = "logo"
	}
	storedName := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), baseName, ext)

	// 5. 创建课件专属Logo目录
	logoDir := filepath.Join(CWLogoUploadDir, coursewareID)
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		return nil, fmt.Errorf("创建Logo目录失败: %w", err)
	}

	// 6. 保存物理文件
	fullPath := filepath.Join(logoDir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}

	// 7. 构建URL并更新数据库
	relativePath := filepath.Join(coursewareID, storedName)
	logoURL := CWLogoURLPrefix + relativePath

	if err := repository.UpdateCoursewareLogo(ctx, coursewareID, logoURL); err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("更新Logo URL失败: %w", err)
	}

	cwServiceLog.Info("课件Logo上传成功",
		"courseware_id", coursewareID,
		"file", header.Filename,
		"size", header.Size,
		"url", logoURL,
	)

	return &models.UploadLogoResponse{URL: logoURL}, nil
}

// ==================== 页面操作 ====================

// GetPages 获取课件的所有页面
func (s *CoursewareService) GetPages(ctx context.Context, coursewareID string) ([]*models.CoursewarePage, error) {
	return repository.ListCoursewarePages(ctx, coursewareID)
}

// UpdatePageIndex 更新单页索引说明
func (s *CoursewareService) UpdatePageIndex(ctx context.Context, coursewareID string, pageNumber int, userID string, req *models.UpdateCWPageIndexRequest) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	return repository.UpdateCWPageIndex(ctx, coursewareID, pageNumber, req)
}

// AddPage 手动添加课件页面
func (s *CoursewareService) AddPage(ctx context.Context, coursewareID string, userID string, req *models.AddCWPageRequest) (*models.CoursewarePage, error) {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权操作此课件")
	}
	// 获取当前最大页码
	count, _ := repository.CountCoursewarePages(ctx, coursewareID)
	page := &models.CoursewarePage{
		CoursewareID:        coursewareID,
		PageNumber:          count + 1,
		Title:               req.Title,
		Purpose:             req.Purpose,
		ContentSummary:      req.ContentSummary,
		InteractionType:     req.InteractionType,
		VisualFormat:        req.VisualFormat,
		MediaRequirements:   req.MediaRequirements,
		EstimatedComplexity: req.EstimatedComplexity,
		Status:              models.CWPageStatusPending,
	}
	if page.EstimatedComplexity <= 0 {
		page.EstimatedComplexity = 1
	}
	if err := repository.CreateCoursewarePage(ctx, page); err != nil {
		return nil, fmt.Errorf("添加页面失败: %w", err)
	}
	_ = repository.UpdateCoursewarePageCount(ctx, coursewareID, count+1)
	_ = s.ResyncCWPageNumbers(ctx, coursewareID) // (2) bar denominator: resync after add
	return page, nil
}

// DeletePage 删除课件页面
func (s *CoursewareService) DeletePage(ctx context.Context, coursewareID string, pageNumber int, userID string) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if err := repository.DeleteCoursewarePage(ctx, coursewareID, pageNumber); err != nil {
		return err
	}
	count, _ := repository.CountCoursewarePages(ctx, coursewareID)
	_ = repository.UpdateCoursewarePageCount(ctx, coursewareID, count)
	_ = s.ResyncCWPageNumbers(ctx, coursewareID) // (2) bar denominator: resync after delete
	return nil
}

// ReorderPages 重新排序课件页面
func (s *CoursewareService) ReorderPages(ctx context.Context, coursewareID string, userID string, pageIDs []string) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if err := repository.ReorderCoursewarePages(ctx, coursewareID, pageIDs); err != nil {
return err
}
// ② 导航栏分母专题：重排后刷新各页导航栏分子分母（页号已由 Reorder 落定，失败不阻断）
_ = s.ResyncCWPageNumbers(ctx, coursewareID)
return nil
}

// ==================== v136新增：步骤回退 ====================

// RollbackStatus 回退课件状态到指定目标步骤
// 校验：只能回退到当前步骤之前的状态；in_pipeline不可回退
// 副作用：
//   - 回退到draft时清空所有已生成的HTML和导航栏模板
//   - 回退到indexing/styling时清空导航栏模板
func (s *CoursewareService) RollbackStatus(ctx context.Context, id string, userID string, targetStatus string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}

	// in_pipeline状态不可回退（已提交审核）
	if cw.Status == models.CoursewareStatusInPipeline {
		return fmt.Errorf("已提交审核的课件不可回退")
	}

	// 校验目标状态合法性
	targetOrder, targetOk := models.CoursewareStatusOrder[targetStatus]
	currentOrder, currentOk := models.CoursewareStatusOrder[cw.Status]
	if !targetOk || !currentOk {
		return fmt.Errorf("无效的状态: current=%s target=%s", cw.Status, targetStatus)
	}
	if targetOrder >= currentOrder {
		return fmt.Errorf("只能回退到更早的步骤（当前=%s, 目标=%s）", cw.Status, targetStatus)
	}

	// 副作用处理：回退到draft或indexing时清空导航栏模板
	if targetStatus == models.CoursewareStatusDraft || targetStatus == models.CoursewareStatusIndexing {
		_ = repository.UpdateCoursewareNavTemplate(ctx, id, "")
	}

	// 副作用处理：回退到draft时无需清空HTML——重新生成索引时会自动删旧页面并创建新页面
	// 回退到styling之前的步骤时，清空已生成的风格配置（用户需重新选择）
	if targetStatus == models.CoursewareStatusDraft || targetStatus == models.CoursewareStatusIndexing {
		_ = repository.UpdateCoursewareStyle(ctx, id, "")
	}

	cwServiceLog.Info("课件状态回退",
		"courseware_id", id,
		"from", cw.Status,
		"to", targetStatus,
		"user_id", userID,
	)
	return repository.UpdateCoursewareStatus(ctx, id, targetStatus)
}


// ==================== v0.42新增：从主题直接创建课件 ====================

// CreateCoursewareFromTopic 从主题直接创建课件（不依赖教案）
// 只创建课件记录（source_type=topic_direct），索引由前端触发 GenerateIndex 异步生成
func (s *CoursewareService) CreateCoursewareFromTopic(ctx context.Context, userID string, req *models.CreateCoursewareFromTopicRequest) (*models.Courseware, error) {
	if req.Subject == "" {
		return nil, fmt.Errorf("学科不能为空")
	}
	if req.Grade == "" {
		return nil, fmt.Errorf("年级不能为空")
	}
	if req.Topic == "" {
		return nil, fmt.Errorf("主题名称不能为空")
	}

	// 标题默认使用主题名
	title := req.Topic

	cw := &models.Courseware{
		LessonPlanID: nil, // 无教案关联
		UserID:       userID,
		Title:        title,
		Subject:      req.Subject,
		Grade:        req.Grade,
		Status:       models.CoursewareStatusDraft,
		SourceType:   models.CWSourceTopicDirect,
		PageCount:    0,
	}

	if err := repository.CreateCourseware(ctx, cw); err != nil {
		return nil, fmt.Errorf("创建课件失败: %w", err)
	}

	// 课程知识库轮：若勾选了课标知识点，先校验有效性再序列化存入 kp_codes 列。
	// 第5点修复（迭代回顾）：校验编码真实存在、且属于本课件的学科年级，过滤掉无效/跨学科年级的脏编码，
	//   双重保险（前端切换学科年级已清空，后端再挡一道）。空/失败均不阻断课件创建，仅记录警告。
	if len(req.KPCodes) > 0 {
		// 按编码查课标，比对学科与年级，只留有效项
		validCodes := s.filterValidKPCodes(ctx, req.KPCodes, req.Subject, req.Grade)
		if len(validCodes) < len(req.KPCodes) {
			cwServiceLog.Warn("部分知识点编码无效已过滤",
				"courseware_id", cw.ID,
				"submitted", len(req.KPCodes),
				"valid", len(validCodes),
			)
		}
		if len(validCodes) > 0 {
			if kpJSON, mErr := json.Marshal(validCodes); mErr == nil {
				if uErr := repository.UpdateCoursewareKPCodes(ctx, cw.ID, string(kpJSON)); uErr != nil {
					cwServiceLog.Warn("存储课件知识点编码失败", "error", uErr, "courseware_id", cw.ID)
				} else {
					cw.KPCodes = string(kpJSON)
				}
			}
		}
	}

	cwServiceLog.Info("从主题创建课件",
		"courseware_id", cw.ID,
		"subject", req.Subject,
		"grade", req.Grade,
		"topic", req.Topic,
		"user_id", userID,
	)
	return cw, nil
}

// ==================== v0.42.11新增：创建3D互动单页课件 ====================

// CreateCoursewareFrom3D 创建3D互动单页课件
// 与普通课件不同：source_type='3d_single'，状态直接设为 generating（跳过索引/风格），自动创建1个页面记录
// 前端创建后直接调 generate-3d-page 触发AI生成
func (s *CoursewareService) CreateCoursewareFrom3D(ctx context.Context, userID string, subject string, grade string, topic string, description string) (*models.Courseware, error) {
	if subject == "" {
		return nil, fmt.Errorf("学科不能为空")
	}
	if grade == "" {
		return nil, fmt.Errorf("年级不能为空")
	}
	if topic == "" {
		return nil, fmt.Errorf("主题名称不能为空")
	}
	if len([]rune(description)) < 20 {
		return nil, fmt.Errorf("详细描述至少需要20个字")
	}

	// 创建课件记录：source_type=3d_single，状态直接设为 generating
	cw := &models.Courseware{
		LessonPlanID: nil,
		UserID:       userID,
		Title:        topic,
		Subject:      subject,
		Grade:        grade,
		Status:       models.CoursewareStatusGenerating, // 直接跳到 generating
		SourceType:   models.CWSource3DSingle,
		PageCount:    1,
	}

	if err := repository.CreateCourseware(ctx, cw); err != nil {
		return nil, fmt.Errorf("创建课件失败: %w", err)
	}

	// 自动创建1个页面记录（用于存放3D HTML）
	page := &models.CoursewarePage{
		CoursewareID:        cw.ID,
		PageNumber:          1,
		Title:               topic,
		Purpose:             "3D互动演示：" + topic,
		ContentSummary:      description,
		InteractionType:     "3d",
		VisualFormat:        "fullscreen_media",
		MediaRequirements:   description,
		EstimatedComplexity: 5,
		Status:              models.CWPageStatusPending,
	}
	if err := repository.CreateCoursewarePage(ctx, page); err != nil {
		cwServiceLog.Warn("创建3D页面记录失败", "error", err, "courseware_id", cw.ID)
	}

	cwServiceLog.Info("创建3D互动单页课件",
		"courseware_id", cw.ID,
		"subject", subject,
		"grade", grade,
		"topic", topic,
		"desc_len", len([]rune(description)),
		"user_id", userID,
	)
	return cw, nil
}


// ==================== 课程知识库：编码有效性校验（第5点修复） ====================

// filterValidKPCodes 校验并过滤知识点编码：只保留"真实存在于课标库 且 学科年级与课件匹配"的编码。
// 用途：从主题创建课件存 kp_codes 前调用，挡住前端伪造、跨学科年级残留等脏编码。
// 设计：复用 repository.GetCurriculumKPsByCodes（一次查询），再在内存里比对学科与年级数字；
//   任何查询失败都返回空切片（退回"无约束"，绝不阻断创建）。
//
// 参数：
//   codes        — 前端提交的知识点编码数组
//   subject      — 课件学科（如"数学"）
//   gradeText    — 课件年级文字（如"三年级"），内部解析为数字与 curriculum_standards.grade_num 比对
//
// 返回：有效编码子集（保持原有顺序），全部无效或出错时返回空切片。
func (s *CoursewareService) filterValidKPCodes(ctx context.Context, codes []string, subject string, gradeText string) []string {
	if len(codes) == 0 {
		return nil
	}
	kps, err := repository.GetCurriculumKPsByCodes(ctx, codes)
	if err != nil || len(kps) == 0 {
		cwServiceLog.Warn("校验知识点编码时查询失败或无匹配", "error", err, "subject", subject, "grade", gradeText)
		return nil
	}

	// 解析课件年级文字为数字（与 KnowledgePointSelector.parseGradeNum 同口径：1-12，0=无法识别）
	gradeNum := parseGradeNumForKP(gradeText)

	// 建立"有效编码"集合：库里存在 + 学科匹配 +（年级可识别时）年级匹配
	validSet := make(map[string]bool, len(kps))
	for _, kp := range kps {
		if kp.Subject != subject {
			continue // 跨学科，剔除
		}
		if gradeNum > 0 && kp.GradeNum > 0 && kp.GradeNum != gradeNum {
			continue // 跨年级，剔除（年级无法识别或库中为0即学段级时不卡年级）
		}
		validSet[kp.KPCode] = true
	}

	// 按原始提交顺序保留有效项
	valid := make([]string, 0, len(codes))
	for _, c := range codes {
		if validSet[c] {
			valid = append(valid, c)
		}
	}
	return valid
}

// parseGradeNumForKP 年级文字→数字（后端校验用，与前端 parseGradeNum 同口径，1-12，0=无法识别）。
// 独立实现避免跨层依赖；逻辑保持与前端一致，便于双端校验口径统一。
func parseGradeNumForKP(gradeText string) int {
	t := strings.TrimSpace(gradeText)
	if t == "" {
		return 0
	}
	// 高中
	if strings.Contains(t, "高一") || strings.Contains(t, "高中一") {
		return 10
	}
	if strings.Contains(t, "高二") || strings.Contains(t, "高中二") {
		return 11
	}
	if strings.Contains(t, "高三") || strings.Contains(t, "高中三") {
		return 12
	}
	// 初中
	if strings.Contains(t, "初一") || strings.Contains(t, "初中一") {
		return 7
	}
	if strings.Contains(t, "初二") || strings.Contains(t, "初中二") {
		return 8
	}
	if strings.Contains(t, "初三") || strings.Contains(t, "初中三") {
		return 9
	}
	// 中文数字年级（一~九、十、十一、十二）
	cnMap := map[string]int{
		"一": 1, "二": 2, "三": 3, "四": 4, "五": 5, "六": 6,
		"七": 7, "八": 8, "九": 9, "十": 10, "十一": 11, "十二": 12,
	}
	for cn, n := range cnMap {
		if strings.Contains(t, cn+"年级") {
			return n
		}
	}
	// 阿拉伯数字年级 / 纯数字
	for i := 12; i >= 1; i-- {
		num := fmt.Sprintf("%d", i)
		if strings.Contains(t, num+"年级") || t == num {
			return i
		}
	}
	return 0
}
