/**
 * CWAIReviewItemDiscussionHistory.tsx
 *
 * 单条问题的讨论与人工执行记录。
 *
 * 本组件只展示父组件已经明确展开的历史内容，不自行发送消息或保存要求。
 * 父组件负责默认折叠，避免讨论区抢占教师改进卡的主要任务。
 *
 * 作者正式整改的execution-note消息会显示为“本次执行补充”，
 * 其他用户消息仍按当前场景显示教师/审核员称呼。
 */

import {
  parseCWAIReviewJSON,
  type CWAIReviewItemDiscussion,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  CW_AI_REVIEW_ITEM_COLORS as C,
  type CWAIReviewItemExperienceCopy,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewItemDiscussionHistoryProps {
  messages: CWAIReviewItemDiscussion["messages"];
  summary?: string | null;
  copy: CWAIReviewItemExperienceCopy;
}

function formatTime(raw: string | null): string {
  if (!raw) {
    return "";
  }

  try {
    return new Date(raw).toLocaleString("zh-CN");
  } catch {
    return raw;
  }
}

function isExecutionNote(
  metaJSON: string,
): boolean {
  const meta = parseCWAIReviewJSON<Record<string, unknown>>(
    metaJSON,
    {},
  );

  return meta.event === "owner_execution_note";
}

export default function CWAIReviewItemDiscussionHistory({
  messages,
  summary,
  copy,
}: CWAIReviewItemDiscussionHistoryProps) {
  return (
    <>
      {messages.length > 0 ? (
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: "7px",
            marginBottom: "9px",
          }}
        >
          {messages.map((message) => {
            const assistant = message.role === "assistant";
            const system = message.role === "system";
            const executionNote =
              message.role === "user" &&
              isExecutionNote(message.meta_json);

            return (
              <div
                key={message.id}
                style={{
                  alignSelf:
                    assistant || system || executionNote
                      ? "stretch"
                      : "flex-end",
                  maxWidth:
                    assistant || system || executionNote
                      ? "100%"
                      : "88%",
                  padding: "8px 9px",
                  borderRadius: "8px",
                  background: executionNote
                    ? "#FFF7ED"
                    : system
                      ? "#FFFBEB"
                      : assistant
                        ? "#F8FAFC"
                        : "#EEF2FF",
                  border: `1px solid ${
                    executionNote
                      ? "#FED7AA"
                      : system
                        ? "#FDE68A"
                        : assistant
                          ? C.border
                          : "#C7D2FE"
                  }`,
                }}
              >
                <div
                  style={{
                    marginBottom: "3px",
                    color: executionNote
                      ? C.warning
                      : system
                        ? C.warning
                        : assistant
                          ? C.primary
                          : "#4338CA",
                    fontSize: "10px",
                    fontWeight: 700,
                  }}
                >
                  {executionNote
                    ? "本次执行补充"
                    : system
                      ? copy.discussionSystemLabel
                      : assistant
                        ? copy.discussionAssistantLabel
                        : copy.discussionUserLabel}
                </div>

                {assistant ? (
                  <div style={{ color: C.text }}>
                    <DiscussionMarkdown
                      content={message.content}
                      compact
                    />
                  </div>
                ) : (
                  <div
                    style={{
                      color: system ? C.textSec : C.text,
                      whiteSpace: "pre-wrap",
                      fontSize: "11px",
                      lineHeight: 1.65,
                      wordBreak: "break-word",
                    }}
                  >
                    {message.content}
                  </div>
                )}

                <div
                  style={{
                    marginTop: "3px",
                    color: C.textMuted,
                    fontSize: "9px",
                  }}
                >
                  {formatTime(message.created_at)}
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <div
          style={{
            marginBottom: "8px",
            color: C.textMuted,
            fontSize: "10px",
            lineHeight: 1.6,
          }}
        >
          {copy.discussionEmpty}
        </div>
      )}

      {!!summary && (
        <div
          style={{
            marginBottom: "8px",
            padding: "7px 9px",
            borderRadius: "7px",
            background: "#F8FAFC",
            color: C.textSec,
          }}
        >
          <div
            style={{
              marginBottom: "4px",
              color: C.text,
              fontSize: "10px",
              fontWeight: 700,
            }}
          >
            {copy.discussionSummaryTitle}
          </div>

          <DiscussionMarkdown content={summary} compact />
        </div>
      )}
    </>
  );
}
