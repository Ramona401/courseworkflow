/**
 * coursewareComicEditorLayout.ts
 *
 * 漫画覆盖层文字适配与安全布局模块：
 *   - 教师拖动和缩放时严格保留手工位置、宽度和高度；
 *   - 教师缩小文本框、保存、刷新和普通预览均保持当前字号；
 *   - 只有字号按钮或教师明确点击“适配”时才允许改变字号；
 *   - 根据浏览器字体宽度、显式换行和真实行距估算文字空间；
 *   - 每次几何变化后只重新规范自动尾巴连接边，不改写人物语义目标；
 *   - 所有坐标保持在0至1归一化画布范围内。
 *
 * 本文件中的函数均为纯函数，不执行网络请求或浏览器持久化。
 */

import type {
  CoursewareComicOverlayDocument,
  CoursewareComicOverlayElement,
  CoursewareComicTextStyle,
} from '@/api/coursewares'
import {
  applyCoursewareComicTailPatch,
  normalizeCoursewareComicSpeechTail,
} from './coursewareComicTailEditing'
import type {
  CoursewareComicTailLayoutPatch,
} from './coursewareComicTailEditing'
import {
  overlayElementDisplayText,
} from './coursewareComicEditorText'
import {
  minimumCoursewareComicStoredFontSize,
  normalizeCoursewareComicStoredFontSize,
} from './coursewareComicTypography'

interface OverlayFitMetrics {
  text: string
  canvasWidth: number
  canvasHeight: number
  horizontalPadding: number
  verticalPadding: number
  minimumWidth: number
  maximumWidth: number
  minimumHeight: number
  maximumHeight: number
  lineHeight: number
  minimumFontSize: number
  fontFamily: string
  fontWeight: number
  elementType: CoursewareComicOverlayElement['type']
}

export type CoursewareComicOverlayLayoutPatch =
  Partial<Pick<
    CoursewareComicOverlayElement,
    'x' | 'y' | 'width' | 'height'
  >> &
  CoursewareComicTailLayoutPatch

const MIN_MANUAL_WIDTH = 0.07
const MIN_MANUAL_HEIGHT = 0.03
const MAX_MANUAL_WIDTH = 0.72
const MAX_MANUAL_HEIGHT = 0.72

function cloneJSON<Value>(value: Value): Value {
  return JSON.parse(JSON.stringify(value)) as Value
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(maximum, Math.max(minimum, value))
}

function updateDocumentElement(
  document: CoursewareComicOverlayDocument,
  elementID: string,
  updater: (
    element: CoursewareComicOverlayElement,
  ) => CoursewareComicOverlayElement,
): CoursewareComicOverlayDocument {
  return {
    ...document,
    elements: document.elements.map(element =>
      element.id === elementID
        ? updater(cloneJSON(element))
        : element,
    ),
  }
}

/**
 * 教师手工布局拥有最高优先级。
 *
 * 本函数只应用教师提交的位置和尺寸，不再调用文字自动扩容。
 * 这样四角拖动可以真实调整高度，松手后也不会被算法弹回。
 */
export function updateCoursewareComicOverlayLayout(
  document: CoursewareComicOverlayDocument,
  elementID: string,
  patch: CoursewareComicOverlayLayoutPatch,
): CoursewareComicOverlayDocument {
  return updateDocumentElement(document, elementID, element => {
    const width = clamp(
      patch.width ?? element.width,
      MIN_MANUAL_WIDTH,
      MAX_MANUAL_WIDTH,
    )
    const height = clamp(
      patch.height ?? element.height,
      MIN_MANUAL_HEIGHT,
      MAX_MANUAL_HEIGHT,
    )
    const positioned: CoursewareComicOverlayElement = {
      ...element,
      width,
      height,
      x: clamp(patch.x ?? element.x, 0, 1 - width),
      y: clamp(patch.y ?? element.y, 0, 1 - height),
      layout_dirty: true,
    }

    return normalizeCoursewareComicSpeechTail(
      applyCoursewareComicTailPatch(positioned, patch),
    )
  })
}

export function updateCoursewareComicOverlayTextStyle(
  document: CoursewareComicOverlayDocument,
  elementID: string,
  patch: Partial<CoursewareComicTextStyle>,
): CoursewareComicOverlayDocument {
  return updateDocumentElement(document, elementID, element => {
    const updated: CoursewareComicOverlayElement = {
      ...element,
      text_style: {
        ...element.text_style,
        ...patch,
        font_size: normalizeCoursewareComicStoredFontSize(
          element,
          patch.font_size ?? element.text_style.font_size,
        ),
        font_weight: clamp(
          patch.font_weight ?? element.text_style.font_weight,
          300,
          900,
        ),
        line_height: clamp(
          patch.line_height ??
            (element.text_style.line_height || 1.35),
          1,
          2.2,
        ),
        outline_width: clamp(
          patch.outline_width ??
            (element.text_style.outline_width || 1),
          0.5,
          3,
        ),
      },
    }

    return normalizeCoursewareComicSpeechTail(updated)
  })
}

/**
 * “适配”按钮是唯一允许自动改变文本框宽高的位置。
 */
export function fitCoursewareComicOverlayElementByID(
  document: CoursewareComicOverlayDocument,
  elementID: string,
): CoursewareComicOverlayDocument {
  return updateDocumentElement(document, elementID, element => ({
    ...autoFitCoursewareComicOverlayElement(
      { ...element, layout_dirty: false },
      document.canvas,
    ),
    layout_dirty: true,
  }))
}

/**
 * 草稿加载只规范边界、描边和尾巴，不修改已保存字号或几何。
 */
export function compactCoursewareComicOverlayDocument(
  document: CoursewareComicOverlayDocument,
): CoursewareComicOverlayDocument {
  return {
    ...document,
    elements: document.elements.map(element =>
      fitCoursewareComicOverlayElement(
        element,
        document.canvas,
        true,
      ),
    ),
  }
}

/**
 * 兼容旧调用的稳定规范化入口。
 *
 * 保存、刷新、草稿恢复、复制和普通预览都可以调用本函数，但它只做
 * 边界与协议规范化，绝不根据框体大小隐式降低字号。
 */
export function fitCoursewareComicOverlayElement(
  element: CoursewareComicOverlayElement,
  _canvas: { width: number; height: number },
  _preserveLayout = true,
): CoursewareComicOverlayElement {
  const width = clamp(
    element.width,
    MIN_MANUAL_WIDTH,
    MAX_MANUAL_WIDTH,
  )
  const height = clamp(
    element.height,
    MIN_MANUAL_HEIGHT,
    MAX_MANUAL_HEIGHT,
  )

  return normalizeCoursewareComicSpeechTail({
    ...element,
    width,
    height,
    x: clamp(element.x, 0, 1 - width),
    y: clamp(element.y, 0, 1 - height),
    text_style: {
      ...element.text_style,
      font_size: normalizeCoursewareComicStoredFontSize(
        element,
        element.text_style.font_size || 40,
      ),
      font_weight: clamp(
        element.text_style.font_weight || 600,
        300,
        900,
      ),
      line_height: clamp(
        element.text_style.line_height || 1.35,
        1,
        2.2,
      ),
      outline_width: clamp(
        element.text_style.outline_width || 1,
        0.5,
        3,
      ),
    },
  })
}

/**
 * 教师明确点击“适配”时重新计算舒适宽高；只有这里可以自动缩字号。
 */
function autoFitCoursewareComicOverlayElement(
  element: CoursewareComicOverlayElement,
  canvas: { width: number; height: number },
): CoursewareComicOverlayElement {
  const metrics = resolveOverlayFitMetrics(element, canvas)
  const currentFontSize = normalizeCoursewareComicStoredFontSize(
    element,
    element.text_style.font_size || 40,
  )
  const textLength = Math.max(1, Array.from(metrics.text).length)
  const bubble =
    element.type === 'speech_bubble' ||
    element.type === 'thought_bubble'
  const targetCharacters = bubble
    ? clamp(
        textLength <= 16 ? textLength : Math.ceil(textLength / 2),
        8,
        26,
      )
    : clamp(
        textLength <= 10
          ? textLength
          : Math.ceil(Math.sqrt(textLength) * 2.2),
        4,
        element.type === 'question_card' ? 20 : 17,
      )

  const baseWidth = clamp(
    (
      targetCharacters * currentFontSize * 0.9 +
      metrics.horizontalPadding
    ) / metrics.canvasWidth,
    metrics.minimumWidth,
    metrics.maximumWidth,
  )
  const comfort = resolveOverlayComfortExpansion(element, canvas)
  const width = clamp(
    baseWidth + comfort.width,
    metrics.minimumWidth,
    metrics.maximumWidth,
  )
  const baseHeight = clamp(
    requiredOverlayHeight(metrics, width, currentFontSize),
    metrics.minimumHeight,
    metrics.maximumHeight,
  )
  const height = clamp(
    baseHeight + comfort.height,
    metrics.minimumHeight,
    metrics.maximumHeight,
  )
  const originalCenterX = element.x + element.width / 2
  const originalCenterY = element.y + element.height / 2
  const x = clamp(originalCenterX - width / 2, 0, 1 - width)
  const y = clamp(originalCenterY - height / 2, 0, 1 - height)

  return normalizeCoursewareComicSpeechTail({
    ...element,
    width,
    height,
    x,
    y,
    text_style: {
      ...element.text_style,
      font_size: resolveFittedFontSize(
        metrics,
        width,
        height,
        currentFontSize,
      ),
    },
  })
}

function resolveOverlayComfortExpansion(
  element: CoursewareComicOverlayElement,
  canvas: { width: number; height: number },
): { width: number; height: number } {
  const bubble =
    element.type === 'speech_bubble' ||
    element.type === 'thought_bubble'
  const question = element.type === 'question_card'
  const horizontalPixels = bubble ? 16 : question ? 14 : 10
  const verticalPixels = bubble ? 10 : question ? 10 : 8

  return {
    width: horizontalPixels / Math.max(320, canvas.width),
    height: verticalPixels / Math.max(180, canvas.height),
  }
}

function estimatedTextHeight(
  lineCount: number,
  fontSize: number,
  lineHeight: number,
): number {
  const safeLineHeight = clamp(lineHeight, 1, 2.2)
  const lineBoxHeight = fontSize * safeLineHeight
  const glyphSafety = Math.max(4, fontSize * 0.12)

  return Math.max(1, lineCount) * lineBoxHeight + glyphSafety
}

function requiredOverlayHeight(
  metrics: OverlayFitMetrics,
  width: number,
  fontSize: number,
): number {
  const availableWidth = Math.max(
    20,
    metrics.canvasWidth * width - metrics.horizontalPadding,
  )
  const lineCount = estimateLineCount(
    metrics.text,
    availableWidth,
    fontSize,
    metrics.fontFamily,
    metrics.fontWeight,
  )

  return (
    estimatedTextHeight(
      lineCount,
      fontSize,
      metrics.lineHeight,
    ) +
    metrics.verticalPadding +
    Math.max(2, fontSize * 0.04)
  ) / metrics.canvasHeight
}

function resolveOverlayFitMetrics(
  element: CoursewareComicOverlayElement,
  canvas: { width: number; height: number },
): OverlayFitMetrics {
  const bubble =
    element.type === 'speech_bubble' ||
    element.type === 'thought_bubble'
  const question = element.type === 'question_card'

  return {
    text: overlayElementDisplayText(element),
    canvasWidth: Math.max(320, canvas.width),
    canvasHeight: Math.max(180, canvas.height),
    horizontalPadding: bubble ? 36 : question ? 44 : 32,
    verticalPadding: bubble ? 20 : question ? 28 : 18,
    minimumWidth: question ? 0.22 : bubble ? 0.18 : 0.12,
    maximumWidth: question ? 0.52 : bubble ? 0.56 : 0.46,
    minimumHeight: question ? 0.12 : bubble ? 0.035 : 0.03,
    maximumHeight: question ? 0.56 : 0.44,
    lineHeight: clamp(element.text_style.line_height || 1.35, 1, 2.2),
    minimumFontSize: minimumCoursewareComicStoredFontSize(element),
    fontFamily:
      element.text_style.font_family ||
      'Noto Sans SC, sans-serif',
    fontWeight: clamp(
      element.text_style.font_weight || 600,
      300,
      900,
    ),
    elementType: element.type,
  }
}

function resolveFittedFontSize(
  metrics: OverlayFitMetrics,
  width: number,
  height: number,
  maximumFontSize: number,
): number {
  const availableHeight = Math.max(
    12,
    metrics.canvasHeight * height - metrics.verticalPadding,
  )

  for (
    let fontSize = clamp(
      maximumFontSize,
      metrics.minimumFontSize,
      96,
    );
    fontSize >= metrics.minimumFontSize;
    fontSize -= 2
  ) {
    const required =
      requiredOverlayHeight(
        { ...metrics, verticalPadding: 0 },
        width,
        fontSize,
      ) *
      metrics.canvasHeight

    if (required <= availableHeight) {
      return fontSize
    }
  }

  return metrics.minimumFontSize
}

let overlayMeasureContext: CanvasRenderingContext2D | null | undefined

function resolveOverlayMeasureContext(): CanvasRenderingContext2D | null {
  if (overlayMeasureContext !== undefined) {
    return overlayMeasureContext
  }

  if (typeof document === 'undefined') {
    overlayMeasureContext = null
    return null
  }

  overlayMeasureContext =
    document.createElement('canvas').getContext('2d')

  return overlayMeasureContext
}

function estimatedCharacterWidth(
  character: string,
  fontSize: number,
): number {
  if (/\s/u.test(character)) {
    return fontSize * 0.34
  }

  if (/[\u2E80-\u9FFF\uF900-\uFAFF\uFF01-\uFF60]/u.test(character)) {
    return fontSize
  }

  if (/[，。！？；：、（）《》“”‘’…—]/u.test(character)) {
    return fontSize * 0.58
  }

  if (/[A-Z]/u.test(character)) {
    return fontSize * 0.68
  }

  if (/[a-z0-9]/u.test(character)) {
    return fontSize * 0.57
  }

  return fontSize * 0.76
}

function measuredCharacterWidth(
  character: string,
  fontSize: number,
  fontFamily: string,
  fontWeight: number,
): number {
  const context = resolveOverlayMeasureContext()

  if (!context) {
    return estimatedCharacterWidth(character, fontSize)
  }

  context.font =
    `${fontWeight} ${fontSize}px ${fontFamily}`

  const measured = context.measureText(character).width

  return Number.isFinite(measured) && measured > 0
    ? measured
    : estimatedCharacterWidth(character, fontSize)
}

function estimateLineCount(
  text: string,
  availableWidth: number,
  fontSize: number,
  fontFamily: string,
  fontWeight: number,
): number {
  const widthSafety = Math.max(4, fontSize * 0.12)
  const safeWidth = Math.max(
    fontSize * 1.5,
    availableWidth - widthSafety,
  )
  let totalLines = 0

  for (const paragraph of text.split('\n')) {
    const characters = Array.from(paragraph)

    if (!characters.length) {
      totalLines += 1
      continue
    }

    let paragraphLines = 1
    let currentWidth = 0

    for (const character of characters) {
      const characterWidth =
        measuredCharacterWidth(
          character,
          fontSize,
          fontFamily,
          fontWeight,
        ) *
        1.04 *
        (fontWeight >= 700 ? 1.025 : 1)

      if (
        currentWidth > 0 &&
        currentWidth + characterWidth > safeWidth
      ) {
        paragraphLines += 1
        currentWidth = characterWidth
      } else {
        currentWidth += characterWidth
      }
    }

    totalLines += paragraphLines
  }

  return Math.max(1, totalLines)
}
