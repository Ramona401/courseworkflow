/**
 * CWReviewDeliveryPreview.tsx
 *
 * R-05“本次修改清单”独立预览组件。
 *
 * 职责严格限定为：
 *   - 展示当前正式交付选择；
 *   - 按整课或稳定page_id分组；
 *   - 展开全部；
 *   - 打开对应课件页；
 *   - 回到原问题工作位置；
 *   - 从当前delivery draft移除。
 *
 * R-08上线后，审核意见重新整理统一由CWReviewCommentCandidatePanel负责。
 * 本组件不再本地拼接或同步审核意见文本。
 *
 * 本组件绝不修改整改项状态，不调用dismiss/resolve，不修改页面，也不调用AI。
 */

import {
  useMemo,
  useState,
  type CSSProperties,
} from "react";

import type { CWAIReviewItem } from "@/api/coursewares";

import type { CWAIReviewContext } from "./CWAIReviewPanel";
import {
  createDefaultCWReviewFilters,
  replaceCWReviewWorkspaceURL,
} from "./coursewareReviewWorkspaceState";

const C = {
  danger: "#EF4444",
  primary: "#4F7BE8",
  text: "#1F2937",
  textSec: "#6B7280",
  textMuted: "#9CA3AF",
  borderMid: "#E5E7EB",
};

interface DeliveryGroup {
  key: string;
  pageNumber: number;
  pageTitle: string;
  items: CWAIReviewItem[];
}

export interface CWReviewDeliveryPreviewProps {
  decision: "approved" | "revision";
  aiReviewContext: CWAIReviewContext;
  onSelectPage: (pageNumber: number) => void;
  onOpenAI: () => void;
}

function buildDeliveryGroups(items: CWAIReviewItem[]): DeliveryGroup[] {
  const groups = new Map<string, DeliveryGroup>();

  for (const item of items) {
    const pageNumber = item.page_number_snapshot;
    const stablePageID = item.page_id?.trim() || "";
    const key =
      stablePageID ||
      (pageNumber > 0
        ? `snapshot:${pageNumber}:${item.page_title_snapshot}`
        : "global");

    const current = groups.get(key) || {
      key,
      pageNumber,
      pageTitle: item.page_title_snapshot || "",
      items: [],
    };

    current.items.push(item);

    if (!current.pageTitle && item.page_title_snapshot) {
      current.pageTitle = item.page_title_snapshot;
    }

    groups.set(key, current);
  }

  return Array.from(groups.values()).sort((left, right) => {
    if (left.pageNumber <= 0 && right.pageNumber > 0) {
      return -1;
    }

    if (right.pageNumber <= 0 && left.pageNumber > 0) {
      return 1;
    }

    return left.pageNumber - right.pageNumber;
  });
}

function resolveTeacherTitle(item: CWAIReviewItem): string {
  return item.teacher_title?.trim() || "已确认整改要求";
}

function resolveTeachingImpact(item: CWAIReviewItem): string {
  return item.teaching_impact?.trim() || "未提供课堂影响说明";
}

export default function CWReviewDeliveryPreview({
  decision,
  aiReviewContext,
  onSelectPage,
  onOpenAI,
}: CWReviewDeliveryPreviewProps) {
  const [expanded, setExpanded] = useState(true);

  /**
   * 通过审核不会携带本轮新问题，因此正式“本次修改清单”必须显示为0。
   * AI问题工作区里的选择草稿仍保留，切回退回修改后立即恢复预览。
   */
  const deliveryItems =
    decision === "revision" ? aiReviewContext.selectedItems : [];

  const deliveryGroups = useMemo(
    () => buildDeliveryGroups(deliveryItems),
    [deliveryItems],
  );

  const deliveryItemCount = deliveryItems.length;
  const deliveryPageCount = deliveryGroups.length;

  const deliveryCountMismatch =
    decision === "revision" &&
    aiReviewContext.selectedItemIds.length !==
      aiReviewContext.selectedItems.length;

  const removeSelectedItem = (itemID: string) => {
    aiReviewContext.removeSelectedItem?.(itemID);
  };

  const openOriginalIssue = (item: CWAIReviewItem) => {
    replaceCWReviewWorkspaceURL(
      "formal",
      createDefaultCWReviewFilters(),
      item.id,
    );

    if (typeof window !== "undefined") {
      window.dispatchEvent(new PopStateEvent("popstate"));
    }

    if (item.page_number_snapshot > 0) {
      onSelectPage(item.page_number_snapshot);
    }

    onOpenAI();
  };

  return (
    <div
      id="cw-review-delivery-preview"
      style={{
        marginBottom: "12px",
        padding: "10px 11px",
        borderRadius: "9px",
        border:
          deliveryItemCount > 0
            ? "1px solid #FED7AA"
            : `1px solid ${C.borderMid}`,
        background:
          deliveryItemCount > 0 ? "#FFF7ED" : "#FFFFFF",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: "8px",
        }}
      >
        <div style={{ minWidth: 0, flex: 1 }}>
          <div
            style={{
              color:
                deliveryItemCount > 0 ? "#9A3412" : C.textSec,
              fontSize: "12px",
              fontWeight: 700,
              lineHeight: 1.6,
            }}
          >
            📋 本次修改清单 {deliveryItemCount} 条
          </div>

          <div
            style={{
              marginTop: "3px",
              color: C.textMuted,
              fontSize: "10px",
              lineHeight: 1.55,
            }}
          >
            {decision === "revision"
              ? deliveryItemCount > 0
                ? `将正式交付 ${deliveryPageCount} 个页面或整课范围，共 ${deliveryItemCount} 条修改要求。`
                : "当前退回决定还没有可正式交付的确认问题。"
              : "当前决定为通过，正式提交时交付数量为 0；切换为退回修改后这里会显示实际交付清单。"}
          </div>

          <div
            style={{
              marginTop: "3px",
              color: C.textMuted,
              fontSize: "9px",
              lineHeight: 1.5,
            }}
          >
            从本清单移出只影响本次退回，不会删除、忽略、解决或改写原整改项。
          </div>
        </div>

        {deliveryItemCount > 0 && (
          <button
            type="button"
            onClick={() => setExpanded((previous) => !previous)}
            style={smallButtonStyle}
          >
            {expanded ? "收起清单" : "展开全部"}
          </button>
        )}
      </div>

      {deliveryCountMismatch && (
        <div
          style={{
            marginTop: "8px",
            padding: "8px 9px",
            borderRadius: "7px",
            border: "1px solid #FECACA",
            background: "#FEF2F2",
            color: C.danger,
            fontSize: "10px",
            lineHeight: 1.55,
          }}
        >
          本次修改清单数量与提交ID数量不一致，已阻止提交。请刷新审核页面后重新确认。
        </div>
      )}

      {decision === "revision" && deliveryItemCount === 0 && (
        <button
          type="button"
          onClick={onOpenAI}
          style={{
            ...smallButtonStyle,
            marginTop: "8px",
            color: C.primary,
            border: `1px solid ${C.primary}`,
          }}
        >
          返回AI审核确认整改项
        </button>
      )}

      {expanded && deliveryGroups.length > 0 && (
        <div
          style={{
            maxHeight: "360px",
            overflowY: "auto",
            marginTop: "9px",
            paddingRight: "2px",
          }}
        >
          {deliveryGroups.map((group) => (
            <div
              key={group.key}
              style={{
                marginTop: "7px",
                padding: "8px",
                borderRadius: "7px",
                border: "1px solid #FED7AA",
                background: "#FFFFFF",
              }}
            >
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "6px",
                  flexWrap: "wrap",
                }}
              >
                <span
                  style={{
                    padding: "3px 7px",
                    borderRadius: "6px",
                    background: "#FFF7ED",
                    color: "#C2410C",
                    fontSize: "10px",
                    fontWeight: 700,
                  }}
                >
                  {group.pageNumber > 0
                    ? `P${group.pageNumber}`
                    : "整课"}
                </span>

                <span
                  style={{
                    minWidth: 0,
                    flex: 1,
                    color: C.text,
                    fontSize: "10px",
                    fontWeight: 700,
                  }}
                >
                  {group.pageTitle ||
                    (group.pageNumber > 0
                      ? `第${group.pageNumber}页`
                      : "整课综合问题")}
                </span>

                <span
                  style={{
                    color: C.textMuted,
                    fontSize: "9px",
                  }}
                >
                  {group.items.length} 条
                </span>

                {group.pageNumber > 0 && (
                  <button
                    type="button"
                    onClick={() => onSelectPage(group.pageNumber)}
                    style={{
                      ...smallButtonStyle,
                      color: "#C2410C",
                      border: "1px solid #FDBA74",
                    }}
                  >
                    打开这一页
                  </button>
                )}
              </div>

              {group.items.map((item) => (
                <div
                  key={item.id}
                  style={{
                    marginTop: "8px",
                    paddingTop: "8px",
                    borderTop: "1px solid #FDE7D3",
                  }}
                >
                  <div
                    style={{
                      color: C.text,
                      fontSize: "11px",
                      fontWeight: 700,
                      lineHeight: 1.5,
                    }}
                  >
                    {resolveTeacherTitle(item)}
                  </div>

                  <div style={deliveryFieldStyle}>
                    <strong>课堂影响：</strong>
                    {resolveTeachingImpact(item)}
                  </div>

                  <div style={deliveryFieldStyle}>
                    <strong>当前修改要求：</strong>
                    {item.confirmed_instruction}
                  </div>

                  <div
                    style={{
                      display: "flex",
                      gap: "6px",
                      flexWrap: "wrap",
                      marginTop: "7px",
                    }}
                  >
                    <button
                      type="button"
                      onClick={() => openOriginalIssue(item)}
                      style={{
                        ...smallButtonStyle,
                        color: C.primary,
                        border: `1px solid ${C.primary}`,
                      }}
                    >
                      回到原问题
                    </button>

                    <button
                      type="button"
                      onClick={() => removeSelectedItem(item.id)}
                      disabled={!aiReviewContext.removeSelectedItem}
                      title="仅从本次正式退回清单移出"
                      style={{
                        ...smallButtonStyle,
                        color: C.danger,
                        border: "1px solid #FECACA",
                        cursor: aiReviewContext.removeSelectedItem
                          ? "pointer"
                          : "not-allowed",
                      }}
                    >
                      从本次清单移除
                    </button>
                  </div>
                </div>
              ))}
            </div>
          ))}
        </div>
      )}

      {decision === "revision" && deliveryItemCount > 0 && (
        <div
          style={{
            marginTop: "9px",
            paddingTop: "8px",
            borderTop: "1px solid #FED7AA",
            color: C.textMuted,
            fontSize: "9px",
            lineHeight: 1.5,
          }}
        >
          审核意见请在下方人工编辑；如需根据当前清单重新整理，请使用“审核意见重新整理”。
        </div>
      )}
    </div>
  );
}

const smallButtonStyle: CSSProperties = {
  padding: "3px 7px",
  borderRadius: "6px",
  border: `1px solid ${C.borderMid}`,
  background: "#fff",
  color: C.textSec,
  fontSize: "10px",
  fontWeight: 600,
  cursor: "pointer",
};

const deliveryFieldStyle: CSSProperties = {
  marginTop: "5px",
  color: C.textSec,
  fontSize: "10px",
  lineHeight: 1.6,
  whiteSpace: "pre-wrap",
  wordBreak: "break-word",
};
