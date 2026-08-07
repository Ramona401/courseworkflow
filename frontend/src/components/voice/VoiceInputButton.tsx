/**
 * VoiceInputButton.tsx — 全平台统一语音输入按钮
 *
 * 组件只负责状态展示和动作触发，
 * 不持有麦克风、WebSocket或业务输入框状态。
 */

import { LoaderCircle, Mic, Square, X } from 'lucide-react'
import type { VoiceInputStatus } from '@/hooks/useVoiceInput'

export interface VoiceInputButtonProps {
  status: VoiceInputStatus
  isSupported: boolean
  elapsedSeconds: number
  disabled?: boolean
  error?: string
  onStart: () => void
  onStop: () => void
  onCancel: () => void
}

function formatDuration(seconds: number): string {
  const safe = Math.max(0, Math.trunc(seconds))
  return `${Math.floor(safe / 60)}:${String(safe % 60).padStart(2, '0')}`
}

export default function VoiceInputButton({
  status,
  isSupported,
  elapsedSeconds,
  disabled = false,
  error,
  onStart,
  onStop,
  onCancel,
}: VoiceInputButtonProps) {
  const unavailable = disabled || !isSupported
  const recording = status === 'recording'
  const connecting = status === 'connecting'
  const stopping = status === 'stopping'

  const handleClick = () => {
    if (unavailable || stopping) return
    if (recording) return onStop()
    if (connecting) return onCancel()
    onStart()
  }

  let title = '使用语音输入'
  if (!isSupported) title = '当前浏览器不支持语音输入'
  else if (disabled) title = '当前状态不能使用语音输入'
  else if (recording) title = `正在录音 ${formatDuration(elapsedSeconds)}，点击停止`
  else if (connecting) title = '正在连接语音识别，点击取消'
  else if (stopping) title = '正在整理最终识别文字'
  else if (status === 'error' && error) title = error

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={unavailable || stopping}
      title={title}
      aria-label={title}
      style={{
        width: '34px',
        height: '34px',
        flexShrink: 0,
        borderRadius: '50%',
        border: 'none',
        background: recording
          ? '#EF4444'
          : connecting || stopping
            ? '#EEF2FF'
            : 'transparent',
        color: recording
          ? '#FFFFFF'
          : connecting || stopping
            ? '#4F46E5'
            : '#64748B',
        cursor: unavailable || stopping ? 'not-allowed' : 'pointer',
        opacity: unavailable ? 0.4 : 1,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        transition: 'all 150ms ease',
      }}
    >
      {recording ? (
        <Square size={14} fill="currentColor" />
      ) : connecting ? (
        <X size={16} />
      ) : stopping ? (
        <LoaderCircle
          size={17}
          style={{ animation: 'tednaVoiceSpin 900ms linear infinite' }}
        />
      ) : (
        <Mic size={18} />
      )}

      <style>{`
        @keyframes tednaVoiceSpin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      `}</style>
    </button>
  )
}
