package repository

// lesson_plan_publish_guard_repo.go — 教案个人发布的原子版本与Word同步守卫
//
// 发布只改变状态，不改变正文版本号。

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
	ErrLessonPlanPublishNotFound        = errors.New("待发布教案不存在")
	ErrLessonPlanPublishNotAuthor       = errors.New("只有作者可以发布此教案")
	ErrLessonPlanPublishStatusInvalid   = errors.New("教案当前状态不允许个人发布")
	ErrLessonPlanPublishContentEmpty    = errors.New("教案正文为空")
	ErrLessonPlanPublishVersionConflict = errors.New("教案版本已变化")
	ErrLessonPlanPublishWordOutOfSync   = errors.New("原格式Word与当前正文不同步")
)

// CommitLessonPlanPersonalPublishAtVersion 原子校验作者、状态、版本、正文和Word后发布。
func CommitLessonPlanPersonalPublishAtVersion(
	ctx context.Context,
	lessonPlanID string,
	ownerID string,
	expectedVersion int,
) error {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	ownerID = strings.TrimSpace(ownerID)
	if lessonPlanID == "" || ownerID == "" {
		return ErrLessonPlanPublishNotFound
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始教案个人发布事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var authorID, status, contentMarkdown string
	var version int
	err = tx.QueryRow(ctx, `
SELECT author_id::text, status, version, COALESCE(content_markdown, '')
FROM lesson_plans
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE
`, lessonPlanID).Scan(&authorID, &status, &version, &contentMarkdown)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLessonPlanPublishNotFound
		}
		return fmt.Errorf("锁定待发布教案失败: %w", err)
	}

	if strings.TrimSpace(authorID) != ownerID {
		return ErrLessonPlanPublishNotAuthor
	}
	switch status {
	case models.LPStatusDraft, models.LPStatusRevision, models.LPStatusPublishedPersonal:
	default:
		return ErrLessonPlanPublishStatusInvalid
	}
	if expectedVersion > 0 && version != expectedVersion {
		return ErrLessonPlanPublishVersionConflict
	}
	if strings.TrimSpace(contentMarkdown) == "" {
		return ErrLessonPlanPublishContentEmpty
	}

	var wordStatus, wordSemantic string
	wordErr := tx.QueryRow(ctx, `
SELECT status, semantic_markdown
FROM lesson_plan_word_documents
WHERE lesson_plan_id = $1
FOR UPDATE
`, lessonPlanID).Scan(&wordStatus, &wordSemantic)
	switch {
	case wordErr == nil:
		if wordStatus != models.LessonPlanWordDocumentStatusActive ||
			wordSemantic != contentMarkdown {
			return ErrLessonPlanPublishWordOutOfSync
		}
	case errors.Is(wordErr, pgx.ErrNoRows):
		// 普通教案没有Word文档，可以继续发布。
	default:
		return fmt.Errorf("锁定待发布Word文档失败: %w", wordErr)
	}

	result, err := tx.Exec(ctx, `
UPDATE lesson_plans
SET status = $1, updated_at = NOW()
WHERE id = $2 AND version = $3 AND deleted_at IS NULL
`, models.LPStatusPublishedPersonal, lessonPlanID, version)
	if err != nil {
		return fmt.Errorf("更新教案个人发布状态失败: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrLessonPlanPublishVersionConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交教案个人发布事务失败: %w", err)
	}
	return nil
}
