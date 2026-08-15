/**
 * 平台课件教学智能体悬浮层的嵌入式展示部件。
 *
 * 从主悬浮层拆出两个纯展示组件，避免运行逻辑文件逼近900行：
 *   - StartPanel：自动建立课堂会话时的过渡状态；
 *   - MessageBubble：学生/智能体消息气泡。
 *
 * 本文件不创建会话、不判断部署状态、不保存数据，也不执行导航。
 */

import DiscussionMarkdown from './DiscussionMarkdown'

import {
  messageBubbleStyle,
} from './PlatformCoursewareAssistantOverlay.styles'

export function StartPanel({
  pageTitle,
  currentVersion,
  maximumTurns,
  starting,
  expired,
  classroomMode,
}: {
  pageTitle: string
  currentVersion: number
  maximumTurns: number
  starting: boolean
  expired: boolean
  classroomMode: boolean
}) {
  return (
    <div style={{ padding: classroomMode ? 20 : 15 }}>
      <div
        style={{
          padding: classroomMode ? 20 : 14,
          borderRadius: classroomMode ? 17 : 13,
          border: expired ? '1px solid #FECACA' : '1px solid #C7D2FE',
          background: expired
            ? 'linear-gradient(145deg, #FFF7ED, #FEF2F2)'
            : 'linear-gradient(145deg, #F8FAFF, #EEF2FF)',
        }}
      >
        <div
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 7,
            padding: classroomMode ? '5px 9px' : '3px 7px',
            borderRadius: 999,
            background: expired
              ? 'rgba(254,226,226,0.86)'
              : 'rgba(79,123,232,0.10)',
            color: expired ? '#B91C1C' : '#4F7BE8',
            fontSize: classroomMode ? 13 : 8.5,
            fontWeight: 800,
          }}
        >
          {!expired && (
            <span
              aria-hidden="true"
              style={{
                width: classroomMode ? 8 : 6,
                height: classroomMode ? 8 : 6,
                borderRadius: '50%',
                background: '#6366F1',
                animation: 'tednaAssistantEntryPulse 1.15s ease-in-out infinite',
              }}
            />
          )}
          {expired ? '课堂暂不可开始' : '智能体正在进入课堂'}
        </div>

        <div
          style={{
            marginTop: classroomMode ? 13 : 9,
            color: '#1E293B',
            fontSize: classroomMode ? 23 : 14,
            fontWeight: 850,
            lineHeight: 1.4,
          }}
        >
          {expired
            ? '使用时间已经结束'
            : '稍后由智能体先主动问候'}
        </div>

        {pageTitle && (
          <div
            style={{
              marginTop: classroomMode ? 6 : 3,
              overflow: 'hidden',
              color: '#64748B',
              fontSize: classroomMode ? 15 : 9,
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {pageTitle}
          </div>
        )}

        <div
          style={{
            marginTop: classroomMode ? 13 : 8,
            color: expired ? '#9A3412' : '#475569',
            fontSize: classroomMode ? 17 : 10.5,
            lineHeight: 1.75,
          }}
        >
          {expired
            ? '请先调整课堂使用时间。保存后，当前页面的教学智能体会自动同步恢复。'
            : '无需点击“开始”。系统正在建立课堂会话，完成后智能体会先打招呼，再等待你的语音输入。'}
        </div>

        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 12,
            marginTop: classroomMode ? 18 : 13,
          }}
        >
          <div
            style={{
              color: '#64748B',
              fontSize: classroomMode ? 15 : 10,
            }}
          >
            最多互动{' '}
            <strong
              style={{
                color: '#1E293B',
                fontSize: classroomMode ? 19 : 12,
              }}
            >
              {maximumTurns}
            </strong>{' '}
            轮
          </div>

          {!expired && (
            <span
              aria-live="polite"
              style={{
                color: '#4F46E5',
                fontSize: classroomMode ? 14 : 9.5,
                fontWeight: 800,
              }}
            >
              {starting ? '正在建立会话…' : '正在准备课堂…'}
            </span>
          )}
        </div>
      </div>

      <div
        style={{
          marginTop: classroomMode ? 13 : 10,
          padding: classroomMode ? '12px 14px' : '9px 10px',
          borderRadius: classroomMode ? 12 : 9,
          border: '1px solid #E2E8F0',
          background: '#FFFFFF',
          color: '#64748B',
          fontSize: classroomMode ? 13 : 9,
          lineHeight: 1.65,
        }}
      >
        <strong style={{ color: '#475569' }}>
          教师预览 · V{currentVersion}
        </strong>
        ：会话建立本身不发送模型问题；智能体开场语来自当前正式发布内容。
      </div>

      <style>{`
        @keyframes tednaAssistantEntryPulse {
          0%, 100% {
            transform: scale(0.82);
            opacity: 0.5;
          }
          50% {
            transform: scale(1.08);
            opacity: 1;
          }
        }
      `}</style>
    </div>
  )
}

export function MessageBubble({
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
        marginBottom: classroomMode ? 13 : 8,
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
