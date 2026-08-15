/**
 * CWAIReviewItemGroups.tsx
 *
 * R-06正式问题组教师治理总容器。
 *
 * 本文件只负责：
 *   - 从问题工作台当前临时选择建立正式问题组；
 *   - 维护问题组集合；
 *   - 把单组治理下沉给CWAIReviewItemGroupCard。
 *
 * group是教师组织问题的工作集合，不替代pairwise relation，
 * 也不会改变每条整改项的稳定身份。
 */

import { useEffect, useMemo, useState } from "react";

import {
  createCWAIReviewItemGroup,
  type CWAIReviewItem,
  type CWAIReviewItemGroup,
} from "@/api/coursewares";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
} from "./CWAIReviewGlobalDiscussion.shared";

import CWAIReviewItemGroupCard from "./CWAIReviewItemGroupCard";

import {
  CWAIReviewItemGroupFeedback,
  cwAIReviewItemGroupInputStyle,
  cwAIReviewItemGroupItemLabel,
  cwAIReviewItemGroupPrimaryButtonStyle,
  cwAIReviewItemGroupSecondaryButtonStyle,
  readCWAIReviewItemGroupError,
  upsertCWAIReviewItemGroups,
} from "./CWAIReviewItemGroups.shared";

export interface CWAIReviewItemGroupsProps {
  sessionId: string;
  items: CWAIReviewItem[];
  groups: CWAIReviewItemGroup[];
  workSelectedItemIds: string[];

  onSelectPage: (pageNumber: number) => void;
  onClearWorkSelection: () => void;
  onGroupsChanged: (groups: CWAIReviewItemGroup[]) => void;
}

export default function CWAIReviewItemGroups({
  sessionId,
  items,
  groups,
  workSelectedItemIds,
  onSelectPage,
  onClearWorkSelection,
  onGroupsChanged,
}: CWAIReviewItemGroupsProps) {
  const [open, setOpen] = useState(false);
  const [createName, setCreateName] = useState("");
  const [createPrimaryItemID, setCreatePrimaryItemID] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const itemMap = useMemo(
    () => new Map(items.map((item) => [item.id, item])),
    [items],
  );

  const selectedItems = useMemo(
    () =>
      workSelectedItemIds
        .map((itemID) => itemMap.get(itemID))
        .filter((item): item is CWAIReviewItem => !!item),
    [itemMap, workSelectedItemIds],
  );

  const activeGroupedItemIDSet = useMemo(() => {
    const result = new Set<string>();

    for (const group of groups) {
      if (group.status !== "active") {
        continue;
      }

      for (const member of group.members) {
        if (member.status === "active") {
          result.add(member.item_id);
        }
      }
    }

    return result;
  }, [groups]);

  const eligibleSelectedItems = useMemo(
    () => selectedItems.filter((item) => !activeGroupedItemIDSet.has(item.id)),
    [activeGroupedItemIDSet, selectedItems],
  );

  useEffect(() => {
    if (
      createPrimaryItemID &&
      !eligibleSelectedItems.some((item) => item.id === createPrimaryItemID)
    ) {
      setCreatePrimaryItemID("");
    }
  }, [createPrimaryItemID, eligibleSelectedItems]);

  const handleCreate = async () => {
    const name = createName.trim();
    const itemIDs = eligibleSelectedItems.map((item) => item.id);

    if (!sessionId || !name || itemIDs.length === 0 || creating) {
      setError(
        itemIDs.length === 0
          ? "请先在问题工作台选择至少一条尚未进入其他有效问题组的问题"
          : "请填写教师可理解的问题组名称",
      );
      return;
    }

    setCreating(true);
    setError("");
    setMessage("");

    try {
      const created = await createCWAIReviewItemGroup(sessionId, {
        name,
        item_ids: itemIDs,
        primary_item_id: createPrimaryItemID,
      });

      onGroupsChanged(upsertCWAIReviewItemGroups(groups, [created]));

      setCreateName("");
      setCreatePrimaryItemID("");
      onClearWorkSelection();

      setMessage(`已建立问题组“${created.name}”。组内问题仍保持独立身份。`);
    } catch (cause) {
      setError(readCWAIReviewItemGroupError(cause, "建立问题组失败"));
    } finally {
      setCreating(false);
    }
  };

  return (
    <section
      id="cw-ai-item-groups"
      style={{
        padding: "12px",
        borderRadius: "10px",
        border: `1px solid ${C.primary}35`,
        background: C.card,
      }}
    >
      <div
        style={{
          display: "flex",
          gap: "8px",
          alignItems: "flex-start",
        }}
      >
        <div style={{ minWidth: 0, flex: 1 }}>
          <div
            style={{
              color: C.text,
              fontSize: "12px",
              fontWeight: 700,
            }}
          >
            🗂 正式问题组
          </div>

          <div
            style={{
              marginTop: "3px",
              color: C.textSec,
              fontSize: "9px",
              lineHeight: 1.55,
            }}
          >
            按教学主题或改进目标组织问题。问题组是工作集合，不会删除、
            合并或替代每条问题本身，也不等同于“重复 / 依赖 / 冲突”等问题关系。
          </div>
        </div>

        <button
          type="button"
          onClick={() => setOpen((previous) => !previous)}
          style={cwAIReviewItemGroupSecondaryButtonStyle}
        >
          {open
            ? "收起问题组"
            : groups.length > 0
              ? `管理问题组（${groups.length}）`
              : "建立问题组"}
        </button>
      </div>

      {open && (
        <div
          style={{
            marginTop: "10px",
            paddingTop: "10px",
            borderTop: `1px solid ${C.border}`,
          }}
        >
          <div
            style={{
              padding: "8px",
              borderRadius: "7px",
              background: C.primarySoft,
            }}
          >
            <div
              style={{
                color: C.text,
                fontSize: "10px",
                fontWeight: 700,
              }}
            >
              用问题工作台当前选择建立组
            </div>

            <input
              value={createName}
              onChange={(event) => setCreateName(event.target.value)}
              disabled={creating}
              placeholder="例如：概念辨析与关键例题衔接"
              style={cwAIReviewItemGroupInputStyle}
            />

            <select
              value={createPrimaryItemID}
              onChange={(event) => setCreatePrimaryItemID(event.target.value)}
              disabled={creating || eligibleSelectedItems.length === 0}
              style={cwAIReviewItemGroupInputStyle}
            >
              <option value="">不预设主问题</option>

              {eligibleSelectedItems.map((item) => (
                <option key={item.id} value={item.id}>
                  {cwAIReviewItemGroupItemLabel(item)}
                </option>
              ))}
            </select>

            <div
              style={{
                marginTop: "5px",
                color: C.textMuted,
                fontSize: "8px",
                lineHeight: 1.5,
              }}
            >
              当前工作选择 {selectedItems.length} 条，其中可新建分组{" "}
              {eligibleSelectedItems.length} 条。
              已属于其他有效问题组的问题不会重复加入。
            </div>

            <button
              type="button"
              disabled={
                creating ||
                !createName.trim() ||
                eligibleSelectedItems.length === 0
              }
              onClick={() => void handleCreate()}
              style={cwAIReviewItemGroupPrimaryButtonStyle(
                creating ||
                  !createName.trim() ||
                  eligibleSelectedItems.length === 0,
              )}
            >
              {creating ? "正在建立…" : "建立正式问题组"}
            </button>
          </div>

          {groups.length === 0 ? (
            <div
              style={{
                marginTop: "8px",
                padding: "12px",
                textAlign: "center",
                color: C.textMuted,
                fontSize: "9px",
              }}
            >
              还没有正式问题组。
            </div>
          ) : (
            <div style={{ marginTop: "8px" }}>
              {groups.map((group) => (
                <CWAIReviewItemGroupCard
                  key={group.id}
                  sessionId={sessionId}
                  group={group}
                  allGroups={groups}
                  items={items}
                  itemMap={itemMap}
                  onSelectPage={onSelectPage}
                  onGroupsChanged={onGroupsChanged}
                />
              ))}
            </div>
          )}

          {error && (
            <CWAIReviewItemGroupFeedback type="error" content={error} />
          )}

          {message && (
            <CWAIReviewItemGroupFeedback type="success" content={message} />
          )}
        </div>
      )}
    </section>
  );
}
