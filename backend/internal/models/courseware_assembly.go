package models

// courseware_assembly.go — 课件自动装配业务版本与布局验收领域模型
//
// 本文件只定义稳定数据契约，不访问数据库、不调用AI、不启动浏览器。
//
// 三层状态彼此正交：
//   1. Courseware.Status：课件生产步骤状态；
//   2. Courseware.PublishState：课件发布与审核状态；
//   3. CoursewareAssemblyStatus：某一次自动装配运行的生命周期。
//
// 设计原则：
//   - 每次装配获得单调递增的Version；
//   - 页面写回必须同时绑定Version和RunID；
//   - 取消、失败、中断或新版本产生后，旧运行不能继续写页面；
//   - 布局报告必须绑定页面HTML哈希，HTML改变后旧报告立即失效。

import "time"

// 课件装配状态。
const (
	CoursewareAssemblyStatusIdle            = "idle"
	CoursewareAssemblyStatusRunning         = "running"
	CoursewareAssemblyStatusCancelRequested = "cancel_requested"
	CoursewareAssemblyStatusCompleted       = "completed"
	CoursewareAssemblyStatusCancelled       = "cancelled"
	CoursewareAssemblyStatusFailed          = "failed"
	CoursewareAssemblyStatusInterrupted     = "interrupted"
)

// 课件页面浏览器布局验收状态。
const (
	CoursewareLayoutStatusUnchecked = "unchecked"
	CoursewareLayoutStatusChecking  = "checking"
	CoursewareLayoutStatusPassed    = "passed"
	CoursewareLayoutStatusFailed    = "failed"
	CoursewareLayoutStatusRepairing = "repairing"
	CoursewareLayoutStatusError     = "error"
)

// IsValidCoursewareAssemblyStatus 判断装配状态是否合法。
func IsValidCoursewareAssemblyStatus(status string) bool {
	switch status {
	case CoursewareAssemblyStatusIdle,
		CoursewareAssemblyStatusRunning,
		CoursewareAssemblyStatusCancelRequested,
		CoursewareAssemblyStatusCompleted,
		CoursewareAssemblyStatusCancelled,
		CoursewareAssemblyStatusFailed,
		CoursewareAssemblyStatusInterrupted:
		return true
	default:
		return false
	}
}

// IsCoursewareAssemblyFinalStatus 判断装配状态是否已经进入终态。
func IsCoursewareAssemblyFinalStatus(status string) bool {
	switch status {
	case CoursewareAssemblyStatusCompleted,
		CoursewareAssemblyStatusCancelled,
		CoursewareAssemblyStatusFailed,
		CoursewareAssemblyStatusInterrupted:
		return true
	default:
		return false
	}
}

// IsValidCoursewareLayoutStatus 判断页面布局验收状态是否合法。
func IsValidCoursewareLayoutStatus(status string) bool {
	switch status {
	case CoursewareLayoutStatusUnchecked,
		CoursewareLayoutStatusChecking,
		CoursewareLayoutStatusPassed,
		CoursewareLayoutStatusFailed,
		CoursewareLayoutStatusRepairing,
		CoursewareLayoutStatusError:
		return true
	default:
		return false
	}
}

// CoursewareAssemblyRun 表示一次不可变身份的课件装配运行。
//
// 运行过程可更新Status、错误和Metadata，但ID、CoursewareID和Version不可改变。
type CoursewareAssemblyRun struct {
	ID           string     `json:"id"`
	CoursewareID string     `json:"courseware_id"`
	Version      int64      `json:"version"`
	StartedBy    string     `json:"started_by"`
	SkipVideo    bool       `json:"skip_video"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"error_message"`
	MetadataJSON string     `json:"metadata"`
	StartedAt    *time.Time `json:"started_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
	FinishedAt   *time.Time `json:"finished_at"`
}

// CoursewareAssemblyState 是课件当前装配状态的轻量快照。
type CoursewareAssemblyState struct {
	CoursewareID string     `json:"courseware_id"`
	Version      int64      `json:"version"`
	Status       string     `json:"status"`
	ActiveRunID  *string    `json:"active_run_id"`
	StartedBy    *string    `json:"started_by"`
	SkipVideo    bool       `json:"skip_video"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
}

// CoursewarePageLayoutAudit 是页面当前浏览器布局验收快照。
//
// AuditJSON保存结构化问题报告，例如：
//   overflow_nodes、clipped_text_nodes、overlapping_nodes、missing_media、
//   fake_media、background_status、navigation_status和interaction_status。
//
// 本结构不保存截图文件或截图二进制。
type CoursewarePageLayoutAudit struct {
	PageID          string     `json:"page_id"`
	CoursewareID    string     `json:"courseware_id"`
	AssemblyVersion int64      `json:"assembly_version"`
	Status          string     `json:"status"`
	HTMLHash        string     `json:"html_hash"`
	AuditJSON       string     `json:"audit"`
	CheckedAt       *time.Time `json:"checked_at"`
}
