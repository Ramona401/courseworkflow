/**
 * AnnotationPanel.tsx — 课件页级批注面板
 *
 * 页面匹配规则：
 *   - 优先使用稳定page_id匹配当前页面；
 *   - 仅在旧后端未返回page_id时回退使用page_number；
 *   - page_id为null表示原页面已删除，不挂载到任何现有页面；
 *   - 页面重排后，批注仍跟随原页面。
 *
 * 本组件接收批注全集，内部筛选当前页面批注；
 * 增删改完成后通过onChanged通知父组件重新加载。
 */
import { useState } from 'react'

import {
  createCWAnnotation,
  deleteCWAnnotation,
  resolveCWAnnotation,
  type CoursewareAnnotation,
} from '@/api/coursewares'

import { C } from './workshopConstants'

interface Props {
  /** 课件ID */
  coursewareId: string

  /**
   * 当前页面稳定ID。
   *
   * 暂时可选，用于兼容父组件分步升级；
   * 父组件接入后应始终传入。
   */
  pageId?: string

  /** 当前页面页码，用于展示和创建请求 */
  pageNumber: number

  /** 课件批注全集 */
  annotations: CoursewareAnnotation[]

  /** 增删改后通知父组件重新加载 */
  onChanged: () => void
}

/** 简短时间格式：YYYY-MM-DD HH:mm */
function fmtTime(iso: string): string {
  if (!iso) return ''

  const date = new Date(iso)

  if (Number.isNaN(date.getTime())) return ''

  const pad = (number: number) =>
    String(number).padStart(2, '0')

  return [
    `${date.getFullYear()}-${pad(
      date.getMonth() + 1,
    )}-${pad(date.getDate())}`,
    `${pad(date.getHours())}:${pad(
      date.getMinutes(),
    )}`,
  ].join(' ')
}

/** 判断批注是否属于当前页面。 */
function annotationBelongsToPage(
  annotation: CoursewareAnnotation,
  pageId: string | undefined,
  pageNumber: number,
): boolean {
  // 新后端明确返回null，表示原页面已经删除。
  if (annotation.page_id === null) {
    return false
  }

  // 当前页面有稳定ID，并且接口返回了page_id时，必须按稳定ID匹配。
  if (
    pageId &&
    annotation.page_id !== undefined
  ) {
    return annotation.page_id === pageId
  }

  // 仅兼容旧后端未返回page_id，或父组件尚未接入pageId的部署窗口。
  return annotation.page_number === pageNumber
}

export default function AnnotationPanel({
  coursewareId,
  pageId,
  pageNumber,
  annotations,
  onChanged,
}: Props) {
  const [input, setInput] = useState('')
  const [submitting, setSubmitting] =
    useState(false)
  const [busyId, setBusyId] = useState('')

  // 当前页面批注：待处理在前，已处理在后，归档项不显示。
  const pageItems = annotations
    .filter(
      (annotation) =>
        annotation.status !== 'archived' &&
        annotationBelongsToPage(
          annotation,
          pageId,
          pageNumber,
        ),
    )
    .sort((left, right) => {
      const leftWeight =
        left.status === 'pending' ? 0 : 1
      const rightWeight =
        right.status === 'pending' ? 0 : 1

      if (leftWeight !== rightWeight) {
        return leftWeight - rightWeight
      }

      return left.created_at <
        right.created_at
        ? -1
        : 1
    })

  const pendingCount = pageItems.filter(
    (annotation) =>
      annotation.status === 'pending',
  ).length

  const handleSubmit = async () => {
    const text = input.trim()

    if (
      !text ||
      submitting ||
      pageNumber <= 0
    ) {
      return
    }

    setSubmitting(true)

    try {
      await createCWAnnotation(
        coursewareId,
        pageNumber,
        text,
      )

      setInput('')
      onChanged()
    } catch (cause: any) {
      alert(
        cause?.response?.data?.message ||
          cause?.message ||
          '批注提交失败',
      )
    } finally {
      setSubmitting(false)
    }
  }

  const handleToggle = async (
    annotation: CoursewareAnnotation,
  ) => {
    if (busyId) return

    setBusyId(annotation.id)

    try {
      const nextStatus =
        annotation.status === 'resolved'
          ? 'pending'
          : 'resolved'

      await resolveCWAnnotation(
        annotation.id,
        nextStatus,
      )

      onChanged()
    } catch (cause: any) {
      alert(
        cause?.response?.data?.message ||
          cause?.message ||
          '操作失败',
      )
    } finally {
      setBusyId('')
    }
  }

  const handleDelete = async (
    annotation: CoursewareAnnotation,
  ) => {
    if (busyId) return

    if (
      !window.confirm(
        '确定删除这条批注吗？此操作不可恢复。',
      )
    ) {
      return
    }

    setBusyId(annotation.id)

    try {
      await deleteCWAnnotation(annotation.id)
      onChanged()
    } catch (cause: any) {
      alert(
        cause?.response?.data?.message ||
          cause?.message ||
          '删除失败',
      )
    } finally {
      setBusyId('')
    }
  }

  if (pageNumber <= 0) return null

  return (
    <div
      style={{
        marginTop: 16,
        borderRadius: 12,
        border: `1px solid ${C.border}`,
        background: C.white,
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          padding: '10px 14px',
          borderBottom: `1px solid ${C.border}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          background: '#FAFAFA',
        }}
      >
        <div
          style={{
            fontSize: 14,
            fontWeight: 600,
            color: C.textPrimary,
          }}
        >
          💬 第 {pageNumber} 页批注

          {pendingCount > 0 && (
            <span
              style={{
                marginLeft: 8,
                padding: '1px 8px',
                borderRadius: 10,
                background: '#FEE2E2',
                color: '#DC2626',
                fontSize: 12,
                fontWeight: 600,
              }}
            >
              {pendingCount} 条待处理
            </span>
          )}
        </div>

        <span
          style={{
            fontSize: 12,
            color: C.textSecondary,
          }}
        >
          共 {pageItems.length} 条
        </span>
      </div>

      <div
        style={{
          maxHeight: 320,
          overflowY: 'auto',
          padding:
            pageItems.length > 0
              ? '8px 0'
              : 0,
        }}
      >
        {pageItems.length === 0 ? (
          <div
            style={{
              padding: '20px 14px',
              textAlign: 'center',
              color: C.textSecondary,
              fontSize: 13,
            }}
          >
            这一页还没有批注，在下方写下第一条吧。
          </div>
        ) : (
          pageItems.map((annotation) => {
            const resolved =
              annotation.status ===
              'resolved'

            return (
              <div
                key={annotation.id}
                style={{
                  padding: '10px 14px',
                  borderBottom:
                    `1px solid ${C.border}`,
                  opacity: resolved ? 0.6 : 1,
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent:
                      'space-between',
                    marginBottom: 4,
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                    }}
                  >
                    <span
                      style={{
                        fontSize: 13,
                        fontWeight: 600,
                        color: C.textPrimary,
                      }}
                    >
                      {annotation.reviewer_name ||
                        '匿名'}
                    </span>

                    <span
                      style={{
                        fontSize: 11,
                        color: C.textSecondary,
                      }}
                    >
                      {fmtTime(
                        annotation.created_at,
                      )}
                    </span>

                    {resolved && (
                      <span
                        style={{
                          fontSize: 11,
                          color: '#059669',
                          fontWeight: 600,
                        }}
                      >
                        ✓ 已处理
                      </span>
                    )}
                  </div>

                  <div
                    style={{
                      display: 'flex',
                      gap: 6,
                    }}
                  >
                    <button
                      onClick={() =>
                        handleToggle(
                          annotation,
                        )
                      }
                      disabled={
                        busyId === annotation.id
                      }
                      title={
                        resolved
                          ? '重新标记为待处理'
                          : '标记为已处理'
                      }
                      style={{
                        padding: '2px 8px',
                        borderRadius: 6,
                        border:
                          `1px solid ${C.border}`,
                        background:
                          'transparent',
                        color: resolved
                          ? C.textSecondary
                          : '#059669',
                        fontSize: 12,
                        cursor:
                          busyId ===
                          annotation.id
                            ? 'default'
                            : 'pointer',
                      }}
                    >
                      {resolved
                        ? '↩ 重开'
                        : '✓ 已处理'}
                    </button>

                    <button
                      onClick={() =>
                        handleDelete(
                          annotation,
                        )
                      }
                      disabled={
                        busyId === annotation.id
                      }
                      title="删除批注"
                      style={{
                        padding: '2px 8px',
                        borderRadius: 6,
                        border:
                          `1px solid ${C.border}`,
                        background:
                          'transparent',
                        color: '#DC2626',
                        fontSize: 12,
                        cursor:
                          busyId ===
                          annotation.id
                            ? 'default'
                            : 'pointer',
                      }}
                    >
                      🗑
                    </button>
                  </div>
                </div>

                <div
                  style={{
                    fontSize: 13,
                    color: C.textPrimary,
                    lineHeight: 1.6,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    textDecoration: resolved
                      ? 'line-through'
                      : 'none',
                  }}
                >
                  {annotation.content}
                </div>
              </div>
            )
          })
        )}
      </div>

      <div
        style={{
          padding: '10px 14px',
          borderTop: `1px solid ${C.border}`,
          background: '#FAFAFA',
        }}
      >
        <textarea
          value={input}
          onChange={(event) =>
            setInput(event.target.value)
          }
          onKeyDown={(event) => {
            if (
              event.key === 'Enter' &&
              (
                event.ctrlKey ||
                event.metaKey
              )
            ) {
              event.preventDefault()
              handleSubmit()
            }
          }}
          placeholder={`对第 ${pageNumber} 页留下批注…（Ctrl+Enter 提交）`}
          rows={2}
          style={{
            width: '100%',
            boxSizing: 'border-box',
            padding: '8px 10px',
            borderRadius: 8,
            border: `1px solid ${C.border}`,
            fontSize: 13,
            resize: 'vertical',
            fontFamily: 'inherit',
            outline: 'none',
          }}
        />

        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            marginTop: 8,
          }}
        >
          <button
            onClick={handleSubmit}
            disabled={
              submitting ||
              !input.trim()
            }
            style={{
              padding: '6px 16px',
              borderRadius: 8,
              border: 'none',
              background:
                input.trim() &&
                !submitting
                  ? C.primary
                  : C.border,
              color: C.white,
              fontSize: 13,
              fontWeight: 600,
              cursor:
                input.trim() &&
                !submitting
                  ? 'pointer'
                  : 'default',
            }}
          >
            {submitting
              ? '提交中…'
              : '发表批注'}
          </button>
        </div>
      </div>
    </div>
  )
}
