/**
 * 通用用户中心 API 封装
 * 对应后端 /api/v1/account/* 路由
 * 所有已登录用户均可调用（跨课件审核/教案两个系统共用）
 */
import client from './client'

// ==================== 类型定义 ====================

/** 个人信息响应体 */
export interface ProfileInfo {
  id: string
  username: string
  display_name: string
  role: string
  status: string
  login_count: number
  last_login_at: string | null
  created_at: string
}

/** 更新个人信息请求体 */
export interface UpdateProfileRequest {
  display_name: string
}

/** 修改密码请求体 */
export interface ChangePasswordRequest {
  old_password: string
  new_password: string
}

// ==================== API 函数 ====================

/** 获取当前用户个人信息 */
export async function getProfile(): Promise<ProfileInfo> {
  const res = await client.get<{ code: number; data: ProfileInfo }>('/account/profile')
  return res.data.data!
}

/** 更新当前用户显示名称 */
export async function updateProfile(
  req: UpdateProfileRequest
): Promise<{ message: string; display_name: string }> {
  const res = await client.put<{ code: number; data: { message: string; display_name: string } }>(
    '/account/profile',
    req
  )
  return res.data.data!
}

/** 修改当前用户密码（需验证旧密码） */
export async function changePassword(
  req: ChangePasswordRequest
): Promise<{ message: string }> {
  const res = await client.put<{ code: number; data: { message: string } }>(
    '/account/password',
    req
  )
  return res.data.data!
}
