/**
 * AI配置 API 封装
 * - 全局配置：读取/更新（API地址、Key、模型、温度、Token数）
 * - 场景配置：读取/更新（各场景AI参数 + v85 fallback降级模型）
 * - 连通性测试：验证AI API连接状态
 * - 可用模型查询：查询当前Key下可用模型列表
 * - TTS语音合成配置：查看/保存/服务端自测（S-V1.5b新增）
 * - 境内文本网关配置：查看/保存/服务端自测（批一新增，双网关分流降级通道）
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

// ==================== TTS配置类型（S-V1.5b新增） ====================

/**
 * TTS配置视图（GET/PUT 共用响应结构）
 * 对齐后端 tts_config_handler.go 的 ttsConfigView
 */
export interface TTSConfigView {
  provider: string            // volcano_v3（火山v3直连，默认）/ volcano_openai（OpenAI兼容备用通道）
  app_id: string              // 火山豆包语音应用 APP ID（明文展示）
  access_token: string        // 脱敏展示（首尾4字符+***），未配置时为"未配置"
  access_token_set: boolean   // 是否已配置 Access Token
  voices_total: number        // 当前内置音色数量
}

/** 更新TTS配置请求（三字段均可选，留空=不修改对应项） */
export interface UpdateTTSConfigRequest {
  provider?: string           // volcano_v3 / volcano_openai
  app_id?: string             // 火山 APP ID
  access_token?: string       // Access Token明文（后端AES加密存储），留空表示不修改
}

/**
 * TTS链路自测结果
 * 对齐后端 TestTTS 的 map 响应：成功时含 model/duration/file_size，
 * 失败时仅 success/message（可能附 provider/latency_ms）
 */
export interface TestTTSResult {
  success: boolean
  provider?: string           // 实际使用的通道
  model?: string              // 实际resource/模型标识
  duration?: number           // 合成音频时长（秒）
  file_size?: number          // 合成音频字节数
  latency_ms?: number         // 合成耗时（毫秒）
  message: string             // 人类可读结论
}

// ==================== 境内文本网关配置类型（批一新增） ====================

/**
 * 境内网关配置视图（GET/PUT 共用响应结构）
 * 对齐后端 domestic_gateway_handler.go 的 domesticGatewayView
 * 用途：双网关分流的「境内降级通道」——未授权学校的境外文本调用会整通道切到这里
 *       （dashscope 兼容模式 + qwen-max）。三键 domestic_text_base_url/key_enc/model。
 */
export interface DomesticGatewayView {
  base_url: string            // 境内网关地址（明文展示，如 dashscope /compatible-mode/v1）
  model: string               // 境内主力模型（明文展示，如 qwen-max）
  api_key: string             // 脱敏展示（首尾4位+***），未配置时为"未配置"
  api_key_set: boolean        // 是否已配置 API Key
}

/** 更新境内网关配置请求（三字段均可选，留空=不修改对应项） */
export interface UpdateDomesticGatewayRequest {
  base_url?: string           // 境内网关地址
  model?: string              // 境内主力模型
  api_key?: string            // API Key明文（后端AES加密存储），留空表示不修改
}

/**
 * 境内网关链路自测结果
 * 对齐后端 TestDomesticGateway 的 dgTestResult
 */
export interface DomesticGatewayTestResult {
  success: boolean
  message: string             // 人类可读结论
  latency_ms?: number         // 请求耗时（毫秒）
  model?: string              // 测试使用的模型
  base_url?: string           // 测试使用的网关地址
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

// ==================== TTS配置 API（S-V1.5b新增，admin专属） ====================

/** 获取当前TTS语音合成配置（Access Token脱敏回显） */
export async function getTTSConfig(): Promise<TTSConfigView> {
  const res = await client.get<{ code: number; data: TTSConfigView }>('/admin/tts-config')
  return res.data.data
}

/** 保存TTS配置（provider/app_id/access_token逐项可选，留空不修改） */
export async function updateTTSConfig(req: UpdateTTSConfigRequest): Promise<TTSConfigView> {
  const res = await client.put<{ code: number; data: TTSConfigView }>('/admin/tts-config', req)
  return res.data.data
}

/**
 * TTS链路自测：后端用库内当前配置直连火山合成一句测试音频，
 * 成功后立即删除测试文件，仅返回链路结论（latency/duration/file_size）
 */
export async function testTTSConnection(): Promise<TestTTSResult> {
  const res = await client.post<{ code: number; data: TestTTSResult }>('/admin/tts-config/test', {})
  return res.data.data
}

// ==================== 境内文本网关 API（批一新增，admin专属） ====================

/** 获取当前境内网关配置（API Key脱敏回显） */
export async function getDomesticGateway(): Promise<DomesticGatewayView> {
  const res = await client.get<{ code: number; data: DomesticGatewayView }>('/admin/domestic-gateway')
  return res.data.data
}

/** 保存境内网关配置（base_url/model/api_key逐项可选，留空不修改）
 *  后端保存成功后会立即作废分流模块的境内通道5分钟缓存，修改即时生效 */
export async function updateDomesticGateway(req: UpdateDomesticGatewayRequest): Promise<DomesticGatewayView> {
  const res = await client.put<{ code: number; data: DomesticGatewayView }>('/admin/domestic-gateway', req)
  return res.data.data
}

/**
 * 境内网关链路自测：后端用库内当前三键直连 dashscope 发一句最短测试请求，
 * 不写任何数据，仅返回链路结论（latency/model/base_url）
 */
export async function testDomesticGateway(): Promise<DomesticGatewayTestResult> {
  const res = await client.post<{ code: number; data: DomesticGatewayTestResult }>('/admin/domestic-gateway/test', {})
  return res.data.data
}

/** 境内网关可用模型查询结果 */
export interface DomesticModelsResult {
  models: string[]
  total: number
  message?: string   // 配置缺失/查询失败时的说明（成功时无）
}

/** 查询境内网关实际可用的模型名列表（用三键调 dashscope /models） */
export async function getDomesticModels(): Promise<DomesticModelsResult> {
  const res = await client.get<{ code: number; data: DomesticModelsResult }>('/admin/domestic-gateway/models')
  return res.data.data
}


// ==================== 双网关展示名 API（批三-1新增，admin专属） ====================
// 后端路由：/api/v1/admin/gateway-naming（adminOnly）
//   GET 查看两网关展示名 / PUT 更新（overseas_label/domestic_label，留空不修改）
// 用途：给境外/境内两网关各起业务可读展示名，供配置界面与将来老师侧渲染读取，
//       避免对外直接暴露"境外/claude/qwen"等字眼。老师侧公开读接口在批三-3统一接。

/** 双网关展示名视图（对齐后端 gatewayNamingView），未配置字段为空串 */
export interface GatewayNamingView {
  overseas_label: string      // 境外网关展示名（空串表示未配置，前端用默认兜底）
  domestic_label: string      // 境内网关展示名（空串表示未配置）
}

/** 更新双网关展示名请求（两字段均可选，留空=不修改对应项） */
export interface UpdateGatewayNamingRequest {
  overseas_label?: string
  domestic_label?: string
}

/** 获取双网关展示名 */
export async function getGatewayNaming(): Promise<GatewayNamingView> {
  const res = await client.get<{ code: number; data: GatewayNamingView }>('/admin/gateway-naming')
  return res.data.data
}

/** 更新双网关展示名（overseas_label/domestic_label 留空表示不修改） */
export async function updateGatewayNaming(req: UpdateGatewayNamingRequest): Promise<GatewayNamingView> {
  const res = await client.put<{ code: number; data: GatewayNamingView }>('/admin/gateway-naming', req)
  return res.data.data
}


// ==================== 模型别名映射 API（批三-2新增，admin专属） ====================
// 后端路由：/api/v1/admin/model-alias/*（adminOnly）
//   GET/POST   /rules        列表/新增规则
//   PUT/DELETE /rules/{id}   更新/删除规则
//   GET/PUT    /fallback     查/改兜底别名
//   POST       /preview      预览（输入模型名→返回别名）
// 用途：真实模型名→业务别名映射（exact精确/prefix前缀，精确优先），避免老师侧暴露真实模型。

/** 模型别名规则（对齐后端 ModelAliasRule） */
export interface ModelAliasRule {
  id: string
  match_type: 'exact' | 'prefix'   // 精确 / 前缀
  pattern: string                   // 模型名或前缀
  alias: string                     // 业务别名
  priority: number                  // 同时命中时大者优先
  enabled: boolean
  note: string
  created_by: string | null
  created_at: string
  updated_at: string
}

/** 新增/更新规则请求 */
export interface ModelAliasRuleRequest {
  match_type: 'exact' | 'prefix'
  pattern: string
  alias: string
  priority: number
  enabled: boolean
  note?: string
}

/** 列出全部别名规则 */
export async function getModelAliasRules(): Promise<ModelAliasRule[]> {
  const res = await client.get<{ code: number; data: { items: ModelAliasRule[]; total: number } }>(
    '/admin/model-alias/rules'
  )
  return res.data.data?.items ?? []
}

/** 新增别名规则，返回新ID */
export async function createModelAliasRule(req: ModelAliasRuleRequest): Promise<string> {
  const res = await client.post<{ code: number; data: { id: string } }>(
    '/admin/model-alias/rules', req
  )
  return res.data.data?.id ?? ''
}

/** 更新别名规则 */
export async function updateModelAliasRule(id: string, req: ModelAliasRuleRequest): Promise<void> {
  await client.put(`/admin/model-alias/rules/${id}`, req)
}

/** 删除别名规则 */
export async function deleteModelAliasRule(id: string): Promise<void> {
  await client.delete(`/admin/model-alias/rules/${id}`)
}

/** 查兜底别名 */
export async function getModelAliasFallback(): Promise<string> {
  const res = await client.get<{ code: number; data: { fallback: string } }>(
    '/admin/model-alias/fallback'
  )
  return res.data.data?.fallback ?? ''
}

/** 改兜底别名 */
export async function setModelAliasFallback(fallback: string): Promise<string> {
  const res = await client.put<{ code: number; data: { fallback: string } }>(
    '/admin/model-alias/fallback', { fallback }
  )
  return res.data.data?.fallback ?? ''
}

/** 预览：输入真实模型名，返回当前会显示的别名 */
export async function previewModelAlias(model: string): Promise<{ model: string; alias: string }> {
  const res = await client.post<{ code: number; data: { model: string; alias: string } }>(
    '/admin/model-alias/preview', { model }
  )
  return res.data.data
}
