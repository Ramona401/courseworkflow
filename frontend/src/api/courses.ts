/**
 * 课程管理API封装
 * P3-3: 课程列表 + 注册 + OSS目录 + 索引拉取
 * FE-CS-01修复：消除全部 (res.data as any).data 类型绕过，改用 extractData 辅助函数
 */
import client from './client'
import type { ApiResponse } from './client'
import type { AxiosResponse } from 'axios'

// ==================== 辅助函数 ====================

/**
 * extractData 从 Axios 响应中安全提取业务数据
 * FE-CS-01修复：替代全部 (res.data as any).data 写法，与 pipelines.ts 保持一致
 */
function extractData<T>(res: AxiosResponse<ApiResponse<T>>): T {
  return res.data.data as T
}

// ==================== 类型定义 ====================

/** 课程列表单条 */
export interface CourseListItem {
  id: string
  course_code: string
  course_name: string
  external_module_id: number | null
  grade_num: number | null
  stage: string
  semester: string
  status: string
  has_index: boolean
  index_page_count: number
  index_total_length: number
  index_fetched_at: string | null
  created_at: string
  updated_at: string
}

/** 课程列表响应 */
export interface CourseListResponse {
  courses: CourseListItem[]
  total: number
}

/** 注册课程请求 */
export interface CreateCourseRequest {
  external_module_id: number
  course_code: string
  course_name?: string
  grade_num?: number
  stage?: string
  semester?: string
}

/** OSS模块列表项 */
export interface OSSModuleItem {
  id: number
  name: string
  lesson_count: number
  status: number
  is_registered: boolean
  course_code: string
  has_index: boolean
}

/** OSS目录响应 */
export interface OSSCatalogResponse {
  version: number | string
  total_modules: number
  total_lessons: number
  modules: OSSModuleItem[]
  generated_at: string
}

/** 索引拉取结果 */
export interface FetchIndexResult {
  course_code: string
  page_count: number
  total_length: number
  index_hash: string
  fetched_at: string
  message: string
}

/** 索引摘要 */
export interface IndexSummary {
  course_code: string
  course_name: string
  page_count: number
  total_length: number
  page_titles: string[] | null
  has_index: boolean
}

// ==================== API方法 ====================

/** 获取课程列表 */
export async function getCourses(): Promise<CourseListResponse> {
  const res = await client.get<ApiResponse<CourseListResponse>>('/courses')
  return extractData<CourseListResponse>(res)
}

/** 注册新课程 */
export async function createCourse(req: CreateCourseRequest): Promise<CourseListItem> {
  const res = await client.post<ApiResponse<CourseListItem>>('/courses', req)
  return extractData<CourseListItem>(res)
}

/** 获取OSS目录（含注册状态） */
export async function getOSSCatalog(): Promise<OSSCatalogResponse> {
  const res = await client.get<ApiResponse<OSSCatalogResponse>>('/courses/oss-catalog')
  return extractData<OSSCatalogResponse>(res)
}

/** 从OSS拉取课程索引 */
export async function fetchCourseIndex(courseCode: string): Promise<FetchIndexResult> {
  const res = await client.post<ApiResponse<FetchIndexResult>>('/courses/' + courseCode + '/fetch-index')
  return extractData<FetchIndexResult>(res)
}

/** 获取索引摘要 */
export async function getIndexSummary(courseCode: string): Promise<IndexSummary> {
  const res = await client.get<ApiResponse<IndexSummary>>('/courses/' + courseCode + '/index-summary')
  return extractData<IndexSummary>(res)
}
