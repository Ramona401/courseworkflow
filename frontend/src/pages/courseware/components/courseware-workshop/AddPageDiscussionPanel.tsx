/**
 * AddPageDiscussionPanel.tsx — 新增页AI多轮讨论与明确生成面板。
 *
 * 讨论阶段只返回老师可见回复和结构化方案，不创建页面。
 * 插入位置变化后必须重新讨论一次，只有独立按钮会执行建页和HTML生成。
 */
import { useRef, useState } from 'react'
import type {
  CSSProperties,
  ChangeEvent,
  KeyboardEvent,
} from 'react'
import { regenerateCWPage } from '@/api/coursewares'
import {
  addCWPageAtPosition,
  discussCoursewareAddPage,
} from '@/api/courseware-add-page-discussion'
import type {
  CoursewareAddPageDiscussionMessage,
} from '@/api/courseware-add-page-discussion'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import { useVoiceDraftInput } from '@/hooks/useVoiceDraftInput'
import VoiceInputButton from '@/components/voice/VoiceInputButton'
import { useAuth } from '@/store/auth'
import DiscussionMarkdown from './DiscussionMarkdown'
import { C } from './workshopConstants'
import {
  addPagePlanField,
  addPagePlanIsReady,
  createAddPageDiscussionDraft,
  parseAddPageDiscussionDraft,
  parseAddPagePlan,
} from './addPageDiscussionDraft'

interface Props {
  coursewareId: string
  insertAt: number
  onDone: (newPageNumber: number) => void
  onBusyChange: (busy: boolean) => void
}

type DiscussionPhase = 'idle' | 'discussing' | 'creating' | 'generating' | 'done'

const cardStyle: CSSProperties = {
  border: '1px solid #E5E7EB',
  borderRadius: 12,
  background: '#fff',
}

const starterMessages = [
  '增加一页课堂检测题',
  '增加一页实验步骤说明',
  '增加一页前后概念对比',
]

export default function AddPageDiscussionPanel({
  coursewareId,
  insertAt,
  onDone,
  onBusyChange,
}: Props) {
  const { user } = useAuth()
  const initialDraft = createAddPageDiscussionDraft()
  const protectedDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'courseware-add-page-discussion',
    resourceId: coursewareId,
    field: 'conversation',
    initialValue: JSON.stringify(initialDraft),
    maxHistory: 12,
    coalesceMs: 900,
  })
  const draft = parseAddPageDiscussionDraft(protectedDraft.value)
  const [phase, setPhase] = useState<DiscussionPhase>('idle')
  const [errorMessage, setErrorMessage] = useState('')
  const [createdPageNumber, setCreatedPageNumber] = useState(0)

  const updateDraft = (
    patch: Partial<ReturnType<typeof createAddPageDiscussionDraft>>,
  ) => {
    protectedDraft.setValue(previous => JSON.stringify({
      ...parseAddPageDiscussionDraft(previous),
      ...patch,
    }))
  }

  const setBusy = (busy: boolean) => onBusyChange(busy)
  const isBusy = phase === 'discussing'
    || phase === 'creating'
    || phase === 'generating'
  const insertPositionChanged = draft.lastDiscussedInsertAt > 0
    && draft.lastDiscussedInsertAt !== insertAt
  const inputRef = useRef<HTMLTextAreaElement>(null)

  /**
   * 新增页讨论语音写入同一份受保护会话草稿。
   * 识别完成后仅回填输入框，不会自动发消息或生成页面。
   */
  const voiceInput = useVoiceDraftInput({
    value: draft.input,
    setValue: (value) => {
      updateDraft({ input: value })
    },
    disabled: isBusy,
    maxDurationSeconds: 120,
    onFinalFocus: (finalValue) => {
      const element = inputRef.current
      if (!element) return

      element.focus()
      element.setSelectionRange(
        finalValue.length,
        finalValue.length,
      )
    },
    onError: (message) => {
      setErrorMessage(`语音输入失败：${message}`)
    },
  })

  const canGenerate = draft.readyForConfirmation
    && !insertPositionChanged
    && addPagePlanIsReady(draft.plan)
    && !isBusy
    && !voiceInput.isActive

  const sendMessage = async () => {
    const message = draft.input.trim()
    if (!message) {
      setErrorMessage('请先输入本轮要和AI讨论的内容')
      return
    }

    if (voiceInput.isActive) {
      setErrorMessage('请先停止语音输入，再发送本轮讨论')
      return
    }

    setPhase('discussing')
    setBusy(true)
    setErrorMessage('')

    try {
      const result = await discussCoursewareAddPage(coursewareId, {
        message,
        messages: draft.messages,
        insert_at: insertAt,
        current_plan: draft.plan,
      })

      const nextMessages: CoursewareAddPageDiscussionMessage[] = [
        ...draft.messages,
        { role: 'teacher' as const, content: message },
        { role: 'assistant' as const, content: result.reply },
      ].slice(-24)

      updateDraft({
        input: '',
        messages: nextMessages,
        summary: result.summary,
        readyForConfirmation: result.ready_for_confirmation,
        lastDiscussedInsertAt: result.insert_at,
        plan: parseAddPagePlan(result.plan),
      })
      setPhase('idle')
    } catch (error: unknown) {
      setPhase('idle')
      setErrorMessage(
        error instanceof Error ? error.message : 'AI讨论失败，请稍后重试',
      )
    } finally {
      setBusy(false)
    }
  }

  const generateAndInsert = async () => {
    if (!canGenerate) {
      setErrorMessage(insertPositionChanged
        ? '插入位置已经变化，请再发送一轮消息，让AI结合新的前后页重新确认方案'
        : '页面方案尚未完整，请继续讨论标题、教学目的和核心内容')
      return
    }

    if (!window.confirm(`确认按当前方案创建页面，并插入为新的第 ${insertAt} 页吗？`)) {
      return
    }

    setPhase('creating')
    setBusy(true)
    setErrorMessage('')

    try {
      const page = await addCWPageAtPosition(coursewareId, {
        insert_at: insertAt,
        title: draft.plan.title.trim(),
        purpose: draft.plan.purpose.trim(),
        content_summary: draft.plan.content_summary.trim(),
        interaction_type: draft.plan.interaction_type.trim() || 'static',
        visual_format: draft.plan.visual_format.trim() || 'text_heavy',
        media_requirements: draft.plan.media_requirements.trim() || undefined,
        estimated_complexity: draft.plan.estimated_complexity,
      })
      const pageNumber = page.page_number
      setCreatedPageNumber(pageNumber)
      setPhase('generating')

      try {
        await regenerateCWPage(coursewareId, pageNumber)
        protectedDraft.clear()
        setPhase('done')
        setBusy(false)
        window.setTimeout(() => onDone(pageNumber), 700)
      } catch (error: unknown) {
        protectedDraft.clear()
        setPhase('done')
        setBusy(false)
        setErrorMessage(
          '页面已创建，但HTML生成失败：'
          + (error instanceof Error ? error.message : '未知错误')
          + '。进入页面后可使用“重新生成”继续完成。',
        )
        window.setTimeout(() => onDone(pageNumber), 2200)
      }
    } catch (error: unknown) {
      setPhase('idle')
      setBusy(false)
      setErrorMessage(error instanceof Error ? error.message : '创建页面失败')
    }
  }

  if (phase === 'creating' || phase === 'generating' || phase === 'done') {
    const icon = phase === 'creating' ? '📝' : phase === 'generating' ? '⚙️' : '✅'
    const title = phase === 'creating'
      ? `正在第 ${insertAt} 页位置创建页面…`
      : phase === 'generating'
        ? `P${createdPageNumber} 已创建，正在生成HTML…`
        : `P${createdPageNumber} 已完成`

    return (
      <div style={{
        minHeight: 430,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        textAlign: 'center',
        padding: 24,
      }}>
        <div style={{ fontSize: 38, marginBottom: 14 }}>{icon}</div>
        <div style={{
          fontSize: 16,
          fontWeight: 700,
          color: phase === 'done' ? '#059669' : '#374151',
        }}>
          {title}
        </div>
        {phase === 'generating' && (
          <div style={{ marginTop: 8, color: '#9CA3AF', fontSize: 13 }}>
            单页生成通常需要15至40秒
          </div>
        )}
        {errorMessage && (
          <div style={{
            maxWidth: 620,
            marginTop: 14,
            padding: '10px 13px',
            borderRadius: 9,
            background: '#FFFBEB',
            color: '#B45309',
            fontSize: 13,
            lineHeight: 1.65,
          }}>
            {errorMessage}
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="add-page-discussion-grid" style={{
      display: 'grid',
      gridTemplateColumns: 'minmax(0, 1.45fr) minmax(280px, 0.8fr)',
      gap: 18,
      minHeight: 500,
    }}>
      <section style={{
        ...cardStyle,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        background: '#F9FAFB',
        overflow: 'hidden',
      }}>
        <div style={{
          padding: '12px 14px',
          borderBottom: '1px solid #E5E7EB',
          background: '#fff',
        }}>
          <div style={{ color: '#1F2937', fontSize: 14, fontWeight: 800 }}>
            和AI把新增页讨论清楚
          </div>
          <div style={{ marginTop: 3, color: '#6B7280', fontSize: 12 }}>
            当前计划插入为第 {insertAt} 页。讨论不会创建页面。
          </div>
        </div>

        <div style={{
          flex: 1,
          minHeight: 300,
          maxHeight: 380,
          overflowY: 'auto',
          padding: 14,
        }}>
          {draft.messages.length === 0 ? (
            <div style={{
              height: '100%',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              textAlign: 'center',
              color: '#6B7280',
              padding: 24,
            }}>
              <div style={{ fontSize: 34, marginBottom: 9 }}>💬</div>
              <div style={{ fontWeight: 700, color: '#374151' }}>
                先描述你希望这一页解决什么教学问题
              </div>
              <div style={{
                marginTop: 5,
                maxWidth: 480,
                fontSize: 12,
                lineHeight: 1.7,
              }}>
                AI会结合插入点前后页面追问关键细节，并持续整理成可执行页面方案。
              </div>
              <div style={{
                marginTop: 14,
                display: 'flex',
                flexWrap: 'wrap',
                justifyContent: 'center',
                gap: 7,
              }}>
                {starterMessages.map(suggestion => (
                  <button
                    key={suggestion}
                    type="button"
                    onClick={() => updateDraft({ input: suggestion })}
                    disabled={voiceInput.isActive}
                    style={{
                      border: '1px solid #DDD6FE',
                      borderRadius: 999,
                      padding: '6px 10px',
                      background: '#F5F3FF',
                      color: '#6D28D9',
                      fontSize: 12,
                      cursor: 'pointer',
                    }}
                  >
                    {suggestion}
                  </button>
                ))}
              </div>
            </div>
          ) : draft.messages.map((message, index) => {
            const teacher = message.role === 'teacher'
            return (
              <div
                key={`${message.role}-${index}`}
                style={{
                  display: 'flex',
                  justifyContent: teacher ? 'flex-end' : 'flex-start',
                  marginBottom: 10,
                }}
              >
                <div style={{
                  maxWidth: '86%',
                  padding: '9px 12px',
                  borderRadius: teacher
                    ? '12px 12px 3px 12px'
                    : '12px 12px 12px 3px',
                  background: teacher ? C.primary : '#fff',
                  color: teacher ? '#fff' : '#374151',
                  border: teacher ? 'none' : '1px solid #E5E7EB',
                  boxShadow: teacher ? 'none' : '0 1px 2px rgba(0,0,0,0.04)',
                }}>
                  {teacher ? message.content : (
                    <DiscussionMarkdown content={message.content} compact />
                  )}
                </div>
              </div>
            )
          })}
          {phase === 'discussing' && (
            <div style={{ color: '#7C3AED', fontSize: 12, padding: '4px 2px' }}>
              AI正在结合前后页整理方案…
            </div>
          )}
        </div>

        <div style={{
          padding: 12,
          borderTop: '1px solid #E5E7EB',
          background: '#fff',
        }}>
          <textarea
            ref={inputRef}
            value={draft.input}
            onChange={(event: ChangeEvent<HTMLTextAreaElement>) => updateDraft({ input: event.target.value })}
            onKeyDown={(event: KeyboardEvent<HTMLTextAreaElement>) => {
              if (protectedDraft.handleKeyDown(event)) {
                return
              }

              if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault()
                void sendMessage()
              }
            }}
            placeholder="说明本页目标、内容、互动或视觉要求；Enter发送，Shift+Enter换行"
            rows={3}
            disabled={isBusy || voiceInput.isActive}
            style={{
              width: '100%',
              boxSizing: 'border-box',
              resize: 'vertical',
              minHeight: 74,
              padding: '9px 11px',
              border: '1px solid #D1D5DB',
              borderRadius: 9,
              fontSize: 13,
              lineHeight: 1.6,
              outline: 'none',
            }}
          />
          <div style={{
            marginTop: 8,
            display: 'flex',
            alignItems: 'center',
            gap: 10,
          }}>
            <span style={{
              flex: 1,
              color: voiceInput.status === 'error'
                ? '#B91C1C'
                : voiceInput.isActive
                  ? '#7C3AED'
                  : '#9CA3AF',
              fontSize: 11,
              lineHeight: 1.5,
            }}>
              {voiceInput.statusText
                || '对话和方案已自动保存 · 点击麦克风可语音输入 · Ctrl/Command+Z可恢复误删'}
            </span>

            <VoiceInputButton
              status={voiceInput.status}
              isSupported={voiceInput.isSupported}
              elapsedSeconds={voiceInput.elapsedSeconds}
              disabled={isBusy}
              error={voiceInput.error}
              onStart={voiceInput.begin}
              onStop={voiceInput.stop}
              onCancel={voiceInput.cancel}
            />

            <button
              type="button"
              onClick={() => void sendMessage()}
              disabled={isBusy || voiceInput.isActive || !draft.input.trim()}
              style={{
                padding: '8px 18px',
                border: 'none',
                borderRadius: 8,
                background: isBusy || voiceInput.isActive || !draft.input.trim()
                  ? '#D1D5DB'
                  : C.primary,
                color: '#fff',
                fontSize: 13,
                fontWeight: 700,
                cursor: isBusy || voiceInput.isActive || !draft.input.trim()
                  ? 'not-allowed'
                  : 'pointer',
              }}
            >
              {phase === 'discussing' ? '讨论中…' : '发送'}
            </button>
          </div>
        </div>
      </section>

      <aside style={{ minWidth: 0, display: 'flex', flexDirection: 'column', gap: 10 }}>
        <div style={{ ...cardStyle, padding: 14 }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 8,
            marginBottom: 10,
          }}>
            <span style={{ fontSize: 14, fontWeight: 800, color: '#1F2937' }}>
              当前页面方案
            </span>
            <span style={{
              padding: '3px 8px',
              borderRadius: 999,
              background: canGenerate ? '#D1FAE5' : '#FEF3C7',
              color: canGenerate ? '#047857' : '#92400E',
              fontSize: 11,
              fontWeight: 700,
            }}>
              {canGenerate ? '可以生成' : '继续讨论'}
            </span>
          </div>

          {draft.summary && (
            <div style={{
              marginBottom: 10,
              padding: '8px 10px',
              borderRadius: 8,
              background: '#F5F3FF',
              color: '#5B21B6',
              fontSize: 12,
              lineHeight: 1.6,
            }}>
              {draft.summary}
            </div>
          )}

          {insertPositionChanged && (
            <div style={{
              marginBottom: 10,
              padding: '8px 10px',
              borderRadius: 8,
              background: '#FFF7ED',
              color: '#C2410C',
              fontSize: 12,
              lineHeight: 1.6,
            }}>
              插入位置已从第 {draft.lastDiscussedInsertAt} 页改为第 {insertAt} 页。
              请再发一轮消息重新确认前后页衔接。
            </div>
          )}

          <dl style={{ margin: 0, display: 'grid', gap: 9 }}>
            {[
              ['标题', addPagePlanField(draft.plan.title)],
              ['教学目的', addPagePlanField(draft.plan.purpose)],
              ['内容概要', addPagePlanField(draft.plan.content_summary)],
              ['互动方式', addPagePlanField(draft.plan.interaction_type, 'static')],
              ['视觉形式', addPagePlanField(draft.plan.visual_format, 'text_heavy')],
              ['媒体需求', addPagePlanField(draft.plan.media_requirements, '无')],
              ['复杂度', `${draft.plan.estimated_complexity} / 5`],
            ].map(item => (
              <div
                key={item[0]}
                style={{
                  display: 'grid',
                  gridTemplateColumns: '68px minmax(0, 1fr)',
                  gap: 8,
                  alignItems: 'start',
                }}
              >
                <dt style={{ color: '#9CA3AF', fontSize: 11, lineHeight: 1.55 }}>
                  {item[0]}
                </dt>
                <dd style={{
                  margin: 0,
                  color: '#374151',
                  fontSize: 12,
                  lineHeight: 1.55,
                  wordBreak: 'break-word',
                }}>
                  {item[1]}
                </dd>
              </div>
            ))}
          </dl>
        </div>

        {errorMessage && (
          <div style={{
            padding: '9px 11px',
            borderRadius: 9,
            background: '#FEE2E2',
            color: '#B91C1C',
            fontSize: 12,
            lineHeight: 1.6,
          }}>
            {errorMessage}
          </div>
        )}

        <button
          type="button"
          onClick={() => void generateAndInsert()}
          disabled={!canGenerate}
          style={{
            width: '100%',
            padding: '11px 14px',
            border: 'none',
            borderRadius: 10,
            background: canGenerate ? '#059669' : '#D1D5DB',
            color: '#fff',
            fontSize: 14,
            fontWeight: 800,
            cursor: canGenerate ? 'pointer' : 'not-allowed',
          }}
        >
          按此方案生成并插入第 {insertAt} 页
        </button>
        <div style={{
          color: '#9CA3AF',
          fontSize: 11,
          lineHeight: 1.6,
          textAlign: 'center',
        }}>
          只有点击上方按钮才会真正创建页面
        </div>
      </aside>
    </div>
  )
}
