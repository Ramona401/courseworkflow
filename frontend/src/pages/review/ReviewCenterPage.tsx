/**
 * 审核中心页面（P6-1 + P6-2 + P7二级审批）
 *
 * P7更新：
 * 1. 新增「待确认定稿」队列（pending_finalize状态，仅senior_operator/admin可见）
 * 2. 待确认定稿列表支持：进入审核页查看、确认定稿、退回重审
 * 3. 待审核队列的「定稿归档」按钮改为「提交定稿」（operator）或「确认定稿」（senior_operator）
 * 4. 状态颜色：pending_finalize = 橙黄色
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getPipelines, markPassed, getOperators, batchAssignPipelines,
  confirmFinalize, rejectFinalize,
  type PipelineListItem, type OperatorInfo,
} from '@/api/pipelines'
import {
  ClipboardCheck, RefreshCw, Zap, AlertTriangle,
  CheckCircle, ArrowRight, Loader, ShieldCheck,
  FileText, XCircle, UserPlus, X, Users, RotateCcw, Clock,
} from 'lucide-react'

// ==================== Toast组件 ====================
function Toast({ message, type, onClose }: { message: string; type: 'ok' | 'err' | 'info'; onClose: () => void }) {
  useEffect(() => { const t = setTimeout(onClose, 5000); return () => clearTimeout(t) }, [onClose])
  const bg = type === 'ok' ? '#34c759' : type === 'err' ? '#ff3b30' : '#007aff'
  return (
    <div style={{
      position: 'fixed', bottom: 24, right: 24, background: bg, color: '#fff',
      padding: '12px 22px', borderRadius: 12, fontSize: 13, fontWeight: 500,
      zIndex: 9999, boxShadow: '0 4px 24px rgba(0,0,0,0.18)', maxWidth: 500,
    }}>
      {message}
    </div>
  )
}

// ==================== 统计卡片 ====================
function StatCard({ label, value, icon, color, sub }: {
  label: string; value: string | number; icon: React.ReactNode; color: string; sub?: string
}) {
  return (
    <div style={{
      background: '#fff', borderRadius: 16, padding: '20px 22px',
      border: '1px solid rgba(0,0,0,0.04)', boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
      flex: 1, minWidth: 140, transition: 'all 0.2s ease', cursor: 'default',
    }}
      onMouseEnter={e => { (e.currentTarget as HTMLElement).style.transform = 'translateY(-2px)'; (e.currentTarget as HTMLElement).style.boxShadow = '0 8px 25px rgba(0,0,0,0.08)' }}
      onMouseLeave={e => { (e.currentTarget as HTMLElement).style.transform = 'translateY(0)'; (e.currentTarget as HTMLElement).style.boxShadow = '0 1px 3px rgba(0,0,0,0.04)' }}>
      <div style={{ width: 36, height: 36, background: color, borderRadius: 10, display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: 12 }}>
        {icon}
      </div>
      <div style={{ fontSize: 11, color: '#86868b', fontWeight: 600, marginBottom: 2 }}>{label}</div>
      <div style={{ fontSize: 28, fontWeight: 700, color: '#1d1d1f', letterSpacing: '-0.5px' }}>{value}</div>
      {sub && <div style={{ fontSize: 11, color: '#86868b', marginTop: 2 }}>{sub}</div>}
    </div>
  )
}

// ==================== 状态徽章 ====================
function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { bg: string; fg: string; label: string }> = {
    review_queue:      { bg: 'rgba(255,149,0,0.12)', fg: '#ff9500', label: '待审核' },
    needs_human:       { bg: 'rgba(255,204,0,0.15)', fg: '#cc9900', label: '需人工' },
    pending_finalize:  { bg: 'rgba(255,149,0,0.15)', fg: '#cc6600', label: '待确认定稿' }, // P7新增
    finalized:         { bg: 'rgba(52,199,89,0.12)', fg: '#34c759', label: '已定稿' },
    verified:          { bg: 'rgba(52,199,89,0.15)', fg: '#248a3d', label: '验收通过' },
    verify_failed:     { bg: 'rgba(255,59,48,0.15)', fg: '#d70015', label: '验收未通过' },
  }
  const c = map[status] || { bg: 'rgba(142,142,147,0.1)', fg: '#8e8e93', label: status }
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 4,
      padding: '3px 10px', borderRadius: 20, fontSize: 11, fontWeight: 600,
      background: c.bg, color: c.fg, whiteSpace: 'nowrap',
    }}>{c.label}</span>
  )
}

// ==================== 审核轮次徽章 ====================
function RoundBadge({ round }: { round: number }) {
  if (round <= 1) return null
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 3,
      padding: '2px 8px', borderRadius: 10, fontSize: 10, fontWeight: 700,
      background: 'rgba(255,59,48,0.1)', color: '#ff3b30',
    }}>{round}审</span>
  )
}

// ==================== 分数显示 ====================
function ScoreDisplay({ value, label }: { value: number | null; label: string }) {
  if (value === null || value === undefined) {
    return <span style={{ fontSize: 11, color: '#c7c7cc' }}>{label}: -</span>
  }
  let color = '#ff3b30'
  if (value >= 9.0) color = '#34c759'
  else if (value >= 7.0) color = '#ff9500'
  return (
    <span style={{ fontSize: 11, color: '#8e8e93' }}>
      {label}: <span style={{ color, fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>{value.toFixed(1)}</span>
    </span>
  )
}

function canMarkPassed(p: PipelineListItem): boolean {
  return ['review_queue', 'needs_human', 'failed'].includes(p.status) && p.meta_score !== null && p.meta_score >= 9.0
}

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

function isToday(t: string | null): boolean {
  if (!t) return false
  const d = new Date(t)
  const now = new Date()
  return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && d.getDate() === now.getDate()
}

// ==================== 主页面组件 ====================

export default function ReviewCenterPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const canOperate = user?.role === 'admin' || user?.role === 'operator' || user?.role === 'senior_operator'
  const isAdmin = user?.role === 'admin' || user?.role === 'senior_operator'
  // P7：超级审核员（senior_operator/admin）可确认/退回定稿
  const isSuperReviewer = user?.role === 'admin' || user?.role === 'senior_operator'

  const [pipelines, setPipelines] = useState<PipelineListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [toast, setToast] = useState<{ message: string; type: 'ok' | 'err' | 'info' } | null>(null)
  const [showCompleted, setShowCompleted] = useState(true)
  // P7：退回重审弹窗
  const [showRejectDialog, setShowRejectDialog] = useState(false)
  const [rejectTargetId, setRejectTargetId] = useState('')
  const [rejectTargetCode, setRejectTargetCode] = useState('')
  const [rejectReason, setRejectReason] = useState('')
  const [actionLoading, setActionLoading] = useState(false)
  // P6-2：批量分配
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [showAssignDialog, setShowAssignDialog] = useState(false)
  const [operators, setOperators] = useState<OperatorInfo[]>([])
  const [selectedOperator, setSelectedOperator] = useState('')
  const [assigning, setAssigning] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getPipelines()
      setPipelines(data.pipelines || [])
    } catch (e: any) {
      setToast({ message: '加载失败: ' + (e.message || ''), type: 'err' })
    }
    setLoading(false)
  }, [])

  useEffect(() => { loadData() }, [loadData])

  const loadOperators = async () => {
    try {
      const ops = await getOperators()
      setOperators(ops || [])
    } catch { /* 静默忽略 */ }
  }

  const openAssignDialog = async () => {
    if (selectedIds.size === 0) { setToast({ message: '请先选择要分配的Pipeline', type: 'info' }); return }
    await loadOperators()
    setSelectedOperator('')
    setShowAssignDialog(true)
  }

  const handleBatchAssign = async () => {
    if (!selectedOperator) { setToast({ message: '请选择审核员', type: 'info' }); return }
    setAssigning(true)
    try {
      const ids = Array.from(selectedIds)
      const result = await batchAssignPipelines(ids, selectedOperator)
      setToast({ message: '分配成功: ' + result.success_count + ' 个Pipeline已分配给 ' + result.assigned_name, type: 'ok' })
      setShowAssignDialog(false)
      setSelectedIds(new Set())
      loadData()
    } catch (e: any) {
      setToast({ message: '分配失败: ' + (e.message || ''), type: 'err' })
    }
    setAssigning(false)
  }

  const toggleSelect = (id: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  // ===== P7：确认定稿 =====
  const handleConfirmFinalize = async (p: PipelineListItem) => {
    if (!confirm('确认定稿归档 ' + p.course_code + '？Pipeline将进入finalized状态。')) return
    setActionLoading(true)
    try {
      await confirmFinalize(p.id)
      setToast({ message: p.course_code + ' 定稿已确认！', type: 'ok' })
      loadData()
    } catch (e: any) {
      setToast({ message: '确认失败: ' + (e.message || ''), type: 'err' })
    }
    setActionLoading(false)
  }

  // ===== P7：退回重审 =====
  const openRejectDialog = (p: PipelineListItem) => {
    setRejectTargetId(p.id)
    setRejectTargetCode(p.course_code)
    setRejectReason('')
    setShowRejectDialog(true)
  }

  const handleRejectFinalize = async () => {
    setActionLoading(true)
    try {
      await rejectFinalize(rejectTargetId, rejectReason)
      setToast({ message: rejectTargetCode + ' 已退回重审！', type: 'ok' })
      setShowRejectDialog(false)
      loadData()
    } catch (e: any) {
      setToast({ message: '退回失败: ' + (e.message || ''), type: 'err' })
    }
    setActionLoading(false)
  }

  // ===== 数据分组 =====

  // 待审核队列：review_queue + needs_human
  const pendingReview = pipelines
    .filter(p => p.status === 'review_queue' || p.status === 'needs_human')
    .sort((a, b) => {
      if (a.status === 'needs_human' && b.status !== 'needs_human') return -1
      if (a.status !== 'needs_human' && b.status === 'needs_human') return 1
      if ((a.review_round || 1) > (b.review_round || 1)) return -1
      if ((a.review_round || 1) < (b.review_round || 1)) return 1
      const ta = a.created_at ? new Date(a.created_at).getTime() : 0
      const tb = b.created_at ? new Date(b.created_at).getTime() : 0
      return ta - tb
    })

  // P7新增：待确认定稿队列（pending_finalize），仅超级审核员可见
  const pendingFinalize = pipelines
    .filter(p => p.status === 'pending_finalize')
    .sort((a, b) => {
      const ta = a.created_at ? new Date(a.created_at).getTime() : 0
      const tb = b.created_at ? new Date(b.created_at).getTime() : 0
      return ta - tb
    })

  const toggleSelectAllPending = () => {
    if (selectedIds.size === pendingReview.length && pendingReview.length > 0) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(pendingReview.map(p => p.id)))
    }
  }
  const isAllPendingSelected = pendingReview.length > 0 && selectedIds.size === pendingReview.length

  // 已完成：finalized + verified + verify_failed（最近20条）
  const completedReview = pipelines
    .filter(p => p.status === 'finalized' || p.status === 'verified' || p.status === 'verify_failed')
    .sort((a, b) => {
      const ta = a.completed_at ? new Date(a.completed_at).getTime() : (a.created_at ? new Date(a.created_at).getTime() : 0)
      const tb = b.completed_at ? new Date(b.completed_at).getTime() : (b.created_at ? new Date(b.created_at).getTime() : 0)
      return tb - ta
    })
    .slice(0, 20)

  // ===== 统计 =====
  const totalPending = pendingReview.length
  const needsHumanCount = pendingReview.filter(p => p.status === 'needs_human').length
  const pendingFinalizeCount = pendingFinalize.length
  const todayFinalized = pipelines.filter(p =>
    (p.status === 'finalized' || p.status === 'verified') && isToday(p.completed_at)
  ).length
  const totalFinalized = pipelines.filter(p => p.status === 'finalized' || p.status === 'verified').length
  const verifiedCount = pipelines.filter(p => p.status === 'verified').length

  const handleMarkPassed = async (p: PipelineListItem) => {
    if (!confirm('确认快捷通过 ' + p.course_code + '？')) return
    try {
      await markPassed(p.id)
      setToast({ message: p.course_code + ' 已快捷通过并归档', type: 'ok' })
      loadData()
    } catch (e: any) {
      setToast({ message: '快捷通过失败: ' + (e.message || ''), type: 'err' })
    }
  }

  const btn: React.CSSProperties = {
    padding: '7px 14px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 5, transition: 'all 0.15s ease',
  }

  return (
    <div>
      {/* ===== 顶部操作栏 ===== */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
        <p style={{ fontSize: 14, color: '#86868b', margin: 0 }}>集中管理所有待审核的Pipeline课程</p>
        <div style={{ display: 'flex', gap: 8 }}>
          {canOperate && isAdmin && selectedIds.size > 0 && (
            <button style={{ ...btn, background: '#5856d6', color: '#fff', border: '1px solid #5856d6' }} onClick={openAssignDialog}>
              <UserPlus size={14} /> 分配审核员 ({selectedIds.size})
            </button>
          )}
          <button style={btn} onClick={loadData}><RefreshCw size={14} /> 刷新</button>
        </div>
      </div>

      {/* ===== 统计卡片 ===== */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 24, flexWrap: 'wrap' }}>
        <StatCard label="待审核" value={totalPending}
          icon={<ClipboardCheck size={18} color="#fff" />}
          color="linear-gradient(135deg, #ff9500, #ffcc00)"
          sub={needsHumanCount > 0 ? needsHumanCount + ' 个需人工干预' : undefined} />
        {/* P7：待确认定稿统计卡片（仅超级审核员可见） */}
        {isSuperReviewer && (
          <StatCard label="待确认定稿" value={pendingFinalizeCount}
            icon={<Clock size={18} color="#fff" />}
            color="linear-gradient(135deg, #cc6600, #ff9500)"
            sub={pendingFinalizeCount > 0 ? '等待超级审核员确认' : undefined} />
        )}
        <StatCard label="今日已审核" value={todayFinalized}
          icon={<CheckCircle size={18} color="#fff" />}
          color="linear-gradient(135deg, #34c759, #30d158)" />
        <StatCard label="总已定稿" value={totalFinalized}
          icon={<FileText size={18} color="#fff" />}
          color="linear-gradient(135deg, #007aff, #5ac8fa)" />
        <StatCard label="验收通过" value={verifiedCount}
          icon={<ShieldCheck size={18} color="#fff" />}
          color="linear-gradient(135deg, #32ade6, #5856d6)" />
      </div>

      {loading && (
        <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>
          <Loader size={24} style={{ animation: 'spin 1s linear infinite', marginBottom: 8 }} />
          <div style={{ fontSize: 14 }}>加载中...</div>
        </div>
      )}

      {/* ===== P7新增：待确认定稿队列（仅超级审核员可见） ===== */}
      {!loading && isSuperReviewer && pendingFinalize.length > 0 && (
        <div style={{
          background: '#fff', borderRadius: 16, border: '1px solid rgba(204,102,0,0.15)',
          boxShadow: '0 1px 3px rgba(0,0,0,0.04)', marginBottom: 24, overflow: 'hidden',
        }}>
          {/* 区块标题 */}
          <div style={{
            display: 'flex', alignItems: 'center', gap: 10,
            padding: '16px 20px', borderBottom: '1px solid rgba(204,102,0,0.1)',
            background: 'rgba(255,149,0,0.04)',
          }}>
            <Clock size={18} color="#cc6600" />
            <span style={{ fontSize: 15, fontWeight: 700, color: '#1c1c1e' }}>待确认定稿</span>
            <span style={{
              fontSize: 12, fontWeight: 600, color: '#cc6600',
              background: 'rgba(204,102,0,0.1)', padding: '2px 10px', borderRadius: 10,
            }}>{pendingFinalizeCount}</span>
            <span style={{ fontSize: 12, color: '#86868b', marginLeft: 4 }}>等待超级审核员确认或退回</span>
          </div>

          {/* 待确认定稿列表 */}
          {pendingFinalize.map((p, idx) => (
            <div
              key={p.id}
              style={{
                display: 'flex', alignItems: 'center', gap: 14,
                padding: '14px 20px',
                borderBottom: idx < pendingFinalize.length - 1 ? '1px solid rgba(0,0,0,0.03)' : 'none',
                transition: 'background 0.15s ease',
              }}
              onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(255,149,0,0.03)' }}
              onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
            >
              {/* 优先级指示器 */}
              <div style={{ width: 4, height: 40, borderRadius: 2, flexShrink: 0, background: '#cc6600' }} />

              {/* 课程信息 */}
              <div style={{ flex: 1, minWidth: 0, cursor: 'pointer' }} onClick={() => navigate('/pipelines/' + p.id + '/review')}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                  <span style={{ fontSize: 14, fontWeight: 700, color: '#1c1c1e' }}>{p.course_code}</span>
                  <StatusBadge status={p.status} />
                  <RoundBadge round={p.review_round || 1} />
                </div>
                <div style={{ fontSize: 12, color: '#86868b', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {p.course_name || p.course_code}
                  {p.assigned_name && (
                    <span style={{ marginLeft: 8, fontSize: 11, color: '#5856d6', fontWeight: 500 }}>
                      <Users size={10} style={{ verticalAlign: -1, marginRight: 2 }} />{p.assigned_name}
                    </span>
                  )}
                </div>
              </div>

              {/* 分数 */}
              <div style={{ display: 'flex', gap: 14, flexShrink: 0 }}>
                <ScoreDisplay value={p.eval_avg_score} label="评估" />
                <ScoreDisplay value={p.meta_score} label="Meta" />
                <ScoreDisplay value={p.translator_score} label="翻译" />
              </div>

              {/* 时间 */}
              <div style={{ fontSize: 11, color: '#aeaeb2', flexShrink: 0, minWidth: 60, textAlign: 'right' }}>
                {formatTime(p.created_at)}
              </div>

              {/* 操作按钮 */}
              <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
                {/* 查看审核内容 */}
                <button
                  title="进入审核页查看课件内容"
                  onClick={() => navigate('/pipelines/' + p.id + '/review')}
                  style={{
                    width: 30, height: 30, borderRadius: 8,
                    border: '1px solid rgba(0,122,255,0.3)', background: 'rgba(0,122,255,0.08)',
                    cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
                    color: '#007aff', padding: 0,
                  }}
                >
                  <ArrowRight size={14} />
                </button>
                {/* 退回重审 */}
                <button
                  title="退回给审核员重审"
                  onClick={() => openRejectDialog(p)}
                  disabled={actionLoading}
                  style={{
                    width: 30, height: 30, borderRadius: 8,
                    border: '1px solid rgba(255,59,48,0.3)', background: 'rgba(255,59,48,0.08)',
                    cursor: actionLoading ? 'not-allowed' : 'pointer',
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    color: '#ff3b30', padding: 0, opacity: actionLoading ? 0.5 : 1,
                  }}
                >
                  <RotateCcw size={14} />
                </button>
                {/* 确认定稿 */}
                <button
                  title="确认定稿，进入finalized状态"
                  onClick={() => handleConfirmFinalize(p)}
                  disabled={actionLoading}
                  style={{
                    padding: '0 14px', height: 30, borderRadius: 8,
                    border: 'none', background: '#34c759',
                    cursor: actionLoading ? 'not-allowed' : 'pointer',
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    color: '#fff', fontSize: 12, fontWeight: 600, gap: 4,
                    opacity: actionLoading ? 0.5 : 1,
                  }}
                >
                  <ShieldCheck size={13} /> 确认定稿
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* ===== 待审核队列 ===== */}
      {!loading && (
        <div style={{
          background: '#fff', borderRadius: 16, border: '1px solid rgba(0,0,0,0.04)',
          boxShadow: '0 1px 3px rgba(0,0,0,0.04)', marginBottom: 24, overflow: 'hidden',
        }}>
          <div style={{
            display: 'flex', alignItems: 'center', gap: 10,
            padding: '16px 20px', borderBottom: '1px solid rgba(0,0,0,0.04)',
            background: 'rgba(255,149,0,0.03)',
          }}>
            {isAdmin && pendingReview.length > 0 && (
              <input type="checkbox" checked={isAllPendingSelected} onChange={toggleSelectAllPending}
                style={{ width: 16, height: 16, cursor: 'pointer', accentColor: '#5856d6' }} />
            )}
            <ClipboardCheck size={18} color="#ff9500" />
            <span style={{ fontSize: 15, fontWeight: 700, color: '#1c1c1e' }}>待审核队列</span>
            <span style={{
              fontSize: 12, fontWeight: 600, color: '#ff9500',
              background: 'rgba(255,149,0,0.12)', padding: '2px 10px', borderRadius: 10,
            }}>{totalPending}</span>
            {needsHumanCount > 0 && (
              <span style={{
                fontSize: 11, fontWeight: 600, color: '#cc9900',
                background: 'rgba(255,204,0,0.15)', padding: '2px 10px', borderRadius: 10,
                display: 'inline-flex', alignItems: 'center', gap: 4,
              }}>
                <AlertTriangle size={11} /> {needsHumanCount} 需人工
              </span>
            )}
          </div>

          {pendingReview.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '40px 20px', color: '#aeaeb2' }}>
              <CheckCircle size={36} style={{ marginBottom: 8, opacity: 0.4 }} />
              <div style={{ fontSize: 14, fontWeight: 500 }}>所有课程审核已完成</div>
              <div style={{ fontSize: 12, marginTop: 4 }}>当前没有待审核的Pipeline</div>
            </div>
          ) : (
            <div>
              {pendingReview.map((p, idx) => (
                <div
                  key={p.id}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 14,
                    padding: '14px 20px',
                    borderBottom: idx < pendingReview.length - 1 ? '1px solid rgba(0,0,0,0.03)' : 'none',
                    transition: 'background 0.15s ease', cursor: 'pointer',
                  }}
                  onClick={() => navigate('/pipelines/' + p.id + '/review')}
                  onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.03)' }}
                  onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
                >
                  {isAdmin && (
                    <div onClick={e => { e.stopPropagation(); toggleSelect(p.id) }} style={{ flexShrink: 0 }}>
                      <input type="checkbox" checked={selectedIds.has(p.id)} onChange={() => toggleSelect(p.id)}
                        style={{ width: 16, height: 16, cursor: 'pointer', accentColor: '#5856d6' }} />
                    </div>
                  )}

                  <div style={{
                    width: 4, height: 40, borderRadius: 2, flexShrink: 0,
                    background: p.status === 'needs_human' ? '#cc9900' : (p.review_round || 1) > 1 ? '#ff3b30' : '#ff9500',
                  }} />

                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                      <span style={{ fontSize: 14, fontWeight: 700, color: '#1c1c1e' }}>{p.course_code}</span>
                      <StatusBadge status={p.status} />
                      <RoundBadge round={p.review_round || 1} />
                    </div>
                    <div style={{ fontSize: 12, color: '#86868b', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {p.course_name || p.course_code}
                      {p.assigned_name && (
                        <span style={{ marginLeft: 8, fontSize: 11, color: '#5856d6', fontWeight: 500 }}>
                          <Users size={10} style={{ verticalAlign: -1, marginRight: 2 }} />{p.assigned_name}
                        </span>
                      )}
                    </div>
                  </div>

                  <div style={{ display: 'flex', gap: 14, flexShrink: 0 }}>
                    <ScoreDisplay value={p.eval_avg_score} label="评估" />
                    <ScoreDisplay value={p.meta_score} label="Meta" />
                    <ScoreDisplay value={p.translator_score} label="翻译" />
                  </div>

                  <div style={{ fontSize: 11, color: '#aeaeb2', flexShrink: 0, minWidth: 60, textAlign: 'right' }}>
                    {formatTime(p.created_at)}
                  </div>

                  <div style={{ display: 'flex', gap: 6, flexShrink: 0 }} onClick={e => e.stopPropagation()}>
                    {canMarkPassed(p) && canOperate && (
                      <button title="快捷通过（Meta>=9.0）" onClick={() => handleMarkPassed(p)}
                        style={{
                          width: 30, height: 30, borderRadius: 8,
                          border: '1px solid rgba(52,199,89,0.3)', background: 'rgba(52,199,89,0.08)',
                          cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
                          color: '#34c759', padding: 0,
                        }}>
                        <Zap size={14} />
                      </button>
                    )}
                    <button title="进入审核" onClick={() => navigate('/pipelines/' + p.id + '/review')}
                      style={{
                        width: 30, height: 30, borderRadius: 8,
                        border: '1px solid rgba(0,122,255,0.3)', background: 'rgba(0,122,255,0.08)',
                        cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
                        color: '#007aff', padding: 0,
                      }}>
                      <ArrowRight size={14} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ===== 已完成审核记录 ===== */}
      {!loading && completedReview.length > 0 && (
        <div style={{
          background: '#fff', borderRadius: 16, border: '1px solid rgba(0,0,0,0.04)',
          boxShadow: '0 1px 3px rgba(0,0,0,0.04)', overflow: 'hidden',
        }}>
          <div
            style={{
              display: 'flex', alignItems: 'center', gap: 10,
              padding: '14px 20px', borderBottom: showCompleted ? '1px solid rgba(0,0,0,0.04)' : 'none',
              cursor: 'pointer', transition: 'background 0.15s ease',
            }}
            onClick={() => setShowCompleted(!showCompleted)}
            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,0,0,0.015)' }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
          >
            <CheckCircle size={18} color="#34c759" />
            <span style={{ fontSize: 15, fontWeight: 700, color: '#1c1c1e', flex: 1 }}>已完成审核</span>
            <span style={{ fontSize: 12, color: '#8e8e93', background: 'rgba(0,0,0,0.04)', padding: '2px 10px', borderRadius: 10, fontWeight: 500 }}>
              最近 {completedReview.length} 条
            </span>
            <span style={{ fontSize: 12, color: '#c7c7cc', transition: 'transform 0.2s', transform: showCompleted ? 'rotate(90deg)' : 'none' }}>&#9654;</span>
          </div>

          {showCompleted && (
            <div>
              {completedReview.map((p, idx) => (
                <div
                  key={p.id}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 14,
                    padding: '12px 20px',
                    borderBottom: idx < completedReview.length - 1 ? '1px solid rgba(0,0,0,0.03)' : 'none',
                    transition: 'background 0.15s ease, opacity 0.15s ease', cursor: 'pointer', opacity: 0.85,
                  }}
                  onClick={() => navigate('/pipelines/' + p.id)}
                  onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.03)'; (e.currentTarget as HTMLElement).style.opacity = '1' }}
                  onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent'; (e.currentTarget as HTMLElement).style.opacity = '0.85' }}
                >
                  <div style={{ flexShrink: 0 }}>
                    {p.status === 'verified' ? <ShieldCheck size={18} color="#248a3d" /> :
                      p.status === 'verify_failed' ? <XCircle size={18} color="#d70015" /> :
                        <CheckCircle size={18} color="#34c759" />}
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <span style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e' }}>{p.course_code}</span>
                      <StatusBadge status={p.status} />
                      <RoundBadge round={p.review_round || 1} />
                    </div>
                    <div style={{ fontSize: 11, color: '#aeaeb2', marginTop: 2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {p.course_name || p.course_code}
                    </div>
                  </div>
                  <div style={{ display: 'flex', gap: 12, flexShrink: 0 }}>
                    <ScoreDisplay value={p.meta_score} label="Meta" />
                    <ScoreDisplay value={p.translator_score} label="翻译" />
                  </div>
                  <div style={{ fontSize: 11, color: '#c7c7cc', flexShrink: 0, minWidth: 60, textAlign: 'right' }}>
                    {formatTime(p.completed_at || p.created_at)}
                  </div>
                  <div style={{ color: '#c7c7cc', fontSize: 14, flexShrink: 0 }}>&#8250;</div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`}</style>

      {/* ===== P6-2：批量分配弹窗 ===== */}
      {showAssignDialog && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
          zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center',
        }} onClick={e => { if (e.target === e.currentTarget && !assigning) setShowAssignDialog(false) }}>
          <div style={{ background: '#fff', borderRadius: 20, width: 440, maxWidth: '94vw', padding: 28, boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 20 }}>
              <UserPlus size={20} color="#5856d6" />
              <div style={{ fontSize: 18, fontWeight: 700, color: '#1c1c1e', flex: 1 }}>分配审核员</div>
              <button onClick={() => !assigning && setShowAssignDialog(false)} style={{
                background: '#f2f2f7', border: 'none', borderRadius: '50%', width: 30, height: 30,
                display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer',
              }}><X size={16} color="#8e8e93" /></button>
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
                      display: 'flex', alignItems: 'center', gap: 12,
                      padding: '10px 14px', borderRadius: 10, cursor: 'pointer',
                      border: selectedOperator === op.id ? '2px solid #5856d6' : '1px solid rgba(0,0,0,0.08)',
                      background: selectedOperator === op.id ? 'rgba(88,86,214,0.05)' : '#fff',
                      transition: 'all 0.15s ease',
                    }}>
                      <div style={{
                        width: 32, height: 32, borderRadius: '50%',
                        background: selectedOperator === op.id ? '#5856d6' : 'rgba(88,86,214,0.1)',
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        color: selectedOperator === op.id ? '#fff' : '#5856d6',
                        fontSize: 13, fontWeight: 600, flexShrink: 0,
                      }}>{op.display_name.charAt(0)}</div>
                      <div style={{ flex: 1 }}>
                        <div style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>{op.display_name}</div>
                        <div style={{ fontSize: 11, color: '#86868b' }}>{op.username} · {op.role === 'admin' ? '管理员' : op.role === 'senior_operator' ? '高级操作员' : '操作员'}</div>
                      </div>
                      {selectedOperator === op.id && <CheckCircle size={18} color="#5856d6" />}
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

      {/* ===== P7：退回重审弹窗 ===== */}
      {showRejectDialog && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
          zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 40,
        }} onClick={e => { if (e.target === e.currentTarget && !actionLoading) setShowRejectDialog(false) }}>
          <div style={{ background: '#fff', borderRadius: 16, padding: 0, maxWidth: 480, width: '100%', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '16px 20px', borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
              <RotateCcw size={16} color="#ff3b30" />
              <span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e', flex: 1 }}>
                退回重审 — {rejectTargetCode}
              </span>
              <button onClick={() => !actionLoading && setShowRejectDialog(false)} style={{
                background: '#f2f2f7', border: 'none', borderRadius: '50%', width: 28, height: 28,
                display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer',
              }}><X size={14} color="#8e8e93" /></button>
            </div>
            <div style={{ padding: '16px 20px' }}>
              <div style={{ fontSize: 13, color: '#3c3c43', marginBottom: 10 }}>
                将退回给原审核员重新审核，可填写退回原因（选填）：
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
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, padding: '12px 20px', borderTop: '1px solid rgba(0,0,0,0.06)' }}>
              <button onClick={() => !actionLoading && setShowRejectDialog(false)} style={{
                padding: '8px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
                background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer', color: '#3c3c43',
              }}>取消</button>
              <button onClick={handleRejectFinalize} disabled={actionLoading} style={{
                padding: '8px 24px', borderRadius: 10, border: 'none',
                background: actionLoading ? '#e5e5ea' : '#ff3b30',
                color: actionLoading ? '#aeaeb2' : '#fff',
                fontSize: 13, fontWeight: 600, cursor: actionLoading ? 'not-allowed' : 'pointer',
              }}>{actionLoading ? '处理中...' : '确认退回'}</button>
            </div>
          </div>
        </div>
      )}

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  )
}
