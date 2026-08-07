/**
 * CWReviewWorkbenchHeader.tsx
 *
 * 课件正式审核工作台顶部导航。
 *
 * 负责：
 *   - 返回审核列表；
 *   - 展示L1或L2审核级别；
 *   - 展示课件标题、学科、年级和页数；
 *   - 在课件存在来源教案时打开教案对照。
 */

const C = {
  primary: "#F59E0B",
  danger: "#EF4444",
  text: "#1F2937",
  textSec: "#6B7280",
  border: "#F3F4F6",
  card: "#FFFFFF",
};

export interface CWReviewWorkbenchHeaderProps {
  level: number;
  coursewareTitle: string;
  subject: string;
  grade: string;
  pageCount: number;
  hasLessonPlan: boolean;
  onBack: () => void;
  onOpenLessonPlan: () => void;
}

export default function CWReviewWorkbenchHeader({
  level,
  coursewareTitle,
  subject,
  grade,
  pageCount,
  hasLessonPlan,
  onBack,
  onOpenLessonPlan,
}: CWReviewWorkbenchHeaderProps) {
  const levelColor = level === 1 ? C.primary : C.danger;
  const levelLabel =
    level === 1 ? "📋 L1 教研组审核" : "🏫 L2 学校审核";

  return (
    <div
      style={{
        height: "48px",
        background: C.card,
        borderBottom: `1px solid ${C.border}`,
        display: "flex",
        alignItems: "center",
        padding: "0 20px",
        gap: "12px",
        flexShrink: 0,
        boxShadow: "0 1px 4px rgba(0,0,0,0.06)",
      }}
    >
      <button
        type="button"
        onClick={onBack}
        style={{
          display: "flex",
          alignItems: "center",
          gap: "5px",
          padding: "4px 8px",
          border: "none",
          borderRadius: "6px",
          background: "none",
          color: C.textSec,
          fontSize: "13px",
          cursor: "pointer",
        }}
      >
        ← 返回列表
      </button>

      <div
        style={{
          width: "1px",
          height: "16px",
          background: C.border,
        }}
      />

      <span
        style={{
          padding: "3px 10px",
          borderRadius: "8px",
          background: `${levelColor}15`,
          color: levelColor,
          fontSize: "12px",
          fontWeight: 600,
          flexShrink: 0,
        }}
      >
        {levelLabel}
      </span>

      <div
        style={{
          minWidth: 0,
          flex: 1,
          overflow: "hidden",
          color: C.text,
          fontSize: "14px",
          fontWeight: 600,
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
        }}
      >
        {coursewareTitle}
      </div>

      {hasLessonPlan && (
        <button
          type="button"
          onClick={onOpenLessonPlan}
          style={{
            padding: "6px 11px",
            borderRadius: "7px",
            border: `1px solid ${C.primary}`,
            background: `${C.primary}0E`,
            color: C.primary,
            fontSize: "12px",
            fontWeight: 600,
            cursor: "pointer",
            flexShrink: 0,
            whiteSpace: "nowrap",
          }}
        >
          📖 打开原教案对照
        </button>
      )}

      <div
        style={{
          color: C.textSec,
          fontSize: "12px",
          flexShrink: 0,
        }}
      >
        {subject} · {grade} · {pageCount} 页
      </div>
    </div>
  );
}
