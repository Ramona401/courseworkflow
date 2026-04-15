/**
 * 统一用户管理中心 API 封装
 * 对应后端 /api/v1/admin/* 和 /api/v1/lesson-plans/organizations/* 路由
 * 仅 admin 可调用（路由层保护）
 *
 * v109改动：
 *   - GroupListItem 新增 lead_user_names（所有组长名称，逗号分隔）
 *   - GroupDetail 新增 lead_user_names
 *   - CreateGroupRequest / UpdateGroupRequest 移除 lead_user_id（改由成员角色管理）
 *   - GroupMemberItem.role 支持 'lead' / 'backbone' / 'member'
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
  status: string
  group_count: number
  member_count: number
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
