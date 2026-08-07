/**
 * ResourceMeaningPill.tsx — 备课资源的教学语义提示
 *
 * 产品原则：
 *   - 教师端不展示附件文件名、挂载数量、字符数或读取状态；
 *   - 只表达资源对当前备课产生的教学意义；
 *   - 上传、替换、移除等资源管理能力继续保留；
 *   - 提示必须安静，不抢占对话，不制造“系统正在加载”的技术感。
 */

import type { CSSProperties } from 'react'

export type ResourceMeaningTone =
  | 'course'
  | 'strategy'
  | 'evidence'

export interface ResourceMeaningPillProps {
  icon: string
  label: string
  title?: string
  tone?: ResourceMeaningTone
  variant?: 'badge' | 'strip'
  onClear?: () => void
  clearLabel?: string
}

const TONE_STYLE: Record<
  ResourceMeaningTone,
  {
    color: string
    background: string
    border: string
  }
> = {
  course: {
    color: '#047857',
    background: 'rgba(16,185,129,0.08)',
    border: 'rgba(16,185,129,0.18)',
  },
  strategy: {
    color: '#4F46E5',
    background: 'rgba(99,102,241,0.08)',
    border: 'rgba(99,102,241,0.16)',
  },
  evidence: {
    color: '#6D5BD0',
    background: 'rgba(129,140,248,0.08)',
    border: 'rgba(129,140,248,0.16)',
  },
}

export default function ResourceMeaningPill({
  icon,
  label,
  title,
  tone = 'course',
  variant = 'badge',
  onClear,
  clearLabel = '移除',
}: ResourceMeaningPillProps) {
  const toneStyle = TONE_STYLE[tone]

  const sharedStyle: CSSProperties = {
    color: toneStyle.color,
    background: toneStyle.background,
    border: `1px solid ${toneStyle.border}`,
  }

  if (variant === 'strip') {
    return (
      <div
        title={title}
        style={{
          ...sharedStyle,
          minHeight: '34px',
          padding: '7px 18px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: '12px',
          flexShrink: 0,
          borderRight: 'none',
          borderLeft: 'none',
          borderRadius: 0,
          fontSize: '12px',
          lineHeight: 1.5,
        }}
      >
        <span
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            minWidth: 0,
          }}
        >
          <span
            aria-hidden="true"
            style={{
              flexShrink: 0,
              fontSize: '14px',
            }}
          >
            {icon}
          </span>

          <span
            style={{
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {label}
          </span>
        </span>

        {onClear && (
          <button
            type="button"
            onClick={onClear}
            style={{
              flexShrink: 0,
              padding: '2px 7px',
              border: 'none',
              background: 'transparent',
              color: '#64748B',
              cursor: 'pointer',
              fontSize: '12px',
            }}
          >
            {clearLabel}
          </button>
        )}
      </div>
    )
  }

  return (
    <span
      title={title}
      style={{
        ...sharedStyle,
        display: 'inline-flex',
        alignItems: 'center',
        gap: '5px',
        maxWidth: '180px',
        padding: '2px 8px',
        borderRadius: '8px',
        flexShrink: 0,
        fontSize: '11px',
        lineHeight: 1.45,
        whiteSpace: 'nowrap',
      }}
    >
      <span aria-hidden="true">{icon}</span>

      <span
        style={{
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        {label}
      </span>
    </span>
  )
}
