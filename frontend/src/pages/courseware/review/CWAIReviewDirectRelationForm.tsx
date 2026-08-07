/**
 * CWAIReviewDirectRelationForm.tsx
 *
 * 老师说明两条审核问题之间的联系。
 *
 * 系统只保存老师明确选择并确认的联系，
 * 不会因此自动删除问题、修改页面或改变本次退回内容。
 */

import type {
  CSSProperties,
} from "react";

import type {
  CWAIReviewGlobalRelationType,
  CWAIReviewItem,
} from "@/api/coursewares";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
  resolveCWGlobalItemPageLabel,
  resolveCWGlobalItemTitle,
} from "./CWAIReviewGlobalDiscussion.shared";
import {
  countCWGlobalRunes,
  CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES,
} from "./CWAIReviewGlobalGovernanceLimits";

export interface CWAIReviewDirectRelationFormProps {
  actionableItems:
    CWAIReviewItem[];

  relationType:
    CWAIReviewGlobalRelationType;

  sourceItemID: string;
  targetItemID: string;
  explanation: string;

  busyKey: string;
  relationAlreadyActive:
    boolean;
  canConfirm: boolean;

  onRelationTypeChange: (
    value:
      CWAIReviewGlobalRelationType,
  ) => void;

  onSourceItemIDChange: (
    value: string,
  ) => void;

  onTargetItemIDChange: (
    value: string,
  ) => void;

  onExplanationChange: (
    value: string,
  ) => void;

  onConfirm: () => void;
}

const RELATION_TYPES:
  CWAIReviewGlobalRelationType[] = [
    "duplicate",
    "conflict",
    "merge",
    "dependency",
    "possibly_resolved",
  ];

const RELATION_TYPE_LABEL:
  Record<
    CWAIReviewGlobalRelationType,
    string
  > = {
    duplicate:
      "两条问题重复，保留一条",
    conflict:
      "两条修改要求互相矛盾",
    merge:
      "两条问题可以一起处理",
    dependency:
      "需要先处理另一条问题",
    possibly_resolved:
      "修改另一处后可能已经解决",
  };

function relationDirectionHelp(
  relationType:
    CWAIReviewGlobalRelationType,
): {
  sourceLabel: string;
  targetLabel: string;
  explanation: string;
} {
  switch (relationType) {
    case "duplicate":
      return {
        sourceLabel:
          "可以合并的问题",
        targetLabel:
          "建议保留的问题",
        explanation:
          "这两条问题说的是同一件事，建议保留第二条作为主要修改任务。",
      };

    case "merge":
      return {
        sourceLabel:
          "准备并入的问题",
        targetLabel:
          "统一处理的问题",
        explanation:
          "两条问题密切相关，可以放在第二条中一起形成完整的修改要求。",
      };

    case "dependency":
      return {
        sourceLabel:
          "后处理的问题",
        targetLabel:
          "需要先处理的问题",
        explanation:
          "先完成第二条，再回来确定第一条应该怎样处理。",
      };

    case "possibly_resolved":
      return {
        sourceLabel:
          "需要再次检查的问题",
        targetLabel:
          "先完成修改的问题",
        explanation:
          "完成第二条修改后，第一条可能已经同时解决，但仍需要老师重新检查。",
      };

    case "conflict":
      return {
        sourceLabel:
          "修改要求A",
        targetLabel:
          "修改要求B",
        explanation:
          "两条修改要求互相矛盾，需要先决定最终采用哪种处理方式。",
      };
  }
}

function itemOptionLabel(
  item: CWAIReviewItem,
): string {
  return [
    resolveCWGlobalItemPageLabel(
      item,
    ),
    resolveCWGlobalItemTitle(
      item,
    ),
  ].join(" · ");
}

export default function CWAIReviewDirectRelationForm({
  actionableItems,
  relationType,
  sourceItemID,
  targetItemID,
  explanation,
  busyKey,
  relationAlreadyActive,
  canConfirm,
  onRelationTypeChange,
  onSourceItemIDChange,
  onTargetItemIDChange,
  onExplanationChange,
  onConfirm,
}: CWAIReviewDirectRelationFormProps) {
  const directionHelp =
    relationDirectionHelp(
      relationType,
    );

  const explanationLength =
    countCWGlobalRunes(
      explanation.trim(),
    );

  const busy = !!busyKey;

  return (
    <div>
      <div
        style={{
          color: C.text,
          fontSize: "10px",
          fontWeight: 700,
        }}
      >
        说明两条问题之间的联系
      </div>

      <div
        style={{
          marginTop: "3px",
          color: C.textMuted,
          fontSize: "9px",
          lineHeight: 1.55,
        }}
      >
        保存后只会记录老师的判断，不会自动关闭问题或修改页面。
      </div>

      {actionableItems.length <
        2 ? (
        <div
          style={{
            marginTop: "7px",
            padding: "8px",
            borderRadius: "7px",
            background:
              C.warningSoft,
            color: C.warning,
            fontSize: "9px",
            lineHeight: 1.55,
          }}
        >
          当前少于两条仍可继续处理的问题，暂时不能新增联系。
          以前保存的内容仍可查看。
        </div>
      ) : (
        <>
          <div
            style={{
              display: "grid",
              gridTemplateColumns:
                "minmax(0, 170px) minmax(0, 1fr) minmax(0, 1fr)",
              gap: "7px",
              marginTop: "8px",
            }}
          >
            <label>
              <span
                style={
                  fieldLabelStyle
                }
              >
                两条问题是什么联系
              </span>

              <select
                value={relationType}
                onChange={(event) =>
                  onRelationTypeChange(
                    event.target
                      .value as
                      CWAIReviewGlobalRelationType,
                  )
                }
                disabled={busy}
                style={inputStyle}
              >
                {RELATION_TYPES.map(
                  (type) => (
                    <option
                      key={type}
                      value={type}
                    >
                      {
                        RELATION_TYPE_LABEL[
                          type
                        ]
                      }
                    </option>
                  ),
                )}
              </select>
            </label>

            <label>
              <span
                style={
                  fieldLabelStyle
                }
              >
                {directionHelp.sourceLabel}
              </span>

              <select
                value={sourceItemID}
                onChange={(event) =>
                  onSourceItemIDChange(
                    event.target.value,
                  )
                }
                disabled={busy}
                style={inputStyle}
              >
                <option value="">
                  请选择
                </option>

                {actionableItems.map(
                  (item) => (
                    <option
                      key={item.id}
                      value={item.id}
                      disabled={
                        item.id ===
                        targetItemID
                      }
                    >
                      {itemOptionLabel(
                        item,
                      )}
                    </option>
                  ),
                )}
              </select>
            </label>

            <label>
              <span
                style={
                  fieldLabelStyle
                }
              >
                {directionHelp.targetLabel}
              </span>

              <select
                value={targetItemID}
                onChange={(event) =>
                  onTargetItemIDChange(
                    event.target.value,
                  )
                }
                disabled={busy}
                style={inputStyle}
              >
                <option value="">
                  请选择
                </option>

                {actionableItems.map(
                  (item) => (
                    <option
                      key={item.id}
                      value={item.id}
                      disabled={
                        item.id ===
                        sourceItemID
                      }
                    >
                      {itemOptionLabel(
                        item,
                      )}
                    </option>
                  ),
                )}
              </select>
            </label>
          </div>

          <div
            style={{
              marginTop: "6px",
              padding: "7px 8px",
              borderRadius: "7px",
              background:
                "#F8FAFC",
              color: C.textSec,
              fontSize: "9px",
              lineHeight: 1.55,
            }}
          >
            {directionHelp.explanation}
          </div>

          <label
            style={{
              display: "block",
              marginTop: "8px",
            }}
          >
            <span
              style={
                fieldLabelStyle
              }
            >
              为什么这样判断？
            </span>

            <textarea
              value={explanation}
              onChange={(event) =>
                onExplanationChange(
                  event.target.value,
                )
              }
              rows={3}
              maxLength={
                CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
              }
              disabled={busy}
              placeholder="请写明判断依据，以及老师修改时应该先做什么"
              style={{
                ...inputStyle,
                resize: "vertical",
                lineHeight: 1.55,
              }}
            />
          </label>

          <div
            style={{
              marginTop: "3px",
              color:
                explanationLength >
                CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                  ? C.danger
                  : C.textMuted,
              fontSize: "8px",
              textAlign: "right",
            }}
          >
            {explanationLength}/
            {
              CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
            }
          </div>

          {relationAlreadyActive && (
            <div
              style={{
                marginTop: "5px",
                color: C.success,
                fontSize: "9px",
                fontWeight: 600,
              }}
            >
              这两条问题已经按同样方式说明过联系。
            </div>
          )}

          <button
            type="button"
            onClick={onConfirm}
            disabled={!canConfirm}
            style={{
              width: "100%",
              marginTop: "7px",
              padding: "7px",
              borderRadius: "7px",
              border: "none",
              background:
                canConfirm
                  ? "#7C3AED"
                  : "#CBD5E1",
              color: "#fff",
              fontSize: "10px",
              fontWeight: 700,
              cursor:
                canConfirm
                  ? "pointer"
                  : "not-allowed",
            }}
          >
            {busyKey.startsWith(
              "confirm:",
            )
              ? "正在保存…"
              : "保存这条联系"}
          </button>
        </>
      )}
    </div>
  );
}

const fieldLabelStyle:
  CSSProperties = {
    display: "block",
    marginBottom: "4px",
    color: C.textSec,
    fontSize: "9px",
    fontWeight: 700,
  };

const inputStyle:
  CSSProperties = {
    width: "100%",
    boxSizing: "border-box",
    padding: "7px 8px",
    borderRadius: "7px",
    border:
      `1px solid ${C.border}`,
    background: "#fff",
    color: C.text,
    fontFamily: "inherit",
    fontSize: "9px",
    outline: "none",
  };
