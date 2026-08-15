/**
 * CWAIReviewUnifiedIssueToolbar.tsx
 *
 * 统一问题工作台顶部区域。
 *
 * 默认状态只展示整体进度、完成项筛选和“下一条任务”入口。
 *
 * R-01.1：
 *   - 选择问题后只提供“一起分析”“说明问题联系”“取消选择”；
 *   - 工作选择不会改变正式交付清单；
 *   - 工具栏不得批量加入或移出本次修改清单；
 *   - 正式交付只能通过单条教师改进卡的主要操作完成。
 */

import {
  useEffect,
  useState,
  type CSSProperties,
} from "react";

import type {
  CWAIReviewUnifiedIssueOverview,
} from "./CWAIReviewUnifiedIssueList";

import {
  openCWReviewDeliveryPreview,
  readCWReviewDeliveryDecision,
  subscribeCWReviewDeliveryDecision,
} from "./cwReviewDeliveryState";

const C = {
  primary: "#4F7BE8",
  success: "#059669",
  warning: "#D97706",
  danger: "#DC2626",
  purple: "#7C3AED",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
};

export interface CWAIReviewUnifiedIssueToolbarProps {
  isSelfReview: boolean;
  overview: CWAIReviewUnifiedIssueOverview;
  activeRelationCount: number;
  deliverySelectedCount: number;
  workSelectedCount: number;
  canOpenDiscussion: boolean;
  canOpenRelation: boolean;
  showCompleted: boolean;
  hasNextItem: boolean;

  onContinueNext: () => void;

  onShowCompletedChange: (
    showCompleted: boolean,
  ) => void;

  onOpenGlobalDiscussion: () => void;
  onOpenDirectRelation: () => void;
  onClearWorkSelection: () => void;
}

export default function CWAIReviewUnifiedIssueToolbar({
  isSelfReview,
  overview,
  activeRelationCount,
  deliverySelectedCount,
  workSelectedCount,
  canOpenDiscussion,
  canOpenRelation,
  showCompleted,
  hasNextItem,
  onContinueNext,
  onShowCompletedChange,
  onOpenGlobalDiscussion,
  onOpenDirectRelation,
  onClearWorkSelection,
}: CWAIReviewUnifiedIssueToolbarProps) {
  const [
    deliveryDecision,
    setDeliveryDecision,
  ] = useState(
    readCWReviewDeliveryDecision,
  );

  useEffect(
    () =>
      subscribeCWReviewDeliveryDecision(
        () => {
          setDeliveryDecision(
            readCWReviewDeliveryDecision(),
          );
        },
      ),
    [],
  );

  const officialDeliveryCount =
    deliveryDecision === "revision"
      ? deliverySelectedCount
      : 0;

  return (
    <>
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: "12px",
          flexWrap: "wrap",
        }}
      >
        <div
          style={{
            minWidth: 0,
            flex: "1 1 360px",
          }}
        >
          <div
            style={{
              color: C.text,
              fontSize: "18px",
              fontWeight: 700,
              lineHeight: 1.4,
            }}
          >
            {isSelfReview
              ? "自审问题工作区"
              : "审核问题工作区"}
          </div>

          <div
            style={{
              marginTop: "5px",
              maxWidth: "75ch",
              color: C.textSec,
              fontSize: "14px",
              lineHeight: 1.6,
            }}
          >
            默认只显示未完成问题。页面分组负责定位任务，
            “选择一起分析”只决定比较范围；
            每条问题卡片的主要操作决定退回候选，
            正式“本次修改清单”还以当前人工审核决定为准。
          </div>
        </div>

        {hasNextItem && (
          <button
            type="button"
            onClick={
              onContinueNext
            }
            style={
              primaryActionStyle
            }
          >
            继续处理下一条
          </button>
        )}
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns:
            "repeat(auto-fit, minmax(92px, 1fr))",
          gap: "8px",
          marginTop: "14px",
        }}
      >
        <OverviewMetric
          label="全部问题"
          value={overview.total}
          tone="primary"
        />

        <OverviewMetric
          label="待处理"
          value={overview.pending}
          tone="warning"
        />

        <OverviewMetric
          label="修改中"
          value={overview.applying}
          tone="warning"
        />

        <OverviewMetric
          label="待确认"
          value={
            overview.waitingConfirm
          }
          tone="primary"
        />

        <OverviewMetric
          label="已完成"
          value={overview.completed}
          tone="success"
        />

        <OverviewMetric
          label="需要重新检查"
          value={overview.stale}
          tone="danger"
        />
      </div>

      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "8px",
          flexWrap: "wrap",
          marginTop: "14px",
          paddingTop: "14px",
          borderTop:
            `1px solid ${C.border}`,
        }}
      >
        <button
          type="button"
          onClick={() =>
            onShowCompletedChange(
              false,
            )
          }
          aria-pressed={
            !showCompleted
          }
          style={
            filterButtonStyle(
              !showCompleted,
              "primary",
            )
          }
        >
          仅看未完成
        </button>

        <button
          type="button"
          onClick={() =>
            onShowCompletedChange(
              true,
            )
          }
          aria-pressed={
            showCompleted
          }
          style={
            filterButtonStyle(
              showCompleted,
              "success",
            )
          }
        >
          查看已完成问题
        </button>

        {activeRelationCount >
          0 && (
          <span
            style={{
              color: C.purple,
              fontSize: "12px",
              fontWeight: 600,
            }}
          >
            已确认关系{" "}
            {activeRelationCount}
          </span>
        )}

        {!isSelfReview && (
          <button
            type="button"
            onClick={
              openCWReviewDeliveryPreview
            }
            title="打开本次正式提交的修改清单"
            style={{
              minHeight: "32px",
              padding: "5px 9px",
              borderRadius: "8px",
              border:
                `1px solid ${C.warning}`,
              background: "#FFF7ED",
              color: C.warning,
              fontSize: "12px",
              fontWeight: 700,
              cursor: "pointer",
            }}
          >
            本次修改清单{" "}
            {officialDeliveryCount} 条
          </button>
        )}
      </div>

      {workSelectedCount > 0 && (
        <div
          style={{
            position: "sticky",
            top: "8px",
            zIndex: 5,
            display: "flex",
            alignItems: "center",
            gap: "8px",
            flexWrap: "wrap",
            marginTop: "12px",
            padding: "10px",
            borderRadius: "9px",
            border:
              "1px solid #BFDBFE",
            background:
              "rgba(248, 250, 252, 0.98)",
            boxShadow:
              "0 6px 18px rgba(30, 64, 175, 0.08)",
          }}
        >
          <div
            style={{
              minWidth: 0,
              flex: "1 1 220px",
            }}
          >
            <div
              style={{
                color: C.primary,
                fontSize: "13px",
                fontWeight: 700,
              }}
            >
              已选择{" "}
              {workSelectedCount}{" "}
              条问题一起分析
            </div>

            <div
              style={{
                marginTop: "2px",
                color: C.textMuted,
                fontSize: "12px",
                lineHeight: 1.5,
              }}
            >
              这里只决定比较和讨论范围，
              不会加入、移出或改变本次修改清单。
            </div>
          </div>

          <button
            type="button"
            onClick={
              onOpenGlobalDiscussion
            }
            disabled={
              !canOpenDiscussion
            }
            title={
              canOpenDiscussion
                ? "把这些问题放在一起，请AI帮助比较和整理"
                : "一起分析需要选择2至12条问题"
            }
            style={
              actionButtonStyle(
                canOpenDiscussion,
                "primary",
              )
            }
          >
            一起分析
          </button>

          <button
            type="button"
            onClick={
              onOpenDirectRelation
            }
            disabled={
              !canOpenRelation
            }
            title={
              canOpenRelation
                ? "说明当前两条问题之间是什么联系"
                : "说明问题联系需要恰好选择2条问题"
            }
            style={
              actionButtonStyle(
                canOpenRelation,
                "purple",
              )
            }
          >
            说明问题联系
          </button>

          <button
            type="button"
            onClick={
              onClearWorkSelection
            }
            style={
              actionButtonStyle(
                true,
                "secondary",
              )
            }
          >
            取消选择
          </button>
        </div>
      )}
    </>
  );
}

function OverviewMetric({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone:
    | "primary"
    | "success"
    | "warning"
    | "danger";
}) {
  const config = {
    primary: {
      color: C.primary,
      background: "#EEF2FF",
    },

    success: {
      color: C.success,
      background: "#ECFDF5",
    },

    warning: {
      color: C.warning,
      background: "#FFF7ED",
    },

    danger: {
      color: C.danger,
      background: "#FEF2F2",
    },
  }[tone];

  const active =
    value > 0 ||
    label === "全部问题";

  return (
    <div
      style={{
        minWidth: "82px",
        padding: "8px 10px",
        borderRadius: "9px",

        background:
          active
            ? config.background
            : "#F8FAFC",

        color:
          active
            ? config.color
            : C.textMuted,
      }}
    >
      <div
        style={{
          fontSize: "20px",
          fontWeight: 700,
          lineHeight: 1.2,
        }}
      >
        {value}
      </div>

      <div
        style={{
          marginTop: "3px",
          fontSize: "12px",
          fontWeight: 600,
          lineHeight: 1.4,
        }}
      >
        {label}
      </div>
    </div>
  );
}

function filterButtonStyle(
  active: boolean,
  tone:
    | "primary"
    | "success",
): CSSProperties {
  const config = {
    primary: {
      color: C.primary,
      background: "#EEF2FF",
    },

    success: {
      color: C.success,
      background: "#ECFDF5",
    },
  }[tone];

  return {
    minHeight: "36px",
    padding: "8px 12px",
    borderRadius: "8px",

    border:
      `1px solid ${
        active
          ? config.color
          : C.border
      }`,

    background:
      active
        ? config.background
        : "#FFFFFF",

    color:
      active
        ? config.color
        : C.textSec,

    fontSize: "13px",
    fontWeight: 700,
    cursor: "pointer",
  };
}

function actionButtonStyle(
  enabled: boolean,
  tone:
    | "primary"
    | "purple"
    | "secondary",
): CSSProperties {
  const enabledConfig = {
    primary: {
      color: "#FFFFFF",
      background: C.primary,
      border: C.primary,
    },

    purple: {
      color: "#FFFFFF",
      background: C.purple,
      border: C.purple,
    },

    secondary: {
      color: C.textSec,
      background: "#FFFFFF",
      border: C.border,
    },
  }[tone];

  return {
    minHeight: "36px",
    padding: "8px 10px",
    borderRadius: "8px",

    border:
      `1px solid ${
        enabled
          ? enabledConfig.border
          : "#CBD5E1"
      }`,

    background:
      enabled
        ? enabledConfig.background
        : "#F1F5F9",

    color:
      enabled
        ? enabledConfig.color
        : C.textMuted,

    fontSize: "12px",
    fontWeight: 700,

    cursor:
      enabled
        ? "pointer"
        : "not-allowed",
  };
}

const primaryActionStyle:
  CSSProperties = {
    minHeight: "36px",
    padding: "8px 12px",
    borderRadius: "8px",
    border:
      `1px solid ${C.primary}`,
    background: C.primary,
    color: "#FFFFFF",
    fontSize: "13px",
    fontWeight: 700,
    cursor: "pointer",
  };
