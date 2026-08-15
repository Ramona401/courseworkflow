/**
 * CWAIReviewExecutionNote.tsx
 *
 * 正式整改作者的“本次执行补充”。
 *
 * 该动作：
 *   - 只追加作者说明；
 *   - 不调用AI；
 *   - 不改写审核员修改要求；
 *   - 不改变问题状态；
 *   - 保存结果仍交回父层统一刷新当前整改项。
 */

import { useState } from "react";

import {
  addCWAIReviewItemExecutionNote,
  type CWAIReviewItemDiscussion,
} from "@/api/coursewares";

import {
  CW_AI_REVIEW_ITEM_COLORS as C,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewExecutionNoteProps {
  itemId: string;
  available: boolean;
  onSaved: (result: CWAIReviewItemDiscussion) => void;
}

export default function CWAIReviewExecutionNote({
  itemId,
  available,
  onSaved,
}: CWAIReviewExecutionNoteProps) {
  const [executionNote, setExecutionNote] = useState("");
  const [saving, setSaving] = useState(false);
  const [successMessage, setSuccessMessage] = useState("");
  const [error, setError] = useState("");

  if (!available) {
    return null;
  }

  const handleSave = async () => {
    const content = executionNote.trim();

    if (!content || saving) {
      return;
    }

    setSaving(true);
    setSuccessMessage("");
    setError("");

    try {
      const result =
        await addCWAIReviewItemExecutionNote(
          itemId,
          content,
        );

      onSaved(result);
      setExecutionNote("");
      setSuccessMessage("本次执行补充已保存。");
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "保存本次执行补充失败",
      );
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      style={{
        marginTop: "10px",
        padding: "10px",
        borderRadius: "9px",
        border: "1px solid #FED7AA",
        background: "#FFF7ED",
      }}
    >
      <div
        style={{
          color: C.text,
          fontSize: "11px",
          fontWeight: 700,
        }}
      >
        本次执行补充
      </div>

      <div
        style={{
          marginTop: "4px",
          color: C.textSec,
          fontSize: "10px",
          lineHeight: 1.6,
        }}
      >
        可以记录你这次实际做了什么、还有什么需要复审时注意。
        这里不会改写审核员已经确认的修改要求。
      </div>

      <textarea
        value={executionNote}
        onChange={(event) => {
          setExecutionNote(event.target.value);
          setSuccessMessage("");
        }}
        rows={3}
        maxLength={2000}
        disabled={saving}
        placeholder="例如：已调整课堂提示顺序，并在真实页面连续操作检查了两次。"
        style={{
          width: "100%",
          boxSizing: "border-box",
          marginTop: "8px",
          padding: "8px 9px",
          borderRadius: "7px",
          border: "1px solid #FED7AA",
          resize: "vertical",
          fontFamily: "inherit",
          fontSize: "11px",
          lineHeight: 1.6,
          outline: "none",
          background: "#FFFFFF",
        }}
      />

      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "8px",
          marginTop: "6px",
          flexWrap: "wrap",
        }}
      >
        <span
          style={{
            flex: 1,
            color: C.textMuted,
            fontSize: "9px",
          }}
        >
          {Array.from(executionNote).length}/2000
        </span>

        <button
          type="button"
          onClick={() => void handleSave()}
          disabled={saving || !executionNote.trim()}
          style={{
            padding: "7px 10px",
            borderRadius: "7px",
            border: "none",
            background:
              saving || !executionNote.trim()
                ? "#CBD5E1"
                : C.primary,
            color: "#FFFFFF",
            fontSize: "10px",
            fontWeight: 700,
            cursor:
              saving || !executionNote.trim()
                ? "not-allowed"
                : "pointer",
          }}
        >
          {saving ? "正在保存…" : "保存补充"}
        </button>
      </div>

      {successMessage && (
        <div
          role="status"
          style={{
            marginTop: "6px",
            color: C.success,
            fontSize: "10px",
            fontWeight: 600,
          }}
        >
          {successMessage}
        </div>
      )}

      {error && (
        <div
          style={{
            marginTop: "7px",
            padding: "7px 8px",
            borderRadius: "6px",
            background: "#FEF2F2",
            color: C.danger,
            fontSize: "10px",
            lineHeight: 1.5,
          }}
        >
          {error}
        </div>
      )}
    </div>
  );
}
