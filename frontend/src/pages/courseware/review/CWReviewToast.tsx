/**
 * CWReviewToast.tsx
 *
 * 课件审核工作区的轻量短时成功提示。
 *
 * R-01.1只把AI成功结果放到Toast，避免成功文案永久占据教师改进卡首屏。
 * 业务错误不使用本组件，错误继续保留在发生问题的卡片附近。
 */

import {
  useEffect,
} from "react";

export interface CWReviewToastProps {
  message: string;
  onClose: () => void;
}

export default function CWReviewToast({
  message,
  onClose,
}: CWReviewToastProps) {
  useEffect(() => {
    if (!message) {
      return undefined;
    }

    const timer =
      window.setTimeout(
        onClose,
        3000,
      );

    return () => {
      window.clearTimeout(
        timer,
      );
    };
  }, [
    message,
    onClose,
  ]);

  if (!message) {
    return null;
  }

  return (
    <div
      role="status"
      aria-live="polite"
      style={{
        position: "fixed",
        left: "50%",
        bottom: "32px",
        zIndex: 9999,
        maxWidth: "min(560px, calc(100vw - 32px))",
        transform: "translateX(-50%)",
        padding: "11px 18px",
        borderRadius: "10px",
        background: "#1F2937",
        color: "#FFFFFF",
        boxShadow:
          "0 8px 24px rgba(0, 0, 0, 0.15)",
        fontSize: "13px",
        fontWeight: 600,
        lineHeight: 1.5,
        textAlign: "center",
        pointerEvents: "none",
      }}
    >
      ✓ {message}
    </div>
  );
}
