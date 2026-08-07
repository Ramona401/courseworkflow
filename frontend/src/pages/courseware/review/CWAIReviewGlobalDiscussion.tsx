/**
 * CWAIReviewGlobalDiscussion.tsx
 *
 * 已完成课件AI审核会话的跨页面、跨问题全局讨论状态编排。
 *
 * 统一问题工作台可以通过selectionRequest带入2至12条问题。
 * 带入动作只改变本组件的临时讨论选择，不改变正式交付清单。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  adoptCWAIReviewGlobalProposal,
  getCWAIReviewGlobalDiscussion,
  messageCWAIReviewGlobalDiscussion,
  type CWAIReviewGlobalDiscussion as CWAIReviewGlobalDiscussionData,
  type CWAIReviewGlobalProposal,
  type CWAIReviewItem,
  type CWAIReviewItemRelation,
} from "@/api/coursewares";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
  CW_GLOBAL_DISCUSSION_EMPTY,
  cwGlobalSecondaryButtonStyle,
  isCWGlobalDiscussionActionableItem,
  sortCWGlobalDiscussionItems,
} from "./CWAIReviewGlobalDiscussion.shared";
import CWAIReviewGlobalDiscussionPicker from "./CWAIReviewGlobalDiscussionPicker";
import CWAIReviewGlobalDiscussionResults from "./CWAIReviewGlobalDiscussionResults";
import CWAIReviewGlobalGovernanceActions from "./CWAIReviewGlobalGovernanceActions";

export interface CWAIReviewGlobalDiscussionProps {
  sessionId: string;
  mode: "formal" | "self";
  items: CWAIReviewItem[];
  governanceRelations: CWAIReviewItemRelation[];

  selectionRequest?: {
    id: number;
    itemIds: string[];
  } | null;

  onSelectPage: (
    pageNumber: number,
  ) => void;

  onItemChanged: (
    item: CWAIReviewItem,
  ) => void;

  onGovernanceRelationsChanged: (
    relations: CWAIReviewItemRelation[],
  ) => void;
}

function normalizeDiscussion(
  discussion: CWAIReviewGlobalDiscussionData,
): CWAIReviewGlobalDiscussionData {
  return {
    messages:
      discussion.messages || [],
    summary:
      discussion.summary || "",
    relations:
      discussion.relations || [],
    proposals:
      discussion.proposals || [],
    selected_item_ids:
      discussion.selected_item_ids || [],
    latest_message_id:
      discussion.latest_message_id || "",
    governance_relations:
      discussion.governance_relations || [],
  };
}

export default function CWAIReviewGlobalDiscussion({
  sessionId,
  mode,
  items,
  governanceRelations,
  selectionRequest,
  onSelectPage,
  onItemChanged,
  onGovernanceRelationsChanged,
}: CWAIReviewGlobalDiscussionProps) {
  const [open, setOpen] =
    useState(false);

  const [
    loadedSessionId,
    setLoadedSessionId,
  ] = useState("");

  const [discussion, setDiscussion] =
    useState<CWAIReviewGlobalDiscussionData>({
      ...CW_GLOBAL_DISCUSSION_EMPTY,
    });

  const [
    selectedItemIds,
    setSelectedItemIds,
  ] = useState<string[]>([]);

  const [content, setContent] =
    useState("");

  const [loading, setLoading] =
    useState(false);

  const [sending, setSending] =
    useState(false);

  const [
    adoptingItemId,
    setAdoptingItemId,
  ] = useState("");

  const [
    adoptedItemIds,
    setAdoptedItemIds,
  ] = useState<string[]>([]);

  const [error, setError] =
    useState("");

  const [message, setMessage] =
    useState("");

  const processedSelectionRequestRef =
    useRef(0);

  const sessionItems =
    useMemo(
      () =>
        items.filter(
          (item) =>
            item.source_session_id ===
            sessionId,
        ),
      [
        items,
        sessionId,
      ],
    );

  const actionableItems =
    useMemo(
      () =>
        sortCWGlobalDiscussionItems(
          sessionItems.filter(
            isCWGlobalDiscussionActionableItem,
          ),
        ),
      [sessionItems],
    );

  const actionableItemIDSet =
    useMemo(
      () =>
        new Set(
          actionableItems.map(
            (item) => item.id,
          ),
        ),
      [actionableItems],
    );

  const itemMap =
    useMemo(
      () =>
        new Map(
          sessionItems.map(
            (item) => [
              item.id,
              item,
            ]),
        ),
      [sessionItems],
    );

  useEffect(() => {
    setOpen(false);
    setLoadedSessionId("");
    setDiscussion({
      ...CW_GLOBAL_DISCUSSION_EMPTY,
    });
    setSelectedItemIds([]);
    setContent("");
    setAdoptedItemIds([]);
    setError("");
    setMessage("");

    processedSelectionRequestRef.current =
      0;

    onGovernanceRelationsChanged([]);
  }, [
    onGovernanceRelationsChanged,
    sessionId,
  ]);

  /**
   * 接收统一问题工作台的一次性选择请求。
   *
   * 使用递增请求ID防止整改项状态更新时重复覆盖用户在讨论区内的调整。
   */
  useEffect(() => {
    const requestID =
      selectionRequest?.id || 0;

    if (
      requestID <= 0 ||
      requestID <=
        processedSelectionRequestRef.current
    ) {
      return;
    }

    processedSelectionRequestRef.current =
      requestID;

    const nextItemIDs =
      (
        selectionRequest?.itemIds ||
        []
      )
        .filter(
          (itemID) =>
            actionableItemIDSet.has(
              itemID,
            ),
        )
        .slice(0, 12);

    setOpen(true);
    setError("");

    if (nextItemIDs.length < 2) {
      setSelectedItemIds([]);
      setMessage("");
      setError(
        "带入的问题不足2条，请回到问题工作台重新选择",
      );
      return;
    }

    setSelectedItemIds(
      nextItemIDs,
    );

    setMessage(
      `已从问题工作台带入${nextItemIDs.length}条问题；本选择不会改变正式交付清单。`,
    );
  }, [
    actionableItemIDSet,
    selectionRequest,
  ]);

  useEffect(() => {
    const scopedRelations =
      governanceRelations.filter(
        (relation) =>
          relation.source_session_id ===
          sessionId,
      );

    setDiscussion(
      (previous) => ({
        ...previous,
        governance_relations:
          scopedRelations,
      }),
    );
  }, [
    governanceRelations,
    sessionId,
  ]);

  useEffect(() => {
    setSelectedItemIds(
      (previous) =>
        previous.filter(
          (itemID) =>
            actionableItemIDSet.has(
              itemID,
            ),
        ),
    );
  }, [actionableItemIDSet]);

  const loadDiscussion =
    useCallback(
      async (
        force = false,
      ) => {
        if (
          !sessionId ||
          loading ||
          (
            !force &&
            loadedSessionId ===
              sessionId
          )
        ) {
          return;
        }

        setLoading(true);
        setError("");
        setMessage("");

        try {
          const result =
            await getCWAIReviewGlobalDiscussion(
              sessionId,
            );

          const normalized =
            normalizeDiscussion(
              result,
            );

          setDiscussion(
            normalized,
          );

          onGovernanceRelationsChanged(
            normalized
              .governance_relations,
          );

          const restorableIDs =
            normalized
              .selected_item_ids
              .filter(
                (itemID) =>
                  actionableItemIDSet.has(
                    itemID,
                  ),
              );

          if (
            restorableIDs.length >
            0
          ) {
            setSelectedItemIds(
              restorableIDs,
            );
          }

          setLoadedSessionId(
            sessionId,
          );
        } catch (cause) {
          setError(
            cause instanceof Error
              ? cause.message
              : "加载全局讨论失败",
          );
        } finally {
          setLoading(false);
        }
      },
      [
        actionableItemIDSet,
        loadedSessionId,
        loading,
        onGovernanceRelationsChanged,
        sessionId,
      ],
    );

  useEffect(() => {
    void loadDiscussion();
  }, [loadDiscussion]);

  const handleGovernanceRelationsChanged =
    useCallback(
      (
        relations:
          CWAIReviewItemRelation[],
      ) => {
        setDiscussion(
          (previous) => ({
            ...previous,
            governance_relations:
              relations,
          }),
        );

        onGovernanceRelationsChanged(
          relations,
        );
      },
      [
        onGovernanceRelationsChanged,
      ],
    );

  const handleManualItemsCreated =
    useCallback(
      (
        createdItems:
          CWAIReviewItem[],
      ) => {
        for (
          const item of
          createdItems
        ) {
          onItemChanged(item);
        }

        setMessage(
          createdItems.length > 1
            ? `已人工新增${createdItems.length}条页级整改项，候选指令仍需逐条独立确认。`
            : "已人工新增整改项，候选指令仍需独立确认。",
        );
      },
      [onItemChanged],
    );


  if (sessionItems.length < 2) {
    return null;
  }

  const handleToggleItem = (
    itemID: string,
    selected: boolean,
  ) => {
    setError("");
    setMessage("");

    if (!selected) {
      setSelectedItemIds(
        (previous) =>
          previous.filter(
            (current) =>
              current !== itemID,
          ),
      );
      return;
    }

    if (
      selectedItemIds.length >=
      12
    ) {
      setError(
        "每轮全局讨论最多选择12条整改项",
      );
      return;
    }

    setSelectedItemIds(
      (previous) =>
        Array.from(
          new Set([
            ...previous,
            itemID,
          ]),
        ),
    );
  };

  const handleSelectAvailable =
    () => {
      setSelectedItemIds(
        actionableItems
          .slice(0, 12)
          .map(
            (item) =>
              item.id,
          ),
      );

      setError("");

      setMessage(
        actionableItems.length > 12
          ? "已选择排序靠前的12条可处理整改项"
          : "已选择全部可处理整改项",
      );
    };

  const handleClearSelection =
    () => {
      setSelectedItemIds([]);
      setError("");
      setMessage("");
    };

  const handleSend =
    async () => {
      const normalizedContent =
        content.trim();

      const contentLength =
        Array.from(
          normalizedContent,
        ).length;

      if (
        selectedItemIds.length < 2 ||
        selectedItemIds.length > 12
      ) {
        setError(
          "请明确选择2至12条整改项",
        );
        return;
      }

      if (
        !normalizedContent ||
        contentLength > 8000
      ) {
        setError(
          contentLength > 8000
            ? "全局讨论内容不能超过8000字"
            : "请填写需要综合分析的问题或要求",
        );
        return;
      }

      if (sending) {
        return;
      }

      setSending(true);
      setError("");
      setMessage("");

      try {
        const result =
          await messageCWAIReviewGlobalDiscussion(
            sessionId,
            normalizedContent,
            selectedItemIds,
          );

        const normalized =
          normalizeDiscussion(
            result,
          );

        setDiscussion(
          normalized,
        );

        onGovernanceRelationsChanged(
          normalized
            .governance_relations,
        );

        setSelectedItemIds(
          normalized
            .selected_item_ids
            .filter(
              (itemID) =>
                actionableItemIDSet.has(
                  itemID,
                ),
            ),
        );

        setContent("");
        setAdoptedItemIds([]);

        setLoadedSessionId(
          sessionId,
        );

        setMessage(
          "全局分析完成。候选指令需逐条采用，并在单条整改项中独立确认。",
        );
      } catch (cause) {
        setError(
          cause instanceof Error
            ? cause.message
            : "全局讨论失败",
        );
      } finally {
        setSending(false);
      }
    };

  const handleAdopt =
    async (
      proposal:
        CWAIReviewGlobalProposal,
    ) => {
      if (
        adoptingItemId ||
        !discussion
          .latest_message_id ||
        !proposal
          .suggested_instruction
          .trim()
      ) {
        return;
      }

      setAdoptingItemId(
        proposal.item_id,
      );

      setError("");
      setMessage("");

      try {
        const result =
          await adoptCWAIReviewGlobalProposal(
            sessionId,
            discussion
              .latest_message_id,
            proposal.item_id,
          );

        onItemChanged(
          result.item,
        );

        setAdoptedItemIds(
          (previous) =>
            Array.from(
              new Set([
                ...previous,
                proposal.item_id,
              ]),
            ),
        );

        setMessage(
          "候选指令已写入对应整改项讨论记录，尚未确认，也未修改页面。",
        );
      } catch (cause) {
        setError(
          cause instanceof Error
            ? cause.message
            : "采用候选指令失败",
        );
      } finally {
        setAdoptingItemId("");
      }
    };

  const modeLabel =
    mode === "self"
      ? "自审问题"
      : "审核问题";

  return (
    <section
      style={{
        padding: "12px",
        borderRadius: "10px",
        border:
          `1px solid ${C.primary}45`,
        background: C.card,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems:
            "flex-start",
          gap: "9px",
        }}
      >
        <div
          style={{
            minWidth: 0,
            flex: 1,
          }}
        >
          <div
            style={{
              color: C.text,
              fontSize: "13px",
              fontWeight: 700,
            }}
          >
            🧩 跨页面、跨问题全局讨论
          </div>

          <div
            style={{
              marginTop: "3px",
              color: C.textSec,
              fontSize: "10px",
              lineHeight: 1.6,
            }}
          >
            同时选择多条
            {modeLabel}
            ，识别重复、冲突、依赖和可合并关系。
            采用结果仍需逐条独立确认。
          </div>
        </div>

        <button
          type="button"
          onClick={() =>
            setOpen(
              (previous) =>
                !previous,
            )
          }
          style={
            cwGlobalSecondaryButtonStyle
          }
        >
          {open
            ? "收起"
            : discussion
                .messages
                .length > 0
              ? "继续讨论"
              : "开始讨论"}
        </button>
      </div>

      {!open &&
        discussion.summary && (
        <div
          style={{
            marginTop: "8px",
            padding: "8px 9px",
            borderRadius: "7px",
            background:
              C.primarySoft,
            color: C.textSec,
            fontSize: "10px",
            lineHeight: 1.6,
          }}
        >
          最近结论：
          {discussion.summary}
        </div>
      )}

      {open && (
        <div
          style={{
            marginTop: "10px",
            paddingTop: "10px",
            borderTop:
              `1px solid ${C.border}`,
          }}
        >
          <CWAIReviewGlobalDiscussionPicker
            mode={mode}
            items={
              actionableItems
            }
            selectedItemIds={
              selectedItemIds
            }
            content={content}
            loading={loading}
            sending={sending}
            error={error}
            message={message}
            onToggleItem={
              handleToggleItem
            }
            onSelectAvailable={
              handleSelectAvailable
            }
            onClearSelection={
              handleClearSelection
            }
            onRefresh={() =>
              void loadDiscussion(
                true,
              )
            }
            onContentChange={
              setContent
            }
            onSend={() =>
              void handleSend()
            }
            onSelectPage={
              onSelectPage
            }
          />

          <CWAIReviewGlobalDiscussionResults
            discussion={
              discussion
            }
            itemMap={itemMap}
            adoptedItemIds={
              adoptedItemIds
            }
            adoptingItemId={
              adoptingItemId
            }
            onSelectPage={
              onSelectPage
            }
            onAdopt={(
              proposal,
            ) =>
              void handleAdopt(
                proposal,
              )
            }
          />

          <CWAIReviewGlobalGovernanceActions
            sessionId={
              sessionId
            }
            discussion={
              discussion
            }
            items={
              sessionItems
            }
            onSelectPage={
              onSelectPage
            }
            onItemChanged={
              onItemChanged
            }
            onItemsCreated={
              handleManualItemsCreated
            }
            onGovernanceRelationsChanged={
              handleGovernanceRelationsChanged
            }
          />
        </div>
      )}
    </section>
  );
}
