/**
 * lessonDocumentStructure.ts — 前端教案目录与段落结构解析。
 *
 * 该规则与后端utils/lesson_plan_section.go保持一致：
 *   - Markdown显式标题进入目录。
 *   - 常见中文教案栏目进入目录。
 *   - 普通“1. 教师展示图片”仍是正文有序列表。
 *   - “1. 教学目标”及明显活动、任务标题可进入目录。
 *
 * 前端解析只用于展示和生成定位参数。
 * 最终应用时后端会重新解析数据库正式正文并复核版本和哈希。
 */

import type {
  LessonPlanSectionLocator,
} from '@/api/lesson-plan-section-rewrite'

export const FULL_DOCUMENT_HEADING = '__FULL_DOCUMENT__'

export interface LessonDocumentSection {
  id: string
  title: string
  headingText: string
  level: number
  headingPath: string[]
  occurrence: number
  startOffset: number
  contentStartOffset: number
  endOffset: number
  bodyMarkdown: string
  locator: LessonPlanSectionLocator
}

export interface LessonDocumentStructure {
  preambleMarkdown: string
  sections: LessonDocumentSection[]
}

interface TextLine {
  start: number
  next: number
  trim: string
}

interface ParsedHeading {
  line: TextLine
  title: string
  level: number
}

const markdownHeadingPattern = /^(#{1,6})[ \t]+(.+?)\s*$/
const chinesePrimaryHeadingPattern =
  /^([一二三四五六七八九十百]+)[、.．][ \t]*(.+?)\s*$/
const chineseSecondaryHeadingPattern =
  /^[（(]([一二三四五六七八九十百]+)[）)][ \t]*(.+?)\s*$/
const numberedHeadingPattern =
  /^(?:([0-9]+)[、.．]|[（(]([0-9]+)[）)])[ \t]*(.+?)\s*$/
const chapterHeadingPattern =
  /^第[一二三四五六七八九十百0-9]+[章节部分单元][ \t、:：.-]*(.*?)\s*$/
const numberedHeadingSemanticPrefixPattern =
  /^(活动|任务|环节|步骤|问题|情境|案例|实验|练习|小结|作业|板书|评价|导入|新授|探究|合作|展示|交流|总结)[一二三四五六七八九十百0-9]*[：:—-]?[ \t]*/

const plainHeadingTitles = new Set([
  '教材分析',
  '课标分析',
  '课程标准',
  '学情分析',
  '教学内容',
  '教学目标',
  '学习目标',
  '核心素养目标',
  '教学重点',
  '教学难点',
  '教学重难点',
  '教学准备',
  '教学方法',
  '教学策略',
  '教学过程',
  '教学活动',
  '教学环节',
  '教师活动',
  '学生活动',
  '设计意图',
  '时间分配',
  '导入新课',
  '新课导入',
  '新授',
  '新课讲授',
  '探究活动',
  '合作学习',
  '课堂练习',
  '巩固练习',
  '课堂小结',
  '总结提升',
  '评价设计',
  '作业设计',
  '板书设计',
  '教学反思',
])

/** 解析整份教案结构。 */
export function parseLessonDocumentStructure(
  content: string,
): LessonDocumentStructure {
  const lines = splitTextLines(content)
  const headings: ParsedHeading[] = []

  for (const line of lines) {
    const parsed = parseHeadingLine(line.trim)
    if (!parsed) continue

    headings.push({
      line,
      title: parsed.title,
      level: parsed.level,
    })
  }

  if (headings.length === 0) {
    return {
      preambleMarkdown: '',
      sections: [
        {
          id: buildSectionID(FULL_DOCUMENT_HEADING, 1),
          title: '教案正文',
          headingText: FULL_DOCUMENT_HEADING,
          level: 1,
          headingPath: ['教案正文'],
          occurrence: 1,
          startOffset: 0,
          contentStartOffset: 0,
          endOffset: content.length,
          bodyMarkdown: content,
          locator: {
            heading_text: FULL_DOCUMENT_HEADING,
            occurrence: 1,
          },
        },
      ],
    }
  }

  const preambleMarkdown = content.slice(
    0,
    headings[0].line.start,
  )
  const occurrenceByHeading = new Map<string, number>()
  const pathStack: Array<{ level: number; title: string }> = []
  const sections: LessonDocumentSection[] = []

  headings.forEach((heading, index) => {
    while (
      pathStack.length > 0 &&
      pathStack[pathStack.length - 1].level >= heading.level
    ) {
      pathStack.pop()
    }

    const headingPath = [
      ...pathStack.map(item => item.title),
      heading.title,
    ]

    pathStack.push({
      level: heading.level,
      title: heading.title,
    })

    const headingText = heading.line.trim
    const occurrence =
      (occurrenceByHeading.get(headingText) || 0) + 1

    occurrenceByHeading.set(headingText, occurrence)

    const endOffset = index + 1 < headings.length
      ? headings[index + 1].line.start
      : content.length

    const contentStartOffset = Math.min(
      heading.line.next,
      endOffset,
    )

    sections.push({
      id: buildSectionID(headingText, occurrence),
      title: heading.title,
      headingText,
      level: heading.level,
      headingPath,
      occurrence,
      startOffset: heading.line.start,
      contentStartOffset,
      endOffset,
      bodyMarkdown: content.slice(
        contentStartOffset,
        endOffset,
      ),
      locator: {
        heading_text: headingText,
        occurrence,
      },
    })
  })

  return {
    preambleMarkdown,
    sections,
  }
}

/** 将正文拆成带字符偏移的行。 */
function splitTextLines(content: string): TextLine[] {
  if (!content) {
    return [{
      start: 0,
      next: 0,
      trim: '',
    }]
  }

  const lines: TextLine[] = []
  let start = 0

  while (start < content.length) {
    const newlineIndex = content.indexOf('\n', start)
    const end = newlineIndex >= 0
      ? newlineIndex
      : content.length
    const next = newlineIndex >= 0
      ? newlineIndex + 1
      : content.length

    const text = content
      .slice(start, end)
      .replace(/\r$/, '')

    lines.push({
      start,
      next,
      trim: text.trim(),
    })

    start = next
  }

  return lines
}

/** 判断一行是否是目录标题。 */
function parseHeadingLine(
  raw: string,
): { level: number; title: string } | null {
  const line = raw.trim()
  if (!line) return null

  const markdownMatch = line.match(markdownHeadingPattern)
  if (markdownMatch) {
    const title = cleanHeadingTitle(markdownMatch[2])
    return title
      ? { level: markdownMatch[1].length, title }
      : null
  }

  const visibleLine = unwrapHeadingEmphasis(line)

  const chinesePrimaryMatch =
    visibleLine.match(chinesePrimaryHeadingPattern)
  if (chinesePrimaryMatch) {
    const title = cleanHeadingTitle(chinesePrimaryMatch[2])
    return title ? { level: 2, title } : null
  }

  const chineseSecondaryMatch =
    visibleLine.match(chineseSecondaryHeadingPattern)
  if (chineseSecondaryMatch) {
    const title = cleanHeadingTitle(chineseSecondaryMatch[2])
    return title ? { level: 3, title } : null
  }

  const numberedMatch =
    visibleLine.match(numberedHeadingPattern)
  if (numberedMatch) {
    const title = cleanHeadingTitle(numberedMatch[3])
    if (!isLikelyNumberedHeading(title)) return null

    return {
      level: numberedMatch[1] ? 2 : 3,
      title,
    }
  }

  const chapterMatch = visibleLine.match(chapterHeadingPattern)
  if (chapterMatch) {
    const title =
      cleanHeadingTitle(chapterMatch[1]) ||
      cleanHeadingTitle(visibleLine)

    return title ? { level: 1, title } : null
  }

  const plainTitle = visibleLine
    .trim()
    .replace(/[：:]$/, '')

  return plainHeadingTitles.has(plainTitle)
    ? { level: 2, title: plainTitle }
    : null
}

/** 数字编号只有明显属于教案结构时才进入目录。 */
function isLikelyNumberedHeading(title: string): boolean {
  const cleaned = cleanHeadingTitle(title)
  if (!cleaned) return false
  if (plainHeadingTitles.has(cleaned)) return true
  if ([...cleaned].length > 40) return false
  if (/[。！？；]/.test(cleaned)) return false

  return numberedHeadingSemanticPrefixPattern.test(cleaned)
}

function unwrapHeadingEmphasis(value: string): string {
  const trimmed = value.trim()

  if (
    trimmed.length >= 4 &&
    trimmed.startsWith('**') &&
    trimmed.endsWith('**')
  ) {
    return trimmed.slice(2, -2).trim()
  }

  if (
    trimmed.length >= 4 &&
    trimmed.startsWith('__') &&
    trimmed.endsWith('__')
  ) {
    return trimmed.slice(2, -2).trim()
  }

  return trimmed
}

function cleanHeadingTitle(value: string): string {
  const cleaned = unwrapHeadingEmphasis(value)
    .trim()
    .replace(/[：:]$/, '')
    .trim()

  return [...cleaned].slice(0, 120).join('')
}

/** 生成仅用于DOM和React键的稳定ID。 */
function buildSectionID(
  headingText: string,
  occurrence: number,
): string {
  const input = `${headingText}\u001f${occurrence}`
  let hash = 2166136261

  for (let index = 0; index < input.length; index += 1) {
    hash ^= input.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }

  return `lp-section-${(hash >>> 0).toString(16).padStart(8, '0')}`
}
