/**
 * CWAIReviewConfigSummary.tsx
 *
 * 最终报告和历史详情中的R-02审核配置摘要。
 *
 * 展示内容来自后端不可变会话或最终报告快照：
 *   - 审核维度；
 *   - 自定义审核要求；
 *   - 教案参考模式；
 *   - 是否允许使用教案类材料；
 *   - 配置哈希。
 *
 * 本组件不提供编辑能力，也不根据页面状态自行猜测配置。
 */

import {
  getCWAIReviewDimensionLabel,
  getCWAIReviewLessonReferenceLabel,
  type CWAIReviewConfigSnapshot,
} from "@/api/coursewares.ai-review-config";

const C = {
  primary: "#4F7BE8",
  text: "#1F2937",
  textSec: "#64748B",
  textMuted: "#94A3B8",
  border: "#DBEAFE",
  background: "#F8FAFF",
  chipBackground: "#EEF4FF",
  chipBorder: "#BFDBFE",
  warning: "#9A3412",
};

export interface CWAIReviewConfigSummaryProps {
  config:
    | CWAIReviewConfigSnapshot
    | null;

  title?: string;
}

export default function CWAIReviewConfigSummary({
  config,
  title = "本次审核配置",
}: CWAIReviewConfigSummaryProps) {
  if (!config) {
    return null;
  }

  const customSelected =
    config.review_dimensions
      .includes("custom");

  const hash =
    config.review_config_hash
      .trim();

  return (
    <div
      style={{
        padding: "12px",
        borderRadius: "10px",
        border:
          `1px solid ${C.border}`,
        background:
          C.background,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems:
            "flex-start",
          gap: "8px",
          marginBottom: "9px",
        }}
      >
        <div
          style={{
            minWidth: 0,
            flex: 1,
          }}
        >
          <div
            style={{
              color: C.primary,
              fontSize: "12px",
              fontWeight: 700,
            }}
          >
            ⚙ {title}
          </div>

          <div
            style={{
              marginTop: "3px",
              color: C.textSec,
              fontSize: "10px",
              lineHeight: 1.55,
            }}
          >
            以下配置在会话创建时固化，全部批次、最终报告和整改项均沿用同一快照。
          </div>
        </div>

        <span
          style={{
            flexShrink: 0,
            padding: "3px 6px",
            borderRadius: "999px",
            border:
              "1px solid #CBD5E1",
            background:
              "#F1F5F9",
            color: C.textSec,
            fontSize: "9px",
            fontWeight: 700,
          }}
        >
          协议 v
          {
            config
              .schema_version
          }
        </span>
      </div>

      <div
        style={{
          marginBottom: "5px",
          color: C.textSec,
          fontSize: "10px",
          fontWeight: 700,
        }}
      >
        审核维度
      </div>

      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          gap: "5px",
        }}
      >
        {config.review_dimensions.map(
          (dimension) => (
            <span
              key={dimension}
              style={{
                padding: "4px 7px",
                borderRadius: "999px",
                border:
                  `1px solid ${C.chipBorder}`,
                background:
                  C.chipBackground,
                color: "#1D4ED8",
                fontSize: "9px",
                fontWeight: 700,
              }}
            >
              {getCWAIReviewDimensionLabel(
                dimension,
              )}
            </span>
          ),
        )}
      </div>

      {customSelected &&
        config
          .custom_dimension_description
          .trim() && (
        <div
          style={{
            marginTop: "8px",
            padding: "7px 8px",
            borderRadius: "7px",
            border:
              "1px solid #E2E8F0",
            background: "#FFFFFF",
          }}
        >
          <div
            style={{
              color: C.textSec,
              fontSize: "9px",
              fontWeight: 700,
              marginBottom: "3px",
            }}
          >
            自定义审核要求
          </div>

          <div
            style={{
              color: C.text,
              fontSize: "10px",
              lineHeight: 1.6,
              whiteSpace:
                "pre-wrap",
            }}
          >
            {
              config
                .custom_dimension_description
            }
          </div>
        </div>
      )}

      <div
        style={{
          display: "grid",
          gridTemplateColumns:
            "minmax(0, 1fr) minmax(0, 1fr)",
          gap: "7px",
          marginTop: "9px",
        }}
      >
        <div
          style={{
            padding: "7px 8px",
            borderRadius: "7px",
            border:
              "1px solid #E2E8F0",
            background: "#FFFFFF",
          }}
        >
          <div
            style={{
              color: C.textMuted,
              fontSize: "9px",
              marginBottom: "2px",
            }}
          >
            教案参考模式
          </div>

          <div
            style={{
              color: C.text,
              fontSize: "10px",
              fontWeight: 700,
            }}
          >
            {
              config
                .lesson_reference_label ||
              getCWAIReviewLessonReferenceLabel(
                config
                  .lesson_reference_mode,
              )
            }
          </div>
        </div>

        <div
          style={{
            padding: "7px 8px",
            borderRadius: "7px",
            border:
              "1px solid #E2E8F0",
            background: "#FFFFFF",
          }}
        >
          <div
            style={{
              color: C.textMuted,
              fontSize: "9px",
              marginBottom: "2px",
            }}
          >
            教案类材料
          </div>

          <div
            style={{
              color:
                config
                  .uses_lesson_materials
                  ? C.text
                  : C.warning,
              fontSize: "10px",
              fontWeight: 700,
            }}
          >
            {config
              .uses_lesson_materials
              ? "允许按模式使用"
              : "未输入AI"}
          </div>
        </div>
      </div>

      {config
        .lesson_reference_mode ===
        "no_lesson" && (
        <div
          style={{
            marginTop: "8px",
            padding: "7px 8px",
            borderRadius: "7px",
            border:
              "1px solid #FED7AA",
            background:
              "#FFF7ED",
            color: C.warning,
            fontSize: "9px",
            lineHeight: 1.55,
          }}
        >
          本次审核采用“不使用教案”：后端未读取或输入教案正文、课程大纲和对齐报告。
        </div>
      )}

      {hash && (
        <div
          style={{
            marginTop: "8px",
            paddingTop: "7px",
            borderTop:
              "1px solid #E2E8F0",
            color: C.textMuted,
            fontSize: "9px",
            lineHeight: 1.5,
          }}
        >
          配置指纹：
          <span
            title={hash}
            style={{
              marginLeft: "4px",
              fontFamily:
                "ui-monospace, SFMono-Regular, Menlo, monospace",
              color: C.textSec,
            }}
          >
            {hash.slice(
              0,
              16,
            )}
            …
          </span>
        </div>
      )}
    </div>
  );
}
