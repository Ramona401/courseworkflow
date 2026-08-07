/**
 * course-outlines.ts — 课程大纲API封装
 *
 * 教案课程大纲主链：
 *   - 正式身份始终是唯一course_outline_id；
 *   - exact模式用于自动匹配，只返回具体年级文本完全相等候选；
 *   - manual模式用于自动匹配失败后的教师选择，返回年级或学段相交候选；
 *   - 两种模式都由后端校验教育域、组织范围、active状态和学科；
 *   - 出版社字符串和publisher-only挂载仅保留旧客户端兼容。
 */

import apiClient from './client'

// ==================== 教材版本与学制常量 ====================

/** 空字符串是数据库中的通用教材版本值。 */
export const COURSE_OUTLINE_PUBLISHER_GENERIC = ''

/** K12管理界面的通用教材版本展示文案。 */
export const COURSE_OUTLINE_PUBLISHER_GENERIC_LABEL =
  '通用 / 不限版本'

/** K12常用教材版本快捷选项，仅用于录入便利。 */
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

/** K12课程大纲学制。 */
export type CourseOutlineSchoolSystem =
  | 'standard'
  | 'five_four'

export const COURSE_OUTLINE_SCHOOL_SYSTEM_STANDARD:
CourseOutlineSchoolSystem = 'standard'

export const COURSE_OUTLINE_SCHOOL_SYSTEM_FIVE_FOUR:
CourseOutlineSchoolSystem = 'five_four'

/** 课程大纲候选查询模式。 */
export type CourseOutlineCandidateMode =
  | 'exact'
  | 'manual'

/** 把publisher值转换成教师可读文案。 */
export function publisherLabel(
  publisher?: string | null,
): string {
  return !publisher
    ? COURSE_OUTLINE_PUBLISHER_GENERIC_LABEL
    : publisher
}

/** 把学制代码转换成教师可读文案。 */
export function schoolSystemLabel(
  schoolSystem?: CourseOutlineSchoolSystem | null,
): string {
  return schoolSystem ===
    COURSE_OUTLINE_SCHOOL_SYSTEM_FIVE_FOUR
    ? '五四制'
    : '普通学制'
}

// ==================== 类型定义 ====================

export type CourseOutlineScope =
  | 'group'
  | 'school'
  | 'system'

export interface CourseOutlineListItem {
  id: string
  scope: CourseOutlineScope
  scope_target_id: string
  scope_name: string
  subject: string
  grade: string
  volume: string
  publisher?: string
  school_system?: CourseOutlineSchoolSystem
  title: string
  creator_name: string
  updated_at: string
}

export interface CourseOutlineDetail {
  id: string
  scope: CourseOutlineScope
  scope_target_id: string
  subject: string
  grade: string
  volume: string
  publisher?: string
  school_system?: CourseOutlineSchoolSystem
  title: string
  content: string
  source_file_path: string
  source_type: string
  created_by: string
  status: string
  created_at: string
  updated_at: string
}

/**
 * 开始备课使用的课程大纲候选。
 *
 * id是唯一正式选择值，其余字段只用于展示和教师判断。
 */
export interface ExactCourseOutlineCandidate {
  id: string
  subject: string
  grade: string
  volume: string
  publisher: string
  school_system: CourseOutlineSchoolSystem
  title: string
  scope: CourseOutlineScope
  scope_name: string
  updated_at: string
}

export interface CourseOutlineListResponse {
  outlines: CourseOutlineListItem[]
  total: number
}

export interface CourseOutlineCandidatesResponse {
  candidates?: ExactCourseOutlineCandidate[]
  outlines?: ExactCourseOutlineCandidate[]
  total?: number
  match_mode?: CourseOutlineCandidateMode
}

export interface CreateCourseOutlineRequest {
  scope: CourseOutlineScope
  scope_target_id: string
  subject: string
  grade: string
  volume: string
  publisher: string
  school_system: CourseOutlineSchoolSystem
  title: string
  content: string
}

export interface UpdateCourseOutlineRequest {
  subject: string
  grade: string
  volume: string
  publisher: string
  school_system: CourseOutlineSchoolSystem
  title: string
  content: string
}

export interface LessonPlanCourseOutlineMountResult {
  message: string
  mounted: boolean
  course_outline_id: string | null
  course_outline_publisher: string | null
  course_outline_volume: string | null
  school_system: CourseOutlineSchoolSystem | null
}

export interface AvailablePublishersResponse {
  publishers: string[]
  total: number
}

// ==================== API函数 ====================

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
 * 按模式查询当前教师可绑定的课程大纲候选。
 *
 * exact：
 *   自动匹配候选，后端要求grade文本完全相等。
 *
 * manual：
 *   手动选择候选，后端允许当前年级和课程大纲学段存在交集。
 */
export async function getCourseOutlineCandidates(
  subject: string,
  grade: string,
  mode: CourseOutlineCandidateMode,
): Promise<ExactCourseOutlineCandidate[]> {
  const normalizedSubject = subject.trim()
  const normalizedGrade = grade.trim()

  if (
    !normalizedSubject ||
    !normalizedGrade
  ) {
    return []
  }

  const { data } =
    await apiClient.get(
      '/course-outlines/candidates',
      {
        params: {
          subject: normalizedSubject,
          grade: normalizedGrade,
          mode,
        },
      },
    )

  const result =
    data.data as
      | CourseOutlineCandidatesResponse
      | ExactCourseOutlineCandidate[]
      | undefined

  if (Array.isArray(result)) {
    return result
  }

  return result?.candidates ??
    result?.outlines ??
    []
}

/**
 * 查询自动匹配使用的具体年级精确候选。
 *
 * 保留原函数名，兼容尚未切换到双模式API的调用点。
 */
export async function getExactCourseOutlineCandidates(
  subject: string,
  grade: string,
): Promise<ExactCourseOutlineCandidate[]> {
  return getCourseOutlineCandidates(
    subject,
    grade,
    'exact',
  )
}

/** 查询自动匹配失败后的手动年级或学段候选。 */
export async function getManualCourseOutlineCandidates(
  subject: string,
  grade: string,
): Promise<ExactCourseOutlineCandidate[]> {
  return getCourseOutlineCandidates(
    subject,
    grade,
    'manual',
  )
}

/**
 * 设置、更换或解除教案的唯一课程大纲。
 *
 * courseOutlineId：
 *   - 非空唯一ID：设置或更换关联；
 *   - null：解除关联，并清除publisher-only旧残留。
 */
export async function updateLessonPlanCourseOutline(
  planId: string,
  courseOutlineId: string | null,
): Promise<LessonPlanCourseOutlineMountResult> {
  const { data } = await apiClient.put(
    `/lesson-plans/plans/${planId}/course-outline`,
    {
      course_outline_id: courseOutlineId,
    },
  )

  return data.data as LessonPlanCourseOutlineMountResult
}

/**
 * 查询K12具体年级下可用教材版本。
 *
 * @deprecated 只供旧publisher-only链兼容。
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
 * 设置或解除教案的publisher-only旧挂载。
 *
 * @deprecated 新教案与新页面必须使用唯一course_outline_id。
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
