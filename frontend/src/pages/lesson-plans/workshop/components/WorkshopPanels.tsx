/**
 * WorkshopPanels.tsx — 备课工坊各面板子组件
 *
 * 组件列表：
 *   StartForm         — 首屏备课表单（v4：课本图片选择+上传引导）
 *   AIBubble          — AI消息气泡（支持Markdown+组件选择）
 *   UserBubble        — 用户消息气泡
 *   ThinkingIndicator — AI思考中动画
 *   ReviewPanel       — AI评审结果面板
 *
 * 迭代8变更：
 *   - 移除 StageGateModal（阶段切换弹窗已迁移到 WorkshopTransitionComponents.tsx）
 *   - 主要组件逻辑保持不变
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import type { ConversationMessage, AIReviewResult, ConvComponent } from '@/api/lesson-plans'
import { getRecipes, type RecipeListItem } from '@/api/recipes'
import { getTextbooks, type TextbookListItem } from '@/api/textbooks'
import { C, SUBJECTS, GRADES, renderMarkdown } from './workshopConstants'

// ==================== 首屏备课表单 ====================

interface StartFormProps {
  onStart: (subject: string, grade: string, topic: string, duration: number, recipeId?: string, textbookPageIds?: string[]) => void
  loading: boolean
}

export function StartForm({ onStart, loading }: StartFormProps) {
  const [subject, setSubject]   = useState('AI')
  const [grade, setGrade]       = useState('七年级')
  const [topic, setTopic]       = useState('')
  const [duration, setDuration] = useState(45)
  const navigate = useNavigate()

  const [recipes, setRecipes]             = useState<RecipeListItem[]>([])
  const [recipesLoading, setRecipesLoad]  = useState(false)
  const [selectedRecipeId, setSelectedId] = useState<string | null>(null)

  const [textbooks, setTextbooks]             = useState<TextbookListItem[]>([])
  const [textbooksLoading, setTextbooksLoad]  = useState(false)
  const [textbooksLoaded, setTextbooksLoaded] = useState(false)
  const [selectedTextbookIds, setSelectedTBIds] = useState<Set<string>>(new Set())

  useEffect(() => {
    const loadRecipes = async () => {
      setRecipesLoad(true)
      try {
        const resp = await getRecipes({ subject, grade_range: grade, limit: 20 })
        setRecipes(resp.recipes || [])
        setSelectedId(null)
      } catch { setRecipes([]) }
      finally { setRecipesLoad(false) }
    }
    loadRecipes()
  }, [subject, grade])

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

  const toggleTextbook = (id: string) => {
    setSelectedTBIds(prev => { const n = new Set(prev); n.has(id) ? n.delete(id) : n.add(id); return n })
  }

  const handleSubmit = () => {
    if (!topic.trim()) return
    onStart(subject, grade, topic.trim(), duration, selectedRecipeId || undefined, selectedTextbookIds.size > 0 ? Array.from(selectedTextbookIds) : undefined)
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
    <div style={{ maxWidth: '960px', margin: '0 auto', padding: '36px 0' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '24px' }}>
        <span style={{ fontSize: '28px', lineHeight: 1 }}>✨</span>
        <div>
          <h1 style={{ fontSize: '20px', fontWeight: 700, color: C.text, margin: '0 0 2px' }}>开始今天的备课</h1>
          <p style={{ fontSize: '13px', color: C.textSec, margin: 0 }}>告诉AI你要上什么课，选择配方让AI从第一句话就带着全局知识工作</p>
        </div>
      </div>

      <div style={{ display: 'flex', gap: '20px', alignItems: 'flex-start' }}>
        {/* 左栏：基本信息 */}
        <div style={{ flex: 1, minWidth: 0, background: C.card, borderRadius: '16px', padding: '28px', boxShadow: '0 4px 24px rgba(0,0,0,0.06)', border: `1px solid ${C.border}` }}>
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
          <div style={{ marginBottom: '24px' }}>
            <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>课时时长</label>
            <div style={{ display: 'flex', gap: '8px' }}>
              {[40, 45, 50, 60].map(d => <button key={d} onClick={() => setDuration(d)} style={selBtn(duration === d)}>{d}分钟</button>)}
            </div>
          </div>
          <button onClick={handleSubmit} disabled={!topic.trim() || loading}
            style={{ width: '100%', padding: '14px', borderRadius: '10px', border: 'none', background: !topic.trim() || loading ? '#E5E7EB' : C.primary, color: !topic.trim() || loading ? C.textMuted : '#fff', fontSize: '16px', fontWeight: 600, cursor: !topic.trim() || loading ? 'not-allowed' : 'pointer', transition: 'all 200ms ease' }}>
            {loading ? '正在准备备课环境...' : selectedRecipeId ? '📦 带配方开始备课 →' : '开始备课 →'}
          </button>
          {selectedRecipe && (
            <div style={{ marginTop: '12px', padding: '10px 12px', background: 'rgba(79,123,232,0.06)', borderRadius: '8px', fontSize: '12px', color: C.primary, lineHeight: 1.6 }}>
              ✅ 已选「{selectedRecipe.name}」— {selectedRecipe.component_count}个组件 · v{selectedRecipe.version}
            </div>
          )}
          {selectedTextbookIds.size > 0 && (
            <div style={{ marginTop: '8px', padding: '10px 12px', background: 'rgba(16,185,129,0.06)', borderRadius: '8px', fontSize: '12px', color: '#166534', lineHeight: 1.6 }}>
              📷 已关联 {selectedTextbookIds.size} 张课本图片，AI会参考课本原文
            </div>
          )}
        </div>

        {/* 右栏：配方选择 */}
        <div style={{ width: '320px', flexShrink: 0, background: '#FFFBEB', borderRadius: '16px', padding: '24px', border: '1px solid #FDE68A', boxShadow: '0 4px 24px rgba(0,0,0,0.04)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
            <label style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>📦 备课配方</label>
            <button onClick={() => navigate('/lesson-plans/recipes/new', { state: { from: '/lesson-plans' } })} style={{ fontSize: '12px', color: C.primary, background: 'none', border: 'none', cursor: 'pointer', textDecoration: 'underline' }}>
              + 新建
            </button>
          </div>
          <p style={{ fontSize: '12px', color: '#92400E', margin: '0 0 12px', lineHeight: 1.5 }}>
            选配方后AI自动带入学情、风格等知识
          </p>
          {recipesLoading ? (
            <div style={{ fontSize: '13px', color: C.textMuted, padding: '20px 0', textAlign: 'center' }}>加载中...</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', maxHeight: '380px', overflowY: 'auto' }}>
              <button onClick={() => setSelectedId(null)} style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '10px 12px', borderRadius: '10px', textAlign: 'left', width: '100%', border: `1px solid ${selectedRecipeId === null ? C.primary : C.border}`, background: selectedRecipeId === null ? C.primaryLight : '#fff', cursor: 'pointer', transition: 'all 150ms ease' }}>
                <div style={{ width: '16px', height: '16px', borderRadius: '50%', border: `2px solid ${selectedRecipeId === null ? C.primary : C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                  {selectedRecipeId === null && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: C.primary }} />}
                </div>
                <span style={{ fontSize: '13px', color: selectedRecipeId === null ? C.primary : C.textSec, fontWeight: selectedRecipeId === null ? 600 : 400 }}>不使用配方</span>
              </button>
              {recipes.map(r => (
                <button key={r.id} onClick={() => setSelectedId(r.id)} style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '10px 12px', borderRadius: '10px', textAlign: 'left', width: '100%', border: `1px solid ${selectedRecipeId === r.id ? C.primary : C.border}`, background: selectedRecipeId === r.id ? C.primaryLight : '#fff', cursor: 'pointer', transition: 'all 150ms ease' }}>
                  <div style={{ width: '16px', height: '16px', borderRadius: '50%', border: `2px solid ${selectedRecipeId === r.id ? C.primary : C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                    {selectedRecipeId === r.id && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: C.primary }} />}
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{r.name}</div>
                    <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '2px' }}>{r.component_count}个组件 · 用{r.use_count}次</div>
                  </div>
                </button>
              ))}
              {recipes.length === 0 && !recipesLoading && (
                <div style={{ fontSize: '13px', color: C.textMuted, padding: '16px 0', textAlign: 'center', lineHeight: 1.6 }}>
                  暂无匹配配方<br />
                  <button onClick={() => navigate('/lesson-plans/recipes/new', { state: { from: '/lesson-plans' } })} style={{ color: C.primary, background: 'none', border: 'none', cursor: 'pointer', fontSize: '13px', textDecoration: 'underline', padding: 0, marginTop: '4px' }}>
                    去创建一个
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* 课本图片区域 */}
      {textbooksLoaded && textbooks.length > 0 && (
        <div style={{ maxWidth: '960px', margin: '16px auto 0', background: '#F0FDF4', borderRadius: '12px', padding: '16px 20px', border: '1px solid #BBF7D0' }}>
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
                <label key={tb.id} onClick={() => toggleTextbook(tb.id)} style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '6px 10px', borderRadius: '8px', cursor: 'pointer', background: checked ? 'rgba(79,123,232,0.08)' : '#fff', border: checked ? '1px solid #4F7BE8' : '1px solid #E5E7EB', fontSize: '12px', color: '#1F2937', transition: 'all 150ms ease' }}>
                  <input type="checkbox" checked={checked} readOnly style={{ accentColor: '#4F7BE8', pointerEvents: 'none' }} />
                  <img src={tb.image_url} alt="" style={{ width: '28px', height: '28px', objectFit: 'cover', borderRadius: '4px' }} onError={e => { (e.target as HTMLImageElement).style.display = 'none' }} />
                  <div style={{ maxWidth: '120px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {tb.chapter || tb.textbook_name}
                    {tb.has_ocr && <span style={{ marginLeft: '4px', color: '#10B981', fontSize: '10px' }}>✓已识别</span>}
                  </div>
                </label>
              )
            })}
          </div>
        </div>
      )}
      {textbooksLoaded && textbooks.length === 0 && !textbooksLoading && (
        <div style={{ maxWidth: '960px', margin: '16px auto 0', borderRadius: '12px', padding: '16px 20px', background: 'rgba(79,123,232,0.04)', border: '1px dashed rgba(79,123,232,0.25)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
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
        <div style={{ maxWidth: '960px', margin: '16px auto 0', textAlign: 'center', fontSize: '12px', color: C.textMuted }}>加载课本图片...</div>
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

interface ReviewPanelProps {
  review: AIReviewResult
  onApply: (ids?: string[]) => void
  applying: boolean
  isStageMode?: boolean  // 阶段模式下隐藏一键应用（避免与阶段流程冲突）
}

export function ReviewPanel({ review, onApply, applying, isStageMode = false }: ReviewPanelProps) {
  const isGood = review.total_score >= 8.5
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
      {review.good_points.length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.success, marginBottom: '8px' }}>✅ 做得好的</div>
          {review.good_points.map((point, i) => (
            <div key={i} style={{ fontSize: '13px', color: C.text, lineHeight: 1.6, padding: '6px 10px', marginBottom: '4px', background: 'rgba(16,185,129,0.06)', borderRadius: '6px' }}>
              {point}
            </div>
          ))}
        </div>
      )}
      {review.improvements.length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.accent, marginBottom: '8px' }}>💡 可以更好</div>
          {review.improvements.map(imp => (
            <div key={imp.id} style={{ marginBottom: '8px', padding: '10px 12px', background: 'rgba(245,158,11,0.06)', borderRadius: '8px', border: '1px solid rgba(245,158,11,0.15)' }}>
              <div style={{ fontSize: '13px', fontWeight: 500, color: C.text, marginBottom: '4px' }}>{imp.issue}</div>
              <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.6 }}>{imp.suggestion}</div>
            </div>
          ))}
        </div>
      )}
      {/* 阶段模式下隐藏一键应用：修改应通过修订定稿阶段的对话完成 */}
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
