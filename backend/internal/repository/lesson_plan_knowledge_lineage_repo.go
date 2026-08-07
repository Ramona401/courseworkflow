package repository

// lesson_plan_knowledge_lineage_repo.go — 教案知识脉络来源与快照仓储
//
// 并发复核不再依赖workshop_stage_outputs.updated_at：
//   - 阶段完成会更新updated_at，但并不代表分析内容变化；
//   - 正确做法是对conversation_log、structured_output和narrative_output
//     计算统一内容哈希；
//   - generating和active写入前均在事务内重新读取并复核该内容哈希；
//   - 同时重新复核课程大纲正文哈希和当前阶段；
//   - 教案、大纲状态和分析产出一次查询形成同一来源快照，避免重复查询漂移。

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrLessonPlanKnowledgeLineageNotFound                  = errors.New("教案知识脉络快照不存在")
	ErrLessonPlanKnowledgeLineageNoCourseOutline           = errors.New("教案没有绑定课程大纲")
	ErrLessonPlanKnowledgeLineageSourceUnavailable         = errors.New("教案知识脉络来源不可用")
	ErrLessonPlanKnowledgeLineageConfirmedStageUnavailable = errors.New("教学分析阶段产出不可用")
	ErrLessonPlanKnowledgeLineageSourceChanged             = errors.New(
		"教案、教学分析结论或课程大纲已经变化，旧知识脉络不能写入",
	)
)

// LessonPlanKnowledgeLineageSource 是一次提取使用的确定性来源快照。
type LessonPlanKnowledgeLineageSource struct {
	LessonPlanID           string
	AuthorID               string
	Subject                string
	Grade                  string
	Topic                  string
	EducationDomain        string
	CurrentStage           string
	ConversationLog        string
	CourseOutlineID        string
	CourseOutlineTitle     string
	CourseOutlineSubject   string
	CourseOutlineGrade     string
	CourseOutlinePublisher string
	CourseOutlineVolume    string
	CourseOutlineContent   string
	CourseOutlineUpdatedAt time.Time
	ConfirmedStageCode     string
	StageStructuredOutput  string
	StageNarrativeOutput   string
	StageOutputUpdatedAt   time.Time
}

// AnalysisSourceHash 返回分析对话与阶段正式产出的统一内容哈希。
func (source *LessonPlanKnowledgeLineageSource) AnalysisSourceHash() string {
	if source == nil {
		return ""
	}

	return CalculateLessonPlanKnowledgeLineageAnalysisSourceHash(
		source.ConversationLog,
		source.StageStructuredOutput,
		source.StageNarrativeOutput,
	)
}

// CalculateLessonPlanKnowledgeLineageAnalysisSourceHash 生成分析来源SHA-256。
//
// 使用不可见分隔符避免三个字符串边界拼接歧义。
func CalculateLessonPlanKnowledgeLineageAnalysisSourceHash(
	conversationLog string,
	structuredOutput string,
	narrativeOutput string,
) string {
	payload := strings.Join(
		[]string{
			strings.TrimSpace(conversationLog),
			strings.TrimSpace(structuredOutput),
			strings.TrimSpace(narrativeOutput),
		},
		"\x1f",
	)

	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

const lessonPlanKnowledgeLineageSelectColumns = `
        id::text,
        lesson_plan_id::text,
        course_outline_id::text,
        status,
        anchor_snapshot::text,
        lineage_snapshot::text,
        context_text,
        anchor_hash,
        outline_hash,
        confirmed_stage_code,
        confirmed_stage_output_updated_at,
        model_used,
        tokens_used,
        error_message,
        generated_at,
        created_at,
        updated_at
`

// lessonPlanKnowledgeLineageSelectColumnsFromLineage 用于包含JOIN的active查询。
//
// lesson_plan_knowledge_lineages、lesson_plans、course_outlines和
// workshop_stage_outputs都包含id或时间戳等同名列。若复用未限定列名的通用
// SELECT清单，PostgreSQL会在执行过滤条件前因列名歧义而拒绝整条查询。
const lessonPlanKnowledgeLineageSelectColumnsFromLineage = `
        lineage.id::text,
        lineage.lesson_plan_id::text,
        lineage.course_outline_id::text,
        lineage.status,
        lineage.anchor_snapshot::text,
        lineage.lineage_snapshot::text,
        lineage.context_text,
        lineage.anchor_hash,
        lineage.outline_hash,
        lineage.confirmed_stage_code,
        lineage.confirmed_stage_output_updated_at,
        lineage.model_used,
        lineage.tokens_used,
        lineage.error_message,
        lineage.generated_at,
        lineage.created_at,
        lineage.updated_at
`

func scanLessonPlanKnowledgeLineage(row pgx.Row) (*models.LessonPlanKnowledgeLineage, error) {
	record := &models.LessonPlanKnowledgeLineage{}

	var confirmedOutputAt, generatedAt, createdAt, updatedAt sql.NullTime

	err := row.Scan(
		&record.ID,
		&record.LessonPlanID,
		&record.CourseOutlineID,
		&record.Status,
		&record.AnchorSnapshot,
		&record.LineageSnapshot,
		&record.ContextText,
		&record.AnchorHash,
		&record.OutlineHash,
		&record.ConfirmedStageCode,
		&confirmedOutputAt,
		&record.ModelUsed,
		&record.TokensUsed,
		&record.ErrorMessage,
		&generatedAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if confirmedOutputAt.Valid {
		value := confirmedOutputAt.Time
		record.ConfirmedStageOutputUpdatedAt = &value
	}
	if generatedAt.Valid {
		value := generatedAt.Time
		record.GeneratedAt = &value
	}
	if createdAt.Valid {
		value := createdAt.Time
		record.CreatedAt = &value
	}
	if updatedAt.Valid {
		value := updatedAt.Time
		record.UpdatedAt = &value
	}

	return record, nil
}

// LoadLessonPlanKnowledgeLineageSource 读取本次提取使用的教案、大纲与分析产出。
func LoadLessonPlanKnowledgeLineageSource(
	ctx context.Context,
	lessonPlanID string,
	confirmedStageCode string,
) (*LessonPlanKnowledgeLineageSource, error) {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	confirmedStageCode = strings.TrimSpace(confirmedStageCode)

	if confirmedStageCode == "" {
		confirmedStageCode = "analyze"
	}

	source := &LessonPlanKnowledgeLineageSource{
		ConfirmedStageCode: confirmedStageCode,
	}

	var (
		outlineStatus    string
		outlineUpdatedAt sql.NullTime
		stageUpdatedAt   sql.NullTime
	)

	err := database.DB.QueryRow(
		ctx,
		`
                        SELECT
                                lp.id::text,
                                lp.author_id::text,
                                COALESCE(lp.subject, ''),
                                COALESCE(lp.grade, ''),
                                COALESCE(lp.topic, ''),
                                COALESCE(lp.education_domain, ''),
                                COALESCE(lp.current_stage, ''),
                                COALESCE(lp.conversation_log::text, '[]'),
                                COALESCE(lp.course_outline_id::text, ''),
                                COALESCE(outline.title, ''),
                                COALESCE(outline.subject, ''),
                                COALESCE(outline.grade, ''),
                                COALESCE(outline.publisher, ''),
                                COALESCE(outline.volume, ''),
                                COALESCE(outline.content, ''),
                                COALESCE(outline.status, ''),
                                outline.updated_at,
                                COALESCE(stage_output.structured_output::text, '{}'),
                                COALESCE(stage_output.narrative_output, ''),
                                stage_output.updated_at
                        FROM lesson_plans lp
                        LEFT JOIN course_outlines outline
                          ON outline.id = lp.course_outline_id
                        LEFT JOIN workshop_stage_outputs stage_output
                          ON stage_output.lesson_plan_id = lp.id
                         AND stage_output.stage_code = $2
                        WHERE lp.id = $1
                          AND lp.deleted_at IS NULL
                `,
		lessonPlanID,
		confirmedStageCode,
	).Scan(
		&source.LessonPlanID,
		&source.AuthorID,
		&source.Subject,
		&source.Grade,
		&source.Topic,
		&source.EducationDomain,
		&source.CurrentStage,
		&source.ConversationLog,
		&source.CourseOutlineID,
		&source.CourseOutlineTitle,
		&source.CourseOutlineSubject,
		&source.CourseOutlineGrade,
		&source.CourseOutlinePublisher,
		&source.CourseOutlineVolume,
		&source.CourseOutlineContent,
		&outlineStatus,
		&outlineUpdatedAt,
		&source.StageStructuredOutput,
		&source.StageNarrativeOutput,
		&stageUpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanNotFound
		}

		return nil, fmt.Errorf("读取教案知识脉络来源失败: %w", err)
	}

	source.CourseOutlineID = strings.TrimSpace(source.CourseOutlineID)

	if source.CourseOutlineID == "" {
		return nil, ErrLessonPlanKnowledgeLineageNoCourseOutline
	}

	if strings.TrimSpace(outlineStatus) != "active" ||
		strings.TrimSpace(source.CourseOutlineContent) == "" ||
		!outlineUpdatedAt.Valid {
		return nil, ErrLessonPlanKnowledgeLineageSourceUnavailable
	}

	if strings.TrimSpace(source.CurrentStage) != confirmedStageCode ||
		!stageUpdatedAt.Valid {
		return nil, ErrLessonPlanKnowledgeLineageConfirmedStageUnavailable
	}

	source.CourseOutlineUpdatedAt = outlineUpdatedAt.Time
	source.StageOutputUpdatedAt = stageUpdatedAt.Time

	return source, nil
}

// GetLessonPlanKnowledgeLineage 读取教案当前知识脉络记录。
func GetLessonPlanKnowledgeLineage(
	ctx context.Context,
	lessonPlanID string,
) (*models.LessonPlanKnowledgeLineage, error) {
	record, err := scanLessonPlanKnowledgeLineage(
		database.DB.QueryRow(
			ctx,
			`SELECT `+lessonPlanKnowledgeLineageSelectColumns+`
                         FROM lesson_plan_knowledge_lineages
                         WHERE lesson_plan_id = $1`,
			strings.TrimSpace(lessonPlanID),
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取教案知识脉络快照失败: %w", err)
	}

	return record, nil
}

// GetActiveLessonPlanKnowledgeLineage 返回仍与正式大纲绑定一致的active快照。
//
// 不再比较stage_output.updated_at，因为阶段完成状态变化也会更新该时间。
// 分析内容发生变化由数据库触发器标记stale。
func GetActiveLessonPlanKnowledgeLineage(
	ctx context.Context,
	lessonPlanID string,
) (*models.LessonPlanKnowledgeLineage, error) {
	record, err := scanLessonPlanKnowledgeLineage(
		database.DB.QueryRow(
			ctx,
			`SELECT `+lessonPlanKnowledgeLineageSelectColumnsFromLineage+`
                         FROM lesson_plan_knowledge_lineages lineage
                         JOIN lesson_plans lp
                           ON lp.id = lineage.lesson_plan_id
                          AND lp.deleted_at IS NULL
                          AND lp.course_outline_id = lineage.course_outline_id
                         JOIN course_outlines outline
                           ON outline.id = lineage.course_outline_id
                          AND outline.status = 'active'
                         JOIN workshop_stage_outputs stage_output
                           ON stage_output.lesson_plan_id = lineage.lesson_plan_id
                          AND stage_output.stage_code = lineage.confirmed_stage_code
                         WHERE lineage.lesson_plan_id = $1
                           AND lineage.status = 'active'`,
			strings.TrimSpace(lessonPlanID),
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取可用教案知识脉络快照失败: %w", err)
	}

	return record, nil
}

// validateLessonPlanKnowledgeLineageSourceTx 在写入事务内重新复核来源内容。
func validateLessonPlanKnowledgeLineageSourceTx(
	ctx context.Context,
	tx pgx.Tx,
	lessonPlanID string,
	courseOutlineID string,
	outlineHash string,
	analysisSourceHash string,
	confirmedStageCode string,
) error {
	var currentOutlineContent, currentStage, currentConversationLog string
	var currentStructuredOutput, currentNarrativeOutput string

	err := tx.QueryRow(
		ctx,
		`
                        SELECT
                                outline.content,
                                lp.current_stage,
                                COALESCE(lp.conversation_log::text, '[]'),
                                COALESCE(stage_output.structured_output::text, '{}'),
                                COALESCE(stage_output.narrative_output, '')
                        FROM lesson_plans lp
                        JOIN course_outlines outline
                          ON outline.id = lp.course_outline_id
                         AND outline.status = 'active'
                        JOIN workshop_stage_outputs stage_output
                          ON stage_output.lesson_plan_id = lp.id
                         AND stage_output.stage_code = $3
                        WHERE lp.id = $1
                          AND lp.deleted_at IS NULL
                          AND lp.course_outline_id = $2::uuid
                        FOR SHARE OF lp, outline, stage_output
                `,
		strings.TrimSpace(lessonPlanID),
		strings.TrimSpace(courseOutlineID),
		strings.TrimSpace(confirmedStageCode),
	).Scan(
		&currentOutlineContent,
		&currentStage,
		&currentConversationLog,
		&currentStructuredOutput,
		&currentNarrativeOutput,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLessonPlanKnowledgeLineageSourceChanged
		}

		return fmt.Errorf("复核教案知识脉络来源失败: %w", err)
	}

	if strings.TrimSpace(currentStage) != strings.TrimSpace(confirmedStageCode) {
		return ErrLessonPlanKnowledgeLineageSourceChanged
	}

	outlineSum := sha256.Sum256([]byte(currentOutlineContent))
	currentOutlineHash := hex.EncodeToString(outlineSum[:])

	if currentOutlineHash != strings.TrimSpace(outlineHash) {
		return ErrLessonPlanKnowledgeLineageSourceChanged
	}

	currentAnalysisHash := CalculateLessonPlanKnowledgeLineageAnalysisSourceHash(
		currentConversationLog,
		currentStructuredOutput,
		currentNarrativeOutput,
	)

	if currentAnalysisHash != strings.TrimSpace(analysisSourceHash) {
		return ErrLessonPlanKnowledgeLineageSourceChanged
	}

	return nil
}

// UpsertGeneratingLessonPlanKnowledgeLineage 在AI读取大纲前写入generating占位。
func UpsertGeneratingLessonPlanKnowledgeLineage(
	ctx context.Context,
	lessonPlanID string,
	courseOutlineID string,
	anchorSnapshot string,
	anchorHash string,
	outlineHash string,
	analysisSourceHash string,
	confirmedStageCode string,
	confirmedStageOutputUpdatedAt *time.Time,
) error {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	courseOutlineID = strings.TrimSpace(courseOutlineID)
	anchorSnapshot = strings.TrimSpace(anchorSnapshot)
	anchorHash = strings.TrimSpace(anchorHash)
	outlineHash = strings.TrimSpace(outlineHash)
	analysisSourceHash = strings.TrimSpace(analysisSourceHash)
	confirmedStageCode = strings.TrimSpace(confirmedStageCode)

	if confirmedStageCode == "" {
		confirmedStageCode = "analyze"
	}

	if lessonPlanID == "" || courseOutlineID == "" ||
		anchorSnapshot == "" || anchorSnapshot == "{}" ||
		len(anchorHash) != 64 || len(outlineHash) != 64 ||
		len(analysisSourceHash) != 64 ||
		confirmedStageOutputUpdatedAt == nil {
		return fmt.Errorf("写入知识脉络生成占位失败: 输入不完整")
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始知识脉络生成占位事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := validateLessonPlanKnowledgeLineageSourceTx(
		ctx,
		tx,
		lessonPlanID,
		courseOutlineID,
		outlineHash,
		analysisSourceHash,
		confirmedStageCode,
	); err != nil {
		return err
	}

	_, err = tx.Exec(
		ctx,
		`
                        INSERT INTO lesson_plan_knowledge_lineages (
                                id, lesson_plan_id, course_outline_id, status,
                                anchor_snapshot, lineage_snapshot, context_text,
                                anchor_hash, outline_hash, confirmed_stage_code,
                                confirmed_stage_output_updated_at,
                                model_used, tokens_used, error_message,
                                generated_at, updated_at
                        )
                        VALUES (
                                gen_random_uuid(), $1, $2, 'generating',
                                $3::jsonb, '{}'::jsonb, '',
                                $4, $5, $6, $7,
                                '', 0, '', NULL, NOW()
                        )
                        ON CONFLICT (lesson_plan_id)
                        DO UPDATE SET
                                course_outline_id = EXCLUDED.course_outline_id,
                                status = 'generating',
                                anchor_snapshot = EXCLUDED.anchor_snapshot,
                                lineage_snapshot = '{}'::jsonb,
                                context_text = '',
                                anchor_hash = EXCLUDED.anchor_hash,
                                outline_hash = EXCLUDED.outline_hash,
                                confirmed_stage_code = EXCLUDED.confirmed_stage_code,
                                confirmed_stage_output_updated_at =
                                        EXCLUDED.confirmed_stage_output_updated_at,
                                model_used = '',
                                tokens_used = 0,
                                error_message = '',
                                generated_at = NULL,
                                updated_at = NOW()
                `,
		lessonPlanID,
		courseOutlineID,
		anchorSnapshot,
		anchorHash,
		outlineHash,
		confirmedStageCode,
		confirmedStageOutputUpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("写入知识脉络生成占位失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交知识脉络生成占位事务失败: %w", err)
	}

	return nil
}

// UpsertActiveLessonPlanKnowledgeLineage 写入最终active快照。
func UpsertActiveLessonPlanKnowledgeLineage(
	ctx context.Context,
	lessonPlanID string,
	courseOutlineID string,
	anchorSnapshot string,
	lineageSnapshot string,
	contextText string,
	anchorHash string,
	outlineHash string,
	analysisSourceHash string,
	confirmedStageCode string,
	confirmedStageOutputUpdatedAt *time.Time,
	modelUsed string,
	tokensUsed int,
) error {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	courseOutlineID = strings.TrimSpace(courseOutlineID)
	anchorSnapshot = strings.TrimSpace(anchorSnapshot)
	lineageSnapshot = strings.TrimSpace(lineageSnapshot)
	contextText = strings.TrimSpace(contextText)
	anchorHash = strings.TrimSpace(anchorHash)
	outlineHash = strings.TrimSpace(outlineHash)
	analysisSourceHash = strings.TrimSpace(analysisSourceHash)
	confirmedStageCode = strings.TrimSpace(confirmedStageCode)
	modelUsed = strings.TrimSpace(modelUsed)

	if confirmedStageCode == "" {
		confirmedStageCode = "analyze"
	}

	if lessonPlanID == "" || courseOutlineID == "" ||
		anchorSnapshot == "" || anchorSnapshot == "{}" ||
		lineageSnapshot == "" || lineageSnapshot == "{}" ||
		contextText == "" || len(anchorHash) != 64 ||
		len(outlineHash) != 64 || len(analysisSourceHash) != 64 ||
		confirmedStageOutputUpdatedAt == nil || tokensUsed < 0 {
		return fmt.Errorf("写入active知识脉络失败: 结果不完整")
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始active知识脉络写入事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := validateLessonPlanKnowledgeLineageSourceTx(
		ctx,
		tx,
		lessonPlanID,
		courseOutlineID,
		outlineHash,
		analysisSourceHash,
		confirmedStageCode,
	); err != nil {
		return err
	}

	_, err = tx.Exec(
		ctx,
		`
                        INSERT INTO lesson_plan_knowledge_lineages (
                                id, lesson_plan_id, course_outline_id, status,
                                anchor_snapshot, lineage_snapshot, context_text,
                                anchor_hash, outline_hash, confirmed_stage_code,
                                confirmed_stage_output_updated_at,
                                model_used, tokens_used, error_message,
                                generated_at, updated_at
                        )
                        VALUES (
                                gen_random_uuid(), $1, $2, 'active',
                                $3::jsonb, $4::jsonb, $5,
                                $6, $7, $8, $9,
                                $10, $11, '', NOW(), NOW()
                        )
                        ON CONFLICT (lesson_plan_id)
                        DO UPDATE SET
                                course_outline_id = EXCLUDED.course_outline_id,
                                status = 'active',
                                anchor_snapshot = EXCLUDED.anchor_snapshot,
                                lineage_snapshot = EXCLUDED.lineage_snapshot,
                                context_text = EXCLUDED.context_text,
                                anchor_hash = EXCLUDED.anchor_hash,
                                outline_hash = EXCLUDED.outline_hash,
                                confirmed_stage_code = EXCLUDED.confirmed_stage_code,
                                confirmed_stage_output_updated_at =
                                        EXCLUDED.confirmed_stage_output_updated_at,
                                model_used = EXCLUDED.model_used,
                                tokens_used = EXCLUDED.tokens_used,
                                error_message = '',
                                generated_at = NOW(),
                                updated_at = NOW()
                `,
		lessonPlanID,
		courseOutlineID,
		anchorSnapshot,
		lineageSnapshot,
		contextText,
		anchorHash,
		outlineHash,
		confirmedStageCode,
		confirmedStageOutputUpdatedAt,
		modelUsed,
		tokensUsed,
	)
	if err != nil {
		return fmt.Errorf("写入active知识脉络失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交active知识脉络写入事务失败: %w", err)
	}

	return nil
}

// MarkLessonPlanKnowledgeLineageFailed 只把仍为generating的记录标记为failed。
func MarkLessonPlanKnowledgeLineageFailed(
	ctx context.Context,
	lessonPlanID string,
	errorMessage string,
) error {
	errorMessage = strings.TrimSpace(errorMessage)

	if len([]rune(errorMessage)) > 2000 {
		errorMessage = string([]rune(errorMessage)[:2000]) + "...(截断)"
	}

	_, err := database.DB.Exec(
		ctx,
		`
                        UPDATE lesson_plan_knowledge_lineages
                        SET
                                status = 'failed',
                                lineage_snapshot = '{}'::jsonb,
                                context_text = '',
                                model_used = '',
                                tokens_used = 0,
                                error_message = $2,
                                generated_at = NULL,
                                updated_at = NOW()
                        WHERE lesson_plan_id = $1
                          AND status = 'generating'
                `,
		strings.TrimSpace(lessonPlanID),
		errorMessage,
	)
	if err != nil {
		return fmt.Errorf("标记知识脉络提取失败: %w", err)
	}

	return nil
}

// MarkLessonPlanKnowledgeLineageStale 主动使快照失效。
func MarkLessonPlanKnowledgeLineageStale(
	ctx context.Context,
	lessonPlanID string,
	reason string,
) error {
	reason = strings.TrimSpace(reason)

	if len([]rune(reason)) > 2000 {
		reason = string([]rune(reason)[:2000]) + "...(截断)"
	}

	_, err := database.DB.Exec(
		ctx,
		`
                        UPDATE lesson_plan_knowledge_lineages
                        SET status = 'stale', error_message = $2, updated_at = NOW()
                        WHERE lesson_plan_id = $1
                `,
		strings.TrimSpace(lessonPlanID),
		reason,
	)
	if err != nil {
		return fmt.Errorf("标记知识脉络失效失败: %w", err)
	}

	return nil
}

// DeleteLessonPlanKnowledgeLineage 删除教案知识脉络快照。
func DeleteLessonPlanKnowledgeLineage(
	ctx context.Context,
	lessonPlanID string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`DELETE FROM lesson_plan_knowledge_lineages WHERE lesson_plan_id = $1`,
		strings.TrimSpace(lessonPlanID),
	)
	if err != nil {
		return fmt.Errorf("删除教案知识脉络快照失败: %w", err)
	}

	return nil
}
