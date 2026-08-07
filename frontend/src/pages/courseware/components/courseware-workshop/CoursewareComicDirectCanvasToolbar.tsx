/**
 * CoursewareComicDirectCanvasToolbar.tsx
 *
 * 漫画画布元素悬浮控制模块：
 *   - 第一行放置文字编辑、字号、字重、对齐和行距；
 *   - 第二行放置颜色、透明度、描边、样式和元素操作；
 *   - 工具栏稳定分成两行，避免横向超出可视屏幕；
 *   - 调整说话气泡整体描边时，主体与尾巴保持一致；
 *   - 显示四角缩放手柄。
 */

import type {
  CSSProperties,
  PointerEvent,
  SyntheticEvent,
} from 'react'
import type {
  CoursewareComicOverlayElement,
  CoursewareComicTextStyle,
} from '@/api/coursewares'
import type {
  ResizeCorner,
} from './CoursewareComicDirectCanvasSupport'
import {
  normalizeCoursewareComicBackgroundOpacity,
  normalizeCoursewareComicOutlineWidth,
  normalizeCoursewareComicTextAlign,
  normalizeCoursewareComicTextColor,
} from './CoursewareComicDirectCanvasElementStyles'
import CoursewareComicStyleGallery from './CoursewareComicStyleGallery'

interface FloatingToolbarProps {
  element: CoursewareComicOverlayElement
  below: boolean
  alignRight: boolean
  canDelete: boolean
  onBeginEditing: (elementID: string) => void
  onTextStyleChange: (
    elementID: string,
    patch: Partial<CoursewareComicTextStyle>,
  ) => void
  onCycleStyle: (
    elementID: string,
    styleID?: string,
  ) => void
  onAutoFit: (elementID: string) => void
  onDuplicate: (elementID: string) => void
  onDelete: (elementID: string) => void
}

const ALIGN_OPTIONS = [
  ['左', 'left'],
  ['中', 'center'],
  ['右', 'right'],
] as const

export function CoursewareComicFloatingToolbar({
  element,
  below,
  alignRight,
  canDelete,
  onBeginEditing,
  onTextStyleChange,
  onCycleStyle,
  onAutoFit,
  onDuplicate,
  onDelete,
}: FloatingToolbarProps) {
  const align = normalizeCoursewareComicTextAlign(
    element.text_style.align,
  )
  const textColor = normalizeCoursewareComicTextColor(
    element.text_style.color,
    '#111827',
  )
  const automaticColor =
    element.text_style.color_mode !== 'manual'
  const lineHeight = Math.round(
    Math.min(
      2.2,
      Math.max(
        1,
        element.text_style.line_height || 1.35,
      ),
    ) * 20,
  ) / 20
  const backgroundOpacity =
    normalizeCoursewareComicBackgroundOpacity(
      element.text_style.background_opacity,
    )
  const outlineWidth =
    normalizeCoursewareComicOutlineWidth(
      element.text_style.outline_width,
    )

  return (
    <div
      style={{
        ...toolbarStyle,
        left: alignRight ? undefined : 0,
        right: alignRight ? 0 : undefined,
        top: below ? 'calc(100% + 8px)' : undefined,
        bottom: below ? undefined : 'calc(100% + 8px)',
      }}
    >
      <div style={toolbarRowStyle}>
        <ToolButton
          label="编辑"
          onClick={() => onBeginEditing(element.id)}
        />
        <ToolButton
          label="A−"
          onClick={() =>
            onTextStyleChange(element.id, {
              font_size:
                element.text_style.font_size - 4,
            })
          }
        />
        <ToolButton
          label="A+"
          onClick={() =>
            onTextStyleChange(element.id, {
              font_size:
                element.text_style.font_size + 4,
            })
          }
        />
        <ToolButton
          label="B"
          active={
            element.text_style.font_weight >= 700
          }
          onClick={() =>
            onTextStyleChange(element.id, {
              font_weight:
                element.text_style.font_weight >= 700
                  ? 500
                  : 800,
            })
          }
        />

        {ALIGN_OPTIONS.map(([label, value]) => (
          <ToolButton
            key={value}
            label={label}
            active={align === value}
            onClick={() =>
              onTextStyleChange(element.id, {
                align: value,
              })
            }
          />
        ))}

        <label
          title="连续调整文字行距"
          style={sliderControlStyle}
          onPointerDown={stopPointerEvent}
          onClick={stopPointerEvent}
        >
          <span>
            行距{lineHeight.toFixed(2)}
          </span>
          <input
            type="range"
            min={1}
            max={2.2}
            step={0.05}
            value={lineHeight}
            aria-label="文字行距"
            onChange={event =>
              onTextStyleChange(element.id, {
                line_height:
                  Number(event.target.value),
              })
            }
            style={sliderInputStyle}
          />
        </label>
      </div>

      <div style={toolbarRowStyle}>
        <ToolButton
          label="自动色"
          active={automaticColor}
          onClick={() =>
            onTextStyleChange(element.id, {
              color_mode: 'auto',
            })
          }
        />

        <label
          title={
            automaticColor
              ? '选择颜色后切换为手动文字颜色'
              : '当前使用教师手动选择的文字颜色'
          }
          style={compactControlStyle}
          onPointerDown={stopPointerEvent}
          onClick={stopPointerEvent}
        >
          <span>字色</span>
          <input
            type="color"
            value={textColor}
            aria-label="文字颜色"
            onChange={event =>
              onTextStyleChange(element.id, {
                color:
                  normalizeCoursewareComicTextColor(
                    event.target.value,
                    textColor,
                  ),
                color_mode: 'manual',
              })
            }
            style={colorInputStyle}
          />
        </label>

        <label
          title="调整气泡或卡片背景透明度"
          style={sliderControlStyle}
          onPointerDown={stopPointerEvent}
          onClick={stopPointerEvent}
        >
          <span>
            透明
            {Math.round(backgroundOpacity * 100)}%
          </span>
          <input
            type="range"
            min={0.2}
            max={1}
            step={0.05}
            value={backgroundOpacity}
            aria-label="背景透明度"
            onChange={event =>
              onTextStyleChange(element.id, {
                background_opacity:
                  Number(event.target.value),
              })
            }
            style={sliderInputStyle}
          />
        </label>

        {element.type === 'speech_bubble' && (
          <label
            title="调整对话框和尾巴共用的整体描边粗细"
            style={sliderControlStyle}
            onPointerDown={stopPointerEvent}
            onClick={stopPointerEvent}
          >
            <span>
              描边{outlineWidth.toFixed(2)}
            </span>
            <input
              type="range"
              min={0.5}
              max={3}
              step={0.25}
              value={outlineWidth}
              aria-label="对话框描边宽度"
              onChange={event =>
                onTextStyleChange(element.id, {
                  outline_width:
                    Number(event.target.value),
                })
              }
              style={sliderInputStyle}
            />
          </label>
        )}

        <CoursewareComicStyleGallery
          element={element}
          onChange={styleID =>
            onCycleStyle(element.id, styleID)
          }
        />

        <ToolButton
          label="适配"
          onClick={() => onAutoFit(element.id)}
        />
        <ToolButton
          label="复制"
          onClick={() => onDuplicate(element.id)}
        />
        <ToolButton
          label="删除"
          danger
          disabled={!canDelete}
          onClick={() => onDelete(element.id)}
        />
      </div>
    </div>
  )
}

function stopPointerEvent(event: SyntheticEvent) {
  event.stopPropagation()
}

function ToolButton({
  label,
  active = false,
  danger = false,
  disabled = false,
  onClick,
}: {
  label: string
  active?: boolean
  danger?: boolean
  disabled?: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onPointerDown={stopPointerEvent}
      onClick={event => {
        event.stopPropagation()
        onClick()
      }}
      style={{
        ...toolButtonStyle,
        color: danger
          ? '#DC2626'
          : active
            ? '#FFFFFF'
            : '#334155',
        background: active
          ? '#7C3AED'
          : danger
            ? '#FEF2F2'
            : '#FFFFFF',
        opacity: disabled ? 0.4 : 1,
      }}
    >
      {label}
    </button>
  )
}

export function CoursewareComicResizeHandle({
  corner,
  onPointerDown,
}: {
  corner: ResizeCorner
  onPointerDown: (
    event: PointerEvent<HTMLDivElement>,
  ) => void
}) {
  return (
    <div
      style={{
        ...resizeHandleStyle,
        ...resizeHandlePositions[corner],
        cursor:
          corner === 'nw' || corner === 'se'
            ? 'nwse-resize'
            : 'nesw-resize',
      }}
      onPointerDown={onPointerDown}
    />
  )
}

const toolbarStyle: CSSProperties = {
  position: 'absolute',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'stretch',
  gap: 3,
  width: 'max-content',
  maxWidth: 'min(520px,calc(100vw - 20px))',
  padding: 4,
  boxSizing: 'border-box',
  borderRadius: 7,
  border: '1px solid #CBD5E1',
  background: 'rgba(248,250,252,0.98)',
  boxShadow: '0 8px 24px rgba(15,23,42,0.20)',
  zIndex: 500,
  whiteSpace: 'nowrap',
}

const toolbarRowStyle: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 3,
  minWidth: 0,
}

const toolButtonStyle: CSSProperties = {
  minWidth: 27,
  height: 29,
  padding: '0 6px',
  borderRadius: 5,
  border: '1px solid #E2E8F0',
  fontSize: 9,
  fontWeight: 800,
  cursor: 'pointer',
}

const compactControlStyle: CSSProperties = {
  height: 29,
  display: 'inline-flex',
  alignItems: 'center',
  gap: 4,
  padding: '0 5px',
  borderRadius: 5,
  border: '1px solid #E2E8F0',
  background: '#FFFFFF',
  color: '#334155',
  fontSize: 9,
  fontWeight: 800,
  cursor: 'pointer',
}

const sliderControlStyle: CSSProperties = {
  ...compactControlStyle,
  minWidth: 110,
}

const colorInputStyle: CSSProperties = {
  width: 18,
  height: 18,
  padding: 0,
  border: 'none',
  background: 'transparent',
  cursor: 'pointer',
}

const sliderInputStyle: CSSProperties = {
  width: 52,
  cursor: 'pointer',
}

const resizeHandleStyle: CSSProperties = {
  position: 'absolute',
  width: 10,
  height: 10,
  borderRadius: 999,
  border: '2px solid #FFFFFF',
  background: '#7C3AED',
  boxShadow: '0 1px 4px rgba(15,23,42,0.35)',
  zIndex: 400,
}

const resizeHandlePositions:
  Record<ResizeCorner, CSSProperties> = {
    nw: { left: -6, top: -6 },
    ne: { right: -6, top: -6 },
    sw: { left: -6, bottom: -6 },
    se: { right: -6, bottom: -6 },
  }
