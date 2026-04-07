/**
 * StepTeacherKnowledge — 配方向导步骤2：教师知识
 *
 * v79 新增：分步向导式配方创建
 *
 * 内容：
 *   - 学情档案（学生特点和班级情况）
 *   - 教学风格（个人教学偏好）
 *   - 学校要求（学校特殊规定）
 *   - 备课心得（经验积累）
 *   - 自定义AI提示词（高级用户）
 *
 * 设计目标：
 *   - 每个字段有清晰的引导文案和示例
 *   - 自定义AI提示词默认收起，避免吓到新手
 *   - 整体可跳过（所有字段都是可选的）
 */
import { useState } from 'react'
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
  advanced?: boolean // 是否为高级选项（默认收起）
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
    key: 'teachingStyle',
    label: '教学风格',
    icon: '🎨',
    placeholder: '例如：喜欢让学生动手探索，先做后讲；课堂气氛偏活跃，鼓励提问...',
    hint: '告诉AI你的教学偏好，生成的教案会匹配你的风格',
    rows: 2,
  },
  {
    key: 'schoolRequirements',
    label: '学校要求',
    icon: '🏫',
    placeholder: '例如：必须包含AI伦理讨论环节，每节课要有小组合作活动...',
    hint: '学校或教研组的特殊规定，AI会确保教案满足这些要求',
    rows: 2,
  },
  {
    key: 'customNotes',
    label: '备课心得',
    icon: '💡',
    placeholder: '例如：上次用决策树教学效果很好，学生反馈分层作业太难了...',
    hint: '记录你的教学经验和反思，AI会参考这些避免重复踩坑',
    rows: 2,
  },
  {
    key: 'customPrompt',
    label: '自定义AI提示词',
    icon: '⚙️',
    placeholder: '给AI的额外指令，例如：请在每个环节加入与生活相关的案例...',
    hint: '高级选项 — 直接给AI写指令，会附加到系统提示词中',
    rows: 2,
    advanced: true,
  },
]

/* ==================== 组件 ==================== */
export default function StepTeacherKnowledge({ formData, updateForm }: StepTeacherKnowledgeProps) {
  // 是否展开高级选项
  const [showAdvanced, setShowAdvanced] = useState(false)

  // 基础字段（非高级）
  const basicFields = KNOWLEDGE_FIELDS.filter(f => !f.advanced)
  // 高级字段
  const advancedFields = KNOWLEDGE_FIELDS.filter(f => f.advanced)

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

      {/* 基础字段 */}
      {basicFields.map(field => (
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

      {/* 高级选项折叠区 */}
      {advancedFields.length > 0 && (
        <div style={{ borderTop: `1px solid ${C.border}`, paddingTop: '16px', marginTop: '8px' }}>
          <button
            onClick={() => setShowAdvanced(!showAdvanced)}
            style={{
              display: 'flex', alignItems: 'center', gap: '6px',
              background: 'none', border: 'none', cursor: 'pointer',
              fontSize: '13px', fontWeight: 500, color: C.textSec,
              padding: '4px 0', marginBottom: showAdvanced ? '16px' : '0',
            }}
          >
            <span style={{
              display: 'inline-block', transition: 'transform 200ms ease',
              transform: showAdvanced ? 'rotate(90deg)' : 'rotate(0deg)',
            }}>
              ▶
            </span>
            高级选项
          </button>

          {showAdvanced && advancedFields.map(field => (
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
      )}
    </div>
  )
}
