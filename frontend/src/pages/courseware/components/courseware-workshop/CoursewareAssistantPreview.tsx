/**
 * 教学智能体教师预览。
 *
 * 外层明确告诉教师这是预览和真实用量环境；
 * 内层启动卡与对话区域尽量还原学生实际体验。
 *
 * 学生模拟区域不展示：
 *   - 发布版本和内部部署状态；
 *   - external入口、运行令牌等技术概念；
 *   - 模型与教师积分结算说明。
 *
 * 教师专属信息统一放在学生体验区域之外。
 */

import {
  useState,
  type CSSProperties,
} from 'react'

import DiscussionMarkdown from './DiscussionMarkdown'

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
  CoursewareAssistantSection,
} from './CoursewareAssistantEditorShared'

import {
  useCoursewareAssistantPreview,
} from './useCoursewareAssistantPreview'

interface CoursewareAssistantPreviewProps {
  coursewareId: string
  pageId: string
  pageTitle: string
  hasSavedSlot: boolean
  hasUnsavedChanges: boolean
  refreshKey?: number
  disabled?: boolean
}

export default function CoursewareAssistantPreview({
  coursewareId,
  pageId,
  pageTitle,
  hasSavedSlot,
  hasUnsavedChanges,
  refreshKey = 0,
  disabled = false,
}: CoursewareAssistantPreviewProps) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS

  const preview =
    useCoursewareAssistantPreview({
      coursewareId,
      pageId,
      refreshKey,
    })

  const [
    input,
    setInput,
  ] = useState('')

  const submit = () => {
    if (
      preview.sendMessage(
        input,
      )
    ) {
      setInput('')
    }
  }

  const noActiveDeploymentMessage =
    (() => {
      if (
        preview
          .latestPageDeployment
          ?.status === 'paused'
      ) {
        return '当前页面的教学智能体已暂停运行。恢复后才能开始教师预览。'
      }

      if (
        preview
          .latestPageDeployment
          ?.status === 'revoked'
      ) {
        return '当前页面最近的发布已经永久撤销，需要重新发布后才能预览。'
      }

      if (!hasSavedSlot) {
        return '当前页面尚未保存教学智能体方案。请先创建、保存并发布。'
      }

      return '当前页面尚无正在运行的发布版本。请先完成发布。'
    })()

  return (
    <CoursewareAssistantSection
      title="教师预览"
      description="以学生看到的方式体验当前已发布版本。打开和浏览不产生模型用量；发送消息并成功生成回答后，才按实际用量结算。"
      actions={
        <button
          type="button"
          onClick={() => {
            void preview
              .loadDeployments()
          }}
          disabled={
            disabled ||
            preview
              .deploymentLoading ||
            preview.starting ||
            preview.sending
          }
          style={
            smallButtonStyle(
              disabled ||
                preview
                  .deploymentLoading ||
                preview.starting ||
                preview.sending,
            )
          }
        >
          {preview.deploymentLoading
            ? '同步中…'
            : '刷新发布状态'}
        </button>
      }
    >
      {hasUnsavedChanges && (
        <NoticeBox
          kind="warning"
          text="当前存在未保存修改。本次预览只使用已经发布的版本。"
        />
      )}

      {preview
        .deploymentError && (
        <NoticeBox
          kind="error"
          text={
            preview
              .deploymentError
          }
        />
      )}

      {preview.notice && (
        <NoticeBox
          kind={
            preview.notice.kind
          }
          text={
            preview.notice.text
          }
        />
      )}

      {preview
        .deploymentLoading &&
        !preview
          .activeDeployment && (
          <div
            style={{
              padding:
                '18px 12px',
              textAlign: 'center',
              color: C.textMuted,
              fontSize: 11,
            }}
          >
            正在读取当前页面的发布状态…
          </div>
        )}

      {!preview
        .deploymentLoading &&
        !preview
          .activeDeployment && (
          <div
            style={{
              padding: 13,
              borderRadius: 9,
              border:
                `1px dashed ${C.border}`,
              background:
                C.background,
              color:
                C.textSecondary,
              fontSize: 11,
              lineHeight: 1.7,
            }}
          >
            {noActiveDeploymentMessage}
          </div>
        )}

      {preview
        .activeDeployment &&
        !preview.session && (
          <PreviewStartCard
            pageTitle={pageTitle}
            currentVersion={
              preview
                .activeDeployment
                .current_version
            }
            maximumTurns={
              preview
                .activeDeployment
                .per_session_turn_limit
            }
            starting={
              preview.starting
            }
            disabled={disabled}
            onStart={() => {
              void preview
                .startPreview()
            }}
          />
        )}

      {preview.session && (
        <div
          style={{
            border:
              `1px solid ${C.border}`,
            borderRadius: 12,
            overflow: 'hidden',
            background: C.white,
          }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent:
                'space-between',
              gap: 10,
              padding: '10px 12px',
              borderBottom:
                `1px solid ${C.border}`,
              background:
                C.background,
            }}
          >
            <div>
              <div
                style={{
                  display:
                    'flex',
                  alignItems:
                    'center',
                  gap: 7,
                }}
              >
                <strong
                  style={{
                    color: C.text,
                    fontSize: 11,
                  }}
                >
                  {pageTitle ||
                    '本页学习互动'}
                </strong>

                <span
                  style={{
                    padding:
                      '2px 6px',
                    borderRadius: 999,
                    background:
                      '#EEF2FF',
                    color: '#4F46E5',
                    fontSize: 8,
                    fontWeight: 800,
                  }}
                >
                  教师预览
                </span>
              </div>

              <div
                style={{
                  marginTop: 3,
                  color: C.textMuted,
                  fontSize: 9,
                }}
              >
                已完成{' '}
                {
                  preview.session
                    .turn_count
                }{' '}
                轮，剩余{' '}
                {
                  preview
                    .remainingTurns
                }{' '}
                轮
              </div>
            </div>

            <button
              type="button"
              onClick={
                preview.clearSession
              }
              disabled={
                preview.sending
              }
              style={
                smallButtonStyle(
                  preview.sending,
                )
              }
            >
              结束预览
            </button>
          </div>

          <div
            style={{
              minHeight: 220,
              maxHeight: 440,
              overflowY: 'auto',
              padding: 14,
              background:
                '#F8FAFC',
            }}
          >
            {preview.messages
              .length === 0 &&
              !preview
                .streamingText && (
                <div
                  style={{
                    padding:
                      '28px 12px',
                    textAlign:
                      'center',
                    color:
                      C.textMuted,
                    fontSize: 11,
                    lineHeight: 1.7,
                  }}
                >
                  学习互动已开始，请用学生身份说出自己的想法。
                </div>
              )}

            {preview.messages.map(
              (
                message,
                index,
              ) => (
                <MessageBubble
                  key={`${message.role}-${message.created_at || 'none'}-${index}`}
                  role={
                    message.role
                  }
                  content={
                    message.content
                  }
                />
              ),
            )}

            {preview
              .streamingText && (
              <MessageBubble
                role="assistant"
                content={
                  preview
                    .streamingText
                }
                streaming
              />
            )}
          </div>

          <div
            style={{
              padding: 12,
              borderTop:
                `1px solid ${C.border}`,
              background: C.white,
            }}
          >
            <textarea
              value={input}
              onChange={event =>
                setInput(
                  event.target.value,
                )
              }
              onKeyDown={event => {
                if (
                  (
                    event.ctrlKey ||
                    event.metaKey
                  ) &&
                  event.key ===
                    'Enter'
                ) {
                  event.preventDefault()
                  submit()
                }
              }}
              disabled={
                disabled ||
                !preview.canSend
              }
              rows={3}
              maxLength={8000}
              placeholder={
                preview.session
                  .status !==
                'active'
                  ? '本次预览已经结束'
                  : preview
                        .remainingTurns <=
                      0
                    ? '本次互动轮数已用尽'
                    : '写下学生可能的回答、尝试或困惑…'
              }
              style={{
                width: '100%',
                minHeight: 72,
                boxSizing:
                  'border-box',
                resize: 'vertical',
                padding:
                  '9px 11px',
                borderRadius: 8,
                border:
                  `1px solid ${C.border}`,
                background:
                  preview.canSend &&
                  !disabled
                    ? C.white
                    : C.background,
                color: C.text,
                fontSize: 12,
                lineHeight: 1.6,
                fontFamily:
                  'inherit',
                outline: 'none',
              }}
            />

            <div
              style={{
                display: 'flex',
                justifyContent:
                  'space-between',
                alignItems:
                  'center',
                gap: 10,
                marginTop: 8,
                flexWrap: 'wrap',
              }}
            >
              <span
                style={{
                  color: C.textMuted,
                  fontSize: 9,
                }}
              >
                Ctrl/⌘ + Enter发送
              </span>

              <button
                type="button"
                onClick={submit}
                disabled={
                  disabled ||
                  !preview.canSend ||
                  !input.trim()
                }
                style={
                  primaryButtonStyle(
                    disabled ||
                      !preview
                        .canSend ||
                      !input.trim(),
                  )
                }
              >
                {preview.sending
                  ? '正在回应…'
                  : '发送想法'}
              </button>
            </div>
          </div>
        </div>
      )}
    </CoursewareAssistantSection>
  )
}

function PreviewStartCard({
  pageTitle,
  currentVersion,
  maximumTurns,
  starting,
  disabled,
  onStart,
}: {
  pageTitle: string
  currentVersion: number
  maximumTurns: number
  starting: boolean
  disabled: boolean
  onStart: () => void
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS

  return (
    <>
      <div
        style={{
          padding: 17,
          borderRadius: 13,
          border:
            '1px solid #BFDBFE',
          background:
            'linear-gradient(145deg, #F8FAFF, #EFF6FF)',
        }}
      >
        <div
          style={{
            display: 'inline-flex',
            padding: '3px 8px',
            borderRadius: 999,
            background:
              'rgba(79,123,232,0.10)',
            color: C.primary,
            fontSize: 9,
            fontWeight: 800,
          }}
        >
          学生看到的开始体验
        </div>

        <div
          style={{
            marginTop: 10,
            color: C.text,
            fontSize: 16,
            fontWeight: 800,
          }}
        >
          开始本页学习互动
        </div>

        {pageTitle && (
          <div
            style={{
              marginTop: 4,
              color:
                C.textSecondary,
              fontSize: 10,
            }}
          >
            {pageTitle}
          </div>
        )}

        <div
          style={{
            marginTop: 9,
            color:
              C.textSecondary,
            fontSize: 11,
            lineHeight: 1.75,
          }}
        >
          教学智能体会结合本页内容，通过提问和提示陪学生一步步理解。
          学生先说出自己的想法，遇到困难时可以请求提示。
        </div>

        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent:
              'space-between',
            gap: 12,
            marginTop: 14,
            flexWrap: 'wrap',
          }}
        >
          <div
            style={{
              color:
                C.textSecondary,
              fontSize: 10,
            }}
          >
            最多互动{' '}
            <strong
              style={{
                color: C.text,
                fontSize: 13,
              }}
            >
              {maximumTurns}
            </strong>{' '}
            轮
          </div>

          <button
            type="button"
            onClick={onStart}
            disabled={
              disabled ||
              starting
            }
            style={
              primaryButtonStyle(
                disabled ||
                  starting,
              )
            }
          >
            {starting
              ? '正在准备…'
              : '开始学习'}
          </button>
        </div>
      </div>

      <div
        style={{
          marginTop: 10,
          padding: '9px 11px',
          borderRadius: 9,
          border:
            `1px solid ${C.border}`,
          background: C.white,
          color:
            C.textSecondary,
          fontSize: 9,
          lineHeight: 1.65,
        }}
      >
        <strong
          style={{
            color: C.text,
          }}
        >
          教师预览 · V
          {currentVersion}
        </strong>
        ：这里使用真实发布版本。发送消息并成功生成回答后，
        才按实际模型用量结算教师教学积分。
      </div>
    </>
  )
}

function MessageBubble({
  role,
  content,
  streaming = false,
}: {
  role:
    | 'student'
    | 'assistant'
  content: string
  streaming?: boolean
}) {
  const assistant =
    role === 'assistant'

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
          maxWidth: '84%',
          padding: '9px 11px',
          borderRadius: assistant
            ? '4px 11px 11px 11px'
            : '11px 4px 11px 11px',
          border: assistant
            ? '1px solid #E2E8F0'
            : '1px solid rgba(79,123,232,0.28)',
          background: assistant
            ? '#FFFFFF'
            : 'rgba(79,123,232,0.10)',
          color: '#1F2937',
          fontSize: 12,
          lineHeight: 1.7,
          wordBreak: 'break-word',
        }}
      >
        {assistant
          ? (
              <DiscussionMarkdown
                content={content}
                compact
              />
            )
          : content}

        {streaming && (
          <span
            style={{
              marginLeft: 4,
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

function NoticeBox({
  kind,
  text,
}: {
  kind:
    | 'info'
    | 'success'
    | 'error'
    | 'warning'
  text: string
}) {
  const style = {
    info: {
      border: '#BFDBFE',
      background: '#EFF6FF',
      color: '#2563EB',
    },
    success: {
      border: '#A7F3D0',
      background: '#ECFDF5',
      color: '#047857',
    },
    error: {
      border: '#FECACA',
      background: '#FEF2F2',
      color: '#B91C1C',
    },
    warning: {
      border: '#FDE68A',
      background: '#FFFBEB',
      color: '#92400E',
    },
  }[kind]

  return (
    <div
      style={{
        marginBottom: 10,
        padding: '9px 11px',
        borderRadius: 8,
        border:
          `1px solid ${style.border}`,
        background:
          style.background,
        color: style.color,
        fontSize: 10,
        lineHeight: 1.65,
      }}
    >
      {text}
    </div>
  )
}

function primaryButtonStyle(
  disabled: boolean,
): CSSProperties {
  return {
    padding: '9px 16px',
    borderRadius: 9,
    border: 'none',
    background: '#4F7BE8',
    color: '#FFFFFF',
    fontSize: 11,
    fontWeight: 700,
    cursor: disabled
      ? 'default'
      : 'pointer',
    opacity: disabled
      ? 0.45
      : 1,
  }
}

function smallButtonStyle(
  disabled: boolean,
): CSSProperties {
  return {
    padding: '5px 10px',
    borderRadius: 7,
    border:
      '1px solid #E2E8F0',
    background: '#FFFFFF',
    color: '#64748B',
    fontSize: 10,
    fontWeight: 700,
    cursor: disabled
      ? 'default'
      : 'pointer',
    opacity: disabled
      ? 0.45
      : 1,
  }
}
