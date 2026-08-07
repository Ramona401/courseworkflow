/**
 * CoursewareComicStyleOptionSection.tsx
 *
 * 第三步通用卡片选项区：
 *   - 美术风格、画幅和清晰度共用；
 *   - 只显示并回传稳定枚举值；
 *   - 不保存状态、不调用网络接口。
 */

import type {
  CSSProperties,
} from 'react'

import type {
  CoursewareComicWorkflowOption,
} from './coursewareComicWorkflow'

interface Props<Value extends string> {
  title: string
  description: string
  options:
    ReadonlyArray<
      CoursewareComicWorkflowOption<Value>
    >
  selected: Value
  disabled: boolean
  onSelect: (value: Value) => void
}

const C = {
  primary: '#7C3AED',
  text: '#1F2937',
  textSecondary: '#64748B',
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  white: '#FFFFFF',
}

export default function CoursewareComicStyleOptionSection<
  Value extends string,
>({
  title,
  description,
  options,
  selected,
  disabled,
  onSelect,
}: Props<Value>) {
  return (
    <fieldset
      disabled={disabled}
      style={fieldsetStyle}
    >
      <legend style={legendStyle}>
        {title}
      </legend>

      <div style={descriptionStyle}>
        {description}
      </div>

      <div style={optionGridStyle}>
        {options.map(option => {
          const active =
            option.value === selected

          return (
            <button
              key={option.value}
              type="button"
              onClick={() =>
                onSelect(option.value)
              }
              disabled={disabled}
              style={{
                ...optionStyle,
                borderColor:
                  active
                    ? C.primary
                    : C.border,
                background:
                  active
                    ? 'rgba(124,58,237,0.07)'
                    : C.white,
              }}
            >
              <div style={{
                ...optionLabelStyle,
                color:
                  active
                    ? C.primary
                    : C.text,
              }}>
                {option.label}
              </div>

              <div style={
                optionDescriptionStyle
              }>
                {option.description}
              </div>
            </button>
          )
        })}
      </div>
    </fieldset>
  )
}

const fieldsetStyle: CSSProperties = {
  minWidth: 0,
  margin: '0 0 12px',
  padding: 0,
  border: 0,
}

const legendStyle: CSSProperties = {
  padding: 0,
  color: C.text,
  fontSize: 10,
  fontWeight: 900,
}

const descriptionStyle: CSSProperties = {
  marginTop: 3,
  marginBottom: 7,
  color: C.textMuted,
  fontSize: 8,
  lineHeight: 1.5,
}

const optionGridStyle: CSSProperties = {
  display: 'grid',
  gridTemplateColumns:
    'repeat(auto-fit,minmax(165px,1fr))',
  gap: 8,
}

const optionStyle: CSSProperties = {
  padding: '9px 10px',
  borderRadius: 9,
  border: '1px solid',
  textAlign: 'left',
  cursor: 'pointer',
}

const optionLabelStyle: CSSProperties = {
  fontSize: 10,
  fontWeight: 900,
}

const optionDescriptionStyle: CSSProperties = {
  marginTop: 3,
  color: C.textSecondary,
  fontSize: 8,
  lineHeight: 1.45,
}
