package repository

// lesson_plan_import_create_repo.go
//
// 本文件只承载“已有教案导入”创建后的核心工作流固化与失败补偿。
//
// 导入正文已经由CreateLessonPlanWithEducationDomain事务显式写入。
// 本文件随后以独立事务原子完成：
//   - 写入stage_config；
//   - 设置current_stage；
//   - 创建review之前的skipped阶段记录；
//   - 创建review阶段in_progress记录；
//   - 追加导入成功开场消息。
//
// 任一步失败时事务整体回滚，Service再调用
// DeleteIncompleteImportedLessonPlanCreation硬清理刚创建的导入教案。
//
// 普通用户删除教案仍走软删除和回收站；
// 本文件的硬删除只用于同一请求内尚未完成的失败导入补偿。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	// ErrImportedLessonPlanFinalizationRejected
	// 表示目标教案不符合导入创建固化的安全条件。
	ErrImportedLessonPlanFinalizationRejected = errors.New(
		"导入教案工作流固化被安全条件拒绝",
	)

	// ErrIncompleteImportedLessonPlanCleanupRejected
	// 表示目标教案不符合失败导入硬清理条件。
	ErrIncompleteImportedLessonPlanCleanupRejected = errors.New(
		"失败导入教案补偿清理被安全条件拒绝",
	)
)

// importedStageCompletedAt 为导入时跳过的前置阶段生成完成时间。
//
// completed_at直接作为独立参数传给SQL，避免同一个PostgreSQL参数
// 同时被推断为状态列类型和CASE比较类型。
func importedStageCompletedAt(
	status string,
	now time.Time,
) *time.Time {
	if status !=
		string(models.StageOutputSkipped) {
		return nil
	}

	value := now
	return &value
}

// isEmptyImportedAIReviewJSON 判断新建教案是否尚无AI评审结果。
//
// ai_review_result是JSON/JSONB列，不能与空字符串直接COALESCE；
// 这里兼容SQL NULL、JSON null和空对象三种未评审状态。
func isEmptyImportedAIReviewJSON(
	raw string,
) bool {
	normalized := strings.TrimSpace(
		raw,
	)
	if normalized == "" {
		return true
	}

	var value any
	if err := json.Unmarshal(
		[]byte(normalized),
		&value,
	); err != nil {
		return false
	}

	switch typed := value.(type) {
	case nil:
		return true
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}

// FinalizeImportedLessonPlanCreation
// 原子固化导入教案的阶段状态和开场消息。
func FinalizeImportedLessonPlanCreation(
	ctx context.Context,
	lessonPlanID string,
	authorID string,
	educationDomain string,
	stageConfigJSON string,
	currentStage string,
	stageOutputs []models.WorkshopStageOutput,
	openingMessage *models.ConversationMessage,
) error {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	authorID = strings.TrimSpace(authorID)
	currentStage = strings.TrimSpace(currentStage)

	if lessonPlanID == "" ||
		authorID == "" ||
		currentStage == "" {
		return fmt.Errorf(
			"%w: 教案ID、作者ID或当前阶段为空",
			ErrImportedLessonPlanFinalizationRejected,
		)
	}

	domain, err :=
		normalizeLessonPlanExplicitEducationDomain(
			educationDomain,
		)
	if err != nil {
		return err
	}

	if !json.Valid([]byte(stageConfigJSON)) {
		return fmt.Errorf(
			"%w: stage_config不是合法JSON",
			ErrImportedLessonPlanFinalizationRejected,
		)
	}

	var stageConfigItems []json.RawMessage
	if err := json.Unmarshal(
		[]byte(stageConfigJSON),
		&stageConfigItems,
	); err != nil ||
		len(stageConfigItems) == 0 {
		return fmt.Errorf(
			"%w: stage_config为空或不是阶段数组",
			ErrImportedLessonPlanFinalizationRejected,
		)
	}

	if openingMessage == nil {
		return fmt.Errorf(
			"%w: 开场消息为空",
			ErrImportedLessonPlanFinalizationRejected,
		)
	}

	openingJSON, err := json.Marshal(
		openingMessage,
	)
	if err != nil {
		return fmt.Errorf(
			"序列化导入开场消息失败: %w",
			err,
		)
	}

	if len(stageOutputs) == 0 {
		return fmt.Errorf(
			"%w: 阶段产出列表为空",
			ErrImportedLessonPlanFinalizationRejected,
		)
	}

	currentStageCount := 0
	for index := range stageOutputs {
		output := &stageOutputs[index]

		if strings.TrimSpace(
			output.StageCode,
		) == "" {
			return fmt.Errorf(
				"%w: 存在空阶段代码",
				ErrImportedLessonPlanFinalizationRejected,
			)
		}

		switch output.Status {
		case models.StageOutputSkipped:
			// review之前的阶段。
		case models.StageOutputInProgress:
			if output.StageCode != currentStage {
				return fmt.Errorf(
					"%w: 非当前阶段被标记为进行中",
					ErrImportedLessonPlanFinalizationRejected,
				)
			}
			currentStageCount++
		default:
			return fmt.Errorf(
				"%w: 非法阶段状态%s",
				ErrImportedLessonPlanFinalizationRejected,
				output.Status,
			)
		}
	}

	if currentStageCount != 1 {
		return fmt.Errorf(
			"%w: 当前阶段进行中记录数量为%d",
			ErrImportedLessonPlanFinalizationRejected,
			currentStageCount,
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始导入教案固化事务失败: %w",
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
		storedDomain    string
		conversationLog string
	)

	err = tx.QueryRow(ctx, `
		SELECT
			author_id::text,
			status,
			visibility,
			COALESCE(content_markdown, ''),
			COALESCE(education_domain, ''),
			COALESCE(conversation_log::text, '[]')
		FROM lesson_plans
		WHERE id = $1
		  AND deleted_at IS NULL
		FOR UPDATE
	`, lessonPlanID).Scan(
		&storedAuthorID,
		&status,
		&visibility,
		&contentMarkdown,
		&storedDomain,
		&conversationLog,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLessonPlanNotFound
		}
		return fmt.Errorf(
			"锁定待固化导入教案失败: %w",
			err,
		)
	}

	storedDomain = strings.ToLower(
		strings.TrimSpace(storedDomain),
	)

	if strings.TrimSpace(storedAuthorID) != authorID ||
		status != models.LPStatusDraft ||
		visibility != models.LPVisibilityPersonal ||
		strings.TrimSpace(contentMarkdown) == "" ||
		storedDomain != domain ||
		!isEmptyConversationLogJSON(conversationLog) {
		return fmt.Errorf(
			"%w: plan_id=%s status=%s visibility=%s domain=%s",
			ErrImportedLessonPlanFinalizationRejected,
			lessonPlanID,
			status,
			visibility,
			storedDomain,
		)
	}

	result, err := tx.Exec(ctx, `
		UPDATE lesson_plans
		SET
			stage_config = $1::jsonb,
			current_stage = $2,
			conversation_log =
				COALESCE(
					conversation_log,
					'[]'::jsonb
				)
				|| jsonb_build_array(
					$3::jsonb
				),
			updated_at = NOW()
		WHERE id = $4
		  AND author_id = $5
		  AND status = $6
		  AND visibility = $7
		  AND deleted_at IS NULL
	`,
		stageConfigJSON,
		currentStage,
		string(openingJSON),
		lessonPlanID,
		authorID,
		models.LPStatusDraft,
		models.LPVisibilityPersonal,
	)
	if err != nil {
		return fmt.Errorf(
			"写入导入教案阶段和开场消息失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf(
			"%w: 更新时目标状态发生变化",
			ErrImportedLessonPlanFinalizationRejected,
		)
	}

	for index := range stageOutputs {
		output := stageOutputs[index]

		structuredOutput :=
			output.StructuredOutput
		if structuredOutput == "" {
			structuredOutput = "{}"
		}

		conversationSnapshot :=
			output.ConversationSnapshot
		if conversationSnapshot == "" {
			conversationSnapshot = "[]"
		}

		completedAt := importedStageCompletedAt(
			string(output.Status),
			time.Now(),
		)

		_, err = tx.Exec(ctx, `
			INSERT INTO workshop_stage_outputs (
				lesson_plan_id,
				stage_code,
				stage_order,
				structured_output,
				narrative_output,
				conversation_snapshot,
				model_used,
				tokens_used,
				status,
				completed_at
			)
			VALUES (
				$1,
				$2,
				$3,
				$4::jsonb,
				$5,
				$6::jsonb,
				$7,
				$8,
				$9,
				$10
			)
		`,
			lessonPlanID,
			output.StageCode,
			output.StageOrder,
			structuredOutput,
			output.NarrativeOutput,
			conversationSnapshot,
			output.ModelUsed,
			output.TokensUsed,
			output.Status,
			completedAt,
		)
		if err != nil {
			return fmt.Errorf(
				"创建导入教案阶段记录失败（%s）: %w",
				output.StageCode,
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交导入教案固化事务失败: %w",
			err,
		)
	}

	return nil
}

// DeleteIncompleteImportedLessonPlanCreation
// 硬清理同一请求内尚未完成的失败导入教案。
//
// 幂等规则：目标教案已经不存在时视为清理完成。
//
// 安全条件：
//   - 作者、教育域、状态和可见范围与本次创建一致；
//   - 导入正文非空；
//   - 对话仍为空；
//   - 尚无AI评审结果。
func DeleteIncompleteImportedLessonPlanCreation(
	ctx context.Context,
	lessonPlanID string,
	authorID string,
	educationDomain string,
) error {
	lessonPlanID = strings.TrimSpace(
		lessonPlanID,
	)
	authorID = strings.TrimSpace(authorID)

	if lessonPlanID == "" || authorID == "" {
		return fmt.Errorf(
			"%w: 教案ID或作者ID为空",
			ErrIncompleteImportedLessonPlanCleanupRejected,
		)
	}

	domain, err :=
		normalizeLessonPlanExplicitEducationDomain(
			educationDomain,
		)
	if err != nil {
		return err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始失败导入补偿事务失败: %w",
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
		storedDomain    string
		conversationLog string
		aiReviewResult  string
	)

	err = tx.QueryRow(ctx, `
		SELECT
			author_id::text,
			status,
			visibility,
			COALESCE(content_markdown, ''),
			COALESCE(education_domain, ''),
			COALESCE(conversation_log::text, '[]'),
			COALESCE(ai_review_result::text, '')
		FROM lesson_plans
		WHERE id = $1
		FOR UPDATE
	`, lessonPlanID).Scan(
		&storedAuthorID,
		&status,
		&visibility,
		&contentMarkdown,
		&storedDomain,
		&conversationLog,
		&aiReviewResult,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return tx.Commit(ctx)
		}
		return fmt.Errorf(
			"锁定失败导入教案失败: %w",
			err,
		)
	}

	storedDomain = strings.ToLower(
		strings.TrimSpace(storedDomain),
	)

	if strings.TrimSpace(storedAuthorID) != authorID ||
		status != models.LPStatusDraft ||
		visibility != models.LPVisibilityPersonal ||
		strings.TrimSpace(contentMarkdown) == "" ||
		storedDomain != domain ||
		!isEmptyConversationLogJSON(conversationLog) ||
		!isEmptyImportedAIReviewJSON(aiReviewResult) {
		return fmt.Errorf(
			"%w: plan_id=%s status=%s visibility=%s domain=%s",
			ErrIncompleteImportedLessonPlanCleanupRejected,
			lessonPlanID,
			status,
			visibility,
			storedDomain,
		)
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM workshop_stage_outputs
		WHERE lesson_plan_id = $1
	`, lessonPlanID); err != nil {
		return fmt.Errorf(
			"清理失败导入阶段记录失败: %w",
			err,
		)
	}

	result, err := tx.Exec(ctx, `
		DELETE FROM lesson_plans
		WHERE id = $1
		  AND author_id = $2
		  AND status = $3
		  AND visibility = $4
		  AND education_domain = $5
		  AND COALESCE(content_markdown, '') <> ''
		  AND COALESCE(conversation_log::text, '[]') = '[]'
		  AND (
                      ai_review_result IS NULL
                      OR ai_review_result::jsonb = '{}'::jsonb
                      OR ai_review_result::jsonb = 'null'::jsonb
                  )
	`,
		lessonPlanID,
		authorID,
		models.LPStatusDraft,
		models.LPVisibilityPersonal,
		domain,
	)
	if err != nil {
		return fmt.Errorf(
			"硬删除失败导入教案失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf(
			"%w: 删除时目标状态发生变化",
			ErrIncompleteImportedLessonPlanCleanupRejected,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交失败导入补偿事务失败: %w",
			err,
		)
	}

	return nil
}
