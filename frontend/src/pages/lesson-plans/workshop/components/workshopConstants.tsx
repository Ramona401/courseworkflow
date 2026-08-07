import { DEFAULT_SUBJECTS } from '@/constants/subjects'
import {
  renderMarkdown as renderLessonPlanMarkdown,
} from '@/pages/lesson-plans/plan-detail/components/planDetailConstants'

/**
 * workshopConstants.tsx — 备课工坊共用常量和工具函数
 *
 * v73新增：
 *   renderMarkdown 增强——预处理清理孤立的*和#符号，
 *   确保AI输出的任何格式符号都不会以原始字符形式显示给用户。
 *
 * FE-WC-01修复：preprocessText 中的占位符从 §BOLD§/§END§ 改为 Unicode 私用区字符
 *   U+E001/U+E002，避免AI输出中恰好包含 § 字符时导致正则碰撞。
 *
 * v202学科统一：SUBJECTS 三端（备课工坊/配方向导/课件工坊）统一为同一份 17 学科清单。
 *   本次变更——
 *   1) 「信息技术」改为「信息科技」，与课标知识库(curriculum_standards)及课件工坊对齐，
 *      否则该学科备课时匹配不到课标约束；
 *   2) 新增 科学/道德与法治/音乐/美术/体育/劳动 六个学科，
 *      其中「道德与法治」用全称与后端 aoci_component.go 的 SubjectCodeMap 对齐；
 *   3) 「政治」保留——义务教育段用「道德与法治」、高中段用「政治」，老师按学段选。
 *   注意：新学科在课标库暂无数据时为预期行为——能选、能备课，但不注入课标约束，
 *   与现有「暂无该学科年级数据」兜底一致，不报错不崩溃。
 *
 * 教案图片预览统一修复：
 *   - 继续保留本文件对AI输出中孤立格式符号的预处理；
 *   - 正式Markdown结构渲染统一复用教案详情页已经生产验证的渲染器；
 *   - 图片、链接、表格、标题、列表和粗体在备课工坊与各类教案预览中保持一致；
 *   - 不使用dangerouslySetInnerHTML，不执行正文中的HTML或脚本。
 */

// ==================== 颜色常量 ====================
export const C = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent:       '#F59E0B',
  success:      '#10B981',
  danger:       '#EF4444',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
  aiBubble:     '#EEF4FF',
  userBubble:   '#FFFFFF',
}

// ==================== 学科和年级选项 ====================
// v106修复：删除重复的 'AI'，统一使用 '人工智能'
// v202学科统一：三端统一为同一份 17 学科；'信息技术'→'信息科技'对齐课标库；
//   新增 科学/道德与法治/音乐/美术/体育/劳动（'道德与法治'用全称对齐后端编码表）。
export const SUBJECTS = [...DEFAULT_SUBJECTS]  // 单一真相源：见 @/constants/subjects（方案甲，v231）
export const GRADES   = ['一年级','二年级','三年级','四年级','五年级','六年级','七年级','八年级','九年级','高一','高二','高三']

// ==================== 流式消息状态类型 ====================
export interface StreamingState { id: string; content: string }

// ==================== Phase 7B-9：阶段相关常量 ====================

export const STAGE_STATUS_ICON: Record<string, string> = {
  pending:     '○',
  in_progress: '◉',
  completed:   '✓',
  skipped:     '⊘',
}

export const STAGE_STATUS_COLOR: Record<string, string> = {
  pending:     C.textMuted,
  in_progress: C.primary,
  completed:   C.success,
  skipped:     C.textMuted,
}

export const STAGE_CODE_EMOJI: Record<string, string> = {
  analyze: '🔍',
  design:  '🎯',
  write:   '✏️',
  review:  '🤖',
  revise:  '📝',
}

export const GATE_MODE_LABEL: Record<string, string> = {
  suggest: '建议确认',
  force:   '强制确认',
  auto:    '自动进入',
}

export const STAGE_CODE_NAME: Record<string, string> = {
  analyze: '教学分析',
  design:  '教学设计',
  write:   '教案撰写',
  review:  'AI评审',
  revise:  '修订定稿',
}

export const STAGE_CODE_ROLE: Record<string, string> = {
  analyze: '课程分析师',
  design:  '教学设计师',
  write:   '教案撰写专家',
  review:  '教学督导',
  revise:  '教案修订助手',
}

export const STAGE_CODE_DESC: Record<string, string> = {
  analyze: '分析教材、课标、学情、核心概念',
  design:  '制定教学目标、策略、活动方案',
  write:   '撰写完整教案内容',
  review:  '自动质量评审+改进建议',
  revise:  '根据评审意见修订定稿',
}

export const STAGE_REMOVABLE: Record<string, boolean> = {
  analyze: true,
  design:  true,
  write:   false,
  review:  true,
  revise:  false,
}

export const FLOW_MSG_COLORS: Record<string, { bg: string; border: string; text: string; icon: string }> = {
  info:    { bg: 'rgba(79,123,232,0.06)', border: 'rgba(79,123,232,0.15)', text: '#3B82F6', icon: 'ℹ️' },
  warning: { bg: 'rgba(245,158,11,0.06)', border: 'rgba(245,158,11,0.15)', text: '#D97706', icon: '⚠️' },
  error:   { bg: 'rgba(239,68,68,0.06)',  border: 'rgba(239,68,68,0.15)',  text: '#DC2626', icon: '🚫' },
}

export interface PromptModeOption {
  mode: 'guided' | 'efficient' | 'per_stage'
  label: string
  icon: string
  desc: string
  shortDesc: string
}

export const PROMPT_MODE_OPTIONS: PromptModeOption[] = [
  { mode: 'guided',    label: '引导版', icon: '🧭', desc: '逐步引导，多轮对话，适合新手或重要课程（15-25分钟）', shortDesc: '逐步引导' },
  { mode: 'efficient', label: '高效版', icon: '⚡', desc: '直接出方案，快速确认，适合经验丰富的老师（5-10分钟）', shortDesc: '快速出稿' },
  { mode: 'per_stage', label: '逐阶段', icon: '🎚️', desc: '每个阶段独立选择引导或高效模式，灵活搭配', shortDesc: '灵活搭配' },
]

export const STAGE_PROMPT_MODE_OPTIONS: { value: string; label: string }[] = [
  { value: 'guided',    label: '🧭 引导' },
  { value: 'efficient', label: '⚡ 高效' },
]

// ==================== 文本预处理：清理不规范的格式符号 ====================

/**
 * FE-WC-01修复：使用 Unicode 私用区字符作为临时占位符
 * U+E001 和 U+E002 属于 Private Use Area（PUA），不会出现在任何正常文本或AI输出中，
 * 彻底避免了原来使用 §BOLD§/§END§ 时可能与AI输出中的 § 字符发生正则碰撞的问题。
 */
const PUA_BOLD_START = '\uE001'
const PUA_BOLD_END   = '\uE002'

/**
 * preprocessText 在渲染前清理AI输出中的格式符号
 *
 * 处理规则（按顺序）：
 * 1. 行首的 # 符号：若不是标准markdown标题格式（#后面没有空格），去掉#
 * 2. 行首孤立的 * 符号（非列表、非粗体）：转为普通文字
 * 3. 行内孤立的单个 * 符号（非**bold**、非- item）：直接去掉
 * 4. 保留标准markdown：## 标题、**粗体**、- 列表、1. 列表、--- 分隔线
 */
function preprocessText(text: string): string {
  // FE-WC-01修复：构建基于PUA字符的正则，替代原来的 §BOLD§/§END§
  const puaBoldRe = new RegExp(
    PUA_BOLD_START +
      '([^' +
      PUA_BOLD_START +
      PUA_BOLD_END +
      ']+)' +
      PUA_BOLD_END,
    'g',
  )

  return text
    .split('\n')
    .map(line => {
      const t = line.trim()

      // 保留标准markdown格式行，不处理
      if (/^#{1,3}\s+/.test(t)) return line
      if (/^[-*]\s+/.test(t)) return line
      if (/^\d+\.\s+/.test(t)) return line
      if (/^---+$/.test(t)) return line
      if (/^\*\*[^*]/.test(t)) return line

      // 行首连续的#号（没有空格，不是标题）→ 去掉
      let result = line.replace(/^(\s*)#+([^#\s])/, '$1$2')

      // 行内孤立的单个*（不是**粗体**的部分）→ 去掉
      // 先用PUA字符保护**粗体**，再清理孤立*，最后还原。
      result = result
        .replace(
          /\*\*([^*]+)\*\*/g,
          PUA_BOLD_START + '$1' + PUA_BOLD_END,
        )
        .replace(/\*/g, '')
        .replace(puaBoldRe, '**$1**')

      return result
    })
    .join('\n')
}

// ==================== 统一Markdown渲染器 ====================

/**
 * renderMarkdown 将备课工坊文本渲染为React节点。
 *
 * 本函数只负责保留备课工坊已有的格式符号清理策略，正式Markdown解析统一复用
 * 教案详情页渲染器，确保以下内容在所有教案预览入口表现一致：
 *
 *   - #、##、### 标题；
 *   - **粗体**；
 *   - 无序列表和有序列表；
 *   - 分割线；
 *   - 块级图片与行内图片；
 *   - Markdown链接；
 *   - GFM表格；
 *   - 普通段落。
 *
 * 渲染过程只创建受控React节点，不执行正文内嵌HTML或脚本。
 */
export function renderMarkdown(text: string): React.ReactNode {
  if (!text) return null

  return renderLessonPlanMarkdown(
    preprocessText(text),
  )
}
