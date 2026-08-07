/**
 * coursewareComicExportDOMMeasure.ts
 *
 * 知识点漫画导出文字的浏览器真实排版测量：
 *   - 在屏幕外创建与正式预览一致的固定尺寸文字容器；
 *   - 由浏览器执行pre-wrap、overflow-wrap:anywhere和字体排版；
 *   - 使用Range读取每个实际字符的最终坐标；
 *   - Canvas按浏览器坐标逐字符绘制，不再猜测中英文换行；
 *   - 返回文字是否超出当前导出框体，供导出专用适配模块决策。
 */

export interface CoursewareComicDOMTextMeasureOptions {
  text: string
  width: number
  height: number
  paddingVertical: number
  paddingHorizontal: number
  fontSize: number
  fontWeight: string
  fontFamily: string
  lineHeight: number
  color: string
  align: 'left' | 'center' | 'right' | 'justify'
  verticalAlign: 'top' | 'center'
}

export interface CoursewareComicDOMTextGlyph {
  text: string
  left: number
  top: number
  width: number
  height: number
}

export interface CoursewareComicDOMTextLayout {
  glyphs: CoursewareComicDOMTextGlyph[]
  overflow: boolean
  contentWidth: number
  contentHeight: number
}

interface TextUnit {
  text: string
  start: number
  end: number
}

export function measureCoursewareComicDOMText(
  options: CoursewareComicDOMTextMeasureOptions,
): CoursewareComicDOMTextLayout {
  const host = createMeasurementHost(options)
  const textNode = document.createTextNode(options.text)

  host.text.appendChild(textNode)
  document.body.appendChild(host.root)

  try {
    const rootRect = host.root.getBoundingClientRect()
    const textRect = host.text.getBoundingClientRect()
    const contentWidth = Math.max(
      0,
      options.width - options.paddingHorizontal * 2,
    )
    const contentHeight = Math.max(
      0,
      options.height - options.paddingVertical * 2,
    )

    return {
      glyphs: collectMeasuredGlyphs(textNode, rootRect),
      overflow:
        textRect.height > contentHeight + 0.8 ||
        host.text.scrollWidth > contentWidth + 0.8,
      contentWidth: textRect.width,
      contentHeight: textRect.height,
    }
  } finally {
    host.root.remove()
  }
}

export function drawCoursewareComicMeasuredText(
  context: CanvasRenderingContext2D,
  layout: CoursewareComicDOMTextLayout,
  options: CoursewareComicDOMTextMeasureOptions,
): void {
  context.save()
  context.font =
    `${options.fontWeight} ${options.fontSize}px ${options.fontFamily}`
  context.fillStyle = options.color
  context.textAlign = 'left'
  context.textBaseline = 'alphabetic'

  for (const glyph of layout.glyphs) {
    if (!glyph.text || /^\s+$/u.test(glyph.text)) {
      continue
    }

    const metrics = context.measureText(glyph.text)
    const ascent = finitePositive(
      metrics.actualBoundingBoxAscent,
      options.fontSize * 0.8,
    )
    const descent = finitePositive(
      metrics.actualBoundingBoxDescent,
      options.fontSize * 0.2,
    )
    const glyphHeight = ascent + descent
    const baseline =
      glyph.top +
      Math.max(0, (glyph.height - glyphHeight) / 2) +
      ascent

    context.fillText(
      glyph.text,
      glyph.left,
      baseline,
      Math.max(1, glyph.width + 1),
    )
  }

  context.restore()
}

function createMeasurementHost(
  options: CoursewareComicDOMTextMeasureOptions,
): {
  root: HTMLDivElement
  text: HTMLSpanElement
} {
  const root = document.createElement('div')
  root.setAttribute('aria-hidden', 'true')
  root.style.position = 'fixed'
  root.style.left = '-100000px'
  root.style.top = '0'
  root.style.width = `${options.width}px`
  root.style.height = `${options.height}px`
  root.style.boxSizing = 'border-box'
  root.style.display = 'flex'
  root.style.flexDirection = 'column'
  root.style.justifyContent =
    options.verticalAlign === 'top'
      ? 'flex-start'
      : 'center'
  root.style.padding =
    `${options.paddingVertical}px ${options.paddingHorizontal}px`
  root.style.overflow = 'hidden'
  root.style.visibility = 'hidden'
  root.style.pointerEvents = 'none'
  root.style.zIndex = '-1'
  root.style.fontFamily = options.fontFamily
  root.style.fontSize = `${options.fontSize}px`
  root.style.fontWeight = options.fontWeight
  root.style.lineHeight = String(options.lineHeight)
  root.style.color = options.color
  root.style.textAlign = options.align
  root.style.fontKerning = 'normal'
  root.style.fontVariantLigatures = 'normal'

  const text = document.createElement('span')
  text.style.display = 'block'
  text.style.width = '100%'
  text.style.boxSizing = 'border-box'
  text.style.margin = '0'
  text.style.padding = '0'
  text.style.whiteSpace = 'pre-wrap'
  text.style.overflowWrap = 'anywhere'
  text.style.wordBreak = 'normal'
  text.style.textAlign = options.align

  root.appendChild(text)

  return {
    root,
    text,
  }
}

function collectMeasuredGlyphs(
  textNode: Text,
  rootRect: DOMRect,
): CoursewareComicDOMTextGlyph[] {
  const glyphs: CoursewareComicDOMTextGlyph[] = []

  for (const unit of buildTextUnits(textNode.data)) {
    if (unit.text === '\n') {
      continue
    }

    const rect = measureTextUnit(textNode, unit)

    if (!rect) {
      continue
    }

    glyphs.push({
      text: unit.text,
      left: rect.left - rootRect.left,
      top: rect.top - rootRect.top,
      width: Math.max(0, rect.width),
      height: Math.max(1, rect.height),
    })
  }

  return glyphs
}

function measureTextUnit(
  textNode: Text,
  unit: TextUnit,
): DOMRect | null {
  const range = document.createRange()

  try {
    range.setStart(textNode, unit.start)
    range.setEnd(textNode, unit.end)

    return (
      Array.from(range.getClientRects()).find(
        rect =>
          rect.height > 0 &&
          (
            rect.width > 0 ||
            /^\s+$/u.test(unit.text)
          ),
      ) ||
      null
    )
  } finally {
    range.detach()
  }
}

function buildTextUnits(value: string): TextUnit[] {
  const units: TextUnit[] = []
  let offset = 0

  for (const character of Array.from(value)) {
    const start = offset
    offset += character.length

    units.push({
      text: character,
      start,
      end: offset,
    })
  }

  return units
}

function finitePositive(
  value: number,
  fallback: number,
): number {
  return Number.isFinite(value) && value > 0
    ? value
    : fallback
}
