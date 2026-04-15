package models

// annotation.go — 教案段落批注数据模型
// 支持评审员对教案正文按段落添加批注，作者可查看并标记处理
//
// v104改动：
//   - LessonPlanAnnotation 新增 ReviewRound 字段（评审轮次，默认1）
//   - 新增 AnnotationStatusArchived 常量（归档状态，提交评审时自动归档上轮批注）

import "time"

// ==================== 批注状态常量 ====================

const (
	AnnotationStatusPending  = "pending"   // 待处理
	AnnotationStatusResolved = "resolved"  // 已处理
	AnnotationStatusArchived = "archived"  // 已归档（提交新一轮评审时自动设置）
)

// ==================== 数据库实体 ====================

// LessonPlanAnnotation 段落批注记录（对应 lesson_plan_annotations 表）
type LessonPlanAnnotation struct {
	ID               string    `json:"id"`
	LessonPlanID     string    `json:"lesson_plan_id"`
	ReviewerID       string    `json:"reviewer_id"`
	ReviewerName     string    `json:"reviewer_name"`
	ParagraphIndex   int       `json:"paragraph_index"`    // 段落序号（从0开始）
	ParagraphPreview string    `json:"paragraph_preview"`  // 段落前50字预览，方便作者定位
	Content          string    `json:"content"`            // 批注内容
	Status           string    `json:"status"`             // pending / resolved / archived
	ReviewRound      int       `json:"review_round"`       // 评审轮次（从1开始，每次提交评审递增）
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ==================== 请求/响应模型 ====================

// CreateAnnotationRequest 创建批注请求
type CreateAnnotationRequest struct {
	ParagraphIndex   int    `json:"paragraph_index"`
	ParagraphPreview string `json:"paragraph_preview"`
	Content          string `json:"content"`
	ReviewRound      int    `json:"review_round"` // 可选，默认由后端从教案评审历史推断
}

// UpdateAnnotationRequest 更新批注内容请求
type UpdateAnnotationRequest struct {
	Content string `json:"content"`
}

// ResolveAnnotationRequest 标记批注已处理请求
type ResolveAnnotationRequest struct {
	Status string `json:"status"` // resolved / pending
}

// AnnotationListResponse 批注列表响应
// 前端可按 review_round 分组，区分历史轮次和当前轮次
type AnnotationListResponse struct {
	Annotations  []*LessonPlanAnnotation `json:"annotations"`
	Total        int                     `json:"total"`
	CurrentRound int                     `json:"current_round"` // 当前最新评审轮次
}
