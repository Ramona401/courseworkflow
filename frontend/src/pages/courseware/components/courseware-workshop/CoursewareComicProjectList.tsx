/**
 * CoursewareComicProjectList.tsx
 *
 * 知识点漫画轻量项目列表：
 *   - 支持教材项目和教师自由输入项目；
 *   - 不把出版社、册次或教材单元当作必填完成度；
 *   - 显示知识主题、格数、项目状态和五步工作流位置；
 *   - 点击后进入五步项目工作台。
 */

import {
  readCoursewareComicWorkflow,
} from '@/api/coursewares'

import type {
  CoursewareComicProject,
  CoursewareComicProjectStatus,
  CoursewareComicWorkflowStage,
} from '@/api/coursewares'

interface CoursewareComicProjectListProps {
  projects: CoursewareComicProject[]
  loading: boolean

  onOpen: (
    projectID: string,
  ) => void
}

const C = {
  primary: '#7C3AED',
  primaryBackground:
    'rgba(124,58,237,0.07)',
  success: '#059669',
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
      label: '待AI规划',
      color: '#64748B',
      background: '#F1F5F9',
    },
    planning: {
      label: 'AI规划中',
      color: '#7C3AED',
      background: '#F3E8FF',
    },
    planned: {
      label: '等待教师确认',
      color: '#2563EB',
      background: '#DBEAFE',
    },
    generating: {
      label: '图片生成中',
      color: '#D97706',
      background: '#FEF3C7',
    },
    ready: {
      label: '可精修使用',
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

export default function CoursewareComicProjectList({
  projects,
  loading,
  onOpen,
}: CoursewareComicProjectListProps) {
  return (
    <section style={sectionStyle}>
      <div style={sectionHeaderStyle}>
        <div>
          <div style={sectionTitleStyle}>
            已有漫画项目
          </div>

          <div style={sectionDescriptionStyle}>
            打开项目后按照“知识点—分镜—样张—自动生图—精修使用”继续完成。
          </div>
        </div>

        {!loading && (
          <span style={countStyle}>
            {projects.length} 个
          </span>
        )}
      </div>

      {loading ? (
        <div style={emptyStyle}>
          正在加载漫画项目…
        </div>
      ) : projects.length === 0 ? (
        <div style={emptyStyle}>
          当前课件还没有知识点漫画。
          在上方输入知识点即可创建第一个项目。
        </div>
      ) : (
        <div style={gridStyle}>
          {projects.map(
            project => (
              <ProjectCard
                key={project.id}
                project={project}
                onOpen={onOpen}
              />
            ),
          )}
        </div>
      )}
    </section>
  )
}

function ProjectCard({
  project,
  onOpen,
}: {
  project: CoursewareComicProject

  onOpen: (
    projectID: string,
  ) => void
}) {
  const status =
    STATUS_CONFIG[
      project.status
    ]

  const workflow =
    readCoursewareComicWorkflow(
      project,
    )

  const knowledgeTitle =
    resolveKnowledgeTitle(
      project,
    )

  const sourceLabel =
    resolveSourceLabel(
      project,
    )

  const workflowLabel =
    resolveWorkflowLabel(
      workflow?.stage,
      project.status,
    )

  return (
    <article style={cardStyle}>
      <div style={cardHeaderStyle}>
        <div style={cardTitleStyle}>
          {project.title}
        </div>

        <span style={{
          ...statusStyle,
          color: status.color,
          background:
            status.background,
        }}>
          {status.label}
        </span>
      </div>

      <div style={workflowStyle}>
        <span style={workflowNumberStyle}>
          {workflowLabel.number}
        </span>

        <span>
          当前步骤：
          {workflowLabel.label}
        </span>
      </div>

      <div style={knowledgeStyle}>
        {knowledgeTitle}
      </div>

      <div style={metadataStyle}>
        {project.panel_count}格
        {' · '}
        {sourceLabel}
      </div>

      {project.status ===
        'inserted' && (
        <div style={insertedStyle}>
          已插入课件第
          {
            project
              .inserted_page_number_snapshot
          }
          页
        </div>
      )}

      {project.last_error && (
        <div style={errorStyle}>
          {project.last_error}
        </div>
      )}

      <button
        type="button"
        onClick={() =>
          onOpen(
            project.id,
          )
        }
        style={openButtonStyle}
      >
        继续第
        {workflowLabel.number}
        步 →
      </button>
    </article>
  )
}

function resolveWorkflowLabel(
  stage:
    | CoursewareComicWorkflowStage
    | undefined,
  status:
    CoursewareComicProjectStatus,
): {
  number: number
  label: string
} {
  if (
    status === 'ready' ||
    status === 'inserted'
  ) {
    return {
      number: 5,
      label: '精修与使用',
    }
  }

  switch (stage) {
  case 'source':
    return {
      number: 1,
      label: '知识点来源',
    }

  case 'storyboard':
    return {
      number: 2,
      label: '确认故事分镜',
    }

  case 'style_preview':
    return {
      number: 3,
      label: '确认首格样张',
    }

  case 'batch_generation':
    return {
      number: 4,
      label: '自动生成图片',
    }

  case 'refinement':
    return {
      number: 5,
      label: '精修与使用',
    }

  default:
    if (status === 'planning') {
      return {
        number: 2,
        label: 'AI正在规划分镜',
      }
    }

    return {
      number: 1,
      label: '知识点来源',
    }
  }
}

function resolveKnowledgeTitle(
  project: CoursewareComicProject,
): string {
  const names =
    project.knowledge_points
      .map(
        item =>
          item.kp_name.trim(),
      )
      .filter(Boolean)

  if (names.length > 0) {
    return names
      .slice(0, 3)
      .join('、')
  }

  const knowledge =
    project.knowledge_content
      .replace(
        /^来源：[^\n]*\n?/,
        '',
      )
      .replace(
        /^知识点与教学要求：\n?/,
        '',
      )
      .trim()

  if (!knowledge) {
    return '知识点内容待查看'
  }

  const firstLine =
    knowledge
      .split('\n')
      .map(
        line =>
          line.trim(),
      )
      .find(Boolean) || ''

  const runes =
    Array.from(firstLine)

  return runes.length > 72
    ? `${runes.slice(0, 72).join('')}…`
    : firstLine
}

function resolveSourceLabel(
  project: CoursewareComicProject,
): string {
  if (
    project.publisher ===
      '教师自定义知识点' ||
    project.semester ===
      '自由输入'
  ) {
    return '教师输入'
  }

  const parts = [
    project.publisher,
    project.semester,
  ].filter(
    value =>
      value.trim(),
  )

  return parts.length > 0
    ? parts.join(' · ')
    : '知识点输入'
}

const sectionStyle:
  React.CSSProperties = {
    marginBottom: 12,
    padding: 13,
    borderRadius: 10,
    border:
      `1px solid ${C.border}`,
    background: C.white,
  }

const sectionHeaderStyle:
  React.CSSProperties = {
    display: 'flex',
    alignItems: 'flex-start',
    justifyContent:
      'space-between',
    gap: 10,
    marginBottom: 10,
  }

const sectionTitleStyle:
  React.CSSProperties = {
    color: C.text,
    fontSize: 12,
    fontWeight: 900,
  }

const sectionDescriptionStyle:
  React.CSSProperties = {
    marginTop: 4,
    color: C.textMuted,
    fontSize: 10,
    lineHeight: 1.6,
  }

const countStyle:
  React.CSSProperties = {
    flexShrink: 0,
    padding: '3px 8px',
    borderRadius: 999,
    color: C.textSecondary,
    background: C.background,
    fontSize: 9,
    fontWeight: 800,
  }

const gridStyle:
  React.CSSProperties = {
    display: 'grid',
    gridTemplateColumns:
      'repeat(auto-fit,minmax(250px,1fr))',
    gap: 9,
  }

const cardStyle:
  React.CSSProperties = {
    padding: 12,
    borderRadius: 10,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    boxShadow:
      '0 4px 14px rgba(15,23,42,0.04)',
  }

const cardHeaderStyle:
  React.CSSProperties = {
    display: 'flex',
    alignItems: 'flex-start',
    justifyContent:
      'space-between',
    gap: 8,
  }

const cardTitleStyle:
  React.CSSProperties = {
    minWidth: 0,
    color: C.text,
    fontSize: 12,
    fontWeight: 900,
    lineHeight: 1.5,
  }

const statusStyle:
  React.CSSProperties = {
    flexShrink: 0,
    padding: '2px 7px',
    borderRadius: 999,
    fontSize: 9,
    fontWeight: 800,
  }

const workflowStyle:
  React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 6,
    marginTop: 9,
    padding: '6px 8px',
    borderRadius: 8,
    background:
      C.primaryBackground,
    color: C.primary,
    fontSize: 9,
    fontWeight: 800,
  }

const workflowNumberStyle:
  React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    width: 18,
    height: 18,
    borderRadius: 999,
    background: C.primary,
    color: C.white,
    fontSize: 8,
    fontWeight: 900,
  }

const knowledgeStyle:
  React.CSSProperties = {
    marginTop: 8,
    color: C.textSecondary,
    fontSize: 10,
    lineHeight: 1.65,
  }

const metadataStyle:
  React.CSSProperties = {
    marginTop: 6,
    color: C.textMuted,
    fontSize: 9,
  }

const insertedStyle:
  React.CSSProperties = {
    marginTop: 7,
    color: C.success,
    fontSize: 10,
    fontWeight: 800,
  }

const errorStyle:
  React.CSSProperties = {
    marginTop: 7,
    padding: '7px 8px',
    borderRadius: 7,
    background: '#FEF2F2',
    color: C.danger,
    fontSize: 9,
    lineHeight: 1.55,
  }

const openButtonStyle:
  React.CSSProperties = {
    width: '100%',
    marginTop: 10,
    padding: '8px 10px',
    borderRadius: 8,
    border:
      '1px solid rgba(124,58,237,0.30)',
    background:
      C.primaryBackground,
    color: C.primary,
    fontSize: 10,
    fontWeight: 800,
    cursor: 'pointer',
  }

const emptyStyle:
  React.CSSProperties = {
    padding: '18px 10px',
    borderRadius: 8,
    border:
      `1px dashed ${C.border}`,
    background: C.background,
    color: C.textMuted,
    textAlign: 'center',
    fontSize: 11,
    lineHeight: 1.7,
  }
