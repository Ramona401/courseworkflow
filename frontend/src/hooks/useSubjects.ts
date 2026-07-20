/**
 * useSubjects — 按教育域与教学组织分桶的课程目录Hook
 *
 * 缓存键：
 *   education_domain + education_org_id
 *
 * 规则：
 *   - K12、职业教育、成人教育和mixed互不共享缓存；
 *   - 同一浏览器切换账号时，useAuthProvider会清空全部缓存；
 *   - K12或mixed请求失败时可回退内置K12学科，保持历史可用性；
 *   - vocational/adult失败或空目录时返回空列表，绝不回退K12；
 *   - 后端实际返回的教育域与请求缓存键不一致时，不把结果写入错误缓存桶；
 *   - education_domain_ready=false时直接返回空数组且不发课程请求。
 *
 * 最后一条是路由守卫之外的第二层fail-closed保护：
 * 即使未来某个组件在管理页面或独立弹窗中误挂载本Hook，
 * 异常区域管理员也不会请求或显示任何默认K12课程。
 */

import {
  useEffect,
  useMemo,
  useState,
} from 'react'
import { getSubjectCatalog } from '@/api/subjects'
import {
  DEFAULT_SUBJECTS,
  withAllOption,
  withAnyOption,
} from '@/constants/subjects'
import type { EducationDomain } from '@/education-domain/types'
import {
  normalizeEducationDomain,
} from '@/education-domain/types'
import { useEducationProfile } from '@/hooks/useEducationProfile'

const cachedNames = new Map<string, string[]>()
const inflight = new Map<string, Promise<string[]>>()
const subscribers = new Set<() => void>()

function notifyAll() {
  subscribers.forEach(listener => {
    try {
      listener()
    } catch {
      // 单个订阅者异常不影响其它组件。
    }
  })
}

function makeCacheKey(
  domain: EducationDomain,
  organizationId: string,
): string {
  return `${domain}::${organizationId || '-'}`
}

function fallbackForDomain(
  domain: EducationDomain,
): string[] {
  if (domain === 'k12' || domain === 'mixed') {
    return [...DEFAULT_SUBJECTS]
  }

  // 职教和成人教育不能出现K12兜底课程。
  return []
}

async function loadSubjects(
  requestedKey: string,
  requestedDomain: EducationDomain,
): Promise<string[]> {
  const cached = cachedNames.get(requestedKey)
  if (cached) return cached

  const activeRequest = inflight.get(requestedKey)
  if (activeRequest) return activeRequest

  const request = (async () => {
    try {
      const result = await getSubjectCatalog()

      const responseDomain = normalizeEducationDomain(
        result.education_domain,
      )

      const responseKey = makeCacheKey(
        responseDomain,
        result.education_org_id || '',
      )

      const names = (result.subjects || [])
        .map(item => item.name)
        .filter(Boolean)

      const normalizedNames =
        names.length > 0
          ? names
          : fallbackForDomain(responseDomain)

      cachedNames.set(
        responseKey,
        normalizedNames,
      )

      // 账号切换或请求竞态时，不把其它教育域结果放进当前桶。
      if (responseKey !== requestedKey) {
        return fallbackForDomain(requestedDomain)
      }

      return normalizedNames
    } catch {
      // 请求失败不写缓存，下次挂载或刷新时继续尝试。
      return fallbackForDomain(requestedDomain)
    } finally {
      inflight.delete(requestedKey)
    }
  })()

  inflight.set(requestedKey, request)
  return request
}

/** 清空全部教育域课程缓存，不主动发请求。 */
export function resetSubjectCache(): void {
  cachedNames.clear()
  inflight.clear()
  notifyAll()
}

/**
 * 后台修改统一课程定义或教育域后清缓存。
 *
 * 当前挂载的Hook收到通知后，会按各自教育域重新请求。
 */
export async function refreshSubjects(): Promise<void> {
  resetSubjectCache()
}

/** useSubjects选项。 */
export interface UseSubjectsOptions {
  withAll?: boolean
  withAny?: boolean
}

export interface UseSubjectsResult {
  subjects: string[]
  loading: boolean
  empty: boolean
}

/**
 * 返回当前用户教育域课程目录。
 *
 * vocational/adult可能合法返回空列表，调用页面应根据empty展示
 * “当前学校尚未配置课程”，而不是自行使用K12常量。
 *
 * 教育域未就绪时：
 *   - subjects恒为空数组；
 *   - loading恒为false；
 *   - 不读取缓存；
 *   - 不发起getSubjectCatalog请求；
 *   - withAll和withAny也不添加虚假的“全部/不限”选项。
 */
export function useSubjects(
  options: UseSubjectsOptions = {},
): UseSubjectsResult {
  const {
    withAll = false,
    withAny = false,
  } = options

  const {
    domain,
    organizationId,
    ready,
  } = useEducationProfile()

  const cacheKey = useMemo(
    () => makeCacheKey(domain, organizationId),
    [domain, organizationId],
  )

  const fallback = useMemo(
    () => ready
      ? fallbackForDomain(domain)
      : [],
    [domain, ready],
  )

  const [names, setNames] = useState<string[]>(
    () => ready
      ? cachedNames.get(cacheKey) || fallback
      : [],
  )

  const [loading, setLoading] = useState<boolean>(
    () => ready && !cachedNames.has(cacheKey),
  )

  const [revision, setRevision] = useState(0)

  useEffect(() => {
    const listener = () => {
      setRevision(value => value + 1)
    }

    subscribers.add(listener)

    return () => {
      subscribers.delete(listener)
    }
  }, [])

  useEffect(() => {
    let mounted = true

    // 教育域异常时立即清空已有页面状态并停止，
    // 不能读取mixed/K12缓存，也不能调用课程接口。
    if (!ready) {
      setNames([])
      setLoading(false)

      return () => {
        mounted = false
      }
    }

    const cached = cachedNames.get(cacheKey)
    if (cached) {
      setNames(cached)
      setLoading(false)

      return () => {
        mounted = false
      }
    }

    setNames(fallback)
    setLoading(true)

    loadSubjects(cacheKey, domain)
      .then(result => {
        if (!mounted) return
        setNames(result)
        setLoading(false)
      })

    return () => {
      mounted = false
    }
  }, [
    cacheKey,
    domain,
    fallback,
    ready,
    revision,
  ])

  // 异常状态禁止生成“全部”或“不限”伪选项。
  let result = ready
    ? names
    : []

  if (ready && withAll) {
    result = withAllOption(names)
  } else if (ready && withAny) {
    result = withAnyOption(names)
  }

  return {
    subjects: result,
    loading: ready ? loading : false,
    empty: !ready || names.length === 0,
  }
}
