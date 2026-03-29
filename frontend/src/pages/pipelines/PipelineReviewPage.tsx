/**
 * Pipeline审核页面（v46：修复新增页面显示+排序逻辑）
 *
 * v46修复：
 * 1. sortPagesLogically重写：虚拟页面(create)作为独立条目插入到对应位置之后，不再合并覆盖普通页面
 * 2. create页面只显示生成的新页面，无"原版/修改后/对比"切换
 * 3. 修复新增页面之后原版错位问题（因为普通页面不再被覆盖）
 *
 * v44v4 全屏预览：
 * - 「全屏预览」按钮：进入浏览器真全屏（隐藏浏览器框架+任务栏），保留顶部工具栏
 *   工具栏含：翻页/原版修改后切换/对比/修改理由/AI快修/纯净全屏/退出
 * - 「纯净全屏」按钮（在全屏预览工具栏内）：隐藏工具栏，只留右上角半透明返回浮标+页码
 *   键盘左右翻页，ESC返回全屏预览
 * - 主预览区只有一个「全屏预览」入口按钮
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getPipelineDetail,
  getGeneratedPages,
  updatePageDecision,
  finalizePipeline,
  submitFinalize,
  confirmFinalize,
  rejectFinalize,
  type PipelineDetailResponse,
  type GeneratedPageFull,
  type UpdatePageDecisionRequest,
  aiFixPage,
} from '@/api/pipelines'
import {
  ArrowLeft, RefreshCw, Check, X, Edit3, CheckCircle, Send,
  Maximize2, Minimize2, ChevronLeft, ChevronRight, FileText, Columns, Wand2, Loader, Eye,
  ShieldCheck, RotateCcw, Monitor,
} from 'lucide-react'

// ==================== 常量 ====================

/** 新增页面(create)的虚拟页码偏移量：page_number = 原位置 + 1000 */
const CREATE_PAGE_OFFSET = 1000

/** 操作类型对应的颜色 */
const OP_COLORS: Record<string, string> = { keep: '#34c759', modify: '#007aff', create: '#af52de', merge: '#ff9500', delete: '#ff3b30' }
/** 操作类型对应的中文名 */
const OP_NAMES: Record<string, string> = { keep: '保留', modify: '修改', create: '新建', merge: '合并', delete: '删除' }
/** 决策状态对应的颜色 */
const DECISION_COLORS: Record<string, string> = { approve: '#34c759', reject: '#ff3b30', edit: '#ff9500', pending: '#c7c7cc' }
/** 决策状态对应的中文名 */
const DECISION_NAMES: Record<string, string> = { approve: '已采用', reject: '已拒绝', edit: '已编辑', pending: '待决策' }

// ==================== 工具函数 ====================

/** 解析合并来源页码数组 */
function parseMergeSources(ms: string): number[] {
  try { if (ms && ms !== 'null') return JSON.parse(ms) } catch { /* ignore */ }
  return []
}

/** 判断是否为虚拟页面（新增页面，page_number >= 1000） */
function isVirtualPage(pageNumber: number): boolean {
  return pageNumber >= CREATE_PAGE_OFFSET
}

/**
 * 获取虚拟页面对应的原始位置页码
 * 例如：1010 → 10（表示插入在P10之后），1013 → 13
 */
function getVirtualPageAnchor(page: { page_number: number; page_title: string }): number {
  // 优先从标题解析 P10-new → 10
  const m = page.page_title.match(/^P(\d{1,3})-new/)
  if (m) return parseInt(m[1], 10)
  // 兜底：page_number - 1000
  const offset = page.page_number - CREATE_PAGE_OFFSET
  return offset < 100 ? offset : offset % 100
}

/**
 * 格式化页面标签用于显示
 * 普通页面：P01, P02, ...
 * 虚拟页面：P10-new, P13-new, ...
 */
function formatPageLabel(pageNumber: number, pageTitle?: string): string {
  if (pageNumber < CREATE_PAGE_OFFSET) return `P${String(pageNumber).padStart(2, '0')}`
  if (pageTitle) {
    const m = pageTitle.match(/^P(\d{1,3})-new/)
    if (m) return `P${m[1].padStart(2, '0')}-new`
  }
  const offset = pageNumber - CREATE_PAGE_OFFSET
  return `P${String(offset < 100 ? offset : offset % 100).padStart(2, '0')}-new`
}

/**
 * v46重写：逻辑排序页面列表
 *
 * 核心修改：虚拟页面(create)作为独立条目插入到对应位置之后
 * 不再把虚拟页面合并到普通页面上（旧逻辑会导致普通页面被覆盖丢失）
 *
 * 排序规则：
 * 1. 普通页面按page_number升序排列
 * 2. 虚拟页面插入到其锚点位置(anchor)的普通页面之后
 *    例如：1010(anchor=10)插入到P10之后，1013(anchor=13)插入到P13之后
 * 3. 同一锚点的多个虚拟页面按page_number升序排列
 */
function sortPagesLogically(pages: GeneratedPageFull[]): GeneratedPageFull[] {
  // 分离普通页面和虚拟页面
  const normalPages = pages.filter(p => !isVirtualPage(p.page_number))
  const virtualPages = pages.filter(p => isVirtualPage(p.page_number))

  // 普通页面按page_number升序排列
  normalPages.sort((a, b) => a.page_number - b.page_number)

  // 按锚点位置分组虚拟页面：anchor → [虚拟页面列表]
  const virtualByAnchor = new Map<number, GeneratedPageFull[]>()
  for (const vp of virtualPages) {
    const anchor = getVirtualPageAnchor(vp)
    if (!virtualByAnchor.has(anchor)) virtualByAnchor.set(anchor, [])
    virtualByAnchor.get(anchor)!.push(vp)
  }
  // 每个锚点内的虚拟页面按page_number升序
  for (const [, vps] of virtualByAnchor) {
    vps.sort((a, b) => a.page_number - b.page_number)
  }

  // 交错合并：普通页面后面插入对应的虚拟页面
  const result: GeneratedPageFull[] = []
  for (const np of normalPages) {
    result.push(np)
    // 检查此普通页面之后是否有新增的虚拟页面
    const vpList = virtualByAnchor.get(np.page_number)
    if (vpList) {
      for (const vp of vpList) {
        result.push(vp)
      }
    }
  }

  // 兜底：如果虚拟页面的锚点没有对应的普通页面
  // （这种情况发生在Translator把新增页纳入新页码体系，导致原页码被顺延跳过）
  // 策略：找到小于锚点的最大普通页面，插入其后；若找不到则追加到末尾
  for (const [anchor, vpList] of virtualByAnchor) {
    const anchorExists = normalPages.some(np => np.page_number === anchor)
    if (!anchorExists) {
      // 找小于anchor的最大普通页面在result中的位置
      let insertAfterIdx = -1
      for (let i = result.length - 1; i >= 0; i--) {
        if (!isVirtualPage(result[i].page_number) && result[i].page_number < anchor) {
          insertAfterIdx = i
          break
        }
      }
      if (insertAfterIdx >= 0) {
        // 插入到该普通页面之后（跳过已插入的虚拟页面）
        let insertPos = insertAfterIdx + 1
        while (insertPos < result.length && isVirtualPage(result[insertPos].page_number)) {
          insertPos++
        }
        result.splice(insertPos, 0, ...vpList)
      } else {
        // 找不到更小的普通页面，追加到末尾
        for (const vp of vpList) {
          result.push(vp)
        }
      }
    }
  }

  return result
}

// ==================== 浏览器全屏API ====================

/** 请求浏览器真全屏 */
function requestTrueFullscreen() {
  const t = document.documentElement
  if (t.requestFullscreen) t.requestFullscreen().catch(() => {})
  else if ((t as any).webkitRequestFullscreen) (t as any).webkitRequestFullscreen()
}
/** 退出浏览器真全屏 */
function exitTrueFullscreen() {
  if (document.fullscreenElement) document.exitFullscreen().catch(() => {})
  else if ((document as any).webkitFullscreenElement) (document as any).webkitExitFullscreen()
}
/** 检查是否处于浏览器真全屏 */
function isTrueFullscreen(): boolean {
  return !!(document.fullscreenElement || (document as any).webkitFullscreenElement)
}

// ==================== 主组件 ====================

export default function PipelineReviewPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()

  // 角色权限判断
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

  // v46：使用重写后的sortPagesLogically，虚拟页面独立插入
  const sortedPages = sortPagesLogically(pages)
  const currentPage = sortedPages[selectedIdx] || null
  const totalPages = sortedPages.length
  const decidedPages = sortedPages.filter(p => p.decision !== 'pending').length
  const allDecided = totalPages > 0 && decidedPages === totalPages
  const isPendingFinalize = pipeline?.status === 'pending_finalize'
  const isReviewQueue = pipeline?.status === 'review_queue' || pipeline?.status === 'needs_human'

  // 统计各操作类型数量
  const opStats: Record<string, number> = {}
  for (const p of sortedPages) { opStats[p.operation || 'keep'] = (opStats[p.operation || 'keep'] || 0) + 1 }

  /** 加载Pipeline详情和页面数据 */
  const loadData = useCallback(async () => {
    if (!id) return
    setLoading(true); setError('')
    try {
      const [pd, pg] = await Promise.all([getPipelineDetail(id), getGeneratedPages(id)])
      setPipeline(pd); setPages(pg || [])
    } catch (e: any) { setError(e.message || '加载失败') }
    setLoading(false)
  }, [id])

  useEffect(() => { loadData() }, [loadData])

  /** 全局键盘事件：ESC退出全屏，左右箭头翻页 */
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && fullscreen) { setFullscreen(false); return }
      if (fullscreen) {
        if (e.key === 'ArrowLeft' && selectedIdx > 0) { setSelectedIdx(i => i - 1); setMergeSourceTab(0) }
        if (e.key === 'ArrowRight' && selectedIdx < sortedPages.length - 1) { setSelectedIdx(i => i + 1); setMergeSourceTab(0) }
      }
    }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [fullscreen, selectedIdx, sortedPages.length])

  /** 选择页面 */
  const selectPage = (idx: number) => { setSelectedIdx(idx); setEditingHTML(false); setMergeSourceTab(0); setEditContent('') }

  /** 处理页面决策（采用/拒绝/编辑） */
  const handleDecision = async (decision: 'approve' | 'reject' | 'edit', finalHTML?: string) => {
    if (!id || !currentPage) return
    setDeciding(true)
    try {
      const req: UpdatePageDecisionRequest = { decision }
      if (decision === 'edit' && finalHTML) req.final_html = finalHTML
      await updatePageDecision(id, currentPage.page_number, req)
      // 更新本地状态
      setPages(prev => prev.map(p =>
        p.page_number === currentPage.page_number ? { ...p, decision, final_html: finalHTML || p.final_html, ...(decision === 'edit' && finalHTML ? { generated_html: finalHTML } : {}) } : p
      ))
      setEditingHTML(false)
      // 自动跳转到下一个待决策页面
      const nextIdx = sortedPages.findIndex((p, i) => i > selectedIdx && p.decision === 'pending')
      if (nextIdx >= 0) setSelectedIdx(nextIdx)
    } catch (e: any) { alert('决策失败: ' + (e.message || '未知错误')) }
    setDeciding(false)
  }

  /** 提交定稿申请 */
  const handleSubmitFinalize = async () => {
    if (!id || !allDecided) return
    if (!confirm('确认提交定稿申请？将发送给超级审核员确认。')) return
    setFinalizing(true)
    try { await submitFinalize(id); alert('已提交定稿申请，等待超级审核员确认！'); navigate('/workflow/pipelines') } catch (e: any) { alert('提交失败: ' + (e.message || '未知错误')) }
    setFinalizing(false)
  }
  /** 确认定稿 */
  const handleConfirmFinalize = async () => {
    if (!id) return; if (!confirm('确认定稿归档？Pipeline将进入finalized状态，可触发验收。')) return
    setFinalizing(true)
    try { await confirmFinalize(id); alert('定稿已确认！'); navigate('/workflow/pipelines') } catch (e: any) { alert('确认失败: ' + (e.message || '未知错误')) }
    setFinalizing(false)
  }
  /** 退回重审 */
  const handleRejectFinalize = async () => {
    if (!id) return; setFinalizing(true)
    try { await rejectFinalize(id, rejectReason); alert('已退回重审！'); setShowRejectDialog(false); navigate('/workflow/review') } catch (e: any) { alert('退回失败: ' + (e.message || '未知错误')) }
    setFinalizing(false)
  }
  /** 直接定稿（admin专用，跳过二级审批） */
  const handleDirectFinalize = async () => {
    if (!id || !allDecided) return; if (!confirm('确认直接定稿归档？（跳过超级审核员确认）')) return
    setFinalizing(true)
    try { await finalizePipeline(id); alert('定稿成功！'); navigate('/workflow/pipelines') } catch (e: any) { alert('定稿失败: ' + (e.message || '未知错误')) }
    setFinalizing(false)
  }
  /** 开始编辑当前页面HTML */
  const startEdit = () => { if (!currentPage) return; setEditContent(currentPage.final_html || currentPage.generated_html || ''); setEditingHTML(true) }

  /** 进入全屏预览：先设置状态再请求浏览器真全屏 */
  const enterFullscreen = () => {
    setFullscreen(true)
    requestAnimationFrame(() => { requestAnimationFrame(() => { requestTrueFullscreen() }) })
  }

  // 通用按钮样式
  const btn: React.CSSProperties = { padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6 }

  // 加载中
  if (loading) return <div style={{ textAlign: 'center', padding: 60, color: '#8e8e93' }}>加载中...</div>
  // 错误
  if (error) return (
    <div style={{ textAlign: 'center', padding: 60 }}>
      <div style={{ color: '#ff3b30', fontSize: 14, marginBottom: 12 }}>{error}</div>
      <button style={btn} onClick={() => navigate(-1)}><ArrowLeft size={14} /> 返回</button>
    </div>
  )
  if (!pipeline) return null

  return (
    <div style={{ height: 'calc(100vh - 80px)', display: 'flex', flexDirection: 'column' }}>
      {/* ===== 顶部栏 ===== */}
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
        {/* 提交定稿按钮（operator可见） */}
        {isReviewQueue && canOperate && !isSuperReviewer && <button style={{ ...btn, background: allDecided ? '#ff9500' : '#e5e5ea', color: allDecided ? '#fff' : '#aeaeb2', border: 'none', cursor: allDecided ? 'pointer' : 'not-allowed' }} onClick={handleSubmitFinalize} disabled={!allDecided || finalizing}><Send size={14} /> {finalizing ? '提交中...' : '提交定稿'}</button>}
        {/* 提交定稿按钮（senior_operator可见） */}
        {isReviewQueue && isSuperReviewer && !isAdmin && <button style={{ ...btn, background: allDecided ? '#ff9500' : '#e5e5ea', color: allDecided ? '#fff' : '#aeaeb2', border: 'none', cursor: allDecided ? 'pointer' : 'not-allowed' }} onClick={handleSubmitFinalize} disabled={!allDecided || finalizing}><Send size={14} /> {finalizing ? '提交中...' : '提交定稿'}</button>}
        {/* 退回重审+确认定稿按钮（super_reviewer在pending_finalize时可见） */}
        {isPendingFinalize && isSuperReviewer && (<>
          <button style={{ ...btn, background: '#ff3b30', color: '#fff', border: 'none', opacity: finalizing ? 0.6 : 1 }} onClick={() => setShowRejectDialog(true)} disabled={finalizing}><RotateCcw size={14} /> 退回重审</button>
          <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: finalizing ? 0.6 : 1 }} onClick={handleConfirmFinalize} disabled={finalizing}><ShieldCheck size={14} /> {finalizing ? '处理中...' : '确认定稿'}</button>
        </>)}
        {/* 直接定稿按钮（admin专用） */}
        {isAdmin && (isReviewQueue || isPendingFinalize) && <button style={{ ...btn, background: allDecided ? '#007aff' : '#e5e5ea', color: allDecided ? '#fff' : '#aeaeb2', border: 'none', fontSize: 12 }} onClick={handleDirectFinalize} disabled={!allDecided || finalizing}><Send size={13} /> 直接定稿</button>}
      </div>

      {/* ===== 主体 ===== */}
      <div style={{ flex: 1, display: 'flex', gap: 0, overflow: 'hidden', marginTop: 12 }}>
        {/* 左侧页面列表 */}
        <PageSidebar
          sortedPages={sortedPages}
          selectedIdx={selectedIdx}
          opStats={opStats}
          totalPages={totalPages}
          onSelectPage={selectPage}
        />

        {/* 右侧预览区 */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)', borderLeft: 'none', borderRadius: '0 14px 14px 0', overflow: 'hidden' }}>
          {currentPage ? (<>
            {/* 工具栏 */}
            <PageToolbar
              page={currentPage}
              btn={btn}
              editingHTML={editingHTML}
              deciding={deciding}
              isPendingFinalize={isPendingFinalize}
              isSuperReviewer={isSuperReviewer}
              canOperate={canOperate}
              reasonExpanded={reasonExpanded}
              onReasonToggle={() => setReasonExpanded(!reasonExpanded)}
              onEnterFullscreen={enterFullscreen}
              onDecision={handleDecision}
              onStartEdit={startEdit}
              onSaveEdit={() => handleDecision('edit', editContent)}
              onCancelEdit={() => { setEditingHTML(false); setEditContent('') }}
            />
            {/* 修改理由展示区 */}
            {currentPage.change_reason && reasonExpanded && (
              <div style={{ padding: '12px 16px', background: '#f8f9ff', borderBottom: '1px solid rgba(0,122,255,0.1)', flexShrink: 0, maxHeight: 200, overflow: 'auto' }}>
                <div style={{ fontSize: 12, fontWeight: 600, color: '#007aff', marginBottom: 6 }}>修改理由（Translator指令）</div>
                <div style={{ fontSize: 12, color: '#3c3c43', lineHeight: 1.7, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{currentPage.change_reason}</div>
              </div>
            )}
            {/* 预览区：根据操作类型渲染不同视图 */}
            <div style={{ flex: 1, overflow: 'hidden' }}>
              <PagePreview
                page={currentPage}
                editingHTML={editingHTML}
                editContent={editContent}
                onEditContentChange={setEditContent}
                mergeSourceTab={mergeSourceTab}
                onMergeSourceTabChange={setMergeSourceTab}
              />
            </div>
          </>) : <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 14 }}>请从左侧选择一个页面进行预览</div>}
        </div>
      </div>

      {/* ===== 全屏预览（带工具栏+纯净子模式） ===== */}
      {fullscreen && currentPage && (
        <FullscreenPreview
          page={currentPage} pages={sortedPages} currentIdx={selectedIdx} pipelineId={id || ''}
          onNavigate={(idx) => { setSelectedIdx(idx); setMergeSourceTab(0) }}
          onClose={() => { if (isTrueFullscreen()) exitTrueFullscreen(); setFullscreen(false) }}
          onPageUpdated={(pn, html) => { setPages(prev => prev.map(p => p.page_number === pn ? { ...p, generated_html: html, final_html: html } : p)) }}
        />
      )}

      {/* ===== 退回重审弹窗 ===== */}
      {showRejectDialog && (
        <RejectDialog
          finalizing={finalizing}
          rejectReason={rejectReason}
          onReasonChange={setRejectReason}
          onConfirm={handleRejectFinalize}
          onClose={() => !finalizing && setShowRejectDialog(false)}
        />
      )}
    </div>
  )
}

// ==================== 左侧页面列表组件 ====================

function PageSidebar({ sortedPages, selectedIdx, opStats, totalPages, onSelectPage }: {
  sortedPages: GeneratedPageFull[]; selectedIdx: number; opStats: Record<string, number>; totalPages: number; onSelectPage: (idx: number) => void
}) {
  return (
    <div style={{ width: 300, flexShrink: 0, background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)', borderRadius: '14px 0 0 14px', overflow: 'auto' }}>
      {/* 列表头部：统计信息 */}
      <div style={{ padding: '14px 16px', fontSize: 13, fontWeight: 600, color: '#1c1c1e', borderBottom: '1px solid rgba(0,0,0,0.04)', position: 'sticky', top: 0, background: 'rgba(255,255,255,0.95)', backdropFilter: 'blur(20px)', zIndex: 2 }}>
        <div>课程页面 ({totalPages})</div>
        <div style={{ display: 'flex', gap: 6, marginTop: 6, flexWrap: 'wrap' }}>
          {Object.entries(opStats).map(([op, count]) => <span key={op} style={{ fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px', borderRadius: 4, background: OP_COLORS[op] || '#aeaeb2' }}>{OP_NAMES[op] || op} {count}</span>)}
        </div>
      </div>
      {/* 页面条目 */}
      {sortedPages.map((page, idx) => (
        <div key={page.page_number} onClick={() => onSelectPage(idx)} style={{ padding: '10px 16px', cursor: 'pointer', background: idx === selectedIdx ? 'rgba(0,122,255,0.08)' : 'transparent', borderBottom: '1px solid rgba(0,0,0,0.03)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 6px', borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2', flexShrink: 0 }}>{OP_NAMES[page.operation] || page.operation}</span>
            <span style={{ fontSize: 13, fontWeight: 500, color: '#1c1c1e', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
              {formatPageLabel(page.page_number, page.page_title)}. {page.page_title || '无标题'}
            </span>
          </div>
          {/* 合并页面来源提示 */}
          {page.operation === 'merge' && page.merge_sources && page.merge_sources !== 'null' && (
            <div style={{ fontSize: 10, color: '#ff9500', marginTop: 2, paddingLeft: 42 }}>合并自: {parseMergeSources(page.merge_sources).map(n => 'P' + n).join('+')}</div>
          )}
          {/* v46：虚拟页面(新增)独立显示，标注插入位置 */}
          {isVirtualPage(page.page_number) && (
            <div style={{ fontSize: 10, color: '#af52de', marginTop: 2, paddingLeft: 42 }}>新增于 P{String(getVirtualPageAnchor(page)).padStart(2, '0')} 之后</div>
          )}
          {/* 决策状态 */}
          <div style={{ fontSize: 11, marginTop: 3, paddingLeft: 42, color: DECISION_COLORS[page.decision] || '#c7c7cc', fontWeight: page.decision !== 'pending' ? 600 : 400 }}>{DECISION_NAMES[page.decision] || page.decision}</div>
        </div>
      ))}
      {sortedPages.length === 0 && <div style={{ padding: 20, textAlign: 'center', color: '#aeaeb2', fontSize: 13 }}>暂无生成页面</div>}
    </div>
  )
}

// ==================== 页面工具栏组件 ====================

function PageToolbar({ page, btn, editingHTML, deciding, isPendingFinalize, isSuperReviewer, canOperate, reasonExpanded, onReasonToggle, onEnterFullscreen, onDecision, onStartEdit, onSaveEdit, onCancelEdit }: {
  page: GeneratedPageFull; btn: React.CSSProperties; editingHTML: boolean; deciding: boolean
  isPendingFinalize: boolean; isSuperReviewer: boolean; canOperate: boolean; reasonExpanded: boolean
  onReasonToggle: () => void; onEnterFullscreen: () => void
  onDecision: (d: 'approve' | 'reject' | 'edit', html?: string) => void
  onStartEdit: () => void; onSaveEdit: () => void; onCancelEdit: () => void
}) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '10px 16px', borderBottom: '1px solid rgba(0,0,0,0.04)', flexShrink: 0, flexWrap: 'wrap' }}>
      {/* 操作类型标签 */}
      <span style={{ fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px', borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2' }}>{OP_NAMES[page.operation] || page.operation}</span>
      {/* 页面标题 */}
      <span style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>{formatPageLabel(page.page_number, page.page_title)}. {page.page_title || '无标题'}</span>
      {/* 合并来源提示 */}
      {page.operation === 'merge' && page.merge_sources && page.merge_sources !== 'null' && <span style={{ fontSize: 11, color: '#ff9500', fontWeight: 500 }}>(合并自 {parseMergeSources(page.merge_sources).map(n => 'P' + n).join(' + ')})</span>}
      {/* v46：虚拟页面提示 */}
      {isVirtualPage(page.page_number) && <span style={{ fontSize: 11, color: '#af52de', fontWeight: 500 }}>（新增页面，位于 P{String(getVirtualPageAnchor(page)).padStart(2, '0')} 之后）</span>}
      <div style={{ flex: 1 }} />
      {/* 全屏预览入口 */}
      {!editingHTML && <button onClick={onEnterFullscreen} style={{ ...btn, padding: '6px 10px', fontSize: 12 }} title="全屏预览（隐藏浏览器框架，带工具栏）"><Maximize2 size={13} /> 全屏预览</button>}
      {/* 修改理由折叠按钮 */}
      {page.change_reason && <button onClick={onReasonToggle} style={{ ...btn, padding: '6px 10px', fontSize: 12, background: reasonExpanded ? '#f0f7ff' : '#fff', color: reasonExpanded ? '#007aff' : '#3c3c43' }}><FileText size={13} /> {reasonExpanded ? '收起理由' : '修改理由'}</button>}
      {/* 决策按钮组（review_queue状态） */}
      {!editingHTML && !isPendingFinalize && canOperate && (
        <div style={{ display: 'flex', gap: 6 }}>
          <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }} onClick={() => onDecision('approve')} disabled={deciding}><Check size={14} /> 采用</button>
          <button style={{ ...btn, background: '#ff3b30', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }} onClick={() => onDecision('reject')} disabled={deciding}><X size={14} /> 拒绝</button>
          <button style={{ ...btn, background: '#ff9500', color: '#fff', border: 'none', padding: '6px 14px' }} onClick={onStartEdit}><Edit3 size={14} /> 编辑</button>
        </div>
      )}
      {/* 决策按钮组（pending_finalize状态，super_reviewer可操作） */}
      {!editingHTML && isPendingFinalize && isSuperReviewer && (
        <div style={{ display: 'flex', gap: 6 }}>
          <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }} onClick={() => onDecision('approve')} disabled={deciding}><Check size={14} /> 采用</button>
          <button style={{ ...btn, background: '#ff3b30', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }} onClick={() => onDecision('reject')} disabled={deciding}><X size={14} /> 拒绝</button>
          <button style={{ ...btn, background: '#ff9500', color: '#fff', border: 'none', padding: '6px 14px' }} onClick={onStartEdit}><Edit3 size={14} /> 编辑</button>
        </div>
      )}
      {/* 等待确认提示 */}
      {isPendingFinalize && !isSuperReviewer && <span style={{ fontSize: 12, color: '#ff9500', fontWeight: 500 }}>⏳ 待超级审核员确认</span>}
      {/* 编辑模式按钮 */}
      {editingHTML && (
        <div style={{ display: 'flex', gap: 6 }}>
          <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1 }} onClick={onSaveEdit} disabled={deciding}><CheckCircle size={14} /> 保存编辑</button>
          <button style={btn} onClick={onCancelEdit}>取消</button>
        </div>
      )}
    </div>
  )
}

// ==================== 页面预览组件 ====================

/**
 * v46：根据操作类型渲染不同的预览视图
 * - keep: 只显示原版
 * - modify: 左右对比（原版 + 修改后）
 * - create: 只显示生成的新页面（无原版）
 * - merge: 合并对比视图
 * - delete: 显示将被删除的原版
 * - 编辑模式: textarea编辑器
 */
function PagePreview({ page, editingHTML, editContent, onEditContentChange, mergeSourceTab, onMergeSourceTabChange }: {
  page: GeneratedPageFull; editingHTML: boolean; editContent: string; onEditContentChange: (v: string) => void
  mergeSourceTab: number; onMergeSourceTabChange: (idx: number) => void
}) {
  // 编辑模式
  if (editingHTML) {
    return (
      <textarea
        value={editContent}
        onChange={e => onEditContentChange(e.target.value)}
        style={{ width: '100%', height: '100%', border: 'none', outline: 'none', fontFamily: 'Monaco, Consolas, "Courier New", monospace', fontSize: 12, lineHeight: 1.6, padding: 16, resize: 'none', background: '#fafafa', color: '#1c1c1e', boxSizing: 'border-box' }}
      />
    )
  }

  // 合并页面对比视图
  if (page.operation === 'merge') {
    return <MergeCompareView page={page} mergeSourceTab={mergeSourceTab} onTabChange={onMergeSourceTabChange} />
  }

  // 修改页面：左右对比（原版 + 修改后）
  if (page.operation === 'modify') {
    return (
      <div style={{ display: 'flex', height: '100%' }}>
        <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}>
          <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0 }}>原版</div>
          <HTMLPreview html={page.original_html} />
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#007aff', background: '#f0f7ff', flexShrink: 0 }}>修改后</div>
          <HTMLPreview html={page.generated_html} />
        </div>
      </div>
    )
  }

  // v46修复：新增页面只显示生成的新页面，无原版/对比
  if (page.operation === 'create') {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#af52de', background: '#faf5ff', flexShrink: 0 }}>
          新增页面（{formatPageLabel(page.page_number, page.page_title)}）
        </div>
        <HTMLPreview html={page.generated_html} />
      </div>
    )
  }

  // 删除页面：显示将被删除的原版
  if (page.operation === 'delete') {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff3b30', background: '#fff5f5', flexShrink: 0 }}>此页面将被删除</div>
        <HTMLPreview html={page.original_html} />
      </div>
    )
  }

  // 保留页面(keep)及其他：只显示原版
  return <HTMLPreview html={page.original_html || page.final_html} />
}

// ==================== 合并页对比 ====================

function MergeCompareView({ page, mergeSourceTab, onTabChange }: { page: GeneratedPageFull; mergeSourceTab: number; onTabChange: (idx: number) => void }) {
  const sourceNums = parseMergeSources(page.merge_sources)
  if (sourceNums.length === 0) return (
    <div style={{ display: 'flex', height: '100%' }}>
      <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0 }}>原版</div><HTMLPreview html={page.original_html} /></div>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff9500', background: '#fff8f0', flexShrink: 0 }}>合并结果</div><HTMLPreview html={page.generated_html} /></div>
    </div>
  )
  return (
    <div style={{ display: 'flex', height: '100%' }}>
      <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}>
        <div style={{ display: 'flex', gap: 0, background: '#f9f9f9', borderBottom: '1px solid rgba(0,0,0,0.04)', flexShrink: 0 }}>
          {sourceNums.map((pn, idx) => <button key={pn} onClick={() => onTabChange(idx)} style={{ padding: '8px 16px', border: 'none', fontSize: 12, fontWeight: 600, cursor: 'pointer', background: mergeSourceTab === idx ? '#fff' : '#f5f5f5', color: mergeSourceTab === idx ? '#ff9500' : '#8e8e93', borderBottom: mergeSourceTab === idx ? '2px solid #ff9500' : '2px solid transparent' }}>源页面 P{pn}</button>)}
          <div style={{ flex: 1, background: '#f5f5f5' }} />
        </div>
        {mergeSourceTab === 0 ? <HTMLPreview html={page.original_html} /> : <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 13, padding: 20, textAlign: 'center' }}>源页面 P{sourceNums[mergeSourceTab]} 的原始HTML暂未单独存储。<br />目前仅第一个源页面(P{sourceNums[0]})可预览。</div>}
      </div>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}><div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff9500', background: '#fff8f0', flexShrink: 0 }}>合并结果（{sourceNums.map(n => 'P' + n).join(' + ')} → P{page.page_number}）</div><HTMLPreview html={page.generated_html} /></div>
    </div>
  )
}

// ==================== AI快修弹窗 ====================

function AIFixModal({ pipelineId, pageNumber, pageTitle, loading, instruction, onInstructionChange, onSubmit, onClose }: { pipelineId: string; pageNumber: number; pageTitle: string; loading: boolean; instruction: string; onInstructionChange: (v: string) => void; onSubmit: () => void; onClose: () => void }) {
  useEffect(() => { const h = (e: KeyboardEvent) => { if (e.key === 'Escape' && !loading) { e.stopPropagation(); onClose() } }; window.addEventListener('keydown', h, true); return () => window.removeEventListener('keydown', h, true) }, [onClose, loading])
  return (
    <div onClick={e => { if (e.target === e.currentTarget && !loading) onClose() }} style={{ position: 'fixed', inset: 0, zIndex: 10001, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 40 }}>
      <div style={{ background: '#fff', borderRadius: 16, maxWidth: 640, width: '100%', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '16px 20px', borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
          <Wand2 size={16} color="#e65100" /><span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e', flex: 1 }}>AI快修 — {formatPageLabel(pageNumber, pageTitle)}. {pageTitle || '无标题'}</span>
          <button onClick={onClose} disabled={loading} style={{ padding: '4px 12px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)', background: '#f5f5f5', fontSize: 12, fontWeight: 500, color: '#3c3c43', cursor: loading ? 'not-allowed' : 'pointer', opacity: loading ? 0.5 : 1, display: 'inline-flex', alignItems: 'center', gap: 4 }}><X size={12} /> 关闭</button>
        </div>
        <div style={{ padding: '12px 20px', background: '#fff8e1', borderBottom: '1px solid rgba(230,81,0,0.08)', fontSize: 12, color: '#e65100', lineHeight: 1.6 }}>输入修改指令，AI将基于当前页面HTML进行修复。</div>
        <div style={{ padding: '16px 20px' }}>
          <textarea value={instruction} onChange={e => onInstructionChange(e.target.value)} placeholder="请输入修复指令..." disabled={loading} style={{ width: '100%', minHeight: 120, border: '1px solid rgba(0,0,0,0.1)', borderRadius: 10, padding: 12, fontSize: 13, lineHeight: 1.6, fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif', resize: 'vertical', outline: 'none', boxSizing: 'border-box', background: loading ? '#f9f9f9' : '#fff' }} />
        </div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, padding: '12px 20px', borderTop: '1px solid rgba(0,0,0,0.06)' }}>
          <button onClick={onClose} disabled={loading} style={{ padding: '8px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 13, fontWeight: 500, color: '#3c3c43', cursor: loading ? 'not-allowed' : 'pointer', opacity: loading ? 0.5 : 1 }}>取消</button>
          <button onClick={onSubmit} disabled={loading || !instruction.trim()} style={{ padding: '8px 24px', borderRadius: 10, border: 'none', background: loading || !instruction.trim() ? '#e5e5ea' : '#e65100', color: loading || !instruction.trim() ? '#aeaeb2' : '#fff', fontSize: 13, fontWeight: 600, cursor: loading || !instruction.trim() ? 'not-allowed' : 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6 }}>
            {loading ? <><Loader size={14} style={{ animation: 'spin 1s linear infinite' }} /> AI修复中...</> : <><Wand2 size={14} /> 执行修复</>}
          </button>
        </div>
      </div>
      {loading && <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`}</style>}
    </div>
  )
}

// ==================== 修改理由弹窗 ====================

function ReasonModal({ reason, onClose }: { reason: string; onClose: () => void }) {
  useEffect(() => { const h = (e: KeyboardEvent) => { if (e.key === 'Escape') { e.stopPropagation(); onClose() } }; window.addEventListener('keydown', h, true); return () => window.removeEventListener('keydown', h, true) }, [onClose])
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

function RejectDialog({ finalizing, rejectReason, onReasonChange, onConfirm, onClose }: {
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
          <textarea value={rejectReason} onChange={e => onReasonChange(e.target.value)} placeholder="请说明退回原因，例如：P05页面内容与课程目标不符..." style={{ width: '100%', minHeight: 100, border: '1px solid rgba(0,0,0,0.1)', borderRadius: 10, padding: 12, fontSize: 13, lineHeight: 1.6, resize: 'vertical', outline: 'none', boxSizing: 'border-box', fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif' }} />
        </div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, padding: '12px 20px', borderTop: '1px solid rgba(0,0,0,0.06)' }}>
          <button onClick={onClose} style={{ padding: '8px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer', color: '#3c3c43' }}>取消</button>
          <button onClick={onConfirm} disabled={finalizing} style={{ padding: '8px 24px', borderRadius: 10, border: 'none', background: finalizing ? '#e5e5ea' : '#ff3b30', color: finalizing ? '#aeaeb2' : '#fff', fontSize: 13, fontWeight: 600, cursor: finalizing ? 'not-allowed' : 'pointer' }}>{finalizing ? '处理中...' : '确认退回'}</button>
        </div>
      </div>
    </div>
  )
}

// ==================== 全屏预览（浏览器真全屏+工具栏+纯净子模式） ====================

function FullscreenPreview({ page, pages, currentIdx, pipelineId, onNavigate, onClose, onPageUpdated }: {
  page: GeneratedPageFull; pages: GeneratedPageFull[]; currentIdx: number; pipelineId: string
  onNavigate: (idx: number) => void; onClose: () => void; onPageUpdated: (pn: number, html: string) => void
}) {
  /** 全屏模式下的视图切换：原版/修改后/对比 */
  const [fsMode, setFsMode] = useState<'generated' | 'original' | 'compare'>('generated')
  const [showReasonModal, setShowReasonModal] = useState(false)
  const [showAIFixModal, setShowAIFixModal] = useState(false)
  const [aiFixInstruction, setAIFixInstruction] = useState('')
  const [aiFixLoading, setAIFixLoading] = useState(false)
  /** 纯净模式：隐藏工具栏，只显示内容+右上角浮标 */
  const [pureMode, setPureMode] = useState(false)

  const hasPrev = currentIdx > 0
  const hasNext = currentIdx < pages.length - 1

  // v46：根据操作类型判断是否有原版和生成版
  const hasOriginal = !!(page.original_html && page.original_html.length > 10)
  const hasGenerated = !!(page.generated_html && page.generated_html.length > 10)
  // v46：create页面不显示原版/对比切换（即使original_html不为空也不显示，因为create页面不应有原版）
  const isCreatePage = page.operation === 'create'
  const hasBoth = hasOriginal && hasGenerated && !isCreatePage

  /** 切换页面时重置视图模式 */
  useEffect(() => {
    // v46：create页面和keep页面使用不同的默认模式
    if (isCreatePage) {
      setFsMode('generated')
    } else if (page.operation === 'keep' || page.operation === 'delete') {
      setFsMode('original')
    } else {
      setFsMode('generated')
    }
    setShowReasonModal(false); setShowAIFixModal(false); setAIFixInstruction(''); setPureMode(false)
  }, [page.page_number, page.operation, isCreatePage])

  /** 监听浏览器退出全屏（用户按了系统ESC） */
  useEffect(() => {
    const h = () => { if (!isTrueFullscreen()) onClose() }
    document.addEventListener('fullscreenchange', h); document.addEventListener('webkitfullscreenchange', h)
    return () => { document.removeEventListener('fullscreenchange', h); document.removeEventListener('webkitfullscreenchange', h) }
  }, [onClose])

  /** 组件卸载时退出浏览器全屏 */
  useEffect(() => { return () => { if (isTrueFullscreen()) exitTrueFullscreen() } }, [])

  /** 纯净模式下ESC返回全屏预览（而非完全退出） */
  useEffect(() => {
    if (!pureMode) return
    const h = (e: KeyboardEvent) => { if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); setPureMode(false) } }
    window.addEventListener('keydown', h, true)
    return () => window.removeEventListener('keydown', h, true)
  }, [pureMode])

  /** 获取纯净模式下应展示的HTML */
  const getDisplayHTML = () => {
    if (pureMode) {
      // v46：create页面始终显示generated_html
      if (isCreatePage) return page.generated_html || ''
      if (fsMode === 'original' && hasOriginal) return page.original_html
      if (hasGenerated) return page.generated_html
      return page.original_html || ''
    }
    return ''
  }

  const navBtn: React.CSSProperties = { padding: '6px 10px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.1)', background: '#fff', display: 'inline-flex', alignItems: 'center', fontSize: 13, fontWeight: 500, color: '#3c3c43', cursor: 'pointer' }

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 9999, background: '#fff', display: 'flex', flexDirection: 'column' }}>

      {/* 工具栏（纯净模式下隐藏） */}
      {!pureMode && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 20px', background: 'rgba(255,255,255,0.95)', backdropFilter: 'blur(20px)', borderBottom: '1px solid rgba(0,0,0,0.08)', flexShrink: 0 }}>
          {/* 翻页按钮 */}
          <button onClick={() => hasPrev && onNavigate(currentIdx - 1)} style={{ ...navBtn, opacity: hasPrev ? 1 : 0.3, cursor: hasPrev ? 'pointer' : 'not-allowed' }} disabled={!hasPrev}><ChevronLeft size={16} /></button>
          {/* 操作类型标签 */}
          <span style={{ fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px', borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2' }}>{OP_NAMES[page.operation] || page.operation}</span>
          {/* 页面标题 */}
          <span style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>{formatPageLabel(page.page_number, page.page_title)}. {page.page_title || '无标题'}</span>
          <span style={{ fontSize: 12, color: '#8e8e93' }}>{currentIdx + 1}/{pages.length}</span>
          <button onClick={() => hasNext && onNavigate(currentIdx + 1)} style={{ ...navBtn, opacity: hasNext ? 1 : 0.3, cursor: hasNext ? 'pointer' : 'not-allowed' }} disabled={!hasNext}><ChevronRight size={16} /></button>
          <div style={{ flex: 1 }} />

          {/* v46：原版/修改后/对比 切换按钮 — create页面不显示 */}
          {hasBoth && (
            <div style={{ display: 'flex', gap: 0, borderRadius: 8, overflow: 'hidden', border: '1px solid rgba(0,0,0,0.1)' }}>
              {([{ key: 'original' as const, label: '原版' }, { key: 'generated' as const, label: '修改后' }, { key: 'compare' as const, label: '对比' }]).map(item => (
                <button key={item.key} onClick={() => setFsMode(item.key)} style={{ padding: '6px 14px', border: 'none', fontSize: 12, fontWeight: 500, cursor: 'pointer', background: fsMode === item.key ? '#007aff' : '#fff', color: fsMode === item.key ? '#fff' : '#3c3c43' }}>
                  {item.key === 'compare' && <Columns size={12} style={{ marginRight: 4, verticalAlign: -1 }} />}{item.label}
                </button>
              ))}
            </div>
          )}
          {/* v46：create页面提示 */}
          {isCreatePage && <span style={{ fontSize: 12, color: '#af52de', fontWeight: 500, fontStyle: 'italic' }}>新增页面（无原版）</span>}
          {/* 非create页面的仅原版/仅生成版提示 */}
          {!isCreatePage && hasOriginal && !hasGenerated && <span style={{ fontSize: 12, color: '#8e8e93', fontStyle: 'italic' }}>仅原版</span>}
          {!isCreatePage && !hasOriginal && hasGenerated && <span style={{ fontSize: 12, color: '#8e8e93', fontStyle: 'italic' }}>仅生成版</span>}

          {/* 修改理由按钮 */}
          {page.change_reason && <button onClick={() => setShowReasonModal(true)} style={{ ...navBtn, padding: '6px 14px', background: '#f0f7ff', color: '#007aff', border: '1px solid rgba(0,122,255,0.15)' }}><FileText size={13} /> 修改理由</button>}
          {/* 纯净全屏按钮 */}
          <button onClick={() => setPureMode(true)} style={{ ...navBtn, padding: '6px 14px', background: '#f5f0ff', color: '#6c3ec1', border: '1px solid rgba(108,62,193,0.15)' }} title="纯净全屏（隐藏工具栏，只看内容）">
            <Eye size={13} /> 纯净全屏
          </button>
          {/* AI快修按钮 */}
          <button onClick={() => { setShowAIFixModal(true); setAIFixInstruction('') }} style={{ ...navBtn, padding: '6px 14px', background: '#fff3e0', color: '#e65100', border: '1px solid rgba(230,81,0,0.15)' }}><Wand2 size={13} /> AI快修</button>
          {/* 退出按钮 */}
          <button onClick={onClose} style={{ ...navBtn, padding: '6px 14px' }}><Minimize2 size={14} /> 退出</button>
        </div>
      )}

      {/* 预览内容区 */}
      <div style={{ flex: 1, overflow: 'hidden', background: '#fff', position: 'relative' }}>
        {pureMode ? (
          /* 纯净模式：只显示单个HTML */
          <HTMLPreview html={getDisplayHTML()} />
        ) : isCreatePage ? (
          /* v46：create页面全屏预览 — 只显示生成版，无对比 */
          <HTMLPreview html={page.generated_html} />
        ) : fsMode === 'compare' && hasBoth ? (
          /* 对比模式：左右并排 */
          <div style={{ display: 'flex', height: '100%' }}>
            <div style={{ flex: 1, borderRight: '2px solid rgba(0,0,0,0.08)', display: 'flex', flexDirection: 'column' }}>
              <div style={{ padding: '8px 16px', fontSize: 13, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0, borderBottom: '1px solid rgba(0,0,0,0.04)' }}>原版</div>
              <HTMLPreview html={page.original_html} />
            </div>
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
              <div style={{ padding: '8px 16px', fontSize: 13, fontWeight: 600, color: '#007aff', background: '#f0f7ff', flexShrink: 0, borderBottom: '1px solid rgba(0,0,0,0.04)' }}>修改后</div>
              <HTMLPreview html={page.generated_html} />
            </div>
          </div>
        ) : fsMode === 'original' && hasOriginal ? (
          <HTMLPreview html={page.original_html} />
        ) : hasGenerated ? (
          <HTMLPreview html={page.generated_html} />
        ) : hasOriginal ? (
          <HTMLPreview html={page.original_html} />
        ) : (
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2' }}>（无内容）</div>
        )}

        {/* 左右翻页热区 */}
        {hasPrev && (
          <div onClick={() => onNavigate(currentIdx - 1)}
            onMouseEnter={e => (e.currentTarget.style.opacity = '1')}
            onMouseLeave={e => (e.currentTarget.style.opacity = pureMode ? '0' : '1')}
            style={{ position: 'absolute', left: 0, top: 0, bottom: 0, width: 60, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(to right, rgba(0,0,0,0.05), transparent)', opacity: pureMode ? 0 : 1, transition: 'opacity 0.2s' }}>
            <ChevronLeft size={24} color="#8e8e93" />
          </div>
        )}
        {hasNext && (
          <div onClick={() => onNavigate(currentIdx + 1)}
            onMouseEnter={e => (e.currentTarget.style.opacity = '1')}
            onMouseLeave={e => (e.currentTarget.style.opacity = pureMode ? '0' : '1')}
            style={{ position: 'absolute', right: 0, top: 0, bottom: 0, width: 60, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(to left, rgba(0,0,0,0.05), transparent)', opacity: pureMode ? 0 : 1, transition: 'opacity 0.2s' }}>
            <ChevronRight size={24} color="#8e8e93" />
          </div>
        )}
      </div>

      {/* 纯净模式：右上角浮标（页码+返回） */}
      {pureMode && (
        <PureFloatingBadge
          page={page} currentIdx={currentIdx} totalPages={pages.length}
          onBack={() => setPureMode(false)}
        />
      )}

      {/* 弹窗层 */}
      {showReasonModal && page.change_reason && <ReasonModal reason={page.change_reason} onClose={() => setShowReasonModal(false)} />}
      {showAIFixModal && (
        <AIFixModal pipelineId={pipelineId} pageNumber={page.page_number} pageTitle={page.page_title}
          loading={aiFixLoading} instruction={aiFixInstruction} onInstructionChange={setAIFixInstruction}
          onSubmit={async () => {
            if (!aiFixInstruction.trim()) { alert('请输入修复指令'); return }
            setAIFixLoading(true)
            try {
              const resp = await aiFixPage(pipelineId, page.page_number, { fix_instruction: aiFixInstruction.trim() })
              onPageUpdated(page.page_number, resp.new_html)
              alert('AI快修完成！新HTML已更新（' + resp.html_length + '字符）'); setShowAIFixModal(false); setAIFixInstruction('')
            } catch (e: any) { alert('AI快修失败: ' + (e?.response?.data?.message || e.message || '未知错误')) }
            setAIFixLoading(false)
          }}
          onClose={() => { if (!aiFixLoading) { setShowAIFixModal(false); setAIFixInstruction('') } }}
        />
      )}
    </div>
  )
}

// ==================== 纯净模式右上角浮标 ====================

function PureFloatingBadge({ page, currentIdx, totalPages, onBack }: {
  page: GeneratedPageFull; currentIdx: number; totalPages: number; onBack: () => void
}) {
  const [hovering, setHovering] = useState(false)
  return (
    <div
      onMouseEnter={() => setHovering(true)}
      onMouseLeave={() => setHovering(false)}
      onClick={onBack}
      title="返回全屏预览（ESC）"
      style={{
        position: 'fixed', top: 16, right: 16, zIndex: 10000,
        display: 'flex', alignItems: 'center', gap: 8,
        background: hovering ? 'rgba(0,0,0,0.7)' : 'rgba(0,0,0,0.2)',
        backdropFilter: 'blur(12px)',
        color: '#fff', fontSize: 12, fontWeight: 500,
        padding: '8px 14px', borderRadius: 20,
        cursor: 'pointer', transition: 'all 0.25s ease',
        opacity: hovering ? 1 : 0.5,
      }}
    >
      <span style={{ fontSize: 11, opacity: 0.9 }}>{formatPageLabel(page.page_number, page.page_title)} · {currentIdx + 1}/{totalPages}</span>
      <span style={{ width: 1, height: 14, background: 'rgba(255,255,255,0.3)' }} />
      <Minimize2 size={13} />
      <span>返回</span>
    </div>
  )
}

// ==================== HTML预览组件 ====================

function HTMLPreview({ html }: { html: string }) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const doc = `<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><style>body{margin:0;padding:16px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;font-size:14px;line-height:1.6;color:#1c1c1e;word-wrap:break-word}img{max-width:100%;height:auto}table{border-collapse:collapse;width:100%}th,td{border:1px solid #e5e5ea;padding:8px 12px;text-align:left}th{background:#f5f5f7;font-weight:600}</style></head><body>${html || '<p style="color:#aeaeb2;text-align:center;padding:40px 0">(空内容)</p>'}</body></html>`
  if (!html && html !== '') return <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 13 }}>（无HTML内容）</div>
  return <iframe ref={iframeRef} srcDoc={doc} sandbox="allow-same-origin allow-scripts" style={{ flex: 1, width: '100%', height: '100%', border: 'none', background: '#fff' }} title="HTML预览" />
}
