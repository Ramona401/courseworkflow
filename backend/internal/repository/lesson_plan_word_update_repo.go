package repository

// lesson_plan_word_update_repo.go — 教案语义正文与原格式Word当前版本的原子同步仓储
//
// 本文件只负责数据库事务边界，不读取或生成DOCX文件：
//   1. 锁定教案正文和当前Word文档；
//   2. 复核作者、状态、正文版本、Word版本、文件哈希和语义正文；
//   3. 保存修改前的教案正文历史；
//   4. 更新lesson_plans语义正文；
//   5. 更新Word当前文档并递增Word版本；
//   6. 依赖数据库触发器自动生成Word不可变完整版本；
//   7. 裁剪教案正文历史，保持最近50份。
//
// DOCX物理文件必须在调用本函数前写入私有不可变路径。事务失败时，
// 调用方负责删除尚未被数据库引用的新文件。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	// ErrLessonPlanWordContentUpdateConflict 表示教案或Word文档在文件生成期间已经变化。
	ErrLessonPlanWordContentUpdateConflict = errors.New("教案或原格式Word版本已变化")

	// ErrLessonPlanWordContentUpdateNotReady 表示当前Word文档不是可同步的active版本。
	ErrLessonPlanWordContentUpdateNotReady = errors.New("原格式Word当前不可同步")
)

// LessonPlanWordContentUpdateInput 是服务层完成DOCX生成后提交的可信事务输入。
type LessonPlanWordContentUpdateInput struct {
	LessonPlanID string
	OwnerID      string

	ExpectedPlanVersion    int
	ExpectedPlanTitle      string
	ExpectedPlanContent    string
	ExpectedPlanStructured string
	ExpectedPlanDuration   int
	ExpectedWordDocumentID string
	ExpectedWordVersion    int
	ExpectedWordStorageKey string
	ExpectedWordFileSHA256 string
	ExpectedWordSemantic   string

	NextTitle             string
	NextContentMarkdown   string
	NextContentStructured string
	NextDurationMinutes   int
	NextWordStorageKey    string
	NextWordFileSHA256    string
	NextWordStructureJSON string
	NextWordSemanticHash  string
	NextWordStructureHash string
	NextWordMetricsJSON   string
	NextWordWarningsJSON  string
	ChangeSource          string
	ChangedBy             *string
	ChangeSummary         string
}

// LessonPlanWordContentUpdateResult 返回事务提交后的双版本号。
type LessonPlanWordContentUpdateResult struct {
	LessonPlanVersion int
	WordVersion       int
}

// LessonPlanContentCASInput 是普通教案或非Word路径的事务级CAS更新输入。
//
// 该协议与Word双版本事务使用相同的作者、状态、版本和旧正文复核口径，
// 避免Service先查询、Repository后覆盖之间出现并发窗口。
type LessonPlanContentCASInput struct {
	LessonPlanID string
	OwnerID      string

	ExpectedVersion           int
	ExpectedTitle             string
	ExpectedContentMarkdown   string
	ExpectedContentStructured string
	ExpectedDurationMinutes   int

	NextTitle             string
	NextContentMarkdown   string
	NextContentStructured string
	NextDurationMinutes   int

	ChangeSource  string
	ChangedBy     *string
	ChangeSummary string
}

// LessonPlanContentCASResult 返回普通正文CAS事务的最终版本。
type LessonPlanContentCASResult struct {
	Changed           bool
	LessonPlanVersion int
	ContentMarkdown   string
}

// CommitLessonPlanContentUpdateCAS 原子更新不带Word文档的教案正文。
func CommitLessonPlanContentUpdateCAS(
	ctx context.Context,
	input LessonPlanContentCASInput,
) (*LessonPlanContentCASResult, error) {
	input.LessonPlanID = strings.TrimSpace(input.LessonPlanID)
	input.OwnerID = strings.TrimSpace(input.OwnerID)
	input.NextTitle = strings.TrimSpace(input.NextTitle)
	input.NextContentMarkdown = strings.TrimSpace(input.NextContentMarkdown)
	input.ChangeSource = strings.TrimSpace(input.ChangeSource)
	input.ChangeSummary = strings.TrimSpace(input.ChangeSummary)

	if input.ExpectedContentStructured == "" {
		input.ExpectedContentStructured = "{}"
	}
	if input.NextContentStructured == "" {
		input.NextContentStructured = "{}"
	}
	if input.ChangeSummary == "" {
		input.ChangeSummary = "更新教案正文"
	}
	if len([]rune(input.ChangeSummary)) > 500 {
		input.ChangeSummary = string([]rune(input.ChangeSummary)[:500])
	}

	if input.LessonPlanID == "" ||
		input.OwnerID == "" ||
		input.ExpectedVersion <= 0 ||
		input.NextTitle == "" ||
		input.NextContentMarkdown == "" ||
		input.NextDurationMinutes <= 0 ||
		!models.IsValidLessonPlanWordChangeSource(input.ChangeSource) {
		return nil, ErrLessonPlanWordInputInvalid
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始教案正文CAS事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		currentTitle      string
		currentContent    string
		currentStructured string
		currentDuration   int
		currentVersion    int
		currentAuthorID   string
		currentStatus     string
	)

	err = tx.QueryRow(
		ctx,
		`
SELECT
        title,
        COALESCE(content_markdown, ''),
        COALESCE(content_structured::text, '{}'),
        duration_minutes,
        version,
        author_id::text,
        status
FROM lesson_plans
WHERE id = $1
  AND deleted_at IS NULL
FOR UPDATE
`,
		input.LessonPlanID,
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
		return nil, fmt.Errorf("锁定教案正文CAS基线失败: %w", err)
	}

	if strings.TrimSpace(currentAuthorID) != input.OwnerID {
		return nil, ErrLessonPlanSectionNotAuthor
	}
	if !isLessonPlanSectionEditableStatus(currentStatus) {
		return nil, ErrLessonPlanSectionNotEditable
	}
	if currentVersion != input.ExpectedVersion ||
		currentTitle != input.ExpectedTitle ||
		currentContent != input.ExpectedContentMarkdown ||
		currentStructured != input.ExpectedContentStructured ||
		currentDuration != input.ExpectedDurationMinutes {
		return nil, ErrLessonPlanWordContentUpdateConflict
	}

	if currentTitle == input.NextTitle &&
		currentContent == input.NextContentMarkdown &&
		currentStructured == input.NextContentStructured &&
		currentDuration == input.NextDurationMinutes {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("提交教案正文CAS无变化事务失败: %w", err)
		}

		return &LessonPlanContentCASResult{
			Changed:           false,
			LessonPlanVersion: currentVersion,
			ContentMarkdown:   currentContent,
		}, nil
	}

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
)
VALUES (
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
		input.LessonPlanID,
		currentVersion,
		currentTitle,
		currentContent,
		currentStructured,
		currentDuration,
		input.ChangeSource,
		input.ChangedBy,
		input.ChangeSummary,
	)
	if err != nil {
		return nil, fmt.Errorf("保存教案CAS修改前版本失败: %w", err)
	}

	nextVersion := currentVersion + 1
	now := time.Now()

	result, err := tx.Exec(
		ctx,
		`
UPDATE lesson_plans
SET
        title = $1,
        content_markdown = $2,
        content_structured = $3::jsonb,
        duration_minutes = $4,
        version = $5,
        updated_at = $6
WHERE id = $7
  AND version = $8
  AND deleted_at IS NULL
`,
		input.NextTitle,
		input.NextContentMarkdown,
		input.NextContentStructured,
		input.NextDurationMinutes,
		nextVersion,
		now,
		input.LessonPlanID,
		currentVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("写入教案正文CAS结果失败: %w", err)
	}
	if result.RowsAffected() != 1 {
		return nil, ErrLessonPlanWordContentUpdateConflict
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
		input.LessonPlanID,
	)
	if err != nil {
		return nil, fmt.Errorf("裁剪教案正文CAS历史失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交教案正文CAS事务失败: %w", err)
	}

	return &LessonPlanContentCASResult{
		Changed:           true,
		LessonPlanVersion: nextVersion,
		ContentMarkdown:   input.NextContentMarkdown,
	}, nil
}

// CommitLessonPlanWordContentUpdate 原子提交教案正文和Word完整新版本。
func CommitLessonPlanWordContentUpdate(
	ctx context.Context,
	input LessonPlanWordContentUpdateInput,
) (*LessonPlanWordContentUpdateResult, error) {
	input.LessonPlanID = strings.TrimSpace(input.LessonPlanID)
	input.OwnerID = strings.TrimSpace(input.OwnerID)
	input.ExpectedPlanTitle = strings.TrimSpace(input.ExpectedPlanTitle)
	input.ExpectedWordDocumentID = strings.TrimSpace(input.ExpectedWordDocumentID)
	input.ExpectedWordStorageKey = strings.TrimSpace(input.ExpectedWordStorageKey)
	input.ExpectedWordFileSHA256 = strings.ToLower(strings.TrimSpace(input.ExpectedWordFileSHA256))
	input.ExpectedWordSemantic = strings.TrimSpace(input.ExpectedWordSemantic)
	input.NextTitle = strings.TrimSpace(input.NextTitle)
	input.NextContentMarkdown = strings.TrimSpace(input.NextContentMarkdown)
	input.NextWordStorageKey = strings.TrimSpace(input.NextWordStorageKey)
	input.NextWordFileSHA256 = strings.ToLower(strings.TrimSpace(input.NextWordFileSHA256))
	input.NextWordStructureJSON = strings.TrimSpace(input.NextWordStructureJSON)
	input.NextWordSemanticHash = strings.ToLower(strings.TrimSpace(input.NextWordSemanticHash))
	input.NextWordStructureHash = strings.ToLower(strings.TrimSpace(input.NextWordStructureHash))
	input.NextWordMetricsJSON = strings.TrimSpace(input.NextWordMetricsJSON)
	input.NextWordWarningsJSON = strings.TrimSpace(input.NextWordWarningsJSON)
	input.ChangeSource = strings.TrimSpace(input.ChangeSource)
	input.ChangeSummary = strings.TrimSpace(input.ChangeSummary)

	if input.NextContentStructured == "" {
		input.NextContentStructured = "{}"
	}
	if input.NextWordMetricsJSON == "" {
		input.NextWordMetricsJSON = "{}"
	}
	if input.NextWordWarningsJSON == "" {
		input.NextWordWarningsJSON = "[]"
	}
	if input.ChangeSummary == "" {
		input.ChangeSummary = "同步更新教案正文与原格式Word"
	}
	if len([]rune(input.ChangeSummary)) > 500 {
		input.ChangeSummary = string([]rune(input.ChangeSummary)[:500])
	}

	if input.LessonPlanID == "" ||
		input.OwnerID == "" ||
		input.ExpectedPlanVersion <= 0 ||
		input.ExpectedWordDocumentID == "" ||
		input.ExpectedWordVersion <= 0 ||
		input.ExpectedWordStorageKey == "" ||
		len(input.ExpectedWordFileSHA256) != 64 ||
		input.ExpectedWordSemantic == "" ||
		input.NextTitle == "" ||
		input.NextContentMarkdown == "" ||
		input.NextDurationMinutes <= 0 ||
		input.NextWordStorageKey == "" ||
		len(input.NextWordFileSHA256) != 64 ||
		input.NextWordStructureJSON == "" ||
		input.NextWordStructureJSON == "{}" ||
		len(input.NextWordSemanticHash) != 64 ||
		len(input.NextWordStructureHash) != 64 ||
		!models.IsValidLessonPlanWordChangeSource(input.ChangeSource) {
		return nil, ErrLessonPlanWordInputInvalid
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始教案与Word同步事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		currentTitle      string
		currentContent    string
		currentStructured string
		currentDuration   int
		currentVersion    int
		currentAuthorID   string
		currentStatus     string
	)

	err = tx.QueryRow(
		ctx,
		`
SELECT
        title,
        COALESCE(content_markdown, ''),
        COALESCE(content_structured::text, '{}'),
        duration_minutes,
        version,
        author_id::text,
        status
FROM lesson_plans
WHERE id = $1
  AND deleted_at IS NULL
FOR UPDATE
`,
		input.LessonPlanID,
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
		return nil, fmt.Errorf("锁定教案正文失败: %w", err)
	}

	if strings.TrimSpace(currentAuthorID) != input.OwnerID {
		return nil, ErrLessonPlanSectionNotAuthor
	}
	if !isLessonPlanSectionEditableStatus(currentStatus) {
		return nil, ErrLessonPlanSectionNotEditable
	}
	if currentVersion != input.ExpectedPlanVersion ||
		currentTitle != input.ExpectedPlanTitle ||
		currentContent != input.ExpectedPlanContent ||
		currentStructured != input.ExpectedPlanStructured ||
		currentDuration != input.ExpectedPlanDuration {
		return nil, ErrLessonPlanWordContentUpdateConflict
	}

	wordRow := tx.QueryRow(
		ctx,
		`
SELECT `+lessonPlanWordDocumentQualifiedSelectColumns+`
FROM lesson_plan_word_documents word_document
WHERE word_document.id = $1
  AND word_document.lesson_plan_id = $2
FOR UPDATE
`,
		input.ExpectedWordDocumentID,
		input.LessonPlanID,
	)

	wordDocument, err := scanLessonPlanWordDocument(wordRow)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanWordDocumentNotFound
		}
		return nil, fmt.Errorf("锁定当前Word文档失败: %w", err)
	}

	if wordDocument.Status != models.LessonPlanWordDocumentStatusActive ||
		wordDocument.Version != input.ExpectedWordVersion ||
		wordDocument.CurrentStorageKey != input.ExpectedWordStorageKey ||
		!strings.EqualFold(wordDocument.CurrentFileSHA256, input.ExpectedWordFileSHA256) ||
		wordDocument.SemanticMarkdown != input.ExpectedWordSemantic ||
		wordDocument.SemanticMarkdown != currentContent {
		return nil, ErrLessonPlanWordContentUpdateNotReady
	}

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
)
VALUES (
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
		input.LessonPlanID,
		currentVersion,
		currentTitle,
		currentContent,
		currentStructured,
		currentDuration,
		input.ChangeSource,
		input.ChangedBy,
		input.ChangeSummary,
	)
	if err != nil {
		return nil, fmt.Errorf("保存教案修改前版本失败: %w", err)
	}

	nextPlanVersion := currentVersion + 1
	now := time.Now()

	result, err := tx.Exec(
		ctx,
		`
UPDATE lesson_plans
SET
        title = $1,
        content_markdown = $2,
        content_structured = $3::jsonb,
        duration_minutes = $4,
        version = $5,
        updated_at = $6
WHERE id = $7
  AND version = $8
  AND deleted_at IS NULL
`,
		input.NextTitle,
		input.NextContentMarkdown,
		input.NextContentStructured,
		input.NextDurationMinutes,
		nextPlanVersion,
		now,
		input.LessonPlanID,
		currentVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("更新教案语义正文失败: %w", err)
	}
	if result.RowsAffected() != 1 {
		return nil, ErrLessonPlanWordContentUpdateConflict
	}

	nextWordVersion := wordDocument.Version + 1

	result, err = tx.Exec(
		ctx,
		`
UPDATE lesson_plan_word_documents
SET
        status = $1,
        version = $2,
        current_storage_key = $3,
        current_file_sha256 = $4,
        structure_json = $5::jsonb,
        semantic_markdown = $6,
        semantic_markdown_hash = $7,
        structure_hash = $8,
        metrics_json = $9::jsonb,
        warnings_json = $10::jsonb,
        last_change_source = $11,
        last_changed_by = $12,
        last_change_summary = $13,
        error_message = '',
        generated_at = $14
WHERE id = $15
  AND lesson_plan_id = $16
  AND version = $17
  AND current_storage_key = $18
  AND current_file_sha256 = $19
  AND semantic_markdown = $20
`,
		models.LessonPlanWordDocumentStatusActive,
		nextWordVersion,
		input.NextWordStorageKey,
		input.NextWordFileSHA256,
		input.NextWordStructureJSON,
		input.NextContentMarkdown,
		input.NextWordSemanticHash,
		input.NextWordStructureHash,
		input.NextWordMetricsJSON,
		input.NextWordWarningsJSON,
		input.ChangeSource,
		input.ChangedBy,
		input.ChangeSummary,
		now,
		input.ExpectedWordDocumentID,
		input.LessonPlanID,
		input.ExpectedWordVersion,
		input.ExpectedWordStorageKey,
		input.ExpectedWordFileSHA256,
		input.ExpectedWordSemantic,
	)
	if err != nil {
		return nil, fmt.Errorf("更新Word当前完整版本失败: %w", err)
	}
	if result.RowsAffected() != 1 {
		return nil, ErrLessonPlanWordContentUpdateConflict
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
		input.LessonPlanID,
	)
	if err != nil {
		return nil, fmt.Errorf("裁剪教案正文历史失败: %w", err)
	}

	var snapshotCount int
	if err := tx.QueryRow(
		ctx,
		`
SELECT COUNT(*)
FROM lesson_plan_word_document_versions
WHERE lesson_plan_id = $1
  AND version = $2
  AND storage_key = $3
  AND file_sha256 = $4
  AND semantic_markdown = $5
`,
		input.LessonPlanID,
		nextWordVersion,
		input.NextWordStorageKey,
		input.NextWordFileSHA256,
		input.NextContentMarkdown,
	).Scan(&snapshotCount); err != nil {
		return nil, fmt.Errorf("验证Word不可变版本失败: %w", err)
	}
	if snapshotCount != 1 {
		return nil, errors.New("Word不可变版本未按预期生成")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交教案与Word同步事务失败: %w", err)
	}

	return &LessonPlanWordContentUpdateResult{
		LessonPlanVersion: nextPlanVersion,
		WordVersion:       nextWordVersion,
	}, nil
}
