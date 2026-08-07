/**
 * CWAIReviewGlobalDiscussionPicker.tsx
 *
 * 跨页面、跨问题全局讨论的整改项选择和问题输入区域。
 *
 * 本组件只收集用户明确选择，不直接发起API请求。
 * 这里的选择与正式退回交付选择完全独立。
 * 人工新增问题会显示来源标识，但不会因此自动选中或确认。
 */

import { useMemo } from "react";

import type { CWAIReviewItem } from "@/api/coursewares";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
  CW_GLOBAL_SEVERITY_LABEL,
  cwGlobalLinkButtonStyle,
  cwGlobalPageButtonStyle,
  resolveCWGlobalItemPageLabel,
  resolveCWGlobalItemTitle,
} from "./CWAIReviewGlobalDiscussion.shared";

export interface CWAIReviewGlobalDiscussionPickerProps {
  mode: "formal" | "self";
  items: CWAIReviewItem[];
  selectedItemIds: string[];
  content: string;

  loading: boolean;
  sending: boolean;
  error: string;
  message: string;

  onToggleItem: (itemId: string, selected: boolean) => void;
  onSelectAvailable: () => void;
  onClearSelection: () => void;
  onRefresh: () => void;
  onContentChange: (content: string) => void;
  onSend: () => void;
  onSelectPage: (pageNumber: number) => void;
}

export default function CWAIReviewGlobalDiscussionPicker({
  mode,
  items,
  selectedItemIds,
  content,
  loading,
  sending,
  error,
  message,
  onToggleItem,
  onSelectAvailable,
  onClearSelection,
  onRefresh,
  onContentChange,
  onSend,
  onSelectPage,
}: CWAIReviewGlobalDiscussionPickerProps) {
  const selectedItemIDSet = useMemo(
    () => new Set(selectedItemIds),
    [selectedItemIds],
  );

  const modeLabel = mode === "self" ? "自审问题" : "审核问题";
  const sendDisabled =
    sending || selectedItemIds.length < 2 || !content.trim();

  return (
    <>
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
            color: C.text,
            fontSize: "11px",
            fontWeight: 700,
          }}
        >
          选择{modeLabel}
        </span>

        <span
          style={{
            padding: "2px 6px",
            borderRadius: "5px",
            background:
              selectedItemIds.length >= 2
                ? C.successSoft
                : "#F1F5F9",
            color:
              selectedItemIds.length >= 2
                ? C.success
                : C.textMuted,
            fontSize: "9px",
            fontWeight: 700,
          }}
        >
          已选 {selectedItemIds.length}/12
        </span>

        <button
          type="button"
          onClick={onSelectAvailable}
          style={cwGlobalLinkButtonStyle}
        >
          选择可处理项
        </button>

        {selectedItemIds.length > 0 && (
          <button
            type="button"
            onClick={onClearSelection}
            style={cwGlobalLinkButtonStyle}
          >
            清空
          </button>
        )}

        <button
          type="button"
          onClick={onRefresh}
          disabled={loading}
          style={{
            ...cwGlobalLinkButtonStyle,
            marginLeft: "auto",
            cursor: loading ? "not-allowed" : "pointer",
          }}
        >
          {loading ? "加载中…" : "刷新历史"}
        </button>
      </div>

      <div
        style={{
          marginTop: "5px",
          color: C.textMuted,
          fontSize: "9px",
          lineHeight: 1.5,
        }}
      >
        本区域勾选只决定本轮综合讨论范围，不会改变正式退回清单。
      </div>

      <div
        style={{
          maxHeight: "260px",
          overflowY: "auto",
          marginTop: "8px",
          padding: "2px",
        }}
      >
        {items.map((item) => {
          const checked = selectedItemIDSet.has(item.id);
          const pageLabel = resolveCWGlobalItemPageLabel(item);

          return (
            <label
              key={item.id}
              style={{
                display: "flex",
                alignItems: "flex-start",
                gap: "7px",
                marginBottom: "6px",
                padding: "8px",
                borderRadius: "7px",
                border: checked
                  ? `1px solid ${C.primary}`
                  : `1px solid ${C.border}`,
                background: checked ? C.primarySoft : C.card,
                cursor: "pointer",
              }}
            >
              <input
                type="checkbox"
                checked={checked}
                onChange={(event) =>
                  onToggleItem(item.id, event.target.checked)
                }
                style={{
                  marginTop: "2px",
                  cursor: "pointer",
                }}
              />

              <div style={{ minWidth: 0, flex: 1 }}>
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "5px",
                    flexWrap: "wrap",
                  }}
                >
                  {item.page_number_snapshot > 0 ? (
                    <button
                      type="button"
                      onClick={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        onSelectPage(item.page_number_snapshot);
                      }}
                      style={{
                        ...cwGlobalPageButtonStyle,
                        color: C.primary,
                      }}
                    >
                      {pageLabel}
                    </button>
                  ) : (
                    <span
                      style={{
                        ...cwGlobalPageButtonStyle,
                        cursor: "default",
                      }}
                    >
                      整课
                    </span>
                  )}

                  <span
                    style={{
                      padding: "1px 5px",
                      borderRadius: "4px",
                      background: "#F1F5F9",
                      color: C.textSec,
                      fontSize: "9px",
                      fontWeight: 700,
                    }}
                  >
                    {CW_GLOBAL_SEVERITY_LABEL[item.severity]}
                  </span>

                  {item.origin_type ===
                    "global_discussion_manual" && (
                    <span
                      style={{
                        padding: "1px 5px",
                        borderRadius: "4px",
                        background: "#F5F3FF",
                        color: "#7C3AED",
                        fontSize: "9px",
                        fontWeight: 700,
                      }}
                    >
                      人工新增
                    </span>
                  )}

                  <span
                    style={{
                      color:
                        item.status === "confirmed"
                          ? C.success
                          : C.textMuted,
                      fontSize: "9px",
                      fontWeight: 700,
                    }}
                  >
                    {item.status === "confirmed"
                      ? "已有确认指令"
                      : item.status === "discussing"
                        ? "讨论中"
                        : "待讨论"}
                  </span>
                </div>

                <div
                  style={{
                    marginTop: "4px",
                    color: C.text,
                    fontSize: "10px",
                    fontWeight: 700,
                    lineHeight: 1.5,
                  }}
                >
                  {resolveCWGlobalItemTitle(item)}
                </div>

                {item.page_title_snapshot && (
                  <div
                    style={{
                      marginTop: "2px",
                      color: C.textMuted,
                      fontSize: "9px",
                    }}
                  >
                    {item.page_title_snapshot}
                  </div>
                )}
              </div>
            </label>
          );
        })}
      </div>

      <div style={{ marginTop: "9px" }}>
        <textarea
          value={content}
          onChange={(event) => onContentChange(event.target.value)}
          rows={3}
          maxLength={8000}
          disabled={sending}
          placeholder="例如：请判断P3和P5的问题是否重复；若合并处理，分别给出不冲突的修改指令，并标出需要人工裁决的地方。"
          style={{
            width: "100%",
            boxSizing: "border-box",
            padding: "9px 10px",
            borderRadius: "8px",
            border: `1px solid ${C.border}`,
            resize: "vertical",
            outline: "none",
            fontFamily: "inherit",
            fontSize: "11px",
            lineHeight: 1.6,
          }}
        />

        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "8px",
            marginTop: "6px",
          }}
        >
          <span
            style={{
              minWidth: 0,
              flex: 1,
              color: C.textMuted,
              fontSize: "9px",
            }}
          >
            {Array.from(content).length}/8000
          </span>

          <button
            type="button"
            onClick={onSend}
            disabled={sendDisabled}
            style={{
              padding: "7px 12px",
              borderRadius: "7px",
              border: "none",
              background: sendDisabled ? "#CBD5E1" : C.primary,
              color: "#fff",
              fontSize: "10px",
              fontWeight: 700,
              cursor: sendDisabled ? "not-allowed" : "pointer",
            }}
          >
            {sending ? "正在综合分析…" : "发送全局讨论"}
          </button>
        </div>
      </div>

      {error && (
        <FeedbackMessage
          type="error"
          content={error}
        />
      )}

      {message && (
        <FeedbackMessage
          type="success"
          content={message}
        />
      )}
    </>
  );
}

function FeedbackMessage({
  type,
  content,
}: {
  type: "success" | "error";
  content: string;
}) {
  return (
    <div
      style={{
        marginTop: "8px",
        padding: "7px 9px",
        borderRadius: "7px",
        background:
          type === "success"
            ? C.successSoft
            : C.dangerSoft,
        color:
          type === "success"
            ? C.success
            : C.danger,
        fontSize: "10px",
        fontWeight: 600,
        lineHeight: 1.6,
      }}
    >
      {content}
    </div>
  );
}
