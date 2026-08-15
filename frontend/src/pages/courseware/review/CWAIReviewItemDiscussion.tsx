/**
 * CWAIReviewItemDiscussion.tsx
 *
 * 单条问题的当前要求、AI协作和历史记录容器。
 *
 * R-01.1展示边界：
 *   - 当前确认要求/方案优先展示；
 *   - 审核员和自审作者可以继续与AI讨论，但讨论工具默认折叠；
 *   - 正式整改作者不能改写审核员要求；
 *   - 作者“本次执行补充”由独立组件处理，不调用AI、不改变问题状态；
 *   - 讨论历史默认折叠，展开后不制造嵌套滚动区；
 *   - 指令历史使用教师语言，不暴露版本技术字段；
 *   - 目标漂移拆出的新问题沿统一问题状态回调上送，当前问题本身不变。
 */

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import {
  getCWAIReviewItemDiscussion,
  messageCWAIReviewItem,
  type CWAIReviewItem,
  type CWAIReviewItemDiscussion,
  type CWAIReviewItemStatus,
} from "@/api/coursewares";

import VoiceInputButton from "@/components/voice/VoiceInputButton";
import { useProtectedDraft } from "@/hooks/useProtectedDraft";
import { useVoiceDraftInput } from "@/hooks/useVoiceDraftInput";
import { useAuth } from "@/store/auth";

import CWAIReviewExecutionNote from "./CWAIReviewExecutionNote";
import CWAIReviewInstructionVersionPanel from "./CWAIReviewInstructionVersionPanel";
import CWAIReviewItemDiscussionHistory from "./CWAIReviewItemDiscussionHistory";

import {
  type CWAIReviewItemExperience,
  CW_AI_REVIEW_ITEM_COLORS as C,
  resolveCWAIReviewItemExperienceCopy,
} from "./CWAIReviewItemPresentation.shared";

import { useLatestValueRef } from "./useLatestValueRef";

export interface CWAIReviewItemDiscussionProps {
  experience: CWAIReviewItemExperience;
  item: CWAIReviewItem;
  onChanged: (item: CWAIReviewItem) => void;
}

function canContinueWorking(
  status: CWAIReviewItemStatus,
): boolean {
  return (
    status === "detected" ||
    status === "discussing" ||
    status === "confirmed"
  );
}

function canAddExecutionNote(
  item: CWAIReviewItem,
  experience: CWAIReviewItemExperience,
): boolean {
  if (
    experience !== "remediation" ||
    item.source_type !== "formal" ||
    !item.courseware_review_id ||
    !item.feedback_id
  ) {
    return false;
  }

  return (
    item.status === "confirmed" ||
    item.status === "applying" ||
    item.status === "applied" ||
    item.status === "stale" ||
    item.status === "orphaned"
  );
}

export default function CWAIReviewItemDiscussionView({
  experience,
  item,
  onChanged,
}: CWAIReviewItemDiscussionProps) {
  const { user } = useAuth();
  const copy =
    resolveCWAIReviewItemExperienceCopy(experience);

  const [discussion, setDiscussion] =
    useState<CWAIReviewItemDiscussion | null>(null);

  const [finalText, setFinalText] =
    useState(item.confirmed_instruction || "");

  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [discussionOpen, setDiscussionOpen] = useState(false);
  const [error, setError] = useState("");

  const onChangedRef = useLatestValueRef(onChanged);

  const messageDraft = useProtectedDraft({
    userId: user?.id,
    scope: "courseware-review-item",
    resourceId: item.id,
    field: "message",
    initialValue: "",
    maxHistory: 40,
  });

  const currentItem = discussion?.item || item;

  const canEdit =
    experience !== "remediation" &&
    canContinueWorking(currentItem.status);

  const executionNoteAvailable =
    canAddExecutionNote(currentItem, experience);

  const messageInputRef =
    useRef<HTMLTextAreaElement>(null);

  const messageVoice = useVoiceDraftInput({
    value: messageDraft.value,
    setValue: messageDraft.setValue,
    disabled: sending || !canEdit,
    maxDurationSeconds: 120,

    onFinalFocus: (finalValue) => {
      const element = messageInputRef.current;
      if (!element) {
        return;
      }

      element.focus();
      element.setSelectionRange(
        finalValue.length,
        finalValue.length,
      );
    },

    onError: setError,
  });

  const loadDiscussion = useCallback(async () => {
    setLoading(true);
    setError("");

    try {
      const result =
        await getCWAIReviewItemDiscussion(item.id);

      setDiscussion(result);
      onChangedRef.current(result.item);

      const suggested =
        result.suggested_instruction.trim();

      setFinalText(
        suggested ||
          result.item.confirmed_instruction ||
          "",
      );
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : copy.discussionLoadError,
      );
    } finally {
      setLoading(false);
    }
  }, [
    copy.discussionLoadError,
    item.id,
    onChangedRef,
  ]);

  useEffect(() => {
    setDiscussionOpen(false);
    void loadDiscussion();
  }, [loadDiscussion]);

  useEffect(() => {
    if (item.confirmed_instruction) {
      setFinalText(item.confirmed_instruction);
    }
  }, [item.confirmed_instruction]);

  const handleSend = async () => {
    const content = messageDraft.value.trim();

    if (
      !content ||
      sending ||
      messageVoice.isActive ||
      !canEdit
    ) {
      return;
    }

    setSending(true);
    setError("");

    try {
      const result =
        await messageCWAIReviewItem(
          item.id,
          content,
        );

      setDiscussion(result);
      onChangedRef.current(result.item);

      const suggested =
        result.suggested_instruction.trim();

      if (suggested) {
        setFinalText(suggested);
      }

      messageDraft.commit();
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "发送补充信息失败",
      );
    } finally {
      setSending(false);
    }
  };

  const handleConfirmed = (
    result: CWAIReviewItemDiscussion,
  ) => {
    setDiscussion(result);
    setFinalText(result.item.confirmed_instruction);
    onChangedRef.current(result.item);
  };

  const handleRelatedImprovementCreated = (
    createdItem: CWAIReviewItem,
  ) => {
    // 新ID沿统一父层入口上送。
    // 聚焦工作区会因ID不同而保持当前问题，不触发自动前进。
    onChangedRef.current(createdItem);
  };

  const handleExecutionNoteSaved = (
    result: CWAIReviewItemDiscussion,
  ) => {
    setDiscussion(result);
    onChangedRef.current(result.item);
  };

  if (loading) {
    return (
      <div
        style={{
          color: C.textMuted,
          fontSize: "11px",
        }}
      >
        {copy.discussionLoading}
      </div>
    );
  }

  return (
    <>
      <CWAIReviewInstructionVersionPanel
        experience={experience}
        item={currentItem}
        fallbackContent={currentItem.confirmed_instruction}
        draftText={finalText}
        canEdit={canEdit}
        onDraftTextChange={setFinalText}
        onConfirmed={handleConfirmed}
        onRelatedImprovementCreated={
          handleRelatedImprovementCreated
        }
      />

      <CWAIReviewExecutionNote
        key={currentItem.id}
        itemId={currentItem.id}
        available={executionNoteAvailable}
        onSaved={handleExecutionNoteSaved}
      />

      <div
        style={{
          marginTop: "10px",
          paddingTop: "10px",
          borderTop: `1px solid ${C.border}`,
        }}
      >
        <button
          type="button"
          onClick={() =>
            setDiscussionOpen((previous) => !previous)
          }
          aria-expanded={discussionOpen}
          style={{
            padding: "6px 9px",
            borderRadius: "7px",
            border: `1px solid ${C.border}`,
            background: "#FFFFFF",
            color: C.textSec,
            fontSize: "10px",
            fontWeight: 700,
            cursor: "pointer",
          }}
        >
          {discussionOpen
            ? "收起讨论记录"
            : experience === "remediation"
              ? `查看讨论记录 ${discussion?.messages.length || 0}`
              : "更多：讨论记录与 AI 辅助"}
        </button>
      </div>

      {discussionOpen && (
        <div
          style={{
            marginTop: "9px",
            padding: "10px",
            borderRadius: "9px",
            border: `1px solid ${C.border}`,
            background: "#FFFFFF",
          }}
        >
          <CWAIReviewItemDiscussionHistory
            messages={discussion?.messages || []}
            summary={discussion?.summary || ""}
            copy={copy}
          />

          {canEdit && (
            <>
              <textarea
                ref={messageInputRef}
                value={messageDraft.value}
                onChange={(event) =>
                  messageDraft.setValue(event.target.value)
                }
                onKeyDown={(event) => {
                  if (messageDraft.handleKeyDown(event)) {
                    return;
                  }

                  if (
                    event.key === "Enter" &&
                    !event.shiftKey
                  ) {
                    event.preventDefault();
                    void handleSend();
                  }
                }}
                rows={2}
                placeholder={copy.discussionPlaceholder}
                disabled={sending || messageVoice.isActive}
                style={{
                  width: "100%",
                  boxSizing: "border-box",
                  padding: "8px 9px",
                  borderRadius: "7px",
                  border: `1px solid ${C.border}`,
                  resize: "vertical",
                  fontFamily: "inherit",
                  fontSize: "11px",
                  lineHeight: 1.6,
                  outline: "none",
                }}
              />

              <div
                style={{
                  display: "flex",
                  gap: "6px",
                  alignItems: "center",
                  marginTop: "6px",
                }}
              >
                <VoiceInputButton
                  status={messageVoice.status}
                  isSupported={messageVoice.isSupported}
                  elapsedSeconds={messageVoice.elapsedSeconds}
                  disabled={sending || !canEdit}
                  error={messageVoice.error}
                  onStart={messageVoice.begin}
                  onStop={messageVoice.stop}
                  onCancel={messageVoice.cancel}
                />

                <button
                  type="button"
                  onClick={() => void handleSend()}
                  disabled={
                    sending ||
                    messageVoice.isActive ||
                    !messageDraft.value.trim()
                  }
                  style={{
                    flex: 1,
                    padding: "7px",
                    borderRadius: "7px",
                    border: "none",
                    background:
                      sending ||
                      messageVoice.isActive ||
                      !messageDraft.value.trim()
                        ? "#CBD5E1"
                        : C.primary,
                    color: "#FFFFFF",
                    fontSize: "11px",
                    fontWeight: 700,
                    cursor:
                      sending ||
                      messageVoice.isActive ||
                      !messageDraft.value.trim()
                        ? "not-allowed"
                        : "pointer",
                  }}
                >
                  {sending
                    ? copy.discussionSendingAction
                    : copy.discussionSendAction}
                </button>
              </div>

              <div
                style={{
                  marginTop: "5px",
                  color:
                    messageVoice.status === "error"
                      ? C.danger
                      : messageVoice.isActive
                        ? C.primary
                        : C.textMuted,
                  fontSize: "9px",
                  lineHeight: 1.5,
                }}
              >
                {messageVoice.statusText ||
                  "可以用语音补充想法；讨论内容不会自动确认或执行"}
              </div>
            </>
          )}
        </div>
      )}

      {!canEdit &&
        currentItem.status === "dismissed" &&
        !currentItem.confirmed_instruction && (
          <div
            style={{
              marginTop: "8px",
              padding: "7px 9px",
              borderRadius: "7px",
              background: "#F8FAFC",
              color: C.textSec,
              fontSize: "10px",
              lineHeight: 1.6,
            }}
          >
            {experience === "review"
              ? "这条问题本次不退回给作者。恢复审核后可以继续完善修改要求。"
              : "这条问题本次暂不调整。恢复后可以继续完善修改方案。"}
          </div>
        )}

      {error && (
        <div
          style={{
            marginTop: "8px",
            padding: "7px 9px",
            borderRadius: "7px",
            background: "#FEF2F2",
            color: C.danger,
            fontSize: "10px",
            lineHeight: 1.6,
          }}
        >
          {error}
        </div>
      )}
    </>
  );
}
