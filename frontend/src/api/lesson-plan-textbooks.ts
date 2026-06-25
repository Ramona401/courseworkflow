/**
 * lesson-plan-textbooks.ts — 教案课本关联API封装（迭代3.5 A2-2 新增）
 *
 * 对应后端 PUT /api/v1/lesson-plans/plans/{id}/textbooks。
 * 独立成文件的原因：lesson-plans.ts 已是大文件，按模块化纪律新功能落新文件。
 *
 * 关键引擎事实：后端 LoadStagePromptContextV2 每轮对话重读
 * lesson_plans.textbook_page_ids 拼课本OCR原文进系统提示词，
 * 因此本接口更新成功后【下一轮对话】自动携带课本上下文，无需刷新页面。
 */
import apiClient from './client'

/** 更新教案关联的课本页面ID列表（传空数组=解除全部关联；上限20张） */
export async function updatePlanTextbooks(planId: string, pageIds: string[]) {
  const { data } = await apiClient.put(`/lesson-plans/plans/${planId}/textbooks`, {
    textbook_page_ids: pageIds,
  })
  return data.data as { message: string; count: number }
}
