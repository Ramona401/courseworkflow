/**
 * VideoEditorModal.tsx — 类剪映多片段视频编辑器主组件 v8.0 (B1瘦身)
 *
 * v8.0 (B1瘦身) 重大改动:
 *   ✨ 从 1157行 瘦身到 ~580行
 *   ✨ 拆出 5 个子组件: Toolbar / DraftPanel / PlayerStage / ExitDialog / ProgressBar
 *   ✨ 主文件只保留: 状态定义 + 播放核心逻辑 + 片段管理 + 布局组装
 *
 * v7.0 (v0.42.7): 模块化拆分 + 音频独立混音 + 时间轴原地裁剪
 * v0.42.8: 字幕轨编辑 + 软字幕预览 + 持久化
 */
import { useState, useRef, useCallback, useEffect, useMemo } from 'react'
import { muteCWVideo, extractCWAudio, listVideoDrafts, saveVideoDraft, deleteVideoDraft, upsertSubtitle, listSubtitles } from '../../../api/coursewares'
import type { VideoDraftItem } from '../../../api/coursewares'
import type { EditorClip, VideoEditorModalProps } from './video-editor/VideoEditorTypes'
import { C, DRAG_TYPE_ASSET, SUBTITLE_LANGUAGES } from './video-editor/VideoEditorConstants'
import {
  fmtSize, urlToBlobUrl, totalPxWidth, pixelToTime,
} from './video-editor/VideoEditorUtils'
import VideoEditorMaterials from './video-editor/VideoEditorMaterials'
import VideoEditorPanel from './video-editor/VideoEditorPanel'
import VideoEditorTimeline from './video-editor/VideoEditorTimeline'
import VideoEditorSubtitleModal from './video-editor/VideoEditorSubtitleModal'
import VideoEditorToolbar from './video-editor/VideoEditorToolbar'
import VideoEditorDraftPanel from './video-editor/VideoEditorDraftPanel'
import VideoEditorPlayerStageComponent from './video-editor/VideoEditorPlayerStage'
import VideoEditorExitDialog from './video-editor/VideoEditorExitDialog'
import VideoEditorProgressBar from './video-editor/VideoEditorProgressBar'
import type { SubtitleSegment } from './video-editor/VideoEditorSubtitleTrack'
import VideoEditorTTSModal from './video-editor/VideoEditorTTSModal'

// ==================== 主组件 ====================
export default function VideoEditorModal({
  videos, coursewareId, onClose, onExport, exporting = false, onUploadVideo,
}: VideoEditorModalProps) {
  // === 片段状态 ===
  const [clips, setClips] = useState<EditorClip[]>([])
  const [activeIdx, setActiveIdx] = useState(-1)
  const [dragIdx, setDragIdx] = useState(-1)
  const [dragOverIdx, setDragOverIdx] = useState(-1)

  // === 播放器 ===
  const videoARef = useRef<HTMLVideoElement>(null)
  const videoBRef = useRef<HTMLVideoElement>(null)
  const audioRef = useRef<HTMLAudioElement>(null)
  const [topPlayer, setTopPlayer] = useState<'A' | 'B'>('A')
  const activeVideoRef = useRef<HTMLVideoElement | null>(null)

  // === 播放状态 ===
  const [playMode, setPlayMode] = useState<'none' | 'single' | 'all'>('none')
  const [playIdx, setPlayIdx] = useState(-1)
  const [playElapsed, setPlayElapsed] = useState(0)

  // === 转场状态 ===
  const [transProgress, setTransProgress] = useState(0)
  const [transActive, setTransActive] = useState(false)
  const [transStyle, setTransStyle] = useState('')
  const transRAF = useRef(0)
  const switchingRef = useRef(false)

  // === 时间轴交互 ===
  const [timelineDragOver, setTimelineDragOver] = useState(false)
  const [assetDurations, setAssetDurations] = useState<Record<string, number>>({})
  const loadedDurIdsRef = useRef<Set<string>>(new Set())

  // === 音轨分离 ===
  const [separatingIdx, setSeparatingIdx] = useState(-1)
  const [downloadPct, setDownloadPct] = useState(0)
  const [videoLoading, setVideoLoading] = useState(false)
  const needPlayRef = useRef(-1)

  // === 标尺播放头 ===
  const [draggingPH, setDraggingPH] = useState(false)
  const [phDragTime, setPhDragTime] = useState(0)
  const rulerRef = useRef<HTMLDivElement>(null)
  const [pausedAtTime, setPausedAtTime] = useState(-1)
  const prevDragPH = useRef(false)

  // === 草稿 ===
  const [showExitConfirm, setShowExitConfirm] = useState(false)
  const [drafts, setDrafts] = useState<VideoDraftItem[]>([])
  const [draftDismissed, setDraftDismissed] = useState(false)

  // === 上传 ===
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)

  // === v0.42.8 字幕轨 ===
  const [subtitleSegments, setSubtitleSegments] = useState<SubtitleSegment[]>([])
  const [subtitleLanguage, setSubtitleLanguage] = useState('zh-CN')
  const [editingSubtitle, setEditingSubtitle] = useState<SubtitleSegment | null>(null)
  // v0.42.9 TTS 弹窗
  const [showTTSModal, setShowTTSModal] = useState(false)
  const [subtitleDbId, setSubtitleDbId] = useState('')

  // === Refs(供 setInterval/RAF 闭包读取最新值) ===
  const clipsRef = useRef(clips); clipsRef.current = clips
  const playIdxRef = useRef(playIdx); playIdxRef.current = playIdx
  const playModeRef = useRef(playMode); playModeRef.current = playMode
  const transActiveRef = useRef(false); transActiveRef.current = transActive

  // ==================== 草稿持久化 ====================
  const saveDraftToServer = useCallback(async (name?: string) => {
    if (!coursewareId || clips.length === 0) return
    const clipsData = clips.map(c => { const { blobUrl: _blobUrl, ...rest } = c; void _blobUrl; return rest })
    await saveVideoDraft(coursewareId, { name: name || '', clips_data: clipsData, clip_count: clips.length })
    if (subtitleSegments.length > 0) {
      try {
        const subResp = await upsertSubtitle(coursewareId, { scope_type: 'editor_draft', language: subtitleLanguage, segments: JSON.stringify(subtitleSegments) })
        if (subResp?.id) setSubtitleDbId(subResp.id)
      } catch (e) { console.warn('[字幕持久化] 保存失败:', e) }
    }
  }, [coursewareId, clips, subtitleSegments, subtitleLanguage])

  // ==================== 视频资源辅助 ====================
  const getOtherVideo = useCallback((): HTMLVideoElement | null => {
    const a = activeVideoRef.current
    if (a === videoARef.current) return videoBRef.current
    if (a === videoBRef.current) return videoARef.current
    return videoBRef.current
  }, [])

  const loadDuration = useCallback(async (url: string): Promise<number> => {
    return new Promise(resolve => {
      const v = document.createElement('video'); v.preload = 'metadata'
      v.onloadedmetadata = () => { resolve(v.duration || 5); v.remove() }
      v.onerror = () => { resolve(5); v.remove() }
      v.src = url
    })
  }, [])

  useEffect(() => { videos.forEach(async (v) => { if (loadedDurIdsRef.current.has(v.id)) return; loadedDurIdsRef.current.add(v.id); const dur = await loadDuration(v.url); setAssetDurations(prev => ({ ...prev, [v.id]: dur })) }) }, [videos, loadDuration])
  useEffect(() => { if (videoARef.current && !activeVideoRef.current) activeVideoRef.current = videoARef.current }, [])
  useEffect(() => { return () => { clipsRef.current.forEach(c => { if (c.blobUrl) try { URL.revokeObjectURL(c.blobUrl) } catch { /* */ } }) } }, [])
  useEffect(() => { if (!coursewareId) return; listVideoDrafts(coursewareId).then(res => { if (res.drafts?.length > 0) setDrafts(res.drafts) }).catch(() => {}) }, [coursewareId])

  const loadDraft = useCallback((draft: VideoDraftItem) => {
    try {
      const arr = Array.isArray(draft.clips_data) ? draft.clips_data : JSON.parse(draft.clips_data)
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const valid = arr.filter((c: any) => c?.url && c?.id && c?.duration > 0)
      if (valid.length === 0) { alert('草稿中的视频已全部失效，请重新编辑'); return }
      if (valid.length < arr.length) alert(`注意: ${arr.length - valid.length} 个片段因视频失效已自动跳过`)
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      setClips(valid.map((c: any) => ({ ...c, blobUrl: undefined, trackType: c.trackType || 'video' })))
      setDraftDismissed(true)
      if (coursewareId) {
        listSubtitles(coursewareId, 'editor_draft').then(subs => {
          if (subs && subs.length > 0) {
            try { const parsed = JSON.parse(subs[0].segments); if (Array.isArray(parsed)) setSubtitleSegments(parsed); if (subs[0].language) setSubtitleLanguage(subs[0].language); if (subs[0].id) setSubtitleDbId(subs[0].id) } catch { /* */ }
          }
        }).catch(() => {})
      }
    } catch { alert('草稿数据解析失败') }
  }, [coursewareId])

  // ==================== 音频同步 ====================
  const syncAudioToVideo = useCallback((clip: EditorClip | null, video: HTMLVideoElement | null) => {
    const audio = audioRef.current; if (!audio) return
    if (!clip || !video) { audio.pause(); audio.removeAttribute('src'); audio.load(); return }
    if (clip.audioUrl) {
      video.muted = true
      const offset = Math.max(0, video.currentTime - clip.trimStart)
      if (!audio.src.endsWith(clip.audioUrl)) { audio.src = clip.audioUrl; audio.load() }
      audio.volume = (clip.audioMuted ?? false) ? 0 : (clip.audioVolume ?? 1.0)
      audio.muted = clip.audioMuted ?? false
      try { audio.currentTime = offset } catch { /* */ }
    } else {
      video.muted = !!clip.muted; audio.pause(); audio.removeAttribute('src'); audio.load()
    }
  }, [])
  const stopAudio = useCallback(() => { audioRef.current?.pause() }, [])
  const playAudio = useCallback(async (clip: EditorClip | null) => { const a = audioRef.current; if (!a || !clip?.audioUrl) return; try { await a.play() } catch { /* */ } }, [])

  // ==================== 片段管理 ====================
  const updateClip = useCallback((idx: number, u: Partial<EditorClip>) => { setClips(prev => prev.map((c, i) => i === idx ? { ...c, ...u } : c)) }, [])

  useEffect(() => { const a = audioRef.current; if (!a || playMode === 'none' || playIdx < 0) return; const clip = clips[playIdx]; if (!clip?.audioUrl) return; a.volume = (clip.audioMuted ?? false) ? 0 : (clip.audioVolume ?? 1.0); a.muted = clip.audioMuted ?? false }, [clips, playMode, playIdx])

  const addClip = useCallback(async (video: { id: string; url: string; label: string }) => {
    if (clips.some(c => c.id === video.id)) return
    const dur = assetDurations[video.id] || await loadDuration(video.url)
    setClips(prev => [...prev, { id: video.id, url: video.url, label: video.label, duration: dur, trimStart: 0, trimEnd: dur, transition: 'none', transDur: 0.5, trackType: 'video', audioMuted: false, audioVolume: 1.0 }])
  }, [clips, loadDuration, assetDurations])

  const resetVolumes = useCallback(() => { if (videoARef.current) videoARef.current.volume = 1; if (videoBRef.current) videoBRef.current.volume = 1 }, [])
  const clearInactiveVideo = useCallback(() => { const act = activeVideoRef.current; const other = act === videoARef.current ? videoBRef.current : videoARef.current; if (other) { other.pause(); other.removeAttribute('src'); other.load() } }, [])

  const removeClip = useCallback((idx: number) => {
    const clip = clips[idx]; if (clip?.blobUrl) try { URL.revokeObjectURL(clip.blobUrl) } catch { /* */ }
    videoARef.current?.pause(); videoBRef.current?.pause(); stopAudio(); resetVolumes(); switchingRef.current = false; setPlayMode('none'); setPlayIdx(-1)
    setClips(prev => prev.filter((_, i) => i !== idx))
    if (activeIdx === idx) setActiveIdx(-1); else if (activeIdx > idx) setActiveIdx(activeIdx - 1)
  }, [activeIdx, clips, resetVolumes, stopAudio])

  // 拖拽排序
  const handleDragStart = (i: number, e: React.DragEvent) => { setDragIdx(i); e.dataTransfer.effectAllowed = 'move' }
  const handleDragOver = (e: React.DragEvent, i: number) => { e.preventDefault(); setDragOverIdx(i) }
  const handleDrop = (i: number) => { if (dragIdx < 0 || dragIdx === i) { setDragIdx(-1); setDragOverIdx(-1); return }; setClips(prev => { const a = [...prev]; const [m] = a.splice(dragIdx, 1); a.splice(i, 0, m); return a }); setDragIdx(-1); setDragOverIdx(-1) }
  const handleTimelineDragOver = useCallback((e: React.DragEvent) => { if (e.dataTransfer.types.includes(DRAG_TYPE_ASSET)) { e.preventDefault(); e.dataTransfer.dropEffect = 'copy'; setTimelineDragOver(true) } }, [])
  const handleTimelineDragLeave = useCallback(() => setTimelineDragOver(false), [])
  const handleTimelineDrop = useCallback((e: React.DragEvent) => { setTimelineDragOver(false); const vid = e.dataTransfer.getData(DRAG_TYPE_ASSET); if (!vid) return; const v = videos.find(x => x.id === vid); if (v) addClip(v) }, [videos, addClip])

  // ==================== 播放核心 ====================
  const stopAll = useCallback(() => { videoARef.current?.pause(); videoBRef.current?.pause(); stopAudio(); cancelAnimationFrame(transRAF.current); resetVolumes(); switchingRef.current = false; setPlayMode('none'); setPlayIdx(-1); setPlayElapsed(0); setTransActive(false); setTransProgress(0); setVideoLoading(false); setPausedAtTime(-1) }, [resetVolumes, stopAudio])

  const loadAndSeek = useCallback((v: HTMLVideoElement, clip: EditorClip): Promise<void> => {
    return new Promise(resolve => {
      let done = false
      const finish = () => { if (done) return; done = true; v.oncanplaythrough = null; v.oncanplay = null; v.onseeked = null; resolve() }
      const doSeek = () => { if (done) return; if (clip.trimStart < 0.1 && v.currentTime < 0.1) { finish(); return }; v.onseeked = () => finish(); v.currentTime = clip.trimStart; setTimeout(() => { if (!done) finish() }, 500) }
      const srcUrl = clip.blobUrl || clip.url; const cur = v.currentSrc || v.src || ''; const isSame = (cur && cur === srcUrl) || (cur && srcUrl && cur.endsWith(srcUrl))
      if (isSame && v.readyState >= 3) { doSeek(); return }
      v.oncanplaythrough = () => { v.oncanplaythrough = null; v.oncanplay = null; doSeek() }
      v.oncanplay = () => { setTimeout(() => { if (!done) { v.oncanplaythrough = null; v.oncanplay = null; doSeek() } }, 200) }
      v.src = srcUrl; v.load(); setTimeout(() => { if (!done) finish() }, 5000)
    })
  }, [])

  const waitForPlaying = useCallback((v: HTMLVideoElement): Promise<void> => {
    return new Promise(resolve => { if (!v.paused && v.readyState >= 3) { resolve(); return }; let done = false; const finish = () => { if (done) return; done = true; v.onplaying = null; resolve() }; v.onplaying = () => finish(); v.volume = 1; v.play().catch(() => finish()); setTimeout(() => finish(), 300) })
  }, [])

  const findClipIdxByGlobalTime = useCallback((globalTime: number, cls: EditorClip[]): { idx: number; offsetInClip: number } => {
    if (cls.length === 0) return { idx: -1, offsetInClip: 0 }; let accT = 0
    for (let i = 0; i < cls.length; i++) { const d = cls[i].trimEnd - cls[i].trimStart; if (globalTime <= accT + d + 0.01) return { idx: i, offsetInClip: Math.max(0, globalTime - accT) }; accT += d }
    return { idx: cls.length - 1, offsetInClip: cls[cls.length - 1].trimEnd - cls[cls.length - 1].trimStart }
  }, [])

  const resumeFromTime = useCallback(async (globalTime: number) => {
    const cls = clipsRef.current; if (cls.length === 0) return
    const totalD = cls.reduce((s, c) => s + (c.trimEnd - c.trimStart), 0)
    if (globalTime >= totalD - 0.1) { playAll(); return }
    const { idx, offsetInClip } = findClipIdxByGlobalTime(globalTime, cls); if (idx < 0) return
    const clip = cls[idx]; const seekPos = clip.trimStart + Math.min(offsetInClip, clip.trimEnd - clip.trimStart - 0.1)
    stopAll(); setVideoLoading(true); const p = videoARef.current; if (!p) { setVideoLoading(false); return }
    activeVideoRef.current = p; setTopPlayer('A'); clearInactiveVideo()
    await loadAndSeek(p, { ...clip, trimStart: seekPos }); syncAudioToVideo(clip, p); await waitForPlaying(p); await playAudio(clip)
    setVideoLoading(false); setPlayMode('all'); setPlayIdx(idx); setActiveIdx(idx); setPlayElapsed(globalTime); setPausedAtTime(-1)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [findClipIdxByGlobalTime, stopAll, loadAndSeek, syncAudioToVideo, waitForPlaying, playAudio, clearInactiveVideo])

  const playSingle = useCallback(async (idx: number) => {
    const clip = clips[idx]; if (!clip) return; stopAll(); setVideoLoading(true)
    const p = videoARef.current; if (!p) return; activeVideoRef.current = p; setTopPlayer('A'); clearInactiveVideo()
    let seekTo = clip.trimStart
    if (pausedAtTime >= 0) { const ph = findClipIdxByGlobalTime(pausedAtTime, clips); if (ph.idx === idx) { const proposed = clip.trimStart + Math.min(ph.offsetInClip, clip.trimEnd - clip.trimStart - 0.1); if (proposed > clip.trimStart) seekTo = proposed } }
    await loadAndSeek(p, { ...clip, trimStart: seekTo }); syncAudioToVideo(clip, p); await waitForPlaying(p); await playAudio(clip)
    setVideoLoading(false); setPlayMode('single'); setPlayIdx(idx); setActiveIdx(idx); setPausedAtTime(-1)
  }, [clips, stopAll, loadAndSeek, waitForPlaying, clearInactiveVideo, syncAudioToVideo, playAudio, pausedAtTime, findClipIdxByGlobalTime])

  const playAll = useCallback(async () => {
    if (clips.length === 0) return; stopAll(); const p = videoARef.current; if (!p) return
    activeVideoRef.current = p; setTopPlayer('A'); clearInactiveVideo()
    await loadAndSeek(p, clips[0]); syncAudioToVideo(clips[0], p); await waitForPlaying(p); await playAudio(clips[0])
    setPlayMode('all'); setPlayIdx(0); setActiveIdx(0); setPlayElapsed(0)
  }, [clips, stopAll, loadAndSeek, waitForPlaying, clearInactiveVideo, syncAudioToVideo, playAudio])

  const togglePlayPause = useCallback(() => {
    if (transActiveRef.current || switchingRef.current) return; const a = activeVideoRef.current; if (!a) return; const audio = audioRef.current
    if (playMode !== 'none' && !a.paused) { a.pause(); audio?.pause(); return }
    if (playMode !== 'none' && a.paused) { a.play().catch(() => {}); audio?.play().catch(() => {}); return }
    if (clips.length > 0) { if (pausedAtTime >= 0) resumeFromTime(pausedAtTime); else playAll() }
  }, [playMode, clips, playAll, pausedAtTime, resumeFromTime])

  // 转场
  const doCrossfade = useCallback((fromIdx: number, toIdx: number) => {
    const all = clipsRef.current; if (toIdx >= all.length) { stopAll(); return }
    const fc = all[fromIdx], tc = all[toIdx]; const trans = fc?.transition || 'none', td = fc?.transDur || 0.5
    const out = activeVideoRef.current, inc = getOtherVideo(); if (!out || !inc) { stopAll(); return }
    switchingRef.current = true
    if (trans === 'none') {
      out.pause(); out.volume = 1; stopAudio()
      loadAndSeek(inc, tc).then(() => { inc.volume = 1; syncAudioToVideo(tc, inc); inc.play().catch(() => {}); playAudio(tc); activeVideoRef.current = inc; setTopPlayer(inc === videoARef.current ? 'A' : 'B'); setPlayIdx(toIdx); setActiveIdx(toIdx); switchingRef.current = false })
      return
    }
    setTransActive(true); setTransStyle(trans); setTransProgress(0); setPlayIdx(toIdx); setActiveIdx(toIdx)
    loadAndSeek(inc, tc).then(() => { inc.volume = 1; syncAudioToVideo(tc, inc); inc.play().catch(() => {}); playAudio(tc) })
    const t0 = performance.now(), ms = td * 1000
    const anim = (now: number) => {
      const p = Math.min(1, (now - t0) / ms); setTransProgress(p); try { out.volume = Math.max(0, 1 - p) } catch { /* */ }
      const audio = audioRef.current; if (audio && fc.audioUrl) { const baseVol = fc.audioMuted ? 0 : (fc.audioVolume ?? 1.0); try { audio.volume = Math.max(0, baseVol * (1 - p)) } catch { /* */ } }
      if (p < 1) { transRAF.current = requestAnimationFrame(anim) }
      else { out.pause(); out.volume = 1; activeVideoRef.current = inc; setTopPlayer(inc === videoARef.current ? 'A' : 'B'); setTransActive(false); setTransProgress(0); setTransStyle(''); switchingRef.current = false }
    }
    transRAF.current = requestAnimationFrame(anim)
  }, [stopAll, getOtherVideo, loadAndSeek, syncAudioToVideo, playAudio, stopAudio])

  // 进度监听
  useEffect(() => {
    const check = () => {
      const mode = playModeRef.current; if (mode === 'none' || switchingRef.current || transActiveRef.current) return
      const idx = playIdxRef.current, all = clipsRef.current; if (idx < 0 || idx >= all.length) return
      const clip = all[idx], a = activeVideoRef.current; if (!a || !clip) return
      if (mode === 'single') {
        if (a.currentTime >= clip.trimEnd - 0.05) { a.pause(); stopAudio(); setPlayMode('none'); setPlayIdx(-1) }
        let singleEl = 0; for (let i = 0; i < idx; i++) singleEl += all[i].trimEnd - all[i].trimStart; singleEl += Math.max(0, a.currentTime - clip.trimStart); setPlayElapsed(singleEl)
        return
      }
      let el = 0; for (let i = 0; i < idx; i++) el += all[i].trimEnd - all[i].trimStart; el += Math.max(0, a.currentTime - clip.trimStart); setPlayElapsed(el)
      const audio = audioRef.current; if (audio && clip.audioUrl && !audio.paused) { const exp = a.currentTime - clip.trimStart; if (Math.abs(audio.currentTime - exp) > 0.2) { try { audio.currentTime = exp } catch { /* */ } } }
      if (idx >= all.length - 1) { if (a.currentTime >= clip.trimEnd - 0.05) stopAll(); return }
      const t = clip.transition || 'none', td = clip.transDur || 0.5
      if (t !== 'none') { if (a.currentTime >= clip.trimEnd - td - 0.05) doCrossfade(idx, idx + 1) }
      else { if (a.currentTime >= clip.trimEnd - 0.05) doCrossfade(idx, idx + 1) }
    }
    const tmr = setInterval(check, 50); return () => clearInterval(tmr)
  }, [doCrossfade, stopAll, stopAudio])

  // 键盘
  useEffect(() => { const fn = (e: KeyboardEvent) => { if (e.key === 'Escape') { e.preventDefault(); if (clips.length > 0) setShowExitConfirm(true); else { stopAll(); onClose() } }; if (e.key === ' ') { e.preventDefault(); togglePlayPause() } }; window.addEventListener('keydown', fn); return () => window.removeEventListener('keydown', fn) }, [onClose, stopAll, togglePlayPause, clips.length])
  useEffect(() => { const idx = needPlayRef.current; if (idx < 0 || idx >= clips.length) return; needPlayRef.current = -1; playSingle(idx) }, [clips, playSingle])

  // ==================== 标尺跳转 + 播放头拖拽 ====================
  const seekToTime = useCallback(async (globalTime: number) => {
    try {
      const cls = clipsRef.current; if (cls.length === 0) return; let accT = 0, targetIdx = 0
      for (let i = 0; i < cls.length; i++) { const d = cls[i].trimEnd - cls[i].trimStart; if (globalTime <= accT + d + 0.01) { targetIdx = i; break }; accT += d; if (i === cls.length - 1) targetIdx = i }
      const clip = cls[targetIdx]; const offsetInClip = Math.max(0, globalTime - accT); const seekPos = clip.trimStart + Math.min(offsetInClip, clip.trimEnd - clip.trimStart - 0.1)
      stopAll(); setVideoLoading(true); const p = videoARef.current; if (!p) { setVideoLoading(false); return }
      activeVideoRef.current = p; setTopPlayer('A'); clearInactiveVideo(); await loadAndSeek(p, { ...clip, trimStart: seekPos }); syncAudioToVideo(clip, p); setVideoLoading(false); setActiveIdx(targetIdx)
      let pt = 0; for (let i = 0; i < targetIdx; i++) pt += cls[i].trimEnd - cls[i].trimStart; pt += seekPos - clip.trimStart; setPausedAtTime(pt)
    } catch (e) { console.warn('[ruler seek]', e); setVideoLoading(false) }
  }, [stopAll, loadAndSeek, clearInactiveVideo, syncAudioToVideo])

  const handleRulerClick = useCallback((e: React.MouseEvent) => { if (draggingPH) return; const ruler = rulerRef.current; if (!ruler) return; const rect = ruler.getBoundingClientRect(); seekToTime(pixelToTime(Math.max(0, e.clientX - rect.left), clipsRef.current)) }, [draggingPH, seekToTime])
  const handlePHDown = useCallback((e: React.PointerEvent) => { e.preventDefault(); e.stopPropagation(); setDraggingPH(true) }, [])

  useEffect(() => { if (!draggingPH) return; const onMove = (e: PointerEvent) => { const ruler = rulerRef.current; if (!ruler) return; const rect = ruler.getBoundingClientRect(); setPhDragTime(pixelToTime(Math.max(0, Math.min(e.clientX - rect.left, totalPxWidth(clipsRef.current))), clipsRef.current)) }; const onUp = () => setDraggingPH(false); document.addEventListener('pointermove', onMove); document.addEventListener('pointerup', onUp); return () => { document.removeEventListener('pointermove', onMove); document.removeEventListener('pointerup', onUp) } }, [draggingPH])
  useEffect(() => { if (prevDragPH.current && !draggingPH && phDragTime > 0) seekToTime(phDragTime); prevDragPH.current = draggingPH }, [draggingPH, phDragTime, seekToTime])

  // ==================== 音轨分离 ====================
  const handleSeparateAudio = useCallback(async () => {
    if (!coursewareId || activeIdx < 0 || separatingIdx >= 0) return; const clip = clips[activeIdx]; if (!clip || clip.muted) return
    setSeparatingIdx(activeIdx); setDownloadPct(0)
    try {
      const muteResp = await muteCWVideo(coursewareId, clip.id)
      let muteBlobUrl: string | undefined; let blobFail: string | null = null
      try { muteBlobUrl = await urlToBlobUrl(muteResp.url, (r, t) => { if (t > 0) setDownloadPct(Math.min(99, Math.round((r / t) * 100))) }) } catch (e) { blobFail = e instanceof Error ? e.message : '未知错误'; console.warn('[VideoEditor] blob降级:', blobFail) }
      let audioInfo: { url: string; duration: string; fileSize: number } | null = null
      try { const ar = await extractCWAudio(coursewareId, clip.id); audioInfo = { url: ar.url, duration: ar.duration, fileSize: ar.file_size } } catch { /* 无音频流 */ }
      stopAll(); updateClip(activeIdx, { id: muteResp.asset_id, url: muteResp.url, blobUrl: muteBlobUrl, muted: true, originalId: clip.id, originalUrl: clip.url, audioUrl: audioInfo?.url, audioDuration: audioInfo?.duration, audioFileSize: audioInfo?.fileSize, audioMuted: false, audioVolume: 1.0 })
      needPlayRef.current = activeIdx
      if (blobFail) setTimeout(() => alert('静音已完成,但本地缓存失败可能轻微卡顿。\n' + blobFail), 100)
    } catch (err) { alert('分离音轨失败: ' + (err instanceof Error ? err.message : '未知错误')) }
    finally { setSeparatingIdx(-1); setDownloadPct(0) }
  }, [coursewareId, activeIdx, clips, separatingIdx, stopAll, updateClip])

  const handleDeleteAudio = useCallback(() => { if (activeIdx < 0) return; updateClip(activeIdx, { audioUrl: undefined, audioDuration: undefined, audioFileSize: undefined, audioMuted: false, audioVolume: 1.0 }) }, [activeIdx, updateClip])

  const handleRestoreOriginal = useCallback(() => {
    if (activeIdx < 0) return; const clip = clips[activeIdx]; if (!clip?.originalId || !clip?.originalUrl) return
    if (clip.blobUrl) try { URL.revokeObjectURL(clip.blobUrl) } catch { /* */ }; stopAll()
    updateClip(activeIdx, { id: clip.originalId, url: clip.originalUrl, blobUrl: undefined, muted: false, originalId: undefined, originalUrl: undefined, audioUrl: undefined, audioDuration: undefined, audioFileSize: undefined, audioMuted: false, audioVolume: 1.0 })
    needPlayRef.current = activeIdx
  }, [activeIdx, clips, stopAll, updateClip])

  // ==================== 素材库上传 ====================
  const handleUploadVideo = useCallback(async () => {
    if (!onUploadVideo) return; const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'video/mp4,video/webm,video/quicktime,.mp4,.webm,.mov,.avi'
    inp.onchange = async (ev) => { const f = (ev.target as HTMLInputElement).files?.[0]; if (!f) return; if (f.size > 50 * 1024 * 1024) { alert('视频不能超过 50MB'); return }; setUploading(true); setUploadProgress(0); try { const result = await onUploadVideo(f, (pct) => setUploadProgress(pct)); if (result) alert('✅ 视频上传成功: ' + result.label + ' (' + fmtSize(f.size) + ')') } catch (e) { alert('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setUploading(false); setUploadProgress(0) } }
    inp.click()
  }, [onUploadVideo])

  // ==================== 导出 + 退出 ====================
  const handleExport = useCallback(() => {
    if (clips.length === 0 || exporting) return; stopAll()
    if (coursewareId && subtitleSegments.length > 0) { upsertSubtitle(coursewareId, { scope_type: 'editor_draft', language: subtitleLanguage, segments: JSON.stringify(subtitleSegments) }).catch(e => console.warn('[字幕持久化] 导出前保存失败:', e)) }
    onExport(clips.map(c => ({ asset_id: c.id, start_sec: c.trimStart, end_sec: c.trimEnd, transition: c.transition, trans_dur: c.transDur })))
  }, [clips, exporting, stopAll, coursewareId, subtitleSegments, subtitleLanguage, onExport])

  const doClose = useCallback(() => { stopAll(); setShowExitConfirm(false); onClose() }, [stopAll, onClose])
  const handleCloseClick = useCallback(() => { if (clips.length > 0) setShowExitConfirm(true); else doClose() }, [clips.length, doClose])

  // ==================== 渲染计算 ====================
  const totalDur = useMemo(() => clips.reduce((s, c) => s + (c.trimEnd - c.trimStart), 0), [clips])
  const ac = activeIdx >= 0 ? clips[activeIdx] : null
  const audioClipCount = useMemo(() => clips.filter(c => !!c.audioUrl).length, [clips])

  let phTime = 0
  if (draggingPH) phTime = phDragTime
  else if (playMode !== 'none') phTime = playElapsed
  else if (pausedAtTime >= 0) phTime = pausedAtTime
  else if (activeIdx >= 0) { for (let i = 0; i < activeIdx; i++) phTime += clips[i].trimEnd - clips[i].trimStart }

  // ==================== 渲染 ====================
  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 99996, background: C.bg, display: 'flex', flexDirection: 'column' }}>
      {/* 顶部工具栏 */}
      <VideoEditorToolbar clipCount={clips.length} totalDur={totalDur} audioClipCount={audioClipCount} draftCount={drafts.length} draftDismissed={draftDismissed} playMode={playMode} exporting={exporting} onToggleDrafts={() => setDraftDismissed(false)} onPlayAll={playAll} onStopAll={stopAll} onClose={handleCloseClick} onExport={handleExport} />

      {/* 草稿恢复列表 */}
      <VideoEditorDraftPanel drafts={drafts} visible={drafts.length > 0 && !draftDismissed} coursewareId={coursewareId} onLoadDraft={loadDraft} onDeleteDraft={async (draftId) => { if (!coursewareId) return; try { await deleteVideoDraft(coursewareId, draftId); setDrafts(prev => prev.filter(x => x.id !== draftId)) } catch { alert('删除失败,请重试') } }} onDismiss={() => setDraftDismissed(true)} />

      {/* 连贯预览进度条 */}
      {playMode === 'all' && <VideoEditorProgressBar clips={clips} playIdx={playIdx} playElapsed={playElapsed} totalDur={totalDur} />}

      {/* 主区域 */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden', minHeight: 0 }}>
        <VideoEditorMaterials videos={videos} clips={clips} assetDurations={assetDurations} uploading={uploading} uploadProgress={uploadProgress} onAddClip={addClip} onUploadClick={onUploadVideo ? handleUploadVideo : undefined} />
        <VideoEditorPlayerStageComponent hasClips={clips.length > 0} topPlayer={topPlayer} transActive={transActive} transStyle={transStyle} transProgress={transProgress} playMode={playMode} playIdx={playIdx} clipCount={clips.length} videoLoading={videoLoading} separatingIdx={separatingIdx} downloadPct={downloadPct} phTime={phTime} totalDur={totalDur} subtitleSegments={subtitleSegments} subtitleLanguage={subtitleLanguage} onTogglePlayPause={togglePlayPause} videoARef={videoARef} videoBRef={videoBRef} audioRef={audioRef} />
        <VideoEditorPanel ac={ac} activeIdx={activeIdx} clipsLength={clips.length} hasAudio={!!coursewareId} separatingIdx={separatingIdx} onUpdateClip={updateClip} onSeparateAudio={handleSeparateAudio} onDeleteAudio={handleDeleteAudio} onRestoreOriginal={handleRestoreOriginal} onPlaySingle={playSingle} onPlayAll={playAll} />
      </div>

      {/* 多轨道时间轴 */}
      <div style={{ borderTop: `1px solid ${C.border}`, background: C.surface, padding: '10px 24px 14px', flexShrink: 0 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
          <span style={{ fontSize: 12, color: C.textMuted }}>🎞️ 多轨道时间轴 — 三轨堆叠 · 拖手柄裁剪 · 点击标尺跳转 · 空格播放</span>
          {clips.length >= 2 && <button onClick={playMode === 'all' ? stopAll : playAll} style={{ padding: '5px 14px', borderRadius: 6, border: `1px solid ${C.playing}`, background: playMode === 'all' ? C.playing + '15' : 'transparent', color: C.playing, fontSize: 12, fontWeight: 600, cursor: 'pointer' }}>{playMode === 'all' ? '⏹ 停止' : '▶ 连贯预览'}</button>}
        </div>
        <VideoEditorTimeline clips={clips} activeIdx={activeIdx} playIdx={playIdx} playMode={playMode} dragIdx={dragIdx} dragOverIdx={dragOverIdx} timelineDragOver={timelineDragOver} phTime={phTime} draggingPH={draggingPH} phDragTime={phDragTime} setActiveIdx={setActiveIdx} removeClip={removeClip} updateClip={updateClip} playSingle={playSingle} handleDragStart={handleDragStart} handleDragOver={handleDragOver} handleDrop={handleDrop} setDragIdx={setDragIdx} setDragOverIdx={setDragOverIdx} handleTimelineDragOver={handleTimelineDragOver} handleTimelineDragLeave={handleTimelineDragLeave} handleTimelineDrop={handleTimelineDrop} handleRulerClick={handleRulerClick} rulerRef={rulerRef} handlePHDown={handlePHDown} subtitleSegments={subtitleSegments.filter(s => s.language === subtitleLanguage)} subtitleLanguage={subtitleLanguage} onSubtitleSegmentsChange={setSubtitleSegments} onEditSubtitleSegment={setEditingSubtitle} onRequestTTS={() => { if (subtitleDbId) { setShowTTSModal(true) } else if (coursewareId && subtitleSegments.length > 0) { upsertSubtitle(coursewareId, { scope_type: 'editor_draft', language: subtitleLanguage, segments: JSON.stringify(subtitleSegments) }).then(r => { if (r?.id) { setSubtitleDbId(r.id); setShowTTSModal(true) } }).catch(e => alert('保存字幕失败: ' + (e instanceof Error ? e.message : '未知错误'))) } else { alert('请先添加字幕条') } }} />
        <div style={{ marginTop: 6, fontSize: 11, color: C.textMuted, textAlign: 'center' }}>💡 拖手柄裁剪 · 点击标尺跳转 · 拖▼播放头定位 · 空格播放/暂停 · ESC退出</div>
      </div>

      {/* v0.42.9 TTS 配音弹窗 */}
      {showTTSModal && coursewareId && subtitleDbId && (
        <VideoEditorTTSModal
          coursewareId={coursewareId}
          subtitleId={subtitleDbId}
          segments={subtitleSegments.filter(s => s.language === subtitleLanguage)}
          language={subtitleLanguage}
          onComplete={(updated) => { setSubtitleSegments(updated); setShowTTSModal(false) }}
          onClose={() => setShowTTSModal(false)}
        />
      )}

      {/* 字幕编辑弹窗 */}
      {editingSubtitle && <VideoEditorSubtitleModal segment={editingSubtitle} languages={SUBTITLE_LANGUAGES} onSave={(updated) => { setSubtitleSegments(prev => prev.map(s => s.id === updated.id ? updated : s)); setEditingSubtitle(null) }} onClose={() => setEditingSubtitle(null)} />}

      {/* 退出确认弹窗 */}
      {showExitConfirm && <VideoEditorExitDialog clipCount={clips.length} onSaveDraft={async (name) => { await saveDraftToServer(name); doClose() }} onDiscard={doClose} onCancel={() => setShowExitConfirm(false)} />}
    </div>
  )
}
