/**
 * ContextReceiptCard.tsx — AI消息下方的“本轮已读取”折叠卡片
 *
 * 默认只显示一行摘要，不打断老师阅读AI正文。
 *
 * 展开后仅展示：
 *   1. 本轮真正读取成功的教学资源；
 *   2. 已关联或已选择但读取失败的警告。
 *
 * 普通的未关联、不适用、稍后读取、已让位和明确不使用不展示。
 */

import { useMemo, useState } from 'react'
import type { ContextReceipt } from '@/api/lesson-plans'
import {
  buildContextReceiptView,
  type ContextReceiptTone,
  type ContextReceiptViewItem,
} from './contextReceiptViewModel'

interface ContextReceiptCardProps {
  receipt?: ContextReceipt
}

const TONE_STYLE: Record<
  ContextReceiptTone,
  {
    color: string
    background: string
    border: string
  }
> = {
  positive: {
    color: '#166534',
    background: '#F0FDF4',
    border: '#BBF7D0',
  },
  neutral: {
    color: '#64748B',
    background: '#F8FAFC',
    border: '#E2E8F0',
  },
  warning: {
    color: '#92400E',
    background: '#FFFBEB',
    border: '#FDE68A',
  },
}

function ReceiptItemRow({
  item,
}: {
  item: ContextReceiptViewItem
}) {
  const style = TONE_STYLE[item.tone]

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns:
          '92px minmax(0, 1fr)',
        gap: '10px',
        padding: '8px 0',
        borderBottom: '1px solid #EEF2F7',
      }}
    >
      <div>
        <div
          style={{
            fontSize: '12px',
            fontWeight: 600,
            color: '#334155',
            marginBottom: '4px',
          }}
        >
          {item.title}
        </div>

        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            padding: '2px 7px',
            borderRadius: '999px',
            border: `1px solid ${style.border}`,
            background: style.background,
            color: style.color,
            fontSize: '10px',
            lineHeight: 1.5,
            whiteSpace: 'nowrap',
          }}
        >
          {item.statusLabel}
        </span>
      </div>

      <div
        style={{
          minWidth: 0,
          fontSize: '12px',
          lineHeight: 1.65,
          color: item.tone === 'warning'
            ? '#92400E'
            : '#475569',
          wordBreak: 'break-word',
        }}
      >
        {item.detail}
      </div>
    </div>
  )
}

export default function ContextReceiptCard({
  receipt,
}: ContextReceiptCardProps) {
  const [expanded, setExpanded] =
    useState(false)

  const view = useMemo(
    () =>
      receipt
        ? buildContextReceiptView(receipt)
        : null,
    [receipt],
  )

  if (
    !receipt ||
    !view ||
    (
      view.usedItems.length === 0 &&
      view.warningItems.length === 0
    )
  ) {
    return null
  }

  const hasWarning =
    view.warningItems.length > 0

  return (
    <div
      style={{
        marginTop: '8px',
        border: `1px solid ${
          hasWarning ? '#FDE68A' : '#DCE5F2'
        }`,
        borderRadius: '10px',
        background: hasWarning
          ? 'linear-gradient(135deg, #FFFBEB, #FFFDF7)'
          : 'linear-gradient(135deg, #F8FAFF, #FBFDFF)',
        overflow: 'hidden',
      }}
    >
      <button
        type="button"
        onClick={() =>
          setExpanded(value => !value)
        }
        aria-expanded={expanded}
        style={{
          width: '100%',
          padding: '9px 12px',
          border: 'none',
          background: 'transparent',
          cursor: 'pointer',
          textAlign: 'left',
          display: 'flex',
          alignItems: 'flex-start',
          gap: '9px',
          color: '#334155',
        }}
      >
        <span
          aria-hidden="true"
          style={{
            flexShrink: 0,
            fontSize: '14px',
            lineHeight: 1.5,
          }}
        >
          {hasWarning ? '⚠️' : '🧭'}
        </span>

        <span
          style={{
            flex: 1,
            minWidth: 0,
          }}
        >
          <span
            style={{
              display: 'block',
              fontSize: '12px',
              fontWeight: 650,
              color: hasWarning
                ? '#92400E'
                : '#334155',
              marginBottom: '2px',
            }}
          >
            {hasWarning
              ? '本轮资源读取情况'
              : '本轮已读取'}
          </span>

          <span
            style={{
              display: 'block',
              fontSize: '11px',
              lineHeight: 1.55,
              color: hasWarning
                ? '#92400E'
                : '#64748B',
              whiteSpace: 'normal',
              wordBreak: 'break-word',
            }}
          >
            {view.summary.replace(
              /^本轮已读取：/,
              '',
            )}
          </span>
        </span>

        <span
          style={{
            flexShrink: 0,
            fontSize: '11px',
            color: '#64748B',
            paddingTop: '2px',
          }}
        >
          {expanded
            ? '收起 ▲'
            : '查看详情 ▼'}
        </span>
      </button>

      {expanded && (
        <div
          style={{
            borderTop:
              '1px solid #E5EBF4',
            padding: '4px 12px 10px',
          }}
        >
          {view.usedItems.length > 0 && (
            <>
              <div
                style={{
                  padding: '10px 0 2px',
                  fontSize: '11px',
                  fontWeight: 700,
                  color: '#166534',
                  letterSpacing: '0.2px',
                }}
              >
                本轮已读取
              </div>

              {view.usedItems.map(item => (
                <ReceiptItemRow
                  key={item.key}
                  item={item}
                />
              ))}
            </>
          )}

          {view.warningItems.length > 0 && (
            <>
              <div
                style={{
                  padding:
                    view.usedItems.length > 0
                      ? '13px 0 2px'
                      : '10px 0 2px',
                  fontSize: '11px',
                  fontWeight: 700,
                  color: '#92400E',
                  letterSpacing: '0.2px',
                }}
              >
                需要注意
              </div>

              {view.warningItems.map(item => (
                <ReceiptItemRow
                  key={item.key}
                  item={item}
                />
              ))}
            </>
          )}

          <div
            style={{
              marginTop: '9px',
              paddingTop: '8px',
              borderTop:
                '1px dashed #DCE5F2',
              fontSize: '10px',
              lineHeight: 1.55,
              color: '#94A3B8',
            }}
          >
            此处只展示本轮实际读取的教学资源和真实读取失败；
            完整系统回执仍保留在消息记录中。
          </div>
        </div>
      )}
    </div>
  )
}
