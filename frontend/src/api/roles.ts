/**
 * roles.ts — 角色权限管理 API 封装
 *
 * 对应后端 /api/v1/admin/roles/* 路由（仅 admin 可调用）
 *
 * 任务七：角色权限 Tab 前端
 * 包含：角色列表/详情/创建/编辑/状态/删除/权限查询/权限更新
 */
import client from './client'

// ==================== 类型定义 ====================

/** 角色列表项 */
export interface RoleListItem {
  id: string
  role_code: string           // 英文标识，如 admin / custom_teacher
  display_name: string        // 中文名，如 管理员
  description: string
  base_role: string           // 继承自哪个内置角色
  is_system: boolean          // true=系统内置，不可编辑/删除
  status: string              // active / disabled
  permission_count: number    // 权限数量
  user_count: number          // 已分配用户数
  created_at: string
}

/** 角色列表响应 */
export interface RoleListResponse {
  roles: RoleListItem[]
  total: number
}

/** 单条权限项 */
export interface PermissionItem {
  id: string
  role_id: string
  permission_code: string   // 格式：resource.action，如 pipeline.create
  resource: string          // 资源分类，如 pipeline
  action: string            // 动作，如 create
  created_at: string
}

/** 角色详情（含权限列表） */
export interface RoleDetail extends RoleListItem {
  permissions: PermissionItem[]
}

/** 创建角色请求 */
export interface CreateRoleRequest {
  role_code: string
  display_name: string
  description?: string
  base_role?: string          // 继承角色，选填；填写后自动复制权限
}

/** 更新角色基本信息请求 */
export interface UpdateRoleRequest {
  display_name: string
  description?: string
}

/** 更新角色状态请求 */
export interface UpdateRoleStatusRequest {
  status: 'active' | 'disabled'
}

/** 更新角色权限请求 */
export interface UpdateRolePermissionsRequest {
  permissions: Array<{
    permission_code: string
    resource: string
    action: string
  }>
}

// ==================== API 函数 ====================

/** 获取角色列表（含系统内置 + 自定义） */
export async function listRoles(): Promise<RoleListResponse> {
  const res = await client.get<{ code: number; data: RoleListResponse }>('/admin/roles')
  return res.data.data!
}

/** 获取角色详情（含权限列表） */
export async function getRoleDetail(id: string): Promise<RoleDetail> {
  const res = await client.get<{ code: number; data: RoleDetail }>(`/admin/roles/${id}`)
  return res.data.data!
}

/** 创建自定义角色 */
export async function createRole(data: CreateRoleRequest): Promise<RoleListItem> {
  const res = await client.post<{ code: number; data: RoleListItem }>('/admin/roles', data)
  return res.data.data!
}

/** 更新角色基本信息（仅中文名+描述，is_system=true 时后端拒绝） */
export async function updateRole(id: string, data: UpdateRoleRequest): Promise<void> {
  await client.put(`/admin/roles/${id}`, data)
}

/** 更新角色启用/禁用状态 */
export async function updateRoleStatus(
  id: string,
  data: UpdateRoleStatusRequest
): Promise<void> {
  await client.put(`/admin/roles/${id}/status`, data)
}

/** 删除角色（后端检查 is_system 和用户数） */
export async function deleteRole(id: string): Promise<void> {
  await client.delete(`/admin/roles/${id}`)
}

/** 获取角色权限列表 */
export async function getRolePermissions(id: string): Promise<PermissionItem[]> {
  const res = await client.get<{ code: number; data: PermissionItem[] }>(
    `/admin/roles/${id}/permissions`
  )
  return res.data.data ?? []
}

/** 全量替换角色权限 */
export async function updateRolePermissions(
  id: string,
  data: UpdateRolePermissionsRequest
): Promise<void> {
  await client.put(`/admin/roles/${id}/permissions`, data)
}
