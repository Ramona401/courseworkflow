/**
 * CoursewareLessonPlanContextDrawer.tsx
 *
 * 课件评审与课件工坊共用的来源教案按需对照抽屉。
 *
 * 外层以coursewareId作为key隔离不同课件的网络请求、全屏状态、拖拽状态、
 * 章节焦点和键盘焦点；内部状态与章节匹配继续由共享组合Hook负责。
 */

import {
  type CSSProperties,
  type ReactNode,
} from "react";

import {
  renderMarkdown,
} from "@/pages/lesson-plans/plan-detail/components/planDetailConstants";

import {
  type CoursewareReviewLessonPlanContextRequest,
  useCoursewareLessonPlanDrawerState,
} from "./coursewareReviewLessonPlanContext";

const C = {
  primary: "#4F7BE8",
  primarySoft: "rgba(79,123,232,0.08)",
  danger: "#DC2626",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  card: "#FFFFFF",
  bg: "#F8FAFC",
};

export interface CoursewareLessonPlanContextDrawerProps {
  open: boolean;
  coursewareId: string;
  lessonPlanId?: string | null;
  contextRequest?: CoursewareReviewLessonPlanContextRequest | null;
  openerElement?: HTMLElement | null;
  storageKey: string;
  topOffset?: number;
  zIndex?: number;
  title?: string;
  onClose: () => void;
  onHasLessonPlanChange?: (hasLessonPlan: boolean) => void;
}

export default function CoursewareLessonPlanContextDrawer(
  props: CoursewareLessonPlanContextDrawerProps,
) {
  return (
    <CoursewareLessonPlanContextDrawerState
      key={props.coursewareId}
      {...props}
    />
  );
}

function CoursewareLessonPlanContextDrawerState({
  open,
  coursewareId,
  lessonPlanId,
  contextRequest,
  openerElement,
  storageKey,
  topOffset = 0,
  zIndex = 10020,
  title = "来源教案对照",
  onClose,
  onHasLessonPlanChange,
}: CoursewareLessonPlanContextDrawerProps) {
  const state = useCoursewareLessonPlanDrawerState({
    open,
    coursewareId,
    contextRequest,
    openerElement,
    storageKey,
    onClose,
    onHasLessonPlanChange,
  });

  if (!open) return null;

  const {
    data,
    loading,
    error,
    retryLoad,
    fullScreen,
    setFullScreen,
    compact,
    panelWidth,
    maximumWidth,
    updateWidth,
    dragging,
    hoveringDivider,
    setHoveringDivider,
    panelRef,
    closeButtonRef,
    sectionRefs,
    focusedSectionID,
    documentStructure,
    matchedSection,
    scrollToSection,
    closeDrawer,
    beginResize,
    continueResize,
    finishResize,
    handleDividerKeyDown,
    constants,
  } = state;

  const panelSize: CSSProperties = fullScreen
    ? { width: "100vw", maxWidth: "100vw" }
    : compact
      ? { width: "min(720px, 94vw)", maxWidth: "94vw" }
      : {
          width: `${panelWidth}px`,
          maxWidth: `calc(100vw - ${constants.minimumPrimaryWidth}px)`,
        };

  const showBackdrop = compact || fullScreen;
  const dividerEmphasized = dragging || hoveringDivider;

  const openFullLessonPlan = () => {
    if (!lessonPlanId) return;

    window.open(
      `/lesson-plans/plans/${encodeURIComponent(lessonPlanId)}`,
      "_blank",
      "noopener,noreferrer",
    );
  };

  return (
    <>
      {showBackdrop && (
        <button
          type="button"
          onClick={closeDrawer}
          aria-label="关闭来源教案对照"
          style={{
            position: "fixed",
            top: topOffset,
            right: 0,
            bottom: 0,
            left: 0,
            zIndex,
            padding: 0,
            border: "none",
            background: "rgba(15,23,42,0.34)",
            cursor: "default",
          }}
        />
      )}

      <aside
        ref={panelRef}
        role="dialog"
        aria-modal={showBackdrop}
        aria-label={title}
        style={{
          position: "fixed",
          top: topOffset,
          right: 0,
          bottom: 0,
          zIndex: zIndex + 1,
          display: "flex",
          flexDirection: "column",
          ...panelSize,
          borderLeft: fullScreen ? "none" : `1px solid ${C.border}`,
          background: C.card,
          boxShadow: "-12px 0 36px rgba(15,23,42,0.18)",
          overflow: "visible",
        }}
      >
        {!compact && !fullScreen && (
          <div
            role="separator"
            aria-label="调整教案对照宽度"
            aria-orientation="vertical"
            aria-valuemin={constants.minimumWidth}
            aria-valuemax={maximumWidth}
            aria-valuenow={Math.round(panelWidth)}
            tabIndex={0}
            title="拖动调整宽度；按左右方向键微调；双击或按Enter恢复默认宽度"
            onPointerDown={beginResize}
            onPointerMove={continueResize}
            onPointerUp={finishResize}
            onPointerCancel={finishResize}
            onLostPointerCapture={finishResize}
            onDoubleClick={() => updateWidth(constants.defaultWidth)}
            onKeyDown={handleDividerKeyDown}
            onMouseEnter={() => setHoveringDivider(true)}
            onMouseLeave={() => setHoveringDivider(false)}
            style={{
              position: "absolute",
              top: 0,
              bottom: 0,
              left: -constants.dividerWidth,
              width: `${constants.dividerWidth}px`,
              zIndex: 2,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              cursor: "col-resize",
              touchAction: "none",
              outline: "none",
              background: dividerEmphasized
                ? "rgba(79,123,232,0.06)"
                : "transparent",
            }}
          >
            <span
              style={{
                width: dividerEmphasized ? "3px" : "2px",
                height: dividerEmphasized ? "56px" : "40px",
                borderRadius: "999px",
                background: dividerEmphasized ? C.primary : "#D1D5DB",
              }}
            />
          </div>
        )}

        <header
          style={{
            flexShrink: 0,
            padding: "14px 18px",
            borderBottom: `1px solid ${C.border}`,
            background:
              "linear-gradient(135deg,rgba(79,123,232,0.08),rgba(99,102,241,0.04))",
          }}
        >
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "10px",
            }}
          >
            <div style={{ minWidth: 0, flex: 1 }}>
              <div
                style={{
                  color: C.text,
                  fontSize: "18px",
                  fontWeight: 700,
                  lineHeight: 1.4,
                }}
              >
                {title}
              </div>

              <div
                style={{
                  marginTop: "3px",
                  overflow: "hidden",
                  color: C.textSec,
                  fontSize: "13px",
                  lineHeight: 1.5,
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
              >
                {data?.title ||
                  (loading ? "正在读取来源教案…" : "按需对照课件与教案")}
              </div>
            </div>

            {lessonPlanId && (
              <button
                type="button"
                onClick={openFullLessonPlan}
                style={headerButtonStyle}
              >
                新标签页
              </button>
            )}

            <button
              type="button"
              onClick={() => setFullScreen((previous) => !previous)}
              style={headerButtonStyle}
            >
              {fullScreen ? "退出全屏" : "全屏对照"}
            </button>

            <button
              ref={closeButtonRef}
              type="button"
              onClick={closeDrawer}
              aria-label="关闭来源教案对照"
              style={{
                ...headerButtonStyle,
                width: "36px",
                padding: 0,
                fontSize: "20px",
                lineHeight: 1,
              }}
            >
              ×
            </button>
          </div>

          {contextRequest && (
            <div
              style={{
                marginTop: "10px",
                padding: "9px 11px",
                borderRadius: "8px",
                background: C.primarySoft,
                color: C.textSec,
                fontSize: "13px",
                lineHeight: 1.6,
              }}
            >
              <strong style={{ color: C.primary }}>当前问题：</strong>{" "}
              {contextRequest.pageNumber > 0
                ? `P${contextRequest.pageNumber} · `
                : "整课 · "}
              {contextRequest.issueTitle ||
                contextRequest.issueDescription ||
                "未填写问题标题"}
            </div>
          )}
        </header>

        <main
          style={{
            flex: 1,
            minHeight: 0,
            overflowY: "auto",
            padding: "18px 22px 30px",
            background: C.bg,
          }}
        >
          {loading && <DrawerState>正在读取来源教案…</DrawerState>}

          {error && !loading && (
            <div
              style={{
                padding: "14px",
                border: "1px solid #FECACA",
                borderRadius: "9px",
                background: "#FEF2F2",
                color: C.danger,
                fontSize: "14px",
                lineHeight: 1.6,
              }}
            >
              <div>⚠️ {error}</div>

              <button
                type="button"
                onClick={retryLoad}
                style={{
                  ...headerButtonStyle,
                  marginTop: "10px",
                  color: C.danger,
                }}
              >
                重新读取
              </button>
            </div>
          )}

          {!loading && !error && data && !data.has_lesson_plan && (
            <DrawerState>该课件没有可用于对照的来源教案。</DrawerState>
          )}

          {!loading &&
            !error &&
            data?.has_lesson_plan &&
            !data.content.trim() && (
              <DrawerState>来源教案目前没有可展示的正文。</DrawerState>
            )}

          {!loading &&
            !error &&
            data?.has_lesson_plan &&
            data.content.trim() && (
              <div
                style={{
                  maxWidth: fullScreen ? "1040px" : "100%",
                  margin: fullScreen ? "0 auto" : 0,
                }}
              >
                {contextRequest && (
                  <div
                    style={{
                      position: "sticky",
                      top: 0,
                      zIndex: 1,
                      marginBottom: "12px",
                      padding: "9px 11px",
                      borderRadius: "8px",
                      border: `1px solid ${
                        matchedSection ? "#BFDBFE" : C.border
                      }`,
                      background: "rgba(255,255,255,0.96)",
                      color: matchedSection ? C.primary : C.textSec,
                      fontSize: "13px",
                      fontWeight: 600,
                      lineHeight: 1.55,
                      boxShadow: "0 5px 14px rgba(15,23,42,0.06)",
                    }}
                  >
                    {matchedSection ? (
                      <>
                        已定位到相关章节：
                        <button
                          type="button"
                          onClick={() => scrollToSection(matchedSection)}
                          style={{
                            marginLeft: "6px",
                            padding: 0,
                            border: "none",
                            background: "transparent",
                            color: C.primary,
                            font: "inherit",
                            fontWeight: 700,
                            cursor: "pointer",
                          }}
                        >
                          {matchedSection.title}
                        </button>
                      </>
                    ) : (
                      "未找到明确对应章节，已展示完整教案供人工对照。"
                    )}
                  </div>
                )}

                {documentStructure.preambleMarkdown.trim() && (
                  <article style={lessonSectionStyle(false)}>
                    {renderMarkdown(documentStructure.preambleMarkdown)}
                  </article>
                )}

                {documentStructure.sections.map((section) => {
                  const active = focusedSectionID === section.id;
                  const sectionMarkdown = data.content.slice(
                    section.startOffset,
                    section.endOffset,
                  );

                  return (
                    <article
                      key={section.id}
                      ref={(node) => {
                        if (node) {
                          sectionRefs.current.set(section.id, node);
                        } else {
                          sectionRefs.current.delete(section.id);
                        }
                      }}
                      tabIndex={-1}
                      style={lessonSectionStyle(active)}
                    >
                      {active && (
                        <div
                          style={{
                            marginBottom: "8px",
                            color: C.primary,
                            fontSize: "12px",
                            fontWeight: 700,
                          }}
                        >
                          当前问题相关章节
                        </div>
                      )}

                      <div
                        style={{
                          color: C.text,
                          fontSize: "14px",
                          lineHeight: 1.85,
                          overflowWrap: "anywhere",
                          wordBreak: "break-word",
                        }}
                      >
                        {renderMarkdown(sectionMarkdown)}
                      </div>
                    </article>
                  );
                })}
              </div>
            )}
        </main>

        <footer
          style={{
            flexShrink: 0,
            padding: "10px 18px",
            borderTop: `1px solid ${C.border}`,
            background: "#FAFAFA",
            color: C.textMuted,
            fontSize: "12px",
            lineHeight: 1.5,
          }}
        >
          宽屏可拖动左侧分隔线调整宽度；按 Escape
          关闭。原教案只读，不会修改课件或问题状态。
        </footer>
      </aside>
    </>
  );
}

const headerButtonStyle: CSSProperties = {
  minHeight: "36px",
  padding: "7px 10px",
  border: `1px solid ${C.border}`,
  borderRadius: "8px",
  background: "#FFFFFF",
  color: C.textSec,
  fontSize: "13px",
  fontWeight: 600,
  cursor: "pointer",
  whiteSpace: "nowrap",
};

function lessonSectionStyle(active: boolean): CSSProperties {
  return {
    marginTop: "12px",
    padding: "16px 18px",
    borderRadius: "10px",
    border: active ? "2px solid #93C5FD" : `1px solid ${C.border}`,
    background: active ? "#EFF6FF" : C.card,
    boxShadow: active ? "0 8px 22px rgba(59,130,246,0.10)" : "none",
    scrollMarginTop: "72px",
    outline: "none",
  };
}

function DrawerState({
  children,
}: {
  children: ReactNode;
}) {
  return (
    <div
      style={{
        padding: "52px 20px",
        color: C.textMuted,
        fontSize: "14px",
        lineHeight: 1.7,
        textAlign: "center",
      }}
    >
      {children}
    </div>
  );
}
