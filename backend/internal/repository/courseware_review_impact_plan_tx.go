package repository

// courseware_review_impact_plan_tx.go
//
// R-07结构化影响方案的事务内部辅助。
//
// 本文件只负责plan自身的锁定、CAS状态迁移和追加式事件。
// 九类业务操作的真正原子执行将在独立Apply Repository中调用这些辅助函数，
// 并与问题组、关系、整改项和消息写入共享同一个pgx.Tx。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/models"
)

func lockCoursewareReviewImpactPlanTx(
	ctx context.Context,
	tx pgx.Tx,
	planID string,
	sessionID string,
	actorID string,
) (*models.CoursewareReviewImpactPlan, error) {
	plan, err := scanCoursewareReviewImpactPlan(
		tx.QueryRow(
			ctx,
			`SELECT `+cwReviewImpactPlanSelectColumns+`
			 FROM courseware_review_impact_plans
			 WHERE id = $1
			   AND source_session_id = $2
			   AND created_by = $3
			 FOR UPDATE`,
			strings.TrimSpace(planID),
			strings.TrimSpace(sessionID),
			strings.TrimSpace(actorID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewImpactPlanNotFound
		}

		return nil, fmt.Errorf(
			"锁定课件审核影响方案失败: %w",
			err,
		)
	}

	return plan, nil
}

func ensureCoursewareReviewImpactPlanDraftVersion(
	plan *models.CoursewareReviewImpactPlan,
	expectedVersion int,
) error {
	if plan == nil ||
		plan.Status != models.CWReviewImpactPlanStatusDraft ||
		plan.Version != 1 ||
		expectedVersion != 1 {
		return ErrCoursewareReviewImpactPlanConflict
	}

	return nil
}

func markCoursewareReviewImpactPlanAppliedTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	expectedVersion int,
	selectedOperationIDsJSON string,
	actorID string,
) (*models.CoursewareReviewImpactPlan, error) {
	if err := ensureCoursewareReviewImpactPlanDraftVersion(
		plan,
		expectedVersion,
	); err != nil {
		return nil, err
	}

	next, err := scanCoursewareReviewImpactPlan(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_review_impact_plans
			 SET
				status = 'applied',
				version = 2,
				applied_operation_ids_json = $4::jsonb,
				applied_by = $5,
				applied_at = clock_timestamp(),
				updated_at = clock_timestamp()
			 WHERE id = $1
			   AND source_session_id = $2
			   AND created_by = $3
			   AND status = 'draft'
			   AND version = $6
			 RETURNING `+cwReviewImpactPlanSelectColumns,
			plan.ID,
			plan.SourceSessionID,
			plan.CreatedBy,
			strings.TrimSpace(selectedOperationIDsJSON),
			strings.TrimSpace(actorID),
			expectedVersion,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewImpactPlanConflict
		}

		return nil, fmt.Errorf(
			"应用课件审核影响方案状态失败: %w",
			err,
		)
	}

	return next, nil
}

func insertCoursewareReviewImpactPlanEventTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	eventType string,
	actorID string,
	selectedOperationIDsJSON string,
	metadataJSON string,
) error {
	if plan == nil {
		return ErrCoursewareReviewImpactPlanNotFound
	}

	selectedOperationIDsJSON = strings.TrimSpace(
		selectedOperationIDsJSON,
	)
	if selectedOperationIDsJSON == "" {
		selectedOperationIDsJSON = "[]"
	}

	metadataJSON = strings.TrimSpace(metadataJSON)
	if metadataJSON == "" {
		metadataJSON = "{}"
	}

	_, err := tx.Exec(
		ctx,
		`INSERT INTO courseware_review_impact_plan_events (
			plan_id,
			source_session_id,
			plan_version,
			event_type,
			actor_id,
			selected_operation_ids_json,
			metadata_json,
			created_at
		 )
		 VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6::jsonb,
			$7::jsonb,
			clock_timestamp()
		 )`,
		plan.ID,
		plan.SourceSessionID,
		plan.Version,
		strings.TrimSpace(eventType),
		strings.TrimSpace(actorID),
		selectedOperationIDsJSON,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf(
			"记录课件审核影响方案事件失败: %w",
			err,
		)
	}

	return nil
}
