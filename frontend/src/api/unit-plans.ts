/**
 * unit-plans.ts — 单元方案 API 封装（大单元备课·独立模块）
 *
 * 对应后端：handlers/unit_plan_handler.go · services/unit_plan_service.go
 *   GET    /api/v1/unit-plans            列出可见单元方案 → {unit_plans,total}
 *   POST   /api/v1/unit-plans            开始会话（建草稿+出第一步）→ {plan,opening}
 *   GET    /api/v1/unit-plans/{id}       详情 → {plan,messages}
 *   POST   /api/v1/unit-plans/{id}/chat  逐步对话一轮 → {reply}
 *   POST   /api/v1/unit-plans/{id}/save  定稿保存 → {saved}
 *   DELETE /api/v1/unit-plans/{id}       软删除
 *
 * 大单元挂载（前端入口）新增两条：
 *   GET /api/v1/unit-plans/mountable[?subject=xxx]   可被教案挂载的单元方案（只列 active）→ {unit_plans,total}
 *   PUT /api/v1/lesson-plans/plans/{id}/unit-plan    教案挂载/解除单元方案（体 {unit_plan_id}，空串=解除）
 *
 * 归属选择器复用 ai-assistants 的 getMyPublishGroups()。
 * 拦截器已处理 code!==0 抛错，本文件直接取 data.data。AI 调用接口超时放宽到 180s。
 */
import apiClient from './client'

export type UnitPlanScope = 'group' | 'school' | 'system'
export type UnitPlanStatus = 'draft' | 'active' | 'archived'

export interface UnitPlanMessage {
  role: 'user' | 'assistant'
  content: string
  created_at: string
}

export interface UnitPlanListItem {
  id: string
  scope: UnitPlanScope
  scope_target_id: string
  scope_name?: string
  subject: string
  grade: string
  volume: string
  unit: string
  unit_theme?: string
  title: string
  status: UnitPlanStatus
  creator_name?: string
  updated_at: string
}

export interface UnitPlanDetail {
  id: string
  scope: UnitPlanScope
  scope_target_id: string
  subject: string
  grade: string
  volume: string
  unit: string
  unit_theme: string
  title: string
  content: string
  atlas: string
  source_type: string
  status: UnitPlanStatus
  created_by: string
  created_at: string
  updated_at: string
}

export interface StartUnitPlanRequest {
  scope: UnitPlanScope
  scope_target_id: string
  subject: string
  grade: string
  volume: string
  unit: string
  title?: string
}

export interface SaveUnitPlanRequest {
  title: string
  unit_theme: string
  content: string
  atlas: string
}

export interface UnitPlanListResponse {
  unit_plans: UnitPlanListItem[]
  total: number
}

export interface StartUnitPlanResponse {
  plan: UnitPlanDetail
  opening: string
}

export interface UnitPlanDetailResponse {
  plan: UnitPlanDetail
  messages: UnitPlanMessage[]
}

export async function getUnitPlans(): Promise<UnitPlanListResponse> {
  const { data } = await apiClient.get('/unit-plans')
  return data.data as UnitPlanListResponse
}

export async function getUnitPlan(id: string): Promise<UnitPlanDetailResponse> {
  const { data } = await apiClient.get(`/unit-plans/${id}`)
  return data.data as UnitPlanDetailResponse
}

export async function startUnitPlan(req: StartUnitPlanRequest): Promise<StartUnitPlanResponse> {
  const { data } = await apiClient.post('/unit-plans', req, { timeout: 180000 })
  return data.data as StartUnitPlanResponse
}

export async function chatUnitPlan(id: string, message: string): Promise<string> {
  const { data } = await apiClient.post(`/unit-plans/${id}/chat`, { message }, { timeout: 180000 })
  return (data.data?.reply ?? '') as string
}

export async function saveUnitPlan(id: string, req: SaveUnitPlanRequest): Promise<void> {
  await apiClient.post(`/unit-plans/${id}/save`, req)
}

export async function deleteUnitPlan(id: string): Promise<void> {
  await apiClient.delete(`/unit-plans/${id}`)
}

// ==================== 大单元挂载（前端入口）====================

/**
 * 列出「可被教案挂载」的单元方案（挂载选择器专用，只列 active）
 *
 * 与 getUnitPlans 的区别：只返回 active（不含任何草稿），口径与后端注入层焊死一致——
 * 老师在选择器里看到的每一项都是挂上即生效的（注入层只注入 active）。
 *
 * @param subject 选填。传非空则只列该学科的单元方案（挂载选择器通常按当前教案学科收窄，
 *                减少噪音）；传空/不传则不按学科过滤，列出全部可见 active。
 */
export async function getMountableUnitPlans(subject?: string): Promise<UnitPlanListResponse> {
  const params = subject ? { subject } : undefined
  const { data } = await apiClient.get('/unit-plans/mountable', { params })
  return data.data as UnitPlanListResponse
}

/**
 * 挂载或解除教案关联的单元方案
 *
 * 起步首屏选定、对话中途挂载/更换、解除，三种操作都走这一个接口：
 *   - 挂载/更换：unitPlanId 传目标单元方案 ID
 *   - 解除：    unitPlanId 传空串 ''
 *
 * 关键引擎事实：后端注入层每轮对话重读 lesson_plans.unit_plan_id 决定是否注入
 * 单元方案上下文（仅注入 active），因此本接口更新成功后【下一轮对话】自动生效，
 * 无需刷新页面（机制与课本挂载 updatePlanTextbooks 完全同款）。
 *
 * @returns { message, mounted, unit_plan_id } —— mounted=false 表示已解除
 */
export async function updatePlanUnitPlan(
  planId: string,
  unitPlanId: string,
): Promise<{ message: string; mounted: boolean; unit_plan_id: string }> {
  const { data } = await apiClient.put(`/lesson-plans/plans/${planId}/unit-plan`, {
    unit_plan_id: unitPlanId,
  })
  return data.data as { message: string; mounted: boolean; unit_plan_id: string }
}
