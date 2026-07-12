/**
 * WorkshopPanels.tsx — 备课工坊各面板子组件
 *
 * 组件列表：
 *   StartForm         — 首屏备课表单
 *   AIBubble          — AI消息气泡（支持Markdown+组件选择）
 *   UserBubble        — 用户消息气泡
 *   ThinkingIndicator — AI思考中动画
 *   ReviewPanel       — AI评审结果面板
 *
 * v203变更（专家模式首屏简化）：
 *   - StartForm 从双栏布局（左基本信息+右320px黄色配方面板）改为单栏布局
 *   - 配方选择从整版面板改为与对话模式一致的下拉选择器
 *   - 复用 getAvailableRecipes API（按用户可见性+学科过滤，学校>教研组>个人排序）
 *   - 去掉"当前学科/全部配方"切换器、去掉"新建配方"入口（需要的去配方管理页）
 *   - 保留课本图片区域不变
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import type { ConversationMessage, AIReviewResult, ConvComponent } from '@/api/lesson-plans'
import { getAvailableRecipes, type RecipeListItem } from '@/api/recipes'
import { getTextbooks, triggerTextbookOCR, type TextbookListItem } from '@/api/textbooks'
import { getAvailablePublishers, publisherLabel } from '@/api/course-outlines'
import { C, SUBJECTS, GRADES, renderMarkdown } from './workshopConstants'

// ==================== 配方 scope 徽标配置（与对话模式共用同一套配色） ====================

const RECIPE_SCOPE_BADGE: Record<string, { label: string; color: string; bg: string }> = {
  school:   { label: '学校', color: '#166534', bg: '#DCFCE7' },
  group:    { label: '教研组', color: '#1E40AF', bg: '#DBEAFE' },
  personal: { label: '个人', color: '#6B7280', bg: '#F3F4F6' },
}

// ==================== 首屏备课表单 ====================

interface StartFormProps {
  onStart: (subject: string, grade: string, topic: string, duration: number, recipeId?: string, textbookPageIds?: string[], coursePublisher?: string | null) => void
  loading: boolean
}

export function StartForm({ onStart, loading }: StartFormProps) {
  const [subject, setSubject]   = useState('AI')
  const [grade, setGrade]       = useState('一年级')
  const [topic, setTopic]       = useState('')
  const [duration, setDuration] = useState(45)
  // 教材版本（专家模式起步选）：三态 null=不关联/''=通用版/具名=该版本
  const [coursePublisher, setCoursePublisher] = useState<string | null>(null)
  const [coursePublishers, setCoursePublishers] = useState<string[]>([])
  const [coursePubLoading, setCoursePubLoading] = useState(false)
  const navigate = useNavigate()

  // v203 简化：配方用 getAvailableRecipes（按用户可见性+学科，学校>教研组>个人排序）
  const [recipes, setRecipes]             = useState<RecipeListItem[]>([])
  const [recipesLoading, setRecipesLoad]  = useState(false)
  const [selectedRecipeId, setSelectedId] = useState<string | null>(null)

  const [textbooks, setTextbooks]             = useState<TextbookListItem[]>([])
  const [textbooksLoading, setTextbooksLoad]  = useState(false)
  const [textbooksLoaded, setTextbooksLoaded] = useState(false)
  const [selectedTextbookIds, setSelectedTBIds] = useState<Set<string>>(new Set())
  const [ocrInProgress, setOcrInProgress] = useState<Set<string>>(new Set())
  const [ocrFailed, setOcrFailed]         = useState<Set<string>>(new Set())

  // v203 简化：按学科拉可见配方（复用对话模式同一 API），学科切换时重拉并清空已选
  useEffect(() => {
    let cancelled = false
    setRecipesLoad(true)
    getAvailableRecipes(subject)
      .then(resp => {
        if (cancelled) return
        const list = resp.recipes || []
        setRecipes(list)
        // 学科切换时：已选配方若不在新列表里则清空
        setSelectedId(prev => prev && list.some(r => r.id === prev) ? prev : null)
      })
      .catch(() => { if (!cancelled) setRecipes([]) })
      .finally(() => { if (!cancelled) setRecipesLoad(false) })
    return () => { cancelled = true }
  }, [subject])

  useEffect(() => {
    const loadTextbooks = async () => {
      setTextbooksLoad(true); setTextbooksLoaded(false)
      try {
        const resp = await getTextbooks({ subject, grade_range: grade, limit: 50 })
        setTextbooks(resp.pages || [])
        setSelectedTBIds(new Set())
      } catch { setTextbooks([]) }
      finally { setTextbooksLoad(false); setTextbooksLoaded(true) }
    }
    loadTextbooks()
  }, [subject, grade])

  // 教材版本下拉：按学科+年级拉该学科年级真实存在大纲的可选版本（与对话模式同款）。
  // 有大纲→默认选中第一个版本（可改可清）；无大纲→不显示、coursePublisher 置 null（不关联）。
  useEffect(() => {
    let cancelled = false
    if (!subject || !grade) { setCoursePublishers([]); setCoursePublisher(null); return }
    setCoursePubLoading(true)
    getAvailablePublishers(subject, grade)
      .then(list => {
        if (cancelled) return
        const pubs = list || []
        setCoursePublishers(pubs)
        if (pubs.length === 0) {
          setCoursePublisher(null)
        } else {
          setCoursePublisher(prev => (prev !== null && pubs.includes(prev)) ? prev : pubs[0])
        }
      })
      .catch(() => { if (!cancelled) { setCoursePublishers([]); setCoursePublisher(null) } })
      .finally(() => { if (!cancelled) setCoursePubLoading(false) })
    return () => { cancelled = true }
  }, [subject, grade])

  const toggleTextbook = (id: string) => {
    const willSelect = !selectedTextbookIds.has(id)
    setSelectedTBIds(prev => {
      const n = new Set(prev)
      if (n.has(id)) { n.delete(id) } else { n.add(id) }
      return n
    })
    if (willSelect) {
      const tb = textbooks.find(t => t.id === id)
      if (tb && !tb.has_ocr && !ocrInProgress.has(id)) {
        maybeTriggerOCR(id)
      }
    }
  }

  const maybeTriggerOCR = async (id: string) => {
    setOcrInProgress(prev => { const n = new Set(prev); n.add(id); return n })
    setOcrFailed(prev => { const n = new Set(prev); n.delete(id); return n })
    try {
      await triggerTextbookOCR(id)
      setTextbooks(prev => prev.map(t => t.id === id ? { ...t, has_ocr: true } : t))
    } catch {
      setOcrFailed(prev => { const n = new Set(prev); n.add(id); return n })
    } finally {
      setOcrInProgress(prev => { const n = new Set(prev); n.delete(id); return n })
    }
  }

  const handleSubmit = () => {
    if (!topic.trim()) return
    if (ocrInProgress.size > 0) return
    onStart(subject, grade, topic.trim(), duration, selectedRecipeId || undefined, selectedTextbookIds.size > 0 ? Array.from(selectedTextbookIds) : undefined, coursePublisher)
  }

  const selectedRecipe = recipes.find(r => r.id === selectedRecipeId)
  const selBtn = (active: boolean): React.CSSProperties => ({
    padding: '6px 14px', borderRadius: '20px',
    border: `1px solid ${active ? C.primary : C.border}`,
    background: active ? C.primaryLight : 'transparent',
    color: active ? C.primary : C.textSec,
    fontSize: '13px', fontWeight: active ? 600 : 400,
    cursor: 'pointer', transition: 'all 150ms ease',
  })

  return (
    <div style={{ maxWidth: '680px', margin: '0 auto', padding: '36px 0' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '24px' }}>
        <span style={{ fontSize: '28px', lineHeight: 1 }}>✨</span>
        <div>
          <h1 style={{ fontSize: '20px', fontWeight: 700, color: C.text, margin: '0 0 2px' }}>开始今天的备课</h1>
          <p style={{ fontSize: '13px', color: C.textSec, margin: 0 }}>告诉AI你要上什么课，选择配方让AI从第一句话就带着教研共识工作</p>
        </div>
      </div>

      {/* 单栏布局（v203 简化：去掉右侧 320px 配方面板） */}
      <div style={{ background: C.card, borderRadius: '16px', padding: '28px', boxShadow: '0 4px 24px rgba(0,0,0,0.06)', border: `1px solid ${C.border}` }}>
        <div style={{ marginBottom: '18px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>学科</label>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
            {SUBJECTS.map(s => <button key={s} onClick={() => setSubject(s)} style={selBtn(subject === s)}>{s}</button>)}
          </div>
        </div>
        <div style={{ marginBottom: '18px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>年级</label>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
            {GRADES.map(g => <button key={g} onClick={() => setGrade(g)} style={selBtn(grade === g)}>{g}</button>)}
          </div>
        </div>
        <div style={{ marginBottom: '18px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
            课题 <span style={{ color: C.danger }}>*</span>
          </label>
          <input type="text" value={topic} onChange={e => setTopic(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleSubmit()}
            placeholder="例如：认识人工智能、图像识别应用..."
            style={{ width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '15px', color: C.text, outline: 'none', boxSizing: 'border-box', transition: 'border-color 150ms ease' }}
            onFocus={e => { e.target.style.borderColor = C.primary }}
            onBlur={e  => { e.target.style.borderColor = C.border }} />
        </div>
        <div style={{ marginBottom: '18px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>课时时长</label>
          <div style={{ display: 'flex', gap: '8px' }}>
            {[40, 45, 50, 60].map(d => <button key={d} onClick={() => setDuration(d)} style={selBtn(duration === d)}>{d}分钟</button>)}
          </div>
        </div>

        {/* v203 简化：配方下拉（与对话模式一致的交互，复用 getAvailableRecipes） */}
        {recipes.length > 0 && (
          <div style={{ marginBottom: '18px' }}>
            <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>📦 备课配方（选填）</label>
            <select value={selectedRecipeId || ''} onChange={e => setSelectedId(e.target.value || null)} disabled={recipesLoading}
              style={{ width: '100%', padding: '11px 14px', borderRadius: '10px', border: `1.5px solid ${selectedRecipeId ? '#F59E0B' : C.border}`, fontSize: '14px', color: selectedRecipeId ? C.text : C.textMuted, background: C.card, cursor: 'pointer', outline: 'none', boxSizing: 'border-box', transition: 'border-color 150ms ease' }}>
              <option value="">不使用配方（AI用系统预置骨架）</option>
              {recipes.map(r => {
                const badge = RECIPE_SCOPE_BADGE[r.scope] || RECIPE_SCOPE_BADGE.personal
                return (
                  <option key={r.id} value={r.id}>
                    [{badge.label}] {r.name} · {r.component_count}组件 · 用{r.use_count}次
                  </option>
                )
              })}
            </select>
            {selectedRecipe && (
              <div style={{ marginTop: '8px', padding: '8px 12px', background: '#FFFBEB', borderRadius: '8px', border: '1px solid #FDE68A', fontSize: '12px', color: '#92400E', lineHeight: 1.6 }}>
                ✅ 已选「{selectedRecipe.name}」— {selectedRecipe.description || '教案结构+流程+学情由此配方定义'}
              </div>
            )}
            {!selectedRecipeId && (
              <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px', lineHeight: 1.5 }}>
                💡 不选配方时，AI助手的指引（含结构）将作为唯一个性化来源
              </div>
            )}
          </div>
        )}
        {recipes.length === 0 && !recipesLoading && (
          <div style={{ marginBottom: '18px', padding: '12px 14px', background: 'rgba(79,123,232,0.04)', borderRadius: '10px', border: '1px dashed rgba(79,123,232,0.25)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div style={{ fontSize: '13px', color: C.textSec, lineHeight: 1.6 }}>
              📦 暂无「{subject}」学科的可用配方 — AI将使用系统预置骨架
            </div>
            <button onClick={() => navigate('/lesson-plans/recipes')} style={{ padding: '6px 12px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '12px', fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap', flexShrink: 0 }}>
              去配方管理
            </button>
          </div>
        )}

        {/* 教材版本（选填）——仅当本学科本年级真实有大纲时才显示 */}
        {coursePublishers.length > 0 && (
          <div style={{ marginBottom: '18px' }}>
            <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>📚 教材版本</label>
            <select
              value={coursePublisher === null ? '__none__' : coursePublisher}
              onChange={e => setCoursePublisher(e.target.value === '__none__' ? null : e.target.value)}
              disabled={coursePubLoading}
              style={{ width: '100%', padding: '11px 14px', borderRadius: '10px', border: `1.5px solid ${coursePublisher !== null ? '#8B5CF6' : C.border}`, fontSize: '14px', color: coursePublisher !== null ? C.text : C.textMuted, background: C.card, cursor: 'pointer', outline: 'none', boxSizing: 'border-box', transition: 'border-color 150ms ease' }}>
              {coursePublishers.map(p => (
                <option key={p || '__generic__'} value={p}>{publisherLabel(p)}</option>
              ))}
              <option value="__none__">不关联大纲（本节课不注入大纲）</option>
            </select>
            {coursePublisher !== null ? (
              <div style={{ fontSize: '11px', color: '#7C3AED', marginTop: '6px', lineHeight: 1.5 }}>
                ✓ 备课时将注入「{publisherLabel(coursePublisher)}」这版的课程大纲
              </div>
            ) : (
              <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px', lineHeight: 1.5 }}>
                💡 当前不关联大纲；如需对齐教材，请在上方选择你所用的版本
              </div>
            )}
          </div>
        )}

        <button onClick={handleSubmit} disabled={!topic.trim() || loading || ocrInProgress.size > 0}
          style={{ width: '100%', padding: '14px', borderRadius: '10px', border: 'none', background: (!topic.trim() || loading || ocrInProgress.size > 0) ? '#E5E7EB' : C.primary, color: (!topic.trim() || loading || ocrInProgress.size > 0) ? C.textMuted : '#fff', fontSize: '16px', fontWeight: 600, cursor: (!topic.trim() || loading || ocrInProgress.size > 0) ? 'not-allowed' : 'pointer', transition: 'all 200ms ease' }}>
          {ocrInProgress.size > 0 ? `课本识别中（${ocrInProgress.size}）请稍候...` : loading ? '正在准备备课环境...' : selectedRecipeId ? '📦 带配方开始备课 →' : '开始备课 →'}
        </button>
        {selectedTextbookIds.size > 0 && (
          <div style={{ marginTop: '8px', padding: '10px 12px', background: 'rgba(16,185,129,0.06)', borderRadius: '8px', fontSize: '12px', color: '#166534', lineHeight: 1.6 }}>
            📷 已关联 {selectedTextbookIds.size} 张课本图片，AI会参考课本原文
          </div>
        )}
      </div>

      {/* 课本图片区域（完整保留，逻辑不动） */}
      {textbooksLoaded && textbooks.length > 0 && (
        <div style={{ maxWidth: '680px', margin: '16px auto 0', background: '#F0FDF4', borderRadius: '12px', padding: '16px 20px', border: '1px solid #BBF7D0' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '10px' }}>
            <div style={{ fontSize: '14px', fontWeight: 600, color: '#166534' }}>
              📷 关联课本图片 <span style={{ fontSize: '12px', fontWeight: 400, color: '#6B7280' }}>（已选 {selectedTextbookIds.size} 张）</span>
            </div>
            <button onClick={() => navigate('/lesson-plans/textbooks')} style={{ fontSize: '12px', color: C.primary, background: 'none', border: 'none', cursor: 'pointer', textDecoration: 'underline' }}>管理课本</button>
          </div>
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', maxHeight: '120px', overflowY: 'auto' }}>
            {textbooks.map(tb => {
              const checked = selectedTextbookIds.has(tb.id)
              return (
                <div key={tb.id} onClick={() => toggleTextbook(tb.id)} style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '6px 10px', borderRadius: '8px', cursor: 'pointer', background: checked ? 'rgba(79,123,232,0.08)' : '#fff', border: checked ? '1px solid #4F7BE8' : '1px solid #E5E7EB', fontSize: '12px', color: '#1F2937', transition: 'all 150ms ease', userSelect: 'none' }}>
                  <input type="checkbox" checked={checked} readOnly style={{ accentColor: '#4F7BE8', pointerEvents: 'none' }} />
                  <img src={tb.image_url} alt="" style={{ width: '28px', height: '28px', objectFit: 'cover', borderRadius: '4px' }} onError={e => { (e.target as HTMLImageElement).style.display = 'none' }} />
                  <div style={{ maxWidth: '120px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {tb.chapter || tb.textbook_name}
                    {ocrInProgress.has(tb.id)
                      ? <span style={{ marginLeft: '4px', color: '#F59E0B', fontSize: '10px' }}>⏳识别中…</span>
                      : ocrFailed.has(tb.id)
                        ? <span style={{ marginLeft: '4px', color: '#EF4444', fontSize: '10px' }}>⚠识别失败</span>
                        : tb.has_ocr
                          ? <span style={{ marginLeft: '4px', color: '#10B981', fontSize: '10px' }}>✓已识别</span>
                          : <span style={{ marginLeft: '4px', color: '#9CA3AF', fontSize: '10px' }}>未识别</span>}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}
      {textbooksLoaded && textbooks.length === 0 && !textbooksLoading && (
        <div style={{ maxWidth: '680px', margin: '16px auto 0', borderRadius: '12px', padding: '16px 20px', background: 'rgba(79,123,232,0.04)', border: '1px dashed rgba(79,123,232,0.25)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ fontSize: '24px' }}>📷</span>
            <div>
              <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>上传课本图片，让AI精准参考课本原文</div>
              <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px', lineHeight: 1.5 }}>拍照或扫描课本相关页面，AI识别文字后备课更贴合教材内容</div>
            </div>
          </div>
          <button onClick={() => navigate('/lesson-plans/textbooks')} style={{ padding: '8px 16px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap', flexShrink: 0 }}>
            去上传课本 →
          </button>
        </div>
      )}
      {textbooksLoading && (
        <div style={{ maxWidth: '680px', margin: '16px auto 0', textAlign: 'center', fontSize: '12px', color: C.textMuted }}>加载课本图片...</div>
      )}

      {/* 快捷入口 */}
      <div style={{ display: 'flex', justifyContent: 'center', gap: '8px', marginTop: '20px' }}>
        {[
          { icon: '📋', text: '我的教案', path: '/lesson-plans/my-plans' },
          { icon: '📦', text: '配方管理', path: '/lesson-plans/recipes' },
          { icon: '📚', text: '教案库',   path: '/lesson-plans/library' },
          { icon: '📷', text: '课本管理', path: '/lesson-plans/textbooks' },
        ].map(item => (
          <button key={item.path} onClick={() => navigate(item.path)}
            style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', color: C.textSec, background: 'transparent', border: 'none', padding: '6px 12px', borderRadius: '8px', cursor: 'pointer', transition: 'all 150ms ease' }}
            onMouseEnter={e => { const el = e.currentTarget as HTMLButtonElement; el.style.background = C.primaryLight; el.style.color = C.primary }}
            onMouseLeave={e => { const el = e.currentTarget as HTMLButtonElement; el.style.background = 'transparent'; el.style.color = C.textSec }}>
            <span>{item.icon}</span><span>{item.text}</span>
          </button>
        ))}
      </div>
    </div>
  )
}

// ==================== AI消息气泡 ====================

interface AIBubbleProps {
  msg: ConversationMessage
  streaming?: boolean
  onSelectComponent: (comp: ConvComponent) => void
  selectedComponentIds: Set<string>
}

export function AIBubble({ msg, streaming = false, onSelectComponent, selectedComponentIds }: AIBubbleProps) {
  const [expandedComponent, setExpandedComponent] = useState<string | null>(null)

  return (
    <div style={{ display: 'flex', gap: '10px', marginBottom: '16px', alignItems: 'flex-start' }}>
      <div style={{ width: '32px', height: '32px', flexShrink: 0, background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '14px' }}>✨</div>
      <div style={{ flex: 1, maxWidth: 'calc(100% - 42px)' }}>
        {msg.content && (
          <div style={{ background: C.aiBubble, borderRadius: '0 12px 12px 12px', padding: '12px 16px', wordBreak: 'break-word' }}>
            {renderMarkdown(msg.content)}
            {streaming && (
              <span style={{ display: 'inline-block', width: '2px', height: '1em', background: C.primary, marginLeft: '2px', verticalAlign: 'text-bottom', animation: 'cursor-blink 0.8s step-end infinite' }} />
            )}
            <style>{`@keyframes cursor-blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }`}</style>
          </div>
        )}
        {msg.type === 'components' && msg.components && msg.components.length > 0 && (
          <div style={{ marginTop: '10px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {msg.components.map(comp => {
              const isSelected = selectedComponentIds.has(comp.id)
              const isExpanded = expandedComponent === comp.id
              return (
                <div key={comp.id} style={{ background: C.card, borderRadius: '10px', border: `1px solid ${isSelected ? C.primary : C.border}`, borderLeft: `3px solid ${C.accent}`, padding: '12px 14px', transition: 'all 200ms ease' }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{comp.display_label}</div>
                      {comp.usage_count > 0 && <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>{comp.usage_count}位老师用过 · 质量分{comp.quality_score.toFixed(1)}</div>}
                    </div>
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexShrink: 0, marginLeft: '12px' }}>
                      {comp.design_logic && (
                        <button onClick={() => setExpandedComponent(isExpanded ? null : comp.id)} style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '12px', color: C.textSec, cursor: 'pointer' }}>
                          {isExpanded ? '收起' : '看逻辑'}
                        </button>
                      )}
                      <button onClick={() => onSelectComponent(comp)} style={{ padding: '4px 12px', borderRadius: '6px', border: `1px solid ${isSelected ? C.primary : C.border}`, background: isSelected ? C.primaryLight : 'transparent', fontSize: '13px', color: isSelected ? C.primary : C.textSec, fontWeight: isSelected ? 600 : 400, cursor: 'pointer', transition: 'all 150ms ease' }}>
                        {isSelected ? '✓ 已选' : '选择✓'}
                      </button>
                    </div>
                  </div>
                  {isExpanded && comp.design_logic && (
                    <div style={{ marginTop: '10px', padding: '10px 12px', background: '#F9FAFB', borderRadius: '8px', fontSize: '13px', color: C.textSec, lineHeight: 1.7 }}>
                      {comp.design_logic}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

// ==================== 用户消息气泡 ====================

export function UserBubble({ msg }: { msg: ConversationMessage }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: '16px' }}>
      <div style={{ maxWidth: '75%', background: C.userBubble, border: `1px solid ${C.border}`, borderRadius: '12px 0 12px 12px', padding: '10px 14px', fontSize: '15px', color: C.text, lineHeight: 1.7, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
        {msg.content}
      </div>
    </div>
  )
}

// ==================== 思考中动画 ====================

export function ThinkingIndicator() {
  return (
    <div style={{ display: 'flex', gap: '10px', marginBottom: '16px', alignItems: 'flex-start' }}>
      <div style={{ width: '32px', height: '32px', flexShrink: 0, background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '14px' }}>✨</div>
      <div style={{ background: C.aiBubble, borderRadius: '0 12px 12px 12px', padding: '14px 18px', display: 'flex', alignItems: 'center', gap: '6px' }}>
        {[0,1,2].map(i => (
          <div key={i} style={{ width: '6px', height: '6px', borderRadius: '50%', background: C.primary, animation: `lp-pulse 1.2s ease-in-out ${i * 0.2}s infinite` }} />
        ))}
        <style>{`@keyframes lp-pulse { 0%, 80%, 100% { opacity: 0.3; transform: scale(0.8); } 40% { opacity: 1; transform: scale(1.2); } }`}</style>
      </div>
    </div>
  )
}

// ==================== AI评审面板 ====================

function stripBoldFE(s: unknown): string {
  return String(s ?? '').replace(/\*/g, '').trim()
}
function isHeaderDimFE(name: string): boolean {
  if (!name) return true
  const headers = ['评审维度', '维度', '评分', '简短评语', '评语', '得分', '分数']
  return headers.includes(name)
}

interface ReviewDimension {
  code?: string
  name?: string
  score?: number
  comment?: string
}

interface ReviewPanelProps {
  review: AIReviewResult
  onApply: (ids?: string[]) => void
  applying: boolean
  isStageMode?: boolean
}

export function ReviewPanel({ review, onApply, applying, isStageMode = false }: ReviewPanelProps) {
  const isGood = review.total_score >= 8.5

  const rawDims = (review as unknown as { dimensions?: ReviewDimension[] }).dimensions
  const cleanDims: ReviewDimension[] = Array.isArray(rawDims)
    ? rawDims
        .map(d => ({
          code: stripBoldFE(d.code),
          name: stripBoldFE(d.name),
          score: typeof d.score === 'number' ? d.score : undefined,
          comment: typeof d.comment === 'string' ? d.comment : '',
        }))
        .filter(d => !isHeaderDimFE(d.name || '') && typeof d.score === 'number')
    : []

  return (
    <div style={{ padding: '16px', height: '100%', overflowY: 'auto', boxSizing: 'border-box' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '16px', padding: '14px 16px', background: isGood ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)', borderRadius: '10px', border: `1px solid ${isGood ? '#10B98130' : '#F59E0B30'}` }}>
        <div style={{ fontSize: '28px', fontWeight: 700, flexShrink: 0, color: isGood ? C.success : C.accent }}>
          {review.total_score.toFixed(1)}
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>AI综合评分</div>
          <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px', lineHeight: 1.5 }}>{review.summary}</div>
        </div>
      </div>

      {cleanDims.length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>📊 各维度评分</div>
          {cleanDims.map((d, i) => {
            const sc = d.score as number
            const barGood = sc >= 8.5
            return (
              <div key={i} style={{ marginBottom: '8px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '3px' }}>
                  <span style={{ fontSize: '12px', color: C.textSec, flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {d.code ? `${d.code} ` : ''}{d.name || `维度${i + 1}`}
                  </span>
                  <span style={{ fontSize: '12px', fontWeight: 600, color: barGood ? C.success : C.accent, width: '32px', textAlign: 'right', flexShrink: 0 }}>
                    {sc.toFixed(1)}
                  </span>
                </div>
                <div style={{ height: '6px', background: '#F3F4F6', borderRadius: '3px', overflow: 'hidden' }}>
                  <div style={{ height: '100%', borderRadius: '3px', width: `${Math.min(100, sc * 10)}%`, background: barGood ? C.success : C.accent, transition: 'width 600ms ease' }} />
                </div>
                {d.comment && (
                  <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '3px', lineHeight: 1.5 }}>{d.comment}</div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {(review.good_points || []).length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.success, marginBottom: '8px' }}>✅ 做得好的</div>
          {(review.good_points || []).map((point, i) => (
            <div key={i} style={{ fontSize: '13px', color: C.text, lineHeight: 1.6, padding: '6px 10px', marginBottom: '4px', background: 'rgba(16,185,129,0.06)', borderRadius: '6px' }}>
              {point}
            </div>
          ))}
        </div>
      )}
      {(review.improvements || []).length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.accent, marginBottom: '8px' }}>💡 可以更好</div>
          {(review.improvements || []).map(imp => (
            <div key={imp.id} style={{ marginBottom: '8px', padding: '10px 12px', background: 'rgba(245,158,11,0.06)', borderRadius: '8px', border: '1px solid rgba(245,158,11,0.15)' }}>
              <div style={{ fontSize: '13px', fontWeight: 500, color: C.text, marginBottom: '4px' }}>{imp.issue}</div>
              <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.6 }}>{imp.suggestion}</div>
            </div>
          ))}
        </div>
      )}
      {!isStageMode && (
        <button onClick={() => onApply()} disabled={applying} style={{ width: '100%', padding: '10px', borderRadius: '8px', border: 'none', background: applying ? '#E5E7EB' : C.primary, color: applying ? C.textMuted : '#fff', fontSize: '13px', fontWeight: 600, cursor: applying ? 'not-allowed' : 'pointer' }}>
          {applying ? '应用中...' : '✨ 一键应用全部建议'}
        </button>
      )}
      {isStageMode && (
        <div style={{ padding: '10px 12px', borderRadius: '8px', background: 'rgba(79,123,232,0.06)', fontSize: '12px', color: '#4F7BE8', textAlign: 'center', lineHeight: 1.6 }}>
          💡 进入"修订定稿"阶段与AI讨论如何修改
        </div>
      )}
    </div>
  )
}
