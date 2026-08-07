package repository

// courseware_review_item_state_repo.go
//
// 课件AI审核整改项的状态迁移、忽略、恢复和应用结果仓储。
//
// 核心约束：
//   1. 状态变化全部采用条件更新，防止重复确认、重复应用和并发覆盖；
//   2. 忽略和恢复必须在同一事务中同时写状态与可追溯系统消息；
//   3. 已交付正式反馈的整改项不能再讨论、忽略或恢复；
//   4. 重新讨论必须保留当前确认版本、兼容正文和确认时间；
//   5. 新草稿由讨论消息、AI候选和浏览器编辑态承载；
//   6. 只有明确“保存为新版并确认”才能替换当前确认版本；
//   7. formal整改项只允许创建者操作，self整改项只允许创建者或作者操作；
//   8. 本仓储只执行记录级边界，Service仍需先完成教育域和页面新鲜度校验。

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

type cwReviewItemStateEventMeta struct {
	Event          string `json:"event"`
	PreviousStatus string `json:"previous_status"`
	NextStatus     string `json:"next_status"`
	Reason         string `json:"reason,omitempty"`
}

// BeginCoursewareReviewItemDiscussion 开始或重新打开问题讨论。
//
// 已有当前确认版本时只将整改项置为discussing。
// 当前版本、兼容正文和确认时间保持不变，直到用户明确保存并确认新版。
//
// 正式交付或页面应用形成后不得重新打开讨论。
func BeginCoursewareReviewItemDiscussion(
	ctx context.Context,
	itemID string,
	participantID string,
) error {
	result, err := database.DB.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = 'discussing',
			updated_at = NOW()
		WHERE id = $1
		  AND (created_by = $2 OR owner_id = $2)
		  AND status IN ('detected', 'discussing', 'confirmed')
		  AND courseware_review_id IS NULL
		  AND feedback_id IS NULL
		  AND delivered_instruction_version_id IS NULL
		  AND applied_instruction_version_id IS NULL
		  AND applied_at IS NULL`,
		strings.TrimSpace(itemID),
		strings.TrimSpace(participantID),
	)
	if err != nil {
		return fmt.Errorf(
			"开始课件整改项讨论失败: %w",
			err,
		)
	}
	if result.RowsAffected() == 0 {
		return ErrCoursewareReviewItemConflict
	}

	return nil
}

// ConfirmCoursewareReviewItemInstruction 保存独立确认动作形成的最终修改指令。
//
// 本入口保留给旧内部调用兼容；数据库守卫会将直接更新自动物化为
// legacy_direct_update版本。正式HTTP确认入口使用显式版本服务和乐观并发ID。
func ConfirmCoursewareReviewItemInstruction(
	ctx context.Context,
	itemID string,
	participantID string,
	instruction string,
) error {
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		return errors.New("确认的课件修改指令不能为空")
	}

	result, err := database.DB.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = 'confirmed',
			confirmed_instruction = $3,
			confirmed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
		  AND (created_by = $2 OR owner_id = $2)
		  AND status IN ('detected', 'discussing', 'confirmed')
		  AND courseware_review_id IS NULL
		  AND feedback_id IS NULL
		  AND delivered_instruction_version_id IS NULL
		  AND applied_instruction_version_id IS NULL
		  AND applied_at IS NULL`,
		strings.TrimSpace(itemID),
		strings.TrimSpace(participantID),
		instruction,
	)
	if err != nil {
		return fmt.Errorf(
			"确认课件整改指令失败: %w",
			err,
		)
	}
	if result.RowsAffected() == 0 {
		return ErrCoursewareReviewItemConflict
	}

	return nil
}

// DismissCoursewareReviewItem 将未交付整改项标记为无需修改。
//
// 状态更新和系统事件写入位于同一事务，不能出现状态已忽略但原因未记录的半完成结果。
func DismissCoursewareReviewItem(
	ctx context.Context,
	itemID string,
	participantID string,
	reason string,
) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("忽略原因不能为空")
	}

	return transitionCoursewareReviewItemDismissal(
		ctx,
		itemID,
		participantID,
		reason,
		false,
	)
}

// RestoreCoursewareReviewItem 恢复一个未交付的已忽略整改项。
//
// 当前确认版本仍存在时恢复为confirmed，否则恢复为detected。
// 历史讨论及忽略原因均保留，不通过删除记录实现恢复。
func RestoreCoursewareReviewItem(
	ctx context.Context,
	itemID string,
	participantID string,
) error {
	return transitionCoursewareReviewItemDismissal(
		ctx,
		itemID,
		participantID,
		"",
		true,
	)
}

func transitionCoursewareReviewItemDismissal(
	ctx context.Context,
	itemID string,
	participantID string,
	reason string,
	restore bool,
) error {
	itemID = strings.TrimSpace(itemID)
	participantID = strings.TrimSpace(participantID)

	if itemID == "" || participantID == "" {
		return ErrCoursewareReviewItemNotFound
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始课件整改项状态事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		sessionID            string
		sourceType           string
		currentStatus        string
		confirmedInstruction string
		createdBy            string
		ownerID              string
		alreadyDelivered     bool
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			source_session_id,
			source_type,
			status,
			confirmed_instruction,
			created_by,
			owner_id,
			(
				courseware_review_id IS NOT NULL
				OR feedback_id IS NOT NULL
				OR delivered_instruction_version_id IS NOT NULL
			)
		FROM courseware_review_items
		WHERE id = $1
		  AND (created_by = $2 OR owner_id = $2)
		FOR UPDATE`,
		itemID,
		participantID,
	).Scan(
		&sessionID,
		&sourceType,
		&currentStatus,
		&confirmedInstruction,
		&createdBy,
		&ownerID,
		&alreadyDelivered,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCoursewareReviewItemNotFound
		}
		return fmt.Errorf(
			"锁定课件审核整改项失败: %w",
			err,
		)
	}

	if alreadyDelivered {
		return ErrCoursewareReviewItemConflict
	}
	if sourceType == models.CWReviewItemSourceFormal &&
		createdBy != participantID {
		return ErrCoursewareReviewItemConflict
	}
	if sourceType == models.CWReviewItemSourceSelf &&
		createdBy != participantID &&
		ownerID != participantID {
		return ErrCoursewareReviewItemConflict
	}

	nextStatus := models.CWReviewItemStatusDismissed
	eventName := "dismissed"
	messageContent := "已将此整改项标记为“无需修改”。原因：" + reason

	if restore {
		if currentStatus != models.CWReviewItemStatusDismissed {
			return ErrCoursewareReviewItemConflict
		}

		nextStatus = models.CWReviewItemStatusDetected
		if strings.TrimSpace(confirmedInstruction) != "" {
			nextStatus = models.CWReviewItemStatusConfirmed
		}

		eventName = "restored"
		messageContent = "已恢复此整改项，重新进入待处理清单。"
	} else {
		switch currentStatus {
		case models.CWReviewItemStatusDetected,
			models.CWReviewItemStatusDiscussing,
			models.CWReviewItemStatusConfirmed:
		default:
			return ErrCoursewareReviewItemConflict
		}
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = $3,
			confirmed_at = CASE
				WHEN $3 = 'detected' THEN NULL
				ELSE confirmed_at
			END,
			updated_at = NOW()
		WHERE id = $1
		  AND status = $2`,
		itemID,
		currentStatus,
		nextStatus,
	)
	if err != nil {
		return fmt.Errorf(
			"更新课件整改项忽略状态失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	metaJSON, err := json.Marshal(
		cwReviewItemStateEventMeta{
			Event:          eventName,
			PreviousStatus: currentStatus,
			NextStatus:     nextStatus,
			Reason:         reason,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"序列化课件整改项状态事件失败: %w",
			err,
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
		sessionID,
		itemID,
		participantID,
		messageContent,
		string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf(
			"记录课件整改项状态事件失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交课件整改项状态事务失败: %w",
			err,
		)
	}

	return nil
}

// TransitionCoursewareReviewItemStatus 按允许的前置状态执行通用CAS状态迁移。
func TransitionCoursewareReviewItemStatus(
	ctx context.Context,
	itemID string,
	participantID string,
	nextStatus string,
	allowedCurrentStatuses []string,
) error {
	if !models.IsCWReviewItemStatus(nextStatus) {
		return errors.New("课件整改项目标状态无效")
	}
	if len(allowedCurrentStatuses) == 0 {
		return errors.New("课件整改项缺少允许的前置状态")
	}

	result, err := database.DB.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = $3::text,
			resolved_at = CASE
				WHEN $3::text = 'resolved'::text THEN NOW()
				ELSE resolved_at
			END,
			updated_at = NOW()
		WHERE id = $1
		  AND (created_by = $2 OR owner_id = $2)
		  AND status = ANY($4::text[])`,
		strings.TrimSpace(itemID),
		strings.TrimSpace(participantID),
		strings.TrimSpace(nextStatus),
		allowedCurrentStatuses,
	)
	if err != nil {
		return fmt.Errorf(
			"迁移课件整改项状态失败: %w",
			err,
		)
	}
	if result.RowsAffected() == 0 {
		return ErrCoursewareReviewItemConflict
	}

	return nil
}

// MarkCoursewareReviewItemApplied 记录页面修改完成后的新页面哈希。
func MarkCoursewareReviewItemApplied(
	ctx context.Context,
	itemID string,
	participantID string,
	appliedPageHash string,
) error {
	result, err := database.DB.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = 'applied',
			applied_page_hash = $3,
			applied_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
		  AND (created_by = $2 OR owner_id = $2)
		  AND status IN ('confirmed', 'applying')`,
		strings.TrimSpace(itemID),
		strings.TrimSpace(participantID),
		strings.TrimSpace(appliedPageHash),
	)
	if err != nil {
		return fmt.Errorf(
			"记录课件整改项应用结果失败: %w",
			err,
		)
	}
	if result.RowsAffected() == 0 {
		return ErrCoursewareReviewItemConflict
	}

	return nil
}
