/**
 * 提示词危险分档 —— 前端元数据配置
 *
 * 后端 GetPromptCategory 下发三档：high / mid / kb（见 models/prompt.go）。
 * 本文件把每档映射为前端展示所需的：色板、图标 emoji、中文标签、分组标题、
 * 以及二次确认弹窗的文案与「是否需要手动键入确认」的强校验开关。
 *
 * 设计意图：
 *   - high（高危）：课件生成/渲染类，改错直接导致课件工坊批量生成崩溃。
 *     红色警示 + 二次确认要求手动键入「确认」二字（做法A，防手滑）。
 *   - kb（知识库）：课标/教材压缩入库类，影响面局限在知识库子系统。绿色 + 普通确认。
 *   - mid（中危）：其余全部（Pipeline/索引/业务），也是未登记新 key 的兜底档。橙色 + 普通确认。
 *
 * 供 PromptsPage.tsx（色标/分组）与 PromptConfirmModal.tsx（确认文案/强校验）共用。
 */
import type { PromptCategory } from '@/api/prompts'

// 单档元数据结构
export interface CategoryMeta {
  key: PromptCategory        // 档位标识
  label: string              // 中文短标签（用于色标 chip）
  groupTitle: string         // 分组标题（列表按档分组时的段落标题）
  emoji: string              // 图标 emoji（🔴/🟡/🟢）
  color: string              // 主色（文字/边框）
  bg: string                 // 浅底色（chip 背景 / 分组底）
  border: string             // 边框色
  // 二次确认弹窗文案
  confirmTitle: string       // 弹窗标题
  confirmDesc: string        // 弹窗正文警示
  requireTyping: boolean     // 是否要求手动键入确认词才能提交（仅 high 为 true）
}

// 需手动键入的确认词（high 档专用）
export const CONFIRM_KEYWORD = '确认'

// 三档元数据映射
export const CATEGORY_META: Record<PromptCategory, CategoryMeta> = {
  high: {
    key: 'high',
    label: '高危',
    groupTitle: '🔴 高危 · 课件生成与渲染（改错可能导致批量生成失败）',
    emoji: '🔴',
    color: '#c62828',
    bg: '#fdecea',
    border: '#ef9a9a',
    confirmTitle: '⚠️ 高危提示词修改确认',
    confirmDesc: '该提示词属于课件生成/渲染核心，包含精密的画布契约、字号硬约束、平台 API 占位符等约定。改动可能直接导致课件工坊批量生成失败或渲染异常。请务必确认改动经过验证。',
    requireTyping: true,
  },
  mid: {
    key: 'mid',
    label: '中危',
    groupTitle: '🟡 中危 · Pipeline / 索引 / 业务提示词',
    emoji: '🟡',
    color: '#e65100',
    bg: '#fff3e0',
    border: '#ffcc80',
    confirmTitle: '提示词修改确认',
    confirmDesc: '保存后将创建新版本并立即在对应业务链路生效（旧版本自动归档，可随时回滚）。请确认改动无误。',
    requireTyping: false,
  },
  kb: {
    key: 'kb',
    label: '知识库',
    groupTitle: '🟢 知识库 · 课标 / 教材压缩入库',
    emoji: '🟢',
    color: '#2e7d32',
    bg: '#e8f5e9',
    border: '#a5d6a7',
    confirmTitle: '提示词修改确认',
    confirmDesc: '该提示词用于知识库课标/教材压缩入库。保存后将创建新版本并生效（旧版本自动归档，可随时回滚）。请确认改动无误。',
    requireTyping: false,
  },
}

// 分组展示顺序（高危在前，引起注意；知识库相对独立放最后）
export const CATEGORY_ORDER: PromptCategory[] = ['high', 'mid', 'kb']

// 取某档元数据，未知档兜底为 mid（与后端兜底一致）
export function getCategoryMeta(cat: PromptCategory | string | undefined): CategoryMeta {
  if (cat === 'high' || cat === 'kb') return CATEGORY_META[cat]
  return CATEGORY_META.mid
}
