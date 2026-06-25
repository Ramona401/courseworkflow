/**
 * ConversationStartScreen.tsx — 对话模式极简首屏
 *
 * 设计依据：产品设计文档 2.1「首屏」——课题 + 学科/年级 + 配方选择 + 单元方案选择。
 *
 * v203 新增：配方下拉选择
 *   配方（教研组/学校共享的教案结构+流程+学情规范）是教研共识的载体。
 *   不选配方时 AI 用系统预置骨架；选了配方则配方定义的结构高于预置。
 *   下拉按「学校>教研组>个人」排序，scope 徽章标识来源；
 *   有配方就显示，没有就不打扰（零负担设计）。
 *   与助手的分工：配方定"教案长什么样"，助手定"AI怎么陪你备课"。
 *
 * 大单元挂载（前端入口·起步选）：
 *   「所属单元方案（选填）」下拉——有就选上，没有就空着走原流程。
 */
import { useState, useEffect } from 'react'
import { C, SUBJECTS, GRADES } from '../components/workshopConstants'
import { getMountableUnitPlans, type UnitPlanListItem } from '@/api/unit-plans'
import { getAvailableRecipes, type RecipeListItem } from '@/api/recipes'

/** 配方 scope 徽标配置 */
const RECIPE_SCOPE_BADGE: Record<string, { label: string; color: string; bg: string }> = {
  school:   { label: '学校', color: '#166534', bg: '#DCFCE7' },
  group:    { label: '教研组', color: '#1E40AF', bg: '#DBEAFE' },
  personal: { label: '个人', color: '#6B7280', bg: '#F3F4F6' },
}

interface ConversationStartScreenProps {
  subject: string
  setSubject: (v: string) => void
  grade: string
  setGrade: (v: string) => void
  topic: string
  setTopic: (v: string) => void
  /** 所属单元方案ID（选填；空串=不挂载）——受控值由父组件持有 */
  unitPlanId: string
  setUnitPlanId: (v: string) => void
  /** 选中的配方ID（选填；空串=不使用配方）——受控值由父组件持有 */
  recipeId: string
  setRecipeId: (v: string) => void
  startLoading: boolean
  /** 点击「开始备课」 */
  onStart: () => void
  /** 点击「导入已有教案」 */
  onImport: () => void
  /** 切换专家模式（可选） */
  onSwitchMode?: () => void
}

export default function ConversationStartScreen({
  subject, setSubject, grade, setGrade, topic, setTopic,
  unitPlanId, setUnitPlanId,
  recipeId, setRecipeId,
  startLoading, onStart, onImport, onSwitchMode,
}: ConversationStartScreenProps) {
  // ===== 配方下拉：按当前学科拉可见配方 =====
  const [recipes, setRecipes] = useState<RecipeListItem[]>([])
  const [recipesLoading, setRecipesLoading] = useState(false)

  useEffect(() => {
    let cancelled = false
    if (!subject) { setRecipes([]); return }
    setRecipesLoading(true)
    getAvailableRecipes(subject)
      .then(resp => {
        if (cancelled) return
        const list = resp.recipes || []
        setRecipes(list)
        // 学科切换时：已选配方若不在新学科列表里则清空
        setRecipeId(recipeId && list.some(r => r.id === recipeId) ? recipeId : '')
      })
      .catch(err => {
        if (cancelled) return
        console.error('获取可用配方失败:', err)
        setRecipes([])
        setRecipeId('')
      })
      .finally(() => { if (!cancelled) setRecipesLoading(false) })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [subject])

  // ===== 单元方案下拉：按当前学科拉可挂载方案 =====
  const [unitPlans, setUnitPlans] = useState<UnitPlanListItem[]>([])
  const [unitLoading, setUnitLoading] = useState(false)

  useEffect(() => {
    let cancelled = false
    if (!subject) { setUnitPlans([]); return }
    setUnitLoading(true)
    getMountableUnitPlans(subject)
      .then(resp => {
        if (cancelled) return
        const list = resp.unit_plans || []
        setUnitPlans(list)
        setUnitPlanId(unitPlanId && list.some(p => p.id === unitPlanId) ? unitPlanId : '')
      })
      .catch(err => {
        if (cancelled) return
        console.error('获取可挂载单元方案失败:', err)
        setUnitPlans([])
        setUnitPlanId('')
      })
      .finally(() => { if (!cancelled) setUnitLoading(false) })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [subject])

  const selectedRecipe = recipes.find(r => r.id === recipeId)

  return (
    <div style={{ height: 'calc(100vh - 120px)', overflow: 'auto', margin: '-28px -32px', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: '24px' }}>
      <div style={{ width: '100%', maxWidth: '520px', textAlign: 'center' }}>
        <h1 style={{ fontSize: '26px', fontWeight: 700, color: C.text, margin: '0 0 28px' }}>今天备什么课？</h1>
        <input
          type="text" value={topic} onChange={e => setTopic(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && onStart()}
          placeholder="课题，如：观潮（第二课时）"
          style={{ width: '100%', padding: '15px 18px', borderRadius: '14px', border: `2px solid ${C.border}`, fontSize: '16px', color: C.text, outline: 'none', boxSizing: 'border-box', transition: 'border-color 150ms ease', boxShadow: '0 4px 18px rgba(0,0,0,0.05)' }}
          onFocus={e => { e.target.style.borderColor = C.primary }}
          onBlur={e => { e.target.style.borderColor = C.border }}
        />
        <div style={{ display: 'flex', gap: '10px', justifyContent: 'center', marginTop: '16px' }}>
          <select value={subject} onChange={e => setSubject(e.target.value)}
            style={{ padding: '9px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', color: C.text, background: C.card, cursor: 'pointer', outline: 'none' }}>
            {SUBJECTS.map(s => <option key={s} value={s}>{s}</option>)}
          </select>
          <select value={grade} onChange={e => setGrade(e.target.value)}
            style={{ padding: '9px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', color: C.text, background: C.card, cursor: 'pointer', outline: 'none' }}>
            {GRADES.map(g => <option key={g} value={g}>{g}</option>)}
          </select>
        </div>

        {/* ===== 配方选择（选填）——仅当本学科有可用配方时才显示 ===== */}
        {recipes.length > 0 && (
          <div style={{ marginTop: '14px', textAlign: 'left' }}>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px', paddingLeft: '2px' }}>
              📦 备课配方（选填）
            </label>
            <select value={recipeId} onChange={e => setRecipeId(e.target.value)} disabled={recipesLoading}
              style={{ width: '100%', padding: '11px 14px', borderRadius: '12px', border: `1.5px solid ${recipeId ? '#F59E0B' : C.border}`, fontSize: '14px', color: recipeId ? C.text : C.textMuted, background: C.card, cursor: 'pointer', outline: 'none', boxSizing: 'border-box', transition: 'border-color 150ms ease' }}>
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
              <div style={{ fontSize: '11px', color: '#92400E', marginTop: '6px', paddingLeft: '2px', lineHeight: 1.5, background: '#FFFBEB', borderRadius: '6px', padding: '6px 10px' }}>
                ✅ 已选「{selectedRecipe.name}」— {selectedRecipe.description || '教案结构+流程+学情由此配方定义'}
              </div>
            )}
            {!recipeId && (
              <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px', paddingLeft: '2px', lineHeight: 1.5 }}>
                💡 不选配方时，AI助手的指引（含结构）将作为唯一个性化来源
              </div>
            )}
          </div>
        )}

        {/* ===== 单元方案（选填）——仅当本学科有可挂载方案时才显示 ===== */}
        {unitPlans.length > 0 && (
          <div style={{ marginTop: '14px', textAlign: 'left' }}>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px', paddingLeft: '2px' }}>
              📐 所属单元方案（选填）
            </label>
            <select value={unitPlanId} onChange={e => setUnitPlanId(e.target.value)} disabled={unitLoading}
              style={{ width: '100%', padding: '11px 14px', borderRadius: '12px', border: `1.5px solid ${unitPlanId ? C.primary : C.border}`, fontSize: '14px', color: unitPlanId ? C.text : C.textMuted, background: C.card, cursor: 'pointer', outline: 'none', boxSizing: 'border-box', transition: 'border-color 150ms ease' }}>
              <option value="">不关联单元方案（普通备课）</option>
              {unitPlans.map(p => (
                <option key={p.id} value={p.id}>
                  {p.grade}{p.volume ? ` · ${p.volume}` : ''} · {p.unit}{p.unit_theme ? `（${p.unit_theme}）` : ''}
                </option>
              ))}
            </select>
            {unitPlanId && (
              <div style={{ fontSize: '11px', color: C.primary, marginTop: '6px', paddingLeft: '2px', lineHeight: 1.5 }}>
                ✓ 备课全程将贴着这份单元方案的整体设计来展开
              </div>
            )}
          </div>
        )}

        <button onClick={onStart} disabled={!topic.trim() || startLoading}
          style={{ marginTop: '22px', padding: '13px 52px', borderRadius: '14px', border: 'none', background: (!topic.trim() || startLoading) ? '#E5E7EB' : `linear-gradient(135deg, ${C.primary}, #818CF8)`, color: (!topic.trim() || startLoading) ? C.textMuted : '#fff', fontSize: '16px', fontWeight: 700, cursor: (!topic.trim() || startLoading) ? 'not-allowed' : 'pointer', boxShadow: (!topic.trim() || startLoading) ? 'none' : '0 6px 20px rgba(79,123,232,0.35)', transition: 'all 200ms ease' }}>
          {startLoading ? '正在准备备课环境…' : recipeId ? '📦 带配方开始备课' : '开始备课'}
        </button>
        <div style={{ borderTop: `1px solid ${C.border}`, margin: '28px 0 16px' }} />
        <div style={{ display: 'flex', justifyContent: 'center', gap: '20px', fontSize: '13px' }}>
          <button onClick={onImport} style={{ background: 'none', border: 'none', color: C.textSec, cursor: 'pointer', padding: '4px 8px' }}
            onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.color = C.primary }}
            onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.color = C.textSec }}>
            📂 导入已有教案
          </button>
          {onSwitchMode && (
            <button onClick={onSwitchMode} style={{ background: 'none', border: 'none', color: C.textSec, cursor: 'pointer', padding: '4px 8px' }}
              onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.color = C.primary }}
              onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.color = C.textSec }}>
              ⚙ 切换专家模式
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
