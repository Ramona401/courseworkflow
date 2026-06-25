package models

// unit_plan.go — 单元方案数据模型（大单元备课·独立模块，不碰课时备课主线）
//
// 单元方案 = 学科负责人用"大单元逐步引导对话"产出的《大单元整体教学设计方案》+《单元整体设计图谱》。
// 它是一份"料"（备课资料），落在「我的备课资料 → 单元方案」Tab；二期再注入到各课时备课。
// 权限/可见性与 course_outlines 同构：写=组长(lead/backbone)+校管(senior_operator)+admin，读=全员。
// scope 三级：group 教研组 / school 学校 / system 全局（admin，所有学校通用，scope_target_id 用全零占位）。

import (
	"encoding/json"
	"time"
)

// scope 常量
const (
	UnitPlanScopeGroup  = "group"
	UnitPlanScopeSchool = "school"
	UnitPlanScopeSystem = "system"
)

// 全局单元方案的占位归属ID（全零UUID，与 course_outlines 同一套）
const UnitPlanSystemTargetID = "00000000-0000-0000-0000-000000000000"

// 来源
const (
	UnitPlanSourceGenerated = "generated" // 平台逐步引导对话产出
	UnitPlanSourcePaste     = "paste"     // 直接粘贴（预留）
)

// 状态
const (
	UnitPlanStatusDraft    = "draft"    // 生成中（草稿，仅本人可见）
	UnitPlanStatusActive   = "active"   // 定稿（按可见性对外）
	UnitPlanStatusArchived = "archived" // 软删除
)

// UnitPlan 单元方案实体（对应 unit_plans 表）
type UnitPlan struct {
	ID              string    `json:"id"`
	Scope           string    `json:"scope"`
	ScopeTargetID   string    `json:"scope_target_id"`
	Subject         string    `json:"subject"`
	Grade           string    `json:"grade"`
	Volume          string    `json:"volume"`
	Unit            string    `json:"unit"`
	UnitTheme       string    `json:"unit_theme"`
	Title           string    `json:"title"`
	Content         string    `json:"content"` // 方案文档
	Atlas           string    `json:"atlas"`   // 图谱表格
	ConversationLog string    `json:"-"`       // jsonb 原始文本，service 层按需 Parse
	SourceType      string    `json:"source_type"`
	CreatedBy       string    `json:"created_by"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UnitPlanMessage 单元备课对话的一条消息（存 conversation_log jsonb 数组元素）
// CreatedAt 用 string，避免不同时间格式导致整段反序列化失败（仅展示用，顺序靠数组位置）
type UnitPlanMessage struct {
	Role      string `json:"role"` // user / assistant
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// ParseUnitPlanLog 解析 conversation_log jsonb 文本为消息数组（空/非法返空切片）
func ParseUnitPlanLog(raw string) []UnitPlanMessage {
	if raw == "" || raw == "[]" {
		return []UnitPlanMessage{}
	}
	var msgs []UnitPlanMessage
	if err := json.Unmarshal([]byte(raw), &msgs); err != nil {
		return []UnitPlanMessage{}
	}
	return msgs
}

// UnitPlanListItem 列表项（含归属名回填，不含正文/图谱/对话）
type UnitPlanListItem struct {
	ID            string    `json:"id"`
	Scope         string    `json:"scope"`
	ScopeTargetID string    `json:"scope_target_id"`
	ScopeName     string    `json:"scope_name"`
	Subject       string    `json:"subject"`
	Grade         string    `json:"grade"`
	Volume        string    `json:"volume"`
	Unit          string    `json:"unit"`
	UnitTheme     string    `json:"unit_theme"`
	Title         string    `json:"title"`
	Status        string    `json:"status"`
	CreatorName   string    `json:"creator_name"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// StartUnitPlanRequest 开始一次单元备课（建草稿 + 出第一步）
type StartUnitPlanRequest struct {
	Scope         string `json:"scope"`           // group/school/system（必填）
	ScopeTargetID string `json:"scope_target_id"` // group/school 必填；system 后端填占位
	Subject       string `json:"subject"`         // 必填
	Grade         string `json:"grade"`           // 必填
	Volume        string `json:"volume"`          // 可选（册次）
	Unit          string `json:"unit"`            // 必填（第几单元/单元名）
	Title         string `json:"title"`           // 可选，缺省后端拼
}

// UnitPlanChatRequest 单元备课对话一轮
type UnitPlanChatRequest struct {
	Message string `json:"message"`
}

// SaveUnitPlanRequest 定稿保存（前端把确认好的方案/图谱回传，后端落库并置 active）
type SaveUnitPlanRequest struct {
	Title     string `json:"title"`
	UnitTheme string `json:"unit_theme"`
	Content   string `json:"content"` // 方案文档
	Atlas     string `json:"atlas"`   // 图谱表格
}

// IsValidUnitPlanScope 校验 scope
func IsValidUnitPlanScope(s string) bool {
	return s == UnitPlanScopeGroup || s == UnitPlanScopeSchool || s == UnitPlanScopeSystem
}
