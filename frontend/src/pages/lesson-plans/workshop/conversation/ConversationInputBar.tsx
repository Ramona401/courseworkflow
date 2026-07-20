/**
 * ConversationInputBar.tsx — 对话模式底部输入区
 *
 * 从 ConversationModePage.tsx 抽出的输入区整体，自上而下三块：
 * 1. 已选组件提示条；
 * 2. 「+」能力菜单、文本输入框和发送按钮；
 * 3. 教案发布入口。
 *
 * 草稿保护：
 * 1. 输入文字按当前用户、当前教案和message字段隔离；
 * 2. 输入变化立即写入sessionStorage；
 * 3. 刷新或切换页面后返回时自动恢复；
 * 4. 支持持久化撤销和重做；
 * 5. Ctrl/Command+Z恢复误删；
 * 6. Ctrl/Command+Shift+Z或Ctrl+Y重做；
 * 7. 发送后清空输入框，但保留可撤销快照；
 * 8. 只缓存文字，不缓存参考资料正文或文件。
 *
 * 状态归属纪律：
 * 1. 输入文字由useProtectedDraft统一管理；
 * 2. showPlusMenu仍为本组件临时UI状态；
 * 3. 其余状态通过props与页面交互；
 * 4. 本组件不直接调用任何业务API。
 */

import {
  forwardRef,
  useImperativeHandle,
  useRef,
  useState,
} from 'react'
import { useAuth } from '@/store/auth'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import { C } from '../components/workshopConstants'
import { PLUS_MENU_ITEMS } from './conversationScript'

/**
 * 对外暴露的命令句柄。
 */
export interface ConversationInputBarHandle {
  /** 仅聚焦输入框。 */
  focus: () => void
  /** 预填文字到输入框并聚焦。 */
  prefill: (text: string) => void
}

/**
 * 输入区组件Props。
 */
export interface ConversationInputBarProps {
  /** AI忙碌中。 */
  isBusy: boolean
  /** 输入框占位文案。 */
  placeholder: string
  /** 已选组件数量。 */
  selectedCount: number
  /** 清空已选组件。 */
  onClearSelected: () => void
  /**
   * 发送一条文字消息。
   *
   * 返回false表示业务明确拒绝本次发送，
   * 输入框将继续保留原文。
   *
   * 现有调用方返回void或Promise<void>时，
   * 视为请求已经被调用方受理。
   */
  onSend: (
    text: string,
  ) =>
    | void
    | boolean
    | Promise<void | boolean>
  /** 教案正文是否非空。 */
  hasContent: boolean
  /** 发布教案。 */
  onPublish: () => void
  /** 「+」菜单各项可用性。 */
  plusItemAvailability: (
    tool: string,
  ) => {
    /**
     * false时菜单项完全不进入DOM。
     *
     * 用于教育域专属能力，不能仅以disabled状态泄露入口。
     * 未提供时默认可见，兼容其它普通能力。
     */
    visible?: boolean
    enabled: boolean
    reason: string
  }
  /** 唤起指定能力。 */
  onOpenTool: (tool: string) => void
  /** 参考资料附件文件名。 */
  refMaterialName?: string
  /** 移除参考资料附件。 */
  onClearRefMaterial?: () => void
}

/**
 * 安全读取当前正在备课的教案ID。
 *
 * ConversationInputBar只在chatting视图挂载；
 * 父页面在进入chatting前已经写入workshop_active_plan_id。
 */
function getActiveConversationPlanID(): string {
  try {
    return (
      sessionStorage.getItem(
        'workshop_active_plan_id',
      ) || 'current-plan'
    )
  } catch {
    return 'current-plan'
  }
}

/**
 * 对话模式底部输入区。
 */
const ConversationInputBar = forwardRef<
  ConversationInputBarHandle,
  ConversationInputBarProps
>(
  function ConversationInputBar(
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

    /**
     * 组件挂载时读取当前教案ID。
     *
     * 草稿键最终为：
     * 当前用户 + lesson-plan-conversation +
     * 当前教案ID + message。
     */
    const activePlanID =
      getActiveConversationPlanID()

    const {
      value: inputText,
      setValue: setInputText,
      commit: commitInputDraft,
      handleKeyDown:
        handleDraftKeyDown,
    } = useProtectedDraft({
      userId: user?.id,
      scope:
        'lesson-plan-conversation',
      resourceId: activePlanID,
      field: 'message',
      initialValue: '',
      maxHistory: 40,
    })

    /** 内部真正的textarea DOM引用。 */
    const taRef =
      useRef<HTMLTextAreaElement>(null)

    /**
     * 对外暴露命令句柄。
     */
    useImperativeHandle(
      ref,
      () => ({
        focus: () =>
          taRef.current?.focus(),

        prefill: (text: string) => {
          setInputText(text)

          requestAnimationFrame(() => {
            const element =
              taRef.current

            if (!element) return

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

    /** 「+」能力菜单展开态。 */
    const [
      showPlusMenu,
      setShowPlusMenu,
    ] = useState(false)

    /**
     * 执行发送。
     *
     * 不再在调用onSend之前直接销毁草稿：
     * 1. 先等待调用方受理；
     * 2. 返回false或抛错时保留原文；
     * 3. 受理后使用commit清空显示值；
     * 4. commit保留撤销快照，Ctrl+Z仍可找回。
     */
    const doSend = async () => {
      if (
        !inputText.trim() ||
        isBusy
      ) {
        return
      }

      const text = inputText.trim()

      try {
        const accepted =
          await Promise.resolve(
            onSend(text),
          )

        if (accepted === false) {
          return
        }

        commitInputDraft()
      } catch (error) {
        /**
         * 调用方抛错时保持当前草稿不变。
         * 业务错误展示由页面层负责。
         */
        console.error(
          '备课消息发送未被受理，草稿已保留:',
          error,
        )
      }
    }

    /**
     * 菜单项点击。
     */
    const handleMenuItem = (
      tool: string,
    ) => {
      setShowPlusMenu(false)
      onOpenTool(tool)
    }

    return (
      <>
        {selectedCount > 0 && (
          <div
            style={{
              padding: '7px 18px',
              background: C.primaryLight,
              borderTop:
                `1px solid ${C.border}`,
              fontSize: '12px',
              color: C.primary,
              display: 'flex',
              alignItems: 'center',
              justifyContent:
                'space-between',
              flexShrink: 0,
            }}
          >
            <span>
              🧩 已选 {selectedCount}{' '}
              个教学组件，下一条消息发出时
              AI 会一并参考
            </span>

            <button
              onClick={onClearSelected}
              style={{
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: C.textMuted,
                fontSize: '12px',
              }}
            >
              清空
            </button>
          </div>
        )}

        {refMaterialName && (
          <div
            style={{
              padding: '7px 18px',
              background:
                'rgba(129,140,248,0.10)',
              borderTop:
                `1px solid ${C.border}`,
              fontSize: '12px',
              color: '#6366F1',
              display: 'flex',
              alignItems: 'center',
              justifyContent:
                'space-between',
              flexShrink: 0,
            }}
          >
            <span
              style={{
                overflow: 'hidden',
                textOverflow:
                  'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              📎 已附参考资料「
              {refMaterialName}
              」，AI 每轮回复都会参考
            </span>

            {onClearRefMaterial && (
              <button
                onClick={
                  onClearRefMaterial
                }
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  color: C.textMuted,
                  fontSize: '12px',
                  flexShrink: 0,
                  marginLeft: '10px',
                }}
              >
                移除
              </button>
            )}
          </div>
        )}

        <div
          style={{
            padding: '12px 18px',
            borderTop:
              `1px solid ${C.border}`,
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
                onClick={() =>
                  setShowPlusMenu(
                    (value) => !value,
                  )
                }
                disabled={isBusy}
                title="更多备课能力"
                style={{
                  width: '38px',
                  height: '38px',
                  borderRadius: '50%',
                  border:
                    `1px solid ${C.border}`,
                  background:
                    showPlusMenu
                      ? C.primaryLight
                      : C.card,
                  color:
                    showPlusMenu
                      ? C.primary
                      : C.textSec,
                  fontSize: '20px',
                  cursor:
                    isBusy
                      ? 'not-allowed'
                      : 'pointer',
                  opacity:
                    isBusy ? 0.5 : 1,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent:
                    'center',
                  transition:
                    'all 150ms ease',
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
                      top: 0,
                      left: 0,
                      right: 0,
                      bottom: 0,
                      zIndex: 998,
                    }}
                  />

                  <div
                    style={{
                      position: 'absolute',
                      bottom: '46px',
                      left: 0,
                      width: '260px',
                      background: C.card,
                      borderRadius: '12px',
                      border:
                        `1px solid ${C.border}`,
                      boxShadow:
                        '0 8px 32px rgba(0,0,0,0.12)',
                      padding: '6px',
                      zIndex: 999,
                    }}
                  >
                    {PLUS_MENU_ITEMS.map(
                      (item) => {
                        const availability =
                          plusItemAvailability(
                            item.tool,
                          )

                        /**
                         * 教育域专属能力在不适用时完全隐藏。
                         *
                         * 返回null意味着不会生成按钮、说明文字或禁用入口，
                         * 但后端仍独立执行权限校验，缓存代码也不能绕过。
                         */
                        if (
                          availability.visible ===
                          false
                        ) {
                          return null
                        }

                        return (
                          <button
                            key={item.tool}
                            onClick={() => {
                              if (
                                availability.enabled
                              ) {
                                handleMenuItem(
                                  item.tool,
                                )
                              }
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
                              display: 'flex',
                              alignItems:
                                'flex-start',
                              gap: '10px',
                              width: '100%',
                              padding:
                                '9px 10px',
                              borderRadius:
                                '8px',
                              border: 'none',
                              background:
                                'transparent',
                              cursor:
                                availability.enabled
                                  ? 'pointer'
                                  : 'not-allowed',
                              opacity:
                                availability.enabled
                                  ? 1
                                  : 0.45,
                              textAlign: 'left',
                              transition:
                                'background 150ms ease',
                            }}
                            onMouseEnter={(
                              event,
                            ) => {
                              if (
                                availability.enabled
                              ) {
                                event.currentTarget.style.background =
                                  '#F3F4F6'
                              }
                            }}
                            onMouseLeave={(
                              event,
                            ) => {
                              event.currentTarget.style.background =
                                'transparent'
                            }}
                          >
                            <span
                              style={{
                                fontSize:
                                  '17px',
                                flexShrink: 0,
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
                                  display:
                                    'block',
                                  fontSize:
                                    '13px',
                                  fontWeight:
                                    600,
                                  color: C.text,
                                }}
                              >
                                {item.label}
                              </span>

                              <span
                                style={{
                                  display:
                                    'block',
                                  fontSize:
                                    '11px',
                                  color:
                                    C.textMuted,
                                  marginTop:
                                    '1px',
                                  lineHeight:
                                    1.4,
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
                display: 'flex',
                gap: '10px',
                alignItems: 'flex-end',
                background: '#F9FAFB',
                borderRadius: '12px',
                border:
                  `1px solid ${C.border}`,
                padding: '9px 12px',
              }}
            >
              <textarea
                ref={taRef}
                value={inputText}
                onChange={(event) =>
                  setInputText(
                    event.target.value,
                  )
                }
                onKeyDown={(event) => {
                  if (
                    handleDraftKeyDown(
                      event,
                    )
                  ) {
                    return
                  }

                  if (
                    event.key ===
                      'Enter' &&
                    !event.shiftKey
                  ) {
                    event.preventDefault()
                    void doSend()
                  }
                }}
                placeholder={placeholder}
                rows={2}
                disabled={isBusy}
                style={{
                  flex: 1,
                  background:
                    'transparent',
                  border: 'none',
                  outline: 'none',
                  fontSize: '15px',
                  color: C.text,
                  resize: 'none',
                  fontFamily: 'inherit',
                  lineHeight: 1.6,
                  opacity:
                    isBusy ? 0.5 : 1,
                }}
              />

              <button
                onClick={() =>
                  void doSend()
                }
                disabled={
                  isBusy ||
                  !inputText.trim()
                }
                style={{
                  width: '36px',
                  height: '36px',
                  flexShrink: 0,
                  borderRadius: '50%',
                  border: 'none',
                  background:
                    isBusy ||
                    !inputText.trim()
                      ? '#E5E7EB'
                      : C.primary,
                  color: '#fff',
                  cursor:
                    isBusy ||
                    !inputText.trim()
                      ? 'not-allowed'
                      : 'pointer',
                  fontSize: '16px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent:
                    'center',
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
              fontSize: '10px',
              color: C.textMuted,
              lineHeight: 1.5,
            }}
          >
            已自动保存草稿 ·
            Ctrl/Command+Z恢复误删 ·
            Shift+Enter换行
          </div>

          {hasContent && (
            <div
              style={{
                display: 'flex',
                justifyContent:
                  'flex-end',
                marginTop: '8px',
              }}
            >
              <button
                onClick={onPublish}
                disabled={isBusy}
                style={{
                  padding:
                    '6px 16px',
                  borderRadius:
                    '16px',
                  border: 'none',
                  background:
                    isBusy
                      ? '#E5E7EB'
                      : 'linear-gradient(135deg, #10B981, #34D399)',
                  color:
                    isBusy
                      ? C.textMuted
                      : '#fff',
                  fontSize: '12px',
                  fontWeight: 600,
                  cursor:
                    isBusy
                      ? 'not-allowed'
                      : 'pointer',
                }}
              >
                🚀 发布教案
              </button>
            </div>
          )}
        </div>
      </>
    )
  },
)

export default ConversationInputBar
