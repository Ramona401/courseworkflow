/**
 * 教学智能体生成要求与基础字段编辑器。
 *
 * generation模式只展示：
 *   - 教师可选补充要求；
 *   - 操作台创建方案说明；
 *   - 页面方案本身就是教学智能体的流程说明。
 *
 * advanced模式才展示：
 *   - 名称、欢迎语、教学角色和教学目标；
 *   - 插槽启用状态。
 *
 * 创建动作统一放在固定浮动操作台中，
 * 不再在内容区重复放置容易被忽略的主按钮。
 */

import type {
  KeyboardEvent,
} from "react";

import {
  COURSEWARE_ASSISTANT_LIMITS,
  coursewareAssistantRuneLength,
  type CoursewareAssistantEditorDraft,
} from "./coursewareAssistantDraft";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
  CoursewareAssistantSection,
  coursewareAssistantInputStyle,
  coursewareAssistantLabelStyle,
} from "./CoursewareAssistantEditorShared";

interface CoursewareAssistantCoreEditorProps {
  draft: CoursewareAssistantEditorDraft;
  onChange: (
    draft: CoursewareAssistantEditorDraft,
  ) => void;

  /**
   * 以下课件元数据由现有父组件继续传入。
   * 当前普通教师流程不展示助手选择器，因此本组件暂不直接使用。
   */
  subject: string;
  grade: string;
  lessonPlanId?: string | null;

  disabled?: boolean;
  generating?: boolean;
  mode?:
    | "all"
    | "generation"
    | "advanced";
  onKeyDown?: (
    event: KeyboardEvent<HTMLElement>,
  ) => boolean;
}

interface TextFieldProps {
  label: string;
  value: string;
  maximumRunes: number;
  rows: number;
  placeholder: string;
  disabled: boolean;
  onChange: (
    value: string,
  ) => void;
  onKeyDown?: (
    event: KeyboardEvent<HTMLElement>,
  ) => boolean;
}

export default function CoursewareAssistantCoreEditor({
  draft,
  onChange,
  disabled = false,
  generating = false,
  mode = "all",
  onKeyDown,
}: CoursewareAssistantCoreEditorProps) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const showGeneration =
    mode === "all" ||
    mode === "generation";

  const showAdvanced =
    mode === "all" ||
    mode === "advanced";

  const setField = <
    Key extends keyof
      CoursewareAssistantEditorDraft,
  >(
    key: Key,
    value:
      CoursewareAssistantEditorDraft[Key],
  ) => {
    onChange({
      ...draft,
      [key]: value,
    });
  };

  const instructionLength =
    coursewareAssistantRuneLength(
      draft.teacherInstruction,
    );

  const instructionOverLimit =
    instructionLength >
    COURSEWARE_ASSISTANT_LIMITS
      .teacherInstruction;

  return (
    <>
      {showGeneration && (
        <CoursewareAssistantSection
          title="还有什么要强调？"
          description="这一步可以留空。系统会结合当前页面、学科、年级和你选择的学习方式自动生成互动方案。"
        >
          <textarea
            value={
              draft.teacherInstruction
            }
            onChange={(event) =>
              setField(
                "teacherInstruction",
                event.target.value,
              )
            }
            onKeyDown={(event) => {
              onKeyDown?.(event);
            }}
            disabled={
              disabled ||
              generating
            }
            rows={3}
            placeholder="例如：重点纠正学生把速度和加速度混淆的问题。留空也可以直接创建。"
            style={{
              ...coursewareAssistantInputStyle,
              resize: "vertical",
              borderColor:
                instructionOverLimit
                  ? C.danger
                  : C.border,
            }}
          />

          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent:
                "space-between",
              gap: 10,
              marginTop: 7,
              flexWrap: "wrap",
            }}
          >
            <span
              style={{
                color:
                  instructionOverLimit
                    ? C.danger
                    : C.textMuted,
                fontSize: 9,
              }}
            >
              {instructionLength}/
              {
                COURSEWARE_ASSISTANT_LIMITS
                  .teacherInstruction
              }
            </span>

            <span
              style={{
                color:
                  generating
                    ? C.primary
                    : instructionOverLimit
                      ? C.danger
                      : C.textSecondary,
                fontSize: 10,
                fontWeight: 700,
              }}
            >
              {generating
                ? "正在创建并检查学生活动方案…"
                : instructionOverLimit
                  ? "补充要求超过长度限制"
                  : "填写完成后，在右下角操作台点击“创建学生活动方案”"}
            </span>
          </div>

          <div
            style={{
              marginTop: 12,
              padding: "9px 11px",
              borderRadius: 8,
              border:
                "1px solid #BFDBFE",
              background:
                "#EFF6FF",
              color: "#1D4ED8",
              fontSize: 9,
              lineHeight: 1.65,
            }}
          >
            当前页面创建并保存的方案本身就是教学智能体，不需要再创建或选择其他AI助手。
            系统会自动检查互动结构和答案保护；首次检查不通过时会自动纠正一次。
          </div>
        </CoursewareAssistantSection>
      )}

      {showAdvanced && (
        <CoursewareAssistantSection
          title="基础教学设置"
          description="这些内容由AI自动生成。需要精细控制时再修改，普通使用不必逐项填写。"
        >
          <div
            style={{
              display: "grid",
              gridTemplateColumns:
                "minmax(0, 1fr) 160px",
              gap: 12,
              marginBottom: 12,
            }}
          >
            <TextField
              label="智能体名称 *"
              value={draft.title}
              maximumRunes={
                COURSEWARE_ASSISTANT_LIMITS
                  .title
              }
              rows={1}
              placeholder="例如：面积推理引导助手"
              disabled={disabled}
              onChange={(value) =>
                setField(
                  "title",
                  value,
                )
              }
              onKeyDown={onKeyDown}
            />

            <div>
              <label
                style={
                  coursewareAssistantLabelStyle
                }
              >
                使用状态
              </label>

              <select
                value={draft.status}
                onChange={(event) =>
                  setField(
                    "status",
                    event.target.value ===
                      "disabled"
                      ? "disabled"
                      : "active",
                  )
                }
                disabled={disabled}
                style={{
                  ...coursewareAssistantInputStyle,
                  cursor: disabled
                    ? "default"
                    : "pointer",
                }}
              >
                <option value="active">
                  已启用
                </option>
                <option value="disabled">
                  已停用
                </option>
              </select>

              <div
                style={{
                  marginTop: 5,
                  color:
                    C.textMuted,
                  fontSize: 9,
                  lineHeight: 1.5,
                }}
              >
                停用不会删除历史发布版本。
              </div>
            </div>
          </div>

          <TextField
            label="学生看到的开场语 *"
            value={
              draft.welcomeMessage
            }
            maximumRunes={
              COURSEWARE_ASSISTANT_LIMITS
                .welcomeMessage
            }
            rows={3}
            placeholder="学生打开助手时看到的第一句话。"
            disabled={disabled}
            onChange={(value) =>
              setField(
                "welcomeMessage",
                value,
              )
            }
            onKeyDown={onKeyDown}
          />

          <TextField
            label="智能体怎样帮助学生 *"
            value={
              draft.teachingRole
            }
            maximumRunes={
              COURSEWARE_ASSISTANT_LIMITS
                .teachingRole
            }
            rows={4}
            placeholder="说明助手在本页中如何互动、提示和纠偏。"
            disabled={disabled}
            onChange={(value) =>
              setField(
                "teachingRole",
                value,
              )
            }
            onKeyDown={onKeyDown}
          />

          <TextField
            label="学生学会什么 *"
            value={
              draft.learningObjective
            }
            maximumRunes={
              COURSEWARE_ASSISTANT_LIMITS
                .learningObjective
            }
            rows={4}
            placeholder="填写学生完成互动后能够表现出的可观察结果。"
            disabled={disabled}
            onChange={(value) =>
              setField(
                "learningObjective",
                value,
              )
            }
            onKeyDown={onKeyDown}
          />
        </CoursewareAssistantSection>
      )}
    </>
  );
}

function TextField({
  label,
  value,
  maximumRunes,
  rows,
  placeholder,
  disabled,
  onChange,
  onKeyDown,
}: TextFieldProps) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const length =
    coursewareAssistantRuneLength(
      value,
    );

  const overLimit =
    length > maximumRunes;

  return (
    <div
      style={{
        marginBottom: 12,
      }}
    >
      <label
        style={
          coursewareAssistantLabelStyle
        }
      >
        {label}
      </label>

      {rows === 1 ? (
        <input
          value={value}
          onChange={(event) =>
            onChange(
              event.target.value,
            )
          }
          onKeyDown={(event) => {
            onKeyDown?.(event);
          }}
          disabled={disabled}
          placeholder={placeholder}
          style={{
            ...coursewareAssistantInputStyle,
            borderColor:
              overLimit
                ? C.danger
                : C.border,
          }}
        />
      ) : (
        <textarea
          value={value}
          onChange={(event) =>
            onChange(
              event.target.value,
            )
          }
          onKeyDown={(event) => {
            onKeyDown?.(event);
          }}
          disabled={disabled}
          rows={rows}
          placeholder={placeholder}
          style={{
            ...coursewareAssistantInputStyle,
            resize: "vertical",
            borderColor:
              overLimit
                ? C.danger
                : C.border,
          }}
        />
      )}

      <div
        style={{
          marginTop: 3,
          color:
            overLimit
              ? C.danger
              : C.textMuted,
          fontSize: 9,
          textAlign: "right",
        }}
      >
        {length}/{maximumRunes}
      </div>
    </div>
  );
}
