/**
 * auth.ts — 认证相关API与登录用户类型
 *
 * 教育域隔离字段：
 *   education_domain
 *   education_org_id
 *   education_domain_ready
 *   education_domain_error
 *   education_profile
 *
 * education_domain_ready/error保持可选，兼容前后端原子部署窗口。
 * 对region_admin而言，业务是否可用必须显式满足
 * education_domain_ready === true，不能根据角色或空教育域自行推断。
 */

import client from './client'
import type { ApiResponse } from './client'
import type {
  EducationDomain,
  EducationProfile,
} from '@/education-domain/types'

export type {
  EducationDomain,
  EducationProfile,
}

export type UserRole =
  | 'admin'
  | 'region_admin'
  | 'district_inspector'
  | 'senior_operator'
  | 'operator'
  | 'viewer'

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

  portal_modules?: Record<string, boolean>

  /**
   * 当前确定性教育域。
   *
   * 区域管理员任命异常时，后端明确返回空字符串，
   * 前端不得将空字符串规范化成K12或mixed用于教学授权。
   */
  education_domain?: EducationDomain | ''

  /** 当前具体教学组织或区域管理员稳定展示组织。 */
  education_org_id?: string

  /**
   * 教育域是否已安全解析完成。
   *
   * region_admin必须显式为true才能进入教学业务。
   */
  education_domain_ready?: boolean

  /** 教育域异常时的统一用户提示。 */
  education_domain_error?: string

  /** 当前教育域页面术语与能力画像。 */
  education_profile?: EducationProfile

  /** 超级管理员标记。 */
  is_super?: boolean
}

export interface LoginResult {
  token: string
  user: UserInfo
}

export async function login(
  username: string,
  password: string,
): Promise<LoginResult> {
  const res = await client.post<ApiResponse<LoginResult>>(
    '/auth/login',
    {
      username,
      password,
    },
  )

  return res.data.data!
}

export async function getMe(): Promise<UserInfo> {
  const res = await client.get<ApiResponse<UserInfo>>('/auth/me')
  return res.data.data!
}

export async function logout(): Promise<void> {
  await client.post('/auth/logout')
}
