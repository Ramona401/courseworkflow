/**
 * 课件工坊 API —— 类型定义与常量配置 (coursewares.types.ts)
 *
 * 从原 coursewares.ts 拆出的纯类型层：所有 interface/type + UI 常量配置表。
 * 不含任何运行时请求逻辑，供 coursewares.core.ts / coursewares.media.ts 及
 * 业务组件统一 import。对外经桶文件 coursewares.ts 用 `export *` 透出，
 * 既有 `import { X } from '@/api/coursewares'` 路径完全不变。
 */

// ==================== 课件主体类型 ====================

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
  // 风格锚点字段（轮2后端已返回，前端读当前锚点状态）
  style_anchor_asset_id: string | null  // null=未设锚点
  style_anchor_vaoci: string            // 锚点VAOCI索引文本，空=未设
  style_anchor_url: string              // 锚点图公网URL（轮3：跨页显示缩略图，优先OSS地址）
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

// ==================== 组件库类型 ====================

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

// ==================== 风格模板类型 ====================

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

/** 提取 SSE 回调类型 */
export interface ExtractSSECallbacks {
  onStart?: (d: { message: string }) => void
  onProgress?: (d: { message: string; stage?: string; elapsed_sec?: number }) => void
  onDone?: (d: {
    template_id: string; suggested_name: string; suggested_desc: string
    suggested_category: string; extraction_notes: string; message: string
  }) => void
  onError?: (d: { message: string }) => void
}

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

// ==================== 方案预设类型 ====================

/** 方案结构预设类型 */
export interface SchemePreset {
  key: string
  name: string
  emoji: string
  description: string
  grade_hint: string
  page_range: string
}

// ==================== SSE 回调类型 ====================

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

// ==================== 多媒体资产类型 ====================

/** 课件图片资产 */
export interface CoursewareAsset {
  id: string
  courseware_id: string
  page_id: string | null
  placeholder_id: string
  asset_type: string
  generation_prompt: string
  oss_url: string
  public_oss_url?: string   // v0.42.11: 上传OSS成功后回写的公网URL，非空表示已上云（删除时需连带删OSS副本）
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

/** 图片多提示词: AI 为本页产出的单条配图建议(一张图) */
export interface ImagePromptSuggestion {
  caption: string   // 该图用途的简短说明(供老师区分多条分别对应什么), 可能为空串
  prompt: string    // 该图的详细中文生图提示词正文
}

/** 视频分镜物料: AI 为本页某个镜头产出的一组提示词(本轮: 视频提示词由单组改为按分镜数组返回) */
export interface VideoStoryboardItem {
  scene: string         // 本镜环节简述(8-20字, 供老师区分各镜)
  image_prompt: string  // 本镜首帧分镜图提示词
  video_prompt: string  // 本镜图生视频提示词
  narration: string     // 本镜口播台词
  // v0.42.x 首帧关联持久化: 老师为本镜生成首帧图后, 把图资产ID与URL随分镜一并存库;
  // 刷新/重进时回填, 恢复"已生成首帧图"缩略图与"可生视频"就绪态。AI首次产出时这两字段为空(此刻尚无首帧)。
  frame_asset_id?: string  // 本镜已选首帧图的资产ID, 空=尚未生成/关联首帧
  frame_url?: string       // 本镜首帧图URL(优先公网OSS地址), 供回填时直接显示缩略图
}

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

/** 视频编辑器草稿项 */
export interface VideoDraftItem {
  id: string
  courseware_id: string
  name: string
  clips_data: any
  clip_count: number
  created_at: string
}

// ==================== 字幕/TTS 类型 ====================

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

/** 上传资产到阿里云OSS的响应 */
export interface UploadToOSSResponse {
  asset_id: string
  local_url: string
  oss_public_url: string
  message: string
}

/** 风格锚点：设置成功后的返回（轮2/轮3） */
export interface SetStyleAnchorResult {
  asset_id: string    // 设为锚点的图片资产ID
  anchor_url: string  // 锚点图公网URL（供前端展示缩略图）
  vaoci: string       // AI提取的VAOCI风格索引文本
}

// ==================== UI 常量配置表 ====================

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

/** 统一从 axios 响应里取 {code:0,data} 的 data，异常即抛 */
export function extractData<T>(resp: { data?: { code?: number; data?: T } }): T {
  const d = resp?.data
  if (d && d.code === 0 && d.data !== undefined) return d.data
  throw new Error('接口返回异常')
}
