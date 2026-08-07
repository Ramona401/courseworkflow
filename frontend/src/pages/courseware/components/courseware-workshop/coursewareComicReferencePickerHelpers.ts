/**
 * coursewareComicReferencePickerHelpers.ts
 *
 * 知识点漫画可选参考资料选择器的稳定辅助模块。
 *
 * 集中维护：
 *   - 数量与图片体积上限；
 *   - 年级数字解析；
 *   - 文件稳定去重键；
 *   - 文档MIME和默认图片文件名；
 *   - 统一错误文案；
 *   - 选择器共享样式。
 */

import type {
  CSSProperties,
} from 'react'

export const MAX_COURSEWARE_COMIC_REFERENCES =
  8

export const MAX_COURSEWARE_COMIC_REFERENCE_IMAGE_BYTES =
  5 * 1024 * 1024

export function parseCoursewareComicGradeNumber(
  grade: string,
): number {
  const digitMatch =
    grade.match(
      /\d+/,
    )

  if (digitMatch) {
    const value =
      Number.parseInt(
        digitMatch[0],
        10,
      )

    return Number.isFinite(
      value,
    )
      ? value
      : 0
  }

  const map:
    Record<string, number> = {
      一: 1,
      二: 2,
      三: 3,
      四: 4,
      五: 5,
      六: 6,
      七: 7,
      八: 8,
      九: 9,
    }

  for (
    const [
      label,
      value,
    ] of Object.entries(
      map,
    )
  ) {
    if (
      grade.includes(
        label,
      )
    ) {
      return value
    }
  }

  return 0
}

export function documentKey(
  file: File,
): string {
  return [
    'document',
    file.name,
    file.size,
    file.lastModified,
  ].join(':')
}

export function imageFileKey(
  file: File,
): string {
  return [
    'image-file',
    file.name,
    file.size,
    file.lastModified,
  ].join(':')
}

export function resolveDocumentMimeType(
  file: File,
): string {
  const name =
    file.name.toLowerCase()

  if (
    name.endsWith(
      '.pdf',
    )
  ) {
    return 'application/pdf'
  }

  return 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
}

export function defaultCoursewareComicReferenceImageName(
  mimeType: string,
): string {
  switch (
    mimeType.toLowerCase()
  ) {
  case 'image/jpeg':
  case 'image/jpg':
    return 'reference-image.jpg'

  case 'image/webp':
    return 'reference-image.webp'

  case 'image/gif':
    return 'reference-image.gif'

  case 'image/svg+xml':
    return 'reference-image.svg'

  default:
    return 'reference-image.png'
  }
}

export function resolveReferencePickerErrorMessage(
  error: unknown,
  fallback: string,
): string {
  return error instanceof Error &&
    error.message.trim()
    ? error.message
    : fallback
}

export const containerStyle:
  CSSProperties = {
    marginTop: 10,
    border:
      '1px solid #E2E8F0',
    borderRadius: 10,
    background: '#FFFFFF',
  }

export const summaryStyle:
  CSSProperties = {
    display: 'flex',
    justifyContent:
      'space-between',
    alignItems: 'center',
    gap: 10,
    padding: '10px 12px',
    color: '#475569',
    fontSize: 11,
    fontWeight: 800,
    cursor: 'pointer',
  }

export const countStyle:
  CSSProperties = {
    padding: '2px 7px',
    borderRadius: 999,
    background:
      'rgba(124,58,237,0.08)',
    color: '#7C3AED',
    fontSize: 9,
  }

export const bodyStyle:
  CSSProperties = {
    padding: '0 12px 12px',
  }

export const hintStyle:
  CSSProperties = {
    marginBottom: 9,
    color: '#64748B',
    fontSize: 9,
    lineHeight: 1.6,
  }

export const selectedListStyle:
  CSSProperties = {
    display: 'flex',
    flexWrap: 'wrap',
    gap: 6,
    marginBottom: 10,
  }

export const selectedItemStyle:
  CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 5,
    maxWidth: '100%',
    padding: '5px 7px',
    borderRadius: 8,
    background: '#F5F3FF',
    color: '#5B21B6',
    fontSize: 9,
  }

export const removeButtonStyle:
  CSSProperties = {
    border: 'none',
    background: 'transparent',
    color: '#7C3AED',
    fontSize: 14,
    cursor: 'pointer',
  }

export const gridStyle:
  CSSProperties = {
    display: 'grid',
    gridTemplateColumns:
      'repeat(auto-fit,minmax(220px,1fr))',
    gap: 8,
  }

export const cardStyle:
  CSSProperties = {
    padding: 9,
    border:
      '1px solid #E2E8F0',
    borderRadius: 9,
    background: '#F8FAFC',
  }

export const cardTitleStyle:
  CSSProperties = {
    marginBottom: 7,
    color: '#334155',
    fontSize: 10,
    fontWeight: 800,
  }

export const cardBodyStyle:
  CSSProperties = {
    display: 'grid',
    gap: 6,
  }

export const controlStyle:
  CSSProperties = {
    width: '100%',
    boxSizing: 'border-box',
    padding: '7px 8px',
    border:
      '1px solid #CBD5E1',
    borderRadius: 7,
    background: '#FFFFFF',
    color: '#334155',
    fontSize: 9,
  }

export const textareaStyle:
  CSSProperties = {
    ...controlStyle,
    resize: 'vertical',
    lineHeight: 1.5,
  }

export const addButtonStyle:
  CSSProperties = {
    padding: '7px 9px',
    border:
      '1px solid rgba(124,58,237,0.28)',
    borderRadius: 7,
    background:
      'rgba(124,58,237,0.07)',
    color: '#7C3AED',
    fontSize: 9,
    fontWeight: 800,
    cursor: 'pointer',
  }

export const uploadLabelStyle:
  CSSProperties = {
    ...addButtonStyle,
    display: 'block',
    textAlign: 'center',
  }

export const hiddenInputStyle:
  CSSProperties = {
    display: 'none',
  }

export const emptyStyle:
  CSSProperties = {
    color: '#94A3B8',
    fontSize: 9,
    lineHeight: 1.5,
  }

export const messageStyle:
  CSSProperties = {
    marginTop: 8,
    color: '#64748B',
    fontSize: 9,
    lineHeight: 1.5,
  }
