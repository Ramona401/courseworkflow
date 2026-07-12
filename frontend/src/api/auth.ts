/**
 * 认证相关 API
 * - 登录 / 登出 / 获取当前用户
 */
import client from './client'
import type { ApiResponse } from './client'

/**
 * 系统角色联合类型（与后端 models/user.go 的 ValidRoles 对齐）
 * - admin            系统管理员
 * - region_admin     区域管理员（迭代一新增：管辖区域下的学校与用户、任命学校管理员、分配积分）
 * - district_inspector 区域教研员（管区域抽查，与 region_admin 并存互不替代）
 * - senior_operator  学校管理员
 * - operator         骨干教师
 * - viewer           普通教师
 */
export type UserRole =
  | 'admin'
  | 'region_admin'
  | 'district_inspector'
  | 'senior_operator'
  | 'operator'
  | 'viewer'

// 用户信息类型
export interface UserInfo {
  id: string
  username: string
  display_name: string
  role: UserRole
  status: string
  last_login_at: string | null
  login_count: number
  org_logo_url: string
  org_name: string
  // v172新增：门户板块可见性。后端按所属组织 settings.portal_modules 计算下发。
  // key 为板块标识：lesson_plan / courseware / workflow，值 true=可见 false=隐藏。
  // 可能为 undefined（老 token 缓存或后端未下发时），前端统一按“缺省可见”兜底处理。
  portal_modules?: Record<string, boolean>
  // 超管收口新增：超级管理员标记位（后端 models.UserInfo.IsSuper 已下发，json 键 is_super）。
  //   - true：真超管，可见"积分管理 / AI管理中心 / AI调用统计 / 提示词管理"等敏感入口；
  //   - false：二线管理员（仍是 admin 角色），前端隐藏上述四个敏感入口，只保留
  //            用户管理 / 基础数据管理（学科·课程大纲）/ 组织架构等常规管理能力。
  // 与 role 正交：仅在 role='admin' 时有区分意义。可能为 undefined（老缓存/后端未下发），
  // 前端一律按"非超管"兜底（缺省 false，收紧方向，不会误放行敏感入口）。
  is_super?: boolean
}

// 登录响应类型
export interface LoginResult {
  token: string
  user: UserInfo
}

// 登录
export async function login(username: string, password: string): Promise<LoginResult> {
  const res = await client.post<ApiResponse<LoginResult>>('/auth/login', {
    username,
    password,
  })
  return res.data.data!
}

// 获取当前用户信息
export async function getMe(): Promise<UserInfo> {
  const res = await client.get<ApiResponse<UserInfo>>('/auth/me')
  return res.data.data!
}

// 登出
export async function logout(): Promise<void> {
  await client.post('/auth/logout')
}
