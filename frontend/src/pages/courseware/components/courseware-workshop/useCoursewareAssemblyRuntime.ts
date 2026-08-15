/**
 * useCoursewareAssemblyRuntime.ts — 自动装配前端生命周期控制器
 *
 * 集中处理启动、刷新恢复、数据库轮询、页面同步和显式取消。
 * SSE订阅与事件映射拆到useCoursewareAssemblySSE，避免本Hook继续膨胀。
 *
 * 运行身份原则：
 *   - 路由恢复门注入当前active assembly、R-04未完整终态或可智能补配终态；
 *   - 本地新启动先记录启动前 assembly_version；
 *   - 终态只有在观察到当前数据库运行，或版本号相对启动前递增后才接受；
 *   - 历史 completed/failed/cancelled/interrupted 不会冒充本次运行。
 */
import { useCallback, useEffect, useRef, useState } from 'react'

import { autoAssemble, getCoursewarePages } from '@/api/coursewares'
import type { CoursewareDetail } from '@/api/coursewares'

import { getCoursewareAssemblyState } from '@/api/coursewareAssembly'
import type { CoursewareAssemblyState } from '@/api/coursewareAssembly'

import type { AssemblyPageState, AssemblySummary } from './AssemblyProgressView'

import {
  createAssemblyMediaStagePatch,
  createAssemblyPagesForLaunch,
  getAssemblyRuntimeMessage,
  isAssemblyTerminalRuntime,
  mergeAssemblyPagesFromStoredCourseware,
  patchAssemblyPageState,
  shouldAcceptAssemblyTerminal,
  shouldRecoverIncompleteAssemblyState,
} from './coursewareAssemblyRuntimeState'
import { resolveCoursewareAssemblyCancellation } from './coursewareAssemblyCancellation'
import { useCoursewareAssemblySSE } from './useCoursewareAssemblySSE'

interface Options {
  coursewareId: string
  courseware: CoursewareDetail
  skipVideo: boolean
  onDone: () => void

  /** 路由恢复门只传当前active、R-04未完整终态或可智能补配终态。 */
  initialState?: CoursewareAssemblyState | null
}

export interface CoursewareAssemblyRuntimeController {
  started: boolean
  starting: boolean
  cancelling: boolean
  pageStates: AssemblyPageState[]
  summary: AssemblySummary
  assemblyState: CoursewareAssemblyState | null
  runAssembly: () => Promise<void>
  repairFailedImages: () => Promise<void>
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
  const initialActive = Boolean(
    initialState?.is_active &&
    initialState.run_kind === 'assembly',
  )
  const initialRecoverableTerminal =
    shouldRecoverIncompleteAssemblyState(
      initialState,
    )
  const initialResumable =
    initialActive ||
    initialRecoverableTerminal
  const initialSkipVideo = initialState?.skip_video ?? skipVideo
  const initialRuntime = initialState?.runtime_status ?? 'idle'

  const [started, setStarted] = useState(initialResumable)
  const [starting, setStarting] = useState(
    initialRuntime === 'starting',
  )
  const [cancelling, setCancelling] = useState(false)
  const [pageStates, setPageStates] = useState<AssemblyPageState[]>(
    () =>
      createAssemblyPagesForLaunch(
        courseware.pages || [],
        [],
        initialSkipVideo,
        initialState,
      ),
  )
  const [summary, setSummary] = useState<AssemblySummary>({
    total_pages:
      courseware.page_count ||
      courseware.pages?.length ||
      0,
    skip_video: initialSkipVideo,
    running: initialActive,
    done: initialRecoverableTerminal,
    runtime_status: initialResumable ? initialRuntime : 'idle',
    message: initialResumable
      ? getAssemblyRuntimeMessage(
          initialRuntime,
          Boolean(initialState?.repair_failed_images),
        )
      : '',
    restored: initialResumable,
  })
  const [assemblyState, setAssemblyState] =
    useState<CoursewareAssemblyState | null>(
      initialState ?? null,
    )

  const mountedRef = useRef(true)
  const cancelRequestedRef = useRef(false)
  const completionNotifiedRef = useRef(false)
  const launchingRef = useRef(false)
  const launchBaselineRef = useRef<CoursewareAssemblyState | null>(
    initialState ?? null,
  )
  const skipVideoRef = useRef(initialSkipVideo)
  const onDoneRef = useRef(onDone)

  // observedRunVersion只记录已落库运行；starting票据不能冒充数据库版本。
  const observedRunVersionRef = useRef(
    initialState &&
      (
        initialActive ||
        initialRecoverableTerminal
      )
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
    }
  }, [])

  const patchPage = useCallback(
    (
      pageNumber: number,
      patch: Partial<AssemblyPageState>,
    ) => {
      setPageStates(previous =>
        patchAssemblyPageState(
          previous,
          pageNumber,
          patch,
          skipVideoRef.current,
        ),
      )
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
      setAssemblyState(state)

      // assembly-state 同时承载普通批量与自动装配。
      // 自动装配Hook只消费assembly运行，避免把batch生命周期误画成装配进度。
      if (state.run_kind === 'batch') {
        if (state.is_active) {
          localStartRef.current = false
          preRunObservedRef.current = false
          setStarted(true)
          setStarting(false)
          setCancelling(false)
          setSummary(current => ({
            ...current,
            running: false,
            done: true,
            runtime_status: 'failed',
            message:
              '普通批量页面生成正在后台运行，请先等待完成或在普通生成入口停止后再启动自动装配。',
          }))
        }
        return
      }

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
          message: getAssemblyRuntimeMessage(
            effectiveRuntime,
            state.repair_failed_images,
          ),
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
          message: getAssemblyRuntimeMessage(
            runtime,
            state.repair_failed_images,
          ),
        }))

        void syncStoredPages()

        if (
          runtime === 'completed' &&
          state.integrity?.complete === true &&
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
    if (initialState && (initialActive || initialRecoverableTerminal)) {
      applyRuntimeState(initialState)
      return
    }

    void refreshAssemblyState()
  }, [
    applyRuntimeState,
    initialActive,
    initialRecoverableTerminal,
    initialState,
    refreshAssemblyState,
  ])

  useCoursewareAssemblySSE({
    coursewareId,
    started,
    done: summary.done,
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
  })

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

  const launchAssembly = useCallback(
    async (repairFailedImages: boolean) => {
      if (
        launchingRef.current ||
        starting ||
        summary.running
      ) {
        return
      }

      // React状态更新不是同步锁，先用ref抢占启动窗口。
      launchingRef.current = true

      cancelRequestedRef.current = false
      completionNotifiedRef.current = false
      observedRunVersionRef.current = 0
      sseDoneRef.current = false
      localStartRef.current = false
      preRunObservedRef.current = false
      launchBaselineRef.current = null
      skipVideoRef.current = skipVideo

      // 普通装配启动时暂时退回“尚未开始”视图；图片补配保持当前失败现场，
      // 避免老师点击后整页进度卡突然被初始画风界面替换。
      if (!repairFailedImages) {
        setStarted(false)
      }
      setStarting(true)
      setCancelling(false)
      setSummary(current => ({
        ...current,
        skip_video: skipVideo,
        running: false,
        done: false,
        runtime_status: 'starting',
        message: repairFailedImages
          ? '正在确认失败配图并提交智能补配任务…'
          : '正在确认后台状态并提交装配任务…',
      }))

      try {
        // 启动前必须拿到数据库版本基线；失败时不提交后台任务。
        const baseline =
          await getCoursewareAssemblyState(
            coursewareId,
          )
        launchBaselineRef.current = baseline

        // 另一个标签页可能已经先启动。直接接管当前assembly，
        // batch运行则拒绝混入自动装配生命周期。
        if (baseline.is_active) {
          baselineVersionRef.current =
            baseline.assembly_version
          baselineKnownRef.current = true

          if (baseline.run_kind !== 'assembly') {
            localStartRef.current = false
            setStarted(true)
            setStarting(false)
            setSummary(current => ({
              ...current,
              running: false,
              done: true,
              runtime_status: 'failed',
              message:
                '普通批量页面生成正在后台运行，请先等待完成或停止该任务后再启动自动装配。',
            }))
            return
          }

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

        // repair_failed_images只表达操作意图；真正失败范围由后端重新读取。
        await autoAssemble(
          coursewareId,
          skipVideo,
          repairFailedImages,
        )

        if (!mountedRef.current) {
          return
        }

        setPageStates(previous =>
          createAssemblyPagesForLaunch(
            courseware.pages || [],
            previous,
            skipVideo,
            baseline,
            repairFailedImages,
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
          message: repairFailedImages
            ? '智能补配任务已登记，正在建立数据库运行…'
            : '装配任务已登记，正在建立数据库运行…',
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
            (repairFailedImages
              ? '❌ 智能补配启动失败：'
              : '❌ 启动失败：') +
            (
              error instanceof Error
                ? error.message
                : '未知错误'
            ),
        }))
      } finally {
        launchingRef.current = false
      }
    },
    [
      applyRuntimeState,
      courseware.page_count,
      courseware.pages,
      coursewareId,
      refreshAssemblyState,
      skipVideo,
      starting,
      summary.running,
    ],
  )

  const runAssembly = useCallback(
    () => launchAssembly(false),
    [launchAssembly],
  )

  const repairFailedImages = useCallback(
    () => launchAssembly(true),
    [launchAssembly],
  )

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
      const resolution =
        await resolveCoursewareAssemblyCancellation(
          coursewareId,
          refreshAssemblyState,
        )

      if (
        mountedRef.current &&
        resolution.message
      ) {
        setSummary(current => ({
          ...current,
          message: resolution.message,
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
    assemblyState,
    runAssembly,
    repairFailedImages,
    cancelAssembly,
    refreshAssemblyState,
    syncStoredPages,
  }
}
