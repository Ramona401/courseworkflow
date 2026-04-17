/**
 * ComponentEditModal — 组件创建/编辑弹窗
 *
 * 迭代4B-1 新增：
 *   - 支持13类组件选择
 *   - 四层内容编辑（展示标签+设计逻辑+参考案例+完整指引）
 *   - 标签管理（自由标签+风格标签）
 *   - 学科/年级/注入模式/可见范围选择
 *   - 创建模式和编辑模式（通过componentId区分）
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getComponent, createComponent, updateComponent,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  type LessonPlanComponent, type LibraryType, type InjectionMode,
} from '../../../api/lesson-plans'

/* ==================== 常量定义 ==================== */

const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981', danger: '#EF4444', accent: '#F59E0B',
  text: '#1F2937', textSec: '#6B7280', textMuted: '#9CA3AF',
  card: '#FFFFFF', border: '#F3F4F6', bg: '#FAFBFC',
}

/** 13类组件元数据 */
const LIBRARY_TYPES: { key: LibraryType; icon: string; name: string; desc: string }[] = [
  { key: 'curriculum_standard',  icon: '📖', name: '课标与能力框架库', desc: '课程标准、能力目标、核心素养要求' },
  { key: 'knowledge_graph',      icon: '🧠', name: '知识图谱库',       desc: '知识点结构、概念关系、前置知识' },
  { key: 'student_profile',      icon: '👤', name: '学情特征库',       desc: '学生认知水平、常见误区、兴趣特点' },
  { key: 'pedagogy',             icon: '🎓', name: '教学法库',         desc: '教学方法、策略、理论指导' },
  { key: 'assessment_strategy',  icon: '📊', name: '评估策略库',       desc: '评估方式、评价标准、反馈机制' },
  { key: 'activity_design',      icon: '🎯', name: '活动设计方案库',   desc: '教学活动方案、游戏化设计、项目式学习' },
  { key: 'questioning_strategy', icon: '❓', name: '提问引导策略库',   desc: '提问技巧、引导策略、追问方法' },
  { key: 'cross_subject',        icon: '🔗', name: '跨学科连接库',     desc: '学科交叉点、跨学科项目、综合实践' },
  { key: 'teaching_tool',        icon: '🛠️', name: '教学工具库',       desc: '教学软件、平台工具、硬件资源' },
  { key: 'scenario_material',    icon: '🎬', name: '素材情境库',       desc: '教学情境、案例素材、视频资源' },
  { key: 'quality_rubric',       icon: '✅', name: '质量评估标准库',   desc: '教案质量标准、评分维度' },
  { key: 'design_defect',        icon: '⚠️', name: '常见设计缺陷库',   desc: '常见教学设计问题、改进建议' },
  { key: 'review_rubric',        icon: '📋', name: '教案评审规则库',   desc: '评审规则、审核标准' },
]

/** 注入模式选项 */
const INJECTION_MODES: { value: InjectionMode; label: string; desc: string }[] = [
  { value: 'silent',    label: '静默注入',  desc: 'AI自动使用，老师无感' },
  { value: 'recommend', label: '推荐确认',  desc: '展示给老师，确认后使用' },
  { value: 'on_demand', label: '按需调用',  desc: 'AI按需匹配或老师主动选择' },
]

/** 可见范围选项 */
const SCOPE_OPTIONS: { value: string; label: string }[] = [
  { value: 'global',   label: '全局' },
  { value: 'school',   label: '学校' },
  { value: 'group',    label: '教研组' },
  { value: 'personal', label: '个人' },
]

const SUBJECTS = ['general','人工智能','语文','数学','英语','物理','化学','生物','历史','地理','政治','信息技术']
const GRADES = ['','七年级','八年级','九年级','高一','高二','高三','小学低段','小学中段','小学高段']

/** 预置风格标签（迭代4B标签体系） */
const STYLE_TAGS = [
  { value: 'style:beginner',  label: '新手友好' },
  { value: 'style:growing',   label: '适合成长型' },
  { value: 'style:mature',    label: '适合成熟型' },
  { value: 'collab:tool',     label: '工具型协作' },
  { value: 'collab:collaborative', label: '协作型' },
  { value: 'collab:guided',   label: '引导型' },
  { value: 'priority:activity_detail',   label: '活动细节丰富' },
  { value: 'priority:student_response',  label: '学生预期反应' },
  { value: 'priority:differentiation',   label: '分层教学' },
  { value: 'priority:assessment',        label: '评估完善' },
  { value: 'priority:engagement',        label: '互动性强' },
]

/* ==================== 样式辅助 ==================== */

const labelStyle: React.CSSProperties = { display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }
const inputStyle: React.CSSProperties = {
  width: '100%', padding: '9px 12px', borderRadius: '8px', border: `1px solid ${C.border}`,
  fontSize: '13px', color: C.text, outline: 'none', boxSizing: 'border-box', fontFamily: 'inherit',
  transition: 'border-color 150ms ease',
}
const textareaStyle: React.CSSProperties = { ...inputStyle, resize: 'vertical' as const, lineHeight: 1.6 }
const selectStyle: React.CSSProperties = { ...inputStyle, cursor: 'pointer', background: '#fff' }

/* ==================== Props ==================== */

interface ComponentEditModalProps {
  /** 组件ID（编辑模式）或null（创建模式） */
  componentId: string | null
  /** 预选的组件库类型（从概览卡片点击时传入） */
  presetLibraryType?: LibraryType
  /** 关闭弹窗回调 */
  onClose: () => void
  /** 保存成功回调（刷新列表用） */
  onSaved: () => void
}

/* ==================== 主组件 ==================== */

export default function ComponentEditModal({ componentId, presetLibraryType, onClose, onSaved }: ComponentEditModalProps) {
  const isEdit = !!componentId

  // ---- 表单状态 ----
  const [libraryType, setLibraryType] = useState<LibraryType>(presetLibraryType || 'activity_design')
  const [subject, setSubject] = useState('general')
  const [gradeRange, setGradeRange] = useState('')
  const [injectionMode, setInjectionMode] = useState<InjectionMode>('on_demand')
  const [scope, setScope] = useState('global')
  const [displayLabel, setDisplayLabel] = useState('')
  const [designLogic, setDesignLogic] = useState('')
  const [exampleSnippet, setExampleSnippet] = useState('')
  const [fullGuide, setFullGuide] = useState('')
  const [tagInput, setTagInput] = useState('')
  const [tags, setTags] = useState<string[]>([])

  // ---- 页面状态 ----
  const [loading, setLoading] = useState(isEdit)
  const [saving, setSaving] = useState(false)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type }); setTimeout(() => setToast(null), 3000)
  }

  // ---- 编辑模式：加载组件数据 ----
  useEffect(() => {
    if (!componentId) return
    const load = async () => {
      try {
        const comp = await getComponent(componentId)
        setLibraryType(comp.library_type)
        setSubject(comp.subject || 'general')
        setGradeRange(comp.grade_range || '')
        setInjectionMode(comp.injection_mode)
        setScope(comp.scope || 'global')
        setDisplayLabel(comp.display_label || '')
        setDesignLogic(comp.design_logic || '')
        setExampleSnippet(comp.example_snippet || '')
        setFullGuide(comp.full_guide || '')
        // 解析标签
        try {
          const parsed = typeof comp.tags === 'string' ? JSON.parse(comp.tags) : comp.tags
          if (Array.isArray(parsed)) setTags(parsed)
        } catch { setTags([]) }
      } catch (e) {
        console.error('加载组件失败:', e)
        showToast('加载组件数据失败', 'error')
      } finally { setLoading(false) }
    }
    load()
  }, [componentId])

  // ---- 标签操作 ----
  const addTag = useCallback(() => {
    const t = tagInput.trim()
    if (t && !tags.includes(t)) { setTags(prev => [...prev, t]); setTagInput('') }
  }, [tagInput, tags])

  const removeTag = (tag: string) => setTags(prev => prev.filter(t => t !== tag))

  const toggleStyleTag = (tagValue: string) => {
    setTags(prev => prev.includes(tagValue) ? prev.filter(t => t !== tagValue) : [...prev, tagValue])
  }

  // ---- 保存 ----
  const handleSave = async () => {
    if (!displayLabel.trim()) { showToast('请填写展示标签', 'error'); return }
    setSaving(true)
    try {
      const payload: Record<string, unknown> = {
        library_type: libraryType,
        subject,
        grade_range: gradeRange,
        injection_mode: injectionMode,
        scope,
        display_label: displayLabel.trim(),
        design_logic: designLogic.trim(),
        example_snippet: exampleSnippet.trim(),
        full_guide: fullGuide.trim(),
        tags: JSON.stringify(tags),
        content: '{}',
      }
      if (isEdit && componentId) {
        await updateComponent(componentId, payload)
        showToast('组件已更新 ✓')
      } else {
        await createComponent(payload)
        showToast('组件已创建 ✓')
      }
      // 延迟关闭让Toast显示
      setTimeout(() => { onSaved(); onClose() }, 500)
    } catch (e: unknown) {
      console.error('保存失败:', e)
      showToast(e instanceof Error ? e.message : '保存失败', 'error')
    } finally { setSaving(false) }
  }

  // ---- 当前选中的组件类型信息 ----
  const currentType = LIBRARY_TYPES.find(t => t.key === libraryType)

  // ==================== 渲染 ====================
  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
      background: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center',
      zIndex: 10000, padding: '20px',
    }} onClick={onClose}>
      <div style={{
        background: C.card, borderRadius: '16px', width: '720px', maxHeight: '90vh',
        overflow: 'hidden', display: 'flex', flexDirection: 'column',
        boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }} onClick={e => e.stopPropagation()}>

        {/* ======== 头部 ======== */}
        <div style={{
          padding: '20px 24px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        }}>
          <div>
            <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>
              {isEdit ? '编辑组件' : '创建组件'}
            </div>
            {currentType && (
              <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>
                {currentType.icon} {currentType.name} — {currentType.desc}
              </div>
            )}
          </div>
          <button onClick={onClose} style={{
            background: 'none', border: 'none', fontSize: '20px', color: C.textMuted,
            cursor: 'pointer', padding: '4px 8px', borderRadius: '6px',
          }}>✕</button>
        </div>

        {/* ======== 内容区（可滚动）======== */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '24px' }}>
          {loading ? (
            <div style={{ textAlign: 'center', padding: '40px', color: C.textMuted }}>加载中...</div>
          ) : (
            <>
              {/* 组件库类型选择 */}
              <div style={{ marginBottom: '20px' }}>
                <label style={labelStyle}>组件库类型 <span style={{ color: C.danger }}>*</span></label>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '6px' }}>
                  {LIBRARY_TYPES.map(lt => (
                    <button key={lt.key} onClick={() => setLibraryType(lt.key)} style={{
                      padding: '8px 6px', borderRadius: '8px', fontSize: '11px', fontWeight: libraryType === lt.key ? 600 : 400,
                      border: `1px solid ${libraryType === lt.key ? C.primary : C.border}`,
                      background: libraryType === lt.key ? C.primaryLight : 'transparent',
                      color: libraryType === lt.key ? C.primary : C.textSec,
                      cursor: 'pointer', transition: 'all 150ms ease', textAlign: 'center',
                    }}>
                      <span style={{ fontSize: '16px', display: 'block', marginBottom: '2px' }}>{lt.icon}</span>
                      {lt.name.replace('库', '')}
                    </button>
                  ))}
                </div>
              </div>

              {/* 基本属性行 */}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', gap: '12px', marginBottom: '20px' }}>
                <div>
                  <label style={labelStyle}>学科</label>
                  <select value={subject} onChange={e => setSubject(e.target.value)} style={selectStyle}>
                    {SUBJECTS.map(s => <option key={s} value={s}>{s === 'general' ? '通用' : s}</option>)}
                  </select>
                </div>
                <div>
                  <label style={labelStyle}>年级</label>
                  <select value={gradeRange} onChange={e => setGradeRange(e.target.value)} style={selectStyle}>
                    {GRADES.map(g => <option key={g} value={g}>{g || '不限'}</option>)}
                  </select>
                </div>
                <div>
                  <label style={labelStyle}>注入模式</label>
                  <select value={injectionMode} onChange={e => setInjectionMode(e.target.value as InjectionMode)} style={selectStyle}>
                    {INJECTION_MODES.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
                  </select>
                </div>
                <div>
                  <label style={labelStyle}>可见范围</label>
                  <select value={scope} onChange={e => setScope(e.target.value)} style={selectStyle}>
                    {SCOPE_OPTIONS.map(s => <option key={s.value} value={s.value}>{s.label}</option>)}
                  </select>
                </div>
              </div>

              {/* 四层内容 */}
              <div style={{ marginBottom: '16px' }}>
                <label style={labelStyle}>展示标签 <span style={{ color: C.danger }}>*</span> <span style={{ fontWeight: 400, color: C.textMuted }}>（emoji+大白话，如"🎯 问题驱动式导入"）</span></label>
                <input type="text" value={displayLabel} onChange={e => setDisplayLabel(e.target.value)}
                  placeholder="例如：🎯 问题驱动式导入——用生活场景引发思考" style={inputStyle} />
              </div>

              <div style={{ marginBottom: '16px' }}>
                <label style={labelStyle}>设计逻辑 <span style={{ fontWeight: 400, color: C.textMuted }}>（为什么这样设计，100-200字）</span></label>
                <textarea value={designLogic} onChange={e => setDesignLogic(e.target.value)} rows={3}
                  placeholder="说明核心设计理念和教学意图：为什么选择这种方式？解决什么教学问题？" style={textareaStyle} />
              </div>

              <div style={{ marginBottom: '16px' }}>
                <label style={labelStyle}>参考案例 <span style={{ fontWeight: 400, color: C.textMuted }}>（具体示例片段，帮助理解）</span></label>
                <textarea value={exampleSnippet} onChange={e => setExampleSnippet(e.target.value)} rows={3}
                  placeholder="提供一个具体的使用案例或教学片段示例" style={textareaStyle} />
              </div>

              <div style={{ marginBottom: '20px' }}>
                <label style={labelStyle}>完整指引 <span style={{ fontWeight: 400, color: C.textMuted }}>（AI使用的完整说明，越详细越好）</span></label>
                <textarea value={fullGuide} onChange={e => setFullGuide(e.target.value)} rows={4}
                  placeholder="给AI的完整指引：包含实施步骤、注意事项、变式建议等" style={textareaStyle} />
              </div>

              {/* 风格标签快捷选择 */}
              <div style={{ marginBottom: '16px' }}>
                <label style={labelStyle}>风格标签 <span style={{ fontWeight: 400, color: C.textMuted }}>（用于画像感知推荐，可多选）</span></label>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                  {STYLE_TAGS.map(st => {
                    const active = tags.includes(st.value)
                    return (
                      <button key={st.value} onClick={() => toggleStyleTag(st.value)} style={{
                        padding: '4px 12px', borderRadius: '16px', fontSize: '12px',
                        border: `1px solid ${active ? C.primary : C.border}`,
                        background: active ? C.primaryLight : 'transparent',
                        color: active ? C.primary : C.textSec,
                        cursor: 'pointer', transition: 'all 150ms ease', fontWeight: active ? 600 : 400,
                      }}>{active ? '✓ ' : ''}{st.label}</button>
                    )
                  })}
                </div>
              </div>

              {/* 自由标签 */}
              <div style={{ marginBottom: '16px' }}>
                <label style={labelStyle}>自由标签 <span style={{ fontWeight: 400, color: C.textMuted }}>（自定义标签，回车添加）</span></label>
                <div style={{ display: 'flex', gap: '8px', marginBottom: '8px' }}>
                  <input type="text" value={tagInput} onChange={e => setTagInput(e.target.value)}
                    onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); addTag() } }}
                    placeholder="输入标签后按回车" style={{ ...inputStyle, flex: 1 }} />
                  <button onClick={addTag} style={{
                    padding: '8px 16px', borderRadius: '8px', border: 'none',
                    background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600,
                    cursor: 'pointer', flexShrink: 0,
                  }}>添加</button>
                </div>
                {tags.length > 0 && (
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                    {tags.map(tag => {
                      const isStyleTag = tag.includes(':')
                      return (
                        <span key={tag} style={{
                          display: 'inline-flex', alignItems: 'center', gap: '4px',
                          padding: '3px 10px', borderRadius: '12px', fontSize: '12px',
                          background: isStyleTag ? 'rgba(79,123,232,0.08)' : 'rgba(16,185,129,0.08)',
                          color: isStyleTag ? C.primary : C.success,
                        }}>
                          {STYLE_TAGS.find(s => s.value === tag)?.label || tag}
                          <button onClick={() => removeTag(tag)} style={{
                            background: 'none', border: 'none', cursor: 'pointer',
                            fontSize: '12px', color: 'inherit', padding: '0 2px',
                          }}>✕</button>
                        </span>
                      )
                    })}
                  </div>
                )}
              </div>
            </>
          )}
        </div>

        {/* ======== 底部操作栏 ======== */}
        <div style={{
          padding: '16px 24px', borderTop: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        }}>
          <div style={{ fontSize: '12px', color: C.textMuted }}>
            {tags.length > 0 && `${tags.length} 个标签`}
          </div>
          <div style={{ display: 'flex', gap: '10px' }}>
            <button onClick={onClose} style={{
              padding: '9px 20px', borderRadius: '8px', border: `1px solid ${C.border}`,
              background: '#fff', color: C.textSec, fontSize: '14px', cursor: 'pointer',
            }}>取消</button>
            <button onClick={handleSave} disabled={saving || !displayLabel.trim()} style={{
              padding: '9px 24px', borderRadius: '8px', border: 'none', fontSize: '14px', fontWeight: 600,
              cursor: saving || !displayLabel.trim() ? 'not-allowed' : 'pointer',
              background: saving || !displayLabel.trim() ? C.border : C.primary,
              color: saving || !displayLabel.trim() ? C.textMuted : '#fff',
            }}>{saving ? '保存中...' : isEdit ? '更新组件' : '创建组件'}</button>
          </div>
        </div>

        {/* Toast */}
        {toast && (
          <div style={{
            position: 'absolute', bottom: '80px', left: '50%', transform: 'translateX(-50%)',
            padding: '10px 20px', borderRadius: '8px',
            background: toast.type === 'error' ? '#FEF2F2' : '#1F2937',
            color: toast.type === 'error' ? C.danger : '#fff',
            fontSize: '13px', fontWeight: 500, boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
            border: toast.type === 'error' ? '1px solid #FECACA' : 'none',
            zIndex: 10001, whiteSpace: 'nowrap',
          }}>
            {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
          </div>
        )}
      </div>
    </div>
  )
}
