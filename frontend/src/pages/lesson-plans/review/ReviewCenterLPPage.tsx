/**
 * 教研组评审中心 — ReviewCenterLPPage（列表页）
 *
 * v106重构：
 *   - 列表页只展示待评审/已评审列表，不再内联展开
 *   - 点击「开始评审」跳转到独立全屏评审页 /lesson-plans/review/:id
 *   - 独立页面有足够空间展示教案预览 + 评审面板 + AI辅助侧边栏三列布局
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getLessonPlans,
  getMyGroups,
  type LessonPlan,
  type TeachingGroup,
} from '@/api/lesson-plans'

/* ==================== 样式常量 ==================== */
const C = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent:       '#F59E0B',
  success:      '#10B981',
  warning:      '#F97316',
  danger:       '#EF4444',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
}

type ReviewTab = 'pending' | 'done'

const DECISION_CONFIG = {
  approved: { label: '评审通过', color: C.success, bg: 'rgba(16,185,129,0.08)', icon: '✅' },
  revision: { label: '退回修改', color: C.warning, bg: 'rgba(249,115,22,0.08)', icon: '↩️' },
}

function formatDate(iso: string): string {
  try {
    const d = new Date(iso)
    return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`
  } catch { return iso }
}

/* ==================== 骨架屏 ==================== */
function SkeletonRow() {
  const shimmer: React.CSSProperties = {
    background: 'linear-gradient(90deg, #F3F4F6 25%, #E5E7EB 50%, #F3F4F6 75%)',
    backgroundSize: '200% 100%',
    animation: 'shimmer 1.4s infinite',
    borderRadius: '4px',
  }
  return (
    <div style={{ padding: '20px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', gap: '16px', alignItems: 'center' }}>
      <style>{`@keyframes shimmer { 0%{background-position:200% 0} 100%{background-position:-200% 0} }`}</style>
      <div style={{ flex: 1 }}>
        <div style={{ ...shimmer, width: '50%', height: '16px', marginBottom: '8px' }} />
        <div style={{ display: 'flex', gap: '8px' }}>
          <div style={{ ...shimmer, width: '60px', height: '12px' }} />
          <div style={{ ...shimmer, width: '60px', height: '12px' }} />
        </div>
      </div>
      <div style={{ ...shimmer, width: '80px', height: '32px', borderRadius: '8px' }} />
    </div>
  )
}

function EmptyState({ tab }: { tab: ReviewTab }) {
  return (
    <div style={{ textAlign: 'center', padding: '60px 40px', color: C.textMuted }}>
      <div style={{ fontSize: '40px', marginBottom: '12px' }}>{tab === 'pending' ? '🎉' : '📋'}</div>
      <div style={{ fontSize: '15px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>
        {tab === 'pending' ? '暂无待评审教案' : '暂无评审记录'}
      </div>
    </div>
  )
}

/* ==================== 待评审卡片 ==================== */
function PendingCard({ plan, onReview }: { plan: LessonPlan; onReview: (id: string) => void }) {
  const [hovered, setHovered] = useState(false)
  return (
    <div
      style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}`, background: hovered ? C.bg : C.card, transition: 'background 200ms ease' }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px' }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '6px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {plan.title}
          </div>
          <div style={{ display: 'flex', gap: '14px', flexWrap: 'wrap', fontSize: '13px', color: C.textSec, marginBottom: '6px' }}>
            <span>📚 {plan.subject}</span>
            <span>🎓 {plan.grade}</span>
            <span>⏱ {plan.duration_minutes}分钟</span>
            <span>📌 {plan.topic}</span>
          </div>
          <div style={{ display: 'flex', gap: '10px', alignItems: 'center', flexWrap: 'wrap' }}>
            <span style={{ fontSize: '12px', color: C.textSec }}>✍️ {plan.author_name || '教师'}</span>
            {plan.ai_review_score != null && (
              <span style={{ fontSize: '12px', padding: '2px 8px', borderRadius: '20px', background: plan.ai_review_score >= 8.5 ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)', color: plan.ai_review_score >= 8.5 ? C.success : C.accent, fontWeight: 600 }}>
                🤖 {plan.ai_review_score.toFixed(1)}
              </span>
            )}
            <span style={{ fontSize: '12px', color: C.textMuted }}>提交于 {formatDate(plan.updated_at)}</span>
          </div>
        </div>
        <button
          onClick={() => onReview(plan.id)}
          style={{ padding: '9px 20px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer', flexShrink: 0, whiteSpace: 'nowrap' }}
        >
          开始评审 →
        </button>
      </div>
    </div>
  )
}

/* ==================== 已评审卡片 ==================== */
function DoneCard({ plan, onView }: { plan: LessonPlan; onView: (id: string) => void }) {
  const [hovered, setHovered] = useState(false)
  const cfg = DECISION_CONFIG[plan.status === 'approved' ? 'approved' : 'revision']
  return (
    <div
      style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}`, background: hovered ? C.bg : C.card, transition: 'background 200ms ease', cursor: 'pointer' }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onClick={() => onView(plan.id)}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px' }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '6px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {plan.title}
          </div>
          <div style={{ display: 'flex', gap: '12px', fontSize: '13px', color: C.textSec, flexWrap: 'wrap' }}>
            <span>📚 {plan.subject}</span>
            <span>🎓 {plan.grade}</span>
            <span>✍️ {plan.author_name || '教师'} 提交</span>
            <span>更新于 {formatDate(plan.updated_at)}</span>
          </div>
        </div>
        <div style={{ flexShrink: 0, display: 'flex', alignItems: 'center', gap: '10px' }}>
          {plan.ai_review_score != null && (
            <span style={{ fontSize: '12px', color: plan.ai_review_score >= 8.5 ? C.success : C.accent, fontWeight: 600 }}>
              🤖 {plan.ai_review_score.toFixed(1)}
            </span>
          )}
          <span style={{ padding: '4px 10px', borderRadius: '20px', background: cfg.bg, color: cfg.color, fontSize: '12px', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '4px' }}>
            {cfg.icon} {cfg.label}
          </span>
        </div>
      </div>
    </div>
  )
}

/* ==================== 主组件 ==================== */
export default function ReviewCenterLPPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [activeTab, setActiveTab]       = useState<ReviewTab>('pending')
  const [pendingPlans, setPendingPlans] = useState<LessonPlan[]>([])
  const [donePlans, setDonePlans]       = useState<LessonPlan[]>([])
  const [loading, setLoading]           = useState(true)
  const [error, setError]               = useState<string | null>(null)
  const [myGroups, setMyGroups]         = useState<TeachingGroup[]>([])
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type }); setTimeout(() => setToast(null), 3000)
  }
  void showToast

  useEffect(() => {
    getMyGroups().then(g => setMyGroups(g || [])).catch(() => setMyGroups([]))
  }, [])

  const loadPlans = useCallback(async () => {
    if (!user) return
    setLoading(true); setError(null)
    try {
      const params: Record<string, string | number> = { limit: 100 }
      if (myGroups.length > 0) params.group_id = myGroups[0].id
      const [submittedResp, approvedResp, revisionResp] = await Promise.all([
        getLessonPlans({ ...params, status: 'submitted' }),
        getLessonPlans({ ...params, status: 'approved' }),
        getLessonPlans({ ...params, status: 'revision' }),
      ])
      setPendingPlans(submittedResp.lesson_plans || [])
      const done = [
        ...(approvedResp.lesson_plans || []),
        ...(revisionResp.lesson_plans || []),
      ].sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
      setDonePlans(done)
    } catch (e) {
      console.error('加载评审列表失败:', e)
      setError('加载失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }, [user, myGroups])

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { loadPlans() }, [loadPlans])

  const handleReview = (planId: string) => {
    navigate(`/lesson-plans/review/${planId}`)
  }

  const handleView = (planId: string) => {
    navigate(`/lesson-plans/plans/${planId}`, { state: { from: '/lesson-plans/review' } })
  }

  return (
    <div>
      <div style={{ marginBottom: '20px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <p style={{ fontSize: '14px', color: C.textSec, margin: 0 }}>
          审核组内教师提交的教案，点击「开始评审」进入全屏评审工作台（含AI助手）
        </p>
        {!loading && myGroups.length === 0 && (
          <div style={{ padding: '8px 14px', background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.2)', borderRadius: '8px', fontSize: '13px', color: C.warning, flexShrink: 0 }}>
            💡 你尚未加入教研组，暂时看不到待评审教案
          </div>
        )}
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: '10px', marginBottom: '20px' }}>
        {[
          { label: '待评审', value: pendingPlans.length, color: C.accent,   bg: 'rgba(245,158,11,0.08)',  icon: '📋' },
          { label: '已通过', value: donePlans.filter(p => p.status === 'approved').length, color: C.success, bg: 'rgba(16,185,129,0.08)', icon: '✅' },
          { label: '已退回', value: donePlans.filter(p => p.status === 'revision').length, color: C.warning, bg: 'rgba(249,115,22,0.08)', icon: '↩️' },
          { label: '合计',   value: pendingPlans.length + donePlans.length,  color: C.primary, bg: C.primaryLight, icon: '📊' },
        ].map(stat => (
          <div key={stat.label} style={{ padding: '14px 18px', background: stat.bg, borderRadius: '10px', border: `1px solid ${stat.color}20` }}>
            <div style={{ fontSize: '18px', marginBottom: '4px' }}>{stat.icon}</div>
            <div style={{ fontSize: '22px', fontWeight: 700, color: stat.color, lineHeight: 1 }}>{loading ? '—' : stat.value}</div>
            <div style={{ fontSize: '12px', color: C.textSec, marginTop: '4px' }}>{stat.label}</div>
          </div>
        ))}
      </div>

      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
        <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, padding: '0 4px' }}>
          {([
            { key: 'pending' as ReviewTab, label: '📋 待评审', count: pendingPlans.length },
            { key: 'done'    as ReviewTab, label: '✅ 已评审', count: donePlans.length    },
          ]).map(tab => {
            const isActive = activeTab === tab.key
            return (
              <button key={tab.key} onClick={() => setActiveTab(tab.key)}
                style={{ padding: '14px 20px', border: 'none', background: 'transparent', fontSize: '14px', fontWeight: isActive ? 600 : 400, color: isActive ? C.primary : C.textSec, cursor: 'pointer', borderBottom: isActive ? `2px solid ${C.primary}` : '2px solid transparent', marginBottom: '-1px', transition: 'all 150ms ease', display: 'flex', alignItems: 'center', gap: '8px' }}>
                {tab.label}
                {!loading && tab.count > 0 && (
                  <span style={{ padding: '1px 7px', borderRadius: '10px', background: isActive ? C.primary : C.border, color: isActive ? '#fff' : C.textMuted, fontSize: '11px', fontWeight: 700 }}>{tab.count}</span>
                )}
              </button>
            )
          })}
        </div>

        {error && (
          <div style={{ padding: '16px 24px', background: '#FEF2F2', fontSize: '14px', color: C.danger, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>⚠️ {error}</span>
            <button onClick={loadPlans} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.primary, fontSize: '13px' }}>重试</button>
          </div>
        )}

        {activeTab === 'pending' && (
          <div>
            {loading && [1,2,3].map(i => <SkeletonRow key={i} />)}
            {!loading && pendingPlans.length === 0 && <EmptyState tab="pending" />}
            {!loading && pendingPlans.map(plan => (
              <PendingCard key={plan.id} plan={plan} onReview={handleReview} />
            ))}
          </div>
        )}

        {activeTab === 'done' && (
          <div>
            {loading && [1,2,3].map(i => <SkeletonRow key={i} />)}
            {!loading && donePlans.length === 0 && <EmptyState tab="done" />}
            {!loading && donePlans.map(plan => (
              <DoneCard key={plan.id} plan={plan} onView={handleView} />
            ))}
          </div>
        )}
      </div>

      {toast && (
        <div style={{ position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)', padding: '12px 24px', borderRadius: '10px', background: toast.type === 'error' ? '#FEF2F2' : '#1F2937', color: toast.type === 'error' ? C.danger : '#fff', fontSize: '14px', fontWeight: 500, boxShadow: '0 8px 24px rgba(0,0,0,0.15)', zIndex: 9999, whiteSpace: 'nowrap', border: toast.type === 'error' ? '1px solid #FECACA' : 'none' }}>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}
