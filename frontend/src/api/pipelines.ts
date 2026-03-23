/**
 * Pipeline管理API封装
 * P4-7: Pipeline列表 + 详情 + 步骤详情 + 操作（创建/启动/取消/删除）
 * P4.5-A增强: PipelineListItem增加eval_avg_score/meta_score/translator_score三个分数字段
 * P4.5-B增强: 新增getEvalRounds API + EvalRoundDetail类型
 * P4.5-C增强: 新增getGeneratedPages/updatePageDecision/finalizePipeline API
 * P4.5-D增强: 新增markPassed API（快捷通过）
 * P4.5-E-2增强: 新增aiFixPage API（AI快修）
 * P4.6-2增强: 新增verifyPipeline API（手动触发验收）
 */
import client from './client'

// ==================== 类型定义 ====================

/** Pipeline配置 */
export interface PipelineConfig {
  eval_rounds: number
  threshold: number
  variance_warn: number
  max_meta_retry: number
  max_tr_loop: number
}

/** Pipeline列表单条（P4.5-A增强：含3个分数字段） */
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
  /** P4.5-A新增：Evaluator均分（从step_data.avg_total提取，可能为null） */
  eval_avg_score: number | null
  /** P4.5-A新增：Meta仲裁分（从step_data.total_final提取，可能为null） */
  meta_score: number | null
  /** P4.5-A新增：Translator最终分（从step_data.final_score提取，可能为null） */
  translator_score: number | null
  /** P4.6新增：审核轮次（1=初审，2=2审） */
  review_round: number
}

/** Pipeline列表响应 */
export interface PipelineListResponse {
  pipelines: PipelineListItem[]
  total: number
}

/** 步骤列表单条 */
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

/** Pipeline详情响应 */
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
  /** P4.6新增：审核轮次（1=初审，2=2审） */
  review_round: number
  steps: StepListItem[]
}

/** 步骤详情响应 */
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

/** 创建Pipeline请求 */
export interface CreatePipelineRequest {
  course_code: string
  auto_mode?: boolean
  config?: Partial<PipelineConfig>
}

// ==================== Generator步骤相关类型 ====================

/** Generator单页记录 */
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

/** Generator步骤step_data */
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

/** Evaluator步骤step_data */
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

/** Meta步骤step_data */
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

/** Translator单轮记录 */
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

/** Translator步骤step_data */
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

/** Scanner步骤step_data */
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

/** DbCheck步骤step_data */
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

// ==================== P4.5-C 审核相关类型 ====================

/** 生成页面完整数据（含HTML内容，用于审核预览） */
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
  /** P4.5-E新增：Translator给出的修改理由和指令（审核页面展示用） */
  change_reason: string
  created_at: string | null
  updated_at: string | null
}

/** 更新页面决策请求 */
export interface UpdatePageDecisionRequest {
  decision: 'approve' | 'reject' | 'edit'
  final_html?: string
}

// ==================== P4.6-2 验收相关类型 ====================

/** Verify步骤step_data（P4.6-2新增） */
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

// ==================== API方法 ====================

/** 获取Pipeline列表 */
export async function getPipelines() {
  const res = await client.get('/pipelines')
  return (res.data as any).data as PipelineListResponse
}

/** 获取Pipeline详情 */
export async function getPipelineDetail(id: string) {
  const res = await client.get('/pipelines/' + id)
  return (res.data as any).data as PipelineDetailResponse
}

/** 创建Pipeline */
export async function createPipeline(req: CreatePipelineRequest) {
  const res = await client.post('/pipelines', req)
  return (res.data as any).data
}

/** 启动Pipeline（同步执行，可能耗时10-30分钟） */
export async function startPipeline(id: string) {
  const res = await client.post('/pipelines/' + id + '/start', null, {
    timeout: 3600000, // 60分钟超时（全链路AI调用耗时长）
  })
  return (res.data as any).data
}

/** 取消Pipeline */
export async function cancelPipeline(id: string) {
  const res = await client.post('/pipelines/' + id + '/cancel')
  return (res.data as any).data
}

/** 删除Pipeline */
export async function deletePipeline(id: string) {
  const res = await client.delete('/pipelines/' + id)
  return (res.data as any).data
}

/** 获取步骤详情 */
export async function getStepDetail(pipelineId: string, stepName: string) {
  const res = await client.get('/pipelines/' + pipelineId + '/steps/' + stepName)
  return (res.data as any).data as StepDetailResponse
}

// ==================== Eval Rounds API（P4.5-B新增）====================

/** 评估轮次详情 */
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

/** 获取评估轮次详情列表 */
export async function getEvalRounds(pipelineId: string) {
  const res = await client.get('/pipelines/' + pipelineId + '/eval-rounds')
  const data = (res.data as any).data
  return data.rounds as EvalRoundDetail[]
}

// ==================== P4.5-C 审核相关API ====================

/** 获取生成页面列表（含完整HTML） */
export async function getGeneratedPages(pipelineId: string) {
  const res = await client.get('/pipelines/' + pipelineId + '/pages')
  const data = (res.data as any).data
  return data.pages as GeneratedPageFull[]
}

/** 更新单页审核决策 */
export async function updatePageDecision(pipelineId: string, pageNumber: number, req: UpdatePageDecisionRequest) {
  const res = await client.put('/pipelines/' + pipelineId + '/pages/' + pageNumber + '/decision', req)
  return (res.data as any).data
}

/** 定稿归档Pipeline */
export async function finalizePipeline(pipelineId: string) {
  const res = await client.post('/pipelines/' + pipelineId + '/finalize')
  return (res.data as any).data
}

// ==================== P4.5-D 快捷通过API ====================

/** 快捷通过Pipeline（评估达标直接标记为finalized） */
export async function markPassed(pipelineId: string) {
  const res = await client.post('/pipelines/' + pipelineId + '/mark-passed')
  return (res.data as any).data
}

// ==================== P4.5-E-2 AI快修API ====================

/** AI快修请求参数 */
export interface AIFixPageRequest {
  fix_instruction: string
}

/** AI快修响应 */
export interface AIFixPageResponse {
  message: string
  page_number: number
  new_html: string
  html_length: number
}

/** AI快修页面（审核员输入修改指令，AI基于当前HTML修复） */
export async function aiFixPage(pipelineId: string, pageNumber: number, req: AIFixPageRequest) {
  const res = await client.post('/pipelines/' + pipelineId + '/pages/' + pageNumber + '/ai-fix', req, {
    timeout: 600000, // 10分钟超时（AI修复可能较慢）
  })
  return (res.data as any).data as AIFixPageResponse
}

// ==================== P4.6-2 验收API ====================

/** 手动触发验收评估（finalized状态的Pipeline） */
// P4.6-2新增：收集最终HTML → 索引生成器压缩 → Evaluator评估 → 判定通过/失败
// 验收过程包含2次AI调用（索引生成+评估），预计耗时5-15分钟
export async function verifyPipeline(pipelineId: string) {
  const res = await client.post('/pipelines/' + pipelineId + '/verify', null, {
    timeout: 1800000, // 30分钟超时（索引生成+评估两次AI调用）
  })
  return (res.data as any).data as PipelineDetailResponse
}
