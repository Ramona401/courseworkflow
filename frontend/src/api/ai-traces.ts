/**
 * AI调用追踪 API 封装（v80新增，v81增加用户/组织维度）
 * - 仪表盘数据：概览数字 + 按场景/模型/用户/组织聚合 + 每日趋势 + 最近错误
 * - 仅 admin 可调用
 */
import client from './client'

// ==================== 类型定义 ====================

/** 按场景聚合统计 */
export interface TraceSceneStats {
  scene_code: string
  scene_name: string
  call_count: number
  success_count: number
  error_count: number
  avg_latency_ms: number
  total_tokens: number
  total_prompt_tokens: number
  total_completion_tokens: number
  estimated_cost_usd: number
}

/** 按模型聚合统计 */
export interface TraceModelStats {
  model_used: string
  call_count: number
  success_count: number
  error_count: number
  avg_latency_ms: number
  total_tokens: number
  estimated_cost_usd: number
}

/** 按用户聚合统计（v81新增） */
export interface TraceUserStats {
  user_id: string
  username: string
  display_name: string
  role: string
  call_count: number
  success_count: number
  error_count: number
  avg_latency_ms: number
  total_tokens: number
  total_prompt_tokens: number
  total_completion_tokens: number
  estimated_cost_usd: number
}

/** 按组织聚合统计（v81新增） */
export interface TraceOrgStats {
  org_id: string
  org_name: string
  org_type: string
  member_count: number
  call_count: number
  success_count: number
  error_count: number
  avg_latency_ms: number
  total_tokens: number
  total_prompt_tokens: number
  total_completion_tokens: number
  estimated_cost_usd: number
}

/** 每日趋势 */
export interface TraceDailyTrend {
  date: string
  call_count: number
  error_count: number
  total_tokens: number
  estimated_cost_usd: number
  avg_latency_ms: number
}

/** 错误记录 */
export interface AICallTrace {
  id: string
  scene_code: string
  model_used: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  latency_ms: number
  status: string
  error_message: string
  pipeline_id?: string
  lesson_plan_id?: string
  user_id?: string
  estimated_cost_usd: number
  output_length: number
  is_stream: boolean
  created_at: string
}

/** 仪表盘总览响应 */
export interface TraceDashboard {
  total_calls: number
  total_tokens: number
  total_cost_usd: number
  avg_latency_ms: number
  error_rate: number
  by_scene: TraceSceneStats[]
  by_model: TraceModelStats[]
  by_user: TraceUserStats[]     // v81新增
  by_org: TraceOrgStats[]       // v81新增
  daily_trend: TraceDailyTrend[]
  recent_errors: AICallTrace[]
}

/** 查询参数 */
export interface TraceQueryParams {
  date_from?: string
  date_to?: string
  scene_code?: string
  model?: string
  status?: string
}

// ==================== API 方法 ====================

/** 获取AI调用仪表盘数据 */
export async function getTraceDashboard(params?: TraceQueryParams): Promise<TraceDashboard> {
  const query = new URLSearchParams()
  if (params?.date_from) query.set('date_from', params.date_from)
  if (params?.date_to) query.set('date_to', params.date_to)
  if (params?.scene_code) query.set('scene_code', params.scene_code)
  if (params?.model) query.set('model', params.model)
  if (params?.status) query.set('status', params.status)
  const qs = query.toString()
  const url = `/admin/ai-traces/dashboard${qs ? '?' + qs : ''}`
  const res = await client.get<{ code: number; data: TraceDashboard }>(url)
  return res.data.data
}
