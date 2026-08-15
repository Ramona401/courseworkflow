/**
 * CWAIReviewUnifiedIssueList.tsx
 *
 * 正式审核与作者自审共用的问题工作台。
 *
 * 本层只负责视图组合：
 *   - 筛选、深链接和键盘状态由useCWAIReviewUnifiedIssueListState管理；
 *   - 页面分组和下一条任务排序由shared纯函数管理；
 *   - “选择一起分析”只决定综合讨论和问题关系的比较范围；
 *   - 正式交付只能由单条教师改进卡的主要操作改变；
 *   - 工具栏没有批量加入或移出本次修改清单的入口。
 */

import {
  useMemo,
  useRef,
} from "react";

import type {
  CWAIReviewItem,
  CWAIReviewItemRelation,
} from "@/api/coursewares";

import CWAIReviewUnifiedIssueGroup from "./CWAIReviewUnifiedIssueGroup";
import CWAIReviewUnifiedIssueToolbar from "./CWAIReviewUnifiedIssueToolbar";
import CWReviewIssueFilters from "./CWReviewIssueFilters";
import CWReviewIssueWorkspace from "./CWReviewIssueWorkspace";

import {
  type CWReviewWorkspaceScope,
} from "./coursewareReviewWorkspaceState";

import {
  useCWAIReviewUnifiedIssueListState,
} from "./useCWAIReviewUnifiedIssueListState";

const C = {
  primary: "#4F7BE8",
  textSec: "#64748B",
  border: "#E2E8F0",
  bg: "#F8FAFC",
};

export interface CWAIReviewUnifiedIssueListProps {
  mode: "formal" | "self";
  items: CWAIReviewItem[];
  governanceRelations: CWAIReviewItemRelation[];

  /** 只用于多问题比较、综合讨论和关系说明。 */
  workSelectedItemIds: string[];

  /** 正式交付选择，只允许单条问题卡主要操作改变。 */
  deliverySelectedItemIds: string[];

  onToggleWorkSelection: (
    itemID: string,
    selected: boolean,
  ) => void;

  onClearWorkSelection: () => void;
  onOpenGlobalDiscussion: () => void;
  onOpenDirectRelation: () => void;

  onToggleDeliverySelection: (
    itemID: string,
    selected: boolean,
  ) => void;

  /**
   * 已有问题与目标漂移新问题共用的upsert入口。
   * 父层mergeCWAIReviewItems按ID更新或新增。
   */
  onItemChanged: (item: CWAIReviewItem) => void;

  onSelectPage: (pageNumber: number) => void;
  onInjectToRefine?: (item: CWAIReviewItem) => void;
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
  const scope =
    mode as CWReviewWorkspaceScope;

  const rootRef =
    useRef<HTMLElement | null>(null);

  const state =
    useCWAIReviewUnifiedIssueListState({
      scope,
      items,
      rootRef,
      onSelectPage,
    });

  const isSelfReview =
    mode === "self";

  const workSelectedItemIDSet =
    useMemo(
      () =>
        new Set(
          workSelectedItemIds,
        ),
      [workSelectedItemIds],
    );

  const deliverySelectedItemIDSet =
    useMemo(
      () =>
        new Set(
          deliverySelectedItemIds,
        ),
      [deliverySelectedItemIds],
    );

  const activeRelations =
    useMemo(
      () =>
        governanceRelations.filter(
          (relation) =>
            relation.status ===
            "active",
        ),
      [governanceRelations],
    );

  const selectedWorkItems =
    useMemo(
      () =>
        items.filter(
          (item) =>
            workSelectedItemIDSet.has(
              item.id,
            ),
        ),
      [
        items,
        workSelectedItemIDSet,
      ],
    );

  if (items.length === 0) {
    return null;
  }

  const nextGroup =
    state.nextItem
      ? state.groups.find(
          (group) =>
            group.items.some(
              (item) =>
                item.id ===
                state.nextItem?.id,
            ),
        )
      : undefined;

  const workSelectedCount =
    selectedWorkItems.length;

  const canOpenDiscussion =
    workSelectedCount >= 2 &&
    workSelectedCount <= 12;

  const canOpenRelation =
    workSelectedCount === 2;

  const continueNext = () => {
    if (
      !state.nextItem ||
      !nextGroup
    ) {
      return;
    }

    if (
      !state.expandedGroupKeys.has(
        nextGroup.key,
      )
    ) {
      state.toggleGroup(
        nextGroup.key,
      );
    }

    state.handleFocusedItemChange(
      state.nextItem.id,
    );

    if (
      state.nextItem
        .page_number_snapshot > 0
    ) {
      onSelectPage(
        state.nextItem
          .page_number_snapshot,
      );
    }
  };

  const openItemWorkspace = (
    itemID: string,
  ) => {
    const target =
      items.find(
        (item) =>
          item.id === itemID,
      );

    if (!target) {
      return;
    }

    state.handleFocusedItemChange(
      itemID,
    );

    if (
      target.page_number_snapshot >
      0
    ) {
      onSelectPage(
        target.page_number_snapshot,
      );
    }
  };

  const workspaceItems =
    state.filteredItems.some(
      (item) =>
        item.id ===
        state.focusedItemID,
    )
      ? state.filteredItems
      : items;

  return (
    <section
      ref={rootRef}
      data-cw-review-shortcut-root="true"
      data-cw-review-scope={scope}
      style={{
        padding: "16px",
        borderRadius: "12px",
        border:
          `1px solid ${C.primary}35`,
        background: C.bg,
      }}
    >
      <CWAIReviewUnifiedIssueToolbar
        isSelfReview={
          isSelfReview
        }
        overview={
          state.overview
        }
        activeRelationCount={
          activeRelations.length
        }
        deliverySelectedCount={
          deliverySelectedItemIds.length
        }
        workSelectedCount={
          workSelectedCount
        }
        canOpenDiscussion={
          canOpenDiscussion
        }
        canOpenRelation={
          canOpenRelation
        }
        showCompleted={
          state.filters.showCompleted
        }
        hasNextItem={
          !!state.nextItem
        }
        onContinueNext={
          continueNext
        }
        onShowCompletedChange={(
          showCompleted: boolean,
        ) =>
          state.handleFiltersChange({
            ...state.filters,
            showCompleted,
            status:
              !showCompleted &&
              state.filters.status ===
                "completed"
                ? "all"
                : state.filters.status,
          })
        }
        onOpenGlobalDiscussion={
          onOpenGlobalDiscussion
        }
        onOpenDirectRelation={
          onOpenDirectRelation
        }
        onClearWorkSelection={
          onClearWorkSelection
        }
      />

      <CWReviewIssueFilters
        items={items}
        value={state.filters}
        resultCount={
          state.filteredItems.length
        }
        shareURL={
          state.shareURL
        }
        onChange={
          state.handleFiltersChange
        }
      />

      {state.focusedItemID ? (
        <CWReviewIssueWorkspace
          mode={scope}
          items={workspaceItems}
          currentItemId={
            state.focusedItemID
          }
          allItems={items}
          governanceRelations={
            activeRelations
          }
          selectable={
            !isSelfReview
          }
          selectedItemIds={
            deliverySelectedItemIds
          }
          onSelectedChange={
            onToggleDeliverySelection
          }
          onClose={() =>
            state.handleFocusedItemChange(
              "",
            )
          }
          onNavigateItem={
            state.handleFocusedItemChange
          }
          onSelectPage={
            onSelectPage
          }
          onItemChanged={
            onItemChanged
          }
          onInjectToRefine={
            onInjectToRefine
          }
        />
      ) : state.groups.length === 0 ? (
        <div
          style={{
            marginTop: "14px",
            padding: "28px 16px",
            borderRadius: "10px",
            border:
              `1px dashed ${C.border}`,
            background: "#FFFFFF",
            color: C.textSec,
            fontSize: "14px",
            lineHeight: 1.7,
            textAlign: "center",
          }}
        >
          当前筛选条件下没有问题。
          可以清除筛选或复制当前视图链接交给其他审核人员。
        </div>
      ) : (
        state.groups.map(
          (group) => (
            <CWAIReviewUnifiedIssueGroup
              key={group.key}
              group={group}
              visibleItems={
                group.items
              }
              expanded={
                state.expandedGroupKeys.has(
                  group.key,
                )
              }
              allItems={items}
              activeRelations={
                activeRelations
              }
              isSelfReview={
                isSelfReview
              }
              workSelectedItemIDSet={
                workSelectedItemIDSet
              }
              deliverySelectedItemIDSet={
                deliverySelectedItemIDSet
              }
              onToggleExpanded={() =>
                state.toggleGroup(
                  group.key,
                )
              }
              onOpenItem={
                openItemWorkspace
              }
              onToggleWorkSelection={
                onToggleWorkSelection
              }
              onToggleDeliverySelection={
                onToggleDeliverySelection
              }
              onItemChanged={
                onItemChanged
              }
              onSelectPage={
                onSelectPage
              }
              onInjectToRefine={
                onInjectToRefine
              }
            />
          ),
        )
      )}
    </section>
  );
}

export type {
  CWAIReviewUnifiedIssueOverview,
} from "./CWAIReviewUnifiedIssueList.shared";
