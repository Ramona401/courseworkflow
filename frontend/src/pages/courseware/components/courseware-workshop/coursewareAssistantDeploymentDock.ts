/**
 * 教学智能体浮动操作台与发布管理器之间的稳定接口。
 *
 * 单独放在无UI文件中，避免浮动操作台反向依赖大型发布组件，
 * 也让主面板可以在页面切换时安全重置发布状态。
 */

import type {
  CoursewareAssistantDeploymentAction,
} from "./useCoursewareAssistantDeployments";

export interface CoursewareAssistantDeploymentController {
  /**
   * 使用当前发布策略执行首次发布或追加新版本。
   *
   * 返回true表示正式发布成功；
   * 返回false表示被校验阻断或接口执行失败。
   */
  publishCurrent: () => Promise<boolean>;

  /** 主动重新读取当前页面发布状态。 */
  refreshStatus: () => Promise<void>;
}

export interface CoursewareAssistantDeploymentDockState {
  loading: boolean;
  workingAction:
    CoursewareAssistantDeploymentAction;

  liveStatus:
    | ""
    | "active"
    | "paused";

  currentVersion: number | null;

  publishedCurrent: boolean;
  canPublish: boolean;
  blocker: string;

  noticeKind:
    | "success"
    | "info"
    | "error"
    | "warning"
    | null;

  noticeText: string;
}

export const EMPTY_COURSEWARE_ASSISTANT_DEPLOYMENT_DOCK_STATE:
  CoursewareAssistantDeploymentDockState = {
    loading: false,
    workingAction: "",
    liveStatus: "",
    currentVersion: null,
    publishedCurrent: false,
    canPublish: false,
    blocker: "正在读取发布状态",
    noticeKind: null,
    noticeText: "",
  };
