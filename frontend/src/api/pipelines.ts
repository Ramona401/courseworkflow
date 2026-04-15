/**
 * Pipeline管理API封装
 * P7新增：二级审批流程 submitFinalize/confirmFinalize/rejectFinalize
 * Phase8修复A-02：删除 assignPipeline() 函数
 * Phase8修复P-02：PipelineDetailResponse 新增 reject_reason 字段
 * v34修复A-01：消除全部 (res.data as any).data 类型绕过
 * v36新增：restartFromStep() 断点续跑API
 * v90-1修复：aiFixPageStream token获取从错误的'auth-storage'改为正确的'token'键
 */
import client from './client'
import type { ApiResponse } from './client'
import type { AxiosResponse } from 'axios'

// ==================== 辅助函数 ====================

/**
 * extractData 从 Axios 响应中安全提取业务数据
 * v34修复A-01：替代全部 (res.data as any).data 写法
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

// ==================== 后端包装响应类型 ====================

interface PagesWrapper {
  pages: GeneratedPageFull[]
}

interface EvalRoundsWrapper {
  rounds: EvalRoundDetail[]
}

interface OperatorsWrapper {
  operators: OperatorInfo[]
}

// ==================== API方法 ====================

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

// ==================== 断点续跑API（v36新增）====================

/**
 * restartFromStep 从指定步骤重新执行Pipeline
 * v36新增：对应后端 POST /api/v1/pipelines/{id}/restart-from
 * 调用后Pipeline立即进入running状态，前端通过SSE或轮询获取进度
 *
 * @param pipelineId - Pipeline ID
 * @param stepName   - 要从哪个步骤开始重跑（dbCheck/scanner/evaluator/meta/translator/generator）
 * @returns 更新后的Pipeline详情（status已变为running）
 */
export async function restartFromStep(
  pipelineId: string,
  stepName: string
): Promise<PipelineDetailResponse> {
  const res = await client.post<ApiResponse<PipelineDetailResponse>>(
    '/pipelines/' + pipelineId + '/restart-from',
    { step_name: stepName } satisfies RestartFromStepRequest,
    { timeout: 3600000 }
  )
  return extractData<PipelineDetailResponse>(res)
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
 */
export async function finalizePipeline(pipelineId: string): Promise<void> {
  await client.post<ApiResponse<void>>('/pipelines/' + pipelineId + '/finalize')
}

// ==================== P7新增：二级审批API ====================

export async function submitFinalize(pipelineId: string): Promise<void> {
  await client.post<ApiResponse<void>>('/pipelines/' + pipelineId + '/submit-finalize')
}

export async function confirmFinalize(pipelineId: string): Promise<void> {
  await client.post<ApiResponse<void>>('/pipelines/' + pipelineId + '/confirm-finalize')
}

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
  /** v68新增：参考页码数组（可选，审核员选择的其他页面作为参考） */
  reference_pages?: number[]
  fix_instruction: string
}

export interface AIFixPageResponse {
  message: string
  page_number: number
  new_html: string
  html_length: number
  /** v68新增：AI修改说明 */
  fix_summary: string
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

// ==================== 回滚API（v68新增）====================

export interface RollbackPageResponse {
  message: string
  page_number: number
  restored_html: string
  html_length: number
  remaining_history: number
}

export async function rollbackPageHTML(
  pipelineId: string,
  pageNumber: number
): Promise<RollbackPageResponse> {
  const res = await client.post<ApiResponse<RollbackPageResponse>>(
    '/pipelines/' + pipelineId + '/pages/' + pageNumber + '/rollback',
    {},
    { timeout: 30000 }
  )
  return extractData<RollbackPageResponse>(res)
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

// ==================== v37新增：批量断点续跑API ====================

/**
 * BatchRestartResult 批量断点续跑结果
 */
export interface BatchRestartResult {
  total_requested: number
  success_count: number
  skipped_ids: string[]
  skipped_reasons: string[]
  failed_ids: string[]
  failed_reasons: string[]
}

/**
 * batchRestartFromStep 批量从指定步骤重新执行多个Pipeline
 * v37新增：对应后端 POST /api/v1/pipelines/batch-restart
 * 权限：仅 admin / senior_operator 可调用
 *
 * @param pipelineIds - 要重跑的Pipeline ID列表（上限50个）
 * @param stepName    - 统一的起跑步骤（如 "generator"）
 * @returns 批量重跑结果（成功数/跳过数/失败数）
 */
export async function batchRestartFromStep(
  pipelineIds: string[],
  stepName: string
): Promise<BatchRestartResult> {
  const res = await client.post<ApiResponse<BatchRestartResult>>(
    '/pipelines/batch-restart',
    { pipeline_ids: pipelineIds, step_name: stepName },
    { timeout: 3600000 }
  )
  return extractData<BatchRestartResult>(res)
}

// ==================== v38新增：Translator FAIL后强制推进API ====================

/**
 * forceProceed 确认使用当前Translator方案，跳过重跑直接启动Generator
 * v38新增：对应后端 POST /api/v1/pipelines/{id}/force-proceed
 *
 * 使用场景：Translator-Reviewer循环FAIL（如评分差0.1不达标），
 * 但方案质量已足够好，操作员确认后直接启动Generator。
 *
 * @param pipelineId - Pipeline ID
 * @returns 更新后的Pipeline详情（status变为running）
 */
export async function forceProceed(
  pipelineId: string
): Promise<PipelineDetailResponse> {
  const res = await client.post<ApiResponse<PipelineDetailResponse>>(
    '/pipelines/' + pipelineId + '/force-proceed',
    null,
    { timeout: 3600000 }
  )
  return extractData<PipelineDetailResponse>(res)
}

// ==================== 发布至课程平台 ====================

/**
 * publishPipeline 骨干教师确认已将课件发布至课程平台
 * 触发条件：Pipeline 状态为 verified
 * 结果：状态变更为 published（单向不可逆）
 */
export async function publishPipeline(id: string): Promise<void> {
  await client.post(`/pipelines/${id}/publish`)
}

// ==================== 历史轮次查询 ====================

export interface HistoryStepItem {
  step_name: string
  step_order: number
  status: string
  status_name: string
  step_name_cn: string
  duration_ms: number
  tokens_used: number
  has_data: boolean
  error_message: string
  step_data?: Record<string, unknown>
}

export interface HistoryEvalRound {
  id: string
  round_number: number
  status: string
  score_total: number | null
  score_e1: number | null
  score_e2: number | null
  score_e3: number | null
  score_e4: number | null
  output: string
  tokens_used: number
}

export interface PipelineHistoryResponse {
  round: number
  available_rounds: number[]
  steps: HistoryStepItem[]
  eval_rounds: HistoryEvalRound[]
}

/** 获取可用历史轮次列表 */
export async function getPipelineAvailableRounds(id: string): Promise<number[]> {
  const res = await client.get<{ code: number; data: { available_rounds: number[] } }>(`/pipelines/${id}/history`)
  return res.data.data?.available_rounds || []
}

/** 获取指定轮次的历史数据 */
export async function getPipelineHistory(id: string, round: number): Promise<PipelineHistoryResponse> {
  const res = await client.get<{ code: number; data: PipelineHistoryResponse }>(`/pipelines/${id}/history?round=${round}`)
  return res.data.data!
}

// ==================== 审核页HTML懒加载API（v69新增，编号8方案2）====================

/**
 * getPagesMeta 获取Pipeline所有页面的轻量元数据（不含HTML内容）
 * v69新增：审核页首次加载只获取元数据，大幅减少传输数据量
 * 页面的 original_html / generated_html / final_html 均为空字符串
 */
export async function getPagesMeta(pipelineId: string): Promise<GeneratedPageFull[]> {
  const res = await client.get<ApiResponse<PagesWrapper>>('/pipelines/' + pipelineId + '/pages-meta')
  const data = extractData<PagesWrapper>(res)
  return data.pages
}

/**
 * getSinglePageHTML 按需加载单页完整HTML数据
 * v69新增：审核页选中页面时才请求完整HTML（含original/generated/final_html）
 */
export async function getSinglePageHTML(pipelineId: string, pageNumber: number): Promise<GeneratedPageFull> {
  const res = await client.get<ApiResponse<GeneratedPageFull>>('/pipelines/' + pipelineId + '/pages/' + pageNumber + '/html')
  return extractData<GeneratedPageFull>(res)
}

// ==================== AI快修流式API（v69新增，编号5）====================

/**
 * aiFixPageStream AI快修流式调用
 * v69新增：通过POST请求建立SSE连接，逐token接收AI输出
 * 注意：这不是标准EventSource（EventSource只支持GET），而是用fetch+ReadableStream实现
 *
 * v90-1修复：token获取方式改为直接从localStorage.getItem('token')读取，
 * 与client.ts中axios拦截器保持一致。原来错误地从'auth-storage'（zustand格式）读取，
 * 但本项目使用React Context + 手动localStorage存储，导致token始终为空。
 *
 * @param pipelineId - Pipeline ID
 * @param pageNumber - 页码
 * @param req - 请求参数（同aiFixPage）
 * @param onChunk - 收到AI输出token时的回调
 * @param onDone - AI输出完成时的回调（含最终结果）
 * @param onError - 出错时的回调
 */
export async function aiFixPageStream(
  pipelineId: string,
  pageNumber: number,
  req: AIFixPageRequest,
  onChunk: (content: string) => void,
  onDone: (result: { new_html: string; fix_summary: string; html_length: number }) => void,
  onError: (message: string) => void,
): Promise<void> {
  // v90-1修复：直接从localStorage读取token，与client.ts拦截器保持一致
  const token = localStorage.getItem('token') || ''

  if (!token) {
    onError('未登录或认证已过期，请重新登录')
    return
  }

  try {
    const resp = await fetch('/api/v1/pipelines/' + pipelineId + '/pages/' + pageNumber + '/ai-fix-stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + token,
      },
      body: JSON.stringify(req),
    })

    if (!resp.ok) {
      const errText = await resp.text()
      try {
        const errJson = JSON.parse(errText)
        onError(errJson.message || errJson.error || '请求失败: HTTP ' + resp.status)
      } catch {
        onError('请求失败: HTTP ' + resp.status)
      }
      return
    }

    // 读取SSE流
    const reader = resp.body?.getReader()
    if (!reader) {
      onError('浏览器不支持流式读取')
      return
    }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      // 按SSE协议逐行解析（双换行分隔消息）
      const messages = buffer.split('\n\n')
      // 最后一个可能不完整，留在buffer中
      buffer = messages.pop() || ''

      for (const msg of messages) {
        if (!msg.trim()) continue

        let eventType = 'message'
        let eventData = ''

        for (const line of msg.split('\n')) {
          if (line.startsWith('event: ')) {
            eventType = line.slice(7).trim()
          } else if (line.startsWith('data: ')) {
            eventData = line.slice(6)
          }
        }

        if (!eventData) continue

        try {
          const parsed = JSON.parse(eventData)

          switch (eventType) {
            case 'chunk':
              if (parsed.content) onChunk(parsed.content)
              break
            case 'done':
              onDone({
                new_html: parsed.new_html || '',
                fix_summary: parsed.fix_summary || '',
                html_length: parsed.html_length || 0,
              })
              return // 流式结束
            case 'error':
              onError(parsed.message || 'AI快修失败')
              return
            case 'connected':
              // 连接建立，不做处理
              break
          }
        } catch {
          // JSON解析失败跳过
        }
      }
    }
  } catch (e: unknown) {
    onError('网络错误: ' + (e instanceof Error ? e.message : '连接失败'))
  }
}
