/**
 * CWAIReviewFindingList.tsx
 *
 * 课件AI审核最终报告中的未物化finding列表与页码按钮。
 *
 * 从CWAIReviewReportView拆分，避免主报告组件超过900行。
 * 本文件只负责展示和触发父级回调，不创建整改项、不修改页面。
 */

import type {
  CWAIReviewFinding,
  CWAIReviewItem,
  CWAIReviewSeverity,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

const C = {
  primary: "#4F7BE8",
  text: "#1F2937",
  textSec: "#6B7280",
  textMuted: "#9CA3AF",
  border: "#E5E7EB",
  card: "#FFFFFF",
};

const SEVERITY: Record<
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
    color: "#6B7280",
    bg: "#F3F4F6",
  },
};

export function severityConfig(
  severity: CWAIReviewSeverity,
) {
  return SEVERITY[severity] ||
    SEVERITY.medium;
}

export function findingMatchesItem(
  findingId: string,
  item: CWAIReviewItem,
): boolean {
  return (
    item.source_finding_id === findingId ||
    item.source_finding_id.startsWith(
      `${findingId}@p`,
    )
  );
}

export interface CWAIReviewFindingListProps {
  title: string;
  findings: CWAIReviewFinding[];
  items: CWAIReviewItem[];
  canAdopt: boolean;
  materializingFindingIds: string[];

  onAdoptFinding: (
    findingId: string,
  ) => void;

  onSelectPage: (
    pageNumber: number,
  ) => void;
}

export default function CWAIReviewFindingList({
  title,
  findings,
  items,
  canAdopt,
  materializingFindingIds,
  onAdoptFinding,
  onSelectPage,
}: CWAIReviewFindingListProps) {
  return (
    <div
      style={{
        padding: "12px",
        borderRadius: "10px",
        border: `1px solid ${C.border}`,
        background: C.card,
      }}
    >
      <div
        style={{
          fontSize: "12px",
          fontWeight: 700,
          color: C.text,
          marginBottom: "8px",
        }}
      >
        {title}
      </div>

      {findings.length === 0 ? (
        <div
          style={{
            fontSize: "11px",
            color: C.textMuted,
          }}
        >
          暂无明确问题。
        </div>
      ) : (
        findings.map((finding) => {
          const severity =
            severityConfig(
              finding.severity,
            );

          const relatedItems =
            items.filter((item) =>
              findingMatchesItem(
                finding.id,
                item,
              ),
            );

          const materializing =
            materializingFindingIds.includes(
              finding.id,
            );

          return (
            <div
              key={finding.id}
              style={{
                padding: "9px",
                marginBottom: "8px",
                borderRadius: "8px",
                border: `1px solid ${severity.color}30`,
                background: severity.bg,
              }}
            >
              <div
                style={{
                  display: "flex",
                  alignItems: "flex-start",
                  gap: "6px",
                }}
              >
                <span
                  style={{
                    padding: "1px 5px",
                    borderRadius: "4px",
                    background:
                      severity.color,
                    color: "#fff",
                    fontSize: "9px",
                    fontWeight: 700,
                    flexShrink: 0,
                  }}
                >
                  {severity.label}
                </span>

                <div
                  style={{
                    minWidth: 0,
                    flex: 1,
                    fontSize: "11px",
                    fontWeight: 700,
                    color: C.text,
                    lineHeight: 1.5,
                  }}
                >
                  <DiscussionMarkdown
                    content={finding.title}
                    compact
                  />
                </div>
              </div>

              <div
                style={{
                  marginTop: "5px",
                  fontSize: "11px",
                  color: C.textSec,
                  lineHeight: 1.6,
                }}
              >
                <DiscussionMarkdown
                  content={finding.description}
                  compact
                />
              </div>

              {finding.lesson_or_outline_basis && (
                <EvidenceLine
                  label="依据"
                  value={
                    finding.lesson_or_outline_basis
                  }
                />
              )}

              {finding.page_evidence && (
                <EvidenceLine
                  label="页面"
                  value={
                    finding.page_evidence
                  }
                />
              )}

              {finding.code_evidence && (
                <EvidenceLine
                  label="代码"
                  value={
                    finding.code_evidence
                  }
                />
              )}

              {finding.continuity_evidence && (
                <EvidenceLine
                  label="连续性"
                  value={
                    finding.continuity_evidence
                  }
                />
              )}

              {finding.suggestion && (
                <div
                  style={{
                    marginTop: "5px",
                    padding: "5px 7px",
                    borderRadius: "5px",
                    background:
                      "rgba(255,255,255,0.72)",
                    color: C.text,
                    fontSize: "10px",
                    lineHeight: 1.5,
                  }}
                >
                  <strong>建议：</strong>

                  <DiscussionMarkdown
                    content={finding.suggestion}
                    compact
                  />
                </div>
              )}

              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "5px",
                  flexWrap: "wrap",
                  marginTop: "7px",
                }}
              >
                <PageButtons
                  pages={
                    finding.page_numbers
                  }
                  onSelectPage={
                    onSelectPage
                  }
                  inline
                />

                {finding.manual_review_required && (
                  <span
                    style={{
                      padding: "2px 6px",
                      borderRadius: "5px",
                      background: "#FFF7ED",
                      color: "#C2410C",
                      fontSize: "10px",
                      fontWeight: 600,
                    }}
                  >
                    需操作复核
                  </span>
                )}

                <span
                  style={{
                    marginLeft: "auto",
                    fontSize: "9px",
                    color: C.textMuted,
                  }}
                >
                  置信度{" "}
                  {finding.confidence}%
                </span>
              </div>

              {canAdopt &&
                relatedItems.length === 0 && (
                <button
                  type="button"
                  onClick={() =>
                    onAdoptFinding(
                      finding.id,
                    )
                  }
                  disabled={materializing}
                  style={{
                    width: "100%",
                    marginTop: "8px",
                    padding: "7px",
                    borderRadius: "7px",
                    border: `1px solid ${C.primary}`,
                    background:
                      materializing
                        ? "#E2E8F0"
                        : "#fff",
                    color:
                      materializing
                        ? C.textMuted
                        : C.primary,
                    fontSize: "10px",
                    fontWeight: 700,
                    cursor:
                      materializing
                        ? "not-allowed"
                        : "pointer",
                  }}
                >
                  {materializing
                    ? "正在建立页级整改项…"
                    : "采纳为可讨论整改项"}
                </button>
              )}

              {relatedItems.length > 0 && (
                <div
                  style={{
                    marginTop: "7px",
                    color: "#10B981",
                    fontSize: "10px",
                    fontWeight: 700,
                  }}
                >
                  ✓ 已进入页级审核清单
                </div>
              )}
            </div>
          );
        })
      )}
    </div>
  );
}

function EvidenceLine({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div
      style={{
        marginTop: "4px",
        fontSize: "10px",
        color: C.textSec,
        lineHeight: 1.5,
        wordBreak: "break-word",
      }}
    >
      <strong>{label}：</strong>

      <DiscussionMarkdown
        content={value}
        compact
      />
    </div>
  );
}

export function PageButtons({
  pages,
  onSelectPage,
  inline = false,
}: {
  pages: number[];
  onSelectPage: (
    pageNumber: number,
  ) => void;
  inline?: boolean;
}) {
  if (pages.length === 0) {
    return null;
  }

  return (
    <span
      style={{
        display: inline
          ? "inline-flex"
          : "flex",
        gap: "4px",
        flexWrap: "wrap",
        marginTop: inline
          ? 0
          : "5px",
        marginLeft: inline
          ? "4px"
          : 0,
      }}
    >
      {pages.map((page) => (
        <button
          key={page}
          type="button"
          onClick={() =>
            onSelectPage(page)
          }
          style={{
            padding: "2px 6px",
            borderRadius: "5px",
            border: `1px solid ${C.primary}40`,
            background: "#fff",
            color: C.primary,
            fontSize: "10px",
            fontWeight: 600,
            cursor: "pointer",
          }}
        >
          P{page}
        </button>
      ))}
    </span>
  );
}
