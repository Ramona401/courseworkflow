/**
 * CWAIReviewItemPresentation.shared.tsx
 *
 * 单条课件问题在三种使用场景中的共用表达：
 *
 *   - review：审核员明确整改要求并决定是否退回；
 *   - self：作者提交前自审并修改自己的课件；
 *   - remediation：作者收到退回后按审核要求完成整改。
 *
 * 本文件只保留纯文案、能力判断与样式，不导出React组件。
 * 状态标签与页面变化教师语言由CWAIReviewItemStatus.shared.ts负责。
 */

import type {
  CSSProperties,
} from "react";

import type {
  CWAIReviewItem,
  CWAIReviewSeverity,
} from "@/api/coursewares";

import type {
  CWAIReviewItemExperience,
} from "./CWAIReviewItemStatus.shared";

export type {
  CWAIReviewItemExperience,
  CWAIReviewPageChangeTeacherCopy,
} from "./CWAIReviewItemStatus.shared";

export {
  resolveCWAIReviewItemStatus,
  resolveCWAIReviewPageChangeTeacherCopy,
} from "./CWAIReviewItemStatus.shared";

export interface CWAIReviewItemExperienceCopy {
  sourceAI: string;
  sourceManual: string;

  prepareAction: string;
  prepareAgainAction: string;
  continueAction: string;
  confirmedAction: string;
  pageAction: string;
  applyingAction: string;
  appliedAction: string;
  resolvedAction: string;
  staleAction: string;
  orphanedAction: string;

  pauseAction: string;
  resumeAction: string;
  pauseQuestion: string;
  pausePlaceholder: string;
  pauseConfirm: string;

  prepareSuccess: string;
  pauseSuccess: string;
  resumeConfirmedSuccess: string;
  resumePendingSuccess: string;

  descriptionTitle: string;
  suggestionAITitle: string;
  suggestionManualTitle: string;
  evidenceTitle: string;

  discussionLoading: string;
  discussionLoadError: string;
  discussionEmpty: string;
  discussionSummaryTitle: string;
  discussionPlaceholder: string;
  discussionSendAction: string;
  discussionSendingAction: string;
  discussionAssistantLabel: string;
  discussionUserLabel: string;
  discussionSystemLabel: string;

  finalTextTitle: string;
  finalTextPlaceholder: string;
  finalConfirmAction: string;
  finalConfirmingAction: string;
  finalConfirmHelp: string;

  readOnlyRequirementTitle: string;
  readOnlyRequirementHelp: string;
}

export const CW_AI_REVIEW_ITEM_COLORS = {
  primary: "#4F7BE8",
  success: "#059669",
  warning: "#D97706",
  danger: "#DC2626",
  info: "#0284C7",
  purple: "#7C3AED",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  card: "#FFFFFF",
};

const C =
  CW_AI_REVIEW_ITEM_COLORS;

const EXPERIENCE_COPY:
  Record<
    CWAIReviewItemExperience,
    CWAIReviewItemExperienceCopy
  > = {
    review: {
      sourceAI: "AI审核发现",
      sourceManual: "审核员补充",

      prepareAction: "准备整改建议",
      prepareAgainAction: "重新准备整改建议",
      continueAction: "继续完善整改要求",
      confirmedAction: "查看已确认的整改要求",
      pageAction: "查看问题页面",
      applyingAction: "查看作者处理进展",
      appliedAction: "检查作者修改结果",
      resolvedAction: "查看复审记录",
      staleAction: "重新检查当前页面",
      orphanedAction: "检查相关页面",

      pauseAction: "本次不退回",
      resumeAction: "恢复审核",
      pauseQuestion: "为什么本次不退回给作者？",
      pausePlaceholder:
        "例如：经审核确认，这项设计符合本节课目标，本次无需作者修改",
      pauseConfirm: "确认本次不退回",

      prepareSuccess:
        "已经准备了一份整改建议，请检查并明确作者应该怎样修改。",
      pauseSuccess:
        "本条问题不会放入本次退回给作者的内容。",
      resumeConfirmedSuccess:
        "已恢复审核，并重新放入本次退回给作者的内容。",
      resumePendingSuccess:
        "已恢复审核，可以继续完善整改要求。",

      descriptionTitle: "问题说明",
      suggestionAITitle: "AI最初给出的整改建议",
      suggestionManualTitle: "审核员最初补充的整改想法",
      evidenceTitle: "为什么会判断为问题",

      discussionLoading: "正在读取审核过程…",
      discussionLoadError: "读取审核过程失败",
      discussionEmpty:
        "还没有补充讨论。可以说明教学目标、希望保留的设计，或作者修改时必须注意的限制。",
      discussionSummaryTitle: "目前的整改思路",
      discussionPlaceholder:
        "补充教学目标、希望保留的设计，或作者修改时必须注意的内容",
      discussionSendAction: "请AI继续完善",
      discussionSendingAction: "AI正在整理…",
      discussionAssistantLabel: "AI审核助手",
      discussionUserLabel: "我的补充",
      discussionSystemLabel: "审核记录",

      finalTextTitle: "最终整改要求",
      finalTextPlaceholder:
        "检查并修改这段文字，确保作者收到后能够直接照着完成整改",
      finalConfirmAction: "确认作为本次整改要求",
      finalConfirmingAction: "正在保存…",
      finalConfirmHelp:
        "只有点击上方按钮，这段内容才会成为正式整改要求；审核员不会在这里直接修改作者的课件。",

      readOnlyRequirementTitle: "已确认的整改要求",
      readOnlyRequirementHelp:
        "这段内容将在本次退回时发给作者。",
    },

    self: {
      sourceAI: "AI自审发现",
      sourceManual: "我补充的问题",

      prepareAction: "准备修改方案",
      prepareAgainAction: "重新准备修改方案",
      continueAction: "继续完善修改方案",
      confirmedAction: "查看已准备好的修改方案",
      pageAction: "开始修改这一页",
      applyingAction: "继续修改页面",
      appliedAction: "确认问题已解决",
      resolvedAction: "查看处理记录",
      staleAction: "重新检查当前页面",
      orphanedAction: "检查相关页面",

      pauseAction: "这次暂不调整",
      resumeAction: "恢复调整",
      pauseQuestion: "为什么这次暂不调整？",
      pausePlaceholder:
        "例如：这个设计符合当前教学目标，暂时不需要调整",
      pauseConfirm: "确认这次暂不调整",

      prepareSuccess:
        "已经准备了一份修改方案，请检查是否符合你的教学设计。",
      pauseSuccess:
        "已标记为这次暂不调整，之后仍可以恢复。",
      resumeConfirmedSuccess:
        "已恢复调整，可以继续使用已准备好的修改方案。",
      resumePendingSuccess:
        "已恢复调整，可以继续完善修改方案。",

      descriptionTitle: "问题说明",
      suggestionAITitle: "AI最初给出的修改建议",
      suggestionManualTitle: "我最初补充的修改想法",
      evidenceTitle: "为什么建议调整",

      discussionLoading: "正在读取自审过程…",
      discussionLoadError: "读取自审过程失败",
      discussionEmpty:
        "还没有补充讨论。可以告诉AI你的教学目标、希望保留的设计，或哪些内容不能改动。",
      discussionSummaryTitle: "目前的修改思路",
      discussionPlaceholder:
        "补充你的教学目标、希望保留的设计或不能修改的内容",
      discussionSendAction: "请AI继续完善",
      discussionSendingAction: "AI正在整理…",
      discussionAssistantLabel: "AI自审助手",
      discussionUserLabel: "我的想法",
      discussionSystemLabel: "调整记录",

      finalTextTitle: "最终修改方案",
      finalTextPlaceholder:
        "检查并修改这段文字，确保自己能够直接照着完成页面调整",
      finalConfirmAction: "确认采用这个修改方案",
      finalConfirmingAction: "正在保存…",
      finalConfirmHelp:
        "保存后仍需要你手动修改页面；系统不会在未确认的情况下自动改动课件。",

      readOnlyRequirementTitle: "已准备好的修改方案",
      readOnlyRequirementHelp:
        "可以按这段方案继续修改并检查页面效果。",
    },

    remediation: {
      sourceAI: "审核中发现",
      sourceManual: "审核员补充",

      prepareAction: "查看审核要求",
      prepareAgainAction: "查看审核要求",
      continueAction: "查看审核要求",
      confirmedAction: "查看审核要求",
      pageAction: "开始修改这一页",
      applyingAction: "继续修改这一页",
      appliedAction: "检查是否达到要求",
      resolvedAction: "查看完成记录",
      staleAction: "重新检查当前页面",
      orphanedAction: "检查相关页面",

      pauseAction: "说明无需修改",
      resumeAction: "恢复整改",
      pauseQuestion: "为什么认为这次无需修改？",
      pausePlaceholder:
        "请说明当前页面为什么已经符合审核要求",
      pauseConfirm: "提交说明",

      prepareSuccess:
        "请按照审核员确认的要求完成修改。",
      pauseSuccess: "已保存说明。",
      resumeConfirmedSuccess: "已恢复整改。",
      resumePendingSuccess: "已恢复整改。",

      descriptionTitle: "审核员指出的问题",
      suggestionAITitle: "审核时最初提出的修改建议",
      suggestionManualTitle: "审核员补充的整改想法",
      evidenceTitle: "审核时看到的依据",

      discussionLoading: "正在读取审核要求…",
      discussionLoadError: "读取审核要求失败",
      discussionEmpty: "暂无补充说明。",
      discussionSummaryTitle: "审核时的处理思路",
      discussionPlaceholder: "",
      discussionSendAction: "",
      discussionSendingAction: "",
      discussionAssistantLabel: "AI审核说明",
      discussionUserLabel: "审核员补充",
      discussionSystemLabel: "整改记录",

      finalTextTitle: "审核员确认的整改要求",
      finalTextPlaceholder: "",
      finalConfirmAction: "",
      finalConfirmingAction: "",
      finalConfirmHelp: "",

      readOnlyRequirementTitle: "审核员确认的整改要求",
      readOnlyRequirementHelp:
        "请按这段要求完成修改，再检查页面是否真正达到要求。",
    },
  };

export const CW_AI_REVIEW_ITEM_SEVERITY:
  Record<
    CWAIReviewSeverity,
    {
      label: string;
      color: string;
      background: string;
    }
  > = {
    critical: {
      label: "必须处理",
      color: "#B91C1C",
      background: "#FEE2E2",
    },
    high: {
      label: "优先处理",
      color: "#C2410C",
      background: "#FFEDD5",
    },
    medium: {
      label: "建议处理",
      color: "#A16207",
      background: "#FEF9C3",
    },
    low: {
      label: "可以优化",
      color: "#0369A1",
      background: "#E0F2FE",
    },
    info: {
      label: "供参考",
      color: C.textSec,
      background: "#F1F5F9",
    },
  };

export type CWAIReviewItemStateAction =
  | "dismiss"
  | "restore"
  | "resolve"
  | "recheck"
  | null;

export type CWAIReviewPrimaryActionTone =
  | "primary"
  | "success"
  | "warning"
  | "neutral";

export function resolveCWAIReviewItemExperience(
  item: CWAIReviewItem,
  selectableForReturn: boolean,
  pageModificationAvailable: boolean,
): CWAIReviewItemExperience {
  if (selectableForReturn) {
    return "review";
  }

  if (
    item.courseware_review_id ||
    item.feedback_id
  ) {
    return "remediation";
  }

  if (item.source_type === "self") {
    return "self";
  }

  return pageModificationAvailable
    ? "self"
    : "review";
}

export function resolveCWAIReviewItemExperienceCopy(
  experience: CWAIReviewItemExperience,
): CWAIReviewItemExperienceCopy {
  return EXPERIENCE_COPY[experience];
}

export function resolveCWAIReviewItemSourceLabel(
  experience: CWAIReviewItemExperience,
  manuallyAdded: boolean,
): string {
  const copy =
    EXPERIENCE_COPY[experience];

  return manuallyAdded
    ? copy.sourceManual
    : copy.sourceAI;
}

export function canManageCWAIReviewItemBeforeReturn(
  item: CWAIReviewItem,
): boolean {
  return (
    !item.courseware_review_id &&
    !item.feedback_id
  );
}

export function canPrepareCWAIReviewItemModification(
  item: CWAIReviewItem,
): boolean {
  if (
    !canManageCWAIReviewItemBeforeReturn(item)
  ) {
    return false;
  }

  return (
    item.status === "detected" ||
    item.status === "discussing" ||
    item.status === "confirmed"
  );
}

export function canPauseCWAIReviewItem(
  item: CWAIReviewItem,
): boolean {
  return canPrepareCWAIReviewItemModification(
    item,
  );
}

export function canResumeCWAIReviewItem(
  item: CWAIReviewItem,
): boolean {
  return (
    canManageCWAIReviewItemBeforeReturn(
      item,
    ) &&
    item.status === "dismissed"
  );
}

export function resolveCWAIReviewItemNextStep(
  experience: CWAIReviewItemExperience,
  item: CWAIReviewItem,
  selectedForReturn: boolean,
  canOpenPageModification: boolean,
): string {
  if (experience === "review") {
    switch (item.status) {
      case "detected":
        return "建议下一步：明确作者需要怎样修改。";

      case "discussing":
        return "建议下一步：补充教学要求，再确认最终整改要求。";

      case "confirmed":
        return selectedForReturn
          ? "这条整改要求会在本次退回时发给作者。"
          : "整改要求已经明确，请决定本次是否发给作者。";

      case "applying":
        return "作者正在修改，完成后再检查实际效果。";

      case "applied":
        return "作者已经完成修改，等待复审确认。";

      case "resolved":
        return "这条问题已经确认解决。";

      case "dismissed":
        return "本次不退回给作者，后续仍可以恢复审核。";

      case "stale":
        return "页面内容已变化，需要人工重新检查当前页面，再判断这个问题是否仍然成立。";

      case "orphaned":
        return "原页面已不存在，需要人工重新检查相关页面或整课内容，再决定是否继续要求修改。";
    }
  }

  if (experience === "self") {
    switch (item.status) {
      case "detected":
        return "建议下一步：先形成一份适合自己课件的修改方案。";

      case "discussing":
        return "建议下一步：补充教学目标，再确认最终修改方案。";

      case "confirmed":
        return canOpenPageModification
          ? "修改方案已经准备好，可以开始修改这一页。"
          : "修改方案已经准备好，请查看后开始调整。";

      case "applying":
        return "页面正在修改，完成后请检查课堂呈现效果。";

      case "applied":
        return "页面已经修改，请实际检查教学内容和互动效果，再明确确认问题是否已经解决。";

      case "resolved":
        return "这条问题已经完成处理。";

      case "dismissed":
        return "这次暂不调整，后续仍可以恢复。";

      case "stale":
        return "页面内容已变化，需要人工重新检查当前页面；确认仍符合原方案后，再重新登记为修改完成。";

      case "orphaned":
        return "原页面已不存在，需要人工重新检查相关页面或整课内容，再判断这项自审修改是否仍然需要继续处理。";
    }
  }

  switch (item.status) {
    case "detected":
    case "discussing":
      return "请先阅读审核员指出的问题和具体整改要求。";

    case "confirmed":
      return canOpenPageModification
        ? "审核要求已经明确，可以开始修改这一页。"
        : "请按照审核员确认的要求完成修改。";

    case "applying":
      return "正在完成整改，修改后请对照审核要求检查。";

    case "applied":
      return "页面已经修改，请先自查并重新提交审核，最终由审核员复审确认。";

    case "resolved":
      return "这条整改已经完成。";

    case "dismissed":
      return "这条问题本次不需要继续修改。";

    case "stale":
      return "页面内容已变化，需要人工重新检查当前页面；确认符合当前修改要求后，再重新登记为修改完成并提交复审。";

    case "orphaned":
      return "原页面已不存在，需要人工重新检查相关页面或整课内容，并确认当前修改要求是否已经由其他页面覆盖。";
  }
}

export function cwAIReviewPrimaryButtonStyle(
  tone: CWAIReviewPrimaryActionTone,
  disabled: boolean,
): CSSProperties {
  const config = {
    primary: {
      color: "#FFFFFF",
      background: C.primary,
      border: C.primary,
    },
    success: {
      color: "#FFFFFF",
      background: C.success,
      border: C.success,
    },
    warning: {
      color: "#FFFFFF",
      background: C.warning,
      border: C.warning,
    },
    neutral: {
      color: C.textSec,
      background: "#FFFFFF",
      border: C.border,
    },
  }[tone];

  return {
    minHeight: "36px",
    padding: "8px 12px",
    borderRadius: "8px",
    border:
      `1px solid ${
        disabled
          ? "#CBD5E1"
          : config.border
      }`,
    background:
      disabled
        ? "#F1F5F9"
        : config.background,
    color:
      disabled
        ? C.textMuted
        : config.color,
    fontSize: "13px",
    fontWeight: 700,
    lineHeight: 1.4,
    cursor:
      disabled
        ? "not-allowed"
        : "pointer",
  };
}

export const cwAIReviewSecondaryButtonStyle:
  CSSProperties = {
    minHeight: "36px",
    padding: "8px 12px",
    borderRadius: "8px",
    border: `1px solid ${C.border}`,
    background: "#FFFFFF",
    color: C.textSec,
    fontSize: "13px",
    fontWeight: 600,
    lineHeight: 1.4,
    cursor: "pointer",
  };

export const cwAIReviewPauseTextareaStyle:
  CSSProperties = {
    width: "100%",
    boxSizing: "border-box",
    marginTop: "8px",
    padding: "10px 12px",
    borderRadius: "8px",
    border: "1px solid #FDBA74",
    resize: "vertical",
    fontFamily: "inherit",
    fontSize: "14px",
    lineHeight: 1.6,
    outline: "none",
  };
