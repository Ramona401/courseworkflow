/**
 * 课件工坊快捷预设画风专用锚点API。
 *
 * 快捷预设只提交preset_style_key：
 *   - 不再先生成课程专属样板图；
 *   - 不提交asset_id；
 *   - 不调用图片模型；
 *   - 不调用多模态模型；
 *   - 后端直接使用系统高清预设图和可信白名单创建课程锚点。
 */

import apiClient from './client'

import {
  extractData,
} from './coursewares.types'

import type {
  SetStyleAnchorResult,
} from './coursewares.types'

export const COURSEWARE_PRESET_STYLE_KEYS = [
  'pixar',
  'flat',
  'ghibli',
  'realistic',
  'chinese',
  'ink_wash',
  'guochao',
  'storybook',
  'science',
  'tech',
] as const

export type CoursewarePresetStyleKey =
  typeof COURSEWARE_PRESET_STYLE_KEYS[number]

function isCoursewarePresetStyleKey(
  value: string,
): value is CoursewarePresetStyleKey {
  return (
    COURSEWARE_PRESET_STYLE_KEYS as readonly string[]
  ).includes(value)
}

/**
 * 直接把系统预设画风设置为课程锚点。
 */
export async function setPresetStyleAnchor(
  coursewareId: string,
  presetStyleKey: string,
): Promise<SetStyleAnchorResult> {
  const normalizedCoursewareId =
    coursewareId.trim()

  const normalizedStyleKey =
    presetStyleKey
      .trim()
      .toLowerCase()

  if (!normalizedCoursewareId) {
    throw new Error(
      '课件ID不能为空',
    )
  }

  if (
    !isCoursewarePresetStyleKey(
      normalizedStyleKey,
    )
  ) {
    throw new Error(
      '不支持的快捷预设画风',
    )
  }

  const response =
    await apiClient.post(
      `/coursewares/${normalizedCoursewareId}/style-anchor`,
      {
        preset_style_key:
          normalizedStyleKey,
      },
      {
        // 仅数据库读取和写入，不调用外部AI。
        timeout: 30000,
      },
    )

  return extractData(response)
}
