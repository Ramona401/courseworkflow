package models

// courseware_style_studio.go — AI美术风格工作室领域模型
//
// 设计原则：
//   - 风格最终事实源是完整课程锚点IAOCI文本；
//   - 对话消息只保存形成风格索引的过程；
//   - 预览分为人物、知识对象、教学图解三种固定类型；
//   - 老师确认后，正式锚点仍写入coursewares.style_anchor_*；
//   - 已确认或归档的会话不可继续修改；
//   - 预览和确认请求都可显式携带reference_mode，
//     防止前端刚切换模式但服务端仍使用旧模式。

import "time"

// CoursewareStyleSession 风格共创会话。
type CoursewareStyleSession struct {
	ID           string `json:"id"`
	CoursewareID string `json:"courseware_id"`
	UserID       string `json:"user_id"`

	Status        string `json:"status"`
	ReferenceMode string `json:"reference_mode"`

	ReferenceAssetID *string `json:"reference_asset_id"`
	ConfirmedAssetID *string `json:"confirmed_asset_id"`

	StyleAOCIText string `json:"style_aoci_text"`
	StyleSummary  string `json:"style_summary"`

	Version     int        `json:"version"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// CoursewareStyleMessage 一条风格共创消息。
type CoursewareStyleMessage struct {
	ID           string `json:"id"`
	SessionID    string `json:"session_id"`
	CoursewareID string `json:"courseware_id"`

	Role    string `json:"role"`
	Content string `json:"content"`

	ReferenceAssetID *string `json:"reference_asset_id"`

	// StyleAOCIText通常只在assistant消息中保存，
	// 表示该轮对话完成后形成的完整IAOCI快照。
	StyleAOCIText string `json:"style_aoci_text"`

	SequenceNo int        `json:"sequence_no"`
	CreatedAt  *time.Time `json:"created_at"`
}

// CoursewareStylePreview 一张风格测试预览。
type CoursewareStylePreview struct {
	ID           string `json:"id"`
	SessionID    string `json:"session_id"`
	CoursewareID string `json:"courseware_id"`

	PreviewType string  `json:"preview_type"`
	AssetID     *string `json:"asset_id"`

	GenerationPrompt string `json:"generation_prompt"`
	Status           string `json:"status"`
	LastError        string `json:"last_error"`

	Version   int        `json:"version"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CoursewareStyleStudioState 风格工作室恢复响应。
type CoursewareStyleStudioState struct {
	Session  *CoursewareStyleSession   `json:"session"`
	Messages []*CoursewareStyleMessage `json:"messages"`
	Previews []*CoursewareStylePreview `json:"previews"`
}

// CreateCoursewareStyleSessionRequest 创建风格会话请求。
type CreateCoursewareStyleSessionRequest struct {
	ReferenceMode    string  `json:"reference_mode"`
	ReferenceAssetID *string `json:"reference_asset_id"`
}

// CoursewareStyleTurnRequest 老师发送一轮风格要求。
type CoursewareStyleTurnRequest struct {
	Content          string  `json:"content"`
	ReferenceMode    string  `json:"reference_mode"`
	ReferenceAssetID *string `json:"reference_asset_id"`
}

// GenerateCoursewareStylePreviewsRequest 生成三类预览请求。
//
// ReferenceMode为空时兼容旧客户端，继续使用会话当前模式。
// 非空时先把模式和规范化IAOCI保存到会话，再生成三类预览。
type GenerateCoursewareStylePreviewsRequest struct {
	ReferenceMode string `json:"reference_mode"`
	OperationID   string `json:"operation_id"`
}

// ConfirmCoursewareStyleSessionRequest 确认风格请求。
//
// ReferenceMode为空时兼容旧客户端。
// 新客户端应显式提交当前界面模式，后端会执行事务级来源校验。
type ConfirmCoursewareStyleSessionRequest struct {
	AssetID       string `json:"asset_id"`
	ReferenceMode string `json:"reference_mode"`
}

// 风格会话状态。
const (
	CWStyleSessionStatusDraft      = "draft"
	CWStyleSessionStatusPreviewing = "previewing"
	CWStyleSessionStatusConfirmed  = "confirmed"
	CWStyleSessionStatusArchived   = "archived"
)

// 参考图片使用模式。
const (
	CWStyleReferenceModeStyleOnly   = "style_only"
	CWStyleReferenceModeCharacter   = "style_character"
	CWStyleReferenceModeInspiration = "inspiration"
)

// 消息角色。
const (
	CWStyleMessageRoleUser      = "user"
	CWStyleMessageRoleAssistant = "assistant"
)

// 三类固定预览。
const (
	CWStylePreviewTypeCharacter = "character"
	CWStylePreviewTypeObject    = "object"
	CWStylePreviewTypeDiagram   = "diagram"
)

// 预览状态。
const (
	CWStylePreviewStatusPending    = "pending"
	CWStylePreviewStatusGenerating = "generating"
	CWStylePreviewStatusGenerated  = "generated"
	CWStylePreviewStatusFailed     = "failed"
	CWStylePreviewStatusStale      = "stale"
)

// CoursewareStylePreviewTypes 返回固定预览顺序。
var CoursewareStylePreviewTypes = []string{
	CWStylePreviewTypeCharacter,
	CWStylePreviewTypeObject,
	CWStylePreviewTypeDiagram,
}

// IsValidCWStyleSessionStatus 校验会话状态。
func IsValidCWStyleSessionStatus(value string) bool {
	switch value {
	case CWStyleSessionStatusDraft,
		CWStyleSessionStatusPreviewing,
		CWStyleSessionStatusConfirmed,
		CWStyleSessionStatusArchived:
		return true
	default:
		return false
	}
}

// IsEditableCWStyleSessionStatus 判断会话是否可继续对话或预览。
func IsEditableCWStyleSessionStatus(value string) bool {
	return value == CWStyleSessionStatusDraft ||
		value == CWStyleSessionStatusPreviewing
}

// IsValidCWStyleReferenceMode 校验参考图模式。
func IsValidCWStyleReferenceMode(value string) bool {
	switch value {
	case CWStyleReferenceModeStyleOnly,
		CWStyleReferenceModeCharacter,
		CWStyleReferenceModeInspiration:
		return true
	default:
		return false
	}
}

// IsValidCWStyleMessageRole 校验消息角色。
func IsValidCWStyleMessageRole(value string) bool {
	return value == CWStyleMessageRoleUser ||
		value == CWStyleMessageRoleAssistant
}

// IsValidCWStylePreviewType 校验预览类型。
func IsValidCWStylePreviewType(value string) bool {
	switch value {
	case CWStylePreviewTypeCharacter,
		CWStylePreviewTypeObject,
		CWStylePreviewTypeDiagram:
		return true
	default:
		return false
	}
}

// IsValidCWStylePreviewStatus 校验预览状态。
func IsValidCWStylePreviewStatus(value string) bool {
	switch value {
	case CWStylePreviewStatusPending,
		CWStylePreviewStatusGenerating,
		CWStylePreviewStatusGenerated,
		CWStylePreviewStatusFailed,
		CWStylePreviewStatusStale:
		return true
	default:
		return false
	}
}
