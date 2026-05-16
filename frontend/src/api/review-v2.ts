/**
 * review-v2.ts — 多级审核API封装
 *
 * v127.2 新增：
 *   - getReviewedRecords: 已审核记录列表（支持按级别/决策过滤）
 */
import apiClient from './client'

// ==================== 类型定义 ====================

export interface ReviewDecisionRequest {
  decision: 'approved' | 'revision'
  score?: number
  comment: string
  dimensions?: string
}

export interface ReviewV2Item {
  id: string
  lesson_plan_id: string
  review_level: number
  level_name: string
  reviewer_id: string
  reviewer_name: string
  decision: string
  score: number | null
  comment: string
  review_round: number
  created_at: string
}

export interface ReviewHistoryResponse {
  reviews: ReviewV2Item[]
  total: number
  current_level: number
}

export interface PendingReviewItem {
  lesson_plan_id: string
  title: string
  subject: string
  grade: string
  author_id: string
  author_name: string
  group_id: string | null
  group_name: string
  school_name: string
  review_level: number
  level_name: string
  ai_review_score: number | null
  submitted_at: string
}

export interface PendingReviewListResponse {
  items: PendingReviewItem[]
  total: number
}

export interface ReviewStatsResponse {
  total_pending: number
  total_reviewed: number
  total_approved: number
  total_revision: number
}

/** 已审核记录列表项（v127.2新增） */
export interface ReviewedListItem {
  id: string
  lesson_plan_id: string
  plan_title: string
  plan_subject: string
  plan_grade: string
  author_name: string
  review_level: number
  level_name: string
  reviewer_name: string
  decision: string
  score: number | null
  comment: string
  created_at: string
}

export interface ReviewedListResponse {
  items: ReviewedListItem[]
  total: number
}

export interface ReviewFlowConfig {
  school_id: string
  school_name: string
  l2_enabled: boolean
  l3_sample_rate: number
  auto_publish_on_approved: boolean
}

export interface UpdateReviewFlowConfigRequest {
  school_id: string
  config: {
    l2_enabled: boolean
    l3_sample_rate: number
    auto_publish_on_approved: boolean
  }
}

// ==================== 辅助函数 ====================

function extractData<T>(resp: { data?: { data?: T } }): T {
  const d = resp?.data as Record<string, unknown> | undefined
  if (d && 'data' in d) return d.data as T
  return d as unknown as T
}

// ==================== API 函数 ====================

export async function reviewL1(planId: string, req: ReviewDecisionRequest) {
  const resp = await apiClient.post(`/reviews/${planId}/l1`, req)
  return extractData<{ message: string }>(resp)
}

export async function reviewL2(planId: string, req: ReviewDecisionRequest) {
  const resp = await apiClient.post(`/reviews/${planId}/l2`, req)
  return extractData<{ message: string }>(resp)
}

export async function getReviewHistory(planId: string) {
  const resp = await apiClient.get(`/reviews/${planId}/history`)
  return extractData<ReviewHistoryResponse>(resp)
}

export async function getPendingReviews(params?: { limit?: number; offset?: number }) {
  const resp = await apiClient.get('/reviews/pending', { params })
  return extractData<PendingReviewListResponse>(resp)
}

export async function getReviewStats(level?: number) {
  const resp = await apiClient.get('/reviews/stats', { params: level ? { level } : {} })
  return extractData<ReviewStatsResponse>(resp)
}

/** 获取已审核记录列表（v127.2新增） */
export async function getReviewedRecords(params: { level: number; decision?: string; limit?: number; offset?: number }) {
  const resp = await apiClient.get('/reviews/reviewed', { params })
  return extractData<ReviewedListResponse>(resp)
}

export async function getReviewFlowConfig(schoolId: string) {
  const resp = await apiClient.get('/review-config', { params: { school_id: schoolId } })
  return extractData<ReviewFlowConfig>(resp)
}

export async function updateReviewFlowConfig(req: UpdateReviewFlowConfigRequest) {
  const resp = await apiClient.put('/review-config', req)
  return extractData<{ message: string }>(resp)
}

// ==================== 审核级别常量 ====================

export const REVIEW_LEVEL_NAMES: Record<number, string> = {
  0: '未提交',
  1: '教研组审核',
  2: '学校审核',
  3: '区域抽查',
}

export const REVIEW_LEVEL_COLORS: Record<number, string> = {
  0: '#9CA3AF',
  1: '#4F7BE8',
  2: '#F59E0B',
  3: '#8B5CF6',
}
