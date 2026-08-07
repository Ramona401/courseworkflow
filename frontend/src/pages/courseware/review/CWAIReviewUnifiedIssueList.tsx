/**
 * CWAIReviewUnifiedIssueList.tsx
 *
 * 正式审核与作者自审共用的问题工作台。
 *
 * Phase 5A增加：
 *   - 关键词、严重程度、状态、页面和来源筛选；
 *   - 只保存问题ID与枚举条件的可恢复深链接；
 *   - “/”搜索、Alt+↓进入下一条任务；
 *   - 与单问题聚焦工作区共享URL和键盘导航。
 *
 * 协作选择与正式退回选择仍保持完全独立。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import type {
  CWAIReviewItem,
  CWAIReviewItemRelation,
  CWAIReviewSeverity,
} from "@/api/coursewares";

import CWAIReviewUnifiedIssueGroup, {
  type CWAIReviewUnifiedIssueGroupData,
} from "./CWAIReviewUnifiedIssueGroup";
import CWAIReviewUnifiedIssueToolbar from "./CWAIReviewUnifiedIssueToolbar";
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
  textSec: "#64748B",
  border: "#E2E8F0",
  bg: "#F8FAFC",
};

const SEVERITY_ORDER: Record<CWAIReviewSeverity, number> = {
  critical: 1,
  high: 2,
  medium: 3,
  low: 4,
  info: 5,
};

interface ReviewOverview {
  total: number;
  pending: number;
  applying: number;
  waitingConfirm: number;
  completed: number;
  stale: number;
}

interface VisibleUnifiedIssueGroup {
  group: CWAIReviewUnifiedIssueGroupData;
  visibleItems: CWAIReviewItem[];
}

export interface CWAIReviewUnifiedIssueListProps {
  mode: "formal" | "self";
  items: CWAIReviewItem[];
  governanceRelations: CWAIReviewItemRelation[];
  workSelectedItemIds: string[];
  deliverySelectedItemIds: string[];
  onToggleWorkSelection: (itemID: string, selected: boolean) => void;
  onClearWorkSelection: () => void;
  onOpenGlobalDiscussion: () => void;
  onOpenDirectRelation: () => void;
  onToggleDeliverySelection: (itemID: string, selected: boolean) => void;
  onItemChanged: (item: CWAIReviewItem) => void;
  onSelectPage: (pageNumber: number) => void;
  onInjectToRefine?: (item: CWAIReviewItem) => void;
}

function isCompletedItem(item: CWAIReviewItem): boolean {
  return item.status === "resolved" || item.status === "dismissed";
}

function buildUnifiedIssueGroups(
  items: CWAIReviewItem[],
): CWAIReviewUnifiedIssueGroupData[] {
  const groupMap = new Map<string, CWAIReviewUnifiedIssueGroupData>();

  for (const item of items) {
    const pageNumber = item.page_number_snapshot;
    const stablePageID = item.page_id?.trim() || "";
    const key =
      pageNumber <= 0
        ? "global"
        : stablePageID ||
          ["snapshot", pageNumber, item.page_title_snapshot].join(":");

    const current = groupMap.get(key) || {
      key,
      pageNumber: pageNumber <= 0 ? 0 : pageNumber,
      pageTitle: pageNumber <= 0 ? "整课综合问题" : item.page_title_snapshot,
      items: [],
    };

    current.items.push(item);

    if (pageNumber > 0 && !current.pageTitle && item.page_title_snapshot) {
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

      return (left.created_at || "").localeCompare(right.created_at || "");
    });
  }

  return groups.sort((left, right) => {
    if (left.pageNumber <= 0 && right.pageNumber > 0) return -1;
    if (right.pageNumber <= 0 && left.pageNumber > 0) return 1;
    return left.pageNumber - right.pageNumber;
  });
}

function buildOverview(items: CWAIReviewItem[]): ReviewOverview {
  const overview: ReviewOverview = {
    total: items.length,
    pending: 0,
    applying: 0,
    waitingConfirm: 0,
    completed: 0,
    stale: 0,
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
  }

  return overview;
}

function isDeliverableFormalItem(item: CWAIReviewItem): boolean {
  return (
    item.source_type === "formal" &&
    item.status === "confirmed" &&
    !!item.confirmed_instruction.trim() &&
    !item.courseware_review_id &&
    !item.feedback_id
  );
}

function nextTaskPriority(item: CWAIReviewItem): number {
  if (item.severity === "critical") return 0;
  if (item.severity === "high") return 10;
  if (item.status === "stale") return 20;
  if (item.status === "applying") return 30;
  return 40 + (SEVERITY_ORDER[item.severity] || 99);
}

function sortNextTasks(
  left: CWAIReviewItem,
  right: CWAIReviewItem,
): number {
  const priorityDifference =
    nextTaskPriority(left) - nextTaskPriority(right);

  if (priorityDifference !== 0) {
    return priorityDifference;
  }

  const pageDifference =
    left.page_number_snapshot - right.page_number_snapshot;

  if (pageDifference !== 0) {
    return pageDifference;
  }

  return (left.created_at || "").localeCompare(right.created_at || "");
}

export default function CWAIReviewUnifiedIssueList({
  mode,
  items,
  governanceRelations,
  workSelectedItemIds,
  deliverySelectedItemIds,
  onToggleWorkSelection,
  onClearWorkSelection,
  onOpenGlobalDiscussion,
  onOpenDirectRelation,
  onToggleDeliverySelection,
  onItemChanged,
  onSelectPage,
  onInjectToRefine,
}: CWAIReviewUnifiedIssueListProps) {
  const scope = mode as CWReviewWorkspaceScope;
  const rootRef = useRef<HTMLElement | null>(null);
  const initialURLStateRef = useRef(readCWReviewWorkspaceURLState(scope));
  const [filters, setFilters] = useState<CWReviewIssueFilterState>(
    initialURLStateRef.current.filters,
  );
  const [focusedItemID, setFocusedItemID] = useState(
    initialURLStateRef.current.focusedItemID,
  );
  const [ownsURL, setOwnsURL] = useState(
    initialURLStateRef.current.owned,
  );
  const [expandedGroupKeys, setExpandedGroupKeys] =
    useState<Set<string>>(new Set());

  const isSelfReview = mode === "self";
  const overview = useMemo(() => buildOverview(items), [items]);
  const filteredItems = useMemo(
    () => filterCWReviewItems(items, filters),
    [filters, items],
  );
  const groups = useMemo(
    () => buildUnifiedIssueGroups(filteredItems),
    [filteredItems],
  );

  const workSelectedItemIDSet = useMemo(
    () => new Set(workSelectedItemIds),
    [workSelectedItemIds],
  );
  const deliverySelectedItemIDSet = useMemo(
    () => new Set(deliverySelectedItemIds),
    [deliverySelectedItemIds],
  );
  const activeRelations = useMemo(
    () => governanceRelations.filter((relation) => relation.status === "active"),
    [governanceRelations],
  );
  const selectedWorkItems = useMemo(
    () => items.filter((item) => workSelectedItemIDSet.has(item.id)),
    [items, workSelectedItemIDSet],
  );
  const visibleGroups = useMemo<VisibleUnifiedIssueGroup[]>(
    () => groups.map((group) => ({ group, visibleItems: group.items })),
    [groups],
  );
  const nextItem = useMemo(
    () =>
      filteredItems
        .filter((item) => !isCompletedItem(item))
        .slice()
        .sort(sortNextTasks)[0],
    [filteredItems],
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

  const nextGroup = nextItem
    ? groups.find((group) => group.items.some((item) => item.id === nextItem.id))
    : undefined;

  const workSelectedCount = selectedWorkItems.length;
  const canOpenDiscussion = workSelectedCount >= 2 && workSelectedCount <= 12;
  const canOpenRelation = workSelectedCount === 2;
  const addableDeliveryItems = isSelfReview
    ? []
    : selectedWorkItems.filter(
        (item) =>
          isDeliverableFormalItem(item) &&
          !deliverySelectedItemIDSet.has(item.id),
      );
  const removableDeliveryItems = isSelfReview
    ? []
    : selectedWorkItems.filter((item) => deliverySelectedItemIDSet.has(item.id));

  const handleBatchDeliveryChange = (selected: boolean) => {
    const targets = selected ? addableDeliveryItems : removableDeliveryItems;

    for (const item of targets) {
      onToggleDeliverySelection(item.id, selected);
    }
  };

  const toggleGroup = (groupKey: string) => {
    setExpandedGroupKeys((previous) => {
      const next = new Set(previous);
      next.has(groupKey) ? next.delete(groupKey) : next.add(groupKey);
      return next;
    });
  };

  const continueNext = () => {
    if (!nextItem || !nextGroup) return;

    setExpandedGroupKeys((previous) => new Set(previous).add(nextGroup.key));
    handleFocusedItemChange(nextItem.id);

    if (nextItem.page_number_snapshot > 0) {
      onSelectPage(nextItem.page_number_snapshot);
    }
  };

  const openItemWorkspace = (itemID: string) => {
    const target = items.find((item) => item.id === itemID);
    if (!target) return;

    handleFocusedItemChange(itemID);

    if (target.page_number_snapshot > 0) {
      onSelectPage(target.page_number_snapshot);
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
      <CWAIReviewUnifiedIssueToolbar
        isSelfReview={isSelfReview}
        overview={overview}
        activeRelationCount={activeRelations.length}
        deliverySelectedCount={deliverySelectedItemIds.length}
        workSelectedCount={workSelectedCount}
        canOpenDiscussion={canOpenDiscussion}
        canOpenRelation={canOpenRelation}
        addableDeliveryCount={addableDeliveryItems.length}
        removableDeliveryCount={removableDeliveryItems.length}
        showCompleted={filters.showCompleted}
        hasNextItem={!!nextItem}
        onContinueNext={continueNext}
        onShowCompletedChange={(showCompleted: boolean) =>
          handleFiltersChange({
            ...filters,
            showCompleted,
            status:
              !showCompleted && filters.status === "completed"
                ? "all"
                : filters.status,
          })
        }
        onOpenGlobalDiscussion={onOpenGlobalDiscussion}
        onOpenDirectRelation={onOpenDirectRelation}
        onAddSelectedToDelivery={() => handleBatchDeliveryChange(true)}
        onRemoveSelectedFromDelivery={() => handleBatchDeliveryChange(false)}
        onClearWorkSelection={onClearWorkSelection}
      />

      <CWReviewIssueFilters
        items={items}
        value={filters}
        resultCount={filteredItems.length}
        shareURL={shareURL}
        onChange={handleFiltersChange}
      />

      {focusedItemID ? (
        <CWReviewIssueWorkspace
          mode={scope}
          items={workspaceItems}
          currentItemId={focusedItemID}
          allItems={items}
          governanceRelations={activeRelations}
          selectable={!isSelfReview}
          selectedItemIds={deliverySelectedItemIds}
          onSelectedChange={onToggleDeliverySelection}
          onClose={() => handleFocusedItemChange("")}
          onNavigateItem={handleFocusedItemChange}
          onSelectPage={onSelectPage}
          onItemChanged={onItemChanged}
          onInjectToRefine={onInjectToRefine}
        />
      ) : visibleGroups.length === 0 ? (
        <div
          style={{
            marginTop: "14px",
            padding: "28px 16px",
            borderRadius: "10px",
            border: `1px dashed ${C.border}`,
            background: "#FFFFFF",
            color: C.textSec,
            fontSize: "14px",
            lineHeight: 1.7,
            textAlign: "center",
          }}
        >
          当前筛选条件下没有问题。可以清除筛选或复制当前视图链接交给其他审核人员。
        </div>
      ) : (
        visibleGroups.map(({ group, visibleItems }) => (
          <CWAIReviewUnifiedIssueGroup
            key={group.key}
            group={group}
            visibleItems={visibleItems}
            expanded={expandedGroupKeys.has(group.key)}
            allItems={items}
            activeRelations={activeRelations}
            isSelfReview={isSelfReview}
            workSelectedItemIDSet={workSelectedItemIDSet}
            deliverySelectedItemIDSet={deliverySelectedItemIDSet}
            onToggleExpanded={() => toggleGroup(group.key)}
            onOpenItem={openItemWorkspace}
            onToggleWorkSelection={onToggleWorkSelection}
            onToggleDeliverySelection={onToggleDeliverySelection}
            onItemChanged={onItemChanged}
            onSelectPage={onSelectPage}
            onInjectToRefine={onInjectToRefine}
          />
        ))
      )}
    </section>
  );
}

export type { ReviewOverview as CWAIReviewUnifiedIssueOverview };
