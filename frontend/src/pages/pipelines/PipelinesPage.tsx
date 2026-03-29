/**
 * Pipeline列表页面
 * v44增强: 表头排序（课程编号/状态/评估均分/Meta分/翻译分/进度/创建时间）+ 前端分页（20/50/100可选）
 * P4.5-A增强: 数据表格视图
 * P4.5-D增强: 快捷通过按钮
 * P5-3增强: 批量创建+批量启动+多选操作
 * v41修复: 导航路径加 /workflow 前缀
 * Apple风格内联CSS
 */
import { useState, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getPipelines, createPipeline, startPipeline, cancelPipeline, deletePipeline,
  markPassed, batchCreatePipelines, batchStartPipelines, getOperators, batchAssignPipelines, batchRestartFromStep,
  type PipelineListItem, type CreatePipelineRequest, type OperatorInfo,
} from '@/api/pipelines'
import {
  Workflow, Play, Square, Trash2, RefreshCw, Plus,
  CheckCircle, XCircle, Clock, AlertTriangle, Loader, Eye, Zap,
  Layers, Rocket, UserPlus, X as XIcon, CheckCircle as CheckCircleIcon, RotateCcw,
  ArrowUp, ArrowDown, ChevronsLeft, ChevronsRight, ChevronLeft, ChevronRight,
} from 'lucide-react'

// ==================== Toast组件 ====================
function Toast({ message, type, onClose }: { message: string; type: 'ok' | 'err' | 'info'; onClose: () => void }) {
  useEffect(() => { const t = setTimeout(onClose, 5000); return () => clearTimeout(t) }, [onClose])
  const bg = type === 'ok' ? '#34c759' : type === 'err' ? '#ff3b30' : '#007aff'
  return (
    <div style={{ position: 'fixed', bottom: 24, right: 24, background: bg, color: '#fff', padding: '12px 22px', borderRadius: 12, fontSize: 13, fontWeight: 500, zIndex: 9999, boxShadow: '0 4px 24px rgba(0,0,0,0.18)', maxWidth: 500 }}>
      {message}
    </div>
  )
}

// ==================== 状态徽章（紧凑版） ====================
function StatusBadge({ status, statusName }: { status: string; statusName: string }) {
  const colorMap: Record<string, { bg: string; fg: string }> = {
    pending:      { bg: 'rgba(142,142,147,0.12)', fg: '#8e8e93' },
    running:      { bg: 'rgba(0,122,255,0.12)', fg: '#007aff' },
    review_queue: { bg: 'rgba(255,149,0,0.12)', fg: '#ff9500' },
    finalized:    { bg: 'rgba(52,199,89,0.12)', fg: '#34c759' },
    needs_human:  { bg: 'rgba(255,204,0,0.12)', fg: '#cc9900' },
    failed:       { bg: 'rgba(255,59,48,0.12)', fg: '#ff3b30' },
    cancelled:    { bg: 'rgba(142,142,147,0.08)', fg: '#aeaeb2' },
    verified:     { bg: 'rgba(52,199,89,0.15)', fg: '#248a3d' },
    verify_failed:{ bg: 'rgba(255,59,48,0.15)', fg: '#d70015' },
    pending_finalize: { bg: 'rgba(204,102,0,0.12)', fg: '#cc6600' },
  }
  const c = colorMap[status] || colorMap.pending
  const iconMap: Record<string, React.ReactNode> = {
    pending:      <Clock size={11} />,
    running:      <Loader size={11} style={{ animation: 'spin 1s linear infinite' }} />,
    review_queue: <Eye size={11} />,
    finalized:    <CheckCircle size={11} />,
    needs_human:  <AlertTriangle size={11} />,
    failed:       <XCircle size={11} />,
    cancelled:    <Square size={11} />,
    verified:     <CheckCircle size={11} />,
    verify_failed:<XCircle size={11} />,
    pending_finalize: <Clock size={11} />,
  }
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 3,
      padding: '2px 8px', borderRadius: 20, fontSize: 11, fontWeight: 500,
      background: c.bg, color: c.fg, whiteSpace: 'nowrap',
    }}>
      {iconMap[status]}{statusName}
    </span>
  )
}

// ==================== 进度条（紧凑版） ====================
function ProgressBar({ completed, total }: { completed: number; total: number }) {
  const pct = total > 0 ? (completed / total) * 100 : 0
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <div style={{ flex: 1, height: 5, background: 'rgba(0,0,0,0.06)', borderRadius: 3, overflow: 'hidden', minWidth: 50 }}>
        <div style={{ width: pct + '%', height: '100%', background: completed === total ? '#34c759' : '#007aff', borderRadius: 3, transition: 'width 0.3s ease' }} />
      </div>
      <span style={{ fontSize: 11, color: '#8e8e93', fontWeight: 500, whiteSpace: 'nowrap' }}>{completed}/{total}</span>
    </div>
  )
}

// ==================== 分数单元格组件 ====================
function ScoreCell({ value }: { value: number | null }) {
  if (value === null || value === undefined) {
    return <span style={{ color: '#c7c7cc', fontSize: 12 }}>-</span>
  }
  let color = '#ff3b30'
  if (value >= 9.0) color = '#34c759'
  else if (value >= 7.0) color = '#ff9500'
  return (
    <span style={{ color, fontSize: 13, fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>
      {value.toFixed(1)}
    </span>
  )
}

// ==================== 创建弹窗（单个） ====================
function CreateDialog({ onClose, onCreate }: { onClose: () => void; onCreate: (req: CreatePipelineRequest) => void }) {
  const [courseCode, setCourseCode] = useState('')
  const [threshold, setThreshold] = useState('9.0')
  const [evalRounds, setEvalRounds] = useState('3')
  const [maxTRLoop, setMaxTRLoop] = useState('3')
  const [maxMetaRetry, setMaxMetaRetry] = useState('3')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async () => {
    if (!courseCode.trim()) return
    setSubmitting(true)
    onCreate({
      course_code: courseCode.trim(),
      config: {
        threshold: parseFloat(threshold) || 9.0,
        eval_rounds: parseInt(evalRounds) || 3,
        max_tr_loop: parseInt(maxTRLoop) || 3,
        max_meta_retry: parseInt(maxMetaRetry) || 3,
      },
    })
  }

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.1)',
    fontSize: 14, outline: 'none', boxSizing: 'border-box', background: '#fafafa',
    transition: 'border-color 0.15s ease',
  }
  const labelStyle: React.CSSProperties = { fontSize: 12, fontWeight: 600, color: '#3c3c43', marginBottom: 4, display: 'block' }

  return (
    <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{ background: '#fff', borderRadius: 20, width: 440, maxWidth: '94vw', padding: '28px', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ fontSize: 18, fontWeight: 700, color: '#1c1c1e', marginBottom: 20 }}>创建 Pipeline</div>
        <div style={{ marginBottom: 16 }}>
          <label style={labelStyle}>课程编号 *</label>
          <input style={inputStyle} placeholder="如 G1-01" value={courseCode}
            onChange={e => setCourseCode(e.target.value)}
            onFocus={e => (e.target.style.borderColor = '#007aff')}
            onBlur={e => (e.target.style.borderColor = 'rgba(0,0,0,0.1)')} />
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 20 }}>
          <div>
            <label style={labelStyle}>达标阈值</label>
            <input style={inputStyle} type="number" step="0.5" value={threshold} onChange={e => setThreshold(e.target.value)} />
          </div>
          <div>
            <label style={labelStyle}>评估轮数</label>
            <input style={inputStyle} type="number" min="1" max="10" value={evalRounds} onChange={e => setEvalRounds(e.target.value)} />
          </div>
          <div>
            <label style={labelStyle}>翻译循环上限</label>
            <input style={inputStyle} type="number" min="1" max="5" value={maxTRLoop} onChange={e => setMaxTRLoop(e.target.value)} />
          </div>
          <div>
            <label style={labelStyle}>Meta重试上限</label>
            <input style={inputStyle} type="number" min="1" max="5" value={maxMetaRetry} onChange={e => setMaxMetaRetry(e.target.value)} />
          </div>
        </div>
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button onClick={onClose} style={{
            padding: '10px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
            background: '#fff', fontSize: 14, fontWeight: 500, cursor: 'pointer', color: '#3c3c43',
          }}>取消</button>
          <button onClick={handleSubmit} disabled={!courseCode.trim() || submitting} style={{
            padding: '10px 24px', borderRadius: 10, border: 'none',
            background: courseCode.trim() && !submitting ? '#007aff' : '#c7c7cc',
            color: '#fff', fontSize: 14, fontWeight: 600, cursor: courseCode.trim() && !submitting ? 'pointer' : 'not-allowed',
          }}>{submitting ? '创建中...' : '创建'}</button>
        </div>
      </div>
    </div>
  )
}

// ==================== 批量创建弹窗 ====================
function BatchCreateDialog({ onClose, onBatchCreate }: { onClose: () => void; onBatchCreate: (codes: string[]) => void }) {
  const [inputText, setInputText] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const parseCodes = (): string[] => {
    return inputText
      .split(/[,，\n\r\s]+/)
      .map(s => s.trim())
      .filter(s => s.length > 0)
  }

  const codes = parseCodes()

  const handleSubmit = () => {
    if (codes.length === 0) return
    setSubmitting(true)
    onBatchCreate(codes)
  }

  return (
    <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{ background: '#fff', borderRadius: 20, width: 500, maxWidth: '94vw', padding: '28px', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 20 }}>
          <Layers size={20} style={{ color: '#007aff' }} />
          <div style={{ fontSize: 18, fontWeight: 700, color: '#1c1c1e' }}>批量创建 Pipeline</div>
        </div>
        <div style={{ marginBottom: 12 }}>
          <label style={{ fontSize: 12, fontWeight: 600, color: '#3c3c43', marginBottom: 4, display: 'block' }}>
            输入课程编号（逗号、换行或空格分隔）
          </label>
          <textarea
            style={{
              width: '100%', height: 140, padding: '12px 14px', borderRadius: 10,
              border: '1px solid rgba(0,0,0,0.1)', fontSize: 14, outline: 'none',
              boxSizing: 'border-box', background: '#fafafa', fontFamily: 'monospace',
              resize: 'vertical', lineHeight: 1.6,
            }}
            placeholder={'G1-01, G1-02, G1-03\nG2-01\nG3-05'}
            value={inputText}
            onChange={e => setInputText(e.target.value)}
          />
        </div>
        <div style={{ fontSize: 12, color: '#8e8e93', marginBottom: 16 }}>
          已识别 <span style={{ fontWeight: 600, color: codes.length > 0 ? '#007aff' : '#8e8e93' }}>{codes.length}</span> 个课程编号
          {codes.length > 0 && codes.length <= 20 && (
            <span style={{ marginLeft: 8, color: '#aeaeb2' }}>{codes.join(', ')}</span>
          )}
          {codes.length > 100 && (
            <span style={{ marginLeft: 8, color: '#ff3b30' }}>（上限100个）</span>
          )}
        </div>
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button onClick={onClose} style={{
            padding: '10px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
            background: '#fff', fontSize: 14, fontWeight: 500, cursor: 'pointer', color: '#3c3c43',
          }}>取消</button>
          <button onClick={handleSubmit}
            disabled={codes.length === 0 || codes.length > 100 || submitting}
            style={{
              padding: '10px 24px', borderRadius: 10, border: 'none',
              background: codes.length > 0 && codes.length <= 100 && !submitting ? '#007aff' : '#c7c7cc',
              color: '#fff', fontSize: 14, fontWeight: 600,
              cursor: codes.length > 0 && codes.length <= 100 && !submitting ? 'pointer' : 'not-allowed',
            }}>
            {submitting ? '创建中...' : '批量创建 (' + codes.length + ')'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ==================== 格式化工具函数 ====================

function formatTime(t: string | null): string {
  if (!t) return '-'
  const d = new Date(t)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前'
  return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

// ==================== 筛选按钮定义 ====================
const FILTER_OPTIONS = [
  { label: '全部',   value: '' },
  { label: '运行中', value: 'running' },
  { label: '待审核', value: 'review_queue' },
  { label: '失败',   value: 'failed' },
  { label: '已完成', value: 'finalized' },
  { label: '待启动', value: 'pending' },
  { label: '已取消', value: 'cancelled' },
  { label: '验收通过', value: 'verified' },
  { label: '验收未通过', value: 'verify_failed' },
]

// ==================== v44新增：排序相关类型和常量 ====================

/** 可排序字段 */
type SortField = 'course_code' | 'status' | 'eval_avg_score' | 'meta_score' | 'translator_score' | 'progress' | 'created_at'
/** 排序方向 */
type SortDirection = 'asc' | 'desc'

/** 状态排序优先级（运行中/待审核/失败排在前面，已完成的排在后面） */
const STATUS_SORT_ORDER: Record<string, number> = {
  running: 1,
  review_queue: 2,
  needs_human: 3,
  pending_finalize: 4,
  failed: 5,
  pending: 6,
  finalized: 7,
  verified: 8,
  verify_failed: 9,
  cancelled: 10,
}

/** 每页条数选项 */
const PAGE_SIZE_OPTIONS = [20, 50, 100]

// ==================== 判断是否可快捷通过 ====================
function canMarkPassed(p: PipelineListItem): boolean {
  const allowedStatuses = ['review_queue', 'needs_human', 'failed']
  return allowedStatuses.includes(p.status) && p.meta_score !== null && p.meta_score >= 9.0
}

// ==================== v44新增：排序表头组件 ====================

function SortableHeader({ label, field, currentSort, currentDir, onSort, align }: {
  label: string; field: SortField
  currentSort: SortField; currentDir: SortDirection
  onSort: (field: SortField) => void
  align?: 'center' | 'left'
}) {
  const isActive = currentSort === field
  return (
    <span
      onClick={(e) => { e.stopPropagation(); onSort(field) }}
      style={{
        cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 3,
        userSelect: 'none', color: isActive ? '#007aff' : '#8e8e93',
        justifyContent: align === 'center' ? 'center' : 'flex-start',
      }}
      title={'点击按' + label + '排序'}
    >
      {label}
      {isActive && (currentDir === 'asc' ? <ArrowUp size={14} strokeWidth={2.5} /> : <ArrowDown size={14} strokeWidth={2.5} />)}
      {!isActive && <ArrowDown size={12} style={{ opacity: 0.4 }} />}
    </span>
  )
}

// ==================== v44新增：分页器组件 ====================

function Pagination({ currentPage, totalPages, totalItems, pageSize, onPageChange, onPageSizeChange }: {
  currentPage: number; totalPages: number; totalItems: number; pageSize: number
  onPageChange: (page: number) => void; onPageSizeChange: (size: number) => void
}) {
  if (totalItems === 0) return null

  const startItem = (currentPage - 1) * pageSize + 1
  const endItem = Math.min(currentPage * pageSize, totalItems)

  const pgBtn: React.CSSProperties = {
    width: 32, height: 32, borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
    cursor: 'pointer', fontSize: 13, fontWeight: 500, color: '#3c3c43',
    transition: 'all 0.15s ease',
  }
  const pgBtnDisabled: React.CSSProperties = { ...pgBtn, opacity: 0.3, cursor: 'not-allowed', color: '#aeaeb2' }

  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
      padding: '12px 16px', borderTop: '1px solid rgba(0,0,0,0.06)',
      background: 'rgba(249,249,249,0.8)', borderRadius: '0 0 16px 16px',
      flexWrap: 'wrap', gap: 12,
    }}>
      {/* 左侧：显示信息 + 每页条数选择 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, fontSize: 12, color: '#8e8e93' }}>
        <span>共 <b style={{ color: '#1c1c1e' }}>{totalItems}</b> 条，显示 {startItem}-{endItem}</span>
        <span style={{ color: '#d1d1d6' }}>|</span>
        <span>每页</span>
        <select
          value={pageSize}
          onChange={e => onPageSizeChange(Number(e.target.value))}
          style={{
            padding: '4px 8px', borderRadius: 6, border: '1px solid rgba(0,0,0,0.1)',
            fontSize: 12, background: '#fff', cursor: 'pointer', outline: 'none',
            color: '#1c1c1e', fontWeight: 500,
          }}
        >
          {PAGE_SIZE_OPTIONS.map(size => (
            <option key={size} value={size}>{size} 条</option>
          ))}
        </select>
      </div>

      {/* 右侧：翻页按钮 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        {/* 首页 */}
        <button
          onClick={() => currentPage > 1 && onPageChange(1)}
          disabled={currentPage <= 1}
          style={currentPage <= 1 ? pgBtnDisabled : pgBtn}
          title="首页"
        ><ChevronsLeft size={14} /></button>
        {/* 上一页 */}
        <button
          onClick={() => currentPage > 1 && onPageChange(currentPage - 1)}
          disabled={currentPage <= 1}
          style={currentPage <= 1 ? pgBtnDisabled : pgBtn}
          title="上一页"
        ><ChevronLeft size={14} /></button>

        {/* 页码按钮（最多显示5个） */}
        {(() => {
          const pages: number[] = []
          let start = Math.max(1, currentPage - 2)
          let end = Math.min(totalPages, start + 4)
          if (end - start < 4) start = Math.max(1, end - 4)
          for (let i = start; i <= end; i++) pages.push(i)
          return pages.map(p => (
            <button key={p} onClick={() => onPageChange(p)} style={{
              ...pgBtn,
              background: p === currentPage ? '#007aff' : '#fff',
              color: p === currentPage ? '#fff' : '#3c3c43',
              border: p === currentPage ? '1px solid #007aff' : '1px solid rgba(0,0,0,0.08)',
              fontWeight: p === currentPage ? 700 : 500,
            }}>{p}</button>
          ))
        })()}

        {/* 下一页 */}
        <button
          onClick={() => currentPage < totalPages && onPageChange(currentPage + 1)}
          disabled={currentPage >= totalPages}
          style={currentPage >= totalPages ? pgBtnDisabled : pgBtn}
          title="下一页"
        ><ChevronRight size={14} /></button>
        {/* 末页 */}
        <button
          onClick={() => currentPage < totalPages && onPageChange(totalPages)}
          disabled={currentPage >= totalPages}
          style={currentPage >= totalPages ? pgBtnDisabled : pgBtn}
          title="末页"
        ><ChevronsRight size={14} /></button>
      </div>
    </div>
  )
}

// ==================== 主页面组件 ====================

export default function PipelinesPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin' || user?.role === 'senior_operator'
  const canOperate = user?.role === 'admin' || user?.role === 'operator' || user?.role === 'senior_operator'

  const [pipelines, setPipelines] = useState<PipelineListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [showBatchCreate, setShowBatchCreate] = useState(false)
  const [operating, setOperating] = useState<string | null>(null)
  const [toast, setToast] = useState<{ message: string; type: 'ok' | 'err' | 'info' } | null>(null)
  const [statusFilter, setStatusFilter] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [showAssignDialog, setShowAssignDialog] = useState(false)
  const [operators, setOperators] = useState<OperatorInfo[]>([])
  const [selectedOperator, setSelectedOperator] = useState('')
  const [assigning, setAssigning] = useState(false)

  /** v44新增：排序状态（默认按创建时间倒序） */
  const [sortField, setSortField] = useState<SortField>('created_at')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')

  /** v44新增：分配人筛选 */
  const [assigneeFilter, setAssigneeFilter] = useState('')

  /** v44新增：分页状态 */
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize, setPageSize] = useState(50)

  const loadPipelines = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getPipelines()
      setPipelines(data.pipelines || [])
    } catch (e: any) {
      setToast({ message: '加载失败: ' + (e.message || ''), type: 'err' })
    }
    setLoading(false)
  }, [])

  useEffect(() => { loadPipelines() }, [loadPipelines])

  // 筛选/排序变化时回到第1页+清空选中
  useEffect(() => { setSelectedIds(new Set()); setCurrentPage(1) }, [statusFilter, sortField, sortDir, assigneeFilter])
  // 每页条数变化时回到第1页
  useEffect(() => { setCurrentPage(1) }, [pageSize])

  /**
   * v44新增：排序逻辑（点击表头切换排序）
   * - 点击当前排序字段：切换升序/降序
   * - 点击新字段：设为降序（数值类大的在前更常用）
   * - 课程编号默认升序（字母序更直观）
   */
  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(prev => prev === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDir(field === 'course_code' ? 'asc' : 'desc')
    }
  }

  /**
   * v44新增：筛选 → 排序 → 分页的完整管道
   * 使用useMemo缓存，避免每次渲染重新计算
   */
  const { pagedPipelines, filteredTotal, totalPages } = useMemo(() => {
    // 1. 状态筛选
    let filtered = statusFilter
      ? pipelines.filter(p => p.status === statusFilter)
      : pipelines

    // 1.5 分配人筛选
    if (assigneeFilter === 'unassigned') {
      filtered = filtered.filter(p => !p.assigned_name)
    } else if (assigneeFilter) {
      filtered = filtered.filter(p => p.assigned_name === assigneeFilter)
    }

    // 2. 排序
    const sorted = [...filtered].sort((a, b) => {
      let cmp = 0
      switch (sortField) {
        case 'course_code':
          // 自然排序：G1-01 < G1-02 < G2-01 < G10-01
          cmp = a.course_code.localeCompare(b.course_code, 'zh-CN', { numeric: true })
          break
        case 'status':
          cmp = (STATUS_SORT_ORDER[a.status] || 99) - (STATUS_SORT_ORDER[b.status] || 99)
          break
        case 'eval_avg_score': {
          const sa = a.eval_avg_score ?? -1
          const sb = b.eval_avg_score ?? -1
          cmp = sa - sb
          break
        }
        case 'meta_score': {
          const ma = a.meta_score ?? -1
          const mb = b.meta_score ?? -1
          cmp = ma - mb
          break
        }
        case 'translator_score': {
          const ta = a.translator_score ?? -1
          const tb = b.translator_score ?? -1
          cmp = ta - tb
          break
        }
        case 'progress': {
          const pa = a.steps_total > 0 ? a.steps_completed / a.steps_total : 0
          const pb = b.steps_total > 0 ? b.steps_completed / b.steps_total : 0
          cmp = pa - pb
          break
        }
        case 'created_at':
          cmp = new Date(a.created_at || 0).getTime() - new Date(b.created_at || 0).getTime()
          break
      }
      return sortDir === 'asc' ? cmp : -cmp
    })

    // 3. 分页
    const total = sorted.length
    const pages = Math.max(1, Math.ceil(total / pageSize))
    const start = (currentPage - 1) * pageSize
    const paged = sorted.slice(start, start + pageSize)

    return { pagedPipelines: paged, filteredTotal: total, totalPages: pages }
  }, [pipelines, statusFilter, sortField, sortDir, currentPage, pageSize, assigneeFilter])

  // ==================== 操作函数（与原版完全相同） ====================

  const handleCreate = async (req: CreatePipelineRequest) => {
    try {
      await createPipeline(req)
      setToast({ message: req.course_code + ' Pipeline创建成功', type: 'ok' })
      setShowCreate(false)
      loadPipelines()
    } catch (e: any) {
      setToast({ message: '创建失败: ' + (e.message || ''), type: 'err' })
    }
  }

  const handleBatchCreate = async (codes: string[]) => {
    try {
      const result = await batchCreatePipelines(codes)
      const parts: string[] = []
      if (result.created_ids.length > 0) parts.push('成功创建 ' + result.created_ids.length + ' 个')
      if (result.skipped_codes.length > 0) parts.push('跳过 ' + result.skipped_codes.length + ' 个')
      if (result.failed_codes.length > 0) parts.push('失败 ' + result.failed_codes.length + ' 个')
      setToast({ message: '批量创建完成: ' + parts.join(', '), type: result.failed_codes.length > 0 ? 'err' : 'ok' })
      setShowBatchCreate(false)
      loadPipelines()
    } catch (e: any) {
      setToast({ message: '批量创建失败: ' + (e.message || ''), type: 'err' })
    }
  }

  const handleBatchStart = async () => {
    const ids = Array.from(selectedIds)
    const pendingIds = ids.filter(id => {
      const p = pipelines.find(pp => pp.id === id)
      return p && p.status === 'pending'
    })
    if (pendingIds.length === 0) {
      setToast({ message: '选中的Pipeline中没有待启动的', type: 'info' })
      return
    }
    if (!confirm('确认批量启动 ' + pendingIds.length + ' 个Pipeline？\n每个Pipeline约需10-50分钟执行。')) return
    try {
      const result = await batchStartPipelines(pendingIds)
      const parts: string[] = []
      if (result.started_ids.length > 0) parts.push('已提交 ' + result.started_ids.length + ' 个')
      if (result.skipped_ids.length > 0) parts.push('跳过 ' + result.skipped_ids.length + ' 个')
      if (result.failed_ids.length > 0) parts.push('失败 ' + result.failed_ids.length + ' 个')
      setToast({ message: '批量启动完成: ' + parts.join(', '), type: result.failed_ids.length > 0 ? 'err' : 'ok' })
      setSelectedIds(new Set())
      loadPipelines()
    } catch (e: any) {
      setToast({ message: '批量启动失败: ' + (e.message || ''), type: 'err' })
    }
  }

  const handleBatchRestartGenerator = async () => {
    const ids = Array.from(selectedIds)
    if (ids.length === 0) {
      setToast({ message: '请先选择要重跑的Pipeline', type: 'info' })
      return
    }
    if (ids.length > 50) {
      setToast({ message: '单次批量重跑上限50个Pipeline', type: 'err' })
      return
    }
    if (!confirm('确认批量从Generator步骤重跑选中的 ' + ids.length + ' 个Pipeline？\n\n此操作将：\n• 重置Generator及后续步骤的执行数据\n• 清空已生成的页面\n• 重置审核和验收步骤\n• 每个Pipeline约需10-30分钟执行\n\n已完成的前序步骤（数据检查、评估、Meta、翻译）不受影响。')) return
    try {
      const result = await batchRestartFromStep(ids, 'generator')
      const parts: string[] = []
      if (result.success_count > 0) parts.push('成功提交 ' + result.success_count + ' 个')
      if (result.skipped_ids.length > 0) parts.push('跳过 ' + result.skipped_ids.length + ' 个')
      if (result.failed_ids.length > 0) parts.push('失败 ' + result.failed_ids.length + ' 个')
      setToast({ message: '批量重跑完成: ' + parts.join(', '), type: result.failed_ids.length > 0 ? 'err' : 'ok' })
      setSelectedIds(new Set())
      loadPipelines()
    } catch (e: any) {
      setToast({ message: '批量重跑失败: ' + (e.message || ''), type: 'err' })
    }
  }

  const openAssignDialog = async () => {
    if (selectedIds.size === 0) return
    try {
      const ops = await getOperators()
      setOperators(ops || [])
    } catch { /* ignore */ }
    setSelectedOperator('')
    setShowAssignDialog(true)
  }

  const handleBatchAssign = async () => {
    if (!selectedOperator) return
    setAssigning(true)
    try {
      const ids = Array.from(selectedIds)
      const result = await batchAssignPipelines(ids, selectedOperator)
      setToast({ message: '分配成功: ' + result.success_count + ' 个Pipeline已分配给 ' + result.assigned_name, type: 'ok' })
      setShowAssignDialog(false)
      setSelectedIds(new Set())
      loadPipelines()
    } catch (e: any) {
      setToast({ message: '分配失败: ' + (e.message || ''), type: 'err' })
    }
    setAssigning(false)
  }

  const handleStart = async (p: PipelineListItem) => {
    if (!confirm('确认启动 ' + p.course_code + ' Pipeline？\n全链路AI调用约需10-30分钟。')) return
    setOperating(p.id)
    setToast({ message: p.course_code + ' 正在启动...', type: 'info' })
    try {
      await startPipeline(p.id)
      setToast({ message: p.course_code + ' Pipeline已提交执行', type: 'ok' })
      loadPipelines()
    } catch (e: any) {
      setToast({ message: '启动失败: ' + (e.message || ''), type: 'err' })
      loadPipelines()
    }
    setOperating(null)
  }

  const handleCancel = async (p: PipelineListItem) => {
    if (!confirm('确认取消 ' + p.course_code + ' Pipeline？')) return
    try {
      await cancelPipeline(p.id)
      setToast({ message: p.course_code + ' 已取消', type: 'ok' })
      loadPipelines()
    } catch (e: any) {
      setToast({ message: '取消失败: ' + (e.message || ''), type: 'err' })
    }
  }

  const handleDelete = async (p: PipelineListItem) => {
    if (!confirm('确认删除 ' + p.course_code + ' Pipeline？\n此操作不可恢复。')) return
    try {
      await deletePipeline(p.id)
      setToast({ message: p.course_code + ' 已删除', type: 'ok' })
      loadPipelines()
    } catch (e: any) {
      setToast({ message: '删除失败: ' + (e.message || ''), type: 'err' })
    }
  }

  const handleMarkPassed = async (p: PipelineListItem) => {
    if (!confirm('确认快捷通过 ' + p.course_code + ' Pipeline？\n将跳过审核流程直接标记为已定稿。')) return
    try {
      await markPassed(p.id)
      setToast({ message: p.course_code + ' 已快捷通过并归档', type: 'ok' })
      loadPipelines()
    } catch (e: any) {
      setToast({ message: '快捷通过失败: ' + (e.message || ''), type: 'err' })
    }
  }

  const toggleSelect = (id: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  // v44修改：全选/取消全选基于当前页数据
  const toggleSelectAll = () => {
    const currentPageIds = pagedPipelines.map(p => p.id)
    const allSelected = currentPageIds.length > 0 && currentPageIds.every(id => selectedIds.has(id))
    if (allSelected) {
      // 取消当前页全部选中
      setSelectedIds(prev => {
        const next = new Set(prev)
        currentPageIds.forEach(id => next.delete(id))
        return next
      })
    } else {
      // 选中当前页全部
      setSelectedIds(prev => {
        const next = new Set(prev)
        currentPageIds.forEach(id => next.add(id))
        return next
      })
    }
  }

  const isAllSelected = pagedPipelines.length > 0 && pagedPipelines.every(p => selectedIds.has(p.id))

  const selectedPendingCount = Array.from(selectedIds).filter(id => {
    const p = pipelines.find(pp => pp.id === id)
    return p && p.status === 'pending'
  }).length

  const total = pipelines.length
  const running = pipelines.filter(p => p.status === 'running').length
  const reviewQueue = pipelines.filter(p => p.status === 'review_queue').length
  const failed = pipelines.filter(p => p.status === 'failed').length
  const passedCount = pipelines.filter(p => p.eval_avg_score !== null && p.eval_avg_score >= 9.0).length

  const stat: React.CSSProperties = { background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)', borderRadius: 14, padding: '16px 20px', flex: 1, minWidth: 100 }
  const btn: React.CSSProperties = { padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6, transition: 'all 0.15s ease' }
  const btnP: React.CSSProperties = { ...btn, background: '#007aff', color: '#fff', border: '1px solid #007aff' }
  const th: React.CSSProperties = { padding: '10px 12px', textAlign: 'left', fontSize: 11, fontWeight: 600, color: '#8e8e93', textTransform: 'uppercase', letterSpacing: '0.02em', borderBottom: '1px solid rgba(0,0,0,0.06)', whiteSpace: 'nowrap' }
  const td: React.CSSProperties = { padding: '12px 12px', fontSize: 13, color: '#1c1c1e', borderBottom: '1px solid rgba(0,0,0,0.04)', verticalAlign: 'middle' }
  const checkboxStyle: React.CSSProperties = { width: 16, height: 16, cursor: 'pointer', accentColor: '#007aff' }

  return (
    <div>
      {/* 统计卡片 */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap' }}>
        <div style={stat}>
          <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 600, marginBottom: 4 }}>总Pipeline</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: '#1c1c1e' }}>{total}</div>
        </div>
        <div style={stat}>
          <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 600, marginBottom: 4 }}>运行中</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: '#007aff' }}>{running}</div>
        </div>
        <div style={stat}>
          <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 600, marginBottom: 4 }}>待审核</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: '#ff9500' }}>{reviewQueue}</div>
        </div>
        <div style={stat}>
          <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 600, marginBottom: 4 }}>失败</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: '#ff3b30' }}>{failed}</div>
        </div>
        <div style={stat}>
          <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 600, marginBottom: 4 }}>达标(≥9.0)</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: '#34c759' }}>{passedCount}</div>
        </div>
      </div>

      {/* 操作栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12, flexWrap: 'wrap', gap: 8 }}>
        <div style={{ fontSize: 13, color: '#8e8e93' }}>
          {loading ? '加载中...' : (statusFilter ? filteredTotal + ' / ' + total + ' 个Pipeline' : total + ' 个Pipeline')}
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button style={btn} onClick={loadPipelines}><RefreshCw size={14} /> 刷新</button>
          {canOperate && (
            <>
              <button style={{ ...btn, color: '#5856d6', borderColor: 'rgba(88,86,214,0.3)' }}
                onClick={() => setShowBatchCreate(true)}>
                <Layers size={14} /> 批量创建
              </button>
              <button style={btnP} onClick={() => setShowCreate(true)}><Plus size={14} /> 创建Pipeline</button>
            </>
          )}
        </div>
      </div>

      {/* 筛选按钮行 */}
      <div style={{ display: 'flex', gap: 6, marginBottom: 16, flexWrap: 'wrap' }}>
        {FILTER_OPTIONS.map(opt => (
          <button key={opt.value} onClick={() => setStatusFilter(opt.value)} style={{
            padding: '6px 14px', borderRadius: 20, fontSize: 12, fontWeight: 500, cursor: 'pointer',
            border: statusFilter === opt.value ? '1px solid #007aff' : '1px solid rgba(0,0,0,0.08)',
            background: statusFilter === opt.value ? 'rgba(0,122,255,0.1)' : '#fff',
            color: statusFilter === opt.value ? '#007aff' : '#3c3c43',
            transition: 'all 0.15s ease',
          }}>
            {opt.label}
            {opt.value && <span style={{ marginLeft: 4, opacity: 0.6 }}>
              {pipelines.filter(p => p.status === opt.value).length}
            </span>}
          </button>
        ))}
      </div>

      {/* 分配人筛选（v44新增） */}
      {(() => {
        const assigneeNames = Array.from(new Set(pipelines.map(p => p.assigned_name).filter(Boolean))).sort()
        if (assigneeNames.length === 0) return null
        const unassignedCount = pipelines.filter(p => !p.assigned_name).length
        const filterBtn = (active: boolean, color: string): React.CSSProperties => ({
          padding: '5px 12px', borderRadius: 16, fontSize: 12, fontWeight: 500, cursor: 'pointer',
          border: active ? ('1px solid ' + color) : '1px solid rgba(0,0,0,0.08)',
          background: active ? (color + '18') : '#fff',
          color: active ? color : '#3c3c43',
          transition: 'all 0.15s ease',
        })
        return (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 16, flexWrap: 'wrap' }}>
            <span style={{ fontSize: 12, color: '#8e8e93', fontWeight: 500, marginRight: 2 }}>分配给：</span>
            <button onClick={() => setAssigneeFilter('')} style={filterBtn(!assigneeFilter, '#5856d6')}>全部</button>
            <button onClick={() => setAssigneeFilter('unassigned')} style={filterBtn(assigneeFilter === 'unassigned', '#ff9500')}>
              未分配 <span style={{ opacity: 0.6, marginLeft: 2 }}>{unassignedCount}</span>
            </button>
            {assigneeNames.map(name => (
              <button key={name} onClick={() => setAssigneeFilter(name)} style={filterBtn(assigneeFilter === name, '#5856d6')}>
                {name} <span style={{ opacity: 0.6, marginLeft: 2 }}>{pipelines.filter(p => p.assigned_name === name).length}</span>
              </button>
            ))}
          </div>
        )
      })()}

      {/* Pipeline数据表格 */}
      <div style={{
        background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
        border: '1px solid rgba(0,0,0,0.06)', borderRadius: 16, overflow: 'hidden',
      }}>
        {loading ? (
          <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>加载中...</div>
        ) : filteredTotal === 0 ? (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Workflow size={40} style={{ color: '#c7c7cc', marginBottom: 12 }} />
            <div style={{ color: '#8e8e93', fontSize: 14 }}>
              {statusFilter ? '当前筛选条件下暂无Pipeline' : '暂无Pipeline'}
            </div>
            {canOperate && !statusFilter && (
              <div style={{ color: '#007aff', fontSize: 13, marginTop: 8, cursor: 'pointer' }}
                onClick={() => setShowCreate(true)}>创建第一个Pipeline →</div>
            )}
          </div>
        ) : (
          <>
            <div style={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', minWidth: 1050 }}>
                <thead>
                  <tr style={{ background: 'rgba(0,0,0,0.02)' }}>
                    {canOperate && (
                      <th style={{ ...th, width: 36, textAlign: 'center', padding: '10px 8px' }}>
                        <input type="checkbox" checked={isAllSelected} onChange={toggleSelectAll} style={checkboxStyle} />
                      </th>
                    )}
                    <th style={th}><SortableHeader label="课程编号" field="course_code" currentSort={sortField} currentDir={sortDir} onSort={handleSort} /></th>
                    <th style={th}>课程名称</th>
                    <th style={th}><SortableHeader label="状态" field="status" currentSort={sortField} currentDir={sortDir} onSort={handleSort} /></th>
                    <th style={th}>当前步骤</th>
                    <th style={{ ...th, textAlign: 'center' }}><SortableHeader label="评估均分" field="eval_avg_score" currentSort={sortField} currentDir={sortDir} onSort={handleSort} align="center" /></th>
                    <th style={{ ...th, textAlign: 'center' }}><SortableHeader label="Meta分" field="meta_score" currentSort={sortField} currentDir={sortDir} onSort={handleSort} align="center" /></th>
                    <th style={{ ...th, textAlign: 'center' }}><SortableHeader label="翻译分" field="translator_score" currentSort={sortField} currentDir={sortDir} onSort={handleSort} align="center" /></th>
                    <th style={th}><SortableHeader label="进度" field="progress" currentSort={sortField} currentDir={sortDir} onSort={handleSort} /></th>
                    <th style={th}>分配给</th>
                    <th style={th}><SortableHeader label="创建时间" field="created_at" currentSort={sortField} currentDir={sortDir} onSort={handleSort} /></th>
                    <th style={{ ...th, textAlign: 'center' }}>操作</th>
                  </tr>
                </thead>
                <tbody>
                  {pagedPipelines.map(p => (
                    <tr key={p.id}
                      style={{ cursor: 'pointer', transition: 'background 0.15s ease', background: selectedIds.has(p.id) ? 'rgba(0,122,255,0.04)' : 'transparent' }}
                      onClick={() => navigate('/workflow/pipelines/' + p.id)}
                      onMouseEnter={e => { if (!selectedIds.has(p.id)) (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.03)' }}
                      onMouseLeave={e => { if (!selectedIds.has(p.id)) (e.currentTarget as HTMLElement).style.background = 'transparent' }}
                    >
                      {canOperate && (
                        <td style={{ ...td, width: 36, textAlign: 'center', padding: '12px 8px' }}
                          onClick={e => e.stopPropagation()}>
                          <input type="checkbox" checked={selectedIds.has(p.id)} onChange={() => toggleSelect(p.id)} style={checkboxStyle} />
                        </td>
                      )}
                      <td style={{ ...td, fontWeight: 600, whiteSpace: 'nowrap' }}>{p.course_code}</td>
                      <td style={{ ...td, color: '#8e8e93', maxWidth: 180, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {p.course_name || p.course_code}
                      </td>
                      <td style={td}><StatusBadge status={p.status} statusName={p.status_name} /></td>
                      <td style={{ ...td, fontSize: 12, color: '#636366', whiteSpace: 'nowrap' }}>{p.current_step_name}</td>
                      <td style={{ ...td, textAlign: 'center' }}><ScoreCell value={p.eval_avg_score} /></td>
                      <td style={{ ...td, textAlign: 'center' }}><ScoreCell value={p.meta_score} /></td>
                      <td style={{ ...td, textAlign: 'center' }}><ScoreCell value={p.translator_score} /></td>
                      <td style={{ ...td, minWidth: 100 }}><ProgressBar completed={p.steps_completed} total={p.steps_total} /></td>
                      <td style={{ ...td, fontSize: 12, color: p.assigned_name ? '#5856d6' : '#c7c7cc', whiteSpace: 'nowrap' }}>
                        {p.assigned_name || '-'}
                      </td>
                      <td style={{ ...td, fontSize: 12, color: '#aeaeb2', whiteSpace: 'nowrap' }}>{formatTime(p.created_at)}</td>
                      <td style={{ ...td, textAlign: 'center' }} onClick={e => e.stopPropagation()}>
                        <div style={{ display: 'flex', gap: 4, justifyContent: 'center' }}>
                          {p.status === 'pending' && canOperate && (
                            <button title="启动" onClick={() => handleStart(p)}
                              disabled={operating === p.id}
                              style={{ width: 28, height: 28, borderRadius: 7, border: '1px solid rgba(52,199,89,0.3)', background: 'rgba(52,199,89,0.1)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#34c759', padding: 0 }}>
                              {operating === p.id ? <Loader size={12} style={{ animation: 'spin 1s linear infinite' }} /> : <Play size={12} />}
                            </button>
                          )}
                          {canMarkPassed(p) && canOperate && (
                            <button title="快捷通过（评估达标）" onClick={() => handleMarkPassed(p)}
                              style={{ width: 28, height: 28, borderRadius: 7, border: '1px solid rgba(52,199,89,0.3)', background: 'rgba(52,199,89,0.1)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#34c759', padding: 0 }}>
                              <Zap size={12} />
                            </button>
                          )}
                          {(p.status === 'pending' || p.status === 'running') && isAdmin && (
                            <button title="取消" onClick={() => handleCancel(p)}
                              style={{ width: 28, height: 28, borderRadius: 7, border: '1px solid rgba(255,149,0,0.3)', background: 'rgba(255,149,0,0.1)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#ff9500', padding: 0 }}>
                              <Square size={12} />
                            </button>
                          )}
                          {p.status !== 'running' && isAdmin && (
                            <button title="删除" onClick={() => handleDelete(p)}
                              style={{ width: 28, height: 28, borderRadius: 7, border: '1px solid rgba(255,59,48,0.2)', background: 'rgba(255,59,48,0.06)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#ff3b30', padding: 0 }}>
                              <Trash2 size={12} />
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {/* v44新增：分页器 */}
            <Pagination
              currentPage={currentPage}
              totalPages={totalPages}
              totalItems={filteredTotal}
              pageSize={pageSize}
              onPageChange={setCurrentPage}
              onPageSizeChange={setPageSize}
            />
          </>
        )}
      </div>

      {/* 底部浮动批量操作栏 */}
      {canOperate && selectedIds.size > 0 && (
        <div style={{
          position: 'fixed', bottom: 24, left: '50%', transform: 'translateX(-50%)',
          background: 'rgba(30,30,30,0.95)', backdropFilter: 'blur(20px)',
          borderRadius: 16, padding: '12px 24px', display: 'flex', alignItems: 'center', gap: 16,
          boxShadow: '0 8px 32px rgba(0,0,0,0.3)', zIndex: 900, color: '#fff',
        }}>
          <span style={{ fontSize: 13, fontWeight: 500 }}>
            已选 <span style={{ fontWeight: 700, color: '#007aff' }}>{selectedIds.size}</span> 个
          </span>
          {selectedPendingCount > 0 && (
            <button onClick={handleBatchStart} style={{
              padding: '8px 18px', borderRadius: 10, border: 'none',
              background: '#34c759', color: '#fff', fontSize: 13, fontWeight: 600,
              cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6,
            }}>
              <Rocket size={14} /> 批量启动 ({selectedPendingCount})
            </button>
          )}
          {isAdmin && (
            <button onClick={handleBatchRestartGenerator} style={{
              padding: '8px 18px', borderRadius: 10, border: 'none',
              background: '#ff9500', color: '#fff', fontSize: 13, fontWeight: 600,
              cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6,
            }}>
              <RotateCcw size={14} /> 批量重跑Generator
            </button>
          )}
          {isAdmin && (
            <button onClick={openAssignDialog} style={{
              padding: '8px 18px', borderRadius: 10, border: 'none',
              background: '#5856d6', color: '#fff', fontSize: 13, fontWeight: 600,
              cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6,
            }}>
              <UserPlus size={14} /> 分配审核员
            </button>
          )}
          <button onClick={() => setSelectedIds(new Set())} style={{
            padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(255,255,255,0.2)',
            background: 'transparent', color: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
          }}>
            取消选择
          </button>
        </div>
      )}

      {showCreate && <CreateDialog onClose={() => setShowCreate(false)} onCreate={handleCreate} />}
      {showBatchCreate && <BatchCreateDialog onClose={() => setShowBatchCreate(false)} onBatchCreate={handleBatchCreate} />}

      {/* 分配审核员弹窗 */}
      {showAssignDialog && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
          onClick={e => { if (e.target === e.currentTarget && !assigning) setShowAssignDialog(false) }}>
          <div style={{ background: '#fff', borderRadius: 20, width: 440, maxWidth: '94vw', padding: 28, boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 20 }}>
              <UserPlus size={20} color="#5856d6" />
              <div style={{ fontSize: 18, fontWeight: 700, color: '#1c1c1e', flex: 1 }}>分配审核员</div>
              <button onClick={() => !assigning && setShowAssignDialog(false)} style={{
                background: '#f2f2f7', border: 'none', borderRadius: '50%', width: 30, height: 30,
                display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer',
              }}><XIcon size={16} color="#8e8e93" /></button>
            </div>
            <div style={{ fontSize: 13, color: '#86868b', marginBottom: 16 }}>
              将 <span style={{ fontWeight: 600, color: '#1c1c1e' }}>{selectedIds.size}</span> 个Pipeline分配给指定审核员
            </div>
            <div style={{ marginBottom: 20 }}>
              <label style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e', display: 'block', marginBottom: 8 }}>选择审核员</label>
              {operators.length === 0 ? (
                <div style={{ fontSize: 13, color: '#aeaeb2', padding: '12px 0' }}>暂无可分配的审核员</div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {operators.map(op => (
                    <div key={op.id} onClick={() => setSelectedOperator(op.id)} style={{
                      display: 'flex', alignItems: 'center', gap: 12, padding: '10px 14px', borderRadius: 10, cursor: 'pointer',
                      border: selectedOperator === op.id ? '2px solid #5856d6' : '1px solid rgba(0,0,0,0.08)',
                      background: selectedOperator === op.id ? 'rgba(88,86,214,0.05)' : '#fff',
                    }}>
                      <div style={{
                        width: 32, height: 32, borderRadius: '50%',
                        background: selectedOperator === op.id ? '#5856d6' : 'rgba(88,86,214,0.1)',
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        color: selectedOperator === op.id ? '#fff' : '#5856d6', fontSize: 13, fontWeight: 600,
                      }}>{op.display_name.charAt(0)}</div>
                      <div style={{ flex: 1 }}>
                        <div style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>{op.display_name}</div>
                        <div style={{ fontSize: 11, color: '#86868b' }}>{op.username} · {op.role === 'admin' ? '管理员' : op.role === 'senior_operator' ? '高级操作员' : '操作员'}</div>
                      </div>
                      {selectedOperator === op.id && <CheckCircleIcon size={18} color="#5856d6" />}
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
              <button onClick={() => !assigning && setShowAssignDialog(false)} style={{
                padding: '10px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
                background: '#fff', fontSize: 14, fontWeight: 500, cursor: 'pointer', color: '#3c3c43',
              }}>取消</button>
              <button onClick={handleBatchAssign} disabled={!selectedOperator || assigning} style={{
                padding: '10px 24px', borderRadius: 10, border: 'none',
                background: selectedOperator && !assigning ? '#5856d6' : '#c7c7cc',
                color: '#fff', fontSize: 14, fontWeight: 600,
                cursor: selectedOperator && !assigning ? 'pointer' : 'not-allowed',
              }}>{assigning ? '分配中...' : '确认分配'}</button>
            </div>
          </div>
        </div>
      )}

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  )
}
