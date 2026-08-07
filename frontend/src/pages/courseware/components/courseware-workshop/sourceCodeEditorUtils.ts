/**
 * sourceCodeEditorUtils.ts — 课件源码编辑器通用计算工具
 *
 * 本文件只处理搜索、光标位置和 Minimap 绘制，不包含 React 状态。
 *
 * Minimap 正式颜色口径：
 *   - 紫色：样式；
 *   - 金色：页面内容；
 *   - 青绿色：函数与交互；
 *   - 灰色：注释和无法明确分类的辅助内容；
 *   - 黄色：全部搜索结果；
 *   - 橙色：当前搜索结果；
 *   - 蓝色半透明框：编辑区当前可见范围。
 */

import {
  buildSourceMinimapLines,
  SOURCE_MINIMAP_PALETTE,
} from './sourceMinimapPalette'
import type {
  SourceMinimapSegment,
} from './sourceMinimapPalette'

export interface SourceSearchMatch {
  start: number
  end: number
  line: number
}

export interface SourceCursorPosition {
  line: number
  column: number
}

export const SOURCE_LINE_HEIGHT =
  20.4

export const SOURCE_MINIMAP_WIDTH =
  104

export const SOURCE_EDITOR_FONT =
  'Monaco, Consolas, "Courier New", monospace'

/**
 * lines 由 SourceCodeEditor useMemo 生成，
 * 同一份源码滚动时引用保持不变，因此用 WeakMap 缓存结构分类结果，
 * 避免每一次滚动都重新分析整份源码。
 */
const minimapLineCache =
  new WeakMap<
    string[],
    SourceMinimapSegment[][]
  >()

function getMinimapLines(
  lines: string[],
): SourceMinimapSegment[][] {
  const cached =
    minimapLineCache.get(lines)

  if (cached) {
    return cached
  }

  const result =
    buildSourceMinimapLines(lines)

  minimapLineCache.set(
    lines,
    result,
  )

  return result
}

function escapeRegExp(
  value: string,
): string {
  return value.replace(
    /[.*+?^${}()|[\]\\]/g,
    '\\$&',
  )
}

function isWordCharacter(
  value: string,
): boolean {
  return Boolean(value)
    && /[\p{L}\p{N}_]/u.test(value)
}

/**
 * 字面量搜索并记录行号。
 *
 * 不开放正则输入，降低非开发老师误操作成本。
 */
export function findSourceMatches(
  source: string,
  query: string,
  matchCase: boolean,
  wholeWord: boolean,
): SourceSearchMatch[] {
  if (!query) return []

  const matcher =
    new RegExp(
      escapeRegExp(query),
      matchCase
        ? 'gu'
        : 'giu',
    )

  const matches:
    SourceSearchMatch[] = []

  let line = 1
  let lineCursor = 0
  let result:
    RegExpExecArray
    | null

  while (
    (
      result =
        matcher.exec(source)
    ) !== null
  ) {
    const start =
      result.index

    const end =
      start
      + result[0].length

    while (
      lineCursor < start
    ) {
      if (
        source.charCodeAt(
          lineCursor,
        ) === 10
      ) {
        line += 1
      }

      lineCursor += 1
    }

    if (wholeWord) {
      const before =
        source.slice(
          Math.max(
            0,
            start - 1,
          ),
          start,
        )

      const after =
        source.slice(
          end,
          end + 1,
        )

      if (
        isWordCharacter(before)
        || isWordCharacter(after)
      ) {
        continue
      }
    }

    matches.push({
      start,
      end,
      line,
    })
  }

  return matches
}

/**
 * 将 textarea 字符位置换算为老师熟悉的
 * 1-based 行号和列号。
 */
export function getSourceLineColumn(
  source: string,
  position: number,
): SourceCursorPosition {
  const safe =
    Math.max(
      0,
      Math.min(
        position,
        source.length,
      ),
    )

  const parts =
    source
      .slice(
        0,
        safe,
      )
      .split('\n')

  return {
    line:
      parts.length,
    column:
      (
        parts.at(-1)?.length
        || 0
      ) + 1,
  }
}

interface DrawSourceMinimapOptions {
  textarea: HTMLTextAreaElement
  host: HTMLDivElement
  canvas: HTMLCanvasElement
  lines: string[]
  matches: SourceSearchMatch[]
  activeMatch: number
}

/**
 * 合并采样范围内的结构片段。
 *
 * 超长源码可能按多行一组采样；
 * 这里保留各行原始先后顺序，并合并相邻同类片段。
 */
function collectSampledSegments(
  minimapLines:
    SourceMinimapSegment[][],
  start: number,
  end: number,
): SourceMinimapSegment[] {
  const result:
    SourceMinimapSegment[] = []

  for (
    let lineIndex = start;
    lineIndex < end;
    lineIndex += 1
  ) {
    const lineSegments =
      minimapLines[lineIndex]
      || []

    for (
      const segment
      of lineSegments
    ) {
      const previous =
        result.at(-1)

      if (
        previous
        && previous.category
          === segment.category
      ) {
        previous.weight +=
          segment.weight
      } else {
        result.push({
          category:
            segment.category,
          weight:
            segment.weight,
        })
      }
    }
  }

  return result
}

/**
 * 在一条缩略线上按字符占比绘制多段颜色。
 *
 * 同一行中：
 *   <div style="..." onclick="...">正文</div>
 * 会依次显示内容色、样式色、函数色和内容色。
 */
function drawSegmentedMinimapBar(
  context:
    CanvasRenderingContext2D,
  segments:
    SourceMinimapSegment[],
  x: number,
  y: number,
  width: number,
  height: number,
): void {
  const totalWeight =
    segments.reduce(
      (
        total,
        segment,
      ) =>
        total
        + segment.weight,
      0,
    )

  if (
    totalWeight <= 0
    || width <= 0
  ) {
    return
  }

  let offset = 0

  segments.forEach(
    (
      segment,
      index,
    ) => {
      const remaining =
        Math.max(
          0,
          width - offset,
        )

      if (
        remaining <= 0
      ) {
        return
      }

      const segmentWidth =
        index
          === segments.length - 1
          ? remaining
          : Math.min(
              remaining,
              Math.max(
                1,
                width
                * (
                  segment.weight
                  / totalWeight
                ),
              ),
            )

      context.fillStyle =
        SOURCE_MINIMAP_PALETTE[
          segment.category
        ]

      context.fillRect(
        x + offset,
        y,
        segmentWidth,
        height,
      )

      offset +=
        segmentWidth
    },
  )
}

/**
 * 绘制右侧代码小窗。
 *
 * 主体颜色表示源码结构；
 * 搜索命中和当前视口仍覆盖在结构颜色之上。
 */
export function drawSourceMinimap({
  textarea,
  host,
  canvas,
  lines,
  matches,
  activeMatch,
}: DrawSourceMinimapOptions): void {
  const width =
    Math.max(
      1,
      host.clientWidth,
    )

  const canvasHeight =
    Math.max(
      1,
      host.clientHeight,
    )

  const lineCount =
    Math.max(
      1,
      lines.length,
    )

  const dpr =
    window.devicePixelRatio
    || 1

  const pixelWidth =
    Math.floor(
      width * dpr,
    )

  const pixelHeight =
    Math.floor(
      canvasHeight * dpr,
    )

  if (
    canvas.width
      !== pixelWidth
    || canvas.height
      !== pixelHeight
  ) {
    canvas.width =
      pixelWidth

    canvas.height =
      pixelHeight
  }

  canvas.style.width =
    `${width}px`

  canvas.style.height =
    `${canvasHeight}px`

  const context =
    canvas.getContext('2d')

  if (!context) return

  context.setTransform(
    dpr,
    0,
    0,
    dpr,
    0,
    0,
  )

  context.clearRect(
    0,
    0,
    width,
    canvasHeight,
  )

  context.fillStyle =
    '#171717'

  context.fillRect(
    0,
    0,
    width,
    canvasHeight,
  )

  const minimapLines =
    getMinimapLines(lines)

  const rowHeight =
    canvasHeight
    / lineCount

  const step =
    Math.max(
      1,
      Math.ceil(
        lineCount / 700,
      ),
    )

  for (
    let index = 0;
    index < lines.length;
    index += step
  ) {
    const sampleEnd =
      Math.min(
        lines.length,
        index + step,
      )

    const segments =
      collectSampledSegments(
        minimapLines,
        index,
        sampleEnd,
      )

    if (
      segments.length === 0
    ) {
      continue
    }

    let longestLine = 0

    for (
      let lineIndex = index;
      lineIndex < sampleEnd;
      lineIndex += 1
    ) {
      longestLine =
        Math.max(
          longestLine,
          lines[lineIndex]
            .trimEnd()
            .length,
        )
    }

    const barWidth =
      Math.max(
        2,
        Math.min(
          width - 12,
          (
            longestLine / 140
          )
          * (width - 12),
        ),
      )

    const barHeight =
      Math.max(
        1,
        Math.min(
          2.2,
          rowHeight
          * step
          * 0.68,
        ),
      )

    drawSegmentedMinimapBar(
      context,
      segments,
      5,
      index * rowHeight,
      barWidth,
      barHeight,
    )
  }

  /**
   * 全部搜索结果使用右侧黄色短线。
   */
  if (
    matches.length > 0
  ) {
    context.fillStyle =
      '#FACC15'

    for (
      const match
      of matches
    ) {
      context.fillRect(
        width - 5,
        (
          match.line - 1
        ) * rowHeight,
        3,
        Math.max(
          2,
          rowHeight,
        ),
      )
    }

    /**
     * 当前结果使用整行橙色覆盖，
     * 比普通命中更容易被老师识别。
     */
    const current =
      matches[activeMatch]

    if (current) {
      context.fillStyle =
        'rgba(251,146,60,0.78)'

      context.fillRect(
        0,
        (
          current.line - 1
        ) * rowHeight,
        width,
        Math.max(
          2,
          rowHeight,
        ),
      )
    }
  }

  const scrollHeight =
    Math.max(
      textarea.scrollHeight,
      textarea.clientHeight,
    )

  const viewportHeight =
    Math.max(
      20,
      Math.min(
        canvasHeight,
        (
          textarea.clientHeight
          / scrollHeight
        ) * canvasHeight,
      ),
    )

  const maxEditorScroll =
    Math.max(
      0,
      scrollHeight
      - textarea.clientHeight,
    )

  const maxMinimapScroll =
    Math.max(
      0,
      canvasHeight
      - viewportHeight,
    )

  const viewportTop =
    maxEditorScroll > 0
      ? (
          textarea.scrollTop
          / maxEditorScroll
        ) * maxMinimapScroll
      : 0

  context.fillStyle =
    'rgba(147,197,253,0.13)'

  context.fillRect(
    0,
    viewportTop,
    width,
    viewportHeight,
  )

  context.strokeStyle =
    'rgba(147,197,253,0.58)'

  context.lineWidth =
    1

  context.strokeRect(
    0.5,
    viewportTop + 0.5,
    width - 1,
    Math.max(
      1,
      viewportHeight - 1,
    ),
  )
}
