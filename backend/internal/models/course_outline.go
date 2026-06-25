package models

// course_outline.go — 课程大纲数据模型（大单元备课能力·批次一）
//
// 课程大纲 = 一册书的完整课时地图（如郯城那份"三年级下册大纲"）。
// 设计要点：存原文整块 content，不拆结构化字段，永不写死格式；
//           只拆少量"外层标签"（学科/年级/册次/归属）用于检索匹配。
// scope 三级：
//   group  教研组级 —— scope_target_id = 教研组ID，组长(lead/backbone)管
//   school 学校级   —— scope_target_id = 学校ID，校管(senior_operator)管
//   system 全局级   —— scope_target_id = 全零占位UUID，仅 admin 管，所有老师可见可注入
// 写权限：组长 + 校管 + admin（system 仅 admin）；读权限：全员登录可查（备课要用）。

import "time"

// 课程大纲 scope 常量
const (
	CourseOutlineScopeGroup  = "group"  // 教研组级
	CourseOutlineScopeSchool = "school" // 学校级
	CourseOutlineScopeSystem = "system" // 全局级（admin 录入，所有学校通用）
)

// CourseOutlineSystemTargetID 全局大纲的占位归属ID（全零UUID）
// system 级大纲无具体归属，但 scope_target_id 列 NOT NULL，故用全零UUID占位；
// 它是真实非NULL值，使 uq_course_outlines_active 唯一索引天然对全局大纲去重
// （两条同学科同年级同册次的全局大纲会因占位ID相同而撞唯一约束被挡）。
const CourseOutlineSystemTargetID = "00000000-0000-0000-0000-000000000000"

// 课程大纲录入方式常量
const (
	CourseOutlineSourcePaste  = "paste"  // 粘贴文本（本步唯一支持）
	CourseOutlineSourceUpload = "upload" // 上传原件（预留，本步不用）
)

// 课程大纲状态常量
const (
	CourseOutlineStatusActive   = "active"   // 生效
	CourseOutlineStatusArchived = "archived" // 软删除
)

// CourseOutline 课程大纲实体（对应 course_outlines 表）
type CourseOutline struct {
	ID             string    `json:"id"`
	Scope          string    `json:"scope"`            // group / school / system
	ScopeTargetID  string    `json:"scope_target_id"`  // 教研组ID / 学校ID / 全零占位ID
	Subject        string    `json:"subject"`          // 学科
	Grade          string    `json:"grade"`            // 年级
	Volume         string    `json:"volume"`           // 册次
	Title          string    `json:"title"`            // 标题
	Content        string    `json:"content"`          // 原文整块
	SourceFilePath string    `json:"source_file_path"` // 原件路径（预留）
	SourceType     string    `json:"source_type"`      // paste / upload
	CreatedBy      string    `json:"created_by"`       // 建立者ID
	Status         string    `json:"status"`           // active / archived
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CourseOutlineListItem 列表项（含归属名称回填，供管理界面展示）
type CourseOutlineListItem struct {
	ID            string    `json:"id"`
	Scope         string    `json:"scope"`
	ScopeTargetID string    `json:"scope_target_id"`
	ScopeName     string    `json:"scope_name"` // 教研组名 / 学校名 / "全局（所有学校通用）"
	Subject       string    `json:"subject"`
	Grade         string    `json:"grade"`
	Volume        string    `json:"volume"`
	Title         string    `json:"title"`
	CreatorName   string    `json:"creator_name"` // 建立者显示名
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateCourseOutlineRequest 创建请求
type CreateCourseOutlineRequest struct {
	Scope         string `json:"scope"`           // group / school / system（必填）
	ScopeTargetID string `json:"scope_target_id"` // 教研组ID / 学校ID（system 可不传，由后端填占位ID）
	Subject       string `json:"subject"`         // 必填
	Grade         string `json:"grade"`           // 必填
	Volume        string `json:"volume"`          // 必填
	Title         string `json:"title"`           // 必填
	Content       string `json:"content"`         // 必填（原文整块）
}

// UpdateCourseOutlineRequest 更新请求（学科/年级/册次/标题/正文均可改）
type UpdateCourseOutlineRequest struct {
	Subject string `json:"subject"`
	Grade   string `json:"grade"`
	Volume  string `json:"volume"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// IsValidCourseOutlineScope 校验 scope 合法性
func IsValidCourseOutlineScope(scope string) bool {
	return scope == CourseOutlineScopeGroup ||
		scope == CourseOutlineScopeSchool ||
		scope == CourseOutlineScopeSystem
}
