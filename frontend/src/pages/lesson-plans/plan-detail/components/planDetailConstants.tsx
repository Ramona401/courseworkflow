/**
 * planDetailConstants.ts — 教案详情页共用常量、类型、工具函数
 *
 * v124 改动(图片插入预览修复):
 *   - renderMarkdown 新增三种 Markdown 元素支持:
 *     1. 整行图片 ![alt](url) → 居中块级 <img>(maxHeight 480px),
 *        alt 作为灰色斜体图注显示在图下方,加载失败显示红色占位
 *     2. 行内图片(图片与文字混在一行) → 行内小图(maxHeight 200px)
 *     3. 链接 [text](url) → <a target="_blank"> 蓝色下划线
 *   - parseInline 升级:用统一正则切分粗体/图片/链接,顺序判断避免误匹配
 *     (! 前缀的图片必须先于普通链接判断)
 *   - 不破坏既有功能:标题/列表/粗体/分割线一切照旧
 */
import type { LessonPlanStatus } from '@/api/lesson-plans'

// ==================== 颜色常量 ====================
export const C = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent:       '#F59E0B',
  success:      '#10B981',
  warning:      '#F97316',
  danger:       '#EF4444',
  purple:       '#8B5CF6',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
  borderHover:  '#E5E7EB',
  aiBubble:     '#EEF4FF',
}

// ==================== 状态配置 ====================
export interface StatusConfig { label: string; color: string; bg: string; dot: string }

export const STATUS_CONFIG: Record<LessonPlanStatus, StatusConfig> = {
  draft:              { label: '草稿',      color: C.textSec,  bg: '#F3F4F6',                dot: C.textMuted },
  published_personal: { label: '已发布',    color: C.primary,  bg: C.primaryLight,           dot: C.primary   },
  submitted:          { label: '待评审',    color: C.accent,   bg: 'rgba(245,158,11,0.08)',  dot: C.accent    },
  revision:           { label: '退回修改',  color: C.warning,  bg: 'rgba(249,115,22,0.08)',  dot: C.warning   },
  approved:           { label: '评审通过',  color: C.success,  bg: 'rgba(16,185,129,0.08)', dot: C.success   },
  published_shared:   { label: '已共享',    color: C.purple,   bg: 'rgba(139,92,246,0.08)', dot: C.purple    },
  developing:         { label: '课件开发中',color: '#0EA5E9',  bg: 'rgba(14,165,233,0.08)', dot: '#0EA5E9'   },
  completed:          { label: '已完成',    color: C.success,  bg: 'rgba(16,185,129,0.08)', dot: C.success   },
}

// ==================== Pipeline状态中文映射 ====================
export const PIPELINE_STATUS_LABEL: Record<string, { label: string; color: string; bg: string }> = {
  pending:          { label: '待启动',     color: C.textSec,  bg: '#F3F4F6' },
  running:          { label: '执行中',     color: C.primary,  bg: C.primaryLight },
  review_queue:     { label: '待人工审核', color: C.accent,   bg: 'rgba(245,158,11,0.08)' },
  pending_finalize: { label: '待确认定稿', color: C.warning,  bg: 'rgba(249,115,22,0.08)' },
  finalized:        { label: '已定稿',     color: C.success,  bg: 'rgba(16,185,129,0.08)' },
  needs_human:      { label: '需人工介入', color: C.warning,  bg: 'rgba(249,115,22,0.08)' },
  failed:           { label: '执行失败',   color: C.danger,   bg: 'rgba(239,68,68,0.08)'  },
  cancelled:        { label: '已取消',     color: C.textSec,  bg: '#F3F4F6' },
  verified:         { label: '验收通过',   color: C.success,  bg: 'rgba(16,185,129,0.08)' },
  verify_failed:    { label: '验收未通过', color: C.danger,   bg: 'rgba(239,68,68,0.08)'  },
}

// ==================== Tab配置 ====================
export type TabKey = 'content' | 'review' | 'stats' | 'courseware'
export interface TabConfig { key: TabKey; label: string }
export const TABS: TabConfig[] = [
  { key: 'content',    label: '📄 教案内容' },
  { key: 'review',     label: '🤖 AI评审'   },
  { key: 'stats',      label: '📊 使用统计' },
  { key: 'courseware', label: '🔗 关联课件' },
]

// ==================== 8步骤配置 ====================
export const STEP_ORDER = ['dbCheck','scanner','evaluator','meta','translator','generator','review','verify']

export const STEP_NAME_MAP: Record<string, string> = {
  dbCheck:'数据检查', scanner:'课程扫描', evaluator:'质量评估',
  meta:'元评估', translator:'方案翻译', generator:'页面生成',
  review:'人工审核', verify:'验收',
}

// ==================== 工具函数 ====================

/**
 * 日期时间格式化 yyyy-MM-dd HH:mm
 */
export function fmtDate(iso: string): string {
  try {
    const d = new Date(iso)
    return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
  } catch { return iso }
}

/**
 * 轻量Markdown渲染器
 * v124 支持的语法:
 *   块级:
 *     # ## ###  标题
 *     - / 1.    列表
 *     ---       分割线
 *     ![alt](url)  整行图片 → 居中大图+图注 (v124 新增)
 *   行内(在 parseInline 中处理):
 *     **粗体**
 *     ![alt](url)  行内小图 (v124 新增)
 *     [text](url)  链接,新标签页打开 (v124 新增)
 */
export function renderMarkdown(text: string): React.ReactNode {
  if (!text) return null
  const lines = text.split('\n')
  const nodes: React.ReactNode[] = []
  let listItems: React.ReactNode[] = []
  let listType: 'ul' | 'ol' | null = null
  let key = 0

  /**
   * 解析行内元素(粗体/图片/链接)
   * 用统一捕获组正则一次性切分,再按类型回填渲染
   *
   * 注意正则顺序:
   *   1. !\[...\]\(...\)  图片必须放最前面,因为它含 [ ] 会被链接正则误匹配
   *   2. \[...\]\(...\)   链接
   *   3. \*\*...\*\*      粗体
   */
  const parseInline = (line: string): React.ReactNode => {
    const INLINE_RE = /(!\[[^\]]*\]\([^)]+\)|\[[^\]]+\]\([^)]+\)|\*\*[^*]+\*\*)/g
    const parts = line.split(INLINE_RE)
    if (parts.length === 1) return line
    return <>{parts.map((p, i) => {
      if (!p) return null
      // 行内图片(放在文字中,最大高 200px 不破坏行高)
      const imgM = p.match(/^!\[([^\]]*)\]\(([^)]+)\)$/)
      if (imgM) {
        return (
          <img
            key={i}
            src={imgM[2]}
            alt={imgM[1]}
            style={{
              maxWidth: '100%', maxHeight: '200px',
              verticalAlign: 'middle', borderRadius: '4px', margin: '0 4px',
            }}
            onError={(e) => { (e.currentTarget as HTMLImageElement).style.opacity = '0.3' }}
          />
        )
      }
      // 链接
      const linkM = p.match(/^\[([^\]]+)\]\(([^)]+)\)$/)
      if (linkM) {
        return (
          <a key={i} href={linkM[2]} target="_blank" rel="noopener noreferrer"
            style={{ color: C.primary, textDecoration: 'underline' }}>
            {linkM[1]}
          </a>
        )
      }
      // 粗体
      if (p.startsWith('**') && p.endsWith('**')) {
        return <strong key={i} style={{ fontWeight: 700, color: C.text }}>{p.slice(2, -2)}</strong>
      }
      // 普通文本
      return p
    })}</>
  }

  const flushList = () => {
    if (!listItems.length) return
    nodes.push(listType === 'ul'
      ? <ul key={key++} style={{ margin: '6px 0 6px 16px', padding: 0, listStyle: 'disc' }}>{listItems}</ul>
      : <ol key={key++} style={{ margin: '6px 0 6px 16px', padding: 0, listStyle: 'decimal' }}>{listItems}</ol>)
    listItems = []; listType = null
  }

  for (const line of lines) {
    const t = line.trim()
    if (!t) { flushList(); continue }

    // ==================== v124: 整行只有图片 → 居中块级大图 ====================
    const imgOnly = t.match(/^!\[([^\]]*)\]\(([^)]+)\)$/)
    if (imgOnly) {
      flushList()
      const alt = imgOnly[1]
      const url = imgOnly[2]
      nodes.push(
        <div key={key++} style={{ textAlign: 'center', margin: '14px 0' }}>
          <img
            src={url}
            alt={alt}
            style={{
              maxWidth: '100%', maxHeight: '480px',
              borderRadius: '6px',
              boxShadow: '0 1px 4px rgba(0,0,0,0.1)',
              display: 'inline-block',
            }}
            onError={(e) => {
              // 图片加载失败:隐藏 img,显示后面的红色占位 div
              const img = e.currentTarget as HTMLImageElement
              img.style.display = 'none'
              const placeholder = img.nextElementSibling as HTMLDivElement | null
              if (placeholder) placeholder.style.display = 'inline-block'
            }}
          />
          {/* 加载失败时的占位提示(默认 hidden,onError 时显示) */}
          <div style={{
            display: 'none',
            padding: '14px 20px',
            background: 'rgba(239,68,68,0.06)',
            border: '1px dashed rgba(239,68,68,0.3)',
            borderRadius: '6px',
            color: C.danger,
            fontSize: '13px',
          }}>
            ⚠️ 图片加载失败:{alt || '(未命名)'}
          </div>
          {alt && (
            <div style={{
              fontSize: '12px', color: C.textMuted,
              marginTop: '6px', fontStyle: 'italic',
            }}>
              {alt}
            </div>
          )}
        </div>
      )
      continue
    }

    // 分割线
    if (/^---+$/.test(t)) {
      flushList()
      nodes.push(<hr key={key++} style={{ border: 'none', borderTop: `1px solid ${C.border}`, margin: '10px 0' }} />)
      continue
    }
    // 标题
    const h3 = t.match(/^###\s+(.+)/)
    if (h3) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '14px', fontWeight: 700, color: C.text, margin: '10px 0 4px' }}>{parseInline(h3[1])}</div>); continue }
    const h2 = t.match(/^##\s+(.+)/)
    if (h2) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '15px', fontWeight: 700, color: C.text, margin: '12px 0 4px' }}>{parseInline(h2[1])}</div>); continue }
    const h1 = t.match(/^#\s+(.+)/)
    if (h1) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '16px', fontWeight: 700, color: C.text, margin: '14px 0 6px' }}>{parseInline(h1[1])}</div>); continue }
    // 列表
    const ul = t.match(/^[-*]\s+(.+)/)
    if (ul) { if (listType !== 'ul') { flushList(); listType = 'ul' }; listItems.push(<li key={key++} style={{ fontSize: '14px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(ul[1])}</li>); continue }
    const ol = t.match(/^\d+\.\s+(.+)/)
    if (ol) { if (listType !== 'ol') { flushList(); listType = 'ol' }; listItems.push(<li key={key++} style={{ fontSize: '14px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(ol[1])}</li>); continue }
    // 普通段落
    flushList()
    nodes.push(<div key={key++} style={{ fontSize: '15px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(t)}</div>)
  }
  flushList()
  return <>{nodes}</>
}
