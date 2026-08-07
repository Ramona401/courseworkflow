/**
 * ConversationTopBar.tsx — 对话模式顶栏与断线提示
 *
 * 顶栏只展示教师真正需要感知的信息：
 *   - 教案标题；
 *   - 当前连接状态；
 *   - 当前助手；
 *   - 课文依据对备课的教学意义。
 *
 * 已关联课本页的数量属于资源管理细节，不在备课对话顶栏展示。
 * textbookCount继续保留为兼容入参，但只用于判断是否已经具备课文依据。
 */

import { C } from '../components/workshopConstants'
import type { SSEConnectionState } from '@/api/lesson-plans'
import ResourceMeaningPill from './ResourceMeaningPill'

export interface ConversationTopBarProps {
  title: string

  /**
   * 已关联课本页数量。
   *
   * 本组件只把它转换成“是否具备课文依据”的布尔信息，
   * 不向教师展示具体数量。
   */
  textbookCount: number

  sseState: SSEConnectionState
  isNarrow: boolean
  narrowTab: 'chat' | 'canvas'
  onNarrowTabChange: (tab: 'chat' | 'canvas') => void
  onSwitchMode?: () => void
  onExit: () => void
  onReconnect: () => void
  assistantLabel?: string
  onAssistantClick?: () => void
}

export default function ConversationTopBar({
  title,
  textbookCount,
  sseState,
  isNarrow,
  narrowTab,
  onNarrowTabChange,
  onSwitchMode,
  onExit,
  onReconnect,
  assistantLabel,
  onAssistantClick,
}: ConversationTopBarProps) {
  const showAssistant =
    Boolean(assistantLabel) &&
    Boolean(onAssistantClick)

  const hasTextbookContext =
    textbookCount > 0

  return (
    <>
      <div
        style={{
          height: '46px',
          flexShrink: 0,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 18px',
          borderBottom: `1px solid ${C.border}`,
          background: C.card,
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            minWidth: 0,
          }}
        >
          <span style={{ fontSize: '15px' }}>💬</span>

          <span
            style={{
              fontSize: '14px',
              fontWeight: 700,
              color: C.text,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {title}
          </span>

          {hasTextbookContext && (
            <ResourceMeaningPill
              icon="📖"
              label="贴着课文备课"
              title="本课已具备课文依据，AI会在需要时结合原文"
              tone="course"
            />
          )}

          <div
            title={
              sseState === 'connected'
                ? '连接正常'
                : sseState === 'reconnecting'
                  ? '重连中…'
                  : '连接断开'
            }
            style={{
              width: '7px',
              height: '7px',
              borderRadius: '50%',
              flexShrink: 0,
              background:
                sseState === 'connected'
                  ? '#10B981'
                  : sseState === 'reconnecting'
                    ? '#F59E0B'
                    : '#EF4444',
            }}
          />

          {showAssistant && (
            <button
              type="button"
              onClick={onAssistantClick}
              title="点击切换本学科的备课助手，下一轮起生效"
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
                maxWidth: '200px',
                padding: '2px 8px',
                borderRadius: '8px',
                border: `1px solid ${C.border}`,
                background: 'rgba(99,102,241,0.06)',
                color: C.primary,
                flexShrink: 0,
                whiteSpace: 'nowrap',
                cursor: 'pointer',
                fontSize: '11px',
              }}
            >
              <span style={{ flexShrink: 0 }}>🎓</span>

              <span
                style={{
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                }}
              >
                {assistantLabel}
              </span>

              <span
                style={{
                  flexShrink: 0,
                  opacity: 0.7,
                }}
              >
                ▾
              </span>
            </button>
          )}
        </div>

        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            flexShrink: 0,
          }}
        >
          {isNarrow && (
            <div
              style={{
                display: 'flex',
                gap: '2px',
                padding: '2px',
                borderRadius: '8px',
                background: '#F3F4F6',
              }}
            >
              {([
                ['chat', '💬 对话'],
                ['canvas', '📄 教案'],
              ] as const).map(([key, label]) => (
                <button
                  type="button"
                  key={key}
                  onClick={() =>
                    onNarrowTabChange(key)
                  }
                  style={{
                    padding: '4px 12px',
                    borderRadius: '6px',
                    border: 'none',
                    background:
                      narrowTab === key
                        ? C.card
                        : 'transparent',
                    color:
                      narrowTab === key
                        ? C.primary
                        : C.textSec,
                    cursor: 'pointer',
                    fontSize: '12px',
                    fontWeight:
                      narrowTab === key
                        ? 600
                        : 400,
                  }}
                >
                  {label}
                </button>
              ))}
            </div>
          )}

          {onSwitchMode && (
            <button
              type="button"
              onClick={onSwitchMode}
              title="切换到专家模式，同一教案可随时互切"
              style={{
                padding: '5px 12px',
                borderRadius: '8px',
                border: `1px solid ${C.border}`,
                background: 'transparent',
                color: C.textSec,
                cursor: 'pointer',
                fontSize: '12px',
              }}
            >
              ⚙ 专家模式
            </button>
          )}

          <button
            type="button"
            onClick={onExit}
            title="退出备课，教案自动保存为草稿"
            style={{
              padding: '5px 12px',
              borderRadius: '8px',
              border: `1px solid ${C.border}`,
              background: 'transparent',
              color: C.textSec,
              cursor: 'pointer',
              fontSize: '12px',
            }}
          >
            🚪 退出
          </button>
        </div>
      </div>

      {sseState !== 'connected' && (
        <div
          style={{
            padding: '6px 16px',
            borderBottom: `1px solid ${C.border}`,
            background:
              sseState === 'reconnecting'
                ? 'rgba(245,158,11,0.08)'
                : 'rgba(239,68,68,0.08)',
            color:
              sseState === 'reconnecting'
                ? '#92400E'
                : '#991B1B',
            textAlign: 'center',
            flexShrink: 0,
            fontSize: '12px',
            fontWeight: 500,
          }}
        >
          {sseState === 'reconnecting'
            ? '网络连接中断，正在尝试重新连接…'
            : (
                <>
                  网络连接已断开
                  <button
                    type="button"
                    onClick={onReconnect}
                    style={{
                      marginLeft: '8px',
                      padding: '2px 10px',
                      borderRadius: '10px',
                      border:
                        '1px solid rgba(239,68,68,0.4)',
                      background: 'transparent',
                      color: '#DC2626',
                      cursor: 'pointer',
                      fontSize: '12px',
                    }}
                  >
                    点击重连
                  </button>
                </>
              )}
        </div>
      )}
    </>
  )
}
