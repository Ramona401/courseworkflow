/**
 * coursewareComicExportCanvasText.ts
 *
 * 知识点漫画Canvas基础图形工具：
 *   - 提供圆角矩形、边框、阴影、内边距和像素值解析；
 *   - 文字换行与垂直布局已迁移到浏览器真实DOM测量模块；
 *   - 本文件不再自行分词或计算可见行数。
 */

export interface CoursewareComicCanvasBorder {
  width: number
  color: string
}

export function coursewareComicRoundedRectPath(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number,
): void {
  const resolved = Math.max(
    0,
    Math.min(radius, width / 2, height / 2),
  )

  context.beginPath()
  context.moveTo(x + resolved, y)
  context.lineTo(x + width - resolved, y)
  context.quadraticCurveTo(
    x + width,
    y,
    x + width,
    y + resolved,
  )
  context.lineTo(x + width, y + height - resolved)
  context.quadraticCurveTo(
    x + width,
    y + height,
    x + width - resolved,
    y + height,
  )
  context.lineTo(x + resolved, y + height)
  context.quadraticCurveTo(
    x,
    y + height,
    x,
    y + height - resolved,
  )
  context.lineTo(x, y + resolved)
  context.quadraticCurveTo(x, y, x + resolved, y)
  context.closePath()
}

export function parseCoursewareComicCanvasBorder(
  value: unknown,
  displayScale: number,
): CoursewareComicCanvasBorder {
  const normalized = String(value || '').trim()

  if (!normalized || normalized === 'none') {
    return {
      width: 0,
      color: 'transparent',
    }
  }

  const match = normalized.match(
    /^([\d.]+)px\s+\S+\s+(.+)$/,
  )

  if (!match) {
    return {
      width: 0,
      color: 'transparent',
    }
  }

  return {
    width:
      (Number.parseFloat(match[1]) || 0) *
      displayScale,
    color: match[2],
  }
}

export function resolveCoursewareComicCanvasRadius(
  value: unknown,
  width: number,
  height: number,
  displayScale: number,
): number {
  const normalized = String(value || '0').trim()

  if (normalized.includes('%')) {
    const percent = Number.parseFloat(normalized)

    return Number.isFinite(percent)
      ? Math.min(width, height) * percent / 100
      : 0
  }

  if (
    normalized === '999px' ||
    normalized === '999'
  ) {
    return Math.min(width, height) / 2
  }

  return (
    Number.parseFloat(normalized) ||
    0
  ) * displayScale
}

export function parseCoursewareComicCanvasPadding(
  value: unknown,
  displayScale: number,
): {
  vertical: number
  horizontal: number
} {
  const numbers =
    String(value || '0').match(/-?[\d.]+/g) || []

  const vertical =
    (Number.parseFloat(numbers[0] || '0') || 0) *
    displayScale

  const horizontal =
    (
      Number.parseFloat(
        numbers[1] ||
        numbers[0] ||
        '0',
      ) ||
      0
    ) *
    displayScale

  return {
    vertical: Math.max(0, vertical),
    horizontal: Math.max(0, horizontal),
  }
}

export function parseCoursewareComicCanvasPixel(
  value: unknown,
  fallback: number,
): number {
  const parsed = Number.parseFloat(String(value || ''))

  return Number.isFinite(parsed)
    ? parsed
    : fallback
}

export function normalizeCoursewareComicCanvasFont(
  value: unknown,
): string {
  const normalized = String(value || '').trim()

  return (
    normalized ||
    'Noto Sans SC,Microsoft YaHei,PingFang SC,sans-serif'
  )
}

export function applyCoursewareComicCanvasShadow(
  context: CanvasRenderingContext2D,
  value: unknown,
  displayScale: number,
): void {
  const normalized = String(value || '')

  if (!normalized || normalized === 'none') {
    clearCoursewareComicCanvasShadow(context)
    return
  }

  const colorMatch = normalized.match(/rgba?\([^)]+\)/)
  const numbers =
    (
      normalized.match(/-?[\d.]+px/g) ||
      []
    ).map(
      item =>
        Number.parseFloat(item) ||
        0,
    )

  context.shadowOffsetX =
    (numbers[0] || 0) *
    displayScale

  context.shadowOffsetY =
    (numbers[1] || 5) *
    displayScale

  context.shadowBlur = Math.max(
    0,
    (numbers[2] || 16) *
      displayScale,
  )

  context.shadowColor =
    colorMatch?.[0] ||
    'rgba(15,23,42,0.18)'
}

export function clearCoursewareComicCanvasShadow(
  context: CanvasRenderingContext2D,
): void {
  context.shadowOffsetX = 0
  context.shadowOffsetY = 0
  context.shadowBlur = 0
  context.shadowColor = 'transparent'
}
