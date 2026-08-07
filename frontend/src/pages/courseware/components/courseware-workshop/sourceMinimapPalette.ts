/**
 * sourceMinimapPalette.ts — 源码 Minimap 结构分类器
 *
 * 目标：
 *   1. 在右侧代码缩略图中，用不同颜色区分页面内容、样式、函数和注释；
 *   2. 不参与源码保存，不修改任何源码字符；
 *   3. 不引入 Monaco / CodeMirror 等大型第三方编辑器；
 *   4. 对不完整 HTML、CSS、JavaScript 保持宽容，分类失败时退回普通灰色。
 *
 * 分类颜色：
 *   - content  金黄色：HTML结构、属性和老师常修改的页面文字；
 *   - style    紫色：<style>、CSS内容和行内 style 属性；
 *   - function 青绿色：<script>、函数代码和 onclick 等事件属性；
 *   - comment  灰色：HTML、CSS、JavaScript 注释；
 *   - neutral  深灰色：无法明确分类的辅助内容。
 */

/** Minimap 支持的结构类别。 */
export type SourceMinimapCategory =
  | 'content'
  | 'style'
  | 'function'
  | 'comment'
  | 'neutral'

/** 一条缩略线中的一个彩色片段。 */
export interface SourceMinimapSegment {
  category: SourceMinimapCategory
  weight: number
}

/** Minimap 正式颜色表，绘制工具和界面图例共用。 */
export const SOURCE_MINIMAP_PALETTE: Record<
  SourceMinimapCategory,
  string
> = {
  content: '#F6C453',
  style: '#A78BFA',
  function: '#2DD4BF',
  comment: '#64748B',
  neutral: '#4B5563',
}

/**
 * 跨行扫描状态。
 *
 * HTML、style、script 和三种块注释都可能跨越多行，
 * 因此不能只用单行正则判断。
 */
interface SourceMinimapScanState {
  mode:
    | 'html'
    | 'style'
    | 'script'
    | 'html-comment'
    | 'css-comment'
    | 'javascript-comment'
}

/** 向结果中加入片段，空白不计入缩略条宽度。 */
function pushSegment(
  segments: SourceMinimapSegment[],
  category: SourceMinimapCategory,
  text: string,
): void {
  const weight = text.replace(/\s/gu, '').length
  if (weight <= 0) return

  const previous = segments.at(-1)
  if (
    previous
    && previous.category === category
  ) {
    previous.weight += weight
    return
  }

  segments.push({
    category,
    weight,
  })
}

/** 从若干候选位置中找最靠前的有效位置。 */
function findFirstIndex(
  candidates: number[],
): number {
  let result = -1

  for (const candidate of candidates) {
    if (candidate < 0) continue

    if (
      result < 0
      || candidate < result
    ) {
      result = candidate
    }
  }

  return result
}

/**
 * 读取带引号的 HTML 属性值结束位置。
 *
 * 处理反斜线转义，避免字符串中的同类引号过早结束。
 */
function findQuotedAttributeEnd(
  line: string,
  quoteIndex: number,
  limit: number,
): number {
  const quote = line[quoteIndex]
  let cursor = quoteIndex + 1

  while (cursor < limit) {
    if (line[cursor] === '\\') {
      cursor += 2
      continue
    }

    cursor += 1

    if (line[cursor - 1] === quote) {
      break
    }
  }

  return Math.min(
    cursor,
    limit,
  )
}

/**
 * 分类普通 HTML 片段。
 *
 * 普通 HTML 默认属于 content；
 * 其中 style="..." 单独标记为 style，
 * onclick/onload/onchange 等事件属性单独标记为 function。
 */
function appendHtmlChunk(
  line: string,
  start: number,
  end: number,
  segments: SourceMinimapSegment[],
): void {
  if (end <= start) return

  const attributePattern =
    /\b(style|on[a-z][\w:-]*)\s*=\s*(["'])/giu

  attributePattern.lastIndex =
    start

  let cursor = start
  let match:
    RegExpExecArray
    | null

  while (
    (
      match =
        attributePattern.exec(line)
    ) !== null
  ) {
    if (
      match.index >= end
    ) {
      break
    }

    const quoteIndex =
      attributePattern.lastIndex - 1

    const attributeEnd =
      findQuotedAttributeEnd(
        line,
        quoteIndex,
        end,
      )

    if (
      match.index > cursor
    ) {
      pushSegment(
        segments,
        'content',
        line.slice(
          cursor,
          match.index,
        ),
      )
    }

    pushSegment(
      segments,
      match[1].toLowerCase()
        === 'style'
        ? 'style'
        : 'function',
      line.slice(
        match.index,
        attributeEnd,
      ),
    )

    cursor =
      attributeEnd

    attributePattern.lastIndex =
      Math.max(
        attributePattern.lastIndex,
        cursor,
      )
  }

  if (
    cursor < end
  ) {
    pushSegment(
      segments,
      'content',
      line.slice(
        cursor,
        end,
      ),
    )
  }
}

/**
 * 分类一行源码。
 *
 * mode 会在调用间持续保存，
 * 因此可以正确处理跨行 style、script 和块注释。
 */
function classifySourceLine(
  line: string,
  state: SourceMinimapScanState,
): SourceMinimapSegment[] {
  const segments:
    SourceMinimapSegment[] = []

  const lowerLine =
    line.toLowerCase()

  let cursor = 0

  while (
    cursor < line.length
  ) {
    if (
      state.mode
        === 'html-comment'
    ) {
      const close =
        line.indexOf(
          '-->',
          cursor,
        )

      if (
        close < 0
      ) {
        pushSegment(
          segments,
          'comment',
          line.slice(cursor),
        )
        cursor =
          line.length
        continue
      }

      pushSegment(
        segments,
        'comment',
        line.slice(
          cursor,
          close + 3,
        ),
      )

      cursor =
        close + 3
      state.mode =
        'html'
      continue
    }

    if (
      state.mode
        === 'css-comment'
      || state.mode
        === 'javascript-comment'
    ) {
      const close =
        line.indexOf(
          '*/',
          cursor,
        )

      if (
        close < 0
      ) {
        pushSegment(
          segments,
          'comment',
          line.slice(cursor),
        )
        cursor =
          line.length
        continue
      }

      pushSegment(
        segments,
        'comment',
        line.slice(
          cursor,
          close + 2,
        ),
      )

      cursor =
        close + 2

      state.mode =
        state.mode
          === 'css-comment'
          ? 'style'
          : 'script'

      continue
    }

    if (
      state.mode
        === 'style'
    ) {
      const commentStart =
        line.indexOf(
          '/*',
          cursor,
        )

      const closeTagStart =
        lowerLine.indexOf(
          '</style',
          cursor,
        )

      const next =
        findFirstIndex([
          commentStart,
          closeTagStart,
        ])

      if (
        next < 0
      ) {
        pushSegment(
          segments,
          'style',
          line.slice(cursor),
        )
        cursor =
          line.length
        continue
      }

      if (
        next > cursor
      ) {
        pushSegment(
          segments,
          'style',
          line.slice(
            cursor,
            next,
          ),
        )
      }

      if (
        next === commentStart
      ) {
        state.mode =
          'css-comment'
        cursor =
          commentStart
        continue
      }

      const closeTagEnd =
        line.indexOf(
          '>',
          closeTagStart,
        )

      if (
        closeTagEnd < 0
      ) {
        pushSegment(
          segments,
          'style',
          line.slice(
            closeTagStart,
          ),
        )
        cursor =
          line.length
        state.mode =
          'html'
        continue
      }

      pushSegment(
        segments,
        'style',
        line.slice(
          closeTagStart,
          closeTagEnd + 1,
        ),
      )

      cursor =
        closeTagEnd + 1
      state.mode =
        'html'
      continue
    }

    if (
      state.mode
        === 'script'
    ) {
      const blockCommentStart =
        line.indexOf(
          '/*',
          cursor,
        )

      const lineCommentStart =
        line.indexOf(
          '//',
          cursor,
        )

      const closeTagStart =
        lowerLine.indexOf(
          '</script',
          cursor,
        )

      const next =
        findFirstIndex([
          blockCommentStart,
          lineCommentStart,
          closeTagStart,
        ])

      if (
        next < 0
      ) {
        pushSegment(
          segments,
          'function',
          line.slice(cursor),
        )
        cursor =
          line.length
        continue
      }

      if (
        next > cursor
      ) {
        pushSegment(
          segments,
          'function',
          line.slice(
            cursor,
            next,
          ),
        )
      }

      if (
        next
          === blockCommentStart
      ) {
        state.mode =
          'javascript-comment'
        cursor =
          blockCommentStart
        continue
      }

      if (
        next
          === lineCommentStart
      ) {
        pushSegment(
          segments,
          'comment',
          line.slice(
            lineCommentStart,
          ),
        )
        cursor =
          line.length
        continue
      }

      const closeTagEnd =
        line.indexOf(
          '>',
          closeTagStart,
        )

      if (
        closeTagEnd < 0
      ) {
        pushSegment(
          segments,
          'function',
          line.slice(
            closeTagStart,
          ),
        )
        cursor =
          line.length
        state.mode =
          'html'
        continue
      }

      pushSegment(
        segments,
        'function',
        line.slice(
          closeTagStart,
          closeTagEnd + 1,
        ),
      )

      cursor =
        closeTagEnd + 1
      state.mode =
        'html'
      continue
    }

    const htmlCommentStart =
      line.indexOf(
        '<!--',
        cursor,
      )

    const styleStart =
      lowerLine.indexOf(
        '<style',
        cursor,
      )

    const scriptStart =
      lowerLine.indexOf(
        '<script',
        cursor,
      )

    const next =
      findFirstIndex([
        htmlCommentStart,
        styleStart,
        scriptStart,
      ])

    if (
      next < 0
    ) {
      appendHtmlChunk(
        line,
        cursor,
        line.length,
        segments,
      )

      cursor =
        line.length
      continue
    }

    if (
      next > cursor
    ) {
      appendHtmlChunk(
        line,
        cursor,
        next,
        segments,
      )
    }

    if (
      next
        === htmlCommentStart
    ) {
      state.mode =
        'html-comment'
      cursor =
        htmlCommentStart
      continue
    }

    const openTagEnd =
      line.indexOf(
        '>',
        next,
      )

    const category:
      SourceMinimapCategory =
        next === styleStart
          ? 'style'
          : 'function'

    if (
      openTagEnd < 0
    ) {
      pushSegment(
        segments,
        category,
        line.slice(next),
      )

      cursor =
        line.length

      state.mode =
        category === 'style'
          ? 'style'
          : 'script'

      continue
    }

    pushSegment(
      segments,
      category,
      line.slice(
        next,
        openTagEnd + 1,
      ),
    )

    cursor =
      openTagEnd + 1

    state.mode =
      category === 'style'
        ? 'style'
        : 'script'
  }

  if (
    segments.length === 0
    && line.trim()
  ) {
    pushSegment(
      segments,
      'neutral',
      line,
    )
  }

  return segments
}

/**
 * 对整份源码逐行分类。
 *
 * 返回数组长度始终与 lines 一致，
 * 便于 Minimap 按原始行号直接绘制。
 */
export function buildSourceMinimapLines(
  lines: string[],
): SourceMinimapSegment[][] {
  const state:
    SourceMinimapScanState = {
      mode: 'html',
    }

  return lines.map(
    line =>
      classifySourceLine(
        line,
        state,
      ),
  )
}
