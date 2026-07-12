/**
 * 课件工坊 API —— 类型定义与常量配置 (coursewares.types.ts)
 *
 * 从原 coursewares.ts 拆出的纯类型层：所有 interface/type + UI 常量配置表。
 * 不含任何运行时请求逻辑，供 coursewares.core.ts / coursewares.media.ts 及
 * 业务组件统一 import。对外经桶文件 coursewares.ts 用 `export *` 透出，
 * 既有 `import { X } from '@/api/coursewares'` 路径完全不变。
 *
 * 阶段1（课件审核与协作·发布与共享）变更：
 *   - CoursewareListItem 补 publish_state/publish_state_name/review_level/code_share_scope
 *     四字段（后端"我的课件"列表已返回，供前端展示发布徽章 + 代码范围）。
 *   - CoursewareDetail 同步补上述发布字段（详情页用）。
 *   - 新增 SharedCoursewareListItem / SharedCoursewareListResponse（共享课件库列表）。
 *   - 新增发布态 / 代码开放范围两套 UI 常量（与后端 CWPublish* / CWCodeShare* 口径对齐）。
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
  // ---- 阶段1：发布/共享维度（后端"我的课件"列表已返回，供发布徽章 + 代码范围展示）----
  publish_state: string        // 发布态：private/published_personal/submitted/approved/published_shared/revision
  publish_state_name: string   // 发布态中文名
  review_level: number         // 审核层级进度：0未提交/1=L1通过/2=L2通过
  code_share_scope: string     // 源代码开放范围：none/group/school/region/public
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
  kp_codes?: string                     // 课程知识库轮：本课件绑定的课标知识点编码数组JSON文本（如 ["MATH-G3-GM-001"]），空/缺=未绑定
  // ---- 阶段1：发布/审核维度（详情页展示发布徽章 / 代码范围 / 审核进度）----
  publish_state?: string         // 发布态
  publish_state_name?: string    // 发布态中文名
  review_level?: number          // 审核层级进度
  review_school_id?: string | null // 提交审核时的作者学校ID（可空）
  code_share_scope?: string      // 源代码开放范围
  // ---- 阶段4：集体备课维度（详情页展示集体备课徽章、判断可否发起/微调）----
  collab_state?: string          // 集体备课态：idle 未集体备课 / in_session 集体备课中
  collab_member_count?: number   // 阶段4治本补丁：集体备课在场参与者数。仅 in_session 且 >0 才算真会话，
                                 //   避免作者忘记结束导致空会话仍显示误导横幅/提示条/红点
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

// ==================== 阶段1：共享课件库类型 ====================

/**
 * 共享课件库列表单条（对应后端 SharedCoursewareListItem）
 * 列出他人共享给"我"（同校/同组）的课件，带作者名、学校名，
 * 以及"当前登录者能否复制此课件源码"的 can_copy 标记（前端据此显隐"复制到我的"按钮）。
 */
export interface SharedCoursewareListItem {
  id: string
  title: string
  subject: string
  grade: string
  page_count: number
  source_type: string
  source_name: string
  author_id: string          // 作者用户ID
  author_name: string        // 作者显示名
  school_name: string        // 作者所属学校名
  publish_state: string      // 发布态（共享库里通常为 published_shared）
  publish_state_name: string
  code_share_scope: string   // 源代码开放范围
  can_copy: boolean          // 当前登录者能否复制源码（后端按 code_share_scope + 归属关系裁决）
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

// ==================== 课程知识库类型（平台级公共，课件/教案复用） ====================

/**
 * 课标知识点（对应后端 curriculum_standards 表 / models.CurriculumKP）
 * 权威/稳定/版本无关，定义各学科各学段知识点与三档深度，是难度自动适配的依据。
 */
export interface CurriculumKP {
  id: string
  subject: string               // 学科
  stage: string                 // 学段
  grade_num: number             // 具体年级1-9（0=学段级）
  domain: string                // 领域（数与代数/图形与几何...）
  theme: string                 // 主题
  kp_code: string               // 知识点全局编码
  kp_name: string               // 知识点名称
  content_requirement: string   // 内容要求：学什么
  academic_requirement: string  // 学业要求：学到什么程度（难度适配核心）
  teaching_hint: string         // 教学提示
  depth_level: number           // 难度档 1体验感知/2理解应用/3分析迁移
  core_competency: string       // 对应核心素养
  source_ref: string            // 出处
  confidence: number            // 置信度0-100
  sort_order: number            // 同年级内排序
}

/** 知识点查询响应 */
export interface CurriculumKPResponse {
  subject: string
  grade: number
  knowledge_points: CurriculumKP[] | null
  total: number
}

/** 难度档 UI 配置（1-3档对应配色与中文标签） */
export const CW_DEPTH_LEVEL_CONFIG: Record<number, { label: string; color: string; bg: string }> = {
  1: { label: '体验感知', color: '#059669', bg: '#D1FAE5' },
  2: { label: '理解应用', color: '#2563EB', bg: '#DBEAFE' },
  3: { label: '分析迁移', color: '#7C3AED', bg: '#EDE9FE' },
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
  // P2：SSE 断线自动重连成功后回调（首次连接不触发）。
  //   业务层据此重新拉取课件最新状态 + 页面列表，补齐断线期间漏收的已生成页，根治"假死"。
  onReconnected?: () => void
  // ==================== 全自动装配事件（assembly_*，后端 courseware_auto_assembly_service 广播）====================
  // 与 gen_* 系列并列的独立事件族。全自动/中间档两种交付模式共用，靠 data.skip_video 区分。
  // 后端配图/视频两个「阶段进度」事件（assembly_page_image / assembly_page_video）在前端合并到
  //   onAssemblyPageMedia 一个回调——二者前端都只是「更新某页某阶段的进行中文案」，无需分开处理。
  /** 装配开始：总页数、待生成HTML页数、配图流水线页数、并发数、是否跳过视频、开场文案 */
  onAssemblyStart?: (data: {
    courseware_id: string; total_pages: number; html_pending: number
    image_pipeline: number; html_concurrency: number; image_concurrency: number
    skip_video: boolean; message: string
  }) => void
  /** 某页 HTML 已生成落库（进度视图据此把该页标为「HTML✓、配图中」，并可即时预览该页HTML） */
  onAssemblyPageHtml?: (data: {
    page_number: number; page_id: string; title: string
    html_content: string; message: string
  }) => void
  /** 某页 HTML 生成失败（该页不再进入配图流水线；进度视图标红该页HTML阶段） */
  onAssemblyProgress?: (data: {
    page_number: number; page_title: string; stage: string
    error: string; message: string
  }) => void
  /** 某页配图/视频链的阶段性进度（stage: image_prompt/image_gen/image_fuse/video_storyboard；仅更新文案） */
  onAssemblyPageMedia?: (data: {
    page_number: number; stage: string; message: string
  }) => void
  /** 某页装配完成（配图/视频最终结果；进度视图据此定格该页三态图标） */
  onAssemblyPageDone?: (data: {
    page_number: number; page_id: string; title: string
    image_ok: boolean; image_skipped: boolean
    video_ok: boolean; video_skipped: boolean; message: string
  }) => void
  /** 全部装配完成（各维度成功/失败/跳过计数 + 总耗时 + 错误清单；进度视图收尾展示汇总） */
  onAssemblyDone?: (data: {
    courseware_id: string; skip_video: boolean
    html_success: number; html_fail: number
    image_success: number; image_fail: number; image_skip: number
    video_success: number; video_skip: number
    total_pages: number; elapsed_ms: number
    errors?: string[]; message: string
  }) => void
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

/**
 * 阶段1：发布态 UI 配置（前端发布徽章用，与后端 CWPublishStateNameMap 口径对齐）
 * private 私有不显徽章（默认态无需提示），其余态显彩色徽章。
 */
export const CW_PUBLISH_STATE_CONFIG: Record<string, { label: string; color: string; bg: string }> = {
  private:            { label: '私有',     color: '#6B7280', bg: '#F3F4F6' },
  published_personal: { label: '个人发布', color: '#0891B2', bg: '#CFFAFE' },
  submitted:          { label: '审核中',   color: '#D97706', bg: '#FEF3C7' },
  approved:           { label: '审核通过', color: '#059669', bg: '#D1FAE5' },
  published_shared:   { label: '已共享',   color: '#059669', bg: '#D1FAE5' },
  revision:           { label: '已退回',   color: '#DC2626', bg: '#FEE2E2' },
}

/**
 * 阶段1：源代码开放范围 UI 配置（前端代码范围选择器 / 共享卡片标签用）
 * 与后端 CWCodeShareScopeNameMap 一一对应。
 */
export const CW_CODE_SHARE_SCOPE_CONFIG: Record<string, { label: string; short: string; color: string; bg: string }> = {
  none:   { label: '不开放源码',     short: '🔒 源码不开放', color: '#6B7280', bg: '#F3F4F6' },
  group:  { label: '本教研组可复制', short: '👥 本组可复制', color: '#2563EB', bg: '#DBEAFE' },
  school: { label: '本校可复制',     short: '🏫 本校可复制', color: '#7C3AED', bg: '#EDE9FE' },
  region: { label: '本区域可复制',   short: '🗺 区域可复制', color: '#0891B2', bg: '#CFFAFE' },
  public: { label: '所有人可复制',   short: '🌐 公开可复制', color: '#059669', bg: '#D1FAE5' },
}

/** 代码开放范围下拉选项顺序（none 默认居首） */
export const CW_CODE_SHARE_SCOPE_OPTIONS: { value: string; label: string }[] = [
  { value: 'none',   label: '🔒 不开放源码（仅可看/放映，不可复制）' },
  { value: 'group',  label: '👥 本教研组可复制源码' },
  { value: 'school', label: '🏫 本校可复制源码' },
  { value: 'region', label: '🗺 本区域可复制源码' },
  { value: 'public', label: '🌐 所有可见者均可复制源码' },
]

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

// ==================== 页面级版本与回退类型 ====================

/**
 * 课件页面 html_content 版本快照列表项（对应后端 ListPageVersions 返回的单条）
 * 轻量：不含 html_content（列表只展示元信息，回退时按 id 让后端取完整 HTML）。
 * 来源 source 的中文标签由后端直接给出 source_label（如 "🎨 微调前"），前端无需再映射。
 */
export interface PageVersionEntry {
  id: string            // 版本快照ID（回退时传给后端定位目标版本）
  version_no: number    // 该页第几版（每页独立从1递增，倒序展示最新在前）
  source: string        // 来源枚举原始值：refine/regenerate/rollback/...
  source_label: string  // 来源中文标签（后端已附，如 "🎨 微调前" "🔄 重生前" "↩️ 回退前"）
  note: string          // 备注（微调指令/重生说明/回退说明，可能为空串）
  created_at: string    // 存版时间（ISO字符串）
}

// ==================== 课件↔教案对齐报告类型 ====================

/** 对齐：单个教学环节的覆盖情况 */
export interface AlignmentCoverageItem {
  plan_segment: string                          // 教案里的环节名（如"情境导入""新知讲解"）
  status: 'covered' | 'partial' | 'missing'     // 覆盖状态
  page_nums: number[]                           // 对应课件页码（missing 时为空数组）
  note: string                                  // 简短说明
}

/** 对齐：课件方案中新增的、教案没有的内容 */
export interface AlignmentAdditionItem {
  page_num: number
  desc: string
}

/** 对齐：教学意图偏移 */
export interface AlignmentIntentShiftItem {
  page_num: number
  plan_intent: string      // 教案对应环节的目标
  scheme_purpose: string   // 课件这一页的目的
  note: string             // 偏移点说明
}

/** 对齐：完整结构化分析结果（落在 report_json 内，前端解析渲染） */
export interface AlignmentResultJSON {
  overall: 'aligned' | 'minor' | 'major'
  summary: string
  coverage: AlignmentCoverageItem[]
  additions: AlignmentAdditionItem[]
  intent_shifts: AlignmentIntentShiftItem[]
}

/** 对齐报告主记录（对应后端 CoursewareAlignmentReport） */
export interface CoursewareAlignmentReport {
  id: string
  courseware_id: string
  lesson_plan_id: string | null
  overall: 'aligned' | 'minor' | 'major' | 'failed'
  summary: string
  report_json: string        // 完整 JSON 文本，前端 JSON.parse 为 AlignmentResultJSON
  status: 'generating' | 'done' | 'failed'
  error_message: string
  model_used: string
  tokens_used: number
  page_count: number
  created_at: string | null
  updated_at: string | null
}

/** 对齐报告查询响应（has_report=false 时不显示对齐卡片） */
export interface AlignmentReportResponse {
  has_report: boolean
  report: CoursewareAlignmentReport | null
}

/** 对齐整体结论 UI 配置 */
export const CW_ALIGNMENT_OVERALL_CONFIG: Record<string, { label: string; color: string; bg: string; emoji: string }> = {
  aligned: { label: '已对齐', color: '#059669', bg: '#D1FAE5', emoji: '✅' },
  minor:   { label: '小幅偏差', color: '#D97706', bg: '#FEF3C7', emoji: '⚠️' },
  major:   { label: '需注意', color: '#DC2626', bg: '#FEE2E2', emoji: '❗' },
  failed:  { label: '校验失败', color: '#6B7280', bg: '#F3F4F6', emoji: '⚪' },
}

/** 对齐覆盖状态 UI 配置 */
export const CW_ALIGNMENT_COVERAGE_CONFIG: Record<string, { label: string; color: string; bg: string; emoji: string }> = {
  covered: { label: '已覆盖', color: '#059669', bg: '#D1FAE5', emoji: '✅' },
  partial: { label: '部分覆盖', color: '#D97706', bg: '#FEF3C7', emoji: '◐' },
  missing: { label: '未覆盖', color: '#DC2626', bg: '#FEE2E2', emoji: '✕' },
}
