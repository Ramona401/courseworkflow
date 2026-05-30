/**
 * 课件工坊API封装 — coursewares.ts v6.1 (P1修复)
 *
 * v6.1 修复:
 *   - rollbackTemplate: URL 从 .replace hack 改为直接拼接
 *   - 清除冗余的 v139.1 重复追加块(getPublishTargets 等类型已合并到主定义)
 */
import apiClient from './client'

// ==================== 类型定义 ====================

/** 课件列表单条 */
export interface CoursewareListItem {
  id: string
  lesson_plan_id: string
  lesson_plan_title: string
  title: string
  subject: string
  grade: string
  status: string
  status_name: string
  page_count: number
  pipeline_id: string | null
  source_type: string        // v0.42: 来源类型
  source_name: string        // v0.42: 来源类型中文名
  created_at: string
  updated_at: string
}

/** 课件列表响应 */
export interface CoursewareListResponse {
  coursewares: CoursewareListItem[]
  total: number
}

/** 课件详情（Phase 4C: 新增nav_template_html） */
export interface CoursewareDetail {
  id: string
  lesson_plan_id: string
  lesson_plan_title: string
  user_id: string
  title: string
  subject: string
  grade: string
  status: string
  status_name: string
  style_config: string
  page_count: number
  index_overview: string
  logo_url: string
  org_name: string
  nav_template_html: string
  pipeline_id: string | null
  source_type: string       // v0.42: 来源类型
  source_name: string       // v0.42: 来源类型中文名
  pages: CoursewarePage[]
  created_at: string
  updated_at: string
}

/** 课件页面（Phase 3.5: 两层架构） */
export interface CoursewarePage {
  id: string
  courseware_id: string
  page_number: number
  title: string
  purpose: string
  content_summary: string
  interaction_type: string
  visual_format: string
  media_requirements: string
  estimated_complexity: number
  page_index: string
  idx_cognitive_level: number
  idx_interaction_level: number
  idx_visual_format: string
  html_content: string
  placeholder_map: string
  matched_component_ids: string
  status: string
  created_at: string
  updated_at: string
}

/** 课件组件列表单条 */
export interface CWComponentListItem {
  id: string
  name: string
  description: string
  component_type: string
  component_type_name: string
  preview_image_url: string
  subject_scope: string
  grade_scope: string
  component_index: string
  idx_interaction_level: number | null
  is_active: boolean
  review_status: string
  created_at: string
}

/** 课件组件完整（含代码） */
export interface CWComponentFull extends CWComponentListItem {
  code_content: string
  preview_html: string
  idx_visual_format: string
  idx_tech_tag: string
  tech_dependencies: string
  tags: string
}

/** 风格模板（Phase 4A: 新增preview_urls） */
export interface CoursewareTemplate {
  id: string
  name: string
  description: string
  style_category: string
  preview_image_url: string
  color_scheme: string
  css_variables: string
  sample_pages: string
  preview_urls: string
  is_active: boolean
  sort_order: number
  user_id: string | null
  scope: string
  source_courseware_id: string | null
  // v139 新增
  scope_target_id?: string | null
  is_draft?: boolean
  refine_history?: string
  extract_source_meta?: string
  created_at: string
  updated_at: string
}

/** 种子数据填充结果 */
export interface SeedResult {
  components_created: number
  templates_created: number
  errors?: string[]
}

/** SSE事件回调类型（Phase 4C: gen_done新增is_preview标记） */
export interface CWSSECallbacks {
  onConnected?: (data: Record<string, unknown>) => void
  // 索引生成事件
  onIndexStart?: (data: Record<string, unknown>) => void
  onIndexPage?: (page: CoursewarePage) => void
  onIndexProgress?: (data: Record<string, unknown>) => void
  onIndexDone?: (data: { courseware_id: string; page_count: number; message: string }) => void
  // 课件HTML生成事件
  onGenStart?: (data: { courseware_id: string; total_pages: number; template: string; message: string; is_preview?: boolean }) => void
  onGenPage?: (data: { page_number: number; page_id: string; title: string; html_content: string; model_used: string; tokens_used: number }) => void
  onGenProgress?: (data: { current_page: number; total_pages: number; page_title: string; message: string; error?: string }) => void
  onGenDone?: (data: { courseware_id: string; success_count: number; fail_count: number; total_pages: number; elapsed_ms: number; message: string; errors?: string[]; is_preview?: boolean }) => void
  onError?: (data: { message: string }) => void
}

/** 课件状态配置 */
export const CW_STATUS_CONFIG: Record<string, { label: string; color: string; bg: string }> = {
  draft:       { label: '草稿',       color: '#6B7280', bg: '#F3F4F6' },
  indexing:    { label: '方案编辑中', color: '#D97706', bg: '#FEF3C7' },
  styling:     { label: '风格选择中', color: '#7C3AED', bg: '#EDE9FE' },
  generating:  { label: '课件生成中', color: '#2563EB', bg: '#DBEAFE' },
  preview:     { label: '预览确认中', color: '#0891B2', bg: '#CFFAFE' },
  confirmed:   { label: '已确认',     color: '#059669', bg: '#D1FAE5' },
  in_pipeline: { label: '审核中',     color: '#4F46E5', bg: '#E0E7FF' },
}

/** 组件类型配色 */
export const CW_COMP_TYPE_CONFIG: Record<string, { label: string; color: string; bg: string }> = {
  layout:      { label: '布局模板',   color: '#2563EB', bg: '#DBEAFE' },
  interaction: { label: '交互功能',   color: '#059669', bg: '#D1FAE5' },
  '3d':        { label: '3D/动画',    color: '#7C3AED', bg: '#EDE9FE' },
  animation:   { label: '动画效果',   color: '#DB2777', bg: '#FCE7F3' },
  data_viz:    { label: '数据可视化', color: '#0891B2', bg: '#CFFAFE' },
  multimedia:  { label: '多媒体容器', color: '#D97706', bg: '#FEF3C7' },
  style:       { label: '样式主题',   color: '#4F46E5', bg: '#E0E7FF' },
}

/** 风格类别配色（Phase 4A: 新增immersive） */
export const CW_STYLE_CONFIG: Record<string, { label: string; color: string; bg: string; emoji: string }> = {
  minimalist: { label: '简约清新', color: '#2563EB', bg: '#DBEAFE', emoji: '✨' },
  playful:    { label: '活泼趣味', color: '#F59E0B', bg: '#FEF3C7', emoji: '🎈' },
  tech:       { label: '科技感',   color: '#7C3AED', bg: '#EDE9FE', emoji: '🔬' },
  academic:   { label: '学术严谨', color: '#1F2937', bg: '#F3F4F6', emoji: '📖' },
  organic:    { label: '自然有机', color: '#059669', bg: '#D1FAE5', emoji: '🌿' },
  immersive:  { label: '3D沉浸式', color: '#DC2626', bg: '#FEE2E2', emoji: '🎮' },
}

/** 交互类型标签 */
export const CW_INTERACTION_TYPES: Record<string, { label: string; emoji: string }> = {
  static:    { label: '静态展示', emoji: '📄' },
  click:     { label: '点击交互', emoji: '👆' },
  drag:      { label: '拖拽操作', emoji: '✋' },
  input:     { label: '输入填写', emoji: '✏️' },
  animation: { label: '动画演示', emoji: '🎬' },
  video:     { label: '视频播放', emoji: '📹' },
  game:      { label: '游戏互动', emoji: '🎮' },
  quiz:      { label: '答题测验', emoji: '❓' },
}

/** 视觉形式标签 */
export const CW_VISUAL_FORMATS: Record<string, { label: string; emoji: string }> = {
  text_heavy:       { label: '文字为主', emoji: '📝' },
  image_text:       { label: '图文混排', emoji: '🖼️' },
  diagram:          { label: '示意图',   emoji: '📊' },
  chart:            { label: '图表',     emoji: '📈' },
  timeline:         { label: '时间线',   emoji: '⏳' },
  comparison:       { label: '对比展示', emoji: '⚖️' },
  gallery:          { label: '图片画廊', emoji: '🎨' },
  fullscreen_media: { label: '全屏媒体', emoji: '🖥️' },
}

/** 认知层次标签（布鲁姆） */
export const CW_COGNITIVE_LEVELS: Record<number, { label: string; color: string; bg: string }> = {
  1: { label: '记忆', color: '#059669', bg: '#D1FAE5' },
  2: { label: '理解', color: '#0891B2', bg: '#CFFAFE' },
  3: { label: '应用', color: '#2563EB', bg: '#DBEAFE' },
  4: { label: '分析', color: '#D97706', bg: '#FEF3C7' },
  5: { label: '评价', color: '#DC2626', bg: '#FEE2E2' },
  6: { label: '创造', color: '#7C3AED', bg: '#EDE9FE' },
}

// ==================== 通用提取函数 ====================

function extractData<T>(resp: { data?: { code?: number; data?: T } }): T {
  const d = resp?.data
  if (d && d.code === 0 && d.data !== undefined) return d.data
  throw new Error('接口返回异常')
}

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

/** P0-4: 单页AI微调 */
export async function refinePage(coursewareId: string, pageNumber: number, instruction: string): Promise<{ page_number: number; html_content: string; message: string }> {
  const resp = await apiClient.post(`/coursewares/${coursewareId}/pages/${pageNumber}/refine`, { instruction })
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

/** 方案结构预设类型 */
export interface SchemePreset {
  key: string
  name: string
  emoji: string
  description: string
  grade_hint: string
  page_range: string
}

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


// ==================== v139: AI 提取 + 微调 + 发布 ====================

/** AI 提取响应 */
export interface ExtractTemplateResponse {
  template_id: string
  suggested_name: string
  suggested_desc: string
  suggested_category: string
  extraction_notes: string
  message: string
}

/** 微调历史条目 */
export interface RefineHistoryEntry {
  timestamp: string
  user_instruction: string
  sample_pages_before: string[]
  css_variables_before: string
  color_scheme_before: string
  change_summary: string
}

/** 微调SSE回调 */
export interface RefineSSECallbacks {
  onStart?: (d: { template_id: string; instruction: string; message: string }) => void
  onChunk?: (d: { chunk_no: number; message: string }) => void
  onProgress?: (d: { message: string }) => void
  onDone?: (d: {
    template_id: string
    color_scheme: Record<string, string>
    css_variables: Record<string, string>
    sample_pages: string[]
    style_category: string
    change_summary: string
    message: string
  }) => void
  onError?: (d: { message: string }) => void
}

/** v145: AI 提取风格模板(异步启动,通过 SSE 推送进度) */
export async function extractTemplateFromHTML(samplePages: string[], sourceType = 'paste'): Promise<void> {
  await apiClient.post('/coursewares/templates/extract',
    { sample_pages: samplePages, source_type: sourceType },
  )
}

/** v145: 提取 SSE 回调类型 */
export interface ExtractSSECallbacks {
  onStart?: (d: { message: string }) => void
  onProgress?: (d: { message: string; stage?: string; elapsed_sec?: number }) => void
  onDone?: (d: {
    template_id: string; suggested_name: string; suggested_desc: string
    suggested_category: string; extraction_notes: string; message: string
  }) => void
  onError?: (d: { message: string }) => void
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


// ==================== v139.1: 发布目标查询 ====================

/** 学校发布目标:用户是该学校的管理员时 available=true */
export interface PublishTargetSchool {
  available: boolean
  school_id: string
  name: string
}

/** 教研组发布目标:用户在此组担任 lead 或 backbone */
export interface PublishTargetGroup {
  id: string
  name: string
  school_name: string
  role: 'lead' | 'backbone'
}

/** 发布目标聚合响应 - 前端据此渲染发布表单的下拉选项 */
export interface PublishTargetsResponse {
  personal: { available: boolean }
  system: { available: boolean; reason?: string }
  school: PublishTargetSchool
  groups: PublishTargetGroup[]
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


// ==================== v0.42 入口B: PPT上传创建 ====================

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


// ==================== v0.42 入口C: Word文档上传创建 ====================

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


// ==================== v0.42 多媒体: 图片生成+上传+管理 ====================

/** 课件图片资产 */
export interface CoursewareAsset {
  id: string
  courseware_id: string
  page_id: string | null
  placeholder_id: string
  asset_type: string
  generation_prompt: string
  oss_url: string
  file_size: number
  mime_type: string
  status: string
  created_at: string
}

/** AI生成图片响应 */
export interface GenerateImageResponse {
  asset_id: string
  url: string
  original_urls: string[]
  model_used: string
  revised_prompt: string
}

/** 手动上传图片响应 */
export interface UploadImageResponse {
  asset_id: string
  url: string
  file_name: string
  file_size: number
  mime_type: string
}

/** v0.42: AI生成图片（调用豆包Seedream API） */
export async function generateCWImage(
  coursewareId: string,
  pageNumber: number,
  prompt: string,
  placeholderId?: string,
  size?: string,
  refImageUrl?: string,
): Promise<GenerateImageResponse> {
  const body: Record<string, string> = {
    prompt,
    placeholder_id: placeholderId || '',
    size: size || '1920x1920',
  }
  if (refImageUrl) body.ref_image_url = refImageUrl
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/generate-image`,
    body,
    { timeout: 60000 },
  )
  return extractData(resp)
}

/** v0.42: 手动上传图片 */
export async function uploadCWImage(
  coursewareId: string,
  pageNumber: number,
  file: File,
  placeholderId?: string,
): Promise<UploadImageResponse> {
  const formData = new FormData()
  formData.append('file', file)
  if (placeholderId) formData.append('placeholder_id', placeholderId)
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/upload-image`,
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' }, timeout: 30000 },
  )
  return extractData(resp)
}

/** v0.42: 获取页面图片列表 */
export async function listPageAssets(
  coursewareId: string,
  pageNumber: number,
): Promise<{ assets: CoursewareAsset[]; total: number }> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/pages/${pageNumber}/assets`,
  )
  return extractData(resp)
}

/** v0.42: 获取课件全部图片 */
export async function listCoursewareAssets(
  coursewareId: string,
): Promise<{ assets: CoursewareAsset[]; total: number }> {
  const resp = await apiClient.get(`/coursewares/${coursewareId}/assets`)
  return extractData(resp)
}

/** v0.42: 删除图片资产 */
export async function deleteCWAsset(
  coursewareId: string,
  assetId: string,
): Promise<void> {
  await apiClient.delete(`/coursewares/${coursewareId}/assets/${assetId}`)
}

/** v0.42: 将图片插入到页面HTML */
export async function insertImageToPage(
  coursewareId: string,
  pageNumber: number,
  assetId: string,
): Promise<{ page_number: number; html_content: string; message: string }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/insert-image`,
    { asset_id: assetId },
  )
  return extractData(resp)
}


// ==================== v0.42.1 视频生成 ====================

/** AI视频生成任务提交响应 */
export interface GenerateVideoResponse {
  asset_id: string
  task_id: string
  model_used: string
  message: string
}

/** 视频任务状态查询响应 */
export interface VideoStatusResponse {
  asset_id: string
  task_id: string
  status: 'generating' | 'uploaded' | 'failed'
  video_url: string
  duration: number
  resolution: string
  ratio: string
  error_msg: string
  message: string
}

/** v0.42.1: AI生成视频（异步提交任务，返回task_id供轮询） */
export async function generateCWVideo(
  coursewareId: string,
  pageNumber: number,
  prompt: string,
  refImageUrl?: string,
): Promise<GenerateVideoResponse> {
  const body: Record<string, string> = { prompt }
  if (refImageUrl) body.ref_image_url = refImageUrl
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/generate-video`,
    body,
    { timeout: 30000 },
  )
  return extractData(resp)
}

/** v0.42.1: 查询视频生成任务状态（前端轮询直到uploaded或failed） */
export async function queryVideoStatus(
  coursewareId: string,
  assetId: string,
): Promise<VideoStatusResponse> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/assets/${assetId}/video-status`,
  )
  return extractData(resp)
}


// ==================== v0.42.1 视频编辑（拼接+裁剪） ====================




/** 视频片段配置（高级拼接） */
export interface VideoClip {
  asset_id: string
  start_sec: number
  end_sec: number
  transition?: string
  trans_dur?: number
}

/** 高级拼接响应 */
export interface AdvancedConcatResponse {
  asset_id: string
  url: string
  duration: string
  message: string
}

/** v0.42.2: 高级视频拼接（支持每段独立裁剪+转场效果） */
export async function advancedConcatCWVideos(
  coursewareId: string,
  clips: VideoClip[],
): Promise<AdvancedConcatResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/advanced-concat`,
    { clips },
    { timeout: 120000 },
  )
  return extractData(resp)
}


// ==================== v0.42.4 视频静音 + 音轨分离 ====================

/** 视频静音响应 */
export interface MuteVideoResponse {
  asset_id: string
  url: string
  duration: string
  message: string
}

/** 音轨分离响应 */
export interface ExtractAudioResponse {
  asset_id: string
  url: string
  duration: string
  format: string
  file_size: number
  message: string
}

/** v0.42.4: 视频静音（去除音轨，生成新的静音视频） */
export async function muteCWVideo(
  coursewareId: string,
  assetId: string,
): Promise<MuteVideoResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/mute`,
    { asset_id: assetId },
    { timeout: 60000 },
  )
  return extractData(resp)
}

/** v0.42.4: 音轨分离（从视频提取音频为MP3） */
export async function extractCWAudio(
  coursewareId: string,
  assetId: string,
): Promise<ExtractAudioResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/extract-audio`,
    { asset_id: assetId },
    { timeout: 60000 },
  )
  return extractData(resp)
}


// ==================== v0.42.5 视频手动上传 ====================

/** v0.42.5: 手动上传视频文件到课件(mp4/webm/mov ≤50MB) */
export async function uploadCWVideo(
  coursewareId: string,
  pageNumber: number,
  file: File,
  onProgress?: (pct: number) => void,
): Promise<{ asset_id: string; url: string; file_name: string; file_size: number; mime_type: string }> {
  const formData = new FormData()
  formData.append('file', file)
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/upload-video`,
    formData,
    {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 120000,
      // v0.42.6+ P1.1: 利用 axios 原生上传进度回调
      onUploadProgress: (e) => {
        if (onProgress && e.total) {
          const pct = Math.round((e.loaded / e.total) * 100)
          onProgress(Math.min(100, pct))
        }
      },
    },
  )
  return extractData(resp)
}


// ==================== v0.42.5 视频编辑器草稿(服务器端多版本) ====================

export interface VideoDraftItem {
  id: string
  courseware_id: string
  name: string
  clips_data: any
  clip_count: number
  created_at: string
}

export async function listVideoDrafts(coursewareId: string): Promise<{ drafts: VideoDraftItem[]; total: number }> {
  const resp = await apiClient.get(`/coursewares/${coursewareId}/video-drafts`)
  return extractData(resp)
}

export async function saveVideoDraft(coursewareId: string, data: {
  name: string; clips_data: any; clip_count: number
}): Promise<{ id: string; created_at: string; message: string }> {
  const resp = await apiClient.post(`/coursewares/${coursewareId}/video-drafts`, data)
  return extractData(resp)
}

export async function deleteVideoDraft(coursewareId: string, draftId: string): Promise<void> {
  await apiClient.delete(`/coursewares/${coursewareId}/video-drafts/${draftId}`)
}


// ==================== v0.42.8 字幕轨 API ====================

/** 字幕片段 */
export interface SubtitleSegment {
  id: string
  start_sec: number
  end_sec: number
  text: string
  language: string
  tts_audio_url?: string
  tts_voice?: string
  tts_duration?: number
  tts_generated_at?: string
}

/** 字幕轨记录 */
export interface CoursewareSubtitle {
  id: string
  courseware_id: string
  scope_type: string
  scope_id: string | null
  language: string
  segments: string  // JSON 字符串（SubtitleSegment[]）
  style_config: string | null
  tts_config: string | null
  created_by: string | null
  created_at: string
  updated_at: string
}

/** 字幕烧录响应 */
export interface BurnInSubtitleResponse {
  asset_id: string
  url: string
  duration: string
  message: string
}

/** v0.42.8: 创建/更新字幕轨（UPSERT by courseware+scope+language） */
export async function upsertSubtitle(
  coursewareId: string,
  data: {
    scope_type: string
    scope_id?: string | null
    language: string
    segments: string
    style_config?: string | null
    tts_config?: string | null
  },
): Promise<CoursewareSubtitle> {
  const resp = await apiClient.post(`/coursewares/${coursewareId}/subtitles`, data)
  return extractData(resp)
}

/** v0.42.8: 查询字幕轨列表 */
export async function listSubtitles(
  coursewareId: string,
  scopeType?: string,
  scopeId?: string,
): Promise<CoursewareSubtitle[]> {
  const params: Record<string, string> = {}
  if (scopeType) params.scope_type = scopeType
  if (scopeId) params.scope_id = scopeId
  const resp = await apiClient.get(`/coursewares/${coursewareId}/subtitles`, { params })
  return extractData(resp)
}

/** v0.42.8: 删除字幕轨 */
export async function deleteSubtitle(
  coursewareId: string,
  subtitleId: string,
): Promise<void> {
  await apiClient.delete(`/coursewares/${coursewareId}/subtitles/${subtitleId}`)
}

/** v0.42.8: 前端本地生成 SRT 文件并触发下载（避免 axios 拦截器对纯文本的处理问题） */
export function exportSubtitleSRTLocal(segments: SubtitleSegment[], filename?: string): void {
  // 格式化 SRT 时间码: HH:MM:SS,mmm
  const fmtTime = (sec: number): string => {
    if (sec < 0) sec = 0
    const ms = Math.round(sec * 1000)
    const h = Math.floor(ms / 3600000)
    const m = Math.floor((ms % 3600000) / 60000)
    const s = Math.floor((ms % 60000) / 1000)
    const mill = ms % 1000
    return `${String(h).padStart(2,'0')}:${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')},${String(mill).padStart(3,'0')}`
  }
  const srt = segments.map((seg, i) =>
    `${i + 1}\n${fmtTime(seg.start_sec)} --> ${fmtTime(seg.end_sec)}\n${seg.text}\n`
  ).join('\n')
  const blob = new Blob([srt], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename || 'subtitle.srt'
  document.body.appendChild(a); a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/** v0.42.8: FFmpeg 硬字幕烧录（生成新视频） */
export async function burnInSubtitle(
  coursewareId: string,
  subtitleId: string,
  videoAssetId: string,
): Promise<BurnInSubtitleResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/subtitles/${subtitleId}/burn-in`,
    { video_asset_id: videoAssetId },
    { timeout: 120000 },
  )
  return extractData(resp)
}


// ==================== v0.42.9 TTS 配音 API ====================

/** TTS 音色定义 */
export interface TTSVoice {
  code: string
  name: string
  language: string
  gender: string
  style: string
}

/** TTS 配音请求 */
export interface GenerateTTSRequest {
  voice: string
  speed?: number
  segment_ids?: string[]
}

/** TTS 配音响应 */
export interface GenerateTTSResponse {
  subtitle_id: string
  success_count: number
  fail_count: number
  total_count: number
  segments: string
  errors: string[]
  message: string
}

/** v0.42.9: 获取可用 TTS 音色列表 */
export async function listTTSVoices(language?: string): Promise<{ voices: TTSVoice[]; total: number }> {
  const params: Record<string, string> = {}
  if (language) params.language = language
  const resp = await apiClient.get('/tts-voices', { params })
  return extractData(resp)
}

/** v0.42.9: 批量生成字幕 TTS 配音 */
export async function generateSubtitleTTS(
  coursewareId: string,
  subtitleId: string,
  voice: string,
  speed?: number,
  segmentIds?: string[],
): Promise<GenerateTTSResponse> {
  const body: GenerateTTSRequest = { voice }
  if (speed && speed !== 1.0) body.speed = speed
  if (segmentIds && segmentIds.length > 0) body.segment_ids = segmentIds
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/subtitles/${subtitleId}/generate-tts`,
    body,
    { timeout: 300000 }, // 5分钟超时（批量TTS可能较慢）
  )
  return extractData(resp)
}


// ==================== v0.42.10 上传资产到阿里云OSS ====================

/** 上传资产到阿里云OSS的响应 */
export interface UploadToOSSResponse {
  asset_id: string
  local_url: string
  oss_public_url: string
  message: string
}

/** v0.42.10: 将课件资产（图片/视频/音频）上传到阿里云OSS，返回公网URL */
export async function uploadAssetToOSS(
  coursewareId: string,
  assetId: string,
): Promise<UploadToOSSResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/assets/${assetId}/upload-oss`,
    {},
    { timeout: 120000 }, // 大视频上传可能较慢，2分钟超时
  )
  return extractData(resp)
}


// ==================== v0.42.11 3D互动单页生成 ====================

/** v0.42.11: 触发3D互动单页生成（异步，通过SSE推送进度） */
export async function generate3DPage(coursewareId: string): Promise<{ message: string; courseware_id: string }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/generate-3d-page`,
    {},
    { timeout: 180000 }, // 3分钟超时（3D生成较慢）
  )
  return extractData(resp)
}


// ==================== v0.42.11 创建3D互动单页课件 ====================

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


// ==================== 离线打包下载 ====================

/**
 * 下载课件离线包(zip)
 * 用原生 fetch 获取二进制流，绕开 axios 响应拦截器对非 JSON(blob) 的处理。
 * 鉴权头与 client.ts 保持一致：Authorization: Bearer <token>
 */
export async function downloadCoursewareBundle(coursewareId: string, title?: string): Promise<void> {
  const token = localStorage.getItem('token') || ''
  const resp = await fetch(`/api/v1/coursewares/${coursewareId}/export-bundle`, {
    method: 'GET',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!resp.ok) {
    // 尝试解析后端 JSON 错误信息
    let msg = `下载失败(HTTP ${resp.status})`
    try {
      const j = await resp.json()
      if (j && j.message) msg = j.message
    } catch { /* 非 JSON 错误体，忽略 */ }
    if (resp.status === 401) {
      // 登录态失效，与 client.ts 行为保持一致
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      if (window.location.pathname !== '/login') window.location.href = '/login'
    }
    throw new Error(msg)
  }
  // 文件名：优先用后端 Content-Disposition，回退到 title.zip
  let filename = (title ? title.trim() : '') || 'courseware'
  filename = filename.replace(/[/\\:*?"<>|]/g, '_') + '.zip'
  const cd = resp.headers.get('Content-Disposition') || ''
  const m = cd.match(/filename\*=UTF-8''([^;]+)/i)
  if (m && m[1]) {
    try { filename = decodeURIComponent(m[1]) } catch { /* 解码失败用回退名 */ }
  }
  // 触发浏览器下载
  const blob = await resp.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

