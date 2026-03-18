/**
 * 外部数据配置 API 封装（P3-1新增）
 * - 获取所有外部数据配置（OSS + 推送API）
 * - 批量更新配置（敏感字段AES加密存储）
 * - 仅 admin 可调用
 */
import client from './client'

// ==================== 类型定义 ====================

/** 单条外部数据配置 */
export interface ExternalDataConfigItem {
  config_key: string       // 配置键名
  config_value: string     // 配置值（敏感字段已脱敏）
  description: string      // 配置描述
  is_sensitive: boolean    // 是否为敏感字段
  is_set: boolean          // 是否已配置（非占位符）
  updated_at: string | null // 最后更新时间
}

/** 配置列表响应 */
export interface ExternalDataConfigListResponse {
  configs: ExternalDataConfigItem[]  // 配置列表
  total: number                      // 总数
}

/** 批量更新配置请求 */
export interface UpdateExternalDataConfigsRequest {
  configs: Record<string, string>    // 键值对（key→value，敏感字段空字符串表示不修改）
}

// ==================== API 方法 ====================

/** 获取所有外部数据配置 */
export async function getExternalDataConfigs(): Promise<ExternalDataConfigListResponse> {
  const res = await client.get<{ code: number; data: ExternalDataConfigListResponse }>('/external-data/configs')
  return res.data.data
}

/** 批量更新外部数据配置 */
export async function updateExternalDataConfigs(req: UpdateExternalDataConfigsRequest): Promise<ExternalDataConfigListResponse> {
  const res = await client.put<{ code: number; data: ExternalDataConfigListResponse }>('/external-data/configs', req)
  return res.data.data
}
