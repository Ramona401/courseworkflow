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
 *
 * v231 新增：课时时长选择
 *   增加 [40, 45, 50, 60] 分钟四档按钮，与专家模式一致。
 *   小学常见 40 分钟课，中学常见 45 分钟课，老师可自由选择。
 *   不选时默认 45 分钟（后端兜底逻辑不变）。
 */
import { useState, useEffect } from 'react'
import type { RecipeSelectionMode } from '@/api/lesson-plans'
import { C, SUBJECTS, GRADES } from '../components/workshopConstants'
import { getMountableUnitPlans, type UnitPlanListItem } from '@/api/unit-plans'
import { getClassProfiles, type ClassProfileListItem } from '@/api/class-profiles'
import { getAvailablePublishers, publisherLabel } from '@/api/course-outlines'
import { getAvailableRecipes, type RecipeListItem } from '@/api/recipes'
import RecipeModeSelector from '../components/RecipeModeSelector'

/** 配方 scope 徽标配置 */
const RECIPE_SCOPE_BADGE: Record<string, { label: string; color: string; bg: string }> = {
  school:   { label: '学校', color: '#166534', bg: '#DCFCE7' },
  group:    { label: '教研组', color: '#1E40AF', bg: '#DBEAFE' },
  personal: { label: '个人', color: '#6B7280', bg: '#F3F4F6' },
}

/** 课时时长可选项（分钟） */
const DURATION_OPTIONS = [40, 45, 50, 60]

interface ConversationStartScreenProps {
  subject: string
  setSubject: (v: string) => void
  grade: string
  setGrade: (v: string) => void
  topic: string
  setTopic: (v: string) => void
  /** 课时时长（分钟）——受控值由父组件持有 */
  duration: number
  setDuration: (v: number) => void
  /** 所属单元方案ID（选填；空串=不挂载）——受控值由父组件持有 */
  unitPlanId: string
  setUnitPlanId: (v: string) => void
  /** 所属班级学情卡ID（选填；空串=不挂载）——受控值由父组件持有 */
  classProfileId: string
  setClassProfileId: (v: string) => void
  /** 选定的课程大纲教材版本（选填；null=不关联大纲；''=通用版；具名=该版本）——受控值由父组件持有 */
  coursePublisher: string | null
  setCoursePublisher: (v: string | null) => void
  /** 配方选择方式：auto=智能选择，selected=指定配方，none=明确不使用 */
  recipeMode: RecipeSelectionMode
  setRecipeMode: (v: RecipeSelectionMode) => void
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
  duration, setDuration,
  unitPlanId, setUnitPlanId,
  classProfileId, setClassProfileId,
  coursePublisher, setCoursePublisher,
  recipeMode, setRecipeMode,
  recipeId, setRecipeId,
  startLoading, onStart, onImport, onSwitchMode,
}: ConversationStartScreenProps) {
  // ===== 配方下拉：按当前学科和具体年级拉可见配方 =====
  const [recipes, setRecipes] = useState<RecipeListItem[]>([])
  const [recipesLoading, setRecipesLoading] = useState(false)

  useEffect(() => {
    let cancelled = false
    if (!subject) { setRecipes([]); return }
    setRecipesLoading(true)
    getAvailableRecipes(subject, grade)
      .then(resp => {
        if (cancelled) return
        const list = resp.recipes || []
        setRecipes(list)
        // 学科或年级切换时：已选配方若不在新列表里则清空
        setRecipeId(
          recipeId && list.some(r => r.id === recipeId)
            ? recipeId
            : recipeMode === 'selected' && list.length > 0
              ? list[0].id
              : ''
        )
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
  }, [subject, grade])

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

  // ===== 班级学情下拉：按当前学科拉本人 active 班级卡 =====
  const [classProfiles, setClassProfiles] = useState<ClassProfileListItem[]>([])
  const [classLoading, setClassLoading] = useState(false)

  useEffect(() => {
    let cancelled = false
    if (!subject) { setClassProfiles([]); return }
    setClassLoading(true)
    getClassProfiles()
      .then(resp => {
        if (cancelled) return
        // 按当前学科过滤（班级卡列表本身是全学科的，挂载选择器按教案学科收窄，减少噪音）
        const list = (resp.profiles || []).filter(p => p.subject === subject)
        setClassProfiles(list)
        setClassProfileId(classProfileId && list.some(p => p.id === classProfileId) ? classProfileId : '')
      })
      .catch(err => {
        if (cancelled) return
        console.error('获取班级学情卡失败:', err)
        setClassProfiles([])
        setClassProfileId('')
      })
      .finally(() => { if (!cancelled) setClassLoading(false) })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [subject])

  // ===== 教材版本下拉：按当前学科+年级拉该学科年级真实存在大纲的可选版本 =====
  const [coursePublishers, setCoursePublishers] = useState<string[]>([])
  const [coursePubLoading, setCoursePubLoading] = useState(false)

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
          // 该学科年级无任何大纲 → 不关联
          setCoursePublisher(null)
        } else if (coursePublisher === null || !pubs.includes(coursePublisher)) {
          // 有大纲：若当前未选/所选版本已不在新列表里 → 默认选中第一个版本（老师可改可清）
          setCoursePublisher(pubs[0])
        }
      })
      .catch(err => {
        if (cancelled) return
        console.error('获取可用教材版本失败:', err)
        setCoursePublishers([])
        setCoursePublisher(null)
      })
      .finally(() => { if (!cancelled) setCoursePubLoading(false) })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [subject, grade])


  /** 时长按钮样式（复用专家模式的 selBtn 交互风格） */
  const durBtn = (active: boolean): React.CSSProperties => ({
    padding: '7px 16px', borderRadius: '20px',
    border: `1.5px solid ${active ? C.primary : C.border}`,
    background: active ? C.primaryLight : 'transparent',
    color: active ? C.primary : C.textSec,
    fontSize: '13px', fontWeight: active ? 600 : 400,
    cursor: 'pointer', transition: 'all 150ms ease',
  })

  const recipeReady =
    recipeMode !== 'selected' || Boolean(recipeId)

  const startButtonText = startLoading
    ? '正在准备备课环境…'
    : recipeMode === 'auto'
      ? '✨ 智能匹配并开始备课'
      : recipeMode === 'selected'
        ? recipeId
          ? '📦 带指定配方开始备课'
          : '请先选择配方'
        : '开始备课（不使用配方）'

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

        {/* ===== 课时时长选择 ===== */}
        <div style={{ marginTop: '14px', textAlign: 'left' }}>
          <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px', paddingLeft: '2px' }}>
            ⏱ 课时时长
          </label>
          <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-start' }}>
            {DURATION_OPTIONS.map(d => (
              <button key={d} onClick={() => setDuration(d)} style={durBtn(duration === d)}>
                {d}分钟
              </button>
            ))}
          </div>
        </div>

        {/* ===== 配方三态选择 ===== */}
        <div style={{ marginTop: '14px', textAlign: 'left' }}>
          <RecipeModeSelector
            mode={recipeMode}
            setMode={setRecipeMode}
            recipes={recipes}
            recipeId={recipeId}
            setRecipeId={setRecipeId}
            loading={recipesLoading}
          />
        </div>

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

        {/* ===== 教材版本（选填）——仅当本学科本年级真实有大纲时才显示 ===== */}
        {coursePublishers.length > 0 && (
          <div style={{ marginTop: '14px', textAlign: 'left' }}>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px', paddingLeft: '2px' }}>
              📚 教材版本
            </label>
            <select
              value={coursePublisher === null ? '__none__' : coursePublisher}
              onChange={e => setCoursePublisher(e.target.value === '__none__' ? null : e.target.value)}
              disabled={coursePubLoading}
              style={{ width: '100%', padding: '11px 14px', borderRadius: '12px', border: `1.5px solid ${coursePublisher !== null ? '#8B5CF6' : C.border}`, fontSize: '14px', color: coursePublisher !== null ? C.text : C.textMuted, background: C.card, cursor: 'pointer', outline: 'none', boxSizing: 'border-box', transition: 'border-color 150ms ease' }}>
              {coursePublishers.map(p => (
                <option key={p || '__generic__'} value={p}>{publisherLabel(p)}</option>
              ))}
              <option value="__none__">不关联大纲（本节课不注入大纲）</option>
            </select>
            {coursePublisher !== null ? (
              <div style={{ fontSize: '11px', color: '#7C3AED', marginTop: '6px', paddingLeft: '2px', lineHeight: 1.5 }}>
                ✓ 备课时将注入「{publisherLabel(coursePublisher)}」这版的课程大纲
              </div>
            ) : (
              <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px', paddingLeft: '2px', lineHeight: 1.5 }}>
                💡 当前不关联大纲；如需对齐教材，请在上方选择你所用的版本
              </div>
            )}
          </div>
        )}

        {/* ===== 本班学情（选填）——仅当本学科有本人班级卡时才显示 ===== */}
        {classProfiles.length > 0 && (
          <div style={{ marginTop: '14px', textAlign: 'left' }}>
            <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px', paddingLeft: '2px' }}>
              👥 本班学情（选填）
            </label>
            <select value={classProfileId} onChange={e => setClassProfileId(e.target.value)} disabled={classLoading}
              style={{ width: '100%', padding: '11px 14px', borderRadius: '12px', border: `1.5px solid ${classProfileId ? '#10B981' : C.border}`, fontSize: '14px', color: classProfileId ? C.text : C.textMuted, background: C.card, cursor: 'pointer', outline: 'none', boxSizing: 'border-box', transition: 'border-color 150ms ease' }}>
              <option value="">不关联班级学情（不做差异化）</option>
              {classProfiles.map(p => (
                <option key={p.id} value={p.id}>
                  {p.class_name}{p.grade ? ` · ${p.grade}` : ''}{p.student_count > 0 ? ` · ${p.student_count}人` : ''}{p.has_profile ? '' : '（待完善）'}
                </option>
              ))}
            </select>
            {classProfileId && (
              <div style={{ fontSize: '11px', color: '#059669', marginTop: '6px', paddingLeft: '2px', lineHeight: 1.5 }}>
                ✓ 备课时 AI 会针对这个班的分层结构与薄弱点做差异化教学设计
              </div>
            )}
          </div>
        )}

        <button
          onClick={onStart}
          disabled={!topic.trim() || startLoading || !recipeReady}
          style={{
            marginTop: '22px',
            padding: '13px 52px',
            borderRadius: '14px',
            border: 'none',
            background:
              (!topic.trim() || startLoading || !recipeReady)
                ? '#E5E7EB'
                : `linear-gradient(135deg, ${C.primary}, #818CF8)`,
            color:
              (!topic.trim() || startLoading || !recipeReady)
                ? C.textMuted
                : '#fff',
            fontSize: '16px',
            fontWeight: 700,
            cursor:
              (!topic.trim() || startLoading || !recipeReady)
                ? 'not-allowed'
                : 'pointer',
            boxShadow:
              (!topic.trim() || startLoading || !recipeReady)
                ? 'none'
                : '0 6px 20px rgba(79,123,232,0.35)',
            transition: 'all 200ms ease',
          }}
        >
          {startButtonText}
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
