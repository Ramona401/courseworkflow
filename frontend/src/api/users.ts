/**
 * 用户管理 API（P1-4）
 * - 用户CRUD + 状态管理 + 课程分配
 * - 所有接口仅admin可调用
 *
 * Phase6.2改动（区域管理员）：
 *   - CreateUserRequest / UpdateUserRequest 的 role 字段改为复用 auth.ts 的 UserRole 联合类型，
 *     与全局角色定义统一（新增 region_admin / district_inspector 后不再产生类型不兼容）。
 *   - 说明：本页（/workflow 旧用户管理页）的角色下拉仍只提供 admin/senior_operator/operator/viewer
 *     四个常用选项，区域级账号由系统管理员在 /admin 统一开通；此处仅放宽类型以保证编译与展示兼容。
 */
import client from './client'
import type { ApiResponse } from './client'
import type { UserInfo, UserRole } from './auth'
// ==================== 请求类型 ====================
// 创建用户请求
export interface CreateUserRequest {
  username: string
  display_name: string
  password: string
  role: UserRole
}
// 编辑用户请求
export interface UpdateUserRequest {
  display_name: string
  role: UserRole
}
// 重置密码请求
export interface ResetPasswordRequest {
  new_password: string
}
// 更新状态请求
export interface UpdateStatusRequest {
  status: 'active' | 'disabled'
}
// 更新课程分配请求
export interface UpdateAssignmentsRequest {
  course_codes: string[]
}
// ==================== 响应类型 ====================
// 用户列表响应
export interface UserListResponse {
  users: UserInfo[]
  total: number
}
// 课程分配记录
export interface CourseAssignment {
  id: string
  user_id: string
  course_code: string
  assigned_by: string
  assigned_at: string | null
}
// ==================== API 方法 ====================
// 获取用户列表
export async function getUsers(): Promise<UserListResponse> {
  const res = await client.get<ApiResponse<UserListResponse>>('/users')
  return res.data.data!
}
// 创建用户
export async function createUser(data: CreateUserRequest): Promise<UserInfo> {
  const res = await client.post<ApiResponse<UserInfo>>('/users', data)
  return res.data.data!
}
// 编辑用户
export async function updateUser(id: string, data: UpdateUserRequest): Promise<UserInfo> {
  const res = await client.put<ApiResponse<UserInfo>>(`/users/${id}`, data)
  return res.data.data!
}
// 重置密码
export async function resetPassword(id: string, data: ResetPasswordRequest): Promise<void> {
  await client.put(`/users/${id}/password`, data)
}
// 更新用户状态
export async function updateUserStatus(id: string, data: UpdateStatusRequest): Promise<void> {
  await client.put(`/users/${id}/status`, data)
}
// 获取用户课程分配
export async function getUserAssignments(id: string): Promise<CourseAssignment[]> {
  const res = await client.get<ApiResponse<CourseAssignment[]>>(`/users/${id}/assignments`)
  return res.data.data!
}
// 更新用户课程分配
export async function updateUserAssignments(id: string, data: UpdateAssignmentsRequest): Promise<CourseAssignment[]> {
  const res = await client.put<ApiResponse<CourseAssignment[]>>(`/users/${id}/assignments`, data)
  return res.data.data!
}
