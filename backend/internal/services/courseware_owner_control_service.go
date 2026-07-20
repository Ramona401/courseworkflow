package services

// courseware_owner_control_service.go — 课件作者私有核心控制面Actor包装层
//
// 本文件不重复实现原有成熟业务逻辑，而是在原实现外增加稳定的安全边界：
//
//	可信Actor
//	→ 重新加载正式课件
//	→ 作者本人校验
//	→ 课件历史education_domain快照校验
//	→ 审核锁校验
//	→ 调用既有业务实现
//
// 原有方法仍保留，避免一次性破坏内部调用和历史测试；所有HTTP正式入口
// 必须改走本文件中的ForActor方法。

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"

	"tedna/internal/models"
)

// ErrCoursewareControlMutationLocked 表示课件正在正式审核流程中，
// 核心控制面必须保持只读。
var ErrCoursewareControlMutationLocked = errors.New(
	"课件正在审核流程中，暂不允许修改",
)

// validateCoursewareControlMutationState 校验核心控制面写入状态。
func validateCoursewareControlMutationState(
	courseware *models.Courseware,
) error {
	if courseware == nil {
		return ErrCoursewareEducationDomainInvalid
	}

	if courseware.Status ==
		models.CoursewareStatusInPipeline {
		return fmt.Errorf(
			"%w: 课件已进入Pipeline审核",
			ErrCoursewareControlMutationLocked,
		)
	}

	if courseware.PublishState ==
		models.CWPublishSubmitted {
		return fmt.Errorf(
			"%w: 课件已提交发布审核",
			ErrCoursewareControlMutationLocked,
		)
	}

	return nil
}

// loadOwnedCoursewareForControlMutation 重新加载课件并进入作者私有写控制面。
func (s *CoursewareService) loadOwnedCoursewareForControlMutation(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*CoursewareActorContext,
	error,
) {
	courseware, scopedActor, err :=
		s.LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, nil, err
	}

	if err := validateCoursewareControlMutationState(
		courseware,
	); err != nil {
		return nil, nil, err
	}

	return courseware,
		scopedActor,
		nil
}

// UpdateCoursewareTitleForActor 更新课件标题。
func (s *CoursewareService) UpdateCoursewareTitleForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	title string,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.UpdateCoursewareTitle(
		ctx,
		coursewareID,
		scopedActor.UserID,
		title,
	)
}

// DeleteCoursewareForActor 删除作者自己的课件。
func (s *CoursewareService) DeleteCoursewareForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.DeleteCourseware(
		ctx,
		coursewareID,
		scopedActor.UserID,
	)
}

// ConfirmIndexForActor 确认课件方案。
func (s *CoursewareService) ConfirmIndexForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.ConfirmIndex(
		ctx,
		coursewareID,
		scopedActor.UserID,
	)
}

// SaveStyleFullForActor 保存完整风格设置。
func (s *CoursewareService) SaveStyleFullForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	request *models.SaveStyleFullRequest,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.SaveStyleFull(
		ctx,
		coursewareID,
		scopedActor.UserID,
		request,
	)
}

// SaveStyleForActor 保存旧版风格JSON。
func (s *CoursewareService) SaveStyleForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	styleConfig string,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.SaveStyle(
		ctx,
		coursewareID,
		scopedActor.UserID,
		styleConfig,
	)
}

// ConfirmStyleForActor 确认课件风格。
func (s *CoursewareService) ConfirmStyleForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.ConfirmStyle(
		ctx,
		coursewareID,
		scopedActor.UserID,
	)
}

// SaveNavTemplateForActor 保存课件导航栏模板。
func (s *CoursewareService) SaveNavTemplateForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	navHTML string,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.SaveNavTemplate(
		ctx,
		coursewareID,
		scopedActor.UserID,
		navHTML,
	)
}

// ConfirmCoursewareForActor 确认课件。
func (s *CoursewareService) ConfirmCoursewareForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.ConfirmCourseware(
		ctx,
		coursewareID,
		scopedActor.UserID,
	)
}

// UploadLogoForActor 上传课件Logo。
//
// Service授权发生在旧实现打开目录和写入文件之前。
func (s *CoursewareService) UploadLogoForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	file multipart.File,
	header *multipart.FileHeader,
) (*models.UploadLogoResponse, error) {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return s.UploadLogo(
		ctx,
		coursewareID,
		file,
		header,
		scopedActor.UserID,
	)
}

// UpdatePageIndexForActor 修改单页方案字段。
func (s *CoursewareService) UpdatePageIndexForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNumber int,
	request *models.UpdateCWPageIndexRequest,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.UpdatePageIndex(
		ctx,
		coursewareID,
		pageNumber,
		scopedActor.UserID,
		request,
	)
}

// AddPageForActor 添加课件页面。
func (s *CoursewareService) AddPageForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	request *models.AddCWPageRequest,
) (*models.CoursewarePage, error) {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return s.AddPage(
		ctx,
		coursewareID,
		scopedActor.UserID,
		request,
	)
}

// DeletePageForActor 删除课件页面。
func (s *CoursewareService) DeletePageForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNumber int,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.DeletePage(
		ctx,
		coursewareID,
		pageNumber,
		scopedActor.UserID,
	)
}

// ReorderPagesForActor 重排课件页面。
func (s *CoursewareService) ReorderPagesForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageIDs []string,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.ReorderPages(
		ctx,
		coursewareID,
		scopedActor.UserID,
		pageIDs,
	)
}

// RollbackStatusForActor 回退课件制作步骤。
func (s *CoursewareService) RollbackStatusForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	targetStatus string,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.RollbackStatus(
		ctx,
		coursewareID,
		scopedActor.UserID,
		targetStatus,
	)
}

// SetPublishStateForActor 设置课件发布状态。
func (s *CoursewareService) SetPublishStateForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	target string,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.SetPublishState(
		ctx,
		coursewareID,
		scopedActor.UserID,
		target,
	)
}

// SetCodeShareScopeForActor 设置课件源码开放范围。
func (s *CoursewareService) SetCodeShareScopeForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	scope string,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.SetCodeShareScope(
		ctx,
		coursewareID,
		scopedActor.UserID,
		scope,
	)
}

// StartCollabForActor 发起集体备课。
func (s *CoursewareService) StartCollabForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	members []string,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.StartCollab(
		ctx,
		coursewareID,
		scopedActor.UserID,
		members,
	)
}

// EndCollabForActor 结束集体备课。
//
// 即使课件随后进入审核锁，作者仍应能够结束遗留的集体备课会话，
// 因此本操作只要求作者运行域，不额外套用审核写锁。
func (s *CoursewareService) EndCollabForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	_, scopedActor, err :=
		s.LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.EndCollab(
		ctx,
		coursewareID,
		scopedActor.UserID,
	)
}

// AddCollabMemberForActor 添加集体备课参与者。
func (s *CoursewareService) AddCollabMemberForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	targetUserID string,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.AddCollabMember(
		ctx,
		coursewareID,
		scopedActor.UserID,
		targetUserID,
	)
}

// RemoveCollabMemberForActor 移除集体备课参与者。
func (s *CoursewareService) RemoveCollabMemberForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	targetUserID string,
) error {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	return s.RemoveCollabMember(
		ctx,
		coursewareID,
		scopedActor.UserID,
		targetUserID,
	)
}

// SelectBackgroundForActor 选择课件背景。
func (s *CoursewareBackgroundService) SelectBackgroundForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	setID string,
) (*models.BackgroundSelectionResult, error) {
	controlService := &CoursewareService{}

	_, scopedActor, err :=
		controlService.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return s.SelectBackground(
		ctx,
		coursewareID,
		scopedActor.UserID,
		setID,
	)
}

// ClearBackgroundForActor 清除课件背景。
func (s *CoursewareBackgroundService) ClearBackgroundForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (*models.BackgroundSelectionResult, error) {
	controlService := &CoursewareService{}

	_, scopedActor, err :=
		controlService.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return s.ClearBackground(
		ctx,
		coursewareID,
		scopedActor.UserID,
	)
}

// SetPageBackgroundForActor 设置页级背景。
func (s *CoursewareBackgroundService) SetPageBackgroundForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNumber int,
	backgroundURL string,
	opacity *float64,
	mode string,
) (map[string]interface{}, error) {
	controlService := &CoursewareService{}

	_, scopedActor, err :=
		controlService.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return s.SetPageBackground(
		ctx,
		coursewareID,
		scopedActor.UserID,
		pageNumber,
		backgroundURL,
		opacity,
		mode,
	)
}

// ClearPageBackgroundForActor 清除页级背景。
func (s *CoursewareBackgroundService) ClearPageBackgroundForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNumber int,
) (map[string]interface{}, error) {
	controlService := &CoursewareService{}

	_, scopedActor, err :=
		controlService.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return s.ClearPageBackground(
		ctx,
		coursewareID,
		scopedActor.UserID,
		pageNumber,
	)
}

// SelectFontForActor 选择课件字体。
func (s *CoursewareFontService) SelectFontForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	schemeCode string,
) (*models.FontSelectionResult, error) {
	controlService := &CoursewareService{}

	_, scopedActor, err :=
		controlService.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return s.SelectFont(
		ctx,
		coursewareID,
		scopedActor.UserID,
		schemeCode,
	)
}

// ClearFontForActor 清除课件字体。
func (s *CoursewareFontService) ClearFontForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (*models.FontSelectionResult, error) {
	controlService := &CoursewareService{}

	_, scopedActor, err :=
		controlService.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return s.ClearFont(
		ctx,
		coursewareID,
		scopedActor.UserID,
	)
}
