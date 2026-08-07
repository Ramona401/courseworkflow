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

// ==================== 基础类型 ====================

export type CWAIReviewSessionStatus =
  | "pending"
  | "preparing"
  | "reviewing"
  | "aggregating"
  | "done"
  | "failed"
  | "cancelled";

export type CWAIReviewBatchStatus =
  | "pending"
  | "running"
  | "done"
  | "failed";

export type CWAIReviewSeverity =
  | "critical"
  | "high"
  | "medium"
  | "low"
  | "info";

export type CWAIReviewItemSource =
  | "self"
  | "formal";

export type CWAIReviewItemOrigin =
  | "ai_finding"
  | "global_discussion_manual";

export type CWAIReviewItemStatus =
  | "detected"
  | "discussing"
  | "confirmed"
  | "applying"
  | "applied"
  | "resolved"
  | "dismissed"
  | "stale"
  | "orphaned";

// ==================== 会话与批次 ====================

export interface CWAIReviewSession {
  id: string;
  courseware_id: string;
  reviewer_id: string;
  assistant_id: string | null;
  lesson_plan_id: string | null;

  review_level: number;
  education_domain: string;
  subject: string;
  grade: string;

  status: CWAIReviewSessionStatus;
  current_stage: string;
  current_batch_no: number;
  total_batches: number;

  final_report_json: string;

  model_used: string;
  tokens_used: number;
  error_message: string;

  created_at: string | null;
  updated_at: string | null;
  completed_at: string | null;
}

export interface CWAIReviewBatch {
  id: string;
  session_id: string;
  batch_no: number;
  status: CWAIReviewBatchStatus;

  result_json: string;
  risk_pages_json: string;

  model_used: string;
  tokens_used: number;
  error_message: string;

  started_at: string | null;
  completed_at: string | null;
  created_at: string | null;
  updated_at: string | null;
}

// ==================== 审核发现 ====================

export interface CWAIReviewFinding {
  id: string;
  severity: CWAIReviewSeverity;
  dimension: string;
  page_numbers: number[];

  title: string;
  description: string;

  lesson_or_outline_basis: string;
  page_evidence: string;
  code_evidence: string;
  continuity_evidence: string;

  suggestion: string;
  confidence: number;
  manual_review_required: boolean;
}

export interface CWAIReviewRiskPage {
  page_number: number;
  severity: CWAIReviewSeverity;
  reason: string;
  evidence_type: string;
  manual_review_required: boolean;
}

export interface CWAIReviewBatchResult {
  batch_no: number;
  page_numbers: number[];
  batch_summary: string;
  findings: CWAIReviewFinding[];
  continuity_ledger: Record<string, unknown>;
  risk_pages: CWAIReviewRiskPage[];
  manual_review_required: boolean;
}

// ==================== 最终报告 ====================

export interface CWAIReviewPriorityAction {
  priority: number;
  title: string;
  description: string;
  page_numbers: number[];
  reason: string;
  manual_review_required: boolean;
}

export interface CWAIReviewFinalReport {
  overall_risk: CWAIReviewSeverity;
  summary: string;
  strengths: string[];
  findings: CWAIReviewFinding[];
  priority_actions: CWAIReviewPriorityAction[];
  manual_review_pages: number[];
  review_comment_draft: string;
  human_decision_reminder: string;
}

// ==================== 整改项 ====================

export interface CWAIReviewItem {
  id: string;
  courseware_id: string;

  source_session_id: string;
  source_finding_id: string;

  /**
   * ai_finding：由AI报告finding物化；
   * global_discussion_manual：由用户在全局讨论中人工新增。
   */
  origin_type: CWAIReviewItemOrigin;
  source_global_message_id: string | null;

  courseware_review_id: string | null;
  feedback_id: string | null;

  source_type: CWAIReviewItemSource;
  review_level: number;
  review_round: number;

  page_id: string | null;
  page_number_snapshot: number;
  page_title_snapshot: string;
  page_html_hash: string;
  page_updated_at_snapshot: string | null;

  severity: CWAIReviewSeverity;
  dimension: string;

  title: string;
  description: string;

  evidence_json: string;
  original_suggestion: string;

  confirmed_instruction: string;
  status: CWAIReviewItemStatus;
  applied_page_hash: string;

  created_at: string | null;
  updated_at: string | null;
  confirmed_at: string | null;
  applied_at: string | null;
  resolved_at: string | null;
}

export interface CWAIReviewItemMessage {
  id: string;
  role: "system" | "user" | "assistant";
  content: string;

  /**
   * 单条整改项AI回复保存方案摘要、候选指令和引用；
   * 全局讨论消息保存本轮选中项、关系和逐项候选指令；
   * 系统消息保存忽略、恢复等状态事件。
   */
  meta_json: string;

  model_used: string;
  tokens_used: number;
  created_at: string | null;
}

export interface CWAIReviewItemMessageMeta {
  summary: string;
  ready_for_confirmation: boolean;
  suggested_instruction: string;
  citations: Record<string, unknown>[];
}

export interface CWAIReviewItemDiscussion {
  item: CWAIReviewItem;
  messages: CWAIReviewItemMessage[];
  summary: string;
  ready_for_confirmation: boolean;
  suggested_instruction: string;
}

// ==================== 跨页面、跨问题全局讨论 ====================

export type CWAIReviewGlobalRelationType =
  | "duplicate"
  | "conflict"
  | "merge"
  | "dependency"
  | "possibly_resolved";

export type CWAIReviewGlobalRecommendation =
  | "keep"
  | "revise"
  | "merge"
  | "manual_review"
  | "consider_dismiss";

export interface CWAIReviewGlobalRelation {
  type: CWAIReviewGlobalRelationType;

  /**
   * duplicate：source重复target，target为保留主问题；
   * merge：source合并进入target；
   * dependency：source依赖target先完成；
   * possibly_resolved：source可能被target的修改连带解决；
   * conflict：无方向，后端按UUID文本顺序规范化。
   */
  source_item_id: string;
  target_item_id: string;

  /**
   * 必须严格等于[source_item_id, target_item_id]。
   * 保留该字段用于兼容全局讨论结构化元数据和审阅展示。
   */
  item_ids: string[];
  explanation: string;
}

export interface CWAIReviewGlobalProposal {
  item_id: string;
  recommendation: CWAIReviewGlobalRecommendation;
  reason: string;
  suggested_instruction: string;
}

export type CWAIReviewItemRelationStatus =
  | "active"
  | "cancelled";

export type CWAIReviewItemRelationEventType =
  | "confirmed"
  | "cancelled"
  | "reactivated";

export interface CWAIReviewItemRelationEvent {
  id: string;
  relation_version: number;
  event_type: CWAIReviewItemRelationEventType;
  reason: string;
  source_global_message_id: string | null;
  created_at: string | null;
}

/**
 * 人工明确确认过的持久化关系。
 *
 * 该结构与AI临时建议CWAIReviewGlobalRelation严格区分。
 * 浏览器不会收到created_by、cancelled_by或事件actor_id。
 */
export interface CWAIReviewItemRelation {
  id: string;
  courseware_id: string;
  source_session_id: string;

  source_item_id: string;
  target_item_id: string;
  relation_type: CWAIReviewGlobalRelationType;

  status: CWAIReviewItemRelationStatus;
  version: number;
  explanation: string;

  source_global_message_id: string | null;

  confirmed_at: string | null;
  cancelled_at: string | null;
  created_at: string | null;
  updated_at: string | null;

  events: CWAIReviewItemRelationEvent[];
}

export interface CWAIReviewGlobalDiscussion {
  messages: CWAIReviewItemMessage[];

  summary: string;
  relations: CWAIReviewGlobalRelation[];
  proposals: CWAIReviewGlobalProposal[];

  selected_item_ids: string[];
  latest_message_id: string;

  /**
   * 只包含用户独立确认过的持久化关系。
   * relations仍是最近一次AI回复中的临时建议。
   */
  governance_relations: CWAIReviewItemRelation[];
}

// ==================== 作者正式反馈快照 ====================

export interface CWAIReviewFeedback {
  id: string;
  courseware_review_id: string;
  courseware_id: string;

  review_level: number;
  review_round: number;
  decision: string;

  overall_risk: string;
  overall_summary: string;

  strengths_json: string;
  obvious_problems_json: string;

  review_comment_snapshot: string;
  created_at: string | null;
}

export interface CWOwnerReviewRemediationResponse {
  feedbacks: CWAIReviewFeedback[];
  items: CWAIReviewItem[];
}

// ==================== API响应 ====================

export interface CWAIReviewSessionBundle {
  session: CWAIReviewSession | null;
  batches: CWAIReviewBatch[];
}

export interface CWAIReviewRunNextResponse {
  session: CWAIReviewSession;
  batch: CWAIReviewBatch | null;
  result: CWAIReviewBatchResult | null;
  has_more: boolean;
  requires_finalize: boolean;
}

export interface CWAIReviewFinalizeResponse {
  session: CWAIReviewSession;
  report: CWAIReviewFinalReport;
}

export interface CWAIReviewItemListResponse {
  items: CWAIReviewItem[];
  message?: string;
}

export interface PrepareCWAIReviewRequest {
  courseware_id: string;
  review_level: number;
  assistant_id?: string;
}

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
