/**
 * unit-plans.ts — 单元方案 API 封装（大单元备课·独立模块）
 *
 * 对应后端：handlers/unit_plan_handler.go · services/unit_plan_service.go
 *   GET    /api/v1/unit-plans            列出可见单元方案 → {unit_plans,total}
 *   POST   /api/v1/unit-plans            开始会话（建草稿+出第一步）→ {plan,opening}
 *   GET    /api/v1/unit-plans/{id}       详情 → {plan,messages,can_edit}
 *   POST   /api/v1/unit-plans/{id}/chat  逐步对话一轮 → {reply}
 *   POST   /api/v1/unit-plans/{id}/save  定稿保存或再次保存优化结果 → {saved}
 *   DELETE /api/v1/unit-plans/{id}       软删除
 *
 * 大单元挂载：
 *   GET /api/v1/unit-plans/mountable[?subject=xxx]
 *   PUT /api/v1/lesson-plans/plans/{id}/unit-plan
 *
 * 续作权限：
 *   详情接口返回 can_edit，由后端根据当前登录用户是否为方案创建者确定。
 *   前端只据此决定是否展示AI续作界面；真正的聊天和保存权限仍由后端服务层校验。
 *
 * 课程大纲教材版本三态：
 *   - 不传 / null     → 不关联课程大纲
 *   - ''              → 通用 / 不限版本
 *   - '人教版'等具名 → 精确匹配该版本
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

  /**
   * 本会话选定的课程大纲教材版本：
   *   null      = 未关联课程大纲
   *   ''        = 通用 / 不限版本
   *   '人教版'  = 具名版本精确匹配
   */
  course_outline_publisher: string | null
}

export interface StartUnitPlanRequest {
  scope: UnitPlanScope
  scope_target_id: string
  subject: string
  grade: string
  volume: string
  unit: string
  title?: string
  course_outline_publisher?: string | null
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

  /**
   * 当前登录用户能否继续聊天和再次保存该方案。
   * 当前后端口径为：只有方案创建者返回 true。
   */
  can_edit: boolean
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
  const { data } = await apiClient.post(
    `/unit-plans/${id}/chat`,
    { message },
    { timeout: 180000 },
  )
  return (data.data?.reply ?? '') as string
}

export async function saveUnitPlan(id: string, req: SaveUnitPlanRequest): Promise<void> {
  await apiClient.post(`/unit-plans/${id}/save`, req)
}

export async function deleteUnitPlan(id: string): Promise<void> {
  await apiClient.delete(`/unit-plans/${id}`)
}

// ==================== 大单元挂载 ====================

/**
 * 列出可以被课时教案挂载的正式单元方案。
 * 后端只返回 active，保证老师在选择器中看到的方案挂载后一定能够被备课引擎读取。
 */
export async function getMountableUnitPlans(
  subject?: string,
): Promise<UnitPlanListResponse> {
  const params = subject ? { subject } : undefined
  const { data } = await apiClient.get('/unit-plans/mountable', { params })
  return data.data as UnitPlanListResponse
}

/**
 * 挂载、更换或解除课时教案关联的单元方案。
 *
 * unitPlanId 非空：挂载或更换。
 * unitPlanId 空串：解除挂载。
 */
export async function updatePlanUnitPlan(
  planId: string,
  unitPlanId: string,
): Promise<{ message: string; mounted: boolean; unit_plan_id: string }> {
  const { data } = await apiClient.put(`/lesson-plans/plans/${planId}/unit-plan`, {
    unit_plan_id: unitPlanId,
  })
  return data.data as {
    message: string
    mounted: boolean
    unit_plan_id: string
  }
}
