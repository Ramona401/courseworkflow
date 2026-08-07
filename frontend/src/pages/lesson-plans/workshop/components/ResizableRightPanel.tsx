/**
 * ResizableRightPanel.tsx — 可拖动调整宽度的右侧面板。
 *
 * 适用于备课工坊的教案预览、目录和章节AI修改区域：
 *   - 鼠标或触控笔拖动分隔线调整宽度；
 *   - 双击分隔线恢复默认宽度；
 *   - 键盘左右方向键调整宽度；
 *   - 宽度保存到localStorage；
 *   - 自动为左侧主要工作区保留最低宽度；
 *   - 窄屏时可禁用分隔线并占满可用空间。
 */

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type PointerEvent,
  type ReactNode,
} from 'react'

const DIVIDER_WIDTH = 10
const KEYBOARD_STEP = 24
const KEYBOARD_LARGE_STEP = 64

interface ResizableRightPanelProps {
  /** 用于分别记忆不同页面的面板宽度 */
  storageKey: string
  /** 默认右侧面板宽度 */
  defaultWidth: number
  /** 允许的最小宽度 */
  minWidth: number
  /** 允许的最大宽度 */
  maxWidth: number
  /** 必须为左侧主要工作区保留的宽度 */
  minPrimaryWidth: number
  /** 窄屏时禁用拖动并占满可用空间 */
  disabled?: boolean
  /** 右侧实际内容区域附加样式 */
  panelStyle?: CSSProperties
  children: ReactNode
}

function clamp(
  value: number,
  minimum: number,
  maximum: number,
): number {
  return Math.min(
    Math.max(value, minimum),
    maximum,
  )
}

function readStoredWidth(
  storageKey: string,
  fallback: number,
  minimum: number,
  maximum: number,
): number {
  try {
    const raw = localStorage.getItem(storageKey)
    const parsed = raw
      ? Number.parseInt(raw, 10)
      : Number.NaN

    if (Number.isFinite(parsed)) {
      return clamp(parsed, minimum, maximum)
    }
  } catch {
    // localStorage不可用时使用默认宽度。
  }

  return clamp(fallback, minimum, maximum)
}

export default function ResizableRightPanel({
  storageKey,
  defaultWidth,
  minWidth,
  maxWidth,
  minPrimaryWidth,
  disabled = false,
  panelStyle,
  children,
}: ResizableRightPanelProps) {
  const [panelWidth, setPanelWidth] = useState(() =>
    readStoredWidth(
      storageKey,
      defaultWidth,
      minWidth,
      Math.min(
        maxWidth,
        Math.max(
          minWidth,
          window.innerWidth -
            minPrimaryWidth -
            DIVIDER_WIDTH,
        ),
      ),
    ),
  )
  const [dragging, setDragging] = useState(false)
  const [hovering, setHovering] = useState(false)

  const rootRef = useRef<HTMLDivElement>(null)
  const startXRef = useRef(0)
  const startWidthRef = useRef(panelWidth)
  const bodyCursorRef = useRef('')
  const bodyUserSelectRef = useRef('')

  const getMaximumWidth = useCallback(() => {
    const parentWidth =
      rootRef.current
        ?.parentElement
        ?.getBoundingClientRect()
        .width ||
      window.innerWidth

    return Math.max(
      minWidth,
      Math.min(
        maxWidth,
        parentWidth -
          minPrimaryWidth -
          DIVIDER_WIDTH,
      ),
    )
  }, [
    maxWidth,
    minPrimaryWidth,
    minWidth,
  ])

  const normalizeWidth = useCallback(
    (candidate: number) =>
      clamp(
        candidate,
        minWidth,
        getMaximumWidth(),
      ),
    [
      getMaximumWidth,
      minWidth,
    ],
  )

  const updateWidth = useCallback(
    (candidate: number) => {
      setPanelWidth(
        normalizeWidth(candidate),
      )
    },
    [normalizeWidth],
  )

  const restoreBodyInteraction = useCallback(() => {
    document.body.style.cursor =
      bodyCursorRef.current
    document.body.style.userSelect =
      bodyUserSelectRef.current
  }, [])

  useEffect(() => {
    if (disabled) return

    try {
      localStorage.setItem(
        storageKey,
        String(Math.round(panelWidth)),
      )
    } catch {
      // 宽度记忆失败不影响当前拖动体验。
    }
  }, [
    disabled,
    panelWidth,
    storageKey,
  ])

  useEffect(() => {
    const handleResize = () => {
      setPanelWidth(previous =>
        normalizeWidth(previous),
      )
    }

    window.addEventListener(
      'resize',
      handleResize,
    )

    return () => {
      window.removeEventListener(
        'resize',
        handleResize,
      )
      restoreBodyInteraction()
    }
  }, [
    normalizeWidth,
    restoreBodyInteraction,
  ])

  const beginResize = (
    event: PointerEvent<HTMLDivElement>,
  ) => {
    if (disabled) return

    event.preventDefault()
    event.currentTarget.setPointerCapture(
      event.pointerId,
    )

    startXRef.current = event.clientX
    startWidthRef.current = panelWidth

    bodyCursorRef.current =
      document.body.style.cursor
    bodyUserSelectRef.current =
      document.body.style.userSelect

    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'

    setDragging(true)
  }

  const continueResize = (
    event: PointerEvent<HTMLDivElement>,
  ) => {
    if (!dragging || disabled) return

    event.preventDefault()

    const movement =
      startXRef.current - event.clientX

    updateWidth(
      startWidthRef.current + movement,
    )
  }

  const finishResize = (
    event: PointerEvent<HTMLDivElement>,
  ) => {
    if (!dragging) return

    if (
      event.currentTarget.hasPointerCapture(
        event.pointerId,
      )
    ) {
      event.currentTarget.releasePointerCapture(
        event.pointerId,
      )
    }

    setDragging(false)
    restoreBodyInteraction()
  }

  const resetWidth = () => {
    updateWidth(defaultWidth)
  }

  const handleDividerKeyDown = (
    event: KeyboardEvent<HTMLDivElement>,
  ) => {
    if (disabled) return

    const step = event.shiftKey
      ? KEYBOARD_LARGE_STEP
      : KEYBOARD_STEP

    if (event.key === 'ArrowLeft') {
      event.preventDefault()
      updateWidth(panelWidth + step)
      return
    }

    if (event.key === 'ArrowRight') {
      event.preventDefault()
      updateWidth(panelWidth - step)
      return
    }

    if (
      event.key === 'Home' ||
      event.key === 'Enter'
    ) {
      event.preventDefault()
      resetWidth()
    }
  }

  const emphasized =
    hovering || dragging

  return (
    <div
      ref={rootRef}
      style={{
        height: '100%',
        minWidth: 0,
        display: 'flex',
        flexShrink: disabled ? 1 : 0,
        flexGrow: disabled ? 1 : 0,
        width: disabled
          ? '100%'
          : `${panelWidth + DIVIDER_WIDTH}px`,
        maxWidth: disabled
          ? '100%'
          : `calc(100% - ${minPrimaryWidth}px)`,
        overflow: 'hidden',
      }}
    >
      {!disabled && (
        <div
          role="separator"
          aria-label="调整教案预览宽度"
          aria-orientation="vertical"
          aria-valuemin={minWidth}
          aria-valuemax={maxWidth}
          aria-valuenow={Math.round(panelWidth)}
          tabIndex={0}
          title="拖动调整右侧预览宽度；双击或按Enter恢复默认宽度"
          onPointerDown={beginResize}
          onPointerMove={continueResize}
          onPointerUp={finishResize}
          onPointerCancel={finishResize}
          onLostPointerCapture={() => {
            if (!dragging) return
            setDragging(false)
            restoreBodyInteraction()
          }}
          onDoubleClick={resetWidth}
          onKeyDown={handleDividerKeyDown}
          onMouseEnter={() => setHovering(true)}
          onMouseLeave={() => setHovering(false)}
          style={{
            width: `${DIVIDER_WIDTH}px`,
            height: '100%',
            flexShrink: 0,
            cursor: 'col-resize',
            touchAction: 'none',
            outline: 'none',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: emphasized
              ? 'rgba(79,123,232,0.06)'
              : 'transparent',
            transition: 'background 120ms ease',
          }}
        >
          <div style={{
            width: emphasized ? '3px' : '2px',
            height: emphasized ? '54px' : '38px',
            borderRadius: '999px',
            background: emphasized
              ? '#4F7BE8'
              : '#D1D5DB',
            boxShadow: emphasized
              ? '0 0 0 3px rgba(79,123,232,0.10)'
              : 'none',
            transition:
              'width 120ms ease, height 120ms ease, background 120ms ease',
          }} />
        </div>
      )}

      <div style={{
        height: '100%',
        minWidth: 0,
        width: disabled
          ? '100%'
          : `${panelWidth}px`,
        maxWidth: disabled
          ? '100%'
          : `calc(100% - ${DIVIDER_WIDTH}px)`,
        flex: '1 1 auto',
        boxSizing: 'border-box',
        ...panelStyle,
      }}>
        {children}
      </div>
    </div>
  )
}
