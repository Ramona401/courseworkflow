/**
 * 教学智能体发布策略的共享展示控件。
 *
 * 本文件不决定预设含义，也不构造后端请求。
 */

import type {
  Dispatch,
  ReactNode,
  SetStateAction,
} from "react";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
} from "./CoursewareAssistantEditorShared";

import type {
  CoursewareAssistantDeploymentChoiceOption,
  CoursewareAssistantDeploymentPolicyDraft,
} from "./coursewareAssistantDeploymentPolicyModel";

import {
  coursewareAssistantDeploymentDetailsStyle,
  coursewareAssistantDeploymentInputStyle,
  coursewareAssistantDeploymentSummaryStyle,
} from "./CoursewareAssistantDeploymentShared";

export function CoursewareAssistantDeploymentPresetSection<
  Value extends string,
>({
  number,
  title,
  description,
  options,
  selected,
  disabled,
  onSelect,
  children,
}: {
  number: string;
  title: string;
  description: string;
  options:
    readonly CoursewareAssistantDeploymentChoiceOption<Value>[];
  selected: Value;
  disabled: boolean;
  onSelect: (
    value: Value,
  ) => void;
  children?: ReactNode;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <section
      style={{
        paddingBottom: 14,
        marginBottom: 14,
        borderBottom:
          `1px solid ${C.border}`,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems:
            "flex-start",
          gap: 9,
        }}
      >
        <span
          style={{
            display: "inline-flex",
            alignItems: "center",
            justifyContent:
              "center",
            width: 22,
            height: 22,
            flex: "0 0 auto",
            borderRadius: 999,
            background:
              C.primary,
            color: C.white,
            fontSize: 10,
            fontWeight: 700,
          }}
        >
          {number}
        </span>

        <div>
          <div
            style={{
              color: C.text,
              fontSize: 11,
              fontWeight: 700,
            }}
          >
            {title}
          </div>

          <div
            style={{
              marginTop: 2,
              color:
                C.textSecondary,
              fontSize: 9,
              lineHeight: 1.55,
            }}
          >
            {description}
          </div>
        </div>
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns:
            "repeat(auto-fit, minmax(125px, 1fr))",
          gap: 8,
          marginTop: 10,
        }}
      >
        {options.map(
          (option) => {
            const active =
              option.value ===
              selected;

            return (
              <button
                key={option.value}
                type="button"
                onClick={() =>
                  onSelect(
                    option.value,
                  )
                }
                disabled={disabled}
                style={{
                  padding:
                    "10px 9px",
                  borderRadius: 9,
                  border: active
                    ? `1px solid ${C.primary}`
                    : `1px solid ${C.border}`,
                  background: active
                    ? C.primaryBackground
                    : C.white,
                  color: active
                    ? "#315FC9"
                    : "#334155",
                  textAlign: "left",
                  cursor: disabled
                    ? "default"
                    : "pointer",
                  opacity: disabled
                    ? 0.5
                    : 1,
                }}
              >
                <strong
                  style={{
                    display: "block",
                    fontSize: 10,
                  }}
                >
                  {option.title}
                </strong>

                <span
                  style={{
                    display: "block",
                    marginTop: 3,
                    color:
                      C.textSecondary,
                    fontSize: 8,
                    lineHeight: 1.45,
                  }}
                >
                  {option.description}
                </span>
              </button>
            );
          },
        )}
      </div>

      {children}
    </section>
  );
}

export function CoursewareAssistantDeploymentNumberField({
  label,
  value,
  minimum,
  maximum,
  disabled,
  onChange,
}: {
  label: string;
  value: string;
  minimum: number;
  maximum: number;
  disabled: boolean;
  onChange: (
    value: string,
  ) => void;
}) {
  return (
    <label
      style={{
        display: "block",
        marginTop: 10,
        maxWidth: 240,
      }}
    >
      <span
        style={{
          display: "block",
          marginBottom: 5,
          color: "#64748B",
          fontSize: 9,
          fontWeight: 700,
        }}
      >
        {label}
      </span>

      <input
        type="number"
        min={minimum}
        max={maximum}
        value={value}
        disabled={disabled}
        onChange={(event) =>
          onChange(
            event.target.value,
          )
        }
        style={
          coursewareAssistantDeploymentInputStyle(
            disabled,
          )
        }
      />
    </label>
  );
}

export function CoursewareAssistantDeploymentDateTimeField({
  value,
  disabled,
  onChange,
}: {
  value: string;
  disabled: boolean;
  onChange: (
    value: string,
  ) => void;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <label
      style={{
        display: "block",
        marginTop: 10,
        maxWidth: 300,
      }}
    >
      <span
        style={{
          display: "block",
          marginBottom: 5,
          color:
            C.textSecondary,
          fontSize: 9,
          fontWeight: 700,
        }}
      >
        使用结束时间
      </span>

      <input
        type="datetime-local"
        value={value}
        disabled={disabled}
        onChange={(event) =>
          onChange(
            event.target.value,
          )
        }
        style={
          coursewareAssistantDeploymentInputStyle(
            disabled,
          )
        }
      />
    </label>
  );
}

export function CoursewareAssistantDeploymentPolicySummary({
  policy,
}: {
  policy:
    CoursewareAssistantDeploymentPolicyDraft;
}) {
  const students =
    Number(
      policy.expectedStudents,
    );

  const turns =
    Number(
      policy
        .perSessionTurnLimit,
    );

  return (
    <div
      style={{
        marginTop: 14,
        padding: "10px 11px",
        borderRadius: 9,
        background:
          "#EFF6FF",
        color: "#1D4ED8",
        fontSize: 10,
        lineHeight: 1.7,
      }}
    >
      当前设置：预计{" "}
      {Number.isFinite(
        students,
      )
        ? students
        : 0}
      {" "}名学生，每人最多{" "}
      {Number.isFinite(
        turns,
      )
        ? turns
        : 0}
      {" "}轮。系统每天预留{" "}
      {policy.dailyCallLimit}
      {" "}次调用，并默认只允许当前TE-DNA平台打开。
    </div>
  );
}

export function CoursewareAssistantDeploymentProfessionalSettings({
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
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <details
      style={
        coursewareAssistantDeploymentDetailsStyle()
      }
    >
      <summary
        style={
          coursewareAssistantDeploymentSummaryStyle(
            disabled,
          )
        }
      >
        专业发布设置
      </summary>

      <div
        style={{
          marginTop: 10,
        }}
      >
        <CoursewareAssistantDeploymentNumberField
          label="精确每日调用额度"
          value={
            policy.dailyCallLimit
          }
          minimum={1}
          maximum={100000}
          disabled={disabled}
          onChange={(value) =>
            setPolicy(
              (previous) => ({
                ...previous,
                dailyCallLimit:
                  value,
              }),
            )
          }
        />

        <label
          style={{
            display: "flex",
            alignItems:
              "flex-start",
            gap: 8,
            marginTop: 12,
            cursor: disabled
              ? "default"
              : "pointer",
          }}
        >
          <input
            type="checkbox"
            checked={
              policy.externalEnabled
            }
            disabled={disabled}
            onChange={(event) =>
              setPolicy(
                (previous) => ({
                  ...previous,
                  externalEnabled:
                    event.target
                      .checked,
                }),
              )
            }
            style={{
              marginTop: 2,
            }}
          />

          <span>
            <span
              style={{
                display: "block",
                color: C.text,
                fontSize: 10,
                fontWeight: 700,
              }}
            >
              允许外部课件平台调用
            </span>

            <span
              style={{
                display: "block",
                marginTop: 2,
                color:
                  C.textSecondary,
                fontSize: 9,
                lineHeight: 1.6,
              }}
            >
              默认关闭。关闭时仅允许当前站点：
              {internalOrigin ||
                "无法识别"}
            </span>
          </span>
        </label>

        {policy.externalEnabled && (
          <label
            style={{
              display: "block",
              marginTop: 10,
            }}
          >
            <span
              style={{
                display: "block",
                marginBottom: 5,
                color:
                  C.textSecondary,
                fontSize: 9,
                fontWeight: 700,
              }}
            >
              外部允许来源
            </span>

            <textarea
              value={
                policy.externalOrigins
              }
              disabled={disabled}
              rows={4}
              placeholder={
                "https://course.example.edu\nhttps://lms.example.edu"
              }
              onChange={(event) =>
                setPolicy(
                  (previous) => ({
                    ...previous,
                    externalOrigins:
                      event.target.value,
                  }),
                )
              }
              style={{
                ...coursewareAssistantDeploymentInputStyle(
                  disabled,
                ),
                resize: "vertical",
                minHeight: 88,
              }}
            />

            <div
              style={{
                marginTop: 4,
                color: C.textMuted,
                fontSize: 8,
                lineHeight: 1.6,
              }}
            >
              每行填写一个精确HTTPS来源，不包含路径、通配符、查询参数或账号密码。
            </div>
          </label>
        )}
      </div>
    </details>
  );
}
