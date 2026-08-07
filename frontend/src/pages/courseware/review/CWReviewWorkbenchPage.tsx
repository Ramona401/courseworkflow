/**
 * CWReviewWorkbenchPage.tsx
 *
 * 课件正式审核全屏工作台主编排。
 *
 * 模块边界：
 *
 *   - CWReviewWorkbenchHeader：顶部导航和课件基本信息；
 *   - CWReviewWorkbenchViewer：课件预览、翻页和全屏放映；
 *   - CWReviewWorkbenchSidebar：上轮整改、AI审核、批注和历史；
 *   - CWReviewDecisionPanel：退回清单、审核意见和正式决定。
 *
 * V1.3复审规则：
 *
 *   - 审核员逐条判断上轮问题是否已经解决；
 *   - 通过审核前必须确认本轮全部旧问题已经解决；
 *   - 继续退回时允许只确认部分问题；
 *   - 复审判断随正式审核决定一次提交；
 *   - AI不能自动确认问题解决。
 */

import {
  useCallback,
  useEffect,
  useState,
  type CSSProperties,
} from "react";

import {
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";

import {
  getCWReviewDetail,
  reviewCWL1,
  reviewCWL2,
  type CWReviewDetailResponse,
} from "@/api/coursewares";

import type {
  CWAIReviewContext,
} from "./CWAIReviewPanel";
import CWLessonPlanCompareDrawer from "./CWLessonPlanCompareDrawer";
import CWReviewDecisionPanel from "./CWReviewDecisionPanel";
import CWReviewWorkbenchHeader from "./CWReviewWorkbenchHeader";
import CWReviewWorkbenchSidebar, {
  resolveCWAnnotationPage,
  type CWReviewSideTab,
} from "./CWReviewWorkbenchSidebar";
import CWReviewWorkbenchViewer from "./CWReviewWorkbenchViewer";
import {
  buildCWReviewDecisionRequest,
} from "./cwReviewDecisionPayload";

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

export default function CWReviewWorkbenchPage() {
  const { id } =
    useParams<{
      id: string;
    }>();

  const navigate =
    useNavigate();

  const [searchParams] =
    useSearchParams();

  const level =
    searchParams.get(
      "level",
    ) === "2"
      ? 2
      : 1;

  const [
    detail,
    setDetail,
  ] =
    useState<
      CWReviewDetailResponse |
      null
    >(null);

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    loadError,
    setLoadError,
  ] = useState("");

  const [
    activePage,
    setActivePage,
  ] = useState(1);

  const [
    showLessonPlanCompare,
    setShowLessonPlanCompare,
  ] = useState(false);

  const [
    sideTab,
    setSideTab,
  ] =
    useState<CWReviewSideTab>(
      "ai",
    );

  const [
    decision,
    setDecision,
  ] =
    useState<
      "approved" |
      "revision"
    >("approved");

  const [
    comment,
    setComment,
  ] = useState("");

  const [
    score,
    setScore,
  ] = useState("");

  const [
    resolvedCarryoverItemIds,
    setResolvedCarryoverItemIds,
  ] =
    useState<string[]>([]);

  const [
    aiReviewContext,
    setAIReviewContext,
  ] =
    useState<CWAIReviewContext>({
      sessionId: null,
      selectedItemIds: [],
      selectedItems: [],
    });

  const [
    submitting,
    setSubmitting,
  ] = useState(false);

  const [
    toast,
    setToast,
  ] =
    useState<{
      message: string;
      type:
        | "success"
        | "error";
    } | null>(null);

  const showToast =
    useCallback(
      (
        message: string,
        type:
          | "success"
          | "error" =
          "success",
      ) => {
        setToast({
          message,
          type,
        });

        window.setTimeout(
          () =>
            setToast(null),
          3000,
        );
      },
      [],
    );

  /**
   * 只在会话、选中项内容或移出动作真正变化时更新父级上下文。
   */
  const handleAIReviewContextChange =
    useCallback(
      (
        next:
          CWAIReviewContext,
      ) => {
        setAIReviewContext(
          (previous) => {
            const sameSession =
              previous
                .sessionId ===
              next.sessionId;

            const sameIDs =
              previous
                .selectedItemIds
                .length ===
                next
                  .selectedItemIds
                  .length &&
              previous
                .selectedItemIds
                .every(
                  (
                    itemID,
                    index,
                  ) =>
                    itemID ===
                    next
                      .selectedItemIds[
                      index
                    ],
                );

            const sameItems =
              previous
                .selectedItems
                .length ===
                next
                  .selectedItems
                  .length &&
              previous
                .selectedItems
                .every(
                  (
                    item,
                    index,
                  ) => {
                    const nextItem =
                      next
                        .selectedItems[
                        index
                      ];

                    return (
                      !!nextItem &&
                      item.id ===
                        nextItem.id &&
                      item.status ===
                        nextItem.status &&
                      item
                        .confirmed_instruction ===
                        nextItem
                          .confirmed_instruction &&
                      item
                        .page_number_snapshot ===
                        nextItem
                          .page_number_snapshot
                    );
                  },
                );

            const sameRemoveAction =
              previous
                .removeSelectedItem ===
              next
                .removeSelectedItem;

            if (
              sameSession &&
              sameIDs &&
              sameItems &&
              sameRemoveAction
            ) {
              return previous;
            }

            return {
              sessionId:
                next.sessionId,
              selectedItemIds: [
                ...next
                  .selectedItemIds,
              ],
              selectedItems: [
                ...next
                  .selectedItems,
              ],
              removeSelectedItem:
                next
                  .removeSelectedItem,
            };
          },
        );
      },
      [],
    );

  const handleCarryoverResolvedChange =
    useCallback(
      (
        itemId: string,
        resolved: boolean,
      ) => {
        setResolvedCarryoverItemIds(
          (previous) => {
            if (!resolved) {
              return previous.filter(
                (currentID) =>
                  currentID !==
                  itemId,
              );
            }

            return Array.from(
              new Set([
                ...previous,
                itemId,
              ]),
            );
          },
        );
      },
      [],
    );

  const loadDetail =
    useCallback(
      async () => {
        if (!id) {
          return;
        }

        setLoading(true);
        setLoadError("");

        try {
          const response =
            await getCWReviewDetail(
              id,
            );

          setDetail(
            response,
          );

          setResolvedCarryoverItemIds(
            [],
          );

          setSideTab(
            (
              response
                .carryover_items ||
              []
            ).length > 0
              ? "carryover"
              : "ai",
          );

          const nextPages =
            response
              .courseware
              ?.pages || [];

          if (
            nextPages.length >
            0
          ) {
            setActivePage(
              nextPages[0]
                .page_number,
            );
          }
        } catch (cause) {
          setLoadError(
            cause instanceof
            Error
              ? cause.message
              : "加载课件审核详情失败",
          );
        } finally {
          setLoading(
            false,
          );
        }
      },
      [id],
    );

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  const pages =
    detail?.courseware
      ?.pages || [];

  const annotations =
    detail?.annotations ||
    [];

  const reviews =
    detail?.reviews || [];

  const carryoverItems =
    detail
      ?.carryover_items ||
    [];

  const pendingReviewRound =
    detail
      ?.pending_review_round ||
    0;

  const coursewareTitle =
    detail?.courseware
      ?.title ||
    "课件审核";

  const annotationCountByPageID =
    annotations.reduce<
      Record<
        string,
        number
      >
    >(
      (
        map,
        annotation,
      ) => {
        const targetPage =
          resolveCWAnnotationPage(
            annotation,
            pages,
          );

        if (targetPage) {
          map[targetPage.id] =
            (
              map[
                targetPage.id
              ] || 0
            ) + 1;
        }

        return map;
      },
      {},
    );

  const handleSubmit =
    async () => {
      if (!id) {
        return;
      }

      if (!comment.trim()) {
        showToast(
          "请填写审核意见",
          "error",
        );
        return;
      }

      if (
        decision ===
          "approved" &&
        carryoverItems.length >
          0 &&
        resolvedCarryoverItemIds
          .length !==
          carryoverItems.length
      ) {
        setSideTab(
          "carryover",
        );

        showToast(
          "通过前，请逐条检查并确认上轮整改问题已经解决",
          "error",
        );
        return;
      }

      setSubmitting(true);

      try {
        const request =
          buildCWReviewDecisionRequest({
            decision,
            comment,
            scoreText:
              score,
            aiReviewContext,
            resolvedReviewItemIds:
              resolvedCarryoverItemIds,
          });

        if (level === 1) {
          await reviewCWL1(
            id,
            request,
          );
        } else {
          await reviewCWL2(
            id,
            request,
          );
        }

        showToast(
          decision ===
          "approved"
            ? "✅ 审核通过"
            : "↩️ 已退回修改",
        );

        window.setTimeout(
          () =>
            navigate(
              "/courseware/review",
            ),
          1200,
        );
      } catch (cause) {
        showToast(
          cause instanceof
          Error
            ? cause.message
            : "审核失败",
          "error",
        );
      } finally {
        setSubmitting(
          false,
        );
      }
    };

  const handleAIComment =
    (
      aiComment: string,
    ) => {
      setComment(
        aiComment,
      );

      showToast(
        "AI意见草稿已填入，请人工检查和编辑",
      );
    };

  if (loading) {
    return (
      <div
        style={
          centerPageStyle
        }
      >
        <div
          style={{
            textAlign:
              "center",
            color:
              C.textMuted,
          }}
        >
          <div
            style={{
              width:
                "28px",
              height:
                "28px",
              margin:
                "0 auto 12px",
              border:
                `3px solid ${C.border}`,
              borderTopColor:
                C.primary,
              borderRadius:
                "50%",
              animation:
                "spin 0.8s linear infinite",
            }}
          />

          <div>
            加载课件中...
          </div>

          <style>
            {`@keyframes spin{to{transform:rotate(360deg)}}`}
          </style>
        </div>
      </div>
    );
  }

  if (
    loadError ||
    !detail
  ) {
    return (
      <div
        style={{
          ...centerPageStyle,
          flexDirection:
            "column",
          gap: "16px",
        }}
      >
        <div
          style={{
            fontSize:
              "44px",
          }}
        >
          😵
        </div>

        <div
          style={{
            color: C.text,
            fontSize:
              "15px",
          }}
        >
          {loadError ||
            "课件不存在或无权限审核"}
        </div>

        <div
          style={{
            display:
              "flex",
            gap: "10px",
          }}
        >
          <button
            type="button"
            onClick={() =>
              void loadDetail()
            }
            style={
              secondaryButtonStyle
            }
          >
            重试
          </button>

          <button
            type="button"
            onClick={() =>
              navigate(
                "/courseware/review",
              )
            }
            style={{
              ...secondaryButtonStyle,
              border:
                "none",
              background:
                C.primary,
              color: "#fff",
              fontWeight:
                600,
            }}
          >
            返回审核列表
          </button>
        </div>
      </div>
    );
  }

  return (
    <div
      style={{
        height: "100vh",
        display: "flex",
        flexDirection:
          "column",
        background: C.bg,
      }}
    >
      <CWReviewWorkbenchHeader
        level={level}
        coursewareTitle={
          coursewareTitle
        }
        subject={
          detail
            .courseware
            .subject
        }
        grade={
          detail
            .courseware
            .grade
        }
        pageCount={
          pages.length
        }
        hasLessonPlan={
          !!detail
            .courseware
            .lesson_plan_id
        }
        onBack={() =>
          navigate(
            "/courseware/review",
          )
        }
        onOpenLessonPlan={() =>
          setShowLessonPlanCompare(
            true,
          )
        }
      />

      <div
        style={{
          flex: 1,
          display: "flex",
          minHeight: 0,
        }}
      >
        <CWReviewWorkbenchViewer
          pages={pages}
          activePage={
            activePage
          }
          annotationCountByPageID={
            annotationCountByPageID
          }
          onActivePageChange={
            setActivePage
          }
        />

        <div
          style={{
            width:
              "420px",
            minHeight: 0,
            display:
              "flex",
            flexDirection:
              "column",
            flexShrink: 0,
            background:
              C.card,
          }}
        >
          <CWReviewWorkbenchSidebar
            coursewareId={
              id || ""
            }
            coursewareTitle={
              coursewareTitle
            }
            subject={
              detail
                .courseware
                .subject
            }
            grade={
              detail
                .courseware
                .grade
            }
            lessonPlanId={
              detail
                .courseware
                .lesson_plan_id
            }
            reviewLevel={
              level
            }
            pages={pages}
            annotations={
              annotations
            }
            reviews={reviews}
            carryoverItems={
              carryoverItems
            }
            pendingReviewRound={
              pendingReviewRound
            }
            resolvedCarryoverItemIds={
              resolvedCarryoverItemIds
            }
            activePage={
              activePage
            }
            sideTab={
              sideTab
            }
            onSideTabChange={
              setSideTab
            }
            onSelectPage={
              setActivePage
            }
            onUseReviewComment={
              handleAIComment
            }
            onReviewContextChange={
              handleAIReviewContextChange
            }
            onCarryoverResolvedChange={
              handleCarryoverResolvedChange
            }
          />

          <CWReviewDecisionPanel
            level={level}
            decision={
              decision
            }
            score={score}
            comment={
              comment
            }
            submitting={
              submitting
            }
            aiReviewContext={
              aiReviewContext
            }
            onDecisionChange={
              setDecision
            }
            onScoreChange={
              setScore
            }
            onCommentChange={
              setComment
            }
            onSelectPage={
              setActivePage
            }
            onOpenAI={() =>
              setSideTab(
                "ai",
              )
            }
            onCancel={() =>
              navigate(
                "/courseware/review",
              )
            }
            onSubmit={() =>
              void handleSubmit()
            }
          />
        </div>
      </div>

      {id && (
        <CWLessonPlanCompareDrawer
          open={
            showLessonPlanCompare
          }
          coursewareId={
            id
          }
          lessonPlanId={
            detail
              .courseware
              .lesson_plan_id
          }
          onClose={() =>
            setShowLessonPlanCompare(
              false,
            )
          }
        />
      )}

      {toast && (
        <div
          style={{
            position:
              "fixed",
            bottom:
              "32px",
            left: "50%",
            zIndex: 99999,
            padding:
              "12px 24px",
            transform:
              "translateX(-50%)",
            border:
              toast.type ===
              "error"
                ? "1px solid #FECACA"
                : "none",
            borderRadius:
              "10px",
            background:
              toast.type ===
              "error"
                ? "#FEF2F2"
                : "#1F2937",
            color:
              toast.type ===
              "error"
                ? C.danger
                : "#fff",
            fontSize:
              "14px",
            fontWeight:
              500,
            whiteSpace:
              "nowrap",
            boxShadow:
              "0 8px 24px rgba(0,0,0,0.15)",
          }}
        >
          {toast.type ===
          "success"
            ? "✓ "
            : "⚠️ "}

          {toast.message}
        </div>
      )}
    </div>
  );
}

const centerPageStyle:
  CSSProperties = {
  height: "100vh",
  display: "flex",
  alignItems:
    "center",
  justifyContent:
    "center",
  background: C.bg,
};

const secondaryButtonStyle:
  CSSProperties = {
  padding: "9px 20px",
  borderRadius: "8px",
  border:
    `1px solid ${C.borderMid}`,
  background: "#fff",
  color: C.textSec,
  fontSize: "14px",
  cursor: "pointer",
};
