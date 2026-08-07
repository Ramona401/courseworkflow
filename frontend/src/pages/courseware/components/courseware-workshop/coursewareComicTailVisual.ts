/**
 * coursewareComicTailVisual.ts
 *
 * 说话气泡主体与尾巴统一外轮廓几何：
 *   - 气泡主体和尾巴由同一个闭合SVG path完成填充与描边；
 *   - 尾巴根部直接替换主体边缘的一小段，不出现内部接缝；
 *   - 几何先在1920×1080设计像素中计算，再转换为100×100 SVG；
 *   - 横向或纵向改变尺寸时，四角仍保持相同的真实圆弧半径；
 *   - 根部宽度按气泡短边自适应并保持克制比例；
 *   - 路径使用短尾巴可见尖端，编辑控制点继续保留人物语义目标。
 *
 * 本文件不修改草稿、不处理指针事件，也不执行网络请求。
 */

import type {
  CoursewareComicOverlayElement,
} from '@/api/coursewares'

import {
  resolveCoursewareComicTailOrigin,
  resolveCoursewareComicVisibleTailTarget,
} from './coursewareComicTailEditing'

interface TailPoint {
  x: number
  y: number
}

type TailEdge = 'top' | 'right' | 'bottom' | 'left'

export interface CoursewareComicSpeechBubblePathGeometry {
  shapePath: string
}

const DESIGN_CANVAS_WIDTH = 1920
const DESIGN_CANVAS_HEIGHT = 1080
const BODY_INSET = 1

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(maximum, Math.max(minimum, value))
}

function designSize(
  element: CoursewareComicOverlayElement,
): TailPoint {
  return {
    x: Math.max(32, element.width * DESIGN_CANVAS_WIDTH),
    y: Math.max(24, element.height * DESIGN_CANVAS_HEIGHT),
  }
}

function svgPoint(
  point: TailPoint,
  size: TailPoint,
): TailPoint {
  return {
    x: point.x / size.x * 100,
    y: point.y / size.y * 100,
  }
}

function formatPoint(point: TailPoint): string {
  return `${point.x.toFixed(3)},${point.y.toFixed(3)}`
}

function moveCommand(
  point: TailPoint,
  size: TailPoint,
): string {
  return `M ${formatPoint(svgPoint(point, size))}`
}

function lineCommand(
  point: TailPoint,
  size: TailPoint,
): string {
  return `L ${formatPoint(svgPoint(point, size))}`
}

function quadraticCommand(
  control: TailPoint,
  endpoint: TailPoint,
  size: TailPoint,
): string {
  return [
    'Q',
    formatPoint(svgPoint(control, size)),
    formatPoint(svgPoint(endpoint, size)),
  ].join(' ')
}

function cubicCommand(
  first: TailPoint,
  second: TailPoint,
  endpoint: TailPoint,
  size: TailPoint,
): string {
  return [
    'C',
    formatPoint(svgPoint(first, size)),
    formatPoint(svgPoint(second, size)),
    formatPoint(svgPoint(endpoint, size)),
  ].join(' ')
}

function resolveTailEdge(origin: TailPoint): TailEdge {
  if (origin.y === 0) return 'top'
  if (origin.x === 1) return 'right'
  if (origin.y === 1) return 'bottom'
  return 'left'
}

/**
 * 返回设计像素中的正圆圆角半径。
 *
 * 横纵方向共用同一个像素半径；最终分别换算到SVG横纵坐标，
 * 因此长方形气泡不会再把圆角拉成椭圆。
 */
function resolveSpeechBubbleRadius(
  styleID: string,
  size: TailPoint,
): number {
  let radius = 30

  switch (styleID.trim().toLowerCase()) {
  case 'speech_capsule':
    radius = 42
    break
  case 'speech_cloud':
    radius = 38
    break
  case 'speech_soft':
    radius = 34
    break
  case 'speech_outline':
    radius = 26
    break
  case 'speech_pop':
    radius = 28
    break
  }

  return clamp(
    radius,
    6,
    Math.max(6, Math.min(size.x, size.y) / 2 - 4),
  )
}

function resolveTailHalfBase(size: TailPoint): number {
  return clamp(Math.min(size.x, size.y) * 0.028, 5, 9)
}

function resolveTailBase(
  origin: TailPoint,
  edge: TailEdge,
  halfBase: number,
  radius: number,
  size: TailPoint,
): { entry: TailPoint; exit: TailPoint } {
  const left = BODY_INSET
  const top = BODY_INSET
  const right = size.x - BODY_INSET
  const bottom = size.y - BODY_INSET

  if (edge === 'top' || edge === 'bottom') {
    const centerX = clamp(
      origin.x * size.x,
      left + radius + halfBase,
      right - radius - halfBase,
    )
    const y = edge === 'top' ? top : bottom
    const first = { x: centerX - halfBase, y }
    const second = { x: centerX + halfBase, y }

    return edge === 'top'
      ? { entry: first, exit: second }
      : { entry: second, exit: first }
  }

  const centerY = clamp(
    origin.y * size.y,
    top + radius + halfBase,
    bottom - radius - halfBase,
  )
  const x = edge === 'left' ? left : right
  const first = { x, y: centerY - halfBase }
  const second = { x, y: centerY + halfBase }

  return edge === 'right'
    ? { entry: first, exit: second }
    : { entry: second, exit: first }
}

function resolveLocalTarget(
  element: CoursewareComicOverlayElement,
  target: TailPoint,
  size: TailPoint,
): TailPoint {
  return {
    x: clamp(
      (target.x - element.x) * DESIGN_CANVAS_WIDTH,
      -size.x * 2,
      size.x * 3,
    ),
    y: clamp(
      (target.y - element.y) * DESIGN_CANVAS_HEIGHT,
      -size.y * 2,
      size.y * 3,
    ),
  }
}

function buildTailSegment(
  entry: TailPoint,
  exit: TailPoint,
  target: TailPoint,
  size: TailPoint,
): string {
  const center = {
    x: (entry.x + exit.x) / 2,
    y: (entry.y + exit.y) / 2,
  }
  const delta = {
    x: target.x - center.x,
    y: target.y - center.y,
  }
  const distance = Math.max(1, Math.hypot(delta.x, delta.y))
  const direction = {
    x: delta.x / distance,
    y: delta.y / distance,
  }
  const tangent = {
    x: exit.x - entry.x,
    y: exit.y - entry.y,
  }
  const tangentLength = Math.max(0.001, Math.hypot(tangent.x, tangent.y))
  const tangentUnit = {
    x: tangent.x / tangentLength,
    y: tangent.y / tangentLength,
  }
  const baseControl = Math.min(distance * 0.30, 70)
  const tipControl = Math.min(distance * 0.13, 24)
  const tipHalf = 0.8

  const firstControl = {
    x: entry.x + direction.x * baseControl,
    y: entry.y + direction.y * baseControl,
  }
  const secondControl = {
    x: exit.x + direction.x * baseControl,
    y: exit.y + direction.y * baseControl,
  }
  const firstTipControl = {
    x: target.x - direction.x * tipControl - tangentUnit.x * tipHalf,
    y: target.y - direction.y * tipControl - tangentUnit.y * tipHalf,
  }
  const secondTipControl = {
    x: target.x - direction.x * tipControl + tangentUnit.x * tipHalf,
    y: target.y - direction.y * tipControl + tangentUnit.y * tipHalf,
  }

  return [
    cubicCommand(
      firstControl,
      firstTipControl,
      target,
      size,
    ),
    cubicCommand(
      secondTipControl,
      secondControl,
      exit,
      size,
    ),
  ].join(' ')
}

function buildUnifiedBubblePath(
  edge: TailEdge,
  entry: TailPoint,
  exit: TailPoint,
  target: TailPoint,
  radius: number,
  size: TailPoint,
): string {
  const left = BODY_INSET
  const top = BODY_INSET
  const right = size.x - BODY_INSET
  const bottom = size.y - BODY_INSET
  const tail = buildTailSegment(entry, exit, target, size)

  const topLeftStart = { x: left + radius, y: top }
  const topRightStart = { x: right - radius, y: top }
  const rightTopEnd = { x: right, y: top + radius }
  const rightBottomStart = { x: right, y: bottom - radius }
  const bottomRightEnd = { x: right - radius, y: bottom }
  const bottomLeftStart = { x: left + radius, y: bottom }
  const leftBottomEnd = { x: left, y: bottom - radius }
  const leftTopStart = { x: left, y: top + radius }

  switch (edge) {
  case 'top':
    return [
      moveCommand(topLeftStart, size),
      lineCommand(entry, size),
      tail,
      lineCommand(topRightStart, size),
      quadraticCommand({ x: right, y: top }, rightTopEnd, size),
      lineCommand(rightBottomStart, size),
      quadraticCommand({ x: right, y: bottom }, bottomRightEnd, size),
      lineCommand(bottomLeftStart, size),
      quadraticCommand({ x: left, y: bottom }, leftBottomEnd, size),
      lineCommand(leftTopStart, size),
      quadraticCommand({ x: left, y: top }, topLeftStart, size),
      'Z',
    ].join(' ')

  case 'right':
    return [
      moveCommand(topLeftStart, size),
      lineCommand(topRightStart, size),
      quadraticCommand({ x: right, y: top }, rightTopEnd, size),
      lineCommand(entry, size),
      tail,
      lineCommand(rightBottomStart, size),
      quadraticCommand({ x: right, y: bottom }, bottomRightEnd, size),
      lineCommand(bottomLeftStart, size),
      quadraticCommand({ x: left, y: bottom }, leftBottomEnd, size),
      lineCommand(leftTopStart, size),
      quadraticCommand({ x: left, y: top }, topLeftStart, size),
      'Z',
    ].join(' ')

  case 'bottom':
    return [
      moveCommand(topLeftStart, size),
      lineCommand(topRightStart, size),
      quadraticCommand({ x: right, y: top }, rightTopEnd, size),
      lineCommand(rightBottomStart, size),
      quadraticCommand({ x: right, y: bottom }, bottomRightEnd, size),
      lineCommand(entry, size),
      tail,
      lineCommand(bottomLeftStart, size),
      quadraticCommand({ x: left, y: bottom }, leftBottomEnd, size),
      lineCommand(leftTopStart, size),
      quadraticCommand({ x: left, y: top }, topLeftStart, size),
      'Z',
    ].join(' ')

  default:
    return [
      moveCommand(topLeftStart, size),
      lineCommand(topRightStart, size),
      quadraticCommand({ x: right, y: top }, rightTopEnd, size),
      lineCommand(rightBottomStart, size),
      quadraticCommand({ x: right, y: bottom }, bottomRightEnd, size),
      lineCommand(bottomLeftStart, size),
      quadraticCommand({ x: left, y: bottom }, leftBottomEnd, size),
      lineCommand(entry, size),
      tail,
      lineCommand(leftTopStart, size),
      quadraticCommand({ x: left, y: top }, topLeftStart, size),
      'Z',
    ].join(' ')
  }
}

export function resolveCoursewareComicSpeechBubblePathGeometry(
  element: CoursewareComicOverlayElement,
): CoursewareComicSpeechBubblePathGeometry | null {
  if (element.type !== 'speech_bubble') {
    return null
  }

  const origin = resolveCoursewareComicTailOrigin(element)
  const target =
    resolveCoursewareComicVisibleTailTarget(element)

  if (!origin || !target) {
    return null
  }

  const size = designSize(element)
  const edge = resolveTailEdge(origin)
  const radius = resolveSpeechBubbleRadius(element.style_id, size)
  const halfBase = resolveTailHalfBase(size)
  const base = resolveTailBase(
    origin,
    edge,
    halfBase,
    radius,
    size,
  )
  const localTarget = resolveLocalTarget(element, target, size)

  return {
    shapePath: buildUnifiedBubblePath(
      edge,
      base.entry,
      base.exit,
      localTarget,
      radius,
      size,
    ),
  }
}
