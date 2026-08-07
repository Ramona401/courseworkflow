/**
 * coursewareComicExportOverlay.ts
 *
 * 知识点漫画Canvas覆盖层事实渲染：
 *   - 复用编辑器正式视觉、字号、内边距和气泡闭合路径；
 *   - 使用屏幕外DOM取得浏览器真实换行与垂直居中位置；
 *   - 气泡保持教师保存的原尺寸，导出时只在框内自动适配文字；
 *   - 适配只影响下载成品，不修改项目数据或编辑器排版；
 *   - JPG与PDF共享本模块，保证两种导出一致。
 */

import type {
  CSSProperties,
} from 'react'

import type {
  CoursewareComicOverlayDocument,
  CoursewareComicOverlayElement,
} from '@/api/coursewares'

import {
  overlayElementDisplayText,
} from './coursewareComicEditorDraft'

import {
  normalizeCoursewareComicTextAlign,
  resolveCoursewareComicBubbleTailGeometry,
  resolveCoursewareComicElementVisual,
  resolveCoursewareComicTextContentStyle,
} from './CoursewareComicDirectCanvasElementStyles'

import {
  applyCoursewareComicCanvasShadow,
  clearCoursewareComicCanvasShadow,
  coursewareComicRoundedRectPath,
  normalizeCoursewareComicCanvasFont,
  parseCoursewareComicCanvasBorder,
  parseCoursewareComicCanvasPadding,
  parseCoursewareComicCanvasPixel,
  resolveCoursewareComicCanvasRadius,
} from './coursewareComicExportCanvasText'

import {
  drawCoursewareComicMeasuredText,
} from './coursewareComicExportDOMMeasure'

import {
  fitCoursewareComicExportText,
} from './coursewareComicExportTextFit'

interface ExportPanelRect {
  x: number
  y: number
  width: number
  height: number
}

const PREVIEW_REFERENCE_WIDTH = 760

export async function drawCoursewareComicExportOverlay(
  context: CanvasRenderingContext2D,
  documentValue: CoursewareComicOverlayDocument,
  panelRect: ExportPanelRect,
  _panelNumber: number,
): Promise<void> {
  const displayScale =
    panelRect.width /
    PREVIEW_REFERENCE_WIDTH

  const ordered =
    [...(documentValue.elements || [])]
      .sort(
        (left, right) =>
          left.z_index -
          right.z_index,
      )

  for (const element of ordered) {
    drawOverlayElement(
      context,
      element,
      panelRect,
      displayScale,
    )
  }
}

function drawOverlayElement(
  context: CanvasRenderingContext2D,
  element: CoursewareComicOverlayElement,
  panelRect: ExportPanelRect,
  displayScale: number,
): void {
  const width = Math.max(
    1,
    element.width * panelRect.width,
  )
  const height = Math.max(
    1,
    element.height * panelRect.height,
  )
  const left =
    panelRect.x +
    element.x * panelRect.width
  const top =
    panelRect.y +
    element.y * panelRect.height
  const visual =
    resolveCoursewareComicElementVisual(element)
  const textStyle =
    resolveCoursewareComicTextContentStyle(element)
  const bubble =
    resolveCoursewareComicBubbleTailGeometry(element)

  context.save()
  context.translate(
    left + width / 2,
    top + height / 2,
  )
  context.rotate(
    (element.rotation || 0) *
    Math.PI /
    180,
  )
  context.translate(
    -width / 2,
    -height / 2,
  )

  if (bubble) {
    drawSpeechBubble(
      context,
      bubble.shapePath,
      bubble.fill,
      bubble.stroke,
      bubble.strokeWidth * displayScale,
      bubble.filter,
      width,
      height,
    )
  } else {
    drawStandardElement(
      context,
      visual,
      width,
      height,
      displayScale,
    )
  }

  const padding =
    parseCoursewareComicCanvasPadding(
      textStyle.padding,
      displayScale,
    )
  const fontSize =
    parseCoursewareComicCanvasPixel(
      visual.fontSize,
      10,
    ) *
    displayScale
  const lineHeight =
    typeof visual.lineHeight === 'number'
      ? visual.lineHeight
      : (
          Number.parseFloat(
            String(
              visual.lineHeight ||
              1.35,
            ),
          ) ||
          1.35
        )

  const fitted =
    fitCoursewareComicExportText({
      text: overlayElementDisplayText(element),
      width,
      height,
      paddingVertical: padding.vertical,
      paddingHorizontal: padding.horizontal,
      fontSize,
      fontWeight: String(visual.fontWeight || 600),
      fontFamily: normalizeCoursewareComicCanvasFont(
        visual.fontFamily,
      ),
      lineHeight,
      color: String(visual.color || '#111827'),
      align: normalizeCoursewareComicTextAlign(
        element.text_style.align,
      ),
      verticalAlign:
        element.type === 'question_card'
          ? 'top'
          : 'center',
    })

  context.save()
  context.beginPath()
  context.rect(0, 0, width, height)
  context.clip()

  drawCoursewareComicMeasuredText(
    context,
    fitted.layout,
    fitted.options,
  )

  context.restore()
  context.restore()
}

function drawSpeechBubble(
  context: CanvasRenderingContext2D,
  pathText: string,
  fill: string,
  stroke: string,
  strokeWidth: number,
  filter: string,
  width: number,
  height: number,
): void {
  let path: Path2D

  try {
    path = new Path2D(pathText)
  } catch {
    drawFallbackBubble(
      context,
      fill,
      stroke,
      strokeWidth,
      width,
      height,
    )
    return
  }

  applyCoursewareComicCanvasShadow(
    context,
    filter,
    Math.max(
      0.5,
      width /
      PREVIEW_REFERENCE_WIDTH,
    ),
  )

  const scaleX = width / 100
  const scaleY = height / 100
  const averageScale = Math.max(
    0.001,
    (scaleX + scaleY) / 2,
  )

  context.save()
  context.scale(scaleX, scaleY)
  context.fillStyle = fill
  context.fill(path)
  context.shadowColor = 'transparent'
  context.strokeStyle = stroke
  context.lineWidth =
    strokeWidth /
    averageScale
  context.lineJoin = 'round'
  context.lineCap = 'round'
  context.stroke(path)
  context.restore()

  clearCoursewareComicCanvasShadow(context)
}

function drawStandardElement(
  context: CanvasRenderingContext2D,
  visual: CSSProperties,
  width: number,
  height: number,
  displayScale: number,
): void {
  const background =
    String(
      visual.background ||
      'transparent',
    )
  const border =
    parseCoursewareComicCanvasBorder(
      visual.border,
      displayScale,
    )
  const radius =
    resolveCoursewareComicCanvasRadius(
      visual.borderRadius,
      width,
      height,
      displayScale,
    )

  applyCoursewareComicCanvasShadow(
    context,
    visual.boxShadow,
    displayScale,
  )
  coursewareComicRoundedRectPath(
    context,
    0,
    0,
    width,
    height,
    radius,
  )

  if (
    background !== 'transparent' &&
    background !== 'none'
  ) {
    context.fillStyle = background
    context.fill()
  }

  clearCoursewareComicCanvasShadow(context)

  if (
    border.width > 0 &&
    border.color !== 'transparent'
  ) {
    context.strokeStyle = border.color
    context.lineWidth = border.width
    context.stroke()
  }
}

function drawFallbackBubble(
  context: CanvasRenderingContext2D,
  fill: string,
  stroke: string,
  strokeWidth: number,
  width: number,
  height: number,
): void {
  coursewareComicRoundedRectPath(
    context,
    0,
    0,
    width,
    height,
    Math.min(28, height / 3),
  )
  context.fillStyle = fill
  context.fill()
  context.strokeStyle = stroke
  context.lineWidth = strokeWidth
  context.stroke()
}
