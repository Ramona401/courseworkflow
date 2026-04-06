/**
 * textbooks.ts — 课本上传API封装
 *
 * 迭代7新增：课本图片上传/列表/详情/更新/删除/OCR识别
 */
import apiClient from './client'

// ==================== 类型定义 ====================

/** 课本页面列表项 */
export interface TextbookListItem {
  id: string
  subject: string
  grade_range: string
  textbook_name: string
  chapter: string
  page_number: number
  file_name: string
  file_size: number
  mime_type: string
  has_ocr: boolean
  description: string
  scope: string
  scope_name: string
  uploaded_by: string
  uploader_name: string
  usage_count: number
  image_url: string
  created_at: string
}

/** 课本页面详情 */
export interface TextbookDetail {
  id: string
  subject: string
  grade_range: string
  textbook_name: string
  chapter: string
  page_number: number
  file_name: string
  file_path: string
  file_size: number
  mime_type: string
  ocr_text: string
  ocr_model: string
  ocr_at: string | null
  description: string
  tags: string
  scope: string
  scope_ref_id: string | null
  uploaded_by: string
  usage_count: number
  status: string
  uploader_name: string
  image_url: string
  has_ocr: boolean
  created_at: string
  updated_at: string
}

/** 课本列表响应 */
export interface TextbookListResponse {
  pages: TextbookListItem[]
  total: number
}

// ==================== API函数 ====================

/** 上传课本图片（multipart/form-data） */
export async function uploadTextbook(formData: FormData) {
  const { data } = await apiClient.post('/lesson-plans/textbooks/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data.data as { id: string; file_name: string; file_size: number; image_url: string }
}

/** 查询课本页面列表 */
export async function getTextbooks(params?: Record<string, string | number>) {
  const { data } = await apiClient.get('/lesson-plans/textbooks', { params })
  return data.data as TextbookListResponse
}

/** 获取课本页面详情 */
export async function getTextbook(id: string) {
  const { data } = await apiClient.get(`/lesson-plans/textbooks/${id}`)
  return data.data as TextbookDetail
}

/** 更新课本页面元数据 */
export async function updateTextbook(id: string, payload: Record<string, unknown>) {
  const { data } = await apiClient.put(`/lesson-plans/textbooks/${id}`, payload)
  return data.data
}

/** 删除课本页面 */
export async function deleteTextbook(id: string) {
  const { data } = await apiClient.delete(`/lesson-plans/textbooks/${id}`)
  return data.data
}

/** 触发AI OCR识别 */
export async function triggerTextbookOCR(id: string) {
  const { data } = await apiClient.post(`/lesson-plans/textbooks/${id}/ocr`)
  return data.data as { ocr_text: string; message: string }
}
