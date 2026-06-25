/**
 * 统一用户管理中心 API 封装
 * 对应后端 /api/v1/admin/* 和 /api/v1/lesson-plans/organizations/* 路由
 * 仅 admin 可调用（路由层保护）
 *
 * v109改动：
 *   - GroupListItem 新增 lead_user_names（所有组长名称，逗号分隔）
 *   - GroupDetail 新增 lead_user_names
 *   - CreateGroupRequest / UpdateGroupRequest 移除 lead_user_id（改由成员角色管理）
 *
 * Phase6.2改动（区域管理员）：
 *   - AdminStats 新增 region_admin_count（后端 stats 已返回，类型对齐避免 TS 报错）
 *   - 新增组织多管理员接口（organization_admins 的 List/Add/Remove），
 *     对应后端 /api/v1/lesson-plans/organizations/{id}/admins，供第二批多管理员 UI 直接调用
 *
 * 合并重构改动（本次，废弃 school-admin 并轨）：
 *   - 末尾新增批量导入用户 API（batchCreateAdminUsers + 相关类型），
 *     对接后端 POST /api/v1/admin/users/batch，替代旧 /school-admin/users/batch。
 *     类型在此独立定义（不再从 school-admin.ts import），因 school-admin.ts 将被删除。
 */
import client from './client'

// ==================== 用户相关类型 ====================

export interface AdminUserListItem {
  id: string
  username: string
  display_name: string
  role: string
  role_name: string
  status: string
  login_count: number
  last_login_at: string | null
  created_at: string
  school_name: string
  group_name: string
  group_role: string
  group_count: number
}

export interface AdminUserListResult {
  users: AdminUserListItem[]
  total: number
  page: number
  page_size: number
}

export interface AdminUserListParams {
  page?: number
  page_size?: number
  role?: string
  status?: string
  keyword?: string
  school_id?: string
  group_id?: string
}

export interface AdminCourseAssignment {
  course_code: string
  course_name: string
  assigned_at: string
}

export interface AdminGroupMembership {
  group_id: string
  group_name: string
  school_name: string
  role: string
  role_name: string
  is_lead: boolean
  joined_at: string
}

export interface AdminUserDetail extends AdminUserListItem {
  course_assignments: AdminCourseAssignment[]
  teaching_groups: AdminGroupMembership[]
}

export interface AdminStats {
  total_users: number
  active_users: number
  disabled_users: number
  total_orgs: number
  total_schools: number
  total_groups: number
  total_members: number
  admin_count: number
  // Phase6.2新增：区域管理员人数（后端统计已返回，可能为 0）
  region_admin_count?: number
  senior_operator_count: number
  operator_count: number
  viewer_count: number
}

export interface AuditLogItem {
  id: string
  user_id: string
  username: string
  display_name: string
  action: string
  action_name: string
  detail: string
  ip: string
  created_at: string
}

export interface AuditLogListResult {
  logs: AuditLogItem[]
  total: number
}

export interface AuditLogQueryParams {
  page?: number
  page_size?: number
  user_id?: string
  username?: string
  action?: string
  start_date?: string
  end_date?: string
}

// ==================== 组织相关类型 ====================

/** 组织列表项（后端 OrganizationListItem） */
export interface OrgListItem {
  id: string
  name: string
  type: string           // region / school
  parent_id: string | null
  parent_name: string
  admin_user_id: string | null
  admin_user_name: string
  logo_url: string
  status: string
  group_count: number
  member_count: number
  settings?: string        // v172新增：组织设置JSON（含 portal_modules），编辑弹窗读取用
  created_at: string
}

/** 创建组织请求 */
export interface CreateOrgRequest {
  name: string
  type: string           // region / school
  parent_id?: string | null
  admin_user_id?: string | null
}

/** 更新组织请求 */
export interface UpdateOrgRequest {
  name: string
  admin_user_id?: string | null
  status?: string
  settings?: string
}

/**
 * 教研组列表项（后端 TeachingGroupListItem）
 * v109改动：新增 lead_user_names（所有组长名称，中文顿号分隔）
 */
export interface GroupListItem {
  id: string
  name: string
  school_id: string
  school_name: string
  subject: string
  grade_range: string
  lead_user_id: string | null    // 兼容保留
  lead_user_name: string         // 兼容保留（第一个组长名称）
  lead_user_names: string        // v109新增：所有组长名称，如"张老师、李老师"
  member_count: number
  status: string
  created_at: string
}

/**
 * 教研组成员列表项（后端 GroupMemberItem）
 * v109改动：role 支持 'lead' / 'backbone' / 'member'
 */
export interface GroupMemberItem {
  id: string
  user_id: string
  username: string
  display_name: string
  role: string           // lead=组长 / backbone=骨干 / member=普通
  joined_at: string | null
}

/**
 * 教研组详情（含成员列表，后端 TeachingGroupDetailResponse）
 * v109改动：新增 lead_user_names
 */
export interface GroupDetail {
  id: string
  name: string
  school_id: string
  school_name: string
  subject: string
  grade_range: string
  lead_user_id: string | null    // 兼容保留
  lead_user_name: string         // 兼容保留
  lead_user_names: string        // v109新增：所有组长名称
  description: string
  settings: string
  status: string
  members: GroupMemberItem[]
  created_at: string
  updated_at: string
}

/**
 * 创建教研组请求
 * v109改动：移除 lead_user_id（改由成员角色管理多组长）
 */
export interface CreateGroupRequest {
  name: string
  school_id: string
  subject: string
  grade_range?: string
  description?: string
}

/**
 * 更新教研组请求
 * v109改动：移除 lead_user_id
 */
export interface UpdateGroupRequest {
  name: string
  subject: string
  grade_range?: string
  description?: string
  status?: string
}

// ==================== 组织多管理员类型（Phase6.2 预埋，第二批 UI 使用）====================

/**
 * 组织管理员列表项（后端 OrganizationAdminItem）
 * 对应 organization_admins 表，一个组织可有多个管理员。
 * role_type：region_admin（区域管理员，挂在 region 组织）/ school_admin（学校管理员，挂在 school 组织）
 */
export interface OrgAdminItem {
  id: string
  org_id: string
  user_id: string
  username: string
  display_name: string
  role_type: string        // region_admin / school_admin
  created_by: string       // 任命人（后端 COALESCE 为字符串，可能为空串）
  created_at: string
}

/** 任命组织管理员请求 */
export interface AddOrgAdminRequest {
  user_id: string
  role_type: string        // region_admin / school_admin
}

// ==================== 统计 API ====================

export async function getAdminStats(): Promise<AdminStats> {
  const res = await client.get<{ code: number; data: AdminStats }>('/admin/stats')
  return res.data.data!
}

// ==================== 用户管理 API ====================

export async function getAdminUsers(params: AdminUserListParams = {}): Promise<AdminUserListResult> {
  const res = await client.get<{ code: number; data: AdminUserListResult }>('/admin/users', { params })
  return res.data.data!
}

export async function getAdminUserDetail(id: string): Promise<AdminUserDetail> {
  const res = await client.get<{ code: number; data: AdminUserDetail }>(`/admin/users/${id}`)
  return res.data.data!
}

export async function createAdminUser(data: {
  username: string; display_name: string; password: string; role: string
}): Promise<AdminUserListItem> {
  const res = await client.post<{ code: number; data: AdminUserListItem }>('/admin/users', data)
  return res.data.data!
}

export async function updateAdminUser(id: string, data: {
  display_name: string; role: string
}): Promise<AdminUserListItem> {
  const res = await client.put<{ code: number; data: AdminUserListItem }>(`/admin/users/${id}`, data)
  return res.data.data!
}

export async function updateAdminUserStatus(id: string, status: 'active' | 'disabled'): Promise<void> {
  await client.put(`/admin/users/${id}/status`, { status })
}

export async function resetAdminUserPassword(id: string, new_password: string): Promise<void> {
  await client.put(`/admin/users/${id}/password`, { new_password })
}

export async function getAdminUserAssignments(id: string): Promise<AdminCourseAssignment[]> {
  const res = await client.get<{ code: number; data: AdminCourseAssignment[] }>(`/admin/users/${id}/assignments`)
  return res.data.data ?? []
}

export async function updateAdminUserAssignments(id: string, course_codes: string[]): Promise<void> {
  await client.put(`/admin/users/${id}/assignments`, { course_codes })
}

// ==================== 用户↔教研组双向分配 API ====================

export async function addUserToGroup(
  userId: string,
  data: { group_id: string; role: string }
): Promise<void> {
  await client.post(`/admin/users/${userId}/groups`, data)
}

export async function removeUserFromGroup(userId: string, groupId: string): Promise<void> {
  await client.delete(`/admin/users/${userId}/groups/${groupId}`)
}

// ==================== 组织管理 API ====================

export async function getAdminOrgs(params?: {
  type?: string
  parent_id?: string
}): Promise<OrgListItem[]> {
  const res = await client.get<{
    code: number
    data: { organizations: OrgListItem[]; total: number }
  }>('/lesson-plans/organizations', { params })
  return res.data.data?.organizations ?? []
}

export async function getAdminOrg(id: string): Promise<OrgListItem> {
  const res = await client.get<{ code: number; data: OrgListItem }>(
    `/lesson-plans/organizations/${id}`
  )
  return res.data.data!
}

export async function createAdminOrg(data: CreateOrgRequest): Promise<OrgListItem> {
  const res = await client.post<{ code: number; data: OrgListItem }>(
    '/lesson-plans/organizations', data
  )
  return res.data.data!
}

export async function updateAdminOrg(id: string, data: UpdateOrgRequest): Promise<void> {
  await client.put(`/lesson-plans/organizations/${id}`, data)
}

export async function deleteAdminOrg(id: string): Promise<void> {
  await client.delete(`/lesson-plans/organizations/${id}`)
}

/** 上传组织Logo */
export async function uploadOrgLogo(orgId: string, file: File): Promise<{ url: string }> {
  const formData = new FormData()
  formData.append('file', file)
  const res = await client.post<{ code: number; data: { url: string } }>(`/admin/orgs/${orgId}/upload-logo`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data.data!
}

// ==================== 组织多管理员 API（Phase6.2 预埋，第二批 UI 使用）====================
// 后端路由：/api/v1/lesson-plans/organizations/{id}/admins
//   GET    列出某组织的全部管理员
//   POST   任命一名管理员（body: {user_id, role_type}）
//   DELETE 移除某管理员（/admins/{user_id}）
// 权限：admin 任意；region_admin 仅其管辖区域本身或辖区学校（后端 service 二次校验）

/** 列出某组织的全部管理员 */
export async function getOrgAdmins(orgId: string): Promise<OrgAdminItem[]> {
  const res = await client.get<{ code: number; data: { admins: OrgAdminItem[]; total: number } }>(
    `/lesson-plans/organizations/${orgId}/admins`
  )
  return res.data.data?.admins ?? []
}

/** 任命一名组织管理员 */
export async function addOrgAdmin(orgId: string, data: AddOrgAdminRequest): Promise<void> {
  await client.post(`/lesson-plans/organizations/${orgId}/admins`, data)
}

/** 移除某组织管理员 */
export async function removeOrgAdmin(orgId: string, userId: string): Promise<void> {
  await client.delete(`/lesson-plans/organizations/${orgId}/admins/${userId}`)
}

// ==================== 教研组管理 API ====================

export async function getAdminGroups(school_id?: string): Promise<GroupListItem[]> {
  const res = await client.get<{
    code: number
    data: { groups: GroupListItem[]; total: number }
  }>('/lesson-plans/teaching-groups', {
    params: school_id ? { school_id } : {}
  })
  return res.data.data?.groups ?? []
}

export async function getAdminGroupDetail(id: string): Promise<GroupDetail> {
  const res = await client.get<{ code: number; data: GroupDetail }>(
    `/lesson-plans/teaching-groups/${id}`
  )
  return res.data.data!
}

export async function createAdminGroup(data: CreateGroupRequest): Promise<GroupListItem> {
  const res = await client.post<{ code: number; data: GroupListItem }>(
    '/lesson-plans/teaching-groups', data
  )
  return res.data.data!
}

export async function updateAdminGroup(id: string, data: UpdateGroupRequest): Promise<void> {
  await client.put(`/lesson-plans/teaching-groups/${id}`, data)
}

export async function deleteAdminGroup(id: string): Promise<void> {
  await client.delete(`/lesson-plans/teaching-groups/${id}`)
}

// ==================== 教研组成员管理 API ====================

export async function getAdminGroupMembers(groupId: string): Promise<GroupMemberItem[]> {
  const res = await client.get<{ code: number; data: GroupMemberItem[] }>(
    `/admin/groups/${groupId}/members`
  )
  return res.data.data ?? []
}

export async function addAdminGroupMember(groupId: string, data: {
  user_id: string; role?: string
}): Promise<void> {
  await client.post(`/admin/groups/${groupId}/members`, data)
}

export async function updateAdminGroupMemberRole(
  groupId: string, userId: string, role: string
): Promise<void> {
  await client.put(`/admin/groups/${groupId}/members/${userId}`, { role })
}

export async function removeAdminGroupMember(groupId: string, userId: string): Promise<void> {
  await client.delete(`/admin/groups/${groupId}/members/${userId}`)
}

// ==================== 操作日志 API ====================

export async function getAdminAuditLogs(
  params: AuditLogQueryParams = {}
): Promise<AuditLogListResult> {
  const res = await client.get<{ code: number; data: AuditLogListResult }>(
    '/admin/audit-logs', { params }
  )
  return res.data.data!
}

// ==================== 批量导入用户 API（合并重构：替代 school-admin 批量端点）====================
// 后端路由：POST /api/v1/admin/users/batch（adminOrSchoolAdmin 中间件）
//   - admin          : 必须传 school_id（目标学校），后端按此入校；为空后端返回 400
//   - senior_operator: 可省 school_id，后端强制本校（忽略前端传入）
//   - 角色白名单：operator / viewer（后端二次强制）
//   - 整批回滚：任一行失败则一个都不创建，返回行号失败明细供前端逐行标红

/** 批量导入的单个用户条目（与后端 services.BatchUserItem 对齐） */
export interface BatchUserItem {
  username: string
  display_name: string
  password: string
}

/** 批量建用户请求（前端只传 role + school_id + users；source 由后端按角色强制） */
export interface BatchCreateAdminUsersRequest {
  role: string                 // operator / viewer
  school_id?: string           // admin 必填（目标学校）；senior 可省（后端强制本校）
  users: BatchUserItem[]
}

/** 单行失败明细（行号 1-based，对齐操作员看到的表格行） */
export interface BatchCreateUserFailure {
  index: number
  username: string
  reason: string
}

/** 批量建用户结果（与后端 services.BatchCreateUsersResult 对齐） */
export interface BatchCreateUsersResult {
  success: boolean
  created_count: number
  total_count: number
  failures: BatchCreateUserFailure[]
}

/**
 * 批量创建用户 POST /admin/users/batch
 * 后端整批回滚 + 行号失败明细：
 *   - 成功 → success=true, created_count=条目数, failures=[]
 *   - 整批校验失败 → success=false, created_count=0, failures 逐行原因（HTTP 仍 200）
 *   - 系统级异常（如空列表、开事务失败）→ 抛错（HTTP 非 200，由调用方 catch）
 */
export async function batchCreateAdminUsers(
  req: BatchCreateAdminUsersRequest
): Promise<BatchCreateUsersResult> {
  const res = await client.post<{ code: number; data: BatchCreateUsersResult }>(
    '/admin/users/batch', req
  )
  return res.data.data!
}


// ==================== 学校境外模型授权策略 API（批二新增，admin专属）====================
// 后端路由：/api/v1/admin/school-model-policies（adminOnly）
//   GET    /school-model-policies            列出全部已登记策略的学校
//   GET    /school-model-policies/{schoolID} 查单校当前策略
//   PUT    /school-model-policies/{schoolID} 授权/取消授权（body: {overseas_enabled, note}）
//   DELETE /school-model-policies/{schoolID} 删除记录（=回到默认境内）
// 业务：默认所有学校只能用境内模型(qwen-max)；仅被授权的学校放行境外模型(claude/gemini等)。

/** 学校授权策略列表项（后端 SchoolModelPolicyItem） */
export interface SchoolModelPolicyItem {
  school_id: string
  school_name: string          // JOIN organizations 填学校名
  overseas_enabled: boolean    // 是否授权境外模型
  note: string                 // 备注
  granted_by_name: string      // 授权人显示名（可空时为空串）
  created_at: string
  updated_at: string
}

/** 单校策略视图（后端 smpPolicyView），无记录时 has_record=false 表默认境内 */
export interface SchoolModelPolicyView {
  school_id: string
  overseas_enabled: boolean
  note: string
  has_record: boolean          // false=该校从未登记策略（=默认境内）
}

/** 列出全部已登记策略的学校 */
export async function getSchoolModelPolicies(): Promise<SchoolModelPolicyItem[]> {
  const res = await client.get<{ code: number; data: { items: SchoolModelPolicyItem[]; total: number } }>(
    '/admin/school-model-policies'
  )
  return res.data.data?.items ?? []
}

/** 查单个学校的当前策略（无记录返回默认境内态 has_record=false） */
export async function getSchoolModelPolicy(schoolId: string): Promise<SchoolModelPolicyView> {
  const res = await client.get<{ code: number; data: SchoolModelPolicyView }>(
    `/admin/school-model-policies/${schoolId}`
  )
  return res.data.data!
}

/** 授权/取消授权某学校走境外模型（overseas_enabled + 可选 note）
 *  分流模块对「学校是否授权」是每次实时查库，无缓存，保存后即时生效 */
export async function setSchoolModelPolicy(
  schoolId: string,
  data: { overseas_enabled: boolean; note?: string }
): Promise<SchoolModelPolicyView> {
  const res = await client.put<{ code: number; data: SchoolModelPolicyView }>(
    `/admin/school-model-policies/${schoolId}`,
    { overseas_enabled: data.overseas_enabled, note: data.note ?? '' }
  )
  return res.data.data!
}

/** 删除某学校策略记录（=回到默认境内） */
export async function deleteSchoolModelPolicy(schoolId: string): Promise<void> {
  await client.delete(`/admin/school-model-policies/${schoolId}`)
}
