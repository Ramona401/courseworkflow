/**
 * 教学风格前测 API 封装
 *
 * 迭代3新增：
 *   - startAssessment：开始前测对话
 *   - chatAssessment：前测对话轮次
 *   - submitAssessment：提交前测结果
 *   - skipAssessment：跳过前测
 *   - getAssessmentResult：获取前测结果
 *   - autoGenerateRecipe：自动生成配方
 */
import apiClient from './client'

// ==================== 类型定义 ====================

/** 前测对话消息 */
export interface AssessmentMessage {
  role: 'user' | 'assistant'
  content: string
  timestamp: string
  step_code: string
}

/** 开始前测响应 */
export interface AssessmentStartResponse {
  session_id: string
  opening_message: AssessmentMessage
  total_steps: number
  current_step: string
}

/** 前测对话响应 */
export interface AssessmentChatResponse {
  ai_message: AssessmentMessage
  current_step: string
  is_complete: boolean
  progress: number
}

/** 前测提交请求 */
export interface AssessmentSubmitRequest {
  experience_years: number
  subject_primary: string
  grade_primary: string
  teaching_style: string
  ai_collaboration: string
  priorities: string[]
  lesson_structure_desc: string
  conversation_log: AssessmentMessage[]
}

/** 教学画像 */
export interface TeachingProfile {
  assessment_version: number
  assessed_at: string
  experience_years: number
  subject_primary: string
  grade_primary: string
  teaching_style: string
  ai_collaboration: string
  priorities: string[]
  recommended_mode: string
  recommended_stages: string[]
  lesson_structure_desc: string
  conversation_log: AssessmentMessage[]
}

/** 前测提交响应 */
export interface AssessmentSubmitResponse {
  profile: TeachingProfile
  recipe_id: string
}

/** 获取前测结果响应 */
export interface AssessmentResultResponse {
  has_profile: boolean
  profile: TeachingProfile | null
}

/** 自动生成配方响应 */
export interface AutoRecipeResponse {
  recipe_id: string
  recipe_name: string
}

// ==================== API 函数 ====================

const BASE = '/lesson-plans/assessment'

/** 开始前测对话 */
export async function startAssessment(): Promise<AssessmentStartResponse> {
  const res = await apiClient.post(`${BASE}/start`)
  return res.data.data
}

/** 前测对话轮次 */
export async function chatAssessment(
  message: string,
  conversationHistory: AssessmentMessage[]
): Promise<AssessmentChatResponse> {
  const res = await apiClient.post(`${BASE}/chat`, {
    message,
    conversation_history: conversationHistory,
  })
  return res.data.data
}

/** 提交前测结果 */
export async function submitAssessment(
  req: AssessmentSubmitRequest
): Promise<AssessmentSubmitResponse> {
  const res = await apiClient.post(`${BASE}/submit`, req)
  return res.data.data
}

/** 跳过前测（使用默认画像） */
export async function skipAssessment(): Promise<AssessmentSubmitResponse> {
  const res = await apiClient.post(`${BASE}/skip`)
  return res.data.data
}

/** 获取前测结果 */
export async function getAssessmentResult(): Promise<AssessmentResultResponse> {
  const res = await apiClient.get(`${BASE}/result`)
  return res.data.data
}

/** 自动生成配方（根据前测结果） */
export async function autoGenerateRecipe(): Promise<AutoRecipeResponse> {
  const res = await apiClient.post(`${BASE}/auto-recipe`)
  return res.data.data
}
