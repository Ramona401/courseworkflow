/**
 * AI配置 API 封装
 * - 全局配置：读取/更新（API地址、Key、模型、温度、Token数）
 * - 场景配置：读取/更新（6个Pipeline步骤各自的AI参数）
 * - 连通性测试：验证AI API连接状态（P2-2新增）
 * - 可用模型查询：查询当前Key下可用模型列表（v19.1新增）
 * - 仅 admin 可调用
 */
import client from './client'

// ==================== 类型定义 ====================

/** 全局AI配置响应 */
export interface GlobalConfig {
  api_base_url: string    // AI API 基础地址
  api_key: string         // API Key（脱敏显示）
  api_key_set: boolean    // API Key 是否已配置
  default_model: string   // 默认模型
  temperature: string     // 默认温度
  max_tokens: string      // 默认最大Token数
  updated_at: string | null // 最近更新时间
}

/** 更新全局配置请求 */
export interface UpdateGlobalConfigRequest {
  api_base_url: string    // API 基础地址
  api_key: string         // API Key（明文；空字符串表示不修改）
  default_model: string   // 默认模型
  temperature: string     // 温度（字符串）
  max_tokens: string      // 最大Token数（字符串）
}

/** 场景配置响应 */
export interface SceneConfig {
  id: string              // UUID
  scene_code: string      // 场景代码
  scene_name: string      // 场景中文名
  model: string | null    // 模型（null=继承全局）
  temperature: number | null // 温度（null=继承全局）
  max_tokens: number | null  // 最大Token（null=继承全局）
  system_prompt_id: string | null // 关联提示词ID
  is_active: boolean      // 是否启用
  updated_at: string | null // 最近更新时间
}

/** 更新场景配置请求 */
export interface UpdateSceneConfigRequest {
  model?: string | null
  temperature?: number | null
  max_tokens?: number | null
  system_prompt_id?: string | null
  is_active?: boolean
}

/** AI连通性测试结果（P2-2新增） */
export interface TestConnectionResult {
  success: boolean        // 测试是否成功
  message: string         // 结果描述
  latency_ms: number      // 延迟（毫秒）
  model: string           // 测试使用的模型
  api_base_url: string    // 测试使用的API地址
}

/** 单个可用模型信息（v19.1新增） */
export interface ModelInfo {
  id: string              // 模型ID（如 anthropic/claude-haiku-4.5）
}

/** 可用模型列表响应（v19.1新增） */
export interface ListModelsResult {
  models: ModelInfo[]     // 模型列表（按字母排序）
  total: number           // 模型总数
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

/** 测试AI API连通性（P2-2新增） */
export async function testConnection(): Promise<TestConnectionResult> {
  const res = await client.post<{ code: number; data: TestConnectionResult }>('/ai-config/test')
  return res.data.data
}

/** 查询当前Key下可用模型列表（v19.1新增） */
export async function listModels(): Promise<ListModelsResult> {
  const res = await client.get<{ code: number; data: ListModelsResult }>('/ai-config/models')
  return res.data.data
}
