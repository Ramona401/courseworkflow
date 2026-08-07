/**
 * CWAIReviewGlobalDiscussion.shared.ts
 *
 * 跨页面、跨问题全局讨论的共享配置和纯数据辅助。
 *
 * 本文件不发起请求、不保存状态，也不改变整改项。
 */

import type { CSSProperties } from "react";

import type {
  CWAIReviewGlobalDiscussion,
  CWAIReviewGlobalRecommendation,
  CWAIReviewGlobalRelationType,
  CWAIReviewItem,
  CWAIReviewSeverity,
} from "@/api/coursewares";

export const CW_GLOBAL_DISCUSSION_COLORS = {
  primary: "#4F7BE8",
  primarySoft: "#EEF2FF",
  success: "#059669",
  successSoft: "#ECFDF5",
  warning: "#D97706",
  warningSoft: "#FFF7ED",
  danger: "#DC2626",
  dangerSoft: "#FEF2F2",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  card: "#FFFFFF",
  bg: "#F8FAFC",
};

const C = CW_GLOBAL_DISCUSSION_COLORS;

export const CW_GLOBAL_DISCUSSION_EMPTY: CWAIReviewGlobalDiscussion = {
  messages: [],
  summary: "",
  relations: [],
  proposals: [],
  selected_item_ids: [],
  latest_message_id: "",
  governance_relations: [],
};

export const CW_GLOBAL_SEVERITY_ORDER: Record<
  CWAIReviewSeverity,
  number
> = {
  critical: 1,
  high: 2,
  medium: 3,
  low: 4,
  info: 5,
};

export const CW_GLOBAL_SEVERITY_LABEL: Record<
  CWAIReviewSeverity,
  string
> = {
  critical: "严重",
  high: "高风险",
  medium: "中风险",
  low: "低风险",
  info: "提示",
};

export const CW_GLOBAL_RELATION_CONFIG: Record<
  CWAIReviewGlobalRelationType,
  {
    label: string;
    color: string;
    background: string;
  }
> = {
  duplicate: {
    label: "重复问题",
    color: "#7C3AED",
    background: "#F5F3FF",
  },
  conflict: {
    label: "存在冲突",
    color: C.danger,
    background: C.dangerSoft,
  },
  merge: {
    label: "可合并处理",
    color: C.primary,
    background: C.primarySoft,
  },
  dependency: {
    label: "存在依赖",
    color: C.warning,
    background: C.warningSoft,
  },
  possibly_resolved: {
    label: "可能连带解决",
    color: C.success,
    background: C.successSoft,
  },
};

export const CW_GLOBAL_RECOMMENDATION_CONFIG: Record<
  CWAIReviewGlobalRecommendation,
  {
    label: string;
    color: string;
    background: string;
  }
> = {
  keep: {
    label: "保留并完善",
    color: C.primary,
    background: C.primarySoft,
  },
  revise: {
    label: "调整指令",
    color: C.warning,
    background: C.warningSoft,
  },
  merge: {
    label: "合并执行",
    color: "#7C3AED",
    background: "#F5F3FF",
  },
  manual_review: {
    label: "需要人工复核",
    color: C.danger,
    background: C.dangerSoft,
  },
  consider_dismiss: {
    label: "可考虑忽略",
    color: C.textSec,
    background: "#F1F5F9",
  },
};

/**
 * 判断整改项是否仍可参加全局讨论。
 *
 * 已交付、已忽略、已应用、已解决、失效或孤立项均不可重新选入。
 */
export function isCWGlobalDiscussionActionableItem(
  item: CWAIReviewItem,
): boolean {
  if (item.courseware_review_id || item.feedback_id) {
    return false;
  }

  return (
    item.status === "detected" ||
    item.status === "discussing" ||
    item.status === "confirmed"
  );
}

export function sortCWGlobalDiscussionItems(
  items: CWAIReviewItem[],
): CWAIReviewItem[] {
  return [...items].sort((left, right) => {
    if (left.page_number_snapshot !== right.page_number_snapshot) {
      return left.page_number_snapshot - right.page_number_snapshot;
    }

    const severityDifference =
      (CW_GLOBAL_SEVERITY_ORDER[left.severity] || 99) -
      (CW_GLOBAL_SEVERITY_ORDER[right.severity] || 99);

    if (severityDifference !== 0) {
      return severityDifference;
    }

    return (left.created_at || "").localeCompare(right.created_at || "");
  });
}

export function resolveCWGlobalItemPageLabel(
  item: CWAIReviewItem,
): string {
  return item.page_number_snapshot > 0
    ? `P${item.page_number_snapshot}`
    : "整课";
}

export function resolveCWGlobalItemTitle(
  item: CWAIReviewItem,
): string {
  return (
    item.title.trim() ||
    item.original_suggestion.trim() ||
    item.description.trim() ||
    "未命名整改项"
  );
}

export const cwGlobalSecondaryButtonStyle: CSSProperties = {
  padding: "5px 9px",
  borderRadius: "6px",
  border: `1px solid ${C.border}`,
  background: "#fff",
  color: C.primary,
  fontSize: "10px",
  fontWeight: 700,
  cursor: "pointer",
};

export const cwGlobalLinkButtonStyle: CSSProperties = {
  padding: "2px 4px",
  border: "none",
  background: "transparent",
  color: C.primary,
  fontSize: "9px",
  fontWeight: 700,
  cursor: "pointer",
};

export const cwGlobalPageButtonStyle: CSSProperties = {
  display: "inline-block",
  padding: "2px 6px",
  borderRadius: "5px",
  border: `1px solid ${C.border}`,
  background: "#fff",
  color: C.textSec,
  fontSize: "9px",
  fontWeight: 700,
  cursor: "pointer",
};
