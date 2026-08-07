/**
 * 教学智能体可参考内容编辑器。
 *
 * 普通使用时系统会自动选择安全默认范围，教师不需要配置。
 * 高级设置只允许开启或关闭已经定义的课件来源，
 * 不允许填写任意网址、数据库查询、工具或知识库编号。
 */

import type { CoursewareAssistantContextConfig } from "@/api/coursewares";

import { COURSEWARE_ASSISTANT_LIMITS } from "./coursewareAssistantDraft";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
  CoursewareAssistantSection,
  coursewareAssistantInputStyle,
  coursewareAssistantLabelStyle,
} from "./CoursewareAssistantEditorShared";

interface Props {
  context: CoursewareAssistantContextConfig;
  onChange: (context: CoursewareAssistantContextConfig) => void;
  disabled?: boolean;
}

export default function CoursewareAssistantContextEditor({
  context,
  onChange,
  disabled = false,
}: Props) {
  const C = COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <CoursewareAssistantSection
      title="AI可以参考哪些内容"
      description="系统默认使用当前页面和相关教学资料。需要减少参考范围时再调整。"
    >
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(230px, 1fr))",
          gap: 8,
        }}
      >
        <ContextToggle
          label="当前页面中的文字"
          description="学生在当前页面能够看到的静态文字"
          checked={context.include_visible_text}
          disabled={disabled}
          onChange={(checked) =>
            onChange({
              ...context,
              include_visible_text: checked,
            })
          }
        />

        <ContextToggle
          label="当前页面的教学说明"
          description="本页的学习目的、内容概要和教学设计"
          checked={context.include_page_plan}
          disabled={disabled}
          onChange={(checked) =>
            onChange({
              ...context,
              include_page_plan: checked,
            })
          }
        />

        <ContextToggle
          label="页面互动线索"
          description="页面中声明的互动入口、操作目标和风险提示"
          checked={context.include_interaction_evidence}
          disabled={disabled}
          onChange={(checked) =>
            onChange({
              ...context,
              include_interaction_evidence: checked,
            })
          }
        />

        <ContextToggle
          label="来源教案的相关片段"
          description="只读取与当前页面有关的受限内容"
          checked={context.include_lesson_plan_excerpt}
          disabled={disabled}
          onChange={(checked) =>
            onChange({
              ...context,
              include_lesson_plan_excerpt: checked,
              max_lesson_plan_excerpt_chars: checked
                ? context.max_lesson_plan_excerpt_chars || 4000
                : 0,
            })
          }
        />

        <ContextToggle
          label="前一页的简要内容"
          description="只用于衔接，不会当作当前页面已经讲过的事实"
          checked={context.include_previous_page_summary}
          disabled={disabled}
          onChange={(checked) =>
            onChange({
              ...context,
              include_previous_page_summary: checked,
            })
          }
        />

        <ContextToggle
          label="后一页的简要内容"
          description="只用于衔接，不会提前展示后一页完整内容"
          checked={context.include_next_page_summary}
          disabled={disabled}
          onChange={(checked) =>
            onChange({
              ...context,
              include_next_page_summary: checked,
            })
          }
        />
      </div>

      {!context.include_visible_text && !context.include_page_plan && (
        <div
          style={{
            marginTop: 10,
            padding: "8px 10px",
            borderRadius: 8,
            border: "1px solid #FDE68A",
            background: "#FFFBEB",
            color: "#92400E",
            fontSize: 10,
            lineHeight: 1.6,
          }}
        >
          当前页面中的文字和当前页面的教学说明不能同时关闭。
        </div>
      )}

      {context.include_lesson_plan_excerpt && (
        <label style={{ display: "block", marginTop: 12, maxWidth: 300 }}>
          <span style={coursewareAssistantLabelStyle}>
            最多读取多少教案字符
          </span>

          <input
            type="number"
            min={COURSEWARE_ASSISTANT_LIMITS.minimumLessonExcerpt}
            max={COURSEWARE_ASSISTANT_LIMITS.maximumLessonExcerpt}
            step={500}
            value={context.max_lesson_plan_excerpt_chars}
            disabled={disabled}
            onChange={(event) =>
              onChange({
                ...context,
                max_lesson_plan_excerpt_chars: Number(event.target.value),
              })
            }
            style={coursewareAssistantInputStyle}
          />

          <div style={{ marginTop: 4, color: C.textMuted, fontSize: 9 }}>
            可设置为 {COURSEWARE_ASSISTANT_LIMITS.minimumLessonExcerpt}–
            {COURSEWARE_ASSISTANT_LIMITS.maximumLessonExcerpt} 个字符
          </div>
        </label>
      )}
    </CoursewareAssistantSection>
  );
}

function ContextToggle({
  label,
  description,
  checked,
  disabled,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  disabled: boolean;
  onChange: (checked: boolean) => void;
}) {
  const C = COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <label
      style={{
        display: "flex",
        alignItems: "flex-start",
        gap: 7,
        padding: 10,
        borderRadius: 9,
        border: `1px solid ${checked ? C.primary : C.border}`,
        background: checked ? C.primaryBackground : C.white,
        cursor: disabled ? "default" : "pointer",
      }}
    >
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
      />

      <span>
        <strong style={{ display: "block", color: C.text, fontSize: 11 }}>
          {label}
        </strong>

        <small
          style={{
            display: "block",
            marginTop: 2,
            color: C.textSecondary,
            fontSize: 9,
            lineHeight: 1.5,
          }}
        >
          {description}
        </small>
      </span>
    </label>
  );
}
