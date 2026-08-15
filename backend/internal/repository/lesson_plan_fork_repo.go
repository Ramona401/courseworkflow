package repository

// lesson_plan_fork_repo.go — 教案Fork原子Repository
//
// 本文件承载上下文13的安全Fork写入，并在上下文17扩展common来源：
//   1. 严格校验显式来源资源域与副本具体教学域；
//   2. 事务锁定来源教案；
//   3. 重新确认来源状态和来源资源域；
//   4. 同域来源写回原具体域，common来源写入调用者具体域；
//   5. 校验数据库最终副本快照；
//   6. 同事务递增来源fork_count；
//   7. 任一步失败时整体回滚。
//
// 不依赖数据库触发器推导作者域。具体域来源不能被调用者域覆盖；
// common只作为公共来源快照，副本必须落入调用者唯一具体教学域。
// 副本只继承旧Fork已有语义中的正文、结构、生成配置、组件、
// 模板、配方、课本引用和阶段配置模板；不继承对话、阶段进度、评审结果、
// 单元方案、班级学情或课程大纲版本。因为Fork复制的是已经存在的完整正文，
// 新副本在同一事务内按“已有完整教案”语义进入review-ready：review之前阶段skipped，
// review为唯一in_progress，保证Fork返回时正文与备课运行态语义一致。

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
	// 表示Fork没有收到合法来源资源域或副本具体教学域。
	ErrLessonPlanForkEducationDomainRequired = errors.New(
		"Fork必须显式提供合法来源资源域和副本具体教学域",
	)

	// ErrLessonPlanForkEducationDomainMismatch
	// 表示事务中来源资源域变化，或具体来源与副本域不一致。
	ErrLessonPlanForkEducationDomainMismatch = errors.New(
		"Fork来源资源域与副本教育域不兼容",
	)

	// ErrLessonPlanForkEducationDomainSnapshotMismatch
	// 表示副本数据库最终快照与目标具体教学域不一致。
	ErrLessonPlanForkEducationDomainSnapshotMismatch = errors.New(
		"Fork副本教育域快照与目标教学域不一致",
	)

	// ErrLessonPlanForkSourceNotForkable
	// 表示来源状态已经不再允许Fork。
	ErrLessonPlanForkSourceNotForkable = errors.New(
		"Fork来源教案状态不允许复制",
	)
)

// normalizeLessonPlanForkSourceDomain 规范化并校验来源资源域。
// 来源允许k12、vocational、adult和common，mixed及非法值拒绝。
func normalizeLessonPlanForkSourceDomain(
	educationDomain string,
) (string, error) {
	domain := strings.ToLower(
		strings.TrimSpace(educationDomain),
	)
	if !models.IsResourceEducationDomain(domain) {
		return "", fmt.Errorf(
			"%w: source=%q",
			ErrLessonPlanForkEducationDomainRequired,
			educationDomain,
		)
	}

	return domain, nil
}

// normalizeLessonPlanForkTargetDomain 规范化并校验副本具体教学域。
// common不能成为副本运行域；副本必须落入调用者的唯一具体教学域。
func normalizeLessonPlanForkTargetDomain(
	educationDomain string,
) (string, error) {
	domain := strings.ToLower(
		strings.TrimSpace(educationDomain),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return "", fmt.Errorf(
			"%w: target=%q",
			ErrLessonPlanForkEducationDomainRequired,
			educationDomain,
		)
	}

	return domain, nil
}

// ForkLessonPlanWithEducationDomain 保留上下文13旧签名兼容。
//
// 旧调用方只支持具体域来源，因此来源域和副本域使用同一个值。
// 上下文17正式Service使用ForkLessonPlanWithEducationDomains。
func ForkLessonPlanWithEducationDomain(
	ctx context.Context,
	sourceID string,
	newAuthorID string,
	educationDomain string,
) (*models.LessonPlan, error) {
	return ForkLessonPlanWithEducationDomains(
		ctx,
		sourceID,
		newAuthorID,
		educationDomain,
		educationDomain,
	)
}

// ForkLessonPlanWithEducationDomains 原子复制教案。
//
// sourceEducationDomain是来源资源快照，可为common；
// targetEducationDomain必须是调用者唯一具体教学域。
func ForkLessonPlanWithEducationDomains(
	ctx context.Context,
	sourceID string,
	newAuthorID string,
	sourceEducationDomain string,
	targetEducationDomain string,
) (*models.LessonPlan, error) {
	sourceID = strings.TrimSpace(sourceID)
	newAuthorID = strings.TrimSpace(newAuthorID)

	if sourceID == "" || newAuthorID == "" {
		return nil, fmt.Errorf(
			"Fork教案失败：来源ID或新作者ID为空",
		)
	}

	sourceDomain, err :=
		normalizeLessonPlanForkSourceDomain(
			sourceEducationDomain,
		)
	if err != nil {
		return nil, err
	}

	targetDomain, err :=
		normalizeLessonPlanForkTargetDomain(
			targetEducationDomain,
		)
	if err != nil {
		return nil, err
	}

	// 具体域来源只能Fork到同一具体域；common来源可落入任一具体教学域。
	if sourceDomain != models.EducationDomainCommon &&
		sourceDomain != targetDomain {
		return nil, fmt.Errorf(
			"%w: source=%s target=%s",
			ErrLessonPlanForkEducationDomainMismatch,
			sourceDomain,
			targetDomain,
		)
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
		sourceStageConfig       string
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
                        COALESCE(stage_config::text, '[]'),
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
		&sourceStageConfig,
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
	if !models.IsResourceEducationDomain(
		sourceStoredDomain,
	) ||
		sourceStoredDomain != sourceDomain ||
		(sourceStoredDomain != models.EducationDomainCommon &&
			sourceStoredDomain != targetDomain) {
		return nil, fmt.Errorf(
			"%w: expected_source=%s source=%s target=%s",
			ErrLessonPlanForkEducationDomainMismatch,
			sourceDomain,
			sourceStoredDomain,
			targetDomain,
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

	stageBootstrap, err :=
		resolveLessonPlanForkStageBootstrapTx(
			ctx,
			tx,
			sourceStageConfig,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"初始化Fork阶段运行态失败: %w",
			err,
		)
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
		EducationDomain: targetDomain,
		CurrentStage:    stageBootstrap.CurrentStage.StageCode,
		StageConfig:     stageBootstrap.ConfigJSON,
	}

	storedDomain := ""
	storedCurrentStage := ""
	storedStageConfig := ""

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
                        forked_from,
                        current_stage,
                        stage_config
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
                        $18,
                        $19,
                        $20
                )
                RETURNING
                        id,
                        education_domain,
                        current_stage,
                        stage_config::text,
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
		targetDomain,
		sourceID,
		newLessonPlan.CurrentStage,
		newLessonPlan.StageConfig,
	).Scan(
		&newLessonPlan.ID,
		&storedDomain,
		&storedCurrentStage,
		&storedStageConfig,
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
	if storedDomain != targetDomain ||
		!models.IsTeachingEducationDomain(
			storedDomain,
		) {
		return nil, fmt.Errorf(
			"%w: target=%s database=%s",
			ErrLessonPlanForkEducationDomainSnapshotMismatch,
			targetDomain,
			storedDomain,
		)
	}

	if storedCurrentStage !=
		stageBootstrap.CurrentStage.StageCode ||
		strings.TrimSpace(storedStageConfig) == "" ||
		strings.TrimSpace(storedStageConfig) == "[]" {
		return nil, fmt.Errorf(
			"Fork副本阶段运行态快照不一致: stage=%q",
			storedCurrentStage,
		)
	}

	if err := createLessonPlanForkStageOutputsTx(
		ctx,
		tx,
		newLessonPlan.ID,
		stageBootstrap.StageOutputs,
	); err != nil {
		return nil, err
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
		sourceDomain,
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
	newLessonPlan.CurrentStage = storedCurrentStage
	newLessonPlan.StageConfig = storedStageConfig
	return newLessonPlan, nil
}
