package repository

// courseware_review_impact_plan_repo.go
//
// R-07结构化影响方案的公开查询和草稿创建仓储。
//
// 设计边界：
//   1. 草稿创建时operations_json一次冻结，后续不可修改；
//   2. source_message_hash和operations_hash由数据库可信守卫计算；
//   3. draft_created事件与plan INSERT位于同一事务；
//   4. 查询始终同时限定source_session_id和created_by；
//   5. 最终原子Apply的锁定与状态迁移辅助位于courseware_review_impact_plan_tx.go。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrCoursewareReviewImpactPlanNotFound = errors.New(
		"课件审核影响方案不存在",
	)

	ErrCoursewareReviewImpactPlanConflict = errors.New(
		"课件审核影响方案状态已变化，请刷新后重试",
	)
)

const cwReviewImpactPlanSelectColumns = `
	id,
	courseware_id,
	source_session_id,
	source_message_id,
	status,
	version,
	operations_schema_version,
	operations_json::text,
	operations_hash,
	source_message_hash,
	created_by,
	created_at,
	applied_operation_ids_json::text,
	COALESCE(applied_by::text, ''),
	applied_at,
	updated_at`

func scanCoursewareReviewImpactPlan(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareReviewImpactPlan, error) {
	plan := &models.CoursewareReviewImpactPlan{}
	var appliedBy string

	err := row.Scan(
		&plan.ID,
		&plan.CoursewareID,
		&plan.SourceSessionID,
		&plan.SourceMessageID,
		&plan.Status,
		&plan.Version,
		&plan.OperationsSchemaVersion,
		&plan.OperationsJSON,
		&plan.OperationsHash,
		&plan.SourceMessageHash,
		&plan.CreatedBy,
		&plan.CreatedAt,
		&plan.AppliedOperationIDsJSON,
		&appliedBy,
		&plan.AppliedAt,
		&plan.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if appliedBy != "" {
		plan.AppliedBy = &appliedBy
	}

	return plan, nil
}

// CreateCoursewareReviewImpactPlanDraft 原子创建不可变影响方案草稿和version=1事件。
func CreateCoursewareReviewImpactPlanDraft(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	sourceMessageID string,
	actorID string,
	operationsJSON string,
) (*models.CoursewareReviewImpactPlan, error) {
	coursewareID = strings.TrimSpace(coursewareID)
	sessionID = strings.TrimSpace(sessionID)
	sourceMessageID = strings.TrimSpace(sourceMessageID)
	actorID = strings.TrimSpace(actorID)
	operationsJSON = strings.TrimSpace(operationsJSON)

	if coursewareID == "" ||
		sessionID == "" ||
		sourceMessageID == "" ||
		actorID == "" ||
		operationsJSON == "" ||
		!json.Valid([]byte(operationsJSON)) {
		return nil, ErrCoursewareReviewImpactPlanConflict
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始创建课件审核影响方案事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	plan, err := scanCoursewareReviewImpactPlan(
		tx.QueryRow(
			ctx,
			`INSERT INTO courseware_review_impact_plans (
				courseware_id,
				source_session_id,
				source_message_id,
				status,
				version,
				operations_schema_version,
				operations_json,
				created_by,
				applied_operation_ids_json,
				created_at,
				updated_at
			 )
			 VALUES (
				$1,
				$2,
				$3,
				'draft',
				1,
				1,
				$4::jsonb,
				$5,
				'[]'::jsonb,
				NOW(),
				NOW()
			 )
			 RETURNING `+cwReviewImpactPlanSelectColumns,
			coursewareID,
			sessionID,
			sourceMessageID,
			operationsJSON,
			actorID,
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"创建课件审核影响方案草稿失败: %w",
			err,
		)
	}

	metadataJSON := fmt.Sprintf(
		`{"operation_count":%d}`,
		countCWReviewImpactOperations(operationsJSON),
	)

	if err := insertCoursewareReviewImpactPlanEventTx(
		ctx,
		tx,
		plan,
		models.CWReviewImpactPlanEventDraftCreated,
		actorID,
		"[]",
		metadataJSON,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交创建课件审核影响方案事务失败: %w",
			err,
		)
	}

	return plan, nil
}

// GetCoursewareReviewImpactPlanByID 读取当前审核者的一条影响方案。
func GetCoursewareReviewImpactPlanByID(
	ctx context.Context,
	planID string,
	sessionID string,
	actorID string,
) (*models.CoursewareReviewImpactPlan, error) {
	plan, err := scanCoursewareReviewImpactPlan(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwReviewImpactPlanSelectColumns+`
			 FROM courseware_review_impact_plans
			 WHERE id = $1
			   AND source_session_id = $2
			   AND created_by = $3`,
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
			"查询课件审核影响方案失败: %w",
			err,
		)
	}

	return plan, nil
}

// ListCoursewareReviewImpactPlansBySession 返回当前审核者会话中的影响方案。
func ListCoursewareReviewImpactPlansBySession(
	ctx context.Context,
	sessionID string,
	actorID string,
) ([]*models.CoursewareReviewImpactPlan, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT `+cwReviewImpactPlanSelectColumns+`
		 FROM courseware_review_impact_plans
		 WHERE source_session_id = $1
		   AND created_by = $2
		 ORDER BY created_at DESC, id DESC`,
		strings.TrimSpace(sessionID),
		strings.TrimSpace(actorID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课件审核影响方案列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	plans := make([]*models.CoursewareReviewImpactPlan, 0)

	for rows.Next() {
		plan, scanErr := scanCoursewareReviewImpactPlan(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描课件审核影响方案失败: %w",
				scanErr,
			)
		}

		plans = append(plans, plan)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件审核影响方案失败: %w",
			err,
		)
	}

	return plans, nil
}

// ListCoursewareReviewImpactPlanEvents 返回影响方案不可变事件历史。
func ListCoursewareReviewImpactPlanEvents(
	ctx context.Context,
	planID string,
	sessionID string,
	actorID string,
) ([]*models.CoursewareReviewImpactPlanEvent, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT
			event.id,
			event.plan_id,
			event.source_session_id,
			event.plan_version,
			event.event_type,
			event.actor_id,
			event.selected_operation_ids_json::text,
			event.metadata_json::text,
			event.created_at
		 FROM courseware_review_impact_plan_events AS event
		 INNER JOIN courseware_review_impact_plans AS plan
			ON plan.id = event.plan_id
			AND plan.source_session_id = event.source_session_id
		 WHERE event.plan_id = $1
		   AND event.source_session_id = $2
		   AND plan.created_by = $3
		 ORDER BY event.plan_version ASC`,
		strings.TrimSpace(planID),
		strings.TrimSpace(sessionID),
		strings.TrimSpace(actorID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课件审核影响方案事件失败: %w",
			err,
		)
	}
	defer rows.Close()

	events := make([]*models.CoursewareReviewImpactPlanEvent, 0)

	for rows.Next() {
		event := &models.CoursewareReviewImpactPlanEvent{}

		if err := rows.Scan(
			&event.ID,
			&event.PlanID,
			&event.SourceSessionID,
			&event.PlanVersion,
			&event.EventType,
			&event.ActorID,
			&event.SelectedOperationIDsJSON,
			&event.MetadataJSON,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描课件审核影响方案事件失败: %w",
				err,
			)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件审核影响方案事件失败: %w",
			err,
		)
	}

	return events, nil
}

func countCWReviewImpactOperations(operationsJSON string) int {
	var values []json.RawMessage
	if err := json.Unmarshal(
		[]byte(operationsJSON),
		&values,
	); err != nil {
		return 0
	}

	return len(values)
}
