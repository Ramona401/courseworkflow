/**
 * 教学智能体课堂发布策略编辑器。
 *
 * 本组件只编排三个教师问题和专业设置。
 * 预设、默认值、部署恢复及请求转换位于
 * coursewareAssistantDeploymentPolicyModel。
 */

import type {
  Dispatch,
  SetStateAction,
} from "react";

import {
  CoursewareAssistantDeploymentDateTimeField,
  CoursewareAssistantDeploymentNumberField,
  CoursewareAssistantDeploymentPolicySummary,
  CoursewareAssistantDeploymentPresetSection,
  CoursewareAssistantDeploymentProfessionalSettings,
} from "./CoursewareAssistantDeploymentPolicyControls";

import {
  COURSEWARE_ASSISTANT_STUDENT_PRESETS,
  COURSEWARE_ASSISTANT_TURN_PRESETS,
  COURSEWARE_ASSISTANT_VALIDITY_PRESETS,
  selectCoursewareAssistantStudentPreset,
  selectCoursewareAssistantTurnPreset,
  selectCoursewareAssistantValidityPreset,
  withAutomaticCoursewareAssistantDailyLimit,
  type CoursewareAssistantDeploymentPolicyDraft,
} from "./coursewareAssistantDeploymentPolicyModel";

export type {
  CoursewareAssistantDeploymentPolicyDraft,
} from "./coursewareAssistantDeploymentPolicyModel";

export {
  buildCoursewareAssistantDeploymentRequest,
  coursewareAssistantDeploymentPolicyFromLive,
  createDefaultCoursewareAssistantDeploymentPolicy,
} from "./coursewareAssistantDeploymentPolicyModel";

export function CoursewareAssistantDeploymentPolicyEditor({
  policy,
  setPolicy,
  internalOrigin,
  disabled,
}: {
  policy:
    CoursewareAssistantDeploymentPolicyDraft;
  setPolicy:
    Dispatch<
      SetStateAction<
        CoursewareAssistantDeploymentPolicyDraft
      >
    >;
  internalOrigin: string;
  disabled: boolean;
}) {
  return (
    <div
      style={{
        marginTop: 12,
        padding: 13,
        borderRadius: 10,
        border:
          "1px solid #E2E8F0",
        background: "#FFFFFF",
      }}
    >
      <CoursewareAssistantDeploymentPresetSection
        number="1"
        title="每位学生大约互动几轮？"
        description="一次学生回答加一次智能体回应，计作一轮。"
        options={
          COURSEWARE_ASSISTANT_TURN_PRESETS
        }
        selected={
          policy.turnPreset
        }
        disabled={disabled}
        onSelect={(preset) =>
          setPolicy(
            (previous) =>
              selectCoursewareAssistantTurnPreset(
                previous,
                preset,
              ),
          )
        }
      >
        {policy.turnPreset ===
          "custom" && (
          <CoursewareAssistantDeploymentNumberField
            label="每位学生最多轮数"
            value={
              policy
                .perSessionTurnLimit
            }
            minimum={1}
            maximum={100}
            disabled={disabled}
            onChange={(value) =>
              setPolicy(
                (previous) =>
                  withAutomaticCoursewareAssistantDailyLimit({
                    ...previous,
                    perSessionTurnLimit:
                      value,
                  }),
              )
            }
          />
        )}
      </CoursewareAssistantDeploymentPresetSection>

      <CoursewareAssistantDeploymentPresetSection
        number="2"
        title="大约多少学生会使用？"
        description="系统据此自动预留当天调用额度，并增加少量重试空间。"
        options={
          COURSEWARE_ASSISTANT_STUDENT_PRESETS
        }
        selected={
          policy.studentPreset
        }
        disabled={disabled}
        onSelect={(preset) =>
          setPolicy(
            (previous) =>
              selectCoursewareAssistantStudentPreset(
                previous,
                preset,
              ),
          )
        }
      >
        {policy.studentPreset ===
          "custom" && (
          <CoursewareAssistantDeploymentNumberField
            label="预计学生人数"
            value={
              policy.expectedStudents
            }
            minimum={1}
            maximum={5000}
            disabled={disabled}
            onChange={(value) =>
              setPolicy(
                (previous) =>
                  withAutomaticCoursewareAssistantDailyLimit({
                    ...previous,
                    expectedStudents:
                      value,
                  }),
              )
            }
          />
        )}
      </CoursewareAssistantDeploymentPresetSection>

      <CoursewareAssistantDeploymentPresetSection
        number="3"
        title="使用到什么时候？"
        description="到期后不能开始新会话，已有记录和历史版本仍会保留。"
        options={
          COURSEWARE_ASSISTANT_VALIDITY_PRESETS
        }
        selected={
          policy.validityPreset
        }
        disabled={disabled}
        onSelect={(preset) =>
          setPolicy(
            (previous) =>
              selectCoursewareAssistantValidityPreset(
                previous,
                preset,
              ),
          )
        }
      >
        {policy.validityPreset ===
          "custom" && (
          <CoursewareAssistantDeploymentDateTimeField
            value={
              policy.validUntil
            }
            disabled={disabled}
            onChange={(value) =>
              setPolicy(
                (previous) => ({
                  ...previous,
                  validUntil:
                    value,
                }),
              )
            }
          />
        )}
      </CoursewareAssistantDeploymentPresetSection>

      <CoursewareAssistantDeploymentPolicySummary
        policy={policy}
      />

      <CoursewareAssistantDeploymentProfessionalSettings
        policy={policy}
        setPolicy={setPolicy}
        internalOrigin={
          internalOrigin
        }
        disabled={disabled}
      />
    </div>
  );
}
