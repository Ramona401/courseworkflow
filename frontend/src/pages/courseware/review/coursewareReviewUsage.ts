/**
 * coursewareReviewUsage.ts
 *
 * 课件评审体验的最小白名单使用事件客户端。
 *
 * 隐私边界：
 *   - 只发送固定事件、工作区模式、快捷键枚举、数量和页码；
 *   - 不发送搜索词、问题正文、整改要求、教案内容或课件ID；
 *   - 失败静默忽略，绝不阻断评审主流程；
 *   - 深链接恢复等一次性事件可在当前页面生命周期内去重。
 */

import {
  useEffect,
  useRef,
} from "react";

import apiClient from "@/api/client";

import type {
  CWAIReviewItem,
} from "@/api/coursewares";

import type {
  CWReviewIssueFilterState,
  CWReviewWorkspaceScope,
} from "./coursewareReviewWorkspaceState";

export type CWReviewUsageEvent =
  | "filter_applied"
  | "link_copied"
  | "deep_link_restored"
  | "workspace_opened"
  | "keyboard_shortcut"
  | "lesson_plan_opened"
  | "auto_advanced";

export type CWReviewUsageShortcut =
  | "focus_search"
  | "open_next_task"
  | "previous_item"
  | "next_item"
  | "back_to_list"
  | "open_lesson_plan";

export interface CWReviewUsagePayload {
  event: CWReviewUsageEvent;
  mode: CWReviewWorkspaceScope;
  shortcut?: CWReviewUsageShortcut;
  totalCount?: number;
  visibleCount?: number;
  activeFilterCount?: number;
  pageNumber?: number;
}

export interface CWReviewListUsageTrackingOptions {
  mode: CWReviewWorkspaceScope;
  initialDeepLinkOwned: boolean;
  items: CWAIReviewItem[];
  visibleItems: CWAIReviewItem[];
  filters: CWReviewIssueFilterState;
  focusedItemID: string;
}

const sentOnceKeys = new Set<string>();

function clampInteger(
  value: number | undefined,
  minimum: number,
  maximum: number,
): number {
  if (!Number.isFinite(value)) {
    return minimum;
  }

  return Math.min(
    maximum,
    Math.max(
      minimum,
      Math.trunc(value || 0),
    ),
  );
}

function normalizePayload(
  payload: CWReviewUsagePayload,
) {
  return {
    event: payload.event,
    mode: payload.mode,
    ...(payload.shortcut
      ? {
          shortcut: payload.shortcut,
        }
      : {}),
    total_count: clampInteger(
      payload.totalCount,
      0,
      10000,
    ),
    visible_count: clampInteger(
      payload.visibleCount,
      0,
      10000,
    ),
    active_filter_count: clampInteger(
      payload.activeFilterCount,
      0,
      8,
    ),
    page_number: clampInteger(
      payload.pageNumber,
      0,
      10000,
    ),
  };
}

/**
 * 异步记录白名单事件。
 *
 * 事件失败不抛出、不显示提示，也不改变任何业务状态。
 */
export function recordCoursewareReviewUsage(
  payload: CWReviewUsagePayload,
): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    if (!window.localStorage.getItem("token")) {
      return;
    }
  } catch {
    return;
  }

  void apiClient
    .post(
      "/courseware-review-usage",
      normalizePayload(payload),
      {
        timeout: 5000,
      },
    )
    .catch(() => undefined);
}

/**
 * 当前页面生命周期内只记录一次。
 */
export function recordCoursewareReviewUsageOnce(
  key: string,
  payload: CWReviewUsagePayload,
): void {
  const normalizedKey =
    Array.from(key.trim())
      .slice(0, 240)
      .join("");

  if (
    !normalizedKey ||
    sentOnceKeys.has(normalizedKey)
  ) {
    return;
  }

  sentOnceKeys.add(normalizedKey);
  recordCoursewareReviewUsage(payload);
}

export function countCWReviewActiveFilters(
  filters: CWReviewIssueFilterState,
): number {
  let count = 0;

  if (filters.query.trim()) count += 1;
  if (filters.severity !== "all") count += 1;
  if (filters.status !== "all") count += 1;
  if (filters.page !== "all") count += 1;
  if (filters.source !== "all") count += 1;
  if (filters.showCompleted) count += 1;

  return count;
}

export function resolveCWReviewFilterPageNumber(
  filters: CWReviewIssueFilterState,
): number {
  return /^\d{1,6}$/.test(filters.page)
    ? Number(filters.page)
    : 0;
}

export function resolveCWReviewUsageModeFromElement(
  element: HTMLElement | null,
): CWReviewWorkspaceScope | null {
  const raw =
    element
      ?.closest<HTMLElement>(
        "[data-cw-review-scope]",
      )
      ?.dataset.cwReviewScope || "";

  return (
    raw === "formal" ||
    raw === "self" ||
    raw === "remediation"
  )
    ? raw
    : null;
}

/**
 * 统一记录列表层的深链接恢复与首次进入聚焦工作区。
 */
export function useCWReviewListUsageTracking({
  mode,
  initialDeepLinkOwned,
  items,
  visibleItems,
  filters,
  focusedItemID,
}: CWReviewListUsageTrackingOptions): void {
  const previousFocusedItemIDRef =
    useRef("");

  const activeFilterCount =
    countCWReviewActiveFilters(
      filters,
    );

  const filterPageNumber =
    resolveCWReviewFilterPageNumber(
      filters,
    );

  useEffect(() => {
    if (!initialDeepLinkOwned) {
      return;
    }

    recordCoursewareReviewUsageOnce(
      [
        "deep-link",
        mode,
        window.location.pathname,
        window.location.search,
      ].join(":"),
      {
        event: "deep_link_restored",
        mode,
        totalCount: items.length,
        visibleCount:
          visibleItems.length,
        activeFilterCount,
        pageNumber:
          filterPageNumber,
      },
    );
  }, [
    activeFilterCount,
    filterPageNumber,
    initialDeepLinkOwned,
    items.length,
    mode,
    visibleItems.length,
  ]);

  useEffect(() => {
    const previous =
      previousFocusedItemIDRef.current;

    if (
      !previous &&
      focusedItemID
    ) {
      const target =
        items.find(
          (item) =>
            item.id === focusedItemID,
        );

      recordCoursewareReviewUsage({
        event: "workspace_opened",
        mode,
        totalCount: items.length,
        visibleCount:
          visibleItems.length,
        activeFilterCount,
        pageNumber:
          target?.page_number_snapshot ||
          filterPageNumber,
      });
    }

    previousFocusedItemIDRef.current =
      focusedItemID;
  }, [
    activeFilterCount,
    filterPageNumber,
    focusedItemID,
    items,
    mode,
    visibleItems.length,
  ]);
}
