/**
 * coursewareComicPanelEditorValidation.ts
 *
 * 知识点漫画覆盖层保存校验：
 *   - 至少保留一个文字或题目元素；
 *   - 元素位置和尺寸必须位于画布范围内；
 *   - 字号、行距、颜色模式、背景透明度和整体描边必须在安全范围内；
 *   - 问题卡必须包含题目、至少两个选项和有效答案；
 *   - 普通文字元素内容不能为空。
 *
 * 本文件只进行纯数据校验，不修改草稿，不调用接口。
 */

import type {
  CoursewareComicOverlayDocument,
} from '@/api/coursewares'

import {
  coursewareComicElementLabel,
} from './coursewareComicEditorDraft'

const HEX_COLOR_PATTERN = /^#[0-9a-fA-F]{6}$/

function finiteInRange(
  value: number,
  minimum: number,
  maximum: number,
): boolean {
  return (
    Number.isFinite(value) &&
    value >= minimum &&
    value <= maximum
  )
}

export function validateCoursewareComicOverlayDocument(
  document: CoursewareComicOverlayDocument,
): string {
  if (!document.elements.length) {
    return '当前格至少需要一个文字或题目元素。'
  }

  for (const element of document.elements) {
    const label = coursewareComicElementLabel(element)

    if (
      !finiteInRange(element.width, 0.001, 1) ||
      !finiteInRange(element.height, 0.001, 1) ||
      !finiteInRange(element.x, 0, 1) ||
      !finiteInRange(element.y, 0, 1) ||
      element.x + element.width > 1.001 ||
      element.y + element.height > 1.001
    ) {
      return `${label}超出画布边界。`
    }

    if (
      !finiteInRange(
        element.text_style.font_size,
        12,
        120,
      )
    ) {
      return `${label}字号必须在12至120之间。`
    }


    if (
      !Number.isInteger(element.text_style.font_weight) ||
      !finiteInRange(
        element.text_style.font_weight,
        300,
        900,
      )
    ) {
      return `${label}字重必须是300至900之间的整数。`
    }

    if (
      !['left', 'center', 'right', 'justify'].includes(
        element.text_style.align || '',
      )
    ) {
      return `${label}文字对齐方式无效。`
    }

    if (!Number.isFinite(element.rotation)) {
      return `${label}旋转角度无效。`
    }

    if (
      !finiteInRange(
        element.text_style.line_height || 1.35,
        1,
        2.2,
      )
    ) {
      return `${label}行距必须在1至2.2之间。`
    }

    const colorMode =
      element.text_style.color_mode || 'auto'

    if (
      colorMode !== 'auto' &&
      colorMode !== 'manual'
    ) {
      return `${label}文字颜色模式无效。`
    }

    if (
      colorMode === 'manual' &&
      !HEX_COLOR_PATTERN.test(
        element.text_style.color || '',
      )
    ) {
      return `${label}手动文字颜色格式无效。`
    }

    const backgroundOpacity =
      element.text_style.background_opacity

    if (
      backgroundOpacity !== undefined &&
      backgroundOpacity !== 0 &&
      !finiteInRange(
        backgroundOpacity,
        0.2,
        1,
      )
    ) {
      return `${label}背景透明度必须在20%至100%之间。`
    }

    const outlineWidth =
      element.text_style.outline_width

    if (
      outlineWidth !== undefined &&
      outlineWidth !== 0 &&
      !finiteInRange(
        outlineWidth,
        0.5,
        3,
      )
    ) {
      return `${label}描边宽度必须在0.5至3像素之间。`
    }

    if (element.type === 'question_card') {
      const question = element.question

      if (
        !question ||
        !question.question.trim()
      ) {
        return '问题卡的题目不能为空。'
      }

      const options =
        question.options
          .map(item => item.trim())
          .filter(Boolean)

      if (options.length < 2) {
        return '问题卡至少需要两个有效选项。'
      }

      if (
        question.answer_index < 0 ||
        question.answer_index >= options.length
      ) {
        return '问题卡的正确答案超出选项范围。'
      }

      continue
    }

    if (!element.content.trim()) {
      return `${label}不能为空。`
    }
  }

  return ''
}

export function coursewareComicEditorErrorMessage(
  error: unknown,
  fallback: string,
): string {
  return error instanceof Error
    ? error.message
    : fallback
}
