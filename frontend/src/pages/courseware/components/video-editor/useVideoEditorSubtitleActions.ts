/**
 * useVideoEditorSubtitleActions.ts — 字幕TTS和导出动作
 *
 * 职责：
 *   - 在当前editor_draft scope中保存字幕；
 *   - 打开TTS生成窗口；
 *   - 导出前持久化字幕；
 *   - 根据用户选择生成视频、烧录字幕或混入配音；
 *   - 把TTS生成结果合并回时间轴字幕。
 */

import {
  useCallback,
  useState,
} from 'react'
import type {
  Dispatch,
  SetStateAction,
} from 'react'
import {
  upsertSubtitle,
} from '../../../../api/coursewares'
import type {
  EditorClip,
  VideoEditorModalProps,
} from './VideoEditorTypes'
import type {
  ExportMode,
} from './VideoEditorExportDialog'
import type {
  SubtitleSegment,
} from './VideoEditorSubtitleTrack'

interface SubtitleActionParams {
  coursewareId?: string
  activeDraftId: string
  clips: EditorClip[]
  exporting: boolean
  onExport: VideoEditorModalProps['onExport']
  stopAll: () => void
  subtitleSegments: SubtitleSegment[]
  setSubtitleSegments: Dispatch<SetStateAction<SubtitleSegment[]>>
  subtitleLanguage: string
  subtitleDbId: string
  setSubtitleDbId: Dispatch<SetStateAction<string>>
}

/**
 * 字幕TTS和导出动作Hook。
 */
export default function useVideoEditorSubtitleActions({
  coursewareId,
  activeDraftId,
  clips,
  exporting,
  onExport,
  stopAll,
  subtitleSegments,
  setSubtitleSegments,
  subtitleLanguage,
  subtitleDbId,
  setSubtitleDbId,
}: SubtitleActionParams) {
  const [showTTSModal, setShowTTSModal] = useState(false)
  const [showExportDialog, setShowExportDialog] = useState(false)
  const [exportSubtitleID, setExportSubtitleID] = useState('')

  const buildExportClips = useCallback(
    () => clips.map((clip) => ({
      asset_id: clip.id,
      start_sec: clip.trimStart,
      end_sec: clip.trimEnd,
      transition: clip.transition,
      trans_dur: clip.transDur,
    })),
    [clips],
  )

  const handleRequestTTS = useCallback(() => {
    if (!coursewareId || subtitleSegments.length === 0) {
      alert('请先添加字幕条')
      return
    }

    if (!subtitleSegments.some(
      (segment) => segment.text.trim().length > 0,
    )) {
      alert('字幕内容为空，请先双击字幕条输入文本')
      return
    }

    upsertSubtitle(coursewareId, {
      scope_type: 'editor_draft',
      scope_id: activeDraftId || undefined,
      language: subtitleLanguage,
      segments: JSON.stringify(subtitleSegments),
    })
      .then((result) => {
        if (result?.id) setSubtitleDbId(result.id)
        setShowTTSModal(true)
      })
      .catch((error) => {
        alert(
          `保存字幕失败：${
            error instanceof Error
              ? error.message
              : '未知错误'
          }`,
        )
      })
  }, [
    activeDraftId,
    coursewareId,
    setSubtitleDbId,
    subtitleLanguage,
    subtitleSegments,
  ])

  const handleExport = useCallback(async () => {
    if (clips.length === 0 || exporting) return

    stopAll()

    let subtitleID = subtitleDbId

    if (coursewareId && subtitleSegments.length > 0) {
      try {
        const result = await upsertSubtitle(coursewareId, {
          scope_type: 'editor_draft',
          scope_id: activeDraftId || undefined,
          language: subtitleLanguage,
          segments: JSON.stringify(subtitleSegments),
        })

        if (result?.id) {
          subtitleID = result.id
          setSubtitleDbId(result.id)
        }
      } catch (error) {
        console.warn(
          '[字幕持久化] 导出前保存失败:',
          error,
        )
      }
    }

    const textCount = subtitleSegments.filter(
      (segment) => segment.text.trim().length > 0,
    ).length

    if (textCount === 0 || !subtitleID) {
      onExport(buildExportClips(), {})
      return
    }

    setExportSubtitleID(subtitleID)
    setShowExportDialog(true)
  }, [
    activeDraftId,
    buildExportClips,
    clips.length,
    coursewareId,
    exporting,
    onExport,
    setSubtitleDbId,
    stopAll,
    subtitleDbId,
    subtitleLanguage,
    subtitleSegments,
  ])

  const handleExportConfirm = useCallback((mode: ExportMode) => {
    setShowExportDialog(false)

    onExport(buildExportClips(), {
      burnSubtitle: mode !== 'video',
      mixNarration: mode === 'narration',
      subtitleId: exportSubtitleID,
    })
  }, [
    buildExportClips,
    exportSubtitleID,
    onExport,
  ])

  const handleTTSComplete = useCallback((
    updated: SubtitleSegment[],
  ) => {
    setSubtitleSegments((current) => current.map((segment) => {
      const match = updated.find((item) => item.id === segment.id)

      if (!match) return segment

      return {
        ...segment,
        tts_audio_url: match.tts_audio_url,
        tts_voice: match.tts_voice,
        tts_duration: match.tts_duration,
        tts_generated_at: match.tts_generated_at,
      }
    }))
  }, [setSubtitleSegments])

  const subtitleCount = subtitleSegments.filter(
    (segment) => segment.text.trim().length > 0,
  ).length

  const narratedCount = subtitleSegments.filter(
    (segment) => !!segment.tts_audio_url,
  ).length

  return {
    showTTSModal,
    setShowTTSModal,
    showExportDialog,
    setShowExportDialog,
    subtitleCount,
    narratedCount,
    handleRequestTTS,
    handleExport,
    handleExportConfirm,
    handleTTSComplete,
  }
}
