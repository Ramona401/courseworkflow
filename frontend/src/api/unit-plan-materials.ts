/**
 * unit-plan-materials.ts — 大单元方案参考资料API
 *
 * 接口：
 *   GET    /unit-plan-materials?unit_plan_id={id}
 *   POST   /unit-plan-materials?unit_plan_id={id}
 *   DELETE /unit-plan-materials/{materialId}?unit_plan_id={id}
 *
 * 第一阶段不上传原始PDF或Word文件。
 * 浏览器提取文字后，把原文和可选压缩摘要保存到数据库。
 */
import apiClient from './client'

export type UnitPlanMaterialType =
  | 'textbook'
  | 'teacher_guide'
  | 'previous_unit_plan'
  | 'teaching_requirement'
  | 'excellent_case'
  | 'other'

export interface UnitPlanMaterialListItem {
  id: string
  unit_plan_id: string
  material_type: UnitPlanMaterialType
  file_name: string
  original_length: number
  summary_length: number
  has_summary: boolean
  uploaded_by: string
  status: 'active' | 'archived'
  created_at: string
  updated_at: string
}

export interface UnitPlanMaterial {
  id: string
  unit_plan_id: string
  material_type: UnitPlanMaterialType
  file_name: string
  content_text?: string
  summary_text?: string
  original_length: number
  summary_length: number
  uploaded_by: string
  status: 'active' | 'archived'
  created_at: string
  updated_at: string
}

export interface UnitPlanMaterialsResponse {
  materials: UnitPlanMaterialListItem[]
  total: number
  can_manage: boolean
}

export interface CreateUnitPlanMaterialRequest {
  material_type: UnitPlanMaterialType
  file_name: string
  content_text: string
  summary_text: string
}

/** 获取一份单元方案的资料轻量列表。 */
export async function getUnitPlanMaterials(
  unitPlanId: string,
): Promise<UnitPlanMaterialsResponse> {
  const { data } = await apiClient.get('/unit-plan-materials', {
    params: {
      unit_plan_id: unitPlanId,
    },
  })

  return data.data as UnitPlanMaterialsResponse
}

/** 保存浏览器已经提取完成的参考资料文字。 */
export async function createUnitPlanMaterial(
  unitPlanId: string,
  request: CreateUnitPlanMaterialRequest,
): Promise<UnitPlanMaterial> {
  const { data } = await apiClient.post(
    '/unit-plan-materials',
    request,
    {
      params: {
        unit_plan_id: unitPlanId,
      },
      timeout: 180000,
    },
  )

  return data.data.material as UnitPlanMaterial
}

/** 软删除一份参考资料。 */
export async function deleteUnitPlanMaterial(
  unitPlanId: string,
  materialId: string,
): Promise<void> {
  await apiClient.delete(
    `/unit-plan-materials/${materialId}`,
    {
      params: {
        unit_plan_id: unitPlanId,
      },
    },
  )
}
