/**
 * CWReviewCarryoverPanel.tsx
 *
 * 审核员复查上一轮整改问题的列表容器。
 *
 * 本组件只负责：
 *   - 复审进度摘要；
 *   - 待处理/本轮已确认视图切换；
 *   - 分组展开状态；
 *   - 将每条问题交给CWReviewCarryoverItemCard。
 *
 * 分组、复审数字和展示样式位于CWReviewCarryoverPanel.shared.ts。
 *
 * 不提供“确认全部”“本页确认”“本组确认”等批量通过入口。
 * 真正的resolved仍只在提交正式审核决定时由后端事务写入。
 */

import {
  useMemo,
  useState,
} from "react";

import type {
  CWReviewCarryoverItem,
} from "@/api/coursewares";

import {
  CW_AI_REVIEW_ITEM_COLORS as C,
} from "./CWAIReviewItemPresentation.shared";

import CWReviewCarryoverItemCard, {
  type CWReviewCarryoverPageReference,
} from "./CWReviewCarryoverItemCard";

import {
  buildCWReviewCarryoverGroups,
  buildCWReviewCarryoverMetrics,
  cwReviewCarryoverArrowStyle,
  cwReviewCarryoverBlockingNoticeStyle,
  cwReviewCarryoverEmptyStateStyle,
  cwReviewCarryoverFilterButtonStyle,
  cwReviewCarryoverFilterRowStyle,
  cwReviewCarryoverGroupBodyStyle,
  cwReviewCarryoverGroupContainerStyle,
  cwReviewCarryoverGroupSummaryStyle,
  cwReviewCarryoverGroupTitleStyle,
  cwReviewCarryoverGroupToggleStyle,
  cwReviewCarryoverMetricGridStyle,
  cwReviewCarryoverOverviewDescriptionStyle,
  cwReviewCarryoverOverviewPanelStyle,
  cwReviewCarryoverOverviewTitleStyle,
  cwReviewCarryoverReviewRuleStyle,
  isCWReviewCarryoverMarkedResolved,
  type CWReviewCarryoverViewMode,
} from "./CWReviewCarryoverPanel.shared";

export interface CWReviewCarryoverPanelProps {
  items: CWReviewCarryoverItem[];
  pages:
    CWReviewCarryoverPageReference[];
  pendingReviewRound: number;
  resolvedItemIds: string[];

  onResolvedChange: (
    itemId: string,
    resolved: boolean,
  ) => void;

  onSelectPage: (
    pageNumber: number,
  ) => void;
}

export default function CWReviewCarryoverPanel({
  items,
  pages,
  pendingReviewRound,
  resolvedItemIds,
  onResolvedChange,
  onSelectPage,
}: CWReviewCarryoverPanelProps) {
  const resolvedIDSet =
    useMemo(
      () =>
        new Set(
          resolvedItemIds,
        ),
      [resolvedItemIds],
    );

  const groups =
    useMemo(
      () =>
        buildCWReviewCarryoverGroups(
          items,
        ),
      [items],
    );

  const metrics =
    useMemo(
      () =>
        buildCWReviewCarryoverMetrics(
          items,
          resolvedIDSet,
        ),
      [
        items,
        resolvedIDSet,
      ],
    );

  const [
    viewMode,
    setViewMode,
  ] =
    useState<
      CWReviewCarryoverViewMode
    >("pending");

  /**
   * null表示使用当前视图默认展开第一组。
   * 教师手动操作后才保存显式展开集合。
   */
  const [
    expandedGroupKeys,
    setExpandedGroupKeys,
  ] =
    useState<
      Set<string> | null
    >(null);

  const visibleGroups =
    useMemo(
      () =>
        groups
          .map(
            (group) => ({
              group,
              visibleItems:
                group.items.filter(
                  (item) => {
                    const marked =
                      isCWReviewCarryoverMarkedResolved(
                        item,
                        resolvedIDSet,
                      );

                    return viewMode ===
                      "confirmed"
                      ? marked
                      : !marked;
                  },
                ),
            }),
          )
          .filter(
            ({ visibleItems }) =>
              visibleItems.length >
              0,
          ),
      [
        groups,
        resolvedIDSet,
        viewMode,
      ],
    );

  const firstVisibleGroupKey =
    visibleGroups[0]
      ?.group.key || "";

  const changeViewMode =
    (
      nextMode:
        CWReviewCarryoverViewMode,
    ) => {
      setViewMode(nextMode);
      setExpandedGroupKeys(null);
    };

  const handleResolvedChange =
    (
      itemID: string,
      resolved: boolean,
    ) => {
      onResolvedChange(
        itemID,
        resolved,
      );

      // 当前项可能在两个视图间移动，恢复默认展开规则。
      setExpandedGroupKeys(null);
    };

  const toggleGroup =
    (
      groupKey: string,
    ) => {
      setExpandedGroupKeys(
        (previous) => {
          const next =
            previous === null
              ? new Set(
                  firstVisibleGroupKey
                    ? [
                        firstVisibleGroupKey,
                      ]
                    : [],
                )
              : new Set(previous);

          if (next.has(groupKey)) {
            next.delete(groupKey);
          } else {
            next.add(groupKey);
          }

          return next;
        },
      );
    };

  if (items.length === 0) {
    return (
      <div
        style={
          cwReviewCarryoverEmptyStateStyle
        }
      >
        <div
          style={{
            marginBottom: "8px",
            fontSize: "28px",
          }}
        >
          ✅
        </div>
        本轮没有需要复查的历史修改问题。
      </div>
    );
  }

  return (
    <section>
      <div
        style={
          cwReviewCarryoverOverviewPanelStyle
        }
      >
        <div
          style={
            cwReviewCarryoverOverviewTitleStyle
          }
        >
          第{" "}
          {pendingReviewRound}{" "}
          轮复审 · 上轮修改
        </div>

        <div
          style={
            cwReviewCarryoverOverviewDescriptionStyle
          }
        >
          请逐条打开教师改进卡，对照当前修改要求和检查项复查。
          对达到可复查状态的问题，请明确选择“确认已解决”或“继续要求修改”。
          页面内容已变化或原页面已不存在时，需要人工重新检查，不能直接确认解决。
        </div>

        <div
          style={
            cwReviewCarryoverMetricGridStyle
          }
        >
          <OverviewMetric
            label="全部问题"
            value={items.length}
            tone="primary"
          />

          <OverviewMetric
            label="待复查"
            value={
              metrics
                .waitingReviewCount
            }
            tone="primary"
          />

          <OverviewMetric
            label="未完成"
            value={
              metrics
                .unfinishedCount
            }
            tone="warning"
          />

          <OverviewMetric
            label="本轮已确认"
            value={
              metrics
                .resolvedCount
            }
            tone="success"
          />

          <OverviewMetric
            label="需重新检查"
            value={
              metrics
                .changedPageCount
            }
            tone="danger"
          />
        </div>

        {metrics.unfinishedCount >
          0 && (
          <div
            style={
              cwReviewCarryoverBlockingNoticeStyle
            }
          >
            暂时不能审核通过：还有{" "}
            {metrics.unfinishedCount}{" "}
            条上轮问题尚未达到可复查状态。
          </div>
        )}

        <div
          style={
            cwReviewCarryoverReviewRuleStyle
          }
        >
          复审判断目前只是本轮审核草稿。
          只有最后提交正式审核决定时，
          “确认已解决”才会由后端事务正式保存；
          “继续要求修改”不会覆盖或删除以前的修改要求和处理记录。
        </div>
      </div>

      <div
        style={
          cwReviewCarryoverFilterRowStyle
        }
      >
        <button
          type="button"
          onClick={() =>
            changeViewMode(
              "pending",
            )
          }
          aria-pressed={
            viewMode === "pending"
          }
          style={
            cwReviewCarryoverFilterButtonStyle(
              viewMode ===
                "pending",
              "primary",
            )
          }
        >
          待处理{" "}
          {items.length -
            metrics.resolvedCount}
        </button>

        <button
          type="button"
          onClick={() =>
            changeViewMode(
              "confirmed",
            )
          }
          aria-pressed={
            viewMode === "confirmed"
          }
          style={
            cwReviewCarryoverFilterButtonStyle(
              viewMode ===
                "confirmed",
              "success",
            )
          }
        >
          本轮已确认{" "}
          {metrics.resolvedCount}
        </button>
      </div>

      {visibleGroups.length === 0 ? (
        <div
          style={{
            ...cwReviewCarryoverEmptyStateStyle,
            marginTop: "12px",
            background: C.card,
          }}
        >
          {viewMode === "confirmed"
            ? "本轮还没有暂选为“已解决”的问题。"
            : "当前没有待处理的问题。可以查看本轮已经确认的记录，或继续完成正式审核决定。"}
        </div>
      ) : (
        visibleGroups.map(
          ({
            group,
            visibleItems,
          }) => {
            const expanded =
              expandedGroupKeys ===
              null
                ? group.key ===
                  firstVisibleGroupKey
                : expandedGroupKeys.has(
                    group.key,
                  );

            const resolvedInGroup =
              group.items.filter(
                (item) =>
                  isCWReviewCarryoverMarkedResolved(
                    item,
                    resolvedIDSet,
                  ),
              ).length;

            const appliedInGroup =
              group.items.filter(
                (item) =>
                  item.status ===
                  "applied",
              ).length;

            const changedInGroup =
              group.items.filter(
                (item) =>
                  item.status ===
                    "stale" ||
                  item.status ===
                    "orphaned",
              ).length;

            return (
              <div
                key={group.key}
                style={
                  cwReviewCarryoverGroupContainerStyle
                }
              >
                <button
                  type="button"
                  onClick={() =>
                    toggleGroup(
                      group.key,
                    )
                  }
                  aria-expanded={
                    expanded
                  }
                  style={
                    cwReviewCarryoverGroupToggleStyle
                  }
                >
                  <span
                    aria-hidden="true"
                    style={{
                      ...cwReviewCarryoverArrowStyle,
                      color:
                        group.pageNumber >
                        0
                          ? C.primary
                          : C.purple,
                      transform:
                        expanded
                          ? "rotate(90deg)"
                          : "rotate(0deg)",
                    }}
                  >
                    ▶
                  </span>

                  <span
                    style={{
                      minWidth: 0,
                      flex: 1,
                    }}
                  >
                    <span
                      style={
                        cwReviewCarryoverGroupTitleStyle
                      }
                    >
                      {group.pageNumber >
                      0
                        ? `P${group.pageNumber} `
                        : "整课 "}
                      {group.pageTitle}
                    </span>

                    <span
                      style={
                        cwReviewCarryoverGroupSummaryStyle
                      }
                    >
                      {group.items.length} 条
                      {appliedInGroup > 0
                        ? ` · ${appliedInGroup} 条可复查`
                        : ""}
                      {resolvedInGroup > 0
                        ? ` · ${resolvedInGroup} 条本轮已确认`
                        : ""}
                      {changedInGroup > 0
                        ? ` · ${changedInGroup} 条需重新检查`
                        : ""}
                    </span>
                  </span>
                </button>

                {expanded && (
                  <div
                    style={
                      cwReviewCarryoverGroupBodyStyle
                    }
                  >
                    {visibleItems.map(
                      (item) => (
                        <CWReviewCarryoverItemCard
                          key={item.id}
                          item={item}
                          pages={pages}
                          selected={
                            isCWReviewCarryoverMarkedResolved(
                              item,
                              resolvedIDSet,
                            )
                          }
                          onResolvedChange={
                            handleResolvedChange
                          }
                          onSelectPage={
                            onSelectPage
                          }
                        />
                      ),
                    )}
                  </div>
                )}
              </div>
            );
          },
        )
      )}
    </section>
  );
}

function OverviewMetric({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone:
    | "primary"
    | "success"
    | "warning"
    | "danger";
}) {
  const config = {
    primary: {
      color: C.primary,
      background: "#EEF2FF",
    },
    success: {
      color: C.success,
      background: "#ECFDF5",
    },
    warning: {
      color: C.warning,
      background: "#FFF7ED",
    },
    danger: {
      color: C.danger,
      background: "#FEF2F2",
    },
  }[tone];

  const active =
    value > 0 ||
    label === "全部问题";

  return (
    <div
      style={{
        minWidth: "82px",
        padding: "8px 10px",
        borderRadius: "9px",
        background:
          active
            ? config.background
            : "#F8FAFC",
        color:
          active
            ? config.color
            : C.textMuted,
      }}
    >
      <div
        style={{
          fontSize: "20px",
          fontWeight: 700,
          lineHeight: 1.2,
        }}
      >
        {value}
      </div>

      <div
        style={{
          marginTop: "3px",
          fontSize: "12px",
          fontWeight: 600,
        }}
      >
        {label}
      </div>
    </div>
  );
}
