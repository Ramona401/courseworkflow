package models

// textbook.go — 课本页面图片数据模型
//
// 迭代7新增：老师上传课本真实图片，AI识别后用于精准备课
//   - TextbookPage 主模型（对应 textbook_pages 表）
//   - 上传/查询/更新请求响应结构体
//   - 可见范围常量

import "time"

// ==================== 课本页面主模型 ====================

// TextbookPage 课本页面图片（对应 textbook_pages 表）
type TextbookPage struct {
	ID           string     `json:"id"`            // UUID主键
	Subject      string     `json:"subject"`       // 学科
	GradeRange   string     `json:"grade_range"`   // 年级范围
	TextbookName string     `json:"textbook_name"` // 教材名称
	Chapter      string     `json:"chapter"`       // 章节名称
	PageNumber   int        `json:"page_number"`   // 页码
	FileName     string     `json:"file_name"`     // 原始文件名
	FilePath     string     `json:"file_path"`     // 服务器存储路径
	FileSize     int64      `json:"file_size"`     // 文件大小（字节）
	MimeType     string     `json:"mime_type"`     // MIME类型
	OCRText      string     `json:"ocr_text"`      // AI识别的文字内容缓存
	OCRModel     string     `json:"ocr_model"`     // OCR使用的模型
	OCRAt        *time.Time `json:"ocr_at"`        // OCR时间
	Description  string     `json:"description"`   // 老师补充的描述
	Tags         string     `json:"tags"`          // 标签（JSONB字符串）
	Scope        string     `json:"scope"`         // 可见范围
	ScopeRefID   *string    `json:"scope_ref_id"`  // 范围引用ID
	UploadedBy   string     `json:"uploaded_by"`   // 上传者ID
	UsageCount   int        `json:"usage_count"`   // 被引用次数
	Status       string     `json:"status"`        // active/archived
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// ==================== 可见范围常量 ====================

const (
	TextbookScopePersonal = "personal" // 仅自己可见
	TextbookScopeGroup    = "group"    // 教研组可见
	TextbookScopeSchool   = "school"   // 全校可见
	TextbookScopePublic   = "public"   // 所有人可见
)

// ==================== 请求结构体 ====================

// UploadTextbookRequest 上传课本图片的元数据（随multipart表单提交）
// 图片文件通过 multipart/form-data 的 "file" 字段上传
type UploadTextbookRequest struct {
	Subject      string `json:"subject"`       // 学科（必填）
	GradeRange   string `json:"grade_range"`   // 年级（必填）
	TextbookName string `json:"textbook_name"` // 教材名称（必填）
	Chapter      string `json:"chapter"`       // 章节（可选）
	PageNumber   int    `json:"page_number"`   // 页码（可选）
	Description  string `json:"description"`   // 描述（可选）
	Scope        string `json:"scope"`         // 可见范围（默认personal）
	ScopeRefID   string `json:"scope_ref_id"`  // 范围引用ID（可选）
}

// UpdateTextbookRequest 更新课本图片元数据
type UpdateTextbookRequest struct {
	Chapter      string `json:"chapter"`       // 章节
	PageNumber   int    `json:"page_number"`   // 页码
	Description  string `json:"description"`   // 描述
	Tags         string `json:"tags"`          // 标签JSON
	Scope        string `json:"scope"`         // 可见范围
	ScopeRefID   string `json:"scope_ref_id"`  // 范围引用ID
}

// ShareTextbookRequest 共享课本图片
type ShareTextbookRequest struct {
	Scope      string `json:"scope"`        // group/school/public
	ScopeRefID string `json:"scope_ref_id"` // 教研组ID/学校ID
}

// ==================== 响应结构体 ====================

// TextbookListItem 列表项（不含OCR大文本）
type TextbookListItem struct {
	ID           string     `json:"id"`
	Subject      string     `json:"subject"`
	GradeRange   string     `json:"grade_range"`
	TextbookName string     `json:"textbook_name"`
	Chapter      string     `json:"chapter"`
	PageNumber   int        `json:"page_number"`
	FileName     string     `json:"file_name"`
	FileSize     int64      `json:"file_size"`
	MimeType     string     `json:"mime_type"`
	HasOCR       bool       `json:"has_ocr"`       // 是否已有OCR结果
	Description  string     `json:"description"`
	Scope        string     `json:"scope"`
	ScopeName    string     `json:"scope_name"`
	UploadedBy   string     `json:"uploaded_by"`
	UploaderName string     `json:"uploader_name"` // 上传者显示名
	UsageCount   int        `json:"usage_count"`
	ImageURL     string     `json:"image_url"`     // 图片访问URL
	CreatedAt    *time.Time `json:"created_at"`
}

// TextbookDetailResponse 详情响应（含OCR文本）
type TextbookDetailResponse struct {
	TextbookPage
	UploaderName string `json:"uploader_name"` // 上传者显示名
	ImageURL     string `json:"image_url"`     // 图片访问URL
	HasOCR       bool   `json:"has_ocr"`       // 是否已有OCR结果
}

// TextbookListResponse 列表响应
type TextbookListResponse struct {
	Pages []*TextbookListItem `json:"pages"`
	Total int                 `json:"total"`
}

// TextbookScopeNameMap 可见范围中文映射
var TextbookScopeNameMap = map[string]string{
	TextbookScopePersonal: "个人",
	TextbookScopeGroup:    "教研组",
	TextbookScopeSchool:   "全校",
	TextbookScopePublic:   "所有人",
}
