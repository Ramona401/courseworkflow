/**
 * sourceCodeSyntax.ts — 课件源码轻量语法着色器
 *
 * 目标不是实现完整编译器，而是在不引入 Monaco / CodeMirror 的前提下，
 * 为课件 HTML、内嵌 CSS 和 JavaScript 提供稳定、可读、低体积的颜色分层。
 * 所有函数均为纯函数，不访问 DOM，不改变源码内容，也不参与保存逻辑。
 */

/** 编辑器支持的语法颜色类别。 */
export type SourceSyntaxKind =
  | 'plain'
  | 'tag'
  | 'attribute'
  | 'string'
  | 'comment'
  | 'content'
  | 'keyword'
  | 'function'
  | 'number'
  | 'property'
  | 'selector'
  | 'operator'

/** 一段连续源码对应的语法类别。字符区间采用左闭右开。 */
export interface SourceSyntaxToken {
  start: number
  end: number
  kind: SourceSyntaxKind
}

/** 搜索结果只需要起止位置；编辑器可额外携带行号而不影响结构兼容。 */
export interface SourceSearchRange {
  start: number
  end: number
}

/** 最终渲染片段：同时携带语法颜色和搜索命中状态。 */
export interface DecoratedSourceSegment {
  text: string
  kind: SourceSyntaxKind
  matchIndex: number
  activeMatch: boolean
}

const JAVASCRIPT_KEYWORDS = new Set([
  'async', 'await', 'break', 'case', 'catch', 'class', 'const', 'continue',
  'debugger', 'default', 'delete', 'do', 'else', 'export', 'extends', 'false',
  'finally', 'for', 'from', 'function', 'get', 'if', 'import', 'in', 'instanceof',
  'let', 'new', 'null', 'of', 'return', 'set', 'static', 'super', 'switch',
  'this', 'throw', 'true', 'try', 'typeof', 'undefined', 'var', 'void', 'while',
  'with', 'yield',
])

function pushToken(
  tokens: SourceSyntaxToken[],
  start: number,
  end: number,
  kind: SourceSyntaxKind,
): void {
  if (end <= start) return
  const previous = tokens.at(-1)
  if (previous && previous.end === start && previous.kind === kind) {
    previous.end = end
    return
  }
  tokens.push({ start, end, kind })
}

function isIdentifierStart(character: string): boolean {
  return Boolean(character) && /[\p{L}_$]/u.test(character)
}

function isIdentifierPart(character: string): boolean {
  return Boolean(character) && /[\p{L}\p{N}_$-]/u.test(character)
}

/** 读取单引号、双引号或模板字符串；反斜线转义不会提前结束字符串。 */
function readQuoted(source: string, start: number, limit: number): number {
  const quote = source[start]
  let cursor = start + 1
  while (cursor < limit) {
    if (source[cursor] === '\\') {
      cursor += 2
      continue
    }
    cursor += 1
    if (source[cursor - 1] === quote) break
  }
  return Math.min(cursor, limit)
}

/** 查找 HTML 标签结束位置，忽略属性字符串内部的 >。 */
function findTagEnd(source: string, start: number): number {
  let cursor = start + 1
  let quote = ''
  while (cursor < source.length) {
    const character = source[cursor]
    if (quote) {
      if (character === '\\') cursor += 2
      else {
        if (character === quote) quote = ''
        cursor += 1
      }
      continue
    }
    if (character === '"' || character === "'") {
      quote = character
      cursor += 1
      continue
    }
    cursor += 1
    if (character === '>') return cursor
  }
  return source.length
}

/** 将标签本身拆成标签名、属性名、属性值和符号。 */
function tokenizeHtmlTag(
  source: string,
  start: number,
  end: number,
  tokens: SourceSyntaxToken[],
): void {
  let cursor = start
  let tagNameSeen = false

  while (cursor < end) {
    const character = source[cursor]

    if (/\s/u.test(character)) {
      const tokenStart = cursor
      while (cursor < end && /\s/u.test(source[cursor])) cursor += 1
      pushToken(tokens, tokenStart, cursor, 'plain')
      continue
    }

    if (character === '"' || character === "'") {
      const tokenEnd = readQuoted(source, cursor, end)
      pushToken(tokens, cursor, tokenEnd, 'string')
      cursor = tokenEnd
      continue
    }

    if (character === '<' || character === '>' || character === '/' || character === '=') {
      const tokenStart = cursor
      if (character === '<' && source[cursor + 1] === '/') cursor += 2
      else if (character === '/' && source[cursor + 1] === '>') cursor += 2
      else cursor += 1
      pushToken(tokens, tokenStart, cursor, 'operator')
      continue
    }

    const tokenStart = cursor
    while (
      cursor < end
      && !/[\s<>/=]/u.test(source[cursor])
      && source[cursor] !== '"'
      && source[cursor] !== "'"
    ) cursor += 1

    if (cursor === tokenStart) {
      pushToken(tokens, cursor, cursor + 1, 'operator')
      cursor += 1
      continue
    }

    pushToken(tokens, tokenStart, cursor, tagNameSeen ? 'attribute' : 'tag')
    tagNameSeen = true
  }
}

/** HTML 标签外的可见文字使用高对比正文色，空白仍保持普通颜色。 */
function tokenizeHtmlContent(
  source: string,
  start: number,
  end: number,
  tokens: SourceSyntaxToken[],
): void {
  let cursor = start
  while (cursor < end) {
    const tokenStart = cursor
    const whitespace = /\s/u.test(source[cursor])
    while (cursor < end && /\s/u.test(source[cursor]) === whitespace) cursor += 1
    pushToken(tokens, tokenStart, cursor, whitespace ? 'plain' : 'content')
  }
}

/** 对 style 标签内部做轻量 CSS 着色。 */
function tokenizeCss(
  source: string,
  start: number,
  end: number,
  tokens: SourceSyntaxToken[],
): void {
  let cursor = start
  let blockDepth = 0

  while (cursor < end) {
    if (source.startsWith('/*', cursor)) {
      const close = source.indexOf('*/', cursor + 2)
      const tokenEnd = close >= 0 && close < end ? close + 2 : end
      pushToken(tokens, cursor, tokenEnd, 'comment')
      cursor = tokenEnd
      continue
    }

    const character = source[cursor]
    if (character === '"' || character === "'") {
      const tokenEnd = readQuoted(source, cursor, end)
      pushToken(tokens, cursor, tokenEnd, 'string')
      cursor = tokenEnd
      continue
    }

    if (/\s/u.test(character)) {
      const tokenStart = cursor
      while (cursor < end && /\s/u.test(source[cursor])) cursor += 1
      pushToken(tokens, tokenStart, cursor, 'plain')
      continue
    }

    if (/\d/u.test(character) || (character === '.' && /\d/u.test(source[cursor + 1] || ''))) {
      const tokenStart = cursor
      while (cursor < end && /[\d.%a-z-]/iu.test(source[cursor])) cursor += 1
      pushToken(tokens, tokenStart, cursor, 'number')
      continue
    }

    if (isIdentifierStart(character) || character === '-') {
      const tokenStart = cursor
      while (cursor < end && isIdentifierPart(source[cursor])) cursor += 1
      let lookahead = cursor
      while (lookahead < end && /\s/u.test(source[lookahead])) lookahead += 1
      const kind: SourceSyntaxKind = blockDepth > 0 && source[lookahead] === ':'
        ? 'property'
        : blockDepth === 0
          ? 'selector'
          : 'plain'
      pushToken(tokens, tokenStart, cursor, kind)
      continue
    }

    if (character === '{') blockDepth += 1
    if (character === '}') blockDepth = Math.max(0, blockDepth - 1)
    pushToken(tokens, cursor, cursor + 1, 'operator')
    cursor += 1
  }
}

/** 对 script 标签内部做轻量 JavaScript 着色。 */
function tokenizeJavascript(
  source: string,
  start: number,
  end: number,
  tokens: SourceSyntaxToken[],
): void {
  let cursor = start

  while (cursor < end) {
    if (source.startsWith('//', cursor)) {
      const newline = source.indexOf('\n', cursor + 2)
      const tokenEnd = newline >= 0 && newline < end ? newline : end
      pushToken(tokens, cursor, tokenEnd, 'comment')
      cursor = tokenEnd
      continue
    }

    if (source.startsWith('/*', cursor)) {
      const close = source.indexOf('*/', cursor + 2)
      const tokenEnd = close >= 0 && close < end ? close + 2 : end
      pushToken(tokens, cursor, tokenEnd, 'comment')
      cursor = tokenEnd
      continue
    }

    const character = source[cursor]
    if (character === '"' || character === "'" || character === '`') {
      const tokenEnd = readQuoted(source, cursor, end)
      pushToken(tokens, cursor, tokenEnd, 'string')
      cursor = tokenEnd
      continue
    }

    if (/\s/u.test(character)) {
      const tokenStart = cursor
      while (cursor < end && /\s/u.test(source[cursor])) cursor += 1
      pushToken(tokens, tokenStart, cursor, 'plain')
      continue
    }

    if (/\d/u.test(character)) {
      const tokenStart = cursor
      while (cursor < end && /[\d._xobabcdef]/iu.test(source[cursor])) cursor += 1
      pushToken(tokens, tokenStart, cursor, 'number')
      continue
    }

    if (isIdentifierStart(character)) {
      const tokenStart = cursor
      cursor += 1
      while (cursor < end && isIdentifierPart(source[cursor])) cursor += 1
      const identifier = source.slice(tokenStart, cursor)
      let lookahead = cursor
      while (lookahead < end && /\s/u.test(source[lookahead])) lookahead += 1
      const kind: SourceSyntaxKind = JAVASCRIPT_KEYWORDS.has(identifier)
        ? 'keyword'
        : source[lookahead] === '('
          ? 'function'
          : 'plain'
      pushToken(tokens, tokenStart, cursor, kind)
      continue
    }

    pushToken(tokens, cursor, cursor + 1, 'operator')
    cursor += 1
  }
}

/**
 * 将完整课件源码标记为连续 token。
 * 对未知或不完整语法保持宽容，任何字符都只用于显示，不会被删除或改写。
 */
export function tokenizeCoursewareSource(source: string): SourceSyntaxToken[] {
  const tokens: SourceSyntaxToken[] = []
  const lowerSource = source.toLowerCase()
  let cursor = 0

  while (cursor < source.length) {
    if (source.startsWith('<!--', cursor)) {
      const close = source.indexOf('-->', cursor + 4)
      const tokenEnd = close >= 0 ? close + 3 : source.length
      pushToken(tokens, cursor, tokenEnd, 'comment')
      cursor = tokenEnd
      continue
    }

    if (source[cursor] !== '<') {
      const nextTag = source.indexOf('<', cursor)
      const tokenEnd = nextTag >= 0 ? nextTag : source.length
      tokenizeHtmlContent(source, cursor, tokenEnd, tokens)
      cursor = tokenEnd
      continue
    }

    const tagStart = cursor
    const tagEnd = findTagEnd(source, tagStart)
    const tagText = source.slice(tagStart, tagEnd)
    tokenizeHtmlTag(source, tagStart, tagEnd, tokens)
    cursor = tagEnd

    const embeddedMatch = tagText.match(/^<\s*(script|style)\b/i)
    const closingTag = /^<\s*\//u.test(tagText)
    const selfClosing = /\/\s*>$/u.test(tagText)
    if (!embeddedMatch || closingTag || selfClosing) continue

    const language = embeddedMatch[1].toLowerCase()
    const closeStart = lowerSource.indexOf(`</${language}`, cursor)
    const bodyEnd = closeStart >= 0 ? closeStart : source.length
    if (language === 'style') tokenizeCss(source, cursor, bodyEnd, tokens)
    else tokenizeJavascript(source, cursor, bodyEnd, tokens)
    cursor = bodyEnd
  }

  return tokens
}

/**
 * 在语法 token 上再叠加搜索命中边界，生成可直接渲染的最小片段。
 * 相邻且颜色、命中状态完全一致的片段会自动合并，减少 React 节点数量。
 */
export function decorateCoursewareSource(
  source: string,
  tokens: SourceSyntaxToken[],
  matches: SourceSearchRange[],
  activeMatchIndex: number,
): DecoratedSourceSegment[] {
  if (!source) return [{ text: '', kind: 'plain', matchIndex: -1, activeMatch: false }]

  const boundaries = new Set<number>([0, source.length])
  for (const token of tokens) {
    boundaries.add(token.start)
    boundaries.add(token.end)
  }
  for (const match of matches) {
    boundaries.add(match.start)
    boundaries.add(match.end)
  }

  const points = [...boundaries].sort((left, right) => left - right)
  const segments: DecoratedSourceSegment[] = []
  let tokenIndex = 0
  let matchIndex = 0

  for (let index = 0; index < points.length - 1; index += 1) {
    const start = points[index]
    const end = points[index + 1]
    if (end <= start) continue

    while (tokenIndex < tokens.length && tokens[tokenIndex].end <= start) tokenIndex += 1
    while (matchIndex < matches.length && matches[matchIndex].end <= start) matchIndex += 1

    const token = tokens[tokenIndex]
    const match = matches[matchIndex]
    const kind = token && token.start <= start && token.end >= end ? token.kind : 'plain'
    const currentMatch = match && match.start <= start && match.end >= end ? matchIndex : -1
    const segment: DecoratedSourceSegment = {
      text: source.slice(start, end),
      kind,
      matchIndex: currentMatch,
      activeMatch: currentMatch === activeMatchIndex,
    }

    const previous = segments.at(-1)
    if (
      previous
      && previous.kind === segment.kind
      && previous.matchIndex === segment.matchIndex
      && previous.activeMatch === segment.activeMatch
    ) previous.text += segment.text
    else segments.push(segment)
  }

  return segments
}

