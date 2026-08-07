/**
 * EducationAwareImportPlanModal — 教育域适配的已有教学设计导入
 *
 * K12保留两步：内容信息 → 关联课本。
 * 职业教育和成人教育只有内容信息一步，不读取K12课本库。
 */

import {
  useEffect,
  useMemo,
  useState,
} from 'react'
import { useNavigate } from 'react-router-dom'
import {
  getTextbooks,
} from '@/api/textbooks'
import type {
  TextbookListItem,
} from '@/api/textbooks'
import {
  importExistingPlan,
} from '@/api/lesson-plans'
import type {
  ImportExistingPlanRequest,
  ImportExistingPlanResponse,
} from '@/api/lesson-plans'
import { useSubjects } from '@/hooks/useSubjects'
import {
  useEducationProfile,
} from '@/hooks/useEducationProfile'
import {
  getEducationLevelOptions,
  getTopicPlaceholder,
} from '@/education-domain/options'
import {
  importWordFidelityPlan,
} from '@/api/lesson-plan-word-import'
import LessonPlanImportSourceSection from './LessonPlanImportSourceSection'
import type {
  LessonPlanImportSourceSelection,
} from './LessonPlanImportSourceSection'
import { C } from './workshopConstants'

interface ImportPlanModalProps {
  onSuccess: (
    response: ImportExistingPlanResponse,
  ) => void | Promise<void>
  onCancel: () => void
}

type Step = 1 | 2

export default function EducationAwareImportPlanModal({
  onSuccess,
  onCancel,
}: ImportPlanModalProps) {
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
    () => levelOptions.map(item => item.value),
    [levelOptions],
  )

  const [step, setStep] = useState<Step>(1)

  const [subject, setSubject] = useState(
    subjects[0] || '',
  )

  const [grade, setGrade] = useState(
    levelValues[0] || '',
  )

  const [topic, setTopic] = useState('')
  const [duration, setDuration] = useState(45)

  const [
    sourceSelection,
    setSourceSelection,
  ] = useState<
    LessonPlanImportSourceSelection
  >({
    sourceType: 'paste',
    content: '',
    fileName: '',
    ready: true,
    wordImportSessionID: undefined,
    wordPreview: null,
  })

  const [textbooks, setTextbooks] =
    useState<TextbookListItem[]>([])

  const [
    textbooksLoading,
    setTextbooksLoading,
  ] = useState(false)

  const [
    selectedTextbookIds,
    setSelectedTextbookIds,
  ] = useState<Set<string>>(new Set())

  const [submitting, setSubmitting] =
    useState(false)

  const [submitError, setSubmitError] =
    useState('')

  const effectiveContent =
    sourceSelection.content


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

  useEffect(() => {
    if (!isK12) {
      setStep(1)
      setTextbooks([])
      setSelectedTextbookIds(new Set())
      return
    }

    if (step !== 2) return

    setTextbooksLoading(true)

    getTextbooks({
      subject,
      grade_range: grade,
      limit: 50,
    })
      .then(response =>
        setTextbooks(response.pages || []),
      )
      .catch(() => setTextbooks([]))
      .finally(() =>
        setTextbooksLoading(false),
      )
  }, [
    isK12,
    step,
    subject,
    grade,
  ])

  const toggleTextbook = (id: string) => {
    setSelectedTextbookIds(previous => {
      const next = new Set(previous)

      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }

      return next
    })
  }

  const formValid =
    subjects.includes(subject) &&
    levelValues.includes(grade) &&
    topic.trim().length > 0 &&
    effectiveContent.trim().length >= 50 &&
    sourceSelection.ready &&
    !subjectsLoading &&
    !subjectsEmpty

  const handleSubmit = async () => {
    if (!formValid || submitting) return

    setSubmitting(true)
    setSubmitError('')

    try {
      const textbookPageIDs =
        isK12 &&
        selectedTextbookIds.size > 0
          ? Array.from(
              selectedTextbookIds,
            )
          : undefined

      if (
        sourceSelection.sourceType ===
        'docx_fidelity'
      ) {
        if (
          !sourceSelection
            .wordImportSessionID
        ) {
          throw new Error(
            'Word预解析会话尚未就绪，请重新选择文件',
          )
        }

        const response =
          await importWordFidelityPlan({
            subject,
            grade,
            topic: topic.trim(),
            duration_minutes: duration,

            // 后端会从可信会话读取并覆盖正式正文。
            content_markdown: '',
            source_type: 'docx_fidelity',
            word_import_session_id:
              sourceSelection
                .wordImportSessionID,
            textbook_page_ids:
              textbookPageIDs,
          })

        await onSuccess(response)

        return
      }

      const ordinarySourceType:
        ImportExistingPlanRequest[
          'source_type'
        ] =
        sourceSelection.sourceType ===
        'paste'
          ? 'paste'
          : sourceSelection.sourceType ===
              'docx'
            ? 'docx'
            : 'pdf'

      const request:
        ImportExistingPlanRequest = {
        subject,
        grade,
        topic: topic.trim(),
        duration_minutes: duration,
        content_markdown:
          effectiveContent.trim(),
        source_type:
          ordinarySourceType,
      }

      if (textbookPageIDs) {
        request.textbook_page_ids =
          textbookPageIDs
      }

      const response =
        await importExistingPlan(request)

      await onSuccess(response)
    } catch (error) {
      setSubmitError(
        error instanceof Error
          ? error.message
          : '导入失败，请检查内容后重试',
      )
      setSubmitting(false)
    }
  }

  const selectStyle: React.CSSProperties = {
    width: '100%',
    padding: '10px 12px',
    borderRadius: '8px',
    border: `1px solid ${C.border}`,
    background: '#fff',
    fontSize: '14px',
    color: C.text,
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 10000,
        background: 'rgba(0,0,0,0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '20px',
      }}
      onClick={onCancel}
    >
      <div
        onClick={event =>
          event.stopPropagation()
        }
        style={{
          background: '#fff',
          borderRadius: '16px',
          width: '100%',
          maxWidth: '680px',
          maxHeight: '90vh',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        <div style={{
          padding: '20px 28px 16px',
          borderBottom: `1px solid ${C.border}`,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}>
          <div>
            <h2 style={{
              margin: 0,
              fontSize: '18px',
              color: C.text,
            }}>
              📂 导入已有
              {profile.lesson_plan_label}
            </h2>

            <p style={{
              margin: '4px 0 0',
              fontSize: '13px',
              color: C.textSec,
            }}>
              导入完成后可立即开始聊天评审，后台质量检查独立进行
            </p>
          </div>

          <span style={{
            padding: '4px 10px',
            borderRadius: '999px',
            background: C.primaryLight,
            color: C.primary,
            fontSize: '11px',
            fontWeight: 700,
          }}>
            {profile.name}
          </span>
        </div>

        <div style={{
          flex: 1,
          overflowY: 'auto',
          padding: '24px 28px',
        }}>
          {step === 1 && (
            <div style={{
              display: 'flex',
              flexDirection: 'column',
              gap: '18px',
            }}>
              <div style={{
                display: 'grid',
                gridTemplateColumns: '1fr 1fr',
                gap: '14px',
              }}>
                <div>
                  <label style={{
                    display: 'block',
                    fontSize: '13px',
                    fontWeight: 600,
                    marginBottom: '7px',
                  }}>
                    {profile.subject_label}
                  </label>

                  <select
                    value={subject}
                    disabled={
                      subjectsLoading ||
                      subjectsEmpty
                    }
                    onChange={event =>
                      setSubject(
                        event.target.value,
                      )
                    }
                    style={selectStyle}
                  >
                    {subjects.map(item => (
                      <option
                        key={item}
                        value={item}
                      >
                        {item}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label style={{
                    display: 'block',
                    fontSize: '13px',
                    fontWeight: 600,
                    marginBottom: '7px',
                  }}>
                    {profile.grade_label}
                  </label>

                  <select
                    value={grade}
                    onChange={event =>
                      setGrade(event.target.value)
                    }
                    style={selectStyle}
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
                  padding: '10px 12px',
                  borderRadius: '8px',
                  background: '#FEF2F2',
                  color: C.danger,
                  fontSize: '12px',
                }}>
                  当前组织尚未配置可用
                  {profile.subject_label}。
                </div>
              )}

              <div style={{
                display: 'grid',
                gridTemplateColumns: '1fr auto',
                gap: '14px',
                alignItems: 'end',
              }}>
                <div>
                  <label style={{
                    display: 'block',
                    fontSize: '13px',
                    fontWeight: 600,
                    marginBottom: '7px',
                  }}>
                    {profile.topic_label} *
                  </label>

                  <input
                    value={topic}
                    onChange={event =>
                      setTopic(event.target.value)
                    }
                    placeholder={
                      getTopicPlaceholder(domain)
                    }
                    style={selectStyle}
                  />
                </div>

                <div>
                  <label style={{
                    display: 'block',
                    fontSize: '13px',
                    fontWeight: 600,
                    marginBottom: '7px',
                  }}>
                    时长
                  </label>

                  <select
                    value={duration}
                    onChange={event =>
                      setDuration(
                        Number(event.target.value),
                      )
                    }
                    style={{
                      ...selectStyle,
                      width: '110px',
                    }}
                  >
                    {[40, 45, 50, 60].map(
                      item => (
                        <option
                          key={item}
                          value={item}
                        >
                          {item}分钟
                        </option>
                      ),
                    )}
                  </select>
                </div>
              </div>

              <LessonPlanImportSourceSection
                lessonPlanLabel={
                  profile.lesson_plan_label
                }
                value={sourceSelection}
                onChange={
                  setSourceSelection
                }
              />

              {submitError && (
                <div style={{
                  padding: '10px 12px',
                  borderRadius: '8px',
                  background: '#FEF2F2',
                  color: C.danger,
                  fontSize: '12px',
                }}>
                  ⚠️ {submitError}
                </div>
              )}
            </div>
          )}

          {step === 2 && isK12 && (
            <div>
              <div style={{
                padding: '12px 14px',
                borderRadius: '9px',
                background: '#F0FDF4',
                color: '#166534',
                fontSize: '12px',
                marginBottom: '16px',
              }}>
                内容已就绪。关联课本图片是可选步骤。
              </div>

              {/* 正式导入失败提示：必须在第二步直接可见。 */}
              {submitError && (
                <div
                  role="alert"
                  aria-live="assertive"
                  style={{
                    padding: '10px 12px',
                    borderRadius: '8px',
                    background: '#FEF2F2',
                    border: '1px solid #FECACA',
                    color: C.danger,
                    fontSize: '12px',
                    lineHeight: 1.65,
                    marginBottom: '16px',
                  }}
                >
                  <div style={{
                    fontWeight: 700,
                    marginBottom: '3px',
                  }}>
                    ⚠️ 导入未完成
                  </div>

                  <div>
                    {submitError}
                  </div>

                  <div style={{
                    marginTop: '5px',
                    color: '#991B1B',
                    fontSize: '11px',
                  }}>
                    可直接再次点击“导入并进入评审对话”重试，
                    或返回上一步重新选择文档。
                  </div>
                </div>
              )}

              {textbooksLoading ? (
                <div style={{
                  textAlign: 'center',
                  padding: '24px',
                  color: C.textMuted,
                }}>
                  加载课本图片中...
                </div>
              ) : textbooks.length === 0 ? (
                <div style={{
                  textAlign: 'center',
                  padding: '24px',
                  background: '#F8FAFC',
                  borderRadius: '9px',
                }}>
                  <div style={{
                    color: C.textMuted,
                    fontSize: '12px',
                  }}>
                    暂无可用课本图片
                  </div>

                  <button
                    onClick={() =>
                      navigate(
                        '/lesson-plans/textbooks',
                      )
                    }
                    style={{
                      marginTop: '10px',
                      padding: '7px 14px',
                      borderRadius: '7px',
                      border: 'none',
                      background: C.primary,
                      color: '#fff',
                    }}
                  >
                    去课本管理
                  </button>
                </div>
              ) : (
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
                        onClick={() =>
                          toggleTextbook(item.id)
                        }
                        style={{
                          padding: '8px 11px',
                          borderRadius: '8px',
                          border: selected
                            ? `1px solid ${C.primary}`
                            : `1px solid ${C.border}`,
                          background: selected
                            ? C.primaryLight
                            : '#fff',
                          cursor: 'pointer',
                        }}
                      >
                        {selected ? '✓ ' : ''}
                        {item.chapter ||
                          item.textbook_name}
                      </button>
                    )
                  })}
                </div>
              )}
            </div>
          )}
        </div>

        <div style={{
          padding: '16px 28px',
          borderTop: `1px solid ${C.border}`,
          display: 'flex',
          justifyContent: 'space-between',
          gap: '10px',
        }}>
          <button
            onClick={
              step === 1
                ? onCancel
                : () => setStep(1)
            }
            style={{
              padding: '10px 18px',
              borderRadius: '8px',
              border: `1px solid ${C.border}`,
              background: '#fff',
            }}
          >
            {step === 1
              ? '取消'
              : '← 上一步'}
          </button>

          {step === 1 && isK12 ? (
            <button
              onClick={() => setStep(2)}
              disabled={!formValid}
              style={{
                padding: '10px 22px',
                borderRadius: '8px',
                border: 'none',
                background: formValid
                  ? C.primary
                  : '#E5E7EB',
                color: formValid
                  ? '#fff'
                  : C.textMuted,
              }}
            >
              下一步：关联课本 →
            </button>
          ) : (
            <button
              onClick={handleSubmit}
              disabled={!formValid || submitting}
              style={{
                padding: '10px 22px',
                borderRadius: '8px',
                border: 'none',
                background:
                  formValid && !submitting
                    ? C.primary
                    : '#E5E7EB',
                color:
                  formValid && !submitting
                    ? '#fff'
                    : C.textMuted,
              }}
            >
              {submitting
                ? '导入中...'
                : `导入并进入评审对话`}
            </button>
          )}

        </div>
      </div>
    </div>
  )
}
