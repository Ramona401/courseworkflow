/**
 * useVideoEditorPlayback.ts — 视频编辑器播放控制
 *
 * 职责：
 *   - 双video播放器预加载、播放与切换；
 *   - 单片段播放和连续播放；
 *   - 独立音轨及TTS配音同步；
 *   - 片段转场与渐变；
 *   - 标尺点击和播放头拖动；
 *   - 统一停止、卸载和blob URL清理。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import type {
  Dispatch,
  MouseEvent as ReactMouseEvent,
  PointerEvent as ReactPointerEvent,
  SetStateAction,
} from 'react'
import {
  pixelToTime,
  totalPxWidth,
} from './VideoEditorUtils'
import type { EditorClip } from './VideoEditorTypes'
import type { SubtitleSegment } from './VideoEditorSubtitleTrack'

export type VideoEditorPlayMode = 'none' | 'single' | 'all'

interface PlaybackParams {
  clips: EditorClip[]
  activeIdx: number
  setActiveIdx: Dispatch<SetStateAction<number>>
  subtitleSegments: SubtitleSegment[]
  subtitleLanguage: string
}

interface ClipPosition {
  idx: number
  offsetInClip: number
}

/**
 * 将课件全局时间映射到具体片段和片段内偏移。
 */
function locateClip(
  globalTime: number,
  clips: EditorClip[],
): ClipPosition {
  if (clips.length === 0) {
    return { idx: -1, offsetInClip: 0 }
  }

  let accumulated = 0

  for (let index = 0; index < clips.length; index += 1) {
    const duration = clips[index].trimEnd - clips[index].trimStart

    if (globalTime <= accumulated + duration + 0.01) {
      return {
        idx: index,
        offsetInClip: Math.max(0, globalTime - accumulated),
      }
    }

    accumulated += duration
  }

  const last = clips[clips.length - 1]

  return {
    idx: clips.length - 1,
    offsetInClip: last.trimEnd - last.trimStart,
  }
}

/**
 * 视频编辑器播放控制Hook。
 */
export default function useVideoEditorPlayback({
  clips,
  activeIdx,
  setActiveIdx,
  subtitleSegments,
  subtitleLanguage,
}: PlaybackParams) {
  const videoARef = useRef<HTMLVideoElement>(null)
  const videoBRef = useRef<HTMLVideoElement>(null)
  const audioRef = useRef<HTMLAudioElement>(null)
  const narrationRef = useRef<HTMLAudioElement>(null)
  const rulerRef = useRef<HTMLDivElement>(null)
  const activeVideoRef = useRef<HTMLVideoElement | null>(null)

  const [topPlayer, setTopPlayer] = useState<'A' | 'B'>('A')
  const [playMode, setPlayMode] = useState<VideoEditorPlayMode>('none')
  const [playIdx, setPlayIdx] = useState(-1)
  const [playElapsed, setPlayElapsed] = useState(0)
  const [videoLoading, setVideoLoading] = useState(false)

  const [transProgress, setTransProgress] = useState(0)
  const [transActive, setTransActive] = useState(false)
  const [transStyle, setTransStyle] = useState('')

  const [draggingPH, setDraggingPH] = useState(false)
  const [phDragTime, setPhDragTime] = useState(0)
  const [pausedAtTime, setPausedAtTime] = useState(-1)

  const transRAF = useRef(0)
  const switchingRef = useRef(false)
  const previousDraggingPH = useRef(false)
  const queuedPlayIndex = useRef(-1)
  const currentNarrationID = useRef('')

  const clipsRef = useRef(clips)
  const playModeRef = useRef(playMode)
  const playIdxRef = useRef(playIdx)
  const transActiveRef = useRef(transActive)
  const subtitleSegmentsRef = useRef(subtitleSegments)
  const subtitleLanguageRef = useRef(subtitleLanguage)

  clipsRef.current = clips
  playModeRef.current = playMode
  playIdxRef.current = playIdx
  transActiveRef.current = transActive
  subtitleSegmentsRef.current = subtitleSegments
  subtitleLanguageRef.current = subtitleLanguage

  const totalDur = useMemo(
    () => clips.reduce(
      (sum, clip) => sum + clip.trimEnd - clip.trimStart,
      0,
    ),
    [clips],
  )

  const playheadTime = useMemo(() => {
    if (draggingPH) return phDragTime
    if (playMode !== 'none') return playElapsed
    if (pausedAtTime >= 0) return pausedAtTime
    if (activeIdx < 0) return 0

    let elapsed = 0

    for (let index = 0; index < activeIdx; index += 1) {
      elapsed += clips[index].trimEnd - clips[index].trimStart
    }

    return elapsed
  }, [
    activeIdx,
    clips,
    draggingPH,
    pausedAtTime,
    phDragTime,
    playElapsed,
    playMode,
  ])

  /**
   * 将TTS配音定位到当前课件全局时间。
   */
  const syncNarration = useCallback((
    globalTime: number,
    videoPaused: boolean,
  ) => {
    const narration = narrationRef.current
    if (!narration) return

    const segment = subtitleSegmentsRef.current
      .filter((item) => (
        item.language === subtitleLanguageRef.current
        && !!item.tts_audio_url
      ))
      .find((item) => {
        const duration = item.tts_duration && item.tts_duration > 0
          ? item.tts_duration
          : item.end_sec - item.start_sec

        return (
          globalTime >= item.start_sec
          && globalTime < item.start_sec + duration
        )
      })

    if (!segment) {
      if (!narration.paused) narration.pause()
      currentNarrationID.current = ''
      return
    }

    if (videoPaused) {
      if (!narration.paused) narration.pause()
      return
    }

    const offset = globalTime - segment.start_sec

    if (currentNarrationID.current !== segment.id) {
      currentNarrationID.current = segment.id
      narration.src = segment.tts_audio_url as string

      try {
        narration.currentTime = offset
      } catch {
        // 元数据未就绪时由后续同步循环校准。
      }

      narration.volume = 1
      narration.play().catch(() => {
        // 浏览器可能拦截非用户触发的自动播放。
      })
      return
    }

    if (Math.abs(narration.currentTime - offset) > 0.3) {
      try {
        narration.currentTime = offset
      } catch {
        // 元数据未就绪时保持当前位置。
      }
    }

    if (narration.paused) {
      narration.play().catch(() => {
        // 浏览器可能拦截自动播放。
      })
    }
  }, [])

  const stopNarration = useCallback(() => {
    narrationRef.current?.pause()
    currentNarrationID.current = ''
  }, [])

  /**
   * 将片段独立音轨同步到video播放器。
   */
  const syncAudioToVideo = useCallback((
    clip: EditorClip | null,
    video: HTMLVideoElement | null,
  ) => {
    const audio = audioRef.current
    if (!audio) return

    if (!clip || !video) {
      audio.pause()
      audio.removeAttribute('src')
      audio.load()
      return
    }

    if (!clip.audioUrl) {
      video.muted = !!clip.muted
      audio.pause()
      audio.removeAttribute('src')
      audio.load()
      return
    }

    video.muted = true

    if (!audio.src.endsWith(clip.audioUrl)) {
      audio.src = clip.audioUrl
      audio.load()
    }

    audio.volume = clip.audioMuted ? 0 : (clip.audioVolume ?? 1)
    audio.muted = clip.audioMuted ?? false

    try {
      audio.currentTime = Math.max(0, video.currentTime - clip.trimStart)
    } catch {
      // 元数据未就绪时由播放循环校准。
    }
  }, [])

  const stopAudio = useCallback(() => {
    audioRef.current?.pause()
  }, [])

  const playAudio = useCallback(async (clip: EditorClip | null) => {
    const audio = audioRef.current

    if (!audio || !clip?.audioUrl) return

    try {
      await audio.play()
    } catch {
      // 浏览器可能拦截自动播放。
    }
  }, [])

  useEffect(() => {
    const audio = audioRef.current
    const clip = playIdx >= 0 ? clips[playIdx] : null

    if (!audio || playMode === 'none' || !clip?.audioUrl) return

    audio.volume = clip.audioMuted ? 0 : (clip.audioVolume ?? 1)
    audio.muted = clip.audioMuted ?? false
  }, [clips, playIdx, playMode])

  const resetVolumes = useCallback(() => {
    if (videoARef.current) videoARef.current.volume = 1
    if (videoBRef.current) videoBRef.current.volume = 1
  }, [])

  const clearInactiveVideo = useCallback(() => {
    const active = activeVideoRef.current
    const other = active === videoARef.current
      ? videoBRef.current
      : videoARef.current

    if (!other) return

    other.pause()
    other.removeAttribute('src')
    other.load()
  }, [])

  const stopAll = useCallback(() => {
    videoARef.current?.pause()
    videoBRef.current?.pause()
    stopAudio()
    stopNarration()
    cancelAnimationFrame(transRAF.current)
    resetVolumes()

    switchingRef.current = false
    setPlayMode('none')
    setPlayIdx(-1)
    setPlayElapsed(0)
    setVideoLoading(false)
    setTransActive(false)
    setTransProgress(0)
    setTransStyle('')
    setPausedAtTime(-1)
  }, [resetVolumes, stopAudio, stopNarration])

  /**
   * 确保播放器加载目标资源，并定位到片段裁剪起点。
   */
  const loadAndSeek = useCallback((
    video: HTMLVideoElement,
    clip: EditorClip,
  ): Promise<void> => new Promise((resolve) => {
    let finished = false

    const finish = () => {
      if (finished) return
      finished = true
      video.oncanplaythrough = null
      video.oncanplay = null
      video.onseeked = null
      resolve()
    }

    const seek = () => {
      if (finished) return

      if (clip.trimStart < 0.1 && video.currentTime < 0.1) {
        finish()
        return
      }

      video.onseeked = finish
      video.currentTime = clip.trimStart
      window.setTimeout(finish, 500)
    }

    const source = clip.blobUrl || clip.url
    const current = video.currentSrc || video.src || ''
    const sameSource = (
      current === source
      || (current && source && current.endsWith(source))
    )

    if (sameSource && video.readyState >= 3) {
      seek()
      return
    }

    video.oncanplaythrough = () => {
      video.oncanplaythrough = null
      video.oncanplay = null
      seek()
    }

    video.oncanplay = () => {
      window.setTimeout(() => {
        if (!finished) seek()
      }, 200)
    }

    video.src = source
    video.load()
    window.setTimeout(finish, 5000)
  }), [])

  const waitForPlaying = useCallback((
    video: HTMLVideoElement,
  ): Promise<void> => new Promise((resolve) => {
    if (!video.paused && video.readyState >= 3) {
      resolve()
      return
    }

    let finished = false

    const finish = () => {
      if (finished) return
      finished = true
      video.onplaying = null
      resolve()
    }

    video.onplaying = finish
    video.volume = 1
    video.play().catch(finish)
    window.setTimeout(finish, 300)
  }), [])

  const startPlayback = useCallback(async (
    index: number,
    seekPosition: number,
    mode: VideoEditorPlayMode,
    elapsed: number,
  ) => {
    const clip = clipsRef.current[index]
    if (!clip) return

    stopAll()
    setVideoLoading(true)

    const player = videoARef.current

    if (!player) {
      setVideoLoading(false)
      return
    }

    activeVideoRef.current = player
    setTopPlayer('A')
    clearInactiveVideo()

    await loadAndSeek(player, {
      ...clip,
      trimStart: seekPosition,
    })

    syncAudioToVideo(clip, player)
    await waitForPlaying(player)
    await playAudio(clip)

    setVideoLoading(false)
    setPlayMode(mode)
    setPlayIdx(index)
    setActiveIdx(index)
    setPlayElapsed(elapsed)
    setPausedAtTime(-1)
  }, [
    clearInactiveVideo,
    loadAndSeek,
    playAudio,
    setActiveIdx,
    stopAll,
    syncAudioToVideo,
    waitForPlaying,
  ])

  const playAll = useCallback(async () => {
    const first = clipsRef.current[0]
    if (!first) return

    await startPlayback(0, first.trimStart, 'all', 0)
  }, [startPlayback])

  const playSingle = useCallback(async (index: number) => {
    const all = clipsRef.current
    const clip = all[index]
    if (!clip) return

    let seekPosition = clip.trimStart
    let elapsed = 0

    if (pausedAtTime >= 0) {
      const position = locateClip(pausedAtTime, all)

      if (position.idx === index) {
        seekPosition = clip.trimStart + Math.min(
          position.offsetInClip,
          Math.max(0, clip.trimEnd - clip.trimStart - 0.1),
        )
        elapsed = pausedAtTime
      }
    }

    await startPlayback(index, seekPosition, 'single', elapsed)
  }, [pausedAtTime, startPlayback])

  const resumeFromTime = useCallback(async (globalTime: number) => {
    const all = clipsRef.current
    const position = locateClip(globalTime, all)

    if (position.idx < 0) return

    if (globalTime >= totalDur - 0.1) {
      await playAll()
      return
    }

    const clip = all[position.idx]
    const seekPosition = clip.trimStart + Math.min(
      position.offsetInClip,
      Math.max(0, clip.trimEnd - clip.trimStart - 0.1),
    )

    await startPlayback(
      position.idx,
      seekPosition,
      'all',
      globalTime,
    )
  }, [playAll, startPlayback, totalDur])

  const togglePlayPause = useCallback(() => {
    if (transActiveRef.current || switchingRef.current) return

    const active = activeVideoRef.current
    if (!active) return

    const audio = audioRef.current

    if (playMode !== 'none' && !active.paused) {
      active.pause()
      audio?.pause()
      narrationRef.current?.pause()
      return
    }

    if (playMode !== 'none' && active.paused) {
      active.play().catch(() => {})
      audio?.play().catch(() => {})
      return
    }

    if (clipsRef.current.length === 0) return

    if (pausedAtTime >= 0) {
      void resumeFromTime(pausedAtTime)
    } else {
      void playAll()
    }
  }, [pausedAtTime, playAll, playMode, resumeFromTime])

  /**
   * 在双video元素之间完成无转场切换或渐变转场。
   */
  const doCrossfade = useCallback((
    fromIndex: number,
    toIndex: number,
  ) => {
    const all = clipsRef.current

    if (toIndex >= all.length) {
      stopAll()
      return
    }

    const from = all[fromIndex]
    const to = all[toIndex]
    const outgoing = activeVideoRef.current
    const incoming = outgoing === videoARef.current
      ? videoBRef.current
      : videoARef.current

    if (!from || !to || !outgoing || !incoming) {
      stopAll()
      return
    }

    const transition = from.transition || 'none'
    const durationMs = Math.max(1, (from.transDur || 0.5) * 1000)

    switchingRef.current = true

    const activateIncoming = () => {
      activeVideoRef.current = incoming
      setTopPlayer(incoming === videoARef.current ? 'A' : 'B')
      setPlayIdx(toIndex)
      setActiveIdx(toIndex)
      switchingRef.current = false
    }

    if (transition === 'none') {
      outgoing.pause()
      outgoing.volume = 1
      stopAudio()

      loadAndSeek(incoming, to)
        .then(() => {
          incoming.volume = 1
          syncAudioToVideo(to, incoming)
          incoming.play().catch(() => {})
          void playAudio(to)
          activateIncoming()
        })
        .catch(stopAll)

      return
    }

    setTransActive(true)
    setTransStyle(transition)
    setTransProgress(0)
    setPlayIdx(toIndex)
    setActiveIdx(toIndex)

    loadAndSeek(incoming, to)
      .then(() => {
        incoming.volume = 1
        syncAudioToVideo(to, incoming)
        incoming.play().catch(() => {})
        void playAudio(to)
      })
      .catch(stopAll)

    const started = performance.now()

    const animate = (now: number) => {
      const progress = Math.min(1, (now - started) / durationMs)
      setTransProgress(progress)

      try {
        outgoing.volume = Math.max(0, 1 - progress)
      } catch {
        // 播放器已卸载。
      }

      if (progress < 1) {
        transRAF.current = requestAnimationFrame(animate)
        return
      }

      outgoing.pause()
      outgoing.volume = 1
      activateIncoming()
      setTransActive(false)
      setTransProgress(0)
      setTransStyle('')
    }

    transRAF.current = requestAnimationFrame(animate)
  }, [
    loadAndSeek,
    playAudio,
    setActiveIdx,
    stopAll,
    stopAudio,
    syncAudioToVideo,
  ])

  /**
   * 50ms播放循环负责裁剪终点、转场、音轨和TTS同步。
   */
  useEffect(() => {
    const check = () => {
      const mode = playModeRef.current
      const index = playIdxRef.current
      const all = clipsRef.current

      if (
        mode === 'none'
        || switchingRef.current
        || transActiveRef.current
        || index < 0
        || index >= all.length
      ) {
        return
      }

      const clip = all[index]
      const active = activeVideoRef.current
      if (!active || !clip) return

      let elapsed = 0

      for (let itemIndex = 0; itemIndex < index; itemIndex += 1) {
        elapsed += all[itemIndex].trimEnd - all[itemIndex].trimStart
      }

      elapsed += Math.max(0, active.currentTime - clip.trimStart)
      setPlayElapsed(elapsed)
      syncNarration(elapsed, active.paused)

      const audio = audioRef.current

      if (audio && clip.audioUrl && !audio.paused) {
        const expected = active.currentTime - clip.trimStart

        if (Math.abs(audio.currentTime - expected) > 0.2) {
          try {
            audio.currentTime = expected
          } catch {
            // 音频元数据未就绪。
          }
        }
      }

      if (mode === 'single') {
        if (active.currentTime >= clip.trimEnd - 0.05) {
          active.pause()
          stopAudio()
          stopNarration()
          setPlayMode('none')
          setPlayIdx(-1)
        }
        return
      }

      if (index >= all.length - 1) {
        if (active.currentTime >= clip.trimEnd - 0.05) {
          stopAll()
        }
        return
      }

      const transition = clip.transition || 'none'
      const transitionDuration = clip.transDur || 0.5
      const transitionPoint = transition === 'none'
        ? clip.trimEnd - 0.05
        : clip.trimEnd - transitionDuration - 0.05

      if (active.currentTime >= transitionPoint) {
        doCrossfade(index, index + 1)
      }
    }

    const timer = window.setInterval(check, 50)

    return () => {
      window.clearInterval(timer)
    }
  }, [
    doCrossfade,
    stopAll,
    stopAudio,
    stopNarration,
    syncNarration,
  ])

  const seekToTime = useCallback(async (globalTime: number) => {
    try {
      const all = clipsRef.current
      const position = locateClip(globalTime, all)

      if (position.idx < 0) return

      const clip = all[position.idx]
      const seekPosition = clip.trimStart + Math.min(
        position.offsetInClip,
        Math.max(0, clip.trimEnd - clip.trimStart - 0.1),
      )

      stopAll()
      setVideoLoading(true)

      const player = videoARef.current

      if (!player) {
        setVideoLoading(false)
        return
      }

      activeVideoRef.current = player
      setTopPlayer('A')
      clearInactiveVideo()

      await loadAndSeek(player, {
        ...clip,
        trimStart: seekPosition,
      })

      syncAudioToVideo(clip, player)
      setVideoLoading(false)
      setActiveIdx(position.idx)
      setPausedAtTime(globalTime)
    } catch (error) {
      console.warn('[视频编辑器标尺定位失败]', error)
      setVideoLoading(false)
    }
  }, [
    clearInactiveVideo,
    loadAndSeek,
    setActiveIdx,
    stopAll,
    syncAudioToVideo,
  ])

  const handleRulerClick = useCallback((
    event: ReactMouseEvent,
  ) => {
    if (draggingPH || !rulerRef.current) return

    const rect = rulerRef.current.getBoundingClientRect()

    void seekToTime(
      pixelToTime(
        Math.max(0, event.clientX - rect.left),
        clipsRef.current,
      ),
    )
  }, [draggingPH, seekToTime])

  const handlePHDown = useCallback((
    event: ReactPointerEvent,
  ) => {
    event.preventDefault()
    event.stopPropagation()
    setDraggingPH(true)
  }, [])

  useEffect(() => {
    if (!draggingPH) return

    const move = (event: PointerEvent) => {
      if (!rulerRef.current) return

      const rect = rulerRef.current.getBoundingClientRect()
      const pixel = Math.max(
        0,
        Math.min(
          event.clientX - rect.left,
          totalPxWidth(clipsRef.current),
        ),
      )

      setPhDragTime(pixelToTime(pixel, clipsRef.current))
    }

    const release = () => setDraggingPH(false)

    document.addEventListener('pointermove', move)
    document.addEventListener('pointerup', release)

    return () => {
      document.removeEventListener('pointermove', move)
      document.removeEventListener('pointerup', release)
    }
  }, [draggingPH])

  useEffect(() => {
    if (previousDraggingPH.current && !draggingPH) {
      void seekToTime(phDragTime)
    }

    previousDraggingPH.current = draggingPH
  }, [draggingPH, phDragTime, seekToTime])

  const queuePlay = useCallback((index: number) => {
    queuedPlayIndex.current = index
  }, [])

  useEffect(() => {
    const index = queuedPlayIndex.current

    if (index < 0 || index >= clips.length) return

    queuedPlayIndex.current = -1
    void playSingle(index)
  }, [clips, playSingle])

  useEffect(() => {
    if (videoARef.current && !activeVideoRef.current) {
      activeVideoRef.current = videoARef.current
    }
  }, [])

  useEffect(() => () => {
    videoARef.current?.pause()
    videoBRef.current?.pause()
    audioRef.current?.pause()
    narrationRef.current?.pause()
    cancelAnimationFrame(transRAF.current)

    clipsRef.current.forEach((clip) => {
      if (!clip.blobUrl) return

      try {
        URL.revokeObjectURL(clip.blobUrl)
      } catch {
        // blob URL可能已由删除片段流程释放。
      }
    })
  }, [])

  return {
    videoARef,
    videoBRef,
    audioRef,
    narrationRef,
    rulerRef,
    topPlayer,
    playMode,
    playIdx,
    playElapsed,
    videoLoading,
    transProgress,
    transActive,
    transStyle,
    draggingPH,
    phDragTime,
    totalDur,
    playheadTime,
    stopAll,
    playSingle,
    playAll,
    togglePlayPause,
    queuePlay,
    handleRulerClick,
    handlePHDown,
  }
}
