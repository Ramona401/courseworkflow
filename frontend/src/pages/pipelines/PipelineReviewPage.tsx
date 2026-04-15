/**
 * PipelineReviewPage — Pipeline审核页面（主文件）
 *
 * v69重构（编号8方案2）：
 *   - 拆分为3个文件：主文件 + ReviewPageParts + ReviewPageModals
 *   - HTML懒加载：首次只加载轻量元数据(getPagesMeta)，选中页面时按需加载HTML(getSinglePageHTML)
 *   - 内存缓存已加载的HTML，切换页面时不重复请求
 *   - 保留所有v68/v69已有功能（代码视图/双版本预览/AI快修/回滚等）
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getPipelineDetail,
  getPagesMeta,
  getSinglePageHTML,
  updatePageDecision,
  finalizePipeline,
  submitFinalize,
  confirmFinalize,
  rejectFinalize,
  type PipelineDetailResponse,
  type GeneratedPageFull,
  type UpdatePageDecisionRequest,
} from '@/api/pipelines'
import { ArrowLeft, RefreshCw, Send, ShieldCheck, RotateCcw } from 'lucide-react'
import {
  sortPagesLogically, isInputElement, getEffectiveHTML,
  requestTrueFullscreen, exitTrueFullscreen, isTrueFullscreen,
  PageSidebar, PageToolbar, PagePreview,
} from './components/ReviewPageParts'
import { FullscreenPreview, RejectDialog } from './components/ReviewPageModals'

// ==================== 主组件 ====================

export default function PipelineReviewPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()

  const isAdmin = user?.role === 'admin'
  const isSuperReviewer = user?.role === 'admin' || user?.role === 'senior_operator'
  const canOperate = user?.role === 'admin' || user?.role === 'operator' || user?.role === 'senior_operator'

  // 状态管理
  const [pipeline, setPipeline] = useState<PipelineDetailResponse | null>(null)
  const [pages, setPages] = useState<GeneratedPageFull[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [selectedIdx, setSelectedIdx] = useState(0)
  const [mergeSourceTab, setMergeSourceTab] = useState(0)
  const [finalizing, setFinalizing] = useState(false)
  const [deciding, setDeciding] = useState(false)
  const [editingHTML, setEditingHTML] = useState(false)
  const [editContent, setEditContent] = useState('')
  const [fullscreen, setFullscreen] = useState(false)
  const [reasonExpanded, setReasonExpanded] = useState(true)
  const [showRejectDialog, setShowRejectDialog] = useState(false)
  const [rejectReason, setRejectReason] = useState('')
  const [codeViewMode, setCodeViewMode] = useState(false)

  // v69新增：HTML懒加载状态
  // htmlLoadedSet 记录已加载HTML的页码集合，避免重复请求
  const htmlLoadedSetRef = useRef<Set<number>>(new Set())
  const [currentPageHtmlLoaded, setCurrentPageHtmlLoaded] = useState(false)

  const sortedPages = sortPagesLogically(pages)
  const currentPage = sortedPages[selectedIdx] || null
  const totalPages = sortedPages.length
  const decidedPages = sortedPages.filter(p => p.decision !== 'pending').length
  const allDecided = totalPages > 0 && decidedPages === totalPages
  const isPendingFinalize = pipeline?.status === 'pending_finalize'
  const isReviewQueue = pipeline?.status === 'review_queue' || pipeline?.status === 'needs_human'

  const opStats: Record<string, number> = {}
  for (const p of sortedPages) { opStats[p.operation || 'keep'] = (opStats[p.operation || 'keep'] || 0) + 1 }

  /** 加载Pipeline详情和页面元数据（不含HTML） */
  const loadData = useCallback(async () => {
    if (!id) return
    setLoading(true); setError('')
    htmlLoadedSetRef.current.clear()
    try {
      const [pd, pg] = await Promise.all([getPipelineDetail(id), getPagesMeta(id)])
      setPipeline(pd); setPages(pg || [])
      setCurrentPageHtmlLoaded(false)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) { setError(e.message || '加载失败') }
    setLoading(false)
  }, [id])

   
  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { loadData() }, [loadData])

  /**
   * v69新增：选中页面时按需加载HTML
   * 使用缓存避免重复请求——已加载过HTML的页面直接标记为已加载
   */
  useEffect(() => {
     
    // eslint-disable-next-line react-hooks/set-state-in-effect
    if (!id || !currentPage) { setCurrentPageHtmlLoaded(false); return }
    const pn = currentPage.page_number

    // 检查是否已经加载过HTML
    if (htmlLoadedSetRef.current.has(pn)) {
      setCurrentPageHtmlLoaded(true)
      return
    }

    // 按需加载单页HTML
    setCurrentPageHtmlLoaded(false)
    let cancelled = false
    getSinglePageHTML(id, pn).then(fullPage => {
      if (cancelled) return
      // 将加载到的HTML数据合并回pages状态
      setPages(prev => prev.map(p =>
        p.page_number === pn
          ? { ...p, original_html: fullPage.original_html, generated_html: fullPage.generated_html, final_html: fullPage.final_html, change_reason: fullPage.change_reason || p.change_reason }
          : p
      ))
      htmlLoadedSetRef.current.add(pn)
      setCurrentPageHtmlLoaded(true)
    }).catch(() => {
      // 加载失败时也标记为已加载（显示空内容），避免无限重试
      if (!cancelled) setCurrentPageHtmlLoaded(true)
    })
    return () => { cancelled = true }
  // eslint-disable-next-line
  }, [id, currentPage?.page_number])

  /** 全局键盘事件 */
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && fullscreen) { setFullscreen(false); return }
      if (isInputElement(e.target)) return
      if (fullscreen) {
        if (e.key === 'ArrowLeft' && selectedIdx > 0) { setSelectedIdx(i => i - 1); setMergeSourceTab(0) }
        if (e.key === 'ArrowRight' && selectedIdx < sortedPages.length - 1) { setSelectedIdx(i => i + 1); setMergeSourceTab(0) }
      }
    }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [fullscreen, selectedIdx, sortedPages.length])

  const selectPage = (idx: number) => { setSelectedIdx(idx); setEditingHTML(false); setMergeSourceTab(0); setEditContent(''); setCodeViewMode(false) }

  const handleDecision = async (decision: 'approve' | 'reject' | 'edit', finalHTML?: string) => {
    if (!id || !currentPage) return
    setDeciding(true)
    try {
      const req: UpdatePageDecisionRequest = { decision }
      if (decision === 'edit' && finalHTML) req.final_html = finalHTML
      await updatePageDecision(id, currentPage.page_number, req)
      setPages(prev => prev.map(p => p.page_number === currentPage.page_number ? { ...p, decision, final_html: finalHTML || p.final_html } : p))
       
      setEditingHTML(false)
      const nextIdx = sortedPages.findIndex((p, i) => i > selectedIdx && p.decision === 'pending')
      if (nextIdx >= 0) setSelectedIdx(nextIdx)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) { alert('决策失败: ' + (e.message || '未知错误')) }
     
    setDeciding(false)
   
  }
  

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleSubmitFinalize = async () => { if (!id || !allDecided) return; if (!confirm('确认提交定稿申请？')) return; setFinalizing(true); try { await submitFinalize(id); alert('已提交！'); navigate('/workflow/pipelines') } catch (e: any) { alert('提交失败: ' + (e.message || '')) } setFinalizing(false) }
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleConfirmFinalize = async () => { if (!id) return; if (!confirm('确认定稿归档？')) return; setFinalizing(true); try { await confirmFinalize(id); alert('定稿已确认！'); navigate('/workflow/pipelines') } catch (e: any) { alert('确认失败: ' + (e.message || '')) } setFinalizing(false) }
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleRejectFinalize = async () => { if (!id) return; setFinalizing(true); try { await rejectFinalize(id, rejectReason); alert('已退回！'); setShowRejectDialog(false); navigate('/workflow/review') } catch (e: any) { alert('退回失败: ' + (e.message || '')) } setFinalizing(false) }
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleDirectFinalize = async () => { if (!id || !allDecided) return; if (!confirm('确认直接定稿？')) return; setFinalizing(true); try { await finalizePipeline(id); alert('定稿成功！'); navigate('/workflow/pipelines') } catch (e: any) { alert('定稿失败: ' + (e.message || '')) } setFinalizing(false) }

  const startEdit = () => { if (!currentPage) return; setEditContent(getEffectiveHTML(currentPage)); setEditingHTML(true); setCodeViewMode(false) }
  const enterFullscreen = () => { setFullscreen(true); requestAnimationFrame(() => { requestAnimationFrame(() => { requestTrueFullscreen() }) }) }

  const btn: React.CSSProperties = { padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6 }

  if (loading) return <div style={{ textAlign: 'center', padding: 60, color: '#8e8e93' }}>加载中...</div>
  if (error) return <div style={{ textAlign: 'center', padding: 60 }}><div style={{ color: '#ff3b30', fontSize: 14, marginBottom: 12 }}>{error}</div><button style={btn} onClick={() => navigate(-1)}><ArrowLeft size={14} /> 返回</button></div>
  if (!pipeline) return null

  return (
    <div style={{ height: 'calc(100vh - 80px)', display: 'flex', flexDirection: 'column' }}>
      {/* 顶部栏 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '12px 0', borderBottom: '1px solid rgba(0,0,0,0.06)', flexShrink: 0 }}>
        <button style={btn} onClick={() => navigate('/workflow/pipelines/' + id)}><ArrowLeft size={14} /> 返回详情</button>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 16, fontWeight: 700, color: '#1c1c1e' }}>审核: {pipeline.course_code} — {pipeline.course_name}</div>
          <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 2 }}>
            {decidedPages}/{totalPages} 页已决策
            {allDecided && <span style={{ color: '#34c759', marginLeft: 8 }}>✓ 全部完成</span>}
            {isPendingFinalize && <span style={{ color: '#ff9500', marginLeft: 8, fontWeight: 600 }}>⏳ 待超级审核员确认定稿</span>}
            {pipeline.reject_reason && isReviewQueue && <span style={{ color: '#ff3b30', marginLeft: 8, fontWeight: 500, background: 'rgba(255,59,48,0.08)', padding: '1px 8px', borderRadius: 4, fontSize: 11 }}>↩ 退回原因：{pipeline.reject_reason}</span>}
          </div>
        </div>
        <button style={btn} onClick={loadData}><RefreshCw size={14} /> 刷新</button>
        {isReviewQueue && canOperate && !isSuperReviewer && <button style={{ ...btn, background: allDecided ? '#ff9500' : '#e5e5ea', color: allDecided ? '#fff' : '#aeaeb2', border: 'none', cursor: allDecided ? 'pointer' : 'not-allowed' }} onClick={handleSubmitFinalize} disabled={!allDecided || finalizing}><Send size={14} /> {finalizing ? '提交中...' : '提交定稿'}</button>}
        {isReviewQueue && isSuperReviewer && !isAdmin && <button style={{ ...btn, background: allDecided ? '#ff9500' : '#e5e5ea', color: allDecided ? '#fff' : '#aeaeb2', border: 'none', cursor: allDecided ? 'pointer' : 'not-allowed' }} onClick={handleSubmitFinalize} disabled={!allDecided || finalizing}><Send size={14} /> {finalizing ? '提交中...' : '提交定稿'}</button>}
        {isPendingFinalize && isSuperReviewer && (<>
          <button style={{ ...btn, background: '#ff3b30', color: '#fff', border: 'none', opacity: finalizing ? 0.6 : 1 }} onClick={() => setShowRejectDialog(true)} disabled={finalizing}><RotateCcw size={14} /> 退回重审</button>
          <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: finalizing ? 0.6 : 1 }} onClick={handleConfirmFinalize} disabled={finalizing}><ShieldCheck size={14} /> {finalizing ? '处理中...' : '确认定稿'}</button>
        </>)}
        {isAdmin && (isReviewQueue || isPendingFinalize) && <button style={{ ...btn, background: allDecided ? '#007aff' : '#e5e5ea', color: allDecided ? '#fff' : '#aeaeb2', border: 'none', fontSize: 12 }} onClick={handleDirectFinalize} disabled={!allDecided || finalizing}><Send size={13} /> 直接定稿</button>}
      </div>

      {/* 主体 */}
      <div style={{ flex: 1, display: 'flex', gap: 0, overflow: 'hidden', marginTop: 12 }}>
        <PageSidebar sortedPages={sortedPages} selectedIdx={selectedIdx} opStats={opStats} totalPages={totalPages} onSelectPage={selectPage} />
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)', borderLeft: 'none', borderRadius: '0 14px 14px 0', overflow: 'hidden' }}>
          {currentPage ? (<>
            <PageToolbar page={currentPage} btn={btn} editingHTML={editingHTML} deciding={deciding} isPendingFinalize={isPendingFinalize} isSuperReviewer={isSuperReviewer} canOperate={canOperate} reasonExpanded={reasonExpanded} codeViewMode={codeViewMode} htmlLoaded={currentPageHtmlLoaded} onCodeViewToggle={() => setCodeViewMode(!codeViewMode)} onReasonToggle={() => setReasonExpanded(!reasonExpanded)} onEnterFullscreen={enterFullscreen} onDecision={handleDecision} onStartEdit={startEdit} onSaveEdit={() => handleDecision('edit', editContent)} onCancelEdit={() => { setEditingHTML(false); setEditContent('') }} />
            {currentPage.change_reason && reasonExpanded && (
              <div style={{ padding: '12px 16px', background: '#f8f9ff', borderBottom: '1px solid rgba(0,122,255,0.1)', flexShrink: 0, maxHeight: 200, overflow: 'auto' }}>
                <div style={{ fontSize: 12, fontWeight: 600, color: '#007aff', marginBottom: 6 }}>修改理由（Translator指令）</div>
                <div style={{ fontSize: 12, color: '#3c3c43', lineHeight: 1.7, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{currentPage.change_reason}</div>
              </div>
            )}
            <div style={{ flex: 1, overflow: 'hidden' }}>
              <PagePreview page={currentPage} editingHTML={editingHTML} editContent={editContent} onEditContentChange={setEditContent} mergeSourceTab={mergeSourceTab} onMergeSourceTabChange={setMergeSourceTab} codeViewMode={codeViewMode} htmlLoaded={currentPageHtmlLoaded} />
            </div>
          </>) : <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 14 }}>请从左侧选择一个页面进行预览</div>}
        </div>
      </div>

      {/* 全屏预览 */}
      {fullscreen && currentPage && (
        <FullscreenPreview page={currentPage} pages={sortedPages} currentIdx={selectedIdx} pipelineId={id || ''}
          onNavigate={(idx) => { setSelectedIdx(idx); setMergeSourceTab(0) }}
          onClose={() => { if (isTrueFullscreen()) exitTrueFullscreen(); setFullscreen(false) }}
          onPageUpdated={(pn, html) => { setPages(prev => prev.map(p => p.page_number === pn ? { ...p, generated_html: html, final_html: html } : p)) }}
        />
      )}

      {/* 退回重审弹窗 */}
      {showRejectDialog && <RejectDialog finalizing={finalizing} rejectReason={rejectReason} onReasonChange={setRejectReason} onConfirm={handleRejectFinalize} onClose={() => !finalizing && setShowRejectDialog(false)} />}
    </div>
  )
}
