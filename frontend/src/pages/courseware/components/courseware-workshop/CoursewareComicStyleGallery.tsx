/**
 * CoursewareComicStyleGallery.tsx
 *
 * 使用SVG示例图选择覆盖层样式：
 *   - 不再依赖浏览器原生select；
 *   - 通过createPortal挂载到document.body，避免画布overflow裁剪；
 *   - 使用fixed定位并限制在可视区，远程桌面和缩放下仍可见；
 *   - 示例图是主要识别方式，文字仅提供辅助名称和无障碍信息。
 */

import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react'
import type {
  CSSProperties,
  PointerEvent,
} from 'react'
import { createPortal } from 'react-dom'
import type {
  CoursewareComicOverlayElement,
} from '@/api/coursewares'
import {
  coursewareComicStyleControlLabel,
  coursewareComicStyleOptions,
} from './coursewareComicStyleOptions'

interface Props {
  element: CoursewareComicOverlayElement
  onChange: (styleID: string) => void
}

interface Position {
  left: number
  top: number
}

interface PreviewPalette {
  fill: string
  stroke: string
  text: string
  strokeWidth: number
}

const WIDTH = 372
const MARGIN = 10

export default function CoursewareComicStyleGallery({
  element,
  onChange,
}: Props) {
  const buttonRef = useRef<HTMLButtonElement | null>(null)
  const panelRef = useRef<HTMLDivElement | null>(null)
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState<Position>({
    left: MARGIN,
    top: MARGIN,
  })

  const options = coursewareComicStyleOptions(element)
  const current =
    options.find(item => item.value === element.style_id) ||
    options[0]

  const updatePosition = () => {
    const button = buttonRef.current
    if (!button) return

    const bounds = button.getBoundingClientRect()
    const height =
      panelRef.current?.getBoundingClientRect().height ||
      286
    const below =
      window.innerHeight - bounds.bottom >= height + MARGIN

    setPosition({
      left: Math.min(
        Math.max(MARGIN, bounds.left),
        Math.max(
          MARGIN,
          window.innerWidth - WIDTH - MARGIN,
        ),
      ),
      top: Math.max(
        MARGIN,
        Math.min(
          below ? bounds.bottom + 6 : bounds.top - height - 6,
          window.innerHeight - height - MARGIN,
        ),
      ),
    })
  }

  useLayoutEffect(() => {
    if (!open) return
    updatePosition()
    const frame = window.requestAnimationFrame(updatePosition)
    return () => window.cancelAnimationFrame(frame)
  }, [open, element.id, options.length])

  useEffect(() => {
    if (!open) return

    const closeOutside = (event: globalThis.PointerEvent) => {
      const target = event.target
      if (!(target instanceof Node)) return
      if (
        buttonRef.current?.contains(target) ||
        panelRef.current?.contains(target)
      ) {
        return
      }
      setOpen(false)
    }

    const closeByKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false)
    }

    window.addEventListener('resize', updatePosition)
    window.addEventListener('scroll', updatePosition, true)
    document.addEventListener('pointerdown', closeOutside, true)
    document.addEventListener('keydown', closeByKey)

    return () => {
      window.removeEventListener('resize', updatePosition)
      window.removeEventListener('scroll', updatePosition, true)
      document.removeEventListener(
        'pointerdown',
        closeOutside,
        true,
      )
      document.removeEventListener('keydown', closeByKey)
    }
  }, [open])

  const stopPointer = (event: PointerEvent<HTMLElement>) => {
    event.stopPropagation()
  }

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        aria-haspopup="dialog"
        aria-expanded={open}
        title="查看气泡或卡片样式示例图"
        onPointerDown={stopPointer}
        onClick={event => {
          event.stopPropagation()
          setOpen(value => !value)
        }}
        style={triggerStyle}
      >
        <StylePreview styleID={current.value} compact />
        <span style={triggerLabelStyle}>
          {coursewareComicStyleControlLabel(element)}
        </span>
        <span aria-hidden="true">▾</span>
      </button>

      {open &&
        typeof document !== 'undefined' &&
        createPortal(
          <div
            ref={panelRef}
            role="dialog"
            aria-label="选择气泡或卡片样式"
            style={{
              ...galleryStyle,
              left: position.left,
              top: position.top,
            }}
            onPointerDown={stopPointer}
            onClick={event => event.stopPropagation()}
          >
            <div style={galleryHeaderStyle}>
              选择样式示例
            </div>

            <div
              role="listbox"
              aria-label="样式示例"
              style={galleryGridStyle}
            >
              {options.map(option => {
                const selected =
                  option.value === element.style_id

                return (
                  <button
                    key={option.value}
                    type="button"
                    role="option"
                    aria-selected={selected}
                    title={option.description}
                    onClick={() => {
                      onChange(option.value)
                      setOpen(false)
                    }}
                    style={{
                      ...optionStyle,
                      borderColor: selected
                        ? '#7C3AED'
                        : '#E2E8F0',
                      background: selected
                        ? '#F5F3FF'
                        : '#FFFFFF',
                      boxShadow: selected
                        ? '0 0 0 2px rgba(124,58,237,0.14)'
                        : 'none',
                    }}
                  >
                    <StylePreview styleID={option.value} />
                    <span style={optionLabelStyle}>
                      {option.label}
                    </span>
                  </button>
                )
              })}
            </div>
          </div>,
          document.body,
        )}
    </>
  )
}

function StylePreview({
  styleID,
  compact = false,
}: {
  styleID: string
  compact?: boolean
}) {
  const id = styleID.trim().toLowerCase()
  const palette = previewPalette(id)
  const thought = id.startsWith('thought_')
  const question = id.startsWith('question_')
  const card = id.startsWith('card_')
  const cloud =
    id === 'speech_cloud' ||
    id === 'thought_cloud' ||
    id === 'thought_soft' ||
    id === 'thought_outline'
  const pop = id === 'speech_pop'

  return (
    <svg
      viewBox="0 0 120 72"
      preserveAspectRatio="none"
      aria-hidden="true"
      style={{
        display: 'block',
        width: compact ? 38 : '100%',
        height: compact ? 22 : 66,
      }}
    >
      {cloud ? (
        <path
          d="M18 25 C13 15 26 8 37 13 C45 4 61 7 64 17 C77 12 91 21 86 33 C97 40 91 54 78 55 C76 67 61 69 52 62 C42 70 27 65 28 55 C15 57 7 47 13 38 C8 32 11 27 18 25 Z"
          fill={palette.fill}
          stroke={palette.stroke}
          strokeWidth={palette.strokeWidth}
          strokeLinejoin="round"
        />
      ) : pop ? (
        <path
          d="M14 9 C29 5 42 10 55 7 C71 4 96 7 103 18 C108 29 103 41 106 51 C108 61 96 67 81 65 C66 69 52 65 39 68 C24 70 10 64 8 53 C6 42 11 33 8 24 C6 16 8 11 14 9 Z"
          fill={palette.fill}
          stroke={palette.stroke}
          strokeWidth={palette.strokeWidth}
          strokeLinejoin="round"
        />
      ) : (
        <rect
          x={card ? 10 : 8}
          y={card ? 10 : 8}
          width={card ? 100 : 104}
          height={card ? 52 : 48}
          rx={
            id === 'speech_capsule'
              ? 26
              : question
                ? 10
                : card
                  ? 8
                  : 15
          }
          fill={palette.fill}
          stroke={palette.stroke}
          strokeWidth={palette.strokeWidth}
        />
      )}

      {!card && !question && !thought && (
        <path
          d="M36 55 C31 60 26 63 19 66 C25 58 27 52 30 48"
          fill={palette.fill}
          stroke={palette.stroke}
          strokeWidth={palette.strokeWidth}
          strokeLinejoin="round"
          strokeLinecap="round"
        />
      )}

      {thought && (
        <>
          <circle
            cx="27"
            cy="61"
            r="4.5"
            fill={palette.fill}
            stroke={palette.stroke}
            strokeWidth={palette.strokeWidth}
          />
          <circle
            cx="18"
            cy="67"
            r="2.5"
            fill={palette.fill}
            stroke={palette.stroke}
            strokeWidth={palette.strokeWidth}
          />
        </>
      )}

      {question ? (
        <text
          x="60"
          y="47"
          textAnchor="middle"
          fontSize="29"
          fontWeight="800"
          fill={palette.text}
        >
          ?
        </text>
      ) : (
        <>
          <rect
            x="29"
            y="26"
            width="62"
            height="5"
            rx="2.5"
            fill={palette.text}
            opacity="0.76"
          />
          <rect
            x="37"
            y="37"
            width="46"
            height="4"
            rx="2"
            fill={palette.text}
            opacity="0.45"
          />
        </>
      )}
    </svg>
  )
}

function previewPalette(id: string): PreviewPalette {
  switch (id) {
  case 'speech_soft':
    return palette('#FFFFFF', '#A78BFA', '#312E81', 1.5)
  case 'speech_cloud':
    return palette('#FDFBFF', '#8B5CF6', '#312E81', 2)
  case 'speech_outline':
    return palette('#FFFFFF', '#0F172A', '#111827', 2.6)
  case 'speech_capsule':
  case 'question_blue':
    return palette('#EFF6FF', '#2563EB', '#172554', 2)
  case 'speech_pop':
  case 'question_orange':
    return palette('#FFF7ED', '#EA580C', '#7C2D12', 2.4)
  case 'thought_cloud':
  case 'thought_soft':
  case 'thought_outline':
    return palette(
      '#F8FAFC',
      '#475569',
      '#1E293B',
      id === 'thought_outline' ? 2.5 : 1.8,
    )
  case 'question_purple':
    return palette('#F3E8FF', '#7E22CE', '#3B0764', 2)
  case 'card_dark':
    return palette('#1E293B', '#64748B', '#FFFFFF', 1.4)
  case 'card_accent':
    return palette('#4C1D95', '#A78BFA', '#FFFFFF', 1.5)
  default:
    return palette('#FFFFFF', '#334155', '#111827', 2)
  }
}

function palette(
  fill: string,
  stroke: string,
  text: string,
  strokeWidth: number,
): PreviewPalette {
  return { fill, stroke, text, strokeWidth }
}

const triggerStyle: CSSProperties = {
  height: 29,
  minWidth: 82,
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  gap: 5,
  padding: '2px 7px 2px 4px',
  borderRadius: 6,
  border: '1px solid #CBD5E1',
  background: '#FFFFFF',
  color: '#334155',
  fontSize: 9,
  fontWeight: 800,
  cursor: 'pointer',
}

const triggerLabelStyle: CSSProperties = {
  maxWidth: 42,
  overflow: 'hidden',
  textOverflow: 'ellipsis',
}

const galleryStyle: CSSProperties = {
  position: 'fixed',
  width: WIDTH,
  maxHeight: 'min(430px,calc(100vh - 20px))',
  overflowY: 'auto',
  boxSizing: 'border-box',
  padding: 10,
  borderRadius: 12,
  border: '1px solid #CBD5E1',
  background: 'rgba(248,250,252,0.99)',
  boxShadow: '0 18px 48px rgba(15,23,42,0.28)',
  zIndex: 100000,
}

const galleryHeaderStyle: CSSProperties = {
  marginBottom: 8,
  color: '#334155',
  fontSize: 11,
  fontWeight: 900,
}

const galleryGridStyle: CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(2,minmax(0,1fr))',
  gap: 8,
}

const optionStyle: CSSProperties = {
  minWidth: 0,
  padding: 6,
  borderRadius: 9,
  border: '1px solid #E2E8F0',
  cursor: 'pointer',
  textAlign: 'center',
}

const optionLabelStyle: CSSProperties = {
  display: 'block',
  marginTop: 3,
  color: '#334155',
  fontSize: 9,
  fontWeight: 800,
}
