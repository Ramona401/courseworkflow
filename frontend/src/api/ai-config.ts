/**
 * AI配置 API 封装
 * - 全局配置：读取/更新（API地址、Key、模型、温度、Token数）
 * - 场景配置：读取/更新（各场景AI参数 + v85 fallback降级模型）
 * - 连通性测试：验证AI API连接状态
 * - 可用模型查询：查询当前Key下可用模型列表
 * - 仅 admin 可调用
 */
import client from './client'

// ==================== 类型定义 ====================

/** 全局AI配置响应 */
export interface GlobalConfig {
  api_base_url: string
  api_key: string
  api_key_set: boolean
  default_model: string
  temperature: string
  max_tokens: string
  updated_at: string | null
}

/** 更新全局配置请求 */
export interface UpdateGlobalConfigRequest {
  api_base_url: string
  api_key: string
  default_model: string
  temperature: string
  max_tokens: string
}

/** 场景配置响应（v85：新增 scene_group + fallback_models） */
export interface SceneConfig {
  id: string
  scene_code: string
  scene_name: string
  scene_group: string           // v78: lesson_plan / pipeline
  model: string | null
  temperature: number | null
  max_tokens: number | null
  system_prompt_id: string | null
  is_active: boolean
  fallback_models: string[]     // v85新增：降级模型列表
  updated_at: string | null
}

/** 更新场景配置请求（v85：新增 fallback_models） */
export interface UpdateSceneConfigRequest {
  model?: string | null
  temperature?: number | null
  max_tokens?: number | null
  system_prompt_id?: string | null
  is_active?: boolean
  fallback_models?: string[]    // v85新增：降级模型列表
}

/** AI连通性测试结果 */
export interface TestConnectionResult {
  success: boolean
  message: string
  latency_ms: number
  model: string
  api_base_url: string
}

/** 单个可用模型信息 */
export interface ModelInfo {
  id: string
}

/** 可用模型列表响应 */
export interface ListModelsResult {
  models: ModelInfo[]
  total: number
}

// ==================== API 方法 ====================

/** 获取全局AI配置 */
export async function getGlobalConfig(): Promise<GlobalConfig> {
  const res = await client.get<{ code: number; data: GlobalConfig }>('/ai-config/global')
  return res.data.data
}

/** 更新全局AI配置 */
export async function updateGlobalConfig(req: UpdateGlobalConfigRequest): Promise<GlobalConfig> {
  const res = await client.put<{ code: number; data: GlobalConfig }>('/ai-config/global', req)
  return res.data.data
}

/** 获取所有场景配置 */
export async function getSceneConfigs(): Promise<SceneConfig[]> {
  const res = await client.get<{ code: number; data: SceneConfig[] }>('/ai-config/scenes')
  return res.data.data
}

/** 更新指定场景配置 */
export async function updateSceneConfig(code: string, req: UpdateSceneConfigRequest): Promise<SceneConfig[]> {
  const res = await client.put<{ code: number; data: SceneConfig[] }>(`/ai-config/scenes/${code}`, req)
  return res.data.data
}

/** 测试AI API连通性 */
export async function testConnection(): Promise<TestConnectionResult> {
  const res = await client.post<{ code: number; data: TestConnectionResult }>('/ai-config/test')
  return res.data.data
}

/** 查询当前Key下可用模型列表 */
export async function listModels(): Promise<ListModelsResult> {
  const res = await client.get<{ code: number; data: ListModelsResult }>('/ai-config/models')
  return res.data.data
}
