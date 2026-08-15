/**
 * PlatformCoursewareAssistantClassroomStage.tsx
 *
 * 全屏预览和真全屏放映使用的极简课堂数字人舞台。
 *
 * 默认课堂原则：
 * - 学生侧只看到靠右数字人，以及左侧的实时聆听声波或当前回答气泡；
 * - 麦克风和回答气泡按状态互斥，避免课堂界面同时出现多套控件；
 * - 数字人可以隐藏，隐藏后只在屏幕右侧留下轻量唤醒入口；
 * - 完整智能体与数字人课堂互斥，历史、文字输入和高级语音控制只在展开后出现；
 * - 完整智能体内可切换女老师/男老师，两套人物共用同一套动作与语音状态机；
 * - 打开课堂后由外层自动建立会话并主动问候，本组件不再要求用户点击“开始课堂”；
 * - 本组件只编排课堂UI，不拥有ASR、会话或TTS业务状态。
 */

import { useEffect, useState } from 'react'

import type { AssistantSpeechProvider } from '@/hooks/useAssistantSpeechPlayback'
import type { VoiceInputStatus } from '@/hooks/useVoiceInput'

import ClassroomDigitalHuman from './ClassroomDigitalHuman'
import type { ClassroomDigitalHumanState } from './ClassroomDigitalHuman'
import PlatformCoursewareAssistantConversationDrawer
  from './PlatformCoursewareAssistantConversationDrawer'
import type { ClassroomConversationMessage }
  from './PlatformCoursewareAssistantConversationDrawer'
import PlatformCoursewareAssistantVoiceControls
  from './PlatformCoursewareAssistantVoiceControls'
import PlatformCoursewareAssistantClassroomEntry
  from './PlatformCoursewareAssistantClassroomEntry'

import {
  headerButtonStyle,
} from './PlatformCoursewareAssistantOverlay.styles'

import type {
  PlatformAssistantOverlayVariant,
} from './PlatformCoursewareAssistantOverlay.styles'

interface ClassroomNotice {
  kind: 'info' | 'success' | 'error'
  text: string
}

export type ClassroomDigitalHumanCharacter = 'female' | 'male'

const CLASSROOM_DIGITAL_HUMAN_CHARACTER_KEY = 'tedna:classroom-digital-human:character'

function readClassroomDigitalHumanCharacter(): ClassroomDigitalHumanCharacter {
  if (typeof window === 'undefined') return 'female'

  try {
    return window.localStorage.getItem(CLASSROOM_DIGITAL_HUMAN_CHARACTER_KEY) === 'male'
      ? 'male'
      : 'female'
  } catch {
    return 'female'
  }
}

function persistClassroomDigitalHumanCharacter(
  character: ClassroomDigitalHumanCharacter,
): void {
  if (typeof window === 'undefined') return

  try {
    window.localStorage.setItem(CLASSROOM_DIGITAL_HUMAN_CHARACTER_KEY, character)
  } catch {
    // localStorage不可用时只保留当前课堂页面内存选择。
  }
}

interface PlatformCoursewareAssistantClassroomStageProps {
  pageTitle: string
  variant: PlatformAssistantOverlayVariant
  classroomMode: boolean
  hidden: boolean
  digitalHumanState: ClassroomDigitalHumanState
  audioElement: HTMLAudioElement | null
  speechProvider: AssistantSpeechProvider
  speechPaused: boolean
  sessionPresent: boolean
  sessionStatus: string
  currentVersion: number
  maximumTurns: number
  turnCount: number
  remainingTurns: number
  starting: boolean
  sending: boolean
  canSend: boolean
  notice: ClassroomNotice | null
  messages: ClassroomConversationMessage[]
  streamingText: string
  subtitleText: string
  input: string
  autoSpeak: boolean
  speechSupported: boolean
  speechPreparing: boolean
  speechSpeaking: boolean
  speechError: string
  speechWarning: string
  speechVoiceLabel: string
  canReplay: boolean
  onStart: () => void
  onEndSession: () => void
  onClose: () => void
  onHide: () => void
  onWake: () => void
  onToggleClassroomMode: () => void
  onInputChange: (text: string) => void
  onSubmitMessage: (text: string) => boolean
  onBeforeVoiceStart: () => void
  onToggleAutoSpeak: () => void
  onToggleSpeechPause: () => void
  onStopSpeech: () => void
  onReplaySpeech: () => void
  onVoiceStatusChange: (status: VoiceInputStatus) => void
}

function normalizeCaption(
  rawText: string,
  state: ClassroomDigitalHumanState,
): string {
  const normalized = rawText
    .replace(/```[\s\S]*?```/g, '代码内容请看屏幕。')
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/https?:\/\/\S+/g, '链接内容请看屏幕。')
    .replace(/[`*_>#|~]+/g, '')
    .replace(/\s+/g, ' ')
    .trim()

  if (normalized) {
    return normalized
  }

  if (state === 'thinking') {
    return '•••'
  }

  return ''
}

function noticeTone(kind: ClassroomNotice['kind']) {
  switch (kind) {
  case 'success':
    return {
      background: 'rgba(236,253,245,0.96)',
      color: '#047857',
      borderColor: 'rgba(16,185,129,0.24)',
    }
  case 'error':
    return {
      background: 'rgba(254,242,242,0.97)',
      color: '#B91C1C',
      borderColor: 'rgba(239,68,68,0.26)',
    }
  default:
    return {
      background: 'rgba(239,246,255,0.96)',
      color: '#1D4ED8',
      borderColor: 'rgba(59,130,246,0.22)',
    }
  }
}

export default function PlatformCoursewareAssistantClassroomStage({
  pageTitle,
  variant,
  classroomMode,
  hidden,
  digitalHumanState,
  audioElement,
  speechProvider,
  speechPaused,
  sessionPresent,
  sessionStatus,
  currentVersion,
  maximumTurns,
  turnCount,
  remainingTurns,
  starting,
  sending,
  canSend,
  notice,
  messages,
  streamingText,
  subtitleText,
  input,
  autoSpeak,
  speechSupported,
  speechPreparing,
  speechSpeaking,
  speechError,
  speechWarning,
  speechVoiceLabel,
  canReplay,
  onStart,
  onEndSession,
  onClose,
  onHide,
  onWake,
  onToggleClassroomMode,
  onInputChange,
  onSubmitMessage,
  onBeforeVoiceStart,
  onToggleAutoSpeak,
  onToggleSpeechPause,
  onStopSpeech,
  onReplaySpeech,
  onVoiceStatusChange,
}: PlatformCoursewareAssistantClassroomStageProps) {
  const [conversationOpen, setConversationOpen] = useState(false)
  const [responseLingering, setResponseLingering] = useState(false)
  const [character, setCharacter] = useState<ClassroomDigitalHumanCharacter>(
    readClassroomDigitalHumanCharacter,
  )

  const selectCharacter = (nextCharacter: ClassroomDigitalHumanCharacter) => {
    if (nextCharacter === character) {
      return
    }

    onStopSpeech()
    setResponseLingering(false)
    setCharacter(nextCharacter)
    persistClassroomDigitalHumanCharacter(nextCharacter)
  }

  useEffect(() => {
    if (!sessionPresent) {
      setConversationOpen(false)
      setResponseLingering(false)
    }
  }, [sessionPresent])

  const listening = digitalHumanState === 'listening'
  const responseActive = sending
    || speechPreparing
    || speechSpeaking
    || Boolean(streamingText.trim())

  useEffect(() => {
    if (listening) {
      setResponseLingering(false)
      return
    }

    if (responseActive) {
      setResponseLingering(true)
      return
    }

    if (!responseLingering || !subtitleText.trim()) {
      return
    }

    const timer = window.setTimeout(() => {
      setResponseLingering(false)
    }, 2400)

    return () => {
      window.clearTimeout(timer)
    }
  }, [
    listening,
    responseActive,
    responseLingering,
    subtitleText,
  ])

  const caption = normalizeCaption(
    subtitleText,
    digitalHumanState,
  )

  const showResponseBubble = sessionPresent
    && !hidden
    && !conversationOpen
    && !listening
    && (responseActive || responseLingering)

  const showMinimalMicrophone = sessionPresent
    && !hidden
    && !conversationOpen
    && !showResponseBubble

  const detailFontSize = classroomMode ? 13 : 11
  const fullPanelWidth = classroomMode ? 540 : 470
  const errorTone = notice?.kind === 'error'
    ? noticeTone('error')
    : null

  const handleBeforeVoiceStart = () => {
    setResponseLingering(false)
    onBeforeVoiceStart()
  }

  const toggleConversation = () => {
    setConversationOpen(current => {
      const next = !current

      if (next) {
        setResponseLingering(false)
        onStopSpeech()
      }

      return next
    })
  }

  if (hidden) {
    return (
      <div
        role="group"
        aria-label="课堂助教已隐藏"
        style={{
          position: 'fixed',
          right: 0,
          bottom: variant === 'slideshow'
            ? classroomMode ? 92 : 80
            : classroomMode ? 30 : 24,
          zIndex: 10,
          pointerEvents: 'none',
        }}
      >
        <button
          type="button"
          onClick={onWake}
          title="唤醒课堂助教"
          aria-label="唤醒课堂助教"
          style={{
            width: classroomMode ? 58 : 50,
            minHeight: classroomMode ? 78 : 68,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 4,
            padding: classroomMode ? '9px 6px' : '7px 5px',
            borderRadius: classroomMode
              ? '18px 0 0 18px'
              : '15px 0 0 15px',
            border: '1px solid rgba(148,163,184,0.32)',
            borderRight: 'none',
            background: 'rgba(255,255,255,0.84)',
            color: '#475569',
            boxShadow: '0 8px 26px rgba(15,23,42,0.14)',
            backdropFilter: 'blur(12px)',
            fontSize: classroomMode ? 12 : 10,
            fontWeight: 800,
            lineHeight: 1.2,
            cursor: 'pointer',
            pointerEvents: 'auto',
          }}
        >
          <span
            aria-hidden="true"
            style={{
              fontSize: classroomMode ? 24 : 20,
              lineHeight: 1,
            }}
          >
            {character === 'male' ? '👨‍🏫' : '👩‍🏫'}
          </span>
          <span>助教</span>
        </button>
      </div>
    )
  }

  return (
    <div
      role="dialog"
      aria-label="教学智能体课堂数字人"
      style={{
        display: 'flex',
        alignItems: 'flex-end',
        justifyContent: 'flex-end',
        gap: classroomMode ? 22 : 16,
        maxWidth: 'calc(100vw - 36px)',
        pointerEvents: 'none',
      }}
    >
      <div
        style={{
          width: conversationOpen
            ? fullPanelWidth
            : classroomMode
              ? 560
              : 480,
          maxWidth: conversationOpen
            ? `min(${fullPanelWidth}px, calc(100vw - 320px))`
            : 'min(560px, 42vw)',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-end',
          gap: classroomMode ? 12 : 9,
          pointerEvents: 'none',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            gap: 7,
            pointerEvents: 'auto',
          }}
        >
          <button
            type="button"
            onClick={toggleConversation}
            style={{
              ...headerButtonStyle(false, false),
              minHeight: classroomMode ? 36 : 32,
              padding: classroomMode ? '7px 11px' : '6px 9px',
              borderColor: conversationOpen
                ? 'rgba(79,123,232,0.38)'
                : 'rgba(255,255,255,0.62)',
              background: conversationOpen
                ? 'rgba(238,242,255,0.97)'
                : 'rgba(255,255,255,0.90)',
              color: conversationOpen ? '#4338CA' : '#475569',
              boxShadow: '0 6px 20px rgba(15,23,42,0.12)',
              backdropFilter: 'blur(10px)',
              fontSize: classroomMode ? 13 : 11,
            }}
          >
            {conversationOpen ? '返回课堂' : '完整智能体'}
          </button>

          <button
            type="button"
            onClick={conversationOpen ? onClose : onHide}
            style={{
              ...headerButtonStyle(false, false),
              minWidth: conversationOpen
                ? classroomMode ? 36 : 32
                : classroomMode ? 76 : 66,
              minHeight: classroomMode ? 36 : 32,
              padding: conversationOpen
                ? 0
                : classroomMode ? '7px 10px' : '6px 8px',
              borderColor: 'rgba(255,255,255,0.62)',
              background: 'rgba(255,255,255,0.82)',
              color: '#64748B',
              boxShadow: '0 6px 20px rgba(15,23,42,0.10)',
              backdropFilter: 'blur(10px)',
              fontSize: classroomMode ? 13 : 11,
            }}
            aria-label={conversationOpen ? '关闭教学智能体' : '隐藏课堂助教'}
            title={conversationOpen ? '关闭教学智能体' : '隐藏课堂助教'}
          >
            {conversationOpen ? '✕' : '隐藏助教'}
          </button>
        </div>

        {!conversationOpen && !sessionPresent && (
          <PlatformCoursewareAssistantClassroomEntry
            mode="minimal"
            classroomMode={classroomMode}
            currentVersion={currentVersion}
            maximumTurns={maximumTurns}
            starting={starting}
            notice={notice}
            onRecover={onStart}
          />
        )}

        {!conversationOpen && sessionPresent && notice?.kind === 'error' && errorTone && (
          <div
            style={{
              maxWidth: classroomMode ? 500 : 410,
              padding: classroomMode ? '10px 13px' : '8px 10px',
              borderRadius: 13,
              border: `1px solid ${errorTone.borderColor}`,
              background: errorTone.background,
              color: errorTone.color,
              boxShadow: '0 8px 24px rgba(15,23,42,0.12)',
              backdropFilter: 'blur(10px)',
              fontSize: classroomMode ? 14 : 11,
              lineHeight: 1.55,
              pointerEvents: 'auto',
            }}
          >
            {notice.text}
          </div>
        )}

        {showResponseBubble && (
          <div
            aria-live="polite"
            style={{
              width: 'fit-content',
              maxWidth: '100%',
              minWidth: caption === '•••'
                ? classroomMode ? 86 : 72
                : classroomMode ? 220 : 180,
              padding: caption === '•••'
                ? classroomMode ? '12px 22px' : '10px 18px'
                : classroomMode ? '17px 20px' : '13px 16px',
              borderRadius: classroomMode ? '22px 22px 6px 22px' : '18px 18px 5px 18px',
              border: '1px solid rgba(255,255,255,0.74)',
              background: 'rgba(255,255,255,0.94)',
              color: '#172033',
              boxShadow: '0 16px 44px rgba(15,23,42,0.22)',
              backdropFilter: 'blur(14px)',
              fontSize: classroomMode ? 22 : 17,
              fontWeight: 650,
              lineHeight: classroomMode ? 1.62 : 1.58,
              textAlign: caption === '•••' ? 'center' : 'left',
              wordBreak: 'break-word',
              maxHeight: classroomMode ? '44vh' : '38vh',
              overflowY: 'auto',
              scrollbarWidth: 'thin',
              pointerEvents: 'auto',
            }}
          >
            {caption || '•••'}
          </div>
        )}

        {showMinimalMicrophone && (
          <div
            style={{
              pointerEvents: 'auto',
            }}
          >
            <PlatformCoursewareAssistantVoiceControls
              minimal
              disabled={!canSend}
              classroomMode={classroomMode}
              autoSpeak={autoSpeak}
              speechSupported={speechSupported}
              speechPreparing={speechPreparing}
              speechSpeaking={speechSpeaking}
              speechPaused={speechPaused}
              speechError={speechError}
              speechWarning={speechWarning}
              speechProvider={speechProvider}
              speechVoiceLabel={speechVoiceLabel}
              canReplay={canReplay}
              onInputChange={onInputChange}
              onSubmitVoice={onSubmitMessage}
              onBeforeVoiceStart={handleBeforeVoiceStart}
              onToggleAutoSpeak={onToggleAutoSpeak}
              onToggleSpeechPause={onToggleSpeechPause}
              onStopSpeech={onStopSpeech}
              onReplaySpeech={onReplaySpeech}
              onVoiceStatusChange={onVoiceStatusChange}
            />
          </div>
        )}

        {conversationOpen && (
          <div
            style={{
              width: '100%',
              overflow: 'hidden',
              borderRadius: classroomMode ? 20 : 16,
              border: '1px solid rgba(148,163,184,0.40)',
              background: 'rgba(255,255,255,0.97)',
              boxShadow: '0 18px 55px rgba(15,23,42,0.24)',
              backdropFilter: 'blur(14px)',
              pointerEvents: 'auto',
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: 12,
                padding: classroomMode ? '12px 14px' : '9px 11px',
                borderBottom: '1px solid rgba(226,232,240,0.94)',
                background: 'linear-gradient(135deg, rgba(238,242,255,0.96), rgba(248,250,252,0.96))',
              }}
            >
              <div style={{ minWidth: 0 }}>
                <div
                  style={{
                    color: '#1E293B',
                    fontSize: classroomMode ? 17 : 14,
                    fontWeight: 850,
                  }}
                >
                  完整智能体
                </div>

                <div
                  title={pageTitle}
                  style={{
                    maxWidth: classroomMode ? 260 : 210,
                    marginTop: 3,
                    overflow: 'hidden',
                    color: '#64748B',
                    fontSize: detailFontSize,
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
                  gap: 6,
                  flexShrink: 0,
                }}
              >
                <button
                  type="button"
                  onClick={onToggleClassroomMode}
                  aria-pressed={classroomMode}
                  style={{
                    ...headerButtonStyle(false, classroomMode),
                    borderColor: classroomMode ? '#4F7BE8' : '#CBD5E1',
                    background: classroomMode ? '#EEF2FF' : '#FFFFFF',
                    color: classroomMode ? '#4338CA' : '#64748B',
                  }}
                >
                  {classroomMode ? '大字幕：开' : '大字幕'}
                </button>

                {sessionPresent && (
                  <button
                    type="button"
                    onClick={onEndSession}
                    disabled={sending}
                    style={headerButtonStyle(
                      sending,
                      classroomMode,
                    )}
                  >
                    结束
                  </button>
                )}
              </div>
            </div>

            <div
              style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                gap: 10, padding: classroomMode ? '9px 14px' : '8px 11px',
                borderBottom: '1px solid rgba(226,232,240,0.88)',
                background: 'rgba(248,250,252,0.82)',
              }}
            >
              <span style={{
                color: '#64748B', fontSize: classroomMode ? 13 : 11, fontWeight: 750,
              }}>
                课堂老师
              </span>

              <div
                role="group"
                aria-label="选择课堂数字人"
                style={{ display: 'flex', alignItems: 'center', gap: 6 }}
              >
                {([
                  ['female', '👩‍🏫 女老师'],
                  ['male', '👨‍🏫 男老师'],
                ] as const).map(([value, label]) => {
                  const selected = character === value
                  return (
                    <button
                      key={value}
                      type="button"
                      onClick={() => selectCharacter(value)}
                      aria-pressed={selected}
                      style={{
                        minHeight: classroomMode ? 34 : 30,
                        padding: classroomMode ? '6px 10px' : '5px 8px',
                        borderRadius: 9,
                        border: selected
                          ? '1px solid rgba(79,123,232,0.48)'
                          : '1px solid #CBD5E1',
                        background: selected ? '#EEF2FF' : '#FFFFFF',
                        color: selected ? '#4338CA' : '#64748B',
                        fontSize: classroomMode ? 13 : 11,
                        fontWeight: selected ? 850 : 700,
                        cursor: 'pointer',
                      }}
                    >
                      {label}
                    </button>
                  )
                })}
              </div>
            </div>

            {notice && (
              <div
                style={{
                  margin: classroomMode ? '10px 12px 0' : '8px 9px 0',
                  padding: classroomMode ? '9px 11px' : '7px 9px',
                  borderRadius: 10,
                  border: `1px solid ${noticeTone(notice.kind).borderColor}`,
                  background: noticeTone(notice.kind).background,
                  color: noticeTone(notice.kind).color,
                  fontSize: classroomMode ? 14 : 11,
                  lineHeight: 1.55,
                }}
              >
                {notice.text}
              </div>
            )}

            {!sessionPresent && (
              <PlatformCoursewareAssistantClassroomEntry
                mode="panel"
                classroomMode={classroomMode}
                currentVersion={currentVersion}
                maximumTurns={maximumTurns}
                starting={starting}
                notice={notice}
                onRecover={onStart}
              />
            )}

            {sessionPresent && (
              <>
                <PlatformCoursewareAssistantConversationDrawer
                  classroomMode={classroomMode}
                  messages={messages}
                  streamingText={streamingText}
                  input={input}
                  sessionStatus={sessionStatus}
                  remainingTurns={remainingTurns}
                  turnCount={turnCount}
                  sending={sending}
                  canSend={canSend}
                  onInputChange={onInputChange}
                  onSubmitMessage={onSubmitMessage}
                />

                <div
                  style={{
                    padding: classroomMode ? '0 14px 14px' : '0 10px 10px',
                  }}
                >
                  <PlatformCoursewareAssistantVoiceControls
                    disabled={!canSend}
                    classroomMode={classroomMode}
                    autoSpeak={autoSpeak}
                    speechSupported={speechSupported}
                    speechPreparing={speechPreparing}
                    speechSpeaking={speechSpeaking}
                    speechPaused={speechPaused}
                    speechError={speechError}
                    speechWarning={speechWarning}
                    speechProvider={speechProvider}
                    speechVoiceLabel={speechVoiceLabel}
                    canReplay={canReplay}
                    onInputChange={onInputChange}
                    onSubmitVoice={onSubmitMessage}
                    onBeforeVoiceStart={handleBeforeVoiceStart}
                    onToggleAutoSpeak={onToggleAutoSpeak}
                    onToggleSpeechPause={onToggleSpeechPause}
                    onStopSpeech={onStopSpeech}
                    onReplaySpeech={onReplaySpeech}
                    onVoiceStatusChange={onVoiceStatusChange}
                  />
                </div>
              </>
            )}
          </div>
        )}
      </div>

      {!conversationOpen && (
        <ClassroomDigitalHuman
          state={digitalHumanState}
          character={character}
          speechText={subtitleText}
          audioElement={audioElement}
          speechProvider={speechProvider}
          speechPaused={speechPaused}
          classroomMode={classroomMode}
          variant={variant}
        />
      )}
    </div>
  )
}
