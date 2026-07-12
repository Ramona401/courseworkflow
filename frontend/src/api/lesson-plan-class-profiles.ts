/**
 * lesson-plan-class-profiles.ts — 教案挂载/解除班级学情卡 API（差异化教学·前端入口）
 *
 * 对应后端：handlers/lesson_plan_handler_classprofile.go · services/lesson_plan_service_classprofile.go
 *   PUT /api/v1/lesson-plans/plans/{id}/class-profile   教案挂载/解除班级学情卡（体 {class_profile_id}，空串=解除）
 *
 * 为什么单独成文件、不并进 class-profiles.ts：
 *   class-profiles.ts 是「班级学情资料本身」的 CRUD（建卡/填档/AI总结/分层），数据归属在 /class-profiles 域；
 *   而「把某张班级卡挂到某份教案上」属于教案侧的关联操作，端点在 /lesson-plans/plans/{id} 域，
 *   与 unit-plans.ts 的 updatePlanUnitPlan 同形态（挂载是教案侧动作，不是资料侧动作）。
 *   故镜像 unit-plans.ts 的做法，把这条挂载接口单独成文件，职责清晰、互不污染。
 *
 * 拦截器已处理 code!==0 抛错，本文件直接取 data.data。
 */
import apiClient from './client'

/**
 * 挂载或解除教案关联的班级学情卡
 *
 * 起步首屏选定、对话中途挂载/更换、解除，三种操作都走这一个接口：
 *   - 挂载/更换：classProfileId 传目标班级学情卡 ID
 *   - 解除：    classProfileId 传空串 ''
 *
 * 关键引擎事实：后端注入层每轮对话重读 lesson_plans.class_profile_id 决定是否注入班级学情
 * 四大段群体学情（仅 analyze/design/write 三阶段、仅 active 且归属本人），因此本接口更新成功后
 * 【下一轮对话】自动生效，无需刷新页面（机制与单元方案挂载 updatePlanUnitPlan 完全同款）。
 *
 * @returns { message, mounted, class_profile_id } —— mounted=false 表示已解除
 */
export async function updatePlanClassProfile(
  planId: string,
  classProfileId: string,
): Promise<{ message: string; mounted: boolean; class_profile_id: string }> {
  const { data } = await apiClient.put(`/lesson-plans/plans/${planId}/class-profile`, {
    class_profile_id: classProfileId,
  })
  return data.data as { message: string; mounted: boolean; class_profile_id: string }
}
