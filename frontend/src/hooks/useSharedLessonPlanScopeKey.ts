/**
 * useSharedLessonPlanScopeKey — 共享教案前端作用域键
 *
 * 切换账号、教学组织或教育域后，旧列表和互动状态必须立即失效。
 * 本Hook仅生成前端生命周期键，不把教育域作为授权参数提交给后端。
 */

import { useMemo } from 'react'
import { useAuth } from '@/store/auth'
import { useEducationProfile } from '@/hooks/useEducationProfile'

export function useSharedLessonPlanScopeKey() {
  const { user } = useAuth()
  const {
    ready,
    domain,
    organizationId,
  } = useEducationProfile()

  const userId = user?.id || ''

  const key = useMemo(
    () => [
      userId || 'anonymous',
      ready ? 'ready' : 'blocked',
      ready ? domain : 'no-domain',
      ready ? organizationId || 'no-org' : 'no-org',
    ].join('|'),
    [
      userId,
      ready,
      domain,
      organizationId,
    ],
  )

  return {
    key,
    userId,
    ready,
  }
}
