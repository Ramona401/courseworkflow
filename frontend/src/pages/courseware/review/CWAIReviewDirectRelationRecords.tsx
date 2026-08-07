/**
 * CWAIReviewDirectRelationRecords.tsx
 *
 * 问题关系状态、端点、说明、取消入口和版本事件历史。
 *
 * 本组件只展示受控关系数据并触发父组件回调，不直接请求接口。
 */

import type {
  CSSProperties,
} from "react";

import type {
  CWAIReviewGlobalRelationType,
  CWAIReviewItem,
  CWAIReviewItemRelation,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
  CW_GLOBAL_RELATION_CONFIG,
  resolveCWGlobalItemPageLabel,
  resolveCWGlobalItemTitle,
} from "./CWAIReviewGlobalDiscussion.shared";
import {
  countCWGlobalRunes,
  CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES,
} from "./CWAIReviewGlobalGovernanceLimits";

export interface CWAIReviewDirectRelationRecordsProps {
  relations: CWAIReviewItemRelation[];
  itemMap: ReadonlyMap<string, CWAIReviewItem>;
  cancelReasons: Record<string, string>;
  busyKey: string;

  onCancelReasonChange: (
    relationID: string,
    reason: string,
  ) => void;
  onSelectPage: (pageNumber: number) => void;
  onCancel: (
    relation: CWAIReviewItemRelation,
  ) => void;
}

function relationDirectionLabels(
  relationType: CWAIReviewGlobalRelationType,
): {
  sourceLabel: string;
  targetLabel: string;
} {
  switch (relationType) {
    case "duplicate":
      return {
        sourceLabel: "重复问题",
        targetLabel: "保留主问题",
      };

    case "merge":
      return {
        sourceLabel: "待合并问题",
        targetLabel: "合并主问题",
      };

    case "dependency":
      return {
        sourceLabel: "依赖问题",
        targetLabel: "前置问题",
      };

    case "possibly_resolved":
      return {
        sourceLabel: "待复核问题",
        targetLabel: "实际执行问题",
      };

    case "conflict":
      return {
        sourceLabel: "冲突端点A",
        targetLabel: "冲突端点B",
      };
  }
}

function relationSourceLabel(
  relation: CWAIReviewItemRelation,
): string {
  return relation.source_global_message_id
    ? "AI建议经人工确认"
    : "问题清单人工建立";
}

function itemOptionLabel(
  item: CWAIReviewItem,
): string {
  return [
    resolveCWGlobalItemPageLabel(item),
    resolveCWGlobalItemTitle(item),
  ].join(" · ");
}

export default function CWAIReviewDirectRelationRecords({
  relations,
  itemMap,
  cancelReasons,
  busyKey,
  onCancelReasonChange,
  onSelectPage,
  onCancel,
}: CWAIReviewDirectRelationRecordsProps) {
  if (relations.length === 0) {
    return null;
  }

  return (
    <div
      style={{
        marginTop: "12px",
      }}
    >
      <div
        style={{
          color: C.text,
          fontSize: "10px",
          fontWeight: 700,
        }}
      >
        已确认关系与版本历史
      </div>

      {relations.map((relation) => (
        <RelationRecordCard
          key={relation.id}
          relation={relation}
          itemMap={itemMap}
          cancelReason={
            cancelReasons[
              relation.id
            ] || ""
          }
          busyKey={busyKey}
          onCancelReasonChange={(reason) =>
            onCancelReasonChange(
              relation.id,
              reason,
            )
          }
          onSelectPage={onSelectPage}
          onCancel={() =>
            onCancel(relation)
          }
        />
      ))}
    </div>
  );
}

function RelationRecordCard({
  relation,
  itemMap,
  cancelReason,
  busyKey,
  onCancelReasonChange,
  onSelectPage,
  onCancel,
}: {
  relation: CWAIReviewItemRelation;
  itemMap: ReadonlyMap<string, CWAIReviewItem>;
  cancelReason: string;
  busyKey: string;
  onCancelReasonChange: (reason: string) => void;
  onSelectPage: (pageNumber: number) => void;
  onCancel: () => void;
}) {
  const config =
    CW_GLOBAL_RELATION_CONFIG[
      relation.relation_type
    ];

  const labels =
    relationDirectionLabels(
      relation.relation_type,
    );

  const sourceItem =
    itemMap.get(
      relation.source_item_id,
    );

  const targetItem =
    itemMap.get(
      relation.target_item_id,
    );

  const active =
    relation.status === "active";

  const cancelReasonLength =
    countCWGlobalRunes(
      cancelReason.trim(),
    );

  const cancelDisabled =
    !!busyKey ||
    !cancelReason.trim() ||
    cancelReasonLength >
      CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES;

  return (
    <div
      style={{
        marginTop: "7px",
        padding: "9px",
        borderRadius: "8px",
        border: `1px solid ${config.color}30`,
        background: "#fff",
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
            padding: "2px 6px",
            borderRadius: "5px",
            background: config.background,
            color: config.color,
            fontSize: "9px",
            fontWeight: 700,
          }}
        >
          {config.label}
        </span>

        <span
          style={{
            color: active
              ? C.success
              : C.textMuted,
            fontSize: "9px",
            fontWeight: 700,
          }}
        >
          {active ? "有效" : "已取消"} · v
          {relation.version}
        </span>

        <span
          style={{
            color: "#7C3AED",
            fontSize: "8px",
            fontWeight: 600,
          }}
        >
          {relationSourceLabel(relation)}
        </span>
      </div>

      <RelationEndpointRow
        label={labels.sourceLabel}
        item={sourceItem}
        fallbackID={
          relation.source_item_id
        }
        onSelectPage={onSelectPage}
      />

      <RelationEndpointRow
        label={labels.targetLabel}
        item={targetItem}
        fallbackID={
          relation.target_item_id
        }
        onSelectPage={onSelectPage}
      />

      <div
        style={{
          marginTop: "6px",
          color: C.textSec,
        }}
      >
        <DiscussionMarkdown
          content={relation.explanation}
          compact
        />
      </div>

      {relation.events.length > 0 && (
        <div style={eventBoxStyle}>
          {relation.events.map((event) => (
            <div
              key={event.id}
              style={eventLineStyle}
            >
              v{event.relation_version} ·{" "}
              {event.event_type} ·{" "}
              {event.reason ||
                "无补充说明"}
            </div>
          ))}
        </div>
      )}

      {active && (
        <>
          <textarea
            value={cancelReason}
            onChange={(event) =>
              onCancelReasonChange(
                event.target.value,
              )
            }
            rows={2}
            maxLength={
              CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
            }
            disabled={!!busyKey}
            placeholder="填写取消关系原因；历史不会删除"
            style={{
              ...inputStyle,
              marginTop: "7px",
              resize: "vertical",
            }}
          />

          <div
            style={{
              marginTop: "3px",
              color:
                cancelReasonLength >
                CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                  ? C.danger
                  : C.textMuted,
              fontSize: "8px",
              textAlign: "right",
            }}
          >
            {cancelReasonLength}/
            {CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES}
          </div>

          <button
            type="button"
            onClick={onCancel}
            disabled={cancelDisabled}
            style={{
              width: "100%",
              marginTop: "6px",
              padding: "6px",
              borderRadius: "6px",
              border: `1px solid ${C.danger}`,
              background: cancelDisabled
                ? "#F1F5F9"
                : C.dangerSoft,
              color: cancelDisabled
                ? C.textMuted
                : C.danger,
              fontSize: "9px",
              fontWeight: 700,
              cursor: cancelDisabled
                ? "not-allowed"
                : "pointer",
            }}
          >
            {busyKey ===
            `cancel:${relation.id}`
              ? "正在取消…"
              : "取消此关系"}
          </button>
        </>
      )}
    </div>
  );
}

function RelationEndpointRow({
  label,
  item,
  fallbackID,
  onSelectPage,
}: {
  label: string;
  item?: CWAIReviewItem;
  fallbackID: string;
  onSelectPage: (pageNumber: number) => void;
}) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: "5px",
        marginTop: "6px",
        color: C.textSec,
        fontSize: "9px",
      }}
    >
      <strong>{label}：</strong>

      {item &&
        item.page_number_snapshot > 0 && (
          <button
            type="button"
            onClick={() =>
              onSelectPage(
                item.page_number_snapshot,
              )
            }
            style={{
              padding: "1px 5px",
              borderRadius: "4px",
              border: `1px solid ${C.primary}`,
              background: "#fff",
              color: C.primary,
              fontSize: "8px",
              fontWeight: 700,
              cursor: "pointer",
            }}
          >
            P{item.page_number_snapshot}
          </button>
        )}

      <span
        style={{
          minWidth: 0,
          flex: 1,
          wordBreak: "break-word",
        }}
      >
        {item
          ? itemOptionLabel(item)
          : fallbackID}
      </span>
    </div>
  );
}

const inputStyle: CSSProperties = {
  width: "100%",
  boxSizing: "border-box",
  padding: "7px 8px",
  borderRadius: "7px",
  border: `1px solid ${C.border}`,
  background: "#fff",
  color: C.text,
  fontFamily: "inherit",
  fontSize: "9px",
  outline: "none",
};

const eventBoxStyle: CSSProperties = {
  marginTop: "7px",
  padding: "6px",
  borderRadius: "6px",
  background: C.bg,
};

const eventLineStyle: CSSProperties = {
  marginBottom: "3px",
  color: C.textMuted,
  fontSize: "8px",
  lineHeight: 1.5,
};
