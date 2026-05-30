/**
 * VideoEditorSubtitleTrack.tsx — 时间轴字幕轨可编辑组件
 *
 * v0.42.8 新增：替换原有的字幕轨占位行
 * v0.42.8.1 性能修复：拖拽字幕条改为 controlled draft 模式
 *   - 拖动期间只更新本地 draftSeg state（零全链路重渲染）
 *   - pointerup 时一次性调 onSegmentsChange 提交最终值
 *
 * 功能：
 *   - 在时间轴第三轨渲染字幕条（按 startSec/endSec 定位）
 *   - 点击空白区域新增字幕条（默认3秒）
 *   - 双击字幕条打开编辑弹窗
 *   - 拖拽字幕条左右边界调整时长（controlled draft）
 *   - 拖拽字幕条中间调整位置（controlled draft）
 *   - 删除字幕条
 */
import { useRef, useState, useEffect } from 'react'
import { PX_PER_SEC } from './VideoEditorConstants'
import { clipPxWidth } from './VideoEditorUtils'
import type { EditorClip } from './VideoEditorTypes'

// ==================== 字幕片段类型（前端使用） ====================
export interface SubtitleSegment {
  id: string
  start_sec: number
  end_sec: number
  text: string
  language: string
  tts_audio_url?: string
  tts_voice?: string
  tts_duration?: number
  tts_generated_at?: string
}

interface SubtitleTrackProps {
  /** 视频片段列表（用于计算总时长和像素宽度） */
  clips: EditorClip[]
  /** 当前语言的字幕片段列表 */
  segments: SubtitleSegment[]
  /** 当前选中的语言 */
  language: string
  /** 字幕变更回调（父组件负责持久化） */
  onSegmentsChange: (segs: SubtitleSegment[]) => void
  /** 双击字幕条触发编辑弹窗 */
  onEditSegment: (seg: SubtitleSegment) => void
  /** 请求打开 TTS 配音弹窗 */
  onRequestTTS?: () => void
  /** 当前激活片段索引 */
  activeIdx: number
}

/** 生成简易UUID */
function makeId(): string {
  return 'sub_' + Date.now().toString(36) + '_' + Math.random().toString(36).slice(2, 8)
}

export default function VideoEditorSubtitleTrack(props: SubtitleTrackProps) {
  const { clips, segments, language, onSegmentsChange, onEditSegment, onRequestTTS } = props

  // 总时长和总像素宽度
  const totalDur = clips.reduce((s, c) => s + (c.trimEnd - c.trimStart), 0)
  const totalPx = clips.reduce((s, c) => s + clipPxWidth(c.trimEnd - c.trimStart), 0)
  // 像素→秒系数（考虑 clamp 后的实际比例）
  const pxToSec = totalDur > 0 && totalPx > 0 ? totalDur / totalPx : 1 / PX_PER_SEC
  const secToPx = totalPx > 0 && totalDur > 0 ? totalPx / totalDur : PX_PER_SEC

  // ==================== controlled draft 拖拽状态 ====================
  // 拖动期间不触发 onSegmentsChange，只更新本地 draft
  // pointerup 时一次性提交，避免每像素重渲染整个时间轴
  const [dragState, setDragState] = useState<{
    segId: string
    type: 'move' | 'left' | 'right'
    startX: number
    origStart: number
    origEnd: number
  } | null>(null)
  const dragRef = useRef(dragState)
  useEffect(() => { dragRef.current = dragState }, [dragState])

  // 拖动期间的实时值（仅用于视觉反馈，不回写 segments）
  const [draftSeg, setDraftSeg] = useState<{
    segId: string
    start_sec: number
    end_sec: number
  } | null>(null)
  const draftSegRef = useRef(draftSeg)
  useEffect(() => { draftSegRef.current = draftSeg }, [draftSeg])

  useEffect(() => {
    if (!dragState) return
    const onMove = (e: PointerEvent) => {
      const ds = dragRef.current
      if (!ds) return
      const dx = e.clientX - ds.startX
      const dt = dx * pxToSec
      const minLen = 0.5 // 最短0.5秒

      let newStart = ds.origStart
      let newEnd = ds.origEnd

      if (ds.type === 'move') {
        const dur = ds.origEnd - ds.origStart
        newStart = Math.max(0, Math.min(totalDur - dur, ds.origStart + dt))
        newEnd = newStart + dur
      } else if (ds.type === 'left') {
        newStart = Math.max(0, Math.min(ds.origEnd - minLen, ds.origStart + dt))
      } else {
        newEnd = Math.max(ds.origStart + minLen, Math.min(totalDur, ds.origEnd + dt))
      }

      // 只更新本地 draft（不触发 segments 重渲染）
      setDraftSeg({ segId: ds.segId, start_sec: newStart, end_sec: newEnd })
    }
    const onUp = () => {
      // pointerup 时一次性提交到 segments
      const draft = draftSegRef.current
      const ds = dragRef.current
      if (draft && ds) {
        onSegmentsChange(segments.map(s =>
          s.id === draft.segId
            ? { ...s, start_sec: draft.start_sec, end_sec: draft.end_sec }
            : s
        ))
      }
      setDraftSeg(null)
      setDragState(null)
    }
    document.addEventListener('pointermove', onMove)
    document.addEventListener('pointerup', onUp)
    return () => {
      document.removeEventListener('pointermove', onMove)
      document.removeEventListener('pointerup', onUp)
    }
  }, [dragState, segments, onSegmentsChange, totalDur, pxToSec])

  // ==================== 点击空白添加字幕 ====================
  const trackRef = useRef<HTMLDivElement>(null)
  const handleTrackClick = (e: React.MouseEvent) => {
    // 只在直接点击轨道背景时触发，不在字幕条上触发
    if (e.target !== trackRef.current) return
    const rect = trackRef.current!.getBoundingClientRect()
    const clickX = e.clientX - rect.left
    const clickSec = clickX * pxToSec
    // 新字幕默认3秒
    const startSec = Math.max(0, clickSec - 1.5)
    const endSec = Math.min(totalDur, startSec + 3)
    const newSeg: SubtitleSegment = {
      id: makeId(),
      start_sec: Math.round(startSec * 100) / 100,
      end_sec: Math.round(endSec * 100) / 100,
      text: '',
      language,
    }
    onSegmentsChange([...segments, newSeg])
    // 立即打开编辑弹窗
    onEditSegment(newSeg)
  }

  // ==================== 删除字幕条 ====================
  const handleDeleteSeg = (segId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    onSegmentsChange(segments.filter(s => s.id !== segId))
  }

  // 空状态
  if (clips.length === 0) {
    return (
      <div style={{
        height: 40, borderRadius: 6,
        background: 'rgba(245,158,11,0.04)',
        border: '1px dashed rgba(245,158,11,0.2)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        marginTop: 4,
      }}>
        <span style={{ fontSize: 11, color: 'rgba(245,158,11,0.5)' }}>
          💬 添加视频片段后可编辑字幕
        </span>
      </div>
    )
  }

  return (
    <div
      ref={trackRef}
      onClick={handleTrackClick}
      style={{
        position: 'relative',
        height: 40, minWidth: totalPx,
        marginTop: 4, borderRadius: 6,
        background: 'rgba(245,158,11,0.06)',
        border: '1px solid rgba(245,158,11,0.2)',
        cursor: 'text',
        boxSizing: 'border-box',
        overflow: 'hidden',
      }}
    >
      {/* 渲染每条字幕 */}
      {segments.map(seg => {
        // 拖动期间用 draftSeg 实时值（视觉反馈），否则用 segments 的值
        const isDragging = draftSeg?.segId === seg.id
        const effectiveStart = isDragging ? draftSeg!.start_sec : seg.start_sec
        const effectiveEnd = isDragging ? draftSeg!.end_sec : seg.end_sec

        const left = effectiveStart * secToPx
        const width = Math.max(20, (effectiveEnd - effectiveStart) * secToPx)
        const hasTTS = !!seg.tts_audio_url

        return (
          <div
            key={seg.id}
            onDoubleClick={(e) => { e.stopPropagation(); onEditSegment(seg) }}
            style={{
              position: 'absolute',
              left, width, top: 3, bottom: 3,
              borderRadius: 4,
              background: isDragging
                ? 'linear-gradient(135deg, rgba(245,158,11,0.45) 0%, rgba(245,158,11,0.3) 100%)'
                : hasTTS
                  ? 'linear-gradient(135deg, rgba(245,158,11,0.35) 0%, rgba(245,158,11,0.2) 100%)'
                  : 'linear-gradient(135deg, rgba(245,158,11,0.25) 0%, rgba(245,158,11,0.12) 100%)',
              border: `1px solid ${isDragging ? 'rgba(245,158,11,0.8)' : 'rgba(245,158,11,0.5)'}`,
              cursor: dragState ? 'grabbing' : 'grab',
              overflow: 'hidden',
              display: 'flex', alignItems: 'center',
              padding: '0 4px',
              boxSizing: 'border-box',
              userSelect: 'none',
              transition: isDragging ? 'none' : 'left 60ms, width 60ms',
            }}
          >
            {/* 左手柄 */}
            <div
              onPointerDown={e => {
                e.preventDefault(); e.stopPropagation()
                setDragState({ segId: seg.id, type: 'left', startX: e.clientX, origStart: seg.start_sec, origEnd: seg.end_sec })
              }}
              style={{
                position: 'absolute', left: 0, top: 0, width: 5, height: '100%',
                cursor: 'ew-resize', background: 'rgba(245,158,11,0.6)',
                borderRight: '1px solid rgba(245,158,11,0.3)',
              }}
            />
            {/* 中间拖拽区(移动位置) */}
            <div
              onPointerDown={e => {
                e.preventDefault(); e.stopPropagation()
                setDragState({ segId: seg.id, type: 'move', startX: e.clientX, origStart: seg.start_sec, origEnd: seg.end_sec })
              }}
              style={{
                flex: 1, height: '100%',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                overflow: 'hidden', cursor: dragState ? 'grabbing' : 'grab',
                padding: '0 6px',
              }}
            >
              <span style={{
                fontSize: 10, color: '#F59E0B',
                whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
                maxWidth: '100%', textShadow: '0 1px 2px rgba(0,0,0,0.5)',
                fontWeight: 500,
              }}>
                {seg.text || '(空字幕)'}
              </span>
            </div>
            {/* 右手柄 */}
            <div
              onPointerDown={e => {
                e.preventDefault(); e.stopPropagation()
                setDragState({ segId: seg.id, type: 'right', startX: e.clientX, origStart: seg.start_sec, origEnd: seg.end_sec })
              }}
              style={{
                position: 'absolute', right: 0, top: 0, width: 5, height: '100%',
                cursor: 'ew-resize', background: 'rgba(245,158,11,0.6)',
                borderLeft: '1px solid rgba(245,158,11,0.3)',
              }}
            />
            {/* 删除按钮 */}
            <button
              onClick={e => handleDeleteSeg(seg.id, e)}
              style={{
                position: 'absolute', top: 1, right: 6, zIndex: 2,
                background: 'rgba(0,0,0,0.5)', border: 'none',
                color: '#FCA5A5', fontSize: 9, cursor: 'pointer',
                padding: '0 3px', lineHeight: '14px', borderRadius: 2,
                opacity: 0.6,
              }}
              onMouseEnter={e => (e.currentTarget.style.opacity = '1')}
              onMouseLeave={e => (e.currentTarget.style.opacity = '0.6')}
            >✕</button>
            {/* TTS 标记 */}
            {hasTTS && (
              <span style={{
                position: 'absolute', top: 1, left: 7, fontSize: 8,
                color: '#F59E0B', textShadow: '0 1px 2px rgba(0,0,0,0.5)',
              }}>🎵</span>
            )}
          </div>
        )
      })}

      {/* TTS 配音入口按钮（字幕数 > 0 时显示） */}
      {segments.length > 0 && onRequestTTS && (
        <button
          onClick={(e) => { e.stopPropagation(); onRequestTTS() }}
          style={{
            position: 'absolute', top: 2, right: 4, zIndex: 3,
            background: 'rgba(124,58,237,0.3)', border: '1px solid rgba(124,58,237,0.5)',
            color: '#C4B5FD', fontSize: 10, cursor: 'pointer',
            padding: '2px 8px', borderRadius: 4, fontWeight: 600,
          }}
          onMouseEnter={e => { e.currentTarget.style.background = 'rgba(124,58,237,0.5)' }}
          onMouseLeave={e => { e.currentTarget.style.background = 'rgba(124,58,237,0.3)' }}
          title="批量 TTS 配音"
        >🎙️ TTS</button>
      )}
      {/* 无字幕时的提示 */}
      {segments.length === 0 && (
        <div style={{
          position: 'absolute', inset: 0,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          pointerEvents: 'none',
        }}>
          <span style={{ fontSize: 11, color: 'rgba(245,158,11,0.5)', fontStyle: 'italic' }}>
            💬 点击添加字幕 · 双击编辑
          </span>
        </div>
      )}
    </div>
  )
}
