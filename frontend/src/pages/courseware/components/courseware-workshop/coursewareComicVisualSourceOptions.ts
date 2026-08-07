/**
 * coursewareComicVisualSourceOptions.ts
 *
 * 知识点漫画第三步“画风来源”稳定选项：
 *   - courseware：唯一画风来源是课件整体风格锚点；
 *   - selected：唯一画风来源是老师选择的漫画预设画风；
 *   - 不提供混合或自动回退模式；
 *   - 这里只保存教师可见文案，不执行网络请求。
 */

import type {
  CoursewareComicStyleSettingsDraft,
} from './coursewareComicWorkflow'

export type CoursewareComicVisualStyleSource =
  CoursewareComicStyleSettingsDraft[
    'visualStyleSource'
  ]

export interface CoursewareComicVisualSourceOption {
  value: CoursewareComicVisualStyleSource
  icon: string
  label: string
  description: string
  detail: string
}

export const COURSEWARE_COMIC_VISUAL_SOURCE_OPTIONS:
  readonly CoursewareComicVisualSourceOption[] = [
    {
      value: 'courseware',
      icon: '🧩',
      label: '跟随课件整体风格',
      description:
        '让漫画与当前课件的页面和插图保持统一。',
      detail:
        '只使用课件风格锚点，不叠加六种漫画预设。课件没有有效风格锚点时，生成会明确停止并提示先设置课件画风。',
    },
    {
      value: 'selected',
      icon: '🎨',
      label: '使用本漫画所选画风',
      description:
        '严格按照下方选择的漫画美术风格生成。',
      detail:
        '完全忽略课件风格锚点。人物设定图、首格样张和后续分格只使用同一个已选漫画画风。',
    },
  ]

export function coursewareComicVisualSourceLabel(
  value: CoursewareComicVisualStyleSource,
): string {
  return (
    COURSEWARE_COMIC_VISUAL_SOURCE_OPTIONS.find(
      option => option.value === value,
    )?.label || value
  )
}
