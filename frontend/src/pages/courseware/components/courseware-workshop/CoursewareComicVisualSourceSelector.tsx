/**
 * CoursewareComicVisualSourceSelector.tsx
 *
 * 画风来源严格二选一组件：
 *   - 使用两个大卡片，不使用下拉框；
 *   - 清楚展示当前唯一生效的画风来源；
 *   - 选择一个来源即排除另一个来源；
 *   - 只触发onChange，不保存草稿、不调用接口。
 */

import type {
  CSSProperties,
} from 'react'

import {
  COURSEWARE_COMIC_VISUAL_SOURCE_OPTIONS,
} from './coursewareComicVisualSourceOptions'

import type {
  CoursewareComicVisualStyleSource,
} from './coursewareComicVisualSourceOptions'

interface Props {
  value: CoursewareComicVisualStyleSource
  disabled: boolean
  onChange: (
    value: CoursewareComicVisualStyleSource,
  ) => void
}

const C = {
  primary: '#7C3AED',
  text: '#1F2937',
  textSecondary: '#64748B',
  border: '#E2E8F0',
  white: '#FFFFFF',
}

export default function CoursewareComicVisualSourceSelector({
  value,
  disabled,
  onChange,
}: Props) {
  return (
    <fieldset
      disabled={disabled}
      style={fieldsetStyle}
    >
      <legend style={legendStyle}>
        画风来源
      </legend>

      <div style={descriptionStyle}>
        两种来源严格二选一，不会混合使用。
      </div>

      <div style={optionGridStyle}>
        {COURSEWARE_COMIC_VISUAL_SOURCE_OPTIONS.map(
          option => {
            const active =
              option.value === value

            return (
              <button
                key={option.value}
                type="button"
                aria-pressed={active}
                disabled={disabled}
                onClick={() =>
                  onChange(option.value)
                }
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
                  boxShadow:
                    active
                      ? '0 0 0 2px rgba(124,58,237,0.12)'
                      : 'none',
                }}
              >
                <div style={optionHeaderStyle}>
                  <span
                    aria-hidden="true"
                    style={iconStyle}
                  >
                    {option.icon}
                  </span>

                  <span style={{
                    ...optionLabelStyle,
                    color:
                      active
                        ? C.primary
                        : C.text,
                  }}>
                    {option.label}
                  </span>

                  <span
                    aria-hidden="true"
                    style={{
                      ...radioStyle,
                      borderColor:
                        active
                          ? C.primary
                          : '#CBD5E1',
                      background:
                        active
                          ? C.primary
                          : C.white,
                    }}
                  >
                    {active ? '✓' : ''}
                  </span>
                </div>

                <div style={
                  optionDescriptionStyle
                }>
                  {option.description}
                </div>

                <div style={
                  optionDetailStyle
                }>
                  {option.detail}
                </div>
              </button>
            )
          },
        )}
      </div>
    </fieldset>
  )
}

const fieldsetStyle: CSSProperties = {
  minWidth: 0,
  margin: '0 0 14px',
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
  marginBottom: 8,
  color: '#94A3B8',
  fontSize: 8,
  lineHeight: 1.5,
}

const optionGridStyle: CSSProperties = {
  display: 'grid',
  gridTemplateColumns:
    'repeat(auto-fit,minmax(220px,1fr))',
  gap: 10,
}

const optionStyle: CSSProperties = {
  minWidth: 0,
  padding: 12,
  borderRadius: 11,
  border: '1px solid',
  textAlign: 'left',
  cursor: 'pointer',
}

const optionHeaderStyle: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 7,
}

const iconStyle: CSSProperties = {
  flexShrink: 0,
  fontSize: 21,
}

const optionLabelStyle: CSSProperties = {
  flex: 1,
  minWidth: 0,
  fontSize: 11,
  fontWeight: 900,
}

const radioStyle: CSSProperties = {
  flexShrink: 0,
  width: 18,
  height: 18,
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  border: '2px solid',
  borderRadius: 999,
  color: '#FFFFFF',
  fontSize: 10,
  fontWeight: 900,
}

const optionDescriptionStyle: CSSProperties = {
  marginTop: 7,
  color: C.textSecondary,
  fontSize: 9,
  fontWeight: 800,
  lineHeight: 1.5,
}

const optionDetailStyle: CSSProperties = {
  marginTop: 5,
  color: '#94A3B8',
  fontSize: 8,
  lineHeight: 1.55,
}
