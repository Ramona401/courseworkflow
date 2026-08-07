/**
 * coursewareComicTailEditing.ts
 *
 * 漫画说话气泡尾巴编辑纯函数模块：
 *   - 兼容历史只有target_x和target_y或type=single的尾巴；
 *   - 自动模式保留人物语义目标点，同时把可见尾巴限制为自然短尾巴；
 *   - 自动连接点按人物方向选择最近外边缘；
 *   - 目标落入文本框内部时，连接点贴近目标所在的最近边缘；
 *   - 框内目标的可见尾巴始终沿实际连接边向外生长；
 *   - manual模式完整保留教师拖动后的连接点和语义目标点；
 *   - origin_x/origin_y使用气泡本地0至1坐标；
 *   - target_x/target_y使用整格画布0至1坐标。
 *
 * 本文件只规范尾巴连接边和可见长度，不移动文本框。
 */

import type {
  CoursewareComicBubbleTail,
  CoursewareComicOverlayElement,
} from '@/api/coursewares'

export interface CoursewareComicEditableBubbleTail
  extends CoursewareComicBubbleTail {
  origin_x?: number
  origin_y?: number
}

export interface CoursewareComicTailLayoutPatch {
  tail_type?: 'auto' | 'manual'
  tail_origin_x?: number
  tail_origin_y?: number
  tail_target_x?: number
  tail_target_y?: number
}

export interface CoursewareComicTailPoint {
  x: number
  y: number
}

interface CoursewareComicTailDirection {
  x: number
  y: number
}

type CoursewareComicTailEdge =
  | 'left'
  | 'right'
  | 'top'
  | 'bottom'

const TAIL_CANVAS_WIDTH = 1920
const TAIL_CANVAS_HEIGHT = 1080
const ORIGIN_EDGE_INSET = 0.12
const AUTO_TAIL_MIN_PIXELS = 30
const AUTO_TAIL_MAX_PIXELS = 58
const INSIDE_TARGET_TAIL_PIXELS = 36
const INSIDE_EPSILON = 0.0001

const LEGACY_TARGET_POINTS:
  Record<string, CoursewareComicTailPoint> = {
    left_top: { x: 0.20, y: 0.32 },
    left_center: { x: 0.20, y: 0.50 },
    left_bottom: { x: 0.20, y: 0.68 },
    center_top: { x: 0.50, y: 0.32 },
    center: { x: 0.50, y: 0.50 },
    center_bottom: { x: 0.50, y: 0.68 },
    right_top: { x: 0.80, y: 0.32 },
    right_center: { x: 0.80, y: 0.50 },
    right_bottom: { x: 0.80, y: 0.68 },
  }

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(maximum, Math.max(minimum, value))
}

function finiteOr(value: unknown, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value)
    ? value
    : fallback
}

function editableTail(
  element: CoursewareComicOverlayElement,
): CoursewareComicEditableBubbleTail | null {
  return element.tail
    ? element.tail as CoursewareComicEditableBubbleTail
    : null
}

function defaultTailTarget(
  element: CoursewareComicOverlayElement,
): CoursewareComicTailPoint {
  const anchor = typeof element.target_anchor === 'string'
    ? element.target_anchor.trim()
    : ''
  const anchorPoint = LEGACY_TARGET_POINTS[anchor]

  if (anchorPoint) {
    return { ...anchorPoint }
  }

  return {
    x: clamp(element.x + element.width * 0.72, 0, 1),
    y: clamp(
      element.y + element.height + Math.max(0.04, element.height * 0.36),
      0,
      1,
    ),
  }
}

export function projectCoursewareComicTailOrigin(
  localX: number,
  localY: number,
): CoursewareComicTailPoint {
  const x = clamp(localX, 0, 1)
  const y = clamp(localY, 0, 1)
  const distances = [
    { edge: 'left', value: x },
    { edge: 'right', value: 1 - x },
    { edge: 'top', value: y },
    { edge: 'bottom', value: 1 - y },
  ] as const
  const closest = distances.reduce((current, candidate) =>
    candidate.value < current.value ? candidate : current,
  )

  switch (closest.edge) {
  case 'left':
    return { x: 0, y: clamp(y, ORIGIN_EDGE_INSET, 1 - ORIGIN_EDGE_INSET) }
  case 'right':
    return { x: 1, y: clamp(y, ORIGIN_EDGE_INSET, 1 - ORIGIN_EDGE_INSET) }
  case 'top':
    return { x: clamp(x, ORIGIN_EDGE_INSET, 1 - ORIGIN_EDGE_INSET), y: 0 }
  default:
    return { x: clamp(x, ORIGIN_EDGE_INSET, 1 - ORIGIN_EDGE_INSET), y: 1 }
  }
}

function normalizeDirection(
  direction: CoursewareComicTailDirection,
): CoursewareComicTailDirection {
  const length = Math.hypot(direction.x, direction.y)
  return length < 0.001
    ? { x: 0, y: 1 }
    : { x: direction.x / length, y: direction.y / length }
}

function fallbackDirectionFromAnchor(
  anchor: string | undefined,
): CoursewareComicTailDirection {
  switch ((anchor || '').trim()) {
  case 'left_top':
    return { x: -1, y: -1 }
  case 'left_center':
    return { x: -1, y: 0 }
  case 'left_bottom':
    return { x: -1, y: 1 }
  case 'center_top':
    return { x: 0, y: -1 }
  case 'center':
    return { x: 0, y: 1 }
  case 'right_top':
    return { x: 1, y: -1 }
  case 'right_center':
    return { x: 1, y: 0 }
  case 'right_bottom':
    return { x: 1, y: 1 }
  default:
    return { x: 0, y: 1 }
  }
}

function targetInsideElement(
  element: CoursewareComicOverlayElement,
  target: CoursewareComicTailPoint,
): boolean {
  return (
    target.x > element.x + INSIDE_EPSILON &&
    target.x < element.x + element.width - INSIDE_EPSILON &&
    target.y > element.y + INSIDE_EPSILON &&
    target.y < element.y + element.height - INSIDE_EPSILON
  )
}

interface CoursewareComicInsideTailPlacement {
  edge: CoursewareComicTailEdge
  origin: CoursewareComicTailPoint
  direction: CoursewareComicTailDirection
}

/**
 * 目标位于框内时，不从边缘中点生长。
 *
 * 先选择目标距离最近的边，再把连接点对齐到目标在该边上的投影，
 * 最后保留12%的圆角安全区。这样尾巴会从“靠近人物的地方”长出。
 */
function resolveInsideTargetPlacement(
  element: CoursewareComicOverlayElement,
  target: CoursewareComicTailPoint,
): CoursewareComicInsideTailPlacement {
  const localX = clamp(
    (target.x - element.x) /
      Math.max(0.001, element.width),
    0,
    1,
  )
  const localY = clamp(
    (target.y - element.y) /
      Math.max(0.001, element.height),
    0,
    1,
  )
  const candidates = [
    {
      edge: 'left' as const,
      distance:
        (target.x - element.x) *
        TAIL_CANVAS_WIDTH,
    },
    {
      edge: 'right' as const,
      distance:
        (element.x + element.width - target.x) *
        TAIL_CANVAS_WIDTH,
    },
    {
      edge: 'top' as const,
      distance:
        (target.y - element.y) *
        TAIL_CANVAS_HEIGHT,
    },
    {
      edge: 'bottom' as const,
      distance:
        (element.y + element.height - target.y) *
        TAIL_CANVAS_HEIGHT,
    },
  ]
  const selected = candidates.reduce(
    (current, candidate) =>
      candidate.distance < current.distance
        ? candidate
        : current,
  )

  switch (selected.edge) {
  case 'left':
    return {
      edge: selected.edge,
      origin: {
        x: 0,
        y: clamp(
          localY,
          ORIGIN_EDGE_INSET,
          1 - ORIGIN_EDGE_INSET,
        ),
      },
      direction: { x: -1, y: 0 },
    }

  case 'right':
    return {
      edge: selected.edge,
      origin: {
        x: 1,
        y: clamp(
          localY,
          ORIGIN_EDGE_INSET,
          1 - ORIGIN_EDGE_INSET,
        ),
      },
      direction: { x: 1, y: 0 },
    }

  case 'top':
    return {
      edge: selected.edge,
      origin: {
        x: clamp(
          localX,
          ORIGIN_EDGE_INSET,
          1 - ORIGIN_EDGE_INSET,
        ),
        y: 0,
      },
      direction: { x: 0, y: -1 },
    }

  default:
    return {
      edge: selected.edge,
      origin: {
        x: clamp(
          localX,
          ORIGIN_EDGE_INSET,
          1 - ORIGIN_EDGE_INSET,
        ),
        y: 1,
      },
      direction: { x: 0, y: 1 },
    }
  }
}

function outwardDirectionForOrigin(
  origin: CoursewareComicTailPoint,
): CoursewareComicTailDirection {
  if (origin.y === 0) {
    return { x: 0, y: -1 }
  }
  if (origin.x === 1) {
    return { x: 1, y: 0 }
  }
  if (origin.y === 1) {
    return { x: 0, y: 1 }
  }
  return { x: -1, y: 0 }
}

function directionFromElementToTarget(
  element: CoursewareComicOverlayElement,
  target: CoursewareComicTailPoint,
): CoursewareComicTailDirection {
  if (targetInsideElement(element, target)) {
    return resolveInsideTargetPlacement(
      element,
      target,
    ).direction
  }

  const direction = {
    x: (target.x - (element.x + element.width / 2)) * TAIL_CANVAS_WIDTH,
    y: (target.y - (element.y + element.height / 2)) * TAIL_CANVAS_HEIGHT,
  }

  if (Math.hypot(direction.x, direction.y) < 1) {
    return normalizeDirection(
      fallbackDirectionFromAnchor(element.target_anchor),
    )
  }

  return normalizeDirection(direction)
}

function originForDirection(
  element: CoursewareComicOverlayElement,
  sourceDirection: CoursewareComicTailDirection,
): CoursewareComicTailPoint {
  const direction = normalizeDirection(sourceDirection)
  const maxX = Math.abs(direction.x) > 0.0001
    ? element.width * TAIL_CANVAS_WIDTH / 2 / Math.abs(direction.x)
    : Number.POSITIVE_INFINITY
  const maxY = Math.abs(direction.y) > 0.0001
    ? element.height * TAIL_CANVAS_HEIGHT / 2 / Math.abs(direction.y)
    : Number.POSITIVE_INFINITY
  const distance = Math.min(maxX, maxY)

  const globalX = element.x + element.width / 2 +
    direction.x * distance / TAIL_CANVAS_WIDTH
  const globalY = element.y + element.height / 2 +
    direction.y * distance / TAIL_CANVAS_HEIGHT

  return projectCoursewareComicTailOrigin(
    (globalX - element.x) / Math.max(0.001, element.width),
    (globalY - element.y) / Math.max(0.001, element.height),
  )
}

function originGlobal(
  element: CoursewareComicOverlayElement,
  origin: CoursewareComicTailPoint,
): CoursewareComicTailPoint {
  return {
    x: element.x + element.width * origin.x,
    y: element.y + element.height * origin.y,
  }
}

function pointFromOrigin(
  origin: CoursewareComicTailPoint,
  direction: CoursewareComicTailDirection,
  lengthPixels: number,
): CoursewareComicTailPoint {
  const normalized = normalizeDirection(direction)

  return {
    x: clamp(
      origin.x + normalized.x * lengthPixels / TAIL_CANVAS_WIDTH,
      0,
      1,
    ),
    y: clamp(
      origin.y + normalized.y * lengthPixels / TAIL_CANVAS_HEIGHT,
      0,
      1,
    ),
  }
}

function normalizedStoredTarget(
  element: CoursewareComicOverlayElement,
  tail: CoursewareComicEditableBubbleTail | null,
): CoursewareComicTailPoint {
  const fallback = defaultTailTarget(element)
  const trusted = tail?.type === 'manual' || tail?.type === 'auto'

  if (!trusted) {
    return fallback
  }

  return {
    x: clamp(finiteOr(tail?.target_x, fallback.x), 0, 1),
    y: clamp(finiteOr(tail?.target_y, fallback.y), 0, 1),
  }
}

/**
 * 计算只用于显示的尾巴尖端：
 *   - 框内目标从最近边缘向外生长；
 *   - 自动尾巴限制在30至58设计像素；
 *   - manual框外目标保持教师设置的实际距离。
 *
 * 稳定文档中的target_x/target_y仍保留人物语义目标，不被这里改写。
 */
function resolveVisibleTailTarget(
  element: CoursewareComicOverlayElement,
  tail: CoursewareComicEditableBubbleTail,
  origin: CoursewareComicTailPoint,
  storedTarget: CoursewareComicTailPoint,
): CoursewareComicTailPoint {
  const globalOrigin = originGlobal(element, origin)
  const inside = targetInsideElement(element, storedTarget)

  if (inside) {
    return pointFromOrigin(
      globalOrigin,
      outwardDirectionForOrigin(origin),
      INSIDE_TARGET_TAIL_PIXELS,
    )
  }

  if (tail.type === 'manual') {
    return storedTarget
  }

  const delta = {
    x: (storedTarget.x - globalOrigin.x) * TAIL_CANVAS_WIDTH,
    y: (storedTarget.y - globalOrigin.y) * TAIL_CANVAS_HEIGHT,
  }
  const distance = Math.hypot(delta.x, delta.y)
  const direction = distance < 1
    ? directionFromElementToTarget(element, storedTarget)
    : normalizeDirection(delta)
  const visibleLength = clamp(
    distance,
    AUTO_TAIL_MIN_PIXELS,
    AUTO_TAIL_MAX_PIXELS,
  )

  return pointFromOrigin(
    globalOrigin,
    direction,
    visibleLength,
  )
}

function resolveAutomaticTail(
  element: CoursewareComicOverlayElement,
  target: CoursewareComicTailPoint,
): CoursewareComicEditableBubbleTail {
  const origin = targetInsideElement(element, target)
    ? resolveInsideTargetPlacement(
        element,
        target,
      ).origin
    : originForDirection(
        element,
        directionFromElementToTarget(
          element,
          target,
        ),
      )

  return {
    type: 'auto',
    target_x: target.x,
    target_y: target.y,
    origin_x: origin.x,
    origin_y: origin.y,
  }
}

export function normalizeCoursewareComicSpeechTail(
  element: CoursewareComicOverlayElement,
): CoursewareComicOverlayElement {
  if (element.type !== 'speech_bubble') {
    return element
  }

  const current = editableTail(element)
  const target = normalizedStoredTarget(element, current)

  if (current?.type === 'manual') {
    const automaticOrigin = originForDirection(
      element,
      directionFromElementToTarget(element, target),
    )
    const origin = projectCoursewareComicTailOrigin(
      finiteOr(current.origin_x, automaticOrigin.x),
      finiteOr(current.origin_y, automaticOrigin.y),
    )
    const tail: CoursewareComicEditableBubbleTail = {
      type: 'manual',
      target_x: target.x,
      target_y: target.y,
      origin_x: origin.x,
      origin_y: origin.y,
    }

    return { ...element, tail: tail as CoursewareComicBubbleTail }
  }

  return {
    ...element,
    tail: resolveAutomaticTail(
      element,
      target,
    ) as CoursewareComicBubbleTail,
  }
}

export function hasCoursewareComicTailPatch(
  patch: CoursewareComicTailLayoutPatch,
): boolean {
  return (
    patch.tail_type !== undefined ||
    patch.tail_origin_x !== undefined ||
    patch.tail_origin_y !== undefined ||
    patch.tail_target_x !== undefined ||
    patch.tail_target_y !== undefined
  )
}

export function applyCoursewareComicTailPatch(
  source: CoursewareComicOverlayElement,
  patch: CoursewareComicTailLayoutPatch,
): CoursewareComicOverlayElement {
  const normalized = normalizeCoursewareComicSpeechTail(source)

  if (
    normalized.type !== 'speech_bubble' ||
    !hasCoursewareComicTailPatch(patch)
  ) {
    return normalized
  }

  const current = editableTail(normalized)
  const target = {
    x: clamp(patch.tail_target_x ?? current?.target_x ?? 0.5, 0, 1),
    y: clamp(patch.tail_target_y ?? current?.target_y ?? 0.5, 0, 1),
  }
  const type = patch.tail_type ??
    (current?.type === 'manual' ? 'manual' : 'auto')

  if (type === 'auto') {
    return {
      ...normalized,
      tail: resolveAutomaticTail(
        normalized,
        target,
      ) as CoursewareComicBubbleTail,
      layout_dirty: true,
    }
  }

  const origin = projectCoursewareComicTailOrigin(
    patch.tail_origin_x ?? current?.origin_x ?? 0.72,
    patch.tail_origin_y ?? current?.origin_y ?? 1,
  )
  const tail: CoursewareComicEditableBubbleTail = {
    type: 'manual',
    target_x: target.x,
    target_y: target.y,
    origin_x: origin.x,
    origin_y: origin.y,
  }

  return {
    ...normalized,
    tail: tail as CoursewareComicBubbleTail,
    layout_dirty: true,
  }
}

export function resolveCoursewareComicTailOrigin(
  element: CoursewareComicOverlayElement,
): CoursewareComicTailPoint | null {
  if (element.type !== 'speech_bubble') {
    return null
  }

  const tail = editableTail(normalizeCoursewareComicSpeechTail(element))
  return tail
    ? {
        x: clamp(finiteOr(tail.origin_x, 0.72), 0, 1),
        y: clamp(finiteOr(tail.origin_y, 1), 0, 1),
      }
    : null
}

/**
 * 返回稳定文档中的人物语义目标。
 *
 * 编辑器橙色控制点继续显示真实人物目标，而不是被截短后的可见尖端。
 */
export function resolveCoursewareComicTailTarget(
  element: CoursewareComicOverlayElement,
): CoursewareComicTailPoint | null {
  if (element.type !== 'speech_bubble') {
    return null
  }

  const normalized = normalizeCoursewareComicSpeechTail(element)
  const tail = editableTail(normalized)

  return tail
    ? normalizedStoredTarget(normalized, tail)
    : null
}

/**
 * 返回实际用于绘制统一气泡路径的可见尾巴尖端。
 */
export function resolveCoursewareComicVisibleTailTarget(
  element: CoursewareComicOverlayElement,
): CoursewareComicTailPoint | null {
  if (element.type !== 'speech_bubble') {
    return null
  }

  const normalized = normalizeCoursewareComicSpeechTail(element)
  const tail = editableTail(normalized)

  if (!tail) {
    return null
  }

  const origin = {
    x: clamp(finiteOr(tail.origin_x, 0.72), 0, 1),
    y: clamp(finiteOr(tail.origin_y, 1), 0, 1),
  }
  const storedTarget = normalizedStoredTarget(normalized, tail)

  return resolveVisibleTailTarget(
    normalized,
    tail,
    origin,
    storedTarget,
  )
}

export function resolveCoursewareComicTailPoints(
  element: CoursewareComicOverlayElement,
): string | null {
  const origin = resolveCoursewareComicTailOrigin(element)
  const target =
    resolveCoursewareComicVisibleTailTarget(element)

  if (!origin || !target) {
    return null
  }

  const localTargetX =
    (target.x - element.x) / Math.max(0.001, element.width) * 100
  const localTargetY =
    (target.y - element.y) / Math.max(0.001, element.height) * 100
  const originX = origin.x * 100
  const originY = origin.y * 100
  const horizontalEdge = origin.y === 0 || origin.y === 1
  const halfBase = 3
  const first = horizontalEdge
    ? `${originX - halfBase},${originY}`
    : `${originX},${originY - halfBase}`
  const second = horizontalEdge
    ? `${originX + halfBase},${originY}`
    : `${originX},${originY + halfBase}`

  return [
    first,
    second,
    `${localTargetX.toFixed(2)},${localTargetY.toFixed(2)}`,
  ].join(' ')
}

export function resolveCoursewareComicTailMode(
  element: CoursewareComicOverlayElement,
): 'auto' | 'manual' {
  const tail = editableTail(normalizeCoursewareComicSpeechTail(element))
  return tail?.type === 'manual' ? 'manual' : 'auto'
}
