package repository

// lesson_plan_explicit_create_repo.go
//
// 本文件只承载“普通教案创建”显式写入教育域的Repository方法。
//
// 为什么不直接修改既有CreateLessonPlan：
//   - 既有方法仍被对话备课、导入和Fork调用；
//   - 上述入口分别属于后续上下文11、12、13；
//   - 本上下文不能提前改变它们的创建语义。
//
// 因此本文件新增CreateLessonPlanWithEducationDomain：
//   - Service必须把上下文9解析出的具体教学域作为独立参数传入；
//   - SQL明确写入lesson_plans.education_domain；
//   - 只接受k12、vocational和adult；
//   - 使用事务验证数据库RETURNING快照与传入值完全一致；
//   - 快照不一致时回滚，绝不留下半成品；
//   - 原CreateLessonPlan继续保留给尚未迁移的创建入口。
//
// 数据库现有BEFORE INSERT触发器在收到合法显式域时会直接RETURN NEW，
// 因此无需修改触发器，也不再依赖其作者归属推导或K12默认回退。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	// ErrLessonPlanExplicitEducationDomainRequired 表示显式创建参数不是具体教学域。
	ErrLessonPlanExplicitEducationDomainRequired = errors.New(
		"普通教案创建必须显式提供具体教学教育域",
	)

	// ErrLessonPlanExplicitEducationDomainSnapshotMismatch
	// 表示数据库最终快照与Service解析结果不一致。
	//
	// 该错误在事务提交前返回，因此INSERT会被回滚，不会留下错误快照。
	ErrLessonPlanExplicitEducationDomainSnapshotMismatch = errors.New(
		"教案数据库教育域快照与解析结果不一致",
	)
)

// ActionLessonPlanCreate 是普通教案创建成功的审计动作。
const ActionLessonPlanCreate = "lesson_plan.create"

// init 将本上下文新增的审计动作登记为中文名称。
//
// actionNameMap定义在audit_repo.go中；同属repository包，
// 在所有包变量完成初始化后执行本init函数是安全的。
func init() {
	actionNameMap[ActionLessonPlanCreate] = "创建教案"
}

// normalizeLessonPlanExplicitEducationDomain
// 规范化并严格校验普通教案显式写入域。
//
// 返回值只可能是k12、vocational或adult。
// 不调用NormalizeEducationDomain，避免非法值回退K12。
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

// CreateLessonPlanWithEducationDomain
// 创建普通教案并显式写入Service解析出的教育域。
//
// educationDomain是独立参数，不信任lp中可能被其它调用方构造的字段值。
//
// 原子性规则：
//  1. 开启事务；
//  2. INSERT显式写入education_domain；
//  3. 读取RETURNING中的数据库最终快照；
//  4. 最终快照必须与传入域完全一致；
//  5. 一致才提交，否则回滚。
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
			education_domain
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
			$14,
			$15,
			$16,
			$17,
			$18,
			$19
		)
		RETURNING
			id,
			education_domain,
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

	storedDomain := ""

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
		domain,
	).Scan(
		&lp.ID,
		&storedDomain,
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

	return nil
}
