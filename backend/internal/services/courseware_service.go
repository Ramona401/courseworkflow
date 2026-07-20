package services

// courseware_service.go — 课件工坊核心服务
//
// 课件CRUD、状态流转、风格、Logo与导航栏模板保存。
//
// 页面操作、步骤回退、主题创建、3D创建与课程知识库编码校验
// 已拆分到courseware_service_curriculum.go，避免单文件超过600行。

import (
	"context"
	"encoding/json"
	"errors"
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

// 从教案创建课件的稳定错误，供Handler准确映射HTTP状态码。
var (
	ErrCoursewareLessonPlanRequired = errors.New(
		"教案ID不能为空",
	)
	ErrCoursewareLessonPlanNotFound = errors.New(
		"关联教案不存在",
	)
)

// CoursewareService 课件工坊服务
type CoursewareService struct{}

// NewCoursewareService 创建课件工坊服务
func NewCoursewareService() *CoursewareService {
	return &CoursewareService{}
}

// ==================== 课件CRUD ====================

// CreateCourseware 创建课件（从教案出发）
// 自动读取教案的标题、学科、年级信息
func (s *CoursewareService) CreateCourseware(
	ctx context.Context,
	actor *CoursewareActorContext,
	req *models.CreateCoursewareRequest,
) (*models.Courseware, error) {
	if req == nil || strings.TrimSpace(req.LessonPlanID) == "" {
		return nil, ErrCoursewareLessonPlanRequired
	}

	lessonPlanID := strings.TrimSpace(req.LessonPlanID)

	// 教案是课件教育域快照的唯一来源。
	// 不得重新根据创建者当前学校推导，否则老师换校后会把历史教案静默重分类。
	lp, err := repository.GetLessonPlanByID(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanNotFound) {
			return nil, ErrCoursewareLessonPlanNotFound
		}
		return nil, fmt.Errorf(
			"查询关联教案失败: %w",
			err,
		)
	}

	domain, err :=
		ResolveCoursewareEducationDomainFromLessonPlan(
			actor,
			lp,
		)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = lp.Title
	}

	cw := &models.Courseware{
		LessonPlanID:    &lessonPlanID,
		UserID:          actor.UserID,
		Title:           title,
		Subject:         lp.Subject,
		Grade:           lp.Grade,
		EducationDomain: domain,
		Status:          models.CoursewareStatusDraft,
		SourceType:      models.CWSourceLessonPlan,
		PageCount:       0,
	}

	if err := repository.CreateCourseware(ctx, cw); err != nil {
		return nil, fmt.Errorf(
			"创建课件失败: %w",
			err,
		)
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
		EducationDomain: cw.EducationDomain,
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
		KPCodes:            cw.KPCodes,
		Pages:              pages,
		CreatedAt:          cw.CreatedAt,
		UpdatedAt:          cw.UpdatedAt,
	}
	return resp, nil
}

// GetCoursewareForView 安全获取课件详情。
//
// 先统一验证作者、admin、同域共享成员或同域集体备课参与者的查看权，
// 再复用原GetCourseware完成详情装配。原GetCourseware暂保留给已经完成
// 独立业务权限校验的审核服务内部调用；HTTP普通详情只能调用本方法。
func (s *CoursewareService) GetCoursewareForView(
	ctx context.Context,
	id string,
	actor *CoursewareActorContext,
) (*models.CoursewareDetailResponse, error) {
	if _, err := s.LoadCoursewareForView(
		ctx,
		id,
		actor,
	); err != nil {
		return nil, err
	}

	return s.GetCourseware(ctx, id)
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
//     这样新课件首次流程完全不变，老课件改方案不再被踢出后续步骤。
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
