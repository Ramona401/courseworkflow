/**
 * 教学智能体安全上下文回执。
 *
 * 只展示后端允许返回浏览器的安全摘要：
 *   - 当前页与相邻页教学摘要；
 *   - 来源教案相关片段预览；
 *   - 静态互动证据和风险标志。
 *
 * 不展示页面完整HTML、教案全文、助手提示词或隐藏推理。
 */

import type {
  CoursewareAssistantContextPreview,
} from "@/api/coursewares";

import {
  COURSEWARE_ASSISTANT_EDITOR_COLORS,
  CoursewareAssistantSection,
} from "./CoursewareAssistantEditorShared";

interface CoursewareAssistantContextReceiptProps {
  preview:
    CoursewareAssistantContextPreview | null;
  loading: boolean;
  error: string;
  onRefresh: () => void;
}

export default function CoursewareAssistantContextReceipt({
  preview,
  loading,
  error,
  onRefresh,
}: CoursewareAssistantContextReceiptProps) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <CoursewareAssistantSection
      title="🧾 上下文回执"
      description="这是后端按稳定页面生成的安全摘要，用于确认AI将依据哪些可信材料工作。"
      actions={
        <button
          type="button"
          onClick={onRefresh}
          disabled={loading}
          style={{
            padding: "5px 10px",
            borderRadius: 7,
            border:
              `1px solid ${C.border}`,
            background: C.white,
            color: C.primary,
            fontSize: 10,
            fontWeight: 700,
            cursor:
              loading
                ? "default"
                : "pointer",
            opacity:
              loading
                ? 0.5
                : 1,
          }}
        >
          {loading
            ? "读取中…"
            : "🔄 刷新回执"}
        </button>
      }
    >
      {loading && (
        <div
          style={{
            padding: "18px 12px",
            textAlign: "center",
            color: C.textMuted,
            fontSize: 11,
          }}
        >
          正在构建当前页面的安全上下文回执…
        </div>
      )}

      {!loading &&
        error && (
        <div
          style={{
            padding: "10px 12px",
            borderRadius: 8,
            border:
              "1px solid #FECACA",
            background:
              "#FEF2F2",
            color: C.danger,
            fontSize: 11,
            lineHeight: 1.65,
          }}
        >
          ❌ {error}
        </div>
      )}

      {!loading &&
        !error &&
        !preview && (
        <div
          style={{
            padding: "14px 12px",
            borderRadius: 8,
            border:
              `1px dashed ${C.border}`,
            color: C.textMuted,
            fontSize: 11,
            textAlign: "center",
          }}
        >
          暂无上下文回执。
        </div>
      )}

      {!loading &&
        !error &&
        preview && (
        <>
          <div
            style={{
              display: "grid",
              gridTemplateColumns:
                "repeat(auto-fit, minmax(210px, 1fr))",
              gap: 9,
            }}
          >
            <PageReceipt
              label="当前页面"
              emoji="📄"
              title={
                `P${preview.current_page.page_number} · ${
                  preview.current_page.title ||
                  "未命名页面"
                }`
              }
              purpose={
                preview.current_page.purpose
              }
              summary={
                preview.current_page.content_summary
              }
              visibleText={
                preview.current_page.visible_text
              }
            />

            {preview.previous_page && (
              <PageReceipt
                label="前一页衔接"
                emoji="⬅️"
                title={
                  `P${preview.previous_page.page_number} · ${
                    preview.previous_page.title ||
                    "未命名页面"
                  }`
                }
                purpose={
                  preview.previous_page.purpose
                }
                summary={
                  preview.previous_page.content_summary
                }
              />
            )}

            {preview.next_page && (
              <PageReceipt
                label="后一页衔接"
                emoji="➡️"
                title={
                  `P${preview.next_page.page_number} · ${
                    preview.next_page.title ||
                    "未命名页面"
                  }`
                }
                purpose={
                  preview.next_page.purpose
                }
                summary={
                  preview.next_page.content_summary
                }
              />
            )}
          </div>

          {preview.lesson_plan && (
            <div
              style={{
                marginTop: 10,
                padding: 11,
                borderRadius: 9,
                border:
                  `1px solid ${C.border}`,
                background: C.background,
              }}
            >
              <div
                style={{
                  color: C.text,
                  fontSize: 11,
                  fontWeight: 700,
                }}
              >
                📘 来源教案相关片段
                {" · "}
                {preview.lesson_plan.title ||
                  "未命名教案"}
              </div>

              <div
                style={{
                  marginTop: 5,
                  color:
                    C.textSecondary,
                  fontSize: 10,
                  lineHeight: 1.65,
                  whiteSpace: "pre-wrap",
                  wordBreak: "break-word",
                }}
              >
                {preview.lesson_plan.excerpt_preview ||
                  "当前没有可展示的相关片段。"}
              </div>

              <div
                style={{
                  marginTop: 5,
                  color: C.textMuted,
                  fontSize: 9,
                }}
              >
                安全回执字符数：
                {preview.lesson_plan.character_count}
              </div>
            </div>
          )}

          <InteractionReceipt
            preview={preview}
          />

          <div
            style={{
              marginTop: 10,
              padding: "8px 10px",
              borderRadius: 8,
              background:
                "#EFF6FF",
              color: "#2563EB",
              fontSize: 10,
              lineHeight: 1.65,
            }}
          >
            🔒 回执只包含确定性摘要和静态证据，不代表系统已经观察到学生真实点击、输入、拖拽或实验结果。
          </div>
        </>
      )}
    </CoursewareAssistantSection>
  );
}

function PageReceipt({
  label,
  emoji,
  title,
  purpose,
  summary,
  visibleText,
}: {
  label: string;
  emoji: string;
  title: string;
  purpose: string;
  summary: string;
  visibleText?: string;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      style={{
        padding: 11,
        borderRadius: 9,
        border:
          `1px solid ${C.border}`,
        background: C.background,
      }}
    >
      <div
        style={{
          color: C.textMuted,
          fontSize: 9,
          fontWeight: 700,
        }}
      >
        {emoji} {label}
      </div>

      <div
        style={{
          marginTop: 4,
          color: C.text,
          fontSize: 11,
          fontWeight: 700,
        }}
      >
        {title}
      </div>

      {purpose && (
        <ReceiptText
          label="教学目的"
          value={purpose}
        />
      )}

      {summary && (
        <ReceiptText
          label="内容摘要"
          value={summary}
        />
      )}

      {visibleText && (
        <ReceiptText
          label="可见文字预览"
          value={visibleText}
        />
      )}
    </div>
  );
}

function ReceiptText({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      style={{
        marginTop: 6,
        color:
          C.textSecondary,
        fontSize: 9,
        lineHeight: 1.6,
        wordBreak: "break-word",
      }}
    >
      <strong>{label}：</strong>
      {value}
    </div>
  );
}

function InteractionReceipt({
  preview,
}: {
  preview:
    CoursewareAssistantContextPreview;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  const interaction =
    preview.interaction;

  const hasRisk =
    interaction.risk_flags.length > 0 ||
    interaction.manual_review_required ||
    !interaction.contract_ok;

  return (
    <div
      style={{
        marginTop: 10,
        padding: 11,
        borderRadius: 9,
        border:
          `1px solid ${
            hasRisk
              ? "#FDE68A"
              : "#A7F3D0"
          }`,
        background:
          hasRisk
            ? "#FFFBEB"
            : "#ECFDF5",
      }}
    >
      <div
        style={{
          color:
            hasRisk
              ? "#92400E"
              : C.success,
          fontSize: 11,
          fontWeight: 700,
        }}
      >
        🧩 互动静态证据
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns:
            "repeat(auto-fit, minmax(130px, 1fr))",
          gap: 7,
          marginTop: 8,
        }}
      >
        <Metric
          label="声明类型"
          value={
            interaction.declared_type ||
            "未声明"
          }
        />

        <Metric
          label="协议检查"
          value={
            interaction.contract_ok
              ? "通过"
              : "未通过"
          }
        />

        <Metric
          label="事件入口"
          value={
            String(
              interaction.event_count,
            )
          }
        />

        <Metric
          label="DOM目标"
          value={
            String(
              interaction.dom_target_count,
            )
          }
        />
      </div>

      {interaction.risk_flags.length >
        0 && (
        <div
          style={{
            marginTop: 8,
            color: "#92400E",
            fontSize: 9,
            lineHeight: 1.6,
          }}
        >
          <strong>风险标志：</strong>
          {interaction.risk_flags.join(
            "；",
          )}
        </div>
      )}

      {interaction.manual_review_required && (
        <div
          style={{
            marginTop: 5,
            color: "#92400E",
            fontSize: 9,
          }}
        >
          ⚠️ 当前页面互动需要教师人工复核。
        </div>
      )}
    </div>
  );
}

function Metric({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  const C =
    COURSEWARE_ASSISTANT_EDITOR_COLORS;

  return (
    <div
      style={{
        padding: "7px 8px",
        borderRadius: 7,
        background:
          "rgba(255,255,255,0.72)",
      }}
    >
      <div
        style={{
          color: C.textMuted,
          fontSize: 8,
        }}
      >
        {label}
      </div>

      <div
        style={{
          marginTop: 2,
          color: C.text,
          fontSize: 10,
          fontWeight: 700,
        }}
      >
        {value}
      </div>
    </div>
  );
}
