package models

// courseware_annotation.go — 课件页级批注数据模型
//
// 页面定位语义：
//   - PageID：稳定页面ID，页面重排后保持不变；
//   - PageNumber：目标页面当前页码，页面已删除时回退到创建时页码；
//   - PageNumberSnapshot：创建批注时的历史页码；
//   - 页面被删除后PageID为空，但批注记录和历史页码继续保留。
//
// 批注状态复用annotation.go中已有常量：
// pending / resolved / archived。

import "time"

// ==================== 数据库实体 ====================

// CoursewareAnnotation 课件页级批注记录。
type CoursewareAnnotation struct {
	ID           string  `json:"id"`
	CoursewareID string  `json:"courseware_id"`
	PageID       *string `json:"page_id"`

	// PageNumber由查询层根据PageID解析当前页码。
	// 页面已删除时回退到PageNumberSnapshot。
	PageNumber int `json:"page_number"`

	// PageNumberSnapshot记录创建批注时的页码，不随页面重排变化。
	PageNumberSnapshot int `json:"page_number_snapshot"`

	ReviewerID   string    `json:"reviewer_id"`
	ReviewerName string    `json:"reviewer_name"`
	Content      string    `json:"content"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ==================== 请求 / 响应模型 ====================

// CreateCWAnnotationRequest 创建课件批注请求。
//
// 前端仍提交当前页码；服务层会在写入前解析稳定PageID，
// 数据库兼容触发器负责保护旧版本后端。
type CreateCWAnnotationRequest struct {
	PageNumber int    `json:"page_number"`
	Content    string `json:"content"`
}

// ResolveCWAnnotationRequest 标记课件批注处理状态请求。
type ResolveCWAnnotationRequest struct {
	Status string `json:"status"`
}

// CWAnnotationListResponse 课件批注列表响应。
//
// 前端可按PageID稳定关联页面；PageNumber用于当前展示和跳转，
// PageNumberSnapshot用于页面删除后的历史说明。
type CWAnnotationListResponse struct {
	Annotations []*CoursewareAnnotation `json:"annotations"`
	Total       int                     `json:"total"`
}
