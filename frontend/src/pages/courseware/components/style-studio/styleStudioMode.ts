/**
 * AI美术风格工作室模式一致性纯函数。
 *
 * 前端存在两个模式：
 *   - currentMode：老师界面上刚选择的模式；
 *   - sessionMode：后端会话已经持久化的模式。
 *
 * 当二者不同：
 *   - 旧预览可能来自另一种模式；
 *   - 原始参考图的可确认权限可能发生变化；
 *   - 不能继续选择旧预览或确认锚点；
 *   - 老师必须发送新要求、重新上传参考图，
 *     或按当前模式重新生成三类预览。
 */

import type {
  CoursewareStyleReferenceMode,
} from '@/api/coursewareStyleStudio'

/** 判断界面模式是否尚未同步到服务端会话。 */
export function isCoursewareStyleModeDirty(
  sessionMode:
    | CoursewareStyleReferenceMode
    | null
    | undefined,
  currentMode: CoursewareStyleReferenceMode,
): boolean {
  if (!sessionMode) {
    return false
  }

  return sessionMode !== currentMode
}

/** 模式发生改变时应清除当前图片选择。 */
export function didCoursewareStyleModeChange(
  previousMode: CoursewareStyleReferenceMode,
  nextMode: CoursewareStyleReferenceMode,
): boolean {
  return previousMode !== nextMode
}
