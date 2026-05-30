/**
 * VideoEditorToolbar.tsx — 视频编辑器顶部工具栏
 *
 * 从 VideoEditorModal.tsx 拆分(B1瘦身)
 *
 * 包含：标题/片段统计/音轨计数/草稿展开按钮/连贯预览/取消/导出
 */
import { C } from './VideoEditorConstants'
import { fmt } from './VideoEditorUtils'

interface ToolbarProps {
  clipCount: number
  totalDur: number
  audioClipCount: number
  draftCount: number
  draftDismissed: boolean
  playMode: 'none' | 'single' | 'all'
  exporting: boolean
  onToggleDrafts: () => void
  onPlayAll: () => void
  onStopAll: () => void
  onClose: () => void
  onExport: () => void
}

export default function VideoEditorToolbar({
  clipCount, totalDur, audioClipCount, draftCount, draftDismissed,
  playMode, exporting,
  onToggleDrafts, onPlayAll, onStopAll, onClose, onExport,
}: ToolbarProps) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
      padding: '10px 24px', borderBottom: `1px solid ${C.border}`, flexShrink: 0,
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <span style={{ fontSize: 18, fontWeight: 700, color: C.text }}>🎬 视频编辑器 v7</span>
        <span style={{ fontSize: 13, color: C.textMuted }}>{clipCount} 个片段 · {fmt(totalDur)}</span>
        {audioClipCount > 0 && (
          <span style={{ fontSize: 12, color: '#10B981', padding: '2px 8px', borderRadius: 4, background: 'rgba(16,185,129,0.1)' }}>
            🎵 {audioClipCount} 个音轨
          </span>
        )}
        {draftCount > 0 && draftDismissed && (
          <button onClick={onToggleDrafts} title="重新展开草稿恢复栏" style={{
            padding: '4px 10px', borderRadius: 6,
            border: '1px solid rgba(124,58,237,0.4)',
            background: 'rgba(124,58,237,0.15)',
            color: '#A78BFA', fontSize: 12, fontWeight: 600,
            cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4, whiteSpace: 'nowrap',
          }}>📝 {draftCount} 个草稿</button>
        )}
      </div>
      <div style={{ display: 'flex', gap: 10 }}>
        {clipCount >= 2 && (
          <button onClick={playMode === 'all' ? onStopAll : onPlayAll} style={{
            padding: '8px 20px', borderRadius: 8,
            border: `1px solid ${C.playing}`,
            background: playMode === 'all' ? C.playing + '20' : 'transparent',
            color: C.playing, fontSize: 14, fontWeight: 600, cursor: 'pointer',
          }}>{playMode === 'all' ? '⏹ 停止预览' : '▶ 连贯预览'}</button>
        )}
        <button onClick={onClose} style={{
          padding: '8px 20px', borderRadius: 8,
          border: `1px solid ${C.border}`, background: 'transparent',
          color: C.text, fontSize: 14, cursor: 'pointer',
        }}>取消</button>
        <button onClick={onExport} disabled={clipCount === 0 || exporting} style={{
          padding: '8px 24px', borderRadius: 8, border: 'none',
          background: clipCount > 0 && !exporting ? C.primary : '#4B5563',
          color: '#fff', fontSize: 14, fontWeight: 600,
          cursor: clipCount > 0 && !exporting ? 'pointer' : 'default',
        }}>{exporting ? '⏳ 导出中...' : `🎬 导出 (${fmt(totalDur)})`}</button>
      </div>
    </div>
  )
}
