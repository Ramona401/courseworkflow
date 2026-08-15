/**
 * CWReviewHistoryPage.tsx
 *
 * R-03“已审核记录只读详情”专用页面。
 *
 * 页面只调用GET history detail：
 *   - 不挂载审核提交；
 *   - 不挂载问题编辑/确认/恢复/AI讨论；
 *   - 不挂载作者课件修改入口；
 *   - 默认展示“审核时页面”；
 *   - “当前页面”是显式独立Tab，不得冒充历史页面。
 */

import {
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import {
  getCWReviewHistoryDetail,
  type CWReviewHistoryDetail,
} from "@/api/coursewares";

import CWReviewWorkbenchViewer from "./CWReviewWorkbenchViewer";
import CWReviewHistoryIssueList from "./CWReviewHistoryIssueList";
import CWReviewHistorySummary from "./CWReviewHistorySummary";

type PageTab =
  | "historical"
  | "current";

const C = {
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  card: "#FFFFFF",
  background: "#F8FAFC",
  primary: "#2563EB",
  warning: "#D97706",
  danger: "#DC2626",
};

const HISTORICAL_UNAVAILABLE_LABELS:
  Record<string, string> = {
    legacy_review_without_page_snapshot:
      "该审核发生在完整页面快照上线前，无法可靠还原审核时页面。当前页面仍可单独查看，但不会被当作历史页面。",
  };

export default function CWReviewHistoryPage() {
  const navigate = useNavigate();

  const { reviewId } =
    useParams<{
      reviewId: string;
    }>();

  const [detail, setDetail] =
    useState<CWReviewHistoryDetail | null>(
      null,
    );

  const [loading, setLoading] =
    useState(true);

  const [error, setError] =
    useState("");

  const [pageTab, setPageTab] =
    useState<PageTab>("historical");

  const [activePage, setActivePage] =
    useState(0);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      if (!reviewId) {
        setError("审核记录地址无效。");
        setLoading(false);
        return;
      }

      setLoading(true);
      setError("");

      try {
        const result =
          await getCWReviewHistoryDetail(
            reviewId,
          );

        if (cancelled) {
          return;
        }

        setDetail(result);
        setPageTab("historical");

        setActivePage(
          result.historical_pages[0]
            ?.page_number || 0,
        );
      } catch (loadError) {
        if (cancelled) {
          return;
        }

        console.error(
          "读取课件已审核历史详情失败:",
          loadError,
        );

        setError(
          "审核记录不存在、无权查看，或暂时无法读取。",
        );
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, [reviewId]);

  const viewerPages =
    useMemo(() => {
      if (!detail) {
        return [];
      }

      if (pageTab === "historical") {
        return detail.historical_pages.map(
          (page) => ({
            id: page.page_id,
            page_number: page.page_number,
            title: page.page_title,
            html_content: page.html_content,
          }),
        );
      }

      return detail.current_pages.map(
        (page) => ({
          id: page.page_id,
          page_number: page.page_number,
          title: page.page_title,
          html_content: page.html_content,
        }),
      );
    }, [
      detail,
      pageTab,
    ]);

  useEffect(() => {
    if (viewerPages.length === 0) {
      setActivePage(0);
      return;
    }

    if (
      !viewerPages.some(
        (page) =>
          page.page_number === activePage,
      )
    ) {
      setActivePage(
        viewerPages[0].page_number,
      );
    }
  }, [
    activePage,
    viewerPages,
  ]);

  const activeHistoricalPage =
    pageTab === "historical" &&
    detail
      ? detail.historical_pages.find(
          (page) =>
            page.page_number ===
            activePage,
        )
      : undefined;

  if (loading) {
    return (
      <div
        style={{
          padding: "60px 24px",
          textAlign: "center",
          color: C.textMuted,
        }}
      >
        正在读取审核记录…
      </div>
    );
  }

  if (!detail || error) {
    return (
      <div
        style={{
          maxWidth: "760px",
          margin: "48px auto",
          padding: "24px",
          borderRadius: "12px",
          border: `1px solid ${C.border}`,
          background: C.card,
          textAlign: "center",
        }}
      >
        <div
          style={{
            color: C.danger,
            fontSize: "15px",
            fontWeight: 700,
          }}
        >
          无法打开审核记录
        </div>

        <div
          style={{
            marginTop: "8px",
            color: C.textSec,
            fontSize: "13px",
            lineHeight: 1.65,
          }}
        >
          {error ||
            "审核记录不存在或当前账号无权查看。"}
        </div>

        <button
          type="button"
          onClick={() =>
            navigate("/courseware/review")
          }
          style={{
            marginTop: "16px",
            padding: "8px 16px",
            borderRadius: "8px",
            border: `1px solid ${C.border}`,
            background: C.card,
            color: C.textSec,
            cursor: "pointer",
            fontWeight: 600,
          }}
        >
          返回已审核列表
        </button>
      </div>
    );
  }

  const historicalUnavailable =
    !detail.historical_pages_available;

  return (
    <div
      style={{
        maxWidth: "1380px",
        margin: "0 auto",
        padding: "20px 24px 32px",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: "12px",
          marginBottom: "16px",
        }}
      >
        <button
          type="button"
          onClick={() =>
            navigate("/courseware/review")
          }
          style={{
            padding: "8px 14px",
            borderRadius: "8px",
            border: `1px solid ${C.border}`,
            background: C.card,
            color: C.textSec,
            cursor: "pointer",
            fontWeight: 600,
            flexShrink: 0,
          }}
        >
          ← 返回已审核列表
        </button>

        <div
          style={{
            minWidth: 0,
          }}
        >
          <h1
            style={{
              margin: 0,
              color: C.text,
              fontSize: "22px",
              lineHeight: 1.35,
            }}
          >
            {detail.record_title}
          </h1>

          <div
            style={{
              marginTop: "5px",
              color: C.textSec,
              fontSize: "13px",
            }}
          >
            这是正式审核历史证据，只读展示；
            后续课件修改不会覆盖本记录。
          </div>
        </div>
      </div>

      <CWReviewHistorySummary
        detail={detail}
      />

      <section
        style={{
          marginTop: "18px",
          padding: "16px",
          borderRadius: "12px",
          border: `1px solid ${C.border}`,
          background: C.background,
        }}
      >
        <div
          style={{
            marginBottom: "10px",
            color: C.text,
            fontSize: "15px",
            fontWeight: 700,
          }}
        >
          本次正式交付的问题
        </div>

        {!detail.issues_available ? (
          <div
            style={{
              padding: "11px 12px",
              borderRadius: "9px",
              border: "1px solid #FED7AA",
              background: "#FFF7ED",
              color: C.warning,
              fontSize: "12px",
              lineHeight: 1.65,
            }}
          >
            该旧审核没有可证明的正式反馈快照，
            因此不会用当前问题列表冒充历史交付内容。
          </div>
        ) : (
          <CWReviewHistoryIssueList
            issues={detail.issues}
            historicalPages={
              detail.historical_pages
            }
          />
        )}
      </section>

      <section
        style={{
          marginTop: "18px",
          borderRadius: "12px",
          border: `1px solid ${C.border}`,
          background: C.card,
          overflow: "hidden",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "6px",
            padding: "10px 12px",
            borderBottom:
              `1px solid ${C.border}`,
          }}
        >
          <button
            type="button"
            onClick={() =>
              setPageTab("historical")
            }
            style={tabButtonStyle(
              pageTab === "historical",
            )}
          >
            审核时页面
          </button>

          <button
            type="button"
            onClick={() =>
              setPageTab("current")
            }
            style={tabButtonStyle(
              pageTab === "current",
            )}
          >
            当前页面
          </button>

          <div
            style={{
              flex: 1,
            }}
          />

          <span
            style={{
              color: C.textMuted,
              fontSize: "11px",
            }}
          >
            两个时间点相互独立
          </span>
        </div>

        {pageTab === "historical" &&
        historicalUnavailable ? (
          <div
            style={{
              padding: "28px",
              color: C.textSec,
              fontSize: "13px",
              lineHeight: 1.75,
              background: C.background,
            }}
          >
            {HISTORICAL_UNAVAILABLE_LABELS[
              detail
                .historical_pages_unavailable_reason
            ] ||
              "该审核没有可证明的完整历史页面快照。"}
          </div>
        ) : (
          <>
            {activeHistoricalPage &&
              !activeHistoricalPage
                .current_exists && (
                <div
                  style={{
                    padding: "9px 14px",
                    borderBottom:
                      "1px solid #FED7AA",
                    background: "#FFF7ED",
                    color: C.warning,
                    fontSize: "12px",
                    fontWeight: 700,
                  }}
                >
                  原页面已删除。下方仍展示本次审核时冻结的历史页面。
                </div>
              )}

            <div
              style={{
                height: "620px",
                minHeight: "480px",
                display: "flex",
                minWidth: 0,
                overflow: "hidden",
              }}
            >
              <CWReviewWorkbenchViewer
                pages={viewerPages}
                activePage={activePage}
                annotationCountByPageID={{}}
                onActivePageChange={
                  setActivePage
                }
              />
            </div>
          </>
        )}
      </section>
    </div>
  );
}

function tabButtonStyle(
  active: boolean,
): React.CSSProperties {
  return {
    padding: "7px 13px",
    borderRadius: "8px",
    border: active
      ? `1px solid ${C.primary}`
      : `1px solid ${C.border}`,
    background: active
      ? "#EFF6FF"
      : C.card,
    color: active
      ? C.primary
      : C.textSec,
    fontSize: "13px",
    fontWeight: active
      ? 700
      : 600,
    cursor: "pointer",
  };
}
