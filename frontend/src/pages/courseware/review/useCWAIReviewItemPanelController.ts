/**
 * useCWAIReviewItemPanelController.ts
 *
 * 单条教师改进卡的异步状态编排。
 *
 * 从CWAIReviewItemPanel中拆出：
 *   - 候选修改要求/修改方案生成；
 *   - 暂不处理；
 *   - 恢复处理；
 *   - stale重新检查；
 *   - 作者自审最终确认解决；
 *   - 展开态、忙碌态和反馈消息。
 *
 * R-01.1：
 *   - AI生成修改要求/方案成功后只返回Toast消息；
 *   - 非AI人工状态动作仍可在当前问题附近展示结果；
 *   - 错误始终保留在问题附近，便于教师处理和排错。
 *
 * 本Hook不决定卡片如何展示，也不决定正式审核是否交付。
 * 前端守卫只用于体验收窄，实际权限和状态仍由后端重新校验。
 */

import {
  useCallback,
  useState,
} from "react";

import {
  dismissCWAIReviewItem,
  generateCWAIReviewItemInstruction,
  restoreCWAIReviewItem,
  type CWAIReviewItem,
} from "@/api/coursewares";

import {
  type CWAIReviewItemExperience,
  type CWAIReviewItemStateAction,
  resolveCWAIReviewItemExperienceCopy,
} from "./CWAIReviewItemPresentation.shared";
import { useCWAIReviewItemResolutionActions } from "./useCWAIReviewItemResolutionActions";

export interface UseCWAIReviewItemPanelControllerOptions {
  item: CWAIReviewItem;
  experience: CWAIReviewItemExperience;
  canPrepare: boolean;
  onChanged: (item: CWAIReviewItem) => void;
}

export function useCWAIReviewItemPanelController({
  item,
  experience,
  canPrepare,
  onChanged,
}: UseCWAIReviewItemPanelControllerOptions) {
  const [detailsOpen, setDetailsOpen] =
    useState(false);

  const [
    discussionVersion,
    setDiscussionVersion,
  ] =
    useState(0);

  const [generating, setGenerating] =
    useState(false);

  const [
    showPauseForm,
    setShowPauseForm,
  ] =
    useState(false);

  const [
    pauseReason,
    setPauseReason,
  ] =
    useState("");

  const [
    stateAction,
    setStateAction,
  ] =
    useState<CWAIReviewItemStateAction>(
      null,
    );

  const [
    stateError,
    setStateError,
  ] =
    useState("");

  const [
    stateMessage,
    setStateMessage,
  ] =
    useState("");

  const [
    toastMessage,
    setToastMessage,
  ] =
    useState("");

  const copy =
    resolveCWAIReviewItemExperienceCopy(
      experience,
    );

  const stateBusy =
    generating ||
    stateAction !== null;

  const clearToastMessage =
    useCallback(() => {
      setToastMessage("");
    }, []);

  const handlePrepareModification =
    useCallback(async () => {
      if (
        !canPrepare ||
        stateBusy
      ) {
        return;
      }

      setGenerating(true);
      setStateError("");
      setStateMessage("");
      setToastMessage("");

      try {
        const result =
          await generateCWAIReviewItemInstruction(
            item.id,
          );

        onChanged(result.item);

        setDiscussionVersion(
          (previous) =>
            previous + 1,
        );

        setDetailsOpen(true);
        setShowPauseForm(false);

        // AI成功不再在卡片正文里重复铺提示。
        setToastMessage(
          copy.prepareSuccess,
        );
      } catch (cause) {
        setStateError(
          cause instanceof Error
            ? cause.message
            : experience === "review"
              ? "准备修改要求失败"
              : "准备修改方案失败",
        );
      } finally {
        setGenerating(false);
      }
    }, [
      canPrepare,
      copy.prepareSuccess,
      experience,
      item.id,
      onChanged,
      stateBusy,
    ]);

  const handlePause =
    useCallback(async () => {
      const reason =
        pauseReason.trim();

      const reasonLength =
        Array.from(reason).length;

      if (
        !reason ||
        reasonLength > 500 ||
        stateBusy
      ) {
        setStateError(
          reasonLength > 500
            ? "说明不能超过500字"
            : "请补充说明，方便以后回看",
        );
        return;
      }

      setStateAction("dismiss");
      setStateError("");
      setStateMessage("");

      try {
        const result =
          await dismissCWAIReviewItem(
            item.id,
            reason,
          );

        onChanged(result.item);

        setPauseReason("");
        setShowPauseForm(false);
        setDetailsOpen(false);

        setStateMessage(
          experience === "self" &&
            item.status === "applied"
            ? "已暂时不处理这条问题，之后仍可以恢复。"
            : copy.pauseSuccess,
        );
      } catch (cause) {
        setStateError(
          cause instanceof Error
            ? cause.message
            : experience === "review"
              ? "保存本次不退回的决定失败"
              : "保存暂时不处理的决定失败",
        );
      } finally {
        setStateAction(null);
      }
    }, [
      copy.pauseSuccess,
      experience,
      item.id,
      item.status,
      onChanged,
      pauseReason,
      stateBusy,
    ]);

  const handleResume =
    useCallback(async () => {
      if (stateBusy) {
        return;
      }

      setStateAction("restore");
      setStateError("");
      setStateMessage("");

      try {
        const result =
          await restoreCWAIReviewItem(
            item.id,
          );

        onChanged(result.item);

        setDiscussionVersion(
          (previous) =>
            previous + 1,
        );

        setDetailsOpen(false);

        setStateMessage(
          result.item.status === "applied"
            ? "已恢复到修改完成状态，请继续检查、调整或确认已经解决。"
            : result.item.status === "confirmed"
              ? copy.resumeConfirmedSuccess
              : copy.resumePendingSuccess,
        );
      } catch (cause) {
        setStateError(
          cause instanceof Error
            ? cause.message
            : "恢复处理失败",
        );
      } finally {
        setStateAction(null);
      }
    }, [
      copy.resumeConfirmedSuccess,
      copy.resumePendingSuccess,
      item.id,
      onChanged,
      stateBusy,
    ]);

  const {
    handleRecheckItem,
    handleResolveSelfItem,
  } =
    useCWAIReviewItemResolutionActions({
      item,
      experience,
      stateBusy,
      onChanged,
      setDiscussionVersion,
      setDetailsOpen,
      setStateAction,
      setStateError,
      setStateMessage,
    });

  const toggleDetails =
    useCallback(() => {
      setDetailsOpen(
        (previous) =>
          !previous,
      );
    }, []);

  const openDetails =
    useCallback(() => {
      setDetailsOpen(true);
    }, []);

  const togglePauseForm =
    useCallback(() => {
      setShowPauseForm(
        (previous) =>
          !previous,
      );

      setStateError("");
      setStateMessage("");
    }, []);

  return {
    copy,

    detailsOpen,
    setDetailsOpen,
    toggleDetails,
    openDetails,

    discussionVersion,
    generating,

    showPauseForm,
    pauseReason,
    setPauseReason,

    stateAction,
    stateError,
    stateMessage,
    stateBusy,

    toastMessage,
    clearToastMessage,

    handlePrepareModification,
    handlePause,
    handleResume,
    handleRecheckItem,
    handleResolveSelfItem,
    togglePauseForm,
  };
}
