/**
 * 认证相关 API
 * - 登录 / 登出 / 获取当前用户
 */
import client from './client'
import type { ApiResponse } from './client'

// 用户信息类型
export interface UserInfo {
  id: string
  username: string
  display_name: string
  role: 'admin' | 'senior_operator' | 'operator' | 'viewer'
  status: string
  last_login_at: string | null
  login_count: number
  org_logo_url: string
  org_name: string
  // v172新增：门户板块可见性。后端按所属组织 settings.portal_modules 计算下发。
  // key 为板块标识：lesson_plan / courseware / workflow，值 true=可见 false=隐藏。
  // 可能为 undefined（老 token 缓存或后端未下发时），前端统一按"缺省可见"兜底处理。
  portal_modules?: Record<string, boolean>
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
