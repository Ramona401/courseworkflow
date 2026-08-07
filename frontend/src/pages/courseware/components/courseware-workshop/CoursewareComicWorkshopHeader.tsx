/**
 * CoursewareComicWorkshopHeader.tsx
 *
 * 知识点漫画五步工作台标题栏。
 */

import type {
  CoursewareDetail,
} from '@/api/coursewares'

interface CoursewareComicWorkshopHeaderProps {
  courseware: CoursewareDetail
  loadingProjects: boolean
  onRefresh: () => void
}

const C = {
  text: '#1F2937',
  textSecondary: '#64748B',
  warning: '#B45309',
  border: '#E2E8F0',
  white: '#FFFFFF',
}

export default function CoursewareComicWorkshopHeader({
  courseware,
  loadingProjects,
  onRefresh,
}: CoursewareComicWorkshopHeaderProps) {
  return (
    <header style={containerStyle}>
      <div>
        <div style={titleStyle}>
          🗯️ 知识点漫画
        </div>

        <div style={descriptionStyle}>
          {courseware.subject}
          {' · '}
          {courseware.grade}
          {' · '}
          输入知识点后，依次确认AI分镜、画风与首格样张，
          再自动生成其余图片并完成单格精修。
        </div>

        <div style={workflowStyle}>
          1 知识点
          {' → '}
          2 确认分镜
          {' → '}
          3 确认样张
          {' → '}
          4 自动生图
          {' → '}
          5 精修使用
        </div>

        {courseware.style_anchor_asset_id && (
          <div style={anchorStyle}>
            ⭐ 当前课件视觉锚点会作为可选参考，
            教师在第3步确认的画风拥有更高优先级
          </div>
        )}
      </div>

      <button
        type="button"
        onClick={onRefresh}
        disabled={loadingProjects}
        style={refreshButtonStyle}
      >
        {loadingProjects
          ? '刷新中…'
          : '🔄 刷新项目'}
      </button>
    </header>
  )
}

const containerStyle:
  React.CSSProperties = {
    display: 'flex',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    gap: 12,
    marginBottom: 14,
  }

const titleStyle:
  React.CSSProperties = {
    color: C.text,
    fontSize: 15,
    fontWeight: 900,
  }

const descriptionStyle:
  React.CSSProperties = {
    marginTop: 4,
    color: C.textSecondary,
    fontSize: 11,
    lineHeight: 1.65,
  }

const workflowStyle:
  React.CSSProperties = {
    marginTop: 7,
    color: '#7C3AED',
    fontSize: 9,
    fontWeight: 800,
    lineHeight: 1.6,
  }

const anchorStyle:
  React.CSSProperties = {
    marginTop: 6,
    color: C.warning,
    fontSize: 10,
    fontWeight: 700,
    lineHeight: 1.55,
  }

const refreshButtonStyle:
  React.CSSProperties = {
    flexShrink: 0,
    padding: '7px 11px',
    borderRadius: 7,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    color: C.textSecondary,
    fontSize: 10,
    fontWeight: 700,
    cursor: 'pointer',
  }
