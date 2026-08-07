/**
 * coursewareComicWorkshopIntegration.ts
 *
 * 漫画创建面板与项目详情编辑器之间的纯函数。
 *
 * 作用：
 *   - 更新列表中的单个项目；
 *   - 服务端内容没有变化时保留原数组引用；
 *   - 保持项目列表按最近更新时间排序；
 *   - 刷新列表后验证当前选中项目是否仍存在；
 *   - 不保存授权字段，不发起网络请求。
 */

import type {
  CoursewareComicProject,
} from '@/api/coursewares'

function projectTimestamp(
  project: CoursewareComicProject,
): number {
  const updated =
    project.updated_at
      ? Date.parse(
          project.updated_at,
        )
      : Number.NaN

  if (Number.isFinite(updated)) {
    return updated
  }

  const created =
    project.created_at
      ? Date.parse(
          project.created_at,
        )
      : Number.NaN

  return Number.isFinite(created)
    ? created
    : 0
}

/**
 * 项目视图全部来自同一浏览器安全API协议，
 * JSON字段顺序稳定，可以用序列化结果判断内容是否真正变化。
 *
 * 该比较只用于最多几十个项目的本地列表，
 * 不参与授权、版本控制或持久化判断。
 */
export function coursewareComicProjectsEqual(
  left: CoursewareComicProject,
  right: CoursewareComicProject,
): boolean {
  if (left === right) {
    return true
  }

  if (
    left.id !== right.id ||
    left.version !== right.version ||
    left.status !== right.status ||
    left.updated_at !== right.updated_at
  ) {
    return false
  }

  try {
    return (
      JSON.stringify(left) ===
      JSON.stringify(right)
    )
  } catch {
    return false
  }
}

export function sortCoursewareComicProjects(
  projects: CoursewareComicProject[],
): CoursewareComicProject[] {
  return [...projects].sort(
    (left, right) => {
      const timeDifference =
        projectTimestamp(right) -
        projectTimestamp(left)

      if (timeDifference !== 0) {
        return timeDifference
      }

      return left.title.localeCompare(
        right.title,
        'zh-CN',
      )
    },
  )
}

export function upsertCoursewareComicProject(
  projects: CoursewareComicProject[],
  updated: CoursewareComicProject,
): CoursewareComicProject[] {
  const existingIndex =
    projects.findIndex(
      project =>
        project.id ===
        updated.id,
    )

  if (existingIndex < 0) {
    return sortCoursewareComicProjects([
      updated,
      ...projects,
    ])
  }

  const existing =
    projects[existingIndex]

  if (
    coursewareComicProjectsEqual(
      existing,
      updated,
    )
  ) {
    return projects
  }

  const next =
    projects.map(
      (project, index) =>
        index === existingIndex
          ? updated
          : project,
    )

  return sortCoursewareComicProjects(
    next,
  )
}

export function resolveCoursewareComicSelectedProjectID(
  previousID: string,
  projects: CoursewareComicProject[],
): string {
  const normalized =
    previousID.trim()

  if (!normalized) {
    return ''
  }

  return projects.some(
    project =>
      project.id ===
      normalized,
  )
    ? normalized
    : ''
}
