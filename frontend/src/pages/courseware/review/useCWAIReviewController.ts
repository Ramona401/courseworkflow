/**
 * useCWAIReviewController.ts
 *
 * 课件AI审核与作者自审主控制器。
 *
 * 本文件只负责页面级状态和各独立Hook的编排：
 *   - 加载后端最新会话；
 *   - 计算批次进度和最终报告；
 *   - 向正式审核工作台回传已选整改项；
 *   - 管理助手选择和助手管理入口；
 *   - 向Panel暴露整改项数据库真值刷新入口。
 *
 * 职责拆分：
 *   - useCWAIReviewConfigState.ts：R-02配置草稿和会话快照；
 *   - useCWAIReviewExecution.ts：会话创建、批次执行和失败恢复；
 *   - useCWAIReviewItemsState.ts：整改项、数据库真值刷新和正式交付选择；
 *   - cwAIReviewControllerHelpers.ts：纯合并和筛选函数。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { useNavigate } from "react-router-dom";

import {
  getLatestCWAIReview,
  parseCWAIReviewFinalReport,
  type CWAIReviewItem,
  type CWAIReviewSessionBundle,
} from "@/api/coursewares";

import {
  collectCWAIReviewBatchFindings,
  isDeliverableFormalReviewItem,
} from "./cwAIReviewControllerHelpers";

import {
  useCWAIReviewConfigState,
} from "./useCWAIReviewConfigState";

import {
  useCWAIReviewExecution,
} from "./useCWAIReviewExecution";

import {
  useCWAIReviewItemsState,
} from "./useCWAIReviewItemsState";

export type CWAIReviewPanelMode =
  | "formal"
  | "self";

export interface CWAIReviewContext {
  sessionId: string | null;
  selectedItemIds: string[];
  selectedItems: CWAIReviewItem[];

  /**
   * 只把整改项从本次正式交付选择中移出。
   *
   * 不删除整改项、不改变状态，也不等同于“忽略此问题”。
   */
  removeSelectedItem?: (
    itemId: string,
  ) => void;
}

export interface UseCWAIReviewControllerOptions {
  coursewareId: string;
  subject: string;
  grade: string;
  reviewLevel: number;
  mode: CWAIReviewPanelMode;

  onReviewContextChange?: (
    context: CWAIReviewContext,
  ) => void;
}

export function useCWAIReviewController({
  coursewareId,
  subject,
  reviewLevel,
  mode,
  onReviewContextChange,
}: UseCWAIReviewControllerOptions) {
  const navigate = useNavigate();

  const isSelfReview =
    mode === "self";

  const assistantScene =
    isSelfReview
      ? "courseware_self_review"
      : "courseware_review";

  const actionName =
    isSelfReview
      ? "自审"
      : "审核";

  const panelTitle =
    isSelfReview
      ? "AI课件自审助手"
      : "AI课件审核助手";

  const mountedRef =
    useRef(true);

  const [
    assistantId,
    setAssistantId,
  ] = useState<string | null>(null);

  const [
    bundle,
    setBundle,
  ] = useState<CWAIReviewSessionBundle>({
    session: null,
    batches: [],
  });

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    running,
    setRunning,
  ] = useState(false);

  const [
    error,
    setError,
  ] = useState("");

  useEffect(() => {
    mountedRef.current = true;

    return () => {
      mountedRef.current = false;
    };
  }, []);

  const session = bundle.session;

  const completedBatches =
    bundle.batches.filter(
      (batch) =>
        batch.status === "done",
    ).length;

  const totalBatches =
    session?.total_batches ||
    bundle.batches.length ||
    0;

  const progress =
    totalBatches > 0
      ? Math.min(
          100,
          Math.round(
            (
              completedBatches /
              totalBatches
            ) * 100,
          ),
        )
      : 0;

  const canContinue =
    session?.status === "reviewing" ||
    session?.status === "aggregating";

  const canRestart =
    !session ||
    session.status === "done" ||
    session.status === "failed" ||
    session.status === "cancelled" ||
    session.status === "preparing";

  const reviewConfig =
    useCWAIReviewConfigState({
      session,
      canRestart,
      running,
    });

  const reviewItems =
    useCWAIReviewItemsState({
      session,
      isSelfReview,
      mountedRef,
      setError,
    });

  /**
   * 正式交付只暴露“仍然可交付且同时存在于当前问题集合”的规范化交集。
   *
   * 顶部清单数量、实际展示数量和提交payload的review_item_ids
   * 始终共享同一事实源；状态变化或异步刷新造成的孤立ID不会进入正式提交。
   */
  const deliverySelectedItems =
    useMemo(() => {
      if (isSelfReview) {
        return [];
      }

      const selectedIDSet =
        new Set(
          reviewItems.selectedItemIds,
        );

      return reviewItems.items.filter(
        (item) =>
          selectedIDSet.has(item.id) &&
          isDeliverableFormalReviewItem(item),
      );
    }, [
      isSelfReview,
      reviewItems.items,
      reviewItems.selectedItemIds,
    ]);

  const deliverySelectedItemIds =
    useMemo(
      () =>
        deliverySelectedItems.map(
          (item) => item.id,
        ),
      [deliverySelectedItems],
    );

  const loadLatest =
    useCallback(
      async () => {
        setLoading(true);
        setError("");

        try {
          const result =
            await getLatestCWAIReview(
              coursewareId,
              reviewLevel,
            );

          if (!mountedRef.current) {
            return;
          }

          setBundle({
            session: result.session,
            batches: result.batches || [],
          });

          setAssistantId(
            result.session?.assistant_id ||
              null,
          );

          reviewConfig
            .applyLoadedSessionConfig(
              result.session,
            );

          if (
            !result.session ||
            result.session.status !== "done"
          ) {
            reviewItems
              .resetReviewItems();
          }
        } catch (cause) {
          if (!mountedRef.current) {
            return;
          }

          setError(
            cause instanceof Error
              ? cause.message
              : "加载AI审核记录失败",
          );
        } finally {
          if (mountedRef.current) {
            setLoading(false);
          }
        }
      },
      [
        coursewareId,
        reviewConfig.applyLoadedSessionConfig,
        reviewItems.resetReviewItems,
        reviewLevel,
      ],
    );

  useEffect(() => {
    void loadLatest();
  }, [loadLatest]);

  useEffect(() => {
    if (!onReviewContextChange) {
      return;
    }

    if (
      !session ||
      session.status !== "done"
    ) {
      onReviewContextChange({
        sessionId: null,
        selectedItemIds: [],
        selectedItems: [],
        removeSelectedItem:
          reviewItems.handleRemoveSelectedItem,
      });

      return;
    }

    onReviewContextChange({
      sessionId: session.id,
      selectedItemIds:
        deliverySelectedItemIds,
      selectedItems:
        deliverySelectedItems,
      removeSelectedItem:
        reviewItems.handleRemoveSelectedItem,
    });
  }, [
    deliverySelectedItemIds,
    deliverySelectedItems,
    onReviewContextChange,
    reviewItems.handleRemoveSelectedItem,
    session,
  ]);

  const finalReport =
    useMemo(
      () =>
        parseCWAIReviewFinalReport(
          session,
        ),
      [session],
    );

  const batchFindings =
    useMemo(
      () =>
        collectCWAIReviewBatchFindings(
          bundle.batches,
        ),
      [bundle.batches],
    );

  const execution =
    useCWAIReviewExecution({
      coursewareId,
      reviewLevel,
      assistantId,

      bundle,
      setBundle,

      running,
      setRunning,

      setError,
      mountedRef,

      reviewConfigDraft:
        reviewConfig.reviewConfigDraft,

      reviewConfigError:
        reviewConfig.reviewConfigError,

      applyLoadedSessionConfig:
        reviewConfig.applyLoadedSessionConfig,

      resetReviewItems:
        reviewItems.resetReviewItems,
    });

  const handleManageAssistants =
    useCallback(() => {
      const query =
        new URLSearchParams({
          scene: assistantScene,
          subject,
          drawer: "1",
        });

      if (assistantId) {
        query.set(
          "assistant_id",
          assistantId,
        );
      }

      navigate(
        `/lesson-plans/my-assistants?${query.toString()}`,
      );
    }, [
      assistantId,
      assistantScene,
      navigate,
      subject,
    ]);

  return {
    isSelfReview,
    assistantScene,
    actionName,
    panelTitle,

    assistantId,
    setAssistantId,

    reviewConfigDraft:
      reviewConfig.reviewConfigDraft,

    sessionReviewConfig:
      reviewConfig.sessionReviewConfig,

    reviewConfigEditable:
      reviewConfig.reviewConfigEditable,

    reviewConfigError:
      reviewConfig.reviewConfigError,

    handleToggleReviewDimension:
      reviewConfig.handleToggleReviewDimension,

    setCustomDimensionDescription:
      reviewConfig.setCustomDimensionDescription,

    setLessonReferenceMode:
      reviewConfig.setLessonReferenceMode,

    bundle,
    session,

    items:
      reviewItems.items,

    selectedItemIds:
      deliverySelectedItemIds,

    materializingFindingIds:
      reviewItems.materializingFindingIds,

    loading,
    running,
    error,

    finalReport,
    batchFindings,
    completedBatches,
    totalBatches,
    progress,
    canContinue,
    canRestart,

    handleStart:
      execution.handleStart,

    handleContinue:
      execution.handleContinue,

    handleAdoptFinding:
      reviewItems.handleAdoptFinding,

    handleToggleItemSelection:
      reviewItems.handleToggleItemSelection,

    handleRemoveSelectedItem:
      reviewItems.handleRemoveSelectedItem,

    handleItemChanged:
      reviewItems.handleItemChanged,

    /**
     * R-07 Atomic Apply完成后供Panel重新读取整改项数据库真值。
     *
     * 该入口不接受Impact Plan payload，也不根据浏览器状态推断整改项结果。
     */
    refreshSessionItems:
      reviewItems.refreshSessionItems,

    handleManageAssistants,
  };
}

export type CWAIReviewController =
  ReturnType<
    typeof useCWAIReviewController
  >;
