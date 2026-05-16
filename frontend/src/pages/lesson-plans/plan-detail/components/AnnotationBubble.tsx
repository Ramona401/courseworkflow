/**
 * AnnotationBubble.tsx — 单条批注显示组件（从PlanDetailTabs.tsx拆分）
 *
 * v104：isHistorical=true 时灰显且不显示操作按钮
 * v121（AI辅助修改 Bug 修复 · 方案C）：
 *   放宽按钮显示条件,archived(已归档)批注也显示操作区。
 *   背景:教研员退回时后端会把 archived 恢复为 pending(改动2+3),但为了兜底,
 *   前端也允许 archived 状态显示"AI辅助修改"按钮 — 双保险,防止未来出现
 *   新的退回场景漏改后端而导致按钮消失。
 *   显示条件从 {isOwner && !isHistorical && !isArchived} 改为
 *   {isOwner && !isHistorical}(允许 archived),操作按钮对 resolved 批注仍隐藏。
 */
import { type Annotation } from '@/api/annotations'

interface AnnotationBubbleProps {
  annotation: Annotation
  isOwner: boolean
  paragraphContent: string
  onResolve: (id: string, status: 'pending' | 'resolved') => void
  onAIFix: (annotation: Annotation) => void
  isHistorical?: boolean
}

export function AnnotationBubble({ annotation, isOwner, paragraphContent, onResolve, onAIFix, isHistorical }: AnnotationBubbleProps) {
  void paragraphContent
  const isResolved = annotation.status === 'resolved'
  const isArchived = annotation.status === 'archived'
  // showDimmed 仅控制视觉上的灰显(告诉用户这批注状态特殊),不再影响操作按钮是否渲染
  const showDimmed = isResolved || isArchived || isHistorical

  return (
    <div style={{
      padding: '10px 12px', borderRadius: '8px',
      background: showDimmed ? '#F9FAFB' : '#FFF7ED',
      border: `1px solid ${showDimmed ? '#E5E7EB' : '#FED7AA'}`,
      marginBottom: '8px', opacity: showDimmed ? 0.85 : 1,
      transition: 'all 200ms ease',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
        <span style={{ fontSize: '12px', fontWeight: 600, color: isHistorical ? '#9CA3AF' : '#92400E' }}>
          💬 {annotation.reviewer_name}
          {isHistorical && <span style={{ marginLeft: '6px', fontSize: '10px', color: '#9CA3AF', fontWeight: 400 }}>（历史批注）</span>}
          {isArchived && !isHistorical && <span style={{ marginLeft: '6px', fontSize: '10px', color: '#9CA3AF', fontWeight: 400 }}>（待重新处理）</span>}
          {isResolved && <span style={{ marginLeft: '6px', fontSize: '10px', color: '#16A34A', fontWeight: 400 }}>（已处理）</span>}
        </span>
        <span style={{ fontSize: '11px', color: '#9CA3AF' }}>{new Date(annotation.created_at).toLocaleDateString('zh-CN')}</span>
      </div>
      <div style={{ fontSize: '13px', color: '#374151', lineHeight: 1.7, marginBottom: isHistorical ? 0 : '8px' }}>
        {annotation.content}
      </div>
      {/* v121：放宽条件 — isOwner + 非历史轮次即可显示操作区
          (archived 也能用 AI 修改,因为退回场景下它们本质上仍是待处理批注) */}
      {isOwner && !isHistorical && (
        <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
          {/* AI辅助修改按钮:resolved 已处理的不显示(真的改完了);pending/archived 都能用 */}
          {!isResolved && (
            <button onClick={() => onAIFix(annotation)} style={{ padding: '3px 10px', borderRadius: '6px', border: 'none', background: '#EFF6FF', color: '#1D4ED8', fontSize: '11px', fontWeight: 600, cursor: 'pointer' }}>🤖 AI辅助修改</button>
          )}
          <button
            onClick={() => onResolve(annotation.id, isResolved ? 'pending' : 'resolved')}
            style={{ padding: '3px 10px', borderRadius: '6px', border: 'none', background: isResolved ? '#E5E7EB' : '#FEF3C7', color: isResolved ? '#6B7280' : '#92400E', fontSize: '11px', fontWeight: 600, cursor: 'pointer' }}
          >{isResolved ? '↩ 重新标记' : '✓ 标记已处理'}</button>
        </div>
      )}
    </div>
  )
}
