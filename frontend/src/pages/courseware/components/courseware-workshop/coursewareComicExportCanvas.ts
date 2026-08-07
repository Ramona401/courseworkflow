/**
 * coursewareComicExportCanvas.ts
 *
 * 知识点漫画JPG/PDF共用Canvas装配：
 *   - 组织底图、浏览器真实排版覆盖层和格号的绘制顺序；
 *   - 教师设计意图不进入学生版导出；
 *   - JPG使用2列连续长图；
 *   - PDF按A4横向比例生成扁平化页面图片。
 */

import type {
  CoursewareComicPanel,
  CoursewareComicWorkflowProject,
} from '@/api/coursewares'

import type {
  PreparedCoursewareComicPanel,
} from './coursewareComicExportMarkup'

import {
  loadCoursewareComicExportImage,
  resolveCoursewareComicExportAspect,
} from './coursewareComicExportMarkup'

import {
  drawCoursewareComicExportOverlay,
} from './coursewareComicExportOverlay'

import {
  COURSEWARE_COMIC_PDF_HEIGHT,
  COURSEWARE_COMIC_PDF_WIDTH,
  createCoursewareComicCanvas,
  createCoursewareComicLongLayout,
  createCoursewareComicPDFLayout,
  drawCoursewareComicCanvasHeader,
  drawCoursewareComicPanelBase,
  drawCoursewareComicPanelNumber,
  fillCoursewareComicCanvasWhite,
  requireCoursewareComicCanvasContext,
} from './coursewareComicExportCanvasLayout'

interface LoadedCoursewareComicPanel {
  panel: CoursewareComicPanel
  image: HTMLImageElement
}

export async function buildCoursewareComicJPGCanvas(
  project: CoursewareComicWorkflowProject,
  prepared: PreparedCoursewareComicPanel[],
): Promise<HTMLCanvasElement> {
  const loaded = await loadPreparedPanels(prepared)
  const layout =
    createCoursewareComicLongLayout(
      project,
      loaded.length,
    )
  const canvas =
    createCoursewareComicCanvas(
      layout.width,
      layout.height,
    )
  const context =
    requireCoursewareComicCanvasContext(canvas)

  fillCoursewareComicCanvasWhite(
    context,
    layout.width,
    layout.height,
  )
  drawCoursewareComicCanvasHeader(
    context,
    project,
    loaded.length,
    layout.margin,
    layout.margin,
    layout.width - layout.margin * 2,
    68,
    30,
  )

  for (let index = 0; index < loaded.length; index += 1) {
    const column = index % layout.columns
    const row = Math.floor(index / layout.columns)
    const x =
      layout.margin +
      column * (layout.panelWidth + layout.columnGap)
    const y =
      layout.margin +
      layout.headerHeight +
      row * (layout.panelHeight + layout.rowGap)

    await drawPanel(
      context,
      loaded[index],
      x,
      y,
      layout.panelWidth,
      layout.panelHeight,
      1,
    )
  }

  return canvas
}

export async function buildCoursewareComicPDFPageCanvases(
  project: CoursewareComicWorkflowProject,
  prepared: PreparedCoursewareComicPanel[],
): Promise<HTMLCanvasElement[]> {
  const loaded = await loadPreparedPanels(prepared)
  const aspect =
    resolveCoursewareComicExportAspect(
      project.workflow.aspect_ratio,
    )
  const panelsPerPage =
    aspect.height > aspect.width
      ? 2
      : 4
  const pages: HTMLCanvasElement[] = []

  for (
    let start = 0;
    start < loaded.length;
    start += panelsPerPage
  ) {
    pages.push(
      await buildPDFPage(
        project,
        loaded.slice(start, start + panelsPerPage),
        start,
        loaded.length,
        panelsPerPage,
      ),
    )
  }

  return pages
}

async function loadPreparedPanels(
  prepared: PreparedCoursewareComicPanel[],
): Promise<LoadedCoursewareComicPanel[]> {
  await document.fonts?.ready

  return Promise.all(
    prepared.map(async item => ({
      panel: item.panel,
      image: await loadCoursewareComicExportImage(
        item.imageDataURL,
      ),
    })),
  )
}

async function buildPDFPage(
  project: CoursewareComicWorkflowProject,
  loaded: LoadedCoursewareComicPanel[],
  startIndex: number,
  totalCount: number,
  panelsPerPage: number,
): Promise<HTMLCanvasElement> {
  const layout =
    createCoursewareComicPDFLayout(
      project,
      panelsPerPage,
    )
  const canvas =
    createCoursewareComicCanvas(
      COURSEWARE_COMIC_PDF_WIDTH,
      COURSEWARE_COMIC_PDF_HEIGHT,
    )
  const context =
    requireCoursewareComicCanvasContext(canvas)

  fillCoursewareComicCanvasWhite(
    context,
    COURSEWARE_COMIC_PDF_WIDTH,
    COURSEWARE_COMIC_PDF_HEIGHT,
  )
  drawCoursewareComicCanvasHeader(
    context,
    project,
    totalCount,
    layout.margin,
    layout.margin,
    COURSEWARE_COMIC_PDF_WIDTH -
      layout.margin * 2,
    54,
    24,
    `第${startIndex + 1}—${startIndex + loaded.length}格`,
  )

  for (let index = 0; index < loaded.length; index += 1) {
    const column = index % layout.columns
    const row = Math.floor(index / layout.columns)
    const x =
      layout.contentStartX +
      column * (layout.panelWidth + layout.columnGap)
    const y =
      layout.contentStartY +
      row * (layout.panelHeight + layout.rowGap)

    await drawPanel(
      context,
      loaded[index],
      x,
      y,
      layout.panelWidth,
      layout.panelHeight,
      1.15,
    )
  }

  return canvas
}

async function drawPanel(
  context: CanvasRenderingContext2D,
  loaded: LoadedCoursewareComicPanel,
  x: number,
  y: number,
  width: number,
  height: number,
  scaleBoost: number,
): Promise<void> {
  drawCoursewareComicPanelBase(
    context,
    loaded.image,
    x,
    y,
    width,
    height,
    scaleBoost,
  )

  await drawCoursewareComicExportOverlay(
    context,
    loaded.panel.overlay_document,
    {
      x,
      y,
      width,
      height,
    },
    loaded.panel.panel_no,
  )

  drawCoursewareComicPanelNumber(
    context,
    loaded.panel.panel_no,
    x + 18 * scaleBoost,
    y + height - 18 * scaleBoost,
    scaleBoost,
  )
}
