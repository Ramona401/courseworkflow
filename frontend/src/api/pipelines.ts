/**
 * Pipeline管理API封装
 * P7新增：二级审批流程 submitFinalize/confirmFinalize/rejectFinalize
 *         新增 pending_finalize 状态
 * Phase8修复A-02：删除 assignPipeline() 函数
 *   原版调用 POST /pipelines/{id}/assign，但 routes.go 中该路由不存在（只有 /batch-assign）
 *   修复：移除该函数，所有分配操作统一走 batchAssignPipelines（单条分配传一个ID即可）
 * Phase8修复P-02：PipelineDetailResponse 新增 reject_reason 字段
 * v33修复A-01：消除全部 (res.data as any).data 类型绕过
 *   使用 extractData<T>() 辅助函数，统一从 AxiosResponse<ApiResponse<T>> 中安全提取数据
 *   所有API函数均获得完整的泛型类型推导，不再依赖 as any 强制转换
 */
import client from './client'
import type { ApiResponse } from './client'
import type { AxiosResponse } from 'axios'

// ==================== 辅助函数（v33修复A-01） ====================

/**
 * extractData 从 Axios 响应中安全提取业务数据
 * v33修复A-01：替代全部 (res.data as any).data 写法
 * 统一类型推导链：AxiosResponse<ApiResponse<T>> → ApiResponse<T>.data → T
 * 注意：响应拦截器已处理 code !== 0 的情况（reject），这里只处理成功响应
 */
function extractData<T>(res: AxiosResponse<ApiResponse<T>>): T {
  return res.data.data as T
}

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
 * 审核员在审核页面可看到超级审核员填写的退回原因
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
  /** 最近一次退回重审的原因（空字符串表示未被退回或未填写原因）Phase8新增 */
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
  step_data: any
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

// ==================== 后端包装响应类型（用于嵌套data结构） ====================
// 部分接口返回 { pages: [...] } 或 { rounds: [...] } 等嵌套结构
// 定义中间类型供 extractData 正确推导

/** 生成页面列表响应包装 */
interface PagesWrapper {
  pages: GeneratedPageFull[]
}

/** 评估轮次列表响应包装 */
interface EvalRoundsWrapper {
  rounds: EvalRoundDetail[]
}

/** 操作员列表响应包装 */
interface OperatorsWrapper {
  operators: OperatorInfo[]
}

// ==================== API方法（v33修复A-01：全部使用 extractData<T> 替代 as any） ====================

export async function getPipelines(): Promise<PipelineListResponse> {
  const res = await client.get<ApiResponse<PipelineListResponse>>('/pipelines')
  return extractData<PipelineListResponse>(res)
}

export async function getPipelineDetail(id: string): Promise<PipelineDetailResponse> {
  const res = await client.get<ApiResponse<PipelineDetailResponse>>('/pipelines/' + id)
  return extractData<PipelineDetailResponse>(res)
}

export async function createPipeline(req: CreatePipelineRequest): Promise<PipelineDetailResponse> {
  const res = await client.post<ApiResponse<PipelineDetailResponse>>('/pipelines', req)
  return extractData<PipelineDetailResponse>(res)
}

export async function startPipeline(id: string): Promise<PipelineDetailResponse> {
  const res = await client.post<ApiResponse<PipelineDetailResponse>>('/pipelines/' + id + '/start', null, {
    timeout: 3600000,
  })
  return extractData<PipelineDetailResponse>(res)
}

export async function cancelPipeline(id: string): Promise<PipelineDetailResponse> {
  const res = await client.post<ApiResponse<PipelineDetailResponse>>('/pipelines/' + id + '/cancel')
  return extractData<PipelineDetailResponse>(res)
}

export async function deletePipeline(id: string): Promise<void> {
  await client.delete<ApiResponse<void>>('/pipelines/' + id)
}

export async function getStepDetail(pipelineId: string, stepName: string): Promise<StepDetailResponse> {
  const res = await client.get<ApiResponse<StepDetailResponse>>('/pipelines/' + pipelineId + '/steps/' + stepName)
  return extractData<StepDetailResponse>(res)
}

// ==================== Eval Rounds API ====================

export interface EvalRoundDetail {
  id: string
  round_number: number
  status: string
  output: string
  score_total: number | null
  score_e1: number | null
  score_e2: number | null
  score_e3: number | null
  score_e4: number | null
  hard_constraint: string
  grade: string
  model_used: string
  tokens_used: number
}

export async function getEvalRounds(pipelineId: string): Promise<EvalRoundDetail[]> {
  const res = await client.get<ApiResponse<EvalRoundsWrapper>>('/pipelines/' + pipelineId + '/eval-rounds')
  const data = extractData<EvalRoundsWrapper>(res)
  return data.rounds
}

// ==================== 审核相关API ====================

export async function getGeneratedPages(pipelineId: string): Promise<GeneratedPageFull[]> {
  const res = await client.get<ApiResponse<PagesWrapper>>('/pipelines/' + pipelineId + '/pages')
  const data = extractData<PagesWrapper>(res)
  return data.pages
}

export async function updatePageDecision(
  pipelineId: string,
  pageNumber: number,
  req: UpdatePageDecisionRequest
): Promise<void> {
  await client.put<ApiResponse<void>>(
    '/pipelines/' + pipelineId + '/pages/' + pageNumber + '/decision',
    req
  )
}

/**
 * 直接定稿归档（仅admin可用，跳过二级审批）
 * P7更新：普通操作员请使用 submitFinalize
 */
export async function finalizePipeline(pipelineId: string): Promise<void> {
  await client.post<ApiResponse<void>>('/pipelines/' + pipelineId + '/finalize')
}

// ==================== P7新增：二级审批API ====================

/**
 * 提交定稿申请（审核员→待超级审核员确认）
 * P7新增：审核员完成逐页决策后调用，状态变为 pending_finalize
 * 权限：operator / senior_operator / admin
 */
export async function submitFinalize(pipelineId: string): Promise<void> {
  await client.post<ApiResponse<void>>('/pipelines/' + pipelineId + '/submit-finalize')
}

/**
 * 确认定稿（超级审核员确认，pending_finalize→finalized）
 * P7新增：senior_operator / admin 在审核中心确认定稿
 */
export async function confirmFinalize(pipelineId: string): Promise<void> {
  await client.post<ApiResponse<void>>('/pipelines/' + pipelineId + '/confirm-finalize')
}

/**
 * 退回重审（超级审核员退回，pending_finalize→review_queue）
 * P7新增：senior_operator / admin 退回给原审核员重新审核
 * Phase8修复P-02：退回原因现在会持久化到数据库，审核员可在审核页面看到
 */
export async function rejectFinalize(pipelineId: string, reason?: string): Promise<void> {
  await client.post<ApiResponse<void>>(
    '/pipelines/' + pipelineId + '/reject-finalize',
    { reason: reason || '' }
  )
}

// ==================== 快捷通过API ====================

export async function markPassed(pipelineId: string): Promise<void> {
  await client.post<ApiResponse<void>>('/pipelines/' + pipelineId + '/mark-passed')
}

// ==================== AI快修API ====================

export interface AIFixPageRequest {
  fix_instruction: string
}

export interface AIFixPageResponse {
  message: string
  page_number: number
  new_html: string
  html_length: number
}

export async function aiFixPage(
  pipelineId: string,
  pageNumber: number,
  req: AIFixPageRequest
): Promise<AIFixPageResponse> {
  const res = await client.post<ApiResponse<AIFixPageResponse>>(
    '/pipelines/' + pipelineId + '/pages/' + pageNumber + '/ai-fix',
    req,
    { timeout: 600000 }
  )
  return extractData<AIFixPageResponse>(res)
}

// ==================== 验收API ====================

export async function verifyPipeline(pipelineId: string): Promise<PipelineDetailResponse> {
  const res = await client.post<ApiResponse<PipelineDetailResponse>>('/pipelines/' + pipelineId + '/verify', null, {
    timeout: 1800000,
  })
  return extractData<PipelineDetailResponse>(res)
}

// ==================== 批量操作API ====================

export interface BatchCreateResult {
  total_requested: number
  created_ids: string[]
  skipped_codes: string[]
  skipped_reasons: string[]
  failed_codes: string[]
  failed_reasons: string[]
}

export interface BatchStartResult {
  total_requested: number
  started_ids: string[]
  skipped_ids: string[]
  skipped_reasons: string[]
  failed_ids: string[]
  failed_reasons: string[]
}

export async function batchCreatePipelines(courseCodes: string[]): Promise<BatchCreateResult> {
  const res = await client.post<ApiResponse<BatchCreateResult>>('/pipelines/batch-create', {
    course_codes: courseCodes,
  })
  return extractData<BatchCreateResult>(res)
}

export async function batchStartPipelines(pipelineIds: string[]): Promise<BatchStartResult> {
  const res = await client.post<ApiResponse<BatchStartResult>>('/pipelines/batch-start', {
    pipeline_ids: pipelineIds,
  })
  return extractData<BatchStartResult>(res)
}

// ==================== 审核分配API ====================

export interface OperatorInfo {
  id: string
  username: string
  display_name: string
  role: string
}

export interface BatchAssignResult {
  total_requested: number
  success_count: number
  assigned_to: string
  assigned_name: string
  failed_ids: string[]
}

export async function getOperators(): Promise<OperatorInfo[]> {
  const res = await client.get<ApiResponse<OperatorsWrapper>>('/pipelines/operators')
  const data = extractData<OperatorsWrapper>(res)
  return data.operators
}

/**
 * 批量分配Pipeline给指定审核员
 * Phase8修复A-02说明：
 *   原版有一个 assignPipeline(pipelineId, assignedTo) 函数，
 *   调用 POST /pipelines/{id}/assign，但该路由在后端 routes.go 中不存在（只有 /batch-assign）。
 *   修复：删除 assignPipeline 函数，单条分配时传入长度为1的数组调用本函数即可。
 *   用法：单条分配 → batchAssignPipelines([pipelineId], assignedTo)
 *         批量分配 → batchAssignPipelines(pipelineIds, assignedTo)
 */
export async function batchAssignPipelines(
  pipelineIds: string[],
  assignedTo: string
): Promise<BatchAssignResult> {
  const res = await client.post<ApiResponse<BatchAssignResult>>('/pipelines/batch-assign', {
    pipeline_ids: pipelineIds,
    assigned_to: assignedTo,
  })
  return extractData<BatchAssignResult>(res)
}
