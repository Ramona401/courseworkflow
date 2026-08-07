/**
 * lessonDocumentImages.ts — 教案正文图片引用定位工具。
 *
 * 只修改平台当前正文里的 Markdown 图片引用。
 * 不修改原始 Word 母版，不删除物理图片文件，也不修改历史版本。
 */

import type {
  RenderedMarkdownImage,
} from '@/pages/lesson-plans/plan-detail/components/planDetailConstants'

export interface RemoveMarkdownImageResult {
  content: string
  removed: boolean
}

/**
 * 删除指定正文范围内第 occurrence 次出现的准确图片引用。
 *
 * 图片独占一行时连同行尾换行删除；
 * 图片与文字同行时只删除图片 Markdown，保留同行文字。
 */
export function removeMarkdownImageOccurrence(
  content: string,
  image: RenderedMarkdownImage,
  rangeStart = 0,
  rangeEnd = content.length,
): RemoveMarkdownImageResult {
  if (
    !content ||
    !image.markdown ||
    image.occurrence < 1
  ) {
    return {
      content,
      removed: false,
    }
  }

  const start = Math.max(
    0,
    Math.min(
      rangeStart,
      content.length,
    ),
  )

  const end = Math.max(
    start,
    Math.min(
      rangeEnd,
      content.length,
    ),
  )

  const scopedContent =
    content.slice(start, end)

  let searchStart = 0
  let occurrence = 0
  let localTargetStart = -1

  while (
    searchStart <= scopedContent.length
  ) {
    const found =
      scopedContent.indexOf(
        image.markdown,
        searchStart,
      )

    if (found < 0) break

    occurrence += 1

    if (
      occurrence === image.occurrence
    ) {
      localTargetStart = found
      break
    }

    searchStart =
      found +
      Math.max(
        image.markdown.length,
        1,
      )
  }

  if (localTargetStart < 0) {
    return {
      content,
      removed: false,
    }
  }

  const targetStart =
    start + localTargetStart

  const targetEnd =
    targetStart +
    image.markdown.length

  let deleteStart = targetStart
  let deleteEnd = targetEnd

  const previousNewline =
    content.lastIndexOf(
      '\n',
      Math.max(
        targetStart - 1,
        0,
      ),
    )

  const lineStart =
    previousNewline >= 0
      ? previousNewline + 1
      : 0

  const followingNewline =
    content.indexOf(
      '\n',
      targetEnd,
    )

  const lineEnd =
    followingNewline >= 0
      ? followingNewline
      : content.length

  const completeLine =
    content
      .slice(
        lineStart,
        lineEnd,
      )
      .replace(/\r$/, '')

  if (
    completeLine.trim() ===
    image.markdown.trim()
  ) {
    deleteStart = lineStart
    deleteEnd =
      followingNewline >= 0
        ? followingNewline + 1
        : lineEnd
  }

  return {
    content:
      content.slice(0, deleteStart) +
      content.slice(deleteEnd),
    removed: true,
  }
}
