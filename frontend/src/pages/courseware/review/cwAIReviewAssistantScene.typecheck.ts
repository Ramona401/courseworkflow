/**
 * cwAIReviewAssistantScene.typecheck.ts
 *
 * 纯编译期契约检查。
 *
 * 当课件审核场景不再属于AssistantScene时，
 * TypeScript构建必须立即失败，防止普通string再次流入AssistantSelector。
 */

import type {
  AssistantScene,
} from "@/api/ai-assistants";

import {
  resolveCWAIReviewAssistantScene,
} from "./cwAIReviewAssistantScene";

const formalScene: AssistantScene =
  resolveCWAIReviewAssistantScene(false);

const selfReviewScene: AssistantScene =
  resolveCWAIReviewAssistantScene(true);

void formalScene;
void selfReviewScene;

export {};
