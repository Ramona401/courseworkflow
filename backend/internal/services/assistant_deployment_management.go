package services

// assistant_deployment_management.go
//
// 本文件提供教师端部署历史查询、版本元数据查询、暂停、恢复、永久撤销
// 和实时策略更新。
//
// 安全边界：
//   - 所有操作都以登录用户绑定owner_user_id；
//   - 列表与版本响应只返回浏览器安全字段；
//   - revoked部署没有恢复路径；
//   - 更新策略不改变public_id、课件、页面、创建者、学校、教育域或历史版本；
//   - 暂停和撤销不受课件草稿审核锁阻断，确保教师可以随时紧急停用公开运行。

import (
	"context"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ListAssistantDeployments 返回作者课件的部署历史。
func (s *AssistantDeploymentService) ListAssistantDeployments(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.AssistantDeploymentListResponse,
	error,
) {
	if err := validateAssistantDeploymentActor(actor); err != nil {
		return nil, err
	}

	courseware, _, err := s.resolveCoursewareService().
		LoadCoursewareForOwnerRuntime(
			ctx,
			strings.TrimSpace(coursewareID),
			actor,
		)
	if err != nil {
		return nil,
			mapCoursewareAssistantWriteAccessError(err)
	}

	records, err := repository.ListAssistantDeploymentsByCoursewareForOwner(
		ctx,
		courseware.ID,
		actor.UserID,
	)
	if err != nil {
		return nil, err
	}

	views := make([]*models.AssistantDeploymentView, 0, len(records))
	for _, record := range records {
		view, viewErr := assistantDeploymentViewFromRecord(record)
		if viewErr != nil {
			return nil, viewErr
		}
		views = append(views, view)
	}

	return &models.AssistantDeploymentListResponse{
		Deployments: views,
		Total:       len(views),
	}, nil
}

// ListAssistantDeploymentVersions 返回版本哈希历史，不返回快照正文。
func (s *AssistantDeploymentService) ListAssistantDeploymentVersions(
	ctx context.Context,
	deploymentID string,
	actor *CoursewareActorContext,
) (
	[]*models.AssistantDeploymentVersionView,
	error,
) {
	deployment, err := s.loadOwnedAssistantDeployment(
		ctx,
		deploymentID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	records, err := repository.ListAssistantDeploymentVersionsForOwner(
		ctx,
		deployment.ID,
		deployment.OwnerUserID,
	)
	if err != nil {
		return nil, err
	}

	views := make(
		[]*models.AssistantDeploymentVersionView,
		0,
		len(records),
	)
	for _, record := range records {
		views = append(
			views,
			assistantDeploymentVersionViewFromRecord(record),
		)
	}

	return views, nil
}

// PauseAssistantDeployment 暂停部署。
func (s *AssistantDeploymentService) PauseAssistantDeployment(
	ctx context.Context,
	deploymentID string,
	actor *CoursewareActorContext,
) (*models.AssistantDeploymentView, error) {
	return s.transitionOwnedAssistantDeployment(
		ctx,
		deploymentID,
		actor,
		models.AssistantDeploymentStatusPaused,
	)
}

// ResumeAssistantDeployment 恢复paused部署。
func (s *AssistantDeploymentService) ResumeAssistantDeployment(
	ctx context.Context,
	deploymentID string,
	actor *CoursewareActorContext,
) (*models.AssistantDeploymentView, error) {
	return s.transitionOwnedAssistantDeployment(
		ctx,
		deploymentID,
		actor,
		models.AssistantDeploymentStatusActive,
	)
}

// RevokeAssistantDeployment 永久撤销部署。
func (s *AssistantDeploymentService) RevokeAssistantDeployment(
	ctx context.Context,
	deploymentID string,
	actor *CoursewareActorContext,
) (*models.AssistantDeploymentView, error) {
	return s.transitionOwnedAssistantDeployment(
		ctx,
		deploymentID,
		actor,
		models.AssistantDeploymentStatusRevoked,
	)
}

// UpdateAssistantDeploymentPolicy 更新当前部署实时运行策略。
func (s *AssistantDeploymentService) UpdateAssistantDeploymentPolicy(
	ctx context.Context,
	deploymentID string,
	actor *CoursewareActorContext,
	request *models.UpdateAssistantDeploymentPolicyRequest,
) (*models.AssistantDeploymentView, error) {
	deployment, err := s.loadOwnedAssistantDeployment(
		ctx,
		deploymentID,
		actor,
	)
	if err != nil {
		return nil, err
	}
	if request == nil {
		return nil,
			ErrAssistantDeploymentPolicyInvalid
	}

	policy, err := normalizeAssistantDeploymentPolicy(
		request.DailyCallLimit,
		request.PerSessionTurnLimit,
		request.AllowedOrigins,
		request.ValidUntil,
		time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}

	updated, err := repository.UpdateAssistantDeploymentPolicy(
		ctx,
		deployment.ID,
		deployment.OwnerUserID,
		policy.DailyCallLimit,
		policy.PerSessionTurnLimit,
		policy.AllowedOriginsJSON,
		policy.ValidUntil,
	)
	if err != nil {
		return nil, err
	}

	return assistantDeploymentViewFromRecord(updated)
}

// loadOwnedAssistantDeployment 按登录用户绑定部署创建者。
func (s *AssistantDeploymentService) loadOwnedAssistantDeployment(
	ctx context.Context,
	deploymentID string,
	actor *CoursewareActorContext,
) (*models.AssistantDeployment, error) {
	if err := validateAssistantDeploymentActor(actor); err != nil {
		return nil, err
	}

	return repository.GetAssistantDeploymentForOwner(
		ctx,
		strings.TrimSpace(deploymentID),
		strings.TrimSpace(actor.UserID),
	)
}

// transitionOwnedAssistantDeployment 执行固定状态机。
func (s *AssistantDeploymentService) transitionOwnedAssistantDeployment(
	ctx context.Context,
	deploymentID string,
	actor *CoursewareActorContext,
	targetStatus string,
) (*models.AssistantDeploymentView, error) {
	deployment, err := s.loadOwnedAssistantDeployment(
		ctx,
		deploymentID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	var updated *models.AssistantDeployment
	switch targetStatus {
	case models.AssistantDeploymentStatusPaused:
		updated, err = repository.PauseAssistantDeployment(
			ctx,
			deployment.ID,
			deployment.OwnerUserID,
		)
	case models.AssistantDeploymentStatusActive:
		updated, err = repository.ResumeAssistantDeployment(
			ctx,
			deployment.ID,
			deployment.OwnerUserID,
		)
	case models.AssistantDeploymentStatusRevoked:
		updated, err = repository.RevokeAssistantDeployment(
			ctx,
			deployment.ID,
			deployment.OwnerUserID,
		)
	default:
		return nil,
			repository.ErrAssistantDeploymentStateConflict
	}
	if err != nil {
		return nil, err
	}

	return assistantDeploymentViewFromRecord(updated)
}
