/**
 * CWAIReviewGlobalDiscussionResults.tsx
 *
 * 跨页面、跨问题全局讨论的结果展示区域。
 *
 * 负责展示：
 *   - 会话级可见讨论历史；
 *   - 本轮综合结论；
 *   - 重复、冲突、依赖、合并和可能连带解决关系；
 *   - V2关系的可信双端方向及待人工确认状态；
 *   - 每条整改项的候选修改指令及明确采用入口。
 *
 * 采用按钮只触发父组件回调，不直接改变整改项状态。
 */

import { useMemo } from "react";

import type {
  CWAIReviewGlobalDiscussion,
  CWAIReviewGlobalProposal,
  CWAIReviewItem,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
  CW_GLOBAL_RECOMMENDATION_CONFIG,
  CW_GLOBAL_RELATION_CONFIG,
  cwGlobalPageButtonStyle,
  resolveCWGlobalItemPageLabel,
  resolveCWGlobalItemTitle,
} from "./CWAIReviewGlobalDiscussion.shared";

export interface CWAIReviewGlobalDiscussionResultsProps {
  discussion: CWAIReviewGlobalDiscussion;
  itemMap: ReadonlyMap<string, CWAIReviewItem>;

  adoptedItemIds: string[];
  adoptingItemId: string;

  onSelectPage: (pageNumber: number) => void;
  onAdopt: (proposal: CWAIReviewGlobalProposal) => void;
}

/**
 * 兼容早期只返回item_ids的历史消息。
 *
 * 新V2消息必须同时返回source_item_id和target_item_id；
 * 历史消息仍可展示，但关系持久化会由治理操作区关闭。
 */
function resolveRelationEndpointIDs(
  relation: CWAIReviewGlobalDiscussion["relations"][number],
): [string, string] {
  const sourceItemID =
    relation.source_item_id?.trim() || "";
  const targetItemID =
    relation.target_item_id?.trim() || "";

  if (sourceItemID && targetItemID) {
    return [sourceItemID, targetItemID];
  }

  const itemIDs = (relation.item_ids || [])
    .map((itemID) => itemID.trim())
    .filter(Boolean);

  return [
    itemIDs[0] || "",
    itemIDs[1] || "",
  ];
}

function resolveRelationDirectionText(
  relation: CWAIReviewGlobalDiscussion["relations"][number],
): string {
  switch (relation.type) {
    case "duplicate":
      return "源问题重复目标问题；目标问题为保留主问题";
    case "merge":
      return "源问题合并进入目标问题";
    case "dependency":
      return "源问题依赖目标问题先完成";
    case "possibly_resolved":
      return "源问题可能被目标问题的修改连带解决";
    case "conflict":
      return "无方向冲突；双端均需人工裁决";
  }
}

function resolveRelationItemLabel(
  itemMap: ReadonlyMap<string, CWAIReviewItem>,
  itemID: string,
): string {
  const item = itemMap.get(itemID);

  return item
    ? `${resolveCWGlobalItemPageLabel(item)} · ${resolveCWGlobalItemTitle(item)}`
    : itemID || "未提供端点";
}

export default function CWAIReviewGlobalDiscussionResults({
  discussion,
  itemMap,
  adoptedItemIds,
  adoptingItemId,
  onSelectPage,
  onAdopt,
}: CWAIReviewGlobalDiscussionResultsProps) {
  const adoptedItemIDSet = useMemo(
    () => new Set(adoptedItemIds),
    [adoptedItemIds],
  );

  const hasContent =
    discussion.messages.length > 0 ||
    !!discussion.summary ||
    discussion.relations.length > 0 ||
    discussion.proposals.length > 0;

  if (!hasContent) {
    return null;
  }

  return (
    <div
      style={{
        marginTop: "12px",
        paddingTop: "10px",
        borderTop: `1px solid ${C.border}`,
      }}
    >
      {discussion.messages.length > 0 && (
        <DiscussionHistory
          discussion={discussion}
        />
      )}

      {discussion.summary && (
        <div
          style={{
            marginTop: "10px",
            padding: "9px 10px",
            borderRadius: "8px",
            background: C.primarySoft,
            border: "1px solid #C7D2FE",
            color: C.text,
          }}
        >
          <div
            style={{
              marginBottom: "5px",
              color: C.primary,
              fontSize: "10px",
              fontWeight: 700,
            }}
          >
            本轮综合结论
          </div>

          <DiscussionMarkdown
            content={discussion.summary}
            compact
          />
        </div>
      )}

      {discussion.relations.length > 0 && (
        <div style={{ marginTop: "10px" }}>
          <div
            style={{
              color: C.text,
              fontSize: "11px",
              fontWeight: 700,
              marginBottom: "6px",
            }}
          >
            问题关系
          </div>

          {discussion.relations.map((relation, index) => {
            const config =
              CW_GLOBAL_RELATION_CONFIG[relation.type];
            const [
              sourceItemID,
              targetItemID,
            ] = resolveRelationEndpointIDs(relation);
            const sourceLabel =
              resolveRelationItemLabel(
                itemMap,
                sourceItemID,
              );
            const targetLabel =
              resolveRelationItemLabel(
                itemMap,
                targetItemID,
              );
            const directionSymbol =
              relation.type === "conflict"
                ? "↔"
                : "→";

            return (
              <div
                key={[
                  relation.type,
                  sourceItemID,
                  targetItemID,
                  index,
                ].join("-")}
                style={{
                  marginBottom: "6px",
                  padding: "8px 9px",
                  borderRadius: "7px",
                  background: config.background,
                  border: `1px solid ${config.color}30`,
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
                      display: "inline-block",
                      padding: "2px 6px",
                      borderRadius: "5px",
                      background: "#FFFFFF",
                      color: config.color,
                      fontSize: "9px",
                      fontWeight: 700,
                    }}
                  >
                    {config.label}
                  </span>

                  <span
                    style={{
                      color: C.warning,
                      fontSize: "9px",
                      fontWeight: 700,
                    }}
                  >
                    AI建议 · 待独立确认
                  </span>
                </div>

                <div
                  style={{
                    display: "grid",
                    gridTemplateColumns: "1fr auto 1fr",
                    alignItems: "center",
                    gap: "6px",
                    marginTop: "6px",
                  }}
                >
                  <RelationEndpointLabel
                    roleLabel={
                      relation.type === "conflict"
                        ? "端点A"
                        : "源问题"
                    }
                    content={sourceLabel}
                  />

                  <span
                    style={{
                      color: config.color,
                      fontSize: "12px",
                      fontWeight: 700,
                    }}
                  >
                    {directionSymbol}
                  </span>

                  <RelationEndpointLabel
                    roleLabel={
                      relation.type === "conflict"
                        ? "端点B"
                        : "目标问题"
                    }
                    content={targetLabel}
                  />
                </div>

                <div
                  style={{
                    marginTop: "5px",
                    color: C.textSec,
                    fontSize: "9px",
                    lineHeight: 1.5,
                  }}
                >
                  方向：{resolveRelationDirectionText(relation)}
                </div>

                <div
                  style={{
                    marginTop: "4px",
                    color: C.text,
                  }}
                >
                  <DiscussionMarkdown
                    content={relation.explanation}
                    compact
                  />
                </div>
              </div>
            );
          })}
        </div>
      )}

      {discussion.proposals.length > 0 && (
        <div style={{ marginTop: "10px" }}>
          <div
            style={{
              color: C.text,
              fontSize: "11px",
              fontWeight: 700,
              marginBottom: "6px",
            }}
          >
            逐项候选指令
          </div>

          {discussion.proposals.map((proposal) => (
            <ProposalCard
              key={proposal.item_id}
              proposal={proposal}
              item={itemMap.get(proposal.item_id)}
              adopted={adoptedItemIDSet.has(proposal.item_id)}
              adopting={adoptingItemId === proposal.item_id}
              disabled={
                !!adoptingItemId ||
                !discussion.latest_message_id
              }
              onSelectPage={onSelectPage}
              onAdopt={() => onAdopt(proposal)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function RelationEndpointLabel({
  roleLabel,
  content,
}: {
  roleLabel: string;
  content: string;
}) {
  return (
    <div
      style={{
        minWidth: 0,
        padding: "6px",
        borderRadius: "6px",
        background: "#FFFFFF",
        border: `1px solid ${C.border}`,
      }}
    >
      <div
        style={{
          color: C.textMuted,
          fontSize: "8px",
          fontWeight: 700,
        }}
      >
        {roleLabel}
      </div>

      <div
        style={{
          marginTop: "2px",
          color: C.textSec,
          fontSize: "9px",
          lineHeight: 1.4,
          wordBreak: "break-word",
        }}
      >
        {content}
      </div>
    </div>
  );
}

function DiscussionHistory({
  discussion,
}: {
  discussion: CWAIReviewGlobalDiscussion;
}) {
  return (
    <div>
      <div
        style={{
          color: C.text,
          fontSize: "11px",
          fontWeight: 700,
          marginBottom: "7px",
        }}
      >
        全局讨论历史
      </div>

      <div
        style={{
          maxHeight: "240px",
          overflowY: "auto",
        }}
      >
        {discussion.messages.map((message) => (
          <div
            key={message.id}
            style={{
              marginBottom: "7px",
              padding: "8px 9px",
              borderRadius: "8px",
              background:
                message.role === "assistant"
                  ? C.primarySoft
                  : message.role === "system"
                    ? "#F1F5F9"
                    : "#FFFFFF",
              border: `1px solid ${
                message.role === "assistant"
                  ? "#C7D2FE"
                  : C.border
              }`,
            }}
          >
            <div
              style={{
                marginBottom: "4px",
                color:
                  message.role === "assistant"
                    ? C.primary
                    : C.textSec,
                fontSize: "9px",
                fontWeight: 700,
              }}
            >
              {message.role === "assistant"
                ? "AI全局顾问"
                : message.role === "system"
                  ? "操作记录"
                  : "我"}
            </div>

            <DiscussionMarkdown
              content={message.content}
              compact
            />
          </div>
        ))}
      </div>
    </div>
  );
}

function ProposalCard({
  proposal,
  item,
  adopted,
  adopting,
  disabled,
  onSelectPage,
  onAdopt,
}: {
  proposal: CWAIReviewGlobalProposal;
  item?: CWAIReviewItem;
  adopted: boolean;
  adopting: boolean;
  disabled: boolean;
  onSelectPage: (pageNumber: number) => void;
  onAdopt: () => void;
}) {
  const recommendation =
    CW_GLOBAL_RECOMMENDATION_CONFIG[proposal.recommendation];

  const hasInstruction =
    !!proposal.suggested_instruction.trim();

  const adoptDisabled =
    disabled ||
    adopted ||
    adopting ||
    !hasInstruction;

  return (
    <div
      style={{
        marginBottom: "8px",
        padding: "9px",
        borderRadius: "8px",
        border: `1px solid ${recommendation.color}35`,
        background: C.card,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: "6px",
        }}
      >
        <div style={{ minWidth: 0, flex: 1 }}>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "5px",
              flexWrap: "wrap",
            }}
          >
            {item && item.page_number_snapshot > 0 ? (
              <button
                type="button"
                onClick={() =>
                  onSelectPage(item.page_number_snapshot)
                }
                style={{
                  ...cwGlobalPageButtonStyle,
                  color: C.primary,
                }}
              >
                {resolveCWGlobalItemPageLabel(item)}
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
                padding: "2px 6px",
                borderRadius: "5px",
                background: recommendation.background,
                color: recommendation.color,
                fontSize: "9px",
                fontWeight: 700,
              }}
            >
              {recommendation.label}
            </span>
          </div>

          <div
            style={{
              marginTop: "5px",
              color: C.text,
              fontSize: "10px",
              fontWeight: 700,
              lineHeight: 1.5,
            }}
          >
            {item
              ? resolveCWGlobalItemTitle(item)
              : proposal.item_id}
          </div>
        </div>
      </div>

      {proposal.reason && (
        <div
          style={{
            marginTop: "6px",
            color: C.textSec,
          }}
        >
          <strong style={{ fontSize: "9px" }}>
            分析理由：
          </strong>

          <DiscussionMarkdown
            content={proposal.reason}
            compact
          />
        </div>
      )}

      {hasInstruction ? (
        <div
          style={{
            marginTop: "7px",
            padding: "8px",
            borderRadius: "7px",
            background: C.bg,
            border: `1px solid ${C.border}`,
            color: C.text,
          }}
        >
          <div
            style={{
              marginBottom: "4px",
              color: C.primary,
              fontSize: "9px",
              fontWeight: 700,
            }}
          >
            候选修改指令
          </div>

          <DiscussionMarkdown
            content={proposal.suggested_instruction}
            compact
          />
        </div>
      ) : (
        <div
          style={{
            marginTop: "7px",
            padding: "7px 8px",
            borderRadius: "7px",
            background: recommendation.background,
            color: recommendation.color,
            fontSize: "9px",
            lineHeight: 1.5,
          }}
        >
          本建议没有可直接采用的修改指令，请先人工复核或回到单条整改项继续讨论。
        </div>
      )}

      <button
        type="button"
        onClick={onAdopt}
        disabled={adoptDisabled}
        style={{
          width: "100%",
          marginTop: "8px",
          padding: "7px",
          borderRadius: "7px",
          border: adopted
            ? `1px solid ${C.success}`
            : `1px solid ${C.primary}`,
          background: adopted
            ? C.successSoft
            : adoptDisabled
              ? "#F1F5F9"
              : C.primarySoft,
          color: adopted
            ? C.success
            : adoptDisabled
              ? C.textMuted
              : C.primary,
          fontSize: "10px",
          fontWeight: 700,
          cursor: adoptDisabled ? "not-allowed" : "pointer",
        }}
      >
        {adopted
          ? "✓ 已写入单条整改讨论"
          : adopting
            ? "正在采用…"
            : "采用为单条候选指令"}
      </button>

      <div
        style={{
          marginTop: "5px",
          color: C.textMuted,
          fontSize: "9px",
          lineHeight: 1.5,
          textAlign: "center",
        }}
      >
        采用后仍需进入对应整改项独立确认，不会自动修改页面。
      </div>
    </div>
  );
}
