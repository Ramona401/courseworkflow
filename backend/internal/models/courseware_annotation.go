package models

// courseware_annotation.go — 课件页级批注数据模型(阶段2)
//
// 镜像 annotation.go(教案段落批注),核心差异:
//   - 挂载点从"段落序号 paragraph_index"换成"页码 page_number"(对齐 courseware_pages.page_number)
//   - 不含 review_round(评审轮次):课件多级审核在阶段3才接入,本期是纯人工批注闭环
//   - 状态保留三态(pending/resolved/archived),archived 留给阶段3,本期只用 pending/resolved
//
// 复用既有的 AnnotationStatusPending/Resolved/Archived 常量(定义在 annotation.go,同 models 包)。

import "time"

// ==================== 数据库实体 ====================

// CoursewareAnnotation 课件页级批注记录(对应 courseware_annotations 表)
type CoursewareAnnotation struct {
	ID           string    `json:"id"`
	CoursewareID string    `json:"courseware_id"`
	PageNumber   int       `json:"page_number"`   // 批注挂在第几页(对齐 courseware_pages.page_number)
	ReviewerID   string    `json:"reviewer_id"`   // 批注人ID
	ReviewerName string    `json:"reviewer_name"` // 批注人显示名(冗余,免每次 JOIN)
	Content      string    `json:"content"`       // 批注内容
	Status       string    `json:"status"`        // pending / resolved / archived
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ==================== 请求 / 响应模型 ====================

// CreateCWAnnotationRequest 创建课件批注请求
type CreateCWAnnotationRequest struct {
	PageNumber int    `json:"page_number"` // 必填,挂在第几页
	Content    string `json:"content"`     // 必填,批注内容
}

// ResolveCWAnnotationRequest 标记课件批注处理状态请求
type ResolveCWAnnotationRequest struct {
	Status string `json:"status"` // resolved / pending
}

// CWAnnotationListResponse 课件批注列表响应
// 前端可按 page_number 分组,在胶片条对应页挂气泡;status 区分待处理/已处理
type CWAnnotationListResponse struct {
	Annotations []*CoursewareAnnotation `json:"annotations"`
	Total       int                     `json:"total"`
}
