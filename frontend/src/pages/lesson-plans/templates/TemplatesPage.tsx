/**
 * 提示词模板列表页 — TemplatesPage
 *
 * Phase2核心页面：展示四级继承链 + 模板列表 + 跳转编辑器
 * - 继承链可视化（region→school→group→personal）
 * - 从后端API加载真实模板数据
 * - 点击模板卡片进入编辑器
 *
 * PRD §1.3：分层继承，上级设标准下级可覆盖
 * PRD §8.2 配色 + §8.3 动效
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { getPromptTemplates, type PromptTemplate, type TemplateLevel } from '@/api/lesson-plans'

/* ==================== 样式常量 ==================== */
const COLORS = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent: '#F59E0B',
  success: '#10B981',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  card: '#FFFFFF',
  border: '#F3F4F6',
  bgHover: '#F9FAFB',
}

/** 四级模板层级定义 */
const LEVELS: Array<{
  level: TemplateLevel; label: string; icon: string; desc: string; color: string
}> = [
  { level: 'region', label: '区域级', icon: '🏛️', desc: '区域管理员设置，适用于所有下属学校', color: '#7C3AED' },
  { level: 'school', label: '学校级', icon: '🏫', desc: '学校管理员设置，可覆盖区域级配置', color: '#4F7BE8' },
  { level: 'group', label: '教研组级', icon: '👥', desc: '教研组长设置，可覆盖学校级配置', color: '#F59E0B' },
  { level: 'personal', label: '个人级', icon: '👤', desc: '教师个人设置，最高优先级覆盖', color: '#10B981' },
]

export default function TemplatesPage() {
  const navigate = useNavigate()
  const [templates, setTemplates] = useState<PromptTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [hoveredId, setHoveredId] = useState<string | null>(null)

  /* ==================== 加载模板列表 ==================== */
  useEffect(() => {
    async function load() {
      setLoading(true)
      setError(null)
      try {
        const data = await getPromptTemplates()
        setTemplates(data.items || [])
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : '加载模板列表失败')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  /** 按层级分组模板 */
  const groupedByLevel = LEVELS.map(l => ({
    ...l,
    items: templates.filter(t => t.level === l.level),
  }))

  /** 统计填充率（6模块中非空字段数） */
  const getModuleFillCount = (t: PromptTemplate): number => {
    let count = 0
    if (t.system_prompt) count++
    if (t.context_rules && Object.keys(t.context_rules).length > 0) count++
    if (t.generation_rules && Object.keys(t.generation_rules).length > 0) count++
    if (t.review_rules && Object.keys(t.review_rules).length > 0) count++
    if (t.output_format && Object.keys(t.output_format).length > 0) count++
    if (t.custom_instructions) count++
    return count
  }

  return (
    <div>
      {/* 页面标题 */}
      <div style={{ marginBottom: '24px' }}>
        <h1 style={{ fontSize: '20px', fontWeight: 600, color: COLORS.textPrimary, margin: '0 0 8px 0' }}>
          提示词模板
        </h1>
        <p style={{ fontSize: '14px', color: COLORS.textSecondary, margin: 0 }}>
          管理四级继承链提示词模板，子级可覆盖父级配置
        </p>
      </div>

      {/* ==================== 继承链可视化 ==================== */}
      <div style={{
        background: COLORS.card, borderRadius: '12px', padding: '24px',
        border: `1px solid ${COLORS.border}`, marginBottom: '24px',
      }}>
        <div style={{ fontSize: '15px', fontWeight: 600, color: COLORS.textPrimary, marginBottom: '16px' }}>
          📐 模板继承链
        </div>
        <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', alignItems: 'center' }}>
          {LEVELS.map((l, i) => (
            <div key={l.level} style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <div style={{
                background: COLORS.card, borderRadius: '12px', padding: '14px 18px',
                border: `2px solid ${l.color}25`, minWidth: '160px',
              }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
                  <span style={{ fontSize: '18px' }}>{l.icon}</span>
                  <span style={{ fontSize: '14px', fontWeight: 600, color: COLORS.textPrimary }}>{l.label}</span>
                  <span style={{
                    fontSize: '10px', fontWeight: 600, color: l.color,
                    background: l.color + '15', padding: '2px 8px', borderRadius: '6px',
                  }}>{l.level}</span>
                </div>
                <div style={{ fontSize: '11px', color: COLORS.textMuted }}>
                  {groupedByLevel[i].items.length} 个模板
                </div>
              </div>
              {i < LEVELS.length - 1 && <span style={{ fontSize: '18px', color: '#D1D5DB' }}>→</span>}
            </div>
          ))}
        </div>
        <div style={{
          marginTop: '14px', fontSize: '12px', color: COLORS.textMuted,
          background: '#F9FAFB', padding: '10px 14px', borderRadius: '8px',
        }}>
          💡 继承规则：子级非空字段覆盖父级，空字段自动从父级继承。最多10层防止死循环。
        </div>
      </div>

      {/* ==================== 模板列表（按层级分组） ==================== */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '40px', color: COLORS.textMuted, fontSize: '14px' }}>
          加载中...
        </div>
      ) : error ? (
        <div style={{ textAlign: 'center', padding: '40px', color: '#EF4444', fontSize: '14px' }}>
          {error}
        </div>
      ) : templates.length === 0 ? (
        <div style={{
          background: COLORS.card, borderRadius: '12px', padding: '60px 40px',
          border: `1px solid ${COLORS.border}`, textAlign: 'center',
        }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>📐</div>
          <div style={{ fontSize: '16px', fontWeight: 500, color: COLORS.textPrimary, marginBottom: '8px' }}>
            暂无模板数据
          </div>
          <div style={{ fontSize: '14px', color: COLORS.textMuted }}>
            请先通过种子数据或手动创建初始模板
          </div>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
          {groupedByLevel.map(group => (
            group.items.length > 0 && (
              <div key={group.level}>
                {/* 层级标题 */}
                <div style={{
                  display: 'flex', alignItems: 'center', gap: '8px',
                  marginBottom: '12px',
                }}>
                  <span style={{ fontSize: '16px' }}>{group.icon}</span>
                  <span style={{ fontSize: '15px', fontWeight: 600, color: COLORS.textPrimary }}>
                    {group.label}
                  </span>
                  <span style={{
                    fontSize: '11px', color: group.color, fontWeight: 600,
                    background: group.color + '15', padding: '2px 8px', borderRadius: '6px',
                  }}>{group.items.length}</span>
                </div>

                {/* 模板卡片列表 */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                  {group.items.map(t => {
                    const hovered = hoveredId === t.id
                    const fillCount = getModuleFillCount(t)
                    return (
                      <div
                        key={t.id}
                        onClick={() => navigate(`/lesson-plans/templates/${t.id}`)}
                        onMouseEnter={() => setHoveredId(t.id)}
                        onMouseLeave={() => setHoveredId(null)}
                        style={{
                          background: COLORS.card, borderRadius: '12px',
                          padding: '18px 22px', cursor: 'pointer',
                          border: `1px solid ${hovered ? group.color + '40' : COLORS.border}`,
                          transition: 'all 200ms ease',
                          transform: hovered ? 'translateY(-1px)' : 'none',
                          boxShadow: hovered ? '0 4px 12px rgba(0,0,0,0.06)' : 'none',
                        }}
                      >
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                          <div style={{ flex: 1 }}>
                            {/* 模板名称 + 标签 */}
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
                              <span style={{ fontSize: '15px', fontWeight: 600, color: COLORS.textPrimary }}>
                                {t.name}
                              </span>
                              {t.is_default && (
                                <span style={{
                                  fontSize: '10px', fontWeight: 600, color: COLORS.accent,
                                  background: COLORS.accent + '15', padding: '2px 8px', borderRadius: '4px',
                                }}>默认</span>
                              )}
                              {t.subject && (
                                <span style={{
                                  fontSize: '10px', color: COLORS.textMuted,
                                  background: '#F3F4F6', padding: '2px 8px', borderRadius: '4px',
                                }}>{t.subject}</span>
                              )}
                              {t.grade_range && (
                                <span style={{
                                  fontSize: '10px', color: COLORS.textMuted,
                                  background: '#F3F4F6', padding: '2px 8px', borderRadius: '4px',
                                }}>{t.grade_range}年级</span>
                              )}
                            </div>
                            {/* 描述 */}
                            {t.description && (
                              <div style={{ fontSize: '13px', color: COLORS.textSecondary, lineHeight: 1.5 }}>
                                {t.description}
                              </div>
                            )}
                          </div>
                          {/* 右侧：模块填充率 */}
                          <div style={{ textAlign: 'right', flexShrink: 0, marginLeft: '20px' }}>
                            <div style={{ fontSize: '12px', color: COLORS.textMuted, marginBottom: '4px' }}>
                              模块配置
                            </div>
                            <div style={{ display: 'flex', gap: '3px' }}>
                              {[0, 1, 2, 3, 4, 5].map(i => (
                                <div key={i} style={{
                                  width: '8px', height: '8px', borderRadius: '2px',
                                  background: i < fillCount ? COLORS.success : '#E5E7EB',
                                }} />
                              ))}
                            </div>
                            <div style={{ fontSize: '11px', color: COLORS.textMuted, marginTop: '2px' }}>
                              {fillCount}/6
                            </div>
                          </div>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            )
          ))}
        </div>
      )}
    </div>
  )
}
