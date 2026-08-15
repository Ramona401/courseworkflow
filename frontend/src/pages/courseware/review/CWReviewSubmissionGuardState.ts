/**
 * CWReviewSubmissionGuardState.ts
 *
 * 正式审核复审就绪度的跨组件共享状态。
 *
 * 本文件只负责：
 *   - 保存当前复审阻断快照；
 *   - 通过useSyncExternalStore向审核决定区发布变化；
 *   - 提供“打开上轮修改复审”跨组件事件。
 *
 * 不渲染UI、不修改整改项、不调用提交接口。
 */

import {
  useSyncExternalStore,
} from "react";

export interface CWReviewApprovalGuardSnapshot {
  totalCount: number;
  resolvedCount: number;
  blockedCount: number;
  notReadyCount: number;
  waitingReviewCount: number;
  changedPageCount: number;
}

export const EMPTY_CW_REVIEW_APPROVAL_GUARD:
  CWReviewApprovalGuardSnapshot = {
    totalCount: 0,
    resolvedCount: 0,
    blockedCount: 0,
    notReadyCount: 0,
    waitingReviewCount: 0,
    changedPageCount: 0,
  };

export const CW_REVIEW_OPEN_CARRYOVER_EVENT =
  "tedna:cw-review-open-carryover";

let currentApprovalGuard =
  EMPTY_CW_REVIEW_APPROVAL_GUARD;

const approvalGuardListeners =
  new Set<() => void>();

function subscribeApprovalGuard(
  listener: () => void,
): () => void {
  approvalGuardListeners.add(listener);

  return () => {
    approvalGuardListeners.delete(listener);
  };
}

export function publishCWReviewApprovalGuard(
  snapshot: CWReviewApprovalGuardSnapshot,
): void {
  currentApprovalGuard = snapshot;

  for (const listener of approvalGuardListeners) {
    listener();
  }
}

export function resetCWReviewApprovalGuard():
  void {
  publishCWReviewApprovalGuard(
    EMPTY_CW_REVIEW_APPROVAL_GUARD,
  );
}

export function useCWReviewApprovalGuard():
  CWReviewApprovalGuardSnapshot {
  return useSyncExternalStore(
    subscribeApprovalGuard,
    () => currentApprovalGuard,
    () => EMPTY_CW_REVIEW_APPROVAL_GUARD,
  );
}

export function requestCWReviewCarryoverFocus():
  void {
  window.dispatchEvent(
    new CustomEvent(
      CW_REVIEW_OPEN_CARRYOVER_EVENT,
    ),
  );
}
