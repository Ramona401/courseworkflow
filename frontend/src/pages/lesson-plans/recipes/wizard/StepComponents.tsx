/**
 * StepComponents — 配方向导步骤3：教学组件选择
 *
 * v79 新增：分步向导式配方创建
 * v79-4 增强：组件展示优化——类型说明+设计逻辑展示+展开详情+阶段标注
 *
 * 设计目标：
 *   - 让老师理解每类组件"帮我解决什么问题"
 *   - 每个组件显示设计逻辑摘要（不只是label+分数）
 *   - 点击展开可看完整说明
 *   - 组件类型标注适用的备课阶段
 *   - 可跳过（不选任何组件时备课时AI自动匹配）
 */
import { useState, useEffect, useCallback } from 'react'
import {
  smartRecommendComponents,
  type RecommendedComponentGroup,
} from '@/api/recipes'
import {
  C, stepCardStyle,
  type WizardFormData,
} from './wizardConstants'

/* ==================== 组件类型说明映射 ==================== */

/** 每种组件类型的说明信息：图标、一句话描述、适用阶段 */
const COMPONENT_TYPE_INFO: Record<string, {
  icon: string       // 图标
  desc: string       // 一句话：这类组件帮你做什么
  stage: string      // 适用的备课阶段
}> = {
  curriculum_standard: {
    icon: '📐',
    desc: '帮AI理解课标要求，确保教学目标对齐国家标准',
    stage: '教学分析',
  },
  knowledge_graph: {
    icon: '🧠',
    desc: '帮AI梳理知识脉络，设计合理的教学递进路径',
    stage: '教学分析',
  },
  student_profile: {
    icon: '👥',
    desc: '帮AI了解学生特点，调整教学难度和活动设计',
    stage: '教学分析',
  },
  pedagogy: {
    icon: '📖',
    desc: '提供经过验证的教学策略，让AI设计更专业',
    stage: '教学设计',
  },
  activity_design: {
    icon: '🎯',
    desc: '提供具体的课堂活动方案，让教案更有操作性',
    stage: '教学设计',
  },
  questioning_strategy: {
    icon: '❓',
    desc: '提供提问技巧模板，帮AI设计高质量课堂提问',
    stage: '教学设计',
  },
  assessment_strategy: {
    icon: '📊',
    desc: '提供评价方法，帮AI设计课堂检测和学习评价',
    stage: '教学设计',
  },
  cross_subject: {
    icon: '🔗',
    desc: '提供跨学科整合思路，丰富教学内容和视角',
    stage: '教学设计',
  },
  teaching_tool: {
    icon: '🛠️',
    desc: '推荐教学工具和资源，让教案落地更方便',
    stage: '教案撰写',
  },
  scenario_material: {
    icon: '🎬',
    desc: '提供情境素材和案例，让课堂导入更生动',
    stage: '教案撰写',
  },
  quality_rubric: {
    icon: '✅',
    desc: '提供教案质量标准，帮AI评审时有据可依',
    stage: 'AI评审',
  },
  review_rubric: {
    icon: '📝',
    desc: '提供评审维度和标准，让评审结果更具参考价值',
    stage: 'AI评审',
  },
  design_defect: {
    icon: '⚠️',
    desc: '收录常见教学设计失误，帮AI评审时精准发现问题',
    stage: 'AI评审',
  },
}

/** 阶段颜色映射 */
const STAGE_COLORS: Record<string, { bg: string; color: string }> = {
  '教学分析': { bg: 'rgba(59,130,246,0.08)', color: '#3B82F6' },
  '教学设计': { bg: 'rgba(139,92,246,0.08)', color: '#8B5CF6' },
  '教案撰写': { bg: 'rgba(245,158,11,0.08)', color: '#F59E0B' },
  'AI评审':   { bg: 'rgba(16,185,129,0.08)', color: '#10B981' },
}

/* ==================== Props 类型 ==================== */
interface StepComponentsProps {
  formData: WizardFormData
  updateForm: (updates: Partial<WizardFormData>) => void
}

/* ==================== 组件 ==================== */
export default function StepComponents({ formData, updateForm }: StepComponentsProps) {
  const [compGroups, setCompGroups] = useState<RecommendedComponentGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  // 当前展开详情的组件ID
  const [expandedId, setExpandedId] = useState<string | null>(null)

  // 加载推荐组件
  const loadRecommend = useCallback(async () => {
    if (!formData.subject || !formData.gradeRange) return
    setLoading(true)
    setError('')
    try {
      const resp = await smartRecommendComponents({
        subject: formData.subject,
        grade_range: formData.gradeRange,
      })
      setCompGroups(resp.groups || [])
    } catch (e) {
      console.error('加载推荐组件失败:', e)
      setError('加载推荐组件失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }, [formData.subject, formData.gradeRange])

  useEffect(() => { loadRecommend() }, [loadRecommend])

  // 切换单个组件选中
  const toggleComp = (compId: string) => {
    const next = new Set(formData.selectedCompIds)
    if (next.has(compId)) { next.delete(compId) } else { next.add(compId) }
    updateForm({ selectedCompIds: next })
  }

  // 全选/取消某组
  const selectAllInGroup = (group: RecommendedComponentGroup) => {
    const next = new Set(formData.selectedCompIds)
    group.components.forEach(c => next.add(c.id))
    updateForm({ selectedCompIds: next })
  }
  const deselectAllInGroup = (group: RecommendedComponentGroup) => {
    const next = new Set(formData.selectedCompIds)
    group.components.forEach(c => next.delete(c.id))
    updateForm({ selectedCompIds: next })
  }

  const groupSelectedCount = (group: RecommendedComponentGroup) =>
    group.components.filter(c => formData.selectedCompIds.has(c.id)).length

  const totalComps = compGroups.reduce((s, g) => s + g.components.length, 0)

  // 截断文本到指定长度
  const truncate = (text: string, maxLen: number) => {
    if (!text) return ''
    // 取前maxLen字符，如果有换行只取第一段
    const firstPara = text.split('\n').filter(l => l.trim())[0] || ''
    const t = firstPara.length > maxLen ? firstPara.slice(0, maxLen) + '...' : firstPara
    return t
  }

  return (
    <div style={stepCardStyle}>
      {/* 顶部提示 */}
      <div style={{
        padding: '14px 16px', borderRadius: '10px', marginBottom: '20px',
        background: 'rgba(79,123,232,0.06)', border: '1px solid rgba(79,123,232,0.12)',
      }}>
        <div style={{ fontSize: '14px', fontWeight: 600, color: C.primary, marginBottom: '6px' }}>
          🧩 什么是教学组件？
        </div>
        <div style={{ fontSize: '13px', color: '#4B5563', lineHeight: 1.7 }}>
          教学组件是经过沉淀的<strong>备课知识库</strong>——包括课标解读、教学策略、活动方案、评价标准等。
          选择组件后，AI备课时会<strong>自动参考这些专业知识</strong>，让教案更规范、更有深度。
        </div>
        <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '6px' }}>
          💡 不选也没关系，备课时AI会根据学科和年级自动匹配。选了可以更精准。
        </div>
      </div>

      {/* 已选统计 + 刷新 */}
      <div style={{
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        marginBottom: '16px',
      }}>
        <div style={{ fontSize: '14px', color: C.text }}>
          已选 <span style={{ fontWeight: 700, color: C.primary }}>{formData.selectedCompIds.size}</span> 个组件
          {totalComps > 0 && (
            <span style={{ color: C.textMuted }}> / 共推荐 {totalComps} 个</span>
          )}
        </div>
        <button
          onClick={loadRecommend}
          disabled={loading}
          style={{
            fontSize: '12px', color: C.primary, background: 'none',
            border: 'none', cursor: loading ? 'not-allowed' : 'pointer',
          }}
        >
          {loading ? '加载中...' : '🔄 刷新推荐'}
        </button>
      </div>

      {/* 加载状态 */}
      {loading && (
        <div style={{ padding: '40px', textAlign: 'center' }}>
          <div style={{
            width: '28px', height: '28px', border: `2px solid ${C.primary}`,
            borderTopColor: 'transparent', borderRadius: '50%',
            animation: 'spin 0.8s linear infinite', margin: '0 auto 12px',
          }} />
          <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
          <div style={{ fontSize: '13px', color: C.textMuted }}>
            正在根据「{formData.subject} · {formData.gradeRange}」智能推荐...
          </div>
        </div>
      )}

      {/* 错误 */}
      {error && (
        <div style={{
          padding: '16px', borderRadius: '8px', textAlign: 'center',
          background: '#FEF2F2', border: '1px solid #FECACA', color: C.danger,
          fontSize: '13px', marginBottom: '16px',
        }}>
          {error}
        </div>
      )}

      {/* 空状态 */}
      {!loading && !error && compGroups.length === 0 && (
        <div style={{
          padding: '40px', textAlign: 'center', color: C.textMuted, fontSize: '13px',
        }}>
          暂无推荐组件，你可以跳过此步，备课时AI会自动匹配
        </div>
      )}

      {/* 组件分组列表 */}
      {!loading && compGroups.map(group => {
        const selCount = groupSelectedCount(group)
        const allSelected = selCount === group.components.length && group.components.length > 0
        const typeInfo = COMPONENT_TYPE_INFO[group.library_type]
        const stageColor = typeInfo ? STAGE_COLORS[typeInfo.stage] : null

        return (
          <div key={group.library_type} style={{
            marginBottom: '16px', borderRadius: '12px',
            border: `1px solid ${C.border}`, overflow: 'hidden',
          }}>
            {/* 分组头部：类型图标+名称+说明+阶段标签+全选 */}
            <div style={{
              padding: '12px 14px', background: C.bg,
              borderBottom: `1px solid ${C.border}`,
            }}>
              <div style={{
                display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start',
              }}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  {/* 类型名称行 */}
                  <div style={{
                    display: 'flex', alignItems: 'center', gap: '6px',
                    marginBottom: '4px',
                  }}>
                    <span style={{ fontSize: '16px' }}>{typeInfo?.icon || '📋'}</span>
                    <span style={{
                      fontSize: '14px', fontWeight: 600, color: C.text,
                    }}>
                      {group.library_name}
                    </span>
                    {/* 阶段标签 */}
                    {typeInfo?.stage && stageColor && (
                      <span style={{
                        fontSize: '10px', padding: '2px 8px', borderRadius: '10px',
                        background: stageColor.bg, color: stageColor.color,
                        fontWeight: 500, whiteSpace: 'nowrap',
                      }}>
                        {typeInfo.stage}阶段用
                      </span>
                    )}
                    <span style={{
                      fontSize: '11px', color: C.textMuted,
                    }}>
                      ({selCount}/{group.components.length})
                    </span>
                  </div>
                  {/* 类型说明 */}
                  {typeInfo?.desc && (
                    <div style={{
                      fontSize: '12px', color: C.textSec, lineHeight: 1.5,
                      paddingLeft: '22px',
                    }}>
                      {typeInfo.desc}
                    </div>
                  )}
                </div>
                {/* 全选按钮 */}
                <button
                  onClick={() => allSelected ? deselectAllInGroup(group) : selectAllInGroup(group)}
                  style={{
                    fontSize: '11px', color: C.primary, background: allSelected ? C.primaryLight : 'none',
                    border: allSelected ? `1px solid ${C.primary}` : 'none',
                    borderRadius: '6px', padding: '4px 10px',
                    cursor: 'pointer', fontWeight: 500, whiteSpace: 'nowrap',
                    flexShrink: 0, marginTop: '2px',
                  }}
                >
                  {allSelected ? '✓ 已全选' : '全选'}
                </button>
              </div>
            </div>

            {/* 组件列表 */}
            <div style={{ padding: '6px 8px' }}>
              {group.components.map(comp => {
                const checked = formData.selectedCompIds.has(comp.id)
                const isExpanded = expandedId === comp.id
                const hasDetail = !!comp.design_logic

                return (
                  <div key={comp.id} style={{
                    marginBottom: '4px', borderRadius: '8px',
                    border: `1px solid ${checked ? 'rgba(79,123,232,0.25)' : 'transparent'}`,
                    background: checked ? C.primaryLight : 'transparent',
                    transition: 'all 120ms ease',
                  }}>
                    {/* 主行：勾选+名称+摘要 */}
                    <div style={{
                      display: 'flex', alignItems: 'flex-start', gap: '10px',
                      padding: '10px 10px', cursor: 'pointer',
                    }}
                      onClick={() => toggleComp(comp.id)}
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        readOnly
                        style={{
                          accentColor: C.primary, flexShrink: 0,
                          marginTop: '3px', cursor: 'pointer',
                        }}
                      />
                      <div style={{ flex: 1, minWidth: 0 }}>
                        {/* 组件名称 */}
                        <div style={{
                          fontSize: '13px', fontWeight: 500, color: C.text,
                          lineHeight: 1.5,
                        }}>
                          {comp.display_label}
                        </div>
                        {/* 设计逻辑摘要（截断显示） */}
                        {comp.design_logic && (
                          <div style={{
                            fontSize: '12px', color: C.textMuted, marginTop: '3px',
                            lineHeight: 1.5,
                          }}>
                            {truncate(String(comp.design_logic), 60)}
                          </div>
                        )}
                      </div>
                      {/* 展开/收起按钮 */}
                      {hasDetail && (
                        <button
                          onClick={e => {
                            e.stopPropagation()
                            setExpandedId(isExpanded ? null : comp.id)
                          }}
                          style={{
                            fontSize: '11px', color: C.primary,
                            background: 'none', border: 'none',
                            cursor: 'pointer', padding: '2px 6px',
                            flexShrink: 0, marginTop: '2px',
                          }}
                        >
                          {isExpanded ? '收起 ▲' : '详情 ▼'}
                        </button>
                      )}
                    </div>

                    {/* 展开详情区 */}
                    {isExpanded && hasDetail && (
                      <div style={{
                        margin: '0 10px 10px 36px',
                        padding: '10px 12px', borderRadius: '8px',
                        background: '#F9FAFB', border: `1px solid ${C.border}`,
                        fontSize: '12px', color: '#4B5563', lineHeight: 1.7,
                        whiteSpace: 'pre-wrap', maxHeight: '200px', overflowY: 'auto',
                      }}>
                        {String(comp.design_logic)}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        )
      })}
    </div>
  )
}
