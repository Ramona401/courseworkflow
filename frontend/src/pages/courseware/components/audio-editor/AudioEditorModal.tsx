/**
 * AudioEditorModal.tsx — 课件音频剪辑器弹窗
 *
 * 功能：波形可视化 + 选区裁剪 + 预览试听 + 后端FFmpeg裁剪导出 + 上传为新资产
 *
 * 设计决策：
 *   - 波形用 Web Audio API 解码取样本数据，Canvas2D 绘制（轻量零依赖）
 *   - 裁剪走后端 FFmpeg -c copy 不重编码（保留原始格式，速度极快，无质量损失）
 *   - 不走纯前端 OfflineAudioContext 导出（那会丢失mp3/aac压缩变WAV文件暴增）
 *   - 弹窗模式对齐视频编辑器 VideoEditorModal（fixed全屏深色主题）
 *   - 单文件 <600行，不拆子组件（音频编辑器比视频简单得多）
 *
 * 路由：无独立路由，由 MediaManagerPanel / MediaAssetCards 按需弹出
 */
import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { trimCWAudio } from '@/api/coursewares.media'
import type { TrimAudioResponse } from '@/api/coursewares.media'
import type { CoursewareAsset } from '@/api/coursewares'
import { makeAsset } from '../courseware-workshop/makeAsset'

// ==================== 配色常量（青蓝深色主题，区别于视频编辑器紫蓝系） ====================
const AC = {
  bg: '#0F172A',          // 主背景（slate-900）
  bgCard: '#1E293B',      // 面板/卡片（slate-800）
  bgWave: '#0F172A',      // 波形区背景
  primary: '#06B6D4',     // 主色调（cyan-500）
  primaryLight: '#22D3EE', // hover态
  primaryDim: 'rgba(6,182,212,0.15)', // 选区半透明底
  text: '#E2E8F0',        // 主文字（slate-200）
  textMuted: '#94A3B8',   // 辅助文字（slate-400）
  waveform: '#34D399',    // 波形色（emerald-400，音频绿直觉）
  waveformDim: '#065F46', // 未选中区波形暗色
  playhead: '#EF4444',    // 播放头红线
  selBorder: '#06B6D4',   // 选区边框
  border: 'rgba(255,255,255,0.1)',
  danger: '#EF4444',
  success: '#22C55E',
} as const

// ==================== 波形参数 ====================
const WAVE_HEIGHT = 140         // 波形画布高度
const RULER_HEIGHT = 24         // 刻度尺高度
const HANDLE_W = 10             // 选区手柄宽度
const MIN_SELECT_SEC = 0.5      // 最短选区(秒)
const SAMPLES_PER_PX = 4        // 每像素取多少采样点取峰值
const DEFAULT_PX_PER_SEC = 80   // 默认缩放

// ==================== 工具函数 ====================
/** 格式化时间 mm:ss.d */
const fmt = (sec: number): string => {
  if (!isFinite(sec) || sec < 0) return '0:00.0'
  const m = Math.floor(sec / 60)
  const s = sec - m * 60
  return m + ':' + (s < 10 ? '0' : '') + s.toFixed(1)
}

/** 格式化时长为友好文本 */
const fmtDur = (sec: number): string => {
  if (sec < 60) return sec.toFixed(1) + '秒'
  const m = Math.floor(sec / 60)
  const s = Math.round(sec - m * 60)
  return m + '分' + (s > 0 ? s + '秒' : '')
}

// ==================== Props ====================
interface Props {
  /** 要剪辑的音频资产 */
  audio: CoursewareAsset
  /** 课件ID */
  coursewareId: string
  /** 当前页码 */
  pageNum: number
  /** 关闭弹窗 */
  onClose: () => void
  /** 裁剪导出成功回调，传回新资产 */
  onExported: (newAsset: CoursewareAsset) => void
}

export default function AudioEditorModal({ audio, coursewareId, pageNum, onClose, onExported }: Props) {
  // ==================== 状态 ====================
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [peaks, setPeaks] = useState<number[]>([])       // 归一化峰值数组[0,1]
  const [duration, setDuration] = useState(0)             // 总时长(秒)
  const [selStart, setSelStart] = useState<number | null>(null)  // 选区起点(秒)，null=未选
  const [selEnd, setSelEnd] = useState<number | null>(null)      // 选区终点(秒)
  const [currentTime, setCurrentTime] = useState(0)       // 播放位置(秒)
  const [playing, setPlaying] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [message, setMessage] = useState('')
  const [pxPerSec, setPxPerSec] = useState(DEFAULT_PX_PER_SEC) // 缩放级别

  // ==================== Refs ====================
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const audioElRef = useRef<HTMLAudioElement>(null)
  const animRef = useRef(0)
  const dragRef = useRef<'none' | 'start' | 'end' | 'body' | 'create'>('none')
  const dragStartXRef = useRef(0)
  const dragOrigStartRef = useRef(0)
  const dragOrigEndRef = useRef(0)

  // ==================== 派生值 ====================
  const hasSelection = selStart !== null && selEnd !== null && selEnd - selStart >= MIN_SELECT_SEC
  const selDuration = hasSelection ? (selEnd! - selStart!) : 0
  const totalWidth = Math.max(600, Math.ceil(duration * pxPerSec))

  // ==================== 加载音频 + 提取波形 ====================
  useEffect(() => {
    if (!audio.oss_url) { setError('音频文件地址为空'); setLoading(false); return }
    let cancelled = false
    const audioCtx = new AudioContext()

    const load = async () => {
      try {
        setLoading(true); setError('')
        // 1. 下载音频二进制
        const resp = await fetch(audio.oss_url)
        if (!resp.ok) throw new Error('音频加载失败(HTTP ' + resp.status + ')')
        const arrayBuf = await resp.arrayBuffer()
        if (cancelled) return

        // 2. Web Audio API 解码
        const audioBuffer = await audioCtx.decodeAudioData(arrayBuf)
        if (cancelled) return

        const dur = audioBuffer.duration
        setDuration(dur)

        // 3. 适配缩放使整首歌约铺满800px宽
        const fitPx = Math.max(20, Math.min(200, 800 / dur))
        setPxPerSec(fitPx)
        const width = Math.ceil(dur * fitPx)

        // 4. 提取峰值（取所有通道混合后的绝对峰值）
        const totalSamples = width * SAMPLES_PER_PX
        const channelCount = audioBuffer.numberOfChannels
        const peakArr: number[] = new Array(width).fill(0)

        for (let ch = 0; ch < channelCount; ch++) {
          const raw = audioBuffer.getChannelData(ch)
          const samplesPerPeak = Math.max(1, Math.floor(raw.length / totalSamples))
          for (let i = 0; i < width; i++) {
            let peak = 0
            const base = Math.floor((i * raw.length) / width)
            for (let j = 0; j < samplesPerPeak * SAMPLES_PER_PX; j++) {
              const idx = base + j
              if (idx < raw.length) peak = Math.max(peak, Math.abs(raw[idx]))
            }
            peakArr[i] = Math.max(peakArr[i], peak)
          }
        }
        if (cancelled) return
        setPeaks(peakArr)
        setLoading(false)
      } catch (e) {
        if (!cancelled) {
          setError('音频解析失败: ' + (e instanceof Error ? e.message : '未知错误'))
          setLoading(false)
        }
      }
    }
    load()
    return () => { cancelled = true; audioCtx.close().catch(() => {}) }
  }, [audio.oss_url])

  // ==================== Canvas绘制 ====================
  const drawWaveform = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || peaks.length === 0 || duration <= 0) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const w = totalWidth
    const h = WAVE_HEIGHT
    canvas.width = w
    canvas.height = h
    ctx.clearRect(0, 0, w, h)

    // 背景
    ctx.fillStyle = AC.bgWave
    ctx.fillRect(0, 0, w, h)

    // 选区底色
    if (hasSelection) {
      const sx = selStart! * pxPerSec
      const ex = selEnd! * pxPerSec
      ctx.fillStyle = AC.primaryDim
      ctx.fillRect(sx, 0, ex - sx, h)
    }

    // 波形条
    const barW = Math.max(1, (w / peaks.length) * 0.7)
    const gap = w / peaks.length
    const mid = h / 2
    for (let i = 0; i < peaks.length; i++) {
      const x = i * gap
      const amp = peaks[i] * (mid - 4)
      // 选区内外不同颜色
      const sec = (i / peaks.length) * duration
      const inSel = hasSelection && sec >= selStart! && sec <= selEnd!
      ctx.fillStyle = inSel ? AC.waveform : (hasSelection ? AC.waveformDim : AC.waveform)
      ctx.fillRect(x, mid - amp, barW, amp * 2 || 1)
    }

    // 选区边框
    if (hasSelection) {
      const sx = selStart! * pxPerSec
      const ex = selEnd! * pxPerSec
      ctx.strokeStyle = AC.selBorder
      ctx.lineWidth = 2
      ctx.strokeRect(sx, 0, ex - sx, h)

      // 手柄
      ctx.fillStyle = AC.primary
      ctx.fillRect(sx - HANDLE_W / 2, 0, HANDLE_W, h)
      ctx.fillRect(ex - HANDLE_W / 2, 0, HANDLE_W, h)
    }

    // 播放头
    const phX = currentTime * pxPerSec
    ctx.fillStyle = AC.playhead
    ctx.fillRect(phX - 1, 0, 2, h)
  }, [peaks, duration, pxPerSec, selStart, selEnd, hasSelection, currentTime, totalWidth])

  useEffect(() => { drawWaveform() }, [drawWaveform])

  // ==================== 播放控制 ====================
  const play = useCallback((from?: number) => {
    const el = audioElRef.current
    if (!el || duration <= 0) return
    const start = from !== undefined ? from : currentTime
    el.currentTime = start
    el.play().catch(() => {})
    setPlaying(true)

    const tick = () => {
      if (!audioElRef.current || audioElRef.current.paused) { setPlaying(false); return }
      const t = audioElRef.current.currentTime
      setCurrentTime(t)
      // 如果有选区且播放到选区末尾则停止
      if (hasSelection && t >= selEnd!) {
        audioElRef.current.pause()
        audioElRef.current.currentTime = selEnd!
        setCurrentTime(selEnd!)
        setPlaying(false)
        return
      }
      animRef.current = requestAnimationFrame(tick)
    }
    animRef.current = requestAnimationFrame(tick)
  }, [currentTime, duration, hasSelection, selEnd])

  const pause = useCallback(() => {
    audioElRef.current?.pause()
    cancelAnimationFrame(animRef.current)
    setPlaying(false)
  }, [])

  const stop = useCallback(() => {
    pause()
    setCurrentTime(selStart ?? 0)
  }, [pause, selStart])

  // 播放选区预览
  const playSelection = useCallback(() => {
    if (!hasSelection) return
    play(selStart!)
  }, [hasSelection, selStart, play])

  // 组件卸载时停止播放
  useEffect(() => () => { cancelAnimationFrame(animRef.current); audioElRef.current?.pause() }, [])

  // ==================== 波形交互（鼠标拖拽创建/调整选区） ====================
  const secFromMouseX = useCallback((e: React.MouseEvent | MouseEvent): number => {
    const canvas = canvasRef.current
    if (!canvas) return 0
    const rect = canvas.getBoundingClientRect()
    const scrollLeft = scrollRef.current?.scrollLeft || 0
    const x = e.clientX - rect.left + scrollLeft
    return Math.max(0, Math.min(duration, x / pxPerSec))
  }, [duration, pxPerSec])

  const handleCanvasMouseDown = useCallback((e: React.MouseEvent) => {
    if (loading || duration <= 0) return
    e.preventDefault()
    const sec = secFromMouseX(e)

    // 判断是否点在手柄上
    if (hasSelection) {
      const sx = selStart! * pxPerSec
      const ex = selEnd! * pxPerSec
      const scrollLeft = scrollRef.current?.scrollLeft || 0
      const rect = canvasRef.current!.getBoundingClientRect()
      const mouseX = e.clientX - rect.left + scrollLeft

      if (Math.abs(mouseX - sx) <= HANDLE_W) {
        dragRef.current = 'start'; dragStartXRef.current = sec; return
      }
      if (Math.abs(mouseX - ex) <= HANDLE_W) {
        dragRef.current = 'end'; dragStartXRef.current = sec; return
      }
      // 点在选区内部——拖拽整体平移
      if (sec >= selStart! && sec <= selEnd!) {
        dragRef.current = 'body'
        dragStartXRef.current = sec
        dragOrigStartRef.current = selStart!
        dragOrigEndRef.current = selEnd!
        return
      }
    }
    // 点在空白区域——创建新选区
    dragRef.current = 'create'
    dragStartXRef.current = sec
    setSelStart(sec)
    setSelEnd(sec)
  }, [loading, duration, hasSelection, selStart, selEnd, pxPerSec, secFromMouseX])

  useEffect(() => {
    const onMove = (e: MouseEvent) => {
      if (dragRef.current === 'none') return
      const sec = secFromMouseX(e)

      if (dragRef.current === 'create') {
        const s = Math.min(dragStartXRef.current, sec)
        const ed = Math.max(dragStartXRef.current, sec)
        setSelStart(s)
        setSelEnd(ed)
      } else if (dragRef.current === 'start') {
        setSelStart(Math.max(0, Math.min(sec, (selEnd ?? duration) - MIN_SELECT_SEC)))
      } else if (dragRef.current === 'end') {
        setSelEnd(Math.min(duration, Math.max(sec, (selStart ?? 0) + MIN_SELECT_SEC)))
      } else if (dragRef.current === 'body') {
        const delta = sec - dragStartXRef.current
        let ns = dragOrigStartRef.current + delta
        let ne = dragOrigEndRef.current + delta
        if (ns < 0) { ne -= ns; ns = 0 }
        if (ne > duration) { ns -= (ne - duration); ne = duration }
        setSelStart(Math.max(0, ns))
        setSelEnd(Math.min(duration, ne))
      }
    }
    const onUp = () => { dragRef.current = 'none' }
    document.addEventListener('mousemove', onMove)
    document.addEventListener('mouseup', onUp)
    return () => { document.removeEventListener('mousemove', onMove); document.removeEventListener('mouseup', onUp) }
  }, [secFromMouseX, duration, selStart, selEnd])

  // 点击波形跳转播放头
  const handleCanvasClick = useCallback((e: React.MouseEvent) => {
    if (dragRef.current !== 'none') return // 拖拽结束不处理click
    const sec = secFromMouseX(e)
    setCurrentTime(sec)
    if (audioElRef.current) audioElRef.current.currentTime = sec
  }, [secFromMouseX])

  // ==================== 裁剪导出 ====================
  const handleExport = useCallback(async () => {
    if (!hasSelection || exporting) return
    setExporting(true); setMessage('')
    try {
      const resp: TrimAudioResponse = await trimCWAudio(coursewareId, audio.id, selStart!, selEnd!)
      setMessage('✅ ' + resp.message)
      // 构造新资产插入列表
      const newAsset = makeAsset(coursewareId, {
        id: resp.asset_id,
        oss_url: resp.url,
        asset_type: 'audio',
        generation_prompt: '✂️ 裁剪 ' + fmt(selStart!) + '-' + fmt(selEnd!),
        file_size: resp.file_size,
        mime_type: resp.mime_type,
      })
      onExported(newAsset)
      // 1.5秒后关闭弹窗让用户看到成功消息
      setTimeout(() => onClose(), 1500)
    } catch (e) {
      setMessage('❌ 裁剪失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally {
      setExporting(false)
    }
  }, [hasSelection, exporting, coursewareId, audio.id, selStart, selEnd, onExported, onClose])

  // ==================== 缩放控制 ====================
  const zoomIn = () => setPxPerSec(prev => Math.min(500, prev * 1.5))
  const zoomOut = () => setPxPerSec(prev => Math.max(5, prev / 1.5))
  const zoomFit = () => { if (duration > 0) setPxPerSec(Math.max(20, Math.min(200, 800 / duration))) }

  // ==================== 快捷操作 ====================
  const selectAll = () => { setSelStart(0); setSelEnd(duration) }
  const clearSelection = () => { setSelStart(null); setSelEnd(null) }

  // 键盘快捷键
  useEffect(() => {
    const fn = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { e.preventDefault(); if (!exporting) onClose() }
      if (e.key === ' ') { e.preventDefault(); playing ? pause() : (hasSelection ? playSelection() : play()) }
    }
    window.addEventListener('keydown', fn)
    return () => window.removeEventListener('keydown', fn)
  }, [onClose, exporting, playing, pause, play, playSelection, hasSelection])

  // ==================== 刻度尺 ====================
  const ticks = useMemo(() => {
    if (duration <= 0) return []
    // 根据缩放级别选择刻度间距
    const intervals = [0.1, 0.25, 0.5, 1, 2, 5, 10, 15, 30, 60]
    let interval = 1
    for (const iv of intervals) {
      if (iv * pxPerSec >= 40) { interval = iv; break }
    }
    const arr: { sec: number; label: string }[] = []
    for (let t = 0; t <= duration; t += interval) {
      arr.push({ sec: t, label: fmt(t) })
    }
    return arr
  }, [duration, pxPerSec])

  // ==================== 文件信息 ====================
  const fileName = audio.oss_url.split('/').pop()?.slice(0, 40) || '音频'
  const fileInfo = [
    audio.file_size > 0 ? (audio.file_size / (1024 * 1024)).toFixed(1) + ' MB' : '',
    audio.mime_type ? audio.mime_type.split('/')[1]?.toUpperCase() : '',
    duration > 0 ? fmtDur(duration) : '',
  ].filter(Boolean).join(' · ')

  // ==================== 渲染 ====================
  const btnStyle = (active?: boolean, disabled?: boolean): React.CSSProperties => ({
    padding: '6px 14px', borderRadius: 6, border: 'none', fontSize: 12, fontWeight: 500,
    cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.4 : 1,
    background: active ? AC.primary : '#334155', color: active ? '#fff' : AC.text,
    display: 'inline-flex', alignItems: 'center', gap: 4, whiteSpace: 'nowrap',
    transition: 'background 0.15s',
  })

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 99994, background: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center', backdropFilter: 'blur(4px)' }}>
      <div style={{ background: AC.bg, borderRadius: 16, width: '92vw', maxWidth: 1000, maxHeight: '88vh', display: 'flex', flexDirection: 'column', overflow: 'hidden', boxShadow: '0 25px 50px rgba(0,0,0,0.5)', border: '1px solid ' + AC.border }}>

        {/* ===== 头部 ===== */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '14px 20px', borderBottom: '1px solid ' + AC.border, background: AC.bgCard }}>
          <div>
            <div style={{ fontSize: 15, fontWeight: 600, color: AC.text }}>✂️ 音频剪辑器</div>
            <div style={{ fontSize: 11, color: AC.textMuted, marginTop: 2 }}>{fileName} · {fileInfo}</div>
          </div>
          <button onClick={onClose} disabled={exporting} style={{ ...btnStyle(), background: 'transparent', fontSize: 18, padding: '4px 10px' }}>✕</button>
        </div>

        {/* ===== 工具栏 ===== */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '10px 20px', borderBottom: '1px solid ' + AC.border, background: AC.bgCard, flexWrap: 'wrap' }}>
          <button onClick={() => playing ? pause() : (hasSelection ? playSelection() : play())} style={btnStyle(playing)} disabled={loading || duration <= 0}>
            {playing ? '⏸ 暂停' : '▶ 播放'}
          </button>
          <button onClick={stop} style={btnStyle()} disabled={loading}>⏹ 停止</button>
          <span style={{ width: 1, height: 20, background: AC.border, margin: '0 4px' }} />
          <button onClick={selectAll} style={btnStyle()} disabled={loading}>全选</button>
          <button onClick={clearSelection} style={btnStyle()} disabled={!hasSelection}>清除选区</button>
          <span style={{ width: 1, height: 20, background: AC.border, margin: '0 4px' }} />
          <button onClick={zoomIn} style={btnStyle()}>🔍+</button>
          <button onClick={zoomOut} style={btnStyle()}>🔍−</button>
          <button onClick={zoomFit} style={btnStyle()}>适配</button>
          <span style={{ flex: 1 }} />
          <span style={{ fontSize: 12, color: AC.textMuted }}>{fmt(currentTime)} / {fmt(duration)}</span>
        </div>

        {/* ===== 波形区域 ===== */}
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column', minHeight: 200 }}>
          {loading ? (
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: AC.textMuted, fontSize: 14 }}>
              ⏳ 加载音频波形中...
            </div>
          ) : error ? (
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: AC.danger, fontSize: 14, padding: 20, textAlign: 'center' }}>
              ❌ {error}
            </div>
          ) : (
            <>
              {/* 操作提示 */}
              <div style={{ padding: '8px 20px', fontSize: 11, color: AC.textMuted, background: AC.bgCard, borderBottom: '1px solid ' + AC.border }}>
                💡 在波形上拖拽选择要保留的片段 → 点「▶ 播放」试听选区 → 点「✂️ 裁剪导出」生成新音频
              </div>

              {/* 刻度尺 */}
              <div ref={scrollRef} style={{ overflowX: 'auto', overflowY: 'hidden', flex: 1 }}>
                <div style={{ width: totalWidth, position: 'relative' }}>
                  {/* 时间刻度 */}
                  <div style={{ height: RULER_HEIGHT, position: 'relative', background: '#1E293B', borderBottom: '1px solid ' + AC.border }}>
                    {ticks.map((t, i) => (
                      <div key={i} style={{ position: 'absolute', left: t.sec * pxPerSec, top: 0, height: RULER_HEIGHT, borderLeft: '1px solid rgba(255,255,255,0.15)' }}>
                        <span style={{ position: 'absolute', left: 4, top: 4, fontSize: 9, color: AC.textMuted, whiteSpace: 'nowrap' }}>{t.label}</span>
                      </div>
                    ))}
                  </div>

                  {/* 波形画布 */}
                  <canvas
                    ref={canvasRef}
                    width={totalWidth}
                    height={WAVE_HEIGHT}
                    style={{ display: 'block', cursor: loading ? 'default' : 'crosshair' }}
                    onMouseDown={handleCanvasMouseDown}
                    onClick={handleCanvasClick}
                  />
                </div>
              </div>
            </>
          )}
        </div>

        {/* ===== 选区信息 + 导出 ===== */}
        <div style={{ padding: '12px 20px', borderTop: '1px solid ' + AC.border, background: AC.bgCard, display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
          {hasSelection ? (
            <>
              <div style={{ fontSize: 13, color: AC.primary, fontWeight: 600 }}>
                ✂️ 已选区间: {fmt(selStart!)} → {fmt(selEnd!)}（{fmtDur(selDuration)}）
              </div>
              <span style={{ flex: 1 }} />
              <button onClick={playSelection} style={btnStyle()} disabled={exporting}>
                ▶ 试听选区
              </button>
              <button
                onClick={handleExport}
                disabled={exporting || !hasSelection}
                style={{
                  ...btnStyle(true, exporting),
                  background: exporting ? '#475569' : 'linear-gradient(135deg, #06B6D4, #0891B2)',
                  padding: '8px 20px', fontSize: 13, fontWeight: 600,
                }}
              >
                {exporting ? '⏳ 裁剪中...' : '✂️ 裁剪导出'}
              </button>
            </>
          ) : (
            <div style={{ fontSize: 13, color: AC.textMuted }}>
              👆 在上方波形上拖拽鼠标选择要保留的音频片段{duration > 0 ? `（总时长 ${fmtDur(duration)}）` : ''}
            </div>
          )}
        </div>

        {/* 消息条 */}
        {message && (
          <div style={{ padding: '10px 20px', background: message.startsWith('❌') ? 'rgba(239,68,68,0.15)' : 'rgba(34,197,94,0.15)', color: message.startsWith('❌') ? '#FCA5A5' : '#86EFAC', fontSize: 13, borderTop: '1px solid ' + AC.border }}>
            {message}
          </div>
        )}
      </div>

      {/* 隐藏的音频播放元素（试听用） */}
      <audio ref={audioElRef} src={audio.oss_url} preload="metadata" style={{ display: 'none' }} />
    </div>
  )
}
