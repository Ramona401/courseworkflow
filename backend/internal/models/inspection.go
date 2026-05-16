package models

// inspection.go — 区域抽查数据模型
//
// v127 新增（多级审核体系 · 区域抽查）：
//   - InspectionRecord：抽查记录实体
//   - DistrictInspectorAssignment：区域教研员管辖分配
//   - 相关常量、请求/响应结构体
//
// 对应数据库表：
//   inspection_records              — 抽查记录
//   district_inspector_assignments  — 区域教研员管辖分配

import "time"

// ==================== 抽查状态常量 ====================

const (
	InspectionStatusPending   = "pending"    // 待分配
	InspectionStatusAssigned  = "assigned"   // 已分配审查员
	InspectionStatusInReview  = "in_review"  // 审查中
	InspectionStatusPassed    = "passed"     // 抽查通过
	InspectionStatusRevoked   = "revoked"    // 发现问题，已撤回教案
)

// InspectionStatusNameMap 抽查状态中文名
var InspectionStatusNameMap = map[string]string{
	InspectionStatusPending:  "待分配",
	InspectionStatusAssigned: "已分配",
	InspectionStatusInReview: "审查中",
	InspectionStatusPassed:   "抽查通过",
	InspectionStatusRevoked:  "已撤回",
}

// ==================== 数据库实体 ====================

// InspectionRecord 抽查记录（对应 inspection_records 表）
type InspectionRecord struct {
	ID           string     `json:"id"`
	LessonPlanID string     `json:"lesson_plan_id"`
	InspectorID  *string    `json:"inspector_id"`   // 分配的教研员（可后分配）
	SampleBatch  string     `json:"sample_batch"`   // 抽样批次号
	Status       string     `json:"status"`         // pending/assigned/in_review/passed/revoked
	Priority     int        `json:"priority"`       // 0普通/1优先/2紧急
	Comment      string     `json:"comment"`        // 抽查意见
	InspectedAt  *time.Time `json:"inspected_at"`   // 完成时间
	CreatedAt    *time.Time `json:"created_at"`
}

// DistrictInspectorAssignment 区域教研员管辖分配（对应 district_inspector_assignments 表）
type DistrictInspectorAssignment struct {
	ID          string     `json:"id"`
	InspectorID string     `json:"inspector_id"`
	RegionID    string     `json:"region_id"`
	CreatedAt   *time.Time `json:"created_at"`
}

// ==================== 请求结构体 ====================

// InspectionReviewRequest 提交抽查结果请求
type InspectionReviewRequest struct {
	Decision string `json:"decision"` // passed / revoked
	Comment  string `json:"comment"`  // 抽查意见
}

// BatchSampleRequest 手动触发抽样请求
type BatchSampleRequest struct {
	SchoolID   string  `json:"school_id"`    // 可选：指定学校
	SampleRate float64 `json:"sample_rate"`  // 可选：覆盖默认抽样比例
}

// AssignInspectorRequest 分配审查员请求
type AssignInspectorRequest struct {
	InspectorID string `json:"inspector_id"`
}

// CreateDistrictInspectorRequest 分配区域教研员请求
type CreateDistrictInspectorRequest struct {
	InspectorID string `json:"inspector_id"`
	RegionID    string `json:"region_id"`
}

// ==================== 响应结构体 ====================

// InspectionListItem 抽查列表项（含教案和教研员信息）
type InspectionListItem struct {
	ID             string     `json:"id"`
	LessonPlanID   string     `json:"lesson_plan_id"`
	PlanTitle      string     `json:"plan_title"`
	PlanSubject    string     `json:"plan_subject"`
	PlanGrade      string     `json:"plan_grade"`
	AuthorName     string     `json:"author_name"`
	SchoolName     string     `json:"school_name"`
	InspectorID    *string    `json:"inspector_id"`
	InspectorName  string     `json:"inspector_name"`
	SampleBatch    string     `json:"sample_batch"`
	Status         string     `json:"status"`
	StatusName     string     `json:"status_name"`
	Priority       int        `json:"priority"`
	Comment        string     `json:"comment"`
	InspectedAt    *time.Time `json:"inspected_at"`
	CreatedAt      *time.Time `json:"created_at"`
}

// InspectionListResponse 抽查列表响应
type InspectionListResponse struct {
	Items []*InspectionListItem `json:"items"`
	Total int                   `json:"total"`
}

// InspectionStatsResponse 抽查统计响应
type InspectionStatsResponse struct {
	TotalSampled  int     `json:"total_sampled"`   // 总抽查数
	TotalPending  int     `json:"total_pending"`   // 待审查
	TotalPassed   int     `json:"total_passed"`    // 已通过
	TotalRevoked  int     `json:"total_revoked"`   // 已撤回
	PassRate      float64 `json:"pass_rate"`       // 通过率
}

// DistrictInspectorListItem 区域教研员列表项
type DistrictInspectorListItem struct {
	ID            string     `json:"id"`
	InspectorID   string     `json:"inspector_id"`
	InspectorName string     `json:"inspector_name"`
	RegionID      string     `json:"region_id"`
	RegionName    string     `json:"region_name"`
	CreatedAt     *time.Time `json:"created_at"`
}
