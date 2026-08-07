package repository

// lesson_plan_word_restore_repo.go — 教案正文历史与Word不可变版本的原子恢复仓储
//
// 本文件负责两类数据库能力：
//   1. 按作者和目标语义正文查询可能对应的Word不可变版本；
//   2. 在同一事务内保存当前正文快照、恢复目标正文、恢复目标Word结构，
//      并递增教案版本和Word版本。
//
// 恢复允许当前Word处于active或stale：
//   - active会在lesson_plans更新触发器中短暂变为stale；
//   - stale可通过可信历史Word快照重新回到active；
//   - failed不允许直接恢复。
//
// DOCX物理文件必须由Service先复制到新的不可变版本路径。
// 事务失败时，Service负责删除未被数据库引用的新文件。

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
	// ErrLessonPlanWordRestoreConflict 表示恢复准备期间教案或Word当前版本发生变化。
	ErrLessonPlanWordRestoreConflict = errors.New(
		"教案或原格式Word版本已变化",
	)
)

// ListLessonPlanWordVersionsBySemanticForOwner 查询作者本人、正文完全一致的Word版本。
//
// 返回顺序为Word版本号倒序。调用方必须继续执行版本选择和歧义判断，
// 不能因为同一Markdown命中多份不同DOCX就随意选择。
func ListLessonPlanWordVersionsBySemanticForOwner(
	ctx context.Context,
	lessonPlanID string,
	ownerID string,
	semanticMarkdown string,
) ([]*models.LessonPlanWordDocumentVersion, error) {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	ownerID = strings.TrimSpace(ownerID)
	semanticMarkdown = strings.TrimSpace(semanticMarkdown)

	if lessonPlanID == "" ||
		ownerID == "" ||
		semanticMarkdown == "" {
		return nil, ErrLessonPlanWordInputInvalid
	}

	rows, err := database.DB.Query(
		ctx,
		`
SELECT `+lessonPlanWordVersionSelectColumns+`
FROM lesson_plan_word_document_versions version_snapshot
INNER JOIN lesson_plans lesson_plan
	ON lesson_plan.id = version_snapshot.lesson_plan_id
LEFT JOIN users changed_user
	ON changed_user.id = version_snapshot.changed_by
WHERE version_snapshot.lesson_plan_id = $1
  AND lesson_plan.author_id = $2
  AND lesson_plan.deleted_at IS NULL
  AND version_snapshot.semantic_markdown = $3
ORDER BY
	version_snapshot.version DESC,
	version_snapshot.created_at DESC
`,
		lessonPlanID,
		ownerID,
		semanticMarkdown,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询正文对应的Word历史版本失败: %w",
			err,
		)
	}
	defer rows.Close()

	versions := make(
		[]*models.LessonPlanWordDocumentVersion,
		0,
	)

	for rows.Next() {
		version, scanErr :=
			scanLessonPlanWordDocumentVersion(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描正文对应的Word历史版本失败: %w",
				scanErr,
			)
		}
		versions = append(versions, version)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历正文对应的Word历史版本失败: %w",
			err,
		)
	}

	return versions, nil
}

// LessonPlanWordRestoreInput 是恢复服务已完成文件复制后提交的可信事务输入。
type LessonPlanWordRestoreInput struct {
	LessonPlanID string
	OwnerID      string

	ExpectedPlanVersion    int
	ExpectedPlanTitle      string
	ExpectedPlanContent    string
	ExpectedPlanStructured string
	ExpectedPlanDuration   int

	ExpectedWordDocumentID string
	ExpectedWordStatus     string
	ExpectedWordVersion    int
	ExpectedWordStorageKey string
	ExpectedWordFileSHA256 string
	ExpectedWordSemantic   string

	NextTitle               string
	NextContentMarkdown     string
	NextContentStructured   string
	NextDurationMinutes     int
	NextWordStorageKey      string
	NextWordFileSHA256      string
	NextWordParserVersion   string
	NextWordStructureSchema int
	NextWordStructureJSON   string
	NextWordSemanticHash    string
	NextWordStructureHash   string
	NextWordMetricsJSON     string
	NextWordWarningsJSON    string
	ChangedBy               *string
	ChangeSummary           string
}

// CommitLessonPlanWordVersionRestore 原子恢复教案正文和对应Word完整快照。
func CommitLessonPlanWordVersionRestore(
	ctx context.Context,
	input LessonPlanWordRestoreInput,
) (*LessonPlanWordContentUpdateResult, error) {
	input.LessonPlanID = strings.TrimSpace(input.LessonPlanID)
	input.OwnerID = strings.TrimSpace(input.OwnerID)
	input.ExpectedPlanTitle = strings.TrimSpace(
		input.ExpectedPlanTitle,
	)
	input.ExpectedWordDocumentID = strings.TrimSpace(
		input.ExpectedWordDocumentID,
	)
	input.ExpectedWordStatus = strings.TrimSpace(
		input.ExpectedWordStatus,
	)
	input.ExpectedWordStorageKey = strings.TrimSpace(
		input.ExpectedWordStorageKey,
	)
	input.ExpectedWordFileSHA256 = strings.ToLower(
		strings.TrimSpace(input.ExpectedWordFileSHA256),
	)
	input.ExpectedWordSemantic = strings.TrimSpace(
		input.ExpectedWordSemantic,
	)
	input.NextTitle = strings.TrimSpace(input.NextTitle)
	input.NextContentMarkdown = strings.TrimSpace(
		input.NextContentMarkdown,
	)
	input.NextWordStorageKey = strings.TrimSpace(
		input.NextWordStorageKey,
	)
	input.NextWordFileSHA256 = strings.ToLower(
		strings.TrimSpace(input.NextWordFileSHA256),
	)
	input.NextWordParserVersion = strings.TrimSpace(
		input.NextWordParserVersion,
	)
	input.NextWordStructureJSON = strings.TrimSpace(
		input.NextWordStructureJSON,
	)
	input.NextWordSemanticHash = strings.ToLower(
		strings.TrimSpace(input.NextWordSemanticHash),
	)
	input.NextWordStructureHash = strings.ToLower(
		strings.TrimSpace(input.NextWordStructureHash),
	)
	input.NextWordMetricsJSON = strings.TrimSpace(
		input.NextWordMetricsJSON,
	)
	input.NextWordWarningsJSON = strings.TrimSpace(
		input.NextWordWarningsJSON,
	)
	input.ChangeSummary = strings.TrimSpace(input.ChangeSummary)

	if input.ExpectedPlanStructured == "" {
		input.ExpectedPlanStructured = "{}"
	}
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
		input.ChangeSummary =
			"恢复教案正文历史版本并同步原格式Word"
	}
	if len([]rune(input.ChangeSummary)) > 500 {
		input.ChangeSummary =
			string([]rune(input.ChangeSummary)[:500])
	}

	statusAllowed :=
		input.ExpectedWordStatus ==
			models.LessonPlanWordDocumentStatusActive ||
			input.ExpectedWordStatus ==
				models.LessonPlanWordDocumentStatusStale

	if input.LessonPlanID == "" ||
		input.OwnerID == "" ||
		input.ExpectedPlanVersion <= 0 ||
		input.ExpectedWordDocumentID == "" ||
		!statusAllowed ||
		input.ExpectedWordVersion <= 0 ||
		input.ExpectedWordStorageKey == "" ||
		len(input.ExpectedWordFileSHA256) != 64 ||
		input.ExpectedWordSemantic == "" ||
		input.NextTitle == "" ||
		input.NextContentMarkdown == "" ||
		input.NextDurationMinutes <= 0 ||
		input.NextWordStorageKey == "" ||
		len(input.NextWordFileSHA256) != 64 ||
		input.NextWordParserVersion == "" ||
		input.NextWordStructureSchema <= 0 ||
		input.NextWordStructureJSON == "" ||
		input.NextWordStructureJSON == "{}" ||
		len(input.NextWordSemanticHash) != 64 ||
		len(input.NextWordStructureHash) != 64 {
		return nil, ErrLessonPlanWordInputInvalid
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始教案与Word历史恢复事务失败: %w",
			err,
		)
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
		return nil, fmt.Errorf(
			"锁定待恢复教案失败: %w",
			err,
		)
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
		return nil, ErrLessonPlanWordRestoreConflict
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

	currentWord, err :=
		scanLessonPlanWordDocument(wordRow)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanWordDocumentNotFound
		}
		return nil, fmt.Errorf(
			"锁定待恢复Word当前文档失败: %w",
			err,
		)
	}

	if currentWord.Status != input.ExpectedWordStatus ||
		currentWord.Version != input.ExpectedWordVersion ||
		currentWord.CurrentStorageKey !=
			input.ExpectedWordStorageKey ||
		!strings.EqualFold(
			currentWord.CurrentFileSHA256,
			input.ExpectedWordFileSHA256,
		) ||
		currentWord.SemanticMarkdown !=
			input.ExpectedWordSemantic {
		return nil, ErrLessonPlanWordRestoreConflict
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
		models.LPVersionSourceRestore,
		input.ChangedBy,
		input.ChangeSummary,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"保存恢复前教案正文版本失败: %w",
			err,
		)
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
		return nil, fmt.Errorf(
			"恢复教案正文失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return nil, ErrLessonPlanWordRestoreConflict
	}

	nextWordVersion := currentWord.Version + 1

	// 注意：lesson_plans更新触发器可能已把active文档改成stale，
	// 因此此处不再以status作为WHERE条件，而是继续用已锁定的版本、
	// 文件哈希和语义正文执行完整CAS。
	result, err = tx.Exec(
		ctx,
		`
UPDATE lesson_plan_word_documents
SET
	status = $1,
	version = $2,
	current_storage_key = $3,
	current_file_sha256 = $4,
	parser_version = $5,
	structure_schema_version = $6,
	structure_json = $7::jsonb,
	semantic_markdown = $8,
	semantic_markdown_hash = $9,
	structure_hash = $10,
	metrics_json = $11::jsonb,
	warnings_json = $12::jsonb,
	last_change_source = $13,
	last_changed_by = $14,
	last_change_summary = $15,
	error_message = '',
	generated_at = $16
WHERE id = $17
  AND lesson_plan_id = $18
  AND version = $19
  AND current_storage_key = $20
  AND current_file_sha256 = $21
  AND semantic_markdown = $22
`,
		models.LessonPlanWordDocumentStatusActive,
		nextWordVersion,
		input.NextWordStorageKey,
		input.NextWordFileSHA256,
		input.NextWordParserVersion,
		input.NextWordStructureSchema,
		input.NextWordStructureJSON,
		input.NextContentMarkdown,
		input.NextWordSemanticHash,
		input.NextWordStructureHash,
		input.NextWordMetricsJSON,
		input.NextWordWarningsJSON,
		models.LessonPlanWordChangeSourceRestore,
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
		return nil, fmt.Errorf(
			"恢复Word当前完整版本失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return nil, ErrLessonPlanWordRestoreConflict
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
		return nil, fmt.Errorf(
			"裁剪教案正文历史失败: %w",
			err,
		)
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
  AND structure_hash = $5
  AND semantic_markdown = $6
`,
		input.LessonPlanID,
		nextWordVersion,
		input.NextWordStorageKey,
		input.NextWordFileSHA256,
		input.NextWordStructureHash,
		input.NextContentMarkdown,
	).Scan(&snapshotCount); err != nil {
		return nil, fmt.Errorf(
			"验证恢复后的Word不可变版本失败: %w",
			err,
		)
	}
	if snapshotCount != 1 {
		return nil, errors.New(
			"恢复后的Word不可变版本未按预期生成",
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交教案与Word历史恢复事务失败: %w",
			err,
		)
	}

	return &LessonPlanWordContentUpdateResult{
		LessonPlanVersion: nextPlanVersion,
		WordVersion:       nextWordVersion,
	}, nil
}
