/**
 * coursewares.ai-review.ts
 *
 * 课件AI审核、自审、整改项和全局讨论API。
 *
 * 浏览器安全边界：
 *   - 会话只接收进度、最终报告、模型、Token、错误和时间；
 *   - 批次只接收审核结果与风险页；
 *   - 不声明或依赖系统提示词、助手提示词、教案正文、课程大纲全文、
 *     页面代码索引、批次输入清单和连续性账本等后端内部字段；
 *   - 全局讨论采用候选时只提交可信消息ID和整改项ID，
 *     不由浏览器回传候选指令正文。
 *
 * 整改闭环：
 *   1. 最终报告完成后，用户明确选择finding；
 *   2. 后端按审核快照映射稳定page_id并物化整改项；
 *   3. 每条整改项可独立与AI讨论；
 *   4. 用户也可跳过形式化聊天，直接生成候选修改指令；
 *   5. 多条整改项可进入跨页面、跨问题全局讨论；
 *   6. 全局讨论候选采用后仍需在单条整改项中独立确认；
 *   7. 最终修改指令通过独立confirm接口确认；
 *   8. 未交付整改项允许人工忽略和恢复；
 *   9. 所有讨论、生成、采用、确认、忽略和恢复动作均不会自动修改页面。
 */

import apiClient from "./client";
import { extractData } from "./coursewares.types";

import type {
  CWAIReviewBatch,
  CWAIReviewBatchResult,
  CWAIReviewFinalReport,
  CWAIReviewFinalizeResponse,
  CWAIReviewGlobalDiscussion,
  CWAIReviewItem,
  CWAIReviewItemDiscussion,
  CWAIReviewItemListResponse,
  CWAIReviewItemMessage,
  CWAIReviewItemMessageMeta,
  CWAIReviewRunNextResponse,
  CWAIReviewSession,
  CWAIReviewSeverity,
  CWAIReviewSessionBundle,
  CWOwnerReviewRemediationResponse,
  PrepareCWAIReviewRequest,
} from "./coursewares.ai-review.types";

export * from "./coursewares.ai-review.types";

// ==================== JSON解析辅助 ====================

export function parseCWAIReviewJSON<T>(
  raw: string | null | undefined,
  fallback: T,
): T {
  if (!raw?.trim()) {
    return fallback;
  }

  try {
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function isCWAIReviewObject(
  value: unknown,
): value is Record<string, unknown> {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value)
  );
}

function isCWAIReviewBatchResult(
  value: unknown,
): value is CWAIReviewBatchResult {
  if (!isCWAIReviewObject(value)) {
    return false;
  }

  return (
    typeof value.batch_no === "number" &&
    Array.isArray(value.page_numbers) &&
    typeof value.batch_summary === "string" &&
    Array.isArray(value.findings) &&
    isCWAIReviewObject(value.continuity_ledger) &&
    Array.isArray(value.risk_pages) &&
    typeof value.manual_review_required === "boolean"
  );
}

function isCWAIReviewFinalReport(
  value: unknown,
): value is CWAIReviewFinalReport {
  if (!isCWAIReviewObject(value)) {
    return false;
  }

  const severityValues: CWAIReviewSeverity[] = [
    "critical",
    "high",
    "medium",
    "low",
    "info",
  ];

  return (
    typeof value.overall_risk === "string" &&
    severityValues.includes(
      value.overall_risk as CWAIReviewSeverity,
    ) &&
    typeof value.summary === "string" &&
    Array.isArray(value.strengths) &&
    Array.isArray(value.findings) &&
    Array.isArray(value.priority_actions) &&
    Array.isArray(value.manual_review_pages) &&
    typeof value.review_comment_draft === "string" &&
    typeof value.human_decision_reminder === "string"
  );
}

export function parseCWAIReviewBatchResult(
  batch: CWAIReviewBatch,
): CWAIReviewBatchResult | null {
  const parsed = parseCWAIReviewJSON<unknown>(
    batch.result_json,
    null,
  );

  return isCWAIReviewBatchResult(parsed)
    ? parsed
    : null;
}

export function parseCWAIReviewFinalReport(
  session: CWAIReviewSession | null,
): CWAIReviewFinalReport | null {
  if (!session || session.status !== "done") {
    return null;
  }

  const parsed = parseCWAIReviewJSON<unknown>(
    session.final_report_json,
    null,
  );

  return isCWAIReviewFinalReport(parsed)
    ? parsed
    : null;
}

export function parseCWAIReviewItemEvidence(
  item: CWAIReviewItem,
): Record<string, unknown> {
  const parsed = parseCWAIReviewJSON<unknown>(
    item.evidence_json,
    {},
  );

  return isCWAIReviewObject(parsed)
    ? parsed
    : {};
}

export function parseCWAIReviewItemMessageMeta(
  message: CWAIReviewItemMessage,
): CWAIReviewItemMessageMeta {
  const fallback: CWAIReviewItemMessageMeta = {
    summary: "",
    ready_for_confirmation: false,
    suggested_instruction: "",
    citations: [],
  };

  const parsed = parseCWAIReviewJSON<unknown>(
    message.meta_json,
    fallback,
  );

  if (!isCWAIReviewObject(parsed)) {
    return fallback;
  }

  return {
    summary:
      typeof parsed.summary === "string"
        ? parsed.summary
        : "",
    ready_for_confirmation:
      parsed.ready_for_confirmation === true,
    suggested_instruction:
      typeof parsed.suggested_instruction === "string"
        ? parsed.suggested_instruction
        : "",
    citations: Array.isArray(parsed.citations)
      ? parsed.citations.filter(
          (
            value,
          ): value is Record<string, unknown> =>
            isCWAIReviewObject(value),
        )
      : [],
  };
}

// ==================== AI审核会话API ====================

export async function prepareCWAIReview(
  request: PrepareCWAIReviewRequest,
): Promise<CWAIReviewSessionBundle> {
  const response = await apiClient.post(
    "/courseware-ai-reviews",
    {
      courseware_id: request.courseware_id,
      review_level: request.review_level,
      assistant_id: request.assistant_id || "",
    },
    {
      timeout: 300000,
    },
  );

  return extractData<CWAIReviewSessionBundle>(
    response,
  );
}

export async function getLatestCWAIReview(
  coursewareId: string,
  reviewLevel: number,
): Promise<CWAIReviewSessionBundle> {
  const response = await apiClient.get(
    "/courseware-ai-reviews",
    {
      params: {
        courseware_id: coursewareId,
        review_level: reviewLevel,
      },
    },
  );

  return extractData<CWAIReviewSessionBundle>(
    response,
  );
}

export async function getCWAIReviewSession(
  sessionId: string,
): Promise<CWAIReviewSessionBundle> {
  const response = await apiClient.get(
    `/courseware-ai-reviews/${sessionId}`,
  );

  return extractData<CWAIReviewSessionBundle>(
    response,
  );
}

export async function runNextCWAIReviewBatch(
  sessionId: string,
): Promise<CWAIReviewRunNextResponse> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/run-next`,
    {},
    {
      timeout: 300000,
    },
  );

  return extractData<CWAIReviewRunNextResponse>(
    response,
  );
}

export async function finalizeCWAIReview(
  sessionId: string,
): Promise<CWAIReviewFinalizeResponse> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/finalize`,
    {},
    {
      timeout: 300000,
    },
  );

  return extractData<CWAIReviewFinalizeResponse>(
    response,
  );
}

// ==================== 全局讨论API ====================

export async function getCWAIReviewGlobalDiscussion(
  sessionId: string,
): Promise<CWAIReviewGlobalDiscussion> {
  const response = await apiClient.get(
    `/courseware-ai-reviews/${sessionId}/global-discussion`,
  );

  return extractData<CWAIReviewGlobalDiscussion>(
    response,
  );
}

export async function messageCWAIReviewGlobalDiscussion(
  sessionId: string,
  content: string,
  itemIds: string[],
): Promise<CWAIReviewGlobalDiscussion> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/global-discussion`,
    {
      content,
      item_ids: itemIds,
    },
    {
      timeout: 300000,
    },
  );

  return extractData<CWAIReviewGlobalDiscussion>(
    response,
  );
}

/**
 * 采用全局讨论中已经保存的可信候选指令。
 *
 * 浏览器不提交指令正文；后端按消息ID和整改项ID重新读取候选。
 * 采用结果仍需通过单条整改项confirm接口独立确认。
 */
export async function adoptCWAIReviewGlobalProposal(
  sessionId: string,
  messageId: string,
  itemId: string,
): Promise<CWAIReviewItemDiscussion> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/global-discussion/adopt`,
    {
      message_id: messageId,
      item_id: itemId,
    },
  );

  return extractData<CWAIReviewItemDiscussion>(
    response,
  );
}

// ==================== 整改项API ====================

export async function materializeCWAIReviewItems(
  sessionId: string,
  findingIds: string[],
): Promise<CWAIReviewItemListResponse> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/items`,
    {
      finding_ids: findingIds,
    },
  );

  return extractData<CWAIReviewItemListResponse>(
    response,
  );
}

export async function getCWAIReviewSessionItems(
  sessionId: string,
): Promise<CWAIReviewItemListResponse> {
  const response = await apiClient.get(
    `/courseware-ai-reviews/${sessionId}/items`,
  );

  return extractData<CWAIReviewItemListResponse>(
    response,
  );
}

export async function getCWOwnerReviewRemediation(
  coursewareId: string,
): Promise<CWOwnerReviewRemediationResponse> {
  const response = await apiClient.get(
    "/courseware-ai-reviews/items",
    {
      params: {
        courseware_id: coursewareId,
      },
    },
  );

  return extractData<CWOwnerReviewRemediationResponse>(
    response,
  );
}

export async function getCWOwnerReviewItems(
  coursewareId: string,
): Promise<CWAIReviewItemListResponse> {
  const result =
    await getCWOwnerReviewRemediation(
      coursewareId,
    );

  return {
    items: result.items || [],
  };
}

export async function getCWAIReviewItemDiscussion(
  itemId: string,
): Promise<CWAIReviewItemDiscussion> {
  const response = await apiClient.get(
    `/courseware-ai-reviews/items/${itemId}`,
  );

  return extractData<CWAIReviewItemDiscussion>(
    response,
  );
}

export async function messageCWAIReviewItem(
  itemId: string,
  content: string,
): Promise<CWAIReviewItemDiscussion> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/items/${itemId}/messages`,
    {
      content,
    },
    {
      timeout: 300000,
    },
  );

  return extractData<CWAIReviewItemDiscussion>(
    response,
  );
}

/**
 * 不要求先发送“同意”，直接生成一条候选修改指令。
 *
 * 生成结果仍需用户通过独立confirm接口确认。
 */
export async function generateCWAIReviewItemInstruction(
  itemId: string,
): Promise<CWAIReviewItemDiscussion> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/items/${itemId}/generate-instruction`,
    {},
    {
      timeout: 300000,
    },
  );

  return extractData<CWAIReviewItemDiscussion>(
    response,
  );
}

export async function confirmCWAIReviewItem(
  itemId: string,
  instruction: string,
): Promise<CWAIReviewItemDiscussion> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/items/${itemId}/confirm`,
    {
      instruction,
    },
  );

  return extractData<CWAIReviewItemDiscussion>(
    response,
  );
}

export async function dismissCWAIReviewItem(
  itemId: string,
  reason: string,
): Promise<CWAIReviewItemDiscussion> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/items/${itemId}/dismiss`,
    {
      reason,
    },
  );

  return extractData<CWAIReviewItemDiscussion>(
    response,
  );
}

export async function restoreCWAIReviewItem(
  itemId: string,
): Promise<CWAIReviewItemDiscussion> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/items/${itemId}/restore`,
    {},
  );

  return extractData<CWAIReviewItemDiscussion>(
    response,
  );
}
