/**
 * CWReviewWorkbenchSidebar.tsx
 *
 * 课件正式审核工作台右侧信息区。
 *
 * 负责：
 *
 *   - 上轮整改复审；
 *   - AI审核助手；
 *   - 分页批注；
 *   - 历史审核记录。
 *
 * 人工审核决定、退回清单和审核意见位于CWReviewDecisionPanel。
 */

import type {
  CoursewareAnnotation,
  CWReviewCarryoverItem,
  CWReviewListItem,
} from "@/api/coursewares";

import CWAIReviewPanel, {
  type CWAIReviewContext,
} from "./CWAIReviewPanel";
import CWAIReviewPanelBoundary from "./CWAIReviewPanelBoundary";
import CWReviewCarryoverPanel from "./CWReviewCarryoverPanel";
import { CWReviewDecisionGuardPublisher } from "./CWReviewSubmissionGuards";

const C = {
  primary: "#F59E0B",
  danger: "#EF4444",
  success: "#10B981",
  text: "#1F2937",
  textSec: "#6B7280",
  textMuted: "#9CA3AF",
  border: "#F3F4F6",
  card: "#FFFFFF",
};

const DECISION_LABELS:
  Record<
    string,
    {
      label: string;
      color: string;
      icon: string;
    }
  > = {
  approved: {
    label: "通过",
    color: "#10B981",
    icon: "✅",
  },
  revision: {
    label: "退回",
    color: "#F59E0B",
    icon: "↩️",
  },
  revoked: {
    label: "撤回",
    color: "#EF4444",
    icon: "🚫",
  },
};

export type CWReviewSideTab =
  | "carryover"
  | "annotations"
  | "history"
  | "ai";

export interface CWReviewPageReference {
  id: string;
  page_number: number;
}

export interface CWReviewWorkbenchSidebarProps {
  coursewareId: string;
  coursewareTitle: string;
  subject: string;
  grade: string;
  lessonPlanId: string | null;
  reviewLevel: number;

  pages:
    CWReviewPageReference[];

  annotations:
    CoursewareAnnotation[];

  reviews:
    CWReviewListItem[];

  carryoverItems:
    CWReviewCarryoverItem[];

  pendingReviewRound: number;

  resolvedCarryoverItemIds:
    string[];

  activePage: number;

  sideTab:
    CWReviewSideTab;

  onSideTabChange: (
    tab: CWReviewSideTab,
  ) => void;

  onSelectPage: (
    pageNumber: number,
  ) => void;

  onUseReviewComment: (
    comment: string,
  ) => void;

  onReviewContextChange: (
    context: CWAIReviewContext,
  ) => void;

  onCarryoverResolvedChange: (
    itemId: string,
    resolved: boolean,
  ) => void;
}

/**
 * 将批注解析到当前稳定页面。
 *
 * 新记录优先按page_id匹配；旧记录缺少page_id时才按页码兼容。
 * page_id为null表示原页面已删除。
 */
export function resolveCWAnnotationPage(
  annotation:
    CoursewareAnnotation,
  pages:
    CWReviewPageReference[],
): CWReviewPageReference | undefined {
  if (
    annotation.page_id ===
    null
  ) {
    return undefined;
  }

  if (
    annotation.page_id !==
    undefined
  ) {
    return pages.find(
      (page) =>
        page.id ===
        annotation.page_id,
    );
  }

  return pages.find(
    (page) =>
      page.page_number ===
      annotation.page_number,
  );
}

function formatDateTime(
  iso: string,
): string {
  try {
    const date =
      new Date(iso);

    const pad = (
      number: number,
    ) =>
      String(number)
        .padStart(2, "0");

    return [
      `${date.getFullYear()}-${pad(
        date.getMonth() + 1,
      )}-${pad(
        date.getDate(),
      )}`,
      `${pad(
        date.getHours(),
      )}:${pad(
        date.getMinutes(),
      )}`,
    ].join(" ");
  } catch {
    return iso;
  }
}

export default function CWReviewWorkbenchSidebar({
  coursewareId,
  coursewareTitle,
  subject,
  grade,
  lessonPlanId,
  reviewLevel,
  pages,
  annotations,
  reviews,
  carryoverItems,
  pendingReviewRound,
  resolvedCarryoverItemIds,
  activePage,
  sideTab,
  onSideTabChange,
  onSelectPage,
  onUseReviewComment,
  onReviewContextChange,
  onCarryoverResolvedChange,
}: CWReviewWorkbenchSidebarProps) {
  const tabs:
    Array<{
      key: CWReviewSideTab;
      label: string;
    }> = [];

  if (
    carryoverItems.length >
    0
  ) {
    tabs.push({
      key: "carryover",
      label:
        `🔁 上轮整改 ${carryoverItems.length}`,
    });
  }

  tabs.push(
    {
      key: "ai",
      label: "🤖 AI审核",
    },
    {
      key: "annotations",
      label:
        `💬 分页批注 ${annotations.length}`,
    },
    {
      key: "history",
      label:
        `📜 历史 ${reviews.length}`,
    },
  );

  return (
    <div
      style={{
        flex: 1,
        display: "flex",
        flexDirection:
          "column",
        minHeight: 0,
        borderBottom:
          `1px solid ${C.border}`,
      }}
    >
      <CWReviewDecisionGuardPublisher
        items={carryoverItems}
        resolvedItemIds={resolvedCarryoverItemIds}
        onOpenCarryover={() => onSideTabChange("carryover")}
      />

      <div
        style={{
          display: "flex",
          borderBottom:
            `1px solid ${C.border}`,
          padding: "0 6px",
          flexShrink: 0,
        }}
      >
        {tabs.map(
          (tab) => {
            const active =
              sideTab ===
              tab.key;

            return (
              <button
                key={tab.key}
                type="button"
                onClick={() =>
                  onSideTabChange(
                    tab.key,
                  )
                }
                style={{
                  flex: 1,
                  padding:
                    "12px 4px",
                  border: "none",
                  background:
                    "transparent",
                  fontSize:
                    "11px",
                  fontWeight:
                    active
                      ? 700
                      : 400,
                  color:
                    active
                      ? C.primary
                      : C.textSec,
                  cursor:
                    "pointer",
                  borderBottom:
                    active
                      ? `2px solid ${C.primary}`
                      : "2px solid transparent",
                  marginBottom:
                    "-1px",
                  whiteSpace:
                    "nowrap",
                }}
              >
                {tab.label}
              </button>
            );
          },
        )}
      </div>

      <div
        style={{
          flex: 1,
          overflowY: "auto",
          padding: "12px 16px",
          minHeight: 0,
          background:
            sideTab === "ai" ||
            sideTab ===
              "carryover"
              ? "#F8FAFC"
              : C.card,
        }}
      >
        {sideTab ===
          "carryover" && (
          <CWReviewCarryoverPanel
            items={
              carryoverItems
            }
            pages={pages}
            pendingReviewRound={
              pendingReviewRound
            }
            resolvedItemIds={
              resolvedCarryoverItemIds
            }
            onResolvedChange={
              onCarryoverResolvedChange
            }
            onSelectPage={
              onSelectPage
            }
          />
        )}

        {sideTab ===
          "annotations" && (
          <AnnotationList
            annotations={
              annotations
            }
            pages={pages}
            activePage={
              activePage
            }
            onSelectPage={
              onSelectPage
            }
          />
        )}

        {sideTab ===
          "history" && (
          <ReviewHistoryList
            reviews={reviews}
          />
        )}

        {sideTab === "ai" && (
          <CWAIReviewPanelBoundary
            resetKey={`${coursewareId}:${reviewLevel}`}
          >
            <CWAIReviewPanel
              coursewareId={
                coursewareId
              }
              coursewareTitle={
                coursewareTitle
              }
              subject={subject}
              grade={grade}
              lessonPlanId={
                lessonPlanId
              }
              reviewLevel={
                reviewLevel
              }
              onSelectPage={
                onSelectPage
              }
              onUseReviewComment={
                onUseReviewComment
              }
              onReviewContextChange={
                onReviewContextChange
              }
            />
          </CWAIReviewPanelBoundary>
        )}
      </div>
    </div>
  );
}

function AnnotationList({
  annotations,
  pages,
  activePage,
  onSelectPage,
}: {
  annotations:
    CoursewareAnnotation[];
  pages:
    CWReviewPageReference[];
  activePage: number;
  onSelectPage: (
    page: number,
  ) => void;
}) {
  if (
    annotations.length ===
    0
  ) {
    return (
      <EmptyList
        icon="💬"
        message="该课件暂无批注"
      />
    );
  }

  return (
    <>
      {annotations.map(
        (annotation) => {
          const targetPage =
            resolveCWAnnotationPage(
              annotation,
              pages,
            );

          const displayPageNumber =
            targetPage
              ?.page_number ||
            annotation
              .page_number_snapshot ||
            annotation
              .page_number;

          const detached =
            !targetPage;

          const current =
            targetPage
              ?.page_number ===
            activePage;

          const resolved =
            annotation.status ===
            "resolved";

          return (
            <div
              key={
                annotation.id
              }
              onClick={() => {
                if (
                  targetPage
                ) {
                  onSelectPage(
                    targetPage
                      .page_number,
                  );
                }
              }}
              title={
                detached
                  ? "原批注页面已删除，保留历史记录但不能跳转"
                  : `跳转到第${targetPage.page_number}页`
              }
              style={{
                padding:
                  "10px 12px",
                marginBottom:
                  "8px",
                borderRadius:
                  "10px",
                border:
                  current
                    ? `1px solid ${C.primary}40`
                    : `1px solid ${C.border}`,
                background:
                  detached
                    ? "#F9FAFB"
                    : current
                      ? `${C.primary}08`
                      : "#fff",
                cursor:
                  detached
                    ? "default"
                    : "pointer",
                opacity:
                  detached
                    ? 0.78
                    : 1,
              }}
            >
              <div
                style={{
                  display:
                    "flex",
                  alignItems:
                    "center",
                  gap: "6px",
                  marginBottom:
                    "4px",
                  flexWrap:
                    "wrap",
                }}
              >
                <span
                  style={{
                    padding:
                      "1px 7px",
                    borderRadius:
                      "8px",
                    background:
                      detached
                        ? "#E5E7EB"
                        : `${C.primary}15`,
                    color:
                      detached
                        ? C.textSec
                        : C.primary,
                    fontSize:
                      "11px",
                    fontWeight:
                      600,
                  }}
                >
                  {detached
                    ? `原 P${displayPageNumber}`
                    : `P${displayPageNumber}`}
                </span>

                {detached && (
                  <span
                    style={{
                      padding:
                        "1px 7px",
                      borderRadius:
                        "8px",
                      background:
                        "#FEF2F2",
                      color:
                        C.danger,
                      fontSize:
                        "10px",
                      fontWeight:
                        600,
                    }}
                  >
                    页面已删除
                  </span>
                )}

                <span
                  style={{
                    fontSize:
                      "12px",
                    color:
                      C.textSec,
                    fontWeight:
                      500,
                  }}
                >
                  {
                    annotation
                      .reviewer_name ||
                    "匿名"
                  }
                </span>

                {resolved && (
                  <span
                    style={{
                      fontSize:
                        "11px",
                      color:
                        C.success,
                    }}
                  >
                    ✓ 已处理
                  </span>
                )}

                <span
                  style={{
                    marginLeft:
                      "auto",
                    fontSize:
                      "11px",
                    color:
                      C.textMuted,
                  }}
                >
                  {formatDateTime(
                    annotation
                      .created_at,
                  )}
                </span>
              </div>

              <div
                style={{
                  fontSize:
                    "13px",
                  color:
                    resolved
                      ? C.textMuted
                      : C.text,
                  lineHeight:
                    1.5,
                  textDecoration:
                    resolved
                      ? "line-through"
                      : "none",
                  wordBreak:
                    "break-word",
                }}
              >
                {
                  annotation
                    .content
                }
              </div>
            </div>
          );
        },
      )}
    </>
  );
}

function ReviewHistoryList({
  reviews,
}: {
  reviews:
    CWReviewListItem[];
}) {
  if (
    reviews.length ===
    0
  ) {
    return (
      <EmptyList
        icon="📜"
        message="暂无历史审核记录"
      />
    );
  }

  return (
    <>
      {reviews.map(
        (review) => {
          const decision =
            DECISION_LABELS[
              review.decision
            ] ||
            DECISION_LABELS
              .approved;

          return (
            <div
              key={review.id}
              style={{
                padding:
                  "10px 12px",
                marginBottom:
                  "8px",
                borderRadius:
                  "10px",
                border:
                  `1px solid ${C.border}`,
                background:
                  "#fff",
              }}
            >
              <div
                style={{
                  display:
                    "flex",
                  alignItems:
                    "center",
                  gap: "6px",
                  marginBottom:
                    "4px",
                  flexWrap:
                    "wrap",
                }}
              >
                <span
                  style={{
                    padding:
                      "1px 7px",
                    borderRadius:
                      "8px",
                    background:
                      `${
                        review
                          .review_level ===
                        1
                          ? C.primary
                          : C.danger
                      }15`,
                    color:
                      review
                        .review_level ===
                      1
                        ? C.primary
                        : C.danger,
                    fontSize:
                      "11px",
                    fontWeight:
                      600,
                  }}
                >
                  {
                    review
                      .level_name
                  }
                </span>

                <span
                  style={{
                    padding:
                      "1px 7px",
                    borderRadius:
                      "8px",
                    background:
                      `${decision.color}15`,
                    color:
                      decision.color,
                    fontSize:
                      "11px",
                    fontWeight:
                      600,
                  }}
                >
                  {decision.icon}
                  {" "}
                  {decision.label}
                </span>

                <span
                  style={{
                    fontSize:
                      "12px",
                    color:
                      C.textSec,
                  }}
                >
                  {
                    review
                      .reviewer_name
                  }
                </span>

                {review.score !=
                  null && (
                  <span
                    style={{
                      fontSize:
                        "12px",
                      color:
                        C.primary,
                      fontWeight:
                        600,
                    }}
                  >
                    ⭐
                    {" "}
                    {review.score.toFixed(
                      1,
                    )}
                  </span>
                )}

                <span
                  style={{
                    marginLeft:
                      "auto",
                    fontSize:
                      "11px",
                    color:
                      C.textMuted,
                  }}
                >
                  {formatDateTime(
                    review
                      .created_at,
                  )}
                </span>
              </div>

              {review.comment && (
                <div
                  style={{
                    fontSize:
                      "13px",
                    color:
                      C.textSec,
                    lineHeight:
                      1.5,
                    wordBreak:
                      "break-word",
                  }}
                >
                  💬
                  {" "}
                  {review.comment}
                </div>
              )}
            </div>
          );
        },
      )}
    </>
  );
}

function EmptyList({
  icon,
  message,
}: {
  icon: string;
  message: string;
}) {
  return (
    <div
      style={{
        padding:
          "40px 0",
        textAlign:
          "center",
        color:
          C.textMuted,
        fontSize:
          "13px",
      }}
    >
      <div
        style={{
          fontSize:
            "28px",
          marginBottom:
            "8px",
        }}
      >
        {icon}
      </div>

      {message}
    </div>
  );
}
