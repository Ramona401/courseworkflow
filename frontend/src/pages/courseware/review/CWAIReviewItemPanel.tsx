/**
 * CWAIReviewItemPanel.tsx
 *
 * 单条课件问题的状态与操作容器。
 *
 * R-01.1职责：
 *   - 解析正式审核、自审和作者整改场景；
 *   - 计算当前问题可用能力；
 *   - 选择唯一主要操作；
 *   - 把教师可见问题事实交给TeacherImprovementCard；
 *   - 把展开后的关系、状态操作和讨论交给CWAIReviewItemDetails；
 *   - AI成功只通过短时Toast反馈。
 *
 * 正式审核的交付选择已经并入唯一主要操作，
 * 不再同时展示独立复选框制造第二个“确认”入口。
 */

import { useMemo } from "react";

import type {
  CWAIReviewItem,
  CWAIReviewItemRelation,
} from "@/api/coursewares";

import CWAIReviewItemDetails from "./CWAIReviewItemDetails";
import CWAIReviewItemFeedback from "./CWAIReviewItemFeedback";
import CWAIReviewSelfAppliedActions from "./CWAIReviewSelfAppliedActions";
import {
  resolveCWAIReviewItemPrimaryAction,
} from "./CWAIReviewItemPrimaryAction";
import {
  canPauseCWAIReviewItem,
  canPrepareCWAIReviewItemModification,
  canResumeCWAIReviewItem,
  cwAIReviewPrimaryButtonStyle,
  cwAIReviewSecondaryButtonStyle,
  resolveCWAIReviewItemExperience,
  resolveCWAIReviewItemNextStep,
  resolveCWAIReviewItemSourceLabel,
} from "./CWAIReviewItemPresentation.shared";
import TeacherImprovementCard from "./TeacherImprovementCard";
import {
  useCWAIReviewItemPanelController,
} from "./useCWAIReviewItemPanelController";
import CWReviewToast from "./CWReviewToast";

export interface CWAIReviewItemPanelProps {
  item: CWAIReviewItem;
  allItems?: CWAIReviewItem[];
  governanceRelations?: CWAIReviewItemRelation[];

  /** 正式审核场景是否允许把问题加入本轮修改清单。 */
  selectable?: boolean;
  selected?: boolean;
  onSelectedChange?: (
    itemID: string,
    selected: boolean,
  ) => void;

  onSelectPage: (
    pageNumber: number,
  ) => void;

  onChanged: (
    item: CWAIReviewItem,
  ) => void;

  /** 只改变展示方式，不改变状态、权限或API行为。 */
  forceDetailsOpen?: boolean;

  /** 作者自审或正式整改时，把确认方案带入页面微调草稿。 */
  onInjectToRefine?: (
    item: CWAIReviewItem,
  ) => void;
}

export default function CWAIReviewItemPanel({
  item,
  allItems,
  governanceRelations,
  selectable = false,
  selected = false,
  onSelectedChange,
  onSelectPage,
  onChanged,
  forceDetailsOpen = false,
  onInjectToRefine,
}: CWAIReviewItemPanelProps) {
  const experience =
    resolveCWAIReviewItemExperience(
      item,
      selectable,
      !!onInjectToRefine,
    );

  const itemMap =
    useMemo(() => {
      const result =
        new Map<
          string,
          CWAIReviewItem
        >();

      for (
        const current of
        allItems || []
      ) {
        result.set(
          current.id,
          current,
        );
      }

      result.set(
        item.id,
        item,
      );

      return result;
    }, [
      allItems,
      item,
    ]);

  const activeRelations =
    useMemo(
      () =>
        (
          governanceRelations ||
          []
        ).filter(
          (relation) =>
            relation.status ===
              "active" &&
            (
              relation
                .source_item_id ===
                item.id ||
              relation
                .target_item_id ===
                item.id
            ),
        ),
      [
        governanceRelations,
        item.id,
      ],
    );

  const manuallyAdded =
    item.origin_type ===
      "global_discussion_manual" ||
    item.origin_type ===
      "goal_drift_manual";

  const sourceLabel =
    resolveCWAIReviewItemSourceLabel(
      experience,
      manuallyAdded,
    );

  const pageLabel =
    item.page_number_snapshot > 0
      ? `P${item.page_number_snapshot}`
      : "整课";

  const canOpenPageModification =
    experience !== "review" &&
    !!onInjectToRefine &&
    item.status ===
      "confirmed" &&
    !!item.page_id &&
    item.page_number_snapshot >
      0 &&
    !!item
      .confirmed_instruction
      .trim();

  const selfApplied =
    experience === "self" &&
    item.status === "applied";

  const canContinueSelfAfterApply =
    selfApplied &&
    !!onInjectToRefine &&
    !!item.page_id &&
    item.page_number_snapshot > 0 &&
    !!item.confirmed_instruction.trim();

  const canSelectForReturn =
    experience === "review" &&
    selectable &&
    item.status ===
      "confirmed" &&
    !!item
      .confirmed_instruction
      .trim();

  const canPrepare =
    experience !==
      "remediation" &&
    canPrepareCWAIReviewItemModification(
      item,
    );

  const canPause =
    experience !==
      "remediation" &&
    (
      canPauseCWAIReviewItem(
        item,
      ) ||
      selfApplied
    );

  const canResume =
    experience !==
      "remediation" &&
    canResumeCWAIReviewItem(
      item,
    );

  const nextStep =
    resolveCWAIReviewItemNextStep(
      experience,
      item,
      selected,
      canOpenPageModification,
    );

  const controller =
    useCWAIReviewItemPanelController({
      item,
      experience,
      canPrepare,
      onChanged,
    });

  const detailsVisible =
    forceDetailsOpen ||
    controller.detailsOpen;

  const primaryAction =
    resolveCWAIReviewItemPrimaryAction({
      item,
      experience,
      copy:
        controller.copy,

      stateAction:
        controller
          .stateAction,

      generating:
        controller
          .generating,

      selected,
      canSelectForReturn,
      onSelectedChange,
      canResume,
      canOpenPageModification,
      onInjectToRefine,

      openDetails:
        controller
          .openDetails,

      handleResume:
        controller
          .handleResume,

      handlePrepareModification:
        controller
          .handlePrepareModification,

      handleRecheckItem:
        controller
          .handleRecheckItem,

      handleResolveSelfItem:
        controller
          .handleResolveSelfItem,
    });

  const actionArea = (
    <>
      <button
        type="button"
        onClick={
          primaryAction.onClick
        }
        disabled={
          controller
            .stateBusy
        }
        style={
          cwAIReviewPrimaryButtonStyle(
            primaryAction.tone,
            controller
              .stateBusy,
          )
        }
      >
        {
          primaryAction.label
        }
      </button>

      {selfApplied && (
        <CWAIReviewSelfAppliedActions
          canContinue={
            canContinueSelfAfterApply
          }
          canPause={canPause}
          busy={
            controller.stateBusy
          }
          stateAction={
            controller.stateAction
          }
          showPauseForm={
            controller.showPauseForm
          }
          pauseReason={
            controller.pauseReason
          }
          onContinue={() =>
            onInjectToRefine?.(
              item,
            )
          }
          onTogglePause={
            controller.togglePauseForm
          }
          onPauseReasonChange={
            controller.setPauseReason
          }
          onPause={() =>
            void controller
              .handlePause()
          }
        />
      )}

      {!forceDetailsOpen &&
        (
          !primaryAction
            .opensDetails ||
          controller
            .detailsOpen
        ) && (
          <button
            type="button"
            onClick={
              controller
                .toggleDetails
            }
            style={
              cwAIReviewSecondaryButtonStyle
            }
          >
            {controller
              .detailsOpen
              ? "收起详情"
              : "更多信息"}
          </button>
        )}
    </>
  );

  const feedbackArea = (
    <>
      {controller
        .stateMessage && (
        <CWAIReviewItemFeedback
          type="success"
          content={
            controller
              .stateMessage
          }
        />
      )}

      {controller
        .stateError && (
        <CWAIReviewItemFeedback
          type="error"
          content={
            controller
              .stateError
          }
        />
      )}
    </>
  );

  const detailsArea =
    detailsVisible ? (
      <CWAIReviewItemDetails
        experience={
          experience
        }
        item={item}
        itemMap={
          itemMap
        }
        activeRelations={
          activeRelations
        }
        pageLabel={
          pageLabel
        }
        canPrepare={
          canPrepare
        }
        canPause={
          canPause &&
          !selfApplied
        }
        canResume={
          canResume
        }
        stateBusy={
          controller
            .stateBusy
        }
        generating={
          controller
            .generating
        }
        showPauseForm={
          controller
            .showPauseForm
        }
        pauseReason={
          controller
            .pauseReason
        }
        stateAction={
          controller
            .stateAction
        }
        discussionVersion={
          controller
            .discussionVersion
        }
        onSelectPage={
          onSelectPage
        }
        onChanged={
          onChanged
        }
        onPrepareModification={() =>
          void controller
            .handlePrepareModification()
        }
        onTogglePauseForm={
          controller
            .togglePauseForm
        }
        onPauseReasonChange={
          controller
            .setPauseReason
        }
        onPause={() =>
          void controller
            .handlePause()
        }
        onResume={() =>
          void controller
            .handleResume()
        }
      />
    ) : null;

  return (
    <>
      <TeacherImprovementCard
        experience={
          experience
        }
        item={item}
        activeRelations={
          activeRelations
        }
        sourceLabel={
          sourceLabel
        }
        pageLabel={
          pageLabel
        }
        nextStep={
          nextStep
        }
        selectable={false}
        selected={
          selected
        }
        canSelectForReturn={
          false
        }
        actions={
          actionArea
        }
        feedback={
          feedbackArea
        }
        details={
          detailsArea
        }
      />

      <CWReviewToast
        message={
          controller
            .toastMessage
        }
        onClose={
          controller
            .clearToastMessage
        }
      />
    </>
  );
}
