/**
 * conversationScript.ts — 对话式备课工坊·固定剧本库（迭代3.5 Phase A + A2-1 + A2-2）
 *
 * 设计依据：产品设计文档 2.2「默认剧本」+ 2.3「芯片协议」+ 2.4「能力注册表」。
 * 剧本=「在哪个引擎节点，AI说什么话，给哪些芯片」。本文件只存数据不存逻辑，
 * 台词与芯片的产品侧迭代只改本文件，不动组件代码（剧本修订规则）。
 * 培训流程卡必须与本文件芯片文案逐字对齐（验收标准7）。
 *
 * A2-1 批次新增（组件唤起）：
 *   - analyze/design/write 阶段芯片组各加一枚 🧩 组件芯片（open_tool components）；
 *   - 新增 PLUS_MENU_ITEMS「+」菜单定义（Phase C 能力注册表的最小先行版）；
 *   - 新增 STAGES_WITH_COMPONENTS 组件阶段表（与 WorkshopPage 同名常量对齐）。
 *
 * A2-2 批次新增（课本中途挂载）：
 *   - N0 开场芯片恢复剧本第三枚「📷 先传课本图片」（open_tool textbook）——
 *     后端 PUT /plans/{id}/textbooks 端点已上线，引擎每轮重读 textbook_page_ids，
 *     挂载后下一轮对话自动携带课本OCR上下文；
 *   - 「+」菜单课本项由页面侧解除灰显（菜单定义本就包含，可用性在页面计算）。
 *
 * v200 批次新增（design 阶段一键生成入口 + 写正文改名）：
 *   - design 芯片组补一枚 ⚡「一键生成教学设计」（full_generate, payload.stage=design）——
 *     对话模式此前 design 阶段无任何 full_generate 入口，老师想"一键出整份教学设计方案"
 *     却无芯片可点（只能逐句对话或直接推进去 write），导致后端 fullGenerateDesignPrompt
 *     （v200 已改为"输出完整教学设计方案骨架"）始终无法被触发。补上此芯片后链路打通：
 *     点击 → dispatchChip 路由到 handleFullGenerate('design') → sendChatMessage(full_generate:true)
 *     → 后端 resolveFullGeneratePrompt('design') → 输出教学设计方案骨架（design 阶段不落正文）。
 *     放在 design 芯片组首位并 highlight（一键出方案是该阶段最常用的省力路径）。
 *   - n2_write 芯片 label 由「按这个写正文」改为「按这个写教案」——"写教案"比"写正文"
 *     更贴近老师日常说法、更直白清晰（老师反馈"正文"一词不够直观）。仅改 label 文案，
 *     action_type/payload 不变（仍 advance_stage 推进到 write）。
 */

/** 阶段分隔符前缀 —— 与 WorkshopPage.tsx 内同名常量保持一致（识别并降级渲染分隔消息用） */
export const STAGE_SEP_PREFIX = '__STAGE_SEP__'

/**
 * 有组件注入映射的阶段列表（revise 无组件）——
 * 与 WorkshopPage.tsx 的 STAGES_WITH_COMPONENTS 及后端组件注入映射表对齐
 */
export const STAGES_WITH_COMPONENTS = ['analyze', 'design', 'write', 'review']

/** 芯片动作类型 */
export type ChipActionType =
  // ===== 协议正式枚举（设计文档2.3，Phase B AI动态芯片只允许这五种）=====
  | 'send_text'          // 把 payload.text 作为用户消息发送（等价老师打字）
  | 'full_generate'      // 一键成稿（payload.stage 指定阶段，默认 write）
  | 'switch_stage'       // 跳转到指定阶段（payload.stage）
  | 'open_tool'          // 唤起能力（payload.tool，Phase C 接完整能力注册表）
  | 'confirm_structure'  // 结构卡确认（迭代三接口预留，v1 按 send_text 降级）
  // ===== Phase A 前端内部扩展类型（仅剧本常量芯片使用）=====
  | 'advance_stage'      // 进入下一阶段（调 advanceStage API）
  | 'publish'            // 发布教案（含完整度确认）
  | 'focus_input'        // 聚焦输入框（"我自己说"类芯片）

/** 单个芯片定义 */
export interface ChipDef {
  /** 芯片唯一标识（埋点/防重复用） */
  id: string
  /** 表情前缀 */
  emoji: string
  /** 文案（≤8字，口语化） */
  label: string
  /** 动作类型 */
  action_type: ChipActionType
  /** 动作参数 */
  payload?: { text?: string; stage?: string; tool?: string }
  /** 是否高亮显示（默认推荐路径） */
  highlight?: boolean
  /** 仅当教案正文非空时显示 */
  requireContent?: boolean
  /** 仅当教案正文为空时显示 */
  requireNoContent?: boolean
}

/**
 * N0 开场芯片 —— StartConversation 返回开场白后、老师尚未发言时显示。
 * 剧本 N0：「按步骤来(默认高亮) / 直接出完整教案 / 先传课本图片」。
 * A2-2：第三枚「📷 先传课本图片」上线（后端中途挂载端点已就绪）。
 */
export const N0_CHIPS: ChipDef[] = [
  {
    id: 'n0_step_by_step',
    emoji: '📋',
    label: '按步骤来',
    action_type: 'send_text',
    payload: { text: '好的，我们按步骤来。请先帮我分析这节课的学情和教学目标。' },
    highlight: true,
  },
  {
    id: 'n0_full_gen',
    emoji: '⚡',
    label: '直接出完整教案',
    action_type: 'full_generate',
    payload: { stage: 'write' },
  },
  {
    id: 'n0_textbook',
    emoji: '📷',
    label: '先传课本图片',
    action_type: 'open_tool',
    payload: { tool: 'textbook' },
  },
]

/**
 * 各阶段后续芯片 —— 每轮 AI 回复完成后、按当前阶段渲染（剧本 N1-N4）。
 * Phase A 为确定性静态芯片；Phase B 接入 AI 动态 suggested_actions 后，
 * AI 给出的芯片优先、本表退化为解析失败时的兜底。
 * A2-1：🧩 组件芯片落地剧本 N1「这个年级学生啥特点」与 N2「看看活动设计组件」的组件唤起意图。
 */
export const STAGE_FOLLOWUP_CHIPS: Record<string, ChipDef[]> = {
  // N1：analyze 产出后 ——「这两条目标和重难点你看准不准？」
  analyze: [
    { id: 'n1_continue',   emoji: '✅', label: '这就开始设计', action_type: 'advance_stage', highlight: true },
    { id: 'n1_revise',     emoji: '✏️', label: '我说下要改的', action_type: 'focus_input' },
    { id: 'n1_components', emoji: '🧩', label: '看看参考组件', action_type: 'open_tool', payload: { tool: 'components' } },
  ],
  // N2：design 产出后 ——「按这个结构写教案？」（迭代三结构确认卡挂载点）
  // v200：首位补一键生成教学设计芯片（full_generate→design），让老师可一键出整份设计方案骨架；
  //       n2_write label「写正文」→「写教案」更直白。
  design: [
    { id: 'n2_full_gen',   emoji: '⚡', label: '一键生成教学设计', action_type: 'full_generate',
      payload: { stage: 'design' }, highlight: true },
    { id: 'n2_write',      emoji: '✅', label: '按这个写教案', action_type: 'advance_stage', highlight: true },
    { id: 'n2_rethink',    emoji: '🔁', label: '换个思路', action_type: 'send_text',
      payload: { text: '这个设计思路我不太满意，请换一个思路重新设计教学环节。' } },
    { id: 'n2_components', emoji: '🧩', label: '看看活动组件', action_type: 'open_tool', payload: { tool: 'components' } },
  ],
  // N3：write 阶段 —— 正文为空给一键成稿，正文已有给评审入口
  write: [
    { id: 'n3_step_write', emoji: '✍️', label: '逐环节写', action_type: 'send_text',
      payload: { text: '我们按环节一步一步写，不用一次写完。请接着往下写，一次只写一两个环节的具体教学话术和师生活动就停下来，等我确认了再继续写后面的。' },
      highlight: true, requireNoContent: true },
    { id: 'n3_full_gen',   emoji: '⚡', label: '一键写出完整正文', action_type: 'full_generate',
      payload: { stage: 'write' }, requireNoContent: true },
    { id: 'n3_review',     emoji: '🔎', label: 'AI评审一遍', action_type: 'advance_stage',
      highlight: true, requireContent: true },
    { id: 'n3_edit',       emoji: '✏️', label: '我再改改', action_type: 'focus_input', requireContent: true },
    { id: 'n3_components', emoji: '🧩', label: '找点素材', action_type: 'open_tool', payload: { tool: 'components' } },
  ],
  // N4：review 产出后 ——「要我按这两条直接改吗？」
  review: [
    { id: 'n4_revise',  emoji: '🛠', label: '按建议修改', action_type: 'advance_stage', highlight: true },
    { id: 'n4_publish', emoji: '✅', label: '不用改了，发布', action_type: 'publish' },
  ],
  // revise 阶段 —— 修订完成后发布收尾（v191 改动F：加"AI帮我改一版"省力路径）
  revise: [
    { id: 'n5_publish',   emoji: '🎉', label: '完成并发布', action_type: 'publish', highlight: true },
    { id: 'n5_ai_revise', emoji: '🤖', label: 'AI帮我改一版', action_type: 'full_generate', payload: { stage: 'revise' } },
    { id: 'n5_edit',      emoji: '✏️', label: '我自己说要改', action_type: 'focus_input' },
  ],
}

/**
 * 「+」能力菜单定义（A2-1：Phase C 能力注册表的最小先行版）。
 * 三条铁律之一在此体现：新增能力只加本表行 + 页面 openTool 分支，不改菜单渲染代码。
 * 可用条件由页面按运行态计算（菜单定义保持纯数据）。
 */
export interface PlusMenuItem {
  /** 能力ID（对应 openTool 的 tool 参数，未来与能力注册表 id 对齐） */
  tool: string
  emoji: string
  label: string
  /** 一句话说明（菜单项副标题） */
  desc: string
}

export const PLUS_MENU_ITEMS: PlusMenuItem[] = [
  { tool: 'components', emoji: '🧩', label: '教学组件', desc: '从组件库挑选参考组件加入对话' },
  { tool: 'textbook',   emoji: '📷', label: '课本图片', desc: '让AI贴着课文原文来设计（下一轮生效）' },
  { tool: 'import',     emoji: '📂', label: '导入教案', desc: '上传已有教案，AI评审并改进' },
]

/**
 * 一键生成各阶段文案配置 —— 与 WorkshopPage.handleFullGenerate 内
 * FULL_GEN_STAGE_META 逐字对齐（触发语契约：后端按固定触发语+full_generate标志出稿）。
 * 两处必须同步修改，勿单边改动。
 */
export const FULL_GEN_STAGE_META: Record<string, { name: string; trigger: string; confirmBody: string }> = {
  analyze: {
    name: '教学分析',
    trigger: '请一次性完成本节课的完整教学分析（教材分析、课程标准对接、学情分析、核心概念与重难点预判）。',
    confirmBody: '将由 AI 一次性生成完整的教学分析（教材分析、课程标准、学情分析、重难点预判）。',
  },
  design: {
    name: '教学设计',
    trigger: '请一次性完成本节课的完整教学设计方案（教学目标、重难点、教学策略、活动设计、评价设计）。',
    confirmBody: '将由 AI 一次性生成完整的教学设计方案（教学目标、重难点、教学策略、活动设计、评价设计）。',
  },
  write: {
    name: '教案撰写',
    trigger: '请一次性生成这节课的完整教案正文。',
    confirmBody: '将由 AI 一次性生成完整的教案正文（教学目标、重难点、教学过程、作业、板书等）。',
  },
  revise: {
    name: '修订定稿',
    trigger: '请基于已有教案正文和 AI 评审建议，一次性输出修订后的完整教案。',
    confirmBody: '将由 AI 参考已有教案正文与评审建议，一次性输出修订后的完整教案。',
  },
}

/**
 * 输入框 placeholder 示例指令（对冲"空白画布问题"，按消息轮次轮换）
 */
export const INPUT_PLACEHOLDERS: string[] = [
  '说说你的想法，比如：学生基础比较弱，进度放慢一点…',
  '试试说：把导入环节换成一个小游戏',
  '试试说：教学目标再聚焦一点，突出朗读训练',
  '试试说：作业设计分层，给学有余力的孩子加一题',
  '直接打字告诉AI你的任何想法…',
]

/**
 * 教案完整度清单定义（画布顶部，产物驱动不是流程步骤条）。
 * 标记词口径对齐后端 DetectLessonPlanContent 的判定标记词，不另造规则。
 */
export const CANVAS_CHECKLIST: Array<{ key: string; label: string; patterns: string[] }> = [
  { key: 'goal',     label: '教学目标', patterns: ['教学目标'] },
  { key: 'points',   label: '重难点',   patterns: ['重难点', '教学重点', '教学难点'] },
  { key: 'process',  label: '教学过程', patterns: ['教学过程', '教学环节', '教学活动', '环节', '教师话术', '学生活动'] },
  { key: 'homework', label: '作业',     patterns: ['作业'] },
  { key: 'board',    label: '板书',     patterns: ['板书'] },
]
