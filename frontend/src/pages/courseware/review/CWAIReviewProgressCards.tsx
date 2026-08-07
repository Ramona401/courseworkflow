/**
 * CWAIReviewProgressCards.tsx
 *
 * AI审核面板的配置、助手选择、启动按钮、错误、会话进度和批次状态展示。
 *
 * 只渲染控制器状态，不管理API调用。
 *
 * R-02配置：
 *   - 启动前选择至少一个审核维度；
 *   - 选择自定义维度时填写具体要求；
 *   - 选择教案参考模式；
 *   - 活动会话显示后端不可变配置快照；
 *   - 重新开始会创建新会话，不修改旧快照。
 *
 * 未选择专属助手时：
 *   - 平台默认系统审核提示词仍然生效；
 *   - 用户可以直接开始审核或自审；
 *   - 专属助手属于可选增强。
 */

import type {
  CSSProperties,
} from "react";

import AssistantSelector from "@/components/ai-assistants/AssistantSelector";

import CWAIReviewConfigCard from "./CWAIReviewConfigCard";

import {
  resolveCWAIReviewAssistantScene,
} from "./cwAIReviewAssistantScene";

import type {
  CWAIReviewController,
} from "./useCWAIReviewController";

const C = {
  primary: "#4F7BE8",
  success: "#10B981",
  danger: "#EF4444",
  text: "#1F2937",
  textSec: "#6B7280",
  textMuted: "#9CA3AF",
  border: "#E5E7EB",
  card: "#FFFFFF",
};

const STATUS_LABELS:
  Record<string, string> = {
  pending: "等待准备",
  preparing: "正在准备上下文",
  reviewing: "正在分批审核",
  aggregating:
    "等待生成综合报告",
  done: "审核分析完成",
  failed: "执行失败",
  cancelled: "已被新会话替代",
};

const BATCH_STATUS_LABELS:
  Record<string, string> = {
  pending: "待执行",
  running: "执行中",
  done: "已完成",
  failed: "失败",
};

export interface CWAIReviewProgressCardsProps {
  controller:
    CWAIReviewController;

  coursewareTitle: string;
  subject: string;
  grade: string;
  lessonPlanId?: string | null;
}

function formatTime(
  raw: string | null,
): string {
  if (!raw) {
    return "";
  }

  try {
    return new Date(
      raw,
    ).toLocaleString("zh-CN");
  } catch {
    return raw;
  }
}

/**
 * 未选择专属助手时的默认规则说明。
 *
 * 这是正常可用状态，不使用警告色：
 * 平台系统提示词仍是本次审核的基础规则，
 * 专属助手只用于叠加学科和个人要求。
 */
function DefaultAssistantNotice({
  isSelfReview,
}: {
  isSelfReview: boolean;
}) {
  const actionName =
    isSelfReview
      ? "自审"
      : "审核";

  return (
    <div
      style={{
        marginBottom: "10px",
        padding: "9px 11px",
        borderRadius: "8px",
        border:
          "1px solid #BFDBFE",
        background:
          "#EFF6FF",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "6px",
          color: "#1D4ED8",
          fontSize: "11px",
          fontWeight: 700,
        }}
      >
        <span
          aria-hidden="true"
        >
          ℹ️
        </span>

        默认系统{actionName}规则已启用
      </div>

      <div
        style={{
          marginTop: "4px",
          color: "#475569",
          fontSize: "10px",
          lineHeight: 1.65,
        }}
      >
        当前未选择专属学科
        {actionName}
        助手，系统仍会使用平台默认的AI
        {actionName}
        提示词，您可以直接开始本次
        {actionName}
        。
      </div>

      <div
        style={{
          marginTop: "3px",
          color: "#64748B",
          fontSize: "10px",
          lineHeight: 1.65,
        }}
      >
        如需更贴合本学科、年级或个人教学要求，可点击上方“管理/创建助手”进行配置。
      </div>
    </div>
  );
}

export default function CWAIReviewProgressCards({
  controller,
  coursewareTitle,
  subject,
  grade,
  lessonPlanId,
}: CWAIReviewProgressCardsProps) {
  if (controller.loading) {
    return (
      <div
        style={{
          padding: "36px 16px",
          textAlign: "center",
          color: C.textMuted,
          fontSize: "13px",
        }}
      >
        正在读取AI分析记录…
      </div>
    );
  }

  const session =
    controller.session;

  /**
   * 不直接消费控制器中被ReturnType扩大后的字符串。
   *
   * 使用严格返回AssistantScene的单一解析函数，
   * 确保AssistantSelector只收到合法课件审核场景。
   */
  const assistantScene =
    resolveCWAIReviewAssistantScene(
      controller.isSelfReview,
    );

  return (
    <>
      <div
        style={{
          padding: "12px",
          borderRadius: "10px",
          border: `1px solid ${C.border}`,
          background: C.card,
        }}
      >
        <div
          style={{
            fontSize: "13px",
            fontWeight: 700,
            color: C.text,
            marginBottom: "4px",
          }}
        >
          🤖 {controller.panelTitle}
        </div>

        <div
          style={{
            fontSize: "11px",
            color: C.textSec,
            lineHeight: 1.6,
            marginBottom: "5px",
          }}
        >
          {coursewareTitle} ·{" "}
          {subject} · {grade}
        </div>

        <div
          style={{
            fontSize: "11px",
            color: C.textSec,
            lineHeight: 1.6,
            marginBottom: "10px",
          }}
        >
          {controller.isSelfReview
            ? "提交前按所选维度检查完整课件和真实互动代码；发现可采纳为私有整改项。"
            : "按本次所选维度审核完整课件；确认后的页级修改指令才会进入退回清单。"}
        </div>

        <CWAIReviewConfigCard
          controller={controller}
          hasLessonPlan={
            !!lessonPlanId
          }
        />

        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "7px",
            marginBottom: "10px",
          }}
        >
          <AssistantSelector
            scene={assistantScene}
            value={
              controller.assistantId
            }
            onChange={
              controller.setAssistantId
            }
            subject={subject}
            lessonPlanId={
              lessonPlanId ||
              undefined
            }
            disabled={
              controller.running ||
              !controller
                .reviewConfigEditable
            }
            compact
          />

          <button
            type="button"
            onClick={
              controller.handleManageAssistants
            }
            disabled={
              controller.running
            }
            style={{
              flexShrink: 0,
              padding: "7px 9px",
              borderRadius: "7px",
              border: `1px solid ${C.border}`,
              background: "#fff",
              color:
                controller.running
                  ? C.textMuted
                  : C.primary,
              fontSize: "11px",
              fontWeight: 600,
              cursor:
                controller.running
                  ? "not-allowed"
                  : "pointer",
              whiteSpace: "nowrap",
            }}
          >
            ⚙ 管理/创建助手
          </button>
        </div>

        {!controller.assistantId && (
          <DefaultAssistantNotice
            isSelfReview={
              controller.isSelfReview
            }
          />
        )}

        {controller.canRestart && (
          <button
            type="button"
            onClick={
              controller.handleStart
            }
            disabled={
              controller.running ||
              !!controller
                .reviewConfigError
            }
            style={actionButtonStyle(
              controller.running ||
              !!controller
                .reviewConfigError,
            )}
          >
            {controller.running
              ? `正在顺序${controller.actionName}…`
              : session
                ? `🔄 以当前配置重新开始AI${controller.actionName}`
                : `▶ 开始完整AI${controller.actionName}`}
          </button>
        )}

        {controller.canContinue && (
          <button
            type="button"
            onClick={
              controller.handleContinue
            }
            disabled={
              controller.running
            }
            style={actionButtonStyle(
              controller.running,
            )}
          >
            {controller.running
              ? `正在顺序${controller.actionName}…`
              : session?.status ===
                  "aggregating"
                ? controller.isSelfReview
                  ? "生成最终自审报告"
                  : "生成最终综合报告"
                : `继续未完成的AI${controller.actionName}`}
          </button>
        )}

        <div
          style={{
            marginTop: "8px",
            fontSize: "10px",
            color: C.textMuted,
            lineHeight: 1.5,
          }}
        >
          每次只执行一个页面批次，关闭页面后已完成批次仍然保留；继续执行始终沿用该会话创建时的审核配置。
        </div>
      </div>

      {controller.error && (
        <div
          style={{
            padding: "10px 12px",
            borderRadius: "8px",
            background: "#FEF2F2",
            border:
              "1px solid #FECACA",
            color: C.danger,
            fontSize: "12px",
            lineHeight: 1.6,
          }}
        >
          ⚠️ {controller.error}
        </div>
      )}

      {session && (
        <div
          style={{
            padding: "12px",
            borderRadius: "10px",
            border: `1px solid ${C.border}`,
            background: C.card,
          }}
        >
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "8px",
              marginBottom: "8px",
            }}
          >
            <span
              style={{
                fontSize: "12px",
                fontWeight: 700,
                color: C.text,
              }}
            >
              {controller.isSelfReview &&
              session.status === "done"
                ? "自审分析完成"
                : STATUS_LABELS[
                    session.status
                  ] ||
                  session.status}
            </span>

            <span
              style={{
                marginLeft: "auto",
                fontSize: "11px",
                color: C.textMuted,
              }}
            >
              {
                controller.completedBatches
              }
              /{controller.totalBatches} 批
            </span>
          </div>

          <div
            style={{
              height: "7px",
              borderRadius: "4px",
              background: "#E5E7EB",
              overflow: "hidden",
            }}
          >
            <div
              style={{
                width: `${controller.progress}%`,
                height: "100%",
                background:
                  session.status === "failed"
                    ? C.danger
                    : session.status ===
                        "done"
                      ? C.success
                      : C.primary,
                transition:
                  "width 200ms",
              }}
            />
          </div>

          <div
            style={{
              display: "flex",
              gap: "10px",
              marginTop: "8px",
              fontSize: "10px",
              color: C.textMuted,
              flexWrap: "wrap",
            }}
          >
            <span>
              模型：
              {session.model_used ||
                "按批记录"}
            </span>

            <span>
              Token：
              {session.tokens_used ||
                0}
            </span>

            <span>
              创建：
              {formatTime(
                session.created_at,
              )}
            </span>

            {controller.items.length >
              0 && (
              <span>
                整改项：
                {controller.items.length}
              </span>
            )}
          </div>

          {session.error_message && (
            <div
              style={{
                marginTop: "8px",
                color: C.danger,
                fontSize: "11px",
                lineHeight: 1.5,
              }}
            >
              {session.error_message}
            </div>
          )}
        </div>
      )}

      {controller.bundle.batches.length >
        0 && (
        <div
          style={{
            padding: "12px",
            borderRadius: "10px",
            border: `1px solid ${C.border}`,
            background: C.card,
          }}
        >
          <div
            style={{
              fontSize: "12px",
              fontWeight: 700,
              color: C.text,
              marginBottom: "8px",
            }}
          >
            分批审核进度
          </div>

          <div
            style={{
              display: "flex",
              gap: "5px",
              flexWrap: "wrap",
            }}
          >
            {controller.bundle.batches.map(
              (batch) => {
                const color =
                  batch.status === "done"
                    ? C.success
                    : batch.status ===
                        "failed"
                      ? C.danger
                      : batch.status ===
                          "running"
                        ? C.primary
                        : C.textMuted;

                return (
                  <div
                    key={batch.id}
                    title={
                      batch.error_message ||
                      BATCH_STATUS_LABELS[
                        batch.status
                      ]
                    }
                    style={{
                      padding: "4px 7px",
                      borderRadius: "6px",
                      border: `1px solid ${color}50`,
                      background: `${color}10`,
                      color,
                      fontSize: "10px",
                      fontWeight: 600,
                    }}
                  >
                    B{batch.batch_no} ·{" "}
                    {
                      BATCH_STATUS_LABELS[
                        batch.status
                      ]
                    }
                  </div>
                );
              },
            )}
          </div>
        </div>
      )}

      {!controller.isSelfReview &&
        controller.selectedItemIds.length >
          0 && (
        <div
          style={{
            padding: "9px 11px",
            borderRadius: "8px",
            background: "#FFF7ED",
            border:
              "1px solid #FED7AA",
            color: "#9A3412",
            fontSize: "10px",
            lineHeight: 1.6,
          }}
        >
          已确认并待交付{" "}
          {
            controller.selectedItemIds
              .length
          }{" "}
          条页级修改指令。执行“退回修改”时才会正式交付作者；仍可在页级审核清单中手动排除。
        </div>
      )}
    </>
  );
}

function actionButtonStyle(
  disabled: boolean,
): CSSProperties {
  return {
    width: "100%",
    padding: "9px 12px",
    borderRadius: "8px",
    border: "none",
    background: disabled
      ? "#D1D5DB"
      : C.primary,
    color: "#fff",
    fontSize: "13px",
    fontWeight: 700,
    cursor: disabled
      ? "not-allowed"
      : "pointer",
  };
}
