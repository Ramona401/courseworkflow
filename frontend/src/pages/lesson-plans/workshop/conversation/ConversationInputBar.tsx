/**
 * ConversationInputBar.tsx — 对话模式底部输入区
 *
 * 语音输入规则：
 *   1. 点击麦克风开始录音；
 *   2. partial实时写入同一受保护草稿；
 *   3. 点击停止后等待final；
 *   4. final只进入输入框，不自动发送AI；
 *   5. 识别失败恢复录音前草稿；
 *   6. AI忙碌时禁止启动语音。
 *
 * 资源提示规则：
 *   - 不展示组件数量、附件文件名或挂载状态码；
 *   - 只说明这些资源会怎样帮助下一轮备课；
 *   - 清空和移除入口继续保留。
 */

import {
  forwardRef,
  useCallback,
  useImperativeHandle,
  useRef,
  useState,
} from 'react'
import { useAuth } from '@/store/auth'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import { useVoiceInput } from '@/hooks/useVoiceInput'
import VoiceInputButton from '@/components/voice/VoiceInputButton'
import { C } from '../components/workshopConstants'
import { PLUS_MENU_ITEMS } from './conversationScript'
import ResourceMeaningPill from './ResourceMeaningPill'
import {
  isConversationPublishIntent,
} from './conversationActionIntent'
import {
  useStageContinuationActivity,
} from './stageContinuationActivity'
import ConversationAttachmentInput, {
  type ConversationAttachmentInputHandle,
} from './ConversationAttachmentInput'

export interface ConversationInputBarHandle {
  focus: () => void
  prefill: (text: string) => void
}

export interface ConversationInputBarProps {
  isBusy: boolean
  placeholder: string

  /**
   * 已选择组件数量。
   *
   * 仅用于判断是否存在待使用的教学策略，
   * 教师界面不展示具体数量。
   */
  selectedCount: number

  onClearSelected: () => void
  onSend: (
    text: string,
  ) => void | boolean | Promise<void | boolean>
  hasContent: boolean
  onPublish: () => void
  plusItemAvailability: (
    tool: string,
  ) => {
    visible?: boolean
    enabled: boolean
    reason: string
  }
  onOpenTool: (tool: string) => void

  /**
   * 参考资料名称。
   *
   * 只用于判断补充依据是否存在，不在输入区展示文件名。
   */
  refMaterialName?: string

  onClearRefMaterial?: () => void
}

function getActiveConversationPlanID(): string {
  try {
    return sessionStorage.getItem(
      'workshop_active_plan_id',
    ) || 'current-plan'
  } catch {
    return 'current-plan'
  }
}

function mergeVoiceText(
  base: string,
  speech: string,
): string {
  const normalized = speech.trim()

  if (!normalized) {
    return base
  }

  if (!base) {
    return normalized
  }

  const needsSpace =
    /[A-Za-z0-9]$/.test(base) &&
    /^[A-Za-z0-9]/.test(normalized)

  return base +
    (needsSpace ? ' ' : '') +
    normalized
}

const ConversationInputBar = forwardRef<
  ConversationInputBarHandle,
  ConversationInputBarProps
>(function ConversationInputBar(
  props,
  ref,
) {
  const {
    isBusy,
    placeholder,
    selectedCount,
    onClearSelected,
    onSend,
    hasContent,
    onPublish,
    plusItemAvailability,
    onOpenTool,
    refMaterialName,
    onClearRefMaterial,
  } = props

  const { user } = useAuth()
  const stageContinuationActive =
    useStageContinuationActivity()
  const activePlanID =
    getActiveConversationPlanID()

  const {
    value: inputText,
    setValue: setInputText,
    commit: commitInputDraft,
    handleKeyDown: handleDraftKeyDown,
  } = useProtectedDraft({
    userId: user?.id,
    scope: 'lesson-plan-conversation',
    resourceId: activePlanID,
    field: 'message',
    initialValue: '',
    maxHistory: 40,
  })

  const taRef =
    useRef<HTMLTextAreaElement>(null)

  const attachmentInputRef =
    useRef<ConversationAttachmentInputHandle>(null)

  const [
    attachmentBlocking,
    setAttachmentBlocking,
  ] = useState(false)

  /**
   * 发送和发布共用单次执行锁。
   *
   * React忙碌状态更新前的快速双击也不能重复发消息或重复弹发布确认。
   */
  const submitInFlightRef =
    useRef(false)

  const voiceBaseTextRef =
    useRef('')

  const handleVoicePartial = useCallback(
    (text: string) => {
      setInputText(
        mergeVoiceText(
          voiceBaseTextRef.current,
          text,
        ),
      )
    },
    [setInputText],
  )

  const handleVoiceFinal = useCallback(
    (text: string) => {
      const merged = mergeVoiceText(
        voiceBaseTextRef.current,
        text,
      )

      setInputText(merged)

      requestAnimationFrame(() => {
        const element = taRef.current

        if (!element) {
          return
        }

        element.focus()
        element.setSelectionRange(
          merged.length,
          merged.length,
        )
      })
    },
    [setInputText],
  )

  const handleVoiceError = useCallback(
    () => {
      setInputText(
        voiceBaseTextRef.current,
      )
    },
    [setInputText],
  )

  const voice = useVoiceInput({
    disabled:
      isBusy ||
      stageContinuationActive,
    maxDurationSeconds: 120,
    onPartial: handleVoicePartial,
    onFinal: handleVoiceFinal,
    onError: handleVoiceError,
  })

  const beginVoiceInput = useCallback(
    () => {
      voiceBaseTextRef.current =
        inputText

      void voice.start()
    },
    [inputText, voice.start],
  )

  useImperativeHandle(
    ref,
    () => ({
      focus: () =>
        taRef.current?.focus(),

      prefill: (text: string) => {
        setInputText(text)

        requestAnimationFrame(() => {
          const element = taRef.current

          if (!element) {
            return
          }

          element.focus()
          element.setSelectionRange(
            text.length,
            text.length,
          )
        })
      },
    }),
    [setInputText],
  )

  const [
    showPlusMenu,
    setShowPlusMenu,
  ] = useState(false)

  const inputDisabled =
    isBusy ||
    stageContinuationActive ||
    voice.isActive

  /*
   * 附件处理中仍允许老师继续打字、继续追加文件。
   * 只有真正发送动作需要等待附件处理完成，交互更接近对话composer。
   */
  const sendDisabled =
    inputDisabled ||
    attachmentBlocking

  const hasSelectedStrategies =
    selectedCount > 0

  const hasReferenceEvidence =
    Boolean(refMaterialName)

  const textbookAvailability =
    plusItemAvailability('textbook')

  const doSend = async () => {
    const text = inputText.trim()

    if (
      !text ||
      sendDisabled ||
      submitInFlightRef.current
    ) {
      return
    }

    submitInFlightRef.current = true

    try {
      /*
       * 明确的定稿或发布确认语不进入AI聊天。
       *
       * 直接复用页面发布流程：
       *   - 不生成用户聊天气泡；
       *   - 不启动Harness；
       *   - 不清空草稿，发布取消时老师仍可继续编辑；
       *   - 发布成功后页面会离开当前工坊。
       */
      if (
        isConversationPublishIntent(
          text,
        )
      ) {
        await Promise.resolve(
          onPublish(),
        )
        return
      }

      const accepted =
        await Promise.resolve(
          onSend(text),
        )

      if (accepted !== false) {
        commitInputDraft()
      }
    } catch (error) {
      console.error(
        '备课消息发送或发布未被受理，草稿已保留:',
        error,
      )
    } finally {
      submitInFlightRef.current = false
    }
  }

  let voiceStatusText = ''

  if (voice.status === 'connecting') {
    voiceStatusText =
      '正在连接语音识别…'
  } else if (
    voice.status === 'recording'
  ) {
    const minute = Math.floor(
      voice.elapsedSeconds / 60,
    )

    const second = String(
      voice.elapsedSeconds % 60,
    ).padStart(2, '0')

    voiceStatusText =
      `正在听写 ${minute}:${second} · 点击红色按钮停止`
  } else if (
    voice.status === 'stopping'
  ) {
    voiceStatusText =
      '正在整理最终文字…'
  } else if (
    voice.status === 'error' &&
    voice.error
  ) {
    voiceStatusText =
      `语音输入未完成：${voice.error}`
  }

  return (
    <>
      <ConversationAttachmentInput
        ref={attachmentInputRef}
        planID={activePlanID}
        textbookEnabled={
          textbookAvailability.visible !== false &&
          textbookAvailability.enabled
        }
        onBlockingChange={setAttachmentBlocking}
        onOpenTextbook={() => onOpenTool('textbook')}
      />

      {hasSelectedStrategies && (
        <ResourceMeaningPill
          variant="strip"
          icon="🧩"
          label="已加入教学策略，下一条会结合使用"
          title="已选择的专业策略会在下一轮帮助AI组织教学设计"
          tone="strategy"
          onClear={onClearSelected}
          clearLabel="清空"
        />
      )}

      {hasReferenceEvidence && (
        <ResourceMeaningPill
          variant="strip"
          icon="📎"
          label="已加入补充依据，后续会结合使用"
          title="补充材料会在需要时参与本课分析和设计"
          tone="evidence"
          onClear={onClearRefMaterial}
          clearLabel="移除"
        />
      )}

      <div
        style={{
          padding: '12px 18px',
          borderTop: `1px solid ${C.border}`,
          background: C.card,
          flexShrink: 0,
        }}
      >
        <div
          style={{
            display: 'flex',
            gap: '10px',
            alignItems: 'flex-end',
          }}
        >
          <div
            style={{
              position: 'relative',
              flexShrink: 0,
            }}
          >
            <button
              type="button"
              onClick={() =>
                setShowPlusMenu(
                  value => !value,
                )
              }
              disabled={inputDisabled}
              title="更多备课能力"
              style={{
                width: '38px',
                height: '38px',
                borderRadius: '50%',
                border: `1px solid ${C.border}`,
                background: showPlusMenu
                  ? C.primaryLight
                  : C.card,
                color: showPlusMenu
                  ? C.primary
                  : C.textSec,
                fontSize: '20px',
                cursor: inputDisabled
                  ? 'not-allowed'
                  : 'pointer',
                opacity: inputDisabled
                  ? 0.5
                  : 1,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              ＋
            </button>

            {showPlusMenu && (
              <>
                <div
                  onClick={() =>
                    setShowPlusMenu(false)
                  }
                  style={{
                    position: 'fixed',
                    inset: 0,
                    zIndex: 998,
                  }}
                />

                <div
                  style={{
                    position: 'absolute',
                    bottom: '46px',
                    left: 0,
                    zIndex: 999,
                    width: '260px',
                    padding: '6px',
                    borderRadius: '12px',
                    border: `1px solid ${C.border}`,
                    background: C.card,
                    boxShadow:
                      '0 8px 32px rgba(0,0,0,0.12)',
                  }}
                >
                  {PLUS_MENU_ITEMS.map(
                    item => {
                      const availability =
                        item.tool === 'attachment'
                          ? {
                              visible: true,
                              enabled: !inputDisabled,
                              reason: inputDisabled
                                ? '请等待当前操作完成'
                                : '',
                            }
                          : plusItemAvailability(
                              item.tool,
                            )

                      if (
                        availability.visible ===
                        false
                      ) {
                        return null
                      }

                      return (
                        <button
                          type="button"
                          key={item.tool}
                          onClick={() => {
                            if (
                              !availability.enabled
                            ) {
                              return
                            }

                            setShowPlusMenu(false)

                            if (item.tool === 'attachment') {
                              attachmentInputRef.current
                                ?.openFilePicker()
                              return
                            }

                            onOpenTool(item.tool)
                          }}
                          disabled={
                            !availability.enabled
                          }
                          title={
                            availability.enabled
                              ? item.desc
                              : availability.reason
                          }
                          style={{
                            width: '100%',
                            padding: '9px 10px',
                            display: 'flex',
                            alignItems: 'flex-start',
                            gap: '10px',
                            borderRadius: '8px',
                            border: 'none',
                            background: 'transparent',
                            cursor:
                              availability.enabled
                                ? 'pointer'
                                : 'not-allowed',
                            opacity:
                              availability.enabled
                                ? 1
                                : 0.45,
                            textAlign: 'left',
                          }}
                          onMouseEnter={event => {
                            if (
                              availability.enabled
                            ) {
                              event.currentTarget.style.background =
                                '#F3F4F6'
                            }
                          }}
                          onMouseLeave={event => {
                            event.currentTarget.style.background =
                              'transparent'
                          }}
                        >
                          <span
                            style={{
                              flexShrink: 0,
                              fontSize: '17px',
                            }}
                          >
                            {item.emoji}
                          </span>

                          <span
                            style={{
                              minWidth: 0,
                            }}
                          >
                            <span
                              style={{
                                display: 'block',
                                color: C.text,
                                fontSize: '13px',
                                fontWeight: 600,
                              }}
                            >
                              {item.label}
                            </span>

                            <span
                              style={{
                                display: 'block',
                                marginTop: '1px',
                                color: C.textMuted,
                                fontSize: '11px',
                                lineHeight: 1.4,
                              }}
                            >
                              {availability.enabled
                                ? item.desc
                                : availability.reason}
                            </span>
                          </span>
                        </button>
                      )
                    },
                  )}
                </div>
              </>
            )}
          </div>

          <div
            style={{
              flex: 1,
              minWidth: 0,
              padding: '9px 10px',
              display: 'flex',
              alignItems: 'flex-end',
              gap: '7px',
              borderRadius: '12px',
              border: `1px solid ${C.border}`,
              background: '#F9FAFB',
            }}
          >
            <textarea
              ref={taRef}
              value={inputText}
              onChange={event =>
                setInputText(
                  event.target.value,
                )
              }
              onKeyDown={event => {
                if (
                  handleDraftKeyDown(
                    event,
                  )
                ) {
                  return
                }

                if (
                  event.key === 'Enter' &&
                  !event.shiftKey
                ) {
                  event.preventDefault()
                  void doSend()
                }
              }}
              placeholder={placeholder}
              rows={2}
              disabled={inputDisabled}
              style={{
                flex: 1,
                minWidth: 0,
                border: 'none',
                outline: 'none',
                background: 'transparent',
                color: C.text,
                resize: 'none',
                opacity: inputDisabled
                  ? 0.5
                  : 1,
                fontFamily: 'inherit',
                fontSize: '15px',
                lineHeight: 1.6,
              }}
            />

            <VoiceInputButton
              status={voice.status}
              isSupported={
                voice.isSupported
              }
              elapsedSeconds={
                voice.elapsedSeconds
              }
              disabled={
                isBusy ||
                stageContinuationActive
              }
              error={voice.error}
              onStart={beginVoiceInput}
              onStop={voice.stop}
              onCancel={voice.cancel}
            />

            <button
              type="button"
              onClick={() => void doSend()}
              disabled={
                sendDisabled ||
                !inputText.trim()
              }
              style={{
                width: '36px',
                height: '36px',
                flexShrink: 0,
                borderRadius: '50%',
                border: 'none',
                background:
                  inputDisabled ||
                  !inputText.trim()
                    ? '#E5E7EB'
                    : C.primary,
                color: '#fff',
                cursor:
                  inputDisabled ||
                  !inputText.trim()
                    ? 'not-allowed'
                    : 'pointer',
                fontSize: '16px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              →
            </button>
          </div>
        </div>

        <div
          style={{
            marginTop: '5px',
            paddingLeft: '48px',
            color:
              voice.status === 'error'
                ? '#DC2626'
                : voice.isActive
                  ? C.primary
                  : C.textMuted,
            fontSize: '10px',
            lineHeight: 1.5,
          }}
        >
          {voiceStatusText ||
            '已自动保存草稿 · 可直接拖入PDF/Word/PPT/图片 · Shift+Enter换行'}
        </div>

        {hasContent && (
          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              marginTop: '8px',
            }}
          >
            <button
              type="button"
              onClick={onPublish}
              disabled={inputDisabled}
              style={{
                padding: '6px 16px',
                borderRadius: '16px',
                border: 'none',
                background: inputDisabled
                  ? '#E5E7EB'
                  : 'linear-gradient(135deg, #10B981, #34D399)',
                color: inputDisabled
                  ? C.textMuted
                  : '#fff',
                cursor: inputDisabled
                  ? 'not-allowed'
                  : 'pointer',
                fontSize: '12px',
                fontWeight: 600,
              }}
            >
              🚀 发布教案
            </button>
          </div>
        )}
      </div>
    </>
  )
})

export default ConversationInputBar
