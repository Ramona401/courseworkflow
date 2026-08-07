/**
 * coursewareComicExportCanvasLayout.ts
 *
 * 知识点漫画导出Canvas的尺寸、底图和公共装饰：
 *   - 计算JPG长图与A4横向PDF页面布局；
 *   - 绘制标题、TE-DNA标记、底图、圆角阴影和格号；
 *   - 不处理覆盖层文字，不访问接口。
 */

import type {
  CoursewareComicWorkflowProject,
} from '@/api/coursewares'

import {
  resolveCoursewareComicExportAspect,
} from './coursewareComicExportMarkup'

import {
  coursewareComicRoundedRectPath,
} from './coursewareComicExportCanvasText'

export const COURSEWARE_COMIC_JPG_WIDTH = 2400
export const COURSEWARE_COMIC_PDF_WIDTH = 2480
export const COURSEWARE_COMIC_PDF_HEIGHT = 1754

export interface CoursewareComicLongCanvasLayout {
  width: number
  height: number
  margin: number
  headerHeight: number
  columns: number
  columnGap: number
  rowGap: number
  panelWidth: number
  panelHeight: number
}

export interface CoursewareComicPDFCanvasLayout {
  columns: number
  rows: number
  margin: number
  headerHeight: number
  columnGap: number
  rowGap: number
  panelWidth: number
  panelHeight: number
  contentStartX: number
  contentStartY: number
}

export function createCoursewareComicLongLayout(
  project: CoursewareComicWorkflowProject,
  panelCount: number,
): CoursewareComicLongCanvasLayout {
  const margin = 96
  const headerHeight = 168
  const columns = 2
  const columnGap = 48
  const rowGap = 48
  const aspect =
    resolveCoursewareComicExportAspect(
      project.workflow.aspect_ratio,
    )
  const panelWidth = Math.floor(
    (
      COURSEWARE_COMIC_JPG_WIDTH -
      margin * 2 -
      columnGap
    ) / columns,
  )
  const panelHeight = Math.max(
    1,
    Math.round(
      panelWidth *
      aspect.height /
      aspect.width,
    ),
  )
  const rows = Math.ceil(panelCount / columns)
  const height =
    margin +
    headerHeight +
    rows * panelHeight +
    Math.max(0, rows - 1) * rowGap +
    margin

  return {
    width: COURSEWARE_COMIC_JPG_WIDTH,
    height,
    margin,
    headerHeight,
    columns,
    columnGap,
    rowGap,
    panelWidth,
    panelHeight,
  }
}

export function createCoursewareComicPDFLayout(
  project: CoursewareComicWorkflowProject,
  panelsPerPage: number,
): CoursewareComicPDFCanvasLayout {
  const columns = 2
  const rows = panelsPerPage === 2 ? 1 : 2
  const margin = 140
  const headerHeight = 190
  const columnGap = 70
  const rowGap = 60
  const aspect =
    resolveCoursewareComicExportAspect(
      project.workflow.aspect_ratio,
    )
  const maximumPanelWidth =
    (
      COURSEWARE_COMIC_PDF_WIDTH -
      margin * 2 -
      columnGap
    ) / columns
  const maximumPanelHeight =
    (
      COURSEWARE_COMIC_PDF_HEIGHT -
      margin * 2 -
      headerHeight -
      rowGap * (rows - 1)
    ) / rows
  const panelWidth = Math.floor(
    Math.min(
      maximumPanelWidth,
      maximumPanelHeight *
      aspect.width /
      aspect.height,
    ),
  )
  const panelHeight = Math.floor(
    panelWidth *
    aspect.height /
    aspect.width,
  )
  const contentWidth =
    columns * panelWidth + columnGap
  const contentHeight =
    rows * panelHeight +
    Math.max(0, rows - 1) * rowGap

  return {
    columns,
    rows,
    margin,
    headerHeight,
    columnGap,
    rowGap,
    panelWidth,
    panelHeight,
    contentStartX:
      (COURSEWARE_COMIC_PDF_WIDTH - contentWidth) / 2,
    contentStartY:
      margin +
      headerHeight +
      Math.max(
        0,
        (
          COURSEWARE_COMIC_PDF_HEIGHT -
          margin * 2 -
          headerHeight -
          contentHeight
        ) / 2,
      ),
  }
}

export function drawCoursewareComicCanvasHeader(
  context: CanvasRenderingContext2D,
  project: CoursewareComicWorkflowProject,
  panelCount: number,
  x: number,
  y: number,
  width: number,
  titleSize: number,
  metaSize: number,
  pageLabel = '',
): void {
  context.save()
  context.fillStyle = '#0F172A'
  context.font =
    `900 ${titleSize}px Arial,Microsoft YaHei,PingFang SC,sans-serif`
  context.textAlign = 'left'
  context.textBaseline = 'top'
  context.fillText(
    project.title || '知识点漫画',
    x,
    y,
    Math.max(200, width - 420),
  )

  context.fillStyle = '#64748B'
  context.font =
    `600 ${metaSize}px Arial,Microsoft YaHei,PingFang SC,sans-serif`
  context.fillText(
    `${project.subject} · ${project.grade} · ${panelCount}格漫画` +
    (pageLabel ? ` · ${pageLabel}` : ''),
    x,
    y + titleSize + 22,
    width - 300,
  )

  const badgeWidth = 180
  const badgeHeight = 62
  const badgeX = x + width - badgeWidth
  const badgeY = y + 4

  coursewareComicRoundedRectPath(
    context,
    badgeX,
    badgeY,
    badgeWidth,
    badgeHeight,
    badgeHeight / 2,
  )
  context.fillStyle = '#F3E8FF'
  context.fill()
  context.fillStyle = '#6D28D9'
  context.font =
    '900 26px Arial,Microsoft YaHei,sans-serif'
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText(
    'TE-DNA',
    badgeX + badgeWidth / 2,
    badgeY + badgeHeight / 2,
  )
  context.restore()
}

export function drawCoursewareComicPanelBase(
  context: CanvasRenderingContext2D,
  image: HTMLImageElement,
  x: number,
  y: number,
  width: number,
  height: number,
  scaleBoost: number,
): void {
  const radius = 22 * scaleBoost

  context.save()
  context.shadowColor = 'rgba(15,23,42,0.14)'
  context.shadowBlur = 30 * scaleBoost
  context.shadowOffsetY = 12 * scaleBoost
  coursewareComicRoundedRectPath(
    context,
    x,
    y,
    width,
    height,
    radius,
  )
  context.fillStyle = '#E2E8F0'
  context.fill()
  context.clip()
  drawImageCover(context, image, x, y, width, height)
  context.restore()
}

export function drawCoursewareComicPanelNumber(
  context: CanvasRenderingContext2D,
  panelNumber: number,
  x: number,
  bottomY: number,
  scaleBoost: number,
): void {
  const height = 50 * scaleBoost
  const width = Math.max(height, 68 * scaleBoost)
  const top = bottomY - height

  coursewareComicRoundedRectPath(
    context,
    x,
    top,
    width,
    height,
    height / 2,
  )
  context.fillStyle = 'rgba(15,23,42,0.82)'
  context.fill()
  context.fillStyle = '#FFFFFF'
  context.font =
    `900 ${26 * scaleBoost}px Arial,Microsoft YaHei,sans-serif`
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText(
    String(panelNumber),
    x + width / 2,
    top + height / 2,
  )
}

export function createCoursewareComicCanvas(
  width: number,
  height: number,
): HTMLCanvasElement {
  const canvas = document.createElement('canvas')
  canvas.width = Math.max(1, Math.trunc(width))
  canvas.height = Math.max(1, Math.trunc(height))
  return canvas
}

export function requireCoursewareComicCanvasContext(
  canvas: HTMLCanvasElement,
): CanvasRenderingContext2D {
  const context = canvas.getContext('2d')

  if (!context) {
    throw new Error(
      '当前浏览器无法创建漫画导出画布。',
    )
  }

  context.imageSmoothingEnabled = true
  context.imageSmoothingQuality = 'high'
  return context
}

export function fillCoursewareComicCanvasWhite(
  context: CanvasRenderingContext2D,
  width: number,
  height: number,
): void {
  context.fillStyle = '#FFFFFF'
  context.fillRect(0, 0, width, height)
}

function drawImageCover(
  context: CanvasRenderingContext2D,
  image: HTMLImageElement,
  x: number,
  y: number,
  width: number,
  height: number,
): void {
  const imageRatio =
    image.naturalWidth / image.naturalHeight
  const boxRatio = width / height
  let sourceX = 0
  let sourceY = 0
  let sourceWidth = image.naturalWidth
  let sourceHeight = image.naturalHeight

  if (imageRatio > boxRatio) {
    sourceWidth = image.naturalHeight * boxRatio
    sourceX =
      (image.naturalWidth - sourceWidth) / 2
  } else {
    sourceHeight = image.naturalWidth / boxRatio
    sourceY =
      (image.naturalHeight - sourceHeight) / 2
  }

  context.drawImage(
    image,
    sourceX,
    sourceY,
    sourceWidth,
    sourceHeight,
    x,
    y,
    width,
    height,
  )
}
