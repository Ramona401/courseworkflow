package repository

// lesson_plan_conversation_cleanup_repo.go
//
// 本文件只处理“开始对话 / 专家模式”创建失败后的同步补偿清理。
//
// 该清理不是普通删除业务：
//   - 普通用户删除教案仍走软删除和回收站；
//   - 本函数只允许删除刚创建、尚未形成完整会话的个人草稿；
//   - 仅在阶段初始化或开场消息持久化失败时由Service调用；
//   - 使用事务锁定目标教案，先删除阶段产出，再硬删除空教案；
//   - 条件不满足时拒绝删除，避免错误补偿伤及已经可用的教案。
//
// conversation_log与正文都必须保持空状态。
// 一旦开场消息已经成功写入，Service会关闭补偿开关，本函数不会被调用。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	// ErrIncompleteLessonPlanConversationCleanupRejected
	// 表示目标教案已经不再符合“失败创建空壳”的安全删除条件。
	ErrIncompleteLessonPlanConversationCleanupRejected = errors.New(
		"失败会话补偿清理被安全条件拒绝",
	)
)

// DeleteIncompleteLessonPlanConversationCreation
// 硬清理一次未完整建立的对话备课教案。
//
// 幂等规则：目标行已经不存在时视为清理完成并返回nil。
// 安全条件：
//   - author_id必须等于本次创建者；
//   - status必须为draft；
//   - visibility必须为personal；
//   - content_markdown必须为空；
//   - conversation_log必须仍为空数组。
func DeleteIncompleteLessonPlanConversationCreation(
	ctx context.Context,
	lessonPlanID string,
	authorID string,
) error {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	authorID = strings.TrimSpace(authorID)
	if lessonPlanID == "" || authorID == "" {
		return fmt.Errorf(
			"%w: 教案ID或作者ID为空",
			ErrIncompleteLessonPlanConversationCleanupRejected,
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始失败会话补偿事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		storedAuthorID  string
		status          string
		visibility      string
		contentMarkdown string
		conversationLog string
	)

	err = tx.QueryRow(ctx, `
		SELECT
			author_id::text,
			status,
			visibility,
			COALESCE(content_markdown, ''),
			COALESCE(conversation_log::text, '[]')
		FROM lesson_plans
		WHERE id = $1
		FOR UPDATE
	`, lessonPlanID).Scan(
		&storedAuthorID,
		&status,
		&visibility,
		&contentMarkdown,
		&conversationLog,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return tx.Commit(ctx)
		}
		return fmt.Errorf(
			"锁定失败会话教案失败: %w",
			err,
		)
	}

	if strings.TrimSpace(storedAuthorID) != authorID ||
		status != models.LPStatusDraft ||
		visibility != models.LPVisibilityPersonal ||
		strings.TrimSpace(contentMarkdown) != "" ||
		!isEmptyConversationLogJSON(conversationLog) {
		return fmt.Errorf(
			"%w: plan_id=%s status=%s visibility=%s",
			ErrIncompleteLessonPlanConversationCleanupRejected,
			lessonPlanID,
			status,
			visibility,
		)
	}

	// 阶段初始化是当前失败创建链唯一会产生的独立子表记录。
	// 显式删除可同时兼容“有外键级联”和“无级联”的历史数据库结构。
	if _, err := tx.Exec(ctx, `
		DELETE FROM workshop_stage_outputs
		WHERE lesson_plan_id = $1
	`, lessonPlanID); err != nil {
		return fmt.Errorf(
			"清理失败会话阶段记录失败: %w",
			err,
		)
	}

	result, err := tx.Exec(ctx, `
		DELETE FROM lesson_plans
		WHERE id = $1
		  AND author_id = $2
		  AND status = $3
		  AND visibility = $4
		  AND COALESCE(content_markdown, '') = ''
		  AND COALESCE(conversation_log::text, '[]') = '[]'
	`,
		lessonPlanID,
		authorID,
		models.LPStatusDraft,
		models.LPVisibilityPersonal,
	)
	if err != nil {
		return fmt.Errorf(
			"硬删除失败会话教案失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf(
			"%w: 删除时安全条件发生变化",
			ErrIncompleteLessonPlanConversationCleanupRejected,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交失败会话补偿事务失败: %w",
			err,
		)
	}

	return nil
}

// isEmptyConversationLogJSON 判断数据库读取出的JSONB文本是否仍为空会话。
func isEmptyConversationLogJSON(value string) bool {
	switch strings.TrimSpace(value) {
	case "", "[]", "null":
		return true
	default:
		return false
	}
}
