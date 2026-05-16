/**
 * lesson-plan-interactions.ts — 教案互动（点赞/收藏）API 封装
 *
 * v125 新增：
 *   - toggleInteraction: 切换点赞/收藏状态（Toggle 模式）
 *   - getInteractions: 查询教案互动统计（计数+当前用户状态）
 *   - getMyFavorites: 查询我的收藏列表
 *
 * 互动类型：
 *   - 'like' — 点赞（可取消）
 *   - 'favorite' — 收藏（可取消）
 */
import apiClient from './client'

// ==================== 类型定义 ====================

/** 互动类型 */
export type InteractionType = 'like' | 'favorite'

/** 切换互动响应 */
export interface ToggleInteractionResponse {
  active: boolean    // true=已添加，false=已取消
  new_count: number  // 该类型的最新计数
}

/** 教案互动统计 */
export interface InteractionCounts {
  like_count: number
  favorite_count: number
  is_liked: boolean
  is_favorited: boolean
}

/** 收藏列表条目 */
export interface FavoriteListItem {
  interaction_id: string
  lesson_plan_id: string
  title: string
  subject: string
  grade: string
  topic: string
  author_name: string
  ai_review_score: number | null
  status: string
  status_name: string
  like_count: number
  favorite_count: number
  favorited_at: string
}

/** 收藏列表响应 */
export interface FavoriteListResponse {
  items: FavoriteListItem[]
  total: number
}

// ==================== API 函数 ====================

/**
 * 切换点赞/收藏状态
 * POST /api/v1/lesson-plans/plans/{planId}/interact
 *
 * Toggle 模式：已点赞→取消，未点赞→添加
 */
export async function toggleInteraction(
  planId: string,
  type: InteractionType
): Promise<ToggleInteractionResponse> {
  const res = await apiClient.post(
    `/lesson-plans/plans/${planId}/interact`,
    { interaction_type: type }
  )
  return res.data?.data ?? res.data
}

/**
 * 查询教案互动统计（计数+当前用户状态）
 * GET /api/v1/lesson-plans/plans/{planId}/interactions
 */
export async function getInteractions(
  planId: string
): Promise<InteractionCounts> {
  const res = await apiClient.get(
    `/lesson-plans/plans/${planId}/interactions`
  )
  return res.data?.data ?? res.data
}

/**
 * 查询我的收藏列表
 * GET /api/v1/lesson-plans/my-favorites
 */
export async function getMyFavorites(
  limit = 20,
  offset = 0
): Promise<FavoriteListResponse> {
  const res = await apiClient.get('/lesson-plans/my-favorites', {
    params: { limit, offset },
  })
  return res.data?.data ?? res.data
}
