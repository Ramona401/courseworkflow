package repository

// lesson_plan_explicit_create_repo.go
//
// 普通教案创建必须显式写入教育域。
// 精确课程大纲选择与教案在同一INSERT事务中落库：
//   - course_outline_id为空时保持未挂载；
//   - 非空时数据库触发器校验active、教育域、学科和具体年级；
//   - 触发器同时固化出版社、册次和学制快照；
//   - Repository在提交前复核数据库返回的精确ID和快照完整性。

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrLessonPlanExplicitEducationDomainRequired = errors.New(
		"普通教案创建必须显式提供具体教学教育域",
	)

	ErrLessonPlanExplicitEducationDomainSnapshotMismatch = errors.New(
		"教案数据库教育域快照与解析结果不一致",
	)

	ErrLessonPlanExactCourseOutlineSnapshotMismatch = errors.New(
		"教案数据库精确课程大纲快照与请求不一致",
	)
)

const ActionLessonPlanCreate = "lesson_plan.create"

func init() {
	actionNameMap[ActionLessonPlanCreate] = "创建教案"
}

func normalizeLessonPlanExplicitEducationDomain(
	educationDomain string,
) (string, error) {
	domain := strings.ToLower(
		strings.TrimSpace(educationDomain),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return "", fmt.Errorf(
			"%w: %q",
			ErrLessonPlanExplicitEducationDomainRequired,
			educationDomain,
		)
	}

	return domain, nil
}

// CreateLessonPlanWithEducationDomain 创建教案并显式固化教育域与精确大纲。
func CreateLessonPlanWithEducationDomain(
	ctx context.Context,
	lp *models.LessonPlan,
	educationDomain string,
) error {
	if lp == nil {
		return fmt.Errorf("创建教案失败: 教案对象为空")
	}

	domain, err := normalizeLessonPlanExplicitEducationDomain(
		educationDomain,
	)
	if err != nil {
		return err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始普通教案创建事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	query := `
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
			course_outline_id,
			education_domain
		)
		VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,
			$16,$17,$18,$19,$20
		)
		RETURNING
			id,
			education_domain,
			course_outline_id::text,
			course_outline_publisher,
			course_outline_volume,
			school_system,
			created_at,
			updated_at
	`

	duration := lp.DurationMinutes
	if duration <= 0 {
		duration = 45
	}

	contentStructured := lp.ContentStructured
	if contentStructured == "" {
		contentStructured = "{}"
	}

	generationConfig := lp.GenerationConfig
	if generationConfig == "" {
		generationConfig = "{}"
	}

	matchedComponents := lp.MatchedComponents
	if matchedComponents == "" {
		matchedComponents = "[]"
	}

	conversationLog := lp.ConversationLog
	if conversationLog == "" {
		conversationLog = "[]"
	}

	status := lp.Status
	if status == "" {
		status = models.LPStatusDraft
	}

	visibility := lp.Visibility
	if visibility == "" {
		visibility = models.LPVisibilityPersonal
	}

	textbookPageIDs := lp.TextbookPageIDs
	if textbookPageIDs == "" {
		textbookPageIDs = "[]"
	}

	requestedOutlineID := ""
	var outlineIDValue interface{}
	if lp.CourseOutlineID != nil {
		requestedOutlineID = strings.TrimSpace(
			*lp.CourseOutlineID,
		)
		if requestedOutlineID != "" {
			outlineIDValue = requestedOutlineID
		}
	}

	var (
		storedDomain    string
		storedOutlineID sql.NullString
		storedPublisher sql.NullString
		storedVolume    sql.NullString
		storedSystem    sql.NullString
	)

	err = tx.QueryRow(
		ctx,
		query,
		lp.Title,
		lp.Subject,
		lp.Grade,
		lp.Topic,
		duration,
		lp.ContentMarkdown,
		contentStructured,
		generationConfig,
		matchedComponents,
		conversationLog,
		status,
		visibility,
		lp.AuthorID,
		lp.GroupID,
		lp.SchoolID,
		lp.TemplateID,
		lp.RecipeID,
		textbookPageIDs,
		outlineIDValue,
		domain,
	).Scan(
		&lp.ID,
		&storedDomain,
		&storedOutlineID,
		&storedPublisher,
		&storedVolume,
		&storedSystem,
		&lp.CreatedAt,
		&lp.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"显式教育域创建教案失败: %w",
			err,
		)
	}

	storedDomain = strings.ToLower(
		strings.TrimSpace(storedDomain),
	)
	if storedDomain != domain ||
		!models.IsTeachingEducationDomain(storedDomain) {
		return fmt.Errorf(
			"%w: service=%s database=%s",
			ErrLessonPlanExplicitEducationDomainSnapshotMismatch,
			domain,
			storedDomain,
		)
	}

	if requestedOutlineID == "" {
		if storedOutlineID.Valid ||
			storedVolume.Valid ||
			storedSystem.Valid {
			return ErrLessonPlanExactCourseOutlineSnapshotMismatch
		}
	} else {
		if !storedOutlineID.Valid ||
			strings.TrimSpace(storedOutlineID.String) !=
				requestedOutlineID ||
			!storedPublisher.Valid ||
			!storedVolume.Valid ||
			strings.TrimSpace(storedVolume.String) == "" ||
			!storedSystem.Valid ||
			!models.IsValidCourseOutlineSchoolSystem(
				strings.TrimSpace(storedSystem.String),
			) {
			return ErrLessonPlanExactCourseOutlineSnapshotMismatch
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交普通教案创建事务失败: %w",
			err,
		)
	}

	lp.DurationMinutes = duration
	lp.Status = status
	lp.Visibility = visibility
	lp.ContentStructured = contentStructured
	lp.GenerationConfig = generationConfig
	lp.MatchedComponents = matchedComponents
	lp.ConversationLog = conversationLog
	lp.TextbookPageIDs = textbookPageIDs
	lp.EducationDomain = storedDomain

	lp.CourseOutlineID = nil
	lp.CourseOutlinePublisher = nil
	lp.CourseOutlineVolume = nil
	lp.SchoolSystem = nil

	if storedOutlineID.Valid {
		value := strings.TrimSpace(storedOutlineID.String)
		lp.CourseOutlineID = &value
	}
	if storedPublisher.Valid {
		value := strings.TrimSpace(storedPublisher.String)
		lp.CourseOutlinePublisher = &value
	}
	if storedVolume.Valid {
		value := strings.TrimSpace(storedVolume.String)
		lp.CourseOutlineVolume = &value
	}
	if storedSystem.Valid {
		value := strings.TrimSpace(storedSystem.String)
		lp.SchoolSystem = &value
	}

	return nil
}
