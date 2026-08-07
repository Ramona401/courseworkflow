/**
 * CWLessonPlanCompareDrawer.tsx
 *
 * 正式课件审核工作台的来源教案对照入口。
 *
 * 既兼容工作台原有的受控打开状态，也监听单问题工作区的共享事件，
 * 从当前问题直接打开教案并定位相关章节。
 *
 * 外层使用coursewareId作为key，保证切换审核课件时不会继承上一课件的
 * 事件打开状态、问题上下文或焦点恢复元素。
 */

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import CoursewareLessonPlanContextDrawer from "./CoursewareLessonPlanContextDrawer";
import {
  subscribeCoursewareReviewLessonPlanContext,
  type CoursewareReviewLessonPlanContextRequest,
} from "./coursewareReviewLessonPlanContext";

export interface CWLessonPlanCompareDrawerProps {
  open: boolean;
  coursewareId: string;
  lessonPlanId?: string | null;
  onClose: () => void;
}

export default function CWLessonPlanCompareDrawer(
  props: CWLessonPlanCompareDrawerProps,
) {
  return (
    <CWLessonPlanCompareDrawerState
      key={props.coursewareId}
      {...props}
    />
  );
}

function CWLessonPlanCompareDrawerState({
  open,
  coursewareId,
  lessonPlanId,
  onClose,
}: CWLessonPlanCompareDrawerProps) {
  const [requestedOpen, setRequestedOpen] = useState(false);
  const [
    contextRequest,
    setContextRequest,
  ] = useState<CoursewareReviewLessonPlanContextRequest | null>(null);
  const [
    openerElement,
    setOpenerElement,
  ] = useState<HTMLElement | null>(null);

  useEffect(
    () =>
      subscribeCoursewareReviewLessonPlanContext((request) => {
        setContextRequest(request);
        setOpenerElement(request.triggerElement || null);
        setRequestedOpen(true);
      }),
    [],
  );

  const effectiveOpen = open || requestedOpen;

  const closeDrawer = useCallback(() => {
    setRequestedOpen(false);
    setContextRequest(null);
    setOpenerElement(null);
    onClose();
  }, [onClose]);

  return (
    <CoursewareLessonPlanContextDrawer
      open={effectiveOpen}
      coursewareId={coursewareId}
      lessonPlanId={lessonPlanId}
      contextRequest={contextRequest}
      openerElement={openerElement}
      storageKey="tedna-cw-review-lesson-plan-drawer-width"
      topOffset={48}
      zIndex={10020}
      title="来源教案对照"
      onClose={closeDrawer}
    />
  );
}
