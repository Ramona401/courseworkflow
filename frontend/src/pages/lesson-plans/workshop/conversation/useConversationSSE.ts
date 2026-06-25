/**
 * useConversationSSE.ts — 对话模式 SSE 连接管理 Hook
 * （迭代3.5 A2-3 拆分 + B-2 + 路线乙第2步 + 子轮二 B2轮次过滤/三层超时）
 *
 * 职责边界：
 *   - 持有 SSE 连接句柄、当前教案ID、连接状态；
 *   - connectSSE 建立教案会话级单一长连接，各事件回调统一复位"生成中"状态、回写页面；
 *   - closeSSE / manualReconnect / 卸载自动关闭。
 *
 * 子轮二（B2 轮次序号 + 三层超时兜底）——本次新增的核心能力：
 *
 *   【轮次过滤】解决"超时作废后，迟到的旧轮回复污染新轮上下文"。
 *     页面持有 currentTurnRef（每发起一轮自增的字符串 id），经 params 传入本 Hook。
 *     chat 主轮次相关事件（thinking/chunk/message_done/suggested_actions/error/retry_notice）
 *     都带后端回传的 clientTurnId。过滤规则（isStaleTurn）：
 *       - clientTurnId 为空 → 放行（系统旁路推送：开场白/评审/手动按钮等，不归属任何轮次）；
 *       - 非空且 === currentTurnRef → 处理（本轮）；
 *       - 非空且 !== currentTurnRef → 丢弃（过期轮次的迟到回复）。
 *
 *   【三层超时】解决"AI 无声挂起/重试慢，老师对着转圈干等只能刷新"（依据业界最佳实践：
 *     等待期必须有可见反馈，10s 是注意力外限；90s 看门狗中止无产出的挂起）。
 *       第一层·软提示(8s)：发起一轮后 8s 没等到首个 chunk，显示"AI 正在认真思考，请稍候…"；
 *       第二层·重试可见性：收到后端 retry_notice 事件（首轮空流自动重试时广播）→ 显示"正在重试…"；
 *       第三层·看门狗(90s)：90s 内无任何本轮 SSE 事件 → 判定挂起，复位 thinking + 人话 +
 *         让重试可用 + 推进 turnID 作废本轮（onWatchdogTimeout 由页面实现）。
 *     计时器在"收到本轮任意事件"时重置（resetActivityTimer），message_done/error 到达即清除。
 *     计时器启停由页面在发起/结束一轮时调用 startTurnTimers/clearTurnTimers 驱动。
 *
 * 状态归属纪律：会话业务状态仍由页面持有，本 Hook 经 params 的 setter / 回调回写。
 */
import { useState, useRef, useEffect, useCallback } from 'react'
import {
  createLessonPlanSSE, getConversation, getLessonPlan,
  type ConversationMessage, type SSEConnectionState, type SSEConnection,
  type SuggestedAction,
} from '@/api/lesson-plans'
import type { StreamingState } from '../components/workshopConstants'
import type { ChipDef } from './conversationScript'
import { suggestedActionsToChips } from './chipActions'

/** 第一层·软提示触发阈值（毫秒）：发起一轮后多久没等到首 chunk 就安抚一句 */
const SLOW_HINT_MS = 8000
/** 第三层·看门狗阈值（毫秒）：多久无任何本轮事件就判定挂起兜底 */
const WATCHDOG_MS = 90000

/** Hook 入参：页面注入的状态 setter、轮次 ref 与回调集合 */
export interface UseConversationSSEParams {
  token: string | null
  setIsThinking: (v: boolean) => void
  setStreaming: React.Dispatch<React.SetStateAction<StreamingState | null>>
  setFullGenerating: (v: boolean) => void
  setMessages: React.Dispatch<React.SetStateAction<ConversationMessage[]>>
  setPlanContent: React.Dispatch<React.SetStateAction<string>>
  setDynamicChips: React.Dispatch<React.SetStateAction<ChipDef[]>>
  showToast: (msg: string) => void
  refreshStages: (planId: string) => Promise<void>
  /** 路线乙：软失败（message_done 带 soft_retry 标记，或 onError）→ 页面 recordFailure */
  onSoftFailure: () => void
  /** 路线乙：正常回复（message_done 不带 soft_retry）→ 页面 recordSuccess */
  onNormalReply: () => void
  /** 助手轻量选择入口·可见性补丁：本轮 message_done 携带的匹配助手名→页面更新顶栏指示器(空=纯骨架,页面回退"自动匹配") */
  onAssistantLabel?: (label: string) => void

  /* ===== 子轮二新增 ===== */
  /** 当前轮次序号 ref（页面持有，发起每轮时自增）。事件 clientTurnId 与之比对做过滤。 */
  currentTurnRef: React.MutableRefObject<string>
  /** 第一层软提示：8s 仍无首 chunk 时调用，页面显示安抚文案 */
  onSlowHint: () => void
  /** 第二层重试可见性：收到后端 retry_notice 时调用，页面显示"正在重试…" */
  onRetryNotice: (content: string) => void
  /** 第三层看门狗：90s 无本轮事件时调用，页面复位+人话+让重试可用+作废本轮 turnID */
  onWatchdogTimeout: () => void
}

/** Hook 返回值 */
export interface UseConversationSSEResult {
  sseState: SSEConnectionState
  connectSSE: (planId: string) => void
  closeSSE: () => void
  manualReconnect: (planId: string) => void
  /** 页面发起一轮 chat 时调用：启动软提示(8s)+看门狗(90s)计时 */
  startTurnTimers: () => void
  /** 页面结束/作废一轮时调用：清除所有计时器 */
  clearTurnTimers: () => void
}

/**
 * 对话模式 SSE 连接管理 Hook
 */
export function useConversationSSE(params: UseConversationSSEParams): UseConversationSSEResult {
  const [sseState, setSseState] = useState<SSEConnectionState>('connected')
  const sseRef = useRef<SSEConnection | null>(null)
  const planIdRef = useRef<string | null>(null)

  const paramsRef = useRef(params)
  paramsRef.current = params

  // 三层超时计时器句柄
  const slowHintTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const watchdogTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  /** 清除所有计时器（本轮结束/作废/卸载） */
  const clearTurnTimers = useCallback(() => {
    if (slowHintTimerRef.current) { clearTimeout(slowHintTimerRef.current); slowHintTimerRef.current = null }
    if (watchdogTimerRef.current) { clearTimeout(watchdogTimerRef.current); watchdogTimerRef.current = null }
  }, [])

  /** 启动本轮计时：软提示(8s) + 看门狗(90s) */
  const startTurnTimers = useCallback(() => {
    clearTurnTimers()
    slowHintTimerRef.current = setTimeout(() => {
      paramsRef.current.onSlowHint()
    }, SLOW_HINT_MS)
    watchdogTimerRef.current = setTimeout(() => {
      // 看门狗触发：清掉计时器并交给页面做兜底（复位+人话+让重试可用+作废 turnID）
      clearTurnTimers()
      paramsRef.current.onWatchdogTimeout()
    }, WATCHDOG_MS)
  }, [clearTurnTimers])

  /** 收到本轮任意事件 → 重置看门狗（活着就续命）。软提示只在"首个有效进展"时撤掉。 */
  const resetWatchdog = useCallback(() => {
    if (watchdogTimerRef.current) {
      clearTimeout(watchdogTimerRef.current)
      watchdogTimerRef.current = setTimeout(() => {
        clearTurnTimers()
        paramsRef.current.onWatchdogTimeout()
      }, WATCHDOG_MS)
    }
  }, [clearTurnTimers])

  /** 撤掉软提示计时器（首个 chunk 到达，已经开始出字，不需要再安抚） */
  const clearSlowHint = useCallback(() => {
    if (slowHintTimerRef.current) { clearTimeout(slowHintTimerRef.current); slowHintTimerRef.current = null }
  }, [])

  /**
   * 轮次过滤：判断一个事件是否"过期轮次的迟到回复"，是则应丢弃。
   *   - clientTurnId 为空 → 不过滤（系统旁路）；
   *   - 非空且 !== 当前轮 → 过期，丢弃；
   *   - 非空且 === 当前轮 → 本轮，处理。
   */
  const isStaleTurn = (clientTurnId?: string): boolean => {
    if (!clientTurnId) return false
    return clientTurnId !== paramsRef.current.currentTurnRef.current
  }

  // 组件卸载：关连接 + 清计时器
  useEffect(() => {
    return () => {
      sseRef.current?.close()
      if (slowHintTimerRef.current) clearTimeout(slowHintTimerRef.current)
      if (watchdogTimerRef.current) clearTimeout(watchdogTimerRef.current)
    }
  }, [])

  /** 建立教案会话级 SSE 长连接 */
  const connectSSE = useCallback((planId: string) => {
    const p = paramsRef.current
    if (!p.token) return
    sseRef.current?.close()
    planIdRef.current = planId
    sseRef.current = createLessonPlanSSE(planId, p.token, {
      onThinking: (clientTurnId?: string) => {
        if (isStaleTurn(clientTurnId)) return
        resetWatchdog()
        paramsRef.current.setIsThinking(true)
        paramsRef.current.setStreaming(null)
      },
      onChunk: (chunk: string, clientTurnId?: string) => {
        if (isStaleTurn(clientTurnId)) return
        // 首个 chunk 到达：撤软提示、续看门狗（开始出字了）
        clearSlowHint()
        resetWatchdog()
        paramsRef.current.setIsThinking(false)
        paramsRef.current.setStreaming(prev =>
          prev
            ? { ...prev, content: prev.content + chunk }
            : { id: `stream_${Date.now()}`, content: chunk }
        )
      },
      onMessageDone: (msg: ConversationMessage, clientTurnId?: string, assistantLabel?: string) => {
        if (isStaleTurn(clientTurnId)) return
        // 本轮收尾：清掉所有计时器
        clearTurnTimers()
        const cur = paramsRef.current
        cur.setIsThinking(false)
        cur.setStreaming(null)
        cur.setFullGenerating(false)
        cur.setDynamicChips([])
        cur.setMessages(prev => [...prev, msg])
        // 可见性补丁：把本轮实际匹配的助手名转发给页面更新顶栏(assistantLabel 可能为 undefined/空=纯骨架)
        cur.onAssistantLabel?.(assistantLabel || '')
        const isSoftRetry = !!(msg.metadata && (msg.metadata as Record<string, unknown>).soft_retry === true)
        if (isSoftRetry) {
          cur.onSoftFailure()
        } else {
          cur.onNormalReply()
        }
      },
      onContentUpdate: (content: string) => {
        // content_update 不带 turnID（write/revise 落库旁路），不参与过滤
        const cur = paramsRef.current
        cur.setPlanContent(prev => {
          const wasEmpty = !prev || prev.trim().length === 0
          if (wasEmpty && content && content.trim().length > 0) {
            cur.showToast('✅ 教案正文已生成！右侧画布可查看全文')
            cur.setFullGenerating(false)
          }
          return content
        })
      },
      onReviewDone: () => { /* 评审报告以对话消息形式呈现，Phase A 不单独渲染面板 */ },
      onStageStarted: () => { paramsRef.current.refreshStages(planId) },
      onStageComplete: () => { paramsRef.current.refreshStages(planId) },
      onStageOutput: () => { paramsRef.current.refreshStages(planId) },
      onSuggestedActions: (actions: SuggestedAction[], clientTurnId?: string) => {
        if (isStaleTurn(clientTurnId)) return
        const chips = suggestedActionsToChips(actions)
        if (chips.length > 0) paramsRef.current.setDynamicChips(chips)
      },
      onRetryNotice: (content: string, clientTurnId?: string) => {
        // 第二层·重试可见性：本轮首轮空流、后端自动重试时到达。续看门狗、显示提示。
        if (isStaleTurn(clientTurnId)) return
        resetWatchdog()
        paramsRef.current.onRetryNotice(content)
      },
      onError: (err: string, clientTurnId?: string) => {
        if (isStaleTurn(clientTurnId)) return
        // 本轮出错收尾：清计时器
        clearTurnTimers()
        const cur = paramsRef.current
        console.error('[对话模式] 生成出错:', err)
        cur.setIsThinking(false)
        cur.setStreaming(null)
        cur.setFullGenerating(false)
        cur.setDynamicChips([])
        cur.setMessages(prev => [...prev, {
          id: `err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
          content: `我这边刚才没接上话，可能是网络打了个盹。你之前的内容都还在、不会丢——把刚才那句再发一次、或点下面的「重新回答」就好。`,
          created_at: new Date().toISOString(),
        }])
        cur.onSoftFailure()
      },
      onDone: () => {
        clearTurnTimers()
        const cur = paramsRef.current
        cur.setIsThinking(false)
        cur.setStreaming(null)
        cur.setFullGenerating(false)
      },
      onConnectionStateChange: (state: SSEConnectionState) => { setSseState(state) },
      onReconnected: async () => {
        const pid = planIdRef.current
        if (!pid) return
        const cur = paramsRef.current
        cur.setDynamicChips([])
        try {
          const convData = await getConversation(pid)
          const serverMsgs = (convData.messages || []).filter(
            (m: ConversationMessage) => m.role === 'user' || m.role === 'assistant' || m.role === 'system'
          )
          cur.setMessages(prev => (serverMsgs.length > prev.length ? serverMsgs : prev))
          const planData = await getLessonPlan(pid)
          if (planData.content_markdown) cur.setPlanContent(planData.content_markdown)
          if (planData.current_stage && planData.stage_config) await cur.refreshStages(pid)
          cur.setIsThinking(false)
          cur.setStreaming(null)
        } catch (err) {
          console.error('[对话模式] 重连补齐失败:', err)
        }
      },
    })
  }, [clearTurnTimers, clearSlowHint, resetWatchdog])

  /** 关闭连接并复位连接状态（退出备课用） */
  const closeSSE = useCallback(() => {
    clearTurnTimers()
    sseRef.current?.close()
    sseRef.current = null
    planIdRef.current = null
    setSseState('connected')
  }, [clearTurnTimers])

  /** 手动重连（断线提示条按钮用） */
  const manualReconnect = useCallback((planId: string) => {
    setSseState('reconnecting')
    connectSSE(planId)
  }, [connectSSE])

  return { sseState, connectSSE, closeSSE, manualReconnect, startTurnTimers, clearTurnTimers }
}
