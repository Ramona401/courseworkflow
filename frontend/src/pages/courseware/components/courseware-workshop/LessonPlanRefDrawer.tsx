/**
 * LessonPlanRefDrawer.tsx
 *
 * 课件工坊中的原教案按需对照入口。
 *
 * 默认只显示底部横向按钮，不长期挤压课件主区域；单问题工作区触发共享事件时，
 * 可直接打开抽屉并定位相关教案章节。抽屉正文与交互复用统一组件。
 *
 * 状态隔离规则：
 *   - 外层组件使用coursewareId作为key；
 *   - 切换课件时由React卸载旧状态并创建新状态；
 *   - 不在Effect中同步执行多次setState；
 *   - JSX渲染期间不读取ref.current；
 *   - 焦点恢复目标只从真实点击事件或共享问题事件中保存。
 */

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import CoursewareLessonPlanContextDrawer from "@/pages/courseware/review/CoursewareLessonPlanContextDrawer";
import {
  subscribeCoursewareReviewLessonPlanContext,
  type CoursewareReviewLessonPlanContextRequest,
} from "@/pages/courseware/review/coursewareReviewLessonPlanContext";

interface Props {
  coursewareId: string;
}

export default function LessonPlanRefDrawer({
  coursewareId,
}: Props) {
  return (
    <LessonPlanRefDrawerState
      key={coursewareId}
      coursewareId={coursewareId}
    />
  );
}

function LessonPlanRefDrawerState({
  coursewareId,
}: Props) {
  const [open, setOpen] = useState(false);
  const [hiddenEntry, setHiddenEntry] = useState(false);
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
        setOpen(true);
      }),
    [],
  );

  const closeDrawer = useCallback(() => {
    setOpen(false);
    setContextRequest(null);
    setOpenerElement(null);
  }, []);

  const handleHasLessonPlanChange = useCallback(
    (hasLessonPlan: boolean) => {
      setHiddenEntry(!hasLessonPlan);
    },
    [],
  );

  if (hiddenEntry && !open) {
    return null;
  }

  return (
    <>
      {!open && (
        <button
          type="button"
          onClick={(event) => {
            setContextRequest(null);
            setOpenerElement(event.currentTarget);
            setOpen(true);
          }}
          aria-haspopup="dialog"
          aria-expanded={false}
          title="按需打开原教案对照"
          style={{
            position: "fixed",
            right: 24,
            bottom: 24,
            zIndex: 999,
            minHeight: 40,
            display: "inline-flex",
            alignItems: "center",
            gap: 8,
            padding: "9px 14px",
            border: "1px solid #C4B5FD",
            borderRadius: 10,
            background: "#FFFFFF",
            color: "#6D28D9",
            fontSize: 14,
            fontWeight: 700,
            cursor: "pointer",
            boxShadow: "0 6px 20px rgba(76,29,149,0.18)",
          }}
        >
          <span
            aria-hidden="true"
            style={{ fontSize: 17 }}
          >
            📄
          </span>
          查看原教案
        </button>
      )}

      <CoursewareLessonPlanContextDrawer
        open={open}
        coursewareId={coursewareId}
        contextRequest={contextRequest}
        openerElement={openerElement}
        storageKey="tedna-courseware-workshop-lesson-plan-drawer-width"
        topOffset={0}
        zIndex={1000}
        title="原教案对照"
        onClose={closeDrawer}
        onHasLessonPlanChange={handleHasLessonPlanChange}
      />
    </>
  );
}
