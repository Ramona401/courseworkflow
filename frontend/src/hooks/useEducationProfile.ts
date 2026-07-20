/**
 * useEducationProfile — 当前用户教育域的统一前端入口
 *
 * 所有业务页面和业务Hook应通过本Hook读取：
 *   domain          当前展示教育域
 *   organizationId 当前具体教学组织
 *   profile         页面术语与能力开关
 *   ready           教育域是否可用于教学业务
 *   error           教育域异常提示
 *
 * 安全规则：
 *   - region_admin必须由后端显式返回education_domain_ready=true；
 *   - region_admin的education_domain还必须是k12/vocational/adult之一；
 *   - 异常状态不调用normalizeEducationDomain处理空值，避免回退K12或mixed；
 *   - 异常状态下domain仅使用mixed画像完成错误页展示，不能作为业务授权依据；
 *   - 教学路由由EducationDomainGuard统一阻断；
 *   - 课程请求等底层Hook仍须读取ready进行第二层fail-closed保护。
 */

import { useAuth } from '@/store/auth'
import {
  fallbackEducationProfile,
  normalizeEducationDomain,
} from '@/education-domain/types'
import type {
  EducationDomain,
  EducationProfile,
} from '@/education-domain/types'

const DEFAULT_NOT_READY_MESSAGE =
  '教育域尚未正确配置，请联系管理员。'

function isTeachingDomain(
  domain: string | undefined,
): domain is Exclude<EducationDomain, 'mixed'> {
  return domain === 'k12' ||
    domain === 'vocational' ||
    domain === 'adult'
}

export interface EducationProfileContext {
  /**
   * 正常状态下为真实教育域。
   *
   * ready=false时仅为错误页所需的mixed展示画像，
   * 业务代码必须先判断ready，禁止单独把domain用于授权。
   */
  domain: EducationDomain

  organizationId: string
  profile: EducationProfile

  /** 是否可以进入教学业务。 */
  ready: boolean

  /** 教育域异常时的统一提示。 */
  error: string

  isK12: boolean
  isVocational: boolean
  isAdult: boolean
  isMixed: boolean
}

export function useEducationProfile(): EducationProfileContext {
  const { user } = useAuth()

  const isRegionAdmin = user?.role === 'region_admin'

  // 区域管理员采用严格模式：
  //   1. 后端必须显式返回ready=true；
  //   2. 教育域必须是三个具体教学域之一。
  //
  // 其它角色保持兼容窗口：
  //   - 后端显式返回false时阻断；
  //   - 字段尚未下发时暂按可用处理，避免原子部署期间误伤存量角色。
  const ready = Boolean(
    user &&
    (
      isRegionAdmin
        ? user.education_domain_ready === true &&
          isTeachingDomain(user.education_domain)
        : user.education_domain_ready !== false
    ),
  )

  // 异常状态不读取空教育域，也不调用角色推断。
  // mixed只作为错误页的中性展示画像，绝不代表业务已经获得跨域授权。
  const domain: EducationDomain = ready
    ? normalizeEducationDomain(
      user?.education_domain,
      user?.role,
    )
    : 'mixed'

  const organizationId = ready
    ? user?.education_org_id || ''
    : ''

  const backendProfile = ready
    ? user?.education_profile
    : undefined

  const profile =
    backendProfile && backendProfile.code === domain
      ? backendProfile
      : fallbackEducationProfile(domain)

  const error = ready
    ? ''
    : user?.education_domain_error?.trim() ||
      DEFAULT_NOT_READY_MESSAGE

  return {
    domain,
    organizationId,
    profile,
    ready,
    error,

    isK12: ready && domain === 'k12',
    isVocational: ready && domain === 'vocational',
    isAdult: ready && domain === 'adult',
    isMixed: ready && domain === 'mixed',
  }
}
