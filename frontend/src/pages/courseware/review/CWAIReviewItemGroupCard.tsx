/**
 * CWAIReviewItemGroupCard.tsx
 *
 * R-06单个正式问题组的基础治理与成员治理。
 *
 * 本文件负责：
 *   - 重命名；
 *   - 设置/清空主问题；
 *   - 添加未分组问题；
 *   - 移除成员；
 *   - 跨组移动成员；
 *   - 展示追加式治理历史。
 *
 * 合并与拆分下沉至CWAIReviewItemGroupStructuralActions。
 */

import { useEffect, useMemo, useState } from "react";

import {
  addCWAIReviewItemGroupMember,
  moveCWAIReviewItemGroupMember,
  removeCWAIReviewItemGroupMember,
  renameCWAIReviewItemGroup,
  setCWAIReviewItemGroupPrimary,
  type CWAIReviewItem,
  type CWAIReviewItemGroup,
  type CWAIReviewItemGroupMember,
} from "@/api/coursewares";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
  cwGlobalPageButtonStyle,
} from "./CWAIReviewGlobalDiscussion.shared";

import CWAIReviewItemGroupStructuralActions from "./CWAIReviewItemGroupStructuralActions";

import {
  CWAIReviewItemGroupFeedback,
  cwAIReviewItemGroupDangerButtonStyle,
  cwAIReviewItemGroupEventLabel,
  cwAIReviewItemGroupInputStyle,
  cwAIReviewItemGroupItemLabel,
  cwAIReviewItemGroupSecondaryButtonStyle,
  readCWAIReviewItemGroupError,
  upsertCWAIReviewItemGroups,
} from "./CWAIReviewItemGroups.shared";

export interface CWAIReviewItemGroupCardProps {
  sessionId: string;
  group: CWAIReviewItemGroup;
  allGroups: CWAIReviewItemGroup[];
  items: CWAIReviewItem[];
  itemMap: Map<string, CWAIReviewItem>;

  onSelectPage: (pageNumber: number) => void;
  onGroupsChanged: (groups: CWAIReviewItemGroup[]) => void;
}

export default function CWAIReviewItemGroupCard({
  sessionId,
  group,
  allGroups,
  items,
  itemMap,
  onSelectPage,
  onGroupsChanged,
}: CWAIReviewItemGroupCardProps) {
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const [renameName, setRenameName] = useState(group.name);
  const [primaryItemID, setPrimaryItemID] = useState(
    group.primary_item_id || "",
  );
  const [addItemID, setAddItemID] = useState("");

  const [memberReasons, setMemberReasons] = useState<Record<string, string>>(
    {},
  );
  const [moveTargets, setMoveTargets] = useState<Record<string, string>>({});

  useEffect(() => {
    setRenameName(group.name);
    setPrimaryItemID(group.primary_item_id || "");
  }, [group.id, group.name, group.primary_item_id]);

  const activeMembers = useMemo(
    () => group.members.filter((member) => member.status === "active"),
    [group.members],
  );

  const activeGroups = useMemo(
    () => allGroups.filter((candidate) => candidate.status === "active"),
    [allGroups],
  );

  const otherActiveGroups = useMemo(
    () => activeGroups.filter((candidate) => candidate.id !== group.id),
    [activeGroups, group.id],
  );

  const activeGroupedItemIDs = useMemo(() => {
    const result = new Set<string>();

    for (const candidate of activeGroups) {
      for (const member of candidate.members) {
        if (member.status === "active") {
          result.add(member.item_id);
        }
      }
    }

    return result;
  }, [activeGroups]);

  const addableItems = useMemo(
    () => items.filter((item) => !activeGroupedItemIDs.has(item.id)),
    [activeGroupedItemIDs, items],
  );

  const merged = group.status === "merged";

  const patchGroups = (incoming: CWAIReviewItemGroup[]) => {
    onGroupsChanged(upsertCWAIReviewItemGroups(allGroups, incoming));
  };

  const runAction = async (
    key: string,
    action: () => Promise<void>,
  ) => {
    if (busy || merged) {
      return;
    }

    setBusy(key);
    setError("");
    setMessage("");

    try {
      await action();
    } catch (cause) {
      setError(readCWAIReviewItemGroupError(cause, "问题组操作失败"));
    } finally {
      setBusy("");
    }
  };

  const handleRename = () =>
    runAction("rename", async () => {
      const name = renameName.trim();

      if (!name) {
        throw new Error("请输入新的问题组名称");
      }

      const updated = await renameCWAIReviewItemGroup(
        sessionId,
        group.id,
        group.version,
        name,
      );

      patchGroups([updated]);
      setMessage("问题组名称已更新。");
    });

  const handlePrimary = () =>
    runAction("primary", async () => {
      const updated = await setCWAIReviewItemGroupPrimary(
        sessionId,
        group.id,
        group.version,
        primaryItemID,
      );

      patchGroups([updated]);
      setMessage("主问题已更新。");
    });

  const handleAddMember = () =>
    runAction("add-member", async () => {
      if (!addItemID) {
        throw new Error("请选择要加入的问题");
      }

      const updated = await addCWAIReviewItemGroupMember(
        sessionId,
        group.id,
        group.version,
        addItemID,
      );

      patchGroups([updated]);
      setAddItemID("");
      setMessage("问题已加入该组。");
    });

  const handleRemoveMember = (
    member: CWAIReviewItemGroupMember,
  ) =>
    runAction(`remove:${member.id}`, async () => {
      const reason = (memberReasons[member.id] || "").trim();

      if (!reason) {
        throw new Error("请填写移除原因");
      }

      const updated = await removeCWAIReviewItemGroupMember(
        sessionId,
        group.id,
        group.version,
        member,
        reason,
      );

      patchGroups([updated]);

      setMemberReasons((previous) => ({
        ...previous,
        [member.id]: "",
      }));

      setMessage("问题已从该组移出，历史仍保留。");
    });

  const handleMoveMember = (
    member: CWAIReviewItemGroupMember,
  ) =>
    runAction(`move:${member.id}`, async () => {
      const targetID = moveTargets[member.id] || "";
      const reason = (memberReasons[member.id] || "").trim();

      const target = allGroups.find(
        (candidate) =>
          candidate.id === targetID && candidate.status === "active",
      );

      if (!target) {
        throw new Error("请选择目标问题组");
      }

      if (!reason) {
        throw new Error("请填写移动原因");
      }

      const result = await moveCWAIReviewItemGroupMember(
        sessionId,
        group,
        target,
        member,
        reason,
      );

      patchGroups([result.source, result.target]);

      setMemberReasons((previous) => ({
        ...previous,
        [member.id]: "",
      }));

      setMoveTargets((previous) => ({
        ...previous,
        [member.id]: "",
      }));

      setMessage(`问题已移动到“${result.target.name}”。`);
    });

  return (
    <div
      style={{
        marginBottom: "8px",
        padding: "9px",
        borderRadius: "8px",
        border: `1px solid ${C.border}`,
        background: merged ? "#F8FAFC" : "#FFFFFF",
        opacity: merged ? 0.75 : 1,
      }}
    >
      <div
        style={{
          display: "flex",
          gap: "6px",
          alignItems: "center",
          flexWrap: "wrap",
        }}
      >
        <strong
          style={{
            color: C.text,
            fontSize: "10px",
          }}
        >
          {group.name}
        </strong>

        <span
          style={{
            color: merged ? C.textMuted : C.success,
            fontSize: "8px",
            fontWeight: 700,
          }}
        >
          {merged ? "已合并" : "有效"}
        </span>

        <span
          style={{
            color: C.textMuted,
            fontSize: "8px",
          }}
        >
          并发版本 v{group.version} · {activeMembers.length} 条成员
        </span>
      </div>

      {merged && (
        <div
          style={{
            marginTop: "5px",
            color: C.textSec,
            fontSize: "8px",
          }}
        >
          该历史组已经合并，不再接受新的治理动作。
        </div>
      )}

      {!merged && (
        <>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "minmax(0,1fr) auto",
              gap: "5px",
              marginTop: "7px",
            }}
          >
            <input
              value={renameName}
              onChange={(event) => setRenameName(event.target.value)}
              disabled={!!busy}
              style={cwAIReviewItemGroupInputStyle}
            />

            <button
              type="button"
              disabled={
                !!busy ||
                !renameName.trim() ||
                renameName.trim() === group.name
              }
              onClick={() => void handleRename()}
              style={cwAIReviewItemGroupSecondaryButtonStyle}
            >
              重命名
            </button>
          </div>

          <div
            style={{
              display: "grid",
              gridTemplateColumns: "minmax(0,1fr) auto",
              gap: "5px",
              marginTop: "5px",
            }}
          >
            <select
              value={primaryItemID}
              onChange={(event) => setPrimaryItemID(event.target.value)}
              disabled={!!busy}
              style={cwAIReviewItemGroupInputStyle}
            >
              <option value="">不设主问题</option>

              {activeMembers.map((member) => (
                <option key={member.id} value={member.item_id}>
                  {cwAIReviewItemGroupItemLabel(itemMap.get(member.item_id))}
                </option>
              ))}
            </select>

            <button
              type="button"
              disabled={
                !!busy || primaryItemID === (group.primary_item_id || "")
              }
              onClick={() => void handlePrimary()}
              style={cwAIReviewItemGroupSecondaryButtonStyle}
            >
              保存主问题
            </button>
          </div>

          {addableItems.length > 0 && (
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "minmax(0,1fr) auto",
                gap: "5px",
                marginTop: "5px",
              }}
            >
              <select
                value={addItemID}
                onChange={(event) => setAddItemID(event.target.value)}
                disabled={!!busy}
                style={cwAIReviewItemGroupInputStyle}
              >
                <option value="">选择未分组问题</option>

                {addableItems.map((item) => (
                  <option key={item.id} value={item.id}>
                    {cwAIReviewItemGroupItemLabel(item)}
                  </option>
                ))}
              </select>

              <button
                type="button"
                disabled={!!busy || !addItemID}
                onClick={() => void handleAddMember()}
                style={cwAIReviewItemGroupSecondaryButtonStyle}
              >
                加入成员
              </button>
            </div>
          )}

          <div
            style={{
              marginTop: "7px",
              paddingTop: "6px",
              borderTop: `1px dashed ${C.border}`,
            }}
          >
            {activeMembers.map((member) => {
              const item = itemMap.get(member.item_id);
              const reason = memberReasons[member.id] || "";
              const targetID = moveTargets[member.id] || "";

              return (
                <div
                  key={member.id}
                  style={{
                    marginBottom: "6px",
                    padding: "6px",
                    borderRadius: "6px",
                    background: "#F8FAFC",
                  }}
                >
                  <div
                    style={{
                      display: "flex",
                      gap: "5px",
                      alignItems: "center",
                      flexWrap: "wrap",
                    }}
                  >
                    {item && item.page_number_snapshot > 0 && (
                      <button
                        type="button"
                        onClick={() => onSelectPage(item.page_number_snapshot)}
                        style={cwGlobalPageButtonStyle}
                      >
                        P{item.page_number_snapshot}
                      </button>
                    )}

                    <span
                      style={{
                        color: C.text,
                        fontSize: "8px",
                        fontWeight: 600,
                      }}
                    >
                      {cwAIReviewItemGroupItemLabel(item)}
                    </span>

                    {group.primary_item_id === member.item_id && (
                      <span
                        style={{
                          color: C.primary,
                          fontSize: "8px",
                          fontWeight: 700,
                        }}
                      >
                        主问题
                      </span>
                    )}
                  </div>

                  <input
                    value={reason}
                    onChange={(event) =>
                      setMemberReasons((previous) => ({
                        ...previous,
                        [member.id]: event.target.value,
                      }))
                    }
                    disabled={!!busy}
                    placeholder="移动或移除原因"
                    style={cwAIReviewItemGroupInputStyle}
                  />

                  {otherActiveGroups.length > 0 ? (
                    <div
                      style={{
                        display: "grid",
                        gridTemplateColumns: "minmax(0,1fr) auto auto",
                        gap: "5px",
                        marginTop: "4px",
                      }}
                    >
                      <select
                        value={targetID}
                        onChange={(event) =>
                          setMoveTargets((previous) => ({
                            ...previous,
                            [member.id]: event.target.value,
                          }))
                        }
                        disabled={!!busy}
                        style={cwAIReviewItemGroupInputStyle}
                      >
                        <option value="">移动到其他组</option>

                        {otherActiveGroups.map((candidate) => (
                          <option key={candidate.id} value={candidate.id}>
                            {candidate.name}
                          </option>
                        ))}
                      </select>

                      <button
                        type="button"
                        disabled={!!busy || !targetID || !reason.trim()}
                        onClick={() => void handleMoveMember(member)}
                        style={cwAIReviewItemGroupSecondaryButtonStyle}
                      >
                        移动
                      </button>

                      <button
                        type="button"
                        disabled={!!busy || !reason.trim()}
                        onClick={() => void handleRemoveMember(member)}
                        style={cwAIReviewItemGroupDangerButtonStyle}
                      >
                        移出
                      </button>
                    </div>
                  ) : (
                    <button
                      type="button"
                      disabled={!!busy || !reason.trim()}
                      onClick={() => void handleRemoveMember(member)}
                      style={{
                        ...cwAIReviewItemGroupDangerButtonStyle,
                        width: "100%",
                        marginTop: "4px",
                      }}
                    >
                      从组中移出
                    </button>
                  )}
                </div>
              );
            })}
          </div>

          <CWAIReviewItemGroupStructuralActions
            sessionId={sessionId}
            group={group}
            allGroups={allGroups}
            itemMap={itemMap}
            onGroupsChanged={onGroupsChanged}
          />
        </>
      )}

      {group.events.length > 0 && (
        <details style={{ marginTop: "7px" }}>
          <summary
            style={{
              color: C.textMuted,
              fontSize: "8px",
              cursor: "pointer",
            }}
          >
            治理历史（{group.events.length}）
          </summary>

          <div style={{ marginTop: "4px" }}>
            {group.events.map((event) => (
              <div
                key={event.id}
                style={{
                  marginTop: "3px",
                  color: C.textSec,
                  fontSize: "8px",
                  lineHeight: 1.5,
                }}
              >
                v{event.group_version} ·{" "}
                {cwAIReviewItemGroupEventLabel(event.event_type)}
                {event.reason ? ` · ${event.reason}` : ""}
              </div>
            ))}
          </div>
        </details>
      )}

      {error && (
        <CWAIReviewItemGroupFeedback type="error" content={error} />
      )}

      {message && (
        <CWAIReviewItemGroupFeedback type="success" content={message} />
      )}
    </div>
  );
}
