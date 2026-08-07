/**
 * useCoursewareLessonPlanContent.ts
 *
 * 来源教案正文的按需读取 Hook。
 *
 * 生命周期规则：
 *   - loading 由当前请求键与已完成状态确定性推导，不在 Effect 中同步 setState；
 *   - 关闭抽屉、切换课件或重试时使用 AbortController 取消旧请求；
 *   - 父组件回调通过 ref 获取最新值，回调引用变化不会取消有效请求；
 *   - 失败后只有教师点击“重新读取”才创建新的请求键。
 */

import { useCallback, useEffect, useRef, useState } from "react";

import {
  getCoursewareLessonPlanContent,
  type CoursewareLessonPlanContent,
} from "@/api/coursewares";

interface UseCoursewareLessonPlanContentOptions {
  open: boolean;
  coursewareId: string;
  onHasLessonPlanChange?: (hasLessonPlan: boolean) => void;
}

type CoursewareLessonPlanRequestStatus = "idle" | "success" | "error";

interface CoursewareLessonPlanRequestState {
  key: string;
  status: CoursewareLessonPlanRequestStatus;
  data: CoursewareLessonPlanContent | null;
  error: string;
}

const INITIAL_REQUEST_STATE: CoursewareLessonPlanRequestState = {
  key: "",
  status: "idle",
  data: null,
  error: "",
};

export function useCoursewareLessonPlanContent({
  open,
  coursewareId,
  onHasLessonPlanChange,
}: UseCoursewareLessonPlanContentOptions) {
  const [loadAttempt, setLoadAttempt] = useState(0);
  const [requestState, setRequestState] =
    useState<CoursewareLessonPlanRequestState>(INITIAL_REQUEST_STATE);

  const requestedKeyRef = useRef("");
  const onHasLessonPlanChangeRef = useRef(onHasLessonPlanChange);
  const normalizedCoursewareID = coursewareId.trim();
  const activeRequestKey = `${normalizedCoursewareID}\u001f${loadAttempt}`;

  useEffect(() => {
    onHasLessonPlanChangeRef.current = onHasLessonPlanChange;
  }, [onHasLessonPlanChange]);

  useEffect(() => {
    if (!open || !normalizedCoursewareID) return;

    if (
      requestState.key === activeRequestKey &&
      requestState.status !== "idle"
    ) {
      return;
    }

    if (requestedKeyRef.current === activeRequestKey) return;

    requestedKeyRef.current = activeRequestKey;
    const controller = new AbortController();

    void getCoursewareLessonPlanContent(
      normalizedCoursewareID,
      controller.signal,
    )
      .then((result) => {
        if (controller.signal.aborted) return;

        setRequestState({
          key: activeRequestKey,
          status: "success",
          data: result,
          error: "",
        });
        onHasLessonPlanChangeRef.current?.(result.has_lesson_plan);
      })
      .catch((cause) => {
        if (controller.signal.aborted) return;

        setRequestState({
          key: activeRequestKey,
          status: "error",
          data: null,
          error:
            cause instanceof Error ? cause.message : "读取来源教案失败",
        });
      });

    return () => {
      controller.abort();
      if (requestedKeyRef.current === activeRequestKey) {
        requestedKeyRef.current = "";
      }
    };
  }, [
    activeRequestKey,
    normalizedCoursewareID,
    open,
    requestState.key,
    requestState.status,
  ]);

  const currentState =
    requestState.key === activeRequestKey ? requestState : null;
  const data = currentState?.status === "success" ? currentState.data : null;
  const error = currentState?.status === "error" ? currentState.error : "";
  const loading = open && Boolean(normalizedCoursewareID) && currentState === null;

  const retryLoad = useCallback(() => {
    setLoadAttempt((current) => current + 1);
  }, []);

  return {
    data,
    loading,
    error,
    retryLoad,
  };
}
