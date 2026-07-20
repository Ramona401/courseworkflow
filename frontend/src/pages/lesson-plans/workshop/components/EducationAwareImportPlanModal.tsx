/**
 * EducationAwareImportPlanModal — 教育域适配的已有教学设计导入
 *
 * K12保留两步：内容信息 → 关联课本。
 * 职业教育和成人教育只有内容信息一步，不读取K12课本库。
 */

import {
  useEffect,
  useMemo,
  useRef,
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
  ConversationMessage,
  ImportExistingPlanRequest,
} from '@/api/lesson-plans'
import { useSubjects } from '@/hooks/useSubjects'
import {
  useEducationProfile,
} from '@/hooks/useEducationProfile'
import {
  getEducationLevelOptions,
  getTopicPlaceholder,
} from '@/education-domain/options'
import { C } from './workshopConstants'

interface ImportPlanModalProps {
  onSuccess: (
    planId: string,
    openingMessage: ConversationMessage,
  ) => void
  onCancel: () => void
}

type SourceType = 'paste' | 'docx' | 'pdf'
type Step = 1 | 2

const LIBS_BASE =
  'https://workflow.pkuailab.com/uploads/courseware-assets/libs'

const JSZIP_URL =
  LIBS_BASE + '/jszip/3.10.1/jszip.min.js'

const PDFJS_URL =
  LIBS_BASE + '/pdfjs-dist/3.11.174/build/pdf.min.js'

const PDFJS_WORKER_URL =
  LIBS_BASE + '/pdfjs-dist/3.11.174/build/pdf.worker.min.js'

function loadScript(
  src: string,
  globalKey: string,
): Promise<void> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  if ((window as any)[globalKey]) {
    return Promise.resolve()
  }

  return new Promise((resolve, reject) => {
    const existing =
      document.querySelector(`script[src="${src}"]`)

    if (existing) {
      existing.addEventListener(
        'load',
        () => resolve(),
      )
      existing.addEventListener(
        'error',
        () => reject(
          new Error(`加载失败: ${src}`),
        ),
      )
      return
    }

    const script = document.createElement('script')
    script.src = src
    script.onload = () => resolve()
    script.onerror = () =>
      reject(new Error(`脚本加载失败: ${src}`))

    document.head.appendChild(script)
  })
}

async function parseDocxFile(
  file: File,
): Promise<string> {
  await loadScript(JSZIP_URL, 'JSZip')

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const JSZip = (window as any).JSZip
  if (!JSZip) {
    throw new Error('JSZip加载失败')
  }

  const zip = await JSZip.loadAsync(
    await file.arrayBuffer(),
  )

  const documentFile =
    zip.file('word/document.xml')

  if (!documentFile) {
    throw new Error('不是有效的docx文件')
  }

  const xml = await documentFile.async('string')
  const xmlDocument = new DOMParser()
    .parseFromString(xml, 'application/xml')

  const paragraphs =
    xmlDocument.querySelectorAll('w\\:p, p')

  const lines: string[] = []

  paragraphs.forEach(paragraph => {
    const texts =
      paragraph.querySelectorAll('w\\:t, t')

    const line = Array.from(texts)
      .map(item => item.textContent || '')
      .join('')
      .trim()

    if (line) lines.push(line)
  })

  return lines.join('\n')
}

async function parsePdfFile(
  file: File,
): Promise<string> {
  await loadScript(PDFJS_URL, 'pdfjsLib')

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const pdfjsLib = (window as any).pdfjsLib

  if (!pdfjsLib) {
    throw new Error('pdf.js加载失败')
  }

  pdfjsLib.GlobalWorkerOptions.workerSrc =
    PDFJS_WORKER_URL

  const pdf = await pdfjsLib
    .getDocument({
      data: await file.arrayBuffer(),
    })
    .promise

  const pages: string[] = []

  for (
    let pageNumber = 1;
    pageNumber <= pdf.numPages;
    pageNumber += 1
  ) {
    const page = await pdf.getPage(pageNumber)
    const content = await page.getTextContent()

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const text = content.items
      .map((item: any) => item.str)
      .join(' ')
      .trim()

    if (text) pages.push(text)
  }

  return pages.join('\n\n')
}

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

  const [sourceType, setSourceType] =
    useState<SourceType>('paste')

  const [pasteContent, setPasteContent] =
    useState('')

  const [parsedContent, setParsedContent] =
    useState('')

  const [fileName, setFileName] =
    useState('')

  const [parseError, setParseError] =
    useState('')

  const [parsing, setParsing] =
    useState(false)

  const fileInputRef =
    useRef<HTMLInputElement>(null)

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
    sourceType === 'paste'
      ? pasteContent
      : parsedContent

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

  const handleFileChange = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0]
    if (!file) return

    setFileName(file.name)
    setParseError('')
    setParsedContent('')
    setParsing(true)

    try {
      const text =
        sourceType === 'docx'
          ? await parseDocxFile(file)
          : await parsePdfFile(file)

      if (!text.trim()) {
        setParseError(
          sourceType === 'pdf'
            ? '该PDF为扫描件或无可提取文字，请改用粘贴方式'
            : '文档内容为空或无法提取，请改用粘贴方式',
        )
      } else {
        setParsedContent(text.trim())
      }
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : ''

      setParseError(
        message.includes('加载失败')
          ? '解析库加载失败，请检查网络或改用粘贴方式'
          : '文件解析失败，请检查文件格式或改用粘贴方式',
      )
    } finally {
      setParsing(false)

      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  const changeSourceType = (
    next: SourceType,
  ) => {
    setSourceType(next)
    setParsedContent('')
    setFileName('')
    setParseError('')
  }

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
    !subjectsLoading &&
    !subjectsEmpty

  const handleSubmit = async () => {
    if (!formValid || submitting) return

    setSubmitting(true)
    setSubmitError('')

    try {
      const request: ImportExistingPlanRequest = {
        subject,
        grade,
        topic: topic.trim(),
        duration_minutes: duration,
        content_markdown:
          effectiveContent.trim(),
        source_type: sourceType,
      }

      if (
        isK12 &&
        selectedTextbookIds.size > 0
      ) {
        request.textbook_page_ids =
          Array.from(selectedTextbookIds)
      }

      const response =
        await importExistingPlan(request)

      onSuccess(
        response.plan.id,
        response.opening_message,
      )
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
              导入后由AI自动评审并提供改进建议
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

              <div>
                <label style={{
                  display: 'block',
                  fontSize: '13px',
                  fontWeight: 600,
                  marginBottom: '8px',
                }}>
                  {profile.lesson_plan_label}来源
                </label>

                <div style={{
                  display: 'flex',
                  gap: '8px',
                }}>
                  {([
                    {
                      key: 'paste',
                      label: '📋 粘贴文本',
                    },
                    {
                      key: 'docx',
                      label: '📝 Word文档',
                    },
                    {
                      key: 'pdf',
                      label: '📄 PDF文件',
                    },
                  ] as {
                    key: SourceType
                    label: string
                  }[]).map(item => (
                    <button
                      key={item.key}
                      onClick={() =>
                        changeSourceType(item.key)
                      }
                      style={{
                        flex: 1,
                        padding: '10px',
                        borderRadius: '9px',
                        border: `1.5px solid ${
                          sourceType === item.key
                            ? C.primary
                            : C.border
                        }`,
                        background:
                          sourceType === item.key
                            ? C.primaryLight
                            : '#fff',
                        cursor: 'pointer',
                      }}
                    >
                      {item.label}
                    </button>
                  ))}
                </div>
              </div>

              {sourceType === 'paste' ? (
                <div>
                  <textarea
                    value={pasteContent}
                    onChange={event =>
                      setPasteContent(
                        event.target.value,
                      )
                    }
                    rows={12}
                    placeholder={
                      `将已有${profile.lesson_plan_label}内容粘贴到这里，至少50字`
                    }
                    style={{
                      width: '100%',
                      boxSizing: 'border-box',
                      padding: '12px 14px',
                      borderRadius: '8px',
                      border: `1px solid ${C.border}`,
                      fontSize: '13px',
                      lineHeight: 1.8,
                      resize: 'vertical',
                    }}
                  />

                  <div style={{
                    marginTop: '4px',
                    textAlign: 'right',
                    color: C.textMuted,
                    fontSize: '11px',
                  }}>
                    已输入
                    {' '}
                    {
                      pasteContent
                        .replace(/\s/g, '')
                        .length
                    }
                    {' '}
                    字
                  </div>
                </div>
              ) : (
                <div>
                  <div
                    onClick={() =>
                      fileInputRef.current?.click()
                    }
                    style={{
                      border: `2px dashed ${
                        parsedContent
                          ? C.success
                          : parseError
                            ? C.danger
                            : C.border
                      }`,
                      borderRadius: '10px',
                      padding: '28px',
                      textAlign: 'center',
                      cursor: 'pointer',
                      background: '#FAFAFA',
                    }}
                  >
                    {parsing
                      ? '⏳ 正在解析文档...'
                      : parsedContent
                        ? `✅ ${fileName} · 已提取${parsedContent.replace(/\s/g, '').length}字`
                        : `点击选择${sourceType === 'docx' ? 'Word文档' : 'PDF文件'}`}
                  </div>

                  <input
                    ref={fileInputRef}
                    type="file"
                    accept={
                      sourceType === 'docx'
                        ? '.docx'
                        : '.pdf'
                    }
                    onChange={handleFileChange}
                    style={{ display: 'none' }}
                  />

                  {parseError && (
                    <div style={{
                      marginTop: '8px',
                      color: C.danger,
                      fontSize: '12px',
                    }}>
                      ⚠️ {parseError}
                    </div>
                  )}

                  {parsedContent && (
                    <div style={{
                      marginTop: '8px',
                      padding: '10px 12px',
                      borderRadius: '8px',
                      background: '#F9FAFB',
                      maxHeight: '110px',
                      overflowY: 'auto',
                      fontSize: '12px',
                      whiteSpace: 'pre-wrap',
                    }}>
                      {parsedContent.slice(0, 400)}
                    </div>
                  )}
                </div>
              )}

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
                : `开始导入并AI评审`}
            </button>
          )}

        </div>
      </div>
    </div>
  )
}
