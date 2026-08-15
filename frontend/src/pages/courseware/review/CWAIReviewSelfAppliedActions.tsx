/**
 * CWAIReviewSelfAppliedActions.tsx
 *
 * 作者自审完成页面修改后的三项人工决策区。
 *
 * 业务规则：
 *   - “确认已经解决”仍由父容器保留为唯一主要操作；
 *   - 本组件只提供两个直接可见的次要操作：
 *     “继续调整”和“暂时不处理”；
 *   - “继续调整”只把同一条当前确认方案重新带入页面微调草稿，
 *     点击本身不修改整改项状态；
 *   - “暂时不处理”复用既有dismiss接口，并要求教师填写可回看的说明；
 *   - 暂时不处理的确认按钮仍保持secondary视觉，避免出现第二个主要操作。
 *
 * 本组件不发请求、不决定权限，最终状态仍由后端校验。
 */

import type { CSSProperties } from "react";

import {
  CW_AI_REVIEW_ITEM_COLORS as C,
  type CWAIReviewItemStateAction,
  cwAIReviewPauseTextareaStyle,
  cwAIReviewSecondaryButtonStyle,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewSelfAppliedActionsProps {
  canContinue: boolean;
  canPause: boolean;
  busy: boolean;
  stateAction: CWAIReviewItemStateAction;

  showPauseForm: boolean;
  pauseReason: string;

  onContinue: () => void;
  onTogglePause: () => void;
  onPauseReasonChange: (value: string) => void;
  onPause: () => void;
}

export default function CWAIReviewSelfAppliedActions({
  canContinue,
  canPause,
  busy,
  stateAction,
  showPauseForm,
  pauseReason,
  onContinue,
  onTogglePause,
  onPauseReasonChange,
  onPause,
}: CWAIReviewSelfAppliedActionsProps) {
  const pauseReasonLength =
    Array.from(pauseReason).length;

  const pauseReasonInvalid =
    !pauseReason.trim() ||
    pauseReasonLength > 500;

  return (
    <>
      <button
        type="button"
        onClick={onContinue}
        disabled={
          busy ||
          !canContinue
        }
        style={
          secondaryActionStyle(
            busy ||
              !canContinue,
          )
        }
      >
        继续调整
      </button>

      <button
        type="button"
        onClick={onTogglePause}
        disabled={
          busy ||
          !canPause
        }
        style={
          secondaryActionStyle(
            busy ||
              !canPause,
            "warning",
          )
        }
      >
        {showPauseForm
          ? "取消"
          : "暂时不处理"}
      </button>

      {showPauseForm &&
        canPause && (
        <div style={pauseDecisionStyle}>
          <div style={pauseTitleStyle}>
            为什么暂时不处理这条问题？
          </div>

          <textarea
            value={pauseReason}
            onChange={(event) =>
              onPauseReasonChange(
                event.target.value,
              )
            }
            rows={2}
            maxLength={500}
            disabled={busy}
            placeholder="例如：当前课堂安排先保持不变，下一轮再继续检查和调整"
            style={
              cwAIReviewPauseTextareaStyle
            }
          />

          <div style={pauseFooterStyle}>
            <span style={pauseCountStyle}>
              {pauseReasonLength}/500；以后仍可恢复
            </span>

            <button
              type="button"
              onClick={onPause}
              disabled={
                busy ||
                pauseReasonInvalid
              }
              style={
                secondaryActionStyle(
                  busy ||
                    pauseReasonInvalid,
                  "warning",
                )
              }
            >
              {stateAction === "dismiss"
                ? "正在保存…"
                : "确认暂时不处理"}
            </button>
          </div>
        </div>
      )}
    </>
  );
}

function secondaryActionStyle(
  disabled: boolean,
  tone: "default" | "warning" = "default",
): CSSProperties {
  return {
    ...cwAIReviewSecondaryButtonStyle,
    width: "auto",
    border:
      tone === "warning"
        ? `1px solid ${C.warning}`
        : cwAIReviewSecondaryButtonStyle.border,
    color:
      disabled
        ? C.textMuted
        : tone === "warning"
          ? C.warning
          : cwAIReviewSecondaryButtonStyle.color,
    background:
      disabled
        ? "#F8FAFC"
        : "#FFFFFF",
    cursor:
      disabled
        ? "not-allowed"
        : "pointer",
  };
}

const pauseDecisionStyle = {
  width: "100%",
  boxSizing: "border-box",
  marginTop: "4px",
  padding: "12px",
  borderRadius: "9px",
  border: "1px solid #FED7AA",
  background: "#FFF7ED",
} as const;

const pauseTitleStyle = {
  color: "#9A3412",
  fontSize: "14px",
  fontWeight: 700,
} as const;

const pauseFooterStyle = {
  display: "flex",
  alignItems: "center",
  gap: "8px",
  marginTop: "6px",
  flexWrap: "wrap",
} as const;

const pauseCountStyle = {
  flex: 1,
  minWidth: "160px",
  color: C.textMuted,
  fontSize: "12px",
} as const;
