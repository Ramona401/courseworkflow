/**
 * CWReviewCommentCandidatePanel.tsx
 *
 * R-08“审核意见重新汇总与差异预览”教师确认组件。
 *
 * 工作流：
 *   1. 教师先维护“本次修改清单”和当前人工审核意见；
 *   2. 点击“重新整理审核意见”；
 *   3. 后端重新读取当前确认指令版本和R-06问题组事实并生成不可变候选；
 *   4. 弹窗同时展示原意见、新候选和新增/删除/调整差异；
 *   5. 教师明确选择“替换原意见”“追加到原意见”或“取消”；
 *   6. Apply时后端再次核验全部可信事实，任何变化均409 stale；
 *   7. Apply成功仅更新当前页面的审核意见输入框，正式提交仍由人工按钮完成。
 *
 * 本组件绝不把candidate正文回传给后端。
 */

import { useMemo, useState, type CSSProperties } from "react";

import {
  applyCWReviewCommentCandidate,
  generateCWReviewCommentCandidate,
  type CWReviewCommentCandidate,
  type CWReviewCommentCandidateApplyAction,
} from "@/api/coursewares";

import type { CWAIReviewContext } from "./CWAIReviewPanel";

const C = {
  primary: "#4F7BE8",
  primaryLight: "#EEF4FF",
  success: "#10B981",
  warning: "#F59E0B",
  danger: "#EF4444",
  text: "#1F2937",
  textSec: "#6B7280",
  textMuted: "#9CA3AF",
  border: "#E5E7EB",
  softBorder: "#F3F4F6",
};

export interface CWReviewCommentCandidatePanelProps {
  decision: "approved" | "revision";
  comment: string;
  submitting: boolean;
  aiReviewContext: CWAIReviewContext;
  onCommentChange: (comment: string) => void;
}

function normalizeCommentForCandidate(value: string): string {
  return value.replace(/\r\n/g, "\n").replace(/\r/g, "\n").trim();
}

function normalizeSelectionSignature(itemIds: string[]): string {
  return Array.from(
    new Set(itemIds.map((itemId) => itemId.trim()).filter(Boolean)),
  )
    .sort()
    .join("|");
}

function resolveErrorMessage(cause: unknown, fallback: string): string {
  return cause instanceof Error && cause.message.trim()
    ? cause.message
    : fallback;
}

function isStaleErrorMessage(message: string): boolean {
  return (
    message.includes("需要重新整理审核意见") ||
    message.includes("审核意见已经变化") ||
    message.includes("修改要求") ||
    message.includes("问题组")
  );
}

export default function CWReviewCommentCandidatePanel({
  decision,
  comment,
  submitting,
  aiReviewContext,
  onCommentChange,
}: CWReviewCommentCandidatePanelProps) {
  const [candidate, setCandidate] =
    useState<CWReviewCommentCandidate | null>(null);
  const [generatedSelectionSignature, setGeneratedSelectionSignature] =
    useState("");
  const [generating, setGenerating] = useState(false);
  const [applyingAction, setApplyingAction] =
    useState<CWReviewCommentCandidateApplyAction | null>(null);
  const [needsRefresh, setNeedsRefresh] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const sessionId = aiReviewContext.sessionId?.trim() || "";

  const deliveryCountMismatch =
    aiReviewContext.selectedItemIds.length !==
    aiReviewContext.selectedItems.length;

  const currentSelectionSignature = useMemo(
    () => normalizeSelectionSignature(aiReviewContext.selectedItemIds),
    [aiReviewContext.selectedItemIds],
  );

  const normalizedCurrentComment = normalizeCommentForCandidate(comment);

  const localStale =
    !!candidate &&
    (candidate.original_comment !== normalizedCurrentComment ||
      generatedSelectionSignature !== currentSelectionSignature);

  const candidateNeedsRefresh = needsRefresh || localStale;

  const canGenerate =
    decision === "revision" &&
    !!sessionId &&
    aiReviewContext.selectedItemIds.length > 0 &&
    !deliveryCountMismatch &&
    !generating &&
    !applyingAction &&
    !submitting;

  const canApply =
    !!candidate &&
    !candidateNeedsRefresh &&
    !generating &&
    !applyingAction &&
    !submitting;

  if (decision !== "revision") {
    return null;
  }

  const generateCandidate = async () => {
    if (!canGenerate) {
      return;
    }

    setGenerating(true);
    setError("");
    setNotice("");
    setNeedsRefresh(false);

    try {
      const result = await generateCWReviewCommentCandidate(sessionId, {
        selected_item_ids: [...aiReviewContext.selectedItemIds],
        original_comment: comment,
      });

      setCandidate(result);
      setGeneratedSelectionSignature(currentSelectionSignature);
    } catch (cause) {
      setError(
        resolveErrorMessage(cause, "重新整理审核意见失败，请稍后重试"),
      );
    } finally {
      setGenerating(false);
    }
  };

  const applyCandidate = async (
    action: CWReviewCommentCandidateApplyAction,
  ) => {
    if (!candidate || !canApply) {
      return;
    }

    setApplyingAction(action);
    setError("");
    setNotice("");

    try {
      const result = await applyCWReviewCommentCandidate(
        sessionId,
        candidate.id,
        {
          action,
          selected_item_ids: [...aiReviewContext.selectedItemIds],
          current_comment: comment,
        },
      );

      onCommentChange(result.next_comment);
      setCandidate(null);
      setGeneratedSelectionSignature("");
      setNeedsRefresh(false);
      setNotice(
        action === "replace"
          ? "新的审核意见已替换到输入框，请继续人工检查后再正式提交。"
          : "新的审核意见已追加到输入框，请继续人工检查后再正式提交。",
      );
    } catch (cause) {
      const message = resolveErrorMessage(
        cause,
        "应用审核意见候选失败，请重新整理后再试",
      );

      setError(message);

      if (isStaleErrorMessage(message)) {
        setNeedsRefresh(true);
      }
    } finally {
      setApplyingAction(null);
    }
  };

  const cancelCandidate = () => {
    setCandidate(null);
    setGeneratedSelectionSignature("");
    setNeedsRefresh(false);
    setError("");
  };

  return (
    <>
      <div
        style={{
          marginBottom: "12px",
          padding: "10px 11px",
          borderRadius: "9px",
          border: `1px solid ${C.border}`,
          background: "#fff",
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
          ✨ 审核意见重新整理
        </div>

        <div
          style={{
            marginTop: "4px",
            color: C.textMuted,
            fontSize: "9px",
            lineHeight: 1.55,
          }}
        >
          基于当前本次修改清单、已确认修改要求和问题组重新生成候选。
          系统不会直接覆盖人工审核意见。
        </div>

        <button
          type="button"
          disabled={!canGenerate}
          onClick={() => void generateCandidate()}
          style={{
            width: "100%",
            marginTop: "8px",
            padding: "8px 10px",
            borderRadius: "7px",
            border: "none",
            background: canGenerate ? C.primary : "#CBD5E1",
            color: "#fff",
            fontSize: "10px",
            fontWeight: 700,
            cursor: canGenerate ? "pointer" : "not-allowed",
          }}
        >
          {generating
            ? "AI正在重新整理…"
            : aiReviewContext.selectedItemIds.length === 0
              ? "请先确认本次修改清单"
              : deliveryCountMismatch
                ? "本次修改清单状态异常"
                : !sessionId
                  ? "请先完成本次AI审核"
                  : "重新整理审核意见并预览差异"}
        </button>

        {notice && <div style={successMessageStyle}>✓ {notice}</div>}
        {error && !candidate && <div style={errorMessageStyle}>⚠️ {error}</div>}
      </div>

      {candidate && (
        <div
          role="dialog"
          aria-modal="true"
          aria-label="审核意见重新整理差异预览"
          style={overlayStyle}
        >
          <div style={dialogStyle}>
            <div style={dialogHeaderStyle}>
              <div>
                <div
                  style={{
                    color: C.text,
                    fontSize: "16px",
                    fontWeight: 700,
                  }}
                >
                  审核意见差异确认
                </div>

                <div
                  style={{
                    marginTop: "4px",
                    color: C.textMuted,
                    fontSize: "11px",
                    lineHeight: 1.5,
                  }}
                >
                  候选不会自动写入。请检查原意见、新候选和差异后明确选择。
                </div>
              </div>
            </div>

            {candidateNeedsRefresh && (
              <div
                style={{
                  margin: "0 18px 12px",
                  padding: "10px 12px",
                  borderRadius: "8px",
                  border: "1px solid #FDE68A",
                  background: "#FFFBEB",
                  color: "#92400E",
                  fontSize: "11px",
                  lineHeight: 1.6,
                }}
              >
                ⚠️ 本次修改清单、修改要求、问题组或人工审核意见可能已经变化。
                当前候选已标记为“需要重新整理”，不能继续替换或追加。
              </div>
            )}

            {error && (
              <div
                style={{
                  ...errorMessageStyle,
                  margin: "0 18px 12px",
                }}
              >
                ⚠️ {error}
              </div>
            )}

            <div style={dialogBodyStyle}>
              <div style={comparisonGridStyle}>
                <CommentPreview
                  title="原审核意见"
                  value={candidate.original_comment}
                  emptyText="生成候选时原审核意见为空"
                />

                <CommentPreview
                  title="新审核意见候选"
                  value={candidate.candidate_text}
                  emptyText="候选为空"
                  emphasized
                />
              </div>

              <div
                style={{
                  marginTop: "14px",
                  paddingTop: "14px",
                  borderTop: `1px solid ${C.softBorder}`,
                }}
              >
                <div
                  style={{
                    color: C.text,
                    fontSize: "12px",
                    fontWeight: 700,
                    marginBottom: "9px",
                  }}
                >
                  差异摘要
                </div>

                <DiffList
                  title="新增"
                  items={candidate.diff.added}
                  emptyText="无新增内容"
                  marker="+"
                />

                <DiffList
                  title="删除"
                  items={candidate.diff.removed}
                  emptyText="无删除内容"
                  marker="-"
                />

                <AdjustmentList
                  items={candidate.diff.adjusted}
                />

                {candidate.diff.added.length === 0 &&
                  candidate.diff.removed.length === 0 &&
                  candidate.diff.adjusted.length === 0 && (
                    <div
                      style={{
                        color: C.textMuted,
                        fontSize: "11px",
                        lineHeight: 1.6,
                      }}
                    >
                      原意见与新候选没有检测到文本差异。
                    </div>
                  )}
              </div>
            </div>

            <div style={dialogFooterStyle}>
              <button
                type="button"
                disabled={!!applyingAction || generating}
                onClick={cancelCandidate}
                style={secondaryButtonStyle}
              >
                取消
              </button>

              {candidateNeedsRefresh ? (
                <button
                  type="button"
                  disabled={!canGenerate}
                  onClick={() => void generateCandidate()}
                  style={{
                    ...primaryButtonStyle,
                    background: canGenerate ? C.warning : "#CBD5E1",
                    cursor: canGenerate ? "pointer" : "not-allowed",
                  }}
                >
                  {generating ? "重新整理中…" : "按当前内容重新整理"}
                </button>
              ) : (
                <>
                  <button
                    type="button"
                    disabled={!canApply}
                    onClick={() => void applyCandidate("append")}
                    style={{
                      ...secondaryButtonStyle,
                      color: C.primary,
                      border: `1px solid ${C.primary}`,
                      opacity: canApply ? 1 : 0.6,
                      cursor: canApply ? "pointer" : "not-allowed",
                    }}
                  >
                    {applyingAction === "append"
                      ? "追加中…"
                      : "追加到原意见"}
                  </button>

                  <button
                    type="button"
                    disabled={!canApply}
                    onClick={() => void applyCandidate("replace")}
                    style={{
                      ...primaryButtonStyle,
                      opacity: canApply ? 1 : 0.6,
                      cursor: canApply ? "pointer" : "not-allowed",
                    }}
                  >
                    {applyingAction === "replace"
                      ? "替换中…"
                      : "替换原意见"}
                  </button>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  );
}

function CommentPreview({
  title,
  value,
  emptyText,
  emphasized = false,
}: {
  title: string;
  value: string;
  emptyText: string;
  emphasized?: boolean;
}) {
  return (
    <div
      style={{
        minWidth: 0,
        padding: "12px",
        borderRadius: "9px",
        border: emphasized
          ? `1px solid ${C.primary}55`
          : `1px solid ${C.border}`,
        background: emphasized ? C.primaryLight : "#FAFAFA",
      }}
    >
      <div
        style={{
          color: emphasized ? C.primary : C.textSec,
          fontSize: "11px",
          fontWeight: 700,
          marginBottom: "7px",
        }}
      >
        {title}
      </div>

      <div
        style={{
          minHeight: "80px",
          color: value ? C.text : C.textMuted,
          fontSize: "12px",
          lineHeight: 1.7,
          whiteSpace: "pre-wrap",
          wordBreak: "break-word",
        }}
      >
        {value || emptyText}
      </div>
    </div>
  );
}

function DiffList({
  title,
  items,
  emptyText,
  marker,
}: {
  title: string;
  items: string[];
  emptyText: string;
  marker: string;
}) {
  if (items.length === 0) {
    return (
      <div style={{ ...diffBlockStyle, color: C.textMuted }}>
        <strong>{title}：</strong>
        {emptyText}
      </div>
    );
  }

  return (
    <div style={diffBlockStyle}>
      <strong>{title}：</strong>

      {items.map((item, index) => (
        <div
          key={`${title}-${index}-${item}`}
          style={{
            marginTop: "5px",
            paddingLeft: "8px",
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
          }}
        >
          {marker} {item}
        </div>
      ))}
    </div>
  );
}

function AdjustmentList({
  items,
}: {
  items: Array<{
    before: string;
    after: string;
  }>;
}) {
  if (items.length === 0) {
    return (
      <div style={{ ...diffBlockStyle, color: C.textMuted }}>
        <strong>调整：</strong>
        无调整内容
      </div>
    );
  }

  return (
    <div style={diffBlockStyle}>
      <strong>调整：</strong>

      {items.map((item, index) => (
        <div
          key={`adjusted-${index}-${item.before}-${item.after}`}
          style={{
            marginTop: "7px",
            padding: "8px",
            borderRadius: "7px",
            background: "#FAFAFA",
            border: `1px solid ${C.softBorder}`,
          }}
        >
          <div
            style={{
              color: C.textMuted,
              whiteSpace: "pre-wrap",
              wordBreak: "break-word",
            }}
          >
            原：{item.before}
          </div>

          <div
            style={{
              marginTop: "4px",
              color: C.text,
              whiteSpace: "pre-wrap",
              wordBreak: "break-word",
            }}
          >
            新：{item.after}
          </div>
        </div>
      ))}
    </div>
  );
}

const overlayStyle: CSSProperties = {
  position: "fixed",
  inset: 0,
  zIndex: 100000,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  padding: "24px",
  background: "rgba(15, 23, 42, 0.48)",
};

const dialogStyle: CSSProperties = {
  width: "min(820px, 100%)",
  maxHeight: "88vh",
  display: "flex",
  flexDirection: "column",
  overflow: "hidden",
  borderRadius: "14px",
  background: "#fff",
  boxShadow: "0 24px 70px rgba(15,23,42,0.24)",
};

const dialogHeaderStyle: CSSProperties = {
  padding: "16px 18px",
  borderBottom: `1px solid ${C.softBorder}`,
};

const dialogBodyStyle: CSSProperties = {
  flex: 1,
  minHeight: 0,
  overflowY: "auto",
  padding: "0 18px 18px",
};

const comparisonGridStyle: CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(260px, 1fr))",
  gap: "10px",
  paddingTop: "14px",
};

const dialogFooterStyle: CSSProperties = {
  display: "flex",
  justifyContent: "flex-end",
  gap: "8px",
  padding: "12px 18px",
  borderTop: `1px solid ${C.softBorder}`,
  background: "#FAFAFA",
};

const secondaryButtonStyle: CSSProperties = {
  padding: "8px 13px",
  borderRadius: "8px",
  border: `1px solid ${C.border}`,
  background: "#fff",
  color: C.textSec,
  fontSize: "11px",
  fontWeight: 700,
  cursor: "pointer",
};

const primaryButtonStyle: CSSProperties = {
  padding: "8px 13px",
  borderRadius: "8px",
  border: "none",
  background: C.primary,
  color: "#fff",
  fontSize: "11px",
  fontWeight: 700,
  cursor: "pointer",
};

const diffBlockStyle: CSSProperties = {
  marginTop: "7px",
  padding: "8px 10px",
  borderRadius: "7px",
  border: `1px solid ${C.softBorder}`,
  color: C.textSec,
  fontSize: "11px",
  lineHeight: 1.6,
};

const errorMessageStyle: CSSProperties = {
  marginTop: "7px",
  padding: "7px 9px",
  borderRadius: "7px",
  border: "1px solid #FECACA",
  background: "#FEF2F2",
  color: C.danger,
  fontSize: "10px",
  lineHeight: 1.55,
};

const successMessageStyle: CSSProperties = {
  marginTop: "7px",
  padding: "7px 9px",
  borderRadius: "7px",
  border: "1px solid #A7F3D0",
  background: "#ECFDF5",
  color: C.success,
  fontSize: "10px",
  lineHeight: 1.55,
};
