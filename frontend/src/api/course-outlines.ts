/**
 * course-outlines.ts — 课程大纲 API 封装
 *
 * 上下文16教育域规则：
 *   - K12列表和详情会返回publisher；
 *   - vocational/adult列表和详情不会下发publisher字段；
 *   - 非K12创建和更新普通课程大纲时，前端明确提交publisher: ''；
 *   - 出版社选择列表仅K12普通教学身份使用；
 *   - 教案挂载三态继续保持：
 *       null = 解除挂载；
 *       ''   = K12通用版，或非K12普通课程大纲；
 *       具名 = 仅K12教材版本。
 */

import apiClient from './client'

// ==================== 教材版本常量 ====================

/** 空字符串是数据库中的通用版本值。 */
export const COURSE_OUTLINE_PUBLISHER_GENERIC = ''

/** K12管理界面的通用版本展示文案。 */
export const COURSE_OUTLINE_PUBLISHER_GENERIC_LABEL =
  '通用 / 不限版本'

/**
 * K12常用教材版本快捷选项。
 *
 * 清单只用于前端录入便利，不是后端授权白名单。
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

/**
 * 把K12 publisher值转成展示文案。
 *
 * 非K12响应会省略publisher；调用方应先判断页面是否允许展示出版社，
 * 本函数只提供防御性空值兼容，不应被用于判断教育域。
 */
export function publisherLabel(
  publisher?: string | null,
): string {
  return !publisher
    ? COURSE_OUTLINE_PUBLISHER_GENERIC_LABEL
    : publisher
}

// ==================== 类型定义 ====================

/** 课程大纲归属层级。 */
export type CourseOutlineScope =
  | 'group'
  | 'school'
  | 'system'

/**
 * 大纲列表项。
 *
 * publisher仅在K12响应中存在。
 */
export interface CourseOutlineListItem {
  id: string
  scope: CourseOutlineScope
  scope_target_id: string
  scope_name: string
  subject: string
  grade: string
  volume: string
  publisher?: string
  title: string
  creator_name: string
  updated_at: string
}

/**
 * 大纲详情。
 *
 * publisher仅在K12响应中存在；非K12页面不得据缺省值展示
 * “通用/不限版本”徽章。
 */
export interface CourseOutlineDetail {
  id: string
  scope: CourseOutlineScope
  scope_target_id: string
  subject: string
  grade: string
  volume: string
  publisher?: string
  title: string
  content: string
  source_file_path: string
  source_type: string
  created_by: string
  status: string
  created_at: string
  updated_at: string
}

/** 列表响应。 */
export interface CourseOutlineListResponse {
  outlines: CourseOutlineListItem[]
  total: number
}

/**
 * 创建请求。
 *
 * 非K12调用方也必须显式传publisher: ''，避免沿用旧表单残值。
 */
export interface CreateCourseOutlineRequest {
  scope: CourseOutlineScope
  scope_target_id: string
  subject: string
  grade: string
  volume: string
  publisher: string
  title: string
  content: string
}

/**
 * 更新请求。
 *
 * 非K12调用方必须显式传publisher: ''。
 */
export interface UpdateCourseOutlineRequest {
  subject: string
  grade: string
  volume: string
  publisher: string
  title: string
  content: string
}

/** K12可用教材版本响应。 */
export interface AvailablePublishersResponse {
  publishers: string[]
  total: number
}

// ==================== API 函数 ====================

/** 列出当前用户同教育域且可见的课程大纲。 */
export async function getCourseOutlines():
Promise<CourseOutlineListResponse> {
  const { data } =
    await apiClient.get('/course-outlines')

  const result =
    data.data as CourseOutlineListResponse | undefined

  return {
    outlines: result?.outlines ?? [],
    total: result?.total ?? 0,
  }
}

/** 获取同教育域可见的大纲详情。 */
export async function getCourseOutline(
  id: string,
): Promise<CourseOutlineDetail> {
  const { data } =
    await apiClient.get(
      `/course-outlines/${id}`,
    )

  return data.data as CourseOutlineDetail
}

/** 创建课程大纲。 */
export async function createCourseOutline(
  request: CreateCourseOutlineRequest,
): Promise<CourseOutlineDetail> {
  const { data } =
    await apiClient.post(
      '/course-outlines',
      request,
    )

  return data.data as CourseOutlineDetail
}

/** 更新课程大纲。 */
export async function updateCourseOutline(
  id: string,
  request: UpdateCourseOutlineRequest,
): Promise<void> {
  await apiClient.put(
    `/course-outlines/${id}`,
    request,
  )
}

/** 软删除课程大纲。 */
export async function deleteCourseOutline(
  id: string,
): Promise<void> {
  await apiClient.delete(
    `/course-outlines/${id}`,
  )
}

/**
 * 查询K12学科和年级下可用的教材版本。
 *
 * vocational、adult、mixed或教育域异常时后端返回空数组。
 */
export async function getAvailablePublishers(
  subject: string,
  grade: string,
): Promise<string[]> {
  const { data } =
    await apiClient.get(
      '/course-outlines/publishers',
      {
        params: {
          subject,
          grade,
        },
      },
    )

  const result =
    data.data as
      | AvailablePublishersResponse
      | undefined

  return result?.publishers ?? []
}

/**
 * 设置或解除教案课程大纲挂载。
 *
 * publisher三态：
 *   - null：解除挂载；
 *   - ''：K12通用版，或非K12普通课程大纲；
 *   - 具名字符串：仅K12教材版本。
 */
export async function setLessonPlanCourseOutlinePublisher(
  planId: string,
  publisher: string | null,
): Promise<void> {
  await apiClient.put(
    `/lesson-plans/plans/${planId}/course-outline-publisher`,
    {
      course_outline_publisher:
        publisher,
    },
  )
}
