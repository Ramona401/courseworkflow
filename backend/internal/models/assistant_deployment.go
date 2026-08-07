package models

// assistant_deployment.go
//
// 本文件定义课件教学智能体部署、不可变版本及发布快照协议。
//
// 安全边界：
//   1. 数据库记录与HTTP响应分离；
//   2. 完整助手提示词只存在内部快照模型，JSON响应永不输出；
//   3. 完整页面上下文只用于后端运行，不直接返回外部iframe；
//   4. 公开运行描述不包含内部部署ID、教师ID、学校ID或模型信息；
//   5. frame-ancestors只供后端生成动态CSP，不进入浏览器JSON协议；
//   6. 发布版本只能追加，不能原地修改。

import (
	"strings"
	"time"
)

// AssistantDeploymentSnapshotVersion 是发布快照协议版本。
const AssistantDeploymentSnapshotVersion = "v1"

// ==================== 部署访问模式 ====================

const AssistantDeploymentAccessOriginAllowlist = "origin_allowlist"

// IsValidAssistantDeploymentAccessMode 校验部署访问模式。
func IsValidAssistantDeploymentAccessMode(mode string) bool {
	return strings.TrimSpace(mode) == AssistantDeploymentAccessOriginAllowlist
}

// ==================== 部署状态 ====================

const (
	AssistantDeploymentStatusActive  = "active"
	AssistantDeploymentStatusPaused  = "paused"
	AssistantDeploymentStatusRevoked = "revoked"
)

// IsValidAssistantDeploymentStatus 校验部署状态。
func IsValidAssistantDeploymentStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case AssistantDeploymentStatusActive, AssistantDeploymentStatusPaused, AssistantDeploymentStatusRevoked:
		return true
	default:
		return false
	}
}

// IsAssistantDeploymentRunnableStatus 判断部署状态是否允许运行。
func IsAssistantDeploymentRunnableStatus(status string) bool {
	return strings.TrimSpace(status) == AssistantDeploymentStatusActive
}

// ==================== 数据库记录 ====================

// AssistantDeployment 对应assistant_deployments。
//
// AllowedOriginsJSON为仓储层原始JSON，不直接作为HTTP响应。
type AssistantDeployment struct {
	ID              string  `json:"-"`
	PublicID        string  `json:"-"`
	SlotID          *string `json:"-"`
	CoursewareID    string  `json:"-"`
	PageID          string  `json:"-"`
	OwnerUserID     string  `json:"-"`
	SchoolID        string  `json:"-"`
	EducationDomain string  `json:"-"`
	CurrentVersion  int     `json:"-"`
	AccessMode      string  `json:"-"`
	Status          string  `json:"-"`

	DailyCallLimit      int    `json:"-"`
	PerSessionTurnLimit int    `json:"-"`
	AllowedOriginsJSON  string `json:"-"`

	ValidFrom  *time.Time `json:"-"`
	ValidUntil *time.Time `json:"-"`
	CreatedAt  *time.Time `json:"-"`
	UpdatedAt  *time.Time `json:"-"`
}

// AssistantDeploymentVersion 对应assistant_deployment_versions。
//
// 所有字段均为后端内部记录。尤其AssistantPromptSnapshot和
// ContextSnapshotJSON禁止直接序列化到浏览器。
type AssistantDeploymentVersion struct {
	DeploymentID string  `json:"-"`
	Version      int     `json:"-"`
	AssistantID  *string `json:"-"`

	AssistantPromptSnapshot string `json:"-"`
	AssistantPromptHash     string `json:"-"`

	TeachingPlanJSON       string `json:"-"`
	ContextSnapshotJSON    string `json:"-"`
	ContextSnapshotHash    string `json:"-"`
	PageHTMLHash           string `json:"-"`
	CoursewareSnapshotJSON string `json:"-"`

	CreatedBy string     `json:"-"`
	CreatedAt *time.Time `json:"-"`
}

// ==================== 不可变教学计划快照 ====================

// AssistantDeploymentTeachingPlanSnapshot 是发布时冻结的教学方案。
type AssistantDeploymentTeachingPlanSnapshot struct {
	Version           string `json:"version"`
	Title             string `json:"title"`
	WelcomeMessage    string `json:"welcome_message"`
	TeachingRole      string `json:"teaching_role"`
	LearningObjective string `json:"learning_objective"`

	DisplayMode     string `json:"display_mode"`
	DisplayPosition string `json:"display_position"`

	GuidancePlan CoursewareAssistantGuidancePlan `json:"guidance_plan"`
}

// AssistantDeploymentPageContextSnapshot 是当前页发布上下文。
//
// VisibleText是从HTML确定性提取的可见文本；不保存页面完整HTML。
type AssistantDeploymentPageContextSnapshot struct {
	PageID            string `json:"page_id"`
	PageNumber        int    `json:"page_number"`
	Title             string `json:"title"`
	Purpose           string `json:"purpose"`
	ContentSummary    string `json:"content_summary"`
	InteractionType   string `json:"interaction_type"`
	VisualFormat      string `json:"visual_format"`
	MediaRequirements string `json:"media_requirements"`
	PageIndex         string `json:"page_index"`
	VisibleText       string `json:"visible_text"`

	InteractionEvidence CWAIReviewInteractionEvidence `json:"interaction_evidence"`
}

// AssistantDeploymentAdjacentPageSnapshot 是相邻页的最小摘要。
type AssistantDeploymentAdjacentPageSnapshot struct {
	PageID         string `json:"page_id"`
	PageNumber     int    `json:"page_number"`
	Title          string `json:"title"`
	Purpose        string `json:"purpose"`
	ContentSummary string `json:"content_summary"`
}

// AssistantDeploymentLessonPlanSnapshot 是来源教案相关片段快照。
//
// RelevantExcerpt只能保存与当前页教学目标相关的受限片段，不能保存完整教案。
type AssistantDeploymentLessonPlanSnapshot struct {
	LessonPlanID    *string `json:"lesson_plan_id"`
	Title           string  `json:"title"`
	RelevantExcerpt string  `json:"relevant_excerpt"`
	ExcerptHash     string  `json:"excerpt_hash"`
}

// AssistantDeploymentContextSnapshot 是发布时确定性的运行上下文。
//
// GeneratedAt是本次确定性装配完成时间。
// 稳定内容哈希不包含该时间字段，避免相同教学内容因装配时间不同而改变哈希。
type AssistantDeploymentContextSnapshot struct {
	Version     string     `json:"version"`
	GeneratedAt *time.Time `json:"generated_at"`

	CurrentPage  AssistantDeploymentPageContextSnapshot   `json:"current_page"`
	PreviousPage *AssistantDeploymentAdjacentPageSnapshot `json:"previous_page,omitempty"`
	NextPage     *AssistantDeploymentAdjacentPageSnapshot `json:"next_page,omitempty"`
	LessonPlan   *AssistantDeploymentLessonPlanSnapshot   `json:"lesson_plan,omitempty"`
}

// AssistantDeploymentCoursewareSnapshot 保存最小课件发布信息。
type AssistantDeploymentCoursewareSnapshot struct {
	CoursewareID    string `json:"courseware_id"`
	PageID          string `json:"page_id"`
	PageNumber      int    `json:"page_number"`
	CoursewareTitle string `json:"courseware_title"`
	PageTitle       string `json:"page_title"`
	Subject         string `json:"subject"`
	Grade           string `json:"grade"`
	EducationDomain string `json:"education_domain"`
}

// AssistantDeploymentVersionSnapshot 是发布服务内部使用的完整快照。
//
// 所有字段均标记json:"-"，禁止把该聚合对象整体作为HTTP响应。
// 发布服务应分别序列化TeachingPlan、ContextSnapshot和CoursewareSnapshot。
type AssistantDeploymentVersionSnapshot struct {
	AssistantID             *string                                 `json:"-"`
	AssistantPromptSnapshot string                                  `json:"-"`
	AssistantPromptHash     string                                  `json:"-"`
	TeachingPlan            AssistantDeploymentTeachingPlanSnapshot `json:"-"`
	ContextSnapshot         AssistantDeploymentContextSnapshot      `json:"-"`
	ContextSnapshotHash     string                                  `json:"-"`
	PageHTMLHash            string                                  `json:"-"`
	CoursewareSnapshot      AssistantDeploymentCoursewareSnapshot   `json:"-"`
}

// ==================== 教师端安全响应 ====================

// AssistantDeploymentView 是教师部署管理页响应。
//
// 不包含提示词快照、上下文正文、IP、运行令牌或模型密钥。
type AssistantDeploymentView struct {
	ID              string  `json:"id"`
	PublicID        string  `json:"public_id"`
	SlotID          *string `json:"slot_id"`
	CoursewareID    string  `json:"courseware_id"`
	PageID          string  `json:"page_id"`
	EducationDomain string  `json:"education_domain"`
	CurrentVersion  int     `json:"current_version"`
	AccessMode      string  `json:"access_mode"`
	Status          string  `json:"status"`

	DailyCallLimit      int      `json:"daily_call_limit"`
	PerSessionTurnLimit int      `json:"per_session_turn_limit"`
	AllowedOrigins      []string `json:"allowed_origins"`

	ValidFrom  *time.Time `json:"valid_from"`
	ValidUntil *time.Time `json:"valid_until"`
	CreatedAt  *time.Time `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

// AssistantDeploymentVersionView 是教师端版本元数据响应。
//
// 只返回哈希和时间，不返回完整提示词或上下文快照。
type AssistantDeploymentVersionView struct {
	Version             int        `json:"version"`
	AssistantID         *string    `json:"assistant_id"`
	AssistantPromptHash string     `json:"assistant_prompt_hash"`
	ContextSnapshotHash string     `json:"context_snapshot_hash"`
	PageHTMLHash        string     `json:"page_html_hash"`
	CreatedAt           *time.Time `json:"created_at"`
}

// AssistantDeploymentListResponse 是教师端部署列表响应。
type AssistantDeploymentListResponse struct {
	Deployments []*AssistantDeploymentView `json:"deployments"`
	Total       int                        `json:"total"`
}

// ==================== 外部iframe安全响应 ====================

// AssistantDeploymentPublicDescriptor 是公开iframe可读取的最小部署描述。
//
// FrameAncestors只在Go后端生成Content-Security-Policy时使用，标记为json:"-"，
// 不会出现在浏览器JSON、HTML数据属性或运行API响应中。
//
// 其余字段不包含内部UUID、教师身份、学校ID、积分、模型和提示词。
type AssistantDeploymentPublicDescriptor struct {
	PublicID            string   `json:"public_id"`
	Title               string   `json:"title"`
	WelcomeMessage      string   `json:"welcome_message"`
	DisplayMode         string   `json:"display_mode"`
	DisplayPosition     string   `json:"display_position"`
	MaximumSessionTurns int      `json:"maximum_session_turns"`
	FrameAncestors      []string `json:"-"`
}

// ==================== 发布与管理请求 ====================

// CreateAssistantDeploymentRequest 创建或首次发布部署。
//
// owner_user_id、school_id和education_domain由服务端解析，正文不得指定。
type CreateAssistantDeploymentRequest struct {
	DailyCallLimit      int        `json:"daily_call_limit"`
	PerSessionTurnLimit int        `json:"per_session_turn_limit"`
	AllowedOrigins      []string   `json:"allowed_origins"`
	ValidUntil          *time.Time `json:"valid_until"`
}

// UpdateAssistantDeploymentPolicyRequest 更新可变运行策略。
//
// revoked部署不得通过本请求恢复。
type UpdateAssistantDeploymentPolicyRequest struct {
	DailyCallLimit      int        `json:"daily_call_limit"`
	PerSessionTurnLimit int        `json:"per_session_turn_limit"`
	AllowedOrigins      []string   `json:"allowed_origins"`
	ValidUntil          *time.Time `json:"valid_until"`
}
