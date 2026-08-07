/**
 * CWAIReviewGlobalRelationGovernance.tsx
 *
 * 全局讨论关系建议的独立确认、重新确认、取消和审计历史展示。
 *
 * 浏览器只提交可信消息ID、关系类型及双端整改项ID；
 * explanation始终由后端从可信assistant消息重新读取。
 */

import {
  useMemo,
  useState,
} from "react";

import {
  cancelCWAIReviewGlobalRelation,
  confirmCWAIReviewGlobalRelation,
  type CWAIReviewGlobalRelation,
  type CWAIReviewItem,
  type CWAIReviewItemRelation,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
  CW_GLOBAL_RELATION_CONFIG,
  cwGlobalPageButtonStyle,
  resolveCWGlobalItemPageLabel,
  resolveCWGlobalItemTitle,
} from "./CWAIReviewGlobalDiscussion.shared";
import {
  countCWGlobalRunes,
  CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES,
} from "./CWAIReviewGlobalGovernanceLimits";

export interface CWAIReviewGlobalRelationGovernanceProps {
  sessionId: string;
  messageId: string;
  suggestions: CWAIReviewGlobalRelation[];
  relations: CWAIReviewItemRelation[];
  items: CWAIReviewItem[];

  onSelectPage: (pageNumber: number) => void;
  onRelationsChanged: (
    relations: CWAIReviewItemRelation[],
  ) => void;
}

function relationKey(
  relationType: string,
  sourceItemID: string,
  targetItemID: string,
): string {
  return [
    relationType.trim(),
    sourceItemID.trim(),
    targetItemID.trim(),
  ].join("|");
}

function mergeRelation(
  relations: CWAIReviewItemRelation[],
  incoming: CWAIReviewItemRelation,
): CWAIReviewItemRelation[] {
  const next = relations.filter(
    (relation) => relation.id !== incoming.id,
  );

  next.push(incoming);

  return next.sort((left, right) => {
    if (left.status !== right.status) {
      return left.status === "active" ? -1 : 1;
    }

    return (right.updated_at || "").localeCompare(
      left.updated_at || "",
    );
  });
}

function directionText(
  relation: CWAIReviewGlobalRelation,
): string {
  switch (relation.type) {
    case "duplicate":
      return "源问题重复目标问题；目标问题作为保留主问题";
    case "merge":
      return "源问题合并进入目标问题";
    case "dependency":
      return "源问题依赖目标问题先完成";
    case "possibly_resolved":
      return "源问题可能被目标问题的修改连带解决";
    case "conflict":
      return "无方向冲突关系，双端均需人工裁决";
    default:
      return "";
  }
}

function itemLabel(
  item: CWAIReviewItem | undefined,
  fallbackID: string,
): string {
  if (!item) {
    return fallbackID;
  }

  return [
    resolveCWGlobalItemPageLabel(item),
    resolveCWGlobalItemTitle(item),
  ].join(" · ");
}

export default function CWAIReviewGlobalRelationGovernance({
  sessionId,
  messageId,
  suggestions,
  relations,
  items,
  onSelectPage,
  onRelationsChanged,
}: CWAIReviewGlobalRelationGovernanceProps) {
  const [busyKey, setBusyKey] = useState("");
  const [cancelReasons, setCancelReasons] =
    useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const itemMap = useMemo(
    () =>
      new Map(
        items.map((item) => [item.id, item]),
      ),
    [items],
  );

  const activeKeySet = useMemo(
    () =>
      new Set(
        relations
          .filter((relation) => relation.status === "active")
          .map((relation) =>
            relationKey(
              relation.relation_type,
              relation.source_item_id,
              relation.target_item_id,
            ),
          ),
      ),
    [relations],
  );

  const handleConfirm = async (
    relation: CWAIReviewGlobalRelation,
  ) => {
    const sourceItemID =
      relation.source_item_id?.trim() || "";
    const targetItemID =
      relation.target_item_id?.trim() || "";
    const key = relationKey(
      relation.type,
      sourceItemID,
      targetItemID,
    );

    if (
      busyKey ||
      !messageId ||
      !sourceItemID ||
      !targetItemID
    ) {
      setError(
        !sourceItemID || !targetItemID
          ? "该历史关系缺少V2双端方向，不能直接持久化，请重新发起全局讨论"
          : "当前关系暂不能确认",
      );
      return;
    }

    setBusyKey(`confirm:${key}`);
    setError("");
    setMessage("");

    try {
      const record =
        await confirmCWAIReviewGlobalRelation(
          sessionId,
          messageId,
          relation.type,
          sourceItemID,
          targetItemID,
        );

      onRelationsChanged(
        mergeRelation(relations, record),
      );
      setMessage(
        "关系已由人工明确确认；页面、指令和审核决定均未改变。",
      );
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "确认整改项关系失败",
      );
    } finally {
      setBusyKey("");
    }
  };

  const handleCancel = async (
    relation: CWAIReviewItemRelation,
  ) => {
    const reason =
      (cancelReasons[relation.id] || "").trim();
    const reasonLength = countCWGlobalRunes(reason);

    if (
      !reason ||
      reasonLength > CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES ||
      busyKey
    ) {
      setError(
        reasonLength > CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
          ? `取消关系原因不能超过${CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES}字`
          : "请填写取消关系原因",
      );
      return;
    }

    setBusyKey(`cancel:${relation.id}`);
    setError("");
    setMessage("");

    try {
      const record =
        await cancelCWAIReviewGlobalRelation(
          sessionId,
          relation.id,
          reason,
        );

      onRelationsChanged(
        mergeRelation(relations, record),
      );
      setCancelReasons((previous) => ({
        ...previous,
        [relation.id]: "",
      }));
      setMessage(
        "关系已取消，历史事件仍完整保留；整改项和页面状态未改变。",
      );
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "取消整改项关系失败",
      );
    } finally {
      setBusyKey("");
    }
  };

  if (suggestions.length === 0 && relations.length === 0) {
    return null;
  }

  return (
    <>
      {suggestions.length > 0 && (
        <div style={{ marginTop: "10px" }}>
          <div style={sectionTitleStyle}>
            AI关系建议的独立确认
          </div>

          {suggestions.map((relation, index) => {
            const sourceItemID =
              relation.source_item_id?.trim() || "";
            const targetItemID =
              relation.target_item_id?.trim() || "";
            const key = relationKey(
              relation.type,
              sourceItemID,
              targetItemID,
            );
            const config =
              CW_GLOBAL_RELATION_CONFIG[relation.type];
            const active = activeKeySet.has(key);
            const canPersist =
              !!messageId &&
              !!sourceItemID &&
              !!targetItemID;

            return (
              <div
                key={`${key}-${index}`}
                style={{
                  marginBottom: "7px",
                  padding: "8px",
                  borderRadius: "7px",
                  border: `1px solid ${config.color}35`,
                  background: config.background,
                }}
              >
                <div style={badgeRowStyle}>
                  <span
                    style={{
                      ...relationBadgeStyle,
                      color: config.color,
                    }}
                  >
                    {config.label}
                  </span>

                  {active && (
                    <span style={activeLabelStyle}>
                      ✓ 已人工确认
                    </span>
                  )}
                </div>

                <RelationEndpoint
                  label="源问题"
                  item={itemMap.get(sourceItemID)}
                  fallbackID={sourceItemID || "缺少V2源端点"}
                  onSelectPage={onSelectPage}
                />

                <RelationEndpoint
                  label="目标问题"
                  item={itemMap.get(targetItemID)}
                  fallbackID={targetItemID || "缺少V2目标端点"}
                  onSelectPage={onSelectPage}
                />

                <div style={directionStyle}>
                  方向：{directionText(relation)}
                </div>

                <div
                  style={{
                    marginTop: "4px",
                    color: C.text,
                  }}
                >
                  <DiscussionMarkdown
                    content={relation.explanation}
                    compact
                  />
                </div>

                {!canPersist && (
                  <div style={legacyWarningStyle}>
                    该历史结果缺少V2双端方向，仍可查看和采用候选指令，
                    但不能持久化关系。请重新发起一轮全局讨论。
                  </div>
                )}

                <button
                  type="button"
                  onClick={() =>
                    void handleConfirm(relation)
                  }
                  disabled={
                    active ||
                    !canPersist ||
                    !!busyKey
                  }
                  style={{
                    ...fullButtonStyle,
                    border: `1px solid ${config.color}`,
                    background:
                      active
                        ? C.successSoft
                        : !canPersist || busyKey
                          ? "#F1F5F9"
                          : "#fff",
                    color:
                      active
                        ? C.success
                        : !canPersist || busyKey
                          ? C.textMuted
                          : config.color,
                    cursor:
                      active ||
                      !canPersist ||
                      busyKey
                        ? "not-allowed"
                        : "pointer",
                  }}
                >
                  {active
                    ? "已确认此关系"
                    : busyKey === `confirm:${key}`
                      ? "正在确认…"
                      : "独立确认此关系"}
                </button>
              </div>
            );
          })}
        </div>
      )}

      {relations.length > 0 && (
        <div style={{ marginTop: "10px" }}>
          <div style={sectionTitleStyle}>
            已确认关系与审计历史
          </div>

          {relations.map((relation) => {
            const config =
              CW_GLOBAL_RELATION_CONFIG[
                relation.relation_type
              ];
            const cancelReason =
              cancelReasons[relation.id] || "";
            const cancelReasonLength =
              countCWGlobalRunes(cancelReason);
            const active =
              relation.status === "active";

            return (
              <div
                key={relation.id}
                style={recordCardStyle}
              >
                <div style={badgeRowStyle}>
                  <span
                    style={{
                      ...relationBadgeStyle,
                      background: config.background,
                      color: config.color,
                    }}
                  >
                    {config.label}
                  </span>

                  <span
                    style={{
                      color: active
                        ? C.success
                        : C.textMuted,
                      fontSize: "9px",
                      fontWeight: 700,
                    }}
                  >
                    {active ? "有效" : "已取消"} · v
                    {relation.version}
                  </span>
                </div>

                <RelationEndpoint
                  label="源问题"
                  item={itemMap.get(relation.source_item_id)}
                  fallbackID={relation.source_item_id}
                  onSelectPage={onSelectPage}
                />

                <RelationEndpoint
                  label="目标问题"
                  item={itemMap.get(relation.target_item_id)}
                  fallbackID={relation.target_item_id}
                  onSelectPage={onSelectPage}
                />

                <div
                  style={{
                    marginTop: "5px",
                    color: C.textSec,
                  }}
                >
                  <DiscussionMarkdown
                    content={relation.explanation}
                    compact
                  />
                </div>

                {relation.events.length > 0 && (
                  <div style={eventBoxStyle}>
                    {relation.events.map((event) => (
                      <div
                        key={event.id}
                        style={eventLineStyle}
                      >
                        v{event.relation_version} ·{" "}
                        {event.event_type} ·{" "}
                        {event.reason || "无补充说明"}
                      </div>
                    ))}
                  </div>
                )}

                {active && (
                  <>
                    <textarea
                      value={cancelReason}
                      onChange={(event) =>
                        setCancelReasons((previous) => ({
                          ...previous,
                          [relation.id]: event.target.value,
                        }))
                      }
                      rows={2}
                      maxLength={
                        CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                      }
                      disabled={!!busyKey}
                      placeholder="填写取消关系原因；关系历史不会删除"
                      style={reasonInputStyle}
                    />

                    <div
                      style={{
                        marginTop: "3px",
                        color:
                          cancelReasonLength >
                          CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                            ? C.danger
                            : C.textMuted,
                        fontSize: "8px",
                        textAlign: "right",
                      }}
                    >
                      {cancelReasonLength}/
                      {CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES}
                    </div>

                    <button
                      type="button"
                      onClick={() =>
                        void handleCancel(relation)
                      }
                      disabled={
                        !!busyKey ||
                        !cancelReason.trim() ||
                        cancelReasonLength >
                          CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                      }
                      style={{
                        ...fullButtonStyle,
                        border: `1px solid ${C.danger}`,
                        background:
                          busyKey ||
                          !cancelReason.trim() ||
                          cancelReasonLength >
                            CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                            ? "#F1F5F9"
                            : C.dangerSoft,
                        color:
                          busyKey ||
                          !cancelReason.trim() ||
                          cancelReasonLength >
                            CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                            ? C.textMuted
                            : C.danger,
                        cursor:
                          busyKey ||
                          !cancelReason.trim() ||
                          cancelReasonLength >
                            CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES
                            ? "not-allowed"
                            : "pointer",
                      }}
                    >
                      {busyKey === `cancel:${relation.id}`
                        ? "正在取消…"
                        : "取消此关系"}
                    </button>
                  </>
                )}
              </div>
            );
          })}
        </div>
      )}

      {error && (
        <FeedbackMessage
          type="error"
          content={error}
        />
      )}

      {message && (
        <FeedbackMessage
          type="success"
          content={message}
        />
      )}
    </>
  );
}

function RelationEndpoint({
  label,
  item,
  fallbackID,
  onSelectPage,
}: {
  label: string;
  item?: CWAIReviewItem;
  fallbackID: string;
  onSelectPage: (pageNumber: number) => void;
}) {
  return (
    <div style={endpointStyle}>
      <strong>{label}：</strong>

      {item && item.page_number_snapshot > 0 && (
        <button
          type="button"
          onClick={() =>
            onSelectPage(item.page_number_snapshot)
          }
          style={{
            ...cwGlobalPageButtonStyle,
            color: C.primary,
          }}
        >
          P{item.page_number_snapshot}
        </button>
      )}

      <span style={{ minWidth: 0, flex: 1 }}>
        {itemLabel(item, fallbackID)}
      </span>
    </div>
  );
}

function FeedbackMessage({
  type,
  content,
}: {
  type: "success" | "error";
  content: string;
}) {
  return (
    <div
      style={{
        marginTop: "8px",
        padding: "7px 9px",
        borderRadius: "7px",
        background:
          type === "success"
            ? C.successSoft
            : C.dangerSoft,
        color:
          type === "success"
            ? C.success
            : C.danger,
        fontSize: "9px",
        fontWeight: 600,
        lineHeight: 1.5,
      }}
    >
      {content}
    </div>
  );
}

const sectionTitleStyle: React.CSSProperties = {
  marginBottom: "6px",
  color: C.text,
  fontSize: "10px",
  fontWeight: 700,
};

const badgeRowStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: "6px",
  flexWrap: "wrap",
};

const relationBadgeStyle: React.CSSProperties = {
  padding: "2px 6px",
  borderRadius: "5px",
  background: "#fff",
  fontSize: "9px",
  fontWeight: 700,
};

const activeLabelStyle: React.CSSProperties = {
  color: C.success,
  fontSize: "9px",
  fontWeight: 700,
};

const endpointStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: "5px",
  marginTop: "5px",
  color: C.textSec,
  fontSize: "9px",
};

const directionStyle: React.CSSProperties = {
  marginTop: "4px",
  color: C.textSec,
  fontSize: "9px",
  lineHeight: 1.5,
};

const legacyWarningStyle: React.CSSProperties = {
  marginTop: "5px",
  color: C.warning,
  fontSize: "9px",
  lineHeight: 1.5,
};

const fullButtonStyle: React.CSSProperties = {
  width: "100%",
  marginTop: "7px",
  padding: "6px",
  borderRadius: "6px",
  fontSize: "9px",
  fontWeight: 700,
};

const recordCardStyle: React.CSSProperties = {
  marginBottom: "7px",
  padding: "8px",
  borderRadius: "7px",
  border: `1px solid ${C.border}`,
  background: C.card,
};

const eventBoxStyle: React.CSSProperties = {
  marginTop: "6px",
  padding: "6px",
  borderRadius: "6px",
  background: C.bg,
};

const eventLineStyle: React.CSSProperties = {
  marginBottom: "3px",
  color: C.textMuted,
  fontSize: "8px",
  lineHeight: 1.5,
};

const reasonInputStyle: React.CSSProperties = {
  width: "100%",
  boxSizing: "border-box",
  marginTop: "7px",
  padding: "7px 8px",
  borderRadius: "6px",
  border: `1px solid ${C.border}`,
  resize: "vertical",
  fontFamily: "inherit",
  fontSize: "9px",
  lineHeight: 1.5,
};
