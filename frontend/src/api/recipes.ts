/**
 * recipes.ts — 备课配方API封装
 *
 * Phase 7A：基础CRUD
 * 迭代2：流程预设+校验
 * 迭代4B-2：画像感知智能推荐
 * 迭代5：自定义阶段CRUD
 */
import apiClient from './client'

// ==================== 类型定义 ====================

/** 教案结构板块 */
export interface LessonStructureBlock {
  name: string
  required: boolean
  requirement: string
  order: number
  sub_sections?: LessonStructureSubSection[]
}

/** 教学过程子环节 */
export interface LessonStructureSubSection {
  name: string
  duration: number
  goal: string
  output_requirement: string
}

/** 备课模式类型 */
export type PromptMode = 'guided' | 'efficient' | 'per_stage'

/** 流程配置项（迭代5扩展：is_custom+stage_name） */
export interface StageFlowItem {
  stage_code: string
  enabled: boolean
  order: number
  prompt_mode?: string
  is_custom?: boolean    // 迭代5：是否为自定义阶段
  stage_name?: string    // 迭代5：自定义阶段显示名称
}

/** 流程预设模板 */
export interface FlowPreset {
  key: string
  name: string
  description: string
  duration: string
  icon: string
  stages: StageFlowItem[]
  prompt_mode: PromptMode
}

/** 流程校验消息 */
export interface FlowValidationMessage {
  level: 'info' | 'warning' | 'error'
  code: string
  message: string
}

/** 流程校验结果 */
export interface FlowValidationResult {
  valid: boolean
  messages: FlowValidationMessage[]
}

/** 配方详情 */
export interface RecipeDetail {
  id: string
  name: string
  description: string
  subject: string
  grade_range: string
  component_ids: string  // JSON字符串
  student_profile: string
  teaching_style: string
  school_requirements: string
  custom_notes: string
  custom_prompt: string
  scope: string
  scope_ref_id?: string
  author_id: string
  fork_count: number
  forked_from?: string
  use_count: number
  version: number
  status: string
  stages_config: string     // JSON字符串
  lesson_structure: string  // JSON字符串
  prompt_mode: PromptMode
  component_count: number
  components: RecipeComponentBrief[]
  author_name: string
  scope_name: string
  created_at: string
  updated_at: string
}

/** 配方列表项 */
export interface RecipeListItem {
  id: string
  name: string
  description: string
  subject: string
  grade_range: string
  component_count: number
  scope: string
  scope_name: string
  author_id: string
  author_name: string
  fork_count: number
  use_count: number
  version: number
  forked_from?: string
  status: string
  prompt_mode: PromptMode
  stages_config: string
  created_at: string
  updated_at: string
}

/** 组件摘要 */
export interface RecipeComponentBrief {
  id: string
  library_type: string
  library_name: string
  display_label: string
  quality_score: number
  status: string
}

/** 推荐组件组 */
export interface RecommendedComponentGroup {
  library_type: string
  library_name: string
  components: RecommendedComponent[]
}

/** 推荐组件 */
export interface RecommendedComponent {
  id: string
  display_label: string
  quality_score: number
  grade_range: string
}

/** 配方上下文预览 */
export interface RecipeContextPreview {
  recipe_id: string
  recipe_name: string
  context_text: string
  token_estimate: number
}

// ==================== 迭代5新增：自定义阶段类型 ====================

/** 创建自定义阶段请求 */
export interface CreateCustomStageRequest {
  stage_code: string
  stage_name: string
  ai_role: string
  system_prompt?: string
  prompt_variants?: string
  output_format?: string
  component_types?: string
  gate_mode?: string
  skippable?: boolean
}

/** 更新自定义阶段请求 */
export interface UpdateCustomStageRequest {
  stage_name: string
  ai_role: string
  system_prompt?: string
  prompt_variants?: string
  output_format?: string
  component_types?: string
  gate_mode?: string
  skippable?: boolean
}

/** 自定义阶段响应 */
export interface CustomStageResponse {
  stage_code: string
  stage_name: string
  ai_role: string
  gate_mode: string
  skippable: boolean
  has_prompt: boolean
}

// ==================== 配方 CRUD ====================

export async function getRecipes(params?: Record<string, string>) {
  const { data } = await apiClient.get('/lesson-plans/recipes', { params })
  return data.data as { recipes: RecipeListItem[]; total: number }
}

export async function getRecipe(id: string) {
  const { data } = await apiClient.get(`/lesson-plans/recipes/${id}`)
  return data.data as RecipeDetail
}

export async function createRecipe(payload: Record<string, unknown>) {
  const { data } = await apiClient.post('/lesson-plans/recipes', payload)
  return data.data as RecipeDetail
}

export async function updateRecipe(id: string, payload: Record<string, unknown>) {
  const { data } = await apiClient.put(`/lesson-plans/recipes/${id}`, payload)
  return data.data
}

export async function deleteRecipe(id: string) {
  const { data } = await apiClient.delete(`/lesson-plans/recipes/${id}`)
  return data.data
}

export async function forkRecipe(id: string) {
  const { data } = await apiClient.post(`/lesson-plans/recipes/${id}/fork`)
  return data.data as RecipeDetail
}

export async function shareRecipe(id: string, payload: { scope: string; scope_ref_id: string }) {
  const { data } = await apiClient.put(`/lesson-plans/recipes/${id}/share`, payload)
  return data.data
}

export async function updateRecipeStudentProfile(id: string, studentProfile: string) {
  const { data } = await apiClient.put(`/lesson-plans/recipes/${id}/student-profile`, { student_profile: studentProfile })
  return data.data
}

export async function previewRecipeContext(id: string) {
  const { data } = await apiClient.get(`/lesson-plans/recipes/${id}/preview-context`)
  return data.data as RecipeContextPreview
}

// ==================== 推荐 ====================

export async function recommendRecipeComponents(payload: { subject: string; grade_range: string }) {
  const { data } = await apiClient.post('/lesson-plans/recipes/recommend', payload)
  return data.data as { groups: RecommendedComponentGroup[] }
}

export async function smartRecommendComponents(payload: { subject: string; grade_range: string }) {
  const { data } = await apiClient.post('/lesson-plans/recipes/smart-recommend', payload)
  return data.data as { groups: RecommendedComponentGroup[] }
}

// ==================== 流程预设+校验 ====================

export async function getFlowPresets() {
  const { data } = await apiClient.get('/lesson-plans/recipes/flow-presets')
  return data.data as { presets: FlowPreset[] }
}

export async function validateFlow(stages: StageFlowItem[]) {
  const { data } = await apiClient.post('/lesson-plans/recipes/validate-flow', { stages })
  return data.data as FlowValidationResult
}

// ==================== 迭代5新增：自定义阶段 CRUD ====================

/** 获取配方的自定义阶段列表 */
export async function getCustomStages(recipeId: string) {
  const { data } = await apiClient.get(`/lesson-plans/recipes/${recipeId}/custom-stages`)
  return data.data as { stages: CustomStageResponse[] }
}

/** 创建配方自定义阶段 */
export async function createCustomStage(recipeId: string, payload: CreateCustomStageRequest) {
  const { data } = await apiClient.post(`/lesson-plans/recipes/${recipeId}/custom-stages`, payload)
  return data.data as CustomStageResponse
}

/** 更新配方自定义阶段 */
export async function updateCustomStage(recipeId: string, stageCode: string, payload: UpdateCustomStageRequest) {
  const { data } = await apiClient.put(`/lesson-plans/recipes/${recipeId}/custom-stages/${stageCode}`, payload)
  return data.data
}

/** 删除配方自定义阶段 */
export async function deleteCustomStage(recipeId: string, stageCode: string) {
  const { data } = await apiClient.delete(`/lesson-plans/recipes/${recipeId}/custom-stages/${stageCode}`)
  return data.data
}

// ==================== 迭代6新增：配方效果统计 ====================

/** 使用记录行 */
export interface RecipeUsageRow {
  lesson_plan_id: string | null
  user_name: string
  ai_review_score: number | null
  created_at: string
}

/** 配方效果统计响应 */
export interface RecipeStatsResponse {
  recipe_id: string
  recipe_name: string
  total_usage: number
  total_plans: number
  avg_score: number
  recent_usages: RecipeUsageRow[]
}

/** 获取配方效果统计 */
export async function getRecipeStats(recipeId: string) {
  const { data } = await apiClient.get(`/lesson-plans/recipes/${recipeId}/stats`)
  return data.data as RecipeStatsResponse
}

// ==================== 迭代6新增：配方市场 ====================

/** 市场配方列表项 */
export interface MarketRecipeItem {
  id: string
  name: string
  description: string
  subject: string
  grade_range: string
  component_count: number
  scope: string
  scope_name: string
  author_id: string
  author_name: string
  fork_count: number
  use_count: number
  avg_score: number
  plan_count: number
  prompt_mode: string
  created_at: string
  updated_at: string
}

/** 配方市场响应 */
export interface MarketRecipesResponse {
  recipes: MarketRecipeItem[]
  total: number
}

/** 查询配方市场排行榜 */
export async function getMarketRecipes(params?: Record<string, string | number>) {
  const { data } = await apiClient.get('/lesson-plans/recipes/market', { params })
  return data.data as MarketRecipesResponse
}
