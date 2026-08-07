/**
 * EducationAwareStartForm — 教育域适配的专家模式起步表单
 *
 * 对外Props保持原StartForm协议不变。
 *
 * 所有具体教学教育域均支持：
 *   - 当前教育域课程目录；
 *   - 当前教育域具体学习层级；
 *   - 自动选择、指定或不使用备课配方；
 *   - 严格按课程与具体层级匹配配方。
 *
 * K12额外保留：
 *   - 精确课程大纲（唯一ID）；
 *   - 课本页面图片与OCR。
 *
 * 职业教育与成人教育不加载K12精确课程大纲或课本图片，
 * 但可以正常创建、选择和自动匹配本教育域配方。
 */

import {
  useEffect,
  useMemo,
  useState,
} from 'react'
import { useNavigate } from 'react-router-dom'
import type {
  RecipeSelectionMode,
} from '@/api/lesson-plans'
import {
  getAvailableRecipes,
} from '@/api/recipes'
import type {
  RecipeListItem,
} from '@/api/recipes'
import {
  getTextbooks,
  triggerTextbookOCR,
} from '@/api/textbooks'
import type {
  TextbookListItem,
} from '@/api/textbooks'
import ExactCourseOutlineSelector from './ExactCourseOutlineSelector'
import {
  useEducationProfile,
} from '@/hooks/useEducationProfile'
import { useSubjects } from '@/hooks/useSubjects'
import {
  getEducationLevelOptions,
  getTopicPlaceholder,
} from '@/education-domain/options'
import {
  C,
} from './workshopConstants'
import RecipeModeSelector from './RecipeModeSelector'

interface StartFormProps {
  onStart: (
    subject: string,
    grade: string,
    topic: string,
    duration: number,
    recipeMode: RecipeSelectionMode,
    recipeId?: string,
    textbookPageIds?: string[],
    courseOutlineId?: string | null,
  ) => void
  loading: boolean
}

export default function EducationAwareStartForm({
  onStart,
  loading,
}: StartFormProps) {
  const navigate = useNavigate()

  const {
    domain,
    profile,
    isK12,
  } = useEducationProfile()

  const {
    subjects,
    loading: subjectsLoading,
    empty: subjectsEmpty,
  } = useSubjects()

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

  const [subject, setSubject] = useState(
    subjects[0] || '',
  )

  const [grade, setGrade] = useState(
    levelValues[0] || '',
  )

  const [topic, setTopic] = useState('')
  const [duration, setDuration] = useState(45)

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
  ])

  useEffect(() => {
    if (
      levelValues.length > 0 &&
      !levelValues.includes(grade)
    ) {
      setGrade(levelValues[0])
    }
  }, [
    levelValues,
    grade,
  ])

  /**
   * 专家模式开始备课的课程大纲正式值。
   * null表示明确不关联；任何非空值都必须是候选接口返回的唯一ID。
   */
  const [courseOutlineId, setCourseOutlineId] = useState<string | null>(null)
  const [courseOutlineLoading, setCourseOutlineLoading] = useState(false)

  /** 非K12或关闭教材版本能力时，清除可能残留的精确课程大纲ID。 */
  useEffect(() => {
    if (isK12 && profile.publisher_enabled) return
    setCourseOutlineId(null)
    setCourseOutlineLoading(false)
  }, [isK12, profile.publisher_enabled])

  const [recipes, setRecipes] =
    useState<RecipeListItem[]>([])

  const [recipesLoading, setRecipesLoading] =
    useState(false)

  const [selectedRecipeId, setSelectedRecipeId] =
    useState<string | null>(null)

  /**
   * 所有具体教学教育域默认使用智能选择。
   *
   * 没有严格匹配配方时，后端会自动回退到系统阶段骨架，
   * 因此不会因为当前学校尚未创建配方而阻断备课。
   */
  const [recipeMode, setRecipeMode] =
    useState<RecipeSelectionMode>('auto')

  useEffect(() => {
    let cancelled = false

    if (!subject || !grade) {
      setRecipes([])
      setSelectedRecipeId(null)
      return
    }

    setRecipesLoading(true)

    getAvailableRecipes(subject, grade)
      .then(response => {
        if (cancelled) return

        const list =
          response.recipes || []

        setRecipes(list)

        setSelectedRecipeId(previous =>
          previous &&
          list.some(
            item => item.id === previous,
          )
            ? previous
            : recipeMode === 'selected' &&
                list.length > 0
              ? list[0].id
              : null,
        )
      })
      .catch(() => {
        if (!cancelled) {
          setRecipes([])
          setSelectedRecipeId(null)
        }
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
    subject,
    grade,
    recipeMode,
  ])

  const [textbooks, setTextbooks] =
    useState<TextbookListItem[]>([])

  const [textbooksLoading, setTextbooksLoading] =
    useState(false)

  const [textbooksLoaded, setTextbooksLoaded] =
    useState(false)

  const [
    selectedTextbookIds,
    setSelectedTextbookIds,
  ] = useState<Set<string>>(new Set())

  const [ocrInProgress, setOcrInProgress] =
    useState<Set<string>>(new Set())

  const [ocrFailed, setOcrFailed] =
    useState<Set<string>>(new Set())

  useEffect(() => {
    let cancelled = false

    if (!isK12 || !subject || !grade) {
      setTextbooks([])
      setTextbooksLoaded(true)
      setSelectedTextbookIds(new Set())
      return
    }

    setTextbooksLoading(true)
    setTextbooksLoaded(false)

    getTextbooks({
      subject,
      grade_range: grade,
      limit: 50,
    })
      .then(response => {
        if (cancelled) return

        setTextbooks(
          response.pages || [],
        )
        setSelectedTextbookIds(
          new Set(),
        )
      })
      .catch(() => {
        if (!cancelled) {
          setTextbooks([])
        }
      })
      .finally(() => {
        if (!cancelled) {
          setTextbooksLoading(false)
          setTextbooksLoaded(true)
        }
      })

    return () => {
      cancelled = true
    }
  }, [
    isK12,
    subject,
    grade,
  ])

  const toggleTextbook = (id: string) => {
    const willSelect =
      !selectedTextbookIds.has(id)

    setSelectedTextbookIds(previous => {
      const next = new Set(previous)

      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }

      return next
    })

    if (willSelect) {
      const textbook = textbooks.find(
        item => item.id === id,
      )

      if (
        textbook &&
        !textbook.has_ocr &&
        !ocrInProgress.has(id)
      ) {
        void triggerOCR(id)
      }
    }
  }

  const triggerOCR = async (id: string) => {
    setOcrInProgress(previous => {
      const next = new Set(previous)
      next.add(id)
      return next
    })

    setOcrFailed(previous => {
      const next = new Set(previous)
      next.delete(id)
      return next
    })

    try {
      await triggerTextbookOCR(id)

      setTextbooks(previous =>
        previous.map(item =>
          item.id === id
            ? {
                ...item,
                has_ocr: true,
              }
            : item,
        ),
      )
    } catch {
      setOcrFailed(previous => {
        const next = new Set(previous)
        next.add(id)
        return next
      })
    } finally {
      setOcrInProgress(previous => {
        const next = new Set(previous)
        next.delete(id)
        return next
      })
    }
  }

  const subjectValid =
    subjects.includes(subject)

  const gradeValid =
    levelValues.includes(grade)

  const recipeReady =
    recipeMode !== 'selected' ||
    Boolean(selectedRecipeId)

  const startReady =
    subjectValid &&
    gradeValid &&
    Boolean(topic.trim()) &&
    !loading &&
    !subjectsLoading &&
    !subjectsEmpty &&
    !courseOutlineLoading &&
    ocrInProgress.size === 0 &&
    recipeReady

  const handleSubmit = () => {
    if (!startReady) return

    onStart(
      subject,
      grade,
      topic.trim(),
      duration,
      recipeMode,
      recipeMode === 'selected'
        ? selectedRecipeId || undefined
        : undefined,
      isK12 &&
      selectedTextbookIds.size > 0
        ? Array.from(
            selectedTextbookIds,
          )
        : undefined,
      isK12
        ? courseOutlineId
        : null,
    )
  }

  const selectionButton = (
    active: boolean,
  ): React.CSSProperties => ({
    padding: '6px 14px',
    borderRadius: '20px',
    border: `1px solid ${
      active ? C.primary : C.border
    }`,
    background:
      active
        ? C.primaryLight
        : 'transparent',
    color:
      active
        ? C.primary
        : C.textSec,
    fontSize: '13px',
    fontWeight: active ? 600 : 400,
    cursor: 'pointer',
  })

  return (
    <div style={{
      maxWidth: '720px',
      margin: '0 auto',
      padding: '32px 0',
    }}>
      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        marginBottom: '22px',
      }}>
        <span style={{ fontSize: '28px' }}>
          ✨
        </span>

        <div>
          <h1 style={{
            fontSize: '20px',
            fontWeight: 700,
            color: C.text,
            margin: 0,
          }}>
            开始今天的
            {profile.lesson_plan_label}
          </h1>

          <p style={{
            fontSize: '13px',
            color: C.textSec,
            margin: '3px 0 0',
          }}>
            当前工作域：
            {profile.name}
          </p>
        </div>
      </div>

      <div style={{
        background: C.card,
        borderRadius: '16px',
        padding: '28px',
        border: `1px solid ${C.border}`,
        boxShadow:
          '0 4px 24px rgba(0,0,0,0.06)',
      }}>
        <div style={{
          marginBottom: '18px',
        }}>
          <label style={{
            display: 'block',
            fontSize: '14px',
            fontWeight: 600,
            color: C.text,
            marginBottom: '8px',
          }}>
            {profile.subject_label}
          </label>

          <div style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: '6px',
          }}>
            {subjects.map(item => (
              <button
                key={item}
                type="button"
                onClick={() =>
                  setSubject(item)
                }
                style={selectionButton(
                  subject === item,
                )}
              >
                {item}
              </button>
            ))}
          </div>

          {subjectsEmpty &&
           !subjectsLoading && (
            <div style={{
              marginTop: '8px',
              color: C.danger,
              fontSize: '12px',
            }}>
              当前组织尚未配置可用
              {profile.subject_label}。
            </div>
          )}
        </div>

        <div style={{
          marginBottom: '18px',
        }}>
          <label style={{
            display: 'block',
            fontSize: '14px',
            fontWeight: 600,
            color: C.text,
            marginBottom: '8px',
          }}>
            {profile.grade_label}
          </label>

          <div style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: '6px',
          }}>
            {levelOptions.map(item => (
              <button
                key={item.value}
                type="button"
                onClick={() =>
                  setGrade(item.value)
                }
                style={selectionButton(
                  grade === item.value,
                )}
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>

        <div style={{
          marginBottom: '18px',
        }}>
          <label style={{
            display: 'block',
            fontSize: '14px',
            fontWeight: 600,
            color: C.text,
            marginBottom: '8px',
          }}>
            {profile.topic_label}
            <span style={{
              color: C.danger,
            }}>
              {' '}*
            </span>
          </label>

          <input
            type="text"
            value={topic}
            onChange={event =>
              setTopic(
                event.target.value,
              )
            }
            onKeyDown={event => {
              if (
                event.key === 'Enter' &&
                startReady
              ) {
                handleSubmit()
              }
            }}
            placeholder={
              getTopicPlaceholder(domain)
            }
            style={{
              width: '100%',
              padding: '10px 14px',
              borderRadius: '8px',
              border:
                `1px solid ${C.border}`,
              fontSize: '15px',
              boxSizing: 'border-box',
            }}
          />
        </div>

        <div style={{
          marginBottom: '18px',
        }}>
          <label style={{
            display: 'block',
            fontSize: '14px',
            fontWeight: 600,
            color: C.text,
            marginBottom: '8px',
          }}>
            {domain === 'adult'
              ? '培训时长'
              : '课时时长'}
          </label>

          <div style={{
            display: 'flex',
            gap: '8px',
          }}>
            {[40, 45, 50, 60].map(
              item => (
                <button
                  key={item}
                  type="button"
                  onClick={() =>
                    setDuration(item)
                  }
                  style={selectionButton(
                    duration === item,
                  )}
                >
                  {item}分钟
                </button>
              ),
            )}
          </div>
        </div>

        <div style={{
          marginBottom: '18px',
        }}>
          <RecipeModeSelector
            currentSubject={subject}
            currentGrade={grade}
            mode={recipeMode}
            setMode={setRecipeMode}
            recipes={recipes}
            recipeId={
              selectedRecipeId || ''
            }
            setRecipeId={value =>
              setSelectedRecipeId(
                value || null,
              )
            }
            loading={recipesLoading}
          />
        </div>

        {isK12 &&
         profile.publisher_enabled && (
          <div style={{
            marginBottom: '18px',
          }}>
            <ExactCourseOutlineSelector
              subject={subject}
              grade={grade}
              value={courseOutlineId}
              onChange={setCourseOutlineId}
              disabled={
                loading ||
                !subjectValid ||
                !gradeValid
              }
              onLoadingChange={
                setCourseOutlineLoading
              }
            />
          </div>
        )}

        <button
          type="button"
          onClick={handleSubmit}
          disabled={!startReady}
          style={{
            width: '100%',
            padding: '14px',
            borderRadius: '10px',
            border: 'none',
            background: startReady
              ? C.primary
              : '#E5E7EB',
            color: startReady
              ? '#fff'
              : C.textMuted,
            fontSize: '16px',
            fontWeight: 600,
            cursor: startReady
              ? 'pointer'
              : 'not-allowed',
          }}
        >
          {courseOutlineLoading
            ? '正在加载精确课程大纲...'
            : loading
              ? `正在准备${profile.lesson_plan_label}环境...`
              : `开始${profile.lesson_plan_label} →`}
        </button>
      </div>

      {isK12 &&
       textbooksLoaded &&
       textbooks.length > 0 && (
        <div style={{
          marginTop: '16px',
          background: '#F0FDF4',
          borderRadius: '12px',
          padding: '16px 20px',
          border:
            '1px solid #BBF7D0',
        }}>
          <div style={{
            fontSize: '14px',
            fontWeight: 600,
            color: '#166534',
            marginBottom: '10px',
          }}>
            📷 关联课本图片
          </div>

          <div style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: '8px',
          }}>
            {textbooks.map(item => {
              const selected =
                selectedTextbookIds.has(
                  item.id,
                )

              return (
                <button
                  key={item.id}
                  type="button"
                  onClick={() =>
                    toggleTextbook(
                      item.id,
                    )
                  }
                  style={{
                    padding: '7px 10px',
                    borderRadius: '8px',
                    border: selected
                      ? `1px solid ${C.primary}`
                      : `1px solid ${C.border}`,
                    background: selected
                      ? C.primaryLight
                      : C.card,
                    cursor: 'pointer',
                    fontSize: '12px',
                  }}
                >
                  {selected ? '✓ ' : ''}
                  {item.chapter ||
                    item.textbook_name}

                  {ocrInProgress.has(
                    item.id,
                  )
                    ? ' · 识别中'
                    : ocrFailed.has(
                          item.id,
                        )
                      ? ' · 识别失败'
                      : ''}
                </button>
              )
            })}
          </div>
        </div>
      )}

      {isK12 &&
       textbooksLoading && (
        <div style={{
          marginTop: '14px',
          textAlign: 'center',
          color: C.textMuted,
          fontSize: '12px',
        }}>
          加载课本图片...
        </div>
      )}

      <div style={{
        display: 'flex',
        justifyContent: 'center',
        flexWrap: 'wrap',
        gap: '8px',
        marginTop: '20px',
      }}>
        <button
          type="button"
          onClick={() =>
            navigate(
              '/lesson-plans/my-plans',
            )
          }
          style={{
            border: 'none',
            background: 'transparent',
            color: C.textSec,
            cursor: 'pointer',
          }}
        >
          📋 我的
          {profile.lesson_plan_label}
        </button>

        <button
          type="button"
          onClick={() =>
            navigate(
              '/lesson-plans/recipes',
            )
          }
          style={{
            border: 'none',
            background: 'transparent',
            color: C.textSec,
            cursor: 'pointer',
          }}
        >
          📦 配方管理
        </button>

        {isK12 && (
          <>
            <button
              type="button"
              onClick={() =>
                navigate(
                  '/lesson-plans/library',
                )
              }
              style={{
                border: 'none',
                background: 'transparent',
                color: C.textSec,
                cursor: 'pointer',
              }}
            >
              📚 教案库
            </button>

            <button
              type="button"
              onClick={() =>
                navigate(
                  '/lesson-plans/textbooks',
                )
              }
              style={{
                border: 'none',
                background: 'transparent',
                color: C.textSec,
                cursor: 'pointer',
              }}
            >
              📷 课本管理
            </button>
          </>
        )}
      </div>
    </div>
  )
}
