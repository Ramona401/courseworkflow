/**
 * LibraryComponents — 共享教案库展示组件
 *
 * 本文件只负责展示，不持有跨页面缓存或教育域授权逻辑。
 */

import {
  useState,
  type CSSProperties,
} from 'react'
import { useNavigate } from 'react-router-dom'
import type {
  LessonPlan,
  LessonPlanStatus,
} from '@/api/lesson-plans'
import type {
  InteractionCounts,
  InteractionType,
} from '@/api/lesson-plan-interactions'
import type { LibraryScope } from './useSharedLessonPlanLibrary'

export const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent: '#F59E0B',
  success: '#10B981',
  warning: '#F97316',
  danger: '#EF4444',
  purple: '#8B5CF6',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  bg: '#FAFBFC',
  card: '#FFFFFF',
  border: '#F3F4F6',
  borderHover: '#E5E7EB',
}

export const SCOPE_TABS = [
  {
    key: 'group' as LibraryScope,
    label: '教研组库',
    icon: '👥',
    desc: '本教研组共享的教案',
  },
  {
    key: 'school' as LibraryScope,
    label: '学校库',
    icon: '🏫',
    desc: '全校共享的优秀教案',
  },
  {
    key: 'region' as LibraryScope,
    label: '区域库',
    icon: '🌏',
    desc: '跨校共享的精品教案',
  },
]

export const GRADES = [
  '全部',
  '七年级',
  '八年级',
  '九年级',
  '高一',
  '高二',
  '高三',
  '小学低段',
  '小学中段',
  '小学高段',
]

const statusConfig: Partial<
  Record<
    LessonPlanStatus,
    {
      label: string
      color: string
      bg: string
    }
  >
> = {
  approved: {
    label: '评审通过',
    color: C.success,
    bg: 'rgba(16,185,129,0.08)',
  },
  published_shared: {
    label: '已共享',
    color: C.purple,
    bg: 'rgba(139,92,246,0.08)',
  },
}

export function StatusBadge({
  status,
}: {
  status: LessonPlanStatus
}) {
  const config = statusConfig[status]
  if (!config) return null

  return (
    <span style={{
      display: 'inline-flex',
      alignItems: 'center',
      gap: 5,
      padding: '3px 8px',
      borderRadius: 20,
      background: config.bg,
      fontSize: 12,
      fontWeight: 500,
      color: config.color,
      whiteSpace: 'nowrap',
    }}>
      <span style={{
        width: 6,
        height: 6,
        borderRadius: '50%',
        background: config.color,
      }} />
      {config.label}
    </span>
  )
}

function MetaTag({
  icon,
  text,
}: {
  icon: string
  text: string
}) {
  return (
    <span style={{
      display: 'inline-flex',
      alignItems: 'center',
      gap: 4,
      fontSize: 13,
      color: C.textSec,
      minWidth: 0,
    }}>
      <span>{icon}</span>
      <span style={{
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
      }}>
        {text}
      </span>
    </span>
  )
}

export function SkeletonCard() {
  const shimmer: CSSProperties = {
    background:
      'linear-gradient(90deg,#F3F4F6 25%,#E5E7EB 50%,#F3F4F6 75%)',
    backgroundSize: '200% 100%',
    animation: 'shimmer 1.4s infinite',
    borderRadius: 4,
  }

  return (
    <div style={{
      background: C.card,
      borderRadius: 12,
      border: `1px solid ${C.border}`,
      padding: 20,
    }}>
      <style>
        {`@keyframes shimmer {
          0% { background-position: 200% 0 }
          100% { background-position: -200% 0 }
        }`}
      </style>
      <div style={{
        ...shimmer,
        width: '60%',
        height: 18,
        marginBottom: 14,
      }} />
      <div style={{
        ...shimmer,
        width: '85%',
        height: 14,
        marginBottom: 14,
      }} />
      <div style={{
        ...shimmer,
        width: '45%',
        height: 14,
        marginBottom: 20,
      }} />
      <div style={{
        ...shimmer,
        width: '100%',
        height: 30,
      }} />
    </div>
  )
}

export function EmptyState({
  filtered,
  scope,
  onReset,
}: {
  filtered: boolean
  scope: LibraryScope
  onReset: () => void
}) {
  const navigate = useNavigate()
  const scopeLabel =
    SCOPE_TABS.find(tab => tab.key === scope)?.label ||
    '教案库'

  return (
    <div style={{
      gridColumn: '1 / -1',
      textAlign: 'center',
      padding: '80px 40px',
      background: C.card,
      borderRadius: 12,
      border: `1px solid ${C.border}`,
    }}>
      <div style={{ fontSize: 48 }}>
        {filtered ? '🔍' : '📚'}
      </div>
      <h3 style={{
        fontSize: 16,
        color: C.text,
      }}>
        {filtered
          ? '没有符合条件的教案'
          : `${scopeLabel}暂无共享教案`}
      </h3>
      <p style={{
        fontSize: 14,
        color: C.textMuted,
      }}>
        {filtered
          ? '试试调整筛选条件'
          : '评审通过的教案共享后将出现在这里'}
      </p>
      <button
        onClick={
          filtered
            ? onReset
            : () => navigate('/lesson-plans')
        }
        style={{
          padding: '10px 24px',
          borderRadius: 8,
          border: 'none',
          background: filtered
            ? C.bg
            : C.primary,
          color: filtered
            ? C.textSec
            : '#fff',
          cursor: 'pointer',
        }}
      >
        {filtered ? '清空筛选' : '✨ 去备课工坊'}
      </button>
    </div>
  )
}

export function FilterSelect({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: string
  options: string[]
  onChange: (value: string) => void
}) {
  const display = (option: string) => {
    if (label === '质量') {
      return {
        '5': '精品',
        '4': '优秀',
        '3': '良好',
        '2': '可用',
      }[option] || option
    }
    if (label === '教法') {
      return {
        '1': '讲授型',
        '2': '探究型',
        '3': '项目型',
        '4': '翻转型',
        '5': '混合型',
      }[option] || option
    }
    return option
  }

  return (
    <label style={{
      display: 'flex',
      alignItems: 'center',
      gap: 8,
      fontSize: 13,
      color: C.textSec,
    }}>
      <span>{label}</span>
      <select
        value={value}
        onChange={event =>
          onChange(event.target.value)
        }
        style={{
          padding: '6px 10px',
          borderRadius: 6,
          border: `1px solid ${
            value === '全部'
              ? C.border
              : C.primary
          }`,
          background: value === '全部'
            ? 'transparent'
            : C.primaryLight,
          color: value === '全部'
            ? C.textSec
            : C.primary,
        }}
      >
        {options.map(option => (
          <option key={option} value={option}>
            {display(option)}
          </option>
        ))}
      </select>
    </label>
  )
}

export function LibraryCard({
  plan,
  currentUserId,
  forkingId,
  interactions,
  likePending,
  favoritePending,
  forkDisabled,
  onFork,
  onToggleInteraction,
}: {
  plan: LessonPlan
  currentUserId: string
  forkingId: string | null
  interactions?: InteractionCounts
  likePending: boolean
  favoritePending: boolean
  forkDisabled: boolean
  onFork: (plan: LessonPlan) => void
  onToggleInteraction: (
    planId: string,
    type: InteractionType,
  ) => void
}) {
  const navigate = useNavigate()
  const [hovered, setHovered] = useState(false)
  const own = plan.author_id === currentUserId
  const forking = plan.id === forkingId

  const openDetail = () => navigate(
    `/lesson-plans/plans/${plan.id}`,
    {
      state: {
        from: '/lesson-plans/library',
      },
    },
  )

  return (
    <article
      onClick={openDetail}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        background: C.card,
        borderRadius: 12,
        border: `1px solid ${
          hovered ? C.borderHover : C.border
        }`,
        padding: 20,
        cursor: 'pointer',
        boxShadow: hovered
          ? '0 4px 16px rgba(0,0,0,.08)'
          : '0 1px 3px rgba(0,0,0,.04)',
      }}
    >
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        gap: 12,
      }}>
        <h3 style={{
          margin: 0,
          fontSize: 15,
          color: C.text,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}>
          {plan.title}
        </h3>
        <StatusBadge status={plan.status} />
      </div>

      <div style={{
        display: 'flex',
        gap: 12,
        flexWrap: 'wrap',
        margin: '12px 0',
      }}>
        <MetaTag icon="📚" text={plan.subject} />
        <MetaTag icon="🎓" text={plan.grade} />
        <MetaTag
          icon="⏱"
          text={`${plan.duration_minutes}分钟`}
        />
      </div>

      <div style={{
        fontSize: 13,
        color: C.textSec,
        marginBottom: 12,
      }}>
        👤 {plan.author_name || '教师'}
        {own && (
          <span style={{
            marginLeft: 6,
            color: C.primary,
          }}>
            我的
          </span>
        )}
      </div>

      {plan.ai_review_score != null && (
        <div style={{
          fontSize: 12,
          color: plan.ai_review_score >= 8.5
            ? C.success
            : C.accent,
          marginBottom: 12,
        }}>
          🤖 AI评分 {plan.ai_review_score.toFixed(1)}
        </div>
      )}

      <div
        onClick={event => event.stopPropagation()}
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: 10,
          paddingTop: 12,
          borderTop: `1px solid ${C.border}`,
        }}
      >
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
        }}>
          <button
            onClick={openDetail}
            style={linkButtonStyle}
          >
            查看详情
          </button>

          {interactions && (
            <>
              <button
                onClick={() =>
                  onToggleInteraction(plan.id, 'like')
                }
                disabled={likePending}
                style={interactionButtonStyle(
                  interactions.is_liked,
                  likePending,
                )}
              >
                👍 {interactions.like_count || ''}
              </button>
              <button
                onClick={() =>
                  onToggleInteraction(
                    plan.id,
                    'favorite',
                  )
                }
                disabled={favoritePending}
                style={interactionButtonStyle(
                  interactions.is_favorited,
                  favoritePending,
                )}
              >
                📌 {interactions.favorite_count || ''}
              </button>
            </>
          )}
        </div>

        {!own ? (
          <button
            onClick={() => onFork(plan)}
            disabled={forkDisabled}
            aria-busy={forking}
            style={{
              padding: '5px 14px',
              borderRadius: 6,
              border: `1px solid ${C.primary}`,
              background: C.primaryLight,
              color: C.primary,
              cursor: forkDisabled
                ? 'not-allowed'
                : 'pointer',
              opacity: forkDisabled && !forking
                ? 0.55
                : 1,
            }}
          >
            {forking ? '处理中...' : '🔀 Fork'}
          </button>
        ) : (
          <span style={{
            fontSize: 12,
            color: C.textMuted,
          }}>
            我发布的
          </span>
        )}
      </div>
    </article>
  )
}

const linkButtonStyle: CSSProperties = {
  padding: 0,
  border: 'none',
  background: 'none',
  color: C.primary,
  fontSize: 12,
  cursor: 'pointer',
}

function interactionButtonStyle(
  active: boolean,
  pending: boolean,
): CSSProperties {
  return {
    padding: '3px 8px',
    borderRadius: 6,
    border: `1px solid ${
      active ? C.primary : C.border
    }`,
    background: active
      ? C.primaryLight
      : 'transparent',
    color: active
      ? C.primary
      : C.textMuted,
    cursor: pending
      ? 'not-allowed'
      : 'pointer',
    opacity: pending ? 0.6 : 1,
    fontSize: 12,
  }
}
