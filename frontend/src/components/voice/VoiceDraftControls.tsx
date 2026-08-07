/**
 * VoiceDraftControls.tsx — 自然语言草稿的统一语音按钮与状态行
 *
 * 适用范围：
 * - AI对话消息；
 * - AI微调、追改和生成描述；
 * - AI审核讨论等自然语言输入。
 *
 * 组件只负责展示和转发语音控制：
 * - 识别结果如何写入草稿，由useVoiceDraftInput负责；
 * - 不持有业务文本；
 * - 不触发发送、生成、确认或保存动作；
 * - 活动录音始终允许停止或取消，即使外部业务状态随后变为禁用。
 */

import type {
  CSSProperties,
} from 'react'

import type {
  UseVoiceDraftInputResult,
} from '@/hooks/useVoiceDraftInput'

import VoiceInputButton from './VoiceInputButton'

interface VoiceDraftControlsProps {
  /** useVoiceDraftInput返回的完整语音状态。 */
  voice: UseVoiceDraftInputResult

  /** 外部业务是否禁止开始新录音。 */
  disabled?: boolean

  /** 空闲状态下的说明文字。 */
  idleText?: string

  /** 录音或处理中状态文字颜色。 */
  accentColor?: string

  /** 错误状态文字颜色。 */
  errorColor?: string

  /** 外层附加样式。 */
  style?: CSSProperties
}

export default function VoiceDraftControls({
  voice,
  disabled = false,
  idleText =
    '点击麦克风可语音输入；识别文字不会自动提交',
  accentColor = '#4F7BE8',
  errorColor = '#DC2626',
  style,
}: VoiceDraftControlsProps) {
  const text =
    voice.statusText ||
    idleText

  const textColor =
    voice.status === 'error'
      ? errorColor
      : voice.isActive
        ? accentColor
        : '#94A3B8'

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        flexWrap: 'wrap',
        marginTop: 7,
        ...style,
      }}
    >
      <VoiceInputButton
        status={voice.status}
        isSupported={voice.isSupported}
        elapsedSeconds={voice.elapsedSeconds}
        /**
         * 已经开始录音后必须继续允许老师停止或取消，
         * 因此外部disabled只限制空闲态开始录音。
         */
        disabled={
          disabled &&
          !voice.isActive
        }
        error={voice.error}
        onStart={voice.begin}
        onStop={voice.stop}
        onCancel={voice.cancel}
      />

      <span
        style={{
          flex: '1 1 180px',
          minWidth: 0,
          color: textColor,
          fontSize: 11,
          lineHeight: 1.5,
          overflowWrap: 'anywhere',
        }}
      >
        {text}
      </span>
    </div>
  )
}
