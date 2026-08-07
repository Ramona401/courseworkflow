/**
 * CWAIReviewPanelBoundary.tsx
 *
 * AI课件审核面板的局部错误边界。
 *
 * 目的：
 *   - AI面板发生运行时异常时，只替换当前面板；
 *   - 不让错误冒泡到整个课件路由；
 *   - 批注、历史、课件预览和人工审核决定仍可继续使用；
 *   - 用户点击重新加载时，真正重新挂载面板内部组件，
 *     避免仅清除错误提示但继续沿用异常组件实例。
 *
 * 该边界不吞掉日志，错误仍会写入浏览器控制台。
 */

import {
  Component,
  Fragment,
  type ErrorInfo,
  type ReactNode,
} from "react";

interface CWAIReviewPanelBoundaryProps {
  children: ReactNode;
  resetKey: string;
}

interface CWAIReviewPanelBoundaryState {
  hasError: boolean;
  retryVersion: number;
}

export default class CWAIReviewPanelBoundary extends Component<
  CWAIReviewPanelBoundaryProps,
  CWAIReviewPanelBoundaryState
> {
  state: CWAIReviewPanelBoundaryState = {
    hasError: false,
    retryVersion: 0,
  };

  static getDerivedStateFromError():
    Partial<CWAIReviewPanelBoundaryState> {
    return {
      hasError: true,
    };
  }

  componentDidCatch(
    error: Error,
    info: ErrorInfo,
  ) {
    console.error(
      "[CWAIReviewPanelBoundary]",
      error,
      info,
    );
  }

  componentDidUpdate(
    previous: CWAIReviewPanelBoundaryProps,
  ) {
    if (
      previous.resetKey !==
        this.props.resetKey &&
      this.state.hasError
    ) {
      this.setState(
        (current) => ({
          hasError: false,
          retryVersion:
            current.retryVersion + 1,
        }),
      );
    }
  }

  private handleRetry = () => {
    this.setState(
      (current) => ({
        hasError: false,
        retryVersion:
          current.retryVersion + 1,
      }),
    );
  };

  render() {
    if (!this.state.hasError) {
      return (
        <Fragment
          key={
            this.state.retryVersion
          }
        >
          {this.props.children}
        </Fragment>
      );
    }

    return (
      <div
        style={{
          padding: "18px 14px",
          borderRadius: "10px",
          border:
            "1px solid #FECACA",
          background: "#FEF2F2",
          color: "#991B1B",
        }}
      >
        <div
          style={{
            fontSize: "14px",
            fontWeight: 700,
            marginBottom: "6px",
          }}
        >
          ⚠️ AI审核面板加载异常
        </div>

        <div
          style={{
            fontSize: "12px",
            lineHeight: 1.7,
            marginBottom: "12px",
          }}
        >
          课件预览和人工审核功能未受影响。可重新加载AI面板，已经保存的审核批次不会丢失。
        </div>

        <button
          type="button"
          onClick={
            this.handleRetry
          }
          style={{
            padding: "7px 14px",
            borderRadius: "7px",
            border: "none",
            background: "#DC2626",
            color: "#fff",
            fontSize: "12px",
            fontWeight: 600,
            cursor: "pointer",
          }}
        >
          重新加载AI审核面板
        </button>
      </div>
    );
  }
}
