/**
 * ClassroomDigitalHuman.tsx — 老师端课堂卡通数字人表现层。
 *
 * 职责边界：
 * - 只负责人物视觉，不拥有ASR、AI会话或TTS业务逻辑。
 * - 待机、倾听、思考、讲话由上层教学智能体状态显式驱动。
 * - 豆包MP3优先通过HTMLMediaElement.captureStream读取真实音频振幅驱动嘴型。
 * - captureStream不可用或降级设备语音时，只降级为节奏嘴型，不影响课堂语音播放。
 * - 人物素材使用平台自有PNG图集，不依赖第三方数字人SDK或运行时授权。
 */

import { useEffect, useRef, useState } from 'react'

import type { AssistantSpeechProvider } from '@/hooks/useAssistantSpeechPlayback'
import type { PlatformAssistantOverlayVariant } from './PlatformCoursewareAssistantOverlay.styles'

export type ClassroomDigitalHumanState = 'idle' | 'listening' | 'thinking' | 'speaking'

interface ClassroomDigitalHumanProps {
  state: ClassroomDigitalHumanState
  audioElement: HTMLAudioElement | null
  speechProvider: AssistantSpeechProvider
  speechPaused: boolean
  classroomMode: boolean
  variant: PlatformAssistantOverlayVariant
}

interface AtlasPatch {
  x: number
  y: number
  width: number
  height: number
  left: number
  top: number
}

const ATLAS_URL = '/assets/classroom-digital-human/teacher-female-atlas.png'
const ATLAS_WIDTH = 1440
const ATLAS_HEIGHT = 720
const FRAME_WIDTH = 480
const FRAME_HEIGHT = 640

const FRAME_X: Record<ClassroomDigitalHumanState, number> = {
  idle: 0,
  listening: 480,
  thinking: 960,
  speaking: 0,
}

const MOUTH_PATCHES: Record<'small' | 'medium' | 'wide' | 'round', AtlasPatch> = {
  small: { x: 0, y: 650, width: 91, height: 44, left: 194, top: 172 },
  medium: { x: 103, y: 650, width: 91, height: 44, left: 194, top: 172 },
  wide: { x: 206, y: 650, width: 91, height: 44, left: 194, top: 172 },
  round: { x: 309, y: 650, width: 91, height: 44, left: 194, top: 172 },
}

const BLINK_PATCH: AtlasPatch = {
  x: 422,
  y: 650,
  width: 186,
  height: 69,
  left: 144,
  top: 94,
}

type CaptureStreamAudioElement = HTMLAudioElement & {
  captureStream?: () => MediaStream
  mozCaptureStream?: () => MediaStream
}

type AudioContextConstructor = typeof AudioContext

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
  if (variant === 'slideshow') return classroomMode ? 0.38 : 0.29
  if (variant === 'fullscreen') return classroomMode ? 0.35 : 0.27
  return classroomMode ? 0.30 : 0.22
}

function digitalHumanStatus(state: ClassroomDigitalHumanState) {
  switch (state) {
  case 'listening':
    return { text: '正在听老师说话', background: '#EFF6FF', color: '#1D4ED8' }
  case 'thinking':
    return { text: '正在思考', background: '#F5F3FF', color: '#6D28D9' }
  case 'speaking':
    return { text: '正在讲解', background: '#ECFDF5', color: '#047857' }
  default:
    return { text: '课堂助教待机', background: '#F8FAFC', color: '#64748B' }
  }
}

function mouthPatchForLevel(level: number, animationTick: number): AtlasPatch | null {
  if (level < 0.08) return null
  if (level < 0.23) return MOUTH_PATCHES.small
  if (level < 0.42) return MOUTH_PATCHES.medium
  if (level < 0.68) return animationTick % 4 === 0 ? MOUTH_PATCHES.round : MOUTH_PATCHES.wide
  return animationTick % 3 === 0 ? MOUTH_PATCHES.round : MOUTH_PATCHES.wide
}

function patchStyle(patch: AtlasPatch): React.CSSProperties {
  return {
    position: 'absolute',
    left: patch.left,
    top: patch.top,
    width: patch.width,
    height: patch.height,
    pointerEvents: 'none',
    backgroundImage: `url("${ATLAS_URL}")`,
    backgroundRepeat: 'no-repeat',
    backgroundSize: `${ATLAS_WIDTH}px ${ATLAS_HEIGHT}px`,
    backgroundPosition: `-${patch.x}px -${patch.y}px`,
  }
}

export default function ClassroomDigitalHuman({
  state,
  audioElement,
  speechProvider,
  speechPaused,
  classroomMode,
  variant,
}: ClassroomDigitalHumanProps) {
  const [audioLevel, setAudioLevel] = useState(0)
  const [animationTick, setAnimationTick] = useState(0)
  const [blinking, setBlinking] = useState(false)
  const [realAudioTracking, setRealAudioTracking] = useState(false)

  const speakingRef = useRef(false)
  speakingRef.current = state === 'speaking' && !speechPaused

  /**
   * 使用captureStream旁路读取豆包音频，不通过createMediaElementSource接管播放器输出。
   * 这样即使AudioContext受浏览器策略限制，原有豆包声音也不会被数字人口型功能静音。
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

        if (timestamp - lastSampleAt >= 72) {
          lastSampleAt = timestamp

          if (speakingRef.current && !audioElement.paused && !audioElement.ended) {
            analyser.getByteTimeDomainData(samples)

            let squareSum = 0
            for (let index = 0; index < samples.length; index += 1) {
              const centered = (samples[index] - 128) / 128
              squareSum += centered * centered
            }

            const rms = Math.sqrt(squareSum / samples.length)
            const normalized = Math.max(0, Math.min(1, (rms - 0.012) / 0.19))
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
      if (audioContext) void audioContext.close().catch(() => undefined)
    }
  }, [audioElement, speechProvider])

  /** captureStream不可用或设备语音兜底时使用轻量节奏嘴型，不影响语音本身。 */
  useEffect(() => {
    if (state !== 'speaking' || speechPaused || realAudioTracking) {
      if (!realAudioTracking) setAudioLevel(0)
      return
    }

    const timer = setInterval(() => {
      setAudioLevel(0.20 + Math.random() * 0.62)
      setAnimationTick(current => current + 1)
    }, 105)

    return () => {
      clearInterval(timer)
      setAudioLevel(0)
    }
  }, [realAudioTracking, speechPaused, state])

  /**
   * 待机和讲话时自然眨眼。倾听和思考素材已有独立微表情，不叠加中性闭眼帧，
   * 避免不同完整人物状态之间发生面部错位。
   */
  useEffect(() => {
    if (state !== 'idle' && state !== 'speaking') {
      setBlinking(false)
      return
    }

    let blinkTimer: ReturnType<typeof setTimeout> | null = null
    let reopenTimer: ReturnType<typeof setTimeout> | null = null

    const scheduleBlink = () => {
      blinkTimer = setTimeout(() => {
        setBlinking(true)
        reopenTimer = setTimeout(() => {
          setBlinking(false)
          scheduleBlink()
        }, 125)
      }, 3200 + Math.floor(Math.random() * 2600))
    }

    scheduleBlink()

    return () => {
      if (blinkTimer) clearTimeout(blinkTimer)
      if (reopenTimer) clearTimeout(reopenTimer)
    }
  }, [state])

  const scale = digitalHumanScale(variant, classroomMode)
  const status = digitalHumanStatus(state)
  const mouthPatch = state === 'speaking' && !speechPaused
    ? mouthPatchForLevel(audioLevel, animationTick)
    : null
  const displayWidth = Math.round(FRAME_WIDTH * scale)
  const displayHeight = Math.round(FRAME_HEIGHT * scale)

  return (
    <div
      aria-label={`课堂卡通老师：${status.text}`}
      style={{ width: displayWidth, pointerEvents: 'none', userSelect: 'none' }}
    >
      <div
        style={{
          width: displayWidth,
          height: displayHeight,
          overflow: 'hidden',
          borderRadius: classroomMode ? 22 : 16,
          border: '1px solid rgba(148,163,184,0.34)',
          background: '#FFFFFF',
          boxShadow: '0 16px 42px rgba(15,23,42,0.20)',
        }}
      >
        <div
          style={{
            position: 'relative',
            width: FRAME_WIDTH,
            height: FRAME_HEIGHT,
            transform: `scale(${scale})`,
            transformOrigin: 'top left',
            backgroundImage: `url("${ATLAS_URL}")`,
            backgroundRepeat: 'no-repeat',
            backgroundSize: `${ATLAS_WIDTH}px ${ATLAS_HEIGHT}px`,
            backgroundPosition: `-${FRAME_X[state]}px 0px`,
            animation: state === 'idle' || state === 'speaking'
              ? 'tednaDigitalHumanBreathe 4.8s ease-in-out infinite'
              : 'none',
          }}
        >
          {mouthPatch && <div aria-hidden="true" style={patchStyle(mouthPatch)} />}
          {blinking && <div aria-hidden="true" style={patchStyle(BLINK_PATCH)} />}
        </div>
      </div>

      <div
        style={{
          marginTop: classroomMode ? 8 : 6,
          padding: classroomMode ? '6px 10px' : '4px 7px',
          borderRadius: 999,
          background: status.background,
          color: status.color,
          fontSize: classroomMode ? 13 : 10,
          fontWeight: 800,
          lineHeight: 1.3,
          textAlign: 'center',
          boxShadow: '0 6px 18px rgba(15,23,42,0.08)',
        }}
      >
        {status.text}
      </div>

      <style>{`
        @keyframes tednaDigitalHumanBreathe {
          0%, 100% { transform: translateY(0); }
          50% { transform: translateY(-3px); }
        }
      `}</style>
    </div>
  )
}
