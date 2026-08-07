/**
 * coursewares.comic.workflow.ts
 *
 * 知识点漫画五步教师工作流浏览器协议。
 *
 * 能力：
 *   1. 读取服务端workflow；
 *   2. 规划或按新叙事方式重新规划；
 *   3. 确认AI分镜；
 *   4. 保存视觉设置；
 *   5. 启动和确认首格完整样张；
 *   6. 为五步编辑器提供严格工作流项目详情。
 *
 * 所有写操作均携带服务端project.version。
 */

import apiClient from './client'

import {
  extractData,
} from './coursewares.types'

import {
  getCoursewareComicProject,
  listCoursewareComicProjects,
} from './coursewares.comic'

import type {
  CoursewareComicGenerationStartResult,
  CoursewareComicPanel,
  CoursewareComicProject,
  CoursewareComicProjectDetail,
  CoursewareComicProjectList,
} from './coursewares.comic'

export type CoursewareComicWorkflowStage =
  | 'source'
  | 'storyboard'
  | 'style_preview'
  | 'batch_generation'
  | 'refinement'

export type CoursewareComicAspectRatio =
  | 'courseware'
  | '16:9'
  | '4:3'
  | '1:1'
  | '3:4'
  | '9:16'

export type CoursewareComicImageQuality =
  | 'standard'
  | 'high'

export type CoursewareComicInsertionMode =
  | 'single_page'
  | 'smart_pages'
  | 'one_panel_per_page'
  | 'library_only'

export type CoursewareComicNarrativeMode =
  | 'knowledge_story'
  | 'inquiry_mystery'
  | 'role_dialogue'
  | 'travel_adventure'
  | 'civic_case'

export type CoursewareComicVisualStyle =
  | 'science_encyclopedia'
  | 'warm_storybook'
  | 'modern_flat'
  | 'chinese_ink'
  | 'cinematic_3d'
  | 'realistic_illustration'

export type CoursewareComicVisualStyleSource =
  | 'courseware'
  | 'selected'

export interface CoursewareComicWorkflow {
  stage:
    CoursewareComicWorkflowStage

  storyboard_confirmed_at:
    string | null

  style_confirmed_at:
    string | null

  style_preview_panel_id:
    string | null

  visual_style_source:
    CoursewareComicVisualStyleSource

  aspect_ratio:
    CoursewareComicAspectRatio

  image_quality:
    CoursewareComicImageQuality

  insertion_mode:
    CoursewareComicInsertionMode

  style_instruction:
    string
}

export interface CoursewareComicWorkflowProject
  extends CoursewareComicProject {
  workflow:
    CoursewareComicWorkflow

  narrative_mode:
    CoursewareComicNarrativeMode

  visual_style:
    CoursewareComicVisualStyle
}

export interface CoursewareComicWorkflowProjectDetail {
  project:
    CoursewareComicWorkflowProject

  panels:
    CoursewareComicPanel[]
}

export interface CoursewareComicWorkflowProjectList {
  projects:
    CoursewareComicWorkflowProject[]

  total: number
}

export interface PlanCoursewareComicWorkflowProjectInput {
  expected_version: number
  teacher_instruction: string
  narrative_mode?:
    CoursewareComicNarrativeMode
}

export interface ConfirmCoursewareComicStoryboardInput {
  expected_version: number
  narrative_mode:
    CoursewareComicNarrativeMode
}

export interface UpdateCoursewareComicStyleSettingsInput {
  expected_version: number

  visual_style_source:
    CoursewareComicVisualStyleSource

  visual_style:
    CoursewareComicVisualStyle

  aspect_ratio:
    CoursewareComicAspectRatio

  image_quality:
    CoursewareComicImageQuality

  style_instruction: string
}

/**
 * CoursewareComicStyleSettingsDraft 是第三步表单使用的浏览器草稿。
 *
 * 使用camelCase避免把UI草稿误当作可直接提交的HTTP请求；
 * 提交时必须显式转换为UpdateCoursewareComicStyleSettingsInput。
 */
export interface CoursewareComicStyleSettingsDraft {
  visualStyleSource:
    CoursewareComicVisualStyleSource

  visualStyle:
    CoursewareComicVisualStyle

  aspectRatio:
    CoursewareComicAspectRatio

  imageQuality:
    CoursewareComicImageQuality

  styleInstruction: string
}

export interface ConfirmCoursewareComicStylePreviewInput {
  expected_version: number
  preview_panel_id: string
}

function requiredPathSegment(
  value: string,
): string {
  const normalized =
    value.trim()

  if (!normalized) {
    throw new Error(
      '知识点漫画资源ID不能为空',
    )
  }

  return encodeURIComponent(
    normalized,
  )
}

function projectEndpoint(
  coursewareId: string,
  projectId: string,
): string {
  return (
    `/coursewares/${requiredPathSegment(
      coursewareId,
    )}/comic-projects/${requiredPathSegment(
      projectId,
    )}`
  )
}

function isRecord(
  value: unknown,
): value is Record<string, unknown> {
  return (
    typeof value === 'object' &&
    value !== null &&
    !Array.isArray(value)
  )
}

function isWorkflowStage(
  value: unknown,
): value is CoursewareComicWorkflowStage {
  return (
    value === 'source' ||
    value === 'storyboard' ||
    value === 'style_preview' ||
    value === 'batch_generation' ||
    value === 'refinement'
  )
}

function isAspectRatio(
  value: unknown,
): value is CoursewareComicAspectRatio {
  return (
    value === 'courseware' ||
    value === '16:9' ||
    value === '4:3' ||
    value === '1:1' ||
    value === '3:4' ||
    value === '9:16'
  )
}

function isImageQuality(
  value: unknown,
): value is CoursewareComicImageQuality {
  return (
    value === 'standard' ||
    value === 'high'
  )
}

function isVisualStyleSource(
  value: unknown,
): value is CoursewareComicVisualStyleSource {
  return (
    value === 'courseware' ||
    value === 'selected'
  )
}

function isInsertionMode(
  value: unknown,
): value is CoursewareComicInsertionMode {
  return (
    value === 'single_page' ||
    value === 'smart_pages' ||
    value === 'one_panel_per_page' ||
    value === 'library_only'
  )
}

function nullableString(
  value: unknown,
): string | null {
  if (
    typeof value !== 'string' ||
    !value.trim()
  ) {
    return null
  }

  return value
}

export function readCoursewareComicWorkflow(
  project:
    CoursewareComicProject,
): CoursewareComicWorkflow | null {
  const rawProject =
    project as unknown as
      Record<string, unknown>

  const value =
    rawProject.workflow

  if (!isRecord(value)) {
    return null
  }

  if (
    !isWorkflowStage(
      value.stage,
    ) ||
    !isVisualStyleSource(
      value.visual_style_source,
    ) ||
    !isAspectRatio(
      value.aspect_ratio,
    ) ||
    !isImageQuality(
      value.image_quality,
    ) ||
    !isInsertionMode(
      value.insertion_mode,
    )
  ) {
    return null
  }

  return {
    stage:
      value.stage,

    storyboard_confirmed_at:
      nullableString(
        value.storyboard_confirmed_at,
      ),

    style_confirmed_at:
      nullableString(
        value.style_confirmed_at,
      ),

    style_preview_panel_id:
      nullableString(
        value.style_preview_panel_id,
      ),

    visual_style_source:
      value.visual_style_source,

    aspect_ratio:
      value.aspect_ratio,

    image_quality:
      value.image_quality,

    insertion_mode:
      value.insertion_mode,

    style_instruction:
      typeof value.style_instruction ===
        'string'
        ? value.style_instruction
        : '',
  }
}

export function requireCoursewareComicWorkflowProject(
  project:
    CoursewareComicProject,
): CoursewareComicWorkflowProject {
  const workflow =
    readCoursewareComicWorkflow(
      project,
    )

  if (!workflow) {
    throw new Error(
      '漫画项目缺少有效的五步工作流状态，请刷新后重试。',
    )
  }

  return {
    ...project,
    workflow,
  } as CoursewareComicWorkflowProject
}

export function requireCoursewareComicWorkflowDetail(
  detail:
    CoursewareComicProjectDetail,
): CoursewareComicWorkflowProjectDetail {
  return {
    project:
      requireCoursewareComicWorkflowProject(
        detail.project,
      ),

    panels:
      detail.panels || [],
  }
}

export async function listCoursewareComicWorkflowProjects(
  coursewareId: string,
): Promise<CoursewareComicWorkflowProjectList> {
  const result:
    CoursewareComicProjectList =
    await listCoursewareComicProjects(
      coursewareId,
    )

  return {
    projects:
      (result.projects || []).map(
        requireCoursewareComicWorkflowProject,
      ),

    total:
      result.total,
  }
}

export async function getCoursewareComicWorkflowProject(
  coursewareId: string,
  projectId: string,
): Promise<CoursewareComicWorkflowProjectDetail> {
  const result =
    await getCoursewareComicProject(
      coursewareId,
      projectId,
    )

  return requireCoursewareComicWorkflowDetail(
    result,
  )
}

export async function planCoursewareComicWorkflowProject(
  coursewareId: string,
  projectId: string,
  input:
    PlanCoursewareComicWorkflowProjectInput,
): Promise<CoursewareComicWorkflowProjectDetail> {
  const response =
    await apiClient.post(
      `${projectEndpoint(
        coursewareId,
        projectId,
      )}/plan`,
      {
        expected_version:
          input.expected_version,

        teacher_instruction:
          input.teacher_instruction.trim(),

        narrative_mode:
          input.narrative_mode || '',
      },
      {
        timeout:
          300000,
      },
    )

  return requireCoursewareComicWorkflowDetail(
    extractData<CoursewareComicProjectDetail>(
      response,
    ),
  )
}

export async function confirmCoursewareComicStoryboard(
  coursewareId: string,
  projectId: string,
  input:
    ConfirmCoursewareComicStoryboardInput,
): Promise<CoursewareComicWorkflowProjectDetail> {
  const response =
    await apiClient.post(
      `${projectEndpoint(
        coursewareId,
        projectId,
      )}/confirm-storyboard`,
      input,
    )

  return requireCoursewareComicWorkflowDetail(
    extractData<CoursewareComicProjectDetail>(
      response,
    ),
  )
}

export async function updateCoursewareComicStyleSettings(
  coursewareId: string,
  projectId: string,
  input:
    UpdateCoursewareComicStyleSettingsInput,
): Promise<CoursewareComicWorkflowProjectDetail> {
  const response =
    await apiClient.put(
      `${projectEndpoint(
        coursewareId,
        projectId,
      )}/style-settings`,
      input,
    )

  return requireCoursewareComicWorkflowDetail(
    extractData<CoursewareComicProjectDetail>(
      response,
    ),
  )
}

export async function generateCoursewareComicStylePreview(
  coursewareId: string,
  projectId: string,
  expectedVersion: number,
): Promise<CoursewareComicGenerationStartResult> {
  const response =
    await apiClient.post(
      `${projectEndpoint(
        coursewareId,
        projectId,
      )}/generate-style-preview`,
      {
        expected_version:
          expectedVersion,
      },
    )

  return extractData<CoursewareComicGenerationStartResult>(
    response,
  )
}

export async function confirmCoursewareComicStylePreview(
  coursewareId: string,
  projectId: string,
  input:
    ConfirmCoursewareComicStylePreviewInput,
): Promise<CoursewareComicWorkflowProjectDetail> {
  const response =
    await apiClient.post(
      `${projectEndpoint(
        coursewareId,
        projectId,
      )}/confirm-style-preview`,
      {
        expected_version:
          input.expected_version,

        preview_panel_id:
          input.preview_panel_id.trim(),
      },
    )

  return requireCoursewareComicWorkflowDetail(
    extractData<CoursewareComicProjectDetail>(
      response,
    ),
  )
}
