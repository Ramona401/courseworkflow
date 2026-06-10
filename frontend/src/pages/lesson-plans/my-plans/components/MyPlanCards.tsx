/**
 * MyPlanCards.tsx — 我的教案页面卡片相关组件
 *   StatusBadge    — 状态徽标
 *   MetaTag        — 元信息标签
 *   ActionButtons  — 操作按钮组（按状态动态渲染）
 *   PlanCard       — 教案卡片（含悬停效果+跳转详情）
 *   SkeletonCard   — 加载骨架屏
 *   EmptyState     — 空状态引导
 *
 * v168改动（第二批治本·功能A·正文产出硬门控 切片A·前端体验层）：
 *   ActionButtons 曾在教案正文（content_markdown）为空时禁用 submit/publish/courseware 按钮。
 *
 * Bug 修复（移除列表页正文门控）：
 *   v168 的正文门控在【我的教案列表页】存在误伤——列表数据来自 getLessonPlans()
 *   返回的 LessonPlanListItem 精简结构，该结构【不包含 content_markdown 字段】，
 *   导致 plan.content_markdown 恒为 undefined，isContentEmpty 恒为 true，
 *   于是 submit/publish/courseware 三个按钮对【所有】教案永远置灰（即使正文完整）。
 *   这正是"列表页按钮灰、详情页能点"现象的根因（详情页数据带 content_markdown，判断准确）。
 *
 *   修复策略：列表页【移除】正文门控，让按钮正常可点。理由——
 *     1) 列表页拿不到正文字段，前端无法准确判断"正文是否为空"，硬判必然误伤；
 *     2) "无正文不可发布/提交/生成课件"的真正防线在【后端】（SubmitForReview/
 *        PublishPersonal/PublishShared/createCourseware 均有正文非空硬校验，返回 400）；
 *     3) MyPlansPage.handleAction 已有 try/catch，后端拒绝时会弹 toast 提示用户去
 *        「继续备课」补全正文。准确性优先于"少一次后端往返"的微小即时性。
 *   注意：详情页 PlanDetailHeader.tsx 的同名门控【保留不动】——详情页数据带
 *   content_markdown，判断准确，门控能省一次无谓的后端往返。两页面差异是有意为之。
 */
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { LessonPlan, LessonPlanStatus } from '@/api/lesson-plans'
import { C, STATUS_CONFIG } from './myPlansConstants'

// ==================== 状态徽标 ====================

export function StatusBadge({ status }: { status: LessonPlanStatus }) {
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

// ==================== 元信息标签 ====================

export function MetaTag({ icon, text, maxWidth }: { icon: string; text: string; maxWidth?: string }) {
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: '4px',
      fontSize: '13px', color: '#6B7280',
      maxWidth, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
    }}>
      <span style={{ flexShrink: 0 }}>{icon}</span>
      <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{text}</span>
    </span>
  )
}

// ==================== 操作按钮组 ====================

interface ActionButtonsProps {
  plan: LessonPlan
  onAction: (planId: string, action: string) => void
  loadingId: string | null
}

export function ActionButtons({ plan, onAction, loadingId }: ActionButtonsProps) {
  const isLoading = loadingId === plan.id
  const navigate  = useNavigate()

  // 说明：此前 v168 在此处依据 plan.content_markdown 判断"正文为空"并禁用
  // submit/publish/courseware 按钮。但列表页的 LessonPlanListItem 不含
  // content_markdown 字段，该判断恒为"空"，造成所有教案按钮误置灰。
  // 已移除该门控——按钮正常可点，正文非空校验由后端负责（返回 400 后
  // MyPlansPage.handleAction 的 catch 会弹 toast 提示用户去补全正文）。

  const btnBase: React.CSSProperties = {
    padding: '5px 12px', borderRadius: '6px',
    fontSize: '12px', fontWeight: 500,
    cursor: isLoading ? 'not-allowed' : 'pointer',
    border: 'none', transition: 'all 150ms ease', whiteSpace: 'nowrap',
  }
  const primaryBtn: React.CSSProperties   = { ...btnBase, background: isLoading ? C.border : C.primary, color: isLoading ? C.textMuted : '#fff' }
  const secondaryBtn: React.CSSProperties = { ...btnBase, background: 'transparent', border: `1px solid ${C.border}`, color: C.textSec }
  const dangerBtn: React.CSSProperties    = { ...btnBase, background: 'transparent', border: '1px solid #FEE2E2', color: C.danger }

  // 按状态动态构建操作列表
  const actions: Array<{ label: string; style: React.CSSProperties; action: string; confirm?: string }> = []

  switch (plan.status) {
    case 'draft':
      actions.push({ label: '✏️ 继续备课', style: primaryBtn,   action: 'resume'  })
      actions.push({ label: '发布教案',    style: secondaryBtn, action: 'publish' })
      actions.push({ label: '提交评审',    style: secondaryBtn, action: 'submit'  })
      break
    case 'published_personal':
      actions.push({ label: '✏️ 继续备课',  style: secondaryBtn, action: 'resume'  })
      actions.push({ label: '🎨 生成课件',  style: primaryBtn,   action: 'courseware' })
      actions.push({ label: '提交评审',    style: secondaryBtn, action: 'submit'  })
      break
    case 'submitted':
      break
    case 'revision':
      actions.push({ label: '修改后重提', style: primaryBtn, action: 'submit' })
      break
    case 'approved':
    case 'published_shared':
      actions.push({ label: '🎨 生成课件', style: primaryBtn, action: 'courseware' })
      break
    case 'developing':
      actions.push({ label: '🎨 生成课件', style: primaryBtn, action: 'courseware' })
      break
    case 'completed':
      break
  }

  // 可删除状态下追加删除按钮
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
            // 查看课件进度：直接跳转，不走onAction
            if (a.action === 'courseware') { onAction(plan.id, a.action); return }
            onAction(plan.id, a.action)
          }}>
          {a.label}
        </button>
      ))}
    </div>
  )
}

// ==================== 教案卡片 ====================

interface PlanCardProps {
  plan: LessonPlan
  onAction: (planId: string, action: string) => void
  loadingId: string | null
}

export function PlanCard({ plan, onAction, loadingId }: PlanCardProps) {
  const [hovered, setHovered] = useState(false)
  const navigate    = useNavigate()
  const statusCfg   = STATUS_CONFIG[plan.status] || STATUS_CONFIG.draft

  const formatDate = (iso: string) => {
    try {
      const d = new Date(iso)
      return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`
    } catch { return iso }
  }

  // 点击卡片主体跳转详情（底部操作区不触发）
  const handleCardClick = () => {
    navigate(`/lesson-plans/plans/${plan.id}`, { state: { from: '/lesson-plans/my-plans' } })
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
      }}>

      {/* 顶行：标题 + 状态 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '12px', marginBottom: '10px' }}>
        <h3 style={{ fontSize: '15px', fontWeight: 600, color: C.text, margin: 0, lineHeight: 1.5, flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
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
        {plan.recipe_name && <MetaTag icon="📦" text={plan.recipe_name} maxWidth="160px" />}
      </div>

      {/* AI评分 */}
      {plan.ai_review_score != null && (
        <div style={{
          display: 'inline-flex', alignItems: 'center', gap: '6px',
          padding: '4px 10px', borderRadius: '20px', marginBottom: '12px',
          background: plan.ai_review_score >= 8.5 ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)',
        }}>
          <span style={{ fontSize: '12px' }}>🤖</span>
          <span style={{ fontSize: '12px', fontWeight: 600, color: plan.ai_review_score >= 8.5 ? C.success : C.accent }}>
            AI评分 {plan.ai_review_score.toFixed(1)}
          </span>
        </div>
      )}

      {/* 退回说明 */}
      {plan.status === 'revision' && (
        <div style={{ padding: '10px 14px', background: 'rgba(249,115,22,0.06)', borderRadius: '8px', fontSize: '12px', color: C.warning, marginBottom: '12px', lineHeight: 1.7, borderLeft: '3px solid #F97316' }}>
          <div style={{ fontWeight: 600, marginBottom: '4px' }}>↩️ 已被退回修改</div>
          <div style={{ color: '#6B7280' }}>评审员已留下修改意见，点击「查看详情」可查看完整评审意见和改进建议。</div>
        </div>
      )}

      {/* 底行：时间 + 操作（阻止冒泡）*/}
      <div
        style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingTop: '12px', borderTop: `1px solid ${C.border}`, gap: '12px', flexWrap: 'wrap' }}
        onClick={e => e.stopPropagation()}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '12px', color: C.textMuted, flexShrink: 0 }}>
            更新于 {formatDate(plan.updated_at)}
          </span>
          <button
            onClick={e => {
              e.stopPropagation()
              navigate(`/lesson-plans/plans/${plan.id}`, { state: { from: '/lesson-plans/my-plans' } })
            }}
            style={{ padding: 0, background: 'none', border: 'none', fontSize: '12px', color: C.primary, cursor: 'pointer', textDecoration: 'underline', textDecorationColor: 'rgba(79,123,232,0.3)', flexShrink: 0 }}>
            查看详情
          </button>
        </div>
        <ActionButtons plan={plan} onAction={onAction} loadingId={loadingId} />
      </div>
    </div>
  )
}

// ==================== 骨架屏 ====================

export function SkeletonCard() {
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
        {[1,2,3].map(i => <div key={i} style={{ ...shimmer, width: '60px', height: '14px' }} />)}
      </div>
      <div style={{ ...shimmer, width: '100%', height: '1px', marginBottom: '12px' }} />
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <div style={{ ...shimmer, width: '30%', height: '12px' }} />
        <div style={{ ...shimmer, width: '25%', height: '26px', borderRadius: '6px' }} />
      </div>
    </div>
  )
}

// ==================== 空状态 ====================

interface EmptyStateProps {
  filtered: boolean
  onReset: () => void
}

export function EmptyState({ filtered, onReset }: EmptyStateProps) {
  const navigate = useNavigate()
  return (
    <div style={{ gridColumn: '1 / -1', textAlign: 'center', padding: '80px 40px', background: C.card, borderRadius: '12px', border: `1px solid ${C.border}` }}>
      <div style={{ fontSize: '48px', marginBottom: '16px' }}>{filtered ? '🔍' : '📋'}</div>
      <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
        {filtered ? '没有符合条件的教案' : '还没有教案'}
      </div>
      <div style={{ fontSize: '14px', color: C.textMuted, marginBottom: '24px', lineHeight: 1.7 }}>
        {filtered ? '试试调整筛选条件，或清空筛选查看全部' : '前往备课工坊创建您的第一份AI辅助教案'}
      </div>
      {filtered ? (
        <button onClick={onReset} style={{ padding: '10px 24px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>
          清空筛选
        </button>
      ) : (
        <button onClick={() => navigate('/lesson-plans')} style={{ padding: '10px 24px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>
          ✨ 去备课工坊
        </button>
      )}
    </div>
  )
}
