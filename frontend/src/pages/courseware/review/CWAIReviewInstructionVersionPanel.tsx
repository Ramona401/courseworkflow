/**
 * CWAIReviewInstructionVersionPanel.tsx
 *
 * 单条整改项的当前修改要求/方案和不可变历史。
 *
 * 教师界面原则：
 *   - 当前确认内容始终优先；
 *   - 不向教师显示V1/V2、哈希、版本状态或内部来源代码；
 *   - “以前的修改记录”默认折叠；
 *   - 新确认只追加历史，绝不覆盖旧记录；
 *   - 正式整改作者只读审核员交付的当前要求，不可改写；
 *   - 技术上的版本ID仍只在内部用于乐观并发；
 *   - 调整后的文字疑似偏离当前问题时，先明确询问教师意图；
 *   - 正式审核确认后会进入本次修改清单。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import {
  confirmCWAIReviewInstructionVersion,
  listCWAIReviewInstructionVersions,
  type CWAIReviewInstructionVersion,
  type CWAIReviewItem,
  type CWAIReviewItemDiscussion,
} from "@/api/coursewares";

import {
  createCWAIReviewRelatedImprovement,
} from "@/api/coursewares.ai-review-goal-drift";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  shouldWarnCWAIReviewGoalDrift,
} from "./CWAIReviewGoalDrift.shared";
import CWAIReviewGoalDriftPrompt from "./CWAIReviewGoalDriftPrompt";

import {
  type CWAIReviewItemExperience,
  CW_AI_REVIEW_ITEM_COLORS as C,
  resolveCWAIReviewItemExperienceCopy,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewInstructionVersionPanelProps {
  experience: CWAIReviewItemExperience;
  item: CWAIReviewItem;
  fallbackContent: string;
  draftText: string;
  canEdit: boolean;
  onDraftTextChange: (value: string) => void;
  onConfirmed: (result: CWAIReviewItemDiscussion) => void;
  onRelatedImprovementCreated: (item: CWAIReviewItem) => void;
}

function formatDateTime(
  value: string | null,
): string {
  if (!value) {
    return "";
  }

  const date = new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return value;
  }

  return date.toLocaleString(
    "zh-CN",
  );
}

export default function CWAIReviewInstructionVersionPanel({
  experience,
  item,
  fallbackContent,
  draftText,
  canEdit,
  onDraftTextChange,
  onConfirmed,
  onRelatedImprovementCreated,
}: CWAIReviewInstructionVersionPanelProps) {
  const copy =
    resolveCWAIReviewItemExperienceCopy(
      experience,
    );

  const [
    versions,
    setVersions,
  ] =
    useState<
      CWAIReviewInstructionVersion[]
    >([]);

  const [
    currentVersionId,
    setCurrentVersionId,
  ] =
    useState("");

  const [
    loading,
    setLoading,
  ] =
    useState(true);

  const [
    confirming,
    setConfirming,
  ] =
    useState(false);

  const [
    creatingRelated,
    setCreatingRelated,
  ] =
    useState(false);

  const [
    historyOpen,
    setHistoryOpen,
  ] =
    useState(false);

  const [
    goalDriftOpen,
    setGoalDriftOpen,
  ] =
    useState(false);

  const [
    notice,
    setNotice,
  ] =
    useState("");

  const [
    error,
    setError,
  ] =
    useState("");

  const busy =
    confirming ||
    creatingRelated;

  const loadVersions =
    useCallback(
      async () => {
        setLoading(true);

        try {
          const result =
            await listCWAIReviewInstructionVersions(
              item.id,
            );

          setVersions(
            result.versions || [],
          );

          setCurrentVersionId(
            result
              .current_instruction_version_id
              ?.trim() ||
              "",
          );
        } catch (cause) {
          setError(
            cause instanceof Error
              ? cause.message
              : "读取修改要求记录失败",
          );
        } finally {
          setLoading(false);
        }
      },
      [item.id],
    );

  useEffect(() => {
    setError("");
    setNotice("");
    setGoalDriftOpen(false);
    setHistoryOpen(false);

    void loadVersions();
  }, [
    loadVersions,
  ]);

  const currentVersion =
    useMemo(
      () =>
        versions.find(
          (version) =>
            version.id ===
              currentVersionId ||
            version.is_current,
        ) || null,
      [
        currentVersionId,
        versions,
      ],
    );

  const historicalVersions =
    useMemo(
      () =>
        versions.filter(
          (version) =>
            version.id !==
              currentVersion?.id &&
            !version.is_current,
        ),
      [
        currentVersion,
        versions,
      ],
    );

  const normalizedDraft =
    draftText.trim();

  const normalizedConfirmed =
    currentVersion
      ?.content
      .trim() || "";

  const comparisonBaseline =
    normalizedConfirmed ||
    fallbackContent.trim();

  const draftDiffers =
    !!comparisonBaseline &&
    normalizedDraft !==
      comparisonBaseline;

  const readOnlyContent =
    currentVersion
      ?.content
      .trim() ||
    fallbackContent.trim();

  const currentTitle =
    experience === "self"
      ? "当前修改方案"
      : "当前修改要求";

  const draftTitle =
    experience === "self"
      ? "完善当前修改方案"
      : "完善当前修改要求";

  const historyTitle =
    experience === "self"
      ? "以前的修改方案"
      : "以前的修改记录";

  const confirmActionLabel =
    experience === "review"
      ? "确认并加入本次修改清单"
      : "确认当前修改方案";

  const confirmDraft =
    async () => {
      if (
        !canEdit ||
        busy ||
        !normalizedDraft
      ) {
        return;
      }

      setConfirming(true);
      setError("");
      setNotice("");
      setGoalDriftOpen(false);

      try {
        const result =
          await confirmCWAIReviewInstructionVersion(
            item.id,
            normalizedDraft,
            currentVersionId,
          );

        onConfirmed(result);

        await loadVersions();
      } catch (cause) {
        setError(
          cause instanceof Error
            ? cause.message
            : experience ===
                "review"
              ? "确认修改要求失败"
              : "确认修改方案失败",
        );

        // 并发确认失败时重新读取教师当前可见记录，
        // 不在浏览器侧猜测哪个窗口的内容应该生效。
        void loadVersions();
      } finally {
        setConfirming(false);
      }
    };

  const handleConfirm =
    async () => {
      if (
        !canEdit ||
        busy ||
        !normalizedDraft
      ) {
        return;
      }

      setError("");
      setNotice("");

      const shouldWarn =
        shouldWarnCWAIReviewGoalDrift({
          draft:
            normalizedDraft,

          baseline:
            comparisonBaseline,

          item,
        });

      if (shouldWarn) {
        setGoalDriftOpen(
          true,
        );

        return;
      }

      await confirmDraft();
    };

  const handleContinueCurrent =
    async () => {
      if (
        busy ||
        !goalDriftOpen
      ) {
        return;
      }

      setGoalDriftOpen(
        false,
      );

      await confirmDraft();
    };

  const handleCreateRelated =
    async () => {
      if (
        busy ||
        !goalDriftOpen ||
        !normalizedDraft
      ) {
        return;
      }

      setCreatingRelated(
        true,
      );

      setError("");
      setNotice("");

      try {
        const result =
          await createCWAIReviewRelatedImprovement(
            item.id,
            normalizedDraft,
          );

        onRelatedImprovementCreated(
          result.item,
        );

        // 新文字已经成为独立问题。
        // 当前问题恢复仍然有效的确认内容，
        // 不把新问题文字继续留作当前草稿。
        if (
          comparisonBaseline
        ) {
          onDraftTextChange(
            comparisonBaseline,
          );
        }

        setGoalDriftOpen(
          false,
        );

        setNotice(
          "已创建新的独立改进项，当前问题没有改变。",
        );
      } catch (cause) {
        setError(
          cause instanceof Error
            ? cause.message
            : "创建新的独立改进项失败",
        );
      } finally {
        setCreatingRelated(
          false,
        );
      }
    };

  const handleDraftChange =
    (
      value: string,
    ) => {
      onDraftTextChange(
        value,
      );

      // 文字变化后，上一轮意图选择不再对应当前草稿。
      setGoalDriftOpen(
        false,
      );

      setNotice("");
    };

  if (
    loading &&
    versions.length === 0
  ) {
    return (
      <div
        style={{
          marginTop: "10px",
          color: C.textMuted,
          fontSize: "10px",
        }}
      >
        正在读取当前修改要求…
      </div>
    );
  }

  return (
    <div
      style={{
        marginTop: "10px",
        padding: "10px",
        borderRadius: "9px",
        border:
          `1px solid ${C.border}`,
        background: "#FAFAFA",
      }}
    >
      {canEdit ? (
        <>
          <div
            style={{
              color: C.text,
              fontSize: "11px",
              fontWeight: 700,
            }}
          >
            {comparisonBaseline
              ? draftTitle
              : currentTitle}
          </div>

          <textarea
            value={draftText}
            onChange={(
              event,
            ) =>
              handleDraftChange(
                event.target
                  .value,
              )
            }
            rows={4}
            placeholder={
              copy
                .finalTextPlaceholder
            }
            disabled={busy}
            style={{
              width: "100%",
              boxSizing:
                "border-box",
              marginTop: "8px",
              padding: "8px 9px",
              borderRadius: "7px",

              border:
                `1px solid ${
                  draftDiffers
                    ? "#F59E0B"
                    : C.border
                }`,

              resize: "vertical",
              fontFamily: "inherit",
              fontSize: "11px",
              lineHeight: 1.6,
              outline: "none",
              background:
                "#FFFFFF",
            }}
          />

          {comparisonBaseline && (
            <div
              style={{
                marginTop: "6px",
                padding: "6px 8px",
                borderRadius: "6px",

                background:
                  draftDiffers
                    ? "#FFF7ED"
                    : "#ECFDF5",

                color:
                  draftDiffers
                    ? C.warning
                    : C.success,

                fontSize: "9px",
                fontWeight: 600,
                lineHeight: 1.5,
              }}
            >
              {draftDiffers
                ? "这里有尚未确认的调整。确认前，原来的修改要求仍然有效。"
                : "这里与当前已经确认的修改要求一致。"}
            </div>
          )}

          <button
            type="button"
            onClick={() =>
              void handleConfirm()
            }
            disabled={
              busy ||
              !normalizedDraft
            }
            style={{
              width: "100%",
              marginTop: "7px",
              padding: "8px",
              borderRadius: "7px",
              border: "none",

              background:
                busy ||
                !normalizedDraft
                  ? "#CBD5E1"
                  : C.success,

              color: "#FFFFFF",
              fontSize: "11px",
              fontWeight: 700,

              cursor:
                busy ||
                !normalizedDraft
                  ? "not-allowed"
                  : "pointer",
            }}
          >
            {confirming
              ? "正在确认…"
              : creatingRelated
                ? "正在创建新改进项…"
                : confirmActionLabel}
          </button>

          {goalDriftOpen && (
            <CWAIReviewGoalDriftPrompt
              busy={busy}
              creatingRelated={
                creatingRelated
              }
              onContinueCurrent={() =>
                void handleContinueCurrent()
              }
              onCreateRelated={() =>
                void handleCreateRelated()
              }
              onCancel={() =>
                setGoalDriftOpen(
                  false,
                )
              }
            />
          )}

          <div
            style={{
              marginTop: "5px",
              color: C.textMuted,
              fontSize: "9px",
              lineHeight: 1.5,
            }}
          >
            {experience ===
            "review"
              ? "确认后会加入本次修改清单；每次重新确认都会保留以前的记录，不会覆盖已经确认过的内容。"
              : "每次重新确认都会保留以前的记录，不会覆盖已经确认过的内容。"}
          </div>
        </>
      ) : (
        <div>
          <div
            style={{
              color: C.text,
              fontSize: "11px",
              fontWeight: 700,
            }}
          >
            {currentTitle}
          </div>

          <div
            style={{
              marginTop: "8px",
              padding: "8px 9px",
              borderRadius: "7px",
              border:
                `1px solid ${C.border}`,
              background:
                "#FFFFFF",
            }}
          >
            {readOnlyContent ? (
              <DiscussionMarkdown
                content={
                  readOnlyContent
                }
                compact
              />
            ) : (
              <div
                style={{
                  color:
                    C.textMuted,
                  fontSize:
                    "10px",
                  lineHeight:
                    1.6,
                }}
              >
                暂无可执行的文字说明，请联系审核员确认。
              </div>
            )}
          </div>

          <div
            style={{
              marginTop: "6px",
              color: C.textMuted,
              fontSize: "9px",
              lineHeight: 1.5,
            }}
          >
            正式审核已经确认的修改要求保持只读。
            作者可以补充本次执行情况，但不能改写原要求。
          </div>
        </div>
      )}

      {notice && (
        <div
          role="status"
          style={{
            marginTop: "8px",
            padding: "7px 8px",
            borderRadius: "6px",
            background:
              "#ECFDF5",
            color: C.success,
            fontSize: "10px",
            fontWeight: 600,
            lineHeight: 1.5,
          }}
        >
          {notice}
        </div>
      )}

      {historicalVersions.length >
        0 && (
        <div
          style={{
            marginTop: "10px",
            paddingTop: "8px",
            borderTop:
              `1px solid ${C.border}`,
          }}
        >
          <button
            type="button"
            onClick={() =>
              setHistoryOpen(
                (previous) =>
                  !previous,
              )
            }
            aria-expanded={
              historyOpen
            }
            style={{
              padding: "5px 8px",
              borderRadius: "6px",
              border:
                `1px solid ${C.border}`,
              background:
                "#FFFFFF",
              color: C.primary,
              fontSize: "9px",
              fontWeight: 700,
              cursor: "pointer",
            }}
          >
            {historyOpen
              ? `收起${historyTitle}`
              : `${historyTitle} ${historicalVersions.length}`}
          </button>

          {historyOpen && (
            <div
              style={{
                marginTop: "8px",
              }}
            >
              {historicalVersions.map(
                (version) => (
                  <div
                    key={
                      version.id
                    }
                    style={{
                      marginTop:
                        "7px",
                      padding:
                        "8px",
                      borderRadius:
                        "7px",
                      border:
                        `1px solid ${C.border}`,
                      background:
                        "#FFFFFF",
                    }}
                  >
                    <div
                      style={{
                        color:
                          C.textMuted,
                        fontSize:
                          "9px",
                        lineHeight:
                          1.5,
                      }}
                    >
                      {version
                        .confirmed_at
                        ? `以前确认于 ${formatDateTime(
                            version.confirmed_at,
                          )}`
                        : "以前确认的修改记录"}
                    </div>

                    <div
                      style={{
                        marginTop:
                          "6px",
                        color:
                          C.textSec,
                        fontSize:
                          "10px",
                        lineHeight:
                          1.6,
                        whiteSpace:
                          "pre-wrap",
                        wordBreak:
                          "break-word",
                      }}
                    >
                      {
                        version
                          .content
                      }
                    </div>
                  </div>
                ),
              )}
            </div>
          )}
        </div>
      )}

      {error && (
        <div
          style={{
            marginTop: "8px",
            padding: "7px 8px",
            borderRadius: "6px",
            background: "#FEF2F2",
            color: C.danger,
            fontSize: "9px",
            lineHeight: 1.5,
          }}
        >
          {error}
        </div>
      )}
    </div>
  );
}
