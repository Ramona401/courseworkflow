/**
 * AnnotationBubble.tsx — 单条批注显示组件（从PlanDetailTabs.tsx拆分）
 * v104：isHistorical=true 时灰显且不显示操作按钮
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
  const showDimmed = isResolved || isArchived || isHistorical

  return (
    <div style={{
      padding: '10px 12px', borderRadius: '8px',
      background: showDimmed ? '#F9FAFB' : '#FFF7ED',
      border: `1px solid ${showDimmed ? '#E5E7EB' : '#FED7AA'}`,
      marginBottom: '8px', opacity: showDimmed ? 0.65 : 1,
      transition: 'all 200ms ease',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
        <span style={{ fontSize: '12px', fontWeight: 600, color: isHistorical ? '#9CA3AF' : '#92400E' }}>
          💬 {annotation.reviewer_name}
          {isHistorical && <span style={{ marginLeft: '6px', fontSize: '10px', color: '#9CA3AF', fontWeight: 400 }}>（历史批注）</span>}
          {isArchived && <span style={{ marginLeft: '6px', fontSize: '10px', color: '#9CA3AF', fontWeight: 400 }}>（已归档）</span>}
        </span>
        <span style={{ fontSize: '11px', color: '#9CA3AF' }}>{new Date(annotation.created_at).toLocaleDateString('zh-CN')}</span>
      </div>
      <div style={{ fontSize: '13px', color: '#374151', lineHeight: 1.7, marginBottom: isHistorical || isArchived ? 0 : '8px' }}>
        {annotation.content}
      </div>
      {isOwner && !isHistorical && !isArchived && (
        <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
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
