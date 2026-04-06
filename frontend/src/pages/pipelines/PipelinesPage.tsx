/**
 * PipelinesPage — Pipeline列表页（主文件）
 *
 * 功能：
 *   - 统计卡片（总数/运行中/待审核/失败/达标）
 *   - 状态筛选 + 分配人筛选
 *   - 表格：排序+分页+多选+行操作
 *   - 批量创建/批量启动/批量重跑Generator/批量分配
 *   - 快捷通过（评估达标时）
 *
 * 子组件均从 ./components/ 引入，本文件只保留：
 *   - 全局状态 + useMemo排序分页逻辑
 *   - 所有操作函数（handleCreate/handleStart/...）
 *   - 页面级渲染框架
 */
import { useState, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getPipelines, createPipeline, startPipeline, cancelPipeline, deletePipeline,
  markPassed, batchCreatePipelines, batchStartPipelines, getOperators,
  batchAssignPipelines, batchRestartFromStep,
  type PipelineListItem, type CreatePipelineRequest, type OperatorInfo,
} from '@/api/pipelines'
import {
  Workflow, Play, Square, Trash2, RefreshCw, Plus, Loader, Eye, Zap, Layers,
} from 'lucide-react'

// ---- 子组件 ----
import {
  FILTER_OPTIONS, STATUS_SORT_ORDER, formatTime, canMarkPassed,
  type SortField, type SortDirection,
} from './components/pipelinesConstants'
import { StatusBadge, ProgressBar, ScoreCell, SortableHeader, Pagination } from './components/PipelineTableParts'
import { CreateDialog, BatchCreateDialog, AssignDialog } from './components/PipelineDialogs'
import { PipelineBatchBar } from './components/PipelineBatchBar'

// ==================== Toast（局部，仅本页使用）====================
function Toast({ message, type, onClose }: { message: string; type: 'ok' | 'err' | 'info'; onClose: () => void }) {
  useEffect(() => { const t = setTimeout(onClose, 5000); return () => clearTimeout(t) }, [onClose])
  const bg = type === 'ok' ? '#34c759' : type === 'err' ? '#ff3b30' : '#007aff'
  return (
    <div style={{ position: 'fixed', bottom: 24, right: 24, background: bg, color: '#fff', padding: '12px 22px', borderRadius: 12, fontSize: 13, fontWeight: 500, zIndex: 9999, boxShadow: '0 4px 24px rgba(0,0,0,0.18)', maxWidth: 500 }}>
      {message}
    </div>
  )
}

// ==================== 主组件 ====================
export default function PipelinesPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const isAdmin    = user?.role === 'admin' || user?.role === 'senior_operator'
  const canOperate = user?.role === 'admin' || user?.role === 'operator' || user?.role === 'senior_operator'

  // ---- 数据状态 ----
  const [pipelines, setPipelines] = useState<PipelineListItem[]>([])
  const [loading, setLoading]     = useState(true)
  const [operating, setOperating] = useState<string | null>(null)
  const [toast, setToast]         = useState<{ message: string; type: 'ok' | 'err' | 'info' } | null>(null)

  // ---- 弹窗状态 ----
  const [showCreate, setShowCreate]           = useState(false)
  const [showBatchCreate, setShowBatchCreate] = useState(false)
  const [showAssignDialog, setShowAssignDialog] = useState(false)
  const [operators, setOperators]             = useState<OperatorInfo[]>([])
  const [selectedOperator, setSelectedOperator] = useState('')
  const [assigning, setAssigning]             = useState(false)

  // ---- 筛选状态 ----
  const [statusFilter, setStatusFilter]   = useState('')
  const [assigneeFilter, setAssigneeFilter] = useState('')

  // ---- 排序状态（默认按创建时间倒序）----
  const [sortField, setSortField] = useState<SortField>('created_at')
  const [sortDir, setSortDir]     = useState<SortDirection>('desc')

  // ---- 分页状态 ----
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize, setPageSize]       = useState(50)

  // ---- 多选状态 ----
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

  // ==================== 数据加载 ====================

  const loadPipelines = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getPipelines()
      setPipelines(data.pipelines || [])
    } catch (e: unknown) {
      setToast({ message: '加载失败: ' + (e instanceof Error ? e.message : ''), type: 'err' })
    }
    setLoading(false)
  }, [])

  useEffect(() => { loadPipelines() }, [loadPipelines])

  // 筛选/排序变化时回到第1页并清空选中
  useEffect(() => { setSelectedIds(new Set()); setCurrentPage(1) }, [statusFilter, sortField, sortDir, assigneeFilter])
  // 每页条数变化时回到第1页
  useEffect(() => { setCurrentPage(1) }, [pageSize])

  // ==================== 排序逻辑 ====================

  /**
   * 点击表头切换排序
   * - 点击当前字段：切换升/降序
   * - 点击新字段：降序（数值大的在前更常用）
   * - 课程编号例外：默认升序（字母序更直观）
   */
  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(prev => prev === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDir(field === 'course_code' ? 'asc' : 'desc')
    }
  }

  // ==================== 筛选→排序→分页管道（useMemo缓存）====================

  const { pagedPipelines, filteredTotal, totalPages } = useMemo(() => {
    // 1. 状态筛选
    let filtered = statusFilter
      ? pipelines.filter(p => p.status === statusFilter)
      : pipelines

    // 2. 分配人筛选
    if (assigneeFilter === 'unassigned') {
      filtered = filtered.filter(p => !p.assigned_name)
    } else if (assigneeFilter) {
      filtered = filtered.filter(p => p.assigned_name === assigneeFilter)
    }

    // 3. 排序
    const sorted = [...filtered].sort((a, b) => {
      let cmp = 0
      switch (sortField) {
        case 'course_code':
          cmp = a.course_code.localeCompare(b.course_code, 'zh-CN', { numeric: true }); break
        case 'status':
          cmp = (STATUS_SORT_ORDER[a.status] || 99) - (STATUS_SORT_ORDER[b.status] || 99); break
        case 'eval_avg_score':
          cmp = (a.eval_avg_score ?? -1) - (b.eval_avg_score ?? -1); break
        case 'meta_score':
          cmp = (a.meta_score ?? -1) - (b.meta_score ?? -1); break
        case 'translator_score':
          cmp = (a.translator_score ?? -1) - (b.translator_score ?? -1); break
        case 'progress': {
          const pa = a.steps_total > 0 ? a.steps_completed / a.steps_total : 0
          const pb = b.steps_total > 0 ? b.steps_completed / b.steps_total : 0
          cmp = pa - pb; break
        }
        case 'created_at':
          cmp = new Date(a.created_at || 0).getTime() - new Date(b.created_at || 0).getTime(); break
      }
      return sortDir === 'asc' ? cmp : -cmp
    })

    // 4. 分页
    const total  = sorted.length
    const pages  = Math.max(1, Math.ceil(total / pageSize))
    const start  = (currentPage - 1) * pageSize
    const paged  = sorted.slice(start, start + pageSize)
    return { pagedPipelines: paged, filteredTotal: total, totalPages: pages }
  }, [pipelines, statusFilter, sortField, sortDir, currentPage, pageSize, assigneeFilter])

  // ==================== 操作函数 ====================

  const handleCreate = async (req: CreatePipelineRequest) => {
    try {
      await createPipeline(req)
      setToast({ message: req.course_code + ' Pipeline创建成功', type: 'ok' })
      setShowCreate(false)
      loadPipelines()
    } catch (e: unknown) {
      setToast({ message: '创建失败: ' + (e instanceof Error ? e.message : ''), type: 'err' })
    }
  }

  const handleBatchCreate = async (codes: string[]) => {
    try {
      const result = await batchCreatePipelines(codes)
      const parts: string[] = []
      if (result.created_ids.length > 0) parts.push('成功创建 ' + result.created_ids.length + ' 个')
      if (result.skipped_codes.length > 0) parts.push('跳过 ' + result.skipped_codes.length + ' 个')
      if (result.failed_codes.length > 0)  parts.push('失败 ' + result.failed_codes.length + ' 个')
      setToast({ message: '批量创建完成: ' + parts.join(', '), type: result.failed_codes.length > 0 ? 'err' : 'ok' })
      setShowBatchCreate(false)
      loadPipelines()
    } catch (e: unknown) {
      setToast({ message: '批量创建失败: ' + (e instanceof Error ? e.message : ''), type: 'err' })
    }
  }

  const handleBatchStart = async () => {
    const ids = Array.from(selectedIds)
    const pendingIds = ids.filter(id => pipelines.find(pp => pp.id === id)?.status === 'pending')
    if (pendingIds.length === 0) { setToast({ message: '选中的Pipeline中没有待启动的', type: 'info' }); return }
    if (!confirm('确认批量启动 ' + pendingIds.length + ' 个Pipeline？\n每个Pipeline约需10-50分钟执行。')) return
    try {
      const result = await batchStartPipelines(pendingIds)
      const parts: string[] = []
      if (result.started_ids.length > 0) parts.push('已提交 ' + result.started_ids.length + ' 个')
      if (result.skipped_ids.length > 0) parts.push('跳过 ' + result.skipped_ids.length + ' 个')
      if (result.failed_ids.length > 0)  parts.push('失败 ' + result.failed_ids.length + ' 个')
      setToast({ message: '批量启动完成: ' + parts.join(', '), type: result.failed_ids.length > 0 ? 'err' : 'ok' })
      setSelectedIds(new Set()); loadPipelines()
    } catch (e: unknown) {
      setToast({ message: '批量启动失败: ' + (e instanceof Error ? e.message : ''), type: 'err' })
    }
  }

  const handleBatchRestartGenerator = async () => {
    const ids = Array.from(selectedIds)
    if (ids.length === 0) { setToast({ message: '请先选择要重跑的Pipeline', type: 'info' }); return }
    if (ids.length > 50)  { setToast({ message: '单次批量重跑上限50个Pipeline', type: 'err' }); return }
    if (!confirm('确认批量从Generator步骤重跑选中的 ' + ids.length + ' 个Pipeline？\n\n此操作将：\n• 重置Generator及后续步骤的执行数据\n• 清空已生成的页面\n• 重置审核和验收步骤\n• 每个Pipeline约需10-30分钟执行\n\n已完成的前序步骤（数据检查、评估、Meta、翻译）不受影响。')) return
    try {
      const result = await batchRestartFromStep(ids, 'generator')
      const parts: string[] = []
      if (result.success_count > 0)      parts.push('成功提交 ' + result.success_count + ' 个')
      if (result.skipped_ids.length > 0) parts.push('跳过 ' + result.skipped_ids.length + ' 个')
      if (result.failed_ids.length > 0)  parts.push('失败 ' + result.failed_ids.length + ' 个')
      setToast({ message: '批量重跑完成: ' + parts.join(', '), type: result.failed_ids.length > 0 ? 'err' : 'ok' })
      setSelectedIds(new Set()); loadPipelines()
    } catch (e: unknown) {
      setToast({ message: '批量重跑失败: ' + (e instanceof Error ? e.message : ''), type: 'err' })
    }
  }

  const openAssignDialog = async () => {
    if (selectedIds.size === 0) return
    try { const ops = await getOperators(); setOperators(ops || []) } catch { /* ignore */ }
    setSelectedOperator(''); setShowAssignDialog(true)
  }

  const handleBatchAssign = async () => {
    if (!selectedOperator) return
    setAssigning(true)
    try {
      const result = await batchAssignPipelines(Array.from(selectedIds), selectedOperator)
      setToast({ message: '分配成功: ' + result.success_count + ' 个Pipeline已分配给 ' + result.assigned_name, type: 'ok' })
      setShowAssignDialog(false); setSelectedIds(new Set()); loadPipelines()
    } catch (e: unknown) {
      setToast({ message: '分配失败: ' + (e instanceof Error ? e.message : ''), type: 'err' })
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
    } catch (e: unknown) {
      setToast({ message: '启动失败: ' + (e instanceof Error ? e.message : ''), type: 'err' })
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
    } catch (e: unknown) {
      setToast({ message: '取消失败: ' + (e instanceof Error ? e.message : ''), type: 'err' })
    }
  }

  const handleDelete = async (p: PipelineListItem) => {
    if (!confirm('确认删除 ' + p.course_code + ' Pipeline？\n此操作不可恢复。')) return
    try {
      await deletePipeline(p.id)
      setToast({ message: p.course_code + ' 已删除', type: 'ok' })
      loadPipelines()
    } catch (e: unknown) {
      setToast({ message: '删除失败: ' + (e instanceof Error ? e.message : ''), type: 'err' })
    }
  }

  const handleMarkPassed = async (p: PipelineListItem) => {
    if (!confirm('确认快捷通过 ' + p.course_code + ' Pipeline？\n将跳过审核流程直接标记为已定稿。')) return
    try {
      await markPassed(p.id)
      setToast({ message: p.course_code + ' 已快捷通过并归档', type: 'ok' })
      loadPipelines()
    } catch (e: unknown) {
      setToast({ message: '快捷通过失败: ' + (e instanceof Error ? e.message : ''), type: 'err' })
    }
  }

  // ---- 多选操作 ----
  const toggleSelect = (id: string) => {
    setSelectedIds(prev => { const next = new Set(prev); next.has(id) ? next.delete(id) : next.add(id); return next })
  }

  // 全选/取消全选（基于当前页）
  const toggleSelectAll = () => {
    const currentPageIds = pagedPipelines.map(p => p.id)
    const allSelected    = currentPageIds.length > 0 && currentPageIds.every(id => selectedIds.has(id))
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (allSelected) currentPageIds.forEach(id => next.delete(id))
      else             currentPageIds.forEach(id => next.add(id))
      return next
    })
  }

  const isAllSelected = pagedPipelines.length > 0 && pagedPipelines.every(p => selectedIds.has(p.id))
  const selectedPendingCount = Array.from(selectedIds).filter(id => pipelines.find(pp => pp.id === id)?.status === 'pending').length

  // ---- 统计 ----
  const total       = pipelines.length
  const running     = pipelines.filter(p => p.status === 'running').length
  const reviewQueue = pipelines.filter(p => p.status === 'review_queue').length
  const failed      = pipelines.filter(p => p.status === 'failed').length
  const passedCount = pipelines.filter(p => p.eval_avg_score !== null && p.eval_avg_score >= 9.0).length

  // ---- 样式常量 ----
  const stat: React.CSSProperties = { background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)', borderRadius: 14, padding: '16px 20px', flex: 1, minWidth: 100 }
  const btn: React.CSSProperties  = { padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6, transition: 'all 0.15s ease' }
  const btnP: React.CSSProperties = { ...btn, background: '#007aff', color: '#fff', border: '1px solid #007aff' }
  const th: React.CSSProperties   = { padding: '10px 12px', textAlign: 'left', fontSize: 11, fontWeight: 600, color: '#8e8e93', textTransform: 'uppercase', letterSpacing: '0.02em', borderBottom: '1px solid rgba(0,0,0,0.06)', whiteSpace: 'nowrap' }
  const td: React.CSSProperties   = { padding: '12px 12px', fontSize: 13, color: '#1c1c1e', borderBottom: '1px solid rgba(0,0,0,0.04)', verticalAlign: 'middle' }
  const checkboxStyle: React.CSSProperties = { width: 16, height: 16, cursor: 'pointer', accentColor: '#007aff' }

  // ==================== 渲染 ====================
  return (
    <div>
      {/* ---- 统计卡片 ---- */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap' }}>
        {[
          { label: '总Pipeline', value: total,       color: '#1c1c1e' },
          { label: '运行中',     value: running,     color: '#007aff' },
          { label: '待审核',     value: reviewQueue, color: '#ff9500' },
          { label: '失败',       value: failed,      color: '#ff3b30' },
          { label: '达标(≥9.0)', value: passedCount, color: '#34c759' },
        ].map(s => (
          <div key={s.label} style={stat}>
            <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 600, marginBottom: 4 }}>{s.label}</div>
            <div style={{ fontSize: 28, fontWeight: 700, color: s.color }}>{s.value}</div>
          </div>
        ))}
      </div>

      {/* ---- 操作栏 ---- */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12, flexWrap: 'wrap', gap: 8 }}>
        <div style={{ fontSize: 13, color: '#8e8e93' }}>
          {loading ? '加载中...' : (statusFilter ? filteredTotal + ' / ' + total + ' 个Pipeline' : total + ' 个Pipeline')}
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button style={btn} onClick={loadPipelines}><RefreshCw size={14} /> 刷新</button>
          {canOperate && (
            <>
              <button style={{ ...btn, color: '#5856d6', borderColor: 'rgba(88,86,214,0.3)' }} onClick={() => setShowBatchCreate(true)}>
                <Layers size={14} /> 批量创建
              </button>
              <button style={btnP} onClick={() => setShowCreate(true)}>
                <Plus size={14} /> 创建Pipeline
              </button>
            </>
          )}
        </div>
      </div>

      {/* ---- 状态筛选按钮行 ---- */}
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
            {opt.value && <span style={{ marginLeft: 4, opacity: 0.6 }}>{pipelines.filter(p => p.status === opt.value).length}</span>}
          </button>
        ))}
      </div>

      {/* ---- 分配人筛选行（有分配人数据时才显示）---- */}
      {(() => {
        const assigneeNames   = Array.from(new Set(pipelines.map(p => p.assigned_name).filter(Boolean))).sort()
        const unassignedCount = pipelines.filter(p => !p.assigned_name).length
        if (assigneeNames.length === 0) return null
        const filterBtn = (active: boolean, color: string): React.CSSProperties => ({
          padding: '5px 12px', borderRadius: 16, fontSize: 12, fontWeight: 500, cursor: 'pointer',
          border: active ? `1px solid ${color}` : '1px solid rgba(0,0,0,0.08)',
          background: active ? `${color}18` : '#fff',
          color: active ? color : '#3c3c43',
          transition: 'all 0.15s ease',
        })
        return (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 16, flexWrap: 'wrap' }}>
            <span style={{ fontSize: 12, color: '#8e8e93', fontWeight: 500, marginRight: 2 }}>分配给：</span>
            <button onClick={() => setAssigneeFilter('')}              style={filterBtn(!assigneeFilter, '#5856d6')}>全部</button>
            <button onClick={() => setAssigneeFilter('unassigned')}    style={filterBtn(assigneeFilter === 'unassigned', '#ff9500')}>
              未分配 <span style={{ opacity: 0.6, marginLeft: 2 }}>{unassignedCount}</span>
            </button>
            {assigneeNames.map(name => (
              <button key={name} onClick={() => setAssigneeFilter(name!)} style={filterBtn(assigneeFilter === name, '#5856d6')}>
                {name} <span style={{ opacity: 0.6, marginLeft: 2 }}>{pipelines.filter(p => p.assigned_name === name).length}</span>
              </button>
            ))}
          </div>
        )
      })()}

      {/* ---- Pipeline数据表格 ---- */}
      <div style={{ background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)', borderRadius: 16, overflow: 'hidden' }}>
        {loading ? (
          <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>加载中...</div>
        ) : filteredTotal === 0 ? (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Workflow size={40} style={{ color: '#c7c7cc', marginBottom: 12 }} />
            <div style={{ color: '#8e8e93', fontSize: 14 }}>
              {statusFilter ? '当前筛选条件下暂无Pipeline' : '暂无Pipeline'}
            </div>
            {canOperate && !statusFilter && (
              <div style={{ color: '#007aff', fontSize: 13, marginTop: 8, cursor: 'pointer' }} onClick={() => setShowCreate(true)}>
                创建第一个Pipeline →
              </div>
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
                    <th style={th}><SortableHeader label="课程编号"  field="course_code"     currentSort={sortField} currentDir={sortDir} onSort={handleSort} /></th>
                    <th style={th}>课程名称</th>
                    <th style={th}><SortableHeader label="状态"      field="status"          currentSort={sortField} currentDir={sortDir} onSort={handleSort} /></th>
                    <th style={th}>当前步骤</th>
                    <th style={{ ...th, textAlign: 'center' }}><SortableHeader label="评估均分" field="eval_avg_score"  currentSort={sortField} currentDir={sortDir} onSort={handleSort} align="center" /></th>
                    <th style={{ ...th, textAlign: 'center' }}><SortableHeader label="Meta分"  field="meta_score"      currentSort={sortField} currentDir={sortDir} onSort={handleSort} align="center" /></th>
                    <th style={{ ...th, textAlign: 'center' }}><SortableHeader label="翻译分"  field="translator_score" currentSort={sortField} currentDir={sortDir} onSort={handleSort} align="center" /></th>
                    <th style={th}><SortableHeader label="进度"      field="progress"        currentSort={sortField} currentDir={sortDir} onSort={handleSort} /></th>
                    <th style={th}>分配给</th>
                    <th style={th}><SortableHeader label="创建时间"  field="created_at"      currentSort={sortField} currentDir={sortDir} onSort={handleSort} /></th>
                    <th style={{ ...th, textAlign: 'center' }}>操作</th>
                  </tr>
                </thead>
                <tbody>
                  {pagedPipelines.map(p => (
                    <tr key={p.id}
                      style={{ cursor: 'pointer', transition: 'background 0.15s ease', background: selectedIds.has(p.id) ? 'rgba(0,122,255,0.04)' : 'transparent' }}
                      onClick={() => navigate('/workflow/pipelines/' + p.id)}
                      onMouseEnter={e => { if (!selectedIds.has(p.id)) (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.03)' }}
                      onMouseLeave={e => { if (!selectedIds.has(p.id)) (e.currentTarget as HTMLElement).style.background = 'transparent' }}>
                      {canOperate && (
                        <td style={{ ...td, width: 36, textAlign: 'center', padding: '12px 8px' }} onClick={e => e.stopPropagation()}>
                          <input type="checkbox" checked={selectedIds.has(p.id)} onChange={() => toggleSelect(p.id)} style={checkboxStyle} />
                        </td>
                      )}
                      <td style={{ ...td, fontWeight: 600, whiteSpace: 'nowrap' }}>{p.course_code}</td>
                      <td style={{ ...td, color: '#8e8e93', maxWidth: 180, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {p.course_name || p.course_code}
                      </td>
                      <td style={td}><StatusBadge status={p.status} statusName={p.status_name} reviewRound={p.review_round} /></td>
                      <td style={{ ...td, fontSize: 12, color: '#636366', whiteSpace: 'nowrap' }}>{p.current_step_name}</td>
                      <td style={{ ...td, textAlign: 'center' }}><ScoreCell value={p.eval_avg_score} /></td>
                      <td style={{ ...td, textAlign: 'center' }}><ScoreCell value={p.meta_score} /></td>
                      <td style={{ ...td, textAlign: 'center' }}><ScoreCell value={p.translator_score} /></td>
                      <td style={{ ...td, minWidth: 100 }}><ProgressBar completed={p.steps_completed} total={p.steps_total} /></td>
                      <td style={{ ...td, fontSize: 11, color: '#8e8e93', textAlign: 'center', whiteSpace: 'nowrap' }}>
                        {p.review_round >= 2 ? <span style={{ fontWeight: 600, color: '#ff9500', background: 'rgba(255,149,0,0.1)', padding: '1px 6px', borderRadius: 4, fontSize: 10 }}>2审</span> : <span style={{ color: '#c7c7cc' }}>1审</span>}
                      </td>
                      <td style={{ ...td, fontSize: 12, color: p.assigned_name ? '#5856d6' : '#c7c7cc', whiteSpace: 'nowrap' }}>
                        {p.assigned_name || '-'}
                      </td>
                      <td style={{ ...td, fontSize: 12, color: '#aeaeb2', whiteSpace: 'nowrap' }}>{formatTime(p.created_at)}</td>
                      {/* 行操作按钮 */}
                      <td style={{ ...td, textAlign: 'center' }} onClick={e => e.stopPropagation()}>
                        <div style={{ display: 'flex', gap: 4, justifyContent: 'center' }}>
                          {p.status === 'pending' && canOperate && (
                            <button title="启动" onClick={() => handleStart(p)} disabled={operating === p.id}
                              style={{ width: 28, height: 28, borderRadius: 7, border: '1px solid rgba(52,199,89,0.3)', background: 'rgba(52,199,89,0.1)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#34c759', padding: 0 }}>
                              {operating === p.id ? <Loader size={12} style={{ animation: 'spin 1s linear infinite' }} /> : <Play size={12} />}
                            </button>
                          )}
                          {canMarkPassed(p.status, p.meta_score) && canOperate && (
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
                          {/* 眼睛图标（始终可见，跳转详情）*/}
                          <button title="查看详情" onClick={() => navigate('/workflow/pipelines/' + p.id)}
                            style={{ width: 28, height: 28, borderRadius: 7, border: '1px solid rgba(0,0,0,0.08)', background: 'rgba(0,0,0,0.03)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#8e8e93', padding: 0 }}>
                            <Eye size={12} />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {/* 分页器 */}
            <Pagination
              currentPage={currentPage} totalPages={totalPages}
              totalItems={filteredTotal} pageSize={pageSize}
              onPageChange={setCurrentPage} onPageSizeChange={setPageSize}
            />
          </>
        )}
      </div>

      {/* ---- 底部浮动批量操作栏 ---- */}
      {canOperate && (
        <PipelineBatchBar
          selectedCount={selectedIds.size}
          selectedPendingCount={selectedPendingCount}
          isAdmin={isAdmin}
          onBatchStart={handleBatchStart}
          onBatchRestart={handleBatchRestartGenerator}
          onOpenAssign={openAssignDialog}
          onClearSelection={() => setSelectedIds(new Set())}
        />
      )}

      {/* ---- 弹窗 ---- */}
      {showCreate     && <CreateDialog onClose={() => setShowCreate(false)} onCreate={handleCreate} />}
      {showBatchCreate && <BatchCreateDialog onClose={() => setShowBatchCreate(false)} onBatchCreate={handleBatchCreate} />}
      {showAssignDialog && (
        <AssignDialog
          selectedCount={selectedIds.size}
          operators={operators}
          selectedOperator={selectedOperator}
          assigning={assigning}
          onSelectOperator={setSelectedOperator}
          onConfirm={handleBatchAssign}
          onClose={() => setShowAssignDialog(false)}
        />
      )}

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  )
}
