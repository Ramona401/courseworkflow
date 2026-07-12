/**
 * 课件工坊 API —— 协作层 (coursewares.collab.ts)（阶段2新建：页级批注）
 *
 * 页级批注：评审员/同组老师对课件某一页留意见，作者能看到、能标记已处理。
 * 镜像教案批注(annotations.ts)，挂载点从"段落号"换成"页码 page_number"。
 *
 * 权限(后端裁决，前端只管调用与展示)：
 *   - 写/读：能"看到"该课件的人(作者本人 / admin / 能看到该共享课件的同校同组成员)。
 *   - 删/标记：批注作者本人 / 课件作者本人 / admin。
 *
 * 经桶文件 coursewares.ts 透出，对外 import 路径不变(import { X } from '@/api/coursewares')。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'

// ==================== 类型 ====================

/** 课件页级批注单条（对应后端 CoursewareAnnotation） */
export interface CoursewareAnnotation {
  id: string
  courseware_id: string
  page_number: number      // 挂在第几页
  reviewer_id: string      // 批注人ID
  reviewer_name: string    // 批注人显示名
  content: string          // 批注内容
  status: string           // pending 待处理 / resolved 已处理 / archived 已归档(阶段3用)
  created_at: string
  updated_at: string
}

/** 批注列表响应（前端按 page_number 分组挂到胶片条对应页） */
export interface CWAnnotationListResponse {
  annotations: CoursewareAnnotation[]
  total: number
}

// ==================== API ====================

/** 列出某课件的全部批注（按页号→时间排序，前端自行分组） */
export async function listCWAnnotations(coursewareId: string): Promise<CWAnnotationListResponse> {
  const resp = await apiClient.get('/coursewares/' + coursewareId + '/annotations')
  return extractData(resp)
}

/** 在某课件某页创建一条批注 */
export async function createCWAnnotation(
  coursewareId: string,
  pageNumber: number,
  content: string,
): Promise<CoursewareAnnotation> {
  const resp = await apiClient.post(
    '/coursewares/' + coursewareId + '/annotations',
    { page_number: pageNumber, content },
  )
  return extractData(resp)
}

/**
 * 标记批注处理状态（resolved 已处理 / pending 重新待处理）。
 * 注意：端点是集合级 /coursewares/annotations/{aid}/resolve（不带课件ID，避免被当 courseware_id）。
 */
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

/**
 * 删除一条批注。
 * 端点同为集合级 /coursewares/annotations/{aid}（不带课件ID）。
 */
export async function deleteCWAnnotation(annotationId: string): Promise<{ message: string }> {
  const resp = await apiClient.delete('/coursewares/annotations/' + annotationId)
  return extractData(resp)
}
