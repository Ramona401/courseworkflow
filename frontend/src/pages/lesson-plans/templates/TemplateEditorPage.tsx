/**
 * 提示词模板编辑器 — TemplateEditorPage
 *
 * Phase2核心页面：6模块编辑器
 * - system_prompt: 系统提示词（纯文本）
 * - context_rules: 上下文注入规则（JSON）
 * - generation_rules: 生成规则（JSON）
 * - review_rules: 评审规则（JSON）
 * - output_format: 输出格式（JSON）
 * - custom_instructions: 自定义指令（纯文本）
 *
 * 功能：
 * - 加载模板详情并填充表单
 * - 6个模块Tab切换编辑
 * - 查看继承链解析结果（resolved）
 * - 保存更新模板
 * - 基本信息编辑（名称/描述/学科/年级/默认标记）
 *
 * PRD §8.2 配色 + §8.3 动效
 */
import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  getPromptTemplate,
  updatePromptTemplate,
  resolvePromptTemplate,
  type PromptTemplate,
  type ResolvedPromptTemplate,
  type UpdatePromptTemplateRequest,
} from '@/api/lesson-plans'

/* ==================== 样式常量 ==================== */
const COLORS = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  primaryBorder: 'rgba(79,123,232,0.2)',
  accent: '#F59E0B',
  success: '#10B981',
  danger: '#EF4444',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  bg: '#FAFBFC',
  card: '#FFFFFF',
  border: '#F3F4F6',
  borderHover: '#E5E7EB',
}

/** 6个编辑模块定义 */
const MODULES = [
  { key: 'system_prompt', label: '系统提示词', icon: '🧠', type: 'text' as const, desc: 'AI角色设定和核心行为准则' },
  { key: 'context_rules', label: '上下文规则', icon: '📥', type: 'json' as const, desc: '组件注入策略和上下文管理' },
  { key: 'generation_rules', label: '生成规则', icon: '⚙️', type: 'json' as const, desc: '教案生成风格和约束条件' },
  { key: 'review_rules', label: '评审规则', icon: '📋', type: 'json' as const, desc: 'AI评审维度和评分标准' },
  { key: 'output_format', label: '输出格式', icon: '📄', type: 'json' as const, desc: '教案输出格式和结构要求' },
  { key: 'custom_instructions', label: '自定义指令', icon: '✏️', type: 'text' as const, desc: '补充说明和特殊要求' },
] as const

/** 模板层级配色映射 */
const LEVEL_COLORS: Record<string, { color: string; label: string }> = {
  region: { color: '#7C3AED', label: '区域级' },
  school: { color: '#4F7BE8', label: '学校级' },
  group: { color: '#F59E0B', label: '教研组级' },
  personal: { color: '#10B981', label: '个人级' },
}

type ModuleKey = typeof MODULES[number]['key']

export default function TemplateEditorPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  /* ==================== 状态 ==================== */
  const [template, setTemplate] = useState<PromptTemplate | null>(null)
  const [resolved, setResolved] = useState<ResolvedPromptTemplate | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saveMsg, setSaveMsg] = useState<string | null>(null)
  const [activeModule, setActiveModule] = useState<ModuleKey>('system_prompt')
  const [showResolved, setShowResolved] = useState(false)

  /* 表单状态 — 基本信息 */
  const [formName, setFormName] = useState('')
  const [formDesc, setFormDesc] = useState('')
  const [formSubject, setFormSubject] = useState('')
  const [formGrade, setFormGrade] = useState('')
  const [formDefault, setFormDefault] = useState(false)

  /* 表单状态 — 6个模块内容 */
  const [formSystemPrompt, setFormSystemPrompt] = useState('')
  const [formContextRules, setFormContextRules] = useState('')
  const [formGenerationRules, setFormGenerationRules] = useState('')
  const [formReviewRules, setFormReviewRules] = useState('')
  const [formOutputFormat, setFormOutputFormat] = useState('')
  const [formCustomInstructions, setFormCustomInstructions] = useState('')

  /** JSON安全格式化 */
  const safeJsonStringify = (obj: unknown): string => {
    if (!obj || (typeof obj === 'object' && Object.keys(obj as Record<string, unknown>).length === 0)) return ''
    try { return JSON.stringify(obj, null, 2) } catch { return '' }
  }

  /** JSON安全解析（返回null表示无效） */
  const safeJsonParse = (str: string): Record<string, unknown> | null => {
    if (!str.trim()) return {}
    try { return JSON.parse(str) } catch { return null }
  }

  /* ==================== 加载数据 ==================== */
  const loadTemplate = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError(null)
    try {
      const data = await getPromptTemplate(id)
      setTemplate(data)
      /* 填充基本信息 */
      setFormName(data.name || '')
      setFormDesc(data.description || '')
      setFormSubject(data.subject || '')
      setFormGrade(data.grade_range || '')
      setFormDefault(data.is_default || false)
      /* 填充6个模块 */
      setFormSystemPrompt(data.system_prompt || '')
      setFormContextRules(safeJsonStringify(data.context_rules))
      setFormGenerationRules(safeJsonStringify(data.generation_rules))
      setFormReviewRules(safeJsonStringify(data.review_rules))
      setFormOutputFormat(safeJsonStringify(data.output_format))
      setFormCustomInstructions(data.custom_instructions || '')
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载模板失败')
    } finally {
      setLoading(false)
    }
  }, [id])

  /** 加载继承链解析结果 */
  const loadResolved = useCallback(async () => {
    if (!id) return
    try {
      const data = await resolvePromptTemplate(id)
      setResolved(data)
    } catch (e: unknown) {
      console.error('加载继承链解析失败:', e)
    }
  }, [id])

  useEffect(() => { loadTemplate() }, [loadTemplate])

  /* ==================== 保存 ==================== */
  const handleSave = async () => {
    if (!id || !template) return
    setSaving(true)
    setSaveMsg(null)
    setError(null)

    /* 校验JSON字段 */
    const jsonFields = [
      { key: 'context_rules', value: formContextRules, label: '上下文规则' },
      { key: 'generation_rules', value: formGenerationRules, label: '生成规则' },
      { key: 'review_rules', value: formReviewRules, label: '评审规则' },
      { key: 'output_format', value: formOutputFormat, label: '输出格式' },
    ]
    for (const f of jsonFields) {
      if (f.value.trim() && safeJsonParse(f.value) === null) {
        setError(`${f.label} JSON格式无效，请检查`)
        setSaving(false)
        return
      }
    }

    try {
      const req: UpdatePromptTemplateRequest = {
        name: formName || undefined,
        description: formDesc || undefined,
        subject: formSubject || undefined,
        grade_range: formGrade || undefined,
        is_default: formDefault,
        system_prompt: formSystemPrompt || null,
        context_rules: formContextRules.trim() ? safeJsonParse(formContextRules) as Record<string, unknown> : undefined,
        generation_rules: formGenerationRules.trim() ? safeJsonParse(formGenerationRules) as Record<string, unknown> : undefined,
        review_rules: formReviewRules.trim() ? safeJsonParse(formReviewRules) as Record<string, unknown> : undefined,
        output_format: formOutputFormat.trim() ? safeJsonParse(formOutputFormat) as Record<string, unknown> : undefined,
        custom_instructions: formCustomInstructions || null,
      }
      await updatePromptTemplate(id, req)
      setSaveMsg('保存成功')
      /* 重新加载以获取最新数据 */
      await loadTemplate()
      setTimeout(() => setSaveMsg(null), 3000)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  /* ==================== 获取当前模块的值和setter ==================== */
  const getModuleValue = (key: ModuleKey): string => {
    const map: Record<ModuleKey, string> = {
      system_prompt: formSystemPrompt,
      context_rules: formContextRules,
      generation_rules: formGenerationRules,
      review_rules: formReviewRules,
      output_format: formOutputFormat,
      custom_instructions: formCustomInstructions,
    }
    return map[key]
  }

  const setModuleValue = (key: ModuleKey, value: string) => {
    const setters: Record<ModuleKey, (v: string) => void> = {
      system_prompt: setFormSystemPrompt,
      context_rules: setFormContextRules,
      generation_rules: setFormGenerationRules,
      review_rules: setFormReviewRules,
      output_format: setFormOutputFormat,
      custom_instructions: setFormCustomInstructions,
    }
    setters[key](value)
  }

  const currentModule = MODULES.find(m => m.key === activeModule)!
  const levelInfo = template ? LEVEL_COLORS[template.level] || { color: '#6B7280', label: template.level } : null

  /* ==================== 渲染 ==================== */

  if (loading) {
    return (
      <div style={{ padding: '60px', textAlign: 'center' }}>
        <div style={{ fontSize: '14px', color: COLORS.textMuted }}>加载模板中...</div>
      </div>
    )
  }

  if (error && !template) {
    return (
      <div style={{ padding: '60px', textAlign: 'center' }}>
        <div style={{ fontSize: '14px', color: COLORS.danger, marginBottom: '16px' }}>{error}</div>
        <button onClick={() => navigate('/lesson-plans/templates')} style={linkBtnStyle}>
          ← 返回模板列表
        </button>
      </div>
    )
  }

  return (
    <div>
      {/* ==================== 顶部导航栏 ==================== */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        marginBottom: '20px', flexWrap: 'wrap', gap: '12px',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <button onClick={() => navigate('/lesson-plans/templates')} style={linkBtnStyle}>
            ← 返回
          </button>
          <h1 style={{ fontSize: '18px', fontWeight: 600, color: COLORS.textPrimary, margin: 0 }}>
            编辑模板
          </h1>
          {levelInfo && (
            <span style={{
              fontSize: '11px', fontWeight: 600, color: levelInfo.color,
              background: levelInfo.color + '15', padding: '3px 10px', borderRadius: '6px',
            }}>{levelInfo.label}</span>
          )}
          {template?.is_default && (
            <span style={{
              fontSize: '11px', fontWeight: 600, color: COLORS.accent,
              background: COLORS.accent + '15', padding: '3px 10px', borderRadius: '6px',
            }}>默认模板</span>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          {/* 查看继承链按钮 */}
          <button
            onClick={() => { setShowResolved(!showResolved); if (!resolved) loadResolved() }}
            style={{
              ...secondaryBtnStyle,
              background: showResolved ? COLORS.primaryLight : 'transparent',
            }}
          >🔗 继承链</button>
          {/* 保存状态提示 */}
          {saveMsg && <span style={{ fontSize: '13px', color: COLORS.success, fontWeight: 500 }}>✓ {saveMsg}</span>}
          {error && template && <span style={{ fontSize: '13px', color: COLORS.danger }}>{error}</span>}
          {/* 保存按钮 */}
          <button onClick={handleSave} disabled={saving} style={{
            ...primaryBtnStyle,
            opacity: saving ? 0.6 : 1,
            cursor: saving ? 'not-allowed' : 'pointer',
          }}>
            {saving ? '保存中...' : '💾 保存'}
          </button>
        </div>
      </div>

      {/* ==================== 继承链解析面板（可折叠） ==================== */}
      {showResolved && (
        <div style={{
          background: COLORS.card, borderRadius: '12px', padding: '20px',
          border: `1px solid ${COLORS.border}`, marginBottom: '16px',
        }}>
          <div style={{ fontSize: '14px', fontWeight: 600, color: COLORS.textPrimary, marginBottom: '12px' }}>
            🔗 继承链解析结果
          </div>
          {resolved ? (
            <div>
              {/* 继承链路径 */}
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center', marginBottom: '12px', flexWrap: 'wrap' }}>
                {resolved.chain.map((node, i) => {
                  const nColor = LEVEL_COLORS[node.level]?.color || '#6B7280'
                  return (
                    <span key={node.id} style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <span style={{
                        fontSize: '12px', padding: '3px 10px', borderRadius: '6px',
                        background: nColor + '15', color: nColor, fontWeight: 500,
                      }}>
                        {LEVEL_COLORS[node.level]?.label || node.level}: {node.name}
                      </span>
                      {i < resolved.chain.length - 1 && <span style={{ color: COLORS.textMuted }}>→</span>}
                    </span>
                  )
                })}
              </div>
              {/* 合并后的system_prompt预览 */}
              <div style={{ fontSize: '12px', color: COLORS.textSecondary, marginBottom: '6px' }}>
                合并后的system_prompt（前200字）：
              </div>
              <div style={{
                fontSize: '13px', color: COLORS.textPrimary, background: '#F9FAFB',
                padding: '12px', borderRadius: '8px', lineHeight: 1.6,
                maxHeight: '120px', overflow: 'auto', whiteSpace: 'pre-wrap',
              }}>
                {resolved.system_prompt ? resolved.system_prompt.slice(0, 200) + (resolved.system_prompt.length > 200 ? '...' : '') : '（空）'}
              </div>
            </div>
          ) : (
            <div style={{ fontSize: '13px', color: COLORS.textMuted }}>加载中...</div>
          )}
        </div>
      )}

      {/* ==================== 基本信息卡片 ==================== */}
      <div style={{
        background: COLORS.card, borderRadius: '12px', padding: '20px',
        border: `1px solid ${COLORS.border}`, marginBottom: '16px',
      }}>
        <div style={{ fontSize: '14px', fontWeight: 600, color: COLORS.textPrimary, marginBottom: '16px' }}>
          📝 基本信息
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
          {/* 名称 */}
          <div style={{ gridColumn: '1 / -1' }}>
            <label style={labelStyle}>模板名称</label>
            <input
              value={formName} onChange={e => setFormName(e.target.value)}
              placeholder="输入模板名称" style={inputStyle}
            />
          </div>
          {/* 描述 */}
          <div style={{ gridColumn: '1 / -1' }}>
            <label style={labelStyle}>描述</label>
            <input
              value={formDesc} onChange={e => setFormDesc(e.target.value)}
              placeholder="简要描述模板用途" style={inputStyle}
            />
          </div>
          {/* 学科 */}
          <div>
            <label style={labelStyle}>学科</label>
            <input
              value={formSubject} onChange={e => setFormSubject(e.target.value)}
              placeholder="如 AI" style={inputStyle}
            />
          </div>
          {/* 年级范围 */}
          <div>
            <label style={labelStyle}>年级范围</label>
            <input
              value={formGrade} onChange={e => setFormGrade(e.target.value)}
              placeholder="如 7-9" style={inputStyle}
            />
          </div>
          {/* 默认模板 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <input
              type="checkbox" checked={formDefault}
              onChange={e => setFormDefault(e.target.checked)}
              style={{ width: '16px', height: '16px', accentColor: COLORS.primary }}
            />
            <label style={{ fontSize: '13px', color: COLORS.textSecondary, cursor: 'pointer' }}
              onClick={() => setFormDefault(!formDefault)}>
              设为该层级默认模板
            </label>
          </div>
        </div>
      </div>

      {/* ==================== 6模块编辑器 ==================== */}
      <div style={{
        background: COLORS.card, borderRadius: '12px',
        border: `1px solid ${COLORS.border}`, overflow: 'hidden',
      }}>
        {/* Tab栏 */}
        <div style={{
          display: 'flex', borderBottom: `1px solid ${COLORS.border}`,
          overflowX: 'auto', flexShrink: 0,
        }}>
          {MODULES.map(m => {
            const isActive = activeModule === m.key
            const hasContent = !!getModuleValue(m.key).trim()
            return (
              <button
                key={m.key} onClick={() => setActiveModule(m.key)}
                style={{
                  display: 'flex', alignItems: 'center', gap: '6px',
                  padding: '14px 18px', border: 'none', cursor: 'pointer',
                  fontSize: '13px', fontWeight: isActive ? 600 : 400, whiteSpace: 'nowrap',
                  color: isActive ? COLORS.primary : COLORS.textSecondary,
                  background: isActive ? COLORS.primaryLight : 'transparent',
                  borderBottom: isActive ? `2px solid ${COLORS.primary}` : '2px solid transparent',
                  transition: 'all 200ms ease',
                }}
              >
                <span>{m.icon}</span>
                <span>{m.label}</span>
                {/* 有内容指示点 */}
                {hasContent && (
                  <span style={{
                    width: '6px', height: '6px', borderRadius: '50%',
                    background: isActive ? COLORS.primary : COLORS.success,
                    flexShrink: 0,
                  }} />
                )}
              </button>
            )
          })}
        </div>

        {/* 编辑区域 */}
        <div style={{ padding: '20px' }}>
          {/* 模块描述 */}
          <div style={{
            fontSize: '12px', color: COLORS.textMuted, marginBottom: '12px',
            display: 'flex', alignItems: 'center', gap: '6px',
          }}>
            <span>{currentModule.icon}</span>
            <span>{currentModule.desc}</span>
            {currentModule.type === 'json' && (
              <span style={{
                fontSize: '10px', color: COLORS.accent, background: COLORS.accent + '15',
                padding: '1px 6px', borderRadius: '4px', fontWeight: 600,
              }}>JSON</span>
            )}
            {/* 空值提示：继承自父级 */}
            {!getModuleValue(activeModule).trim() && template?.parent_template_id && (
              <span style={{
                fontSize: '10px', color: '#7C3AED', background: 'rgba(124,58,237,0.1)',
                padding: '1px 6px', borderRadius: '4px',
              }}>将继承父级</span>
            )}
          </div>

          {/* 文本区域 */}
          <textarea
            value={getModuleValue(activeModule)}
            onChange={e => setModuleValue(activeModule, e.target.value)}
            placeholder={
              currentModule.type === 'json'
                ? '输入JSON格式配置，留空则继承父级模板...\n例如:\n{\n  "key": "value"\n}'
                : '输入文本内容，留空则继承父级模板...'
            }
            style={{
              width: '100%', minHeight: currentModule.type === 'json' ? '320px' : '240px',
              padding: '14px 16px', borderRadius: '8px',
              border: `1px solid ${COLORS.border}`, outline: 'none',
              fontSize: '14px', lineHeight: 1.7, color: COLORS.textPrimary,
              fontFamily: currentModule.type === 'json' ? "'SF Mono', 'Fira Code', 'Consolas', monospace" : 'inherit',
              background: '#FAFBFC', resize: 'vertical',
              transition: 'border-color 200ms ease',
            }}
            onFocus={e => { e.currentTarget.style.borderColor = COLORS.primary }}
            onBlur={e => { e.currentTarget.style.borderColor = COLORS.border }}
          />

          {/* JSON校验提示 */}
          {currentModule.type === 'json' && getModuleValue(activeModule).trim() && (
            <div style={{ marginTop: '8px', fontSize: '12px' }}>
              {safeJsonParse(getModuleValue(activeModule)) !== null ? (
                <span style={{ color: COLORS.success }}>✓ JSON格式有效</span>
              ) : (
                <span style={{ color: COLORS.danger }}>✗ JSON格式无效，请检查</span>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

/* ==================== 样式常量 ==================== */
const labelStyle: React.CSSProperties = {
  display: 'block', fontSize: '12px', fontWeight: 500,
  color: '#6B7280', marginBottom: '6px',
}

const inputStyle: React.CSSProperties = {
  width: '100%', padding: '10px 14px', borderRadius: '8px',
  border: '1px solid #F3F4F6', outline: 'none', fontSize: '14px',
  color: '#1F2937', background: '#FAFBFC', transition: 'border-color 200ms ease',
}

const primaryBtnStyle: React.CSSProperties = {
  padding: '9px 20px', borderRadius: '8px', border: 'none',
  background: '#4F7BE8', color: '#FFFFFF', fontSize: '13px',
  fontWeight: 600, cursor: 'pointer', transition: 'all 200ms ease',
}

const secondaryBtnStyle: React.CSSProperties = {
  padding: '9px 16px', borderRadius: '8px',
  border: '1px solid #F3F4F6', background: 'transparent',
  color: '#6B7280', fontSize: '13px', fontWeight: 400,
  cursor: 'pointer', transition: 'all 200ms ease',
}

const linkBtnStyle: React.CSSProperties = {
  background: 'none', border: 'none', color: '#6B7280',
  fontSize: '13px', cursor: 'pointer', padding: '4px 0',
}
