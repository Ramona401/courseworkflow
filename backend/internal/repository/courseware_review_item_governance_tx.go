package repository

// courseware_review_item_governance_tx.go
//
// 整改项关系治理仓储的事务内部辅助模块。
//
// 本文件负责：
//   1. 规范关系输入；
//   2. 按稳定顺序锁定关系两端整改项；
//   3. 校验未交付、会话归属、来源类型和操作者边界；
//   4. 查询、创建、重新启用和取消关系；
//   5. 为每个关系版本追加不可变事件。
//
// 对外仓储入口和只读查询位于courseware_review_item_governance_repo.go。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/models"
)

func normalizeCWReviewItemRelation(
	relation *models.CoursewareReviewItemRelation,
) {
	relation.CoursewareID =
		strings.TrimSpace(relation.CoursewareID)
	relation.SourceSessionID =
		strings.TrimSpace(relation.SourceSessionID)
	relation.SourceItemID =
		strings.TrimSpace(relation.SourceItemID)
	relation.TargetItemID =
		strings.TrimSpace(relation.TargetItemID)
	relation.RelationType =
		strings.TrimSpace(relation.RelationType)
	relation.Explanation =
		strings.TrimSpace(relation.Explanation)

	if relation.SourceGlobalMessageID != nil {
		value := strings.TrimSpace(
			*relation.SourceGlobalMessageID,
		)
		if value == "" {
			relation.SourceGlobalMessageID = nil
		} else {
			relation.SourceGlobalMessageID = &value
		}
	}
}

func lockGovernableCWReviewItemPairTx(
	ctx context.Context,
	tx pgx.Tx,
	firstItemID string,
	secondItemID string,
	coursewareID string,
	sessionID string,
	actorID string,
	requireActionable bool,
) error {
	itemIDs := []string{
		strings.TrimSpace(firstItemID),
		strings.TrimSpace(secondItemID),
	}
	sort.Strings(itemIDs)

	for _, itemID := range itemIDs {
		if err := lockGovernableCWReviewItemTx(
			ctx,
			tx,
			itemID,
			coursewareID,
			sessionID,
			actorID,
			requireActionable,
		); err != nil {
			return err
		}
	}

	return nil
}

func lockGovernableCWReviewItemTx(
	ctx context.Context,
	tx pgx.Tx,
	itemID string,
	coursewareID string,
	sessionID string,
	actorID string,
	requireActionable bool,
) error {
	locked := &cwReviewItemGovernanceItemLock{}

	err := tx.QueryRow(
		ctx,
		`
                SELECT
                        courseware_id,
                        source_session_id,
                        source_type,
                        created_by,
                        owner_id,
                        status,
                        (
                                courseware_review_id IS NOT NULL
                                OR feedback_id IS NOT NULL
                        )
                FROM courseware_review_items
                WHERE id = $1
                FOR UPDATE`,
		strings.TrimSpace(itemID),
	).Scan(
		&locked.CoursewareID,
		&locked.SourceSessionID,
		&locked.SourceType,
		&locked.CreatedBy,
		&locked.OwnerID,
		&locked.Status,
		&locked.AlreadyDelivered,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCoursewareReviewItemNotFound
		}
		return fmt.Errorf(
			"锁定关系整改项失败: %w",
			err,
		)
	}

	if locked.CoursewareID !=
		strings.TrimSpace(coursewareID) ||
		locked.SourceSessionID !=
			strings.TrimSpace(sessionID) ||
		locked.AlreadyDelivered {
		return ErrCoursewareReviewItemConflict
	}

	switch locked.SourceType {
	case models.CWReviewItemSourceFormal:
		if locked.CreatedBy !=
			strings.TrimSpace(actorID) {
			return ErrCoursewareReviewItemConflict
		}

	case models.CWReviewItemSourceSelf:
		if locked.OwnerID !=
			strings.TrimSpace(actorID) {
			return ErrCoursewareReviewItemConflict
		}

	default:
		return ErrCoursewareReviewItemConflict
	}

	if requireActionable {
		switch locked.Status {
		case models.CWReviewItemStatusDetected,
			models.CWReviewItemStatusDiscussing,
			models.CWReviewItemStatusConfirmed:
		default:
			return ErrCoursewareReviewItemConflict
		}
	}

	return nil
}

func getCoursewareReviewItemRelationByKeyTx(
	ctx context.Context,
	tx pgx.Tx,
	relation *models.CoursewareReviewItemRelation,
	actorID string,
) (*models.CoursewareReviewItemRelation, error) {
	current, err :=
		scanCoursewareReviewItemRelation(
			tx.QueryRow(
				ctx,
				`SELECT `+
					cwReviewItemRelationSelectColumns+
					`
                         FROM courseware_review_item_relations
                         WHERE courseware_id = $1
                           AND source_session_id = $2
                           AND relation_type = $3
                           AND source_item_id = $4
                           AND target_item_id = $5
                           AND created_by = $6
                         FOR UPDATE`,
				relation.CoursewareID,
				relation.SourceSessionID,
				relation.RelationType,
				relation.SourceItemID,
				relation.TargetItemID,
				strings.TrimSpace(actorID),
			),
		)
	if err != nil &&
		!errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf(
			"查询现有整改项关系失败: %w",
			err,
		)
	}

	return current, err
}

func insertCoursewareReviewItemRelationTx(
	ctx context.Context,
	tx pgx.Tx,
	relation *models.CoursewareReviewItemRelation,
) (*models.CoursewareReviewItemRelation, error) {
	current, err :=
		scanCoursewareReviewItemRelation(
			tx.QueryRow(
				ctx,
				`INSERT INTO courseware_review_item_relations (
                                        courseware_id,
                                        source_session_id,
                                        source_item_id,
                                        target_item_id,
                                        relation_type,
                                        status,
                                        version,
                                        explanation,
                                        source_global_message_id,
                                        created_by,
                                        confirmed_at,
                                        created_at,
                                        updated_at
                                )
                                VALUES (
                                        $1, $2, $3, $4, $5,
                                        'active', 1, $6, $7, $8,
                                        NOW(), NOW(), NOW()
                                )
                                RETURNING `+
					cwReviewItemRelationSelectColumns,
				relation.CoursewareID,
				relation.SourceSessionID,
				relation.SourceItemID,
				relation.TargetItemID,
				relation.RelationType,
				relation.Explanation,
				relation.SourceGlobalMessageID,
				relation.CreatedBy,
			),
		)
	if err != nil {
		return nil, fmt.Errorf(
			"创建整改项关系失败: %w",
			err,
		)
	}

	return current, nil
}

func reactivateCoursewareReviewItemRelationTx(
	ctx context.Context,
	tx pgx.Tx,
	current *models.CoursewareReviewItemRelation,
	explanation string,
) (*models.CoursewareReviewItemRelation, error) {
	next, err :=
		scanCoursewareReviewItemRelation(
			tx.QueryRow(
				ctx,
				`UPDATE courseware_review_item_relations
                                 SET
                                        status = 'active',
                                        version = version + 1,
                                        explanation = $3,
                                        cancelled_by = NULL,
                                        cancelled_at = NULL,
                                        updated_at = NOW()
                                 WHERE id = $1
                                   AND status = 'cancelled'
                                   AND version = $2
                                 RETURNING `+
					cwReviewItemRelationSelectColumns,
				current.ID,
				current.Version,
				strings.TrimSpace(explanation),
			),
		)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewItemConflict
		}
		return nil, fmt.Errorf(
			"重新启用整改项关系失败: %w",
			err,
		)
	}

	return next, nil
}

func cancelCoursewareReviewItemRelationTx(
	ctx context.Context,
	tx pgx.Tx,
	current *models.CoursewareReviewItemRelation,
	actorID string,
) (*models.CoursewareReviewItemRelation, error) {
	next, err :=
		scanCoursewareReviewItemRelation(
			tx.QueryRow(
				ctx,
				`UPDATE courseware_review_item_relations
                                 SET
                                        status = 'cancelled',
                                        version = version + 1,
                                        cancelled_by = $3,
                                        cancelled_at = NOW(),
                                        updated_at = NOW()
                                 WHERE id = $1
                                   AND status = 'active'
                                   AND version = $2
                                 RETURNING `+
					cwReviewItemRelationSelectColumns,
				current.ID,
				current.Version,
				strings.TrimSpace(actorID),
			),
		)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewItemConflict
		}
		return nil, fmt.Errorf(
			"取消整改项关系失败: %w",
			err,
		)
	}

	return next, nil
}

func insertCoursewareReviewItemRelationEventTx(
	ctx context.Context,
	tx pgx.Tx,
	relation *models.CoursewareReviewItemRelation,
	eventType string,
	actorID string,
	reason string,
	sourceGlobalMessageID *string,
) error {
	metaJSON, err := json.Marshal(
		cwReviewItemRelationEventMeta{
			RelationType: relation.RelationType,
			SourceItemID: relation.SourceItemID,
			TargetItemID: relation.TargetItemID,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"序列化整改项关系事件失败: %w",
			err,
		)
	}

	_, err = tx.Exec(
		ctx,
		`
                INSERT INTO courseware_review_item_relation_events (
                        relation_id,
                        source_session_id,
                        relation_version,
                        event_type,
                        actor_id,
                        reason,
                        source_global_message_id,
                        metadata_json,
                        created_at
                )
                VALUES (
                        $1, $2, $3, $4, $5, $6, $7,
                        $8::jsonb,
                        clock_timestamp()
                )`,
		relation.ID,
		relation.SourceSessionID,
		relation.Version,
		strings.TrimSpace(eventType),
		strings.TrimSpace(actorID),
		strings.TrimSpace(reason),
		sourceGlobalMessageID,
		string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf(
			"记录整改项关系事件失败: %w",
			err,
		)
	}

	return nil
}
