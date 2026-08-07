/**
 * AutoAssemblyPanel.tsx — 全自动装配视图容器
 *
 * 异步生命周期由 useCoursewareAssemblyRuntime 统一管理；
 * 本组件只负责画风入口、进度展示和操作按钮。
 */
import {
  useCallback,
  useState,
} from 'react'

import type {
  CoursewareDetail,
  SetStyleAnchorResult,
} from '@/api/coursewares'
import type {
  CoursewareAssemblyState,
} from '@/api/coursewareAssembly'

import { C } from './workshopConstants'
import AnchorStylePicker
  from './AnchorStylePicker'
import AssemblyProgressView
  from './AssemblyProgressView'
import {
  useCoursewareAssemblyRuntime,
} from './useCoursewareAssemblyRuntime'

interface Props {
  coursewareId: string
  courseware: CoursewareDetail
  skipVideo: boolean
  onDone: () => void
  onBack: () => void

  /** 路由恢复门传入的当前active运行。 */
  initialState?: CoursewareAssemblyState | null

  onAnchorChanged?: (
    result: SetStyleAnchorResult,
  ) => void
}

export default function AutoAssemblyPanel({
  coursewareId,
  courseware,
  skipVideo,
  onDone,
  onBack,
  initialState,
  onAnchorChanged,
}: Props) {
  const [showStylePicker, setShowStylePicker] =
    useState(false)

  const runtime =
    useCoursewareAssemblyRuntime({
      coursewareId,
      courseware,
      skipVideo,
      onDone,
      initialState,
    })

  const currentSkipVideo =
    runtime.summary.skip_video
  const hasAnchor = Boolean(
    courseware.style_anchor_asset_id,
  )

  const runAssembly = useCallback(() => {
    setShowStylePicker(false)
    void runtime.runAssembly()
  }, [runtime])

  const modeLabel = currentSkipVideo
    ? 'HTML + 配图（不做视频）'
    : '全自动装配（HTML + 配图 + 视频占位）'
  const estimateHint = currentSkipVideo
    ? '每页需要生成HTML并完成配图。'
    : '每页需要生成HTML、完成配图，并为命中视频需求的页面生成视频首帧占位。'
  const selectStyleButtonLabel = hasAnchor
    ? currentSkipVideo
      ? '🎨 重新选择画风并开始装配'
      : '🎨 重新选择画风并开始全自动装配'
    : currentSkipVideo
      ? '🎨 选择画风并开始装配'
      : '🎨 选择画风并开始全自动装配'

  return (
    <div>
      {!runtime.started && (
        <div>
          <div
            style={{
              marginBottom: 16,
              padding: '16px 18px',
              borderRadius: 12,
              border: `1px solid ${
                currentSkipVideo
                  ? '#A5F3FC'
                  : '#E9D5FF'
              }`,
              background: currentSkipVideo
                ? '#F0FDFF'
                : '#FAF5FF',
            }}
          >
            <div
              style={{
                marginBottom: 8,
                color: C.textPrimary,
                fontSize: 15,
                fontWeight: 700,
              }}
            >
              {currentSkipVideo
                ? '🖼'
                : '⚡'}{' '}
              {modeLabel}
            </div>

            <div
              style={{
                color: C.textSecondary,
                fontSize: 13,
                lineHeight: 1.7,
              }}
            >
              一键生成剩余页面并自动配图
              {currentSkipVideo
                ? ''
                : '，同时为含视频或动画需求的页面生成视频首帧占位'}
              。
              <br />
              <span
                style={{
                  color: '#7C3AED',
                }}
              >
                ⭐ 点击主按钮会打开真实画风缩略图选择弹窗，预设画风不现场调用图片AI。
              </span>
              <br />
              {hasAnchor && (
                <>
                  <span
                    style={{
                      color: '#059669',
                    }}
                  >
                    ✓ 当前课件已有画风锚点；可以重新选择，也可以沿用当前画风直接开始。
                  </span>
                  <br />
                </>
              )}
              <span
                style={{
                  color: C.textMuted,
                }}
              >
                ⏳ 这是重操作，{estimateHint}
                启动后可刷新页面，系统会从数据库恢复运行状态。
              </span>
            </div>
          </div>

          <div
            style={{
              display: 'flex',
              gap: 12,
              flexWrap: 'wrap',
            }}
          >
            <button
              type="button"
              disabled={runtime.starting}
              onClick={() => {
                if (!runtime.starting) {
                  setShowStylePicker(true)
                }
              }}
              style={{
                padding: '14px 32px',
                borderRadius: 10,
                border: 'none',
                background: currentSkipVideo
                  ? 'linear-gradient(135deg, #0891B2, #06B6D4)'
                  : 'linear-gradient(135deg, #7C3AED, #6366F1)',
                boxShadow: currentSkipVideo
                  ? '0 4px 16px rgba(8,145,178,0.3)'
                  : '0 4px 16px rgba(124,58,237,0.3)',
                color: '#fff',
                cursor: runtime.starting
                  ? 'not-allowed'
                  : 'pointer',
                fontSize: 16,
                fontWeight: 700,
                opacity: runtime.starting
                  ? 0.65
                  : 1,
              }}
            >
              {runtime.starting
                ? '正在启动…'
                : selectStyleButtonLabel}
            </button>

            {hasAnchor && (
              <button
                type="button"
                disabled={runtime.starting}
                onClick={runAssembly}
                style={{
                  padding: '14px 24px',
                  borderRadius: 10,
                  border:
                    '1px solid #7C3AED',
                  background: '#F5F3FF',
                  color: '#6D28D9',
                  cursor: runtime.starting
                    ? 'not-allowed'
                    : 'pointer',
                  fontSize: 14,
                  fontWeight: 700,
                  opacity: runtime.starting
                    ? 0.65
                    : 1,
                }}
              >
                {runtime.starting
                  ? '正在启动…'
                  : '沿用当前画风直接开始'}
              </button>
            )}

            <button
              type="button"
              disabled={runtime.starting}
              onClick={onBack}
              style={{
                padding: '14px 24px',
                borderRadius: 10,
                border:
                  `1px solid ${C.border}`,
                background: 'transparent',
                color: C.textSecondary,
                cursor: runtime.starting
                  ? 'not-allowed'
                  : 'pointer',
                fontSize: 14,
              }}
            >
              ← 重选交付模式
            </button>
          </div>
        </div>
      )}

      {runtime.started && (
        <div>
          <AssemblyProgressView
            summary={runtime.summary}
            pages={runtime.pageStates}
          />

          <div
            style={{
              marginTop: 18,
              display: 'flex',
              gap: 12,
              flexWrap: 'wrap',
            }}
          >
            {runtime.summary.running && (
              <button
                type="button"
                disabled={
                  runtime.cancelling ||
                  runtime.summary.runtime_status ===
                    'cancel_requested'
                }
                onClick={() => {
                  void runtime.cancelAssembly()
                }}
                style={{
                  padding: '10px 22px',
                  borderRadius: 9,
                  border:
                    '1px solid #EF4444',
                  background: '#FEF2F2',
                  color: '#DC2626',
                  cursor:
                    runtime.cancelling ||
                    runtime.summary.runtime_status ===
                      'cancel_requested'
                      ? 'not-allowed'
                      : 'pointer',
                  fontSize: 13,
                  fontWeight: 600,
                  opacity:
                    runtime.cancelling ||
                    runtime.summary.runtime_status ===
                      'cancel_requested'
                      ? 0.65
                      : 1,
                }}
              >
                {runtime.summary.runtime_status ===
                'cancel_requested'
                  ? '⏳ 正在停止…'
                  : runtime.cancelling
                    ? '正在发送停止信号…'
                    : '⏸ 停止装配'}
              </button>
            )}

            {runtime.summary.done &&
              runtime.summary.runtime_status ===
                'completed' && (
                <button
                  type="button"
                  onClick={onDone}
                  style={{
                    padding: '12px 28px',
                    borderRadius: 10,
                    border: 'none',
                    background:
                      'linear-gradient(135deg, #059669, #10B981)',
                    boxShadow:
                      '0 2px 8px rgba(5,150,105,0.3)',
                    color: '#fff',
                    cursor: 'pointer',
                    fontSize: 15,
                    fontWeight: 600,
                  }}
                >
                  进入工作台预览与微调 →
                </button>
              )}

            {runtime.summary.done &&
              runtime.summary.runtime_status !==
                'completed' && (
                <>
                  <button
                    type="button"
                    disabled={runtime.starting}
                    onClick={runAssembly}
                    style={{
                      padding: '10px 22px',
                      borderRadius: 9,
                      border: 'none',
                      background: currentSkipVideo
                        ? 'linear-gradient(135deg, #0891B2, #06B6D4)'
                        : 'linear-gradient(135deg, #7C3AED, #6366F1)',
                      color: '#fff',
                      cursor: runtime.starting
                        ? 'not-allowed'
                        : 'pointer',
                      fontSize: 13,
                      fontWeight: 700,
                      opacity: runtime.starting
                        ? 0.65
                        : 1,
                    }}
                  >
                    {runtime.starting
                      ? '正在重新启动…'
                      : currentSkipVideo
                        ? '▶️ 继续HTML+配图装配'
                        : '▶️ 继续全自动装配'}
                  </button>

                  <button
                    type="button"
                    disabled={runtime.starting}
                    onClick={onBack}
                    style={{
                      padding: '10px 22px',
                      borderRadius: 9,
                      border:
                        `1px solid ${C.border}`,
                      background: '#fff',
                      color: C.textSecondary,
                      cursor: runtime.starting
                        ? 'not-allowed'
                        : 'pointer',
                      fontSize: 13,
                      fontWeight: 600,
                      opacity: runtime.starting
                        ? 0.65
                        : 1,
                    }}
                  >
                    ← 返回课件工坊
                  </button>
                </>
              )}

            <button
              type="button"
              onClick={() => {
                void runtime.refreshAssemblyState()
                void runtime.syncStoredPages()
              }}
              style={{
                padding: '10px 18px',
                borderRadius: 9,
                border:
                  `1px solid ${C.border}`,
                background: '#fff',
                color: C.textSecondary,
                cursor: 'pointer',
                fontSize: 13,
              }}
            >
              🔄 同步后台状态
            </button>
          </div>
        </div>
      )}

      {showStylePicker && (
        <AnchorStylePicker
          coursewareId={coursewareId}
          skipVideo={currentSkipVideo}
          onConfirmed={runAssembly}
          onCancel={() => {
            setShowStylePicker(false)
          }}
          onAnchorChanged={result => {
            onAnchorChanged?.(result)
          }}
        />
      )}
    </div>
  )
}
