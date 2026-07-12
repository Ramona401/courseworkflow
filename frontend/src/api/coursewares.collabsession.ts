/**
 * 课件工坊 API —— 集体备课会话层 (coursewares.collabsession.ts)（阶段4新建）
 *
 * 集体备课（线下一群老师聚在一起，对着同一课件当场改、当场议）：
 *   系统只做最小三件事 —— ① 标记态 collab_state ② 共享微调权 ③ 留痕（走现有版本快照）。
 *   议课走现有页级批注（coursewares.collab.ts），不在本文件；本文件只管"标记态 + 参与者名单"。
 *
 * 注意命名：本文件叫 collabsession（集体备课会话），与装着"页级批注"的 coursewares.collab.ts 区分开
 *   （后者名字虽含 collab 但实为批注，历史命名遗留）。
 *
 * 权限（后端裁决，前端只管调用与展示）：
 *   - 发起/结束/加人/移人：仅课件作者。
 *   - 查状态：任何登录者（前端据 can_edit 决定显隐微调入口）。
 *
 * 经桶文件 coursewares.ts 透出，对外 import 路径不变（import { X } from '@/api/coursewares'）。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'

// ==================== 类型 ====================

/** 集体备课参与者（带显示名，对应后端 CollabMemberView） */
export interface CollabMemberView {
  id: string
  courseware_id: string
  user_id: string
  user_name: string        // 参与者显示名（display_name 优先，回退 username）
  initiator_id: string     // 本场集体备课发起者ID
  added_at: string         // 加入时间
}

/** 集体备课状态（对应后端 CollabStatusResponse） */
export interface CollabStatusResponse {
  courseware_id: string
  collab_state: string             // 'idle' 未集体备课 / 'in_session' 集体备课中
  owner_id: string                 // 课件作者ID（发起/加人/结束的权限主体）
  can_edit: boolean                // 当前登录者此刻能否微调本课件（作者/参与者，且非锁定态）
  members: CollabMemberView[] | null // 参与者列表（无人时后端返回 null）
  total: number                    // 参与者数量
}

/** 集体备课态常量（与后端 CWCollab* 对齐） */
export const CW_COLLAB_IDLE = 'idle'
export const CW_COLLAB_IN_SESSION = 'in_session'

/** 集体备课态中文名（前端徽章用） */
export const CW_COLLAB_STATE_NAME: Record<string, string> = {
  idle: '未集体备课',
  in_session: '集体备课中',
}

// ==================== API ====================

/** 查询某课件的集体备课状态（状态 + 参与者列表 + 我能否微调） */
export async function getCollabStatus(coursewareId: string): Promise<CollabStatusResponse> {
  const resp = await apiClient.get('/coursewares/' + coursewareId + '/collab')
  return extractData(resp)
}

/**
 * 发起集体备课（仅作者）。可选携带首批参与者用户ID数组。
 * members 为空则只发起、后续再逐个加人。
 */
export async function startCollab(
  coursewareId: string,
  members?: string[],
): Promise<{ message: string }> {
  const resp = await apiClient.post(
    '/coursewares/' + coursewareId + '/collab/start',
    { members: members || [] },
  )
  return extractData(resp)
}

/** 结束集体备课（仅作者）。课件回 idle、清空参与者名单。 */
export async function endCollab(coursewareId: string): Promise<{ message: string }> {
  const resp = await apiClient.post('/coursewares/' + coursewareId + '/collab/end')
  return extractData(resp)
}

/** 加一名参与者（仅作者）。 */
export async function addCollabMember(
  coursewareId: string,
  userId: string,
): Promise<{ message: string }> {
  const resp = await apiClient.post(
    '/coursewares/' + coursewareId + '/collab/members',
    { user_id: userId },
  )
  return extractData(resp)
}

/** 移除一名参与者（仅作者）。 */
export async function removeCollabMember(
  coursewareId: string,
  userId: string,
): Promise<{ message: string }> {
  const resp = await apiClient.delete(
    '/coursewares/' + coursewareId + '/collab/members/' + userId,
  )
  return extractData(resp)
}

// ==================== 候选成员（加参与者下拉选人）====================

/** 集体备课候选成员（同校同组老师，对应后端 CollabCandidate） */
export interface CollabCandidate {
  user_id: string
  username: string
  display_name: string
  role: string
}

/** 候选成员列表响应 */
export interface CollabCandidateListResponse {
  candidates: CollabCandidate[]
  total: number
}

/**
 * 列出当前用户可拉入集体备课的候选成员（同校 + 同教研组，已排除自己/admin/viewer）。
 * 集合级端点（不带课件ID）：GET /coursewares/collab/candidates
 */
export async function listCollabCandidates(): Promise<CollabCandidateListResponse> {
  const resp = await apiClient.get('/coursewares/collab/candidates')
  return extractData(resp)
}

// ==================== 我参与的集体备课（参与者入口）====================

/** "我参与的集体备课"列表单条（对应后端 JoinedCollabItem） */
export interface JoinedCollabItem {
  id: string           // 课件ID
  title: string        // 课件标题
  subject: string      // 学科
  grade: string        // 年级
  status: string       // 生产态
  status_name: string  // 生产态中文名
  page_count: number   // 页数
  owner_id: string     // 课件作者ID
  owner_name: string   // 课件作者显示名（这是谁的课件）
  collab_state: string // 集体备课态（恒为 in_session）
  joined_at: string    // 我被加入的时间
  updated_at: string   // 课件更新时间
}

/** "我参与的集体备课"列表响应 */
export interface JoinedCollabListResponse {
  coursewares: JoinedCollabItem[] | null
  total: number
}

/**
 * 列出我作为参与者被拉入、且仍在进行中的集体备课课件。
 * 集合级端点：GET /coursewares/collab/joined
 * 供参与者（非作者）在自己界面找到并进入被邀请的课件。
 */
export async function listJoinedCollab(): Promise<JoinedCollabListResponse> {
  const resp = await apiClient.get('/coursewares/collab/joined')
  return extractData(resp)
}
