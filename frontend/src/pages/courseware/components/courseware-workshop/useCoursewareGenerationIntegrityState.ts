/**
 * useCoursewareGenerationIntegrityState.ts — R-04 普通批量生成完整性状态控制器
 *
 * 负责读取后端 assembly-state 中的生成完整性事实，并为普通批量生成提供
 * “只补生成缺失页”的安全启动与后台轮询。
 *
 * 事实与并发边界：
 *   - complete、各类页数和问题清单完全使用后端返回值，不在浏览器重算；
 *   - 补生成前重新读取最新数据库状态，服务端随后再次冻结当前方案快照；
 *   - POST 成功后等待观察到新的数据库版本，避免旧终态冒充新任务结果；
 *   - 只追踪 run_kind=batch 的活动运行，不把自动装配生命周期混进普通生成。
 */

import { useCallback, useEffect, useRef, useState } from 'react'

import { generateCWPages } from '@/api/coursewares'
import { getCoursewareAssemblyState } from '@/api/coursewareAssembly'
import type {
  CoursewareAssemblyState,
  CoursewareGenerationIntegrity,
} from '@/api/coursewareAssembly'

interface Options {
  coursewareId?: string
  enabled?: boolean
  onSettled?: () => void
}

export interface CoursewareGenerationIntegrityController {
  state: CoursewareAssemblyState | null
  integrity: CoursewareGenerationIntegrity | null
  loading: boolean
  retrying: boolean
  error: string
  refresh: () => Promise<CoursewareAssemblyState | null>
  retryMissingPages: () => Promise<void>
}

interface AwaitingVersion {
  baselineVersion: number
  deadline: number
}

export function useCoursewareGenerationIntegrityState({
  coursewareId,
  enabled = true,
  onSettled,
}: Options): CoursewareGenerationIntegrityController {
  const mountedRef = useRef(true)
  const onSettledRef = useRef(onSettled)
  const previousBatchActiveRef = useRef(false)
  const awaitingVersionRef = useRef<AwaitingVersion | null>(null)

  const [state, setState] = useState<CoursewareAssemblyState | null>(null)
  const [loading, setLoading] = useState(Boolean(enabled && coursewareId))
  const [retrying, setRetrying] = useState(false)
  const [watchingSubmittedRun, setWatchingSubmittedRun] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    onSettledRef.current = onSettled
  }, [onSettled])

  useEffect(() => {
    mountedRef.current = true

    return () => {
      mountedRef.current = false
    }
  }, [])

  const refresh = useCallback(async () => {
    if (!enabled || !coursewareId) {
      if (mountedRef.current) {
        setState(null)
        setLoading(false)
        setError('')
      }

      return null
    }

    try {
      const next = await getCoursewareAssemblyState(coursewareId)

      if (!mountedRef.current) {
        return null
      }

      let nextError = ''
      const awaiting = awaitingVersionRef.current

      if (awaiting) {
        if (next.assembly_version > awaiting.baselineVersion) {
          awaitingVersionRef.current = null
          setWatchingSubmittedRun(false)

          if (next.run_kind !== 'batch') {
            nextError = '后台运行类型已变化，请刷新后确认当前任务。'
          } else if (!next.is_active) {
            // 极快任务可能在首次看到新版本时已经结束；此时没有经历active=true，
            // 仍需通知父级刷新页面，避免工作台停留在补生前的页面数组。
            onSettledRef.current?.()
          }
        } else if (Date.now() >= awaiting.deadline) {
          awaitingVersionRef.current = null
          setWatchingSubmittedRun(false)
          nextError = '补生成任务已提交，但暂未观察到新的后台运行，请稍后重新同步。'
        }
      }

      setState(next)
      setError(nextError)
      return next
    } catch {
      if (mountedRef.current) {
        setError('暂时无法读取页面完整性状态，请稍后刷新重试。')
      }

      return null
    } finally {
      if (mountedRef.current) {
        setLoading(false)
      }
    }
  }, [coursewareId, enabled])

  useEffect(() => {
    previousBatchActiveRef.current = false
    awaitingVersionRef.current = null
    setWatchingSubmittedRun(false)

    if (!enabled || !coursewareId) {
      setState(null)
      setLoading(false)
      setError('')
      return
    }

    setLoading(true)
    setError('')
    void refresh()
  }, [coursewareId, enabled, refresh])

  const batchActive = Boolean(state?.is_active && state.run_kind === 'batch')

  useEffect(() => {
    if (!batchActive && !watchingSubmittedRun) {
      return
    }

    const timer = window.setInterval(() => {
      void refresh()
    }, 2500)

    return () => {
      window.clearInterval(timer)
    }
  }, [batchActive, refresh, watchingSubmittedRun])

  useEffect(() => {
    if (previousBatchActiveRef.current && !batchActive) {
      onSettledRef.current?.()
    }

    previousBatchActiveRef.current = batchActive
  }, [batchActive])

  const retryMissingPages = useCallback(async () => {
    if (!enabled || !coursewareId) {
      throw new Error('当前课件不能启动补生成。')
    }

    // PRD要求补生前重新校验当前方案与页面状态。
    const current = await refresh()

    if (!current) {
      throw new Error('暂时无法确认后台生成状态，请稍后重试。')
    }

    if (current.is_active) {
      throw new Error('后台页面生成仍在进行，请等待当前任务结束。')
    }

    if (current.run_kind !== 'batch' || !current.integrity) {
      throw new Error('当前记录不是普通批量生成，请从对应装配入口继续。')
    }

    if (current.integrity.complete) {
      return
    }

    const unresolved =
      current.integrity.failed_count +
      current.integrity.cancelled_count +
      current.integrity.missing_count

    if (unresolved <= 0) {
      throw new Error('当前没有可补生成的失败、取消或缺失页面。')
    }

    setRetrying(true)
    setError('')

    awaitingVersionRef.current = {
      baselineVersion: current.assembly_version,
      deadline: Date.now() + 30000,
    }
    setWatchingSubmittedRun(true)

    try {
      await generateCWPages(coursewareId)

      // generate-pages 是后台入口。短暂等待后主动同步一次；
      // 若run仍未可见，2500ms轮询继续观察数据库版本推进。
      await new Promise<void>(resolve => {
        window.setTimeout(resolve, 300)
      })

      await refresh()
    } catch {
      awaitingVersionRef.current = null

      if (mountedRef.current) {
        setWatchingSubmittedRun(false)
        setError('补生成任务提交失败，请稍后重试。')
      }

      throw new Error('补生成任务提交失败，请稍后重试。')
    } finally {
      if (mountedRef.current) {
        setRetrying(false)
      }
    }
  }, [coursewareId, enabled, refresh])

  return {
    state,
    integrity: state?.integrity ?? null,
    loading,
    retrying,
    error,
    refresh,
    retryMissingPages,
  }
}
