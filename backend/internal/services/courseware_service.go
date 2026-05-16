package services

import (
	"context"
	"fmt"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 课件工坊核心服务 ====================
// 课件CRUD + 状态流转 + 索引确认 + 风格保存 + 提交Pipeline

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
		LessonPlanID: req.LessonPlanID,
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
	lp, lpErr := repository.GetLessonPlanByID(ctx, cw.LessonPlanID)
	if lpErr == nil {
		lpTitle = lp.Title
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
		PipelineID:      cw.PipelineID,
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

// DeleteCourseware 删除课件（仅draft状态）
func (s *CoursewareService) DeleteCourseware(ctx context.Context, id string, userID string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if cw.Status != models.CoursewareStatusDraft {
		return fmt.Errorf("仅草稿状态的课件可删除")
	}
	return repository.DeleteCourseware(ctx, id)
}

// ==================== 状态流转 ====================

// ConfirmIndex 确认课件索引，状态从 indexing → styling
func (s *CoursewareService) ConfirmIndex(ctx context.Context, id string, userID string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	// draft或indexing状态都可以确认索引
	if cw.Status != models.CoursewareStatusDraft && cw.Status != models.CoursewareStatusIndexing {
		return fmt.Errorf("当前状态不允许确认索引: %s", cw.Status)
	}
	// 更新页数
	count, _ := repository.CountCoursewarePages(ctx, id)
	if count == 0 {
		return fmt.Errorf("课件没有任何页面，请先生成索引")
	}
	_ = repository.UpdateCoursewarePageCount(ctx, id, count)
	return repository.UpdateCoursewareStatus(ctx, id, models.CoursewareStatusStyling)
}

// SaveStyle 保存风格选择，状态从 styling → generating（准备生成）
func (s *CoursewareService) SaveStyle(ctx context.Context, id string, userID string, styleConfig string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if cw.Status != models.CoursewareStatusStyling && cw.Status != models.CoursewareStatusDraft && cw.Status != models.CoursewareStatusIndexing {
		return fmt.Errorf("当前状态不允许保存风格: %s", cw.Status)
	}
	if styleConfig == "" {
		return fmt.Errorf("风格配置不能为空")
	}
	if err := repository.UpdateCoursewareStyle(ctx, id, styleConfig); err != nil {
		return err
	}
	// 状态不自动前进到generating，需要用户主动触发生成
	return nil
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
	return repository.ReorderCoursewarePages(ctx, coursewareID, pageIDs)
}
