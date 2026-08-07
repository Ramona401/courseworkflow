/**
 * CWAIReviewDirectRelationGovernance.tsx
 *
 * 问题清单人工直接关系治理的状态与请求编排容器。
 *
 * 统一问题工作台可以通过selectionRequest带入恰好两条问题。
 * 带入动作不会创建关系，用户仍需选择类型、填写说明并明确确认。
 */

import {
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  cancelCWAIReviewItemRelation,
  confirmCWAIReviewManualRelation,
  getCWAIReviewItemRelations,
  type CWAIReviewGlobalRelationType,
  type CWAIReviewItem,
  type CWAIReviewItemRelation,
} from "@/api/coursewares";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
  isCWGlobalDiscussionActionableItem,
  sortCWGlobalDiscussionItems,
} from "./CWAIReviewGlobalDiscussion.shared";
import {
  countCWGlobalRunes,
  CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES,
} from "./CWAIReviewGlobalGovernanceLimits";
import CWAIReviewDirectRelationForm from "./CWAIReviewDirectRelationForm";
import CWAIReviewDirectRelationRecords from "./CWAIReviewDirectRelationRecords";
import {
  buildCWAIReviewDirectRelationKey,
  CWAIReviewDirectRelationFeedback,
  cwAIReviewDirectRelationSecondaryButtonStyle,
  cwAIReviewDirectRelationSummaryBadgeStyle,
  mergeCWAIReviewDirectRelation,
} from "./CWAIReviewDirectRelationGovernance.shared";

export interface CWAIReviewDirectRelationGovernanceProps {
  sessionId: string;
  items: CWAIReviewItem[];
  relations: CWAIReviewItemRelation[];

  selectionRequest?: {
    id: number;
    itemIds: string[];
  } | null;

  onSelectPage: (
    pageNumber: number,
  ) => void;

  onRelationsChanged: (
    relations:
      CWAIReviewItemRelation[],
  ) => void;
}

export default function CWAIReviewDirectRelationGovernance({
  sessionId,
  items,
  relations,
  selectionRequest,
  onSelectPage,
  onRelationsChanged,
}: CWAIReviewDirectRelationGovernanceProps) {
  const [open, setOpen] =
    useState(false);

  const [
    relationType,
    setRelationType,
  ] =
    useState<CWAIReviewGlobalRelationType>(
      "duplicate",
    );

  const [
    sourceItemID,
    setSourceItemID,
  ] = useState("");

  const [
    targetItemID,
    setTargetItemID,
  ] = useState("");

  const [
    explanation,
    setExplanation,
  ] = useState("");

  const [
    cancelReasons,
    setCancelReasons,
  ] =
    useState<
      Record<string, string>
    >({});

  const [loading, setLoading] =
    useState(false);

  const [busyKey, setBusyKey] =
    useState("");

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

  const activeRelationKeySet =
    useMemo(
      () =>
        new Set(
          relations
            .filter(
              (relation) =>
                relation.status ===
                "active",
            )
            .map(
              (relation) =>
                buildCWAIReviewDirectRelationKey(
                  relation.relation_type,
                  relation.source_item_id,
                  relation.target_item_id,
                ),
            ),
        ),
      [relations],
    );

  const normalizedExplanation =
    explanation.trim();

  const explanationLength =
    countCWGlobalRunes(
      normalizedExplanation,
    );

  const selectedRelationKey =
    sourceItemID &&
    targetItemID &&
    sourceItemID !== targetItemID
      ? buildCWAIReviewDirectRelationKey(
          relationType,
          sourceItemID,
          targetItemID,
        )
      : "";

  const relationAlreadyActive =
    !!selectedRelationKey &&
    activeRelationKeySet.has(
      selectedRelationKey,
    );

  const canConfirm =
    !!sessionId &&
    !!sourceItemID &&
    !!targetItemID &&
    sourceItemID !== targetItemID &&
    explanationLength > 0 &&
    explanationLength <=
      CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES &&
    !relationAlreadyActive &&
    !busyKey;

  useEffect(() => {
    setOpen(false);
    setRelationType(
      "duplicate",
    );
    setSourceItemID("");
    setTargetItemID("");
    setExplanation("");
    setCancelReasons({});
    setError("");
    setMessage("");

    processedSelectionRequestRef.current =
      0;
  }, [sessionId]);

  /**
   * 接收统一问题工作台的一次性两端选择。
   *
   * 只预填关系端点，不自动确认关系。
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
      ).filter(
        (itemID) =>
          actionableItemIDSet.has(
            itemID,
          ),
      );

    setOpen(true);
    setError("");

    if (
      nextItemIDs.length !== 2
    ) {
      setSourceItemID("");
      setTargetItemID("");
      setMessage("");
      setError(
        "建立关系需要两条仍可处理的问题，请回到问题工作台重新选择",
      );
      return;
    }

    setSourceItemID(
      nextItemIDs[0],
    );

    setTargetItemID(
      nextItemIDs[1],
    );

    setMessage(
      "已从问题工作台带入两条问题；请选择关系类型、核对方向并填写说明。",
    );
  }, [
    actionableItemIDSet,
    selectionRequest,
  ]);

  useEffect(() => {
    if (!sessionId) {
      return;
    }

    let active = true;

    setLoading(true);

    void getCWAIReviewItemRelations(
      sessionId,
    )
      .then((result) => {
        if (!active) {
          return;
        }

        onRelationsChanged(
          result.relations || [],
        );
      })
      .catch((cause) => {
        if (!active) {
          return;
        }

        setError(
          cause instanceof Error
            ? cause.message
            : "读取问题关系失败",
        );
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [
    sessionId,
    onRelationsChanged,
  ]);

  useEffect(() => {
    if (
      sourceItemID &&
      !actionableItemIDSet.has(
        sourceItemID,
      )
    ) {
      setSourceItemID("");
    }

    if (
      targetItemID &&
      !actionableItemIDSet.has(
        targetItemID,
      )
    ) {
      setTargetItemID("");
    }
  }, [
    actionableItemIDSet,
    sourceItemID,
    targetItemID,
  ]);

  const clearFeedback =
    () => {
      setError("");
      setMessage("");
    };

  const handleRefresh =
    async () => {
      if (
        !sessionId ||
        loading
      ) {
        return;
      }

      setLoading(true);
      clearFeedback();

      try {
        const result =
          await getCWAIReviewItemRelations(
            sessionId,
          );

        onRelationsChanged(
          result.relations || [],
        );

        setMessage(
          "关系列表已刷新。",
        );
      } catch (cause) {
        setError(
          cause instanceof Error
            ? cause.message
            : "刷新问题关系失败",
        );
      } finally {
        setLoading(false);
      }
    };

  const handleConfirm =
    async () => {
      if (!canConfirm) {
        if (
          relationAlreadyActive
        ) {
          setError(
            "相同关系当前已经有效",
          );
        } else if (
          explanationLength >
          CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
        ) {
          setError(
            `关系说明不能超过${CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES}字`,
          );
        } else {
          setError(
            "请选择两个不同问题并填写人工关系说明",
          );
        }

        return;
      }

      const key =
        selectedRelationKey;

      setBusyKey(
        `confirm:${key}`,
      );

      clearFeedback();

      try {
        const record =
          await confirmCWAIReviewManualRelation(
            sessionId,
            {
              relation_type:
                relationType,
              source_item_id:
                sourceItemID,
              target_item_id:
                targetItemID,
              explanation:
                normalizedExplanation,
            },
          );

        onRelationsChanged(
          mergeCWAIReviewDirectRelation(
            relations,
            record,
          ),
        );

        setSourceItemID("");
        setTargetItemID("");
        setExplanation("");

        setMessage(
          record.version > 1
            ? "关系已重新启用并追加新的版本事件；问题和页面状态未改变。"
            : "关系已由人工明确建立；问题和页面状态未改变。",
        );
      } catch (cause) {
        setError(
          cause instanceof Error
            ? cause.message
            : "建立问题关系失败",
        );
      } finally {
        setBusyKey("");
      }
    };

  const handleCancel =
    async (
      relation:
        CWAIReviewItemRelation,
    ) => {
      const reason =
        (
          cancelReasons[
            relation.id
          ] || ""
        ).trim();

      const reasonLength =
        countCWGlobalRunes(
          reason,
        );

      if (
        !reason ||
        reasonLength >
          CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES ||
        busyKey
      ) {
        setError(
          reasonLength >
            CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
            ? `取消关系原因不能超过${CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES}字`
            : "请填写取消关系原因",
        );

        return;
      }

      setBusyKey(
        `cancel:${relation.id}`,
      );

      clearFeedback();

      try {
        const record =
          await cancelCWAIReviewItemRelation(
            sessionId,
            relation.id,
            reason,
          );

        onRelationsChanged(
          mergeCWAIReviewDirectRelation(
            relations,
            record,
          ),
        );

        setCancelReasons(
          (previous) => ({
            ...previous,
            [relation.id]: "",
          }),
        );

        setMessage(
          "关系已取消，历史事件仍完整保留；整改项和页面状态未改变。",
        );
      } catch (cause) {
        setError(
          cause instanceof Error
            ? cause.message
            : "取消问题关系失败",
        );
      } finally {
        setBusyKey("");
      }
    };

  if (
    sessionItems.length < 2 &&
    relations.length === 0
  ) {
    return null;
  }

  const activeRelationCount =
    relations.filter(
      (relation) =>
        relation.status ===
        "active",
    ).length;

  return (
    <section
      style={{
        padding: "12px",
        borderRadius: "10px",
        border:
          "1px solid #DDD6FE",
        background:
          "#FAF8FF",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems:
            "flex-start",
          gap: "8px",
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
              fontSize: "12px",
              fontWeight: 700,
            }}
          >
            🔗 问题关系
          </div>

          <div
            style={{
              marginTop: "3px",
              color: C.textSec,
              fontSize: "9px",
              lineHeight: 1.55,
            }}
          >
            在问题工作台协作选择两条问题后，可直接带入此处建立关系。
            所有关系仍需人工明确确认。
          </div>
        </div>

        <button
          type="button"
          onClick={() => {
            setOpen(
              (previous) =>
                !previous,
            );

            clearFeedback();
          }}
          style={
            cwAIReviewDirectRelationSecondaryButtonStyle
          }
        >
          {open
            ? "收起"
            : "管理关系"}
        </button>
      </div>

      <div
        style={{
          display: "flex",
          gap: "6px",
          marginTop: "8px",
          flexWrap: "wrap",
        }}
      >
        <span
          style={
            cwAIReviewDirectRelationSummaryBadgeStyle
          }
        >
          有效关系
          {" "}
          {activeRelationCount}
        </span>

        <span
          style={
            cwAIReviewDirectRelationSummaryBadgeStyle
          }
        >
          历史关系
          {" "}
          {relations.length}
        </span>

        <button
          type="button"
          onClick={() =>
            void handleRefresh()
          }
          disabled={loading}
          style={{
            ...cwAIReviewDirectRelationSecondaryButtonStyle,
            padding: "3px 7px",
            cursor:
              loading
                ? "not-allowed"
                : "pointer",
          }}
        >
          {loading
            ? "读取中…"
            : "刷新关系"}
        </button>
      </div>

      {open && (
        <div
          style={{
            marginTop: "10px",
            paddingTop: "10px",
            borderTop:
              `1px solid ${C.border}`,
          }}
        >
          <CWAIReviewDirectRelationForm
            actionableItems={
              actionableItems
            }
            relationType={
              relationType
            }
            sourceItemID={
              sourceItemID
            }
            targetItemID={
              targetItemID
            }
            explanation={
              explanation
            }
            busyKey={busyKey}
            relationAlreadyActive={
              relationAlreadyActive
            }
            canConfirm={
              canConfirm
            }
            onRelationTypeChange={(
              value,
            ) => {
              setRelationType(
                value,
              );

              clearFeedback();
            }}
            onSourceItemIDChange={(
              value,
            ) => {
              setSourceItemID(
                value,
              );

              clearFeedback();
            }}
            onTargetItemIDChange={(
              value,
            ) => {
              setTargetItemID(
                value,
              );

              clearFeedback();
            }}
            onExplanationChange={(
              value,
            ) => {
              setExplanation(
                value,
              );

              clearFeedback();
            }}
            onConfirm={() =>
              void handleConfirm()
            }
          />

          <CWAIReviewDirectRelationRecords
            relations={
              relations
            }
            itemMap={itemMap}
            cancelReasons={
              cancelReasons
            }
            busyKey={busyKey}
            onCancelReasonChange={(
              relationID,
              reason,
            ) =>
              setCancelReasons(
                (previous) => ({
                  ...previous,
                  [relationID]:
                    reason,
                }),
              )
            }
            onSelectPage={
              onSelectPage
            }
            onCancel={(
              relation,
            ) =>
              void handleCancel(
                relation,
              )
            }
          />

          {error && (
            <CWAIReviewDirectRelationFeedback
              type="error"
              content={error}
            />
          )}

          {message && (
            <CWAIReviewDirectRelationFeedback
              type="success"
              content={message}
            />
          )}
        </div>
      )}
    </section>
  );
}
