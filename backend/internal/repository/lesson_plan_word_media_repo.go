package repository

// lesson_plan_word_media_repo.go — Word图片资产语义同步仓储
//
// Word图片物理文件和lesson_plan_assets记录创建后，本仓储在同一事务中：
//   1. 锁定短时Word导入会话；
//   2. 锁定刚创建的正式教案；
//   3. 核对作者、教育域、文件哈希、状态和旧语义正文；
//   4. 更新会话中的Word结构和带图片URL的语义正文；
//   5. 同步更新lesson_plans.content_markdown。
//
// Word当前文档此时尚未创建，因此不会产生短暂的Word文档失步状态。

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// SyncLessonPlanWordImportMediaInput 是Word图片资产创建后的可信同步输入。
type SyncLessonPlanWordImportMediaInput struct {
	ImportSessionID          string
	LessonPlanID             string
	OwnerID                  string
	EducationDomain          string
	ExpectedFileSHA256       string
	ExpectedSemanticMarkdown string
	StructureJSON            string
	SemanticMarkdown         string
	SemanticMarkdownHash     string
	MetricsJSON              string
	WarningsJSON             string
}

// SyncLessonPlanWordImportMedia 同步短时会话和新教案的图片语义正文。
func SyncLessonPlanWordImportMedia(
	ctx context.Context,
	input SyncLessonPlanWordImportMediaInput,
) error {
	input.ImportSessionID = strings.TrimSpace(
		input.ImportSessionID,
	)
	input.LessonPlanID = strings.TrimSpace(
		input.LessonPlanID,
	)
	input.OwnerID = strings.TrimSpace(
		input.OwnerID,
	)
	input.EducationDomain = strings.ToLower(
		strings.TrimSpace(
			input.EducationDomain,
		),
	)
	input.ExpectedFileSHA256 = strings.TrimSpace(
		input.ExpectedFileSHA256,
	)
	input.ExpectedSemanticMarkdown = strings.TrimSpace(
		input.ExpectedSemanticMarkdown,
	)
	input.StructureJSON = strings.TrimSpace(
		input.StructureJSON,
	)
	input.SemanticMarkdown = strings.TrimSpace(
		input.SemanticMarkdown,
	)
	input.SemanticMarkdownHash = strings.TrimSpace(
		input.SemanticMarkdownHash,
	)
	input.MetricsJSON = strings.TrimSpace(
		input.MetricsJSON,
	)
	input.WarningsJSON = strings.TrimSpace(
		input.WarningsJSON,
	)

	if input.ImportSessionID == "" ||
		input.LessonPlanID == "" ||
		input.OwnerID == "" ||
		!models.IsTeachingEducationDomain(
			input.EducationDomain,
		) ||
		len(input.ExpectedFileSHA256) != 64 ||
		input.ExpectedSemanticMarkdown == "" ||
		input.StructureJSON == "" ||
		input.StructureJSON == "{}" ||
		input.SemanticMarkdown == "" ||
		len(input.SemanticMarkdownHash) != 64 {
		return ErrLessonPlanWordInputInvalid
	}

	if input.MetricsJSON == "" {
		input.MetricsJSON = "{}"
	}
	if input.WarningsJSON == "" {
		input.WarningsJSON = "[]"
	}

	var structureObject map[string]any
	if err := json.Unmarshal(
		[]byte(input.StructureJSON),
		&structureObject,
	); err != nil ||
		len(structureObject) == 0 {
		return fmt.Errorf(
			"%w: structure_json需要非空JSON对象",
			ErrLessonPlanWordInputInvalid,
		)
	}

	var metricsObject map[string]any
	if err := json.Unmarshal(
		[]byte(input.MetricsJSON),
		&metricsObject,
	); err != nil {
		return fmt.Errorf(
			"%w: metrics_json需要JSON对象",
			ErrLessonPlanWordInputInvalid,
		)
	}

	var warningArray []any
	if err := json.Unmarshal(
		[]byte(input.WarningsJSON),
		&warningArray,
	); err != nil {
		return fmt.Errorf(
			"%w: warnings_json需要JSON数组",
			ErrLessonPlanWordInputInvalid,
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始同步Word图片语义事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		sessionOwnerID      string
		sessionDomain       string
		sessionStatus       string
		sessionFileSHA256   string
		sessionSemantic     string
		sessionExpiresAt    time.Time
		sessionLessonPlanID sql.NullString
	)

	err = tx.QueryRow(
		ctx,
		`
SELECT
	created_by::text,
	education_domain,
	status,
	file_sha256,
	semantic_markdown,
	expires_at,
	lesson_plan_id::text
FROM lesson_plan_word_import_sessions
WHERE id = $1
FOR UPDATE
`,
		input.ImportSessionID,
	).Scan(
		&sessionOwnerID,
		&sessionDomain,
		&sessionStatus,
		&sessionFileSHA256,
		&sessionSemantic,
		&sessionExpiresAt,
		&sessionLessonPlanID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLessonPlanWordImportNotFound
		}

		return fmt.Errorf(
			"锁定Word导入会话失败: %w",
			err,
		)
	}

	if strings.TrimSpace(sessionOwnerID) !=
		input.OwnerID ||
		sessionDomain != input.EducationDomain ||
		sessionStatus !=
			models.LessonPlanWordImportStatusParsed ||
		sessionFileSHA256 !=
			input.ExpectedFileSHA256 ||
		strings.TrimSpace(sessionSemantic) !=
			input.ExpectedSemanticMarkdown ||
		sessionLessonPlanID.Valid ||
		!sessionExpiresAt.After(time.Now()) {
		return ErrLessonPlanWordImportConflict
	}

	var (
		planAuthorID   string
		planDomain     string
		planStatus     string
		planVisibility string
		planContent    string
		planDeletedAt  sql.NullTime
	)

	err = tx.QueryRow(
		ctx,
		`
SELECT
	author_id::text,
	lower(btrim(COALESCE(education_domain, ''))),
	status,
	visibility,
	COALESCE(content_markdown, ''),
	deleted_at
FROM lesson_plans
WHERE id = $1
FOR UPDATE
`,
		input.LessonPlanID,
	).Scan(
		&planAuthorID,
		&planDomain,
		&planStatus,
		&planVisibility,
		&planContent,
		&planDeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLessonPlanNotFound
		}

		return fmt.Errorf(
			"锁定Word图片目标教案失败: %w",
			err,
		)
	}

	if strings.TrimSpace(planAuthorID) !=
		input.OwnerID ||
		planDomain != input.EducationDomain ||
		planStatus != models.LPStatusDraft ||
		planVisibility != models.LPVisibilityPersonal ||
		planDeletedAt.Valid ||
		strings.TrimSpace(planContent) !=
			input.ExpectedSemanticMarkdown {
		return ErrLessonPlanWordDocumentConflict
	}

	result, err := tx.Exec(
		ctx,
		`
UPDATE lesson_plan_word_import_sessions
SET
	structure_json = $1::jsonb,
	semantic_markdown = $2,
	semantic_markdown_hash = $3,
	metrics_json = $4::jsonb,
	warnings_json = $5::jsonb,
	error_message = ''
WHERE id = $6
  AND created_by = $7
  AND status = $8
  AND lesson_plan_id IS NULL
  AND expires_at > NOW()
`,
		input.StructureJSON,
		input.SemanticMarkdown,
		input.SemanticMarkdownHash,
		input.MetricsJSON,
		input.WarningsJSON,
		input.ImportSessionID,
		input.OwnerID,
		models.LessonPlanWordImportStatusParsed,
	)
	if err != nil {
		return fmt.Errorf(
			"更新Word会话图片语义失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return ErrLessonPlanWordImportConflict
	}

	result, err = tx.Exec(
		ctx,
		`
UPDATE lesson_plans
SET
	content_markdown = $1,
	updated_at = NOW()
WHERE id = $2
  AND author_id = $3
  AND education_domain = $4
  AND status = $5
  AND visibility = $6
  AND deleted_at IS NULL
  AND COALESCE(content_markdown, '') = $7
`,
		input.SemanticMarkdown,
		input.LessonPlanID,
		input.OwnerID,
		input.EducationDomain,
		models.LPStatusDraft,
		models.LPVisibilityPersonal,
		input.ExpectedSemanticMarkdown,
	)
	if err != nil {
		return fmt.Errorf(
			"同步教案Word图片语义正文失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return ErrLessonPlanWordDocumentConflict
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交Word图片语义同步事务失败: %w",
			err,
		)
	}

	return nil
}
