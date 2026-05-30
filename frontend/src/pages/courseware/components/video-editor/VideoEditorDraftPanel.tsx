/**
 * VideoEditorDraftPanel.tsx — 草稿恢复栏
 *
 * 从 VideoEditorModal.tsx 拆分(B1瘦身)
 *
 * 包含：草稿列表卡片（缩略图+名称+片段数+日期+加载/删除按钮）+ 收起按钮
 * 支持过渡动画（max-height + opacity）
 */
import { C } from './VideoEditorConstants'
import { withFirstFrame, seekToFirstFrame, timeAgo, fmtDate } from './VideoEditorUtils'
import type { VideoDraftItem } from '../../../../api/coursewares'

interface DraftPanelProps {
  drafts: VideoDraftItem[]
  visible: boolean
  coursewareId: string | undefined
  onLoadDraft: (draft: VideoDraftItem) => void
  onDeleteDraft: (draftId: string) => void
  onDismiss: () => void
}

export default function VideoEditorDraftPanel({
  drafts, visible, coursewareId, onLoadDraft, onDeleteDraft, onDismiss,
}: DraftPanelProps) {
  if (drafts.length === 0) return null

  const panelStyle: React.CSSProperties = {
    padding: visible ? '10px 24px' : '0 24px',
    background: '#7C3AED10',
    borderBottom: visible ? `1px solid ${C.border}` : 'none',
    flexShrink: 0,
    overflow: 'hidden',
    maxHeight: visible ? 280 : 0,
    opacity: visible ? 1 : 0,
    transition: 'max-height 300ms ease-out, opacity 250ms ease-out, padding 300ms ease-out, border-bottom 300ms ease-out',
  }

  return (
    <div style={panelStyle}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
        <span style={{ fontSize: 13, fontWeight: 600, color: C.text }}>📝 {drafts.length} 个草稿可恢复</span>
        <div style={{ flex: 1 }} />
        <button onClick={onDismiss} title="本次不加载任何草稿,收起此栏" style={{
          padding: '6px 14px', borderRadius: 6,
          border: `1px solid ${C.accent}`, background: C.accent + '20',
          color: C.accent, fontSize: 12, fontWeight: 600, cursor: 'pointer',
          display: 'flex', alignItems: 'center', gap: 4, whiteSpace: 'nowrap',
        }}>✕ 不加载,收起此栏</button>
      </div>
      <div style={{ display: 'flex', gap: 8, overflowX: 'auto', paddingBottom: 4 }}>
        {drafts.map(d => (
          <div key={d.id} style={{
            minWidth: 220, padding: '8px 10px', borderRadius: 8,
            border: '1px solid rgba(124,58,237,0.2)', background: C.surface, flexShrink: 0,
          }}>
            {(() => {
              try {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                const ca: any = Array.isArray(d.clips_data) ? d.clips_data : JSON.parse(d.clips_data)
                const u = ca[0]?.url
                if (u) return <video src={withFirstFrame(u)} preload="metadata" muted playsInline onLoadedMetadata={seekToFirstFrame} style={{ width: '100%', height: 80, objectFit: 'cover', borderRadius: 6, background: '#000', display: 'block', marginBottom: 6, pointerEvents: 'none' }} />
                return null
              } catch { return null }
            })()}
            <div style={{ fontSize: 12, fontWeight: 600, color: C.text, marginBottom: 2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{d.name || '未命名草稿'}</div>
            <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 2 }}>{d.clip_count} 个片段</div>
            <div style={{ fontSize: 10, color: C.textMuted, marginBottom: 8 }}>{fmtDate(d.created_at)}（{timeAgo(d.created_at)}）</div>
            <div style={{ display: 'flex', gap: 6 }}>
              <button onClick={() => onLoadDraft(d)} style={{ flex: 1, padding: '5px 0', borderRadius: 6, border: 'none', background: C.primary, color: '#fff', fontSize: 11, fontWeight: 600, cursor: 'pointer' }}>加载</button>
              <button onClick={() => {
                if (!coursewareId) return
                if (!confirm('确定删除草稿「' + (d.name || '未命名草稿') + '」吗?此操作不可恢复。')) return
                onDeleteDraft(d.id)
              }} style={{ padding: '5px 10px', borderRadius: 6, border: `1px solid ${C.danger}`, background: 'transparent', color: C.danger, fontSize: 11, cursor: 'pointer' }}>删除</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
