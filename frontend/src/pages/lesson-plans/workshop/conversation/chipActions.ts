/**
 * chipActions.ts — 芯片动作统一分发器 + 动态芯片协议转换层（迭代3.5 Phase A + B-2）
 *
 * 设计依据：产品设计文档 2.3「芯片协议」——所有芯片点击经本文件单点分发。
 * Phase B 接入 AI 动态 suggested_actions、Phase C 接入能力注册表时，
 * 只扩展本文件的 case 分支与 ChipContext 能力，不改各调用组件。
 *
 * B-2 新增：suggestedActionsToChips 协议转换函数——
 * 把后端 SSE suggested_actions 事件携带的 SuggestedAction 数组转换为前端 ChipDef，
 * 转换时做第二道白名单防御（后端 ParseSuggestedActions 是第一道）：
 *   - action_type 只放行协议五枚举（send_text/full_generate/switch_stage/open_tool/confirm_structure）；
 *   - 按类型校验 payload 必备字段，缺字段的"死按钮"芯片直接丢弃；
 *   - 上限 4 条（与后端 saMaxChips 对齐）。
 *
 * ChipContext 由页面组件注入：分发器自身无状态、不直接调 API，
 * 保证分发逻辑可独立单测、页面实现可替换。
 *
 * v192-fix（修复 full_generate 等动态芯片点击无反应）：
 *   问题根因——dispatchChip 的 switch 原本只实现了 send_text / advance_stage / publish /
 *   focus_input 四个 case，而协议五枚举里的 full_generate / switch_stage / open_tool /
 *   confirm_structure 全部掉进 default 被"静默忽略"，导致 write/revise 阶段 AI 下发的
 *   【⚡直接出完整教案】(full_generate)、【🔎去评审一遍】(switch_stage) 等芯片点了没反应。
 *   矛盾点：suggestedActionsToChips（转换层）放行这些类型、芯片能正常渲染，但 dispatchChip
 *   （分发层）不认它们——两层不一致。且 ChipContext 早已注入 fullGenerate/switchStage/openTool
 *   能力，只差分发器没去调。
 *   修法：在 dispatchChip 补全这四个 case，各自调用 ChipContext 中已注入的对应能力。
 *   仅补 case、不动任何现有 case 与转换层逻辑。
 */

import type { SuggestedAction } from '@/api/lesson-plans'
import type { ChipDef, ChipActionType } from './conversationScript'

/**
 * 芯片执行上下文 —— 页面组件实现并注入的能力集合
 */
export interface ChipContext {
  /** 发送一条文本消息（等价老师打字：本地上屏 + 调 chat 接口） */
  sendText: (text: string) => Promise<void> | void
  /** 一键生成指定阶段（含二次确认与必要的阶段切换） */
  fullGenerate: (stage: string) => Promise<void> | void
  /** 进入下一阶段（调 advanceStage） */
  advanceNext: () => Promise<void> | void
  /** 切换到指定阶段（调 switchToStage） */
  switchStage: (stageCode: string) => Promise<void> | void
  /** 发布教案（含确认流程） */
  publish: () => Promise<void> | void
  /** 聚焦输入框 */
  focusInput: () => void
  /** 唤起能力（Phase C 接能力注册表；当前支持 components/textbook/import，其余给提示） */
  openTool: (tool: string) => void
}

/**
 * 统一分发芯片动作
 *
 * 协议安全底线：未知 action_type 静默忽略（芯片是增强，缺了不阻塞）——
 * 这样 Phase B 中 AI 输出了前端尚未支持的类型时不会报错。
 */
/**
 * 判断一条芯片文本是否是"进入下一阶段"的确认语（而非实质教学内容）。
 * 命中则点击时直接走推进（advanceNext），不把它当聊天消息发给上一阶段——
 * 否则上一阶段会回一句多余的过渡告别，和下一阶段开场白重复成"两条"。
 * 口径收紧：必须同时含"进入"+阶段相关词，尽量不误伤正常内容芯片。
 */
function isStageAdvanceConfirmText(text: string): boolean {
  const t = (text || '').trim()
  if (!t.includes('进入')) return false
  return /进入.{0,12}(阶段|教学设计|教案撰写|撰写|修订|定稿|评审|下一)/.test(t)
}

export async function dispatchChip(ctx: ChipContext, chip: ChipDef): Promise<void> {
  switch (chip.action_type) {
    case 'send_text':
      // v192：若芯片文本是"进入下一阶段"的确认语，则直接走推进（写分隔符+触发下一阶段开场白，
      //       开场白已会承接前文），不再作为聊天消息发给上一阶段——避免上一阶段多回一句过渡告别。
      if (chip.payload?.text && isStageAdvanceConfirmText(chip.payload.text)) {
        await ctx.advanceNext()
      } else if (chip.payload?.text) {
        await ctx.sendText(chip.payload.text)
      }
      break

    case 'full_generate':
      // 协议五枚举之一（v192-fix 补全）：一键直接出完整教案。
      // payload.stage 指定目标阶段（如 write）；缺省时默认 write（与转换层注释一致）。
      // handleFullGenerate 内部已处理"非目标阶段则先 switchToStage + 二次态校验 + 常驻幻觉提示"。
      await ctx.fullGenerate((chip.payload?.stage && chip.payload.stage.trim()) ? chip.payload.stage : 'write')
      break

    case 'switch_stage':
      // 协议五枚举之一（v192-fix 补全）：跳到 payload.stage 指定的阶段（如 review/revise）。
      // 转换层 suggestedActionsToChips 已保证 switch_stage 芯片必带非空 payload.stage，
      // 此处再防一次：缺 stage 时静默忽略，不报错。
      if (chip.payload?.stage && chip.payload.stage.trim()) {
        await ctx.switchStage(chip.payload.stage)
      }
      break

    case 'open_tool':
      // 协议五枚举之一（v192-fix 补全）：唤起 payload.tool 指定的能力（components/textbook/import 等）。
      // 转换层已保证 open_tool 芯片必带非空 payload.tool；此处再防一次。
      if (chip.payload?.tool && chip.payload.tool.trim()) {
        ctx.openTool(chip.payload.tool)
      }
      break

    case 'confirm_structure':
      // 协议五枚举之一（v192-fix 补全）：确认结构卡。按协议约定"暂按 send_text 处理"，
      // 即把 payload.text 当作老师的确认话发送。转换层已保证 confirm_structure 必带非空 text。
      if (chip.payload?.text) {
        await ctx.sendText(chip.payload.text)
      }
      break

    case 'advance_stage':
      // Phase A 内部类型：进入下一阶段
      await ctx.advanceNext()
      break

    case 'publish':
      // Phase A 内部类型：发布教案
      await ctx.publish()
      break

    case 'focus_input':
      // Phase A 内部类型：聚焦输入框让老师自己说
      ctx.focusInput()
      break

    default:
      // 协议安全底线：未知类型静默忽略，不报错不阻塞
      break
  }
}

/* ==================== B-2 新增：动态芯片协议转换层 ==================== */

/**
 * AI 动态芯片允许的动作类型白名单 —— 协议五枚举（设计文档2.3）。
 * advance_stage/publish/focus_input 是前端内部扩展类型，仅剧本常量芯片可用，
 * AI 动态下发不允许（与后端 lesson_plan_gen_actions.go 的 saActionTypeWhitelist 逐字对齐）。
 */
const DYNAMIC_CHIP_ALLOWED_TYPES: ReadonlyArray<ChipActionType> = [
  'send_text', 'full_generate', 'switch_stage', 'open_tool', 'confirm_structure',
]

/** 动态芯片数量上限 —— 与后端 saMaxChips=4 对齐 */
const DYNAMIC_CHIP_MAX = 4

/**
 * 把后端 SSE 下发的 SuggestedAction 数组转换为前端 ChipDef 数组
 *
 * 防御规则（任一不满足即丢弃该条，绝不报错——芯片是增强缺了不阻塞）：
 *   1. label 必须为非空字符串（后端已保证，此处再防一次）；
 *   2. action_type 必须在五枚举白名单内；
 *   3. 按类型校验 payload 必备字段：
 *      - send_text / confirm_structure 必须有非空 payload.text（否则是点了没反应的死按钮）；
 *      - switch_stage 必须有非空 payload.stage；
 *      - open_tool 必须有非空 payload.tool；
 *      - full_generate 无硬性要求（dispatchChip 缺 stage 时默认 write）；
 *   4. 总数截断到 4 条。
 *
 * @param actions 后端 suggested_actions 事件携带的原始数组
 * @returns 清洗后的 ChipDef 数组（可能为空数组）
 */
export function suggestedActionsToChips(actions: SuggestedAction[]): ChipDef[] {
  const chips: ChipDef[] = []
  if (!Array.isArray(actions)) return chips

  for (const a of actions) {
    if (chips.length >= DYNAMIC_CHIP_MAX) break
    // 规则1：label 非空
    if (!a || typeof a.label !== 'string' || a.label.trim().length === 0) continue
    // 规则2：action_type 白名单
    const actionType = a.action_type as ChipActionType
    if (!DYNAMIC_CHIP_ALLOWED_TYPES.includes(actionType)) continue
    // 提取 payload 三个可能字段（后端 payload 是 map[string]interface{}，逐字段类型校验）
    const rawPayload = (a.payload || {}) as Record<string, unknown>
    const text = typeof rawPayload.text === 'string' && rawPayload.text.trim() ? rawPayload.text : undefined
    const stage = typeof rawPayload.stage === 'string' && rawPayload.stage.trim() ? rawPayload.stage : undefined
    const tool = typeof rawPayload.tool === 'string' && rawPayload.tool.trim() ? rawPayload.tool : undefined
    // 规则3：按类型校验必备字段，缺了直接丢弃该条
    if ((actionType === 'send_text' || actionType === 'confirm_structure') && !text) continue
    if (actionType === 'switch_stage' && !stage) continue
    if (actionType === 'open_tool' && !tool) continue

    chips.push({
      id: a.id && typeof a.id === 'string' ? a.id : `sa_${chips.length + 1}`,
      emoji: typeof a.emoji === 'string' ? a.emoji : '',
      label: a.label.trim(),
      action_type: actionType,
      payload: { text, stage, tool },
    })
  }
  return chips
}
