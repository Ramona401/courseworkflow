/**
 * CWReviewDecisionPanel.tsx
 *
 * 课件正式审核决定、退回清单和审核意见面板。
 *
 * 草稿同步规则：
 *   1. 审核员点击按钮后，才首次用当前退回清单生成审核意见；
 *   2. 如果审核意见仍与上次系统草稿完全一致，清单变化时可安全同步；
 *   3. 一旦审核员手工编辑，后续清单变化不再自动覆盖；
 *   4. 审核员可显式点击“更新审核意见草稿”重新采用最新清单；
 *   5. 从总览移出只影响本次正式交付选择，不删除或忽略整改项。
 */

import {
  useEffect,
  useMemo,
  useState,
  type CSSProperties,
} from "react";

import type { CWAIReviewItem } from "@/api/coursewares";

import type { CWAIReviewContext } from "./CWAIReviewPanel";
import {
  requestCWReviewCarryoverFocus,
  useCWReviewApprovalGuard,
} from "./CWReviewSubmissionGuards";

const C = {
  success: "#10B981",
  warning: "#F59E0B",
  danger: "#EF4444",
  primary: "#4F7BE8",
  text: "#1F2937",
  textSec: "#6B7280",
  textMuted: "#9CA3AF",
  borderMid: "#E5E7EB",
  bg: "#FAFBFC",
};

interface DeliveryGroup {
  key: string;
  pageNumber: number;
  pageTitle: string;
  items: CWAIReviewItem[];
}

export interface CWReviewDecisionPanelProps {
  level: number;
  decision: "approved" | "revision";
  score: string;
  comment: string;
  submitting: boolean;
  aiReviewContext: CWAIReviewContext;

  onDecisionChange: (decision: "approved" | "revision") => void;
  onScoreChange: (score: string) => void;
  onCommentChange: (comment: string) => void;
  onSelectPage: (pageNumber: number) => void;
  onOpenAI: () => void;
  onCancel: () => void;
  onSubmit: () => void;
}

function buildDeliveryGroups(
  items: CWAIReviewItem[],
): DeliveryGroup[] {
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

function buildDeliveryCommentDraft(
  groups: DeliveryGroup[],
): string {
  const itemCount = groups.reduce(
    (total, group) => total + group.items.length,
    0,
  );

  if (itemCount === 0) {
    return "";
  }

  const lines = [
    `经审核，本课件需退回修改。本次共确认 ${groups.length} 个页面或整课范围、${itemCount} 条整改要求：`,
    "",
  ];

  groups.forEach((group, groupIndex) => {
    const pageLabel =
      group.pageNumber > 0 ? `P${group.pageNumber}` : "整课";

    const titleSuffix = group.pageTitle
      ? `（${group.pageTitle}）`
      : "";

    lines.push(`${groupIndex + 1}. ${pageLabel}${titleSuffix}`);

    group.items.forEach((item, itemIndex) => {
      const title = item.title.trim();
      const instruction = item.confirmed_instruction.trim();

      lines.push(
        `   ${itemIndex + 1}) ${
          title ? `${title}：` : ""
        }${instruction}`,
      );
    });

    lines.push("");
  });

  lines.push(
    "请在修改后逐项复核页面内容、交互运行效果及与教案目标的一致性，再重新提交审核。",
  );

  return lines.join("\n").trim();
}

export default function CWReviewDecisionPanel({
  level,
  decision,
  score,
  comment,
  submitting,
  aiReviewContext,
  onDecisionChange,
  onScoreChange,
  onCommentChange,
  onSelectPage,
  onOpenAI,
  onCancel,
  onSubmit,
}: CWReviewDecisionPanelProps) {
  const [expanded, setExpanded] = useState(true);
  const [lastAppliedDraft, setLastAppliedDraft] = useState("");

  const approvalGuard =
    useCWReviewApprovalGuard();

  const approvalBlocked =
    approvalGuard.blockedCount > 0;

  const deliveryGroups = useMemo(
    () => buildDeliveryGroups(aiReviewContext.selectedItems),
    [aiReviewContext.selectedItems],
  );

  const currentDeliveryDraft = useMemo(
    () => buildDeliveryCommentDraft(deliveryGroups),
    [deliveryGroups],
  );

  const deliveryItemCount =
    aiReviewContext.selectedItems.length;

  const deliveryPageCount =
    deliveryGroups.length;

  useEffect(() => {
    if (
      approvalBlocked &&
      decision === "approved"
    ) {
      onDecisionChange(
        "revision",
      );
    }
  }, [
    approvalBlocked,
    decision,
    onDecisionChange,
  ]);

  /**
   * 只有当前审核意见仍等于上次系统草稿时，才随清单变化自动同步。
   *
   * comment一旦被人工编辑，与lastAppliedDraft不再相等，
   * 本effect便不会覆盖人工内容。
   */
  useEffect(() => {
    if (!lastAppliedDraft || comment !== lastAppliedDraft) {
      return;
    }
    if (currentDeliveryDraft === lastAppliedDraft) {
      return;
    }

    onCommentChange(currentDeliveryDraft);
    setLastAppliedDraft(currentDeliveryDraft);
  }, [
    comment,
    currentDeliveryDraft,
    lastAppliedDraft,
    onCommentChange,
  ]);

  const draftSynced =
    !!currentDeliveryDraft &&
    comment === currentDeliveryDraft &&
    lastAppliedDraft === currentDeliveryDraft;

  const draftNeedsManualRefresh =
    !!lastAppliedDraft &&
    currentDeliveryDraft !== lastAppliedDraft &&
    comment !== lastAppliedDraft;

  const applyCurrentDraft = () => {
    if (!currentDeliveryDraft) {
      return;
    }

    onCommentChange(currentDeliveryDraft);
    setLastAppliedDraft(currentDeliveryDraft);
  };

  const removeSelectedItem = (itemID: string) => {
    aiReviewContext.removeSelectedItem?.(itemID);
  };

  const submitBlocked =
    submitting ||
    (
      decision === "approved" &&
      approvalBlocked
    );

  const handleSubmit = () => {
    if (submitBlocked) {
      return;
    }

    onSubmit();
  };

  return (
    <div
      style={{
        flexShrink: 0,
        padding: "16px",
        background: C.bg,
      }}
    >
      <div
        style={{
          display: "flex",
          gap: "10px",
          marginBottom: "12px",
        }}
      >
        {[
          {
            value: "approved" as const,
            label: "✅ 通过",
            color: C.success,
            disabled:
              approvalBlocked,
          },
          {
            value: "revision" as const,
            label: "↩️ 退回修改",
            color: C.warning,
            disabled: false,
          },
        ].map((option) => (
          <button
            key={option.value}
            type="button"
            disabled={
              option.disabled
            }
            title={
              option.disabled
                ? `还有${approvalGuard.blockedCount}条上轮问题未确认解决`
                : undefined
            }
            onClick={() => {
              if (!option.disabled) {
                onDecisionChange(
                  option.value,
                );
              }
            }}
            style={{
              flex: 1,
              padding: "10px",
              borderRadius: "10px",
              border:
                decision === option.value
                  ? `2px solid ${option.color}`
                  : `1px solid ${C.borderMid}`,
              background:
                decision === option.value
                  ? `${option.color}10`
                  : "#fff",
              cursor:
                option.disabled
                  ? "not-allowed"
                  : "pointer",
              opacity:
                option.disabled
                  ? 0.55
                  : 1,
              fontSize: "14px",
              fontWeight:
                decision === option.value ? 600 : 400,
              color:
                decision === option.value
                  ? option.color
                  : C.textSec,
            }}
          >
            {option.label}
          </button>
        ))}
      </div>

      {approvalGuard.totalCount >
        0 && (
        <div
          style={{
            marginBottom: "12px",
            padding: "11px 12px",
            borderRadius: "9px",
            border:
              approvalBlocked
                ? "1px solid #FECACA"
                : "1px solid #A7F3D0",
            background:
              approvalBlocked
                ? "#FEF2F2"
                : "#ECFDF5",
          }}
        >
          <div
            style={{
              color:
                approvalBlocked
                  ? C.danger
                  : C.success,
              fontSize: "13px",
              fontWeight: 700,
              lineHeight: 1.55,
            }}
          >
            {approvalBlocked
              ? `暂时不能审核通过：还有 ${approvalGuard.blockedCount} 条上轮问题未确认解决`
              : `上轮 ${approvalGuard.totalCount} 条问题均已确认解决，可以审核通过`}
          </div>

          <div
            style={{
              marginTop: "4px",
              color: C.textSec,
              fontSize: "12px",
              lineHeight: 1.6,
            }}
          >
            已确认解决
            {" "}
            {approvalGuard.resolvedCount}
            {" "}
            条
            {approvalGuard.notReadyCount >
              0
              ? `；${approvalGuard.notReadyCount} 条作者尚未完成或状态异常`
              : ""}
            {approvalGuard.waitingReviewCount >
              0
              ? `；${approvalGuard.waitingReviewCount} 条已完成修改但尚未勾选确认`
              : ""}
            {approvalGuard.changedPageCount >
              0
              ? `；其中 ${approvalGuard.changedPageCount} 条涉及页面变化`
              : ""}
            。
            继续退回不受此限制。
          </div>

          {approvalBlocked && (
            <button
              type="button"
              onClick={
                requestCWReviewCarryoverFocus
              }
              style={{
                marginTop: "9px",
                minHeight: "36px",
                padding: "8px 11px",
                borderRadius: "8px",
                border:
                  `1px solid ${C.danger}`,
                background: "#FFFFFF",
                color: C.danger,
                fontSize: "12px",
                fontWeight: 700,
                cursor: "pointer",
              }}
            >
              去复查上轮整改
            </button>
          )}
        </div>
      )}


      {decision === "revision" && (
        <div
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
                    deliveryItemCount > 0
                      ? "#9A3412"
                      : C.textSec,
                  fontSize: "11px",
                  fontWeight: 700,
                  lineHeight: 1.6,
                }}
              >
                {deliveryItemCount > 0
                  ? `📤 本次将交付 ${deliveryPageCount} 个页面或整课范围、${deliveryItemCount} 条确认指令`
                  : "📭 当前没有随本次退回交付的确认指令"}
              </div>

              <div
                style={{
                  marginTop: "3px",
                  color: C.textMuted,
                  fontSize: "9px",
                  lineHeight: 1.5,
                }}
              >
                从本区域移出只影响本次退回，不会删除、忽略或修改原整改项。
              </div>
            </div>

            {deliveryItemCount > 0 && (
              <button
                type="button"
                onClick={() => setExpanded((previous) => !previous)}
                style={smallButtonStyle}
              >
                {expanded ? "收起清单" : "展开清单"}
              </button>
            )}
          </div>

          {deliveryItemCount === 0 && (
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
                maxHeight: "230px",
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
                    }}
                  >
                    {group.pageNumber > 0 ? (
                      <button
                        type="button"
                        onClick={() => {
                          onSelectPage(group.pageNumber);
                          onOpenAI();
                        }}
                        style={{
                          ...smallButtonStyle,
                          color: "#C2410C",
                          border: "1px solid #FDBA74",
                        }}
                      >
                        P{group.pageNumber}
                      </button>
                    ) : (
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
                        整课
                      </span>
                    )}

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
                  </div>

                  {group.items.map((item) => (
                    <div
                      key={item.id}
                      style={{
                        marginTop: "7px",
                        paddingTop: "7px",
                        borderTop: "1px solid #FDE7D3",
                      }}
                    >
                      <div
                        style={{
                          display: "flex",
                          alignItems: "flex-start",
                          gap: "7px",
                        }}
                      >
                        <div
                          style={{
                            minWidth: 0,
                            flex: 1,
                          }}
                        >
                          <div
                            style={{
                              color: C.text,
                              fontSize: "10px",
                              fontWeight: 700,
                              lineHeight: 1.5,
                            }}
                          >
                            {item.title || "已确认整改要求"}
                          </div>

                          <div
                            style={{
                              marginTop: "3px",
                              color: C.textSec,
                              fontSize: "10px",
                              lineHeight: 1.55,
                              whiteSpace: "pre-wrap",
                              wordBreak: "break-word",
                            }}
                          >
                            {item.confirmed_instruction}
                          </div>
                        </div>

                        <button
                          type="button"
                          onClick={() => removeSelectedItem(item.id)}
                          disabled={!aiReviewContext.removeSelectedItem}
                          title="仅从本次正式退回清单移出"
                          style={{
                            ...smallButtonStyle,
                            flexShrink: 0,
                            color: C.danger,
                            border: "1px solid #FECACA",
                            cursor: aiReviewContext.removeSelectedItem
                              ? "pointer"
                              : "not-allowed",
                          }}
                        >
                          移出
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              ))}
            </div>
          )}

          {deliveryItemCount > 0 && (
            <div
              style={{
                marginTop: "9px",
                paddingTop: "9px",
                borderTop: "1px solid #FED7AA",
              }}
            >
              <button
                type="button"
                onClick={applyCurrentDraft}
                style={{
                  width: "100%",
                  padding: "7px",
                  borderRadius: "7px",
                  border: "none",
                  background: C.warning,
                  color: "#fff",
                  fontSize: "10px",
                  fontWeight: 700,
                  cursor: "pointer",
                }}
              >
                {lastAppliedDraft
                  ? "按当前清单更新审核意见草稿"
                  : "按当前清单生成审核意见草稿"}
              </button>

              <div
                style={{
                  marginTop: "5px",
                  color: draftSynced
                    ? C.success
                    : draftNeedsManualRefresh
                      ? C.warning
                      : C.textMuted,
                  fontSize: "9px",
                  lineHeight: 1.5,
                }}
              >
                {draftSynced
                  ? "✓ 当前审核意见已与退回清单同步；未人工编辑时会安全跟随清单变化。"
                  : draftNeedsManualRefresh
                    ? "清单已变化，但检测到人工编辑内容，因此没有自动覆盖。"
                    : "系统不会自动覆盖人工审核意见；首次使用请点击上方按钮。"}
              </div>
            </div>
          )}
        </div>
      )}

      <input
        type="number"
        min="1"
        max="10"
        step="0.5"
        value={score}
        onChange={(event) => onScoreChange(event.target.value)}
        placeholder="评分（可选，1-10）"
        style={{
          width: "100%",
          padding: "9px 12px",
          borderRadius: "10px",
          border: `1px solid ${C.borderMid}`,
          fontSize: "14px",
          outline: "none",
          boxSizing: "border-box",
          marginBottom: "12px",
        }}
      />

      <textarea
        value={comment}
        onChange={(event) => onCommentChange(event.target.value)}
        placeholder={
          decision === "approved"
            ? "课件整体质量良好，可通过…"
            : "请说明需要修改的地方…"
        }
        rows={4}
        style={{
          width: "100%",
          padding: "10px 12px",
          borderRadius: "10px",
          border: `1px solid ${C.borderMid}`,
          fontSize: "14px",
          outline: "none",
          resize: "vertical",
          boxSizing: "border-box",
          fontFamily: "inherit",
          marginBottom: "12px",
        }}
      />

      <div
        style={{
          display: "flex",
          gap: "10px",
        }}
      >
        <button
          type="button"
          onClick={onCancel}
          style={{
            padding: "10px 18px",
            borderRadius: "10px",
            border: `1px solid ${C.borderMid}`,
            background: "#fff",
            cursor: "pointer",
            fontSize: "14px",
            color: C.textSec,
            flexShrink: 0,
          }}
        >
          取消
        </button>

        <button
          type="button"
          onClick={
            handleSubmit
          }
          disabled={
            submitBlocked
          }
          title={
            decision === "approved" &&
            approvalBlocked
              ? `还有${approvalGuard.blockedCount}条上轮问题未确认解决`
              : undefined
          }
          style={{
            flex: 1,
            padding: "10px",
            borderRadius: "10px",
            border: "none",
            background:
              submitBlocked
                ? "#CBD5E1"
                : decision === "approved"
                  ? C.success
                  : C.warning,
            color: "#fff",
            cursor:
              submitBlocked
                ? "not-allowed"
                : "pointer",
            fontSize: "14px",
            fontWeight: 600,
            opacity:
              submitBlocked
                ? 0.7
                : 1,
          }}
        >
          {submitting
            ? "提交中…"
            : decision === "approved"
              ? approvalBlocked
                ? `还有 ${approvalGuard.blockedCount} 条未确认`
                : "✅ 确认通过"
              : "↩️ 确认退回"}
        </button>
      </div>

      <p
        style={{
          margin: "8px 0 0",
          fontSize: "11px",
          color: C.textMuted,
          textAlign: "center",
        }}
      >
        {level === 1
          ? "L1通过后若学校开启L2，将进入学校审核"
          : "L2通过后课件进入待发布，作者可共享"}
      </p>
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
