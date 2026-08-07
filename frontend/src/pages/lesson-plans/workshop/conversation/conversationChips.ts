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
import {
  isConversationPublishIntent,
} from './conversationActionIntent'

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
 * 原Word格式投影仍失败时，只允许老师沿原段落继续修订。
 *
 * 该状态禁止发布和普通阶段推进；候选稿已由后端完整保留，
 * 第一枚芯片把同一候选重新投影到既有段落/表格槽位，
 * 第二枚芯片让老师直接指出要修改的原段落。
 */
const WORD_FORMAT_REJECTED_CHIPS: ChipDef[] = [
  {
    id: 'word_format_retry',
    emoji: '🛠',
    label: '按原段落重改',
    action_type: 'send_text',
    payload: {
      text:
        '请把上方候选修改内容严格映射回当前正式教案的原有段落和表格单元格：只能修改原段落文字，不得新增、删除、移动、拆分或合并任何段落及表格行列；保留全部图片和公式，格式校验通过后自动同步到正式教案。',
    },
    highlight: true,
  },
  {
    id: 'word_format_manual',
    emoji: '✏️',
    label: '我指定原段落',
    action_type: 'focus_input',
  },
]

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

  /*
   * Word格式投影失败是可恢复的特殊终态：
   * 候选稿已保留，但只能沿原段落重改，不能显示发布或阶段推进。
   */
  if (
    last.metadata?.word_format_rejected ===
    true
  ) {
    return WORD_FORMAT_REJECTED_CHIPS
  }

  /*
   * 错误、超时和未提交正式稿都不能继续显示发布或阶段推进芯片。
   *
   * 历史版本中的write/revise完整教案消息可能是在Word校验前展示的，
   * 因而没有content_committed=true。此类消息必须按未保存稿处理。
   */
  if (
    isConversationFailureMessage(last) ||
    isUncommittedLessonPlanArtifact(
      last,
      currentStage,
    )
  ) {
    return []
  }

  // AI动态芯片只补充内容动作，终态推进和发布始终使用确定性剧本芯片。
  if (dynamicChips.length > 0) {
    const aiContentChips = dynamicChips.filter(
      chip => {
        if (
          chip.action_type ===
            'switch_stage' ||
          chip.action_type ===
            'advance_stage'
        ) {
          return false
        }

        if (
          (
            chip.action_type ===
              'send_text' ||
            chip.action_type ===
              'confirm_structure'
          ) &&
          (
            isConversationPublishIntent(
              chip.payload?.text || '',
            ) ||
            isConversationPublishIntent(
              chip.label,
            )
          )
        ) {
          return false
        }

        return true
      },
    )

    const scriptTerminalChips =
      (
        STAGE_FOLLOWUP_CHIPS[
          currentStage
        ] || []
      ).filter(
        chip =>
          (
            chip.action_type ===
              'advance_stage' ||
            chip.action_type ===
              'publish'
          ) &&
          !(
            chip.requireContent &&
            !hasContent
          ) &&
          !(
            chip.requireNoContent &&
            hasContent
          ),
      )

    return deduplicateVisibleChips([
      ...aiContentChips,
      ...scriptTerminalChips,
    ])
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

/** 判断前端或SSE生成的失败消息。 */
function isConversationFailureMessage(
  message: ConversationMessage,
): boolean {
  const id = message.id || ''
  const content = (message.content || '').trim()

  const failurePrefixes = [
    'err_',
    'send_err_',
    'retry_err_',
    'fullgen_err_',
    'watchdog_',
    'adv_err_',
  ]

  return (
    failurePrefixes.some(prefix =>
      id.startsWith(prefix),
    ) ||
    content.startsWith('⚠️') ||
    content.includes('本轮没有保存') ||
    content.includes('本轮内容未展示也未发布')
  )
}

/**
 * 判断write/revise完整成稿是否缺少后端提交凭据。
 *
 * 普通讨论消息不受影响；只有严格命中完整教案判据的消息才需要
 * content_committed=true。
 */
function isUncommittedLessonPlanArtifact(
  message: ConversationMessage,
  currentStage: string,
): boolean {
  if (
    currentStage !== 'write' &&
    currentStage !== 'revise'
  ) {
    return false
  }

  if (!isFullLessonPlanMessage(message)) {
    return false
  }

  return (
    message.metadata?.content_committed !==
    true
  )
}

function deduplicateVisibleChips(
  chips: ChipDef[],
): ChipDef[] {
  const seen = new Set<string>()
  const result: ChipDef[] = []

  for (const chip of chips) {
    const key = [
      chip.action_type,
      chip.payload?.text || '',
      chip.payload?.stage || '',
      chip.payload?.tool || '',
      chip.label,
    ].join('\u001f')

    if (seen.has(key)) {
      continue
    }

    seen.add(key)
    result.push(chip)
  }

  return result
}
