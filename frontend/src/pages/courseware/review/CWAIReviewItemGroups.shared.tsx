/**
 * CWAIReviewItemGroups.shared.tsx
 *
 * R-06正式问题组前端共享纯函数、展示文案和轻量样式。
 *
 * 本文件无请求、无业务状态。
 */

import type {
  CWAIReviewItem,
  CWAIReviewItemGroup,
} from "@/api/coursewares";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
} from "./CWAIReviewGlobalDiscussion.shared";

export function upsertCWAIReviewItemGroups(
  current: CWAIReviewItemGroup[],
  incoming: CWAIReviewItemGroup[],
): CWAIReviewItemGroup[] {
  const groupMap = new Map(
    current.map((group) => [group.id, group]),
  );

  for (const group of incoming) {
    groupMap.set(group.id, group);
  }

  return Array.from(groupMap.values()).sort((left, right) => {
    if (left.status !== right.status) {
      return left.status === "active" ? -1 : 1;
    }

    return left.name.localeCompare(right.name, "zh-CN");
  });
}

export function cwAIReviewItemGroupItemLabel(
  item: CWAIReviewItem | undefined,
): string {
  if (!item) {
    return "问题快照已不可用";
  }

  const title =
    item.teacher_title?.trim() ||
    item.title?.trim() ||
    item.improvement_goal?.trim() ||
    "未命名问题";

  return item.page_number_snapshot > 0
    ? `P${item.page_number_snapshot} · ${title}`
    : `整课 · ${title}`;
}

export function readCWAIReviewItemGroupError(
  cause: unknown,
  fallback: string,
): string {
  return cause instanceof Error ? cause.message : fallback;
}

export function cwAIReviewItemGroupEventLabel(
  eventType: string,
): string {
  switch (eventType) {
    case "created":
      return "建立问题组";
    case "renamed":
      return "重命名";
    case "primary_changed":
      return "调整主问题";
    case "member_added":
      return "加入成员";
    case "member_removed":
      return "移出成员";
    case "member_moved":
      return "移动成员";
    case "merged":
      return "合并问题组";
    case "split":
      return "拆分问题组";
    default:
      return "治理操作";
  }
}

export function CWAIReviewItemGroupFeedback({
  type,
  content,
}: {
  type: "success" | "error";
  content: string;
}) {
  return (
    <div
      style={{
        marginTop: "7px",
        padding: "6px 8px",
        borderRadius: "6px",
        background:
          type === "success"
            ? C.successSoft
            : C.dangerSoft,
        color:
          type === "success"
            ? C.success
            : C.danger,
        fontSize: "8px",
        fontWeight: 600,
        lineHeight: 1.5,
      }}
    >
      {content}
    </div>
  );
}

export const cwAIReviewItemGroupInputStyle = {
  width: "100%",
  boxSizing: "border-box",
  marginTop: "5px",
  padding: "6px 7px",
  borderRadius: "6px",
  border: `1px solid ${C.border}`,
  background: "#FFFFFF",
  color: C.text,
  fontFamily: "inherit",
  fontSize: "8px",
} as const;

export const cwAIReviewItemGroupSecondaryButtonStyle = {
  padding: "5px 8px",
  borderRadius: "6px",
  border: `1px solid ${C.border}`,
  background: "#FFFFFF",
  color: C.textSec,
  fontSize: "8px",
  fontWeight: 600,
  cursor: "pointer",
} as const;

export const cwAIReviewItemGroupDangerButtonStyle = {
  ...cwAIReviewItemGroupSecondaryButtonStyle,
  color: C.danger,
  border: `1px solid ${C.danger}55`,
} as const;

export const cwAIReviewItemGroupSectionTitleStyle = {
  color: C.text,
  fontSize: "9px",
  fontWeight: 700,
} as const;

export function cwAIReviewItemGroupPrimaryButtonStyle(
  disabled: boolean,
) {
  return {
    width: "100%",
    marginTop: "5px",
    padding: "6px 8px",
    borderRadius: "6px",
    border: `1px solid ${C.primary}`,
    background: disabled ? "#F1F5F9" : C.primary,
    color: disabled ? C.textMuted : "#FFFFFF",
    fontSize: "8px",
    fontWeight: 700,
    cursor: disabled ? "not-allowed" : "pointer",
  } as const;
}
