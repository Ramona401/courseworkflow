/**
 * organization-education-domains.ts — 组织教育域只读API
 *
 * 组织教育域在组织创建时确定：
 *   - 区域固定为mixed；
 *   - 学校必须选择k12、vocational或adult；
 *   - 创建成功后普通业务不能修改。
 *
 * 前端只保留GET列表，不再导出任何更新函数。
 */

import client from './client'
import type { ApiResponse } from './client'
import type { EducationDomain } from '@/education-domain/types'

export interface OrganizationEducationDomainItem {
  id: string
  name: string
  type: 'region' | 'school'

  parent_id: string | null
  parent_name: string

  education_domain: EducationDomain
  status: string

  group_count: number
  member_count: number
}

export interface OrganizationEducationDomainListResult {
  organizations: OrganizationEducationDomainItem[]
  total: number
}

/**
 * 获取组织教育域只读列表。
 */
export async function getOrganizationEducationDomains():
Promise<OrganizationEducationDomainListResult> {
  const res = await client.get<
    ApiResponse<OrganizationEducationDomainListResult>
  >('/admin/organization-education-domains')

  return res.data.data || {
    organizations: [],
    total: 0,
  }
}
