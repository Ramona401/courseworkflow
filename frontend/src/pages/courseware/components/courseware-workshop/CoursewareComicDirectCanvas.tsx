/**
 * CoursewareComicDirectCanvas.tsx
 *
 * 知识点漫画直接编辑画布主组件：
 *   - 管理选择、拖动、缩放和指针捕获；
 *   - 管理气泡尾巴连接点与指向点拖动；
 *   - 使用ref保存最后一次交互结果；
 *   - 拖动和四角缩放严格展示教师的实时尺寸，不调用自动适配；
 *   - 使用项目确认后的真实画幅；
 *   - 元素、工具栏和问题卡编辑均由独立组件承担。
 */

import {
  useMemo,
  useRef,
  useState,
} from 'react'
import type {
  PointerEvent,
} from 'react'
import type {
  CoursewareComicOverlayElement,
} from '@/api/coursewares'
import {
  clampCanvasValue,
  CoursewareComicCanvasElement,
  CoursewareComicQuestionPopover,
  resolveResizePatch,
} from './CoursewareComicDirectCanvasControls'
import type {
  LayoutPatch,
  PointerInteraction,
  PointerInteractionMode,
  ResizeCorner,
} from './CoursewareComicDirectCanvasSupport'
import {
  applyCoursewareComicTailPatch,
  projectCoursewareComicTailOrigin,
} from './coursewareComicTailEditing'
import {
  directCanvasEmptyIconStyle,
  directCanvasEmptyStyle,
  directCanvasImageStyle,
  directCanvasInstructionStyle,
  directCanvasSelectedHintStyle,
  directCanvasStyle,
  directCanvasWorkspaceStyle,
  resolveCoursewareComicCanvasAspectRatio,
} from './CoursewareComicDirectCanvasSupport'
import type {
  CoursewareComicDirectCanvasProps,
} from './CoursewareComicDirectCanvasSupport'

function applyTransientLayoutPatch(
  element: CoursewareComicOverlayElement,
  patch: LayoutPatch,
): CoursewareComicOverlayElement {
  const positioned: CoursewareComicOverlayElement = {
    ...element,
    x: patch.x ?? element.x,
    y: patch.y ?? element.y,
    width: patch.width ?? element.width,
    height: patch.height ?? element.height,
  }

  return applyCoursewareComicTailPatch(
    positioned,
    patch,
  )
}

export default function CoursewareComicDirectCanvas({
  panel,
  aspectRatio,
  overlayDocument,
  disabled,
  selectedElementID,
  editingElementID,
  onSelectElement,
  onBeginEditing,
  onEndEditing,
  onContentChange,
  onQuestionTextChange,
  onQuestionOptionsChange,
  onQuestionAnswerChange,
  onLayoutChange,
  onTextStyleChange,
  onCycleStyle,
  onAutoFit,
  onDuplicate,
  onDelete,
  onKeyDown,
}: CoursewareComicDirectCanvasProps) {
  const canvasRef = useRef<HTMLDivElement | null>(null)
  const interactionRef = useRef<PointerInteraction | null>(null)
  const transientRef = useRef<{
    elementID: string
    patch: LayoutPatch
  } | null>(null)
  const [transient, setTransient] = useState<{
    elementID: string
    patch: LayoutPatch
  } | null>(null)

  const selectedElement = useMemo(
    () =>
      overlayDocument.elements.find(
        element => element.id === selectedElementID,
      ) || null,
    [
      overlayDocument.elements,
      selectedElementID,
    ],
  )

  const editingElement = useMemo(
    () =>
      overlayDocument.elements.find(
        element => element.id === editingElementID,
      ) || null,
    [
      overlayDocument.elements,
      editingElementID,
    ],
  )

  const renderedElements = overlayDocument.elements.map(element =>
    transient?.elementID === element.id
      ? applyTransientLayoutPatch(
          element,
          transient.patch,
        )
      : element,
  )

  const setTransientPatch = (
    value: {
      elementID: string
      patch: LayoutPatch
    } | null,
  ) => {
    transientRef.current = value
    setTransient(value)
  }

  const beginPointerInteraction = (
    event: PointerEvent<HTMLDivElement>,
    element: CoursewareComicOverlayElement,
    mode: PointerInteractionMode,
    corner?: ResizeCorner,
  ) => {
    if (disabled || editingElementID) {
      return
    }

    event.preventDefault()
    event.stopPropagation()
    onSelectElement(element.id)

    try {
      canvasRef.current?.setPointerCapture(event.pointerId)
    } catch {
      // 不支持指针捕获时仍保留当前交互状态。
    }

    interactionRef.current = {
      elementID: element.id,
      mode,
      corner,
      pointerID: event.pointerId,
      startClientX: event.clientX,
      startClientY: event.clientY,
      startX: element.x,
      startY: element.y,
      startWidth: element.width,
      startHeight: element.height,
    }
  }

  const handlePointerMove = (
    event: PointerEvent<HTMLDivElement>,
  ) => {
    const interaction = interactionRef.current
    const canvas = canvasRef.current

    if (
      !interaction ||
      !canvas ||
      interaction.pointerID !== event.pointerId
    ) {
      return
    }

    const bounds = canvas.getBoundingClientRect()

    if (bounds.width <= 0 || bounds.height <= 0) {
      return
    }

    const pointerX = clampCanvasValue(
      (event.clientX - bounds.left) / bounds.width,
      0,
      1,
    )
    const pointerY = clampCanvasValue(
      (event.clientY - bounds.top) / bounds.height,
      0,
      1,
    )
    const deltaX =
      (event.clientX - interaction.startClientX) /
      bounds.width
    const deltaY =
      (event.clientY - interaction.startClientY) /
      bounds.height

    let patch: LayoutPatch

    switch (interaction.mode) {
    case 'tail_target':
      patch = {
        tail_type: 'manual',
        tail_target_x: pointerX,
        tail_target_y: pointerY,
      }
      break

    case 'tail_origin': {
      const localX =
        (pointerX - interaction.startX) /
        Math.max(0.001, interaction.startWidth)
      const localY =
        (pointerY - interaction.startY) /
        Math.max(0.001, interaction.startHeight)
      const origin = projectCoursewareComicTailOrigin(
        localX,
        localY,
      )

      patch = {
        tail_type: 'manual',
        tail_origin_x: origin.x,
        tail_origin_y: origin.y,
      }
      break
    }

    case 'drag':
      patch = {
        x: clampCanvasValue(
          interaction.startX + deltaX,
          0,
          1 - interaction.startWidth,
        ),
        y: clampCanvasValue(
          interaction.startY + deltaY,
          0,
          1 - interaction.startHeight,
        ),
      }
      break

    default:
      patch = resolveResizePatch(
        interaction,
        deltaX,
        deltaY,
      )
      break
    }

    setTransientPatch({
      elementID: interaction.elementID,
      patch,
    })
  }

  const finishPointerInteraction = (
    event: PointerEvent<HTMLDivElement>,
  ) => {
    const interaction = interactionRef.current

    if (
      !interaction ||
      interaction.pointerID !== event.pointerId
    ) {
      return
    }

    const latest = transientRef.current

    if (
      latest &&
      latest.elementID === interaction.elementID
    ) {
      onLayoutChange(
        interaction.elementID,
        latest.patch,
      )
    }

    try {
      if (
        canvasRef.current?.hasPointerCapture(
          event.pointerId,
        )
      ) {
        canvasRef.current.releasePointerCapture(
          event.pointerId,
        )
      }
    } catch {
      // 浏览器已自动释放时无需重复处理。
    }

    interactionRef.current = null
    setTransientPatch(null)
  }

  return (
    <div style={directCanvasWorkspaceStyle}>
      <div style={directCanvasInstructionStyle}>
        单击选中 · 拖动排版 ·
        四角缩放 · 双击编辑文字 ·
        拖动气泡连接点和指向点
      </div>

      <div
        ref={canvasRef}
        style={{
          ...directCanvasStyle,
          aspectRatio:
            resolveCoursewareComicCanvasAspectRatio(
              aspectRatio,
            ),
        }}
        onPointerMove={handlePointerMove}
        onPointerUp={finishPointerInteraction}
        onPointerCancel={finishPointerInteraction}
        onPointerDown={event => {
          if (event.target === event.currentTarget) {
            onSelectElement('')
            onEndEditing()
          }
        }}
      >
        {panel.current_asset_url ? (
          <img
            src={panel.current_asset_url}
            alt={`第${panel.panel_no}格漫画`}
            draggable={false}
            style={directCanvasImageStyle}
          />
        ) : (
          <div style={directCanvasEmptyStyle}>
            <div>
              <div style={directCanvasEmptyIconStyle}>
                🖼️
              </div>
              尚未生成无文字底图
            </div>
          </div>
        )}

        {renderedElements.map(element => (
          <CoursewareComicCanvasElement
            key={element.id}
            element={element}
            selected={
              selectedElementID === element.id
            }
            editing={
              editingElementID === element.id
            }
            disabled={disabled}
            canDelete={
              overlayDocument.elements.length > 1
            }
            onSelectElement={onSelectElement}
            onBeginEditing={onBeginEditing}
            onEndEditing={onEndEditing}
            onContentChange={onContentChange}
            onLayoutChange={onLayoutChange}
            onTextStyleChange={onTextStyleChange}
            onCycleStyle={onCycleStyle}
            onAutoFit={onAutoFit}
            onDuplicate={onDuplicate}
            onDelete={onDelete}
            onBeginPointerInteraction={
              beginPointerInteraction
            }
            onKeyDown={onKeyDown}
          />
        ))}

        {editingElement?.type === 'question_card' &&
          editingElement.question && (
          <CoursewareComicQuestionPopover
            element={editingElement}
            disabled={disabled}
            onClose={onEndEditing}
            onTextChange={onQuestionTextChange}
            onOptionsChange={onQuestionOptionsChange}
            onAnswerChange={onQuestionAnswerChange}
            onKeyDown={onKeyDown}
          />
        )}
      </div>

      {selectedElement && (
        <div style={directCanvasSelectedHintStyle}>
          已选择一个画布元素
        </div>
      )}
    </div>
  )
}
