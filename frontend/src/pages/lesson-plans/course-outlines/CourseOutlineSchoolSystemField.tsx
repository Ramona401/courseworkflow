/**
 * CourseOutlineSchoolSystemField.tsx
 *
 * K12课程大纲学制选择字段：
 *   - 只允许普通学制standard和五四制five_four；
 *   - 使用course-outlines API模块中的正式枚举和常量；
 *   - 不持有业务状态，不执行网络请求；
 *   - 由课程大纲管理页决定是否展示。
 */

import type {
  CSSProperties,
} from 'react'

import {
  type CourseOutlineSchoolSystem,
  COURSE_OUTLINE_SCHOOL_SYSTEM_STANDARD,
  COURSE_OUTLINE_SCHOOL_SYSTEM_FIVE_FOUR,
} from '@/api/course-outlines'

interface CourseOutlineSchoolSystemFieldProps {
  value: CourseOutlineSchoolSystem

  onChange: (
    value: CourseOutlineSchoolSystem,
  ) => void

  inputStyle:
    CSSProperties
}

export function CourseOutlineSchoolSystemField({
  value,
  onChange,
  inputStyle,
}: CourseOutlineSchoolSystemFieldProps) {
  return (
    <div style={{
      marginBottom: 14,
    }}>
      <div style={{
        fontSize: 13,
        fontWeight: 600,
        color: '#6B7280',
        marginBottom: 6,
      }}>
        学制
      </div>

      <select
        value={value}
        onChange={(event) =>
          onChange(
            event.target.value as
              CourseOutlineSchoolSystem,
          )
        }
        style={inputStyle}
      >
        <option
          value={
            COURSE_OUTLINE_SCHOOL_SYSTEM_STANDARD
          }
        >
          普通学制
        </option>

        <option
          value={
            COURSE_OUTLINE_SCHOOL_SYSTEM_FIVE_FOUR
          }
        >
          五四制
        </option>
      </select>
    </div>
  )
}
