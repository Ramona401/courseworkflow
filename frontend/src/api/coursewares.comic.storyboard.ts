/**
 * coursewares.comic.storyboard.ts
 *
 * 知识点漫画第二步分镜安全编辑协议。
 *
 * 浏览器只提交教师可理解的教学分镜字段。
 * 图片提示词、负面提示词、IAOCI、内部图片键和跨格关系仍由后端维护。
 */

import apiClient from './client'
import { extractData } from './coursewares.types'

import type {
  CoursewareComicPanel,
} from './coursewares.comic'

export interface CoursewareComicStoryboardPanelDraft {
  storyPurpose: string
  knowledgeClaim: string
  sceneText: string
  actionText: string
  cameraText: string
  knowledgePresentation: string
}

export interface UpdateCoursewareComicStoryboardPanelInput {
  expected_version: number
  story_purpose: string
  knowledge_claim: string
  scene_text: string
  action_text: string
  camera_text: string
  knowledge_presentation: string
}

function requiredPathSegment(value: string): string {
  const normalized = value.trim()

  if (!normalized) {
    throw new Error('知识点漫画资源ID不能为空')
  }

  return encodeURIComponent(normalized)
}

function panelStoryboardEndpoint(
  coursewareId: string,
  projectId: string,
  panelId: string,
): string {
  return (
    `/coursewares/${requiredPathSegment(coursewareId)}` +
    `/comic-projects/${requiredPathSegment(projectId)}` +
    `/panels/${requiredPathSegment(panelId)}/storyboard`
  )
}

export function createCoursewareComicStoryboardPanelDraft(
  panel: CoursewareComicPanel,
): CoursewareComicStoryboardPanelDraft {
  return {
    storyPurpose: panel.story_purpose || '',
    knowledgeClaim: panel.knowledge_claim || '',
    sceneText: panel.scene_text || '',
    actionText: panel.action_text || '',
    cameraText: panel.camera_text || '',
    knowledgePresentation: panel.knowledge_presentation || '',
  }
}

export async function updateCoursewareComicStoryboardPanel(
  coursewareId: string,
  projectId: string,
  panelId: string,
  input: UpdateCoursewareComicStoryboardPanelInput,
): Promise<CoursewareComicPanel> {
  const response = await apiClient.put(
    panelStoryboardEndpoint(
      coursewareId,
      projectId,
      panelId,
    ),
    {
      expected_version: input.expected_version,
      story_purpose: input.story_purpose.trim(),
      knowledge_claim: input.knowledge_claim.trim(),
      scene_text: input.scene_text.trim(),
      action_text: input.action_text.trim(),
      camera_text: input.camera_text.trim(),
      knowledge_presentation:
        input.knowledge_presentation.trim(),
    },
  )

  return extractData<CoursewareComicPanel>(response)
}
