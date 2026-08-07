/**
 * useCWAIReviewItemResolutionActions.ts
 *
 * 单条整改问题中的两类作者人工确认动作：
 *
 *   1. 页面变化后重新检查当前页面；
 *   2. 作者自审问题最终确认解决。
 *
 * 从问题卡片中拆出异步请求和反馈状态编排，避免卡片组件继续膨胀。
 *
 * 业务边界：
 *
 *   - recheck只把当前页面重新登记为applied；
 *   - 正式整改仍需审核员复审；
 *   - 自审问题仍需作者另行确认解决；
 *   - resolve只适用于作者自己的自审问题；
 *   - 浏览器不提交页面HTML、页面指纹或目标状态；
 *   - 全部权限、页面和状态事实由后端重新校验。
 */

import {
  useCallback,
  type Dispatch,
  type SetStateAction,
} from "react";

import {
  recheckCWAIReviewItem,
  resolveSelfCWAIReviewItem,
  type CWAIReviewItem,
} from "@/api/coursewares";

import type {
  CWAIReviewItemExperience,
  CWAIReviewItemStateAction,
} from "./CWAIReviewItemPresentation.shared";

export interface UseCWAIReviewItemResolutionActionsOptions {
  item: CWAIReviewItem;
  experience: CWAIReviewItemExperience;
  stateBusy: boolean;

  onChanged: (
    item: CWAIReviewItem,
  ) => void;

  setDiscussionVersion:
    Dispatch<
      SetStateAction<number>
    >;

  setDetailsOpen:
    Dispatch<
      SetStateAction<boolean>
    >;

  setStateAction:
    Dispatch<
      SetStateAction<
        CWAIReviewItemStateAction
      >
    >;

  setStateError:
    Dispatch<
      SetStateAction<string>
    >;

  setStateMessage:
    Dispatch<
      SetStateAction<string>
    >;
}

export function useCWAIReviewItemResolutionActions({
  item,
  experience,
  stateBusy,
  onChanged,
  setDiscussionVersion,
  setDetailsOpen,
  setStateAction,
  setStateError,
  setStateMessage,
}: UseCWAIReviewItemResolutionActionsOptions) {
  /**
   * 作者重新检查发生变化的当前页面。
   *
   * 本动作只把当前页面重新登记为“修改完成”。
   */
  const handleRecheckItem =
    useCallback(
      async () => {
        if (
          (
            experience !== "self" &&
            experience !==
              "remediation"
          ) ||
          item.status !== "stale" ||
          stateBusy
        ) {
          return;
        }

        const confirmationMessage =
          experience ===
          "remediation"
            ? "请先打开当前页面，对照审核员的整改要求实际检查。确认当前页面已经符合这条要求，并重新进入待复审状态吗？"
            : "请先打开当前页面，对照原修改方案实际检查。确认当前页面仍符合这条方案，并重新进入待确认状态吗？";

        if (
          !window.confirm(
            confirmationMessage,
          )
        ) {
          return;
        }

        setStateAction("recheck");
        setStateError("");
        setStateMessage("");

        try {
          const result =
            await recheckCWAIReviewItem(
              item.id,
            );

          onChanged(
            result.item,
          );

          setDiscussionVersion(
            (previous) =>
              previous + 1,
          );

          setDetailsOpen(false);

          setStateMessage(
            result.message ||
              (
                experience ===
                "remediation"
                  ? "已重新检查当前页面，等待审核员复审"
                  : "已重新检查当前页面，等待最终确认"
              ),
          );
        } catch (cause) {
          setStateError(
            cause instanceof Error
              ? cause.message
              : "重新检查当前页面失败",
          );
        } finally {
          setStateAction(null);
        }
      },
      [
        experience,
        item.id,
        item.status,
        onChanged,
        setDetailsOpen,
        setDiscussionVersion,
        setStateAction,
        setStateError,
        setStateMessage,
        stateBusy,
      ],
    );

  /**
   * 作者检查当前页面后，明确确认自审问题已经解决。
   */
  const handleResolveSelfItem =
    useCallback(
      async () => {
        if (
          experience !== "self" ||
          item.status !== "applied" ||
          stateBusy
        ) {
          return;
        }

        const confirmed =
          window.confirm(
            "请先实际检查当前页面的教学内容和互动效果。确认这个问题已经真正解决吗？",
          );

        if (!confirmed) {
          return;
        }

        setStateAction("resolve");
        setStateError("");
        setStateMessage("");

        try {
          const result =
            await resolveSelfCWAIReviewItem(
              item.id,
            );

          onChanged(
            result.item,
          );

          setDetailsOpen(false);

          setStateMessage(
            result.message ||
              "已确认这条自审问题已经解决",
          );
        } catch (cause) {
          setStateError(
            cause instanceof Error
              ? cause.message
              : "确认问题已经解决失败",
          );
        } finally {
          setStateAction(null);
        }
      },
      [
        experience,
        item.id,
        item.status,
        onChanged,
        setDetailsOpen,
        setStateAction,
        setStateError,
        setStateMessage,
        stateBusy,
      ],
    );

  return {
    handleRecheckItem,
    handleResolveSelfItem,
  };
}
