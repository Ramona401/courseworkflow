/**
 * 教学智能体公开iframe学生端纯展示组件。
 *
 * 本文件不创建会话、不持有运行令牌、不调用API。
 * 状态和网络生命周期全部由assistant-embed.tsx负责。
 *
 * 学生体验原则：
 *   1. 首屏只解释要做什么、怎样获得帮助；
 *   2. 不展示发布版本、教师身份、模型、入口或积分；
 *   3. 学生主动点击“开始学习”后才建立短时会话；
 *   4. 对话中突出学生表达、尝试和请求提示；
 *   5. AI消息使用安全Markdown，学生输入始终作为React文本节点。
 */

import type {
  RefObject,
} from 'react'

import type {
  AssistantRuntimeMessage,
  AssistantRuntimeSessionView,
} from '../api/coursewares.assistant.types'

import DiscussionMarkdown from '../pages/courseware/components/courseware-workshop/DiscussionMarkdown'

import {
  ASSISTANT_EMBED_COLORS as C,
  assistantEmbedPrimaryButtonStyle,
  assistantEmbedSecondaryButtonStyle,
  type AssistantEmbedBootstrap,
  type AssistantEmbedNotice,
} from './assistantEmbedSupport'

interface AssistantEmbedStartScreenProps {
  bootstrap: AssistantEmbedBootstrap
  onStart: () => void
}

/**
 * 学生主动开始前看到的学习邀请。
 *
 * 这里完全不出现部署、外部入口、运行令牌、模型或积分等系统概念。
 */
export function AssistantEmbedStartScreen({
  bootstrap,
  onStart,
}: AssistantEmbedStartScreenProps) {
  return (
    <main
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        boxSizing: 'border-box',
        padding: 18,
        background:
          'linear-gradient(160deg, #F8FAFC 0%, #EEF4FF 100%)',
      }}
    >
      <section
        aria-label="开始本页学习互动"
        style={{
          width: '100%',
          maxWidth: 620,
          overflow: 'hidden',
          borderRadius: 20,
          border: `1px solid ${C.border}`,
          background: C.white,
          boxShadow:
            '0 18px 48px rgba(15,23,42,0.10)',
        }}
      >
        <div
          style={{
            padding: '24px 24px 20px',
            background:
              'linear-gradient(135deg, rgba(79,123,232,0.12), rgba(99,102,241,0.04))',
          }}
        >
          <div
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
              padding: '4px 9px',
              borderRadius: 999,
              background: C.primarySoft,
              color: C.primary,
              fontSize: 10,
              fontWeight: 800,
            }}
          >
            <span aria-hidden="true">
              ✨
            </span>
            本页学习互动
          </div>

          <h1
            style={{
              margin: '13px 0 0',
              color: C.text,
              fontSize: 21,
              lineHeight: 1.4,
            }}
          >
            {bootstrap.title}
          </h1>

          <p
            style={{
              margin: '10px 0 0',
              color: C.textSecondary,
              fontSize: 13,
              lineHeight: 1.8,
            }}
          >
            教学智能体会结合本页内容，通过提问和提示陪你一步步理解。
            先说出自己的想法，遇到困难时可以请求提示。
          </p>
        </div>

        <div
          style={{
            padding: '18px 24px 22px',
          }}
        >
          <div
            style={{
              display: 'grid',
              gridTemplateColumns:
                'repeat(auto-fit, minmax(180px, 1fr))',
              gap: 9,
            }}
          >
            <LearningPromise
              icon="💭"
              title="先表达自己的想法"
              description="答案不完整也没关系，从你的理解开始。"
            />

            <LearningPromise
              icon="🪜"
              title="需要时请求提示"
              description="系统会逐步给线索，不会直接替你完成答案。"
            />
          </div>

          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 14,
              marginTop: 18,
              flexWrap: 'wrap',
            }}
          >
            <div
              style={{
                color: C.textSecondary,
                fontSize: 11,
                lineHeight: 1.6,
              }}
            >
              本次最多互动{' '}
              <strong
                style={{
                  color: C.text,
                  fontSize: 14,
                }}
              >
                {bootstrap.maximumSessionTurns}
              </strong>{' '}
              轮
            </div>

            <button
              type="button"
              onClick={onStart}
              style={{
                ...assistantEmbedPrimaryButtonStyle(false),
                minWidth: 128,
                padding: '10px 20px',
                borderRadius: 10,
                fontSize: 13,
              }}
            >
              开始学习
            </button>
          </div>
        </div>
      </section>
    </main>
  )
}

function LearningPromise({
  icon,
  title,
  description,
}: {
  icon: string
  title: string
  description: string
}) {
  return (
    <div
      style={{
        padding: 12,
        borderRadius: 11,
        border: `1px solid ${C.border}`,
        background: C.background,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 7,
          color: C.text,
          fontSize: 11,
          fontWeight: 800,
        }}
      >
        <span
          aria-hidden="true"
          style={{
            fontSize: 16,
          }}
        >
          {icon}
        </span>
        {title}
      </div>

      <div
        style={{
          marginTop: 5,
          color: C.textSecondary,
          fontSize: 10,
          lineHeight: 1.65,
        }}
      >
        {description}
      </div>
    </div>
  )
}

interface AssistantEmbedConversationProps {
  bootstrap: AssistantEmbedBootstrap
  session: AssistantRuntimeSessionView
  messages: AssistantRuntimeMessage[]
  streamingText: string
  input: string
  sending: boolean
  canSend: boolean
  statusText: string
  notice: AssistantEmbedNotice | null
  messageEndRef:
    RefObject<HTMLDivElement | null>
  onInputChange: (value: string) => void
  onSend: () => void
  onRestart: () => void
}

export function AssistantEmbedConversation({
  bootstrap,
  session,
  messages,
  streamingText,
  input,
  sending,
  canSend,
  statusText,
  notice,
  messageEndRef,
  onInputChange,
  onSend,
  onRestart,
}: AssistantEmbedConversationProps) {
  return (
    <main
      style={{
        minHeight: '100vh',
        padding: 14,
        background: C.background,
      }}
    >
      <section
        aria-label={bootstrap.title}
        style={{
          width: '100%',
          maxWidth: 720,
          margin: '0 auto',
          overflow: 'hidden',
          borderRadius: 16,
          border: `1px solid ${C.border}`,
          background: C.white,
          boxShadow:
            '0 12px 32px rgba(15,23,42,0.08)',
        }}
      >
        <header
          style={{
            padding: '14px 16px',
            borderBottom:
              `1px solid ${C.border}`,
            background:
              'linear-gradient(135deg, rgba(79,123,232,0.10), rgba(99,102,241,0.05))',
          }}
        >
          <div
            style={{
              display: 'flex',
              justifyContent:
                'space-between',
              alignItems:
                'flex-start',
              gap: 12,
            }}
          >
            <div>
              <h1
                style={{
                  margin: 0,
                  color: C.text,
                  fontSize: 16,
                  lineHeight: 1.4,
                }}
              >
                {bootstrap.title}
              </h1>

              <div
                style={{
                  marginTop: 4,
                  color:
                    C.textSecondary,
                  fontSize: 10,
                  lineHeight: 1.5,
                }}
              >
                先说出自己的想法，遇到困难时可以请求提示。
              </div>
            </div>

            <span
              style={{
                flexShrink: 0,
                padding: '3px 8px',
                borderRadius: 999,
                background:
                  C.primarySoft,
                color: C.primary,
                fontSize: 9,
                fontWeight: 700,
              }}
            >
              学习互动
            </span>
          </div>

          {statusText && (
            <div
              style={{
                marginTop: 8,
                color: C.textMuted,
                fontSize: 9,
              }}
            >
              {statusText}
            </div>
          )}
        </header>

        {notice && (
          <AssistantEmbedNoticeBox
            notice={notice}
          />
        )}

        <div
          role="log"
          aria-live="polite"
          style={{
            minHeight: 300,
            maxHeight: 520,
            overflowY: 'auto',
            padding: 14,
            background: C.background,
          }}
        >
          {messages.map(
            (message, index) => (
              <AssistantEmbedMessageBubble
                key={`${message.role}-${message.created_at || 'none'}-${index}`}
                message={message}
              />
            ),
          )}

          {streamingText && (
            <AssistantEmbedMessageBubble
              message={{
                role: 'assistant',
                content:
                  streamingText,
                created_at: null,
              }}
              streaming
            />
          )}

          {sending &&
            !streamingText && (
              <div
                style={{
                  marginBottom: 10,
                  color: C.textMuted,
                  fontSize: 10,
                }}
              >
                正在根据你的想法准备下一步引导…
              </div>
            )}

          <div ref={messageEndRef} />
        </div>

        <footer
          style={{
            padding: 12,
            borderTop:
              `1px solid ${C.border}`,
            background: C.white,
          }}
        >
          <textarea
            value={input}
            rows={3}
            maxLength={8000}
            disabled={!canSend}
            placeholder={
              session.status !==
                'active'
                ? '本次学习互动已结束'
                : session.remaining_turns <=
                    0
                  ? '本次互动轮数已用尽'
                  : '写下你的理解、尝试或困惑…'
            }
            onChange={event =>
              onInputChange(
                event.target.value,
              )
            }
            onKeyDown={event => {
              if (
                event.key ===
                  'Enter' &&
                !event.shiftKey
              ) {
                event.preventDefault()
                onSend()
              }
            }}
            style={{
              width: '100%',
              minHeight: 76,
              boxSizing: 'border-box',
              padding: '10px 11px',
              resize: 'vertical',
              borderRadius: 10,
              border:
                `1px solid ${C.border}`,
              background: canSend
                ? C.white
                : C.background,
              color: C.text,
              fontFamily: 'inherit',
              fontSize: 13,
              lineHeight: 1.6,
              outline: 'none',
            }}
          />

          <div
            style={{
              display: 'flex',
              justifyContent:
                'space-between',
              alignItems: 'center',
              gap: 10,
              marginTop: 8,
              flexWrap: 'wrap',
            }}
          >
            <span
              style={{
                color: C.textMuted,
                fontSize: 9,
                lineHeight: 1.5,
              }}
            >
              Enter发送，Shift+Enter换行
            </span>

            <div
              style={{
                display: 'flex',
                gap: 7,
              }}
            >
              {session.status !==
                'active' && (
                <button
                  type="button"
                  onClick={onRestart}
                  style={
                    assistantEmbedSecondaryButtonStyle(
                      false,
                    )
                  }
                >
                  重新开始
                </button>
              )}

              <button
                type="button"
                onClick={onSend}
                disabled={
                  !canSend ||
                  !input.trim()
                }
                style={
                  assistantEmbedPrimaryButtonStyle(
                    !canSend ||
                      !input.trim(),
                  )
                }
              >
                {sending
                  ? '正在回应…'
                  : '发送想法'}
              </button>
            </div>
          </div>
        </footer>
      </section>
    </main>
  )
}

function AssistantEmbedMessageBubble({
  message,
  streaming = false,
}: {
  message: AssistantRuntimeMessage
  streaming?: boolean
}) {
  const assistant =
    message.role === 'assistant'

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: assistant
          ? 'flex-start'
          : 'flex-end',
        marginBottom: 10,
      }}
    >
      <div
        style={{
          maxWidth: '86%',
          padding: '9px 11px',
          borderRadius: assistant
            ? '4px 12px 12px 12px'
            : '12px 4px 12px 12px',
          border: assistant
            ? `1px solid ${C.border}`
            : '1px solid rgba(79,123,232,0.28)',
          background: assistant
            ? C.white
            : C.primarySoft,
          color: C.text,
          fontSize: 12,
          lineHeight: 1.7,
          wordBreak: 'break-word',
        }}
      >
        {assistant
          ? (
              <DiscussionMarkdown
                content={
                  message.content
                }
                compact
              />
            )
          : message.content}

        {streaming && (
          <span
            aria-hidden="true"
            style={{
              marginLeft: 4,
              color: C.primary,
            }}
          >
            ▍
          </span>
        )}
      </div>
    </div>
  )
}

function AssistantEmbedNoticeBox({
  notice,
}: {
  notice: AssistantEmbedNotice
}) {
  const style = {
    info: {
      background: '#EFF6FF',
      color: '#1D4ED8',
      border: '#BFDBFE',
    },
    success: {
      background: '#ECFDF5',
      color: C.success,
      border: '#A7F3D0',
    },
    error: {
      background: '#FEF2F2',
      color: C.danger,
      border: '#FECACA',
    },
  }[notice.kind]

  return (
    <div
      style={{
        margin: '10px 12px 0',
        padding: '8px 10px',
        borderRadius: 8,
        border:
          `1px solid ${style.border}`,
        background:
          style.background,
        color: style.color,
        fontSize: 10,
        lineHeight: 1.6,
      }}
    >
      {notice.text}
    </div>
  )
}

export function AssistantEmbedStandaloneMessage({
  icon,
  title,
  message,
  loading = false,
  actionLabel,
  onAction,
}: {
  icon: string
  title: string
  message: string
  loading?: boolean
  actionLabel?: string
  onAction?: () => void
}) {
  return (
    <main
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent:
          'center',
        boxSizing: 'border-box',
        padding: 20,
        background: C.background,
      }}
    >
      <section
        style={{
          width: '100%',
          maxWidth: 520,
          padding: 24,
          textAlign: 'center',
          borderRadius: 16,
          border:
            `1px solid ${C.border}`,
          background: C.white,
          boxShadow:
            '0 12px 32px rgba(15,23,42,0.08)',
        }}
      >
        <div
          style={{
            fontSize: 30,
            marginBottom: 10,
          }}
        >
          {icon}
        </div>

        <h1
          style={{
            margin: 0,
            color: C.text,
            fontSize: 18,
          }}
        >
          {title}
        </h1>

        <p
          style={{
            margin: '9px 0 0',
            color: C.textSecondary,
            fontSize: 12,
            lineHeight: 1.7,
          }}
        >
          {message}
        </p>

        {loading && (
          <div
            aria-label="加载中"
            style={{
              width: 24,
              height: 24,
              margin:
                '16px auto 0',
              borderRadius: '50%',
              border:
                `3px solid ${C.border}`,
              borderTopColor:
                C.primary,
              animation:
                'assistantEmbedSpin 0.8s linear infinite',
            }}
          />
        )}

        {actionLabel &&
          onAction && (
            <button
              type="button"
              onClick={onAction}
              style={{
                ...assistantEmbedPrimaryButtonStyle(
                  false,
                ),
                marginTop: 16,
              }}
            >
              {actionLabel}
            </button>
          )}

        <style>
          {'@keyframes assistantEmbedSpin{to{transform:rotate(360deg)}}'}
        </style>
      </section>
    </main>
  )
}
