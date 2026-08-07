/**
 * 平台课件预览悬浮教学智能体。
 *
 * 该组件只在教师登录态中运行，使用正式发布版本和真实教师预览会话。
 *
 * 老师端课堂语音增强：
 *   - 复用教师JWT语音识别，最终识别文字自动发送；
 *   - AI完整回答的done事件到达后请求豆包自然语音，不朗读流式半成品；
 *   - 中文默认vivi 2.0，英文或字母为主默认Tim，失败时自动降级设备语音；
 *   - 开始录音前停止当前朗读和未完成的TTS请求，避免扬声器声音重新进入麦克风；
 *   - 全屏和放映默认启用课堂大字，嵌入预览可手动切换；
 *   - 文字输入继续作为语音不可用或识别失败时的备用入口；
 *   - 面板关闭、页面切换和会话结束时停止当前朗读。
 *
 * UE边界：
 *   - 学生体验卡只展示学习任务、提示方式和互动轮数；
 *   - 教师身份、发布版本和积分说明放在学生体验卡之外；
 *   - 不在模拟学生界面中出现external、模型或结算技术术语；
 *   - 打开面板不创建会话，教师点击“开始学习”后才启动预览；
 *   - 页面切换和组件卸载时由Hook清理令牌与流连接。
 */

import { useEffect, useRef, useState } from 'react'
import type { SyntheticEvent } from 'react'
import { useParams } from 'react-router-dom'

import useAssistantSpeechPlayback from '@/hooks/useAssistantSpeechPlayback'
import type { VoiceInputStatus } from '@/hooks/useVoiceInput'

import ClassroomDigitalHuman from './ClassroomDigitalHuman'
import type { ClassroomDigitalHumanState } from './ClassroomDigitalHuman'
import DiscussionMarkdown from './DiscussionMarkdown'
import PlatformCoursewareAssistantVoiceControls
  from './PlatformCoursewareAssistantVoiceControls'
import { useCoursewareAssistantPreview } from './useCoursewareAssistantPreview'

import {
  composerStyle,
  conversationStyle,
  floatingButtonStyle,
  headerButtonStyle,
  messageBubbleStyle,
  noticeStyle,
  overlayLayout,
  overlayRootBaseStyle,
  panelHeaderStyle,
  panelStyle,
  primaryButtonStyle,
  sessionSummaryStyle,
  textareaStyle,
} from './PlatformCoursewareAssistantOverlay.styles'

import type {
  PlatformAssistantOverlayVariant,
} from './PlatformCoursewareAssistantOverlay.styles'

interface PlatformCoursewareAssistantOverlayProps {
  coursewareId?: string
  pageId: string
  pageTitle: string
  variant?: PlatformAssistantOverlayVariant
}

export default function PlatformCoursewareAssistantOverlay({
  coursewareId,
  pageId,
  pageTitle,
  variant = 'embedded',
}: PlatformCoursewareAssistantOverlayProps) {
  const params = useParams<{ id: string }>()

  const resolvedCoursewareID = (coursewareId || params.id || '').trim()
  const resolvedPageID = pageId.trim()

  const preview = useCoursewareAssistantPreview({
    coursewareId: resolvedCoursewareID,
    pageId: resolvedPageID,
  })

  const speech = useAssistantSpeechPlayback(
    preview.activeDeployment?.id || '',
  )

  const [open, setOpen] = useState(false)
  const [input, setInput] = useState('')
  const [autoSpeak, setAutoSpeak] = useState(true)
  const [classroomMode, setClassroomMode] = useState(
    variant !== 'embedded',
  )
  const [voiceStatus, setVoiceStatus] = useState<VoiceInputStatus>('idle')

  const messagesEndRef = useRef<HTMLDivElement | null>(null)
  const handledReplySequenceRef = useRef(0)

  useEffect(() => {
    setOpen(false)
    setInput('')
    setVoiceStatus('idle')
    speech.stop()
  }, [
    resolvedCoursewareID,
    resolvedPageID,
    speech.stop,
  ])

  useEffect(() => {
    if (!open) {
      speech.stop()
      return
    }

    messagesEndRef.current?.scrollIntoView({
      block: 'end',
    })
  }, [
    open,
    preview.messages,
    preview.streamingText,
    speech.stop,
  ])

  useEffect(() => {
    const completedReply = preview.completedReply

    if (
      !completedReply
      || handledReplySequenceRef.current === completedReply.sequence
    ) {
      return
    }

    handledReplySequenceRef.current = completedReply.sequence

    if (open && autoSpeak) {
      speech.speak(completedReply.text)
    }
  }, [
    autoSpeak,
    open,
    preview.completedReply,
    speech.speak,
  ])

  if (
    !resolvedCoursewareID
    || !resolvedPageID
    || !preview.activeDeployment
  ) {
    return null
  }

  const submitMessage = (rawMessage: string): boolean => {
    speech.stop()

    if (!preview.sendMessage(rawMessage)) {
      return false
    }

    setInput('')
    return true
  }

  const closePanel = () => {
    speech.stop()
    setVoiceStatus('idle')
    setOpen(false)
  }

  const endSession = () => {
    speech.stop()
    setVoiceStatus('idle')
    preview.clearSession()
    setInput('')
  }

  const toggleAutoSpeak = () => {
    setAutoSpeak(current => {
      if (current) {
        speech.stop()
      }

      return !current
    })
  }

  const stopPropagation = (event: SyntheticEvent) => {
    event.stopPropagation()
  }

  const layout = overlayLayout(
    variant,
    classroomMode,
  )

  const digitalHumanState: ClassroomDigitalHumanState =
    voiceStatus === 'connecting'
    || voiceStatus === 'recording'
    || voiceStatus === 'stopping'
      ? 'listening'
      : speech.speaking && !speech.paused
        ? 'speaking'
        : preview.sending || speech.preparing
          ? 'thinking'
          : 'idle'

  return (
    <div
      onClick={stopPropagation}
      onDoubleClick={stopPropagation}
      onMouseDown={stopPropagation}
      onMouseUp={stopPropagation}
      onMouseMove={stopPropagation}
      onPointerDown={stopPropagation}
      onPointerUp={stopPropagation}
      style={{
        ...overlayRootBaseStyle,
        right: layout.right,
        bottom: layout.bottom,
        zIndex: layout.zIndex,
      }}
    >
      {!open && (
        <button
          type="button"
          onClick={() => {
            setOpen(true)
          }}
          title="打开当前页面的教学智能体"
          style={floatingButtonStyle(variant)}
        >
          <span
            aria-hidden="true"
            style={{
              fontSize: variant === 'embedded' ? 18 : 24,
              lineHeight: 1,
            }}
          >
            🤖
          </span>

          <span>教学智能体</span>

          <span
            aria-hidden="true"
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: '#10B981',
              boxShadow: '0 0 0 3px rgba(16,185,129,0.15)',
            }}
          />
        </button>
      )}

      {open && (
        <div
          style={{
            position: 'absolute',
            right: layout.width + (classroomMode ? 18 : 12),
            bottom: 0,
            pointerEvents: 'none',
          }}
        >
          <ClassroomDigitalHuman
            state={digitalHumanState}
            audioElement={speech.audioElement}
            speechProvider={speech.provider}
            speechPaused={speech.paused}
            classroomMode={classroomMode}
            variant={variant}
          />
        </div>
      )}

      {open && (
        <div
          role="dialog"
          aria-label="教学智能体教师预览"
          style={panelStyle(layout, classroomMode)}
        >
          <div style={panelHeaderStyle(classroomMode)}>
            <div style={{ minWidth: 0 }}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: classroomMode ? 9 : 6,
                  color: '#1E293B',
                  fontSize: classroomMode ? 19 : 12,
                  fontWeight: 850,
                }}
              >
                <span>🤖 教学智能体</span>

                <span
                  style={{
                    padding: classroomMode ? '4px 8px' : '2px 6px',
                    borderRadius: 999,
                    background: '#EEF2FF',
                    color: '#4F46E5',
                    fontSize: classroomMode ? 12 : 8,
                    fontWeight: 800,
                  }}
                >
                  教师预览
                </span>
              </div>

              <div
                title={pageTitle}
                style={{
                  maxWidth: classroomMode ? 420 : 280,
                  marginTop: classroomMode ? 5 : 3,
                  overflow: 'hidden',
                  color: '#64748B',
                  fontSize: classroomMode ? 14 : 9,
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {pageTitle || '当前课件页面'}
              </div>
            </div>

            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: classroomMode ? 8 : 6,
              }}
            >
              <button
                type="button"
                onClick={() => {
                  setClassroomMode(current => !current)
                }}
                aria-pressed={classroomMode}
                style={{
                  ...headerButtonStyle(false, classroomMode),
                  borderColor: classroomMode ? '#4F7BE8' : '#CBD5E1',
                  background: classroomMode ? '#EEF2FF' : '#FFFFFF',
                  color: classroomMode ? '#4338CA' : '#64748B',
                }}
              >
                {classroomMode ? '课堂大字：开' : '课堂大字'}
              </button>

              {preview.session && (
                <button
                  type="button"
                  onClick={endSession}
                  disabled={preview.sending}
                  style={headerButtonStyle(
                    preview.sending,
                    classroomMode,
                  )}
                >
                  结束
                </button>
              )}

              <button
                type="button"
                onClick={closePanel}
                style={headerButtonStyle(false, classroomMode)}
                aria-label="关闭教学智能体面板"
              >
                ✕
              </button>
            </div>
          </div>

          {preview.notice && (
            <div
              style={noticeStyle(
                preview.notice.kind,
                classroomMode,
              )}
            >
              {preview.notice.text}
            </div>
          )}

          {!preview.session && (
            <StartPanel
              pageTitle={pageTitle}
              currentVersion={preview.activeDeployment.current_version}
              maximumTurns={preview.activeDeployment.per_session_turn_limit}
              starting={preview.starting}
              classroomMode={classroomMode}
              onStart={() => {
                void preview.startPreview()
              }}
            />
          )}

          {preview.session && (
            <>
              <div style={sessionSummaryStyle(classroomMode)}>
                <span
                  style={{
                    color: '#475569',
                    fontSize: classroomMode ? 14 : 9.5,
                    fontWeight: 700,
                  }}
                >
                  教师预览 · 已完成{preview.session.turn_count}轮
                </span>

                <span
                  style={{
                    color: preview.remainingTurns > 0 ? '#059669' : '#B45309',
                    fontSize: classroomMode ? 14 : 9.5,
                    fontWeight: 850,
                  }}
                >
                  剩余{preview.remainingTurns}轮
                </span>
              </div>

              <div
                style={{
                  ...conversationStyle(classroomMode),
                  minHeight: classroomMode
                    ? variant === 'embedded'
                      ? 250
                      : 320
                    : variant === 'embedded'
                      ? 155
                      : 220,
                  maxHeight: classroomMode
                    ? variant === 'embedded'
                      ? 360
                      : 440
                    : variant === 'embedded'
                      ? 245
                      : 340,
                }}
              >
                {preview.messages.length === 0 && !preview.streamingText && (
                  <div
                    style={{
                      padding: classroomMode ? '42px 18px' : '28px 12px',
                      color: '#94A3B8',
                      fontSize: classroomMode ? 17 : 10,
                      lineHeight: 1.7,
                      textAlign: 'center',
                    }}
                  >
                    学习互动已开始。老师可以点击“说话”，直接提出问题或模拟学生回答。
                  </div>
                )}

                {preview.messages.map((message, index) => (
                  <MessageBubble
                    key={`${message.role}-${message.created_at || 'none'}-${index}`}
                    role={message.role}
                    content={message.content}
                    classroomMode={classroomMode}
                  />
                ))}

                {preview.streamingText && (
                  <MessageBubble
                    role="assistant"
                    content={preview.streamingText}
                    classroomMode={classroomMode}
                    streaming
                  />
                )}

                <div ref={messagesEndRef} />
              </div>

              <div style={composerStyle(classroomMode)}>
                <PlatformCoursewareAssistantVoiceControls
                  disabled={!preview.canSend}
                  classroomMode={classroomMode}
                  autoSpeak={autoSpeak}
                  speechSupported={speech.isSupported}
                  speechPreparing={speech.preparing}
                  speechSpeaking={speech.speaking}
                  speechPaused={speech.paused}
                  speechError={speech.error}
                  speechWarning={speech.warning}
                  speechProvider={speech.provider}
                  speechVoiceLabel={speech.voiceLabel}
                  canReplay={Boolean(speech.lastText)}
                  onInputChange={setInput}
                  onSubmitVoice={submitMessage}
                  onBeforeVoiceStart={speech.stop}
                  onToggleAutoSpeak={toggleAutoSpeak}
                  onToggleSpeechPause={speech.togglePause}
                  onStopSpeech={speech.stop}
                  onReplaySpeech={speech.replay}
                  onVoiceStatusChange={setVoiceStatus}
                />

                <div
                  style={{
                    color: '#64748B',
                    fontSize: classroomMode ? 14 : 10,
                    fontWeight: 750,
                  }}
                >
                  文字备用输入
                </div>

                <textarea
                  value={input}
                  onChange={event => {
                    setInput(event.target.value)
                  }}
                  onKeyDown={event => {
                    event.stopPropagation()

                    if (
                      (event.ctrlKey || event.metaKey)
                      && event.key === 'Enter'
                    ) {
                      event.preventDefault()
                      submitMessage(input)
                    }
                  }}
                  disabled={!preview.canSend}
                  rows={classroomMode ? 2 : 3}
                  maxLength={8000}
                  placeholder={
                    preview.session.status !== 'active'
                      ? '本次预览已结束，请重新开始'
                      : preview.remainingTurns <= 0
                        ? '本次互动轮数已用尽'
                        : '语音不可用时，可在这里输入老师的问题或学生的想法'
                  }
                  style={{
                    ...textareaStyle(classroomMode),
                    background: preview.canSend ? '#FFFFFF' : '#F1F5F9',
                  }}
                />

                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    gap: classroomMode ? 12 : 8,
                  }}
                >
                  <span
                    style={{
                      color: '#94A3B8',
                      fontSize: classroomMode ? 13 : 8.5,
                      lineHeight: 1.5,
                    }}
                  >
                    语音识别会自动发送；完整回答优先使用豆包自然音色朗读
                  </span>

                  <button
                    type="button"
                    onClick={() => {
                      submitMessage(input)
                    }}
                    disabled={!preview.canSend || !input.trim()}
                    style={primaryButtonStyle(
                      !preview.canSend || !input.trim(),
                      classroomMode,
                    )}
                  >
                    {preview.sending ? '正在回应…' : '发送文字'}
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  )
}

function StartPanel({
  pageTitle,
  currentVersion,
  maximumTurns,
  starting,
  classroomMode,
  onStart,
}: {
  pageTitle: string
  currentVersion: number
  maximumTurns: number
  starting: boolean
  classroomMode: boolean
  onStart: () => void
}) {
  return (
    <div style={{ padding: classroomMode ? 20 : 15 }}>
      <div
        style={{
          padding: classroomMode ? 20 : 14,
          borderRadius: classroomMode ? 17 : 13,
          border: '1px solid #DBEAFE',
          background: 'linear-gradient(145deg, #F8FAFF, #EFF6FF)',
        }}
      >
        <div
          style={{
            display: 'inline-flex',
            padding: classroomMode ? '5px 9px' : '3px 7px',
            borderRadius: 999,
            background: 'rgba(79,123,232,0.10)',
            color: '#4F7BE8',
            fontSize: classroomMode ? 13 : 8.5,
            fontWeight: 800,
          }}
        >
          老师课堂体验
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
          开始本页语音学习互动
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
            color: '#475569',
            fontSize: classroomMode ? 17 : 10.5,
            lineHeight: 1.75,
          }}
        >
          点击开始后，老师可以直接说话。系统识别完成会自动向教学智能体提问，
          回答完成后优先使用豆包自然音色朗读，并同时在屏幕上显示大字内容。
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

          <button
            type="button"
            onClick={onStart}
            disabled={starting}
            style={primaryButtonStyle(
              starting,
              classroomMode,
            )}
          >
            {starting ? '正在准备…' : '开始学习'}
          </button>
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
        ：这里使用真实发布版本进行体验。发送消息和豆包朗读会按照实际用量结算教师教学积分。
      </div>
    </div>
  )
}

function MessageBubble({
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

