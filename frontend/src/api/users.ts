/**
 * 用户管理 API（P1-4）
 * - 用户CRUD + 状态管理 + 课程分配
 * - 所有接口仅admin可调用
 */
import client from './client'
import type { ApiResponse } from './client'
import type { UserInfo } from './auth'

// ==================== 请求类型 ====================

// 创建用户请求
export interface CreateUserRequest {
  username: string
  display_name: string
  password: string
  role: 'admin' | 'senior_operator' | 'operator' | 'viewer'
}

// 编辑用户请求
export interface UpdateUserRequest {
  display_name: string
  role: 'admin' | 'senior_operator' | 'operator' | 'viewer'
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
