/**
 * RebuildDiscussionPanel.tsx
 *
 * 课件全页重构“先讨论、后确认执行”面板。
 *
 * 业务原则：
 * 1. “与AI讨论”只澄清需求、形成方案，不生成HTML、不修改页面；
 * 2. 老师在消息中输入“开始”“执行”“确认”等自然语言不会触发重构；
 * 3. 只有点击独立的“确认并开始重构”按钮才会生成代码；
 * 4. 后端会校验讨论开始时的页面版本，禁止旧方案覆盖新页面；
 * 5. 确认执行后继续复用教育域授权、版本快照和HTML完整性校验；
 * 6. AI回复、共识摘要和最终执行方案统一使用安全Markdown渲染。
 */

import {
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'

import type {
  CSSProperties,
} from 'react'

import {
  cancelRebuildDiscussion,
  confirmRebuildDiscussion,
  loadRebuildDiscussion,
  sendRebuildDiscussionMessage,
} from '@/api/courseware-rebuild-discussion'

import type {
  CWRebuildDiscussion,
  CWRebuildDiscussionStatus,
} from '@/api/courseware-rebuild-discussion'

import DiscussionMarkdown from './DiscussionMarkdown'

import { useVoiceDraftInput } from '@/hooks/useVoiceDraftInput'
import VoiceInputButton from '@/components/voice/VoiceInputButton'

import {
  C,
} from './workshopConstants'

interface Props {
  coursewareId: string
  pageNum: number

  /**
   * 与RefinePanel共用的受保护输入草稿。
   */
  content: string
  onContentChange: (
    value: string,
  ) => void

  /**
   * 老师明确选择的模板、代码收藏和本课前页参考信息。
   *
   * 最终执行时后端仍会重新读取并校验资源权限。
   */
  referenceContext: string

  /**
   * 可选参考截图，只用于当前请求。
   */
  image?: string

  disabled?: boolean

  /**
   * 明确确认并成功重构后，把正式HTML同步给父页面。
   */
  onPageUpdated: (
    pageNumber: number,
    html: string,
  ) => void

  /**
   * 一轮讨论消息成功后的旁路回调。
   */
  onMessageSent?: () => void

  /**
   * 通知父组件当前是否存在活动讨论。
   */
  onActiveChange?: (
    active: boolean,
  ) => void
}

const ACTIVE_STATUSES:
  CWRebuildDiscussionStatus[] = [
    'discussing',
    'awaiting_confirmation',
    'executing',
  ]

const TERMINAL_STATUSES:
  CWRebuildDiscussionStatus[] = [
    'completed',
    'cancelled',
    'stale',
  ]

const STATUS_LABELS:
  Record<
    CWRebuildDiscussionStatus,
    string
  > = {
    discussing:
      '讨论中',
    awaiting_confirmation:
      '待老师确认',
    executing:
      '正在重构',
    completed:
      '已完成',
    cancelled:
      '已取消',
    stale:
      '方案已过期',
  }

const panelStyle:
  CSSProperties = {
    flex: 1,
    minWidth: 0,
    padding: 14,
    borderRadius: 10,
    border:
      '1px solid #FDBA74',
    background:
      '#FFFBEB',
  }

const baseButtonStyle:
  CSSProperties = {
    padding:
      '9px 14px',
    borderRadius: 8,
    fontSize: 12,
    fontWeight: 700,
    whiteSpace:
      'nowrap',
  }

/**
 * 将未知异常转换成教师可见文案。
 */
function getErrorMessage(
  error: unknown,
): string {
  if (
    error instanceof Error
    && error.message
  ) {
    return error.message
  }

  return '未知错误'
}

export default function RebuildDiscussionPanel({
  coursewareId,
  pageNum,
  content,
  onContentChange,
  referenceContext,
  image,
  disabled = false,
  onPageUpdated,
  onMessageSent,
  onActiveChange,
}: Props) {
  const [
    discussion,
    setDiscussion,
  ] = useState<
    CWRebuildDiscussion | null
  >(null)

  const [
    loading,
    setLoading,
  ] = useState(false)

  const [
    sending,
    setSending,
  ] = useState(false)

  const [
    confirming,
    setConfirming,
  ] = useState(false)

  const [
    cancelling,
    setCancelling,
  ] = useState(false)

  const [
    notice,
    setNotice,
  ] = useState('')

  const active =
    useMemo(
      () =>
        !!discussion
        && ACTIVE_STATUSES.includes(
          discussion.status,
        ),
      [
        discussion,
      ],
    )

  const terminal =
    useMemo(
      () =>
        !!discussion
        && TERMINAL_STATUSES.includes(
          discussion.status,
        ),
      [
        discussion,
      ],
    )

  /**
   * 同步活动讨论状态给父组件。
   */
  useEffect(
    () => {
      onActiveChange?.(
        active,
      )
    },
    [
      active,
      onActiveChange,
    ],
  )

  /**
   * 切换课件或页面后恢复该页尚未结束的讨论。
   *
   * load动作不调用AI，也不会修改页面。
   */
  useEffect(
    () => {
      let disposed = false

      async function load():
        Promise<void> {
        if (
          !coursewareId
          || pageNum <= 0
        ) {
          setDiscussion(
            null,
          )
          return
        }

        setLoading(
          true,
        )

        setNotice(
          '',
        )

        try {
          const value =
            await loadRebuildDiscussion(
              coursewareId,
              pageNum,
            )

          if (!disposed) {
            setDiscussion(
              value,
            )
          }
        } catch (error) {
          if (!disposed) {
            setNotice(
              '❌ 加载讨论失败：'
              + getErrorMessage(
                error,
              ),
            )
          }
        } finally {
          if (!disposed) {
            setLoading(
              false,
            )
          }
        }
      }

      setDiscussion(
        null,
      )

      void load()

      return () => {
        disposed = true
      }
    },
    [
      coursewareId,
      pageNum,
    ],
  )

  const busy =
    disabled
    || loading
    || sending
    || confirming
    || cancelling

  const inputRef =
    useRef<HTMLTextAreaElement>(
      null,
    )

  /**
   * 重构讨论语音复用父组件的受保护微调草稿。
   * final只写入讨论输入，不会触发确认或生成代码。
   */
  const voiceInput =
    useVoiceDraftInput({
      value: content,
      setValue:
        onContentChange,
      disabled:
        busy
        || terminal
        || discussion?.status
          === 'executing',
      maxDurationSeconds: 120,
      onFinalFocus: (
        finalValue,
      ) => {
        const element =
          inputRef.current

        if (!element) {
          return
        }

        element.focus()
        element.setSelectionRange(
          finalValue.length,
          finalValue.length,
        )
      },
      onError: (
        message,
      ) => {
        setNotice(
          '❌ 语音输入失败：'
          + message,
        )
      },
    })

  const canSend =
    !busy
    && !voiceInput.isActive
    && !terminal
    && discussion?.status
      !== 'executing'
    && content.trim()
      .length > 0

  /**
   * 发送一轮讨论消息。
   *
   * 成功后只更新讨论记录，不写课件HTML。
   */
  const handleSend =
    async (): Promise<void> => {
      if (!canSend) {
        return
      }

      setSending(
        true,
      )

      setNotice(
        '💬 AI正在分析并整理方案，本阶段不会修改页面……',
      )

      try {
        const result =
          await sendRebuildDiscussionMessage(
            coursewareId,
            pageNum,
            {
              discussionId:
                discussion?.id,
              content:
                content.trim(),
              referenceContext,
              image,
            },
          )

        setDiscussion(
          result.discussion,
        )

        /**
         * 后端成功保存老师消息并返回AI回复后，
         * 才清空当前输入。
         */
        onContentChange(
          '',
        )

        onMessageSent?.()

        setNotice(
          '✅ AI已回复，页面尚未修改',
        )
      } catch (error) {
        setNotice(
          '❌ 讨论失败：'
          + getErrorMessage(
            error,
          ),
        )
      } finally {
        setSending(
          false,
        )
      }
    }

  /**
   * 老师点击独立确认按钮后执行最终重构方案。
   *
   * 只有本动作会生成代码并写回课件页面。
   */
  const handleConfirm =
    async (): Promise<void> => {
      if (
        !discussion
          ?.ready_for_confirmation
        || busy
      ) {
        return
      }

      const accepted =
        window.confirm(
          '确认按最终执行方案重构当前页面吗？\n\n'
          + '确认后系统才会生成代码并覆盖页面；'
          + '覆盖前会自动保存当前历史版本。',
        )

      if (!accepted) {
        return
      }

      setConfirming(
        true,
      )

      setNotice(
        '🧱 已收到明确确认，正在生成并校验页面……',
      )

      try {
        const result =
          await confirmRebuildDiscussion(
            coursewareId,
            pageNum,
            discussion.id,
          )

        setDiscussion(
          result.discussion,
        )

        if (
          result.html_content
          && result.page_number > 0
        ) {
          onPageUpdated(
            result.page_number,
            result.html_content,
          )
        }

        setNotice(
          '✅ '
          + result.message,
        )
      } catch (error) {
        setNotice(
          '❌ 重构失败：'
          + getErrorMessage(
            error,
          ),
        )
      } finally {
        setConfirming(
          false,
        )
      }
    }

  /**
   * 取消尚未执行的讨论。
   *
   * cancel只修改讨论状态，不修改课件页面。
   */
  const handleCancel =
    async (): Promise<void> => {
      if (
        !discussion
        || !active
        || discussion.executing
        || busy
      ) {
        return
      }

      const accepted =
        window.confirm(
          '确定取消本次讨论吗？\n\n'
          + '取消后课件页面不会发生任何修改。',
        )

      if (!accepted) {
        return
      }

      setCancelling(
        true,
      )

      try {
        const result =
          await cancelRebuildDiscussion(
            coursewareId,
            pageNum,
            discussion.id,
          )

        setDiscussion(
          result.discussion,
        )

        setNotice(
          '✅ 讨论已取消，页面未发生修改',
        )
      } catch (error) {
        setNotice(
          '❌ 取消失败：'
          + getErrorMessage(
            error,
          ),
        )
      } finally {
        setCancelling(
          false,
        )
      }
    }

  /**
   * 清理当前界面的终态记录。
   *
   * 下次发送消息时，后端会创建新的活动讨论。
   */
  const handleStartNew =
    (): void => {
      setDiscussion(
        null,
      )

      setNotice(
        '可以输入新的修改想法，重新开始讨论。',
      )
    }

  return (
    <div style={panelStyle}>
      <div
        style={{
          display:
            'flex',
          alignItems:
            'center',
          gap: 8,
          flexWrap:
            'wrap',
        }}
      >
        <strong
          style={{
            color:
              '#9A3412',
          }}
        >
          💬 先讨论，确认后再重构
        </strong>

        {discussion && (
          <span
            style={{
              padding:
                '3px 8px',
              borderRadius:
                999,
              background:
                '#FFF7ED',
              color:
                '#C2410C',
              fontSize:
                11,
              fontWeight:
                700,
            }}
          >
            {
              STATUS_LABELS[
                discussion.status
              ]
            }
          </span>
        )}

        <span
          style={{
            color:
              '#92400E',
            fontSize:
              11,
            lineHeight:
              1.5,
          }}
        >
          对话中的“开始”不会执行，必须点击独立确认按钮
        </span>
      </div>

      {loading && (
        <div
          style={{
            padding:
              '14px 0',
            color:
              '#6B7280',
            fontSize:
              13,
          }}
        >
          ⏳ 正在恢复讨论记录……
        </div>
      )}

      {!loading
        && discussion
        && discussion.messages
          .length > 0 && (
          <div
            style={{
              maxHeight:
                380,
              overflowY:
                'auto',
              marginTop:
                10,
              padding:
                10,
              border:
                `1px solid ${C.border}`,
              borderRadius:
                8,
              background:
                '#fff',
            }}
          >
            {discussion.messages.map(
              (
                discussionMessage,
                index,
              ) => {
                const teacher =
                  discussionMessage.role
                    === 'teacher'

                return (
                  <div
                    key={
                      `${discussionMessage.created_at}-${index}`
                    }
                    style={{
                      display:
                        'flex',
                      justifyContent:
                        teacher
                          ? 'flex-end'
                          : 'flex-start',
                      marginBottom:
                        9,
                    }}
                  >
                    <div
                      style={{
                        maxWidth:
                          '90%',
                        padding:
                          '9px 11px',
                        borderRadius:
                          8,
                        background:
                          teacher
                            ? '#EDE9FE'
                            : '#F3F4F6',
                        color:
                          teacher
                            ? '#3B0764'
                            : '#1F2937',
                      }}
                    >
                      <div
                        style={{
                          marginBottom:
                            5,
                          color:
                            teacher
                              ? '#6D28D9'
                              : '#374151',
                          fontSize:
                            11,
                          fontWeight:
                            700,
                        }}
                      >
                        {
                          teacher
                            ? '老师'
                            : 'AI讨论顾问'
                        }
                      </div>

                      <DiscussionMarkdown
                        content={
                          discussionMessage.content
                        }
                      />
                    </div>
                  </div>
                )
              },
            )}
          </div>
        )}

      {discussion
        ?.ai_summary && (
        <div
          style={{
            marginTop:
              10,
            padding:
              '9px 11px',
            borderRadius:
              8,
            border:
              '1px solid #BFDBFE',
            background:
              '#EFF6FF',
            color:
              '#1E3A8A',
          }}
        >
          <strong
            style={{
              display:
                'block',
              marginBottom:
                5,
              color:
                '#1D4ED8',
              fontSize:
                12,
            }}
          >
            当前共识摘要
          </strong>

          <DiscussionMarkdown
            content={
              discussion.ai_summary
            }
            compact
          />
        </div>
      )}

      {discussion
        ?.final_instruction && (
        <div
          style={{
            marginTop:
              10,
            padding:
              '9px 11px',
            borderRadius:
              8,
            border:
              '1px solid #FDBA74',
            background:
              '#FFF7ED',
            color:
              '#7C2D12',
          }}
        >
          <strong
            style={{
              display:
                'block',
              marginBottom:
                5,
              color:
                '#C2410C',
              fontSize:
                12,
            }}
          >
            最终执行方案（确认后才会编程）
          </strong>

          <DiscussionMarkdown
            content={
              discussion
                .final_instruction
            }
            compact
          />
        </div>
      )}

      {discussion
        ?.error_message && (
        <div
          style={{
            marginTop:
              10,
            padding:
              '8px 10px',
            borderRadius:
              8,
            background:
              '#FEF2F2',
            color:
              '#B91C1C',
            fontSize:
              12,
            lineHeight:
              1.55,
          }}
        >
          ⚠️ {
            discussion.error_message
          }
        </div>
      )}

      {!terminal
        && discussion?.status
          !== 'executing' && (
        <>
        <div
          style={{
            display:
              'flex',
            gap: 8,
            alignItems:
              'flex-start',
            marginTop:
              10,
            flexWrap:
              'wrap',
          }}
        >
          <textarea
            ref={
              inputRef
            }
            value={
              content
            }
            onChange={
              (
                event,
              ) => {
                onContentChange(
                  event.target.value,
                )
              }
            }
            rows={3}
            disabled={
              busy
              || voiceInput.isActive
            }
            placeholder={
              discussion
                ? '继续补充教学重点、布局、互动方式，或回答AI的问题……'
                : '说明希望如何重构本页，AI会先和你讨论，不会立即生成代码……'
            }
            style={{
              flex:
                '1 1 360px',
              minWidth:
                220,
              resize:
                'vertical',
              padding:
                '9px 11px',
              borderRadius:
                8,
              border:
                `1px solid ${C.border}`,
              outline:
                'none',
              fontFamily:
                'inherit',
              fontSize:
                13,
              lineHeight:
                1.6,
              boxSizing:
                'border-box',
            }}
          />

          <VoiceInputButton
            status={
              voiceInput.status
            }
            isSupported={
              voiceInput.isSupported
            }
            elapsedSeconds={
              voiceInput.elapsedSeconds
            }
            disabled={
              /**
               * 外层渲染条件已经排除了终态和executing状态。
               * 此处只需反映当前网络或业务忙碌状态，
               * 避免TypeScript把重复状态比较判定为不可达分支。
               */
              busy
            }
            error={
              voiceInput.error
            }
            onStart={
              voiceInput.begin
            }
            onStop={
              voiceInput.stop
            }
            onCancel={
              voiceInput.cancel
            }
          />

          <button
            type="button"
            onClick={
              () => {
                void handleSend()
              }
            }
            disabled={
              !canSend
            }
            style={{
              ...baseButtonStyle,
              border:
                'none',
              background:
                canSend
                  ? '#7C3AED'
                  : '#E5E7EB',
              color:
                canSend
                  ? '#fff'
                  : '#9CA3AF',
              cursor:
                canSend
                  ? 'pointer'
                  : 'default',
            }}
          >
            {
              sending
                ? '讨论中……'
                : discussion
                  ? '继续讨论'
                  : '与AI讨论'
            }
          </button>
        </div>

        <div
          style={{
            marginTop: 5,
            color:
              voiceInput.status ===
              'error'
                ? '#B91C1C'
                : voiceInput.isActive
                  ? '#7C3AED'
                  : '#92400E',
            fontSize: 10,
            lineHeight: 1.5,
          }}
        >
          {
            voiceInput.statusText
            || '点击麦克风可语音输入；识别文字不会自动发送或执行重构'
          }
        </div>
        </>
      )}

      {discussion?.status
        === 'executing' && (
        <div
          style={{
            marginTop:
              10,
            padding:
              '10px 12px',
            borderRadius:
              8,
            background:
              '#FFF7ED',
            color:
              '#C2410C',
            fontSize:
              13,
            fontWeight:
              700,
          }}
        >
          ⏳ 已确认方案，正在执行全页重构，请勿重复提交。
        </div>
      )}

      <div
        style={{
          display:
            'flex',
          gap: 8,
          flexWrap:
            'wrap',
          marginTop:
            10,
        }}
      >
        {discussion
          ?.ready_for_confirmation && (
          <button
            type="button"
            onClick={
              () => {
                void handleConfirm()
              }
            }
            disabled={
              busy
            }
            style={{
              ...baseButtonStyle,
              border:
                'none',
              background:
                busy
                  ? '#E5E7EB'
                  : '#EA580C',
              color:
                busy
                  ? '#9CA3AF'
                  : '#fff',
              cursor:
                busy
                  ? 'default'
                  : 'pointer',
            }}
          >
            {
              confirming
                ? '正在重构……'
                : '确认并开始重构'
            }
          </button>
        )}

        {active
          && !discussion
            ?.executing && (
          <button
            type="button"
            onClick={
              () => {
                void handleCancel()
              }
            }
            disabled={
              busy
            }
            style={{
              ...baseButtonStyle,
              border:
                '1px solid #DC2626',
              background:
                '#fff',
              color:
                '#DC2626',
              cursor:
                busy
                  ? 'default'
                  : 'pointer',
            }}
          >
            {
              cancelling
                ? '取消中……'
                : '取消本次讨论'
            }
          </button>
        )}

        {terminal && (
          <button
            type="button"
            onClick={
              handleStartNew
            }
            disabled={
              busy
            }
            style={{
              ...baseButtonStyle,
              border:
                '1px solid #7C3AED',
              background:
                '#fff',
              color:
                '#7C3AED',
              cursor:
                busy
                  ? 'default'
                  : 'pointer',
            }}
          >
            开始新的讨论
          </button>
        )}
      </div>

      {notice && (
        <div
          style={{
            marginTop:
              10,
            padding:
              '8px 10px',
            borderRadius:
              7,
            background:
              notice.startsWith(
                '❌',
              )
                ? '#FEF2F2'
                : notice.startsWith(
                    '✅',
                  )
                  ? '#F0FDF4'
                  : '#EFF6FF',
            color:
              notice.startsWith(
                '❌',
              )
                ? '#B91C1C'
                : notice.startsWith(
                    '✅',
                  )
                  ? '#166534'
                  : '#1D4ED8',
            fontSize:
              12,
            lineHeight:
              1.55,
          }}
        >
          {notice}
        </div>
      )}
    </div>
  )
}
