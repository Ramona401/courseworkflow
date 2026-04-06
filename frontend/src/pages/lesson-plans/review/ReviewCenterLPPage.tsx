/**
 * 教研组评审中心 — ReviewCenterLPPage
 *
 * 面向教研组长/骨干教师，提供人工评审工作台：
 *
 * Tab1：待评审队列
 *   - 列出本组submitted状态的教案
 *   - 点击"评审"展开评审表单（评分/意见/通过or退回）
 *   - 提交后教案状态流转：approved / revision
 *
 * Tab2：已评审记录
 *   - 已完成评审的教案历史
 *   - 显示评审结果和评分
 *
 * PRD §2.3 教研组长权限
 * PRD §7.1 状态路径二：共享沉淀（需要评审）
 * PRD §3.2 AI评审展示规范（人工评审参考同样原则）
 * v56修改：移除页面内重复标题（LPLayout header已有标题）
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getLessonPlans,
  getLessonPlan,
  reviewLessonPlan,
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
  purple:       '#8B5CF6',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
  borderHover:  '#E5E7EB',
}

/* ==================== Tab配置 ==================== */
type ReviewTab = 'pending' | 'done'

/* ==================== 评审决定配置 ==================== */
const DECISION_CONFIG = {
  approved: { label: '评审通过',  color: C.success, bg: 'rgba(16,185,129,0.08)', icon: '✅' },
  revision: { label: '退回修改',  color: C.warning, bg: 'rgba(249,115,22,0.08)', icon: '↩️' },
  rejected: { label: '不予通过',  color: C.danger,  bg: 'rgba(239,68,68,0.08)', icon: '❌' },
}

/* ==================== 评分维度（供参考）==================== */
const REVIEW_DIMENSIONS = [
  { code: 'T1', name: '目标清晰度', hint: '三维目标是否具体、可观察、可评估' },
  { code: 'T2', name: '结构完整性', hint: '环节是否齐全、时间分配是否合理' },
  { code: 'T3', name: '学生参与度', hint: '学生主动参与vs被动接收，讲授占比' },
  { code: 'T4', name: '评估对齐度', hint: '评估方式能否检验目标达成' },
  { code: 'T5', name: '可操作性',   hint: '活动步骤清晰、材料可获得' },
]

/* ==================== 子组件：骨架屏 ==================== */
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
          <div style={{ ...shimmer, width: '80px', height: '12px' }} />
        </div>
      </div>
      <div style={{ ...shimmer, width: '80px', height: '32px', borderRadius: '8px' }} />
    </div>
  )
}

/* ==================== 子组件：空状态 ==================== */
function EmptyState({ tab }: { tab: ReviewTab }) {
  return (
    <div style={{ textAlign: 'center', padding: '60px 40px', color: C.textMuted }}>
      <div style={{ fontSize: '40px', marginBottom: '12px' }}>{tab === 'pending' ? '🎉' : '📋'}</div>
      <div style={{ fontSize: '15px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>
        {tab === 'pending' ? '暂无待评审教案' : '暂无评审记录'}
      </div>
      <div style={{ fontSize: '13px', lineHeight: 1.7 }}>
        {tab === 'pending'
          ? '组内老师提交评审后，教案将出现在这里'
          : '完成评审的教案记录将显示在这里'}
      </div>
    </div>
  )
}

/* ==================== 子组件：评审表单（展开在卡片内）==================== */
interface ReviewFormProps {
  plan: LessonPlan
  onSubmit: (planId: string, decision: string, score: number, comments: string, suggestions: string[]) => Promise<void>
  onCancel: () => void
  submitting: boolean
}

function ReviewForm({ plan, onSubmit, onCancel, submitting }: ReviewFormProps) {
  const navigate = useNavigate()
  const [decision, setDecision] = useState<'approved' | 'revision' | 'rejected'>('approved')
  const [score, setScore]       = useState<number>(8)
  const [comments, setComments] = useState('')
  const [suggestion, setSuggestion] = useState('')
  const [suggestions, setSuggestions] = useState<string[]>([])

  /* 添加改进建议 */
  const addSuggestion = () => {
    if (!suggestion.trim()) return
    setSuggestions(prev => [...prev, suggestion.trim()])
    setSuggestion('')
  }

  /* 移除建议 */
  const removeSuggestion = (i: number) => {
    setSuggestions(prev => prev.filter((_, idx) => idx !== i))
  }

  const handleSubmit = () => {
    if (!comments.trim()) { alert('请填写评审意见'); return }
    onSubmit(plan.id, decision, score, comments, suggestions)
  }

  return (
    <div style={{
      margin: '0 -24px -20px',
      padding: '24px',
      background: C.bg,
      borderTop: `1px solid ${C.border}`,
    }}>
      {/* 查看教案内容入口 */}
      <div style={{ marginBottom: '20px', display: 'flex', justifyContent: 'flex-end' }}>
        <button
          onClick={() => navigate(`/lesson-plans/plans/${plan.id}`, { state: { from: '/lesson-plans/review' } })}
          style={{ padding: '6px 14px', borderRadius: '6px', border: `1px solid ${C.border}`, background: C.card, fontSize: '13px', color: C.primary, cursor: 'pointer' }}
        >
          📄 查看完整教案内容 →
        </button>
      </div>

      {/* 评审决定 */}
      <div style={{ marginBottom: '20px' }}>
        <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '10px' }}>评审决定</div>
        <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
          {(Object.entries(DECISION_CONFIG) as Array<[typeof decision, typeof DECISION_CONFIG[keyof typeof DECISION_CONFIG]]>).map(([key, cfg]) => (
            <button
              key={key}
              onClick={() => setDecision(key)}
              style={{
                padding: '10px 20px', borderRadius: '8px', cursor: 'pointer',
                border: `2px solid ${decision === key ? cfg.color : C.border}`,
                background: decision === key ? cfg.bg : 'transparent',
                fontSize: '14px', fontWeight: decision === key ? 700 : 400,
                color: decision === key ? cfg.color : C.textSec,
                transition: 'all 150ms ease',
                display: 'flex', alignItems: 'center', gap: '6px',
              }}
            >
              <span>{cfg.icon}</span><span>{cfg.label}</span>
            </button>
          ))}
        </div>
      </div>

      {/* 综合评分 */}
      <div style={{ marginBottom: '20px' }}>
        <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '10px' }}>
          综合评分
          <span style={{ marginLeft: '12px', fontSize: '24px', fontWeight: 700, color: score >= 8.5 ? C.success : score >= 7 ? C.accent : C.danger }}>{score.toFixed(1)}</span>
          <span style={{ marginLeft: '4px', fontSize: '12px', color: C.textMuted }}>/ 10</span>
        </div>
        <input
          type="range" min={1} max={10} step={0.5}
          value={score} onChange={e => setScore(parseFloat(e.target.value))}
          style={{ width: '100%', accentColor: C.primary, cursor: 'pointer' }}
        />
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', color: C.textMuted, marginTop: '4px' }}>
          <span>1 — 较差</span><span>5 — 一般</span><span>8 — 良好</span><span>10 — 优秀</span>
        </div>
      </div>

      {/* 参考维度 */}
      <div style={{ marginBottom: '20px' }}>
        <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '10px' }}>评审维度参考</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          {REVIEW_DIMENSIONS.map(dim => (
            <div key={dim.code} style={{ display: 'flex', gap: '10px', padding: '8px 12px', background: C.card, borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px' }}>
              <span style={{ fontWeight: 700, color: C.primary, flexShrink: 0, minWidth: '24px' }}>{dim.code}</span>
              <span style={{ fontWeight: 600, color: C.text, flexShrink: 0 }}>{dim.name}</span>
              <span style={{ color: C.textSec, flex: 1 }}>— {dim.hint}</span>
            </div>
          ))}
        </div>
      </div>

      {/* 评审意见（必填）*/}
      <div style={{ marginBottom: '20px' }}>
        <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
          评审意见 <span style={{ color: C.danger }}>*</span>
          <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px' }}>（请用对话式口吻写给老师看）</span>
        </div>
        <textarea
          value={comments}
          onChange={e => setComments(e.target.value)}
          placeholder="例如：整体来说这份教案目标清晰，导入环节设计得很有创意。建议在主体活动部分增加更多学生动手操作的机会..."
          rows={4}
          style={{
            width: '100%', padding: '10px 14px', borderRadius: '8px',
            border: `1px solid ${comments ? C.primary : C.border}`,
            fontSize: '14px', color: C.text, outline: 'none',
            boxSizing: 'border-box', resize: 'vertical', lineHeight: 1.7,
            fontFamily: 'inherit',
          }}
          onFocus={e => { e.target.style.borderColor = C.primary }}
          onBlur={e  => { e.target.style.borderColor = comments ? C.primary : C.border }}
        />
      </div>

      {/* 改进建议（选填，退回时建议填写）*/}
      <div style={{ marginBottom: '24px' }}>
        <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
          具体改进建议
          <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px' }}>（选填，退回时建议添加，帮助老师明确改进方向）</span>
        </div>

        {/* 已添加建议列表 */}
        {suggestions.length > 0 && (
          <div style={{ marginBottom: '10px', display: 'flex', flexDirection: 'column', gap: '6px' }}>
            {suggestions.map((s, i) => (
              <div key={i} style={{ display: 'flex', alignItems: 'flex-start', gap: '8px', padding: '8px 12px', background: 'rgba(245,158,11,0.06)', borderRadius: '8px', border: '1px solid rgba(245,158,11,0.2)' }}>
                <span style={{ fontSize: '13px', color: C.accent, flexShrink: 0 }}>💡</span>
                <span style={{ fontSize: '13px', color: C.text, flex: 1, lineHeight: 1.6 }}>{s}</span>
                <button onClick={() => removeSuggestion(i)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.textMuted, fontSize: '16px', lineHeight: 1, flexShrink: 0, padding: '0 4px' }}>×</button>
              </div>
            ))}
          </div>
        )}

        {/* 添加建议输入框 */}
        <div style={{ display: 'flex', gap: '8px' }}>
          <input
            type="text"
            value={suggestion}
            onChange={e => setSuggestion(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && addSuggestion()}
            placeholder="输入一条具体建议，回车添加..."
            style={{
              flex: 1, padding: '8px 12px', borderRadius: '8px',
              border: `1px solid ${C.border}`, fontSize: '13px', color: C.text,
              outline: 'none', fontFamily: 'inherit',
            }}
            onFocus={e => { e.target.style.borderColor = C.primary }}
            onBlur={e  => { e.target.style.borderColor = C.border }}
          />
          <button
            onClick={addSuggestion}
            disabled={!suggestion.trim()}
            style={{ padding: '8px 14px', borderRadius: '8px', border: 'none', background: suggestion.trim() ? C.primaryLight : C.border, color: suggestion.trim() ? C.primary : C.textMuted, fontSize: '13px', fontWeight: 600, cursor: suggestion.trim() ? 'pointer' : 'not-allowed' }}
          >+ 添加</button>
        </div>
      </div>

      {/* 操作按钮 */}
      <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
        <button
          onClick={onCancel}
          disabled={submitting}
          style={{ padding: '10px 20px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '14px', color: C.textSec, cursor: submitting ? 'not-allowed' : 'pointer' }}
        >取消</button>
        <button
          onClick={handleSubmit}
          disabled={submitting || !comments.trim()}
          style={{
            padding: '10px 24px', borderRadius: '8px', border: 'none',
            background: submitting || !comments.trim()
              ? C.border
              : DECISION_CONFIG[decision].color,
            color: submitting || !comments.trim() ? C.textMuted : '#fff',
            fontSize: '14px', fontWeight: 600,
            cursor: submitting || !comments.trim() ? 'not-allowed' : 'pointer',
            transition: 'all 150ms ease',
          }}
        >
          {submitting ? '提交中...' : `${DECISION_CONFIG[decision].icon} 提交评审`}
        </button>
      </div>
    </div>
  )
}

/* ==================== 子组件：待评审教案卡片 ==================== */
interface PendingCardProps {
  plan: LessonPlan
  expanded: boolean
  onExpand: () => void
  onCollapse: () => void
  onSubmitReview: (planId: string, decision: string, score: number, comments: string, suggestions: string[]) => Promise<void>
  submittingId: string | null
}

function PendingCard({ plan, expanded, onExpand, onCollapse, onSubmitReview, submittingId }: PendingCardProps) {
  const [hovered, setHovered] = useState(false)
  const isSubmitting = submittingId === plan.id

  const formatDate = (iso: string) => {
    try {
      const d = new Date(iso)
      return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`
    } catch { return iso }
  }

  return (
    <div style={{
      borderBottom: `1px solid ${C.border}`,
      transition: 'background 200ms ease',
      background: expanded ? C.bg : (hovered ? '#FAFBFC' : C.card),
    }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* 卡片主体 */}
      <div style={{ padding: '20px 24px' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px' }}>
          {/* 左侧信息 */}
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '8px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {plan.title}
            </div>
            <div style={{ display: 'flex', gap: '16px', flexWrap: 'wrap', fontSize: '13px', color: C.textSec, marginBottom: '8px' }}>
              <span>📚 {plan.subject}</span>
              <span>🎓 {plan.grade}</span>
              <span>⏱ {plan.duration_minutes}分钟</span>
              <span>📌 {plan.topic}</span>
            </div>
            <div style={{ display: 'flex', gap: '12px', alignItems: 'center', flexWrap: 'wrap' }}>
              {/* 作者 */}
              <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <div style={{ width: '20px', height: '20px', borderRadius: '50%', background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '10px', color: '#fff', fontWeight: 600 }}>
                  {(plan.author_name || '教').charAt(0)}
                </div>
                <span style={{ fontSize: '12px', color: C.textSec }}>{plan.author_name || '教师'}</span>
              </div>
              {/* AI评分 */}
              {plan.ai_review_score != null && (
                <span style={{ fontSize: '12px', padding: '2px 8px', borderRadius: '20px', background: plan.ai_review_score >= 8.5 ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)', color: plan.ai_review_score >= 8.5 ? C.success : C.accent, fontWeight: 600 }}>
                  🤖 AI评分 {plan.ai_review_score.toFixed(1)}
                </span>
              )}
              {/* 提交时间 */}
              <span style={{ fontSize: '12px', color: C.textMuted }}>提交于 {formatDate(plan.updated_at)}</span>
            </div>
          </div>

          {/* 右侧操作 */}
          <div style={{ flexShrink: 0, display: 'flex', gap: '8px', alignItems: 'center' }}>
            {!expanded ? (
              <button
                onClick={onExpand}
                style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer', transition: 'opacity 150ms ease' }}
                onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.opacity = '0.88' }}
                onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.opacity = '1' }}
              >
                开始评审 →
              </button>
            ) : (
              <button
                onClick={onCollapse}
                style={{ padding: '8px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '13px', color: C.textSec, cursor: 'pointer' }}
              >
                收起
              </button>
            )}
          </div>
        </div>
      </div>

      {/* 评审表单（展开时显示）*/}
      {expanded && (
        <div style={{ padding: '0 24px 20px' }}>
          <ReviewForm
            plan={plan}
            onSubmit={onSubmitReview}
            onCancel={onCollapse}
            submitting={isSubmitting}
          />
        </div>
      )}
    </div>
  )
}

/* ==================== 子组件：已评审记录卡片 ==================== */
function DoneCard({ plan }: { plan: LessonPlan }) {
  const navigate = useNavigate()
  const [hovered, setHovered] = useState(false)
  const cfg = DECISION_CONFIG[plan.status === 'approved' ? 'approved' : plan.status === 'revision' ? 'revision' : 'rejected']

  const formatDate = (iso: string) => {
    try {
      const d = new Date(iso)
      return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`
    } catch { return iso }
  }

  return (
    <div
      style={{
        padding: '20px 24px', borderBottom: `1px solid ${C.border}`,
        background: hovered ? C.bg : C.card,
        transition: 'background 200ms ease', cursor: 'pointer',
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onClick={() => navigate(`/lesson-plans/plans/${plan.id}`, { state: { from: '/lesson-plans/review' } })}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px' }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '6px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {plan.title}
          </div>
          <div style={{ display: 'flex', gap: '12px', fontSize: '13px', color: C.textSec, flexWrap: 'wrap' }}>
            <span>📚 {plan.subject}</span>
            <span>🎓 {plan.grade}</span>
            <span>{plan.author_name || '教师'} 提交</span>
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
  const navigate  = useNavigate()

  /* ===== Tab状态 ===== */
  const [activeTab, setActiveTab] = useState<ReviewTab>('pending')

  /* ===== 数据状态 ===== */
  const [pendingPlans, setPendingPlans] = useState<LessonPlan[]>([])
  const [donePlans, setDonePlans]       = useState<LessonPlan[]>([])
  const [loading, setLoading]           = useState(true)
  const [error, setError]               = useState<string | null>(null)

  /* ===== 用户教研组 ===== */
  const [myGroups, setMyGroups] = useState<TeachingGroup[]>([])

  /* ===== 展开的评审卡片ID ===== */
  const [expandedId, setExpandedId] = useState<string | null>(null)

  /* ===== 正在提交评审的planId ===== */
  const [submittingId, setSubmittingId] = useState<string | null>(null)

  /* ===== Toast ===== */
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type }); setTimeout(() => setToast(null), 3000)
  }

  /* ===== 加载教研组 ===== */
  useEffect(() => {
    getMyGroups().then(g => setMyGroups(g || [])).catch(() => setMyGroups([]))
  }, [])

  /* ===== 加载教案列表 ===== */
  const loadPlans = useCallback(async () => {
    if (!user) return
    setLoading(true); setError(null)
    try {
      const params: Record<string, string | number> = { limit: 100 }
      /* 有教研组时按组过滤 */
      if (myGroups.length > 0) params.group_id = myGroups[0].id

      /* 并发拉取：待评审（submitted）+ 已评审（approved/revision）*/
      const [submittedResp, approvedResp, revisionResp] = await Promise.all([
        getLessonPlans({ ...params, status: 'submitted' }),
        getLessonPlans({ ...params, status: 'approved' }),
        getLessonPlans({ ...params, status: 'revision' }),
      ])

      setPendingPlans(submittedResp.lesson_plans || [])

      /* 合并已评审（approved + revision），按更新时间降序 */
      const done = [
        ...(approvedResp.lesson_plans || []),
        ...(revisionResp.lesson_plans || []),
      ].sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
      setDonePlans(done)
    } catch (e) {
      console.error('加载评审列表失败:', e); setError('加载失败，请稍后重试')
    } finally { setLoading(false) }
  }, [user, myGroups])

  useEffect(() => { loadPlans() }, [loadPlans])

  /* ===== 提交评审 ===== */
  const handleSubmitReview = async (
    planId: string,
    decision: string,
    score: number,
    comments: string,
    suggestions: string[],
  ) => {
    if (submittingId) return
    setSubmittingId(planId)
    try {
      await reviewLessonPlan(planId, { decision, score, comments, suggestions })
      const decisionLabel = DECISION_CONFIG[decision as keyof typeof DECISION_CONFIG]?.label || decision
      showToast(`评审完成：${decisionLabel} ✓`)
      setExpandedId(null)
      /* 刷新列表 */
      await loadPlans()
    } catch (e) {
      console.error('提交评审失败:', e)
      showToast('提交失败，请稍后重试', 'error')
    } finally { setSubmittingId(null) }
  }

  /* ==================== 渲染 ==================== */
  return (
    <div>
      {/* 描述 + 无教研组提示（标题已在LPLayout header中显示，此处不再重复） */}
      <div style={{ marginBottom: '24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <p style={{ fontSize: '14px', color: C.textSec, margin: 0 }}>
          审核组内教师提交的教案，通过评审的教案可共享到教研组库
        </p>
        {/* 无教研组提示 */}
        {!loading && myGroups.length === 0 && (
          <div style={{ padding: '8px 14px', background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.2)', borderRadius: '8px', fontSize: '13px', color: C.warning, flexShrink: 0 }}>
            💡 你尚未加入教研组，暂时看不到待评审教案
          </div>
        )}
      </div>

      {/* 统计卡片 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: '12px', marginBottom: '24px' }}>
        {[
          { label: '待评审', value: pendingPlans.length, color: C.accent,   bg: 'rgba(245,158,11,0.08)',   icon: '📋' },
          { label: '已通过', value: donePlans.filter(p => p.status === 'approved').length, color: C.success, bg: 'rgba(16,185,129,0.08)', icon: '✅' },
          { label: '已退回', value: donePlans.filter(p => p.status === 'revision').length, color: C.warning, bg: 'rgba(249,115,22,0.08)', icon: '↩️' },
          { label: '合计',   value: pendingPlans.length + donePlans.length, color: C.primary,  bg: C.primaryLight, icon: '📊' },
        ].map(stat => (
          <div key={stat.label} style={{ padding: '16px 20px', background: stat.bg, borderRadius: '10px', border: `1px solid ${stat.color}20` }}>
            <div style={{ fontSize: '20px', marginBottom: '4px' }}>{stat.icon}</div>
            <div style={{ fontSize: '24px', fontWeight: 700, color: stat.color, lineHeight: 1 }}>{loading ? '—' : stat.value}</div>
            <div style={{ fontSize: '12px', color: C.textSec, marginTop: '4px' }}>{stat.label}</div>
          </div>
        ))}
      </div>

      {/* Tab栏 */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
        <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, padding: '0 4px' }}>
          {([
            { key: 'pending' as ReviewTab, label: `📋 待评审`, count: pendingPlans.length },
            { key: 'done'    as ReviewTab, label: `✅ 已评审`, count: donePlans.length    },
          ]).map(tab => {
            const isActive = activeTab === tab.key
            return (
              <button
                key={tab.key}
                onClick={() => { setActiveTab(tab.key); setExpandedId(null) }}
                style={{
                  padding: '14px 20px', border: 'none', background: 'transparent',
                  fontSize: '14px', fontWeight: isActive ? 600 : 400,
                  color: isActive ? C.primary : C.textSec, cursor: 'pointer',
                  borderBottom: isActive ? `2px solid ${C.primary}` : '2px solid transparent',
                  marginBottom: '-1px', transition: 'all 150ms ease',
                  display: 'flex', alignItems: 'center', gap: '8px',
                }}
              >
                {tab.label}
                {!loading && tab.count > 0 && (
                  <span style={{ padding: '1px 7px', borderRadius: '10px', background: isActive ? C.primary : C.border, color: isActive ? '#fff' : C.textMuted, fontSize: '11px', fontWeight: 700 }}>
                    {tab.count}
                  </span>
                )}
              </button>
            )
          })}
        </div>

        {/* 错误提示 */}
        {error && (
          <div style={{ padding: '16px 24px', background: '#FEF2F2', fontSize: '14px', color: C.danger, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>⚠️ {error}</span>
            <button onClick={loadPlans} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.primary, fontSize: '13px' }}>重试</button>
          </div>
        )}

        {/* 待评审Tab内容 */}
        {activeTab === 'pending' && (
          <div>
            {loading && [1,2,3].map(i => <SkeletonRow key={i} />)}
            {!loading && pendingPlans.length === 0 && <EmptyState tab="pending" />}
            {!loading && pendingPlans.map(plan => (
              <PendingCard
                key={plan.id}
                plan={plan}
                expanded={expandedId === plan.id}
                onExpand={() => setExpandedId(plan.id)}
                onCollapse={() => setExpandedId(null)}
                onSubmitReview={handleSubmitReview}
                submittingId={submittingId}
              />
            ))}
          </div>
        )}

        {/* 已评审Tab内容 */}
        {activeTab === 'done' && (
          <div>
            {loading && [1,2,3].map(i => <SkeletonRow key={i} />)}
            {!loading && donePlans.length === 0 && <EmptyState tab="done" />}
            {!loading && donePlans.map(plan => (
              <DoneCard key={plan.id} plan={plan} />
            ))}
          </div>
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
