/**
 * CWAIReviewUnifiedIssueGroup.tsx
 *
 * 统一问题工作台中的整课或单页问题分组。
 *
 * 页面分组默认折叠，只有当前任务所在分组主动展开。分组标题整行可点击，
 * 并集中展示问题数量、未完成数量、待确认数量和页面变化数量。
 *
 * “选择一起分析”只决定哪些问题需要放在一起比较或讨论；
 * 问题卡片中的“本次退回给作者”才决定最终会发给作者哪些内容。
 */

import type {
  CWAIReviewItem,
  CWAIReviewItemRelation,
} from "@/api/coursewares";

import { isCWGlobalDiscussionActionableItem } from "./CWAIReviewGlobalDiscussion.shared";
import CWAIReviewItemPanel from "./CWAIReviewItemPanel";

const C = {
  primary: "#4F7BE8",
  primarySoft: "#EEF2FF",
  warning: "#D97706",
  danger: "#DC2626",
  success: "#059669",
  purple: "#7C3AED",
  purpleSoft: "#F5F3FF",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  card: "#FFFFFF",
};

export interface CWAIReviewUnifiedIssueGroupData {
  key: string;
  pageNumber: number;
  pageTitle: string;
  items: CWAIReviewItem[];
}

export interface CWAIReviewUnifiedIssueGroupProps {
  group: CWAIReviewUnifiedIssueGroupData;
  visibleItems: CWAIReviewItem[];
  expanded: boolean;
  allItems: CWAIReviewItem[];
  activeRelations: CWAIReviewItemRelation[];
  isSelfReview: boolean;
  workSelectedItemIDSet: ReadonlySet<string>;
  deliverySelectedItemIDSet: ReadonlySet<string>;

  onToggleExpanded: () => void;
  onOpenItem: (itemID: string) => void;
  onToggleWorkSelection: (itemID: string, selected: boolean) => void;
  onToggleDeliverySelection: (itemID: string, selected: boolean) => void;
  onItemChanged: (item: CWAIReviewItem) => void;
  onSelectPage: (pageNumber: number) => void;
  onInjectToRefine?: (item: CWAIReviewItem) => void;
}

function isCompletedItem(item: CWAIReviewItem): boolean {
  return item.status === "resolved" || item.status === "dismissed";
}

export default function CWAIReviewUnifiedIssueGroup({
  group,
  visibleItems,
  expanded,
  allItems,
  activeRelations,
  isSelfReview,
  workSelectedItemIDSet,
  deliverySelectedItemIDSet,
  onToggleExpanded,
  onOpenItem,
  onToggleWorkSelection,
  onToggleDeliverySelection,
  onItemChanged,
  onSelectPage,
  onInjectToRefine,
}: CWAIReviewUnifiedIssueGroupProps) {
  const groupItemIDSet = new Set(group.items.map((item) => item.id));

  const groupRelations = activeRelations.filter(
    (relation) =>
      groupItemIDSet.has(relation.source_item_id) ||
      groupItemIDSet.has(relation.target_item_id),
  );

  const togetherSelectedCount = group.items.filter((item) =>
    workSelectedItemIDSet.has(item.id),
  ).length;

  const deliverySelectedCount = group.items.filter((item) =>
    deliverySelectedItemIDSet.has(item.id),
  ).length;

  const pendingCount = group.items.filter((item) => !isCompletedItem(item)).length;
  const waitingConfirmCount = group.items.filter((item) => item.status === "applied").length;
  const staleCount = group.items.filter((item) => item.status === "stale").length;
  const completedCount = group.items.filter(isCompletedItem).length;

  return (
    <section
      style={{
        marginTop: "12px",
        borderRadius: "10px",
        border: `1px solid ${C.border}`,
        background: C.card,
        overflow: "hidden",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "stretch",
          gap: "8px",
          padding: "10px 12px",
          background: pendingCount > 0 ? "#FFFFFF" : "#F8FAFC",
        }}
      >
        <button
          type="button"
          onClick={onToggleExpanded}
          aria-expanded={expanded}
          style={{
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
          }}
        >
          <span
            aria-hidden="true"
            style={{
              flexShrink: 0,
              color: group.pageNumber > 0 ? C.primary : C.purple,
              fontSize: "16px",
              transform: expanded ? "rotate(90deg)" : "rotate(0deg)",
              transition: "transform 160ms ease",
            }}
          >
            ▶
          </span>

          <span style={{ minWidth: 0, flex: 1 }}>
            <span
              style={{
                display: "block",
                overflow: "hidden",
                color: C.text,
                fontSize: "15px",
                fontWeight: 700,
                lineHeight: 1.5,
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
              }}
            >
              {group.pageNumber > 0 ? `P${group.pageNumber} ` : "整课 "}
              {group.pageTitle ||
                (group.pageNumber > 0
                  ? `第${group.pageNumber}页`
                  : "整课综合问题")}
            </span>

            <span
              style={{
                display: "block",
                marginTop: "3px",
                color: C.textMuted,
                fontSize: "12px",
                lineHeight: 1.5,
              }}
            >
              {group.items.length} 条问题
              {pendingCount > 0 ? ` · ${pendingCount} 条未完成` : ""}
              {waitingConfirmCount > 0 ? ` · ${waitingConfirmCount} 条待确认` : ""}
              {staleCount > 0 ? ` · ${staleCount} 条页面变化` : ""}
              {completedCount > 0 ? ` · ${completedCount} 条已完成` : ""}
            </span>
          </span>
        </button>

        {group.pageNumber > 0 && (
          <button
            type="button"
            onClick={() => onSelectPage(group.pageNumber)}
            style={{
              flexShrink: 0,
              minHeight: "36px",
              padding: "8px 10px",
              borderRadius: "8px",
              border: `1px solid ${C.primary}`,
              background: C.primarySoft,
              color: C.primary,
              fontSize: "12px",
              fontWeight: 700,
              cursor: "pointer",
            }}
          >
            打开页面
          </button>
        )}
      </div>

      {(groupRelations.length > 0 ||
        togetherSelectedCount > 0 ||
        (!isSelfReview && deliverySelectedCount > 0)) && (
        <div
          style={{
            display: "flex",
            gap: "12px",
            flexWrap: "wrap",
            padding: "0 12px 10px",
            color: C.textMuted,
            fontSize: "12px",
            lineHeight: 1.5,
          }}
        >
          {groupRelations.length > 0 && (
            <span style={{ color: C.purple }}>关联 {groupRelations.length}</span>
          )}

          {togetherSelectedCount > 0 && (
            <span style={{ color: C.primary }}>
              已选 {togetherSelectedCount} 条一起分析
            </span>
          )}

          {!isSelfReview && deliverySelectedCount > 0 && (
            <span style={{ color: C.warning }}>本次退回 {deliverySelectedCount}</span>
          )}
        </div>
      )}

      {expanded && (
        <div
          style={{
            padding: "4px 12px 12px",
            borderTop: `1px solid ${C.border}`,
          }}
        >
          {visibleItems.map((item) => {
            const canAnalyzeTogether = isCWGlobalDiscussionActionableItem(item);
            const selectedTogether = workSelectedItemIDSet.has(item.id);

            return (
              <div
                key={item.id}
                style={{
                  marginTop: "10px",
                  padding: "8px",
                  borderRadius: "9px",
                  border: selectedTogether
                    ? "1px solid #93C5FD"
                    : "1px solid transparent",
                  background: selectedTogether ? "#F8FAFF" : "transparent",
                }}
              >
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "8px",
                    flexWrap: "wrap",
                    padding: "0 2px 2px",
                  }}
                >
                  <button
                    type="button"
                    onClick={() => onOpenItem(item.id)}
                    style={{
                      minHeight: "32px",
                      padding: "6px 10px",
                      borderRadius: "8px",
                      border: `1px solid ${C.primary}`,
                      background: C.primary,
                      color: "#FFFFFF",
                      fontSize: "12px",
                      fontWeight: 700,
                      cursor: "pointer",
                    }}
                  >
                    处理这条
                  </button>

                  <button
                    type="button"
                    aria-pressed={selectedTogether}
                    disabled={!canAnalyzeTogether}
                    onClick={() => onToggleWorkSelection(item.id, !selectedTogether)}
                    title={
                      canAnalyzeTogether
                        ? "选择这条问题，与其他问题放在一起比较或讨论"
                        : "这条问题目前不能加入新的综合分析"
                    }
                    style={{
                      minHeight: "32px",
                      padding: "6px 9px",
                      borderRadius: "999px",
                      border: selectedTogether
                        ? "1px solid #60A5FA"
                        : "1px solid #CBD5E1",
                      background: selectedTogether ? "#DBEAFE" : "#FFFFFF",
                      color: selectedTogether
                        ? "#1D4ED8"
                        : canAnalyzeTogether
                          ? C.primary
                          : C.textMuted,
                      fontSize: "12px",
                      fontWeight: 700,
                      cursor: canAnalyzeTogether ? "pointer" : "not-allowed",
                      opacity: canAnalyzeTogether ? 1 : 0.6,
                    }}
                  >
                    {selectedTogether ? "✓ 已选一起分析" : "选择一起分析"}
                  </button>

                  <span
                    style={{
                      color: C.textMuted,
                      fontSize: "12px",
                      lineHeight: 1.5,
                    }}
                  >
                    不会改变本次退回内容
                  </span>
                </div>

                <CWAIReviewItemPanel
                  item={item}
                  allItems={allItems}
                  governanceRelations={activeRelations}
                  selectable={!isSelfReview}
                  selected={deliverySelectedItemIDSet.has(item.id)}
                  onSelectedChange={onToggleDeliverySelection}
                  onSelectPage={onSelectPage}
                  onChanged={onItemChanged}
                  onInjectToRefine={onInjectToRefine}
                />
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
