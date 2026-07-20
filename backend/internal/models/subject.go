package models

// subject.go — 统一课程定义与教育域课程目录模型
//
// 系统中的“课程”分为两个层次：
//
//   1. subjects
//      全平台统一课程定义，例如“语文”“机械制图”“药学”。
//      负责课程名称、索引编码、全局启停和基础排序。
//
//   2. subject_catalog_entries
//      课程在具体教育域和学校中的可见目录。
//      负责教育域、适用学校、域内展示名、域内排序和目录启停。
//
// 课程定义本身不能决定教师是否可见。
// 普通教师只能看到当前教育域公共目录，以及当前学校的专属目录。
// 因此后台新增和编辑课程时，必须能够同时维护两层数据。

import "time"

/* ==================== 统一课程定义 ==================== */

// Subject 对应subjects表的一行。
//
// 该结构继续用于公开课程目录响应，避免把管理端目录配置
// 暴露给普通教师接口。
type Subject struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	SortOrder int       `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
	IsSystem  bool      `json:"is_system"`
	Note      string    `json:"note"`
	UpdatedBy *string   `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

/* ==================== 课程目录归属 ==================== */

// SubjectCatalogEntry 对应subject_catalog_entries表的一行。
//
// OrganizationID为空表示该教育域公共课程；
// 非空表示仅指定学校可见。
type SubjectCatalogEntry struct {
	ID               string    `json:"id"`
	SubjectID        string    `json:"subject_id"`
	EducationDomain  string    `json:"education_domain"`
	OrganizationID   *string   `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	DisplayName      string    `json:"display_name"`
	SortOrder        int       `json:"sort_order"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// SubjectAdminItem 是后台课程管理列表项。
//
// 通过嵌入Subject保持原有字段扁平JSON结构，
// 同时增加catalog_entries供管理页面展示和编辑目录归属。
type SubjectAdminItem struct {
	Subject
	CatalogEntries []*SubjectCatalogEntry `json:"catalog_entries"`
}

// SubjectCatalogEntryRequest 是新增或编辑课程时提交的目录配置。
//
// OrganizationID：
//   - nil：该教育域公共课程；
//   - 非nil：指定学校专属课程。
//
// DisplayName留空时，后端使用课程定义名称。
// SortOrder小于等于0时，后端使用课程定义的基础排序。
// IsActive由管理端明确提交。
type SubjectCatalogEntryRequest struct {
	EducationDomain string  `json:"education_domain"`
	OrganizationID  *string `json:"organization_id"`
	DisplayName     string  `json:"display_name"`
	SortOrder       int     `json:"sort_order"`
	IsActive        bool    `json:"is_active"`
}

/* ==================== 管理端写入请求 ==================== */

// CreateSubjectRequest 新建课程请求。
//
// 新建时必须同时提交至少一条课程目录配置，
// 防止再次产生“课程定义已存在、教师下拉不可见”的孤立记录。
type CreateSubjectRequest struct {
	Name           string                       `json:"name"`
	Code           string                       `json:"code"`
	SortOrder      int                          `json:"sort_order"`
	Note           string                       `json:"note"`
	CatalogEntries []SubjectCatalogEntryRequest `json:"catalog_entries"`
}

// UpdateSubjectRequest 编辑课程请求。
//
// CatalogEntries使用指针表达三态：
//   - nil：本次只修改课程定义，保留现有目录配置；
//   - 非nil空数组：清空该课程的全部目录配置；
//   - 非nil非空数组：使用提交内容完整替换现有目录配置。
//
// 行内启停只提交IsActive，因此不会误删目录配置。
type UpdateSubjectRequest struct {
	Name           *string                       `json:"name"`
	Code           *string                       `json:"code"`
	SortOrder      *int                          `json:"sort_order"`
	IsActive       *bool                         `json:"is_active"`
	Note           *string                       `json:"note"`
	CatalogEntries *[]SubjectCatalogEntryRequest `json:"catalog_entries"`
}
