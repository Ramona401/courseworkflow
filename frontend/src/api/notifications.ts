/**
 * 通知中心 API —— 站内信（阶段5c 前端）
 *
 * 对接后端通用通知中心四端点（baseURL=/api/v1，token 由 client 拦截器自动注入）：
 *   GET  /notifications?unread_only=&limit=&offset=   通知列表（含未读总数）
 *   GET  /notifications/unread-count                  未读数（顶栏红点轮询，极轻）
 *   PUT  /notifications/{id}/read                     标单条已读
 *   PUT  /notifications/read-all                      全部标已读
 *
 * 通知是只读消费型数据：前端不创建通知（通知由各业务模块在后端旁路发出），
 * 只做"拉列表 / 查未读数 / 标已读"。被 NotificationBell.tsx 组件消费。
 *
 * 响应解包：client 响应拦截器已拦掉 code!==0 的情况，故这里拿到的 resp 必为成功，
 * 业务数据在 resp.data.data。本模块自带极简 unwrap，不依赖课件专用的 coursewares.types。
 */
import apiClient from './client'

// ==================== 类型 ====================

/** 单条通知（对应后端 models.Notification） */
export interface Notification {
  id: string
  recipient_id: string
  type: string          // 事件类型：cw_collab_invited/removed/ended、cw_review_submitted/approved/revision ...
  title: string         // 一句话标题
  body: string          // 详情（可空；如审核退回时为审核意见）
  entity_type: string   // 业务对象类型：courseware/lesson_plan/inspection
  entity_id: string     // 业务对象ID（软关联，可空）
  actor_id: string      // 触发人ID（可空）
  actor_name: string    // 触发人显示名（冗余，可空）
  link: string          // 前端点击跳转路径（如 /courseware/{id} 或 /courseware/review）
  is_read: boolean
  read_at: string | null
  created_at: string
}

/** 通知列表响应 */
export interface NotificationListResponse {
  notifications: Notification[]
  total: number
  unread_count: number  // 顺带回未读总数，供前端刷红点
}

// ==================== 内部辅助 ====================

/**
 * 极简响应解包：取 resp.data.data。
 * client 拦截器已保证 code===0（否则走 reject），故无需再判 code。
 * data 缺失时按调用处期望给出安全默认（由各函数兜底）。
 */
function unwrap<T>(resp: { data?: { data?: T } }): T | undefined {
  return resp?.data?.data
}

// ==================== API ====================

/**
 * 拉通知列表。
 * @param limit      每页条数（默认 20）
 * @param offset     偏移（默认 0）
 * @param unreadOnly 仅未读（默认 false=全部）
 * 返回非空结构（列表兜底空数组，防组件 .map 崩）。
 */
export async function listNotifications(
  limit = 20,
  offset = 0,
  unreadOnly = false,
): Promise<NotificationListResponse> {
  const params: Record<string, string | number | boolean> = { limit, offset }
  if (unreadOnly) params.unread_only = true
  const resp = await apiClient.get('/notifications', { params })
  const data = unwrap<NotificationListResponse>(resp)
  return {
    notifications: data?.notifications ?? [],
    total: data?.total ?? 0,
    unread_count: data?.unread_count ?? 0,
  }
}

/**
 * 查未读数（顶栏红点轮询专用，极轻）。
 * 失败由调用方 catch（轮询里静默吞掉，不打扰用户）。
 */
export async function getUnreadCount(): Promise<number> {
  const resp = await apiClient.get('/notifications/unread-count')
  const data = unwrap<{ unread_count: number }>(resp)
  return data?.unread_count ?? 0
}

/** 标单条已读。 */
export async function markRead(notificationId: string): Promise<void> {
  await apiClient.put('/notifications/' + notificationId + '/read')
}

/** 全部标已读，返回受影响条数。 */
export async function markAllRead(): Promise<number> {
  const resp = await apiClient.put('/notifications/read-all')
  const data = unwrap<{ marked: number }>(resp)
  return data?.marked ?? 0
}
