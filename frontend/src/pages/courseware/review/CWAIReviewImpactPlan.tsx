/**
 * CWAIReviewImpactPlan.tsx
 *
 * R-07全局讨论“统一影响方案”教师Preview与一次确认组件。
 *
 * 安全边界：
 *   1. Draft只提交可信message_id；
 *   2. Preview payload只用于展示，绝不参与最终Apply请求；
 *   3. 最终Apply只提交version + selected_operation_ids；
 *   4. 默认全选，教师可以逐项取消；
 *   5. Apply成功后只采用后端返回的applied计划作为结果，不做业务对象乐观改写；
 *   6. 目标状态变化时后端409并整体回滚，教师可重新生成新方案。
 */

import {
  useEffect,
  useMemo,
  useState,
} from "react";

import {
  applyCWAIReviewImpactPlan,
  createCWAIReviewImpactPlan,
  getCWAIReviewImpactPlan,
  type CWAIReviewImpactPlan as CWAIReviewImpactPlanData,
} from "@/api/coursewares";

import {
  CW_GLOBAL_DISCUSSION_COLORS as C,
  cwGlobalSecondaryButtonStyle,
} from "./CWAIReviewGlobalDiscussion.shared";
import CWAIReviewImpactPlanOperationList from "./CWAIReviewImpactPlanOperation";

export interface CWAIReviewImpactPlanProps {
  sessionId: string;
  messageId: string;
}

const IMPACT_PLAN_STORAGE_PREFIX =
  "tedna:cw-ai-review-impact-plan:v1";

export default function CWAIReviewImpactPlan({
  sessionId,
  messageId,
}: CWAIReviewImpactPlanProps) {
  const [plan, setPlan] =
    useState<CWAIReviewImpactPlanData | null>(null);

  const [
    selectedOperationIDs,
    setSelectedOperationIDs,
  ] = useState<string[]>([]);

  const [busy, setBusy] =
    useState<"" | "restore" | "create" | "apply">("");

  const [error, setError] =
    useState("");

  const [message, setMessage] =
    useState("");

  const storageKey = useMemo(
    () =>
      `${IMPACT_PLAN_STORAGE_PREFIX}:${sessionId}:${messageId}`,
    [messageId, sessionId],
  );

  const selectedSet = useMemo(
    () => new Set(selectedOperationIDs),
    [selectedOperationIDs],
  );

  useEffect(() => {
    setPlan(null);
    setSelectedOperationIDs([]);
    setError("");
    setMessage("");

    if (!sessionId || !messageId) {
      setBusy("");
      return;
    }

    let cancelled = false;
    let storedPlanID = "";

    try {
      storedPlanID =
        window.sessionStorage.getItem(storageKey) || "";
    } catch {
      storedPlanID = "";
    }

    if (!storedPlanID) {
      setBusy("");
      return;
    }

    setBusy("restore");

    void getCWAIReviewImpactPlan(
      sessionId,
      storedPlanID,
    )
      .then((restored) => {
        if (cancelled) {
          return;
        }

        setPlan(restored);

        setSelectedOperationIDs(
          restored.status === "applied"
            ? restored.applied_operation_ids || []
            : restored.operations.map(
                (operation) =>
                  operation.operation_id,
              ),
        );
      })
      .catch(() => {
        if (cancelled) {
          return;
        }

        try {
          window.sessionStorage.removeItem(storageKey);
        } catch {
          // sessionStorage不可用时不影响主流程。
        }
      })
      .finally(() => {
        if (!cancelled) {
          setBusy("");
        }
      });

    return () => {
      cancelled = true;
    };
  }, [
    messageId,
    sessionId,
    storageKey,
  ]);

  const rememberPlan = (
    nextPlan: CWAIReviewImpactPlanData,
  ) => {
    try {
      window.sessionStorage.setItem(
        storageKey,
        nextPlan.id,
      );
    } catch {
      // 仅用于恢复Preview，不作为业务状态事实源。
    }
  };

  const handleGenerate = async () => {
    if (
      !sessionId ||
      !messageId ||
      busy
    ) {
      return;
    }

    setBusy("create");
    setError("");
    setMessage("");

    try {
      const nextPlan =
        await createCWAIReviewImpactPlan(
          sessionId,
          messageId,
        );

      setPlan(nextPlan);

      setSelectedOperationIDs(
        nextPlan.operations.map(
          (operation) =>
            operation.operation_id,
        ),
      );

      rememberPlan(nextPlan);

      setMessage(
        `已生成${nextPlan.operations.length}项影响动作。请先取消不需要的项，再一次确认。`,
      );
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "生成统一影响方案失败",
      );
    } finally {
      setBusy("");
    }
  };

  const handleToggleOperation = (
    operationID: string,
    selected: boolean,
  ) => {
    if (
      !plan ||
      plan.status !== "draft" ||
      busy
    ) {
      return;
    }

    setSelectedOperationIDs(
      (previous) => {
        if (!selected) {
          return previous.filter(
            (current) =>
              current !== operationID,
          );
        }

        return Array.from(
          new Set([
            ...previous,
            operationID,
          ]),
        );
      },
    );

    setError("");
    setMessage("");
  };

  const handleSelectAll = () => {
    if (
      !plan ||
      plan.status !== "draft" ||
      busy
    ) {
      return;
    }

    setSelectedOperationIDs(
      plan.operations.map(
        (operation) =>
          operation.operation_id,
      ),
    );

    setError("");
  };

  const handleClearSelection = () => {
    if (
      !plan ||
      plan.status !== "draft" ||
      busy
    ) {
      return;
    }

    setSelectedOperationIDs([]);
    setError("");
  };

  const handleApply = async () => {
    if (
      !plan ||
      plan.status !== "draft" ||
      plan.version !== 1 ||
      selectedOperationIDs.length === 0 ||
      busy
    ) {
      setError(
        selectedOperationIDs.length === 0
          ? "请至少保留一项需要应用的动作"
          : "当前影响方案不能提交，请重新生成",
      );
      return;
    }

    setBusy("apply");
    setError("");
    setMessage("");

    try {
      const applied =
        await applyCWAIReviewImpactPlan(
          sessionId,
          plan.id,
          plan.version,
          selectedOperationIDs,
        );

      setPlan(applied);

      setSelectedOperationIDs(
        applied.applied_operation_ids || [],
      );

      rememberPlan(applied);

      setMessage(
        `已原子应用${applied.applied_operation_ids.length}项动作；未勾选动作没有执行。`,
      );

      window.dispatchEvent(
        new CustomEvent(
          "tedna:cw-ai-review-impact-plan-applied",
          {
            detail: {
              sessionId,
              planId: applied.id,
            },
          },
        ),
      );
    } catch (cause) {
      setError(
        cause instanceof Error
          ? `${cause.message}；若问题或分组状态已经变化，请重新生成方案。`
          : "应用统一影响方案失败；请刷新后重新生成方案。",
      );
    } finally {
      setBusy("");
    }
  };

  const draft =
    plan?.status === "draft";

  const applied =
    plan?.status === "applied";

  return (
    <section
      style={{
        marginTop: "10px",
        padding: "9px",
        borderRadius: "8px",
        border:
          `1px solid ${C.primary}45`,
        background: C.primarySoft,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: "8px",
        }}
      >
        <div
          style={{
            minWidth: 0,
            flex: 1,
          }}
        >
          <div
            style={{
              color: C.text,
              fontSize: "10px",
              fontWeight: 700,
            }}
          >
            ✨ 统一影响方案
          </div>

          <div
            style={{
              marginTop: "3px",
              color: C.textSec,
              fontSize: "9px",
              lineHeight: 1.55,
            }}
          >
            把本轮全局讨论整理成可预览的动作清单。
            默认全选，你可以取消任意动作；
            最终只提交所选operation ID，
            由后端再次核验并一次原子应用。
          </div>
        </div>

        <button
          type="button"
          onClick={() =>
            void handleGenerate()
          }
          disabled={
            !messageId ||
            !!busy
          }
          style={{
            ...cwGlobalSecondaryButtonStyle,
            opacity:
              !messageId || busy
                ? 0.55
                : 1,
            cursor:
              !messageId || busy
                ? "not-allowed"
                : "pointer",
          }}
        >
          {busy === "create"
            ? "正在生成…"
            : plan
              ? "重新生成"
              : "生成方案"}
        </button>
      </div>

      {!messageId && (
        <div
          style={{
            marginTop: "7px",
            color: C.textMuted,
            fontSize: "9px",
            lineHeight: 1.5,
          }}
        >
          先完成一轮全局讨论并保存AI回复，
          才能生成统一影响方案。
        </div>
      )}

      {busy === "restore" && (
        <div
          style={{
            marginTop: "7px",
            color: C.textMuted,
            fontSize: "9px",
          }}
        >
          正在恢复本次讨论的影响方案…
        </div>
      )}

      {plan && (
        <>
          <div
            style={{
              marginTop: "8px",
              display: "flex",
              alignItems: "center",
              gap: "6px",
              flexWrap: "wrap",
            }}
          >
            <span
              style={{
                padding: "2px 6px",
                borderRadius: "999px",
                background:
                  applied
                    ? C.successSoft
                    : "#fff",
                color:
                  applied
                    ? C.success
                    : C.primary,
                fontSize: "8px",
                fontWeight: 700,
              }}
            >
              {applied
                ? "已应用"
                : "待确认"}
            </span>

            <span
              style={{
                color: C.textMuted,
                fontSize: "8px",
              }}
            >
              方案版本 {plan.version}
              {" · "}
              共 {plan.operations.length} 项
              {draft
                ? ` · 已选${selectedOperationIDs.length}项`
                : ` · 已执行${plan.applied_operation_ids.length}项`}
            </span>

            {draft && (
              <>
                <button
                  type="button"
                  onClick={handleSelectAll}
                  disabled={!!busy}
                  style={miniButtonStyle}
                >
                  全选
                </button>

                <button
                  type="button"
                  onClick={handleClearSelection}
                  disabled={!!busy}
                  style={miniButtonStyle}
                >
                  全不选
                </button>
              </>
            )}
          </div>

          <CWAIReviewImpactPlanOperationList
            operations={plan.operations}
            selectedOperationIDs={selectedSet}
            disabled={!draft || !!busy}
            onSelectedChange={
              handleToggleOperation
            }
          />

          {draft && (
            <button
              type="button"
              onClick={() =>
                void handleApply()
              }
              disabled={
                !!busy ||
                selectedOperationIDs.length === 0
              }
              style={{
                width: "100%",
                marginTop: "8px",
                padding: "7px 9px",
                borderRadius: "7px",
                border:
                  `1px solid ${C.primary}`,
                background:
                  busy ||
                  selectedOperationIDs.length === 0
                    ? "#F1F5F9"
                    : C.primary,
                color:
                  busy ||
                  selectedOperationIDs.length === 0
                    ? C.textMuted
                    : "#fff",
                fontSize: "9px",
                fontWeight: 700,
                cursor:
                  busy ||
                  selectedOperationIDs.length === 0
                    ? "not-allowed"
                    : "pointer",
              }}
            >
              {busy === "apply"
                ? "正在原子应用…"
                : `确认并原子应用${selectedOperationIDs.length}项`}
            </button>
          )}

          {applied && (
            <div
              style={{
                marginTop: "7px",
                color: C.success,
                fontSize: "9px",
                fontWeight: 600,
                lineHeight: 1.5,
              }}
            >
              ✓ 此方案已经完成一次原子确认，
              不能再次应用。
              问题、关系和问题组的最新状态以后端数据库为准。
            </div>
          )}
        </>
      )}

      {error && (
        <ImpactFeedback
          type="error"
          content={error}
        />
      )}

      {message && (
        <ImpactFeedback
          type="success"
          content={message}
        />
      )}
    </section>
  );
}

function ImpactFeedback({
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
        padding: "7px 8px",
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

const miniButtonStyle = {
  padding: "2px 6px",
  borderRadius: "999px",
  border:
    `1px solid ${C.border}`,
  background: "#fff",
  color: C.textSec,
  fontSize: "8px",
  cursor: "pointer",
} as const;
