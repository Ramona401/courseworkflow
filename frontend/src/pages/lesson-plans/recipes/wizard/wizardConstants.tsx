import { DEFAULT_SUBJECTS } from '@/constants/subjects'
/**
 * wizardConstants.tsx — 配方向导共享常量、样式、类型定义
 *
 * v79 新增：分步向导式配方创建
 * 所有步骤组件共享此文件的颜色、样式、类型
 *
 * 动作2（批次A）：WizardFormData 移除 teachingStyle/customNotes/customPrompt
 *   三字段（教学风格/备课心得/自定义提示词归属 AI 助手层）；
 *   createEmptyFormData 默认学科由 'AI' 修正为 '人工智能'（与 SUBJECTS 对齐）。
 * 动作2（批次A.1）：WIZARD_STEPS 删除「教学组件」步骤项——向导下线该步骤，
 *   组件改由备课时 skill_router 后端按学科年级自动匹配（手动指定组件保留在编辑器高级区）；
 *   向导步骤由 6 步缩为 5 步（基本信息/教师知识/教案结构/备课流程/预览确认）。
 *   selectedCompIds 字段保留（向导中恒为空集 → 备课时后端自动匹配；亦供提交 payload 与编辑器使用）；
 *   promptMode 字段保留并恒为 'guided'——向导不再提供「备课模式」选择器
 *   （引导/高效 仅在专家配方里曾用、对话模式恒 guided，故下线该旋钮，后端逻辑不变）。
 *
 * v202 学科统一：SUBJECTS 三端（备课工坊/配方向导/课件工坊）统一为同一份 17 学科清单。
 *   本次变更——'信息技术'→'信息科技'对齐课标库；新增 科学/道德与法治/音乐/美术/体育/劳动
 *   （'道德与法治'用全称对齐后端 SubjectCodeMap）；'政治'保留（高中段用，义务段用'道德与法治'）。
 *   createEmptyFormData 默认学科仍为 '人工智能'（仍在新清单中，无需调整）。
 */

import type {
  LessonStructureBlock, PromptMode, StageFlowItem,
} from '@/api/recipes'

/* ==================== 颜色常量（与现有系统保持一致）==================== */
export const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981',
  danger: '#EF4444',
  accent: '#F59E0B',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  card: '#FFFFFF',
  border: '#F3F4F6',
  borderHover: '#E5E7EB',
  bg: '#FAFBFC',
}

/* ==================== 学科和年级选项 ==================== */
// v202 学科统一：三端统一为同一份 17 学科；'信息技术'→'信息科技'对齐课标库；
//   新增 科学/道德与法治/音乐/美术/体育/劳动（'道德与法治'用全称对齐后端编码表）。
export const SUBJECTS = [...DEFAULT_SUBJECTS]  // 单一真相源：见 @/constants/subjects（方案甲，v231）

/**
 * 配方只绑定一个具体年级。
 *
 * 严格匹配模式下不再创建“小学低段”“高中”“1-6”等学段或范围配方，
 * 避免生成永远无法在具体年级备课中自动命中的资源。
 */
export const GRADES = [
  '一年级',
  '二年级',
  '三年级',
  '四年级',
  '五年级',
  '六年级',
  '七年级',
  '八年级',
  '九年级',
  '高一',
  '高二',
  '高三',
]

/* ==================== 默认教案结构模板 ==================== */
export const DEFAULT_LESSON_STRUCTURE: LessonStructureBlock[] = [
  { name: '教学目标', required: true, requirement: '知识与技能、过程与方法、情感态度价值观三维目标', order: 1 },
  { name: '教学重难点', required: true, requirement: '重点2-3个，难点1-2个', order: 2 },
  { name: '课前准备', required: false, requirement: '教师准备和学生准备分开列', order: 3 },
  {
    name: '教学过程', required: true,
    requirement: '按以下环节输出，每环节含教师活动和学生活动', order: 4,
    sub_sections: [
      { name: '导入', duration: 5, goal: '激发兴趣，建立新旧知识连接', output_requirement: '用生活场景切入' },
      { name: '新授', duration: 20, goal: '理解概念，掌握操作', output_requirement: '讲+演+练交替，写教师话术' },
      { name: '练习', duration: 15, goal: '分层巩固', output_requirement: '基础+进阶任务' },
      { name: '小结', duration: 5, goal: '归纳要点', output_requirement: '学生总结，布置思考题' },
    ],
  },
  { name: '作业设计', required: false, requirement: '分层作业：基础题+提高题', order: 5 },
  { name: '板书设计', required: false, requirement: '思维导图式板书', order: 6 },
]

/* ==================== 默认5阶段全开流程 ==================== */
export const DEFAULT_FLOW: StageFlowItem[] = [
  { stage_code: 'analyze', enabled: true, order: 1 },
  { stage_code: 'design', enabled: true, order: 2 },
  { stage_code: 'write', enabled: true, order: 3 },
  { stage_code: 'review', enabled: true, order: 4 },
  { stage_code: 'revise', enabled: true, order: 5 },
]

/* ==================== 向导步骤定义 ==================== */
export interface WizardStepDef {
  key: string       // 步骤标识
  title: string     // 步骤标题
  icon: string      // 步骤图标
  desc: string      // 简短说明
  optional: boolean // 是否可跳过
}

export const WIZARD_STEPS: WizardStepDef[] = [
  { key: 'basic', title: '基本信息', icon: '📦', desc: '配方名称、学科和年级', optional: false },
  { key: 'knowledge', title: '教师知识', icon: '🧠', desc: '学情和学校要求', optional: true },
  { key: 'structure', title: '教案结构', icon: '📋', desc: '定义教案的格式要求', optional: true },
  { key: 'workflow', title: '备课流程', icon: '🔧', desc: '配置备课的阶段流程', optional: true },
  { key: 'preview', title: '预览确认', icon: '👁️', desc: '确认配置并创建配方', optional: false },
]

/* ==================== 向导共享状态类型 ==================== */
export interface WizardFormData {
  // 步骤1：基本信息
  name: string
  description: string
  subject: string
  gradeRange: string

  // 步骤2：教师知识
  studentProfile: string
  schoolRequirements: string

  // 组件选择（批次A.1 起向导已无「教学组件」步骤，此字段恒为空集 →
  //   备课时由后端 skill_router 自动匹配；字段保留以兼容提交 payload 与编辑器手动指定）
  selectedCompIds: Set<string>

  // 步骤3：教案结构
  lessonStructure: LessonStructureBlock[]

  // 步骤4：备课流程（promptMode 恒为 'guided'，向导已无备课模式选择器；字段保留照常提交）
  promptMode: PromptMode
  stageFlow: StageFlowItem[]
}

/** 创建空白表单数据 */
export function createEmptyFormData(): WizardFormData {
  return {
    name: '',
    description: '',
    subject: '人工智能',
    gradeRange: '七年级',
    studentProfile: '',
    schoolRequirements: '',
    selectedCompIds: new Set(),
    lessonStructure: [],
    promptMode: 'guided',
    stageFlow: DEFAULT_FLOW.map(s => ({ ...s })),
  }
}

/* ==================== 共享样式 ==================== */

/** 标签样式 */
export const labelStyle: React.CSSProperties = {
  display: 'block', fontSize: '14px', fontWeight: 600,
  color: C.text, marginBottom: '8px',
}

/** 文本域样式 */
export const textareaStyle: React.CSSProperties = {
  width: '100%', padding: '10px 14px', borderRadius: '8px',
  border: `1px solid ${C.border}`, fontSize: '14px', color: C.text,
  outline: 'none', boxSizing: 'border-box', fontFamily: 'inherit',
  lineHeight: 1.6, resize: 'vertical', transition: 'border-color 150ms ease',
}

/** 输入框样式 */
export const inputStyle: React.CSSProperties = {
  ...textareaStyle, resize: 'none' as const,
}

/** 小输入框样式 */
export const smallInput: React.CSSProperties = {
  padding: '6px 10px', borderRadius: '6px',
  border: `1px solid ${C.border}`, fontSize: '13px', color: C.text,
  outline: 'none', boxSizing: 'border-box' as const, fontFamily: 'inherit',
}

/** 选择按钮样式（学科/年级等） */
export const selBtn = (active: boolean): React.CSSProperties => ({
  padding: '6px 14px', borderRadius: '20px',
  border: `1px solid ${active ? C.primary : C.border}`,
  background: active ? C.primaryLight : 'transparent',
  color: active ? C.primary : C.textSec,
  fontSize: '13px', fontWeight: active ? 600 : 400,
  cursor: 'pointer', transition: 'all 150ms ease',
})

/** 步骤容器卡片样式 */
export const stepCardStyle: React.CSSProperties = {
  background: C.card, borderRadius: '12px',
  border: `1px solid ${C.border}`, padding: '28px 32px',
}

/** 步骤标题样式 */
export const stepTitleStyle: React.CSSProperties = {
  fontSize: '18px', fontWeight: 700, color: C.text, marginBottom: '6px',
}

/** 步骤描述样式 */
export const stepDescStyle: React.CSSProperties = {
  fontSize: '14px', color: C.textMuted, marginBottom: '24px', lineHeight: 1.6,
}
