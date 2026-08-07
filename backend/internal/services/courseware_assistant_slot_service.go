package services

// courseware_assistant_slot_service.go
//
// 本文件提供教师端课件教学智能体插槽Service：
//   - 列出课件全部插槽；
//   - 按稳定页面ID读取插槽；
//   - 作者创建插槽；
//   - 作者更新插槽；
//   - 作者删除插槽。
//
// 本单元不包含页面上下文构建、AI方案生成、HTTP请求解析、
// 部署、匿名运行时或计费。
//
// 所有浏览器响应均使用CoursewareAssistantSlotView，
// 不包含助手full_prompt、页面完整HTML或来源教案全文。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// CoursewareAssistantSlotService 是教师端插槽业务服务。
type CoursewareAssistantSlotService struct {
	coursewareService *CoursewareService
	assistantService  *AIAssistantService
}

// NewCoursewareAssistantSlotService 创建默认插槽服务。
func NewCoursewareAssistantSlotService() *CoursewareAssistantSlotService {
	return &CoursewareAssistantSlotService{
		coursewareService: NewCoursewareService(),
		assistantService:  NewAIAssistantService(),
	}
}

// NewCoursewareAssistantSlotServiceWithDependencies 创建可注入依赖的插槽服务。
func NewCoursewareAssistantSlotServiceWithDependencies(
	coursewareService *CoursewareService,
	assistantService *AIAssistantService,
) *CoursewareAssistantSlotService {
	return &CoursewareAssistantSlotService{
		coursewareService: coursewareService,
		assistantService:  assistantService,
	}
}

// ListCoursewareAssistantSlots 列出可查看课件的安全插槽摘要。
func (s *CoursewareAssistantSlotService) ListCoursewareAssistantSlots(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.CoursewareAssistantSlotListResponse,
	error,
) {
	if _, err :=
		s.loadCoursewareForAssistantRead(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return nil, err
	}

	items, err :=
		repository.ListCoursewareAssistantSlotsByCoursewareID(
			ctx,
			strings.TrimSpace(coursewareID),
		)
	if err != nil {
		return nil,
			mapCoursewareAssistantRepositoryError(err)
	}

	if items == nil {
		items =
			[]*models.CoursewareAssistantSlotView{}
	}

	return &models.CoursewareAssistantSlotListResponse{
		Slots: items,
		Total: len(items),
	}, nil
}

// GetCoursewareAssistantSlotByPage 读取当前页唯一安全插槽摘要。
func (s *CoursewareAssistantSlotService) GetCoursewareAssistantSlotByPage(
	ctx context.Context,
	coursewareID string,
	pageID string,
	actor *CoursewareActorContext,
) (
	*models.CoursewareAssistantSlotView,
	error,
) {
	if strings.TrimSpace(pageID) == "" {
		return nil,
			ErrCoursewareAssistantInvalidRequest
	}

	if _, err :=
		s.loadCoursewareForAssistantRead(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return nil, err
	}

	item, err :=
		repository.GetCoursewareAssistantSlotByPage(
			ctx,
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(pageID),
		)
	if err != nil {
		return nil,
			mapCoursewareAssistantRepositoryError(err)
	}

	return item, nil
}

// CreateCoursewareAssistantSlot 由课件作者为稳定页面创建唯一插槽。
func (s *CoursewareAssistantSlotService) CreateCoursewareAssistantSlot(
	ctx context.Context,
	coursewareID string,
	pageID string,
	actor *CoursewareActorContext,
	request *models.CreateCoursewareAssistantSlotRequest,
) (
	*models.CoursewareAssistantSlotView,
	error,
) {
	if strings.TrimSpace(pageID) == "" {
		return nil,
			ErrCoursewareAssistantInvalidRequest
	}

	courseware,
		scopedActor,
		err :=
		s.loadCoursewareForAssistantWrite(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	if err := prepareCoursewareAssistantCreateRequest(
		request,
	); err != nil {
		return nil, err
	}

	if err := s.validateCoursewareAssistantSelection(
		ctx,
		courseware,
		scopedActor,
		request.AssistantID,
	); err != nil {
		return nil, err
	}

	item, err :=
		repository.CreateCoursewareAssistantSlot(
			ctx,
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(pageID),
			scopedActor.UserID,
			request,
		)
	if err != nil {
		return nil,
			mapCoursewareAssistantRepositoryError(err)
	}

	return item, nil
}

// UpdateCoursewareAssistantSlot 由课件作者更新插槽可编辑字段。
//
// page_id不来自请求正文，而是从正式插槽记录重新解析。
func (s *CoursewareAssistantSlotService) UpdateCoursewareAssistantSlot(
	ctx context.Context,
	coursewareID string,
	slotID string,
	actor *CoursewareActorContext,
	request *models.UpdateCoursewareAssistantSlotRequest,
) (
	*models.CoursewareAssistantSlotView,
	error,
) {
	courseware,
		scopedActor,
		err :=
		s.loadCoursewareForAssistantWrite(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	existing, err :=
		resolveCoursewareAssistantSlotByID(
			ctx,
			coursewareID,
			slotID,
		)
	if err != nil {
		return nil, err
	}

	if err := prepareCoursewareAssistantUpdateRequest(
		request,
	); err != nil {
		return nil, err
	}

	if err := s.validateCoursewareAssistantSelection(
		ctx,
		courseware,
		scopedActor,
		request.AssistantID,
	); err != nil {
		return nil, err
	}

	item, err :=
		repository.UpdateCoursewareAssistantSlot(
			ctx,
			strings.TrimSpace(coursewareID),
			existing.PageID,
			existing.ID,
			request,
		)
	if err != nil {
		return nil,
			mapCoursewareAssistantRepositoryError(err)
	}

	return item, nil
}

// DeleteCoursewareAssistantSlot 由课件作者硬删除插槽。
//
// 已发布部署不会删除；数据库只会把deployment.slot_id置空。
func (s *CoursewareAssistantSlotService) DeleteCoursewareAssistantSlot(
	ctx context.Context,
	coursewareID string,
	slotID string,
	actor *CoursewareActorContext,
) error {
	if _, _, err :=
		s.loadCoursewareForAssistantWrite(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return err
	}

	existing, err :=
		resolveCoursewareAssistantSlotByID(
			ctx,
			coursewareID,
			slotID,
		)
	if err != nil {
		return err
	}

	if err := repository.DeleteCoursewareAssistantSlot(
		ctx,
		strings.TrimSpace(coursewareID),
		existing.PageID,
		existing.ID,
	); err != nil {
		return mapCoursewareAssistantRepositoryError(
			err,
		)
	}

	return nil
}
