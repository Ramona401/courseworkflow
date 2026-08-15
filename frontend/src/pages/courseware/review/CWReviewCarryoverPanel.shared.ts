/**
 * CWReviewCarryoverPanel.shared.ts
 *
 * 上轮修改复审列表的纯计算与展示样式。
 *
 * 不持有React状态、不修改复审草稿：
 *   - 按稳定页面分组；
 *   - 判断本轮暂选已解决；
 *   - 汇总复审数字；
 *   - 提供列表容器样式。
 */

import type {
  CSSProperties,
} from "react";

import type {
  CWReviewCarryoverItem,
} from "@/api/coursewares";

import {
  CW_AI_REVIEW_ITEM_COLORS as C,
} from "./CWAIReviewItemPresentation.shared";

export interface CWReviewCarryoverGroup {
  key: string;
  pageNumber: number;
  pageTitle: string;
  items: CWReviewCarryoverItem[];
}

export interface CWReviewCarryoverMetrics {
  appliedCount: number;
  resolvedCount: number;
  waitingReviewCount: number;
  unfinishedCount: number;
  changedPageCount: number;
}

export type CWReviewCarryoverViewMode =
  | "pending"
  | "confirmed";

const SEVERITY_ORDER = {
  critical: 1,
  high: 2,
  medium: 3,
  low: 4,
  info: 5,
} as const;

export function isCWReviewCarryoverMarkedResolved(
  item: CWReviewCarryoverItem,
  resolvedIDSet:
    ReadonlySet<string>,
): boolean {
  return (
    item.status === "applied" &&
    resolvedIDSet.has(item.id)
  );
}

export function buildCWReviewCarryoverGroups(
  items: CWReviewCarryoverItem[],
): CWReviewCarryoverGroup[] {
  const groupMap =
    new Map<
      string,
      CWReviewCarryoverGroup
    >();

  for (const item of items) {
    const isGlobalIssue =
      item.page_number_snapshot <= 0;

    const stablePageID =
      item.page_id?.trim() || "";

    const key =
      isGlobalIssue
        ? "global"
        : stablePageID ||
          `snapshot:${item.page_number_snapshot}`;

    const current =
      groupMap.get(key) || {
        key,
        pageNumber:
          isGlobalIssue
            ? 0
            : item.page_number_snapshot,
        pageTitle:
          isGlobalIssue
            ? "整课问题"
            : item.page_title_snapshot ||
              `第${item.page_number_snapshot}页`,
        items: [],
      };

    current.items.push(item);
    groupMap.set(key, current);
  }

  const groups =
    Array.from(
      groupMap.values(),
    );

  for (const group of groups) {
    group.items.sort(
      (left, right) => {
        const leftOrder =
          SEVERITY_ORDER[
            left.severity
          ];

        const rightOrder =
          SEVERITY_ORDER[
            right.severity
          ];

        if (
          leftOrder !== rightOrder
        ) {
          return (
            leftOrder - rightOrder
          );
        }

        return left.id.localeCompare(
          right.id,
        );
      },
    );
  }

  return groups.sort(
    (left, right) => {
      if (left.pageNumber === 0) {
        return -1;
      }

      if (right.pageNumber === 0) {
        return 1;
      }

      return (
        left.pageNumber -
        right.pageNumber
      );
    },
  );
}

export function buildCWReviewCarryoverMetrics(
  items: CWReviewCarryoverItem[],
  resolvedIDSet:
    ReadonlySet<string>,
): CWReviewCarryoverMetrics {
  let appliedCount = 0;
  let resolvedCount = 0;
  let changedPageCount = 0;

  for (const item of items) {
    if (item.status === "applied") {
      appliedCount += 1;
    }

    if (
      isCWReviewCarryoverMarkedResolved(
        item,
        resolvedIDSet,
      )
    ) {
      resolvedCount += 1;
    }

    if (
      item.status === "stale" ||
      item.status === "orphaned"
    ) {
      changedPageCount += 1;
    }
  }

  return {
    appliedCount,
    resolvedCount,
    waitingReviewCount:
      appliedCount - resolvedCount,
    unfinishedCount:
      items.length - appliedCount,
    changedPageCount,
  };
}

export const cwReviewCarryoverEmptyStateStyle:
  CSSProperties = {
    padding: "28px 16px",
    border:
      `1px dashed ${C.border}`,
    borderRadius: "10px",
    color: C.textSec,
    fontSize: "14px",
    lineHeight: 1.7,
    textAlign: "center",
  };

export const cwReviewCarryoverOverviewPanelStyle:
  CSSProperties = {
    padding: "14px",
    border:
      `1px solid ${C.primary}30`,
    borderRadius: "10px",
    background: "#EEF2FF",
  };

export const cwReviewCarryoverOverviewTitleStyle:
  CSSProperties = {
    color: C.text,
    fontSize: "18px",
    fontWeight: 700,
    lineHeight: 1.4,
  };

export const cwReviewCarryoverOverviewDescriptionStyle:
  CSSProperties = {
    marginTop: "5px",
    maxWidth: "75ch",
    color: C.textSec,
    fontSize: "14px",
    lineHeight: 1.6,
  };

export const cwReviewCarryoverMetricGridStyle:
  CSSProperties = {
    display: "grid",
    gridTemplateColumns:
      "repeat(auto-fit, minmax(92px, 1fr))",
    gap: "8px",
    marginTop: "12px",
  };

export const cwReviewCarryoverBlockingNoticeStyle:
  CSSProperties = {
    marginTop: "12px",
    padding: "10px 12px",
    border:
      "1px solid #FECACA",
    borderRadius: "8px",
    background: "#FEF2F2",
    color: C.danger,
    fontSize: "14px",
    fontWeight: 600,
    lineHeight: 1.6,
  };

export const cwReviewCarryoverReviewRuleStyle:
  CSSProperties = {
    marginTop: "10px",
    color: C.textSec,
    fontSize: "12px",
    lineHeight: 1.6,
  };

export const cwReviewCarryoverFilterRowStyle:
  CSSProperties = {
    display: "flex",
    alignItems: "center",
    gap: "8px",
    flexWrap: "wrap",
    marginTop: "12px",
  };

export const cwReviewCarryoverGroupContainerStyle:
  CSSProperties = {
    marginTop: "12px",
    overflow: "hidden",
    border:
      `1px solid ${C.border}`,
    borderRadius: "10px",
    background: C.card,
  };

export const cwReviewCarryoverGroupToggleStyle:
  CSSProperties = {
    width: "100%",
    minWidth: 0,
    display: "flex",
    alignItems: "center",
    gap: "10px",
    padding: "10px 12px",
    border: "none",
    background: "#FFFFFF",
    textAlign: "left",
    cursor: "pointer",
  };

export const cwReviewCarryoverArrowStyle:
  CSSProperties = {
    flexShrink: 0,
    fontSize: "16px",
    transition:
      "transform 160ms ease",
  };

export const cwReviewCarryoverGroupTitleStyle:
  CSSProperties = {
    display: "block",
    overflow: "hidden",
    color: C.text,
    fontSize: "15px",
    fontWeight: 700,
    lineHeight: 1.5,
    textOverflow: "ellipsis",
    whiteSpace: "nowrap",
  };

export const cwReviewCarryoverGroupSummaryStyle:
  CSSProperties = {
    display: "block",
    marginTop: "3px",
    color: C.textMuted,
    fontSize: "12px",
    lineHeight: 1.5,
  };

export const cwReviewCarryoverGroupBodyStyle:
  CSSProperties = {
    padding: "2px 12px 12px",
    borderTop:
      `1px solid ${C.border}`,
  };

export function cwReviewCarryoverFilterButtonStyle(
  active: boolean,
  tone:
    | "primary"
    | "success",
): CSSProperties {
  const config = {
    primary: {
      color: C.primary,
      background: "#EEF2FF",
    },
    success: {
      color: C.success,
      background: "#ECFDF5",
    },
  }[tone];

  return {
    minHeight: "36px",
    padding: "8px 12px",
    borderRadius: "8px",
    border:
      `1px solid ${
        active
          ? config.color
          : C.border
      }`,
    background:
      active
        ? config.background
        : "#FFFFFF",
    color:
      active
        ? config.color
        : C.textSec,
    fontSize: "13px",
    fontWeight: 700,
    cursor: "pointer",
  };
}
