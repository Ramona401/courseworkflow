/**
 * CoursewareComicStyleSettingsStep.tsx
 *
 * 第三步视觉设置采用分段式单任务界面：
 *   - 一次只处理画风来源、视觉风格或画面规格中的一项；
 *   - 顶部摘要持续展示当前选择，避免反复阅读全部选项；
 *   - 风格补充要求默认折叠，仅在教师需要精细控制时展开；
 *   - 底部操作栏集中保存与样张生成动作。
 */

import {
  useEffect,
  useMemo,
  useState,
} from 'react'

import type {
  CoursewareComicPanel,
  CoursewareComicWorkflowProject,
} from '@/api/coursewares'

import {
  COURSEWARE_COMIC_ASPECT_OPTIONS,
  COURSEWARE_COMIC_QUALITY_OPTIONS,
  COURSEWARE_COMIC_VISUAL_OPTIONS,
  canEditCoursewareComicStyle,
  coursewareComicStyleInstructionLength,
  coursewareComicVisualLabel,
  createCoursewareComicStyleSettingsDraft,
  validateCoursewareComicStyleSettings,
} from './coursewareComicWorkflow'

import type {
  CoursewareComicStyleSettingsDraft,
} from './coursewareComicWorkflow'

import CoursewareComicStyleInstructionField from './CoursewareComicStyleInstructionField'
import CoursewareComicStyleOptionSection from './CoursewareComicStyleOptionSection'
import CoursewareComicVisualSourceSelector from './CoursewareComicVisualSourceSelector'
import * as S from './CoursewareComicStyleSettingsWorkspaceStyles'

interface Props {
  project: CoursewareComicWorkflowProject
  panels: CoursewareComicPanel[]
  busy: boolean
  onSave: (
    draft: CoursewareComicStyleSettingsDraft,
  ) => void
  onSaveAndGenerate: (
    draft: CoursewareComicStyleSettingsDraft,
  ) => void
}

type SettingsSection =
  | 'source'
  | 'style'
  | 'format'

const C = {
  primary: '#7C3AED',
  success: '#059669',
  warning: '#D97706',
  text: '#1F2937',
  textSecondary: '#64748B',
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  background: '#F8FAFC',
  white: '#FFFFFF',
}

export default function CoursewareComicStyleSettingsStep({
  project,
  panels,
  busy,
  onSave,
  onSaveAndGenerate,
}: Props) {
  const serverDraft = useMemo(
    () =>
      createCoursewareComicStyleSettingsDraft(
        project,
      ),
    [
      project.id,
      project.version,
      project.visual_style,
      project.workflow.visual_style_source,
      project.workflow.aspect_ratio,
      project.workflow.image_quality,
      project.workflow.style_instruction,
    ],
  )

  const [draft, setDraft] =
    useState<CoursewareComicStyleSettingsDraft>(
      serverDraft,
    )

  const [
    activeSection,
    setActiveSection,
  ] = useState<SettingsSection>('source')

  const [
    instructionOpen,
    setInstructionOpen,
  ] = useState(
    Boolean(
      serverDraft.styleInstruction.trim(),
    ),
  )

  useEffect(() => {
    setDraft(serverDraft)
    setInstructionOpen(
      Boolean(
        serverDraft.styleInstruction.trim(),
      ),
    )
  }, [serverDraft])

  const editable =
    canEditCoursewareComicStyle(
      project,
      panels,
    )

  const validationError =
    validateCoursewareComicStyleSettings(
      draft,
    )

  const dirty =
    JSON.stringify(draft) !==
    JSON.stringify(serverDraft)

  const disabled =
    busy || !editable

  const selectedMode =
    draft.visualStyleSource ===
    'selected'

  const instructionLength =
    coursewareComicStyleInstructionLength(
      draft.styleInstruction,
    )

  const sourceLabel =
    selectedMode
      ? '本漫画画风'
      : '跟随课件'

  const styleLabel =
    selectedMode
      ? coursewareComicVisualLabel(
          draft.visualStyle,
        )
      : '课件风格锚点'

  const qualityLabel =
    draft.imageQuality === 'high'
      ? '高清'
      : '标准'

  return (
    <section style={S.containerStyle}>
      <header style={S.headerStyle}>
        <div>
          <div style={S.eyebrowStyle}>
            第3步
          </div>

          <h2 style={S.titleStyle}>
            先定视觉，再看一格样张
          </h2>
        </div>

        <div style={{
          ...S.syncBadgeStyle,
          color:
            dirty
              ? C.warning
              : C.success,
          background:
            dirty
              ? '#FFF7ED'
              : '#ECFDF5',
        }}>
          {dirty
            ? '有未保存设置'
            : '已保存'}
        </div>
      </header>

      <div style={S.summaryBarStyle}>
        <SummaryPill
          label="来源"
          value={sourceLabel}
        />
        <SummaryPill
          label="画风"
          value={styleLabel}
        />
        <SummaryPill
          label="画幅"
          value={draft.aspectRatio}
        />
        <SummaryPill
          label="清晰度"
          value={qualityLabel}
        />
      </div>

      <nav
        aria-label="视觉设置分段"
        style={S.sectionNavStyle}
      >
        <SectionButton
          number="1"
          label="画风来源"
          active={
            activeSection === 'source'
          }
          onClick={() =>
            setActiveSection('source')
          }
        />

        <SectionButton
          number="2"
          label="视觉风格"
          active={
            activeSection === 'style'
          }
          onClick={() =>
            setActiveSection('style')
          }
        />

        <SectionButton
          number="3"
          label="画面规格"
          active={
            activeSection === 'format'
          }
          onClick={() =>
            setActiveSection('format')
          }
        />
      </nav>

      <div style={S.contentStyle}>
        {activeSection === 'source' && (
          <div>
            <SectionHeading
              title="选择唯一画风来源"
              hint="二选一即可，后续不会混合两套画风。"
            />

            <CoursewareComicVisualSourceSelector
              value={
                draft.visualStyleSource
              }
              disabled={disabled}
              onChange={value =>
                setDraft(previous => ({
                  ...previous,
                  visualStyleSource:
                    value,
                }))
              }
            />

            {!selectedMode && (
              <div style={S.compactNoticeStyle}>
                系统将直接继承当前课件的风格锚点。
                没有有效锚点时，生成会停止并给出提示。
              </div>
            )}

            <div style={S.sectionFooterStyle}>
              <button
                type="button"
                onClick={() =>
                  setActiveSection(
                    selectedMode
                      ? 'style'
                      : 'format',
                  )
                }
                style={S.nextButtonStyle}
              >
                下一项 →
              </button>
            </div>
          </div>
        )}

        {activeSection === 'style' && (
          <div>
            <SectionHeading
              title={
                selectedMode
                  ? '选择本漫画的视觉风格'
                  : '视觉风格由课件决定'
              }
              hint={
                selectedMode
                  ? '只需选择最接近课堂表达的一种。'
                  : '当前无需重复选择漫画预设。'
              }
            />

            {selectedMode ? (
              <>
                <CoursewareComicStyleOptionSection
                  title=""
                  description=""
                  options={
                    COURSEWARE_COMIC_VISUAL_OPTIONS
                  }
                  selected={
                    draft.visualStyle
                  }
                  disabled={disabled}
                  onSelect={value =>
                    setDraft(previous => ({
                      ...previous,
                      visualStyle:
                        value,
                    }))
                  }
                />

                <button
                  type="button"
                  onClick={() =>
                    setInstructionOpen(
                      previous => !previous,
                    )
                  }
                  style={S.advancedToggleStyle}
                >
                  {instructionOpen
                    ? '收起补充要求'
                    : '＋ 添加风格补充要求'}
                </button>

                {instructionOpen && (
                  <div style={S.advancedPanelStyle}>
                    <CoursewareComicStyleInstructionField
                      value={
                        draft.styleInstruction
                      }
                      disabled={disabled}
                      length={
                        instructionLength
                      }
                      onChange={value =>
                        setDraft(previous => ({
                          ...previous,
                          styleInstruction:
                            value,
                        }))
                      }
                    />
                  </div>
                )}
              </>
            ) : (
              <div style={S.sourceSummaryStyle}>
                <div style={S.sourceIconStyle}>
                  ✓
                </div>

                <div>
                  <div style={S.sourceTitleStyle}>
                    跟随课件整体风格
                  </div>
                  <div style={S.sourceDetailStyle}>
                    色彩、线条、材质与光影语言由课件风格锚点统一控制。
                  </div>
                </div>
              </div>
            )}

            <div style={S.sectionFooterStyle}>
              <button
                type="button"
                onClick={() =>
                  setActiveSection('format')
                }
                style={S.nextButtonStyle}
              >
                下一项 →
              </button>
            </div>
          </div>
        )}

        {activeSection === 'format' && (
          <div>
            <SectionHeading
              title="确定画幅与清晰度"
              hint="样张和后续全部分格将保持一致。"
            />

            <div style={S.formatGridStyle}>
              <CoursewareComicStyleOptionSection
                title="图片比例"
                description=""
                options={
                  COURSEWARE_COMIC_ASPECT_OPTIONS
                }
                selected={
                  draft.aspectRatio
                }
                disabled={disabled}
                onSelect={value =>
                  setDraft(previous => ({
                    ...previous,
                    aspectRatio:
                      value,
                  }))
                }
              />

              <CoursewareComicStyleOptionSection
                title="清晰度"
                description=""
                options={
                  COURSEWARE_COMIC_QUALITY_OPTIONS
                }
                selected={
                  draft.imageQuality
                }
                disabled={disabled}
                onSelect={value =>
                  setDraft(previous => ({
                    ...previous,
                    imageQuality:
                      value,
                  }))
                }
              />
            </div>
          </div>
        )}
      </div>

      {!editable && (
        <div style={S.lockedStyle}>
          {panels.some(
            panel =>
              panel.panel_no === 1 &&
              panel.status ===
                'generating',
          )
            ? '首格样张生成中，设置暂时锁定。'
            : '当前步骤只读。'}
        </div>
      )}

      {validationError && (
        <div style={S.warningStyle}>
          {validationError}
        </div>
      )}

      <footer style={S.actionBarStyle}>
        <div style={S.actionSummaryStyle}>
          {sourceLabel}
          <span style={S.dotStyle}>·</span>
          {styleLabel}
          <span style={S.dotStyle}>·</span>
          {draft.aspectRatio}
          <span style={S.dotStyle}>·</span>
          {qualityLabel}
        </div>

        <div style={S.actionsStyle}>
          {dirty && (
            <button
              type="button"
              onClick={() =>
                setDraft(serverDraft)
              }
              disabled={disabled}
              style={S.textButtonStyle}
            >
              撤销修改
            </button>
          )}

          <button
            type="button"
            onClick={() =>
              onSave(draft)
            }
            disabled={
              disabled ||
              !dirty ||
              Boolean(validationError)
            }
            style={{
              ...S.secondaryButtonStyle,
              opacity:
                disabled ||
                !dirty ||
                Boolean(validationError)
                  ? 0.5
                  : 1,
            }}
          >
            保存
          </button>

          <button
            type="button"
            onClick={() =>
              onSaveAndGenerate(draft)
            }
            disabled={
              disabled ||
              Boolean(validationError)
            }
            style={{
              ...S.primaryButtonStyle,
              opacity:
                disabled ||
                Boolean(validationError)
                  ? 0.55
                  : 1,
            }}
          >
            {busy
              ? '处理中…'
              : '生成首格样张 →'}
          </button>
        </div>
      </footer>
    </section>
  )
}

function SummaryPill({
  label,
  value,
}: {
  label: string
  value: string
}) {
  return (
    <div style={S.summaryPillStyle}>
      <span style={S.summaryLabelStyle}>
        {label}
      </span>
      <strong style={S.summaryValueStyle}>
        {value}
      </strong>
    </div>
  )
}

function SectionButton({
  number,
  label,
  active,
  onClick,
}: {
  number: string
  label: string
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        ...S.sectionButtonStyle,
        borderColor:
          active
            ? C.primary
            : C.border,
        background:
          active
            ? 'rgba(124,58,237,0.08)'
            : C.white,
        color:
          active
            ? C.primary
            : C.textSecondary,
      }}
    >
      <span style={{
        ...S.sectionNumberStyle,
        background:
          active
            ? C.primary
            : C.background,
        color:
          active
            ? C.white
            : C.textMuted,
      }}>
        {number}
      </span>
      {label}
    </button>
  )
}

function SectionHeading({
  title,
  hint,
}: {
  title: string
  hint: string
}) {
  return (
    <div style={S.sectionHeadingStyle}>
      <h3 style={S.sectionTitleStyle}>
        {title}
      </h3>
      <div style={S.sectionHintStyle}>
        {hint}
      </div>
    </div>
  )
}

