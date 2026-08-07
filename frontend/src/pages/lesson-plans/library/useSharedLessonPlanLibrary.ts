/**
 * useSharedLessonPlanLibrary — 共享教案库数据与互动状态
 *
 * 账号、组织或教育域变化时立即清空旧数据，并通过请求序号阻止旧域响应回写。
 * 关键词只过滤后端已经授权的候选集，不触发全平台搜索。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import {
  getMyGroups,
  type LessonPlan,
  type TeachingGroup,
} from '@/api/lesson-plans'
import {
  forkSharedLessonPlan,
  getSharedLessonPlans,
  type SharedLessonPlanListParams,
} from '@/api/shared-lesson-plans'
import {
  getInteractions,
  toggleInteraction,
  type InteractionCounts,
  type InteractionType,
} from '@/api/lesson-plan-interactions'
import { useSharedLessonPlanScopeKey } from '@/hooks/useSharedLessonPlanScopeKey'

export type LibraryScope = 'group' | 'school' | 'region'

export interface SharedLessonPlanLibraryFilters {
  scope: LibraryScope
  keyword: string
  subject: string
  grade: string
  qualityLevel: string
  structureType: string
}

interface ToastState {
  msg: string
  type: 'success' | 'error'
}

const emptyInteractions = (): InteractionCounts => ({
  like_count: 0,
  favorite_count: 0,
  is_liked: false,
  is_favorited: false,
})

export function useSharedLessonPlanLibrary(
  filters: SharedLessonPlanLibraryFilters,
) {
  const {
    key: scopeKey,
    userId,
    ready,
  } = useSharedLessonPlanScopeKey()

  const [serverPlans, setServerPlans] = useState<LessonPlan[]>([])
  const [interactionsMap, setInteractionsMap] = useState<
    Record<string, InteractionCounts>
  >({})
  const [myGroups, setMyGroups] = useState<TeachingGroup[]>([])
  const [groupsLoaded, setGroupsLoaded] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [forkingId, setForkingId] = useState<string | null>(null)
  const [interactionPending, setInteractionPending] = useState<
    Record<string, boolean>
  >({})
  const [toast, setToast] = useState<ToastState | null>(null)

  const scopeKeyRef = useRef(scopeKey)
  scopeKeyRef.current = scopeKey

  // Ref与展示state同步维护，防止快速双击发生在React完成重渲染之前。
  const interactionPendingRef = useRef<
    Record<string, boolean>
  >({})
  const forkingIdRef = useRef<string | null>(null)

  const listVersion = useRef(0)
  const groupVersion = useRef(0)
  const toastTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const showToast = useCallback((
    msg: string,
    type: ToastState['type'] = 'success',
  ) => {
    if (toastTimer.current) {
      clearTimeout(toastTimer.current)
    }
    setToast({ msg, type })
    toastTimer.current = setTimeout(
      () => setToast(null),
      3000,
    )
  }, [])

  useEffect(() => {
    listVersion.current += 1
    groupVersion.current += 1
    setServerPlans([])
    setInteractionsMap({})
    setMyGroups([])
    setGroupsLoaded(false)
    setLoading(true)
    setError(null)
    forkingIdRef.current = null
    interactionPendingRef.current = {}
    setForkingId(null)
    setInteractionPending({})
    setToast(null)

    if (toastTimer.current) {
      clearTimeout(toastTimer.current)
      toastTimer.current = null
    }
  }, [scopeKey])

  useEffect(() => {
    return () => {
      listVersion.current += 1
      groupVersion.current += 1
      forkingIdRef.current = null
      interactionPendingRef.current = {}
      if (toastTimer.current) {
        clearTimeout(toastTimer.current)
      }
    }
  }, [])

  useEffect(() => {
    const version = ++groupVersion.current

    if (!ready || !userId) {
      setMyGroups([])
      setGroupsLoaded(true)
      return
    }

    setGroupsLoaded(false)
    getMyGroups()
      .then(groups => {
        if (version !== groupVersion.current) return
        setMyGroups(groups || [])
        setGroupsLoaded(true)
      })
      .catch(() => {
        if (version !== groupVersion.current) return
        setMyGroups([])
        setGroupsLoaded(true)
      })
  }, [
    scopeKey,
    ready,
    userId,
  ])

  const loadPlans = useCallback(async () => {
    if (!ready || !userId) {
      setServerPlans([])
      setInteractionsMap({})
      setError(null)
      setLoading(false)
      return
    }

    if (
      filters.scope === 'group' &&
      !groupsLoaded
    ) {
      setLoading(true)
      return
    }

    if (
      filters.scope === 'group' &&
      myGroups.length === 0
    ) {
      listVersion.current += 1
      setServerPlans([])
      setInteractionsMap({})
      setError(null)
      setLoading(false)
      return
    }

    const version = ++listVersion.current
    setLoading(true)
    setError(null)

    try {
      const base: Omit<
        SharedLessonPlanListParams,
        'status'
      > = {
        limit: 100,
      }

      if (filters.subject !== '全部') {
        base.subject = filters.subject
      }
      if (filters.grade !== '全部') {
        base.grade = filters.grade
      }
      if (filters.qualityLevel !== '全部') {
        base.quality_level = Number(
          filters.qualityLevel,
        )
      }
      if (filters.structureType !== '全部') {
        base.structure_type = Number(
          filters.structureType,
        )
      }
      if (filters.scope === 'group') {
        base.group_id = myGroups[0].id
      }

      const [shared, approved] = await Promise.all([
        getSharedLessonPlans({
          ...base,
          status: 'published_shared',
        }),
        getSharedLessonPlans({
          ...base,
          status: 'approved',
        }),
      ])

      if (version !== listVersion.current) return

      const ids = new Set<string>()
      const merged: LessonPlan[] = []
      for (const plan of [
        ...(shared.lesson_plans || []),
        ...(approved.lesson_plans || []),
      ]) {
        if (ids.has(plan.id)) continue
        ids.add(plan.id)
        merged.push(plan)
      }

      merged.sort(
        (left, right) =>
          (right.ai_review_score ?? -1) -
          (left.ai_review_score ?? -1),
      )
      setServerPlans(merged)

      const entries = await Promise.all(
        merged.map(async plan => {
          try {
            return [
              plan.id,
              await getInteractions(plan.id),
            ] as const
          } catch {
            return [
              plan.id,
              emptyInteractions(),
            ] as const
          }
        }),
      )

      if (version !== listVersion.current) return
      setInteractionsMap(Object.fromEntries(entries))
    } catch (loadError) {
      if (version !== listVersion.current) return
      console.error('加载共享教案库失败:', loadError)
      setServerPlans([])
      setInteractionsMap({})
      setError('加载失败，请稍后重试')
    } finally {
      if (version === listVersion.current) {
        setLoading(false)
      }
    }
  }, [
    scopeKey,
    ready,
    userId,
    filters.scope,
    filters.subject,
    filters.grade,
    filters.qualityLevel,
    filters.structureType,
    groupsLoaded,
    myGroups,
  ])

  useEffect(() => {
    void loadPlans()
    return () => {
      listVersion.current += 1
    }
  }, [loadPlans])

  const plans = useMemo(() => {
    const keyword = filters.keyword.trim().toLowerCase()
    if (!keyword) return serverPlans

    return serverPlans.filter(plan =>
      plan.title.toLowerCase().includes(keyword) ||
      plan.topic.toLowerCase().includes(keyword) ||
      plan.subject.toLowerCase().includes(keyword),
    )
  }, [
    serverPlans,
    filters.keyword,
  ])

  const handleToggle = useCallback(async (
    planId: string,
    type: InteractionType,
  ) => {
    const pendingKey = `${planId}:${type}`
    const actionScopeKey = scopeKeyRef.current

    if (interactionPendingRef.current[pendingKey]) {
      return
    }

    const nextPending = {
      ...interactionPendingRef.current,
      [pendingKey]: true,
    }
    interactionPendingRef.current = nextPending
    setInteractionPending(nextPending)

    try {
      const response = await toggleInteraction(planId, type)

      // 用户、组织或教育域已变化时，丢弃旧作用域的迟到响应。
      if (actionScopeKey !== scopeKeyRef.current) {
        return
      }

      setInteractionsMap(previous => ({
        ...previous,
        [planId]: {
          ...(previous[planId] || emptyInteractions()),
          ...(type === 'like'
            ? {
              like_count: response.new_count,
              is_liked: response.active,
            }
            : {
              favorite_count: response.new_count,
              is_favorited: response.active,
            }),
        },
      }))

      showToast(
        response.active
          ? type === 'like'
            ? '已点赞 👍'
            : '已收藏 📌'
          : type === 'like'
            ? '已取消点赞'
            : '已取消收藏',
      )
    } catch {
      if (actionScopeKey !== scopeKeyRef.current) {
        return
      }

      showToast('操作失败，已刷新当前教案库', 'error')
      await loadPlans()
    } finally {
      if (actionScopeKey === scopeKeyRef.current) {
        const next = {
          ...interactionPendingRef.current,
        }
        delete next[pendingKey]
        interactionPendingRef.current = next
        setInteractionPending(next)
      }
    }
  }, [
    loadPlans,
    showToast,
  ])

  const handleFork = useCallback(async (
    plan: LessonPlan,
  ) => {
    if (forkingIdRef.current) return

    const actionScopeKey = scopeKeyRef.current
    forkingIdRef.current = plan.id
    setForkingId(plan.id)

    try {
      const forked = await forkSharedLessonPlan(plan.id)

      if (actionScopeKey !== scopeKeyRef.current) {
        return
      }

      showToast(`已Fork到我的草稿：${forked.title} ✓`)
    } catch (forkError) {
      if (actionScopeKey !== scopeKeyRef.current) {
        return
      }

      console.error('Fork共享教案失败:', forkError)
      showToast('Fork失败，已刷新当前教案库', 'error')
      await loadPlans()
    } finally {
      if (actionScopeKey === scopeKeyRef.current) {
        forkingIdRef.current = null
        setForkingId(null)
      }
    }
  }, [
    loadPlans,
    showToast,
  ])

  return {
    userId,
    plans,
    total: plans.length,
    loading,
    error,
    myGroups,
    interactionsMap,
    interactionPending,
    forkingId,
    toast,
    reload: loadPlans,
    toggleInteraction: handleToggle,
    forkPlan: handleFork,
  }
}
