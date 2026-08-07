/**
 * ConversationStartScreen — 教育域适配的对话模式起步页
 *
 * K12：
 *   学科、年级、课程大纲版本、单元方案、班级学情、配方。
 *
 * 职业教育：
 *   课程、中职层级、教学主题或工作任务。
 *
 * 成人教育：
 *   培训类别、学习基础、培训主题。
 *
 * 在资源表完成education_domain硬隔离前：
 *   vocational/adult不读取K12配方、单元方案、班级学情和课程大纲。
 */

import {
  useEffect,
  useMemo,
  useState,
} from 'react'
import type {
  KeyboardEvent as ReactKeyboardEvent,
} from 'react'
import type {
  RecipeSelectionMode,
} from '@/api/lesson-plans'
import {
  getMountableUnitPlans,
} from '@/api/unit-plans'
import type {
  UnitPlanListItem,
} from '@/api/unit-plans'
import {
  getClassProfiles,
} from '@/api/class-profiles'
import type {
  ClassProfileListItem,
} from '@/api/class-profiles'
import ExactCourseOutlineSelector from '../components/ExactCourseOutlineSelector'
import {
  getAvailableRecipes,
} from '@/api/recipes'
import type {
  RecipeListItem,
} from '@/api/recipes'
import { useSubjects } from '@/hooks/useSubjects'
import {
  useEducationProfile,
} from '@/hooks/useEducationProfile'
import {
  getEducationLevelOptions,
  getTopicPlaceholder,
} from '@/education-domain/options'
import { C } from '../components/workshopConstants'
import RecipeModeSelector from '../components/RecipeModeSelector'

const DURATION_OPTIONS = [40, 45, 50, 60]

interface ConversationStartScreenProps {
  subject: string
  setSubject: (value: string) => void

  grade: string
  setGrade: (value: string) => void

  topic: string
  setTopic: (value: string) => void

  onTopicDraftKeyDown?: (
    event: ReactKeyboardEvent<HTMLInputElement>,
  ) => boolean

  duration: number
  setDuration: (value: number) => void

  unitPlanId: string
  setUnitPlanId: (value: string) => void

  classProfileId: string
  setClassProfileId: (value: string) => void

  courseOutlineId: string | null
  setCourseOutlineId: (
    value: string | null,
  ) => void

  /** 精确候选是否正在加载。加载时禁止开始请求。 */
  courseOutlineLoading: boolean
  onCourseOutlineLoadingChange: (
    loading: boolean,
  ) => void

  /**
   * 检测到旧publisher-only草稿时的迁移提示。
   *
   * 旧出版社字符串不能自动转成唯一大纲ID，
   * 必须由老师重新选择或明确保持不关联。
   */
  legacyCourseOutlineNotice?: string

  recipeMode: RecipeSelectionMode
  setRecipeMode: (
    value: RecipeSelectionMode,
  ) => void

  recipeId: string
  setRecipeId: (value: string) => void

  startLoading: boolean
  onStart: () => void
  onImport: () => void
  onSwitchMode?: () => void
}

export default function ConversationStartScreen({
  subject,
  setSubject,
  grade,
  setGrade,
  topic,
  setTopic,
  onTopicDraftKeyDown,
  duration,
  setDuration,
  unitPlanId,
  setUnitPlanId,
  classProfileId,
  setClassProfileId,
  courseOutlineId,
  setCourseOutlineId,
  courseOutlineLoading,
  onCourseOutlineLoadingChange,
  legacyCourseOutlineNotice,

  recipeMode,
  setRecipeMode,
  recipeId,
  setRecipeId,
  startLoading,
  onStart,
  onImport,
  onSwitchMode,
}: ConversationStartScreenProps) {
  const {
    subjects,
    loading: subjectsLoading,
    empty: subjectsEmpty,
  } = useSubjects()

  const {
    domain,
    profile,
    isK12,
  } = useEducationProfile()

  const levelOptions = useMemo(
    () => getEducationLevelOptions(domain),
    [domain],
  )

  const levelValues = useMemo(
    () => levelOptions.map(item => item.value),
    [levelOptions],
  )

  const subjectValid =
    subjects.includes(subject)

  const gradeValid =
    levelValues.includes(grade)

  useEffect(() => {
    if (subjectsLoading) return

    if (subjects.length === 0) {
      if (subject) setSubject('')
      return
    }

    if (!subjects.includes(subject)) {
      setSubject(subjects[0])
    }
  }, [
    subjectsLoading,
    subjects,
    subject,
    setSubject,
  ])

  useEffect(() => {
    if (levelValues.length === 0) return

    if (!levelValues.includes(grade)) {
      setGrade(levelValues[0])
    }
  }, [
    levelValues,
    grade,
    setGrade,
  ])

  /**
   * 非K12或当前教育画像不支持教材版本时，
   * 精确课程大纲必须保持明确不关联。
   *
   * 这里只清理前端临时选择；后端仍以教育域硬闸为最终防线。
   */
  useEffect(() => {
    if (
      isK12 &&
      profile.publisher_enabled
    ) {
      return
    }

    if (courseOutlineId !== null) {
      setCourseOutlineId(null)
    }

    if (courseOutlineLoading) {
      onCourseOutlineLoadingChange(false)
    }
  }, [
    isK12,
    profile.publisher_enabled,
    courseOutlineId,
    courseOutlineLoading,
    setCourseOutlineId,
    onCourseOutlineLoadingChange,
  ])

  const [recipes, setRecipes] =
    useState<RecipeListItem[]>([])

  const [recipesLoading, setRecipesLoading] =
    useState(false)

  useEffect(() => {
    let cancelled = false

    if (
      !isK12 ||
      !subjectValid ||
      !gradeValid
    ) {
      setRecipes([])
      setRecipeId('')

      if (!isK12 && recipeMode !== 'none') {
        setRecipeMode('none')
      }

      return
    }

    setRecipesLoading(true)

    getAvailableRecipes(subject, grade)
      .then(response => {
        if (cancelled) return

        const list = response.recipes || []
        setRecipes(list)

        setRecipeId(
          recipeId &&
          list.some(item => item.id === recipeId)
            ? recipeId
            : recipeMode === 'selected' &&
                list.length > 0
              ? list[0].id
              : '',
        )
      })
      .catch(error => {
        if (cancelled) return

        console.error(
          '获取可用配方失败:',
          error,
        )
        setRecipes([])
        setRecipeId('')
      })
      .finally(() => {
        if (!cancelled) {
          setRecipesLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [
    isK12,
    subjectValid,
    gradeValid,
    subject,
    grade,
    recipeMode,
    recipeId,
    setRecipeId,
    setRecipeMode,
  ])

  const [unitPlans, setUnitPlans] =
    useState<UnitPlanListItem[]>([])

  const [unitLoading, setUnitLoading] =
    useState(false)

  useEffect(() => {
    let cancelled = false

    if (!isK12 || !subjectValid) {
      setUnitPlans([])
      setUnitPlanId('')
      return
    }

    setUnitLoading(true)

    getMountableUnitPlans(subject)
      .then(response => {
        if (cancelled) return

        const list = response.unit_plans || []
        setUnitPlans(list)

        setUnitPlanId(
          unitPlanId &&
          list.some(item => item.id === unitPlanId)
            ? unitPlanId
            : '',
        )
      })
      .catch(error => {
        if (cancelled) return

        console.error(
          '获取可挂载单元方案失败:',
          error,
        )
        setUnitPlans([])
        setUnitPlanId('')
      })
      .finally(() => {
        if (!cancelled) {
          setUnitLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [
    isK12,
    subjectValid,
    subject,
    unitPlanId,
    setUnitPlanId,
  ])

  const [classProfiles, setClassProfiles] =
    useState<ClassProfileListItem[]>([])

  const [classLoading, setClassLoading] =
    useState(false)

  useEffect(() => {
    let cancelled = false

    if (!isK12 || !subjectValid) {
      setClassProfiles([])
      setClassProfileId('')
      return
    }

    setClassLoading(true)

    getClassProfiles()
      .then(response => {
        if (cancelled) return

        const list = (response.profiles || [])
          .filter(item => item.subject === subject)

        setClassProfiles(list)

        setClassProfileId(
          classProfileId &&
          list.some(item => item.id === classProfileId)
            ? classProfileId
            : '',
        )
      })
      .catch(error => {
        if (cancelled) return

        console.error(
          '获取班级学情卡失败:',
          error,
        )
        setClassProfiles([])
        setClassProfileId('')
      })
      .finally(() => {
        if (!cancelled) {
          setClassLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [
    isK12,
    subjectValid,
    subject,
    classProfileId,
    setClassProfileId,
  ])

  const durationButtonStyle = (
    active: boolean,
  ): React.CSSProperties => ({
    padding: '7px 16px',
    borderRadius: '20px',
    border: `1.5px solid ${
      active ? C.primary : C.border
    }`,
    background:
      active ? C.primaryLight : 'transparent',
    color:
      active ? C.primary : C.textSec,
    fontSize: '13px',
    fontWeight: active ? 600 : 400,
    cursor: 'pointer',
  })

  const recipeReady =
    !isK12 ||
    recipeMode !== 'selected' ||
    Boolean(recipeId)

  const startReady =
    Boolean(topic.trim()) &&
    subjectValid &&
    gradeValid &&
    !subjectsLoading &&
    !subjectsEmpty &&
    !courseOutlineLoading &&
    !startLoading &&
    recipeReady

  const startButtonText =
    courseOutlineLoading
      ? '正在加载精确课程大纲…'
      : startLoading
        ? `正在准备${profile.lesson_plan_label}环境…`
        : isK12 && recipeMode === 'auto'
          ? '✨ 智能匹配并开始备课'
          : isK12 && recipeMode === 'selected'
            ? recipeId
              ? '📦 带指定配方开始备课'
              : '请先选择配方'
            : `开始${profile.lesson_plan_label}`

  return (
    <div style={{
      minHeight: 'calc(100vh - 120px)',
      overflow: 'auto',
      margin: '-28px -32px',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '24px',
    }}>
      <div style={{
        width: '100%',
        maxWidth: '560px',
        textAlign: 'center',
      }}>
        <div style={{
          display: 'inline-flex',
          padding: '4px 11px',
          borderRadius: '999px',
          background: C.primaryLight,
          color: C.primary,
          fontSize: '11px',
          fontWeight: 700,
          marginBottom: '12px',
        }}>
          {profile.name}
        </div>

        <h1 style={{
          fontSize: '26px',
          fontWeight: 700,
          color: C.text,
          margin: '0 0 24px',
        }}>
          今天准备什么？
        </h1>

        <input
          type="text"
          value={topic}
          onChange={event =>
            setTopic(event.target.value)
          }
          onKeyDown={event => {
            if (onTopicDraftKeyDown?.(event)) {
              return
            }

            if (event.key === 'Enter' && startReady) {
              onStart()
            }
          }}
          placeholder={getTopicPlaceholder(domain)}
          style={{
            width: '100%',
            padding: '15px 18px',
            borderRadius: '14px',
            border: `2px solid ${C.border}`,
            fontSize: '16px',
            color: C.text,
            outline: 'none',
            boxSizing: 'border-box',
            boxShadow: '0 4px 18px rgba(0,0,0,0.05)',
          }}
          onFocus={event => {
            event.currentTarget.style.borderColor =
              C.primary
          }}
          onBlur={event => {
            event.currentTarget.style.borderColor =
              C.border
          }}
        />

        <div style={{
          marginTop: '7px',
          fontSize: '11px',
          color: C.textMuted,
        }}>
          {profile.topic_label}与下方选择已自动保存
          · Ctrl/Command+Z可恢复误删
        </div>

        <div style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: '10px',
          marginTop: '16px',
        }}>
          <div style={{ textAlign: 'left' }}>
            <label style={{
              display: 'block',
              fontSize: '11px',
              color: C.textMuted,
              marginBottom: '5px',
            }}>
              {profile.subject_label}
            </label>

            <select
              value={subject}
              disabled={subjectsLoading || subjectsEmpty}
              onChange={event =>
                setSubject(event.target.value)
              }
              style={{
                width: '100%',
                padding: '9px 12px',
                borderRadius: '10px',
                border: `1px solid ${C.border}`,
                fontSize: '14px',
                color: C.text,
                background: C.card,
              }}
            >
              {subjects.map(item => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
          </div>

          <div style={{ textAlign: 'left' }}>
            <label style={{
              display: 'block',
              fontSize: '11px',
              color: C.textMuted,
              marginBottom: '5px',
            }}>
              {profile.grade_label}
            </label>

            <select
              value={grade}
              onChange={event =>
                setGrade(event.target.value)
              }
              style={{
                width: '100%',
                padding: '9px 12px',
                borderRadius: '10px',
                border: `1px solid ${C.border}`,
                fontSize: '14px',
                color: C.text,
                background: C.card,
              }}
            >
              {levelOptions.map(item => (
                <option
                  key={item.value}
                  value={item.value}
                >
                  {item.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {subjectsEmpty && !subjectsLoading && (
          <div style={{
            marginTop: '10px',
            padding: '10px 12px',
            borderRadius: '9px',
            background: '#FEF2F2',
            color: C.danger,
            fontSize: '12px',
            lineHeight: 1.6,
          }}>
            当前组织尚未配置可用
            {profile.subject_label}，请联系管理员。
          </div>
        )}

        <div style={{
          marginTop: '14px',
          textAlign: 'left',
        }}>
          <label style={{
            display: 'block',
            fontSize: '12px',
            fontWeight: 600,
            color: C.textSec,
            marginBottom: '6px',
          }}>
            ⏱ {domain === 'adult'
              ? '培训时长'
              : '课时时长'}
          </label>

          <div style={{
            display: 'flex',
            gap: '8px',
          }}>
            {DURATION_OPTIONS.map(item => (
              <button
                key={item}
                onClick={() => setDuration(item)}
                style={durationButtonStyle(
                  duration === item,
                )}
              >
                {item}分钟
              </button>
            ))}
          </div>
        </div>

        {isK12 ? (
          <div style={{
            marginTop: '14px',
            textAlign: 'left',
          }}>
            <RecipeModeSelector
              currentSubject={subject}
              currentGrade={grade}
              mode={recipeMode}
              setMode={setRecipeMode}
              recipes={recipes}
              recipeId={recipeId}
              setRecipeId={setRecipeId}
              loading={recipesLoading}
            />
          </div>
        ) : (
          <div style={{
            marginTop: '14px',
            padding: '10px 12px',
            borderRadius: '10px',
            background: '#F8FAFC',
            color: '#64748B',
            fontSize: '11px',
            lineHeight: 1.65,
            textAlign: 'left',
          }}>
            当前使用通用教学流程骨架。
            职教和成人教育专属配方将在资源教育域
            完成硬隔离后开放，当前不会加载K12配方。
          </div>
        )}

        {isK12 && unitPlans.length > 0 && (
          <div style={{
            marginTop: '14px',
            textAlign: 'left',
          }}>
            <label style={{
              display: 'block',
              fontSize: '12px',
              fontWeight: 600,
              color: C.textSec,
              marginBottom: '6px',
            }}>
              📐 所属单元方案（选填）
            </label>

            <select
              value={unitPlanId}
              disabled={unitLoading}
              onChange={event =>
                setUnitPlanId(event.target.value)
              }
              style={{
                width: '100%',
                padding: '11px 14px',
                borderRadius: '12px',
                border: `1.5px solid ${
                  unitPlanId ? C.primary : C.border
                }`,
                fontSize: '14px',
                background: C.card,
              }}
            >
              <option value="">
                不关联单元方案
              </option>

              {unitPlans.map(item => (
                <option
                  key={item.id}
                  value={item.id}
                >
                  {item.grade}
                  {item.volume
                    ? ` · ${item.volume}`
                    : ''}
                  {' · '}
                  {item.unit}
                </option>
              ))}
            </select>
          </div>
        )}

        {isK12 &&
         profile.publisher_enabled && (
          <div style={{
            marginTop: '14px',
          }}>
            <ExactCourseOutlineSelector
              subject={subject}
              grade={grade}
              value={courseOutlineId}
              onChange={setCourseOutlineId}
              disabled={
                startLoading ||
                !subjectValid ||
                !gradeValid
              }
              onLoadingChange={
                onCourseOutlineLoadingChange
              }
            />

            {legacyCourseOutlineNotice && (
              <div
                role="alert"
                aria-live="polite"
                style={{
                  marginTop: '7px',
                  padding: '8px 10px',
                  borderRadius: '8px',
                  background: '#FFF7ED',
                  border: '1px solid #FED7AA',
                  color: '#9A3412',
                  fontSize: '11px',
                  lineHeight: 1.65,
                }}
              >
                ⚠️ {legacyCourseOutlineNotice}
              </div>
            )}
          </div>
        )}

        {isK12 && classProfiles.length > 0 && (
          <div style={{
            marginTop: '14px',
            textAlign: 'left',
          }}>
            <label style={{
              display: 'block',
              fontSize: '12px',
              fontWeight: 600,
              color: C.textSec,
              marginBottom: '6px',
            }}>
              👥 本班学情（选填）
            </label>

            <select
              value={classProfileId}
              disabled={classLoading}
              onChange={event =>
                setClassProfileId(
                  event.target.value,
                )
              }
              style={{
                width: '100%',
                padding: '11px 14px',
                borderRadius: '12px',
                border: `1.5px solid ${
                  classProfileId
                    ? '#10B981'
                    : C.border
                }`,
                fontSize: '14px',
                background: C.card,
              }}
            >
              <option value="">
                不关联班级学情
              </option>

              {classProfiles.map(item => (
                <option
                  key={item.id}
                  value={item.id}
                >
                  {item.class_name}
                  {item.grade
                    ? ` · ${item.grade}`
                    : ''}
                </option>
              ))}
            </select>
          </div>
        )}

        <button
          onClick={onStart}
          disabled={!startReady}
          style={{
            marginTop: '22px',
            padding: '13px 52px',
            borderRadius: '14px',
            border: 'none',
            background: startReady
              ? `linear-gradient(135deg, ${C.primary}, #818CF8)`
              : '#E5E7EB',
            color: startReady
              ? '#fff'
              : C.textMuted,
            fontSize: '16px',
            fontWeight: 700,
            cursor: startReady
              ? 'pointer'
              : 'not-allowed',
          }}
        >
          {startButtonText}
        </button>

        <div style={{
          borderTop: `1px solid ${C.border}`,
          margin: '28px 0 16px',
        }} />

        <div style={{
          display: 'flex',
          justifyContent: 'center',
          gap: '20px',
          fontSize: '13px',
        }}>
          <button
            onClick={onImport}
            style={{
              background: 'none',
              border: 'none',
              color: C.textSec,
              cursor: 'pointer',
            }}
          >
            📂 导入已有
            {profile.lesson_plan_label}
          </button>

          {onSwitchMode && (
            <button
              onClick={onSwitchMode}
              style={{
                background: 'none',
                border: 'none',
                color: C.textSec,
                cursor: 'pointer',
              }}
            >
              ⚙ 切换专家模式
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
