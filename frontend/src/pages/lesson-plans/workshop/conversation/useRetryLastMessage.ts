/**
 * useRetryLastMessage.ts — 对话模式「重试上一句 + 连续失败升级引导」Hook（路线乙·第2步）
 *
 * 设计依据：迭代3.5 harness 升级。第1步已治理"空流报错"——
 *   后端整轮空流软兜底（落库消息带 metadata.soft_retry=true）+ 前端 onError 改人话。
 * 第2步在此基础上给老师一个主动出口：一枚「🔄 重新回答」按钮，
 *   以及连续失败两次后的"换个说法/切专家模式"升级引导。
 *
 * 职责边界（与页面的分工）：
 *   - 本 Hook 只管「判断与计数」——canRetry 判定、failStreak 累计与归零、shouldEscalate 升级判定、
 *     handleRetry 编排（取最后一句 user 原文 + 上次组件 → 调页面注入的静默重发）；
 *   - 页面只管「动手与渲染」——通过 doSilentResend 注入真正的重发动作（调 chat 接口、不插重复 user 气泡），
 *     并消费 canRetry/shouldEscalate 渲染按钮与插入引导消息。
 *   这样重试逻辑内聚在 Hook 可独立推理，页面保持瘦。
 *
 * failStreak 计数契约（信号由 useConversationSSE 经页面回调透出）：
 *   - recordFailure()：识别到一次"软失败"时调用 —— 即 message_done 消息带 soft_retry 标记，
 *     或 SSE onError 触发。每次 +1。
 *   - recordSuccess()：识别到一次"正常 AI 回复"时调用 —— message_done 消息不带 soft_retry 标记。
 *     归零。
 *   - shouldEscalate：failStreak >= ESCALATE_THRESHOLD(2) 时为 true，由页面据此决定是否在
 *     该条失败消息之后追加一条本地引导消息（"换个说法/切专家模式"，不落库）。
 *
 * 与软兜底文案的递进关系（有意为之，非冗余）：
 *   软兜底消息本身已引导"再发一次/点按钮"（=重试）；
 *   第2次失败的升级引导再补"换个说法描述需求 / 切到专家模式分步来"（=换策略）。
 *   前者解决"偶发抽风"，后者解决"反复接不上"，是递进不是重复。
 */
import { useState, useRef, useCallback } from 'react'
import type { ConversationMessage } from '@/api/lesson-plans'

/** 连续失败达到此次数时触发升级引导 */
const ESCALATE_THRESHOLD = 2

/** Hook 入参 */
export interface UseRetryLastMessageParams {
  /** 当前对话消息列表（取"最后一句 user 原文"用） */
  messages: ConversationMessage[]
  /** AI 是否忙碌中（思考/流式/一键生成）——忙碌时不允许重试 */
  isBusy: boolean
  /**
   * 页面注入的"静默重发"动作：把指定文本作为用户消息重新发送，
   * 但本地不再插入新的 user 气泡（内容与上一句完全相同，重复插才违和），
   * 并复用上一次发送时携带的组件ID（页面用 ref 记住）。
   */
  doSilentResend: (text: string) => Promise<void> | void
  /** 阶段分隔符前缀（过滤系统分隔消息，定位真正的对话消息用） */
  stageSepPrefix: string
}

/** Hook 返回值 */
export interface UseRetryLastMessageResult {
  /** 是否可重试（!忙碌 且 最后一条真实消息是 assistant 且 存在至少一条 user 消息） */
  canRetry: boolean
  /** 连续失败是否已达升级阈值（页面据此决定是否插入升级引导消息） */
  shouldEscalate: boolean
  /** 执行重试：取最后一句 user 原文，经 doSilentResend 静默重发 */
  handleRetry: () => void
  /** 记录一次软失败（failStreak+1）——由页面在 onSoftFailure 信号里调用 */
  recordFailure: () => void
  /** 记录一次正常回复（failStreak 归零）——由页面在 onNormalReply 信号里调用 */
  recordSuccess: () => void
  /** 重置计数（退出备课 / 切换教案时调用，防跨会话残留） */
  resetStreak: () => void
}

/**
 * 取消息列表中最后一条「真实 user 消息」的原文。
 * 跳过系统阶段分隔消息；只认 role==='user'。找不到返回空串。
 */
function findLastUserText(messages: ConversationMessage[], stageSepPrefix: string): string {
  for (let i = messages.length - 1; i >= 0; i--) {
    const m = messages[i]
    if ((m.role as string) === 'system' && m.content.startsWith(stageSepPrefix)) continue
    if (m.role === 'user') {
      // 去掉 sendText 本地上屏时可能附加的"（已附 N 个参考组件）"尾注，
      // 重发只重发老师真正打的那句话（组件由 doSilentResend 复用 ref 携带，不靠文本）。
      const idx = m.content.indexOf('\n（已附 ')
      return (idx >= 0 ? m.content.slice(0, idx) : m.content).trim()
    }
  }
  return ''
}

/**
 * 对话模式重试 Hook
 */
export function useRetryLastMessage(params: UseRetryLastMessageParams): UseRetryLastMessageResult {
  const { messages, isBusy, doSilentResend, stageSepPrefix } = params

  /** 连续失败计数（驱动升级引导；正常回复归零） */
  const [failStreak, setFailStreak] = useState(0)

  // params 经 ref 透传，使 handleRetry 等回调身份稳定、不随渲染频繁重建
  const paramsRef = useRef(params)
  paramsRef.current = params

  /** 是否可重试：派生自当前 messages 与忙碌态 */
  const canRetry: boolean = (() => {
    if (isBusy) return false
    // 过滤系统分隔消息后取真实消息序列
    const real = messages.filter(m => !((m.role as string) === 'system' && m.content.startsWith(stageSepPrefix)))
    const last = real[real.length - 1]
    if (!last || last.role !== 'assistant') return false
    // 必须存在至少一条 user 消息（有"上一句"可重发）
    return real.some(m => m.role === 'user')
  })()

  /** 连续失败是否达升级阈值 */
  const shouldEscalate = failStreak >= ESCALATE_THRESHOLD

  /** 执行重试：取最后一句 user 原文静默重发（组件由页面侧 ref 复用，不在此携带） */
  const handleRetry = useCallback(() => {
    const p = paramsRef.current
    if (p.isBusy) return
    const text = findLastUserText(p.messages, p.stageSepPrefix)
    if (!text) return
    void p.doSilentResend(text)
  }, [])

  /** 软失败 +1 */
  const recordFailure = useCallback(() => {
    setFailStreak(n => n + 1)
  }, [])

  /** 正常回复归零 */
  const recordSuccess = useCallback(() => {
    setFailStreak(0)
  }, [])

  /** 重置（退出 / 切教案） */
  const resetStreak = useCallback(() => {
    setFailStreak(0)
  }, [])

  return { canRetry, shouldEscalate, handleRetry, recordFailure, recordSuccess, resetStreak }
}
