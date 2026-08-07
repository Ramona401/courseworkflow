/**
 * CWSelfReviewPanel.tsx
 *
 * 课件工坊最终确认页中的作者AI自审面板。
 *
 * 底层复用正式课件审核能力：
 *   - 全课件结构索引；
 *   - 来源教案和明确关联的课程大纲；
 *   - 具体教育域、学科和年级；
 *   - HTML、CSS、JavaScript代码证据；
 *   - 事件入口、可达函数、DOM目标和状态变量；
 *   - 答案显隐、反馈、重试和外部运行时依赖；
 *   - 顺序分批与跨页连续性账本；
 *   - 最终风险回看和优先修改清单。
 *
 * 安全边界：
 *   - review_level固定为0；
 *   - 仅课件作者本人可运行；
 *   - 使用courseware_self_review专用助手场景；
 *   - 不写正式课件审核记录；
 *   - 不改变publish_state或review_level；
 *   - 不自动修改或提交课件。
 *
 * 状态体验改进：
 *   1. 打开面板时重新读取课件最新状态；
 *   2. 不可自审时不再发送必然失败的准备请求；
 *   3. 按“已提交审核、已共享、已通过审核、尚未生成”等状态，
 *      展示普通老师能够理解的具体原因；
 *   4. 共享发布状态提供“撤回共享并继续修改”操作；
 *   5. 后端原有权限和状态校验保持不变，前端只做提前解释和引导。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  getCourseware,
  publishCourseware,
  type CWAIReviewItem,
} from "@/api/coursewares";

import CWAIReviewPanel from "@/pages/courseware/review/CWAIReviewPanel";
import CWAIReviewPanelBoundary from "@/pages/courseware/review/CWAIReviewPanelBoundary";

const C = {
  primary: "#4F7BE8",
  success: "#059669",
  warning: "#D97706",
  danger: "#DC2626",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#E2E8F0",
  bg: "#F8FAFC",
};

export interface CWSelfReviewPanelProps {
  coursewareId: string;
  coursewareTitle: string;
  subject: string;
  grade: string;
  lessonPlanId?: string | null;

  onSelectPage: (
    pageNumber: number,
  ) => void;

  onInjectToRefine: (
    item: CWAIReviewItem,
  ) => void;
}

/**
 * AI自审入口读取到的最小课件状态。
 *
 * 这里只保留决定自审可用性的字段，
 * 不在本组件复制完整课件详情。
 */
interface SelfReviewCoursewareState {
  loading: boolean;
  status: string;
  publishState: string;
  error: string;
}

/**
 * 不可自审时展示给老师的解释。
 */
interface SelfReviewBlockReason {
  title: string;
  description: string;

  /**
   * 只有共享发布状态允许在当前面板直接撤回。
   *
   * 正式审核和已通过审核仍必须遵循原审核流程，
   * 不能由本组件擅自改变状态。
   */
  canWithdrawShared: boolean;
}

/**
 * 将后端状态值标准化，避免空格和大小写导致判断偏差。
 */
function normalizeState(
  value: string | null | undefined,
): string {
  return (
    value || ""
  )
    .trim()
    .toLowerCase();
}

/**
 * 根据课件生产状态和发布状态，生成老师可理解的说明。
 *
 * 后端允许作者自审的条件保持原样：
 *   - status只能是preview或confirmed；
 *   - publish_state只能是空、private、published_personal或revision。
 */
function buildSelfReviewBlockReason(
  rawStatus: string,
  rawPublishState: string,
): SelfReviewBlockReason | null {
  const status =
    normalizeState(
      rawStatus,
    );

  const publishState =
    normalizeState(
      rawPublishState,
    );

  /**
   * 发布与审核状态优先解释。
   *
   * 同一个课件可能同时有生产状态和发布状态，
   * 用户更需要先知道“为什么现在不能操作”以及下一步怎么做。
   */
  if (
    publishState ===
    "submitted"
  ) {
    return {
      title:
        "该课件已提交正式审核",
      description:
        "审核期间暂不能发起AI自审，以免正在审核的版本发生变化。待审核退回修改后，可再次进行自审。",
      canWithdrawShared:
        false,
    };
  }

  if (
    publishState ===
    "approved"
  ) {
    return {
      title:
        "该课件已通过正式审核",
      description:
        "当前审核版本暂不能重新发起AI自审。如需继续修改，请先进入修订流程，再对修订版本进行自审。",
      canWithdrawShared:
        false,
    };
  }

  if (
    publishState ===
    "published_shared"
  ) {
    return {
      title:
        "该课件正在共享发布",
      description:
        "为避免共享中的课件版本在自审过程中发生变化，请先撤回共享。撤回后即可继续修改并发起AI自审。",
      canWithdrawShared:
        true,
    };
  }

  /**
   * 后端允许个人发布状态继续自审，
   * 因此published_personal不在这里拦截。
   */
  const allowedPublishStates =
    new Set([
      "",
      "private",
      "published_personal",
      "revision",
    ]);

  if (
    !allowedPublishStates.has(
      publishState,
    )
  ) {
    return {
      title:
        "该课件当前发布状态暂不支持AI自审",
      description:
        "请先将课件恢复到私有或修订状态，再重新发起AI自审。",
      canWithdrawShared:
        false,
    };
  }

  if (
    status === "preview" ||
    status === "confirmed"
  ) {
    return null;
  }

  if (
    status === "generating" ||
    status === "indexing" ||
    status === "draft" ||
    status === "created"
  ) {
    return {
      title:
        "该课件尚未完成生成",
      description:
        "AI自审需要读取完整页面和互动代码。请先完成课件页面生成，再返回这里开始自审。",
      canWithdrawShared:
        false,
    };
  }

  if (
    status === "in_pipeline" ||
    status === "submitted"
  ) {
    return {
      title:
        "该课件正在审核流程中",
      description:
        "审核期间暂不能发起AI自审。待审核退回修改后，可再次进行自审。",
      canWithdrawShared:
        false,
    };
  }

  return {
    title:
      "该课件当前处于不可修改状态",
    description:
      "暂不能发起AI自审。请先将课件恢复到可修改状态，或完成当前生成、审核和发布操作。",
    canWithdrawShared:
      false,
  };
}

/**
 * 复制自审修改摘要。
 *
 * 优先使用安全上下文Clipboard API；
 * 不支持时退回隐藏textarea和execCommand。
 */
async function copyText(
  content: string,
): Promise<void> {
  if (
    navigator.clipboard &&
    window.isSecureContext
  ) {
    await navigator.clipboard.writeText(
      content,
    );
    return;
  }

  const textarea =
    document.createElement(
      "textarea",
    );

  textarea.value =
    content;

  textarea.style.position =
    "fixed";

  textarea.style.left =
    "-9999px";

  textarea.style.opacity =
    "0";

  document.body.appendChild(
    textarea,
  );

  textarea.focus();
  textarea.select();

  const copied =
    document.execCommand(
      "copy",
    );

  document.body.removeChild(
    textarea,
  );

  if (!copied) {
    throw new Error(
      "浏览器拒绝复制",
    );
  }
}

export default function CWSelfReviewPanel({
  coursewareId,
  coursewareTitle,
  subject,
  grade,
  lessonPlanId,
  onSelectPage,
  onInjectToRefine,
}: CWSelfReviewPanelProps) {
  const [
    message,
    setMessage,
  ] = useState("");

  const [
    withdrawingShared,
    setWithdrawingShared,
  ] = useState(false);

  const [
    coursewareState,
    setCoursewareState,
  ] = useState<SelfReviewCoursewareState>({
    loading: true,
    status: "",
    publishState: "",
    error: "",
  });

  /**
   * 每次状态请求使用递增编号。
   *
   * 用户快速切换课件或组件卸载时，
   * 旧请求结果不会覆盖新课件状态。
   */
  const stateRequestIDRef =
    useRef(0);

  /**
   * 重新读取服务器上的课件最新状态。
   *
   * 不依赖父组件缓存，避免课件刚提交审核或刚撤回共享后，
   * 自审面板仍使用旧状态。
   */
  const reloadCoursewareState =
    useCallback(async () => {
      const requestID =
        stateRequestIDRef.current +
        1;

      stateRequestIDRef.current =
        requestID;

      setCoursewareState(
        (previous) => ({
          ...previous,
          loading: true,
          error: "",
        }),
      );

      try {
        const courseware =
          await getCourseware(
            coursewareId,
          );

        if (
          stateRequestIDRef.current !==
          requestID
        ) {
          return;
        }

        setCoursewareState({
          loading: false,
          status:
            courseware.status || "",
          publishState:
            courseware.publish_state ||
            "",
          error: "",
        });
      } catch (cause) {
        if (
          stateRequestIDRef.current !==
          requestID
        ) {
          return;
        }

        setCoursewareState(
          (previous) => ({
            ...previous,
            loading: false,
            error:
              cause instanceof Error
                ? cause.message
                : "读取课件状态失败",
          }),
        );
      }
    }, [
      coursewareId,
    ]);

  useEffect(() => {
    void reloadCoursewareState();

    return () => {
      stateRequestIDRef.current +=
        1;
    };
  }, [
    reloadCoursewareState,
  ]);

  const blockReason =
    useMemo(
      () =>
        buildSelfReviewBlockReason(
          coursewareState.status,
          coursewareState.publishState,
        ),
      [
        coursewareState.status,
        coursewareState.publishState,
      ],
    );

  const handleUseSummary =
    useCallback(
      (
        summary: string,
      ) => {
        void copyText(
          summary,
        )
          .then(() => {
            setMessage(
              "✅ 自审修改摘要已复制，可粘贴到备课记录或逐项修改。",
            );
          })
          .catch(() => {
            setMessage(
              "⚠️ 浏览器未允许自动复制，请手动选择摘要文本复制。",
            );
          });
      },
      [],
    );

  /**
   * 撤回共享发布。
   *
   * 只调用已有的发布状态接口把课件恢复为private；
   * 不改变课件生产状态、审核级别或页面内容。
   */
  const handleWithdrawShared =
    async () => {
      if (
        withdrawingShared
      ) {
        return;
      }

      const confirmed =
        window.confirm(
          "撤回共享后，其他用户将暂时无法从共享课件库访问该课件。撤回后可继续修改和进行AI自审。确定撤回共享吗？",
        );

      if (!confirmed) {
        return;
      }

      setWithdrawingShared(
        true,
      );

      setMessage("");

      try {
        await publishCourseware(
          coursewareId,
          "private",
        );

        setMessage(
          "✅ 已撤回共享，课件已恢复为私有状态，可以继续修改和进行AI自审。",
        );

        await reloadCoursewareState();
      } catch (cause) {
        setMessage(
          "❌ 撤回共享失败：" +
            (
              cause instanceof Error
                ? cause.message
                : "未知错误"
            ),
        );
      } finally {
        setWithdrawingShared(
          false,
        );
      }
    };

  const messageAppearance =
    message.startsWith("✅")
      ? {
          border:
            "1px solid #A7F3D0",
          background:
            "#ECFDF5",
          color:
            C.success,
        }
      : message.startsWith("⚠️")
        ? {
            border:
              "1px solid #FDE68A",
            background:
              "#FFFBEB",
            color:
              C.warning,
          }
        : {
            border:
              "1px solid #FECACA",
            background:
              "#FEF2F2",
            color:
              C.danger,
          };

  return (
    <section
      style={{
        marginTop: "16px",
        padding: "16px",
        borderRadius: "12px",
        border: `1px solid ${C.border}`,
        background: C.bg,
      }}
    >
      <div
        style={{
          marginBottom: "12px",
          padding: "10px 12px",
          borderRadius: "9px",
          border:
            "1px solid rgba(79,123,232,0.24)",
          background:
            "rgba(79,123,232,0.07)",
        }}
      >
        <div
          style={{
            color: C.text,
            fontSize: "13px",
            fontWeight: 700,
          }}
        >
          🛡️ 提交前AI课件自审
        </div>

        <div
          style={{
            marginTop: "4px",
            color: C.textSec,
            fontSize: "11px",
            lineHeight: 1.65,
          }}
        >
          使用与正式审核相同的内容和代码扫描能力，提前定位页面、互动脚本、答案显隐、反馈重试和跨页连续性问题。自审不会自动修改或提交课件。
        </div>
      </div>

      {message && (
        <div
          style={{
            marginBottom: "10px",
            padding: "8px 10px",
            borderRadius: "8px",
            border:
              messageAppearance.border,
            background:
              messageAppearance.background,
            color:
              messageAppearance.color,
            fontSize: "11px",
            lineHeight: 1.6,
          }}
        >
          {message}
        </div>
      )}

      {coursewareState.loading ? (
        <div
          style={{
            padding: "14px",
            borderRadius: "9px",
            border:
              `1px solid ${C.border}`,
            background: "#FFFFFF",
            color: C.textSec,
            fontSize: "12px",
            lineHeight: 1.7,
          }}
        >
          正在确认课件当前状态…
        </div>
      ) : coursewareState.error ? (
        <div
          style={{
            padding: "14px",
            borderRadius: "9px",
            border:
              "1px solid #FECACA",
            background:
              "#FEF2F2",
          }}
        >
          <div
            style={{
              color: C.danger,
              fontSize: "13px",
              fontWeight: 700,
            }}
          >
            暂时无法确认课件状态
          </div>

          <div
            style={{
              marginTop: "5px",
              color: C.textSec,
              fontSize: "11px",
              lineHeight: 1.65,
            }}
          >
            {coursewareState.error}
          </div>

          <button
            type="button"
            onClick={() => {
              void reloadCoursewareState();
            }}
            style={{
              marginTop: "10px",
              padding: "7px 14px",
              borderRadius: "7px",
              border:
                `1px solid ${C.primary}`,
              background: "#FFFFFF",
              color: C.primary,
              fontSize: "11px",
              fontWeight: 700,
              cursor: "pointer",
            }}
          >
            重新读取状态
          </button>
        </div>
      ) : blockReason ? (
        <div
          style={{
            padding: "14px",
            borderRadius: "9px",
            border:
              "1px solid #FDE68A",
            background:
              "#FFFBEB",
          }}
        >
          <div
            style={{
              display: "flex",
              alignItems: "flex-start",
              gap: "8px",
            }}
          >
            <span
              aria-hidden="true"
              style={{
                fontSize: "18px",
                lineHeight: 1.2,
              }}
            >
              ⚠️
            </span>

            <div
              style={{
                minWidth: 0,
                flex: 1,
              }}
            >
              <div
                style={{
                  color:
                    "#92400E",
                  fontSize:
                    "13px",
                  fontWeight:
                    700,
                }}
              >
                {blockReason.title}
              </div>

              <div
                style={{
                  marginTop:
                    "5px",
                  color:
                    "#78350F",
                  fontSize:
                    "11px",
                  lineHeight:
                    1.7,
                }}
              >
                {
                  blockReason.description
                }
              </div>
            </div>
          </div>

          {blockReason.canWithdrawShared && (
            <button
              type="button"
              onClick={() => {
                void handleWithdrawShared();
              }}
              disabled={
                withdrawingShared
              }
              style={{
                width: "100%",
                marginTop: "12px",
                padding: "9px 14px",
                borderRadius: "8px",
                border: "none",
                background:
                  withdrawingShared
                    ? "#CBD5E1"
                    : C.primary,
                color: "#FFFFFF",
                fontSize: "12px",
                fontWeight: 700,
                cursor:
                  withdrawingShared
                    ? "not-allowed"
                    : "pointer",
              }}
            >
              {withdrawingShared
                ? "正在撤回共享…"
                : "撤回共享并继续修改"}
            </button>
          )}
        </div>
      ) : (
        <CWAIReviewPanelBoundary
          resetKey={
            `${coursewareId}:self`
          }
        >
          <CWAIReviewPanel
            mode="self"
            coursewareId={
              coursewareId
            }
            coursewareTitle={
              coursewareTitle
            }
            subject={
              subject
            }
            grade={
              grade
            }
            lessonPlanId={
              lessonPlanId
            }
            reviewLevel={0}
            onSelectPage={
              onSelectPage
            }
            onUseReviewComment={
              handleUseSummary
            }
            onInjectToRefine={
              onInjectToRefine
            }
          />
        </CWAIReviewPanelBoundary>
      )}
    </section>
  );
}
