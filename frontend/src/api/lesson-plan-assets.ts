/**
 * lesson-plan-assets.ts — 教案附属资产 API 封装
 *
 * v123 新增:为教案系统的图片/文件上传提供前端 API
 *
 * 5 个接口:
 *   - uploadAsset(planID, file, altText)     上传图片(multipart),返回 url + 已拼好的 markdown
 *   - listAssets(planID)                      列出教案所有资产
 *   - getAsset(assetID)                       单条资产详情
 *   - updateAssetAltText(assetID, altText)    更新 alt 文本
 *   - deleteAsset(assetID)                    删除资产
 *
 * 设计要点:
 *   - 所有上传通过 FormData(multipart/form-data),走与 textbooks.ts 一致的风格
 *   - 上传响应直接返回拼好的 markdown 片段,前端编辑器无需自己拼字符串
 */
import apiClient from './client'

// ==================== 类型定义 ====================

/** 单条资产实体 */
export interface LessonPlanAsset {
  id: string
  lesson_plan_id: string
  uploader_id: string
  asset_type: 'image' | 'file'
  file_name: string
  file_path: string
  file_size: number
  mime_type: string
  alt_text: string
  width: number
  height: number
  created_at: string
  updated_at: string
}

/** 列表项(包含可访问 URL 和上传者名) */
export interface LessonPlanAssetListItem extends LessonPlanAsset {
  url: string
  uploader_name: string
}

/** 列表响应 */
export interface AssetListResponse {
  assets: LessonPlanAssetListItem[]
  total: number
}

/** 上传成功响应 */
export interface AssetUploadResponse {
  id: string
  file_name: string
  file_size: number
  url: string       // 完整 URL,可直接拼到 ![alt](url) 中
  markdown: string  // 已拼好的 markdown 片段,可一键插入编辑器
}

// ==================== API 函数 ====================

/**
 * 上传教案图片
 * @param planID  教案 ID
 * @param file    图片文件(浏览器 File 对象)
 * @param altText 图片描述(可选,空则后端用文件名兜底)
 */
export async function uploadAsset(
  planID: string,
  file: File,
  altText?: string,
): Promise<AssetUploadResponse> {
  const formData = new FormData()
  formData.append('file', file)
  if (altText) formData.append('alt_text', altText)

  const { data } = await apiClient.post(
    `/lesson-plans/plans/${planID}/assets`,
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return data.data as AssetUploadResponse
}

/** 列出教案所有资产 */
export async function listAssets(planID: string): Promise<AssetListResponse> {
  const { data } = await apiClient.get(`/lesson-plans/plans/${planID}/assets`)
  return data.data as AssetListResponse
}

/** 获取单条资产详情 */
export async function getAsset(assetID: string): Promise<LessonPlanAssetListItem> {
  const { data } = await apiClient.get(`/lesson-plans/assets/${assetID}`)
  return data.data as LessonPlanAssetListItem
}

/** 更新资产 alt 文本 */
export async function updateAssetAltText(assetID: string, altText: string): Promise<void> {
  await apiClient.put(`/lesson-plans/assets/${assetID}`, { alt_text: altText })
}

/** 删除资产 */
export async function deleteAsset(assetID: string): Promise<void> {
  await apiClient.delete(`/lesson-plans/assets/${assetID}`)
}

// ==================== 工具函数 ====================

/**
 * 校验文件是否符合上传规范(前端预校验,减少无谓请求)
 * @returns 错误消息字符串,空字符串表示通过
 */
export function validateImageFile(file: File): string {
  const MAX_SIZE = 5 * 1024 * 1024 // 5MB
  const ALLOWED_TYPES = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp', 'image/gif']

  if (!ALLOWED_TYPES.includes(file.type)) {
    return '仅支持 JPG / PNG / WEBP / GIF 格式图片'
  }
  if (file.size > MAX_SIZE) {
    return `文件过大(${(file.size / 1024 / 1024).toFixed(1)}MB),最大支持 5MB`
  }
  return ''
}
