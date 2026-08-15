/**
 * CWAIReviewItemGroupStructuralActions.tsx
 *
 * R-06正式问题组的结构治理：
 *   - 合并整个问题组；
 *   - 从来源组拆出部分成员建立新组。
 *
 * 所有请求都携带后端最近返回的group version；
 * 发生并发变化时由后端409拒绝旧版本覆盖。
 */

import { useMemo, useState } from "react";

import {
  mergeCWAIReviewItemGroups,
  splitCWAIReviewItemGroup,
  type CWAIReviewItem,
  type CWAIReviewItemGroup,
} from "@/api/coursewares";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
} from "./CWAIReviewGlobalDiscussion.shared";

import {
  CWAIReviewItemGroupFeedback,
  cwAIReviewItemGroupInputStyle,
  cwAIReviewItemGroupItemLabel,
  cwAIReviewItemGroupPrimaryButtonStyle,
  cwAIReviewItemGroupSectionTitleStyle,
  readCWAIReviewItemGroupError,
  upsertCWAIReviewItemGroups,
} from "./CWAIReviewItemGroups.shared";

export interface CWAIReviewItemGroupStructuralActionsProps {
  sessionId: string;
  group: CWAIReviewItemGroup;
  allGroups: CWAIReviewItemGroup[];
  itemMap: Map<string, CWAIReviewItem>;
  onGroupsChanged: (groups: CWAIReviewItemGroup[]) => void;
}

export default function CWAIReviewItemGroupStructuralActions({
  sessionId,
  group,
  allGroups,
  itemMap,
  onGroupsChanged,
}: CWAIReviewItemGroupStructuralActionsProps) {
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const [mergeTargetID, setMergeTargetID] = useState("");
  const [mergeReason, setMergeReason] = useState("");

  const [splitName, setSplitName] = useState("");
  const [splitPrimaryItemID, setSplitPrimaryItemID] = useState("");
  const [splitReason, setSplitReason] = useState("");
  const [splitItemIDs, setSplitItemIDs] = useState<string[]>([]);

  const activeMembers = useMemo(
    () => group.members.filter((member) => member.status === "active"),
    [group.members],
  );

  const otherActiveGroups = useMemo(
    () =>
      allGroups.filter(
        (candidate) =>
          candidate.status === "active" && candidate.id !== group.id,
      ),
    [allGroups, group.id],
  );

  const patchGroups = (incoming: CWAIReviewItemGroup[]) => {
    onGroupsChanged(upsertCWAIReviewItemGroups(allGroups, incoming));
  };

  const runAction = async (
    key: string,
    action: () => Promise<void>,
  ) => {
    if (busy || group.status !== "active") {
      return;
    }

    setBusy(key);
    setError("");
    setMessage("");

    try {
      await action();
    } catch (cause) {
      setError(readCWAIReviewItemGroupError(cause, "问题组结构治理失败"));
    } finally {
      setBusy("");
    }
  };

  const handleMerge = () =>
    runAction("merge", async () => {
      const target = allGroups.find(
        (candidate) =>
          candidate.id === mergeTargetID && candidate.status === "active",
      );
      const reason = mergeReason.trim();

      if (!target) {
        throw new Error("请选择合并目标组");
      }

      if (!reason) {
        throw new Error("请填写合并原因");
      }

      const result = await mergeCWAIReviewItemGroups(
        sessionId,
        group,
        target,
        reason,
      );

      patchGroups([result.source, result.target]);
      setMergeTargetID("");
      setMergeReason("");
      setMessage(`已合并进入“${result.target.name}”。`);
    });

  const handleToggleSplitItem = (
    itemID: string,
    selected: boolean,
  ) => {
    setSplitItemIDs((previous) =>
      selected
        ? Array.from(new Set([...previous, itemID]))
        : previous.filter((current) => current !== itemID),
    );

    if (!selected && splitPrimaryItemID === itemID) {
      setSplitPrimaryItemID("");
    }
  };

  const handleSplit = () =>
    runAction("split", async () => {
      const name = splitName.trim();
      const reason = splitReason.trim();

      if (!name) {
        throw new Error("请填写拆出后的新问题组名称");
      }

      if (
        splitItemIDs.length === 0 ||
        splitItemIDs.length >= activeMembers.length
      ) {
        throw new Error("拆分必须选择部分成员，不能为空或把整组全部移走");
      }

      if (!reason) {
        throw new Error("请填写拆分原因");
      }

      const result = await splitCWAIReviewItemGroup(
        sessionId,
        group,
        name,
        splitItemIDs,
        splitPrimaryItemID,
        reason,
      );

      patchGroups([result.source, result.target]);

      setSplitName("");
      setSplitItemIDs([]);
      setSplitPrimaryItemID("");
      setSplitReason("");

      setMessage(`已拆出新问题组“${result.target.name}”。`);
    });

  return (
    <>
      {otherActiveGroups.length > 0 && (
        <div
          style={{
            marginTop: "7px",
            paddingTop: "7px",
            borderTop: `1px dashed ${C.border}`,
          }}
        >
          <div style={cwAIReviewItemGroupSectionTitleStyle}>
            合并整个问题组
          </div>

          <select
            value={mergeTargetID}
            onChange={(event) => setMergeTargetID(event.target.value)}
            disabled={!!busy}
            style={cwAIReviewItemGroupInputStyle}
          >
            <option value="">选择保留的目标组</option>

            {otherActiveGroups.map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {candidate.name}
              </option>
            ))}
          </select>

          <input
            value={mergeReason}
            onChange={(event) => setMergeReason(event.target.value)}
            disabled={!!busy}
            placeholder="填写合并原因"
            style={cwAIReviewItemGroupInputStyle}
          />

          <button
            type="button"
            disabled={!!busy || !mergeTargetID || !mergeReason.trim()}
            onClick={() => void handleMerge()}
            style={cwAIReviewItemGroupPrimaryButtonStyle(
              !!busy || !mergeTargetID || !mergeReason.trim(),
            )}
          >
            {busy === "merge" ? "正在合并…" : "合并进入目标组"}
          </button>
        </div>
      )}

      {activeMembers.length > 1 && (
        <div
          style={{
            marginTop: "7px",
            paddingTop: "7px",
            borderTop: `1px dashed ${C.border}`,
          }}
        >
          <div style={cwAIReviewItemGroupSectionTitleStyle}>
            拆出新的问题组
          </div>

          <div
            style={{
              marginTop: "4px",
              color: C.textMuted,
              fontSize: "8px",
              lineHeight: 1.5,
            }}
          >
            只能拆出部分成员。来源组和新组都会保留各自独立版本与事件历史。
          </div>

          {activeMembers.map((member) => (
            <label
              key={member.id}
              style={{
                display: "flex",
                gap: "5px",
                alignItems: "center",
                marginTop: "4px",
                color: C.textSec,
                fontSize: "8px",
              }}
            >
              <input
                type="checkbox"
                checked={splitItemIDs.includes(member.item_id)}
                disabled={!!busy}
                onChange={(event) =>
                  handleToggleSplitItem(member.item_id, event.target.checked)
                }
              />

              {cwAIReviewItemGroupItemLabel(itemMap.get(member.item_id))}
            </label>
          ))}

          <input
            value={splitName}
            onChange={(event) => setSplitName(event.target.value)}
            disabled={!!busy}
            placeholder="新问题组名称"
            style={cwAIReviewItemGroupInputStyle}
          />

          <select
            value={splitPrimaryItemID}
            onChange={(event) => setSplitPrimaryItemID(event.target.value)}
            disabled={!!busy || splitItemIDs.length === 0}
            style={cwAIReviewItemGroupInputStyle}
          >
            <option value="">新组不预设主问题</option>

            {splitItemIDs.map((itemID) => (
              <option key={itemID} value={itemID}>
                {cwAIReviewItemGroupItemLabel(itemMap.get(itemID))}
              </option>
            ))}
          </select>

          <input
            value={splitReason}
            onChange={(event) => setSplitReason(event.target.value)}
            disabled={!!busy}
            placeholder="填写拆分原因"
            style={cwAIReviewItemGroupInputStyle}
          />

          <button
            type="button"
            disabled={
              !!busy ||
              !splitName.trim() ||
              !splitReason.trim() ||
              splitItemIDs.length === 0 ||
              splitItemIDs.length >= activeMembers.length
            }
            onClick={() => void handleSplit()}
            style={cwAIReviewItemGroupPrimaryButtonStyle(
              !!busy ||
                !splitName.trim() ||
                !splitReason.trim() ||
                splitItemIDs.length === 0 ||
                splitItemIDs.length >= activeMembers.length,
            )}
          >
            {busy === "split" ? "正在拆分…" : "拆出新问题组"}
          </button>
        </div>
      )}

      {error && (
        <CWAIReviewItemGroupFeedback type="error" content={error} />
      )}

      {message && (
        <CWAIReviewItemGroupFeedback type="success" content={message} />
      )}
    </>
  );
}
