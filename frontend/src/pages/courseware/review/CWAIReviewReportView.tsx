/**
 * CWAIReviewReportView.tsx
 *
 * 课件AI最终报告、不可变审核配置、统一问题工作台、
 * 问题采纳和关系治理视图。
 *
 * 本组件不会执行页面修改或提交人工审核决定。
 */

import type {
  CWAIReviewConfigSnapshot,
} from "@/api/coursewares.ai-review-config";

import {
  readCWAIReviewFinalReportConfig,
} from "@/api/coursewares.ai-review-config";

import type {
  CWAIReviewFinalReport,
  CWAIReviewFinding,
  CWAIReviewItem,
  CWAIReviewItemRelation,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import CWAIReviewConfigSummary from "./CWAIReviewConfigSummary";
import CWAIReviewDirectRelationGovernance from "./CWAIReviewDirectRelationGovernance";
import CWAIReviewUnifiedIssueList from "./CWAIReviewUnifiedIssueList";
import CWAIReviewFindingList, {
  PageButtons,
  findingMatchesItem,
  severityConfig,
} from "./CWAIReviewFindingList";

const C = {
  primary: "#4F7BE8",
  primaryLight:
    "rgba(79,123,232,0.08)",
  success: "#10B981",
  text: "#1F2937",
  textSec: "#6B7280",
  textMuted: "#9CA3AF",
  border: "#E5E7EB",
  softBorder: "#F3F4F6",
  card: "#FFFFFF",
};

export interface CWAIReviewSelectionRequest {
  id: number;
  itemIds: string[];
}

export interface CWAIReviewReportViewProps {
  sessionId: string;
  mode?: "formal" | "self";

  finalReport:
    | CWAIReviewFinalReport
    | null;

  /**
   * 会话不可变配置。
   *
   * 新报告优先显示report.review_config；
   * R-02上线前旧报告没有该字段时回退显示会话快照。
   */
  reviewConfig:
    | CWAIReviewConfigSnapshot
    | null;

  batchFindings:
    CWAIReviewFinding[];

  items: CWAIReviewItem[];

  governanceRelations:
    CWAIReviewItemRelation[];

  /**
   * 临时工作选择。
   */
  workSelectedItemIds: string[];

  /**
   * 正式退回交付选择。
   */
  deliverySelectedItemIds: string[];

  materializingFindingIds:
    string[];

  relationSelectionRequest:
    CWAIReviewSelectionRequest | null;

  onSelectPage: (
    pageNumber: number,
  ) => void;

  onUseReviewComment: (
    comment: string,
  ) => void;

  onAdoptFinding: (
    findingId: string,
  ) => void;

  onToggleWorkSelection: (
    itemID: string,
    selected: boolean,
  ) => void;

  onClearWorkSelection: () => void;

  onOpenGlobalDiscussion: () => void;

  onOpenDirectRelation: () => void;

  onToggleDeliverySelection: (
    itemId: string,
    selected: boolean,
  ) => void;

  onItemChanged: (
    item: CWAIReviewItem,
  ) => void;

  onGovernanceRelationsChanged: (
    relations:
      CWAIReviewItemRelation[],
  ) => void;

  onInjectToRefine?: (
    item: CWAIReviewItem,
  ) => void;
}

export default function CWAIReviewReportView({
  sessionId,
  mode = "formal",
  finalReport,
  reviewConfig,
  batchFindings,
  items,
  governanceRelations,
  workSelectedItemIds,
  deliverySelectedItemIds,
  materializingFindingIds,
  relationSelectionRequest,
  onSelectPage,
  onUseReviewComment,
  onAdoptFinding,
  onToggleWorkSelection,
  onClearWorkSelection,
  onOpenGlobalDiscussion,
  onOpenDirectRelation,
  onToggleDeliverySelection,
  onItemChanged,
  onGovernanceRelationsChanged,
  onInjectToRefine,
}: CWAIReviewReportViewProps) {
  const isSelfReview =
    mode === "self";

  if (!finalReport) {
    if (
      batchFindings.length === 0
    ) {
      return null;
    }

    return (
      <CWAIReviewFindingList
        title={`当前已发现 ${batchFindings.length}`}
        findings={
          batchFindings
        }
        items={items}
        canAdopt={false}
        materializingFindingIds={
          materializingFindingIds
        }
        onAdoptFinding={
          onAdoptFinding
        }
        onSelectPage={
          onSelectPage
        }
      />
    );
  }

  const overall =
    severityConfig(
      finalReport.overall_risk,
    );

  const reportConfig =
    readCWAIReviewFinalReportConfig(
      finalReport,
    );

  const effectiveReviewConfig =
    reportConfig ||
    reviewConfig;

  const remainingFindings =
    finalReport.findings.filter(
      (finding) =>
        !items.some(
          (item) =>
            findingMatchesItem(
              finding.id,
              item,
            ),
        ),
    );

  return (
    <>
      <CWAIReviewConfigSummary
        config={
          effectiveReviewConfig
        }
        title={
          isSelfReview
            ? "本次自审配置"
            : "本次审核配置"
        }
      />

      <div
        style={{
          padding: "12px",
          borderRadius: "10px",
          border:
            `1px solid ${overall.color}40`,
          background: overall.bg,
        }}
      >
        <div
          style={{
            fontSize: "13px",
            fontWeight: 700,
            color: overall.color,
            marginBottom: "7px",
          }}
        >
          整课综合风险：
          {overall.label}
        </div>

        <div
          style={{
            fontSize: "12px",
            lineHeight: 1.7,
            color: C.text,
            whiteSpace:
              "pre-wrap",
          }}
        >
          <DiscussionMarkdown
            content={
              finalReport.summary
            }
            compact
          />
        </div>
      </div>

      {finalReport
        .strengths.length >
        0 && (
        <div
          style={{
            padding: "12px",
            borderRadius: "10px",
            border:
              `1px solid ${C.border}`,
            background: C.card,
          }}
        >
          <div
            style={{
              fontSize: "12px",
              fontWeight: 700,
              color: C.success,
              marginBottom: "7px",
            }}
          >
            ✓ 明确优点
          </div>

          {finalReport.strengths.map(
            (
              strength,
              index,
            ) => (
              <div
                key={`${strength}-${index}`}
                style={{
                  display: "flex",
                  alignItems:
                    "flex-start",
                  gap: "4px",
                  color: C.textSec,
                  marginBottom: "4px",
                }}
              >
                <span
                  style={{
                    flexShrink: 0,
                    fontSize: "11px",
                    lineHeight: 1.7,
                  }}
                >
                  •
                </span>

                <div
                  style={{
                    minWidth: 0,
                    flex: 1,
                  }}
                >
                  <DiscussionMarkdown
                    content={
                      strength
                    }
                    compact
                  />
                </div>
              </div>
            ),
          )}
        </div>
      )}

      {items.length > 0 && (
        <CWAIReviewUnifiedIssueList
          mode={mode}
          items={items}
          governanceRelations={
            governanceRelations
          }
          workSelectedItemIds={
            workSelectedItemIds
          }
          deliverySelectedItemIds={
            deliverySelectedItemIds
          }
          onToggleWorkSelection={
            onToggleWorkSelection
          }
          onClearWorkSelection={
            onClearWorkSelection
          }
          onOpenGlobalDiscussion={
            onOpenGlobalDiscussion
          }
          onOpenDirectRelation={
            onOpenDirectRelation
          }
          onToggleDeliverySelection={
            onToggleDeliverySelection
          }
          onItemChanged={
            onItemChanged
          }
          onSelectPage={
            onSelectPage
          }
          onInjectToRefine={
            onInjectToRefine
          }
        />
      )}

      {sessionId &&
        items.length > 0 && (
        <div
          id="cw-ai-direct-relation-governance"
        >
          <CWAIReviewDirectRelationGovernance
            sessionId={
              sessionId
            }
            items={items}
            relations={
              governanceRelations
            }
            selectionRequest={
              relationSelectionRequest
            }
            onSelectPage={
              onSelectPage
            }
            onRelationsChanged={
              onGovernanceRelationsChanged
            }
          />
        </div>
      )}

      {remainingFindings.length >
        0 && (
        <CWAIReviewFindingList
          title={`${
            isSelfReview
              ? "其他待采纳自审发现"
              : "其他AI审核发现"
          } ${remainingFindings.length}`}
          findings={
            remainingFindings
          }
          items={items}
          canAdopt
          materializingFindingIds={
            materializingFindingIds
          }
          onAdoptFinding={
            onAdoptFinding
          }
          onSelectPage={
            onSelectPage
          }
        />
      )}

      {finalReport
        .priority_actions
        .length > 0 && (
        <div
          style={{
            padding: "12px",
            borderRadius: "10px",
            border:
              `1px solid ${C.border}`,
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
            优先修改动作
          </div>

          {finalReport
            .priority_actions
            .map((action) => (
              <div
                key={`${action.priority}-${action.title}`}
                style={{
                  padding: "8px 0",
                  borderBottom:
                    `1px solid ${C.softBorder}`,
                }}
              >
                <div
                  style={{
                    fontSize: "12px",
                    fontWeight: 600,
                    color: C.text,
                  }}
                >
                  <div
                    style={{
                      display: "flex",
                      alignItems:
                        "flex-start",
                      gap: "4px",
                    }}
                  >
                    <span
                      style={{
                        flexShrink: 0,
                      }}
                    >
                      {
                        action.priority
                      }.
                    </span>

                    <div
                      style={{
                        minWidth: 0,
                        flex: 1,
                      }}
                    >
                      <DiscussionMarkdown
                        content={
                          action.title
                        }
                        compact
                      />
                    </div>
                  </div>
                </div>

                <div
                  style={{
                    marginTop: "3px",
                    fontSize: "11px",
                    color: C.textSec,
                    lineHeight: 1.6,
                  }}
                >
                  <DiscussionMarkdown
                    content={
                      action.description
                    }
                    compact
                  />
                </div>

                {action.reason && (
                  <div
                    style={{
                      marginTop: "3px",
                      fontSize: "10px",
                      color: C.textMuted,
                      lineHeight: 1.5,
                    }}
                  >
                    <strong>
                      原因：
                    </strong>

                    <DiscussionMarkdown
                      content={
                        action.reason
                      }
                      compact
                    />
                  </div>
                )}

                {action
                  .page_numbers
                  .length > 0 && (
                  <PageButtons
                    pages={
                      action.page_numbers
                    }
                    onSelectPage={
                      onSelectPage
                    }
                  />
                )}
              </div>
            ))}
        </div>
      )}

      {finalReport
        .review_comment_draft && (
        <div
          style={{
            padding: "12px",
            borderRadius: "10px",
            border:
              `1px solid ${C.primary}40`,
            background:
              C.primaryLight,
          }}
        >
          <div
            style={{
              fontSize: "12px",
              fontWeight: 700,
              color: C.primary,
              marginBottom: "7px",
            }}
          >
            {isSelfReview
              ? "AI自审修改摘要"
              : "AI审核意见草稿"}
          </div>

          <div
            style={{
              fontSize: "11px",
              lineHeight: 1.7,
              color: C.text,
              whiteSpace:
                "pre-wrap",
            }}
          >
            <DiscussionMarkdown
              content={
                finalReport
                  .review_comment_draft
              }
              compact
            />
          </div>

          <button
            type="button"
            onClick={() =>
              onUseReviewComment(
                finalReport
                  .review_comment_draft,
              )
            }
            style={{
              width: "100%",
              marginTop: "9px",
              padding: "7px",
              borderRadius: "7px",
              border: "none",
              background: C.primary,
              color: "#fff",
              fontSize: "11px",
              fontWeight: 700,
              cursor: "pointer",
            }}
          >
            {isSelfReview
              ? "复制自审修改摘要"
              : "填入人工审核意见"}
          </button>

          <div
            style={{
              marginTop: "6px",
              fontSize: "10px",
              color: C.textMuted,
              lineHeight: 1.5,
            }}
          >
            {isSelfReview
              ? "自审项仅作者可见；确认后的指令仍需手动注入页面微调。"
              : "所有已确认整改项会自动进入退回清单；审核者仍可手动排除。"}
          </div>
        </div>
      )}

      {finalReport
        .manual_review_pages
        .length > 0 && (
        <div
          style={{
            padding: "10px 12px",
            borderRadius: "8px",
            background:
              "#FFF7ED",
            border:
              "1px solid #FED7AA",
            color: "#9A3412",
            fontSize: "11px",
            lineHeight: 1.6,
          }}
        >
          需要人工实际操作复核：
          <PageButtons
            pages={
              finalReport
                .manual_review_pages
            }
            onSelectPage={
              onSelectPage
            }
            inline
          />
        </div>
      )}

      {finalReport
        .human_decision_reminder && (
        <div
          style={{
            fontSize: "10px",
            color: C.textMuted,
            lineHeight: 1.6,
            textAlign: "center",
            padding: "0 6px 8px",
          }}
        >
          <DiscussionMarkdown
            content={
              finalReport
                .human_decision_reminder
            }
            compact
          />
        </div>
      )}
    </>
  );
}
