/**
 * CWReviewSubmissionGuardViews.tsx
 *
 * 正式审核复审守卫发布器与作者重新提交前的就绪提示。
 *
 * 边界：
 *   - 后端仍是审核状态和提交权限的最终安全边界；
 *   - 本文件只根据已加载的整改项生成提示和阻断快照；
 *   - 不修改问题状态，不构造审核请求体，不调用提交接口；
 *   - 页面内容已变化与原页面已不存在统一提示为“需重新检查”。
 */

import {
  useEffect,
  useMemo,
  type CSSProperties,
} from "react";

import type {
  CWAIReviewItem,
  CWReviewCarryoverItem,
} from "@/api/coursewares";

import {
  resolveCWAIReviewPageChangeTeacherCopy,
} from "./CWAIReviewItemPresentation.shared";

import {
  CW_REVIEW_OPEN_CARRYOVER_EVENT,
  EMPTY_CW_REVIEW_APPROVAL_GUARD,
  publishCWReviewApprovalGuard,
  resetCWReviewApprovalGuard,
  type CWReviewApprovalGuardSnapshot,
} from "./CWReviewSubmissionGuardState";

const C = {
  success: "#059669",
  warning: "#D97706",
  danger: "#DC2626",
  primary: "#4F7BE8",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
};

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
    } else if (item.status === "applied") {
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
      items.length - resolvedCount,
    notReadyCount,
    waitingReviewCount,
    changedPageCount,
  };
}

/**
 * 向审核决定面板发布当前上轮问题复审状态。
 *
 * 同一工作台只应存在一个Publisher。
 * 卸载时必须清空快照，避免旧课件阻断状态残留。
 */
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
    publishCWReviewApprovalGuard(
      snapshot,
    );
  }, [snapshot]);

  useEffect(
    () => () => {
      resetCWReviewApprovalGuard();
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
      CW_REVIEW_OPEN_CARRYOVER_EVENT,
      handleOpenCarryover,
    );

    return () => {
      window.removeEventListener(
        CW_REVIEW_OPEN_CARRYOVER_EVENT,
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
  const pageChangeCopy =
    resolveCWAIReviewPageChangeTeacherCopy(
      item.status,
    );

  if (pageChangeCopy) {
    return (
      `${pageChangeCopy.label}，` +
      "需要人工重新检查"
    );
  }

  switch (item.status) {
    case "detected":
      return "尚未形成修改要求";

    case "discussing":
      return "修改要求仍在讨论";

    case "confirmed":
      return "尚未完成页面修改";

    case "applying":
      return "页面修改进行中";

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

/**
 * 作者正式整改重新提交前的就绪说明。
 *
 * 只统计正式审核整改项，自审项不参与正式提交阻断。
 */
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
            item.source_type === "formal",
        ),
      [items],
    );

  if (formalItems.length === 0) {
    return null;
  }

  const blockingItems =
    formalItems.filter(
      (item) =>
        !isFormalSubmissionReady(item),
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
        正式整改共{" "}
        {formalItems.length}{" "}
        条，已有{" "}
        {readyCount}{" "}
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
              value={blockingItems.length}
              tone="danger"
            />

            <Metric
              label="修改中"
              value={applyingCount}
              tone="warning"
            />

            <Metric
              label="需重新检查"
              value={changedPageCount}
              tone="danger"
            />
          </div>

          {changedPageCount > 0 && (
            <div style={pageChangeNoticeStyle}>
              其中有{" "}
              {changedPageCount}{" "}
              条因为页面内容已变化或原页面已不存在，
              需要人工重新检查当前课件后才能继续提交。
            </div>
          )}

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
              (item) => {
                const teacherTitle =
                  item.teacher_title?.trim() ||
                  item.title.trim() ||
                  item.description.trim() ||
                  "未填写问题标题";

                const canOpenPage =
                  item.page_number_snapshot > 0 &&
                  item.status !== "orphaned";

                return (
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
                      {item.page_number_snapshot > 0
                        ? `P${item.page_number_snapshot}`
                        : "整课"}
                      {" · "}
                      {teacherTitle}
                      {" · "}
                      {statusLabel(item)}
                    </span>

                    {canOpenPage && (
                      <button
                        type="button"
                        onClick={() =>
                          onSelectPage(
                            item.page_number_snapshot,
                          )
                        }
                        style={openPageButtonStyle}
                      >
                        打开页面
                      </button>
                    )}
                  </div>
                );
              },
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
                另有{" "}
                {blockingItems.length -
                  previewItems.length}{" "}
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
        background: config.background,
        color:
          value > 0
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

const pageChangeNoticeStyle:
  CSSProperties = {
    marginTop: 10,
    padding: "9px 11px",
    borderRadius: 8,
    border:
      "1px solid #FECACA",
    background: "#FFFFFF",
    color: C.danger,
    fontSize: 13,
    lineHeight: 1.6,
  };

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

/**
 * 保留引用，确保空快照的结构仍被本组件编译链验证。
 */
void EMPTY_CW_REVIEW_APPROVAL_GUARD;
