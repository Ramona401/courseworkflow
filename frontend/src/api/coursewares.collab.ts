/**
 * 课件工坊 API —— 协作层
 *
 * 页级批注使用稳定page_id关联课件页面：
 *   - 页面重排后，批注继续跟随原页面；
 *   - page_number由后端解析为页面当前页码；
 *   - page_number_snapshot保留创建批注时的历史页码；
 *   - 页面被删除后page_id为null，批注历史仍然保留。
 *
 * page_id和page_number_snapshot暂时声明为可选，
 * 用于兼容前后端滚动部署期间尚未返回新字段的旧接口。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'

// ==================== 类型 ====================

/** 课件页级批注单条 */
export interface CoursewareAnnotation {
  id: string
  courseware_id: string

  /**
   * 稳定页面ID。
   *
   * string：页面仍存在；
   * null：原页面已删除；
   * undefined：滚动部署期间旧后端尚未返回该字段。
   */
  page_id?: string | null

  /**
   * 页面当前页码。
   *
   * 页面仍存在时由后端通过page_id动态解析；
   * 页面已删除时回退为创建时页码。
   */
  page_number: number

  /**
   * 创建批注时的历史页码。
   *
   * undefined仅用于兼容尚未返回该字段的旧后端。
   */
  page_number_snapshot?: number

  reviewer_id: string
  reviewer_name: string
  content: string
  status: string
  created_at: string
  updated_at: string
}

/** 批注列表响应 */
export interface CWAnnotationListResponse {
  annotations: CoursewareAnnotation[]
  total: number
}

// ==================== API ====================

/**
 * 列出某课件的全部批注。
 *
 * 后端按页面当前页码排序；
 * 页面已删除时按创建时页码快照排序。
 */
export async function listCWAnnotations(
  coursewareId: string,
): Promise<CWAnnotationListResponse> {
  const resp = await apiClient.get(
    '/coursewares/' + coursewareId + '/annotations',
  )
  return extractData(resp)
}

/**
 * 在当前页创建批注。
 *
 * 请求继续提交当前页码；后端在写入事务中解析并保存稳定page_id。
 */
export async function createCWAnnotation(
  coursewareId: string,
  pageNumber: number,
  content: string,
): Promise<CoursewareAnnotation> {
  const resp = await apiClient.post(
    '/coursewares/' + coursewareId + '/annotations',
    {
      page_number: pageNumber,
      content,
    },
  )
  return extractData(resp)
}

/** 标记批注已处理或重新待处理。 */
export async function resolveCWAnnotation(
  annotationId: string,
  status: 'resolved' | 'pending',
): Promise<{ message: string }> {
  const resp = await apiClient.put(
    '/coursewares/annotations/' + annotationId + '/resolve',
    { status },
  )
  return extractData(resp)
}

/** 删除一条批注。 */
export async function deleteCWAnnotation(
  annotationId: string,
): Promise<{ message: string }> {
  const resp = await apiClient.delete(
    '/coursewares/annotations/' + annotationId,
  )
  return extractData(resp)
}
