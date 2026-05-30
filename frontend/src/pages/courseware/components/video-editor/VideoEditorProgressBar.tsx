/**
 * VideoEditorProgressBar.tsx — 连贯预览进度条
 *
 * 从 VideoEditorModal.tsx 拆分(B1瘦身)
 * 包含：时间显示 + 片段分段背景 + 播放进度条
 */
import { useMemo } from 'react'
import { C } from './VideoEditorConstants'
import { fmtMS } from './VideoEditorUtils'
import type { EditorClip } from './VideoEditorTypes'

interface ProgressBarProps {
  clips: EditorClip[]
  playIdx: number
  playElapsed: number
  totalDur: number
}

export default function VideoEditorProgressBar({
  clips, playIdx, playElapsed, totalDur,
}: ProgressBarProps) {
  const playPct = totalDur > 0 ? (playElapsed / totalDur) * 100 : 0

  // 预计算每段的左偏移和宽度百分比（用 reduce 累积，避免渲染期变量重赋值）
  const segmentLayouts = useMemo(() => {
    const result: { l: number; w: number }[] = []
    clips.reduce((acc, cl) => {
      const d = cl.trimEnd - cl.trimStart
      result.push({
        l: totalDur > 0 ? (acc / totalDur) * 100 : 0,
        w: totalDur > 0 ? (d / totalDur) * 100 : 0,
      })
      return acc + d
    }, 0)
    return result
  }, [clips, totalDur])

  return (
    <div style={{ padding: '6px 24px', background: C.surface, borderBottom: `1px solid ${C.border}`, flexShrink: 0 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <span style={{ fontSize: 12, color: C.playing, fontWeight: 600, fontFamily: 'monospace', minWidth: 50 }}>{fmtMS(playElapsed)}</span>
        <div style={{ flex: 1, height: 6, borderRadius: 3, background: C.surfaceLight, overflow: 'hidden', position: 'relative' }}>
          {segmentLayouts.map((seg, i) => (
            <div key={i} style={{
              position: 'absolute', left: `${seg.l}%`, width: `${seg.w}%`, height: '100%',
              background: i === playIdx ? C.playing + '40' : C.primary + '20',
              borderRight: i < clips.length - 1 ? `1px solid ${C.border}` : 'none',
            }} />
          ))}
          <div style={{
            position: 'absolute', left: 0, top: 0, height: '100%',
            width: `${playPct}%`, background: C.playing,
            borderRadius: 3, transition: 'width 80ms linear',
          }} />
        </div>
        <span style={{ fontSize: 12, color: C.textMuted, fontFamily: 'monospace', minWidth: 50, textAlign: 'right' }}>{fmtMS(totalDur)}</span>
        <span style={{ fontSize: 11, color: C.playing }}>片段 {playIdx + 1}/{clips.length}</span>
      </div>
    </div>
  )
}
