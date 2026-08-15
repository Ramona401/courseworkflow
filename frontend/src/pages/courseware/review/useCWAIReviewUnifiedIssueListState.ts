/**
 * useCWAIReviewUnifiedIssueListState.ts
 *
 * 统一问题工作台的筛选、深链接、展开分组和键盘导航状态。
 *
 * 关键边界：
 *   - 初始URL状态使用useState惰性初始化，不在render阶段读取ref.current；
 *   - 分组展开状态由当前分组和用户显式操作共同派生，不在effect中同步setState；
 *   - 已不存在的聚焦问题在渲染时自然退回列表，不在effect中同步setState；
 *   - 浏览器地址只保存问题ID与筛选枚举；
 *   - Alt+↓仅在当前工作台是最近可见工作区时打开下一条；
 *   - 不参与正式交付选择，也不修改整改项状态。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type RefObject,
} from "react";

import type {
  CWAIReviewItem,
} from "@/api/coursewares";

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

import {
  buildCWAIReviewUnifiedIssueGroups,
  buildCWAIReviewUnifiedIssueOverview,
  isCWAIReviewUnifiedIssueCompleted,
  sortCWAIReviewUnifiedNextTasks,
} from "./CWAIReviewUnifiedIssueList.shared";

export interface UseCWAIReviewUnifiedIssueListStateOptions {
  scope: CWReviewWorkspaceScope;
  items: CWAIReviewItem[];
  rootRef: RefObject<HTMLElement | null>;
  onSelectPage: (pageNumber: number) => void;
}

export function useCWAIReviewUnifiedIssueListState({
  scope,
  items,
  rootRef,
  onSelectPage,
}: UseCWAIReviewUnifiedIssueListStateOptions) {
  const [initialURLState] = useState(
    () => readCWReviewWorkspaceURLState(scope),
  );

  const [filters, setFilters] =
    useState<CWReviewIssueFilterState>(
      initialURLState.filters,
    );

  /**
   * 保存用户或浏览器地址请求打开的问题ID。
   *
   * 当该问题已经不在当前items中时，不需要effect再把state写空；
   * 下面的focusedItemID会直接派生为空。
   */
  const [
    requestedFocusedItemID,
    setRequestedFocusedItemID,
  ] = useState(
    initialURLState.focusedItemID,
  );

  const [ownsURL, setOwnsURL] =
    useState(initialURLState.owned);

  /**
   * 只保存教师对某个分组的明确展开/收起选择。
   *
   * 没有显式选择的分组使用当前列表的默认展开规则，
   * 因此无需在groups变化后通过effect清洗state。
   */
  const [
    groupExpansionOverrides,
    setGroupExpansionOverrides,
  ] = useState<Map<string, boolean>>(
    () => new Map(),
  );

  const overview = useMemo(
    () => buildCWAIReviewUnifiedIssueOverview(items),
    [items],
  );

  const filteredItems = useMemo(
    () => filterCWReviewItems(items, filters),
    [filters, items],
  );

  const groups = useMemo(
    () => buildCWAIReviewUnifiedIssueGroups(filteredItems),
    [filteredItems],
  );

  const nextItem = useMemo(
    () =>
      filteredItems
        .filter(
          (item) =>
            !isCWAIReviewUnifiedIssueCompleted(item),
        )
        .slice()
        .sort(sortCWAIReviewUnifiedNextTasks)[0],
    [filteredItems],
  );

  /**
   * 如果浏览器地址里的问题已经不存在，
   * 视图自然回到问题列表。
   *
   * requestedFocusedItemID仍保留原始请求，
   * 便于下面的URL同步effect判断是否需要清理地址。
   */
  const focusedItemID = useMemo(
    () =>
      requestedFocusedItemID &&
      items.some(
        (item) =>
          item.id === requestedFocusedItemID,
      )
        ? requestedFocusedItemID
        : "",
    [
      items,
      requestedFocusedItemID,
    ],
  );

  const defaultExpandedGroupKey = useMemo(() => {
    const firstPendingGroup =
      groups.find((group) =>
        group.items.some(
          (item) =>
            !isCWAIReviewUnifiedIssueCompleted(item),
        ),
      ) || groups[0];

    return firstPendingGroup?.key || "";
  }, [groups]);

  /**
   * 当前真正展开的分组完全由已有数据派生：
   *   - 用户明确操作过的分组尊重用户选择；
   *   - 其余分组只默认打开第一组待处理问题；
   *   - 已经消失的分组不会进入结果。
   */
  const expandedGroupKeys = useMemo(() => {
    const result = new Set<string>();

    for (const group of groups) {
      const override =
        groupExpansionOverrides.get(group.key);

      const expanded =
        override === undefined
          ? group.key === defaultExpandedGroupKey
          : override;

      if (expanded) {
        result.add(group.key);
      }
    }

    return result;
  }, [
    defaultExpandedGroupKey,
    groupExpansionOverrides,
    groups,
  ]);

  const activeFilterCount =
    countCWReviewActiveFilters(filters);

  useCWReviewListUsageTracking({
    mode: scope,
    initialDeepLinkOwned: initialURLState.owned,
    items,
    visibleItems: filteredItems,
    filters,
    focusedItemID,
  });

  const claimURL = useCallback(
    (
      nextFilters: CWReviewIssueFilterState,
      nextFocusedItemID: string,
    ) => {
      setOwnsURL(true);

      replaceCWReviewWorkspaceURL(
        scope,
        nextFilters,
        nextFocusedItemID,
      );
    },
    [scope],
  );

  const handleFiltersChange = useCallback(
    (nextFilters: CWReviewIssueFilterState) => {
      setFilters(nextFilters);
      claimURL(nextFilters, focusedItemID);
    },
    [
      claimURL,
      focusedItemID,
    ],
  );

  const handleFocusedItemChange = useCallback(
    (nextFocusedItemID: string) => {
      setRequestedFocusedItemID(
        nextFocusedItemID,
      );

      claimURL(
        filters,
        nextFocusedItemID,
      );
    },
    [
      claimURL,
      filters,
    ],
  );

  const toggleGroup = useCallback(
    (groupKey: string) => {
      setGroupExpansionOverrides(
        (previous) => {
          const next =
            new Map(previous);

          const currentlyExpanded =
            expandedGroupKeys.has(
              groupKey,
            );

          next.set(
            groupKey,
            !currentlyExpanded,
          );

          return next;
        },
      );
    },
    [expandedGroupKeys],
  );

  /**
   * 当深链接问题已经不存在时，只同步外部浏览器地址。
   *
   * React内部展示状态已经由focusedItemID派生为空，
   * 因此effect中不需要再调用setState。
   */
  useEffect(() => {
    if (
      !requestedFocusedItemID ||
      focusedItemID ||
      !ownsURL
    ) {
      return;
    }

    replaceCWReviewWorkspaceURL(
      scope,
      filters,
      "",
    );
  }, [
    filters,
    focusedItemID,
    ownsURL,
    requestedFocusedItemID,
    scope,
  ]);

  useEffect(() => {
    const handlePopState = () => {
      const next =
        readCWReviewWorkspaceURLState(scope);

      setOwnsURL(next.owned);

      if (next.owned) {
        setFilters(next.filters);
        setRequestedFocusedItemID(
          next.focusedItemID,
        );
      } else {
        setFilters(
          createDefaultCWReviewFilters(),
        );

        setRequestedFocusedItemID("");
      }
    };

    const handleURLClaim = (event: Event) => {
      const claimedScope =
        (
          event as CustomEvent<{
            scope?: string;
          }>
        ).detail?.scope;

      if (
        claimedScope &&
        claimedScope !== scope
      ) {
        setOwnsURL(false);
      }
    };

    window.addEventListener(
      "popstate",
      handlePopState,
    );

    window.addEventListener(
      CW_REVIEW_WORKSPACE_URL_EVENT,
      handleURLClaim,
    );

    return () => {
      window.removeEventListener(
        "popstate",
        handlePopState,
      );

      window.removeEventListener(
        CW_REVIEW_WORKSPACE_URL_EVENT,
        handleURLClaim,
      );
    };
  }, [scope]);

  useEffect(() => {
    if (
      focusedItemID ||
      !nextItem
    ) {
      return;
    }

    const handleKeyDown = (
      event: KeyboardEvent,
    ) => {
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
        visibleCount: filteredItems.length,
        activeFilterCount,
        pageNumber:
          nextItem.page_number_snapshot,
      });

      handleFocusedItemChange(
        nextItem.id,
      );

      if (
        nextItem.page_number_snapshot > 0
      ) {
        onSelectPage(
          nextItem.page_number_snapshot,
        );
      }
    };

    window.addEventListener(
      "keydown",
      handleKeyDown,
    );

    return () => {
      window.removeEventListener(
        "keydown",
        handleKeyDown,
      );
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
    rootRef,
    scope,
  ]);

  return {
    filters,
    focusedItemID,
    expandedGroupKeys,
    overview,
    filteredItems,
    groups,
    nextItem,
    activeFilterCount,

    handleFiltersChange,
    handleFocusedItemChange,
    toggleGroup,

    shareURL:
      buildCWReviewWorkspaceURL(
        scope,
        filters,
        focusedItemID,
      ),
  };
}
