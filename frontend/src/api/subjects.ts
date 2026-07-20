/**
 * subjects.ts — 统一课程定义与分域课程目录API
 *
 * 公开接口：
 *   GET /api/v1/subjects
 *
 *   后端根据当前用户的教育域和教学组织返回：
 *     - 当前教育域公共课程；
 *     - 当前学校专属课程。
 *
 * 管理接口：
 *   /api/v1/admin/subjects
 *
 *   后台管理员维护两层数据：
 *     1. subjects：全平台统一课程定义；
 *     2. subject_catalog_entries：教育域和学校可见目录。
 *
 * 新增课程时必须同时提交至少一条目录配置，
 * 避免产生“课程定义已创建，但教师下拉不可见”的孤立课程。
 */

import apiClient from './client'
import type { ApiResponse } from './client'
import type { SubjectItem } from '@/constants/subjects'
import type {
  EducationDomain,
} from '@/education-domain/types'

export type { SubjectItem }

/* ==================== 公开课程目录 ==================== */

export interface SubjectCatalogResult {
  subjects: SubjectItem[]
  total: number
  education_domain: EducationDomain
  education_org_id: string
}

/**
 * 获取当前用户的分域课程目录和后端解析出的教育上下文。
 */
export async function getSubjectCatalog():
Promise<SubjectCatalogResult> {
  const res = await apiClient.get<
    ApiResponse<SubjectCatalogResult>
  >('/subjects')

  return res.data.data || {
    subjects: [],
    total: 0,
    education_domain: 'k12',
    education_org_id: '',
  }
}

/**
 * 兼容现有调用点，只返回课程定义数组。
 */
export async function getSubjects():
Promise<SubjectItem[]> {
  const result = await getSubjectCatalog()

  return result.subjects || []
}

/* ==================== 管理端目录模型 ==================== */

/**
 * 可承载具体教学资源的教育域。
 *
 * mixed只用于跨域管理页面，不允许成为课程目录归属。
 */
export type TeachingEducationDomain =
  | 'k12'
  | 'vocational'
  | 'adult'

/**
 * 一条课程目录归属。
 *
 * organization_id为空表示该教育域公共课程；
 * 非空表示仅指定学校可见。
 */
export interface SubjectCatalogEntry {
  id: string
  subject_id: string

  education_domain: TeachingEducationDomain

  organization_id: string | null
  organization_name: string

  display_name: string
  sort_order: number
  is_active: boolean

  created_at: string
  updated_at: string
}

/**
 * 后台课程管理列表项。
 *
 * 保留SubjectItem全部统一定义字段，
 * 并增加该课程的全部教育域和学校目录配置。
 */
export interface SubjectAdminItem
extends SubjectItem {
  catalog_entries: SubjectCatalogEntry[]
}

/**
 * 新增或编辑时提交的一条目录配置。
 */
export interface SubjectCatalogEntryRequest {
  education_domain: TeachingEducationDomain

  /**
   * null表示教育域公共课程；
   * 具体UUID表示指定学校专属课程。
   */
  organization_id: string | null

  /**
   * 目录展示名。
   * 留空时后端自动使用统一课程名称。
   */
  display_name: string

  /**
   * 域内排序。
   * 小于等于0时后端使用课程基础排序。
   */
  sort_order: number

  is_active: boolean
}

/* ==================== 管理端写入请求 ==================== */

export interface CreateSubjectRequest {
  name: string
  code?: string
  sort_order?: number
  note?: string

  /**
   * 新建课程必须同时配置至少一条可见目录。
   */
  catalog_entries: SubjectCatalogEntryRequest[]
}

export interface UpdateSubjectRequest {
  name?: string
  code?: string
  sort_order?: number
  is_active?: boolean
  note?: string

  /**
   * 不提交表示保留原目录；
   * 提交数组表示使用该数组完整替换原目录。
   */
  catalog_entries?: SubjectCatalogEntryRequest[]
}

/* ==================== 管理端接口 ==================== */

/**
 * admin：获取全部统一课程定义及其目录配置。
 */
export async function getAllSubjects():
Promise<SubjectAdminItem[]> {
  const res = await apiClient.get<
    ApiResponse<{
      subjects: SubjectAdminItem[]
      total: number
    }>
  >('/admin/subjects')

  return res.data.data?.subjects || []
}

/**
 * admin：在一个后端事务中创建课程定义和目录配置。
 */
export async function createSubject(
  req: CreateSubjectRequest,
): Promise<SubjectAdminItem> {
  const res = await apiClient.post<
    ApiResponse<SubjectAdminItem>
  >('/admin/subjects', req)

  return res.data.data!
}

/**
 * admin：编辑课程定义，并可完整替换目录配置。
 */
export async function updateSubject(
  id: string,
  req: UpdateSubjectRequest,
): Promise<SubjectAdminItem> {
  const res = await apiClient.put<
    ApiResponse<SubjectAdminItem>
  >(
    `/admin/subjects/${id}`,
    req,
  )

  return res.data.data!
}

/**
 * admin：删除非内置课程及其目录配置。
 */
export async function deleteSubject(
  id: string,
): Promise<void> {
  await apiClient.delete(
    `/admin/subjects/${id}`,
  )
}
