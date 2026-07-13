/**
 * assistant-style-profile.ts — 教学风格与成长画像前端API
 *
 * 原始Word/PDF在浏览器端完成文字提取，本接口只发送提取后的文本。
 * 平台教案只发送source_id，由后端校验归属并读取正文。
 */

import client from './client'

export type StyleProfileSourceType =
  | 'platform_plan'
  | 'docx'
  | 'pdf'
  | 'pasted'

export type StyleProfileIntent =
  | 'satisfied_example'
  | 'representative'
  | 'local_standard'
  | 'needs_improvement'
  | 'structure_only'
  | 'language_only'
  | 'negative_example'

export interface StyleProfileMaterial {
  title: string
  source_type: StyleProfileSourceType
  source_id?: string
  intent: StyleProfileIntent
  content?: string
}

export interface StyleProfileRequest {
  subject?: string
  grade?: string
  materials: StyleProfileMaterial[]
}

export interface StyleProfileResponse {
  profile_markdown: string
  material_count: number
  total_characters: number
  confidence: 'low' | 'medium' | 'high' | string
  warnings: string[]
}

/**
 * 分析历史教案和教研材料，生成可编辑的教学风格与成长画像。
 */
export async function analyzeAssistantStyleProfile(
  req: StyleProfileRequest,
): Promise<StyleProfileResponse> {
  const res = await client.post<{
    code: number
    data: StyleProfileResponse
  }>('/ai-assistants/design/profile-materials', req)

  return res.data.data!
}
