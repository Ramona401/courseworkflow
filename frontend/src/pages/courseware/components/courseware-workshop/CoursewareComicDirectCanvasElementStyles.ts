/**
 * CoursewareComicDirectCanvasElementStyles.ts
 *
 * 漫画覆盖层统一视觉样式：
 *   - 说话气泡主体和尾巴使用同一个SVG闭合路径；
 *   - 整体只填充一次、描边一次，透明时不会暴露内部接缝；
 *   - 编辑画布和普通预览共用相同颜色、透明度与线宽；
 *   - 文字颜色默认根据背景明暗自动选择黑字或白字；
 *   - 教师缩小文本框时保持字号，不由视觉层隐式缩字；
 *   - 气泡内边距收紧，优先把已有留白让给正文；
 *   - 文字层占满框体并在框内垂直居中；
 *   - 教师可调描边宽度，气泡主体与尾巴始终共用同一个值。
 */

import type { CSSProperties } from 'react'
import type { CoursewareComicOverlayElement } from '@/api/coursewares'
import {
  resolveCoursewareComicSpeechBubblePathGeometry,
} from './coursewareComicTailVisual'
import {
  COURSEWARE_COMIC_EDITOR_FONT_SCALE,
  resolveCoursewareComicEditorFontSize,
} from './coursewareComicTypography'

interface ElementPalette {
  background: string
  color: string
  border: string
  borderRadius: CSSProperties['borderRadius']
  boxShadow: string
  stroke: string
  strokeWidth: number
  shapeFilter: string
}

interface RGBColor {
  red: number
  green: number
  blue: number
}

export interface CoursewareComicBubbleTailGeometry {
  shapePath: string
  fill: string
  stroke: string
  strokeWidth: number
  filter: string
}

const HEX_COLOR_PATTERN = /^#[0-9a-fA-F]{6}$/

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(maximum, Math.max(minimum, value))
}

export function normalizeCoursewareComicTextAlign(
  value: string,
): 'left' | 'right' | 'center' | 'justify' {
  switch (value) {
  case 'right':
  case 'center':
  case 'justify':
    return value
  default:
    return 'left'
  }
}

export function normalizeCoursewareComicTextColor(
  value: string,
  fallback = '#111827',
): string {
  const normalized = typeof value === 'string' ? value.trim() : ''
  return HEX_COLOR_PATTERN.test(normalized)
    ? normalized.toUpperCase()
    : fallback
}

export function normalizeCoursewareComicBackgroundOpacity(value: unknown): number {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return 1
  }
  return clamp(value, 0.2, 1)
}

/**
 * 历史缺失或零值按1px处理。说话气泡主体与尾巴共用该值。
 */
export function normalizeCoursewareComicOutlineWidth(value: unknown): number {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return 1
  }

  return Math.round(clamp(value, 0.5, 3) * 4) / 4
}

function parseHexColor(value: string): RGBColor | null {
  const normalized = value.trim()
  if (!HEX_COLOR_PATTERN.test(normalized)) {
    return null
  }

  return {
    red: Number.parseInt(normalized.slice(1, 3), 16),
    green: Number.parseInt(normalized.slice(3, 5), 16),
    blue: Number.parseInt(normalized.slice(5, 7), 16),
  }
}

function backgroundWithOpacity(value: string, opacity: number): string {
  const color = parseHexColor(value)
  if (!color) {
    return value
  }

  return `rgba(${color.red},${color.green},${color.blue},${opacity.toFixed(3)})`
}

function relativeLuminance(value: string): number {
  const color = parseHexColor(value)
  if (!color) {
    return 1
  }

  const channel = (component: number) => {
    const normalized = component / 255
    return normalized <= 0.03928
      ? normalized / 12.92
      : Math.pow((normalized + 0.055) / 1.055, 2.4)
  }

  return (
    0.2126 * channel(color.red) +
    0.7152 * channel(color.green) +
    0.0722 * channel(color.blue)
  )
}

function automaticTextColor(background: string): string {
  return relativeLuminance(background) < 0.46 ? '#FFFFFF' : '#111827'
}

function basePalette(element: CoursewareComicOverlayElement): ElementPalette {
  const bubble =
    element.type === 'speech_bubble' ||
    element.type === 'thought_bubble'
  const question = element.type === 'question_card'

  return {
    background: bubble || question ? '#FFFFFF' : '#0F172A',
    color: bubble || question ? '#111827' : '#FFFFFF',
    border: bubble
      ? '2px solid rgba(30,41,59,0.84)'
      : question
        ? '2px solid rgba(124,58,237,0.55)'
        : '1px solid rgba(255,255,255,0.55)',
    borderRadius: bubble ? 18 : 7,
    boxShadow: '0 6px 18px rgba(15,23,42,0.18)',
    stroke: 'rgba(30,41,59,0.84)',
    strokeWidth: 2,
    shapeFilter: 'drop-shadow(0 6px 9px rgba(15,23,42,0.18))',
  }
}

function paletteFor(element: CoursewareComicOverlayElement): ElementPalette {
  const styleID = element.style_id.trim().toLowerCase()
  let palette = basePalette(element)

  switch (styleID) {
  case 'speech_soft':
    palette = makePalette(
      '#FFFFFF',
      '#312E81',
      '1.5px solid rgba(139,92,246,0.46)',
      22,
      '0 8px 24px rgba(109,40,217,0.14)',
      'rgba(139,92,246,0.66)',
      1.5,
    )
    break
  case 'speech_cloud':
    palette = makePalette(
      '#FDFBFF',
      '#312E81',
      '2px solid rgba(124,58,237,0.64)',
      26,
      '0 9px 25px rgba(109,40,217,0.16)',
      'rgba(124,58,237,0.74)',
      2,
    )
    break
  case 'speech_outline':
    palette = makePalette(
      '#FFFFFF',
      '#111827',
      '2.5px solid rgba(15,23,42,0.92)',
      17,
      '3px 3px 0 rgba(15,23,42,0.16),0 9px 22px rgba(15,23,42,0.14)',
      'rgba(15,23,42,0.92)',
      2.5,
    )
    break
  case 'speech_capsule':
    palette = makePalette(
      '#EFF6FF',
      '#172554',
      '2px solid rgba(37,99,235,0.62)',
      999,
      '0 8px 24px rgba(37,99,235,0.16)',
      'rgba(37,99,235,0.72)',
      2,
    )
    break
  case 'speech_pop':
    palette = makePalette(
      '#FFF7ED',
      '#7C2D12',
      '2.5px solid rgba(234,88,12,0.82)',
      19,
      '4px 4px 0 rgba(251,146,60,0.24),0 10px 24px rgba(124,45,18,0.16)',
      'rgba(234,88,12,0.86)',
      2.5,
    )
    break
  case 'thought_cloud':
  case 'thought_soft':
  case 'thought_outline':
    palette = thoughtPalette(styleID)
    break
  case 'question_blue':
    palette = {
      ...palette,
      background: '#EFF6FF',
      color: '#172554',
      border: '2px solid rgba(37,99,235,0.58)',
      boxShadow: '0 8px 22px rgba(37,99,235,0.14)',
      stroke: 'rgba(37,99,235,0.68)',
    }
    break
  case 'question_orange':
    palette = {
      ...palette,
      background: '#FFF7ED',
      color: '#7C2D12',
      border: '2px solid rgba(234,88,12,0.58)',
      boxShadow: '0 8px 22px rgba(234,88,12,0.14)',
      stroke: 'rgba(234,88,12,0.68)',
    }
    break
  case 'card_light':
    palette = {
      ...palette,
      background: '#FFFFFF',
      color: '#111827',
      border: '1px solid rgba(148,163,184,0.52)',
      stroke: 'rgba(148,163,184,0.62)',
      strokeWidth: 1,
    }
    break
  case 'card_accent':
    palette = {
      ...palette,
      background: '#4C1D95',
      color: '#FFFFFF',
      border: '1px solid rgba(167,139,250,0.72)',
      stroke: 'rgba(167,139,250,0.82)',
      strokeWidth: 1,
    }
    break
  }

  const opacity = normalizeCoursewareComicBackgroundOpacity(
    element.text_style.background_opacity,
  )
  const colorMode = element.text_style.color_mode === 'manual'
    ? 'manual'
    : 'auto'

  return {
    ...palette,
    background: backgroundWithOpacity(palette.background, opacity),
    strokeWidth: element.type === 'speech_bubble'
      ? normalizeCoursewareComicOutlineWidth(
          element.text_style.outline_width,
        )
      : palette.strokeWidth,
    color: colorMode === 'manual'
      ? normalizeCoursewareComicTextColor(
          element.text_style.color,
          palette.color,
        )
      : automaticTextColor(palette.background),
  }
}

function makePalette(
  background: string,
  color: string,
  border: string,
  borderRadius: CSSProperties['borderRadius'],
  boxShadow: string,
  stroke: string,
  strokeWidth: number,
): ElementPalette {
  return {
    background,
    color,
    border,
    borderRadius,
    boxShadow,
    stroke,
    strokeWidth,
    shapeFilter: 'drop-shadow(0 7px 10px rgba(15,23,42,0.18))',
  }
}

function thoughtPalette(styleID: string): ElementPalette {
  const outline = styleID === 'thought_outline'
  const soft = styleID === 'thought_soft'

  return makePalette(
    soft ? '#FFFFFF' : '#F8FAFC',
    soft ? '#475569' : '#1E293B',
    outline
      ? '2.5px solid rgba(51,65,85,0.86)'
      : soft
        ? '1.5px solid rgba(148,163,184,0.58)'
        : '2px solid rgba(71,85,105,0.76)',
    outline
      ? '43% 57% 50% 50% / 55% 45% 55% 45%'
      : soft
        ? '48% 52% 46% 54% / 54% 47% 53% 46%'
        : '46% 54% 49% 51% / 53% 44% 56% 47%',
    outline
      ? '3px 3px 0 rgba(51,65,85,0.15),0 8px 20px rgba(15,23,42,0.13)'
      : '0 8px 22px rgba(100,116,139,0.14)',
    outline
      ? 'rgba(51,65,85,0.86)'
      : soft
        ? 'rgba(148,163,184,0.58)'
        : 'rgba(71,85,105,0.76)',
    outline ? 2.5 : soft ? 1.5 : 2,
  )
}

export function resolveCoursewareComicElementVisual(
  element: CoursewareComicOverlayElement,
): CSSProperties {
  const bubble =
    element.type === 'speech_bubble' ||
    element.type === 'thought_bubble'
  const speech = element.type === 'speech_bubble'
  const question = element.type === 'question_card'
  const palette = paletteFor(element)

  return {
    boxSizing: 'border-box',
    display: 'flex',
    flexDirection: 'column',
    justifyContent: bubble
      ? 'center'
      : question
        ? 'flex-start'
        : 'center',
    padding: 0,
    borderRadius: speech ? 0 : palette.borderRadius,
    border: speech ? 'none' : palette.border,
    background: speech ? 'transparent' : palette.background,
    color: palette.color,
    fontFamily:
      element.text_style.font_family ||
      'Noto Sans SC, sans-serif',
    fontSize:
      `${resolveCoursewareComicEditorFontSize(
        element,
        element.text_style.font_size,
      )}px`,
    fontWeight: element.text_style.font_weight || 600,
    lineHeight: element.text_style.line_height || 1.35,
    textAlign: normalizeCoursewareComicTextAlign(
      element.text_style.align,
    ),
    boxShadow: speech ? 'none' : palette.boxShadow,
  }
}

export function resolveCoursewareComicBubbleTailGeometry(
  element: CoursewareComicOverlayElement,
): CoursewareComicBubbleTailGeometry | null {
  const geometry = resolveCoursewareComicSpeechBubblePathGeometry(element)
  if (!geometry) {
    return null
  }

  const palette = paletteFor(element)
  return {
    ...geometry,
    fill: palette.background,
    stroke: palette.stroke,
    strokeWidth: palette.strokeWidth,
    filter: palette.shapeFilter,
  }
}

function editorPadding(
  designPixels: number,
  minimumPixels: number,
): number {
  return Math.max(
    minimumPixels,
    designPixels / COURSEWARE_COMIC_EDITOR_FONT_SCALE,
  )
}

/**
 * 覆盖层文档中的padding使用1920×1080设计像素。
 *
 * 内边距与字号使用同一显示缩放，但不再预留过大的固定空白。
 * 缩小气泡时优先压缩留白，字号仍由教师的A−、A+或“适配”控制。
 */
export function resolveCoursewareComicTextContentStyle(
  element: CoursewareComicOverlayElement,
): CSSProperties {
  const bubble =
    element.type === 'speech_bubble' ||
    element.type === 'thought_bubble'
  const question = element.type === 'question_card'

  const verticalDesign = bubble ? 10 : question ? 14 : 9
  const horizontalDesign = bubble ? 18 : question ? 22 : 16
  const vertical = editorPadding(
    verticalDesign,
    bubble ? 3 : question ? 4 : 3,
  )
  const horizontal = editorPadding(
    horizontalDesign,
    bubble ? 5 : question ? 6 : 4.5,
  )

  return {
    width: '100%',
    height: '100%',
    minHeight: 0,
    display: 'flex',
    flexDirection: 'column',
    justifyContent: question ? 'flex-start' : 'center',
    boxSizing: 'border-box',
    overflow: 'hidden',
    padding: `${vertical.toFixed(2)}px ${horizontal.toFixed(2)}px`,
  }
}

export const directCanvasElementBaseStyle: CSSProperties = {
  position: 'absolute',
  transformOrigin: 'center',
  overflow: 'visible',
  touchAction: 'none',
}

export const directCanvasDisplayTextStyle: CSSProperties = {
  position: 'relative',
  zIndex: 1,
  width: '100%',
  height: '100%',
  minHeight: 0,
  boxSizing: 'border-box',
  display: 'flex',
  flexDirection: 'column',
  justifyContent: 'center',
  padding: 0,
  overflow: 'hidden',
  whiteSpace: 'pre-wrap',
  overflowWrap: 'anywhere',
  pointerEvents: 'none',
}

export const directCanvasInlineEditorStyle: CSSProperties = {
  position: 'absolute',
  inset: 0,
  zIndex: 2,
  width: '100%',
  height: '100%',
  boxSizing: 'border-box',
  padding: 0,
  resize: 'none',
  border: 'none',
  borderRadius: 'inherit',
  background: 'transparent',
  color: 'inherit',
  caretColor: 'currentColor',
  outline: '2px solid #7C3AED',
  fontSize: 'inherit',
  overflow: 'auto',
}
