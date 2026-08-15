/**
 * coursewares.ai-review.types.ts
 *
 * 课件AI审核、自审、整改项、全局讨论和R-07结构化影响方案的浏览器契约类型。
 *
 * R-01.1把类型从接近900行的API实现文件拆出，避免继续在单文件堆叠职责。
 * 整改项同时保留旧兼容字段和后端教师字段；共享教师改进卡只消费教师字段。
 */

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
  | "global_discussion_manual"
  | "goal_drift_manual";

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
   * global_discussion_manual：由用户在全局讨论中人工新增；
   * goal_drift_manual：完善当前修改要求时明确拆出的独立新问题。
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

  teacher_title: string;
  what_happened: string;
  teaching_impact: string;
  improvement_goal: string;
  acceptance_checks: string[];
  teacher_context: string;
  manual_check_required: boolean;

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

// ==================== R-07结构化影响方案 ====================

export type CWAIReviewImpactPlanStatus =
  | "draft"
  | "applied";

export type CWAIReviewImpactOperationType =
  | "create_group"
  | "move_group_member"
  | "merge_groups"
  | "split_group"
  | "create_relation"
  | "cancel_relation"
  | "create_item"
  | "dismiss_item"
  | "update_candidate_suggestion";

export interface CWAIReviewImpactOperation {
  operation_id: string;
  operation_type: CWAIReviewImpactOperationType;
  action_label: string;
  summary: string;

  /**
   * payload只用于教师Preview。
   * 最终Apply请求绝不能把它回传给后端作为事实源。
   */
  payload: Record<string, unknown>;
}

export interface CWAIReviewImpactPlanEvent {
  plan_version: number;
  event_type: "draft_created" | "applied";
  selected_operation_ids: string[];
  created_at: string | null;
}

export interface CWAIReviewImpactPlan {
  id: string;
  courseware_id: string;
  source_session_id: string;

  status: CWAIReviewImpactPlanStatus;
  version: number;
  operations_schema_version: number;

  operations: CWAIReviewImpactOperation[];
  applied_operation_ids: string[];

  created_at: string | null;
  applied_at: string | null;
  updated_at: string | null;

  events: CWAIReviewImpactPlanEvent[];
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
