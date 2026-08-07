/**
 * CWAIReviewDirectRelationGovernance.shared.tsx
 *
 * 问题关系治理容器共享的纯函数、反馈组件和样式。
 *
 * 本文件不保存状态、不发起请求。
 */

import type {
  CSSProperties,
} from "react";

import type {
  CWAIReviewGlobalRelationType,
  CWAIReviewItemRelation,
} from "@/api/coursewares";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
} from "./CWAIReviewGlobalDiscussion.shared";

export function buildCWAIReviewDirectRelationKey(
  relationType:
    CWAIReviewGlobalRelationType,
  sourceItemID: string,
  targetItemID: string,
): string {
  let source =
    sourceItemID.trim();

  let target =
    targetItemID.trim();

  if (
    relationType === "conflict" &&
    source > target
  ) {
    [
      source,
      target,
    ] = [
      target,
      source,
    ];
  }

  return [
    relationType,
    source,
    target,
  ].join("|");
}

export function mergeCWAIReviewDirectRelation(
  relations:
    CWAIReviewItemRelation[],
  incoming:
    CWAIReviewItemRelation,
): CWAIReviewItemRelation[] {
  const next =
    relations.filter(
      (relation) =>
        relation.id !==
        incoming.id,
    );

  next.push(incoming);

  return next.sort(
    (left, right) => {
      if (
        left.status !==
        right.status
      ) {
        return left.status ===
          "active"
          ? -1
          : 1;
      }

      return (
        right.updated_at || ""
      ).localeCompare(
        left.updated_at || "",
      );
    },
  );
}

export function CWAIReviewDirectRelationFeedback({
  type,
  content,
}: {
  type:
    | "success"
    | "error";
  content: string;
}) {
  return (
    <div
      style={{
        marginTop: "8px",
        padding: "7px 9px",
        borderRadius: "7px",
        background:
          type === "success"
            ? C.successSoft
            : C.dangerSoft,
        color:
          type === "success"
            ? C.success
            : C.danger,
        fontSize: "9px",
        fontWeight: 600,
        lineHeight: 1.5,
      }}
    >
      {content}
    </div>
  );
}

export const cwAIReviewDirectRelationSecondaryButtonStyle:
  CSSProperties = {
    padding: "5px 9px",
    borderRadius: "6px",
    border:
      "1px solid #C4B5FD",
    background: "#fff",
    color: "#7C3AED",
    fontSize: "9px",
    fontWeight: 700,
    cursor: "pointer",
  };

export const cwAIReviewDirectRelationSummaryBadgeStyle:
  CSSProperties = {
    padding: "3px 7px",
    borderRadius: "6px",
    background: "#F5F3FF",
    color: "#7C3AED",
    fontSize: "9px",
    fontWeight: 700,
  };
