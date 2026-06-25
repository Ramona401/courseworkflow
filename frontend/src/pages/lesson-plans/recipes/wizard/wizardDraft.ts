/**
 * wizardDraft.ts — 配方向导草稿（sessionStorage）辅助模块
 *
 * 【批次2A】从 RecipeWizardPage.tsx 原样抽离（零逻辑变更，仅搬迁还 600 行债）：
 *   - RecipeDraft 草稿数据结构（含批次1.1 的进度字段 currentStep/visitedSteps）
 *   - getDraftKey / saveDraft / loadDraft / clearDraft 四个 sessionStorage 读写函数
 *   - draftToForm 草稿 → WizardFormData 片段
 *   - draftToCurrentStep / draftToVisited 草稿进度解析（含越界钳制 + 强制含当前步）
 *
 * 设计口径照搬编辑器 v88-fix：防抖 500ms 暂存、24 小时过期、旧学科兼容、旧草稿进度字段兜底。
 * 主页面（RecipeWizardPage）只负责 effect 编排与 state，存取细节与兼容逻辑收口在本模块。
 */
import {
  WIZARD_STEPS, DEFAULT_FLOW,
  type WizardFormData,
} from './wizardConstants'
import type { LessonStructureBlock, StageFlowItem } from '@/api/recipes'

/* ==================== 草稿数据结构 ==================== */

/**
 * 草稿数据结构（与编辑器 RecipeDraft 对齐 + 批次1.1 进度字段）
 *   - 无 teachingStyle/customNotes/customPrompt/promptMode（这些已归属 AI 助手层 / 恒 guided）
 *   - currentStep / visitedSteps：刷新后还原「走到第几步、访问过哪些步」，圆点点跳不丢
 */
export interface RecipeDraft {
  name: string; description: string; subject: string; gradeRange: string
  studentProfile: string; schoolRequirements: string
  selectedCompIds: string[]            // Set 序列化为数组存储
  lessonStructure: LessonStructureBlock[]
  stageFlow: StageFlowItem[]
  currentStep: number                  // 批次1.1：保存当前步
  visitedSteps: number[]               // 批次1.1：保存已访问步集合（Set 序列化为数组）
  savedAt: number                      // 保存时间戳（毫秒），用于与后端 updated_at 比较新旧
}

/* ==================== sessionStorage 读写 ==================== */

/** 构建草稿在 sessionStorage 中的 key：编辑用配方 id，新建用固定 key */
export const getDraftKey = (recipeId?: string) => recipeId ? `recipe_draft_${recipeId}` : 'recipe_draft_new'

/** 保存草稿到 sessionStorage（满或不可用时静默忽略） */
export const saveDraft = (key: string, draft: RecipeDraft) => {
  try { sessionStorage.setItem(key, JSON.stringify(draft)) } catch { /* 忽略 */ }
}

/** 从 sessionStorage 读取草稿（含旧学科兼容 + 旧草稿进度字段兜底 + 24 小时过期清理） */
export const loadDraft = (key: string): RecipeDraft | null => {
  try {
    const raw = sessionStorage.getItem(key)
    if (!raw) return null
    const draft = JSON.parse(raw) as RecipeDraft
    // 兼容：旧草稿的 subject='AI' 自动转为 '人工智能'
    if (draft.subject === 'AI') draft.subject = '人工智能'
    // 兼容：旧草稿无进度字段 → 兜底为第 1 步、仅第 1 步已访问
    if (typeof draft.currentStep !== 'number') draft.currentStep = 0
    if (!Array.isArray(draft.visitedSteps)) draft.visitedSteps = [0]
    // 超过 24 小时的草稿自动过期
    if (Date.now() - draft.savedAt > 24 * 60 * 60 * 1000) {
      sessionStorage.removeItem(key)
      return null
    }
    return draft
  } catch { return null }
}

/** 清除草稿 */
export const clearDraft = (key: string) => {
  try { sessionStorage.removeItem(key) } catch { /* 忽略 */ }
}

/* ==================== 草稿 → 表单 / 进度 解析 ==================== */

/** 把草稿内容灌进 formData（新建恢复 / 编辑时草稿更新 共用） */
export const draftToForm = (draft: RecipeDraft): Partial<WizardFormData> => ({
  name: draft.name,
  description: draft.description,
  subject: draft.subject,
  gradeRange: draft.gradeRange,
  studentProfile: draft.studentProfile,
  schoolRequirements: draft.schoolRequirements,
  selectedCompIds: new Set(draft.selectedCompIds),
  lessonStructure: draft.lessonStructure.length > 0 ? draft.lessonStructure : [],
  stageFlow: draft.stageFlow.length > 0 ? draft.stageFlow : DEFAULT_FLOW.map(s => ({ ...s })),
})

/** 从草稿解析「当前步」（越界钳制在 [0, 步数-1]） */
export const draftToCurrentStep = (draft: RecipeDraft): number => {
  const max = WIZARD_STEPS.length - 1
  if (draft.currentStep < 0) return 0
  if (draft.currentStep > max) return max
  return draft.currentStep
}

/** 从草稿解析「已访问步集合」（过滤越界值 + 强制含当前步，保证当前步永远可见可判定） */
export const draftToVisited = (draft: RecipeDraft, currentStep: number): Set<number> => {
  const max = WIZARD_STEPS.length - 1
  const set = new Set<number>()
  for (const s of draft.visitedSteps) {
    if (typeof s === 'number' && s >= 0 && s <= max) set.add(s)
  }
  set.add(currentStep)   // 当前步必然已访问
  if (set.size === 0) set.add(0)
  return set
}
