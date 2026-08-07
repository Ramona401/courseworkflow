/**
 * CoursewareComicStyleInstructionField.tsx
 *
 * “使用本漫画所选画风”模式的补充要求输入框：
 *   - 只细化当前选中的漫画预设画风；
 *   - 跟随课件模式不渲染本组件；
 *   - 字符上限仍由前后端共同校验为1200。
 */

import type {
  CSSProperties,
} from 'react'

interface Props {
  value: string
  disabled: boolean
  length: number
  onChange: (value: string) => void
}

const C = {
  text: '#1F2937',
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  white: '#FFFFFF',
}

export default function CoursewareComicStyleInstructionField({
  value,
  disabled,
  length,
  onChange,
}: Props) {
  const tooLong =
    length > 1200

  return (
    <label style={fieldStyle}>
      <span style={labelStyle}>
        补充所选漫画画风要求

        <span style={optionalStyle}>
          可选
        </span>
      </span>

      <textarea
        value={value}
        onChange={event =>
          onChange(event.target.value)
        }
        disabled={disabled}
        rows={4}
        maxLength={1400}
        placeholder="例如：淡雅青绿色；人物保持中学生特征；实验器材结构准确；避免过度卡通化。"
        style={{
          ...textareaStyle,
          borderColor:
            tooLong
              ? '#DC2626'
              : C.border,
        }}
      />

      <div style={{
        ...lengthStyle,
        color:
          tooLong
            ? '#DC2626'
            : C.textMuted,
      }}>
        {length}/1200
      </div>
    </label>
  )
}

const fieldStyle: CSSProperties = {
  display: 'block',
  marginTop: 2,
}

const labelStyle: CSSProperties = {
  display: 'block',
  marginBottom: 5,
  color: C.text,
  fontSize: 10,
  fontWeight: 900,
}

const optionalStyle: CSSProperties = {
  marginLeft: 5,
  color: C.textMuted,
  fontSize: 8,
  fontWeight: 600,
}

const textareaStyle: CSSProperties = {
  width: '100%',
  boxSizing: 'border-box',
  padding: '9px 10px',
  borderRadius: 8,
  border: '1px solid',
  background: C.white,
  color: C.text,
  fontSize: 10,
  lineHeight: 1.6,
  resize: 'vertical',
  outline: 'none',
}

const lengthStyle: CSSProperties = {
  marginTop: 4,
  textAlign: 'right',
  fontSize: 8,
}
