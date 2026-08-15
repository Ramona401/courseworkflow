/**
 * CWAIReviewImpactPlanOperation.tsx
 *
 * R-07统一影响方案的纯展示operation列表。
 *
 * 只展示后端安全Preview中的action_label、summary和教师需要的payload字段。
 * operation_id仅作为勾选键使用，不展示；preconditions从未进入浏览器响应。
 */

import type {
  CWAIReviewImpactOperation,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
} from "./CWAIReviewGlobalDiscussion.shared";

type ImpactDetail = {
  label: string;
  value: string;
};

export interface CWAIReviewImpactPlanOperationListProps {
  operations: CWAIReviewImpactOperation[];
  selectedOperationIDs: Set<string>;
  disabled: boolean;

  onSelectedChange: (
    operationID: string,
    selected: boolean,
  ) => void;
}

export default function CWAIReviewImpactPlanOperationList({
  operations,
  selectedOperationIDs,
  disabled,
  onSelectedChange,
}: CWAIReviewImpactPlanOperationListProps) {
  return (
    <div style={{ marginTop: "7px" }}>
      {operations.map((operation) => (
        <ImpactOperationCard
          key={operation.operation_id}
          operation={operation}
          selected={
            selectedOperationIDs.has(
              operation.operation_id,
            )
          }
          disabled={disabled}
          onSelectedChange={(selected) =>
            onSelectedChange(
              operation.operation_id,
              selected,
            )
          }
        />
      ))}
    </div>
  );
}

function ImpactOperationCard({
  operation,
  selected,
  disabled,
  onSelectedChange,
}: {
  operation: CWAIReviewImpactOperation;
  selected: boolean;
  disabled: boolean;
  onSelectedChange: (selected: boolean) => void;
}) {
  const details =
    buildImpactOperationDetails(operation);

  return (
    <label
      style={{
        display: "block",
        marginBottom: "6px",
        padding: "8px",
        borderRadius: "7px",
        border:
          `1px solid ${
            selected
              ? `${C.primary}55`
              : C.border
          }`,
        background:
          selected
            ? "#fff"
            : "#F8FAFC",
        cursor:
          disabled
            ? "default"
            : "pointer",
        opacity:
          disabled && !selected
            ? 0.65
            : 1,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: "7px",
        }}
      >
        <input
          type="checkbox"
          checked={selected}
          disabled={disabled}
          onChange={(event) =>
            onSelectedChange(
              event.target.checked,
            )
          }
          style={{
            marginTop: "2px",
            flex: "0 0 auto",
          }}
        />

        <div
          style={{
            minWidth: 0,
            flex: 1,
          }}
        >
          <div
            style={{
              display: "flex",
              gap: "5px",
              alignItems: "center",
              flexWrap: "wrap",
            }}
          >
            <span
              style={{
                color: C.primary,
                fontSize: "9px",
                fontWeight: 700,
              }}
            >
              {operation.action_label ||
                fallbackImpactActionLabel(
                  operation.operation_type,
                )}
            </span>

            <span
              style={{
                color: C.textMuted,
                fontSize: "8px",
              }}
            >
              {impactOperationTypeLabel(
                operation.operation_type,
              )}
            </span>
          </div>

          <div
            style={{
              marginTop: "4px",
              color: C.text,
              fontSize: "9px",
              lineHeight: 1.5,
            }}
          >
            <DiscussionMarkdown
              content={operation.summary}
              compact
            />
          </div>

          {details.length > 0 && (
            <div
              style={{
                marginTop: "5px",
                paddingTop: "5px",
                borderTop:
                  `1px dashed ${C.border}`,
              }}
            >
              {details.map((detail) => (
                <div
                  key={`${detail.label}:${detail.value}`}
                  style={{
                    display: "flex",
                    gap: "6px",
                    marginTop: "3px",
                    fontSize: "8px",
                    lineHeight: 1.5,
                  }}
                >
                  <span
                    style={{
                      minWidth: "52px",
                      color: C.textMuted,
                    }}
                  >
                    {detail.label}
                  </span>

                  <span
                    style={{
                      minWidth: 0,
                      color: C.textSec,
                      whiteSpace: "pre-wrap",
                      overflowWrap: "anywhere",
                    }}
                  >
                    {detail.value}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </label>
  );
}

function buildImpactOperationDetails(
  operation: CWAIReviewImpactOperation,
): ImpactDetail[] {
  const payload =
    operation.payload || {};

  const details: ImpactDetail[] =
    [];

  const push = (
    label: string,
    value: unknown,
  ) => {
    const normalized =
      impactDetailValue(value);

    if (normalized) {
      details.push({
        label,
        value: normalized,
      });
    }
  };

  switch (operation.operation_type) {
    case "create_group":
      push("问题组", payload.name);
      push(
        "包含问题",
        impactItemCount(
          payload.item_ids,
        ),
      );
      push("原因", payload.reason);
      break;

    case "move_group_member":
      push(
        "调整原因",
        payload.reason,
      );
      break;

    case "merge_groups":
      push(
        "合并原因",
        payload.reason,
      );
      break;

    case "split_group":
      push(
        "新问题组",
        payload.name,
      );
      push(
        "拆出问题",
        impactItemCount(
          payload.item_ids,
        ),
      );
      push(
        "拆分原因",
        payload.reason,
      );
      break;

    case "create_relation":
      push(
        "判断依据",
        payload.explanation,
      );
      break;

    case "cancel_relation":
      push(
        "取消原因",
        payload.reason,
      );
      break;

    case "create_item":
      push(
        "问题标题",
        payload.title,
      );
      push(
        "严重程度",
        payload.severity,
      );
      push(
        "改进维度",
        payload.dimension,
      );
      push(
        "问题说明",
        payload.description,
      );
      push(
        "候选建议",
        payload.candidate_instruction,
      );
      break;

    case "dismiss_item":
      push(
        "暂不处理原因",
        payload.reason,
      );
      break;

    case "update_candidate_suggestion":
      push(
        "新的候选建议",
        payload.candidate_instruction,
      );
      break;
  }

  return details;
}

function impactDetailValue(
  value: unknown,
): string {
  if (typeof value === "string") {
    return value.trim();
  }

  if (
    typeof value === "number" ||
    typeof value === "boolean"
  ) {
    return String(value);
  }

  return "";
}

function impactItemCount(
  value: unknown,
): string {
  if (!Array.isArray(value)) {
    return "";
  }

  return `${value.length}条`;
}

function impactOperationTypeLabel(
  operationType:
    CWAIReviewImpactOperation["operation_type"],
): string {
  switch (operationType) {
    case "create_group":
      return "建立问题组";

    case "move_group_member":
      return "调整问题分组";

    case "merge_groups":
      return "合并问题组";

    case "split_group":
      return "拆分问题组";

    case "create_relation":
      return "建立问题关系";

    case "cancel_relation":
      return "取消问题关系";

    case "create_item":
      return "新增独立问题";

    case "dismiss_item":
      return "暂不处理问题";

    case "update_candidate_suggestion":
      return "更新候选建议";
  }
}

function fallbackImpactActionLabel(
  operationType:
    CWAIReviewImpactOperation["operation_type"],
): string {
  switch (operationType) {
    case "create_group":
    case "create_relation":
    case "create_item":
      return "将新增";

    case "merge_groups":
      return "将合并";

    case "dismiss_item":
      return "将暂不处理";

    case "update_candidate_suggestion":
      return "将更新建议";

    case "split_group":
      return "将拆分";

    case "cancel_relation":
      return "将取消关系";

    case "move_group_member":
      return "将调整分组";
  }
}
