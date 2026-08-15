package repository

// courseware_review_item_goal_drift_repo.go
//
// 目标漂移人工拆项的事务仓储。
//
// 该事务只创建新的独立整改项和它自己的系统审计消息。
// 来源问题只被FOR UPDATE重新校验，绝不修改其状态、确认要求、版本或讨论历史。
//
// 并发与重复提交：
//   - 来源问题在事务内重新锁定并校验仍未正式交付；
//   - 同一来源问题、同一操作者、同一文字的重复请求复用已有未交付拆项；
//   - 创建整改项与系统审计消息必须同一事务成功。

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

type cwReviewGoalDriftCreationMeta struct {
	Event string `json:"event"`
	Label string `json:"label"`
}

// CreateCoursewareReviewGoalDriftItem 从一个仍可编辑的未交付问题创建独立新问题。
func CreateCoursewareReviewGoalDriftItem(
	ctx context.Context,
	sourceItemID string,
	actorID string,
	item *models.CoursewareReviewItem,
) (*models.CoursewareReviewItem, error) {
	sourceItemID = strings.TrimSpace(sourceItemID)
	actorID = strings.TrimSpace(actorID)

	if sourceItemID == "" ||
		actorID == "" ||
		item == nil ||
		item.OriginType != models.CWReviewItemOriginGoalDriftManual ||
		strings.TrimSpace(item.CoursewareID) == "" ||
		strings.TrimSpace(item.SourceSessionID) == "" ||
		strings.TrimSpace(item.SourceFindingID) == "" ||
		strings.TrimSpace(item.CreatedBy) != actorID ||
		strings.TrimSpace(item.OwnerID) == "" ||
		strings.TrimSpace(item.OriginalSuggestion) == "" {
		return nil, errors.New("目标漂移人工拆项参数无效")
	}

	if item.SourceGlobalMessageID != nil ||
		item.CoursewareReviewID != nil ||
		item.FeedbackID != nil ||
		item.DeliveredInstructionVersionID != nil ||
		item.AppliedInstructionVersionID != nil {
		return nil, ErrCoursewareReviewItemConflict
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始目标漂移人工拆项事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		sourceType      string
		sourceCreatedBy string
		sourceOwnerID   string
		sourceStatus    string
		hasReview       bool
		hasFeedback     bool
		hasDeliveredVer bool
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			source_type,
			created_by::text,
			owner_id::text,
			status,
			courseware_review_id IS NOT NULL,
			feedback_id IS NOT NULL,
			delivered_instruction_version_id IS NOT NULL
		FROM courseware_review_items
		WHERE id = $1
		  AND courseware_id = $2
		  AND source_session_id = $3
		FOR UPDATE`,
		sourceItemID,
		strings.TrimSpace(item.CoursewareID),
		strings.TrimSpace(item.SourceSessionID),
	).Scan(
		&sourceType,
		&sourceCreatedBy,
		&sourceOwnerID,
		&sourceStatus,
		&hasReview,
		&hasFeedback,
		&hasDeliveredVer,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewItemConflict
		}
		return nil, fmt.Errorf("锁定目标漂移来源问题失败: %w", err)
	}

	actorAllowed :=
		(sourceType == models.CWReviewItemSourceFormal &&
			sourceCreatedBy == actorID) ||
			(sourceType == models.CWReviewItemSourceSelf &&
				sourceOwnerID == actorID)

	actionable :=
		sourceStatus == models.CWReviewItemStatusDetected ||
			sourceStatus == models.CWReviewItemStatusDiscussing ||
			sourceStatus == models.CWReviewItemStatusConfirmed

	if !actorAllowed ||
		!actionable ||
		hasReview ||
		hasFeedback ||
		hasDeliveredVer ||
		item.SourceType != sourceType ||
		item.OwnerID != sourceOwnerID {
		return nil, ErrCoursewareReviewItemConflict
	}

	// 双击或网络重试时复用同一份未交付人工拆项。
	existing, existingErr := scanCoursewareReviewItem(
		tx.QueryRow(
			ctx,
			`SELECT `+cwReviewItemSelectColumns+`
			 FROM courseware_review_items
			 WHERE courseware_id = $1
			   AND source_session_id = $2
			   AND created_by = $3
			   AND origin_type = 'goal_drift_manual'
			   AND evidence_json->>'source_goal_drift_item_id' = $4
			   AND BTRIM(original_suggestion) = BTRIM($5)
			   AND courseware_review_id IS NULL
			   AND feedback_id IS NULL
			   AND delivered_instruction_version_id IS NULL
			 ORDER BY created_at ASC, id ASC
			 LIMIT 1`,
			strings.TrimSpace(item.CoursewareID),
			strings.TrimSpace(item.SourceSessionID),
			actorID,
			sourceItemID,
			strings.TrimSpace(item.OriginalSuggestion),
		),
	)
	if existingErr == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("提交目标漂移幂等复用事务失败: %w", err)
		}
		return existing, nil
	}
	if !errors.Is(existingErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("检查重复目标漂移拆项失败: %w", existingErr)
	}

	if err := CreateCoursewareReviewItemTx(ctx, tx, item); err != nil {
		return nil, err
	}

	metaJSON, err := json.Marshal(
		cwReviewGoalDriftCreationMeta{
			Event: "goal_drift_manual_item_created",
			Label: "从当前问题拆分的新改进项",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("序列化目标漂移拆项审计信息失败: %w", err)
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
		"已把这段内容作为新的独立改进项保存。原问题的修改要求和以前记录没有改变。",
		string(metaJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("记录目标漂移人工拆项事件失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交目标漂移人工拆项事务失败: %w", err)
	}

	return item, nil
}
