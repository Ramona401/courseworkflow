/**
 * LessonSectionAIEditor.tsx — 教案章节AI修改面板。
 *
 * 交互流程：
 *   1. 老师填写修改要求；
 *   2. 流式生成修改后的章节正文；
 *   3. 老师预览、继续调整或重新生成；
 *   4. 确认后调用原子应用接口。
 *
 * 父组件使用section.id作为React key。
 * 切换章节时组件会自然重新挂载，不在effect中同步重置本地状态。
 */

import { useEffect, useRef, useState } from 'react'
import { renderMarkdown } from '@/pages/lesson-plans/plan-detail/components/planDetailConstants'
import {
  applyLessonPlanSectionRewrite,
  generateLessonPlanSectionRewrite,
  type LessonPlanSectionRewriteApplyResponse,
  type LessonPlanSectionRewriteConnection,
  type LessonPlanSectionRewritePreview,
} from '@/api/lesson-plan-section-rewrite'
import type {
  LessonDocumentSection,
} from './lessonDocumentStructure'

interface LessonSectionAIEditorProps {
  planID: string
  currentVersion: number
  section: LessonDocumentSection
  disabled?: boolean
  onApplied: (
    result: LessonPlanSectionRewriteApplyResponse,
  ) => void | Promise<void>
  onClose: () => void
}

type EditorStatus =
  | 'idle'
  | 'connecting'
  | 'streaming'
  | 'done'
  | 'applying'
  | 'success'
  | 'error'

const quickInstructions = [
  '表达更具体，补充可执行的教学步骤',
  '适当精简，保留核心要求和关键信息',
  '增加分层设计，照顾不同基础的学生',
]

export default function LessonSectionAIEditor({
  planID,
  currentVersion,
  section,
  disabled = false,
  onApplied,
  onClose,
}: LessonSectionAIEditorProps) {
  const [instruction, setInstruction] = useState('')
  const [streamText, setStreamText] = useState('')
  const [preview, setPreview] =
    useState<LessonPlanSectionRewritePreview | null>(null)
  const [status, setStatus] = useState<EditorStatus>('idle')
  const [errorMessage, setErrorMessage] = useState('')
  const [successMessage, setSuccessMessage] = useState('')

  const connectionRef =
    useRef<LessonPlanSectionRewriteConnection | null>(null)

  useEffect(() => {
    return () => {
      connectionRef.current?.close()
    }
  }, [])

  const generate = () => {
    const requestText = instruction.trim()
    if (!requestText || disabled) return

    connectionRef.current?.close()
    setStatus('connecting')
    setStreamText('')
    setPreview(null)
    setErrorMessage('')
    setSuccessMessage('')

    connectionRef.current = generateLessonPlanSectionRewrite(
      planID,
      {
        base_version: currentVersion,
        locator: section.locator,
        instruction: requestText,
      },
      {
        onConnected: () => setStatus('streaming'),
        onChunk: chunk => {
          setStatus('streaming')
          setStreamText(previous => previous + chunk)
        },
        onDone: result => {
          setPreview(result)
          setStreamText(result.replacement_markdown)
          setStatus('done')
          connectionRef.current = null
        },
        onError: message => {
          setErrorMessage(message)
          setStatus('error')
          connectionRef.current = null
        },
      },
    )
  }

  const apply = async () => {
    if (!preview || disabled) return

    setStatus('applying')
    setErrorMessage('')
    setSuccessMessage('')

    try {
      const result = await applyLessonPlanSectionRewrite(
        planID,
        {
          base_version: preview.base_version,
          locator: preview.section.locator,
          section_hash: preview.section.section_hash,
          replacement_markdown:
            preview.replacement_markdown,
        },
      )

      await onApplied(result)

      setPreview(null)
      setStreamText('')
      setSuccessMessage(
        result.changed
          ? `已应用修改，当前版本v${result.current_version}`
          : `正文没有变化，仍为v${result.current_version}`,
      )
      setStatus('success')
    } catch (error) {
      const message = error instanceof Error
        ? error.message
        : '应用修改失败'

      setErrorMessage(message)
      setStatus('error')
    }
  }

  const busy =
    status === 'connecting' ||
    status === 'streaming' ||
    status === 'applying'

  return (
    <div style={{
      height: '100%',
      display: 'flex',
      flexDirection: 'column',
      background: '#FFFFFF',
      boxSizing: 'border-box',
    }}>
      <div style={{
        padding: '12px 14px',
        borderBottom: '1px solid #E5E7EB',
        display: 'flex',
        alignItems: 'flex-start',
        justifyContent: 'space-between',
        gap: '12px',
        flexShrink: 0,
      }}>
        <div style={{ minWidth: 0 }}>
          <div style={{
            fontSize: '13px',
            fontWeight: 700,
            color: '#1F2937',
          }}>
            ✨ AI修改：{section.title}
          </div>
          <div style={{
            marginTop: '3px',
            fontSize: '11px',
            color: '#9CA3AF',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}>
            {section.headingPath.join(' / ')}
          </div>
        </div>

        <button
          type="button"
          onClick={() => {
            connectionRef.current?.close()
            onClose()
          }}
          aria-label="关闭AI修改面板"
          style={{
            width: '30px',
            height: '30px',
            border: 'none',
            borderRadius: '8px',
            background: '#F3F4F6',
            color: '#6B7280',
            cursor: 'pointer',
            fontSize: '17px',
            flexShrink: 0,
          }}
        >
          ×
        </button>
      </div>

      <div style={{
        flex: 1,
        minHeight: 0,
        overflowY: 'auto',
        padding: '12px 14px 18px',
      }}>
        <div style={{
          padding: '10px 12px',
          borderRadius: '9px',
          background: '#F9FAFB',
          border: '1px solid #F3F4F6',
          marginBottom: '12px',
        }}>
          <div style={{
            fontSize: '11px',
            fontWeight: 700,
            color: '#6B7280',
            marginBottom: '5px',
          }}>
            当前章节正文
          </div>
          <div style={{
            maxHeight: '110px',
            overflowY: 'auto',
            fontSize: '12px',
            lineHeight: 1.65,
            color: '#4B5563',
          }}>
            {section.bodyMarkdown.trim()
              ? renderMarkdown(section.bodyMarkdown)
              : '当前标题下暂无直属正文。'}
          </div>
        </div>

        <label style={{
          display: 'block',
          fontSize: '12px',
          fontWeight: 700,
          color: '#374151',
          marginBottom: '6px',
        }}>
          希望AI怎么修改？
        </label>

        <textarea
          value={instruction}
          onChange={event => setInstruction(event.target.value)}
          disabled={busy || disabled}
          maxLength={4000}
          rows={4}
          placeholder="例如：增加一个贴近学生生活的导入活动，并明确教师提问和学生预期回答。"
          style={{
            width: '100%',
            boxSizing: 'border-box',
            padding: '10px 11px',
            borderRadius: '9px',
            border: '1px solid #D1D5DB',
            outline: 'none',
            resize: 'vertical',
            fontFamily: 'inherit',
            fontSize: '13px',
            lineHeight: 1.65,
            color: '#1F2937',
            background: busy || disabled ? '#F9FAFB' : '#FFFFFF',
          }}
        />

        <div style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: '6px',
          marginTop: '8px',
        }}>
          {quickInstructions.map(item => (
            <button
              key={item}
              type="button"
              onClick={() => setInstruction(item)}
              disabled={busy || disabled}
              style={{
                padding: '5px 9px',
                borderRadius: '14px',
                border: '1px solid #E5E7EB',
                background: '#FFFFFF',
                color: '#6B7280',
                fontSize: '11px',
                cursor: busy || disabled
                  ? 'not-allowed'
                  : 'pointer',
              }}
            >
              {item}
            </button>
          ))}
        </div>

        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: '10px',
          marginTop: '10px',
        }}>
          <span style={{
            fontSize: '10px',
            color: '#9CA3AF',
          }}>
            {instruction.length}/4000
          </span>

          <button
            type="button"
            onClick={generate}
            disabled={
              busy ||
              disabled ||
              !instruction.trim()
            }
            style={{
              padding: '8px 15px',
              borderRadius: '8px',
              border: 'none',
              background:
                busy ||
                disabled ||
                !instruction.trim()
                  ? '#E5E7EB'
                  : '#4F7BE8',
              color:
                busy ||
                disabled ||
                !instruction.trim()
                  ? '#9CA3AF'
                  : '#FFFFFF',
              fontSize: '12px',
              fontWeight: 700,
              cursor:
                busy ||
                disabled ||
                !instruction.trim()
                  ? 'not-allowed'
                  : 'pointer',
            }}
          >
            {status === 'connecting'
              ? '正在连接…'
              : status === 'streaming'
                ? '正在生成…'
                : preview
                  ? '重新生成'
                  : '生成修改预览'}
          </button>
        </div>

        {errorMessage && (
          <div style={{
            marginTop: '12px',
            padding: '9px 11px',
            borderRadius: '8px',
            background: 'rgba(239,68,68,0.07)',
            border: '1px solid rgba(239,68,68,0.22)',
            color: '#DC2626',
            fontSize: '12px',
            lineHeight: 1.6,
          }}>
            ⚠️ {errorMessage}
          </div>
        )}

        {successMessage && (
          <div style={{
            marginTop: '12px',
            padding: '9px 11px',
            borderRadius: '8px',
            background: 'rgba(16,185,129,0.08)',
            border: '1px solid rgba(16,185,129,0.24)',
            color: '#047857',
            fontSize: '12px',
            lineHeight: 1.6,
          }}>
            ✓ {successMessage}
          </div>
        )}

        {(status === 'streaming' || preview) && streamText && (
          <div style={{
            marginTop: '12px',
            padding: '11px 12px',
            borderRadius: '9px',
            background: '#F8FAFF',
            border: '1px solid rgba(79,123,232,0.22)',
          }}>
            <div style={{
              display: 'flex',
              justifyContent: 'space-between',
              gap: '10px',
              marginBottom: '7px',
            }}>
              <span style={{
                fontSize: '12px',
                fontWeight: 700,
                color: '#365FB8',
              }}>
                修改预览
              </span>
              {status === 'streaming' && (
                <span style={{
                  fontSize: '11px',
                  color: '#4F7BE8',
                }}>
                  生成中…
                </span>
              )}
            </div>

            <div style={{
              fontSize: '13px',
              lineHeight: 1.75,
              color: '#1F2937',
            }}>
              {renderMarkdown(streamText)}
            </div>
          </div>
        )}

        {preview && (
          <div style={{
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px',
            marginTop: '12px',
          }}>
            <button
              type="button"
              onClick={generate}
              disabled={busy || disabled}
              style={{
                padding: '7px 12px',
                borderRadius: '8px',
                border: '1px solid #D1D5DB',
                background: '#FFFFFF',
                color: '#6B7280',
                fontSize: '12px',
                cursor: busy || disabled
                  ? 'not-allowed'
                  : 'pointer',
              }}
            >
              重新生成
            </button>

            <button
              type="button"
              onClick={() => void apply()}
              disabled={busy || disabled}
              style={{
                padding: '7px 14px',
                borderRadius: '8px',
                border: 'none',
                background:
                  busy || disabled
                    ? '#E5E7EB'
                    : '#10B981',
                color:
                  busy || disabled
                    ? '#9CA3AF'
                    : '#FFFFFF',
                fontSize: '12px',
                fontWeight: 700,
                cursor:
                  busy || disabled
                    ? 'not-allowed'
                    : 'pointer',
              }}
            >
              {status === 'applying'
                ? '正在应用…'
                : '✓ 应用到教案'}
            </button>
          </div>
        )}

        <div style={{
          marginTop: '12px',
          fontSize: '10px',
          color: '#9CA3AF',
          lineHeight: 1.6,
        }}>
          AI只会修改当前标题下的直属正文。应用前会再次校验教案版本和段落内容，冲突时不会覆盖新内容。
        </div>
      </div>
    </div>
  )
}
