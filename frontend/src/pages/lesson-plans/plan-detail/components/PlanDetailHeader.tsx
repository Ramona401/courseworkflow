/**
 * PlanDetailHeader.tsx — 教案详情页头部组件
 *   DetailSkeleton  — 骨架屏（加载中占位）
 *   StatusBadge     — 状态徽标
 *   MetaTag         — 元信息标签（学科/年级/课时等）
 *   ActionBar       — 操作按钮组（按状态动态渲染 + 导出Word按钮）
 *
 * v142优化：exportWord 改为动态导入（按需加载 docx 库 ~300KB），
 *   用户点击「导出Word」按钮时才加载 docx + file-saver，
 *   PlanDetailPage chunk 从 418KB 降至 ~120KB
 */
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { LessonPlan, LessonPlanStatus } from '@/api/lesson-plans'
import type { InteractionCounts } from '@/api/lesson-plan-interactions'
import { C, STATUS_CONFIG } from './planDetailConstants'
// v142: 删除静态导入 exportLessonPlanToWord
// 改为 handleExportWord 内部 await import() 动态加载

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
  /** v125新增：互动数据（点赞/收藏计数+用户状态） */
  interactions?: InteractionCounts
  /** v125新增：切换互动回调 */
  onToggleInteraction?: (type: 'like' | 'favorite') => void
}

export function ActionBar({ plan, isOwner, actionLoading, onAction, interactions, onToggleInteraction }: ActionBarProps) {
  const navigate = useNavigate()
  const isLoading = !!actionLoading

  // 导出Word加载状态
  const [exporting, setExporting] = useState(false)

  /**
   * v142优化：点击"导出Word"时动态加载 docx 库
   * 首次点击会多等约 0.5-1 秒加载 chunk，后续浏览器缓存秒开
   */
  const handleExportWord = async () => {
    if (exporting) return
    setExporting(true)
    try {
      // 动态导入：docx + file-saver 只在用户点击时才加载（~300KB独立chunk）
      const { exportLessonPlanToWord } = await import('@/utils/exportWord')
      await exportLessonPlanToWord({
        title:            plan.title,
        subject:          plan.subject,
        grade:            plan.grade,
        topic:            plan.topic,
        duration_minutes: plan.duration_minutes,
        content_markdown: plan.content_markdown,
        author_name:      plan.author_name,
        ai_review_score:  plan.ai_review_score,
        created_at:       plan.created_at,
      })
    } catch (err) {
      console.error('导出Word失败:', err)
      alert('导出失败，请稍后重试')
    } finally {
      setExporting(false)
    }
  }

  // ---- 按钮样式 ----
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
  // 导出按钮：绿色系，独立于主操作加载状态
  const exportBtn: React.CSSProperties = {
    padding: '9px 20px', borderRadius: '8px',
    border: '1px solid rgba(16,185,129,0.3)',
    background: exporting ? '#F3F4F6' : 'rgba(16,185,129,0.08)',
    fontSize: '14px', fontWeight: 500,
    color: exporting ? C.textMuted : '#059669',
    cursor: exporting ? 'not-allowed' : 'pointer',
    transition: 'all 150ms ease', whiteSpace: 'nowrap',
    display: 'inline-flex', alignItems: 'center', gap: '5px',
  }

  // ---- 按状态动态构建主操作按钮列表 ----
  const buttons: Array<{ label: string; style: React.CSSProperties; action: string; confirm?: string }> = []

  if (isOwner) {
    switch (plan.status) {
      case 'draft':
        buttons.push({ label: '发布教案',   style: primaryBtn,   action: 'publish' })
        buttons.push({ label: '提交评审',   style: secondaryBtn, action: 'submit'  })
        break
      case 'published_personal':
        buttons.push({ label: '🎨 生成课件', style: primaryBtn,   action: 'courseware' })
        buttons.push({ label: '提交评审',    style: secondaryBtn, action: 'submit'  })
        break
      case 'revision':
        buttons.push({ label: '修改后重提', style: primaryBtn, action: 'submit' })
        // 支持退回备课工坊深度修改（大改场景）
        buttons.push({ label: '🛠 返回备课工坊', style: secondaryBtn, action: 'workshop' })
        break
      case 'approved':
      case 'published_shared':
        buttons.push({ label: '🎨 生成课件', style: primaryBtn, action: 'courseware' })
        break
      case 'developing':
        buttons.push({ label: '🎨 生成课件', style: primaryBtn, action: 'courseware' })
        break
    }
    if (['draft', 'published_personal', 'revision'].includes(plan.status)) {
      buttons.push({
        label: '删除教案', style: dangerBtn, action: 'delete',
        confirm: `确定删除教案「${plan.title}」吗？此操作不可恢复。`,
      })
    }
  } else {
    if (['approved', 'published_shared'].includes(plan.status)) {
      buttons.push({ label: '🔀 Fork到我的草稿', style: primaryBtn, action: 'fork' })
    }
  }

  return (
    <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', alignItems: 'center' }}>
      {/* 主操作加载提示 */}
      {actionLoading && (
        <span style={{ fontSize: '13px', color: C.primary, marginRight: '4px' }}>处理中...</span>
      )}

      {/* 主操作按钮列表 */}
      {!actionLoading && buttons.map(btn => (
        <button
          key={btn.action}
          style={btn.style}
          disabled={isLoading}
          onClick={() => {
            if (btn.confirm && !window.confirm(btn.confirm)) return
            if (btn.action === 'view_pipeline') {
              if (plan.linked_pipeline_id) navigate(`/workflow/pipelines/${plan.linked_pipeline_id}`)
              else navigate('/workflow/pipelines')
              return
            }
            // 返回备课工坊：写入sessionStorage后跳转，工坊会自动加载该教案
            if (btn.action === 'workshop') {
              sessionStorage.setItem('workshop_active_plan_id', plan.id)
              navigate('/lesson-plans')
              return
            }
            onAction(btn.action)
          }}>
          {btn.label}
        </button>
      ))}

      {/* 导出Word按钮：始终显示，不受主操作加载状态影响 */}
      <button
        style={exportBtn}
        disabled={exporting}
        onClick={handleExportWord}
        title="将教案导出为Word文档（.docx）">
        {exporting ? '⏳ 导出中...' : '📄 导出 Word'}
      </button>

      {/* v125新增：点赞 + 收藏按钮（与导出Word按钮统一风格） */}
      {interactions && onToggleInteraction && (
        <>
          <div style={{ width: '1px', height: '24px', background: '#E5E7EB', margin: '0 4px', flexShrink: 0 }} />
          <button
            onClick={() => onToggleInteraction('like')}
            title={interactions.is_liked ? '取消点赞' : '点赞'}
            style={{
              padding: '9px 20px', borderRadius: '8px',
              border: `1px solid ${interactions.is_liked ? 'rgba(239,68,68,0.3)' : C.border}`,
              background: interactions.is_liked ? 'rgba(239,68,68,0.06)' : 'transparent',
              fontSize: '14px', fontWeight: 500,
              color: interactions.is_liked ? '#DC2626' : C.textSec,
              cursor: 'pointer', transition: 'all 150ms ease', whiteSpace: 'nowrap',
              display: 'inline-flex', alignItems: 'center', gap: '5px',
            }}
          >
            <span style={{ fontSize: '15px' }}>{interactions.is_liked ? '👍' : '👍'}</span>
            {interactions.like_count > 0 ? `${interactions.like_count}` : '点赞'}
          </button>
          <button
            onClick={() => onToggleInteraction('favorite')}
            title={interactions.is_favorited ? '取消收藏' : '收藏'}
            style={{
              padding: '9px 20px', borderRadius: '8px',
              border: `1px solid ${interactions.is_favorited ? 'rgba(245,158,11,0.3)' : C.border}`,
              background: interactions.is_favorited ? 'rgba(245,158,11,0.06)' : 'transparent',
              fontSize: '14px', fontWeight: 500,
              color: interactions.is_favorited ? '#D97706' : C.textSec,
              cursor: 'pointer', transition: 'all 150ms ease', whiteSpace: 'nowrap',
              display: 'inline-flex', alignItems: 'center', gap: '5px',
            }}
          >
            <span style={{ fontSize: '15px' }}>📌</span>
            {interactions.favorite_count > 0 ? `${interactions.favorite_count}` : '收藏'}
          </button>
        </>
      )}

      <style>{`@keyframes spin { from{transform:rotate(0deg)} to{transform:rotate(360deg)} }`}</style>
    </div>
  )
}
