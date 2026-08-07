/**
 * 教学智能体草稿解析与恢复。
 *
 * 职责：
 *   - 防御性解析sessionStorage中的未知JSON；
 *   - 从数据库安全插槽恢复编辑草稿；
 *   - 兼容没有teaching_mode的历史v1方案；
 *   - 把历史方案在浏览器编辑态升级为v2；
 *   - 记录当前互动最后一次生成或保存时采用的学习方式；
 *   - 把AI生成结果覆盖进当前草稿；
 *   - 序列化完整结构化草稿。
 */

import type {
  CoursewareAssistantContextConfig,
  CoursewareAssistantGuidancePlan,
  CoursewareAssistantMisconceptionBranch,
  CoursewareAssistantPlanResult,
  CoursewareAssistantQuestionStep,
  CoursewareAssistantSlotView,
  CoursewareAssistantTeachingMode,
} from "@/api/coursewares";

import {
  createCoursewareAssistantBranch,
  createCoursewareAssistantQuestionStep,
  createDefaultCoursewareAssistantContextConfig,
  createDefaultCoursewareAssistantGuidancePlan,
  createEmptyCoursewareAssistantDraft,
  isCoursewareAssistantTeachingMode,
  type CoursewareAssistantEditorDraft,
} from "./coursewareAssistantDraftSchema";

type UnknownRecord = Record<string, unknown>;

function isRecord(
  value: unknown,
): value is UnknownRecord {
  return (
    typeof value === "object" &&
    value !== null
  );
}

function stringValue(
  value: unknown,
  fallback = "",
): string {
  return typeof value === "string"
    ? value
    : fallback;
}

function booleanValue(
  value: unknown,
  fallback: boolean,
): boolean {
  return typeof value === "boolean"
    ? value
    : fallback;
}

function numberValue(
  value: unknown,
  fallback: number,
): number {
  return (
    typeof value === "number" &&
    Number.isFinite(value)
  )
    ? value
    : fallback;
}

function stringArray(
  value: unknown,
  fallback: string[] = [],
): string[] {
  if (!Array.isArray(value)) {
    return [...fallback];
  }

  return value.filter(
    (item): item is string =>
      typeof item === "string",
  );
}

function teachingModeValue(
  value: unknown,
  fallback: CoursewareAssistantTeachingMode,
): CoursewareAssistantTeachingMode {
  return isCoursewareAssistantTeachingMode(
    value,
  )
    ? value
    : fallback;
}

function nullableTeachingModeValue(
  value: unknown,
  fallback: CoursewareAssistantTeachingMode | null,
): CoursewareAssistantTeachingMode | null {
  if (value === null) {
    return null;
  }

  if (
    isCoursewareAssistantTeachingMode(
      value,
    )
  ) {
    return value;
  }

  return fallback;
}

function parseQuestionStep(
  value: unknown,
  index: number,
): CoursewareAssistantQuestionStep {
  const fallback =
    createCoursewareAssistantQuestionStep(
      `Q${index + 1}`,
    );

  if (!isRecord(value)) {
    return fallback;
  }

  return {
    id: stringValue(
      value.id,
      fallback.id,
    ),
    prompt: stringValue(
      value.prompt,
    ),
    teaching_intent:
      stringValue(
        value.teaching_intent,
      ),
    expected_signals:
      stringArray(
        value.expected_signals,
        fallback.expected_signals,
      ),
    hint_ladder:
      stringArray(
        value.hint_ladder,
        fallback.hint_ladder,
      ),
    misconception_branch_ids:
      stringArray(
        value.misconception_branch_ids,
      ),
    next_step_id:
      stringValue(
        value.next_step_id,
      ),
    completion_signal:
      stringValue(
        value.completion_signal,
      ),
  };
}

function parseBranch(
  value: unknown,
  index: number,
  firstStepID: string,
): CoursewareAssistantMisconceptionBranch {
  const fallback =
    createCoursewareAssistantBranch(
      `M${index + 1}`,
      firstStepID,
    );

  if (!isRecord(value)) {
    return fallback;
  }

  return {
    id: stringValue(
      value.id,
      fallback.id,
    ),
    match_signals:
      stringArray(
        value.match_signals,
        fallback.match_signals,
      ),
    response_strategy:
      stringValue(
        value.response_strategy,
      ),
    follow_up_question:
      stringValue(
        value.follow_up_question,
      ),
    return_to_step_id:
      stringValue(
        value.return_to_step_id,
        firstStepID,
      ),
  };
}

function parseGuidancePlan(
  value: unknown,
  fallback:
    CoursewareAssistantGuidancePlan,
): CoursewareAssistantGuidancePlan {
  if (!isRecord(value)) {
    return {
      ...fallback,
      version: "v2",
      teaching_mode:
        teachingModeValue(
          fallback.teaching_mode,
          "guided_reasoning",
        ),
    };
  }

  const rawSteps =
    Array.isArray(
      value.question_chain,
    )
      ? value.question_chain
      : fallback.question_chain;

  const steps =
    rawSteps.map(
      parseQuestionStep,
    );

  const safeSteps =
    steps.length > 0
      ? steps
      : [
          createCoursewareAssistantQuestionStep(),
        ];

  const firstStepID =
    safeSteps[0]?.id || "Q1";

  const rawBranches =
    Array.isArray(
      value.misconception_branches,
    )
      ? value.misconception_branches
      : fallback.misconception_branches;

  const policy =
    isRecord(
      value.answer_leak_policy,
    )
      ? value.answer_leak_policy
      : {};

  const fallbackMode =
    teachingModeValue(
      fallback.teaching_mode,
      "guided_reasoning",
    );

  return {
    version: "v2",
    teaching_mode:
      teachingModeValue(
        value.teaching_mode,
        fallbackMode,
      ),
    guiding_principles:
      stringArray(
        value.guiding_principles,
        fallback.guiding_principles,
      ),
    question_chain:
      safeSteps,
    misconception_branches:
      rawBranches.map(
        (branch, index) =>
          parseBranch(
            branch,
            index,
            firstStepID,
          ),
      ),
    forbidden_behaviors:
      stringArray(
        value.forbidden_behaviors,
        fallback.forbidden_behaviors,
      ),
    completion_criteria:
      stringArray(
        value.completion_criteria,
        fallback.completion_criteria,
      ),
    answer_leak_policy: {
      direct_answer_allowed:
        false,
      require_student_try:
        booleanValue(
          policy.require_student_try,
          fallback.answer_leak_policy
            .require_student_try,
        ),
      maximum_hint_level:
        numberValue(
          policy.maximum_hint_level,
          fallback.answer_leak_policy
            .maximum_hint_level,
        ),
      prohibited_behaviors:
        stringArray(
          policy.prohibited_behaviors,
          fallback.answer_leak_policy
            .prohibited_behaviors,
        ),
      safe_closure_guidance:
        stringValue(
          policy.safe_closure_guidance,
          fallback.answer_leak_policy
            .safe_closure_guidance ||
            "",
        ),
    },
  };
}

function parseContextConfig(
  value: unknown,
  fallback:
    CoursewareAssistantContextConfig,
): CoursewareAssistantContextConfig {
  if (!isRecord(value)) {
    return fallback;
  }

  return {
    version: "v1",
    include_visible_text:
      booleanValue(
        value.include_visible_text,
        fallback.include_visible_text,
      ),
    include_page_plan:
      booleanValue(
        value.include_page_plan,
        fallback.include_page_plan,
      ),
    include_interaction_evidence:
      booleanValue(
        value.include_interaction_evidence,
        fallback
          .include_interaction_evidence,
      ),
    include_lesson_plan_excerpt:
      booleanValue(
        value.include_lesson_plan_excerpt,
        fallback
          .include_lesson_plan_excerpt,
      ),
    include_previous_page_summary:
      booleanValue(
        value
          .include_previous_page_summary,
        fallback
          .include_previous_page_summary,
      ),
    include_next_page_summary:
      booleanValue(
        value
          .include_next_page_summary,
        fallback
          .include_next_page_summary,
      ),
    max_lesson_plan_excerpt_chars:
      numberValue(
        value
          .max_lesson_plan_excerpt_chars,
        fallback
          .max_lesson_plan_excerpt_chars,
      ),
  };
}

export function draftFromCoursewareAssistantSlot(
  slot: CoursewareAssistantSlotView,
): CoursewareAssistantEditorDraft {
  const guidancePlan =
    parseGuidancePlan(
      slot.guidance_plan,
      createDefaultCoursewareAssistantGuidancePlan(),
    );

  return {
    assistantId:
      slot.assistant_id,
    teacherInstruction: "",
    title: slot.title,
    welcomeMessage:
      slot.welcome_message,
    teachingRole:
      slot.teaching_role,
    learningObjective:
      slot.learning_objective,
    status: slot.status,
    generatedTeachingMode:
      guidancePlan.teaching_mode,
    guidancePlan,
    contextConfig:
      parseContextConfig(
        slot.context_config,
        createDefaultCoursewareAssistantContextConfig(),
      ),
  };
}

export function applyGeneratedCoursewareAssistantPlan(
  previous:
    CoursewareAssistantEditorDraft,
  result:
    CoursewareAssistantPlanResult,
): CoursewareAssistantEditorDraft {
  const guidancePlan =
    parseGuidancePlan(
      result.guidance_plan,
      previous.guidancePlan,
    );

  return {
    ...previous,
    title: result.title,
    welcomeMessage:
      result.welcome_message,
    teachingRole:
      result.teaching_role,
    learningObjective:
      result.learning_objective,
    generatedTeachingMode:
      guidancePlan.teaching_mode,
    guidancePlan,
    contextConfig:
      parseContextConfig(
        result.context_config,
        previous.contextConfig,
      ),
  };
}

export function serializeCoursewareAssistantDraft(
  draft:
    CoursewareAssistantEditorDraft,
): string {
  return JSON.stringify(draft);
}

export function parseCoursewareAssistantDraft(
  raw: string,
  fallback:
    CoursewareAssistantEditorDraft =
      createEmptyCoursewareAssistantDraft(),
): CoursewareAssistantEditorDraft {
  if (!raw.trim()) {
    return fallback;
  }

  try {
    const parsed =
      JSON.parse(raw) as unknown;

    if (!isRecord(parsed)) {
      return fallback;
    }

    const status =
      parsed.status === "active" ||
      parsed.status === "disabled"
        ? parsed.status
        : fallback.status;

    const guidancePlan =
      parseGuidancePlan(
        parsed.guidancePlan,
        fallback.guidancePlan,
      );

    return {
      assistantId:
        typeof parsed.assistantId ===
          "string"
          ? parsed.assistantId
          : parsed.assistantId === null
            ? null
            : fallback.assistantId,
      teacherInstruction:
        stringValue(
          parsed.teacherInstruction,
          fallback.teacherInstruction,
        ),
      title:
        stringValue(
          parsed.title,
          fallback.title,
        ),
      welcomeMessage:
        stringValue(
          parsed.welcomeMessage,
          fallback.welcomeMessage,
        ),
      teachingRole:
        stringValue(
          parsed.teachingRole,
          fallback.teachingRole,
        ),
      learningObjective:
        stringValue(
          parsed.learningObjective,
          fallback.learningObjective,
        ),
      status,
      generatedTeachingMode:
        nullableTeachingModeValue(
          parsed.generatedTeachingMode,
          fallback.generatedTeachingMode,
        ),
      guidancePlan,
      contextConfig:
        parseContextConfig(
          parsed.contextConfig,
          fallback.contextConfig,
        ),
    };
  } catch {
    return fallback;
  }
}
