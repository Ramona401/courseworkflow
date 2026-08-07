/**
 * useCoursewareComicWorkflowProjectEditor.ts
 *
 * 知识点漫画五步编辑器的状态、请求、SSE和并发动作编排。
 *
 * 主编辑器只负责渲染步骤。本Hook保持：
 *   - 所有写操作携带服务端version；
 *   - 同一时刻只启动一个教师动作；
 *   - SSE与5秒轮询共同恢复后台生成状态；
 *   - 浏览步骤独立于服务端生产步骤，回看不会倒写数据库。
 */
import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'
import {
  confirmCoursewareComicStoryboard,
  confirmCoursewareComicStylePreview,
  generateCoursewareComicProject,
  generateCoursewareComicStylePreview,
  getCoursewareComicWorkflowProject,
  insertCoursewareComicPage,
  planCoursewareComicWorkflowProject,
  regenerateCoursewareComicPanel,
  subscribeCoursewareComicGeneration,
  syncCoursewareComicPanelPage,
  updateCoursewareComicStyleSettings,
} from '@/api/coursewares'
import type {
  CoursewareComicGenerationEvent,
  CoursewareComicNarrativeMode,
  CoursewareComicPanel,
  CoursewareComicProject,
  CoursewareComicStyleSettingsDraft,
  CoursewareComicWorkflowProjectDetail,
  CoursewareComicWorkflowStage,
} from '@/api/coursewares'
import {
  updateCoursewareComicStoryboardPanel,
} from '@/api/coursewares.comic.storyboard'
import type {
  CoursewareComicStoryboardPanelDraft,
} from '@/api/coursewares.comic.storyboard'
import {
  coursewareComicStageIndex,
  createCoursewareComicStyleSettingsDraft,
  resolveCoursewareComicEffectiveStage,
} from './coursewareComicWorkflow'
type WorkflowAction =
  | ''
  | 'plan'
  | 'confirm_storyboard'
  | 'save_style'
  | 'preview'
  | 'confirm_preview'
  | 'batch'
  | 'insert'
interface UseCoursewareComicWorkflowProjectEditorOptions {
  coursewareId: string
  projectId: string
  onProjectChanged?: (
    project: CoursewareComicProject,
  ) => void
  onPagesChanged?: (
    pageNumber: number,
  ) => void | Promise<void>
}

export default function useCoursewareComicWorkflowProjectEditor({
  coursewareId,
  projectId,
  onProjectChanged,
  onPagesChanged,
}: UseCoursewareComicWorkflowProjectEditorOptions) {
  const [detail, setDetail] =
    useState<CoursewareComicWorkflowProjectDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [action, setAction] = useState<WorkflowAction>('')
  const [notice, setNotice] = useState('')
  const [previewTaskActive, setPreviewTaskActive] = useState(false)
  const [regeneratingPanelID, setRegeneratingPanelID] = useState('')
  const [syncingPanelID, setSyncingPanelID] = useState('')
  const [savingStoryboardPanelID, setSavingStoryboardPanelID] = useState('')
  const [selectedStage, setSelectedStage] =
    useState<CoursewareComicWorkflowStage | null>(null)
  const sseRef = useRef<{ close: () => void } | null>(null)
  const effectiveStageRef = useRef<CoursewareComicWorkflowStage | null>(null)
  const applyDetail = useCallback(
    (result: CoursewareComicWorkflowProjectDetail) => {
      const nextEffective =
        resolveCoursewareComicEffectiveStage(result.project)
      const previousEffective = effectiveStageRef.current
      setSelectedStage(previous => {
        if (!previous || previous === previousEffective) {
          return nextEffective
        }
        if (
          coursewareComicStageIndex(previous) >
          coursewareComicStageIndex(nextEffective)
        ) {
          return nextEffective
        }
        return previous
      })
      effectiveStageRef.current = nextEffective
      setDetail(result)
      onProjectChanged?.(result.project)
      return result
    },
    [onProjectChanged],
  )
  const refreshDetail = useCallback(async () => {
    const result = await getCoursewareComicWorkflowProject(
      coursewareId,
      projectId,
    )
    return applyDetail(result)
  }, [coursewareId, projectId, applyDetail])
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setDetail(null)
    setSelectedStage(null)
    effectiveStageRef.current = null
    refreshDetail()
      .catch(error => {
        if (!cancelled) {
          setNotice(
            '❌ ' +
              errorMessage(error, '漫画项目加载失败'),
          )
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [refreshDetail])
  const handleGenerationEvent = useCallback(
    (event: CoursewareComicGenerationEvent) => {
      if (event.project_id !== projectId) {
        return
      }
      if (event.message) {
        setNotice(
          generationPrefix(event.stage) +
            event.message,
        )
      }
      if (
        event.stage === 'panel_done' ||
        event.stage === 'panel_failed'
      ) {
        setRegeneratingPanelID(previous =>
          previous === event.panel_id
            ? ''
            : previous,
        )
        if (event.panel_no === 1) {
          setPreviewTaskActive(false)
          setAction(previous =>
            previous === 'preview'
              ? ''
              : previous,
          )
        }
      }
      if (
        event.stage === 'project_done' ||
        event.stage === 'project_failed'
      ) {
        setAction('')
        setRegeneratingPanelID('')
        if (event.stage === 'project_done') {
          setSelectedStage('refinement')
        }
      }
      void refreshDetail()
    },
    [projectId, refreshDetail],
  )
  const startProgressStream = useCallback(() => {
    sseRef.current?.close()
    sseRef.current =
      subscribeCoursewareComicGeneration(
        coursewareId,
        {
          onConnected: () => {
            setNotice('⏳ 已连接漫画生成进度。')
          },
          onReconnected: () => {
            setNotice(
              '🔄 生成进度连接已恢复，正在同步服务器状态…',
            )
            void refreshDetail()
          },
          onEvent: handleGenerationEvent,
          onTransportError: message => {
            setNotice('⚠️ ' + message)
          },
        },
      )
  }, [
    coursewareId,
    refreshDetail,
    handleGenerationEvent,
  ])
  useEffect(() => {
    const projectGenerating =
      detail?.project.status === 'generating'
    const panelGenerating =
      detail?.panels.some(
        panel => panel.status === 'generating',
      ) || false
    if (
      (
        projectGenerating ||
        panelGenerating ||
        previewTaskActive
      ) &&
      !sseRef.current
    ) {
      startProgressStream()
    }
  }, [
    detail,
    previewTaskActive,
    startProgressStream,
  ])
  useEffect(() => {
    const projectGenerating =
      detail?.project.status === 'generating'
    const panelGenerating =
      detail?.panels.some(
        panel => panel.status === 'generating',
      ) || false
    if (
      !projectGenerating &&
      !panelGenerating &&
      !previewTaskActive
    ) {
      return
    }
    const timer = window.setInterval(
      () => {
        void refreshDetail()
      },
      5000,
    )
    return () => {
      window.clearInterval(timer)
    }
  }, [
    detail,
    previewTaskActive,
    refreshDetail,
  ])
  useEffect(() => {
    if (!previewTaskActive || !detail) {
      return
    }
    const firstPanel = detail.panels.find(
      panel => panel.panel_no === 1,
    )
    if (
      firstPanel &&
      firstPanel.status !== 'generating'
    ) {
      setPreviewTaskActive(false)
      setAction(previous =>
        previous === 'preview'
          ? ''
          : previous,
      )
    }
  }, [
    detail,
    previewTaskActive,
  ])
  useEffect(() => {
    return () => {
      sseRef.current?.close()
      sseRef.current = null
    }
  }, [])
  const replacePanel = useCallback(
    (updated: CoursewareComicPanel) => {
      setDetail(previous => {
        if (!previous) {
          return previous
        }
        return {
          ...previous,
          panels: previous.panels.map(panel =>
            panel.id === updated.id
              ? updated
              : panel,
          ),
        }
      })
    },
    [],
  )
  const handleSelectStage = useCallback(
    (stage: CoursewareComicWorkflowStage) => {
      if (!detail) {
        return
      }
      const effective =
        resolveCoursewareComicEffectiveStage(
          detail.project,
        )
      if (
        coursewareComicStageIndex(stage) <=
        coursewareComicStageIndex(effective)
      ) {
        setSelectedStage(stage)
      }
    },
    [detail],
  )
  const handlePlan = async (
    narrativeMode: CoursewareComicNarrativeMode,
    teacherInstruction = '',
  ) => {
    if (
      !detail ||
      action ||
      savingStoryboardPanelID
    ) {
      return
    }
    const normalizedInstruction =
      teacherInstruction.trim()
    if (
      Array.from(normalizedInstruction).length >
      8000
    ) {
      setNotice(
        '⚠️ AI分镜优化要求不能超过8000字。',
      )
      return
    }
    setAction('plan')
    setNotice(
      normalizedInstruction
        ? '⏳ 正在按教师要求让AI重新优化全部故事分镜…'
        : '⏳ 正在按选择的叙事方式重新生成全部故事分镜…',
    )
    try {
      const result =
        await planCoursewareComicWorkflowProject(
          coursewareId,
          projectId,
          {
            expected_version:
              detail.project.version,
            teacher_instruction:
              normalizedInstruction,
            narrative_mode:
              narrativeMode,
          },
        )
      applyDetail(result)
      setSelectedStage('storyboard')
      setNotice(
        `✅ AI已生成${result.panels.length}格分镜，可以继续逐格修改。`,
      )
    } catch (error) {
      setNotice(
        '❌ ' +
          errorMessage(
            error,
            '漫画分镜重新规划失败',
          ),
      )
    } finally {
      setAction('')
    }
  }
  const handleSaveStoryboardPanel = async (
    panel: CoursewareComicPanel,
    draft: CoursewareComicStoryboardPanelDraft,
  ) => {
    if (
      !detail ||
      action ||
      savingStoryboardPanelID
    ) {
      return
    }
    setSavingStoryboardPanelID(panel.id)
    setNotice(
      `⏳ 正在保存第${panel.panel_no}格分镜…`,
    )
    try {
      const updated =
        await updateCoursewareComicStoryboardPanel(
          coursewareId,
          projectId,
          panel.id,
          {
            expected_version: panel.version,
            story_purpose:
              draft.storyPurpose,
            knowledge_claim:
              draft.knowledgeClaim,
            scene_text:
              draft.sceneText,
            action_text:
              draft.actionText,
            camera_text:
              draft.cameraText,
            knowledge_presentation:
              draft.knowledgePresentation,
          },
        )
      replacePanel(updated)
      setNotice(
        `✅ 第${panel.panel_no}格分镜已保存。`,
      )
    } catch (error) {
      setNotice(
        '❌ ' +
          errorMessage(
            error,
            `第${panel.panel_no}格分镜保存失败`,
          ),
      )
    } finally {
      setSavingStoryboardPanelID('')
    }
  }
  const handleConfirmStoryboard = async (
    narrativeMode: CoursewareComicNarrativeMode,
  ) => {
    if (
      !detail ||
      action ||
      savingStoryboardPanelID
    ) {
      return
    }
    setAction('confirm_storyboard')
    setNotice('⏳ 正在确认故事分镜…')
    try {
      const result =
        await confirmCoursewareComicStoryboard(
          coursewareId,
          projectId,
          {
            expected_version:
              detail.project.version,
            narrative_mode:
              narrativeMode,
          },
        )
      applyDetail(result)
      setSelectedStage('style_preview')
      setNotice(
        '✅ 分镜已确认，请选择画风并生成首格样张。',
      )
    } catch (error) {
      setNotice(
        '❌ ' +
          errorMessage(error, '分镜确认失败'),
      )
    } finally {
      setAction('')
    }
  }
  const saveStyleSettings = async (
    draft: CoursewareComicStyleSettingsDraft,
  ) => {
    if (!detail) {
      throw new Error('漫画项目尚未加载')
    }
    const result =
      await updateCoursewareComicStyleSettings(
        coursewareId,
        projectId,
        {
          expected_version:
            detail.project.version,
          visual_style_source:
            draft.visualStyleSource,
          visual_style:
            draft.visualStyle,
          aspect_ratio:
            draft.aspectRatio,
          image_quality:
            draft.imageQuality,
          style_instruction:
            draft.styleInstruction.trim(),
        },
      )
    return applyDetail(result)
  }
  const handleSaveStyle = async (
    draft: CoursewareComicStyleSettingsDraft,
  ) => {
    if (!detail || action) {
      return
    }
    setAction('save_style')
    setNotice('⏳ 正在保存视觉设置…')
    try {
      await saveStyleSettings(draft)
      setNotice('✅ 视觉设置已保存。')
    } catch (error) {
      setNotice(
        '❌ ' +
          errorMessage(
            error,
            '视觉设置保存失败',
          ),
      )
    } finally {
      setAction('')
    }
  }
  const startStylePreview = async (
    projectVersion: number,
  ) => {
    startProgressStream()
    setPreviewTaskActive(true)
    const result =
      await generateCoursewareComicStylePreview(
        coursewareId,
        projectId,
        projectVersion,
      )
    setNotice('⏳ ' + result.message)
    await refreshDetail()
  }
  const handleSaveAndGeneratePreview = async (
    draft: CoursewareComicStyleSettingsDraft,
  ) => {
    if (!detail || action) {
      return
    }
    setAction('preview')
    setNotice(
      '⏳ 正在保存视觉设置并启动首格完整样张…',
    )
    try {
      const currentDraft =
        createCoursewareComicStyleSettingsDraft(
          detail.project,
        )
      const changed =
        JSON.stringify(currentDraft) !==
        JSON.stringify(draft)
      const saved =
        changed
          ? await saveStyleSettings(draft)
          : detail
      await startStylePreview(
        saved.project.version,
      )
    } catch (error) {
      setPreviewTaskActive(false)
      setAction('')
      sseRef.current?.close()
      sseRef.current = null
      setNotice(
        '❌ ' +
          errorMessage(
            error,
            '首格样张生成启动失败',
          ),
      )
    }
  }
  const handleGeneratePreview = async () => {
    if (!detail || action) {
      return
    }
    setAction('preview')
    setNotice(
      '⏳ 正在启动首格完整样张生成…',
    )
    try {
      await startStylePreview(
        detail.project.version,
      )
    } catch (error) {
      setPreviewTaskActive(false)
      setAction('')
      sseRef.current?.close()
      sseRef.current = null
      setNotice(
        '❌ ' +
          errorMessage(
            error,
            '首格样张生成启动失败',
          ),
      )
    }
  }
  const handleConfirmPreview = async (
    panel: CoursewareComicPanel,
  ) => {
    if (!detail || action) {
      return
    }
    setAction('confirm_preview')
    setNotice(
      '⏳ 正在确认首格完整样张…',
    )
    try {
      const result =
        await confirmCoursewareComicStylePreview(
          coursewareId,
          projectId,
          {
            expected_version:
              detail.project.version,
            preview_panel_id:
              panel.id,
          },
        )
      applyDetail(result)
      setSelectedStage('batch_generation')
      setNotice(
        '✅ 首格样张已确认，可以四路并发生成其余图片。',
      )
    } catch (error) {
      setNotice(
        '❌ ' +
          errorMessage(
            error,
            '首格样张确认失败',
          ),
      )
    } finally {
      setAction('')
    }
  }
  const handleGenerateBatch = async () => {
    if (!detail || action) {
      return
    }
    setAction('batch')
    setNotice(
      '⏳ 正在启动最多四路并发的漫画图片生成…',
    )
    startProgressStream()
    try {
      const result =
        await generateCoursewareComicProject(
          coursewareId,
          projectId,
          detail.project.version,
        )
      setNotice('⏳ ' + result.message)
      await refreshDetail()
    } catch (error) {
      setAction('')
      sseRef.current?.close()
      sseRef.current = null
      setNotice(
        '❌ ' +
          errorMessage(
            error,
            '整批图片生成启动失败',
          ),
      )
    }
  }
  const handleRegeneratePanel = async (
    panel: CoursewareComicPanel,
    regenerationInstruction: string,
  ) => {
    if (
      action ||
      regeneratingPanelID ||
      savingStoryboardPanelID
    ) {
      return
    }
    const normalizedInstruction =
      regenerationInstruction.trim()
    if (!normalizedInstruction) {
      setNotice(
        '⚠️ 请先填写本次要修改的画面要求，再重新生成。',
      )
      return
    }
    if (
      Array.from(normalizedInstruction).length >
      1200
    ) {
      setNotice(
        '⚠️ 画面微调要求不能超过1200字。',
      )
      return
    }
    setRegeneratingPanelID(panel.id)
    setNotice(
      `⏳ 正在按画面微调要求重新生成第${panel.panel_no}格…`,
    )
    startProgressStream()
    try {
      const result =
        await regenerateCoursewareComicPanel(
          coursewareId,
          projectId,
          panel.id,
          panel.version,
          normalizedInstruction,
        )
      setNotice('⏳ ' + result.message)
      await refreshDetail()
    } catch (error) {
      setRegeneratingPanelID('')
      setNotice(
        '❌ ' +
          errorMessage(
            error,
            '单格重新生成启动失败',
          ),
      )
    }
  }
  const handleSyncPanel = async (
    panel: CoursewareComicPanel,
  ) => {
    if (
      action ||
      syncingPanelID ||
      savingStoryboardPanelID
    ) {
      return
    }
    setSyncingPanelID(panel.id)
    setNotice(
      `⏳ 正在同步第${panel.panel_no}格到已插入页面…`,
    )
    try {
      const result =
        await syncCoursewareComicPanelPage(
          coursewareId,
          projectId,
          panel.id,
          panel.version,
        )
      await refreshDetail()
      await onPagesChanged?.(
        result.page_number,
      )
      setNotice(
        `✅ 第${panel.panel_no}格已同步到课件第${result.page_number}页。`,
      )
    } catch (error) {
      setNotice(
        '❌ ' +
          errorMessage(
            error,
            '漫画格同步失败',
          ),
      )
    } finally {
      setSyncingPanelID('')
    }
  }
  const handleInsert = async (
    insertAt: number,
  ) => {
    if (!detail || action) {
      return
    }
    const inserted =
      Boolean(
        detail.project.inserted_page_id,
      )
    setAction('insert')
    setNotice(
      inserted
        ? '⏳ 正在刷新已插入的完整漫画页面…'
        : '⏳ 正在合成漫画并插入课件…',
    )
    try {
      const result =
        await insertCoursewareComicPage(
          coursewareId,
          projectId,
          {
            expected_version:
              detail.project.version,
            insert_at:
              insertAt,
          },
        )
      await refreshDetail()
      await onPagesChanged?.(
        result.page_number,
      )
      setNotice(
        result.created
          ? `✅ 漫画已插入课件第${result.page_number}页。`
          : `✅ 课件第${result.page_number}页漫画已刷新。`,
      )
    } catch (error) {
      setNotice(
        '❌ ' +
          errorMessage(
            error,
            '漫画页面写入失败',
          ),
      )
    } finally {
      setAction('')
    }
  }
  const project = detail?.project
  const panels = detail?.panels || []
  const effectiveStage =
    project
      ? resolveCoursewareComicEffectiveStage(
          project,
        )
      : 'source'
  const stage =
    selectedStage || effectiveStage
  const panelGenerating =
    panels.some(
      panel =>
        panel.status === 'generating',
    )
  const globalBusy =
    Boolean(action) ||
    Boolean(savingStoryboardPanelID) ||
    Boolean(regeneratingPanelID) ||
    previewTaskActive ||
    panelGenerating ||
    project?.status === 'planning' ||
    project?.status === 'generating'
  return {
    detail,
    loading,
    notice,
    stage,
    effectiveStage,
    globalBusy,
    regeneratingPanelID,
    syncingPanelID,
    savingStoryboardPanelID,
    handleSelectStage,
    handlePlan,
    handleSaveStoryboardPanel,
    handleConfirmStoryboard,
    handleSaveStyle,
    handleSaveAndGeneratePreview,
    handleGeneratePreview,
    handleConfirmPreview,
    handleGenerateBatch,
    handleRegeneratePanel,
    handleSyncPanel,
    handleInsert,
    replacePanel,
  }
}

function generationPrefix(stage: string): string {
  if (stage.includes('failed')) {
    return '❌ '
  }
  if (stage.includes('done')) {
    return '✅ '
  }
  if (stage.includes('warning')) {
    return '⚠️ '
  }
  return '⏳ '
}

function errorMessage(
  error: unknown,
  fallback: string,
): string {
  return error instanceof Error
    ? error.message
    : fallback
}
