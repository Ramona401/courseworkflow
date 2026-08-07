/**
 * useCWAIReviewConfigState.ts
 *
 * R-02审核维度与教案参考模式的前端状态。
 *
 * 职责：
 *   - 维护新会话配置草稿；
 *   - 从后端不可变会话快照恢复展示状态；
 *   - 校验至少选择一个维度；
 *   - 校验custom维度说明；
 *   - 控制活动会话期间配置只读；
 *   - 按平台固定顺序保存维度代码。
 *
 * 本Hook不调用创建、批次执行或最终汇总API。
 */

import {
  useCallback,
  useMemo,
  useState,
} from "react";

import {
  createDefaultCWAIReviewConfigDraft,
  readCWAIReviewSessionConfig,
  sortCWAIReviewDimensions,
  toCWAIReviewConfigDraft,
  type CWAIReviewDimension,
  type CWAIReviewLessonReferenceMode,
} from "@/api/coursewares.ai-review-config";

import type {
  CWAIReviewSession,
} from "@/api/coursewares";

export interface UseCWAIReviewConfigStateOptions {
  session: CWAIReviewSession | null;
  canRestart: boolean;
  running: boolean;
}

export function useCWAIReviewConfigState({
  session,
  canRestart,
  running,
}: UseCWAIReviewConfigStateOptions) {
  const [
    reviewConfigDraft,
    setReviewConfigDraft,
  ] = useState(
    createDefaultCWAIReviewConfigDraft,
  );

  const sessionReviewConfig = useMemo(
    () =>
      readCWAIReviewSessionConfig(
        session,
      ),
    [session],
  );

  const reviewConfigEditable =
    canRestart && !running;

  const reviewConfigError = useMemo(
    () => {
      if (
        reviewConfigDraft
          .review_dimensions
          .length === 0
      ) {
        return "请至少选择一个审核维度";
      }

      if (
        reviewConfigDraft
          .review_dimensions
          .includes("custom") &&
        !reviewConfigDraft
          .custom_dimension_description
          .trim()
      ) {
        return "请选择自定义维度后填写具体审核要求";
      }

      return "";
    },
    [reviewConfigDraft],
  );

  const applyLoadedSessionConfig =
    useCallback(
      (
        nextSession:
          | CWAIReviewSession
          | null,
      ) => {
        const snapshot =
          readCWAIReviewSessionConfig(
            nextSession,
          );

        setReviewConfigDraft(
          snapshot
            ? toCWAIReviewConfigDraft(
                snapshot,
              )
            : createDefaultCWAIReviewConfigDraft(),
        );
      },
      [],
    );

  const handleToggleReviewDimension =
    useCallback(
      (
        dimension:
          CWAIReviewDimension,
        selected: boolean,
      ) => {
        setReviewConfigDraft(
          (previous) => {
            const nextDimensions =
              selected
                ? sortCWAIReviewDimensions([
                    ...previous
                      .review_dimensions,
                    dimension,
                  ])
                : previous
                    .review_dimensions
                    .filter(
                      (current) =>
                        current !== dimension,
                    );

            return {
              ...previous,
              review_dimensions:
                nextDimensions,

              custom_dimension_description:
                dimension === "custom" &&
                !selected
                  ? ""
                  : previous
                      .custom_dimension_description,
            };
          },
        );
      },
      [],
    );

  const setCustomDimensionDescription =
    useCallback(
      (
        description: string,
      ) => {
        setReviewConfigDraft(
          (previous) => ({
            ...previous,
            custom_dimension_description:
              description,
          }),
        );
      },
      [],
    );

  const setLessonReferenceMode =
    useCallback(
      (
        lessonReferenceMode:
          CWAIReviewLessonReferenceMode,
      ) => {
        setReviewConfigDraft(
          (previous) => ({
            ...previous,
            lesson_reference_mode:
              lessonReferenceMode,
          }),
        );
      },
      [],
    );

  return {
    reviewConfigDraft,
    sessionReviewConfig,
    reviewConfigEditable,
    reviewConfigError,

    applyLoadedSessionConfig,
    handleToggleReviewDimension,
    setCustomDimensionDescription,
    setLessonReferenceMode,
  };
}

export type CWAIReviewConfigState =
  ReturnType<
    typeof useCWAIReviewConfigState
  >;
