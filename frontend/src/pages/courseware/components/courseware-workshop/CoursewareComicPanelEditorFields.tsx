/**
 * CoursewareComicPanelEditorFields.tsx
 *
 * 单格编辑器的轻量标题、状态和兼容出口。
 *
 * 当前主流程直接在画布内编辑，旧静态预览和旧表单名称继续保留，
 * 避免影响其他历史调用方。
 */

import type {
  CSSProperties,
  KeyboardEvent,
} from 'react'

import type {
  CoursewareComicOverlayDocument,
  CoursewareComicOverlayElement,
  CoursewareComicPanel,
} from '@/api/coursewares'

import CoursewareComicPanelPreview from './CoursewareComicPanelPreview'

export {
  CoursewareComicPanelFooter,
  coursewareComicPanelEditorStyles,
} from './CoursewareComicPanelEditorFooter'

const C = {
  primary: '#7C3AED',
  success: '#059669',
  warning: '#D97706',
  danger: '#DC2626',
  text: '#1F2937',
  textSecondary: '#64748B',
  textMuted: '#94A3B8',
}

export function CoursewareComicPanelHeader({
  panel,
  overlayDirty,
}: {
  panel:
    CoursewareComicPanel
  overlayDirty:
    boolean
}) {
  const status =
    resolvePanelStatus(
      panel,
    )

  return (
    <header style={headerStyle}>
      <div style={headerMainStyle}>
        <div style={headerTitleStyle}>
          第{panel.panel_no}格
          <span style={headerDividerStyle}>
            ·
          </span>
          {panel.story_purpose}
        </div>

        <div style={headerClaimStyle}>
          {panel.knowledge_claim}
        </div>
      </div>

      <div style={headerStatusStyle}>
        <span
          style={{
            ...statusBadgeStyle,
            color:
              status.color,
            background:
              `${status.color}16`,
          }}
        >
          {status.label}
        </span>

        {overlayDirty && (
          <span
            title="修改已保存在当前标签页，将自动同步"
            style={dirtyDotStyle}
          />
        )}
      </div>
    </header>
  )
}

export function CoursewareComicNotice({
  text,
}: {
  text: string
}) {
  const error =
    text.startsWith(
      '❌',
    )

  const warning =
    text.startsWith(
      '⚠️',
    )

  return (
    <div
      style={{
        ...noticeStyle,
        borderColor:
          error
            ? '#FECACA'
            : warning
              ? '#FDE68A'
              : '#A7F3D0',
        background:
          error
            ? '#FEF2F2'
            : warning
              ? '#FFFBEB'
              : '#ECFDF5',
        color:
          error
            ? C.danger
            : warning
              ? '#92400E'
              : C.success,
      }}
    >
      {text}
    </div>
  )
}

/**
 * 旧静态预览兼容出口。
 * 当前主编辑器已经直接使用CoursewareComicDirectCanvas。
 */
export function CoursewareComicEditablePreview({
  panel,
  overlayDocument,
}: {
  panel:
    CoursewareComicPanel

  overlayDocument:
    CoursewareComicOverlayDocument
}) {
  return (
    <CoursewareComicPanelPreview
      panel={panel}
      aspectRatio="courseware"
      overlayDocument={
        overlayDocument
      }
    />
  )
}

/**
 * 旧下方表单兼容出口。
 * 画布编辑器不再渲染该组件。
 */
export function CoursewareComicOverlayEditor(
  props: {
    narrationText: string
    elements:
      CoursewareComicOverlayElement[]
    disabled: boolean

    onNarrationChange:
      (
        value: string,
      ) => void

    onContentChange: (
      elementID: string,
      value: string,
    ) => void

    onQuestionTextChange: (
      elementID: string,
      field:
        | 'question'
        | 'explanation',
      value: string,
    ) => void

    onQuestionOptionsChange: (
      elementID: string,
      value: string,
    ) => void

    onQuestionAnswerChange: (
      elementID: string,
      value: number,
    ) => void

    onKeyDown: (
      event:
        KeyboardEvent<HTMLElement>,
    ) => boolean
  },
) {
  void props
  return null
}

function resolvePanelStatus(
  panel:
    CoursewareComicPanel,
): {
  label: string
  color: string
} {
  switch (panel.status) {
  case 'generated':
    return {
      label: '图片就绪',
      color: C.success,
    }

  case 'stale':
    return {
      label: '需要重画',
      color: C.warning,
    }

  case 'generating':
    return {
      label: '生成中',
      color: C.primary,
    }

  case 'failed':
    return {
      label: '生成失败',
      color: C.danger,
    }

  default:
    return {
      label: '待生成',
      color: C.textMuted,
    }
  }
}

const headerStyle:
  CSSProperties = {
    display: 'flex',
    alignItems: 'flex-start',
    justifyContent:
      'space-between',
    gap: 14,
    marginBottom: 12,
  }

const headerMainStyle:
  CSSProperties = {
    minWidth: 0,
  }

const headerTitleStyle:
  CSSProperties = {
    overflow: 'hidden',
    color: C.text,
    fontSize: 17,
    fontWeight: 900,
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  }

const headerDividerStyle:
  CSSProperties = {
    margin: '0 6px',
    color: C.textMuted,
  }

const headerClaimStyle:
  CSSProperties = {
    display: '-webkit-box',
    overflow: 'hidden',
    marginTop: 5,
    color: C.textSecondary,
    fontSize: 13,
    lineHeight: 1.5,
    WebkitBoxOrient: 'vertical',
    WebkitLineClamp: 2,
  }

const headerStatusStyle:
  CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 7,
    flexShrink: 0,
  }

const statusBadgeStyle:
  CSSProperties = {
    padding: '5px 9px',
    borderRadius: 999,
    fontSize: 12,
    fontWeight: 900,
  }

const dirtyDotStyle:
  CSSProperties = {
    width: 9,
    height: 9,
    borderRadius: 999,
    background: C.warning,
    boxShadow:
      '0 0 0 4px rgba(217,119,6,0.12)',
  }

const noticeStyle:
  CSSProperties = {
    marginBottom: 10,
    padding: '9px 11px',
    borderRadius: 9,
    border: '1px solid',
    fontSize: 13,
    lineHeight: 1.5,
  }
