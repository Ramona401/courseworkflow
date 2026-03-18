package models

import (
	"encoding/json"
	"time"
)

// ==================== 课程模型（对应 courses 表） ====================

// Course 课程注册信息（对应 courses 表）
type Course struct {
	ID               string     `json:"id"`                // UUID主键
	CourseCode       string     `json:"course_code"`       // 课程编号（如 G1-01）
	CourseName       string     `json:"course_name"`       // 课程名称
	ExternalModuleID *int       `json:"external_module_id"` // 外部模块ID（OSS中的module_id）
	GradeNum         *int       `json:"grade_num"`         // 年级编号（1-12）
	Stage            string     `json:"stage"`             // 学段（primary/middle/high）
	Semester         string     `json:"semester"`          // 学期
	Status           string     `json:"status"`            // 状态（active/archived）
	CreatedAt        *time.Time `json:"created_at"`        // 创建时间
	UpdatedAt        *time.Time `json:"updated_at"`        // 更新时间
}

// CourseListResponse 课程列表响应
type CourseListResponse struct {
	Courses []*CourseListItem `json:"courses"` // 课程列表
	Total   int              `json:"total"`   // 总数
}

// CourseListItem 课程列表单条（含索引摘要信息）
type CourseListItem struct {
	ID               string     `json:"id"`                 // UUID主键
	CourseCode       string     `json:"course_code"`        // 课程编号
	CourseName       string     `json:"course_name"`        // 课程名称
	ExternalModuleID *int       `json:"external_module_id"` // 外部模块ID
	GradeNum         *int       `json:"grade_num"`          // 年级编号
	Stage            string     `json:"stage"`              // 学段
	Semester         string     `json:"semester"`           // 学期
	Status           string     `json:"status"`             // 状态
	HasIndex         bool       `json:"has_index"`          // 是否已有索引
	IndexPageCount   int        `json:"index_page_count"`   // 索引页面数
	IndexTotalLength int        `json:"index_total_length"` // 索引总字符数
	IndexFetchedAt   *time.Time `json:"index_fetched_at"`   // 索引拉取时间
	CreatedAt        *time.Time `json:"created_at"`         // 创建时间
	UpdatedAt        *time.Time `json:"updated_at"`         // 更新时间
}

// ==================== 课程索引模型（对应 course_indexes 表） ====================

// CourseIndex 课程索引（安全数据，仅admin可查看完整内容）
type CourseIndex struct {
	ID           string     `json:"id"`            // UUID主键
	CourseID     string     `json:"course_id"`     // 关联课程ID
	IndexContent string     `json:"index_content"` // TE-DNA编码索引完整内容
	IndexHash    string     `json:"index_hash"`    // SHA-256校验码
	PageCount    int        `json:"page_count"`    // 页面数
	TotalLength  int        `json:"total_length"`  // 总字符数
	FetchedAt    *time.Time `json:"fetched_at"`    // 拉取时间
}

// CourseIndexFullResponse 完整索引响应（仅admin）
type CourseIndexFullResponse struct {
	CourseCode   string     `json:"course_code"`   // 课程编号
	CourseName   string     `json:"course_name"`   // 课程名称
	ModuleID     *int       `json:"module_id"`     // 外部模块ID
	IndexContent string     `json:"index_content"` // 完整索引内容
	IndexHash    string     `json:"index_hash"`    // SHA-256校验码
	PageCount    int        `json:"page_count"`    // 页面数
	TotalLength  int        `json:"total_length"`  // 总字符数
	FetchedAt    *time.Time `json:"fetched_at"`    // 拉取时间
}

// CourseIndexSummaryResponse 索引摘要响应（非admin）
type CourseIndexSummaryResponse struct {
	CourseCode  string   `json:"course_code"`  // 课程编号
	CourseName  string   `json:"course_name"`  // 课程名称
	PageCount   int      `json:"page_count"`   // 页面数
	TotalLength int      `json:"total_length"` // 总字符数
	PageTitles  []string `json:"page_titles"`  // 页面标题列表（脱敏）
	HasIndex    bool     `json:"has_index"`    // 是否已有索引
}

// ==================== 请求结构体 ====================

// CreateCourseRequest 注册课程请求（admin从OSS目录选择注册）
type CreateCourseRequest struct {
	ExternalModuleID int    `json:"external_module_id"` // OSS模块ID（必填）
	CourseCode       string `json:"course_code"`        // 课程编号（必填，如G1-01）
	CourseName       string `json:"course_name"`        // 课程名称（可选，不填则从OSS读取）
	GradeNum         *int   `json:"grade_num"`          // 年级编号（可选）
	Stage            string `json:"stage"`              // 学段（可选）
	Semester         string `json:"semester"`            // 学期（可选）
}

// ==================== OSS 数据结构 ====================

// OSSCatalog OSS全局目录结构（对应 catalog.json）
type OSSCatalog struct {
	Version      json.Number      `json:"version"`       // 版本号（可能是字符串或数字）
	TotalModules int              `json:"total_modules"` // 总模块数
	TotalLessons int              `json:"total_lessons"` // 总课时数
	Modules      []*OSSModule     `json:"modules"`       // 模块列表
	GeneratedAt  string           `json:"generated_at"`  // 生成时间
}

// OSSModule OSS模块信息（catalog.json中的单条）
type OSSModule struct {
	ID          int    `json:"id"`           // 模块ID
	Name        string `json:"name"`         // 模块名称
	LessonCount int    `json:"lesson_count"` // 课时数
	Status      int    `json:"status"`       // 状态（1=启用）
}

// OSSModuleDetail OSS模块详情（对应 modules/{id}.json）
type OSSModuleDetail struct {
	ID             int          `json:"id"`              // 模块ID
	Name           string       `json:"name"`            // 模块名称
	CourseSubjects string       `json:"course_subjects"` // 课程类别JSON字符串
	Status         int          `json:"status"`          // 状态
	Lessons        []*OSSLesson `json:"lessons"`         // 课时列表
	UpdatedAt      string       `json:"updated_at"`      // 更新时间
	SyncedAt       string       `json:"synced_at"`       // 同步时间
}

// OSSLesson OSS课时信息
type OSSLesson struct {
	ID              int    `json:"id"`               // 课时ID
	Title           string `json:"title"`            // 课时标题
	Order           int    `json:"order"`            // 排序
	Status          int    `json:"status"`           // 状态
	StudentDisabled int    `json:"student_disabled"` // 是否对学生禁用
}

// OSSIndexFile OSS索引文件结构（对应 indexes/{module_id}.json）
type OSSIndexFile struct {
	ModuleID   int              `json:"module_id"`   // 模块ID
	ModuleName string           `json:"module_name"` // 模块名称
	Indexes    []*OSSIndexEntry `json:"indexes"`     // 索引条目列表
}

// OSSIndexEntry OSS索引单条记录
type OSSIndexEntry struct {
	ID        int    `json:"id"`         // 索引记录ID
	Name      string `json:"name"`       // 页面名称（如 P01-课程封面-V1.0）
	Content   string `json:"content"`    // TE-DNA编码索引内容
	Remark    string `json:"remark"`     // 备注
	SortOrder int    `json:"sort_order"` // 排序
	CreatedAt string `json:"created_at"` // 创建时间
	UpdatedAt string `json:"updated_at"` // 更新时间
}

// ==================== OSS目录响应（给前端展示可注册课程） ====================

// OSSCatalogResponse OSS目录响应（展示所有可注册模块）
type OSSCatalogResponse struct {
	Version      json.Number          `json:"version"`       // 目录版本
	TotalModules int                  `json:"total_modules"` // 总模块数
	TotalLessons int                  `json:"total_lessons"` // 总课时数
	Modules      []*OSSModuleListItem `json:"modules"`       // 模块列表
	GeneratedAt  string               `json:"generated_at"`  // 生成时间
}

// OSSModuleListItem 单个模块展示项（含注册状态）
type OSSModuleListItem struct {
	ID           int    `json:"id"`            // 模块ID
	Name         string `json:"name"`          // 模块名称
	LessonCount  int    `json:"lesson_count"`  // 课时数
	Status       int    `json:"status"`        // OSS状态
	IsRegistered bool   `json:"is_registered"` // 是否已注册到本系统
	CourseCode   string `json:"course_code"`   // 已注册的课程编号（已注册时有值）
	HasIndex     bool   `json:"has_index"`     // OSS上是否有索引文件
}

// ==================== 常量 ====================

// 课程状态
const (
	CourseStatusActive   = "active"   // 活跃
	CourseStatusArchived = "archived" // 归档
)

// 学段
const (
	StagePrimary = "primary" // 小学
	StageMiddle  = "middle"  // 初中
	StageHigh    = "high"    // 高中
)

// ValidCourseStatuses 合法的课程状态列表
var ValidCourseStatuses = []string{CourseStatusActive, CourseStatusArchived}

// ValidStages 合法的学段列表
var ValidStages = []string{StagePrimary, StageMiddle, StageHigh, ""}

// IsValidCourseStatus 检查课程状态是否合法
func IsValidCourseStatus(status string) bool {
	for _, s := range ValidCourseStatuses {
		if s == status {
			return true
		}
	}
	return false
}
