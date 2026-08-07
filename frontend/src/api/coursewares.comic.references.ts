/**
 * coursewares.comic.references.ts
 *
 * 知识点漫画可选参考资源浏览器协议。
 *
 * 安全边界：
 *   - 正式教材、已有课件和课程大纲只提交source_id；
 *   - 标题、正文和可见范围由后端重新读取；
 *   - 文档只提交浏览器提取出的纯文字，不上传原始二进制；
 *   - 图片只提交当前课件正式资产的asset_id；
 *   - GET响应不包含content_text和summary_text；
 *   - 本模块不承载IAOCI、图片提示词或模型内部数据。
 */

import apiClient from './client'
import {
  extractData,
} from './coursewares.types'

export type CoursewareComicReferenceType =
  | 'textbook_unit'
  | 'courseware'
  | 'course_outline'
  | 'uploaded_document'
  | 'uploaded_image'
  | 'other_text'

export interface CoursewareComicReferenceResource {
  id: string
  project_id: string
  courseware_id: string

  resource_type:
    CoursewareComicReferenceType

  source_id: string | null
  asset_id: string | null

  title: string
  file_name: string
  mime_type: string

  original_length: number
  summary_length: number
  sort_order: number

  image_url: string

  created_at: string | null
  updated_at: string | null
}

export interface CoursewareComicReferenceList {
  references:
    CoursewareComicReferenceResource[]
  total: number
}

export interface CreateCoursewareComicReferenceInput {
  resource_type:
    CoursewareComicReferenceType

  source_id?: string
  asset_id?: string

  title?: string
  file_name?: string
  mime_type?: string

  content_text?: string
  summary_text?: string

  sort_order: number
}

export interface DeleteCoursewareComicReferenceResult {
  id: string
  deleted: boolean
}

function requiredSegment(
  value: string,
  fieldName: string,
): string {
  const normalized =
    value.trim()

  if (!normalized) {
    throw new Error(
      `${fieldName}不能为空`,
    )
  }

  return encodeURIComponent(
    normalized,
  )
}

function referenceEndpoint(
  coursewareId: string,
  projectId: string,
): string {
  return (
    `/coursewares/${requiredSegment(
      coursewareId,
      '课件ID',
    )}` +
    `/comic-projects/${requiredSegment(
      projectId,
      '漫画项目ID',
    )}` +
    '/references'
  )
}

export async function listCoursewareComicReferences(
  coursewareId: string,
  projectId: string,
): Promise<CoursewareComicReferenceList> {
  const response =
    await apiClient.get(
      referenceEndpoint(
        coursewareId,
        projectId,
      ),
    )

  const data =
    extractData<CoursewareComicReferenceList>(
      response,
    )

  return {
    references:
      data.references || [],
    total:
      Number.isFinite(data.total)
        ? data.total
        : data.references?.length || 0,
  }
}

export async function createCoursewareComicReference(
  coursewareId: string,
  projectId: string,
  input: CreateCoursewareComicReferenceInput,
): Promise<CoursewareComicReferenceResource> {
  const response =
    await apiClient.post(
      referenceEndpoint(
        coursewareId,
        projectId,
      ),
      input,
    )

  return extractData<CoursewareComicReferenceResource>(
    response,
  )
}

export async function deleteCoursewareComicReference(
  coursewareId: string,
  projectId: string,
  referenceId: string,
): Promise<DeleteCoursewareComicReferenceResult> {
  const response =
    await apiClient.delete(
      `${referenceEndpoint(
        coursewareId,
        projectId,
      )}/${requiredSegment(
        referenceId,
        '参考资源ID',
      )}`,
    )

  return extractData<DeleteCoursewareComicReferenceResult>(
    response,
  )
}
