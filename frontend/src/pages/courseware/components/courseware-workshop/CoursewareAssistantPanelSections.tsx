/**
 * 教学智能体主面板的纯展示组件。
 *
 * 本文件不访问网络、不持有教学草稿，也不执行保存和发布操作。
 * 它集中提供主Tab、摘要、页面状态、校验提示和底部保存栏，
 * 让CoursewareAssistantPanel只负责业务组件编排。
 */

import type {
  CSSProperties,
} from "react";

import type {
  CoursewareAssistantSelectedPage,
} from "./coursewareAssistantSelection";

import {
  COURSEWARE_ASSISTANT_TEACHING_MODE_OPTIONS,
  type CoursewareAssistantEditorDraft,
} from "./coursewareAssistantDraft";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
} from "./CoursewareAssistantEditorShared";

export interface CoursewareAssistantTabOption<
  Value extends string,
> {
  value: Value;
  label: string;
  description: string;
}

export function CoursewareAssistantTabBar<
  Value extends string,
>({
  options,
  value,
  onChange,
  disabled,
  compact = false,
}: {
  options:
    readonly CoursewareAssistantTabOption<Value>[];
  value: Value;
  onChange: (value: Value) => void;
  disabled: boolean;
  compact?: boolean;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      role="tablist"
      style={{
        display: "grid",
        gridTemplateColumns:
          `repeat(${options.length}, minmax(0, 1fr))`,
        gap: compact ? 6 : 8,
        marginTop: compact ? 12 : 4,
        marginBottom: 12,
        padding: compact ? 5 : 6,
        borderRadius: 11,
        border: `1px solid ${C.border}`,
        background: compact
          ? C.background
          : C.white,
      }}
    >
      {options.map((option) => {
        const active =
          option.value === value;

        return (
          <button
            key={option.value}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() =>
              onChange(option.value)
            }
            disabled={disabled}
            style={{
              minWidth: 0,
              padding: compact
                ? "8px 7px"
                : "10px 8px",
              borderRadius: 8,
              border: active
                ? `1px solid ${C.primary}`
                : "1px solid transparent",
              background: active
                ? C.primaryBackground
                : "transparent",
              color: active
                ? C.primary
                : C.textSecondary,
              cursor: disabled
                ? "default"
                : "pointer",
              opacity: disabled ? 0.55 : 1,
              textAlign: "center",
            }}
          >
            <strong
              style={{
                display: "block",
                fontSize: compact ? 10 : 11,
                lineHeight: 1.4,
              }}
            >
              {option.label}
            </strong>

            {!compact && (
              <span
                style={{
                  display: "block",
                  marginTop: 3,
                  fontSize: 8,
                  lineHeight: 1.4,
                }}
              >
                {option.description}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}

export function CoursewareAssistantCurrentPageCard({
  selectedPage,
}: {
  selectedPage:
    CoursewareAssistantSelectedPage;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      title={selectedPage.pageId}
      style={{
        marginBottom: 12,
        padding: "10px 12px",
        borderRadius: 9,
        border:
          "1px solid rgba(79,123,232,0.24)",
        background:
          C.primaryBackground,
      }}
    >
      <div
        style={{
          color: C.text,
          fontSize: 12,
          fontWeight: 700,
        }}
      >
        当前页面：P
        {selectedPage.pageNumber}
        {" · "}
        {selectedPage.pageTitle ||
          "未命名页面"}
      </div>

      <div
        style={{
          marginTop: 3,
          color: C.textMuted,
          fontSize: 9,
        }}
      >
        页面重排后，教学智能体仍会跟随当前页面。
      </div>
    </div>
  );
}

export function CoursewareAssistantNextStepCard({
  title,
  description,
  buttonLabel,
  disabled,
  onClick,
}: {
  title: string;
  description: string;
  buttonLabel: string;
  disabled: boolean;
  onClick: () => void;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        gap: 12,
        marginTop: 12,
        padding: 13,
        borderRadius: 10,
        border: `1px solid ${C.border}`,
        background: C.white,
        flexWrap: "wrap",
      }}
    >
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
            marginTop: 3,
            color: C.textSecondary,
            fontSize: 9,
            lineHeight: 1.6,
          }}
        >
          {description}
        </div>
      </div>

      <button
        type="button"
        onClick={onClick}
        disabled={disabled}
        style={
          coursewareAssistantPrimaryButtonStyle(
            disabled,
          )
        }
      >
        {buttonLabel}
      </button>
    </div>
  );
}

export function CoursewareAssistantPlanSummary({
  draft,
  generating,
}: {
  draft:
    CoursewareAssistantEditorDraft;
  generating: boolean;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const mode =
    COURSEWARE_ASSISTANT_TEACHING_MODE_OPTIONS.find(
      (option) =>
        option.value ===
        draft.guidancePlan.teaching_mode,
    );

  const ready =
    Boolean(
      draft.title.trim() &&
      draft.welcomeMessage.trim() &&
      draft.guidancePlan.question_chain.some(
        (step) =>
          step.prompt.trim(),
      ),
    );

  const visibleSteps =
    draft.guidancePlan.question_chain
      .filter(
        (step) =>
          step.prompt.trim(),
      );

  return (
    <section
      style={{
        padding: 14,
        borderRadius: 11,
        border: `1px solid ${C.border}`,
        background: C.white,
      }}
    >
      <div
        style={{
          color: C.text,
          fontSize: 13,
          fontWeight: 700,
        }}
      >
        智能体会这样教学
      </div>

      {!ready && (
        <div
          style={{
            marginTop: 10,
            padding: "18px 12px",
            borderRadius: 9,
            border:
              `1px dashed ${C.border}`,
            background: C.background,
            color: C.textSecondary,
            fontSize: 10,
            lineHeight: 1.7,
            textAlign: "center",
          }}
        >
          {generating
            ? "正在根据当前页面生成学生互动…"
            : "请先在“学生怎么学”中选择方式并生成互动。"}
        </div>
      )}

      {ready && (
        <>
          <SummaryRow
            label="学习方式"
            content={
              mode?.title ||
              "一步步想明白"
            }
          />

          <SummaryRow
            label="开场"
            content={
              draft.welcomeMessage
            }
          />

          <div
            style={{
              marginTop: 11,
            }}
          >
            <div
              style={{
                color: C.textSecondary,
                fontSize: 9,
                fontWeight: 700,
              }}
            >
              互动过程
            </div>

            <ol
              style={{
                margin:
                  "6px 0 0 20px",
                padding: 0,
                color: C.text,
                fontSize: 10,
                lineHeight: 1.7,
              }}
            >
              {visibleSteps
                .slice(0, 4)
                .map(
                  (step, index) => (
                    <li
                      key={
                        `${step.id}-${index}`
                      }
                    >
                      {step.prompt}
                    </li>
                  ),
                )}
            </ol>

            {visibleSteps.length >
              4 && (
              <div
                style={{
                  marginTop: 4,
                  color: C.textMuted,
                  fontSize: 9,
                }}
              >
                其余步骤可在“学生互动”Tab中查看。
              </div>
            )}
          </div>

          <SummaryRow
            label="完成标准"
            content={
              draft.guidancePlan
                .completion_criteria
                .filter(Boolean)
                .slice(0, 2)
                .join("；") ||
              "由系统根据页面目标判断学生是否已经理解。"
            }
          />
        </>
      )}
    </section>
  );
}

function SummaryRow({
  label,
  content,
}: {
  label: string;
  content: string;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns:
          "72px minmax(0, 1fr)",
        gap: 9,
        marginTop: 10,
        alignItems: "start",
      }}
    >
      <div
        style={{
          color: C.textSecondary,
          fontSize: 9,
          fontWeight: 700,
        }}
      >
        {label}
      </div>

      <div
        style={{
          color: C.text,
          fontSize: 10,
          lineHeight: 1.65,
        }}
      >
        {content}
      </div>
    </div>
  );
}

export function CoursewareAssistantAdvancedToolbar({
  hasSlot,
  busy,
  deleting,
  canUndo,
  canRedo,
  onUndo,
  onRedo,
  onRefresh,
  onDelete,
}: {
  hasSlot: boolean;
  busy: boolean;
  deleting: boolean;
  canUndo: boolean;
  canRedo: boolean;
  onUndo: () => void;
  onRedo: () => void;
  onRefresh: () => void;
  onDelete: () => void;
}) {
  return (
    <div
      style={{
        display: "flex",
        justifyContent: "flex-end",
        gap: 6,
        marginBottom: 10,
        flexWrap: "wrap",
      }}
    >
      <ToolbarButton
        label="撤销"
        disabled={
          busy ||
          !canUndo
        }
        onClick={onUndo}
      />

      <ToolbarButton
        label="重做"
        disabled={
          busy ||
          !canRedo
        }
        onClick={onRedo}
      />

      <ToolbarButton
        label="重新读取"
        disabled={busy}
        onClick={onRefresh}
      />

      {hasSlot && (
        <ToolbarButton
          label={
            deleting
              ? "删除中…"
              : "删除方案"
          }
          disabled={busy}
          danger
          onClick={onDelete}
        />
      )}
    </div>
  );
}

function ToolbarButton({
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
        padding: "6px 10px",
        borderRadius: 7,
        border: `1px solid ${C.border}`,
        background: C.white,
        color: danger
          ? C.danger
          : C.textSecondary,
        fontSize: 10,
        fontWeight: 700,
        cursor: disabled
          ? "default"
          : "pointer",
        opacity: disabled
          ? 0.45
          : 1,
      }}
    >
      {label}
    </button>
  );
}

export function CoursewareAssistantValidationBox({
  errors,
}: {
  errors: string[];
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      style={{
        marginBottom: 12,
        padding: "10px 12px",
        borderRadius: 9,
        border:
          "1px solid #FECACA",
        background: "#FEF2F2",
        color: C.danger,
        fontSize: 10,
        lineHeight: 1.7,
      }}
    >
      <strong>
        请先修正以下问题：
      </strong>

      <ol
        style={{
          margin:
            "6px 0 0 18px",
          padding: 0,
        }}
      >
        {errors.map(
          (error) => (
            <li key={error}>
              {error}
            </li>
          ),
        )}
      </ol>
    </div>
  );
}

export function CoursewareAssistantStickySaveBar({
  isDirty,
  hasSlot,
  saving,
  busy,
  onSave,
}: {
  isDirty: boolean;
  hasSlot: boolean;
  saving: boolean;
  busy: boolean;
  onSave: () => void;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      style={{
        position: "sticky",
        bottom: 8,
        zIndex: 10,
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        gap: 10,
        marginTop: 14,
        padding: "10px 12px",
        borderRadius: 10,
        border: `1px solid ${C.border}`,
        background:
          "rgba(255,255,255,0.96)",
        boxShadow:
          "0 8px 24px rgba(15,23,42,0.10)",
      }}
    >
      <div
        style={{
          color: isDirty
            ? C.warning
            : C.success,
          fontSize: 10,
          fontWeight: 700,
        }}
      >
        {isDirty
          ? "当前方案尚未保存"
          : hasSlot
            ? "方案已保存"
            : "尚未生成并保存方案"}
      </div>

      <button
        type="button"
        onClick={onSave}
        disabled={busy}
        style={
          coursewareAssistantPrimaryButtonStyle(
            busy,
          )
        }
      >
        {saving
          ? "保存中…"
          : "保存方案"}
      </button>
    </div>
  );
}

export function CoursewareAssistantPanelHeader({
  selectedPage,
  hasSlot,
  isDirty,
}: {
  selectedPage:
    CoursewareAssistantSelectedPage | null;
  hasSlot: boolean;
  isDirty: boolean;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      style={{
        display: "flex",
        alignItems: "flex-start",
        justifyContent: "space-between",
        gap: 12,
        marginBottom: 12,
        flexWrap: "wrap",
      }}
    >
      <div
        style={{
          minWidth: 0,
          flex: 1,
        }}
      >
        <div
          style={{
            color: C.text,
            fontSize: 14,
            fontWeight: 700,
          }}
        >
          页面教学智能体
        </div>

        <div
          style={{
            marginTop: 4,
            color: C.textSecondary,
            fontSize: 10,
            lineHeight: 1.65,
          }}
        >
          按三个步骤完成：选择学习方式、检查互动内容、设置课堂使用方式。
        </div>

        {selectedPage && (
          <div
            style={{
              marginTop: 6,
              display: "flex",
              gap: 6,
              flexWrap: "wrap",
            }}
          >
            <StatusBadge
              text={
                hasSlot
                  ? "已有方案"
                  : "尚未保存"
              }
              color={
                hasSlot
                  ? C.success
                  : C.primary
              }
            />

            <StatusBadge
              text={
                isDirty
                  ? "有未保存修改"
                  : "已同步"
              }
              color={
                isDirty
                  ? C.warning
                  : C.textMuted
              }
            />
          </div>
        )}
      </div>
    </div>
  );
}

function StatusBadge({
  text,
  color,
}: {
  text: string;
  color: string;
}) {
  return (
    <span
      style={{
        padding: "2px 7px",
        borderRadius: 999,
        background:
          `${color}14`,
        color,
        fontSize: 9,
        fontWeight: 700,
      }}
    >
      {text}
    </span>
  );
}

export function CoursewareAssistantMessageBox({
  kind,
  text,
}: {
  kind:
    | "success"
    | "info"
    | "error"
    | "warning";
  text: string;
}) {
  const styles = {
    success: {
      border: "#A7F3D0",
      background: "#ECFDF5",
      color: "#047857",
    },
    info: {
      border: "#BFDBFE",
      background: "#EFF6FF",
      color: "#2563EB",
    },
    error: {
      border: "#FECACA",
      background: "#FEF2F2",
      color: "#B91C1C",
    },
    warning: {
      border: "#FDE68A",
      background: "#FFFBEB",
      color: "#92400E",
    },
  }[kind];

  return (
    <div
      style={{
        marginBottom: 12,
        padding: "10px 12px",
        borderRadius: 9,
        border:
          `1px solid ${styles.border}`,
        background:
          styles.background,
        color: styles.color,
        fontSize: 11,
        lineHeight: 1.65,
      }}
    >
      {text}
    </div>
  );
}

export function coursewareAssistantPrimaryButtonStyle(
  disabled: boolean,
): CSSProperties {
  return {
    padding: "9px 16px",
    borderRadius: 9,
    border: "none",
    background: "#4F7BE8",
    color: "#FFFFFF",
    fontSize: 11,
    fontWeight: 700,
    cursor: disabled
      ? "default"
      : "pointer",
    opacity: disabled
      ? 0.5
      : 1,
  };
}

export function coursewareAssistantSecondaryButtonStyle(
  disabled: boolean,
): CSSProperties {
  return {
    padding: "9px 16px",
    borderRadius: 9,
    border:
      "1px solid #E2E8F0",
    background: "#FFFFFF",
    color: "#4F7BE8",
    fontSize: 11,
    fontWeight: 700,
    cursor: disabled
      ? "default"
      : "pointer",
    opacity: disabled
      ? 0.5
      : 1,
  };
}
