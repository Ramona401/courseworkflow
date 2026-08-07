/**
 * CWAIReviewPanel.tsx
 *
 * 课件AI审核面板编排壳。
 *
 * 状态和API调用位于useCWAIReviewController.ts；
 * 配置与进度卡片位于CWAIReviewProgressCards.tsx；
 * 跨页面、跨问题讨论位于CWAIReviewGlobalDiscussion.tsx；
 * 最终报告、配置摘要、问题工作台和关系治理位于
 * CWAIReviewReportView.tsx。
 *
 * 本层明确区分：
 *   1. workSelectedItemIds：临时工作选择；
 *   2. controller.selectedItemIds：正式退回交付选择。
 *
 * 临时选择不会进入正式审核决定payload。
 */

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import type {
  CWAIReviewItem,
  CWAIReviewItemRelation,
} from "@/api/coursewares";

import {
  isCWGlobalDiscussionActionableItem,
} from "./CWAIReviewGlobalDiscussion.shared";

import CWAIReviewGlobalDiscussion from "./CWAIReviewGlobalDiscussion";
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

  onUseReviewComment: (
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

  useEffect(() => {
    setGovernanceRelations([]);
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

  const completedSession =
    controller.session
      ?.status === "done"
      ? controller.session
      : null;

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
            onUseReviewComment
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
