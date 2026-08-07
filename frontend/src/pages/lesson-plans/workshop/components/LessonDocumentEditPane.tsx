/**
 * LessonDocumentEditPane.tsx — 教案完整正文输入区。
 *
 * 只负责编辑态文本框、拖拽、粘贴和字数提示。
 * 保存、上传与版本冲突判断由父组件统一控制。
 */

import type {
  ClipboardEvent,
  DragEvent,
  KeyboardEvent,
  RefObject,
} from 'react'

interface LessonDocumentEditPaneProps {
  compact: boolean
  textareaRef:
    RefObject<HTMLTextAreaElement | null>
  draft: string
  saving: boolean
  uploading: boolean
  disabled: boolean
  externalChanged: boolean
  charCount: number
  onDraftChange: (
    value: string,
  ) => void
  onDrop: (
    event:
      DragEvent<HTMLTextAreaElement>,
  ) => void
  onPaste: (
    event:
      ClipboardEvent<HTMLTextAreaElement>,
  ) => void
  onKeyDown: (
    event:
      KeyboardEvent<HTMLTextAreaElement>,
  ) => void
}

export default function LessonDocumentEditPane({
  compact,
  textareaRef,
  draft,
  saving,
  uploading,
  disabled,
  externalChanged,
  charCount,
  onDraftChange,
  onDrop,
  onPaste,
  onKeyDown,
}: LessonDocumentEditPaneProps) {
  const inputDisabled =
    saving ||
    uploading ||
    disabled

  return (
    <div style={{
      height: '100%',
      display: 'flex',
      flexDirection: 'column',
      minHeight: compact
        ? '300px'
        : '420px',
      padding: compact
        ? '12px 14px'
        : '16px 20px',
      boxSizing: 'border-box',
    }}>
      <textarea
        ref={textareaRef}
        value={draft}
        onChange={event =>
          onDraftChange(
            event.target.value,
          )
        }
        onDrop={onDrop}
        onDragOver={event =>
          event.preventDefault()
        }
        onPaste={onPaste}
        onKeyDown={onKeyDown}
        disabled={inputDisabled}
        placeholder="在这里直接编辑完整教案正文，支持Markdown格式……"
        style={{
          width: '100%',
          flex: 1,
          minHeight: compact
            ? '300px'
            : '420px',
          padding: '12px 14px',
          boxSizing: 'border-box',
          border: `1px solid ${
            externalChanged
              ? '#F97316'
              : '#D1D5DB'
          }`,
          borderRadius: '9px',
          outline: 'none',
          resize: 'none',
          fontFamily: 'inherit',
          fontSize: compact
            ? '13px'
            : '14px',
          lineHeight: 1.75,
          color: '#1F2937',
          background:
            inputDisabled
              ? '#F9FAFB'
              : '#FFFFFF',
        }}
      />

      <div style={{
        marginTop: '7px',
        display: 'flex',
        justifyContent:
          'space-between',
        gap: '10px',
        fontSize: '11px',
        color: '#9CA3AF',
        flexShrink: 0,
      }}>
        <span>
          支持Markdown · 拖拽/粘贴图片 · Ctrl+S保存
        </span>

        <span>
          约 {charCount} 字
        </span>
      </div>
    </div>
  )
}
