/**
 * DiscussionMarkdown.tsx
 *
 * AI讨论消息专用的轻量、安全Markdown渲染组件。
 *
 * 支持的教师可见格式：
 * 1. # 至 ###### 标题，兼容井号后有空格或无空格；
 * 2. **粗体**、__粗体__；
 * 3. *斜体*、_斜体_；
 * 4. `行内代码`；
 * 5. ~~删除线~~；
 * 6. 有序列表和无序列表；
 * 7. 引用块；
 * 8. ``` 围栏代码块；
 * 9. Markdown分隔线。
 *
 * 安全原则：
 * 1. 不使用dangerouslySetInnerHTML；
 * 2. 不执行AI返回的HTML、脚本或事件属性；
 * 3. 所有内容最终都作为React文本节点渲染；
 * 4. 无法识别的Markdown按普通文字展示，不丢失原文。
 */

import {
  Fragment,
  useMemo,
} from 'react'

import type {
  CSSProperties,
  ReactNode,
} from 'react'

interface Props {
  content: string
  compact?: boolean
}

type HeadingLevel =
  | 1
  | 2
  | 3
  | 4
  | 5
  | 6

interface ListItem {
  text: string
  depth: number
}

type MarkdownBlock =
  | {
      type: 'heading'
      level: HeadingLevel
      text: string
    }
  | {
      type: 'paragraph'
      lines: string[]
    }
  | {
      type: 'list'
      ordered: boolean
      items: ListItem[]
    }
  | {
      type: 'quote'
      lines: string[]
    }
  | {
      type: 'code'
      language: string
      code: string
    }
  | {
      type: 'divider'
    }

const headingSizes:
  Record<
    HeadingLevel,
    number
  > = {
    1: 20,
    2: 18,
    3: 16,
    4: 15,
    5: 14,
    6: 14,
  }

const inlineCodeStyle:
  CSSProperties = {
    padding:
      '1px 5px',
    borderRadius:
      4,
    background:
      '#E5E7EB',
    color:
      '#BE123C',
    fontFamily:
      'Menlo, Monaco, Consolas, monospace',
    fontSize:
      '0.9em',
  }

/**
 * 将列表前导空格转换成有限的视觉缩进。
 *
 * 最多展示三级缩进，防止异常AI输出把内容推到气泡外。
 */
function resolveListDepth(
  indentation: string,
): number {
  const spaces =
    indentation.replace(
      /\t/g,
      '    ',
    ).length

  return Math.min(
    3,
    Math.floor(
      spaces / 2,
    ),
  )
}

/**
 * 把原始讨论正文解析成块级结构。
 *
 * 本函数只返回纯数据，不生成HTML。
 */
function parseMarkdownBlocks(
  source: string,
): MarkdownBlock[] {
  const normalized =
    String(
      source || '',
    ).replace(
      /\r\n?/g,
      '\n',
    )

  const lines =
    normalized.split(
      '\n',
    )

  const blocks:
    MarkdownBlock[] = []

  let paragraphLines:
    string[] = []

  let quoteLines:
    string[] = []

  let listItems:
    ListItem[] = []

  let listOrdered:
    boolean | null = null

  let insideCode =
    false

  let codeLanguage =
    ''

  let codeLines:
    string[] = []

  const flushParagraph =
    (): void => {
      if (
        paragraphLines.length
        === 0
      ) {
        return
      }

      blocks.push({
        type:
          'paragraph',
        lines:
          paragraphLines,
      })

      paragraphLines = []
    }

  const flushQuote =
    (): void => {
      if (
        quoteLines.length
        === 0
      ) {
        return
      }

      blocks.push({
        type:
          'quote',
        lines:
          quoteLines,
      })

      quoteLines = []
    }

  const flushList =
    (): void => {
      if (
        listOrdered === null
        || listItems.length === 0
      ) {
        listOrdered = null
        listItems = []
        return
      }

      blocks.push({
        type:
          'list',
        ordered:
          listOrdered,
        items:
          listItems,
      })

      listOrdered = null
      listItems = []
    }

  const flushTextBlocks =
    (): void => {
      flushParagraph()
      flushQuote()
      flushList()
    }

  for (
    const line
    of lines
  ) {
    const trimmed =
      line.trim()

    const fenceMatch =
      trimmed.match(
        /^```([A-Za-z0-9_+-]*)$/,
      )

    if (fenceMatch) {
      if (!insideCode) {
        flushTextBlocks()

        insideCode = true
        codeLanguage =
          fenceMatch[1] || ''
        codeLines = []
      } else {
        blocks.push({
          type:
            'code',
          language:
            codeLanguage,
          code:
            codeLines.join(
              '\n',
            ),
        })

        insideCode = false
        codeLanguage = ''
        codeLines = []
      }

      continue
    }

    if (insideCode) {
      codeLines.push(
        line,
      )
      continue
    }

    if (trimmed === '') {
      flushTextBlocks()
      continue
    }

    if (
      /^(---+|\*\*\*+|___+)$/.test(
        trimmed,
      )
    ) {
      flushTextBlocks()

      blocks.push({
        type:
          'divider',
      })

      continue
    }

    const headingMatch =
      line.match(
        /^\s*(#{1,6})(?!#)\s*(\S.*?)\s*$/,
      )

    if (headingMatch) {
      flushTextBlocks()

      blocks.push({
        type:
          'heading',
        level:
          headingMatch[1]
            .length as HeadingLevel,
        text:
          headingMatch[2],
      })

      continue
    }

    const orderedMatch =
      line.match(
        /^(\s*)\d+[.)]\s+(.+)$/,
      )

    if (orderedMatch) {
      flushParagraph()
      flushQuote()

      if (
        listOrdered !== null
        && listOrdered !== true
      ) {
        flushList()
      }

      listOrdered = true

      listItems.push({
        text:
          orderedMatch[2],
        depth:
          resolveListDepth(
            orderedMatch[1],
          ),
      })

      continue
    }

    const unorderedMatch =
      line.match(
        /^(\s*)[-*+]\s+(.+)$/,
      )

    if (unorderedMatch) {
      flushParagraph()
      flushQuote()

      if (
        listOrdered !== null
        && listOrdered !== false
      ) {
        flushList()
      }

      listOrdered = false

      listItems.push({
        text:
          unorderedMatch[2],
        depth:
          resolveListDepth(
            unorderedMatch[1],
          ),
      })

      continue
    }

    const quoteMatch =
      line.match(
        /^\s*>\s?(.*)$/,
      )

    if (quoteMatch) {
      flushParagraph()
      flushList()

      quoteLines.push(
        quoteMatch[1],
      )

      continue
    }

    flushQuote()
    flushList()

    paragraphLines.push(
      line,
    )
  }

  if (insideCode) {
    /**
     * AI偶尔漏掉结束围栏。
     *
     * 为避免吞掉后续内容，仍按代码块展示已收集的文本。
     */
    blocks.push({
      type:
        'code',
      language:
        codeLanguage,
      code:
        codeLines.join(
          '\n',
        ),
    })
  } else {
    flushTextBlocks()
  }

  return blocks
}

interface InlineToken {
  start: number
  end: number
  type:
    | 'bold'
    | 'italic'
    | 'code'
    | 'strike'
  inner: string
}

/**
 * 查找字符串中最先出现的行内Markdown标记。
 */
function findNextInlineToken(
  content: string,
): InlineToken | null {
  const patterns:
    Array<{
      type:
        InlineToken['type']
      regex:
        RegExp
    }> = [
      {
        type:
          'code',
        regex:
          /`([^`\n]+)`/,
      },
      {
        type:
          'bold',
        regex:
          /\*\*([^*\n]+)\*\*/,
      },
      {
        type:
          'bold',
        regex:
          /__([^_\n]+)__/,
      },
      {
        type:
          'strike',
        regex:
          /~~([^~\n]+)~~/,
      },
      {
        type:
          'italic',
        regex:
          /\*([^*\n]+)\*/,
      },
      {
        type:
          'italic',
        regex:
          /_([^_\n]+)_/,
      },
    ]

  const candidates:
    InlineToken[] = []

  for (
    const pattern
    of patterns
  ) {
    const match =
      pattern.regex.exec(
        content,
      )

    if (
      !match
      || match.index < 0
    ) {
      continue
    }

    candidates.push({
      start:
        match.index,
      end:
        match.index
        + match[0].length,
      type:
        pattern.type,
      inner:
        match[1],
    })
  }

  if (
    candidates.length === 0
  ) {
    return null
  }

  candidates.sort(
    (
      left,
      right,
    ) => {
      if (
        left.start
        !== right.start
      ) {
        return (
          left.start
          - right.start
        )
      }

      /**
       * 同一位置出现多个候选时，
       * 优先选择覆盖范围更长的粗体等标记。
       */
      return (
        right.end
        - left.end
      )
    },
  )

  return candidates[0]
}

/**
 * 将一行中的Markdown标记渲染为安全React节点。
 */
function renderInlineMarkdown(
  content: string,
  keyPrefix: string,
): ReactNode[] {
  const nodes:
    ReactNode[] = []

  let remaining =
    content

  let sequence =
    0

  while (
    remaining.length > 0
  ) {
    const token =
      findNextInlineToken(
        remaining,
      )

    if (!token) {
      nodes.push(
        <Fragment
          key={
            `${keyPrefix}-text-${sequence}`
          }
        >
          {remaining}
        </Fragment>,
      )

      break
    }

    if (
      token.start > 0
    ) {
      nodes.push(
        <Fragment
          key={
            `${keyPrefix}-plain-${sequence}`
          }
        >
          {
            remaining.slice(
              0,
              token.start,
            )
          }
        </Fragment>,
      )
    }

    const tokenKey =
      `${keyPrefix}-token-${sequence}`

    if (
      token.type === 'bold'
    ) {
      nodes.push(
        <strong
          key={
            tokenKey
          }
          style={{
            fontWeight:
              800,
          }}
        >
          {
            renderInlineMarkdown(
              token.inner,
              `${tokenKey}-inner`,
            )
          }
        </strong>,
      )
    } else if (
      token.type === 'italic'
    ) {
      nodes.push(
        <em
          key={
            tokenKey
          }
        >
          {
            renderInlineMarkdown(
              token.inner,
              `${tokenKey}-inner`,
            )
          }
        </em>,
      )
    } else if (
      token.type === 'strike'
    ) {
      nodes.push(
        <span
          key={
            tokenKey
          }
          style={{
            textDecoration:
              'line-through',
            opacity:
              0.72,
          }}
        >
          {
            renderInlineMarkdown(
              token.inner,
              `${tokenKey}-inner`,
            )
          }
        </span>,
      )
    } else {
      nodes.push(
        <code
          key={
            tokenKey
          }
          style={
            inlineCodeStyle
          }
        >
          {token.inner}
        </code>,
      )
    }

    remaining =
      remaining.slice(
        token.end,
      )

    sequence += 1
  }

  return nodes
}

/**
 * 渲染包含显式换行的文本块。
 */
function renderTextLines(
  lines: string[],
  keyPrefix: string,
): ReactNode[] {
  const result:
    ReactNode[] = []

  lines.forEach(
    (
      line,
      index,
    ) => {
      result.push(
        <Fragment
          key={
            `${keyPrefix}-line-${index}`
          }
        >
          {
            renderInlineMarkdown(
              line,
              `${keyPrefix}-${index}`,
            )
          }
        </Fragment>,
      )

      if (
        index
        < lines.length - 1
      ) {
        result.push(
          <br
            key={
              `${keyPrefix}-break-${index}`
            }
          />,
        )
      }
    },
  )

  return result
}

export default function DiscussionMarkdown({
  content,
  compact = false,
}: Props) {
  const blocks =
    useMemo(
      () =>
        parseMarkdownBlocks(
          content,
        ),
      [
        content,
      ],
    )

  if (
    blocks.length === 0
  ) {
    return null
  }

  return (
    <div
      style={{
        color:
          'inherit',
        fontSize:
          compact
            ? 12
            : 13,
        lineHeight:
          1.7,
        wordBreak:
          'break-word',
      }}
    >
      {blocks.map(
        (
          block,
          blockIndex,
        ) => {
          const blockKey =
            `discussion-markdown-${blockIndex}`

          if (
            block.type === 'heading'
          ) {
            return (
              <div
                key={
                  blockKey
                }
                style={{
                  margin:
                    blockIndex === 0
                      ? '0 0 8px'
                      : '13px 0 8px',
                  fontSize:
                    compact
                      ? Math.max(
                          13,
                          headingSizes[
                            block.level
                          ] - 2,
                        )
                      : headingSizes[
                          block.level
                        ],
                  fontWeight:
                    800,
                  lineHeight:
                    1.42,
                }}
              >
                {
                  renderInlineMarkdown(
                    block.text,
                    `${blockKey}-heading`,
                  )
                }
              </div>
            )
          }

          if (
            block.type === 'paragraph'
          ) {
            return (
              <p
                key={
                  blockKey
                }
                style={{
                  margin:
                    '0 0 9px',
                  lineHeight:
                    1.72,
                }}
              >
                {
                  renderTextLines(
                    block.lines,
                    `${blockKey}-paragraph`,
                  )
                }
              </p>
            )
          }

          if (
            block.type === 'list'
          ) {
            const ListTag =
              block.ordered
                ? 'ol'
                : 'ul'

            return (
              <ListTag
                key={
                  blockKey
                }
                style={{
                  margin:
                    '5px 0 11px',
                  paddingLeft:
                    block.ordered
                      ? 25
                      : 22,
                }}
              >
                {
                  block.items.map(
                    (
                      item,
                      itemIndex,
                    ) => (
                      <li
                        key={
                          `${blockKey}-item-${itemIndex}`
                        }
                        style={{
                          margin:
                            '5px 0',
                          marginLeft:
                            item.depth
                            * 15,
                          paddingLeft:
                            2,
                          lineHeight:
                            1.7,
                        }}
                      >
                        {
                          renderInlineMarkdown(
                            item.text,
                            `${blockKey}-item-${itemIndex}`,
                          )
                        }
                      </li>
                    ),
                  )
                }
              </ListTag>
            )
          }

          if (
            block.type === 'quote'
          ) {
            return (
              <blockquote
                key={
                  blockKey
                }
                style={{
                  margin:
                    '7px 0 11px',
                  padding:
                    '8px 11px',
                  borderLeft:
                    '3px solid #A78BFA',
                  borderRadius:
                    '0 7px 7px 0',
                  background:
                    '#F5F3FF',
                  color:
                    '#4C1D95',
                  lineHeight:
                    1.68,
                }}
              >
                {
                  renderTextLines(
                    block.lines,
                    `${blockKey}-quote`,
                  )
                }
              </blockquote>
            )
          }

          if (
            block.type === 'code'
          ) {
            return (
              <div
                key={
                  blockKey
                }
                style={{
                  margin:
                    '8px 0 11px',
                }}
              >
                {block.language && (
                  <div
                    style={{
                      padding:
                        '5px 9px',
                      borderRadius:
                        '7px 7px 0 0',
                      background:
                        '#374151',
                      color:
                        '#D1D5DB',
                      fontSize:
                        10,
                      fontFamily:
                        'Menlo, Monaco, Consolas, monospace',
                    }}
                  >
                    {block.language}
                  </div>
                )}

                <pre
                  style={{
                    margin:
                      0,
                    padding:
                      '10px 12px',
                    borderRadius:
                      block.language
                        ? '0 0 7px 7px'
                        : 7,
                    background:
                      '#111827',
                    color:
                      '#E5E7EB',
                    fontSize:
                      11,
                    lineHeight:
                      1.55,
                    whiteSpace:
                      'pre-wrap',
                    wordBreak:
                      'break-word',
                    overflowX:
                      'auto',
                    fontFamily:
                      'Menlo, Monaco, Consolas, monospace',
                  }}
                >
                  <code>
                    {block.code}
                  </code>
                </pre>
              </div>
            )
          }

          return (
            <hr
              key={
                blockKey
              }
              style={{
                margin:
                  '13px 0',
                border:
                  'none',
                borderTop:
                  '1px solid #D1D5DB',
              }}
            />
          )
        },
      )}
    </div>
  )
}
