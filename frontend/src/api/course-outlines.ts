/**
 * course-outlines.ts — 课程大纲 API 封装（大单元备课能力·批次一 + 教材版本增强）
 *
 * 对应后端：
 *   - handlers/course_outline_handler.go
 *   - services/course_outline_service.go
 *   - routes/routes_course_outline.go
 *
 * 接口：
 *   GET    /api/v1/course-outlines              列出可见大纲（全员；全局 system 人人可见）
 *   POST   /api/v1/course-outlines              创建（组长/校管/admin；system 仅 admin）
 *   GET    /api/v1/course-outlines/publishers   查某学科+年级可选教材版本（备课首屏选择器用）★新增
 *   GET    /api/v1/course-outlines/{id}         单条详情（含正文）
 *   PUT    /api/v1/course-outlines/{id}         更新
 *   DELETE /api/v1/course-outlines/{id}         软删除
 *
 * 教材版本(publisher)：一标多本，空串=通用/不限版本。CRUD 全程透传；
 * 备课首屏据 getAvailablePublishers 列出该学科年级可选版本，选定后经
 * setLessonPlanCourseOutlinePublisher 写到教案上，注入层据此精确匹配（零跨版本兜底）。
 *
 * 归属下拉数据复用 ai-assistants 的 getMyPublishGroups()。
 * 响应拦截器已处理 code!==0 抛错，本文件直接取 data.data。
 */
import apiClient from './client'

// ==================== 教材版本常量 ====================

/** 通用/不限版本：publisher 为空串时的语义。空串大纲只被"选了通用版"的教案精确命中 */
export const COURSE_OUTLINE_PUBLISHER_GENERIC = ''

/** 通用版在下拉里展示的中文名（空串 → 显示此文案） */
export const COURSE_OUTLINE_PUBLISHER_GENERIC_LABEL = '通用 / 不限版本'

/**
 * 预置教材版本清单（与后端 models.CourseOutlinePublishers 保持同步）。
 * 仅为常用内置选项，非穷尽——管理页下拉允许手动输入新版本名。
 * 后期增删常用版本：前后端两处常量同步修改。
 */
export const COURSE_OUTLINE_PUBLISHERS: string[] = [
  '人教版',
  '统编版',
  '北师大版',
  '苏教版',
  '外研版',
  'PEP人教版',
  '鄂教版',
  '沪教版',
  '湘教版',
  '青岛版',
]

/** 把 publisher 值转成展示文案（空串 → "通用 / 不限版本"） */
export function publisherLabel(publisher: string): string {
  return publisher === COURSE_OUTLINE_PUBLISHER_GENERIC
    ? COURSE_OUTLINE_PUBLISHER_GENERIC_LABEL
    : publisher
}

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
  publisher: string       // 教材版本（空串=通用/不限版本）
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
  publisher: string       // 教材版本（空串=通用/不限版本）
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
  publisher: string        // 教材版本（空串=通用/不限版本）
  title: string
  content: string
}

/** 更新请求 */
export interface UpdateCourseOutlineRequest {
  subject: string
  grade: string
  volume: string
  publisher: string        // 教材版本（空串=通用/不限版本）
  title: string
  content: string
}

/** 可用版本响应（备课首屏选择器用） */
export interface AvailablePublishersResponse {
  publishers: string[]     // 该学科年级真实存在大纲的版本列表；空串元素=通用版；空数组=无大纲
  total: number
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

/**
 * 查某学科+年级下可选的教材版本列表（备课首屏教材版本选择器用）。
 * 返回 publishers：该学科年级真实存在、且学段相交的大纲所拥有的版本（去重）；
 *   - 数组元素含空串("")时代表"通用/不限版本"；
 *   - 返回空数组 = 该学科年级没有任何大纲 → 前端不显示版本选择、不关联大纲。
 */
export async function getAvailablePublishers(subject: string, grade: string): Promise<string[]> {
  const { data } = await apiClient.get('/course-outlines/publishers', {
    params: { subject, grade },
  })
  const resp = data.data as AvailablePublishersResponse
  return resp?.publishers ?? []
}

/**
 * 设置/解除教案选定的课程大纲教材版本（备课首屏选定后调用）。
 *
 * 三态（与后端 *string 语义对齐）：
 *   - publisher = null  → 解除关联（教案不注入大纲）
 *   - publisher = ''    → 选"通用/不限版本"（只注入 publisher 为空串的大纲）
 *   - publisher = '人教版' → 选具名版本（只注入该版本大纲，零跨版本兜底）
 *
 * 对应 PUT /api/v1/lesson-plans/plans/{planId}/course-outline-publisher
 */
export async function setLessonPlanCourseOutlinePublisher(
  planId: string,
  publisher: string | null,
): Promise<void> {
  await apiClient.put(`/lesson-plans/plans/${planId}/course-outline-publisher`, {
    course_outline_publisher: publisher,
  })
}
