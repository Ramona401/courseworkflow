/**
 * cwReviewDecisionPayload.typecheck.ts
 *
 * 正式审核请求的编译期契约检查。
 *
 * 覆盖四种请求：
 *
 *   - 首次审核通过，没有历史整改问题；
 *   - 首次审核退回并交付本轮新问题；
 *   - 复审通过并确认历史问题全部解决；
 *   - 复审继续退回并确认部分历史问题解决。
 */

import type {
  CWReviewDecisionRequest,
} from "@/api/coursewares";

import {
  buildCWReviewDecisionRequest,
} from "./cwReviewDecisionPayload";

const approvedRequest:
  CWReviewDecisionRequest =
  buildCWReviewDecisionRequest({
    decision: "approved",
    comment: "可以通过",
    scoreText: "9",
    aiReviewContext: {
      sessionId:
        "session-must-not-be-submitted",
      selectedItemIds: [
        "item-must-not-be-submitted",
      ],
      selectedItems: [],
    },
  });

const revisionRequest:
  CWReviewDecisionRequest =
  buildCWReviewDecisionRequest({
    decision: "revision",
    comment: "请按整改项修改",
    scoreText: "6.5",
    aiReviewContext: {
      sessionId: "session-1",
      selectedItemIds: [
        "item-1",
        "item-1",
        " item-2 ",
      ],
      selectedItems: [],
    },
  });

const carryoverApprovedRequest:
  CWReviewDecisionRequest =
  buildCWReviewDecisionRequest({
    decision: "approved",
    comment:
      "上轮问题已经逐条复查，确认均已解决",
    scoreText: "9.5",
    aiReviewContext: {
      sessionId:
        "session-must-not-be-submitted",
      selectedItemIds: [
        "new-item-must-not-be-submitted",
      ],
      selectedItems: [],
    },
    resolvedReviewItemIds: [
      "old-item-1",
      "old-item-1",
      " old-item-2 ",
    ],
  });

const carryoverRevisionRequest:
  CWReviewDecisionRequest =
  buildCWReviewDecisionRequest({
    decision: "revision",
    comment:
      "部分问题已经解决，其余问题请继续整改",
    scoreText: "",
    aiReviewContext: {
      sessionId: "session-2",
      selectedItemIds: [
        "new-item-1",
      ],
      selectedItems: [],
    },
    resolvedReviewItemIds: [
      "old-item-1",
    ],
  });

void approvedRequest;
void revisionRequest;
void carryoverApprovedRequest;
void carryoverRevisionRequest;

export {};
