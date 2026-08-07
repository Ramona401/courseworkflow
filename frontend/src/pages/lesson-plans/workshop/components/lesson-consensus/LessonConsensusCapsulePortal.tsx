/**
 * LessonConsensusCapsulePortal.tsx — 环境式“本课共识”胶囊
 *
 * 三种视觉状态：
 *   - 静默：右侧一枚轻量胶囊，不抢占对话内容；
 *   - 呼吸：收到新版本后短暂展开recent_change，不弹Toast、不改变布局；
 *   - 聚焦：教师主动点击后展开完整共识视图。
 *
 * 交互边界：
 *   - 没有确认、编辑、删除等管理按钮；
 *   - 教师直接在对话里表达修正、否定、搁置或恢复；
 *   - 流式输出期间组件只更新自身fixed浮层，不推动消息区重排；
 *   - 多个AIBubble会同时挂载本组件，但模块级Host仲裁只允许一个Portal渲染。
 */

import {
  useEffect,
  useId,
  useRef,
  useState,
  useSyncExternalStore,
} from 'react'
import { createPortal } from 'react-dom'
import {
  getLessonConsensusCapsuleSnapshot,
  subscribeLessonConsensusCapsule,
} from '@/store/lessonConsensusCapsule'
import type {
  LessonPlanContextCapsuleDisplaySection,
} from '@/api/lesson-plans'

const hostIDs = new Set<string>()
const hostListeners = new Set<() => void>()
let activeHostID = ''

function notifyHosts(): void {
  hostListeners.forEach(listener => listener())
}

function registerHost(hostID: string): () => void {
  hostIDs.add(hostID)

  if (!activeHostID) {
    activeHostID = hostID
    notifyHosts()
  }

  return () => {
    hostIDs.delete(hostID)

    if (activeHostID === hostID) {
      activeHostID =
        hostIDs.values().next().value || ''
      notifyHosts()
    }
  }
}

function subscribeHost(
  listener: () => void,
): () => void {
  hostListeners.add(listener)

  return () => {
    hostListeners.delete(listener)
  }
}

function getActiveHostID(): string {
  return activeHostID
}

const EMPHASIS_STYLE: Record<
  string,
  {
    accent: string
    background: string
  }
> = {
  stable: {
    accent: '#3567D6',
    background: '#F4F7FF',
  },
  active: {
    accent: '#0F8A72',
    background: '#F2FBF8',
  },
  guard: {
    accent: '#B7791F',
    background: '#FFF9EE',
  },
  soft: {
    accent: '#7C6DB0',
    background: '#F8F6FC',
  },
}

function sectionStyle(
  section: LessonPlanContextCapsuleDisplaySection,
) {
  return EMPHASIS_STYLE[
    section.emphasis || ''
  ] || {
    accent: '#64748B',
    background: '#F8FAFC',
  }
}

function ConsensusSection({
  section,
}: {
  section: LessonPlanContextCapsuleDisplaySection
}) {
  const style = sectionStyle(section)

  if (!section.items?.length) {
    return null
  }

  return (
    <section
      style={{
        borderRadius: '12px',
        padding: '11px 12px',
        background: style.background,
        borderLeft:
          `3px solid ${style.accent}`,
      }}
    >
      <div
        style={{
          marginBottom: '7px',
          color: style.accent,
          fontSize: '12px',
          fontWeight: 700,
          letterSpacing: '0.1px',
        }}
      >
        {section.title}
      </div>

      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: '6px',
        }}
      >
        {section.items.map(
          (item, index) => (
            <div
              key={`${section.key}_${index}`}
              style={{
                display: 'grid',
                gridTemplateColumns:
                  '8px minmax(0, 1fr)',
                gap: '7px',
                color: '#334155',
                fontSize: '12px',
                lineHeight: 1.65,
              }}
            >
              <span
                aria-hidden="true"
                style={{
                  width: '4px',
                  height: '4px',
                  marginTop: '8px',
                  borderRadius: '50%',
                  background: style.accent,
                }}
              />

              <span>{item}</span>
            </div>
          ),
        )}
      </div>
    </section>
  )
}

export default function LessonConsensusCapsulePortal() {
  const hostID = useId()

  useEffect(
    () => registerHost(hostID),
    [hostID],
  )

  const ownerID = useSyncExternalStore(
    subscribeHost,
    getActiveHostID,
    getActiveHostID,
  )

  const snapshot = useSyncExternalStore(
    subscribeLessonConsensusCapsule,
    getLessonConsensusCapsuleSnapshot,
    getLessonConsensusCapsuleSnapshot,
  )

  const [expanded, setExpanded] =
    useState(false)

  const [breathing, setBreathing] =
    useState(false)

  const previousVersionRef =
    useRef(0)

  const capsule = snapshot.capsule
  const display = capsule?.display

  useEffect(() => {
    if (
      !capsule?.version ||
      capsule.version ===
        previousVersionRef.current
    ) {
      return
    }

    previousVersionRef.current =
      capsule.version

    setBreathing(true)

    const timer = window.setTimeout(
      () => setBreathing(false),
      2600,
    )

    return () => {
      window.clearTimeout(timer)
    }
  }, [capsule?.version])

  useEffect(() => {
    setExpanded(false)
  }, [snapshot.activePlanId])

  if (
    ownerID !== hostID ||
    typeof document === 'undefined' ||
    !snapshot.activePlanId ||
    !display
  ) {
    return null
  }

  const recentChange =
    display.recent_change?.trim() ||
    display.summary?.trim()

  const className = [
    'lesson-consensus-capsule',
    expanded ? 'is-focused' : '',
    breathing && !expanded
      ? 'is-breathing'
      : '',
  ]
    .filter(Boolean)
    .join(' ')

  return createPortal(
    <>
      <aside
        className={className}
        aria-label="本课共识"
        aria-live={
          breathing ? 'polite' : 'off'
        }
      >
        {!expanded ? (
          <button
            type="button"
            className="lesson-consensus-capsule__summary"
            onClick={() => setExpanded(true)}
            aria-expanded="false"
            title="查看本课当前共识"
          >
            <span
              className="lesson-consensus-capsule__mark"
              aria-hidden="true"
            >
              ✦
            </span>

            <span
              className="lesson-consensus-capsule__summary-text"
            >
              <strong>本课共识</strong>

              <span>
                {breathing
                  ? recentChange
                  : (
                    display.state_label ||
                    display.summary
                  )}
              </span>
            </span>

            <span
              className="lesson-consensus-capsule__chevron"
              aria-hidden="true"
            >
              ›
            </span>
          </button>
        ) : (
          <div
            className="lesson-consensus-capsule__panel"
          >
            <header
              className="lesson-consensus-capsule__header"
            >
              <div>
                <div
                  className="lesson-consensus-capsule__eyebrow"
                >
                  {display.state_label ||
                    '理解同步'}
                </div>

                <h2>
                  {display.headline ||
                    '我们已经确定的'}
                </h2>
              </div>

              <button
                type="button"
                onClick={() => setExpanded(false)}
                aria-label="收起本课共识"
              >
                ×
              </button>
            </header>

            {display.summary && (
              <p
                className="lesson-consensus-capsule__lead"
              >
                {display.summary}
              </p>
            )}

            {display.recent_change && (
              <div
                className="lesson-consensus-capsule__change"
              >
                <span aria-hidden="true">↗</span>
                <span>
                  {display.recent_change}
                </span>
              </div>
            )}

            <div
              className="lesson-consensus-capsule__sections"
            >
              {(display.sections || []).map(
                section => (
                  <ConsensusSection
                    key={section.key}
                    section={section}
                  />
                ),
              )}
            </div>

            <footer>
              需要调整时，直接在对话里说就好。
            </footer>
          </div>
        )}
      </aside>

      <style>{`
        .lesson-consensus-capsule {
          position: fixed;
          top: 118px;
          right: 18px;
          z-index: 9200;
          width: 280px;
          pointer-events: none;
          color: #1E293B;
          font-family: inherit;
        }

        .lesson-consensus-capsule__summary,
        .lesson-consensus-capsule__panel {
          pointer-events: auto;
          width: 100%;
          box-sizing: border-box;
          border: 1px solid rgba(148, 163, 184, 0.28);
          background: rgba(255, 255, 255, 0.88);
          box-shadow:
            0 12px 36px rgba(15, 23, 42, 0.12),
            inset 0 1px 0 rgba(255, 255, 255, 0.7);
          backdrop-filter: blur(18px);
          -webkit-backdrop-filter: blur(18px);
        }

        .lesson-consensus-capsule__summary {
          min-height: 48px;
          padding: 8px 10px;
          border-radius: 16px 4px 4px 16px;
          display: grid;
          grid-template-columns: 28px minmax(0, 1fr) 16px;
          align-items: center;
          gap: 8px;
          text-align: left;
          cursor: pointer;
          transition:
            width 220ms ease,
            transform 220ms ease,
            box-shadow 220ms ease,
            background 220ms ease;
        }

        .lesson-consensus-capsule__summary:hover {
          transform: translateX(-3px);
          box-shadow:
            0 16px 42px rgba(15, 23, 42, 0.16),
            inset 0 1px 0 rgba(255, 255, 255, 0.8);
        }

        .lesson-consensus-capsule__mark {
          width: 28px;
          height: 28px;
          border-radius: 10px;
          display: inline-flex;
          align-items: center;
          justify-content: center;
          background: linear-gradient(
            135deg,
            #4F7BE8,
            #7C6EE6
          );
          color: #FFFFFF;
          font-size: 13px;
          box-shadow:
            0 5px 14px rgba(79, 123, 232, 0.24);
        }

        .lesson-consensus-capsule__summary-text {
          min-width: 0;
          display: flex;
          flex-direction: column;
          gap: 2px;
        }

        .lesson-consensus-capsule__summary-text strong {
          color: #334155;
          font-size: 12px;
          font-weight: 700;
        }

        .lesson-consensus-capsule__summary-text span {
          overflow: hidden;
          color: #64748B;
          font-size: 11px;
          line-height: 1.45;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .lesson-consensus-capsule__chevron {
          color: #94A3B8;
          font-size: 20px;
          line-height: 1;
        }

        .lesson-consensus-capsule.is-breathing {
          width: 340px;
        }

        .lesson-consensus-capsule.is-breathing
        .lesson-consensus-capsule__summary {
          background:
            linear-gradient(
              135deg,
              rgba(244, 247, 255, 0.96),
              rgba(247, 253, 251, 0.94)
            );
          animation:
            lesson-consensus-breathe
            1.3s ease-in-out 2;
        }

        .lesson-consensus-capsule__panel {
          max-height: min(640px, calc(100vh - 150px));
          padding: 16px;
          border-radius: 18px 6px 6px 18px;
          overflow-y: auto;
          animation:
            lesson-consensus-focus-in
            180ms ease-out;
        }

        .lesson-consensus-capsule__header {
          display: flex;
          align-items: flex-start;
          justify-content: space-between;
          gap: 12px;
          margin-bottom: 10px;
        }

        .lesson-consensus-capsule__eyebrow {
          margin-bottom: 3px;
          color: #6D7E9F;
          font-size: 10px;
          font-weight: 700;
          letter-spacing: 0.5px;
        }

        .lesson-consensus-capsule__header h2 {
          margin: 0;
          color: #1E293B;
          font-size: 16px;
          line-height: 1.35;
        }

        .lesson-consensus-capsule__header button {
          width: 28px;
          height: 28px;
          flex-shrink: 0;
          border: none;
          border-radius: 9px;
          background: #F1F5F9;
          color: #64748B;
          cursor: pointer;
          font-size: 18px;
          line-height: 1;
        }

        .lesson-consensus-capsule__lead {
          margin: 0 0 12px;
          color: #475569;
          font-size: 13px;
          line-height: 1.75;
        }

        .lesson-consensus-capsule__change {
          margin-bottom: 12px;
          padding: 8px 10px;
          display: flex;
          align-items: flex-start;
          gap: 7px;
          border-radius: 11px;
          background:
            linear-gradient(
              135deg,
              #F4F7FF,
              #F4FBF9
            );
          color: #46618C;
          font-size: 11px;
          line-height: 1.65;
        }

        .lesson-consensus-capsule__sections {
          display: flex;
          flex-direction: column;
          gap: 9px;
        }

        .lesson-consensus-capsule footer {
          margin-top: 12px;
          padding-top: 10px;
          border-top: 1px dashed #DCE5F2;
          color: #94A3B8;
          font-size: 10px;
          line-height: 1.55;
        }

        @keyframes lesson-consensus-breathe {
          0%, 100% {
            transform: translateX(0);
            box-shadow:
              0 12px 36px rgba(15, 23, 42, 0.12);
          }

          50% {
            transform: translateX(-5px);
            box-shadow:
              0 18px 46px rgba(79, 123, 232, 0.2);
          }
        }

        @keyframes lesson-consensus-focus-in {
          from {
            opacity: 0;
            transform: translateX(12px) scale(0.985);
          }

          to {
            opacity: 1;
            transform: translateX(0) scale(1);
          }
        }

        @media (max-width: 1023px) {
          .lesson-consensus-capsule {
            top: auto;
            right: 12px;
            bottom: 76px;
            left: 12px;
            width: auto;
          }

          .lesson-consensus-capsule.is-breathing {
            width: auto;
          }

          .lesson-consensus-capsule__summary {
            border-radius: 15px;
          }

          .lesson-consensus-capsule__panel {
            max-height: min(58vh, 520px);
            border-radius: 18px;
          }
        }

        @media (prefers-reduced-motion: reduce) {
          .lesson-consensus-capsule *,
          .lesson-consensus-capsule {
            animation: none !important;
            transition: none !important;
          }
        }
      `}</style>
    </>,
    document.body,
  )
}
