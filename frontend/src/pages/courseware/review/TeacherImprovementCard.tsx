/**
 * TeacherImprovementCard.tsx
 *
 * R-01.1共享教师改进卡。
 *
 * 本组件只负责教师任务的公共展示结构，不负责API、角色权限或状态迁移。
 * 普通AI审核项和跨轮复审项只要提供相同的教师视图字段，就共用本卡。
 *
 * 安全边界：
 *   - 只读取teacher_title、what_happened、teaching_impact、
 *     improvement_goal、acceptance_checks、teacher_context；
 *   - 禁止回退读取title/description/original_suggestion/evidence_json；
 *   - confirmed_instruction只展示已经人工确认的当前修改要求；
 *   - nextStep由各业务容器计算，本组件不推断复审或提交状态。
 */

import type { ReactNode } from "react";

import type {
  CWAIReviewItemRelation,
  CWAIReviewItemStatus,
  CWAIReviewSeverity,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  CW_AI_REVIEW_ITEM_COLORS as C,
  CW_AI_REVIEW_ITEM_SEVERITY,
  type CWAIReviewItemExperience,
  resolveCWAIReviewItemStatus,
} from "./CWAIReviewItemPresentation.shared";

/**
 * 共享教师卡真正需要的数据最小集合。
 *
 * CWAIReviewItem与CWReviewCarryoverItem都应结构化满足该契约，
 * 从而避免复审为了复用UI而伪造AI会话、关系或内部字段。
 */
export interface TeacherImprovementCardItem {
  id: string;
  status: CWAIReviewItemStatus;
  severity: CWAIReviewSeverity;

  teacher_title: string;
  what_happened: string;
  teaching_impact: string;
  improvement_goal: string;
  acceptance_checks: string[];
  teacher_context: string;
  manual_check_required: boolean;

  confirmed_instruction: string;
}

export interface TeacherImprovementCardProps {
  experience: CWAIReviewItemExperience;
  item: TeacherImprovementCardItem;
  activeRelations: CWAIReviewItemRelation[];
  sourceLabel: string;
  pageLabel: string;
  nextStep: string;

  selectable: boolean;
  selected: boolean;
  canSelectForReturn: boolean;
  onSelectedChange?: (itemID: string, selected: boolean) => void;

  actions: ReactNode;
  feedback?: ReactNode;
  details?: ReactNode;
}

function teacherText(
  value: string | null | undefined,
  fallback: string,
): string {
  const normalized = (value || "").trim();
  return normalized || fallback;
}

export default function TeacherImprovementCard({
  experience,
  item,
  activeRelations,
  sourceLabel,
  pageLabel,
  nextStep,
  selectable,
  selected,
  canSelectForReturn,
  onSelectedChange,
  actions,
  feedback,
  details,
}: TeacherImprovementCardProps) {
  const status =
    resolveCWAIReviewItemStatus(
      experience,
      item.status,
    );

  const severity =
    CW_AI_REVIEW_ITEM_SEVERITY[
      item.severity
    ];

  const title = teacherText(
    item.teacher_title,
    "需要教师进一步检查的问题",
  );

  const whatHappened = teacherText(
    item.what_happened,
    "请打开对应页面确认当前课堂呈现。",
  );

  const teachingImpact = teacherText(
    item.teaching_impact,
    "请结合本节课目标判断是否影响讲解、互动或理解。",
  );

  const improvementGoal = teacherText(
    item.improvement_goal,
    "请明确希望页面调整后达到的教学效果。",
  );

  const currentRequirement =
    item.confirmed_instruction.trim();

  const acceptanceChecks =
    (item.acceptance_checks || [])
      .map((value) => value.trim())
      .filter(Boolean)
      .slice(0, 5);

  const teacherContext =
    (item.teacher_context || "").trim();

  const currentRequirementTitle =
    experience === "self"
      ? "当前修改方案"
      : "当前修改要求";

  return (
    <article
      aria-label={`教师改进卡：${title}`}
      style={{
        marginTop: "8px",
        borderRadius: "12px",
        border: `1px solid ${status.color}35`,
        background: C.card,
        overflow: "hidden",
        opacity:
          item.status === "dismissed"
            ? 0.88
            : 1,
      }}
    >
      <div style={{ padding: "14px" }}>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "6px",
            flexWrap: "wrap",
          }}
        >
          <span
            style={badgeStyle(
              severity.color,
              severity.background,
            )}
          >
            {severity.label}
          </span>

          <span
            style={badgeStyle(
              status.color,
              status.background,
            )}
          >
            {status.label}
          </span>

          <span style={neutralBadgeStyle}>
            {pageLabel}
          </span>

          <span style={neutralBadgeStyle}>
            {sourceLabel}
          </span>

          {activeRelations.length > 0 && (
            <span style={relationBadgeStyle}>
              关联 {activeRelations.length}
            </span>
          )}

          {item.manual_check_required && (
            <span style={manualBadgeStyle}>
              需要实际检查
            </span>
          )}
        </div>

        <div
          style={{
            marginTop: "10px",
            color: C.text,
            fontSize: "16px",
            fontWeight: 700,
            lineHeight: 1.55,
          }}
        >
          {title}
        </div>

        <TeacherField
          title="课堂中会看到什么"
          content={whatHappened}
        />

        <TeacherField
          title="为什么值得调整"
          content={teachingImpact}
        />

        {currentRequirement && (
          <TeacherField
            title={currentRequirementTitle}
            content={currentRequirement}
            tone="requirement"
          />
        )}

        {(!currentRequirement ||
          currentRequirement !==
            improvementGoal) && (
          <TeacherField
            title={
              currentRequirement
                ? "建议调整方向"
                : "建议调整到什么效果"
            }
            content={improvementGoal}
            tone="goal"
          />
        )}

        <div style={checksContainerStyle}>
          <div style={fieldTitleStyle}>
            完成后这样检查
          </div>

          {acceptanceChecks.length > 0 ? (
            <ol
              style={{
                margin: "7px 0 0",
                paddingLeft: "22px",
                color: C.textSec,
              }}
            >
              {acceptanceChecks.map(
                (check) => (
                  <li
                    key={check}
                    style={{
                      marginTop: "4px",
                      fontSize: "13px",
                      lineHeight: 1.65,
                    }}
                  >
                    {check}
                  </li>
                ),
              )}
            </ol>
          ) : (
            <div style={emptyCheckStyle}>
              当前没有明确检查项，请先实际查看页面并补充确认标准。
            </div>
          )}
        </div>

        {teacherContext && (
          <TeacherField
            title="教师补充的课堂背景"
            content={teacherContext}
            tone="context"
          />
        )}

        {item.manual_check_required && (
          <div style={manualNoticeStyle}>
            这条问题包含系统无法完全确认的部分。请在真实页面中完成上面的检查，
            再决定是否退回、修改或确认解决。
          </div>
        )}

        {!!nextStep && (
          <div style={nextStepStyle}>
            {nextStep}
          </div>
        )}

        {selectable && (
          <label style={selectionStyle}>
            <input
              type="checkbox"
              checked={
                selected &&
                canSelectForReturn
              }
              disabled={!canSelectForReturn}
              onChange={(event) =>
                onSelectedChange?.(
                  item.id,
                  event.target.checked,
                )
              }
            />

            <span>
              {canSelectForReturn
                ? "本次退回给作者"
                : item.status ===
                    "dismissed"
                  ? "本次不退回给作者"
                  : "先确认修改要求后，才能加入本次退回"}
            </span>
          </label>
        )}

        <div style={actionsStyle}>
          {actions}
        </div>

        {feedback}
      </div>

      {details && (
        <div style={detailsStyle}>
          {details}
        </div>
      )}
    </article>
  );
}

function TeacherField({
  title,
  content,
  tone = "default",
}: {
  title: string;
  content: string;
  tone?:
    | "default"
    | "goal"
    | "context"
    | "requirement";
}) {
  const background =
    tone === "requirement"
      ? "#FFF7ED"
      : tone === "goal"
        ? "#F0FDF4"
        : tone === "context"
          ? "#F8FAFC"
          : "#FFFFFF";

  const border =
    tone === "requirement"
      ? "#FED7AA"
      : tone === "goal"
        ? "#BBF7D0"
        : C.border;

  return (
    <div
      style={{
        marginTop: "10px",
        padding: "10px 12px",
        borderRadius: "9px",
        border: `1px solid ${border}`,
        background,
      }}
    >
      <div style={fieldTitleStyle}>
        {title}
      </div>

      <div
        style={{
          marginTop: "5px",
          color: C.textSec,
          fontSize: "13px",
          lineHeight: 1.65,
        }}
      >
        <DiscussionMarkdown
          content={content}
          compact
        />
      </div>
    </div>
  );
}

function badgeStyle(
  color: string,
  background: string,
) {
  return {
    padding: "3px 7px",
    borderRadius: "999px",
    background,
    color,
    fontSize: "11px",
    fontWeight: 700,
    lineHeight: 1.4,
  } as const;
}

const neutralBadgeStyle = {
  padding: "3px 7px",
  borderRadius: "999px",
  background: "#F8FAFC",
  color: C.textSec,
  fontSize: "11px",
  fontWeight: 600,
  lineHeight: 1.4,
} as const;

const relationBadgeStyle = {
  ...neutralBadgeStyle,
  background: "#F5F3FF",
  color: C.purple,
} as const;

const manualBadgeStyle = {
  ...neutralBadgeStyle,
  background: "#FFF7ED",
  color: C.warning,
} as const;

const fieldTitleStyle = {
  color: C.text,
  fontSize: "12px",
  fontWeight: 700,
  lineHeight: 1.5,
} as const;

const checksContainerStyle = {
  marginTop: "10px",
  padding: "10px 12px",
  borderRadius: "9px",
  border: "1px solid #BFDBFE",
  background: "#EFF6FF",
} as const;

const emptyCheckStyle = {
  marginTop: "6px",
  color: C.textMuted,
  fontSize: "12px",
  lineHeight: 1.6,
} as const;

const manualNoticeStyle = {
  marginTop: "10px",
  padding: "9px 11px",
  borderRadius: "8px",
  border: "1px solid #FED7AA",
  background: "#FFF7ED",
  color: "#9A3412",
  fontSize: "12px",
  lineHeight: 1.65,
} as const;

const nextStepStyle = {
  marginTop: "10px",
  color: C.textSec,
  fontSize: "12px",
  fontWeight: 600,
  lineHeight: 1.6,
} as const;

const selectionStyle = {
  display: "flex",
  alignItems: "flex-start",
  gap: "8px",
  marginTop: "10px",
  padding: "9px 10px",
  borderRadius: "8px",
  border: `1px solid ${C.border}`,
  background: "#F8FAFC",
  color: C.textSec,
  fontSize: "12px",
  lineHeight: 1.55,
  cursor: "pointer",
} as const;

const actionsStyle = {
  display: "flex",
  gap: "6px",
  flexWrap: "wrap",
  marginTop: "10px",
} as const;

const detailsStyle = {
  padding: "0 14px 14px",
  borderTop: `1px solid ${C.border}`,
} as const;
