package models

// lesson_plan_word.go — 原格式Word教案领域模型
//
// 本文件只定义Word保真导入、当前文档和不可变版本的数据协议，
// 不负责DOCX解析、文件落盘、权限校验或HTTP响应脱敏。
//
// 双层教案原则：
//   - lesson_plans.content_markdown继续作为AI评审、索引和课件生成的语义事实源；
//   - LessonPlanWordDocument保存原始Word版式、内容块、图片和公式关系；
//   - 两层内容必须在同一业务事务中同步，发生漂移时Word文档标记为stale；
//   - 浏览器不得直接获得私有storage_key或服务端物理路径。

import "time"

const (
        LessonPlanWordMimeDOCX =
                "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

        LessonPlanWordStructureSchemaVersion = 1

        LessonPlanWordImportStatusUploaded  = "uploaded"
        LessonPlanWordImportStatusParsed    = "parsed"
        LessonPlanWordImportStatusConfirmed = "confirmed"
        LessonPlanWordImportStatusFailed    = "failed"
        LessonPlanWordImportStatusExpired   = "expired"

        LessonPlanWordDocumentStatusActive = "active"
        LessonPlanWordDocumentStatusStale  = "stale"
        LessonPlanWordDocumentStatusFailed = "failed"

        LessonPlanWordChangeSourceImport  = "import"
        LessonPlanWordChangeSourceManual  = "manual"
        LessonPlanWordChangeSourceAI      = "ai"
        LessonPlanWordChangeSourceRestore = "restore"
        LessonPlanWordChangeSourceSystem  = "system"
)

// IsValidLessonPlanWordImportStatus 判断Word导入会话状态是否合法。
func IsValidLessonPlanWordImportStatus(status string) bool {
        switch status {
        case LessonPlanWordImportStatusUploaded,
                LessonPlanWordImportStatusParsed,
                LessonPlanWordImportStatusConfirmed,
                LessonPlanWordImportStatusFailed,
                LessonPlanWordImportStatusExpired:
                return true
        default:
                return false
        }
}

// IsValidLessonPlanWordDocumentStatus 判断当前Word文档状态是否合法。
func IsValidLessonPlanWordDocumentStatus(status string) bool {
        switch status {
        case LessonPlanWordDocumentStatusActive,
                LessonPlanWordDocumentStatusStale,
                LessonPlanWordDocumentStatusFailed:
                return true
        default:
                return false
        }
}

// IsValidLessonPlanWordChangeSource 判断Word版本变更来源是否合法。
func IsValidLessonPlanWordChangeSource(source string) bool {
        switch source {
        case LessonPlanWordChangeSourceImport,
                LessonPlanWordChangeSourceManual,
                LessonPlanWordChangeSourceAI,
                LessonPlanWordChangeSourceRestore,
                LessonPlanWordChangeSourceSystem:
                return true
        default:
                return false
        }
}

// LessonPlanWordImportSession 是原DOCX上传后的短时导入会话。
//
// StorageKey、文件哈希和完整结构均属于后端内部数据，
// 后续HTTP层必须构造单独的浏览器安全预览视图。
type LessonPlanWordImportSession struct {
        ID                     string     `json:"id"`
        CreatedBy              string     `json:"created_by"`
        EducationDomain        string     `json:"education_domain"`
        Status                 string     `json:"status"`
        OriginalFileName       string     `json:"original_file_name"`
        StorageKey             string     `json:"-"`
        FileSize               int64      `json:"file_size"`
        MimeType               string     `json:"mime_type"`
        FileSHA256             string     `json:"-"`
        ParserVersion          string     `json:"parser_version"`
        StructureSchemaVersion int        `json:"structure_schema_version"`
        StructureJSON          string     `json:"-"`
        SemanticMarkdown       string     `json:"-"`
        SemanticMarkdownHash   string     `json:"-"`
        MetricsJSON            string     `json:"-"`
        WarningsJSON           string     `json:"-"`
        ErrorMessage           string     `json:"error_message,omitempty"`
        LessonPlanID           *string    `json:"lesson_plan_id,omitempty"`
        ExpiresAt              time.Time  `json:"expires_at"`
        ParsedAt               *time.Time `json:"parsed_at,omitempty"`
        ConfirmedAt            *time.Time `json:"confirmed_at,omitempty"`
        CreatedAt              time.Time  `json:"created_at"`
        UpdatedAt              time.Time  `json:"updated_at"`
}

// LessonPlanWordDocument 是每份正式教案当前唯一的Word保真文档。
type LessonPlanWordDocument struct {
        ID                     string     `json:"id"`
        LessonPlanID           string     `json:"lesson_plan_id"`
        ImportSessionID        *string    `json:"import_session_id,omitempty"`
        CreatedBy              *string    `json:"created_by,omitempty"`
        EducationDomain        string     `json:"education_domain"`
        Status                 string     `json:"status"`
        Version                int        `json:"version"`
        SourceFormat           string     `json:"source_format"`
        OriginalFileName       string     `json:"original_file_name"`
        OriginalStorageKey     string     `json:"-"`
        OriginalFileSHA256     string     `json:"-"`
        CurrentStorageKey      string     `json:"-"`
        CurrentFileSHA256      string     `json:"-"`
        ParserVersion          string     `json:"parser_version"`
        StructureSchemaVersion int        `json:"structure_schema_version"`
        StructureJSON          string     `json:"-"`
        SemanticMarkdown       string     `json:"-"`
        SemanticMarkdownHash   string     `json:"-"`
        StructureHash          string     `json:"-"`
        MetricsJSON            string     `json:"-"`
        WarningsJSON           string     `json:"-"`
        LastChangeSource       string     `json:"last_change_source"`
        LastChangedBy          *string    `json:"last_changed_by,omitempty"`
        LastChangeSummary      string     `json:"last_change_summary"`
        ErrorMessage           string     `json:"error_message,omitempty"`
        GeneratedAt            time.Time  `json:"generated_at"`
        CreatedAt              time.Time  `json:"created_at"`
        UpdatedAt              time.Time  `json:"updated_at"`
}

// LessonPlanWordDocumentVersion 是Word文档的不可变完整历史版本。
type LessonPlanWordDocumentVersion struct {
        ID                     string    `json:"id"`
        LessonPlanID           string    `json:"lesson_plan_id"`
        Version                int       `json:"version"`
        StorageKey             string    `json:"-"`
        FileSHA256             string    `json:"-"`
        ParserVersion          string    `json:"parser_version"`
        StructureSchemaVersion int       `json:"structure_schema_version"`
        StructureJSON          string    `json:"-"`
        SemanticMarkdown       string    `json:"-"`
        SemanticMarkdownHash   string    `json:"-"`
        StructureHash          string    `json:"-"`
        MetricsJSON            string    `json:"-"`
        WarningsJSON           string    `json:"-"`
        ChangeSource           string    `json:"change_source"`
        ChangedBy              *string   `json:"changed_by,omitempty"`
        ChangedByName          string    `json:"changed_by_name,omitempty"`
        ChangeSummary          string    `json:"change_summary"`
        CreatedAt              time.Time `json:"created_at"`
}

// LessonPlanWordDocumentVersionListItem 是版本列表轻量条目。
// 列表不返回DOCX私有路径、完整结构或完整语义正文。
type LessonPlanWordDocumentVersionListItem struct {
        ID                string    `json:"id"`
        Version           int       `json:"version"`
        ParserVersion     string    `json:"parser_version"`
        BlockCount        int       `json:"block_count"`
        TableCount        int       `json:"table_count"`
        ImageCount        int       `json:"image_count"`
        FormulaCount      int       `json:"formula_count"`
        WarningCount      int       `json:"warning_count"`
        CharacterCount    int       `json:"character_count"`
        ChangeSource      string    `json:"change_source"`
        ChangedBy         *string   `json:"changed_by,omitempty"`
        ChangedByName     string    `json:"changed_by_name,omitempty"`
        ChangeSummary     string    `json:"change_summary"`
        CreatedAt         time.Time `json:"created_at"`
}

// CreateLessonPlanWordImportSessionInput 是文件已安全落盘后的会话创建输入。
//
// StorageKey必须由服务端生成，不能来自浏览器。
// FileSHA256必须根据实际落盘文件计算，不能信任multipart请求字段。
type CreateLessonPlanWordImportSessionInput struct {
        CreatedBy              string
        EducationDomain        string
        OriginalFileName       string
        StorageKey             string
        FileSize               int64
        MimeType               string
        FileSHA256             string
        StructureSchemaVersion int
        ExpiresAt              time.Time
}

// LessonPlanWordParsedPayload 是DOCX解析成功后的完整结构结果。
type LessonPlanWordParsedPayload struct {
        ParserVersion          string
        StructureSchemaVersion int
        StructureJSON          string
        SemanticMarkdown       string
        SemanticMarkdownHash   string
        MetricsJSON            string
        WarningsJSON           string
}

// ConfirmLessonPlanWordImportInput 把已解析导入会话绑定到刚创建的正式教案。
//
// PermanentStorageKey必须指向已经复制或原子移动完成的私有不可变版本文件。
// PermanentFileSHA256必须与导入会话原文件哈希一致。
type ConfirmLessonPlanWordImportInput struct {
        ImportSessionID     string
        LessonPlanID        string
        OwnerID             string
        EducationDomain     string
        PermanentStorageKey string
        PermanentFileSHA256 string
        ChangeSummary       string
}
