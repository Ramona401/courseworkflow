/**
 * useLessonDocumentImageRemoval.ts — 预览区图片快捷移除流程。
 *
 * 删除成功后复用页面现有正文保存接口，自动产生正文历史版本。
 * 原始 Word 母版和物理图片文件仍然保留。
 */

import {
  useCallback,
  useState,
} from 'react'
import type {
  RenderedMarkdownImage,
} from '@/pages/lesson-plans/plan-detail/components/planDetailConstants'
import {
  removeMarkdownImageOccurrence,
} from './lessonDocumentImages'

type MessageType =
  'success' |
  'error' |
  'info'

interface ImageRemovalParams {
  content: string
  planID: string
  disabled: boolean
  disabledReason: string
  saving: boolean
  uploading: boolean
  onSave: (
    nextContent: string,
  ) => Promise<void>
  showMessage: (
    text: string,
    type?: MessageType,
  ) => void
}

export function useLessonDocumentImageRemoval({
  content,
  planID,
  disabled,
  disabledReason,
  saving,
  uploading,
  onSave,
  showMessage,
}: ImageRemovalParams) {
  const [
    removingImageKey,
    setRemovingImageKey,
  ] = useState('')

  const removePreviewImage =
    useCallback(
      async (
        image: RenderedMarkdownImage,
        rangeStart: number,
        rangeEnd: number,
      ) => {
        if (
          disabled ||
          saving ||
          uploading ||
          removingImageKey
        ) {
          showMessage(
            removingImageKey
              ? '当前正在移除图片，请稍候'
              : disabledReason ||
                '当前正在处理正文，请稍后再移除图片',
            'info',
          )
          return
        }

        if (!planID) {
          showMessage(
            '教案ID缺失，暂时无法移除图片',
            'error',
          )
          return
        }

        const result =
          removeMarkdownImageOccurrence(
            content || '',
            image,
            rangeStart,
            rangeEnd,
          )

        if (!result.removed) {
          showMessage(
            '没有找到对应图片，请刷新页面后重试',
            'error',
          )
          return
        }

        if (!result.content.trim()) {
          showMessage(
            '不能移除正文中的唯一内容',
            'error',
          )
          return
        }

        const confirmed =
          window.confirm(
            '确认从当前教案正文移除这张图片吗？\n\n' +
            (
              image.alt
                ? `图片：${image.alt}\n\n`
                : ''
            ) +
            '原始Word母版和历史版本仍会保留，可通过版本历史恢复。',
          )

        if (!confirmed) return

        setRemovingImageKey(
          image.key,
        )

        try {
          await onSave(
            result.content,
          )

          showMessage(
            '图片已从当前教案正文移除；原始Word母版和历史版本仍会保留',
            'success',
          )
        } catch (error) {
          const text =
            error instanceof Error
              ? error.message
              : '正文保存失败'

          showMessage(
            `移除图片失败：${text}`,
            'error',
          )
        } finally {
          setRemovingImageKey('')
        }
      },
      [
        content,
        planID,
        disabled,
        disabledReason,
        saving,
        uploading,
        removingImageKey,
        onSave,
        showMessage,
      ],
    )

  return {
    removingImageKey,
    removePreviewImage,
  }
}
