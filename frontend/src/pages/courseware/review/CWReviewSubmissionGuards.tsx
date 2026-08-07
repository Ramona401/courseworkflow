/**
 * CWReviewSubmissionGuards.tsx
 *
 * 课件正式审核和作者重新提交的前端就绪提示。
 *
 * 边界：
 *   - 后端仍是审核状态和提交权限的最终安全边界；
 *   - 本文件只根据已加载的历史整改项生成前端提示和按钮禁用状态；
 *   - 不修改问题状态，不构造审核请求体，不调用提交接口；
 *   - 正式审核工作台使用共享快照，避免继续扩张超过900行的主编排组件。
 */

import {
  useEffect,
  useMemo,
  useSyncExternalStore,
  type CSSProperties,
} from "react";

import type {
  CWAIReviewItem,
  CWReviewCarryoverItem,
} from "@/api/coursewares";

const OPEN_CARRYOVER_EVENT =
  "tedna:cw-review-open-carryover";

const C = {
  success: "#059669",
  warning: "#D97706",
  danger: "#DC2626",
  primary: "#4F7BE8",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
};

export interface CWReviewApprovalGuardSnapshot {
  totalCount: number;
  resolvedCount: number;
  blockedCount: number;
  notReadyCount: number;
  waitingReviewCount: number;
  changedPageCount: number;
}

const EMPTY_APPROVAL_GUARD:
  CWReviewApprovalGuardSnapshot = {
  totalCount: 0,
  resolvedCount: 0,
  blockedCount: 0,
  notReadyCount: 0,
  waitingReviewCount: 0,
  changedPageCount: 0,
};

let currentApprovalGuard =
  EMPTY_APPROVAL_GUARD;

const approvalGuardListeners =
  new Set<() => void>();

function subscribeApprovalGuard(
  listener: () => void,
): () => void {
  approvalGuardListeners.add(
    listener,
  );

  return () => {
    approvalGuardListeners.delete(
      listener,
    );
  };
}

function publishApprovalGuard(
  snapshot:
    CWReviewApprovalGuardSnapshot,
): void {
  currentApprovalGuard =
    snapshot;

  for (const listener of
    approvalGuardListeners) {
    listener();
  }
}

function buildApprovalGuard(
  items: CWReviewCarryoverItem[],
  resolvedItemIds: string[],
): CWReviewApprovalGuardSnapshot {
  const resolvedIDSet =
    new Set(
      resolvedItemIds
        .map((value) => value.trim())
        .filter(Boolean),
    );

  let resolvedCount = 0;
  let notReadyCount = 0;
  let waitingReviewCount = 0;
  let changedPageCount = 0;

  for (const item of items) {
    const resolved =
      item.status === "applied" &&
      resolvedIDSet.has(item.id);

    if (resolved) {
      resolvedCount += 1;
    } else if (
      item.status === "applied"
    ) {
      waitingReviewCount += 1;
    } else {
      notReadyCount += 1;
    }

    if (
      item.status === "stale" ||
      item.status === "orphaned"
    ) {
      changedPageCount += 1;
    }
  }

  return {
    totalCount: items.length,
    resolvedCount,
    blockedCount:
      items.length -
      resolvedCount,
    notReadyCount,
    waitingReviewCount,
    changedPageCount,
  };
}

export function useCWReviewApprovalGuard():
  CWReviewApprovalGuardSnapshot {
  return useSyncExternalStore(
    subscribeApprovalGuard,
    () => currentApprovalGuard,
    () => EMPTY_APPROVAL_GUARD,
  );
}

export function requestCWReviewCarryoverFocus():
  void {
  window.dispatchEvent(
    new CustomEvent(
      OPEN_CARRYOVER_EVENT,
    ),
  );
}

export function CWReviewDecisionGuardPublisher({
  items,
  resolvedItemIds,
  onOpenCarryover,
}: {
  items: CWReviewCarryoverItem[];
  resolvedItemIds: string[];
  onOpenCarryover: () => void;
}) {
  const snapshot =
    useMemo(
      () =>
        buildApprovalGuard(
          items,
          resolvedItemIds,
        ),
      [
        items,
        resolvedItemIds,
      ],
    );

  useEffect(() => {
    publishApprovalGuard(
      snapshot,
    );
  }, [snapshot]);

  useEffect(
    () => () => {
      publishApprovalGuard(
        EMPTY_APPROVAL_GUARD,
      );
    },
    [],
  );

  useEffect(() => {
    const handleOpenCarryover =
      () => {
        if (items.length > 0) {
          onOpenCarryover();
        }
      };

    window.addEventListener(
      OPEN_CARRYOVER_EVENT,
      handleOpenCarryover,
    );

    return () => {
      window.removeEventListener(
        OPEN_CARRYOVER_EVENT,
        handleOpenCarryover,
      );
    };
  }, [
    items.length,
    onOpenCarryover,
  ]);

  return null;
}

function isFormalSubmissionReady(
  item: CWAIReviewItem,
): boolean {
  return (
    item.status === "applied" ||
    item.status === "resolved" ||
    item.status === "dismissed"
  );
}

function statusLabel(
  item: CWAIReviewItem,
): string {
  switch (item.status) {
    case "detected":
      return "尚未形成修改要求";
    case "discussing":
      return "修改要求仍在讨论";
    case "confirmed":
      return "尚未完成页面修改";
    case "applying":
      return "页面修改进行中";
    case "stale":
      return "页面修改后又发生变化";
    case "orphaned":
      return "原问题页面已删除";
    case "applied":
      return "已完成并等待复审";
    case "resolved":
      return "已确认解决";
    case "dismissed":
      return "已记录暂不处理";
    default:
      return "等待继续处理";
  }
}

export function CWOwnerReviewSubmissionReadiness({
  items,
  onSelectPage,
}: {
  items: CWAIReviewItem[];
  onSelectPage: (
    pageNumber: number,
  ) => void;
}) {
  const formalItems =
    useMemo(
      () =>
        items.filter(
          (item) =>
            item.source_type ===
            "formal",
        ),
      [items],
    );

  if (formalItems.length === 0) {
    return null;
  }

  const blockingItems =
    formalItems.filter(
      (item) =>
        !isFormalSubmissionReady(
          item,
        ),
    );

  const readyCount =
    formalItems.length -
    blockingItems.length;

  const changedPageCount =
    blockingItems.filter(
      (item) =>
        item.status === "stale" ||
        item.status === "orphaned",
    ).length;

  const applyingCount =
    blockingItems.filter(
      (item) =>
        item.status === "applying",
    ).length;

  const previewItems =
    blockingItems.slice(0, 5);

  const ready =
    blockingItems.length === 0;

  return (
    <section
      aria-label="重新提交审核就绪状态"
      style={{
        marginBottom: 14,
        padding: 14,
        borderRadius: 10,
        border:
          ready
            ? "1px solid #A7F3D0"
            : "1px solid #FECACA",
        background:
          ready
            ? "#ECFDF5"
            : "#FEF2F2",
      }}
    >
      <div
        style={{
          color:
            ready
              ? C.success
              : C.danger,
          fontSize: 16,
          fontWeight: 700,
          lineHeight: 1.45,
        }}
      >
        {ready
          ? "正式整改已达到重新提交条件"
          : `重新提交前还有 ${blockingItems.length} 条正式整改未完成`}
      </div>

      <div
        style={{
          marginTop: 5,
          color: C.textSec,
          fontSize: 13,
          lineHeight: 1.65,
        }}
      >
        正式整改共
        {" "}
        {formalItems.length}
        {" "}
        条，已有
        {" "}
        {readyCount}
        {" "}
        条达到可重新提交状态。
        自审项不会计入正式提交阻断。
      </div>

      {!ready && (
        <>
          <div
            style={{
              display: "flex",
              gap: 8,
              flexWrap: "wrap",
              marginTop: 10,
            }}
          >
            <Metric
              label="未完成"
              value={
                blockingItems.length
              }
              tone="danger"
            />

            <Metric
              label="修改中"
              value={applyingCount}
              tone="warning"
            />

            <Metric
              label="页面变化"
              value={
                changedPageCount
              }
              tone="danger"
            />
          </div>

          <div
            style={{
              marginTop: 10,
              paddingTop: 10,
              borderTop:
                "1px solid #FECACA",
            }}
          >
            <div
              style={{
                color: C.text,
                fontSize: 13,
                fontWeight: 700,
              }}
            >
              当前未完成摘要
            </div>

            {previewItems.map(
              (item) => (
                <div
                  key={item.id}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 8,
                    marginTop: 7,
                    color: C.textSec,
                    fontSize: 13,
                    lineHeight: 1.55,
                  }}
                >
                  <span
                    style={{
                      minWidth: 0,
                      flex: 1,
                    }}
                  >
                    {item.page_number_snapshot >
                    0
                      ? `P${item.page_number_snapshot}`
                      : "整课"}
                    {" · "}
                    {item.title.trim() ||
                      item.description.trim() ||
                      "未填写问题标题"}
                    {" · "}
                    {statusLabel(item)}
                  </span>

                  {item.page_number_snapshot >
                    0 && (
                    <button
                      type="button"
                      onClick={() =>
                        onSelectPage(
                          item.page_number_snapshot,
                        )
                      }
                      style={
                        openPageButtonStyle
                      }
                    >
                      打开页面
                    </button>
                  )}
                </div>
              ),
            )}

            {blockingItems.length >
              previewItems.length && (
              <div
                style={{
                  marginTop: 7,
                  color: C.textMuted,
                  fontSize: 12,
                  lineHeight: 1.5,
                }}
              >
                另有
                {" "}
                {blockingItems.length -
                  previewItems.length}
                {" "}
                条未在摘要中展开，请在下方整改工作区继续处理。
              </div>
            )}
          </div>
        </>
      )}
    </section>
  );
}

function Metric({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone:
    | "warning"
    | "danger";
}) {
  const config =
    tone === "danger"
      ? {
          color: C.danger,
          background: "#FFFFFF",
        }
      : {
          color: C.warning,
          background: "#FFFFFF",
        };

  return (
    <div
      style={{
        minWidth: 86,
        padding: "7px 9px",
        borderRadius: 8,
        background:
          config.background,
        color: value > 0
          ? config.color
          : C.textMuted,
      }}
    >
      <div
        style={{
          fontSize: 18,
          fontWeight: 700,
          lineHeight: 1.2,
        }}
      >
        {value}
      </div>

      <div
        style={{
          marginTop: 2,
          fontSize: 12,
          fontWeight: 600,
        }}
      >
        {label}
      </div>
    </div>
  );
}

const openPageButtonStyle:
  CSSProperties = {
  flexShrink: 0,
  minHeight: 32,
  padding: "5px 8px",
  borderRadius: 7,
  border:
    `1px solid ${C.primary}`,
  background: "#FFFFFF",
  color: C.primary,
  fontSize: 12,
  fontWeight: 700,
  cursor: "pointer",
};
