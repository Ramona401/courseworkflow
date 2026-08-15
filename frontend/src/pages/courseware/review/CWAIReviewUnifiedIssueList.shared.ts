/**
 * CWAIReviewUnifiedIssueList.shared.ts
 *
 * 统一问题工作台的纯计算。
 *
 * 不持有React状态、不调用API：
 *   - 问题按稳定页面分组；
 *   - 汇总教师可见的处理进度；
 *   - 计算“下一条任务”的稳定排序。
 */

import type {
  CWAIReviewItem,
  CWAIReviewSeverity,
} from "@/api/coursewares";

import type {
  CWAIReviewUnifiedIssueGroupData,
} from "./CWAIReviewUnifiedIssueGroup";

const SEVERITY_ORDER: Record<CWAIReviewSeverity, number> = {
  critical: 1,
  high: 2,
  medium: 3,
  low: 4,
  info: 5,
};

export interface CWAIReviewUnifiedIssueOverview {
  total: number;
  pending: number;
  applying: number;
  waitingConfirm: number;
  completed: number;
  stale: number;
}

export function isCWAIReviewUnifiedIssueCompleted(
  item: CWAIReviewItem,
): boolean {
  return (
    item.status === "resolved" ||
    item.status === "dismissed"
  );
}

export function buildCWAIReviewUnifiedIssueGroups(
  items: CWAIReviewItem[],
): CWAIReviewUnifiedIssueGroupData[] {
  const groupMap =
    new Map<string, CWAIReviewUnifiedIssueGroupData>();

  for (const item of items) {
    const pageNumber = item.page_number_snapshot;
    const stablePageID = item.page_id?.trim() || "";

    const key =
      pageNumber <= 0
        ? "global"
        : stablePageID ||
          [
            "snapshot",
            pageNumber,
            item.page_title_snapshot,
          ].join(":");

    const current = groupMap.get(key) || {
      key,
      pageNumber: pageNumber <= 0 ? 0 : pageNumber,
      pageTitle:
        pageNumber <= 0
          ? "整课综合问题"
          : item.page_title_snapshot,
      items: [],
    };

    current.items.push(item);

    if (
      pageNumber > 0 &&
      !current.pageTitle &&
      item.page_title_snapshot
    ) {
      current.pageTitle = item.page_title_snapshot;
    }

    groupMap.set(key, current);
  }

  const groups = Array.from(groupMap.values());

  for (const group of groups) {
    group.items.sort((left, right) => {
      const severityDifference =
        (SEVERITY_ORDER[left.severity] || 99) -
        (SEVERITY_ORDER[right.severity] || 99);

      if (severityDifference !== 0) {
        return severityDifference;
      }

      return (left.created_at || "").localeCompare(
        right.created_at || "",
      );
    });
  }

  return groups.sort((left, right) => {
    if (left.pageNumber <= 0 && right.pageNumber > 0) {
      return -1;
    }

    if (right.pageNumber <= 0 && left.pageNumber > 0) {
      return 1;
    }

    return left.pageNumber - right.pageNumber;
  });
}

export function buildCWAIReviewUnifiedIssueOverview(
  items: CWAIReviewItem[],
): CWAIReviewUnifiedIssueOverview {
  const overview: CWAIReviewUnifiedIssueOverview = {
    total: items.length,
    pending: 0,
    applying: 0,
    waitingConfirm: 0,
    completed: 0,
    stale: 0,
  };

  for (const item of items) {
    if (isCWAIReviewUnifiedIssueCompleted(item)) {
      overview.completed += 1;
      continue;
    }

    if (item.status === "applying") {
      overview.applying += 1;
    } else if (item.status === "applied") {
      overview.waitingConfirm += 1;
    } else {
      overview.pending += 1;
    }

    if (item.status === "stale") {
      overview.stale += 1;
    }
  }

  return overview;
}

function cwAIReviewNextTaskPriority(
  item: CWAIReviewItem,
): number {
  if (item.severity === "critical") {
    return 0;
  }

  if (item.severity === "high") {
    return 10;
  }

  if (item.status === "stale") {
    return 20;
  }

  if (item.status === "applying") {
    return 30;
  }

  return 40 + (SEVERITY_ORDER[item.severity] || 99);
}

export function sortCWAIReviewUnifiedNextTasks(
  left: CWAIReviewItem,
  right: CWAIReviewItem,
): number {
  const priorityDifference =
    cwAIReviewNextTaskPriority(left) -
    cwAIReviewNextTaskPriority(right);

  if (priorityDifference !== 0) {
    return priorityDifference;
  }

  const pageDifference =
    left.page_number_snapshot -
    right.page_number_snapshot;

  if (pageDifference !== 0) {
    return pageDifference;
  }

  return (left.created_at || "").localeCompare(
    right.created_at || "",
  );
}
