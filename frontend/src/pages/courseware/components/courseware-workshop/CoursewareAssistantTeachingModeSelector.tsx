/**
 * 教师端“学生怎么学”选择器。
 *
 * 产品原则：
 *   - 不要求教师理解教学法术语；
 *   - 默认只显示系统推荐的3种方式；
 *   - 教师可展开查看全部8种；
 *   - 推荐只影响排序和展示，不替教师自动做最终选择；
 *   - 所有方式最终仍通过后端安全协议生成完整互动方案。
 */

import {
  useMemo,
  useState,
} from "react";

import type {
  CoursewareAssistantTeachingMode,
} from "@/api/coursewares";

import {
  COURSEWARE_ASSISTANT_TEACHING_MODE_OPTIONS,
  type CoursewareAssistantEditorDraft,
  type CoursewareAssistantTeachingModeOption,
} from "./coursewareAssistantDraft";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
  CoursewareAssistantSection,
} from "./CoursewareAssistantEditorShared";

interface Props {
  draft: CoursewareAssistantEditorDraft;
  onChange: (
    draft: CoursewareAssistantEditorDraft,
  ) => void;
  subject: string;
  grade: string;
  pageTitle: string;
  pageSummary?: string;
  interactionType?: string;
  disabled?: boolean;
}

export default function CoursewareAssistantTeachingModeSelector({
  draft,
  onChange,
  subject,
  grade,
  pageTitle,
  pageSummary = "",
  interactionType = "",
  disabled = false,
}: Props) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const [
    showAll,
    setShowAll,
  ] = useState(false);

  const primaryGrade =
    isPrimaryGrade(grade);

  const options =
    useMemo(
      () =>
        COURSEWARE_ASSISTANT_TEACHING_MODE_OPTIONS.map(
          (option) =>
            localizeModeOption(
              option,
              primaryGrade,
            ),
        ),
      [
        primaryGrade,
      ],
    );

  const recommendedModes =
    useMemo(
      () =>
        recommendTeachingModes({
          subject,
          grade,
          pageTitle,
          pageSummary,
          interactionType,
        }),
      [
        grade,
        interactionType,
        pageSummary,
        pageTitle,
        subject,
      ],
    );

  const selectedMode =
    draft.guidancePlan
      .teaching_mode;

  const visibleOptions =
    useMemo(
      () => {
        if (showAll) {
          return options;
        }

        const recommended =
          recommendedModes
            .map((mode) =>
              options.find(
                (option) =>
                  option.value ===
                  mode,
              ),
            )
            .filter(
              (
                option,
              ): option is
                CoursewareAssistantTeachingModeOption =>
                Boolean(option),
            );

        if (
          !recommended.some(
            (option) =>
              option.value ===
              selectedMode,
          )
        ) {
          const selected =
            options.find(
              (option) =>
                option.value ===
                selectedMode,
            );

          if (selected) {
            return [
              recommended[0],
              recommended[1],
              selected,
            ].filter(
              (
                option,
              ): option is
                CoursewareAssistantTeachingModeOption =>
                Boolean(option),
            );
          }
        }

        return recommended.slice(0, 3);
      },
      [
        options,
        recommendedModes,
        selectedMode,
        showAll,
      ],
    );

  const selectMode = (
    mode:
      CoursewareAssistantTeachingMode,
  ) => {
    if (disabled) {
      return;
    }

    onChange({
      ...draft,
      guidancePlan: {
        ...draft.guidancePlan,
        version: "v2",
        teaching_mode: mode,
      },
    });
  };

  return (
    <CoursewareAssistantSection
      title="这页你希望学生怎么学？"
      description="系统已根据学科、年级和当前页面推荐3种方式。只需选择最符合本页目标的一种。"
    >
      <div
        style={{
          display: "grid",
          gridTemplateColumns:
            "repeat(auto-fit, minmax(210px, 1fr))",
          gap: 10,
        }}
      >
        {visibleOptions.map(
          (option) => {
            const active =
              option.value ===
              selectedMode;

            const recommended =
              recommendedModes.includes(
                option.value,
              );

            return (
              <button
                key={option.value}
                type="button"
                onClick={() =>
                  selectMode(
                    option.value,
                  )
                }
                disabled={disabled}
                style={{
                  padding: 13,
                  borderRadius: 11,
                  border:
                    `1px solid ${
                      active
                        ? C.primary
                        : C.border
                    }`,
                  background:
                    active
                      ? C.primaryBackground
                      : C.white,
                  color: C.text,
                  textAlign: "left",
                  cursor:
                    disabled
                      ? "default"
                      : "pointer",
                  opacity:
                    disabled
                      ? 0.55
                      : 1,
                  boxShadow:
                    active
                      ? "0 5px 16px rgba(79,123,232,0.12)"
                      : "none",
                }}
              >
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    justifyContent:
                      "space-between",
                    gap: 8,
                  }}
                >
                  <strong
                    style={{
                      fontSize: 12,
                    }}
                  >
                    {option.title}
                  </strong>

                  {recommended && (
                    <span
                      style={{
                        padding:
                          "2px 6px",
                        borderRadius: 999,
                        background:
                          "rgba(79,123,232,0.10)",
                        color:
                          C.primary,
                        fontSize: 8,
                        fontWeight: 700,
                      }}
                    >
                      推荐
                    </span>
                  )}
                </div>

                <div
                  style={{
                    marginTop: 6,
                    color:
                      C.textSecondary,
                    fontSize: 10,
                    lineHeight: 1.55,
                  }}
                >
                  {option.description}
                </div>

                <div
                  style={{
                    marginTop: 8,
                    padding:
                      "7px 8px",
                    borderRadius: 7,
                    background:
                      C.background,
                    color:
                      C.textSecondary,
                    fontSize: 9,
                    lineHeight: 1.55,
                  }}
                >
                  可能先问：
                  {option.example}
                </div>

                {active && (
                  <div
                    style={{
                      marginTop: 8,
                      color:
                        C.primary,
                      fontSize: 9,
                      fontWeight: 700,
                    }}
                  >
                    已选择
                  </div>
                )}
              </button>
            );
          },
        )}
      </div>

      <div
        style={{
          display: "flex",
          justifyContent: "center",
          marginTop: 11,
        }}
      >
        <button
          type="button"
          onClick={() =>
            setShowAll(
              (previous) =>
                !previous,
            )
          }
          disabled={disabled}
          style={{
            padding: "6px 12px",
            borderRadius: 8,
            border:
              `1px solid ${C.border}`,
            background: C.white,
            color:
              C.textSecondary,
            fontSize: 10,
            fontWeight: 700,
            cursor:
              disabled
                ? "default"
                : "pointer",
          }}
        >
          {showAll
            ? "只看推荐方式"
            : "查看其他学习方式"}
        </button>
      </div>

      <div
        style={{
          marginTop: 10,
          color: C.textMuted,
          fontSize: 9,
          lineHeight: 1.6,
          textAlign: "center",
        }}
      >
        更换方式后，请点击“生成学生互动”，系统会重新设计实际运行过程。
      </div>
    </CoursewareAssistantSection>
  );
}

function isPrimaryGrade(
  grade: string,
): boolean {
  const normalized =
    grade.trim();

  return (
    normalized.includes("小学") ||
    /^[一二三四五六1-6]年级$/.test(
      normalized,
    )
  );
}

function localizeModeOption(
  option:
    CoursewareAssistantTeachingModeOption,
  primaryGrade: boolean,
): CoursewareAssistantTeachingModeOption {
  if (
    primaryGrade &&
    option.value ===
      "explain_back"
  ) {
    return {
      ...option,
      title:
        "当小老师讲一遍",
      description:
        "让学生像小老师一样讲出来，再补充没有说清楚的地方。",
    };
  }

  if (
    primaryGrade &&
    option.value ===
      "retrieval_check"
  ) {
    return {
      ...option,
      title:
        "快速回忆一下",
      description:
        "用几个短问题看看哪些内容已经记住。",
    };
  }

  return option;
}

interface RecommendInput {
  subject: string;
  grade: string;
  pageTitle: string;
  pageSummary: string;
  interactionType: string;
}

function recommendTeachingModes({
  subject,
  grade,
  pageTitle,
  pageSummary,
  interactionType,
}: RecommendInput):
  CoursewareAssistantTeachingMode[] {
  const text =
    `${subject} ${grade} ${pageTitle} ${pageSummary} ${interactionType}`
      .toLowerCase();

  const scores =
    new Map<
      CoursewareAssistantTeachingMode,
      number
    >();

  COURSEWARE_ASSISTANT_TEACHING_MODE_OPTIONS.forEach(
    (option, index) => {
      scores.set(
        option.value,
        8 - index * 0.01,
      );
    },
  );

  addScore(
    scores,
    "guided_reasoning",
    3,
  );
  addScore(
    scores,
    "explain_back",
    2,
  );
  addScore(
    scores,
    "coached_practice",
    1,
  );

  if (
    hasAny(
      text,
      [
        "实验",
        "现象",
        "观察",
        "动画",
        "模拟",
        "探究",
        "变化",
        "physics",
        "chemistry",
        "biology",
      ],
    ) ||
    (
      interactionType.trim() &&
      interactionType !== "none"
    )
  ) {
    addScore(
      scores,
      "predict_observe_explain",
      10,
    );
    addScore(
      scores,
      "coached_practice",
      4,
    );
  }

  if (
    hasAny(
      text,
      [
        "例题",
        "示例",
        "范例",
        "步骤",
        "解法",
        "演示",
        "操作流程",
      ],
    )
  ) {
    addScore(
      scores,
      "worked_example",
      11,
    );
    addScore(
      scores,
      "coached_practice",
      5,
    );
  }

  if (
    hasAny(
      text,
      [
        "练习",
        "作答",
        "计算",
        "闯关",
        "测试",
        "选择题",
        "填空",
      ],
    )
  ) {
    addScore(
      scores,
      "coached_practice",
      10,
    );
    addScore(
      scores,
      "retrieval_check",
      5,
    );
  }

  if (
    hasAny(
      text,
      [
        "复习",
        "总结",
        "回顾",
        "检测",
        "巩固",
        "知识清单",
      ],
    )
  ) {
    addScore(
      scores,
      "retrieval_check",
      11,
    );
    addScore(
      scores,
      "explain_back",
      4,
    );
  }

  if (
    hasAny(
      text,
      [
        "比较",
        "区别",
        "异同",
        "对比",
        "分类",
        "共同点",
        "不同点",
      ],
    )
  ) {
    addScore(
      scores,
      "compare_contrast",
      12,
    );
  }

  if (
    hasAny(
      text,
      [
        "观点",
        "证据",
        "材料",
        "论证",
        "评价",
        "支持",
        "反驳",
        "历史",
        "议题",
      ],
    )
  ) {
    addScore(
      scores,
      "evidence_argument",
      10,
    );
  }

  if (
    hasAny(
      text,
      [
        "概念",
        "定义",
        "原理",
        "公式",
        "含义",
        "主旨",
        "为什么",
      ],
    )
  ) {
    addScore(
      scores,
      "explain_back",
      8,
    );
    addScore(
      scores,
      "guided_reasoning",
      5,
    );
  }

  if (
    isPrimaryGrade(grade)
  ) {
    addScore(
      scores,
      "predict_observe_explain",
      2,
    );
    addScore(
      scores,
      "compare_contrast",
      2,
    );
    addScore(
      scores,
      "evidence_argument",
      -3,
    );
  }

  return Array.from(
    scores.entries(),
  )
    .sort(
      (
        first,
        second,
      ) =>
        second[1] - first[1],
    )
    .slice(0, 3)
    .map(
      ([mode]) => mode,
    );
}

function addScore(
  scores: Map<
    CoursewareAssistantTeachingMode,
    number
  >,
  mode:
    CoursewareAssistantTeachingMode,
  value: number,
): void {
  scores.set(
    mode,
    (scores.get(mode) || 0) +
      value,
  );
}

function hasAny(
  text: string,
  keywords: string[],
): boolean {
  return keywords.some(
    (keyword) =>
      text.includes(keyword),
  );
}
