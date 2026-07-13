package models

// unit_plan_material.go — 大单元方案参考资料数据模型
//
// 第一阶段设计：
//   1. 浏览器端从docx或文字型PDF提取文字；
//   2. 短资料保存原文，长资料保存原文与压缩摘要；
//   3. 不保存原始PDF或Word物理文件；
//   4. 资料可见范围继承所属UnitPlan；
//   5. 删除UnitPlan时由数据库外键级联删除资料。

import (
	"strings"
	"time"
)

// 大单元参考资料类型。
const (
	UnitPlanMaterialTypeTextbook            = "textbook"
	UnitPlanMaterialTypeTeacherGuide        = "teacher_guide"
	UnitPlanMaterialTypePreviousUnitPlan    = "previous_unit_plan"
	UnitPlanMaterialTypeTeachingRequirement = "teaching_requirement"
	UnitPlanMaterialTypeExcellentCase       = "excellent_case"
	UnitPlanMaterialTypeOther               = "other"
)

// 大单元参考资料状态。
const (
	UnitPlanMaterialStatusActive   = "active"
	UnitPlanMaterialStatusArchived = "archived"
)

// UnitPlanMaterial 对应数据库unit_plan_materials表。
type UnitPlanMaterial struct {
	ID             string    `json:"id"`
	UnitPlanID     string    `json:"unit_plan_id"`
	MaterialType   string    `json:"material_type"`
	FileName       string    `json:"file_name"`
	ContentText    string    `json:"content_text,omitempty"`
	SummaryText    string    `json:"summary_text,omitempty"`
	OriginalLength int       `json:"original_length"`
	SummaryLength  int       `json:"summary_length"`
	UploadedBy     string    `json:"uploaded_by"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreateUnitPlanMaterialRequest 创建一份大单元参考资料。
//
// content_text为浏览器提取的原始文字。
// summary_text为长资料压缩结果，短资料可以为空。
type CreateUnitPlanMaterialRequest struct {
	MaterialType   string `json:"material_type"`
	FileName       string `json:"file_name"`
	ContentText    string `json:"content_text"`
	SummaryText    string `json:"summary_text"`
	OriginalLength int    `json:"original_length"`
	SummaryLength  int    `json:"summary_length"`
}

// UnitPlanMaterialListItem 资料列表轻量项。
//
// 默认不把原始全文返回到列表接口，减少无意义传输与资料暴露。
// 详情或AI装配时再读取正文。
type UnitPlanMaterialListItem struct {
	ID             string    `json:"id"`
	UnitPlanID     string    `json:"unit_plan_id"`
	MaterialType   string    `json:"material_type"`
	FileName       string    `json:"file_name"`
	OriginalLength int       `json:"original_length"`
	SummaryLength  int       `json:"summary_length"`
	HasSummary     bool      `json:"has_summary"`
	UploadedBy     string    `json:"uploaded_by"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// IsValidUnitPlanMaterialType 校验资料类型。
func IsValidUnitPlanMaterialType(value string) bool {
	switch strings.TrimSpace(value) {
	case UnitPlanMaterialTypeTextbook,
		UnitPlanMaterialTypeTeacherGuide,
		UnitPlanMaterialTypePreviousUnitPlan,
		UnitPlanMaterialTypeTeachingRequirement,
		UnitPlanMaterialTypeExcellentCase,
		UnitPlanMaterialTypeOther:
		return true
	default:
		return false
	}
}

// EffectiveText 返回AI实际优先使用的文本。
//
// 有压缩摘要时优先返回摘要，避免每轮注入大段原文；
// 没有摘要时返回短资料原文。
func (m *UnitPlanMaterial) EffectiveText() string {
	if m == nil {
		return ""
	}
	if summary := strings.TrimSpace(m.SummaryText); summary != "" {
		return summary
	}
	return strings.TrimSpace(m.ContentText)
}
