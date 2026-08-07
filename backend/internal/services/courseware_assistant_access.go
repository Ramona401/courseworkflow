package services

// courseware_assistant_access.go
//
// 本文件建立教师端课件教学智能体插槽的可信访问边界。
//
// 读取规则：
//   - 复用课件现有查看权限；
//   - 作者、明确获得查看权的协作成员可以读取安全插槽摘要；
//   - 不读取或返回助手完整提示词。
//
// 写入规则：
//   - 只有课件作者本人可以创建、更新或删除插槽；
//   - admin不会因为平台角色自动取得教师课件插槽写权限；
//   - 作者换校后继续使用课件历史education_domain快照；
//   - submitted与in_pipeline状态禁止修改插槽；
//   - 写入前重新读取正式课件，不信任前端提交的owner、school或教育域。
//
// 助手规则：
//   - assistant_id允许为空，表示先建立纯结构化教学方案；
//   - 非空助手必须存在、启用、当前作者可见；
//   - 助手必须与课件历史教育域兼容；
//   - 助手学科必须匹配课件学科；
//   - MVP复用既有workshop_design场景作为教师方案设计候选场景；
//   - use_only等prompt_protected助手仍可使用，但本服务不返回提示词正文。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// CoursewareAssistantSelectionScene 是MVP选择已有助手时使用的既有场景。
//
// 本单元不扩大全局AI助手场景枚举，避免顺手改变现有助手创建与列表行为。
// 教学智能体运行时的苏格拉底协议由插槽方案和后续不可变发布快照控制。
const CoursewareAssistantSelectionScene = models.SceneWorkshopDesign

var (
	// ErrCoursewareAssistantActorRequired 表示缺少可信登录Actor。
	ErrCoursewareAssistantActorRequired = errors.New(
		"缺少可信课件教学智能体操作上下文",
	)

	// ErrCoursewareAssistantCoursewareNotFound 表示课件不存在或已经进入回收站。
	ErrCoursewareAssistantCoursewareNotFound = errors.New(
		"课件教学智能体所属课件不存在",
	)

	// ErrCoursewareAssistantReadDenied 表示当前用户无权读取课件插槽。
	ErrCoursewareAssistantReadDenied = errors.New(
		"无权查看此课件的教学智能体配置",
	)

	// ErrCoursewareAssistantWriteDenied 表示当前用户不是课件作者。
	ErrCoursewareAssistantWriteDenied = errors.New(
		"只有课件作者本人可以修改教学智能体配置",
	)

	// ErrCoursewareAssistantMutationLocked 表示课件正在正式审核流程中。
	ErrCoursewareAssistantMutationLocked = errors.New(
		"课件正在审核流程中，暂不允许修改教学智能体配置",
	)

	// ErrCoursewareAssistantCoursewareDomainInvalid 表示课件缺少具体运行教育域。
	ErrCoursewareAssistantCoursewareDomainInvalid = errors.New(
		"课件教学智能体运行教育域无效",
	)

	// ErrCoursewareAssistantSlotNotFound 表示插槽不存在。
	ErrCoursewareAssistantSlotNotFound = errors.New(
		"课件教学智能体插槽不存在",
	)

	// ErrCoursewareAssistantSlotAlreadyExists 表示当前页面已有插槽。
	ErrCoursewareAssistantSlotAlreadyExists = errors.New(
		"当前课件页面已经存在教学智能体插槽",
	)

	// ErrCoursewareAssistantPageNotFound 表示稳定页面不存在或不属于课件。
	ErrCoursewareAssistantPageNotFound = errors.New(
		"课件教学智能体目标页面不存在",
	)

	// ErrCoursewareAssistantCreatorNotFound 表示可信创建者不存在。
	ErrCoursewareAssistantCreatorNotFound = errors.New(
		"课件教学智能体创建者不存在",
	)

	// ErrCoursewareAssistantAssistantNotFound 表示选择的AI助手不存在。
	ErrCoursewareAssistantAssistantNotFound = errors.New(
		"选择的AI助手不存在",
	)

	// ErrCoursewareAssistantAssistantInactive 表示助手已停用。
	ErrCoursewareAssistantAssistantInactive = errors.New(
		"选择的AI助手已停用",
	)

	// ErrCoursewareAssistantAssistantDomainMismatch 表示助手资源域不兼容。
	ErrCoursewareAssistantAssistantDomainMismatch = errors.New(
		"选择的AI助手与课件教育域不匹配",
	)

	// ErrCoursewareAssistantAssistantScopeMismatch 表示助手学科或场景不兼容。
	ErrCoursewareAssistantAssistantScopeMismatch = errors.New(
		"选择的AI助手不适用于当前课件学科或教学智能体场景",
	)

	// ErrCoursewareAssistantAssistantUseDenied 表示作者无权使用该助手。
	ErrCoursewareAssistantAssistantUseDenied = errors.New(
		"当前用户无权使用选择的AI助手",
	)

	// ErrCoursewareAssistantInvalidRequest 表示请求协议不合法。
	ErrCoursewareAssistantInvalidRequest = errors.New(
		"课件教学智能体配置无效",
	)
)

// coursewareAssistantCoursewareService 返回课件访问服务。
//
// 支持零值Service，便于后续Handler依赖注入前保持简单稳定。
func (s *CoursewareAssistantSlotService) coursewareAssistantCoursewareService() *CoursewareService {
	if s != nil && s.coursewareService != nil {
		return s.coursewareService
	}

	return NewCoursewareService()
}

// coursewareAssistantAIService 返回AI助手校验服务。
func (s *CoursewareAssistantSlotService) coursewareAssistantAIService() *AIAssistantService {
	if s != nil && s.assistantService != nil {
		return s.assistantService
	}

	return NewAIAssistantService()
}

// loadCoursewareForAssistantRead 加载并验证插槽读取权。
func (s *CoursewareAssistantSlotService) loadCoursewareForAssistantRead(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	error,
) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil,
			ErrCoursewareAssistantActorRequired
	}

	courseware, err :=
		s.coursewareAssistantCoursewareService().
			LoadCoursewareForView(
				ctx,
				strings.TrimSpace(coursewareID),
				actor,
			)
	if err != nil {
		return nil,
			mapCoursewareAssistantReadAccessError(err)
	}

	if !models.IsTeachingEducationDomain(
		courseware.EducationDomain,
	) {
		return nil,
			ErrCoursewareAssistantCoursewareDomainInvalid
	}

	return courseware, nil
}

// loadCoursewareForAssistantWrite 加载作者课件、收敛Actor并检查写锁。
func (s *CoursewareAssistantSlotService) loadCoursewareForAssistantWrite(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*CoursewareActorContext,
	error,
) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil,
			nil,
			ErrCoursewareAssistantActorRequired
	}

	courseware,
		scopedActor,
		err :=
		s.coursewareAssistantCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				strings.TrimSpace(coursewareID),
				actor,
			)
	if err != nil {
		return nil,
			nil,
			mapCoursewareAssistantWriteAccessError(err)
	}

	if err := validateCoursewareControlMutationState(
		courseware,
	); err != nil {
		return nil,
			nil,
			ErrCoursewareAssistantMutationLocked
	}

	return courseware,
		scopedActor,
		nil
}

// validateCoursewareAssistantSelection 校验可选助手。
//
// 使用ValidateAssistantForManualLesson而不是公开详情接口：
//   - 内部可读取受保护提示词用于后续发布；
//   - 当前插槽响应仍只返回助手名称和启用状态；
//   - 不递增助手使用次数，只有真实运行才应计数。
func (s *CoursewareAssistantSlotService) validateCoursewareAssistantSelection(
	ctx context.Context,
	courseware *models.Courseware,
	scopedActor *CoursewareActorContext,
	assistantID *string,
) error {
	if assistantID == nil ||
		strings.TrimSpace(*assistantID) == "" {
		return nil
	}

	_, err :=
		s.coursewareAssistantAIService().
			ValidateAssistantForManualLesson(
				ctx,
				scopedActor,
				strings.TrimSpace(*assistantID),
				strings.TrimSpace(
					courseware.Subject,
				),
				CoursewareAssistantSelectionScene,
			)
	if err != nil {
		return mapCoursewareAssistantSelectionError(
			err,
		)
	}

	return nil
}

// resolveCoursewareAssistantSlotByID 在已经完成课件访问校验后定位插槽。
//
// 当前教师端更新和删除路由只携带courseware_id与slot_id，
// 因此Service先从安全列表中解析稳定page_id，再调用仓储三重边界写方法。
func resolveCoursewareAssistantSlotByID(
	ctx context.Context,
	coursewareID string,
	slotID string,
) (
	*models.CoursewareAssistantSlotView,
	error,
) {
	normalizedSlotID :=
		strings.TrimSpace(slotID)
	if normalizedSlotID == "" {
		return nil,
			ErrCoursewareAssistantInvalidRequest
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

	for _, item := range items {
		if item != nil &&
			item.ID == normalizedSlotID {
			return item, nil
		}
	}

	return nil,
		ErrCoursewareAssistantSlotNotFound
}

// mapCoursewareAssistantReadAccessError 映射课件查看授权错误。
func mapCoursewareAssistantReadAccessError(
	err error,
) error {
	switch {
	case errors.Is(
		err,
		ErrCoursewareActorRequired,
	):
		return ErrCoursewareAssistantActorRequired

	case errors.Is(
		err,
		ErrCoursewareAccessNotFound,
	):
		return ErrCoursewareAssistantCoursewareNotFound

	case errors.Is(
		err,
		ErrCoursewareViewDenied,
	),
		errors.Is(
			err,
			ErrCoursewareEducationDomainMismatch,
		):
		return ErrCoursewareAssistantReadDenied

	case errors.Is(
		err,
		ErrCoursewareEducationDomainInvalid,
	),
		errors.Is(
			err,
			ErrCoursewareRuntimeDomainRequired,
		):
		return ErrCoursewareAssistantCoursewareDomainInvalid

	default:
		return err
	}
}

// mapCoursewareAssistantWriteAccessError 映射作者写入授权错误。
func mapCoursewareAssistantWriteAccessError(
	err error,
) error {
	switch {
	case errors.Is(
		err,
		ErrCoursewareActorRequired,
	):
		return ErrCoursewareAssistantActorRequired

	case errors.Is(
		err,
		ErrCoursewareAccessNotFound,
	):
		return ErrCoursewareAssistantCoursewareNotFound

	case errors.Is(
		err,
		ErrCoursewareOwnerRuntimeDenied,
	),
		errors.Is(
			err,
			ErrCoursewareEditDenied,
		),
		errors.Is(
			err,
			ErrCoursewareEducationDomainMismatch,
		):
		return ErrCoursewareAssistantWriteDenied

	case errors.Is(
		err,
		ErrCoursewareEducationDomainInvalid,
	),
		errors.Is(
			err,
			ErrCoursewareRuntimeDomainRequired,
		):
		return ErrCoursewareAssistantCoursewareDomainInvalid

	default:
		return err
	}
}

// mapCoursewareAssistantSelectionError 映射AI助手选择错误。
func mapCoursewareAssistantSelectionError(
	err error,
) error {
	switch {
	case errors.Is(
		err,
		repository.ErrAIAssistantNotFound,
	):
		return ErrCoursewareAssistantAssistantNotFound

	case errors.Is(
		err,
		repository.ErrAIAssistantInactive,
	):
		return ErrCoursewareAssistantAssistantInactive

	case errors.Is(
		err,
		ErrAssistantEducationDomainMismatch,
	):
		return ErrCoursewareAssistantAssistantDomainMismatch

	case errors.Is(
		err,
		ErrAssistantManualMismatch,
	):
		return ErrCoursewareAssistantAssistantScopeMismatch

	case errors.Is(
		err,
		ErrAssistantPermDenied,
	):
		return ErrCoursewareAssistantAssistantUseDenied

	default:
		return err
	}
}

// mapCoursewareAssistantRepositoryError 映射插槽仓储错误。
func mapCoursewareAssistantRepositoryError(
	err error,
) error {
	switch {
	case errors.Is(
		err,
		repository.ErrCoursewareAssistantSlotNotFound,
	):
		return ErrCoursewareAssistantSlotNotFound

	case errors.Is(
		err,
		repository.ErrCoursewareAssistantSlotAlreadyExists,
	):
		return ErrCoursewareAssistantSlotAlreadyExists

	case errors.Is(
		err,
		repository.ErrCoursewareAssistantSlotPageNotFound,
	):
		return ErrCoursewareAssistantPageNotFound

	case errors.Is(
		err,
		repository.ErrCoursewareAssistantSlotCreatorNotFound,
	):
		return ErrCoursewareAssistantCreatorNotFound

	case errors.Is(
		err,
		repository.ErrAIAssistantNotFound,
	):
		return ErrCoursewareAssistantAssistantNotFound

	default:
		return err
	}
}
