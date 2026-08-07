/**
 * coursewareComicEditorText.ts
 *
 * 漫画覆盖层文字读取与中文标签模块。
 *
 * 本文件只负责安全的纯文字转换：
 *   - 普通气泡和文字卡读取content；
 *   - 问题卡组合题目与选项供画布显示；
 *   - 把覆盖层元素类型转换为教师可理解的中文名称。
 *
 * 不负责布局、网络请求、持久化或内部图片索引。
 */

import type {
  CoursewareComicOverlayElement,
} from '@/api/coursewares'

/**
 * overlayElementDisplayText
 *
 * 返回覆盖层元素在画布中实际显示的文字。
 * 问题卡只展示题目和选项，正确答案与解析保留在编辑浮层中。
 */
export function overlayElementDisplayText(
  element:
    CoursewareComicOverlayElement,
): string {
  if (
    element.type ===
      'question_card' &&
    element.question
  ) {
    const options =
      element.question.options
        .filter(Boolean)
        .map(
          (
            option,
            index,
          ) =>
            `${String.fromCharCode(
              65 + index,
            )}. ${option}`,
        )

    return [
      element.question.question,
      ...options,
    ]
      .filter(Boolean)
      .join('\n')
  }

  return element.content
}

/**
 * coursewareComicElementLabel
 *
 * 返回覆盖层元素的教师界面中文名称。
 */
export function coursewareComicElementLabel(
  element:
    CoursewareComicOverlayElement,
): string {
  const labels:
    Record<string, string> = {
      speech_bubble:
        '对白气泡',
      thought_bubble:
        '思考气泡',
      narration:
        '旁白',
      knowledge_card:
        '知识卡',
      warning_card:
        '易错提醒',
      question_card:
        '问题卡',
      answer_card:
        '答案卡',
      caption:
        '说明文字',
      emphasis:
        '重点强调',
    }

  return (
    labels[element.type] ||
    element.type
  )
}
