/**
 * PlatformCoursewareAssistantVoiceControls.tsx
 *
 * 老师端课堂语音控制：
 * - 复用全平台 useVoiceInput 完成教师JWT语音识别；
 * - partial文字实时写入备用输入框，final文字直接提交教学智能体；
 * - 开始录音前停止豆包或设备朗读，避免声音回灌；
 * - minimal模式只显示一个麦克风状态按钮，供全屏/放映的极简课堂界面使用；
 * - 完整模式继续提供自动朗读、暂停、继续、停止和内存重播操作。
 */

import { useEffect, useState } from 'react'

import {
  LoaderCircle,
  Mic,
  Pause,
  Play,
  RotateCcw,
  Square,
  Volume2,
  VolumeX,
  X,
} from 'lucide-react'

import { useVoiceInput } from '@/hooks/useVoiceInput'
import type { VoiceInputStatus } from '@/hooks/useVoiceInput'
import type { AssistantSpeechProvider } from '@/hooks/useAssistantSpeechPlayback'

interface PlatformCoursewareAssistantVoiceControlsProps {
  disabled: boolean
  classroomMode: boolean
  minimal?: boolean
  autoSpeak: boolean
  speechSupported: boolean
  speechPreparing: boolean
  speechSpeaking: boolean
  speechPaused: boolean
  speechError: string
  speechWarning: string
  speechProvider: AssistantSpeechProvider
  speechVoiceLabel: string
  canReplay: boolean
  onInputChange: (text: string) => void
  onSubmitVoice: (text: string) => boolean
  onBeforeVoiceStart: () => void
  onToggleAutoSpeak: () => void
  onToggleSpeechPause: () => void
  onStopSpeech: () => void
  onReplaySpeech: () => void
  onVoiceStatusChange?: (status: VoiceInputStatus) => void
}

function formatDuration(seconds: number): string {
  const safeSeconds = Math.max(0, Math.trunc(seconds))
  const minutes = Math.floor(safeSeconds / 60)
  const remainder = String(safeSeconds % 60).padStart(2, '0')

  return `${minutes}:${remainder}`
}

function VoiceWaveform({
  heardSpeech,
  classroomMode,
  minimal,
}: {
  heardSpeech: boolean
  classroomMode: boolean
  minimal: boolean
}) {
  const bars = [0.5, 0.72, 0.92, 0.64, 1, 0.68, 0.88, 0.7, 0.48]
  const maximumHeight = minimal
    ? classroomMode ? 34 : 28
    : classroomMode ? 28 : 22
  const barWidth = minimal
    ? classroomMode ? 5 : 4
    : classroomMode ? 4 : 3

  return (
    <span
      aria-hidden="true"
      style={{
        height: maximumHeight,
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: minimal ? 3 : 2.5,
        flex: '0 0 auto',
      }}
    >
      {bars.map((ratio, index) => (
        <span
          key={`${ratio}-${index}`}
          style={{
            width: barWidth,
            height: heardSpeech
              ? Math.max(7, Math.round(maximumHeight * ratio))
              : Math.max(4, Math.round(maximumHeight * 0.16)),
            borderRadius: 999,
            background: 'currentColor',
            opacity: heardSpeech ? 1 : 0.5,
            transformOrigin: 'center',
            transition: 'height 120ms ease, opacity 120ms ease',
            animation: heardSpeech
              ? `tednaAssistantWaveBar ${520 + index * 35}ms ease-in-out infinite alternate`
              : 'none',
            animationDelay: `-${index * 55}ms`,
          }}
        />
      ))}
    </span>
  )
}

export default function PlatformCoursewareAssistantVoiceControls({
  disabled,
  classroomMode,
  minimal = false,
  autoSpeak,
  speechSupported,
  speechPreparing,
  speechSpeaking,
  speechPaused,
  speechError,
  speechWarning,
  speechProvider,
  speechVoiceLabel,
  canReplay,
  onInputChange,
  onSubmitVoice,
  onBeforeVoiceStart,
  onToggleAutoSpeak,
  onToggleSpeechPause,
  onStopSpeech,
  onReplaySpeech,
  onVoiceStatusChange,
}: PlatformCoursewareAssistantVoiceControlsProps) {
  const [heardSpeech, setHeardSpeech] = useState(false)

  const voice = useVoiceInput({
    disabled,
    maxDurationSeconds: 60,
    onPartial: text => {
      if (text.trim()) {
        setHeardSpeech(true)
      }

      onInputChange(text)
    },
    onFinal: text => {
      if (text.trim()) {
        setHeardSpeech(true)
      }

      onInputChange(text)

      if (onSubmitVoice(text)) {
        onInputChange('')
      }
    },
  })

  useEffect(() => {
    onVoiceStatusChange?.(voice.status)
  }, [
    onVoiceStatusChange,
    voice.status,
  ])

  useEffect(() => {
    if (voice.status !== 'recording') {
      setHeardSpeech(false)
    }
  }, [voice.status])

  useEffect(() => {
    return () => {
      onVoiceStatusChange?.('idle')
    }
  }, [onVoiceStatusChange])

  const recording = voice.status === 'recording'
  const connecting = voice.status === 'connecting'
  const stopping = voice.status === 'stopping'
  const voiceUnavailable = disabled || !voice.isSupported

  const handleVoiceClick = () => {
    if (voiceUnavailable || stopping) {
      return
    }

    if (recording) {
      voice.stop()
      return
    }

    if (connecting) {
      voice.cancel()
      return
    }

    onBeforeVoiceStart()
    void voice.start()
  }

  const voiceLabel = (() => {
    if (!voice.isSupported) {
      return '浏览器不支持语音输入'
    }

    if (disabled) {
      return '回答生成中，请稍候'
    }

    if (recording) {
      return heardSpeech
        ? `已听到声音 ${formatDuration(voice.elapsedSeconds)} · 点击结束并发送`
        : `正在听… ${formatDuration(voice.elapsedSeconds)} · 请直接说话`
    }

    if (connecting) {
      return '正在连接麦克风，点击取消'
    }

    if (stopping) {
      return '正在识别并准备发送'
    }

    return '点击说话'
  })()

  const voiceIcon = (() => {
    if (recording) {
      return (
        <VoiceWaveform
          heardSpeech={heardSpeech}
          classroomMode={classroomMode}
          minimal={minimal}
        />
      )
    }

    if (connecting) {
      return (
        <X
          size={minimal
            ? classroomMode ? 34 : 29
            : classroomMode ? 23 : 19}
        />
      )
    }

    if (stopping) {
      return (
        <LoaderCircle
          size={minimal
            ? classroomMode ? 34 : 29
            : classroomMode ? 24 : 20}
          style={{ animation: 'tednaAssistantVoiceSpin 900ms linear infinite' }}
        />
      )
    }

    return (
      <Mic
        size={minimal
          ? classroomMode ? 38 : 32
          : classroomMode ? 27 : 22}
      />
    )
  })()

  if (minimal) {
    const compactError = voice.error || speechError
    const buttonSize = classroomMode ? 88 : 74
    const listeningWidth = classroomMode ? 238 : 204

    return (
      <div
        role="group"
        aria-label="教学智能体课堂麦克风"
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 7,
        }}
      >
        <button
          type="button"
          onClick={handleVoiceClick}
          disabled={voiceUnavailable || stopping}
          aria-label={voiceLabel}
          title={voiceLabel}
          style={{
            width: recording ? listeningWidth : buttonSize,
            height: recording
              ? classroomMode ? 76 : 64
              : buttonSize,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: recording
              ? classroomMode ? '10px 18px' : '8px 14px'
              : 0,
            borderRadius: recording
              ? classroomMode ? 24 : 20
              : '50%',
            border: recording
              ? `2px solid ${heardSpeech
                  ? 'rgba(167,243,208,0.92)'
                  : 'rgba(199,210,254,0.92)'}`
              : '2px solid rgba(255,255,255,0.82)',
            background: recording
              ? heardSpeech
                ? 'linear-gradient(135deg, #0F766E, #0D9488)'
                : 'linear-gradient(135deg, #4338CA, #4F46E5)'
              : connecting || stopping
                ? 'rgba(238,242,255,0.97)'
                : '#4F7BE8',
            color: recording
              ? '#FFFFFF'
              : connecting || stopping
                ? '#4F46E5'
                : '#FFFFFF',
            boxShadow: recording
              ? heardSpeech
                ? '0 0 0 8px rgba(13,148,136,0.12), 0 16px 38px rgba(15,118,110,0.26)'
                : '0 0 0 8px rgba(79,70,229,0.10), 0 16px 38px rgba(49,46,129,0.24)'
              : '0 0 0 7px rgba(79,123,232,0.12), 0 16px 38px rgba(30,64,175,0.28)',
            cursor: voiceUnavailable || stopping ? 'not-allowed' : 'pointer',
            opacity: voiceUnavailable ? 0.5 : 1,
            transition: 'all 180ms ease',
          }}
        >
          {recording ? (
            <span
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: classroomMode ? 13 : 10,
              }}
            >
              {voiceIcon}

              <span
                style={{
                  minWidth: 0,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'flex-start',
                  lineHeight: 1.2,
                }}
              >
                <strong
                  style={{
                    fontSize: classroomMode ? 15 : 12,
                    fontWeight: 900,
                    whiteSpace: 'nowrap',
                  }}
                >
                  {heardSpeech ? '已听到声音' : '正在听…'}
                </strong>

                <span
                  style={{
                    marginTop: 4,
                    color: 'rgba(255,255,255,0.82)',
                    fontSize: classroomMode ? 11.5 : 9.5,
                    fontWeight: 700,
                    whiteSpace: 'nowrap',
                  }}
                >
                  {formatDuration(voice.elapsedSeconds)} · 点击结束并发送
                </span>
              </span>
            </span>
          ) : (
            voiceIcon
          )}
        </button>

        {recording && (
          <div
            aria-live="polite"
            style={{
              color: heardSpeech ? '#0F766E' : '#4338CA',
              fontSize: classroomMode ? 12 : 10,
              fontWeight: 800,
              textShadow: '0 1px 8px rgba(255,255,255,0.95)',
            }}
          >
            {heardSpeech
              ? '声音已进入识别，智能体正在听'
              : '请直接说话，听到后声波会亮起'}
          </div>
        )}

        {compactError && (
          <div
            aria-live="polite"
            style={{
              maxWidth: classroomMode ? 300 : 250,
              padding: '6px 9px',
              borderRadius: 9,
              background: 'rgba(254,242,242,0.96)',
              color: '#B91C1C',
              boxShadow: '0 6px 18px rgba(15,23,42,0.10)',
              fontSize: classroomMode ? 12 : 10,
              lineHeight: 1.45,
              textAlign: 'center',
            }}
          >
            {compactError}
          </div>
        )}

        <style>{`
          @keyframes tednaAssistantVoiceSpin {
            from { transform: rotate(0deg); }
            to { transform: rotate(360deg); }
          }

          @keyframes tednaAssistantWaveBar {
            0% { transform: scaleY(0.42); }
            100% { transform: scaleY(1); }
          }
        `}</style>
      </div>
    )
  }

  const secondaryButtonStyle: React.CSSProperties = {
    minHeight: classroomMode ? 44 : 34,
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 7,
    padding: classroomMode ? '9px 14px' : '7px 10px',
    borderRadius: classroomMode ? 12 : 9,
    border: '1px solid #CBD5E1',
    background: '#FFFFFF',
    color: '#475569',
    fontSize: classroomMode ? 15 : 11,
    fontWeight: 750,
    cursor: 'pointer',
  }

  const speechStatus = (() => {
    if (speechPreparing) {
      return {
        text: '正在生成豆包自然语音…',
        background: '#EFF6FF',
        color: '#1D4ED8',
      }
    }

    if (speechProvider === 'doubao' && speechVoiceLabel) {
      return {
        text: `${speechSpeaking ? '正在播放' : '当前音色'}：豆包 · ${speechVoiceLabel}`,
        background: '#ECFDF5',
        color: '#047857',
      }
    }

    if (speechProvider === 'browser') {
      return {
        text: `${speechSpeaking ? '正在播放' : '当前音色'}：设备语音兜底${speechVoiceLabel ? ` · ${speechVoiceLabel}` : ''}`,
        background: '#FFFBEB',
        color: '#B45309',
      }
    }

    return null
  })()

  return (
    <div
      role="group"
      aria-label="教学智能体课堂语音控制"
      style={{
        padding: classroomMode ? 14 : 10,
        borderRadius: classroomMode ? 16 : 12,
        border: '1px solid #DBEAFE',
        background: 'linear-gradient(145deg, #F8FAFF, #EFF6FF)',
      }}
    >
      <button
        type="button"
        onClick={handleVoiceClick}
        disabled={voiceUnavailable || stopping}
        aria-label={voiceLabel}
        title={voiceLabel}
        style={{
          width: '100%',
          minHeight: classroomMode ? 60 : 46,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: classroomMode ? 12 : 9,
          padding: classroomMode ? '12px 18px' : '9px 14px',
          borderRadius: classroomMode ? 16 : 12,
          border: recording
            ? heardSpeech
              ? '1px solid #0D9488'
              : '1px solid #6366F1'
            : '1px solid rgba(79,123,232,0.38)',
          background: recording
            ? heardSpeech
              ? 'linear-gradient(135deg, #0F766E, #0D9488)'
              : 'linear-gradient(135deg, #4338CA, #4F46E5)'
            : connecting || stopping
              ? '#EEF2FF'
              : '#4F7BE8',
          color: recording
            ? '#FFFFFF'
            : connecting || stopping
              ? '#4F46E5'
              : '#FFFFFF',
          boxShadow: recording
            ? heardSpeech
              ? '0 8px 24px rgba(13,148,136,0.28)'
              : '0 8px 24px rgba(79,70,229,0.24)'
            : '0 8px 24px rgba(79,123,232,0.22)',
          fontSize: classroomMode ? 18 : 13,
          fontWeight: 850,
          cursor: voiceUnavailable || stopping ? 'not-allowed' : 'pointer',
          opacity: voiceUnavailable ? 0.48 : 1,
        }}
      >
        {voiceIcon}
        <span>{voiceLabel}</span>
      </button>

      <div
        style={{
          marginTop: classroomMode ? 10 : 7,
          color: '#64748B',
          fontSize: classroomMode ? 14 : 10,
          lineHeight: 1.55,
          textAlign: 'center',
        }}
      >
        识别完成后会自动发送。开始说话前，系统会停止正在播放的豆包回答。
      </div>

      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: classroomMode ? 9 : 7,
          marginTop: classroomMode ? 12 : 9,
        }}
      >
        <button
          type="button"
          onClick={onToggleAutoSpeak}
          disabled={!speechSupported}
          aria-pressed={autoSpeak}
          style={{
            ...secondaryButtonStyle,
            borderColor: autoSpeak ? '#4F7BE8' : '#CBD5E1',
            background: autoSpeak ? '#EEF2FF' : '#FFFFFF',
            color: autoSpeak ? '#4338CA' : '#64748B',
            cursor: speechSupported ? 'pointer' : 'not-allowed',
            opacity: speechSupported ? 1 : 0.45,
          }}
        >
          {autoSpeak ? (
            <Volume2 size={classroomMode ? 20 : 16} />
          ) : (
            <VolumeX size={classroomMode ? 20 : 16} />
          )}
          豆包自动朗读：{autoSpeak ? '开' : '关'}
        </button>

        {speechSpeaking && (
          <>
            <button
              type="button"
              onClick={onToggleSpeechPause}
              style={secondaryButtonStyle}
            >
              {speechPaused ? (
                <Play size={classroomMode ? 19 : 16} />
              ) : (
                <Pause size={classroomMode ? 19 : 16} />
              )}
              {speechPaused ? '继续朗读' : '暂停朗读'}
            </button>

            <button
              type="button"
              onClick={onStopSpeech}
              style={secondaryButtonStyle}
            >
              <Square size={classroomMode ? 17 : 14} fill="currentColor" />
              停止朗读
            </button>
          </>
        )}

        <button
          type="button"
          onClick={onReplaySpeech}
          disabled={!canReplay || speechPreparing}
          style={{
            ...secondaryButtonStyle,
            cursor: canReplay && !speechPreparing ? 'pointer' : 'not-allowed',
            opacity: canReplay && !speechPreparing ? 1 : 0.42,
          }}
        >
          <RotateCcw size={classroomMode ? 19 : 16} />
          重新朗读
        </button>
      </div>

      {speechStatus && (
        <div
          aria-live="polite"
          style={{
            marginTop: 9,
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: classroomMode ? '9px 11px' : '7px 9px',
            borderRadius: 9,
            background: speechStatus.background,
            color: speechStatus.color,
            fontSize: classroomMode ? 14 : 10,
            lineHeight: 1.55,
          }}
        >
          {speechPreparing && (
            <LoaderCircle
              size={classroomMode ? 18 : 15}
              style={{ animation: 'tednaAssistantVoiceSpin 900ms linear infinite' }}
            />
          )}
          <span>{speechStatus.text}</span>
        </div>
      )}

      {speechWarning && (
        <div
          style={{
            marginTop: 8,
            padding: classroomMode ? '9px 11px' : '7px 9px',
            borderRadius: 9,
            background: '#FFFBEB',
            color: '#B45309',
            fontSize: classroomMode ? 14 : 10,
            lineHeight: 1.55,
          }}
        >
          {speechWarning}
        </div>
      )}

      {!speechSupported && (
        <div
          style={{
            marginTop: 8,
            color: '#B45309',
            fontSize: classroomMode ? 14 : 10,
            lineHeight: 1.5,
          }}
        >
          当前浏览器不支持自动朗读，教学智能体回答仍会以大字显示。
        </div>
      )}

      {(voice.error || speechError) && (
        <div
          aria-live="polite"
          style={{
            marginTop: 8,
            padding: classroomMode ? '9px 11px' : '7px 9px',
            borderRadius: 9,
            background: '#FEF2F2',
            color: '#B91C1C',
            fontSize: classroomMode ? 14 : 10,
            lineHeight: 1.55,
          }}
        >
          {voice.error || speechError}
        </div>
      )}

      <style>{`
        @keyframes tednaAssistantVoiceSpin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }

        @keyframes tednaAssistantWaveBar {
          0% { transform: scaleY(0.42); }
          100% { transform: scaleY(1); }
        }
      `}</style>
    </div>
  )
}
