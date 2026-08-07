/**
 * SourceCodeEditor.tsx — 课件源码轻量编辑器
 *
 * 不引入 Monaco / CodeMirror，避免主包明显变大；本组件由 PagePreviewBlock
 * 使用 React.lazy 按需加载。提供老师高频需要的默认搜索框、替换、行号、
 * HTML/CSS/JavaScript 轻量语法着色、正文高亮和右侧 Minimap 小窗定位。
 *
 * 快捷键仍保留：Ctrl/Command+F 搜索；Ctrl/Command+H 替换；
 * Enter/Shift+Enter 下一处/上一处；Esc 关闭搜索；编辑态 Tab 插入两个空格。
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type {
  ChangeEvent,
  KeyboardEvent,
  MouseEvent as ReactMouseEvent,
  PointerEvent,
  SyntheticEvent,
} from 'react'
import {
  decorateCoursewareSource,
  tokenizeCoursewareSource,
} from './sourceCodeSyntax'
import {
  drawSourceMinimap,
  findSourceMatches,
  getSourceLineColumn,
  SOURCE_LINE_HEIGHT,
  SOURCE_MINIMAP_WIDTH,
} from './sourceCodeEditorUtils'
import { SOURCE_EDITOR_CSS } from './sourceCodeEditorStyles'

interface SourceCodeEditorProps {
  value: string
  onChange?: (value: string) => void
  readOnly?: boolean
  disabled?: boolean
  height?: number
}

export default function SourceCodeEditor({
  value,
  onChange,
  readOnly = false,
  disabled = false,
  height = 520,
}: SourceCodeEditorProps) {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const highlightRef = useRef<HTMLPreElement | null>(null)
  const lineNumberRef = useRef<HTMLPreElement | null>(null)
  const minimapHostRef = useRef<HTMLDivElement | null>(null)
  const minimapCanvasRef = useRef<HTMLCanvasElement | null>(null)
  const searchInputRef = useRef<HTMLInputElement | null>(null)
  const replaceInputRef = useRef<HTMLInputElement | null>(null)
  const minimapDraggingRef = useRef(false)
  const lastSearchControlRef = useRef<'search' | 'replace'>('search')

  /** 搜索栏默认展开，进入源码视图后无需快捷键即可直接输入。 */
  const [searchOpen, setSearchOpen] = useState(true)
  const [replaceOpen, setReplaceOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [replacement, setReplacement] = useState('')
  const [matchCase, setMatchCase] = useState(false)
  const [wholeWord, setWholeWord] = useState(false)
  const [activeMatch, setActiveMatch] = useState(0)
  const [cursorPosition, setCursorPosition] = useState(0)

  const lines = useMemo(() => value.split('\n'), [value])
  const lineCount = Math.max(1, lines.length)
  const gutterWidth = Math.max(50, 28 + String(lineCount).length * 8)
  const matches = useMemo(
    () => findSourceMatches(value, query, matchCase, wholeWord),
    [value, query, matchCase, wholeWord],
  )
  const syntaxTokens = useMemo(
    () => tokenizeCoursewareSource(value),
    [value],
  )
  const decoratedSegments = useMemo(
    () => decorateCoursewareSource(
      value,
      syntaxTokens,
      matches,
      activeMatch,
    ),
    [activeMatch, matches, syntaxTokens, value],
  )
  const cursor = useMemo(
    () => getSourceLineColumn(value, cursorPosition),
    [cursorPosition, value],
  )

  /** 将当前源码密度、搜索命中和可见视口绘制到右侧小窗。 */
  const drawMinimap = useCallback(() => {
    const textarea = textareaRef.current
    const host = minimapHostRef.current
    const canvas = minimapCanvasRef.current
    if (!textarea || !host || !canvas) return

    drawSourceMinimap({
      textarea,
      host,
      canvas,
      lines,
      matches,
      activeMatch,
    })
  }, [activeMatch, lines, matches])

  /** 同步行号、彩色展示层和 Minimap 的滚动位置。 */
  const syncScroll = useCallback(() => {
    const textarea = textareaRef.current
    if (!textarea) return

    if (lineNumberRef.current) {
      lineNumberRef.current.scrollTop = textarea.scrollTop
    }
    if (highlightRef.current) {
      highlightRef.current.scrollTop = textarea.scrollTop
      highlightRef.current.scrollLeft = textarea.scrollLeft
    }
    drawMinimap()
  }, [drawMinimap])

  useEffect(() => drawMinimap(), [drawMinimap, height])

  useEffect(() => {
    const host = minimapHostRef.current
    if (!host) return
    const observer = new ResizeObserver(drawMinimap)
    observer.observe(host)
    return () => observer.disconnect()
  }, [drawMinimap])

  useEffect(() => {
    if (matches.length === 0) setActiveMatch(0)
    else setActiveMatch(index => Math.min(index, matches.length - 1))
  }, [matches.length])

  /** 初次打开源码视图时，搜索栏可见且自动获得焦点。 */
  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      searchInputRef.current?.focus({ preventScroll: true })
    })
    return () => cancelAnimationFrame(frame)
  }, [])

  const openSearch = useCallback((showReplace: boolean) => {
    setSearchOpen(true)
    setReplaceOpen(showReplace && !readOnly)
    lastSearchControlRef.current = 'search'
    requestAnimationFrame(() => {
      searchInputRef.current?.focus({ preventScroll: true })
      searchInputRef.current?.select()
    })
  }, [readOnly])

  const closeSearch = useCallback(() => {
    setSearchOpen(false)
    setReplaceOpen(false)
    textareaRef.current?.focus()
  }, [])

  /**
   * 定位搜索结果。
   * preserveSearchFocus=true 时保持搜索框或替换框焦点，不让 textarea 抢走输入。
   */
  const focusMatch = useCallback((
    requestedIndex: number,
    preserveSearchFocus = false,
  ) => {
    if (matches.length === 0) return

    const normalized = (
      requestedIndex % matches.length + matches.length
    ) % matches.length
    const match = matches[normalized]
    setActiveMatch(normalized)

    requestAnimationFrame(() => {
      const textarea = textareaRef.current
      if (!textarea) return

      if (!preserveSearchFocus) textarea.focus({ preventScroll: true })
      textarea.setSelectionRange(match.start, match.end)
      setCursorPosition(match.start)
      textarea.scrollTop = Math.max(
        0,
        (match.line - 1) * SOURCE_LINE_HEIGHT - textarea.clientHeight / 2,
      )
      syncScroll()

      if (preserveSearchFocus) {
        const target = lastSearchControlRef.current === 'replace'
          ? replaceInputRef.current
          : searchInputRef.current
        target?.focus({ preventScroll: true })
      }
    })
  }, [matches, syncScroll])

  useEffect(() => {
    if (searchOpen && query && matches.length > 0) {
      focusMatch(activeMatch, true)
    } else {
      drawMinimap()
    }
    // 搜索条件变化时定位当前命中；activeMatch由focusMatch内部维护，避免重复跳转。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query, matchCase, wholeWord, searchOpen, matches.length])

  const handleEditorKeyDown = (
    event: KeyboardEvent<HTMLTextAreaElement>,
  ) => {
    const command = event.ctrlKey || event.metaKey

    if (command && event.key.toLowerCase() === 'f') {
      event.preventDefault()
      openSearch(false)
      return
    }

    if (command && event.key.toLowerCase() === 'h' && !readOnly) {
      event.preventDefault()
      openSearch(true)
      return
    }

    if (event.key === 'Escape' && searchOpen) {
      event.preventDefault()
      closeSearch()
      return
    }

    if (event.key === 'Tab' && !readOnly && !disabled && onChange) {
      event.preventDefault()
      const textarea = event.currentTarget
      const start = textarea.selectionStart
      const end = textarea.selectionEnd
      onChange(value.slice(0, start) + '  ' + value.slice(end))

      requestAnimationFrame(() => {
        textarea.selectionStart = start + 2
        textarea.selectionEnd = start + 2
        setCursorPosition(start + 2)
      })
    }
  }

  const handleSearchKeyDown = (
    event: KeyboardEvent<HTMLInputElement>,
  ) => {
    if (event.key === 'Enter') {
      event.preventDefault()
      focusMatch(
        activeMatch + (event.shiftKey ? -1 : 1),
        true,
      )
    } else if (event.key === 'Escape') {
      event.preventDefault()
      closeSearch()
    }
  }

  const replaceCurrent = () => {
    if (readOnly || disabled || !onChange || matches.length === 0) return

    const match = matches[activeMatch]
    lastSearchControlRef.current = 'replace'
    onChange(
      value.slice(0, match.start)
      + replacement
      + value.slice(match.end),
    )
    setCursorPosition(match.start + replacement.length)

    requestAnimationFrame(() => {
      textareaRef.current?.setSelectionRange(
        match.start,
        match.start + replacement.length,
      )
      replaceInputRef.current?.focus({ preventScroll: true })
    })
  }

  const replaceAll = () => {
    if (readOnly || disabled || !onChange || matches.length === 0) return

    let cursor = 0
    let next = ''
    for (const match of matches) {
      next += value.slice(cursor, match.start) + replacement
      cursor = match.end
    }

    lastSearchControlRef.current = 'replace'
    onChange(next + value.slice(cursor))
    setActiveMatch(0)
    setCursorPosition(0)
    requestAnimationFrame(() => {
      replaceInputRef.current?.focus({ preventScroll: true })
    })
  }

  const scrollFromMinimap = useCallback((clientY: number) => {
    const textarea = textareaRef.current
    const host = minimapHostRef.current
    if (!textarea || !host) return

    const rect = host.getBoundingClientRect()
    const y = Math.max(0, Math.min(clientY - rect.top, rect.height))
    const ratio = rect.height > 0 ? y / rect.height : 0
    const maxScroll = Math.max(
      0,
      textarea.scrollHeight - textarea.clientHeight,
    )
    textarea.scrollTop = Math.max(
      0,
      Math.min(
        maxScroll,
        ratio * textarea.scrollHeight - textarea.clientHeight / 2,
      ),
    )
    syncScroll()
  }, [syncScroll])

  const handleMinimapPointerDown = (
    event: PointerEvent<HTMLCanvasElement>,
  ) => {
    minimapDraggingRef.current = true
    event.currentTarget.setPointerCapture(event.pointerId)
    scrollFromMinimap(event.clientY)
  }

  const handleMinimapPointerMove = (
    event: PointerEvent<HTMLCanvasElement>,
  ) => {
    if (minimapDraggingRef.current) scrollFromMinimap(event.clientY)
  }

  const handleMinimapPointerUp = (
    event: PointerEvent<HTMLCanvasElement>,
  ) => {
    minimapDraggingRef.current = false
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
  }

  const matchLabel = matches.length > 0
    ? `${activeMatch + 1}/${matches.length}`
    : query
      ? '0/0'
      : '—'
  const gridStyle = {
    gridTemplateColumns: `${gutterWidth}px minmax(0,1fr) ${SOURCE_MINIMAP_WIDTH}px`,
    height,
  }

  return (
    <div
      className={`tedna-source-editor ${readOnly ? 'is-readonly' : 'is-editing'}`}
    >
      <style>{SOURCE_EDITOR_CSS}</style>

      {searchOpen ? (
        <div
          className="tedna-source-search"
          style={{ right: SOURCE_MINIMAP_WIDTH + 10 }}
        >
          <div className="tedna-source-search-row">
            <input
              ref={searchInputRef}
              value={query}
              onFocus={() => {
                lastSearchControlRef.current = 'search'
              }}
              onChange={(event: ChangeEvent<HTMLInputElement>) => {
                setQuery(event.target.value)
                setActiveMatch(0)
              }}
              onKeyDown={handleSearchKeyDown}
              placeholder="搜索正文、样式或函数"
              aria-label="搜索课件源码"
            />
            <span
              className={`tedna-source-match-count ${query && matches.length === 0 ? 'is-empty' : ''}`}
            >
              {matchLabel}
            </span>
            <button
              type="button"
              className={matchCase ? 'is-active' : ''}
              onClick={() => setMatchCase(current => !current)}
              title="区分大小写"
            >
              Aa
            </button>
            <button
              type="button"
              className={wholeWord ? 'is-active' : ''}
              onClick={() => setWholeWord(current => !current)}
              title="全词匹配"
            >
              W
            </button>
            <button
              type="button"
              onClick={() => focusMatch(activeMatch - 1, true)}
              disabled={matches.length === 0}
              title="上一处（Shift+Enter）"
            >
              ↑
            </button>
            <button
              type="button"
              onClick={() => focusMatch(activeMatch + 1, true)}
              disabled={matches.length === 0}
              title="下一处（Enter）"
            >
              ↓
            </button>
            {!readOnly && (
              <button
                type="button"
                onClick={() => setReplaceOpen(current => !current)}
                title="展开或收起替换"
              >
                ⇄
              </button>
            )}
            <button
              type="button"
              onClick={closeSearch}
              title="关闭搜索（Esc）"
            >
              ×
            </button>
          </div>

          {replaceOpen && !readOnly && (
            <div className="tedna-source-search-row tedna-source-replace-row">
              <input
                ref={replaceInputRef}
                value={replacement}
                onFocus={() => {
                  lastSearchControlRef.current = 'replace'
                }}
                onChange={(event: ChangeEvent<HTMLInputElement>) => {
                  setReplacement(event.target.value)
                }}
                onKeyDown={(event: KeyboardEvent<HTMLInputElement>) => {
                  if (event.key === 'Enter') {
                    event.preventDefault()
                    replaceCurrent()
                  }
                  if (event.key === 'Escape') {
                    event.preventDefault()
                    closeSearch()
                  }
                }}
                placeholder="替换为"
                aria-label="替换课件源码"
              />
              <button
                type="button"
                className="tedna-source-action"
                onClick={replaceCurrent}
                disabled={disabled || matches.length === 0}
              >
                替换
              </button>
              <button
                type="button"
                className="tedna-source-action"
                onClick={replaceAll}
                disabled={disabled || matches.length === 0}
              >
                全部替换
              </button>
            </div>
          )}
        </div>
      ) : (
        <button
          type="button"
          className="tedna-source-search-reopen"
          style={{ right: SOURCE_MINIMAP_WIDTH + 10 }}
          onClick={() => openSearch(false)}
        >
          🔎 搜索
        </button>
      )}

      <div className="tedna-source-grid" style={gridStyle}>
        <pre
          ref={lineNumberRef}
          aria-hidden="true"
          className="tedna-source-lines"
        >
          {Array.from(
            { length: lineCount },
            (_, index) => index + 1,
          ).join('\n')}
        </pre>

        <div className="tedna-source-code-pane">
          <pre
            ref={highlightRef}
            aria-hidden="true"
            className="tedna-source-highlight"
          >
            {decoratedSegments.map((segment, index) => (
              <span
                key={`${index}-${segment.kind}-${segment.matchIndex}`}
                className={[
                  `tedna-token-${segment.kind}`,
                  segment.matchIndex >= 0 ? 'tedna-source-match' : '',
                  segment.activeMatch ? 'is-current-match' : '',
                ].filter(Boolean).join(' ')}
              >
                {segment.text}
              </span>
            ))}
          </pre>

          <textarea
            ref={textareaRef}
            value={value}
            onChange={(event: ChangeEvent<HTMLTextAreaElement>) => {
              if (!readOnly && !disabled && onChange) {
                onChange(event.target.value)
              }
            }}
            onKeyDown={handleEditorKeyDown}
            onScroll={syncScroll}
            onSelect={(event: SyntheticEvent<HTMLTextAreaElement>) => {
              setCursorPosition(event.currentTarget.selectionStart)
            }}
            onClick={(event: ReactMouseEvent<HTMLTextAreaElement>) => {
              setCursorPosition(event.currentTarget.selectionStart)
            }}
            onKeyUp={(event: KeyboardEvent<HTMLTextAreaElement>) => {
              setCursorPosition(event.currentTarget.selectionStart)
            }}
            readOnly={readOnly || disabled}
            wrap="off"
            spellCheck={false}
            aria-label={readOnly
              ? '课件源码只读查看器'
              : '课件源码编辑器'}
            className={disabled ? 'is-disabled' : ''}
          />
        </div>

        <div
          ref={minimapHostRef}
          className="tedna-source-minimap"
          title="代码小窗定位：点击或拖动可快速跳转"
        >
          <canvas
            ref={minimapCanvasRef}
            onPointerDown={handleMinimapPointerDown}
            onPointerMove={handleMinimapPointerMove}
            onPointerUp={handleMinimapPointerUp}
            onPointerCancel={handleMinimapPointerUp}
          />
        </div>
      </div>

      <div className="tedna-source-status">
        <span>第 {cursor.line} 行，第 {cursor.column} 列</span>
        <span>共 {lineCount} 行</span>
        <span>HTML · CSS · JavaScript 分色</span>
        <span className="tedna-source-shortcuts">
          搜索栏默认展开
          {!readOnly ? ' · Ctrl/⌘+H 替换 · Tab=2空格' : ''}
          {' · 右侧小窗点击定位'}
        </span>
      </div>
    </div>
  )
}

