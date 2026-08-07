/**
 * 教学智能体发布管理器纯展示组件。
 *
 * 本文件只展示：
 *   - 三项课堂使用设置；
 *   - 当前正式发布状态；
 *   - 运行设置、暂停、恢复、撤销和版本历史。
 *
 * 首次发布与追加版本统一由右下角浮动操作台触发，
 * 避免教师在内容区和发布页之间寻找重复按钮。
 */

import type {
  Dispatch,
  SetStateAction,
} from "react";

import type {
  CoursewareAssistantDeploymentVersionView,
  CoursewareAssistantDeploymentView as DeploymentRecord,
} from "@/api/coursewares";

import DangerConfirmModal from "./DangerConfirmModal";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
  CoursewareAssistantSection,
} from "./CoursewareAssistantEditorShared";

import {
  CoursewareAssistantDeploymentPolicyEditor,
  type CoursewareAssistantDeploymentPolicyDraft,
} from "./CoursewareAssistantDeploymentPolicyEditor";

import {
  CoursewareAssistantCurrentDeploymentCard,
  CoursewareAssistantDeploymentNotice,
  CoursewareAssistantDeploymentVersionHistory,
  coursewareAssistantDeploymentDangerButtonStyle,
  coursewareAssistantDeploymentDetailsStyle,
  coursewareAssistantDeploymentFormatTime,
  coursewareAssistantDeploymentPrimaryButtonStyle,
  coursewareAssistantDeploymentSecondaryButtonStyle,
  coursewareAssistantDeploymentSummaryStyle,
  type CoursewareAssistantDeploymentStatusMeta,
} from "./CoursewareAssistantDeploymentShared";

import type {
  CoursewareAssistantDeploymentAction,
  CoursewareAssistantDeploymentNotice as DeploymentNoticeState,
} from "./useCoursewareAssistantDeployments";

interface CoursewareAssistantDeploymentViewProps {
  visible: boolean;
  pageTitle: string;
  internalOrigin: string;

  policy: CoursewareAssistantDeploymentPolicyDraft;
  setPolicy: Dispatch<
    SetStateAction<CoursewareAssistantDeploymentPolicyDraft>
  >;

  busy: boolean;
  loading: boolean;
  workingAction: CoursewareAssistantDeploymentAction;

  error: string;
  notice: DeploymentNoticeState | null;
  formError: string;

  hasSavedSlot: boolean;
  hasUnsavedChanges: boolean;
  slotStatus?: "active" | "disabled";

  liveDeployment: DeploymentRecord | null;
  status: CoursewareAssistantDeploymentStatusMeta | null;
  savedChangesNotPublished: boolean;

  versions: CoursewareAssistantDeploymentVersionView[];
  revokedHistory: DeploymentRecord[];

  confirmRevoke: boolean;
  setConfirmRevoke: Dispatch<SetStateAction<boolean>>;

  onLoad: () => Promise<void>;
  onUpdatePolicy: () => Promise<boolean>;
  onPause: () => Promise<unknown>;
  onResume: () => Promise<unknown>;
  onRevoke: () => Promise<boolean>;
}

export default function CoursewareAssistantDeploymentView({
  visible,
  pageTitle,
  internalOrigin,
  policy,
  setPolicy,
  busy,
  loading,
  workingAction,
  error,
  notice,
  formError,
  hasSavedSlot,
  hasUnsavedChanges,
  slotStatus,
  liveDeployment,
  status,
  savedChangesNotPublished,
  versions,
  revokedHistory,
  confirmRevoke,
  setConfirmRevoke,
  onLoad,
  onUpdatePolicy,
  onPause,
  onResume,
  onRevoke,
}: CoursewareAssistantDeploymentViewProps) {
  const C = COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <>
      <div
        aria-hidden={!visible}
        style={{
          display: visible ? "block" : "none",
        }}
      >
        <CoursewareAssistantSection
          title="设置课堂使用方式"
          description="回答三个问题即可。设置完成后，使用右下角操作台发布给学生。"
          actions={
            <button
              type="button"
              onClick={() => {
                void onLoad();
              }}
              disabled={busy}
              style={coursewareAssistantDeploymentSecondaryButtonStyle(
                busy,
              )}
            >
              {loading ? "同步中…" : "刷新状态"}
            </button>
          }
        >
          {error && (
            <CoursewareAssistantDeploymentNotice
              kind="error"
              text={error}
            />
          )}

          {notice && (
            <CoursewareAssistantDeploymentNotice
              kind={notice.kind}
              text={notice.text}
            />
          )}

          {formError && (
            <CoursewareAssistantDeploymentNotice
              kind="error"
              text={formError}
            />
          )}

          {hasUnsavedChanges && (
            <CoursewareAssistantDeploymentNotice
              kind="warning"
              text="当前还有未保存修改。发布只会读取已经保存的正式方案，请先使用操作台保存。"
            />
          )}

          {!hasSavedSlot && (
            <CoursewareAssistantDeploymentNotice
              kind="warning"
              text="当前页面还没有保存教学智能体方案。请先使用右下角操作台完成创建和保存。"
            />
          )}

          {hasSavedSlot && slotStatus !== "active" && (
            <CoursewareAssistantDeploymentNotice
              kind="warning"
              text="当前教学智能体方案已停用，需要先在互动内容中恢复启用。"
            />
          )}

          {liveDeployment && status && (
            <CoursewareAssistantCurrentDeploymentCard
              pageTitle={pageTitle}
              version={liveDeployment.current_version}
              status={status}
              turns={liveDeployment.per_session_turn_limit}
              dailyLimit={liveDeployment.daily_call_limit}
              validUntil={liveDeployment.valid_until}
            />
          )}

          {savedChangesNotPublished && (
            <CoursewareAssistantDeploymentNotice
              kind="warning"
              text="已保存方案比学生当前使用的版本更新。请点击右下角操作台的“发布给学生”。"
            />
          )}

          <CoursewareAssistantDeploymentPolicyEditor
            policy={policy}
            setPolicy={setPolicy}
            internalOrigin={internalOrigin}
            disabled={busy}
          />

          <CoursewareAssistantDeploymentNotice
            kind="info"
            text={
              liveDeployment
                ? "发布新方案统一使用右下角操作台。本区域只管理课堂使用范围和运行状态。"
                : "课堂使用设置完成后，直接点击右下角操作台的“发布给学生”。"
            }
          />

          {liveDeployment && (
            <>
              <div
                style={{
                  display: "flex",
                  justifyContent: "flex-end",
                  marginTop: 12,
                }}
              >
                <button
                  type="button"
                  onClick={() => {
                    void onUpdatePolicy();
                  }}
                  disabled={busy}
                  style={coursewareAssistantDeploymentSecondaryButtonStyle(
                    busy,
                  )}
                >
                  {workingAction === "policy"
                    ? "正在保存…"
                    : "保存课堂使用设置"}
                </button>
              </div>

              <details
                style={coursewareAssistantDeploymentDetailsStyle()}
              >
                <summary
                  style={coursewareAssistantDeploymentSummaryStyle(
                    busy,
                  )}
                >
                  更多运行管理
                </summary>

                <div
                  style={{
                    display: "flex",
                    gap: 8,
                    marginTop: 10,
                    flexWrap: "wrap",
                  }}
                >
                  {liveDeployment.status === "active" ? (
                    <button
                      type="button"
                      onClick={() => {
                        void onPause();
                      }}
                      disabled={busy}
                      style={coursewareAssistantDeploymentSecondaryButtonStyle(
                        busy,
                      )}
                    >
                      {workingAction === "pause"
                        ? "暂停中…"
                        : "暂停运行"}
                    </button>
                  ) : (
                    <button
                      type="button"
                      onClick={() => {
                        void onResume();
                      }}
                      disabled={busy}
                      style={coursewareAssistantDeploymentPrimaryButtonStyle(
                        busy,
                      )}
                    >
                      {workingAction === "resume"
                        ? "恢复中…"
                        : "恢复运行"}
                    </button>
                  )}

                  <button
                    type="button"
                    onClick={() => {
                      setConfirmRevoke(true);
                    }}
                    disabled={busy}
                    style={coursewareAssistantDeploymentDangerButtonStyle(
                      busy,
                    )}
                  >
                    永久撤销
                  </button>
                </div>
              </details>

              <details
                style={coursewareAssistantDeploymentDetailsStyle()}
              >
                <summary
                  style={coursewareAssistantDeploymentSummaryStyle(
                    false,
                  )}
                >
                  查看历史版本
                </summary>

                <CoursewareAssistantDeploymentVersionHistory
                  versions={versions}
                />

                {revokedHistory.length > 0 && (
                  <div style={{ marginTop: 12 }}>
                    <div
                      style={{
                        color: C.text,
                        fontSize: 10,
                        fontWeight: 700,
                        marginBottom: 6,
                      }}
                    >
                      历史已撤销部署
                    </div>

                    {revokedHistory.map((deployment) => (
                      <div
                        key={deployment.id}
                        style={{
                          marginBottom: 6,
                          padding: "8px 10px",
                          borderRadius: 8,
                          border: "1px solid #FECACA",
                          background: "#FEF2F2",
                          color: "#7F1D1D",
                          fontSize: 9,
                          lineHeight: 1.6,
                        }}
                      >
                        V{deployment.current_version}
                        {" · "}
                        永久撤销
                        {" · "}
                        {coursewareAssistantDeploymentFormatTime(
                          deployment.updated_at,
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </details>
            </>
          )}
        </CoursewareAssistantSection>
      </div>

      {visible && confirmRevoke && liveDeployment && (
        <DangerConfirmModal
          title="永久撤销教学智能体部署"
          message={`将永久撤销当前页面的教学智能体部署V${liveDeployment.current_version}。\n撤销后不能恢复，现有运行会话将失效；如需再次使用，必须重新创建部署。\n历史版本和运行流水仍会保留用于审计。`}
          confirmText={
            workingAction === "revoke"
              ? "撤销中…"
              : "确认永久撤销"
          }
          busy={workingAction === "revoke"}
          onCancel={() => {
            setConfirmRevoke(false);
          }}
          onConfirm={async () => {
            await onRevoke();
          }}
        />
      )}
    </>
  );
}
