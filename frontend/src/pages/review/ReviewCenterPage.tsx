/**
 * 审核中心页面 v8.0
 *
 * v8.0 变更：
 * 1. 按角色区分可见内容：
 *    - operator（骨干教师）：看到自己负责的 review_queue，可提交定稿
 *    - senior_operator（学校管理员）：看到 pending_finalize（骨干提交上来的），可确认/退回
 *    - admin：看到全部（review_queue + pending_finalize）
 * 2. 待审核队列按 review_round 分为「一审」和「二审」两个区块分开展示
 * 3. 统计卡片按角色差异化显示
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

// ==================== Toast ====================
function Toast({ message, type, onClose }: { message: string; type: 'ok' | 'err' | 'info'; onClose: () => void }) {
  useEffect(() => { const t = setTimeout(onClose, 5000); return () => clearTimeout(t) }, [onClose])
  const bg = type === 'ok' ? '#34c759' : type === 'err' ? '#ff3b30' : '#007aff'
  return (
    <div style={{
      position: 'fixed', bottom: 24, right: 24, background: bg, color: '#fff',
      padding: '12px 22px', borderRadius: 12, fontSize: 13, fontWeight: 500,
      zIndex: 9999, boxShadow: '0 4px 24px rgba(0,0,0,0.18)', maxWidth: 500,
    }}>{message}</div>
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
      flex: 1, minWidth: 140,
    }}>
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
    review_queue:     { bg: 'rgba(255,149,0,0.12)',  fg: '#ff9500', label: '待审核' },
    needs_human:      { bg: 'rgba(255,204,0,0.15)',  fg: '#cc9900', label: '需人工' },
    pending_finalize: { bg: 'rgba(255,149,0,0.15)',  fg: '#cc6600', label: '待确认定稿' },
    finalized:        { bg: 'rgba(52,199,89,0.12)',  fg: '#34c759', label: '已定稿' },
    verified:         { bg: 'rgba(52,199,89,0.15)',  fg: '#248a3d', label: '验收通过' },
    verify_failed:    { bg: 'rgba(255,59,48,0.15)',  fg: '#d70015', label: '验收未通过' },
  }
  const c = map[status] || { bg: 'rgba(142,142,147,0.1)', fg: '#8e8e93', label: status }
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, padding: '3px 10px', borderRadius: 20, fontSize: 11, fontWeight: 600, background: c.bg, color: c.fg, whiteSpace: 'nowrap' }}>
      {c.label}
    </span>
  )
}

// ==================== 审核轮次徽章 ====================
function RoundBadge({ round }: { round: number }) {
  if (round <= 1) return null
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 3, padding: '2px 8px', borderRadius: 10, fontSize: 10, fontWeight: 700, background: 'rgba(255,59,48,0.1)', color: '#ff3b30' }}>
      {round}审
    </span>
  )
}

// ==================== 分数显示 ====================
function ScoreDisplay({ value, label }: { value: number | null; label: string }) {
  if (value === null || value === undefined) return <span style={{ fontSize: 11, color: '#c7c7cc' }}>{label}: -</span>
  const color = value >= 9.0 ? '#34c759' : value >= 7.0 ? '#ff9500' : '#ff3b30'
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
  const diff = Date.now() - d.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前'
  return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function isToday(t: string | null): boolean {
  if (!t) return false
  const d = new Date(t), now = new Date()
  return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && d.getDate() === now.getDate()
}

// ==================== 单条Pipeline行 ====================
function PipelineRow({
  p, idx, total, showCheckbox, isSelected, onToggle, onNavigate,
  showActionBtns, onMarkPassed, onReview,
  accentColor,
}: {
  p: PipelineListItem; idx: number; total: number
  showCheckbox: boolean; isSelected: boolean; onToggle: () => void
  onNavigate: () => void; showActionBtns: boolean
  onMarkPassed: () => void; onReview: () => void
  accentColor: string
}) {
  return (
    <div
      style={{
        display: 'flex', alignItems: 'center', gap: 14,
        padding: '14px 20px',
        borderBottom: idx < total - 1 ? '1px solid rgba(0,0,0,0.03)' : 'none',
        transition: 'background 0.15s ease', cursor: 'pointer',
      }}
      onClick={onNavigate}
      onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.03)' }}
      onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
    >
      {/* 复选框 */}
      {showCheckbox && (
        <div onClick={e => { e.stopPropagation(); onToggle() }} style={{ flexShrink: 0 }}>
          <input type="checkbox" checked={isSelected} onChange={onToggle}
            style={{ width: 16, height: 16, cursor: 'pointer', accentColor: '#5856d6' }} />
        </div>
      )}

      {/* 左侧色条 */}
      <div style={{
        width: 4, height: 40, borderRadius: 2, flexShrink: 0,
        background: p.status === 'needs_human' ? '#cc9900' : accentColor,
      }} />

      {/* 课程信息 */}
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
      {showActionBtns && (
        <div style={{ display: 'flex', gap: 6, flexShrink: 0 }} onClick={e => e.stopPropagation()}>
          {canMarkPassed(p) && (
            <button title="快捷通过（Meta≥9.0）" onClick={onMarkPassed}
              style={{ width: 30, height: 30, borderRadius: 8, border: '1px solid rgba(52,199,89,0.3)', background: 'rgba(52,199,89,0.08)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#34c759', padding: 0 }}>
              <Zap size={14} />
            </button>
          )}
          <button title="进入审核" onClick={onReview}
            style={{ width: 30, height: 30, borderRadius: 8, border: '1px solid rgba(0,122,255,0.3)', background: 'rgba(0,122,255,0.08)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#007aff', padding: 0 }}>
            <ArrowRight size={14} />
          </button>
        </div>
      )}
    </div>
  )
}

// ==================== 队列区块容器 ====================
function QueueBlock({
  title, icon, titleColor, bgColor, borderColor, count, badge, hint,
  showCheckbox, isAllSelected, onToggleAll,
  children, empty,
}: {
  title: string; icon: React.ReactNode; titleColor: string; bgColor: string; borderColor: string
  count: number; badge?: React.ReactNode; hint?: string
  showCheckbox?: boolean; isAllSelected?: boolean; onToggleAll?: () => void
  children: React.ReactNode; empty?: string
}) {
  return (
    <div style={{ background: '#fff', borderRadius: 16, border: `1px solid ${borderColor}`, boxShadow: '0 1px 3px rgba(0,0,0,0.04)', marginBottom: 20, overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '16px 20px', borderBottom: `1px solid ${borderColor}`, background: bgColor }}>
        {showCheckbox && (
          <input type="checkbox" checked={isAllSelected} onChange={onToggleAll}
            style={{ width: 16, height: 16, cursor: 'pointer', accentColor: '#5856d6' }} />
        )}
        {icon}
        <span style={{ fontSize: 15, fontWeight: 700, color: '#1c1c1e' }}>{title}</span>
        <span style={{ fontSize: 12, fontWeight: 600, color: titleColor, background: `${titleColor}18`, padding: '2px 10px', borderRadius: 10 }}>{count}</span>
        {badge}
        {hint && <span style={{ fontSize: 12, color: '#86868b', marginLeft: 4 }}>{hint}</span>}
      </div>
      {count === 0 ? (
        <div style={{ textAlign: 'center', padding: '32px 20px', color: '#aeaeb2' }}>
          <CheckCircle size={32} style={{ marginBottom: 8, opacity: 0.3 }} />
          <div style={{ fontSize: 13 }}>{empty || '暂无数据'}</div>
        </div>
      ) : children}
    </div>
  )
}

// ==================== 主页面组件 ====================
export default function ReviewCenterPage() {
  const navigate = useNavigate()
  const { user } = useAuth()

  // 角色判断
  const isAdmin          = user?.role === 'admin'
  const isSeniorOperator = user?.role === 'senior_operator'
  const isOperator       = user?.role === 'operator'
  // 学校管理员和系统管理员都是超级审核员（可确认/退回定稿）
  const isSuperReviewer  = isAdmin || isSeniorOperator
  // 骨干教师和以上都可操作审核
  const canOperate       = isAdmin || isSeniorOperator || isOperator

  const [pipelines, setPipelines]       = useState<PipelineListItem[]>([])
  const [loading, setLoading]           = useState(true)
  const [toast, setToast]               = useState<{ message: string; type: 'ok' | 'err' | 'info' } | null>(null)
  const [showCompleted, setShowCompleted] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // 退回重审弹窗
  const [showRejectDialog, setShowRejectDialog] = useState(false)
  const [rejectTargetId, setRejectTargetId]     = useState('')
  const [rejectTargetCode, setRejectTargetCode] = useState('')
  const [rejectReason, setRejectReason]         = useState('')

  // 批量分配
  const [selectedIds, setSelectedIds]         = useState<Set<string>>(new Set())
  const [showAssignDialog, setShowAssignDialog] = useState(false)
  const [operators, setOperators]             = useState<OperatorInfo[]>([])
  const [selectedOperator, setSelectedOperator] = useState('')
  const [assigning, setAssigning]             = useState(false)

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

  // ==================== 数据分组 ====================

  // ① pending_finalize：骨干教师提交上来待学校管理员确认的
  //   - senior_operator/admin 可见
  const pendingFinalize = pipelines
    .filter(p => p.status === 'pending_finalize')
    .sort((a, b) => new Date(a.created_at || 0).getTime() - new Date(b.created_at || 0).getTime())

  // ② review_queue + needs_human：待骨干教师审核的
  //   - operator：看自己负责的（assigned_to 或 started_by）
  //   - senior_operator：原则上应看 pending_finalize，但也可辅助看 review_queue（此处显示全部供参考）
  //   - admin：全看
  const allReviewQueue = pipelines
    .filter(p => p.status === 'review_queue' || p.status === 'needs_human')
    .sort((a, b) => {
      // needs_human 优先
      if (a.status === 'needs_human' && b.status !== 'needs_human') return -1
      if (a.status !== 'needs_human' && b.status === 'needs_human') return 1
      // 高轮次优先
      if ((a.review_round || 1) > (b.review_round || 1)) return -1
      if ((a.review_round || 1) < (b.review_round || 1)) return 1
      return new Date(a.created_at || 0).getTime() - new Date(b.created_at || 0).getTime()
    })

  // 按 review_round 分为一审和二审
  const round1Queue = allReviewQueue.filter(p => (p.review_round || 1) === 1)
  const round2Queue = allReviewQueue.filter(p => (p.review_round || 1) >= 2)

  // ③ 已完成
  const completedReview = pipelines
    .filter(p => ['finalized', 'verified', 'verify_failed'].includes(p.status))
    .sort((a, b) => new Date(b.completed_at || b.created_at || 0).getTime() - new Date(a.completed_at || a.created_at || 0).getTime())
    .slice(0, 20)

  // ==================== 统计 ====================
  const todayFinalized = pipelines.filter(p => ['finalized', 'verified'].includes(p.status) && isToday(p.completed_at)).length
  const totalFinalized = pipelines.filter(p => ['finalized', 'verified'].includes(p.status)).length
  const verifiedCount  = pipelines.filter(p => p.status === 'verified').length
  const needsHumanCount = allReviewQueue.filter(p => p.status === 'needs_human').length

  // ==================== 操作 ====================

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

  const handleConfirmFinalize = async (p: PipelineListItem) => {
    if (!confirm('确认定稿归档 ' + p.course_code + '？')) return
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

  const openRejectDialog = (p: PipelineListItem) => {
    setRejectTargetId(p.id); setRejectTargetCode(p.course_code); setRejectReason(''); setShowRejectDialog(true)
  }

  const handleRejectFinalize = async () => {
    setActionLoading(true)
    try {
      await rejectFinalize(rejectTargetId, rejectReason)
      setToast({ message: rejectTargetCode + ' 已退回重审！', type: 'ok' })
      setShowRejectDialog(false); loadData()
    } catch (e: any) {
      setToast({ message: '退回失败: ' + (e.message || ''), type: 'err' })
    }
    setActionLoading(false)
  }

  // 批量分配
  const openAssignDialog = async () => {
    if (selectedIds.size === 0) { setToast({ message: '请先选择要分配的Pipeline', type: 'info' }); return }
    try { setOperators((await getOperators()) || []) } catch { /* ignore */ }
    setSelectedOperator(''); setShowAssignDialog(true)
  }

  const handleBatchAssign = async () => {
    if (!selectedOperator) { setToast({ message: '请选择审核员', type: 'info' }); return }
    setAssigning(true)
    try {
      const result = await batchAssignPipelines(Array.from(selectedIds), selectedOperator)
      setToast({ message: '分配成功: ' + result.success_count + ' 个Pipeline已分配给 ' + result.assigned_name, type: 'ok' })
      setShowAssignDialog(false); setSelectedIds(new Set()); loadData()
    } catch (e: any) {
      setToast({ message: '分配失败: ' + (e.message || ''), type: 'err' })
    }
    setAssigning(false)
  }

  const toggleSelect    = (id: string) => setSelectedIds(prev => { const n = new Set(prev); n.has(id) ? n.delete(id) : n.add(id); return n })
  const isAllR1Selected = round1Queue.length > 0 && round1Queue.every(p => selectedIds.has(p.id))
  const toggleAllR1     = () => {
    const ids = round1Queue.map(p => p.id)
    const all = ids.every(id => selectedIds.has(id))
    setSelectedIds(prev => { const n = new Set(prev); all ? ids.forEach(id => n.delete(id)) : ids.forEach(id => n.add(id)); return n })
  }

  const btn: React.CSSProperties = {
    padding: '7px 14px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 5,
  }

  // ==================== 渲染 ====================
  return (
    <div>
      {/* 顶部操作栏 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
        <p style={{ fontSize: 14, color: '#86868b', margin: 0 }}>
          {isSeniorOperator && !isAdmin
            ? '管理骨干教师提交的定稿申请，并进行最终审批'
            : isOperator
              ? '审核分配给你的课件，完成后提交定稿申请'
              : '统一管理所有待审核的Pipeline课程'}
        </p>
        <div style={{ display: 'flex', gap: 8 }}>
          {/* 批量分配：admin可见，且有review_queue时 */}
          {isAdmin && selectedIds.size > 0 && (
            <button style={{ ...btn, background: '#5856d6', color: '#fff', border: '1px solid #5856d6' }} onClick={openAssignDialog}>
              <UserPlus size={14} /> 分配审核员 ({selectedIds.size})
            </button>
          )}
          <button style={btn} onClick={loadData}><RefreshCw size={14} /> 刷新</button>
        </div>
      </div>

      {/* 统计卡片 — 按角色差异化 */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 24, flexWrap: 'wrap' }}>
        {/* operator：显示自己队列中的待审核数 */}
        {(isOperator || isAdmin) && (
          <StatCard label="待审核"
            value={allReviewQueue.length}
            icon={<ClipboardCheck size={18} color="#fff" />}
            color="linear-gradient(135deg,#ff9500,#ffcc00)"
            sub={needsHumanCount > 0 ? needsHumanCount + ' 个需人工干预' : undefined}
          />
        )}
        {/* senior_operator/admin：显示待确认定稿数 */}
        {isSuperReviewer && (
          <StatCard label="待确认定稿"
            value={pendingFinalize.length}
            icon={<Clock size={18} color="#fff" />}
            color="linear-gradient(135deg,#cc6600,#ff9500)"
            sub={pendingFinalize.length > 0 ? '等待学校管理员确认' : undefined}
          />
        )}
        {/* 二审队列数（对所有角色都有意义） */}
        {round2Queue.length > 0 && (
          <StatCard label="二审队列"
            value={round2Queue.length}
            icon={<AlertTriangle size={18} color="#fff" />}
            color="linear-gradient(135deg,#ff3b30,#ff6b6b)"
            sub="需重点关注"
          />
        )}
        <StatCard label="今日已审核" value={todayFinalized}
          icon={<CheckCircle size={18} color="#fff" />}
          color="linear-gradient(135deg,#34c759,#30d158)" />
        <StatCard label="总已定稿" value={totalFinalized}
          icon={<FileText size={18} color="#fff" />}
          color="linear-gradient(135deg,#007aff,#5ac8fa)" />
        <StatCard label="验收通过" value={verifiedCount}
          icon={<ShieldCheck size={18} color="#fff" />}
          color="linear-gradient(135deg,#32ade6,#5856d6)" />
      </div>

      {loading && (
        <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>
          <Loader size={24} style={{ animation: 'spin 1s linear infinite', marginBottom: 8 }} />
          <div style={{ fontSize: 14 }}>加载中...</div>
        </div>
      )}

      {/* ===== ① 待确认定稿（学校管理员/admin视角）===== */}
      {!loading && isSuperReviewer && (
        <QueueBlock
          title="待确认定稿"
          icon={<Clock size={18} color="#cc6600" />}
          titleColor="#cc6600" bgColor="rgba(255,149,0,0.04)" borderColor="rgba(204,102,0,0.15)"
          count={pendingFinalize.length}
          hint="骨干教师已提交定稿申请，等待学校管理员审批"
          empty="暂无待确认定稿，骨干教师尚未提交定稿申请"
        >
          {pendingFinalize.map((p, idx) => (
            <div key={p.id}
              style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '14px 20px', borderBottom: idx < pendingFinalize.length - 1 ? '1px solid rgba(0,0,0,0.03)' : 'none', transition: 'background 0.15s ease' }}
              onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(255,149,0,0.03)' }}
              onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
            >
              <div style={{ width: 4, height: 40, borderRadius: 2, flexShrink: 0, background: '#cc6600' }} />
              <div style={{ flex: 1, minWidth: 0, cursor: 'pointer' }} onClick={() => navigate('/workflow/pipelines/' + p.id + '/review')}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                  <span style={{ fontSize: 14, fontWeight: 700, color: '#1c1c1e' }}>{p.course_code}</span>
                  <StatusBadge status={p.status} />
                  <RoundBadge round={p.review_round || 1} />
                </div>
                <div style={{ fontSize: 12, color: '#86868b', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {p.course_name || p.course_code}
                  {p.assigned_name && <span style={{ marginLeft: 8, fontSize: 11, color: '#5856d6', fontWeight: 500 }}><Users size={10} style={{ verticalAlign: -1, marginRight: 2 }} />{p.assigned_name}</span>}
                </div>
              </div>
              <div style={{ display: 'flex', gap: 14, flexShrink: 0 }}>
                <ScoreDisplay value={p.eval_avg_score} label="评估" />
                <ScoreDisplay value={p.meta_score} label="Meta" />
                <ScoreDisplay value={p.translator_score} label="翻译" />
              </div>
              <div style={{ fontSize: 11, color: '#aeaeb2', flexShrink: 0, minWidth: 60, textAlign: 'right' }}>{formatTime(p.created_at)}</div>
              <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
                {/* 查看课件内容 */}
                <button title="查看课件内容" onClick={() => navigate('/workflow/pipelines/' + p.id + '/review')}
                  style={{ width: 30, height: 30, borderRadius: 8, border: '1px solid rgba(0,122,255,0.3)', background: 'rgba(0,122,255,0.08)', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#007aff', padding: 0 }}>
                  <ArrowRight size={14} />
                </button>
                {/* 退回重审 */}
                <button title="退回给骨干教师重审" onClick={() => openRejectDialog(p)} disabled={actionLoading}
                  style={{ width: 30, height: 30, borderRadius: 8, border: '1px solid rgba(255,59,48,0.3)', background: 'rgba(255,59,48,0.08)', cursor: actionLoading ? 'not-allowed' : 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#ff3b30', padding: 0, opacity: actionLoading ? 0.5 : 1 }}>
                  <RotateCcw size={14} />
                </button>
                {/* 确认定稿 */}
                <button title="确认定稿" onClick={() => handleConfirmFinalize(p)} disabled={actionLoading}
                  style={{ padding: '0 14px', height: 30, borderRadius: 8, border: 'none', background: '#34c759', cursor: actionLoading ? 'not-allowed' : 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: 12, fontWeight: 600, gap: 4, opacity: actionLoading ? 0.5 : 1 }}>
                  <ShieldCheck size={13} /> 确认定稿
                </button>
              </div>
            </div>
          ))}
        </QueueBlock>
      )}

      {/* ===== ② 一审队列（骨干教师/admin视角）===== */}
      {!loading && (isOperator || isAdmin) && (
        <QueueBlock
          title="一审队列"
          icon={<ClipboardCheck size={18} color="#ff9500" />}
          titleColor="#ff9500" bgColor="rgba(255,149,0,0.03)" borderColor="rgba(0,0,0,0.04)"
          count={round1Queue.length}
          badge={needsHumanCount > 0 ? (
            <span style={{ fontSize: 11, fontWeight: 600, color: '#cc9900', background: 'rgba(255,204,0,0.15)', padding: '2px 10px', borderRadius: 10, display: 'inline-flex', alignItems: 'center', gap: 4 }}>
              <AlertTriangle size={11} /> {needsHumanCount} 需人工
            </span>
          ) : undefined}
          showCheckbox={isAdmin && round1Queue.length > 0}
          isAllSelected={isAllR1Selected}
          onToggleAll={toggleAllR1}
          empty="一审队列为空，所有一审课程已处理完毕"
        >
          {round1Queue.map((p, idx) => (
            <PipelineRow key={p.id} p={p} idx={idx} total={round1Queue.length}
              showCheckbox={isAdmin} isSelected={selectedIds.has(p.id)} onToggle={() => toggleSelect(p.id)}
              onNavigate={() => navigate('/workflow/pipelines/' + p.id + '/review')}
              showActionBtns={canOperate}
              onMarkPassed={() => handleMarkPassed(p)}
              onReview={() => navigate('/workflow/pipelines/' + p.id + '/review')}
              accentColor="#ff9500"
            />
          ))}
        </QueueBlock>
      )}

      {/* ===== ③ 二审队列（骨干教师/admin视角）===== */}
      {!loading && (isOperator || isAdmin) && round2Queue.length > 0 && (
        <QueueBlock
          title="二审队列"
          icon={<AlertTriangle size={18} color="#ff3b30" />}
          titleColor="#ff3b30" bgColor="rgba(255,59,48,0.03)" borderColor="rgba(255,59,48,0.12)"
          count={round2Queue.length}
          hint="验收未通过，已自动重跑，需重新审核"
          empty="二审队列为空"
        >
          {round2Queue.map((p, idx) => (
            <PipelineRow key={p.id} p={p} idx={idx} total={round2Queue.length}
              showCheckbox={false} isSelected={false} onToggle={() => {}}
              onNavigate={() => navigate('/workflow/pipelines/' + p.id + '/review')}
              showActionBtns={canOperate}
              onMarkPassed={() => handleMarkPassed(p)}
              onReview={() => navigate('/workflow/pipelines/' + p.id + '/review')}
              accentColor="#ff3b30"
            />
          ))}
        </QueueBlock>
      )}

      {/* ===== ④ 已完成审核记录 ===== */}
      {!loading && completedReview.length > 0 && (
        <div style={{ background: '#fff', borderRadius: 16, border: '1px solid rgba(0,0,0,0.04)', boxShadow: '0 1px 3px rgba(0,0,0,0.04)', overflow: 'hidden' }}>
          <div
            style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '14px 20px', borderBottom: showCompleted ? '1px solid rgba(0,0,0,0.04)' : 'none', cursor: 'pointer' }}
            onClick={() => setShowCompleted(!showCompleted)}
            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,0,0,0.015)' }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
          >
            <CheckCircle size={18} color="#34c759" />
            <span style={{ fontSize: 15, fontWeight: 700, color: '#1c1c1e', flex: 1 }}>已完成审核</span>
            <span style={{ fontSize: 12, color: '#8e8e93', background: 'rgba(0,0,0,0.04)', padding: '2px 10px', borderRadius: 10, fontWeight: 500 }}>最近 {completedReview.length} 条</span>
            <span style={{ fontSize: 12, color: '#c7c7cc', transform: showCompleted ? 'rotate(90deg)' : 'none', display: 'inline-block', transition: 'transform 0.2s' }}>&#9654;</span>
          </div>
          {showCompleted && (
            <div>
              {completedReview.map((p, idx) => (
                <div key={p.id}
                  style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '12px 20px', borderBottom: idx < completedReview.length - 1 ? '1px solid rgba(0,0,0,0.03)' : 'none', cursor: 'pointer', opacity: 0.85, transition: 'all 0.15s ease' }}
                  onClick={() => navigate('/workflow/pipelines/' + p.id)}
                  onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.03)'; (e.currentTarget as HTMLElement).style.opacity = '1' }}
                  onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent'; (e.currentTarget as HTMLElement).style.opacity = '0.85' }}
                >
                  <div style={{ flexShrink: 0 }}>
                    {p.status === 'verified' ? <ShieldCheck size={18} color="#248a3d" /> : p.status === 'verify_failed' ? <XCircle size={18} color="#d70015" /> : <CheckCircle size={18} color="#34c759" />}
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <span style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e' }}>{p.course_code}</span>
                      <StatusBadge status={p.status} />
                      <RoundBadge round={p.review_round || 1} />
                    </div>
                    <div style={{ fontSize: 11, color: '#aeaeb2', marginTop: 2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{p.course_name || p.course_code}</div>
                  </div>
                  <div style={{ display: 'flex', gap: 12, flexShrink: 0 }}>
                    <ScoreDisplay value={p.meta_score} label="Meta" />
                    <ScoreDisplay value={p.translator_score} label="翻译" />
                  </div>
                  <div style={{ fontSize: 11, color: '#c7c7cc', flexShrink: 0, minWidth: 60, textAlign: 'right' }}>{formatTime(p.completed_at || p.created_at)}</div>
                  <div style={{ color: '#c7c7cc', fontSize: 14, flexShrink: 0 }}>&#8250;</div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`}</style>

      {/* 批量分配弹窗 */}
      {showAssignDialog && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center' }} onClick={e => { if (e.target === e.currentTarget && !assigning) setShowAssignDialog(false) }}>
          <div style={{ background: '#fff', borderRadius: 20, width: 440, maxWidth: '94vw', padding: 28, boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 20 }}>
              <UserPlus size={20} color="#5856d6" />
              <div style={{ fontSize: 18, fontWeight: 700, color: '#1c1c1e', flex: 1 }}>分配审核员</div>
              <button onClick={() => !assigning && setShowAssignDialog(false)} style={{ background: '#f2f2f7', border: 'none', borderRadius: '50%', width: 30, height: 30, display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}><X size={16} color="#8e8e93" /></button>
            </div>
            <div style={{ fontSize: 13, color: '#86868b', marginBottom: 16 }}>
              将 <span style={{ fontWeight: 600, color: '#1c1c1e' }}>{selectedIds.size}</span> 个Pipeline分配给指定审核员
            </div>
            <div style={{ marginBottom: 20 }}>
              {operators.length === 0
                ? <div style={{ fontSize: 13, color: '#aeaeb2' }}>暂无可分配的审核员</div>
                : operators.map(op => (
                  <div key={op.id} onClick={() => setSelectedOperator(op.id)} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '10px 14px', borderRadius: 10, cursor: 'pointer', border: selectedOperator === op.id ? '2px solid #5856d6' : '1px solid rgba(0,0,0,0.08)', background: selectedOperator === op.id ? 'rgba(88,86,214,0.05)' : '#fff', marginBottom: 6 }}>
                    <div style={{ width: 32, height: 32, borderRadius: '50%', background: selectedOperator === op.id ? '#5856d6' : 'rgba(88,86,214,0.1)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: selectedOperator === op.id ? '#fff' : '#5856d6', fontSize: 13, fontWeight: 600, flexShrink: 0 }}>{op.display_name.charAt(0)}</div>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>{op.display_name}</div>
                      <div style={{ fontSize: 11, color: '#86868b' }}>{op.username} · {op.role === 'admin' ? '系统管理员' : op.role === 'senior_operator' ? '学校管理员' : '骨干教师'}</div>
                    </div>
                    {selectedOperator === op.id && <CheckCircle size={18} color="#5856d6" />}
                  </div>
                ))}
            </div>
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
              <button onClick={() => !assigning && setShowAssignDialog(false)} style={{ padding: '10px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 14, fontWeight: 500, cursor: 'pointer', color: '#3c3c43' }}>取消</button>
              <button onClick={handleBatchAssign} disabled={!selectedOperator || assigning} style={{ padding: '10px 24px', borderRadius: 10, border: 'none', background: selectedOperator && !assigning ? '#5856d6' : '#c7c7cc', color: '#fff', fontSize: 14, fontWeight: 600, cursor: selectedOperator && !assigning ? 'pointer' : 'not-allowed' }}>{assigning ? '分配中...' : '确认分配'}</button>
            </div>
          </div>
        </div>
      )}

      {/* 退回重审弹窗 */}
      {showRejectDialog && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 40 }} onClick={e => { if (e.target === e.currentTarget && !actionLoading) setShowRejectDialog(false) }}>
          <div style={{ background: '#fff', borderRadius: 16, maxWidth: 480, width: '100%', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '16px 20px', borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
              <RotateCcw size={16} color="#ff3b30" />
              <span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e', flex: 1 }}>退回重审 — {rejectTargetCode}</span>
              <button onClick={() => !actionLoading && setShowRejectDialog(false)} style={{ background: '#f2f2f7', border: 'none', borderRadius: '50%', width: 28, height: 28, display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}><X size={14} color="#8e8e93" /></button>
            </div>
            <div style={{ padding: '16px 20px' }}>
              <div style={{ fontSize: 13, color: '#3c3c43', marginBottom: 10 }}>退回给骨干教师重新审核，可填写退回原因（选填）：</div>
              <textarea value={rejectReason} onChange={e => setRejectReason(e.target.value)} placeholder="请说明退回原因..." style={{ width: '100%', minHeight: 100, border: '1px solid rgba(0,0,0,0.1)', borderRadius: 10, padding: 12, fontSize: 13, lineHeight: 1.6, resize: 'vertical', outline: 'none', boxSizing: 'border-box', fontFamily: 'inherit' }} />
            </div>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, padding: '12px 20px', borderTop: '1px solid rgba(0,0,0,0.06)' }}>
              <button onClick={() => !actionLoading && setShowRejectDialog(false)} style={{ padding: '8px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer', color: '#3c3c43' }}>取消</button>
              <button onClick={handleRejectFinalize} disabled={actionLoading} style={{ padding: '8px 24px', borderRadius: 10, border: 'none', background: actionLoading ? '#e5e5ea' : '#ff3b30', color: actionLoading ? '#aeaeb2' : '#fff', fontSize: 13, fontWeight: 600, cursor: actionLoading ? 'not-allowed' : 'pointer' }}>{actionLoading ? '处理中...' : '确认退回'}</button>
            </div>
          </div>
        </div>
      )}

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  )
}
