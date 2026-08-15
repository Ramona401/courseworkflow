/**
 * TextbookAttachOCRStatus.tsx — 对话模式课本卡片OCR状态展示
 *
 * 纯展示组件，不持有OCR状态、不发请求。
 */

import type { MouseEvent } from 'react'

interface Props {
  hasOCR: boolean
  running: boolean
  failed: boolean
  onRecognize: (event: MouseEvent<HTMLButtonElement>) => void
}

export default function TextbookAttachOCRStatus({
  hasOCR,
  running,
  failed,
  onRecognize,
}: Props) {
  if (running) {
    return (
      <span style={{ fontSize: '9px', color: '#4F7BE8' }}>
        识别中…
      </span>
    )
  }

  if (hasOCR) {
    return (
      <span
        style={{
          fontSize: '9px',
          padding: '1px 5px',
          borderRadius: '4px',
          background: 'rgba(16,185,129,0.08)',
          color: '#10B981',
        }}
      >
        已识别
      </span>
    )
  }

  return (
    <button
      onClick={onRecognize}
      style={{
        border: 'none',
        background: 'none',
        cursor: 'pointer',
        fontSize: '9px',
        color: failed ? '#EF4444' : '#4F7BE8',
        padding: 0,
        textDecoration: 'underline',
      }}
      title={failed ? '重新尝试AI识别文字' : 'AI识别文字'}
    >
      {failed ? '重试识别' : '识别文字'}
    </button>
  )
}
