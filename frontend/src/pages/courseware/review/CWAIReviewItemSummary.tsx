/**
 * CWAIReviewItemSummary.tsx
 *
 * 单条问题默认展示的紧凑摘要。
 *
 * 设计约束：
 *   - 每条摘要最多使用两个彩色标签：严重程度和当前状态；
 *   - 页面、来源和关系数量使用普通辅助文字，避免标签噪音；
 *   - 标题和说明使用正常阅读字号，说明最多展示两行；
 *   - 下一步提示使用行动语言，不直接暴露内部状态码。
 */

import type {
  CWAIReviewItem,
  CWAIReviewItemRelation,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  CW_AI_REVIEW_ITEM_COLORS as C,
  CW_AI_REVIEW_ITEM_SEVERITY,
  type CWAIReviewItemExperience,
  resolveCWAIReviewItemNextStep,
  resolveCWAIReviewItemStatus,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewItemSummaryProps {
  experience:
    CWAIReviewItemExperience;

  item: CWAIReviewItem;

  activeRelations:
    CWAIReviewItemRelation[];

  sourceLabel: string;
  pageLabel: string;

  summaryTitle: string;
  summaryDescription: string;

  selectable: boolean;
  selected: boolean;
  canSelectForReturn: boolean;
  canOpenPageModification: boolean;

  onSelectedChange?: (
    itemID: string,
    selected: boolean,
  ) => void;
}

export default function CWAIReviewItemSummary({
  experience,
  item,
  activeRelations,
  sourceLabel,
  pageLabel,
  summaryTitle,
  summaryDescription,
  selectable,
  selected,
  canSelectForReturn,
  canOpenPageModification,
  onSelectedChange,
}: CWAIReviewItemSummaryProps) {
  const status =
    resolveCWAIReviewItemStatus(
      experience,
      item.status,
    );

  const severity =
    CW_AI_REVIEW_ITEM_SEVERITY[
      item.severity
    ] ||
    CW_AI_REVIEW_ITEM_SEVERITY.info;

  const pageDescription =
    item.page_title_snapshot
      ? `${pageLabel} · ${item.page_title_snapshot}`
      : pageLabel;

  return (
    <div
      style={{
        minWidth: 0,
        flex: 1,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "7px",
          flexWrap: "wrap",
        }}
      >
        <span
          style={{
            padding: "3px 7px",
            borderRadius: "6px",
            color: severity.color,
            background:
              severity.background,
            fontSize: "11px",
            fontWeight: 700,
            lineHeight: 1.35,
          }}
        >
          {severity.label}
        </span>

        <span
          style={{
            padding: "3px 7px",
            borderRadius: "6px",
            color: status.color,
            background:
              status.background,
            fontSize: "11px",
            fontWeight: 700,
            lineHeight: 1.35,
          }}
        >
          {status.label}
        </span>
      </div>

      <div
        style={{
          marginTop: "8px",
          color: C.textMuted,
          fontSize: "12px",
          fontWeight: 600,
          lineHeight: 1.55,
        }}
      >
        {pageDescription}
        {" · "}
        {sourceLabel}

        {activeRelations.length > 0
          ? ` · 与其他问题有 ${activeRelations.length} 条联系`
          : ""}
      </div>

      <div
        style={{
          marginTop: "7px",
          color: C.text,
          fontSize: "15px",
          fontWeight: 700,
          lineHeight: 1.6,
          wordBreak: "break-word",
        }}
      >
        <DiscussionMarkdown
          content={summaryTitle}
          compact
        />
      </div>

      {summaryDescription && (
        <div
          style={{
            marginTop: "4px",
            color: C.textSec,
            fontSize: "14px",
            lineHeight: 1.6,
            display: "-webkit-box",
            WebkitLineClamp: 2,
            WebkitBoxOrient:
              "vertical",
            overflow: "hidden",
            wordBreak: "break-word",
          }}
        >
          <DiscussionMarkdown
            content={
              summaryDescription
            }
            compact
          />
        </div>
      )}

      {experience === "review" &&
        selectable && (
        <div
          style={{
            marginTop: "10px",
          }}
        >
          {canSelectForReturn ? (
            <label
              style={{
                display:
                  "inline-flex",
                minHeight: "36px",
                alignItems: "center",
                gap: "7px",
                padding: "7px 10px",
                borderRadius: "8px",
                border:
                  selected
                    ? "1px solid #FDBA74"
                    : "1px solid #CBD5E1",
                background:
                  selected
                    ? "#FFF7ED"
                    : "#F8FAFC",
                color:
                  selected
                    ? "#C2410C"
                    : C.textSec,
                fontSize: "12px",
                fontWeight: 700,
                cursor: "pointer",
              }}
            >
              <input
                type="checkbox"
                checked={selected}
                onChange={(event) =>
                  onSelectedChange?.(
                    item.id,
                    event.target.checked,
                  )
                }
              />

              {selected
                ? "本次退回给作者"
                : "本次不退回给作者"}
            </label>
          ) : (
            <div
              style={{
                color: C.textMuted,
                fontSize: "12px",
                lineHeight: 1.6,
              }}
            >
              {item.status ===
              "dismissed"
                ? "这条不会发给作者。"
                : "确认整改要求后，才能决定是否发给作者。"}
            </div>
          )}
        </div>
      )}

      <div
        style={{
          marginTop: "10px",
          padding: "10px 12px",
          borderRadius: "8px",
          background:
            status.background,
          color: status.color,
          fontSize: "13px",
          fontWeight: 600,
          lineHeight: 1.6,
        }}
      >
        {resolveCWAIReviewItemNextStep(
          experience,
          item,
          selected,
          canOpenPageModification,
        )}
      </div>
    </div>
  );
}
