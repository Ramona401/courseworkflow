/**
 * course-outlines.ts — 课程大纲 API 封装（大单元备课能力·批次一）
 *
 * 对应后端：
 *   - handlers/course_outline_handler.go
 *   - services/course_outline_service.go
 *   - routes/routes_course_outline.go
 *
 * 接口：
 *   GET    /api/v1/course-outlines        列出可见大纲（全员；全局 system 人人可见）
 *   POST   /api/v1/course-outlines        创建（组长/校管/admin；system 仅 admin）
 *   GET    /api/v1/course-outlines/{id}   单条详情（含正文）
 *   PUT    /api/v1/course-outlines/{id}   更新
 *   DELETE /api/v1/course-outlines/{id}   软删除
 *
 * 归属下拉数据复用 ai-assistants 的 getMyPublishGroups()（同一套"我能发布的教研组"）；
 * 全局(system)选项仅 admin 在页面侧本地追加，target 留空由后端填占位ID。
 * 响应拦截器已处理 code!==0 抛错，本文件直接取 data.data。
 */
import apiClient from './client'

// ==================== 类型定义 ====================

/** 大纲归属层级（group 教研组 / school 学校 / system 全局，admin 录入所有学校通用） */
export type CourseOutlineScope = 'group' | 'school' | 'system'

/** 大纲列表项（管理界面用，不含正文） */
export interface CourseOutlineListItem {
  id: string
  scope: CourseOutlineScope
  scope_target_id: string
  scope_name: string      // 教研组名 / 学校名 / "全局（所有学校通用）"（后端回填）
  subject: string
  grade: string
  volume: string
  title: string
  creator_name: string
  updated_at: string
}

/** 大纲详情（含原文整块 content） */
export interface CourseOutlineDetail {
  id: string
  scope: CourseOutlineScope
  scope_target_id: string
  subject: string
  grade: string
  volume: string
  title: string
  content: string         // 原文整块
  source_file_path: string
  source_type: string
  created_by: string
  status: string
  created_at: string
  updated_at: string
}

/** 列表响应 */
export interface CourseOutlineListResponse {
  outlines: CourseOutlineListItem[]
  total: number
}

/** 创建请求 */
export interface CreateCourseOutlineRequest {
  scope: CourseOutlineScope
  scope_target_id: string  // system 可传空串，后端填占位ID
  subject: string
  grade: string
  volume: string
  title: string
  content: string
}

/** 更新请求 */
export interface UpdateCourseOutlineRequest {
  subject: string
  grade: string
  volume: string
  title: string
  content: string
}

// ==================== API 函数 ====================

/** 列出可见的课程大纲 */
export async function getCourseOutlines(): Promise<CourseOutlineListResponse> {
  const { data } = await apiClient.get('/course-outlines')
  return data.data as CourseOutlineListResponse
}

/** 获取大纲详情（含正文，用于编辑/查看） */
export async function getCourseOutline(id: string): Promise<CourseOutlineDetail> {
  const { data } = await apiClient.get(`/course-outlines/${id}`)
  return data.data as CourseOutlineDetail
}

/** 创建大纲 */
export async function createCourseOutline(req: CreateCourseOutlineRequest): Promise<CourseOutlineDetail> {
  const { data } = await apiClient.post('/course-outlines', req)
  return data.data as CourseOutlineDetail
}

/** 更新大纲 */
export async function updateCourseOutline(id: string, req: UpdateCourseOutlineRequest): Promise<void> {
  await apiClient.put(`/course-outlines/${id}`, req)
}

/** 删除大纲（软删除） */
export async function deleteCourseOutline(id: string): Promise<void> {
  await apiClient.delete(`/course-outlines/${id}`)
}
