/**
 * 课件工坊 API —— 就地文字编辑 + 粘贴HTML导入层 (coursewares.inlineedit.ts)
 *
 * 职责：页面级"确定性 HTML 覆盖"类接口（不调 AI 的整页写入）。当前含两个函数：
 *   - savePageHtml    保存老师改过的整页 HTML（就地文字编辑 / 批次A源码编辑共用）
 *   - importPageHtml  【批次B】把粘贴的外部HTML导入指定页（后端做画布归一/导航重编号/背景补注/快照）
 *
 * 与 refinePage / regenerateCWPage / rollbackPage 同域（都属"页面级修改"），但因它们已在
 * coursewares.core.ts（38KB）中、为避免大文件整体覆盖风险，本项拆为独立小文件，
 * 经桶文件 coursewares.ts re-export，对外 import 路径不变（import { savePageHtml } from '@/api/coursewares'）。
 *
 * 后端对应：
 *   POST /api/v1/coursewares/{id}/pages/{num}/save-html
 *     —— 不调 AI，仅"存旧版(manual 快照) + 写新版"两步落库，属确定性覆盖（同背景/字体秒换、回退）。
 *   POST /api/v1/coursewares/{id}/pages/{num}/import-html
 *     —— 不调 AI；外来HTML先做画布契约归一(1920×1080) + 导航栏替换重编号(仅当自带NAV标记)
 *        + 背景幂等补注 + 覆盖前 manual 快照，再落库并置 generated 状态。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'

/**
 * 保存就地文字编辑结果：把前端在 iframe 内改过（仅文字/字号/颜色，纯 DOM、无脏节点）
 * 并已清理编辑器痕迹的整页纯净 HTML 回传后端落库。
 * 批次A起「✏️ 编辑源码」（Step5 源码视图直接编辑/整页替换）也复用本函数保存。
 *
 * 后端会在覆盖前把【旧】HTML 存为一条 manual 版本快照（可在"版本历史"里回退/对比），
 * 再写回新 HTML（只动 html_content，不碰 placeholder_map/status）。
 *
 * @param coursewareId 课件ID
 * @param pageNumber   页码（1-based）
 * @param htmlContent  编辑后的整页纯净 HTML
 * @returns { page_number, html_content, message } —— html_content 为落库后的最终内容，供前端刷新预览
 *
 * 说明：不设长超时——本接口不调 AI，仅两次 SQL，走 apiClient 默认超时即可。
 */
export async function savePageHtml(
  coursewareId: string,
  pageNumber: number,
  htmlContent: string,
): Promise<{ page_number: number; html_content: string; message: string }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/save-html`,
    { html_content: htmlContent },
  )
  return extractData(resp)
}

/**
 * 【批次B·粘贴HTML建页】把老师粘贴的外部完整HTML导入指定页。
 *
 * 使用场景：Step5「＋添加页面 → 📋 粘贴HTML」模式——addCWPage 建页后调本函数导入粘贴内容。
 * 后端整备流程（均在服务端完成，前端只管透传原始粘贴内容）：
 *   1. 画布契约归一：强制 1920×1080 根容器、剥除误带 transform、补 cw-page 类；
 *   2. 导航栏替换重编号：仅当粘贴代码自带 NAV_START/NAV_END 标记（即从平台其它课件复制来的页）
 *      时替换为本课件导航栏并注入正确页码；外部HTML无标记则跳过、不强插导航；
 *   3. 背景幂等补注：按本课件当前背景（含页级覆盖）补注，视觉与整套课件统一；
 *   4. 覆盖前版本快照（manual）：新建空页旧值为空自动跳过，重复导入则旧版可回退。
 * 导入后页面状态置为 generated，立即计入已生成页面。
 *
 * @param coursewareId 课件ID
 * @param pageNumber   页码（1-based，通常是 addCWPage 刚返回的新页号）
 * @param htmlContent  粘贴的完整页面HTML原文（≤5MB，后端校验）
 * @returns { page_number, html_content, message } —— html_content 为整备落库后的最终内容
 *
 * 说明：不调 AI、纯字符串整备+落库，走 apiClient 默认超时即可。
 */
export async function importPageHtml(
  coursewareId: string,
  pageNumber: number,
  htmlContent: string,
): Promise<{ page_number: number; html_content: string; message: string }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/import-html`,
    { html_content: htmlContent },
  )
  return extractData(resp)
}
