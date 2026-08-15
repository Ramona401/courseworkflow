/**
 * CWReviewDecisionPanel.tsx
 *
 * 课件正式审核决定与人工审核意见面板。
 *
 * R-05“本次修改清单”由CWReviewDeliveryPreview独立负责。
 * R-08审核意见重新整理与差异确认由CWReviewCommentCandidatePanel独立负责。
 *
 * 本文件保留：
 *   - 审核决定；
 *   - 上轮整改通过守卫；
 *   - 本次修改清单；
 *   - 评分；
 *   - 人工审核意见；
 *   - R-08入口；
 *   - 最终正式提交动作。
 */

import {
  useEffect,
  type CSSProperties,
} from "react";

import type { CWAIReviewContext } from "./CWAIReviewPanel";
import CWReviewCommentCandidatePanel from "./CWReviewCommentCandidatePanel";
import CWReviewDeliveryPreview from "./CWReviewDeliveryPreview";
import { publishCWReviewDeliveryDecision } from "./cwReviewDeliveryState";
import {
  requestCWReviewCarryoverFocus,
  useCWReviewApprovalGuard,
} from "./CWReviewSubmissionGuards";

const C = {
  success: "#10B981",
  warning: "#F59E0B",
  danger: "#EF4444",
  textSec: "#6B7280",
  textMuted: "#9CA3AF",
  borderMid: "#E5E7EB",
  bg: "#FAFBFC",
};

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
  const approvalGuard = useCWReviewApprovalGuard();
  const approvalBlocked = approvalGuard.blockedCount > 0;

  /**
   * 把当前人工审核决定发布给同页AI问题工具栏。
   *
   * 工具栏顶部“本次修改清单”数量必须与正式提交口径一致：
   * approved固定为0，revision才使用当前规范化交付选择数量。
   */
  useEffect(() => {
    publishCWReviewDeliveryDecision(decision);

    return () => {
      publishCWReviewDeliveryDecision(null);
    };
  }, [decision]);

  /**
   * 控制器应保证selectedItemIds与selectedItems是同一规范化集合。
   * 提交前再做一次fail-closed，防止显示数量与review_item_ids分叉。
   */
  const deliveryCountMismatch =
    decision === "revision" &&
    aiReviewContext.selectedItemIds.length !==
      aiReviewContext.selectedItems.length;

  const deliveryItemCount =
    decision === "revision"
      ? aiReviewContext.selectedItems.length
      : 0;

  const submitBlocked =
    submitting ||
    deliveryCountMismatch ||
    (decision === "approved" && approvalBlocked);

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
            disabled: approvalBlocked,
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
            disabled={option.disabled}
            title={
              option.disabled
                ? `还有${approvalGuard.blockedCount}条上轮问题未确认解决`
                : undefined
            }
            onClick={() => {
              if (!option.disabled) {
                onDecisionChange(option.value);
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
              cursor: option.disabled ? "not-allowed" : "pointer",
              opacity: option.disabled ? 0.55 : 1,
              fontSize: "14px",
              fontWeight: decision === option.value ? 600 : 400,
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

      {approvalGuard.totalCount > 0 && (
        <div
          style={{
            marginBottom: "12px",
            padding: "11px 12px",
            borderRadius: "9px",
            border: approvalBlocked
              ? "1px solid #FECACA"
              : "1px solid #A7F3D0",
            background: approvalBlocked ? "#FEF2F2" : "#ECFDF5",
          }}
        >
          <div
            style={{
              color: approvalBlocked ? C.danger : C.success,
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
            已确认解决 {approvalGuard.resolvedCount} 条
            {approvalGuard.notReadyCount > 0
              ? `；${approvalGuard.notReadyCount} 条作者尚未完成或状态异常`
              : ""}
            {approvalGuard.waitingReviewCount > 0
              ? `；${approvalGuard.waitingReviewCount} 条已完成修改但尚未勾选确认`
              : ""}
            {approvalGuard.changedPageCount > 0
              ? `；其中 ${approvalGuard.changedPageCount} 条涉及页面变化`
              : ""}
            。继续退回不受此限制。
          </div>

          {approvalBlocked && (
            <button
              type="button"
              onClick={requestCWReviewCarryoverFocus}
              style={{
                ...smallButtonStyle,
                marginTop: "9px",
                minHeight: "36px",
                color: C.danger,
                border: `1px solid ${C.danger}`,
              }}
            >
              去复查上轮整改
            </button>
          )}
        </div>
      )}

      <CWReviewDeliveryPreview
        decision={decision}
        aiReviewContext={aiReviewContext}
        onSelectPage={onSelectPage}
        onOpenAI={onOpenAI}
      />

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

      <CWReviewCommentCandidatePanel
        decision={decision}
        comment={comment}
        submitting={submitting}
        aiReviewContext={aiReviewContext}
        onCommentChange={onCommentChange}
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
          onClick={handleSubmit}
          disabled={submitBlocked}
          title={
            deliveryCountMismatch
              ? "本次修改清单数量与提交ID数量不一致，请刷新后重试"
              : decision === "approved" && approvalBlocked
                ? `还有${approvalGuard.blockedCount}条上轮问题未确认解决`
                : undefined
          }
          style={{
            flex: 1,
            padding: "10px",
            borderRadius: "10px",
            border: "none",
            background: submitBlocked
              ? "#CBD5E1"
              : decision === "approved"
                ? C.success
                : C.warning,
            color: "#fff",
            cursor: submitBlocked ? "not-allowed" : "pointer",
            fontSize: "14px",
            fontWeight: 600,
            opacity: submitBlocked ? 0.7 : 1,
          }}
        >
          {submitting
            ? "提交中…"
            : deliveryCountMismatch
              ? "清单数量异常"
              : decision === "approved"
                ? approvalBlocked
                  ? `还有 ${approvalGuard.blockedCount} 条未确认`
                  : "✅ 确认通过"
                : `↩️ 确认退回（${deliveryItemCount} 条）`}
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
