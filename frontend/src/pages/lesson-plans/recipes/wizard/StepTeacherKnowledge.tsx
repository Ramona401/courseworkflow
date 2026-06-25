/**
 * StepTeacherKnowledge — 配方向导步骤2：教师知识
 *
 * v79 新增：分步向导式配方创建
 * 动作2（批次A）：教学风格/备课心得/自定义提示词三字段已移除（归属 AI 助手层），
 *   仅保留学情档案与学校要求两项配方级上下文；随之删除已失效的"高级选项折叠"。
 *
 * 内容：
 *   - 学情档案（学生特点和班级情况）
 *   - 学校要求（学校特殊规定）
 *
 * 设计目标：
 *   - 每个字段有清晰的引导文案和示例
 *   - 整体可跳过（所有字段都是可选的）
 */
import {
  C, labelStyle, textareaStyle, stepCardStyle,
  type WizardFormData,
} from './wizardConstants'

/* ==================== Props 类型 ==================== */
interface StepTeacherKnowledgeProps {
  formData: WizardFormData
  updateForm: (updates: Partial<WizardFormData>) => void
}

/* ==================== 知识字段配置 ==================== */
interface KnowledgeField {
  key: keyof WizardFormData
  label: string
  icon: string
  placeholder: string
  hint: string
  rows: number
}

const KNOWLEDGE_FIELDS: KnowledgeField[] = [
  {
    key: 'studentProfile',
    label: '学情档案',
    icon: '👥',
    placeholder: '例如：32人班级，5个编程积极分子，大部分学生首次接触AI概念，有3个学生需要特别关注...',
    hint: '描述你班级学生的整体情况，AI会据此调整教学难度和活动设计',
    rows: 3,
  },
  {
    key: 'schoolRequirements',
    label: '学校要求',
    icon: '🏫',
    placeholder: '例如：必须包含AI伦理讨论环节，每节课要有小组合作活动...',
    hint: '学校或教研组的特殊规定，AI会确保教案满足这些要求',
    rows: 2,
  },
]

/* ==================== 组件 ==================== */
export default function StepTeacherKnowledge({ formData, updateForm }: StepTeacherKnowledgeProps) {
  // 计算已填写字段数
  const filledCount = KNOWLEDGE_FIELDS.filter(f => {
    const val = formData[f.key]
    return typeof val === 'string' && val.trim().length > 0
  }).length

  return (
    <div style={stepCardStyle}>
      {/* 顶部提示 */}
      <div style={{
        padding: '12px 16px', borderRadius: '8px', marginBottom: '24px',
        background: 'rgba(79,123,232,0.06)', border: '1px solid rgba(79,123,232,0.12)',
      }}>
        <div style={{ fontSize: '13px', color: C.primary, lineHeight: 1.6 }}>
          💡 这些信息会帮助AI更好地理解你的教学场景。
          <strong>全部都是可选的</strong>，你可以随时在配方编辑页补充。
          {filledCount > 0 && (
            <span style={{ marginLeft: '8px', fontWeight: 600 }}>
              （已填写 {filledCount}/{KNOWLEDGE_FIELDS.length}）
            </span>
          )}
        </div>
      </div>

      {/* 知识字段 */}
      {KNOWLEDGE_FIELDS.map(field => (
        <div key={field.key} style={{ marginBottom: '20px' }}>
          <label style={labelStyle}>
            <span style={{ marginRight: '6px' }}>{field.icon}</span>
            {field.label}
          </label>
          <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '8px' }}>
            {field.hint}
          </div>
          <textarea
            value={formData[field.key] as string}
            onChange={e => updateForm({ [field.key]: e.target.value })}
            placeholder={field.placeholder}
            rows={field.rows}
            style={textareaStyle}
          />
        </div>
      ))}
    </div>
  )
}
