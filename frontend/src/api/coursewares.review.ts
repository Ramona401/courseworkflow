/**
 * 课件工坊 API —— 多级审核层 (coursewares.review.ts)（阶段3新建：L1教研组 / L2学校审核）
 *
 * 镜像教案审核 review-v2.ts 的封装范式，但有三处本质差异，务必理解：
 *
 *  1. 审核主键是【课件ID】(courseware_id)，不是教案ID。所有审核端点路径用课件ID。
 *
 *  2. 课件审核态承载在 coursewares.publish_state（与 status 生产状态机正交），
 *     不像教案靠 status。前端不直接读写该字段，只调审核 API，状态机由后端维护：
 *       提交审核   → publish_state=submitted, review_level=0
 *       L1通过(无L2) → publish_state=approved,  review_level=1（待发布，作者再走发布面板共享）
 *       L1通过(有L2) → publish_state=submitted, review_level=1（进入L2待审核）
 *       L2通过      → publish_state=approved,  review_level=2
 *       退回        → publish_state=revision,  review_level=0（可回流重提）
 *
 *  3. 课件【无 L3 区域抽查】，只有 L1/L2 两级。审核级别常量复用教案 REVIEW_LEVEL_*。
 *
 * 路由前缀 /api/v1/courseware-reviews/ 与教案 /api/v1/reviews/ 物理隔离。
 * 「提交审核」端点是 POST /api/v1/coursewares/{id}/submit-review（挂在课件子路由，作者发起）。
 *
 * 经桶文件 coursewares.ts 透出，对外 import 路径不变(import { X } from '@/api/coursewares')。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'
import type { CoursewareDetail } from './coursewares.types'
import type { CoursewareAnnotation } from './coursewares.collab'

// ==================== 类型定义 ====================

/** 审核决策请求（审核员 L1/L2 操作） */
export interface CWReviewDecisionRequest {
  decision: 'approved' | 'revision'
  score?: number
  comment: string
  dimensions?: string
}

/** 审核记录列表项（审核历史用，含审核员名） */
export interface CWReviewListItem {
  id: string
  courseware_id: string
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

/** 审核历史响应 */
export interface CWReviewHistoryResponse {
  reviews: CWReviewListItem[]
  total: number
  current_level: number   // 课件当前审核层级进度（coursewares.review_level）
}

/** 待审核列表项 */
export interface CWPendingReviewItem {
  courseware_id: string
  title: string
  subject: string
  grade: string
  page_count: number
  source_type: string
  source_name: string
  author_id: string
  author_name: string
  school_name: string
  review_level: number
  level_name: string
  submitted_at: string
}

/** 待审核列表响应 */
export interface CWPendingReviewListResponse {
  items: CWPendingReviewItem[]
  total: number
}

/** 审核统计响应 */
export interface CWReviewStatsResponse {
  total_pending: number
  total_reviewed: number
  total_approved: number
  total_revision: number
}

/** 已审核记录列表项 */
export interface CWReviewedListItem {
  id: string
  courseware_id: string
  courseware_title: string
  subject: string
  grade: string
  author_name: string
  review_level: number
  level_name: string
  reviewer_name: string
  decision: string
  score: number | null
  comment: string
  created_at: string
}

/** 已审核记录列表响应 */
export interface CWReviewedListResponse {
  items: CWReviewedListItem[]
  total: number
}

/**
 * 审核详情响应（决策二核心：审核员边看课件+批注边决策）。
 * 一次返：课件详情（含 pages）+ 该课件全部批注 + 历史审核记录。
 */
export interface CWReviewDetailResponse {
  courseware: CoursewareDetail
  annotations: CoursewareAnnotation[]
  reviews: CWReviewListItem[]
}

// ==================== API 函数 ====================

/**
 * 提交课件审核（作者发起）。
 * 课件须 status≥preview（已生成），publish_state∈{private,published_personal,revision}。
 * 后端反查作者学校写 review_school_id，置 publish_state=submitted/review_level=0。
 * 接口：POST /api/v1/coursewares/{id}/submit-review
 */
export async function submitCoursewareForReview(coursewareId: string): Promise<{ message: string }> {
  const resp = await apiClient.post(`/coursewares/${coursewareId}/submit-review`, {})
  return extractData(resp)
}

/** L1 教研组审核。接口：POST /api/v1/courseware-reviews/{id}/l1 */
export async function reviewCWL1(coursewareId: string, req: CWReviewDecisionRequest) {
  const resp = await apiClient.post(`/courseware-reviews/${coursewareId}/l1`, req)
  return extractData<{ message: string }>(resp)
}

/** L2 学校审核。接口：POST /api/v1/courseware-reviews/{id}/l2 */
export async function reviewCWL2(coursewareId: string, req: CWReviewDecisionRequest) {
  const resp = await apiClient.post(`/courseware-reviews/${coursewareId}/l2`, req)
  return extractData<{ message: string }>(resp)
}

/** 审核历史。接口：GET /api/v1/courseware-reviews/{id}/history */
export async function getCWReviewHistory(coursewareId: string) {
  const resp = await apiClient.get(`/courseware-reviews/${coursewareId}/history`)
  return extractData<CWReviewHistoryResponse>(resp)
}

/**
 * 审核详情（决策二：课件+全部批注+历史）。
 * 接口：GET /api/v1/courseware-reviews/{id}/detail
 */
export async function getCWReviewDetail(coursewareId: string) {
  const resp = await apiClient.get(`/courseware-reviews/${coursewareId}/detail`)
  return extractData<CWReviewDetailResponse>(resp)
}

/** 待审核列表（后端按角色分流：operator/viewer→L1，senior→L1+L2，admin→全部）。 */
export async function getCWPendingReviews(params?: { limit?: number; offset?: number }) {
  const resp = await apiClient.get('/courseware-reviews/pending', { params })
  return extractData<CWPendingReviewListResponse>(resp)
}

/** 已审核记录列表。参数：level(1/2)，decision(approved/revision/空=全部)。 */
export async function getCWReviewedRecords(params: {
  level: number; decision?: string; limit?: number; offset?: number
}) {
  const resp = await apiClient.get('/courseware-reviews/reviewed', { params })
  return extractData<CWReviewedListResponse>(resp)
}

/** 审核统计。参数：level(1/2)。接口：GET /api/v1/courseware-reviews/stats */
export async function getCWReviewStats(level?: number) {
  const resp = await apiClient.get('/courseware-reviews/stats', { params: level ? { level } : {} })
  return extractData<CWReviewStatsResponse>(resp)
}

// ==================== 审核级别常量（课件只有 L1/L2，无 L3） ====================

export const CW_REVIEW_LEVEL_NAMES: Record<number, string> = {
  0: '未提交',
  1: '教研组审核',
  2: '学校审核',
}

// 课件工坊暖色系：L1 橙、L2 红，与教案审核蓝紫系区分。
export const CW_REVIEW_LEVEL_COLORS: Record<number, string> = {
  0: '#9CA3AF',
  1: '#F59E0B',
  2: '#EF4444',
}
