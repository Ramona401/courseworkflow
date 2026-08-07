/**
 * CWAIReviewConfigCard.tsx
 *
 * R-02审核维度与教案参考模式选择卡片。
 *
 * 配置创建会话后即成为不可变快照：
 *   - 活动会话继续执行时只读展示；
 *   - 已完成、失败或取消的会话可以重新选择；
 *   - 点击重新开始会创建新会话，不修改历史会话；
 *   - 自定义维度必须填写具体要求。
 */

import type {
  CSSProperties,
} from "react";

import {
  CW_AI_REVIEW_DIMENSION_OPTIONS,
  CW_AI_REVIEW_LESSON_REFERENCE_OPTIONS,
  getCWAIReviewLessonReferenceLabel,
  type CWAIReviewDimension,
  type CWAIReviewLessonReferenceMode,
} from "@/api/coursewares.ai-review-config";

import type {
  CWAIReviewController,
} from "./useCWAIReviewController";

const C = {
  primary: "#4F7BE8",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  soft: "#F8FAFC",
  selected: "#EEF4FF",
  warning: "#9A3412",
};

const DIMENSION_DESCRIPTIONS:
  Record<
    CWAIReviewDimension,
    string
  > = {
  teaching_logic:
    "教学目标、环节顺序、跨页衔接与认知递进",
  technical_implementation:
    "HTML、CSS、脚本、依赖和运行实现",
  interaction_experience:
    "互动入口、反馈、答案暴露和交互体验",
  lesson_alignment:
    "课件与来源教案、课程大纲的匹配程度",
  authenticity:
    "案例、数据、媒体和教学情境的真实性",
  knowledge_accuracy:
    "概念、事实、公式、结论和学科边界",
  page_readability:
    "字号、排版、信息密度、层级和视觉负担",
  operational_usability:
    "点击、触控、键盘、可达性和实际可操作性",
  custom:
    "补充本次审核需要特别关注的自定义要求",
};

const LESSON_MODE_DESCRIPTIONS:
  Record<
    CWAIReviewLessonReferenceMode,
    string
  > = {
  current_compatible:
    "保持现有审核行为，综合参考教案、大纲和对齐报告。",
  strict_alignment:
    "严格核对目标、环节、案例、活动顺序和知识边界。",
  lesson_intent:
    "参考核心教学意图，允许课件采用不同表达和页面组织。",
  no_lesson:
    "后端不读取教案正文、大纲或对齐报告，也不把它们输入AI。",
};

export interface CWAIReviewConfigCardProps {
  controller: CWAIReviewController;
  hasLessonPlan: boolean;
}

export default function CWAIReviewConfigCard({
  controller,
  hasLessonPlan,
}: CWAIReviewConfigCardProps) {
  const draft =
    controller.reviewConfigDraft;

  const editable =
    controller.reviewConfigEditable;

  const hasCustom =
    draft.review_dimensions
      .includes("custom");

  const sessionConfig =
    controller.sessionReviewConfig;

  const configHash =
    sessionConfig
      ?.review_config_hash
      ?.trim() || "";

  return (
    <div
      style={{
        marginBottom: "12px",
        padding: "11px",
        borderRadius: "9px",
        border:
          `1px solid ${C.border}`,
        background: C.soft,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: "8px",
          marginBottom: "9px",
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
              fontSize: "12px",
              fontWeight: 700,
              color: C.text,
            }}
          >
            审核范围与教案参考方式
          </div>

          <div
            style={{
              marginTop: "3px",
              fontSize: "10px",
              lineHeight: 1.55,
              color: C.textSec,
            }}
          >
            至少选择一个审核维度。启动后配置会固化到本次会话、全部批次和最终报告。
          </div>
        </div>

        <span
          style={{
            flexShrink: 0,
            padding: "3px 6px",
            borderRadius: "999px",
            border:
              `1px solid ${
                editable
                  ? "#BFDBFE"
                  : "#CBD5E1"
              }`,
            background:
              editable
                ? "#EFF6FF"
                : "#F1F5F9",
            color:
              editable
                ? "#1D4ED8"
                : C.textSec,
            fontSize: "9px",
            fontWeight: 700,
          }}
        >
          {editable
            ? "可配置"
            : "会话快照"}
        </span>
      </div>

      <div
        style={{
          marginBottom: "7px",
          fontSize: "10px",
          fontWeight: 700,
          color: C.textSec,
        }}
      >
        审核维度
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns:
            "repeat(2, minmax(0, 1fr))",
          gap: "6px",
        }}
      >
        {CW_AI_REVIEW_DIMENSION_OPTIONS.map(
          (option) => {
            const checked =
              draft.review_dimensions
                .includes(
                  option.code,
                );

            return (
              <label
                key={option.code}
                style={{
                  display: "flex",
                  alignItems:
                    "flex-start",
                  gap: "6px",
                  padding: "7px",
                  borderRadius: "7px",
                  border:
                    `1px solid ${
                      checked
                        ? "#93C5FD"
                        : C.border
                    }`,
                  background:
                    checked
                      ? C.selected
                      : "#FFFFFF",
                  cursor:
                    editable
                      ? "pointer"
                      : "default",
                  opacity:
                    editable
                      ? 1
                      : 0.82,
                }}
              >
                <input
                  type="checkbox"
                  checked={checked}
                  disabled={!editable}
                  onChange={(event) =>
                    controller
                      .handleToggleReviewDimension(
                        option.code,
                        event.target
                          .checked,
                      )
                  }
                  style={{
                    marginTop: "1px",
                    flexShrink: 0,
                  }}
                />

                <span
                  style={{
                    minWidth: 0,
                  }}
                >
                  <span
                    style={{
                      display: "block",
                      color: C.text,
                      fontSize: "10px",
                      fontWeight: 700,
                    }}
                  >
                    {option.label}
                  </span>

                  <span
                    style={{
                      display: "block",
                      marginTop: "2px",
                      color:
                        C.textMuted,
                      fontSize: "9px",
                      lineHeight: 1.45,
                    }}
                  >
                    {
                      DIMENSION_DESCRIPTIONS[
                        option.code
                      ]
                    }
                  </span>
                </span>
              </label>
            );
          },
        )}
      </div>

      {hasCustom && (
        <div
          style={{
            marginTop: "8px",
          }}
        >
          <label
            htmlFor="cw-ai-review-custom-dimension"
            style={{
              display: "block",
              marginBottom: "4px",
              color: C.textSec,
              fontSize: "10px",
              fontWeight: 700,
            }}
          >
            自定义审核要求
          </label>

          <textarea
            id="cw-ai-review-custom-dimension"
            value={
              draft
                .custom_dimension_description
            }
            disabled={!editable}
            maxLength={1000}
            rows={3}
            onChange={(event) =>
              controller
                .setCustomDimensionDescription(
                  event.target.value,
                )
            }
            placeholder="例如：重点检查实验步骤是否符合本校器材条件，并关注学生安全提示。"
            style={{
              width: "100%",
              boxSizing:
                "border-box",
              padding: "7px 8px",
              resize: "vertical",
              borderRadius: "7px",
              border:
                `1px solid ${C.border}`,
              background:
                editable
                  ? "#FFFFFF"
                  : "#F1F5F9",
              color: C.text,
              fontSize: "10px",
              lineHeight: 1.55,
              outline: "none",
            }}
          />
        </div>
      )}

      <div
        style={{
          marginTop: "11px",
          marginBottom: "7px",
          fontSize: "10px",
          fontWeight: 700,
          color: C.textSec,
        }}
      >
        教案参考模式
      </div>

      <div
        style={{
          display: "flex",
          flexDirection: "column",
          gap: "6px",
        }}
      >
        {CW_AI_REVIEW_LESSON_REFERENCE_OPTIONS.map(
          (option) => {
            const checked =
              draft
                .lesson_reference_mode ===
              option.code;

            return (
              <label
                key={option.code}
                style={{
                  display: "flex",
                  alignItems:
                    "flex-start",
                  gap: "7px",
                  padding: "7px 8px",
                  borderRadius: "7px",
                  border:
                    `1px solid ${
                      checked
                        ? "#93C5FD"
                        : C.border
                    }`,
                  background:
                    checked
                      ? C.selected
                      : "#FFFFFF",
                  cursor:
                    editable
                      ? "pointer"
                      : "default",
                  opacity:
                    editable
                      ? 1
                      : 0.82,
                }}
              >
                <input
                  type="radio"
                  name="cw-ai-review-lesson-mode"
                  value={option.code}
                  checked={checked}
                  disabled={!editable}
                  onChange={() =>
                    controller
                      .setLessonReferenceMode(
                        option.code,
                      )
                  }
                  style={{
                    marginTop: "1px",
                    flexShrink: 0,
                  }}
                />

                <span
                  style={{
                    minWidth: 0,
                  }}
                >
                  <span
                    style={{
                      display: "block",
                      color: C.text,
                      fontSize: "10px",
                      fontWeight: 700,
                    }}
                  >
                    {option.label}
                  </span>

                  <span
                    style={{
                      display: "block",
                      marginTop: "2px",
                      color:
                        C.textMuted,
                      fontSize: "9px",
                      lineHeight: 1.45,
                    }}
                  >
                    {
                      LESSON_MODE_DESCRIPTIONS[
                        option.code
                      ]
                    }
                  </span>
                </span>
              </label>
            );
          },
        )}
      </div>

      {!hasLessonPlan &&
        draft.lesson_reference_mode !==
          "no_lesson" && (
        <div
          style={{
            marginTop: "8px",
            padding: "7px 8px",
            borderRadius: "7px",
            border:
              "1px solid #FDE68A",
            background:
              "#FFFBEB",
            color: "#92400E",
            fontSize: "9px",
            lineHeight: 1.55,
          }}
        >
          当前课件未显示来源教案关联。后端只会使用真实存在且教育域一致的材料，不会伪造教案内容。
        </div>
      )}

      {draft.lesson_reference_mode ===
        "no_lesson" && (
        <div
          style={{
            marginTop: "8px",
            padding: "7px 8px",
            borderRadius: "7px",
            border:
              "1px solid #FED7AA",
            background:
              "#FFF7ED",
            color: C.warning,
            fontSize: "9px",
            lineHeight: 1.55,
          }}
        >
          “不使用教案”是后端真实隔离模式：不会读取教案正文、课程大纲或对齐报告，也不会把来源教案ID写入AI调用追踪。
        </div>
      )}

      {controller.reviewConfigError && (
        <div
          style={{
            marginTop: "8px",
            color: "#DC2626",
            fontSize: "10px",
            lineHeight: 1.5,
          }}
        >
          ⚠️{" "}
          {
            controller
              .reviewConfigError
          }
        </div>
      )}

      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "6px",
          marginTop: "8px",
          paddingTop: "8px",
          borderTop:
            `1px solid ${C.border}`,
          color: C.textMuted,
          fontSize: "9px",
          lineHeight: 1.45,
          flexWrap: "wrap",
        }}
      >
        <span>
          当前模式：
          {getCWAIReviewLessonReferenceLabel(
            draft
              .lesson_reference_mode,
          )}
        </span>

        <span>
          · 已选{" "}
          {
            draft
              .review_dimensions
              .length
          }{" "}
          个维度
        </span>

        {configHash && (
          <span
            title={configHash}
          >
            · 配置指纹：
            {configHash.slice(
              0,
              10,
            )}
            …
          </span>
        )}
      </div>
    </div>
  );
}
