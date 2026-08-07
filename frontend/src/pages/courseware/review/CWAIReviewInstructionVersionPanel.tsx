/**
 * CWAIReviewInstructionVersionPanel.tsx
 *
 * 单条整改项的当前草稿、当前确认版本和历史版本面板。
 *
 * 本组件负责：
 *   1. 显示最新确认版本号、确认时间和状态；
 *   2. 明确提示当前编辑草稿是否偏离确认版本；
 *   3. 使用最近读取的当前版本ID执行乐观并发确认；
 *   4. “保存为新版并确认”成功后刷新版本历史；
 *   5. 正式整改作者以只读方式查看实际交付版本。
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
  type CWAIReviewItemDiscussion,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  type CWAIReviewItemExperience,
  CW_AI_REVIEW_ITEM_COLORS as C,
  resolveCWAIReviewItemExperienceCopy,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewInstructionVersionPanelProps {
  experience:
    CWAIReviewItemExperience;

  itemId: string;

  fallbackContent: string;

  draftText: string;

  canEdit: boolean;

  onDraftTextChange: (
    value: string,
  ) => void;

  onConfirmed: (
    result:
      CWAIReviewItemDiscussion,
  ) => void;
}

const VERSION_STATUS_LABEL:
  Record<
    CWAIReviewInstructionVersion["status"],
    string
  > = {
    draft: "草稿",
    confirmed: "当前有效",
    superseded: "已被新版替代",
    invalid_for_page: "页面已变化，不可执行",
  };

const VERSION_SOURCE_LABEL:
  Record<
    CWAIReviewInstructionVersion["source_type"],
    string
  > = {
    legacy_backfill: "历史确认内容",
    legacy_direct_update: "旧版确认入口",
    manual: "人工编辑",
    ai_candidate: "AI候选",
    global_discussion: "全局讨论候选",
  };

function formatDateTime(
  value: string | null,
): string {
  if (!value) {
    return "";
  }

  const date =
    new Date(value);

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
  itemId,
  fallbackContent,
  draftText,
  canEdit,
  onDraftTextChange,
  onConfirmed,
}: CWAIReviewInstructionVersionPanelProps) {
  const copy =
    resolveCWAIReviewItemExperienceCopy(
      experience,
    );

  const [
    versions,
    setVersions,
  ] = useState<
    CWAIReviewInstructionVersion[]
  >([]);

  const [
    currentVersionId,
    setCurrentVersionId,
  ] = useState("");

  const [loading, setLoading] =
    useState(true);

  const [
    confirming,
    setConfirming,
  ] = useState(false);

  const [
    historyOpen,
    setHistoryOpen,
  ] = useState(false);

  const [error, setError] =
    useState("");

  const [
    successMessage,
    setSuccessMessage,
  ] = useState("");

  const loadVersions =
    useCallback(async () => {
      setLoading(true);

      try {
        const result =
          await listCWAIReviewInstructionVersions(
            itemId,
          );

        setVersions(
          result.versions || [],
        );

        setCurrentVersionId(
          result
            .current_instruction_version_id
            ?.trim() || "",
        );
      } catch (cause) {
        setError(
          cause instanceof Error
            ? cause.message
            : "读取整改指令版本失败",
        );
      } finally {
        setLoading(false);
      }
    }, [
      itemId,
    ]);

  useEffect(() => {
    setError("");
    setSuccessMessage("");
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

  const normalizedDraft =
    draftText.trim();

  const normalizedConfirmed =
    currentVersion
      ?.content
      .trim() || "";

  const draftDiffers =
    !!currentVersion &&
    normalizedDraft !==
      normalizedConfirmed;

  const readOnlyContent =
    currentVersion
      ?.content
      .trim() ||
    fallbackContent.trim();

  const handleConfirm =
    async () => {
      if (
        !canEdit ||
        confirming ||
        !normalizedDraft
      ) {
        return;
      }

      setConfirming(true);
      setError("");
      setSuccessMessage("");

      try {
        const result =
          await confirmCWAIReviewInstructionVersion(
            itemId,
            normalizedDraft,
            currentVersionId,
          );

        onConfirmed(
          result,
        );

        setSuccessMessage(
          currentVersion
            ? `已保存为V${currentVersion.version_no + 1}并确认。旧版本仍保留在历史中。`
            : "首个确认版本已保存。",
        );

        await loadVersions();
      } catch (cause) {
        setError(
          cause instanceof Error
            ? cause.message
            : experience ===
                "review"
              ? "保存新版整改要求失败"
              : "保存新版修改方案失败",
        );

        // 并发确认失败时立即刷新当前版本，
        // 让用户看到另一个窗口已经形成的新版本。
        void loadVersions();
      } finally {
        setConfirming(false);
      }
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
        正在读取确认版本…
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
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: "8px",
          flexWrap: "wrap",
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
              fontSize: "11px",
              fontWeight: 700,
            }}
          >
            {canEdit
              ? copy.finalTextTitle
              : copy.readOnlyRequirementTitle}
          </div>

          {currentVersion ? (
            <div
              style={{
                marginTop: "3px",
                color: C.textSec,
                fontSize: "9px",
                lineHeight: 1.5,
              }}
            >
              当前确认版本 V
              {currentVersion.version_no}
              {currentVersion.confirmed_at
                ? ` · ${formatDateTime(currentVersion.confirmed_at)}`
                : ""}
              {" · "}
              {
                VERSION_STATUS_LABEL[
                  currentVersion.status
                ]
              }
            </div>
          ) : (
            <div
              style={{
                marginTop: "3px",
                color: C.textMuted,
                fontSize: "9px",
                lineHeight: 1.5,
              }}
            >
              尚未形成确认版本
            </div>
          )}
        </div>

        {versions.length > 0 && (
          <button
            type="button"
            onClick={() =>
              setHistoryOpen(
                (previous) =>
                  !previous,
              )
            }
            style={{
              flexShrink: 0,
              padding: "5px 8px",
              borderRadius: "6px",
              border:
                `1px solid ${C.border}`,
              background: "#FFFFFF",
              color: C.primary,
              fontSize: "9px",
              fontWeight: 700,
              cursor: "pointer",
            }}
          >
            {historyOpen
              ? "收起版本历史"
              : `版本历史 ${versions.length}`}
          </button>
        )}
      </div>

      {canEdit ? (
        <>
          <textarea
            value={draftText}
            onChange={(event) =>
              onDraftTextChange(
                event.target.value,
              )
            }
            rows={4}
            placeholder={
              copy.finalTextPlaceholder
            }
            disabled={
              confirming
            }
            style={{
              width: "100%",
              boxSizing:
                "border-box",
              marginTop: "8px",
              padding: "8px 9px",
              borderRadius:
                "7px",
              border:
                `1px solid ${
                  draftDiffers
                    ? "#F59E0B"
                    : C.border
                }`,
              resize: "vertical",
              fontFamily:
                "inherit",
              fontSize: "11px",
              lineHeight: 1.6,
              outline: "none",
              background: "#FFFFFF",
            }}
          />

          {currentVersion && (
            <div
              style={{
                marginTop: "6px",
                padding: "6px 8px",
                borderRadius:
                  "6px",
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
                ? `当前草稿与已确认的V${currentVersion.version_no}不同。只有保存为新版并确认后，新内容才可交付或执行。`
                : `当前草稿与已确认的V${currentVersion.version_no}一致。`}
            </div>
          )}

          <button
            type="button"
            onClick={() =>
              void handleConfirm()
            }
            disabled={
              confirming ||
              !normalizedDraft
            }
            style={{
              width: "100%",
              marginTop: "7px",
              padding: "8px",
              borderRadius:
                "7px",
              border: "none",
              background:
                confirming ||
                !normalizedDraft
                  ? "#CBD5E1"
                  : C.success,
              color: "#FFFFFF",
              fontSize: "11px",
              fontWeight: 700,
              cursor:
                confirming ||
                !normalizedDraft
                  ? "not-allowed"
                  : "pointer",
            }}
          >
            {confirming
              ? "正在保存并确认…"
              : currentVersion
                ? "保存为新版并确认"
                : "保存并确认"}
          </button>

          <div
            style={{
              marginTop: "5px",
              color: C.textMuted,
              fontSize: "9px",
              lineHeight: 1.5,
            }}
          >
            每次确认都会形成连续的新版本；旧版本不可覆盖，可以在下方历史中回看。
          </div>
        </>
      ) : (
        <div
          style={{
            marginTop: "8px",
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
                color: C.textMuted,
                fontSize: "10px",
                lineHeight: 1.6,
              }}
            >
              暂无可执行的文字说明，请联系审核员确认。
            </div>
          )}

          <div
            style={{
              marginTop: "6px",
              color: C.textMuted,
              fontSize: "9px",
              lineHeight: 1.5,
            }}
          >
            {currentVersion
              ? `这是正式交付或当前可见的V${currentVersion.version_no}，内容只读。`
              : copy.readOnlyRequirementHelp}
          </div>
        </div>
      )}

      {historyOpen && (
        <div
          style={{
            marginTop: "10px",
            paddingTop: "8px",
            borderTop:
              `1px solid ${C.border}`,
          }}
        >
          {versions.map(
            (version) => (
              <div
                key={version.id}
                style={{
                  marginTop: "7px",
                  padding: "8px",
                  borderRadius:
                    "7px",
                  border:
                    `1px solid ${
                      version.is_current ||
                      version.id ===
                        currentVersionId
                        ? "#A7F3D0"
                        : C.border
                    }`,
                  background:
                    version.is_current ||
                    version.id ===
                      currentVersionId
                      ? "#F0FDF4"
                      : "#FFFFFF",
                }}
              >
                <div
                  style={{
                    display: "flex",
                    alignItems:
                      "center",
                    gap: "6px",
                    flexWrap: "wrap",
                  }}
                >
                  <span
                    style={{
                      color: C.text,
                      fontSize: "10px",
                      fontWeight: 700,
                    }}
                  >
                    V{version.version_no}
                  </span>

                  <span
                    style={{
                      color:
                        version.status ===
                        "invalid_for_page"
                          ? C.danger
                          : version.status ===
                              "confirmed"
                            ? C.success
                            : C.textMuted,
                      fontSize: "9px",
                      fontWeight: 700,
                    }}
                  >
                    {
                      VERSION_STATUS_LABEL[
                        version.status
                      ]
                    }
                  </span>

                  <span
                    style={{
                      color: C.textMuted,
                      fontSize: "9px",
                    }}
                  >
                    {
                      VERSION_SOURCE_LABEL[
                        version.source_type
                      ]
                    }
                  </span>

                  {version.confirmed_at && (
                    <span
                      style={{
                        marginLeft:
                          "auto",
                        color:
                          C.textMuted,
                        fontSize: "9px",
                      }}
                    >
                      {formatDateTime(
                        version.confirmed_at,
                      )}
                    </span>
                  )}
                </div>

                <div
                  style={{
                    marginTop: "6px",
                    color: C.textSec,
                    fontSize: "10px",
                    lineHeight: 1.6,
                    whiteSpace:
                      "pre-wrap",
                    wordBreak:
                      "break-word",
                  }}
                >
                  {version.content}
                </div>
              </div>
            ),
          )}
        </div>
      )}

      {successMessage && (
        <div
          style={{
            marginTop: "8px",
            padding: "7px 8px",
            borderRadius: "6px",
            background: "#ECFDF5",
            color: C.success,
            fontSize: "9px",
            fontWeight: 600,
            lineHeight: 1.5,
          }}
        >
          {successMessage}
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
