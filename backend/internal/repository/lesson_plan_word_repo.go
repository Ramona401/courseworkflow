package repository

// lesson_plan_word_repo.go — 原格式Word教案当前文档与版本查询仓储
//
// 职责：
//   1. 定义Word保真仓储共用错误、扫描接口和当前文档列协议；
//   2. 查询作者本人的当前Word文档；
//   3. 查询不可变Word版本列表；
//   4. 精确读取单个不可变Word版本。
//
// Word上传会话、解析状态和正式确认绑定已经拆分到
// lesson_plan_word_import_repo.go，避免单文件继续超过900行。

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
	ErrLessonPlanWordImportNotFound = errors.New(
		"Word导入会话不存在",
	)

	ErrLessonPlanWordImportConflict = errors.New(
		"Word导入会话状态冲突",
	)

	ErrLessonPlanWordDocumentNotFound = errors.New(
		"原格式Word教案不存在",
	)

	ErrLessonPlanWordDocumentConflict = errors.New(
		"原格式Word教案状态冲突",
	)

	ErrLessonPlanWordInputInvalid = errors.New(
		"原格式Word教案输入无效",
	)
)

type lessonPlanWordRowScanner interface {
	Scan(dest ...any) error
}

// lessonPlanWordDocumentSelectColumns 仅用于没有JOIN歧义的
// INSERT ... RETURNING 等语句。字段顺序必须与scanLessonPlanWordDocument保持一致。
const lessonPlanWordDocumentSelectColumns = `
	id,
	lesson_plan_id::text,
	import_session_id::text,
	created_by::text,
	education_domain,
	status,
	version,
	source_format,
	original_file_name,
	original_storage_key,
	original_file_sha256,
	current_storage_key,
	current_file_sha256,
	parser_version,
	structure_schema_version,
	structure_json::text,
	semantic_markdown,
	semantic_markdown_hash,
	structure_hash,
	metrics_json::text,
	warnings_json::text,
	last_change_source,
	last_changed_by::text,
	last_change_summary,
	error_message,
	generated_at,
	created_at,
	updated_at
`

// lessonPlanWordDocumentQualifiedSelectColumns 专用于带JOIN的当前Word文档查询。
//
// 所有列都显式绑定word_document表别名，避免lesson_plans等关联表中的
// id、status、version、created_at、updated_at字段造成SQLSTATE 42702歧义。
// 字段顺序必须与lessonPlanWordDocumentSelectColumns及扫描函数完全一致。
const lessonPlanWordDocumentQualifiedSelectColumns = `
	word_document.id,
	word_document.lesson_plan_id::text,
	word_document.import_session_id::text,
	word_document.created_by::text,
	word_document.education_domain,
	word_document.status,
	word_document.version,
	word_document.source_format,
	word_document.original_file_name,
	word_document.original_storage_key,
	word_document.original_file_sha256,
	word_document.current_storage_key,
	word_document.current_file_sha256,
	word_document.parser_version,
	word_document.structure_schema_version,
	word_document.structure_json::text,
	word_document.semantic_markdown,
	word_document.semantic_markdown_hash,
	word_document.structure_hash,
	word_document.metrics_json::text,
	word_document.warnings_json::text,
	word_document.last_change_source,
	word_document.last_changed_by::text,
	word_document.last_change_summary,
	word_document.error_message,
	word_document.generated_at,
	word_document.created_at,
	word_document.updated_at
`

const lessonPlanWordVersionSelectColumns = `
	version_snapshot.id,
	version_snapshot.lesson_plan_id::text,
	version_snapshot.version,
	version_snapshot.storage_key,
	version_snapshot.file_sha256,
	version_snapshot.parser_version,
	version_snapshot.structure_schema_version,
	version_snapshot.structure_json::text,
	version_snapshot.semantic_markdown,
	version_snapshot.semantic_markdown_hash,
	version_snapshot.structure_hash,
	version_snapshot.metrics_json::text,
	version_snapshot.warnings_json::text,
	version_snapshot.change_source,
	version_snapshot.changed_by::text,
	COALESCE(changed_user.display_name, ''),
	version_snapshot.change_summary,
	version_snapshot.created_at
`

func scanLessonPlanWordDocument(
	row lessonPlanWordRowScanner,
) (*models.LessonPlanWordDocument, error) {
	record := &models.LessonPlanWordDocument{}

	var importSessionID sql.NullString
	var createdBy sql.NullString
	var lastChangedBy sql.NullString

	if err := row.Scan(
		&record.ID,
		&record.LessonPlanID,
		&importSessionID,
		&createdBy,
		&record.EducationDomain,
		&record.Status,
		&record.Version,
		&record.SourceFormat,
		&record.OriginalFileName,
		&record.OriginalStorageKey,
		&record.OriginalFileSHA256,
		&record.CurrentStorageKey,
		&record.CurrentFileSHA256,
		&record.ParserVersion,
		&record.StructureSchemaVersion,
		&record.StructureJSON,
		&record.SemanticMarkdown,
		&record.SemanticMarkdownHash,
		&record.StructureHash,
		&record.MetricsJSON,
		&record.WarningsJSON,
		&record.LastChangeSource,
		&lastChangedBy,
		&record.LastChangeSummary,
		&record.ErrorMessage,
		&record.GeneratedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if importSessionID.Valid {
		value := strings.TrimSpace(importSessionID.String)
		record.ImportSessionID = &value
	}
	if createdBy.Valid {
		value := strings.TrimSpace(createdBy.String)
		record.CreatedBy = &value
	}
	if lastChangedBy.Valid {
		value := strings.TrimSpace(lastChangedBy.String)
		record.LastChangedBy = &value
	}

	return record, nil
}

func scanLessonPlanWordDocumentVersion(
	row lessonPlanWordRowScanner,
) (*models.LessonPlanWordDocumentVersion, error) {
	record := &models.LessonPlanWordDocumentVersion{}

	var changedBy sql.NullString

	if err := row.Scan(
		&record.ID,
		&record.LessonPlanID,
		&record.Version,
		&record.StorageKey,
		&record.FileSHA256,
		&record.ParserVersion,
		&record.StructureSchemaVersion,
		&record.StructureJSON,
		&record.SemanticMarkdown,
		&record.SemanticMarkdownHash,
		&record.StructureHash,
		&record.MetricsJSON,
		&record.WarningsJSON,
		&record.ChangeSource,
		&changedBy,
		&record.ChangedByName,
		&record.ChangeSummary,
		&record.CreatedAt,
	); err != nil {
		return nil, err
	}

	if changedBy.Valid {
		value := strings.TrimSpace(changedBy.String)
		record.ChangedBy = &value
	}

	return record, nil
}

// GetLessonPlanWordDocumentForOwner 读取作者本人的当前Word保真文档。
func GetLessonPlanWordDocumentForOwner(
	ctx context.Context,
	lessonPlanID string,
	ownerID string,
) (*models.LessonPlanWordDocument, error) {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	ownerID = strings.TrimSpace(ownerID)

	if lessonPlanID == "" || ownerID == "" {
		return nil, ErrLessonPlanWordDocumentNotFound
	}

	row := database.DB.QueryRow(
		ctx,
		`
SELECT `+lessonPlanWordDocumentQualifiedSelectColumns+`
FROM lesson_plan_word_documents word_document
INNER JOIN lesson_plans lesson_plan
	ON lesson_plan.id = word_document.lesson_plan_id
WHERE word_document.lesson_plan_id = $1
  AND lesson_plan.author_id = $2
  AND lesson_plan.deleted_at IS NULL
`,
		lessonPlanID,
		ownerID,
	)

	record, err := scanLessonPlanWordDocument(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanWordDocumentNotFound
		}
		return nil, fmt.Errorf(
			"查询当前Word保真文档失败: %w",
			err,
		)
	}

	return record, nil
}

// ListLessonPlanWordDocumentVersionsForOwner 查询作者本人的Word版本列表。
func ListLessonPlanWordDocumentVersionsForOwner(
	ctx context.Context,
	lessonPlanID string,
	ownerID string,
	limit int,
	offset int,
) ([]*models.LessonPlanWordDocumentVersionListItem, int, error) {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	ownerID = strings.TrimSpace(ownerID)

	if lessonPlanID == "" || ownerID == "" {
		return nil, 0, ErrLessonPlanWordDocumentNotFound
	}

	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := database.DB.QueryRow(
		ctx,
		`
SELECT COUNT(*)
FROM lesson_plan_word_document_versions version_snapshot
INNER JOIN lesson_plans lesson_plan
	ON lesson_plan.id = version_snapshot.lesson_plan_id
WHERE version_snapshot.lesson_plan_id = $1
  AND lesson_plan.author_id = $2
  AND lesson_plan.deleted_at IS NULL
`,
		lessonPlanID,
		ownerID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"查询Word版本总数失败: %w",
			err,
		)
	}

	rows, err := database.DB.Query(
		ctx,
		`
SELECT
	version_snapshot.id,
	version_snapshot.version,
	version_snapshot.parser_version,
	COALESCE(
		(version_snapshot.metrics_json ->> 'block_count')::integer,
		0
	),
	COALESCE(
		(version_snapshot.metrics_json ->> 'table_count')::integer,
		0
	),
	COALESCE(
		(version_snapshot.metrics_json ->> 'image_count')::integer,
		0
	),
	COALESCE(
		(version_snapshot.metrics_json ->> 'formula_count')::integer,
		0
	),
	jsonb_array_length(version_snapshot.warnings_json),
	char_length(version_snapshot.semantic_markdown),
	version_snapshot.change_source,
	version_snapshot.changed_by::text,
	COALESCE(changed_user.display_name, ''),
	version_snapshot.change_summary,
	version_snapshot.created_at
FROM lesson_plan_word_document_versions version_snapshot
INNER JOIN lesson_plans lesson_plan
	ON lesson_plan.id = version_snapshot.lesson_plan_id
LEFT JOIN users changed_user
	ON changed_user.id = version_snapshot.changed_by
WHERE version_snapshot.lesson_plan_id = $1
  AND lesson_plan.author_id = $2
  AND lesson_plan.deleted_at IS NULL
ORDER BY
	version_snapshot.version DESC,
	version_snapshot.created_at DESC
LIMIT $3 OFFSET $4
`,
		lessonPlanID,
		ownerID,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"查询Word版本列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.LessonPlanWordDocumentVersionListItem,
		0,
	)

	for rows.Next() {
		item := &models.LessonPlanWordDocumentVersionListItem{}

		var changedBy sql.NullString

		if err := rows.Scan(
			&item.ID,
			&item.Version,
			&item.ParserVersion,
			&item.BlockCount,
			&item.TableCount,
			&item.ImageCount,
			&item.FormulaCount,
			&item.WarningCount,
			&item.CharacterCount,
			&item.ChangeSource,
			&changedBy,
			&item.ChangedByName,
			&item.ChangeSummary,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf(
				"扫描Word版本列表失败: %w",
				err,
			)
		}

		if changedBy.Valid {
			value := strings.TrimSpace(changedBy.String)
			item.ChangedBy = &value
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"遍历Word版本列表失败: %w",
			err,
		)
	}

	return items, total, nil
}

// GetLessonPlanWordDocumentVersionForOwner 精确读取作者本人的完整Word版本。
//
// lessonPlanID同时参与过滤，防止拿其它教案的versionID跨资源读取私有文件键。
func GetLessonPlanWordDocumentVersionForOwner(
	ctx context.Context,
	lessonPlanID string,
	versionID string,
	ownerID string,
) (*models.LessonPlanWordDocumentVersion, error) {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	versionID = strings.TrimSpace(versionID)
	ownerID = strings.TrimSpace(ownerID)

	if lessonPlanID == "" || versionID == "" || ownerID == "" {
		return nil, ErrLessonPlanWordDocumentNotFound
	}

	row := database.DB.QueryRow(
		ctx,
		`
SELECT `+lessonPlanWordVersionSelectColumns+`
FROM lesson_plan_word_document_versions version_snapshot
INNER JOIN lesson_plans lesson_plan
	ON lesson_plan.id = version_snapshot.lesson_plan_id
LEFT JOIN users changed_user
	ON changed_user.id = version_snapshot.changed_by
WHERE version_snapshot.id = $1
  AND version_snapshot.lesson_plan_id = $2
  AND lesson_plan.author_id = $3
  AND lesson_plan.deleted_at IS NULL
`,
		versionID,
		lessonPlanID,
		ownerID,
	)

	record, err := scanLessonPlanWordDocumentVersion(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanWordDocumentNotFound
		}
		return nil, fmt.Errorf(
			"查询Word完整版本失败: %w",
			err,
		)
	}

	return record, nil
}
