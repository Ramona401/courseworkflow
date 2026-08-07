/**
 * cwReviewDecisionPayload.ts
 *
 * 正式课件审核请求的单一构造入口。
 *
 * 关键约束：
 *
 *   - 上轮问题复审判断无论通过还是退回都可以提交；
 *   - 首次审核没有历史整改问题时，复审问题列表可以省略；
 *   - 通过审核永不携带本轮新AI会话或新整改问题；
 *   - 退回修改时，可以携带已完成的本轮AI审核会话；
 *   - 新问题和已解决旧问题均去空、去重；
 *   - 没有会话ID时不会孤立提交本轮新问题ID；
 *   - 非法评分不进入请求体。
 */

import type {
  CWReviewDecisionRequest,
} from "@/api/coursewares";

import type {
  CWAIReviewContext,
} from "./useCWAIReviewController";

export interface CWReviewDecisionDraft {
  decision:
    | "approved"
    | "revision";

  comment: string;
  scoreText: string;

  aiReviewContext:
    CWAIReviewContext;

  /**
   * 本轮明确确认解决的历史问题。
   *
   * 首次审核或没有作出解决确认时可以省略。
   */
  resolvedReviewItemIds?:
    string[];
}

function normalizeReviewItemIDs(
  input: string[],
): string[] {
  const result:
    string[] = [];

  const seen =
    new Set<string>();

  for (const raw of input) {
    const value =
      raw.trim();

    if (
      !value ||
      seen.has(value)
    ) {
      continue;
    }

    seen.add(value);
    result.push(value);
  }

  return result;
}

/**
 * 构造正式审核请求。
 */
export function buildCWReviewDecisionRequest({
  decision,
  comment,
  scoreText,
  aiReviewContext,
  resolvedReviewItemIds = [],
}: CWReviewDecisionDraft): CWReviewDecisionRequest {
  const request:
    CWReviewDecisionRequest = {
    decision,
    comment:
      comment.trim(),
  };

  const normalizedScoreText =
    scoreText.trim();

  if (normalizedScoreText) {
    const score =
      Number.parseFloat(
        normalizedScoreText,
      );

    if (
      Number.isFinite(
        score,
      )
    ) {
      request.score =
        score;
    }
  }

  const resolvedIDs =
    normalizeReviewItemIDs(
      resolvedReviewItemIds,
    );

  if (
    resolvedIDs.length >
    0
  ) {
    request
      .resolved_review_item_ids =
      resolvedIDs;
  }

  if (
    decision !==
    "revision"
  ) {
    return request;
  }

  const sessionID =
    aiReviewContext
      .sessionId
      ?.trim() || "";

  if (!sessionID) {
    return request;
  }

  request.ai_review_session_id =
    sessionID;

  const itemIDs =
    normalizeReviewItemIDs(
      aiReviewContext
        .selectedItemIds,
    );

  if (
    itemIDs.length >
    0
  ) {
    request.review_item_ids =
      itemIDs;
  }

  return request;
}
