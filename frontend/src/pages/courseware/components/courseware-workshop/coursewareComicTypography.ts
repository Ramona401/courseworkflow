/**
 * coursewareComicTypography.ts
 *
 * 知识点漫画编辑器文字尺寸统一模块：
 *   - text_style.font_size是覆盖层文档中的稳定字号；
 *   - 浏览器编辑画布为了适配缩略画布，会按固定比例显示该字号；
 *   - 自动换行和高度保护必须使用与浏览器相同的显示字号；
 *   - 说话与思考气泡的稳定字号下限为22，其他卡片保持18。
 *
 * 本文件只进行纯数值计算，不读取DOM、不修改草稿、不调用网络接口。
 */

import type {
  CoursewareComicOverlayElement,
} from '@/api/coursewares'

/**
 * 当前编辑画布沿用的稳定字号缩放比例。
 *
 * 该值原先只存在于视觉样式中，布局模块仍按未缩放字号测量，
 * 因而会把实际单行文字误判成多行并强制撑高。
 */
export const COURSEWARE_COMIC_EDITOR_FONT_SCALE =
  3.45

function isCoursewareComicBubble(
  element:
    Pick<
      CoursewareComicOverlayElement,
      'type'
    >,
): boolean {
  return (
    element.type ===
      'speech_bubble' ||
    element.type ===
      'thought_bubble'
  )
}

/**
 * minimumCoursewareComicStoredFontSize
 *
 * 返回文档中允许保存的最小字号。
 * 气泡按教师要求固定为22；卡片继续允许18。
 */
export function minimumCoursewareComicStoredFontSize(
  element:
    Pick<
      CoursewareComicOverlayElement,
      'type'
    >,
): number {
  return isCoursewareComicBubble(
    element,
  )
    ? 22
    : 18
}

/**
 * normalizeCoursewareComicStoredFontSize
 *
 * 对保存字号执行统一边界保护。
 */
export function normalizeCoursewareComicStoredFontSize(
  element:
    Pick<
      CoursewareComicOverlayElement,
      'type'
    >,
  value: number,
): number {
  const minimum =
    minimumCoursewareComicStoredFontSize(
      element,
    )

  return Math.min(
    96,
    Math.max(
      minimum,
      Number.isFinite(value)
        ? value
        : minimum,
    ),
  )
}

/**
 * resolveCoursewareComicEditorFontSize
 *
 * 返回浏览器编辑画布实际使用的CSS像素字号。
 * 本函数同时供视觉渲染和布局测量使用，避免两套口径再次漂移。
 */
export function resolveCoursewareComicEditorFontSize(
  element:
    Pick<
      CoursewareComicOverlayElement,
      'type'
    >,
  storedFontSize: number,
): number {
  const visualMinimum =
    isCoursewareComicBubble(
      element,
    )
      ? 9.5
      : 9

  return Math.max(
    visualMinimum,
    normalizeCoursewareComicStoredFontSize(
      element,
      storedFontSize,
    ) /
      COURSEWARE_COMIC_EDITOR_FONT_SCALE,
  )
}
