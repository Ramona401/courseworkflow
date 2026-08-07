/**
 * CWAIReviewGlobalGovernanceActions.tsx
 *
 * 全局讨论结论的人工治理操作区。
 *
 * 负责：
 *   - 从可信全局讨论消息人工新增整改项；
 *   - 接入关系建议的独立确认、取消和审计历史；
 *   - 独立确认AI的consider_dismiss建议。
 *
 * 本组件不会确认候选修改指令、修改页面或提交人工审核决定。
 */

import {
  useMemo,
  useState,
} from "react";

import {
  confirmCWAIReviewGlobalDismissal,
  type CWAIReviewGlobalDiscussion,
  type CWAIReviewGlobalProposal,
  type CWAIReviewItem,
  type CWAIReviewItemRelation,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
  cwGlobalPageButtonStyle,
  resolveCWGlobalItemTitle,
} from "./CWAIReviewGlobalDiscussion.shared";
import {
  countCWGlobalRunes,
  CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES,
} from "./CWAIReviewGlobalGovernanceLimits";
import CWAIReviewGlobalManualItemForm from "./CWAIReviewGlobalManualItemForm";
import CWAIReviewGlobalRelationGovernance from "./CWAIReviewGlobalRelationGovernance";

export interface CWAIReviewGlobalGovernanceActionsProps {
  sessionId: string;
  discussion: CWAIReviewGlobalDiscussion;
  items: CWAIReviewItem[];

  onSelectPage: (pageNumber: number) => void;
  onItemChanged: (item: CWAIReviewItem) => void;
  onItemsCreated: (items: CWAIReviewItem[]) => void;
  onGovernanceRelationsChanged: (
    relations: CWAIReviewItemRelation[],
  ) => void;
}

export default function CWAIReviewGlobalGovernanceActions({
  sessionId,
  discussion,
  items,
  onSelectPage,
  onItemChanged,
  onItemsCreated,
  onGovernanceRelationsChanged,
}: CWAIReviewGlobalGovernanceActionsProps) {
  const [busyItemID, setBusyItemID] = useState("");
  const [dismissReasons, setDismissReasons] =
    useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const itemMap = useMemo(
    () =>
      new Map(
        items.map((item) => [item.id, item]),
      ),
    [items],
  );

  const dismissProposals = useMemo(
    () =>
      discussion.proposals.filter(
        (proposal) =>
          proposal.recommendation === "consider_dismiss",
      ),
    [discussion.proposals],
  );

  const handleConfirmDismiss = async (
    proposal: CWAIReviewGlobalProposal,
  ) => {
    const reason =
      (dismissReasons[proposal.item_id] || "").trim();
    const reasonLength = countCWGlobalRunes(reason);

    if (
      !discussion.latest_message_id ||
      !reason ||
      reasonLength > CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES ||
      busyItemID
    ) {
      setError(
        reasonLength > CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
          ? `确认忽略原因不能超过${CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES}字`
          : "请填写独立人工复核后的忽略原因",
      );
      return;
    }

    setBusyItemID(proposal.item_id);
    setError("");
    setMessage("");

    try {
      const result =
        await confirmCWAIReviewGlobalDismissal(
          sessionId,
          discussion.latest_message_id,
          proposal.item_id,
          reason,
        );

      onItemChanged(result.item);
      setDismissReasons((previous) => ({
        ...previous,
        [proposal.item_id]: "",
      }));
      setMessage(
        "已独立确认忽略该问题，并保留原因和讨论历史；页面未修改。",
      );
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "确认忽略整改项失败",
      );
    } finally {
      setBusyItemID("");
    }
  };

  return (
    <div
      style={{
        marginTop: "12px",
        paddingTop: "10px",
        borderTop: `1px solid ${C.border}`,
      }}
    >
      <div
        style={{
          color: C.text,
          fontSize: "11px",
          fontWeight: 700,
        }}
      >
        人工治理操作
      </div>

      <div
        style={{
          marginTop: "3px",
          color: C.textMuted,
          fontSize: "9px",
          lineHeight: 1.5,
        }}
      >
        AI关系和忽略建议只供参考。以下每个动作都需要独立人工确认，
        不会自动确认修改指令、修改页面或提交审核决定。
      </div>

      <CWAIReviewGlobalManualItemForm
        sessionId={sessionId}
        messageId={discussion.latest_message_id}
        items={items}
        onSelectPage={onSelectPage}
        onCreated={onItemsCreated}
      />

      <CWAIReviewGlobalRelationGovernance
        sessionId={sessionId}
        messageId={discussion.latest_message_id}
        suggestions={discussion.relations}
        relations={discussion.governance_relations || []}
        items={items}
        onSelectPage={onSelectPage}
        onRelationsChanged={onGovernanceRelationsChanged}
      />

      {dismissProposals.length > 0 && (
        <div style={{ marginTop: "10px" }}>
          <div
            style={{
              marginBottom: "6px",
              color: C.text,
              fontSize: "10px",
              fontWeight: 700,
            }}
          >
            AI“可考虑忽略”建议的独立确认
          </div>

          {dismissProposals.map((proposal) => (
            <DismissProposalCard
              key={proposal.item_id}
              proposal={proposal}
              item={itemMap.get(proposal.item_id)}
              reason={
                dismissReasons[proposal.item_id] || ""
              }
              busy={busyItemID === proposal.item_id}
              disabled={!!busyItemID}
              onReasonChange={(reason) =>
                setDismissReasons((previous) => ({
                  ...previous,
                  [proposal.item_id]: reason,
                }))
              }
              onSelectPage={onSelectPage}
              onConfirm={() =>
                void handleConfirmDismiss(proposal)
              }
            />
          ))}
        </div>
      )}

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
    </div>
  );
}

function DismissProposalCard({
  proposal,
  item,
  reason,
  busy,
  disabled,
  onReasonChange,
  onSelectPage,
  onConfirm,
}: {
  proposal: CWAIReviewGlobalProposal;
  item?: CWAIReviewItem;
  reason: string;
  busy: boolean;
  disabled: boolean;
  onReasonChange: (reason: string) => void;
  onSelectPage: (pageNumber: number) => void;
  onConfirm: () => void;
}) {
  const alreadyDismissed =
    item?.status === "dismissed";
  const reasonLength = countCWGlobalRunes(reason);

  return (
    <div
      style={{
        marginBottom: "7px",
        padding: "8px",
        borderRadius: "7px",
        border: `1px solid ${C.warning}35`,
        background: C.warningSoft,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "5px",
          flexWrap: "wrap",
        }}
      >
        {item && item.page_number_snapshot > 0 && (
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
            P{item.page_number_snapshot}
          </button>
        )}

        <span
          style={{
            color: C.text,
            fontSize: "9px",
            fontWeight: 700,
          }}
        >
          {item
            ? resolveCWGlobalItemTitle(item)
            : proposal.item_id}
        </span>

        {alreadyDismissed && (
          <span
            style={{
              color: C.success,
              fontSize: "9px",
              fontWeight: 700,
            }}
          >
            ✓ 已忽略
          </span>
        )}
      </div>

      <div
        style={{
          marginTop: "5px",
          color: C.textSec,
        }}
      >
        <DiscussionMarkdown
          content={proposal.reason}
          compact
        />
      </div>

      {!alreadyDismissed && (
        <>
          <textarea
            value={reason}
            onChange={(event) =>
              onReasonChange(event.target.value)
            }
            rows={2}
            maxLength={CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES}
            disabled={disabled}
            placeholder="填写人工复核后的忽略原因；不能直接照抄AI结论"
            style={{
              width: "100%",
              boxSizing: "border-box",
              marginTop: "6px",
              padding: "7px 8px",
              borderRadius: "6px",
              border: `1px solid ${C.border}`,
              resize: "vertical",
              fontFamily: "inherit",
              fontSize: "9px",
              lineHeight: 1.5,
            }}
          />

          <div
            style={{
              marginTop: "3px",
              color:
                reasonLength >
                CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                  ? C.danger
                  : C.textMuted,
              fontSize: "8px",
              textAlign: "right",
            }}
          >
            {reasonLength}/{CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES}
          </div>

          <button
            type="button"
            onClick={onConfirm}
            disabled={
              disabled ||
              !reason.trim() ||
              reasonLength >
                CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
            }
            style={{
              width: "100%",
              marginTop: "5px",
              padding: "6px",
              borderRadius: "6px",
              border: `1px solid ${C.warning}`,
              background:
                disabled ||
                !reason.trim() ||
                reasonLength >
                  CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                  ? "#F1F5F9"
                  : "#fff",
              color:
                disabled ||
                !reason.trim() ||
                reasonLength >
                  CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                  ? C.textMuted
                  : C.warning,
              fontSize: "9px",
              fontWeight: 700,
              cursor:
                disabled ||
                !reason.trim() ||
                reasonLength >
                  CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                  ? "not-allowed"
                  : "pointer",
            }}
          >
            {busy
              ? "正在确认忽略…"
              : "独立确认忽略"}
          </button>
        </>
      )}
    </div>
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
        fontSize: "9px",
        fontWeight: 600,
        lineHeight: 1.5,
      }}
    >
      {content}
    </div>
  );
}
