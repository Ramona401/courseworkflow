/**
 * lessonConsensusCapsule.ts — “本课共识”胶囊的前端外部状态
 *
 * 设计目标：
 *   - SSE层收到context_capsule事件后，只更新一个轻量全局快照；
 *   - conversation冷启动响应可用数据库active胶囊校正浏览器缓存；
 *   - 专家模式和对话模式共享同一份状态，不各自复制一套逻辑；
 *   - 胶囊更新不改变主回复的thinking、streaming或终态状态；
 *   - 按教案ID写入sessionStorage，刷新或重连后可先显示最近视图；
 *   - 不保存课本全文、文件名清单、Token、提示词或内部回执。
 */

import type {
  LessonPlanContextCapsuleEventData,
} from '@/api/lesson-plans.types'

export type LessonConsensusCapsuleSource =
  | 'none'
  | 'cache'
  | 'server'
  | 'sse'

export interface LessonConsensusCapsuleSnapshot {
  activePlanId: string
  capsule: LessonPlanContextCapsuleEventData | null
  sequence: number
  source: LessonConsensusCapsuleSource
}

const EMPTY_SNAPSHOT: LessonConsensusCapsuleSnapshot = {
  activePlanId: '',
  capsule: null,
  sequence: 0,
  source: 'none',
}

let snapshot = EMPTY_SNAPSHOT
const listeners = new Set<() => void>()

function cacheKey(planId: string): string {
  return `lesson_consensus_capsule:${planId}`
}

function notify(): void {
  listeners.forEach(listener => listener())
}

function setSnapshot(
  next: LessonConsensusCapsuleSnapshot,
): void {
  snapshot = next
  notify()
}

function isUsableCapsule(
  capsule: LessonPlanContextCapsuleEventData | null,
): capsule is LessonPlanContextCapsuleEventData {
  return Boolean(
    capsule &&
    typeof capsule.version === 'number' &&
    capsule.version >= 1 &&
    capsule.status === 'active' &&
    capsule.display,
  )
}

function readCachedCapsule(
  planId: string,
): LessonPlanContextCapsuleEventData | null {
  if (
    typeof window === 'undefined' ||
    !planId
  ) {
    return null
  }

  try {
    const raw = window.sessionStorage.getItem(
      cacheKey(planId),
    )

    if (!raw) {
      return null
    }

    const parsed = JSON.parse(raw) as
      LessonPlanContextCapsuleEventData

    return isUsableCapsule(parsed)
      ? parsed
      : null
  } catch {
    return null
  }
}

function writeCachedCapsule(
  planId: string,
  capsule: LessonPlanContextCapsuleEventData,
): void {
  if (
    typeof window === 'undefined' ||
    !planId
  ) {
    return
  }

  try {
    window.sessionStorage.setItem(
      cacheKey(planId),
      JSON.stringify(capsule),
    )
  } catch {
    // 浏览器禁用存储时只保留当前内存状态，不影响主对话。
  }
}

function removeCachedCapsule(
  planId: string,
): void {
  if (
    typeof window === 'undefined' ||
    !planId
  ) {
    return
  }

  try {
    window.sessionStorage.removeItem(
      cacheKey(planId),
    )
  } catch {
    // 浏览器禁用存储时忽略。
  }
}

/** 订阅胶囊状态，供useSyncExternalStore使用。 */
export function subscribeLessonConsensusCapsule(
  listener: () => void,
): () => void {
  listeners.add(listener)

  return () => {
    listeners.delete(listener)
  }
}

/** 获取当前稳定快照。 */
export function getLessonConsensusCapsuleSnapshot():
  LessonConsensusCapsuleSnapshot {
  return snapshot
}

/**
 * 激活当前教案。
 *
 * SSE重连或模式切换时会重复调用；同一教案不会清空当前胶囊。
 * 切换到另一教案时先恢复该教案的会话级缓存，随后由conversation
 * 响应中的数据库active胶囊进行最终校正。
 */
export function activateLessonConsensusPlan(
  planId: string,
): void {
  const normalizedPlanId = planId.trim()

  if (!normalizedPlanId) {
    return
  }

  if (snapshot.activePlanId === normalizedPlanId) {
    return
  }

  const cached =
    readCachedCapsule(normalizedPlanId)

  setSnapshot({
    activePlanId: normalizedPlanId,
    capsule: cached,
    sequence: snapshot.sequence + 1,
    source: cached ? 'cache' : 'none',
  })
}

/**
 * 用conversation接口返回的数据库active胶囊校正冷启动状态。
 *
 * 安全处理并发：
 *   - SSE已经收到更高版本时，不被稍早返回的HTTP响应覆盖；
 *   - HTTP明确返回空胶囊时，只清除缓存来源，不清除刚收到的SSE版本；
 *   - 非当前教案只更新对应缓存，不覆盖正在显示的教案。
 */
export function hydrateLessonConsensusCapsule(
  planId: string,
  capsule: LessonPlanContextCapsuleEventData | null,
): void {
  const normalizedPlanId = planId.trim()
  if (!normalizedPlanId) {
    return
  }

  const usable = isUsableCapsule(capsule)

  if (!usable) {
    removeCachedCapsule(normalizedPlanId)

    if (
      snapshot.activePlanId === normalizedPlanId &&
      snapshot.source !== 'sse'
    ) {
      setSnapshot({
        activePlanId: normalizedPlanId,
        capsule: null,
        sequence: snapshot.sequence + 1,
        source: 'server',
      })
    }

    return
  }

  writeCachedCapsule(
    normalizedPlanId,
    capsule,
  )

  if (
    snapshot.activePlanId &&
    snapshot.activePlanId !== normalizedPlanId
  ) {
    return
  }

  if (
    snapshot.activePlanId === normalizedPlanId &&
    snapshot.capsule &&
    snapshot.capsule.version > capsule.version
  ) {
    return
  }

  setSnapshot({
    activePlanId: normalizedPlanId,
    capsule,
    sequence: snapshot.sequence + 1,
    source: 'server',
  })
}

/**
 * 取消当前教案激活状态。
 *
 * 缓存仍然保留，下一次恢复同一教案时可先显示最近快照。
 */
export function deactivateLessonConsensusPlan(
  planId: string,
): void {
  if (
    !planId ||
    snapshot.activePlanId !== planId
  ) {
    return
  }

  setSnapshot({
    activePlanId: '',
    capsule: null,
    sequence: snapshot.sequence + 1,
    source: 'none',
  })
}

/**
 * 发布旁路更新产生的新胶囊。
 *
 * 非当前教案的事件只写缓存，不覆盖正在显示的教案。
 * 同版本重复事件不会制造多余的呼吸动画。
 */
export function publishLessonConsensusCapsule(
  planId: string,
  capsule: LessonPlanContextCapsuleEventData,
): void {
  const normalizedPlanId = planId.trim()

  if (
    !normalizedPlanId ||
    !isUsableCapsule(capsule)
  ) {
    return
  }

  writeCachedCapsule(
    normalizedPlanId,
    capsule,
  )

  if (
    snapshot.activePlanId &&
    snapshot.activePlanId !== normalizedPlanId
  ) {
    return
  }

  if (
    snapshot.activePlanId === normalizedPlanId &&
    snapshot.capsule?.version === capsule.version
  ) {
    return
  }

  setSnapshot({
    activePlanId: normalizedPlanId,
    capsule,
    sequence: snapshot.sequence + 1,
    source: 'sse',
  })
}
