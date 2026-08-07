/**
 * 教学智能体编辑器共享样式和轻量控件。
 *
 * 本文件不持有业务状态，不访问网络。
 */

import type {
  CSSProperties,
  KeyboardEvent,
  ReactNode,
} from "react";

import {
  coursewareAssistantRuneLength,
} from "./coursewareAssistantDraft";

export const COURSEWARE_ASSISTANT_EDITOR_COLORS = {
  primary: "#4F7BE8",
  primaryBackground:
    "rgba(79,123,232,0.07)",
  success: "#059669",
  warning: "#D97706",
  danger: "#DC2626",
  text: "#1F2937",
  textSecondary: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  background: "#F8FAFC",
  white: "#FFFFFF",
};

export const coursewareAssistantInputStyle:
  CSSProperties = {
    width: "100%",
    boxSizing: "border-box",
    padding: "8px 10px",
    borderRadius: 8,
    border:
      `1px solid ${COURSEWARE_ASSISTANT_EDITOR_COLORS.border}`,
    background: "#FFFFFF",
    color:
      COURSEWARE_ASSISTANT_EDITOR_COLORS.text,
    fontSize: 12,
    lineHeight: 1.6,
    outline: "none",
    fontFamily: "inherit",
  };

export const coursewareAssistantLabelStyle:
  CSSProperties = {
    display: "block",
    marginBottom: 5,
    color:
      COURSEWARE_ASSISTANT_EDITOR_COLORS
        .textSecondary,
    fontSize: 11,
    fontWeight: 700,
  };

interface CoursewareAssistantSectionProps {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
}

export function CoursewareAssistantSection({
  title,
  description,
  actions,
  children,
}: CoursewareAssistantSectionProps) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <section
      style={{
        marginTop: 14,
        padding: 14,
        borderRadius: 11,
        border:
          `1px solid ${C.border}`,
        background: C.white,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems:
            "flex-start",
          justifyContent:
            "space-between",
          gap: 10,
          marginBottom: 12,
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
              fontSize: 13,
              fontWeight: 700,
            }}
          >
            {title}
          </div>

          {description && (
            <div
              style={{
                marginTop: 3,
                color:
                  C.textSecondary,
                fontSize: 10,
                lineHeight: 1.6,
              }}
            >
              {description}
            </div>
          )}
        </div>

        {actions}
      </div>

      {children}
    </section>
  );
}

interface CoursewareAssistantStringListEditorProps {
  label: string;
  values: string[];
  onChange: (values: string[]) => void;
  disabled?: boolean;
  minimumItems?: number;
  maximumItems?: number;
  maximumRunes?: number;
  placeholder?: string;
  onKeyDown?: (
    event: KeyboardEvent<HTMLElement>,
  ) => boolean;
}

export function CoursewareAssistantStringListEditor({
  label,
  values,
  onChange,
  disabled = false,
  minimumItems = 0,
  maximumItems,
  maximumRunes,
  placeholder,
  onKeyDown,
}: CoursewareAssistantStringListEditorProps) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const updateValue = (
    index: number,
    value: string,
  ) => {
    onChange(
      values.map(
        (item, itemIndex) =>
          itemIndex === index
            ? value
            : item,
      ),
    );
  };

  const removeValue = (
    index: number,
  ) => {
    if (
      values.length <= minimumItems
    ) {
      return;
    }

    onChange(
      values.filter(
        (_, itemIndex) =>
          itemIndex !== index,
      ),
    );
  };

  const moveValue = (
    index: number,
    offset: -1 | 1,
  ) => {
    const target =
      index + offset;

    if (
      target < 0 ||
      target >= values.length
    ) {
      return;
    }

    const next = [...values];
    const [moved] =
      next.splice(index, 1);

    next.splice(
      target,
      0,
      moved,
    );

    onChange(next);
  };

  const canAdd =
    !disabled &&
    (
      maximumItems === undefined ||
      values.length < maximumItems
    );

  return (
    <div
      style={{
        marginBottom: 12,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent:
            "space-between",
          gap: 8,
          marginBottom: 6,
        }}
      >
        <label
          style={{
            ...coursewareAssistantLabelStyle,
            marginBottom: 0,
          }}
        >
          {label}
        </label>

        <button
          type="button"
          onClick={() =>
            onChange([
              ...values,
              "",
            ])
          }
          disabled={!canAdd}
          style={{
            padding: "4px 9px",
            borderRadius: 6,
            border:
              `1px dashed ${C.primary}`,
            background:
              C.primaryBackground,
            color: C.primary,
            fontSize: 10,
            cursor: canAdd
              ? "pointer"
              : "default",
            opacity: canAdd
              ? 1
              : 0.5,
          }}
        >
          ＋ 添加
        </button>
      </div>

      {values.length === 0 && (
        <div
          style={{
            padding: "9px 10px",
            borderRadius: 8,
            border:
              `1px dashed ${C.border}`,
            color: C.textMuted,
            fontSize: 10,
          }}
        >
          暂无内容，可点击“添加”。
        </div>
      )}

      {values.map(
        (value, index) => {
          const length =
            coursewareAssistantRuneLength(
              value,
            );

          const overLimit =
            maximumRunes !== undefined &&
            length > maximumRunes;

          return (
            <div
              key={index}
              style={{
                display: "grid",
                gridTemplateColumns:
                  "28px minmax(0, 1fr) auto",
                gap: 6,
                alignItems:
                  "start",
                marginBottom: 7,
              }}
            >
              <div
                style={{
                  paddingTop: 8,
                  color:
                    C.textMuted,
                  fontSize: 10,
                  textAlign:
                    "center",
                }}
              >
                {index + 1}
              </div>

              <div>
                <textarea
                  value={value}
                  onChange={(event) =>
                    updateValue(
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
                  rows={2}
                  placeholder={
                    placeholder
                  }
                  style={{
                    ...coursewareAssistantInputStyle,
                    resize:
                      "vertical",
                    borderColor:
                      overLimit
                        ? C.danger
                        : C.border,
                  }}
                />

                {maximumRunes !==
                  undefined && (
                  <div
                    style={{
                      marginTop: 2,
                      color:
                        overLimit
                          ? C.danger
                          : C.textMuted,
                      fontSize: 9,
                      textAlign:
                        "right",
                    }}
                  >
                    {length}/
                    {maximumRunes}
                  </div>
                )}
              </div>

              <div
                style={{
                  display: "flex",
                  gap: 3,
                  paddingTop: 2,
                }}
              >
                <button
                  type="button"
                  onClick={() =>
                    moveValue(
                      index,
                      -1,
                    )
                  }
                  disabled={
                    disabled ||
                    index === 0
                  }
                  title="上移"
                  style={miniButtonStyle(
                    disabled ||
                    index === 0,
                  )}
                >
                  ↑
                </button>

                <button
                  type="button"
                  onClick={() =>
                    moveValue(
                      index,
                      1,
                    )
                  }
                  disabled={
                    disabled ||
                    index ===
                      values.length - 1
                  }
                  title="下移"
                  style={miniButtonStyle(
                    disabled ||
                    index ===
                      values.length - 1,
                  )}
                >
                  ↓
                </button>

                <button
                  type="button"
                  onClick={() =>
                    removeValue(
                      index,
                    )
                  }
                  disabled={
                    disabled ||
                    values.length <=
                      minimumItems
                  }
                  title="删除"
                  style={{
                    ...miniButtonStyle(
                      disabled ||
                      values.length <=
                        minimumItems,
                    ),
                    color: C.danger,
                  }}
                >
                  ×
                </button>
              </div>
            </div>
          );
        },
      )}
    </div>
  );
}

function miniButtonStyle(
  disabled: boolean,
): CSSProperties {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return {
    width: 26,
    height: 26,
    padding: 0,
    borderRadius: 6,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    color:
      C.textSecondary,
    fontSize: 11,
    cursor: disabled
      ? "default"
      : "pointer",
    opacity: disabled
      ? 0.4
      : 1,
  };
}
