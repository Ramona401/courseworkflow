/**
 * PlatformCoursewareAssistantVoiceControls.tsx
 *
 * 老师端课堂语音控制：
 *   - 复用全平台 useVoiceInput 完成教师JWT语音识别；
 *   - partial文字实时写入备用输入框；
 *   - final文字直接提交教学智能体；
 *   - 开始录音前停止豆包或设备朗读，避免声音回灌；
 *   - 默认使用豆包自然音色，失败时自动降级设备语音；
 *   - 提供自动朗读、暂停、继续、停止和内存重播操作。
 */

import { useEffect } from 'react'

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

export default function PlatformCoursewareAssistantVoiceControls({
  disabled,
  classroomMode,
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
  const voice = useVoiceInput({
    disabled,
    maxDurationSeconds: 60,
    onPartial: text => {
      onInputChange(text)
    },
    onFinal: text => {
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
      return `正在听 ${formatDuration(voice.elapsedSeconds)}，点击结束并发送`
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
      return <Square size={classroomMode ? 22 : 18} fill="currentColor" />
    }

    if (connecting) {
      return <X size={classroomMode ? 23 : 19} />
    }

    if (stopping) {
      return (
        <LoaderCircle
          size={classroomMode ? 24 : 20}
          style={{ animation: 'tednaAssistantVoiceSpin 900ms linear infinite' }}
        />
      )
    }

    return <Mic size={classroomMode ? 27 : 22} />
  })()

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
            ? '1px solid #DC2626'
            : '1px solid rgba(79,123,232,0.38)',
          background: recording
            ? '#EF4444'
            : connecting || stopping
              ? '#EEF2FF'
              : '#4F7BE8',
          color: recording
            ? '#FFFFFF'
            : connecting || stopping
              ? '#4F46E5'
              : '#FFFFFF',
          boxShadow: recording
            ? '0 8px 24px rgba(239,68,68,0.28)'
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
      `}</style>
    </div>
  )
}

