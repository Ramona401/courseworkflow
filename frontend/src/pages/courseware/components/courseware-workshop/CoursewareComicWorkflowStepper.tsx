/**
 * CoursewareComicWorkflowStepper.tsx
 *
 * 紧凑的知识点漫画五步导航。
 *
 * 步骤条只承担定位和回看，不长期展示教学说明。
 * 详细说明通过title提示提供；未完成步骤继续禁止点击。
 */

import type {
  CoursewareComicWorkflowProject,
  CoursewareComicWorkflowStage,
} from '@/api/coursewares'

import {
  COURSEWARE_COMIC_WORKFLOW_STEPS,
  coursewareComicStageIndex,
  coursewareComicStepCompleted,
  resolveCoursewareComicEffectiveStage,
} from './coursewareComicWorkflow'

interface CoursewareComicWorkflowStepperProps {
  project: CoursewareComicWorkflowProject
  selectedStage?: CoursewareComicWorkflowStage
  onSelect?: (stage: CoursewareComicWorkflowStage) => void
}

const C = {
  primary: '#7C3AED',
  success: '#059669',
  text: '#172033',
  textMuted: '#8A94A7',
  border: '#E4E8F0',
  background: '#F6F7FB',
  white: '#FFFFFF',
}

export default function CoursewareComicWorkflowStepper({
  project,
  selectedStage,
  onSelect,
}: CoursewareComicWorkflowStepperProps) {
  const effectiveStage = resolveCoursewareComicEffectiveStage(project)
  const activeStage = selectedStage || effectiveStage
  const effectiveIndex = coursewareComicStageIndex(effectiveStage)
  const activeIndex = coursewareComicStageIndex(activeStage)

  return (
    <nav aria-label="知识点漫画制作步骤" style={containerStyle}>
      <div style={summaryStyle}>
        <span style={summaryStrongStyle}>第{activeIndex + 1}/5步</span>
        <span style={summaryMutedStyle}>
          {activeStage === effectiveStage ? '当前任务' : '回看模式'}
        </span>
      </div>

      <div style={stepsStyle}>
        {COURSEWARE_COMIC_WORKFLOW_STEPS.map((step, index) => {
          const active = step.stage === activeStage
          const current = step.stage === effectiveStage
          const completed = coursewareComicStepCompleted(project, step.stage)
          const selectable =
            Boolean(onSelect) &&
            coursewareComicStageIndex(step.stage) <= effectiveIndex

          return (
            <div key={step.stage} style={stepGroupStyle}>
              <button
                type="button"
                title={step.description}
                onClick={() => selectable && onSelect?.(step.stage)}
                disabled={!selectable}
                aria-current={current ? 'step' : undefined}
                aria-pressed={active}
                style={{
                  ...stepButtonStyle,
                  borderColor: active
                    ? C.primary
                    : completed
                      ? '#A7F3D0'
                      : C.border,
                  background: active
                    ? '#F5F3FF'
                    : completed
                      ? '#F0FDF4'
                      : C.white,
                  color: active
                    ? '#6D28D9'
                    : completed
                      ? '#047857'
                      : C.textMuted,
                  cursor: selectable ? 'pointer' : 'default',
                  opacity: selectable || current ? 1 : 0.62,
                }}
              >
                <span
                  style={{
                    ...numberStyle,
                    background: active
                      ? C.primary
                      : completed
                        ? C.success
                        : C.background,
                    color: active || completed ? C.white : C.textMuted,
                  }}
                >
                  {completed && !active ? '✓' : step.number}
                </span>

                <span style={labelStyle}>{step.label}</span>
                {current && <span aria-label="当前生产步骤" style={currentDotStyle} />}
              </button>

              {index < COURSEWARE_COMIC_WORKFLOW_STEPS.length - 1 && (
                <span
                  aria-hidden="true"
                  style={{
                    ...connectorStyle,
                    background: completed ? '#86EFAC' : C.border,
                  }}
                />
              )}
            </div>
          )
        })}
      </div>
    </nav>
  )
}

const containerStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 14,
  marginBottom: 14,
  padding: '9px 11px',
  overflowX: 'auto',
  borderRadius: 14,
  border: `1px solid ${C.border}`,
  background: C.white,
}

const summaryStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  minWidth: 72,
  paddingRight: 12,
  borderRight: `1px solid ${C.border}`,
}

const summaryStrongStyle: React.CSSProperties = {
  color: C.text,
  fontSize: 13,
  fontWeight: 900,
}

const summaryMutedStyle: React.CSSProperties = {
  marginTop: 2,
  color: C.textMuted,
  fontSize: 11,
}

const stepsStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  minWidth: 650,
  flex: 1,
}

const stepGroupStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  flex: 1,
}

const stepButtonStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  minWidth: 104,
  minHeight: 38,
  flex: 1,
  gap: 7,
  padding: '7px 10px',
  borderRadius: 10,
  border: '1px solid',
  fontFamily: 'inherit',
  textAlign: 'center',
}

const numberStyle: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 22,
  height: 22,
  flexShrink: 0,
  borderRadius: 999,
  fontSize: 11,
  fontWeight: 900,
}

const labelStyle: React.CSSProperties = {
  fontSize: 13,
  fontWeight: 800,
  whiteSpace: 'nowrap',
}

const currentDotStyle: React.CSSProperties = {
  width: 6,
  height: 6,
  flexShrink: 0,
  borderRadius: 999,
  background: C.primary,
  boxShadow: '0 0 0 3px rgba(124,58,237,0.12)',
}

const connectorStyle: React.CSSProperties = {
  width: 18,
  height: 2,
  flexShrink: 0,
  margin: '0 4px',
  borderRadius: 999,
}
