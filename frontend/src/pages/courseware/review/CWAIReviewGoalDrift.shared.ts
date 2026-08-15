/**
 * CWAIReviewGoalDrift.shared.ts
 *
 * R-01.1 修改要求目标漂移的前端保守判定。
 *
 * 这里只决定“是否先提醒教师确认意图”，不会自动创建问题、
 * 不会修改当前要求，也不会替代后端权限与状态校验。
 *
 * 判定原则：
 *   1. 只有已经存在当前确认内容、且草稿发生实质变化时才检查；
 *   2. 明确提到另一页、带“另外/此外/还有”等新增问题信号时提高敏感度；
 *   3. 只有草稿与当前要求、问题事实都呈现很低文字重合时才兜底提醒；
 *   4. 宁可少提醒，也不把普通措辞润色误判成另一个问题。
 */

import type { CWAIReviewItem } from "@/api/coursewares";

export const CW_AI_REVIEW_GOAL_DRIFT_PROMPT =
  "这段内容看起来是在处理另一个问题。";

export const CW_AI_REVIEW_GOAL_DRIFT_CONTINUE =
  "继续处理当前问题";

export const CW_AI_REVIEW_GOAL_DRIFT_CREATE =
  "创建新改进项";

export const CW_AI_REVIEW_GOAL_DRIFT_CANCEL =
  "取消";

const DRIFT_CUES = [
  "另外",
  "另一个",
  "另一项",
  "此外",
  "还有",
  "同时还",
  "顺便",
  "新增",
  "再加",
  "补充一个",
  "另需",
  "另外需要",
] as const;

function normalizeGoalDriftText(value: string): string {
  return value
    .toLowerCase()
    .replace(/\s+/g, "")
    .replace(/[^\u3400-\u9fffa-z0-9]+/g, "");
}

function buildGoalDriftTokens(value: string): Set<string> {
  const normalized = normalizeGoalDriftText(value);
  const result = new Set<string>();

  const cjk = Array.from(
    normalized.matchAll(/[\u3400-\u9fff]/g),
    (match) => match[0],
  );

  for (let index = 0; index < cjk.length - 1; index += 1) {
    result.add(`${cjk[index]}${cjk[index + 1]}`);
  }

  for (const match of normalized.matchAll(/[a-z0-9]{2,}/g)) {
    result.add(match[0]);
  }

  return result;
}

function goalDriftOverlap(left: string, right: string): number {
  const leftTokens = buildGoalDriftTokens(left);
  const rightTokens = buildGoalDriftTokens(right);

  if (leftTokens.size === 0 || rightTokens.size === 0) {
    return 0;
  }

  let intersection = 0;

  for (const token of leftTokens) {
    if (rightTokens.has(token)) {
      intersection += 1;
    }
  }

  return intersection / Math.min(leftTokens.size, rightTokens.size);
}

function extractExplicitPageNumbers(value: string): Set<number> {
  const result = new Set<number>();

  for (const match of value.matchAll(/(?:第\s*)?(\d+)\s*页|p\s*(\d+)/gi)) {
    const raw = match[1] || match[2];
    const pageNumber = Number(raw);

    if (Number.isInteger(pageNumber) && pageNumber > 0) {
      result.add(pageNumber);
    }
  }

  return result;
}

function hasPageConflict(draft: string, baseline: string): boolean {
  const draftPages = extractExplicitPageNumbers(draft);
  const baselinePages = extractExplicitPageNumbers(baseline);

  if (draftPages.size === 0 || baselinePages.size === 0) {
    return false;
  }

  for (const pageNumber of draftPages) {
    if (baselinePages.has(pageNumber)) {
      return false;
    }
  }

  return true;
}

function maxProblemContextOverlap(
  draft: string,
  item: CWAIReviewItem,
): number {
  const references = [
    item.teacher_title,
    item.what_happened,
    item.teaching_impact,
    item.improvement_goal,
    item.confirmed_instruction,
  ];

  let maximum = 0;

  for (const reference of references) {
    const normalized = reference.trim();

    if (!normalized) {
      continue;
    }

    maximum = Math.max(
      maximum,
      goalDriftOverlap(draft, normalized),
    );
  }

  return maximum;
}

export function shouldWarnCWAIReviewGoalDrift({
  draft,
  baseline,
  item,
}: {
  draft: string;
  baseline: string;
  item: CWAIReviewItem;
}): boolean {
  const normalizedDraft = draft.trim();
  const normalizedBaseline = baseline.trim();

  if (
    !normalizedDraft ||
    !normalizedBaseline ||
    normalizedDraft === normalizedBaseline
  ) {
    return false;
  }

  const draftLength = Array.from(normalizedDraft).length;
  const baselineLength = Array.from(normalizedBaseline).length;

  if (draftLength < 12 || baselineLength < 8) {
    return false;
  }

  if (
    normalizedDraft.includes(normalizedBaseline) ||
    normalizedBaseline.includes(normalizedDraft)
  ) {
    return false;
  }

  const baselineOverlap = goalDriftOverlap(
    normalizedDraft,
    normalizedBaseline,
  );

  const contextOverlap = maxProblemContextOverlap(
    normalizedDraft,
    item,
  );

  const hasDriftCue = DRIFT_CUES.some((cue) =>
    normalizedDraft.includes(cue),
  );

  if (
    hasPageConflict(normalizedDraft, normalizedBaseline) &&
    baselineOverlap < 0.45
  ) {
    return true;
  }

  if (
    hasDriftCue &&
    baselineOverlap < 0.42 &&
    contextOverlap < 0.45
  ) {
    return true;
  }

  return (
    draftLength >= 24 &&
    baselineLength >= 20 &&
    baselineOverlap < 0.1 &&
    contextOverlap < 0.12
  );
}
