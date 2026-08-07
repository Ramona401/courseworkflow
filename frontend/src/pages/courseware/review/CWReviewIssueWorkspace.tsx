/**
 * CWReviewIssueWorkspace.tsx
 *
 * 课件评审单问题聚焦工作区。
 *
 * 支持：
 *   - 返回列表、上一条、下一条未完成问题；
 *   - 成功完成一个处理阶段后自动前进；
 *   - 当前问题深链接与页面定位；
 *   - Esc返回、Alt+←/→切换、Alt+L打开相关教案。
 *
 * 本组件只复用既有状态机和API，不自行决定问题状态。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  type CSSProperties,
  type MouseEvent,
} from "react";

import type {
  CWAIReviewItem,
  CWAIReviewItemRelation,
  CWAIReviewSeverity,
} from "@/api/coursewares";

import CWAIReviewItemPanel from "./CWAIReviewItemPanel";
import { openCoursewareReviewLessonPlanContext } from "./coursewareReviewLessonPlanContext";
import {
  isCWReviewTextEntryTarget,
  type CWReviewWorkspaceScope,
} from "./coursewareReviewWorkspaceState";
import {
  recordCoursewareReviewUsage,
  type CWReviewUsageEvent,
  type CWReviewUsageShortcut,
} from "./coursewareReviewUsage";

const C = {
  primary: "#4F7BE8",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  card: "#FFFFFF",
  bg: "#F8FAFC",
};

const SEVERITY_ORDER: Record<CWAIReviewSeverity, number> = {
  critical: 1,
  high: 2,
  medium: 3,
  low: 4,
  info: 5,
};

export interface CWReviewIssueWorkspaceProps {
  mode: CWReviewWorkspaceScope;
  items: CWAIReviewItem[];
  currentItemId: string;
  allItems?: CWAIReviewItem[];
  governanceRelations?: CWAIReviewItemRelation[];
  selectable?: boolean;
  selectedItemIds?: string[];
  onSelectedChange?: (itemID: string, selected: boolean) => void;
  onClose: () => void;
  onNavigateItem: (itemID: string) => void;
  onSelectPage: (pageNumber: number) => void;
  onItemChanged: (item: CWAIReviewItem) => void;
  onInjectToRefine?: (item: CWAIReviewItem) => void;
}

function isCompletedItem(item: CWAIReviewItem): boolean {
  return item.status === "resolved" || item.status === "dismissed";
}

function taskPriority(item: CWAIReviewItem): number {
  if (item.severity === "critical") return 0;
  if (item.severity === "high") return 10;
  if (item.status === "stale") return 20;
  if (item.status === "applying") return 30;
  if (item.status === "applied") return 35;
  return 40 + (SEVERITY_ORDER[item.severity] || 99);
}

function compareWorkspaceItems(
  left: CWAIReviewItem,
  right: CWAIReviewItem,
): number {
  const completedDifference =
    Number(isCompletedItem(left)) -
    Number(isCompletedItem(right));

  if (completedDifference !== 0) {
    return completedDifference;
  }

  const priorityDifference =
    taskPriority(left) -
    taskPriority(right);

  if (priorityDifference !== 0) {
    return priorityDifference;
  }

  const pageDifference =
    left.page_number_snapshot -
    right.page_number_snapshot;

  if (pageDifference !== 0) {
    return pageDifference;
  }

  return (left.created_at || "").localeCompare(right.created_at || "");
}

function shouldAdvanceAfterChange(
  previous: CWAIReviewItem,
  changed: CWAIReviewItem,
): boolean {
  if (previous.status === changed.status) {
    return false;
  }

  if (
    changed.status === "resolved" ||
    changed.status === "dismissed" ||
    changed.status === "applied"
  ) {
    return true;
  }

  return (
    changed.status === "confirmed" &&
    (
      previous.status === "detected" ||
      previous.status === "discussing"
    )
  );
}

export default function CWReviewIssueWorkspace({
  mode,
  items,
  currentItemId,
  allItems,
  governanceRelations,
  selectable = false,
  selectedItemIds = [],
  onSelectedChange,
  onClose,
  onNavigateItem,
  onSelectPage,
  onItemChanged,
  onInjectToRefine,
}: CWReviewIssueWorkspaceProps) {
  const orderedItems = useMemo(
    () => items.slice().sort(compareWorkspaceItems),
    [items],
  );

  const currentIndex =
    orderedItems.findIndex(
      (item) => item.id === currentItemId,
    );
  const currentItem =
    currentIndex >= 0
      ? orderedItems[currentIndex]
      : undefined;
  const previousItem =
    currentIndex > 0
      ? orderedItems[currentIndex - 1]
      : undefined;
  const nextItem =
    currentIndex >= 0 &&
    currentIndex < orderedItems.length - 1
      ? orderedItems[currentIndex + 1]
      : undefined;

  const nextPendingItem = useMemo(() => {
    if (!currentItem) {
      return undefined;
    }

    const afterCurrent =
      orderedItems
        .slice(currentIndex + 1)
        .find(
          (item) =>
            !isCompletedItem(item),
        );

    return (
      afterCurrent ||
      orderedItems.find(
        (item) =>
          item.id !== currentItem.id &&
          !isCompletedItem(item),
      )
    );
  }, [
    currentIndex,
    currentItem,
    orderedItems,
  ]);

  const selectedItemIDSet =
    useMemo(
      () =>
        new Set(
          selectedItemIds,
        ),
      [selectedItemIds],
    );

  const autoAdvanceKeyRef =
    useRef("");

  const totalItemCount =
    Math.max(
      allItems?.length || 0,
      items.length,
    );

  const visibleItemCount =
    items.length;

  const currentPageNumber =
    currentItem
      ?.page_number_snapshot ||
    0;

  const recordWorkspaceUsage =
    useCallback(
      (
        event:
          CWReviewUsageEvent,
        shortcut?:
          CWReviewUsageShortcut,
        pageNumber:
          number =
          currentPageNumber,
      ) => {
        recordCoursewareReviewUsage({
          event,
          mode,
          shortcut,
          totalCount:
            totalItemCount,
          visibleCount:
            visibleItemCount,
          pageNumber,
        });
      },
      [
        currentPageNumber,
        mode,
        totalItemCount,
        visibleItemCount,
      ],
    );

  const navigateToItem =
    useCallback(
      (
        target:
          CWAIReviewItem |
          undefined,
      ) => {
        if (!target) {
          return;
        }

        if (
          target.page_number_snapshot >
          0
        ) {
          onSelectPage(
            target.page_number_snapshot,
          );
        }

        onNavigateItem(target.id);
      },
      [
        onNavigateItem,
        onSelectPage,
      ],
    );

  const openRelatedLessonPlan =
    useCallback(
      (
        triggerElement?:
          HTMLElement |
          null,
      ) => {
        if (!currentItem) {
          return;
        }

        recordWorkspaceUsage(
          "lesson_plan_opened",
          undefined,
          currentItem.page_number_snapshot,
        );

        openCoursewareReviewLessonPlanContext({
          issueId: currentItem.id,
          pageNumber:
            currentItem.page_number_snapshot,
          pageTitle:
            currentItem.page_title_snapshot,
          issueTitle:
            currentItem.title,
          issueDescription:
            currentItem.description,
          confirmedInstruction:
            currentItem.confirmed_instruction,
          originalSuggestion:
            currentItem.original_suggestion,
          triggerElement:
            triggerElement ||
            undefined,
        });
      },
      [
        currentItem,
        recordWorkspaceUsage,
      ],
    );

  useEffect(() => {
    if (
      !currentItem ||
      currentItem.page_number_snapshot <=
        0
    ) {
      return;
    }

    onSelectPage(
      currentItem.page_number_snapshot,
    );
  }, [
    currentItem,
    onSelectPage,
  ]);

  useEffect(() => {
    const handleKeyDown = (
      event: KeyboardEvent,
    ) => {
      if (event.defaultPrevented) {
        return;
      }

      if (
        isCWReviewTextEntryTarget(
          event.target,
        )
      ) {
        return;
      }

      if (
        event.key === "Escape" &&
        !event.altKey &&
        !event.ctrlKey &&
        !event.metaKey
      ) {
        event.preventDefault();

        recordWorkspaceUsage(
          "keyboard_shortcut",
          "back_to_list",
        );

        onClose();
        return;
      }

      if (!event.altKey) {
        return;
      }

      if (
        event.key === "ArrowLeft" &&
        previousItem
      ) {
        event.preventDefault();

        recordWorkspaceUsage(
          "keyboard_shortcut",
          "previous_item",
          previousItem
            .page_number_snapshot,
        );

        navigateToItem(previousItem);
        return;
      }

      if (
        event.key === "ArrowRight" &&
        (nextPendingItem || nextItem)
      ) {
        event.preventDefault();

        const target =
          nextPendingItem ||
          nextItem;

        recordWorkspaceUsage(
          "keyboard_shortcut",
          "next_item",
          target
            ?.page_number_snapshot ||
            0,
        );

        navigateToItem(
          target,
        );
        return;
      }

      if (
        event.key.toLowerCase() === "l"
      ) {
        event.preventDefault();

        recordWorkspaceUsage(
          "keyboard_shortcut",
          "open_lesson_plan",
        );

        openRelatedLessonPlan();
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
    navigateToItem,
    nextItem,
    nextPendingItem,
    onClose,
    openRelatedLessonPlan,
    previousItem,
    recordWorkspaceUsage,
  ]);

  if (!currentItem) {
    return (
      <section
        style={{
          padding: "20px",
          borderRadius: "12px",
          border:
            `1px solid ${C.border}`,
          background: C.card,
        }}
      >
        <div
          style={{
            color: C.text,
            fontSize: "16px",
            fontWeight: 700,
          }}
        >
          当前问题已不在清单中
        </div>

        <div
          style={{
            marginTop: "6px",
            color: C.textSec,
            fontSize: "14px",
            lineHeight: 1.6,
          }}
        >
          问题数据可能已经刷新，请返回列表重新选择。
        </div>

        <button
          type="button"
          onClick={onClose}
          style={primaryButtonStyle}
        >
          返回问题列表
        </button>
      </section>
    );
  }

  const handleItemChanged = (
    changed: CWAIReviewItem,
  ) => {
    onItemChanged(changed);

    if (
      changed.id !== currentItem.id ||
      !shouldAdvanceAfterChange(
        currentItem,
        changed,
      ) ||
      !nextPendingItem
    ) {
      return;
    }

    const autoAdvanceKey = [
      currentItem.id,
      currentItem.status,
      changed.status,
      nextPendingItem.id,
    ].join(":");

    if (
      autoAdvanceKeyRef.current ===
      autoAdvanceKey
    ) {
      return;
    }

    autoAdvanceKeyRef.current =
      autoAdvanceKey;

    recordWorkspaceUsage(
      "auto_advanced",
      undefined,
      nextPendingItem
        .page_number_snapshot,
    );

    window.setTimeout(
      () =>
        navigateToItem(
          nextPendingItem,
        ),
      0,
    );
  };

  const pageLabel =
    currentItem.page_number_snapshot >
    0
      ? `P${currentItem.page_number_snapshot}`
      : "整课";

  return (
    <section
      aria-label="单问题聚焦工作区"
      style={{
        marginTop: "14px",
        borderRadius: "12px",
        border:
          `1px solid ${C.primary}35`,
        background: C.bg,
        overflow: "hidden",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "10px",
          flexWrap: "wrap",
          padding: "12px 14px",
          borderBottom:
            `1px solid ${C.border}`,
          background: C.card,
        }}
      >
        <button
          type="button"
          onClick={onClose}
          style={secondaryButtonStyle}
        >
          ← 返回问题列表
        </button>

        <div
          style={{
            minWidth: 0,
            flex: 1,
          }}
        >
          <div
            style={{
              color: C.text,
              fontSize: "18px",
              fontWeight: 700,
              lineHeight: 1.4,
            }}
          >
            单问题处理工作区
          </div>

          <div
            style={{
              marginTop: "3px",
              color: C.textMuted,
              fontSize: "12px",
              lineHeight: 1.5,
            }}
          >
            第 {currentIndex + 1} / {orderedItems.length} 条 · {pageLabel}
            {currentItem.page_title_snapshot
              ? ` · ${currentItem.page_title_snapshot}`
              : ""}
          </div>
        </div>

        {currentItem.page_number_snapshot > 0 && (
          <button
            type="button"
            onClick={() =>
              onSelectPage(
                currentItem.page_number_snapshot,
              )
            }
            style={{
              ...secondaryButtonStyle,
              border:
                `1px solid ${C.primary}`,
              color: C.primary,
            }}
          >
            打开当前页面
          </button>
        )}

        <button
          type="button"
          onClick={(event: MouseEvent<HTMLButtonElement>) =>
            openRelatedLessonPlan(
              event.currentTarget,
            )
          }
          style={{
            ...secondaryButtonStyle,
            border:
              "1px solid #C4B5FD",
            color: "#6D28D9",
          }}
        >
          查看相关教案
        </button>
      </div>

      <div
        style={{
          padding:
            "6px 14px 14px",
        }}
      >
        <CWAIReviewItemPanel
          key={currentItem.id}
          item={currentItem}
          allItems={
            allItems || items
          }
          governanceRelations={
            governanceRelations
          }
          selectable={selectable}
          selected={
            selectedItemIDSet.has(
              currentItem.id,
            )
          }
          onSelectedChange={
            onSelectedChange
          }
          onSelectPage={
            onSelectPage
          }
          onChanged={
            handleItemChanged
          }
          forceDetailsOpen
          onInjectToRefine={
            onInjectToRefine
          }
        />
      </div>

      <div
        style={{
          position: "sticky",
          bottom: 0,
          zIndex: 4,
          display: "flex",
          alignItems: "center",
          gap: "8px",
          flexWrap: "wrap",
          padding: "10px 14px",
          borderTop:
            `1px solid ${C.border}`,
          background:
            "rgba(255,255,255,0.97)",
          boxShadow:
            "0 -8px 22px rgba(15,23,42,0.08)",
        }}
      >
        <button
          type="button"
          onClick={() =>
            navigateToItem(
              previousItem,
            )
          }
          disabled={!previousItem}
          style={navigationButtonStyle(
            !!previousItem,
            "secondary",
          )}
        >
          上一条
        </button>

        <button
          type="button"
          onClick={onClose}
          style={navigationButtonStyle(
            true,
            "secondary",
          )}
        >
          返回列表
        </button>

        <div
          style={{
            minWidth: "220px",
            flex: 1,
            color: C.textMuted,
            fontSize: "12px",
            lineHeight: 1.5,
            textAlign: "center",
          }}
        >
          Esc 返回 · Alt+←/→ 切换 · Alt+L 查看教案；保存完成后自动进入下一条未完成问题。
        </div>

        <button
          type="button"
          onClick={() =>
            navigateToItem(
              nextPendingItem ||
              nextItem,
            )
          }
          disabled={
            !nextPendingItem &&
            !nextItem
          }
          style={navigationButtonStyle(
            !!nextPendingItem ||
              !!nextItem,
            "primary",
          )}
        >
          {nextPendingItem
            ? "下一条未完成"
            : "下一条"}
        </button>
      </div>
    </section>
  );
}

const primaryButtonStyle: CSSProperties = {
  minHeight: "36px",
  marginTop: "14px",
  padding: "8px 14px",
  borderRadius: "8px",
  border: "none",
  background: C.primary,
  color: "#FFFFFF",
  fontSize: "13px",
  fontWeight: 700,
  cursor: "pointer",
};

const secondaryButtonStyle: CSSProperties = {
  minHeight: "36px",
  padding: "8px 12px",
  borderRadius: "8px",
  border: `1px solid ${C.border}`,
  background: "#FFFFFF",
  color: C.textSec,
  fontSize: "13px",
  fontWeight: 700,
  cursor: "pointer",
};

function navigationButtonStyle(
  enabled: boolean,
  tone: "primary" | "secondary",
): CSSProperties {
  return {
    minHeight: "36px",
    padding: "8px 12px",
    borderRadius: "8px",
    border:
      tone === "primary"
        ? `1px solid ${
            enabled
              ? C.primary
              : C.border
          }`
        : `1px solid ${C.border}`,
    background:
      !enabled
        ? "#F1F5F9"
        : tone === "primary"
          ? C.primary
          : "#FFFFFF",
    color:
      !enabled
        ? C.textMuted
        : tone === "primary"
          ? "#FFFFFF"
          : C.textSec,
    fontSize: "13px",
    fontWeight: 700,
    cursor:
      enabled
        ? "pointer"
        : "not-allowed",
  };
}
