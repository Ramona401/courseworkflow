package repository

// lesson_plan_fork_repo.go — 教案Fork原子Repository
//
// 本文件承载上下文13的安全Fork写入：
//   1. 严格校验显式来源教育域；
//   2. 事务锁定来源教案；
//   3. 重新确认来源状态和教育域；
//   4. 显式写入副本education_domain；
//   5. 校验数据库最终快照；
//   6. 同事务递增来源fork_count；
//   7. 任一步失败时整体回滚。
//
// 不依赖数据库触发器推导作者域，也不允许调用者域覆盖来源资源域。
// 副本只继承旧Fork已有语义中的正文、结构、生成配置、组件、
// 模板、配方和课本引用；不继承对话、阶段运行状态、评审结果、
// 单元方案、班级学情或课程大纲版本。

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	// ErrLessonPlanForkEducationDomainRequired
	// 表示Fork没有收到具体来源教育域。
	ErrLessonPlanForkEducationDomainRequired = errors.New(
		"Fork必须显式提供来源教案的具体教学教育域",
	)

	// ErrLessonPlanForkEducationDomainMismatch
	// 表示事务中重新读取的来源域与Service传入域不一致。
	ErrLessonPlanForkEducationDomainMismatch = errors.New(
		"Fork来源教案教育域与调用者教育域不一致",
	)

	// ErrLessonPlanForkEducationDomainSnapshotMismatch
	// 表示副本数据库最终快照与来源域不一致。
	ErrLessonPlanForkEducationDomainSnapshotMismatch = errors.New(
		"Fork副本教育域快照与来源教案不一致",
	)

	// ErrLessonPlanForkSourceNotForkable
	// 表示来源状态已经不再允许Fork。
	ErrLessonPlanForkSourceNotForkable = errors.New(
		"Fork来源教案状态不允许复制",
	)
)

// normalizeLessonPlanForkEducationDomain
// 规范化并严格校验Fork来源教育域。
func normalizeLessonPlanForkEducationDomain(
	educationDomain string,
) (string, error) {
	domain := strings.ToLower(
		strings.TrimSpace(educationDomain),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return "", fmt.Errorf(
			"%w: %q",
			ErrLessonPlanForkEducationDomainRequired,
			educationDomain,
		)
	}

	return domain, nil
}

// ForkLessonPlanWithEducationDomain
// 原子复制教案并显式继承来源教案教育域。
func ForkLessonPlanWithEducationDomain(
	ctx context.Context,
	sourceID string,
	newAuthorID string,
	educationDomain string,
) (*models.LessonPlan, error) {
	sourceID = strings.TrimSpace(sourceID)
	newAuthorID = strings.TrimSpace(newAuthorID)

	if sourceID == "" || newAuthorID == "" {
		return nil, fmt.Errorf(
			"Fork教案失败：来源ID或新作者ID为空",
		)
	}

	domain, err :=
		normalizeLessonPlanForkEducationDomain(
			educationDomain,
		)
	if err != nil {
		return nil, err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始教案Fork事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		sourceTitle             string
		sourceSubject           string
		sourceGrade             string
		sourceTopic             string
		sourceDuration          int
		sourceContentMarkdown   string
		sourceContentStructured string
		sourceGenerationConfig  string
		sourceMatchedComponents string
		sourceTextbookPageIDs   string
		sourceStoredDomain      string
		sourceStatus            string
		sourceTemplateID        sql.NullString
		sourceRecipeID          sql.NullString
	)

	err = tx.QueryRow(ctx, `
		SELECT
			title,
			subject,
			grade,
			topic,
			duration_minutes,
			COALESCE(content_markdown, ''),
			COALESCE(content_structured::text, '{}'),
			COALESCE(generation_config::text, '{}'),
			COALESCE(matched_components::text, '[]'),
			template_id::text,
			recipe_id::text,
			COALESCE(textbook_page_ids::text, '[]'),
			COALESCE(education_domain, ''),
			status
		FROM lesson_plans
		WHERE id = $1
		  AND deleted_at IS NULL
		FOR UPDATE
	`, sourceID).Scan(
		&sourceTitle,
		&sourceSubject,
		&sourceGrade,
		&sourceTopic,
		&sourceDuration,
		&sourceContentMarkdown,
		&sourceContentStructured,
		&sourceGenerationConfig,
		&sourceMatchedComponents,
		&sourceTemplateID,
		&sourceRecipeID,
		&sourceTextbookPageIDs,
		&sourceStoredDomain,
		&sourceStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanNotFound
		}
		return nil, fmt.Errorf(
			"锁定Fork来源教案失败: %w",
			err,
		)
	}

	if sourceStatus !=
		models.LPStatusPublishedShared &&
		sourceStatus != models.LPStatusApproved {
		return nil,
			ErrLessonPlanForkSourceNotForkable
	}

	sourceStoredDomain = strings.ToLower(
		strings.TrimSpace(
			sourceStoredDomain,
		),
	)
	if !models.IsTeachingEducationDomain(
		sourceStoredDomain,
	) ||
		sourceStoredDomain != domain {
		return nil, fmt.Errorf(
			"%w: service=%s source=%s",
			ErrLessonPlanForkEducationDomainMismatch,
			domain,
			sourceStoredDomain,
		)
	}

	var templateID *string
	if sourceTemplateID.Valid {
		value := sourceTemplateID.String
		templateID = &value
	}

	var recipeID *string
	if sourceRecipeID.Valid {
		value := sourceRecipeID.String
		recipeID = &value
	}

	if sourceDuration <= 0 {
		sourceDuration = 45
	}
	if sourceContentStructured == "" {
		sourceContentStructured = "{}"
	}
	if sourceGenerationConfig == "" {
		sourceGenerationConfig = "{}"
	}
	if sourceMatchedComponents == "" {
		sourceMatchedComponents = "[]"
	}
	if sourceTextbookPageIDs == "" {
		sourceTextbookPageIDs = "[]"
	}

	newLessonPlan := &models.LessonPlan{
		Title: sourceTitle + "（副本）",

		Subject:         sourceSubject,
		Grade:           sourceGrade,
		Topic:           sourceTopic,
		DurationMinutes: sourceDuration,

		ContentMarkdown:   sourceContentMarkdown,
		ContentStructured: sourceContentStructured,
		GenerationConfig:  sourceGenerationConfig,
		MatchedComponents: sourceMatchedComponents,
		ConversationLog:   "[]",

		Status:     models.LPStatusDraft,
		Visibility: models.LPVisibilityPersonal,
		AuthorID:   newAuthorID,

		ForkedFrom: &sourceID,
		TemplateID: templateID,
		RecipeID:   recipeID,

		TextbookPageIDs: sourceTextbookPageIDs,
		EducationDomain: domain,
	}

	storedDomain := ""

	err = tx.QueryRow(ctx, `
		INSERT INTO lesson_plans (
			title,
			subject,
			grade,
			topic,
			duration_minutes,
			content_markdown,
			content_structured,
			generation_config,
			matched_components,
			conversation_log,
			status,
			visibility,
			author_id,
			group_id,
			school_id,
			template_id,
			recipe_id,
			textbook_page_ids,
			education_domain,
			forked_from
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			NULL,
			NULL,
			$14,
			$15,
			$16,
			$17,
			$18
		)
		RETURNING
			id,
			education_domain,
			created_at,
			updated_at
	`,
		newLessonPlan.Title,
		newLessonPlan.Subject,
		newLessonPlan.Grade,
		newLessonPlan.Topic,
		newLessonPlan.DurationMinutes,
		newLessonPlan.ContentMarkdown,
		newLessonPlan.ContentStructured,
		newLessonPlan.GenerationConfig,
		newLessonPlan.MatchedComponents,
		newLessonPlan.ConversationLog,
		newLessonPlan.Status,
		newLessonPlan.Visibility,
		newLessonPlan.AuthorID,
		newLessonPlan.TemplateID,
		newLessonPlan.RecipeID,
		newLessonPlan.TextbookPageIDs,
		domain,
		sourceID,
	).Scan(
		&newLessonPlan.ID,
		&storedDomain,
		&newLessonPlan.CreatedAt,
		&newLessonPlan.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"显式教育域创建Fork副本失败: %w",
			err,
		)
	}

	storedDomain = strings.ToLower(
		strings.TrimSpace(storedDomain),
	)
	if storedDomain != domain ||
		!models.IsTeachingEducationDomain(
			storedDomain,
		) {
		return nil, fmt.Errorf(
			"%w: source=%s database=%s",
			ErrLessonPlanForkEducationDomainSnapshotMismatch,
			domain,
			storedDomain,
		)
	}

	result, err := tx.Exec(ctx, `
		UPDATE lesson_plans
		SET
			fork_count = fork_count + 1,
			updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND education_domain = $2
		  AND status IN ($3, $4)
	`,
		sourceID,
		domain,
		models.LPStatusPublishedShared,
		models.LPStatusApproved,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"更新Fork来源计数失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return nil, fmt.Errorf(
			"%w: 来源状态或教育域发生变化",
			ErrLessonPlanForkEducationDomainMismatch,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交教案Fork事务失败: %w",
			err,
		)
	}

	newLessonPlan.EducationDomain = storedDomain
	return newLessonPlan, nil
}
