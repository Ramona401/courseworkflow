package models

// review_v2.go — 多级审核数据模型
//
// v127 新增（多级审核体系）
// v127.2 新增：ReviewedListItem（已审核记录列表项）+ ReviewedListResponse

import "time"

// ==================== 审核级别常量 ====================

const (
	ReviewLevelL1 = 1 // L1：教研组审核（教研组长/骨干）
	ReviewLevelL2 = 2 // L2：学校审核（学校管理员）
	ReviewLevelL3 = 3 // L3：区域抽查（区域教研员）
)

// ReviewLevelNameMap 审核级别中文名映射
var ReviewLevelNameMap = map[int]string{
	ReviewLevelL1: "教研组审核",
	ReviewLevelL2: "学校审核",
	ReviewLevelL3: "区域抽查",
}

// ==================== 审核决策常量 ====================

const (
	ReviewDecisionApproved = "approved" // 通过
	ReviewDecisionRevision = "revision" // 退回修改
	ReviewDecisionRevoked  = "revoked"  // 撤回（L3抽查专用）
)

// ValidReviewDecisions 有效的审核决策值
var ValidReviewDecisions = []string{
	ReviewDecisionApproved,
	ReviewDecisionRevision,
	ReviewDecisionRevoked,
}

// IsValidReviewDecision 校验审核决策是否有效
func IsValidReviewDecision(decision string) bool {
	for _, d := range ValidReviewDecisions {
		if d == decision {
			return true
		}
	}
	return false
}

// ==================== 数据库实体 ====================

// ReviewV2 多级审核记录（对应 lesson_plan_reviews_v2 表）
type ReviewV2 struct {
	ID           string     `json:"id"`
	LessonPlanID string     `json:"lesson_plan_id"`
	ReviewLevel  int        `json:"review_level"`  // 1=L1, 2=L2, 3=L3
	ReviewerID   string     `json:"reviewer_id"`
	Decision     string     `json:"decision"`      // approved/revision/revoked
	Score        *float64   `json:"score"`          // 可选评分
	Comment      string     `json:"comment"`        // 审核意见
	Dimensions   string     `json:"dimensions"`     // 多维度评分JSONB
	ReviewRound  int        `json:"review_round"`   // 审核轮次
	CreatedAt    *time.Time `json:"created_at"`
}

// ReviewFlowConfig 学校级审核流程配置（对应 review_flow_configs 表）
type ReviewFlowConfig struct {
	ID                    string     `json:"id"`
	SchoolID              string     `json:"school_id"`
	L2Enabled             bool       `json:"l2_enabled"`
	L3SampleRate          float64    `json:"l3_sample_rate"`
	AutoPublishOnApproved bool       `json:"auto_publish_on_approved"`
	UpdatedBy             *string    `json:"updated_by"`
	UpdatedAt             *time.Time `json:"updated_at"`
}

// ReviewerStats 审核员绩效统计快照（对应 reviewer_stats 表）
type ReviewerStats struct {
	ID                   string     `json:"id"`
	ReviewerID           string     `json:"reviewer_id"`
	ReviewLevel          int        `json:"review_level"`
	Period               string     `json:"period"`
	TotalReviewed        int        `json:"total_reviewed"`
	TotalApproved        int        `json:"total_approved"`
	TotalRevision        int        `json:"total_revision"`
	AvgScore             *float64   `json:"avg_score"`
	AvgReviewTimeMinutes *int       `json:"avg_review_time_minutes"`
	UpdatedAt            *time.Time `json:"updated_at"`
}

// ==================== 请求结构体 ====================

// SubmitReviewV2Request 提交审核请求（教师发起）
type SubmitReviewV2Request struct {
	GroupID string `json:"group_id"`
}

// ReviewDecisionV2Request 审核决策请求（审核员操作）
type ReviewDecisionV2Request struct {
	Decision   string   `json:"decision"`
	Score      *float64 `json:"score"`
	Comment    string   `json:"comment"`
	Dimensions string   `json:"dimensions"`
}

// UpdateReviewFlowConfigRequest 更新审核流程配置请求
type UpdateReviewFlowConfigRequest struct {
	L2Enabled             bool    `json:"l2_enabled"`
	L3SampleRate          float64 `json:"l3_sample_rate"`
	AutoPublishOnApproved bool    `json:"auto_publish_on_approved"`
}

// ==================== 响应结构体 ====================

// ReviewV2ListItem 审核记录列表项（含审核员名称）
type ReviewV2ListItem struct {
	ID           string     `json:"id"`
	LessonPlanID string     `json:"lesson_plan_id"`
	ReviewLevel  int        `json:"review_level"`
	LevelName    string     `json:"level_name"`
	ReviewerID   string     `json:"reviewer_id"`
	ReviewerName string     `json:"reviewer_name"`
	Decision     string     `json:"decision"`
	Score        *float64   `json:"score"`
	Comment      string     `json:"comment"`
	ReviewRound  int        `json:"review_round"`
	CreatedAt    *time.Time `json:"created_at"`
}

// ReviewHistoryResponse 审核历史响应
type ReviewHistoryResponse struct {
	Reviews      []*ReviewV2ListItem `json:"reviews"`
	Total        int                 `json:"total"`
	CurrentLevel int                 `json:"current_level"`
}

// PendingReviewItem 待审核列表项
type PendingReviewItem struct {
	LessonPlanID  string     `json:"lesson_plan_id"`
	Title         string     `json:"title"`
	Subject       string     `json:"subject"`
	Grade         string     `json:"grade"`
	AuthorID      string     `json:"author_id"`
	AuthorName    string     `json:"author_name"`
	GroupID       *string    `json:"group_id"`
	GroupName     string     `json:"group_name"`
	SchoolName    string     `json:"school_name"`
	ReviewLevel   int        `json:"review_level"`
	LevelName     string     `json:"level_name"`
	AIReviewScore *float64   `json:"ai_review_score"`
	SubmittedAt   *time.Time `json:"submitted_at"`
}

// PendingReviewListResponse 待审核列表响应
type PendingReviewListResponse struct {
	Items []*PendingReviewItem `json:"items"`
	Total int                  `json:"total"`
}

// ReviewStatsResponse 审核统计响应
type ReviewStatsResponse struct {
	TotalPending  int `json:"total_pending"`
	TotalReviewed int `json:"total_reviewed"`
	TotalApproved int `json:"total_approved"`
	TotalRevision int `json:"total_revision"`
}

// ReviewedListItem 已审核记录列表项（v127.2新增，展示审核历史）
type ReviewedListItem struct {
	ID           string     `json:"id"`
	LessonPlanID string     `json:"lesson_plan_id"`
	PlanTitle    string     `json:"plan_title"`
	PlanSubject  string     `json:"plan_subject"`
	PlanGrade    string     `json:"plan_grade"`
	AuthorName   string     `json:"author_name"`
	ReviewLevel  int        `json:"review_level"`
	LevelName    string     `json:"level_name"`
	ReviewerName string     `json:"reviewer_name"`
	Decision     string     `json:"decision"`
	Score        *float64   `json:"score"`
	Comment      string     `json:"comment"`
	CreatedAt    *time.Time `json:"created_at"`
}

// ReviewedListResponse 已审核记录列表响应
type ReviewedListResponse struct {
	Items []*ReviewedListItem `json:"items"`
	Total int                 `json:"total"`
}

// ReviewFlowConfigResponse 审核流程配置响应（含学校名称）
type ReviewFlowConfigResponse struct {
	SchoolID              string  `json:"school_id"`
	SchoolName            string  `json:"school_name"`
	L2Enabled             bool    `json:"l2_enabled"`
	L3SampleRate          float64 `json:"l3_sample_rate"`
	AutoPublishOnApproved bool    `json:"auto_publish_on_approved"`
}
