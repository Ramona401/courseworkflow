/**
 * adminUserCreate.ts — 管理中心单用户建号的独立API模块
 *
 * 拆分原因：
 *   admin.ts 已超过项目600行红线。本模块只承载“单用户建号弹窗”需要的三项能力：
 *   1. 查询启用区域；
 *   2. 按区域查询启用学校；
 *   3. 创建用户并提交school_id。
 *
 * 权限与安全：
 *   - 仅系统管理员会在弹窗中查询区域和学校；
 *   - 学校管理员不依赖客户端学校参数，后端会强制使用其真实管理学校；
 *   - 后端仍会校验学校类型、状态、教育域和积分账户父链，前端选项不是安全边界。
 */

import client from './client'

export interface AdminCreateOrganizationOption {
  id: string
  name: string
  type: 'region' | 'school'
  status: string
  education_domain?: 'k12' | 'vocational' | 'adult' | 'mixed'
  parent_id: string | null
}

interface OrganizationListResponse {
  organizations?: AdminCreateOrganizationOption[]
  total?: number
}

export interface CreateAdminUserWithSchoolRequest {
  username: string
  display_name: string
  password: string
  role: string
  school_id?: string
}

export interface CreatedAdminUser {
  id: string
  username: string
  display_name: string
  role: string
  status: string
}

/**
 * listActiveAdminRegions 查询可用于建号选择的启用区域。
 */
export async function listActiveAdminRegions():
Promise<AdminCreateOrganizationOption[]> {
  const response = await client.get<{
    code: number
    data: OrganizationListResponse
  }>('/lesson-plans/organizations', {
    params: {
      type: 'region',
    },
  })

  return (
    response.data.data?.organizations ?? []
  ).filter(item =>
    item.type === 'region' &&
    item.status === 'active'
  )
}

/**
 * listActiveAdminSchoolsByRegion 查询指定区域直接下属的启用学校。
 */
export async function listActiveAdminSchoolsByRegion(
  regionId: string,
): Promise<AdminCreateOrganizationOption[]> {
  const response = await client.get<{
    code: number
    data: OrganizationListResponse
  }>('/lesson-plans/organizations', {
    params: {
      type: 'school',
      parent_id: regionId,
    },
  })

  return (
    response.data.data?.organizations ?? []
  ).filter(item =>
    item.type === 'school' &&
    item.status === 'active' &&
    item.parent_id === regionId &&
    (
      item.education_domain === 'k12' ||
      item.education_domain === 'vocational' ||
      item.education_domain === 'adult'
    )
  )
}

/**
 * createAdminUserWithSchool 创建用户。
 *
 * 系统管理员创建operator/viewer时school_id必填；
 * 创建平台admin时不提交school_id；
 * 学校管理员路径由后端强制本校。
 */
export async function createAdminUserWithSchool(
  request: CreateAdminUserWithSchoolRequest,
): Promise<CreatedAdminUser> {
  const response = await client.post<{
    code: number
    data: CreatedAdminUser
  }>('/admin/users', request)

  return response.data.data!
}
