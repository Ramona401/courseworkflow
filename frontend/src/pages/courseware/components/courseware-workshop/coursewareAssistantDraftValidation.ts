/**
 * 教学智能体草稿前端镜像校验。
 *
 * 目的：
 *   - 保存前尽早向教师解释具体问题；
 *   - 镜像后端字符、数量、答案保护、教学方式和引用关系约束；
 *   - 防止更换学习方式后误保存旧互动步骤；
 *   - 不替代后端最终校验和权限检查。
 */

import {
  COURSEWARE_ASSISTANT_LIMITS,
  coursewareAssistantRuneLength,
  isCoursewareAssistantTeachingMode,
  type CoursewareAssistantEditorDraft,
} from "./coursewareAssistantDraftSchema";

import {
  normalizeCoursewareAssistantDraft,
} from "./coursewareAssistantDraftNormalize";

function validateRequiredText(
  errors: string[],
  label: string,
  value: string,
  maximum: number,
): void {
  if (!value.trim()) {
    errors.push(
      `${label}不能为空`,
    );
    return;
  }

  if (
    coursewareAssistantRuneLength(
      value,
    ) > maximum
  ) {
    errors.push(
      `${label}不能超过${maximum}个字符`,
    );
  }
}

function validateStringList(
  errors: string[],
  label: string,
  values: string[],
  maximum: number,
): void {
  values.forEach(
    (value, index) => {
      if (
        coursewareAssistantRuneLength(
          value,
        ) > maximum
      ) {
        errors.push(
          `${label}第${index + 1}项不能超过${maximum}个字符`,
        );
      }
    },
  );
}

export function validateCoursewareAssistantDraft(
  source:
    CoursewareAssistantEditorDraft,
): string[] {
  const draft =
    normalizeCoursewareAssistantDraft(
      source,
    );

  const errors: string[] = [];
  const limits =
    COURSEWARE_ASSISTANT_LIMITS;

  if (
    !isCoursewareAssistantTeachingMode(
      source.guidancePlan
        .teaching_mode,
    )
  ) {
    errors.push(
      "请选择一种有效的学生学习方式",
    );
  }

  /**
   * 非null表示当前草稿来自AI生成结果或数据库正式方案。
   * 此后若教师只更换teaching_mode而没有重新生成，
   * 旧互动步骤可能与新方式不一致，因此必须阻止保存。
   *
   * null表示教师可能从空白开始手工设计完整方案，
   * 不强制其调用AI生成。
   */
  if (
    source.generatedTeachingMode !==
      null &&
    source.generatedTeachingMode !==
      source.guidancePlan
        .teaching_mode
  ) {
    errors.push(
      "学习方式已经更换，请重新点击“生成学生互动”后再保存",
    );
  }

  validateRequiredText(
    errors,
    "智能体名称",
    draft.title,
    limits.title,
  );

  validateRequiredText(
    errors,
    "欢迎语",
    draft.welcomeMessage,
    limits.welcomeMessage,
  );

  validateRequiredText(
    errors,
    "教学角色",
    draft.teachingRole,
    limits.teachingRole,
  );

  validateRequiredText(
    errors,
    "教学目标",
    draft.learningObjective,
    limits.learningObjective,
  );

  if (
    coursewareAssistantRuneLength(
      draft.teacherInstruction,
    ) >
    limits.teacherInstruction
  ) {
    errors.push(
      `补充要求不能超过${limits.teacherInstruction}个字符`,
    );
  }

  const plan =
    draft.guidancePlan;

  if (
    plan.guiding_principles.length ===
    0
  ) {
    errors.push(
      "至少需要一条引导原则",
    );
  }

  if (
    plan.completion_criteria.length ===
    0
  ) {
    errors.push(
      "至少需要一条完成标准",
    );
  }

  validateStringList(
    errors,
    "引导原则",
    plan.guiding_principles,
    limits.principle,
  );

  validateStringList(
    errors,
    "禁止行为",
    plan.forbidden_behaviors,
    limits.forbiddenBehavior,
  );

  validateStringList(
    errors,
    "完成标准",
    plan.completion_criteria,
    limits.completionCriterion,
  );

  if (
    plan.question_chain.length === 0
  ) {
    errors.push(
      "至少需要一个教学互动步骤",
    );
  }

  if (
    plan.question_chain.length >
    limits.maximumQuestionSteps
  ) {
    errors.push(
      `教学互动步骤不能超过${limits.maximumQuestionSteps}个`,
    );
  }

  if (
    plan.misconception_branches
      .length >
    limits.maximumBranches
  ) {
    errors.push(
      `学习困难方案不能超过${limits.maximumBranches}个`,
    );
  }

  const rawMaximumHintLevel =
    plan.answer_leak_policy
      .maximum_hint_level;

  const maximumHintLevel =
    Number.isFinite(
      rawMaximumHintLevel,
    )
      ? Math.trunc(
          rawMaximumHintLevel,
        )
      : 0;

  if (
    maximumHintLevel < 1 ||
    maximumHintLevel >
      limits.maximumHints
  ) {
    errors.push(
      `最大提示层级必须在1至${limits.maximumHints}之间`,
    );
  }

  if (
    source.guidancePlan
      .answer_leak_policy
      .direct_answer_allowed
  ) {
    errors.push(
      "教学智能体禁止直接泄露当前学生任务的答案",
    );
  }

  validateStringList(
    errors,
    "答案保护禁止行为",
    plan.answer_leak_policy
      .prohibited_behaviors,
    limits.forbiddenBehavior,
  );

  if (
    coursewareAssistantRuneLength(
      plan.answer_leak_policy
        .safe_closure_guidance ||
        "",
    ) >
    limits.safeClosure
  ) {
    errors.push(
      `安全收束说明不能超过${limits.safeClosure}个字符`,
    );
  }

  const stepIDs =
    new Set<string>();

  plan.question_chain.forEach(
    (step, index) => {
      const label =
        `第${index + 1}个教学互动步骤`;

      if (!step.id) {
        errors.push(
          `${label}内部编号不能为空`,
        );
      } else if (
        stepIDs.has(step.id)
      ) {
        errors.push(
          `教学互动步骤编号“${step.id}”重复`,
        );
      } else {
        stepIDs.add(step.id);
      }

      validateRequiredText(
        errors,
        `${label}的学习动作`,
        step.prompt,
        limits.questionPrompt,
      );

      validateRequiredText(
        errors,
        `${label}的教学意图`,
        step.teaching_intent,
        limits.teachingIntent,
      );

      if (
        step.expected_signals.length >
        limits.maximumSignals
      ) {
        errors.push(
          `${label}的预期表现不能超过${limits.maximumSignals}项`,
        );
      }

      if (
        step.hint_ladder.length >
        maximumHintLevel
      ) {
        errors.push(
          `${label}的提示数量超过最大提示层级`,
        );
      }

      if (
        step.misconception_branch_ids
          .length >
        limits
          .maximumBranchReferences
      ) {
        errors.push(
          `${label}引用的学习困难方案不能超过${limits.maximumBranchReferences}个`,
        );
      }

      validateStringList(
        errors,
        `${label}的预期表现`,
        step.expected_signals,
        limits.signal,
      );

      validateStringList(
        errors,
        `${label}的提示`,
        step.hint_ladder,
        limits.hint,
      );
    },
  );

  const branchIDs =
    new Set<string>();

  plan.misconception_branches
    .forEach(
      (branch, index) => {
        const label =
          `第${index + 1}个学习困难方案`;

        if (!branch.id) {
          errors.push(
            `${label}内部编号不能为空`,
          );
        } else if (
          branchIDs.has(branch.id)
        ) {
          errors.push(
            `学习困难方案编号“${branch.id}”重复`,
          );
        } else {
          branchIDs.add(branch.id);
        }

        if (
          branch.match_signals.length ===
          0
        ) {
          errors.push(
            `${label}至少需要一个学生表现`,
          );
        }

        validateStringList(
          errors,
          `${label}的学生表现`,
          branch.match_signals,
          limits.signal,
        );

        validateRequiredText(
          errors,
          `${label}的帮助方式`,
          branch.response_strategy,
          limits.branchStrategy,
        );

        validateRequiredText(
          errors,
          `${label}的继续互动`,
          branch.follow_up_question,
          limits.followUpQuestion,
        );

        if (
          !branch.return_to_step_id
        ) {
          errors.push(
            `${label}必须指定帮助后返回的互动步骤`,
          );
        }
      },
    );

  plan.question_chain.forEach(
    (step) => {
      if (
        step.next_step_id &&
        !stepIDs.has(
          step.next_step_id,
        )
      ) {
        errors.push(
          `互动步骤“${step.id}”设置的下一步骤不存在`,
        );
      }

      step.misconception_branch_ids
        .forEach(
          (branchID) => {
            if (
              !branchIDs.has(
                branchID,
              )
            ) {
              errors.push(
                `互动步骤“${step.id}”引用的学习困难方案“${branchID}”不存在`,
              );
            }
          },
        );
    },
  );

  plan.misconception_branches
    .forEach(
      (branch) => {
        if (
          !stepIDs.has(
            branch.return_to_step_id,
          )
        ) {
          errors.push(
            `学习困难方案“${branch.id}”设置的返回步骤不存在`,
          );
        }
      },
    );

  const context =
    draft.contextConfig;

  if (
    !context.include_visible_text &&
    !context.include_page_plan
  ) {
    errors.push(
      "AI参考内容必须包含当前页面文字或当前页面教学说明",
    );
  }

  if (
    context
      .include_lesson_plan_excerpt &&
    (
      context
        .max_lesson_plan_excerpt_chars <
        limits.minimumLessonExcerpt ||
      context
        .max_lesson_plan_excerpt_chars >
        limits.maximumLessonExcerpt
    )
  ) {
    errors.push(
      `教案片段长度必须在${limits.minimumLessonExcerpt}至${limits.maximumLessonExcerpt}之间`,
    );
  }

  return Array.from(
    new Set(errors),
  );
}
