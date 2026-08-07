/**
 * CWOwnerReviewRemediationPanel.tsx
 *
 * 课件作者整改中心。
 *
 * 展示：
 *   - 正式审核提交时生成的不可变整体反馈；
 *   - 整课明显问题和优点；
 *   - 正式审核交付的页级整改项；
 *   - 作者自己的AI自审整改项。
 *
 * 操作：
 *   - 每条整改项可继续独立讨论和确认；
 *   - 已确认的页级指令可以注入页面微调草稿；
 *   - 注入只填充草稿，不自动执行页面修改。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import {
  getCWOwnerReviewRemediation,
  parseCWAIReviewJSON,
  type CWAIReviewFeedback,
  type CWAIReviewItem,
  type CWAIReviewSeverity,
} from "@/api/coursewares";

import {
  CWOwnerReviewSubmissionReadiness,
} from "@/pages/courseware/review/CWReviewSubmissionGuards";

import CWOwnerReviewItemsByPage from "./CWOwnerReviewItemsByPage";

const C = {
  primary: "#4F7BE8",
  success: "#059669",
  danger: "#DC2626",
  warning: "#D97706",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  card: "#FFFFFF",
  bg: "#F8FAFC",
};

const SEVERITY_CONFIG: Record<
  CWAIReviewSeverity,
  {
    label: string;
    color: string;
    bg: string;
  }
> = {
  critical: {
    label: "严重",
    color: "#B91C1C",
    bg: "#FEE2E2",
  },
  high: {
    label: "高风险",
    color: "#DC2626",
    bg: "#FEF2F2",
  },
  medium: {
    label: "中风险",
    color: "#D97706",
    bg: "#FEF3C7",
  },
  low: {
    label: "低风险",
    color: "#2563EB",
    bg: "#DBEAFE",
  },
  info: {
    label: "提示",
    color: "#64748B",
    bg: "#F1F5F9",
  },
};

export interface CWOwnerReviewRemediationPanelProps {
  coursewareId: string;

  onSelectPage: (
    pageNumber: number,
  ) => void;

  onInjectToRefine: (
    item: CWAIReviewItem,
  ) => void;
}

function formatDateTime(
  raw: string | null,
): string {
  if (!raw) {
    return "";
  }

  try {
    return new Date(
      raw,
    ).toLocaleString("zh-CN");
  } catch {
    return raw;
  }
}

/**
 * 将反馈JSON数组转换为可显示文本。
 *
 * 历史数据可能是字符串数组，也可能是结构化对象数组，
 * 因此必须防御性处理。
 */
function parseFeedbackList(
  raw: string,
): string[] {
  const values =
    parseCWAIReviewJSON<unknown[]>(
      raw,
      [],
    );

  return values
    .map((value) => {
      if (
        typeof value === "string"
      ) {
        return value.trim();
      }

      if (
        value &&
        typeof value === "object"
      ) {
        const candidate =
          value as Record<
            string,
            unknown
          >;

        for (const key of [
          "title",
          "description",
          "summary",
          "problem",
          "content",
        ]) {
          if (
            typeof candidate[key] ===
            "string"
          ) {
            return String(
              candidate[key],
            ).trim();
          }
        }

        try {
          return JSON.stringify(
            value,
          );
        } catch {
          return "";
        }
      }

      return "";
    })
    .filter(Boolean);
}

function feedbackLevelLabel(
  level: number,
): string {
  if (level === 1) {
    return "L1 教研组审核";
  }

  if (level === 2) {
    return "L2 学校审核";
  }

  return `L${level} 审核`;
}

function decisionLabel(
  decision: string,
): string {
  return decision === "revision"
    ? "↩️ 退回修改"
    : "✅ 审核通过";
}

export default function CWOwnerReviewRemediationPanel({
  coursewareId,
  onSelectPage,
  onInjectToRefine,
}: CWOwnerReviewRemediationPanelProps) {
  const [
    feedbacks,
    setFeedbacks,
  ] = useState<CWAIReviewFeedback[]>(
    [],
  );

  const [
    items,
    setItems,
  ] = useState<CWAIReviewItem[]>(
    [],
  );

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    error,
    setError,
  ] = useState("");

  const loadRemediation =
    useCallback(async () => {
      setLoading(true);
      setError("");

      try {
        const result =
          await getCWOwnerReviewRemediation(
            coursewareId,
          );

        setFeedbacks(
          result.feedbacks || [],
        );
        setItems(
          result.items || [],
        );
      } catch (cause) {
        setError(
          cause instanceof Error
            ? cause.message
            : "加载课件审核整改信息失败",
        );
      } finally {
        setLoading(false);
      }
    }, [
      coursewareId,
    ]);

  useEffect(() => {
    void loadRemediation();
  }, [
    loadRemediation,
  ]);

  const formalItemsByFeedback =
    useMemo(() => {
      const result = new Map<
        string,
        CWAIReviewItem[]
      >();

      for (const item of items) {
        if (
          item.source_type !==
            "formal" ||
          !item.feedback_id
        ) {
          continue;
        }

        const current =
          result.get(
            item.feedback_id,
          ) || [];

        current.push(item);

        result.set(
          item.feedback_id,
          current,
        );
      }

      return result;
    }, [
      items,
    ]);

  const selfReviewItems =
    useMemo(
      () =>
        items.filter(
          (item) =>
            item.source_type ===
            "self",
        ),
      [items],
    );

  const unmatchedFormalItems =
    useMemo(
      () =>
        items.filter(
          (item) =>
            item.source_type ===
              "formal" &&
            (
              !item.feedback_id ||
              !feedbacks.some(
                (feedback) =>
                  feedback.id ===
                  item.feedback_id,
              )
            ),
        ),
      [
        items,
        feedbacks,
      ],
    );

  const handleItemChanged =
    useCallback(
      (
        changed:
          CWAIReviewItem,
      ) => {
        setItems((previous) =>
          previous.map((item) =>
            item.id === changed.id
              ? changed
              : item,
          ),
        );
      },
      [],
    );

  if (loading) {
    return (
      <div
        style={{
          marginTop: 16,
          padding: "30px 16px",
          borderRadius: 12,
          border: `1px solid ${C.border}`,
          background: C.bg,
          textAlign: "center",
          color: C.textMuted,
          fontSize: 13,
        }}
      >
        正在读取审核反馈和整改项…
      </div>
    );
  }

  return (
    <section
      style={{
        marginTop: 16,
        padding: 16,
        borderRadius: 12,
        border: `1px solid ${C.border}`,
        background: C.bg,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: 12,
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
              fontSize: 14,
              fontWeight: 700,
            }}
          >
            📋 审核与自审整改中心
          </div>

          <div
            style={{
              marginTop: 4,
              color: C.textSec,
              fontSize: 11,
              lineHeight: 1.65,
            }}
          >
            正式审核意见是提交时的不可变快照。页级整改项可以继续讨论；确认后的指令只能手动注入微调草稿，不会自动运行AI。
          </div>
        </div>

        <button
          type="button"
          onClick={() => {
            void loadRemediation();
          }}
          style={{
            flexShrink: 0,
            padding: "6px 11px",
            borderRadius: 7,
            border: `1px solid ${C.border}`,
            background: "#fff",
            color: C.primary,
            fontSize: 11,
            fontWeight: 600,
            cursor: "pointer",
          }}
        >
          🔄 刷新
        </button>
      </div>

      {error && (
        <div
          style={{
            marginBottom: 10,
            padding: "9px 11px",
            borderRadius: 8,
            background: "#FEF2F2",
            border:
              "1px solid #FECACA",
            color: C.danger,
            fontSize: 11,
            lineHeight: 1.6,
          }}
        >
          ⚠️ {error}
        </div>
      )}

      <CWOwnerReviewSubmissionReadiness
        items={items}
        onSelectPage={
          onSelectPage
        }
      />

      {feedbacks.length === 0 &&
      items.length === 0 ? (
        <div
          style={{
            padding: "32px 12px",
            textAlign: "center",
            borderRadius: 10,
            border: `1px dashed ${C.border}`,
            background: "#fff",
            color: C.textMuted,
            fontSize: 12,
            lineHeight: 1.7,
          }}
        >
          暂无正式审核反馈或已保存的自审整改项。
        </div>
      ) : (
        <>
          {feedbacks.map(
            (feedback) => {
              const risk =
                SEVERITY_CONFIG[
                  feedback.overall_risk as
                    CWAIReviewSeverity
                ] ||
                SEVERITY_CONFIG.info;

              const strengths =
                parseFeedbackList(
                  feedback.strengths_json,
                );

              const problems =
                parseFeedbackList(
                  feedback.obvious_problems_json,
                );

              const feedbackItems =
                formalItemsByFeedback.get(
                  feedback.id,
                ) || [];

              return (
                <div
                  key={feedback.id}
                  style={{
                    marginBottom: 14,
                    padding: 13,
                    borderRadius: 10,
                    border: `1px solid ${risk.color}35`,
                    background: C.card,
                  }}
                >
                  <div
                    style={{
                      display: "flex",
                      alignItems:
                        "center",
                      gap: 7,
                      flexWrap: "wrap",
                    }}
                  >
                    <span
                      style={{
                        padding:
                          "2px 7px",
                        borderRadius: 6,
                        background:
                          risk.bg,
                        color:
                          risk.color,
                        fontSize: 10,
                        fontWeight: 700,
                      }}
                    >
                      {risk.label}
                    </span>

                    <span
                      style={{
                        color: C.text,
                        fontSize: 12,
                        fontWeight: 700,
                      }}
                    >
                      {feedbackLevelLabel(
                        feedback.review_level,
                      )}
                      · 第
                      {
                        feedback.review_round
                      }
                      轮
                    </span>

                    <span
                      style={{
                        color:
                          feedback.decision ===
                          "revision"
                            ? C.warning
                            : C.success,
                        fontSize: 11,
                        fontWeight: 700,
                      }}
                    >
                      {decisionLabel(
                        feedback.decision,
                      )}
                    </span>

                    <span
                      style={{
                        marginLeft:
                          "auto",
                        color:
                          C.textMuted,
                        fontSize: 10,
                      }}
                    >
                      {formatDateTime(
                        feedback.created_at,
                      )}
                    </span>
                  </div>

                  {feedback.overall_summary && (
                    <div
                      style={{
                        marginTop: 9,
                        padding: "9px 10px",
                        borderRadius: 8,
                        background:
                          risk.bg,
                        color: C.text,
                        fontSize: 11,
                        lineHeight: 1.7,
                        whiteSpace:
                          "pre-wrap",
                      }}
                    >
                      <strong>
                        整课评价：
                      </strong>
                      {
                        feedback.overall_summary
                      }
                    </div>
                  )}

                  {problems.length > 0 && (
                    <FeedbackList
                      title="明显和高层问题"
                      values={problems}
                      color={C.danger}
                    />
                  )}

                  {strengths.length > 0 && (
                    <FeedbackList
                      title="明确优点"
                      values={strengths}
                      color={C.success}
                    />
                  )}

                  {feedback.review_comment_snapshot && (
                    <div
                      style={{
                        marginTop: 9,
                        padding: "9px 10px",
                        borderRadius: 8,
                        border: `1px solid ${C.border}`,
                        color: C.textSec,
                        background:
                          "#FAFAFA",
                        fontSize: 11,
                        lineHeight: 1.7,
                        whiteSpace:
                          "pre-wrap",
                      }}
                    >
                      <strong
                        style={{
                          color: C.text,
                        }}
                      >
                        人工审核意见：
                      </strong>
                      {
                        feedback.review_comment_snapshot
                      }
                    </div>
                  )}

                  <div
                    style={{
                      marginTop: 11,
                      paddingTop: 10,
                      borderTop: `1px solid ${C.border}`,
                    }}
                  >
                    <div
                      style={{
                        color: C.text,
                        fontSize: 11,
                        fontWeight: 700,
                      }}
                    >
                      本轮交付整改清单{" "}
                      {
                        feedbackItems.length
                      }
                    </div>

                    <CWOwnerReviewItemsByPage
                      items={
                        feedbackItems
                      }
                      emptyMessage="本轮未交付页级整改项。"
                      onSelectPage={
                        onSelectPage
                      }
                      onChanged={
                        handleItemChanged
                      }
                      onInjectToRefine={
                        onInjectToRefine
                      }
                    />
                  </div>
                </div>
              );
            },
          )}

          {selfReviewItems.length >
            0 && (
            <div
              style={{
                marginBottom: 14,
                padding: 13,
                borderRadius: 10,
                border: `1px solid ${C.primary}35`,
                background: C.card,
              }}
            >
              <div
                style={{
                  color: C.text,
                  fontSize: 12,
                  fontWeight: 700,
                }}
              >
                🛡️ 作者AI自审整改项{" "}
                {selfReviewItems.length}
              </div>

              <div
                style={{
                  marginTop: 4,
                  color: C.textMuted,
                  fontSize: 10,
                  lineHeight: 1.6,
                }}
              >
                自审项不会进入正式审核记录，但可以继续讨论、确认并注入页面微调。
              </div>

              <CWOwnerReviewItemsByPage
                items={
                  selfReviewItems
                }
                emptyMessage="暂无作者AI自审整改项。"
                onSelectPage={
                  onSelectPage
                }
                onChanged={
                  handleItemChanged
                }
                onInjectToRefine={
                  onInjectToRefine
                }
              />
            </div>
          )}

          {unmatchedFormalItems.length >
            0 && (
            <div
              style={{
                padding: 13,
                borderRadius: 10,
                border: `1px solid ${C.border}`,
                background: C.card,
              }}
            >
              <div
                style={{
                  color: C.text,
                  fontSize: 12,
                  fontWeight: 700,
                }}
              >
                其他正式整改项{" "}
                {
                  unmatchedFormalItems.length
                }
              </div>

              <CWOwnerReviewItemsByPage
                items={
                  unmatchedFormalItems
                }
                emptyMessage="暂无其他正式整改项。"
                onSelectPage={
                  onSelectPage
                }
                onChanged={
                  handleItemChanged
                }
                onInjectToRefine={
                  onInjectToRefine
                }
              />
            </div>
          )}
        </>
      )}
    </section>
  );
}

function FeedbackList({
  title,
  values,
  color,
}: {
  title: string;
  values: string[];
  color: string;
}) {
  return (
    <div
      style={{
        marginTop: 9,
      }}
    >
      <div
        style={{
          marginBottom: 4,
          color,
          fontSize: 11,
          fontWeight: 700,
        }}
      >
        {title}
      </div>

      {values.map(
        (value, index) => (
          <div
            key={`${index}-${value}`}
            style={{
              color: C.textSec,
              fontSize: 10,
              lineHeight: 1.65,
              wordBreak:
                "break-word",
            }}
          >
            • {value}
          </div>
        ),
      )}
    </div>
  );
}
