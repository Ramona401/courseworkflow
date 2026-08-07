/**
 * coursewareComicQuickCreate.ts
 *
 * 知识点漫画一键创建与可选参考资料绑定协议。
 *
 * 创建顺序：
 *   1. 使用knowledge_text和教师选择的4—8格数量创建漫画项目；
 *   2. 图片文件在项目创建成功后上传到当前课件第一个页面；
 *   3. 最多8项参考资料逐项绑定到漫画项目；
 *   4. 全部绑定尝试完成后再启动AI规划。
 *
 * 正式教材、已有课件和课程大纲只保存source_id，
 * 后端会重新读取可信标题与正文。
 */

import apiClient from '@/api/client'

import {
  extractData,
} from '@/api/coursewares.types'

import {
  uploadCWImage,
} from '@/api/coursewares'

import {
  createCoursewareComicReference,
} from '@/api/coursewares.comic.references'

import type {
  CoursewareComicProject,
} from '@/api/coursewares'

import type {
  CreateCoursewareComicReferenceInput,
} from '@/api/coursewares.comic.references'

export const COURSEWARE_COMIC_QUICK_PANEL_COUNTS = [
  4,
  5,
  6,
  7,
  8,
] as const

export type CoursewareComicQuickPanelCount =
  typeof COURSEWARE_COMIC_QUICK_PANEL_COUNTS[number]

export interface CoursewareComicPendingRequestReference {
  mode: 'request'
  key: string
  label: string
  input:
    Omit<
      CreateCoursewareComicReferenceInput,
      'sort_order'
    >
}

export interface CoursewareComicPendingImageUpload {
  mode: 'image_upload'
  key: string
  label: string
  file: File
  pageNumber: number
}

export type CoursewareComicPendingReference =
  | CoursewareComicPendingRequestReference
  | CoursewareComicPendingImageUpload

export interface AttachCoursewareComicReferencesResult {
  attached: number
  errors: string[]
}

export function normalizeCoursewareComicQuickKnowledge(
  value: string,
): string {
  return value.trim()
}

export function coursewareComicQuickKnowledgeLength(
  value: string,
): number {
  return Array.from(
    normalizeCoursewareComicQuickKnowledge(
      value,
    ),
  ).length
}

export function validateCoursewareComicQuickKnowledge(
  value: string,
): string {
  const normalized =
    normalizeCoursewareComicQuickKnowledge(
      value,
    )

  if (!normalized) {
    return '请输入要讲解的知识点。'
  }

  if (
    coursewareComicQuickKnowledgeLength(
      normalized,
    ) > 8000
  ) {
    return '知识点内容不能超过8000个字符。'
  }

  return ''
}

export function normalizeCoursewareComicQuickPanelCount(
  value: number,
): CoursewareComicQuickPanelCount {
  const normalized =
    Number.isFinite(value)
      ? Math.trunc(value)
      : 0

  if (
    COURSEWARE_COMIC_QUICK_PANEL_COUNTS.includes(
      normalized as CoursewareComicQuickPanelCount,
    )
  ) {
    return normalized as CoursewareComicQuickPanelCount
  }

  throw new Error(
    '漫画格数必须选择4至8格。',
  )
}

function requiredPathSegment(
  value: string,
): string {
  const normalized =
    value.trim()

  if (!normalized) {
    throw new Error(
      '课件ID不能为空',
    )
  }

  return encodeURIComponent(
    normalized,
  )
}

export async function createCoursewareComicFromKnowledge(
  coursewareId: string,
  knowledgeText: string,
  panelCount: number,
): Promise<CoursewareComicProject> {
  const validationError =
    validateCoursewareComicQuickKnowledge(
      knowledgeText,
    )

  if (validationError) {
    throw new Error(
      validationError,
    )
  }

  const normalizedPanelCount =
    normalizeCoursewareComicQuickPanelCount(
      panelCount,
    )

  const response =
    await apiClient.post(
      `/coursewares/${requiredPathSegment(
        coursewareId,
      )}/comic-projects`,
      {
        knowledge_text:
          normalizeCoursewareComicQuickKnowledge(
            knowledgeText,
          ),

        panel_count:
          normalizedPanelCount,
      },
    )

  return extractData<CoursewareComicProject>(
    response,
  )
}

/**
 * attachCoursewareComicPendingReferences
 *
 * 逐项绑定而不是Promise.all并发：
 *   - 稳定保持老师选择的sort_order；
 *   - 图片先上传，再使用正式asset_id绑定；
 *   - 单项失败不会阻断其它资料的绑定；
 *   - 调用方可以继续规划，并明确提示失败数量。
 */
export async function attachCoursewareComicPendingReferences(
  coursewareId: string,
  projectId: string,
  references:
    CoursewareComicPendingReference[],
): Promise<AttachCoursewareComicReferencesResult> {
  const limited =
    references.slice(
      0,
      8,
    )

  let attached = 0

  const errors:
    string[] = []

  for (
    let index = 0;
    index < limited.length;
    index += 1
  ) {
    const pending =
      limited[index]

    try {
      let input:
        CreateCoursewareComicReferenceInput

      if (
        pending.mode ===
        'image_upload'
      ) {
        const uploaded =
          await uploadCWImage(
            coursewareId,
            pending.pageNumber,
            pending.file,
          )

        input = {
          resource_type:
            'uploaded_image',
          asset_id:
            uploaded.asset_id,
          title:
            pending.file.name,
          file_name:
            uploaded.file_name ||
            pending.file.name,
          mime_type:
            uploaded.mime_type ||
            pending.file.type ||
            'image/png',
          sort_order:
            index,
        }
      } else {
        input = {
          ...pending.input,
          sort_order:
            index,
        }
      }

      await createCoursewareComicReference(
        coursewareId,
        projectId,
        input,
      )

      attached += 1
    } catch (error) {
      errors.push(
        `${pending.label}：${resolveQuickCreateErrorMessage(
          error,
          '关联失败',
        )}`,
      )
    }
  }

  return {
    attached,
    errors,
  }
}

export function resolveQuickCreateErrorMessage(
  error: unknown,
  fallback: string,
): string {
  return error instanceof Error &&
    error.message.trim()
    ? error.message
    : fallback
}
