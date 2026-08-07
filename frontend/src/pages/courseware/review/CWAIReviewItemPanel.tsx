/**
 * CWAIReviewItemPanel.tsx
 *
 * 单条课件问题的状态与操作容器。
 *
 * 页面会根据当前使用场景自动切换主要目标：
 *   - 审核员：明确整改要求并决定是否退回；
 *   - 自审作者：形成修改方案并修改自己的课件；
 *   - 整改作者：理解审核要求并完成修改。
 */

import {
  useMemo,
  useState,
} from "react";

import {
  dismissCWAIReviewItem,
  generateCWAIReviewItemInstruction,
  parseCWAIReviewItemEvidence,
  restoreCWAIReviewItem,
  type CWAIReviewItem,
  type CWAIReviewItemRelation,
} from "@/api/coursewares";

import CWAIReviewItemDetails from "./CWAIReviewItemDetails";
import {
  CW_AI_REVIEW_ITEM_COLORS as C,
  type CWAIReviewItemStateAction,
  type CWAIReviewPrimaryActionTone,
  CWAIReviewItemFeedback,
  canPauseCWAIReviewItem,
  canPrepareCWAIReviewItemModification,
  canResumeCWAIReviewItem,
  cwAIReviewPrimaryButtonStyle,
  cwAIReviewSecondaryButtonStyle,
  resolveCWAIReviewItemExperience,
  resolveCWAIReviewItemExperienceCopy,
  resolveCWAIReviewItemSourceLabel,
  resolveCWAIReviewItemStatus,
} from "./CWAIReviewItemPresentation.shared";
import CWAIReviewItemSummary from "./CWAIReviewItemSummary";
import {
  useCWAIReviewItemResolutionActions,
} from "./useCWAIReviewItemResolutionActions";

export interface CWAIReviewItemPanelProps {
  item: CWAIReviewItem;

  allItems?: CWAIReviewItem[];

  governanceRelations?: CWAIReviewItemRelation[];

  /**
   * 只有审核员的正式审核界面可以决定本次是否退回给作者。
   */
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

  /**
   * 聚焦工作区使用。开启后详情始终展开，并隐藏“更多信息/收起详情”切换按钮。
   *
   * 该属性只改变展示方式，不改变问题状态、操作权限或API调用。
   */
  forceDetailsOpen?: boolean;

  /**
   * 作者自审或作者整改时，允许把修改方案带入页面修改。
   * 审核员正式审核时不会显示直接修改作者课件的入口。
   */
  onInjectToRefine?: (
    item: CWAIReviewItem,
  ) => void;
}

interface PrimaryAction {
  label: string;
  tone: CWAIReviewPrimaryActionTone;
  opensDetails: boolean;
  onClick: () => void;
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
  const [detailsOpen, setDetailsOpen] =
    useState(false);
  const [discussionVersion, setDiscussionVersion] =
    useState(0);
  const [generating, setGenerating] =
    useState(false);
  const [showPauseForm, setShowPauseForm] =
    useState(false);
  const [pauseReason, setPauseReason] =
    useState("");
  const [stateAction, setStateAction] =
    useState<CWAIReviewItemStateAction>(null);
  const [stateError, setStateError] =
    useState("");
  const [stateMessage, setStateMessage] =
    useState("");

  const detailsVisible =
    forceDetailsOpen ||
    detailsOpen;

  const experience =
    resolveCWAIReviewItemExperience(
      item,
      selectable,
      !!onInjectToRefine,
    );

  const copy =
    resolveCWAIReviewItemExperienceCopy(
      experience,
    );

  const evidence = useMemo(
    () =>
      parseCWAIReviewItemEvidence(
        item,
      ),
    [item],
  );

  const itemMap = useMemo(() => {
    const result =
      new Map<string, CWAIReviewItem>();

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
              relation.source_item_id ===
                item.id ||
              relation.target_item_id ===
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
      "global_discussion_manual";

  const sourceLabel =
    resolveCWAIReviewItemSourceLabel(
      experience,
      manuallyAdded,
    );

  const pageLabel =
    item.page_number_snapshot > 0
      ? `P${item.page_number_snapshot}`
      : "整课";

  const evidenceSummary =
    typeof evidence.page_evidence ===
    "string"
      ? evidence.page_evidence
      : "";

  const summaryTitle =
    item.title.trim() ||
    item.description.trim() ||
    "未填写问题标题";

  const summaryDescription =
    item.title.trim()
      ? item.description.trim()
      : "";

  const canOpenPageModification =
    experience !== "review" &&
    !!onInjectToRefine &&
    item.status === "confirmed" &&
    !!item.page_id &&
    item.page_number_snapshot > 0 &&
    !!item.confirmed_instruction.trim();

  const canSelectForReturn =
    experience === "review" &&
    selectable &&
    item.status === "confirmed" &&
    !!item.confirmed_instruction.trim();

  const canPrepare =
    experience !== "remediation" &&
    canPrepareCWAIReviewItemModification(
      item,
    );

  const canPause =
    experience !== "remediation" &&
    canPauseCWAIReviewItem(
      item,
    );

  const canResume =
    experience !== "remediation" &&
    canResumeCWAIReviewItem(
      item,
    );

  const stateBusy =
    generating ||
    stateAction !== null;

  const handlePrepareModification =
    async () => {
      if (
        !canPrepare ||
        stateBusy
      ) {
        return;
      }

      setGenerating(true);
      setStateError("");
      setStateMessage("");

      try {
        const result =
          await generateCWAIReviewItemInstruction(
            item.id,
          );

        onChanged(
          result.item,
        );

        setDiscussionVersion(
          (previous) =>
            previous + 1,
        );

        setDetailsOpen(true);
        setShowPauseForm(false);
        setStateMessage(
          copy.prepareSuccess,
        );
      } catch (cause) {
        setStateError(
          cause instanceof Error
            ? cause.message
            : experience === "review"
              ? "准备整改建议失败"
              : "准备修改方案失败",
        );
      } finally {
        setGenerating(false);
      }
    };

  const handlePause =
    async () => {
      const reason =
        pauseReason.trim();

      const reasonLength =
        Array.from(
          reason,
        ).length;

      if (
        !reason ||
        reasonLength > 500 ||
        stateBusy
      ) {
        setStateError(
          reasonLength > 500
            ? "说明不能超过500字"
            : "请补充说明，方便以后回看",
        );
        return;
      }

      setStateAction("dismiss");
      setStateError("");
      setStateMessage("");

      try {
        const result =
          await dismissCWAIReviewItem(
            item.id,
            reason,
          );

        onChanged(
          result.item,
        );

        setPauseReason("");
        setShowPauseForm(false);
        setDetailsOpen(false);
        setStateMessage(
          copy.pauseSuccess,
        );
      } catch (cause) {
        setStateError(
          cause instanceof Error
            ? cause.message
            : experience === "review"
              ? "保存本次不退回的决定失败"
              : "保存暂不调整的决定失败",
        );
      } finally {
        setStateAction(null);
      }
    };

  const handleResume =
    async () => {
      if (stateBusy) {
        return;
      }

      setStateAction("restore");
      setStateError("");
      setStateMessage("");

      try {
        const result =
          await restoreCWAIReviewItem(
            item.id,
          );

        onChanged(
          result.item,
        );

        setDiscussionVersion(
          (previous) =>
            previous + 1,
        );

        setDetailsOpen(false);
        setStateMessage(
          result.item.status ===
            "confirmed"
            ? copy.resumeConfirmedSuccess
            : copy.resumePendingSuccess,
        );
      } catch (cause) {
        setStateError(
          cause instanceof Error
            ? cause.message
            : "恢复处理失败",
        );
      } finally {
        setStateAction(null);
      }
    };

  const {
    handleRecheckItem,
    handleResolveSelfItem,
  } =
    useCWAIReviewItemResolutionActions({
      item,
      experience,
      stateBusy,
      onChanged,
      setDiscussionVersion,
      setDetailsOpen,
      setStateAction,
      setStateError,
      setStateMessage,
    });

  const primaryAction:
    PrimaryAction =
    (() => {
      if (
        experience === "remediation"
      ) {
        if (
          item.status ===
            "confirmed" &&
          canOpenPageModification
        ) {
          return {
            label: copy.pageAction,
            tone: "success",
            opensDetails: false,
            onClick: () =>
              onInjectToRefine?.(
                item,
              ),
          };
        }

        if (
          item.status ===
            "applying" &&
          onInjectToRefine
        ) {
          return {
            label: copy.applyingAction,
            tone: "warning",
            opensDetails: false,
            onClick: () =>
              onInjectToRefine(
                item,
              ),
          };
        }

        if (
          item.status ===
          "applied"
        ) {
          return {
            label: copy.appliedAction,
            tone: "primary",
            opensDetails: true,
            onClick: () =>
              setDetailsOpen(true),
          };
        }

        if (
          item.status ===
          "stale"
        ) {
          return {
            label:
              stateAction ===
              "recheck"
                ? "正在检查…"
                : copy.staleAction,
            tone: "warning",
            opensDetails: false,
            onClick: () =>
              void handleRecheckItem(),
          };
        }

        if (
          item.status ===
          "orphaned"
        ) {
          return {
            label: copy.orphanedAction,
            tone: "neutral",
            opensDetails: true,
            onClick: () =>
              setDetailsOpen(true),
          };
        }

        if (
          item.status ===
          "resolved"
        ) {
          return {
            label: copy.resolvedAction,
            tone: "neutral",
            opensDetails: true,
            onClick: () =>
              setDetailsOpen(true),
          };
        }

        return {
          label: copy.confirmedAction,
          tone: "primary",
          opensDetails: true,
          onClick: () =>
            setDetailsOpen(true),
        };
      }

      if (canResume) {
        return {
          label:
            stateAction ===
            "restore"
              ? "正在恢复…"
              : copy.resumeAction,
          tone: "primary",
          opensDetails: false,
          onClick: () =>
            void handleResume(),
        };
      }

      if (
        item.status ===
        "detected"
      ) {
        return {
          label: generating
            ? "正在准备…"
            : copy.prepareAction,
          tone: "primary",
          opensDetails: false,
          onClick: () =>
            void handlePrepareModification(),
        };
      }

      if (
        item.status ===
        "discussing"
      ) {
        return {
          label: copy.continueAction,
          tone: "primary",
          opensDetails: true,
          onClick: () =>
            setDetailsOpen(true),
        };
      }

      if (
        experience === "self" &&
        item.status ===
          "confirmed" &&
        canOpenPageModification
      ) {
        return {
          label: copy.pageAction,
          tone: "success",
          opensDetails: false,
          onClick: () =>
            onInjectToRefine?.(
              item,
            ),
        };
      }

      if (
        item.status ===
        "confirmed"
      ) {
        return {
          label: copy.confirmedAction,
          tone: "success",
          opensDetails: true,
          onClick: () =>
            setDetailsOpen(true),
        };
      }

      if (
        item.status ===
        "applying"
      ) {
        return {
          label: copy.applyingAction,
          tone: "warning",
          opensDetails: true,
          onClick: () =>
            setDetailsOpen(true),
        };
      }

      if (
        item.status ===
        "applied"
      ) {
        if (
          experience ===
          "self"
        ) {
          return {
            label:
              stateAction ===
              "resolve"
                ? "正在确认…"
                : copy.appliedAction,
            tone: "success",
            opensDetails: false,
            onClick: () =>
              void handleResolveSelfItem(),
          };
        }

        return {
          label: copy.appliedAction,
          tone: "primary",
          opensDetails: true,
          onClick: () =>
            setDetailsOpen(true),
        };
      }

      if (
        item.status ===
        "resolved"
      ) {
        return {
          label: copy.resolvedAction,
          tone: "neutral",
          opensDetails: true,
          onClick: () =>
            setDetailsOpen(true),
        };
      }

      if (
        item.status ===
        "stale"
      ) {
        if (
          experience ===
          "self"
        ) {
          return {
            label:
              stateAction ===
              "recheck"
                ? "正在检查…"
                : copy.staleAction,
            tone: "warning",
            opensDetails: false,
            onClick: () =>
              void handleRecheckItem(),
          };
        }

        return {
          label: copy.staleAction,
          tone: "warning",
          opensDetails: true,
          onClick: () =>
            setDetailsOpen(true),
        };
      }

      return {
        label: copy.orphanedAction,
        tone: "neutral",
        opensDetails: true,
        onClick: () =>
          setDetailsOpen(true),
      };
    })();

  const status =
    resolveCWAIReviewItemStatus(
      experience,
      item.status,
    );

  return (
    <article
      style={{
        marginTop: "8px",
        padding: "10px",
        borderRadius: "9px",
        border:
          `1px solid ${status.color}35`,
        background: C.card,
        opacity:
          item.status ===
          "dismissed"
            ? 0.88
            : 1,
      }}
    >
      <CWAIReviewItemSummary
        experience={experience}
        item={item}
        activeRelations={
          activeRelations
        }
        sourceLabel={sourceLabel}
        pageLabel={pageLabel}
        summaryTitle={summaryTitle}
        summaryDescription={
          summaryDescription
        }
        selectable={
          experience === "review" &&
          selectable
        }
        selected={selected}
        canSelectForReturn={
          canSelectForReturn
        }
        canOpenPageModification={
          canOpenPageModification
        }
        onSelectedChange={
          onSelectedChange
        }
      />

      <div
        style={{
          display: "flex",
          gap: "6px",
          flexWrap: "wrap",
          marginTop: "8px",
        }}
      >
        <button
          type="button"
          onClick={
            primaryAction.onClick
          }
          disabled={stateBusy}
          style={
            cwAIReviewPrimaryButtonStyle(
              primaryAction.tone,
              stateBusy,
            )
          }
        >
          {primaryAction.label}
        </button>

        {!forceDetailsOpen &&
          (
            !primaryAction.opensDetails ||
            detailsOpen
          ) && (
            <button
            type="button"
            onClick={() =>
              setDetailsOpen(
                (previous) =>
                  !previous,
              )
            }
            style={
              cwAIReviewSecondaryButtonStyle
            }
          >
            {detailsOpen
              ? "收起详情"
              : "更多信息"}
            </button>
          )}
      </div>

      {stateMessage && (
        <CWAIReviewItemFeedback
          type="success"
          content={stateMessage}
        />
      )}

      {stateError && (
        <CWAIReviewItemFeedback
          type="error"
          content={stateError}
        />
      )}

      {detailsVisible && (
        <div
          style={{
            marginTop: "10px",
            paddingTop: "10px",
            borderTop:
              `1px solid ${C.border}`,
          }}
        >
          <CWAIReviewItemDetails
            experience={experience}
            item={item}
            itemMap={itemMap}
            activeRelations={
              activeRelations
            }
            manuallyAdded={
              manuallyAdded
            }
            evidenceSummary={
              evidenceSummary
            }
            pageLabel={pageLabel}
            canPrepare={canPrepare}
            canPause={canPause}
            canResume={canResume}
            stateBusy={stateBusy}
            generating={generating}
            showPauseForm={
              showPauseForm
            }
            pauseReason={
              pauseReason
            }
            stateAction={
              stateAction
            }
            discussionVersion={
              discussionVersion
            }
            onSelectPage={
              onSelectPage
            }
            onChanged={onChanged}
            onPrepareModification={() =>
              void handlePrepareModification()
            }
            onTogglePauseForm={() => {
              setShowPauseForm(
                (previous) =>
                  !previous,
              );

              setStateError("");
              setStateMessage("");
            }}
            onPauseReasonChange={
              setPauseReason
            }
            onPause={() =>
              void handlePause()
            }
            onResume={() =>
              void handleResume()
            }
          />
        </div>
      )}
    </article>
  );
}
