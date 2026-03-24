/**
 * Pipeline列表页面
 * P4.5-A增强: 从卡片列表改为数据表格视图
 * P4.5-D增强: 新增快捷通过按钮
 * P5-3增强: 新增批量创建+批量启动+多选操作
 * Apple风格内联CSS
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getPipelines, createPipeline, startPipeline, cancelPipeline, deletePipeline,
  markPassed, batchCreatePipelines, batchStartPipelines,
  type PipelineListItem, type CreatePipelineRequest,
} from '@/api/pipelines'
import {
  Workflow, Play, Square, Trash2, RefreshCw, Plus,
  CheckCircle, XCircle, Clock, AlertTriangle, Loader, Eye, Zap,
  Layers, Rocket,
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

// ==================== P5-3新增：批量创建弹窗 ====================
function BatchCreateDialog({ onClose, onBatchCreate }: { onClose: () => void; onBatchCreate: (codes: string[]) => void }) {
  const [inputText, setInputText] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // 解析输入文本为课程编号列表（支持逗号、换行、空格分隔）
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
            <span style={{ marginLeft: 8, color: '#aeaeb2' }}>
              {codes.join(', ')}
            </span>
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

// ==================== 判断是否可快捷通过 ====================
function canMarkPassed(p: PipelineListItem): boolean {
  const allowedStatuses = ['review_queue', 'needs_human', 'failed']
  return allowedStatuses.includes(p.status) && p.meta_score !== null && p.meta_score >= 9.0
}

// ==================== 主页面组件 ====================

export default function PipelinesPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'
  const canOperate = user?.role === 'admin' || user?.role === 'operator'

  const [pipelines, setPipelines] = useState<PipelineListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [showBatchCreate, setShowBatchCreate] = useState(false)
  const [operating, setOperating] = useState<string | null>(null)
  const [toast, setToast] = useState<{ message: string; type: 'ok' | 'err' | 'info' } | null>(null)
  const [statusFilter, setStatusFilter] = useState('')
  // P5-3新增：多选状态
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

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

  // 筛选变化时清空选中
  useEffect(() => { setSelectedIds(new Set()) }, [statusFilter])

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

  // P5-3新增：批量创建
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

  // P5-3新增：批量启动
  const handleBatchStart = async () => {
    const ids = Array.from(selectedIds)
    // 只启动pending状态的
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

  // P5-3：多选操作
  const toggleSelect = (id: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const filteredPipelines = statusFilter
    ? pipelines.filter(p => p.status === statusFilter)
    : pipelines

  const toggleSelectAll = () => {
    if (selectedIds.size === filteredPipelines.length && filteredPipelines.length > 0) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(filteredPipelines.map(p => p.id)))
    }
  }

  const isAllSelected = filteredPipelines.length > 0 && selectedIds.size === filteredPipelines.length

  // 选中的pending数量（用于批量启动按钮显示）
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
  // P5-3：checkbox样式
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
          {loading ? '加载中...' : (statusFilter ? filteredPipelines.length + ' / ' + total + ' 个Pipeline' : total + ' 个Pipeline')}
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

      {/* Pipeline数据表格 */}
      <div style={{
        background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
        border: '1px solid rgba(0,0,0,0.06)', borderRadius: 16, overflow: 'hidden',
      }}>
        {loading ? (
          <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>加载中...</div>
        ) : filteredPipelines.length === 0 ? (
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
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', minWidth: 1050 }}>
              <thead>
                <tr style={{ background: 'rgba(0,0,0,0.02)' }}>
                  {/* P5-3：全选checkbox */}
                  {canOperate && (
                    <th style={{ ...th, width: 36, textAlign: 'center', padding: '10px 8px' }}>
                      <input type="checkbox" checked={isAllSelected} onChange={toggleSelectAll} style={checkboxStyle} />
                    </th>
                  )}
                  <th style={th}>课程编号</th>
                  <th style={th}>课程名称</th>
                  <th style={th}>状态</th>
                  <th style={th}>当前步骤</th>
                  <th style={{ ...th, textAlign: 'center' }}>评估均分</th>
                  <th style={{ ...th, textAlign: 'center' }}>Meta分</th>
                  <th style={{ ...th, textAlign: 'center' }}>翻译分</th>
                  <th style={th}>进度</th>
                  <th style={th}>分配给</th>
                  <th style={th}>创建时间</th>
                  <th style={{ ...th, textAlign: 'center' }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {filteredPipelines.map(p => (
                  <tr key={p.id}
                    style={{ cursor: 'pointer', transition: 'background 0.15s ease', background: selectedIds.has(p.id) ? 'rgba(0,122,255,0.04)' : 'transparent' }}
                    onClick={() => navigate('/pipelines/' + p.id)}
                    onMouseEnter={e => { if (!selectedIds.has(p.id)) (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.03)' }}
                    onMouseLeave={e => { if (!selectedIds.has(p.id)) (e.currentTarget as HTMLElement).style.background = 'transparent' }}
                  >
                    {/* P5-3：行选择checkbox */}
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
        )}
      </div>

      {/* P5-3新增：底部浮动批量操作栏 */}
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
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  )
}
