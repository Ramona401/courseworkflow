/**
 * RecipeWizardPage — 配方搭建向导主页面（新建 + 编辑 双模式）
 *
 * v79 新增：分步向导式配方创建
 * 动作2（批次A）：提交 payload 移除 teaching_style/custom_notes/custom_prompt 三字段。
 * 动作2（批次A.1）：删除「教学组件」步骤，步骤由 6 步缩为 5 步。
 *
 * 【配方搭建一页化 · 批次1】向导从「只新建」升级为「新建 + 编辑」唯一入口：
 *   1. 编辑模式：useParams 读 :id（路由形态 C：/recipes/wizard/:id），有 id 即编辑态——
 *      调 getRecipe 回填各步，保存走 updateRecipe；无 id 为新建态，保存走 createRecipe。
 *      回填字段口径与原 RecipeEditorPage 逐字对齐（component_ids/lesson_structure/
 *      stages_config 三个 JSON 字符串照样 JSON.parse + try/catch；prompt_mode 不读，恒 guided）。
 *   2. 草稿自动保存：sessionStorage 防抖 500ms 暂存 + 恢复横幅；dataLoadedRef 防初始态误触发。
 *   3. 步骤自由点跳：进度条圆点「已访问过的步可点」；编辑态初始化即全步已访问。
 *
 * 【批次1.1】草稿同时保存「进度」（currentStep + visitedSteps），刷新后不再回到第 1 步。
 *
 * 【批次2A】草稿辅助（RecipeDraft 接口 + 七个存取/解析函数）抽离到 wizard/wizardDraft.ts，
 *   本文件改为 import，纯搬迁还 600 行债（628 → ~535），零逻辑变更。
 *
 * 【批次1 与后续批次的边界（重要）】
 *   - 路由 C 的 /recipes/wizard/:id 这条 Route 要到批次4 改 App.tsx 时才挂上。
 *     故本批新建成功后的默认行为是「toast + 回配方列表页」（不依赖 /wizard/:id）；
 *     「新建后立刻转编辑态」的跳转留到批次4 路由就位后再接。
 *   - 自定义阶段 / AI token 预览：批次2B 搬入对应步骤（StepWorkflow / StepPreview）。
 *   - 教案结构轻化 / 文案澄清：批次3 处理。
 *
 * 路由：/lesson-plans/recipes/wizard（新建） | /lesson-plans/recipes/wizard/:id（编辑，批次4 挂载）
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  getRecipe, createRecipe, updateRecipe,
} from '@/api/recipes'
import {
  C, WIZARD_STEPS, createEmptyFormData, DEFAULT_FLOW,
  type WizardFormData,
} from './wizard/wizardConstants'
import {
  getDraftKey, saveDraft, loadDraft, clearDraft,
  draftToForm, draftToCurrentStep, draftToVisited,
  type RecipeDraft,
} from './wizard/wizardDraft'
import StepBasicInfo from './wizard/StepBasicInfo'
import StepTeacherKnowledge from './wizard/StepTeacherKnowledge'
import StepLessonStructure from './wizard/StepLessonStructure'
import StepWorkflow from './wizard/StepWorkflow'
import StepPreview from './wizard/StepPreview'

/* ==================== 主组件 ==================== */
export default function RecipeWizardPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isEdit = !!id

  // 草稿 key：编辑态绑配方 id，新建态固定
  const draftKey = getDraftKey(id)

  // ---- 当前步骤（0-based） ----
  const [currentStep, setCurrentStep] = useState(0)

  // ---- 已访问过的步骤集合（步骤自由点跳的依据）----
  //   新建态初始仅含第 0 步；前进时把目标步加入。
  //   编辑态在加载完成后一次性标记全部步骤为已访问（编辑可任意跳）。
  const [visitedSteps, setVisitedSteps] = useState<Set<number>>(new Set([0]))

  // ---- 表单数据（所有步骤共享） ----
  const [formData, setFormData] = useState<WizardFormData>(createEmptyFormData)

  // ---- 页面状态 ----
  const [pageLoading, setPageLoading] = useState(isEdit)   // 编辑态需先拉数据
  const [saving, setSaving] = useState(false)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  // 数据是否加载完成（防初始态触发草稿保存 / 防编辑态拉取前误存空表单）
  const dataLoadedRef = useRef(false)
  // 是否恢复了草稿（用于显示提示横幅）
  const [draftRestored, setDraftRestored] = useState(false)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  // ---- 更新表单数据的通用方法 ----
  const updateForm = useCallback((updates: Partial<WizardFormData>) => {
    setFormData(prev => ({ ...prev, ...updates }))
  }, [])

  // ---- 标记某步为已访问 ----
  const markVisited = useCallback((step: number) => {
    setVisitedSteps(prev => {
      if (prev.has(step)) return prev
      const next = new Set(prev); next.add(step); return next
    })
  }, [])

  // ==================== 加载：编辑态拉后端 / 新建态恢复草稿 ====================
  useEffect(() => {
    if (!id) {
      // 新建模式：尝试恢复草稿（含进度还原）
      const draft = loadDraft(draftKey)
      if (draft) {
        setFormData(prev => ({ ...prev, ...draftToForm(draft) }))
        const step = draftToCurrentStep(draft)
        setCurrentStep(step)
        setVisitedSteps(draftToVisited(draft, step))
        setDraftRestored(true)
      }
      dataLoadedRef.current = true
      return
    }
    // 编辑模式：拉配方详情（草稿更新则优先草稿）
    const load = async () => {
      try {
        const detail = await getRecipe(id)
        const draft = loadDraft(draftKey)
        const detailUpdatedAt = new Date(detail.updated_at).getTime()
        const hasFresherDraft = draft && draft.savedAt > detailUpdatedAt

        if (hasFresherDraft && draft) {
          // 草稿比后端新 → 恢复草稿（含进度）
          setFormData(prev => ({ ...prev, ...draftToForm(draft) }))
          setCurrentStep(draftToCurrentStep(draft))
          setDraftRestored(true)
        } else {
          // 从后端回填（口径与原编辑器逐字对齐）
          const next: WizardFormData = {
            name: detail.name,
            description: detail.description || '',
            subject: detail.subject,
            gradeRange: detail.grade_range,
            studentProfile: detail.student_profile || '',
            schoolRequirements: detail.school_requirements || '',
            selectedCompIds: new Set<string>(),
            lessonStructure: [],
            promptMode: 'guided',                  // 配方一律 guided，不读 detail.prompt_mode
            stageFlow: DEFAULT_FLOW.map(s => ({ ...s })),
          }
          // component_ids：JSON 字符串 → Set
          if (detail.component_ids) {
            try {
              const ids = JSON.parse(detail.component_ids)
              if (Array.isArray(ids)) next.selectedCompIds = new Set(ids)
            } catch { /* 忽略 */ }
          }
          // lesson_structure：JSON 字符串 → 数组
          if (detail.lesson_structure) {
            try {
              const p = JSON.parse(detail.lesson_structure)
              if (Array.isArray(p) && p.length > 0) next.lessonStructure = p
            } catch { /* 忽略 */ }
          }
          // stages_config：JSON 字符串 → 数组（校验首项含 enabled 字段，避免脏数据）
          if (detail.stages_config) {
            try {
              const p = JSON.parse(detail.stages_config)
              if (Array.isArray(p) && p.length > 0 && p[0].enabled !== undefined) next.stageFlow = p
            } catch { /* 忽略 */ }
          }
          setFormData(next)
        }
        // 编辑态：全部步骤标记为已访问（编辑可任意点跳）
        //   注意：visitedSteps 编辑态强制全开，不被草稿里偏小的集合覆盖；
        //   currentStep 上面已按「草稿优先」处理，编辑刷新回到正看的步。
        setVisitedSteps(new Set(WIZARD_STEPS.map((_, i) => i)))
      } catch (e) {
        console.error('加载配方失败:', e)
        showToast('加载配方失败', 'error')
      } finally {
        setPageLoading(false)
        dataLoadedRef.current = true
      }
    }
    load()
  }, [id, draftKey])

  // ==================== 草稿自动保存（防抖 500ms）====================
  //   批次1.1：依赖含 currentStep / visitedSteps，切步即重存（带最新进度）。
  useEffect(() => {
    if (!dataLoadedRef.current) return   // 数据未加载完成前不保存
    const timer = setTimeout(() => {
      const draft: RecipeDraft = {
        name: formData.name,
        description: formData.description,
        subject: formData.subject,
        gradeRange: formData.gradeRange,
        studentProfile: formData.studentProfile,
        schoolRequirements: formData.schoolRequirements,
        selectedCompIds: Array.from(formData.selectedCompIds),
        lessonStructure: formData.lessonStructure,
        stageFlow: formData.stageFlow,
        currentStep,                              // 批次1.1：存当前步
        visitedSteps: Array.from(visitedSteps),   // 批次1.1：存已访问集
        savedAt: Date.now(),
      }
      saveDraft(draftKey, draft)
    }, 500)
    return () => clearTimeout(timer)
  }, [formData, currentStep, visitedSteps, draftKey])

  // ---- 跳转到指定步骤（预览页"修改"按钮 / 圆点点跳 共用） ----
  const goToStep = useCallback((step: number) => {
    if (step >= 0 && step < WIZARD_STEPS.length) {
      markVisited(step)
      setCurrentStep(step)
    }
  }, [markVisited])

  // ---- 步骤校验 ----
  const validateStep = (step: number): string | null => {
    switch (step) {
      case 0: // 基本信息
        if (!formData.name.trim()) return '请填写配方名称'
        return null
      case 4: // 预览确认
        if (!formData.name.trim()) return '配方名称不能为空'
        return null
      default:
        return null // 其他步骤可跳过
    }
  }

  // ---- 下一步 ----
  const handleNext = () => {
    const error = validateStep(currentStep)
    if (error) { showToast(error, 'error'); return }
    if (currentStep < WIZARD_STEPS.length - 1) {
      const target = currentStep + 1
      markVisited(target)
      setCurrentStep(target)
      window.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }

  // ---- 上一步 ----
  const handleBack = () => {
    if (currentStep > 0) {
      setCurrentStep(prev => prev - 1)
      window.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }

  // ---- 跳过当前步骤 ----
  const handleSkip = () => {
    if (WIZARD_STEPS[currentStep]?.optional && currentStep < WIZARD_STEPS.length - 1) {
      const target = currentStep + 1
      markVisited(target)
      setCurrentStep(target)
      window.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }

  // ---- 组装提交 payload（新建/更新共用，字段口径与编辑器对齐） ----
  const buildPayload = () => ({
    name: formData.name.trim(),
    description: formData.description.trim(),
    subject: formData.subject,
    grade_range: formData.gradeRange,
    component_ids: Array.from(formData.selectedCompIds),
    student_profile: formData.studentProfile.trim(),
    school_requirements: formData.schoolRequirements.trim(),
    lesson_structure: formData.lessonStructure.length > 0
      ? JSON.stringify(formData.lessonStructure) : '[]',
    prompt_mode: formData.promptMode,
    // 剥离 stageFlow 里的 per-stage prompt_mode，存量逐阶段配方归一到 guided
    stages_config: JSON.stringify(
      formData.stageFlow.map(s => { const c = { ...s }; delete c.prompt_mode; return c })
    ),
  })

  // ---- 保存（最后一步提交：新建 createRecipe / 编辑 updateRecipe） ----
  const handleSave = async () => {
    const error = validateStep(currentStep)
    if (error) { showToast(error, 'error'); return }

    setSaving(true)
    try {
      const payload = buildPayload()
      if (isEdit && id) {
        await updateRecipe(id, payload)
        clearDraft(draftKey)
        setDraftRestored(false)
        showToast('配方已更新 ✓')
        setTimeout(() => navigate('/lesson-plans/recipes', { replace: true }), 800)
      } else {
        await createRecipe(payload)
        clearDraft(draftKey)
        setDraftRestored(false)
        showToast('配方创建成功 ✓')
        // 批次1：新建成功默认回列表页（不依赖批次4 才挂载的 /wizard/:id 编辑态路由）
        setTimeout(() => navigate('/lesson-plans/recipes', { replace: true }), 800)
      }
    } catch (e: unknown) {
      console.error('保存配方失败:', e)
      showToast(e instanceof Error ? e.message : (isEdit ? '更新失败' : '创建失败'), 'error')
    } finally {
      setSaving(false)
    }
  }

  // ---- 放弃草稿，重新从后端加载（编辑态）/ 清空（新建态） ----
  const handleDiscardDraft = () => {
    clearDraft(draftKey)
    setDraftRestored(false)
    window.location.reload()
  }

  // ---- 渲染当前步骤内容 ----
  const renderStepContent = () => {
    switch (currentStep) {
      case 0:
        return <StepBasicInfo formData={formData} updateForm={updateForm} />
      case 1:
        return <StepTeacherKnowledge formData={formData} updateForm={updateForm} />
      case 2:
        return <StepLessonStructure formData={formData} updateForm={updateForm} />
      case 3:
        return <StepWorkflow formData={formData} updateForm={updateForm} recipeId={id} />
      case 4:
        return <StepPreview formData={formData} onGoToStep={goToStep} recipeId={id} />
      default:
        return null
    }
  }

  // ---- 是否最后一步 ----
  const isLastStep = currentStep === WIZARD_STEPS.length - 1
  const canSkip = WIZARD_STEPS[currentStep]?.optional && !isLastStep

  // ==================== 编辑态加载中 ====================
  if (pageLoading) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '50vh' }}>
        <div style={{ textAlign: 'center' }}>
          <div style={{
            width: '32px', height: '32px', border: `2px solid ${C.primary}`,
            borderTopColor: 'transparent', borderRadius: '50%',
            animation: 'spin 0.8s linear infinite', margin: '0 auto 12px',
          }} />
          <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
          <div style={{ color: C.textMuted, fontSize: '14px' }}>加载配方数据...</div>
        </div>
      </div>
    )
  }

  return (
    <div style={{ minHeight: 'calc(100vh - 180px)', display: 'flex', flexDirection: 'column' }}>
      {/* ======== 顶部：返回按钮 + 步骤指示 ======== */}
      <div style={{
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        marginBottom: '24px',
      }}>
        <button
          onClick={() => navigate('/lesson-plans/recipes')}
          style={{
            background: 'none', border: 'none', cursor: 'pointer',
            fontSize: '14px', color: C.textSec,
          }}
        >
          ← 返回配方列表
        </button>
        <div style={{ fontSize: '13px', color: C.textMuted }}>
          {isEdit ? '编辑配方' : '新建配方'} · 步骤 {currentStep + 1} / {WIZARD_STEPS.length}
        </div>
      </div>

      {/* ======== 草稿恢复提示横幅 ======== */}
      {draftRestored && (
        <div style={{
          marginBottom: '16px', padding: '12px 18px', borderRadius: '10px',
          background: 'linear-gradient(135deg, rgba(245,158,11,0.08), rgba(251,191,36,0.05))',
          border: '1px solid rgba(245,158,11,0.25)',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          fontSize: '13px', color: '#92400E',
          maxWidth: '720px', width: '100%', margin: '0 auto 16px',
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

      {/* ======== 进度条（步骤自由点跳：已访问过的步可点） ======== */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        marginBottom: '32px', padding: '0 20px',
      }}>
        {WIZARD_STEPS.map((step, idx) => {
          const isCompleted = idx < currentStep
          const isCurrent = idx === currentStep
          const isClickable = visitedSteps.has(idx) && idx !== currentStep

          return (
            <div key={step.key} style={{ display: 'flex', alignItems: 'center' }}>
              {/* 圆点 */}
              <div
                onClick={() => { if (isClickable) goToStep(idx) }}
                style={{
                  width: '40px', height: '40px', borderRadius: '50%',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  fontSize: isCurrent ? '18px' : '14px',
                  fontWeight: 600,
                  cursor: isClickable ? 'pointer' : 'default',
                  transition: 'all 200ms ease',
                  background: isCompleted ? C.success
                    : isCurrent ? C.primary
                    : visitedSteps.has(idx) ? '#C7D2FE'   // 已访问但在当前步之后：可点的浅蓝
                    : '#E5E7EB',
                  color: (isCompleted || isCurrent) ? '#fff'
                    : visitedSteps.has(idx) ? C.primary
                    : C.textMuted,
                  boxShadow: isCurrent ? `0 0 0 4px ${C.primaryLight}` : 'none',
                }}
                title={step.title}
              >
                {isCompleted ? '✓' : step.icon}
              </div>

              {/* 连接线 */}
              {idx < WIZARD_STEPS.length - 1 && (
                <div style={{
                  width: '48px', height: '3px', borderRadius: '2px',
                  background: isCompleted ? C.success : '#E5E7EB',
                  transition: 'background 200ms ease',
                  margin: '0 4px',
                }} />
              )}
            </div>
          )
        })}
      </div>

      {/* ======== 步骤标题 ======== */}
      <div style={{ textAlign: 'center', marginBottom: '24px' }}>
        <div style={{ fontSize: '20px', fontWeight: 700, color: C.text, marginBottom: '4px' }}>
          {WIZARD_STEPS[currentStep].icon} {WIZARD_STEPS[currentStep].title}
        </div>
        <div style={{ fontSize: '14px', color: C.textMuted }}>
          {WIZARD_STEPS[currentStep].desc}
        </div>
      </div>

      {/* ======== 步骤内容区域 ======== */}
      <div style={{ flex: 1, maxWidth: '720px', width: '100%', margin: '0 auto' }}>
        {renderStepContent()}
      </div>

      {/* ======== 底部操作栏 ======== */}
      <div style={{
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        padding: '20px 0', marginTop: '32px',
        borderTop: `1px solid ${C.border}`,
        maxWidth: '720px', width: '100%', margin: '32px auto 0',
      }}>
        {/* 左侧：上一步 */}
        <div>
          {currentStep > 0 ? (
            <button
              onClick={handleBack}
              style={{
                padding: '10px 24px', borderRadius: '8px',
                border: `1px solid ${C.border}`, background: 'transparent',
                fontSize: '14px', color: C.textSec, cursor: 'pointer',
                transition: 'all 150ms ease',
              }}
            >
              ← 上一步
            </button>
          ) : (
            <div />
          )}
        </div>

        {/* 右侧：跳过 + 下一步/保存 */}
        <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
          {canSkip && (
            <button
              onClick={handleSkip}
              style={{
                padding: '10px 20px', borderRadius: '8px',
                border: 'none', background: 'transparent',
                fontSize: '14px', color: C.textMuted, cursor: 'pointer',
              }}
            >
              跳过此步 →
            </button>
          )}

          <button
            onClick={isLastStep ? handleSave : handleNext}
            disabled={saving}
            style={{
              padding: '10px 28px', borderRadius: '8px',
              border: 'none', fontSize: '14px', fontWeight: 600,
              cursor: saving ? 'not-allowed' : 'pointer',
              background: saving ? C.border : (isLastStep ? C.success : C.primary),
              color: saving ? C.textMuted : '#fff',
              transition: 'all 150ms ease',
              boxShadow: saving ? 'none' : '0 2px 8px rgba(0,0,0,0.12)',
            }}
          >
            {saving
              ? (isEdit ? '更新中...' : '创建中...')
              : isLastStep
                ? (isEdit ? '✓ 更新配方' : '✓ 创建配方')
                : '下一步 →'}
          </button>
        </div>
      </div>

      {/* ======== Toast ======== */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)',
          padding: '12px 24px', borderRadius: '10px',
          background: toast.type === 'error' ? '#FEF2F2' : '#1F2937',
          color: toast.type === 'error' ? C.danger : '#fff',
          fontSize: '14px', fontWeight: 500,
          boxShadow: '0 8px 24px rgba(0,0,0,0.15)',
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
