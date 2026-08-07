package repository

// lesson_plan_section_rewrite_repo.go — 教案段落AI修改的原子确认写入。
//
// 本仓储只负责最终“采用建议”动作，不调用AI：
//   1. FOR UPDATE锁定教案正式记录；
//   2. 复核作者、可编辑状态和expected version；
//   3. 使用统一解析器重新定位段落；
//   4. 校验生成预览时服务端返回的section hash；
//   5. 保存修改前完整正文版本快照；
//   6. 只替换目标标题下方的直属正文并递增版本；
//   7. 在同一事务裁剪到最近50个版本。
//
// 任一步失败都整体回滚，不允许出现“版本已保存但正文未更新”或反向半成功。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/utils"
)

var (
	// ErrLessonPlanSectionNotFound 合并段落不存在和定位信息失效。
	ErrLessonPlanSectionNotFound = errors.New("教案段落不存在")

	// ErrLessonPlanSectionVersionConflict 表示浏览器基于旧教案版本生成了修改建议。
	ErrLessonPlanSectionVersionConflict = errors.New("教案版本已变化")

	// ErrLessonPlanSectionHashConflict 表示目标段落在生成预览后又被修改。
	ErrLessonPlanSectionHashConflict = errors.New("教案段落内容已变化")

	// ErrLessonPlanSectionNotAuthor 表示当前用户不是教案作者。
	ErrLessonPlanSectionNotAuthor = errors.New("只有作者可以修改教案段落")

	// ErrLessonPlanSectionNotEditable 表示教案处于提交、开发或完成等锁定状态。
	ErrLessonPlanSectionNotEditable = errors.New("当前教案状态不允许修改段落")
)

// LessonPlanSectionApplyInput 是仓储执行原子替换所需的可信输入。
type LessonPlanSectionApplyInput struct {
	PlanID             string
	CallerID           string
	BaseVersion        int
	Locator            models.LessonPlanSectionLocator
	SectionHash        string
	ReplacementMarkdown string
}

// ApplyLessonPlanSectionRewrite 原子应用一条教案段落修改建议。
func ApplyLessonPlanSectionRewrite(
	ctx context.Context,
	input LessonPlanSectionApplyInput,
) (
	*models.LessonPlanSectionRewriteApplyResponse,
	error,
) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始教案段落修改事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		currentTitle         string
		currentContent       string
		currentStructured    string
		currentDuration      int
		currentVersion       int
		currentAuthorID      string
		currentStatus        string
	)

	err = tx.QueryRow(
		ctx,
		`
SELECT
	title,
	content_markdown,
	content_structured::text,
	duration_minutes,
	version,
	author_id::text,
	status
FROM lesson_plans
WHERE id = $1
  AND deleted_at IS NULL
FOR UPDATE
`,
		input.PlanID,
	).Scan(
		&currentTitle,
		&currentContent,
		&currentStructured,
		&currentDuration,
		&currentVersion,
		&currentAuthorID,
		&currentStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanNotFound
		}
		return nil, fmt.Errorf("锁定教案正式正文失败: %w", err)
	}

	if currentAuthorID != input.CallerID {
		return nil, ErrLessonPlanSectionNotAuthor
	}

	if !isLessonPlanSectionEditableStatus(currentStatus) {
		return nil, ErrLessonPlanSectionNotEditable
	}

	if input.BaseVersion <= 0 ||
		currentVersion != input.BaseVersion {
		return nil, ErrLessonPlanSectionVersionConflict
	}

	section, found := utils.FindLessonPlanDocumentSection(
		currentContent,
		input.Locator,
	)
	if !found {
		return nil, ErrLessonPlanSectionNotFound
	}

	if input.SectionHash == "" ||
		section.SectionHash != input.SectionHash {
		return nil, ErrLessonPlanSectionHashConflict
	}

	nextContent := utils.ReplaceLessonPlanDocumentSectionBody(
		currentContent,
		section,
		input.ReplacementMarkdown,
	)

	// 相同正文属于幂等成功，不新增版本快照，也不递增版本。
	if nextContent == currentContent {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("提交教案段落无变化事务失败: %w", err)
		}

		return &models.LessonPlanSectionRewriteApplyResponse{
			Changed:         false,
			CurrentVersion:  currentVersion,
			ContentMarkdown: currentContent,
		}, nil
	}

	changeSummary := fmt.Sprintf(
		"AI修改教案段落：%s",
		section.Title,
	)

	_, err = tx.Exec(
		ctx,
		`
INSERT INTO lesson_plan_content_versions (
	lesson_plan_id,
	version_number,
	title,
	content_markdown,
	content_structured,
	duration_minutes,
	change_source,
	changed_by,
	change_summary
) VALUES (
	$1,
	$2,
	$3,
	$4,
	$5::jsonb,
	$6,
	$7,
	$8,
	$9
)
ON CONFLICT (
	lesson_plan_id,
	version_number
) DO NOTHING
`,
		input.PlanID,
		currentVersion,
		currentTitle,
		currentContent,
		currentStructured,
		currentDuration,
		models.LPVersionSourceAI,
		input.CallerID,
		changeSummary,
	)
	if err != nil {
		return nil, fmt.Errorf("保存教案段落修改前版本失败: %w", err)
	}

	nextVersion := currentVersion + 1
	now := time.Now()

	result, err := tx.Exec(
		ctx,
		`
UPDATE lesson_plans
SET
	content_markdown = $1,
	version = $2,
	updated_at = $3
WHERE id = $4
  AND version = $5
  AND deleted_at IS NULL
`,
		nextContent,
		nextVersion,
		now,
		input.PlanID,
		currentVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("写入教案段落修改结果失败: %w", err)
	}
	if result.RowsAffected() != 1 {
		return nil, ErrLessonPlanSectionVersionConflict
	}

	_, err = tx.Exec(
		ctx,
		`
DELETE FROM lesson_plan_content_versions
WHERE id IN (
	SELECT id
	FROM lesson_plan_content_versions
	WHERE lesson_plan_id = $1
	ORDER BY version_number DESC, created_at DESC
	OFFSET 50
)
`,
		input.PlanID,
	)
	if err != nil {
		return nil, fmt.Errorf("裁剪教案正文历史版本失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交教案段落修改事务失败: %w", err)
	}

	return &models.LessonPlanSectionRewriteApplyResponse{
		Changed:         true,
		CurrentVersion:  nextVersion,
		ContentMarkdown: nextContent,
	}, nil
}

// isLessonPlanSectionEditableStatus 与教案正文普通编辑和版本恢复共用同一白名单。
func isLessonPlanSectionEditableStatus(
	status string,
) bool {
	switch status {
	case models.LPStatusDraft,
		models.LPStatusPublishedPersonal,
		models.LPStatusRevision,
		models.LPStatusApproved,
		models.LPStatusPublishedShared:
		return true
	default:
		return false
	}
}
