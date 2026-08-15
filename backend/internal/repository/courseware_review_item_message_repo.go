package repository

// courseware_review_item_message_repo.go
//
// 课件AI审核整改项独立讨论消息的数据访问层。
//
// 核心约束：
//   1. 消息必须同时匹配整改项ID和来源AI会话ID，防止跨会话串线；
//   2. INSERT ... SELECT确保消息只能写入实际属于该会话的整改项；
//   3. 消息按创建时间和ID稳定排序；
//   4. 本仓储不执行教育域授权，调用方仍需先完成参与者权限校验。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// AppendCoursewareReviewItemMessage 追加一条独立问题讨论消息。
//
// INSERT ... SELECT确保整改项确实属于指定AI审核会话。
func AppendCoursewareReviewItemMessage(
	ctx context.Context,
	message *models.CoursewareReviewItemMessage,
) error {
	if message == nil {
		return errors.New("课件整改讨论消息不能为空")
	}

	message.Content = strings.TrimSpace(message.Content)
	if message.Content == "" {
		return errors.New("课件整改讨论消息内容不能为空")
	}

	err := database.DB.QueryRow(
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
                SELECT
                        $1, $2, $3, $4, $5, $6::jsonb, $7, $8, NOW()
                FROM courseware_review_items item
                WHERE item.id = $2
                  AND item.source_session_id = $1
                RETURNING id, created_at`,
		strings.TrimSpace(message.SessionID),
		strings.TrimSpace(message.ReviewItemID),
		message.UserID,
		strings.TrimSpace(message.Role),
		message.Content,
		cwAIReviewJSONOrDefault(
			message.CitationsJSON,
			"[]",
		),
		message.TokensUsed,
		strings.TrimSpace(message.ModelUsed),
	).Scan(&message.ID, &message.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCoursewareReviewItemNotFound
		}
		return fmt.Errorf("追加课件整改讨论消息失败: %w", err)
	}

	return nil
}

// AppendCoursewareReviewItemExecutionNote 追加作者正式整改过程中的“本次执行补充”。
//
// 与通用消息写入不同，本入口在同一条INSERT ... SELECT中再次校验：
//   - 当前操作者仍是课件作者；
//   - 整改项仍是已经正式交付的formal项；
//   - 状态仍处于作者整改或待复审生命周期。
//
// 这样即使Service校验后发生并发状态变化，也不会把补充写进已经解决的历史记录。
func AppendCoursewareReviewItemExecutionNote(
	ctx context.Context,
	message *models.CoursewareReviewItemMessage,
	ownerID string,
) error {
	if message == nil {
		return errors.New("课件整改执行补充消息不能为空")
	}

	message.Content = strings.TrimSpace(message.Content)
	ownerID = strings.TrimSpace(ownerID)
	if message.Content == "" || ownerID == "" {
		return errors.New("课件整改执行补充内容或作者不能为空")
	}

	err := database.DB.QueryRow(
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
                SELECT
                        $1, $2, $3, 'user', $4, $5::jsonb, 0, '', NOW()
                FROM courseware_review_items item
                WHERE item.id = $2
                  AND item.source_session_id = $1
                  AND item.owner_id = $3
                  AND item.source_type = 'formal'
                  AND item.courseware_review_id IS NOT NULL
                  AND item.feedback_id IS NOT NULL
                  AND item.delivered_instruction_version_id IS NOT NULL
                  AND item.status IN (
                        'confirmed',
                        'applying',
                        'applied',
                        'stale',
                        'orphaned'
                  )
                RETURNING id, created_at`,
		strings.TrimSpace(message.SessionID),
		strings.TrimSpace(message.ReviewItemID),
		ownerID,
		message.Content,
		cwAIReviewJSONOrDefault(
			message.CitationsJSON,
			"{}",
		),
	).Scan(&message.ID, &message.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCoursewareReviewItemConflict
		}
		return fmt.Errorf(
			"追加课件整改执行补充失败: %w",
			err,
		)
	}

	return nil
}

// ListCoursewareReviewItemMessages 按时间正序返回单条整改项讨论。
func ListCoursewareReviewItemMessages(
	ctx context.Context,
	itemID string,
) ([]*models.CoursewareReviewItemMessage, error) {
	rows, err := database.DB.Query(
		ctx,
		`
                SELECT
                        id,
                        session_id,
                        review_item_id,
                        COALESCE(user_id::text, ''),
                        role,
                        content,
                        COALESCE(citations_json::text, '[]'),
                        tokens_used,
                        model_used,
                        created_at
                FROM courseware_ai_review_messages
                WHERE review_item_id = $1
                ORDER BY created_at ASC, id ASC`,
		strings.TrimSpace(itemID),
	)
	if err != nil {
		return nil, fmt.Errorf("查询课件整改讨论消息失败: %w", err)
	}
	defer rows.Close()

	messages := make([]*models.CoursewareReviewItemMessage, 0)
	for rows.Next() {
		message := &models.CoursewareReviewItemMessage{}
		var userID string

		if err := rows.Scan(
			&message.ID,
			&message.SessionID,
			&message.ReviewItemID,
			&userID,
			&message.Role,
			&message.Content,
			&message.CitationsJSON,
			&message.TokensUsed,
			&message.ModelUsed,
			&message.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描课件整改讨论消息失败: %w",
				err,
			)
		}

		if userID != "" {
			message.UserID = &userID
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件整改讨论消息失败: %w",
			err,
		)
	}

	return messages, nil
}
