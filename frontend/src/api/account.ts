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

// ==================== 组织归属类型（个人中心「我的组织」Tab） ====================

/**
 * 我所属的单个学校（来自 school_members 直接成员名单）
 * region_* 可能为空串（学校未挂区域）
 */
export interface UserOrgSchoolItem {
  school_id: string
  school_name: string
  region_id: string
  region_name: string
  source: string          // 入校来源（group_member/admin_create/...）
  is_school_admin: boolean // 我是否为该校的学校管理员
}

/**
 * 我所在的单个教研组（含我在该组的角色 + 该组所属学校/区域）
 * my_role: member=普通成员 / backbone=骨干教师 / lead=教研组长
 */
export interface UserOrgGroupItem {
  group_id: string
  group_name: string
  subject: string
  grade_range: string
  my_role: string
  school_id: string
  school_name: string
  region_id: string
  region_name: string
}

/** 个人组织归属聚合结果 */
export interface UserOrganizationProfile {
  schools: UserOrgSchoolItem[]
  groups: UserOrgGroupItem[]
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

/**
 * 获取当前用户的组织归属（区域/学校/教研组 + 我在各组的职位角色）
 * 只查自己（后端从 JWT 取 userID），登录即可。
 */
export async function getMyOrganization(): Promise<UserOrganizationProfile> {
  const res = await client.get<{ code: number; data: UserOrganizationProfile }>('/account/organization')
  return res.data.data!
}
