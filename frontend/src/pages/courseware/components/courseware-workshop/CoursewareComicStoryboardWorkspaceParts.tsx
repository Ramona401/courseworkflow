/**
 * CoursewareComicStoryboardWorkspaceParts.tsx
 *
 * 第二步单任务工作台的可复用展示部件和纯校验函数。
 */

import type {
  CoursewareComicNarrativeMode,
  CoursewareComicPanel,
} from '@/api/coursewares'

import {
  createCoursewareComicStoryboardPanelDraft,
} from '@/api/coursewares.comic.storyboard'

import type {
  CoursewareComicStoryboardPanelDraft,
} from '@/api/coursewares.comic.storyboard'

import {
  COURSEWARE_COMIC_NARRATIVE_OPTIONS,
} from './coursewareComicWorkflow'

import {
  storyboardColors,
  storyboardStyles as S,
} from './CoursewareComicStoryboardWorkspaceStyles'

export const STORYBOARD_FIELD_LIMITS = {
  storyPurpose: 500,
  knowledgeClaim: 2000,
  sceneText: 4000,
  actionText: 3000,
  cameraText: 1000,
  knowledgePresentation: 3000,
} as const

const AI_QUICK_PROMPTS = [
  '减少说教感',
  '增加故事冲突',
  '加入生活案例',
  '强化知识递进',
  '结尾增加课堂追问',
  '语言更适合当前年级',
]

export function PanelNavigator({
  panels,
  selectedPanelID,
  dirtyPanelIDs,
  savingPanelID,
  onSelect,
}: {
  panels: CoursewareComicPanel[]
  selectedPanelID: string
  dirtyPanelIDs: Set<string>
  savingPanelID: string
  onSelect: (panel: CoursewareComicPanel) => void
}) {
  return (
    <aside style={S.navigator}>
      <div style={S.navigatorHeader}>
        <span style={S.navigatorTitle}>分镜</span>
        <span style={S.navigatorCount}>{panels.length}格</span>
      </div>

      <div style={S.navigatorList}>
        {panels.map(panel => {
          const active = panel.id === selectedPanelID
          const dirty = dirtyPanelIDs.has(panel.id)
          const saving = panel.id === savingPanelID

          return (
            <button
              key={panel.id}
              type="button"
              onClick={() => onSelect(panel)}
              aria-current={active ? 'true' : undefined}
              style={{
                ...S.navigatorButton,
                borderColor: active
                  ? storyboardColors.primary
                  : storyboardColors.border,
                background: active ? '#F5F3FF' : storyboardColors.white,
                cursor: 'pointer',
              }}
            >
              <span
                style={{
                  ...S.navigatorNumber,
                  background: active
                    ? storyboardColors.primary
                    : storyboardColors.background,
                  color: active
                    ? storyboardColors.white
                    : storyboardColors.textSecondary,
                }}
              >
                {panel.panel_no}
              </span>

              <span style={S.navigatorText}>
                <span style={S.navigatorPurpose}>
                  {panel.story_purpose || `第${panel.panel_no}格`}
                </span>
                <span style={S.navigatorStatus}>
                  {saving ? '保存中' : dirty ? '待保存' : panelStatusLabel(panel)}
                </span>
              </span>

              {dirty && <span aria-label="有未保存修改" style={S.dirtyDot} />}
            </button>
          )
        })}
      </div>
    </aside>
  )
}

export function EditorSection({
  title,
  hint,
  children,
}: {
  title: string
  hint: string
  children: React.ReactNode
}) {
  return (
    <section style={S.section}>
      <div style={S.sectionHeader}>
        <span style={S.sectionTitle}>{title}</span>
        <span style={S.sectionHint}>{hint}</span>
      </div>
      {children}
    </section>
  )
}

export function EditableField({
  label,
  value,
  maxLength,
  rows,
  compact = false,
  emphasis = false,
  disabled,
  onChange,
}: {
  label: string
  value: string
  maxLength: number
  rows: number
  compact?: boolean
  emphasis?: boolean
  disabled: boolean
  onChange: (value: string) => void
}) {
  return (
    <label style={S.field}>
      <span style={S.fieldLabelRow}>
        <span style={S.fieldLabel}>{label}</span>
        {Array.from(value).length > maxLength * 0.8 && (
          <span style={S.fieldCounter}>
            {Array.from(value).length}/{maxLength}
          </span>
        )}
      </span>

      <textarea
        value={value}
        rows={rows}
        maxLength={maxLength}
        disabled={disabled}
        onChange={event => onChange(event.target.value)}
        style={{
          ...S.fieldInput,
          ...(compact ? S.compactInput : {}),
          ...(emphasis ? S.knowledgeInput : {}),
          background: disabled
            ? storyboardColors.background
            : emphasis
              ? '#FCFAFF'
              : storyboardColors.white,
        }}
      />
    </label>
  )
}

export function DialogueSummary({ panel }: { panel: CoursewareComicPanel }) {
  const dialogues = panel.dialogues
    .map(dialogue => dialogue.content.trim())
    .filter(Boolean)

  return (
    <div style={S.section}>
      <div style={S.sectionHeader}>
        <span style={S.sectionTitle}>文字摘要</span>
        <span style={S.sectionHint}>在第5步编辑气泡与题目</span>
      </div>

      <div style={S.textSummary}>
        <span>旁白</span>
        <span style={S.textSummaryValue}>{panel.narration_text || '无旁白'}</span>
      </div>

      <div style={{ ...S.textSummary, marginTop: 7 }}>
        <span>对白</span>
        <span style={S.textSummaryValue}>
          {dialogues.length > 0
            ? `${dialogues.length}条 · ${dialogues[0]}`
            : '无对白'}
        </span>
      </div>
    </div>
  )
}

export function AIAssistantPanel({
  open,
  busy,
  canReplan,
  selectedMode,
  aiInstruction,
  narrativeChanged,
  onOpen,
  onClose,
  onModeChange,
  onInstructionChange,
  onQuickPrompt,
  onSubmit,
}: {
  open: boolean
  busy: boolean
  canReplan: boolean
  selectedMode: CoursewareComicNarrativeMode
  aiInstruction: string
  narrativeChanged: boolean
  onOpen: () => void
  onClose: () => void
  onModeChange: (mode: CoursewareComicNarrativeMode) => void
  onInstructionChange: (value: string) => void
  onQuickPrompt: (value: string) => void
  onSubmit: () => void
}) {
  const normalizedInstruction = aiInstruction.trim()

  if (!open) {
    return (
      <aside style={S.assistant}>
        <button type="button" onClick={onOpen} style={S.assistantCollapsed}>
          <div style={S.assistantCollapsedTitle}>✨ AI帮我优化</div>
          <div style={S.assistantCollapsedHint}>
            用快捷建议或一句话，让AI重做整套分镜。
          </div>
        </button>
      </aside>
    )
  }

  return (
    <aside style={S.assistant}>
      <div style={S.assistantHeader}>
        <span style={S.assistantTitle}>AI优化全部分镜</span>
        <button
          type="button"
          onClick={onClose}
          aria-label="收起AI助手"
          style={S.assistantClose}
        >
          ×
        </button>
      </div>

      <div style={S.assistantBody}>
        <label>
          <span style={S.quickTitle}>叙事方式</span>
          <select
            value={selectedMode}
            disabled={busy || !canReplan}
            onChange={event => {
              onModeChange(event.target.value as CoursewareComicNarrativeMode)
            }}
            style={{ ...S.select, marginTop: 6 }}
          >
            {COURSEWARE_COMIC_NARRATIVE_OPTIONS.map(option => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>

        <div>
          <div style={S.quickTitle}>快捷建议</div>
          <div style={{ ...S.quickGrid, marginTop: 7 }}>
            {AI_QUICK_PROMPTS.map(prompt => (
              <button
                key={prompt}
                type="button"
                onClick={() => onQuickPrompt(prompt)}
                disabled={busy || !canReplan}
                style={S.quickButton}
              >
                {prompt}
              </button>
            ))}
          </div>
        </div>

        <label>
          <span style={S.quickTitle}>具体要求</span>
          <textarea
            value={aiInstruction}
            maxLength={8000}
            disabled={busy || !canReplan}
            placeholder="补充需要调整的情节、年级语言或某一格内容…"
            onChange={event => onInstructionChange(event.target.value)}
            style={{ ...S.assistantInput, marginTop: 6 }}
          />
        </label>

        {Array.from(aiInstruction).length > 6400 && (
          <div style={S.assistantCounter}>
            {Array.from(aiInstruction).length}/8000
          </div>
        )}

        <button
          type="button"
          onClick={onSubmit}
          disabled={
            busy ||
            !canReplan ||
            (!narrativeChanged && !normalizedInstruction)
          }
          style={{
            ...S.assistantButton,
            opacity:
              busy ||
              !canReplan ||
              (!narrativeChanged && !normalizedInstruction)
                ? 0.55
                : 1,
          }}
        >
          {busy ? 'AI处理中…' : '✨ 重新优化全部分镜'}
        </button>
      </div>
    </aside>
  )
}

export function validateStoryboardDraft(
  draft: CoursewareComicStoryboardPanelDraft,
): string {
  if (!draft.storyPurpose.trim()) {
    return '故事职责不能为空。'
  }
  if (!draft.knowledgeClaim.trim()) {
    return '知识结论不能为空。'
  }
  if (!draft.sceneText.trim() && !draft.actionText.trim()) {
    return '场景和人物动作至少填写一项。'
  }

  for (
    const [key, limit] of Object.entries(STORYBOARD_FIELD_LIMITS) as Array<
      [keyof CoursewareComicStoryboardPanelDraft, number]
    >
  ) {
    if (Array.from(draft[key]).length > limit) {
      return '分镜字段长度超过服务端限制。'
    }
  }
  return ''
}

export function storyboardDraftMatchesPanel(
  panel: CoursewareComicPanel,
  draft: CoursewareComicStoryboardPanelDraft,
): boolean {
  const saved = createCoursewareComicStoryboardPanelDraft(panel)

  return (
    saved.storyPurpose === draft.storyPurpose &&
    saved.knowledgeClaim === draft.knowledgeClaim &&
    saved.sceneText === draft.sceneText &&
    saved.actionText === draft.actionText &&
    saved.cameraText === draft.cameraText &&
    saved.knowledgePresentation === draft.knowledgePresentation
  )
}

function panelStatusLabel(panel: CoursewareComicPanel): string {
  switch (panel.status) {
  case 'generated':
    return '图片已完成'
  case 'generating':
    return '图片生成中'
  case 'failed':
    return '图片失败'
  case 'stale':
    return '图片需更新'
  default:
    return '已保存'
  }
}
