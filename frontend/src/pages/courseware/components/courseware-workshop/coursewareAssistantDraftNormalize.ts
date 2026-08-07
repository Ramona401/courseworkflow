/**
 * 教学智能体草稿规范化与请求构造。
 *
 * 保存前统一：
 *   - 去除首尾空白和数组空项；
 *   - 编辑态统一升级为v2教学方案协议；
 *   - 保留八种合法教学方式，非法值fail-closed回退到历史兼容方式；
 *   - 强制禁止直接答案；
 *   - 未启用教案片段时把长度置零；
 *   - 构造创建或更新请求。
 */

import type {
  CreateCoursewareAssistantSlotRequest,
  UpdateCoursewareAssistantSlotRequest,
} from "@/api/coursewares";

import {
  isCoursewareAssistantTeachingMode,
  type CoursewareAssistantEditorDraft,
} from "./coursewareAssistantDraftSchema";

function trimStringList(
  values: string[],
): string[] {
  return values
    .map((value) => value.trim())
    .filter(Boolean);
}

export function normalizeCoursewareAssistantDraft(
  draft: CoursewareAssistantEditorDraft,
): CoursewareAssistantEditorDraft {
  const questionChain =
    draft.guidancePlan.question_chain.map(
      (step) => ({
        ...step,
        id: step.id.trim(),
        prompt: step.prompt.trim(),
        teaching_intent:
          step.teaching_intent.trim(),
        expected_signals:
          trimStringList(
            step.expected_signals,
          ),
        hint_ladder:
          trimStringList(
            step.hint_ladder,
          ),
        misconception_branch_ids:
          trimStringList(
            step.misconception_branch_ids,
          ),
        next_step_id:
          step.next_step_id?.trim() ||
          undefined,
        completion_signal:
          step.completion_signal?.trim() ||
          undefined,
      }),
    );

  const branches =
    draft.guidancePlan
      .misconception_branches.map(
        (branch) => ({
          ...branch,
          id: branch.id.trim(),
          match_signals:
            trimStringList(
              branch.match_signals,
            ),
          response_strategy:
            branch.response_strategy.trim(),
          follow_up_question:
            branch.follow_up_question.trim(),
          return_to_step_id:
            branch.return_to_step_id.trim(),
        }),
      );

  const teachingMode =
    isCoursewareAssistantTeachingMode(
      draft.guidancePlan.teaching_mode,
    )
      ? draft.guidancePlan.teaching_mode
      : "guided_reasoning";

  return {
    ...draft,
    assistantId:
      draft.assistantId?.trim() || null,
    teacherInstruction:
      draft.teacherInstruction.trim(),
    title: draft.title.trim(),
    welcomeMessage:
      draft.welcomeMessage.trim(),
    teachingRole:
      draft.teachingRole.trim(),
    learningObjective:
      draft.learningObjective.trim(),
    guidancePlan: {
      ...draft.guidancePlan,
      version: "v2",
      teaching_mode: teachingMode,
      guiding_principles:
        trimStringList(
          draft.guidancePlan
            .guiding_principles,
        ),
      question_chain: questionChain,
      misconception_branches:
        branches,
      forbidden_behaviors:
        trimStringList(
          draft.guidancePlan
            .forbidden_behaviors,
        ),
      completion_criteria:
        trimStringList(
          draft.guidancePlan
            .completion_criteria,
        ),
      answer_leak_policy: {
        ...draft.guidancePlan
          .answer_leak_policy,
        direct_answer_allowed: false,
        prohibited_behaviors:
          trimStringList(
            draft.guidancePlan
              .answer_leak_policy
              .prohibited_behaviors,
          ),
        safe_closure_guidance:
          draft.guidancePlan
            .answer_leak_policy
            .safe_closure_guidance
            ?.trim() ||
          undefined,
      },
    },
    contextConfig: {
      ...draft.contextConfig,
      version: "v1",
      max_lesson_plan_excerpt_chars:
        draft.contextConfig
          .include_lesson_plan_excerpt
          ? draft.contextConfig
              .max_lesson_plan_excerpt_chars
          : 0,
    },
  };
}

export function toCreateCoursewareAssistantRequest(
  draft: CoursewareAssistantEditorDraft,
): CreateCoursewareAssistantSlotRequest {
  const normalized =
    normalizeCoursewareAssistantDraft(
      draft,
    );

  return {
    assistant_id:
      normalized.assistantId,
    title: normalized.title,
    welcome_message:
      normalized.welcomeMessage,
    teaching_role:
      normalized.teachingRole,
    learning_objective:
      normalized.learningObjective,
    guidance_plan:
      normalized.guidancePlan,
    context_config:
      normalized.contextConfig,
  };
}

export function toUpdateCoursewareAssistantRequest(
  draft: CoursewareAssistantEditorDraft,
): UpdateCoursewareAssistantSlotRequest {
  return {
    ...toCreateCoursewareAssistantRequest(
      draft,
    ),
    status: draft.status,
  };
}
