/**
 * CWReviewHistorySummary.tsx
 *
 * R-03已审核记录只读摘要。
 *
 * 只展示后端按review_id返回的历史事实，不读取当前课件审核状态，
 * 不提供配置编辑、审核提交或其它写入口。
 */

import {
  getCWAIReviewDimensionLabel,
  getCWAIReviewLessonReferenceLabel,
} from "@/api/coursewares.ai-review-config";

import type {
  CWReviewHistoryDetail,
} from "@/api/coursewares";

const C = {
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  card: "#FFFFFF",
  background: "#F8FAFC",
  primary: "#2563EB",
  success: "#059669",
  warning: "#D97706",
};

const CONFIG_UNAVAILABLE_LABELS:
  Record<string, string> = {
    review_without_ai_session:
      "本次人工审核没有关联AI审核会话。",
    legacy_review_without_immutable_config:
      "该旧审核没有可证明的不可变审核配置。",
    review_ai_session_unavailable:
      "本次审核关联的历史AI会话已经无法读取。",
  };

const DECISION_LABELS:
  Record<string, string> = {
    approved: "通过",
    revision: "退回修改",
    revoked: "撤回",
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

function Fact({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div
      style={{
        minWidth: 0,
        padding: "10px 12px",
        borderRadius: "9px",
        border: `1px solid ${C.border}`,
        background: C.card,
      }}
    >
      <div
        style={{
          color: C.textMuted,
          fontSize: "11px",
          marginBottom: "4px",
        }}
      >
        {label}
      </div>

      <div
        style={{
          color: C.text,
          fontSize: "13px",
          fontWeight: 600,
          lineHeight: 1.55,
          wordBreak: "break-word",
        }}
      >
        {value || "—"}
      </div>
    </div>
  );
}

export default function CWReviewHistorySummary({
  detail,
}: {
  detail: CWReviewHistoryDetail;
}) {
  const decision =
    DECISION_LABELS[detail.review.decision] ||
    detail.review.decision ||
    "—";

  const score =
    detail.review.score == null
      ? "—"
      : detail.review.score.toFixed(1);

  const config = detail.review_config;

  return (
    <section
      style={{
        display: "grid",
        gap: "14px",
      }}
    >
      <div
        style={{
          padding: "16px",
          borderRadius: "12px",
          border: `1px solid ${C.border}`,
          background: C.background,
        }}
      >
        <div
          style={{
            marginBottom: "12px",
            color: C.text,
            fontSize: "15px",
            fontWeight: 700,
          }}
        >
          审核记录
        </div>

        <div
          style={{
            display: "grid",
            gridTemplateColumns:
              "repeat(auto-fit, minmax(180px, 1fr))",
            gap: "8px",
          }}
        >
          <Fact
            label="课件名称"
            value={detail.courseware.title}
          />
          <Fact
            label="学科"
            value={detail.courseware.subject}
          />
          <Fact
            label="年级"
            value={detail.courseware.grade}
          />
          <Fact
            label="审核级别"
            value={`L${detail.review.review_level}`}
          />
          <Fact
            label="审核轮次"
            value={`第${detail.review.review_round}轮`}
          />
          <Fact
            label="审核时间"
            value={formatDateTime(
              detail.review.reviewed_at,
            )}
          />
          <Fact
            label="审核教师"
            value={detail.reviewer.display_name}
          />
          <Fact
            label="审核决定"
            value={decision}
          />
          <Fact
            label="审核评分"
            value={score}
          />
        </div>

        <div
          style={{
            marginTop: "10px",
            padding: "12px",
            borderRadius: "9px",
            border: `1px solid ${C.border}`,
            background: C.card,
          }}
        >
          <div
            style={{
              color: C.textMuted,
              fontSize: "11px",
              marginBottom: "5px",
            }}
          >
            审核意见
          </div>

          <div
            style={{
              color: C.text,
              fontSize: "13px",
              lineHeight: 1.7,
              whiteSpace: "pre-wrap",
            }}
          >
            {detail.review.comment || "本次审核未填写意见。"}
          </div>
        </div>
      </div>

      <div
        style={{
          padding: "16px",
          borderRadius: "12px",
          border: `1px solid ${C.border}`,
          background: C.background,
        }}
      >
        <div
          style={{
            marginBottom: "5px",
            color: C.text,
            fontSize: "15px",
            fontWeight: 700,
          }}
        >
          当时审核配置
        </div>

        <div
          style={{
            marginBottom: "12px",
            color: C.textSec,
            fontSize: "12px",
            lineHeight: 1.6,
          }}
        >
          以下内容来自本次审核的不可变配置事实，不读取当前审核配置。
        </div>

        {!config.available ? (
          <div
            style={{
              padding: "11px 12px",
              borderRadius: "9px",
              border: "1px solid #FED7AA",
              background: "#FFF7ED",
              color: C.warning,
              fontSize: "12px",
              lineHeight: 1.65,
            }}
          >
            {CONFIG_UNAVAILABLE_LABELS[
              config.unavailable_reason
            ] ||
              "该历史审核没有足够事实还原当时审核配置。"}
          </div>
        ) : (
          <>
            <div
              style={{
                display: "grid",
                gridTemplateColumns:
                  "repeat(auto-fit, minmax(180px, 1fr))",
                gap: "8px",
              }}
            >
              <Fact
                label="配置协议"
                value={`v${config.schema_version}`}
              />

              <Fact
                label="教案参考模式"
                value={
                  getCWAIReviewLessonReferenceLabel(
                    config.lesson_reference_mode,
                  ) ||
                  config.lesson_reference_mode
                }
              />

              <Fact
                label="本次是否实际使用教案类材料"
                value={
                  config.lesson_materials_used == null
                    ? "历史事实不足，无法证明"
                    : config.lesson_materials_used
                      ? "是"
                      : "否"
                }
              />
            </div>

            <div
              style={{
                marginTop: "10px",
                padding: "11px 12px",
                borderRadius: "9px",
                border: `1px solid ${C.border}`,
                background: C.card,
              }}
            >
              <div
                style={{
                  color: C.textMuted,
                  fontSize: "11px",
                  marginBottom: "7px",
                }}
              >
                审核维度
              </div>

              <div
                style={{
                  display: "flex",
                  flexWrap: "wrap",
                  gap: "6px",
                }}
              >
                {config.dimensions.length > 0 ? (
                  config.dimensions.map(
                    (dimension) => (
                      <span
                        key={dimension}
                        style={{
                          padding: "4px 8px",
                          borderRadius: "999px",
                          border:
                            "1px solid #BFDBFE",
                          background: "#EFF6FF",
                          color: C.primary,
                          fontSize: "11px",
                          fontWeight: 700,
                        }}
                      >
                        {getCWAIReviewDimensionLabel(
                          dimension,
                        )}
                      </span>
                    ),
                  )
                ) : (
                  <span
                    style={{
                      color: C.textMuted,
                      fontSize: "12px",
                    }}
                  >
                    未记录审核维度
                  </span>
                )}
              </div>
            </div>

            {config.custom_focus.trim() && (
              <div
                style={{
                  marginTop: "10px",
                  padding: "11px 12px",
                  borderRadius: "9px",
                  border: `1px solid ${C.border}`,
                  background: C.card,
                }}
              >
                <div
                  style={{
                    color: C.textMuted,
                    fontSize: "11px",
                    marginBottom: "5px",
                  }}
                >
                  自定义关注要求
                </div>

                <div
                  style={{
                    color: C.text,
                    fontSize: "13px",
                    lineHeight: 1.7,
                    whiteSpace: "pre-wrap",
                  }}
                >
                  {config.custom_focus}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </section>
  );
}
