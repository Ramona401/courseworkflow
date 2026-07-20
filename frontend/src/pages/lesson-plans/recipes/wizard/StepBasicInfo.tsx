/**
 * StepBasicInfo — 配方向导步骤1：基本信息
 *
 * 当前职责：
 *   - 配方名称和描述；
 *   - 从当前登录用户的教育域课程目录中选择具体课程；
 *   - 从当前教育域学习层级中选择具体层级；
 *   - 课程与层级必须精确匹配，供后续资源匹配使用。
 *
 * 教育域规则：
 *   - K12显示学科和一年级至高三；
 *   - 职业教育显示课程和中职层级；
 *   - 成人教育显示培训类别和学习基础；
 *   - 职业教育与成人教育目录为空时不回退K12课程。
 */

import {
  useEffect,
  useMemo,
} from 'react'
import { useSubjects } from '@/hooks/useSubjects'
import {
  useEducationProfile,
} from '@/hooks/useEducationProfile'
import {
  getEducationLevelOptions,
} from '@/education-domain/options'
import {
  C,
  labelStyle,
  inputStyle,
  selBtn,
  stepCardStyle,
  type WizardFormData,
} from './wizardConstants'

/* ==================== Props 类型 ==================== */

interface StepBasicInfoProps {
  formData: WizardFormData
  updateForm: (
    updates: Partial<WizardFormData>,
  ) => void
}

/* ==================== 主组件 ==================== */

export default function StepBasicInfo({
  formData,
  updateForm,
}: StepBasicInfoProps) {
  const {
    subjects,
    loading: subjectsLoading,
    empty: subjectsEmpty,
  } = useSubjects()

  const {
    domain,
    profile,
  } = useEducationProfile()

  const levelOptions = useMemo(
    () => getEducationLevelOptions(domain),
    [domain],
  )

  const levelValues = useMemo(
    () => levelOptions.map(
      item => item.value,
    ),
    [levelOptions],
  )

  /**
   * createEmptyFormData仍保留历史K12默认值，
   * 目录异步到达后在这里校正为当前教育域第一个合法课程。
   *
   * 目录合法为空时只在旧值非空的情况下清空一次，
   * 避免反复写入同一个空字符串造成重复渲染。
   */
  useEffect(() => {
    if (subjectsLoading) return

    if (subjects.length === 0) {
      if (formData.subject) {
        updateForm({
          subject: '',
        })
      }

      return
    }

    if (
      !subjects.includes(
        formData.subject,
      )
    ) {
      updateForm({
        subject: subjects[0],
      })
    }
  }, [
    subjectsLoading,
    subjects,
    formData.subject,
    updateForm,
  ])

  /**
   * 当前教育域变化或恢复了旧草稿时，
   * 将不合法的K12年级修正为当前域首个学习层级。
   */
  useEffect(() => {
    if (levelValues.length === 0) {
      if (formData.gradeRange) {
        updateForm({
          gradeRange: '',
        })
      }

      return
    }

    if (
      !levelValues.includes(
        formData.gradeRange,
      )
    ) {
      updateForm({
        gradeRange:
          levelValues[0],
      })
    }
  }, [
    levelValues,
    formData.gradeRange,
    updateForm,
  ])

  return (
    <div style={stepCardStyle}>
      {/* 配方名称 */}
      <div style={{
        marginBottom: '24px',
      }}>
        <label style={labelStyle}>
          配方名称
          {' '}
          <span style={{
            color: C.danger,
          }}>
            *
          </span>
        </label>

        <input
          type="text"
          value={formData.name}
          onChange={event =>
            updateForm({
              name: event.target.value,
            })
          }
          placeholder={
            domain === 'vocational'
              ? '例如：数控车削实训教学配方'
              : domain === 'adult'
                ? '例如：新员工沟通培训配方'
                : '例如：七年级AI课通用配方、九年级数学期末复习配方'
          }
          style={inputStyle}
          autoFocus
        />

        <div style={{
          fontSize: '12px',
          color: C.textMuted,
          marginTop: '6px',
        }}>
          起一个便于识别的名字，
          方便以后快速找到这个配方
        </div>
      </div>

      {/* 配方描述 */}
      <div style={{
        marginBottom: '24px',
      }}>
        <label style={labelStyle}>
          描述
        </label>

        <input
          type="text"
          value={formData.description}
          onChange={event =>
            updateForm({
              description:
                event.target.value,
            })
          }
          placeholder="简要描述这个配方的适用场景（可选）"
          style={inputStyle}
        />
      </div>

      {/* 课程选择 */}
      <div style={{
        marginBottom: '24px',
      }}>
        <label style={labelStyle}>
          {profile.subject_label}
          {' '}
          <span style={{
            color: C.danger,
          }}>
            *
          </span>
        </label>

        <div style={{
          fontSize: '12px',
          color: C.textMuted,
          marginBottom: '10px',
        }}>
          选择一个具体
          {profile.subject_label}。
          配方只会在课程完全一致的备课中出现。
        </div>

        {subjectsLoading && (
          <div style={{
            color: C.textMuted,
            fontSize: '12px',
            marginBottom: '10px',
          }}>
            正在加载当前组织课程...
          </div>
        )}

        {subjectsEmpty &&
         !subjectsLoading && (
          <div style={{
            padding: '10px 12px',
            borderRadius: '8px',
            background: '#FEF2F2',
            color: C.danger,
            fontSize: '12px',
            lineHeight: 1.6,
          }}>
            当前组织尚未配置可用
            {profile.subject_label}，
            请联系管理员。
          </div>
        )}

        {!subjectsEmpty && (
          <div style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: '8px',
          }}>
            {subjects.map(subject => (
              <button
                key={subject}
                type="button"
                onClick={() =>
                  updateForm({
                    subject,
                  })
                }
                style={selBtn(
                  formData.subject ===
                    subject,
                )}
              >
                {subject}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* 学习层级选择 */}
      <div>
        <label style={labelStyle}>
          {profile.grade_label}
          {' '}
          <span style={{
            color: C.danger,
          }}>
            *
          </span>
        </label>

        <div style={{
          fontSize: '12px',
          color: C.textMuted,
          marginBottom: '10px',
        }}>
          选择当前教育域中的一个具体
          {profile.grade_label}。
          跨层级范围不参与自动匹配。
        </div>

        <div style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: '8px',
        }}>
          {levelOptions.map(option => (
            <button
              key={option.value}
              type="button"
              onClick={() =>
                updateForm({
                  gradeRange:
                    option.value,
                })
              }
              style={selBtn(
                formData.gradeRange ===
                  option.value,
              )}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
