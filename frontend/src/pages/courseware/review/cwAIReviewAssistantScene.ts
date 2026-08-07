/**
 * cwAIReviewAssistantScene.ts
 *
 * 课件AI审核面板使用的助手场景单一映射。
 *
 * 这里显式返回AssistantScene，避免条件表达式经过Hook返回类型推导后
 * 被扩大为普通string，导致AssistantSelector的严格场景类型失效。
 */

import type {
  AssistantScene,
} from "@/api/ai-assistants";

/**
 * 根据课件AI审核模式返回合法助手场景。
 */
export function resolveCWAIReviewAssistantScene(
  isSelfReview: boolean,
): AssistantScene {
  return isSelfReview
    ? "courseware_self_review"
    : "courseware_review";
}
