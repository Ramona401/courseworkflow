/**
 * CoursewareWorkshopPage.tsx — 课件工坊路由级自动装配恢复门
 *
 * 本文件只负责进入工坊前的生命周期分流：
 *   1. 同时读取课件详情与数据库装配状态；
 *   2. 当前active assembly，或终态但R-04完整性未通过的assembly，恢复自动装配面板；
 *   3. active batch 与其它状态都进入原课件工坊内容组件；
 *   4. 已完整历史终态不恢复；终态未完整运行只恢复对账与补生入口。
 *
 * 原完整工坊实现保存在 CoursewareWorkshopContent.tsx。
 */
import {
  useEffect,
  useState,
} from 'react'

import { useParams } from 'react-router-dom'

import { useAuth } from '@/store/auth'

import {
  getCourseware,
} from '@/api/coursewares'
import type {
  CoursewareDetail,
} from '@/api/coursewares'

import {
  getCoursewareAssemblyState,
} from '@/api/coursewareAssembly'
import type {
  CoursewareAssemblyState,
} from '@/api/coursewareAssembly'

import AutoAssemblyPanel
  from './components/courseware-workshop/AutoAssemblyPanel'
import {
  shouldRecoverIncompleteAssemblyState,
} from './components/courseware-workshop/coursewareAssemblyRuntimeState'
import {
  C,
} from './components/courseware-workshop/workshopConstants'
import CoursewareWorkshopContent
  from './CoursewareWorkshopContent'

type GateMode =
  | 'checking'
  | 'content'
  | 'resume'
  | 'error'

export default function CoursewareWorkshopPage() {
  const { id } = useParams<{
    id: string
  }>()

  const { user } = useAuth()
  const currentUserID = user?.id

  const [mode, setMode] =
    useState<GateMode>('checking')
  const [courseware, setCourseware] =
    useState<CoursewareDetail | null>(
      null,
    )
  const [initialAssemblyState, setInitialAssemblyState] =
    useState<CoursewareAssemblyState | null>(
      null,
    )
  const [retryKey, setRetryKey] =
    useState(0)

  useEffect(() => {
    let disposed = false

    const resolveRoute = async () => {
      if (!id) {
        if (!disposed) {
          setMode('error')
        }
        return
      }

      setMode('checking')

      const [
        coursewareResult,
        assemblyResult,
      ] = await Promise.allSettled([
        getCourseware(id),
        getCoursewareAssemblyState(id),
      ])

      if (disposed) return

      // 课件详情是工坊的必要数据；读取失败才阻断路由。
      if (
        coursewareResult.status ===
        'rejected'
      ) {
        setMode('error')
        return
      }

      // 装配状态端点是作者专属能力。
      //
      // 集体备课参与者被拒绝读取该状态时直接进入原工坊。
      // 作者遇到瞬时网络错误时额外重试一次，降低活动装配被误放行到
      // 普通编辑界面的概率；第二次仍失败则降级进入原工坊，不永久阻断访问。
      let resolvedAssemblyState =
        assemblyResult.status ===
        'fulfilled'
          ? assemblyResult.value
          : null

      const isOwner =
        Boolean(
          currentUserID &&
          currentUserID ===
            coursewareResult.value.user_id,
        )

      if (
        !resolvedAssemblyState &&
        isOwner
      ) {
        await new Promise<void>(
          resolve => {
            window.setTimeout(
              resolve,
              500,
            )
          },
        )

        if (disposed) return

        try {
          resolvedAssemblyState =
            await getCoursewareAssemblyState(
              id,
            )
        } catch {
          // 第二次生命周期查询仍失败时降级进入原工坊。
        }
      }

      const shouldResumeAssembly =
        Boolean(
          resolvedAssemblyState?.is_active &&
            resolvedAssemblyState.run_kind ===
              'assembly',
        ) ||
        shouldRecoverIncompleteAssemblyState(
          resolvedAssemblyState,
        )

      if (
        shouldResumeAssembly &&
        resolvedAssemblyState
      ) {
        setCourseware(
          coursewareResult.value,
        )
        setInitialAssemblyState(
          resolvedAssemblyState,
        )
        setMode('resume')
        return
      }

      setCourseware(null)
      setInitialAssemblyState(null)
      setMode('content')
    }

    void resolveRoute()

    return () => {
      disposed = true
    }
  }, [currentUserID, id, retryKey])

  if (mode === 'content') {
    return <CoursewareWorkshopContent />
  }

  if (mode === 'checking') {
    return (
      <div
        style={{
          textAlign: 'center',
          padding: '80px 0',
          color: C.textMuted,
        }}
      >
        <div
          style={{
            fontSize: 36,
            marginBottom: 12,
          }}
        >
          🔄
        </div>
        正在确认课件与后台装配状态…
      </div>
    )
  }

  if (
    mode === 'error' ||
    !id
  ) {
    return (
      <div
        style={{
          textAlign: 'center',
          padding: '80px 0',
          color: C.textMuted,
        }}
      >
        <div
          style={{
            fontSize: 36,
            marginBottom: 12,
          }}
        >
          ⚠️
        </div>
        暂时无法读取课件后台状态。
        <div
          style={{
            marginTop: 16,
          }}
        >
          <button
            type="button"
            onClick={() => {
              setRetryKey(
                current =>
                  current + 1,
              )
            }}
            style={{
              padding: '9px 20px',
              borderRadius: 8,
              border: 'none',
              background: C.primary,
              color: '#fff',
              cursor: 'pointer',
              fontSize: 13,
              fontWeight: 600,
            }}
          >
            重新检查
          </button>
        </div>
      </div>
    )
  }

  if (
    !courseware ||
    !initialAssemblyState
  ) {
    return null
  }

  return (
    <div
      style={{
        maxWidth: 960,
        margin: '0 auto',
      }}
    >
      <div
        style={{
          marginBottom: 18,
        }}
      >
        <h2
          style={{
            margin: 0,
            color: C.textPrimary,
            fontSize: 22,
            fontWeight: 700,
          }}
        >
          {courseware.title}
        </h2>
        <div
          style={{
            marginTop: 6,
            color: C.textMuted,
            fontSize: 13,
            lineHeight: 1.6,
          }}
        >
          {initialAssemblyState.is_active
            ? '检测到该课件仍有后台自动装配运行，已直接恢复运行面板。'
            : '检测到上次自动装配未完整生成，已恢复完整性对账与补生入口。'}
        </div>
      </div>

      <div
        style={{
          padding: 24,
          minHeight: 400,
          borderRadius: 12,
          border:
            `1px solid ${C.border}`,
          background: C.white,
        }}
      >
        <AutoAssemblyPanel
          coursewareId={id}
          courseware={courseware}
          skipVideo={
            initialAssemblyState
              .skip_video
          }
          initialState={
            initialAssemblyState
          }
          onDone={() => {
            setMode('content')
          }}
          onBack={() => {
            setMode('content')
          }}
        />
      </div>
    </div>
  )
}
