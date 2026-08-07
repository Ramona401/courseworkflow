/**
 * 课件工坊 API —— 类型定义与常量配置 (coursewares.types.ts)
 *
 * 从原 coursewares.ts 拆出的纯类型层：所有interface/type与UI常量配置表。
 * 不含任何运行时请求逻辑，供coursewares.core.ts、coursewares.media.ts及
 * 业务组件统一import。对外经桶文件coursewares.ts用export *透出，
 * 既有import路径保持不变。
 *
 * 图片成本隐私：
 *   普通图片生成响应只包含平台资产ID、平台可用URL和安全修订提示词；
 *   不声明、不依赖供应商原始URL与真实模型名称。
 */

import type {
  EducationDomain,
  ResourceEducationDomain,
} from '@/education-domain/types'

type TeachingEducationDomain =
  Exclude<EducationDomain, 'mixed'>

// ==================== 课件主体类型 ====================

/** 课件列表单条 */
export interface CoursewareListItem {
  id: string
  lesson_plan_id: string
  lesson_plan_title: string
  title: string
  subject: string
  grade: string
  education_domain: TeachingEducationDomain
  status: string
  status_name: string
  page_count: number
  pipeline_id: string | null
  source_type: string
  source_name: string
  publish_state: string
  publish_state_name: string
  review_level: number
  code_share_scope: string
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
  education_domain: TeachingEducationDomain
  status: string
  status_name: string
  style_config: string
  page_count: number
  index_overview: string
  logo_url: string
  org_name: string
  nav_template_html: string
  pipeline_id: string | null
  source_type: string
  source_name: string
  style_anchor_asset_id: string | null
  style_anchor_vaoci: string
  style_anchor_url: string
  kp_codes?: string
  publish_state?: string
  publish_state_name?: string
  review_level?: number
  review_school_id?: string | null
  code_share_scope?: string
  collab_state?: string
  collab_member_count?: number
  pages: CoursewarePage[]
  created_at: string
  updated_at: string
}

/** 课件页面 */
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

// ==================== 共享课件库类型 ====================

/** 共享课件库列表单条 */
export interface SharedCoursewareListItem {
  id: string
  title: string
  subject: string
  grade: string
  education_domain: TeachingEducationDomain
  page_count: number
  source_type: string
  source_name: string
  author_id: string
  author_name: string
  school_name: string
  publish_state: string
  publish_state_name: string
  code_share_scope: string
  can_copy: boolean
  created_at: string
  updated_at: string
}

/** 共享课件库列表响应 */
export interface SharedCoursewareListResponse {
  coursewares: SharedCoursewareListItem[]
  total: number
}

// ==================== 组件库类型 ====================

/** 课件组件列表单条 */
export interface CWComponentListItem {
  id: string
  education_domain: ResourceEducationDomain
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

/** 课件组件完整记录 */
export interface CWComponentFull
  extends CWComponentListItem {
  code_content: string
  preview_html: string
  idx_visual_format: string
  idx_tech_tag: string
  tech_dependencies: string
  tags: string
}

// ==================== 风格模板类型 ====================

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
  preview_urls: string
  is_active: boolean
  sort_order: number
  user_id: string | null
  scope: string
  source_courseware_id: string | null
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
  templates_skipped?: string
  errors?: string[]
}

/** AI提取响应 */
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
  onStart?: (
    data: {
      template_id: string
      instruction: string
      message: string
    },
  ) => void

  onChunk?: (
    data: {
      chunk_no: number
      message: string
    },
  ) => void

  onProgress?: (
    data: {
      message: string
    },
  ) => void

  onDone?: (
    data: {
      template_id: string
      color_scheme: Record<string, string>
      css_variables: Record<string, string>
      sample_pages: string[]
      style_category: string
      change_summary: string
      message: string
    },
  ) => void

  onError?: (
    data: {
      message: string
    },
  ) => void
}

/** 提取SSE回调 */
export interface ExtractSSECallbacks {
  onStart?: (
    data: {
      message: string
    },
  ) => void

  onProgress?: (
    data: {
      message: string
      stage?: string
      elapsed_sec?: number
    },
  ) => void

  onDone?: (
    data: {
      template_id: string
      suggested_name: string
      suggested_desc: string
      suggested_category: string
      extraction_notes: string
      message: string
    },
  ) => void

  onError?: (
    data: {
      message: string
    },
  ) => void
}

/** 学校发布目标 */
export interface PublishTargetSchool {
  available: boolean
  school_id: string
  name: string
}

/** 教研组发布目标 */
export interface PublishTargetGroup {
  id: string
  name: string
  school_name: string
  role: 'lead' | 'backbone'
}

/** 发布目标聚合响应 */
export interface PublishTargetsResponse {
  personal: {
    available: boolean
  }

  system: {
    available: boolean
    reason?: string
  }

  school: PublishTargetSchool
  groups: PublishTargetGroup[]
}

// ==================== 方案预设类型 ====================

/** 方案结构预设 */
export interface SchemePreset {
  key: string
  name: string
  emoji: string
  description: string
  grade_hint: string
  page_range: string
}

// ==================== 课程知识库类型 ====================

/** 课标知识点 */
export interface CurriculumKP {
  id: string
  subject: string
  stage: string
  grade_num: number
  domain: string
  theme: string
  kp_code: string
  kp_name: string
  content_requirement: string
  academic_requirement: string
  teaching_hint: string
  depth_level: number
  core_competency: string
  source_ref: string
  confidence: number
  sort_order: number
}

/** 知识点查询响应 */
export interface CurriculumKPResponse {
  subject: string
  grade: number
  knowledge_points: CurriculumKP[] | null
  total: number
}

/** 难度档UI配置 */
export const CW_DEPTH_LEVEL_CONFIG:
Record<number, {
  label: string
  color: string
  bg: string
}> = {
  1: {
    label: '体验感知',
    color: '#059669',
    bg: '#D1FAE5',
  },
  2: {
    label: '理解应用',
    color: '#2563EB',
    bg: '#DBEAFE',
  },
  3: {
    label: '分析迁移',
    color: '#7C3AED',
    bg: '#EDE9FE',
  },
}

// ==================== SSE回调类型 ====================

/** 课件SSE事件回调 */
export interface CWSSECallbacks {
  onConnected?: (
    data: Record<string, unknown>,
  ) => void

  onIndexStart?: (
    data: Record<string, unknown>,
  ) => void

  onIndexPage?: (
    page: CoursewarePage,
  ) => void

  onIndexProgress?: (
    data: Record<string, unknown>,
  ) => void

  onIndexDone?: (
    data: {
      courseware_id: string
      page_count: number
      message: string
    },
  ) => void

  onGenStart?: (
    data: {
      courseware_id: string
      total_pages: number
      template: string
      message: string
      is_preview?: boolean
    },
  ) => void

  onGenPage?: (
    data: {
      page_number: number
      page_id: string
      title: string
      html_content: string
      model_used: string
      tokens_used: number
    },
  ) => void

  onGenProgress?: (
    data: {
      current_page: number
      total_pages: number
      page_title: string
      message: string
      error?: string
    },
  ) => void

  onGenDone?: (
    data: {
      courseware_id: string
      success_count: number
      fail_count: number
      total_pages: number
      elapsed_ms: number
      message: string
      errors?: string[]
      is_preview?: boolean
    },
  ) => void

  onError?: (
    data: {
      message: string
    },
  ) => void

  onReconnected?: () => void

  onAssemblyStart?: (
    data: {
      courseware_id: string
      total_pages: number
      html_pending: number
      image_pipeline: number
      html_concurrency: number
      image_concurrency: number
      skip_video: boolean
      message: string
    },
  ) => void

  onAssemblyPageHtml?: (
    data: {
      page_number: number
      page_id: string
      title: string
      html_content: string
      message: string
    },
  ) => void

  onAssemblyProgress?: (
    data: {
      page_number: number
      page_title: string
      stage: string
      error: string
      message: string
    },
  ) => void

  onAssemblyPageMedia?: (
    data: {
      page_number: number
      stage: string
      message: string
    },
  ) => void

  onAssemblyPageDone?: (
    data: {
      page_number: number
      page_id: string
      title: string
      image_ok: boolean
      image_skipped: boolean
      video_ok: boolean
      video_skipped: boolean
      message: string
    },
  ) => void

  onAssemblyDone?: (
    data: {
      courseware_id: string
      skip_video: boolean
      html_success: number
      html_fail: number
      image_success: number
      image_fail: number
      image_skip: number
      video_success: number
      video_skip: number
      total_pages: number
      elapsed_ms: number
      errors?: string[]
      message: string
    },
  ) => void
}

// ==================== 多媒体资产类型 ====================

/** 课件媒体资产 */
export interface CoursewareAsset {
  id: string
  courseware_id: string
  page_id: string | null
  placeholder_id: string
  asset_type: string
  generation_prompt: string
  oss_url: string
  public_oss_url?: string
  file_size: number
  mime_type: string
  status: string
  created_at: string
}

/**
 * 普通用户图片生成响应。
 *
 * 真实供应商原始URL和真实模型名称只保留在后端内部，
 * 不属于浏览器响应协议。
 */
export interface GenerateImageResponse {
  asset_id: string
  url: string
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

/** 图片提示词建议 */
export interface ImagePromptSuggestion {
  caption: string
  prompt: string
}

/** 视频分镜物料 */
export interface VideoStoryboardItem {
  scene: string
  image_prompt: string
  video_prompt: string
  narration: string
  frame_asset_id?: string
  frame_url?: string
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

/** 视频片段配置 */
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

/** 视频编辑器草稿 */
export interface VideoDraftItem {
  id: string
  courseware_id: string
  name: string
  clips_data: any
  clip_count: number
  created_at: string
}

// ==================== 字幕与TTS类型 ====================

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

/** 字幕轨 */
export interface CoursewareSubtitle {
  id: string
  courseware_id: string
  scope_type: string
  scope_id: string | null
  language: string
  segments: string
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

/** TTS音色 */
export interface TTSVoice {
  code: string
  name: string
  language: string
  gender: string
  style: string
}

/** TTS生成请求 */
export interface GenerateTTSRequest {
  voice: string
  speed?: number
  segment_ids?: string[]
  operation_id: string
}

/** TTS生成响应 */
export interface GenerateTTSResponse {
  subtitle_id: string
  success_count: number
  fail_count: number
  total_count: number
  segments: string
  errors: string[]
  message: string
}

/** 上传OSS响应 */
export interface UploadToOSSResponse {
  asset_id: string
  local_url: string
  oss_public_url: string
  message: string
}

/** 设置风格锚点响应 */
export interface SetStyleAnchorResult {
  asset_id: string
  anchor_url: string
  vaoci: string
}

// ==================== UI常量配置 ====================

/** 课件状态配置 */
export const CW_STATUS_CONFIG:
Record<string, {
  label: string
  color: string
  bg: string
}> = {
  draft: {
    label: '草稿',
    color: '#6B7280',
    bg: '#F3F4F6',
  },
  indexing: {
    label: '方案编辑中',
    color: '#D97706',
    bg: '#FEF3C7',
  },
  styling: {
    label: '风格选择中',
    color: '#7C3AED',
    bg: '#EDE9FE',
  },
  generating: {
    label: '课件生成中',
    color: '#2563EB',
    bg: '#DBEAFE',
  },
  preview: {
    label: '预览确认中',
    color: '#0891B2',
    bg: '#CFFAFE',
  },
  confirmed: {
    label: '已确认',
    color: '#059669',
    bg: '#D1FAE5',
  },
  in_pipeline: {
    label: '审核中',
    color: '#4F46E5',
    bg: '#E0E7FF',
  },
}

/** 发布态UI配置 */
export const CW_PUBLISH_STATE_CONFIG:
Record<string, {
  label: string
  color: string
  bg: string
}> = {
  private: {
    label: '私有',
    color: '#6B7280',
    bg: '#F3F4F6',
  },
  published_personal: {
    label: '个人发布',
    color: '#0891B2',
    bg: '#CFFAFE',
  },
  submitted: {
    label: '审核中',
    color: '#D97706',
    bg: '#FEF3C7',
  },
  approved: {
    label: '审核通过',
    color: '#059669',
    bg: '#D1FAE5',
  },
  published_shared: {
    label: '已共享',
    color: '#059669',
    bg: '#D1FAE5',
  },
  revision: {
    label: '已退回',
    color: '#DC2626',
    bg: '#FEE2E2',
  },
}

/** 源代码开放范围UI配置 */
export const CW_CODE_SHARE_SCOPE_CONFIG:
Record<string, {
  label: string
  short: string
  color: string
  bg: string
}> = {
  none: {
    label: '不开放源码',
    short: '🔒 源码不开放',
    color: '#6B7280',
    bg: '#F3F4F6',
  },
  group: {
    label: '本教研组可复制',
    short: '👥 本组可复制',
    color: '#2563EB',
    bg: '#DBEAFE',
  },
  school: {
    label: '本校可复制',
    short: '🏫 本校可复制',
    color: '#7C3AED',
    bg: '#EDE9FE',
  },
  region: {
    label: '本区域可复制',
    short: '🗺 区域可复制',
    color: '#0891B2',
    bg: '#CFFAFE',
  },
  public: {
    label: '所有人可复制',
    short: '🌐 公开可复制',
    color: '#059669',
    bg: '#D1FAE5',
  },
}

/** 源代码开放范围下拉选项 */
export const CW_CODE_SHARE_SCOPE_OPTIONS: {
  value: string
  label: string
}[] = [
  {
    value: 'none',
    label: '🔒 不开放源码（仅可看/放映，不可复制）',
  },
  {
    value: 'group',
    label: '👥 本教研组可复制源码',
  },
  {
    value: 'school',
    label: '🏫 本校可复制源码',
  },
  {
    value: 'region',
    label: '🗺 本区域可复制源码',
  },
  {
    value: 'public',
    label: '🌐 所有可见者均可复制源码',
  },
]

/** 组件类型配色 */
export const CW_COMP_TYPE_CONFIG:
Record<string, {
  label: string
  color: string
  bg: string
}> = {
  layout: {
    label: '布局模板',
    color: '#2563EB',
    bg: '#DBEAFE',
  },
  interaction: {
    label: '交互功能',
    color: '#059669',
    bg: '#D1FAE5',
  },
  '3d': {
    label: '3D/动画',
    color: '#7C3AED',
    bg: '#EDE9FE',
  },
  animation: {
    label: '动画效果',
    color: '#DB2777',
    bg: '#FCE7F3',
  },
  data_viz: {
    label: '数据可视化',
    color: '#0891B2',
    bg: '#CFFAFE',
  },
  multimedia: {
    label: '多媒体容器',
    color: '#D97706',
    bg: '#FEF3C7',
  },
  style: {
    label: '样式主题',
    color: '#4F46E5',
    bg: '#E0E7FF',
  },
}

/** 风格类别配色 */
export const CW_STYLE_CONFIG:
Record<string, {
  label: string
  color: string
  bg: string
  emoji: string
}> = {
  minimalist: {
    label: '简约清新',
    color: '#2563EB',
    bg: '#DBEAFE',
    emoji: '✨',
  },
  playful: {
    label: '活泼趣味',
    color: '#F59E0B',
    bg: '#FEF3C7',
    emoji: '🎈',
  },
  tech: {
    label: '科技感',
    color: '#7C3AED',
    bg: '#EDE9FE',
    emoji: '🔬',
  },
  academic: {
    label: '学术严谨',
    color: '#1F2937',
    bg: '#F3F4F6',
    emoji: '📖',
  },
  organic: {
    label: '自然有机',
    color: '#059669',
    bg: '#D1FAE5',
    emoji: '🌿',
  },
  immersive: {
    label: '3D沉浸式',
    color: '#DC2626',
    bg: '#FEE2E2',
    emoji: '🎮',
  },
}

/** 交互类型标签 */
export const CW_INTERACTION_TYPES:
Record<string, {
  label: string
  emoji: string
}> = {
  static: {
    label: '静态展示',
    emoji: '📄',
  },
  click: {
    label: '点击交互',
    emoji: '👆',
  },
  drag: {
    label: '拖拽操作',
    emoji: '✋',
  },
  input: {
    label: '输入填写',
    emoji: '✏️',
  },
  animation: {
    label: '动画演示',
    emoji: '🎬',
  },
  video: {
    label: '视频播放',
    emoji: '📹',
  },
  game: {
    label: '游戏互动',
    emoji: '🎮',
  },
  quiz: {
    label: '答题测验',
    emoji: '❓',
  },
}

/** 视觉形式标签 */
export const CW_VISUAL_FORMATS:
Record<string, {
  label: string
  emoji: string
}> = {
  text_heavy: {
    label: '文字为主',
    emoji: '📝',
  },
  image_text: {
    label: '图文混排',
    emoji: '🖼️',
  },
  diagram: {
    label: '示意图',
    emoji: '📊',
  },
  chart: {
    label: '图表',
    emoji: '📈',
  },
  timeline: {
    label: '时间线',
    emoji: '⏳',
  },
  comparison: {
    label: '对比展示',
    emoji: '⚖️',
  },
  gallery: {
    label: '图片画廊',
    emoji: '🎨',
  },
  fullscreen_media: {
    label: '全屏媒体',
    emoji: '🖥️',
  },
}

/** 认知层次标签 */
export const CW_COGNITIVE_LEVELS:
Record<number, {
  label: string
  color: string
  bg: string
}> = {
  1: {
    label: '记忆',
    color: '#059669',
    bg: '#D1FAE5',
  },
  2: {
    label: '理解',
    color: '#0891B2',
    bg: '#CFFAFE',
  },
  3: {
    label: '应用',
    color: '#2563EB',
    bg: '#DBEAFE',
  },
  4: {
    label: '分析',
    color: '#D97706',
    bg: '#FEF3C7',
  },
  5: {
    label: '评价',
    color: '#DC2626',
    bg: '#FEE2E2',
  },
  6: {
    label: '创造',
    color: '#7C3AED',
    bg: '#EDE9FE',
  },
}

// ==================== 通用提取函数 ====================

/** 从统一Axios响应中提取data */
export function extractData<T>(
  response: {
    data?: {
      code?: number
      data?: T
    }
  },
): T {
  const body = response?.data

  if (
    body &&
    body.code === 0 &&
    body.data !== undefined
  ) {
    return body.data
  }

  throw new Error('接口返回异常')
}

// ==================== 页面级版本与回退类型 ====================

/** 页面HTML版本列表项 */
export interface PageVersionEntry {
  id: string
  version_no: number
  source: string
  source_label: string
  note: string
  created_at: string
}

// ==================== 课件与教案对齐报告类型 ====================

/** 对齐覆盖项 */
export interface AlignmentCoverageItem {
  plan_segment: string
  status: 'covered' | 'partial' | 'missing'
  page_nums: number[]
  note: string
}

/** 对齐新增项 */
export interface AlignmentAdditionItem {
  page_num: number
  desc: string
}

/** 对齐意图偏移项 */
export interface AlignmentIntentShiftItem {
  page_num: number
  plan_intent: string
  scheme_purpose: string
  note: string
}

/** 对齐完整结构化分析结果 */
export interface AlignmentResultJSON {
  overall: 'aligned' | 'minor' | 'major'
  summary: string
  coverage: AlignmentCoverageItem[]
  additions: AlignmentAdditionItem[]
  intent_shifts: AlignmentIntentShiftItem[]
}

/** 对齐报告主记录 */
export interface CoursewareAlignmentReport {
  id: string
  courseware_id: string
  lesson_plan_id: string | null
  overall: 'aligned' | 'minor' | 'major' | 'failed'
  summary: string
  report_json: string
  status: 'generating' | 'done' | 'failed'
  error_message: string
  model_used: string
  tokens_used: number
  page_count: number
  created_at: string | null
  updated_at: string | null
}

/** 对齐报告查询响应 */
export interface AlignmentReportResponse {
  has_report: boolean
  report: CoursewareAlignmentReport | null
}

/** 对齐整体结论UI配置 */
export const CW_ALIGNMENT_OVERALL_CONFIG:
Record<string, {
  label: string
  color: string
  bg: string
  emoji: string
}> = {
  aligned: {
    label: '已对齐',
    color: '#059669',
    bg: '#D1FAE5',
    emoji: '✅',
  },
  minor: {
    label: '小幅偏差',
    color: '#D97706',
    bg: '#FEF3C7',
    emoji: '⚠️',
  },
  major: {
    label: '需注意',
    color: '#DC2626',
    bg: '#FEE2E2',
    emoji: '❗',
  },
  failed: {
    label: '校验失败',
    color: '#6B7280',
    bg: '#F3F4F6',
    emoji: '⚪',
  },
}

/** 对齐覆盖状态UI配置 */
export const CW_ALIGNMENT_COVERAGE_CONFIG:
Record<string, {
  label: string
  color: string
  bg: string
  emoji: string
}> = {
  covered: {
    label: '已覆盖',
    color: '#059669',
    bg: '#D1FAE5',
    emoji: '✅',
  },
  partial: {
    label: '部分覆盖',
    color: '#D97706',
    bg: '#FEF3C7',
    emoji: '◐',
  },
  missing: {
    label: '未覆盖',
    color: '#DC2626',
    bg: '#FEE2E2',
    emoji: '✕',
  },
}
