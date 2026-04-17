/**
 * RecipeEditorPage — 备课配方编辑器
 *
 * Phase 7A：基础CRUD
 * 迭代1：教案结构表格编辑器 + 备课模式选择
 * 迭代2：流程搭建器（拖拽排序+启用/禁用+三级提示+预设模板+实时校验）
 * 迭代4：per_stage模式前端支持（备课模式3选1+阶段级对话模式下拉）
 * 迭代5：自定义阶段编辑器（添加/编辑/删除自定义阶段，与系统阶段混排）
 * v88-fix：sessionStorage草稿自动保存 — 编辑过程中自动暂存，页面离开/刷新不丢失
 *
 * 路由：/lesson-plans/recipes/new | /lesson-plans/recipes/:id/edit
 */
import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import {
  getRecipe, createRecipe, updateRecipe, previewRecipeContext,
  smartRecommendComponents, getFlowPresets, validateFlow,
  getCustomStages, createCustomStage, updateCustomStage, deleteCustomStage,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  type RecipeDetail, type RecipeContextPreview, type RecommendedComponentGroup,
  type LessonStructureBlock, type LessonStructureSubSection, type PromptMode,
  type StageFlowItem, type FlowPreset, type FlowValidationMessage,
  type CustomStageResponse, type CreateCustomStageRequest, type UpdateCustomStageRequest,
} from '@/api/recipes'
import {
  STAGE_CODE_EMOJI, STAGE_CODE_NAME, STAGE_CODE_ROLE, STAGE_CODE_DESC,
  STAGE_REMOVABLE, FLOW_MSG_COLORS, PROMPT_MODE_OPTIONS, STAGE_PROMPT_MODE_OPTIONS,
  SUBJECTS
} from '../workshop/components/workshopConstants'
import CustomStageModal from './components/CustomStageModal'

/* ==================== 颜色常量 ==================== */
const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981', danger: '#EF4444', accent: '#F59E0B',
  text: '#1F2937', textSec: '#6B7280', textMuted: '#9CA3AF',
  card: '#FFFFFF', border: '#F3F4F6', borderHover: '#E5E7EB', bg: '#FAFBFC',
}

const GRADES = ['七年级','八年级','九年级','高一','高二','高三','小学低段','小学中段','小学高段']

/* ==================== 默认教案结构模板 ==================== */
const DEFAULT_LESSON_STRUCTURE: LessonStructureBlock[] = [
  { name: '教学目标', required: true, requirement: '知识与技能、过程与方法、情感态度价值观三维目标', order: 1 },
  { name: '教学重难点', required: true, requirement: '重点2-3个，难点1-2个', order: 2 },
  { name: '课前准备', required: false, requirement: '教师准备和学生准备分开列', order: 3 },
  { name: '教学过程', required: true, requirement: '按以下环节输出，每环节含教师活动和学生活动', order: 4, sub_sections: [
    { name: '导入', duration: 5, goal: '激发兴趣，建立新旧知识连接', output_requirement: '用生活场景切入' },
    { name: '新授', duration: 20, goal: '理解概念，掌握操作', output_requirement: '讲+演+练交替，写教师话术' },
    { name: '练习', duration: 15, goal: '分层巩固', output_requirement: '基础+进阶任务' },
    { name: '小结', duration: 5, goal: '归纳要点', output_requirement: '学生总结，布置思考题' },
  ]},
  { name: '作业设计', required: false, requirement: '分层作业：基础题+提高题', order: 5 },
  { name: '板书设计', required: false, requirement: '思维导图式板书', order: 6 },
]

/* ==================== 默认5阶段全开流程 ==================== */
const DEFAULT_FLOW: StageFlowItem[] = [
  { stage_code: 'analyze', enabled: true, order: 1 },
  { stage_code: 'design',  enabled: true, order: 2 },
  { stage_code: 'write',   enabled: true, order: 3 },
  { stage_code: 'review',  enabled: true, order: 4 },
  { stage_code: 'revise',  enabled: true, order: 5 },
]

/* ==================== v88-fix：草稿自动保存辅助函数 ==================== */

/** 构建草稿在sessionStorage中的key */
const getDraftKey = (recipeId?: string) => recipeId ? `recipe_draft_${recipeId}` : 'recipe_draft_new'

/** 草稿数据结构（包含所有表单字段） */
interface RecipeDraft {
  name: string; description: string; subject: string; gradeRange: string
  studentProfile: string; teachingStyle: string; schoolRequirements: string
  customNotes: string; customPrompt: string; selectedCompIds: string[]
  lessonStructure: LessonStructureBlock[]; promptMode: PromptMode
  stageFlow: StageFlowItem[]; savedAt: number  // 保存时间戳（毫秒）
}

/** 保存草稿到sessionStorage */
const saveDraft = (key: string, draft: RecipeDraft) => {
  try { sessionStorage.setItem(key, JSON.stringify(draft)) } catch { /* sessionStorage满或不可用时静默忽略 */ }
}

/** 从sessionStorage读取草稿 */
const loadDraft = (key: string): RecipeDraft | null => {
  try {
    const raw = sessionStorage.getItem(key)
    if (!raw) return null
    const draft = JSON.parse(raw) as RecipeDraft
    // 兼容转换：旧草稿的subject='AI'自动转为'人工智能'
    if (draft.subject === 'AI') {
      draft.subject = '人工智能'
    }
    // 草稿超过24小时自动过期
    if (Date.now() - draft.savedAt > 24 * 60 * 60 * 1000) {
      sessionStorage.removeItem(key)
      return null
    }
    return draft
  } catch { return null }
}

/** 清除草稿 */
const clearDraft = (key: string) => {
  try { sessionStorage.removeItem(key) } catch { /* 忽略 */ }
}

/* ==================== 样式辅助 ==================== */
const labelStyle: React.CSSProperties = { display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }
const textareaStyle: React.CSSProperties = {
  width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`,
  fontSize: '14px', color: C.text, outline: 'none', boxSizing: 'border-box', fontFamily: 'inherit',
  lineHeight: 1.6, resize: 'vertical', transition: 'border-color 150ms ease',
}
const inputStyle: React.CSSProperties = { ...textareaStyle, resize: 'none' as const }
const selBtn = (active: boolean): React.CSSProperties => ({
  padding: '6px 14px', borderRadius: '20px', border: `1px solid ${active ? C.primary : C.border}`,
  background: active ? C.primaryLight : 'transparent', color: active ? C.primary : C.textSec,
  fontSize: '13px', fontWeight: active ? 600 : 400, cursor: 'pointer', transition: 'all 150ms ease',
})
const smallInput: React.CSSProperties = {
  padding: '6px 10px', borderRadius: '6px', border: `1px solid ${C.border}`,
  fontSize: '13px', color: C.text, outline: 'none', boxSizing: 'border-box' as const, fontFamily: 'inherit',
}

/* ==================== 主组件 ==================== */
export default function RecipeEditorPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isEdit = !!id
  const location = useLocation()
  const fromPath = (location.state as { from?: string } | null)?.from || '/lesson-plans/recipes'

  // v88-fix：草稿key（编辑模式用配方ID，新建模式用固定key）
  const draftKey = getDraftKey(id)

  // ---- 表单状态 ----
  const [name, setName] = useState(''); const [description, setDescription] = useState('')
  const [subject, setSubject] = useState('人工智能'); const [gradeRange, setGradeRange] = useState('七年级')
  const [studentProfile, setStudentProfile] = useState(''); const [teachingStyle, setTeachingStyle] = useState('')
  const [schoolRequirements, setSchoolReqs] = useState(''); const [customNotes, setCustomNotes] = useState('')
  const [customPrompt, setCustomPrompt] = useState(''); const [selectedCompIds, setSelectedCompIds] = useState<Set<string>>(new Set())

  // ---- 迭代1：教案结构 + 备课模式 ----
  const [lessonStructure, setLessonStructure] = useState<LessonStructureBlock[]>([])
  const [promptMode, setPromptMode] = useState<PromptMode>('guided')

  // ---- 迭代2：流程配置 ----
  const [stageFlow, setStageFlow] = useState<StageFlowItem[]>(DEFAULT_FLOW)
  const [flowPresets, setFlowPresets] = useState<FlowPreset[]>([])
  const [flowMessages, setFlowMessages] = useState<FlowValidationMessage[]>([])
  const [flowValid, setFlowValid] = useState(true)

  // ---- 迭代5：自定义阶段 ----
  const [customStages, setCustomStages] = useState<CustomStageResponse[]>([])
  const [stageModalMode, setStageModalMode] = useState<'create' | 'edit' | null>(null)
  const [editingStageCode, setEditingStageCode] = useState<string>('')
  const [editingStageData, setEditingStageData] = useState<Record<string, unknown> | null>(null)
  const [stageSaving, setStageSaving] = useState(false)

  // ---- 推荐组件 ----
  const [compGroups, setCompGroups] = useState<RecommendedComponentGroup[]>([])
  const [compLoading, setCompLoading] = useState(false)

  // ---- 预览 ----
  const [preview, setPreview] = useState<RecipeContextPreview | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)

  // ---- 页面状态 ----
  const [pageLoading, setPageLoading] = useState(isEdit)
  const [saving, setSaving] = useState(false)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  // v88-fix：是否已从后端/草稿加载完成（避免初始状态触发草稿保存）
  const dataLoadedRef = useRef(false)
  // v88-fix：是否恢复了草稿（用于显示提示横幅）
  const [draftRestored, setDraftRestored] = useState(false)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type }); setTimeout(() => setToast(null), 3000)
  }

  // ==================== v88-fix：自动保存草稿（防抖500ms）====================
  useEffect(() => {
    if (!dataLoadedRef.current) return  // 数据未加载完成前不保存
    const timer = setTimeout(() => {
      const draft: RecipeDraft = {
        name, description, subject, gradeRange: gradeRange,
        studentProfile, teachingStyle, schoolRequirements: schoolRequirements,
        customNotes, customPrompt, selectedCompIds: Array.from(selectedCompIds),
        lessonStructure, promptMode, stageFlow, savedAt: Date.now(),
      }
      saveDraft(draftKey, draft)
    }, 500)
    return () => clearTimeout(timer)
  }, [name, description, subject, gradeRange, studentProfile, teachingStyle,
      schoolRequirements, customNotes, customPrompt, selectedCompIds,
      lessonStructure, promptMode, stageFlow, draftKey])

  // ==================== 迭代4：切换备课模式时清理/初始化阶段级prompt_mode ====================
  const handlePromptModeChange = useCallback((newMode: PromptMode) => {
    setPromptMode(newMode)
    if (newMode === 'per_stage') {
      setStageFlow(prev => prev.map(s => ({ ...s, prompt_mode: s.prompt_mode || 'guided' })))
    } else {
      setStageFlow(prev => prev.map(s => {
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { prompt_mode: _, ...rest } = s
        return rest as StageFlowItem
      }))
    }
  }, [])

  // ==================== 迭代4：更新单个阶段的对话模式 ====================
  const updateStagePromptMode = useCallback((stageCode: string, mode: string) => {
    setStageFlow(prev => prev.map(s =>
      s.stage_code === stageCode ? { ...s, prompt_mode: mode } : s
    ))
  }, [])

  // ==================== 加载预设流程模板（迭代2）====================
  useEffect(() => {
    getFlowPresets().then(resp => setFlowPresets(resp.presets || [])).catch(() => {})
  }, [])

  // ==================== 迭代5：加载自定义阶段 ====================
  const loadCustomStages = useCallback(async (recipeId: string) => {
    try {
      const resp = await getCustomStages(recipeId)
      setCustomStages(resp.stages || [])
    } catch { setCustomStages([]) }
  }, [])

  // ==================== 流程实时校验（迭代2）====================
  const triggerValidation = useCallback(async (flow: StageFlowItem[]) => {
    try {
      // 校验时只发送系统阶段（自定义阶段不参与系统规则校验）
      const systemFlow = flow.filter(s => !s.is_custom)
      const result = await validateFlow(systemFlow)
      setFlowMessages(result.messages || [])
      setFlowValid(result.valid)
    } catch { setFlowMessages([]); setFlowValid(true) }
  }, [])

  useEffect(() => { triggerValidation(stageFlow) }, [stageFlow, triggerValidation])

  // ==================== 编辑模式：加载配方数据（v88-fix：优先恢复草稿）====================
  useEffect(() => {
    if (!id) {
      // 新建模式：尝试恢复草稿
      const draft = loadDraft(draftKey)
      if (draft) {
        setName(draft.name); setDescription(draft.description)
        setSubject(draft.subject); setGradeRange(draft.gradeRange)
        setStudentProfile(draft.studentProfile); setTeachingStyle(draft.teachingStyle)
        setSchoolReqs(draft.schoolRequirements); setCustomNotes(draft.customNotes)
        setCustomPrompt(draft.customPrompt); setSelectedCompIds(new Set(draft.selectedCompIds))
        if (draft.lessonStructure.length > 0) setLessonStructure(draft.lessonStructure)
        setPromptMode(draft.promptMode)
        if (draft.stageFlow.length > 0) setStageFlow(draft.stageFlow)
        setDraftRestored(true)
      }
      dataLoadedRef.current = true
      return
    }
    const load = async () => {
      try {
        const detail = await getRecipe(id)

        // v88-fix：检查是否有比后端数据更新的草稿
        const draft = loadDraft(draftKey)
        const detailUpdatedAt = new Date(detail.updated_at).getTime()
        const hasFresherDraft = draft && draft.savedAt > detailUpdatedAt

        if (hasFresherDraft && draft) {
          // 草稿比后端新 → 恢复草稿内容
          setName(draft.name); setDescription(draft.description)
          setSubject(draft.subject); setGradeRange(draft.gradeRange)
          setStudentProfile(draft.studentProfile); setTeachingStyle(draft.teachingStyle)
          setSchoolReqs(draft.schoolRequirements); setCustomNotes(draft.customNotes)
          setCustomPrompt(draft.customPrompt); setSelectedCompIds(new Set(draft.selectedCompIds))
          if (draft.lessonStructure.length > 0) setLessonStructure(draft.lessonStructure)
          setPromptMode(draft.promptMode)
          if (draft.stageFlow.length > 0) setStageFlow(draft.stageFlow)
          setDraftRestored(true)
        } else {
          // 无草稿或后端更新 → 从后端加载
          setName(detail.name); setDescription(detail.description || '')
          setSubject(detail.subject); setGradeRange(detail.grade_range)
          setStudentProfile(detail.student_profile || ''); setTeachingStyle(detail.teaching_style || '')
          setSchoolReqs(detail.school_requirements || ''); setCustomNotes(detail.custom_notes || '')
          setCustomPrompt(detail.custom_prompt || '')
          let compIds: string[] = []
          if (detail.component_ids) {
            try { compIds = JSON.parse(detail.component_ids) } catch { compIds = [] }
          }
          setSelectedCompIds(new Set(compIds))
          if (detail.lesson_structure) {
            try { const p = JSON.parse(detail.lesson_structure); if (Array.isArray(p) && p.length > 0) setLessonStructure(p) } catch { /* 忽略 */ }
          }
          if (detail.prompt_mode) setPromptMode(detail.prompt_mode)
          if (detail.stages_config) {
            try {
              const p = JSON.parse(detail.stages_config)
              if (Array.isArray(p) && p.length > 0 && p[0].enabled !== undefined) setStageFlow(p)
            } catch { /* 忽略 */ }
          }
        }
        // 迭代5：加载自定义阶段（无论是否恢复草稿都需要）
        await loadCustomStages(id)
      } catch (e) { console.error('加载配方失败:', e); showToast('加载配方失败', 'error') }
      finally {
        setPageLoading(false)
        dataLoadedRef.current = true
      }
    }
    load()
  }, [id, loadCustomStages, draftKey])

  // ==================== 加载推荐组件 ====================
  const loadRecommend = useCallback(async () => {
    if (!subject || !gradeRange) return
    setCompLoading(true)
    try { const resp = await smartRecommendComponents({ subject, grade_range: gradeRange }); setCompGroups(resp.groups || []) }
    catch (e) { console.error('加载推荐组件失败:', e) }
    finally { setCompLoading(false) }
  }, [subject, gradeRange])
  useEffect(() => { loadRecommend() }, [loadRecommend])

  const toggleComp = (compId: string) => {
    // eslint-disable-next-line @typescript-eslint/no-unused-expressions
    setSelectedCompIds(prev => { const next = new Set(prev); next.has(compId) ? next.delete(compId) : next.add(compId); return next })
  }

  // ==================== 预览上下文 ====================
  const handlePreview = async (recipeId: string) => {
    setPreviewLoading(true)
    try { setPreview(await previewRecipeContext(recipeId)) }
    catch { showToast('预览上下文失败', 'error') }
    finally { setPreviewLoading(false) }
  }

  // ==================== 教案结构操作 ====================
  const totalDuration = useMemo(() => {
    const b = lessonStructure.find(b => b.sub_sections && b.sub_sections.length > 0)
    return b?.sub_sections?.reduce((s, x) => s + (x.duration || 0), 0) || 0
  }, [lessonStructure])

  const addBlock = () => setLessonStructure(p => [...p, { name: '', required: false, requirement: '', order: p.length + 1 }])
  const removeBlock = (i: number) => setLessonStructure(p => p.filter((_, j) => j !== i).map((b, j) => ({ ...b, order: j + 1 })))
  const updateBlock = (i: number, f: keyof LessonStructureBlock, v: unknown) => setLessonStructure(p => p.map((b, j) => j === i ? { ...b, [f]: v } : b))
  const addSubSection = (bi: number) => setLessonStructure(p => p.map((b, i) => i !== bi ? b : { ...b, sub_sections: [...(b.sub_sections || []), { name: '', duration: 5, goal: '', output_requirement: '' }] }))
  const removeSubSection = (bi: number, si: number) => setLessonStructure(p => p.map((b, i) => i !== bi ? b : { ...b, sub_sections: (b.sub_sections || []).filter((_, j) => j !== si) }))
  const updateSubSection = (bi: number, si: number, f: keyof LessonStructureSubSection, v: unknown) => setLessonStructure(p => p.map((b, i) => i !== bi ? b : { ...b, sub_sections: (b.sub_sections || []).map((s, j) => j === si ? { ...s, [f]: v } : s) }))
  const loadDefaultStructure = () => setLessonStructure(DEFAULT_LESSON_STRUCTURE)
  const moveBlock = (i: number, d: -1 | 1) => {
    const t = i + d; if (t < 0 || t >= lessonStructure.length) return
    setLessonStructure(p => { const n = [...p]; [n[i], n[t]] = [n[t], n[i]]; return n.map((b, j) => ({ ...b, order: j + 1 })) })
  }

  // ==================== 迭代2+5：流程操作 ====================
  const toggleStage = (code: string) => {
    const isCustom = stageFlow.find(s => s.stage_code === code)?.is_custom
    if (!isCustom && STAGE_REMOVABLE[code] === false) return
    setStageFlow(prev => prev.map(s => s.stage_code === code ? { ...s, enabled: !s.enabled } : s))
  }
  const moveStage = (idx: number, dir: -1 | 1) => {
    const target = idx + dir
    if (target < 0 || target >= stageFlow.length) return
    if (stageFlow[idx].stage_code === 'revise' || stageFlow[target].stage_code === 'revise') return
    setStageFlow(prev => {
      const n = [...prev]; [n[idx], n[target]] = [n[target], n[idx]]
      return n.map((s, i) => ({ ...s, order: i + 1 }))
    })
  }
  const applyPreset = (preset: FlowPreset) => {
    const customItems = stageFlow.filter(s => s.is_custom)
    let newFlow = preset.stages.map(s => ({ ...s }))
    if (customItems.length > 0) {
      const reviseIdx = newFlow.findIndex(s => s.stage_code === 'revise')
      if (reviseIdx >= 0) {
        const before = newFlow.slice(0, reviseIdx)
        const after = newFlow.slice(reviseIdx)
        newFlow = [...before, ...customItems.map((s, i) => ({ ...s, enabled: true, order: before.length + i + 1 })), ...after]
      } else {
        newFlow = [...newFlow, ...customItems.map((s, i) => ({ ...s, enabled: true, order: newFlow.length + i + 1 }))]
      }
    }
    setStageFlow(newFlow.map((s, i) => ({ ...s, order: i + 1 })))
    handlePromptModeChange(preset.prompt_mode)
    showToast(`已应用「${preset.name}」模板`)
  }
  const enabledCount = useMemo(() => stageFlow.filter(s => s.enabled).length, [stageFlow])

  // ==================== 迭代5：自定义阶段操作 ====================
  const openCreateStageModal = () => {
    if (!isEdit || !id) { showToast('请先保存配方后再添加自定义阶段', 'error'); return }
    setStageModalMode('create')
    setEditingStageCode('')
    setEditingStageData(null)
  }

  const openEditStageModal = async (stageCode: string) => {
    if (!id) return
    const cs = customStages.find(s => s.stage_code === stageCode)
    if (!cs) return
    setStageModalMode('edit')
    setEditingStageCode(stageCode)
    setEditingStageData({
      stage_code: cs.stage_code,
      stage_name: cs.stage_name,
      ai_role: cs.ai_role,
      system_prompt: '',
      prompt_variants: '{}',
      output_format: '{}',
      gate_mode: cs.gate_mode,
      skippable: cs.skippable,
    })
  }

  const handleStageModalConfirm = async (data: CreateCustomStageRequest | UpdateCustomStageRequest) => {
    if (!id) return
    setStageSaving(true)
    try {
      if (stageModalMode === 'create') {
        const created = await createCustomStage(id, data as CreateCustomStageRequest)
        setStageFlow(prev => {
          const reviseIdx = prev.findIndex(s => s.stage_code === 'revise')
          const newItem: StageFlowItem = {
            stage_code: created.stage_code, enabled: true, order: 0,
            is_custom: true, stage_name: created.stage_name,
          }
          let updated: StageFlowItem[]
          if (reviseIdx >= 0) {
            updated = [...prev.slice(0, reviseIdx), newItem, ...prev.slice(reviseIdx)]
          } else {
            updated = [...prev, newItem]
          }
          return updated.map((s, i) => ({ ...s, order: i + 1 }))
        })
        await loadCustomStages(id)
        showToast(`自定义阶段「${created.stage_name}」已添加`)
      } else if (stageModalMode === 'edit') {
        await updateCustomStage(id, editingStageCode, data as UpdateCustomStageRequest)
        const upd = data as UpdateCustomStageRequest
        setStageFlow(prev => prev.map(s =>
          s.stage_code === editingStageCode ? { ...s, stage_name: upd.stage_name } : s
        ))
        await loadCustomStages(id)
        showToast('自定义阶段已更新')
      }
      setStageModalMode(null)
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '操作失败', 'error')
    } finally { setStageSaving(false) }
  }

  const handleDeleteCustomStage = async (stageCode: string) => {
    if (!id) return
    const cs = customStages.find(s => s.stage_code === stageCode)
    if (!confirm(`确认删除自定义阶段「${cs?.stage_name || stageCode}」？`)) return
    try {
      await deleteCustomStage(id, stageCode)
      setStageFlow(prev => prev.filter(s => s.stage_code !== stageCode).map((s, i) => ({ ...s, order: i + 1 })))
      await loadCustomStages(id)
      showToast('自定义阶段已删除')
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '删除失败', 'error')
    }
  }

  // ==================== 保存（v88-fix：保存成功后清除草稿）====================
  const handleSave = async () => {
    if (!name.trim()) { showToast('请填写配方名称', 'error'); return }
    if (!flowValid) { showToast('流程配置有阻断错误，请先修正', 'error'); return }
    setSaving(true)
    try {
      const payload = {
        name: name.trim(), description: description.trim(), subject, grade_range: gradeRange,
        component_ids: Array.from(selectedCompIds),
        student_profile: studentProfile.trim(), teaching_style: teachingStyle.trim(),
        school_requirements: schoolRequirements.trim(), custom_notes: customNotes.trim(),
        custom_prompt: customPrompt.trim(),
        lesson_structure: lessonStructure.length > 0 ? JSON.stringify(lessonStructure) : '[]',
        prompt_mode: promptMode,
        stages_config: JSON.stringify(stageFlow),
      }
      if (isEdit && id) {
        await updateRecipe(id, payload)
        clearDraft(draftKey)  // v88-fix：保存成功后清除草稿
        setDraftRestored(false)
        showToast('配方已更新 ✓'); handlePreview(id)
      } else {
        const created = await createRecipe(payload)
        clearDraft(draftKey)  // v88-fix：保存成功后清除草稿
        setDraftRestored(false)
        showToast('配方已创建 ✓')
        navigate(`/lesson-plans/recipes/${created.id}/edit`, { replace: true, state: { from: fromPath } })
      }
    } catch (e: unknown) {
      console.error('保存失败:', e); showToast(e instanceof Error ? e.message : '保存失败', 'error')
    } finally { setSaving(false) }
  }

  // ==================== v88-fix：放弃草稿，重新从后端加载 ====================
  const handleDiscardDraft = () => {
    clearDraft(draftKey)
    setDraftRestored(false)
    // 重新加载页面以从后端获取最新数据
    window.location.reload()
  }

  // ==================== 获取阶段显示信息辅助函数（迭代5）====================
  const getStageName = (stage: StageFlowItem) => {
    if (stage.is_custom) return stage.stage_name || stage.stage_code
    return STAGE_CODE_NAME[stage.stage_code] || stage.stage_code
  }
  const getStageEmoji = (stage: StageFlowItem) => {
    if (stage.is_custom) return '🔧'
    return STAGE_CODE_EMOJI[stage.stage_code] || '📋'
  }
  const getStageRole = (stage: StageFlowItem) => {
    if (stage.is_custom) {
      const cs = customStages.find(c => c.stage_code === stage.stage_code)
      return cs?.ai_role || '自定义角色'
    }
    return STAGE_CODE_ROLE[stage.stage_code] || ''
  }
  const getStageDesc = (stage: StageFlowItem) => {
    if (stage.is_custom) return '自定义阶段'
    return STAGE_CODE_DESC[stage.stage_code] || ''
  }
  const isStageRemovable = (stage: StageFlowItem) => {
    if (stage.is_custom) return true
    return STAGE_REMOVABLE[stage.stage_code] !== false
  }

  // ==================== 加载中 ====================
  if (pageLoading) return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '50vh' }}>
      <div style={{ textAlign: 'center' }}>
        <div style={{ width: '32px', height: '32px', border: `2px solid ${C.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite', margin: '0 auto 12px' }} />
        <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
        <div style={{ color: C.textMuted, fontSize: '14px' }}>加载配方数据...</div>
      </div>
    </div>
  )

  // ==================== 渲染 ====================
  return (
    <div style={{ display: 'flex', gap: '24px', minHeight: 'calc(100vh - 180px)' }}>
      {/* ======== 左侧：编辑区 ======== */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {/* 顶部操作栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
          <button onClick={() => fromPath === '/lesson-plans/recipes' ? navigate('/lesson-plans/recipes') : window.history.back()}
            style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '14px', color: C.textSec }}>
            {fromPath === '/lesson-plans/recipes' ? "← 返回配方列表" : "← 返回"}
          </button>
          <button onClick={handleSave} disabled={saving || !name.trim()} style={{
            padding: '9px 24px', borderRadius: '8px', border: 'none', fontSize: '14px', fontWeight: 600,
            cursor: saving || !name.trim() ? 'not-allowed' : 'pointer',
            background: saving || !name.trim() ? C.border : C.primary, color: saving || !name.trim() ? C.textMuted : '#fff',
          }}>{saving ? '保存中...' : isEdit ? '更新配方' : '创建配方'}</button>
        </div>

        {/* v88-fix：草稿恢复提示横幅 */}
        {draftRestored && (
          <div style={{
            marginBottom: '16px', padding: '12px 18px', borderRadius: '10px',
            background: 'linear-gradient(135deg, rgba(245,158,11,0.08), rgba(251,191,36,0.05))',
            border: '1px solid rgba(245,158,11,0.25)',
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            fontSize: '13px', color: '#92400E',
          }}>
            <span>📋 已恢复上次未保存的编辑内容</span>
            <div style={{ display: 'flex', gap: '8px' }}>
              <button onClick={handleDiscardDraft} style={{
                padding: '4px 12px', borderRadius: '6px', border: '1px solid rgba(245,158,11,0.3)',
                background: 'transparent', fontSize: '12px', color: '#B45309', cursor: 'pointer',
              }}>放弃草稿</button>
              <button onClick={() => setDraftRestored(false)} style={{
                padding: '4px 12px', borderRadius: '6px', border: 'none',
                background: '#F59E0B', fontSize: '12px', color: '#fff', cursor: 'pointer', fontWeight: 600,
              }}>继续编辑</button>
            </div>
          </div>
        )}

        {/* 基本信息卡片 */}
        <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '24px', marginBottom: '16px' }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '16px' }}>📦 基本信息</div>
          <div style={{ marginBottom: '16px' }}>
            <label style={labelStyle}>配方名称 <span style={{ color: C.danger }}>*</span></label>
            <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="例如：七年级AI课通用配方" style={inputStyle} />
          </div>
          <div style={{ marginBottom: '16px' }}>
            <label style={labelStyle}>描述</label>
            <input type="text" value={description} onChange={e => setDescription(e.target.value)} placeholder="简要描述配方适用场景" style={inputStyle} />
          </div>
          <div style={{ marginBottom: '16px' }}>
            <label style={labelStyle}>学科</label>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>{SUBJECTS.map(s => <button key={s} onClick={() => setSubject(s)} style={selBtn(subject === s)}>{s}</button>)}</div>
          </div>
          <div>
            <label style={labelStyle}>年级</label>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>{GRADES.map(g => <button key={g} onClick={() => setGradeRange(g)} style={selBtn(gradeRange === g)}>{g}</button>)}</div>
          </div>
        </div>

        {/* ======== 备课模式 ======== */}
        <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '24px', marginBottom: '16px' }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '12px' }}>🎛️ 备课模式</div>
          <div style={{ fontSize: '13px', color: C.textMuted, marginBottom: '14px' }}>选择AI的对话风格，影响每个阶段的交互轮数和详细程度</div>
          <div style={{ display: 'flex', gap: '12px' }}>
            {PROMPT_MODE_OPTIONS.map(item => (
              <div key={item.mode} onClick={() => handlePromptModeChange(item.mode)} style={{
                flex: 1, padding: '16px', borderRadius: '10px', cursor: 'pointer', transition: 'all 150ms ease',
                border: `2px solid ${promptMode === item.mode ? C.primary : C.border}`,
                background: promptMode === item.mode ? C.primaryLight : 'transparent',
              }}>
                <div style={{ fontSize: '15px', fontWeight: 600, color: promptMode === item.mode ? C.primary : C.text, marginBottom: '6px' }}>
                  {item.icon} {item.label}
                </div>
                <div style={{ fontSize: '12px', color: C.textMuted, lineHeight: 1.5 }}>{item.desc}</div>
              </div>
            ))}
          </div>
          {promptMode === 'per_stage' && (
            <div style={{
              marginTop: '12px', padding: '10px 14px', borderRadius: '8px',
              background: 'rgba(79,123,232,0.06)', border: '1px solid rgba(79,123,232,0.15)',
              fontSize: '12px', color: '#3B82F6', lineHeight: 1.6,
            }}>
              🎚️ 逐阶段模式已启用 — 在下方「备课流程」的每个阶段右侧选择该阶段使用的对话模式
            </div>
          )}
        </div>

        {/* ======== 流程配置（迭代5：支持自定义阶段）======== */}
        <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '24px', marginBottom: '16px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
            <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>
              🔧 备课流程 <span style={{ fontSize: '12px', fontWeight: 400, color: C.textMuted }}>（{enabledCount}个阶段启用）</span>
            </div>
            <button onClick={openCreateStageModal} style={{
              fontSize: '12px', color: C.primary, background: C.primaryLight, border: 'none',
              padding: '6px 14px', borderRadius: '6px', cursor: 'pointer', fontWeight: 600,
            }}>＋ 自定义阶段</button>
          </div>
          <div style={{ fontSize: '13px', color: C.textMuted, marginBottom: '14px' }}>配置备课的阶段流程，可启用/禁用阶段、调整顺序，或快速选择预设模板</div>

          {flowPresets.length > 0 && (
            <div style={{ display: 'flex', gap: '8px', marginBottom: '16px', flexWrap: 'wrap' }}>
              {flowPresets.map(preset => (
                <button key={preset.key} onClick={() => applyPreset(preset)} style={{
                  padding: '8px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent',
                  fontSize: '12px', color: C.textSec, cursor: 'pointer', transition: 'all 150ms ease', display: 'flex', alignItems: 'center', gap: '6px',
                }}
                onMouseEnter={e => { (e.currentTarget as HTMLElement).style.borderColor = C.primary; (e.currentTarget as HTMLElement).style.color = C.primary }}
                onMouseLeave={e => { (e.currentTarget as HTMLElement).style.borderColor = C.border; (e.currentTarget as HTMLElement).style.color = C.textSec }}>
                  <span>{preset.icon}</span>
                  <span style={{ fontWeight: 600 }}>{preset.name}</span>
                  <span style={{ color: C.textMuted }}>({preset.duration})</span>
                </button>
              ))}
            </div>
          )}

          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {stageFlow.map((stage, idx) => {
              const removable = isStageRemovable(stage)
              const isRevise = stage.stage_code === 'revise'
              const showPerStageSelector = promptMode === 'per_stage' && stage.enabled
              return (
                <div key={stage.stage_code} style={{
                  display: 'flex', alignItems: 'center', gap: '10px', padding: '12px 14px', borderRadius: '10px',
                  border: `1px solid ${stage.enabled ? (stage.is_custom ? 'rgba(79,123,232,0.3)' : C.border) : 'rgba(156,163,175,0.2)'}`,
                  background: stage.enabled ? (stage.is_custom ? 'rgba(79,123,232,0.03)' : '#FAFBFC') : 'rgba(156,163,175,0.04)',
                  opacity: stage.enabled ? 1 : 0.6, transition: 'all 150ms ease',
                }}>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                    <button onClick={() => moveStage(idx, -1)} disabled={idx === 0 || isRevise}
                      style={{ border: 'none', background: 'none', cursor: idx === 0 || isRevise ? 'default' : 'pointer', fontSize: '10px', color: idx === 0 || isRevise ? C.border : C.textMuted, padding: '0' }}>▲</button>
                    <button onClick={() => moveStage(idx, 1)} disabled={idx === stageFlow.length - 1 || isRevise}
                      style={{ border: 'none', background: 'none', cursor: idx === stageFlow.length - 1 || isRevise ? 'default' : 'pointer', fontSize: '10px', color: idx === stageFlow.length - 1 || isRevise ? C.border : C.textMuted, padding: '0' }}>▼</button>
                  </div>
                  <div onClick={() => toggleStage(stage.stage_code)} style={{
                    width: '36px', height: '20px', borderRadius: '10px', cursor: removable ? 'pointer' : 'not-allowed',
                    background: stage.enabled ? C.success : '#D1D5DB', position: 'relative', transition: 'background 200ms ease', flexShrink: 0,
                  }}>
                    <div style={{
                      width: '16px', height: '16px', borderRadius: '50%', background: '#fff', position: 'absolute', top: '2px',
                      left: stage.enabled ? '18px' : '2px', transition: 'left 200ms ease', boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
                    }} />
                  </div>
                  <span style={{ fontSize: '18px', flexShrink: 0 }}>{getStageEmoji(stage)}</span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                      <span style={{ fontSize: '14px', fontWeight: 600, color: stage.enabled ? C.text : C.textMuted }}>
                        {getStageName(stage)}
                      </span>
                      {!removable && (
                        <span style={{ fontSize: '10px', padding: '1px 6px', borderRadius: '4px', background: 'rgba(239,68,68,0.08)', color: C.danger }}>必须</span>
                      )}
                      {stage.is_custom && (
                        <span style={{ fontSize: '10px', padding: '1px 6px', borderRadius: '4px', background: C.primaryLight, color: C.primary }}>自定义</span>
                      )}
                    </div>
                    <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '2px' }}>
                      {getStageRole(stage)} · {getStageDesc(stage)}
                    </div>
                  </div>
                  {showPerStageSelector && (
                    <select
                      value={stage.prompt_mode || 'guided'}
                      onChange={e => updateStagePromptMode(stage.stage_code, e.target.value)}
                      style={{
                        padding: '4px 8px', borderRadius: '6px', border: `1px solid ${C.border}`,
                        fontSize: '12px', color: C.text, background: '#fff', cursor: 'pointer',
                        outline: 'none', flexShrink: 0, fontFamily: 'inherit',
                      }}
                    >
                      {STAGE_PROMPT_MODE_OPTIONS.map(opt => (
                        <option key={opt.value} value={opt.value}>{opt.label}</option>
                      ))}
                    </select>
                  )}
                  {stage.is_custom && (
                    <div style={{ display: 'flex', gap: '4px', flexShrink: 0 }}>
                      <button onClick={() => openEditStageModal(stage.stage_code)} title="编辑"
                        style={{ border: 'none', background: 'none', cursor: 'pointer', fontSize: '13px', color: C.primary, padding: '2px 4px' }}>✏️</button>
                      <button onClick={() => handleDeleteCustomStage(stage.stage_code)} title="删除"
                        style={{ border: 'none', background: 'none', cursor: 'pointer', fontSize: '13px', color: C.danger, padding: '2px 4px' }}>🗑️</button>
                    </div>
                  )}
                  <span style={{ fontSize: '12px', color: C.textMuted, flexShrink: 0 }}>
                    {stage.enabled ? `第${stageFlow.filter((s, j) => j <= idx && s.enabled).length}步` : '已禁用'}
                  </span>
                </div>
              )
            })}
          </div>

          {flowMessages.length > 0 && (
            <div style={{ marginTop: '14px', display: 'flex', flexDirection: 'column', gap: '6px' }}>
              {flowMessages.map((msg, i) => {
                const style = FLOW_MSG_COLORS[msg.level] || FLOW_MSG_COLORS.info
                return (
                  <div key={i} style={{
                    padding: '8px 12px', borderRadius: '8px', fontSize: '12px', lineHeight: 1.5,
                    background: style.bg, border: `1px solid ${style.border}`, color: style.text,
                  }}>
                    {style.icon} {msg.message}
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* 教案结构 */}
        <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '24px', marginBottom: '16px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
            <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>📋 教案结构 <span style={{ fontSize: '12px', fontWeight: 400, color: C.textMuted }}>（定义你想要的教案格式）</span></div>
            <div style={{ display: 'flex', gap: '8px' }}>
              {lessonStructure.length === 0 && <button onClick={loadDefaultStructure} style={{ fontSize: '12px', color: C.primary, background: C.primaryLight, border: 'none', padding: '6px 14px', borderRadius: '6px', cursor: 'pointer', fontWeight: 600 }}>📥 加载默认模板</button>}
              <button onClick={addBlock} style={{ fontSize: '12px', color: C.success, background: 'rgba(16,185,129,0.08)', border: 'none', padding: '6px 14px', borderRadius: '6px', cursor: 'pointer', fontWeight: 600 }}>＋ 添加板块</button>
            </div>
          </div>
          {totalDuration > 0 && (
            <div style={{ marginBottom: '16px', padding: '10px 14px', borderRadius: '8px', background: totalDuration > 45 ? 'rgba(239,68,68,0.06)' : 'rgba(16,185,129,0.06)', border: `1px solid ${totalDuration > 45 ? 'rgba(239,68,68,0.15)' : 'rgba(16,185,129,0.15)'}` }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                <span style={{ fontSize: '12px', fontWeight: 600, color: totalDuration > 45 ? C.danger : C.success }}>⏱ 教学过程合计 {totalDuration} 分钟</span>
                {totalDuration > 45 && <span style={{ fontSize: '11px', color: C.danger }}>超出标准课时！</span>}
              </div>
              <div style={{ display: 'flex', height: '8px', borderRadius: '4px', overflow: 'hidden', background: '#E5E7EB' }}>
                {lessonStructure.find(b => b.sub_sections)?.sub_sections?.map((sub, i) => {
                  const colors = ['#3B82F6', '#10B981', '#F59E0B', '#8B5CF6', '#EC4899', '#14B8A6']
                  return <div key={i} style={{ width: `${totalDuration > 0 ? (sub.duration / totalDuration) * 100 : 0}%`, background: colors[i % colors.length], transition: 'width 300ms ease' }} title={`${sub.name} ${sub.duration}分钟`} />
                })}
              </div>
            </div>
          )}
          {lessonStructure.length === 0 && (
            <div style={{ padding: '30px', textAlign: 'center', color: C.textMuted, fontSize: '13px', lineHeight: 1.7 }}>
              暂未定义教案结构。点击「加载默认模板」快速开始，或点击「添加板块」自定义。<br /><span style={{ fontSize: '12px' }}>不定义时，AI使用系统默认格式。</span>
            </div>
          )}
          {lessonStructure.map((block, bIdx) => (
            <div key={bIdx} style={{ border: `1px solid ${C.border}`, borderRadius: '10px', padding: '14px', marginBottom: '10px', background: '#FAFBFC' }}>
              <div style={{ display: 'flex', gap: '10px', alignItems: 'center', marginBottom: '8px' }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                  <button onClick={() => moveBlock(bIdx, -1)} disabled={bIdx === 0} style={{ border: 'none', background: 'none', cursor: bIdx === 0 ? 'default' : 'pointer', fontSize: '10px', color: bIdx === 0 ? C.border : C.textMuted, padding: '0' }}>▲</button>
                  <button onClick={() => moveBlock(bIdx, 1)} disabled={bIdx === lessonStructure.length - 1} style={{ border: 'none', background: 'none', cursor: bIdx === lessonStructure.length - 1 ? 'default' : 'pointer', fontSize: '10px', color: bIdx === lessonStructure.length - 1 ? C.border : C.textMuted, padding: '0' }}>▼</button>
                </div>
                <input value={block.name} onChange={e => updateBlock(bIdx, 'name', e.target.value)} placeholder="板块名称" style={{ ...smallInput, flex: 1, fontWeight: 600 }} />
                <label style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '12px', color: C.textSec, cursor: 'pointer', whiteSpace: 'nowrap' }}>
                  <input type="checkbox" checked={block.required} onChange={e => updateBlock(bIdx, 'required', e.target.checked)} /> 必填
                </label>
                <button onClick={() => removeBlock(bIdx)} style={{ border: 'none', background: 'none', cursor: 'pointer', fontSize: '14px', color: C.danger, padding: '4px' }}>✕</button>
              </div>
              <input value={block.requirement} onChange={e => updateBlock(bIdx, 'requirement', e.target.value)} placeholder="你对这个板块的要求（自然语言描述）" style={{ ...smallInput, width: '100%', marginBottom: '8px' }} />
              {block.sub_sections && block.sub_sections.length > 0 && (
                <div style={{ marginTop: '8px', paddingLeft: '12px', borderLeft: `3px solid ${C.primary}` }}>
                  <div style={{ fontSize: '12px', fontWeight: 600, color: C.primary, marginBottom: '8px' }}>教学过程环节安排</div>
                  {block.sub_sections.map((sub, sIdx) => (
                    <div key={sIdx} style={{ display: 'flex', gap: '6px', alignItems: 'center', marginBottom: '6px', flexWrap: 'wrap' }}>
                      <input value={sub.name} onChange={e => updateSubSection(bIdx, sIdx, 'name', e.target.value)} placeholder="环节名" style={{ ...smallInput, width: '80px' }} />
                      <div style={{ display: 'flex', alignItems: 'center', gap: '3px' }}>
                        <input type="number" value={sub.duration} onChange={e => updateSubSection(bIdx, sIdx, 'duration', parseInt(e.target.value) || 0)} style={{ ...smallInput, width: '50px', textAlign: 'center' }} min={0} />
                        <span style={{ fontSize: '11px', color: C.textMuted }}>分钟</span>
                      </div>
                      <input value={sub.goal} onChange={e => updateSubSection(bIdx, sIdx, 'goal', e.target.value)} placeholder="设计目标" style={{ ...smallInput, flex: 1, minWidth: '100px' }} />
                      <input value={sub.output_requirement} onChange={e => updateSubSection(bIdx, sIdx, 'output_requirement', e.target.value)} placeholder="输出要求" style={{ ...smallInput, flex: 1, minWidth: '100px' }} />
                      <button onClick={() => removeSubSection(bIdx, sIdx)} style={{ border: 'none', background: 'none', cursor: 'pointer', fontSize: '12px', color: C.danger, padding: '2px' }}>✕</button>
                    </div>
                  ))}
                  <button onClick={() => addSubSection(bIdx)} style={{ fontSize: '11px', color: C.primary, background: 'none', border: `1px dashed ${C.primary}`, padding: '4px 12px', borderRadius: '4px', cursor: 'pointer', marginTop: '4px' }}>+ 添加环节</button>
                </div>
              )}
              {(!block.sub_sections || block.sub_sections.length === 0) && block.name.includes('教学过程') && (
                <button onClick={() => addSubSection(bIdx)} style={{ fontSize: '11px', color: C.primary, background: 'none', border: `1px dashed ${C.primary}`, padding: '4px 12px', borderRadius: '4px', cursor: 'pointer' }}>+ 添加教学过程子环节</button>
              )}
            </div>
          ))}
        </div>

        {/* 组件选择器 */}
        <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '24px', marginBottom: '16px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
            <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>🧩 教学组件 <span style={{ fontSize: '12px', fontWeight: 400, color: C.textMuted }}>（已选 {selectedCompIds.size} 个）</span></div>
            <button onClick={loadRecommend} disabled={compLoading} style={{ fontSize: '12px', color: C.primary, background: 'none', border: 'none', cursor: 'pointer' }}>{compLoading ? '加载中...' : '🔄 刷新推荐'}</button>
          </div>
          {compGroups.length === 0 && !compLoading && <div style={{ padding: '20px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>暂无推荐组件，请确认学科和年级</div>}
          {compGroups.map(group => (
            <div key={group.library_type} style={{ marginBottom: '12px' }}>
              <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>{group.library_name}</div>
              {group.components.map(comp => {
                const checked = selectedCompIds.has(comp.id)
                return (
                  <label key={comp.id} style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '8px 12px', borderRadius: '8px', marginBottom: '4px', cursor: 'pointer', background: checked ? C.primaryLight : 'transparent', border: `1px solid ${checked ? C.primary : 'transparent'}`, transition: 'all 150ms ease' }}>
                    <input type="checkbox" checked={checked} onChange={() => toggleComp(comp.id)} style={{ accentColor: C.primary }} />
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: '13px', color: C.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{comp.display_label}</div>
                      <div style={{ fontSize: '11px', color: C.textMuted }}>质量分 {comp.quality_score.toFixed(1)}</div>
                    </div>
                  </label>
                )
              })}
            </div>
          ))}
        </div>

        {/* 教师知识区 */}
        <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '24px', marginBottom: '16px' }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '16px' }}>🧠 教师知识</div>
          {([
            { label: '学情档案', value: studentProfile, setter: setStudentProfile, placeholder: '例如：32人班级，5个编程积极分子...', rows: 3 },
            { label: '教学风格', value: teachingStyle, setter: setTeachingStyle, placeholder: '例如：喜欢让学生动手探索...', rows: 2 },
            { label: '学校要求', value: schoolRequirements, setter: setSchoolReqs, placeholder: '例如：必须包含AI伦理讨论...', rows: 2 },
            { label: '备课心得', value: customNotes, setter: setCustomNotes, placeholder: '例如：上次用决策树教学效果很好...', rows: 2 },
            { label: '自定义AI提示词', value: customPrompt, setter: setCustomPrompt, placeholder: '给AI的额外指令...', rows: 2 },
          ] as const).map(field => (
            <div key={field.label} style={{ marginBottom: '14px' }}>
              <label style={labelStyle}>{field.label}</label>
              <textarea value={field.value} onChange={e => field.setter(e.target.value)} placeholder={field.placeholder} rows={field.rows} style={textareaStyle} />
            </div>
          ))}
        </div>
      </div>

      {/* ======== 右侧：AI上下文预览 ======== */}
      <div style={{ width: '340px', flexShrink: 0, position: 'sticky', top: '0', alignSelf: 'flex-start' }}>
        <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '20px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
            <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>👁️ AI上下文预览</div>
            {isEdit && id && <button onClick={() => handlePreview(id)} disabled={previewLoading} style={{ fontSize: '12px', color: C.primary, background: 'none', border: 'none', cursor: 'pointer' }}>{previewLoading ? '加载中...' : '🔄 刷新'}</button>}
          </div>
          {!isEdit && <div style={{ padding: '24px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>请先创建配方后查看AI上下文预览</div>}
          {isEdit && !preview && !previewLoading && (
            <div style={{ padding: '24px', textAlign: 'center' }}>
              <button onClick={() => id && handlePreview(id)} style={{ padding: '9px 20px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>加载预览</button>
            </div>
          )}
          {preview && (
            <>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 12px', background: 'rgba(16,185,129,0.08)', borderRadius: '8px', marginBottom: '12px' }}>
                <span style={{ fontSize: '12px' }}>📊</span>
                <span style={{ fontSize: '12px', color: C.success, fontWeight: 600 }}>预估 {preview.token_estimate} tokens</span>
              </div>
              <div style={{ maxHeight: '500px', overflowY: 'auto', padding: '12px', background: C.bg, borderRadius: '8px', fontSize: '12px', color: C.textSec, lineHeight: 1.7, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{preview.context_text}</div>
            </>
          )}
        </div>
      </div>

      {/* 迭代5：自定义阶段弹窗 */}
      {stageModalMode && (
        <CustomStageModal
          mode={stageModalMode}
          initial={editingStageData as Parameters<typeof CustomStageModal>[0]['initial']}
          onConfirm={handleStageModalConfirm}
          onCancel={() => setStageModalMode(null)}
          saving={stageSaving}
        />
      )}

      {/* Toast */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)',
          padding: '12px 24px', borderRadius: '10px',
          background: toast.type === 'error' ? '#FEF2F2' : '#1F2937',
          color: toast.type === 'error' ? C.danger : '#fff',
          fontSize: '14px', fontWeight: 500, boxShadow: '0 8px 24px rgba(0,0,0,0.15)',
          zIndex: 9999, whiteSpace: 'nowrap',
          border: toast.type === 'error' ? '1px solid #FECACA' : 'none',
          animation: 'toast-in 200ms ease',
        }}>
          <style>{`@keyframes toast-in { from{opacity:0;transform:translateX(-50%) translateY(8px)} to{opacity:1;transform:translateX(-50%) translateY(0)} }`}</style>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}
