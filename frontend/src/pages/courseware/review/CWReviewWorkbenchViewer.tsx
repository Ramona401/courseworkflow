/**
 * CWReviewWorkbenchViewer.tsx
 *
 * 课件正式审核工作台左侧预览区域。
 *
 * 负责：
 *   - 1920×1080课件等比预览；
 *   - 上一页、下一页和键盘方向键翻页；
 *   - 页面胶片条及批注数量角标；
 *   - 全屏放映；
 *   - 只使用preview模式渲染，不修改课件页面。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import CWFullscreenPreview from "../components/courseware-workshop/CWFullscreenPreview";
import { injectPreviewMode } from "../components/courseware-workshop/previewInject";
import {
  CW_HEIGHT,
  CW_WIDTH,
} from "../components/courseware-workshop/workshopConstants";

const C = {
  primary: "#F59E0B",
  danger: "#EF4444",
  text: "#1F2937",
  textSec: "#6B7280",
  textMuted: "#9CA3AF",
  border: "#F3F4F6",
  borderMid: "#E5E7EB",
  card: "#FFFFFF",
  bg: "#FAFBFC",
};

export interface CWReviewWorkbenchPageData {
  id: string;
  page_number: number;
  title?: string | null;
  html_content?: string | null;
}

export interface CWReviewWorkbenchViewerProps {
  pages: CWReviewWorkbenchPageData[];
  activePage: number;
  annotationCountByPageID: Record<string, number>;
  onActivePageChange: (pageNumber: number) => void;
}

export default function CWReviewWorkbenchViewer({
  pages,
  activePage,
  annotationCountByPageID,
  onActivePageChange,
}: CWReviewWorkbenchViewerProps) {
  const previewBoxRef = useRef<HTMLDivElement>(null);
  const [boxSize, setBoxSize] = useState({
    width: 0,
    height: 0,
  });
  const [showFullscreen, setShowFullscreen] = useState(false);

  const pageIndex = useMemo(
    () => pages.findIndex((page) => page.page_number === activePage),
    [activePage, pages],
  );

  const currentPage = pages[pageIndex] || pages[0];
  const hasPrevious = pageIndex > 0;
  const hasNext =
    pageIndex >= 0 && pageIndex < pages.length - 1;

  const goPrevious = useCallback(() => {
    if (!hasPrevious) {
      return;
    }

    onActivePageChange(pages[pageIndex - 1].page_number);
  }, [
    hasPrevious,
    onActivePageChange,
    pageIndex,
    pages,
  ]);

  const goNext = useCallback(() => {
    if (!hasNext) {
      return;
    }

    onActivePageChange(pages[pageIndex + 1].page_number);
  }, [
    hasNext,
    onActivePageChange,
    pageIndex,
    pages,
  ]);

  useEffect(() => {
    const measure = () => {
      const element = previewBoxRef.current;
      if (!element) {
        return;
      }

      setBoxSize({
        width: element.clientWidth,
        height: element.clientHeight,
      });
    };

    measure();
    window.addEventListener("resize", measure);

    return () => {
      window.removeEventListener("resize", measure);
    };
  }, []);

  useEffect(() => {
    if (showFullscreen) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement;

      if (
        target?.tagName === "INPUT" ||
        target?.tagName === "TEXTAREA"
      ) {
        return;
      }

      if (event.key === "ArrowLeft") {
        goPrevious();
      }

      if (event.key === "ArrowRight") {
        goNext();
      }
    };

    window.addEventListener("keydown", handleKeyDown);

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [
    goNext,
    goPrevious,
    showFullscreen,
  ]);

  if (pages.length === 0) {
    return (
      <div
        style={{
          flex: 1,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          minWidth: 0,
          borderRight: `1px solid ${C.border}`,
          background: C.bg,
          color: C.textMuted,
          fontSize: "14px",
        }}
      >
        该课件暂无已生成页面
      </div>
    );
  }

  const padding = 24;
  const availableWidth = Math.max(
    0,
    boxSize.width - padding * 2,
  );
  const availableHeight = Math.max(
    0,
    boxSize.height - padding * 2,
  );

  const scale =
    availableWidth > 0 && availableHeight > 0
      ? Math.min(
          availableWidth / CW_WIDTH,
          availableHeight / CW_HEIGHT,
        )
      : 0;

  const previewHTML = currentPage?.html_content
    ? injectPreviewMode(currentPage.html_content)
    : "";

  return (
    <>
      <div
        style={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          minWidth: 0,
          borderRight: `1px solid ${C.border}`,
          background: C.bg,
        }}
      >
        <div
          style={{
            height: "46px",
            display: "flex",
            alignItems: "center",
            gap: "10px",
            padding: "0 16px",
            flexShrink: 0,
            background: C.card,
            borderBottom: `1px solid ${C.border}`,
          }}
        >
          <button
            type="button"
            onClick={goPrevious}
            disabled={!hasPrevious}
            style={pageButtonStyle(hasPrevious)}
          >
            ‹ 上一页
          </button>

          <span
            style={{
              color: C.text,
              fontSize: "14px",
              fontWeight: 600,
            }}
          >
            P{currentPage?.page_number}
            {currentPage?.title ? ` — ${currentPage.title}` : ""}
          </span>

          <span
            style={{
              color: C.textMuted,
              fontSize: "12px",
            }}
          >
            {(pageIndex >= 0 ? pageIndex : 0) + 1}/{pages.length}
          </span>

          <button
            type="button"
            onClick={goNext}
            disabled={!hasNext}
            style={pageButtonStyle(hasNext)}
          >
            下一页 ›
          </button>

          <div style={{ flex: 1 }} />

          <span
            style={{
              color: C.textMuted,
              fontSize: "11px",
            }}
          >
            ← → 键翻页
          </span>

          <button
            type="button"
            onClick={() => setShowFullscreen(true)}
            style={{
              padding: "6px 14px",
              borderRadius: "8px",
              border: `1px solid ${C.primary}`,
              background: `${C.primary}0E`,
              color: C.primary,
              fontSize: "13px",
              fontWeight: 600,
              cursor: "pointer",
            }}
          >
            🔍 全屏放映
          </button>
        </div>

        <div
          ref={previewBoxRef}
          style={{
            flex: 1,
            position: "relative",
            overflow: "hidden",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            padding: `${padding}px`,
          }}
        >
          {currentPage && previewHTML && scale > 0 ? (
            <div
              style={{
                width: CW_WIDTH,
                height: CW_HEIGHT,
                overflow: "hidden",
                flexShrink: 0,
                transform: `scale(${scale})`,
                transformOrigin: "center center",
                borderRadius: "6px",
                background: "#fff",
                boxShadow: "0 2px 16px rgba(0,0,0,0.12)",
              }}
            >
              <iframe
                title={`cw-review-p${activePage}`}
                srcDoc={previewHTML}
                scrolling="no"
                sandbox="allow-scripts"
                style={{
                  width: CW_WIDTH,
                  height: CW_HEIGHT,
                  display: "block",
                  overflow: "hidden",
                  border: "none",
                }}
              />
            </div>
          ) : (
            <div
              style={{
                color: C.textMuted,
                fontSize: "13px",
              }}
            >
              该页尚未生成HTML内容
            </div>
          )}
        </div>

        <div
          style={{
            display: "flex",
            gap: "8px",
            padding: "10px 12px",
            overflowX: "auto",
            flexShrink: 0,
            background: C.card,
            borderTop: `1px solid ${C.border}`,
          }}
        >
          {pages.map((page) => {
            const current = page.page_number === activePage;
            const annotationCount =
              annotationCountByPageID[page.id] || 0;

            return (
              <button
                key={page.id}
                type="button"
                onClick={() =>
                  onActivePageChange(page.page_number)
                }
                style={{
                  position: "relative",
                  minWidth: "52px",
                  padding: "6px 14px",
                  flexShrink: 0,
                  borderRadius: "8px",
                  border: current
                    ? `2px solid ${C.primary}`
                    : `1px solid ${C.borderMid}`,
                  background: current
                    ? `${C.primary}12`
                    : "#fff",
                  color: current ? C.primary : C.textSec,
                  fontSize: "12px",
                  fontWeight: current ? 600 : 400,
                  cursor: "pointer",
                }}
              >
                P{page.page_number}

                {annotationCount > 0 && (
                  <span
                    style={{
                      position: "absolute",
                      top: "-6px",
                      right: "-6px",
                      minWidth: "16px",
                      height: "16px",
                      padding: "0 4px",
                      borderRadius: "8px",
                      background: C.danger,
                      color: "#fff",
                      fontSize: "10px",
                      fontWeight: 700,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                    }}
                  >
                    {annotationCount}
                  </span>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {showFullscreen && (
        <CWFullscreenPreview
          pages={pages.map((page) => ({
            page_number: page.page_number,
            title: page.title || "",
            html_content: page.html_content || "",
          }))}
          initialPageNum={activePage}
          codeView={false}
          onToggleCode={() => {}}
          onClose={(finalPage) => {
            if (finalPage) {
              onActivePageChange(finalPage);
            }

            setShowFullscreen(false);
          }}
          onSlideshow={() => {}}
        />
      )}
    </>
  );
}

function pageButtonStyle(
  enabled: boolean,
): React.CSSProperties {
  return {
    padding: "6px 14px",
    borderRadius: "8px",
    border: `1px solid ${C.borderMid}`,
    background: "#fff",
    color: enabled ? C.text : C.textMuted,
    fontSize: "14px",
    opacity: enabled ? 1 : 0.5,
    cursor: enabled ? "pointer" : "not-allowed",
  };
}
