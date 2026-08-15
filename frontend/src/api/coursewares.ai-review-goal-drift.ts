/**
 * coursewares.ai-review-goal-drift.ts
 *
 * R-01.1 修改要求目标漂移的专用浏览器API。
 *
 * 该入口只把教师明确选择的另一项问题创建为独立改进项。
 * 后端保证不会改写当前问题、确认要求、历史版本或页面内容。
 */

import apiClient from "./client";
import { extractData } from "./coursewares.types";

import type {
  CWAIReviewItem,
} from "./coursewares.ai-review.types";

export interface CWAIReviewRelatedImprovementResponse {
  item: CWAIReviewItem;
  message?: string;
}

export async function createCWAIReviewRelatedImprovement(
  itemId: string,
  content: string,
): Promise<CWAIReviewRelatedImprovementResponse> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/items/${itemId}/create-related-improvement`,
    {
      content,
    },
  );

  return extractData<CWAIReviewRelatedImprovementResponse>(
    response,
  );
}
