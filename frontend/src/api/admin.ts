/**
 * 统一用户管理中心 API 封装
 * 对应后端 /api/v1/admin/* 路由
 * 仅 admin 可调用（路由层保护）
 */
import client from './client'

// ==================== 类型定义 ====================

/** 用户列表项（含跨系统权限摘要）*/
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

/** 用户列表分页结果 */
export interface AdminUserListResult {
  users: AdminUserListItem[]
  total: number
  page: number
  page_size: number
}

/** 用户列表查询参数 */
export interface AdminUserListParams {
  page?: number
  page_size?: number
  role?: string
  status?: string
  keyword?: string
  school_id?: string
  group_id?: string
}

/** 课程分配 */
export interface AdminCourseAssignment {
  course_code: string
  course_name: string
  assigned_at: string
}

/** 教研组归属 */
export interface AdminGroupMembership {
  group_id: string
  group_name: string
  school_name: string
  role: string
  role_name: string
  is_lead: boolean
  joined_at: string
}

/** 用户详情（含完整权限全貌）*/
export interface AdminUserDetail extends AdminUserListItem {
  course_assignments: AdminCourseAssignment[]
  teaching_groups: AdminGroupMembership[]
}

/** 管理中心统计摘要 */
export interface AdminStats {
  total_users: number
  active_users: number
  disabled_users: number
  total_orgs: number
  total_schools: number
  total_groups: number
  total_members: number
  admin_count: number
  senior_operator_count: number
  operator_count: number
  viewer_count: number
}

/** 操作日志列表项 */
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

/** 操作日志分页结果 */
export interface AuditLogListResult {
  logs: AuditLogItem[]
  total: number
}

// ==================== 组织相关类型（匹配后端实际返回）====================

/** 组织列表项（后端 OrganizationListItem）*/
export interface OrgListItem {
  id: string
  name: string
  type: string           // region / school
  parent_id: string | null
  parent_name: string
  admin_user_id: string | null
  admin_user_name: string
  status: string
  group_count: number
  member_count: number
  created_at: string
}

/** 教研组列表项（后端 TeachingGroupListItem）*/
export interface GroupListItem {
  id: string
  name: string
  school_id: string
  school_name: string
  subject: string
  grade_range: string
  lead_user_id: string | null
  lead_user_name: string
  member_count: number
  status: string
  created_at: string
}

// ==================== API 函数 ====================

/** 获取统计摘要 */
export async function getAdminStats(): Promise<AdminStats> {
  const res = await client.get<{ code: number; data: AdminStats }>('/admin/stats')
  return res.data.data!
}

/** 获取用户列表 */
export async function getAdminUsers(params: AdminUserListParams = {}): Promise<AdminUserListResult> {
  const res = await client.get<{ code: number; data: AdminUserListResult }>('/admin/users', { params })
  return res.data.data!
}

/** 获取用户详情 */
export async function getAdminUserDetail(id: string): Promise<AdminUserDetail> {
  const res = await client.get<{ code: number; data: AdminUserDetail }>(`/admin/users/${id}`)
  return res.data.data!
}

/** 创建用户 */
export async function createAdminUser(data: {
  username: string; display_name: string; password: string; role: string
}): Promise<AdminUserListItem> {
  const res = await client.post<{ code: number; data: AdminUserListItem }>('/admin/users', data)
  return res.data.data!
}

/** 编辑用户 */
export async function updateAdminUser(id: string, data: {
  display_name: string; role: string
}): Promise<AdminUserListItem> {
  const res = await client.put<{ code: number; data: AdminUserListItem }>(`/admin/users/${id}`, data)
  return res.data.data!
}

/** 启用/禁用用户 */
export async function updateAdminUserStatus(id: string, status: 'active' | 'disabled'): Promise<void> {
  await client.put(`/admin/users/${id}/status`, { status })
}

/** 重置密码 */
export async function resetAdminUserPassword(id: string, new_password: string): Promise<void> {
  await client.put(`/admin/users/${id}/password`, { new_password })
}

/** 获取用户课程分配 */
export async function getAdminUserAssignments(id: string): Promise<AdminCourseAssignment[]> {
  const res = await client.get<{ code: number; data: AdminCourseAssignment[] }>(`/admin/users/${id}/assignments`)
  return res.data.data ?? []
}

/** 更新用户课程分配 */
export async function updateAdminUserAssignments(id: string, course_codes: string[]): Promise<void> {
  await client.put(`/admin/users/${id}/assignments`, { course_codes })
}

/**
 * 获取组织列表
 * 后端返回：{ organizations: [...], total: N }
 */
export async function getAdminOrgs(params?: { type?: string; parent_id?: string }): Promise<OrgListItem[]> {
  const res = await client.get<{
    code: number
    data: { organizations: OrgListItem[]; total: number }
  }>('/admin/orgs', { params })
  return res.data.data?.organizations ?? []
}

/**
 * 获取教研组列表
 * 后端返回：{ groups: [...], total: N }
 */
export async function getAdminGroups(school_id?: string): Promise<GroupListItem[]> {
  const res = await client.get<{
    code: number
    data: { groups: GroupListItem[]; total: number }
  }>('/admin/groups', {
    params: school_id ? { school_id } : {}
  })
  return res.data.data?.groups ?? []
}

/** 获取教研组成员 */
export async function getAdminGroupMembers(groupId: string): Promise<unknown[]> {
  const res = await client.get<{ code: number; data: unknown[] }>(`/admin/groups/${groupId}/members`)
  return res.data.data ?? []
}

/** 添加教研组成员 */
export async function addAdminGroupMember(groupId: string, data: {
  user_id: string; role?: string
}): Promise<void> {
  await client.post(`/admin/groups/${groupId}/members`, data)
}

/** 更新教研组成员角色 */
export async function updateAdminGroupMemberRole(groupId: string, userId: string, role: string): Promise<void> {
  await client.put(`/admin/groups/${groupId}/members/${userId}`, { role })
}

/** 移除教研组成员 */
export async function removeAdminGroupMember(groupId: string, userId: string): Promise<void> {
  await client.delete(`/admin/groups/${groupId}/members/${userId}`)
}

/** 查询操作日志 */
export async function getAdminAuditLogs(params: {
  page?: number; page_size?: number; user_id?: string; action?: string
} = {}): Promise<AuditLogListResult> {
  const res = await client.get<{ code: number; data: AuditLogListResult }>('/admin/audit-logs', { params })
  return res.data.data!
}
