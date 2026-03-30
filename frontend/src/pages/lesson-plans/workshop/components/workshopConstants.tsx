/**
 * workshopConstants.tsx — 备课工坊共用常量和工具函数
 * 注意：含JSX（renderMarkdown返回ReactNode），必须用.tsx后缀
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
export const SUBJECTS = ['AI','人工智能','语文','数学','英语','物理','化学','生物','历史','地理','政治','信息技术']
export const GRADES   = ['七年级','八年级','九年级','高一','高二','高三','小学低段','小学中段','小学高段']

// ==================== 流式消息状态类型 ====================
export interface StreamingState { id: string; content: string }

// ==================== 轻量Markdown渲染器（含JSX）====================
export function renderMarkdown(text: string): React.ReactNode {
  if (!text) return null
  const lines = text.split('\n')
  const nodes: React.ReactNode[] = []
  let listItems: React.ReactNode[] = []
  let listType: 'ul' | 'ol' | null = null
  let key = 0

  const parseInline = (line: string): React.ReactNode => {
    const parts = line.split(/(\*\*[^*]+\*\*)/)
    if (parts.length === 1) return line
    return (
      <>
        {parts.map((part, i) =>
          part.startsWith('**') && part.endsWith('**')
            ? <strong key={i} style={{ fontWeight: 700, color: C.text }}>{part.slice(2, -2)}</strong>
            : part
        )}
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
    if (/^---+$/.test(t)) {
      flushList()
      nodes.push(<hr key={key++} style={{ border: 'none', borderTop: `1px solid ${C.border}`, margin: '10px 0' }} />)
      continue
    }
    const h3 = t.match(/^###\s+(.+)/); if (h3) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '14px', fontWeight: 700, color: C.text, margin: '10px 0 4px' }}>{parseInline(h3[1])}</div>); continue }
    const h2 = t.match(/^##\s+(.+)/);  if (h2) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '15px', fontWeight: 700, color: C.text, margin: '12px 0 4px' }}>{parseInline(h2[1])}</div>); continue }
    const h1 = t.match(/^#\s+(.+)/);   if (h1) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '16px', fontWeight: 700, color: C.text, margin: '14px 0 6px' }}>{parseInline(h1[1])}</div>); continue }
    const ul = t.match(/^[-*]\s+(.+)/); if (ul) { if (listType !== 'ul') { flushList(); listType = 'ul' }; listItems.push(<li key={key++} style={{ fontSize: '14px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(ul[1])}</li>); continue }
    const ol = t.match(/^\d+\.\s+(.+)/); if (ol) { if (listType !== 'ol') { flushList(); listType = 'ol' }; listItems.push(<li key={key++} style={{ fontSize: '14px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(ol[1])}</li>); continue }
    flushList()
    nodes.push(<div key={key++} style={{ fontSize: '15px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(t)}</div>)
  }
  flushList()
  return <>{nodes}</>
}
