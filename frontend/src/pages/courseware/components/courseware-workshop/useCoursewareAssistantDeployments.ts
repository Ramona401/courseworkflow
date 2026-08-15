/**
 * 页面教学智能体部署管理Hook。
 *
 * 职责：
 *   - 读取当前课件的部署历史并筛选当前稳定page_id；
 *   - 读取当前未撤销部署的不可变版本元数据；
 *   - 首次发布、追加版本、暂停、恢复、撤销和更新运行策略；
 *   - 页面切换后阻止旧异步响应覆盖新页面状态。
 *
 * 安全边界：
 *   - 部署所有者和付费身份由教师JWT与后端数据库确定；
 *   - 浏览器不能提交owner_user_id、school_id或积分账户；
 *   - 不读取完整提示词、上下文快照或页面HTML；
 *   - 撤销是永久状态，必须由正式确认弹窗触发。
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  listCoursewareAssistantDeployments,
  listCoursewareAssistantDeploymentVersions,
  pauseCoursewareAssistantDeployment,
  publishCoursewareAssistantDeployment,
  publishCoursewareAssistantDeploymentVersion,
  resumeCoursewareAssistantDeployment,
  revokeCoursewareAssistantDeployment,
  updateCoursewareAssistantDeploymentPolicy,
} from '@/api/coursewares'
import type {
  CoursewareAssistantDeploymentVersionView,
  CoursewareAssistantDeploymentView,
  PublishCoursewareAssistantDeploymentRequest,
  UpdateCoursewareAssistantDeploymentPolicyRequest,
} from '@/api/coursewares'

import {
  publishCoursewareAssistantDeploymentRefresh,
} from './coursewareAssistantDeploymentSync'

export type CoursewareAssistantDeploymentAction =
  | ''
  | 'publish'
  | 'version'
  | 'pause'
  | 'resume'
  | 'revoke'
  | 'policy'

export interface CoursewareAssistantDeploymentNotice {
  kind: 'success' | 'info' | 'error'
  text: string
}

interface UseCoursewareAssistantDeploymentsOptions {
  coursewareId: string
  pageId: string
  onChanged?: () => void
}

function timestamp(value: string | null): number {
  if (!value) return 0
  const parsed = Date.parse(value)
  return Number.isFinite(parsed) ? parsed : 0
}

function sortDeployments(items: CoursewareAssistantDeploymentView[]): CoursewareAssistantDeploymentView[] {
  return [...items].sort((left, right) => timestamp(right.updated_at) - timestamp(left.updated_at))
}

function sortVersions(items: CoursewareAssistantDeploymentVersionView[]): CoursewareAssistantDeploymentVersionView[] {
  return [...items].sort((left, right) => right.version - left.version)
}

export function useCoursewareAssistantDeployments({
  coursewareId,
  pageId,
  onChanged,
}: UseCoursewareAssistantDeploymentsOptions) {
  const resourceKey = `${coursewareId.trim()}:${pageId.trim()}`
  const resourceRef = useRef(resourceKey)
  resourceRef.current = resourceKey

  const loadRequestRef = useRef(0)
  const operationRef = useRef(0)

  const [pageDeployments, setPageDeployments] = useState<CoursewareAssistantDeploymentView[]>([])
  const [versions, setVersions] = useState<CoursewareAssistantDeploymentVersionView[]>([])
  const [loading, setLoading] = useState(false)
  const [workingAction, setWorkingAction] = useState<CoursewareAssistantDeploymentAction>('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState<CoursewareAssistantDeploymentNotice | null>(null)

  const liveDeployment = useMemo(
    () => pageDeployments.find((item) => item.status === 'active' || item.status === 'paused') || null,
    [pageDeployments],
  )

  const latestPageDeployment = pageDeployments[0] || null
  const latestVersion = versions[0] || null

  const load = useCallback(async () => {
    const normalizedCoursewareID = coursewareId.trim()
    const normalizedPageID = pageId.trim()
    const requestID = loadRequestRef.current + 1
    loadRequestRef.current = requestID

    if (!normalizedCoursewareID || !normalizedPageID) {
      setPageDeployments([])
      setVersions([])
      setLoading(false)
      setError('')
      return
    }

    const capturedResource = resourceKey
    setLoading(true)
    setError('')

    try {
      const response = await listCoursewareAssistantDeployments(normalizedCoursewareID)
      if (loadRequestRef.current !== requestID || resourceRef.current !== capturedResource) return

      const currentPageDeployments = sortDeployments(
        (response.deployments || []).filter((item) => item.page_id === normalizedPageID),
      )
      const currentLive = currentPageDeployments.find((item) => item.status === 'active' || item.status === 'paused') || null

      setPageDeployments(currentPageDeployments)

      if (!currentLive) {
        setVersions([])
        return
      }

      const versionResult = await listCoursewareAssistantDeploymentVersions(currentLive.id)
      if (loadRequestRef.current !== requestID || resourceRef.current !== capturedResource) return
      setVersions(sortVersions(versionResult || []))
    } catch (cause) {
      if (loadRequestRef.current !== requestID || resourceRef.current !== capturedResource) return
      setPageDeployments([])
      setVersions([])
      setError(cause instanceof Error ? cause.message : '读取教学智能体部署状态失败')
    } finally {
      if (loadRequestRef.current === requestID && resourceRef.current === capturedResource) setLoading(false)
    }
  }, [coursewareId, pageId, resourceKey])

  useEffect(() => {
    setNotice(null)
    void load()

    return () => {
      loadRequestRef.current += 1
      operationRef.current += 1
    }
  }, [load, resourceKey])

  const runAction = useCallback(
    async <T,>(
      action: CoursewareAssistantDeploymentAction,
      successText: string,
      task: () => Promise<T>,
    ): Promise<T | null> => {
      if (workingAction) return null

      const operationID = operationRef.current + 1
      operationRef.current = operationID
      const capturedResource = resourceKey

      setWorkingAction(action)
      setError('')
      setNotice(null)

      try {
        const result = await task()
        if (operationRef.current !== operationID || resourceRef.current !== capturedResource) return null

        await load()
        if (operationRef.current !== operationID || resourceRef.current !== capturedResource) return null

        setNotice({ kind: 'success', text: successText })

        // 当前课件页可能同时挂载管理区预览、画布悬浮预览、全屏或放映预览。
        // 所有部署写操作成功后统一广播，让各独立Hook只重读当前页部署状态，
        // 不刷新整个课件页面，也不复制服务端返回数据作为第二事实源。
        publishCoursewareAssistantDeploymentRefresh(
          coursewareId,
          pageId,
        )

        onChanged?.()
        return result
      } catch (cause) {
        if (operationRef.current !== operationID || resourceRef.current !== capturedResource) return null
        setNotice({ kind: 'error', text: cause instanceof Error ? cause.message : '教学智能体部署操作失败' })
        return null
      } finally {
        if (operationRef.current === operationID && resourceRef.current === capturedResource) setWorkingAction('')
      }
    },
    [
      coursewareId,
      load,
      onChanged,
      pageId,
      resourceKey,
      workingAction,
    ],
  )

  const publishFirst = useCallback(
    (request: PublishCoursewareAssistantDeploymentRequest) =>
      runAction(
        'publish',
        '教学智能体已发布为不可变版本V1。默认只允许当前TE-DNA站点使用。',
        () => publishCoursewareAssistantDeployment(coursewareId, pageId, request),
      ),
    [coursewareId, pageId, runAction],
  )

  const publishVersion = useCallback(
    () =>
      liveDeployment
        ? runAction(
            'version',
            '当前已保存方案已发布为新的不可变版本。已有旧版本会话将按安全规则失效。',
            () => publishCoursewareAssistantDeploymentVersion(liveDeployment.id),
          )
        : Promise.resolve(null),
    [liveDeployment, runAction],
  )

  const pause = useCallback(
    () =>
      liveDeployment
        ? runAction(
            'pause',
            '教学智能体部署已暂停，不能建立新的学生或教师预览会话。',
            () => pauseCoursewareAssistantDeployment(liveDeployment.id),
          )
        : Promise.resolve(null),
    [liveDeployment, runAction],
  )

  const resume = useCallback(
    () =>
      liveDeployment
        ? runAction(
            'resume',
            '教学智能体部署已恢复运行。',
            () => resumeCoursewareAssistantDeployment(liveDeployment.id),
          )
        : Promise.resolve(null),
    [liveDeployment, runAction],
  )

  const revoke = useCallback(
    () =>
      liveDeployment
        ? runAction(
            'revoke',
            '教学智能体部署已永久撤销。历史版本仅保留审计记录，不能恢复运行。',
            () => revokeCoursewareAssistantDeployment(liveDeployment.id),
          )
        : Promise.resolve(null),
    [liveDeployment, runAction],
  )

  const updatePolicy = useCallback(
    (request: UpdateCoursewareAssistantDeploymentPolicyRequest) =>
      liveDeployment
        ? runAction(
            'policy',
            '教学智能体使用权限和运行额度已更新。',
            () => updateCoursewareAssistantDeploymentPolicy(liveDeployment.id, request),
          )
        : Promise.resolve(null),
    [liveDeployment, runAction],
  )

  return {
    pageDeployments,
    liveDeployment,
    latestPageDeployment,
    versions,
    latestVersion,
    loading,
    workingAction,
    error,
    notice,
    load,
    publishFirst,
    publishVersion,
    pause,
    resume,
    revoke,
    updatePolicy,
  }
}
