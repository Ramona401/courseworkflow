package models

// unit_plan.go — 单元方案数据模型（大单元备课·独立模块，不碰课时备课主线）
//
// 单元方案 = 学科负责人用"大单元逐步引导对话"产出的《大单元整体教学设计方案》+《单元整体设计图谱》。
// 它是一份"料"（备课资料），落在「我的备课资料 → 单元方案」Tab；二期再注入到各课时备课。
// 权限/可见性与 course_outlines 同构：写=组长(lead/backbone)+校管(senior_operator)+admin，读=全员。
// scope 三级：group 教研组 / school 学校 / system 全局（admin，所有学校通用，scope_target_id 用全零占位）。
//
// v233 新增（课程大纲教材版本绑定，对齐备课工坊）：
//   UnitPlan / StartUnitPlanRequest 各新增 CourseOutlinePublisher *string 字段，
//   落 unit_plans.course_outline_publisher 列，三态语义与 lesson_plans 侧完全一致：
//     nil（列为 NULL）    = 未关联大纲 —— 对话时不注入任何课程大纲（含存量老数据，零回归）
//     ""（列为空串）      = 通用/不限版本 —— 只注入 publisher 为空串的大纲
//     "人教版" 等具名版本 = 只注入该版本大纲（零跨版本兜底，对不上就不注入）
//   替代旧的"MatchBestOutline 自动学段打分取一份"注入逻辑（一标多本时可能注入错版本教材）。

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

	// CourseOutlinePublisher 选定的课程大纲教材版本（三态，v233 新增）。
	// 语义与 lesson_plans.course_outline_publisher 完全对齐：
	//   nil    = 未关联大纲（AI 对话时不注入任何课程大纲；存量老数据均为此态，零回归）
	//   ""     = 通用/不限版本（只注入 publisher 为空串的大纲）
	//   "人教版" 等具名 = 只注入该版本大纲（零跨版本兜底）
	// JSON 序列化为 null / "" / "人教版"，前端据此展示"未关联/通用版/具名版本"三态。
	CourseOutlinePublisher *string `json:"course_outline_publisher"`
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

	// CourseOutlinePublisher 可选：选定课程大纲教材版本（三态，v233 新增）。
	//   前端不传 / 传 null → 解码为 nil = 不关联大纲（老客户端天然兼容）
	//   传 ""              → 通用/不限版本
	//   传 "人教版" 等具名  → 该版本精确匹配注入
	// 会话建立时定版落库；Chat 每轮 buildSystemPrompt 重读该列天然生效。
	CourseOutlinePublisher *string `json:"course_outline_publisher"`
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
