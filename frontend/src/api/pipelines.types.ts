/**
 * pipelines.types.ts — Pipeline 系统类型定义
 *
 * 从 pipelines.ts 拆分
 * 纯类型定义文件，不包含任何运行时代码、import实例或函数
 */

// ==================== 类型定义 ====================

export interface PipelineConfig {
  eval_rounds: number
  threshold: number
  variance_warn: number
  max_meta_retry: number
  max_tr_loop: number
}

export interface PipelineListItem {
  id: string
  course_code: string
  course_name: string
  external_module_id: number | null
  current_step: string
  current_step_name: string
  status: string
  status_name: string
  auto_mode: boolean
  steps_completed: number
  steps_total: number
  error_message: string
  started_by: string | null
  started_at: string | null
  completed_at: string | null
  created_at: string | null
  eval_avg_score: number | null
  meta_score: number | null
  translator_score: number | null
  review_round: number
  assigned_to: string | null
  assigned_name: string
}

export interface PipelineListResponse {
  pipelines: PipelineListItem[]
  total: number
}

export interface StepListItem {
  id: string
  step_name: string
  step_name_cn: string
  step_order: number
  status: string
  status_name: string
  started_at: string | null
  completed_at: string | null
  duration_ms: number
  attempts: number
  model_used: string
  tokens_used: number
  error_message: string
  has_data: boolean
}

/**
 * PipelineDetailResponse Pipeline详情响应类型
 * Phase8修复P-02：新增 reject_reason 字段
 */
export interface PipelineDetailResponse {
  id: string
  course_code: string
  course_name: string
  external_module_id: number | null
  current_step: string
  current_step_name: string
  status: string
  status_name: string
  auto_mode: boolean
  config: PipelineConfig | null
  error_message: string
  started_by: string | null
  started_at: string | null
  completed_at: string | null
  created_at: string | null
  updated_at: string | null
  review_round: number
  assigned_to: string | null
  assigned_name: string
  /** 最近一次退回重审的原因 Phase8新增 */
  reject_reason: string
  steps: StepListItem[]
}

export interface StepDetailResponse {
  id: string
  pipeline_id: string
  step_name: string
  step_name_cn: string
  step_order: number
  status: string
  status_name: string
  started_at: string | null
  completed_at: string | null
  duration_ms: number
  attempts: number
  step_data: Record<string, unknown>
  error_message: string
  model_used: string
  tokens_used: number
  created_at: string | null
  updated_at: string | null
}

export interface CreatePipelineRequest {
  course_code: string
  auto_mode?: boolean
  config?: Partial<PipelineConfig>
}

// ==================== Generator步骤相关类型 ====================

export interface GeneratorPageRecord {
  page_number: number
  page_title: string
  operation: string
  lesson_id: number
  has_orig_html: boolean
  orig_html_len: number
  gen_html_len: number
  merge_sources: number[] | null
  tokens_used: number
  latency_ms: number
  status: string
  error: string
}

export interface GeneratorStepData {
  total_pages: number
  kept_pages: number
  modified_pages: number
  created_pages: number
  merged_pages: number
  deleted_pages: number
  failed_pages: number
  pages: GeneratorPageRecord[]
  total_tokens: number
  total_latency_ms: number
  model_used: string
}

// ==================== Evaluator步骤相关类型 ====================

export interface EvaluatorStepData {
  total_rounds: number
  done_rounds: number
  failed_rounds: number
  avg_total: number
  avg_e1: number
  avg_e2: number
  avg_e3: number
  avg_e4: number
  variance: number
  variance_warn: boolean
  round_scores: number[]
  total_tokens: number
  total_latency_ms: number
  model_used: string
}

// ==================== Meta步骤相关类型 ====================

export interface MetaStepData {
  total_final: number
  e1_final: number
  e2_final: number
  e3_final: number
  e4_final: number
  hard_constraint: string
  grade: string
  passed: boolean
  e1_rounds: number[]
  e2_rounds: number[]
  e3_rounds: number[]
  e4_rounds: number[]
  attempt: number
  total_retries: number
  raw_output: string
  model_used: string
  tokens_used: number
  latency_ms: number
}

// ==================== Translator步骤相关类型 ====================

export interface TranslatorRoundRecord {
  round: number
  trans_output: string
  trans_tokens: number
  trans_latency_ms: number
  trans_error: string
  review_output: string
  review_tokens: number
  review_latency_ms: number
  review_error: string
  score: number
  quality_gate: string
  hard_check: string
  grade: string
  e1: number
  e2: number
  e3: number
  e4: number
  passed: boolean
}

export interface TranslatorStepData {
  max_loops: number
  threshold: number
  passed: boolean
  final_score: number
  final_quality_gate: string
  final_grade: string
  final_round: number
  final_trans_output: string
  final_review_output: string
  rounds: TranslatorRoundRecord[]
  total_tokens: number
  total_latency_ms: number
  model_used: string
}

// ==================== Scanner步骤相关类型 ====================

export interface ScannerStepData {
  raw_output: string
  parsed: {
    target: string
    ability_targets: string[]
    grade_standard: string
    course_standard: string
  } | null
  is_valid: boolean
  model_used: string
  tokens_used: number
}

// ==================== DbCheck步骤相关类型 ====================

export interface DbCheckStepData {
  course_code: string
  course_id: string
  module_id: number
  has_index: boolean
  index_hash: string
  page_count: number
  total_length: number
  is_valid: boolean
  error_detail: string
}

// ==================== 审核相关类型 ====================

export interface GeneratedPageFull {
  id: string
  pipeline_id: string
  page_number: number
  page_title: string
  operation: string
  original_html: string
  generated_html: string
  final_html: string
  decision: string
  lesson_id: number | null
  merge_sources: string
  change_reason: string
  created_at: string | null
  updated_at: string | null
  /** FE-RM-01修复：后端可能返回的HTML历史版本数（回滚功能用） */
  html_history_count?: number
}

export interface UpdatePageDecisionRequest {
  decision: 'approve' | 'reject' | 'edit'
  final_html?: string
}

// ==================== 验收相关类型 ====================

export interface VerifyStepData {
  generated_index: string
  eval_score: number
  eval_output: string
  eval_e1: number
  eval_e2: number
  eval_e3: number
  eval_e4: number
  passed: boolean
  review_round: number
  model_used: string
  tokens_used: number
  latency_ms: number
}

// ==================== 断点续跑相关类型（v36新增）====================

/**
 * RestartFromStepRequest 断点续跑请求
 * step_name: 要从哪个步骤开始重跑
 * 支持: dbCheck / scanner / evaluator / meta / translator / generator
 */
export interface RestartFromStepRequest {
  step_name: string
}
