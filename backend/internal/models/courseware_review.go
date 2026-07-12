package models

// courseware_review.go — 课件多级审核数据模型（阶段3）
//
// 镜像 review_v2.go（教案多级审核），但有三处本质差异，务必理解：
//
//  1. 审核态载体不同：教案靠 lesson_plans.status（生产状态机）承载 submitted/revision/approved；
//     课件靠与 status【正交】的 coursewares.publish_state 承载。绝不改动 courseware.status。
//     状态机对照：
//       提交审核   → publish_state=submitted, review_level=0, review_school_id=作者学校
//       L1通过(无L2) → publish_state=approved,  review_level=1（待发布，作者再 published_shared）
//       L1通过(有L2) → publish_state=submitted, review_level=1（进入L2待审核）
//       L2通过      → publish_state=approved,  review_level=2
//       退回        → publish_state=revision,  review_level=0
//
//  2. 审核记录表不同：教案写 lesson_plan_reviews_v2，课件写 courseware_reviews（本期新建）。
//
//  3. 审核流程配置【复用】教案的 review_flow_configs 表（按 school_id，同校 l2_enabled 教案课件共用）。
//     因此本文件不重复定义 ReviewFlowConfig / UpdateReviewFlowConfigRequest 等——直接用 review_v2.go 里的。
//
// 同时，课件无 content_markdown 概念，教案侧"approved 正文非空硬校验"不适用；
// 课件的"是否做完"由 status≥preview 表达，在【提交审核】环节校验即可。
//
// 审核级别常量 / 决策常量【复用】review_v2.go 中已定义的 ReviewLevelL1/L2/L3、
// ReviewDecisionApproved/Revision/Revoked、ReviewLevelNameMap——同包内直接引用，不重复定义。

import "time"

// ==================== 数据库实体 ====================

// CoursewareReview 课件多级审核记录（对应 courseware_reviews 表）
type CoursewareReview struct {
	ID           string     `json:"id"`
	CoursewareID string     `json:"courseware_id"`
	ReviewLevel  int        `json:"review_level"` // 1=L1教研组 / 2=L2学校 / 3=L3区域(预留)
	ReviewerID   string     `json:"reviewer_id"`
	Decision     string     `json:"decision"`     // approved/revision/revoked
	Score        *float64   `json:"score"`        // 可选评分（对应 numeric(4,1)）
	Comment      string     `json:"comment"`      // 审核意见
	Dimensions   string     `json:"dimensions"`   // 多维度评分 JSONB 文本
	ReviewRound  int        `json:"review_round"` // 审核轮次
	CreatedAt    *time.Time `json:"created_at"`
}

// ==================== 请求结构体 ====================

// SubmitCoursewareReviewRequest 提交课件审核请求（作者发起）
// 课件提交审核无需像教案那样选教研组——作者所属教研组/学校由后端反查 school_id 确定。
// 预留空结构体便于将来扩展（如附带提交说明）。
type SubmitCoursewareReviewRequest struct {
	Note string `json:"note"` // 可选：提交说明（当前不落库，仅占位）
}

// CWReviewDecisionRequest 课件审核决策请求（审核员操作）
type CWReviewDecisionRequest struct {
	Decision   string   `json:"decision"`   // approved / revision
	Score      *float64 `json:"score"`      // 可选评分
	Comment    string   `json:"comment"`    // 审核意见
	Dimensions string   `json:"dimensions"` // 多维度评分 JSONB 文本
}

// ==================== 响应结构体 ====================

// CWReviewListItem 课件审核记录列表项（含审核员名称）
type CWReviewListItem struct {
	ID           string     `json:"id"`
	CoursewareID string     `json:"courseware_id"`
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

// CWReviewHistoryResponse 课件审核历史响应
type CWReviewHistoryResponse struct {
	Reviews      []*CWReviewListItem `json:"reviews"`
	Total        int                 `json:"total"`
	CurrentLevel int                 `json:"current_level"` // 课件当前审核层级进度（coursewares.review_level）
}

// CWPendingReviewItem 课件待审核列表项
type CWPendingReviewItem struct {
	CoursewareID string     `json:"courseware_id"`
	Title        string     `json:"title"`
	Subject      string     `json:"subject"`
	Grade        string     `json:"grade"`
	PageCount    int        `json:"page_count"`
	SourceType   string     `json:"source_type"`
	SourceName   string     `json:"source_name"`
	AuthorID     string     `json:"author_id"`
	AuthorName   string     `json:"author_name"`
	SchoolName   string     `json:"school_name"`
	ReviewLevel  int        `json:"review_level"`
	LevelName    string     `json:"level_name"`
	SubmittedAt  *time.Time `json:"submitted_at"` // 取 updated_at 近似提交时间
}

// CWPendingReviewListResponse 课件待审核列表响应
type CWPendingReviewListResponse struct {
	Items []*CWPendingReviewItem `json:"items"`
	Total int                    `json:"total"`
}

// CWReviewStatsResponse 课件审核统计响应
type CWReviewStatsResponse struct {
	TotalPending  int `json:"total_pending"`
	TotalReviewed int `json:"total_reviewed"`
	TotalApproved int `json:"total_approved"`
	TotalRevision int `json:"total_revision"`
}

// CWReviewedListItem 课件已审核记录列表项（展示审核历史）
type CWReviewedListItem struct {
	ID              string     `json:"id"`
	CoursewareID    string     `json:"courseware_id"`
	CoursewareTitle string     `json:"courseware_title"`
	Subject         string     `json:"subject"`
	Grade           string     `json:"grade"`
	AuthorName      string     `json:"author_name"`
	ReviewLevel     int        `json:"review_level"`
	LevelName       string     `json:"level_name"`
	ReviewerName    string     `json:"reviewer_name"`
	Decision        string     `json:"decision"`
	Score           *float64   `json:"score"`
	Comment         string     `json:"comment"`
	CreatedAt       *time.Time `json:"created_at"`
}

// CWReviewedListResponse 课件已审核记录列表响应
type CWReviewedListResponse struct {
	Items []*CWReviewedListItem `json:"items"`
	Total int                   `json:"total"`
}

// CWReviewDetailResponse 课件审核详情响应（审核台用：课件基本信息 + 页面 + 批注，供审核员边看边决策）
// 决策二落地：审核详情联动阶段2批注——审核员在同一界面看渲染页 + 该课件全部批注。
type CWReviewDetailResponse struct {
	Courseware  *CoursewareDetailResponse `json:"courseware"`  // 复用课件详情（含 pages）
	Annotations []*CoursewareAnnotation   `json:"annotations"` // 该课件全部批注（阶段2复用）
	Reviews     []*CWReviewListItem       `json:"reviews"`     // 历史审核记录
}
