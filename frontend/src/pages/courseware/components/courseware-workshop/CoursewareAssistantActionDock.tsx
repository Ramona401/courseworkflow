/**
 * 页面教学智能体统一浮动操作台。
 *
 * 创建、保存、发布和更新页面始终固定在视线范围内。
 * 每个动作直接展示当前进度和阻塞原因。
 */

import {
  useState,
  type CSSProperties,
} from "react";

import type {
  CoursewareAssistantSelectedPage,
} from "./coursewareAssistantSelection";

import type {
  CoursewareAssistantDeploymentDockState,
} from "./coursewareAssistantDeploymentDock";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
} from "./CoursewareAssistantEditorShared";

interface CoursewareAssistantActionDockProps {
  selectedPage: CoursewareAssistantSelectedPage | null;

  hasGeneratedContent: boolean;
  hasSavedSlot: boolean;
  isDirty: boolean;
  generateBlocker: string;

  loading: boolean;
  generating: boolean;
  saving: boolean;
  deleting: boolean;

  deployment: CoursewareAssistantDeploymentDockState;

  onGenerate: () => void;
  onSave: () => void;
  onPublish: () => Promise<void>;
  onOpenDelivery: () => void;
}

const CURRENT_BUILD_ID = String(
  import.meta.env.VITE_TEDNA_BUILD_ID || "",
).trim();

export default function CoursewareAssistantActionDock({
  selectedPage,
  hasGeneratedContent,
  hasSavedSlot,
  isDirty,
  generateBlocker,
  loading,
  generating,
  saving,
  deleting,
  deployment,
  onGenerate,
  onSave,
  onPublish,
  onOpenDelivery,
}: CoursewareAssistantActionDockProps) {
  const C = COURSEWARE_ASSISTANT_EDITOR_COLORS;
  const [refreshing, setRefreshing] = useState(false);

  const editorBusy =
    loading ||
    generating ||
    saving ||
    deleting;

  const deploymentBusy =
    deployment.loading ||
    Boolean(deployment.workingAction);

  const generatedComplete = hasGeneratedContent;
  const savedComplete = hasSavedSlot && !isDirty;
  const publishedComplete = deployment.publishedCurrent;

  const createDisabled =
    !selectedPage ||
    Boolean(generateBlocker) ||
    editorBusy ||
    deploymentBusy;

  const saveDisabled =
    !selectedPage ||
    !hasGeneratedContent ||
    editorBusy ||
    deploymentBusy;

  const publishDisabled =
    !selectedPage ||
    editorBusy ||
    !deployment.canPublish;

  const refreshDisabled =
    refreshing ||
    editorBusy ||
    deploymentBusy;

  const createStatus = generating
    ? "正在根据当前页面创建并检查"
    : generateBlocker
      ? generateBlocker
      : hasGeneratedContent
        ? "已有方案，可重新创建"
        : "先为当前页面创建学生活动";

  const saveStatus = saving
    ? "正在保存到当前页面"
    : !hasGeneratedContent
      ? "创建方案后即可保存"
      : isDirty
        ? "当前修改尚未保存"
        : hasSavedSlot
          ? "当前方案已经保存"
          : "方案尚未保存";

  const publishStatus = deployment.workingAction
    ? deployment.workingAction === "publish"
      ? "正在首次发布"
      : deployment.workingAction === "version"
        ? "正在发布新版本"
        : "正在处理发布状态"
    : deployment.publishedCurrent
      ? `当前已发布${
          deployment.currentVersion
            ? ` V${deployment.currentVersion}`
            : ""
        }`
      : deployment.blocker || "保存后即可发布给学生";

  const requestHardRefresh = async () => {
    if (refreshDisabled) {
      return;
    }

    if (
      isDirty &&
      !window.confirm(
        "当前教学智能体还有未保存修改。更新页面可能丢失这些修改，确定继续吗？",
      )
    ) {
      return;
    }

    setRefreshing(true);
    let targetBuildID = CURRENT_BUILD_ID;

    try {
      const response = await fetch(
        `/version.json?manual_refresh=${Date.now()}`,
        {
          method: "GET",
          cache: "no-store",
          credentials: "same-origin",
          headers: {
            "Cache-Control":
              "no-cache, no-store, must-revalidate",
            Pragma: "no-cache",
          },
        },
      );

      if (response.ok) {
        const payload = await response.json() as {
          build_id?: unknown;
        };

        if (
          typeof payload.build_id === "string" &&
          payload.build_id.trim()
        ) {
          targetBuildID = payload.build_id.trim();
        }
      }
    } catch {
      // 版本清单暂不可用时仍继续执行教师主动刷新。
    }

    try {
      const targetURL = new URL(window.location.href);

      if (targetBuildID) {
        targetURL.searchParams.set(
          "__tedna_build",
          targetBuildID,
        );
      }

      targetURL.searchParams.set(
        "__tedna_refresh",
        String(Date.now()),
      );

      window.location.replace(targetURL.toString());
    } catch {
      window.location.reload();
    }
  };

  const mainNotice =
    deployment.noticeText ||
    generateBlocker ||
    (
      isDirty
        ? "当前方案有未保存修改。发布只会读取已经保存的版本。"
        : publishedComplete
          ? "当前页面的教学智能体已完成创建、保存和发布。"
          : "按照左到右的顺序完成创建、保存和发布。"
    );

  const noticeIsError =
    deployment.noticeKind === "error" ||
    Boolean(generateBlocker);

  return (
    <aside
      aria-label="教学智能体操作台"
      style={{
        position: "fixed",
        right: 20,
        bottom: 20,
        zIndex: 1200,
        width: "min(760px, calc(100vw - 40px))",
        maxHeight: "calc(100vh - 40px)",
        overflowY: "auto",
        boxSizing: "border-box",
        padding: 14,
        borderRadius: 15,
        border: "1px solid rgba(79,123,232,0.30)",
        background: "rgba(255,255,255,0.97)",
        boxShadow: "0 18px 48px rgba(15,23,42,0.22)",
        backdropFilter: "blur(12px)",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          justifyContent: "space-between",
          gap: 12,
          marginBottom: 11,
          flexWrap: "wrap",
        }}
      >
        <div style={{ minWidth: 0, flex: 1 }}>
          <div
            style={{
              color: C.text,
              fontSize: 14,
              fontWeight: 800,
            }}
          >
            教学智能体操作台
          </div>

          <div
            style={{
              marginTop: 3,
              color: C.textSecondary,
              fontSize: 10,
              lineHeight: 1.55,
            }}
          >
            {selectedPage
              ? `当前页面：P${selectedPage.pageNumber} · ${
                  selectedPage.pageTitle || "未命名页面"
                }`
              : "请先选择一个具有稳定页面编号的课件页面"}
          </div>
        </div>

        <div
          style={{
            display: "flex",
            gap: 5,
            flexWrap: "wrap",
          }}
        >
          <StepBadge
            number="1"
            label="创建"
            complete={generatedComplete}
            active={!generatedComplete}
          />

          <StepBadge
            number="2"
            label="保存"
            complete={savedComplete}
            active={generatedComplete && !savedComplete}
          />

          <StepBadge
            number="3"
            label="发布"
            complete={publishedComplete}
            active={savedComplete && !publishedComplete}
          />
        </div>
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns:
            "repeat(auto-fit, minmax(128px, 1fr))",
          gap: 8,
        }}
      >
        <DockActionButton
          number="1"
          title={
            generating
              ? "正在创建…"
              : hasGeneratedContent
                ? "重新创建活动方案"
                : "创建学生活动方案"
          }
          description={createStatus}
          disabled={createDisabled}
          complete={generatedComplete}
          primary={!generatedComplete}
          onClick={onGenerate}
        />

        <DockActionButton
          number="2"
          title={saving ? "正在保存…" : "保存方案"}
          description={saveStatus}
          disabled={saveDisabled}
          complete={savedComplete}
          primary={generatedComplete && !savedComplete}
          onClick={onSave}
        />

        <DockActionButton
          number="3"
          title={
            deployment.workingAction === "publish"
              ? "正在发布…"
              : deployment.workingAction === "version"
                ? "正在发布新版本…"
                : deployment.publishedCurrent
                  ? "已经发布给学生"
                  : "发布给学生"
          }
          description={publishStatus}
          disabled={publishDisabled}
          complete={publishedComplete}
          primary={savedComplete && !publishedComplete}
          onClick={() => {
            void onPublish();
          }}
        />

        <DockActionButton
          number="↻"
          title={refreshing ? "正在更新页面…" : "更新页面"}
          description={
            refreshDisabled && !refreshing
              ? "请等待当前操作完成后再更新页面"
              : "重新读取服务器最新版本，相当于一次安全硬刷新"
          }
          disabled={refreshDisabled}
          complete={false}
          primary={false}
          onClick={() => {
            void requestHardRefresh();
          }}
        />
      </div>

      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: 10,
          marginTop: 10,
          paddingTop: 9,
          borderTop: `1px solid ${C.border}`,
          flexWrap: "wrap",
        }}
      >
        <div
          style={{
            minWidth: 0,
            flex: 1,
            color: noticeIsError
              ? C.danger
              : deployment.noticeKind === "success"
                ? C.success
                : isDirty
                  ? C.warning
                  : C.textSecondary,
            fontSize: 10,
            fontWeight: 700,
            lineHeight: 1.55,
          }}
        >
          {mainNotice}
        </div>

        <button
          type="button"
          onClick={onOpenDelivery}
          disabled={!selectedPage}
          style={{
            padding: "6px 10px",
            borderRadius: 7,
            border: `1px solid ${C.border}`,
            background: C.white,
            color: C.primary,
            fontSize: 10,
            fontWeight: 700,
            cursor: selectedPage ? "pointer" : "default",
            opacity: selectedPage ? 1 : 0.45,
          }}
        >
          查看发布设置与学生试用
        </button>
      </div>
    </aside>
  );
}

function StepBadge({
  number,
  label,
  complete,
  active,
}: {
  number: string;
  label: string;
  complete: boolean;
  active: boolean;
}) {
  const C = COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const color = complete
    ? C.success
    : active
      ? C.primary
      : C.textMuted;

  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 4,
        padding: "4px 7px",
        borderRadius: 999,
        border: `1px solid ${color}33`,
        background: `${color}12`,
        color,
        fontSize: 9,
        fontWeight: 800,
      }}
    >
      <span>{complete ? "✓" : number}</span>
      {label}
    </span>
  );
}

function DockActionButton({
  number,
  title,
  description,
  disabled,
  complete,
  primary,
  onClick,
}: {
  number: string;
  title: string;
  description: string;
  disabled: boolean;
  complete: boolean;
  primary: boolean;
  onClick: () => void;
}) {
  const C = COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const background = primary
    ? C.primary
    : complete
      ? "#ECFDF5"
      : C.white;

  const foreground = primary
    ? "#FFFFFF"
    : complete
      ? C.success
      : C.text;

  const descriptionColor = primary
    ? "rgba(255,255,255,0.82)"
    : complete
      ? "#047857"
      : C.textSecondary;

  const style: CSSProperties = {
    minHeight: 84,
    padding: "10px 11px",
    borderRadius: 11,
    border: primary
      ? `1px solid ${C.primary}`
      : complete
        ? "1px solid #A7F3D0"
        : `1px solid ${C.border}`,
    background,
    color: foreground,
    cursor: disabled ? "default" : "pointer",
    opacity: disabled ? 0.48 : 1,
    textAlign: "left",
    boxShadow:
      primary && !disabled
        ? "0 8px 20px rgba(79,123,232,0.24)"
        : "none",
  };

  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      style={style}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 7,
        }}
      >
        <span
          style={{
            display: "inline-flex",
            width: 22,
            height: 22,
            alignItems: "center",
            justifyContent: "center",
            borderRadius: 999,
            background: primary
              ? "rgba(255,255,255,0.18)"
              : complete
                ? "rgba(5,150,105,0.12)"
                : C.background,
            fontSize: 10,
            fontWeight: 800,
          }}
        >
          {complete ? "✓" : number}
        </span>

        <strong
          style={{
            minWidth: 0,
            fontSize: 11,
            lineHeight: 1.35,
          }}
        >
          {title}
        </strong>
      </div>

      <div
        style={{
          marginTop: 7,
          color: descriptionColor,
          fontSize: 9,
          lineHeight: 1.5,
        }}
      >
        {description}
      </div>
    </button>
  );
}
