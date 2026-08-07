/**
 * coursewareComicStyleOptions.ts
 *
 * 知识点漫画覆盖层稳定样式选项：
 *   - 说话气泡包含经典、柔和、云朵、描边、胶囊和活力六种样式；
 *   - 思考气泡、问题卡、教学卡使用各自的样式集合；
 *   - 历史未知style_id继续显示，教师主动选择后才替换；
 *   - 所有选择仍保存到既有style_id，不新增数据库字段。
 */

import type {
  CoursewareComicOverlayElement,
} from '@/api/coursewares'

export interface CoursewareComicStyleOption {
  value: string
  label: string
  description: string
}

const SPEECH_OPTIONS:
  readonly CoursewareComicStyleOption[] = [
    {
      value: 'speech_round',
      label: '经典圆角',
      description: '清楚稳定的经典漫画对白框',
    },
    {
      value: 'speech_soft',
      label: '柔和轻盈',
      description: '低对比边框与轻盈阴影',
    },
    {
      value: 'speech_cloud',
      label: '云朵对白',
      description: '圆润活泼的云朵漫画对白框',
    },
    {
      value: 'speech_outline',
      label: '漫画描边',
      description: '更有手绘感的强调轮廓',
    },
    {
      value: 'speech_capsule',
      label: '现代胶囊',
      description: '圆润简洁的现代对白框',
    },
    {
      value: 'speech_pop',
      label: '活力对白',
      description: '适合强调语气与活跃场景',
    },
  ]

const THOUGHT_OPTIONS:
  readonly CoursewareComicStyleOption[] = [
    {
      value: 'thought_cloud',
      label: '云朵思考',
      description: '经典漫画云朵思考框',
    },
    {
      value: 'thought_soft',
      label: '柔和思考',
      description: '轻盈弱边框的思考框',
    },
    {
      value: 'thought_outline',
      label: '漫画思考',
      description: '强调轮廓的思考气泡',
    },
  ]

const QUESTION_OPTIONS:
  readonly CoursewareComicStyleOption[] = [
    {
      value: 'question_purple',
      label: '紫色问题卡',
      description: '默认教学问题卡',
    },
    {
      value: 'question_blue',
      label: '蓝色问题卡',
      description: '理性清晰的蓝色问题卡',
    },
    {
      value: 'question_orange',
      label: '橙色问题卡',
      description: '适合提醒与挑战的问题卡',
    },
  ]

const CARD_OPTIONS:
  readonly CoursewareComicStyleOption[] = [
    {
      value: 'card_dark',
      label: '深色卡片',
      description: '深色半透明教学卡',
    },
    {
      value: 'card_light',
      label: '浅色卡片',
      description: '简洁明亮的教学卡',
    },
    {
      value: 'card_accent',
      label: '强调卡片',
      description: '用于重点结论和提醒',
    },
  ]

function baseOptions(
  element: CoursewareComicOverlayElement,
): readonly CoursewareComicStyleOption[] {
  switch (element.type) {
  case 'speech_bubble':
    return SPEECH_OPTIONS
  case 'thought_bubble':
    return THOUGHT_OPTIONS
  case 'question_card':
    return QUESTION_OPTIONS
  default:
    return CARD_OPTIONS
  }
}

export function coursewareComicStyleOptions(
  element: CoursewareComicOverlayElement,
): readonly CoursewareComicStyleOption[] {
  const options = baseOptions(element)
  const current = element.style_id.trim()

  if (
    !current ||
    options.some(option => option.value === current)
  ) {
    return options
  }

  return [
    {
      value: current,
      label: '当前历史样式',
      description: '兼容既有项目中的样式标识',
    },
    ...options,
  ]
}

export function resolveCoursewareComicStyleID(
  element: CoursewareComicOverlayElement,
  requestedStyleID = '',
): string {
  const options = baseOptions(element)
  const requested = requestedStyleID.trim()

  if (
    requested &&
    options.some(option => option.value === requested)
  ) {
    return requested
  }

  const currentIndex = options.findIndex(
    option => option.value === element.style_id,
  )

  return options[
    (currentIndex + 1) % options.length
  ].value
}

export function coursewareComicStyleControlLabel(
  element: CoursewareComicOverlayElement,
): string {
  switch (element.type) {
  case 'speech_bubble':
    return '对白框'
  case 'thought_bubble':
    return '思考框'
  case 'question_card':
    return '问题卡'
  default:
    return '卡片'
  }
}
