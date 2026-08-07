/**
 * useCoursewareAssemblyRuntime.ts — 自动装配前端生命周期控制器
 *
 * 集中处理启动、刷新恢复、SSE、数据库轮询、页面同步和显式取消。
 *
 * 运行身份原则：
 *   - 路由恢复门只把当前 active 运行作为 initialState 传入；
 *   - 本地新启动先记录启动前 assembly_version；
 *   - 终态只有在观察到当前数据库运行，或版本号相对启动前递增后才接受；
 *   - 历史 completed/failed/cancelled/interrupted 不会冒充本次运行。
 */
import { useCallback, useEffect, useRef, useState } from 'react'

import { autoAssemble, getCoursewarePages, subscribeCWIndexSSE } from '@/api/coursewares'
import type { CoursewareDetail } from '@/api/coursewares'

import { cancelCoursewareAutoAssembly, getCoursewareAssemblyState } from '@/api/coursewareAssembly'
import type { CoursewareAssemblyState } from '@/api/coursewareAssembly'

import type { AssemblyPageState, AssemblyStageState, AssemblySummary } from './AssemblyProgressView'

import {
  createAssemblyMediaStagePatch,
  createAssemblyPagesFromStoredCourseware,
  createBlankAssemblyPage,
  getAssemblyRuntimeMessage,
  isAssemblyTerminalRuntime,
  mergeAssemblyPagesFromStoredCourseware,
  shouldAcceptAssemblyTerminal,
} from './coursewareAssemblyRuntimeState'

interface Options {
  coursewareId: string
  courseware: CoursewareDetail
  skipVideo: boolean
  onDone: () => void

  /** 仅路由恢复门传入当前 active 运行，历史终态不得注入。 */
  initialState?: CoursewareAssemblyState | null
}

export interface CoursewareAssemblyRuntimeController {
  started: boolean
  starting: boolean
  cancelling: boolean
  pageStates: AssemblyPageState[]
  summary: AssemblySummary
  runAssembly: () => Promise<void>
  cancelAssembly: () => Promise<void>
  refreshAssemblyState: () => Promise<CoursewareAssemblyState | null>
  syncStoredPages: () => Promise<void>
}

export function useCoursewareAssemblyRuntime({
  coursewareId,
  courseware,
  skipVideo,
  onDone,
  initialState,
}: Options): CoursewareAssemblyRuntimeController {
  const initialActive = Boolean(initialState?.is_active)
  const initialSkipVideo = initialState?.skip_video ?? skipVideo
  const initialRuntime = initialState?.runtime_status ?? 'idle'

  const [started, setStarted] = useState(initialActive)
  const [starting, setStarting] = useState(
    initialRuntime === 'starting',
  )
  const [cancelling, setCancelling] = useState(false)
  const [pageStates, setPageStates] = useState<AssemblyPageState[]>(
    () =>
      createAssemblyPagesFromStoredCourseware(
        courseware.pages || [],
        initialSkipVideo,
      ),
  )
  const [summary, setSummary] = useState<AssemblySummary>({
    total_pages:
      courseware.page_count ||
      courseware.pages?.length ||
      0,
    skip_video: initialSkipVideo,
    running: initialActive,
    done: false,
    runtime_status: initialActive ? initialRuntime : 'idle',
    message: initialActive ? getAssemblyRuntimeMessage(initialRuntime) : '',
    restored: initialActive,
  })

  const sseRef = useRef<{ close: () => void } | null>(null)
  const mountedRef = useRef(true)
  const cancelRequestedRef = useRef(false)
  const completionNotifiedRef = useRef(false)
  const launchingRef = useRef(false)
  const skipVideoRef = useRef(initialSkipVideo)
  const onDoneRef = useRef(onDone)

  // observedRunVersion只记录已落库运行；starting票据不能冒充数据库版本。
  const observedRunVersionRef = useRef(
    initialState?.active_run_id
      ? initialState.assembly_version
      : 0,
  )
  const localStartRef = useRef(false)
  const baselineVersionRef = useRef(
    initialState?.assembly_version ?? 0,
  )
  const baselineKnownRef = useRef(Boolean(initialState))
  const sseDoneRef = useRef(false)
  const preRunObservedRef = useRef(
    initialActive && initialRuntime === 'starting',
  )

  useEffect(() => {
    onDoneRef.current = onDone
  }, [onDone])

  useEffect(() => {
    skipVideoRef.current = summary.skip_video
  }, [summary.skip_video])

  useEffect(() => {
    mountedRef.current = true

    return () => {
      mountedRef.current = false
      sseRef.current?.close()
      sseRef.current = null
    }
  }, [])

  const patchPage = useCallback(
    (
      pageNumber: number,
      patch: Partial<AssemblyPageState>,
    ) => {
      setPageStates(previous => {
        const index = previous.findIndex(
          page => page.page_number === pageNumber,
        )

        if (index === -1) {
          return [
            ...previous,
            {
              ...createBlankAssemblyPage(
                pageNumber,
                patch.title || `第 ${pageNumber} 页`,
                skipVideoRef.current,
              ),
              ...patch,
            },
          ].sort((left, right) => left.page_number - right.page_number)
        }

        const next = previous.slice()
        next[index] = { ...next[index], ...patch }
        return next
      })
    },
    [],
  )

  const syncStoredPages = useCallback(async () => {
    const effectiveSkipVideo = skipVideoRef.current

    try {
      const freshPages = await getCoursewarePages(coursewareId)

      if (!mountedRef.current) return

      setPageStates(previous =>
        mergeAssemblyPagesFromStoredCourseware(
          freshPages,
          previous,
          effectiveSkipVideo,
        ),
      )
    } catch {
      // 页面同步失败不改变数据库运行状态。
    }
  }, [coursewareId])

  const shouldAcceptTerminal = useCallback(
    (state: CoursewareAssemblyState) =>
      shouldAcceptAssemblyTerminal(
        state,
        {
          observedRunVersion:
            observedRunVersionRef.current,
          localStart:
            localStartRef.current,
          preRunObserved:
            preRunObservedRef.current,
          baselineKnown:
            baselineKnownRef.current,
          baselineVersion:
            baselineVersionRef.current,
          sseDone:
            sseDoneRef.current,
        },
      ),
    [],
  )

  const applyRuntimeState = useCallback(
    (state: CoursewareAssemblyState) => {
      const runtime = state.runtime_status
      let effectiveRuntime = runtime

      if (state.is_active) {
        skipVideoRef.current = state.skip_video

        if (runtime === 'starting') {
          preRunObservedRef.current = true

          // 启动票据阶段尚未落库cancel_requested。
          // 用户已经发出取消时，前端继续展示“正在停止”，避免短暂跳回“正在启动”。
          if (cancelRequestedRef.current) {
            effectiveRuntime = 'cancel_requested'
          }
        }

        if (
          state.active_run_id &&
          (
            runtime === 'running' ||
            runtime === 'cancel_requested'
          )
        ) {
          observedRunVersionRef.current = state.assembly_version
          preRunObservedRef.current = false
        }

        // 取消请求报错后若数据库明确进入running，说明启动前取消没有生效。
        // 恢复真实运行态并允许老师再次点击停止。
        if (
          cancelRequestedRef.current &&
          runtime === 'running'
        ) {
          cancelRequestedRef.current = false
        }

        setStarted(true)
        setStarting(effectiveRuntime === 'starting')
        setSummary(current => ({
          ...current,
          skip_video: state.skip_video,
          running: true,
          done: false,
          runtime_status: effectiveRuntime,
          restored:
            current.runtime_status === 'idle' ||
            Boolean(current.restored),
          message: getAssemblyRuntimeMessage(effectiveRuntime),
        }))

        void syncStoredPages()
        return
      }

      // 启动前取消不会创建新的数据库运行。
      //
      // 此时数据库可能仍保留上一次completed/failed等历史终态，不能把它
      // 当成本次结果；只要版本没有推进到本次运行，就应收敛为“启动前已取消”。
      if (
        cancelRequestedRef.current &&
        (
          !isAssemblyTerminalRuntime(runtime) ||
          !shouldAcceptTerminal(state)
        )
      ) {
        cancelRequestedRef.current = false
        localStartRef.current = false
        preRunObservedRef.current = false
        setStarted(true)
        setStarting(false)
        setCancelling(false)
        setSummary(current => ({
          ...current,
          running: false,
          done: true,
          runtime_status: 'cancelled',
          message: '装配已在正式运行创建前取消。',
        }))
        return
      }

      if (isAssemblyTerminalRuntime(runtime)) {
        if (!shouldAcceptTerminal(state)) {
          if (preRunObservedRef.current) {
            preRunObservedRef.current = false
            setStarting(false)
            setStarted(true)
            setSummary(current => ({
              ...current,
              running: false,
              done: true,
              runtime_status: 'failed',
              message:
                '装配未能建立数据库运行，请检查前置条件后重新开始。',
            }))
          }

          // 没观察到本次启动时，历史终态与当前面板无关。
          return
        }

        cancelRequestedRef.current = false
        localStartRef.current = false
        preRunObservedRef.current = false
        setStarted(true)
        setStarting(false)
        setCancelling(false)
        setSummary(current => ({
          ...current,
          skip_video: state.skip_video,
          running: false,
          done: true,
          runtime_status: runtime,
          message: getAssemblyRuntimeMessage(runtime),
        }))

        void syncStoredPages()

        if (
          runtime === 'completed' &&
          !completionNotifiedRef.current
        ) {
          completionNotifiedRef.current = true
          window.setTimeout(() => {
            if (mountedRef.current) {
              onDoneRef.current()
            }
          }, 150)
        }

        return
      }

      if (
        runtime === 'idle' &&
        preRunObservedRef.current
      ) {
        localStartRef.current = false
        preRunObservedRef.current = false
        setStarting(false)
        setStarted(true)
        setSummary(current => ({
          ...current,
          running: false,
          done: true,
          runtime_status: 'failed',
          message:
            '装配未能建立数据库运行，请检查前置条件后重新开始。',
        }))
      }
    },
    [shouldAcceptTerminal, syncStoredPages],
  )

  const refreshAssemblyState = useCallback(async () => {
    try {
      const state = await getCoursewareAssemblyState(coursewareId)

      if (!mountedRef.current) {
        return null
      }

      applyRuntimeState(state)
      return state
    } catch {
      if (mountedRef.current) {
        setSummary(current => ({
          ...current,
          message:
            current.message ||
            '暂时无法读取后台装配状态，系统会继续自动重试。',
        }))
      }

      return null
    }
  }, [applyRuntimeState, coursewareId])

  const applyMediaStage = useCallback(
    (
      pageNumber: number,
      stage: string,
      note: string,
    ) => {
      patchPage(
        pageNumber,
        createAssemblyMediaStagePatch(
          stage,
          note,
        ),
      )
    },
    [patchPage],
  )

  useEffect(() => {
    if (initialState?.is_active) {
      applyRuntimeState(initialState)
      return
    }

    void refreshAssemblyState()
  }, [
    applyRuntimeState,
    initialState,
    refreshAssemblyState,
  ])

  useEffect(() => {
    if (!started || summary.done) {
      return
    }

    sseRef.current?.close()

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

          setPageStates(() => {
            const pages: AssemblyPageState[] = []

            for (
              let pageNumber = 1;
              pageNumber <= data.total_pages;
              pageNumber++
            ) {
              pages.push(
                createBlankAssemblyPage(
                  pageNumber,
                  `第 ${pageNumber} 页`,
                  data.skip_video,
                ),
              )
            }

            return pages
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
          const image: AssemblyStageState = data.image_ok
            ? 'ok'
            : data.image_skipped
              ? 'skipped'
              : 'failed'
          const video: AssemblyStageState = skipVideoRef.current
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
            const state = await refreshAssemblyState()

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

    sseRef.current = subscription

    return () => {
      subscription.close()

      if (sseRef.current === subscription) {
        sseRef.current = null
      }
    }
  }, [
    applyMediaStage,
    coursewareId,
    patchPage,
    refreshAssemblyState,
    shouldAcceptTerminal,
    started,
    summary.done,
    syncStoredPages,
  ])

  useEffect(() => {
    if (!started || summary.done) {
      return
    }

    const timer = window.setInterval(() => {
      void refreshAssemblyState()
    }, 3000)

    return () => {
      window.clearInterval(timer)
    }
  }, [
    refreshAssemblyState,
    started,
    summary.done,
  ])

  const runAssembly = useCallback(async () => {
    if (
      launchingRef.current ||
      starting ||
      summary.running
    ) {
      return
    }

    // React状态更新不是同步锁。
    //
    // 先用ref抢占本次启动，避免读取baseline版本与提交HTTP期间重复点击。
    launchingRef.current = true

    cancelRequestedRef.current = false
    completionNotifiedRef.current = false
    observedRunVersionRef.current = 0
    sseDoneRef.current = false
    localStartRef.current = false
    preRunObservedRef.current = false
    skipVideoRef.current = skipVideo

    // 启动HTTP成功返回前不开放“停止装配”按钮。
    //
    // 后端只有在接收auto-assemble请求后才会创建精确启动票据；
    // 提前开放取消会产生“取消先到、票据尚不存在、随后仍启动”的竞态。
    setStarted(false)
    setStarting(true)
    setCancelling(false)
    setSummary(current => ({
      ...current,
      skip_video: skipVideo,
      running: false,
      done: false,
      runtime_status: 'starting',
      message:
        '正在确认后台状态并提交装配任务…',
    }))

    try {
      // 启动前必须拿到数据库版本基线。
      //
      // 若此查询失败就不提交启动请求，避免无法区分“本次终态”和历史终态。
      const baseline =
        await getCoursewareAssemblyState(
          coursewareId,
        )

      // 另一个标签页可能已经先启动了装配。
      //
      // 直接接管数据库中的当前运行，不再提交第二次启动请求。
      if (baseline.is_active) {
        baselineVersionRef.current =
          baseline.assembly_version
        baselineKnownRef.current = true
        preRunObservedRef.current =
          baseline.runtime_status ===
          'starting'
        applyRuntimeState(baseline)
        return
      }

      baselineVersionRef.current =
        baseline.assembly_version
      baselineKnownRef.current = true
      localStartRef.current = true

      // HTTP成功只表示Tracker已登记任务；此时后端精确启动票据已经存在，
      // 页面才进入可取消的starting状态。
      await autoAssemble(
        coursewareId,
        skipVideo,
      )

      if (!mountedRef.current) {
        return
      }

      setPageStates(
        createAssemblyPagesFromStoredCourseware(
          courseware.pages || [],
          skipVideo,
        ),
      )
      setStarted(true)
      setStarting(true)
      setSummary({
        total_pages:
          courseware.page_count ||
          courseware.pages?.length ||
          0,
        skip_video: skipVideo,
        running: true,
        done: false,
        runtime_status: 'starting',
        message:
          '装配任务已登记，正在建立数据库运行…',
      })

      // 真实starting/running状态继续以数据库查询为准。
      await refreshAssemblyState()
    } catch (error) {
      localStartRef.current = false
      preRunObservedRef.current = false

      if (!mountedRef.current) {
        return
      }

      setStarted(true)
      setStarting(false)
      setSummary(current => ({
        ...current,
        running: false,
        done: true,
        runtime_status: 'failed',
        message:
          '❌ 启动失败：' +
          (
            error instanceof Error
              ? error.message
              : '未知错误'
          ),
      }))
    } finally {
      launchingRef.current = false
    }
  }, [
    applyRuntimeState,
    courseware.page_count,
    courseware.pages,
    coursewareId,
    refreshAssemblyState,
    skipVideo,
    starting,
    summary.running,
  ])

  const cancelAssembly = useCallback(async () => {
    if (
      cancelling ||
      summary.runtime_status === 'cancel_requested'
    ) {
      return
    }

    if (
      !window.confirm(
        '确定停止自动装配？已完成页面会保留，尚未落库的工作会停止继续写入。',
      )
    ) {
      return
    }

    cancelRequestedRef.current = true
    setCancelling(true)
    setSummary(current => ({
      ...current,
      running: true,
      done: false,
      runtime_status: 'cancel_requested',
      message: '正在发送停止信号…',
    }))

    try {
      const result =
        await cancelCoursewareAutoAssembly(coursewareId)

      setSummary(current => ({
        ...current,
        message: result.message,
      }))

      window.setTimeout(() => {
        void refreshAssemblyState()
      }, 300)
    } catch (error) {
      const failureMessage =
        '❌ 停止失败：' +
        (
          error instanceof Error
            ? error.message
            : '未知错误'
        )

      // 请求报错不等于服务端一定没有收到。
      //
      // 先保留cancelRequestedRef读取一次数据库：
      //   - 非active且版本未推进：按启动前取消收敛；
      //   - starting：继续等待票据被消费；
      //   - cancel_requested：服务端实际已受理；
      //   - running：取消未生效，applyRuntimeState会恢复真实运行态并允许重试。
      const refreshed =
        await refreshAssemblyState()

      if (!mountedRef.current) {
        return
      }

      if (!refreshed) {
        setSummary(current => ({
          ...current,
          running: true,
          done: false,
          runtime_status: 'cancel_requested',
          message:
            `${failureMessage} 系统会继续确认后台状态。`,
        }))
        return
      }

      if (
        refreshed.is_active &&
        refreshed.runtime_status ===
          'starting'
      ) {
        setSummary(current => ({
          ...current,
          message:
            '停止请求状态暂未确认，系统会继续检查；请勿重复启动。',
        }))
        return
      }

      if (
        refreshed.is_active &&
        refreshed.runtime_status ===
          'cancel_requested'
      ) {
        setSummary(current => ({
          ...current,
          message:
            '停止请求已进入后台，正在等待当前工作收敛。',
        }))
        return
      }

      if (refreshed.is_active) {
        setSummary(current => ({
          ...current,
          message: failureMessage,
        }))
      }
    } finally {
      if (mountedRef.current) {
        setCancelling(false)
      }
    }
  }, [
    cancelling,
    coursewareId,
    refreshAssemblyState,
    summary.runtime_status,
  ])

  return {
    started,
    starting,
    cancelling,
    pageStates,
    summary,
    runAssembly,
    cancelAssembly,
    refreshAssemblyState,
    syncStoredPages,
  }
}
