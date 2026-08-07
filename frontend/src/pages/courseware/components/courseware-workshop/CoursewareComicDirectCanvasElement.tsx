/**
 * CoursewareComicDirectCanvasElement.tsx
 *
 * 漫画画布元素主组件：
 *   - 说话气泡主体与尾巴使用一个SVG闭合路径；
 *   - 整体统一填充、透明度和描边，不再出现连接接缝；
 *   - 双击后在元素内部直接编辑普通文字；
 *   - 选中后显示尾巴控制点、工具栏和四角缩放手柄。
 */

import {
  coursewareComicElementLabel,
  overlayElementDisplayText,
} from './coursewareComicEditorDraft'

import {
  directCanvasDisplayTextStyle,
  directCanvasElementBaseStyle,
  directCanvasInlineEditorStyle,
  normalizeCoursewareComicTextAlign,
  resolveCoursewareComicBubbleTailGeometry,
  resolveCoursewareComicElementVisual,
  resolveCoursewareComicTextContentStyle,
} from './CoursewareComicDirectCanvasElementStyles'

import {
  CoursewareComicFloatingToolbar,
  CoursewareComicResizeHandle,
} from './CoursewareComicDirectCanvasToolbar'

import {
  resolveCoursewareComicTailMode,
  resolveCoursewareComicTailOrigin,
  resolveCoursewareComicTailTarget,
} from './coursewareComicTailEditing'

import type {
  CoursewareComicCanvasElementProps,
  ResizeCorner,
} from './CoursewareComicDirectCanvasSupport'

export function CoursewareComicCanvasElement({
  element,
  selected,
  editing,
  disabled,
  canDelete,
  onSelectElement,
  onBeginEditing,
  onEndEditing,
  onContentChange,
  onLayoutChange,
  onTextStyleChange,
  onCycleStyle,
  onAutoFit,
  onDuplicate,
  onDelete,
  onBeginPointerInteraction,
  onKeyDown,
}: CoursewareComicCanvasElementProps) {
  const visual = resolveCoursewareComicElementVisual(element)
  const textContentStyle = resolveCoursewareComicTextContentStyle(element)
  const bubbleShape = resolveCoursewareComicBubbleTailGeometry(element)
  const tailOrigin = resolveCoursewareComicTailOrigin(element)
  const tailTarget = resolveCoursewareComicTailTarget(element)
  const tailMode = resolveCoursewareComicTailMode(element)
  const question = element.type === 'question_card'
  const text = overlayElementDisplayText(element)
  const toolbarBelow = element.y < 0.15
  const toolbarRight = element.x + element.width > 0.72
  const localTarget = tailTarget
    ? {
        x: (tailTarget.x - element.x) / Math.max(0.001, element.width),
        y: (tailTarget.y - element.y) / Math.max(0.001, element.height),
      }
    : null

  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={coursewareComicElementLabel(element)}
      style={{
        ...directCanvasElementBaseStyle,
        ...visual,
        left: `${element.x * 100}%`,
        top: `${element.y * 100}%`,
        width: `${element.width * 100}%`,
        height: `${element.height * 100}%`,
        transform: `rotate(${element.rotation || 0}deg)`,
        zIndex: selected
          ? Math.max(100, element.z_index)
          : element.z_index,
        outline: selected ? '2px solid #7C3AED' : 'none',
        boxShadow: selected
          ? '0 0 0 3px rgba(124,58,237,0.20),0 8px 24px rgba(15,23,42,0.22)'
          : visual.boxShadow,
        cursor: disabled
          ? 'default'
          : editing
            ? 'text'
            : 'move',
      }}
      onClick={event => {
        event.stopPropagation()
        onSelectElement(element.id)
      }}
      onDoubleClick={event => {
        event.stopPropagation()
        if (!disabled) {
          onBeginEditing(element.id)
        }
      }}
      onPointerDown={event =>
        onBeginPointerInteraction(event, element, 'drag')
      }
      onKeyDown={event => {
        if (event.key === 'Enter' && !disabled) {
          onBeginEditing(element.id)
        }
      }}
    >
      {bubbleShape && (
        <svg
          viewBox="0 0 100 100"
          preserveAspectRatio="none"
          aria-hidden="true"
          style={{
            position: 'absolute',
            inset: 0,
            zIndex: 0,
            width: '100%',
            height: '100%',
            overflow: 'visible',
            pointerEvents: 'none',
            filter: bubbleShape.filter,
          }}
        >
          <path
            d={bubbleShape.shapePath}
            fill={bubbleShape.fill}
            stroke={bubbleShape.stroke}
            strokeWidth={bubbleShape.strokeWidth}
            strokeLinejoin="round"
            strokeLinecap="round"
            vectorEffect="non-scaling-stroke"
          />
        </svg>
      )}

      {editing && !question ? (
        <textarea
          autoFocus
          value={element.content}
          onChange={event =>
            onContentChange(element.id, event.target.value)
          }
          onBlur={onEndEditing}
          onPointerDown={event => event.stopPropagation()}
          onClick={event => event.stopPropagation()}
          onKeyDown={event => {
            if (onKeyDown(event)) {
              return
            }

            if (event.key === 'Escape') {
              event.preventDefault()
              onEndEditing()
            }

            if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
              event.preventDefault()
              onEndEditing()
            }
          }}
          style={{
            ...directCanvasInlineEditorStyle,
            ...textContentStyle,
            color: visual.color,
            textAlign: normalizeCoursewareComicTextAlign(
              element.text_style.align,
            ),
            fontFamily: element.text_style.font_family,
            fontWeight: element.text_style.font_weight,
            lineHeight: element.text_style.line_height,
          }}
        />
      ) : (
        <div
          style={{
            ...directCanvasDisplayTextStyle,
            ...textContentStyle,
          }}
        >
          {text}
        </div>
      )}

      {selected &&
        !disabled &&
        element.type === 'speech_bubble' &&
        tailOrigin &&
        localTarget && (
        <>
          <div
            title="拖动：调整尾巴与气泡外轮廓的连接点"
            style={{
              ...tailHandleStyle,
              left: `${tailOrigin.x * 100}%`,
              top: `${tailOrigin.y * 100}%`,
              borderColor: '#7C3AED',
              background: '#FFFFFF',
              cursor: 'grab',
            }}
            onPointerDown={event =>
              onBeginPointerInteraction(
                event,
                element,
                'tail_origin',
              )
            }
          />

          <div
            title="拖动：调整尾巴指向人物的位置"
            style={{
              ...tailHandleStyle,
              left: `${localTarget.x * 100}%`,
              top: `${localTarget.y * 100}%`,
              borderColor: '#FFFFFF',
              background: '#F97316',
              boxShadow:
                '0 0 0 2px #F97316,0 2px 7px rgba(15,23,42,0.32)',
              cursor: 'crosshair',
            }}
            onPointerDown={event =>
              onBeginPointerInteraction(
                event,
                element,
                'tail_target',
              )
            }
          />

          <button
            type="button"
            title={tailMode === 'manual'
              ? '点击恢复自动连接点；人物目标点保持不变'
              : '当前自动指向人物；拖动任一控制点后切换为手动'}
            onPointerDown={event => event.stopPropagation()}
            onClick={event => {
              event.stopPropagation()
              onLayoutChange(element.id, {
                tail_type: tailMode === 'manual' ? 'auto' : 'manual',
              })
            }}
            style={tailModeButtonStyle}
          >
            尾巴：{tailMode === 'manual' ? '手动' : '自动'}
          </button>
        </>
      )}

      {selected && !disabled && (
        <>
          <CoursewareComicFloatingToolbar
            element={element}
            below={toolbarBelow}
            alignRight={toolbarRight}
            canDelete={canDelete}
            onBeginEditing={onBeginEditing}
            onTextStyleChange={onTextStyleChange}
            onCycleStyle={onCycleStyle}
            onAutoFit={onAutoFit}
            onDuplicate={onDuplicate}
            onDelete={onDelete}
          />

          {(['nw', 'ne', 'sw', 'se'] as ResizeCorner[]).map(corner => (
            <CoursewareComicResizeHandle
              key={corner}
              corner={corner}
              onPointerDown={event =>
                onBeginPointerInteraction(
                  event,
                  element,
                  'resize',
                  corner,
                )
              }
            />
          ))}
        </>
      )}
    </div>
  )
}

const tailHandleStyle: React.CSSProperties = {
  position: 'absolute',
  width: 12,
  height: 12,
  boxSizing: 'border-box',
  border: '3px solid',
  borderRadius: 999,
  transform: 'translate(-50%,-50%)',
  zIndex: 620,
  touchAction: 'none',
}

const tailModeButtonStyle: React.CSSProperties = {
  position: 'absolute',
  right: 3,
  bottom: 3,
  zIndex: 610,
  padding: '3px 6px',
  borderRadius: 999,
  border: '1px solid rgba(124,58,237,0.45)',
  background: 'rgba(255,255,255,0.94)',
  color: '#6D28D9',
  fontSize: 8,
  fontWeight: 800,
  cursor: 'pointer',
}
