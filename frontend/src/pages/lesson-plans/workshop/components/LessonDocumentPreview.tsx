/**
 * LessonDocumentPreview.tsx — 教案正文预览和章节定位。
 *
 * 负责Markdown渲染、滚动定位、当前章节高亮和
 * 章节AI修改入口。图片快捷操作将在本组件接入。
 */

import type {
  RefObject,
  UIEvent,
} from 'react'
import {
  renderMarkdown,
  type RenderedMarkdownImage,
} from '@/pages/lesson-plans/plan-detail/components/planDetailConstants'
import type {
  LessonDocumentSection,
  LessonDocumentStructure,
} from './lessonDocumentStructure'

interface LessonDocumentPreviewProps {
  content: string
  structure: LessonDocumentStructure
  activeSectionID: string
  compact: boolean
  rewriteDisabled: boolean
  disabledReason: string
  imageActionDisabled: boolean
  removingImageKey: string
  sectionRefs: RefObject<
    Map<string, HTMLDivElement>
  >
  onActiveSectionChange: (
    sectionID: string,
  ) => void
  onOpenSectionRewrite: (
    section: LessonDocumentSection,
  ) => void
  onRemoveImage: (
    image: RenderedMarkdownImage,
    rangeStart: number,
    rangeEnd: number,
  ) => void
}

export default function LessonDocumentPreview({
  content,
  structure,
  activeSectionID,
  compact,
  rewriteDisabled,
  disabledReason,
  imageActionDisabled,
  removingImageKey,
  sectionRefs,
  onActiveSectionChange,
  onOpenSectionRewrite,
  onRemoveImage,
}: LessonDocumentPreviewProps) {
  const handlePreviewScroll = (
    event: UIEvent<HTMLDivElement>,
  ) => {
    const container =
      event.currentTarget

    const containerTop =
      container
        .getBoundingClientRect()
        .top + 32

    let nextActive =
      structure.sections[0]

    for (
      const section of
      structure.sections
    ) {
      const element =
        sectionRefs.current?.get(
          section.id,
        )

      if (!element) continue

      if (
        element
          .getBoundingClientRect()
          .top <= containerTop
      ) {
        nextActive = section
      } else {
        break
      }
    }

    if (
      nextActive &&
      nextActive.id !==
        activeSectionID
    ) {
      onActiveSectionChange(
        nextActive.id,
      )
    }
  }

  return (
    <div
      onScroll={handlePreviewScroll}
      style={{
        height: '100%',
        minHeight: 0,
        overflowY: 'auto',
        padding: compact
          ? '12px 14px'
          : '16px 20px',
        boxSizing: 'border-box',
        fontSize: compact
          ? '13px'
          : '14px',
        lineHeight: 1.85,
        scrollBehavior: 'smooth',
      }}
    >
      {structure
        .preambleMarkdown
        .trim() && (
        <div style={{
          marginBottom: '8px',
        }}>
          {renderMarkdown(
            structure
              .preambleMarkdown,
            {
              imageKeyPrefix:
                'preamble',
              imageActionDisabled,
              removingImageKey,
              onRemoveImage: image =>
                onRemoveImage(
                  image,
                  0,
                  structure
                    .preambleMarkdown
                    .length,
                ),
            },
          )}
        </div>
      )}

      {structure.sections.map(
        section => {
          const active =
            section.id ===
            activeSectionID

          const sectionMarkdown =
            content.slice(
              section.startOffset,
              section.endOffset,
            )

          return (
            <div
              key={section.id}
              id={section.id}
              ref={element => {
                if (element) {
                  sectionRefs
                    .current
                    ?.set(
                      section.id,
                      element,
                    )
                } else {
                  sectionRefs
                    .current
                    ?.delete(
                      section.id,
                    )
                }
              }}
              onClick={() =>
                onActiveSectionChange(
                  section.id,
                )
              }
              style={{
                position: 'relative',
                margin:
                  '0 -6px 8px',
                padding:
                  '6px 84px 6px 6px',
                borderRadius: '9px',
                border: active
                  ? '1px solid rgba(79,123,232,0.16)'
                  : '1px solid transparent',
                background: active
                  ? 'rgba(79,123,232,0.025)'
                  : 'transparent',
                scrollMarginTop: '12px',
                transition:
                  'background 150ms ease, border-color 150ms ease',
              }}
            >
              <button
                type="button"
                onClick={event => {
                  event.stopPropagation()

                  onOpenSectionRewrite(
                    section,
                  )
                }}
                disabled={
                  rewriteDisabled
                }
                title={
                  rewriteDisabled
                    ? disabledReason ||
                      '当前暂不能使用AI修改'
                    : `让AI修改“${section.title}”`
                }
                style={{
                  position:
                    'absolute',
                  top: '8px',
                  right: '6px',
                  padding: '4px 7px',
                  borderRadius: '7px',
                  border:
                    '1px solid rgba(79,123,232,0.2)',
                  background:
                    rewriteDisabled
                      ? '#F9FAFB'
                      : '#FFFFFF',
                  color:
                    rewriteDisabled
                      ? '#D1D5DB'
                      : '#4F7BE8',
                  fontSize: '10px',
                  fontWeight: 600,
                  cursor:
                    rewriteDisabled
                      ? 'not-allowed'
                      : 'pointer',
                  boxShadow:
                    rewriteDisabled
                      ? 'none'
                      : '0 2px 8px rgba(79,123,232,0.08)',
                  whiteSpace:
                    'nowrap',
                }}
              >
                ✨ AI修改
              </button>

              {renderMarkdown(
                sectionMarkdown,
                {
                  imageKeyPrefix:
                    section.id,
                  imageActionDisabled,
                  removingImageKey,
                  onRemoveImage: image =>
                    onRemoveImage(
                      image,
                      section.startOffset,
                      section.endOffset,
                    ),
                },
              )}
            </div>
          )
        },
      )}
    </div>
  )
}
