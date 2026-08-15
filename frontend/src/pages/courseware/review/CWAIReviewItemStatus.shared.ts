/**
 * CWAIReviewItemStatus.shared.ts
 *
 * 教师改进卡的统一状态表达。
 *
 * R-01.1边界：
 *   - 内部仍可继续使用stale/orphaned状态；
 *   - 教师界面不得直接出现这些技术状态名；
 *   - stale统一表达为“页面内容已变化”；
 *   - orphaned统一表达为“原页面已不存在”；
 *   - 三种业务场景共享同一页面变化状态名称。
 */

import type {
  CWAIReviewItemStatus,
} from "@/api/coursewares";

export type CWAIReviewItemExperience =
  | "review"
  | "self"
  | "remediation";

export interface CWAIReviewPageChangeTeacherCopy {
  label: string;
  guidance: string;
}

const STATUS_STYLE:
  Record<
    CWAIReviewItemStatus,
    {
      color: string;
      background: string;
    }
  > = {
    detected: {
      color: "#4F7BE8",
      background: "#EEF2FF",
    },
    discussing: {
      color: "#7C3AED",
      background: "#F5F3FF",
    },
    confirmed: {
      color: "#059669",
      background: "#ECFDF5",
    },
    applying: {
      color: "#D97706",
      background: "#FFFBEB",
    },
    applied: {
      color: "#0284C7",
      background: "#F0F9FF",
    },
    resolved: {
      color: "#059669",
      background: "#ECFDF5",
    },
    dismissed: {
      color: "#64748B",
      background: "#F8FAFC",
    },
    stale: {
      color: "#DC2626",
      background: "#FEF2F2",
    },
    orphaned: {
      color: "#DC2626",
      background: "#FEF2F2",
    },
  };

const STATUS_LABEL:
  Record<
    CWAIReviewItemExperience,
    Record<
      CWAIReviewItemStatus,
      string
    >
  > = {
    review: {
      detected: "还需明确整改要求",
      discussing: "正在完善整改要求",
      confirmed: "整改要求已确认",
      applying: "作者正在修改",
      applied: "作者已完成修改",
      resolved: "已确认解决",
      dismissed: "本次不退回",
      stale: "页面内容已变化",
      orphaned: "原页面已不存在",
    },

    self: {
      detected: "还需形成修改方案",
      discussing: "正在完善修改方案",
      confirmed: "修改方案已准备好",
      applying: "正在修改页面",
      applied: "修改完成，等待确认",
      resolved: "问题已解决",
      dismissed: "这次暂不调整",
      stale: "页面内容已变化",
      orphaned: "原页面已不存在",
    },

    remediation: {
      detected: "请查看审核要求",
      discussing: "请查看审核要求",
      confirmed: "可以开始修改",
      applying: "正在完成整改",
      applied: "修改完成，等待复审",
      resolved: "已完成整改",
      dismissed: "本次无需修改",
      stale: "页面内容已变化",
      orphaned: "原页面已不存在",
    },
  };

const PAGE_CHANGE_TEACHER_COPY:
  Record<
    "stale" | "orphaned",
    CWAIReviewPageChangeTeacherCopy
  > = {
    stale: {
      label: "页面内容已变化",
      guidance:
        "页面内容已经变化，需要人工重新检查当前页面后再继续处理。",
    },

    orphaned: {
      label: "原页面已不存在",
      guidance:
        "原页面已经不存在，需要人工重新检查相关页面或整课内容后再继续处理。",
    },
  };

export function resolveCWAIReviewPageChangeTeacherCopy(
  status: CWAIReviewItemStatus,
): CWAIReviewPageChangeTeacherCopy | null {
  if (
    status !== "stale" &&
    status !== "orphaned"
  ) {
    return null;
  }

  return PAGE_CHANGE_TEACHER_COPY[status];
}

export function resolveCWAIReviewItemStatus(
  experience: CWAIReviewItemExperience,
  status: CWAIReviewItemStatus,
): {
  label: string;
  color: string;
  background: string;
} {
  const pageChangeCopy =
    resolveCWAIReviewPageChangeTeacherCopy(status);

  return {
    label:
      pageChangeCopy?.label ||
      STATUS_LABEL[experience][status],
    ...STATUS_STYLE[status],
  };
}
