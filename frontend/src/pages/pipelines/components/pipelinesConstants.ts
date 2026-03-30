/**
 * pipelinesConstants.ts — Pipeline列表页共用常量、类型、工具函数
 */

// ==================== 排序相关类型 ====================

/** 可排序字段 */
export type SortField = 'course_code' | 'status' | 'eval_avg_score' | 'meta_score' | 'translator_score' | 'progress' | 'created_at'

/** 排序方向 */
export type SortDirection = 'asc' | 'desc'

// ==================== 常量 ====================

/** 状态筛选按钮选项 */
export const FILTER_OPTIONS = [
  { label: '全部',       value: '' },
  { label: '运行中',     value: 'running' },
  { label: '待审核',     value: 'review_queue' },
  { label: '失败',       value: 'failed' },
  { label: '已完成',     value: 'finalized' },
  { label: '待启动',     value: 'pending' },
  { label: '已取消',     value: 'cancelled' },
  { label: '验收通过',   value: 'verified' },
  { label: '验收未通过', value: 'verify_failed' },
]

/**
 * 状态排序优先级
 * 运行中/待审核/失败排在前面，已完成排在后面
 */
export const STATUS_SORT_ORDER: Record<string, number> = {
  running: 1,
  review_queue: 2,
  needs_human: 3,
  pending_finalize: 4,
  failed: 5,
  pending: 6,
  finalized: 7,
  verified: 8,
  verify_failed: 9,
  cancelled: 10,
}

/** 每页条数选项 */
export const PAGE_SIZE_OPTIONS = [20, 50, 100]

// ==================== 工具函数 ====================

/**
 * 相对时间格式化
 * 刚刚 / N分钟前 / N小时前 / 月日时分
 */
export function formatTime(t: string | null): string {
  if (!t) return '-'
  const d = new Date(t)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  if (diff < 60000)    return '刚刚'
  if (diff < 3600000)  return Math.floor(diff / 60000) + '分钟前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前'
  return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

/**
 * 判断某个Pipeline是否可以快捷通过
 * 条件：状态在允许列表内 且 meta_score >= 9.0
 */
export function canMarkPassed(status: string, metaScore: number | null): boolean {
  const allowedStatuses = ['review_queue', 'needs_human', 'failed']
  return allowedStatuses.includes(status) && metaScore !== null && metaScore >= 9.0
}
