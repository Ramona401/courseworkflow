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
 *   - 全屏和放映默认使用极简数字人课堂：人物靠右、当前回答气泡在左、麦克风与回答互斥；
 *   - 课堂数字人支持男女老师两套动作资产，人物选择只放在完整智能体中；
 *   - 数字人支持课堂隐藏和右侧边缘唤醒，隐藏期间不自动朗读新回答；
 *   - 完整智能体只在老师主动展开时显示历史、文字输入和语音高级控制，并与数字人互斥；
 *   - 嵌入预览继续保留完整聊天面板，便于老师调试；
 *   - 文字输入继续作为语音不可用或识别失败时的备用入口；
 *   - 面板关闭、页面切换和会话结束时停止当前朗读。
 *
 * UE边界：
 *   - 学生体验卡只展示学习任务、提示方式和互动轮数；
 *   - 教师身份、发布版本和积分说明放在学生体验卡之外；
 *   - 不在模拟学生界面中出现external、模型或结算技术术语；
 *   - 打开教学智能体后自动建立教师课堂预览会话，智能体先用已发布开场语主动问候；
 *   - 页面切换和组件卸载时由Hook清理令牌与流连接。
 */

import { useEffect, useRef, useState } from 'react'
import type { SyntheticEvent } from 'react'
import { useParams } from 'react-router-dom'

import useAssistantSpeechPlayback from '@/hooks/useAssistantSpeechPlayback'
import type { VoiceInputStatus } from '@/hooks/useVoiceInput'

import type { ClassroomDigitalHumanState } from './ClassroomDigitalHuman'
import CoursewareAssistantValidityRecoveryNotice from './CoursewareAssistantValidityRecoveryNotice'
import { MessageBubble, StartPanel } from './PlatformCoursewareAssistantOverlayParts'
import PlatformCoursewareAssistantClassroomStage
  from './PlatformCoursewareAssistantClassroomStage'
import PlatformCoursewareAssistantVoiceControls
  from './PlatformCoursewareAssistantVoiceControls'
import { useCoursewareAssistantPreview } from './useCoursewareAssistantPreview'
import { requestCoursewareAssistantValiditySettings } from './coursewareAssistantValidityNavigation'

import {
  composerStyle,
  conversationStyle,
  floatingButtonStyle,
  headerButtonStyle,
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

  const activeDeployment = preview.activeDeployment
  const validUntilTimestamp = activeDeployment?.valid_until
    ? Date.parse(activeDeployment.valid_until)
    : Number.NaN
  const deploymentExpired = Boolean(
    activeDeployment
    && Number.isFinite(validUntilTimestamp)
    && validUntilTimestamp <= Date.now(),
  )

  const [open, setOpen] = useState(false)
  const [input, setInput] = useState('')
  const [autoSpeak, setAutoSpeak] = useState(true)
  const [classroomMode, setClassroomMode] = useState(
    variant !== 'embedded',
  )
  const [voiceStatus, setVoiceStatus] = useState<VoiceInputStatus>('idle')
  const [classroomAssistantHidden, setClassroomAssistantHidden] = useState(false)

  const projected = variant !== 'embedded'
  const classroomHiddenStorageKey = projected && resolvedCoursewareID
    ? `tedna:courseware-assistant:hidden:${resolvedCoursewareID}`
    : ''

  const messagesEndRef = useRef<HTMLDivElement | null>(null)
  const handledReplySequenceRef = useRef(0)
  const autoStartAttemptRef = useRef('')
  const greetedSessionRef = useRef('')

  useEffect(() => {
    if (!classroomHiddenStorageKey || typeof window === 'undefined') {
      setClassroomAssistantHidden(false)
      return
    }

    try {
      setClassroomAssistantHidden(window.sessionStorage.getItem(classroomHiddenStorageKey) === '1')
    } catch {
      setClassroomAssistantHidden(false)
    }
  }, [classroomHiddenStorageKey])

  useEffect(() => {
    autoStartAttemptRef.current = ''
    greetedSessionRef.current = ''
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
      autoStartAttemptRef.current = ''
      return
    }

    if (
      !activeDeployment
      || deploymentExpired
      || preview.session
      || preview.starting
      || preview.deploymentLoading
    ) {
      return
    }

    const attemptKey = [
      resolvedCoursewareID,
      resolvedPageID,
      activeDeployment.id,
      activeDeployment.updated_at || String(activeDeployment.current_version),
    ].join(':')

    if (autoStartAttemptRef.current === attemptKey) {
      return
    }

    autoStartAttemptRef.current = attemptKey
    void preview.startPreview()
  }, [
    activeDeployment,
    deploymentExpired,
    open,
    preview.deploymentLoading,
    preview.session,
    preview.startPreview,
    preview.starting,
    resolvedCoursewareID,
    resolvedPageID,
  ])

  const sessionGreeting = preview.session
    ? (
        preview.messages.find(message => message.role === 'assistant')?.content.trim()
        || `同学们好，我们一起来学习${pageTitle ? `《${pageTitle}》` : '这一页'}。你可以直接说出你的想法。`
      )
    : ''

  useEffect(() => {
    const sessionID = preview.session?.id || ''

    if (
      !open
      || !sessionID
      || !sessionGreeting
      || greetedSessionRef.current === sessionID
    ) {
      return
    }

    greetedSessionRef.current = sessionID

    if (autoSpeak && !classroomAssistantHidden) {
      speech.speak(sessionGreeting)
    }
  }, [
    autoSpeak,
    classroomAssistantHidden,
    open,
    preview.session,
    sessionGreeting,
    speech.speak,
  ])

  useEffect(() => {
    if (!open) {
      speech.stop()
      return
    }

    if (projected) {
      return
    }

    messagesEndRef.current?.scrollIntoView({
      block: 'end',
    })
  }, [
    open,
    preview.messages,
    preview.streamingText,
    projected,
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

    if (
      open
      && autoSpeak
      && !classroomAssistantHidden
    ) {
      speech.speak(completedReply.text)
    }
  }, [
    autoSpeak,
    classroomAssistantHidden,
    open,
    preview.completedReply,
    speech.speak,
  ])

  if (
    !resolvedCoursewareID
    || !resolvedPageID
    || !activeDeployment
  ) {
    return null
  }

  const openValiditySettings = () => {
    speech.stop()
    const opened = requestCoursewareAssistantValiditySettings(resolvedCoursewareID, resolvedPageID)
    if (opened) setOpen(false)
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

  const writeClassroomHiddenPreference = (hidden: boolean) => {
    if (!classroomHiddenStorageKey || typeof window === 'undefined') return

    try {
      window.sessionStorage.setItem(classroomHiddenStorageKey, hidden ? '1' : '0')
    } catch {
      // sessionStorage不可用时只保留当前组件内存状态。
    }
  }

  const hideClassroomAssistant = () => {
    speech.stop()
    setVoiceStatus('idle')
    setClassroomAssistantHidden(true)
    writeClassroomHiddenPreference(true)
  }

  const wakeClassroomAssistant = () => {
    setVoiceStatus('idle')
    setClassroomAssistantHidden(false)
    writeClassroomHiddenPreference(false)
  }

  const endSession = () => {
    speech.stop()
    setVoiceStatus('idle')
    preview.clearSession()
    setInput('')
    setOpen(false)
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

  const classroomSubtitleText = preview.streamingText.trim()
    ? preview.streamingText
    : preview.sending
      ? ''
      : preview.completedReply?.text || sessionGreeting

  return (
    <div
      data-courseware-assistant-overlay-root="true"
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
            if (projected && classroomAssistantHidden) {
              wakeClassroomAssistant()
            }

            setOpen(true)
          }}
          title={
            projected && classroomAssistantHidden
              ? '唤醒当前页面的课堂助教'
              : '打开当前页面的教学智能体'
          }
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

          <span>
            {projected && classroomAssistantHidden
              ? '唤醒助教'
              : '教学智能体'}
          </span>

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

      {open && projected && (
        <PlatformCoursewareAssistantClassroomStage
          pageTitle={pageTitle}
          variant={variant}
          classroomMode={classroomMode}
          hidden={classroomAssistantHidden}
          digitalHumanState={digitalHumanState}
          audioElement={speech.audioElement}
          speechProvider={speech.provider}
          speechPaused={speech.paused}
          sessionPresent={Boolean(preview.session)}
          sessionStatus={preview.session?.status || ''}
          currentVersion={activeDeployment.current_version}
          maximumTurns={activeDeployment.per_session_turn_limit}
          turnCount={preview.session?.turn_count || 0}
          remainingTurns={preview.remainingTurns}
          starting={preview.starting}
          sending={preview.sending}
          canSend={preview.canSend}
          notice={deploymentExpired
            ? { kind: 'error', text: '教学智能体使用时间已到，请修改课堂使用时间后继续。' }
            : preview.notice}
          messages={preview.messages}
          streamingText={preview.streamingText}
          subtitleText={classroomSubtitleText}
          input={input}
          autoSpeak={autoSpeak}
          speechSupported={speech.isSupported}
          speechPreparing={speech.preparing}
          speechSpeaking={speech.speaking}
          speechError={speech.error}
          speechWarning={speech.warning}
          speechVoiceLabel={speech.voiceLabel}
          canReplay={Boolean(speech.lastText)}
          onStart={deploymentExpired ? openValiditySettings : () => { void preview.startPreview() }}
          onEndSession={endSession}
          onClose={closePanel}
          onHide={hideClassroomAssistant}
          onWake={wakeClassroomAssistant}
          onToggleClassroomMode={() => {
            setClassroomMode(current => !current)
          }}
          onInputChange={setInput}
          onSubmitMessage={submitMessage}
          onBeforeVoiceStart={speech.stop}
          onToggleAutoSpeak={toggleAutoSpeak}
          onToggleSpeechPause={speech.togglePause}
          onStopSpeech={speech.stop}
          onReplaySpeech={speech.replay}
          onVoiceStatusChange={setVoiceStatus}
        />
      )}

      {open && !projected && (
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

          {deploymentExpired ? (
            <CoursewareAssistantValidityRecoveryNotice
              classroomMode={classroomMode}
              onAdjust={openValiditySettings}
            />
          ) : preview.notice ? (
            <div style={noticeStyle(preview.notice.kind, classroomMode)}>
              {preview.notice.text}
            </div>
          ) : null}

          {!preview.session && (
            <StartPanel
              pageTitle={pageTitle}
              currentVersion={activeDeployment.current_version}
              maximumTurns={activeDeployment.per_session_turn_limit}
              starting={preview.starting}
              expired={deploymentExpired}
              classroomMode={classroomMode}
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
