package models

// lesson_plan_version.go — 教案正文版本历史数据模型
//
// 每次正文被人工编辑、AI生成、导入或历史恢复覆盖之前，
// repository.UpdateLessonPlanContent 会把修改前的完整状态保存为快照。
//
// 版本历史只保存教案正文及与正文紧密相关的字段：
//   - 标题
//   - Markdown正文
//   - 结构化正文
//   - 课时时长
//
// 不保存状态、可见范围、评审结果等业务流程字段，
// 避免恢复正文时意外回退审核和发布状态。

import "time"

// ==================== 版本来源常量 ====================

const (
	LPVersionSourceManual  = "manual"  // 老师手动编辑
	LPVersionSourceAI      = "ai"      // AI生成、AI修订或应用AI建议
	LPVersionSourceImport  = "import"  // 导入Word、PDF或粘贴内容
	LPVersionSourceRestore = "restore" // 恢复历史版本
	LPVersionSourceSystem  = "system"  // 阶段重启等其它系统写入
)

// LessonPlanVersionMeta 描述即将覆盖旧正文的修改来源。
//
// 注意：版本表保存的是“修改前快照”，但ChangeSource描述的是
// “什么操作覆盖了这份快照”，便于老师理解历史版本产生原因。
type LessonPlanVersionMeta struct {
	ChangeSource  string
	ChangedBy     *string
	ChangeSummary string
}

// LessonPlanContentVersion 教案正文历史完整版本。
type LessonPlanContentVersion struct {
	ID                string    `json:"id"`
	LessonPlanID      string    `json:"lesson_plan_id"`
	VersionNumber     int       `json:"version_number"`
	Title             string    `json:"title"`
	ContentMarkdown   string    `json:"content_markdown"`
	ContentStructured string    `json:"content_structured"`
	DurationMinutes   int       `json:"duration_minutes"`
	ChangeSource      string    `json:"change_source"`
	ChangedBy         *string   `json:"changed_by"`
	ChangedByName     string    `json:"changed_by_name"`
	ChangeSummary     string    `json:"change_summary"`
	CreatedAt         time.Time `json:"created_at"`
}

// LessonPlanContentVersionListItem 版本列表轻量条目。
// 列表不返回完整正文，避免长教案一次加载造成不必要流量。
type LessonPlanContentVersionListItem struct {
	ID              string    `json:"id"`
	VersionNumber   int       `json:"version_number"`
	Title           string    `json:"title"`
	ContentPreview  string    `json:"content_preview"`
	CharacterCount  int       `json:"character_count"`
	DurationMinutes int       `json:"duration_minutes"`
	ChangeSource    string    `json:"change_source"`
	ChangedBy       *string   `json:"changed_by"`
	ChangedByName   string    `json:"changed_by_name"`
	ChangeSummary   string    `json:"change_summary"`
	CreatedAt       time.Time `json:"created_at"`
}

// LessonPlanContentVersionListResponse 版本列表响应。
type LessonPlanContentVersionListResponse struct {
	Versions       []*LessonPlanContentVersionListItem `json:"versions"`
	Total          int                                 `json:"total"`
	CurrentVersion int                                 `json:"current_version"`
}

// LessonPlanContentRestoreResponse 恢复成功响应。
// 前端可直接用ContentMarkdown同步当前画布，不必再额外请求详情。
type LessonPlanContentRestoreResponse struct {
	RestoredFromVersion int    `json:"restored_from_version"`
	CurrentVersion      int    `json:"current_version"`
	Title               string `json:"title"`
	ContentMarkdown     string `json:"content_markdown"`
	DurationMinutes     int    `json:"duration_minutes"`
}
