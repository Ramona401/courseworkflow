package models

// assistant_runtime.go
//
// 本文件定义教学智能体匿名运行会话、单轮对话和使用流水协议。
//
// 安全边界：
//   1. 数据库哈希、教师ID和学校ID只存在内部模型；
//   2. 外部iframe只接收短时运行令牌和最小会话信息；
//   3. 消息只允许student和assistant角色，不保存system提示词或隐藏推理；
//   4. 公开聊天响应不返回模型、供应商、积分账户或教师身份；
//   5. 使用流水是追加式记录，不能作为学生画像或长期身份档案；
//   6. 外部会话创建同时绑定浏览器访问的官方embed页面和真实父页面Origin。

import (
	"strings"
	"time"
)

// ==================== 会话类型 ====================

const (
	AssistantRuntimeSessionKindExternal       = "external"
	AssistantRuntimeSessionKindTeacherPreview = "teacher_preview"
)

// IsValidAssistantRuntimeSessionKind 校验会话类型。
func IsValidAssistantRuntimeSessionKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case AssistantRuntimeSessionKindExternal,
		AssistantRuntimeSessionKindTeacherPreview:
		return true
	default:
		return false
	}
}

// ==================== 会话状态 ====================

const (
	AssistantRuntimeSessionStatusActive    = "active"
	AssistantRuntimeSessionStatusCompleted = "completed"
	AssistantRuntimeSessionStatusExpired   = "expired"
	AssistantRuntimeSessionStatusRevoked   = "revoked"
)

// IsValidAssistantRuntimeSessionStatus 校验会话状态。
func IsValidAssistantRuntimeSessionStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case AssistantRuntimeSessionStatusActive,
		AssistantRuntimeSessionStatusCompleted,
		AssistantRuntimeSessionStatusExpired,
		AssistantRuntimeSessionStatusRevoked:
		return true
	default:
		return false
	}
}

// IsAssistantRuntimeSessionTerminal 判断会话是否已进入终态。
func IsAssistantRuntimeSessionTerminal(status string) bool {
	switch strings.TrimSpace(status) {
	case AssistantRuntimeSessionStatusCompleted,
		AssistantRuntimeSessionStatusExpired,
		AssistantRuntimeSessionStatusRevoked:
		return true
	default:
		return false
	}
}

// ==================== 可见消息角色 ====================

const (
	AssistantRuntimeMessageRoleStudent   = "student"
	AssistantRuntimeMessageRoleAssistant = "assistant"
)

// IsValidAssistantRuntimeMessageRole 校验持久化消息角色。
func IsValidAssistantRuntimeMessageRole(role string) bool {
	switch strings.TrimSpace(role) {
	case AssistantRuntimeMessageRoleStudent,
		AssistantRuntimeMessageRoleAssistant:
		return true
	default:
		return false
	}
}

// ==================== 使用流水状态 ====================

const (
	AssistantRuntimeUsageStatusSucceeded = "succeeded"
	AssistantRuntimeUsageStatusFailed    = "failed"
)

// IsValidAssistantRuntimeUsageStatus 校验使用流水状态。
func IsValidAssistantRuntimeUsageStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case AssistantRuntimeUsageStatusSucceeded,
		AssistantRuntimeUsageStatusFailed:
		return true
	default:
		return false
	}
}

// ==================== 可见消息协议 ====================

// AssistantRuntimeMessage 是会话中正式可见的一条消息。
//
// 不允许system、tool或隐藏推理角色进入MessagesJSON。
type AssistantRuntimeMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	CreatedAt *time.Time `json:"created_at"`
}

// ==================== 数据库会话记录 ====================

// AssistantRuntimeSession 对应assistant_runtime_sessions。
//
// TokenJTIHash、AnonymousClientHash和IPHash是后端安全字段，禁止返回浏览器。
// MessagesJSON由仓储层读取后解码为AssistantRuntimeSessionView.Messages。
type AssistantRuntimeSession struct {
	ID                string `json:"-"`
	DeploymentID      string `json:"-"`
	DeploymentVersion int    `json:"-"`

	TokenJTIHash        string `json:"-"`
	AnonymousClientHash string `json:"-"`
	OriginSnapshot      string `json:"-"`
	IPHash              string `json:"-"`

	SessionKind string `json:"-"`
	Status      string `json:"-"`
	TurnCount   int    `json:"-"`
	MaxTurns    int    `json:"-"`

	ActiveTurnID        *string    `json:"-"`
	ActiveTurnStartedAt *time.Time `json:"-"`

	MessagesJSON string `json:"-"`

	ExpiresAt    *time.Time `json:"-"`
	LastActiveAt *time.Time `json:"-"`
	CreatedAt    *time.Time `json:"-"`
	UpdatedAt    *time.Time `json:"-"`
}

// AssistantRuntimeSessionView 是外部或教师预览可安全读取的会话状态。
type AssistantRuntimeSessionView struct {
	ID                string                    `json:"id"`
	DeploymentVersion int                       `json:"deployment_version"`
	SessionKind       string                    `json:"session_kind"`
	Status            string                    `json:"status"`
	TurnCount         int                       `json:"turn_count"`
	MaxTurns          int                       `json:"max_turns"`
	RemainingTurns    int                       `json:"remaining_turns"`
	Messages          []AssistantRuntimeMessage `json:"messages"`
	ExpiresAt         *time.Time                `json:"expires_at"`
	LastActiveAt      *time.Time                `json:"last_active_at"`
}

// ==================== 会话创建 ====================

// AssistantRuntimeSessionCreateInput 是服务层写入仓储的内部参数。
//
// 所有哈希必须在进入仓储前由服务层计算完成。
type AssistantRuntimeSessionCreateInput struct {
	DeploymentID      string
	DeploymentVersion int
	TokenJTIHash      string

	AnonymousClientHash string
	OriginSnapshot      string
	IPHash              string

	SessionKind string
	MaxTurns    int
	ExpiresAt   time.Time
}

// AssistantRuntimeStartRequest 是公开iframe创建会话的请求。
//
// AnonymousClientID是浏览器生成的随机标识，不是学生真实身份。
// 服务端不得直接落库，必须加盐哈希后保存。
//
// ParentOrigin必须由官方embed脚本从document.referrer解析得到。
// 它仍属于不可信请求字段，后端必须同时执行：
//   - HTTP Origin与当前TE-DNA运行站点匹配；
//   - Referer精确指向当前public_id的官方embed页面；
//   - ParentOrigin精确命中部署allowed_origins。
type AssistantRuntimeStartRequest struct {
	AnonymousClientID string `json:"anonymous_client_id"`
	ParentOrigin      string `json:"parent_origin"`
}

// AssistantRuntimeStartResponse 是会话创建后的公开响应。
//
// RuntimeToken是短时专用令牌，不是教师登录JWT。
type AssistantRuntimeStartResponse struct {
	SessionID      string     `json:"session_id"`
	RuntimeToken   string     `json:"runtime_token"`
	Status         string     `json:"status"`
	MaxTurns       int        `json:"max_turns"`
	ExpiresAt      *time.Time `json:"expires_at"`
	WelcomeMessage string     `json:"welcome_message"`
}

// ==================== 单轮并发控制 ====================

// AssistantRuntimeTurnClaim 表示仓储层成功领取的主轮次。
type AssistantRuntimeTurnClaim struct {
	TurnID            string
	SessionID         string
	DeploymentID      string
	DeploymentVersion int
	TurnCount         int
	MaxTurns          int
	Messages          []AssistantRuntimeMessage
	ClaimedAt         time.Time
}

// AssistantRuntimeChatRequest 是学生或教师预览发送的一轮消息。
type AssistantRuntimeChatRequest struct {
	Message string `json:"message"`
}

// AssistantRuntimeTurnCompletion 是成功完成主轮次的内部参数。
type AssistantRuntimeTurnCompletion struct {
	TurnID           string
	SessionID        string
	StudentMessage   AssistantRuntimeMessage
	AssistantMessage AssistantRuntimeMessage

	InputChars   int
	OutputChars  int
	InputTokens  int
	OutputTokens int
	CreditsUsed  float64
	ModelName    string
	Provider     string
	LatencyMs    int
}

// AssistantRuntimeTurnFailure 是失败主轮次的内部参数。
type AssistantRuntimeTurnFailure struct {
	TurnID     string
	SessionID  string
	InputChars int
	ErrorCode  string
	ModelName  string
	Provider   string
	LatencyMs  int
}

// AssistantRuntimeChatResponse 是公开单轮聊天响应。
//
// 不返回模型、供应商、积分和教师账户信息。
type AssistantRuntimeChatResponse struct {
	TurnID         string                  `json:"turn_id"`
	Message        AssistantRuntimeMessage `json:"message"`
	TurnCount      int                     `json:"turn_count"`
	RemainingTurns int                     `json:"remaining_turns"`
	SessionStatus  string                  `json:"session_status"`
}

// ==================== 数据库使用流水 ====================

// AssistantRuntimeUsage 对应assistant_runtime_usage。
//
// DeploymentID和RuntimeSessionID是非空软关联审计快照，不建立外键。
// 父部署或会话删除后仍保留调用发生时的原始UUID。
//
// 本结构是内部流水记录，禁止直接作为外部iframe响应。
type AssistantRuntimeUsage struct {
	ID                string `json:"-"`
	TurnID            string `json:"-"`
	DeploymentID      string `json:"-"`
	RuntimeSessionID  string `json:"-"`
	DeploymentVersion int    `json:"-"`

	OwnerUserID  string `json:"-"`
	SchoolID     string `json:"-"`
	CoursewareID string `json:"-"`
	PageID       string `json:"-"`

	SessionKind  string  `json:"-"`
	InputChars   int     `json:"-"`
	OutputChars  int     `json:"-"`
	InputTokens  int     `json:"-"`
	OutputTokens int     `json:"-"`
	CreditsUsed  float64 `json:"-"`

	ModelName string `json:"-"`
	Provider  string `json:"-"`
	Status    string `json:"-"`
	ErrorCode string `json:"-"`
	LatencyMs int    `json:"-"`

	CreatedAt *time.Time `json:"-"`
}

// AssistantRuntimeUsageCreateInput 是服务层追加使用流水的内部参数。
type AssistantRuntimeUsageCreateInput struct {
	TurnID            string
	DeploymentID      string
	RuntimeSessionID  string
	DeploymentVersion int

	OwnerUserID  string
	SchoolID     string
	CoursewareID string
	PageID       string
	SessionKind  string

	InputChars   int
	OutputChars  int
	InputTokens  int
	OutputTokens int
	CreditsUsed  float64

	ModelName string
	Provider  string
	Status    string
	ErrorCode string
	LatencyMs int
}

// AssistantRuntimeUsageAdminView 是教师或管理端的使用流水响应。
//
// 本模型不得被公开运行接口使用。
type AssistantRuntimeUsageAdminView struct {
	ID                string     `json:"id"`
	TurnID            string     `json:"turn_id"`
	DeploymentVersion int        `json:"deployment_version"`
	SessionKind       string     `json:"session_kind"`
	InputChars        int        `json:"input_chars"`
	OutputChars       int        `json:"output_chars"`
	InputTokens       int        `json:"input_tokens"`
	OutputTokens      int        `json:"output_tokens"`
	CreditsUsed       float64    `json:"credits_used"`
	ModelName         string     `json:"model_name"`
	Provider          string     `json:"provider"`
	Status            string     `json:"status"`
	ErrorCode         string     `json:"error_code"`
	LatencyMs         int        `json:"latency_ms"`
	CreatedAt         *time.Time `json:"created_at"`
}

// AssistantRuntimeUsageListResponse 是管理端使用流水列表。
type AssistantRuntimeUsageListResponse struct {
	Items []*AssistantRuntimeUsageAdminView `json:"items"`
	Total int                               `json:"total"`
}
