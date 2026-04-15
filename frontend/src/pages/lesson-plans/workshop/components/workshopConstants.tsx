/**
 * workshopConstants.tsx — 备课工坊共用常量和工具函数
 *
 * v73新增：
 *   renderMarkdown 增强——预处理清理孤立的*和#符号，
 *   确保AI输出的任何格式符号都不会以原始字符形式显示给用户。
 *
 * FE-WC-01修复：preprocessText 中的占位符从 §BOLD§/§END§ 改为 Unicode 私用区字符
 *   U+E001/U+E002，避免AI输出中恰好包含 § 字符时导致正则碰撞。
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
export const SUBJECTS = ['人工智能','语文','数学','英语','物理','化学','生物','历史','地理','政治','信息技术']
export const GRADES   = ['七年级','八年级','九年级','高一','高二','高三','小学低段','小学中段','小学高段']

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
  const puaBoldRe   = new RegExp(PUA_BOLD_START + '([^' + PUA_BOLD_START + PUA_BOLD_END + ']+)' + PUA_BOLD_END, 'g')

  return text
    .split('\n')
    .map(line => {
      const t = line.trim()

      // 保留标准markdown格式行，不处理
      if (/^#{1,3}\s+/.test(t)) return line      // ## 标题（有空格）
      if (/^[-*]\s+/.test(t)) return line         // - 列表或 * 列表
      if (/^\d+\.\s+/.test(t)) return line        // 1. 有序列表
      if (/^---+$/.test(t)) return line           // --- 分隔线
      if (/^\*\*[^*]/.test(t)) return line        // **开头的粗体行

      // 行首连续的#号（没有空格，不是标题）→ 去掉
      let result = line.replace(/^(\s*)#+([^#\s])/, '$1$2')

      // 行内孤立的单个*（不是**粗体**的部分）→ 去掉
      // 先用PUA字符保护**粗体**，再清理孤立*，再还原
      result = result
        .replace(/\*\*([^*]+)\*\*/g, PUA_BOLD_START + '$1' + PUA_BOLD_END)  // 保护**粗体**为PUA占位符
        .replace(/\*/g, '')                                                    // 去掉所有孤立*
        .replace(puaBoldRe, '**$1**')                                          // 还原**粗体**

      return result
    })
    .join('\n')
}

// ==================== 增强版Markdown渲染器 ====================

/**
 * renderMarkdown 将AI输出文本渲染为React节点
 *
 * 支持格式：
 *   # ## ### 标题（渲染为不同字号的粗体）
 *   **粗体**
 *   - 无序列表  * 无序列表
 *   1. 有序列表
 *   --- 分隔线
 *   普通段落
 *
 * v73增强：渲染前调用preprocessText清理孤立的*和#符号
 */
export function renderMarkdown(text: string): React.ReactNode {
  if (!text) return null

  // 预处理：清理不规范格式符号
  const cleaned = preprocessText(text)

  const lines = cleaned.split('\n')
  const nodes: React.ReactNode[] = []
  let listItems: React.ReactNode[] = []
  let listType: 'ul' | 'ol' | null = null
  let key = 0

  // 解析行内格式（**粗体**）
  const parseInline = (line: string): React.ReactNode => {
    // 同时处理 *斜体*（转为普通加粗显示，不用斜体）
    const parts = line.split(/(\*\*[^*]+\*\*|\*[^*]+\*)/)
    if (parts.length === 1) return line
    return (
      <>
        {parts.map((part, i) => {
          if (part.startsWith('**') && part.endsWith('**')) {
            return <strong key={i} style={{ fontWeight: 700, color: C.text }}>{part.slice(2, -2)}</strong>
          }
          if (part.startsWith('*') && part.endsWith('*') && part.length > 2) {
            // *斜体* → 渲染为普通加粗（不用斜体，更易读）
            return <strong key={i} style={{ fontWeight: 600, color: C.text }}>{part.slice(1, -1)}</strong>
          }
          return part
        })}
      </>
    )
  }

  const flushList = () => {
    if (!listItems.length) return
    nodes.push(
      listType === 'ul'
        ? <ul key={key++} style={{ margin: '6px 0 6px 16px', padding: 0, listStyle: 'disc' }}>{listItems}</ul>
        : <ol key={key++} style={{ margin: '6px 0 6px 16px', padding: 0, listStyle: 'decimal' }}>{listItems}</ol>
    )
    listItems = []; listType = null
  }

  for (const line of lines) {
    const t = line.trim()
    if (!t) { flushList(); continue }

    // 分隔线
    if (/^---+$/.test(t)) {
      flushList()
      nodes.push(<hr key={key++} style={{ border: 'none', borderTop: `1px solid ${C.border}`, margin: '10px 0' }} />)
      continue
    }

    // 标题（### ## #）
    const h3 = t.match(/^###\s+(.+)/)
    if (h3) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '14px', fontWeight: 700, color: C.text, margin: '10px 0 4px' }}>{parseInline(h3[1])}</div>); continue }

    const h2 = t.match(/^##\s+(.+)/)
    if (h2) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '15px', fontWeight: 700, color: C.text, margin: '12px 0 4px', paddingTop: '4px' }}>{parseInline(h2[1])}</div>); continue }

    const h1 = t.match(/^#\s+(.+)/)
    if (h1) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '16px', fontWeight: 700, color: C.text, margin: '14px 0 6px' }}>{parseInline(h1[1])}</div>); continue }

    // 无序列表（- 或 *）
    const ul = t.match(/^[-*]\s+(.+)/)
    if (ul) {
      if (listType !== 'ul') { flushList(); listType = 'ul' }
      listItems.push(<li key={key++} style={{ fontSize: '14px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(ul[1])}</li>)
      continue
    }

    // 有序列表
    const ol = t.match(/^\d+\.\s+(.+)/)
    if (ol) {
      if (listType !== 'ol') { flushList(); listType = 'ol' }
      listItems.push(<li key={key++} style={{ fontSize: '14px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(ol[1])}</li>)
      continue
    }

    // 普通段落
    flushList()
    nodes.push(<div key={key++} style={{ fontSize: '15px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(t)}</div>)
  }

  flushList()
  return <>{nodes}</>
}
