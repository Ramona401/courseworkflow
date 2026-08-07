/**
 * 页面教学智能体发布状态控制器。
 *
 * 职责：
 *   - 读取当前稳定page_id的部署与版本；
 *   - 管理课堂使用策略；
 *   - 判断统一浮动操作台是否允许发布；
 *   - 暴露首次发布或追加版本控制器；
 *   - 将完整发布界面交给纯展示组件。
 */

import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useMemo,
  useState,
} from "react";

import {
  buildCoursewareAssistantDeploymentRequest,
  coursewareAssistantDeploymentPolicyFromLive,
  createDefaultCoursewareAssistantDeploymentPolicy,
  type CoursewareAssistantDeploymentPolicyDraft,
} from "./CoursewareAssistantDeploymentPolicyEditor";

import {
  coursewareAssistantCurrentOrigin,
  coursewareAssistantDeploymentStatusMeta,
} from "./CoursewareAssistantDeploymentShared";

import CoursewareAssistantDeploymentView from "./CoursewareAssistantDeploymentView";

import {
  useCoursewareAssistantDeployments,
} from "./useCoursewareAssistantDeployments";

import type {
  CoursewareAssistantDeploymentController,
  CoursewareAssistantDeploymentDockState,
} from "./coursewareAssistantDeploymentDock";

export type {
  CoursewareAssistantDeploymentController,
  CoursewareAssistantDeploymentDockState,
} from "./coursewareAssistantDeploymentDock";

interface CoursewareAssistantDeploymentManagerProps {
  coursewareId: string;
  pageId: string;
  pageTitle: string;

  hasSavedSlot: boolean;
  hasUnsavedChanges: boolean;

  slotStatus?: "active" | "disabled";
  slotUpdatedAt?: string | null;

  disabled?: boolean;
  visible?: boolean;

  onChanged: () => void;
  onStateChange?: (
    state: CoursewareAssistantDeploymentDockState,
  ) => void;
}

const CoursewareAssistantDeploymentManager =
  forwardRef<
    CoursewareAssistantDeploymentController,
    CoursewareAssistantDeploymentManagerProps
  >(function CoursewareAssistantDeploymentManager(
    {
      coursewareId,
      pageId,
      pageTitle,
      hasSavedSlot,
      hasUnsavedChanges,
      slotStatus,
      slotUpdatedAt,
      disabled = false,
      visible = true,
      onChanged,
      onStateChange,
    },
    ref,
  ) {
    const internalOrigin = useMemo(
      coursewareAssistantCurrentOrigin,
      [],
    );

    const manager = useCoursewareAssistantDeployments({
      coursewareId,
      pageId,
      onChanged,
    });

    const [policy, setPolicy] =
      useState<CoursewareAssistantDeploymentPolicyDraft>(
        createDefaultCoursewareAssistantDeploymentPolicy,
      );

    const [formError, setFormError] = useState("");
    const [confirmRevoke, setConfirmRevoke] = useState(false);

    useEffect(() => {
      setPolicy(
        createDefaultCoursewareAssistantDeploymentPolicy(),
      );
      setFormError("");
      setConfirmRevoke(false);
    }, [pageId]);

    useEffect(() => {
      setFormError("");
    }, [
      hasSavedSlot,
      hasUnsavedChanges,
      slotStatus,
    ]);

    useEffect(() => {
      if (!manager.liveDeployment) {
        return;
      }

      setPolicy(
        coursewareAssistantDeploymentPolicyFromLive(
          manager.liveDeployment,
          internalOrigin,
        ),
      );
    }, [
      internalOrigin,
      manager.liveDeployment,
    ]);

    const updatePolicyDraft: typeof setPolicy = (next) => {
      setFormError("");
      setPolicy(next);
    };

    const busy =
      disabled ||
      manager.loading ||
      Boolean(manager.workingAction);

    const status = manager.liveDeployment
      ? coursewareAssistantDeploymentStatusMeta(
          manager.liveDeployment.status,
        )
      : null;

    const savedChangesNotPublished = useMemo(() => {
      if (
        !manager.liveDeployment ||
        !manager.latestVersion ||
        !slotUpdatedAt ||
        hasUnsavedChanges
      ) {
        return false;
      }

      const slotTime = Date.parse(slotUpdatedAt);
      const versionTime = Date.parse(
        manager.latestVersion.created_at || "",
      );

      return (
        Number.isFinite(slotTime) &&
        Number.isFinite(versionTime) &&
        slotTime > versionTime
      );
    }, [
      hasUnsavedChanges,
      manager.latestVersion,
      manager.liveDeployment,
      slotUpdatedAt,
    ]);

    const resolveRequest = () => {
      const result =
        buildCoursewareAssistantDeploymentRequest(
          policy,
          internalOrigin,
        );

      setFormError(result.error);
      return result.request;
    };

    const publishFirst = async (): Promise<boolean> => {
      if (
        manager.error ||
        !hasSavedSlot ||
        hasUnsavedChanges ||
        slotStatus !== "active"
      ) {
        return false;
      }

      const request = resolveRequest();
      if (!request) {
        return false;
      }

      const result = await manager.publishFirst(request);
      return Boolean(result);
    };

    const updatePolicy = async (): Promise<boolean> => {
      if (manager.error || !manager.liveDeployment) {
        return false;
      }

      const request = resolveRequest();
      if (!request) {
        return false;
      }

      const result = await manager.updatePolicy(request);
      return Boolean(result);
    };

    const publishVersion = async (): Promise<boolean> => {
      if (
        manager.error ||
        !manager.liveDeployment ||
        !hasSavedSlot ||
        hasUnsavedChanges ||
        slotStatus !== "active"
      ) {
        return false;
      }

      const result = await manager.publishVersion();
      return Boolean(result);
    };

    const handleRevoke = async (): Promise<boolean> => {
      const result = await manager.revoke();
      const succeeded = Boolean(result);

      if (succeeded) {
        setConfirmRevoke(false);
      }

      return succeeded;
    };

    const versionMetadataReady =
      !manager.liveDeployment ||
      Boolean(manager.latestVersion);

    const publishedCurrent = Boolean(
      manager.liveDeployment?.status === "active" &&
      manager.latestVersion?.version ===
        manager.liveDeployment.current_version &&
      hasSavedSlot &&
      !hasUnsavedChanges &&
      slotStatus === "active" &&
      !savedChangesNotPublished,
    );

    const publishBlocker = (() => {
      if (manager.loading) {
        return "正在同步发布状态";
      }

      if (manager.workingAction) {
        return "正在处理发布操作";
      }

      if (manager.error) {
        return "发布状态读取失败，请先打开发布设置并刷新状态";
      }

      if (!versionMetadataReady) {
        return "当前发布版本信息不完整，请先刷新发布状态";
      }

      if (!hasSavedSlot) {
        return "请先保存方案";
      }

      if (hasUnsavedChanges) {
        return "请先保存当前修改";
      }

      if (slotStatus !== "active") {
        return "当前方案已停用，请先恢复启用";
      }

      if (manager.liveDeployment?.status === "paused") {
        return "当前发布已暂停，请在发布设置中恢复运行";
      }

      if (publishedCurrent) {
        return manager.liveDeployment
          ? `当前已是最新发布版本 V${manager.liveDeployment.current_version}`
          : "当前方案已发布";
      }

      return "";
    })();

    const canPublish = !disabled && !publishBlocker;

    const dockNotice = (() => {
      if (formError) {
        return {
          kind: "error" as const,
          text: formError,
        };
      }

      if (manager.notice) {
        return {
          kind: manager.notice.kind,
          text: manager.notice.text,
        };
      }

      if (manager.error) {
        return {
          kind: "error" as const,
          text: manager.error,
        };
      }

      return {
        kind: null,
        text: "",
      };
    })();

    useEffect(() => {
      onStateChange?.({
        loading: manager.loading,
        workingAction: manager.workingAction,
        liveStatus:
          manager.liveDeployment?.status === "active" ||
          manager.liveDeployment?.status === "paused"
            ? manager.liveDeployment.status
            : "",
        currentVersion:
          manager.liveDeployment?.current_version || null,
        publishedCurrent,
        canPublish,
        blocker: publishBlocker,
        noticeKind: dockNotice.kind,
        noticeText: dockNotice.text,
      });
    }, [
      canPublish,
      dockNotice.kind,
      dockNotice.text,
      manager.liveDeployment,
      manager.loading,
      manager.workingAction,
      onStateChange,
      publishBlocker,
      publishedCurrent,
    ]);

    const publishCurrent = async (): Promise<boolean> => {
      setFormError("");

      if (publishBlocker) {
        setFormError(publishBlocker);
        return false;
      }

      if (manager.liveDeployment) {
        return publishVersion();
      }

      return publishFirst();
    };

    useImperativeHandle(ref, () => ({
      publishCurrent,
      refreshStatus: manager.load,
    }));

    const revokedHistory = manager.pageDeployments.filter(
      (item) => item.status === "revoked",
    );

    return (
      <CoursewareAssistantDeploymentView
        visible={visible}
        pageTitle={pageTitle}
        internalOrigin={internalOrigin}
        policy={policy}
        setPolicy={updatePolicyDraft}
        busy={busy}
        loading={manager.loading}
        workingAction={manager.workingAction}
        error={manager.error}
        notice={manager.notice}
        formError={formError}
        hasSavedSlot={hasSavedSlot}
        hasUnsavedChanges={hasUnsavedChanges}
        slotStatus={slotStatus}
        liveDeployment={manager.liveDeployment}
        status={status}
        savedChangesNotPublished={savedChangesNotPublished}
        versions={manager.versions}
        revokedHistory={revokedHistory}
        confirmRevoke={confirmRevoke}
        setConfirmRevoke={setConfirmRevoke}
        onLoad={manager.load}
        onUpdatePolicy={updatePolicy}
        onPause={manager.pause}
        onResume={manager.resume}
        onRevoke={handleRevoke}
      />
    );
  });

export default CoursewareAssistantDeploymentManager;
