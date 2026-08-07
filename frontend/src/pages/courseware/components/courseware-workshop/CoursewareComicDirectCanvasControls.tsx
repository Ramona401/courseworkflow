/**
 * CoursewareComicDirectCanvasControls.tsx
 *
 * 漫画直接画布控制兼容入口。
 *
 * 元素渲染、问题卡编辑和指针数学已拆分到独立文件。
 * 现有CoursewareComicDirectCanvas继续从本文件导入，
 * 无需改变其公开调用接口。
 */

export {
  CoursewareComicCanvasElement,
} from './CoursewareComicDirectCanvasElement'

export {
  CoursewareComicQuestionPopover,
} from './CoursewareComicDirectCanvasQuestionPopover'

export {
  clampCanvasValue,
  resolveResizePatch,
} from './CoursewareComicDirectCanvasSupport'

export type {
  LayoutPatch,
  PointerInteraction,
  ResizeCorner,
} from './CoursewareComicDirectCanvasSupport'
