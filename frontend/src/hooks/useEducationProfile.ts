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

  const role =
    user?.role || ''

  const isRegionAdmin =
    role === 'region_admin'

  const isMixedManagementRole =
    role === 'admin' ||
    role === 'district_inspector'

  /**
   * 后端显式返回false时，任何角色都必须阻断教学业务。
   *
   * 字段暂未返回时仍允许继续检查具体教育域，但不允许通过
   * normalizeEducationDomain根据角色把空值静默推断成K12。
   */
  const backendReady =
    user?.education_domain_ready !== false

  const concreteDomainReady =
    isTeachingDomain(
      user?.education_domain,
    )

  const mixedManagementReady =
    user?.education_domain === 'mixed'

  /**
   * 教育域就绪规则：
   *
   * 1. region_admin：
   *    必须由后端显式返回ready=true，并具有唯一具体教学域。
   *
   * 2. admin、district_inspector：
   *    只允许明确的mixed管理教育域。
   *
   * 3. senior_operator、operator、viewer等教学身份：
   *    必须具有k12、vocational或adult具体教学域。
   *
   * 普通账号教育域为空时不再被前端静默识别成K12，
   * 从而避免页面允许提交、后端可信Actor却返回403的状态错位。
   */
  const ready = Boolean(
    user &&
    (
      isRegionAdmin
        ? user.education_domain_ready === true &&
          concreteDomainReady
        : isMixedManagementRole
          ? backendReady &&
            mixedManagementReady
          : backendReady &&
            concreteDomainReady
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
