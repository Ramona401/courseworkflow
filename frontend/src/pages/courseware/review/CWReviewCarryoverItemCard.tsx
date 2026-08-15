/**
 * CWReviewCarryoverItemCard.tsx
 *
 * 跨轮复审单条问题适配器。
 *
 * 复审的“确认已解决”只更新当前工作台的resolved_review_item_ids草稿，
 * 真正resolved仍只在审核员最终提交正式审核决定时由后端事务写入。
 *
 * R-01.1复审动作：
 *   - “确认已解决”始终可见，但只有作者已完成修改时可以选择；
 *   - “继续要求修改”始终可见，只把当前复审草稿保持为未解决；
 *   - 两个动作都不会覆盖、删除或改写上一轮修改要求和历史记录；
 *   - 每个状态仍只有一个当前主要操作，另一个选择作为次要操作或不可用提示。
 *
 * “打开这一页”只是辅助动作，不改变问题状态。
 */

import type {
  CWReviewCarryoverItem,
} from "@/api/coursewares";

import {
  cwAIReviewPrimaryButtonStyle,
  cwAIReviewSecondaryButtonStyle,
  resolveCWAIReviewPageChangeTeacherCopy,
} from "./CWAIReviewItemPresentation.shared";
import TeacherImprovementCard from "./TeacherImprovementCard";

export interface CWReviewCarryoverPageReference {
  id: string;
  page_number: number;
}

export interface CWReviewCarryoverItemCardProps {
  item: CWReviewCarryoverItem;
  pages: CWReviewCarryoverPageReference[];
  selected: boolean;
  onResolvedChange: (
    itemID: string,
    resolved: boolean,
  ) => void;
  onSelectPage: (
    pageNumber: number,
  ) => void;
}

function resolveCurrentPage(
  item: CWReviewCarryoverItem,
  pages: CWReviewCarryoverPageReference[],
): CWReviewCarryoverPageReference | undefined {
  const pageID =
    item.page_id?.trim() || "";

  if (!pageID) {
    return undefined;
  }

  return pages.find(
    (page) => page.id === pageID,
  );
}

function resolveCarryoverNextStep(
  item: CWReviewCarryoverItem,
  selected: boolean,
): string {
  if (selected) {
    return "本轮已经暂选为“已解决”。只有提交本轮正式审核决定后，这个判断才会保存；仍可选择“继续要求修改”撤回本轮判断。";
  }

  switch (item.status) {
    case "applied":
      return "请打开当前页面，对照当前修改要求和检查项实际复查，再选择“确认已解决”或“继续要求修改”。";

    case "stale": {
      const copy =
        resolveCWAIReviewPageChangeTeacherCopy(
          item.status,
        );

      return `${copy?.label || "页面内容已变化"}，需要人工重新检查当前页面。当前不能确认已解决，请继续要求修改。`;
    }

    case "orphaned": {
      const copy =
        resolveCWAIReviewPageChangeTeacherCopy(
          item.status,
        );

      return `${copy?.label || "原页面已不存在"}，需要人工重新检查相关页面或整课内容。当前不能确认已解决，请继续要求修改。`;
    }

    case "applying":
      return "作者仍在修改，本轮暂时不能确认已解决，请继续要求修改。";

    case "confirmed":
      return "作者尚未登记完成修改，本轮不能确认已解决，请继续要求修改。";

    case "detected":
    case "discussing":
      return "这条历史修改要求尚未形成完整的可复查结果，本轮不能确认已解决，请继续要求修改。";

    default:
      return "请结合当前课件实际情况继续人工复查。";
  }
}

export default function CWReviewCarryoverItemCard({
  item,
  pages,
  selected,
  onResolvedChange,
  onSelectPage,
}: CWReviewCarryoverItemCardProps) {
  const canResolve =
    item.status === "applied";

  const currentPage =
    resolveCurrentPage(
      item,
      pages,
    );

  const isGlobal =
    item.page_number_snapshot <= 0;

  const pageLabel =
    isGlobal
      ? "整课"
      : currentPage
        ? `P${currentPage.page_number}`
        : `原P${item.page_number_snapshot}`;

  const sourceLabel =
    `原L${item.original_review_level}第${item.original_review_round}轮修改要求`;

  const confirmDisabled =
    !canResolve ||
    selected;

  const continueIsPrimary =
    selected ||
    !canResolve;

  const handleConfirmResolved = () => {
    if (!canResolve || selected) {
      return;
    }

    onResolvedChange(
      item.id,
      true,
    );
  };

  const handleContinueModification = () => {
    // 只把当前复审草稿保持为“未解决”。
    // 不修改历史要求、不创建覆盖版本，也不删除任何历史记录。
    onResolvedChange(
      item.id,
      false,
    );
  };

  const actions = (
    <>
      <button
        type="button"
        aria-pressed={selected}
        onClick={handleConfirmResolved}
        disabled={confirmDisabled}
        title={
          canResolve
            ? selected
              ? "本轮已经暂选为已解决"
              : "确认当前修改已经达到要求"
            : "只有作者完成修改并进入待复查状态后才能确认已解决"
        }
        style={cwAIReviewPrimaryButtonStyle(
          "success",
          confirmDisabled,
        )}
      >
        确认已解决
      </button>

      <button
        type="button"
        onClick={handleContinueModification}
        style={
          continueIsPrimary
            ? cwAIReviewPrimaryButtonStyle(
                "warning",
                false,
              )
            : {
                ...cwAIReviewSecondaryButtonStyle,
                border: "1px solid #D97706",
                color: "#D97706",
              }
        }
      >
        继续要求修改
      </button>

      {currentPage && (
        <button
          type="button"
          onClick={() =>
            onSelectPage(
              currentPage.page_number,
            )
          }
          style={cwAIReviewSecondaryButtonStyle}
        >
          打开这一页
        </button>
      )}
    </>
  );

  return (
    <TeacherImprovementCard
      experience="review"
      item={item}
      activeRelations={[]}
      sourceLabel={sourceLabel}
      pageLabel={pageLabel}
      nextStep={resolveCarryoverNextStep(
        item,
        selected,
      )}
      selectable={false}
      selected={selected}
      canSelectForReturn={false}
      actions={actions}
    />
  );
}
