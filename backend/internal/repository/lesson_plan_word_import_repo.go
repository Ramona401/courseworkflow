package repository

// lesson_plan_word_import_repo.go — 原格式Word导入会话与确认绑定仓储
//
// 职责：
//   1. 创建和读取DOCX短时导入会话；
//   2. 原子保存解析成功或失败状态；
//   3. 将可信解析会话与新建教案绑定；
//   4. 创建当前Word文档并依赖数据库触发器生成不可变版本1；
//   5. 正式导入补偿失败后，把失去教案绑定的会话安全恢复为parsed。
//
// 文件落盘、DOCX解析、物理文件复制和清理由Service层负责。
// 本仓储不接收浏览器提供的文件路径、文件哈希或语义正文。

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

const lessonPlanWordImportSelectColumns = `
        id,
        created_by::text,
        education_domain,
        status,
        original_file_name,
        storage_key,
        file_size,
        mime_type,
        file_sha256,
        parser_version,
        structure_schema_version,
        structure_json::text,
        semantic_markdown,
        semantic_markdown_hash,
        metrics_json::text,
        warnings_json::text,
        error_message,
        lesson_plan_id::text,
        expires_at,
        parsed_at,
        confirmed_at,
        created_at,
        updated_at
`

func scanLessonPlanWordImportSession(
	row lessonPlanWordRowScanner,
) (*models.LessonPlanWordImportSession, error) {
	record := &models.LessonPlanWordImportSession{}

	var lessonPlanID sql.NullString
	var parsedAt sql.NullTime
	var confirmedAt sql.NullTime

	if err := row.Scan(
		&record.ID,
		&record.CreatedBy,
		&record.EducationDomain,
		&record.Status,
		&record.OriginalFileName,
		&record.StorageKey,
		&record.FileSize,
		&record.MimeType,
		&record.FileSHA256,
		&record.ParserVersion,
		&record.StructureSchemaVersion,
		&record.StructureJSON,
		&record.SemanticMarkdown,
		&record.SemanticMarkdownHash,
		&record.MetricsJSON,
		&record.WarningsJSON,
		&record.ErrorMessage,
		&lessonPlanID,
		&record.ExpiresAt,
		&parsedAt,
		&confirmedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if lessonPlanID.Valid {
		value := strings.TrimSpace(lessonPlanID.String)
		record.LessonPlanID = &value
	}
	if parsedAt.Valid {
		value := parsedAt.Time
		record.ParsedAt = &value
	}
	if confirmedAt.Valid {
		value := confirmedAt.Time
		record.ConfirmedAt = &value
	}

	return record, nil
}

func normalizeLessonPlanWordJSONObject(
	raw string,
	fallback string,
) (string, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		normalized = fallback
	}

	var value map[string]any
	if err := json.Unmarshal(
		[]byte(normalized),
		&value,
	); err != nil {
		return "", fmt.Errorf(
			"%w: 需要JSON对象",
			ErrLessonPlanWordInputInvalid,
		)
	}

	return normalized, nil
}

func normalizeLessonPlanWordJSONArray(
	raw string,
) (string, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		normalized = "[]"
	}

	var value []any
	if err := json.Unmarshal(
		[]byte(normalized),
		&value,
	); err != nil {
		return "", fmt.Errorf(
			"%w: warnings_json需要JSON数组",
			ErrLessonPlanWordInputInvalid,
		)
	}

	return normalized, nil
}

// CreateLessonPlanWordImportSession 创建DOCX原文件已安全落盘后的短时会话。
func CreateLessonPlanWordImportSession(
	ctx context.Context,
	input models.CreateLessonPlanWordImportSessionInput,
) (*models.LessonPlanWordImportSession, error) {
	input.CreatedBy = strings.TrimSpace(input.CreatedBy)
	input.EducationDomain = strings.ToLower(
		strings.TrimSpace(input.EducationDomain),
	)
	input.OriginalFileName = strings.TrimSpace(
		input.OriginalFileName,
	)
	input.StorageKey = strings.TrimSpace(input.StorageKey)
	input.MimeType = strings.TrimSpace(input.MimeType)
	input.FileSHA256 = strings.TrimSpace(input.FileSHA256)

	if input.CreatedBy == "" ||
		!models.IsTeachingEducationDomain(
			input.EducationDomain,
		) ||
		input.OriginalFileName == "" ||
		input.StorageKey == "" ||
		input.FileSize <= 0 ||
		len(input.FileSHA256) != 64 {
		return nil, ErrLessonPlanWordInputInvalid
	}

	if input.MimeType == "" {
		input.MimeType = models.LessonPlanWordMimeDOCX
	}
	if input.MimeType != models.LessonPlanWordMimeDOCX {
		return nil, ErrLessonPlanWordInputInvalid
	}
	if input.StructureSchemaVersion <= 0 {
		input.StructureSchemaVersion =
			models.LessonPlanWordStructureSchemaVersion
	}
	if input.ExpiresAt.IsZero() {
		input.ExpiresAt = time.Now().Add(24 * time.Hour)
	}

	row := database.DB.QueryRow(
		ctx,
		`
INSERT INTO lesson_plan_word_import_sessions (
        created_by,
        education_domain,
        status,
        original_file_name,
        storage_key,
        file_size,
        mime_type,
        file_sha256,
        structure_schema_version,
        expires_at
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
        $10
)
RETURNING `+lessonPlanWordImportSelectColumns,
		input.CreatedBy,
		input.EducationDomain,
		models.LessonPlanWordImportStatusUploaded,
		input.OriginalFileName,
		input.StorageKey,
		input.FileSize,
		input.MimeType,
		input.FileSHA256,
		input.StructureSchemaVersion,
		input.ExpiresAt,
	)

	record, err := scanLessonPlanWordImportSession(row)
	if err != nil {
		return nil, fmt.Errorf(
			"创建Word导入会话失败: %w",
			err,
		)
	}

	return record, nil
}

// GetLessonPlanWordImportSessionForUser 按会话和创建者读取导入记录。
//
// 不存在、越权或创建者不匹配统一返回NotFound，防止跨用户枚举会话ID。
func GetLessonPlanWordImportSessionForUser(
	ctx context.Context,
	sessionID string,
	userID string,
) (*models.LessonPlanWordImportSession, error) {
	sessionID = strings.TrimSpace(sessionID)
	userID = strings.TrimSpace(userID)

	if sessionID == "" || userID == "" {
		return nil, ErrLessonPlanWordImportNotFound
	}

	row := database.DB.QueryRow(
		ctx,
		`
SELECT `+lessonPlanWordImportSelectColumns+`
FROM lesson_plan_word_import_sessions
WHERE id = $1
  AND created_by = $2
`,
		sessionID,
		userID,
	)

	record, err := scanLessonPlanWordImportSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanWordImportNotFound
		}
		return nil, fmt.Errorf(
			"查询Word导入会话失败: %w",
			err,
		)
	}

	return record, nil
}

// MarkLessonPlanWordImportSessionParsed 原子保存DOCX结构解析结果。
//
// 只有未过期uploaded会话可以进入parsed，防止并发解析覆盖已确认结果。
func MarkLessonPlanWordImportSessionParsed(
	ctx context.Context,
	sessionID string,
	userID string,
	expectedFileSHA256 string,
	payload models.LessonPlanWordParsedPayload,
) (*models.LessonPlanWordImportSession, error) {
	sessionID = strings.TrimSpace(sessionID)
	userID = strings.TrimSpace(userID)
	expectedFileSHA256 = strings.TrimSpace(expectedFileSHA256)
	payload.ParserVersion = strings.TrimSpace(payload.ParserVersion)
	payload.SemanticMarkdown = strings.TrimSpace(
		payload.SemanticMarkdown,
	)
	payload.SemanticMarkdownHash = strings.TrimSpace(
		payload.SemanticMarkdownHash,
	)

	structureJSON, err := normalizeLessonPlanWordJSONObject(
		payload.StructureJSON,
		"{}",
	)
	if err != nil {
		return nil, err
	}

	metricsJSON, err := normalizeLessonPlanWordJSONObject(
		payload.MetricsJSON,
		"{}",
	)
	if err != nil {
		return nil, err
	}

	warningsJSON, err := normalizeLessonPlanWordJSONArray(
		payload.WarningsJSON,
	)
	if err != nil {
		return nil, err
	}

	if sessionID == "" ||
		userID == "" ||
		len(expectedFileSHA256) != 64 ||
		payload.ParserVersion == "" ||
		payload.SemanticMarkdown == "" ||
		len(payload.SemanticMarkdownHash) != 64 ||
		structureJSON == "{}" {
		return nil, ErrLessonPlanWordInputInvalid
	}

	if payload.StructureSchemaVersion <= 0 {
		payload.StructureSchemaVersion =
			models.LessonPlanWordStructureSchemaVersion
	}

	row := database.DB.QueryRow(
		ctx,
		`
UPDATE lesson_plan_word_import_sessions
SET
        status = $1,
        parser_version = $2,
        structure_schema_version = $3,
        structure_json = $4::jsonb,
        semantic_markdown = $5,
        semantic_markdown_hash = $6,
        metrics_json = $7::jsonb,
        warnings_json = $8::jsonb,
        error_message = '',
        parsed_at = NOW()
WHERE id = $9
  AND created_by = $10
  AND file_sha256 = $11
  AND status = $12
  AND expires_at > NOW()
RETURNING `+lessonPlanWordImportSelectColumns,
		models.LessonPlanWordImportStatusParsed,
		payload.ParserVersion,
		payload.StructureSchemaVersion,
		structureJSON,
		payload.SemanticMarkdown,
		payload.SemanticMarkdownHash,
		metricsJSON,
		warningsJSON,
		sessionID,
		userID,
		expectedFileSHA256,
		models.LessonPlanWordImportStatusUploaded,
	)

	record, scanErr := scanLessonPlanWordImportSession(row)
	if scanErr != nil {
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return nil, ErrLessonPlanWordImportConflict
		}
		return nil, fmt.Errorf(
			"保存Word解析结果失败: %w",
			scanErr,
		)
	}

	return record, nil
}

// MarkLessonPlanWordImportSessionFailed 保存安全、限长的解析失败说明。
func MarkLessonPlanWordImportSessionFailed(
	ctx context.Context,
	sessionID string,
	userID string,
	errorMessage string,
) error {
	sessionID = strings.TrimSpace(sessionID)
	userID = strings.TrimSpace(userID)
	errorMessage = strings.TrimSpace(errorMessage)

	if sessionID == "" || userID == "" || errorMessage == "" {
		return ErrLessonPlanWordInputInvalid
	}

	if len([]rune(errorMessage)) > 4000 {
		errorMessage = string([]rune(errorMessage)[:4000])
	}

	result, err := database.DB.Exec(
		ctx,
		`
UPDATE lesson_plan_word_import_sessions
SET
        status = $1,
        error_message = $2
WHERE id = $3
  AND created_by = $4
  AND status = $5
  AND expires_at > NOW()
`,
		models.LessonPlanWordImportStatusFailed,
		errorMessage,
		sessionID,
		userID,
		models.LessonPlanWordImportStatusUploaded,
	)
	if err != nil {
		return fmt.Errorf(
			"保存Word导入失败状态失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return ErrLessonPlanWordImportConflict
	}

	return nil
}

// ConfirmLessonPlanWordImport 原子绑定已解析会话与新建教案。
//
// 调用前Service必须已经把导入文件复制或原子移动到正式私有版本路径，
// 并重新计算PermanentFileSHA256。
// 事务内再次验证作者、教育域、教案状态和语义正文完全一致。
func ConfirmLessonPlanWordImport(
	ctx context.Context,
	input models.ConfirmLessonPlanWordImportInput,
) (*models.LessonPlanWordDocument, error) {
	input.ImportSessionID = strings.TrimSpace(
		input.ImportSessionID,
	)
	input.LessonPlanID = strings.TrimSpace(input.LessonPlanID)
	input.OwnerID = strings.TrimSpace(input.OwnerID)
	input.EducationDomain = strings.ToLower(
		strings.TrimSpace(input.EducationDomain),
	)
	input.PermanentStorageKey = strings.TrimSpace(
		input.PermanentStorageKey,
	)
	input.PermanentFileSHA256 = strings.TrimSpace(
		input.PermanentFileSHA256,
	)
	input.ChangeSummary = strings.TrimSpace(
		input.ChangeSummary,
	)

	if input.ImportSessionID == "" ||
		input.LessonPlanID == "" ||
		input.OwnerID == "" ||
		!models.IsTeachingEducationDomain(
			input.EducationDomain,
		) ||
		input.PermanentStorageKey == "" ||
		len(input.PermanentFileSHA256) != 64 {
		return nil, ErrLessonPlanWordInputInvalid
	}

	if input.ChangeSummary == "" {
		input.ChangeSummary =
			"保留原Word格式导入并创建首个版本"
	}
	if len([]rune(input.ChangeSummary)) > 500 {
		input.ChangeSummary =
			string([]rune(input.ChangeSummary)[:500])
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始确认Word导入事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	sessionRow := tx.QueryRow(
		ctx,
		`
SELECT `+lessonPlanWordImportSelectColumns+`
FROM lesson_plan_word_import_sessions
WHERE id = $1
  AND created_by = $2
FOR UPDATE
`,
		input.ImportSessionID,
		input.OwnerID,
	)

	importSession, err := scanLessonPlanWordImportSession(
		sessionRow,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanWordImportNotFound
		}
		return nil, fmt.Errorf(
			"锁定Word导入会话失败: %w",
			err,
		)
	}

	if importSession.Status !=
		models.LessonPlanWordImportStatusParsed ||
		!importSession.ExpiresAt.After(time.Now()) ||
		importSession.EducationDomain !=
			input.EducationDomain ||
		importSession.FileSHA256 !=
			input.PermanentFileSHA256 {
		return nil, ErrLessonPlanWordImportConflict
	}

	structureDigest := sha256.Sum256(
		[]byte(
			importSession.StructureJSON,
		),
	)
	structureHash := hex.EncodeToString(
		structureDigest[:],
	)

	var (
		storedAuthorID   string
		storedDomain     string
		storedStatus     string
		storedVisibility string
		storedContent    string
		storedDeletedAt  sql.NullTime
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
		&storedAuthorID,
		&storedDomain,
		&storedStatus,
		&storedVisibility,
		&storedContent,
		&storedDeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanNotFound
		}
		return nil, fmt.Errorf(
			"锁定Word导入目标教案失败: %w",
			err,
		)
	}

	if strings.TrimSpace(storedAuthorID) != input.OwnerID ||
		storedDomain != input.EducationDomain ||
		storedStatus != models.LPStatusDraft ||
		storedVisibility != models.LPVisibilityPersonal ||
		storedDeletedAt.Valid ||
		storedContent != importSession.SemanticMarkdown {
		return nil, ErrLessonPlanWordDocumentConflict
	}

	documentRow := tx.QueryRow(
		ctx,
		`
INSERT INTO lesson_plan_word_documents (
        lesson_plan_id,
        import_session_id,
        created_by,
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
        structure_json,
        semantic_markdown,
        semantic_markdown_hash,
        structure_hash,
        metrics_json,
        warnings_json,
        last_change_source,
        last_changed_by,
        last_change_summary,
        error_message,
        generated_at
)
VALUES (
        $1,
        $2,
        $3,
        $4,
        $5,
        1,
        'docx',
        $6,
        $7,
        $8,
        $7,
        $8,
        $9,
        $10,
        $11::jsonb,
        $12,
        $13,
        $14,
        $15::jsonb,
        $16::jsonb,
        $17,
        $3,
        $18,
        '',
        COALESCE($19, NOW())
)
RETURNING `+lessonPlanWordDocumentSelectColumns,
		input.LessonPlanID,
		input.ImportSessionID,
		input.OwnerID,
		input.EducationDomain,
		models.LessonPlanWordDocumentStatusActive,
		importSession.OriginalFileName,
		input.PermanentStorageKey,
		input.PermanentFileSHA256,
		importSession.ParserVersion,
		importSession.StructureSchemaVersion,
		importSession.StructureJSON,
		importSession.SemanticMarkdown,
		importSession.SemanticMarkdownHash,
		structureHash,
		importSession.MetricsJSON,
		importSession.WarningsJSON,
		models.LessonPlanWordChangeSourceImport,
		input.ChangeSummary,
		importSession.ParsedAt,
	)

	document, err := scanLessonPlanWordDocument(documentRow)
	if err != nil {
		return nil, fmt.Errorf(
			"创建当前Word保真文档失败: %w",
			err,
		)
	}

	result, err := tx.Exec(
		ctx,
		`
UPDATE lesson_plan_word_import_sessions
SET
        status = $1,
        lesson_plan_id = $2,
        confirmed_at = NOW(),
        error_message = ''
WHERE id = $3
  AND created_by = $4
  AND status = $5
  AND expires_at > NOW()
`,
		models.LessonPlanWordImportStatusConfirmed,
		input.LessonPlanID,
		input.ImportSessionID,
		input.OwnerID,
		models.LessonPlanWordImportStatusParsed,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"确认Word导入会话失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return nil, ErrLessonPlanWordImportConflict
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交确认Word导入事务失败: %w",
			err,
		)
	}

	return document, nil
}

// ResetConfirmedLessonPlanWordImportSession 在正式教案补偿删除成功后，
// 确保已经解除lesson_plan_id绑定的导入会话处于可解释终态。
//
// 数据库解除绑定触发器会优先把会话变为：
//   - 未过期：parsed，可由老师重新确认；
//   - 已过期：expired，不再允许确认。
//
// 本函数同时兼容迁移前后的调用顺序：即使触发器已经完成状态转换，
// 再次调用仍会幂等成功；正常绑定正式教案的会话不会被修改。
func ResetConfirmedLessonPlanWordImportSession(
	ctx context.Context,
	sessionID string,
	userID string,
) error {
	sessionID = strings.TrimSpace(
		sessionID,
	)
	userID = strings.TrimSpace(
		userID,
	)

	if sessionID == "" ||
		userID == "" {
		return ErrLessonPlanWordInputInvalid
	}

	result, err := database.DB.Exec(
		ctx,
		`
UPDATE lesson_plan_word_import_sessions
SET
	status =
		CASE
			WHEN expires_at > NOW() THEN $1
			ELSE $2
		END,
	lesson_plan_id = NULL,
	confirmed_at = NULL,
	error_message = ''
WHERE id = $3
  AND created_by = $4
  AND lesson_plan_id IS NULL
  AND status IN (
	  $5,
	  $1,
	  $2
  )
`,
		models.LessonPlanWordImportStatusParsed,
		models.LessonPlanWordImportStatusExpired,
		sessionID,
		userID,
		models.LessonPlanWordImportStatusConfirmed,
	)
	if err != nil {
		return fmt.Errorf(
			"恢复Word导入会话失败: %w",
			err,
		)
	}

	if result.RowsAffected() != 1 {
		return ErrLessonPlanWordImportConflict
	}

	return nil
}
