/**
 * 课件工坊API封装 — coursewares.ts v4.0 (Phase 3.5)
 *
 * 课件CRUD + 页面操作 + 状态流转 + 风格模板 + 组件库
 * Phase 2: 种子数据填充 + admin模板管理
 * Phase 3: 课件索引AI生成（SSE流式）+ 删除单页
 * Phase 3.5: 两层索引架构 — CoursewarePage新增page_index+3个冗余列
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
  created_at: string
  updated_at: string
}

/** 课件列表响应 */
export interface CoursewareListResponse {
  coursewares: CoursewareListItem[]
  total: number
}

/** 课件详情 */
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
  pipeline_id: string | null
  pages: CoursewarePage[]
  created_at: string
  updated_at: string
}

/** 课件页面（Phase 3.5: 两层架构） */
export interface CoursewarePage {
  id: string
  courseware_id: string
  page_number: number
  // 层2：用户友好方案字段
  title: string
  purpose: string
  content_summary: string
  interaction_type: string
  visual_format: string
  media_requirements: string
  estimated_complexity: number
  // 层1：AOCI技术索引（admin可见）
  page_index: string
  idx_cognitive_level: number
  idx_interaction_level: number
  idx_visual_format: string
  // 生成相关
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

/** 风格模板 */
export interface CoursewareTemplate {
  id: string
  name: string
  description: string
  style_category: string
  preview_image_url: string
  color_scheme: string
  css_variables: string
  sample_pages: string
  is_active: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

/** 种子数据填充结果 */
export interface SeedResult {
  components_created: number
  templates_created: number
  errors?: string[]
}

/** Phase 3: SSE事件回调类型 */
export interface CWSSECallbacks {
  onConnected?: (data: Record<string, unknown>) => void
  onIndexStart?: (data: Record<string, unknown>) => void
  onIndexPage?: (page: CoursewarePage) => void
  onIndexProgress?: (data: Record<string, unknown>) => void
  onIndexDone?: (data: { courseware_id: string; page_count: number; message: string }) => void
  onError?: (data: { message: string }) => void
}

/** 课件状态配置（前端用，Phase 3.5: indexing改为"方案编辑中"） */
export const CW_STATUS_CONFIG: Record<string, { label: string; color: string; bg: string }> = {
  draft:       { label: '草稿',       color: '#6B7280', bg: '#F3F4F6' },
  indexing:    { label: '方案编辑中', color: '#D97706', bg: '#FEF3C7' },
  styling:     { label: '风格选择中', color: '#7C3AED', bg: '#EDE9FE' },
  generating:  { label: '课件生成中', color: '#2563EB', bg: '#DBEAFE' },
  preview:     { label: '预览确认中', color: '#0891B2', bg: '#CFFAFE' },
  confirmed:   { label: '已确认',     color: '#059669', bg: '#D1FAE5' },
  in_pipeline: { label: '审核中',     color: '#4F46E5', bg: '#E0E7FF' },
}

/** 组件类型配色（前端用） */
export const CW_COMP_TYPE_CONFIG: Record<string, { label: string; color: string; bg: string }> = {
  layout:      { label: '布局模板',   color: '#2563EB', bg: '#DBEAFE' },
  interaction: { label: '交互功能',   color: '#059669', bg: '#D1FAE5' },
  '3d':        { label: '3D/动画',    color: '#7C3AED', bg: '#EDE9FE' },
  animation:   { label: '动画效果',   color: '#DB2777', bg: '#FCE7F3' },
  data_viz:    { label: '数据可视化', color: '#0891B2', bg: '#CFFAFE' },
  multimedia:  { label: '多媒体容器', color: '#D97706', bg: '#FEF3C7' },
  style:       { label: '样式主题',   color: '#4F46E5', bg: '#E0E7FF' },
}

/** 风格类别配色（前端用） */
export const CW_STYLE_CONFIG: Record<string, { label: string; color: string; bg: string; emoji: string }> = {
  minimalist: { label: '简约清新', color: '#2563EB', bg: '#DBEAFE', emoji: '✨' },
  playful:    { label: '活泼趣味', color: '#F59E0B', bg: '#FEF3C7', emoji: '🎈' },
  tech:       { label: '科技感',   color: '#7C3AED', bg: '#EDE9FE', emoji: '🔬' },
  academic:   { label: '学术严谨', color: '#1F2937', bg: '#F3F4F6', emoji: '📖' },
  organic:    { label: '自然有机', color: '#059669', bg: '#D1FAE5', emoji: '🌿' },
}

/** 交互类型标签（前端用） */
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

/** 视觉形式标签（前端用） */
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

/** 认知层次标签（Phase 3.5新增，布鲁姆） */
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

/** Phase 3: 删除课件单页 */
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

export async function saveCWStyle(coursewareId: string, styleConfig: string): Promise<void> {
  await apiClient.put(`/coursewares/${coursewareId}/style`, { style_config: styleConfig })
}

export async function confirmCourseware(coursewareId: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/confirm`)
}

// ==================== Phase 3: 课件索引AI生成（SSE流式） ====================

/** 触发AI生成课件索引（异步，返回后通过SSE监听进度） */
export async function generateCWIndex(coursewareId: string): Promise<void> {
  await apiClient.post(`/coursewares/${coursewareId}/generate-index`)
}

/**
 * 订阅课件索引生成SSE事件流
 * 返回 { close } 句柄，组件卸载时调用 close() 断开连接
 */
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

// ==================== Phase 2: 种子数据填充（admin） ====================

export async function seedCoursewareData(force?: boolean): Promise<SeedResult> {
  const resp = await apiClient.post('/admin/courseware-seed', { force: !!force })
  return extractData(resp)
}

// ==================== Phase 2: Admin模板管理 ====================

export async function createCWTemplate(data: {
  name: string; description?: string; style_category: string
  color_scheme?: string; css_variables?: string; sample_pages?: string
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
