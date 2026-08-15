/**
 * 教学智能体部署状态的同页即时同步事件。
 *
 * 使用场景：
 *   - 发布、追加版本、暂停、恢复、撤销或课堂使用策略更新成功后，
 *     通知当前页面中所有独立的教学智能体预览实例重新读取部署状态；
 *   - 解决“发布管理区已经保存成功，但课件画面里的悬浮智能体仍持有旧valid_until”
 *     这类同页多Hook实例状态不同步问题。
 *
 * 边界：
 *   - 只在当前浏览器页面内广播，不使用localStorage/sessionStorage；
 *   - 事件只携带courseware_id和稳定page_id，不携带部署正文、策略详情或身份信息；
 *   - 不调用API、不写数据库，真正数据仍以服务端重新读取结果为准。
 */

export const COURSEWARE_ASSISTANT_DEPLOYMENT_REFRESH_EVENT =
  'tedna:courseware-assistant:deployment-refresh'

interface CoursewareAssistantDeploymentRefreshDetail {
  coursewareId: string
  pageId: string
}

function normalizeResourceID(value: string): string {
  return value.trim()
}

export function publishCoursewareAssistantDeploymentRefresh(
  coursewareId: string,
  pageId: string,
): void {
  if (typeof window === 'undefined') return

  const normalizedCoursewareID = normalizeResourceID(coursewareId)
  const normalizedPageID = normalizeResourceID(pageId)

  if (!normalizedCoursewareID || !normalizedPageID) return

  window.dispatchEvent(
    new CustomEvent<CoursewareAssistantDeploymentRefreshDetail>(
      COURSEWARE_ASSISTANT_DEPLOYMENT_REFRESH_EVENT,
      {
        detail: {
          coursewareId: normalizedCoursewareID,
          pageId: normalizedPageID,
        },
      },
    ),
  )
}

export function subscribeCoursewareAssistantDeploymentRefresh(
  coursewareId: string,
  pageId: string,
  listener: () => void,
): () => void {
  if (typeof window === 'undefined') {
    return () => undefined
  }

  const normalizedCoursewareID = normalizeResourceID(coursewareId)
  const normalizedPageID = normalizeResourceID(pageId)

  if (!normalizedCoursewareID || !normalizedPageID) {
    return () => undefined
  }

  const handleRefresh = (event: Event) => {
    if (!(event instanceof CustomEvent)) return

    const detail = event.detail as
      | Partial<CoursewareAssistantDeploymentRefreshDetail>
      | undefined

    if (
      normalizeResourceID(detail?.coursewareId || '') !== normalizedCoursewareID
      || normalizeResourceID(detail?.pageId || '') !== normalizedPageID
    ) {
      return
    }

    listener()
  }

  window.addEventListener(
    COURSEWARE_ASSISTANT_DEPLOYMENT_REFRESH_EVENT,
    handleRefresh,
  )

  return () => {
    window.removeEventListener(
      COURSEWARE_ASSISTANT_DEPLOYMENT_REFRESH_EVENT,
      handleRefresh,
    )
  }
}
