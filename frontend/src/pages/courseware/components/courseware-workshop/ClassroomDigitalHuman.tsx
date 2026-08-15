/**
 * ClassroomDigitalHuman.tsx — 老师端课堂卡通数字人表现层。
 *
 * 8.11语义动作版：
 * - 每位老师使用一张统一WebP图集，核心表情与可用动作都在同一资源中；
 * - 页面始终只有一个完整人物DOM层，动作切换只改变background-position；
 * - 不叠加第二个人物、不透明交叉、不重新创建人物节点，避免穿帮与闪白；
 * - 待机偶尔眨眼，AI思考使用思考眉形，朗读结束短暂微笑；
 * - AI朗读以五档口型为主，豆包MP3优先用真实音量驱动；
 * - 8.11动作由独立Hook按回答语义和音频播放进度低频触发；
 * - 男女老师都启用列举、指向课件、展开、邀请、强调、鼓励、鼓掌；
 * - 鼓掌只用于较强正向反馈，普通“很好/不错”优先使用轻鼓励；
 * - 手势期间继续按同一音量口径切换五档完整人物口型帧。
 *
 * 职责边界：
 * - 本组件只负责人像显示、口型、眨眼和单层帧选择；
 * - 动作语义与时间调度放在useClassroomDigitalHumanGesture；
 * - 不拥有ASR、AI会话、TTS请求、计费或人物选择状态。
 */

import { useEffect, useRef, useState } from 'react'

import type { AssistantSpeechProvider } from '@/hooks/useAssistantSpeechPlayback'
import type { PlatformAssistantOverlayVariant } from './PlatformCoursewareAssistantOverlay.styles'
import useClassroomDigitalHumanGesture from './useClassroomDigitalHumanGesture'
import type { ClassroomDigitalHumanGesture } from './useClassroomDigitalHumanGesture'

export type ClassroomDigitalHumanState = 'idle' | 'listening' | 'thinking' | 'speaking'
export type ClassroomDigitalHumanCharacter = 'female' | 'male'

type MouthLevel = 'closed' | 'small' | 'medium' | 'open' | 'round'
type PortraitFrame =
  | 'idle'
  | 'blink'
  | 'thinking'
  | 'small'
  | 'medium'
  | 'open'
  | 'round'
  | 'smile'

interface ClassroomDigitalHumanProps {
  state: ClassroomDigitalHumanState
  character: ClassroomDigitalHumanCharacter
  speechText: string
  audioElement: HTMLAudioElement | null
  speechProvider: AssistantSpeechProvider
  speechPaused: boolean
  classroomMode: boolean
  variant: PlatformAssistantOverlayVariant
}

interface CharacterAtlasConfig {
  url: string
  columns: number
  rows: number
  gestureStarts: Record<ClassroomDigitalHumanGesture, number>
}

type CaptureStreamAudioElement = HTMLAudioElement & {
  captureStream?: () => MediaStream
  mozCaptureStream?: () => MediaStream
}

type AudioContextConstructor = typeof AudioContext

const FRAME_WIDTH = 360
const FRAME_HEIGHT = 480

const CHARACTER_ATLAS: Record<
  ClassroomDigitalHumanCharacter,
  CharacterAtlasConfig
> = {
  female: {
    url: '/assets/classroom-digital-human/teacher-female-unified-v2.webp',
    columns: 8,
    rows: 6,
    gestureStarts: {
      enumerate: 8,
      point_left: 13,
      expand: 18,
      invite: 23,
      emphasis: 28,
      encourage: 33,
      applause: 38,
    },
  },
  male: {
    url: '/assets/classroom-digital-human/teacher-male-unified-v2.webp',
    columns: 8,
    rows: 6,
    gestureStarts: {
      enumerate: 8,
      point_left: 13,
      expand: 18,
      invite: 23,
      emphasis: 28,
      encourage: 33,
      applause: 38,
    },
  },
}

const CORE_FRAME_INDEX: Record<PortraitFrame, number> = {
  idle: 0,
  blink: 1,
  thinking: 2,
  small: 3,
  medium: 4,
  open: 5,
  round: 6,
  smile: 7,
}

const MOUTH_OFFSET: Record<MouthLevel, number> = {
  closed: 0,
  small: 1,
  medium: 2,
  open: 3,
  round: 4,
}

function resolveAudioContextConstructor(): AudioContextConstructor | null {
  if (typeof window === 'undefined') return null

  const candidate = window as typeof window & {
    webkitAudioContext?: AudioContextConstructor
  }

  return window.AudioContext || candidate.webkitAudioContext || null
}

function digitalHumanScale(
  variant: PlatformAssistantOverlayVariant,
  classroomMode: boolean,
): number {
  if (variant === 'slideshow') return classroomMode ? 0.67 : 0.57
  if (variant === 'fullscreen') return classroomMode ? 0.65 : 0.55
  return classroomMode ? 0.47 : 0.39
}

function mouthLevelForAudio(
  level: number,
  animationTick: number,
): MouthLevel {
  if (level < 0.09) return 'closed'
  if (level < 0.28) return 'small'
  if (level < 0.48) return 'medium'

  if (level < 0.72) {
    return animationTick % 5 === 0 ? 'round' : 'open'
  }

  return animationTick % 3 === 0 ? 'round' : 'open'
}

function frameForMouth(level: MouthLevel): PortraitFrame {
  switch (level) {
  case 'small':
    return 'small'
  case 'medium':
    return 'medium'
  case 'open':
    return 'open'
  case 'round':
    return 'round'
  default:
    return 'idle'
  }
}

function frameIndexForDisplay(
  atlas: CharacterAtlasConfig,
  displayFrame: PortraitFrame,
  activeGesture: ClassroomDigitalHumanGesture | null,
  mouthLevel: MouthLevel,
): number {
  if (activeGesture) {
    return atlas.gestureStarts[activeGesture] + MOUTH_OFFSET[mouthLevel]
  }

  return CORE_FRAME_INDEX[displayFrame]
}

function atlasFrameStyle(
  atlas: CharacterAtlasConfig,
  frameIndex: number,
): React.CSSProperties {
  const safeIndex = Math.max(
    0,
    Math.min(frameIndex, atlas.columns * atlas.rows - 1),
  )
  const column = safeIndex % atlas.columns
  const row = Math.floor(safeIndex / atlas.columns)

  return {
    position: 'absolute',
    inset: 0,
    width: FRAME_WIDTH,
    height: FRAME_HEIGHT,
    backgroundImage: `url("${atlas.url}")`,
    backgroundRepeat: 'no-repeat',
    backgroundSize: `${atlas.columns * FRAME_WIDTH}px ${atlas.rows * FRAME_HEIGHT}px`,
    backgroundPosition: `-${column * FRAME_WIDTH}px -${row * FRAME_HEIGHT}px`,
    opacity: 1,
    transition: 'none',
    willChange: 'background-position',
    pointerEvents: 'none',
  }
}

export default function ClassroomDigitalHuman({
  state,
  character,
  speechText,
  audioElement,
  speechProvider,
  speechPaused,
  classroomMode,
  variant,
}: ClassroomDigitalHumanProps) {
  const [audioLevel, setAudioLevel] = useState(0)
  const [animationTick, setAnimationTick] = useState(0)
  const [realAudioTracking, setRealAudioTracking] = useState(false)
  const [blinkClosed, setBlinkClosed] = useState(false)
  const [endingSmile, setEndingSmile] = useState(false)

  const speakingRef = useRef(false)
  const previousStateRef = useRef<ClassroomDigitalHumanState>(state)

  speakingRef.current = state === 'speaking' && !speechPaused

  const atlas = CHARACTER_ATLAS[character]
  const gesture = useClassroomDigitalHumanGesture({
    state,
    character,
    speechText,
    speechPaused,
    audioElement,
  })

  /**
   * 当前人物全部核心表情和动作都在同一张图集。
   * 先异步解码，后续口型和动作只移动background-position。
   */
  useEffect(() => {
    if (typeof Image === 'undefined') return

    const preload = new Image()
    preload.decoding = 'async'
    preload.src = atlas.url

    if (typeof preload.decode === 'function') {
      void preload.decode().catch(() => undefined)
    }
  }, [atlas.url])

  /** 回答真正朗读结束后短暂微笑；暂停时禁止误触发。 */
  useEffect(() => {
    const previousState = previousStateRef.current
    previousStateRef.current = state

    if (
      previousState === 'speaking'
      && state === 'idle'
      && !speechPaused
    ) {
      setEndingSmile(true)

      const timer = window.setTimeout(() => {
        setEndingSmile(false)
      }, 720)

      return () => {
        window.clearTimeout(timer)
      }
    }

    setEndingSmile(false)
  }, [
    speechPaused,
    state,
  ])

  /** 待机和倾听阶段偶尔眨眼；讲话和思考时不与其他高频表情竞争。 */
  useEffect(() => {
    setBlinkClosed(false)

    if (
      state === 'speaking'
      || state === 'thinking'
      || endingSmile
    ) {
      return
    }

    let cancelled = false
    let blinkTimer = 0
    let reopenTimer = 0

    const scheduleBlink = () => {
      const delay = 3200 + Math.floor(Math.random() * 2500)

      blinkTimer = window.setTimeout(() => {
        if (cancelled) return

        setBlinkClosed(true)

        reopenTimer = window.setTimeout(() => {
          if (cancelled) return

          setBlinkClosed(false)
          scheduleBlink()
        }, 110 + Math.floor(Math.random() * 55))
      }, delay)
    }

    scheduleBlink()

    return () => {
      cancelled = true
      window.clearTimeout(blinkTimer)
      window.clearTimeout(reopenTimer)
      setBlinkClosed(false)
    }
  }, [
    character,
    endingSmile,
    state,
  ])

  /**
   * 使用captureStream旁路读取豆包音频，不通过createMediaElementSource接管播放器输出。
   * 即使AudioContext受浏览器策略限制，原有豆包声音也不会被数字人口型功能静音。
   */
  useEffect(() => {
    setRealAudioTracking(false)

    if (!audioElement || speechProvider !== 'doubao') return

    const captureTarget = audioElement as CaptureStreamAudioElement
    const capture = captureTarget.captureStream || captureTarget.mozCaptureStream
    const AudioContextClass = resolveAudioContextConstructor()

    if (!capture || !AudioContextClass) return

    let cancelled = false
    let animationFrame = 0
    let lastSampleAt = 0
    let audioContext: AudioContext | null = null
    let stream: MediaStream | null = null
    let source: MediaStreamAudioSourceNode | null = null
    let analyser: AnalyserNode | null = null

    try {
      stream = capture.call(captureTarget)
      audioContext = new AudioContextClass()
      source = audioContext.createMediaStreamSource(stream)
      analyser = audioContext.createAnalyser()
      analyser.fftSize = 256
      analyser.smoothingTimeConstant = 0.72
      source.connect(analyser)

      void audioContext.resume().catch(() => undefined)

      const samples = new Uint8Array(analyser.fftSize)

      const sample = (timestamp: number) => {
        if (cancelled || !analyser) return

        if (timestamp - lastSampleAt >= 78) {
          lastSampleAt = timestamp

          if (
            speakingRef.current
            && !audioElement.paused
            && !audioElement.ended
          ) {
            analyser.getByteTimeDomainData(samples)

            let squareSum = 0

            for (let index = 0; index < samples.length; index += 1) {
              const centered = (samples[index] - 128) / 128
              squareSum += centered * centered
            }

            const rms = Math.sqrt(squareSum / samples.length)
            const normalized = Math.max(
              0,
              Math.min(1, (rms - 0.012) / 0.19),
            )

            setAudioLevel(normalized)
            setAnimationTick(current => current + 1)
          } else {
            setAudioLevel(0)
          }
        }

        animationFrame = requestAnimationFrame(sample)
      }

      setRealAudioTracking(true)
      animationFrame = requestAnimationFrame(sample)
    } catch {
      setRealAudioTracking(false)
    }

    return () => {
      cancelled = true
      cancelAnimationFrame(animationFrame)

      try {
        source?.disconnect()
        analyser?.disconnect()
      } catch {
        // 浏览器可能已经主动释放媒体分析节点。
      }

      stream?.getTracks().forEach(track => track.stop())

      if (audioContext) {
        void audioContext.close().catch(() => undefined)
      }
    }
  }, [
    audioElement,
    speechProvider,
  ])

  /** captureStream不可用或设备语音兜底时只模拟口型节奏，不切换人物身体。 */
  useEffect(() => {
    if (state !== 'speaking' || speechPaused || realAudioTracking) {
      if (!realAudioTracking) setAudioLevel(0)
      return
    }

    const timer = window.setInterval(() => {
      setAudioLevel(0.16 + Math.random() * 0.66)
      setAnimationTick(current => current + 1)
    }, 112)

    return () => {
      window.clearInterval(timer)
      setAudioLevel(0)
    }
  }, [
    realAudioTracking,
    speechPaused,
    state,
  ])

  const scale = digitalHumanScale(variant, classroomMode)
  const displayWidth = Math.round(FRAME_WIDTH * scale)
  const displayHeight = Math.round(FRAME_HEIGHT * scale)

  const mouthLevel = state === 'speaking' && !speechPaused
    ? mouthLevelForAudio(audioLevel, animationTick)
    : 'closed'

  let displayFrame: PortraitFrame = 'idle'

  if (endingSmile) {
    displayFrame = 'smile'
  } else if (state === 'thinking') {
    displayFrame = 'thinking'
  } else if (state === 'speaking' && !speechPaused) {
    displayFrame = frameForMouth(mouthLevel)
  } else if (blinkClosed) {
    displayFrame = 'blink'
  }

  const activeGesture = state === 'speaking' && !speechPaused
    ? gesture
    : null

  const frameIndex = frameIndexForDisplay(
    atlas,
    displayFrame,
    activeGesture,
    mouthLevel,
  )

  const calmMotion = state === 'idle' || state === 'listening'

  return (
    <div
      aria-label={`课堂卡通老师：${character === 'male' ? '男老师' : '女老师'}`}
      data-character={character}
      data-state={state}
      data-frame={displayFrame}
      data-gesture={activeGesture || 'none'}
      data-frame-index={frameIndex}
      data-speaking-text-length={speechText.length}
      style={{
        width: displayWidth,
        pointerEvents: 'none',
        userSelect: 'none',
      }}
    >
      <div
        style={{
          width: displayWidth,
          height: displayHeight,
          overflow: 'visible',
          animation: calmMotion
            ? 'tednaDigitalHumanCalm 5.8s ease-in-out infinite'
            : 'none',
        }}
      >
        <div
          style={{
            position: 'relative',
            width: FRAME_WIDTH,
            height: FRAME_HEIGHT,
            transform: `scale(${scale})`,
            transformOrigin: 'top left',
            filter: 'drop-shadow(0 14px 16px rgba(15,23,42,0.16))',
          }}
        >
          <div
            aria-hidden="true"
            style={atlasFrameStyle(
              atlas,
              frameIndex,
            )}
          />
        </div>
      </div>

      <style>{`
        @keyframes tednaDigitalHumanCalm {
          0%, 100% { transform: translateY(0); }
          50% { transform: translateY(-1px); }
        }
      `}</style>
    </div>
  )
}

