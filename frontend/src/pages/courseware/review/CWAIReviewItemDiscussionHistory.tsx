/**
 * CWAIReviewItemDiscussionHistory.tsx
 *
 * 单条问题的处理过程展示。
 *
 * 根据当前使用场景显示不同称呼：
 *   - 审核员看到审核记录；
 *   - 自审作者看到调整记录；
 *   - 整改作者看到审核员留下的说明和整改记录。
 *
 * 本组件只展示已有内容，不发送消息、不保存整改要求。
 */

import type {
  CWAIReviewItemDiscussion,
} from "@/api/coursewares";

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  CW_AI_REVIEW_ITEM_COLORS as C,
  type CWAIReviewItemExperienceCopy,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewItemDiscussionHistoryProps {
  messages:
    CWAIReviewItemDiscussion["messages"];

  summary?: string | null;

  copy:
    CWAIReviewItemExperienceCopy;
}

function formatTime(
  raw: string | null,
): string {
  if (!raw) {
    return "";
  }

  try {
    return new Date(
      raw,
    ).toLocaleString(
      "zh-CN",
    );
  } catch {
    return raw;
  }
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
            maxHeight: "260px",
            overflowY: "auto",
            display: "flex",
            flexDirection: "column",
            gap: "7px",
            marginBottom: "9px",
          }}
        >
          {messages.map(
            (message) => {
              const assistant =
                message.role ===
                "assistant";

              const system =
                message.role ===
                  "system";

              return (
                <div
                  key={message.id}
                  style={{
                    alignSelf:
                      assistant ||
                      system
                        ? "stretch"
                        : "flex-end",
                    maxWidth:
                      assistant ||
                      system
                        ? "100%"
                        : "88%",
                    padding:
                      "8px 9px",
                    borderRadius:
                      "8px",
                    background:
                      system
                        ? "#FFFBEB"
                        : assistant
                          ? "#F8FAFC"
                          : "#EEF2FF",
                    border:
                      `1px solid ${
                        system
                          ? "#FDE68A"
                          : assistant
                            ? C.border
                            : "#C7D2FE"
                      }`,
                  }}
                >
                  <div
                    style={{
                      marginBottom:
                        "3px",
                      color:
                        system
                          ? C.warning
                          : assistant
                            ? C.primary
                            : "#4338CA",
                      fontSize:
                        "10px",
                      fontWeight:
                        700,
                    }}
                  >
                    {system
                      ? copy
                          .discussionSystemLabel
                      : assistant
                        ? copy
                            .discussionAssistantLabel
                        : copy
                            .discussionUserLabel}
                  </div>

                  {assistant ? (
                    <div
                      style={{
                        color: C.text,
                      }}
                    >
                      <DiscussionMarkdown
                        content={
                          message.content
                        }
                        compact
                      />
                    </div>
                  ) : (
                    <div
                      style={{
                        color:
                          system
                            ? C.textSec
                            : C.text,
                        whiteSpace:
                          "pre-wrap",
                        fontSize:
                          "11px",
                        lineHeight:
                          1.65,
                        wordBreak:
                          "break-word",
                      }}
                    >
                      {message.content}
                    </div>
                  )}

                  <div
                    style={{
                      marginTop:
                        "3px",
                      color:
                        C.textMuted,
                      fontSize:
                        "9px",
                    }}
                  >
                    {formatTime(
                      message.created_at,
                    )}
                  </div>
                </div>
              );
            },
          )}
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
            {
              copy
                .discussionSummaryTitle
            }
          </div>

          <DiscussionMarkdown
            content={summary}
            compact
          />
        </div>
      )}
    </>
  );
}
