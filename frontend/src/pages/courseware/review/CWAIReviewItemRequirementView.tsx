/**
 * CWAIReviewItemRequirementView.tsx
 *
 * 只读展示已经确认的整改要求或修改方案。
 *
 * 主要用于：
 *   - 整改作者阅读审核员确认的要求；
 *   - 已经结束讨论后回看最终文字；
 *   - 避免作者误以为可以改写审核员的要求。
 */

import DiscussionMarkdown from "@/pages/courseware/components/courseware-workshop/DiscussionMarkdown";

import {
  CW_AI_REVIEW_ITEM_COLORS as C,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewItemRequirementViewProps {
  title: string;
  content: string;
  help: string;
}

export default function CWAIReviewItemRequirementView({
  title,
  content,
  help,
}: CWAIReviewItemRequirementViewProps) {
  return (
    <div
      style={{
        padding: "9px",
        borderRadius: "8px",
        border:
          `1px solid ${C.border}`,
        background: "#FAFAFA",
      }}
    >
      <div
        style={{
          marginBottom: "5px",
          color: C.text,
          fontSize: "10px",
          fontWeight: 700,
        }}
      >
        {title}
      </div>

      {content.trim() ? (
        <DiscussionMarkdown
          content={content}
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
        {help}
      </div>
    </div>
  );
}
