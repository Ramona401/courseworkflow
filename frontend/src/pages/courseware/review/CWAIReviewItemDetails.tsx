/**
 * CWAIReviewItemDetails.tsx
 *
 * 教师改进卡展开后的操作与过程区。
 *
 * R-01.1开始后，问题标题、可观察现象、教学影响、调整目标和验收检查
 * 统一由TeacherImprovementCard读取后端教师字段展示。本组件不再重复读取
 * title/description/original_suggestion/evidence_json，避免同一问题出现两套事实源。
 *
 * 本组件只负责：
 *   - 已确认问题关系的解释和跳转；
 *   - 准备方案、暂缓、恢复等既有状态操作入口；
 *   - 页面内容变化后的人工重新检查提示；
 *   - 单条整改讨论与指令版本流程。
 *
 * 这里仍是纯受控展示组件，不自行发请求、不决定权限。
 */

import type {
  CWAIReviewItem,
  CWAIReviewItemRelation,
} from "@/api/coursewares";

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
  resolveCWAIReviewPageChangeTeacherCopy,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewItemDetailsProps {
  experience: CWAIReviewItemExperience;
  item: CWAIReviewItem;

  itemMap: ReadonlyMap<
    string,
    CWAIReviewItem
  >;

  activeRelations:
    CWAIReviewItemRelation[];

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

function resolvePageChangeNotice(
  experience: CWAIReviewItemExperience,
  item: CWAIReviewItem,
): string {
  const pageChangeCopy =
    resolveCWAIReviewPageChangeTeacherCopy(
      item.status,
    );

  if (!pageChangeCopy) {
    return "";
  }

  if (item.status === "stale") {
    if (experience === "review") {
      return (
        `${pageChangeCopy.guidance} ` +
        "请重新判断这个问题是否仍然成立，再决定后续审核要求。"
      );
    }

    if (experience === "self") {
      return (
        `${pageChangeCopy.guidance} ` +
        "确认仍符合当前修改方案后，再重新登记为修改完成。"
      );
    }

    return (
      `${pageChangeCopy.guidance} ` +
      "请对照当前修改要求确认页面是否仍然达到要求，再重新登记为修改完成。"
    );
  }

  if (experience === "review") {
    return (
      `${pageChangeCopy.guidance} ` +
      "请确认原问题是否已经由其他页面处理，再决定是否继续要求修改。"
    );
  }

  if (experience === "self") {
    return (
      `${pageChangeCopy.guidance} ` +
      "请确认这项自审修改是否已经由其他页面处理，或是否仍需继续调整。"
    );
  }

  return (
    `${pageChangeCopy.guidance} ` +
    "请确认当前修改要求是否已经由其他页面处理，再决定后续整改方式。"
  );
}

export default function CWAIReviewItemDetails({
  experience,
  item,
  itemMap,
  activeRelations,
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

  const pauseReasonLength =
    Array.from(pauseReason).length;

  const pauseReasonInvalid =
    !pauseReason.trim() ||
    pauseReasonLength > 500;

  const pageChangeNotice =
    resolvePageChangeNotice(
      experience,
      item,
    );

  return (
    <div
      style={{
        paddingTop: "12px",
      }}
    >
      {activeRelations.length > 0 && (
        <div style={relationContainerStyle}>
          <div style={sectionTitleStyle}>
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
                    relation.source_item_id ===
                      item.id
                      ? relation.target_item_id
                      : relation.source_item_id,
                  )
                }
                onSelectPage={onSelectPage}
              />
            ),
          )}
        </div>
      )}

      <div style={actionRowStyle}>
        {item.page_number_snapshot > 0 && (
          <button
            type="button"
            onClick={() =>
              onSelectPage(
                item.page_number_snapshot,
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
              : item.status === "confirmed"
                ? copy.prepareAgainAction
                : copy.prepareAction}
          </button>
        )}

        {canPause && (
          <button
            type="button"
            onClick={onTogglePauseForm}
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

      {showPauseForm && canPause && (
        <div style={pauseContainerStyle}>
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

          <div style={pauseFooterStyle}>
            <span
              style={{
                flex: 1,
                color: C.textMuted,
                fontSize: "12px",
              }}
            >
              {pauseReasonLength}/500；以后仍可恢复
            </span>

            <button
              type="button"
              onClick={onPause}
              disabled={
                stateBusy ||
                pauseReasonInvalid
              }
              style={
                cwAIReviewPrimaryButtonStyle(
                  "warning",
                  stateBusy ||
                    pauseReasonInvalid,
                )
              }
            >
              {stateAction === "dismiss"
                ? "正在保存…"
                : copy.pauseConfirm}
            </button>
          </div>
        </div>
      )}

      {pageChangeNotice && (
        <div
          role="status"
          style={pageChangeNoticeStyle}
        >
          {pageChangeNotice}
        </div>
      )}

      <div
        style={discussionContainerStyle}
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

  const relatedLabel =
    (
      relatedItem?.teacher_title ||
      ""
    ).trim() ||
    "相关问题";

  return (
    <div style={relationRowStyle}>
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
            .page_number_snapshot > 0 && (
          <button
            type="button"
            onClick={() =>
              onSelectPage(
                relatedItem
                  .page_number_snapshot,
              )
            }
            style={
              relationPageButtonStyle
            }
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
        相关问题：{relatedLabel}
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

const sectionTitleStyle = {
  color: C.text,
  fontSize: "14px",
  fontWeight: 700,
} as const;

const relationContainerStyle = {
  padding: "12px",
  borderRadius: "9px",
  border:
    `1px solid ${C.border}`,
  background: "#F8FAFC",
} as const;

const relationRowStyle = {
  marginTop: "6px",
  paddingTop: "6px",
  borderTop:
    `1px solid ${C.border}`,
} as const;

const relationPageButtonStyle = {
  minHeight: "32px",
  padding: "5px 8px",
  borderRadius: "6px",
  border:
    `1px solid ${C.primary}`,
  background: "#FFFFFF",
  color: C.primary,
  fontSize: "11px",
  fontWeight: 700,
  cursor: "pointer",
} as const;

const actionRowStyle = {
  display: "flex",
  gap: "8px",
  flexWrap: "wrap",
  marginTop: "12px",
} as const;

const pauseContainerStyle = {
  marginTop: "12px",
  padding: "12px",
  borderRadius: "9px",
  border:
    "1px solid #FED7AA",
  background: "#FFF7ED",
} as const;

const pauseFooterStyle = {
  display: "flex",
  alignItems: "center",
  gap: "8px",
  marginTop: "6px",
} as const;

const pageChangeNoticeStyle = {
  marginTop: "12px",
  padding: "10px 12px",
  borderRadius: "8px",
  background: "#FEF2F2",
  color: C.danger,
  fontSize: "14px",
  lineHeight: 1.6,
} as const;

const discussionContainerStyle = {
  marginTop: "14px",
  paddingTop: "14px",
  borderTop:
    `1px solid ${C.border}`,
} as const;
