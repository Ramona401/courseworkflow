/**
 * DeliveryModeSelect.tsx — Step4交付模式选择器
 *
 * 除三档选择外，本组件会在挂载时查询后端装配生命周期：
 *   - 若发现starting/running/cancel_requested，自动恢复到对应装配面板；
 *   - 查询期间暂时锁定卡片，避免刷新后重复启动；
 *   - 查询失败只提示，不阻断老师继续选择。
 */
import {
  useEffect,
  useState,
} from 'react'
import { useParams } from 'react-router-dom'

import {
  getCoursewareAssemblyState,
} from '@/api/coursewareAssembly'

import { C } from './workshopConstants'

export type DeliveryMode =
  | 'manual'
  | 'no_video'
  | 'full'

interface Props {
  hasAnchor: boolean
  remainingCount: number
  onSelect: (mode: DeliveryMode) => void
}

interface ModeCard {
  mode: DeliveryMode
  emoji: string
  title: string
  desc: string
  bullets: string[]
  accent: string
  accentBg: string
}

const MODE_CARDS: ModeCard[] = [
  {
    mode: 'manual',
    emoji: '✋',
    title: '纯手动',
    desc: '只生成页面，配图和视频你来把控',
    bullets: [
      '逐页生成 HTML 排版',
      '配图 / 视频后续在工作台手动做',
      '最省积分，最灵活',
    ],
    accent: '#059669',
    accentBg: '#ECFDF5',
  },
  {
    mode: 'no_video',
    emoji: '🖼',
    title: 'HTML + 配图',
    desc: '自动生成并配图，不做视频',
    bullets: [
      '逐页生成 HTML + 自动配图',
      '配图套用风格锚点，全课件统一',
      '不生成视频占位，比全自动更快省',
    ],
    accent: '#0891B2',
    accentBg: '#F0FDFF',
  },
  {
    mode: 'full',
    emoji: '⚡',
    title: '全自动装配',
    desc: '一键交付，图文视频齐备',
    bullets: [
      'HTML + 配图 + 视频首帧占位',
      '含视频/动画的页自动备好首帧',
      '一步到位，耗时与积分最高',
    ],
    accent: '#7C3AED',
    accentBg: '#FAF5FF',
  },
]

export default function DeliveryModeSelect({
  hasAnchor,
  remainingCount,
  onSelect,
}: Props) {
  const { id } = useParams<{
    id: string
  }>()

  const [checkingRuntime, setCheckingRuntime] =
    useState(true)
  const [runtimeWarning, setRuntimeWarning] =
    useState('')

  useEffect(() => {
    let disposed = false

    const restore = async () => {
      if (!id) {
        if (!disposed) {
          setCheckingRuntime(false)
        }
        return
      }

      try {
        const state =
          await getCoursewareAssemblyState(id)

        if (disposed) return

        if (state.is_active) {
          onSelect(
            state.skip_video
              ? 'no_video'
              : 'full',
          )
          return
        }
      } catch {
        if (!disposed) {
          setRuntimeWarning(
            '未能确认后台装配状态。你仍可继续选择；若系统提示任务已运行，请刷新页面。',
          )
        }
      } finally {
        if (!disposed) {
          setCheckingRuntime(false)
        }
      }
    }

    void restore()

    return () => {
      disposed = true
    }
  }, [id, onSelect])

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <div
          style={{
            fontSize: 13,
            color: C.textSecondary,
            lineHeight: 1.7,
          }}
        >
          选择本次交付方式
          {remainingCount > 0 ? (
            <>
              ，剩余{' '}
              <b
                style={{
                  color: C.textPrimary,
                }}
              >
                {remainingCount}
              </b>{' '}
              页待生成
            </>
          ) : (
            ''
          )}
          。自动配图会套用你设置的风格锚点，保持全课件视觉统一。
        </div>
      </div>

      {checkingRuntime && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '10px 14px',
            borderRadius: 10,
            marginBottom: 16,
            background: '#EFF6FF',
            border: '1px solid #BFDBFE',
            fontSize: 12.5,
            color: '#1D4ED8',
          }}
        >
          <span>🔄</span>
          <span>
            正在确认是否有后台装配任务，请稍候…
          </span>
        </div>
      )}

      {runtimeWarning && (
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: 8,
            padding: '10px 14px',
            borderRadius: 10,
            marginBottom: 16,
            background: '#FFFBEB',
            border: '1px solid #FDE68A',
            fontSize: 12.5,
            color: '#92400E',
            lineHeight: 1.6,
          }}
        >
          <span>⚠️</span>
          <span>{runtimeWarning}</span>
        </div>
      )}

      {!hasAnchor && (
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: 8,
            padding: '10px 14px',
            borderRadius: 10,
            marginBottom: 16,
            background: '#FFFBEB',
            border: '1px solid #FDE68A',
            fontSize: 12.5,
            color: '#92400E',
            lineHeight: 1.6,
          }}
        >
          <span style={{ fontSize: 15 }}>💡</span>
          <span>
            自动配图会在下一步打开画风选择弹窗，可当场建立风格锚点后开始装配。
          </span>
        </div>
      )}

      <div
        style={{
          display: 'grid',
          gridTemplateColumns:
            'repeat(auto-fit, minmax(220px, 1fr))',
          gap: 14,
        }}
      >
        {MODE_CARDS.map(card => {
          const locked = checkingRuntime

          return (
            <div
              key={card.mode}
              onClick={() => {
                if (!locked) {
                  onSelect(card.mode)
                }
              }}
              style={{
                position: 'relative',
                borderRadius: 14,
                padding: '18px 16px',
                border: `2px solid ${
                  locked
                    ? C.border
                    : `${card.accent}55`
                }`,
                background: locked
                  ? '#F9FAFB'
                  : card.accentBg,
                cursor: locked
                  ? 'wait'
                  : 'pointer',
                opacity: locked ? 0.65 : 1,
                transition: 'all 200ms',
                display: 'flex',
                flexDirection: 'column',
              }}
              onMouseEnter={event => {
                if (!locked) {
                  event.currentTarget.style.boxShadow =
                    `0 6px 20px ${card.accent}22`
                }
              }}
              onMouseLeave={event => {
                event.currentTarget.style.boxShadow =
                  'none'
              }}
            >
              <div
                style={{
                  fontSize: 26,
                  marginBottom: 8,
                }}
              >
                {card.emoji}
              </div>
              <div
                style={{
                  fontSize: 16,
                  fontWeight: 700,
                  color: locked
                    ? C.textMuted
                    : card.accent,
                  marginBottom: 4,
                }}
              >
                {card.title}
              </div>
              <div
                style={{
                  fontSize: 12.5,
                  color: C.textSecondary,
                  lineHeight: 1.5,
                  marginBottom: 12,
                  minHeight: 34,
                }}
              >
                {card.desc}
              </div>

              <ul
                style={{
                  margin: 0,
                  padding: 0,
                  listStyle: 'none',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 6,
                  flex: 1,
                }}
              >
                {card.bullets.map(
                  (bullet, index) => (
                    <li
                      key={index}
                      style={{
                        display: 'flex',
                        alignItems: 'flex-start',
                        gap: 6,
                        fontSize: 12,
                        color: C.textSecondary,
                        lineHeight: 1.5,
                      }}
                    >
                      <span
                        style={{
                          color: locked
                            ? C.textMuted
                            : card.accent,
                          fontWeight: 700,
                          flexShrink: 0,
                        }}
                      >
                        ·
                      </span>
                      <span>{bullet}</span>
                    </li>
                  ),
                )}
              </ul>

              <div
                style={{
                  marginTop: 14,
                  textAlign: 'center',
                  fontSize: 13,
                  fontWeight: 600,
                  padding: '8px 0',
                  borderRadius: 8,
                  color: locked
                    ? C.textMuted
                    : '#fff',
                  background: locked
                    ? '#F3F4F6'
                    : card.accent,
                }}
              >
                {locked
                  ? '正在检查后台状态…'
                  : '选择此方式 →'}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
