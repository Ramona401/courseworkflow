/**
 * 教案详情页 — PlanDetailPage
 *
 * 功能：
 *   1. 教案基本信息头部（标题/学科/年级/课时/状态/作者）
 *   2. 四Tab内容区：
 *      - 📄 教案内容（Markdown渲染）
 *      - 🤖 AI评审（评分/做得好的/可以更好/各维度详情）
 *      - 📊 使用统计（查看数/版本/Fork来源）
 *      - 🔗 关联课件（Phase6：接入真实Pipeline数据）
 *   3. 操作栏：发布/提交评审/进入课件开发/Fork/删除（按状态动态渲染）
 *   4. 返回按钮（智能返回：从库来→库，从我的来→我的）
 *
 * PRD §7.3 教案详情页 | Phase6：课件衔接
 */
import { useState, useEffect } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getLessonPlan,
  publishLessonPlanPersonal,
  submitLessonPlanForReview,
  startDevelopment,
  deleteLessonPlan,
  forkLessonPlan,
  type LessonPlan,
  type LessonPlanStatus,
  type AIReviewResult,
  type StartDevelopmentResult,
} from '@/api/lesson-plans'
import { getPipelineDetail, type PipelineDetail } from '@/api/pipelines'

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
  aiBubble:     '#EEF4FF',
}

/* ==================== 状态配置 ==================== */
interface StatusConfig { label: string; color: string; bg: string; dot: string }

const STATUS_CONFIG: Record<LessonPlanStatus, StatusConfig> = {
  draft:             { label: '草稿',      color: C.textSec,  bg: '#F3F4F6',                dot: C.textMuted },
  published_personal:{ label: '已发布',    color: C.primary,  bg: C.primaryLight,           dot: C.primary   },
  submitted:         { label: '待评审',    color: C.accent,   bg: 'rgba(245,158,11,0.08)',  dot: C.accent    },
  revision:          { label: '退回修改',  color: C.warning,  bg: 'rgba(249,115,22,0.08)',  dot: C.warning   },
  approved:          { label: '评审通过',  color: C.success,  bg: 'rgba(16,185,129,0.08)', dot: C.success   },
  published_shared:  { label: '已共享',    color: C.purple,   bg: 'rgba(139,92,246,0.08)', dot: C.purple    },
  developing:        { label: '课件开发中',color: '#0EA5E9',  bg: 'rgba(14,165,233,0.08)', dot: '#0EA5E9'   },
  completed:         { label: '已完成',    color: C.success,  bg: 'rgba(16,185,129,0.08)', dot: C.success   },
}

/* Pipeline状态中文映射 */
const PIPELINE_STATUS_LABEL: Record<string, { label: string; color: string; bg: string }> = {
  pending:          { label: '待启动',    color: C.textSec,  bg: '#F3F4F6' },
  running:          { label: '执行中',    color: C.primary,  bg: C.primaryLight },
  review_queue:     { label: '待人工审核', color: C.accent,  bg: 'rgba(245,158,11,0.08)' },
  pending_finalize: { label: '待确认定稿', color: C.warning, bg: 'rgba(249,115,22,0.08)' },
  finalized:        { label: '已定稿',    color: C.success,  bg: 'rgba(16,185,129,0.08)' },
  needs_human:      { label: '需人工介入', color: C.warning, bg: 'rgba(249,115,22,0.08)' },
  failed:           { label: '执行失败',  color: C.danger,   bg: 'rgba(239,68,68,0.08)'  },
  cancelled:        { label: '已取消',    color: C.textSec,  bg: '#F3F4F6' },
  verified:         { label: '验收通过',  color: C.success,  bg: 'rgba(16,185,129,0.08)' },
  verify_failed:    { label: '验收未通过', color: C.danger,  bg: 'rgba(239,68,68,0.08)'  },
}

/* ==================== Tab配置 ==================== */
type TabKey = 'content' | 'review' | 'stats' | 'courseware'
interface TabConfig { key: TabKey; label: string }
const TABS: TabConfig[] = [
  { key: 'content',    label: '📄 教案内容' },
  { key: 'review',     label: '🤖 AI评审'   },
  { key: 'stats',      label: '📊 使用统计' },
  { key: 'courseware', label: '🔗 关联课件' },
]

/* ==================== 轻量 Markdown 渲染器 ==================== */
function renderMarkdown(text: string): React.ReactNode {
  if (!text) return null
  const lines = text.split('\n')
  const nodes: React.ReactNode[] = []
  let listItems: React.ReactNode[] = []
  let listType: 'ul' | 'ol' | null = null
  let key = 0

  const parseInline = (line: string): React.ReactNode => {
    const parts = line.split(/(\*\*[^*]+\*\*)/)
    if (parts.length === 1) return line
    return <>{parts.map((p, i) => p.startsWith('**') && p.endsWith('**')
      ? <strong key={i} style={{ fontWeight: 700, color: C.text }}>{p.slice(2, -2)}</strong>
      : p)}</>
  }

  const flushList = () => {
    if (!listItems.length) return
    nodes.push(listType === 'ul'
      ? <ul key={key++} style={{ margin: '6px 0 6px 16px', padding: 0, listStyle: 'disc' }}>{listItems}</ul>
      : <ol key={key++} style={{ margin: '6px 0 6px 16px', padding: 0, listStyle: 'decimal' }}>{listItems}</ol>)
    listItems = []; listType = null
  }

  for (const line of lines) {
    const t = line.trim()
    if (!t) { flushList(); continue }
    if (/^---+$/.test(t)) { flushList(); nodes.push(<hr key={key++} style={{ border: 'none', borderTop: `1px solid ${C.border}`, margin: '10px 0' }} />); continue }
    const h3 = t.match(/^###\s+(.+)/); if (h3) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '14px', fontWeight: 700, color: C.text, margin: '10px 0 4px' }}>{parseInline(h3[1])}</div>); continue }
    const h2 = t.match(/^##\s+(.+)/);  if (h2) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '15px', fontWeight: 700, color: C.text, margin: '12px 0 4px' }}>{parseInline(h2[1])}</div>); continue }
    const h1 = t.match(/^#\s+(.+)/);   if (h1) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '16px', fontWeight: 700, color: C.text, margin: '14px 0 6px' }}>{parseInline(h1[1])}</div>); continue }
    const ul = t.match(/^[-*]\s+(.+)/); if (ul) { if (listType !== 'ul') { flushList(); listType = 'ul' }; listItems.push(<li key={key++} style={{ fontSize: '14px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(ul[1])}</li>); continue }
    const ol = t.match(/^\d+\.\s+(.+)/); if (ol) { if (listType !== 'ol') { flushList(); listType = 'ol' }; listItems.push(<li key={key++} style={{ fontSize: '14px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(ol[1])}</li>); continue }
    flushList()
    nodes.push(<div key={key++} style={{ fontSize: '15px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(t)}</div>)
  }
  flushList()
  return <>{nodes}</>
}

/* ==================== 子组件：状态徽标 ==================== */
function StatusBadge({ status }: { status: LessonPlanStatus }) {
  const cfg = STATUS_CONFIG[status] || STATUS_CONFIG.draft
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: '5px',
      padding: '4px 10px', borderRadius: '20px',
      background: cfg.bg, fontSize: '13px', fontWeight: 500, color: cfg.color,
    }}>
      <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: cfg.dot }} />
      {cfg.label}
    </span>
  )
}

/* ==================== 子组件：元信息标签 ==================== */
function MetaTag({ icon, label, value }: { icon: string; label: string; value: string }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
      <span style={{ fontSize: '11px', color: C.textMuted, fontWeight: 500 }}>{label}</span>
      <span style={{ fontSize: '14px', color: C.text, display: 'flex', alignItems: 'center', gap: '4px' }}>
        <span>{icon}</span><span>{value}</span>
      </span>
    </div>
  )
}

/* ==================== 子组件：骨架屏 ==================== */
function DetailSkeleton() {
  const shimmer: React.CSSProperties = {
    background: 'linear-gradient(90deg, #F3F4F6 25%, #E5E7EB 50%, #F3F4F6 75%)',
    backgroundSize: '200% 100%', animation: 'shimmer 1.4s infinite', borderRadius: '4px',
  }
  return (
    <div>
      <style>{`@keyframes shimmer { 0%{background-position:200% 0} 100%{background-position:-200% 0} }`}</style>
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '28px', marginBottom: '20px' }}>
        <div style={{ ...shimmer, width: '60%', height: '28px', marginBottom: '16px' }} />
        <div style={{ display: 'flex', gap: '24px', marginBottom: '16px' }}>
          {[1,2,3,4].map(i => <div key={i} style={{ ...shimmer, width: '80px', height: '36px' }} />)}
        </div>
        <div style={{ display: 'flex', gap: '8px' }}>
          {[1,2].map(i => <div key={i} style={{ ...shimmer, width: '100px', height: '34px', borderRadius: '8px' }} />)}
        </div>
      </div>
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '28px' }}>
        {[1,2,3,4,5].map(i => <div key={i} style={{ ...shimmer, width: i%2===0?'80%':'100%', height: '14px', marginBottom: '10px' }} />)}
      </div>
    </div>
  )
}

/* ==================== 子组件：教案内容Tab ==================== */
function ContentTab({ plan }: { plan: LessonPlan }) {
  const hasContent = plan.content_markdown && plan.content_markdown.trim().length > 0
  return (
    <div style={{ padding: '28px' }}>
      {hasContent ? (
        <div style={{ fontSize: '14px', lineHeight: 1.9 }}>{renderMarkdown(plan.content_markdown!)}</div>
      ) : (
        <div style={{ textAlign: 'center', padding: '60px 40px', color: C.textMuted }}>
          <div style={{ fontSize: '40px', marginBottom: '12px' }}>📄</div>
          <div style={{ fontSize: '15px', fontWeight: 500, color: C.textSec, marginBottom: '6px' }}>暂无教案内容</div>
          <div style={{ fontSize: '13px', lineHeight: 1.7 }}>前往备课工坊与AI一起生成教案内容</div>
        </div>
      )}
    </div>
  )
}

/* ==================== 子组件：AI评审Tab ==================== */
function ReviewTab({ plan }: { plan: LessonPlan }) {
  let review: AIReviewResult | null = null
  if (plan.ai_review_result) {
    try {
      review = typeof plan.ai_review_result === 'string'
        ? JSON.parse(plan.ai_review_result) : plan.ai_review_result as AIReviewResult
      if (!review || !review.total_score) review = null
    } catch { review = null }
  }

  if (!review) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted }}>
        <div style={{ fontSize: '40px', marginBottom: '12px' }}>🤖</div>
        <div style={{ fontSize: '15px', fontWeight: 500, color: C.textSec, marginBottom: '6px' }}>尚未进行AI评审</div>
        <div style={{ fontSize: '13px', lineHeight: 1.7 }}>在备课工坊生成教案后，点击"AI评审"可获取质量分析</div>
      </div>
    )
  }

  return (
    <div style={{ padding: '28px' }}>
      {/* 总分卡片 */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: '20px',
        padding: '20px 24px', borderRadius: '12px', marginBottom: '24px',
        background: review.total_score >= 8.5 ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)',
        border: `1px solid ${review.total_score >= 8.5 ? '#10B98130' : '#F59E0B30'}`,
      }}>
        <div style={{ fontSize: '48px', fontWeight: 700, flexShrink: 0, lineHeight: 1,
          color: review.total_score >= 8.5 ? C.success : C.accent }}>
          {review.total_score.toFixed(1)}
        </div>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
            AI综合评分
            <span style={{ marginLeft: '10px', fontSize: '12px', fontWeight: 400, color: C.textMuted }}>
              {review.reviewed_at ? `评审于 ${review.reviewed_at.slice(0, 10)}` : ''}
            </span>
          </div>
          <div style={{ fontSize: '14px', color: C.textSec, lineHeight: 1.7 }}>{review.summary}</div>
        </div>
      </div>

      {/* 做得好的 + 可以更好 */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '24px' }}>
        {review.good_points?.length > 0 && (
          <div>
            <div style={{ fontSize: '14px', fontWeight: 600, color: C.success, marginBottom: '10px' }}>✅ 做得好的</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              {review.good_points.map((p, i) => (
                <div key={i} style={{ padding: '10px 12px', borderRadius: '8px',
                  background: 'rgba(16,185,129,0.06)', border: '1px solid rgba(16,185,129,0.12)',
                  fontSize: '13px', color: C.text, lineHeight: 1.6 }}>{p}</div>
              ))}
            </div>
          </div>
        )}
        {review.improvements?.length > 0 && (
          <div>
            <div style={{ fontSize: '14px', fontWeight: 600, color: C.accent, marginBottom: '10px' }}>💡 可以更好</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {review.improvements.map(imp => (
                <div key={imp.id} style={{ padding: '10px 12px', borderRadius: '8px',
                  background: 'rgba(245,158,11,0.06)', border: '1px solid rgba(245,158,11,0.15)' }}>
                  <div style={{ fontSize: '13px', fontWeight: 500, color: C.text, marginBottom: '4px' }}>{imp.issue}</div>
                  <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.6 }}>{imp.suggestion}</div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* 各维度 */}
      {review.dimensions?.length > 0 && (
        <div>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '12px' }}>📊 各维度详情</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {review.dimensions.map(dim => (
              <div key={dim.code} style={{ display: 'flex', alignItems: 'flex-start', gap: '16px',
                padding: '12px 16px', borderRadius: '8px', background: C.bg, border: `1px solid ${C.border}` }}>
                <div style={{ flexShrink: 0, textAlign: 'center', minWidth: '52px' }}>
                  <div style={{ fontSize: '12px', fontWeight: 700, color: C.textMuted }}>{dim.code}</div>
                  <div style={{ fontSize: '22px', fontWeight: 700, lineHeight: 1.2,
                    color: dim.score >= 8 ? C.success : dim.score >= 6 ? C.accent : C.danger }}>{dim.score}</div>
                </div>
                <div style={{ flex: 1, paddingTop: '4px' }}>
                  <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                    {dim.name}{dim.good && <span style={{ marginLeft: '6px', fontSize: '11px', color: C.success }}>✓ 优秀</span>}
                  </div>
                  <div style={{ height: '4px', background: C.border, borderRadius: '2px', overflow: 'hidden', marginBottom: '6px' }}>
                    <div style={{ height: '100%', borderRadius: '2px', width: `${dim.score * 10}%`,
                      background: dim.score >= 8 ? C.success : dim.score >= 6 ? C.accent : C.danger }} />
                  </div>
                  <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.6 }}>{dim.comment}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

/* ==================== 子组件：使用统计Tab ==================== */
function StatsTab({ plan }: { plan: LessonPlan }) {
  const fmt = (iso: string) => {
    try {
      const d = new Date(iso)
      return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
    } catch { return iso }
  }
  const statItems = [
    { icon: '👁', label: '浏览次数', value: `${(plan as any).view_count ?? 0} 次` },
    { icon: '📋', label: '使用次数', value: `${(plan as any).use_count ?? 0} 次`  },
    { icon: '🔀', label: '版本号',   value: `v${plan.version}`                   },
    { icon: '📅', label: '创建时间', value: fmt(plan.created_at)                  },
    { icon: '🔄', label: '最后更新', value: fmt(plan.updated_at)                  },
  ]
  return (
    <div style={{ padding: '28px' }}>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '16px', marginBottom: '28px' }}>
        {statItems.map(item => (
          <div key={item.label} style={{ padding: '20px', borderRadius: '10px', background: C.bg, border: `1px solid ${C.border}`, display: 'flex', flexDirection: 'column', gap: '6px' }}>
            <span style={{ fontSize: '22px' }}>{item.icon}</span>
            <span style={{ fontSize: '12px', color: C.textMuted, fontWeight: 500 }}>{item.label}</span>
            <span style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>{item.value}</span>
          </div>
        ))}
      </div>
      {(plan as any).forked_from && (
        <div style={{ padding: '16px 20px', borderRadius: '10px', background: 'rgba(139,92,246,0.06)', border: '1px solid rgba(139,92,246,0.15)', display: 'flex', alignItems: 'center', gap: '12px' }}>
          <span style={{ fontSize: '20px' }}>🔀</span>
          <div>
            <div style={{ fontSize: '13px', fontWeight: 600, color: C.purple, marginBottom: '2px' }}>Fork自其他教案</div>
            <div style={{ fontSize: '12px', color: C.textSec }}>原始教案ID：{(plan as any).forked_from}</div>
          </div>
        </div>
      )}
      {plan.ai_review_score != null && (
        <div style={{ marginTop: '24px' }}>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '12px' }}>🤖 AI评审记录</div>
          <div style={{ padding: '16px 20px', borderRadius: '10px', background: C.bg, border: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', gap: '16px' }}>
            <div style={{ fontSize: '32px', fontWeight: 700, color: plan.ai_review_score >= 8.5 ? C.success : C.accent }}>
              {plan.ai_review_score.toFixed(1)}
            </div>
            <div>
              <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>最新AI评分</div>
              <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px' }}>
                {plan.ai_review_score >= 8.5 ? '✅ 达到共享推荐标准（≥8.5）' : '💡 继续优化可提升至推荐标准'}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

/* ==================== 子组件：关联课件Tab（Phase6完整实现）==================== */
interface CoursewareTabProps {
  plan: LessonPlan
  onNavigatePipeline: (pipelineId?: string) => void
}

function CoursewareTab({ plan, onNavigatePipeline }: CoursewareTabProps) {
  const [pipeline, setPipeline]     = useState<PipelineDetail | null>(null)
  const [pipelineLoading, setPipelineLoading] = useState(false)
  const [pipelineError, setPipelineError]     = useState(false)

  const linkedId = plan.linked_pipeline_id

  // 有关联Pipeline ID时，加载Pipeline详情
  useEffect(() => {
    if (!linkedId) return
    setPipelineLoading(true)
    setPipelineError(false)
    getPipelineDetail(linkedId)
      .then(data => { setPipeline(data); setPipelineLoading(false) })
      .catch(() => { setPipelineError(true); setPipelineLoading(false) })
  }, [linkedId])

  // 未进入课件开发
  if (!['developing', 'completed'].includes(plan.status) && !linkedId) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted }}>
        <div style={{ fontSize: '40px', marginBottom: '12px' }}>🖥️</div>
        <div style={{ fontSize: '15px', fontWeight: 500, color: C.textSec, marginBottom: '6px' }}>尚未进入课件开发</div>
        <div style={{ fontSize: '13px', lineHeight: 1.7, marginBottom: '20px' }}>
          教案发布后，可进入课件开发流程，系统将自动创建课件开发任务
        </div>
        {['published_personal', 'approved', 'published_shared'].includes(plan.status) && (
          <div style={{ display: 'inline-flex', alignItems: 'center', gap: '6px',
            padding: '8px 14px', borderRadius: '8px', background: C.primaryLight,
            border: `1px solid ${C.primary}30`, fontSize: '13px', color: C.primary }}>
            💡 点击上方"进入课件开发"按钮开始
          </div>
        )}
      </div>
    )
  }

  // 加载中
  if (pipelineLoading) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted, fontSize: '14px' }}>
        加载课件开发信息...
      </div>
    )
  }

  // 加载失败
  if (pipelineError || (!pipeline && linkedId)) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted }}>
        <div style={{ fontSize: '32px', marginBottom: '12px' }}>⚠️</div>
        <div style={{ fontSize: '14px', color: C.textSec, marginBottom: '16px' }}>课件开发信息加载失败</div>
        <button onClick={() => onNavigatePipeline(linkedId!)}
          style={{ padding: '8px 20px', borderRadius: '8px', border: 'none',
            background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
          前往课件审核系统查看
        </button>
      </div>
    )
  }

  // 有Pipeline数据
  const pStatus = pipeline?.status || 'pending'
  const pStatusCfg = PIPELINE_STATUS_LABEL[pStatus] || PIPELINE_STATUS_LABEL['pending']

  // 8步进度计算
  const stepOrder = ['dbCheck','scanner','evaluator','meta','translator','generator','review','verify']
  const stepNameMap: Record<string, string> = {
    dbCheck:'数据检查', scanner:'课程扫描', evaluator:'质量评估',
    meta:'元评估', translator:'方案翻译', generator:'页面生成',
    review:'人工审核', verify:'验收',
  }
  const steps = pipeline?.steps || []
  const doneCount = steps.filter(s => s.status === 'done').length
  const totalSteps = 8
  const progressPct = Math.round((doneCount / totalSteps) * 100)

  return (
    <div style={{ padding: '28px' }}>
      {/* 状态总览卡片 */}
      <div style={{
        padding: '20px 24px', borderRadius: '12px', marginBottom: '20px',
        background: 'rgba(14,165,233,0.06)', border: '1px solid rgba(14,165,233,0.2)',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '16px',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <div style={{ fontSize: '32px' }}>🖥️</div>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '6px' }}>
              <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>课件开发任务</div>
              <span style={{
                fontSize: '11px', fontWeight: 600, padding: '2px 8px', borderRadius: '6px',
                color: pStatusCfg.color, background: pStatusCfg.bg,
              }}>{pStatusCfg.label}</span>
            </div>
            {pipeline && (
              <div style={{ fontSize: '13px', color: C.textSec }}>
                课程：{pipeline.course_name || pipeline.course_code}
              </div>
            )}
          </div>
        </div>
        <button
          onClick={() => onNavigatePipeline(linkedId!)}
          style={{
            padding: '9px 18px', borderRadius: '8px', border: 'none',
            background: '#0EA5E9', color: '#fff',
            fontSize: '13px', fontWeight: 600, cursor: 'pointer', flexShrink: 0,
          }}
        >
          查看课件进度 →
        </button>
      </div>

      {/* 进度条 */}
      {pipeline && (
        <div style={{ marginBottom: '20px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
            <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>执行进度</span>
            <span style={{ fontSize: '13px', color: C.textSec }}>{doneCount}/{totalSteps} 步完成 ({progressPct}%)</span>
          </div>
          <div style={{ height: '6px', background: C.border, borderRadius: '3px', overflow: 'hidden' }}>
            <div style={{
              height: '100%', borderRadius: '3px',
              width: `${progressPct}%`,
              background: pStatus === 'verified' ? C.success : pStatus === 'failed' ? C.danger : C.primary,
              transition: 'width 600ms ease',
            }} />
          </div>
        </div>
      )}

      {/* 8步进度列表 */}
      {pipeline && steps.length > 0 && (
        <div>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '10px' }}>步骤详情</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            {stepOrder.map((stepKey, idx) => {
              const step = steps.find(s => s.step_name === stepKey)
              const status = step?.status || 'pending'
              const icon = status === 'done' ? '✅' : status === 'running' ? '⏳' : status === 'failed' ? '❌' : status === 'skipped' ? '⏭' : '⬜'
              const color = status === 'done' ? C.success : status === 'running' ? C.primary : status === 'failed' ? C.danger : C.textMuted
              return (
                <div key={stepKey} style={{
                  display: 'flex', alignItems: 'center', gap: '10px',
                  padding: '8px 12px', borderRadius: '8px',
                  background: status === 'running' ? C.primaryLight : C.bg,
                  border: `1px solid ${status === 'running' ? C.primary+'30' : C.border}`,
                }}>
                  <span style={{ fontSize: '14px', flexShrink: 0 }}>{icon}</span>
                  <span style={{ fontSize: '12px', color: C.textMuted, flexShrink: 0 }}>
                    {String(idx + 1).padStart(2, '0')}
                  </span>
                  <span style={{ fontSize: '13px', fontWeight: status === 'running' ? 600 : 400, color, flex: 1 }}>
                    {stepNameMap[stepKey] || stepKey}
                  </span>
                  {step?.duration_ms && step.duration_ms > 0 && (
                    <span style={{ fontSize: '11px', color: C.textMuted }}>
                      {(step.duration_ms / 1000).toFixed(1)}s
                    </span>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* 无步骤数据时的简要提示 */}
      {!pipeline && linkedId && (
        <div style={{ textAlign: 'center', padding: '20px', color: C.textMuted, fontSize: '13px' }}>
          课件开发任务已创建，请前往课件审核系统查看详情
        </div>
      )}
    </div>
  )
}

/* ==================== 子组件：操作按钮组 ==================== */
interface ActionBarProps {
  plan: LessonPlan
  isOwner: boolean
  actionLoading: string | null
  onAction: (action: string) => void
}

function ActionBar({ plan, isOwner, actionLoading, onAction }: ActionBarProps) {
  const navigate = useNavigate()
  const isLoading = !!actionLoading

  const primaryBtn: React.CSSProperties = {
    padding: '9px 20px', borderRadius: '8px', border: 'none',
    background: isLoading ? '#E5E7EB' : C.primary, color: isLoading ? C.textMuted : '#fff',
    fontSize: '14px', fontWeight: 600, cursor: isLoading ? 'not-allowed' : 'pointer',
    transition: 'all 150ms ease', whiteSpace: 'nowrap',
  }
  const secondaryBtn: React.CSSProperties = {
    padding: '9px 20px', borderRadius: '8px', border: `1px solid ${C.border}`,
    background: 'transparent', fontSize: '14px', color: C.textSec,
    cursor: isLoading ? 'not-allowed' : 'pointer', transition: 'all 150ms ease', whiteSpace: 'nowrap',
  }
  const dangerBtn: React.CSSProperties = {
    padding: '9px 20px', borderRadius: '8px', border: '1px solid #FEE2E2',
    background: 'transparent', fontSize: '14px', color: C.danger,
    cursor: isLoading ? 'not-allowed' : 'pointer', transition: 'all 150ms ease', whiteSpace: 'nowrap',
  }

  const buttons: Array<{ label: string; style: React.CSSProperties; action: string; confirm?: string }> = []

  if (isOwner) {
    switch (plan.status) {
      case 'draft':
        buttons.push({ label: '发布教案', style: primaryBtn, action: 'publish' })
        buttons.push({ label: '提交评审', style: secondaryBtn, action: 'submit' })
        break
      case 'published_personal':
        buttons.push({ label: '进入课件开发', style: primaryBtn, action: 'develop' })
        buttons.push({ label: '提交评审', style: secondaryBtn, action: 'submit' })
        break
      case 'revision':
        buttons.push({ label: '修改后重提', style: primaryBtn, action: 'submit' })
        break
      case 'approved':
      case 'published_shared':
        buttons.push({ label: '进入课件开发', style: primaryBtn, action: 'develop' })
        break
      case 'developing':
        buttons.push({ label: '查看课件进度', style: primaryBtn, action: 'view_pipeline' })
        break
    }
    if (['draft', 'published_personal', 'revision'].includes(plan.status)) {
      buttons.push({ label: '删除教案', style: dangerBtn, action: 'delete',
        confirm: `确定删除教案「${plan.title}」吗？此操作不可恢复。` })
    }
  } else {
    if (['approved', 'published_shared'].includes(plan.status)) {
      buttons.push({ label: '🔀 Fork到我的草稿', style: primaryBtn, action: 'fork' })
    }
  }

  if (!buttons.length) return null

  return (
    <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', alignItems: 'center' }}>
      {actionLoading && <span style={{ fontSize: '13px', color: C.primary, marginRight: '4px' }}>处理中...</span>}
      {!actionLoading && buttons.map(btn => (
        <button key={btn.action} style={btn.style} disabled={isLoading}
          onClick={() => {
            if (btn.confirm && !window.confirm(btn.confirm)) return
            if (btn.action === 'view_pipeline') {
              if (plan.linked_pipeline_id) navigate(`/workflow/pipelines/${plan.linked_pipeline_id}`)
              else navigate('/workflow/pipelines')
              return
            }
            onAction(btn.action)
          }}>
          {btn.label}
        </button>
      ))}
    </div>
  )
}

/* ==================== 主组件 ==================== */
export default function PlanDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { user } = useAuth()

  const [plan, setPlan]           = useState<LessonPlan | null>(null)
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<TabKey>('content')
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [toast, setToast]         = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  useEffect(() => {
    if (!id) return
    setLoading(true); setError(null)
    getLessonPlan(id)
      .then(data => { setPlan(data); setLoading(false) })
      .catch(() => { setError('加载失败，教案不存在或无权限查看'); setLoading(false) })
  }, [id])

  const handleBack = () => {
    const from = (location.state as { from?: string })?.from
    if (from) { navigate(from); return }
    navigate('/lesson-plans/my-plans')
  }

  const handleAction = async (action: string) => {
    if (!plan || actionLoading) return
    setActionLoading(action)
    try {
      switch (action) {
        case 'publish':
          await publishLessonPlanPersonal(plan.id)
          showToast('教案已个人发布 ✓')
          break
        case 'submit':
          await submitLessonPlanForReview(plan.id)
          showToast('已提交教研组评审 ✓')
          break
        case 'develop': {
          // Phase6：startDevelopment 返回 pipeline_id
          const result = await startDevelopment(plan.id) as StartDevelopmentResult
          showToast(`已创建课件开发任务 ✓`)
          // 刷新教案数据（状态变为developing，linked_pipeline_id填入）
          const refreshed = await getLessonPlan(plan.id)
          setPlan(refreshed)
          // 自动切换到关联课件Tab
          setActiveTab('courseware')
          // 提示跳转
          setTimeout(() => {
            if (window.confirm(`课件开发任务已创建（Pipeline ID: ${result.pipeline_id}）\n是否立即前往课件审核系统查看？`)) {
              navigate(`/workflow/pipelines/${result.pipeline_id}`)
            }
          }, 500)
          return
        }
        case 'fork': {
          const forked = await forkLessonPlan(plan.id)
          showToast(`已Fork到我的草稿：${forked.title} ✓`)
          break
        }
        case 'delete':
          await deleteLessonPlan(plan.id)
          showToast('教案已删除')
          setTimeout(() => navigate('/lesson-plans/my-plans'), 1200)
          return
      }
      const refreshed = await getLessonPlan(plan.id)
      setPlan(refreshed)
    } catch (e) {
      console.error(`操作${action}失败:`, e)
      showToast('操作失败，请稍后重试', 'error')
    } finally {
      setActionLoading(null)
    }
  }

  if (loading) return <DetailSkeleton />

  if (error || !plan) {
    return (
      <div style={{ textAlign: 'center', padding: '80px 40px', background: C.card, borderRadius: '12px', border: `1px solid ${C.border}` }}>
        <div style={{ fontSize: '48px', marginBottom: '16px' }}>😕</div>
        <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>{error || '教案不存在'}</div>
        <button onClick={handleBack} style={{ marginTop: '16px', padding: '9px 20px', borderRadius: '8px',
          border: 'none', background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>
          返回列表
        </button>
      </div>
    )
  }

  const isOwner = plan.author_id === user?.id

  return (
    <div>
      {/* 返回 */}
      <button onClick={handleBack} style={{ display: 'flex', alignItems: 'center', gap: '6px',
        marginBottom: '16px', padding: '6px 0', background: 'none', border: 'none',
        fontSize: '13px', color: C.textSec, cursor: 'pointer' }}>
        ← 返回
      </button>

      {/* 头部信息卡 */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`,
        padding: '28px', marginBottom: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px', marginBottom: '20px' }}>
          <h1 style={{ fontSize: '22px', fontWeight: 700, color: C.text, margin: 0, lineHeight: 1.4, flex: 1 }}>
            {plan.title}
          </h1>
          <StatusBadge status={plan.status} />
        </div>

        <div style={{ display: 'flex', gap: '32px', flexWrap: 'wrap',
          paddingBottom: '20px', marginBottom: '20px', borderBottom: `1px solid ${C.border}` }}>
          <MetaTag icon="📚" label="学科" value={plan.subject} />
          <MetaTag icon="🎓" label="年级" value={plan.grade} />
          <MetaTag icon="⏱" label="课时" value={`${plan.duration_minutes} 分钟`} />
          <MetaTag icon="📌" label="课题" value={plan.topic} />
          {plan.author_name && <MetaTag icon="👤" label="作者" value={plan.author_name} />}
          {plan.ai_review_score != null && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
              <span style={{ fontSize: '11px', color: C.textMuted, fontWeight: 500 }}>AI评分</span>
              <span style={{ fontSize: '14px', fontWeight: 700, color: plan.ai_review_score >= 8.5 ? C.success : C.accent }}>
                🤖 {plan.ai_review_score.toFixed(1)}
              </span>
            </div>
          )}
        </div>

        <ActionBar plan={plan} isOwner={isOwner} actionLoading={actionLoading} onAction={handleAction} />
      </div>

      {/* Tab内容卡 */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`,
        boxShadow: '0 1px 3px rgba(0,0,0,0.04)', overflow: 'hidden' }}>
        {/* Tab导航 */}
        <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, padding: '0 4px' }}>
          {TABS.map(tab => {
            const isActive = activeTab === tab.key
            let label = tab.label
            if (tab.key === 'review' && plan.ai_review_score != null) {
              label = `🤖 AI评审 ${plan.ai_review_score.toFixed(1)}`
            }
            if (tab.key === 'courseware' && plan.linked_pipeline_id) {
              label = '🔗 关联课件 ●'
            }
            return (
              <button key={tab.key} onClick={() => setActiveTab(tab.key)} style={{
                padding: '14px 20px', border: 'none', background: 'transparent',
                fontSize: '13px', fontWeight: isActive ? 600 : 400,
                color: isActive ? C.primary : C.textSec,
                cursor: 'pointer',
                borderBottom: isActive ? `2px solid ${C.primary}` : '2px solid transparent',
                marginBottom: '-1px', transition: 'all 150ms ease', whiteSpace: 'nowrap',
              }}>{label}</button>
            )
          })}
        </div>

        {/* Tab内容 */}
        {activeTab === 'content'    && <ContentTab plan={plan} />}
        {activeTab === 'review'     && <ReviewTab plan={plan} />}
        {activeTab === 'stats'      && <StatsTab plan={plan} />}
        {activeTab === 'courseware' && (
          <CoursewareTab
            plan={plan}
            onNavigatePipeline={(pipelineId) => {
              if (pipelineId) navigate(`/workflow/pipelines/${pipelineId}`)
              else navigate('/workflow/pipelines')
            }}
          />
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
        }}>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}
