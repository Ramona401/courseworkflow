/**
 * CWAIReviewGoalDriftPrompt.tsx
 *
 * R-01.1 修改要求目标漂移的三选一确认区。
 *
 * 本组件只展示教师意图选择，不自行发送请求：
 *   - 继续处理当前问题；
 *   - 创建新改进项；
 *   - 取消。
 */

import type { CSSProperties } from "react";

import {
  CW_AI_REVIEW_GOAL_DRIFT_CANCEL,
  CW_AI_REVIEW_GOAL_DRIFT_CONTINUE,
  CW_AI_REVIEW_GOAL_DRIFT_CREATE,
  CW_AI_REVIEW_GOAL_DRIFT_PROMPT,
} from "./CWAIReviewGoalDrift.shared";

import {
  CW_AI_REVIEW_ITEM_COLORS as C,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewGoalDriftPromptProps {
  busy: boolean;
  creatingRelated: boolean;
  onContinueCurrent: () => void;
  onCreateRelated: () => void;
  onCancel: () => void;
}

export default function CWAIReviewGoalDriftPrompt({
  busy,
  creatingRelated,
  onContinueCurrent,
  onCreateRelated,
  onCancel,
}: CWAIReviewGoalDriftPromptProps) {
  return (
    <div
      role="alert"
      style={{
        marginTop: "8px",
        padding: "10px",
        borderRadius: "8px",
        border: "1px solid #F59E0B",
        background: "#FFF7ED",
      }}
    >
      <div
        style={{
          color: "#9A3412",
          fontSize: "11px",
          fontWeight: 700,
          lineHeight: 1.6,
        }}
      >
        {CW_AI_REVIEW_GOAL_DRIFT_PROMPT}
      </div>

      <div
        style={{
          marginTop: "5px",
          color: C.textSec,
          fontSize: "10px",
          lineHeight: 1.6,
        }}
      >
        请选择这段文字应该继续属于当前问题，
        还是单独成为一个新的改进项。选择取消不会保存任何内容。
      </div>

      <div
        style={{
          display: "flex",
          gap: "6px",
          flexWrap: "wrap",
          marginTop: "9px",
        }}
      >
        <button
          type="button"
          onClick={onContinueCurrent}
          disabled={busy}
          style={goalDriftButtonStyle("primary", busy)}
        >
          {CW_AI_REVIEW_GOAL_DRIFT_CONTINUE}
        </button>

        <button
          type="button"
          onClick={onCreateRelated}
          disabled={busy}
          style={goalDriftButtonStyle("success", busy)}
        >
          {creatingRelated
            ? "正在创建…"
            : CW_AI_REVIEW_GOAL_DRIFT_CREATE}
        </button>

        <button
          type="button"
          onClick={onCancel}
          disabled={busy}
          style={goalDriftButtonStyle("secondary", busy)}
        >
          {CW_AI_REVIEW_GOAL_DRIFT_CANCEL}
        </button>
      </div>
    </div>
  );
}

function goalDriftButtonStyle(
  tone: "primary" | "success" | "secondary",
  disabled: boolean,
): CSSProperties {
  const config = {
    primary: {
      background: C.primary,
      border: C.primary,
      color: "#FFFFFF",
    },
    success: {
      background: C.success,
      border: C.success,
      color: "#FFFFFF",
    },
    secondary: {
      background: "#FFFFFF",
      border: C.border,
      color: C.textSec,
    },
  }[tone];

  return {
    minHeight: "34px",
    padding: "7px 10px",
    borderRadius: "7px",
    border: `1px solid ${disabled ? "#CBD5E1" : config.border}`,
    background: disabled ? "#F1F5F9" : config.background,
    color: disabled ? C.textMuted : config.color,
    fontSize: "10px",
    fontWeight: 700,
    cursor: disabled ? "not-allowed" : "pointer",
  };
}
