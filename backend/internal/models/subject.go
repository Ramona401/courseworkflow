package models

import "time"

// ==================== 学科字典（v231）====================
//
// 对应 subjects 表。学科的单一真相源，供全平台下拉/筛选统一消费。
// 原学科列表散落在前端 8+ 处硬编码，各副本不一致（备课下拉缺劳动/道德与法治/
// 美术/音乐/体育等），本表建立后统一从 GET /api/v1/subjects 拉取。
//
// 定位：本表仅管「能选哪些学科」的展示层。学科深层能力（AOCI 索引编码 code、
// 课标库 curriculum_standards 约束）仍「有就用、没有就降级」——新学科能选能备课，
// 暂无课标约束注入，与现有兜底一致，不报错不崩溃。

// Subject 对应 subjects 表的一行
type Subject struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`       // 学科名（下拉展示值，唯一）
	Code      string    `json:"code"`       // 可选索引编码（对应 AOCI SubjectCodeMap，留空=后端用默认码）
	SortOrder int       `json:"sort_order"` // 排序（越小越靠前）
	IsActive  bool      `json:"is_active"`  // 启停（false=下拉不显示）
	IsSystem  bool      `json:"is_system"`  // 内置核心学科（true=不可删除，仅可停用/改名/调序）
	Note      string    `json:"note"`       // 备注
	UpdatedBy *string   `json:"updated_by"` // 最后修改人（可空）
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateSubjectRequest 新建学科请求（admin）
type CreateSubjectRequest struct {
	Name      string `json:"name"`       // 必填
	Code      string `json:"code"`       // 可选
	SortOrder int    `json:"sort_order"` // 可选，默认 100
	Note      string `json:"note"`       // 可选
}

// UpdateSubjectRequest 编辑学科请求（admin）
// 指针字段：nil=本次不修改该列，非 nil=更新为该值。
type UpdateSubjectRequest struct {
	Name      *string `json:"name"`
	Code      *string `json:"code"`
	SortOrder *int    `json:"sort_order"`
	IsActive  *bool   `json:"is_active"`
	Note      *string `json:"note"`
}
