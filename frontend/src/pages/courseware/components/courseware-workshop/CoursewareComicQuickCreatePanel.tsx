/**
 * CoursewareComicQuickCreatePanel.tsx
 *
 * 知识点漫画紧凑创建首屏：
 *   - 老师输入核心知识点并明确选择4—8格；
 *   - 教材、课件、课程大纲、文档、图片和其它文字均为可选参考；
 *   - 标题、叙事、视觉风格和页面布局继续由后端自动补齐；
 *   - 创建项目后先绑定可选参考资料，再调用统一AI规划；
 *   - 首次规划失败时重新读取服务器最新项目状态和精确错误。
 */

import { useState } from 'react'

import {
  getCoursewareComicProject,
  planCoursewareComicProject,
} from '@/api/coursewares'

import type {
  CoursewareComicProject,
  CoursewareDetail,
} from '@/api/coursewares'

import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import { useAuth } from '@/store/auth'

import CoursewareComicReferencePicker from './CoursewareComicReferencePicker'

import {
  attachCoursewareComicPendingReferences,
  COURSEWARE_COMIC_QUICK_PANEL_COUNTS,
  coursewareComicQuickKnowledgeLength,
  createCoursewareComicFromKnowledge,
  resolveQuickCreateErrorMessage,
  validateCoursewareComicQuickKnowledge,
} from './coursewareComicQuickCreate'

import type {
  CoursewareComicPendingReference,
  CoursewareComicQuickPanelCount,
} from './coursewareComicQuickCreate'

interface CoursewareComicQuickCreatePanelProps {
  coursewareId: string
  courseware: CoursewareDetail
  onProjectCreated: (project: CoursewareComicProject) => void
  onNotice: (message: string) => void
}

const C = {
  primary: '#7C3AED',
  primaryBackground: 'rgba(124,58,237,0.07)',
  text: '#1F2937',
  textSecondary: '#64748B',
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  background: '#F8FAFC',
  white: '#FFFFFF',
}

export default function CoursewareComicQuickCreatePanel({
  coursewareId,
  courseware,
  onProjectCreated,
  onNotice,
}: CoursewareComicQuickCreatePanelProps) {
  const { user } = useAuth()
  const [busy, setBusy] = useState(false)
  const [panelCount, setPanelCount] =
    useState<CoursewareComicQuickPanelCount>(4)
  const [references, setReferences] =
    useState<CoursewareComicPendingReference[]>([])

  const protectedDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'courseware-comic-create',
    resourceId: coursewareId,
    field: 'quick-knowledge-text',
    initialValue: '',
    maxHistory: 50,
  })

  const knowledgeText = protectedDraft.value
  const length = coursewareComicQuickKnowledgeLength(knowledgeText)
  const validationError = validateCoursewareComicQuickKnowledge(knowledgeText)
  const canSubmit = !busy && !validationError

  const handleCreate = async () => {
    if (!canSubmit) {
      onNotice(`⚠️ ${validationError || '请输入要讲解的知识点。'}`)
      return
    }

    setBusy(true)
    onNotice(`⏳ 正在保存知识点并创建${panelCount}格漫画项目…`)

    try {
      const created = await createCoursewareComicFromKnowledge(
        coursewareId,
        knowledgeText,
        panelCount,
      )

      let referenceWarning = ''

      if (references.length > 0) {
        onNotice(
          `⏳ 漫画项目已创建，正在关联${references.length}项可选参考资料…`,
        )

        const result = await attachCoursewareComicPendingReferences(
          coursewareId,
          created.id,
          references,
        )

        if (result.errors.length > 0) {
          referenceWarning = `；${result.errors.length}项参考资料未能关联`
          onNotice(
            `⚠️ 已成功关联${result.attached}项资料，${result.errors.length}项失败。AI将使用已成功关联的资料继续规划。`,
          )
        }
      }

      onNotice(
        `⏳ 核心知识与参考资料已保存，AI正在自动规划${panelCount}格标题、角色、情节和分镜…`,
      )

      try {
        const detail = await planCoursewareComicProject(
          coursewareId,
          created.id,
          {
            expected_version: created.version,
            teacher_instruction: '',
          },
        )

        protectedDraft.setValue('')
        setReferences([])
        onNotice(
          `✅ AI已自动生成《${detail.project.title}》的${detail.panels.length}格漫画方案${referenceWarning}。`,
        )
        onProjectCreated(detail.project)
      } catch (planError) {
        let latestProject = created
        let failureMessage = resolveQuickCreateErrorMessage(
          planError,
          '可以进入项目后重新规划',
        )

        try {
          const latest = await getCoursewareComicProject(
            coursewareId,
            created.id,
          )
          latestProject = latest.project

          if (
            latestProject.version > created.version &&
            latestProject.last_error.trim()
          ) {
            failureMessage = latestProject.last_error.trim()
          }
        } catch {
          // 详情读取失败时仍保留项目创建响应；项目编辑器挂载后会再次读取服务器状态。
        }

        onNotice(
          `⚠️ 漫画项目和参考资料已经保存，但AI规划暂未完成：${failureMessage}`,
        )
        onProjectCreated(latestProject)
      }
    } catch (error) {
      onNotice(
        `❌ ${resolveQuickCreateErrorMessage(error, '知识点漫画创建失败')}`,
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <section style={containerStyle}>
      <div style={headerStyle}>
        <div>
          <div style={titleStyle}>✨ 输入知识点，AI自动生成漫画方案</div>
          <div style={descriptionStyle}>
            当前课件：{courseware.subject} · {courseware.grade}。核心知识必填，
            课本、课件、课程大纲、图片和文件均可选。
          </div>
        </div>
        <div style={autoBadgeStyle}>4—8格可选</div>
      </div>

      <label style={fieldStyle}>
        <span style={labelStyle}>要讲解的核心知识点</span>
        <textarea
          autoFocus
          value={knowledgeText}
          onChange={event => protectedDraft.setValue(event.target.value)}
          onKeyDown={event => protectedDraft.handleKeyDown(event)}
          disabled={busy}
          rows={4}
          placeholder="输入要讲清楚的概念、判断依据、常见误区、课堂情境或练习要求。"
          style={{
            ...textareaStyle,
            borderColor: length > 8000 ? '#DC2626' : C.border,
          }}
        />
        <div style={lengthStyle}>{length}/8000</div>
      </label>

      <div style={panelCountSectionStyle}>
        <div>
          <div style={panelCountTitleStyle}>漫画格数</div>
          <div style={panelCountDescriptionStyle}>
            选择本次故事需要的分镜数量。格数越多，情节和知识展开越充分。
          </div>
        </div>

        <div style={panelCountOptionsStyle}>
          {COURSEWARE_COMIC_QUICK_PANEL_COUNTS.map(count => {
            const selected = count === panelCount

            return (
              <button
                key={count}
                type="button"
                disabled={busy}
                onClick={() => setPanelCount(count)}
                style={{
                  ...panelCountButtonStyle,
                  borderColor: selected ? C.primary : C.border,
                  background: selected ? C.primaryBackground : C.white,
                  color: selected ? C.primary : C.textSecondary,
                  boxShadow: selected
                    ? '0 0 0 2px rgba(124,58,237,0.12)'
                    : 'none',
                  cursor: busy ? 'default' : 'pointer',
                }}
              >
                {count}格
              </button>
            )
          })}
        </div>
      </div>

      <CoursewareComicReferencePicker
        coursewareId={coursewareId}
        courseware={courseware}
        value={references}
        disabled={busy}
        onChange={setReferences}
      />

      <div style={automaticPlanStyle}>
        <strong>AI将自动完成：</strong>
        按你选择的{panelCount}格规划漫画标题、人物设定、故事情节、场景、
        知识呈现、视觉风格、页面布局、题目与答案。可选资料只作补充，
        不改变核心知识事实。
      </div>

      <div style={footerStyle}>
        <div style={draftStatusStyle}>核心知识输入已自动保存在本浏览器标签页</div>
        <div style={actionStyle}>
          <button
            type="button"
            onClick={protectedDraft.undo}
            disabled={busy || !protectedDraft.canUndo}
            style={secondaryButtonStyle}
          >
            ↶ 撤销
          </button>
          <button
            type="button"
            onClick={protectedDraft.redo}
            disabled={busy || !protectedDraft.canRedo}
            style={secondaryButtonStyle}
          >
            ↷ 重做
          </button>
          <button
            type="button"
            onClick={() => void handleCreate()}
            disabled={!canSubmit}
            style={{
              ...primaryButtonStyle,
              background: canSubmit
                ? 'linear-gradient(135deg,#7C3AED,#4F46E5)'
                : '#CBD5E1',
              cursor: canSubmit ? 'pointer' : 'default',
            }}
          >
            {busy
              ? `⏳ AI正在生成${panelCount}格方案…`
              : `✨ AI生成${panelCount}格知识点漫画`}
          </button>
        </div>
      </div>
    </section>
  )
}

const containerStyle: React.CSSProperties = {
  marginBottom: 12,
  padding: 14,
  borderRadius: 12,
  border: '1px solid rgba(124,58,237,0.22)',
  background: 'linear-gradient(145deg,#FFFFFF,#F8F7FF)',
  boxShadow: '0 8px 28px rgba(76,29,149,0.06)',
}

const headerStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'flex-start',
  justifyContent: 'space-between',
  gap: 12,
  marginBottom: 10,
}

const titleStyle: React.CSSProperties = {
  color: C.text,
  fontSize: 15,
  fontWeight: 900,
}

const descriptionStyle: React.CSSProperties = {
  marginTop: 4,
  color: C.textSecondary,
  fontSize: 10,
  lineHeight: 1.6,
}

const autoBadgeStyle: React.CSSProperties = {
  flexShrink: 0,
  padding: '4px 9px',
  borderRadius: 999,
  background: C.primaryBackground,
  color: C.primary,
  fontSize: 10,
  fontWeight: 800,
}

const fieldStyle: React.CSSProperties = { display: 'block' }

const labelStyle: React.CSSProperties = {
  display: 'block',
  marginBottom: 5,
  color: C.text,
  fontSize: 11,
  fontWeight: 800,
}

const textareaStyle: React.CSSProperties = {
  width: '100%',
  boxSizing: 'border-box',
  padding: '9px 11px',
  borderRadius: 9,
  border: `1px solid ${C.border}`,
  background: C.white,
  color: C.text,
  fontSize: 11,
  lineHeight: 1.6,
  resize: 'vertical',
  outline: 'none',
}

const lengthStyle: React.CSSProperties = {
  marginTop: 3,
  textAlign: 'right',
  color: C.textMuted,
  fontSize: 9,
}

const panelCountSectionStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 12,
  marginTop: 8,
  marginBottom: 10,
  padding: '9px 10px',
  borderRadius: 9,
  border: `1px solid ${C.border}`,
  background: C.white,
  flexWrap: 'wrap',
}

const panelCountTitleStyle: React.CSSProperties = {
  color: C.text,
  fontSize: 11,
  fontWeight: 800,
}

const panelCountDescriptionStyle: React.CSSProperties = {
  marginTop: 2,
  color: C.textMuted,
  fontSize: 9,
  lineHeight: 1.5,
}

const panelCountOptionsStyle: React.CSSProperties = {
  display: 'flex',
  gap: 6,
  flexWrap: 'wrap',
}

const panelCountButtonStyle: React.CSSProperties = {
  minWidth: 46,
  padding: '7px 9px',
  borderRadius: 8,
  border: '1px solid',
  fontSize: 10,
  fontWeight: 900,
}

const automaticPlanStyle: React.CSSProperties = {
  marginTop: 10,
  padding: '8px 10px',
  borderRadius: 8,
  background: C.background,
  color: C.textSecondary,
  fontSize: 9,
  lineHeight: 1.65,
}

const footerStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 10,
  marginTop: 11,
  flexWrap: 'wrap',
}

const draftStatusStyle: React.CSSProperties = {
  color: C.textMuted,
  fontSize: 9,
}

const actionStyle: React.CSSProperties = {
  display: 'flex',
  gap: 7,
  flexWrap: 'wrap',
}

const secondaryButtonStyle: React.CSSProperties = {
  padding: '7px 10px',
  borderRadius: 8,
  border: `1px solid ${C.border}`,
  background: C.white,
  color: C.textSecondary,
  fontSize: 10,
  fontWeight: 700,
  cursor: 'pointer',
}

const primaryButtonStyle: React.CSSProperties = {
  padding: '9px 15px',
  borderRadius: 9,
  border: 'none',
  color: '#FFFFFF',
  fontSize: 11,
  fontWeight: 900,
}
