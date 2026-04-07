/**
 * RecipeWizardPage — 配方创建向导主页面
 *
 * v79 新增：分步向导式配方创建
 *
 * 功能：
 *   - 6步进度条导航（已完成打勾 + 当前高亮 + 未来灰色）
 *   - 步骤内容区域（按当前步骤渲染对应子组件）
 *   - 底部固定操作栏（上一步 / 下一步 / 跳过 / 创建配方）
 *   - 所有步骤共享一个 formData 状态对象
 *   - 最后一步提交时组装payload调用 createRecipe API
 *
 * 路由：/lesson-plans/recipes/wizard
 */
import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { createRecipe } from '@/api/recipes'
import {
  C, WIZARD_STEPS, createEmptyFormData,
  type WizardFormData,
} from './wizard/wizardConstants'
import StepBasicInfo from './wizard/StepBasicInfo'
import StepTeacherKnowledge from './wizard/StepTeacherKnowledge'
import StepComponents from './wizard/StepComponents'
import StepLessonStructure from './wizard/StepLessonStructure'
import StepWorkflow from './wizard/StepWorkflow'
import StepPreview from './wizard/StepPreview'

/* ==================== 主组件 ==================== */
export default function RecipeWizardPage() {
  const navigate = useNavigate()

  // ---- 当前步骤（0-based） ----
  const [currentStep, setCurrentStep] = useState(0)

  // ---- 表单数据（所有步骤共享） ----
  const [formData, setFormData] = useState<WizardFormData>(createEmptyFormData)

  // ---- 页面状态 ----
  const [saving, setSaving] = useState(false)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  // ---- 更新表单数据的通用方法 ----
  const updateForm = useCallback((updates: Partial<WizardFormData>) => {
    setFormData(prev => ({ ...prev, ...updates }))
  }, [])

  // ---- 跳转到指定步骤（预览页"修改"按钮用） ----
  const goToStep = useCallback((step: number) => {
    if (step >= 0 && step < WIZARD_STEPS.length) {
      setCurrentStep(step)
    }
  }, [])

  // ---- 步骤校验 ----
  const validateStep = (step: number): string | null => {
    switch (step) {
      case 0: // 基本信息
        if (!formData.name.trim()) return '请填写配方名称'
        return null
      case 5: // 预览确认
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
      setCurrentStep(prev => prev + 1)
      // 滚回顶部
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
      setCurrentStep(prev => prev + 1)
      window.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }

  // ---- 创建配方（最后一步提交） ----
  const handleCreate = async () => {
    const error = validateStep(currentStep)
    if (error) { showToast(error, 'error'); return }

    setSaving(true)
    try {
      const payload = {
        name: formData.name.trim(),
        description: formData.description.trim(),
        subject: formData.subject,
        grade_range: formData.gradeRange,
        component_ids: Array.from(formData.selectedCompIds),
        student_profile: formData.studentProfile.trim(),
        teaching_style: formData.teachingStyle.trim(),
        school_requirements: formData.schoolRequirements.trim(),
        custom_notes: formData.customNotes.trim(),
        custom_prompt: formData.customPrompt.trim(),
        lesson_structure: formData.lessonStructure.length > 0
          ? JSON.stringify(formData.lessonStructure) : '[]',
        prompt_mode: formData.promptMode,
        stages_config: JSON.stringify(formData.stageFlow),
      }
      await createRecipe(payload)
      showToast('配方创建成功 ✓')
      // 创建成功后跳转到配方列表页
      setTimeout(() => {
        navigate('/lesson-plans/recipes', { replace: true })
      }, 800)
    } catch (e: unknown) {
      console.error('创建配方失败:', e)
      showToast(e instanceof Error ? e.message : '创建失败', 'error')
    } finally {
      setSaving(false)
    }
  }

  // ---- 渲染当前步骤内容 ----
  const renderStepContent = () => {
    switch (currentStep) {
      case 0:
        return <StepBasicInfo formData={formData} updateForm={updateForm} />
      case 1:
        return <StepTeacherKnowledge formData={formData} updateForm={updateForm} />
      case 2:
        return <StepComponents formData={formData} updateForm={updateForm} />
      case 3:
        return <StepLessonStructure formData={formData} updateForm={updateForm} />
      case 4:
        return <StepWorkflow formData={formData} updateForm={updateForm} />
      case 5:
        return <StepPreview formData={formData} onGoToStep={goToStep} />
      default:
        return null
    }
  }

  // ---- 是否最后一步 ----
  const isLastStep = currentStep === WIZARD_STEPS.length - 1
  const canSkip = WIZARD_STEPS[currentStep]?.optional && !isLastStep

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
          步骤 {currentStep + 1} / {WIZARD_STEPS.length}
        </div>
      </div>

      {/* ======== 进度条 ======== */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        marginBottom: '32px', padding: '0 20px',
      }}>
        {WIZARD_STEPS.map((step, idx) => {
          const isCompleted = idx < currentStep
          const isCurrent = idx === currentStep

          return (
            <div key={step.key} style={{ display: 'flex', alignItems: 'center' }}>
              {/* 圆点 */}
              <div
                onClick={() => { if (isCompleted) setCurrentStep(idx) }}
                style={{
                  width: '40px', height: '40px', borderRadius: '50%',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  fontSize: isCurrent ? '18px' : '14px',
                  fontWeight: 600,
                  cursor: isCompleted ? 'pointer' : 'default',
                  transition: 'all 200ms ease',
                  background: isCompleted ? C.success
                    : isCurrent ? C.primary
                    : '#E5E7EB',
                  color: (isCompleted || isCurrent) ? '#fff' : C.textMuted,
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

        {/* 右侧：跳过 + 下一步/创建 */}
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
            onClick={isLastStep ? handleCreate : handleNext}
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
            {saving ? '创建中...' : isLastStep ? '✓ 创建配方' : '下一步 →'}
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
