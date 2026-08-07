/**
 * coursewareReviewLessonPlanContext.ts
 *
 * 课件问题与来源教案按需对照的公共入口。
 *
 * 本文件只保留跨组件事件协议与组合Hook，具体职责拆分为：
 *   - coursewareReviewLessonPlanStructure.ts：审核对照专用只读章节结构；
 *   - coursewareReviewLessonPlanMatch.ts：章节匹配纯函数；
 *   - useCoursewareLessonPlanContent.ts：来源教案请求生命周期；
 *   - useCoursewareLessonPlanDrawerInteraction.ts：抽屉尺寸、拖拽、焦点和键盘管理。
 *
 * 对外导出名称保持不变，现有正式审核、自审和课件工坊调用方无需改动。
 */

import { useMemo } from "react";

import {
  findBestLessonPlanSection,
  type CoursewareLessonPlanMatchRequest,
} from "./coursewareReviewLessonPlanMatch";
import {
  parseCoursewareReviewLessonDocumentStructure,
} from "./coursewareReviewLessonPlanStructure";
import {
  useCoursewareLessonPlanContent,
} from "./useCoursewareLessonPlanContent";
import {
  useCoursewareLessonPlanDrawerInteraction,
} from "./useCoursewareLessonPlanDrawerInteraction";

export {
  findBestLessonPlanSection,
} from "./coursewareReviewLessonPlanMatch";

export const COURSEWARE_REVIEW_LESSON_PLAN_CONTEXT_EVENT =
  "tedna:courseware-review-open-lesson-plan";

export interface CoursewareReviewLessonPlanContextRequest
  extends CoursewareLessonPlanMatchRequest {
  issueId: string;
  pageNumber: number;

  /** 仅用于抽屉关闭后恢复键盘焦点，不参与接口、日志或权限判断。 */
  triggerElement?: HTMLElement | null;
}

export interface CoursewareLessonPlanDrawerStateOptions {
  open: boolean;
  coursewareId: string;
  contextRequest?: CoursewareReviewLessonPlanContextRequest | null;
  openerElement?: HTMLElement | null;
  storageKey: string;
  onClose: () => void;
  onHasLessonPlanChange?: (
    hasLessonPlan: boolean,
  ) => void;
}

/**
 * 从任意课件评审问题入口发出“打开来源教案对照”事件。
 * 事件只携带教师当前查看问题的展示上下文，不携带权限或页面正文事实。
 */
export function openCoursewareReviewLessonPlanContext(
  request: CoursewareReviewLessonPlanContextRequest,
): void {
  if (typeof window === "undefined") return;

  window.dispatchEvent(
    new CustomEvent<CoursewareReviewLessonPlanContextRequest>(
      COURSEWARE_REVIEW_LESSON_PLAN_CONTEXT_EVENT,
      {
        detail: request,
      },
    ),
  );
}

/** 订阅来源教案对照事件，并返回标准清理函数。 */
export function subscribeCoursewareReviewLessonPlanContext(
  listener: (
    request: CoursewareReviewLessonPlanContextRequest,
  ) => void,
): () => void {
  if (typeof window === "undefined") {
    return () => undefined;
  }

  const handleEvent = (
    event: Event,
  ) => {
    const detail = (
      event as CustomEvent<CoursewareReviewLessonPlanContextRequest>
    ).detail;

    if (
      !detail ||
      typeof detail.issueId !== "string"
    ) {
      return;
    }

    listener(detail);
  };

  window.addEventListener(
    COURSEWARE_REVIEW_LESSON_PLAN_CONTEXT_EVENT,
    handleEvent,
  );

  return () => {
    window.removeEventListener(
      COURSEWARE_REVIEW_LESSON_PLAN_CONTEXT_EVENT,
      handleEvent,
    );
  };
}

/**
 * 组合来源教案内容、章节匹配和抽屉交互状态。
 *
 * 请求与交互职责分别由独立Hook管理，避免一个大型Hook同时维护网络、
 * 章节算法、拖拽、焦点和键盘副作用。
 */
export function useCoursewareLessonPlanDrawerState({
  open,
  coursewareId,
  contextRequest,
  openerElement,
  storageKey,
  onClose,
  onHasLessonPlanChange,
}: CoursewareLessonPlanDrawerStateOptions) {
  const contentState =
    useCoursewareLessonPlanContent({
      open,
      coursewareId,
      onHasLessonPlanChange,
    });

  const documentStructure = useMemo(
    () =>
      parseCoursewareReviewLessonDocumentStructure(
        contentState.data?.content || "",
      ),
    [contentState.data?.content],
  );

  const matchedSection = useMemo(
    () =>
      findBestLessonPlanSection(
        documentStructure.sections,
        contextRequest,
      ),
    [
      contextRequest,
      documentStructure.sections,
    ],
  );

  const focusContextKey = [
    coursewareId,
    contextRequest?.issueId || "",
    matchedSection?.id || "",
  ].join("\u001f");

  const interactionState =
    useCoursewareLessonPlanDrawerInteraction({
      open,
      storageKey,
      openerElement,
      onClose,
      hasContent: Boolean(
        contentState.data?.content.trim(),
      ),
      matchedSection,
      focusContextKey,
    });

  return {
    ...contentState,
    ...interactionState,
    documentStructure,
    matchedSection,
  };
}
