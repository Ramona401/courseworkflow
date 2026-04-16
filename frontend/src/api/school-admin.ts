/**
 * school-admin.ts — 学校管理员 API 封装
 *
 * 适用角色：
 *   - senior_operator（且后端校验其为某学校 admin_user_id）
 *
 * 说明：
 *   - 全部接口走 /api/v1/school-admin/*
 *   - 权限由后端严格控制，前端仅做展示层控制
 */
import client from './client'

// ==================== 类型定义：学校 ====================

export interface SchoolInfo {
  id: string
  name: string
  type: string
  status: string
  created: string | null
}

// ==================== 类型定义：教师 ====================

export interface SchoolUserItem {
  id: string
  username: string
  display_name: string
  role: 'operator' | 'viewer' | 'senior_operator' | 'admin'
  status: 'active' | 'disabled'
  login_count: number
  last_login_at: string | null
  created_at: string | null
  school_id?: string | null
  has_teaching_profile?: boolean
}

export interface SchoolUserListResult {
  users: SchoolUserItem[]
  total: number
}

export interface CreateSchoolUserRequest {
  username: string
  display_name: string
  password: string
  role: 'operator' | 'viewer'
}

export interface UpdateSchoolUserRequest {
  display_name: string
  role: 'operator' | 'viewer'
}

// ==================== 类型定义：教研组 ====================

export interface SchoolGroupListItem {
  id: string
  name: string
  school_id: string
  school_name: string
  subject: string
  grade_range: string
  lead_user_id: string | null
  lead_user_name: string
  lead_user_names: string
  member_count: number
  status: string
  created_at: string
}

export interface SchoolGroupListResult {
  groups: SchoolGroupListItem[]
  total: number
}

export interface CreateSchoolGroupRequest {
  name: string
  subject: string
  grade_range?: string
  description?: string
}

export interface UpdateSchoolGroupRequest {
  name: string
  subject: string
  grade_range?: string
  description?: string
  status?: string
}

export interface SchoolGroupMemberItem {
  id: string
  user_id: string
  username: string
  display_name: string
  role: 'member' | 'backbone' | 'lead'
  joined_at: string | null
}

// ==================== API：学校信息 ====================

/** 获取当前学校管理员绑定的学校信息 */
export async function getMySchool(): Promise<SchoolInfo> {
  const res = await client.get<{ code: number; data: SchoolInfo }>('/school-admin/my-school')
  return res.data.data!
}

// ==================== API：教师管理 ====================

/** 获取本校教师列表 */
export async function getSchoolUsers(): Promise<SchoolUserListResult> {
  const res = await client.get<{ code: number; data: SchoolUserListResult }>('/school-admin/users')
  return res.data.data!
}

/** 新建本校教师（仅 operator/viewer） */
export async function createSchoolUser(req: CreateSchoolUserRequest): Promise<SchoolUserItem> {
  const res = await client.post<{ code: number; data: SchoolUserItem }>('/school-admin/users', req)
  return res.data.data!
}

/** 获取教师详情 */
export async function getSchoolUserDetail(userId: string): Promise<unknown> {
  const res = await client.get<{ code: number; data: unknown }>(`/school-admin/users/${userId}`)
  return res.data.data!
}

/** 更新教师（显示名+角色） */
export async function updateSchoolUser(userId: string, req: UpdateSchoolUserRequest): Promise<SchoolUserItem> {
  const res = await client.put<{ code: number; data: SchoolUserItem }>(`/school-admin/users/${userId}`, req)
  return res.data.data!
}

/** 启用/禁用教师 */
export async function updateSchoolUserStatus(userId: string, status: 'active' | 'disabled'): Promise<void> {
  await client.put(`/school-admin/users/${userId}/status`, { status })
}

/** 重置教师密码 */
export async function resetSchoolUserPassword(userId: string, newPassword: string): Promise<void> {
  await client.put(`/school-admin/users/${userId}/password`, { new_password: newPassword })
}

// ==================== API：教研组管理 ====================

/** 获取本校教研组列表 */
export async function getSchoolGroups(): Promise<SchoolGroupListResult> {
  const res = await client.get<{ code: number; data: SchoolGroupListResult }>('/school-admin/groups')
  return res.data.data!
}

/** 新建教研组 */
export async function createSchoolGroup(req: CreateSchoolGroupRequest): Promise<SchoolGroupListItem> {
  const res = await client.post<{ code: number; data: SchoolGroupListItem }>('/school-admin/groups', req)
  return res.data.data!
}

/** 更新教研组 */
export async function updateSchoolGroup(groupId: string, req: UpdateSchoolGroupRequest): Promise<void> {
  await client.put(`/school-admin/groups/${groupId}`, req)
}

/** 删除教研组 */
export async function deleteSchoolGroup(groupId: string): Promise<void> {
  await client.delete(`/school-admin/groups/${groupId}`)
}

// ==================== API：教研组成员管理 ====================

/** 获取教研组成员 */
export async function getSchoolGroupMembers(groupId: string): Promise<SchoolGroupMemberItem[]> {
  const res = await client.get<{ code: number; data: SchoolGroupMemberItem[] }>(`/school-admin/groups/${groupId}/members`)
  return res.data.data ?? []
}

/** 添加成员 */
export async function addSchoolGroupMember(
  groupId: string,
  payload: { user_id: string; role?: 'member' | 'backbone' | 'lead' }
): Promise<void> {
  await client.post(`/school-admin/groups/${groupId}/members`, payload)
}

/** 更新成员角色 */
export async function updateSchoolGroupMemberRole(
  groupId: string,
  userId: string,
  role: 'member' | 'backbone' | 'lead'
): Promise<void> {
  await client.put(`/school-admin/groups/${groupId}/members/${userId}`, { role })
}

/** 移除成员 */
export async function removeSchoolGroupMember(groupId: string, userId: string): Promise<void> {
  await client.delete(`/school-admin/groups/${groupId}/members/${userId}`)
}
