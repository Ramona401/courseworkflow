/**
 * VideoEditorSubtitleTrack.tsx — 时间轴字幕轨可编辑组件
 *
 * v0.42.8 新增：替换原有的字幕轨占位行
 * v0.42.8.1 性能修复：拖拽字幕条改为 controlled draft 模式
 * S-V3a 专业化改造（对标专业剪辑软件交互习惯）：
 *   A1 防误加：单击空白=取消选中（不再新增！）；新增改为「双击空白处」或「+字幕」按钮
 *   A3 选中态：单击字幕条=选中（金色高亮描边），Delete/Backspace 删除选中条
 *      （已生成配音的条目删除前弹确认；输入框聚焦时按键不触发删除）
 *   A4 防重叠：拖拽移动/调边界时钳制在相邻字幕条之间，临近边界6像素自动吸附；
 *      双击新增时自动适配点击处的可用空隙，空间不足提示而非强行重叠
 *
 * 功能：
 *   - 在时间轴第三轨渲染字幕条（按 startSec/endSec 定位）
 *   - 双击空白区域 / 点「+字幕」按钮 新增字幕条（默认3秒，自动避让相邻条）
 *   - 单击字幕条选中，Delete 删除；双击字幕条打开编辑弹窗
 *   - 拖拽字幕条左右边界调整时长（controlled draft + 邻界钳制吸附）
 *   - 拖拽字幕条中间调整位置（controlled draft + 邻界钳制吸附）
 *   - 删除字幕条（✕按钮或Delete键，带配音的先确认）
 */
import { useRef, useState, useEffect, useCallback } from 'react'
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

  // ==================== S-V3a A3: 选中态 ====================
  const [selectedId, setSelectedId] = useState('')

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
      const SNAP = 6 * pxToSec // S-V3a A4: 临近边界6像素自动吸附

      // S-V3a A4: 计算相邻字幕条形成的左右边界（防重叠）
      let lb = 0
      let rb = totalDur
      for (const s of segments) {
        if (s.id === ds.segId) continue
        if (s.end_sec <= ds.origStart + 0.01 && s.end_sec > lb) lb = s.end_sec
        if (s.start_sec >= ds.origEnd - 0.01 && s.start_sec < rb) rb = s.start_sec
      }
      // 历史重叠数据兜底：边界倒挂时退回全轨范围，不阻塞拖拽
      if (lb > rb) { lb = 0; rb = totalDur }

      let newStart = ds.origStart
      let newEnd = ds.origEnd

      if (ds.type === 'move') {
        const dur = ds.origEnd - ds.origStart
        newStart = Math.max(lb, Math.min(rb - dur, ds.origStart + dt))
        // 吸附到相邻边界
        if (Math.abs(newStart - lb) < SNAP) newStart = lb
        if (Math.abs(newStart + dur - rb) < SNAP) newStart = rb - dur
        newStart = Math.max(0, newStart)
        newEnd = newStart + dur
      } else if (ds.type === 'left') {
        newStart = Math.max(lb, Math.min(ds.origEnd - minLen, ds.origStart + dt))
        if (Math.abs(newStart - lb) < SNAP) newStart = lb
        newStart = Math.max(0, newStart)
      } else {
        newEnd = Math.max(ds.origStart + minLen, Math.min(rb, ds.origEnd + dt))
        if (Math.abs(newEnd - rb) < SNAP) newEnd = rb
        newEnd = Math.min(totalDur, newEnd)
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
            ? { ...s, start_sec: Math.round(draft.start_sec * 100) / 100, end_sec: Math.round(draft.end_sec * 100) / 100 }
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

  // ==================== S-V3a A3: 删除（统一入口，带配音先确认） ====================
  const deleteSeg = useCallback((segId: string) => {
    const seg = segments.find(s => s.id === segId)
    if (!seg) return
    if (seg.tts_audio_url && !window.confirm('该字幕已生成配音，删除后配音关联将丢失。确定删除？')) return
    onSegmentsChange(segments.filter(s => s.id !== segId))
    setSelectedId(prev => (prev === segId ? '' : prev))
  }, [segments, onSegmentsChange])

  // Delete/Backspace 删除选中条（输入框/文本域/可编辑区聚焦时不触发）
  useEffect(() => {
    const fn = (e: KeyboardEvent) => {
      if (e.key !== 'Delete' && e.key !== 'Backspace') return
      const t = e.target as HTMLElement | null
      if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable)) return
      if (!selectedId) return
      e.preventDefault()
      deleteSeg(selectedId)
    }
    window.addEventListener('keydown', fn)
    return () => window.removeEventListener('keydown', fn)
  }, [selectedId, deleteSeg])

  // ==================== S-V3a A1: 单击空白=取消选中；双击空白=新增 ====================
  const trackRef = useRef<HTMLDivElement>(null)

  // 单击轨道背景：只取消选中，绝不新增（专业剪辑软件习惯）
  const handleTrackClick = (e: React.MouseEvent) => {
    if (e.target !== trackRef.current) return
    setSelectedId('')
  }

  // 双击轨道背景：在点击处新增字幕（A4: 自动适配可用空隙，不与相邻条重叠）
  const handleTrackDoubleClick = (e: React.MouseEvent) => {
    if (e.target !== trackRef.current) return
    const rect = trackRef.current!.getBoundingClientRect()
    const clickX = e.clientX - rect.left
    const clickSec = clickX * pxToSec

    // 求点击处的可用空隙（左界=左侧最近字幕的end，右界=右侧最近字幕的start）
    let lb = 0
    let rb = totalDur
    for (const s of segments) {
      if (s.end_sec <= clickSec && s.end_sec > lb) lb = s.end_sec
      if (s.start_sec >= clickSec && s.start_sec < rb) rb = s.start_sec
    }
    if (rb - lb < 0.5) { alert('此处空间不足（不足0.5秒），请换个位置双击或先调整相邻字幕') ; return }

    let startSec = Math.max(lb, clickSec - 1.5)
    let endSec = Math.min(rb, startSec + 3)
    if (endSec - startSec < 0.5) startSec = Math.max(lb, endSec - 0.5)

    const newSeg: SubtitleSegment = {
      id: makeId(),
      start_sec: Math.round(startSec * 100) / 100,
      end_sec: Math.round(endSec * 100) / 100,
      text: '',
      language,
    }
    onSegmentsChange([...segments, newSeg])
    setSelectedId(newSeg.id)
    // 立即打开编辑弹窗（取消且未填文本时由父组件自动清理空条）
    onEditSegment(newSeg)
  }

  // 「+字幕」按钮：追加到最后一条字幕之后（无字幕时从0开始）
  const handleAddAtEnd = (e: React.MouseEvent) => {
    e.stopPropagation()
    const lastEnd = segments.reduce((m, s) => Math.max(m, s.end_sec), 0)
    if (totalDur - lastEnd < 0.5) { alert('时间轴尾部空间不足（不足0.5秒），请双击轨道空白处插入') ; return }
    const startSec = lastEnd
    const endSec = Math.min(totalDur, startSec + 3)
    const newSeg: SubtitleSegment = {
      id: makeId(),
      start_sec: Math.round(startSec * 100) / 100,
      end_sec: Math.round(endSec * 100) / 100,
      text: '',
      language,
    }
    onSegmentsChange([...segments, newSeg])
    setSelectedId(newSeg.id)
    onEditSegment(newSeg)
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
      onDoubleClick={handleTrackDoubleClick}
      style={{
        position: 'relative',
        height: 40, minWidth: totalPx,
        marginTop: 4, borderRadius: 6,
        background: 'rgba(245,158,11,0.06)',
        border: '1px solid rgba(245,158,11,0.2)',
        cursor: 'default',
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
        const isSelected = selectedId === seg.id

        return (
          <div
            key={seg.id}
            onClick={(e) => { e.stopPropagation(); setSelectedId(seg.id) }}
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
              // S-V3a A3: 选中态金色高亮描边（boxShadow不改变布局尺寸）
              boxShadow: isSelected ? '0 0 0 1.5px #F59E0B, 0 0 8px rgba(245,158,11,0.4)' : 'none',
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
                setSelectedId(seg.id)
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
                setSelectedId(seg.id)
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
                fontWeight: isSelected ? 700 : 500,
              }}>
                {seg.text || '(空字幕)'}
              </span>
            </div>
            {/* 右手柄 */}
            <div
              onPointerDown={e => {
                e.preventDefault(); e.stopPropagation()
                setSelectedId(seg.id)
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
              onClick={e => { e.stopPropagation(); deleteSeg(seg.id) }}
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

      {/* S-V3a A1: 「+字幕」按钮（常驻，新增的主入口之一） */}
      <button
        onClick={handleAddAtEnd}
        style={{
          position: 'absolute', top: 2, right: onRequestTTS && segments.length > 0 ? 58 : 4, zIndex: 3,
          background: 'rgba(245,158,11,0.2)', border: '1px solid rgba(245,158,11,0.5)',
          color: '#FCD34D', fontSize: 10, cursor: 'pointer',
          padding: '2px 8px', borderRadius: 4, fontWeight: 600,
        }}
        onMouseEnter={e => { e.currentTarget.style.background = 'rgba(245,158,11,0.35)' }}
        onMouseLeave={e => { e.currentTarget.style.background = 'rgba(245,158,11,0.2)' }}
        title="在最后一条字幕之后添加新字幕"
      >＋ 字幕</button>

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
            💬 双击空白添加字幕 · 或点右上「＋字幕」
          </span>
        </div>
      )}
    </div>
  )
}
