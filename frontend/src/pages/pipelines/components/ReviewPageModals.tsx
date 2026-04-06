/**
 * ReviewPageModals.tsx — 审核页弹窗和全屏预览组件集合
 *
 * v69拆分自 PipelineReviewPage.tsx（编号8方案2）
 * 包含：
 *   - AIFixModal — AI快修弹窗
 *   - ReasonModal — 修改理由弹窗
 *   - RejectDialog — 退回重审弹窗
 *   - FullscreenPreview — 全屏预览（含纯净模式+代码视图）
 *   - PureFloatingBadge — 纯净模式浮标
 */
import { useState, useEffect } from 'react'
import type { GeneratedPageFull } from '@/api/pipelines'
import { aiFixPage, aiFixPageStream, rollbackPageHTML } from '@/api/pipelines'
import {
  X, ChevronLeft, ChevronRight, FileText, Columns, Wand2, Loader, Eye, Code,
  Minimize2, RotateCcw,
} from 'lucide-react'
import {
  OP_COLORS, OP_NAMES,
  formatPageLabel, getEffectiveHTML, isVirtualPage, getVirtualPageAnchor, parseMergeSources,
  requestTrueFullscreen, exitTrueFullscreen, isTrueFullscreen,
  CodeView, HTMLPreview,
} from './ReviewPageParts'

// ==================== AI快修弹窗 ====================

export function AIFixModal({ pipelineId, pageNumber, pageTitle, loading, instruction, onInstructionChange, onSubmit, onClose, allPages, fixSummary, streamText }: {
  pipelineId: string; pageNumber: number; pageTitle: string; loading: boolean
  instruction: string; onInstructionChange: (v: string) => void
  onSubmit: (refPages: number[]) => void; onClose: () => void
  allPages: { page_number: number; page_title: string; operation: string }[]
  fixSummary: string; streamText?: string
}) {
  const [selectedRefs, setSelectedRefs] = useState<number[]>([])
  const toggleRef = (pn: number) => { setSelectedRefs(prev => prev.includes(pn) ? prev.filter(x => x !== pn) : [...prev, pn]) }

  useEffect(() => {
    const h = (e: KeyboardEvent) => { if (e.key === 'Escape' && !loading) { e.stopPropagation(); onClose() } }
    window.addEventListener('keydown', h, true)
    return () => window.removeEventListener('keydown', h, true)
  }, [onClose, loading])

  return (
    <div onClick={e => { if (e.target === e.currentTarget && !loading) onClose() }} style={{ position: 'fixed', inset: 0, zIndex: 10001, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 40 }}>
      <div style={{ background: '#fff', borderRadius: 16, maxWidth: 640, width: '100%', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        {/* 标题栏 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '16px 20px', borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
          <Wand2 size={16} color="#e65100" />
          <span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e', flex: 1 }}>AI快修 — {formatPageLabel(pageNumber, pageTitle)}. {pageTitle || '无标题'}</span>
          <button onClick={onClose} disabled={loading} style={{ padding: '4px 12px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)', background: '#f5f5f5', fontSize: 12, fontWeight: 500, color: '#3c3c43', cursor: loading ? 'not-allowed' : 'pointer', opacity: loading ? 0.5 : 1, display: 'inline-flex', alignItems: 'center', gap: 4 }}><X size={12} /> 关闭</button>
        </div>
        {/* 提示 */}
        <div style={{ padding: '12px 20px', background: '#fff8e1', borderBottom: '1px solid rgba(230,81,0,0.08)', fontSize: 12, color: '#e65100', lineHeight: 1.6 }}>输入修改指令，AI将基于当前页面HTML进行修复。</div>
        {/* 输入区 */}
        <div style={{ padding: '16px 20px' }}>
          <textarea value={instruction} onChange={e => onInstructionChange(e.target.value)}
            onKeyDown={e => { if (['ArrowLeft','ArrowRight','ArrowUp','ArrowDown'].includes(e.key)) e.stopPropagation() }}
            placeholder="请输入修复指令..." disabled={loading}
            style={{ width: '100%', minHeight: 120, border: '1px solid rgba(0,0,0,0.1)', borderRadius: 10, padding: 12, fontSize: 13, lineHeight: 1.6, fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif', resize: 'vertical', outline: 'none', boxSizing: 'border-box', background: loading ? '#f9f9f9' : '#fff' }}
          />
        </div>
        {/* 参考页面选择 */}
        <div style={{ padding: '0 20px 12px', maxHeight: 160, overflow: 'auto' }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: '#3c3c43', marginBottom: 6 }}>选择参考页面（可选，AI将参考这些页面的风格和格式）</div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
            {allPages.filter(p => p.page_number !== pageNumber).map(p => (
              <button key={p.page_number} onClick={() => toggleRef(p.page_number)} disabled={loading}
                style={{ padding: '3px 10px', borderRadius: 6, fontSize: 11, fontWeight: 500, border: selectedRefs.includes(p.page_number) ? '2px solid #007aff' : '1px solid rgba(0,0,0,0.1)', background: selectedRefs.includes(p.page_number) ? '#e8f2ff' : '#f9f9f9', color: selectedRefs.includes(p.page_number) ? '#007aff' : '#6e6e73', cursor: loading ? 'not-allowed' : 'pointer' }}>
                P{String(p.page_number).padStart(2,'0')} {p.page_title ? p.page_title.slice(0,12) : ''}
              </button>
            ))}
          </div>
          {selectedRefs.length > 0 && <div style={{ fontSize: 11, color: '#007aff', marginTop: 4 }}>已选 {selectedRefs.length} 个参考页面</div>}
        </div>
        {/* v69新增：AI流式输出实时预览区（AI正在生成时显示） */}
        {loading && streamText && (
          <div style={{ margin: '0 20px 12px', padding: '10px 14px', background: '#f8f9fa', border: '1px solid rgba(0,0,0,0.08)', borderRadius: 10, maxHeight: 200, overflow: 'auto' }}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#007aff', marginBottom: 4, display: 'flex', alignItems: 'center', gap: 6 }}>
              <Loader size={12} style={{ animation: 'spin 1s linear infinite' }} /> AI正在生成...
            </div>
            <pre style={{ fontSize: 11, color: '#3c3c43', lineHeight: 1.5, whiteSpace: 'pre-wrap', wordBreak: 'break-word', margin: 0, fontFamily: 'Monaco, Consolas, "Courier New", monospace', maxHeight: 150, overflow: 'auto' }}>{streamText.slice(-2000)}</pre>
          </div>
        )}
        {/* 修改说明 */}
        {fixSummary && (
          <div style={{ margin: '0 20px 12px', padding: '10px 14px', background: '#f0faf0', border: '1px solid rgba(52,199,89,0.2)', borderRadius: 10 }}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#34c759', marginBottom: 4 }}>✅ AI修改说明</div>
            <div style={{ fontSize: 12, color: '#1c1c1e', lineHeight: 1.7, whiteSpace: 'pre-wrap' }}>{fixSummary}</div>
          </div>
        )}
        {/* 操作按钮 */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, padding: '12px 20px', borderTop: '1px solid rgba(0,0,0,0.06)' }}>
          <button onClick={onClose} disabled={loading} style={{ padding: '8px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 13, fontWeight: 500, color: '#3c3c43', cursor: loading ? 'not-allowed' : 'pointer', opacity: loading ? 0.5 : 1 }}>取消</button>
          <button onClick={() => onSubmit(selectedRefs)} disabled={loading || !instruction.trim()} style={{ padding: '8px 24px', borderRadius: 10, border: 'none', background: loading || !instruction.trim() ? '#e5e5ea' : '#e65100', color: loading || !instruction.trim() ? '#aeaeb2' : '#fff', fontSize: 13, fontWeight: 600, cursor: loading || !instruction.trim() ? 'not-allowed' : 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6 }}>
            {loading ? <><Loader size={14} style={{ animation: 'spin 1s linear infinite' }} /> AI修复中...</> : <><Wand2 size={14} /> 执行修复</>}
          </button>
        </div>
      </div>
      {loading && <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`}</style>}
    </div>
  )
}

// ==================== 修改理由弹窗 ====================

export function ReasonModal({ reason, onClose }: { reason: string; onClose: () => void }) {
  useEffect(() => {
    const h = (e: KeyboardEvent) => { if (e.key === 'Escape') { e.stopPropagation(); onClose() } }
    window.addEventListener('keydown', h, true)
    return () => window.removeEventListener('keydown', h, true)
  }, [onClose])
  return (
    <div onClick={e => { if (e.target === e.currentTarget) onClose() }} style={{ position: 'fixed', inset: 0, zIndex: 10001, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 40 }}>
      <div style={{ background: '#fff', borderRadius: 16, maxWidth: 680, width: '100%', maxHeight: '70vh', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '16px 20px', borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
          <FileText size={16} color="#007aff" /><span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e', flex: 1 }}>修改理由（Translator指令）</span>
          <button onClick={onClose} style={{ padding: '4px 12px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)', background: '#f5f5f5', fontSize: 12, fontWeight: 500, color: '#3c3c43', cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 4 }}><X size={12} /> 关闭</button>
        </div>
        <div style={{ padding: '20px 24px', overflow: 'auto', flex: 1, fontSize: 13, color: '#3c3c43', lineHeight: 1.8, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{reason}</div>
      </div>
    </div>
  )
}

// ==================== 退回重审弹窗 ====================

export function RejectDialog({ finalizing, rejectReason, onReasonChange, onConfirm, onClose }: {
  finalizing: boolean; rejectReason: string; onReasonChange: (v: string) => void; onConfirm: () => void; onClose: () => void
}) {
  return (
    <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', zIndex: 10000, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 40 }} onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{ background: '#fff', borderRadius: 16, padding: 0, maxWidth: 480, width: '100%', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '16px 20px', borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
          <RotateCcw size={16} color="#ff3b30" /><span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e', flex: 1 }}>退回重审</span>
          <button onClick={onClose} style={{ background: '#f2f2f7', border: 'none', borderRadius: '50%', width: 28, height: 28, display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}><X size={14} color="#8e8e93" /></button>
        </div>
        <div style={{ padding: '16px 20px' }}>
          <div style={{ fontSize: 13, color: '#3c3c43', marginBottom: 10 }}>将退回给原审核员重新审核，退回原因将显示在审核员的审核页面顶部：</div>
          <textarea value={rejectReason} onChange={e => onReasonChange(e.target.value)} placeholder="请说明退回原因..." style={{ width: '100%', minHeight: 100, border: '1px solid rgba(0,0,0,0.1)', borderRadius: 10, padding: 12, fontSize: 13, lineHeight: 1.6, resize: 'vertical', outline: 'none', boxSizing: 'border-box', fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif' }} />
        </div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, padding: '12px 20px', borderTop: '1px solid rgba(0,0,0,0.06)' }}>
          <button onClick={onClose} style={{ padding: '8px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer', color: '#3c3c43' }}>取消</button>
          <button onClick={onConfirm} disabled={finalizing} style={{ padding: '8px 24px', borderRadius: 10, border: 'none', background: finalizing ? '#e5e5ea' : '#ff3b30', color: finalizing ? '#aeaeb2' : '#fff', fontSize: 13, fontWeight: 600, cursor: finalizing ? 'not-allowed' : 'pointer' }}>{finalizing ? '处理中...' : '确认退回'}</button>
        </div>
      </div>
    </div>
  )
}

// ==================== 全屏预览 ====================

export function FullscreenPreview({ page, pages, currentIdx, pipelineId, onNavigate, onClose, onPageUpdated }: {
  page: GeneratedPageFull; pages: GeneratedPageFull[]; currentIdx: number; pipelineId: string
  onNavigate: (idx: number) => void; onClose: () => void; onPageUpdated: (pn: number, html: string) => void
}) {
  const [fsMode, setFsMode] = useState<'generated' | 'original' | 'compare'>('generated')
  const [showReasonModal, setShowReasonModal] = useState(false)
  const [showAIFixModal, setShowAIFixModal] = useState(false)
  const [aiFixInstruction, setAIFixInstruction] = useState('')
  const [aiFixLoading, setAIFixLoading] = useState(false)
  const [aiFixSummary, setAIFixSummary] = useState('')
  // v69新增：流式AI输出文本（实时展示AI正在生成的内容）
  const [aiStreamText, setAiStreamText] = useState('')
  const [rollbackLoading, setRollbackLoading] = useState(false)
  const [pureMode, setPureMode] = useState(false)
  const [fsCodeView, setFsCodeView] = useState(false)

  const hasPrev = currentIdx > 0
  const hasNext = currentIdx < pages.length - 1
  const hasOriginal = !!(page.original_html && page.original_html.length > 10)
  const hasGenerated = !!((page.final_html && page.final_html.length > 10) || (page.generated_html && page.generated_html.length > 10))
  const isCreatePage = page.operation === 'create'
  const hasBoth = hasOriginal && hasGenerated && !isCreatePage

  useEffect(() => {
    if (isCreatePage) setFsMode('generated')
    else if (page.operation === 'keep' || page.operation === 'delete') setFsMode('original')
    else setFsMode('generated')
    setShowReasonModal(false); setShowAIFixModal(false); setAIFixInstruction(''); setPureMode(false); setFsCodeView(false)
  }, [page.page_number, page.operation, isCreatePage])

  useEffect(() => { const h = () => { if (!isTrueFullscreen()) onClose() }; document.addEventListener('fullscreenchange', h); document.addEventListener('webkitfullscreenchange', h); return () => { document.removeEventListener('fullscreenchange', h); document.removeEventListener('webkitfullscreenchange', h) } }, [onClose])
  useEffect(() => { return () => { if (isTrueFullscreen()) exitTrueFullscreen() } }, [])
  useEffect(() => { if (!pureMode) return; const h = (e: KeyboardEvent) => { if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); setPureMode(false) } }; window.addEventListener('keydown', h, true); return () => window.removeEventListener('keydown', h, true) }, [pureMode])

  const getDisplayHTML = () => {
    if (!pureMode) return ''
    if (isCreatePage) return getEffectiveHTML(page)
    if (fsMode === 'original' && hasOriginal) return page.original_html
    if (hasGenerated) return getEffectiveHTML(page)
    return page.original_html || ''
  }

  const navBtn: React.CSSProperties = { padding: '6px 10px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.1)', background: '#fff', display: 'inline-flex', alignItems: 'center', fontSize: 13, fontWeight: 500, color: '#3c3c43', cursor: 'pointer' }

  const renderContent = () => {
    if (pureMode) return fsCodeView ? <CodeView html={getDisplayHTML()} /> : <HTMLPreview html={getDisplayHTML()} />
    if (isCreatePage) return fsCodeView ? <CodeView html={getEffectiveHTML(page)} label="新增页面代码" /> : <HTMLPreview html={getEffectiveHTML(page)} />
    if (fsMode === 'compare' && hasBoth) return (
      <div style={{ display: 'flex', height: '100%' }}>
        <div style={{ flex: 1, borderRight: '2px solid rgba(0,0,0,0.08)', display: 'flex', flexDirection: 'column' }}><div style={{ padding: '8px 16px', fontSize: 13, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0, borderBottom: '1px solid rgba(0,0,0,0.04)' }}>{fsCodeView ? '原版代码' : '原版'}</div>{fsCodeView ? <CodeView html={page.original_html} /> : <HTMLPreview html={page.original_html} />}</div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}><div style={{ padding: '8px 16px', fontSize: 13, fontWeight: 600, color: '#007aff', background: '#f0f7ff', flexShrink: 0, borderBottom: '1px solid rgba(0,0,0,0.04)' }}>{fsCodeView ? '修改后代码' : '修改后'}</div>{fsCodeView ? <CodeView html={getEffectiveHTML(page)} /> : <HTMLPreview html={getEffectiveHTML(page)} />}</div>
      </div>
    )
    if (fsMode === 'original' && hasOriginal) return fsCodeView ? <CodeView html={page.original_html} label="原版代码" /> : <HTMLPreview html={page.original_html} />
    if (hasGenerated) return fsCodeView ? <CodeView html={getEffectiveHTML(page)} label="修改后代码" /> : <HTMLPreview html={getEffectiveHTML(page)} />
    if (hasOriginal) return fsCodeView ? <CodeView html={page.original_html} label="页面代码" /> : <HTMLPreview html={page.original_html} />
    return <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2' }}>（无内容）</div>
  }

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 9999, background: '#fff', display: 'flex', flexDirection: 'column' }}>
      {/* 工具栏 */}
      {!pureMode && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 20px', background: 'rgba(255,255,255,0.95)', backdropFilter: 'blur(20px)', borderBottom: '1px solid rgba(0,0,0,0.08)', flexShrink: 0 }}>
          <button onClick={() => hasPrev && onNavigate(currentIdx - 1)} style={{ ...navBtn, opacity: hasPrev ? 1 : 0.3, cursor: hasPrev ? 'pointer' : 'not-allowed' }} disabled={!hasPrev}><ChevronLeft size={16} /></button>
          <span style={{ fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px', borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2' }}>{OP_NAMES[page.operation] || page.operation}</span>
          <span style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>{formatPageLabel(page.page_number, page.page_title)}. {page.page_title || '无标题'}</span>
          <span style={{ fontSize: 12, color: '#8e8e93' }}>{currentIdx + 1}/{pages.length}</span>
          <button onClick={() => hasNext && onNavigate(currentIdx + 1)} style={{ ...navBtn, opacity: hasNext ? 1 : 0.3, cursor: hasNext ? 'pointer' : 'not-allowed' }} disabled={!hasNext}><ChevronRight size={16} /></button>
          <div style={{ flex: 1 }} />
          {hasBoth && <div style={{ display: 'flex', gap: 0, borderRadius: 8, overflow: 'hidden', border: '1px solid rgba(0,0,0,0.1)' }}>{([{ key: 'original' as const, label: '原版' }, { key: 'generated' as const, label: '修改后' }, { key: 'compare' as const, label: '对比' }]).map(item => (<button key={item.key} onClick={() => setFsMode(item.key)} style={{ padding: '6px 14px', border: 'none', fontSize: 12, fontWeight: 500, cursor: 'pointer', background: fsMode === item.key ? '#007aff' : '#fff', color: fsMode === item.key ? '#fff' : '#3c3c43' }}>{item.key === 'compare' && <Columns size={12} style={{ marginRight: 4, verticalAlign: -1 }} />}{item.label}</button>))}</div>}
          {isCreatePage && <span style={{ fontSize: 12, color: '#af52de', fontWeight: 500, fontStyle: 'italic' }}>新增页面</span>}
          <button onClick={() => setFsCodeView(!fsCodeView)} style={{ ...navBtn, padding: '6px 14px', background: fsCodeView ? '#f5f0ff' : '#fff', color: fsCodeView ? '#5856d6' : '#3c3c43', border: fsCodeView ? '1px solid rgba(88,86,214,0.3)' : '1px solid rgba(0,0,0,0.1)' }}><Code size={13} /> {fsCodeView ? '预览' : '代码'}</button>
          {page.change_reason && <button onClick={() => setShowReasonModal(true)} style={{ ...navBtn, padding: '6px 14px', background: '#f0f7ff', color: '#007aff', border: '1px solid rgba(0,122,255,0.15)' }}><FileText size={13} /> 修改理由</button>}
          <button onClick={() => setPureMode(true)} style={{ ...navBtn, padding: '6px 14px', background: '#f5f0ff', color: '#6c3ec1', border: '1px solid rgba(108,62,193,0.15)' }}><Eye size={13} /> 纯净全屏</button>
          <button onClick={() => { setShowAIFixModal(true); setAIFixInstruction(''); setAIFixSummary('') }} style={{ ...navBtn, padding: '6px 14px', background: '#fff3e0', color: '#e65100', border: '1px solid rgba(230,81,0,0.15)' }}><Wand2 size={13} /> AI快修</button>
          {(page as any).html_history_count > 0 && (
            <button onClick={async () => { if (!confirm('确认回滚到上一版本？')) return; setRollbackLoading(true); try { const resp = await rollbackPageHTML(pipelineId, page.page_number); onPageUpdated(page.page_number, resp.restored_html); alert('已回滚！剩余 ' + resp.remaining_history + ' 个历史版本') } catch (e: any) { alert('回滚失败: ' + (e?.response?.data?.message || e.message || '未知错误')) } setRollbackLoading(false) }} disabled={rollbackLoading}
              style={{ ...navBtn, padding: '6px 14px', background: '#f0f0ff', color: '#5856d6', border: '1px solid rgba(88,86,214,0.15)' }}>
              <RotateCcw size={13} /> {rollbackLoading ? '回滚中...' : '撤销 (' + (page as any).html_history_count + ')'}
            </button>
          )}
          <button onClick={onClose} style={{ ...navBtn, padding: '6px 14px' }}><Minimize2 size={14} /> 退出</button>
        </div>
      )}
      {/* 内容区 */}
      <div style={{ flex: 1, overflow: 'hidden', background: fsCodeView && !pureMode ? '#1e1e1e' : '#fff', position: 'relative' }}>
        {renderContent()}
        {hasPrev && <div onClick={() => onNavigate(currentIdx - 1)} onMouseEnter={e => (e.currentTarget.style.opacity = '1')} onMouseLeave={e => (e.currentTarget.style.opacity = pureMode ? '0' : '1')} style={{ position: 'absolute', left: 0, top: 0, bottom: 0, width: 60, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(to right, rgba(0,0,0,0.05), transparent)', opacity: pureMode ? 0 : 1, transition: 'opacity 0.2s' }}><ChevronLeft size={24} color="#8e8e93" /></div>}
        {hasNext && <div onClick={() => onNavigate(currentIdx + 1)} onMouseEnter={e => (e.currentTarget.style.opacity = '1')} onMouseLeave={e => (e.currentTarget.style.opacity = pureMode ? '0' : '1')} style={{ position: 'absolute', right: 0, top: 0, bottom: 0, width: 60, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(to left, rgba(0,0,0,0.05), transparent)', opacity: pureMode ? 0 : 1, transition: 'opacity 0.2s' }}><ChevronRight size={24} color="#8e8e93" /></div>}
      </div>
      {/* 纯净模式浮标 */}
      {pureMode && <PureFloatingBadge page={page} currentIdx={currentIdx} totalPages={pages.length} onBack={() => setPureMode(false)} />}
      {/* 弹窗 */}
      {showReasonModal && page.change_reason && <ReasonModal reason={page.change_reason} onClose={() => setShowReasonModal(false)} />}
      {showAIFixModal && <AIFixModal pipelineId={pipelineId} pageNumber={page.page_number} pageTitle={page.page_title} loading={aiFixLoading} instruction={aiFixInstruction} onInstructionChange={setAIFixInstruction} allPages={pages.map(p => ({ page_number: p.page_number, page_title: p.page_title, operation: p.operation }))} fixSummary={aiFixSummary}
          streamText={aiStreamText}
        onSubmit={async (refPages) => {
            if (!aiFixInstruction.trim()) { alert('请输入修复指令'); return }
            setAIFixLoading(true); setAIFixSummary(''); setAiStreamText('')
            await aiFixPageStream(
              pipelineId, page.page_number,
              { fix_instruction: aiFixInstruction.trim(), reference_pages: refPages.length > 0 ? refPages : undefined },
              (chunk) => { setAiStreamText(prev => prev + chunk) },
              (result) => { onPageUpdated(page.page_number, result.new_html); setAIFixSummary(result.fix_summary || ''); setAIFixLoading(false); if (result.fix_summary) alert('AI快修完成！请查看修改说明。'); else alert('AI快修完成！') },
              (errMsg) => { alert('AI快修失败: ' + errMsg); setAIFixLoading(false) },
            )
          }}
        onClose={() => { if (!aiFixLoading) { setShowAIFixModal(false); setAIFixInstruction('') } }}
      />}
    </div>
  )
}

// ==================== 纯净模式浮标 ====================

function PureFloatingBadge({ page, currentIdx, totalPages, onBack }: {
  page: GeneratedPageFull; currentIdx: number; totalPages: number; onBack: () => void
}) {
  const [hovering, setHovering] = useState(false)
  return (
    <div onMouseEnter={() => setHovering(true)} onMouseLeave={() => setHovering(false)} onClick={onBack} title="返回全屏预览（ESC）"
      style={{ position: 'fixed', top: 16, right: 16, zIndex: 10000, display: 'flex', alignItems: 'center', gap: 8, background: hovering ? 'rgba(0,0,0,0.7)' : 'rgba(0,0,0,0.2)', backdropFilter: 'blur(12px)', color: '#fff', fontSize: 12, fontWeight: 500, padding: '8px 14px', borderRadius: 20, cursor: 'pointer', transition: 'all 0.25s ease', opacity: hovering ? 1 : 0.5 }}>
      <span style={{ fontSize: 11, opacity: 0.9 }}>{formatPageLabel(page.page_number, page.page_title)} · {currentIdx + 1}/{totalPages}</span>
      <span style={{ width: 1, height: 14, background: 'rgba(255,255,255,0.3)' }} />
      <Minimize2 size={13} /><span>返回</span>
    </div>
  )
}
