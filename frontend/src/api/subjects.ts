/**
 * subjects.ts — 学科字典 API 封装
 *
 * 对应后端：
 *   - handlers/subject_handler.go
 *   - repository/subject_repo.go
 *   - routes/routes_subject.go
 *
 * 接口：
 *   公开只读（登录即可，前端各下拉消费）：
 *     GET    /api/v1/subjects              启用学科列表（按 sort_order 排）
 *   管理 CRUD（admin）：
 *     GET    /api/v1/admin/subjects        全部学科（含停用）
 *     POST   /api/v1/admin/subjects        新建
 *     PUT    /api/v1/admin/subjects/{id}   编辑
 *     DELETE /api/v1/admin/subjects/{id}   删除（内置学科禁删，后端拦截）
 *
 * 响应拦截器已处理 code!==0 抛错，本文件直接取 data.data。
 */
import apiClient from './client'
import type { SubjectItem } from '@/constants/subjects'

export type { SubjectItem }

/** 后端列表响应 { subjects, total } */
interface SubjectListResp {
  subjects: SubjectItem[]
  total: number
}

/** 新建学科请求体 */
export interface CreateSubjectRequest {
  name: string
  code?: string
  sort_order?: number
  note?: string
}

/** 编辑学科请求体（仅传要改的字段） */
export interface UpdateSubjectRequest {
  name?: string
  code?: string
  sort_order?: number
  is_active?: boolean
  note?: string
}

/** 公开：获取启用学科列表（前端各下拉消费；useSubjects 内部调用） */
export async function getSubjects(): Promise<SubjectItem[]> {
  const res = await apiClient.get<{ data: SubjectListResp }>('/subjects')
  return res.data.data?.subjects || []
}

/** admin：获取全部学科（含停用），供后台学科管理界面 */
export async function getAllSubjects(): Promise<SubjectItem[]> {
  const res = await apiClient.get<{ data: SubjectListResp }>('/admin/subjects')
  return res.data.data?.subjects || []
}

/** admin：新建学科 */
export async function createSubject(req: CreateSubjectRequest): Promise<SubjectItem> {
  const res = await apiClient.post<{ data: SubjectItem }>('/admin/subjects', req)
  return res.data.data
}

/** admin：编辑学科 */
export async function updateSubject(id: string, req: UpdateSubjectRequest): Promise<SubjectItem> {
  const res = await apiClient.put<{ data: SubjectItem }>(`/admin/subjects/${id}`, req)
  return res.data.data
}

/** admin：删除学科（内置学科后端会拒） */
export async function deleteSubject(id: string): Promise<void> {
  await apiClient.delete(`/admin/subjects/${id}`)
}
