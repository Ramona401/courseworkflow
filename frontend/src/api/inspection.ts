/**
 * inspection.ts — 区域抽查API封装
 *
 * v127 新增（多级审核体系 · 区域抽查）
 */
import apiClient from './client'

// ==================== 类型定义 ====================

export interface InspectionListItem {
  id: string
  lesson_plan_id: string
  plan_title: string
  plan_subject: string
  plan_grade: string
  author_name: string
  school_name: string
  inspector_id: string | null
  inspector_name: string
  sample_batch: string
  status: string
  status_name: string
  priority: number
  comment: string
  inspected_at: string | null
  created_at: string
}

export interface InspectionListResponse {
  items: InspectionListItem[]
  total: number
}

export interface InspectionStatsResponse {
  total_sampled: number
  total_pending: number
  total_passed: number
  total_revoked: number
  pass_rate: number
}

export interface DistrictInspectorItem {
  id: string
  inspector_id: string
  inspector_name: string
  region_id: string
  region_name: string
  created_at: string
}

// ==================== 辅助函数 ====================

function extractData<T>(resp: { data?: { data?: T } }): T {
  const d = resp?.data as Record<string, unknown> | undefined
  if (d && 'data' in d) return d.data as T
  return d as unknown as T
}

// ==================== API 函数 ====================

/** 获取抽查列表 */
export async function getInspections(params?: { status?: string; limit?: number; offset?: number }) {
  const resp = await apiClient.get('/inspections', { params })
  return extractData<InspectionListResponse>(resp)
}

/** 获取抽查详情 */
export async function getInspection(id: string) {
  const resp = await apiClient.get(`/inspections/${id}`)
  return extractData<InspectionListItem>(resp)
}

/** 提交抽查结果 */
export async function reviewInspection(id: string, req: { decision: 'passed' | 'revoked'; comment: string }) {
  const resp = await apiClient.post(`/inspections/${id}/review`, req)
  return extractData<{ message: string }>(resp)
}

/** 手动触发抽样 */
export async function batchSample(req?: { school_id?: string; sample_rate?: number }) {
  const resp = await apiClient.post('/inspections/batch-sample', req || {})
  return extractData<{ message: string; sampled_count: number }>(resp)
}

/** 分配审查员 */
export async function assignInspector(id: string, inspectorId: string) {
  const resp = await apiClient.put(`/inspections/${id}/assign`, { inspector_id: inspectorId })
  return extractData<{ message: string }>(resp)
}

/** 获取抽查统计 */
export async function getInspectionStats() {
  const resp = await apiClient.get('/inspections/stats')
  return extractData<InspectionStatsResponse>(resp)
}

/** 获取区域教研员列表 */
export async function getDistrictInspectors(regionId?: string) {
  const resp = await apiClient.get('/district-inspectors', { params: regionId ? { region_id: regionId } : {} })
  return extractData<DistrictInspectorItem[]>(resp)
}

/** 分配教研员到区域 */
export async function createDistrictInspector(req: { inspector_id: string; region_id: string }) {
  const resp = await apiClient.post('/district-inspectors', req)
  return extractData<DistrictInspectorItem>(resp)
}

/** 取消教研员分配 */
export async function deleteDistrictInspector(id: string) {
  const resp = await apiClient.delete(`/district-inspectors/${id}`)
  return extractData<{ message: string }>(resp)
}

// ==================== 状态常量 ====================

export const INSPECTION_STATUS_CONFIG: Record<string, { label: string; color: string; bg: string; icon: string }> = {
  pending:   { label: '待分配', color: '#9CA3AF', bg: 'rgba(156,163,175,0.08)', icon: '⏳' },
  assigned:  { label: '已分配', color: '#4F7BE8', bg: 'rgba(79,123,232,0.08)',  icon: '📌' },
  in_review: { label: '审查中', color: '#F59E0B', bg: 'rgba(245,158,11,0.08)',  icon: '🔍' },
  passed:    { label: '抽查通过', color: '#10B981', bg: 'rgba(16,185,129,0.08)', icon: '✅' },
  revoked:   { label: '已撤回', color: '#EF4444', bg: 'rgba(239,68,68,0.08)',   icon: '🚫' },
}
