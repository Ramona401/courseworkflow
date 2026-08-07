/**
 * CoursewareComicWorkshopPanel.tsx
 *
 * 知识点漫画轻量工作台：
 *   - 首屏只要求老师输入知识点；
 *   - 不要求教材、出版社、册次、单元或课标编码；
 *   - 不要求老师选择标题、格数、叙事、风格或布局；
 *   - 创建后由后端自动补齐参数，并立即调用AI规划；
 *   - 已有项目仍可进入完整工作台生成图片、编辑、重画和插页；
 *   - 项目编辑器通过稳定回调桥挂载，避免父组件重渲染触发详情循环加载。
 */

import {
  useCallback,
  useEffect,
  useState,
} from 'react'

import {
  listCoursewareComicProjects,
} from '@/api/coursewares'

import type {
  CoursewareComicProject,
  CoursewareDetail,
} from '@/api/coursewares'

import CoursewareComicProjectEditorBridge from './CoursewareComicProjectEditorBridge'
import CoursewareComicProjectList from './CoursewareComicProjectList'
import CoursewareComicQuickCreatePanel from './CoursewareComicQuickCreatePanel'
import CoursewareComicWorkshopHeader from './CoursewareComicWorkshopHeader'

import {
  resolveCoursewareComicSelectedProjectID,
  sortCoursewareComicProjects,
  upsertCoursewareComicProject,
} from './coursewareComicWorkshopIntegration'

interface CoursewareComicWorkshopPanelProps {
  coursewareId: string
  courseware: CoursewareDetail
  pageCount: number

  onPagesChanged?: (
    pageNumber: number,
  ) => void | Promise<void>
}

const C = {
  danger: '#DC2626',
  warning: '#D97706',
  success: '#059669',
  border: '#E2E8F0',
  background: '#F8FAFC',
}

export default function CoursewareComicWorkshopPanel({
  coursewareId,
  courseware,
  pageCount,
  onPagesChanged,
}: CoursewareComicWorkshopPanelProps) {
  const [
    projects,
    setProjects,
  ] = useState<
    CoursewareComicProject[]
  >([])

  const [
    selectedProjectID,
    setSelectedProjectID,
  ] = useState('')

  const [
    loadingProjects,
    setLoadingProjects,
  ] = useState(true)

  const [
    notice,
    setNotice,
  ] = useState('')

  const loadProjects =
    useCallback(async () => {
      if (
        !coursewareId ||
        courseware.education_domain !==
          'k12'
      ) {
        setProjects([])
        setSelectedProjectID('')
        setLoadingProjects(false)
        return
      }

      setLoadingProjects(true)

      try {
        const result =
          await listCoursewareComicProjects(
            coursewareId,
          )

        const nextProjects =
          sortCoursewareComicProjects(
            result.projects || [],
          )

        setProjects(
          previous => {
            if (
              previous.length ===
                nextProjects.length &&
              previous.every(
                (
                  project,
                  index,
                ) =>
                  project.id ===
                    nextProjects[index]
                      ?.id &&
                  project.version ===
                    nextProjects[index]
                      ?.version &&
                  project.updated_at ===
                    nextProjects[index]
                      ?.updated_at,
              )
            ) {
              return previous
            }

            return nextProjects
          },
        )

        setSelectedProjectID(
          previous =>
            resolveCoursewareComicSelectedProjectID(
              previous,
              nextProjects,
            ),
        )
      } catch (error) {
        setNotice(
          '❌ ' +
            errorMessage(
              error,
              '漫画项目加载失败',
            ),
        )
      } finally {
        setLoadingProjects(false)
      }
    }, [
      coursewareId,
      courseware.education_domain,
    ])

  useEffect(() => {
    void loadProjects()
  }, [
    loadProjects,
  ])

  const handleBackToProjects =
    useCallback(() => {
      setSelectedProjectID('')
      void loadProjects()
    }, [
      loadProjects,
    ])

  const handleProjectChanged =
    useCallback(
      (
        updated:
          CoursewareComicProject,
      ) => {
        setProjects(
          previous =>
            upsertCoursewareComicProject(
              previous,
              updated,
            ),
        )
      },
      [],
    )

  const handleProjectCreated =
    useCallback(
      (
        project:
          CoursewareComicProject,
      ) => {
        setProjects(
          previous =>
            upsertCoursewareComicProject(
              previous,
              project,
            ),
        )

        setSelectedProjectID(
          project.id,
        )
      },
      [],
    )

  const handleOpenProject =
    useCallback(
      (
        projectID: string,
      ) => {
        setSelectedProjectID(
          projectID,
        )
      },
      [],
    )

  const handleRefreshProjects =
    useCallback(() => {
      void loadProjects()
    }, [
      loadProjects,
    ])

  if (
    courseware.education_domain !==
      'k12'
  ) {
    return (
      <section style={panelStyle}>
        <div style={titleStyle}>
          🗯️ 知识点漫画
        </div>

        <div style={warningStyle}>
          当前版本暂时支持K12课件。
          漫画创建不要求关联教材，但仍会使用课件的学科、年级和视觉风格作为AI规划背景。
        </div>
      </section>
    )
  }

  if (selectedProjectID) {
    return (
      <section style={panelStyle}>
        <CoursewareComicProjectEditorBridge
          coursewareId={
            coursewareId
          }
          projectId={
            selectedProjectID
          }
          pageCount={
            pageCount
          }
          onBack={
            handleBackToProjects
          }
          onProjectChanged={
            handleProjectChanged
          }
          onPagesChanged={
            onPagesChanged
          }
        />
      </section>
    )
  }

  return (
    <section style={panelStyle}>
      <CoursewareComicWorkshopHeader
        courseware={courseware}
        loadingProjects={
          loadingProjects
        }
        onRefresh={
          handleRefreshProjects
        }
      />

      {notice && (
        <Notice text={notice} />
      )}

      <CoursewareComicQuickCreatePanel
        coursewareId={
          coursewareId
        }
        courseware={
          courseware
        }
        onNotice={
          setNotice
        }
        onProjectCreated={
          handleProjectCreated
        }
      />

      <CoursewareComicProjectList
        projects={projects}
        loading={
          loadingProjects
        }
        onOpen={
          handleOpenProject
        }
      />
    </section>
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
      marginBottom: 12,
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
            ? C.warning
            : C.success,
      fontSize: 11,
      lineHeight: 1.6,
    }}>
      {text}
    </div>
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

const panelStyle:
  React.CSSProperties = {
    marginTop: 16,
    padding: 16,
    borderRadius: 12,
    border:
      `1px solid ${C.border}`,
    background: C.background,
  }

const titleStyle:
  React.CSSProperties = {
    color: '#1F2937',
    fontSize: 15,
    fontWeight: 900,
  }

const warningStyle:
  React.CSSProperties = {
    marginTop: 12,
    padding: 12,
    borderRadius: 9,
    border:
      '1px solid #FDE68A',
    background: '#FFFBEB',
    color: '#92400E',
    fontSize: 11,
    lineHeight: 1.7,
  }
