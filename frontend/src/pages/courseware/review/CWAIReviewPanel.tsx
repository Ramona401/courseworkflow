/**
 * CWAIReviewPanel.tsx
 *
 * 课件AI审核面板编排壳。
 *
 * 状态和整改项API调用位于useCWAIReviewController.ts；
 * 配置与进度卡片位于CWAIReviewProgressCards.tsx；
 * 跨页面、跨问题讨论位于CWAIReviewGlobalDiscussion.tsx；
 * R-06正式问题组位于CWAIReviewItemGroups.tsx；
 * 最终报告、问题工作台和直接关系治理位于CWAIReviewReportView.tsx。
 *
 * 本层明确区分：
 *   1. workSelectedItemIds：临时工作选择；
 *   2. controller.selectedItemIds：正式退回交付选择。
 *
 * R-07 Atomic Apply成功后：
 *   - 重新GET整改项；
 *   - 重新GET持久化关系；
 *   - 重新GET正式问题组；
 *   - 不根据Impact Plan Preview payload乐观伪造业务状态。
 */

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import {
  getCWAIReviewItemGroups,
  getCWAIReviewItemRelations,
  type CWAIReviewItem,
  type CWAIReviewItemGroup,
  type CWAIReviewItemRelation,
} from "@/api/coursewares";

import {
  isCWGlobalDiscussionActionableItem,
} from "./CWAIReviewGlobalDiscussion.shared";

import CWAIReviewGlobalDiscussion from "./CWAIReviewGlobalDiscussion";
import CWAIReviewItemGroups from "./CWAIReviewItemGroups";
import CWAIReviewProgressCards from "./CWAIReviewProgressCards";

import CWAIReviewReportView, {
  type CWAIReviewSelectionRequest,
} from "./CWAIReviewReportView";

import {
  useCWAIReviewController,
  type CWAIReviewContext,
  type CWAIReviewPanelMode,
} from "./useCWAIReviewController";

import {
  useLatestValueRef,
} from "./useLatestValueRef";

export type {
  CWAIReviewContext,
  CWAIReviewPanelMode,
} from "./useCWAIReviewController";

export interface CWAIReviewPanelProps {
  coursewareId: string;
  coursewareTitle: string;
  subject: string;
  grade: string;
  lessonPlanId?: string | null;
  reviewLevel: number;
  mode?: CWAIReviewPanelMode;

  onSelectPage: (
    pageNumber: number,
  ) => void;

  onUseReviewComment?: (
    comment: string,
  ) => void;

  onReviewContextChange?: (
    context: CWAIReviewContext,
  ) => void;

  onInjectToRefine?: (
    item: CWAIReviewItem,
  ) => void;
}

export default function CWAIReviewPanel({
  coursewareId,
  coursewareTitle,
  subject,
  grade,
  lessonPlanId,
  reviewLevel,
  mode = "formal",
  onSelectPage,
  onUseReviewComment,
  onReviewContextChange,
  onInjectToRefine,
}: CWAIReviewPanelProps) {
  const controller =
    useCWAIReviewController({
      coursewareId,
      subject,
      grade,
      reviewLevel,
      mode,
      onReviewContextChange,
    });

  const [
    governanceRelations,
    setGovernanceRelations,
  ] =
    useState<
      CWAIReviewItemRelation[]
    >([]);

  const [
    itemGroups,
    setItemGroups,
  ] =
    useState<
      CWAIReviewItemGroup[]
    >([]);

  const [
    governanceRefreshError,
    setGovernanceRefreshError,
  ] = useState("");

  /**
   * 统一问题工作台的临时工作选择。
   *
   * 该状态不会回传给正式审核决定面板。
   */
  const [
    workSelectedItemIds,
    setWorkSelectedItemIds,
  ] = useState<string[]>([]);

  const [
    discussionSelectionRequest,
    setDiscussionSelectionRequest,
  ] =
    useState<
      CWAIReviewSelectionRequest |
      null
    >(null);

  const [
    relationSelectionRequest,
    setRelationSelectionRequest,
  ] =
    useState<
      CWAIReviewSelectionRequest |
      null
    >(null);

  const selectionRequestSequenceRef =
    useRef(0);

  const completedSession =
    controller.session
      ?.status === "done"
      ? controller.session
      : null;

  const completedSessionID =
    completedSession?.id || "";

  const completedSessionIDRef =
    useLatestValueRef(
      completedSessionID,
    );

  useEffect(() => {
    setGovernanceRelations([]);
    setItemGroups([]);
    setGovernanceRefreshError("");
    setWorkSelectedItemIds([]);

    setDiscussionSelectionRequest(
      null,
    );

    setRelationSelectionRequest(
      null,
    );
  }, [
    controller.session?.id,
  ]);

  /**
   * 会话完成后读取关系和问题组数据库真值。
   *
   * 整改项由useCWAIReviewItemsState负责加载，避免双重所有权。
   */
  useEffect(() => {
    if (!completedSessionID) {
      return;
    }

    let cancelled = false;

    setGovernanceRefreshError("");

    void Promise.all([
      getCWAIReviewItemRelations(
        completedSessionID,
      ),
      getCWAIReviewItemGroups(
        completedSessionID,
      ),
    ])
      .then(
        ([
          relationResult,
          groupResult,
        ]) => {
          if (cancelled) {
            return;
          }

          setGovernanceRelations(
            relationResult.relations || [],
          );

          setItemGroups(
            groupResult.groups || [],
          );
        },
      )
      .catch((cause) => {
        if (cancelled) {
          return;
        }

        setGovernanceRefreshError(
          cause instanceof Error
            ? cause.message
            : "加载问题治理数据失败",
        );
      });

    return () => {
      cancelled = true;
    };
  }, [completedSessionID]);

  /**
   * R-07 Atomic Apply完成后必须重新读取三类数据库真值。
   *
   * 自定义事件只作为“需要刷新”的通知，不携带任何业务对象真值。
   */
  useEffect(() => {
    if (!completedSessionID) {
      return;
    }

    const handleImpactPlanApplied =
      (
        rawEvent: Event,
      ) => {
        const event =
          rawEvent as CustomEvent<{
            sessionId?: string;
            planId?: string;
          }>;

        if (
          event.detail?.sessionId !==
          completedSessionID
        ) {
          return;
        }

        setGovernanceRefreshError("");

        void Promise.all([
          controller
            .refreshSessionItems(
              completedSessionID,
            ),
          getCWAIReviewItemRelations(
            completedSessionID,
          ),
          getCWAIReviewItemGroups(
            completedSessionID,
          ),
        ])
          .then(
            ([
              nextItems,
              relationResult,
              groupResult,
            ]) => {
              if (
                completedSessionIDRef
                  .current !==
                completedSessionID
              ) {
                return;
              }

              setGovernanceRelations(
                relationResult.relations ||
                  [],
              );

              setItemGroups(
                groupResult.groups || [],
              );

              const actionableIDSet =
                new Set(
                  nextItems
                    .filter(
                      isCWGlobalDiscussionActionableItem,
                    )
                    .map(
                      (item) =>
                        item.id,
                    ),
                );

              setWorkSelectedItemIds(
                (previous) =>
                  previous.filter(
                    (itemID) =>
                      actionableIDSet.has(
                        itemID,
                      ),
                  ),
              );

              setDiscussionSelectionRequest(
                null,
              );

              setRelationSelectionRequest(
                null,
              );
            },
          )
          .catch((cause) => {
            if (
              completedSessionIDRef
                .current !==
              completedSessionID
            ) {
              return;
            }

            setGovernanceRefreshError(
              cause instanceof Error
                ? cause.message
                : "影响方案已应用，但刷新问题治理视图失败，请手动刷新",
            );
          });
      };

    window.addEventListener(
      "tedna:cw-ai-review-impact-plan-applied",
      handleImpactPlanApplied,
    );

    return () => {
      window.removeEventListener(
        "tedna:cw-ai-review-impact-plan-applied",
        handleImpactPlanApplied,
      );
    };
  }, [
    completedSessionID,
    completedSessionIDRef,
    controller.refreshSessionItems,
  ]);

  /**
   * 整改项状态变化后，及时移除已经不能参加综合治理的问题。
   */
  useEffect(() => {
    const actionableIDSet =
      new Set(
        controller.items
          .filter(
            isCWGlobalDiscussionActionableItem,
          )
          .map(
            (item) =>
              item.id,
          ),
      );

    setWorkSelectedItemIds(
      (previous) =>
        previous.filter(
          (itemID) =>
            actionableIDSet.has(
              itemID,
            ),
        ),
    );
  }, [controller.items]);

  const handleGovernanceRelationsChanged =
    useCallback(
      (
        relations:
          CWAIReviewItemRelation[],
      ) => {
        setGovernanceRelations(
          relations || [],
        );
      },
      [],
    );

  const handleGroupsChanged =
    useCallback(
      (
        groups:
          CWAIReviewItemGroup[],
      ) => {
        setItemGroups(
          groups || [],
        );
      },
      [],
    );

  const handleToggleWorkSelection =
    useCallback(
      (
        itemID: string,
        selected: boolean,
      ) => {
        setWorkSelectedItemIds(
          (previous) => {
            if (!selected) {
              return previous.filter(
                (currentID) =>
                  currentID !==
                  itemID,
              );
            }

            return Array.from(
              new Set([
                ...previous,
                itemID,
              ]),
            );
          },
        );
      },
      [],
    );

  const handleClearWorkSelection =
    useCallback(() => {
      setWorkSelectedItemIds([]);
    }, []);

  const scrollToWorkspaceSection =
    (
      elementID: string,
    ) => {
      window.setTimeout(() => {
        document
          .getElementById(
            elementID,
          )
          ?.scrollIntoView({
            behavior:
              "smooth",
            block: "start",
          });
      }, 0);
    };

  const handleOpenGlobalDiscussion =
    useCallback(
      () => {
        if (
          workSelectedItemIds
            .length < 2 ||
          workSelectedItemIds
            .length > 12
        ) {
          return;
        }

        selectionRequestSequenceRef
          .current += 1;

        setDiscussionSelectionRequest({
          id:
            selectionRequestSequenceRef
              .current,

          itemIds: [
            ...workSelectedItemIds,
          ],
        });

        scrollToWorkspaceSection(
          "cw-ai-global-discussion",
        );
      },
      [workSelectedItemIds],
    );

  const handleOpenDirectRelation =
    useCallback(
      () => {
        if (
          workSelectedItemIds
            .length !== 2
        ) {
          return;
        }

        selectionRequestSequenceRef
          .current += 1;

        setRelationSelectionRequest({
          id:
            selectionRequestSequenceRef
              .current,

          itemIds: [
            ...workSelectedItemIds,
          ],
        });

        scrollToWorkspaceSection(
          "cw-ai-direct-relation-governance",
        );
      },
      [workSelectedItemIds],
    );

  const controllerItemChangedRef =
    useLatestValueRef(
      controller
        .handleItemChanged,
    );

  const handleItemChanged =
    useCallback(
      (
        item: CWAIReviewItem,
      ) => {
        controllerItemChangedRef
          .current(item);
      },
      [
        controllerItemChangedRef,
      ],
    );

  return (
    <div
      style={{
        display: "flex",
        flexDirection:
          "column",
        gap: "12px",
      }}
    >
      <CWAIReviewProgressCards
        controller={
          controller
        }
        coursewareTitle={
          coursewareTitle
        }
        subject={subject}
        grade={grade}
        lessonPlanId={
          lessonPlanId
        }
      />

      {governanceRefreshError && (
        <div
          style={{
            padding: "8px 10px",
            borderRadius: "8px",
            border:
              "1px solid #FECACA",
            background:
              "#FEF2F2",
            color: "#B91C1C",
            fontSize: "10px",
            lineHeight: 1.5,
          }}
        >
          {governanceRefreshError}
        </div>
      )}

      {!controller.loading &&
        completedSession &&
        controller.items
          .length >= 2 && (
        <div
          id="cw-ai-global-discussion"
        >
          <CWAIReviewGlobalDiscussion
            sessionId={
              completedSession.id
            }
            mode={mode}
            items={
              controller.items
            }
            governanceRelations={
              governanceRelations
            }
            selectionRequest={
              discussionSelectionRequest
            }
            onSelectPage={
              onSelectPage
            }
            onItemChanged={
              handleItemChanged
            }
            onGovernanceRelationsChanged={
              handleGovernanceRelationsChanged
            }
          />
        </div>
      )}

      {!controller.loading &&
        completedSession &&
        controller.items.length >
          0 && (
        <CWAIReviewItemGroups
          sessionId={
            completedSession.id
          }
          items={
            controller.items
          }
          groups={
            itemGroups
          }
          workSelectedItemIds={
            workSelectedItemIds
          }
          onSelectPage={
            onSelectPage
          }
          onClearWorkSelection={
            handleClearWorkSelection
          }
          onGroupsChanged={
            handleGroupsChanged
          }
        />
      )}

      {!controller.loading && (
        <CWAIReviewReportView
          sessionId={
            completedSession
              ?.id || ""
          }
          mode={mode}
          finalReport={
            controller.finalReport
          }
          reviewConfig={
            controller
              .sessionReviewConfig
          }
          batchFindings={
            controller.batchFindings
          }
          items={
            controller.items
          }
          governanceRelations={
            governanceRelations
          }
          workSelectedItemIds={
            workSelectedItemIds
          }
          deliverySelectedItemIds={
            controller
              .selectedItemIds
          }
          materializingFindingIds={
            controller
              .materializingFindingIds
          }
          relationSelectionRequest={
            relationSelectionRequest
          }
          onSelectPage={
            onSelectPage
          }
          onUseReviewComment={
            mode === "self"
              ? onUseReviewComment
              : undefined
          }
          onAdoptFinding={
            controller
              .handleAdoptFinding
          }
          onToggleWorkSelection={
            handleToggleWorkSelection
          }
          onClearWorkSelection={
            handleClearWorkSelection
          }
          onOpenGlobalDiscussion={
            handleOpenGlobalDiscussion
          }
          onOpenDirectRelation={
            handleOpenDirectRelation
          }
          onToggleDeliverySelection={
            controller
              .handleToggleItemSelection
          }
          onItemChanged={
            handleItemChanged
          }
          onGovernanceRelationsChanged={
            handleGovernanceRelationsChanged
          }
          onInjectToRefine={
            onInjectToRefine
          }
        />
      )}
    </div>
  );
}
