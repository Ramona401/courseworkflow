/**
 * 教学智能体编辑草稿协议骨架。
 *
 * 本文件只定义：
 *   - 前端镜像限制；
 *   - 八种教学方式的教师可读元数据；
 *   - 草稿结构；
 *   - 默认教学互动步骤、学习困难方案和参考内容配置；
 *   - Unicode字符计数和稳定内部编号生成。
 *
 * generatedTeachingMode只存在于浏览器编辑草稿中：
 *   - 记录当前互动方案是按哪一种学习方式生成或保存的；
 *   - 不进入后端插槽请求和发布快照；
 *   - 用于防止教师更换学习方式后误保存旧互动步骤。
 *
 * 本文件不解析存储、不校验业务、不构造HTTP请求。
 */

import type {
  CoursewareAssistantContextConfig,
  CoursewareAssistantGuidancePlan,
  CoursewareAssistantMisconceptionBranch,
  CoursewareAssistantQuestionStep,
  CoursewareAssistantSlotStatus,
  CoursewareAssistantTeachingMode,
} from "@/api/coursewares";

export const COURSEWARE_ASSISTANT_LIMITS = {
  title: 120,
  welcomeMessage: 4000,
  teachingRole: 4000,
  learningObjective: 8000,
  teacherInstruction: 8000,
  principle: 1000,
  questionPrompt: 4000,
  teachingIntent: 2000,
  signal: 500,
  hint: 2000,
  branchStrategy: 4000,
  followUpQuestion: 4000,
  completionCriterion: 1000,
  forbiddenBehavior: 1000,
  safeClosure: 2000,
  maximumQuestionSteps: 64,
  maximumBranches: 64,
  maximumHints: 8,
  maximumSignals: 32,
  maximumBranchReferences: 16,
  minimumLessonExcerpt: 500,
  maximumLessonExcerpt: 12000,
} as const;

export interface CoursewareAssistantTeachingModeOption {
  value: CoursewareAssistantTeachingMode;
  title: string;
  description: string;
  example: string;
}

export const COURSEWARE_ASSISTANT_TEACHING_MODE_OPTIONS:
  readonly CoursewareAssistantTeachingModeOption[] = [
    {
      value: "guided_reasoning",
      title: "一步步想明白",
      description: "通过连续小问题，让学生自己推导出结论。",
      example: "你从页面中先观察到了什么？为什么会这样？",
    },
    {
      value: "explain_back",
      title: "用自己的话讲清楚",
      description: "让学生先解释，再发现理解中的缺口。",
      example: "请不用课本原句，试着用自己的话讲一遍。",
    },
    {
      value: "predict_observe_explain",
      title: "先猜，再看，再解释",
      description: "先预测结果，再观察现象并解释原因。",
      example: "你预测接下来会发生什么？依据是什么？",
    },
    {
      value: "worked_example",
      title: "看一个例子，再自己做",
      description: "先理解示例，再逐步减少帮助并独立完成。",
      example: "这个示例的第一步为什么要这样做？",
    },
    {
      value: "coached_practice",
      title: "先自己做，错了再提示",
      description: "让学生先尝试，只在需要时给最小提示。",
      example: "你先试着完成，遇到困难时我再给提示。",
    },
    {
      value: "retrieval_check",
      title: "快速回忆，检查掌握",
      description: "用几个短问题发现当前知识薄弱点。",
      example: "先不看页面，你还记得这个概念的关键条件吗？",
    },
    {
      value: "compare_contrast",
      title: "比一比，找规律和区别",
      description: "比较两个对象，发现共同点、差异和规律。",
      example: "这两个例子最关键的相同点和不同点是什么？",
    },
    {
      value: "evidence_argument",
      title: "选择观点，用证据说明",
      description: "让学生作出判断，并用页面证据支持观点。",
      example: "你更支持哪一种观点？页面中哪条证据能支持它？",
    },
  ];

export interface CoursewareAssistantEditorDraft {
  assistantId: string | null;
  teacherInstruction: string;
  title: string;
  welcomeMessage: string;
  teachingRole: string;
  learningObjective: string;
  status: CoursewareAssistantSlotStatus;

  /**
   * 当前互动内容最后一次生成或正式保存时采用的学习方式。
   *
   * null表示尚未通过AI生成，也可能是教师从空白开始手工设计；
   * 非null且与guidancePlan.teaching_mode不一致时，必须重新生成后才能保存。
   */
  generatedTeachingMode: CoursewareAssistantTeachingMode | null;

  guidancePlan: CoursewareAssistantGuidancePlan;
  contextConfig: CoursewareAssistantContextConfig;
}

/**
 * 按Unicode码点统计字符数，与后端rune长度口径对应。
 */
export function coursewareAssistantRuneLength(
  value: string,
): number {
  return Array.from(value).length;
}

export function isCoursewareAssistantTeachingMode(
  value: unknown,
): value is CoursewareAssistantTeachingMode {
  return (
    value === "guided_reasoning" ||
    value === "explain_back" ||
    value === "predict_observe_explain" ||
    value === "worked_example" ||
    value === "coached_practice" ||
    value === "retrieval_check" ||
    value === "compare_contrast" ||
    value === "evidence_argument"
  );
}

export function createCoursewareAssistantQuestionStep(
  id = "Q1",
): CoursewareAssistantQuestionStep {
  return {
    id,
    prompt: "",
    teaching_intent: "",
    expected_signals: [""],
    hint_ladder: [""],
    misconception_branch_ids: [],
    completion_signal: "",
  };
}

export function createCoursewareAssistantBranch(
  id = "M1",
  returnToStepID = "Q1",
): CoursewareAssistantMisconceptionBranch {
  return {
    id,
    match_signals: [""],
    response_strategy: "",
    follow_up_question: "",
    return_to_step_id: returnToStepID,
  };
}

export function createDefaultCoursewareAssistantContextConfig():
  CoursewareAssistantContextConfig {
  return {
    version: "v1",
    include_visible_text: true,
    include_page_plan: true,
    include_interaction_evidence: true,
    include_lesson_plan_excerpt: true,
    include_previous_page_summary: true,
    include_next_page_summary: true,
    max_lesson_plan_excerpt_chars: 4000,
  };
}

export function createDefaultCoursewareAssistantGuidancePlan():
  CoursewareAssistantGuidancePlan {
  return {
    version: "v2",
    teaching_mode: "guided_reasoning",
    guiding_principles: [""],
    question_chain: [
      createCoursewareAssistantQuestionStep(),
    ],
    misconception_branches: [],
    forbidden_behaviors: [
      "不得直接公布当前学生任务的最终答案",
    ],
    completion_criteria: [""],
    answer_leak_policy: {
      direct_answer_allowed: false,
      require_student_try: true,
      maximum_hint_level: 3,
      prohibited_behaviors: [
        "不得跳过学生尝试直接给答案",
      ],
      safe_closure_guidance: "",
    },
  };
}

export function createEmptyCoursewareAssistantDraft():
  CoursewareAssistantEditorDraft {
  return {
    assistantId: null,
    teacherInstruction: "",
    title: "",
    welcomeMessage: "",
    teachingRole: "",
    learningObjective: "",
    status: "active",
    generatedTeachingMode: null,
    guidancePlan:
      createDefaultCoursewareAssistantGuidancePlan(),
    contextConfig:
      createDefaultCoursewareAssistantContextConfig(),
  };
}

/**
 * 为动态步骤或学习困难方案生成当前草稿中尚未使用的可读内部编号。
 */
export function nextAvailableCoursewareAssistantID(
  prefix: "Q" | "M",
  existingIDs: string[],
): string {
  const occupied = new Set(
    existingIDs.map((value) => value.trim()),
  );

  for (
    let index = 1;
    index <= 9999;
    index += 1
  ) {
    const candidate = `${prefix}${index}`;

    if (!occupied.has(candidate)) {
      return candidate;
    }
  }

  return `${prefix}${Date.now()}`;
}
