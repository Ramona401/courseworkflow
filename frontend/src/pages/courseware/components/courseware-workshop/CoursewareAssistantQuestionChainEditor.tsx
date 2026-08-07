/**
 * 教学智能体学生互动步骤编排器。
 *
 * 八种学习方式共用同一套确定性互动步骤结构。
 * 本组件负责步骤增删排序，以及步骤ID、下一步骤和学习困难引用的同步维护。
 * 单个步骤字段界面由CoursewareAssistantQuestionStepCard负责。
 *
 * React身份：
 *   - 步骤ID是教师可编辑业务字段，不能作为key；
 *   - 卡片自身不持有业务状态，使用数组位置作为受控列表身份；
 *   - 输入步骤ID不会重建卡片或丢失输入焦点；
 *   - 排序后所有字段立即由最新受控草稿重新渲染。
 */

import type {
  KeyboardEvent,
} from "react";

import type {
  CoursewareAssistantQuestionStep,
} from "@/api/coursewares";

import {
  COURSEWARE_ASSISTANT_LIMITS,
  createCoursewareAssistantQuestionStep,
  nextAvailableCoursewareAssistantID,
  type CoursewareAssistantEditorDraft,
} from "./coursewareAssistantDraft";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
  CoursewareAssistantSection,
} from "./CoursewareAssistantEditorShared";

import CoursewareAssistantQuestionStepCard from "./CoursewareAssistantQuestionStepCard";

interface CoursewareAssistantQuestionChainEditorProps {
  draft: CoursewareAssistantEditorDraft;
  onChange: (
    draft: CoursewareAssistantEditorDraft,
  ) => void;
  disabled?: boolean;
  onKeyDown?: (
    event: KeyboardEvent<HTMLElement>,
  ) => boolean;
}

export default function CoursewareAssistantQuestionChainEditor({
  draft,
  onChange,
  disabled = false,
  onKeyDown,
}: CoursewareAssistantQuestionChainEditorProps) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const steps =
    draft.guidancePlan.question_chain;

  const branches =
    draft.guidancePlan
      .misconception_branches;

  const setSteps = (
    nextSteps:
      CoursewareAssistantQuestionStep[],
  ) => {
    onChange({
      ...draft,
      guidancePlan: {
        ...draft.guidancePlan,
        question_chain:
          nextSteps,
      },
    });
  };

  const updateStep = (
    index: number,
    next:
      CoursewareAssistantQuestionStep,
  ) => {
    setSteps(
      steps.map(
        (step, stepIndex) =>
          stepIndex === index
            ? next
            : step,
      ),
    );
  };

  const renameStep = (
    index: number,
    nextID: string,
  ) => {
    const previousID =
      steps[index]?.id || "";

    const nextSteps =
      steps.map(
        (step, stepIndex) => ({
          ...step,
          id:
            stepIndex === index
              ? nextID
              : step.id,
          next_step_id:
            previousID &&
            step.next_step_id ===
              previousID
              ? nextID
              : step.next_step_id,
        }),
      );

    const nextBranches =
      branches.map(
        (branch) => ({
          ...branch,
          return_to_step_id:
            previousID &&
            branch.return_to_step_id ===
              previousID
              ? nextID
              : branch.return_to_step_id,
        }),
      );

    onChange({
      ...draft,
      guidancePlan: {
        ...draft.guidancePlan,
        question_chain:
          nextSteps,
        misconception_branches:
          nextBranches,
      },
    });
  };

  const addStep = () => {
    if (
      disabled ||
      steps.length >=
        COURSEWARE_ASSISTANT_LIMITS
          .maximumQuestionSteps
    ) {
      return;
    }

    const newID =
      nextAvailableCoursewareAssistantID(
        "Q",
        steps.map(
          (step) => step.id,
        ),
      );

    const nextSteps =
      steps.map(
        (step, index) =>
          index ===
            steps.length - 1 &&
          !step.next_step_id
            ? {
                ...step,
                next_step_id:
                  newID,
              }
            : step,
      );

    nextSteps.push(
      createCoursewareAssistantQuestionStep(
        newID,
      ),
    );

    setSteps(nextSteps);
  };

  const removeStep = (
    index: number,
  ) => {
    if (
      disabled ||
      steps.length <= 1
    ) {
      return;
    }

    const removedID =
      steps[index]?.id || "";

    const remaining =
      steps.filter(
        (_, stepIndex) =>
          stepIndex !== index,
      );

    const fallbackID =
      remaining[0]?.id || "";

    const nextSteps =
      remaining.map(
        (step) => ({
          ...step,
          next_step_id:
            removedID &&
            step.next_step_id ===
              removedID
              ? undefined
              : step.next_step_id,
        }),
      );

    const nextBranches =
      branches.map(
        (branch) => ({
          ...branch,
          return_to_step_id:
            removedID &&
            branch.return_to_step_id ===
              removedID
              ? fallbackID
              : branch.return_to_step_id,
        }),
      );

    onChange({
      ...draft,
      guidancePlan: {
        ...draft.guidancePlan,
        question_chain:
          nextSteps,
        misconception_branches:
          nextBranches,
      },
    });
  };

  const moveStep = (
    index: number,
    offset: -1 | 1,
  ) => {
    const target =
      index + offset;

    if (
      disabled ||
      target < 0 ||
      target >= steps.length
    ) {
      return;
    }

    const next =
      [...steps];

    const [moved] =
      next.splice(index, 1);

    next.splice(
      target,
      0,
      moved,
    );

    setSteps(next);
  };

  const toggleBranchReference = (
    index: number,
    branchID: string,
  ) => {
    const step =
      steps[index];

    if (!step) {
      return;
    }

    const references =
      step.misconception_branch_ids;

    updateStep(
      index,
      {
        ...step,
        misconception_branch_ids:
          references.includes(
            branchID,
          )
            ? references.filter(
                (value) =>
                  value !== branchID,
              )
            : [
                ...references,
                branchID,
              ],
      },
    );
  };

  return (
    <CoursewareAssistantSection
      title="学生互动步骤"
      description={`AI已经按所选学习方式生成互动过程。至少1个步骤，最多${COURSEWARE_ASSISTANT_LIMITS.maximumQuestionSteps}个；提示应由弱到强。`}
      actions={
        <button
          type="button"
          onClick={addStep}
          disabled={
            disabled ||
            steps.length >=
              COURSEWARE_ASSISTANT_LIMITS
                .maximumQuestionSteps
          }
          style={{
            padding: "6px 11px",
            borderRadius: 7,
            border:
              `1px dashed ${C.primary}`,
            background:
              C.primaryBackground,
            color: C.primary,
            fontSize: 11,
            fontWeight: 700,
            cursor:
              disabled
                ? "default"
                : "pointer",
          }}
        >
          ＋ 添加互动步骤
        </button>
      }
    >
      {steps.map(
        (step, index) => (
          <CoursewareAssistantQuestionStepCard
            key={index}
            step={step}
            index={index}
            steps={steps}
            branches={branches}
            maximumHintLevel={
              draft.guidancePlan
                .answer_leak_policy
                .maximum_hint_level
            }
            disabled={disabled}
            onChange={(next) =>
              updateStep(
                index,
                next,
              )
            }
            onRename={(nextID) =>
              renameStep(
                index,
                nextID,
              )
            }
            onMove={(offset) =>
              moveStep(
                index,
                offset,
              )
            }
            onRemove={() =>
              removeStep(index)
            }
            onToggleBranchReference={(
              branchID,
            ) =>
              toggleBranchReference(
                index,
                branchID,
              )
            }
            onKeyDown={onKeyDown}
          />
        ),
      )}
    </CoursewareAssistantSection>
  );
}
