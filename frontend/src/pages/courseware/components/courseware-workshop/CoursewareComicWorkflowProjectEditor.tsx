/**
 * CoursewareComicWorkflowProjectEditor.tsx
 *
 * 知识点漫画五步项目编辑器。
 *
 * 页面只负责步骤编排；浏览步骤与服务端生产步骤继续分离。
 * 教师回看已完成步骤不会倒写数据库，也不能跳过尚未完成的步骤。
 */

import { useEffect, useState } from 'react'

import type {
  CoursewareComicProject,
  CoursewareComicWorkflowProject,
  CoursewareComicWorkflowStage,
} from '@/api/coursewares'

import CoursewareComicBatchGenerationStep from './CoursewareComicBatchGenerationStep'
import CoursewareComicRefinementStep from './CoursewareComicRefinementStep'
import CoursewareComicStoryboardStep from './CoursewareComicStoryboardStep'
import CoursewareComicStylePreviewStep from './CoursewareComicStylePreviewStep'
import CoursewareComicStyleSettingsStep from './CoursewareComicStyleSettingsStep'
import CoursewareComicWorkflowStepper from './CoursewareComicWorkflowStepper'
import useCoursewareComicWorkflowProjectEditor from './useCoursewareComicWorkflowProjectEditor'

interface CoursewareComicWorkflowProjectEditorProps {
  coursewareId: string
  projectId: string
  pageCount: number
  onBack: () => void
  onProjectChanged?: (project: CoursewareComicProject) => void
  onPagesChanged?: (pageNumber: number) => void | Promise<void>
}

const C = {
  primary: '#7C3AED',
  danger: '#DC2626',
  text: '#172033',
  textSecondary: '#5B667A',
  textMuted: '#8A94A7',
  border: '#E4E8F0',
  background: '#F6F7FB',
  white: '#FFFFFF',
}

export default function CoursewareComicWorkflowProjectEditor({
  coursewareId,
  projectId,
  pageCount,
  onBack,
  onProjectChanged,
  onPagesChanged,
}: CoursewareComicWorkflowProjectEditorProps) {
  const editor = useCoursewareComicWorkflowProjectEditor({
    coursewareId,
    projectId,
    onProjectChanged,
    onPagesChanged,
  })

  if (editor.loading) {
    return <div style={emptyStyle}>正在加载漫画工作台…</div>
  }

  if (!editor.detail) {
    return (
      <div style={workspaceStyle}>
        {editor.notice && <Notice text={editor.notice} />}
        <button type="button" onClick={onBack} style={secondaryButtonStyle}>
          ← 返回项目列表
        </button>
      </div>
    )
  }

  const { project } = editor.detail
  const panels = editor.detail.panels
  const storyboardEditable =
    editor.effectiveStage === 'storyboard' &&
    project.status === 'planned' &&
    !project.workflow.storyboard_confirmed_at

  return (
    <div style={workspaceStyle}>
      <ProjectHeader project={project} selectedStage={editor.stage} onBack={onBack} />

      <CoursewareComicWorkflowStepper
        project={project}
        selectedStage={editor.stage}
        onSelect={editor.handleSelectStage}
      />

      {editor.notice && <Notice text={editor.notice} />}

      {project.last_error && (
        <div style={errorStyle}>
          <strong>需要处理：</strong> {project.last_error}
        </div>
      )}

      <KnowledgeSummary project={project} stage={editor.stage} />

      <main style={contentStyle}>
        {editor.stage === 'source' && (
          <SourceStep
            busy={editor.globalBusy}
            canPlan={editor.effectiveStage === 'source'}
            onPlan={() => void editor.handlePlan(project.narrative_mode, '')}
          />
        )}

        {editor.stage === 'storyboard' && (
          <CoursewareComicStoryboardStep
            project={project}
            panels={panels}
            busy={editor.globalBusy}
            editable={storyboardEditable}
            savingPanelID={editor.savingStoryboardPanelID}
            onSavePanel={editor.handleSaveStoryboardPanel}
            onConfirm={mode => void editor.handleConfirmStoryboard(mode)}
            onReplan={(mode, instruction) => {
              void editor.handlePlan(mode, instruction)
            }}
          />
        )}

        {editor.stage === 'style_preview' && (
          <div style={twoColumnStyle}>
            <CoursewareComicStyleSettingsStep
              project={project}
              panels={panels}
              busy={editor.globalBusy}
              onSave={draft => void editor.handleSaveStyle(draft)}
              onSaveAndGenerate={draft => {
                void editor.handleSaveAndGeneratePreview(draft)
              }}
            />

            <CoursewareComicStylePreviewStep
              project={project}
              panels={panels}
              busy={editor.globalBusy}
              onRegenerate={() => void editor.handleGeneratePreview()}
              onConfirm={panel => void editor.handleConfirmPreview(panel)}
            />
          </div>
        )}

        {editor.stage === 'batch_generation' && (
          <CoursewareComicBatchGenerationStep
            project={project}
            panels={panels}
            busy={editor.globalBusy}
            onGenerate={() => void editor.handleGenerateBatch()}
          />
        )}

        {editor.stage === 'refinement' && (
          <CoursewareComicRefinementStep
            coursewareId={coursewareId}
            project={project}
            panels={panels}
            pageCount={pageCount}
            busy={editor.globalBusy}
            regeneratingPanelID={editor.regeneratingPanelID}
            syncingPanelID={editor.syncingPanelID}
            onPanelUpdated={editor.replacePanel}
            onRegenerate={(panel, instruction) => {
              void editor.handleRegeneratePanel(panel, instruction)
            }}
            onSync={panel => void editor.handleSyncPanel(panel)}
            onInsert={insertAt => void editor.handleInsert(insertAt)}
          />
        )}
      </main>
    </div>
  )
}

function ProjectHeader({
  project,
  selectedStage,
  onBack,
}: {
  project: CoursewareComicWorkflowProject
  selectedStage: CoursewareComicWorkflowStage
  onBack: () => void
}) {
  return (
    <header style={headerStyle}>
      <div style={headerIdentityStyle}>
        <button
          type="button"
          onClick={onBack}
          aria-label="返回漫画项目列表"
          style={backButtonStyle}
        >
          ←
        </button>

        <div style={headerTextStyle}>
          <div style={eyebrowStyle}>知识点漫画</div>
          <h1 style={projectTitleStyle}>{project.title}</h1>
          <div style={projectMetaStyle}>
            {project.subject} · {project.grade} · {project.panel_count}格
          </div>
        </div>
      </div>

      <div style={headerStatusStyle}>
        <span style={stageBadgeStyle}>{stageLabel(selectedStage)}</span>
        <span style={statusBadgeStyle}>{projectStatusLabel(project.status)}</span>
      </div>
    </header>
  )
}

function KnowledgeSummary({
  project,
  stage,
}: {
  project: CoursewareComicWorkflowProject
  stage: CoursewareComicWorkflowStage
}) {
  const [expanded, setExpanded] = useState(stage === 'source')

  useEffect(() => {
    if (stage === 'source') {
      setExpanded(true)
    }
  }, [stage])

  const knowledge =
    project.knowledge_points.length > 0
      ? project.knowledge_points.map(item => item.kp_name).filter(Boolean).join('、')
      : project.knowledge_content

  return (
    <section style={knowledgeStyle}>
      <button
        type="button"
        onClick={() => setExpanded(previous => !previous)}
        aria-expanded={expanded}
        style={knowledgeSummaryButtonStyle}
      >
        <span style={knowledgeIconStyle}>知</span>
        <span style={knowledgeMainStyle}>
          <strong style={knowledgeLabelStyle}>本漫画讲什么</strong>
          <span style={knowledgeValueStyle}>{knowledge || '尚未填写知识点'}</span>
        </span>
        <span style={knowledgeToggleStyle}>
          {expanded ? '收起 ⌃' : '详情 ⌄'}
        </span>
      </button>

      {expanded && (
        <div style={knowledgeDetailStyle}>
          <span>{project.publisher || '未选择出版社'}</span>
          <span>·</span>
          <span>{project.semester || '未选择学期'}</span>
          <span>·</span>
          <span>{project.textbook_unit.unit_title || '教师自由输入'}</span>
        </div>
      )}
    </section>
  )
}

function SourceStep({
  busy,
  canPlan,
  onPlan,
}: {
  busy: boolean
  canPlan: boolean
  onPlan: () => void
}) {
  return (
    <section style={sourceCardStyle}>
      <div>
        <div style={sourceIconStyle}>✦</div>
        <h2 style={sourceTitleStyle}>从知识点生成故事分镜</h2>
        <p style={sourceDescriptionStyle}>
          AI先给出4至8格故事结构；之后可逐格修改，再进入样张和批量生图。
        </p>
      </div>

      <button
        type="button"
        onClick={onPlan}
        disabled={busy || !canPlan}
        style={{ ...primaryButtonStyle, opacity: busy || !canPlan ? 0.55 : 1 }}
      >
        {busy ? '生成中…' : canPlan ? '生成故事分镜 →' : '本步骤已完成'}
      </button>
    </section>
  )
}

function Notice({ text }: { text: string }) {
  const error = text.startsWith('❌')
  const warning = text.startsWith('⚠️')
  const loading = text.startsWith('⏳') || text.startsWith('🔄')

  return (
    <div
      role={error ? 'alert' : 'status'}
      style={{
        ...noticeStyle,
        borderColor: error
          ? '#FECACA'
          : warning
            ? '#FDE68A'
            : loading
              ? '#C4B5FD'
              : '#A7F3D0',
        background: error
          ? '#FEF2F2'
          : warning
            ? '#FFFBEB'
            : loading
              ? '#F5F3FF'
              : '#ECFDF5',
        color: error
          ? C.danger
          : warning
            ? '#92400E'
            : loading
              ? '#6D28D9'
              : '#047857',
      }}
    >
      {text}
    </div>
  )
}

function projectStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    draft: '待规划',
    planning: 'AI规划中',
    planned: '待确认',
    generating: '图片生成中',
    ready: '可精修',
    inserted: '已插入课件',
    failed: '需要重试',
    archived: '已归档',
  }
  return labels[status] || status
}

function stageLabel(stage: CoursewareComicWorkflowStage): string {
  const labels: Record<CoursewareComicWorkflowStage, string> = {
    source: '知识点',
    storyboard: '分镜设计',
    style_preview: '样张确认',
    batch_generation: '批量生图',
    refinement: '精修使用',
  }
  return labels[stage]
}

const workspaceStyle: React.CSSProperties = {
  width: '100%',
  maxWidth: 1480,
  margin: '0 auto',
  color: C.text,
}

const contentStyle: React.CSSProperties = {
  minWidth: 0,
}

const headerStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 18,
  marginBottom: 14,
  padding: '6px 2px',
  flexWrap: 'wrap',
}

const headerIdentityStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 12,
  minWidth: 0,
}

const backButtonStyle: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 40,
  height: 40,
  flexShrink: 0,
  borderRadius: 12,
  border: `1px solid ${C.border}`,
  background: C.white,
  color: C.textSecondary,
  fontSize: 20,
  cursor: 'pointer',
}

const headerTextStyle: React.CSSProperties = {
  minWidth: 0,
}

const eyebrowStyle: React.CSSProperties = {
  marginBottom: 2,
  color: C.primary,
  fontSize: 12,
  fontWeight: 800,
  letterSpacing: '0.04em',
}

const projectTitleStyle: React.CSSProperties = {
  margin: 0,
  overflow: 'hidden',
  color: C.text,
  fontSize: 22,
  fontWeight: 900,
  lineHeight: 1.25,
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
}

const projectMetaStyle: React.CSSProperties = {
  marginTop: 4,
  color: C.textSecondary,
  fontSize: 13,
}

const headerStatusStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  flexWrap: 'wrap',
}

const stageBadgeStyle: React.CSSProperties = {
  padding: '6px 10px',
  borderRadius: 999,
  background: '#F5F3FF',
  color: '#6D28D9',
  fontSize: 12,
  fontWeight: 800,
}

const statusBadgeStyle: React.CSSProperties = {
  padding: '6px 10px',
  borderRadius: 999,
  background: C.background,
  color: C.textSecondary,
  fontSize: 12,
  fontWeight: 800,
}

const noticeStyle: React.CSSProperties = {
  position: 'sticky',
  top: 8,
  zIndex: 5,
  maxWidth: 760,
  margin: '0 auto 12px',
  padding: '10px 14px',
  borderRadius: 12,
  border: '1px solid',
  boxShadow: '0 10px 30px rgba(15,23,42,0.08)',
  fontSize: 13,
  fontWeight: 700,
  lineHeight: 1.5,
}

const errorStyle: React.CSSProperties = {
  marginBottom: 12,
  padding: '10px 13px',
  borderRadius: 12,
  border: '1px solid #FECACA',
  background: '#FEF2F2',
  color: C.danger,
  fontSize: 13,
  lineHeight: 1.55,
}

const knowledgeStyle: React.CSSProperties = {
  marginBottom: 14,
  overflow: 'hidden',
  borderRadius: 14,
  border: `1px solid ${C.border}`,
  background: C.white,
}

const knowledgeSummaryButtonStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  width: '100%',
  gap: 10,
  padding: '10px 13px',
  border: 'none',
  background: 'transparent',
  color: C.text,
  textAlign: 'left',
  cursor: 'pointer',
}

const knowledgeIconStyle: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 30,
  height: 30,
  flexShrink: 0,
  borderRadius: 9,
  background: '#EEF2FF',
  color: '#4F46E5',
  fontSize: 13,
  fontWeight: 900,
}

const knowledgeMainStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'baseline',
  minWidth: 0,
  flex: 1,
  gap: 10,
}

const knowledgeLabelStyle: React.CSSProperties = {
  flexShrink: 0,
  fontSize: 13,
}

const knowledgeValueStyle: React.CSSProperties = {
  overflow: 'hidden',
  color: C.textSecondary,
  fontSize: 13,
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
}

const knowledgeToggleStyle: React.CSSProperties = {
  flexShrink: 0,
  color: C.textMuted,
  fontSize: 12,
  fontWeight: 700,
}

const knowledgeDetailStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  padding: '0 13px 11px 53px',
  color: C.textMuted,
  fontSize: 12,
  flexWrap: 'wrap',
}

const twoColumnStyle: React.CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(auto-fit,minmax(360px,1fr))',
  gap: 14,
  alignItems: 'start',
}

const sourceCardStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  minHeight: 220,
  gap: 24,
  padding: 26,
  borderRadius: 18,
  border: `1px solid ${C.border}`,
  background: 'linear-gradient(135deg,#FFFFFF 0%,#F5F3FF 100%)',
  flexWrap: 'wrap',
}

const sourceIconStyle: React.CSSProperties = {
  marginBottom: 10,
  color: C.primary,
  fontSize: 30,
}

const sourceTitleStyle: React.CSSProperties = {
  margin: 0,
  color: C.text,
  fontSize: 20,
  fontWeight: 900,
}

const sourceDescriptionStyle: React.CSSProperties = {
  maxWidth: 620,
  margin: '8px 0 0',
  color: C.textSecondary,
  fontSize: 14,
  lineHeight: 1.65,
}

const emptyStyle: React.CSSProperties = {
  padding: 32,
  borderRadius: 14,
  border: `1px dashed ${C.border}`,
  background: C.background,
  color: C.textMuted,
  textAlign: 'center',
  fontSize: 14,
}

const secondaryButtonStyle: React.CSSProperties = {
  minHeight: 40,
  padding: '9px 14px',
  borderRadius: 10,
  border: `1px solid ${C.border}`,
  background: C.white,
  color: C.textSecondary,
  fontSize: 13,
  fontWeight: 800,
  cursor: 'pointer',
}

const primaryButtonStyle: React.CSSProperties = {
  minHeight: 42,
  padding: '10px 17px',
  borderRadius: 11,
  border: 'none',
  background: 'linear-gradient(135deg,#7C3AED,#4F46E5)',
  color: '#FFFFFF',
  fontSize: 14,
  fontWeight: 900,
  cursor: 'pointer',
}
