/**
 * useCoursewareAssemblySSE.ts — 自动装配SSE订阅与页面阶段映射
 *
 * 只负责长连接生命周期和SSE事件到前端状态的映射。
 * 数据库运行身份、启动、轮询、取消和最终完成通知仍由useCoursewareAssemblyRuntime负责。
 */
import {
  useEffect,
  useRef,
} from 'react'
import type {
  Dispatch,
  SetStateAction,
} from 'react'

import {
  subscribeCWIndexSSE,
} from '@/api/coursewares'
import type {
  CoursewareAssemblyState,
} from '@/api/coursewareAssembly'

import type {
  AssemblyPageState,
  AssemblyStageState,
  AssemblySummary,
} from './AssemblyProgressView'
import {
  createBlankAssemblyPage,
  isCoursewareAssemblyImageRepairRetry,
  isCoursewareAssemblyIntegrityRetry,
} from './coursewareAssemblyRuntimeState'

interface RefBox<T> {
  current: T
}

interface Options {
  coursewareId: string
  started: boolean
  done: boolean

  mountedRef: RefBox<boolean>
  launchBaselineRef:
    RefBox<CoursewareAssemblyState | null>
  skipVideoRef: RefBox<boolean>
  sseDoneRef: RefBox<boolean>

  setStarted: Dispatch<SetStateAction<boolean>>
  setStarting: Dispatch<SetStateAction<boolean>>
  setPageStates:
    Dispatch<SetStateAction<AssemblyPageState[]>>
  setSummary: Dispatch<SetStateAction<AssemblySummary>>

  patchPage: (
    pageNumber: number,
    patch: Partial<AssemblyPageState>,
  ) => void
  applyMediaStage: (
    pageNumber: number,
    stage: string,
    note: string,
  ) => void
  refreshAssemblyState:
    () => Promise<CoursewareAssemblyState | null>
  shouldAcceptTerminal:
    (state: CoursewareAssemblyState) => boolean
  syncStoredPages: () => Promise<void>
}

export function useCoursewareAssemblySSE({
  coursewareId,
  started,
  done,
  mountedRef,
  launchBaselineRef,
  skipVideoRef,
  sseDoneRef,
  setStarted,
  setStarting,
  setPageStates,
  setSummary,
  patchPage,
  applyMediaStage,
  refreshAssemblyState,
  shouldAcceptTerminal,
  syncStoredPages,
}: Options) {
  const subscriptionRef =
    useRef<{ close: () => void } | null>(null)

  useEffect(() => {
    if (!started || done) {
      return
    }

    subscriptionRef.current?.close()

    const subscription = subscribeCWIndexSSE(
      coursewareId,
      {
        onConnected: () => {
          setSummary(current => ({
            ...current,
            message:
              current.runtime_status === 'starting'
                ? '已连接，装配即将开始…'
                : current.message ||
                  '已连接后台装配进度。',
          }))
        },

        onAssemblyStart: data => {
          skipVideoRef.current = data.skip_video

          setSummary(current => ({
            ...current,
            total_pages: data.total_pages,
            skip_video: data.skip_video,
            running: true,
            done: false,
            runtime_status: 'running',
            message: data.message,
          }))

          setPageStates(previous => {
            const baseline =
              launchBaselineRef.current
            const preserveExisting =
              isCoursewareAssemblyIntegrityRetry(
                baseline,
              ) ||
              isCoursewareAssemblyImageRepairRetry(
                baseline,
              )

            if (preserveExisting) {
              return previous
            }

            return Array.from(
              { length: data.total_pages },
              (_, index) =>
                createBlankAssemblyPage(
                  index + 1,
                  `第 ${index + 1} 页`,
                  data.skip_video,
                ),
            )
          })

          window.setTimeout(() => {
            void refreshAssemblyState()
          }, 100)
        },

        onAssemblyPageHtml: data => {
          patchPage(data.page_number, {
            title: data.title,
            html: 'ok',
            image: 'running',
            note: '正在配图…',
          })
        },

        onAssemblyProgress: data => {
          patchPage(data.page_number, {
            title: data.page_title || undefined,
            html: 'failed',
            image: 'skipped',
            video: 'skipped',
            note: data.error
              ? `HTML生成失败：${data.error}`
              : 'HTML生成失败',
          })
        },

        onAssemblyPageMedia: data => {
          applyMediaStage(
            data.page_number,
            data.stage,
            data.message,
          )
        },

        onAssemblyPageDone: data => {
          const image: AssemblyStageState =
            data.image_ok
              ? 'ok'
              : data.image_skipped
                ? 'skipped'
                : 'failed'
          const video: AssemblyStageState =
            skipVideoRef.current
              ? 'skipped'
              : data.video_ok
                ? 'ok'
                : data.video_skipped
                  ? 'skipped'
                  : 'pending'

          patchPage(data.page_number, {
            title: data.title || undefined,
            html: 'ok',
            image,
            video,
            note: undefined,
          })
        },

        onAssemblyDone: data => {
          sseDoneRef.current = true

          setSummary(current => ({
            ...current,
            message: data.message,
            html_success: data.html_success,
            html_fail: data.html_fail,
            image_success: data.image_success,
            image_fail: data.image_fail,
            image_skip: data.image_skip,
            video_success: data.video_success,
            video_skip: data.video_skip,
            elapsed_ms: data.elapsed_ms,
            errors: data.errors,
          }))

          window.setTimeout(() => {
            void refreshAssemblyState()
          }, 350)
        },

        onError: data => {
          setSummary(current => ({
            ...current,
            message: `❌ ${data.message}`,
          }))

          window.setTimeout(async () => {
            const state =
              await refreshAssemblyState()

            if (
              mountedRef.current &&
              state &&
              !state.is_active &&
              (
                state.runtime_status === 'idle' ||
                !shouldAcceptTerminal(state)
              )
            ) {
              setStarting(false)
              setStarted(true)
              setSummary(current => ({
                ...current,
                running: false,
                done: true,
                runtime_status: 'failed',
              }))
            }
          }, 500)
        },

        onReconnected: () => {
          setSummary(current => ({
            ...current,
            restored: true,
            message:
              '🔄 连接已恢复，正在同步数据库状态与已生成页面…',
          }))

          void syncStoredPages()
          void refreshAssemblyState()
        },
      },
    )

    subscriptionRef.current = subscription

    return () => {
      subscription.close()

      if (subscriptionRef.current === subscription) {
        subscriptionRef.current = null
      }
    }
  }, [
    applyMediaStage,
    coursewareId,
    done,
    launchBaselineRef,
    mountedRef,
    patchPage,
    refreshAssemblyState,
    setPageStates,
    setStarted,
    setStarting,
    setSummary,
    shouldAcceptTerminal,
    skipVideoRef,
    sseDoneRef,
    started,
    syncStoredPages,
  ])
}
