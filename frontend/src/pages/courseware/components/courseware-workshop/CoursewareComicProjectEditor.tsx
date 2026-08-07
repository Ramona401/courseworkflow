/**
 * CoursewareComicProjectEditor.tsx
 *
 * 知识点漫画项目详情与生产编排：
 *   - 恢复角色设定、分镜方案和当前安全图片URL；
 *   - 订阅comic_generation事件；
 *   - 启动人物设定图和4至8格图片顺序生成；
 *   - 支持失败任务继续生成；
 *   - 支持单格重新生成；
 *   - 支持确定性HTML首次插页和完整页面刷新；
 *   - 支持按稳定漫画格标记替换已插页中的单格；
 *   - SSE断线后重新读取服务器状态；
 *   - 生成期间使用10秒轮询兜底；
 *   - 重新规划开始时清除浏览器缓存的旧错误；
 *   - 重新规划失败后立即读取服务器最新版本和精确last_error。
 */

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'

import {
  generateCoursewareComicProject,
  getCoursewareComicProject,
  insertCoursewareComicPage,
  planCoursewareComicProject,
  regenerateCoursewareComicPanel,
  subscribeCoursewareComicGeneration,
  syncCoursewareComicPanelPage,
} from '@/api/coursewares'

import type {
  CoursewareComicGenerationEvent,
  CoursewareComicPanel,
  CoursewareComicProject,
  CoursewareComicProjectDetail,
  CoursewareComicProjectStatus,
} from '@/api/coursewares'

import {
  useAuth,
} from '@/store/auth'

import {
  useProtectedDraft,
} from '@/hooks/useProtectedDraft'

import CoursewareComicPanelEditor from './CoursewareComicPanelEditor'

interface CoursewareComicProjectEditorProps {
  coursewareId: string
  projectId: string
  pageCount: number

  onBack: () => void

  onProjectChanged?: (
    project: CoursewareComicProject,
  ) => void

  onPagesChanged?: (
    pageNumber: number,
  ) => void | Promise<void>
}

type ProjectAction =
  | ''
  | 'plan'
  | 'generate'
  | 'insert'

const C = {
  primary: '#7C3AED',
  primaryBackground:
    'rgba(124,58,237,0.07)',
  success: '#059669',
  warning: '#D97706',
  danger: '#DC2626',
  text: '#1F2937',
  textSecondary: '#64748B',
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  background: '#F8FAFC',
  white: '#FFFFFF',
}

const STATUS_CONFIG:
  Record<
    CoursewareComicProjectStatus,
    {
      label: string
      color: string
      background: string
    }
  > = {
    draft: {
      label: '待规划',
      color: '#64748B',
      background: '#F1F5F9',
    },
    planning: {
      label: 'AI规划中',
      color: '#7C3AED',
      background: '#F3E8FF',
    },
    planned: {
      label: '分镜已规划',
      color: '#2563EB',
      background: '#DBEAFE',
    },
    generating: {
      label: '图片生成中',
      color: '#D97706',
      background: '#FEF3C7',
    },
    ready: {
      label: '可插入课件',
      color: '#059669',
      background: '#D1FAE5',
    },
    inserted: {
      label: '已插入课件',
      color: '#047857',
      background: '#ECFDF5',
    },
    failed: {
      label: '需要重试',
      color: '#DC2626',
      background: '#FEE2E2',
    },
    archived: {
      label: '已归档',
      color: '#6B7280',
      background: '#F3F4F6',
    },
  }

export default function CoursewareComicProjectEditor({
  coursewareId,
  projectId,
  pageCount,
  onBack,
  onProjectChanged,
  onPagesChanged,
}: CoursewareComicProjectEditorProps) {
  const { user } =
    useAuth()

  const [
    detail,
    setDetail,
  ] = useState<
    CoursewareComicProjectDetail |
    null
  >(null)

  const [
    selectedPanelID,
    setSelectedPanelID,
  ] = useState('')

  const [
    loading,
    setLoading,
  ] = useState(true)

  const [
    action,
    setAction,
  ] = useState<ProjectAction>('')

  const [
    regeneratingPanelID,
    setRegeneratingPanelID,
  ] = useState('')

  const [
    syncingPanelID,
    setSyncingPanelID,
  ] = useState('')

  const [
    insertAt,
    setInsertAt,
  ] = useState(
    Math.max(
      1,
      pageCount + 1,
    ),
  )

  const [
    notice,
    setNotice,
  ] = useState('')

  const sseRef =
    useRef<{
      close: () => void
    } | null>(null)

  const retryPlanDraft =
    useProtectedDraft({
      userId:
        user?.id,
      scope:
        'courseware-comic-project',
      resourceId:
        projectId,
      field:
        'retry-plan-instruction',
      initialValue: '',
      maxHistory: 30,
    })

  const refreshDetail =
    useCallback(async () => {
      const result =
        await getCoursewareComicProject(
          coursewareId,
          projectId,
        )

      setDetail(result)

      setSelectedPanelID(
        previous =>
          result.panels.some(
            panel =>
              panel.id ===
              previous,
          )
            ? previous
            : result.panels[0]
                ?.id || '',
      )

      if (
        result.project.status !==
        'generating'
      ) {
        setAction(previous =>
          previous === 'generate'
            ? ''
            : previous,
        )

        setRegeneratingPanelID('')
      }

      onProjectChanged?.(
        result.project,
      )

      return result
    }, [
      coursewareId,
      projectId,
      onProjectChanged,
    ])

  useEffect(() => {
    let cancelled = false

    setLoading(true)

    refreshDetail()
      .catch(error => {
        if (!cancelled) {
          setNotice(
            '❌ ' +
              errorMessage(
                error,
                '漫画项目详情加载失败',
              ),
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
  }, [
    refreshDetail,
  ])

  useEffect(() => {
    setInsertAt(
      Math.max(
        1,
        pageCount + 1,
      ),
    )
  }, [
    pageCount,
  ])

  const handleGenerationEvent =
    useCallback(
      (
        event:
          CoursewareComicGenerationEvent,
      ) => {
        if (
          event.project_id !==
          projectId
        ) {
          return
        }

        if (event.message) {
          setNotice(
            generationNoticePrefix(
              event.stage,
            ) +
              event.message,
          )
        }

        if (
          event.stage ===
            'panel_done' ||
          event.stage ===
            'panel_failed'
        ) {
          setRegeneratingPanelID(
            previous =>
              previous ===
              event.panel_id
                ? ''
                : previous,
          )
        }

        if (
          shouldRefreshForEvent(
            event.stage,
          )
        ) {
          void refreshDetail()
        }

        if (
          event.stage ===
            'project_done' ||
          event.stage ===
            'project_failed'
        ) {
          setAction('')
          setRegeneratingPanelID('')

          sseRef.current?.close()
          sseRef.current = null
        }
      },
      [
        projectId,
        refreshDetail,
      ],
    )

  const startProgressStream =
    useCallback(() => {
      sseRef.current?.close()

      sseRef.current =
        subscribeCoursewareComicGeneration(
          coursewareId,
          {
            onConnected: () => {
              setNotice(
                '⏳ 已连接漫画生成进度。',
              )
            },

            onReconnected: () => {
              setNotice(
                '🔄 进度连接已恢复，正在同步项目状态…',
              )

              void refreshDetail()
            },

            onEvent:
              handleGenerationEvent,

            onTransportError:
              message => {
                setNotice(
                  '⚠️ ' +
                    message,
                )
              },
          },
        )
    }, [
      coursewareId,
      refreshDetail,
      handleGenerationEvent,
    ])

  useEffect(() => {
    if (
      detail?.project.status ===
        'generating' &&
      !sseRef.current
    ) {
      startProgressStream()
    }
  }, [
    detail?.project.status,
    startProgressStream,
  ])

  useEffect(() => {
    if (
      detail?.project.status !==
      'generating'
    ) {
      return
    }

    const timer =
      window.setInterval(
        () => {
          void refreshDetail()
        },
        10000,
      )

    return () => {
      window.clearInterval(
        timer,
      )
    }
  }, [
    detail?.project.status,
    refreshDetail,
  ])

  useEffect(() => {
    return () => {
      sseRef.current?.close()
      sseRef.current = null
    }
  }, [])

  const replacePanel = (
    updated:
      CoursewareComicPanel,
  ) => {
    setDetail(previous => {
      if (!previous) {
        return previous
      }

      return {
        ...previous,
        panels:
          previous.panels.map(
            panel =>
              panel.id ===
              updated.id
                ? updated
                : panel,
          ),
      }
    })
  }

  const handleRetryPlan =
    async () => {
      if (
        !detail ||
        action
      ) {
        return
      }

      const expectedVersion =
        detail.project.version

      setAction('plan')

      setDetail(previous => {
        if (!previous) {
          return previous
        }

        return {
          ...previous,
          project: {
            ...previous.project,
            last_error: '',
          },
        }
      })

      setNotice(
        '⏳ 正在重新生成角色、故事和分镜方案…',
      )

      try {
        const result =
          await planCoursewareComicProject(
            coursewareId,
            projectId,
            {
              expected_version:
                expectedVersion,

              teacher_instruction:
                retryPlanDraft.value
                  .trim(),
            },
          )

        setDetail(result)

        setSelectedPanelID(
          result.panels[0]
            ?.id || '',
        )

        retryPlanDraft.commit()

        onProjectChanged?.(
          result.project,
        )

        setNotice(
          `✅ 已完成${result.panels.length}格漫画分镜规划。`,
        )
      } catch (error) {
        let failureMessage =
          errorMessage(
            error,
            '漫画规划失败',
          )

        try {
          const latest =
            await refreshDetail()

          if (
            latest.project.version >
              expectedVersion &&
            latest.project.last_error
              .trim()
          ) {
            failureMessage =
              latest.project.last_error
                .trim()
          }
        } catch {
          // 详情刷新失败时保留本次HTTP错误，
          // 不使用重试前缓存的旧last_error覆盖新错误。
        }

        setNotice(
          '❌ ' +
            failureMessage,
        )
      } finally {
        setAction('')
      }
    }

  const handleGenerateProject =
    async () => {
      if (
        !detail ||
        action
      ) {
        return
      }

      setAction('generate')

      setNotice(
        '⏳ 正在启动人物设定图和漫画分格顺序生成…',
      )

      startProgressStream()

      try {
        const result =
          await generateCoursewareComicProject(
            coursewareId,
            projectId,
            detail.project.version,
          )

        setNotice(
          '⏳ ' +
            result.message,
        )

        await refreshDetail()
      } catch (error) {
        setAction('')

        sseRef.current?.close()
        sseRef.current = null

        setNotice(
          '❌ ' +
            errorMessage(
              error,
              '漫画图片生成启动失败',
            ),
        )
      }
    }

  const handleRegeneratePanel =
    async (
      panel:
        CoursewareComicPanel,
    ) => {
      if (
        regeneratingPanelID ||
        action
      ) {
        return
      }

      setRegeneratingPanelID(
        panel.id,
      )

      setNotice(
        `⏳ 正在启动第${panel.panel_no}格重新生成…`,
      )

      startProgressStream()

      try {
        const result =
          await regenerateCoursewareComicPanel(
            coursewareId,
            projectId,
            panel.id,
            panel.version,
          )

        setNotice(
          '⏳ ' +
            result.message,
        )

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

  const handleInsertPage =
    async () => {
      if (
        !detail ||
        action
      ) {
        return
      }

      const firstInsert =
        !detail.project
          .inserted_page_id

      const normalizedInsertAt =
        firstInsert
          ? clampInteger(
              insertAt,
              1,
              pageCount + 1,
            )
          : detail.project
              .inserted_page_number_snapshot ||
            1

      setAction('insert')

      setNotice(
        firstInsert
          ? '⏳ 正在生成确定性漫画HTML并插入课件…'
          : '⏳ 正在刷新已插入的完整漫画页面…',
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
                normalizedInsertAt,
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

  const handleSyncPanel =
    async (
      panel:
        CoursewareComicPanel,
    ) => {
      if (
        syncingPanelID ||
        action
      ) {
        return
      }

      setSyncingPanelID(
        panel.id,
      )

      setNotice(
        `⏳ 正在把第${panel.panel_no}格同步到已插入漫画页…`,
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
              '漫画格页面同步失败',
            ),
        )
      } finally {
        setSyncingPanelID('')
      }
    }

  if (loading) {
    return (
      <div style={emptyStyle}>
        正在加载漫画项目详情…
      </div>
    )
  }

  if (!detail) {
    return (
      <div>
        {notice && (
          <Notice text={notice} />
        )}

        <button
          type="button"
          onClick={onBack}
          style={secondaryButtonStyle}
        >
          ← 返回项目列表
        </button>
      </div>
    )
  }

  const project =
    detail.project

  const panels =
    detail.panels

  const selectedPanel =
    panels.find(
      panel =>
        panel.id ===
        selectedPanelID,
    ) ||
    panels[0] ||
    null

  const generatedCount =
    panels.filter(
      panel =>
        panel.status ===
          'generated' &&
        Boolean(
          panel.current_asset_id,
        ),
    ).length

  const allGenerated =
    panels.length > 0 &&
    generatedCount ===
      panels.length

  const status =
    STATUS_CONFIG[
      project.status
    ]

  const canPlan =
    panels.length === 0 &&
    (
      project.status ===
        'draft' ||
      project.status ===
        'failed'
    )

  const canGenerate =
    panels.length > 0 &&
    (
      project.status ===
        'planned' ||
      project.status ===
        'failed'
    )

  const canInsert =
    allGenerated &&
    (
      project.status ===
        'ready' ||
      project.status ===
        'inserted'
    )

  const globalBusy =
    Boolean(action) ||
    project.status ===
      'generating'

  const noticeErrorText =
    notice.startsWith('❌')
      ? notice.slice(1).trim()
      : ''

  return (
    <div>
      <ProjectHeader
        project={project}
        status={status}
        onBack={onBack}
      />

      {notice && (
        <Notice text={notice} />
      )}

      {project.last_error &&
        project.last_error.trim() !==
          noticeErrorText && (
        <div style={errorBoxStyle}>
          {project.last_error}
        </div>
      )}

      <ProjectSummary
        project={project}
        generatedCount={
          generatedCount
        }
      />

      <KnowledgeSummary
        project={project}
      />

      {canPlan && (
        <div style={sectionStyle}>
          <div style={sectionTitleStyle}>
            重新生成分镜方案
          </div>

          <div style={sectionDescriptionStyle}>
            原教材、单元、知识点、叙事模式和视觉风格已由服务器固化。
            此处只补充本次规划要求。
          </div>

          <textarea
            value={
              retryPlanDraft.value
            }
            onChange={event =>
              retryPlanDraft.setValue(
                event.target.value,
              )
            }
            onKeyDown={event => {
              retryPlanDraft
                .handleKeyDown(event)
            }}
            rows={4}
            placeholder="例如：保持两名学生角色；第3格安排常见误区；最后一格加入点击揭晓答案的选择题。"
            style={textareaStyle}
          />

          <div style={inlineActionsStyle}>
            <button
              type="button"
              onClick={
                retryPlanDraft.undo
              }
              disabled={
                Boolean(action) ||
                !retryPlanDraft.canUndo
              }
              style={secondaryButtonStyle}
            >
              ↶ 撤销
            </button>

            <button
              type="button"
              onClick={
                retryPlanDraft.redo
              }
              disabled={
                Boolean(action) ||
                !retryPlanDraft.canRedo
              }
              style={secondaryButtonStyle}
            >
              ↷ 重做
            </button>

            <button
              type="button"
              onClick={() => {
                void handleRetryPlan()
              }}
              disabled={
                Boolean(action)
              }
              style={primaryButtonStyle}
            >
              {action === 'plan'
                ? '⏳ 正在规划…'
                : '✨ 重新生成4至8格方案'}
            </button>
          </div>
        </div>
      )}

      {panels.length > 0 && (
        <GenerationSection
          project={project}
          panels={panels}
          generatedCount={
            generatedCount
          }
          canGenerate={
            canGenerate
          }
          action={action}
          onGenerate={() => {
            void handleGenerateProject()
          }}
        />
      )}

      {project.character_sheet_url && (
        <CharacterSheetSection
          project={project}
        />
      )}

      {panels.length > 0 && (
        <>
          <PanelSelector
            panels={panels}
            selectedPanelID={
              selectedPanel?.id || ''
            }
            onSelect={
              setSelectedPanelID
            }
          />

          {selectedPanel && (
            <CoursewareComicPanelEditor
              coursewareId={
                coursewareId
              }
              projectId={
                projectId
              }
              projectStatus={
                project.status
              }
              panel={
                selectedPanel
              }
              disabled={
                globalBusy
              }
              regenerating={
                regeneratingPanelID ===
                selectedPanel.id
              }
              syncing={
                syncingPanelID ===
                selectedPanel.id
              }
              onPanelUpdated={
                replacePanel
              }
              onRegenerate={
                panel => {
                  void handleRegeneratePanel(
                    panel,
                  )
                }
              }
              onSync={
                panel => {
                  void handleSyncPanel(
                    panel,
                  )
                }
              }
            />
          )}
        </>
      )}

      {canInsert && (
        <InsertSection
          project={project}
          pageCount={pageCount}
          insertAt={insertAt}
          action={action}
          onInsertAtChange={
            setInsertAt
          }
          onInsert={() => {
            void handleInsertPage()
          }}
        />
      )}
    </div>
  )
}

function ProjectHeader({
  project,
  status,
  onBack,
}: {
  project: CoursewareComicProject

  status: {
    label: string
    color: string
    background: string
  }

  onBack: () => void
}) {
  return (
    <div style={projectHeaderStyle}>
      <div>
        <button
          type="button"
          onClick={onBack}
          style={{
            ...secondaryButtonStyle,
            marginBottom: 8,
          }}
        >
          ← 返回项目列表
        </button>

        <div style={projectTitleStyle}>
          {project.title}
        </div>

        <div style={projectMetaStyle}>
          {project.publisher}
          {' · '}
          {project.semester}
          {' · '}
          {project.textbook_unit
            .unit_title}
        </div>
      </div>

      <span style={{
        padding: '4px 10px',
        borderRadius: 999,
        color: status.color,
        background:
          status.background,
        fontSize: 10,
        fontWeight: 900,
      }}>
        {status.label}
      </span>
    </div>
  )
}

function ProjectSummary({
  project,
  generatedCount,
}: {
  project: CoursewareComicProject
  generatedCount: number
}) {
  return (
    <div style={summaryGridStyle}>
      <SummaryItem
        label="分格进度"
        value={
          `${generatedCount}/${project.panel_count}格`
        }
      />

      <SummaryItem
        label="叙事模式"
        value={
          project.narrative_mode
        }
      />

      <SummaryItem
        label="视觉风格"
        value={
          project.visual_style
        }
      />

      <SummaryItem
        label="页面布局"
        value={
          project.layout_mode
        }
      />

      <SummaryItem
        label="人物数量"
        value={
          String(
            project
              .character_bible
              .characters
              .length,
          )
        }
      />

      <SummaryItem
        label="项目版本"
        value={
          String(
            project.version,
          )
        }
      />
    </div>
  )
}

function KnowledgeSummary({
  project,
}: {
  project: CoursewareComicProject
}) {
  return (
    <div style={sectionStyle}>
      <div style={sectionTitleStyle}>
        教材与知识点快照
      </div>

      <div style={tagContainerStyle}>
        {project.knowledge_points.map(
          item => (
            <span
              key={item.kp_code}
              title={
                item.academic_requirement ||
                item.content_requirement
              }
              style={tagStyle}
            >
              {item.kp_name}
            </span>
          ),
        )}
      </div>

      {project.teacher_focus && (
        <div style={teacherFocusStyle}>
          <strong>
            教师教学重点：
          </strong>
          {' '}
          {project.teacher_focus}
        </div>
      )}
    </div>
  )
}

function GenerationSection({
  project,
  panels,
  generatedCount,
  canGenerate,
  action,
  onGenerate,
}: {
  project: CoursewareComicProject
  panels: CoursewareComicPanel[]
  generatedCount: number
  canGenerate: boolean
  action: ProjectAction
  onGenerate: () => void
}) {
  const percent =
    panels.length > 0
      ? Math.round(
          generatedCount /
            panels.length *
            100,
        )
      : 0

  return (
    <div style={sectionStyle}>
      <div style={sectionHeaderStyle}>
        <div>
          <div style={sectionTitleStyle}>
            分镜与图片生产
          </div>

          <div style={sectionDescriptionStyle}>
            图片按格号严格串行生成；后一格通过IAOCI继承人物身份和连续性。
          </div>
        </div>

        {canGenerate && (
          <button
            type="button"
            onClick={onGenerate}
            disabled={
              Boolean(action)
            }
            style={primaryButtonStyle}
          >
            {action === 'generate'
              ? '⏳ 生成中…'
              : project.status ===
                    'failed'
                ? '▶️ 继续生成未完成格'
                : '🎨 生成人物设定图与全部分格'}
          </button>
        )}
      </div>

      <div style={progressTrackStyle}>
        <div style={{
          ...progressValueStyle,
          width:
            `${percent}%`,
        }} />
      </div>

      <div style={progressTextStyle}>
        已生成
        {' '}
        {generatedCount}/
        {panels.length}
        {' '}
        格
      </div>
    </div>
  )
}

function CharacterSheetSection({
  project,
}: {
  project: CoursewareComicProject
}) {
  return (
    <div style={sectionStyle}>
      <div style={sectionTitleStyle}>
        项目人物设定参考图
      </div>

      <div style={sectionDescriptionStyle}>
        后续所有漫画格均以该人物设定图和角色圣经为身份连续性依据。
      </div>

      <img
        src={
          project.character_sheet_url
        }
        alt="知识点漫画人物设定参考图"
        draggable={false}
        style={characterSheetImageStyle}
      />
    </div>
  )
}

function PanelSelector({
  panels,
  selectedPanelID,
  onSelect,
}: {
  panels: CoursewareComicPanel[]
  selectedPanelID: string
  onSelect: (panelID: string) => void
}) {
  return (
    <div style={panelSelectorGridStyle}>
      {panels.map(panel => {
        const selected =
          panel.id ===
          selectedPanelID

        return (
          <button
            key={panel.id}
            type="button"
            onClick={() =>
              onSelect(panel.id)
            }
            style={{
              ...panelSelectorButtonStyle,
              border:
                `2px solid ${
                  selected
                    ? C.primary
                    : C.border
                }`,
            }}
          >
            <div style={panelThumbnailStyle}>
              {panel.current_asset_url ? (
                <img
                  src={
                    panel.current_asset_url
                  }
                  alt={`第${panel.panel_no}格`}
                  draggable={false}
                  style={thumbnailImageStyle}
                />
              ) : (
                <div style={thumbnailEmptyStyle}>
                  待生成
                </div>
              )}
            </div>

            <div style={panelThumbnailMetaStyle}>
              <div style={{
                color: C.text,
                fontSize: 10,
                fontWeight: 800,
              }}>
                第{panel.panel_no}格
              </div>

              <div style={panelThumbnailPurposeStyle}>
                {panel.story_purpose}
              </div>
            </div>
          </button>
        )
      })}
    </div>
  )
}

function InsertSection({
  project,
  pageCount,
  insertAt,
  action,
  onInsertAtChange,
  onInsert,
}: {
  project: CoursewareComicProject
  pageCount: number
  insertAt: number
  action: ProjectAction
  onInsertAtChange: (value: number) => void
  onInsert: () => void
}) {
  const inserted =
    Boolean(
      project.inserted_page_id,
    )

  return (
    <div style={{
      ...sectionStyle,
      marginTop: 12,
    }}>
      <div style={sectionTitleStyle}>
        {inserted
          ? '刷新课件漫画页面'
          : '插入课件页面'}
      </div>

      <div style={sectionDescriptionStyle}>
        系统使用稳定项目与漫画格标记生成确定性HTML。
        已插入项目再次执行时只刷新原页面，不会创建重复页。
      </div>

      {!inserted && (
        <label style={insertFieldStyle}>
          <span style={miniLabelStyle}>
            插入位置
          </span>

          <input
            type="number"
            min={1}
            max={pageCount + 1}
            value={insertAt}
            onChange={event =>
              onInsertAtChange(
                Number(
                  event.target.value,
                ),
              )
            }
            style={inputStyle}
          />

          <span style={insertHintStyle}>
            当前课件共
            {' '}
            {pageCount}
            {' '}
            页，可插入到第1至
            {pageCount + 1}
            页。
          </span>
        </label>
      )}

      {inserted && (
        <div style={insertedInfoStyle}>
          当前漫画位于课件第
          {
            project
              .inserted_page_number_snapshot
          }
          页。
        </div>
      )}

      <button
        type="button"
        onClick={onInsert}
        disabled={
          Boolean(action)
        }
        style={successButtonStyle}
      >
        {action === 'insert'
          ? '⏳ 正在写入页面…'
          : inserted
            ? '🔄 刷新完整漫画页面'
            : '➕ 插入知识点漫画页'}
      </button>
    </div>
  )
}

function SummaryItem({
  label,
  value,
}: {
  label: string
  value: string
}) {
  return (
    <div style={summaryItemStyle}>
      <div style={summaryLabelStyle}>
        {label}
      </div>

      <div style={summaryValueStyle}>
        {value}
      </div>
    </div>
  )
}

function Notice({
  text,
}: {
  text: string
}) {
  const error =
    text.startsWith('❌')

  const warning =
    text.startsWith('⚠️')

  return (
    <div style={{
      marginBottom: 10,
      padding: '9px 11px',
      borderRadius: 8,
      border:
        `1px solid ${
          error
            ? '#FECACA'
            : warning
              ? '#FDE68A'
              : '#A7F3D0'
        }`,
      background:
        error
          ? '#FEF2F2'
          : warning
            ? '#FFFBEB'
            : '#ECFDF5',
      color:
        error
          ? C.danger
          : warning
            ? '#92400E'
            : C.success,
      fontSize: 10,
      lineHeight: 1.6,
    }}>
      {text}
    </div>
  )
}

function generationNoticePrefix(
  stage: string,
): string {
  if (
    stage ===
      'project_failed' ||
    stage ===
      'panel_failed'
  ) {
    return '❌ '
  }

  if (
    stage ===
      'project_done' ||
    stage ===
      'panel_done' ||
    stage ===
      'character_sheet_done'
  ) {
    return '✅ '
  }

  if (
    stage ===
      'character_sheet_warning'
  ) {
    return '⚠️ '
  }

  return '⏳ '
}

function shouldRefreshForEvent(
  stage: string,
): boolean {
  return (
    stage ===
      'character_sheet_done' ||
    stage ===
      'character_sheet_warning' ||
    stage ===
      'panel_done' ||
    stage ===
      'panel_failed' ||
    stage ===
      'project_done' ||
    stage ===
      'project_failed'
  )
}

function clampInteger(
  value: number,
  minimum: number,
  maximum: number,
): number {
  if (!Number.isFinite(value)) {
    return minimum
  }

  return Math.min(
    maximum,
    Math.max(
      minimum,
      Math.trunc(value),
    ),
  )
}

function errorMessage(
  error: unknown,
  fallback: string,
): string {
  return error instanceof Error
    ? error.message
    : fallback
}

const projectHeaderStyle:
  React.CSSProperties = {
    display: 'flex',
    alignItems: 'flex-start',
    justifyContent:
      'space-between',
    gap: 12,
    marginBottom: 12,
  }

const projectTitleStyle:
  React.CSSProperties = {
    color: C.text,
    fontSize: 16,
    fontWeight: 900,
  }

const projectMetaStyle:
  React.CSSProperties = {
    marginTop: 4,
    color: C.textSecondary,
    fontSize: 10,
    lineHeight: 1.65,
  }

const summaryGridStyle:
  React.CSSProperties = {
    display: 'grid',
    gridTemplateColumns:
      'repeat(auto-fit,minmax(135px,1fr))',
    gap: 8,
    marginBottom: 10,
  }

const summaryItemStyle:
  React.CSSProperties = {
    padding: '9px 10px',
    borderRadius: 8,
    border:
      `1px solid ${C.border}`,
    background: C.background,
  }

const summaryLabelStyle:
  React.CSSProperties = {
    color: C.textMuted,
    fontSize: 8,
    fontWeight: 700,
  }

const summaryValueStyle:
  React.CSSProperties = {
    marginTop: 3,
    color: C.text,
    fontSize: 10,
    fontWeight: 800,
  }

const sectionStyle:
  React.CSSProperties = {
    marginBottom: 10,
    padding: 12,
    borderRadius: 10,
    border:
      `1px solid ${C.border}`,
    background: C.white,
  }

const sectionHeaderStyle:
  React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    justifyContent:
      'space-between',
    gap: 10,
    marginBottom: 9,
    flexWrap: 'wrap',
  }

const sectionTitleStyle:
  React.CSSProperties = {
    color: C.text,
    fontSize: 11,
    fontWeight: 900,
  }

const sectionDescriptionStyle:
  React.CSSProperties = {
    marginTop: 3,
    marginBottom: 8,
    color: C.textMuted,
    fontSize: 9,
    lineHeight: 1.6,
  }

const tagContainerStyle:
  React.CSSProperties = {
    display: 'flex',
    gap: 6,
    flexWrap: 'wrap',
    marginTop: 8,
  }

const tagStyle:
  React.CSSProperties = {
    padding: '3px 8px',
    borderRadius: 999,
    border:
      '1px solid rgba(124,58,237,0.25)',
    background:
      C.primaryBackground,
    color: C.primary,
    fontSize: 9,
    fontWeight: 700,
  }

const teacherFocusStyle:
  React.CSSProperties = {
    marginTop: 9,
    padding: '8px 9px',
    borderRadius: 8,
    background: C.background,
    color: C.textSecondary,
    fontSize: 9,
    lineHeight: 1.6,
  }

const progressTrackStyle:
  React.CSSProperties = {
    height: 7,
    borderRadius: 999,
    overflow: 'hidden',
    background: '#E2E8F0',
  }

const progressValueStyle:
  React.CSSProperties = {
    height: '100%',
    background:
      'linear-gradient(90deg,#7C3AED,#10B981)',
    transition:
      'width 300ms',
  }

const progressTextStyle:
  React.CSSProperties = {
    marginTop: 5,
    textAlign: 'right',
    color: C.textMuted,
    fontSize: 9,
  }

const characterSheetImageStyle:
  React.CSSProperties = {
    width: '100%',
    marginTop: 8,
    borderRadius: 9,
    border:
      `1px solid ${C.border}`,
    display: 'block',
  }

const panelSelectorGridStyle:
  React.CSSProperties = {
    display: 'grid',
    gridTemplateColumns:
      'repeat(auto-fit,minmax(125px,1fr))',
    gap: 8,
    marginBottom: 10,
  }

const panelSelectorButtonStyle:
  React.CSSProperties = {
    overflow: 'hidden',
    padding: 0,
    borderRadius: 9,
    background: C.white,
    cursor: 'pointer',
    textAlign: 'left',
  }

const panelThumbnailStyle:
  React.CSSProperties = {
    width: '100%',
    aspectRatio: '16 / 9',
    background: C.background,
  }

const thumbnailImageStyle:
  React.CSSProperties = {
    width: '100%',
    height: '100%',
    objectFit: 'cover',
    display: 'block',
  }

const thumbnailEmptyStyle:
  React.CSSProperties = {
    height: '100%',
    display: 'flex',
    alignItems: 'center',
    justifyContent:
      'center',
    color: C.textMuted,
    fontSize: 9,
  }

const panelThumbnailMetaStyle:
  React.CSSProperties = {
    padding: '7px 8px',
  }

const panelThumbnailPurposeStyle:
  React.CSSProperties = {
    marginTop: 2,
    color: C.textMuted,
    fontSize: 8,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  }

const inlineActionsStyle:
  React.CSSProperties = {
    display: 'flex',
    gap: 7,
    marginTop: 8,
    flexWrap: 'wrap',
  }

const inputStyle:
  React.CSSProperties = {
    width: '100%',
    boxSizing: 'border-box',
    padding: '8px 9px',
    borderRadius: 7,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    color: C.text,
    fontSize: 10,
    outline: 'none',
  }

const textareaStyle:
  React.CSSProperties = {
    ...inputStyle,
    resize: 'vertical',
    lineHeight: 1.6,
  }

const insertFieldStyle:
  React.CSSProperties = {
    display: 'block',
    maxWidth: 220,
    marginTop: 9,
    marginBottom: 9,
  }

const miniLabelStyle:
  React.CSSProperties = {
    display: 'block',
    marginBottom: 4,
    color: C.text,
    fontSize: 9,
    fontWeight: 800,
  }

const insertHintStyle:
  React.CSSProperties = {
    display: 'block',
    marginTop: 4,
    color: C.textMuted,
    fontSize: 8,
    lineHeight: 1.5,
  }

const insertedInfoStyle:
  React.CSSProperties = {
    marginTop: 8,
    marginBottom: 9,
    color: C.success,
    fontSize: 10,
    fontWeight: 800,
  }

const emptyStyle:
  React.CSSProperties = {
    padding: '24px 12px',
    borderRadius: 9,
    border:
      `1px dashed ${C.border}`,
    background: C.background,
    color: C.textMuted,
    textAlign: 'center',
    fontSize: 11,
  }

const errorBoxStyle:
  React.CSSProperties = {
    marginBottom: 10,
    padding: '8px 10px',
    borderRadius: 8,
    border:
      '1px solid #FECACA',
    background: '#FEF2F2',
    color: C.danger,
    fontSize: 10,
    lineHeight: 1.55,
  }

const secondaryButtonStyle:
  React.CSSProperties = {
    padding: '7px 11px',
    borderRadius: 7,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    color: C.textSecondary,
    fontSize: 10,
    fontWeight: 800,
    cursor: 'pointer',
  }

const primaryButtonStyle:
  React.CSSProperties = {
    padding: '8px 13px',
    borderRadius: 8,
    border: 'none',
    background:
      'linear-gradient(135deg,#7C3AED,#4F46E5)',
    color: '#FFFFFF',
    fontSize: 10,
    fontWeight: 900,
    cursor: 'pointer',
  }

const successButtonStyle:
  React.CSSProperties = {
    padding: '9px 14px',
    borderRadius: 8,
    border:
      '1px solid #10B981',
    background: '#ECFDF5',
    color: '#047857',
    fontSize: 10,
    fontWeight: 900,
    cursor: 'pointer',
  }
