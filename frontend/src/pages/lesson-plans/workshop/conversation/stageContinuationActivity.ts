/**
 * stageContinuationActivity.ts — 对话模式“阶段自动自然承接”轻量活动状态。
 *
 * 背景：
 *   - 教师主动聊天使用页面生成的 client_turn_id，并由 ConversationModePage 的 isBusy 管理；
 *   - 阶段推进后的自动自然承接是后台 Chat，不能和导入后评审等其它后台任务混为一谈；
 *   - 后端为阶段承接分配 stage_continuation_ 前缀的专用 turnID，本模块只识别这一类。
 *
 * 责任：
 *   1. 芯片发起阶段推进时立即进入短暂 pending，堵住 HTTP 返回到后台承接登记之间的点击窗口；
 *   2. SSE 观察到专用 turnID 后把 pending 升级为真实 active；
 *   3. 专用 message_done/error 或连接关闭时清理；
 *   4. 不修改 ConversationModePage，因此不会让 1558 行主页面继续膨胀。
 */

import { useSyncExternalStore } from 'react'

export const STAGE_CONTINUATION_TURN_ID_PREFIX =
  'stage_continuation_'

const EXPECTATION_TIMEOUT_MS = 5000

let active = false
let expectationTimer:
  ReturnType<typeof setTimeout> | null = null

const listeners = new Set<() => void>()

function emitChange() {
  listeners.forEach(listener => listener())
}

function setActive(next: boolean) {
  if (active === next) return
  active = next
  emitChange()
}

function clearExpectationTimer() {
  if (!expectationTimer) return
  clearTimeout(expectationTimer)
  expectationTimer = null
}

export function isStageContinuationTurnID(
  clientTurnID?: string,
): boolean {
  return Boolean(
    clientTurnID &&
      clientTurnID.startsWith(
        STAGE_CONTINUATION_TURN_ID_PREFIX,
      ),
  )
}

/**
 * beginStageContinuationExpectation 在用户点击“进入下一阶段”时立即锁住芯片。
 *
 * 正常情况下后端会在百毫秒级发出专用 thinking 事件并接管 active；
 * 若目标阶段没有自动承接或推进失败，5 秒兜底自动释放，避免形成死锁按钮。
 */
export function beginStageContinuationExpectation() {
  clearExpectationTimer()
  setActive(true)

  expectationTimer = setTimeout(() => {
    expectationTimer = null
    setActive(false)
  }, EXPECTATION_TIMEOUT_MS)
}

/** SSE 已确认阶段自动承接任务真正开始。 */
export function markStageContinuationStarted() {
  clearExpectationTimer()
  setActive(true)
}

/** 阶段自动承接已完成、失败或连接关闭。 */
export function finishStageContinuationActivity() {
  clearExpectationTimer()
  setActive(false)
}

function subscribe(listener: () => void) {
  listeners.add(listener)

  return () => {
    listeners.delete(listener)
  }
}

function getSnapshot() {
  return active
}

function getServerSnapshot() {
  return false
}

/** ChipRow 使用的只读 Hook。 */
export function useStageContinuationActivity() {
  return useSyncExternalStore(
    subscribe,
    getSnapshot,
    getServerSnapshot,
  )
}
