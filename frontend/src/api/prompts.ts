/**
 * 提示词管理 API 封装（P2-3）
 * - 提示词列表：获取所有8个槽位的当前生效版本
 * - 提示词详情：获取指定槽位的完整内容
 * - 提示词更新：创建新版本（保留历史版本）
 * - 版本历史：查看指定槽位的所有历史版本
 * - 版本回滚：将指定槽位恢复到历史版本
 * - 仅 admin 可调用
 */
import client from './client'

// ==================== 类型定义 ====================

/** 提示词信息（当前生效版本） */
export interface PromptInfo {
  id: string              // UUID
  prompt_key: string      // 提示词标识（prompt_a~f / dict / ability_table）
  prompt_name: string     // 提示词中文名
  content: string         // 提示词完整内容
  version: number         // 当前版本号
  content_len: number     // 内容长度（字符数）
  is_current: boolean     // 是否为当前版本
  created_by: string | null // 创建者ID
  created_at: string      // 创建时间
}

/** 提示词列表响应 */
export interface PromptListResponse {
  prompts: PromptInfo[]   // 提示词列表
  total: number           // 总数
}

/** 版本历史记录 */
export interface PromptVersion {
  id: string              // 版本记录UUID
  version: number         // 版本号
  content: string         // 该版本的内容
  content_len: number     // 内容长度
  is_current: boolean     // 是否为当前生效版本
  created_by: string | null // 创建者ID
  created_at: string      // 创建时间
}

/** 版本历史列表响应 */
export interface PromptVersionListResponse {
  prompt_key: string           // 提示词标识
  prompt_name: string          // 提示词中文名
  versions: PromptVersion[]    // 版本列表（按版本号倒序）
  total: number                // 总版本数
}

/** 更新提示词请求 */
export interface UpdatePromptRequest {
  content: string         // 新的提示词完整内容
}

/** 回滚请求 */
export interface RollbackRequest {
  version_id: string      // 目标版本ID
}

// ==================== API 方法 ====================

/** 获取所有提示词（当前生效版本） */
export async function getPrompts(): Promise<PromptListResponse> {
  const res = await client.get<{ code: number; data: PromptListResponse }>('/prompts')
  return res.data.data
}

/** 获取指定提示词详情 */
export async function getPromptByKey(key: string): Promise<PromptInfo> {
  const res = await client.get<{ code: number; data: PromptInfo }>(`/prompts/${key}`)
  return res.data.data
}

/** 更新提示词内容（创建新版本） */
export async function updatePrompt(key: string, req: UpdatePromptRequest): Promise<PromptInfo> {
  const res = await client.put<{ code: number; data: PromptInfo }>(`/prompts/${key}`, req)
  return res.data.data
}

/** 获取指定提示词的版本历史 */
export async function getPromptVersions(key: string): Promise<PromptVersionListResponse> {
  const res = await client.get<{ code: number; data: PromptVersionListResponse }>(`/prompts/${key}/versions`)
  return res.data.data
}

/** 回滚到指定版本 */
export async function rollbackPromptVersion(key: string, req: RollbackRequest): Promise<PromptInfo> {
  const res = await client.post<{ code: number; data: PromptInfo }>(`/prompts/${key}/rollback`, req)
  return res.data.data
}
