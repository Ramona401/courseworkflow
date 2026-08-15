/**
 * cwReviewDeliveryState.ts
 *
 * 正式课件审核“本次修改清单”的轻量页面内状态桥。
 *
 * 作用：
 *   - 人工审核决定面板发布当前approved/revision决定；
 *   - AI问题工作台顶部稳定入口据此显示正式交付数量；
 *   - 不保存问题正文、不修改整改项状态、不参与审核提交；
 *   - 页面卸载时发布null，避免SPA切换后沿用上一课件决定。
 */

export type CWReviewDeliveryDecision =
  | "approved"
  | "revision";

export const CW_REVIEW_DELIVERY_DECISION_EVENT =
  "tedna:cw-review-delivery-decision";

let currentDecision:
  CWReviewDeliveryDecision |
  null = null;

export function readCWReviewDeliveryDecision():
  CWReviewDeliveryDecision |
  null {
  return currentDecision;
}

export function publishCWReviewDeliveryDecision(
  decision:
    | CWReviewDeliveryDecision
    | null,
): void {
  if (currentDecision === decision) {
    return;
  }

  currentDecision = decision;

  if (typeof window === "undefined") {
    return;
  }

  window.dispatchEvent(
    new CustomEvent(
      CW_REVIEW_DELIVERY_DECISION_EVENT,
      {
        detail: {
          decision,
        },
      },
    ),
  );
}

export function subscribeCWReviewDeliveryDecision(
  listener: () => void,
): () => void {
  if (typeof window === "undefined") {
    return () => undefined;
  }

  const handleDecisionChanged = () => {
    listener();
  };

  window.addEventListener(
    CW_REVIEW_DELIVERY_DECISION_EVENT,
    handleDecisionChanged,
  );

  return () => {
    window.removeEventListener(
      CW_REVIEW_DELIVERY_DECISION_EVENT,
      handleDecisionChanged,
    );
  };
}

export function openCWReviewDeliveryPreview(): void {
  if (typeof document === "undefined") {
    return;
  }

  document
    .getElementById(
      "cw-review-delivery-preview",
    )
    ?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
}

