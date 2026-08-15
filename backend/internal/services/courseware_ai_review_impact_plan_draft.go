package services

// courseware_ai_review_impact_plan_draft.go
//
// R-07结构化影响方案草稿编排。
//
// 流程：
//   1. 重新授权已完成AI审核会话并校验当前快照；
//   2. 按message_id重新读取可信assistant消息；
//   3. 重新加载消息中selected_item_ids对应的当前整改项；
//   4. 读取当前问题组、成员和pairwise relation治理快照；
//   5. AI仅生成候选operation_type/summary/payload；
//   6. 服务端严格冻结可信preconditions并生成operation_id；
//   7. operations_json一次性写入不可变draft；
//   8. AI不会在本流程中执行任何治理动作。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// CreateCWAIReviewImpactPlanDraft 从一条可信全局assistant消息生成候选影响方案。
func (s *CoursewareAIReviewRunner) CreateCWAIReviewImpactPlanDraft(
	ctx context.Context,
	sessionID string,
	messageID string,
	actor *CoursewareActorContext,
) (*CWAIReviewImpactPlanRecord, error) {
	sessionID = strings.TrimSpace(sessionID)
	messageID = strings.TrimSpace(messageID)

	if sessionID == "" || messageID == "" {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	session, courseware, pageDigests, err :=
		s.authorizeCWAIReviewGlobalDiscussionSession(
			ctx,
			sessionID,
			actor,
			true,
		)
	if err != nil {
		return nil, err
	}

	message, meta, err :=
		loadTrustedCWAIReviewGlobalMessage(
			ctx,
			session,
			messageID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	// R-07的relation操作必须依赖带明确source/target方向的可信协议。
	// 老版本全局消息可以继续展示，但不能作为结构化影响方案来源。
	if meta.RelationSchemaVersion !=
		cwAIReviewGlobalRelationSchemaVersion {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	items, err := loadCWAIReviewGlobalSelectedItems(
		ctx,
		session,
		meta.SelectedItemIDs,
		actor,
	)
	if err != nil {
		return nil, err
	}

	groups, relations, err :=
		repository.LoadCoursewareReviewImpactGovernanceSnapshot(
			ctx,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	aiResponse, _, err :=
		s.generateCWAIReviewImpactPlan(
			ctx,
			session,
			courseware,
			pageDigests,
			items,
			message,
			meta,
			groups,
			relations,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	operations, err := freezeCWAIReviewImpactOperations(
		aiResponse.Operations,
		items,
		pageDigests,
		meta,
		groups,
		relations,
	)
	if err != nil {
		return nil, err
	}

	operationsJSON, err :=
		marshalCWAIReviewImpactOperations(operations)
	if err != nil {
		return nil, err
	}

	plan, err :=
		repository.CreateCoursewareReviewImpactPlanDraft(
			ctx,
			courseware.ID,
			session.ID,
			message.ID,
			actor.UserID,
			operationsJSON,
		)
	if err != nil {
		return nil, err
	}

	events, err :=
		repository.ListCoursewareReviewImpactPlanEvents(
			ctx,
			plan.ID,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return &CWAIReviewImpactPlanRecord{
		Plan:       plan,
		Operations: operations,
		Events:     events,
	}, nil
}

// GetCWAIReviewImpactPlan 读取一条已经冻结的影响方案。
// 返回时只解析数据库中的不可变operations_json，不接受浏览器提供operation正文。
func (s *CoursewareAIReviewRunner) GetCWAIReviewImpactPlan(
	ctx context.Context,
	sessionID string,
	planID string,
	actor *CoursewareActorContext,
) (*CWAIReviewImpactPlanRecord, error) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCWAIReviewActorRequired
	}

	session, _, _, err :=
		s.authorizeCWAIReviewGlobalDiscussionSession(
			ctx,
			sessionID,
			actor,
			false,
		)
	if err != nil {
		return nil, err
	}

	plan, err :=
		repository.GetCoursewareReviewImpactPlanByID(
			ctx,
			strings.TrimSpace(planID),
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	operations, err :=
		parseCWAIReviewImpactOperationsJSON(
			plan.OperationsJSON,
		)
	if err != nil {
		return nil, err
	}

	events, err :=
		repository.ListCoursewareReviewImpactPlanEvents(
			ctx,
			plan.ID,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return &CWAIReviewImpactPlanRecord{
		Plan:       plan,
		Operations: operations,
		Events:     events,
	}, nil
}

// mapCWAIReviewImpactPlanRepositoryError 后续Handler与Apply共用。
// 当前只把Repository的plan CAS冲突收敛成Service层冲突语义。
func mapCWAIReviewImpactPlanRepositoryError(
	err error,
) error {
	switch {
	case errors.Is(
		err,
		repository.ErrCoursewareReviewImpactPlanConflict,
	):
		return ErrCWAIReviewImpactPlanConflict

	default:
		return err
	}
}

// keep imports of models explicit for the R-07 contract package boundary.
var _ = models.CWReviewImpactPlanStatusDraft
