/**
 * CWReviewHistoryIssueList.tsx
 *
 * R-03本次正式交付问题的纯只读展示。
 *
 * 复用R-01.1 TeacherImprovementCard，但：
 *   - 固定为正式交付瞬间confirmed展示状态；
 *   - 不提供选择、退回、确认解决、AI讨论或编辑动作；
 *   - 修改要求只取delivered_instruction；
 *   - 页面身份使用历史page_id和page_number；
 *   - 原页面删除时明确展示“原页面已删除”。
 */

import {
  getCWAIReviewDimensionLabel,
} from "@/api/coursewares.ai-review-config";

import type {
  CWReviewHistoryIssue,
  CWReviewHistoryPage,
} from "@/api/coursewares";

import TeacherImprovementCard from "./TeacherImprovementCard";

const C = {
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  warning: "#D97706",
};

function formatDateTime(
  value: string | null,
): string {
  if (!value) {
    return "—";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString("zh-CN", {
    hour12: false,
  });
}

function resolvePageLabel(
  issue: CWReviewHistoryIssue,
  historicalPages: CWReviewHistoryPage[],
): string {
  if (issue.page_number <= 0) {
    return "整课";
  }

  const pageID =
    issue.page_id?.trim() || "";

  const historicalPage =
    pageID
      ? historicalPages.find(
          (page) => page.page_id === pageID,
        )
      : undefined;

  if (
    historicalPage &&
    !historicalPage.current_exists
  ) {
    return `原P${issue.page_number} · 原页面已删除`;
  }

  return `审核时P${issue.page_number}`;
}

function IssueDetails({
  issue,
}: {
  issue: CWReviewHistoryIssue;
}) {
  const records =
    issue.previous_modification_records || [];

  return (
    <div
      style={{
        paddingTop: "12px",
        display: "grid",
        gap: "10px",
      }}
    >
      {issue.delivered_instruction_available &&
      issue.delivered_instruction ? (
        <div
          style={{
            color: C.textSec,
            fontSize: "12px",
            lineHeight: 1.65,
          }}
        >
          正式交付版本：
          V{issue.delivered_instruction.version_no}
          {" · "}
          确认于{" "}
          {formatDateTime(
            issue.delivered_instruction.confirmed_at,
          )}
        </div>
      ) : (
        <div
          style={{
            padding: "9px 10px",
            borderRadius: "8px",
            border: "1px solid #FED7AA",
            background: "#FFF7ED",
            color: C.warning,
            fontSize: "12px",
            lineHeight: 1.65,
          }}
        >
          该旧问题没有可证明的正式交付指令版本，
          历史详情不会使用当前版本进行补写。
        </div>
      )}

      {records.length > 0 && (
        <div>
          <div
            style={{
              marginBottom: "6px",
              color: C.text,
              fontSize: "12px",
              fontWeight: 700,
            }}
          >
            后续执行补充
          </div>

          <div
            style={{
              display: "grid",
              gap: "6px",
            }}
          >
            {records.map(
              (record, index) => (
                <div
                  key={`${record.created_at || "record"}-${index}`}
                  style={{
                    padding: "9px 10px",
                    borderRadius: "8px",
                    border: `1px solid ${C.border}`,
                    color: C.textSec,
                    fontSize: "12px",
                    lineHeight: 1.65,
                    whiteSpace: "pre-wrap",
                  }}
                >
                  <div>
                    {record.content}
                  </div>

                  <div
                    style={{
                      marginTop: "4px",
                      color: C.textMuted,
                      fontSize: "11px",
                    }}
                  >
                    {formatDateTime(
                      record.created_at,
                    )}
                  </div>
                </div>
              ),
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export default function CWReviewHistoryIssueList({
  issues,
  historicalPages,
}: {
  issues: CWReviewHistoryIssue[];
  historicalPages: CWReviewHistoryPage[];
}) {
  if (issues.length === 0) {
    return (
      <div
        style={{
          padding: "24px",
          textAlign: "center",
          color: C.textMuted,
          fontSize: "13px",
        }}
      >
        本次审核没有正式交付整改问题。
      </div>
    );
  }

  return (
    <div
      style={{
        display: "grid",
        gap: "12px",
      }}
    >
      {issues.map((issue) => {
        const deliveredInstruction =
          issue.delivered_instruction_available &&
          issue.delivered_instruction
            ? issue.delivered_instruction.content
            : "";

        return (
          <TeacherImprovementCard
            key={issue.id}
            experience="review"
            item={{
              id: issue.id,
              status: "confirmed",
              severity: issue.severity,
              teacher_title:
                issue.teacher_view.teacher_title,
              what_happened:
                issue.teacher_view.what_happened,
              teaching_impact:
                issue.teacher_view.teaching_impact,
              improvement_goal:
                issue.teacher_view.improvement_goal,
              acceptance_checks:
                issue.teacher_view.acceptance_checks,
              teacher_context:
                issue.teacher_view.teacher_context,
              manual_check_required:
                issue.teacher_view.manual_check_required,
              confirmed_instruction:
                deliveredInstruction,
            }}
            activeRelations={[]}
            sourceLabel={
              `本次正式交付 · ${
                getCWAIReviewDimensionLabel(
                  issue.dimension,
                )
              }`
            }
            pageLabel={resolvePageLabel(
              issue,
              historicalPages,
            )}
            nextStep=""
            selectable={false}
            selected={false}
            canSelectForReturn={false}
            actions={null}
            details={
              <IssueDetails issue={issue} />
            }
          />
        );
      })}
    </div>
  );
}
