/**
 * CWComponentGrid — 课件组件列表与空状态。
 *
 * 本组件只负责展示：
 *   - 资源教育域徽章；
 *   - common只读提示；
 *   - 组件类型、学科、层级和互动等级；
 *   - 已由父组件裁决后的删除按钮。
 */
import type {
  CWComponentListItem,
} from '@/api/coursewares'
import {
  RESOURCE_EDUCATION_DOMAIN_LABELS,
} from '@/education-domain/types'
import {
  CW_COMPONENT_COLORS as C,
  getCWComponentTypeConfig,
} from './cwComponentUi'

interface CWComponentGridProps {
  items: CWComponentListItem[]
  loading: boolean
  canRunSeed: boolean
  isMixed: boolean
  canMutate: (
    component: CWComponentListItem,
  ) => boolean
  onOpen: (id: string) => void
  onDelete: (
    component: CWComponentListItem,
  ) => void
}

export default function CWComponentGrid({
  items,
  loading,
  canRunSeed,
  isMixed,
  canMutate,
  onOpen,
  onDelete,
}: CWComponentGridProps) {
  if (loading) {
    return (
      <div style={{
        textAlign: 'center',
        padding: '60px 0',
        color: C.textMuted,
      }}>
        加载中...
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div style={{
        textAlign: 'center',
        padding: '80px 0',
      }}>
        <div style={{
          fontSize: '48px',
          marginBottom: '16px',
        }}>
          🧩
        </div>

        <div style={{
          fontSize: '16px',
          color: C.textSecondary,
          marginBottom: '8px',
        }}>
          当前范围暂无组件
        </div>

        <div style={{
          fontSize: '13px',
          color: C.textMuted,
        }}>
          {canRunSeed
            ? '可填充K12种子，或切换资源域筛选。'
            : '当前教育域没有可用的同域或通用组件。'}
        </div>
      </div>
    )
  }

  return (
    <div style={{
      display: 'grid',
      gridTemplateColumns:
        'repeat(auto-fill, minmax(300px, 1fr))',
      gap: '16px',
    }}>
      {items.map(item => {
        const typeConfig =
          getCWComponentTypeConfig(
            item.component_type,
          )

        const readOnlyCommon =
          !isMixed &&
          item.education_domain === 'common'

        return (
          <div
            key={item.id}
            onClick={() => onOpen(item.id)}
            style={{
              background: C.bgCard,
              borderRadius: '12px',
              padding: '20px',
              border: `1px solid ${C.border}`,
              cursor: 'pointer',
              transition: 'all 200ms ease',
            }}
            onMouseEnter={event => {
              event.currentTarget.style
                .borderColor =
                  'rgba(245,158,11,0.3)'
              event.currentTarget.style
                .transform =
                  'translateY(-2px)'
            }}
            onMouseLeave={event => {
              event.currentTarget.style
                .borderColor = C.border
              event.currentTarget.style
                .transform = 'none'
            }}
          >
            <div style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'flex-start',
              gap: '10px',
              marginBottom: '10px',
            }}>
              <div style={{
                fontSize: '15px',
                fontWeight: 600,
                color: C.textPrimary,
              }}>
                {item.name}
              </div>

              <span style={{
                padding: '2px 10px',
                borderRadius: '12px',
                fontSize: '11px',
                fontWeight: 500,
                color: typeConfig.color,
                background: typeConfig.bg,
                flexShrink: 0,
              }}>
                {typeConfig.label}
              </span>
            </div>

            <div style={{
              display: 'flex',
              gap: '6px',
              flexWrap: 'wrap',
              marginBottom: '10px',
            }}>
              <span style={{
                padding: '2px 8px',
                borderRadius: '10px',
                background:
                  item.education_domain ===
                  'common'
                    ? '#FFFBEB'
                    : '#F3F4F6',
                color:
                  item.education_domain ===
                  'common'
                    ? '#92400E'
                    : C.textSecondary,
                fontSize: '10px',
                fontWeight: 600,
              }}>
                {
                  RESOURCE_EDUCATION_DOMAIN_LABELS[
                    item.education_domain
                  ]
                }
              </span>

              {readOnlyCommon && (
                <span style={{
                  padding: '2px 8px',
                  borderRadius: '10px',
                  background: '#F3F4F6',
                  color: C.textMuted,
                  fontSize: '10px',
                }}>
                  只读
                </span>
              )}
            </div>

            {item.description && (
              <div style={{
                fontSize: '13px',
                color: C.textSecondary,
                marginBottom: '10px',
                lineHeight: 1.5,
              }}>
                {item.description.length > 80
                  ? `${item.description.slice(
                    0,
                    80,
                  )}...`
                  : item.description}
              </div>
            )}

            <div style={{
              display: 'flex',
              gap: '12px',
              fontSize: '12px',
              color: C.textMuted,
              justifyContent: 'space-between',
              alignItems: 'center',
            }}>
              <div style={{
                display: 'flex',
                gap: '12px',
                flexWrap: 'wrap',
              }}>
                <span>
                  📚 {item.subject_scope}
                </span>

                <span>
                  🎓 {item.grade_scope}
                </span>

                {item.idx_interaction_level
                  != null && (
                  <span>
                    ⚡ IL:
                    {item.idx_interaction_level}
                  </span>
                )}
              </div>

              {canMutate(item) && (
                <button
                  onClick={event => {
                    event.stopPropagation()
                    onDelete(item)
                  }}
                  style={{
                    padding: '2px 8px',
                    borderRadius: '4px',
                    border:
                      `1px solid ${C.border}`,
                    background: 'transparent',
                    color: C.danger,
                    fontSize: '11px',
                    cursor: 'pointer',
                  }}
                >
                  删除
                </button>
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}
