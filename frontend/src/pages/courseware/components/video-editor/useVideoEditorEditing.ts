/**
 * useVideoEditorEditing.ts — 视频编辑器剪辑业务
 *
 * 职责：
 *   - 素材时长探测；
 *   - 片段新增、删除和字段更新；
 *   - 时间轴拖拽排序及素材拖入；
 *   - 音轨分离、删除和恢复原视频；
 *   - 视频文件上传。
 */

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'
import type {
  Dispatch,
  DragEvent,
  SetStateAction,
} from 'react'
import {
  extractCWAudio,
  muteCWVideo,
} from '../../../../api/coursewares'
import {
  DRAG_TYPE_ASSET,
} from './VideoEditorConstants'
import {
  fmtSize,
  urlToBlobUrl,
} from './VideoEditorUtils'
import type {
  EditorClip,
  VideoEditorModalProps,
} from './VideoEditorTypes'

type EditorVideo = VideoEditorModalProps['videos'][number]

interface EditingParams {
  videos: VideoEditorModalProps['videos']
  coursewareId?: string
  onUploadVideo: VideoEditorModalProps['onUploadVideo']
  clips: EditorClip[]
  setClips: Dispatch<SetStateAction<EditorClip[]>>
  activeIdx: number
  setActiveIdx: Dispatch<SetStateAction<number>>
  stopAll: () => void
  queuePlay: (index: number) => void
}

/**
 * 视频编辑器剪辑业务Hook。
 */
export default function useVideoEditorEditing({
  videos,
  coursewareId,
  onUploadVideo,
  clips,
  setClips,
  activeIdx,
  setActiveIdx,
  stopAll,
  queuePlay,
}: EditingParams) {
  const [dragIdx, setDragIdx] = useState(-1)
  const [dragOverIdx, setDragOverIdx] = useState(-1)
  const [timelineDragOver, setTimelineDragOver] = useState(false)
  const [assetDurations, setAssetDurations] =
    useState<Record<string, number>>({})

  const [separatingIdx, setSeparatingIdx] = useState(-1)
  const [downloadPct, setDownloadPct] = useState(0)
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)

  const loadedDurationIDs = useRef<Set<string>>(new Set())

  const loadDuration = useCallback(
    (url: string): Promise<number> => new Promise((resolve) => {
      const video = document.createElement('video')

      video.preload = 'metadata'
      video.onloadedmetadata = () => {
        resolve(video.duration || 5)
        video.remove()
      }
      video.onerror = () => {
        resolve(5)
        video.remove()
      }
      video.src = url
    }),
    [],
  )

  useEffect(() => {
    videos.forEach(async (video) => {
      if (loadedDurationIDs.current.has(video.id)) return

      loadedDurationIDs.current.add(video.id)
      const duration = await loadDuration(video.url)

      setAssetDurations((current) => ({
        ...current,
        [video.id]: duration,
      }))
    })
  }, [loadDuration, videos])

  const updateClip = useCallback((
    index: number,
    update: Partial<EditorClip>,
  ) => {
    setClips((current) => current.map(
      (clip, clipIndex) => (
        clipIndex === index
          ? { ...clip, ...update }
          : clip
      ),
    ))
  }, [setClips])

  const addClip = useCallback(async (video: EditorVideo) => {
    if (clips.some((clip) => clip.id === video.id)) return

    const duration = assetDurations[video.id]
      || await loadDuration(video.url)

    setClips((current) => [
      ...current,
      {
        id: video.id,
        url: video.url,
        label: video.label,
        duration,
        trimStart: 0,
        trimEnd: duration,
        transition: 'none',
        transDur: 0.5,
        trackType: 'video',
        audioMuted: false,
        audioVolume: 1,
      },
    ])
  }, [assetDurations, clips, loadDuration, setClips])

  const removeClip = useCallback((index: number) => {
    const clip = clips[index]

    if (clip?.blobUrl) {
      try {
        URL.revokeObjectURL(clip.blobUrl)
      } catch {
        // blob URL可能已经释放。
      }
    }

    stopAll()

    setClips((current) => current.filter(
      (_, clipIndex) => clipIndex !== index,
    ))

    if (activeIdx === index) {
      setActiveIdx(-1)
    } else if (activeIdx > index) {
      setActiveIdx(activeIdx - 1)
    }
  }, [
    activeIdx,
    clips,
    setActiveIdx,
    setClips,
    stopAll,
  ])

  const handleDragStart = (
    index: number,
    event: DragEvent,
  ) => {
    setDragIdx(index)
    event.dataTransfer.effectAllowed = 'move'
  }

  const handleDragOver = (
    event: DragEvent,
    index: number,
  ) => {
    event.preventDefault()
    setDragOverIdx(index)
  }

  const handleDrop = (index: number) => {
    if (dragIdx < 0 || dragIdx === index) {
      setDragIdx(-1)
      setDragOverIdx(-1)
      return
    }

    setClips((current) => {
      const next = [...current]
      const [moved] = next.splice(dragIdx, 1)

      next.splice(index, 0, moved)
      return next
    })

    setDragIdx(-1)
    setDragOverIdx(-1)
  }

  const handleTimelineDragOver = useCallback((
    event: DragEvent,
  ) => {
    if (!event.dataTransfer.types.includes(DRAG_TYPE_ASSET)) return

    event.preventDefault()
    event.dataTransfer.dropEffect = 'copy'
    setTimelineDragOver(true)
  }, [])

  const handleTimelineDragLeave = useCallback(() => {
    setTimelineDragOver(false)
  }, [])

  const handleTimelineDrop = useCallback((
    event: DragEvent,
  ) => {
    setTimelineDragOver(false)

    const videoID = event.dataTransfer.getData(DRAG_TYPE_ASSET)
    const video = videos.find((item) => item.id === videoID)

    if (video) void addClip(video)
  }, [addClip, videos])

  const handleSeparateAudio = useCallback(async () => {
    if (
      !coursewareId
      || activeIdx < 0
      || separatingIdx >= 0
    ) {
      return
    }

    const clip = clips[activeIdx]
    if (!clip || clip.muted) return

    setSeparatingIdx(activeIdx)
    setDownloadPct(0)

    try {
      const muted = await muteCWVideo(coursewareId, clip.id)
      let blobUrl: string | undefined
      let blobFailure: string | null = null

      try {
        blobUrl = await urlToBlobUrl(
          muted.url,
          (received, total) => {
            if (total <= 0) return

            setDownloadPct(
              Math.min(99, Math.round(received / total * 100)),
            )
          },
        )
      } catch (error) {
        blobFailure = error instanceof Error
          ? error.message
          : '未知错误'

        console.warn(
          '[视频编辑器blob缓存降级]',
          blobFailure,
        )
      }

      let audioInfo: {
        url: string
        duration: string
        fileSize: number
      } | null = null

      try {
        const audio = await extractCWAudio(coursewareId, clip.id)

        audioInfo = {
          url: audio.url,
          duration: audio.duration,
          fileSize: audio.file_size,
        }
      } catch {
        // 原视频可能没有可分离音频。
      }

      stopAll()

      updateClip(activeIdx, {
        id: muted.asset_id,
        url: muted.url,
        blobUrl,
        muted: true,
        originalId: clip.id,
        originalUrl: clip.url,
        audioUrl: audioInfo?.url,
        audioDuration: audioInfo?.duration,
        audioFileSize: audioInfo?.fileSize,
        audioMuted: false,
        audioVolume: 1,
      })

      queuePlay(activeIdx)

      if (blobFailure) {
        window.setTimeout(() => {
          alert(
            `静音已完成，但本地缓存失败可能轻微卡顿。\n${blobFailure}`,
          )
        }, 100)
      }
    } catch (error) {
      alert(
        `分离音轨失败：${
          error instanceof Error
            ? error.message
            : '未知错误'
        }`,
      )
    } finally {
      setSeparatingIdx(-1)
      setDownloadPct(0)
    }
  }, [
    activeIdx,
    clips,
    coursewareId,
    queuePlay,
    separatingIdx,
    stopAll,
    updateClip,
  ])

  const handleDeleteAudio = useCallback(() => {
    if (activeIdx < 0) return

    updateClip(activeIdx, {
      audioUrl: undefined,
      audioDuration: undefined,
      audioFileSize: undefined,
      audioMuted: false,
      audioVolume: 1,
    })
  }, [activeIdx, updateClip])

  const handleRestoreOriginal = useCallback(() => {
    if (activeIdx < 0) return

    const clip = clips[activeIdx]

    if (!clip?.originalId || !clip.originalUrl) return

    if (clip.blobUrl) {
      try {
        URL.revokeObjectURL(clip.blobUrl)
      } catch {
        // blob URL可能已经释放。
      }
    }

    stopAll()

    updateClip(activeIdx, {
      id: clip.originalId,
      url: clip.originalUrl,
      blobUrl: undefined,
      muted: false,
      originalId: undefined,
      originalUrl: undefined,
      audioUrl: undefined,
      audioDuration: undefined,
      audioFileSize: undefined,
      audioMuted: false,
      audioVolume: 1,
    })

    queuePlay(activeIdx)
  }, [
    activeIdx,
    clips,
    queuePlay,
    stopAll,
    updateClip,
  ])

  const handleUploadVideo = useCallback(() => {
    if (!onUploadVideo) return

    const input = document.createElement('input')

    input.type = 'file'
    input.accept =
      'video/mp4,video/webm,video/quicktime,.mp4,.webm,.mov,.avi'

    input.onchange = async (event) => {
      const file = (event.target as HTMLInputElement).files?.[0]

      if (!file) return

      if (file.size > 50 * 1024 * 1024) {
        alert('视频不能超过50MB')
        return
      }

      setUploading(true)
      setUploadProgress(0)

      try {
        const result = await onUploadVideo(
          file,
          setUploadProgress,
        )

        if (result) {
          alert(
            `✅ 视频上传成功：${result.label}（${fmtSize(file.size)}）`,
          )
        }
      } catch (error) {
        alert(
          `❌ 上传失败：${
            error instanceof Error
              ? error.message
              : '未知错误'
          }`,
        )
      } finally {
        setUploading(false)
        setUploadProgress(0)
      }
    }

    input.click()
  }, [onUploadVideo])

  return {
    dragIdx,
    dragOverIdx,
    timelineDragOver,
    assetDurations,
    separatingIdx,
    downloadPct,
    uploading,
    uploadProgress,
    setDragIdx,
    setDragOverIdx,
    updateClip,
    addClip,
    removeClip,
    handleDragStart,
    handleDragOver,
    handleDrop,
    handleTimelineDragOver,
    handleTimelineDragLeave,
    handleTimelineDrop,
    handleSeparateAudio,
    handleDeleteAudio,
    handleRestoreOriginal,
    handleUploadVideo,
  }
}
