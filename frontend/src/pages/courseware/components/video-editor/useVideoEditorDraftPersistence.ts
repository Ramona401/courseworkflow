/**
 * useVideoEditorDraftPersistence.ts — 视频编辑器草稿持久化治理
 *
 * 职责：
 *   1. 加载、保存和删除当前用户在当前课件中的视频草稿；
 *   2. 保存草稿后使用后端返回的真实 draft_id 绑定字幕；
 *   3. 加载草稿时优先按 draft_id 读取字幕；
 *   4. 新格式字幕不存在时，兼容读取历史 scope_id 为空的编辑器字幕；
 *   5. 删除草稿后同步更新前端草稿列表；
 *   6. 将草稿代码从主编辑器组件中拆出，保持单文件低于600行。
 *
 * 兼容规则：
 *   - 新保存的草稿：字幕使用 editor_draft + draft_id；
 *   - 已加载的新草稿：TTS和导出继续更新该draft_id下的字幕；
 *   - 尚未保存的新编辑会话：仍允许使用历史空scope_id字幕口径；
 *   - 加载历史草稿后，一旦再次保存/TTS/导出，会自然迁移到真实draft_id。
 */

import {
  useCallback,
  useEffect,
  useState,
} from 'react'
import type {
  Dispatch,
  SetStateAction,
} from 'react'
import {
  deleteVideoDraft,
  listSubtitles,
  listVideoDrafts,
  saveVideoDraft,
  upsertSubtitle,
} from '../../../../api/coursewares'
import type {
  CoursewareSubtitle,
  VideoDraftItem,
} from '../../../../api/coursewares'
import type {
  EditorClip,
} from './VideoEditorTypes'
import type {
  SubtitleSegment,
} from './VideoEditorSubtitleTrack'

interface DraftPersistenceParams {
  coursewareId?: string
  clips: EditorClip[]
  setClips: Dispatch<SetStateAction<EditorClip[]>>
  subtitleSegments: SubtitleSegment[]
  setSubtitleSegments: Dispatch<SetStateAction<SubtitleSegment[]>>
  subtitleLanguage: string
  setSubtitleLanguage: Dispatch<SetStateAction<string>>
  setSubtitleDbId: Dispatch<SetStateAction<string>>
}

interface ParsedSubtitle {
  item: CoursewareSubtitle
  segments: SubtitleSegment[]
}

interface DraftPersistenceResult {
  drafts: VideoDraftItem[]
  draftDismissed: boolean
  setDraftDismissed: Dispatch<SetStateAction<boolean>>
  activeDraftId: string
  saveDraftToServer: (name?: string) => Promise<void>
  loadDraft: (draft: VideoDraftItem) => Promise<void>
  deleteDraftById: (draftId: string) => Promise<void>
}

/**
 * 判断未知值是否可作为编辑器视频片段恢复。
 *
 * 服务端已经对新草稿做严格校验；这里仍保留客户端防御，
 * 以兼容治理前写入的历史草稿。
 */
function isUsableDraftClip(
  value: unknown,
): value is EditorClip {
  if (
    typeof value !== 'object'
    || value === null
  ) {
    return false
  }

  const candidate = value as Partial<EditorClip>

  return (
    typeof candidate.id === 'string'
    && candidate.id.trim().length > 0
    && typeof candidate.url === 'string'
    && candidate.url.trim().length > 0
    && typeof candidate.duration === 'number'
    && Number.isFinite(candidate.duration)
    && candidate.duration > 0
    && typeof candidate.trimStart === 'number'
    && Number.isFinite(candidate.trimStart)
    && candidate.trimStart >= 0
    && typeof candidate.trimEnd === 'number'
    && Number.isFinite(candidate.trimEnd)
    && candidate.trimEnd > candidate.trimStart
  )
}

/**
 * 从字幕候选中选择一个可解析版本。
 *
 * 选择优先级：
 *   1. 当前编辑器语言；
 *   2. updated_at最新；
 *   3. 其它可成功解析的字幕。
 */
function pickUsableSubtitle(
  items: CoursewareSubtitle[],
  preferredLanguage: string,
): ParsedSubtitle | null {
  const sorted = [...items].sort((left, right) => {
    const leftTime = Date.parse(left.updated_at) || 0
    const rightTime = Date.parse(right.updated_at) || 0

    const leftPreferred =
      left.language === preferredLanguage ? 1 : 0
    const rightPreferred =
      right.language === preferredLanguage ? 1 : 0

    if (leftPreferred !== rightPreferred) {
      return rightPreferred - leftPreferred
    }

    return rightTime - leftTime
  })

  for (const item of sorted) {
    try {
      const parsed: unknown = JSON.parse(item.segments)

      if (
        Array.isArray(parsed)
        && parsed.every((segment) => (
          typeof segment === 'object'
          && segment !== null
        ))
      ) {
        return {
          item,
          segments: parsed as SubtitleSegment[],
        }
      }
    } catch {
      // 历史损坏字幕跳过，继续尝试下一条候选。
    }
  }

  return null
}

/**
 * 移除不能跨刷新保存的blob URL。
 */
function serializeDraftClips(
  clips: EditorClip[],
): EditorClip[] {
  return clips.map((clip) => {
    const {
      blobUrl: _blobUrl,
      ...persistentClip
    } = clip

    void _blobUrl

    return persistentClip
  })
}

/**
 * 视频编辑器草稿持久化Hook。
 */
export default function useVideoEditorDraftPersistence({
  coursewareId,
  clips,
  setClips,
  subtitleSegments,
  setSubtitleSegments,
  subtitleLanguage,
  setSubtitleLanguage,
  setSubtitleDbId,
}: DraftPersistenceParams): DraftPersistenceResult {
  const [drafts, setDrafts] =
    useState<VideoDraftItem[]>([])

  const [draftDismissed, setDraftDismissed] =
    useState(false)

  /**
   * 当前工作区关联的持久化草稿ID。
   *
   * 空字符串表示尚未保存的新编辑会话，此时字幕继续使用
   * 历史兼容的editor_draft空scope_id。
   */
  const [activeDraftId, setActiveDraftId] =
    useState('')

  /**
   * 课件变化时重新加载当前用户草稿。
   */
  useEffect(() => {
    let cancelled = false

    setDrafts([])
    setDraftDismissed(false)
    setActiveDraftId('')

    if (!coursewareId) {
      return () => {
        cancelled = true
      }
    }

    listVideoDrafts(coursewareId)
      .then((response) => {
        if (cancelled) {
          return
        }

        setDrafts(
          Array.isArray(response.drafts)
            ? response.drafts
            : [],
        )
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return
        }

        console.warn(
          '[视频草稿] 列表加载失败:',
          error,
        )
        setDrafts([])
      })

    return () => {
      cancelled = true
    }
  }, [coursewareId])

  /**
   * 按真实draft_id读取字幕。
   *
   * 没有命中新格式字幕时，再读取scope_id为空的历史字幕。
   */
  const loadSubtitleForDraft = useCallback(
    async (
      draftId: string,
    ): Promise<void> => {
      if (!coursewareId) {
        setSubtitleSegments([])
        setSubtitleDbId('')
        return
      }

      setSubtitleSegments([])
      setSubtitleDbId('')

      try {
        const exactItems = await listSubtitles(
          coursewareId,
          'editor_draft',
          draftId,
        )

        let selected = pickUsableSubtitle(
          exactItems,
          subtitleLanguage,
        )

        if (!selected) {
          const legacyItems = await listSubtitles(
            coursewareId,
            'editor_draft',
          )

          selected = pickUsableSubtitle(
            legacyItems.filter(
              (item) => !item.scope_id,
            ),
            subtitleLanguage,
          )
        }

        if (!selected) {
          return
        }

        setSubtitleSegments(
          selected.segments,
        )
        setSubtitleLanguage(
          selected.item.language || 'zh-CN',
        )
        setSubtitleDbId(
          selected.item.id || '',
        )
      } catch (error: unknown) {
        console.warn(
          '[视频草稿] 字幕恢复失败:',
          error,
        )
        setSubtitleSegments([])
        setSubtitleDbId('')
      }
    },
    [
      coursewareId,
      setSubtitleDbId,
      setSubtitleLanguage,
      setSubtitleSegments,
      subtitleLanguage,
    ],
  )

  /**
   * 保存草稿，并将当前字幕绑定到刚创建的真实draft_id。
   */
  const saveDraftToServer = useCallback(
    async (
      name?: string,
    ): Promise<void> => {
      if (
        !coursewareId
        || clips.length === 0
      ) {
        return
      }

      const clipsData =
        serializeDraftClips(clips)

      const result = await saveVideoDraft(
        coursewareId,
        {
          name: name || '',
          clips_data: clipsData,
          clip_count: clipsData.length,
        },
      )

      setActiveDraftId(result.id)

      setDrafts((current) => [
        {
          id: result.id,
          courseware_id: coursewareId,
          name: name || '',
          clips_data: clipsData,
          clip_count: clipsData.length,
          created_at: result.created_at,
        },
        ...current.filter(
          (draft) => draft.id !== result.id,
        ),
      ].slice(0, 20))

      if (subtitleSegments.length === 0) {
        return
      }

      try {
        const subtitle = await upsertSubtitle(
          coursewareId,
          {
            scope_type: 'editor_draft',
            scope_id: result.id,
            language: subtitleLanguage,
            segments: JSON.stringify(
              subtitleSegments,
            ),
          },
        )

        if (subtitle?.id) {
          setSubtitleDbId(subtitle.id)
        }
      } catch (error: unknown) {
        /**
         * 草稿主体已经保存成功，字幕保存失败不撤销草稿。
         * 与历史行为保持一致，避免用户退出时丢失整个剪辑版本。
         */
        console.warn(
          '[视频草稿] 草稿字幕保存失败:',
          error,
        )
      }
    },
    [
      clips,
      coursewareId,
      setSubtitleDbId,
      subtitleLanguage,
      subtitleSegments,
    ],
  )

  /**
   * 恢复草稿片段，并按draft_id恢复对应字幕。
   */
  const loadDraft = useCallback(
    async (
      draft: VideoDraftItem,
    ): Promise<void> => {
      try {
        const rawData: unknown =
          Array.isArray(draft.clips_data)
            ? draft.clips_data
            : JSON.parse(
              String(draft.clips_data),
            )

        if (!Array.isArray(rawData)) {
          throw new Error(
            '草稿片段不是数组',
          )
        }

        const validClips =
          rawData.filter(
            isUsableDraftClip,
          )

        if (validClips.length === 0) {
          alert(
            '草稿中的视频已全部失效，请重新编辑',
          )
          return
        }

        if (validClips.length < rawData.length) {
          alert(
            `注意：${rawData.length - validClips.length} 个片段因数据失效已自动跳过`,
          )
        }

        setClips(
          validClips.map((clip) => ({
            ...clip,
            blobUrl: undefined,
            trackType:
              clip.trackType || 'video',
          })),
        )

        setDraftDismissed(true)
        setActiveDraftId(draft.id)

        await loadSubtitleForDraft(
          draft.id,
        )
      } catch (error: unknown) {
        console.warn(
          '[视频草稿] 草稿解析失败:',
          error,
        )
        alert('草稿数据解析失败')
      }
    },
    [
      loadSubtitleForDraft,
      setClips,
    ],
  )

  /**
   * 删除草稿。
   *
   * 后端会在同一事务中删除该draft_id绑定的字幕。
   */
  const deleteDraftById = useCallback(
    async (
      draftId: string,
    ): Promise<void> => {
      if (!coursewareId) {
        return
      }

      await deleteVideoDraft(
        coursewareId,
        draftId,
      )

      setDrafts((current) =>
        current.filter(
          (draft) => draft.id !== draftId,
        ),
      )

      if (activeDraftId === draftId) {
        setActiveDraftId('')
        setSubtitleSegments([])
        setSubtitleDbId('')
      }
    },
    [
      activeDraftId,
      coursewareId,
      setSubtitleDbId,
      setSubtitleSegments,
    ],
  )

  return {
    drafts,
    draftDismissed,
    setDraftDismissed,
    activeDraftId,
    saveDraftToServer,
    loadDraft,
    deleteDraftById,
  }
}
