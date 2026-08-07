/**
 * CWAIReviewPageChecklist.tsx
 *
 * 作者正式整改与共享按页审核清单。
 *
 * Phase 5A增加搜索、筛选、可恢复深链接和键盘进入下一条任务；
 * 页面聚合、正式提交阻断、关系展示和问题状态规则保持不变。
 */

import {
  type CSSProperties,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import type {
  CWAIReviewItem,
  CWAIReviewItemRelation,
  CWAIReviewItemStatus,
  CWAIReviewSeverity,
} from "@/api/coursewares";

import CWAIReviewItemPanel from "./CWAIReviewItemPanel";
import CWReviewIssueFilters from "./CWReviewIssueFilters";
import CWReviewIssueWorkspace from "./CWReviewIssueWorkspace";
import {
  buildCWReviewWorkspaceURL,
  createDefaultCWReviewFilters,
  CW_REVIEW_WORKSPACE_URL_EVENT,
  filterCWReviewItems,
  isCWReviewTextEntryTarget,
  isNearestVisibleCWReviewElement,
  readCWReviewWorkspaceURLState,
  replaceCWReviewWorkspaceURL,
  type CWReviewIssueFilterState,
  type CWReviewWorkspaceScope,
} from "./coursewareReviewWorkspaceState";

import {
  countCWReviewActiveFilters,
  recordCoursewareReviewUsage,
  useCWReviewListUsageTracking,
} from "./coursewareReviewUsage";

const C = {
  primary: "#4F7BE8",
  success: "#059669",
  warning: "#D97706",
  danger: "#DC2626",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  bg: "#F8FAFC",
  card: "#FFFFFF",
};

const SEVERITY_ORDER: Record<CWAIReviewSeverity, number> = {
  critical: 1,
  high: 2,
  medium: 3,
  low: 4,
  info: 5,
};

const STATUS_ORDER: Record<CWAIReviewItemStatus, number> = {
  stale: 1,
  orphaned: 2,
  detected: 3,
  discussing: 4,
  confirmed: 5,
  applying: 6,
  applied: 7,
  dismissed: 8,
  resolved: 9,
};

interface PageReviewGroup {
  key: string;
  pageNumber: number;
  pageTitle: string;
  items: CWAIReviewItem[];
}

interface ReviewOverview {
  total: number;
  pending: number;
  applying: number;
  waitingConfirm: number;
  completed: number;
  stale: number;
  blocking: number;
}

export interface CWAIReviewPageChecklistProps {
  items: CWAIReviewItem[];
  allItems?: CWAIReviewItem[];
  governanceRelations?: CWAIReviewItemRelation[];
  mode?: "review" | "remediation";
  selectable: boolean;
  selectedItemIds: string[];
  onToggleItemSelection: (itemId: string, selected: boolean) => void;
  onItemChanged: (item: CWAIReviewItem) => void;
  onSelectPage: (pageNumber: number) => void;
  onInjectToRefine?: (item: CWAIReviewItem) => void;
}

function isCompletedItem(item: CWAIReviewItem): boolean {
  return item.status === "resolved" || item.status === "dismissed";
}

function isSubmissionBlockingStatus(status: CWAIReviewItemStatus): boolean {
  return status !== "resolved" && status !== "dismissed" && status !== "applied";
}

function buildPageGroups(items: CWAIReviewItem[]): PageReviewGroup[] {
  const groupMap = new Map<string, PageReviewGroup>();

  for (const item of items) {
    const isGlobalIssue = item.page_number_snapshot <= 0;
    const stablePageID = item.page_id?.trim() || "";
    const key = isGlobalIssue
      ? "global"
      : stablePageID ||
        ["snapshot", item.page_number_snapshot, item.page_title_snapshot].join(":");

    const current = groupMap.get(key) || {
      key,
      pageNumber: isGlobalIssue ? 0 : item.page_number_snapshot,
      pageTitle: isGlobalIssue ? "整课问题" : item.page_title_snapshot,
      items: [],
    };

    current.items.push(item);

    if (!current.pageTitle && item.page_title_snapshot) {
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

      const statusDifference =
        (STATUS_ORDER[left.status] || 99) -
        (STATUS_ORDER[right.status] || 99);

      if (statusDifference !== 0) {
        return statusDifference;
      }

      return (left.created_at || "").localeCompare(right.created_at || "");
    });
  }

  return groups.sort((left, right) => {
    if (left.pageNumber === 0) return -1;
    if (right.pageNumber === 0) return 1;
    return left.pageNumber - right.pageNumber;
  });
}

function buildOverview(
  items: CWAIReviewItem[],
  mode: "review" | "remediation",
): ReviewOverview {
  const overview: ReviewOverview = {
    total: items.length,
    pending: 0,
    applying: 0,
    waitingConfirm: 0,
    completed: 0,
    stale: 0,
    blocking: 0,
  };

  for (const item of items) {
    if (isCompletedItem(item)) {
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

    if (
      mode === "remediation" &&
      item.source_type === "formal" &&
      isSubmissionBlockingStatus(item.status)
    ) {
      overview.blocking += 1;
    }
  }

  return overview;
}

function metricStyle(
  active: boolean,
  tone: "primary" | "success" | "warning" | "danger",
): CSSProperties {
  const config = {
    primary: { color: C.primary, background: "#EEF2FF" },
    success: { color: C.success, background: "#ECFDF5" },
    warning: { color: C.warning, background: "#FFF7ED" },
    danger: { color: C.danger, background: "#FEF2F2" },
  }[tone];

  return {
    minWidth: "82px",
    padding: "8px 10px",
    borderRadius: "9px",
    background: active ? config.background : "#F8FAFC",
    color: active ? config.color : C.textMuted,
  };
}

export default function CWAIReviewPageChecklist({
  items,
  allItems,
  governanceRelations,
  mode = "review",
  selectable,
  selectedItemIds,
  onToggleItemSelection,
  onItemChanged,
  onSelectPage,
  onInjectToRefine,
}: CWAIReviewPageChecklistProps) {
  const scope: CWReviewWorkspaceScope =
    mode === "remediation" ? "remediation" : "formal";
  const rootRef = useRef<HTMLElement | null>(null);
  const initialURLStateRef = useRef(readCWReviewWorkspaceURLState(scope));
  const [filters, setFilters] = useState<CWReviewIssueFilterState>(
    initialURLStateRef.current.filters,
  );
  const [focusedItemID, setFocusedItemID] = useState(
    initialURLStateRef.current.focusedItemID,
  );
  const [ownsURL, setOwnsURL] = useState(initialURLStateRef.current.owned);
  const [expandedGroupKeys, setExpandedGroupKeys] =
    useState<Set<string>>(new Set());

  const isRemediation = mode === "remediation";
  const overview = useMemo(() => buildOverview(items, mode), [items, mode]);
  const filteredItems = useMemo(
    () => filterCWReviewItems(items, filters),
    [filters, items],
  );
  const groups = useMemo(() => buildPageGroups(filteredItems), [filteredItems]);
  const selectedItemIDSet = useMemo(
    () => new Set(selectedItemIds),
    [selectedItemIds],
  );
  const relationItems = allItems || items;
  const activeGovernanceRelations = useMemo(
    () => (governanceRelations || []).filter((relation) => relation.status === "active"),
    [governanceRelations],
  );
  const primaryItemIDSet = useMemo(() => {
    const result = new Set<string>();

    for (const relation of activeGovernanceRelations) {
      if (relation.relation_type === "duplicate" || relation.relation_type === "merge") {
        result.add(relation.target_item_id);
      }
    }

    return result;
  }, [activeGovernanceRelations]);

  const nextGroup = useMemo(
    () =>
      groups.find((group) =>
        group.items.some((item) => !isCompletedItem(item)),
      ),
    [groups],
  );
  const nextItem =
    nextGroup?.items.find(
      (item) =>
        !isCompletedItem(item),
    );

  const activeFilterCount =
    countCWReviewActiveFilters(
      filters,
    );

  useCWReviewListUsageTracking({
    mode: scope,
    initialDeepLinkOwned:
      initialURLStateRef.current.owned,
    items,
    visibleItems: filteredItems,
    filters,
    focusedItemID,
  });

  const claimURL = useCallback(
    (nextFilters: CWReviewIssueFilterState, nextFocusedItemID: string) => {
      setOwnsURL(true);
      replaceCWReviewWorkspaceURL(scope, nextFilters, nextFocusedItemID);
    },
    [scope],
  );

  const handleFiltersChange = useCallback(
    (nextFilters: CWReviewIssueFilterState) => {
      setFilters(nextFilters);
      claimURL(nextFilters, focusedItemID);
    },
    [claimURL, focusedItemID],
  );

  const handleFocusedItemChange = useCallback(
    (nextFocusedItemID: string) => {
      setFocusedItemID(nextFocusedItemID);
      claimURL(filters, nextFocusedItemID);
    },
    [claimURL, filters],
  );

  useEffect(() => {
    setExpandedGroupKeys((previous) => {
      const validKeys = new Set(groups.map((group) => group.key));
      const next = new Set(Array.from(previous).filter((key) => validKeys.has(key)));

      if (next.size === 0) {
        const firstPendingGroup =
          groups.find((group) => group.items.some((item) => !isCompletedItem(item))) ||
          groups[0];

        if (firstPendingGroup) {
          next.add(firstPendingGroup.key);
        }
      }

      return next;
    });
  }, [groups]);

  useEffect(() => {
    if (items.length === 0 || !focusedItemID) return;

    if (!items.some((item) => item.id === focusedItemID)) {
      setFocusedItemID("");
      if (ownsURL) {
        replaceCWReviewWorkspaceURL(scope, filters, "");
      }
    }
  }, [filters, focusedItemID, items, ownsURL, scope]);

  useEffect(() => {
    const handlePopState = () => {
      const next = readCWReviewWorkspaceURLState(scope);
      setOwnsURL(next.owned);

      if (next.owned) {
        setFilters(next.filters);
        setFocusedItemID(next.focusedItemID);
      } else {
        setFilters(createDefaultCWReviewFilters());
        setFocusedItemID("");
      }
    };

    const handleURLClaim = (event: Event) => {
      const claimedScope =
        (event as CustomEvent<{ scope?: string }>).detail?.scope;

      if (claimedScope && claimedScope !== scope) {
        setOwnsURL(false);
      }
    };

    window.addEventListener("popstate", handlePopState);
    window.addEventListener(CW_REVIEW_WORKSPACE_URL_EVENT, handleURLClaim);

    return () => {
      window.removeEventListener("popstate", handlePopState);
      window.removeEventListener(CW_REVIEW_WORKSPACE_URL_EVENT, handleURLClaim);
    };
  }, [scope]);

  useEffect(() => {
    if (focusedItemID || !nextItem) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        !event.altKey ||
        event.key !== "ArrowDown" ||
        isCWReviewTextEntryTarget(event.target) ||
        (
          !ownsURL &&
          !isNearestVisibleCWReviewElement(
            rootRef.current,
            '[data-cw-review-shortcut-root="true"]',
          )
        )
      ) {
        return;
      }

      event.preventDefault();

      recordCoursewareReviewUsage({
        event: "keyboard_shortcut",
        mode: scope,
        shortcut: "open_next_task",
        totalCount: items.length,
        visibleCount:
          filteredItems.length,
        activeFilterCount,
        pageNumber:
          nextItem.page_number_snapshot,
      });

      handleFocusedItemChange(nextItem.id);

      if (nextItem.page_number_snapshot > 0) {
        onSelectPage(nextItem.page_number_snapshot);
      }
    };

    window.addEventListener("keydown", handleKeyDown);

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [
    activeFilterCount,
    filteredItems.length,
    focusedItemID,
    handleFocusedItemChange,
    items.length,
    nextItem,
    onSelectPage,
    ownsURL,
    scope,
  ]);

  if (items.length === 0) {
    return null;
  }

  const toggleGroup = (groupKey: string) => {
    setExpandedGroupKeys((previous) => {
      const next = new Set(previous);
      next.has(groupKey) ? next.delete(groupKey) : next.add(groupKey);
      return next;
    });
  };

  const continueNext = () => {
    if (!nextGroup || !nextItem) return;

    setExpandedGroupKeys((previous) => new Set(previous).add(nextGroup.key));
    handleFocusedItemChange(nextItem.id);

    if (nextItem.page_number_snapshot > 0) {
      onSelectPage(nextItem.page_number_snapshot);
    }
  };

  const openItemWorkspace = (item: CWAIReviewItem) => {
    handleFocusedItemChange(item.id);

    if (item.page_number_snapshot > 0) {
      onSelectPage(item.page_number_snapshot);
    }
  };

  const workspaceItems =
    filteredItems.some((item) => item.id === focusedItemID)
      ? filteredItems
      : items;
  const shareURL = buildCWReviewWorkspaceURL(scope, filters, focusedItemID);

  return (
    <section
      ref={rootRef}
      data-cw-review-shortcut-root="true"
      data-cw-review-scope={scope}
      style={{
        padding: "16px",
        borderRadius: "12px",
        border: `1px solid ${C.primary}35`,
        background: C.bg,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: "12px",
          flexWrap: "wrap",
        }}
      >
        <div style={{ minWidth: 0, flex: "1 1 360px" }}>
          <div style={{ color: C.text, fontSize: "18px", fontWeight: 700, lineHeight: 1.4 }}>
            {isRemediation ? "按页整改工作区" : "按页审核工作区"}
          </div>

          <div
            style={{
              marginTop: "5px",
              maxWidth: "75ch",
              color: C.textSec,
              fontSize: "14px",
              lineHeight: 1.6,
            }}
          >
            默认只显示未完成任务。可按问题、页面和状态快速收窄范围，再进入单问题工作区处理。
          </div>
        </div>

        {nextItem && (
          <button type="button" onClick={continueNext} style={primaryButtonStyle}>
            继续处理下一条
          </button>
        )}
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(92px, 1fr))",
          gap: "8px",
          marginTop: "14px",
        }}
      >
        <OverviewMetric label="全部问题" value={overview.total} style={metricStyle(true, "primary")} />
        <OverviewMetric
          label="待处理"
          value={overview.pending}
          style={metricStyle(overview.pending > 0, "warning")}
        />
        <OverviewMetric
          label="修改中"
          value={overview.applying}
          style={metricStyle(overview.applying > 0, "warning")}
        />
        <OverviewMetric
          label="待确认"
          value={overview.waitingConfirm}
          style={metricStyle(overview.waitingConfirm > 0, "primary")}
        />
        <OverviewMetric
          label="已完成"
          value={overview.completed}
          style={metricStyle(overview.completed > 0, "success")}
        />
        <OverviewMetric
          label="页面变化"
          value={overview.stale}
          style={metricStyle(overview.stale > 0, "danger")}
        />
      </div>

      {isRemediation && overview.blocking > 0 && (
        <div style={blockingNoticeStyle}>
          暂时不能重新提交：还有 {overview.blocking} 条正式整改尚未达到可提交状态。
        </div>
      )}

      <CWReviewIssueFilters
        items={items}
        value={filters}
        resultCount={filteredItems.length}
        shareURL={shareURL}
        onChange={handleFiltersChange}
      />

      {activeGovernanceRelations.length > 0 && (
        <div
          style={{
            marginTop: "10px",
            color: "#7C3AED",
            fontSize: "12px",
            fontWeight: 600,
          }}
        >
          已确认关系 {activeGovernanceRelations.length}
        </div>
      )}

      {focusedItemID ? (
        <CWReviewIssueWorkspace
          mode={scope}
          items={workspaceItems}
          currentItemId={focusedItemID}
          allItems={relationItems}
          governanceRelations={activeGovernanceRelations}
          selectable={selectable}
          selectedItemIds={selectedItemIds}
          onSelectedChange={onToggleItemSelection}
          onClose={() => handleFocusedItemChange("")}
          onNavigateItem={handleFocusedItemChange}
          onSelectPage={onSelectPage}
          onItemChanged={onItemChanged}
          onInjectToRefine={onInjectToRefine}
        />
      ) : groups.length === 0 ? (
        <div style={emptyStateStyle}>
          当前筛选条件下没有问题。可以清除筛选，或复制当前视图链接用于后续继续处理。
        </div>
      ) : (
        groups.map((group) => {
          const expanded = expandedGroupKeys.has(group.key);
          const selectedCount = group.items.filter((item) => selectedItemIDSet.has(item.id)).length;
          const pendingCount = group.items.filter((item) => !isCompletedItem(item)).length;
          const waitingCount = group.items.filter((item) => item.status === "applied").length;
          const staleCount = group.items.filter((item) => item.status === "stale").length;
          const completedCount = group.items.filter(isCompletedItem).length;
          const groupItemIDSet = new Set(group.items.map((item) => item.id));
          const groupRelations = activeGovernanceRelations.filter(
            (relation) =>
              groupItemIDSet.has(relation.source_item_id) ||
              groupItemIDSet.has(relation.target_item_id),
          );
          const groupPrimaryCount = group.items.filter((item) =>
            primaryItemIDSet.has(item.id),
          ).length;

          return (
            <div key={group.key} style={groupContainerStyle}>
              <div
                style={{
                  ...groupHeaderStyle,
                  background: pendingCount > 0 ? "#FFFFFF" : "#F8FAFC",
                }}
              >
                <button
                  type="button"
                  onClick={() => toggleGroup(group.key)}
                  aria-expanded={expanded}
                  style={groupToggleStyle}
                >
                  <span
                    aria-hidden="true"
                    style={{
                      ...arrowStyle,
                      transform: expanded ? "rotate(90deg)" : "rotate(0deg)",
                    }}
                  >
                    ▶
                  </span>

                  <span style={{ minWidth: 0, flex: 1 }}>
                    <span style={groupTitleStyle}>
                      {group.pageNumber > 0 ? `P${group.pageNumber} ` : "整课 "}
                      {group.pageTitle ||
                        (group.pageNumber > 0 ? `第${group.pageNumber}页` : "整课问题")}
                    </span>

                    <span style={groupSummaryStyle}>
                      {group.items.length} 条问题
                      {pendingCount > 0 ? ` · ${pendingCount} 条未完成` : ""}
                      {waitingCount > 0 ? ` · ${waitingCount} 条待确认` : ""}
                      {staleCount > 0 ? ` · ${staleCount} 条页面变化` : ""}
                      {completedCount > 0 ? ` · ${completedCount} 条已完成` : ""}
                    </span>
                  </span>
                </button>

                {group.pageNumber > 0 && (
                  <button
                    type="button"
                    onClick={() => onSelectPage(group.pageNumber)}
                    style={openPageButtonStyle}
                  >
                    打开页面
                  </button>
                )}
              </div>

              {(groupRelations.length > 0 ||
                groupPrimaryCount > 0 ||
                (!isRemediation && selectedCount > 0)) && (
                <div style={groupMetaStyle}>
                  {groupRelations.length > 0 && <span>关联 {groupRelations.length}</span>}
                  {groupPrimaryCount > 0 && <span>主问题 {groupPrimaryCount}</span>}
                  {!isRemediation && selectedCount > 0 && <span>待交付 {selectedCount}</span>}
                </div>
              )}

              {expanded && (
                <div style={groupBodyStyle}>
                  {group.items.map((item) => (
                    <div key={item.id} style={{ marginTop: "10px" }}>
                      <div style={{ display: "flex", justifyContent: "flex-end" }}>
                        <button
                          type="button"
                          onClick={() => openItemWorkspace(item)}
                          style={compactPrimaryButtonStyle}
                        >
                          处理这条
                        </button>
                      </div>

                      <CWAIReviewItemPanel
                        item={item}
                        allItems={relationItems}
                        governanceRelations={activeGovernanceRelations}
                        selectable={selectable}
                        selected={selectedItemIDSet.has(item.id)}
                        onSelectedChange={onToggleItemSelection}
                        onSelectPage={onSelectPage}
                        onChanged={onItemChanged}
                        onInjectToRefine={onInjectToRefine}
                      />
                    </div>
                  ))}
                </div>
              )}
            </div>
          );
        })
      )}
    </section>
  );
}

function OverviewMetric({
  label,
  value,
  style,
}: {
  label: string;
  value: number;
  style: CSSProperties;
}) {
  return (
    <div style={style}>
      <div style={{ fontSize: "20px", fontWeight: 700, lineHeight: 1.2 }}>{value}</div>
      <div style={{ marginTop: "3px", fontSize: "12px", fontWeight: 600, lineHeight: 1.4 }}>
        {label}
      </div>
    </div>
  );
}

const primaryButtonStyle: CSSProperties = {
  minHeight: "36px",
  padding: "8px 12px",
  borderRadius: "8px",
  border: `1px solid ${C.primary}`,
  background: C.primary,
  color: "#FFFFFF",
  fontSize: "13px",
  fontWeight: 700,
  cursor: "pointer",
};

const blockingNoticeStyle: CSSProperties = {
  marginTop: "12px",
  padding: "10px 12px",
  borderRadius: "8px",
  border: "1px solid #FECACA",
  background: "#FEF2F2",
  color: C.danger,
  fontSize: "14px",
  fontWeight: 600,
  lineHeight: 1.6,
};

const emptyStateStyle: CSSProperties = {
  marginTop: "14px",
  padding: "28px 16px",
  borderRadius: "10px",
  border: `1px dashed ${C.border}`,
  background: "#FFFFFF",
  color: C.textSec,
  fontSize: "14px",
  lineHeight: 1.7,
  textAlign: "center",
};

const groupContainerStyle: CSSProperties = {
  marginTop: "12px",
  borderRadius: "10px",
  border: `1px solid ${C.border}`,
  background: C.card,
  overflow: "hidden",
};

const groupHeaderStyle: CSSProperties = {
  display: "flex",
  alignItems: "stretch",
  gap: "8px",
  padding: "10px 12px",
};

const groupToggleStyle: CSSProperties = {
  minWidth: 0,
  flex: 1,
  display: "flex",
  alignItems: "center",
  gap: "10px",
  padding: 0,
  border: "none",
  background: "transparent",
  color: C.text,
  textAlign: "left",
  cursor: "pointer",
};

const arrowStyle: CSSProperties = {
  flexShrink: 0,
  color: C.primary,
  fontSize: "16px",
  transition: "transform 160ms ease",
};

const groupTitleStyle: CSSProperties = {
  display: "block",
  overflow: "hidden",
  color: C.text,
  fontSize: "15px",
  fontWeight: 700,
  lineHeight: 1.5,
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
};

const groupSummaryStyle: CSSProperties = {
  display: "block",
  marginTop: "3px",
  color: C.textMuted,
  fontSize: "12px",
  lineHeight: 1.5,
};

const openPageButtonStyle: CSSProperties = {
  flexShrink: 0,
  minHeight: "36px",
  padding: "8px 10px",
  borderRadius: "8px",
  border: `1px solid ${C.primary}`,
  background: "#EEF2FF",
  color: C.primary,
  fontSize: "12px",
  fontWeight: 700,
  cursor: "pointer",
};

const groupMetaStyle: CSSProperties = {
  display: "flex",
  gap: "10px",
  flexWrap: "wrap",
  padding: "0 12px 10px",
  color: C.textMuted,
  fontSize: "12px",
  lineHeight: 1.5,
};

const groupBodyStyle: CSSProperties = {
  padding: "0 12px 12px",
  borderTop: `1px solid ${C.border}`,
};

const compactPrimaryButtonStyle: CSSProperties = {
  minHeight: "32px",
  padding: "6px 10px",
  borderRadius: "8px",
  border: `1px solid ${C.primary}`,
  background: C.primary,
  color: "#FFFFFF",
  fontSize: "12px",
  fontWeight: 700,
  cursor: "pointer",
};
