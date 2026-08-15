/**
 * CWAIReviewItemFeedback.tsx
 *
 * 教师改进卡中的非AI状态操作反馈。
 *
 * 独立成纯React组件，避免共享常量/纯函数文件混入组件导出，
 * 保证Vite Fast Refresh边界清晰。
 */

import {
  CW_AI_REVIEW_ITEM_COLORS as C,
} from "./CWAIReviewItemPresentation.shared";

export interface CWAIReviewItemFeedbackProps {
  type: "success" | "error";
  content: string;
}

export default function CWAIReviewItemFeedback({
  type,
  content,
}: CWAIReviewItemFeedbackProps) {
  return (
    <div
      style={{
        marginTop: "10px",
        padding: "10px 12px",
        borderRadius: "8px",
        background:
          type === "success"
            ? "#ECFDF5"
            : "#FEF2F2",
        color:
          type === "success"
            ? C.success
            : C.danger,
        fontSize: "13px",
        fontWeight: 600,
        lineHeight: 1.6,
      }}
    >
      {content}
    </div>
  );
}
