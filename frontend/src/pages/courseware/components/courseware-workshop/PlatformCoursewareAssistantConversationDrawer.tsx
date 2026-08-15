/**
 * PlatformCoursewareAssistantConversationDrawer.tsx
 *
 * 课堂数字人的按需完整对话抽屉。
 *
 * 只负责历史消息展示与文字备用输入，不包含语音识别、数字人或TTS逻辑。
 * 全屏/放映默认不展示本抽屉，老师主动点击“完整对话”后才挂载。
 */

import { useEffect, useRef } from 'react'

import DiscussionMarkdown from './DiscussionMarkdown'

import {
  conversationStyle,
  messageBubbleStyle,
  primaryButtonStyle,
  sessionSummaryStyle,
  textareaStyle,
} from './PlatformCoursewareAssistantOverlay.styles'

export interface ClassroomConversationMessage {
  role: 'student' | 'assistant'
  content: string
  created_at?: string | null
}

interface PlatformCoursewareAssistantConversationDrawerProps {
  classroomMode: boolean
  messages: ClassroomConversationMessage[]
  streamingText: string
  input: string
  sessionStatus: string
  remainingTurns: number
  turnCount: number
  sending: boolean
  canSend: boolean
  onInputChange: (text: string) => void
  onSubmitMessage: (text: string) => boolean
}

function ClassroomMessageBubble({
  role,
  content,
  classroomMode,
  streaming = false,
}: {
  role: 'student' | 'assistant'
  content: string
  classroomMode: boolean
  streaming?: boolean
}) {
  const assistant = role === 'assistant'

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: assistant ? 'flex-start' : 'flex-end',
        marginBottom: classroomMode ? 12 : 8,
      }}
    >
      <div style={messageBubbleStyle(assistant, classroomMode)}>
        {assistant ? (
          <DiscussionMarkdown
            content={content}
            compact={!classroomMode}
          />
        ) : (
          content
        )}

        {streaming && (
          <span
            style={{
              marginLeft: 3,
              color: '#4F7BE8',
            }}
          >
            ▍
          </span>
        )}
      </div>
    </div>
  )
}

export default function PlatformCoursewareAssistantConversationDrawer({
  classroomMode,
  messages,
  streamingText,
  input,
  sessionStatus,
  remainingTurns,
  turnCount,
  sending,
  canSend,
  onInputChange,
  onSubmitMessage,
}: PlatformCoursewareAssistantConversationDrawerProps) {
  const endRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({
      block: 'end',
    })
  }, [
    messages,
    streamingText,
  ])

  return (
    <>
      <div style={sessionSummaryStyle(classroomMode)}>
        <span
          style={{
            color: '#475569',
            fontSize: classroomMode ? 14 : 10,
            fontWeight: 700,
          }}
        >
          教师预览 · 已完成{turnCount}轮
        </span>

        <span
          style={{
            color: remainingTurns > 0 ? '#059669' : '#B45309',
            fontSize: classroomMode ? 14 : 10,
            fontWeight: 850,
          }}
        >
          剩余{remainingTurns}轮
        </span>
      </div>

      <div
        style={{
          ...conversationStyle(classroomMode),
          minHeight: classroomMode ? 260 : 210,
          maxHeight: classroomMode
            ? 'min(430px, calc(100vh - 330px))'
            : 'min(330px, calc(100vh - 280px))',
        }}
      >
        {messages.length === 0 && !streamingText && (
          <div
            style={{
              padding: classroomMode ? '34px 16px' : '24px 12px',
              color: '#94A3B8',
              fontSize: classroomMode ? 16 : 11,
              lineHeight: 1.7,
              textAlign: 'center',
            }}
          >
            还没有对话内容。返回课堂后可以直接点击“说话”开始互动。
          </div>
        )}

        {messages.map((message, index) => (
          <ClassroomMessageBubble
            key={`${message.role}-${message.created_at || 'none'}-${index}`}
            role={message.role}
            content={message.content}
            classroomMode={classroomMode}
          />
        ))}

        {streamingText && (
          <ClassroomMessageBubble
            role="assistant"
            content={streamingText}
            classroomMode={classroomMode}
            streaming
          />
        )}

        <div ref={endRef} />
      </div>

      <div
        style={{
          padding: classroomMode ? 14 : 10,
          borderTop: '1px solid #E2E8F0',
          background: '#FFFFFF',
        }}
      >
        <div
          style={{
            marginBottom: 7,
            color: '#64748B',
            fontSize: classroomMode ? 13 : 10,
            fontWeight: 750,
          }}
        >
          文字备用输入
        </div>

        <textarea
          value={input}
          onChange={event => {
            onInputChange(event.target.value)
          }}
          onKeyDown={event => {
            event.stopPropagation()

            if (
              (event.ctrlKey || event.metaKey)
              && event.key === 'Enter'
            ) {
              event.preventDefault()
              onSubmitMessage(input)
            }
          }}
          disabled={!canSend}
          rows={classroomMode ? 2 : 3}
          maxLength={8000}
          placeholder={
            sessionStatus !== 'active'
              ? '本次预览已结束，请重新开始'
              : remainingTurns <= 0
                ? '本次互动轮数已用尽'
                : '语音不可用时，可在这里输入问题或学生的想法'
          }
          style={{
            ...textareaStyle(classroomMode),
            background: canSend ? '#FFFFFF' : '#F1F5F9',
          }}
        />

        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            marginTop: 9,
          }}
        >
          <button
            type="button"
            onClick={() => {
              onSubmitMessage(input)
            }}
            disabled={!canSend || !input.trim()}
            style={primaryButtonStyle(
              !canSend || !input.trim(),
              classroomMode,
            )}
          >
            {sending ? '正在回应…' : '发送文字'}
          </button>
        </div>
      </div>
    </>
  )
}
