package services

// permission_matrix.go — 全系统唯一的权威权限矩阵（迭代一·组织与权限重铸）
//
// 设计目标（对应规划"核心洞察"）：
//   把散落在各 handler 里的"谁能对什么资源做什么操作"判断，收口为唯一一处权威矩阵。
//   以后所有"角色 × 资源 × 操作"的判定都问 CanAccess，不在 handler 内散写权限逻辑。
//
// 本文件特性：
//   - 纯静态：只是查表返回 bool，零依赖、零 I/O、不碰数据库，可纯单元测试。
//   - 数据范围解析（ResolveDataScope，依赖数据库）拆到独立文件 data_scope.go，
//     因其职责（重 I/O）与本文件（纯静态）不同，便于各自测试与维护。
//
// fail-closed 原则：矩阵未显式授予的组合，一律返回 false（拒绝）。
//
// 当前阶段说明（迭代一 Phase 1）：
//   本矩阵先建立权威定义，尚未接入各现有 handler 的判断点。
//   现有 handler 的权限判断在 Phase 4/5 逐步改为调用 CanAccess，
//   届时散落逻辑收口到此处。Phase 1 纯新增、不动任何现有逻辑。

import (
	"tedna/internal/models"
)

// ==================== 资源常量（Resource）====================
//
// 资源粒度对齐规划的权限矩阵表行。命名用领域名词，避免与具体表名耦合。
const (
	ResourceOrgRegion  = "organization_region" // 区域组织（region 类型的 organizations）
	ResourceOrgSchool  = "organization_school" // 学校组织（school 类型的 organizations）
	ResourceUser       = "user"                // 教师用户账号
	ResourceOrgAdmin   = "org_admin"           // 组织管理员任命（organization_admins）
	ResourceLessonPlan = "lesson_plan"         // 教案
	ResourceSystemRole = "system_role"         // 系统角色修改（改 users.role）
	ResourceToken      = "token"               // 积分账户/分配/采购等
	ResourceBatchUser  = "batch_user"          // 批量建账号
)

// ==================== 操作常量（Action）====================
const (
	ActionView   = "view"   // 查看/列表
	ActionCreate = "create" // 新建
	ActionEdit   = "edit"   // 编辑
	ActionDelete = "delete" // 删除
	ActionAssign = "assign" // 分配/任命（如任命管理员、分配积分）
)

// permissionKey 矩阵查表用的复合键（角色|资源|操作）
type permissionKey struct {
	role     string
	resource string
	action   string
}

// permissionMatrix 权威权限矩阵
//
// 落地规划那张表（仅登记"被授予"的组合，未登记的组合 CanAccess 返回 false=拒绝）：
//
//   资源\角色          | admin | region_admin    | senior_operator | operator/viewer
//   区域组织           | CRUD  | view/edit本区域 | ✗               | ✗
//   学校组织           | CRUD  | v/e/c本区域下   | view/edit本校   | ✗
//   教师用户           | 全部  | ✗(经校管)       | 建管本校        | 仅自己(view)
//   组织管理员任命     | 全部  | 任命本区域校管  | ✗               | ✗
//   教案查看           | view  | view本区域树    | view本校        | view本人+共享
//   系统角色修改       | 全部  | ✗               | ✗               | ✗
//   批量建账号         | 全部  | ✗               | 本校op/viewer   | ✗
//
// 注意：本矩阵只回答"该角色对该资源能否做该操作"（粗粒度门禁），
//   "具体能操作哪些数据行"（细粒度数据范围，如"本校""本区域树"）由 ResolveDataScope 解决。
//   两者配合：CanAccess 先判门禁，ResolveDataScope 再框定数据行。
var permissionMatrix = func() map[permissionKey]bool {
	m := make(map[permissionKey]bool)

	// grant 批量登记某角色对某资源的多个操作为允许
	grant := func(role, resource string, actions ...string) {
		for _, a := range actions {
			m[permissionKey{role: role, resource: resource, action: a}] = true
		}
	}

	// ---------- admin：全系统所有资源所有操作 ----------
	allResources := []string{
		ResourceOrgRegion, ResourceOrgSchool, ResourceUser, ResourceOrgAdmin,
		ResourceLessonPlan, ResourceSystemRole, ResourceToken, ResourceBatchUser,
	}
	allActions := []string{ActionView, ActionCreate, ActionEdit, ActionDelete, ActionAssign}
	for _, r := range allResources {
		grant(models.RoleAdmin, r, allActions...)
	}

	// ---------- region_admin：区域管理员 ----------
	// 区域组织：可查看、编辑本区域树（不可删除区域，删除区域是 admin 专属高危操作）
	grant(models.RoleRegionAdmin, ResourceOrgRegion, ActionView, ActionEdit)
	// 学校组织：可在本区域下查看/编辑/新建学校
	grant(models.RoleRegionAdmin, ResourceOrgSchool, ActionView, ActionCreate, ActionEdit)
	// 组织管理员任命：可在本区域下任命学校管理员
	grant(models.RoleRegionAdmin, ResourceOrgAdmin, ActionView, ActionAssign)
	// 教案：可查看本区域树（不直接编辑老师教案）
	grant(models.RoleRegionAdmin, ResourceLessonPlan, ActionView)
	// 积分：可查看/分配本区域树范围（迭代二三级分配依赖）
	grant(models.RoleRegionAdmin, ResourceToken, ActionView, ActionAssign)
	// 注意：region_admin 不直接建老师（ResourceUser create 不授予，经由校管），
	//   也不能改系统角色（ResourceSystemRole 不授予），不能批量建账号。

	// ---------- senior_operator：学校管理员 ----------
	// 学校组织：查看/编辑本校
	grant(models.RoleSeniorOperator, ResourceOrgSchool, ActionView, ActionEdit)
	// 教师用户：在本校建/管（创建+编辑+查看，删除走禁用不在此）
	grant(models.RoleSeniorOperator, ResourceUser, ActionView, ActionCreate, ActionEdit)
	// 教案：查看本校
	grant(models.RoleSeniorOperator, ResourceLessonPlan, ActionView)
	// 积分：查看/分配本校（迭代二学校→老师分配）
	grant(models.RoleSeniorOperator, ResourceToken, ActionView, ActionAssign)
	// 批量建账号：本校 operator/viewer
	grant(models.RoleSeniorOperator, ResourceBatchUser, ActionCreate)

	// ---------- operator / viewer：普通教师 ----------
	// 仅能查看自己的教案（细粒度"本人+共享"由 ResolveDataScope 框定）
	grant(models.RoleOperator, ResourceLessonPlan, ActionView)
	grant(models.RoleViewer, ResourceLessonPlan, ActionView)
	// 用户资源：仅查看自己（细粒度由数据范围控制）
	grant(models.RoleOperator, ResourceUser, ActionView)
	grant(models.RoleViewer, ResourceUser, ActionView)
	// 积分：仅查看自己
	grant(models.RoleOperator, ResourceToken, ActionView)
	grant(models.RoleViewer, ResourceToken, ActionView)

	// ---------- district_inspector：区域教研员 ----------
	// 现有职责仅"区域抽查"，本迭代不扩其权限；教案查看授予（抽查需要），其余默认拒绝。
	grant(models.RoleDistrictInspector, ResourceLessonPlan, ActionView)

	return m
}()

// CanAccess 权威权限判定：给定角色，能否对某资源做某操作
//
// 返回 true 仅当矩阵显式授予；任何未登记组合返回 false（fail-closed 拒绝）。
//
// 注意：这是粗粒度门禁判定（能不能做这类操作），
//   不负责"具体能操作哪几行数据"（那是 ResolveDataScope 的职责）。
//   典型用法：handler 先 CanAccess 判门禁 → 再用 ResolveDataScope 框定数据范围。
func CanAccess(role, resource, action string) bool {
	if role == "" || resource == "" || action == "" {
		return false
	}
	return permissionMatrix[permissionKey{role: role, resource: resource, action: action}]
}
