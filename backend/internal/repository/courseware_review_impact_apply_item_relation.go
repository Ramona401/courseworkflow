package repository

// courseware_review_impact_apply_item_relation.go
//
// R-07 Atomic Apply第二部分：整改项和pairwise relation操作的协议、准备与锁定总控。
//
// 接入：
//   - create_relation
//   - cancel_relation
//   - create_item
//   - dismiss_item
//   - update_candidate_suggestion
//
// 职责：
//   1. 定义数据库冻结payload/preconditions的严格类型；
//   2. cancel_relation按既有锁序先锁relation行；
//   3. 解析全部选中operation并声明其现有item资源；
//   4. 按稳定item ID顺序统一锁定整改项；
//   5. 调度relation和item两类最终precondition验证。
//
// 关系验证位于courseware_review_impact_apply_relation_validate.go。
// 整改项验证位于courseware_review_impact_apply_item_validate.go。
// 本文件不执行业务写入。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/models"
)

type cwReviewImpactRelationPrecondition struct {
	ExpectedAbsent bool   `json:"expected_absent,omitempty"`
	RelationID     string `json:"relation_id,omitempty"`
	Status         string `json:"status,omitempty"`
	Version        int    `json:"version,omitempty"`
}

type cwReviewImpactCreateRelationPayload struct {
	RelationType string `json:"relation_type"`
	SourceItemID string `json:"source_item_id"`
	TargetItemID string `json:"target_item_id"`
	Explanation  string `json:"explanation"`
}

type cwReviewImpactCancelRelationPayload struct {
	RelationID string `json:"relation_id"`
	Reason     string `json:"reason"`
}

type cwReviewImpactCreateItemPayload struct {
	PageID               string `json:"page_id"`
	Severity             string `json:"severity"`
	Dimension            string `json:"dimension"`
	Title                string `json:"title"`
	Description          string `json:"description"`
	CandidateInstruction string `json:"candidate_instruction"`
}

type cwReviewImpactDismissItemPayload struct {
	ItemID string `json:"item_id"`
	Reason string `json:"reason"`
}

type cwReviewImpactUpdateCandidatePayload struct {
	ItemID               string `json:"item_id"`
	CandidateInstruction string `json:"candidate_instruction"`
}

type cwReviewImpactCreateRelationPreconditions struct {
	SourceItem cwReviewImpactItemPrecondition     `json:"source_item"`
	TargetItem cwReviewImpactItemPrecondition     `json:"target_item"`
	Relation   cwReviewImpactRelationPrecondition `json:"relation"`
}

type cwReviewImpactCancelRelationPreconditions struct {
	Relation cwReviewImpactRelationPrecondition `json:"relation"`
}

type cwReviewImpactCreateItemPreconditions struct {
	Page cwReviewImpactPagePrecondition `json:"page"`
}

type cwReviewImpactItemPreconditions struct {
	Item cwReviewImpactItemPrecondition `json:"item"`
}

type cwReviewImpactPreparedItemRelationOperation struct {
	OperationID   string
	OperationType string

	CreateRelation  *cwReviewImpactPreparedCreateRelation
	CancelRelation  *cwReviewImpactPreparedCancelRelation
	CreateItem      *cwReviewImpactPreparedCreateItem
	DismissItem     *cwReviewImpactPreparedDismissItem
	UpdateCandidate *cwReviewImpactPreparedUpdateCandidate
}

type cwReviewImpactPreparedCreateRelation struct {
	Payload       cwReviewImpactCreateRelationPayload
	Preconditions cwReviewImpactCreateRelationPreconditions

	CurrentRelation *models.CoursewareReviewItemRelation
}

type cwReviewImpactPreparedCancelRelation struct {
	Payload       cwReviewImpactCancelRelationPayload
	Preconditions cwReviewImpactCancelRelationPreconditions

	CurrentRelation *models.CoursewareReviewItemRelation
}

type cwReviewImpactPreparedCreateItem struct {
	Payload       cwReviewImpactCreateItemPayload
	Preconditions cwReviewImpactCreateItemPreconditions

	ReviewLevel int
	SourceType  string
	OwnerID     string

	PageUpdatedAt *time.Time
}

type cwReviewImpactPreparedDismissItem struct {
	Payload       cwReviewImpactDismissItemPayload
	Preconditions cwReviewImpactItemPreconditions
}

type cwReviewImpactPreparedUpdateCandidate struct {
	Payload       cwReviewImpactUpdateCandidatePayload
	Preconditions cwReviewImpactItemPreconditions
}

func isCoursewareReviewImpactItemRelationOperation(
	operationType string,
) bool {
	switch strings.TrimSpace(operationType) {
	case models.CWReviewImpactOperationCreateRelation,
		models.CWReviewImpactOperationCancelRelation,
		models.CWReviewImpactOperationCreateItem,
		models.CWReviewImpactOperationDismissItem,
		models.CWReviewImpactOperationUpdateCandidateSuggestion:
		return true

	default:
		return false
	}
}

// prelockCoursewareReviewImpactCancelRelationsTx 保持既有取消关系锁序：
// relation行 -> endpoint items。
//
// 必须在group/item统一锁定阶段之前调用，避免与已有Cancel入口形成反向锁环。
func prelockCoursewareReviewImpactCancelRelationsTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operations []models.CoursewareReviewImpactOperation,
	actorID string,
) (
	map[string]*models.CoursewareReviewItemRelation,
	error,
) {
	relationIDs := make(map[string]struct{})

	for _, operation := range operations {
		if operation.OperationType != models.CWReviewImpactOperationCancelRelation {
			continue
		}

		var payload cwReviewImpactCancelRelationPayload
		if err := decodeCoursewareReviewImpactMap(
			operation.Payload,
			&payload,
		); err != nil {
			return nil, err
		}

		var preconditions cwReviewImpactCancelRelationPreconditions
		if err := decodeCoursewareReviewImpactMap(
			operation.Preconditions,
			&preconditions,
		); err != nil {
			return nil, err
		}

		payload.RelationID = strings.TrimSpace(payload.RelationID)
		preconditions.Relation.RelationID = strings.TrimSpace(preconditions.Relation.RelationID)

		if payload.RelationID == "" || payload.RelationID != preconditions.Relation.RelationID {
			return nil, ErrCoursewareReviewImpactSelectionInvalid
		}

		if _, exists := relationIDs[payload.RelationID]; exists {
			return nil, ErrCoursewareReviewImpactSelectionInvalid
		}

		relationIDs[payload.RelationID] = struct{}{}
	}

	result := make(
		map[string]*models.CoursewareReviewItemRelation,
		len(relationIDs),
	)

	for _, relationID := range sortedCoursewareReviewImpactKeys(relationIDs) {
		relation, err := scanCoursewareReviewItemRelation(
			tx.QueryRow(
				ctx,
				`SELECT `+cwReviewItemRelationSelectColumns+`
				 FROM courseware_review_item_relations
				 WHERE id = $1
				   AND source_session_id = $2
				   AND created_by = $3
				 FOR UPDATE`,
				relationID,
				plan.SourceSessionID,
				strings.TrimSpace(actorID),
			),
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrCoursewareReviewImpactPlanConflict
			}

			return nil, fmt.Errorf(
				"预锁定影响方案取消关系失败: %w",
				err,
			)
		}

		if relation.CoursewareID != plan.CoursewareID {
			return nil, ErrCoursewareReviewImpactPlanConflict
		}

		result[relationID] = relation
	}

	return result, nil
}

func prevalidateCoursewareReviewImpactItemRelationOperationsTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operations []models.CoursewareReviewImpactOperation,
	actorID string,
	prelockedCancelRelations map[string]*models.CoursewareReviewItemRelation,
) (
	[]cwReviewImpactPreparedItemRelationOperation,
	error,
) {
	prepared := make(
		[]cwReviewImpactPreparedItemRelationOperation,
		0,
		len(operations),
	)

	itemIDs := make(map[string]struct{})
	semanticClaims := make(map[string]struct{})

	for _, operation := range operations {
		value, err := prepareCoursewareReviewImpactItemRelationOperation(operation)
		if err != nil {
			return nil, err
		}

		if err := claimCoursewareReviewImpactItemRelationOperation(
			value,
			prelockedCancelRelations,
			itemIDs,
			semanticClaims,
		); err != nil {
			return nil, err
		}

		prepared = append(prepared, value)
	}

	lockedItems := make(
		map[string]*models.CoursewareReviewItem,
		len(itemIDs),
	)

	for _, itemID := range sortedCoursewareReviewImpactKeys(itemIDs) {
		item, err := lockCoursewareReviewImpactItemTx(
			ctx,
			tx,
			itemID,
		)
		if err != nil {
			return nil, err
		}

		lockedItems[itemID] = item
	}

	for index := range prepared {
		if err := validatePreparedCoursewareReviewImpactItemRelationOperationTx(
			ctx,
			tx,
			plan,
			&prepared[index],
			actorID,
			lockedItems,
			prelockedCancelRelations,
		); err != nil {
			return nil, err
		}
	}

	return prepared, nil
}

func prepareCoursewareReviewImpactItemRelationOperation(
	operation models.CoursewareReviewImpactOperation,
) (
	cwReviewImpactPreparedItemRelationOperation,
	error,
) {
	prepared := cwReviewImpactPreparedItemRelationOperation{
		OperationID:   operation.OperationID,
		OperationType: operation.OperationType,
	}

	switch operation.OperationType {
	case models.CWReviewImpactOperationCreateRelation:
		value := &cwReviewImpactPreparedCreateRelation{}

		if err := decodeCoursewareReviewImpactMap(
			operation.Payload,
			&value.Payload,
		); err != nil {
			return prepared, err
		}

		if err := decodeCoursewareReviewImpactMap(
			operation.Preconditions,
			&value.Preconditions,
		); err != nil {
			return prepared, err
		}

		normalizePreparedImpactCreateRelation(value)
		prepared.CreateRelation = value

	case models.CWReviewImpactOperationCancelRelation:
		value := &cwReviewImpactPreparedCancelRelation{}

		if err := decodeCoursewareReviewImpactMap(
			operation.Payload,
			&value.Payload,
		); err != nil {
			return prepared, err
		}

		if err := decodeCoursewareReviewImpactMap(
			operation.Preconditions,
			&value.Preconditions,
		); err != nil {
			return prepared, err
		}

		value.Payload.RelationID = strings.TrimSpace(value.Payload.RelationID)
		value.Payload.Reason = strings.TrimSpace(value.Payload.Reason)
		value.Preconditions.Relation.RelationID =
			strings.TrimSpace(value.Preconditions.Relation.RelationID)
		value.Preconditions.Relation.Status =
			strings.TrimSpace(value.Preconditions.Relation.Status)

		prepared.CancelRelation = value

	case models.CWReviewImpactOperationCreateItem:
		value := &cwReviewImpactPreparedCreateItem{}

		if err := decodeCoursewareReviewImpactMap(
			operation.Payload,
			&value.Payload,
		); err != nil {
			return prepared, err
		}

		if err := decodeCoursewareReviewImpactMap(
			operation.Preconditions,
			&value.Preconditions,
		); err != nil {
			return prepared, err
		}

		normalizePreparedImpactCreateItem(value)
		prepared.CreateItem = value

	case models.CWReviewImpactOperationDismissItem:
		value := &cwReviewImpactPreparedDismissItem{}

		if err := decodeCoursewareReviewImpactMap(
			operation.Payload,
			&value.Payload,
		); err != nil {
			return prepared, err
		}

		if err := decodeCoursewareReviewImpactMap(
			operation.Preconditions,
			&value.Preconditions,
		); err != nil {
			return prepared, err
		}

		value.Payload.ItemID = strings.TrimSpace(value.Payload.ItemID)
		value.Payload.Reason = strings.TrimSpace(value.Payload.Reason)
		normalizePreparedImpactItemPrecondition(&value.Preconditions.Item)

		prepared.DismissItem = value

	case models.CWReviewImpactOperationUpdateCandidateSuggestion:
		value := &cwReviewImpactPreparedUpdateCandidate{}

		if err := decodeCoursewareReviewImpactMap(
			operation.Payload,
			&value.Payload,
		); err != nil {
			return prepared, err
		}

		if err := decodeCoursewareReviewImpactMap(
			operation.Preconditions,
			&value.Preconditions,
		); err != nil {
			return prepared, err
		}

		value.Payload.ItemID = strings.TrimSpace(value.Payload.ItemID)
		value.Payload.CandidateInstruction =
			strings.TrimSpace(value.Payload.CandidateInstruction)
		normalizePreparedImpactItemPrecondition(&value.Preconditions.Item)

		prepared.UpdateCandidate = value

	default:
		return prepared, ErrCoursewareReviewImpactOperationUnsupported
	}

	return prepared, nil
}

func claimCoursewareReviewImpactItemRelationOperation(
	operation cwReviewImpactPreparedItemRelationOperation,
	prelockedCancelRelations map[string]*models.CoursewareReviewItemRelation,
	itemIDs map[string]struct{},
	semanticClaims map[string]struct{},
) error {
	claim := func(key string) error {
		key = strings.TrimSpace(key)
		if key == "" {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if _, exists := semanticClaims[key]; exists {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		semanticClaims[key] = struct{}{}
		return nil
	}

	switch operation.OperationType {
	case models.CWReviewImpactOperationCreateRelation:
		value := operation.CreateRelation
		if value == nil {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if err := claim(
			"create_relation|" +
				value.Payload.RelationType + "|" +
				value.Payload.SourceItemID + "|" +
				value.Payload.TargetItemID,
		); err != nil {
			return err
		}

		itemIDs[value.Preconditions.SourceItem.ItemID] = struct{}{}
		itemIDs[value.Preconditions.TargetItem.ItemID] = struct{}{}

	case models.CWReviewImpactOperationCancelRelation:
		value := operation.CancelRelation
		if value == nil {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if err := claim("cancel_relation|" + value.Payload.RelationID); err != nil {
			return err
		}

		// Go索引表达式保持在单行，避免换行触发自动分号插入。
		relation := prelockedCancelRelations[value.Payload.RelationID]
		if relation == nil {
			return ErrCoursewareReviewImpactPlanConflict
		}

		itemIDs[relation.SourceItemID] = struct{}{}
		itemIDs[relation.TargetItemID] = struct{}{}

	case models.CWReviewImpactOperationDismissItem:
		value := operation.DismissItem
		if value == nil {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if err := claim("dismiss_item|" + value.Payload.ItemID); err != nil {
			return err
		}

		itemIDs[value.Preconditions.Item.ItemID] = struct{}{}

	case models.CWReviewImpactOperationUpdateCandidateSuggestion:
		value := operation.UpdateCandidate
		if value == nil {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if err := claim(
			"update_candidate_suggestion|" + value.Payload.ItemID,
		); err != nil {
			return err
		}

		itemIDs[value.Preconditions.Item.ItemID] = struct{}{}

	case models.CWReviewImpactOperationCreateItem:
		// create_item没有现有item_id，不参与现有整改项锁集合。
		return nil
	}

	return nil
}
