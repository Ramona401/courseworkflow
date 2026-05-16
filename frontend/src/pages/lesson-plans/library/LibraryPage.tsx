/**
 * 教案库页面 — LibraryPage（Phase4完整实现）
 *
 * 功能：
 *   1. 三视角Tab：教研组库 / 学校库 / 区域库
 *   2. 关键词 + 学科 + 年级三重筛选
 *   3. 教案卡片：作者/AI评分/状态/查看详情/Fork
 *   4. Fork功能：一键Fork到我的草稿
 *   5. 骨架屏/空状态/错误重试
 *
 * PRD §7.3 教研组视角教案库
 * v56修改：移除页面内重复标题（LPLayout header已有标题）
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getLessonPlans,
  forkLessonPlan,
  getMyGroups,
  type LessonPlan,
  type LessonPlanStatus,
  type TeachingGroup,
} from '@/api/lesson-plans'
import { SUBJECTS } from '@/pages/lesson-plans/workshop/components/workshopConstants'
import { toggleInteraction, getInteractions, type InteractionCounts } from '@/api/lesson-plan-interactions'

/* ==================== 样式常量 ==================== */
const C = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent:       '#F59E0B',
  success:      '#10B981',
  warning:      '#F97316',
  danger:       '#EF4444',
  purple:       '#8B5CF6',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
  borderHover:  '#E5E7EB',
}

/* ==================== 库视角配置 ==================== */
type LibraryScope = 'group' | 'school' | 'region'

const SCOPE_TABS = [
  { key: 'group'  as LibraryScope, label: '教研组库', icon: '👥', desc: '本教研组共享的教案' },
  { key: 'school' as LibraryScope, label: '学校库',   icon: '🏫', desc: '全校共享的优秀教案' },
  { key: 'region' as LibraryScope, label: '区域库',   icon: '🌏', desc: '跨校共享的精品教案' },
]

const GRADES   = ['全部', '七年级', '八年级', '九年级', '高一', '高二', '高三', '小学低段', '小学中段', '小学高段']

/* ==================== 状态配置 ==================== */
interface StatusConfig { label: string; color: string; bg: string; dot: string }

const STATUS_CONFIG: Partial<Record<LessonPlanStatus, StatusConfig>> = {
  approved:         { label: '评审通过', color: C.success, bg: 'rgba(16,185,129,0.08)', dot: C.success },
  published_shared: { label: '已共享',   color: C.purple,  bg: 'rgba(139,92,246,0.08)', dot: C.purple  },
}

/* ==================== 子组件：状态徽标 ==================== */
function StatusBadge({ status }: { status: LessonPlanStatus }) {
  const cfg = STATUS_CONFIG[status]
  if (!cfg) return null
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: '5px',
      padding: '3px 8px', borderRadius: '20px',
      background: cfg.bg, fontSize: '12px', fontWeight: 500, color: cfg.color, whiteSpace: 'nowrap',
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

/* ==================== 子组件：骨架屏 ==================== */
function SkeletonCard() {
  const shimmer: React.CSSProperties = {
    background: 'linear-gradient(90deg, #F3F4F6 25%, #E5E7EB 50%, #F3F4F6 75%)',
    backgroundSize: '200% 100%', animation: 'shimmer 1.4s infinite', borderRadius: '4px',
  }
  return (
    <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '20px' }}>
      <style>{`@keyframes shimmer { 0%{background-position:200% 0} 100%{background-position:-200% 0} }`}</style>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '12px' }}>
        <div style={{ ...shimmer, width: '55%', height: '18px' }} />
        <div style={{ ...shimmer, width: '18%', height: '18px', borderRadius: '20px' }} />
      </div>
      <div style={{ display: 'flex', gap: '10px', marginBottom: '14px' }}>
        {[1,2,3].map(i => <div key={i} style={{ ...shimmer, width: '60px', height: '14px' }} />)}
      </div>
      <div style={{ ...shimmer, width: '40%', height: '14px', marginBottom: '14px' }} />
      <div style={{ ...shimmer, width: '100%', height: '1px', marginBottom: '12px' }} />
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <div style={{ ...shimmer, width: '30%', height: '12px' }} />
        <div style={{ ...shimmer, width: '20%', height: '26px', borderRadius: '6px' }} />
      </div>
    </div>
  )
}

/* ==================== 子组件：空状态 ==================== */
function EmptyState({ filtered, scope, onReset }: { filtered: boolean; scope: LibraryScope; onReset: () => void }) {
  const navigate = useNavigate()
  const scopeLabel = SCOPE_TABS.find(t => t.key === scope)?.label || '教案库'
  return (
    <div style={{
      gridColumn: '1 / -1', textAlign: 'center', padding: '80px 40px',
      background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`,
    }}>
      <div style={{ fontSize: '48px', marginBottom: '16px' }}>{filtered ? '🔍' : '📚'}</div>
      <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
        {filtered ? '没有符合条件的教案' : `${scopeLabel}暂无共享教案`}
      </div>
      <div style={{ fontSize: '14px', color: C.textMuted, marginBottom: '24px', lineHeight: 1.7 }}>
        {filtered ? '试试调整筛选条件' : '评审通过的教案共享后将出现在这里'}
      </div>
      {filtered
        ? <button onClick={onReset} style={{ padding: '10px 24px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>清空筛选</button>
        : <button onClick={() => navigate('/lesson-plans')} style={{ padding: '10px 24px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>✨ 去备课工坊</button>
      }
    </div>
  )
}

/* ==================== 子组件：教案库卡片 ==================== */
interface LibraryCardProps {
  plan: LessonPlan
  currentUserId: string
  forkingId: string | null
  onFork: (plan: LessonPlan) => void
  /** v125新增：互动数据 */
  interactions?: InteractionCounts
  /** v125新增：切换互动 */
  onToggleInteraction?: (planId: string, type: 'like' | 'favorite') => void
}

function LibraryCard({ plan, currentUserId, forkingId, onFork, interactions, onToggleInteraction }: LibraryCardProps) {
  const [hovered, setHovered] = useState(false)
  const navigate   = useNavigate()
  const isForking  = forkingId === plan.id
  const isOwnPlan  = plan.author_id === currentUserId

  const formatDate = (iso: string) => {
    try {
      const d = new Date(iso)
      return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`
    } catch { return iso }
  }

  /* 点击卡片主体跳转详情 */
  const handleCardClick = () => {
    navigate(`/lesson-plans/plans/${plan.id}`, { state: { from: '/lesson-plans/library' } })
  }

  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onClick={handleCardClick}
      style={{
        background: C.card, borderRadius: '12px',
        border: `1px solid ${hovered ? C.borderHover : C.border}`,
        padding: '20px', transition: 'all 200ms ease',
        boxShadow: hovered ? '0 4px 16px rgba(0,0,0,0.08)' : '0 1px 3px rgba(0,0,0,0.04)',
        transform: hovered ? 'translateY(-2px)' : 'none',
        cursor: 'pointer',
      }}
    >
      {/* 顶行：标题 + 状态 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '12px', marginBottom: '10px' }}>
        <h3 style={{
          fontSize: '15px', fontWeight: 600, color: C.text, margin: 0,
          lineHeight: 1.5, flex: 1, minWidth: 0,
          overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        }}>{plan.title}</h3>
        <StatusBadge status={plan.status} />
      </div>

      {/* 元信息 */}
      <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', marginBottom: '10px' }}>
        <MetaTag icon="📚" text={plan.subject} />
        <MetaTag icon="🎓" text={plan.grade} />
        <MetaTag icon="⏱"  text={`${plan.duration_minutes}分钟`} />
        {plan.topic && <MetaTag icon="📌" text={plan.topic} maxWidth="160px" />}
      </div>

      {/* 作者 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '10px' }}>
        <div style={{
          width: '22px', height: '22px', borderRadius: '50%', flexShrink: 0,
          background: 'linear-gradient(135deg, #4F7BE8, #818CF8)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          fontSize: '11px', color: '#fff', fontWeight: 600,
        }}>
          {(plan.author_name || '教').charAt(0)}
        </div>
        <span style={{ fontSize: '13px', color: C.textSec }}>
          {plan.author_name || '教师'}
          {isOwnPlan && (
            <span style={{ marginLeft: '6px', fontSize: '11px', padding: '1px 6px', borderRadius: '10px', background: C.primaryLight, color: C.primary }}>我的</span>
          )}
        </span>
      </div>

      {/* AI评分 */}
      {plan.ai_review_score != null && (
        <div style={{ marginBottom: '12px' }}>
          <span style={{
            display: 'inline-flex', alignItems: 'center', gap: '4px',
            padding: '3px 8px', borderRadius: '20px', fontSize: '12px', fontWeight: 600,
            background: plan.ai_review_score >= 8.5 ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)',
            color: plan.ai_review_score >= 8.5 ? C.success : C.accent,
          }}>
            🤖 AI评分 {plan.ai_review_score.toFixed(1)}
          </span>
        </div>
      )}

      {/* 底行：时间 + 操作（阻止冒泡）*/}
      <div
        style={{
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          paddingTop: '12px', borderTop: `1px solid ${C.border}`, gap: '12px',
        }}
        onClick={e => e.stopPropagation()}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '12px', color: C.textMuted }}>
            {formatDate(plan.updated_at)}
          </span>
          {/* 查看详情链接 */}
          <button
            onClick={e => {
              e.stopPropagation()
              navigate(`/lesson-plans/plans/${plan.id}`, { state: { from: '/lesson-plans/library' } })
            }}
            style={{
              padding: '0', background: 'none', border: 'none',
              fontSize: '12px', color: C.primary, cursor: 'pointer',
              textDecoration: 'underline', textDecorationColor: 'rgba(79,123,232,0.3)',
            }}
          >查看详情</button>
          {/* v125新增：点赞/收藏小按钮（与卡片内其他按钮统一风格） */}
          {interactions && onToggleInteraction && (
            <>
              <button
                onClick={e => { e.stopPropagation(); onToggleInteraction(plan.id, 'like') }}
                title={interactions.is_liked ? '取消点赞' : '点赞'}
                style={{
                  padding: '3px 10px', borderRadius: '6px',
                  border: `1px solid ${interactions.is_liked ? 'rgba(239,68,68,0.25)' : C.border}`,
                  background: interactions.is_liked ? 'rgba(239,68,68,0.06)' : 'transparent',
                  fontSize: '12px', fontWeight: 500,
                  color: interactions.is_liked ? '#DC2626' : C.textMuted,
                  cursor: 'pointer', transition: 'all 150ms ease',
                  display: 'inline-flex', alignItems: 'center', gap: '4px',
                }}
              >👍{interactions.like_count > 0 ? ` ${interactions.like_count}` : ''}</button>
              <button
                onClick={e => { e.stopPropagation(); onToggleInteraction(plan.id, 'favorite') }}
                title={interactions.is_favorited ? '取消收藏' : '收藏'}
                style={{
                  padding: '3px 10px', borderRadius: '6px',
                  border: `1px solid ${interactions.is_favorited ? 'rgba(245,158,11,0.25)' : C.border}`,
                  background: interactions.is_favorited ? 'rgba(245,158,11,0.06)' : 'transparent',
                  fontSize: '12px', fontWeight: 500,
                  color: interactions.is_favorited ? '#D97706' : C.textMuted,
                  cursor: 'pointer', transition: 'all 150ms ease',
                  display: 'inline-flex', alignItems: 'center', gap: '4px',
                }}
              >📌{interactions.favorite_count > 0 ? ` ${interactions.favorite_count}` : ''}</button>
            </>
          )}
        </div>

        {/* Fork按钮 / 我的标记 */}
        {!isOwnPlan ? (
          <button
            onClick={e => { e.stopPropagation(); onFork(plan) }}
            disabled={isForking}
            style={{
              padding: '5px 14px', borderRadius: '6px',
              border: `1px solid ${isForking ? C.border : C.primary}`,
              background: isForking ? C.bg : C.primaryLight,
              color: isForking ? C.textMuted : C.primary,
              fontSize: '12px', fontWeight: 600,
              cursor: isForking ? 'not-allowed' : 'pointer',
              transition: 'all 150ms ease', whiteSpace: 'nowrap',
            }}
          >
            {isForking ? '处理中...' : '🔀 Fork'}
          </button>
        ) : (
          <span style={{ fontSize: '12px', color: C.textMuted, fontStyle: 'italic' }}>我发布的</span>
        )}
      </div>
    </div>
  )
}

/* ==================== 主组件 ==================== */
export default function LibraryPage() {
  const { user } = useAuth()
  const navigate  = useNavigate()

  const [scope, setScope]               = useState<LibraryScope>('group')
  const [keyword, setKeyword]           = useState('')
  const [subjectFilter, setSubjectFilter] = useState('全部')
  const [gradeFilter,   setGradeFilter]   = useState('全部')
  const [qualityFilter, setQualityFilter] = useState('全部')
  const [structFilter,  setStructFilter]  = useState('全部')

  const [plans, setPlans]     = useState<LessonPlan[]>([])
  const [total, setTotal]     = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState<string | null>(null)

  const [myGroups, setMyGroups]   = useState<TeachingGroup[]>([])
  const [forkingId, setForkingId] = useState<string | null>(null)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  // v125新增：教案互动数据（按 planId 索引）
  const [interactionsMap, setInteractionsMap] = useState<Record<string, InteractionCounts>>({})

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type }); setTimeout(() => setToast(null), 3000)
  }

  useEffect(() => {
    getMyGroups().then(g => setMyGroups(g || [])).catch(() => setMyGroups([]))
  }, [])

  const loadPlans = useCallback(async () => {
    if (!user) return
    setLoading(true); setError(null)
    try {
      const params: Record<string, string | number> = { limit: 100 }
      if (subjectFilter !== '全部') params.subject = subjectFilter
      if (gradeFilter   !== '全部') params.grade   = gradeFilter
      if (qualityFilter !== '全部') params.quality_level = qualityFilter
      if (structFilter  !== '全部') params.structure_type = structFilter
      if (scope === 'group' && myGroups.length > 0) params.group_id = myGroups[0].id

      const [sharedResp, approvedResp] = await Promise.all([
        getLessonPlans({ ...params, status: 'published_shared' }),
        getLessonPlans({ ...params, status: 'approved' }),
      ])

      const idSet = new Set<string>()
      let merged: LessonPlan[] = []
      for (const p of [...(sharedResp.lesson_plans || []), ...(approvedResp.lesson_plans || [])]) {
        if (!idSet.has(p.id)) { idSet.add(p.id); merged.push(p) }
      }

      if (keyword.trim()) {
        const kw = keyword.trim().toLowerCase()
        merged = merged.filter(p =>
          p.title.toLowerCase().includes(kw) ||
          p.topic.toLowerCase().includes(kw) ||
          p.subject.toLowerCase().includes(kw)
        )
      }

      merged.sort((a, b) => (b.ai_review_score ?? -1) - (a.ai_review_score ?? -1))
      setPlans(merged); setTotal(merged.length)

      // v125新增：批量加载互动数据（逐个请求，不阻断列表显示）
      const iMap: Record<string, InteractionCounts> = {}
      for (const p of merged) {
        try {
          const ic = await getInteractions(p.id)
          iMap[p.id] = ic
        } catch {
          iMap[p.id] = { like_count: 0, favorite_count: 0, is_liked: false, is_favorited: false }
        }
      }
      setInteractionsMap(iMap)
    } catch (e) {
      console.error('加载教案库失败:', e); setError('加载失败，请稍后重试')
    } finally { setLoading(false) }
  }, [user, scope, subjectFilter, gradeFilter, keyword, myGroups, qualityFilter, structFilter])

  useEffect(() => { loadPlans() }, [loadPlans])

  const handleReset = () => { setKeyword(''); setSubjectFilter('全部'); setGradeFilter('全部'); setQualityFilter('全部'); setStructFilter('全部') }
  const isFiltered  = keyword.trim() !== '' || subjectFilter !== '全部' || gradeFilter !== '全部' || qualityFilter !== '全部' || structFilter !== '全部'

  // v125新增：切换点赞/收藏
  const handleToggleInteraction = async (planId: string, type: 'like' | 'favorite') => {
    try {
      const resp = await toggleInteraction(planId, type)
      setInteractionsMap(prev => ({
        ...prev,
        [planId]: {
          ...prev[planId],
          ...(type === 'like'
            ? { like_count: resp.new_count, is_liked: resp.active }
            : { favorite_count: resp.new_count, is_favorited: resp.active }
          ),
        },
      }))
      showToast(resp.active
        ? (type === 'like' ? '已点赞 👍' : '已收藏 📌')
        : (type === 'like' ? '已取消点赞' : '已取消收藏')
      )
    } catch {
      showToast('操作失败，请重试', 'error')
    }
  }

  const handleFork = async (plan: LessonPlan) => {
    if (forkingId) return
    setForkingId(plan.id)
    try {
      const forked = await forkLessonPlan(plan.id)
      showToast(`已Fork到我的草稿：${forked.title} ✓`)
    } catch (e) {
      console.error('Fork失败:', e); showToast('Fork失败，请稍后重试', 'error')
    } finally { setForkingId(null) }
  }

  /* ==================== 渲染 ==================== */
  return (
    <div>
      {/* 描述 + 新建按钮（标题已在LPLayout header中显示，此处不再重复） */}
      <div style={{ marginBottom: '20px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <p style={{ fontSize: '14px', color: C.textSec, margin: 0 }}>浏览共享教案，Fork优秀教案到我的草稿进行微调</p>
        <button
          onClick={() => navigate('/lesson-plans')}
          style={{
            display: 'flex', alignItems: 'center', gap: '6px',
            padding: '9px 18px', borderRadius: '8px', border: 'none',
            background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600,
            cursor: 'pointer', flexShrink: 0,
          }}
          onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.opacity = '0.88' }}
          onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.opacity = '1' }}
        ><span>✨</span><span>新建教案</span></button>
      </div>

      {/* 视角Tab */}
      <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, marginBottom: '20px', gap: '4px' }}>
        {SCOPE_TABS.map(tab => {
          const isActive = scope === tab.key
          return (
            <button key={tab.key} onClick={() => { setScope(tab.key); handleReset() }} title={tab.desc}
              style={{
                padding: '12px 20px', border: 'none', background: 'transparent',
                fontSize: '14px', fontWeight: isActive ? 600 : 400,
                color: isActive ? C.primary : C.textSec, cursor: 'pointer',
                borderBottom: isActive ? `2px solid ${C.primary}` : '2px solid transparent',
                marginBottom: '-1px', transition: 'all 150ms ease',
                display: 'flex', alignItems: 'center', gap: '6px',
              }}
            >
              <span>{tab.icon}</span><span>{tab.label}</span>
              {!loading && total > 0 && isActive && (
                <span style={{ padding: '1px 7px', borderRadius: '10px', background: C.primary, color: '#fff', fontSize: '11px', fontWeight: 700 }}>{total}</span>
              )}
            </button>
          )
        })}
      </div>

      {/* 筛选栏 */}
      <div style={{
        background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`,
        padding: '16px 20px', marginBottom: '20px',
        display: 'flex', gap: '16px', flexWrap: 'wrap', alignItems: 'center',
      }}>
        <div style={{ flex: '1 1 200px', minWidth: '160px' }}>
          <input
            type="text" value={keyword} onChange={e => setKeyword(e.target.value)}
            placeholder="搜索标题、课题、学科..."
            style={{
              width: '100%', padding: '7px 12px', borderRadius: '8px',
              border: `1px solid ${keyword ? C.primary : C.border}`,
              background: keyword ? C.primaryLight : 'transparent',
              fontSize: '13px', color: C.text, outline: 'none', boxSizing: 'border-box',
            }}
            onFocus={e => { e.target.style.borderColor = C.primary }}
            onBlur={e  => { e.target.style.borderColor = keyword ? C.primary : C.border }}
          />
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec, flexShrink: 0 }}>学科</span>
          <select value={subjectFilter} onChange={e => setSubjectFilter(e.target.value)}
            style={{ padding: '6px 10px', borderRadius: '6px', border: `1px solid ${subjectFilter !== '全部' ? C.primary : C.border}`, background: subjectFilter !== '全部' ? C.primaryLight : 'transparent', color: subjectFilter !== '全部' ? C.primary : C.textSec, fontSize: '13px', cursor: 'pointer', outline: 'none' }}>
            {SUBJECTS.map(s => <option key={s} value={s}>{s}</option>)}
          </select>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec, flexShrink: 0 }}>年级</span>
          <select value={gradeFilter} onChange={e => setGradeFilter(e.target.value)}
            style={{ padding: '6px 10px', borderRadius: '6px', border: `1px solid ${gradeFilter !== '全部' ? C.primary : C.border}`, background: gradeFilter !== '全部' ? C.primaryLight : 'transparent', color: gradeFilter !== '全部' ? C.primary : C.textSec, fontSize: '13px', cursor: 'pointer', outline: 'none' }}>
            {GRADES.map(g => <option key={g} value={g}>{g}</option>)}
          </select>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec, flexShrink: 0 }}>质量</span>
          <select value={qualityFilter} onChange={e => setQualityFilter(e.target.value)}
            style={{ padding: '6px 10px', borderRadius: '6px', border: `1px solid ${qualityFilter !== '全部' ? '#10B981' : C.border}`, background: qualityFilter !== '全部' ? 'rgba(16,185,129,0.08)' : 'transparent', color: qualityFilter !== '全部' ? '#10B981' : C.textSec, fontSize: '13px', cursor: 'pointer', outline: 'none' }}>
            {['全部', '5', '4', '3', '2'].map(v => <option key={v} value={v}>{v === '全部' ? '全部' : v === '5' ? '精品' : v === '4' ? '优秀' : v === '3' ? '良好' : '可用'}</option>)}
          </select>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec, flexShrink: 0 }}>教法</span>
          <select value={structFilter} onChange={e => setStructFilter(e.target.value)}
            style={{ padding: '6px 10px', borderRadius: '6px', border: `1px solid ${structFilter !== '全部' ? '#8B5CF6' : C.border}`, background: structFilter !== '全部' ? 'rgba(139,92,246,0.08)' : 'transparent', color: structFilter !== '全部' ? '#8B5CF6' : C.textSec, fontSize: '13px', cursor: 'pointer', outline: 'none' }}>
            {['全部', '1', '2', '3', '4', '5'].map(v => <option key={v} value={v}>{v === '全部' ? '全部' : v === '1' ? '讲授型' : v === '2' ? '探究型' : v === '3' ? '项目型' : v === '4' ? '翻转型' : '混合型'}</option>)}
          </select>
        </div>
        {isFiltered && (
          <button onClick={handleReset} style={{ padding: '6px 10px', borderRadius: '6px', border: 'none', background: 'transparent', fontSize: '12px', color: C.textMuted, cursor: 'pointer', textDecoration: 'underline' }}>清空筛选</button>
        )}
        {!loading && (
          <span style={{ fontSize: '13px', color: C.textMuted, marginLeft: 'auto' }}>
            {isFiltered ? `筛选后 ${plans.length} 份` : `共 ${total} 份`}
          </span>
        )}
      </div>

      {/* 错误 */}
      {error && (
        <div style={{ padding: '12px 16px', marginBottom: '16px', background: '#FEF2F2', border: '1px solid #FECACA', borderRadius: '8px', fontSize: '14px', color: C.danger, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span>⚠️ {error}</span>
          <button onClick={loadPlans} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.primary, fontSize: '13px' }}>重试</button>
        </div>
      )}

      {/* 无教研组提示 */}
      {scope === 'group' && !loading && myGroups.length === 0 && !error && (
        <div style={{ padding: '20px 24px', marginBottom: '16px', background: 'rgba(245,158,11,0.06)', border: '1px solid rgba(245,158,11,0.2)', borderRadius: '10px', fontSize: '14px', color: C.warning, display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '18px' }}>💡</span>
          <span>你还没有加入任何教研组，加入后可查看组内共享教案。</span>
        </div>
      )}

      {/* 卡片网格 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))', gap: '16px' }}>
        {loading && Array.from({ length: 6 }).map((_, i) => <SkeletonCard key={i} />)}
        {!loading && plans.map(plan => (
          <LibraryCard key={plan.id} plan={plan} currentUserId={user?.id || ''} forkingId={forkingId} onFork={handleFork} interactions={interactionsMap[plan.id]} onToggleInteraction={handleToggleInteraction} />
        ))}
        {!loading && !error && plans.length === 0 && (
          <EmptyState filtered={isFiltered} scope={scope} onReset={handleReset} />
        )}
      </div>

      {/* Toast */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)',
          padding: '12px 24px', borderRadius: '10px',
          background: toast.type === 'error' ? '#FEF2F2' : '#1F2937',
          color: toast.type === 'error' ? C.danger : '#fff',
          fontSize: '14px', fontWeight: 500, boxShadow: '0 8px 24px rgba(0,0,0,0.15)',
          zIndex: 9999, whiteSpace: 'nowrap',
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
