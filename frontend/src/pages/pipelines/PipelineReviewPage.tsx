/**
 * Pipeline审核页面（P4.5-E 重构版 v3.5 | P7：二级审批流程 | Phase8修复 | v35排序优化v2）
 *
 * v35排序优化v2：
 * - 新增页面（create操作，page_number>=1000）按逻辑位置插入到对应原始页码之后
 *   排序后审核顺序：P01(保留) → P02(修改) → P02-new(新增) → P03(保留)
 * - 显示标签保持数据原始含义，不重新编号（避免标签与实际内容脱节）
 *   page_number=4 → P04, page_number=1004 → P04-new
 * - 修复 formatPageLabel 对双位数页码的还原错误
 *
 * Phase8修复：
 * - FP-03：HTMLPreview 中 iframe 移除 allow-scripts，防止AI生成HTML中的JS在审核员浏览器执行
 * - FP-04：selectPage 函数切换页面时同步清空 editContent
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
  Maximize2, Minimize2, ChevronLeft, ChevronRight, FileText, Columns, Wand2, Loader, Eye, EyeOff,
  ShieldCheck, RotateCcw,
} from 'lucide-react'

// ==================== 常量 ====================

/** 虚拟页码偏移量（与后端 createPageOffset=1000 保持一致） */
const CREATE_PAGE_OFFSET = 1000

const OP_COLORS: Record<string, string> = {
  keep: '#34c759', modify: '#007aff', create: '#af52de',
  merge: '#ff9500', delete: '#ff3b30',
}
const OP_NAMES: Record<string, string> = {
  keep: '保留', modify: '修改', create: '新建', merge: '合并', delete: '删除',
}
const DECISION_COLORS: Record<string, string> = {
  approve: '#34c759', reject: '#ff3b30', edit: '#ff9500', pending: '#c7c7cc',
}
const DECISION_NAMES: Record<string, string> = {
  approve: '已采用', reject: '已拒绝', edit: '已编辑', pending: '待决策',
}

// ==================== 工具函数 ====================

function parseMergeSources(ms: string): number[] {
  try {
    if (ms && ms !== 'null') return JSON.parse(ms)
  } catch { /* ignore */ }
  return []
}

/**
 * 从虚拟页码还原原始页码（v35新增）
 *
 * 后端规则：virtualPageNum = originalPage + 1000 + (counter-1)*10
 * 最可靠的方式是从 page_title 提取（格式："P04-new 标题内容"）
 * 降级方案：直接用 (page_number - 1000)，counter=1 时准确
 */
function getOriginalPageNumber(page: { page_number: number; page_title: string }): number {
  if (page.page_number < CREATE_PAGE_OFFSET) {
    return page.page_number
  }
  // 从标题提取原始页码：标题格式为 "P04-new 标题内容"
  const titleMatch = page.page_title.match(/^P(\d{1,3})-new/)
  if (titleMatch) {
    return parseInt(titleMatch[1], 10)
  }
  // 降级方案：直接用 (page_number - 1000)，对于counter=1时是准确的
  const offset = page.page_number - CREATE_PAGE_OFFSET
  if (offset < 100) {
    return offset
  }
  return offset % 100
}

/**
 * 格式化页码标签（v35修复）
 *
 * page_number < 1000 → "P04"（原有页面，保持原始页码）
 * page_number >= 1000 → "P04-new"（新增页面，从标题或数学还原原始页码）
 *
 * v35修复：原版使用 (pageNumber - 1000) % 10 对双位数页码会出错
 * 例如 page_number=1012 原版显示 P02-new（错误），应显示 P12-new
 *
 * v35v2：不再重新编号，标签直接反映数据库中的原始含义
 * 这样标签和实际HTML内容始终一致，不会出现"标签说P05但内容是P04"的问题
 */
function formatPageLabel(pageNumber: number, pageTitle?: string): string {
  if (pageNumber < CREATE_PAGE_OFFSET) {
    return `P${String(pageNumber).padStart(2, '0')}`
  }
  // 优先从标题提取原始页码
  if (pageTitle) {
    const m = pageTitle.match(/^P(\d{1,3})-new/)
    if (m) {
      return `P${m[1].padStart(2, '0')}-new`
    }
  }
  // 降级：数学还原
  const offset = pageNumber - CREATE_PAGE_OFFSET
  const origPage = offset < 100 ? offset : (offset % 100)
  return `P${String(origPage).padStart(2, '0')}-new`
}

/**
 * 智能排序页面列表（v35新增）
 *
 * 排序规则：
 * 1. 按原始页码从小到大排列
 * 2. 同一原始页码位置：新增页面（create）排在原有页面之后
 *    因为Translator已重排页码，新增页面是"在此页码之后插入"的语义
 * 3. 同一位置有多个新增页面时，按 page_number 从小到大排列
 *
 * 排序键设计：
 * - 原有页面（page_number < 1000）：sortKey = page_number * 1000 + 500
 * - 新增页面（page_number >= 1000）：sortKey = origPage * 1000 + subOrder
 *   600 + subOrder < 1000 保证新增页面排在对应原始页码之后、下一个页码之前
 *
 * 示例（G1-03实际场景，P04之后有新增）：
 *   P03(keep, pn=3)         → sortKey = 3500
 *   P04(keep, pn=4)         → sortKey = 4500
 *   P04-new(create, pn=1004) → sortKey = 4600
 *   P05(keep, pn=5)         → sortKey = 5500
 */
function sortPagesLogically(pages: GeneratedPageFull[]): GeneratedPageFull[] {
  // v35v4：Translator已经重编了所有页码（P01-P20按新顺序排列）
  // 虚拟页码（>=1000）是parsePageOps对create操作创建的副本
  // 如果同一原始页码同时存在普通版本和虚拟版本，需要合并：
  //   - 虚拟版本有AI生成的HTML（generated_html），普通版本可能被错误标记为keep
  //   - 用虚拟版本的内容替换普通版本，但保持普通版本的page_number（用于正确排序）
  
  // 1. 分离普通页面和虚拟页面
  const normalPages = pages.filter(p => p.page_number < CREATE_PAGE_OFFSET)
  const virtualPages = pages.filter(p => p.page_number >= CREATE_PAGE_OFFSET)
  
  // 2. 建立虚拟页面的映射：原始页码 → 虚拟页面数据
  const virtualMap = new Map<number, GeneratedPageFull>()
  for (const vp of virtualPages) {
    const origPage = getOriginalPageNumber(vp)
    virtualMap.set(origPage, vp)
  }
  
  // 3. 合并：如果普通页面对应位置有虚拟页面，用虚拟页面的内容替换
  const merged = normalPages.map(np => {
    const vp = virtualMap.get(np.page_number)
    if (vp) {
      // 用虚拟页面的内容覆盖，但保持普通页面的page_number用于排序
      // 这样P04位置显示的是create操作的AI生成内容
      return {
        ...np,
        operation: vp.operation,           // create
        generated_html: vp.generated_html, // AI生成的新页面HTML
        final_html: vp.final_html,         // 定稿HTML
        page_title: vp.page_title.replace(/^P\d{1,3}-new\s*/, ''), // 去掉"P04-new "前缀
        change_reason: vp.change_reason || np.change_reason,
        // 保留原始的 page_number, original_html, decision, lesson_id 等
      }
    }
    return np
  })
  
  // 4. 按page_number排序
  return merged.sort((a, b) => a.page_number - b.page_number)
}

/**
 * 计算单个页面的排序键（v35新增）
 */
function computeSortKey(page: { page_number: number; page_title: string }): number {
  if (page.page_number < CREATE_PAGE_OFFSET) {
    // 原有页面：原始页码 * 1000 + 500
    return page.page_number * 1000 + 500
  }
  // 新增页面：先还原原始页码
  const origPage = getOriginalPageNumber(page)
  // 计算子排序（同一位置多个新增页面的顺序）
  const subOrder = page.page_number - origPage - CREATE_PAGE_OFFSET
  // 排序键：原始页码 * 1000 + 600 + subOrder
  // 600 > 500（原有页面的偏移），所以新增页面排在对应原始页码的原有页面之后
  // 600 + subOrder < 1000（下一个原始页码的起始），保证不会越界
  return origPage * 1000 + 600 + subOrder
}

// ==================== 主组件 ====================

export default function PipelineReviewPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()

  const isAdmin = user?.role === 'admin'
  const isSuperReviewer = user?.role === 'admin' || user?.role === 'senior_operator'
  const canOperate = user?.role === 'admin' || user?.role === 'operator' || user?.role === 'senior_operator'

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

  /** v35：排序后的页面列表（新增页面插入到正确位置） */
  const sortedPages = sortPagesLogically(pages)

  const currentPage = sortedPages[selectedIdx] || null
  const totalPages = sortedPages.length
  const decidedPages = sortedPages.filter(p => p.decision !== 'pending').length
  const allDecided = totalPages > 0 && decidedPages === totalPages

  const isPendingFinalize = pipeline?.status === 'pending_finalize'
  const isReviewQueue = pipeline?.status === 'review_queue' || pipeline?.status === 'needs_human'

  const opStats: Record<string, number> = {}
  for (const p of sortedPages) {
    const op = p.operation || 'keep'
    opStats[op] = (opStats[op] || 0) + 1
  }

  const loadData = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError('')
    try {
      const [pd, pg] = await Promise.all([getPipelineDetail(id), getGeneratedPages(id)])
      setPipeline(pd)
      setPages(pg || [])
    } catch (e: any) {
      setError(e.message || '加载失败')
    }
    setLoading(false)
  }, [id])

  useEffect(() => { loadData() }, [loadData])

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

  /**
   * 切换页面
   * 修复FP-04：同时清空 editContent，防止切换页面后打开编辑仍显示上一页内容
   */
  const selectPage = (idx: number) => {
    setSelectedIdx(idx)
    setEditingHTML(false)
    setMergeSourceTab(0)
    setEditContent('')
  }

  const handleDecision = async (decision: 'approve' | 'reject' | 'edit', finalHTML?: string) => {
    if (!id || !currentPage) return
    setDeciding(true)
    try {
      const req: UpdatePageDecisionRequest = { decision }
      if (decision === 'edit' && finalHTML) req.final_html = finalHTML
      await updatePageDecision(id, currentPage.page_number, req)
      setPages(prev => prev.map(p =>
        p.page_number === currentPage.page_number ? { ...p, decision, final_html: finalHTML || p.final_html } : p
      ))
      setEditingHTML(false)
      // v35：在排序后的列表中查找下一个待决策的页面
      const nextIdx = sortedPages.findIndex((p, i) => i > selectedIdx && p.decision === 'pending')
      if (nextIdx >= 0) setSelectedIdx(nextIdx)
    } catch (e: any) { alert('决策失败: ' + (e.message || '未知错误')) }
    setDeciding(false)
  }

  const handleSubmitFinalize = async () => {
    if (!id || !allDecided) return
    if (!confirm('确认提交定稿申请？将发送给超级审核员确认。')) return
    setFinalizing(true)
    try {
      await submitFinalize(id)
      alert('已提交定稿申请，等待超级审核员确认！')
      navigate('/pipelines')
    } catch (e: any) { alert('提交失败: ' + (e.message || '未知错误')) }
    setFinalizing(false)
  }

  const handleConfirmFinalize = async () => {
    if (!id) return
    if (!confirm('确认定稿归档？Pipeline将进入finalized状态，可触发验收。')) return
    setFinalizing(true)
    try {
      await confirmFinalize(id)
      alert('定稿已确认！')
      navigate('/pipelines')
    } catch (e: any) { alert('确认失败: ' + (e.message || '未知错误')) }
    setFinalizing(false)
  }

  const handleRejectFinalize = async () => {
    if (!id) return
    setFinalizing(true)
    try {
      await rejectFinalize(id, rejectReason)
      alert('已退回重审！')
      setShowRejectDialog(false)
      navigate('/review')
    } catch (e: any) { alert('退回失败: ' + (e.message || '未知错误')) }
    setFinalizing(false)
  }

  const handleDirectFinalize = async () => {
    if (!id || !allDecided) return
    if (!confirm('确认直接定稿归档？（跳过超级审核员确认）')) return
    setFinalizing(true)
    try {
      await finalizePipeline(id)
      alert('定稿成功！')
      navigate('/pipelines')
    } catch (e: any) { alert('定稿失败: ' + (e.message || '未知错误')) }
    setFinalizing(false)
  }

  const startEdit = () => {
    if (!currentPage) return
    setEditContent(currentPage.final_html || currentPage.generated_html || '')
    setEditingHTML(true)
  }

  const btn: React.CSSProperties = {
    padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 6,
  }

  if (loading) return <div style={{ textAlign: 'center', padding: 60, color: '#8e8e93' }}>加载中...</div>
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
      <div style={{
        display: 'flex', alignItems: 'center', gap: 12, padding: '12px 0',
        borderBottom: '1px solid rgba(0,0,0,0.06)', flexShrink: 0,
      }}>
        <button style={btn} onClick={() => navigate('/pipelines/' + id)}>
          <ArrowLeft size={14} /> 返回详情
        </button>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 16, fontWeight: 700, color: '#1c1c1e' }}>
            审核: {pipeline.course_code} — {pipeline.course_name}
          </div>
          <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 2 }}>
            {decidedPages}/{totalPages} 页已决策
            {allDecided && <span style={{ color: '#34c759', marginLeft: 8 }}>✓ 全部完成</span>}
            {isPendingFinalize && (
              <span style={{ color: '#ff9500', marginLeft: 8, fontWeight: 600 }}>
                ⏳ 待超级审核员确认定稿
              </span>
            )}
            {pipeline.reject_reason && isReviewQueue && (
              <span style={{
                color: '#ff3b30', marginLeft: 8, fontWeight: 500,
                background: 'rgba(255,59,48,0.08)', padding: '1px 8px',
                borderRadius: 4, fontSize: 11,
              }}>
                ↩ 退回原因：{pipeline.reject_reason}
              </span>
            )}
          </div>
        </div>
        <button style={btn} onClick={loadData}><RefreshCw size={14} /> 刷新</button>

        {/* 场景1：审核队列中，普通操作员完成决策后显示「提交定稿」 */}
        {isReviewQueue && canOperate && !isSuperReviewer && (
          <button
            style={{
              ...btn,
              background: allDecided ? '#ff9500' : '#e5e5ea',
              color: allDecided ? '#fff' : '#aeaeb2',
              border: 'none',
              cursor: allDecided ? 'pointer' : 'not-allowed',
            }}
            onClick={handleSubmitFinalize}
            disabled={!allDecided || finalizing}
            title="提交给超级审核员确认定稿"
          >
            <Send size={14} /> {finalizing ? '提交中...' : '提交定稿'}
          </button>
        )}

        {/* 场景2：审核队列中，超级审核员完成决策后提交定稿申请 */}
        {isReviewQueue && isSuperReviewer && !isAdmin && (
          <button
            style={{
              ...btn,
              background: allDecided ? '#ff9500' : '#e5e5ea',
              color: allDecided ? '#fff' : '#aeaeb2',
              border: 'none',
              cursor: allDecided ? 'pointer' : 'not-allowed',
            }}
            onClick={handleSubmitFinalize}
            disabled={!allDecided || finalizing}
            title="提交定稿申请，等待二次确认"
          >
            <Send size={14} /> {finalizing ? '提交中...' : '提交定稿'}
          </button>
        )}

        {/* 场景3：pending_finalize 状态，超级审核员显示确认/退回按钮 */}
        {isPendingFinalize && isSuperReviewer && (
          <>
            <button
              style={{
                ...btn,
                background: '#ff3b30', color: '#fff', border: 'none',
                cursor: finalizing ? 'not-allowed' : 'pointer',
                opacity: finalizing ? 0.6 : 1,
              }}
              onClick={() => setShowRejectDialog(true)}
              disabled={finalizing}
              title="退回给审核员重新审核"
            >
              <RotateCcw size={14} /> 退回重审
            </button>
            <button
              style={{
                ...btn,
                background: '#34c759', color: '#fff', border: 'none',
                cursor: finalizing ? 'not-allowed' : 'pointer',
                opacity: finalizing ? 0.6 : 1,
              }}
              onClick={handleConfirmFinalize}
              disabled={finalizing}
              title="确认定稿，进入finalized状态"
            >
              <ShieldCheck size={14} /> {finalizing ? '处理中...' : '确认定稿'}
            </button>
          </>
        )}

        {/* 场景4：admin 可直接定稿（跳过二级审批） */}
        {isAdmin && (isReviewQueue || isPendingFinalize) && (
          <button
            style={{
              ...btn,
              background: allDecided ? '#007aff' : '#e5e5ea',
              color: allDecided ? '#fff' : '#aeaeb2',
              border: 'none',
              cursor: allDecided ? 'pointer' : 'not-allowed',
              fontSize: 12,
            }}
            onClick={handleDirectFinalize}
            disabled={!allDecided || finalizing}
            title="admin直接定稿（跳过二级审批）"
          >
            <Send size={13} /> 直接定稿
          </button>
        )}
      </div>

      {/* ===== 主体 ===== */}
      <div style={{ flex: 1, display: 'flex', gap: 0, overflow: 'hidden', marginTop: 12 }}>

        {/* ===== 左侧列表 ===== */}
        <div style={{
          width: 300, flexShrink: 0, background: 'rgba(255,255,255,0.72)',
          backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)',
          borderRadius: '14px 0 0 14px', overflow: 'auto',
        }}>
          <div style={{
            padding: '14px 16px', fontSize: 13, fontWeight: 600, color: '#1c1c1e',
            borderBottom: '1px solid rgba(0,0,0,0.04)', position: 'sticky', top: 0,
            background: 'rgba(255,255,255,0.95)', backdropFilter: 'blur(20px)', zIndex: 2,
          }}>
            <div>课程页面 ({totalPages})</div>
            <div style={{ display: 'flex', gap: 6, marginTop: 6, flexWrap: 'wrap' }}>
              {Object.entries(opStats).map(([op, count]) => (
                <span key={op} style={{
                  fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px',
                  borderRadius: 4, background: OP_COLORS[op] || '#aeaeb2',
                }}>{OP_NAMES[op] || op} {count}</span>
              ))}
            </div>
          </div>

          {sortedPages.map((page, idx) => (
            <div
              key={page.page_number}
              onClick={() => selectPage(idx)}
              style={{
                padding: '10px 16px', cursor: 'pointer',
                background: idx === selectedIdx ? 'rgba(0,122,255,0.08)' : 'transparent',
                borderBottom: '1px solid rgba(0,0,0,0.03)',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{
                  fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 6px',
                  borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2', flexShrink: 0,
                }}>{OP_NAMES[page.operation] || page.operation}</span>
                <span style={{
                  fontSize: 13, fontWeight: 500, color: '#1c1c1e',
                  overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1,
                }}>
                  {formatPageLabel(page.page_number, page.page_title)}. {page.page_title || '无标题'}
                </span>
              </div>
              {page.operation === 'merge' && page.merge_sources && page.merge_sources !== 'null' && (
                <div style={{ fontSize: 10, color: '#ff9500', marginTop: 2, paddingLeft: 42 }}>
                  合并自: {parseMergeSources(page.merge_sources).map(n => 'P' + n).join('+')}
                </div>
              )}
              {page.page_number >= CREATE_PAGE_OFFSET && (
                <div style={{ fontSize: 10, color: '#af52de', marginTop: 2, paddingLeft: 42 }}>
                  新增于 P{String(getOriginalPageNumber(page)).padStart(2, '0')} 之后
                </div>
              )}
              <div style={{
                fontSize: 11, marginTop: 3, paddingLeft: 42,
                color: DECISION_COLORS[page.decision] || '#c7c7cc',
                fontWeight: page.decision !== 'pending' ? 600 : 400,
              }}>{DECISION_NAMES[page.decision] || page.decision}</div>
            </div>
          ))}
          {sortedPages.length === 0 && (
            <div style={{ padding: 20, textAlign: 'center', color: '#aeaeb2', fontSize: 13 }}>暂无生成页面</div>
          )}
        </div>

        {/* ===== 右侧预览 ===== */}
        <div style={{
          flex: 1, display: 'flex', flexDirection: 'column',
          background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
          border: '1px solid rgba(0,0,0,0.06)', borderLeft: 'none',
          borderRadius: '0 14px 14px 0', overflow: 'hidden',
        }}>
          {currentPage ? (
            <>
              {/* 工具栏 */}
              <div style={{
                display: 'flex', alignItems: 'center', gap: 8, padding: '10px 16px',
                borderBottom: '1px solid rgba(0,0,0,0.04)', flexShrink: 0, flexWrap: 'wrap',
              }}>
                <span style={{
                  fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px',
                  borderRadius: 4, background: OP_COLORS[currentPage.operation] || '#aeaeb2',
                }}>{OP_NAMES[currentPage.operation] || currentPage.operation}</span>
                <span style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>
                  {formatPageLabel(currentPage.page_number, currentPage.page_title)}. {currentPage.page_title || '无标题'}
                </span>
                {currentPage.operation === 'merge' && currentPage.merge_sources && currentPage.merge_sources !== 'null' && (
                  <span style={{ fontSize: 11, color: '#ff9500', fontWeight: 500 }}>
                    (合并自 {parseMergeSources(currentPage.merge_sources).map(n => 'P' + n).join(' + ')})
                  </span>
                )}
                {currentPage.page_number >= CREATE_PAGE_OFFSET && (
                  <span style={{ fontSize: 11, color: '#af52de', fontWeight: 500 }}>
                    （新增页面，位于 P{String(getOriginalPageNumber(currentPage)).padStart(2, '0')} 之后）
                  </span>
                )}
                <div style={{ flex: 1 }} />

                {!editingHTML && (
                  <button onClick={() => setFullscreen(true)} style={{ ...btn, padding: '6px 10px', fontSize: 12 }}>
                    <Maximize2 size={13} /> 全屏
                  </button>
                )}
                {currentPage.change_reason && (
                  <button onClick={() => setReasonExpanded(!reasonExpanded)} style={{
                    ...btn, padding: '6px 10px', fontSize: 12,
                    background: reasonExpanded ? '#f0f7ff' : '#fff',
                    color: reasonExpanded ? '#007aff' : '#3c3c43',
                  }}><FileText size={13} /> {reasonExpanded ? '收起理由' : '修改理由'}</button>
                )}

                {!editingHTML && !isPendingFinalize && canOperate && (
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }}
                      onClick={() => handleDecision('approve')} disabled={deciding}><Check size={14} /> 采用</button>
                    <button style={{ ...btn, background: '#ff3b30', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }}
                      onClick={() => handleDecision('reject')} disabled={deciding}><X size={14} /> 拒绝</button>
                    <button style={{ ...btn, background: '#ff9500', color: '#fff', border: 'none', padding: '6px 14px' }}
                      onClick={startEdit}><Edit3 size={14} /> 编辑</button>
                  </div>
                )}
                {!editingHTML && isPendingFinalize && isSuperReviewer && (
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }}
                      onClick={() => handleDecision('approve')} disabled={deciding}><Check size={14} /> 采用</button>
                    <button style={{ ...btn, background: '#ff3b30', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }}
                      onClick={() => handleDecision('reject')} disabled={deciding}><X size={14} /> 拒绝</button>
                    <button style={{ ...btn, background: '#ff9500', color: '#fff', border: 'none', padding: '6px 14px' }}
                      onClick={startEdit}><Edit3 size={14} /> 编辑</button>
                  </div>
                )}
                {isPendingFinalize && !isSuperReviewer && (
                  <span style={{ fontSize: 12, color: '#ff9500', fontWeight: 500 }}>
                    ⏳ 待超级审核员确认
                  </span>
                )}

                {editingHTML && (
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1 }}
                      onClick={() => handleDecision('edit', editContent)} disabled={deciding}><CheckCircle size={14} /> 保存编辑</button>
                    <button style={btn} onClick={() => { setEditingHTML(false); setEditContent('') }}>取消</button>
                  </div>
                )}
              </div>

              {/* 修改理由 */}
              {currentPage.change_reason && reasonExpanded && (
                <div style={{
                  padding: '12px 16px', background: '#f8f9ff',
                  borderBottom: '1px solid rgba(0,122,255,0.1)', flexShrink: 0,
                  maxHeight: 200, overflow: 'auto',
                }}>
                  <div style={{ fontSize: 12, fontWeight: 600, color: '#007aff', marginBottom: 6 }}>
                    修改理由（Translator指令）
                  </div>
                  <div style={{
                    fontSize: 12, color: '#3c3c43', lineHeight: 1.7,
                    whiteSpace: 'pre-wrap', wordBreak: 'break-word',
                  }}>{currentPage.change_reason}</div>
                </div>
              )}

              {/* 预览区 */}
              <div style={{ flex: 1, overflow: 'hidden' }}>
                {editingHTML ? (
                  <textarea value={editContent} onChange={e => setEditContent(e.target.value)} style={{
                    width: '100%', height: '100%', border: 'none', outline: 'none',
                    fontFamily: 'Monaco, Consolas, "Courier New", monospace',
                    fontSize: 12, lineHeight: 1.6, padding: 16, resize: 'none',
                    background: '#fafafa', color: '#1c1c1e', boxSizing: 'border-box',
                  }} />
                ) : currentPage.operation === 'merge' ? (
                  <MergeCompareView page={currentPage} mergeSourceTab={mergeSourceTab} onTabChange={setMergeSourceTab} />
                ) : currentPage.operation === 'modify' ? (
                  <div style={{ display: 'flex', height: '100%' }}>
                    <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}>
                      <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0 }}>原版</div>
                      <HTMLPreview html={currentPage.original_html} />
                    </div>
                    <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
                      <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#007aff', background: '#f0f7ff', flexShrink: 0 }}>修改后</div>
                      <HTMLPreview html={currentPage.generated_html} />
                    </div>
                  </div>
                ) : currentPage.operation === 'create' ? (
                  <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
                    <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#af52de', background: '#faf5ff', flexShrink: 0 }}>
                      新增页面（{formatPageLabel(currentPage.page_number, currentPage.page_title)}，无原版）
                    </div>
                    <HTMLPreview html={currentPage.generated_html} />
                  </div>
                ) : currentPage.operation === 'delete' ? (
                  <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
                    <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff3b30', background: '#fff5f5', flexShrink: 0 }}>此页面将被删除</div>
                    <HTMLPreview html={currentPage.original_html} />
                  </div>
                ) : (
                  <HTMLPreview html={currentPage.original_html || currentPage.final_html} />
                )}
              </div>
            </>
          ) : (
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 14 }}>
              请从左侧选择一个页面进行预览
            </div>
          )}
        </div>
      </div>

      {/* ===== 全屏预览 ===== */}
      {fullscreen && currentPage && (
        <FullscreenPreview
          page={currentPage} pages={sortedPages} currentIdx={selectedIdx}
          pipelineId={id || ''}
          onNavigate={(idx) => { setSelectedIdx(idx); setMergeSourceTab(0) }}
          onClose={() => setFullscreen(false)}
          onPageUpdated={(pageNumber, newHTML) => {
            setPages(prev => prev.map(p =>
              p.page_number === pageNumber ? { ...p, generated_html: newHTML, final_html: newHTML } : p
            ))
          }}
        />
      )}

      {/* ===== 退回重审弹窗 ===== */}
      {showRejectDialog && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
          zIndex: 10000, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 40,
        }} onClick={e => { if (e.target === e.currentTarget && !finalizing) setShowRejectDialog(false) }}>
          <div style={{
            background: '#fff', borderRadius: 16, padding: 0, maxWidth: 480, width: '100%',
            boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
          }}>
            <div style={{
              display: 'flex', alignItems: 'center', gap: 10, padding: '16px 20px',
              borderBottom: '1px solid rgba(0,0,0,0.06)',
            }}>
              <RotateCcw size={16} color="#ff3b30" />
              <span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e', flex: 1 }}>退回重审</span>
              <button onClick={() => !finalizing && setShowRejectDialog(false)} style={{
                background: '#f2f2f7', border: 'none', borderRadius: '50%', width: 28, height: 28,
                display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer',
              }}><X size={14} color="#8e8e93" /></button>
            </div>
            <div style={{ padding: '16px 20px' }}>
              <div style={{ fontSize: 13, color: '#3c3c43', marginBottom: 10 }}>
                将退回给原审核员重新审核，退回原因将显示在审核员的审核页面顶部：
              </div>
              <textarea
                value={rejectReason}
                onChange={e => setRejectReason(e.target.value)}
                placeholder="请说明退回原因，例如：P05页面内容与课程目标不符..."
                style={{
                  width: '100%', minHeight: 100, border: '1px solid rgba(0,0,0,0.1)',
                  borderRadius: 10, padding: 12, fontSize: 13, lineHeight: 1.6,
                  resize: 'vertical', outline: 'none', boxSizing: 'border-box',
                  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                }}
              />
            </div>
            <div style={{
              display: 'flex', justifyContent: 'flex-end', gap: 8,
              padding: '12px 20px', borderTop: '1px solid rgba(0,0,0,0.06)',
            }}>
              <button onClick={() => !finalizing && setShowRejectDialog(false)} style={{
                padding: '8px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
                background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer', color: '#3c3c43',
              }}>取消</button>
              <button onClick={handleRejectFinalize} disabled={finalizing} style={{
                padding: '8px 24px', borderRadius: 10, border: 'none',
                background: finalizing ? '#e5e5ea' : '#ff3b30',
                color: finalizing ? '#aeaeb2' : '#fff',
                fontSize: 13, fontWeight: 600,
                cursor: finalizing ? 'not-allowed' : 'pointer',
              }}>
                {finalizing ? '处理中...' : '确认退回'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// ==================== 合并页对比 ====================

function MergeCompareView({ page, mergeSourceTab, onTabChange }: {
  page: GeneratedPageFull; mergeSourceTab: number; onTabChange: (idx: number) => void
}) {
  const sourceNums = parseMergeSources(page.merge_sources)
  if (sourceNums.length === 0) {
    return (
      <div style={{ display: 'flex', height: '100%' }}>
        <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}>
          <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0 }}>原版（第一个源页面）</div>
          <HTMLPreview html={page.original_html} />
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff9500', background: '#fff8f0', flexShrink: 0 }}>合并结果</div>
          <HTMLPreview html={page.generated_html} />
        </div>
      </div>
    )
  }
  return (
    <div style={{ display: 'flex', height: '100%' }}>
      <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}>
        <div style={{ display: 'flex', gap: 0, background: '#f9f9f9', borderBottom: '1px solid rgba(0,0,0,0.04)', flexShrink: 0 }}>
          {sourceNums.map((pn, idx) => (
            <button key={pn} onClick={() => onTabChange(idx)} style={{
              padding: '8px 16px', border: 'none', fontSize: 12, fontWeight: 600, cursor: 'pointer',
              background: mergeSourceTab === idx ? '#fff' : '#f5f5f5',
              color: mergeSourceTab === idx ? '#ff9500' : '#8e8e93',
              borderBottom: mergeSourceTab === idx ? '2px solid #ff9500' : '2px solid transparent',
            }}>源页面 P{pn}</button>
          ))}
          <div style={{ flex: 1, background: '#f5f5f5' }} />
        </div>
        {mergeSourceTab === 0 ? (
          <HTMLPreview html={page.original_html} />
        ) : (
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 13, padding: 20, textAlign: 'center' }}>
            源页面 P{sourceNums[mergeSourceTab]} 的原始HTML暂未单独存储。<br />目前仅第一个源页面(P{sourceNums[0]})可预览。
          </div>
        )}
      </div>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff9500', background: '#fff8f0', flexShrink: 0 }}>
          合并结果（{sourceNums.map(n => 'P' + n).join(' + ')} → P{page.page_number}）
        </div>
        <HTMLPreview html={page.generated_html} />
      </div>
    </div>
  )
}

// ==================== AI快修弹窗 ====================

function AIFixModal({ pipelineId, pageNumber, pageTitle, loading, instruction, onInstructionChange, onSubmit, onClose }: {
  pipelineId: string; pageNumber: number; pageTitle: string
  loading: boolean; instruction: string
  onInstructionChange: (v: string) => void
  onSubmit: () => void; onClose: () => void
}) {
  useEffect(() => {
    const stopPropagation = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !loading) { e.stopPropagation(); onClose() }
    }
    window.addEventListener('keydown', stopPropagation, true)
    return () => window.removeEventListener('keydown', stopPropagation, true)
  }, [onClose, loading])

  return (
    <div onClick={(e) => { if (e.target === e.currentTarget && !loading) onClose() }}
      style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        zIndex: 10001, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 40,
      }}>
      <div style={{
        background: '#fff', borderRadius: 16, padding: 0, maxWidth: 640, width: '100%',
        display: 'flex', flexDirection: 'column',
        boxShadow: '0 20px 60px rgba(0,0,0,0.2), 0 0 0 1px rgba(0,0,0,0.05)',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '16px 20px', borderBottom: '1px solid rgba(0,0,0,0.06)', flexShrink: 0 }}>
          <Wand2 size={16} color="#e65100" />
          <span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e', flex: 1 }}>
            AI快修 — {formatPageLabel(pageNumber, pageTitle)}. {pageTitle || '无标题'}
          </span>
          <button onClick={onClose} disabled={loading} style={{
            padding: '4px 12px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)',
            background: '#f5f5f5', fontSize: 12, fontWeight: 500, color: '#3c3c43',
            cursor: loading ? 'not-allowed' : 'pointer', opacity: loading ? 0.5 : 1,
            display: 'inline-flex', alignItems: 'center', gap: 4,
          }}><X size={12} /> 关闭</button>
        </div>
        <div style={{ padding: '12px 20px', background: '#fff8e1', borderBottom: '1px solid rgba(230,81,0,0.08)', fontSize: 12, color: '#e65100', lineHeight: 1.6 }}>
          输入修改指令，AI将基于当前页面HTML进行修复。
        </div>
        <div style={{ padding: '16px 20px' }}>
          <textarea value={instruction} onChange={(e) => onInstructionChange(e.target.value)}
            placeholder="请输入修复指令..." disabled={loading}
            style={{
              width: '100%', minHeight: 120, border: '1px solid rgba(0,0,0,0.1)',
              borderRadius: 10, padding: 12, fontSize: 13, lineHeight: 1.6,
              fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
              resize: 'vertical', outline: 'none', boxSizing: 'border-box',
              background: loading ? '#f9f9f9' : '#fff', color: '#1c1c1e',
            }} />
        </div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, padding: '12px 20px', borderTop: '1px solid rgba(0,0,0,0.06)' }}>
          <button onClick={onClose} disabled={loading} style={{
            padding: '8px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
            background: '#fff', fontSize: 13, fontWeight: 500, color: '#3c3c43',
            cursor: loading ? 'not-allowed' : 'pointer', opacity: loading ? 0.5 : 1,
          }}>取消</button>
          <button onClick={onSubmit} disabled={loading || !instruction.trim()} style={{
            padding: '8px 24px', borderRadius: 10, border: 'none',
            background: loading || !instruction.trim() ? '#e5e5ea' : '#e65100',
            color: loading || !instruction.trim() ? '#aeaeb2' : '#fff',
            fontSize: 13, fontWeight: 600,
            cursor: loading || !instruction.trim() ? 'not-allowed' : 'pointer',
            display: 'inline-flex', alignItems: 'center', gap: 6,
          }}>
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
  useEffect(() => {
    const stopPropagation = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { e.stopPropagation(); onClose() }
    }
    window.addEventListener('keydown', stopPropagation, true)
    return () => window.removeEventListener('keydown', stopPropagation, true)
  }, [onClose])

  return (
    <div onClick={(e) => { if (e.target === e.currentTarget) onClose() }}
      style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        zIndex: 10001, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 40,
      }}>
      <div style={{
        background: '#fff', borderRadius: 16, padding: 0,
        maxWidth: 680, width: '100%', maxHeight: '70vh',
        display: 'flex', flexDirection: 'column',
        boxShadow: '0 20px 60px rgba(0,0,0,0.2), 0 0 0 1px rgba(0,0,0,0.05)',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '16px 20px', borderBottom: '1px solid rgba(0,0,0,0.06)', flexShrink: 0 }}>
          <FileText size={16} color="#007aff" />
          <span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e', flex: 1 }}>修改理由（Translator指令）</span>
          <button onClick={onClose} style={{
            padding: '4px 12px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)',
            background: '#f5f5f5', fontSize: 12, fontWeight: 500, color: '#3c3c43',
            cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 4,
          }}><X size={12} /> 关闭</button>
        </div>
        <div style={{ padding: '20px 24px', overflow: 'auto', flex: 1, fontSize: 13, color: '#3c3c43', lineHeight: 1.8, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
          {reason}
        </div>
      </div>
    </div>
  )
}

// ==================== 全屏预览 ====================

function FullscreenPreview({ page, pages, currentIdx, pipelineId, onNavigate, onClose, onPageUpdated }: {
  page: GeneratedPageFull; pages: GeneratedPageFull[]; currentIdx: number
  pipelineId: string
  onNavigate: (idx: number) => void; onClose: () => void
  onPageUpdated: (pageNumber: number, newHTML: string) => void
}) {
  const [fsMode, setFsMode] = useState<'generated' | 'original' | 'compare'>('generated')
  const [showReasonModal, setShowReasonModal] = useState(false)
  const [showAIFixModal, setShowAIFixModal] = useState(false)
  const [aiFixInstruction, setAIFixInstruction] = useState('')
  const [aiFixLoading, setAIFixLoading] = useState(false)
  const [purePreview, setPurePreview] = useState(false)

  const hasPrev = currentIdx > 0
  const hasNext = currentIdx < pages.length - 1
  const hasOriginal = !!(page.original_html && page.original_html.length > 10)
  const hasGenerated = !!(page.generated_html && page.generated_html.length > 10)
  const hasBoth = hasOriginal && hasGenerated

  useEffect(() => {
    if (page.operation === 'keep' || page.operation === 'delete') {
      setFsMode('original')
    } else {
      setFsMode('generated')
    }
    setShowReasonModal(false)
    setShowAIFixModal(false)
    setAIFixInstruction('')
    setPurePreview(false)
  }, [page.page_number, page.operation])

  const navBtn: React.CSSProperties = {
    padding: '6px 10px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.1)',
    background: '#fff', display: 'inline-flex', alignItems: 'center',
    fontSize: 13, fontWeight: 500, color: '#3c3c43', cursor: 'pointer',
  }

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
      zIndex: 9999, background: purePreview ? '#fff' : 'rgba(0,0,0,0.6)',
      backdropFilter: purePreview ? 'none' : 'blur(8px)',
      display: 'flex', flexDirection: 'column',
    }} onClick={(e) => { if (e.target === e.currentTarget) { purePreview ? setPurePreview(false) : onClose() } }}>

      {!purePreview && <div style={{
        display: 'flex', alignItems: 'center', gap: 10, padding: '10px 20px',
        background: 'rgba(255,255,255,0.95)', backdropFilter: 'blur(20px)',
        borderBottom: '1px solid rgba(0,0,0,0.08)', flexShrink: 0,
      }}>
        <button onClick={() => hasPrev && onNavigate(currentIdx - 1)}
          style={{ ...navBtn, opacity: hasPrev ? 1 : 0.3, cursor: hasPrev ? 'pointer' : 'not-allowed' }}
          disabled={!hasPrev}><ChevronLeft size={16} /></button>

        <span style={{ fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px', borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2' }}>
          {OP_NAMES[page.operation] || page.operation}
        </span>
        <span style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>
          {formatPageLabel(page.page_number, page.page_title)}. {page.page_title || '无标题'}
        </span>
        <span style={{ fontSize: 12, color: '#8e8e93' }}>{currentIdx + 1}/{pages.length}</span>

        <button onClick={() => hasNext && onNavigate(currentIdx + 1)}
          style={{ ...navBtn, opacity: hasNext ? 1 : 0.3, cursor: hasNext ? 'pointer' : 'not-allowed' }}
          disabled={!hasNext}><ChevronRight size={16} /></button>

        <div style={{ flex: 1 }} />

        {hasBoth && (
          <div style={{ display: 'flex', gap: 0, borderRadius: 8, overflow: 'hidden', border: '1px solid rgba(0,0,0,0.1)' }}>
            {([
              { key: 'original' as const, label: '原版' },
              { key: 'generated' as const, label: '修改后' },
              { key: 'compare' as const, label: '对比' },
            ]).map(item => (
              <button key={item.key} onClick={() => setFsMode(item.key)} style={{
                padding: '6px 14px', border: 'none', fontSize: 12, fontWeight: 500, cursor: 'pointer',
                background: fsMode === item.key ? '#007aff' : '#fff',
                color: fsMode === item.key ? '#fff' : '#3c3c43',
              }}>{item.key === 'compare' && <Columns size={12} style={{ marginRight: 4, verticalAlign: -1 }} />}{item.label}</button>
            ))}
          </div>
        )}
        {hasOriginal && !hasGenerated && <span style={{ fontSize: 12, color: '#8e8e93', fontStyle: 'italic' }}>仅原版</span>}
        {!hasOriginal && hasGenerated && <span style={{ fontSize: 12, color: '#8e8e93', fontStyle: 'italic' }}>仅生成版（新建）</span>}

        {page.change_reason && (
          <button onClick={() => setShowReasonModal(true)} style={{ ...navBtn, padding: '6px 14px', background: '#f0f7ff', color: '#007aff', border: '1px solid rgba(0,122,255,0.15)' }}>
            <FileText size={13} /> 修改理由
          </button>
        )}
        <button onClick={() => setPurePreview(true)} style={{ ...navBtn, padding: '6px 14px', background: '#f0faf0', color: '#2e7d32', border: '1px solid rgba(46,125,50,0.15)' }}>
          <Eye size={13} /> 纯净预览
        </button>
        <button onClick={() => { setShowAIFixModal(true); setAIFixInstruction('') }} style={{ ...navBtn, padding: '6px 14px', background: '#fff3e0', color: '#e65100', border: '1px solid rgba(230,81,0,0.15)' }}>
          <Wand2 size={13} /> AI快修
        </button>
        <button onClick={onClose} style={{ ...navBtn, padding: '6px 14px' }}>
          <Minimize2 size={14} /> 退出
        </button>
      </div>}

      <div style={{ flex: 1, overflow: 'hidden', background: '#fff', position: 'relative' }}>
        {fsMode === 'compare' && hasBoth ? (
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

        {hasPrev && !purePreview && (
          <div onClick={() => onNavigate(currentIdx - 1)} style={{
            position: 'absolute', left: 0, top: 0, bottom: 0, width: 60, cursor: 'pointer',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'linear-gradient(to right, rgba(0,0,0,0.05), transparent)',
          }}><ChevronLeft size={24} color="#8e8e93" /></div>
        )}
        {hasNext && !purePreview && (
          <div onClick={() => onNavigate(currentIdx + 1)} style={{
            position: 'absolute', right: 0, top: 0, bottom: 0, width: 60, cursor: 'pointer',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'linear-gradient(to left, rgba(0,0,0,0.05), transparent)',
          }}><ChevronRight size={24} color="#8e8e93" /></div>
        )}
      </div>

      {purePreview && (
        <div onClick={() => setPurePreview(false)} style={{
          position: 'fixed', bottom: 20, left: '50%', transform: 'translateX(-50%)',
          zIndex: 10002, background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(8px)',
          color: '#fff', fontSize: 12, fontWeight: 500, padding: '8px 20px',
          borderRadius: 20, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6,
          opacity: 0.7, transition: 'opacity 0.2s',
        }}
          onMouseEnter={(e) => { (e.target as HTMLElement).style.opacity = '1' }}
          onMouseLeave={(e) => { (e.target as HTMLElement).style.opacity = '0.7' }}>
          <EyeOff size={13} /> 点击此处或按 ESC 退出纯净预览
        </div>
      )}

      {showReasonModal && page.change_reason && (
        <ReasonModal reason={page.change_reason} onClose={() => setShowReasonModal(false)} />
      )}

      {showAIFixModal && (
        <AIFixModal
          pipelineId={pipelineId}
          pageNumber={page.page_number}
          pageTitle={page.page_title}
          loading={aiFixLoading}
          instruction={aiFixInstruction}
          onInstructionChange={setAIFixInstruction}
          onSubmit={async () => {
            if (!aiFixInstruction.trim()) { alert('请输入修复指令'); return }
            setAIFixLoading(true)
            try {
              const resp = await aiFixPage(pipelineId, page.page_number, { fix_instruction: aiFixInstruction.trim() })
              onPageUpdated(page.page_number, resp.new_html)
              alert('AI快修完成！新HTML已更新（' + resp.html_length + '字符）')
              setShowAIFixModal(false)
              setAIFixInstruction('')
            } catch (e: any) {
              alert('AI快修失败: ' + (e?.response?.data?.message || e.message || '未知错误'))
            }
            setAIFixLoading(false)
          }}
          onClose={() => { if (!aiFixLoading) { setShowAIFixModal(false); setAIFixInstruction('') } }}
        />
      )}
    </div>
  )
}

// ==================== HTML预览 ====================

/**
 * HTMLPreview iframe HTML内容预览组件
 * 修复FP-03：移除 allow-scripts，防止AI生成HTML中的JS在审核员浏览器中执行
 */
function HTMLPreview({ html }: { html: string }) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const doc = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<style>
body{margin:0;padding:16px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;font-size:14px;line-height:1.6;color:#1c1c1e;word-wrap:break-word}
img{max-width:100%;height:auto}table{border-collapse:collapse;width:100%}th,td{border:1px solid #e5e5ea;padding:8px 12px;text-align:left}th{background:#f5f5f7;font-weight:600}
</style></head><body>${html || '<p style="color:#aeaeb2;text-align:center;padding:40px 0">(空内容)</p>'}</body></html>`

  if (!html && html !== '') {
    return <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 13 }}>（无HTML内容）</div>
  }
  return (
    <iframe
      ref={iframeRef}
      srcDoc={doc}
      sandbox="allow-same-origin allow-scripts"
      style={{ flex: 1, width: '100%', height: '100%', border: 'none', background: '#fff' }}
      title="HTML预览"
    />
  )
}
