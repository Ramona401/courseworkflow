/**
 * StageComponentsModal — 阶段组件推荐弹窗
 *
 * 迭代12新增 / v78增强：
 *   阶段过渡动画结束后弹出，展示当前阶段推荐的教学组件。
 *   用户可以勾选需要的组件，选中的组件ID会在 advanceStage 时传给后端。
 *   v78新增：点击组件卡片可展开查看详情（设计逻辑+完整指引+示例片段）。
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getStageRecommendedComponents,
  type RecommendedComponentItem,
} from '@/api/lesson-plans'
import { C } from './workshopConstants'

// ==================== 组件类型中文名映射 ====================
const LIBRARY_TYPE_LABELS: Record<string, string> = {
  curriculum_standard: '课标要求',
  knowledge_graph: '知识图谱',
  student_profile: '学情画像',
  pedagogy: '教学法',
  activity_design: '活动设计',
  questioning_strategy: '提问策略',
  assessment_strategy: '评价策略',
  cross_subject: '跨学科',
  teaching_tool: '教学工具',
  scenario_material: '情境素材',
  quality_rubric: '质量标准',
  review_rubric: '评审标准',
  design_defect: '设计缺陷',
}

// ==================== Props ====================
interface StageComponentsModalProps {
  planId: string
  stageCode: string
  stageName: string
  loading?: boolean
  /** v121 任务B:
   *  - 'transition'(默认):阶段过渡时打开,保留"跳过,直接开始"按钮
   *  - 'pick-only':阶段进行中随时打开,隐藏"跳过",按钮文案改为"添加到对话" */
  mode?: 'transition' | 'pick-only'
  onConfirm: (selectedIds: string[]) => void
  onSkip: () => void
  onCancel: () => void
}

// ==================== 主组件 ====================
export default function StageComponentsModal({
  planId,
  stageCode,
  stageName,
  loading: externalLoading,
  mode = 'transition',  // v121 任务B:默认过渡模式,向后兼容
  onConfirm,
  onSkip,
  onCancel,
}: StageComponentsModalProps) {
  const [components, setComponents] = useState<RecommendedComponentItem[]>([])
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [fetchLoading, setFetchLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  // v78新增：展开详情的组件ID
  const [expandedId, setExpandedId] = useState<string | null>(null)

  // 加载推荐组件
  const fetchComponents = useCallback(async () => {
    setFetchLoading(true)
    setError(null)
    try {
      const resp = await getStageRecommendedComponents(planId, stageCode)
      setComponents(resp.components || [])
    } catch (err) {
      console.error('获取推荐组件失败:', err)
      setError('加载推荐组件失败')
      setComponents([])
    } finally {
      setFetchLoading(false)
    }
  }, [planId, stageCode])

  useEffect(() => {
    fetchComponents()
  }, [fetchComponents])

  // 切换选中状态
  const toggleSelect = (id: string, e: React.MouseEvent) => {
    e.stopPropagation()
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) { next.delete(id) } else { next.add(id) }
      return next
    })
  }

  // 全选/取消全选
  const toggleAll = () => {
    if (selectedIds.size === components.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(components.map(c => c.id)))
    }
  }

  // v78：展开/收起详情
  const toggleExpand = (id: string) => {
    setExpandedId(prev => prev === id ? null : id)
  }

  // 判断组件是否有详情内容可展示
  const hasDetail = (comp: RecommendedComponentItem) => {
    return !!(comp.full_guide || comp.example_snippet || (comp.design_logic && comp.design_logic.length > 80))
  }

  const isLoading = fetchLoading || externalLoading

  // 按 library_type 分组
  const groupedComponents: { type: string; name: string; items: RecommendedComponentItem[] }[] = []
  const typeMap = new Map<string, RecommendedComponentItem[]>()
  for (const comp of components) {
    const existing = typeMap.get(comp.library_type)
    if (existing) { existing.push(comp) } else { typeMap.set(comp.library_type, [comp]) }
  }
  for (const [type, items] of typeMap) {
    groupedComponents.push({
      type,
      name: items[0]?.library_name || LIBRARY_TYPE_LABELS[type] || type,
      items,
    })
  }

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
      background: 'rgba(0,0,0,0.45)', zIndex: 1000,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}
      onClick={e => { if (e.target === e.currentTarget) onCancel() }}
    >
      <div style={{
        background: '#fff', borderRadius: '16px',
        width: '640px', maxHeight: '85vh',
        display: 'flex', flexDirection: 'column',
        boxShadow: '0 20px 60px rgba(0,0,0,0.15)',
        overflow: 'hidden',
      }}>
        {/* 头部 */}
        <div style={{
          padding: '24px 28px 16px',
          borderBottom: `1px solid ${C.border}`,
        }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <h3 style={{ margin: 0, fontSize: '17px', fontWeight: 700, color: C.text }}>
                {mode === 'pick-only' ? '📚 补充参考组件' : '📚 选择参考组件'}
              </h3>
              <p style={{ margin: '6px 0 0', fontSize: '13px', color: C.textMuted, lineHeight: 1.5 }}>
                {mode === 'pick-only'
                  ? `从「${stageName}」阶段的组件库中挑选,选中后添加到下一条消息的上下文`
                  : `为「${stageName}」阶段选择教学参考组件,点击卡片可查看详情`
                }
              </p>
            </div>
            <button
              onClick={onCancel}
              style={{
                background: 'none', border: 'none', cursor: 'pointer',
                fontSize: '20px', color: C.textMuted, padding: '4px',
              }}
            >✕</button>
          </div>
        </div>

        {/* 内容区 */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '16px 28px' }}>
          {/* 加载中 */}
          {isLoading && (
            <div style={{
              display: 'flex', flexDirection: 'column', alignItems: 'center',
              justifyContent: 'center', padding: '48px 0', gap: '12px',
            }}>
              <div style={{
                width: '32px', height: '32px',
                border: `3px solid ${C.primary}`, borderTopColor: 'transparent',
                borderRadius: '50%', animation: 'spin 0.8s linear infinite',
              }} />
              <span style={{ fontSize: '14px', color: C.textMuted }}>正在加载推荐组件...</span>
              <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
            </div>
          )}

          {/* 错误 */}
          {!isLoading && error && (
            <div style={{ textAlign: 'center', padding: '40px 0', color: C.textMuted, fontSize: '14px' }}>
              <div style={{ fontSize: '28px', marginBottom: '8px' }}>⚠️</div>
              {error}
              <div style={{ marginTop: '12px' }}>
                <button onClick={fetchComponents} style={{
                  padding: '6px 16px', borderRadius: '8px',
                  border: `1px solid ${C.border}`, background: 'transparent',
                  fontSize: '13px', color: C.primary, cursor: 'pointer',
                }}>重试</button>
              </div>
            </div>
          )}

          {/* 空状态 */}
          {!isLoading && !error && components.length === 0 && (
            <div style={{ textAlign: 'center', padding: '40px 0', color: C.textMuted, fontSize: '14px' }}>
              <div style={{ fontSize: '28px', marginBottom: '8px' }}>📭</div>
              该阶段暂无推荐组件
              <p style={{ fontSize: '12px', marginTop: '8px', color: C.textMuted }}>
                将使用AI自动匹配的组件
              </p>
            </div>
          )}

          {/* 组件列表 */}
          {!isLoading && !error && components.length > 0 && (
            <>
              {/* 全选栏 */}
              <div style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                marginBottom: '12px',
              }}>
                <span style={{ fontSize: '13px', color: C.textSec }}>
                  共 {components.length} 个组件，已选 {selectedIds.size} 个
                </span>
                <button onClick={toggleAll} style={{
                  background: 'none', border: 'none', cursor: 'pointer',
                  fontSize: '13px', color: C.primary, padding: '2px 4px',
                }}>
                  {selectedIds.size === components.length ? '取消全选' : '全选'}
                </button>
              </div>

              {/* 分组展示 */}
              {groupedComponents.map(group => (
                <div key={group.type} style={{ marginBottom: '16px' }}>
                  <div style={{
                    fontSize: '12px', fontWeight: 600, color: C.textSec,
                    marginBottom: '8px', padding: '4px 0',
                    borderBottom: `1px solid ${C.border}`,
                  }}>
                    {group.name}
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                    {group.items.map(comp => {
                      const isSelected = selectedIds.has(comp.id)
                      const isExpanded = expandedId === comp.id
                      const canExpand = hasDetail(comp)

                      return (
                        <div key={comp.id} style={{
                          borderRadius: '10px',
                          border: `1.5px solid ${isSelected ? C.primary : C.border}`,
                          background: isSelected ? 'rgba(79,123,232,0.04)' : '#fff',
                          transition: 'all 150ms ease',
                          overflow: 'hidden',
                        }}>
                          {/* 组件头部（可点击展开） */}
                          <div
                            onClick={() => canExpand && toggleExpand(comp.id)}
                            style={{
                              display: 'flex', alignItems: 'flex-start', gap: '10px',
                              padding: '10px 12px',
                              cursor: canExpand ? 'pointer' : 'default',
                            }}
                            onMouseEnter={e => {
                              if (!isSelected) (e.currentTarget.parentElement as HTMLElement).style.borderColor = 'rgba(79,123,232,0.4)'
                            }}
                            onMouseLeave={e => {
                              if (!isSelected) (e.currentTarget.parentElement as HTMLElement).style.borderColor = C.border
                            }}
                          >
                            {/* 复选框（点击只切换选中，不展开） */}
                            <div
                              onClick={(e) => toggleSelect(comp.id, e)}
                              style={{
                                width: '20px', height: '20px', borderRadius: '5px', flexShrink: 0,
                                marginTop: '1px',
                                border: `2px solid ${isSelected ? C.primary : '#D1D5DB'}`,
                                background: isSelected ? C.primary : '#fff',
                                display: 'flex', alignItems: 'center', justifyContent: 'center',
                                transition: 'all 150ms ease', cursor: 'pointer',
                              }}
                            >
                              {isSelected && <span style={{ color: '#fff', fontSize: '12px', fontWeight: 700 }}>✓</span>}
                            </div>
                            {/* 内容 */}
                            <div style={{ flex: 1, minWidth: 0 }}>
                              <div style={{ display: 'flex', alignItems: 'center', gap: '6px', flexWrap: 'wrap' }}>
                                <span style={{
                                  fontSize: '14px', fontWeight: 500, color: C.text,
                                }}>{comp.display_label}</span>
                                {comp.source === 'recipe' && (
                                  <span style={{
                                    fontSize: '10px', padding: '1px 6px', borderRadius: '4px',
                                    background: 'rgba(245,158,11,0.1)', color: '#D97706', whiteSpace: 'nowrap',
                                  }}>配方</span>
                                )}
                                {comp.quality_score > 0 && (
                                  <span style={{ fontSize: '10px', color: C.textMuted, whiteSpace: 'nowrap' }}>
                                    ⭐{comp.quality_score.toFixed(1)}
                                  </span>
                                )}
                              </div>
                              {/* 摘要（未展开时显示） */}
                              {!isExpanded && comp.design_logic && (
                                <div style={{
                                  fontSize: '12px', color: C.textMuted, marginTop: '3px', lineHeight: 1.5,
                                  overflow: 'hidden', textOverflow: 'ellipsis',
                                  display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' as const,
                                }}>{comp.design_logic}</div>
                              )}
                              {/* 展开提示 */}
                              {canExpand && !isExpanded && (
                                <div style={{ fontSize: '11px', color: C.primary, marginTop: '4px', opacity: 0.7 }}>
                                  点击查看详情 ▾
                                </div>
                              )}
                            </div>
                          </div>

                          {/* v78新增：展开详情区域 */}
                          {isExpanded && (
                            <div style={{
                              padding: '0 12px 14px 42px',
                              borderTop: `1px solid ${C.border}`,
                              background: 'rgba(0,0,0,0.015)',
                            }}>
                              {/* 设计逻辑 */}
                              {comp.design_logic && (
                                <div style={{ marginTop: '12px' }}>
                                  <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '4px' }}>
                                    💡 设计逻辑
                                  </div>
                                  <div style={{
                                    fontSize: '13px', color: C.text, lineHeight: 1.7,
                                    whiteSpace: 'pre-wrap', wordBreak: 'break-word',
                                  }}>{comp.design_logic}</div>
                                </div>
                              )}

                              {/* 完整指引 */}
                              {comp.full_guide && (
                                <div style={{ marginTop: '12px' }}>
                                  <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '4px' }}>
                                    📖 完整指引
                                  </div>
                                  <div style={{
                                    fontSize: '13px', color: C.text, lineHeight: 1.7,
                                    whiteSpace: 'pre-wrap', wordBreak: 'break-word',
                                    maxHeight: '300px', overflowY: 'auto',
                                    padding: '8px 12px', borderRadius: '8px',
                                    background: 'rgba(79,123,232,0.03)',
                                    border: `1px solid rgba(79,123,232,0.1)`,
                                  }}>{comp.full_guide}</div>
                                </div>
                              )}

                              {/* 示例片段 */}
                              {comp.example_snippet && (
                                <div style={{ marginTop: '12px' }}>
                                  <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '4px' }}>
                                    📝 示例片段
                                  </div>
                                  <div style={{
                                    fontSize: '13px', color: C.text, lineHeight: 1.7,
                                    whiteSpace: 'pre-wrap', wordBreak: 'break-word',
                                    padding: '8px 12px', borderRadius: '8px',
                                    background: 'rgba(16,185,129,0.03)',
                                    border: '1px solid rgba(16,185,129,0.1)',
                                  }}>{comp.example_snippet}</div>
                                </div>
                              )}

                              {/* 收起按钮 */}
                              <div style={{ marginTop: '10px', textAlign: 'right' }}>
                                <button
                                  onClick={() => setExpandedId(null)}
                                  style={{
                                    background: 'none', border: 'none', cursor: 'pointer',
                                    fontSize: '12px', color: C.primary, padding: '2px 8px',
                                  }}
                                >▴ 收起</button>
                              </div>
                            </div>
                          )}
                        </div>
                      )
                    })}
                  </div>
                </div>
              ))}
            </>
          )}
        </div>

        {/* 底部操作栏
            v121 任务B:pick-only 模式下隐藏"跳过"按钮(没有下一步可跳),
            确认按钮文案改为"添加到对话" */}
        <div style={{
          padding: '16px 28px', borderTop: `1px solid ${C.border}`,
          display: 'flex', alignItems: 'center',
          justifyContent: mode === 'pick-only' ? 'flex-end' : 'space-between',
          background: '#FAFBFC',
          gap: '12px',
        }}>
          {mode === 'transition' && (
            <button
              onClick={onSkip}
              style={{
                padding: '9px 20px', borderRadius: '10px',
                border: `1px solid ${C.border}`, background: 'transparent',
                fontSize: '13px', color: C.textSec, cursor: 'pointer',
              }}
            >跳过,直接开始</button>
          )}

          {mode === 'pick-only' && (
            <button
              onClick={onCancel}
              style={{
                padding: '9px 20px', borderRadius: '10px',
                border: `1px solid ${C.border}`, background: 'transparent',
                fontSize: '13px', color: C.textSec, cursor: 'pointer',
              }}
            >取消</button>
          )}

          <button
            onClick={() => onConfirm(Array.from(selectedIds))}
            disabled={isLoading || (mode === 'pick-only' && selectedIds.size === 0)}
            style={{
              padding: '9px 24px', borderRadius: '10px', border: 'none',
              background: selectedIds.size > 0
                ? 'linear-gradient(135deg, #4F7BE8, #818CF8)'
                : C.primary,
              color: '#fff', fontSize: '14px', fontWeight: 600,
              cursor: (isLoading || (mode === 'pick-only' && selectedIds.size === 0)) ? 'not-allowed' : 'pointer',
              opacity: (isLoading || (mode === 'pick-only' && selectedIds.size === 0)) ? 0.5 : 1,
              boxShadow: selectedIds.size > 0 ? '0 3px 12px rgba(79,123,232,0.3)' : 'none',
              transition: 'all 200ms ease',
            }}
          >
            {mode === 'pick-only'
              ? (selectedIds.size > 0
                  ? `✅ 添加${selectedIds.size}个组件到对话`
                  : '请至少选择1个组件')
              : (selectedIds.size > 0
                  ? `选好了,开始${stageName}(${selectedIds.size}个组件)`
                  : `开始${stageName}`)
            }
          </button>
        </div>
      </div>
    </div>
  )
}
