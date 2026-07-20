/**
 * 课件工坊 API —— 核心流程层 (coursewares.core.ts)
 *
 * 从原 coursewares.ts 拆出：课件 CRUD、页面操作、状态流转、索引 SSE 生成、
 * 风格模板（系统/个人/AI提取/微调/发布）、组件库、种子数据、方案预设。
 * 不含多媒体（图片/视频/字幕/TTS/OSS/离线包），那部分在 coursewares.media.ts。
 * 经桶文件 coursewares.ts 透出，对外 import 路径不变。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'
import type {
  CoursewareListResponse, CoursewareDetail, CoursewarePage,
  CWComponentListItem, CWComponentFull, CoursewareTemplate, SeedResult,
  CWSSECallbacks, SchemePreset, RefineSSECallbacks, ExtractSSECallbacks,
  RefineHistoryEntry, PublishTargetsResponse,
  CurriculumKPResponse, PageVersionEntry,
  AlignmentReportResponse,
  SharedCoursewareListResponse,
} from './coursewares.types'

// ==================== 课件CRUD ====================

export async function getCoursewares(params?: {
  status?: string; subject?: string; limit?: number; offset?: number
}): Promise<CoursewareListResponse> {
  const resp = await apiClient.get('/coursewares', { params })
  return extractData(resp)
}

export async function createCourseware(data: {
  lesson_plan_id: string; title?: string
}): Promise<{ id: string }> {
  const resp = await apiClient.post('/coursewares', data)
  return extractData(resp)
}

export async function getCourseware(id: string): Promise<CoursewareDetail> {
  const resp = await apiClient.get(`/coursewares/${id}`)
  return extractData(resp)
}

export async function updateCourseware(id: string, data: { title: string }): Promise<void> {
  await apiClient.put(`/coursewares/${id}`, data)
}

export async function deleteCourseware(id: string): Promise<void> {
  await apiClient.delete(`/coursewares/${id}`)
}

// ==================== 课件页面操作 ====================

export async function getCoursewarePages(coursewareId: string): Promise<CoursewarePage[]> {
  const resp = await apiClient.get(`/coursewares/${coursewareId}/pages`)
  return extractData(resp)
}

export async function updateCWPageIndex(coursewareId: string, pageNumber: number, data: {
  title?: string; purpose?: string; content_summary?: string
  interaction_type?: string; visual_format?: string
  media_requirements?: string; estimated_complexity?: number
}): Promise<void> {
  await apiClient.put(`/coursewares/${coursewareId}/pages/${pageNumber}`, data)
}

export async function addCWPage(coursewareId: string, data: {
  title: string; purpose?: string; content_summary?: string
  interaction_type?: string; visual_format?: string
  media_requirements?: string; estimated_complexity?: number
}): Promise<CoursewarePage> {
  const resp = await apiClient.post(`/coursewares/${coursewareId}/pages`, data)
  return extractData(resp)
}

export async function deleteCWPage(coursewareId: string, pageNumber: number): Promise<void> {
  await apiClient.delete(`/coursewares/${coursewareId}/pages/${pageNumber}`)
}

export async function reorderCWPages(coursewareId: string, pageIds: string[]): Promise<void> {
  await apiClient.put(`/coursewares/${coursewareId}/pages/reorder`, { page_ids: pageIds })
}

// ==================== 状态流转 ====================

export async function confirmCWIndex(coursewareId: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/confirm-index`)
}

/** 旧接口：直接保存风格JSON字符串 */
export async function saveCWStyle(coursewareId: string, styleConfig: string): Promise<void> {
  await apiClient.put(`/coursewares/${coursewareId}/style`, { style_config: styleConfig })
}

/** Phase 4A: 保存完整风格配置（模板+Logo+机构名+自定义色） */
export async function saveStyleFull(coursewareId: string, data: {
  template_id: string
  logo_url?: string
  org_name?: string
  custom_primary_color?: string
}): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/save-style`, data)
}

/** Phase 4A: 确认风格选择，进入下一步 */
export async function confirmCWStyle(coursewareId: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/confirm-style`)
}

/** Phase 4A: 上传课件Logo */
export async function uploadCWLogo(coursewareId: string, file: File): Promise<{ url: string }> {
  const formData = new FormData()
  formData.append('file', file)
  const resp = await apiClient.post(`/coursewares/${coursewareId}/upload-logo`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return extractData(resp)
}

/** 需求2: 获取当前用户历史用过的 Logo URL 列表（去重、最近优先，风格页免重传复用） */
export async function getLogoHistory(limit = 20): Promise<string[]> {
  const resp = await apiClient.get('/coursewares/logo-history', { params: { limit } })
  const data = extractData<{ logos: string[] }>(resp)
  return data.logos || []
}

/** 需求2: 删除一条历史 Logo（清空所有使用它的课件的 logo_url，使其不再出现在历史中） */
export async function deleteLogoHistory(logoURL: string): Promise<{ affected: number }> {
  const resp = await apiClient.delete('/coursewares/logo-history', { params: { url: logoURL } })
  return extractData(resp)
}

export async function confirmCourseware(coursewareId: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/confirm`)
}

/** Phase 4C: 仅生成预览页（封面+第1内容页），让老师确认导航栏 */
export async function generateCWPreview(coursewareId: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/generate-preview`)
}

/** Phase 4C: 保存导航栏HTML模板（老师确认后） */
export async function saveCWNavTemplate(coursewareId: string, navTemplateHTML: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/save-nav-template`, {
    nav_template_html: navTemplateHTML,
  })
}

/** Phase 4B/4C: 用固定导航栏批量生成剩余页 */
export async function generateCWPages(coursewareId: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/generate-pages`)
}

/** P0-2: 导航栏AI微调 */
export async function refineNav(coursewareId: string, instruction: string): Promise<{ nav_html: string; message: string }> {
  const resp = await apiClient.post(`/coursewares/${coursewareId}/refine-nav`, { instruction })
  return extractData(resp)
}

/**
 * P0-4: 单页AI微调
 * v6.3: 新增可选 image 参数(data URI)。非空时随请求体传 image 字段，
 *       后端走多模态微调(CallAIMultimodal,失败降级CallAI)。
 *       向后兼容：不传 image 时与旧版行为完全一致。
 */
export type CWRefineMode = 'preserve' | 'rebuild'

export async function refinePage(
  coursewareId: string,
  pageNumber: number,
  instruction: string,
  image?: string,
  mode: CWRefineMode = 'preserve',
): Promise<{
  page_number: number
  html_content: string
  message: string
  mode?: CWRefineMode
}> {
  const body: Record<string, string> = { instruction, mode }
  if (image) body.image = image
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/refine`,
    body,
    { timeout: 300000 }, // 整页HTML修改与多模态推理可能较慢，统一使用5分钟超时
  )
  return extractData(resp)
}

/**
 * v6.3 (批次3后端已上线): 单页从零重画
 * 异于 refinePage 的增量修改——RegenerateSinglePage 按方案从零重画整页，
 * 不保留页内已插入的图片。前端调用前须二次确认提示用户。
 */
export async function regenerateCWPage(
  coursewareId: string,
  pageNumber: number,
): Promise<{ page_number: number; html_content: string; message: string }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/regenerate`,
    {},
    { timeout: 300000 }, // 单页重生=大模型从零重画整页，5分钟超时（防后端已成功而前端先超时报假失败）
  )
  return extractData(resp)
}

// ==================== 页面级版本与回退（新增） ====================

/**
 * 页面级版本：获取某页的版本快照列表（轻量，不含 html_content，按 version_no 倒序，最新在前）。
 * 每条含后端已附的来源中文标签 source_label，前端直接展示。
 * 接口：GET /api/v1/coursewares/{id}/pages/{num}/versions
 */
export async function listPageVersions(
  coursewareId: string,
  pageNumber: number,
): Promise<{ page_number: number; versions: PageVersionEntry[]; total: number }> {
  const resp = await apiClient.get(`/coursewares/${coursewareId}/pages/${pageNumber}/versions`)
  return extractData(resp)
}

/**
 * 页面级版本：回退某页到指定历史版本，返回回退后的完整 HTML。
 * 后端在写回目标版本前，会先把【当前】内容另存为一个 rollback 版本，故回退本身可逆。
 * 接口：POST /api/v1/coursewares/{id}/pages/{num}/rollback   body: { version_id }
 */
export async function rollbackPage(
  coursewareId: string,
  pageNumber: number,
  versionId: string,
): Promise<{ page_number: number; html_content: string; message: string }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/rollback`,
    { version_id: versionId },
  )
  return extractData(resp)
}

/**
 * 页面级版本对比：取某个历史版本的完整 HTML（版本对比弹窗左侧渲染用，只读）。
 * 当前版 HTML 前端已有（走 getCoursewarePages 拿当前页 html_content），
 * 历史版就靠本接口按 versionId 单独取，用于左右并排 diff。
 * 接口：GET /api/v1/coursewares/{id}/pages/{num}/versions/{versionId}
 */
export async function getPageVersionDetail(
  coursewareId: string,
  pageNumber: number,
  versionId: string,
): Promise<{
  page_number: number
  version_id: string
  version_no: number
  source: string
  source_label: string
  html_content: string
}> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/pages/${pageNumber}/versions/${versionId}`,
  )
  return extractData(resp)
}

/** P0-5: 中途中断批量生成 */
export async function cancelGenerate(coursewareId: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/cancel-generate`)
}

// ==================== 课件索引AI生成（SSE流式） ====================

export async function generateCWIndex(coursewareId: string, preset?: string, customPromptHint?: string): Promise<void> {
  const body: Record<string, string> = {}
  if (preset) body.preset = preset
  if (customPromptHint) body.custom_prompt_hint = customPromptHint
  await apiClient.post(`/coursewares/${coursewareId}/generate-index`, body)
}

// P2：SSE 自动重连参数（照搬教案 lesson-plans.ts 已验证的指数退避范式）
const CW_SSE_RECONNECT_MAX_RETRIES = 5      // 最大重连次数
const CW_SSE_RECONNECT_BASE_DELAY_MS = 1000 // 基础重连延迟（毫秒），指数退避基数 1s/2s/4s/8s/16s
const CW_SSE_RECONNECT_MAX_DELAY_MS = 30000 // 最大重连延迟（毫秒），防退避过长

/**
 * 订阅课件工坊 SSE（索引生成 / 课件HTML批量生成 共用）
 *
 * P2 修复（断线重连 + 假死根治）：
 *   1. 指数退避自动重连：连接异常断开后 1s→2s→4s→8s→16s 重连，最多 5 次（照搬教案范式）。
 *      —— 根治"老师刷新页面/切标签页/网络抖动后前端假死、进度不动"。
 *   2. 单页 error 事件不再关闭整条连接：后端某页 AI 失败会广播 error/gen_progress，
 *      旧逻辑收到 error 就 close() 整条 SSE，导致后续成功页与 gen_done 完成事件全收不到 → 永久假死。
 *      新逻辑：error 事件只回调 onError 透传消息，连接保持，直到 gen_done/index_done 正常收尾才关闭。
 *   3. 重连成功后触发 onReconnected：业务层据此重新拉取课件状态+页面列表，补齐断线期间漏收的已生成页。
 *
 * 终止条件（主动 close，不再重连）：收到 gen_done / index_done（正常完成）或业务层手动 close()。
 * 注意：EventSource 的 onerror 既包括"真正断线"也包括"连接正常关闭后"，故用 isClosed 标志区分，
 *   正常完成 close 后不再触发重连。
 */
export function subscribeCWIndexSSE(
  coursewareId: string,
  callbacks: CWSSECallbacks,
): { close: () => void } {
  const token = localStorage.getItem('token') || ''
  const url = `${window.location.origin}/api/v1/sse/courseware/${coursewareId}?token=${encodeURIComponent(token)}`

  let currentES: EventSource | null = null
  let retryCount = 0
  let retryTimer: ReturnType<typeof setTimeout> | null = null
  let isClosed = false        // 业务层主动关闭 或 正常完成关闭：置 true 后不再重连
  let isFirstConnect = true   // 首次连接的 connected 不算"重连成功"

  const bindEventListeners = (es: EventSource) => {
    es.addEventListener('connected', (e: MessageEvent) => {
      retryCount = 0 // 连上即重置重连计数
      try { callbacks.onConnected?.(JSON.parse(e.data)) } catch { /* */ }
      if (!isFirstConnect) {
        // 这是一次"重连成功"：通知业务层补齐断线期间漏收的页面/进度
        callbacks.onReconnected?.()
      }
      isFirstConnect = false
    })
    es.addEventListener('index_start', (e: MessageEvent) => {
      try { callbacks.onIndexStart?.(JSON.parse(e.data)) } catch { /* */ }
    })
    es.addEventListener('index_page', (e: MessageEvent) => {
      try { callbacks.onIndexPage?.(JSON.parse(e.data)) } catch { /* */ }
    })
    es.addEventListener('index_progress', (e: MessageEvent) => {
      try { callbacks.onIndexProgress?.(JSON.parse(e.data)) } catch { /* */ }
    })
    es.addEventListener('index_done', (e: MessageEvent) => {
      try { callbacks.onIndexDone?.(JSON.parse(e.data)) } catch { /* */ }
      isClosed = true // 正常完成，主动关闭且不再重连
      es.close()
    })
    es.addEventListener('gen_start', (e: MessageEvent) => {
      try { callbacks.onGenStart?.(JSON.parse(e.data)) } catch { /* */ }
    })
    es.addEventListener('gen_page', (e: MessageEvent) => {
      try { callbacks.onGenPage?.(JSON.parse(e.data)) } catch { /* */ }
    })
    es.addEventListener('gen_progress', (e: MessageEvent) => {
      try { callbacks.onGenProgress?.(JSON.parse(e.data)) } catch { /* */ }
    })
    es.addEventListener('gen_done', (e: MessageEvent) => {
      try { callbacks.onGenDone?.(JSON.parse(e.data)) } catch { /* */ }
      isClosed = true // 正常完成，主动关闭且不再重连
      es.close()
    })
    // P2 关键：业务级 error 事件（如某页 AI 失败）只透传消息，绝不关闭整条连接，
    //   让后续成功页与 gen_done 完成事件继续到达，根治"失败一页就整体假死"。
    es.addEventListener('error', (e: MessageEvent) => {
      if (e.data) {
        try { callbacks.onError?.(JSON.parse(e.data)) } catch { /* */ }
      }
      // 不再 es.close()
    })

    // ==================== 全自动装配事件（assembly_*）====================
    // 与 gen_* 并列的独立事件族，全自动/中间档两种交付模式共用。
    // 复用同一条 SSE 连接与断线重连机制，无需另开通道。
    es.addEventListener('assembly_start', (e: MessageEvent) => {
      try { callbacks.onAssemblyStart?.(JSON.parse(e.data)) } catch { /* */ }
    })
    es.addEventListener('assembly_page_html', (e: MessageEvent) => {
      try { callbacks.onAssemblyPageHtml?.(JSON.parse(e.data)) } catch { /* */ }
    })
    // HTML 生成失败进度（后端 EventType 'assembly_progress'）
    es.addEventListener('assembly_progress', (e: MessageEvent) => {
      try { callbacks.onAssemblyProgress?.(JSON.parse(e.data)) } catch { /* */ }
    })
    // 配图/视频阶段进度：后端分 assembly_page_image / assembly_page_video 两个事件，
    //   前端合并到 onAssemblyPageMedia 一个回调（二者 data 同构：{page_number, stage, message}）。
    es.addEventListener('assembly_page_image', (e: MessageEvent) => {
      try { callbacks.onAssemblyPageMedia?.(JSON.parse(e.data)) } catch { /* */ }
    })
    es.addEventListener('assembly_page_video', (e: MessageEvent) => {
      try { callbacks.onAssemblyPageMedia?.(JSON.parse(e.data)) } catch { /* */ }
    })
    es.addEventListener('assembly_page_done', (e: MessageEvent) => {
      try { callbacks.onAssemblyPageDone?.(JSON.parse(e.data)) } catch { /* */ }
    })
    // 装配全部完成：与 gen_done/index_done 同样是正常终止事件，收尾主动关闭且不再重连。
    es.addEventListener('assembly_done', (e: MessageEvent) => {
      try { callbacks.onAssemblyDone?.(JSON.parse(e.data)) } catch { /* */ }
      isClosed = true // 正常完成，主动关闭且不再重连
      es.close()
    })

    // 传输层断开（网络抖动/刷新/切页/服务端关连接）：指数退避自动重连
    es.onerror = () => {
      if (isClosed) return // 已正常完成或业务层主动关闭，不重连
      es.close()
      currentES = null

      if (retryCount >= CW_SSE_RECONNECT_MAX_RETRIES) {
        // 重连次数耗尽：透传一条提示给业务层，让其改为轮询/手动刷新兜底
        callbacks.onError?.({ message: '连接已断开且多次重连失败，请刷新页面查看最新进度（生成仍在后台继续，不会丢失）' })
        return
      }

      const delay = Math.min(
        CW_SSE_RECONNECT_BASE_DELAY_MS * Math.pow(2, retryCount),
        CW_SSE_RECONNECT_MAX_DELAY_MS,
      )
      retryCount++
      console.log(`[CW-SSE] 连接断开，${delay / 1000}秒后第${retryCount}次重连…（courseware: ${coursewareId}）`)
      retryTimer = setTimeout(() => {
        if (isClosed) return
        connectSSE()
      }, delay)
    }
  }

  const connectSSE = () => {
    if (isClosed) return
    const es = new EventSource(url)
    currentES = es
    bindEventListeners(es)
  }

  connectSSE()

  return {
    close: () => {
      isClosed = true
      if (retryTimer) { clearTimeout(retryTimer); retryTimer = null }
      if (currentES) { currentES.close(); currentES = null }
    },
  }
}

// ==================== 风格模板 ====================

export async function getCWTemplates(): Promise<CoursewareTemplate[]> {
  const resp = await apiClient.get('/courseware-templates')
  return extractData(resp)
}

export async function getCWTemplatePreview(id: string): Promise<CoursewareTemplate> {
  const resp = await apiClient.get(`/courseware-templates/${id}/preview`)
  return extractData(resp)
}

// ==================== 课件组件库 ====================

export async function getCWComponents(params?: {
  component_type?: string; subject_scope?: string; grade_scope?: string
  limit?: number; offset?: number
}): Promise<{ components: CWComponentListItem[]; total: number }> {
  const resp = await apiClient.get('/courseware-components', { params })
  return extractData(resp)
}

export async function createCWComponent(data: {
  name: string; description?: string; component_type: string
  code_content: string; preview_html?: string
  subject_scope?: string; grade_scope?: string
}): Promise<CWComponentListItem> {
  const resp = await apiClient.post('/courseware-components', data)
  return extractData(resp)
}

export async function getCWComponent(id: string): Promise<CWComponentFull> {
  const resp = await apiClient.get(`/courseware-components/${id}`)
  return extractData(resp)
}

export async function updateCWComponent(id: string, data: Record<string, unknown>): Promise<void> {
  await apiClient.put(`/courseware-components/${id}`, data)
}

export async function deleteCWComponent(id: string): Promise<void> {
  await apiClient.delete(`/courseware-components/${id}`)
}

export async function matchCWComponents(data: {
  component_type?: string; subject_scope?: string; grade_scope?: string
  interaction_level?: number; visual_format?: string; limit?: number
}): Promise<CWComponentListItem[]> {
  const resp = await apiClient.post('/courseware-components/match', data)
  return extractData(resp)
}

// ==================== 种子数据填充（admin） ====================

export async function seedCoursewareData(force?: boolean): Promise<SeedResult> {
  const resp = await apiClient.post('/admin/courseware-seed', { force: !!force })
  return extractData(resp)
}

// ==================== Admin模板管理 ====================

export async function createCWTemplate(data: {
  name: string; description?: string; style_category: string
  color_scheme?: string; css_variables?: string; sample_pages?: string; preview_urls?: string
}): Promise<CoursewareTemplate> {
  const resp = await apiClient.post('/admin/courseware-templates', data)
  return extractData(resp)
}

export async function updateCWTemplate(id: string, data: Record<string, unknown>): Promise<void> {
  await apiClient.put(`/admin/courseware-templates/${id}`, data)
}

export async function deleteCWTemplate(id: string): Promise<void> {
  await apiClient.delete(`/admin/courseware-templates/${id}`)
}

// ==================== v136: 步骤回退+AI修改方案+方案预设 ====================

/** v136: 回退课件状态到指定步骤 */
export async function rollbackCWStatus(coursewareId: string, targetStatus: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/rollback-status`, { target_status: targetStatus })
}

/** v136: AI修改方案（异步，通过SSE推送进度） */
export async function refineCWIndex(coursewareId: string, feedback: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/refine-index`, { feedback })
}

/** v136: 获取方案结构预设列表 */
export async function getSchemePresets(): Promise<SchemePreset[]> {
  const resp = await apiClient.get('/courseware-presets')
  return extractData(resp)
}

// ==================== v137: 个人模板管理 ====================

/** v137: 获取系统模板+我的个人模板（风格选择页用） */
export async function getCWTemplatesWithUser(): Promise<CoursewareTemplate[]> {
  const resp = await apiClient.get('/courseware-templates/with-user')
  return extractData(resp)
}

/** v137: 保存当前课件为我的模板 */
export async function saveAsMyTemplate(coursewareId: string, data: {
  name: string; description?: string; style_category?: string
}): Promise<{ id: string; name: string; message: string }> {
  const resp = await apiClient.post(`/coursewares/${coursewareId}/save-as-template`, data)
  return extractData(resp)
}

/** v137: 删除我的个人模板 */
export async function deleteMyTemplate(templateId: string): Promise<void> {
  await apiClient.delete(`/courseware-templates/personal/${templateId}`)
}

// ==================== v139/v145: AI 提取 + 微调 + 发布 ====================

/** v145: AI 提取风格模板(异步启动,通过 SSE 推送进度) */
export async function extractTemplateFromHTML(samplePages: string[], sourceType = 'paste'): Promise<void> {
  await apiClient.post('/coursewares/templates/extract',
    { sample_pages: samplePages, source_type: sourceType },
  )
}

/** v145: 订阅模板 AI 提取 SSE 事件流 */
export function subscribeExtractSSE(callbacks: ExtractSSECallbacks): { close: () => void } {
  const token = localStorage.getItem('token') || ''
  const url = `${window.location.origin}/api/v1/sse/template-extract?token=${encodeURIComponent(token)}`
  const es = new EventSource(url)
  es.addEventListener('extract_start', e => { try { callbacks.onStart?.(JSON.parse((e as MessageEvent).data)) } catch { /* */ } })
  es.addEventListener('extract_progress', e => { try { callbacks.onProgress?.(JSON.parse((e as MessageEvent).data)) } catch { /* */ } })
  es.addEventListener('extract_done', e => { try { callbacks.onDone?.(JSON.parse((e as MessageEvent).data)) } catch { /* */ } ; es.close() })
  es.addEventListener('extract_error', e => { try { callbacks.onError?.(JSON.parse((e as MessageEvent).data)) } catch { /* */ } ; es.close() })
  es.onerror = () => { es.close() }
  return { close: () => es.close() }
}

/** v139: 查询我的草稿列表 */
export async function listMyDrafts(): Promise<CoursewareTemplate[]> {
  const resp = await apiClient.get('/coursewares/templates/my-drafts')
  return extractData(resp)
}

/** v139: 删除草稿 */
export async function deleteDraft(templateId: string): Promise<void> {
  await apiClient.delete(`/coursewares/templates/drafts/${templateId}`)
}

/** v139: 触发AI微调(异步,通过SSE推送进度) */
export async function refineTemplate(templateId: string, instruction: string): Promise<void> {
  await apiClient.post(`/coursewares/templates/${templateId}/refine`, { instruction })
}

/** v139: 订阅模板微调SSE事件流 */
export function subscribeTemplateRefineSSE(templateId: string, callbacks: RefineSSECallbacks): { close: () => void } {
  const token = localStorage.getItem('token') || ''
  const url = `${window.location.origin}/api/v1/sse/template-refine/${templateId}?token=${encodeURIComponent(token)}`
  const es = new EventSource(url)
  es.addEventListener('refine_start', e => { try { callbacks.onStart?.(JSON.parse((e as MessageEvent).data)) } catch { /* */ } })
  es.addEventListener('refine_chunk', e => { try { callbacks.onChunk?.(JSON.parse((e as MessageEvent).data)) } catch { /* */ } })
  es.addEventListener('refine_progress', e => { try { callbacks.onProgress?.(JSON.parse((e as MessageEvent).data)) } catch { /* */ } })
  es.addEventListener('refine_done', e => { try { callbacks.onDone?.(JSON.parse((e as MessageEvent).data)) } catch { /* */ } ; es.close() })
  es.addEventListener('refine_error', e => { try { callbacks.onError?.(JSON.parse((e as MessageEvent).data)) } catch { /* */ } ; es.close() })
  es.onerror = () => { es.close() }
  return { close: () => es.close() }
}

/** v139: 获取微调历史 */
export async function getTemplateHistory(templateId: string): Promise<{ template_id: string; history: RefineHistoryEntry[]; total: number }> {
  const resp = await apiClient.get(`/coursewares/templates/${templateId}/history`)
  return extractData(resp)
}

/**
 * v139: 回退到历史快照
 * 修复: 直接拼接正确URL,不再使用 .replace() hack
 */
export async function rollbackTemplate(templateId: string, historyIndex: number): Promise<{
  template_id: string; color_scheme: string; css_variables: string; sample_pages: string; style_category: string; message: string
}> {
  const resp = await apiClient.post(`/coursewares/templates/${templateId}/rollback`, { history_index: historyIndex })
  return extractData(resp)
}

/** v139: 发布草稿为正式模板 */
export async function publishDraft(templateId: string, data: {
  name: string; description?: string; style_category?: string; scope: string; scope_target_id?: string
}): Promise<{ template_id: string; name: string; scope: string; message: string }> {
  const resp = await apiClient.post(`/coursewares/templates/${templateId}/publish`, data)
  return extractData(resp)
}

/** v142: 撤回已发布模板为草稿 */
export async function unpublishTemplate(templateId: string): Promise<{ template_id: string; message: string }> {
  const resp = await apiClient.post(`/coursewares/templates/${templateId}/unpublish`)
  return extractData(resp)
}

/** v139.1: 查询当前用户可发布的所有目标 */
export async function getPublishTargets(): Promise<PublishTargetsResponse> {
  const resp = await apiClient.get('/coursewares/templates/publish-targets')
  return extractData(resp)
}

// ==================== v0.42: 多入口创建 ====================

/** v0.42: 从主题直接创建课件 */
export async function createCoursewareFromTopic(data: {
  subject: string
  grade: string
  topic: string
  page_range?: string
  extra_notes?: string
  kp_codes?: string[]   // 课程知识库轮：勾选的课标知识点编码数组（可选，用于难度自动适配）
}): Promise<{ id: string }> {
  const resp = await apiClient.post('/coursewares/from-topic', data)
  return extractData(resp)
}

/** v0.42: 从主题直接生成课件索引（异步，通过SSE推送进度） */
export async function generateCWIndexFromTopic(coursewareId: string, data: {
  subject: string
  grade: string
  topic: string
  page_range?: string
  extra_notes?: string
  preset?: string
  custom_prompt_hint?: string
}): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/generate-index-topic`, data)
}

/** v0.42 入口B: 上传PPT创建课件（multipart/form-data） */
export async function createCoursewareFromPPT(
  file: File,
  subject: string,
  grade: string,
  title?: string,
): Promise<{ id: string; title: string; subject: string; grade: string; source_type: string; slide_count: number; message: string }> {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('subject', subject)
  formData.append('grade', grade)
  if (title) formData.append('title', title)
  const resp = await apiClient.post('/coursewares/from-ppt', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 300000, // v0.43.1: 大PPT(20MB+)上传较慢, 2分钟→5分钟超时(配合后端ReadHeaderTimeout解除上传读取上限)
  })
  return extractData(resp)
}

/** v0.42 入口B: 从PPT内容生成课件索引（异步，通过SSE推送进度） */
export async function generateCWIndexFromPPT(coursewareId: string, preset?: string, customPromptHint?: string): Promise<void> {
  const body: Record<string, string> = {}
  if (preset) body.preset = preset
  if (customPromptHint) body.custom_prompt_hint = customPromptHint
  await apiClient.post(`/coursewares/${coursewareId}/generate-index-ppt`, body)
}

/** v0.42 入口C: 上传Word文档创建课件（multipart/form-data） */
export async function createCoursewareFromDoc(
  file: File,
  subject: string,
  grade: string,
  title?: string,
): Promise<{ id: string; title: string; subject: string; grade: string; source_type: string; word_count: number; message: string }> {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('subject', subject)
  formData.append('grade', grade)
  if (title) formData.append('title', title)
  const resp = await apiClient.post('/coursewares/from-doc', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 300000, // v0.43.1: 大Word文档上传较慢, 1分钟→5分钟超时(配合后端ReadHeaderTimeout解除上传读取上限)
  })
  return extractData(resp)
}

/** v0.42 入口C: 从Word文档生成课件索引（异步SSE） */
export async function generateCWIndexFromDoc(coursewareId: string, preset?: string, customPromptHint?: string): Promise<void> {
  const body: Record<string, string> = {}
  if (preset) body.preset = preset
  if (customPromptHint) body.custom_prompt_hint = customPromptHint
  await apiClient.post(`/coursewares/${coursewareId}/generate-index-doc`, body)
}

// ==================== v0.42.11 3D互动单页 ====================

/** v0.42.11: 触发3D互动单页生成（异步，通过SSE推送进度） */
export async function generate3DPage(coursewareId: string): Promise<{ message: string; courseware_id: string }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/generate-3d-page`,
    {},
    { timeout: 180000 }, // 3分钟超时（3D生成较慢）
  )
  return extractData(resp)
}

/** v0.42.11: 创建3D互动单页课件（source_type='3d_single'，状态直接为generating） */
export async function createCoursewareFrom3D(data: {
  subject: string
  grade: string
  topic: string
  description: string
}): Promise<{ id: string; title: string; source_type: string; status: string; message: string }> {
  const resp = await apiClient.post('/coursewares/from-3d', data)
  return extractData(resp)
}


// ==================== 课程知识库（平台级公共只读查询，课件/教案复用） ====================

/**
 * 查询某学科某年级的课标知识点清单（供"从主题创建"勾选 + 难度适配）。
 * grade 传年级数字 1-9；传 0 或省略则查该学科全部年级（一般应传具体年级）。
 * 接口：GET /api/v1/curriculum/knowledge-points?subject=数学&grade=3
 */
export async function getCurriculumKnowledgePoints(
  subject: string,
  grade?: number,
): Promise<CurriculumKPResponse> {
  const params: Record<string, string | number> = { subject }
  if (grade && grade > 0) params.grade = grade
  const resp = await apiClient.get('/curriculum/knowledge-points', { params })
  return extractData(resp)
}

// ==================== 课件↔教案对齐报告 ====================

/**
 * 查询课件↔教案对齐报告。
 * 前端 Step1（确认方案）加载时调用；若返回 report.status==='generating' 则短轮询。
 * has_report=false 表示非教案来源或从未校验，前端不显示对齐卡片。
 */
export async function getAlignmentReport(coursewareId: string): Promise<AlignmentReportResponse> {
  const resp = await apiClient.get(`/coursewares/${coursewareId}/alignment-report`)
  return extractData(resp)
}

/**
 * 手动触发对齐重算（老师改完方案后主动重新校验）。
 * 立即返回，校验异步进行；调用方随后短轮询 getAlignmentReport 直到 done/failed。
 */
export async function recheckAlignment(coursewareId: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/recheck-alignment`)
}


// ==================== 断裂B: 课件关联教案正文（对照抽屉） ====================

/** 课件关联教案正文响应 */
export interface CoursewareLessonPlanContent {
  has_lesson_plan: boolean
  title: string
  content: string
}

/**
 * 取课件关联教案的纯文本正文（供 Step4/Step5 工作台"原教案对照抽屉"展示）。
 * 后端复用教案正文提取优先级链，对话生成型教案也能拿到正文。
 * has_lesson_plan=false 表示非教案来源/无关联教案，前端不显示抽屉入口。
 */
export async function getCoursewareLessonPlanContent(coursewareId: string): Promise<CoursewareLessonPlanContent> {
  const resp = await apiClient.get(`/coursewares/${coursewareId}/lesson-plan-content`)
  return extractData(resp)
}


// ==================== 阶段1：课件发布与共享 + 产权分级 ====================
//
// 与 status 生产状态机正交的"发布/共享维度"接口（对应后端 courseware_share_*）。
// 四个端点：发布/撤回、设代码开放范围、共享课件库列表、复制到我的。

/**
 * 发布 / 撤回课件（设置发布态）。
 * target 仅允许：published_personal（个人发布）/ published_shared（共享发布）/ private（撤回到私有）。
 *   - submitted/approved/revision 等审核态由阶段3的审核接口管理，此处不可传。
 *   - published_shared 要求课件已生成到至少 preview 状态，否则后端拒绝（不能共享半成品）。
 * 接口：POST /api/v1/coursewares/{id}/publish   body: { target }
 */
export async function publishCourseware(
  coursewareId: string,
  target: 'published_personal' | 'published_shared' | 'private',
): Promise<{ message: string }> {
  const resp = await apiClient.post(`/coursewares/${coursewareId}/publish`, { target })
  return extractData(resp)
}

/**
 * 设置课件源代码开放范围（产权分级，独立于可见范围）。
 * scope 仅允许：none/group/school/region/public。
 *   可见范围决定谁能"看渲染效果"，code_share_scope 决定谁能"复制源码"，两者解耦。
 * 接口：PUT /api/v1/coursewares/{id}/code-share-scope   body: { code_share_scope }
 */
export async function setCodeShareScope(
  coursewareId: string,
  scope: 'none' | 'group' | 'school' | 'region' | 'public',
): Promise<{ message: string }> {
  const resp = await apiClient.put(`/coursewares/${coursewareId}/code-share-scope`, {
    code_share_scope: scope,
  })
  return extractData(resp)
}

/**
 * 查询共享课件库（他人共享给"我"——同校/同组的课件）。
 * admin 看全部已共享课件；其他角色按"同校∪同组作者白名单"过滤。
 * 每条带 can_copy 标记（当前登录者能否复制该课件源码），前端据此显隐"复制到我的"按钮。
 *
 * 空值兜底关键：后端共享列表为空时返回 coursewares:null（非 []），
 *   调用方务必用 `resp.coursewares || []` 兜底，避免 .map 崩。
 * 接口：GET /api/v1/coursewares/shared?subject=&limit=&offset=
 */
export async function listSharedCoursewares(params?: {
  subject?: string; limit?: number; offset?: number
}): Promise<SharedCoursewareListResponse> {
  const resp = await apiClient.get('/coursewares/shared', { params })
  return extractData(resp)
}

/**
 * 复制共享课件到我的（Fork：深拷贝主记录 + 页面 + 资产）。
 * 前置（后端校验）：源课件须为 published_shared、当前用户对其 code_share_scope 有复制权、不能复制自己的。
 * 副本归当前用户、private、code_share_scope=none、不挂源教案。
 * 接口：POST /api/v1/coursewares/{id}/fork
 */
export async function forkCourseware(
  coursewareId: string,
): Promise<{ id: string; title: string; message: string }> {
  const resp = await apiClient.post(`/coursewares/${coursewareId}/fork`, {})
  return extractData(resp)
}


// ==================== 全自动一键装配 ====================

/**
 * 全自动一键装配课件（HTML生成 + AI配图 + 视频首帧占位 总装线）。
 * 异步启动，立即返回；真实进度经 subscribeCWIndexSSE 的 assembly_* 事件推送。
 *
 * 交付模式（三档）由 skipVideo 区分——本函数对应后两档，纯手动档走 generateCWPages 不调此函数：
 *   - skipVideo=false（默认）：全自动装配，HTML + 配图 + 视频首帧占位（视频按方案关键词命中页决定）
 *   - skipVideo=true         ：HTML+配图不做视频（中间档），所有页一律跳过视频占位
 *
 * 前置强约束（后端 prepareAssembly 校验，未满足则经 SSE 推 error 事件并中止，前端应在调用前先自查）：
 *   ① 已确认导航栏（nav_template_html 非空）；② 已设风格锚点（style_anchor_asset_id 非空）。
 *
 * 接口：POST /api/v1/coursewares/{id}/auto-assemble   body: { skip_video }
 */
export async function autoAssemble(coursewareId: string, skipVideo = false): Promise<{
  message: string; courseware_id: string; skip_video: boolean
}> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/auto-assemble`,
    { skip_video: skipVideo },
  )
  return extractData(resp)
}
