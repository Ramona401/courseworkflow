/**
 * coursewareComicEditorDraft.ts
 *
 * 漫画覆盖层浏览器草稿协议与不可变更新入口：
 *   - 创建、解析、序列化和比较单格覆盖层草稿；
 *   - 草稿记录建立时的服务端版本和内容指纹；
 *   - 服务端版本变化时保留本地内容，不再静默丢弃；
 *   - 区分“只有版本变化”和“服务端覆盖层真实变化”；
 *   - 更新普通文字、问题卡、样式、复制和删除元素；
 *   - 所有隐式草稿流程保持教师字号，不按框体大小自动缩字；
 *   - 统一转出布局与文字显示工具。
 *
 * 本文件只处理纯数据，不发起网络请求，也不直接操作浏览器存储。
 */

import type {
  CoursewareComicOverlayDocument,
  CoursewareComicOverlayElement,
  CoursewareComicPanel,
  CoursewareComicQuestionContent,
} from '@/api/coursewares'

import {
  compactCoursewareComicOverlayDocument,
} from './coursewareComicEditorLayout'

import {
  normalizeCoursewareComicSpeechTail,
} from './coursewareComicTailEditing'

import {
  resolveCoursewareComicStyleID,
} from './coursewareComicStyleOptions'

export {
  compactCoursewareComicOverlayDocument,
  fitCoursewareComicOverlayElement,
  fitCoursewareComicOverlayElementByID,
  updateCoursewareComicOverlayLayout,
  updateCoursewareComicOverlayTextStyle,
} from './coursewareComicEditorLayout'

export {
  coursewareComicElementLabel,
  overlayElementDisplayText,
} from './coursewareComicEditorText'

export interface CoursewareComicOverlayDraft {
  narrationText: string
  overlayDocument: CoursewareComicOverlayDocument

  /** 建立当前本地编辑分支时对应的服务端panel.version。 */
  sourceVersion: number

  /**
   * 建立当前编辑分支时服务端文字与覆盖层内容的稳定指纹。
   * 版本变化但指纹相同，表示服务器只更新了图片状态等无关字段。
   */
  sourceFingerprint: string
}

function cloneJSON<Value>(value: Value): Value {
  return JSON.parse(JSON.stringify(value)) as Value
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(maximum, Math.max(minimum, value))
}

function normalizeString(value: unknown, fallback: string): string {
  return typeof value === 'string' ? value : fallback
}

function normalizeNumber(value: unknown, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value)
    ? value
    : fallback
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isOverlayDocument(
  value: unknown,
): value is CoursewareComicOverlayDocument {
  if (!isRecord(value)) {
    return false
  }

  if (
    typeof value.version !== 'number' ||
    !isRecord(value.canvas) ||
    typeof value.canvas.width !== 'number' ||
    typeof value.canvas.height !== 'number' ||
    !Array.isArray(value.elements)
  ) {
    return false
  }

  return value.elements.every(element =>
    isRecord(element) &&
    typeof element.id === 'string' &&
    typeof element.type === 'string',
  )
}

/**
 * 使用FNV-1a 32位散列生成轻量稳定指纹。
 * 指纹只用于判断本地草稿的服务端基线是否变化，不作为安全签名。
 */
export function coursewareComicOverlayContentFingerprint(
  value: Pick<
    CoursewareComicOverlayDraft,
    'narrationText' | 'overlayDocument'
  >,
): string {
  const serialized = JSON.stringify({
    narrationText: value.narrationText,
    overlayDocument: value.overlayDocument,
  })

  let hash = 0x811c9dc5

  for (let index = 0; index < serialized.length; index += 1) {
    hash ^= serialized.charCodeAt(index)
    hash = Math.imul(hash, 0x01000193)
  }

  return `fnv1a-${(hash >>> 0).toString(16).padStart(8, '0')}`
}

export function createCoursewareComicOverlayDraft(
  panel: CoursewareComicPanel,
): CoursewareComicOverlayDraft {
  const value = {
    narrationText: panel.narration_text,
    overlayDocument: compactCoursewareComicOverlayDocument(
      cloneJSON(panel.overlay_document),
    ),
  }

  return {
    ...value,
    sourceVersion: panel.version,
    sourceFingerprint: coursewareComicOverlayContentFingerprint(value),
  }
}

/**
 * 旧实现发现sourceVersion不一致时直接回退服务端内容，刷新会丢掉
 * 尚未提交的排版。现在只在草稿结构损坏时回退；版本不一致时继续
 * 保留本地内容，由编辑Hook决定安全换基线、自动保存或提示冲突。
 */
export function parseCoursewareComicOverlayDraft(
  raw: string,
  fallback: CoursewareComicOverlayDraft,
): CoursewareComicOverlayDraft {
  if (!raw.trim()) {
    return {
      ...fallback,
      overlayDocument: cloneJSON(fallback.overlayDocument),
    }
  }

  try {
    const parsed = JSON.parse(raw) as unknown

    if (!isRecord(parsed)) {
      throw new Error('漫画覆盖层草稿不是对象')
    }

    const parsedSourceVersion = normalizeNumber(
      parsed.sourceVersion,
      fallback.sourceVersion,
    )

    const selectedDocument = isOverlayDocument(parsed.overlayDocument)
      ? cloneJSON(parsed.overlayDocument)
      : cloneJSON(fallback.overlayDocument)

    const narrationText = normalizeString(
      parsed.narrationText,
      fallback.narrationText,
    )

    const storedFingerprint = normalizeString(
      parsed.sourceFingerprint,
      '',
    ).trim()

    /*
     * 历史草稿没有sourceFingerprint：
     *   - 版本仍相同时可安全补当前服务端指纹；
     *   - 版本不同时保持空值，交给Hook保守判定真实冲突。
     */
    const sourceFingerprint = storedFingerprint || (
      parsedSourceVersion === fallback.sourceVersion
        ? fallback.sourceFingerprint
        : ''
    )

    return {
      narrationText,
      overlayDocument: compactCoursewareComicOverlayDocument(
        selectedDocument,
      ),
      sourceVersion: parsedSourceVersion,
      sourceFingerprint,
    }
  } catch {
    return {
      ...fallback,
      overlayDocument: cloneJSON(fallback.overlayDocument),
    }
  }
}

export function serializeCoursewareComicOverlayDraft(
  draft: CoursewareComicOverlayDraft,
): string {
  return JSON.stringify(draft)
}

export function coursewareComicOverlayDraftEquals(
  left: CoursewareComicOverlayDraft,
  right: CoursewareComicOverlayDraft,
): boolean {
  return (
    left.narrationText === right.narrationText &&
    JSON.stringify(left.overlayDocument) ===
      JSON.stringify(right.overlayDocument)
  )
}

/**
 * sourceFingerprint相同表示覆盖层内容没有变化，只是panel其他字段更新；
 * 指纹缺失或不同则保守判定冲突，避免自动覆盖其他标签页修改。
 */
export function coursewareComicOverlayDraftHasVersionConflict(
  draft: CoursewareComicOverlayDraft,
  serverDraft: CoursewareComicOverlayDraft,
): boolean {
  if (draft.sourceVersion === serverDraft.sourceVersion) {
    return false
  }

  return (
    !draft.sourceFingerprint ||
    draft.sourceFingerprint !== serverDraft.sourceFingerprint
  )
}

/** 保留教师本地内容，只把其服务端基线升级到最新版本。 */
export function rebaseCoursewareComicOverlayDraft(
  draft: CoursewareComicOverlayDraft,
  serverDraft: CoursewareComicOverlayDraft,
): CoursewareComicOverlayDraft {
  return {
    ...draft,
    sourceVersion: serverDraft.sourceVersion,
    sourceFingerprint: serverDraft.sourceFingerprint,
  }
}

export function updateCoursewareComicOverlayElement(
  document: CoursewareComicOverlayDocument,
  elementID: string,
  updater: (
    element: CoursewareComicOverlayElement,
  ) => CoursewareComicOverlayElement,
): CoursewareComicOverlayDocument {
  return {
    ...document,
    elements: document.elements.map(element =>
      element.id === elementID
        ? updater(cloneJSON(element))
        : element,
    ),
  }
}

export function updateCoursewareComicOverlayContent(
  document: CoursewareComicOverlayDocument,
  elementID: string,
  content: string,
): CoursewareComicOverlayDocument {
  return updateCoursewareComicOverlayElement(
    document,
    elementID,
    element => ({
      ...element,
      content,
      content_dirty:
        content !== element.original_content,
    }),
  )
}

export function updateCoursewareComicQuestion(
  document: CoursewareComicOverlayDocument,
  elementID: string,
  updater: (
    question: CoursewareComicQuestionContent,
  ) => CoursewareComicQuestionContent,
): CoursewareComicOverlayDocument {
  return updateCoursewareComicOverlayElement(
    document,
    elementID,
    element => {
      const current = element.question || {
        question: '',
        options: ['', ''],
        answer_index: 0,
        explanation: '',
        answer_mode: 'click_reveal',
      }

      return {
        ...element,
        question: updater(cloneJSON(current)),
        content_dirty: true,
      }
    },
  )
}

/** styleID非空时明确选择样式；空值继续兼容旧调用的循环切换。 */
export function cycleCoursewareComicOverlayStyle(
  document: CoursewareComicOverlayDocument,
  elementID: string,
  styleID = '',
): CoursewareComicOverlayDocument {
  return updateCoursewareComicOverlayElement(
    document,
    elementID,
    element => ({
      ...element,
      style_id: resolveCoursewareComicStyleID(element, styleID),
      layout_dirty: element.layout_dirty,
    }),
  )
}

export function duplicateCoursewareComicOverlayElement(
  document: CoursewareComicOverlayDocument,
  elementID: string,
): CoursewareComicOverlayDocument {
  if (document.elements.length >= 8) {
    return document
  }

  const source = document.elements.find(element => element.id === elementID)

  if (!source) {
    return document
  }

  const usedIDs = new Set(document.elements.map(element => element.id))
  const safeBase = source.id.slice(0, 42)

  let index = 1
  let nextID = `${safeBase}-COPY-${index}`

  while (usedIDs.has(nextID)) {
    index += 1
    nextID = `${safeBase}-COPY-${index}`
  }

  const maximumZ = Math.max(
    20,
    ...document.elements.map(element => element.z_index),
  )

  const duplicated = normalizeCoursewareComicSpeechTail({
    ...cloneJSON(source),
    id: nextID,
    x: clamp(source.x + 0.025, 0, 1 - source.width),
    y: clamp(source.y + 0.025, 0, 1 - source.height),
    z_index: maximumZ + 1,
    locked: false,
    content_dirty: true,
    layout_dirty: true,
  })

  return {
    ...document,
    elements: [...document.elements, duplicated],
  }
}

export function deleteCoursewareComicOverlayElement(
  document: CoursewareComicOverlayDocument,
  elementID: string,
): CoursewareComicOverlayDocument {
  if (document.elements.length <= 1) {
    return document
  }

  return {
    ...document,
    elements: document.elements.filter(element => element.id !== elementID),
  }
}
