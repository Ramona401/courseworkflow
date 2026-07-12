/**
 * subjectExperiment.ts — 物理/化学/生命科学/地理组件 AI 生成 API
 *
 * 为降低后端路由变更风险，统一复用 /math-graph/generate。
 *
 * target:
 *   - physics_lab：物理实验
 *   - chem_experiment：化学实验
 *   - biology_lab：生命科学互动观察
 *   - geography_lab：地理互动探究
 *
 * 后端返回完整 HTML 组件片段。
 */

import apiClient from './client'

const GENERATE_TIMEOUT_MS = 300000

export type SubjectExperimentTarget =
  | 'physics_lab'
  | 'chem_experiment'
  | 'biology_lab'
  | 'geography_lab'

export interface SubjectExperimentGenerateRequest {
  target: SubjectExperimentTarget
  mode: 'adapt' | 'create'
  description: string
  base_code?: string
  template_name?: string
  image?: string
}

export interface SubjectExperimentGenerateResponse {
  /** 完整 HTML 组件片段，内部必须含 __ROOT_ID__ 占位符 */
  code: string
}

export async function generateSubjectExperimentCode(
  data: SubjectExperimentGenerateRequest,
): Promise<SubjectExperimentGenerateResponse> {
  const resp = await apiClient.post(
    '/math-graph/generate',
    data,
    { timeout: GENERATE_TIMEOUT_MS },
  )

  return resp.data.data as SubjectExperimentGenerateResponse
}
