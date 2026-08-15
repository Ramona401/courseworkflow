package repository

// courseware_review_impact_apply_item_validate.go
//
// R-07 Atomic Apply中整改项新增、忽略和候选建议更新的事务前置验证。
//
// create_item重新读取会话、课件作者和当前页面事实。
// dismiss/update candidate重新核对冻结整改项指纹。
// 本文件不执行业务写入。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/models"
)

func validatePreparedImpactCreateItemTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	value *cwReviewImpactPreparedCreateItem,
	actorID string,
) error {
	if value == nil ||
		value.Payload.Title == "" ||
		value.Payload.Description == "" ||
		!models.IsCWReviewSeverity(value.Payload.Severity) {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	var (
		reviewLevel int
		ownerID     string
	)

	err := tx.QueryRow(
		ctx,
		`SELECT
			session.review_level,
			courseware.user_id
		 FROM courseware_ai_review_sessions AS session
		 INNER JOIN coursewares AS courseware
			ON courseware.id = session.courseware_id
		 WHERE session.id = $1
		   AND session.courseware_id = $2
		   AND session.reviewer_id = $3
		   AND session.status = 'done'`,
		plan.SourceSessionID,
		plan.CoursewareID,
		strings.TrimSpace(actorID),
	).Scan(
		&reviewLevel,
		&ownerID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCoursewareReviewImpactPlanConflict
		}

		return fmt.Errorf(
			"重新读取影响方案新增问题会话事实失败: %w",
			err,
		)
	}

	sourceType := models.CWReviewItemSourceFormal
	if reviewLevel == models.CWAIReviewLevelSelf {
		sourceType = models.CWReviewItemSourceSelf

		if ownerID != strings.TrimSpace(actorID) {
			return ErrCoursewareReviewImpactPlanConflict
		}
	}

	pageUpdatedAt, err := validateCoursewareReviewImpactCreateItemPageTx(
		ctx,
		tx,
		plan.CoursewareID,
		value.Payload.PageID,
		value.Preconditions.Page,
	)
	if err != nil {
		return err
	}

	value.ReviewLevel = reviewLevel
	value.SourceType = sourceType
	value.OwnerID = ownerID
	value.PageUpdatedAt = pageUpdatedAt

	return nil
}

func validatePreparedImpactItemMutation(
	plan *models.CoursewareReviewImpactPlan,
	payloadItemID string,
	precondition cwReviewImpactItemPrecondition,
	actorID string,
	items map[string]*models.CoursewareReviewItem,
) error {
	payloadItemID = strings.TrimSpace(payloadItemID)

	if payloadItemID == "" ||
		payloadItemID != strings.TrimSpace(precondition.ItemID) {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	return validateCoursewareReviewImpactItemPrecondition(
		plan,
		items[payloadItemID],
		precondition,
		actorID,
	)
}

func normalizePreparedImpactCreateItem(
	value *cwReviewImpactPreparedCreateItem,
) {
	value.Payload.PageID =
		strings.TrimSpace(value.Payload.PageID)
	value.Payload.Severity =
		strings.TrimSpace(value.Payload.Severity)
	value.Payload.Dimension =
		strings.TrimSpace(value.Payload.Dimension)
	value.Payload.Title =
		strings.TrimSpace(value.Payload.Title)
	value.Payload.Description =
		strings.TrimSpace(value.Payload.Description)
	value.Payload.CandidateInstruction =
		strings.TrimSpace(value.Payload.CandidateInstruction)

	value.Preconditions.Page.Scope =
		strings.TrimSpace(value.Preconditions.Page.Scope)
	value.Preconditions.Page.PageID =
		strings.TrimSpace(value.Preconditions.Page.PageID)
	value.Preconditions.Page.Title =
		strings.TrimSpace(value.Preconditions.Page.Title)
	value.Preconditions.Page.HTMLHash =
		strings.TrimSpace(value.Preconditions.Page.HTMLHash)
}

func normalizePreparedImpactItemPrecondition(
	value *cwReviewImpactItemPrecondition,
) {
	value.ItemID = strings.TrimSpace(value.ItemID)
	value.Status = strings.TrimSpace(value.Status)
	value.Fingerprint = strings.TrimSpace(value.Fingerprint)
}
