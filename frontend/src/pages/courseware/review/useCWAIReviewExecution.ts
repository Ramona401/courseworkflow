/**
 * useCWAIReviewExecution.ts
 *
 * 课件AI审核会话创建、分批执行、失败恢复和最终综合编排。
 *
 * 本Hook负责：
 *   - 使用当前R-02配置创建新会话；
 *   - 自动顺序执行页面批次；
 *   - 在aggregating阶段生成最终报告；
 *   - 失败后重新读取后端最新会话；
 *   - 保留已完成批次，不在浏览器重建后端状态。
 *
 * 本Hook不管理整改项讨论和正式交付选择。
 */

import {
  useCallback,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from "react";

import {
  prepareConfiguredCWAIReview,
  type CWAIReviewConfigDraft,
} from "@/api/coursewares.ai-review-config";

import {
  finalizeCWAIReview,
  getLatestCWAIReview,
  runNextCWAIReviewBatch,
  type CWAIReviewBatch,
  type CWAIReviewSession,
  type CWAIReviewSessionBundle,
} from "@/api/coursewares";

import {
  mergeCWAIReviewBatch,
} from "./cwAIReviewControllerHelpers";

export interface UseCWAIReviewExecutionOptions {
  coursewareId: string;
  reviewLevel: number;
  assistantId: string | null;

  bundle: CWAIReviewSessionBundle;
  setBundle: Dispatch<
    SetStateAction<CWAIReviewSessionBundle>
  >;

  running: boolean;
  setRunning: Dispatch<
    SetStateAction<boolean>
  >;

  setError: Dispatch<
    SetStateAction<string>
  >;

  mountedRef: MutableRefObject<boolean>;

  reviewConfigDraft: CWAIReviewConfigDraft;
  reviewConfigError: string;

  applyLoadedSessionConfig: (
    session: CWAIReviewSession | null,
  ) => void;

  resetReviewItems: () => void;
}

export function useCWAIReviewExecution({
  coursewareId,
  reviewLevel,
  assistantId,
  bundle,
  setBundle,
  running,
  setRunning,
  setError,
  mountedRef,
  reviewConfigDraft,
  reviewConfigError,
  applyLoadedSessionConfig,
  resetReviewItems,
}: UseCWAIReviewExecutionOptions) {
  const updateFromRun = useCallback(
    (
      nextSession: CWAIReviewSession,
      nextBatch: CWAIReviewBatch | null,
    ) => {
      setBundle((previous) => ({
        session: nextSession,
        batches: mergeCWAIReviewBatch(
          previous.batches,
          nextBatch,
        ),
      }));
    },
    [setBundle],
  );

  const runUntilComplete = useCallback(
    async (
      initial: CWAIReviewSessionBundle,
    ) => {
      let currentSession =
        initial.session;

      if (!currentSession) {
        throw new Error(
          "AI分析会话创建失败",
        );
      }

      for (
        let step = 0;
        step < 100;
        step += 1
      ) {
        if (
          currentSession.status ===
          "aggregating"
        ) {
          const finalized =
            await finalizeCWAIReview(
              currentSession.id,
            );

          if (!mountedRef.current) {
            return;
          }

          setBundle((previous) => ({
            session:
              finalized.session,
            batches:
              previous.batches,
          }));

          return;
        }

        if (
          currentSession.status !==
          "reviewing"
        ) {
          return;
        }

        const next =
          await runNextCWAIReviewBatch(
            currentSession.id,
          );

        currentSession =
          next.session;

        if (!mountedRef.current) {
          return;
        }

        updateFromRun(
          next.session,
          next.batch,
        );

        if (
          next.requires_finalize ||
          next.session.status ===
            "aggregating"
        ) {
          const finalized =
            await finalizeCWAIReview(
              next.session.id,
            );

          if (!mountedRef.current) {
            return;
          }

          setBundle((previous) => ({
            session:
              finalized.session,
            batches:
              previous.batches,
          }));

          return;
        }

        if (!next.has_more) {
          return;
        }
      }

      throw new Error(
        "AI分析批次数量异常，已停止自动执行",
      );
    },
    [
      mountedRef,
      setBundle,
      updateFromRun,
    ],
  );

  const reloadAfterFailure =
    useCallback(
      async () => {
        try {
          const latest =
            await getLatestCWAIReview(
              coursewareId,
              reviewLevel,
            );

          if (!mountedRef.current) {
            return;
          }

          setBundle(latest);

          applyLoadedSessionConfig(
            latest.session,
          );
        } catch {
          // 保留原始错误。
        }
      },
      [
        applyLoadedSessionConfig,
        coursewareId,
        mountedRef,
        reviewLevel,
        setBundle,
      ],
    );

  const handleStart =
    useCallback(
      async () => {
        if (running) {
          return;
        }

        if (reviewConfigError) {
          setError(
            reviewConfigError,
          );
          return;
        }

        setRunning(true);
        setError("");
        resetReviewItems();

        try {
          const prepared =
            await prepareConfiguredCWAIReview(
              {
                courseware_id:
                  coursewareId,

                review_level:
                  reviewLevel,

                assistant_id:
                  assistantId || "",

                review_dimensions: [
                  ...reviewConfigDraft
                    .review_dimensions,
                ],

                custom_dimension_description:
                  reviewConfigDraft
                    .custom_dimension_description
                    .trim(),

                lesson_reference_mode:
                  reviewConfigDraft
                    .lesson_reference_mode,
              },
            );

          if (!mountedRef.current) {
            return;
          }

          setBundle(prepared);

          applyLoadedSessionConfig(
            prepared.session,
          );

          await runUntilComplete(
            prepared,
          );
        } catch (cause) {
          if (!mountedRef.current) {
            return;
          }

          setError(
            cause instanceof Error
              ? cause.message
              : "课件AI分析执行失败",
          );

          await reloadAfterFailure();
        } finally {
          if (mountedRef.current) {
            setRunning(false);
          }
        }
      },
      [
        applyLoadedSessionConfig,
        assistantId,
        coursewareId,
        mountedRef,
        reloadAfterFailure,
        resetReviewItems,
        reviewConfigDraft,
        reviewConfigError,
        reviewLevel,
        runUntilComplete,
        running,
        setBundle,
        setError,
        setRunning,
      ],
    );

  const handleContinue =
    useCallback(
      async () => {
        if (
          running ||
          !bundle.session
        ) {
          return;
        }

        setRunning(true);
        setError("");

        try {
          await runUntilComplete(
            bundle,
          );
        } catch (cause) {
          if (!mountedRef.current) {
            return;
          }

          setError(
            cause instanceof Error
              ? cause.message
              : "继续AI分析失败",
          );

          await reloadAfterFailure();
        } finally {
          if (mountedRef.current) {
            setRunning(false);
          }
        }
      },
      [
        bundle,
        mountedRef,
        reloadAfterFailure,
        runUntilComplete,
        running,
        setError,
        setRunning,
      ],
    );

  return {
    handleStart,
    handleContinue,
  };
}
