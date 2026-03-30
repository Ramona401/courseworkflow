/**
 * PlanDetailHeader.tsx — 教案详情页头部组件
 *   DetailSkeleton  — 骨架屏（加载中占位）
 *   StatusBadge     — 状态徽标
 *   MetaTag         — 元信息标签（学科/年级/课时等）
 *   ActionBar       — 操作按钮组（按状态动态渲染）
 */
import { useNavigate } from 'react-router-dom'
import type { LessonPlan, LessonPlanStatus } from '@/api/lesson-plans'
import { C, STATUS_CONFIG } from './planDetailConstants'

// ==================== 骨架屏 ====================

export function DetailSkeleton() {
  const shimmer: React.CSSProperties = {
    background: 'linear-gradient(90deg, #F3F4F6 25%, #E5E7EB 50%, #F3F4F6 75%)',
    backgroundSize: '200% 100%',
    animation: 'shimmer 1.4s infinite',
    borderRadius: '4px',
  }
  return (
    <div>
      <style>{`@keyframes shimmer { 0%{background-position:200% 0} 100%{background-position:-200% 0} }`}</style>
      {/* 头部骨架 */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '28px', marginBottom: '20px' }}>
        <div style={{ ...shimmer, width: '60%', height: '28px', marginBottom: '16px' }} />
        <div style={{ display: 'flex', gap: '24px', marginBottom: '16px' }}>
          {[1,2,3,4].map(i => <div key={i} style={{ ...shimmer, width: '80px', height: '36px' }} />)}
        </div>
        <div style={{ display: 'flex', gap: '8px' }}>
          {[1,2].map(i => <div key={i} style={{ ...shimmer, width: '100px', height: '34px', borderRadius: '8px' }} />)}
        </div>
      </div>
      {/* 内容骨架 */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '28px' }}>
        {[1,2,3,4,5].map(i => (
          <div key={i} style={{ ...shimmer, width: i%2===0 ? '80%' : '100%', height: '14px', marginBottom: '10px' }} />
        ))}
      </div>
    </div>
  )
}

// ==================== 状态徽标 ====================

export function StatusBadge({ status }: { status: LessonPlanStatus }) {
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

// ==================== 元信息标签 ====================

export function MetaTag({ icon, label, value }: { icon: string; label: string; value: string }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
      <span style={{ fontSize: '11px', color: C.textMuted, fontWeight: 500 }}>{label}</span>
      <span style={{ fontSize: '14px', color: C.text, display: 'flex', alignItems: 'center', gap: '4px' }}>
        <span>{icon}</span><span>{value}</span>
      </span>
    </div>
  )
}

// ==================== 操作按钮组 ====================

interface ActionBarProps {
  plan: LessonPlan
  isOwner: boolean
  actionLoading: string | null
  onAction: (action: string) => void
}

export function ActionBar({ plan, isOwner, actionLoading, onAction }: ActionBarProps) {
  const navigate = useNavigate()
  const isLoading = !!actionLoading

  // 按钮基础样式
  const primaryBtn: React.CSSProperties = {
    padding: '9px 20px', borderRadius: '8px', border: 'none',
    background: isLoading ? '#E5E7EB' : C.primary,
    color: isLoading ? C.textMuted : '#fff',
    fontSize: '14px', fontWeight: 600,
    cursor: isLoading ? 'not-allowed' : 'pointer',
    transition: 'all 150ms ease', whiteSpace: 'nowrap',
  }
  const secondaryBtn: React.CSSProperties = {
    padding: '9px 20px', borderRadius: '8px', border: `1px solid ${C.border}`,
    background: 'transparent', fontSize: '14px', color: C.textSec,
    cursor: isLoading ? 'not-allowed' : 'pointer',
    transition: 'all 150ms ease', whiteSpace: 'nowrap',
  }
  const dangerBtn: React.CSSProperties = {
    padding: '9px 20px', borderRadius: '8px', border: '1px solid #FEE2E2',
    background: 'transparent', fontSize: '14px', color: C.danger,
    cursor: isLoading ? 'not-allowed' : 'pointer',
    transition: 'all 150ms ease', whiteSpace: 'nowrap',
  }

  // 按状态动态构建按钮列表
  const buttons: Array<{ label: string; style: React.CSSProperties; action: string; confirm?: string }> = []

  if (isOwner) {
    switch (plan.status) {
      case 'draft':
        buttons.push({ label: '发布教案',   style: primaryBtn,   action: 'publish' })
        buttons.push({ label: '提交评审',   style: secondaryBtn, action: 'submit'  })
        break
      case 'published_personal':
        buttons.push({ label: '进入课件开发', style: primaryBtn,   action: 'develop' })
        buttons.push({ label: '提交评审',    style: secondaryBtn, action: 'submit'  })
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
    // 可删除的状态
    if (['draft', 'published_personal', 'revision'].includes(plan.status)) {
      buttons.push({
        label: '删除教案', style: dangerBtn, action: 'delete',
        confirm: `确定删除教案「${plan.title}」吗？此操作不可恢复。`,
      })
    }
  } else {
    // 非作者：可Fork
    if (['approved', 'published_shared'].includes(plan.status)) {
      buttons.push({ label: '🔀 Fork到我的草稿', style: primaryBtn, action: 'fork' })
    }
  }

  if (!buttons.length) return null

  return (
    <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', alignItems: 'center' }}>
      {/* 加载中提示 */}
      {actionLoading && (
        <span style={{ fontSize: '13px', color: C.primary, marginRight: '4px' }}>处理中...</span>
      )}
      {/* 按钮列表 */}
      {!actionLoading && buttons.map(btn => (
        <button
          key={btn.action}
          style={btn.style}
          disabled={isLoading}
          onClick={() => {
            if (btn.confirm && !window.confirm(btn.confirm)) return
            // 查看课件进度：直接跳转，不走onAction
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
