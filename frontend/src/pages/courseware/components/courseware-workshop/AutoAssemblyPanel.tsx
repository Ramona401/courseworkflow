/**
 * AutoAssemblyPanel.tsx — 全自动装配「自包含容器组件」
 *
 * 职责边界（把装配的全部复杂度锁在本组件内，主页面只需渲染它 + 传几个 props）：
 *   1. 启动确认：
 *        · 【已设锚点】点"开始" → DangerConfirmModal 单次确认 → 跑装配；
 *        · 【未设锚点】点"开始" → 直接弹 AnchorStylePicker 画风窗（选画风→生成定调图→设锚点→
 *          在画风窗里【看预览、确认满意】→ 点"就用这个"即唯一确认）→ 跑装配。
 *      —— 两条路径都恰好一次确认，不重复。
 *   2. 触发：确认后调 autoAssemble(coursewareId, skipVideo) 异步启动后端总装线；
 *   3. 订阅：自持一条 subscribeCWIndexSSE，挂 6 个 assembly_* 回调，把事件翻译成每页三态数组 + 汇总；
 *   4. 展示：把三态数组 + 汇总喂给纯展示组件 AssemblyProgressView 渲染页面网格进度；
 *   5. 收尾：收到 assembly_done 后回调父组件 onDone（父组件据此 loadCourseware + 跳 Step5）。
 *
 * 【本轮体验修复 · 画风弹窗消失/预览跳过 根治】
 *   根因：旧版画风弹窗设锚点成功后，onAnchorChanged 触发父页面整页 loadCourseware，
 *   把本组件（连同弹窗）卸载重挂，state 全丢，回到起点。
 *   本组件配合修复：把 onAnchorChanged 的签名改为【携带 SetStyleAnchorResult】透传给上层，
 *   上层（主页面）据此做 setCourseware 乐观更新，【不再整页刷新】，弹窗稳定停留、可预览可确认，
 *   确认后 handleStyleConfirmed → runAssembly 一气呵成进入进度视图。
 *
 * 与主页面现有「纯手动批量生成」(handleBuildStart) 物理隔离：各自独立的 SSE 句柄与 state，
 *   互不干扰。交付模式选择器决定渲染谁——纯手动走主页面老逻辑，全自动/中间档走本组件。
 *
 * skipVideo 由父组件（交付模式选择器）决定：
 *   false = 全自动装配（HTML+配图+视频占位）；true = HTML+配图不做视频（中间档）。
 *
 * 前置约束（后端 prepareAssembly 强校验：已确认导航栏 + 已设风格锚点）：
 *   导航栏由前几步保证；风格锚点由本组件"未设锚点→画风窗当场设"兜住。若装配启动后后端仍因
 *   约束不满足推 error 事件，onError 会把消息显示出来。
 */
import { useState, useRef, useEffect, useCallback } from 'react'
import { subscribeCWIndexSSE, autoAssemble } from '@/api/coursewares'
import type { CoursewareDetail, SetStyleAnchorResult } from '@/api/coursewares'
import { C } from './workshopConstants'
import DangerConfirmModal from './DangerConfirmModal'
import AnchorStylePicker from './AnchorStylePicker'
import AssemblyProgressView, {
  type AssemblyPageState, type AssemblySummary, type AssemblyStageState,
} from './AssemblyProgressView'

interface Props {
  coursewareId: string
  courseware: CoursewareDetail
  /** 交付模式：true=HTML+配图不做视频（中间档），false=全自动含视频占位 */
  skipVideo: boolean
  /** 装配完成回调：父组件据此 loadCourseware + 跳 Step5 */
  onDone: () => void
  /** 返回上一步（重选交付模式）；装配运行中隐藏 */
  onBack: () => void
  /**
   * 画风弹窗设锚点成功后回传结果给父级做「乐观更新」。
   * 【关键】父级必须用 setCourseware 局部更新 style_anchor_* 三字段，绝不能整页 loadCourseware——
   * 否则本组件连同画风弹窗被卸载重挂，state 全丢，回到起点（这正是"弹窗莫名消失"的根因）。
   */
  onAnchorChanged?: (res: SetStyleAnchorResult) => void
}

// 空白单页三态（新建某页卡片时的初值）
function blankPage(page_number: number, title: string): AssemblyPageState {
  return { page_number, title, html: 'pending', image: 'pending', video: 'pending' }
}

export default function AutoAssemblyPanel({ coursewareId, courseware, skipVideo, onDone, onBack, onAnchorChanged }: Props) {
  // 二次确认弹窗显隐（仅"已设锚点"路径用；未设锚点直接走画风窗，不弹此窗）
  const [showConfirm, setShowConfirm] = useState(false)
  // 画风选择弹窗显隐（未设锚点时点"开始"直接弹此窗，选画风→设锚点→看预览确认→装配一气呵成）
  const [showStylePicker, setShowStylePicker] = useState(false)
  // 是否已启动装配（启动后隐藏"开始"按钮、显示进度视图）
  const [started, setStarted] = useState(false)
  // 启动请求进行中（防重复点击）
  const [starting, setStarting] = useState(false)

  // 每页三态数组（按 page_number 升序维护，唯一真相源）
  const [pageStates, setPageStates] = useState<AssemblyPageState[]>([])
  // 装配汇总
  const [summary, setSummary] = useState<AssemblySummary>({
    total_pages: courseware.page_count || 0,
    skip_video: skipVideo,
    running: false,
    done: false,
    message: '',
  })

  // 本组件独立的 SSE 句柄（与主页面 sseRef 隔离），卸载时关闭
  const sseRef = useRef<{ close: () => void } | null>(null)
  useEffect(() => () => { sseRef.current?.close() }, [])

  // ---- 小工具：更新某页某条流水线状态 ----
  const patchPage = useCallback((pageNum: number, patch: Partial<AssemblyPageState>) => {
    setPageStates(prev => {
      const idx = prev.findIndex(p => p.page_number === pageNum)
      if (idx === -1) {
        // 该页卡片尚未建立（assembly_start 时会按 total 预建；此处兜底）
        const np = { ...blankPage(pageNum, patch.title || `第 ${pageNum} 页`), ...patch }
        return [...prev, np].sort((a, b) => a.page_number - b.page_number)
      }
      const next = prev.slice()
      next[idx] = { ...next[idx], ...patch }
      return next
    })
  }, [])

  // ---- 阶段文案 → 该页各链状态推断（配图/视频进度事件 stage 驱动）----
  //   后端 assembly_page_image 的 stage: image_prompt/image_gen/image_fuse
  //   后端 assembly_page_video 的 stage: video_storyboard
  const applyMediaStage = useCallback((pageNum: number, stage: string, note: string) => {
    if (stage === 'video_storyboard') {
      patchPage(pageNum, { video: 'running', note })
    } else {
      // image_prompt / image_gen / image_fuse 都归为配图进行中
      patchPage(pageNum, { image: 'running', note })
    }
  }, [patchPage])

  // ---- 真正执行装配（锚点已就绪后调用）----
  const runAssembly = useCallback(async () => {
    setShowConfirm(false)
    setShowStylePicker(false)
    if (starting || started) return
    setStarting(true)
    setStarted(true)
    setSummary(s => ({ ...s, running: true, message: '正在启动装配…' }))
    try {
      await autoAssemble(coursewareId, skipVideo)
      sseRef.current?.close()
      sseRef.current = subscribeCWIndexSSE(coursewareId, {
        onConnected: () => setSummary(s => ({ ...s, message: '已连接，装配即将开始…' })),

        // 装配开始：按 total 预建全部页卡片（title 暂用占位，page_html 到达时补真实标题）
        onAssemblyStart: d => {
          setSummary(s => ({
            ...s,
            total_pages: d.total_pages,
            skip_video: d.skip_video,
            running: true,
            done: false,
            message: d.message,
          }))
          // 预建 1..total 的空白卡片，让老师一开始就看到全貌
          setPageStates(() => {
            const arr: AssemblyPageState[] = []
            for (let i = 1; i <= d.total_pages; i++) arr.push(blankPage(i, `第 ${i} 页`))
            return arr
          })
        },

        // 某页 HTML 生成完成落库
        onAssemblyPageHtml: d => {
          patchPage(d.page_number, { title: d.title, html: 'ok', image: 'running', note: '正在配图…' })
        },

        // 某页 HTML 生成失败（该页不再配图）
        onAssemblyProgress: d => {
          patchPage(d.page_number, {
            title: d.page_title || undefined,
            html: 'failed',
            image: 'skipped',
            video: 'skipped',
            note: d.error ? `HTML生成失败：${d.error}` : 'HTML生成失败',
          })
        },

        // 配图/视频阶段进行中（仅更新文案 + 对应链置 running）
        onAssemblyPageMedia: d => {
          applyMediaStage(d.page_number, d.stage, d.message)
        },

        // 某页装配完成：定格配图/视频最终状态
        onAssemblyPageDone: d => {
          const image: AssemblyStageState = d.image_ok ? 'ok' : d.image_skipped ? 'skipped' : 'failed'
          const video: AssemblyStageState = skipVideo
            ? 'skipped'
            : d.video_ok ? 'ok' : d.video_skipped ? 'skipped' : 'pending'
          patchPage(d.page_number, {
            title: d.title || undefined,
            // HTML 能走到 page_done 说明已就绪（失败页不会到这），置 ok 兜底
            html: 'ok',
            image,
            video,
            note: undefined,
          })
        },

        // 全部装配完成：定格汇总，回调父组件
        onAssemblyDone: d => {
          setSummary(s => ({
            ...s,
            running: false,
            done: true,
            message: d.message,
            html_success: d.html_success,
            html_fail: d.html_fail,
            image_success: d.image_success,
            image_fail: d.image_fail,
            image_skip: d.image_skip,
            video_success: d.video_success,
            video_skip: d.video_skip,
            elapsed_ms: d.elapsed_ms,
            errors: d.errors,
          }))
          setStarting(false)
          // 通知父组件刷新课件（装配已把页面 HTML+配图落库，父组件 loadCourseware 恢复真实数据）
          onDone()
        },

        // 业务级 error（如后端前置约束不满足）：显示消息，结束运行态
        onError: d => {
          setSummary(s => ({ ...s, running: false, message: `❌ ${d.message}` }))
          setStarting(false)
        },

        // 断线重连成功：提示（装配在后台继续；本视图靠后续事件补齐，最终以 onDone 的 loadCourseware 为准）
        onReconnected: () => {
          setSummary(s => ({ ...s, message: '🔄 连接已恢复，装配仍在后台继续…' }))
        },
      })
    } catch (e) {
      setSummary(s => ({ ...s, running: false, message: '❌ 启动失败：' + (e instanceof Error ? e.message : '未知错误') }))
      setStarting(false)
      setStarted(false)
    }
  }, [coursewareId, skipVideo, starting, started, patchPage, applyMediaStage, onDone])

  const hasAnchor = !!courseware.style_anchor_asset_id

  // ---- 点"开始"入口 ----
  //   · 已设锚点：弹 DangerConfirmModal 单次确认（无画风窗兜底，需一次确认）；
  //   · 未设锚点：直接弹画风窗（画风窗里看预览、点"就用这个"即唯一确认，确认后直接跑），不再先弹确认弹窗。
  const handleStartClick = useCallback(() => {
    if (hasAnchor) {
      setShowConfirm(true)
    } else {
      setShowStylePicker(true)
    }
  }, [hasAnchor])

  // 画风窗里老师点"就用这个，开始装配"——锚点此刻已设好，直接一气呵成装配
  const handleStyleConfirmed = useCallback(() => {
    setShowStylePicker(false)
    runAssembly()
  }, [runAssembly])

  // ==================== 渲染 ====================

  const modeLabel = skipVideo ? 'HTML + 配图（不做视频）' : '全自动装配（HTML + 配图 + 视频占位）'
  const estimateHint = skipVideo
    ? '每页需 HTML 生成 + 配图（写提示词→生图→上云→融图）约 3–4 次 AI 调用。'
    : '每页需 HTML 生成 + 配图 + 视频首帧占位，命中视频的页 AI 调用更多。'

  // 未设锚点时，"开始"按钮文案改为"选画风并开始"，让老师知道下一步是选画风
  const startBtnLabel = hasAnchor
    ? (skipVideo ? '🖼 开始装配（HTML + 配图）' : '⚡ 开始全自动装配')
    : (skipVideo ? '🎨 选画风并开始装配' : '🎨 选画风并开始全自动装配')

  return (
    <div>
      {/* 未启动：说明卡 + 开始按钮 */}
      {!started && (
        <div>
          <div style={{
            padding: '16px 18px', borderRadius: 12, marginBottom: 16,
            background: skipVideo ? '#F0FDFF' : '#FaF5FF',
            border: `1px solid ${skipVideo ? '#A5F3FC' : '#E9D5FF'}`,
          }}>
            <div style={{ fontSize: 15, fontWeight: 700, color: C.textPrimary, marginBottom: 8 }}>
              {skipVideo ? '🖼' : '⚡'} {modeLabel}
            </div>
            <div style={{ fontSize: 13, color: C.textSecondary, lineHeight: 1.7 }}>
              一键把剩余页全部生成并自动配图{skipVideo ? '' : '、为含视频/动画的页生成视频首帧占位'}，
              配图会按统一画风自动生成，保持全课件视觉一致。<br />
              {!hasAnchor && (
                <span style={{ color: '#7C3AED' }}>
                  ⭐ 点下方按钮会先让你选一个插图画风，系统生成一张风格样板给你确认，满意后即开始。<br />
                </span>
              )}
              <span style={{ color: C.textMuted }}>⏳ 这是重操作，{estimateHint}整套流程耗时较长、消耗较多积分，请在网络稳定时运行。</span>
            </div>
          </div>

          <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            <button
              onClick={handleStartClick}
              style={{
                padding: '14px 36px', borderRadius: 10, border: 'none', color: '#fff',
                fontSize: 16, fontWeight: 600, cursor: 'pointer',
                background: skipVideo ? 'linear-gradient(135deg, #0891B2, #06B6D4)' : 'linear-gradient(135deg, #7C3AED, #6366F1)',
                boxShadow: skipVideo ? '0 4px 16px rgba(8,145,178,0.3)' : '0 4px 16px rgba(124,58,237,0.3)',
              }}
            >
              {startBtnLabel}
            </button>
            <button
              onClick={onBack}
              style={{
                padding: '14px 24px', borderRadius: 10, border: `1px solid ${C.border}`,
                background: 'transparent', color: C.textSecondary, fontSize: 14, cursor: 'pointer',
              }}
            >← 重选交付模式</button>
          </div>
        </div>
      )}

      {/* 已启动：进度视图 */}
      {started && (
        <div>
          <AssemblyProgressView summary={summary} pages={pageStates} />

          {/* 完成后：进入工作台按钮（onDone 已触发父组件 loadCourseware，此处再给个显式入口） */}
          {summary.done && (
            <div style={{ marginTop: 18, display: 'flex', gap: 12, flexWrap: 'wrap' }}>
              <button
                onClick={onDone}
                style={{
                  padding: '12px 28px', borderRadius: 10, border: 'none',
                  background: 'linear-gradient(135deg, #059669, #10B981)', color: '#fff',
                  fontSize: 15, fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 8px rgba(5,150,105,0.3)',
                }}
              >进入工作台预览与微调 →</button>
            </div>
          )}
        </div>
      )}

      {/* 二次确认弹窗（仅"已设锚点"路径用；复用现有 DangerConfirmModal） */}
      {showConfirm && (
        <DangerConfirmModal
          title={skipVideo ? '🖼 确认开始装配' : '⚡ 确认全自动装配'}
          message={[
            `即将对《${courseware.title}》执行${modeLabel}。`,
            '',
            `· 共约 ${summary.total_pages} 页，逐页生成 HTML 并自动配图${skipVideo ? '' : '、生成视频首帧占位'}`,
            '· 这是重操作：每页多次 AI 调用，总耗时可能达数分钟到十几分钟',
            '· 将消耗较多积分，且启动后不建议中途关闭页面',
            '',
            '确认现在开始吗？',
          ].join('\n')}
          confirmText={starting ? '启动中…' : '确认开始'}
          busy={starting}
          onConfirm={runAssembly}
          onCancel={() => setShowConfirm(false)}
        />
      )}

      {/* 画风选择弹窗：未设锚点时点"开始"直接弹出，选画风→生成定调图→设锚点→【看预览确认】→一气呵成装配。
          画风窗里的"就用这个，开始装配"即唯一确认，无需外层再套 DangerConfirmModal。
          onAnchorChanged 携带 SetStyleAnchorResult 透传给上层做乐观更新，绝不整页刷新，弹窗稳定不消失。 */}
      {showStylePicker && (
        <AnchorStylePicker
          coursewareId={coursewareId}
          skipVideo={skipVideo}
          onConfirmed={handleStyleConfirmed}
          onCancel={() => setShowStylePicker(false)}
          onAnchorChanged={(res) => { onAnchorChanged?.(res) }}
        />
      )}
    </div>
  )
}
