/**
 * CWAIReviewItemDetails.tsx
 *
 * 用户主动打开“更多信息”后显示的深入处理区。
 *
 * 审核员可以完善整改要求；
 * 自审作者可以完善自己的修改方案；
 * 整改作者只能查看审核要求和处理记录，不会改写审核员的要求。
 */

import type {
  CWAIReviewItem,
  CWAIReviewItemRelation,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  CW_GLOBAL_RELATION_CONFIG,
} from "./CWAIReviewGlobalDiscussion.shared";
import CWAIReviewItemDiscussionView from "./CWAIReviewItemDiscussion";
import {
  CW_AI_REVIEW_ITEM_COLORS as C,
  type CWAIReviewItemExperience,
  type CWAIReviewItemStateAction,
  cwAIReviewPauseTextareaStyle,
  cwAIReviewPrimaryButtonStyle,
  cwAIReviewSecondaryButtonStyle,
  resolveCWAIReviewItemExperienceCopy,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewItemDetailsProps {
  experience:
    CWAIReviewItemExperience;

  item: CWAIReviewItem;

  itemMap:
    ReadonlyMap<
      string,
      CWAIReviewItem
    >;

  activeRelations:
    CWAIReviewItemRelation[];

  manuallyAdded: boolean;
  evidenceSummary: string;
  pageLabel: string;

  canPrepare: boolean;
  canPause: boolean;
  canResume: boolean;

  stateBusy: boolean;
  generating: boolean;

  showPauseForm: boolean;
  pauseReason: string;

  stateAction:
    CWAIReviewItemStateAction;

  discussionVersion: number;

  onSelectPage: (
    pageNumber: number,
  ) => void;

  onChanged: (
    item: CWAIReviewItem,
  ) => void;

  onPrepareModification: () => void;
  onTogglePauseForm: () => void;

  onPauseReasonChange: (
    value: string,
  ) => void;

  onPause: () => void;
  onResume: () => void;
}

export default function CWAIReviewItemDetails({
  experience,
  item,
  itemMap,
  activeRelations,
  manuallyAdded,
  evidenceSummary,
  pageLabel,
  canPrepare,
  canPause,
  canResume,
  stateBusy,
  generating,
  showPauseForm,
  pauseReason,
  stateAction,
  discussionVersion,
  onSelectPage,
  onChanged,
  onPrepareModification,
  onTogglePauseForm,
  onPauseReasonChange,
  onPause,
  onResume,
}: CWAIReviewItemDetailsProps) {
  const copy =
    resolveCWAIReviewItemExperienceCopy(
      experience,
    );

  return (
    <div>
      {item.title.trim() &&
        item.description.trim() && (
        <DetailBlock
          title={
            copy.descriptionTitle
          }
          content={
            item.description
          }
        />
      )}

      {item.original_suggestion.trim() && (
        <DetailBlock
          title={
            manuallyAdded
              ? copy.suggestionManualTitle
              : copy.suggestionAITitle
          }
          content={
            item.original_suggestion
          }
        />
      )}

      {evidenceSummary && (
        <DetailBlock
          title={
            copy.evidenceTitle
          }
          content={evidenceSummary}
        />
      )}

      {activeRelations.length > 0 && (
        <div
          style={{
            marginTop: "12px",
            padding: "12px",
            borderRadius: "9px",
            border:
              `1px solid ${C.border}`,
            background: "#F8FAFC",
          }}
        >
          <div
            style={{
              color: C.text,
              fontSize: "14px",
              fontWeight: 700,
            }}
          >
            与其他问题的联系
          </div>

          {activeRelations.map(
            (relation) => (
              <ProblemConnectionRow
                key={relation.id}
                itemID={item.id}
                relation={relation}
                relatedItem={
                  itemMap.get(
                    relation
                      .source_item_id ===
                      item.id
                      ? relation
                          .target_item_id
                      : relation
                          .source_item_id,
                  )
                }
                onSelectPage={
                  onSelectPage
                }
              />
            ),
          )}
        </div>
      )}

      <div
        style={{
          display: "flex",
          gap: "8px",
          flexWrap: "wrap",
          marginTop: "12px",
        }}
      >
        {item.page_number_snapshot >
          0 && (
          <button
            type="button"
            onClick={() =>
              onSelectPage(
                item
                  .page_number_snapshot,
              )
            }
            style={
              cwAIReviewSecondaryButtonStyle
            }
          >
            打开{pageLabel}
          </button>
        )}

        {canPrepare && (
          <button
            type="button"
            onClick={
              onPrepareModification
            }
            disabled={stateBusy}
            style={
              cwAIReviewSecondaryButtonStyle
            }
          >
            {generating
              ? "正在准备…"
              : item.status ===
                  "confirmed"
                ? copy.prepareAgainAction
                : copy.prepareAction}
          </button>
        )}

        {canPause && (
          <button
            type="button"
            onClick={
              onTogglePauseForm
            }
            disabled={stateBusy}
            style={{
              ...cwAIReviewSecondaryButtonStyle,
              border:
                `1px solid ${C.warning}`,
              color: C.warning,
            }}
          >
            {showPauseForm
              ? "取消"
              : copy.pauseAction}
          </button>
        )}

        {canResume && (
          <button
            type="button"
            onClick={onResume}
            disabled={stateBusy}
            style={
              cwAIReviewSecondaryButtonStyle
            }
          >
            {copy.resumeAction}
          </button>
        )}
      </div>

      {showPauseForm &&
        canPause && (
        <div
          style={{
            marginTop: "12px",
            padding: "12px",
            borderRadius: "9px",
            border:
              "1px solid #FED7AA",
            background: "#FFF7ED",
          }}
        >
          <div
            style={{
              color: "#9A3412",
              fontSize: "14px",
              fontWeight: 700,
            }}
          >
            {copy.pauseQuestion}
          </div>

          <textarea
            value={pauseReason}
            onChange={(event) =>
              onPauseReasonChange(
                event.target.value,
              )
            }
            rows={2}
            maxLength={500}
            disabled={stateBusy}
            placeholder={
              copy.pausePlaceholder
            }
            style={
              cwAIReviewPauseTextareaStyle
            }
          />

          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "8px",
              marginTop: "6px",
            }}
          >
            <span
              style={{
                flex: 1,
                color: C.textMuted,
                fontSize: "12px",
              }}
            >
              {
                Array.from(
                  pauseReason,
                ).length
              }
              /500；以后仍可恢复
            </span>

            <button
              type="button"
              onClick={onPause}
              disabled={
                stateBusy ||
                !pauseReason.trim()
              }
              style={
                cwAIReviewPrimaryButtonStyle(
                  "warning",
                  stateBusy ||
                    !pauseReason.trim(),
                )
              }
            >
              {stateAction ===
              "dismiss"
                ? "正在保存…"
                : copy.pauseConfirm}
            </button>
          </div>
        </div>
      )}

      {(
        item.status === "stale" ||
        item.status === "orphaned"
      ) && (
        <div
          style={{
            marginTop: "12px",
            padding: "10px 12px",
            borderRadius: "8px",
            background: "#FEF2F2",
            color: C.danger,
            fontSize: "14px",
            lineHeight: 1.6,
          }}
        >
          {item.status === "stale"
            ? experience ===
              "review"
              ? "页面内容已经在审核后发生变化，请重新查看页面，再判断这个问题是否仍然成立。"
              : experience ===
                "self"
                ? "页面内容已经变化。请先打开当前页面实际检查；确认仍符合原修改方案后，点击“重新检查当前页面”。"
                : "页面内容已经变化。请先对照审核要求实际检查；确认当前页面符合要求后，点击“重新检查当前页面”。"
            : "原来的页面已经删除，这条内容只保留供以后回看。"}
        </div>
      )}

      <div
        style={{
          marginTop: "14px",
          paddingTop: "14px",
          borderTop:
            `1px solid ${C.border}`,
        }}
      >
        <CWAIReviewItemDiscussionView
          key={`${item.id}-${discussionVersion}`}
          experience={experience}
          item={item}
          onChanged={onChanged}
        />
      </div>
    </div>
  );
}

function DetailBlock({
  title,
  content,
}: {
  title: string;
  content: string;
}) {
  return (
    <div
      style={{
        marginTop: "12px",
        padding: "12px",
        borderRadius: "9px",
        background: "#F8FAFC",
        color: C.textSec,
      }}
    >
      <div
        style={{
          marginBottom: "6px",
          color: C.text,
          fontSize: "14px",
          fontWeight: 700,
        }}
      >
        {title}
      </div>

      <div
        style={{
          maxWidth: "75ch",
          fontSize: "14px",
          lineHeight: 1.6,
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

function resolveConnectionExplanation(
  relationType:
    CWAIReviewItemRelation[
      "relation_type"
    ],
  isSource: boolean,
): [
  string,
  string,
] {
  switch (relationType) {
    case "duplicate":
      return isSource
        ? [
            "与保留的问题重复",
            "两条问题说的是同一件事，需要决定是否只保留另一条。",
          ]
        : [
            "建议保留这条",
            "另一条问题与本条重复，但不会被自动删除。",
          ];

    case "merge":
      return isSource
        ? [
            "可以并入另一条一起处理",
            "建议把本条需要修改的内容放到关联问题中统一考虑。",
          ]
        : [
            "可以在这条中统一处理",
            "关联问题可以与本条一起形成一份更完整的修改要求。",
          ];

    case "dependency":
      return isSource
        ? [
            "需要先处理另一条问题",
            "完成关联问题后，再回来处理本条。",
          ]
        : [
            "建议先处理这条",
            "本条完成后，才能继续处理关联问题。",
          ];

    case "possibly_resolved":
      return isSource
        ? [
            "完成关联修改后再检查",
            "另一条问题修改完成后，本条可能已经同时解决，但仍需重新确认。",
          ]
        : [
            "这条修改可能同时解决另一条",
            "完成本条修改后，请再检查关联问题是否仍然存在。",
          ];

    case "conflict":
      return [
        "两条修改要求互相矛盾",
        "需要先决定最终采用哪种处理方式。",
      ];
  }
}

function ProblemConnectionRow({
  itemID,
  relation,
  relatedItem,
  onSelectPage,
}: {
  itemID: string;
  relation:
    CWAIReviewItemRelation;
  relatedItem?:
    CWAIReviewItem;
  onSelectPage: (
    pageNumber: number,
  ) => void;
}) {
  const config =
    CW_GLOBAL_RELATION_CONFIG[
      relation.relation_type
    ];

  const isSource =
    relation.source_item_id ===
    itemID;

  const [
    connectionLabel,
    guidance,
  ] =
    resolveConnectionExplanation(
      relation.relation_type,
      isSource,
    );

  const relatedID =
    isSource
      ? relation.target_item_id
      : relation.source_item_id;

  const relatedLabel =
    relatedItem?.title.trim() ||
    relatedItem?.description.trim() ||
    relatedID;

  return (
    <div
      style={{
        marginTop: "6px",
        paddingTop: "6px",
        borderTop:
          `1px solid ${C.border}`,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "5px",
          flexWrap: "wrap",
        }}
      >
        <span
          style={{
            padding: "2px 6px",
            borderRadius: "5px",
            background:
              config.background,
            color: config.color,
            fontSize: "11px",
            fontWeight: 700,
          }}
        >
          {connectionLabel}
        </span>

        {relatedItem &&
          relatedItem
            .page_number_snapshot >
            0 && (
          <button
            type="button"
            onClick={() =>
              onSelectPage(
                relatedItem
                  .page_number_snapshot,
              )
            }
            style={{
              minHeight: "32px",
              padding: "5px 8px",
              borderRadius: "6px",
              border:
                `1px solid ${C.primary}`,
              background: "#fff",
              color: C.primary,
              fontSize: "11px",
              fontWeight: 700,
              cursor: "pointer",
            }}
          >
            打开P
            {
              relatedItem
                .page_number_snapshot
            }
          </button>
        )}
      </div>

      <div
        style={{
          marginTop: "6px",
          color: C.textSec,
          fontSize: "13px",
          lineHeight: 1.5,
          wordBreak: "break-word",
        }}
      >
        相关问题：
        {relatedLabel}
      </div>

      <div
        style={{
          marginTop: "4px",
          color: C.textMuted,
          fontSize: "12px",
          lineHeight: 1.5,
        }}
      >
        {guidance}
      </div>
    </div>
  );
}
