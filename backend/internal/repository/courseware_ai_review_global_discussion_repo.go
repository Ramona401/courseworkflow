package repository

// courseware_ai_review_global_discussion_repo.go
//
// 课件AI审核会话级全局讨论的数据访问层。
//
// 存储协议：
//   1. 继续复用courseware_ai_review_messages，不建设第二套消息表；
//   2. review_item_id为NULL表示会话级全局讨论；
//   3. review_item_id非空仍表示原有单条整改项讨论；
//   4. 会话级读写必须同时绑定session_id和reviewer_id；
//   5. 从全局讨论采用候选指令时，只追加整改项assistant消息，
//      不修改整改项状态、确认指令或正式审核决定；
//   6. 候选消息写入再次校验整改项来源会话、参与者、状态和未交付边界。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrCoursewareAIReviewMessageNotFound 合并消息不存在与会话写入边界不满足。
var ErrCoursewareAIReviewMessageNotFound = errors.New(
	"课件AI审核全局讨论消息不存在",
)

func scanCoursewareAIReviewSessionMessage(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareAIReviewMessage, error) {
	message := &models.CoursewareAIReviewMessage{}
	var userID string

	err := row.Scan(
		&message.ID,
		&message.SessionID,
		&userID,
		&message.Role,
		&message.Content,
		&message.CitationsJSON,
		&message.TokensUsed,
		&message.ModelUsed,
		&message.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if userID != "" {
		message.UserID = &userID
	}

	return message, nil
}

// AppendCoursewareAIReviewSessionMessage 追加一条会话级全局讨论消息。
//
// INSERT ... SELECT确保会话属于指定审核员；review_item_id明确写NULL，
// 不会混入单条整改项讨论记录。
func AppendCoursewareAIReviewSessionMessage(
	ctx context.Context,
	message *models.CoursewareAIReviewMessage,
	reviewerID string,
) error {
	if message == nil {
		return errors.New("课件AI审核全局讨论消息不能为空")
	}

	message.SessionID = strings.TrimSpace(message.SessionID)
	message.Role = strings.TrimSpace(message.Role)
	message.Content = strings.TrimSpace(message.Content)
	reviewerID = strings.TrimSpace(reviewerID)

	if message.SessionID == "" || reviewerID == "" {
		return ErrCoursewareAIReviewMessageNotFound
	}
	if message.Content == "" {
		return errors.New("课件AI审核全局讨论消息内容不能为空")
	}

	switch message.Role {
	case "system", "user", "assistant":
	default:
		return errors.New("课件AI审核全局讨论消息角色无效")
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
			session.id,
			NULL,
			$2,
			$3,
			$4,
			$5::jsonb,
			$6,
			$7,
			NOW()
		FROM courseware_ai_review_sessions session
		WHERE session.id = $1
		  AND session.reviewer_id = $8
		RETURNING id, created_at`,
		message.SessionID,
		message.UserID,
		message.Role,
		message.Content,
		cwAIReviewJSONOrDefault(message.CitationsJSON, "{}"),
		message.TokensUsed,
		strings.TrimSpace(message.ModelUsed),
		reviewerID,
	).Scan(
		&message.ID,
		&message.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCoursewareAIReviewMessageNotFound
		}
		return fmt.Errorf("追加课件AI审核全局讨论消息失败: %w", err)
	}

	return nil
}

// ListCoursewareAIReviewSessionMessages 返回指定审核员会话的全部全局讨论消息。
func ListCoursewareAIReviewSessionMessages(
	ctx context.Context,
	sessionID string,
	reviewerID string,
) ([]*models.CoursewareAIReviewMessage, error) {
	rows, err := database.DB.Query(
		ctx,
		`
		SELECT
			message.id,
			message.session_id,
			COALESCE(message.user_id::text, ''),
			message.role,
			message.content,
			COALESCE(message.citations_json::text, '{}'),
			message.tokens_used,
			message.model_used,
			message.created_at
		FROM courseware_ai_review_messages message
		INNER JOIN courseware_ai_review_sessions session
			ON session.id = message.session_id
		WHERE message.session_id = $1
		  AND message.review_item_id IS NULL
		  AND session.reviewer_id = $2
		ORDER BY message.created_at ASC, message.id ASC`,
		strings.TrimSpace(sessionID),
		strings.TrimSpace(reviewerID),
	)
	if err != nil {
		return nil, fmt.Errorf("查询课件AI审核全局讨论消息失败: %w", err)
	}
	defer rows.Close()

	messages := make([]*models.CoursewareAIReviewMessage, 0)

	for rows.Next() {
		message, scanErr := scanCoursewareAIReviewSessionMessage(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("扫描课件AI审核全局讨论消息失败: %w", scanErr)
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历课件AI审核全局讨论消息失败: %w", err)
	}

	return messages, nil
}

// GetCoursewareAIReviewSessionMessageByID 读取一条可信会话级助手消息。
//
// 采用候选指令时必须重新从本函数读取，不接受浏览器回传候选正文。
func GetCoursewareAIReviewSessionMessageByID(
	ctx context.Context,
	sessionID string,
	messageID string,
	reviewerID string,
) (*models.CoursewareAIReviewMessage, error) {
	message, err := scanCoursewareAIReviewSessionMessage(
		database.DB.QueryRow(
			ctx,
			`
			SELECT
				message.id,
				message.session_id,
				COALESCE(message.user_id::text, ''),
				message.role,
				message.content,
				COALESCE(message.citations_json::text, '{}'),
				message.tokens_used,
				message.model_used,
				message.created_at
			FROM courseware_ai_review_messages message
			INNER JOIN courseware_ai_review_sessions session
				ON session.id = message.session_id
			WHERE message.id = $1
			  AND message.session_id = $2
			  AND message.review_item_id IS NULL
			  AND session.reviewer_id = $3`,
			strings.TrimSpace(messageID),
			strings.TrimSpace(sessionID),
			strings.TrimSpace(reviewerID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareAIReviewMessageNotFound
		}
		return nil, fmt.Errorf("读取课件AI审核全局讨论消息失败: %w", err)
	}

	return message, nil
}

// AppendCoursewareReviewItemCandidateFromGlobalDiscussion
// 将全局讨论中的可信候选指令追加到单条整改项讨论历史。
//
// 本函数只写assistant消息，不改状态或confirmed_instruction。
// SQL层再次要求：
//   - 整改项属于同一来源会话；
//   - 正式项由创建审核员操作，自审项由课件作者操作；
//   - 状态仍可处理；
//   - 尚未绑定正式审核反馈。
func AppendCoursewareReviewItemCandidateFromGlobalDiscussion(
	ctx context.Context,
	message *models.CoursewareReviewItemMessage,
	participantID string,
) error {
	if message == nil {
		return errors.New("全局讨论采用的整改项候选消息不能为空")
	}

	message.SessionID = strings.TrimSpace(message.SessionID)
	message.ReviewItemID = strings.TrimSpace(message.ReviewItemID)
	message.Content = strings.TrimSpace(message.Content)
	participantID = strings.TrimSpace(participantID)

	if message.SessionID == "" ||
		message.ReviewItemID == "" ||
		participantID == "" {
		return ErrCoursewareReviewItemNotFound
	}
	if message.Content == "" {
		return errors.New("全局讨论采用的候选消息内容不能为空")
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
			item.source_session_id,
			item.id,
			NULL,
			'assistant',
			$3,
			$4::jsonb,
			$5,
			$6,
			NOW()
		FROM courseware_review_items item
		WHERE item.id = $2
		  AND item.source_session_id = $1
		  AND (
			(
				item.source_type = 'formal'
				AND item.created_by = $7
			)
			OR
			(
				item.source_type = 'self'
				AND item.owner_id = $7
			)
		  )
		  AND item.status IN ('detected', 'discussing', 'confirmed')
		  AND item.courseware_review_id IS NULL
		  AND item.feedback_id IS NULL
		RETURNING id, created_at`,
		message.SessionID,
		message.ReviewItemID,
		message.Content,
		cwAIReviewJSONOrDefault(message.CitationsJSON, "{}"),
		message.TokensUsed,
		strings.TrimSpace(message.ModelUsed),
		participantID,
	).Scan(
		&message.ID,
		&message.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCoursewareReviewItemConflict
		}
		return fmt.Errorf("保存全局讨论采用的整改项候选指令失败: %w", err)
	}

	message.Role = "assistant"

	return nil
}
