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
  CurriculumKPResponse,
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
export async function refinePage(
  coursewareId: string,
  pageNumber: number,
  instruction: string,
  image?: string,
): Promise<{ page_number: number; html_content: string; message: string }> {
  const body: Record<string, string> = { instruction }
  if (image) body.image = image
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/refine`,
    body,
    { timeout: 120000 }, // 多模态微调可能较慢，2分钟超时
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
    { timeout: 120000 }, // 单页重生需调用大模型，2分钟超时
  )
  return extractData(resp)
}

/** P0-5: 中途中断批量生成 */
export async function cancelGenerate(coursewareId: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/cancel-generate`)
}

// ==================== 课件索引AI生成（SSE流式） ====================

export async function generateCWIndex(coursewareId: string, preset?: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/generate-index`, preset ? { preset } : {})
}

export function subscribeCWIndexSSE(
  coursewareId: string,
  callbacks: CWSSECallbacks,
): { close: () => void } {
  const token = localStorage.getItem('token') || ''
  const url = `${window.location.origin}/api/v1/sse/courseware/${coursewareId}?token=${encodeURIComponent(token)}`

  const evtSource = new EventSource(url)

  evtSource.addEventListener('connected', (e: MessageEvent) => {
    try { callbacks.onConnected?.(JSON.parse(e.data)) } catch { /* */ }
  })
  evtSource.addEventListener('index_start', (e: MessageEvent) => {
    try { callbacks.onIndexStart?.(JSON.parse(e.data)) } catch { /* */ }
  })
  evtSource.addEventListener('index_page', (e: MessageEvent) => {
    try { callbacks.onIndexPage?.(JSON.parse(e.data)) } catch { /* */ }
  })
  evtSource.addEventListener('index_progress', (e: MessageEvent) => {
    try { callbacks.onIndexProgress?.(JSON.parse(e.data)) } catch { /* */ }
  })
  evtSource.addEventListener('index_done', (e: MessageEvent) => {
    try { callbacks.onIndexDone?.(JSON.parse(e.data)) } catch { /* */ }
    evtSource.close()
  })
  evtSource.addEventListener('gen_start', (e: MessageEvent) => {
    try { callbacks.onGenStart?.(JSON.parse(e.data)) } catch { /* */ }
  })
  evtSource.addEventListener('gen_page', (e: MessageEvent) => {
    try { callbacks.onGenPage?.(JSON.parse(e.data)) } catch { /* */ }
  })
  evtSource.addEventListener('gen_progress', (e: MessageEvent) => {
    try { callbacks.onGenProgress?.(JSON.parse(e.data)) } catch { /* */ }
  })
  evtSource.addEventListener('gen_done', (e: MessageEvent) => {
    try { callbacks.onGenDone?.(JSON.parse(e.data)) } catch { /* */ }
    evtSource.close()
  })
  evtSource.addEventListener('error', (e: MessageEvent) => {
    if (e.data) {
      try { callbacks.onError?.(JSON.parse(e.data)) } catch { /* */ }
    }
    evtSource.close()
  })
  evtSource.onerror = () => {
    evtSource.close()
  }

  return { close: () => evtSource.close() }
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
    timeout: 120000, // PPT上传+解析可能较慢，2分钟超时
  })
  return extractData(resp)
}

/** v0.42 入口B: 从PPT内容生成课件索引（异步，通过SSE推送进度） */
export async function generateCWIndexFromPPT(coursewareId: string, preset?: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/generate-index-ppt`, preset ? { preset } : {})
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
    timeout: 60000,
  })
  return extractData(resp)
}

/** v0.42 入口C: 从Word文档生成课件索引（异步SSE） */
export async function generateCWIndexFromDoc(coursewareId: string, preset?: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/generate-index-doc`, preset ? { preset } : {})
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
