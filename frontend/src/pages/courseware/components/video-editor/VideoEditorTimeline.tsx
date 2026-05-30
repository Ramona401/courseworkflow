/**
 * VideoEditorTimeline.tsx — 多轨道时间轴(三轨堆叠 + 标尺播放头 + 原地裁剪手柄)
 *
 * 从 VideoEditorModal.tsx v6.1 拆分而来(v0.42.7)
 *
 * v0.42.7 新增:
 *   - 视频片段卡片左右两侧 6px 原地裁剪手柄(pointer 拖拽实时改 trimStart/trimEnd,松手生效)
 *
 * 职责:
 *   - 左侧 88px 固定轨道标签栏(不参与水平滚动)
 *   - 右侧滚动区: 标尺行 + 视频轨片段行 + 音频轨片段行 + 字幕轨占位行
 *   - 视频片段: 缩略图 + 拖拽排序 + 原地裁剪手柄 + 删除按钮
 *   - 音频片段: 差异化 SVG 波形(基于 clip.id 哈希) / 无音轨占位
 *   - 字幕片段: v0.42.8 占位提示
 *   - 标尺: 每秒一个 tick + 每 5 秒大刻度带 MM:SS 标签
 *   - 播放头: 三角形+竖线,可拖拽精准定位时间
 */
import { Fragment, useEffect, useRef, useState } from 'react'
import type { EditorClip, TrimDragState } from './VideoEditorTypes'
import {
  C, TRANS, TRACKS, TRACK_LABEL_WIDTH,
  getTransColor, MIN_DURATION, TRIM_HANDLE_WIDTH, PX_PER_SEC,
} from './VideoEditorConstants'
import {
  fmt, fmtMS, withFirstFrame, seekToFirstFrame,
  clipPxWidth, timeToPixel, totalPxWidth, buildTicks,
  buildWaveformPath,
} from './VideoEditorUtils'
import VideoEditorSubtitleTrack from './VideoEditorSubtitleTrack'
import type { SubtitleSegment } from './VideoEditorSubtitleTrack'

interface VideoEditorTimelineProps {
  clips: EditorClip[]
  activeIdx: number
  playIdx: number
  playMode: 'none' | 'single' | 'all'
  dragIdx: number
  dragOverIdx: number
  timelineDragOver: boolean
  /** 播放头当前应显示的全局时间(秒) */
  phTime: number
  /** 是否正在拖播放头 */
  draggingPH: boolean
  /** 拖播放头时的实时时间(用于显示浮动 tooltip) */
  phDragTime: number
  /** 设置激活片段 */
  setActiveIdx: (idx: number) => void
  /** 删除片段 */
  removeClip: (idx: number) => void
  /** 更新片段字段 */
  updateClip: (idx: number, u: Partial<EditorClip>) => void
  /** 单段预览 */
  playSingle: (idx: number) => void
  /** 拖拽排序回调 */
  handleDragStart: (idx: number, e: React.DragEvent) => void
  handleDragOver: (e: React.DragEvent, idx: number) => void
  handleDrop: (idx: number) => void
  setDragIdx: (n: number) => void
  setDragOverIdx: (n: number) => void
  /** 从素材库拖入回调 */
  handleTimelineDragOver: (e: React.DragEvent) => void
  handleTimelineDragLeave: () => void
  handleTimelineDrop: (e: React.DragEvent) => void
  /** 标尺点击跳转 */
  handleRulerClick: (e: React.MouseEvent) => void
  rulerRef: React.RefObject<HTMLDivElement | null>
  /** 播放头 pointerdown */
  handlePHDown: (e: React.PointerEvent) => void
  /** v0.42.8: 字幕片段列表 */
  subtitleSegments: SubtitleSegment[]
  /** v0.42.8: 当前字幕语言 */
  subtitleLanguage: string
  /** v0.42.8: 字幕变更回调 */
  onSubtitleSegmentsChange: (segs: SubtitleSegment[]) => void
  /** v0.42.8: 双击字幕条编辑 */
  onEditSubtitleSegment: (seg: SubtitleSegment) => void
  /** v0.42.9: 请求打开 TTS 配音弹窗 */
  onRequestTTS?: () => void
}

export default function VideoEditorTimeline(props: VideoEditorTimelineProps) {
  const {
    clips, activeIdx, playIdx, playMode, dragIdx, dragOverIdx, timelineDragOver,
    phTime: _phTime, draggingPH, phDragTime,
    setActiveIdx, removeClip, updateClip, playSingle,
    handleDragStart, handleDragOver, handleDrop, setDragIdx, setDragOverIdx,
    handleTimelineDragOver, handleTimelineDragLeave, handleTimelineDrop,
    handleRulerClick, rulerRef, handlePHDown,
    subtitleSegments, subtitleLanguage, onSubtitleSegmentsChange, onEditSubtitleSegment, onRequestTTS,
  } = props

  // 缓存计算值
  const totalDur = clips.reduce((s, c) => s + (c.trimEnd - c.trimStart), 0)
  const tlWidth = totalPxWidth(clips)
  const ticks = clips.length > 0 ? buildTicks(totalDur, clips) : []

  // 重新计算播放头像素位置(用 props.phTime,父组件已计算)
  const phPx = clips.length > 0 ? timeToPixel(_phTime, clips) : 0

  // === v0.42.7 原地裁剪手柄拖拽状态 ===
  const [trimDrag, setTrimDrag] = useState<TrimDragState | null>(null)
  const trimDragRef = useRef<TrimDragState | null>(null)
  useEffect(() => { trimDragRef.current = trimDrag }, [trimDrag])

  // 全局监听 pointer 事件实现拖拽(无论鼠标在哪都响应)
  // A1修复: controlled draft 模式 — 拖动期间只更新本地 draftTrim state(零重渲染),
  //         pointerup 时一次性调 updateClip 提交最终值,避免每像素触发整个时间轴重渲染
  const [draftTrim, setDraftTrim] = useState<{ clipIdx: number; side: 'left' | 'right'; value: number } | null>(null)
  const draftTrimRef = useRef(draftTrim)
  useEffect(() => { draftTrimRef.current = draftTrim }, [draftTrim])

  useEffect(() => {
    if (!trimDrag) return
    const clipsSnapshot = clips // 闭包内固定 clips 引用

    const onMove = (e: PointerEvent) => {
      const tcurrent = trimDragRef.current
      if (!tcurrent) return
      const dx = e.clientX - tcurrent.startX
      const dt = dx / PX_PER_SEC
      const clip = clipsSnapshot[tcurrent.clipIdx]
      if (!clip) return

      let newVal: number
      if (tcurrent.side === 'left') {
        newVal = Math.max(0, Math.min(clip.trimEnd - MIN_DURATION, tcurrent.initialValue + dt))
      } else {
        newVal = Math.max(clip.trimStart + MIN_DURATION, Math.min(clip.duration, tcurrent.initialValue + dt))
      }
      // 只更新本地 draft(不触发 clips 重渲染)
      setDraftTrim({ clipIdx: tcurrent.clipIdx, side: tcurrent.side, value: newVal })
    }

    const onUp = () => {
      // pointerup 时一次性提交到 clips
      const draft = draftTrimRef.current
      const tc = trimDragRef.current
      if (draft && tc) {
        if (draft.side === 'left') updateClip(draft.clipIdx, { trimStart: draft.value })
        else updateClip(draft.clipIdx, { trimEnd: draft.value })
      }
      setDraftTrim(null)
      setTrimDrag(null)
    }

    document.addEventListener('pointermove', onMove)
    document.addEventListener('pointerup', onUp)
    return () => {
      document.removeEventListener('pointermove', onMove)
      document.removeEventListener('pointerup', onUp)
    }
  }, [trimDrag, clips, updateClip])

  /** 启动裁剪手柄拖拽(在 pointerdown 时调用) */
  const startTrimDrag = (clipIdx: number, side: 'left' | 'right', e: React.PointerEvent) => {
    e.preventDefault()
    e.stopPropagation()
    const clip = clips[clipIdx]
    if (!clip) return
    setTrimDrag({
      clipIdx,
      side,
      startX: e.clientX,
      initialValue: side === 'left' ? clip.trimStart : clip.trimEnd,
    })
  }

  // v0.42.7 P2.1: 音频波形预计算
  const waveformPaths: Record<string, { topD: string; botD: string }> = {}
  clips.forEach(c => { if (c.audioUrl) waveformPaths[c.id] = buildWaveformPath(c.id, 28) })

  // 渲染空时间轴的占位
  if (clips.length === 0) {
    return (
      <div onDragOver={handleTimelineDragOver} onDragLeave={handleTimelineDragLeave} onDrop={handleTimelineDrop}>
        <div style={{
          textAlign: 'center', padding: '24px 0', color: C.textMuted, fontSize: 13,
          border: `2px dashed ${timelineDragOver ? C.primary : C.border}`,
          borderRadius: 10, transition: 'all 200ms',
          background: timelineDragOver ? C.primaryLight : 'transparent',
        }}>
          {timelineDragOver ? '🎯 松开鼠标添加到时间轴' : '从左侧素材库拖拽或点击视频添加到这里'}
        </div>
      </div>
    )
  }

  return (
    <div onDragOver={handleTimelineDragOver} onDragLeave={handleTimelineDragLeave} onDrop={handleTimelineDrop}>
      <div style={{
        display: 'flex', alignItems: 'flex-start',
        border: timelineDragOver ? `2px dashed ${C.primary}` : '2px dashed transparent',
        borderRadius: 10, transition: 'border-color 200ms',
      }}>
        {/* ===== 左侧轨道标签栏(不参与水平滚动) ===== */}
        <div style={{ width: TRACK_LABEL_WIDTH, flexShrink: 0, paddingTop: 0, paddingRight: 4 }}>
          {/* 标尺对应的空位(高度匹配标尺 28+4) */}
          <div style={{ height: 32 }} />
          {/* 三个轨道标签 */}
          {TRACKS.map(t => (
            <div key={t.type} style={{
              height: t.height, marginTop: 4, borderRadius: 6,
              background: t.color + '15',
              border: `1px solid ${t.color}40`,
              display: 'flex', flexDirection: 'column',
              alignItems: 'flex-start', justifyContent: 'center',
              paddingLeft: 8, gap: 2,
              opacity: t.enabled ? 1 : 0.5, boxSizing: 'border-box',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <span style={{ fontSize: 14 }}>{t.emoji}</span>
                <span style={{ fontSize: 11, color: t.color, fontWeight: 700 }}>{t.label}</span>
              </div>
              {!t.enabled && t.comingHint && (
                <span style={{ fontSize: 9, color: C.textMuted, fontStyle: 'italic' }}>{t.comingHint}</span>
              )}
            </div>
          ))}
        </div>

        {/* ===== 右侧滚动区 ===== */}
        <div style={{ flex: 1, minWidth: 0, overflowX: 'auto', paddingBottom: 4 }}>
          {/* ----- 标尺行 ----- */}
          <div
            ref={rulerRef}
            onClick={handleRulerClick}
            style={{
              position: 'relative', height: 28,
              minWidth: Math.max(tlWidth, 100),
              cursor: 'pointer', marginBottom: 4, userSelect: 'none',
            }}
          >
            {ticks.map((tk, i) => (
              <Fragment key={i}>
                <div style={{
                  position: 'absolute', left: tk.px, bottom: 0,
                  width: 1, height: tk.major ? 14 : 7,
                  background: tk.major ? C.textMuted : C.border,
                }} />
                {tk.label && (
                  <div style={{
                    position: 'absolute', left: tk.px - 14, top: 1,
                    width: 28, textAlign: 'center', fontSize: 9,
                    color: C.textMuted, fontFamily: 'monospace', pointerEvents: 'none',
                  }}>{tk.label}</div>
                )}
              </Fragment>
            ))}
            {/* 播放头(可拖拽) */}
            <div onPointerDown={handlePHDown} style={{
              position: 'absolute', left: phPx - 7, top: 0,
              width: 14, cursor: 'ew-resize', zIndex: 5, touchAction: 'none',
            }}>
              <svg width="14" height="12" viewBox="0 0 14 12" style={{ display: 'block', margin: '0 auto' }}>
                <polygon points="0,0 14,0 7,12" fill={C.accent} />
              </svg>
              <div style={{ width: 1, height: 16, background: C.accent, margin: '0 auto' }} />
            </div>
            {/* 拖播放头时的时间 tooltip */}
            {draggingPH && (
              <div style={{
                position: 'absolute', left: phPx - 22, top: -20,
                width: 44, textAlign: 'center', fontSize: 10, fontWeight: 700,
                color: C.accent, fontFamily: 'monospace',
                background: C.bg, borderRadius: 4, padding: '2px 0',
                border: `1px solid ${C.accent}40`,
              }}>{fmtMS(phDragTime)}</div>
            )}
          </div>

          {/* ----- 视频轨片段行 ----- */}
          <div style={{ display: 'flex', gap: 0, alignItems: 'center', minWidth: tlWidth, marginTop: 4 }}>
            {clips.map((clip, idx) => {
              const dur = clip.trimEnd - clip.trimStart
              const w = clipPxWidth(dur)
              const isAct = idx === activeIdx
              const isPlay = idx === playIdx && playMode !== 'none'
              const isDO = idx === dragOverIdx && dragIdx !== idx
              const tc = getTransColor(clip.transition)
              const overlayBg = isPlay ? 'rgba(34,211,238,0.45)' : isAct ? 'rgba(124,58,237,0.45)' : 'rgba(0,0,0,0.45)'
              // v0.42.7: 只在片段宽度 ≥ 80px 时显示裁剪手柄(短片段无法挤下)
              const showTrimHandles = w >= 80
              // v0.42.7: 当前正在拖此片段的某个手柄,提供视觉反馈
              const isTrimmingLeft = trimDrag?.clipIdx === idx && trimDrag.side === 'left'
              const isTrimmingRight = trimDrag?.clipIdx === idx && trimDrag.side === 'right'
              // A1: 拖动期间用 draftTrim 实时值渲染指示条(视觉反馈),而非等 pointerup 才更新
              const effectiveTrimStart = (draftTrim?.clipIdx === idx && draftTrim.side === 'left') ? draftTrim.value : clip.trimStart
              const effectiveTrimEnd = (draftTrim?.clipIdx === idx && draftTrim.side === 'right') ? draftTrim.value : clip.trimEnd
              const effectiveDur = effectiveTrimEnd - effectiveTrimStart

              return (
                <div key={clip.id} style={{ display: 'flex', alignItems: 'center' }}>
                  <div
                    // v0.42.7: 裁剪手柄拖拽中时禁用 draggable,避免冲突
                    draggable={!trimDrag}
                    onDragStart={e => handleDragStart(idx, e)}
                    onDragOver={e => handleDragOver(e, idx)}
                    onDrop={() => handleDrop(idx)}
                    onDragEnd={() => { setDragIdx(-1); setDragOverIdx(-1) }}
                    onClick={() => {
                      // v0.42.7: 拖拽手柄期间禁止 onClick 触发预览
                      if (trimDrag) return
                      setActiveIdx(idx)
                      if (playMode === 'none') playSingle(idx)
                    }}
                    style={{
                      width: w, height: 56, borderRadius: 8,
                      cursor: trimDrag ? 'ew-resize' : 'grab',
                      background: '#000',
                      border: `2px solid ${isDO ? C.accent : isPlay ? C.playing : isAct ? C.primary : 'transparent'}`,
                      boxSizing: 'border-box', transition: trimDrag ? 'none' : 'all 150ms',
                      position: 'relative', overflow: 'hidden',
                      opacity: dragIdx === idx ? 0.5 : 1,
                    }}
                  >
                    {/* 视频缩略图 */}
                    <video
                      src={withFirstFrame(clip.url)}
                      preload="metadata" muted playsInline
                      onLoadedMetadata={seekToFirstFrame}
                      style={{
                        position: 'absolute', inset: 0,
                        width: '100%', height: '100%',
                        objectFit: 'cover', pointerEvents: 'none',
                      }}
                    />
                    {/* 颜色遮罩 */}
                    <div style={{ position: 'absolute', inset: 0, pointerEvents: 'none', background: overlayBg }} />
                    {/* 底部裁剪范围指示条(整段长度的相对比例) */}
                    <div style={{
                      position: 'absolute', bottom: 0,
                      left: `${(effectiveTrimStart / clip.duration) * 100}%`,
                      width: `${(effectiveDur / clip.duration) * 100}%`,
                      height: 3, background: isPlay ? C.playing : C.primary,
                      borderRadius: 2, zIndex: 2,
                    }} />
                    {/* 正在播放圆点 */}
                    {isPlay && (
                      <div style={{
                        position: 'absolute', top: 4, right: 6,
                        width: 8, height: 8, borderRadius: '50%',
                        background: C.playing, boxShadow: `0 0 6px ${C.playing}`, zIndex: 2,
                      }} />
                    )}
                    {/* 静音标记 */}
                    {clip.muted && (
                      <div style={{
                        position: 'absolute', top: 3, left: 6,
                        fontSize: 10, zIndex: 2,
                        textShadow: '0 1px 2px rgba(0,0,0,0.8)',
                      }} title="已静音">🔇</div>
                    )}

                    {/* 文字内容层 */}
                    <div style={{
                      position: 'absolute', inset: 0,
                      padding: '6px 10px', boxSizing: 'border-box',
                      pointerEvents: 'none',
                      display: 'flex', flexDirection: 'column',
                      justifyContent: 'space-between', zIndex: 3,
                    }}>
                      <div style={{
                        fontSize: 11, fontWeight: 600, color: '#fff',
                        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                        paddingLeft: clip.muted ? 16 : 0,
                        textShadow: '0 1px 2px rgba(0,0,0,0.85)',
                      }}>{clip.label.length > 12 ? clip.label.slice(0, 12) + '...' : clip.label}</div>
                      <div style={{
                        fontSize: 10, color: '#fff',
                        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                        textShadow: '0 1px 2px rgba(0,0,0,0.85)',
                      }}>
                        <span>{fmt(effectiveDur)}</span>
                        <div style={{ display: 'flex', gap: 3, pointerEvents: 'auto' }}>
                          {/* 删除按钮 */}
                          <button
                            onClick={e => { e.stopPropagation(); removeClip(idx) }}
                            style={{
                              background: 'rgba(0,0,0,0.55)', border: 'none',
                              color: '#FCA5A5', fontSize: 12, cursor: 'pointer',
                              padding: '0 5px', lineHeight: 1.2, borderRadius: 3,
                            }}
                          >✕</button>
                        </div>
                      </div>
                    </div>

                    {/* v0.42.7: 左侧裁剪手柄 */}
                    {showTrimHandles && (
                      <div
                        onPointerDown={e => startTrimDrag(idx, 'left', e)}
                        title="拖拽调整入点(松手生效)"
                        style={{
                          position: 'absolute', left: 0, top: 0,
                          width: TRIM_HANDLE_WIDTH, height: '100%',
                          cursor: 'ew-resize',
                          background: isTrimmingLeft ? C.accent : 'rgba(124,58,237,0.6)',
                          borderRight: `1px solid ${isTrimmingLeft ? C.accent : 'rgba(255,255,255,0.3)'}`,
                          zIndex: 4, touchAction: 'none',
                          transition: isTrimmingLeft ? 'none' : 'background 120ms',
                        }}
                      >
                        {/* 手柄视觉提示: 中间一条小白线 */}
                        <div style={{
                          position: 'absolute', left: '50%', top: '40%', transform: 'translate(-50%, 0)',
                          width: 1, height: '20%', background: 'rgba(255,255,255,0.8)',
                        }} />
                      </div>
                    )}

                    {/* v0.42.7: 右侧裁剪手柄 */}
                    {showTrimHandles && (
                      <div
                        onPointerDown={e => startTrimDrag(idx, 'right', e)}
                        title="拖拽调整出点(松手生效)"
                        style={{
                          position: 'absolute', right: 0, top: 0,
                          width: TRIM_HANDLE_WIDTH, height: '100%',
                          cursor: 'ew-resize',
                          background: isTrimmingRight ? C.accent : 'rgba(124,58,237,0.6)',
                          borderLeft: `1px solid ${isTrimmingRight ? C.accent : 'rgba(255,255,255,0.3)'}`,
                          zIndex: 4, touchAction: 'none',
                          transition: isTrimmingRight ? 'none' : 'background 120ms',
                        }}
                      >
                        <div style={{
                          position: 'absolute', left: '50%', top: '40%', transform: 'translate(-50%, 0)',
                          width: 1, height: '20%', background: 'rgba(255,255,255,0.8)',
                        }} />
                      </div>
                    )}
                  </div>

                  {/* 转场圆点 */}
                  {idx < clips.length - 1 && (
                    <div
                      onClick={() => setActiveIdx(idx)}
                      title={`转场: ${TRANS.find(t => t.key === clip.transition)?.label || '无'}`}
                      style={{
                        width: 24, height: 24, borderRadius: '50%',
                        margin: '0 4px', flexShrink: 0,
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        fontSize: 10,
                        background: tc + '30', color: tc,
                        border: `1px solid ${tc}50`, cursor: 'pointer',
                      }}
                    >{clip.transition === 'none' ? '·' : TRANS.find(t => t.key === clip.transition)?.emoji || '·'}</div>
                  )}
                </div>
              )
            })}
            {timelineDragOver && (
              <div style={{
                width: 80, height: 56, borderRadius: 8,
                border: `2px dashed ${C.primary}`,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                color: C.primary, fontSize: 11,
                marginLeft: 8, flexShrink: 0,
              }}>+ 放置</div>
            )}
          </div>

          {/* ----- 音频轨片段行 ----- */}
          <div style={{ display: 'flex', gap: 0, alignItems: 'center', minWidth: tlWidth, marginTop: 4 }}>
            {clips.map((clip, idx) => {
              const dur = clip.trimEnd - clip.trimStart
              const w = clipPxWidth(dur)
              const hasAudioClip = !!clip.audioUrl
              const wf = hasAudioClip ? waveformPaths[clip.id] : null
              // v0.42.7: 音频静音/低音量时,卡片变暗
              const audioMuted = clip.audioMuted ?? false
              const audioVolume = clip.audioVolume ?? 1.0
              const cardOpacity = audioMuted ? 0.4 : Math.max(0.5, audioVolume)

              return (
                <div key={`audio-${clip.id}`} style={{ display: 'flex', alignItems: 'center' }}>
                  {hasAudioClip && wf ? (
                    /* === 有音频: 差异化波形 === */
                    <div
                      onClick={() => setActiveIdx(idx)}
                      title={`音轨: ${clip.label}${clip.audioDuration ? ' · ' + clip.audioDuration : ''}${audioMuted ? ' · 已静音' : ' · 音量 ' + Math.round(audioVolume * 100) + '%'}`}
                      style={{
                        width: w, height: 40, borderRadius: 6,
                        background: 'linear-gradient(135deg, #10B98140 0%, #10B98120 100%)',
                        border: `1px solid ${idx === activeIdx ? '#10B981' : '#10B98160'}`,
                        boxSizing: 'border-box',
                        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                        padding: '0 8px', cursor: 'pointer', overflow: 'hidden', position: 'relative',
                        opacity: cardOpacity,
                        transition: 'opacity 150ms',
                      }}
                    >
                      <svg
                        width="100%" height="22"
                        style={{ position: 'absolute', left: 0, top: '50%', transform: 'translateY(-50%)', pointerEvents: 'none' }}
                        preserveAspectRatio="none" viewBox="0 0 100 20"
                      >
                        <path d={wf.topD} stroke="#10B981" strokeWidth="1.2" fill="none" opacity="0.85" strokeLinecap="round" strokeLinejoin="round" />
                        <path d={wf.botD} stroke="#10B981" strokeWidth="1" fill="none" opacity="0.45" strokeLinecap="round" strokeLinejoin="round" />
                      </svg>
                      <span style={{ fontSize: 10, fontWeight: 600, color: '#10B981', zIndex: 1, textShadow: '0 1px 2px rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', gap: 3 }}>
                        {audioMuted ? '🔇' : '🎵'} 音轨
                      </span>
                      {clip.audioDuration && !audioMuted && (
                        <span style={{ fontSize: 9, color: '#10B981', fontFamily: 'monospace', zIndex: 1, textShadow: '0 1px 2px rgba(0,0,0,0.5)' }}>
                          {Math.round(audioVolume * 100)}%
                        </span>
                      )}
                    </div>
                  ) : (
                    /* === 无音频: 增强占位 === */
                    <div
                      title={clip.muted ? '此片段音轨已删除' : '此片段未分离音轨'}
                      style={{
                        width: w, height: 40, borderRadius: 6,
                        background: 'repeating-linear-gradient(45deg, rgba(255,255,255,0.02) 0px, rgba(255,255,255,0.02) 6px, rgba(255,255,255,0.05) 6px, rgba(255,255,255,0.05) 12px)',
                        border: '1px dashed rgba(16,185,129,0.18)',
                        boxSizing: 'border-box',
                        display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4,
                      }}
                    >
                      <span style={{ fontSize: 12, opacity: 0.35 }}>🔇</span>
                      <span style={{ fontSize: 9, color: 'rgba(255,255,255,0.32)', fontWeight: 500 }}>无音轨</span>
                    </div>
                  )}
                  {idx < clips.length - 1 && (
                    <div style={{ width: 24, height: 24, margin: '0 4px', flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }} />
                  )}
                </div>
              )
            })}
          </div>

          {/* ----- v0.42.8: 字幕轨(可编辑) ----- */}
          <VideoEditorSubtitleTrack
            clips={clips}
            segments={subtitleSegments}
            language={subtitleLanguage}
            onSegmentsChange={onSubtitleSegmentsChange}
            onEditSegment={onEditSubtitleSegment} onRequestTTS={onRequestTTS}
            activeIdx={activeIdx}
          />
        </div>
      </div>
    </div>
  )
}
