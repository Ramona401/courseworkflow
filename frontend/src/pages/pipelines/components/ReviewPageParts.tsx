/* eslint-disable react-refresh/only-export-components */
/**
 * ReviewPageParts.tsx — 审核页子组件集合
 *
 * v69拆分自 PipelineReviewPage.tsx（编号8方案2）
 * 包含：
 *   - 常量和工具函数（OP_COLORS/OP_NAMES等）
 *   - CodeView — 只读代码视图组件
 *   - HTMLPreview — iframe HTML预览组件
 *   - PageSidebar — 左侧页面列表
 *   - PageToolbar — 页面工具栏
 *   - PagePreview — 页面预览区（支持代码视图+编辑+双版本对比）
 *   - MergeCompareView — 合并页对比视图
 */
import { useRef } from 'react'
import type { GeneratedPageFull } from '@/api/pipelines'
import {
  Check, X, Edit3, CheckCircle,
  Maximize2, FileText, Code,
} from 'lucide-react'

// ==================== 常量（导出供主组件和弹窗组件使用）====================

/** 新增页面(create)的虚拟页码偏移量 */
export const CREATE_PAGE_OFFSET = 1000

/** 操作类型颜色 */
export const OP_COLORS: Record<string, string> = { keep: '#34c759', modify: '#007aff', create: '#af52de', merge: '#ff9500', delete: '#ff3b30' }
/** 操作类型中文名 */
export const OP_NAMES: Record<string, string> = { keep: '保留', modify: '修改', create: '新建', merge: '合并', delete: '删除' }
/** 决策状态颜色 */
export const DECISION_COLORS: Record<string, string> = { approve: '#34c759', reject: '#ff3b30', edit: '#ff9500', pending: '#c7c7cc' }
/** 决策状态中文名 */
export const DECISION_NAMES: Record<string, string> = { approve: '已采用', reject: '已拒绝', edit: '已编辑', pending: '待决策' }

// ==================== 工具函数（导出供外部使用）====================

/** 解析合并来源页码数组 */
export function parseMergeSources(ms: string): number[] {
  try { if (ms && ms !== 'null') return JSON.parse(ms) } catch { /* ignore */ }
  return []
}

/** 判断是否为虚拟页面 */
export function isVirtualPage(pageNumber: number): boolean {
  return pageNumber >= CREATE_PAGE_OFFSET
}

/** 获取虚拟页面对应的原始位置页码 */
export function getVirtualPageAnchor(page: { page_number: number; page_title: string }): number {
  const m = page.page_title.match(/^P(\d{1,3})-new/)
  if (m) return parseInt(m[1], 10)
  const offset = page.page_number - CREATE_PAGE_OFFSET
  return offset < 100 ? offset : offset % 100
}

/** 格式化页面标签 */
export function formatPageLabel(pageNumber: number, pageTitle?: string): string {
  if (pageNumber < CREATE_PAGE_OFFSET) return `P${String(pageNumber).padStart(2, '0')}`
  if (pageTitle) {
    const m = pageTitle.match(/^P(\d{1,3})-new/)
    if (m) return `P${m[1].padStart(2, '0')}-new`
  }
  const offset = pageNumber - CREATE_PAGE_OFFSET
  return `P${String(offset < 100 ? offset : offset % 100).padStart(2, '0')}-new`
}

/** 获取页面最终版本HTML（优先final_html） */
export function getEffectiveHTML(page: GeneratedPageFull): string {
  return page.final_html || page.generated_html || ''
}

/** v99优化7：检测modify页面的change_reason中是否包含合并操作描述 */
export function hasHiddenMerge(page: GeneratedPageFull): boolean {
  if (page.operation !== 'modify' || !page.change_reason) return false
  const reason = page.change_reason
  return reason.includes('合并') || reason.includes('merge') || /原P\d+\+P\d+/.test(reason) || /P\d+\+P\d+合并/.test(reason)
}

/** 判断事件目标是否为输入元素 */
export function isInputElement(target: EventTarget | null): boolean {
  if (!target || !(target instanceof HTMLElement)) return false
  const tag = target.tagName.toLowerCase()
  if (tag === 'textarea' || tag === 'input') return true
  if (target.isContentEditable) return true
  return false
}

/** 逻辑排序页面列表 */
export function sortPagesLogically(pages: GeneratedPageFull[]): GeneratedPageFull[] {
  const normalPages = pages.filter(p => !isVirtualPage(p.page_number))
  const virtualPages = pages.filter(p => isVirtualPage(p.page_number))
  normalPages.sort((a, b) => a.page_number - b.page_number)
  const virtualByAnchor = new Map<number, GeneratedPageFull[]>()
  for (const vp of virtualPages) {
    const anchor = getVirtualPageAnchor(vp)
    if (!virtualByAnchor.has(anchor)) virtualByAnchor.set(anchor, [])
    virtualByAnchor.get(anchor)!.push(vp)
  }
  for (const [, vps] of virtualByAnchor) { vps.sort((a, b) => a.page_number - b.page_number) }
  const result: GeneratedPageFull[] = []
  for (const np of normalPages) {
    result.push(np)
    const vpList = virtualByAnchor.get(np.page_number)
    if (vpList) { for (const vp of vpList) { result.push(vp) } }
  }
  for (const [anchor, vpList] of virtualByAnchor) {
    const anchorExists = normalPages.some(np => np.page_number === anchor)
    if (!anchorExists) {
      let insertAfterIdx = -1
      for (let i = result.length - 1; i >= 0; i--) {
        if (!isVirtualPage(result[i].page_number) && result[i].page_number < anchor) { insertAfterIdx = i; break }
      }
      if (insertAfterIdx >= 0) {
        let insertPos = insertAfterIdx + 1
        while (insertPos < result.length && isVirtualPage(result[insertPos].page_number)) { insertPos++ }
        result.splice(insertPos, 0, ...vpList)
      } else {
        for (const vp of vpList) { result.push(vp) }
      }
    }
  }
  return result
}

// ==================== 浏览器全屏API ====================

export function requestTrueFullscreen() {
  const t = document.documentElement
   
  if (t.requestFullscreen) t.requestFullscreen().catch(() => {})
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  else if ((t as any).webkitRequestFullscreen) (t as any).webkitRequestFullscreen()
}
export function exitTrueFullscreen() {
   
  if (document.fullscreenElement) document.exitFullscreen().catch(() => {})
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  else if ((document as any).webkitFullscreenElement) (document as any).webkitExitFullscreen()
}
 
export function isTrueFullscreen(): boolean {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return !!(document.fullscreenElement || (document as any).webkitFullscreenElement)
}

// ==================== CodeView 只读代码视图 ====================

export function CodeView({ html, label }: { html: string; label?: string }) {
  const lines = (html || '(空内容)').split('\n')
  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      {label && (
        <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#5856d6', background: '#f5f0ff', flexShrink: 0, borderBottom: '1px solid rgba(88,86,214,0.1)', display: 'flex', alignItems: 'center', gap: 6 }}>
          <Code size={12} /> {label}
          <div style={{ flex: 1 }} />
          <button onClick={() => { navigator.clipboard.writeText(html || '').then(() => { const btn = document.activeElement as HTMLButtonElement; if (btn) { btn.textContent = '已复制 ✓'; setTimeout(() => { btn.textContent = '复制'; }, 1500) } }).catch(() => {}) }} style={{ padding: '2px 10px', borderRadius: 4, border: '1px solid rgba(88,86,214,0.2)', background: 'rgba(88,86,214,0.08)', fontSize: 11, fontWeight: 500, color: '#5856d6', cursor: 'pointer' }}>复制</button>
        </div>
      )}
      <div style={{ flex: 1, overflow: 'auto', background: '#1e1e1e', padding: 0, fontFamily: 'Monaco, Consolas, "Courier New", monospace', fontSize: 12, lineHeight: 1.7 }}>
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <tbody>
            {lines.map((line, i) => (
              <tr key={i}>
                <td style={{ width: 50, minWidth: 50, textAlign: 'right', padding: '0 10px 0 8px', color: '#858585', userSelect: 'none', verticalAlign: 'top', borderRight: '1px solid #333', whiteSpace: 'nowrap' }}>{i + 1}</td>
                <td style={{ padding: '0 12px', color: '#d4d4d4', whiteSpace: 'pre', wordBreak: 'break-all' }}>{line || ' '}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

// ==================== HTMLPreview iframe预览 ====================

export function HTMLPreview({ html }: { html: string }) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const doc = '<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><style>body{margin:0;padding:16px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;font-size:14px;line-height:1.6;color:#1c1c1e;word-wrap:break-word}img{max-width:100%;height:auto}table{border-collapse:collapse;width:100%}th,td{border:1px solid #e5e5ea;padding:8px 12px;text-align:left}th{background:#f5f5f7;font-weight:600}</style></head><body>' + (html || '<p style="color:#aeaeb2;text-align:center;padding:40px 0">(空内容)</p>') + '</body></html>'
  if (!html && html !== '') return <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 13 }}>（无HTML内容）</div>
  return <iframe ref={iframeRef} srcDoc={doc} sandbox="allow-same-origin allow-scripts" style={{ flex: 1, width: '100%', height: '100%', border: 'none', background: '#fff' }} title="HTML预览" />
}

// ==================== 页面HTML加载中占位组件 ====================

export function PageHTMLLoading() {
  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 12, color: '#8e8e93' }}>
      <div style={{ width: 24, height: 24, border: '3px solid #e5e5ea', borderTopColor: '#007aff', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
      <div style={{ fontSize: 13 }}>加载页面内容...</div>
      <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`}</style>
    </div>
  )
}

// ==================== PageSidebar 左侧页面列表 ====================

export function PageSidebar({ sortedPages, selectedIdx, opStats, totalPages, onSelectPage }: {
  sortedPages: GeneratedPageFull[]; selectedIdx: number; opStats: Record<string, number>; totalPages: number; onSelectPage: (idx: number) => void
}) {
  return (
    <div style={{ width: 300, flexShrink: 0, background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)', borderRadius: '14px 0 0 14px', overflow: 'auto' }}>
      <div style={{ padding: '14px 16px', fontSize: 13, fontWeight: 600, color: '#1c1c1e', borderBottom: '1px solid rgba(0,0,0,0.04)', position: 'sticky', top: 0, background: 'rgba(255,255,255,0.95)', backdropFilter: 'blur(20px)', zIndex: 2 }}>
        <div>课程页面 ({totalPages})</div>
        <div style={{ display: 'flex', gap: 6, marginTop: 6, flexWrap: 'wrap' }}>
          {Object.entries(opStats).map(([op, count]) => <span key={op} style={{ fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px', borderRadius: 4, background: OP_COLORS[op] || '#aeaeb2' }}>{OP_NAMES[op] || op} {count}</span>)}
        </div>
      </div>
      {sortedPages.map((page, idx) => (
        <div key={page.page_number} onClick={() => onSelectPage(idx)} style={{ padding: '10px 16px', cursor: 'pointer', background: idx === selectedIdx ? 'rgba(0,122,255,0.08)' : 'transparent', borderBottom: '1px solid rgba(0,0,0,0.03)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 6px', borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2', flexShrink: 0 }}>{OP_NAMES[page.operation] || page.operation}</span>
            <span style={{ fontSize: 13, fontWeight: 500, color: '#1c1c1e', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
              {formatPageLabel(page.page_number, page.page_title)}. {page.page_title || '无标题'}
            </span>
          </div>
          {page.operation === 'merge' && page.merge_sources && page.merge_sources !== 'null' && (
            <div style={{ fontSize: 10, color: '#ff9500', marginTop: 2, paddingLeft: 42 }}>合并自: {parseMergeSources(page.merge_sources).map(n => 'P' + n).join('+')}</div>
          )}
          {isVirtualPage(page.page_number) && (
            <div style={{ fontSize: 10, color: '#af52de', marginTop: 2, paddingLeft: 42 }}>新增于 P{String(getVirtualPageAnchor(page)).padStart(2, '0')} 之后</div>
          )}
          {hasHiddenMerge(page) && (
            <div style={{ fontSize: 10, color: '#ff9500', marginTop: 2, paddingLeft: 42 }}>含页面合并</div>
          )}
          <div style={{ fontSize: 11, marginTop: 3, paddingLeft: 42, color: DECISION_COLORS[page.decision] || '#c7c7cc', fontWeight: page.decision !== 'pending' ? 600 : 400 }}>{DECISION_NAMES[page.decision] || page.decision}</div>
        </div>
      ))}
      {sortedPages.length === 0 && <div style={{ padding: 20, textAlign: 'center', color: '#aeaeb2', fontSize: 13 }}>暂无生成页面</div>}
    </div>
  )
}

// ==================== PageToolbar 页面工具栏 ====================

export function PageToolbar({ page, btn, editingHTML, deciding, isPendingFinalize, isSuperReviewer, canOperate, reasonExpanded, codeViewMode, htmlLoaded, onCodeViewToggle, onReasonToggle, onEnterFullscreen, onDecision, onStartEdit, onSaveEdit, onCancelEdit }: {
  page: GeneratedPageFull; btn: React.CSSProperties; editingHTML: boolean; deciding: boolean
  isPendingFinalize: boolean; isSuperReviewer: boolean; canOperate: boolean; reasonExpanded: boolean
  codeViewMode: boolean; htmlLoaded: boolean; onCodeViewToggle: () => void
  onReasonToggle: () => void; onEnterFullscreen: () => void
  onDecision: (d: 'approve' | 'reject' | 'edit', html?: string) => void
  onStartEdit: () => void; onSaveEdit: () => void; onCancelEdit: () => void
}) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '10px 16px', borderBottom: '1px solid rgba(0,0,0,0.04)', flexShrink: 0, flexWrap: 'wrap' }}>
      <span style={{ fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px', borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2' }}>{OP_NAMES[page.operation] || page.operation}</span>
      <span style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>{formatPageLabel(page.page_number, page.page_title)}. {page.page_title || '无标题'}</span>
      {page.operation === 'merge' && page.merge_sources && page.merge_sources !== 'null' && <span style={{ fontSize: 11, color: '#ff9500', fontWeight: 500 }}>(合并自 {parseMergeSources(page.merge_sources).map(n => 'P' + n).join(' + ')})</span>}
      {isVirtualPage(page.page_number) && <span style={{ fontSize: 11, color: '#af52de', fontWeight: 500 }}>（新增页面，位于 P{String(getVirtualPageAnchor(page)).padStart(2, '0')} 之后）</span>}
      {hasHiddenMerge(page) && <span style={{ fontSize: 11, color: '#ff9500', fontWeight: 500 }}>（含页面合并）</span>}
      {/* v69新增：HTML未加载提示 */}
      {!htmlLoaded && !editingHTML && <span style={{ fontSize: 11, color: '#007aff', fontWeight: 500 }}>⏳ 加载中...</span>}
      <div style={{ flex: 1 }} />
      {!editingHTML && htmlLoaded && (
        <button onClick={onCodeViewToggle} style={{ ...btn, padding: '6px 10px', fontSize: 12, background: codeViewMode ? '#f5f0ff' : '#fff', color: codeViewMode ? '#5856d6' : '#3c3c43', border: codeViewMode ? '1px solid rgba(88,86,214,0.3)' : '1px solid rgba(0,0,0,0.08)' }} title={codeViewMode ? '切换到预览' : '查看源代码'}>
          <Code size={13} /> {codeViewMode ? '预览' : '代码'}
        </button>
      )}
      {!editingHTML && htmlLoaded && <button onClick={onEnterFullscreen} style={{ ...btn, padding: '6px 10px', fontSize: 12 }} title="全屏预览"><Maximize2 size={13} /> 全屏预览</button>}
      {page.change_reason && <button onClick={onReasonToggle} style={{ ...btn, padding: '6px 10px', fontSize: 12, background: reasonExpanded ? '#f0f7ff' : '#fff', color: reasonExpanded ? '#007aff' : '#3c3c43' }}><FileText size={13} /> {reasonExpanded ? '收起理由' : '修改理由'}</button>}
      {!editingHTML && !isPendingFinalize && canOperate && htmlLoaded && (
        <div style={{ display: 'flex', gap: 6 }}>
          <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }} onClick={() => onDecision('approve')} disabled={deciding}><Check size={14} /> 采用</button>
          <button style={{ ...btn, background: '#ff3b30', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }} onClick={() => onDecision('reject')} disabled={deciding}><X size={14} /> 拒绝</button>
          <button style={{ ...btn, background: '#ff9500', color: '#fff', border: 'none', padding: '6px 14px' }} onClick={onStartEdit}><Edit3 size={14} /> 编辑</button>
        </div>
      )}
      {!editingHTML && isPendingFinalize && isSuperReviewer && htmlLoaded && (
        <div style={{ display: 'flex', gap: 6 }}>
          <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }} onClick={() => onDecision('approve')} disabled={deciding}><Check size={14} /> 采用</button>
          <button style={{ ...btn, background: '#ff3b30', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }} onClick={() => onDecision('reject')} disabled={deciding}><X size={14} /> 拒绝</button>
          <button style={{ ...btn, background: '#ff9500', color: '#fff', border: 'none', padding: '6px 14px' }} onClick={onStartEdit}><Edit3 size={14} /> 编辑</button>
        </div>
      )}
      {isPendingFinalize && !isSuperReviewer && <span style={{ fontSize: 12, color: '#ff9500', fontWeight: 500 }}>⏳ 待超级审核员确认</span>}
      {editingHTML && (
        <div style={{ display: 'flex', gap: 6 }}>
          <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1 }} onClick={onSaveEdit} disabled={deciding}><CheckCircle size={14} /> 保存编辑</button>
          <button style={btn} onClick={onCancelEdit}>取消</button>
        </div>
      )}
    </div>
  )
}

// ==================== PagePreview 页面预览 ====================

export function PagePreview({ page, editingHTML, editContent, onEditContentChange, mergeSourceTab, onMergeSourceTabChange, codeViewMode, htmlLoaded }: {
  page: GeneratedPageFull; editingHTML: boolean; editContent: string; onEditContentChange: (v: string) => void
  mergeSourceTab: number; onMergeSourceTabChange: (idx: number) => void
  codeViewMode: boolean; htmlLoaded: boolean
}) {
  // v69：HTML未加载完成时显示加载占位
  if (!htmlLoaded && !editingHTML) {
    return <PageHTMLLoading />
  }

  // 编辑模式
  if (editingHTML) {
    return (
      <textarea value={editContent} onChange={e => onEditContentChange(e.target.value)}
        onKeyDown={e => { if (['ArrowLeft','ArrowRight','ArrowUp','ArrowDown'].includes(e.key)) e.stopPropagation() }}
        style={{ width: '100%', height: '100%', border: 'none', outline: 'none', fontFamily: 'Monaco, Consolas, "Courier New", monospace', fontSize: 12, lineHeight: 1.6, padding: 16, resize: 'none', background: '#fafafa', color: '#1c1c1e', boxSizing: 'border-box' }}
      />
    )
  }

  // 代码视图
  if (codeViewMode) {
    if (page.operation === 'modify') return <div style={{ display: 'flex', height: '100%' }}><div style={{ flex: '1 1 50%', minWidth: 0, maxWidth: '50%', borderRight: '2px solid rgba(0,0,0,0.08)', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}><CodeView html={page.original_html} label="原版代码" /></div><div style={{ flex: '1 1 50%', minWidth: 0, maxWidth: '50%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}><CodeView html={getEffectiveHTML(page)} label="修改后代码" /></div></div>
    if (page.operation === 'merge') return <div style={{ display: 'flex', height: '100%' }}><div style={{ flex: '1 1 50%', minWidth: 0, maxWidth: '50%', borderRight: '2px solid rgba(0,0,0,0.08)', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}><CodeView html={page.original_html} label="源页面代码" /></div><div style={{ flex: '1 1 50%', minWidth: 0, maxWidth: '50%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}><CodeView html={getEffectiveHTML(page)} label="合并结果代码" /></div></div>
    if (page.operation === 'create') return <CodeView html={getEffectiveHTML(page)} label="新增页面代码" />
    if (page.operation === 'delete') return <CodeView html={page.original_html} label="待删除页面代码" />
    return <CodeView html={page.original_html || page.final_html || ''} label="页面代码" />
  }

  // 预览模式
  if (page.operation === 'merge') return <MergeCompareView page={page} mergeSourceTab={mergeSourceTab} onTabChange={onMergeSourceTabChange} />
  if (page.operation === 'modify') return (
    <div style={{ display: 'flex', height: '100%' }}>
      <div style={{ flex: '1 1 50%', minWidth: 0, maxWidth: '50%', borderRight: '2px solid rgba(0,0,0,0.08)', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0, display: 'flex', alignItems: 'center' }}>原版<div style={{ flex: 1 }} /><button onClick={() => navigator.clipboard.writeText(page.original_html || '')} style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid rgba(0,0,0,0.1)', background: '#fff', fontSize: 11, color: '#8e8e93', cursor: 'pointer' }}>复制</button></div><HTMLPreview html={page.original_html} /></div>
      <div style={{ flex: '1 1 50%', minWidth: 0, maxWidth: '50%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#007aff', background: '#f0f7ff', flexShrink: 0, display: 'flex', alignItems: 'center' }}>修改后<div style={{ flex: 1 }} /><button onClick={() => navigator.clipboard.writeText(getEffectiveHTML(page) || '')} style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid rgba(0,0,0,0.1)', background: '#fff', fontSize: 11, color: '#007aff', cursor: 'pointer' }}>复制</button></div><HTMLPreview html={getEffectiveHTML(page)} /></div>
    </div>
  )
  if (page.operation === 'create') return <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#af52de', background: '#faf5ff', flexShrink: 0 }}>新增页面（{formatPageLabel(page.page_number, page.page_title)}）</div><HTMLPreview html={getEffectiveHTML(page)} /></div>
  if (page.operation === 'delete') return <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff3b30', background: '#fff5f5', flexShrink: 0 }}>此页面将被删除</div><HTMLPreview html={page.original_html} /></div>

  // edit决策双版本对比
  if (page.decision === 'edit' && page.final_html && page.final_html.length > 10) {
    const beforeHTML = page.generated_html || page.original_html || ''
    return (
      <div style={{ display: 'flex', height: '100%' }}>
        <div style={{ flex: '1 1 50%', minWidth: 0, maxWidth: '50%', borderRight: '2px solid rgba(0,0,0,0.08)', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0, display: 'flex', alignItems: 'center' }}>编辑前<div style={{ flex: 1 }} /><button onClick={() => navigator.clipboard.writeText(beforeHTML || '')} style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid rgba(0,0,0,0.1)', background: '#fff', fontSize: 11, color: '#8e8e93', cursor: 'pointer' }}>复制</button></div><HTMLPreview html={beforeHTML} /></div>
        <div style={{ flex: '1 1 50%', minWidth: 0, maxWidth: '50%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff9500', background: '#fff8f0', flexShrink: 0, display: 'flex', alignItems: 'center', gap: 6 }}><Edit3 size={12} /> 编辑后<div style={{ flex: 1 }} /><button onClick={() => navigator.clipboard.writeText(page.final_html || '')} style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid rgba(0,0,0,0.1)', background: '#fff', fontSize: 11, color: '#ff9500', cursor: 'pointer' }}>复制</button></div><HTMLPreview html={page.final_html} /></div>
      </div>
    )
  }

  return <HTMLPreview html={page.original_html || page.final_html} />
}

// ==================== MergeCompareView 合并页对比 ====================

function MergeCompareView({ page, mergeSourceTab, onTabChange }: { page: GeneratedPageFull; mergeSourceTab: number; onTabChange: (idx: number) => void }) {
  const sourceNums = parseMergeSources(page.merge_sources)
  if (sourceNums.length === 0) return (
    <div style={{ display: 'flex', height: '100%' }}>
      <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0 }}>原版</div><HTMLPreview html={page.original_html} /></div>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff9500', background: '#fff8f0', flexShrink: 0 }}>合并结果</div><HTMLPreview html={getEffectiveHTML(page)} /></div>
    </div>
  )
  return (
    <div style={{ display: 'flex', height: '100%' }}>
      <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}>
        <div style={{ display: 'flex', gap: 0, background: '#f9f9f9', borderBottom: '1px solid rgba(0,0,0,0.04)', flexShrink: 0 }}>
          {sourceNums.map((pn, idx) => <button key={pn} onClick={() => onTabChange(idx)} style={{ padding: '8px 16px', border: 'none', fontSize: 12, fontWeight: 600, cursor: 'pointer', background: mergeSourceTab === idx ? '#fff' : '#f5f5f5', color: mergeSourceTab === idx ? '#ff9500' : '#8e8e93', borderBottom: mergeSourceTab === idx ? '2px solid #ff9500' : '2px solid transparent' }}>源页面 P{pn}</button>)}
          <div style={{ flex: 1, background: '#f5f5f5' }} />
        </div>
        {mergeSourceTab === 0 ? <HTMLPreview html={page.original_html} /> : <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 13, padding: 20, textAlign: 'center' }}>源页面 P{sourceNums[mergeSourceTab]} 的原始HTML暂未单独存储。</div>}
      </div>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff9500', background: '#fff8f0', flexShrink: 0 }}>合并结果（{sourceNums.map(n => 'P' + n).join(' + ')} → P{page.page_number}）</div><HTMLPreview html={getEffectiveHTML(page)} /></div>
    </div>
  )
}
