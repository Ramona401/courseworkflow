/**
 * coursewareReviewWorkspaceState.ts
 *
 * 课件评审工作区的搜索筛选与可恢复URL状态。
 *
 * URL只保存问题ID和枚举筛选值，不写入问题正文、整改要求、教案内容或内部状态快照。
 * 同一页面可能同时挂载自审与正式整改面板，因此用cw_view声明当前URL属于哪个工作区；
 * 未拥有URL的面板只维护本地状态，不会覆盖另一个面板的深链接。
 */

import type {
  CWAIReviewItem,
  CWAIReviewSeverity,
} from "@/api/coursewares";

export type CWReviewWorkspaceScope =
  | "formal"
  | "self"
  | "remediation";

export type CWReviewSeverityFilter =
  | "all"
  | CWAIReviewSeverity;

export type CWReviewStatusFilter =
  | "all"
  | "pending"
  | "applying"
  | "waiting_confirm"
  | "completed"
  | "stale";

export type CWReviewSourceFilter =
  | "all"
  | "formal"
  | "self";

export type CWReviewPageFilter =
  | "all"
  | "global"
  | string;

export interface CWReviewIssueFilterState {
  query: string;
  severity: CWReviewSeverityFilter;
  status: CWReviewStatusFilter;
  page: CWReviewPageFilter;
  source: CWReviewSourceFilter;
  showCompleted: boolean;
}

export interface CWReviewWorkspaceURLState {
  owned: boolean;
  filters: CWReviewIssueFilterState;
  focusedItemID: string;
}

export const CW_REVIEW_WORKSPACE_URL_EVENT =
  "tedna:cw-review-url-state";

const PARAM_VIEW = "cw_view";
const PARAM_ISSUE = "cw_issue";
const PARAM_QUERY = "cw_q";
const PARAM_SEVERITY = "cw_severity";
const PARAM_STATUS = "cw_status";
const PARAM_PAGE = "cw_page";
const PARAM_SOURCE = "cw_source";
const PARAM_COMPLETED = "cw_completed";

const VALID_SCOPES =
  new Set<CWReviewWorkspaceScope>([
    "formal",
    "self",
    "remediation",
  ]);

const VALID_SEVERITIES =
  new Set<CWReviewSeverityFilter>([
    "all",
    "critical",
    "high",
    "medium",
    "low",
    "info",
  ]);

const VALID_STATUSES =
  new Set<CWReviewStatusFilter>([
    "all",
    "pending",
    "applying",
    "waiting_confirm",
    "completed",
    "stale",
  ]);

const VALID_SOURCES =
  new Set<CWReviewSourceFilter>([
    "all",
    "formal",
    "self",
  ]);

export function createDefaultCWReviewFilters(): CWReviewIssueFilterState {
  return {
    query: "",
    severity: "all",
    status: "all",
    page: "all",
    source: "all",
    showCompleted: false,
  };
}

function sanitizeQuery(value: string): string {
  return Array.from(value.trim())
    .slice(0, 120)
    .join("");
}

function sanitizeFocusedItemID(value: string): string {
  return Array.from(value.trim())
    .slice(0, 160)
    .join("");
}

function sanitizePageFilter(value: string): CWReviewPageFilter {
  const normalized = value.trim();

  if (
    normalized === "all" ||
    normalized === "global"
  ) {
    return normalized;
  }

  return /^\d{1,6}$/.test(normalized)
    ? normalized
    : "all";
}

export function readCWReviewWorkspaceURLState(
  scope: CWReviewWorkspaceScope,
): CWReviewWorkspaceURLState {
  const defaults =
    createDefaultCWReviewFilters();

  if (typeof window === "undefined") {
    return {
      owned: false,
      filters: defaults,
      focusedItemID: "",
    };
  }

  const params =
    new URL(window.location.href)
      .searchParams;

  const rawScope =
    params.get(PARAM_VIEW) || "";

  if (
    !VALID_SCOPES.has(
      rawScope as CWReviewWorkspaceScope,
    ) ||
    rawScope !== scope
  ) {
    return {
      owned: false,
      filters: defaults,
      focusedItemID: "",
    };
  }

  const rawSeverity =
    params.get(PARAM_SEVERITY) ||
    "all";

  const rawStatus =
    params.get(PARAM_STATUS) ||
    "all";

  const rawSource =
    params.get(PARAM_SOURCE) ||
    "all";

  const status =
    VALID_STATUSES.has(
      rawStatus as CWReviewStatusFilter,
    )
      ? rawStatus as CWReviewStatusFilter
      : "all";

  return {
    owned: true,
    focusedItemID:
      sanitizeFocusedItemID(
        params.get(PARAM_ISSUE) || "",
      ),
    filters: {
      query: sanitizeQuery(
        params.get(PARAM_QUERY) || "",
      ),
      severity:
        VALID_SEVERITIES.has(
          rawSeverity as CWReviewSeverityFilter,
        )
          ? rawSeverity as CWReviewSeverityFilter
          : "all",
      status,
      page: sanitizePageFilter(
        params.get(PARAM_PAGE) || "all",
      ),
      source:
        VALID_SOURCES.has(
          rawSource as CWReviewSourceFilter,
        )
          ? rawSource as CWReviewSourceFilter
          : "all",
      showCompleted:
        status === "completed" ||
        params.get(PARAM_COMPLETED) === "1",
    },
  };
}

function setOrDelete(
  params: URLSearchParams,
  key: string,
  value: string,
  defaultValue: string,
): void {
  if (!value || value === defaultValue) {
    params.delete(key);
    return;
  }

  params.set(key, value);
}

export function buildCWReviewWorkspaceURL(
  scope: CWReviewWorkspaceScope,
  filters: CWReviewIssueFilterState,
  focusedItemID: string,
): string {
  if (typeof window === "undefined") {
    return "";
  }

  const url =
    new URL(window.location.href);

  url.searchParams.set(
    PARAM_VIEW,
    scope,
  );

  setOrDelete(
    url.searchParams,
    PARAM_ISSUE,
    sanitizeFocusedItemID(
      focusedItemID,
    ),
    "",
  );

  setOrDelete(
    url.searchParams,
    PARAM_QUERY,
    sanitizeQuery(filters.query),
    "",
  );

  setOrDelete(
    url.searchParams,
    PARAM_SEVERITY,
    filters.severity,
    "all",
  );

  setOrDelete(
    url.searchParams,
    PARAM_STATUS,
    filters.status,
    "all",
  );

  setOrDelete(
    url.searchParams,
    PARAM_PAGE,
    sanitizePageFilter(
      filters.page,
    ),
    "all",
  );

  setOrDelete(
    url.searchParams,
    PARAM_SOURCE,
    filters.source,
    "all",
  );

  if (
    filters.showCompleted &&
    filters.status !== "completed"
  ) {
    url.searchParams.set(
      PARAM_COMPLETED,
      "1",
    );
  } else {
    url.searchParams.delete(
      PARAM_COMPLETED,
    );
  }

  return url.toString();
}

export function replaceCWReviewWorkspaceURL(
  scope: CWReviewWorkspaceScope,
  filters: CWReviewIssueFilterState,
  focusedItemID: string,
): void {
  if (typeof window === "undefined") {
    return;
  }

  const fullURL =
    buildCWReviewWorkspaceURL(
      scope,
      filters,
      focusedItemID,
    );

  const parsed =
    new URL(fullURL);

  window.history.replaceState(
    window.history.state,
    "",
    [
      parsed.pathname,
      parsed.search,
      parsed.hash,
    ].join(""),
  );

  window.dispatchEvent(
    new CustomEvent(
      CW_REVIEW_WORKSPACE_URL_EVENT,
      {
        detail: {
          scope,
        },
      },
    ),
  );
}

function isCompletedItem(
  item: CWAIReviewItem,
): boolean {
  return (
    item.status === "resolved" ||
    item.status === "dismissed"
  );
}

function matchesStatusFilter(
  item: CWAIReviewItem,
  status: CWReviewStatusFilter,
): boolean {
  switch (status) {
    case "pending":
      return (
        item.status === "detected" ||
        item.status === "discussing" ||
        item.status === "confirmed" ||
        item.status === "orphaned"
      );

    case "applying":
      return item.status === "applying";

    case "waiting_confirm":
      return item.status === "applied";

    case "completed":
      return isCompletedItem(item);

    case "stale":
      return item.status === "stale";

    case "all":
      return true;
  }
}

function buildSearchText(
  item: CWAIReviewItem,
): string {
  return [
    item.title,
    item.description,
    item.confirmed_instruction,
    item.original_suggestion,
    item.page_title_snapshot,
    item.dimension,
    item.severity,
    item.status,
    item.source_type,
    item.page_number_snapshot > 0
      ? `p${item.page_number_snapshot} 第${item.page_number_snapshot}页`
      : "整课",
  ]
    .join("\n")
    .toLocaleLowerCase("zh-CN");
}

export function filterCWReviewItems(
  items: CWAIReviewItem[],
  filters: CWReviewIssueFilterState,
): CWAIReviewItem[] {
  const query =
    sanitizeQuery(filters.query)
      .toLocaleLowerCase("zh-CN");

  return items.filter((item) => {
    if (
      !filters.showCompleted &&
      filters.status !== "completed" &&
      isCompletedItem(item)
    ) {
      return false;
    }

    if (
      filters.severity !== "all" &&
      item.severity !== filters.severity
    ) {
      return false;
    }

    if (
      !matchesStatusFilter(
        item,
        filters.status,
      )
    ) {
      return false;
    }

    if (
      filters.source !== "all" &&
      item.source_type !== filters.source
    ) {
      return false;
    }

    if (
      filters.page === "global" &&
      item.page_number_snapshot > 0
    ) {
      return false;
    }

    if (
      filters.page !== "all" &&
      filters.page !== "global" &&
      item.page_number_snapshot !==
        Number(filters.page)
    ) {
      return false;
    }

    return (
      !query ||
      buildSearchText(item)
        .includes(query)
    );
  });
}

export function hasActiveCWReviewFilters(
  filters: CWReviewIssueFilterState,
): boolean {
  return (
    !!filters.query.trim() ||
    filters.severity !== "all" ||
    filters.status !== "all" ||
    filters.page !== "all" ||
    filters.source !== "all" ||
    filters.showCompleted
  );
}


export function isNearestVisibleCWReviewElement(
  element: HTMLElement | null,
  selector: string,
): boolean {
  if (!element || typeof document === "undefined") {
    return false;
  }

  const viewportCenter =
    window.innerHeight / 2;

  const candidates =
    Array.from(
      document.querySelectorAll<HTMLElement>(
        selector,
      ),
    )
      .filter((candidate) => {
        const rect =
          candidate.getBoundingClientRect();

        return (
          rect.width > 0 &&
          rect.height > 0 &&
          rect.bottom >= 0 &&
          rect.top <= window.innerHeight
        );
      })
      .sort((left, right) => {
        const leftRect =
          left.getBoundingClientRect();

        const rightRect =
          right.getBoundingClientRect();

        const leftDistance =
          Math.abs(
            leftRect.top +
              leftRect.height / 2 -
              viewportCenter,
          );

        const rightDistance =
          Math.abs(
            rightRect.top +
              rightRect.height / 2 -
              viewportCenter,
          );

        return leftDistance - rightDistance;
      });

  return candidates[0] === element;
}

export function isCWReviewTextEntryTarget(
  target: EventTarget | null,
): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }

  return (
    target.isContentEditable ||
    target.tagName === "INPUT" ||
    target.tagName === "TEXTAREA" ||
    target.tagName === "SELECT"
  );
}
