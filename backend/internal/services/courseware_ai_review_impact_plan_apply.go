package services

// courseware_ai_review_impact_plan_apply.go
//
// R-07教师最终确认影响方案的Service入口。
//
// 浏览器最终只允许提交：
//   - URL中的session_id；
//   - plan_id；
//   - version；
//   - selected_operation_ids。
//
// 浏览器不能提交：
//   - operations_json；
//   - operation payload；
//   - preconditions；
//   - AI正文；
//   - source_message_id；
//   - actor身份字段。
//
// 当前第一阶段只接入四类问题组operation。
// 后续整改项和relation operation继续接入同一Repository原子事务。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/repository"
)

// ApplyCWAIReviewImpactPlan 执行教师明确勾选后的原子影响方案。
func (s *CoursewareAIReviewRunner) ApplyCWAIReviewImpactPlan(
	ctx context.Context,
	sessionID string,
	planID string,
	version int,
	selectedOperationIDs []string,
	actor *CoursewareActorContext,
) (*CWAIReviewImpactPlanRecord, error) {
	sessionID = strings.TrimSpace(sessionID)
	planID = strings.TrimSpace(planID)

	if sessionID == "" ||
		planID == "" ||
		version != 1 {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	normalizedSelectedIDs, err :=
		normalizeCWAIReviewImpactSelectedOperationIDs(
			selectedOperationIDs,
		)
	if err != nil {
		return nil, err
	}

	// 最终Apply属于写操作，继续沿用全局讨论最严格的当前快照授权。
	session, _, _, err :=
		s.authorizeCWAIReviewGlobalDiscussionSession(
			ctx,
			sessionID,
			actor,
			true,
		)
	if err != nil {
		return nil, err
	}

	appliedPlan, err :=
		repository.ApplyCoursewareReviewImpactPlan(
			ctx,
			planID,
			session.ID,
			version,
			actor.UserID,
			normalizedSelectedIDs,
		)
	if err != nil {
		return nil,
			mapCWAIReviewImpactApplyRepositoryError(
				err,
			)
	}

	operations, err :=
		parseCWAIReviewImpactOperationsJSON(
			appliedPlan.OperationsJSON,
		)
	if err != nil {
		return nil, err
	}

	events, err :=
		repository.ListCoursewareReviewImpactPlanEvents(
			ctx,
			appliedPlan.ID,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return &CWAIReviewImpactPlanRecord{
		Plan:       appliedPlan,
		Operations: operations,
		Events:     events,
	}, nil
}

func normalizeCWAIReviewImpactSelectedOperationIDs(
	values []string,
) ([]string, error) {
	if len(values) == 0 ||
		len(values) > cwAIReviewImpactPlanMaxOperations {
		return nil,
			ErrCWAIReviewImpactPlanNoOperations
	}

	result := make(
		[]string,
		0,
		len(values),
	)
	seen := make(
		map[string]struct{},
		len(values),
	)

	for _, raw := range values {
		operationID := strings.TrimSpace(raw)

		if !cwAIReviewImpactOperationIDPattern.MatchString(
			operationID,
		) {
			return nil,
				ErrCWAIReviewImpactPlanInvalid
		}

		if _, exists := seen[operationID]; exists {
			return nil,
				ErrCWAIReviewImpactPlanInvalid
		}

		seen[operationID] = struct{}{}
		result = append(result, operationID)
	}

	return result, nil
}

func mapCWAIReviewImpactApplyRepositoryError(
	err error,
) error {
	switch {
	case errors.Is(
		err,
		repository.ErrCoursewareReviewImpactPlanConflict,
	),
		errors.Is(
			err,
			repository.ErrCoursewareReviewItemGroupConflict,
		),
		errors.Is(
			err,
			repository.ErrCoursewareReviewItemConflict,
		):
		return ErrCWAIReviewImpactPlanConflict

	case errors.Is(
		err,
		repository.ErrCoursewareReviewImpactSelectionInvalid,
	):
		return ErrCWAIReviewImpactPlanInvalid

	default:
		return err
	}
}
