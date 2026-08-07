/**
 * 课件工坊 Step 5 工作台Tab定义。
 *
 * 单一职责：
 *   - 固定Tab顺序和中文标签；
 *   - 集中处理作者专属和参与者受限视图；
 *   - 避免继续在超长主页面维护联合类型、数组和重复过滤规则。
 *
 * 第一阶段作者专属能力：
 *   - 教学智能体；
 *   - AI自审；
 *   - 审核整改；
 *   - 知识点漫画。
 */

export const COURSEWARE_WORKSHOP_TABS = [
  {
    key: "refine",
    label: "🛠 页面微调",
  },
  {
    key: "assistant",
    label: "🤖 教学智能体",
  },
  {
    key: "self_review",
    label: "🛡️ AI自审",
  },
  {
    key: "remediation",
    label: "📋 审核整改",
  },
  {
    key: "background",
    label: "🎨 背景",
  },
  {
    key: "font",
    label: "🔤 字体",
  },
  {
    key: "image",
    label: "🖼 图片",
  },
  {
    key: "comic",
    label: "🗯️ 知识点漫画",
  },
  {
    key: "video",
    label: "🎬 视频",
  },
  {
    key: "audio",
    label: "🎵 音频",
  },
  {
    key: "subject",
    label: "🧪 学科工具",
  },
  {
    key: "template",
    label: "💾 保存模板",
  },
  {
    key: "annotation",
    label: "💬 批注",
  },
  {
    key: "collab",
    label: "👥 集体备课",
  },
] as const;

export type CoursewareWorkshopTab =
  (typeof COURSEWARE_WORKSHOP_TABS)[number]["key"];

interface CoursewareWorkshopTabVisibility {
  isOwner: boolean;
  isParticipant: boolean;
}

const OWNER_ONLY_TABS =
  new Set<CoursewareWorkshopTab>([
    "assistant",
    "self_review",
    "remediation",
    "comic",
  ]);

const PARTICIPANT_VISIBLE_TABS =
  new Set<CoursewareWorkshopTab>([
    "refine",
    "image",
    "video",
    "audio",
    "subject",
    "annotation",
    "collab",
  ]);

/**
 * 作者专属工具不向协作参与者提供伪只读入口。
 */
export function isCoursewareWorkshopTabVisible(
  key: CoursewareWorkshopTab,
  visibility: CoursewareWorkshopTabVisibility,
): boolean {
  if (
    OWNER_ONLY_TABS.has(key) &&
    !visibility.isOwner
  ) {
    return false;
  }

  if (
    visibility.isParticipant &&
    !PARTICIPANT_VISIBLE_TABS.has(key)
  ) {
    return false;
  }

  return true;
}
