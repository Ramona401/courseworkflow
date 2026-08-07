/**
 * CoursewareComicStoryboardStep.tsx
 *
 * 第二步单任务分镜工作台：
 * 左侧选格，中间一次只编辑一格，右侧AI助手按需展开。
 *
 * 浏览器仍只编辑教师可理解的教学字段。
 * 对白、旁白、题目、IAOCI和图片内部协议不在本步骤修改。
 */

import { useEffect, useMemo, useState } from 'react'

import type {
  CoursewareComicNarrativeMode,
  CoursewareComicPanel,
  CoursewareComicWorkflowProject,
} from '@/api/coursewares'

import {
  createCoursewareComicStoryboardPanelDraft,
} from '@/api/coursewares.comic.storyboard'

import type {
  CoursewareComicStoryboardPanelDraft,
} from '@/api/coursewares.comic.storyboard'

import {
  canConfirmCoursewareComicStoryboard,
  coursewareComicNarrativeLabel,
} from './coursewareComicWorkflow'

import {
  AIAssistantPanel,
  DialogueSummary,
  EditableField,
  EditorSection,
  PanelNavigator,
  STORYBOARD_FIELD_LIMITS,
  storyboardDraftMatchesPanel,
  validateStoryboardDraft,
} from './CoursewareComicStoryboardWorkspaceParts'

import {
  storyboardStyles as S,
} from './CoursewareComicStoryboardWorkspaceStyles'

interface CoursewareComicStoryboardStepProps {
  project: CoursewareComicWorkflowProject
  panels: CoursewareComicPanel[]
  busy: boolean
  savingPanelID?: string
  editable?: boolean
  onConfirm: (mode: CoursewareComicNarrativeMode) => void
  onReplan: (
    mode: CoursewareComicNarrativeMode,
    teacherInstruction: string,
  ) => void
  onSavePanel?: (
    panel: CoursewareComicPanel,
    draft: CoursewareComicStoryboardPanelDraft,
  ) => void | Promise<void>
}

interface StoryboardDraftState {
  version: number
  draft: CoursewareComicStoryboardPanelDraft
}

export default function CoursewareComicStoryboardStep({
  project,
  panels,
  busy,
  savingPanelID = '',
  editable,
  onConfirm,
  onReplan,
  onSavePanel,
}: CoursewareComicStoryboardStepProps) {
  const [selectedMode, setSelectedMode] =
    useState<CoursewareComicNarrativeMode>(project.narrative_mode)
  const [selectedPanelID, setSelectedPanelID] = useState(panels[0]?.id || '')
  const [aiInstruction, setAIInstruction] = useState('')
  const [assistantOpen, setAssistantOpen] = useState(false)
  const [draftStates, setDraftStates] =
    useState<Record<string, StoryboardDraftState>>({})

  useEffect(() => {
    setSelectedMode(project.narrative_mode)
    setAIInstruction('')
  }, [project.id, project.version, project.narrative_mode])

  useEffect(() => {
    setSelectedPanelID(previous =>
      panels.some(panel => panel.id === previous)
        ? previous
        : panels[0]?.id || '',
    )

    setDraftStates(previous => {
      const next: Record<string, StoryboardDraftState> = {}

      for (const panel of panels) {
        const existing = previous[panel.id]
        next[panel.id] =
          existing && existing.version === panel.version
            ? existing
            : {
                version: panel.version,
                draft: createCoursewareComicStoryboardPanelDraft(panel),
              }
      }
      return next
    })
  }, [panels])

  const canEdit =
    editable ??
    (project.status === 'planned' &&
      project.workflow.stage === 'storyboard' &&
      !project.workflow.storyboard_confirmed_at)
  const canReplan =
    canEdit &&
    project.status === 'planned' &&
    project.workflow.stage === 'storyboard'

  const selectedPanel =
    panels.find(panel => panel.id === selectedPanelID) || panels[0] || null
  const selectedIndex = selectedPanel
    ? panels.findIndex(panel => panel.id === selectedPanel.id)
    : -1
  const selectedDraft = selectedPanel
    ? draftStates[selectedPanel.id]?.draft ||
      createCoursewareComicStoryboardPanelDraft(selectedPanel)
    : null

  const dirtyPanelIDs = useMemo(
    () =>
      new Set(
        panels
          .filter(panel => {
            const draft =
              draftStates[panel.id]?.draft ||
              createCoursewareComicStoryboardPanelDraft(panel)
            return !storyboardDraftMatchesPanel(panel, draft)
          })
          .map(panel => panel.id),
      ),
    [panels, draftStates],
  )

  const narrativeChanged = selectedMode !== project.narrative_mode
  const normalizedInstruction = aiInstruction.trim()
  const selectedChanged = Boolean(selectedPanel && dirtyPanelIDs.has(selectedPanel.id))
  const allDraftsSaved = dirtyPanelIDs.size === 0
  const canConfirm =
    canConfirmCoursewareComicStoryboard(project) &&
    !narrativeChanged &&
    allDraftsSaved
  const selectedValidationError = selectedDraft
    ? validateStoryboardDraft(selectedDraft)
    : ''
  const saving = Boolean(selectedPanel && savingPanelID === selectedPanel.id)

  const updateSelectedDraft = <
    Key extends keyof CoursewareComicStoryboardPanelDraft,
  >(
    key: Key,
    value: CoursewareComicStoryboardPanelDraft[Key],
  ) => {
    if (!selectedPanel || !selectedDraft) {
      return
    }

    setDraftStates(previous => ({
      ...previous,
      [selectedPanel.id]: {
        version: previous[selectedPanel.id]?.version ?? selectedPanel.version,
        draft: { ...selectedDraft, [key]: value },
      },
    }))
  }

  const saveSelectedPanel = async (): Promise<boolean> => {
    if (
      !selectedPanel ||
      !selectedDraft ||
      !onSavePanel ||
      !canEdit ||
      busy ||
      saving ||
      !selectedChanged ||
      selectedValidationError
    ) {
      return false
    }

    await onSavePanel(selectedPanel, selectedDraft)
    return true
  }

  const selectAdjacent = (offset: number) => {
    const next = panels[selectedIndex + offset]
    if (next) {
      setSelectedPanelID(next.id)
    }
  }

  const handlePrimaryAction = async () => {
    if (!selectedPanel) {
      return
    }

    if (selectedChanged) {
      const saved = await saveSelectedPanel()
      if (saved && selectedIndex < panels.length - 1) {
        selectAdjacent(1)
      }
      return
    }

    if (selectedIndex < panels.length - 1) {
      selectAdjacent(1)
      return
    }

    if (canConfirm) {
      onConfirm(selectedMode)
    }
  }

  const requestAIReplan = () => {
    if (
      !canReplan ||
      busy ||
      (!narrativeChanged && !normalizedInstruction)
    ) {
      return
    }

    if (
      dirtyPanelIDs.size > 0 &&
      !window.confirm(
        'AI优化会替换当前全部分镜。尚未保存的单格修改将丢失，是否继续？',
      )
    ) {
      return
    }

    onReplan(selectedMode, normalizedInstruction)
  }

  const appendQuickPrompt = (prompt: string) => {
    setAIInstruction(previous => {
      const normalized = previous.trim()
      if (!normalized) {
        return prompt
      }
      if (normalized.includes(prompt)) {
        return previous
      }
      return `${normalized}；${prompt}`
    })
  }

  const primaryLabel = selectedChanged
    ? selectedIndex < panels.length - 1
      ? '保存并下一格 →'
      : '保存当前格'
    : selectedIndex < panels.length - 1
      ? '下一格 →'
      : project.workflow.storyboard_confirmed_at
        ? '当前分镜已确认'
        : '确认分镜并进入样张 →'

  return (
    <section style={S.container}>
      <div style={S.header}>
        <div>
          <h2 style={S.title}>分镜设计</h2>
          <div style={S.description}>
            一次只处理一格。先看画面与知识，高级信息按需展开。
          </div>
        </div>

        <div style={S.headerBadges}>
          <span style={S.badge}>{panels.length}格</span>
          {dirtyPanelIDs.size > 0 && (
            <span style={S.mutedBadge}>{dirtyPanelIDs.size}格待保存</span>
          )}
        </div>
      </div>

      {!canEdit && (
        <div style={S.reviewBar}>
          <span>👁</span>
          <span>只读回看。已确认的样张、图片和课件页面不会被改变。</span>
        </div>
      )}

      {narrativeChanged && (
        <div style={S.warningBar}>
          叙事方式已从“{coursewareComicNarrativeLabel(project.narrative_mode)}”
          切换为“{coursewareComicNarrativeLabel(selectedMode)}”，请先执行AI优化。
        </div>
      )}

      <div style={S.workspace}>
        <PanelNavigator
          panels={panels}
          selectedPanelID={selectedPanel?.id || ''}
          dirtyPanelIDs={dirtyPanelIDs}
          savingPanelID={savingPanelID}
          onSelect={panel => setSelectedPanelID(panel.id)}
        />

        {selectedPanel && selectedDraft ? (
          <div style={S.editor}>
            <div style={S.editorHeader}>
              <div style={S.editorTitleRow}>
                <span style={S.editorNumber}>{selectedPanel.panel_no}</span>
                <div>
                  <div style={S.editorTitle}>第{selectedPanel.panel_no}格</div>
                  <div style={S.editorStatus}>
                    {saving
                      ? '正在保存…'
                      : selectedChanged
                        ? '有未保存修改'
                        : '已与服务器同步'}
                  </div>
                </div>
              </div>

              <span style={S.mutedBadge}>
                {selectedIndex + 1}/{panels.length}
              </span>
            </div>

            <div style={S.editorBody}>
              <EditorSection title="画面" hint="这一格画什么">
                <div style={S.fieldGrid}>
                  <EditableField
                    label="场景"
                    value={selectedDraft.sceneText}
                    maxLength={STORYBOARD_FIELD_LIMITS.sceneText}
                    rows={4}
                    disabled={!canEdit || busy}
                    onChange={value => updateSelectedDraft('sceneText', value)}
                  />
                  <EditableField
                    label="人物动作"
                    value={selectedDraft.actionText}
                    maxLength={STORYBOARD_FIELD_LIMITS.actionText}
                    rows={4}
                    disabled={!canEdit || busy}
                    onChange={value => updateSelectedDraft('actionText', value)}
                  />
                </div>
              </EditorSection>

              <EditorSection title="知识" hint="这一格让学生理解什么">
                <div style={S.fieldGrid}>
                  <EditableField
                    label="知识结论 *"
                    value={selectedDraft.knowledgeClaim}
                    maxLength={STORYBOARD_FIELD_LIMITS.knowledgeClaim}
                    rows={4}
                    emphasis
                    disabled={!canEdit || busy}
                    onChange={value => updateSelectedDraft('knowledgeClaim', value)}
                  />
                  <EditableField
                    label="呈现方式"
                    value={selectedDraft.knowledgePresentation}
                    maxLength={STORYBOARD_FIELD_LIMITS.knowledgePresentation}
                    rows={4}
                    disabled={!canEdit || busy}
                    onChange={value => {
                      updateSelectedDraft('knowledgePresentation', value)
                    }}
                  />
                </div>
              </EditorSection>

              <details style={S.details}>
                <summary style={S.detailsSummary}>高级设置：故事职责与镜头</summary>
                <div style={S.detailsBody}>
                  <EditableField
                    label="故事职责 *"
                    value={selectedDraft.storyPurpose}
                    maxLength={STORYBOARD_FIELD_LIMITS.storyPurpose}
                    rows={3}
                    compact
                    disabled={!canEdit || busy}
                    onChange={value => updateSelectedDraft('storyPurpose', value)}
                  />
                  <EditableField
                    label="镜头"
                    value={selectedDraft.cameraText}
                    maxLength={STORYBOARD_FIELD_LIMITS.cameraText}
                    rows={3}
                    compact
                    disabled={!canEdit || busy}
                    onChange={value => updateSelectedDraft('cameraText', value)}
                  />
                </div>
              </details>

              <DialogueSummary panel={selectedPanel} />
              {selectedValidationError && (
                <div style={S.fieldError}>{selectedValidationError}</div>
              )}
            </div>

            <div style={S.actionBar}>
              <div style={S.actionGroup}>
                <button
                  type="button"
                  onClick={() => selectAdjacent(-1)}
                  disabled={selectedIndex <= 0 || busy}
                  style={{
                    ...S.secondaryButton,
                    opacity: selectedIndex <= 0 || busy ? 0.5 : 1,
                  }}
                >
                  ← 上一格
                </button>
                <span style={S.actionHint}>
                  {selectedChanged
                    ? '当前格尚未保存'
                    : allDraftsSaved
                      ? '全部修改已保存'
                      : `${dirtyPanelIDs.size}格待保存`}
                </span>
              </div>

              <button
                type="button"
                onClick={() => void handlePrimaryAction()}
                disabled={
                  busy ||
                  saving ||
                  Boolean(selectedValidationError) ||
                  (selectedIndex === panels.length - 1 &&
                    !selectedChanged &&
                    !canConfirm)
                }
                style={{
                  ...S.primaryButton,
                  opacity:
                    busy ||
                    saving ||
                    Boolean(selectedValidationError) ||
                    (selectedIndex === panels.length - 1 &&
                      !selectedChanged &&
                      !canConfirm)
                      ? 0.55
                      : 1,
                }}
              >
                {saving ? '保存中…' : primaryLabel}
              </button>
            </div>
          </div>
        ) : (
          <div style={S.editor}>
            <div style={S.editorBody}>暂无可编辑分镜。</div>
          </div>
        )}

        <AIAssistantPanel
          open={assistantOpen}
          busy={busy}
          canReplan={canReplan}
          selectedMode={selectedMode}
          aiInstruction={aiInstruction}
          narrativeChanged={narrativeChanged}
          onOpen={() => setAssistantOpen(true)}
          onClose={() => setAssistantOpen(false)}
          onModeChange={setSelectedMode}
          onInstructionChange={setAIInstruction}
          onQuickPrompt={appendQuickPrompt}
          onSubmit={requestAIReplan}
        />
      </div>
    </section>
  )
}
