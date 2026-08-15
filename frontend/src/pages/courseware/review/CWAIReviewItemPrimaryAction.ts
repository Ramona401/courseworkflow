/**
 * CWAIReviewItemPrimaryAction.ts
 *
 * 教师改进卡“唯一主要操作”的纯决策函数。
 *
 * PRD主操作：
 *   - 正式审核：要求修改 → 确认并加入本次修改清单；
 *   - 作者自审：请 AI 帮我完善；
 *   - 作者正式整改：开始修改；
 *   - 自审修改完成：确认已经解决；
 *   - 页面内容已变化：重新检查当前页面；原页面已不存在：检查相关页面。
 *
 * 本文件只选择动作，不发请求、不执行权限判断。
 */

import type { CWAIReviewItem } from "@/api/coursewares";

import {
  resolveCWAIReviewPageChangeTeacherCopy,
  type CWAIReviewItemExperience,
  type CWAIReviewItemExperienceCopy,
  type CWAIReviewItemStateAction,
  type CWAIReviewPrimaryActionTone,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewItemPrimaryAction {
  label: string;
  tone: CWAIReviewPrimaryActionTone;
  opensDetails: boolean;
  onClick: () => void;
}

export interface ResolveCWAIReviewItemPrimaryActionOptions {
  item: CWAIReviewItem;
  experience: CWAIReviewItemExperience;
  copy: CWAIReviewItemExperienceCopy;
  stateAction: CWAIReviewItemStateAction;
  generating: boolean;

  selected: boolean;
  canSelectForReturn: boolean;
  onSelectedChange?: (
    itemID: string,
    selected: boolean,
  ) => void;

  canResume: boolean;
  canOpenPageModification: boolean;
  onInjectToRefine?: (
    item: CWAIReviewItem,
  ) => void;

  openDetails: () => void;
  handleResume: () => Promise<void>;
  handlePrepareModification: () => Promise<void>;
  handleRecheckItem: () => Promise<void>;
  handleResolveSelfItem: () => Promise<void>;
}

export function resolveCWAIReviewItemPrimaryAction({
  item,
  experience,
  copy,
  stateAction,
  generating,
  selected,
  canSelectForReturn,
  onSelectedChange,
  canResume,
  canOpenPageModification,
  onInjectToRefine,
  openDetails,
  handleResume,
  handlePrepareModification,
  handleRecheckItem,
  handleResolveSelfItem,
}: ResolveCWAIReviewItemPrimaryActionOptions): CWAIReviewItemPrimaryAction {
  const pageChangeCopy =
    resolveCWAIReviewPageChangeTeacherCopy(
      item.status,
    );

  if (experience === "review") {
    if (canResume) {
      return {
        label:
          stateAction === "restore"
            ? "正在恢复…"
            : copy.resumeAction,
        tone: "primary",
        opensDetails: false,
        onClick: () =>
          void handleResume(),
      };
    }

    switch (item.status) {
      case "detected":
        return {
          label: generating
            ? "正在准备修改要求…"
            : "要求修改",
          tone: "primary",
          opensDetails: false,
          onClick: () =>
            void handlePrepareModification(),
        };

      case "discussing":
        return {
          label: "继续完善修改要求",
          tone: "primary",
          opensDetails: true,
          onClick: openDetails,
        };

      case "confirmed":
        if (
          canSelectForReturn &&
          onSelectedChange
        ) {
          return selected
            ? {
                label: "从本次修改清单移出",
                tone: "warning",
                opensDetails: false,
                onClick: () =>
                  onSelectedChange(
                    item.id,
                    false,
                  ),
              }
            : {
                label: "确认并加入本次修改清单",
                tone: "success",
                opensDetails: false,
                onClick: () =>
                  onSelectedChange(
                    item.id,
                    true,
                  ),
              };
        }

        return {
          label: "查看当前修改要求",
          tone: "primary",
          opensDetails: true,
          onClick: openDetails,
        };

      case "stale":
        return {
          label:
            `${pageChangeCopy?.label || "页面内容已变化"}，查看情况`,
          tone: "warning",
          opensDetails: true,
          onClick: openDetails,
        };

      case "orphaned":
        return {
          label:
            `${pageChangeCopy?.label || "原页面已不存在"}，查看情况`,
          tone: "warning",
          opensDetails: true,
          onClick: openDetails,
        };

      case "resolved":
        return {
          label: "查看已解决记录",
          tone: "neutral",
          opensDetails: true,
          onClick: openDetails,
        };

      default:
        return {
          label: "查看处理情况",
          tone: "neutral",
          opensDetails: true,
          onClick: openDetails,
        };
    }
  }

  if (
    experience ===
    "remediation"
  ) {
    if (
      item.status ===
        "confirmed" &&
      canOpenPageModification
    ) {
      return {
        label: "开始修改",
        tone: "success",
        opensDetails: false,
        onClick: () =>
          onInjectToRefine?.(
            item,
          ),
      };
    }

    if (
      item.status ===
        "applying" &&
      onInjectToRefine
    ) {
      return {
        label: "继续修改",
        tone: "warning",
        opensDetails: false,
        onClick: () =>
          onInjectToRefine(
            item,
          ),
      };
    }

    if (
      item.status ===
      "applied"
    ) {
      return {
        label: "已完成修改，等待复审",
        tone: "primary",
        opensDetails: true,
        onClick: openDetails,
      };
    }

    if (
      item.status ===
      "stale"
    ) {
      return {
        label:
          stateAction ===
          "recheck"
            ? "正在检查…"
            : "重新检查当前页面",
        tone: "warning",
        opensDetails: false,
        onClick: () =>
          void handleRecheckItem(),
      };
    }

    if (
      item.status ===
      "orphaned"
    ) {
      return {
        label:
          `${pageChangeCopy?.label || "原页面已不存在"}，检查相关页面`,
        tone: "neutral",
        opensDetails: true,
        onClick: openDetails,
      };
    }

    if (
      item.status ===
      "resolved"
    ) {
      return {
        label: "查看已解决记录",
        tone: "neutral",
        opensDetails: true,
        onClick: openDetails,
      };
    }

    return {
      label: "查看当前修改要求",
      tone: "primary",
      opensDetails: true,
      onClick: openDetails,
    };
  }

  if (canResume) {
    return {
      label:
        stateAction ===
        "restore"
          ? "正在恢复…"
          : copy.resumeAction,
      tone: "primary",
      opensDetails: false,
      onClick: () =>
        void handleResume(),
    };
  }

  if (
    item.status ===
    "detected"
  ) {
    return {
      label: generating
        ? "AI 正在完善…"
        : "请 AI 帮我完善",
      tone: "primary",
      opensDetails: false,
      onClick: () =>
        void handlePrepareModification(),
    };
  }

  if (
    item.status ===
    "discussing"
  ) {
    return {
      label: "继续完善修改方案",
      tone: "primary",
      opensDetails: true,
      onClick: openDetails,
    };
  }

  if (
    item.status ===
      "confirmed" &&
    canOpenPageModification
  ) {
    return {
      label: "开始修改",
      tone: "success",
      opensDetails: false,
      onClick: () =>
        onInjectToRefine?.(
          item,
        ),
    };
  }

  if (
    item.status ===
    "confirmed"
  ) {
    return {
      label: "查看当前修改方案",
      tone: "success",
      opensDetails: true,
      onClick: openDetails,
    };
  }

  if (
    item.status ===
    "applying"
  ) {
    return {
      label: "继续修改",
      tone: "warning",
      opensDetails: true,
      onClick: openDetails,
    };
  }

  if (
    item.status ===
    "applied"
  ) {
    return {
      label:
        stateAction ===
        "resolve"
          ? "正在确认…"
          : "确认已经解决",
      tone: "success",
      opensDetails: false,
      onClick: () =>
        void handleResolveSelfItem(),
    };
  }

  if (
    item.status ===
    "stale"
  ) {
    return {
      label:
        stateAction ===
        "recheck"
          ? "正在检查…"
          : "重新检查当前页面",
      tone: "warning",
      opensDetails: false,
      onClick: () =>
        void handleRecheckItem(),
    };
  }

  if (
    item.status ===
    "resolved"
  ) {
    return {
      label: "查看已解决记录",
      tone: "neutral",
      opensDetails: true,
      onClick: openDetails,
    };
  }

  return {
    label:
      pageChangeCopy?.label ===
      "原页面已不存在"
        ? "原页面已不存在，检查相关页面"
        : "查看处理情况",
    tone: "neutral",
    opensDetails: true,
    onClick: openDetails,
  };
}
