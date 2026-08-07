/**
 * 单个学生互动步骤的高级编辑卡片。
 *
 * 本组件只负责展示和编辑一个互动步骤，不负责步骤数组的增删排序，
 * 也不直接修改其它步骤或学习困难方案的引用。
 *
 * 教师界面使用课堂语言；内部仍保留步骤编号和分支编号，
 * 以支持步骤顺序、跳转关系和发布快照的确定性校验。
 *
 * 下拉选项和学习困难选项使用数组位置作为局部key，
 * 避免编辑内部编号时重建当前表单控件。
 */

import type { KeyboardEvent } from "react";

import type {
  CoursewareAssistantMisconceptionBranch,
  CoursewareAssistantQuestionStep,
} from "@/api/coursewares";

import { COURSEWARE_ASSISTANT_LIMITS } from "./coursewareAssistantDraft";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
  CoursewareAssistantStringListEditor,
  coursewareAssistantInputStyle,
  coursewareAssistantLabelStyle,
} from "./CoursewareAssistantEditorShared";

interface CoursewareAssistantQuestionStepCardProps {
  step: CoursewareAssistantQuestionStep;
  index: number;
  steps: CoursewareAssistantQuestionStep[];
  branches: CoursewareAssistantMisconceptionBranch[];
  maximumHintLevel: number;
  disabled: boolean;
  onChange: (step: CoursewareAssistantQuestionStep) => void;
  onRename: (nextID: string) => void;
  onMove: (offset: -1 | 1) => void;
  onRemove: () => void;
  onToggleBranchReference: (branchID: string) => void;
  onKeyDown?: (event: KeyboardEvent<HTMLElement>) => boolean;
}

export default function CoursewareAssistantQuestionStepCard({
  step,
  index,
  steps,
  branches,
  maximumHintLevel,
  disabled,
  onChange,
  onRename,
  onMove,
  onRemove,
  onToggleBranchReference,
  onKeyDown,
}: CoursewareAssistantQuestionStepCardProps) {
  const C = COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const hintLimit = Math.min(
    COURSEWARE_ASSISTANT_LIMITS.maximumHints,
    Math.max(1, Math.trunc(maximumHintLevel) || 1),
  );

  return (
    <article
      style={{
        marginBottom: 12,
        padding: 12,
        borderRadius: 10,
        border: `1px solid ${C.border}`,
        background: C.background,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: 8,
          marginBottom: 10,
        }}
      >
        <div style={{ color: C.text, fontSize: 12, fontWeight: 700 }}>
          互动第 {index + 1} 步
          {" · "}
          {step.id || "未填写编号"}
        </div>

        <div style={{ display: "flex", gap: 4 }}>
          <ActionButton
            label="↑"
            title="上移"
            disabled={disabled || index === 0}
            onClick={() => onMove(-1)}
          />

          <ActionButton
            label="↓"
            title="下移"
            disabled={disabled || index === steps.length - 1}
            onClick={() => onMove(1)}
          />

          <ActionButton
            label="删除"
            title="删除互动步骤"
            danger
            disabled={disabled || steps.length <= 1}
            onClick={onRemove}
          />
        </div>
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "160px minmax(180px, 1fr)",
          gap: 10,
          marginBottom: 10,
        }}
      >
        <label>
          <span style={coursewareAssistantLabelStyle}>内部编号 *</span>

          <input
            value={step.id}
            onChange={(event) => onRename(event.target.value)}
            onKeyDown={(event) => {
              onKeyDown?.(event);
            }}
            disabled={disabled}
            placeholder="例如 Q1"
            style={coursewareAssistantInputStyle}
          />
        </label>

        <label>
          <span style={coursewareAssistantLabelStyle}>完成后进入</span>

          <select
            value={step.next_step_id || ""}
            onChange={(event) =>
              onChange({
                ...step,
                next_step_id: event.target.value || undefined,
              })
            }
            disabled={disabled}
            style={{
              ...coursewareAssistantInputStyle,
              cursor: disabled ? "default" : "pointer",
            }}
          >
            <option value="">按当前顺序进入下一步</option>

            {steps.map((candidate, candidateIndex) => (
              <option
                key={candidateIndex}
                value={candidate.id}
                disabled={candidateIndex === index}
              >
                {candidate.id || "未命名步骤"}
              </option>
            ))}
          </select>
        </label>
      </div>

      <label>
        <span style={coursewareAssistantLabelStyle}>学生看到的问题或学习动作 *</span>

        <textarea
          value={step.prompt}
          onChange={(event) =>
            onChange({
              ...step,
              prompt: event.target.value,
            })
          }
          onKeyDown={(event) => {
            onKeyDown?.(event);
          }}
          disabled={disabled}
          rows={3}
          placeholder="例如：请比较页面中的两个例子，说出一个相同点和一个不同点。"
          style={{
            ...coursewareAssistantInputStyle,
            resize: "vertical",
          }}
        />
      </label>

      <label style={{ display: "block", marginTop: 10 }}>
        <span style={coursewareAssistantLabelStyle}>
          这一步想帮助学生学会什么 *
        </span>

        <textarea
          value={step.teaching_intent}
          onChange={(event) =>
            onChange({
              ...step,
              teaching_intent: event.target.value,
            })
          }
          onKeyDown={(event) => {
            onKeyDown?.(event);
          }}
          disabled={disabled}
          rows={2}
          placeholder="例如：帮助学生发现两个概念的关键区别，而不是只记住名称。"
          style={{
            ...coursewareAssistantInputStyle,
            resize: "vertical",
          }}
        />
      </label>

      <div style={{ marginTop: 12 }}>
        <CoursewareAssistantStringListEditor
          label="学生出现哪些表现说明正在理解"
          values={step.expected_signals}
          onChange={(values) =>
            onChange({
              ...step,
              expected_signals: values,
            })
          }
          disabled={disabled}
          maximumItems={COURSEWARE_ASSISTANT_LIMITS.maximumSignals}
          maximumRunes={COURSEWARE_ASSISTANT_LIMITS.signal}
          placeholder="例如：学生能够指出两种方法都需要满足的条件"
          onKeyDown={onKeyDown}
        />

        <CoursewareAssistantStringListEditor
          label={`分层提示（最多${hintLimit}层）`}
          values={step.hint_ladder}
          onChange={(values) =>
            onChange({
              ...step,
              hint_ladder: values,
            })
          }
          disabled={disabled}
          maximumItems={hintLimit}
          maximumRunes={COURSEWARE_ASSISTANT_LIMITS.hint}
          placeholder="先提醒观察方向，再提供更具体的线索，但不直接给出答案"
          onKeyDown={onKeyDown}
        />
      </div>

      <label>
        <span style={coursewareAssistantLabelStyle}>怎样算这一步已经完成</span>

        <input
          value={step.completion_signal || ""}
          onChange={(event) =>
            onChange({
              ...step,
              completion_signal: event.target.value,
            })
          }
          onKeyDown={(event) => {
            onKeyDown?.(event);
          }}
          disabled={disabled}
          placeholder="例如：学生能够独立说明判断理由"
          style={coursewareAssistantInputStyle}
        />
      </label>

      <div style={{ marginTop: 12 }}>
        <div style={coursewareAssistantLabelStyle}>
          学生出现困难时可采用的帮助方案
        </div>

        {branches.length === 0 ? (
          <div
            style={{
              padding: "8px 10px",
              borderRadius: 8,
              border: `1px dashed ${C.border}`,
              color: C.textMuted,
              fontSize: 10,
            }}
          >
            暂无预设帮助方案，可在“常见学习困难与应对”中添加。
          </div>
        ) : (
          <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
            {branches.map((branch, branchIndex) => {
              const checked = step.misconception_branch_ids.includes(branch.id);
              const limitReached =
                !checked &&
                step.misconception_branch_ids.length >=
                  COURSEWARE_ASSISTANT_LIMITS.maximumBranchReferences;

              return (
                <label
                  key={branchIndex}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 5,
                    padding: "5px 8px",
                    borderRadius: 7,
                    border: `1px solid ${checked ? C.primary : C.border}`,
                    background: checked ? C.primaryBackground : C.white,
                    color: C.textSecondary,
                    fontSize: 10,
                    cursor: disabled || limitReached ? "default" : "pointer",
                    opacity: limitReached ? 0.5 : 1,
                  }}
                >
                  <input
                    type="checkbox"
                    checked={checked}
                    disabled={disabled || limitReached}
                    onChange={() => onToggleBranchReference(branch.id)}
                  />

                  学习困难 {branch.id || "未命名"}
                </label>
              );
            })}
          </div>
        )}
      </div>
    </article>
  );
}

function ActionButton({
  label,
  title,
  disabled,
  danger = false,
  onClick,
}: {
  label: string;
  title: string;
  disabled: boolean;
  danger?: boolean;
  onClick: () => void;
}) {
  const C = COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <button
      type="button"
      title={title}
      onClick={onClick}
      disabled={disabled}
      style={{
        padding: "4px 8px",
        borderRadius: 6,
        border: `1px solid ${C.border}`,
        background: C.white,
        color: danger ? C.danger : C.textSecondary,
        fontSize: 10,
        cursor: disabled ? "default" : "pointer",
        opacity: disabled ? 0.4 : 1,
      }}
    >
      {label}
    </button>
  );
}
