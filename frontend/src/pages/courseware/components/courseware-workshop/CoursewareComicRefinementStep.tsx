/**
 * CoursewareComicRefinementStep.tsx
 *
 * 第五步采用单格画布工作台：
 *   - 胶片条只承担快速选格和异常提示；
 *   - 中间区域始终聚焦当前漫画格；
 *   - 导出与课件写入收进统一底栏；
 *   - 不改变自动保存、重画、同步和插页协议。
 */

import {
  useEffect,
  useState,
} from 'react'

import type {
  CoursewareComicPanel,
  CoursewareComicWorkflowProject,
} from '@/api/coursewares'

import {
  COURSEWARE_COMIC_INSERTION_OPTIONS,
} from './coursewareComicWorkflow'

import CoursewareComicPanelEditorAspectBridge from './CoursewareComicPanelEditorAspectBridge'

import {
  exportCoursewareComicAsJPG,
  exportCoursewareComicAsPDF,
} from './coursewareComicExport'

import {
  clampInteger,
  refinementColors,
  refinementStyles,
  thumbnailAspectRatio,
} from './CoursewareComicRefinementStepStyles'

interface CoursewareComicRefinementStepProps {
  coursewareId: string
  project: CoursewareComicWorkflowProject
  panels: CoursewareComicPanel[]
  pageCount: number
  busy: boolean
  regeneratingPanelID: string
  syncingPanelID: string
  onPanelUpdated: (
    panel: CoursewareComicPanel,
  ) => void
  onRegenerate: (
    panel: CoursewareComicPanel,
    regenerationInstruction: string,
  ) => void
  onSync: (
    panel: CoursewareComicPanel,
  ) => void
  onInsert: (
    insertAt: number,
  ) => void
}

type CoursewareComicExportFormat =
  | ''
  | 'jpg'
  | 'pdf'

export default function CoursewareComicRefinementStep({
  coursewareId,
  project,
  panels,
  pageCount,
  busy,
  regeneratingPanelID,
  syncingPanelID,
  onPanelUpdated,
  onRegenerate,
  onSync,
  onInsert,
}: CoursewareComicRefinementStepProps) {
  const [
    selectedPanelID,
    setSelectedPanelID,
  ] = useState(
    panels[0]?.id || '',
  )

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
    exporting,
    setExporting,
  ] = useState<CoursewareComicExportFormat>('')

  const [
    exportNotice,
    setExportNotice,
  ] = useState('')

  useEffect(() => {
    setSelectedPanelID(previous =>
      panels.some(
        panel => panel.id === previous,
      )
        ? previous
        : panels[0]?.id || '',
    )
  }, [
    panels,
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

  const selectedPanel =
    panels.find(
      panel =>
        panel.id === selectedPanelID,
    ) ||
    panels[0] ||
    null

  const selectedIndex =
    selectedPanel
      ? panels.findIndex(
          panel =>
            panel.id === selectedPanel.id,
        )
      : -1

  const allGenerated =
    panels.length > 0 &&
    panels.every(
      panel =>
        panel.status === 'generated' &&
        Boolean(
          panel.current_asset_id,
        ),
    )

  const inserted =
    Boolean(
      project.inserted_page_id,
    )

  const insertionOption =
    COURSEWARE_COMIC_INSERTION_OPTIONS.find(
      option =>
        option.value ===
        project.workflow.insertion_mode,
    )

  const singlePageSupported =
    project.workflow.insertion_mode ===
    'single_page'

  const canInsert =
    allGenerated &&
    (
      project.status === 'ready' ||
      project.status === 'inserted'
    ) &&
    singlePageSupported

  const canExport =
    allGenerated &&
    !busy &&
    !exporting

  const styles =
    refinementStyles

  const selectAdjacentPanel = (
    offset: number,
  ) => {
    if (
      selectedIndex < 0 ||
      panels.length === 0
    ) {
      return
    }

    const nextIndex =
      Math.min(
        panels.length - 1,
        Math.max(
          0,
          selectedIndex + offset,
        ),
      )

    const next =
      panels[nextIndex]

    if (next) {
      setSelectedPanelID(
        next.id,
      )
    }
  }

  const handleExport = async (
    format:
      Exclude<
        CoursewareComicExportFormat,
        ''
      >,
  ) => {
    if (!canExport) {
      setExportNotice(
        '⚠️ 请等待全部漫画图片生成完成后再导出。',
      )
      return
    }

    setExporting(
      format,
    )

    setExportNotice(
      format === 'jpg'
        ? '⏳ 正在生成完整漫画JPG…'
        : '⏳ 正在准备PDF排版窗口…',
    )

    try {
      if (format === 'jpg') {
        await exportCoursewareComicAsJPG(
          project,
          panels,
        )

        setExportNotice(
          '✅ 完整漫画JPG已经开始下载。',
        )
      } else {
        await exportCoursewareComicAsPDF(
          project,
          panels,
        )

        setExportNotice(
          '✅ PDF排版窗口已打开，请选择“另存为PDF”。',
        )
      }
    } catch (error) {
      setExportNotice(
        `❌ ${
          error instanceof Error &&
          error.message.trim()
            ? error.message
            : '漫画导出失败'
        }`,
      )
    } finally {
      setExporting('')
    }
  }

  return (
    <section>
      <div style={styles.compactHeader}>
        <div>
          <div style={styles.title}>
            精修与使用
          </div>

          <div style={styles.description}>
            选择一格，在画布中直接调整文字、气泡和排版。
          </div>
        </div>

        <span
          style={{
            ...styles.statusBadge,
            color:
              allGenerated
                ? refinementColors.success
                : '#B45309',
            background:
              allGenerated
                ? '#ECFDF5'
                : '#FFFBEB',
          }}
        >
          {allGenerated
            ? `${panels.length}格已就绪`
            : '仍有图片待处理'}
        </span>
      </div>

      <div style={styles.panelNavigator}>
        <button
          type="button"
          onClick={() =>
            selectAdjacentPanel(-1)
          }
          disabled={
            selectedIndex <= 0
          }
          style={styles.panelNavButton}
        >
          ← 上一格
        </button>

        <div style={styles.panelCounter}>
          第{
            selectedPanel?.panel_no ||
            0
          }格
          <span style={styles.panelCounterMuted}>
            {' / '}
            {panels.length}
          </span>
        </div>

        <button
          type="button"
          onClick={() =>
            selectAdjacentPanel(1)
          }
          disabled={
            selectedIndex < 0 ||
            selectedIndex >=
              panels.length - 1
          }
          style={styles.panelNavButton}
        >
          下一格 →
        </button>
      </div>

      <div style={styles.filmstrip}>
        {panels.map(panel => {
          const selected =
            panel.id ===
            selectedPanel?.id

          const problem =
            panel.status === 'failed' ||
            panel.status === 'stale'

          return (
            <button
              key={panel.id}
              type="button"
              onClick={() =>
                setSelectedPanelID(
                  panel.id,
                )
              }
              title={
                panel.story_purpose ||
                `第${panel.panel_no}格`
              }
              style={{
                ...styles.filmButton,
                borderColor:
                  selected
                    ? refinementColors.primary
                    : refinementColors.border,
                boxShadow:
                  selected
                    ? '0 0 0 3px rgba(124,58,237,0.14)'
                    : 'none',
              }}
            >
              <div
                style={{
                  ...styles.thumbnail,
                  aspectRatio:
                    thumbnailAspectRatio(
                      project.workflow
                        .aspect_ratio,
                    ),
                }}
              >
                {panel.current_asset_url ? (
                  <img
                    src={
                      panel.current_asset_url
                    }
                    alt={
                      `第${panel.panel_no}格`
                    }
                    draggable={false}
                    style={styles.image}
                  />
                ) : (
                  <div style={styles.empty}>
                    {panel.status ===
                    'generating'
                      ? '生成中'
                      : '待生成'}
                  </div>
                )}

                <span style={styles.numberBadge}>
                  {panel.panel_no}
                </span>

                {problem && (
                  <span style={styles.problemBadge}>
                    !
                  </span>
                )}
              </div>
            </button>
          )
        })}
      </div>

      {selectedPanel && (
        <CoursewareComicPanelEditorAspectBridge
          aspectRatio={
            project.workflow
              .aspect_ratio
          }
          coursewareId={
            coursewareId
          }
          projectId={
            project.id
          }
          projectStatus={
            project.status
          }
          panel={
            selectedPanel
          }
          disabled={
            busy
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
            onPanelUpdated
          }
          onRegenerate={
            onRegenerate
          }
          onSync={
            onSync
          }
        />
      )}

      {exportNotice && (
        <div
          style={{
            ...styles.notice,
            borderColor:
              exportNotice.startsWith(
                '❌',
              )
                ? '#FECACA'
                : exportNotice.startsWith(
                      '⚠️',
                    )
                  ? '#FDE68A'
                  : '#A7F3D0',
            background:
              exportNotice.startsWith(
                '❌',
              )
                ? '#FEF2F2'
                : exportNotice.startsWith(
                      '⚠️',
                    )
                  ? '#FFFBEB'
                  : '#ECFDF5',
            color:
              exportNotice.startsWith(
                '❌',
              )
                ? '#DC2626'
                : exportNotice.startsWith(
                      '⚠️',
                    )
                  ? '#92400E'
                  : '#047857',
          }}
        >
          {exportNotice}
        </div>
      )}

      <div style={styles.bottomBar}>
        <div style={styles.useText}>
          <strong>
            {insertionOption?.label ||
              project.workflow
                .insertion_mode}
          </strong>

          {inserted && (
            <span>
              {' · '}
              第{
                project
                  .inserted_page_number_snapshot
              }页
            </span>
          )}
        </div>

        <div style={styles.bottomActions}>
          {!singlePageSupported ? (
            <div style={styles.unsupported}>
              当前插入方式暂不支持写入课件
            </div>
          ) : (
            <>
              {!inserted && (
                <label style={styles.insertField}>
                  插入第
                  <input
                    type="number"
                    min={1}
                    max={
                      pageCount + 1
                    }
                    value={
                      insertAt
                    }
                    onChange={event =>
                      setInsertAt(
                        clampInteger(
                          Number(
                            event.target
                              .value,
                          ),
                          1,
                          pageCount + 1,
                        ),
                      )
                    }
                    disabled={
                      busy
                    }
                    style={styles.input}
                  />
                  页
                </label>
              )}

              <button
                type="button"
                onClick={() =>
                  onInsert(
                    inserted
                      ? project
                          .inserted_page_number_snapshot ||
                        1
                      : insertAt,
                  )
                }
                disabled={
                  busy ||
                  !canInsert
                }
                style={styles.successButton}
              >
                {busy
                  ? '处理中…'
                  : inserted
                    ? '刷新漫画页'
                    : '插入课件'}
              </button>
            </>
          )}

          <details style={styles.disclosure}>
            <summary style={styles.disclosureSummary}>
              导出
            </summary>

            <div style={styles.disclosureBody}>
              <button
                type="button"
                onClick={() =>
                  void handleExport(
                    'jpg',
                  )
                }
                disabled={
                  !canExport
                }
                style={styles.utilityButton}
              >
                {exporting === 'jpg'
                  ? '生成JPG中…'
                  : '下载JPG'}
              </button>

              <button
                type="button"
                onClick={() =>
                  void handleExport(
                    'pdf',
                  )
                }
                disabled={
                  !canExport
                }
                style={styles.utilityButton}
              >
                {exporting === 'pdf'
                  ? '准备PDF中…'
                  : '排版为PDF'}
              </button>
            </div>
          </details>
        </div>
      </div>
    </section>
  )
}
