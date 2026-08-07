/**
 * 教学智能体常见学习困难与应对编辑器。
 *
 * 功能：
 *   - 添加、删除、上移和下移学习困难；
 *   - 编辑内部编号、学生表现、帮助方式、继续互动和返回步骤；
 *   - 内部编号变化时同步全部互动步骤引用；
 *   - 删除学习困难时清理互动步骤中的悬空引用。
 *
 * 教师界面使用课堂语言；数据层仍保留misconception_branch协议字段，
 * 以兼容既有数据库JSON、发布快照和运行时逻辑。
 *
 * React身份：
 *   - 分支ID是教师可编辑业务字段，不能作为key；
 *   - 分支卡片没有独立业务状态，使用数组位置作为受控列表身份；
 *   - 输入分支ID不会重建卡片或丢失焦点；
 *   - 排序后字段立即由最新受控草稿重新渲染。
 */

import type {
  CSSProperties,
  KeyboardEvent,
} from "react";

import type {
  CoursewareAssistantMisconceptionBranch,
} from "@/api/coursewares";

import {
  COURSEWARE_ASSISTANT_LIMITS,
  createCoursewareAssistantBranch,
  nextAvailableCoursewareAssistantID,
  type CoursewareAssistantEditorDraft,
} from "./coursewareAssistantDraft";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
  CoursewareAssistantSection,
  CoursewareAssistantStringListEditor,
  coursewareAssistantInputStyle,
  coursewareAssistantLabelStyle,
} from "./CoursewareAssistantEditorShared";

interface CoursewareAssistantBranchEditorProps {
  draft: CoursewareAssistantEditorDraft;
  onChange: (
    draft: CoursewareAssistantEditorDraft,
  ) => void;
  disabled?: boolean;
  onKeyDown?: (
    event: KeyboardEvent<HTMLElement>,
  ) => boolean;
}

export default function CoursewareAssistantBranchEditor({
  draft,
  onChange,
  disabled = false,
  onKeyDown,
}: CoursewareAssistantBranchEditorProps) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const {
    question_chain: steps,
    misconception_branches: branches,
  } = draft.guidancePlan;

  const replacePlan = (
    nextBranches:
      CoursewareAssistantMisconceptionBranch[],
    nextSteps = steps,
  ) => {
    onChange({
      ...draft,
      guidancePlan: {
        ...draft.guidancePlan,
        question_chain: nextSteps,
        misconception_branches:
          nextBranches,
      },
    });
  };

  const updateBranch = (
    index: number,
    next:
      CoursewareAssistantMisconceptionBranch,
  ) => {
    replacePlan(
      branches.map(
        (branch, branchIndex) =>
          branchIndex === index
            ? next
            : branch,
      ),
    );
  };

  const renameBranch = (
    index: number,
    nextID: string,
  ) => {
    const previousID =
      branches[index]?.id || "";

    const nextBranches =
      branches.map(
        (branch, branchIndex) => ({
          ...branch,
          id:
            branchIndex === index
              ? nextID
              : branch.id,
        }),
      );

    const nextSteps =
      steps.map((step) => ({
        ...step,
        misconception_branch_ids:
          step.misconception_branch_ids.map(
            (reference) =>
              previousID &&
              reference === previousID
                ? nextID
                : reference,
          ),
      }));

    replacePlan(
      nextBranches,
      nextSteps,
    );
  };

  const addBranch = () => {
    if (
      disabled ||
      branches.length >=
        COURSEWARE_ASSISTANT_LIMITS
          .maximumBranches
    ) {
      return;
    }

    const branchID =
      nextAvailableCoursewareAssistantID(
        "M",
        branches.map(
          (branch) => branch.id,
        ),
      );

    replacePlan([
      ...branches,
      createCoursewareAssistantBranch(
        branchID,
        steps[0]?.id || "Q1",
      ),
    ]);
  };

  const removeBranch = (
    index: number,
  ) => {
    if (disabled) {
      return;
    }

    const removedID =
      branches[index]?.id || "";

    const nextBranches =
      branches.filter(
        (_, branchIndex) =>
          branchIndex !== index,
      );

    const nextSteps =
      steps.map((step) => ({
        ...step,
        misconception_branch_ids:
          step.misconception_branch_ids
            .filter(
              (reference) =>
                !removedID ||
                reference !== removedID,
            ),
      }));

    replacePlan(
      nextBranches,
      nextSteps,
    );
  };

  const moveBranch = (
    index: number,
    offset: -1 | 1,
  ) => {
    const target =
      index + offset;

    if (
      disabled ||
      target < 0 ||
      target >= branches.length
    ) {
      return;
    }

    const next =
      [...branches];

    const [moved] =
      next.splice(index, 1);

    next.splice(
      target,
      0,
      moved,
    );

    replacePlan(next);
  };

  const canAdd =
    !disabled &&
    branches.length <
      COURSEWARE_ASSISTANT_LIMITS
        .maximumBranches;

  return (
    <CoursewareAssistantSection
      title="常见学习困难与应对"
      description={`最多${COURSEWARE_ASSISTANT_LIMITS.maximumBranches}项。只记录教师能够观察到的学生表现，不记录AI隐藏推理。`}
      actions={
        <button
          type="button"
          onClick={addBranch}
          disabled={!canAdd}
          style={addButtonStyle(
            canAdd,
          )}
        >
          ＋ 添加学习困难
        </button>
      }
    >
      {branches.length === 0 && (
        <div
          style={{
            padding: "18px 12px",
            borderRadius: 9,
            border:
              `1px dashed ${C.border}`,
            background: C.background,
            color: C.textMuted,
            textAlign: "center",
            fontSize: 11,
            lineHeight: 1.7,
          }}
        >
          当前方案没有预设常见学习困难。没有明确典型问题时可以保持为空。
        </div>
      )}

      {branches.map(
        (branch, index) => (
          <article
            key={index}
            style={{
              marginBottom: 12,
              padding: 12,
              borderRadius: 10,
              border:
                `1px solid ${C.border}`,
              background: C.background,
            }}
          >
            <BranchHeader
              branchID={branch.id}
              index={index}
              total={branches.length}
              disabled={disabled}
              onMove={(offset) =>
                moveBranch(
                  index,
                  offset,
                )
              }
              onRemove={() =>
                removeBranch(index)
              }
            />

            <div
              style={{
                display: "grid",
                gridTemplateColumns:
                  "160px minmax(180px, 1fr)",
                gap: 10,
                marginBottom: 12,
              }}
            >
              <label>
                <span
                  style={
                    coursewareAssistantLabelStyle
                  }
                >
                  内部编号 *
                </span>

                <input
                  value={branch.id}
                  onChange={(event) =>
                    renameBranch(
                      index,
                      event.target.value,
                    )
                  }
                  onKeyDown={(event) => {
                    onKeyDown?.(
                      event,
                    );
                  }}
                  disabled={disabled}
                  placeholder="例如 M1"
                  style={
                    coursewareAssistantInputStyle
                  }
                />
              </label>

              <label>
                <span
                  style={
                    coursewareAssistantLabelStyle
                  }
                >
                  帮助后回到哪一步 *
                </span>

                <select
                  value={
                    branch.return_to_step_id
                  }
                  onChange={(event) =>
                    updateBranch(
                      index,
                      {
                        ...branch,
                        return_to_step_id:
                          event.target.value,
                      },
                    )
                  }
                  disabled={disabled}
                  style={{
                    ...coursewareAssistantInputStyle,
                    cursor:
                      disabled
                        ? "default"
                        : "pointer",
                  }}
                >
                  <option value="">
                    请选择互动步骤
                  </option>

                  {steps.map(
                    (
                      step,
                      stepIndex,
                    ) => (
                      <option
                        key={stepIndex}
                        value={step.id}
                      >
                        {step.id ||
                          "未命名步骤"}
                      </option>
                    ),
                  )}
                </select>
              </label>
            </div>

            <CoursewareAssistantStringListEditor
              label="学生可能出现的表现 *"
              values={
                branch.match_signals
              }
              onChange={(values) =>
                updateBranch(
                  index,
                  {
                    ...branch,
                    match_signals:
                      values,
                  },
                )
              }
              disabled={disabled}
              minimumItems={1}
              maximumRunes={
                COURSEWARE_ASSISTANT_LIMITS
                  .signal
              }
              placeholder="例如：把面积单位直接相加"
              onKeyDown={onKeyDown}
            />

            <BranchTextarea
              label="智能体应怎样帮助学生 *"
              value={
                branch.response_strategy
              }
              rows={3}
              placeholder="说明如何指出冲突、提供支架并让学生重新尝试，但不能直接给出答案。"
              disabled={disabled}
              marginBottom
              onKeyDown={onKeyDown}
              onChange={(value) =>
                updateBranch(
                  index,
                  {
                    ...branch,
                    response_strategy:
                      value,
                  },
                )
              }
            />

            <BranchTextarea
              label="帮助后接着怎样互动 *"
              value={
                branch.follow_up_question
              }
              rows={2}
              placeholder="用一个问题或学习动作，让学生验证并修正刚才的判断。"
              disabled={disabled}
              onKeyDown={onKeyDown}
              onChange={(value) =>
                updateBranch(
                  index,
                  {
                    ...branch,
                    follow_up_question:
                      value,
                  },
                )
              }
            />
          </article>
        ),
      )}
    </CoursewareAssistantSection>
  );
}

function BranchHeader({
  branchID,
  index,
  total,
  disabled,
  onMove,
  onRemove,
}: {
  branchID: string;
  index: number;
  total: number;
  disabled: boolean;
  onMove: (offset: -1 | 1) => void;
  onRemove: () => void;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent:
          "space-between",
        gap: 8,
        marginBottom: 10,
      }}
    >
      <div
        style={{
          color: C.text,
          fontSize: 12,
          fontWeight: 700,
        }}
      >
        学习困难 {index + 1}
        {" · "}
        {branchID ||
          "未填写编号"}
      </div>

      <div
        style={{
          display: "flex",
          gap: 4,
        }}
      >
        <MiniButton
          label="↑"
          disabled={
            disabled ||
            index === 0
          }
          onClick={() =>
            onMove(-1)
          }
        />

        <MiniButton
          label="↓"
          disabled={
            disabled ||
            index === total - 1
          }
          onClick={() =>
            onMove(1)
          }
        />

        <MiniButton
          label="删除"
          danger
          disabled={disabled}
          onClick={onRemove}
        />
      </div>
    </div>
  );
}

function BranchTextarea({
  label,
  value,
  rows,
  placeholder,
  disabled,
  marginBottom = false,
  onChange,
  onKeyDown,
}: {
  label: string;
  value: string;
  rows: number;
  placeholder: string;
  disabled: boolean;
  marginBottom?: boolean;
  onChange: (value: string) => void;
  onKeyDown?: (
    event: KeyboardEvent<HTMLElement>,
  ) => boolean;
}) {
  return (
    <label
      style={{
        display: "block",
        marginBottom:
          marginBottom
            ? 10
            : 0,
      }}
    >
      <span
        style={
          coursewareAssistantLabelStyle
        }
      >
        {label}
      </span>

      <textarea
        value={value}
        onChange={(event) =>
          onChange(
            event.target.value,
          )
        }
        onKeyDown={(event) => {
          onKeyDown?.(
            event,
          );
        }}
        disabled={disabled}
        rows={rows}
        placeholder={placeholder}
        style={{
          ...coursewareAssistantInputStyle,
          resize: "vertical",
        }}
      />
    </label>
  );
}

function MiniButton({
  label,
  disabled,
  danger = false,
  onClick,
}: {
  label: string;
  disabled: boolean;
  danger?: boolean;
  onClick: () => void;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      style={{
        padding: "4px 8px",
        borderRadius: 6,
        border:
          `1px solid ${C.border}`,
        background: C.white,
        color:
          danger
            ? C.danger
            : C.textSecondary,
        fontSize: 10,
        cursor:
          disabled
            ? "default"
            : "pointer",
        opacity:
          disabled
            ? 0.4
            : 1,
      }}
    >
      {label}
    </button>
  );
}

function addButtonStyle(
  enabled: boolean,
): CSSProperties {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return {
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
      enabled
        ? "pointer"
        : "default",
    opacity:
      enabled
        ? 1
        : 0.5,
  };
}
