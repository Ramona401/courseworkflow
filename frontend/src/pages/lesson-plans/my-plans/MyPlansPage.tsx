/**
 * 我的教案页面 — MyPlansPage（Phase4完整实现）
 *
 * 功能：
 *   1. 教案列表展示（卡片式，含状态标签/学科/年级/AI评分）
 *   2. 状态筛选栏（全部/草稿/已发布/评审中/已共享/开发中）
 *   3. 学科+年级二级筛选
 *   4. 每张卡片的状态操作按钮 + 查看详情入口
 *   5. 空状态引导到备课工坊
 *   6. 从WorkshopPage跳转后自动刷新
 *
 * PRD §7.3 个人视角教案列表
 * PRD §7.1 状态流转双路径
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getLessonPlans,
  deleteLessonPlan,
  publishLessonPlanPersonal,
  submitLessonPlanForReview,
  startDevelopment,
  type LessonPlan,
  type LessonPlanStatus,
} from '@/api/lesson-plans'

/* ==================== 样式常量（PRD §8.2）==================== */
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
  borderHover:  '#E5E7EB',
}

/* ==================== 状态配置 ==================== */
interface StatusConfig {
  label: string
  color: string
  bg: string
  dot: string
  desc: string
}

const STATUS_CONFIG: Record<LessonPlanStatus, StatusConfig> = {
  draft: {
    label: '草稿', color: '#6B7280', bg: '#F3F4F6', dot: '#9CA3AF',
    desc: '尚未发布，仅自己可见',
  },
  published_personal: {
    label: '已发布', color: C.primary, bg: C.primaryLight, dot: C.primary,
    desc: '个人发布，可进入课件开发',
  },
  submitted: {
    label: '待评审', color: C.accent, bg: 'rgba(245,158,11,0.08)', dot: C.accent,
    desc: '已提交教研组，等待评审',
  },
  revision: {
    label: '退回修改', color: C.warning, bg: 'rgba(249,115,22,0.08)', dot: C.warning,
    desc: '评审退回，需修改后重新提交',
  },
  approved: {
    label: '评审通过', color: C.success, bg: 'rgba(16,185,129,0.08)', dot: C.success,
    desc: '已通过教研组评审',
  },
  published_shared: {
    label: '已共享', color: '#8B5CF6', bg: 'rgba(139,92,246,0.08)', dot: '#8B5CF6',
    desc: '已共享到教研组/学校，其他老师可查看',
  },
  developing: {
    label: '课件开发中', color: '#0EA5E9', bg: 'rgba(14,165,233,0.08)', dot: '#0EA5E9',
    desc: '已进入课件开发流程，教案已锁定',
  },
  completed: {
    label: '已完成', color: C.success, bg: 'rgba(16,185,129,0.08)', dot: C.success,
    desc: '课件开发完成',
  },
}

/* ==================== 筛选栏配置 ==================== */
const STATUS_FILTERS: Array<{ key: string; label: string; statuses: LessonPlanStatus[] | null }> = [
  { key: 'all',      label: '全部',   statuses: null },
  { key: 'draft',    label: '草稿',   statuses: ['draft'] },
  { key: 'personal', label: '已发布', statuses: ['published_personal'] },
  { key: 'review',   label: '评审中', statuses: ['submitted', 'revision', 'approved'] },
  { key: 'shared',   label: '已共享', statuses: ['published_shared'] },
  { key: 'dev',      label: '开发中', statuses: ['developing', 'completed'] },
]

const SUBJECTS = ['全部', 'AI', '人工智能', '语文', '数学', '英语', '物理', '化学', '生物', '历史', '地理', '政治', '信息技术']
const GRADES   = ['全部', '七年级', '八年级', '九年级', '高一', '高二', '高三', '小学低段', '小学中段', '小学高段']

/* ==================== 子组件：状态徽标 ==================== */
function StatusBadge({ status }: { status: LessonPlanStatus }) {
  const cfg = STATUS_CONFIG[status] || STATUS_CONFIG.draft
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: '5px',
      padding: '3px 8px', borderRadius: '20px',
      background: cfg.bg, fontSize: '12px', fontWeight: 500, color: cfg.color,
      whiteSpace: 'nowrap',
    }}>
      <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: cfg.dot, flexShrink: 0 }} />
      {cfg.label}
    </span>
  )
}

/* ==================== 子组件：元信息标签 ==================== */
function MetaTag({ icon, text, maxWidth }: { icon: string; text: string; maxWidth?: string }) {
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: '4px',
      fontSize: '13px', color: C.textSec,
      maxWidth, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
    }}>
      <span style={{ flexShrink: 0 }}>{icon}</span>
      <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{text}</span>
    </span>
  )
}

/* ==================== 子组件：操作按钮组 ==================== */
interface ActionButtonsProps {
  plan: LessonPlan
  onAction: (planId: string, action: string) => void
  loadingId: string | null
}

function ActionButtons({ plan, onAction, loadingId }: ActionButtonsProps) {
  const isLoading = loadingId === plan.id
  const navigate  = useNavigate()

  const btnBase: React.CSSProperties = {
    padding: '5px 12px', borderRadius: '6px',
    fontSize: '12px', fontWeight: 500,
    cursor: isLoading ? 'not-allowed' : 'pointer',
    border: 'none', transition: 'all 150ms ease', whiteSpace: 'nowrap',
  }
  const primaryBtn: React.CSSProperties = {
    ...btnBase, background: isLoading ? C.border : C.primary,
    color: isLoading ? C.textMuted : '#fff',
  }
  const secondaryBtn: React.CSSProperties = {
    ...btnBase, background: 'transparent',
    border: `1px solid ${C.border}`, color: C.textSec,
  }
  const dangerBtn: React.CSSProperties = {
    ...btnBase, background: 'transparent',
    border: '1px solid #FEE2E2', color: C.danger,
  }

  const actions: Array<{ label: string; style: React.CSSProperties; action: string; confirm?: string }> = []

  switch (plan.status) {
    case 'draft':
      actions.push({ label: '✏️ 继续备课', style: primaryBtn,   action: 'resume'  })
      actions.push({ label: '发布教案',   style: secondaryBtn, action: 'publish' })
      actions.push({ label: '提交评审',   style: secondaryBtn, action: 'submit'  })
      break
    case 'published_personal':
      actions.push({ label: '✏️ 继续备课',  style: secondaryBtn, action: 'resume'  })
      actions.push({ label: '进入课件开发', style: primaryBtn,   action: 'develop' })
      actions.push({ label: '提交评审',    style: secondaryBtn, action: 'submit'  })
      break
    case 'submitted':
      break
    case 'revision':
      actions.push({ label: '修改后重提', style: primaryBtn, action: 'submit' })
      break
    case 'approved':
    case 'published_shared':
      actions.push({ label: '进入课件开发', style: primaryBtn, action: 'develop' })
      break
    case 'developing':
      actions.push({ label: '查看课件进度', style: primaryBtn, action: 'view_pipeline' })
      break
    case 'completed':
      break
  }

  if (['draft', 'published_personal', 'revision'].includes(plan.status)) {
    actions.push({
      label: '删除', style: dangerBtn, action: 'delete',
      confirm: `确定删除教案「${plan.title}」吗？此操作不可恢复。`,
    })
  }

  return (
    <div style={{ display: 'flex', gap: '6px', alignItems: 'center', flexWrap: 'wrap' }}>
      {isLoading && <span style={{ fontSize: '12px', color: C.primary }}>处理中...</span>}
      {!isLoading && actions.map(a => (
        <button
          key={a.action}
          style={a.style}
          onClick={e => {
            e.stopPropagation()
            if (a.confirm && !window.confirm(a.confirm)) return
            if (a.action === 'view_pipeline') { navigate('/workflow/pipelines'); return }
            onAction(plan.id, a.action)
          }}
        >
          {a.label}
        </button>
      ))}
    </div>
  )
}

/* ==================== 子组件：教案卡片 ==================== */
interface PlanCardProps {
  plan: LessonPlan
  onAction: (planId: string, action: string) => void
  loadingId: string | null
}

function PlanCard({ plan, onAction, loadingId }: PlanCardProps) {
  const [hovered, setHovered] = useState(false)
  const navigate = useNavigate()
  const statusCfg = STATUS_CONFIG[plan.status] || STATUS_CONFIG.draft

  const formatDate = (iso: string) => {
    try {
      const d = new Date(iso)
      return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`
    } catch { return iso }
  }

  /* 点击卡片主体区域跳转详情（底部操作区不触发）*/
  const handleCardClick = () => {
    navigate(`/lesson-plans/plans/${plan.id}`, { state: { from: '/lesson-plans/my-plans' } })
  }

  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        background: C.card, borderRadius: '12px',
        border: `1px solid ${hovered ? C.borderHover : C.border}`,
        padding: '20px', transition: 'all 200ms ease',
        boxShadow: hovered ? '0 4px 16px rgba(0,0,0,0.08)' : '0 1px 3px rgba(0,0,0,0.04)',
        transform: hovered ? 'translateY(-2px)' : 'none',
        cursor: 'pointer',
      }}
      onClick={handleCardClick}
    >
      {/* 顶行：标题 + 状态 */}
      <div style={{
        display: 'flex', justifyContent: 'space-between',
        alignItems: 'flex-start', gap: '12px', marginBottom: '10px',
      }}>
        <h3 style={{
          fontSize: '15px', fontWeight: 600, color: C.text, margin: 0,
          lineHeight: 1.5, flex: 1, minWidth: 0,
          overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        }}>
          {plan.title}
        </h3>
        <StatusBadge status={plan.status} />
      </div>

      {/* 元信息 */}
      <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', marginBottom: '12px' }}>
        <MetaTag icon="📚" text={plan.subject} />
        <MetaTag icon="🎓" text={plan.grade} />
        <MetaTag icon="⏱"  text={`${plan.duration_minutes}分钟`} />
        {plan.topic && <MetaTag icon="📌" text={plan.topic} maxWidth="180px" />}
      </div>

      {/* AI评分 */}
      {plan.ai_review_score != null && (
        <div style={{
          display: 'inline-flex', alignItems: 'center', gap: '6px',
          padding: '4px 10px', borderRadius: '20px', marginBottom: '12px',
          background: plan.ai_review_score >= 8.5
            ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)',
        }}>
          <span style={{ fontSize: '12px' }}>🤖</span>
          <span style={{
            fontSize: '12px', fontWeight: 600,
            color: plan.ai_review_score >= 8.5 ? C.success : C.accent,
          }}>
            AI评分 {plan.ai_review_score.toFixed(1)}
          </span>
        </div>
      )}

      {/* 退回说明 */}
      {plan.status === 'revision' && (
        <div style={{
          padding: '8px 12px', background: 'rgba(249,115,22,0.06)',
          borderRadius: '8px', fontSize: '12px', color: C.warning,
          marginBottom: '12px', lineHeight: 1.6,
        }}>
          💬 {statusCfg.desc}
        </div>
      )}

      {/* 底行：时间 + 操作（阻止冒泡，不触发卡片跳转）*/}
      <div
        style={{
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          paddingTop: '12px', borderTop: `1px solid ${C.border}`,
          gap: '12px', flexWrap: 'wrap',
        }}
        onClick={e => e.stopPropagation()}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '12px', color: C.textMuted, flexShrink: 0 }}>
            更新于 {formatDate(plan.updated_at)}
          </span>
          {/* 查看详情文字链接 */}
          <button
            onClick={e => {
              e.stopPropagation()
              navigate(`/lesson-plans/plans/${plan.id}`, { state: { from: '/lesson-plans/my-plans' } })
            }}
            style={{
              padding: '0', background: 'none', border: 'none',
              fontSize: '12px', color: C.primary,
              cursor: 'pointer', textDecoration: 'underline',
              textDecorationColor: 'rgba(79,123,232,0.3)',
              flexShrink: 0,
            }}
          >
            查看详情
          </button>
        </div>
        <ActionButtons plan={plan} onAction={onAction} loadingId={loadingId} />
      </div>
    </div>
  )
}

/* ==================== 子组件：骨架屏 ==================== */
function SkeletonCard() {
  const shimmer: React.CSSProperties = {
    background: 'linear-gradient(90deg, #F3F4F6 25%, #E5E7EB 50%, #F3F4F6 75%)',
    backgroundSize: '200% 100%',
    animation: 'shimmer 1.4s infinite',
    borderRadius: '4px',
  }
  return (
    <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '20px' }}>
      <style>{`@keyframes shimmer { 0%{background-position:200% 0} 100%{background-position:-200% 0} }`}</style>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '12px' }}>
        <div style={{ ...shimmer, width: '55%', height: '18px' }} />
        <div style={{ ...shimmer, width: '18%', height: '18px', borderRadius: '20px' }} />
      </div>
      <div style={{ display: 'flex', gap: '10px', marginBottom: '14px' }}>
        <div style={{ ...shimmer, width: '60px', height: '14px' }} />
        <div style={{ ...shimmer, width: '60px', height: '14px' }} />
        <div style={{ ...shimmer, width: '60px', height: '14px' }} />
      </div>
      <div style={{ ...shimmer, width: '100%', height: '1px', marginBottom: '12px' }} />
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <div style={{ ...shimmer, width: '30%', height: '12px' }} />
        <div style={{ ...shimmer, width: '25%', height: '26px', borderRadius: '6px' }} />
      </div>
    </div>
  )
}

/* ==================== 子组件：空状态 ==================== */
function EmptyState({ filtered, onReset }: { filtered: boolean; onReset: () => void }) {
  const navigate = useNavigate()
  return (
    <div style={{
      gridColumn: '1 / -1', textAlign: 'center', padding: '80px 40px',
      background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`,
    }}>
      <div style={{ fontSize: '48px', marginBottom: '16px' }}>{filtered ? '🔍' : '📋'}</div>
      <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
        {filtered ? '没有符合条件的教案' : '还没有教案'}
      </div>
      <div style={{ fontSize: '14px', color: C.textMuted, marginBottom: '24px', lineHeight: 1.7 }}>
        {filtered ? '试试调整筛选条件，或清空筛选查看全部' : '前往备课工坊创建您的第一份AI辅助教案'}
      </div>
      {filtered ? (
        <button
          onClick={onReset}
          style={{
            padding: '10px 24px', borderRadius: '8px',
            border: `1px solid ${C.border}`, background: 'transparent',
            fontSize: '14px', color: C.textSec, cursor: 'pointer',
          }}
        >清空筛选</button>
      ) : (
        <button
          onClick={() => navigate('/lesson-plans')}
          style={{
            padding: '10px 24px', borderRadius: '8px', border: 'none',
            background: C.primary, color: '#fff',
            fontSize: '14px', fontWeight: 600, cursor: 'pointer',
          }}
        >✨ 去备课工坊</button>
      )}
    </div>
  )
}

/* ==================== 主组件 ==================== */
export default function MyPlansPage() {
  const { user } = useAuth()
  const navigate = useNavigate()

  const [plans, setPlans]     = useState<LessonPlan[]>([])
  const [total, setTotal]     = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState<string | null>(null)

  const [statusFilter,  setStatusFilter]  = useState<string>('all')
  const [subjectFilter, setSubjectFilter] = useState('全部')
  const [gradeFilter,   setGradeFilter]   = useState('全部')

  const [loadingId, setLoadingId] = useState<string | null>(null)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  /* 加载教案列表 */
  const loadPlans = useCallback(async () => {
    if (!user) return
    setLoading(true); setError(null)
    try {
      const statusGroup = STATUS_FILTERS.find(f => f.key === statusFilter)
      const needsClientFilter = statusGroup?.statuses && statusGroup.statuses.length > 1

      const params: Record<string, string | number> = { author_id: user.id, limit: 100 }
      if (statusGroup?.statuses?.length === 1) params.status = statusGroup.statuses[0]
      if (subjectFilter !== '全部') params.subject = subjectFilter
      if (gradeFilter   !== '全部') params.grade   = gradeFilter

      const resp = await getLessonPlans(params)
      let list = resp.lesson_plans || []

      if (needsClientFilter && statusGroup?.statuses) {
        const allowed = new Set(statusGroup.statuses)
        list = list.filter(p => allowed.has(p.status))
      }

      setPlans(list); setTotal(list.length)
    } catch (e) {
      console.error('加载教案列表失败:', e)
      setError('加载失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }, [user, statusFilter, subjectFilter, gradeFilter])

  useEffect(() => { loadPlans() }, [loadPlans])

  const handleReset = () => {
    setStatusFilter('all'); setSubjectFilter('全部'); setGradeFilter('全部')
  }

  const isFiltered = statusFilter !== 'all' || subjectFilter !== '全部' || gradeFilter !== '全部'
  const allCount   = total

  /* 操作处理 */
  const handleAction = async (planId: string, action: string) => {
    if (loadingId) return
    setLoadingId(planId)
    try {
      switch (action) {
        case 'resume':
          // 继续编辑草稿：跳转到备课工坊并恢复对话
          navigate('/lesson-plans', { state: { resumePlanId: planId } })
          setLoadingId(null)
          return
        case 'publish':  await publishLessonPlanPersonal(planId); showToast('教案已个人发布 ✓'); break
        case 'submit':   await submitLessonPlanForReview(planId); showToast('已提交教研组评审 ✓'); break
        case 'develop':  await startDevelopment(planId);          showToast('已进入课件开发流程 ✓'); break
        case 'delete':   await deleteLessonPlan(planId);          showToast('教案已删除'); break
        default: console.warn('未知操作:', action)
      }
      await loadPlans()
    } catch (e: unknown) {
      console.error(`操作${action}失败:`, e)
      showToast(e instanceof Error ? e.message : '操作失败，请稍后重试', 'error')
    } finally {
      setLoadingId(null)
    }
  }

  /* 各状态计数 */
  const statusCounts = plans.reduce<Record<string, number>>((acc, p) => {
    acc[p.status] = (acc[p.status] || 0) + 1; return acc
  }, {})

  /* ==================== 渲染 ==================== */
  return (
    <div>
      {/* 页面标题区 */}
      <div style={{ marginBottom: '24px', display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <h1 style={{ fontSize: '20px', fontWeight: 600, color: C.text, margin: '0 0 4px 0' }}>我的教案</h1>
          <p style={{ fontSize: '14px', color: C.textSec, margin: 0 }}>
            共 {allCount} 份教案{isFiltered ? `（筛选后 ${plans.length} 份）` : ''}
          </p>
        </div>
        <button
          onClick={() => navigate('/lesson-plans')}
          style={{
            display: 'flex', alignItems: 'center', gap: '6px',
            padding: '9px 18px', borderRadius: '8px', border: 'none',
            background: C.primary, color: '#fff',
            fontSize: '14px', fontWeight: 600, cursor: 'pointer',
            transition: 'all 150ms ease', flexShrink: 0,
          }}
          onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.opacity = '0.88' }}
          onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.opacity = '1' }}
        >
          <span>✨</span><span>新建教案</span>
        </button>
      </div>

      {/* 筛选栏 */}
      <div style={{
        background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`,
        padding: '16px 20px', marginBottom: '20px',
      }}>
        {/* 状态筛选 */}
        <div style={{ marginBottom: '12px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec, marginRight: '12px' }}>状态</span>
          <div style={{ display: 'inline-flex', flexWrap: 'wrap', gap: '6px' }}>
            {STATUS_FILTERS.map(f => {
              const isActive = statusFilter === f.key
              const count = f.statuses
                ? f.statuses.reduce((s, st) => s + (statusCounts[st] || 0), 0)
                : allCount
              return (
                <button
                  key={f.key}
                  onClick={() => setStatusFilter(f.key)}
                  style={{
                    padding: '5px 12px', borderRadius: '20px',
                    border: `1px solid ${isActive ? C.primary : C.border}`,
                    background: isActive ? C.primaryLight : 'transparent',
                    color: isActive ? C.primary : C.textSec,
                    fontSize: '13px', fontWeight: isActive ? 600 : 400,
                    cursor: 'pointer', transition: 'all 150ms ease',
                  }}
                >
                  {f.label}
                  {count > 0 && (
                    <span style={{
                      marginLeft: '5px', padding: '0 5px',
                      background: isActive ? C.primary : C.border,
                      color: isActive ? '#fff' : C.textMuted,
                      borderRadius: '10px', fontSize: '11px', fontWeight: 600,
                    }}>{count}</span>
                  )}
                </button>
              )
            })}
          </div>
        </div>

        {/* 学科+年级 */}
        <div style={{ display: 'flex', gap: '16px', flexWrap: 'wrap', alignItems: 'center' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec, flexShrink: 0 }}>学科</span>
            <select
              value={subjectFilter} onChange={e => setSubjectFilter(e.target.value)}
              style={{
                padding: '5px 10px', borderRadius: '6px',
                border: `1px solid ${subjectFilter !== '全部' ? C.primary : C.border}`,
                background: subjectFilter !== '全部' ? C.primaryLight : 'transparent',
                color: subjectFilter !== '全部' ? C.primary : C.textSec,
                fontSize: '13px', cursor: 'pointer', outline: 'none',
              }}
            >
              {SUBJECTS.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec, flexShrink: 0 }}>年级</span>
            <select
              value={gradeFilter} onChange={e => setGradeFilter(e.target.value)}
              style={{
                padding: '5px 10px', borderRadius: '6px',
                border: `1px solid ${gradeFilter !== '全部' ? C.primary : C.border}`,
                background: gradeFilter !== '全部' ? C.primaryLight : 'transparent',
                color: gradeFilter !== '全部' ? C.primary : C.textSec,
                fontSize: '13px', cursor: 'pointer', outline: 'none',
              }}
            >
              {GRADES.map(g => <option key={g} value={g}>{g}</option>)}
            </select>
          </div>
          {isFiltered && (
            <button
              onClick={handleReset}
              style={{
                padding: '5px 10px', borderRadius: '6px', border: 'none',
                background: 'transparent', fontSize: '12px', color: C.textMuted,
                cursor: 'pointer', textDecoration: 'underline',
              }}
            >清空筛选</button>
          )}
        </div>
      </div>

      {/* 错误提示 */}
      {error && (
        <div style={{
          padding: '12px 16px', background: '#FEF2F2', border: '1px solid #FECACA',
          borderRadius: '8px', marginBottom: '16px', fontSize: '14px', color: C.danger,
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        }}>
          <span>⚠️ {error}</span>
          <button onClick={loadPlans} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.primary, fontSize: '13px' }}>重试</button>
        </div>
      )}

      {/* 教案卡片网格 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))', gap: '16px' }}>
        {loading && Array.from({ length: 6 }).map((_, i) => <SkeletonCard key={i} />)}
        {!loading && plans.map(plan => (
          <PlanCard key={plan.id} plan={plan} onAction={handleAction} loadingId={loadingId} />
        ))}
        {!loading && !error && plans.length === 0 && (
          <EmptyState filtered={isFiltered} onReset={handleReset} />
        )}
      </div>

      {/* Toast */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)',
          padding: '12px 24px', borderRadius: '10px',
          background: toast.type === 'error' ? '#FEF2F2' : '#1F2937',
          color: toast.type === 'error' ? C.danger : '#fff',
          fontSize: '14px', fontWeight: 500,
          boxShadow: '0 8px 24px rgba(0,0,0,0.15)', zIndex: 9999, whiteSpace: 'nowrap',
          border: toast.type === 'error' ? '1px solid #FECACA' : 'none',
          animation: 'toast-in 200ms ease',
        }}>
          <style>{`@keyframes toast-in { from{opacity:0;transform:translateX(-50%) translateY(8px)} to{opacity:1;transform:translateX(-50%) translateY(0)} }`}</style>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}
