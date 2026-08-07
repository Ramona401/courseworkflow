/**
 * 教学智能体教学规则、答案保护和参考内容编辑器。
 *
 * mode="teaching"：
 *   - 教学规则与完成标准；
 *   - 提示层级和答案保护。
 *
 * mode="context"：
 *   - AI可以参考的当前页面、教案和相邻页面范围。
 *
 * mode="all"保留兼容能力，但教师主工作台使用明确子Tab分别展示，
 * 避免所有高级字段连续堆叠在同一长页面中。
 */

import type {
  CSSProperties,
  KeyboardEvent,
} from "react";

import {
  COURSEWARE_ASSISTANT_LIMITS,
  type CoursewareAssistantEditorDraft,
} from "./coursewareAssistantDraft";

import {
  CoursewareAssistantSection,
  CoursewareAssistantStringListEditor,
  coursewareAssistantInputStyle,
  coursewareAssistantLabelStyle,
} from "./CoursewareAssistantEditorShared";

import CoursewareAssistantContextEditor from "./CoursewareAssistantContextEditor";

interface Props {
  draft: CoursewareAssistantEditorDraft;
  onChange: (
    draft: CoursewareAssistantEditorDraft,
  ) => void;
  disabled?: boolean;
  mode?:
    | "all"
    | "teaching"
    | "context";
  onKeyDown?: (
    event: KeyboardEvent<HTMLElement>,
  ) => boolean;
}

export default function CoursewareAssistantPolicyContextEditor({
  draft,
  onChange,
  disabled = false,
  mode = "all",
  onKeyDown,
}: Props) {
  const plan =
    draft.guidancePlan;

  const policy =
    plan.answer_leak_policy;

  const showTeaching =
    mode === "all" ||
    mode === "teaching";

  const showContext =
    mode === "all" ||
    mode === "context";

  const setPlan = (
    nextPlan:
      CoursewareAssistantEditorDraft[
        "guidancePlan"
      ],
  ) => {
    onChange({
      ...draft,
      guidancePlan:
        nextPlan,
    });
  };

  const updatePolicy = (
    next:
      CoursewareAssistantEditorDraft[
        "guidancePlan"
      ]["answer_leak_policy"],
  ) => {
    setPlan({
      ...plan,
      answer_leak_policy:
        next,
    });
  };

  return (
    <>
      {showTeaching && (
        <>
          <CoursewareAssistantSection
            title="怎样引导和判断学生"
            description="这些内容由AI自动生成。老师只在需要改变教学重点或完成标准时修改。"
          >
            <CoursewareAssistantStringListEditor
              label="智能体应怎样引导学生 *"
              values={
                plan.guiding_principles
              }
              onChange={(values) =>
                setPlan({
                  ...plan,
                  guiding_principles:
                    values,
                })
              }
              disabled={disabled}
              minimumItems={1}
              maximumRunes={
                COURSEWARE_ASSISTANT_LIMITS
                  .principle
              }
              placeholder="例如：先让学生描述观察结果，再提供一个方向提示"
              onKeyDown={onKeyDown}
            />

            <CoursewareAssistantStringListEditor
              label="怎样算学生已经学会 *"
              values={
                plan.completion_criteria
              }
              onChange={(values) =>
                setPlan({
                  ...plan,
                  completion_criteria:
                    values,
                })
              }
              disabled={disabled}
              minimumItems={1}
              maximumRunes={
                COURSEWARE_ASSISTANT_LIMITS
                  .completionCriterion
              }
              placeholder="例如：学生能够独立说明选择这种方法的理由"
              onKeyDown={onKeyDown}
            />
          </CoursewareAssistantSection>

          <CoursewareAssistantSection
            title="提示与答案保护"
            description="系统始终禁止直接公布当前任务答案。老师可以调整提示层数和困难后的收束方式。"
          >
            <div
              style={{
                display: "grid",
                gridTemplateColumns:
                  "repeat(auto-fit, minmax(190px, 1fr))",
                gap: 10,
                marginBottom: 12,
              }}
            >
              <label
                style={
                  toggleCardStyle(true)
                }
              >
                <input
                  type="checkbox"
                  checked
                  disabled
                />

                <span>
                  <strong>
                    不直接公布答案
                  </strong>
                  <small>
                    安全规则固定开启
                  </small>
                </span>
              </label>

              <label
                style={
                  toggleCardStyle(
                    policy
                      .require_student_try,
                  )
                }
              >
                <input
                  type="checkbox"
                  checked={
                    policy
                      .require_student_try
                  }
                  disabled={disabled}
                  onChange={(event) =>
                    updatePolicy({
                      ...policy,
                      require_student_try:
                        event.target.checked,
                    })
                  }
                />

                <span>
                  <strong>
                    先让学生尝试
                  </strong>
                  <small>
                    AI生成时默认开启
                  </small>
                </span>
              </label>

              <label>
                <span
                  style={
                    coursewareAssistantLabelStyle
                  }
                >
                  最多提供几层提示
                </span>

                <input
                  type="number"
                  min={1}
                  max={
                    COURSEWARE_ASSISTANT_LIMITS
                      .maximumHints
                  }
                  value={
                    policy
                      .maximum_hint_level
                  }
                  disabled={disabled}
                  onChange={(event) =>
                    updatePolicy({
                      ...policy,
                      maximum_hint_level:
                        Number(
                          event.target.value,
                        ),
                    })
                  }
                  style={
                    coursewareAssistantInputStyle
                  }
                />
              </label>
            </div>

            <label>
              <span
                style={
                  coursewareAssistantLabelStyle
                }
              >
                多次提示后仍有困难时怎样结束
              </span>

              <textarea
                value={
                  policy
                    .safe_closure_guidance ||
                  ""
                }
                onChange={(event) =>
                  updatePolicy({
                    ...policy,
                    safe_closure_guidance:
                      event.target.value,
                  })
                }
                onKeyDown={(event) => {
                  onKeyDown?.(
                    event,
                  );
                }}
                disabled={disabled}
                rows={3}
                placeholder="例如：请学生回看页面示例，并建议其向教师说明具体卡住的步骤。"
                style={{
                  ...coursewareAssistantInputStyle,
                  resize: "vertical",
                }}
              />
            </label>

            <details
              style={{
                marginTop: 12,
                padding: "9px 10px",
                borderRadius: 8,
                border:
                  "1px solid #E2E8F0",
                background: "#F8FAFC",
              }}
            >
              <summary
                style={{
                  color: "#64748B",
                  fontSize: 10,
                  fontWeight: 700,
                  cursor:
                    disabled
                      ? "default"
                      : "pointer",
                }}
              >
                专业教学规则
              </summary>

              <div
                style={{
                  marginTop: 10,
                }}
              >
                <CoursewareAssistantStringListEditor
                  label="智能体不能做什么"
                  values={
                    plan.forbidden_behaviors
                  }
                  onChange={(values) =>
                    setPlan({
                      ...plan,
                      forbidden_behaviors:
                        values,
                    })
                  }
                  disabled={disabled}
                  maximumRunes={
                    COURSEWARE_ASSISTANT_LIMITS
                      .forbiddenBehavior
                  }
                  placeholder="例如：不得替学生完成页面中的操作任务"
                  onKeyDown={onKeyDown}
                />

                <CoursewareAssistantStringListEditor
                  label="提供提示时不能做什么"
                  values={
                    policy
                      .prohibited_behaviors
                  }
                  onChange={(values) =>
                    updatePolicy({
                      ...policy,
                      prohibited_behaviors:
                        values,
                    })
                  }
                  disabled={disabled}
                  maximumRunes={
                    COURSEWARE_ASSISTANT_LIMITS
                      .forbiddenBehavior
                  }
                  placeholder="例如：不得跳过学生尝试，直接给出完整解法"
                  onKeyDown={onKeyDown}
                />
              </div>
            </details>
          </CoursewareAssistantSection>
        </>
      )}

      {showContext && (
        <CoursewareAssistantContextEditor
          context={
            draft.contextConfig
          }
          onChange={(
            contextConfig,
          ) =>
            onChange({
              ...draft,
              contextConfig,
            })
          }
          disabled={disabled}
        />
      )}
    </>
  );
}

function toggleCardStyle(
  active: boolean,
): CSSProperties {
  return {
    display: "flex",
    alignItems:
      "flex-start",
    gap: 7,
    padding: 10,
    borderRadius: 9,
    border:
      `1px solid ${
        active
          ? "#4F7BE8"
          : "#E2E8F0"
      }`,
    background:
      active
        ? "rgba(79,123,232,0.08)"
        : "#FFFFFF",
    color: "#1F2937",
    fontSize: 11,
    cursor: "pointer",
  };
}
