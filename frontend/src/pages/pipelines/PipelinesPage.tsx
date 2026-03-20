/**
 * Pipeline列表页面
 * P4-7: 展示所有Pipeline，含状态、课程、进度、操作按钮
 * Apple风格内联CSS
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getPipelines, createPipeline, startPipeline, cancelPipeline, deletePipeline,
  type PipelineListItem, type CreatePipelineRequest,
} from '@/api/pipelines'
import {
  Workflow, Play, Square, Trash2, RefreshCw, Plus, ChevronRight,
  CheckCircle, XCircle, Clock, AlertTriangle, Loader, Eye,
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

// ==================== 状态徽章 ====================
function StatusBadge({ status, statusName }: { status: string; statusName: string }) {
  const colorMap: Record<string, { bg: string; fg: string }> = {
    pending:      { bg: 'rgba(142,142,147,0.12)', fg: '#8e8e93' },
    running:      { bg: 'rgba(0,122,255,0.12)', fg: '#007aff' },
    review_queue: { bg: 'rgba(255,149,0,0.12)', fg: '#ff9500' },
    finalized:    { bg: 'rgba(52,199,89,0.12)', fg: '#34c759' },
    needs_human:  { bg: 'rgba(255,204,0,0.12)', fg: '#cc9900' },
    failed:       { bg: 'rgba(255,59,48,0.12)', fg: '#ff3b30' },
    cancelled:    { bg: 'rgba(142,142,147,0.08)', fg: '#aeaeb2' },
  }
  const c = colorMap[status] || colorMap.pending
  const iconMap: Record<string, React.ReactNode> = {
    pending:      <Clock size={12} />,
    running:      <Loader size={12} style={{ animation: 'spin 1s linear infinite' }} />,
    review_queue: <Eye size={12} />,
    finalized:    <CheckCircle size={12} />,
    needs_human:  <AlertTriangle size={12} />,
    failed:       <XCircle size={12} />,
    cancelled:    <Square size={12} />,
  }
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 4,
      padding: '3px 10px', borderRadius: 20, fontSize: 12, fontWeight: 500,
      background: c.bg, color: c.fg,
    }}>
      {iconMap[status]}{statusName}
    </span>
  )
}

// ==================== 进度条 ====================
function ProgressBar({ completed, total }: { completed: number; total: number }) {
  const pct = total > 0 ? (completed / total) * 100 : 0
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <div style={{ flex: 1, height: 6, background: 'rgba(0,0,0,0.06)', borderRadius: 3, overflow: 'hidden' }}>
        <div style={{ width: pct + '%', height: '100%', background: completed === total ? '#34c759' : '#007aff', borderRadius: 3, transition: 'width 0.3s ease' }} />
      </div>
      <span style={{ fontSize: 11, color: '#8e8e93', fontWeight: 500, whiteSpace: 'nowrap' }}>{completed}/{total}</span>
    </div>
  )
}

// ==================== 创建弹窗 ====================
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

        {/* 课程编号（必填） */}
        <div style={{ marginBottom: 16 }}>
          <label style={labelStyle}>课程编号 *</label>
          <input style={inputStyle} placeholder="如 G1-01" value={courseCode}
            onChange={e => setCourseCode(e.target.value)}
            onFocus={e => (e.target.style.borderColor = '#007aff')}
            onBlur={e => (e.target.style.borderColor = 'rgba(0,0,0,0.1)')} />
        </div>

        {/* 配置参数（两列布局） */}
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

        {/* 操作按钮 */}
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

// ==================== 格式化工具函数 ====================

/** 格式化时间为相对或绝对显示 */
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

/** 格式化耗时毫秒为可读字符串 */
function formatDuration(ms: number): string {
  if (ms <= 0) return '-'
  if (ms < 1000) return ms + 'ms'
  if (ms < 60000) return (ms / 1000).toFixed(1) + '秒'
  return (ms / 60000).toFixed(1) + '分钟'
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
  const [operating, setOperating] = useState<string | null>(null) // 正在操作的Pipeline ID
  const [toast, setToast] = useState<{ message: string; type: 'ok' | 'err' | 'info' } | null>(null)

  // 加载Pipeline列表
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

  // 创建Pipeline
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

  // 启动Pipeline（异步不等待完成，只发请求后刷新列表）
  const handleStart = async (p: PipelineListItem) => {
    if (!confirm('确认启动 ' + p.course_code + ' Pipeline？\n全链路AI调用约需10-30分钟。')) return
    setOperating(p.id)
    setToast({ message: p.course_code + ' 正在启动...', type: 'info' })
    try {
      await startPipeline(p.id)
      setToast({ message: p.course_code + ' Pipeline执行完成', type: 'ok' })
      loadPipelines()
    } catch (e: any) {
      setToast({ message: '启动失败: ' + (e.message || ''), type: 'err' })
      loadPipelines()
    }
    setOperating(null)
  }

  // 取消Pipeline
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

  // 删除Pipeline
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

  // 统计数据
  const total = pipelines.length
  const running = pipelines.filter(p => p.status === 'running').length
  const reviewQueue = pipelines.filter(p => p.status === 'review_queue').length
  const failed = pipelines.filter(p => p.status === 'failed').length

  // 通用样式
  const card: React.CSSProperties = { background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)', borderRadius: 16, padding: 20, marginBottom: 16 }
  const stat: React.CSSProperties = { background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)', borderRadius: 14, padding: '16px 20px', flex: 1, minWidth: 120 }
  const btn: React.CSSProperties = { padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6, transition: 'all 0.15s ease' }
  const btnP: React.CSSProperties = { ...btn, background: '#007aff', color: '#fff', border: '1px solid #007aff' }

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
      </div>

      {/* 操作栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, flexWrap: 'wrap', gap: 8 }}>
        <div style={{ fontSize: 13, color: '#8e8e93' }}>{loading ? '加载中...' : total + ' 个Pipeline'}</div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button style={btn} onClick={loadPipelines}><RefreshCw size={14} /> 刷新</button>
          {canOperate && <button style={btnP} onClick={() => setShowCreate(true)}><Plus size={14} /> 创建Pipeline</button>}
        </div>
      </div>

      {/* Pipeline列表 */}
      <div style={card}>
        {loading ? (
          <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>加载中...</div>
        ) : pipelines.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Workflow size={40} style={{ color: '#c7c7cc', marginBottom: 12 }} />
            <div style={{ color: '#8e8e93', fontSize: 14 }}>暂无Pipeline</div>
            {canOperate && <div style={{ color: '#007aff', fontSize: 13, marginTop: 8, cursor: 'pointer' }} onClick={() => setShowCreate(true)}>创建第一个Pipeline →</div>}
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {pipelines.map(p => (
              <div key={p.id}
                style={{
                  display: 'flex', alignItems: 'center', gap: 16, padding: '14px 16px',
                  borderRadius: 12, border: '1px solid rgba(0,0,0,0.04)', background: 'rgba(0,0,0,0.01)',
                  cursor: 'pointer', transition: 'all 0.15s ease',
                }}
                onClick={() => navigate('/pipelines/' + p.id)}
                onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.04)'; (e.currentTarget as HTMLElement).style.borderColor = 'rgba(0,122,255,0.12)' }}
                onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,0,0,0.01)'; (e.currentTarget as HTMLElement).style.borderColor = 'rgba(0,0,0,0.04)' }}
              >
                {/* 左侧信息 */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 6 }}>
                    <span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e' }}>{p.course_code}</span>
                    <StatusBadge status={p.status} statusName={p.status_name} />
                    {p.auto_mode && <span style={{ fontSize: 10, color: '#8e8e93', background: 'rgba(0,0,0,0.04)', padding: '1px 6px', borderRadius: 4 }}>自动</span>}
                  </div>
                  <div style={{ fontSize: 13, color: '#8e8e93', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', marginBottom: 6 }}>
                    {p.course_name || p.course_code}
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 16, fontSize: 12, color: '#aeaeb2' }}>
                    <span>当前: {p.current_step_name}</span>
                    <span>创建: {formatTime(p.created_at)}</span>
                    {p.error_message && <span style={{ color: '#ff3b30' }}>错误: {p.error_message.substring(0, 40)}</span>}
                  </div>
                </div>

                {/* 中间进度条 */}
                <div style={{ width: 140, flexShrink: 0 }}>
                  <ProgressBar completed={p.steps_completed} total={p.steps_total} />
                </div>

                {/* 右侧操作按钮 */}
                <div style={{ display: 'flex', gap: 6, flexShrink: 0 }} onClick={e => e.stopPropagation()}>
                  {/* 启动按钮：仅pending状态且有操作权限 */}
                  {p.status === 'pending' && canOperate && (
                    <button title="启动" onClick={() => handleStart(p)}
                      disabled={operating === p.id}
                      style={{ width: 32, height: 32, borderRadius: 8, border: '1px solid rgba(52,199,89,0.3)', background: 'rgba(52,199,89,0.1)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#34c759' }}>
                      {operating === p.id ? <Loader size={14} style={{ animation: 'spin 1s linear infinite' }} /> : <Play size={14} />}
                    </button>
                  )}
                  {/* 取消按钮：pending或running状态，仅admin */}
                  {(p.status === 'pending' || p.status === 'running') && isAdmin && (
                    <button title="取消" onClick={() => handleCancel(p)}
                      style={{ width: 32, height: 32, borderRadius: 8, border: '1px solid rgba(255,149,0,0.3)', background: 'rgba(255,149,0,0.1)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#ff9500' }}>
                      <Square size={14} />
                    </button>
                  )}
                  {/* 删除按钮：非running状态，仅admin */}
                  {p.status !== 'running' && isAdmin && (
                    <button title="删除" onClick={() => handleDelete(p)}
                      style={{ width: 32, height: 32, borderRadius: 8, border: '1px solid rgba(255,59,48,0.2)', background: 'rgba(255,59,48,0.06)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#ff3b30' }}>
                      <Trash2 size={14} />
                    </button>
                  )}
                  {/* 查看详情箭头 */}
                  <div style={{ width: 32, height: 32, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#c7c7cc' }}>
                    <ChevronRight size={16} />
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 创建弹窗 */}
      {showCreate && <CreateDialog onClose={() => setShowCreate(false)} onCreate={handleCreate} />}

      {/* Toast提示 */}
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  )
}
