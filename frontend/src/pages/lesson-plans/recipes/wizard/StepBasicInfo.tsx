/**
 * StepBasicInfo — 配方向导步骤1：基本信息
 *
 * v79 新增：分步向导式配方创建
 *
 * 内容：
 *   - 配方名称（必填）
 *   - 配方描述（可选）
 *   - 学科选择（按钮组）
 *   - 年级选择（按钮组）
 *
 * 设计目标：门槛最低，只需4个字段，30秒可完成
 */

import {
  C, SUBJECTS, GRADES,
  labelStyle, inputStyle, selBtn, stepCardStyle,
  type WizardFormData,
} from './wizardConstants'

/* ==================== Props 类型 ==================== */
interface StepBasicInfoProps {
  formData: WizardFormData
  updateForm: (updates: Partial<WizardFormData>) => void
}

/* ==================== 组件 ==================== */
export default function StepBasicInfo({ formData, updateForm }: StepBasicInfoProps) {
  return (
    <div style={stepCardStyle}>
      {/* 配方名称 */}
      <div style={{ marginBottom: '24px' }}>
        <label style={labelStyle}>
          配方名称 <span style={{ color: C.danger }}>*</span>
        </label>
        <input
          type="text"
          value={formData.name}
          onChange={e => updateForm({ name: e.target.value })}
          placeholder="例如：七年级AI课通用配方、九年级数学期末复习配方"
          style={inputStyle}
          autoFocus
        />
        <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '6px' }}>
          起一个便于识别的名字，方便你以后快速找到这个配方
        </div>
      </div>

      {/* 配方描述 */}
      <div style={{ marginBottom: '24px' }}>
        <label style={labelStyle}>描述</label>
        <input
          type="text"
          value={formData.description}
          onChange={e => updateForm({ description: e.target.value })}
          placeholder="简要描述这个配方的适用场景（可选）"
          style={inputStyle}
        />
      </div>

      {/* 学科选择 */}
      <div style={{ marginBottom: '24px' }}>
        <label style={labelStyle}>学科</label>
        <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '10px' }}>
          选择这个配方主要适用的学科，AI会据此推荐匹配的教学组件
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
          {SUBJECTS.map(s => (
            <button
              key={s}
              onClick={() => updateForm({ subject: s })}
              style={selBtn(formData.subject === s)}
            >
              {s}
            </button>
          ))}
        </div>
      </div>

      {/* 年级选择 */}
      <div>
        <label style={labelStyle}>年级</label>
        <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '10px' }}>
          选择这个配方主要适用的年级段
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
          {GRADES.map(g => (
            <button
              key={g}
              onClick={() => updateForm({ gradeRange: g })}
              style={selBtn(formData.gradeRange === g)}
            >
              {g}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
