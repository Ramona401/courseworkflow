package repository

// courseware_review_item_governance_repo.go
//
// 全局讨论结论落地与整改项关系治理的数据访问层。
//
// 数据库已经通过复合外键、不可变守卫、关系版本和延迟约束保证：
//   1. 人工新增整改项只能绑定同会话全局assistant消息；
//   2. 关系两端必须属于同一课件和同一AI审核会话；
//   3. 关系创建、取消或重新启用必须与追加式事件位于同一事务；
//   4. 关系端点、类型、首次来源和首次确认时间创建后不可修改；
//   5. 已正式交付的整改项不能再参与新的治理动作；
//   6. 关系历史读取必须同时匹配关系ID、会话和关系创建者。
//
// Service仍负责会话所有权、教育域、当前页面快照以及可信AI元数据校验。
// 私有事务锁定与状态变更辅助位于courseware_review_item_governance_tx.go。

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
	// ErrCoursewareReviewItemRelationNotFound 合并关系不存在与无访问边界两种情况。
	ErrCoursewareReviewItemRelationNotFound = errors.New(
		"课件审核整改项关系不存在",
	)
)

const cwReviewItemRelationSelectColumns = `
	id,
	courseware_id,
	source_session_id,
	source_item_id,
	target_item_id,
	relation_type,
	status,
	version,
	explanation,
	COALESCE(source_global_message_id::text, ''),
	created_by,
	confirmed_at,
	COALESCE(cancelled_by::text, ''),
	cancelled_at,
	created_at,
	updated_at`

type cwReviewItemGovernanceItemLock struct {
	CoursewareID     string
	SourceSessionID  string
	SourceType       string
	CreatedBy        string
	OwnerID          string
	Status           string
	AlreadyDelivered bool
}

type cwReviewItemManualCreationMeta struct {
	Event                 string  `json:"event"`
	OriginType            string  `json:"origin_type"`
	SourceGlobalMessageID string  `json:"source_global_message_id"`
	PageID                *string `json:"page_id,omitempty"`
	PageNumberSnapshot    int     `json:"page_number_snapshot"`
}

type cwReviewItemRelationEventMeta struct {
	RelationType string `json:"relation_type"`
	SourceItemID string `json:"source_item_id"`
	TargetItemID string `json:"target_item_id"`
}

func scanCoursewareReviewItemRelation(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareReviewItemRelation, error) {
	relation := &models.CoursewareReviewItemRelation{}
	var sourceGlobalMessageID, cancelledBy string

	err := row.Scan(
		&relation.ID,
		&relation.CoursewareID,
		&relation.SourceSessionID,
		&relation.SourceItemID,
		&relation.TargetItemID,
		&relation.RelationType,
		&relation.Status,
		&relation.Version,
		&relation.Explanation,
		&sourceGlobalMessageID,
		&relation.CreatedBy,
		&relation.ConfirmedAt,
		&cancelledBy,
		&relation.CancelledAt,
		&relation.CreatedAt,
		&relation.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if sourceGlobalMessageID != "" {
		relation.SourceGlobalMessageID = &sourceGlobalMessageID
	}
	if cancelledBy != "" {
		relation.CancelledBy = &cancelledBy
	}

	return relation, nil
}

// CreateManualCoursewareReviewItemsFromGlobalDiscussion 在一个事务中创建
// 一组由同一人工问题拆分出的页级整改项，并写入每条问题的系统操作记录。
func CreateManualCoursewareReviewItemsFromGlobalDiscussion(
	ctx context.Context,
	items []*models.CoursewareReviewItem,
	actorID string,
) error {
	actorID = strings.TrimSpace(actorID)
	if len(items) == 0 || actorID == "" {
		return errors.New(
			"全局讨论人工新增整改项参数无效",
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始人工新增整改项事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for _, item := range items {
		if item == nil ||
			item.SourceGlobalMessageID == nil ||
			strings.TrimSpace(
				*item.SourceGlobalMessageID,
			) == "" {
			return errors.New(
				"人工新增整改项缺少可信全局讨论来源消息",
			)
		}

		if err := CreateCoursewareReviewItemTx(
			ctx,
			tx,
			item,
		); err != nil {
			return err
		}

		metaJSON, marshalErr := json.Marshal(
			cwReviewItemManualCreationMeta{
				Event:      "global_discussion_manual_item_created",
				OriginType: item.OriginType,
				SourceGlobalMessageID: strings.TrimSpace(
					*item.SourceGlobalMessageID,
				),
				PageID:             item.PageID,
				PageNumberSnapshot: item.PageNumberSnapshot,
			},
		)
		if marshalErr != nil {
			return fmt.Errorf(
				"序列化人工新增整改项事件失败: %w",
				marshalErr,
			)
		}

		_, err = tx.Exec(
			ctx,
			`
			INSERT INTO courseware_ai_review_messages (
				session_id,
				review_item_id,
				user_id,
				role,
				content,
				citations_json,
				tokens_used,
				model_used,
				created_at
			)
			VALUES (
				$1,
				$2,
				$3,
				'system',
				$4,
				$5::jsonb,
				0,
				'',
				NOW()
			)`,
			item.SourceSessionID,
			item.ID,
			actorID,
			"已从全局讨论人工新增此整改项。"+
				"候选修改指令尚未独立确认，"+
				"页面和审核决定均未改变。",
			string(metaJSON),
		)
		if err != nil {
			return fmt.Errorf(
				"记录人工新增整改项系统事件失败: %w",
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交人工新增整改项事务失败: %w",
			err,
		)
	}

	return nil
}

// ConfirmCoursewareReviewItemRelation 明确确认或重新启用一条整改项关系。
//
// 对同一业务关系重复确认是幂等的；已取消关系会递增version，
// 并将本次确认使用的关系说明写入reactivated事件。
func ConfirmCoursewareReviewItemRelation(
	ctx context.Context,
	relation *models.CoursewareReviewItemRelation,
	actorID string,
) (*models.CoursewareReviewItemRelation, error) {
	if relation == nil {
		return nil, errors.New(
			"课件审核整改项关系不能为空",
		)
	}

	actorID = strings.TrimSpace(actorID)
	normalizeCWReviewItemRelation(relation)

	if actorID == "" ||
		relation.CoursewareID == "" ||
		relation.SourceSessionID == "" ||
		relation.SourceItemID == "" ||
		relation.TargetItemID == "" ||
		relation.SourceItemID == relation.TargetItemID ||
		!models.IsCWReviewItemRelationType(
			relation.RelationType,
		) ||
		relation.Explanation == "" {
		return nil, errors.New(
			"课件审核整改项关系参数无效",
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始确认整改项关系事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockGovernableCWReviewItemPairTx(
		ctx,
		tx,
		relation.SourceItemID,
		relation.TargetItemID,
		relation.CoursewareID,
		relation.SourceSessionID,
		actorID,
		true,
	); err != nil {
		return nil, err
	}

	lockKey := strings.Join(
		[]string{
			relation.CoursewareID,
			relation.SourceSessionID,
			relation.RelationType,
			relation.SourceItemID,
			relation.TargetItemID,
		},
		"|",
	)
	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(
			hashtextextended($1, 0)
		)`,
		lockKey,
	); err != nil {
		return nil, fmt.Errorf(
			"锁定整改项关系业务键失败: %w",
			err,
		)
	}

	current, err :=
		getCoursewareReviewItemRelationByKeyTx(
			ctx,
			tx,
			relation,
			actorID,
		)
	if err != nil &&
		!errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	eventType :=
		models.CWReviewItemRelationEventConfirmed
	eventReason := relation.Explanation

	if errors.Is(err, pgx.ErrNoRows) {
		relation.Status =
			models.CWReviewItemRelationStatusActive
		relation.Version = 1
		relation.CreatedBy = actorID

		current, err =
			insertCoursewareReviewItemRelationTx(
				ctx,
				tx,
				relation,
			)
		if err != nil {
			return nil, err
		}
	} else if current.Status ==
		models.CWReviewItemRelationStatusActive {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf(
				"提交幂等关系确认事务失败: %w",
				err,
			)
		}
		return current, nil
	} else {
		eventType =
			models.CWReviewItemRelationEventReactivated

		// 重新启用时保存本次明确采用的说明。
		// 对AI建议关系，这是可信消息中的说明；
		// 对问题清单直接关系，这是用户本次填写的人工说明。
		eventReason = relation.Explanation

		current, err =
			reactivateCoursewareReviewItemRelationTx(
				ctx,
				tx,
				current,
				relation.Explanation,
			)
		if err != nil {
			return nil, err
		}
	}

	if err := insertCoursewareReviewItemRelationEventTx(
		ctx,
		tx,
		current,
		eventType,
		actorID,
		eventReason,
		relation.SourceGlobalMessageID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交整改项关系确认事务失败: %w",
			err,
		)
	}

	return current, nil
}

// CancelCoursewareReviewItemRelation 取消一条尚未涉及正式交付历史的关系。
func CancelCoursewareReviewItemRelation(
	ctx context.Context,
	sessionID string,
	relationID string,
	actorID string,
	reason string,
) (*models.CoursewareReviewItemRelation, error) {
	sessionID = strings.TrimSpace(sessionID)
	relationID = strings.TrimSpace(relationID)
	actorID = strings.TrimSpace(actorID)
	reason = strings.TrimSpace(reason)

	if sessionID == "" ||
		relationID == "" ||
		actorID == "" ||
		reason == "" {
		return nil, errors.New(
			"取消整改项关系参数无效",
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始取消整改项关系事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err :=
		scanCoursewareReviewItemRelation(
			tx.QueryRow(
				ctx,
				`SELECT `+
					cwReviewItemRelationSelectColumns+
					`
				 FROM courseware_review_item_relations
				 WHERE id = $1
				   AND source_session_id = $2
				   AND created_by = $3
				 FOR UPDATE`,
				relationID,
				sessionID,
				actorID,
			),
		)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewItemRelationNotFound
		}
		return nil, fmt.Errorf(
			"锁定整改项关系失败: %w",
			err,
		)
	}

	if current.Status ==
		models.CWReviewItemRelationStatusCancelled {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf(
				"提交幂等关系取消事务失败: %w",
				err,
			)
		}
		return current, nil
	}

	if err := lockGovernableCWReviewItemPairTx(
		ctx,
		tx,
		current.SourceItemID,
		current.TargetItemID,
		current.CoursewareID,
		current.SourceSessionID,
		actorID,
		false,
	); err != nil {
		return nil, err
	}

	next, err :=
		cancelCoursewareReviewItemRelationTx(
			ctx,
			tx,
			current,
			actorID,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		insertCoursewareReviewItemRelationEventTx(
			ctx,
			tx,
			next,
			models.
				CWReviewItemRelationEventCancelled,
			actorID,
			reason,
			nil,
		); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交整改项关系取消事务失败: %w",
			err,
		)
	}

	return next, nil
}

// ListCoursewareReviewItemRelationsBySession 返回会话创建者确认过的全部关系。
func ListCoursewareReviewItemRelationsBySession(
	ctx context.Context,
	sessionID string,
	creatorID string,
) ([]*models.CoursewareReviewItemRelation, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT `+
			cwReviewItemRelationSelectColumns+
			`
		 FROM courseware_review_item_relations
		 WHERE source_session_id = $1
		   AND created_by = $2
		 ORDER BY
			CASE status
				WHEN 'active' THEN 1
				ELSE 2
			END,
			updated_at DESC,
			id ASC`,
		strings.TrimSpace(sessionID),
		strings.TrimSpace(creatorID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询整改项关系列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	relations := make(
		[]*models.CoursewareReviewItemRelation,
		0,
	)
	for rows.Next() {
		relation, scanErr :=
			scanCoursewareReviewItemRelation(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描整改项关系失败: %w",
				scanErr,
			)
		}
		relations = append(relations, relation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历整改项关系失败: %w",
			err,
		)
	}

	return relations, nil
}

// ListCoursewareReviewItemRelationEvents 返回一条关系的追加式历史。
//
// 查询必须同时匹配关系ID、来源会话和关系创建者，不能仅凭关系ID
// 读取另一名审核员或另一会话中的治理历史。
func ListCoursewareReviewItemRelationEvents(
	ctx context.Context,
	relationID string,
	sessionID string,
	creatorID string,
) ([]*models.CoursewareReviewItemRelationEvent, error) {
	rows, err := database.DB.Query(
		ctx,
		`
		SELECT
			event.id,
			event.relation_id,
			event.source_session_id,
			event.relation_version,
			event.event_type,
			event.actor_id,
			event.reason,
			COALESCE(
				event.source_global_message_id::text,
				''
			),
			COALESCE(event.metadata_json::text, '{}'),
			event.created_at
		FROM courseware_review_item_relation_events AS event
		INNER JOIN courseware_review_item_relations AS relation
			ON relation.id = event.relation_id
			AND relation.source_session_id =
				event.source_session_id
		WHERE event.relation_id = $1
		  AND event.source_session_id = $2
		  AND relation.created_by = $3
		ORDER BY event.relation_version ASC`,
		strings.TrimSpace(relationID),
		strings.TrimSpace(sessionID),
		strings.TrimSpace(creatorID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询整改项关系事件失败: %w",
			err,
		)
	}
	defer rows.Close()

	events := make(
		[]*models.CoursewareReviewItemRelationEvent,
		0,
	)
	for rows.Next() {
		event :=
			&models.CoursewareReviewItemRelationEvent{}
		var sourceGlobalMessageID string

		if err := rows.Scan(
			&event.ID,
			&event.RelationID,
			&event.SourceSessionID,
			&event.RelationVersion,
			&event.EventType,
			&event.ActorID,
			&event.Reason,
			&sourceGlobalMessageID,
			&event.MetadataJSON,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描整改项关系事件失败: %w",
				err,
			)
		}

		if sourceGlobalMessageID != "" {
			event.SourceGlobalMessageID =
				&sourceGlobalMessageID
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历整改项关系事件失败: %w",
			err,
		)
	}

	return events, nil
}
