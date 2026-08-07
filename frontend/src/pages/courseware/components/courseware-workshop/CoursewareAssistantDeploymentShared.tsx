/**
 * 教学智能体发布向导的共享展示组件与样式。
 *
 * 不持有发布策略状态，不发起网络请求。
 */

import type {
  CSSProperties,
} from "react";

import type {
  CoursewareAssistantDeploymentVersionView,
} from "@/api/coursewares";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
} from "./CoursewareAssistantEditorShared";

export interface CoursewareAssistantDeploymentStatusMeta {
  label: string;
  color: string;
  background: string;
}

export function coursewareAssistantCurrentOrigin():
  string {
  return typeof window ===
    "undefined"
    ? ""
    : window.location.origin;
}

export function coursewareAssistantDeploymentFormatTime(
  value: string | null,
): string {
  if (!value) {
    return "长期有效";
  }

  const parsed =
    new Date(value);

  return Number.isFinite(
    parsed.getTime(),
  )
    ? parsed.toLocaleString(
        "zh-CN",
      )
    : "时间无效";
}

export function coursewareAssistantDeploymentStatusMeta(
  status: string,
): CoursewareAssistantDeploymentStatusMeta {
  if (status === "active") {
    return {
      label: "运行中",
      color: "#047857",
      background: "#ECFDF5",
    };
  }

  if (status === "paused") {
    return {
      label: "已暂停",
      color: "#B45309",
      background: "#FFFBEB",
    };
  }

  return {
    label: "已永久撤销",
    color: "#B91C1C",
    background: "#FEF2F2",
  };
}

export function CoursewareAssistantDeploymentNotice({
  kind,
  text,
}: {
  kind:
    | "success"
    | "info"
    | "error"
    | "warning";
  text: string;
}) {
  const style = {
    success: {
      border: "#A7F3D0",
      background: "#ECFDF5",
      color: "#047857",
    },
    info: {
      border: "#BFDBFE",
      background: "#EFF6FF",
      color: "#2563EB",
    },
    error: {
      border: "#FECACA",
      background: "#FEF2F2",
      color: "#B91C1C",
    },
    warning: {
      border: "#FDE68A",
      background: "#FFFBEB",
      color: "#92400E",
    },
  }[kind];

  return (
    <div
      style={{
        marginBottom: 10,
        padding: "9px 11px",
        borderRadius: 8,
        border:
          `1px solid ${style.border}`,
        background:
          style.background,
        color: style.color,
        fontSize: 10,
        lineHeight: 1.65,
      }}
    >
      {text}
    </div>
  );
}

export function CoursewareAssistantCurrentDeploymentCard({
  pageTitle,
  version,
  status,
  turns,
  dailyLimit,
  validUntil,
}: {
  pageTitle: string;
  version: number;
  status:
    CoursewareAssistantDeploymentStatusMeta;
  turns: number;
  dailyLimit: number;
  validUntil: string | null;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      style={{
        padding: 13,
        borderRadius: 10,
        border: `1px solid ${C.border}`,
        background: C.white,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          flexWrap: "wrap",
        }}
      >
        <strong
          style={{
            color: C.text,
            fontSize: 12,
          }}
        >
          当前发布版本 V
          {version}
        </strong>

        <span
          style={{
            padding: "2px 8px",
            borderRadius: 999,
            background:
              status.background,
            color: status.color,
            fontSize: 9,
            fontWeight: 700,
          }}
        >
          {status.label}
        </span>
      </div>

      <div
        style={{
          marginTop: 5,
          color: C.textSecondary,
          fontSize: 10,
          lineHeight: 1.65,
        }}
      >
        页面：
        {pageTitle ||
          "未命名页面"}
        {" · "}
        每人最多 {turns} 轮
        {" · "}
        每日额度 {dailyLimit} 次
      </div>

      <div
        style={{
          marginTop: 3,
          color: C.textMuted,
          fontSize: 9,
        }}
      >
        使用结束时间：
        {coursewareAssistantDeploymentFormatTime(
          validUntil,
        )}
      </div>
    </div>
  );
}

export function CoursewareAssistantDeploymentVersionHistory({
  versions,
}: {
  versions:
    CoursewareAssistantDeploymentVersionView[];
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  if (versions.length === 0) {
    return (
      <div
        style={{
          marginTop: 10,
          color: C.textMuted,
          fontSize: 9,
        }}
      >
        暂无历史版本。
      </div>
    );
  }

  return (
    <div
      style={{
        marginTop: 10,
      }}
    >
      {versions.map(
        (version) => (
          <div
            key={version.version}
            style={{
              marginBottom: 6,
              padding: "8px 10px",
              borderRadius: 8,
              border:
                `1px solid ${C.border}`,
              background: C.white,
            }}
          >
            <div
              style={{
                display: "flex",
                justifyContent:
                  "space-between",
                gap: 10,
                flexWrap: "wrap",
              }}
            >
              <strong
                style={{
                  color: C.text,
                  fontSize: 10,
                }}
              >
                V{version.version}
              </strong>

              <span
                style={{
                  color: C.textMuted,
                  fontSize: 8,
                }}
              >
                {coursewareAssistantDeploymentFormatTime(
                  version.created_at,
                )}
              </span>
            </div>

            <div
              style={{
                marginTop: 4,
                color: C.textMuted,
                fontSize: 8,
                lineHeight: 1.6,
                wordBreak:
                  "break-all",
              }}
            >
              助手{" "}
              {version
                .assistant_prompt_hash
                .slice(0, 12)}
              … · 上下文{" "}
              {version
                .context_snapshot_hash
                .slice(0, 12)}
              … · 页面{" "}
              {version
                .page_html_hash
                .slice(0, 12)}
              …
            </div>
          </div>
        ),
      )}
    </div>
  );
}

export function coursewareAssistantDeploymentDetailsStyle():
  CSSProperties {
  return {
    marginTop: 12,
    padding: "9px 10px",
    borderRadius: 8,
    border:
      "1px solid #E2E8F0",
    background: "#F8FAFC",
  };
}

export function coursewareAssistantDeploymentSummaryStyle(
  disabled: boolean,
): CSSProperties {
  return {
    color: "#64748B",
    fontSize: 10,
    fontWeight: 700,
    cursor: disabled
      ? "default"
      : "pointer",
  };
}

export function coursewareAssistantDeploymentInputStyle(
  disabled: boolean,
): CSSProperties {
  return {
    width: "100%",
    boxSizing:
      "border-box",
    padding: "8px 9px",
    borderRadius: 7,
    border:
      "1px solid #E2E8F0",
    background: disabled
      ? "#F8FAFC"
      : "#FFFFFF",
    color: "#1F2937",
    fontSize: 10,
    lineHeight: 1.5,
    fontFamily: "inherit",
    outline: "none",
  };
}

export function coursewareAssistantDeploymentPrimaryButtonStyle(
  disabled: boolean,
): CSSProperties {
  return {
    padding: "8px 14px",
    borderRadius: 8,
    border: "none",
    background: "#4F7BE8",
    color: "#FFFFFF",
    fontSize: 10,
    fontWeight: 700,
    cursor: disabled
      ? "default"
      : "pointer",
    opacity: disabled
      ? 0.45
      : 1,
  };
}

export function coursewareAssistantDeploymentSecondaryButtonStyle(
  disabled: boolean,
): CSSProperties {
  return {
    padding: "7px 12px",
    borderRadius: 8,
    border:
      "1px solid #E2E8F0",
    background: "#FFFFFF",
    color: "#64748B",
    fontSize: 10,
    fontWeight: 700,
    cursor: disabled
      ? "default"
      : "pointer",
    opacity: disabled
      ? 0.45
      : 1,
  };
}

export function coursewareAssistantDeploymentDangerButtonStyle(
  disabled: boolean,
): CSSProperties {
  return {
    padding: "7px 12px",
    borderRadius: 8,
    border:
      "1px solid #FECACA",
    background: "#FEF2F2",
    color: "#B91C1C",
    fontSize: 10,
    fontWeight: 700,
    cursor: disabled
      ? "default"
      : "pointer",
    opacity: disabled
      ? 0.45
      : 1,
  };
}
