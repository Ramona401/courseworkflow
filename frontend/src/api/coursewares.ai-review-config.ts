/**
 * coursewares.ai-review-config.ts
 *
 * R-02课件AI审核配置的浏览器类型、运行时解析和启动请求。
 *
 * 单独拆分原因：
 *   - coursewares.ai-review.ts已经接近900行；
 *   - 审核配置有独立协议和运行时守卫；
 *   - 避免继续扩大整改项、全局讨论和会话API的单一文件。
 *
 * 安全边界：
 *   - 浏览器只提交维度代码、自定义说明和教案参考模式；
 *   - 不提交配置哈希、教案ID、材料可用状态或任何正文；
 *   - 会话配置和最终报告配置只作为后端快照读取；
 *   - 解析失败时不猜测配置事实。
 */

import apiClient from "./client";
import type {
  CWAIReviewFinalReport,
  CWAIReviewSession,
  CWAIReviewSessionBundle,
} from "./coursewares.ai-review";
import {
  extractData,
} from "./coursewares.types";

export type CWAIReviewDimension =
  | "teaching_logic"
  | "technical_implementation"
  | "interaction_experience"
  | "lesson_alignment"
  | "authenticity"
  | "knowledge_accuracy"
  | "page_readability"
  | "operational_usability"
  | "custom";

export type CWAIReviewLessonReferenceMode =
  | "current_compatible"
  | "strict_alignment"
  | "lesson_intent"
  | "no_lesson";

export interface CWAIReviewDimensionOption {
  code: CWAIReviewDimension;
  label: string;
}

export interface CWAIReviewLessonReferenceOption {
  code: CWAIReviewLessonReferenceMode;
  label: string;
}

export interface CWAIReviewConfigSnapshot {
  schema_version: number;

  review_dimensions: CWAIReviewDimension[];
  review_dimension_items: CWAIReviewDimensionOption[];

  custom_dimension_description: string;

  lesson_reference_mode: CWAIReviewLessonReferenceMode;
  lesson_reference_label: string;
  uses_lesson_materials: boolean;

  review_config_hash: string;
}

export interface CWAIReviewConfigDraft {
  review_dimensions: CWAIReviewDimension[];
  custom_dimension_description: string;
  lesson_reference_mode: CWAIReviewLessonReferenceMode;
}

export interface PrepareConfiguredCWAIReviewRequest {
  courseware_id: string;
  review_level: number;
  assistant_id?: string;

  review_dimensions: CWAIReviewDimension[];
  custom_dimension_description: string;
  lesson_reference_mode: CWAIReviewLessonReferenceMode;
}

export const CW_AI_REVIEW_DIMENSION_OPTIONS:
  readonly CWAIReviewDimensionOption[] = [
  {
    code: "teaching_logic",
    label: "教学逻辑",
  },
  {
    code: "technical_implementation",
    label: "技术实现",
  },
  {
    code: "interaction_experience",
    label: "交互体验",
  },
  {
    code: "lesson_alignment",
    label: "教案一致性",
  },
  {
    code: "authenticity",
    label: "真实性",
  },
  {
    code: "knowledge_accuracy",
    label: "知识严谨性",
  },
  {
    code: "page_readability",
    label: "页面可读性",
  },
  {
    code: "operational_usability",
    label: "操作可用性",
  },
  {
    code: "custom",
    label: "自定义维度",
  },
];

export const CW_AI_REVIEW_LESSON_REFERENCE_OPTIONS:
  readonly CWAIReviewLessonReferenceOption[] = [
  {
    code: "current_compatible",
    label: "现行兼容",
  },
  {
    code: "strict_alignment",
    label: "严格一致",
  },
  {
    code: "lesson_intent",
    label: "参考教案意图",
  },
  {
    code: "no_lesson",
    label: "不使用教案",
  },
];

export const DEFAULT_CW_AI_REVIEW_DIMENSIONS:
  readonly CWAIReviewDimension[] =
    CW_AI_REVIEW_DIMENSION_OPTIONS
      .map((option) => option.code)
      .filter(
        (code) => code !== "custom",
      );

const DIMENSION_SET =
  new Set<CWAIReviewDimension>(
    CW_AI_REVIEW_DIMENSION_OPTIONS.map(
      (option) => option.code,
    ),
  );

const LESSON_REFERENCE_SET =
  new Set<CWAIReviewLessonReferenceMode>(
    CW_AI_REVIEW_LESSON_REFERENCE_OPTIONS.map(
      (option) => option.code,
    ),
  );

function isObject(
  value: unknown,
): value is Record<string, unknown> {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value)
  );
}

function isDimension(
  value: unknown,
): value is CWAIReviewDimension {
  return (
    typeof value === "string" &&
    DIMENSION_SET.has(
      value as CWAIReviewDimension,
    )
  );
}

function isLessonReferenceMode(
  value: unknown,
): value is CWAIReviewLessonReferenceMode {
  return (
    typeof value === "string" &&
    LESSON_REFERENCE_SET.has(
      value as CWAIReviewLessonReferenceMode,
    )
  );
}

export function getCWAIReviewDimensionLabel(
  dimension: CWAIReviewDimension,
): string {
  return (
    CW_AI_REVIEW_DIMENSION_OPTIONS.find(
      (option) => option.code === dimension,
    )?.label || dimension
  );
}

export function getCWAIReviewLessonReferenceLabel(
  mode: CWAIReviewLessonReferenceMode,
): string {
  return (
    CW_AI_REVIEW_LESSON_REFERENCE_OPTIONS.find(
      (option) => option.code === mode,
    )?.label || mode
  );
}

export function sortCWAIReviewDimensions(
  dimensions: readonly CWAIReviewDimension[],
): CWAIReviewDimension[] {
  const selected = new Set(dimensions);

  return CW_AI_REVIEW_DIMENSION_OPTIONS
    .map((option) => option.code)
    .filter((code) => selected.has(code));
}

export function createDefaultCWAIReviewConfigDraft():
  CWAIReviewConfigDraft {
  return {
    review_dimensions: [
      ...DEFAULT_CW_AI_REVIEW_DIMENSIONS,
    ],
    custom_dimension_description: "",
    lesson_reference_mode:
      "current_compatible",
  };
}

export function toCWAIReviewConfigDraft(
  snapshot: CWAIReviewConfigSnapshot,
): CWAIReviewConfigDraft {
  return {
    review_dimensions: [
      ...snapshot.review_dimensions,
    ],
    custom_dimension_description:
      snapshot.custom_dimension_description,
    lesson_reference_mode:
      snapshot.lesson_reference_mode,
  };
}

function parseCWAIReviewConfigSnapshot(
  value: unknown,
): CWAIReviewConfigSnapshot | null {
  if (!isObject(value)) {
    return null;
  }

  if (
    typeof value.schema_version !== "number" ||
    !Array.isArray(value.review_dimensions) ||
    !isLessonReferenceMode(
      value.lesson_reference_mode,
    )
  ) {
    return null;
  }

  const dimensions =
    value.review_dimensions.filter(
      isDimension,
    );

  if (
    dimensions.length === 0 ||
    dimensions.length !==
      value.review_dimensions.length
  ) {
    return null;
  }

  const normalizedDimensions =
    sortCWAIReviewDimensions(dimensions);

  if (
    normalizedDimensions.length !==
    dimensions.length
  ) {
    return null;
  }

  const customDescription =
    typeof value.custom_dimension_description ===
    "string"
      ? value.custom_dimension_description
      : "";

  const hasCustom =
    normalizedDimensions.includes("custom");

  if (
    (hasCustom &&
      !customDescription.trim()) ||
    (!hasCustom &&
      !!customDescription.trim())
  ) {
    return null;
  }

  const mode = value.lesson_reference_mode;

  return {
    schema_version: value.schema_version,

    review_dimensions:
      normalizedDimensions,

    review_dimension_items:
      normalizedDimensions.map((code) => ({
        code,
        label:
          getCWAIReviewDimensionLabel(code),
      })),

    custom_dimension_description:
      customDescription,

    lesson_reference_mode: mode,
    lesson_reference_label:
      typeof value.lesson_reference_label ===
      "string"
        ? value.lesson_reference_label
        : getCWAIReviewLessonReferenceLabel(
            mode,
          ),
    uses_lesson_materials:
      typeof value.uses_lesson_materials ===
      "boolean"
        ? value.uses_lesson_materials
        : mode !== "no_lesson",

    review_config_hash:
      typeof value.review_config_hash ===
      "string"
        ? value.review_config_hash
        : "",
  };
}

export function readCWAIReviewSessionConfig(
  session: CWAIReviewSession | null,
): CWAIReviewConfigSnapshot | null {
  if (!session) {
    return null;
  }

  const sessionObject =
    session as unknown as Record<
      string,
      unknown
    >;

  return parseCWAIReviewConfigSnapshot(
    sessionObject.review_config,
  );
}

export function readCWAIReviewFinalReportConfig(
  report: CWAIReviewFinalReport | null,
): CWAIReviewConfigSnapshot | null {
  if (!report) {
    return null;
  }

  const reportObject =
    report as unknown as Record<
      string,
      unknown
    >;

  return parseCWAIReviewConfigSnapshot(
    reportObject.review_config,
  );
}

export async function prepareConfiguredCWAIReview(
  request: PrepareConfiguredCWAIReviewRequest,
): Promise<CWAIReviewSessionBundle> {
  const response = await apiClient.post(
    "/courseware-ai-reviews",
    {
      courseware_id: request.courseware_id,
      review_level: request.review_level,
      assistant_id: request.assistant_id || "",

      review_dimensions:
        request.review_dimensions,
      custom_dimension_description:
        request.custom_dimension_description,
      lesson_reference_mode:
        request.lesson_reference_mode,
    },
    {
      timeout: 300000,
    },
  );

  return extractData<CWAIReviewSessionBundle>(
    response,
  );
}
