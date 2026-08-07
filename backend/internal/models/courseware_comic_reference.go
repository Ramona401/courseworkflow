package models

// courseware_comic_reference.go — 知识点漫画可选参考资源稳定协议
//
// 安全原则：
//   - 知识点仍是漫画项目的唯一必填来源；
//   - 教材单元、已有课件、课程大纲、文档、图片和其他文字均为可选增强；
//   - 正式资源由服务端重新读取并保存创建时快照；
//   - DOCX和文字型PDF只保存浏览器提取文字，不保存原始二进制文件；
//   - 参考图片复用courseware_assets，并且必须属于当前课件；
//   - 浏览器视图不返回content_text和summary_text；
//   - AI上下文由服务层按数量和字符预算重新裁剪。

import (
	"strings"
	"time"
)

// 知识点漫画参考资源类型。
const (
	CWComicReferenceTextbookUnit =
		"textbook_unit"

	CWComicReferenceCourseware =
		"courseware"

	CWComicReferenceCourseOutline =
		"course_outline"

	CWComicReferenceUploadedDocument =
		"uploaded_document"

	CWComicReferenceUploadedImage =
		"uploaded_image"

	CWComicReferenceOtherText =
		"other_text"
)

// CoursewareComicReferenceResource
// 对应courseware_comic_reference_resources表。
//
// ContentText和SummaryText只供后端可信快照与AI上下文使用，
// 不得直接把本数据库实体作为浏览器响应。
type CoursewareComicReferenceResource struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	CoursewareID string `json:"courseware_id"`
	CreatedBy    string `json:"created_by"`

	ResourceType string `json:"resource_type"`

	SourceID *string `json:"source_id"`
	AssetID  *string `json:"asset_id"`

	Title    string `json:"title"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`

	ContentText string `json:"-"`
	SummaryText string `json:"-"`

	OriginalLength int `json:"original_length"`
	SummaryLength  int `json:"summary_length"`
	SortOrder      int `json:"sort_order"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CreateCoursewareComicReferenceRequest
// 新增一条项目参考资源。
//
// 正式来源只接收SourceID，标题和正文由服务端重新读取。
// 文档与其他文字允许提交浏览器提取或教师粘贴的正文。
// 参考图片只接收同课件AssetID，图片URL由服务端重新解析。
type CreateCoursewareComicReferenceRequest struct {
	ResourceType string `json:"resource_type"`

	SourceID string `json:"source_id"`
	AssetID  string `json:"asset_id"`

	Title    string `json:"title"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`

	ContentText string `json:"content_text"`
	SummaryText string `json:"summary_text"`

	SortOrder int `json:"sort_order"`
}

// CoursewareComicReferenceResourceView
// 是浏览器安全的参考资源视图。
//
// 不包含正文和压缩摘要，仅返回列表展示、图片预览和删除定位所需元数据。
type CoursewareComicReferenceResourceView struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	CoursewareID string `json:"courseware_id"`

	ResourceType string `json:"resource_type"`

	SourceID *string `json:"source_id"`
	AssetID  *string `json:"asset_id"`

	Title    string `json:"title"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`

	OriginalLength int `json:"original_length"`
	SummaryLength  int `json:"summary_length"`
	SortOrder      int `json:"sort_order"`

	ImageURL string `json:"image_url"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CoursewareComicReferenceResourceListView
// 返回当前项目的安全参考资源列表。
type CoursewareComicReferenceResourceListView struct {
	References []*CoursewareComicReferenceResourceView `json:"references"`
	Total      int                                      `json:"total"`
}

// IsValidCWComicReferenceResourceType
// 校验参考资源类型。
func IsValidCWComicReferenceResourceType(
	value string,
) bool {
	switch strings.TrimSpace(value) {
	case CWComicReferenceTextbookUnit,
		CWComicReferenceCourseware,
		CWComicReferenceCourseOutline,
		CWComicReferenceUploadedDocument,
		CWComicReferenceUploadedImage,
		CWComicReferenceOtherText:
		return true

	default:
		return false
	}
}

// IsCWComicReferenceOfficialSource
// 判断是否属于必须由服务端重新读取的正式来源。
func IsCWComicReferenceOfficialSource(
	value string,
) bool {
	switch strings.TrimSpace(value) {
	case CWComicReferenceTextbookUnit,
		CWComicReferenceCourseware,
		CWComicReferenceCourseOutline:
		return true

	default:
		return false
	}
}
