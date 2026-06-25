/**
 * conversationChips.ts — 对话模式纯逻辑工具（从 ConversationModePage.tsx 抽出）
 *
 * 抽出动机：ConversationModePage.tsx 逼近 600 行红线，第2步（重试 harness）接入新逻辑前，
 * 把两块无副作用的纯逻辑搬到此处，给主页面腾出空间：
 *   1. computeVisibleChips —— 计算"当前应显示的芯片"（B-2 动态芯片优先 + 剧本常量兜底）；
 *   2. shouldHideHistoryMessage —— 渲染消息时的过滤谓词（隐藏引擎自动触发语等）。
 * 二者均为纯输入输出，可独立单测，不持任何状态、不调任何 API。
 *
 * 行为与原内联逻辑逐字等价，仅做位置搬移——本次不改任何判定规则。
 */
import {
  STAGE_SEP_PREFIX, N0_CHIPS, STAGE_FOLLOWUP_CHIPS, type ChipDef,
} from './conversationScript'
import type { ConversationMessage } from '@/api/lesson-plans'

/** computeVisibleChips 入参（页面把当前会话派生量打包传入） */
export interface VisibleChipsInput {
  phase: 'start' | 'chatting' | 'resuming'
  isBusy: boolean
  isStageMode: boolean
  messages: ConversationMessage[]
  dynamicChips: ChipDef[]
  currentStage: string
  planContent: string
}

/**
 * 计算当前应显示的芯片（行为与原 ConversationModePage 内联 IIFE 逐字一致）。
 *
 * B-2 优先级：本轮有 AI 动态芯片则用动态芯片（B-3 开提示词后才会出现），
 * 否则回退剧本常量（N0 或当前阶段后续芯片）兜底——剧本芯片永远是安全网。
 *
 * 方向A（动态芯片分支）：AI 芯片只取"内容芯片"，丢掉会静默切换的 switch_stage 导航芯片；
 * 推进统一交回剧本那枚确定性 advance 芯片（advanceNext → 推进+开场白，可靠且带过渡）。
 */
export function computeVisibleChips(input: VisibleChipsInput): ChipDef[] {
  const { phase, isBusy, isStageMode, messages, dynamicChips, currentStage, planContent } = input
  if (phase !== 'chatting' || isBusy || !isStageMode) return []

  const realMsgs = messages.filter(
    m => !((m.role as string) === 'system' && m.content.startsWith(STAGE_SEP_PREFIX))
  )
  const last = realMsgs[realMsgs.length - 1]
  if (!last || last.role !== 'assistant') return []

  const hasContent = !!(planContent && planContent.trim().length > 0)

  // B-2：AI 动态芯片优先（已经过 suggestedActionsToChips 白名单清洗）
  if (dynamicChips.length > 0) {
    const aiContentChips = dynamicChips.filter(
      c => c.action_type !== 'switch_stage' && c.action_type !== 'advance_stage'
    )
    const scriptAdvanceChip = (STAGE_FOLLOWUP_CHIPS[currentStage] || []).filter(
      c =>
        c.action_type === 'advance_stage' &&
        !(c.requireContent && !hasContent) &&
        !(c.requireNoContent && hasContent)
    )
    return [...aiContentChips, ...scriptAdvanceChip]
  }

  // 兜底：剧本常量芯片（确定性节点用确定性芯片）
  const userCount = realMsgs.filter(m => m.role === 'user').length
  const pool = userCount === 0 ? N0_CHIPS : (STAGE_FOLLOWUP_CHIPS[currentStage] || [])
  return pool.filter(c => {
    if (c.requireContent && !hasContent) return false
    if (c.requireNoContent && hasContent) return false
    return true
  })
}

/**
 * 渲染消息时的过滤谓词：返回 true 表示该消息应被隐藏（不渲染）。
 * 行为与原 ConversationModePage.renderMessages 内联 filter 逐字一致：
 *   - 隐藏引擎自动插入的"我们进入XX阶段了。"用户触发语；
 *   - 兜底过滤历史遗留的"📋 阶段评估"过程打分消息（对话模式从不展示过程打分）；
 *   - 隐藏评审阶段固定的触发指令文本。
 */
/**
 * isFullLessonPlanMessage —— 判定一条消息是否"一份完整教案"。
 * 用于 #2-乙：后面又生成了一份完整教案(定稿)时，把前面被取代的初稿折叠，避免整份教案出现两遍。
 * 判据从严，避免把"讨论里提到教案结构"的普通消息误判。
 */
export function isFullLessonPlanMessage(m: ConversationMessage): boolean {
  if (m.role !== 'assistant') return false
  const c = m.content || ''
  // 过程段：可能叫“教学过程/教学环节”，也可能直接用“环节一/二…”分节（与后端 hasProcess 口径对齐并补环节式命名）
  const hasProcess = c.includes('教学过程') || c.includes('教学环节') || c.includes('教学活动') || c.includes('环节一') || ((c.includes('教师话术') || c.includes('教师活动')) && c.includes('学生活动'))
  // 收尾段：与后端 hasEnding 口径对齐
  const hasEnding = c.includes('板书设计') || c.includes('作业') || c.includes('课堂小结') || c.includes('课堂总结')
  // 教案抬头
  const hasHead = c.includes('教学目标') || c.includes('教学重难点') || c.includes('教学重点')
  // 详案特征：必须含逐句教学话术/师生活动，把“设计阶段的环节大纲/方案总结”排除在外（大纲没这些细节）
  const hasDetail = c.includes('教师话术') || c.includes('学生活动') || c.includes('教师活动') || c.includes('预期反应')
  return hasProcess && hasEnding && hasHead && hasDetail
}

export function shouldHideHistoryMessage(m: ConversationMessage): boolean {
  if (m.role === 'user' && m.content.startsWith('我们进入') && m.content.includes('阶段了。')) return true
  if ((m.role as string) === 'assistant' && m.content.startsWith('📋 阶段评估')) return true
  if (m.role === 'user' && m.content === '请对上一阶段完成的教案进行全面专业评审,直接输出评审报告,包含各维度评分和改进建议。') return true
  return false
}
