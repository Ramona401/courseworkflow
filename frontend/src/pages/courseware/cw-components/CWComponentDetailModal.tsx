/**
 * CWComponentDetailModal — 课件组件详情只读弹窗。
 *
 * 详情接口已经由后端执行直接ID教育域隔离。
 * 本组件仅展示返回的可信资源域，不提供教育域编辑或迁移控件。
 */
import type {
  CWComponentFull,
} from '@/api/coursewares'
import {
  RESOURCE_EDUCATION_DOMAIN_LABELS,
} from '@/education-domain/types'
import {
  CW_COMPONENT_COLORS as C,
  getCWComponentTypeConfig,
} from './cwComponentUi'

interface CWComponentDetailModalProps {
  component: CWComponentFull | null
  loading: boolean
  isMixed: boolean
  onClose: () => void
}

export default function CWComponentDetailModal({
  component,
  loading,
  isMixed,
  onClose,
}: CWComponentDetailModalProps) {
  if (!component && !loading) {
    return null
  }

  const readOnlyCommon =
    !isMixed &&
    component?.education_domain === 'common'

  const typeConfig = component
    ? getCWComponentTypeConfig(
      component.component_type,
    )
    : null

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.5)',
        zIndex: 9999,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
      onClick={() => {
        if (!loading) {
          onClose()
        }
      }}
    >
      <div
        style={{
          background: '#fff',
          borderRadius: '16px',
          width: '90%',
          maxWidth: '1000px',
          maxHeight: '85vh',
          overflow: 'auto',
        }}
        onClick={event =>
          event.stopPropagation()
        }
      >
        {loading ? (
          <div style={{
            padding: '60px',
            textAlign: 'center',
            color: C.textMuted,
          }}>
            加载中...
          </div>
        ) : component && typeConfig && (
          <>
            <div style={{
              padding: '24px 28px 16px',
              borderBottom:
                `1px solid ${C.border}`,
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'flex-start',
              gap: '16px',
            }}>
              <div>
                <div style={{
                  fontSize: '20px',
                  fontWeight: 700,
                  color: C.textPrimary,
                  marginBottom: '8px',
                }}>
                  {component.name}
                </div>

                <div style={{
                  display: 'flex',
                  gap: '8px',
                  fontSize: '12px',
                  color: C.textMuted,
                  flexWrap: 'wrap',
                  alignItems: 'center',
                }}>
                  <span style={{
                    padding: '2px 10px',
                    borderRadius: '12px',
                    fontSize: '11px',
                    fontWeight: 500,
                    color: typeConfig.color,
                    background: typeConfig.bg,
                  }}>
                    {typeConfig.label}
                  </span>

                  <span style={{
                    padding: '2px 8px',
                    borderRadius: '10px',
                    background:
                      component.education_domain ===
                      'common'
                        ? '#FFFBEB'
                        : '#F3F4F6',
                    color:
                      component.education_domain ===
                      'common'
                        ? '#92400E'
                        : C.textSecondary,
                    fontSize: '10px',
                    fontWeight: 600,
                  }}>
                    {
                      RESOURCE_EDUCATION_DOMAIN_LABELS[
                        component.education_domain
                      ]
                    }
                  </span>

                  {readOnlyCommon && (
                    <span>
                      当前身份只读
                    </span>
                  )}

                  <span>
                    📚 {component.subject_scope}
                  </span>

                  <span>
                    🎓 {component.grade_scope}
                  </span>

                  {component.idx_interaction_level
                    != null && (
                    <span>
                      ⚡ IL:
                      {component.idx_interaction_level}
                    </span>
                  )}
                </div>

                {component.description && (
                  <div style={{
                    fontSize: '13px',
                    color: C.textSecondary,
                    marginTop: '8px',
                    lineHeight: 1.5,
                  }}>
                    {component.description}
                  </div>
                )}
              </div>

              <button
                onClick={onClose}
                style={{
                  background: 'none',
                  border: 'none',
                  fontSize: '24px',
                  cursor: 'pointer',
                  color: C.textMuted,
                  padding: '0 4px',
                }}
              >
                ✕
              </button>
            </div>

            <div style={{
              padding: '20px 28px',
            }}>
              <div style={{
                fontSize: '14px',
                fontWeight: 600,
                color: C.textPrimary,
                marginBottom: '12px',
              }}>
                📺 实时预览
              </div>

              <div style={{
                border:
                  `1px solid ${C.border}`,
                borderRadius: '12px',
                overflow: 'hidden',
                background: '#f9f9f9',
              }}>
                <iframe
                  srcDoc={component.code_content}
                  style={{
                    width: '100%',
                    height: '400px',
                    border: 'none',
                  }}
                  sandbox="allow-scripts"
                  title="组件预览"
                />
              </div>
            </div>

            <details style={{
              padding: '0 28px 24px',
            }}>
              <summary style={{
                fontSize: '14px',
                fontWeight: 600,
                color: C.textPrimary,
                cursor: 'pointer',
                marginBottom: '8px',
              }}>
                💻 查看源代码（
                {component.code_content.length}
                字符）
              </summary>

              <pre style={{
                background: '#1E293B',
                color: '#E2E8F0',
                padding: '16px',
                borderRadius: '8px',
                fontSize: '12px',
                lineHeight: 1.6,
                overflow: 'auto',
                maxHeight: '300px',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
              }}>
                {component.code_content}
              </pre>
            </details>
          </>
        )}
      </div>
    </div>
  )
}
