/**
 * RefinePanelVersionHistory.tsx
 *
 * 页面历史、回退与版本对比。
 *
 * 本组件完全独立于整改项状态：
 *   - 读取当前页历史版本；
 *   - 回退到指定历史版；
 *   - 并排查看历史版与当前版；
 *   - refreshKey变化时，如面板已打开则重新加载。
 *
 * 所有对比操作均只读。
 */

import { useCallback, useEffect, useState } from "react";

import {
  getCoursewarePages,
  getPageVersionDetail,
  listPageVersions,
  rollbackPage,
} from "@/api/coursewares";
import type { PageVersionEntry } from "@/api/coursewares";

import { injectPreviewMode } from "./previewInject";
import { C, CW_HEIGHT, CW_WIDTH } from "./workshopConstants";

interface Props {
  coursewareId: string;
  pageNum: number;
  refreshKey: number;
  discussionActive: boolean;
  voiceActive: boolean;
  onPageUpdated: (pageNum: number, html: string) => void;
  onMessage: (message: string) => void;
}

interface CompareState {
  open: boolean;
  loading: boolean;
  error: string;
  versionNo: number;
  sourceLabel: string;
  historyHTML: string;
  currentHTML: string;
  mode: "render" | "code";
}

const EMPTY_COMPARE: CompareState = {
  open: false,
  loading: false,
  error: "",
  versionNo: 0,
  sourceLabel: "",
  historyHTML: "",
  currentHTML: "",
  mode: "render",
};

export default function RefinePanelVersionHistory({
  coursewareId,
  pageNum,
  refreshKey,
  discussionActive,
  voiceActive,
  onPageUpdated,
  onMessage,
}: Props) {
  const [showVersions, setShowVersions] = useState(false);
  const [versions, setVersions] = useState<PageVersionEntry[]>([]);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [rollbackingID, setRollbackingID] = useState("");
  const [compare, setCompare] = useState<CompareState>(EMPTY_COMPARE);

  const loadVersions = useCallback(async (): Promise<void> => {
    if (!coursewareId || pageNum <= 0) {
      return;
    }

    setVersionsLoading(true);

    try {
      const result = await listPageVersions(coursewareId, pageNum);
      setVersions(result.versions || []);
    } catch (cause) {
      onMessage(
        `❌ 加载历史版本失败: ${
          cause instanceof Error ? cause.message : "未知错误"
        }`,
      );
    } finally {
      setVersionsLoading(false);
    }
  }, [coursewareId, onMessage, pageNum]);

  useEffect(() => {
    setShowVersions(false);
    setVersions([]);
    setRollbackingID("");
    setCompare(EMPTY_COMPARE);
  }, [coursewareId, pageNum]);

  useEffect(() => {
    if (showVersions && refreshKey > 0) {
      void loadVersions();
    }
  }, [loadVersions, refreshKey, showVersions]);

  const handleToggleVersions = (): void => {
    const next = !showVersions;

    setShowVersions(next);

    if (next) {
      void loadVersions();
    }
  };

  const handleRollback = async (
    version: PageVersionEntry,
  ): Promise<void> => {
    if (!coursewareId || pageNum <= 0 || rollbackingID) {
      return;
    }

    if (
      !window.confirm(
        `确定将第 ${pageNum} 页回退到` +
          `【${version.source_label} · 第${version.version_no}版】吗？\n\n` +
          "回退不会丢失当前内容：系统会先保存当前这一版，之后仍可以再退回来。",
      )
    ) {
      return;
    }

    setRollbackingID(version.id);
    onMessage(`↩️ 正在回退第 ${pageNum} 页...`);

    try {
      const result = await rollbackPage(
        coursewareId,
        pageNum,
        version.id,
      );

      if (result.html_content) {
        onPageUpdated(pageNum, result.html_content);
      }

      onMessage(`✅ ${result.message}`);
      await loadVersions();
    } catch (cause) {
      onMessage(
        `❌ 回退失败: ${
          cause instanceof Error ? cause.message : "未知错误"
        }`,
      );
    } finally {
      setRollbackingID("");
    }
  };

  const handleOpenCompare = async (
    version: PageVersionEntry,
  ): Promise<void> => {
    if (!coursewareId || pageNum <= 0) {
      return;
    }

    setCompare({
      ...EMPTY_COMPARE,
      open: true,
      loading: true,
      versionNo: version.version_no,
      sourceLabel: version.source_label,
    });

    try {
      const [detail, pages] = await Promise.all([
        getPageVersionDetail(coursewareId, pageNum, version.id),
        getCoursewarePages(coursewareId),
      ]);

      const currentPage = (pages || []).find(
        (page) => page.page_number === pageNum,
      );

      setCompare((previous) => ({
        ...previous,
        loading: false,
        error: "",
        versionNo: detail.version_no || version.version_no,
        sourceLabel: detail.source_label || version.source_label,
        historyHTML: detail.html_content || "",
        currentHTML: currentPage?.html_content || "",
      }));
    } catch (cause) {
      setCompare((previous) => ({
        ...previous,
        loading: false,
        error:
          "加载对比内容失败: " +
          (cause instanceof Error ? cause.message : "未知错误"),
      }));
    }
  };

  return (
    <>
      <button
        type="button"
        onClick={handleToggleVersions}
        disabled={pageNum <= 0 || voiceActive}
        title={
          pageNum <= 0
            ? "请先在上方预览区选中页"
            : "查看本页历史版本，可回退或与当前内容对比"
        }
        style={{
          padding: "8px 16px",
          borderRadius: 8,
          border: `1px solid ${showVersions ? "#2563EB" : C.border}`,
          background: showVersions ? "#EFF6FF" : "#FFFFFF",
          color: pageNum > 0 ? "#2563EB" : "#9CA3AF",
          fontSize: 13,
          fontWeight: 600,
          cursor: pageNum > 0 ? "pointer" : "default",
          whiteSpace: "nowrap",
        }}
      >
        📜 历史版本{showVersions ? " ▲" : " ▼"}
      </button>

      {showVersions && (
        <div style={versionPanelStyle}>
          <div style={versionTitleStyle}>
            📜 第 {pageNum} 页历史版本
            <span style={versionHelpStyle}>
              （记录修改前的旧版，最多保留近20版；可以对比或回退）
            </span>
          </div>

          {versionsLoading ? (
            <div style={emptyStyle}>⏳ 加载中...</div>
          ) : versions.length === 0 ? (
            <div style={emptyStyle}>
              暂无历史版本，修改页面后会自动生成。
            </div>
          ) : (
            <div style={versionListStyle}>
              {versions.map((version) => (
                <div key={version.id} style={versionRowStyle}>
                  <span style={versionBadgeStyle}>
                    第 {version.version_no} 版
                  </span>

                  <span style={versionSourceStyle}>
                    {version.source_label}
                  </span>

                  <span style={versionTimeStyle}>
                    {formatVersionTime(version.created_at)}
                  </span>

                  {version.note ? (
                    <span
                      style={versionNoteStyle}
                      title={version.note}
                    >
                      {version.note}
                    </span>
                  ) : null}

                  <div style={{ flex: 1 }} />

                  <button
                    type="button"
                    onClick={() => void handleOpenCompare(version)}
                    disabled={!!rollbackingID || compare.loading}
                    style={compareButtonStyle}
                  >
                    👁 对比当前版
                  </button>

                  <button
                    type="button"
                    onClick={() => void handleRollback(version)}
                    disabled={!!rollbackingID || discussionActive}
                    style={rollbackButtonStyle(
                      rollbackingID === version.id,
                    )}
                  >
                    {rollbackingID === version.id
                      ? "↩️ 回退中..."
                      : "↩️ 回退到此版"}
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {compare.open && (
        <VersionCompareModal
          pageNum={pageNum}
          state={compare}
          onSwitchMode={(mode) =>
            setCompare((previous) => ({
              ...previous,
              mode,
            }))
          }
          onClose={() => setCompare(EMPTY_COMPARE)}
        />
      )}
    </>
  );
}

function formatVersionTime(iso: string): string {
  if (!iso) {
    return "";
  }

  const date = new Date(iso);

  if (Number.isNaN(date.getTime())) {
    return iso;
  }

  const pad = (value: number) =>
    value < 10 ? `0${value}` : String(value);

  return (
    `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ` +
    `${pad(date.getHours())}:${pad(date.getMinutes())}`
  );
}

interface CompareModalProps {
  pageNum: number;
  state: CompareState;
  onSwitchMode: (mode: "render" | "code") => void;
  onClose: () => void;
}

function VersionCompareModal({
  pageNum,
  state,
  onSwitchMode,
  onClose,
}: CompareModalProps) {
  const renderIframe = (
    html: string,
    label: string,
    accent: string,
  ) => {
    if (!html.trim()) {
      return (
        <div style={emptyCompareStyle}>
          该版本无可渲染内容
        </div>
      );
    }

    return (
      <div style={compareColumnStyle}>
        <div
          style={{
            ...compareLabelStyle,
            color: accent,
          }}
        >
          {label}
        </div>

        <div
          style={{
            ...compareFrameStyle,
            border: `2px solid ${accent}`,
          }}
        >
          <div
            style={compareCanvasStyle}
            ref={(element) => {
              if (!element) {
                return;
              }

              const parent = element.parentElement;

              if (!parent) {
                return;
              }

              const setScale = () => {
                const scale = Math.min(
                  parent.clientWidth / CW_WIDTH,
                  parent.clientHeight / CW_HEIGHT,
                );

                element.style.setProperty(
                  "--cmp-scale",
                  String(scale > 0 ? scale : 0.1),
                );
              };

              setScale();
              window.requestAnimationFrame(setScale);
            }}
          >
            <iframe
              title={label}
              srcDoc={injectPreviewMode(html)}
              sandbox="allow-scripts"
              style={{
                width: CW_WIDTH,
                height: CW_HEIGHT,
                border: "none",
                display: "block",
              }}
            />
          </div>
        </div>
      </div>
    );
  };

  const renderCode = (
    html: string,
    label: string,
    accent: string,
  ) => (
    <div style={compareColumnStyle}>
      <div
        style={{
          ...compareLabelStyle,
          color: accent,
        }}
      >
        {label}
      </div>

      <pre
        style={{
          ...compareCodeStyle,
          border: `2px solid ${accent}`,
        }}
      >
        {html || "（该版本无内容）"}
      </pre>
    </div>
  );

  return (
    <div onClick={onClose} style={modalOverlayStyle}>
      <div
        onClick={(event) => event.stopPropagation()}
        style={modalStyle}
      >
        <div style={modalHeaderStyle}>
          <div style={modalTitleStyle}>
            🆚 第 {pageNum} 页版本对比

            {state.versionNo > 0 && (
              <span style={modalVersionStyle}>
                历史版：{state.sourceLabel} · 第{state.versionNo}版
              </span>
            )}
          </div>

          <div style={{ flex: 1 }} />

          <div style={switchStyle}>
            <button
              type="button"
              onClick={() => onSwitchMode("render")}
              style={switchButtonStyle(state.mode === "render")}
            >
              🖼 渲染对比
            </button>

            <button
              type="button"
              onClick={() => onSwitchMode("code")}
              style={switchButtonStyle(state.mode === "code")}
            >
              {"</> 源码对比"}
            </button>
          </div>

          <button
            type="button"
            onClick={onClose}
            style={closeButtonStyle}
          >
            ✕ 关闭
          </button>
        </div>

        <div style={modalBodyStyle}>
          {state.loading ? (
            <div style={modalCenterStyle}>
              ⏳ 正在加载两版内容...
            </div>
          ) : state.error ? (
            <div
              style={{
                ...modalCenterStyle,
                color: "#DC2626",
              }}
            >
              ❌ {state.error}
            </div>
          ) : (
            <div style={compareBodyStyle}>
              {state.mode === "render" ? (
                <>
                  {renderIframe(
                    state.historyHTML,
                    `📜 历史版（第${state.versionNo}版 · ${state.sourceLabel}）`,
                    "#7C3AED",
                  )}
                  {renderIframe(
                    state.currentHTML,
                    "✨ 当前版（现在显示的内容）",
                    "#059669",
                  )}
                </>
              ) : (
                <>
                  {renderCode(
                    state.historyHTML,
                    `📜 历史版（第${state.versionNo}版 · ${state.sourceLabel}）`,
                    "#7C3AED",
                  )}
                  {renderCode(
                    state.currentHTML,
                    "✨ 当前版（现在显示的内容）",
                    "#059669",
                  )}
                </>
              )}
            </div>
          )}
        </div>

        <div style={modalFooterStyle}>
          左侧为历史版、右侧为当前版。对比为只读，不会修改任何内容；
          如需换回历史内容，请关闭本窗后点击“回退到此版”。
        </div>
      </div>
    </div>
  );
}

function rollbackButtonStyle(active: boolean) {
  return {
    padding: "5px 14px",
    borderRadius: 6,
    border: "1px solid #2563EB",
    background: active ? "#DBEAFE" : "#FFFFFF",
    color: "#2563EB",
    fontSize: 12,
    fontWeight: 600,
    cursor: active ? "default" : "pointer",
    whiteSpace: "nowrap",
  } as const;
}

function switchButtonStyle(active: boolean) {
  return {
    padding: "6px 14px",
    borderRadius: 6,
    border: "none",
    fontSize: 13,
    fontWeight: 600,
    cursor: "pointer",
    background: active ? "#FFFFFF" : "transparent",
    color: active ? "#7C3AED" : "#6B7280",
    boxShadow: active
      ? "0 1px 3px rgba(0,0,0,0.12)"
      : "none",
  } as const;
}

const versionPanelStyle = {
  flexBasis: "100%",
  width: "100%",
  marginTop: 2,
  padding: "12px 14px",
  borderRadius: 8,
  border: `1px solid ${C.border}`,
  background: "#FFFFFF",
  boxSizing: "border-box",
} as const;

const versionTitleStyle = {
  fontSize: 12,
  fontWeight: 600,
  color: C.textPrimary,
  marginBottom: 8,
} as const;

const versionHelpStyle = {
  fontWeight: 400,
  color: "#9CA3AF",
  marginLeft: 8,
} as const;

const emptyStyle = {
  fontSize: 13,
  color: "#9CA3AF",
  padding: "12px 0",
} as const;

const versionListStyle = {
  display: "flex",
  flexDirection: "column",
  gap: 8,
  maxHeight: 280,
  overflowY: "auto",
} as const;

const versionRowStyle = {
  display: "flex",
  alignItems: "center",
  gap: 10,
  padding: "8px 10px",
  borderRadius: 6,
  border: `1px solid ${C.border}`,
  background: "#FAFAFA",
  flexWrap: "wrap",
} as const;

const versionBadgeStyle = {
  padding: "2px 8px",
  borderRadius: 4,
  background: "#EEF2FF",
  color: "#4F46E5",
  fontSize: 12,
  fontWeight: 600,
  whiteSpace: "nowrap",
} as const;

const versionSourceStyle = {
  fontSize: 13,
  color: C.textPrimary,
  whiteSpace: "nowrap",
} as const;

const versionTimeStyle = {
  fontSize: 12,
  color: "#9CA3AF",
  whiteSpace: "nowrap",
} as const;

const versionNoteStyle = {
  fontSize: 12,
  color: "#6B7280",
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
} as const;

const compareButtonStyle = {
  padding: "5px 12px",
  borderRadius: 6,
  border: "1px solid #7C3AED",
  background: "#FFFFFF",
  color: "#7C3AED",
  fontSize: 12,
  fontWeight: 600,
  cursor: "pointer",
  whiteSpace: "nowrap",
} as const;

const modalOverlayStyle = {
  position: "fixed",
  inset: 0,
  zIndex: 99990,
  background: "rgba(15,23,42,0.72)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  padding: 24,
} as const;

const modalStyle = {
  width: "96vw",
  height: "92vh",
  background: "#FFFFFF",
  borderRadius: 14,
  display: "flex",
  flexDirection: "column",
  overflow: "hidden",
  boxShadow: "0 20px 60px rgba(0,0,0,0.4)",
} as const;

const modalHeaderStyle = {
  display: "flex",
  alignItems: "center",
  gap: 14,
  padding: "14px 20px",
  borderBottom: `1px solid ${C.border}`,
  flexWrap: "wrap",
} as const;

const modalTitleStyle = {
  fontSize: 15,
  fontWeight: 700,
  color: C.textPrimary,
} as const;

const modalVersionStyle = {
  marginLeft: 10,
  fontSize: 13,
  fontWeight: 500,
  color: "#7C3AED",
} as const;

const switchStyle = {
  display: "flex",
  gap: 6,
  background: "#F3F4F6",
  borderRadius: 8,
  padding: 3,
} as const;

const closeButtonStyle = {
  padding: "6px 16px",
  borderRadius: 8,
  border: `1px solid ${C.border}`,
  background: "#FFFFFF",
  color: C.textSecondary,
  fontSize: 14,
  fontWeight: 600,
  cursor: "pointer",
} as const;

const modalBodyStyle = {
  flex: 1,
  padding: "16px 20px",
  overflow: "hidden",
  display: "flex",
  flexDirection: "column",
} as const;

const modalCenterStyle = {
  flex: 1,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  color: "#6B7280",
  fontSize: 15,
} as const;

const compareBodyStyle = {
  flex: 1,
  display: "flex",
  gap: 16,
  minHeight: 0,
} as const;

const compareColumnStyle = {
  flex: 1,
  display: "flex",
  flexDirection: "column",
  minWidth: 0,
} as const;

const compareLabelStyle = {
  fontSize: 13,
  fontWeight: 700,
  marginBottom: 6,
  textAlign: "center",
} as const;

const emptyCompareStyle = {
  flex: 1,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  color: "#9CA3AF",
  fontSize: 14,
  background: "#F9FAFB",
  borderRadius: 8,
} as const;

const compareFrameStyle = {
  flex: 1,
  position: "relative",
  overflow: "hidden",
  borderRadius: 8,
  background: "#FFFFFF",
} as const;

const compareCanvasStyle = {
  position: "absolute",
  top: 0,
  left: "50%",
  width: CW_WIDTH,
  height: CW_HEIGHT,
  transform: "translateX(-50%) scale(var(--cmp-scale))",
  transformOrigin: "top center",
} as const;

const compareCodeStyle = {
  flex: 1,
  margin: 0,
  padding: "12px 14px",
  borderRadius: 8,
  background: "#1E1E1E",
  color: "#D4D4D4",
  fontSize: 12,
  lineHeight: 1.5,
  overflow: "auto",
  whiteSpace: "pre-wrap",
  wordBreak: "break-all",
  fontFamily: "Menlo, Consolas, monospace",
} as const;

const modalFooterStyle = {
  padding: "10px 20px",
  borderTop: `1px solid ${C.border}`,
  fontSize: 12,
  color: "#9CA3AF",
  textAlign: "center",
} as const;
