/**
 * coursewareAssembly.ts — 课件全自动装配生命周期 API
 *
 * 本模块只负责装配运行状态与显式取消：
 *   - GET  /coursewares/{id}/assembly-state
 *   - POST /coursewares/{id}/cancel-auto-assemble
 *
 * 启动接口仍保留在 coursewares.core.ts 的 autoAssemble，避免改变既有对外入口。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'

export type CoursewareAssemblyRuntimeStatus =
  | 'idle'
  | 'starting'
  | 'running'
  | 'cancel_requested'
  | 'completed'
  | 'cancelled'
  | 'failed'
  | 'interrupted'

/** R-04 受控页面生成运行类型。 */
export type CoursewareGenerationRunKind =
  | 'assembly'
  | 'batch'

/** R-04 对教师展示的稳定页面引用。 */
export interface CoursewareGenerationPageRef {
  page_id: string
  page_number: number
  title: string
  reason?: string
}

/** R-04 服务端页数完整性事实，前端不得自行重算 complete。 */
export interface CoursewareGenerationIntegrity {
  schema_version: number
  run_kind: CoursewareGenerationRunKind
  expected_count: number
  actual_page_count: number
  success_count: number
  failed_count: number
  cancelled_count: number
  missing_count: number
  pending_count: number
  complete: boolean
  success_pages: CoursewareGenerationPageRef[]
  failed_pages: CoursewareGenerationPageRef[]
  cancelled_pages: CoursewareGenerationPageRef[]
  missing_pages: CoursewareGenerationPageRef[]
  reconciled_at?: string | null
}

export type CoursewareImageRepairErrorCode =
  | 'iaoci_plan_invalid'
  | 'image_content_review'

export interface CoursewareImageRepairItem {
  page_id: string
  page_number: number
  page_title: string
  placeholder_id: string
  image_key?: string
  error_code: CoursewareImageRepairErrorCode
  message: string
}

export interface CoursewareImageRepairState {
  retryable_count: number
  affected_page_count: number
  items: CoursewareImageRepairItem[]
}

export interface CoursewareAssemblyState {
  courseware_id: string
  assembly_version: number
  assembly_status:
    | 'idle'
    | 'running'
    | 'cancel_requested'
    | 'completed'
    | 'cancelled'
    | 'failed'
    | 'interrupted'
  runtime_status: CoursewareAssemblyRuntimeStatus
  run_kind: CoursewareGenerationRunKind | ''
  integrity: CoursewareGenerationIntegrity | null
  image_repair: CoursewareImageRepairState | null
  repair_failed_images: boolean
  active_run_id: string | null
  started_by: string | null
  skip_video: boolean
  started_at: string | null
  finished_at: string | null
  is_starting: boolean
  launch_started_at: string | null
  is_active: boolean
}

/** 读取当前课件的装配生命周期状态。 */
export async function getCoursewareAssemblyState(
  coursewareId: string,
): Promise<CoursewareAssemblyState> {
  const response = await apiClient.get(
    `/coursewares/${coursewareId}/assembly-state`,
  )

  return extractData<CoursewareAssemblyState>(response)
}

/** 显式取消正在启动或运行中的自动装配。 */
export async function cancelCoursewareAutoAssembly(
  coursewareId: string,
): Promise<{
  courseware_id: string
  message: string
}> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/cancel-auto-assemble`,
  )

  return extractData<{
    courseware_id: string
    message: string
  }>(response)
}
