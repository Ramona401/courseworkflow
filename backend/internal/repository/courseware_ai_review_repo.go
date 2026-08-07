package repository

// courseware_ai_review_repo.go
//
// 课件 AI 审核助手的数据访问层。
//
// 持久化边界：
//   - Session：一次完整课件审核任务及其不可变配置和内容快照；
//   - Batch：顺序分批审核结果和连续性账本；
//   - Message：后续审核员与 AI 的追问对话。
//
// 本文件不负责：
//   - 人工审核权限判断；
//   - 页面互动代码解析；
//   - AI 调用；
//   - 自动作出通过或退回决定。
//
// JSONB 字段读取时统一使用 ::text，交给 service 层按结构解析；
// 写入时保证空值归一为合法的 {} 或 []，避免 NOT NULL JSONB 列写入 NULL。
//
// R-02 配置字段只在 CreateCoursewareAIReviewSession 中写入。
// UpdateCoursewareAIReviewPrepared 不得重新写配置，数据库触发器也会阻止创建后修改。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== Session 扫描 ====================

const cwAIReviewSessionSelectColumns = `
	id,
	courseware_id,
	reviewer_id,
	COALESCE(assistant_id::text, ''),
	COALESCE(lesson_plan_id::text, ''),
	review_level,
	education_domain,
	subject,
	grade,
	review_config_schema_version,
	COALESCE(review_dimensions_json::text, '[]'),
	custom_dimension_description,
	lesson_reference_mode,
	review_config_hash,
	status,
	current_stage,
	current_batch_no,
	total_batches,
	courseware_snapshot_hash,
	pages_snapshot_hash,
	lesson_plan_snapshot_hash,
	course_outline_snapshot_hash,
	system_prompt_key,
	system_prompt_version,
	system_prompt_snapshot,
	assistant_prompt_snapshot,
	COALESCE(context_manifest_json::text, '{}'),
	COALESCE(baseline_json::text, '{}'),
	COALESCE(page_index_json::text, '[]'),
	COALESCE(continuity_ledger_json::text, '{}'),
	COALESCE(final_report_json::text, '{}'),
	model_used,
	tokens_used,
	error_message,
	created_at,
	updated_at,
	completed_at`

func scanCoursewareAIReviewSession(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareAIReviewSession, error) {
	var session models.CoursewareAIReviewSession
	var assistantID string
	var lessonPlanID string

	err := row.Scan(
		&session.ID,
		&session.CoursewareID,
		&session.ReviewerID,
		&assistantID,
		&lessonPlanID,
		&session.ReviewLevel,
		&session.EducationDomain,
		&session.Subject,
		&session.Grade,
		&session.ReviewConfigSchemaVersion,
		&session.ReviewDimensionsJSON,
		&session.CustomDimensionDescription,
		&session.LessonReferenceMode,
		&session.ReviewConfigHash,
		&session.Status,
		&session.CurrentStage,
		&session.CurrentBatchNo,
		&session.TotalBatches,
		&session.CoursewareSnapshotHash,
		&session.PagesSnapshotHash,
		&session.LessonPlanSnapshotHash,
		&session.CourseOutlineSnapshotHash,
		&session.SystemPromptKey,
		&session.SystemPromptVersion,
		&session.SystemPromptSnapshot,
		&session.AssistantPromptSnapshot,
		&session.ContextManifestJSON,
		&session.BaselineJSON,
		&session.PageIndexJSON,
		&session.ContinuityLedgerJSON,
		&session.FinalReportJSON,
		&session.ModelUsed,
		&session.TokensUsed,
		&session.ErrorMessage,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.CompletedAt,
	)
	if err != nil {
		return nil, err
	}

	if assistantID != "" {
		session.AssistantID = &assistantID
	}
	if lessonPlanID != "" {
		session.LessonPlanID = &lessonPlanID
	}

	return &session, nil
}

// ==================== Batch 扫描 ====================

const cwAIReviewBatchSelectColumns = `
	id,
	session_id,
	batch_no,
	COALESCE(page_scope_json::text, '{}'),
	status,
	input_hash,
	COALESCE(continuity_before_json::text, '{}'),
	COALESCE(input_manifest_json::text, '{}'),
	COALESCE(result_json::text, '{}'),
	COALESCE(continuity_after_json::text, '{}'),
	COALESCE(risk_pages_json::text, '[]'),
	model_used,
	tokens_used,
	error_message,
	started_at,
	completed_at,
	created_at,
	updated_at`

func scanCoursewareAIReviewBatch(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareAIReviewBatch, error) {
	var batch models.CoursewareAIReviewBatch

	err := row.Scan(
		&batch.ID,
		&batch.SessionID,
		&batch.BatchNo,
		&batch.PageScopeJSON,
		&batch.Status,
		&batch.InputHash,
		&batch.ContinuityBeforeJSON,
		&batch.InputManifestJSON,
		&batch.ResultJSON,
		&batch.ContinuityAfterJSON,
		&batch.RiskPagesJSON,
		&batch.ModelUsed,
		&batch.TokensUsed,
		&batch.ErrorMessage,
		&batch.StartedAt,
		&batch.CompletedAt,
		&batch.CreatedAt,
		&batch.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &batch, nil
}

// ==================== Session 写入 ====================

// CreateCoursewareAIReviewSession 创建处于 preparing 阶段的新会话。
//
// R-02 配置字段在此处一次性写入。数据库 BEFORE INSERT 触发器会重新计算
// review_config_hash，因此调用方不能通过传入哈希伪造配置事实。
func CreateCoursewareAIReviewSession(
	ctx context.Context,
	session *models.CoursewareAIReviewSession,
) error {
	if session == nil {
		return errors.New("课件AI审核会话不能为空")
	}

	query := `
		INSERT INTO courseware_ai_review_sessions (
			courseware_id,
			reviewer_id,
			assistant_id,
			lesson_plan_id,
			review_level,
			education_domain,
			subject,
			grade,
			review_config_schema_version,
			review_dimensions_json,
			custom_dimension_description,
			lesson_reference_mode,
			review_config_hash,
			status,
			current_stage,
			current_batch_no,
			total_batches,
			courseware_snapshot_hash,
			pages_snapshot_hash,
			lesson_plan_snapshot_hash,
			course_outline_snapshot_hash,
			system_prompt_key,
			system_prompt_version,
			system_prompt_snapshot,
			assistant_prompt_snapshot,
			context_manifest_json,
			baseline_json,
			page_index_json,
			continuity_ledger_json,
			final_report_json,
			model_used,
			tokens_used,
			error_message,
			created_at,
			updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10::jsonb, $11, $12, $13,
			'preparing', 'baseline', 0, 0,
			$14, $15, $16, $17,
			$18, $19, $20, $21,
			$22, $23, $24, $25, $26,
			'', 0, '', NOW(), NOW()
		)
		RETURNING
			id,
			status,
			current_stage,
			review_config_hash,
			created_at,
			updated_at`

	err := database.DB.QueryRow(
		ctx,
		query,
		session.CoursewareID,
		session.ReviewerID,
		session.AssistantID,
		session.LessonPlanID,
		session.ReviewLevel,
		strings.TrimSpace(session.EducationDomain),
		strings.TrimSpace(session.Subject),
		strings.TrimSpace(session.Grade),
		session.ReviewConfigSchemaVersion,
		cwAIReviewJSONOrDefault(session.ReviewDimensionsJSON, "[]"),
		strings.TrimSpace(session.CustomDimensionDescription),
		strings.TrimSpace(session.LessonReferenceMode),
		strings.TrimSpace(session.ReviewConfigHash),
		strings.TrimSpace(session.CoursewareSnapshotHash),
		strings.TrimSpace(session.PagesSnapshotHash),
		strings.TrimSpace(session.LessonPlanSnapshotHash),
		strings.TrimSpace(session.CourseOutlineSnapshotHash),
		strings.TrimSpace(session.SystemPromptKey),
		session.SystemPromptVersion,
		session.SystemPromptSnapshot,
		session.AssistantPromptSnapshot,
		cwAIReviewJSONOrDefault(session.ContextManifestJSON, "{}"),
		cwAIReviewJSONOrDefault(session.BaselineJSON, "{}"),
		cwAIReviewJSONOrDefault(session.PageIndexJSON, "[]"),
		cwAIReviewJSONOrDefault(session.ContinuityLedgerJSON, "{}"),
		cwAIReviewJSONOrDefault(session.FinalReportJSON, "{}"),
	).Scan(
		&session.ID,
		&session.Status,
		&session.CurrentStage,
		&session.ReviewConfigHash,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建课件AI审核会话失败: %w", err)
	}

	return nil
}

// UpdateCoursewareAIReviewPrepared 写入完整基准、页面索引和批次数量。
//
// 调用本函数前，分批记录应已经通过 ReplaceCoursewareAIReviewBatches 写入。
// 成功后会话进入 reviewing / batch_review 状态，等待后续顺序执行各批 AI 审核。
//
// 本函数不得写 R-02 配置字段。配置在创建会话时已经冻结。
func UpdateCoursewareAIReviewPrepared(
	ctx context.Context,
	session *models.CoursewareAIReviewSession,
) error {
	if session == nil || strings.TrimSpace(session.ID) == "" {
		return errors.New("缺少课件AI审核会话")
	}

	result, err := database.DB.Exec(
		ctx,
		`
		UPDATE courseware_ai_review_sessions
		SET
			assistant_id = $2,
			lesson_plan_id = $3,
			education_domain = $4,
			subject = $5,
			grade = $6,
			status = 'reviewing',
			current_stage = 'batch_review',
			current_batch_no = 0,
			total_batches = $7,
			courseware_snapshot_hash = $8,
			pages_snapshot_hash = $9,
			lesson_plan_snapshot_hash = $10,
			course_outline_snapshot_hash = $11,
			system_prompt_key = $12,
			system_prompt_version = $13,
			system_prompt_snapshot = $14,
			assistant_prompt_snapshot = $15,
			context_manifest_json = $16,
			baseline_json = $17,
			page_index_json = $18,
			continuity_ledger_json = $19,
			final_report_json = '{}'::jsonb,
			model_used = '',
			tokens_used = 0,
			error_message = '',
			completed_at = NULL,
			updated_at = NOW()
		WHERE id = $1`,
		session.ID,
		session.AssistantID,
		session.LessonPlanID,
		strings.TrimSpace(session.EducationDomain),
		strings.TrimSpace(session.Subject),
		strings.TrimSpace(session.Grade),
		session.TotalBatches,
		strings.TrimSpace(session.CoursewareSnapshotHash),
		strings.TrimSpace(session.PagesSnapshotHash),
		strings.TrimSpace(session.LessonPlanSnapshotHash),
		strings.TrimSpace(session.CourseOutlineSnapshotHash),
		strings.TrimSpace(session.SystemPromptKey),
		session.SystemPromptVersion,
		session.SystemPromptSnapshot,
		session.AssistantPromptSnapshot,
		cwAIReviewJSONOrDefault(session.ContextManifestJSON, "{}"),
		cwAIReviewJSONOrDefault(session.BaselineJSON, "{}"),
		cwAIReviewJSONOrDefault(session.PageIndexJSON, "[]"),
		cwAIReviewJSONOrDefault(session.ContinuityLedgerJSON, "{}"),
	)
	if err != nil {
		return fmt.Errorf("更新课件AI审核准备结果失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("课件AI审核会话不存在")
	}

	session.Status = models.CWAIReviewStatusReviewing
	session.CurrentStage = models.CWAIReviewStageBatch
	session.CurrentBatchNo = 0

	return nil
}

// CancelActiveCoursewareAIReviewSessions 取消同一审核员、课件和级别下的旧活动会话。
//
// 明确重新开始审核时调用，避免数据库活动会话唯一索引冲突。
// 已完成、已失败和已取消的历史记录不受影响。
func CancelActiveCoursewareAIReviewSessions(
	ctx context.Context,
	coursewareID string,
	reviewerID string,
	reviewLevel int,
	reason string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`
		UPDATE courseware_ai_review_sessions
		SET
			status = 'cancelled',
			error_message = $4,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE courseware_id = $1
			AND reviewer_id = $2
			AND review_level = $3
			AND status IN (
				'pending',
				'preparing',
				'reviewing',
				'aggregating'
			)`,
		coursewareID,
		reviewerID,
		reviewLevel,
		strings.TrimSpace(reason),
	)
	if err != nil {
		return fmt.Errorf("取消旧课件AI审核会话失败: %w", err)
	}

	return nil
}

// MarkCoursewareAIReviewSessionFailed 标记准备或执行失败。
func MarkCoursewareAIReviewSessionFailed(
	ctx context.Context,
	sessionID string,
	errorMessage string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`
		UPDATE courseware_ai_review_sessions
		SET
			status = 'failed',
			error_message = $2,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1`,
		sessionID,
		strings.TrimSpace(errorMessage),
	)
	if err != nil {
		return fmt.Errorf("标记课件AI审核会话失败状态失败: %w", err)
	}

	return nil
}

// ==================== Session 查询 ====================

// GetCoursewareAIReviewSessionByID 按会话ID查询。
func GetCoursewareAIReviewSessionByID(
	ctx context.Context,
	sessionID string,
) (*models.CoursewareAIReviewSession, error) {
	session, err := scanCoursewareAIReviewSession(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwAIReviewSessionSelectColumns+`
			 FROM courseware_ai_review_sessions
			 WHERE id = $1`,
			sessionID,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询课件AI审核会话失败: %w", err)
	}

	return session, nil
}

// GetLatestCoursewareAIReviewSession 查询审核员对某课件最新一次会话。
func GetLatestCoursewareAIReviewSession(
	ctx context.Context,
	coursewareID string,
	reviewerID string,
	reviewLevel int,
) (*models.CoursewareAIReviewSession, error) {
	session, err := scanCoursewareAIReviewSession(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwAIReviewSessionSelectColumns+`
			 FROM courseware_ai_review_sessions
			 WHERE courseware_id = $1
				AND reviewer_id = $2
				AND review_level = $3
			 ORDER BY created_at DESC
			 LIMIT 1`,
			coursewareID,
			reviewerID,
			reviewLevel,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询最新课件AI审核会话失败: %w", err)
	}

	return session, nil
}

// ==================== Batch 写入与查询 ====================

// ReplaceCoursewareAIReviewBatches 原子替换某会话的全部待执行批次。
func ReplaceCoursewareAIReviewBatches(
	ctx context.Context,
	sessionID string,
	batches []*models.CoursewareAIReviewBatch,
) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启课件AI审核批次事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(
		ctx,
		`DELETE FROM courseware_ai_review_batches
		 WHERE session_id = $1`,
		sessionID,
	); err != nil {
		return fmt.Errorf("清理旧课件AI审核批次失败: %w", err)
	}

	for _, batch := range batches {
		if batch == nil {
			continue
		}

		batch.SessionID = sessionID
		batch.Status = models.CWAIReviewBatchPending

		err := tx.QueryRow(
			ctx,
			`
			INSERT INTO courseware_ai_review_batches (
				session_id,
				batch_no,
				page_scope_json,
				status,
				input_hash,
				continuity_before_json,
				input_manifest_json,
				result_json,
				continuity_after_json,
				risk_pages_json,
				model_used,
				tokens_used,
				error_message,
				created_at,
				updated_at
			)
			VALUES (
				$1, $2, $3, 'pending', $4,
				$5, $6, '{}'::jsonb, '{}'::jsonb, '[]'::jsonb,
				'', 0, '', NOW(), NOW()
			)
			RETURNING id, created_at, updated_at`,
			sessionID,
			batch.BatchNo,
			cwAIReviewJSONOrDefault(batch.PageScopeJSON, "{}"),
			strings.TrimSpace(batch.InputHash),
			cwAIReviewJSONOrDefault(batch.ContinuityBeforeJSON, "{}"),
			cwAIReviewJSONOrDefault(batch.InputManifestJSON, "{}"),
		).Scan(
			&batch.ID,
			&batch.CreatedAt,
			&batch.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("创建课件AI审核第%d批失败: %w", batch.BatchNo, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交课件AI审核批次事务失败: %w", err)
	}

	return nil
}

// ListCoursewareAIReviewBatches 按批次号查询。
func ListCoursewareAIReviewBatches(
	ctx context.Context,
	sessionID string,
) ([]*models.CoursewareAIReviewBatch, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT `+cwAIReviewBatchSelectColumns+`
		 FROM courseware_ai_review_batches
		 WHERE session_id = $1
		 ORDER BY batch_no ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询课件AI审核批次失败: %w", err)
	}
	defer rows.Close()

	batches := make([]*models.CoursewareAIReviewBatch, 0)

	for rows.Next() {
		batch, err := scanCoursewareAIReviewBatch(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描课件AI审核批次失败: %w", err)
		}
		batches = append(batches, batch)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历课件AI审核批次失败: %w", err)
	}

	return batches, nil
}

// ==================== JSON 辅助 ====================

func cwAIReviewJSONOrDefault(raw string, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		return raw
	}

	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		return "{}"
	}

	return fallback
}
