/**
 * AssistantStyleProfileModal.tsx — 从历史教案生成教学风格与成长画像
 *
 * 三种材料来源为“任选一种”，不是三个连续必填步骤：
 *   1. 从平台选择本人教案
 *   2. 上传 Word/PDF
 *   3. 粘贴教学材料
 *
 * 页面通过来源切换标签一次只展示一种入口，避免老师误解为三项都必须填写。
 * 老师可以只使用一种来源，也可以切换来源继续追加，最多加入5份材料。
 *
 * 原始文件不落库：
 *   - Word/PDF 在浏览器端提取文字；
 *   - 平台教案只发送 ID，由后端读取并校验归属；
 *   - 画像生成后由老师人工修改确认，再交给现有助手设计器。
 */

import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type CSSProperties,
} from 'react'
import {
  getLessonPlans,
  type LessonPlan,
} from '@/api/lesson-plans'
import {
  analyzeAssistantStyleProfile,
  type StyleProfileIntent,
  type StyleProfileMaterial,
  type StyleProfileResponse,
  type StyleProfileSourceType,
} from '@/api/assistant-style-profile'
import {
  extractDocFile,
} from '@/pages/lesson-plans/workshop/utils/docExtract'

const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981',
  successLight: 'rgba(16,185,129,0.08)',
  accent: '#F59E0B',
  danger: '#EF4444',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  bg: '#FAFBFC',
  card: '#FFFFFF',
  border: '#E5E7EB',
  borderMid: '#D1D5DB',
}

const MAX_MATERIALS = 5
const MAX_LOCAL_RUNES = 12000
const MAX_TOTAL_RUNES = 30000

type SourceMode = 'platform' | 'file' | 'paste'

interface MaterialDraft extends StyleProfileMaterial {
  key: string
  localRuneCount?: number
}

interface AssistantStyleProfileModalProps {
  open: boolean
  subject: string
  grade: string
  currentUserID: string
  onClose: () => void
  onUseProfile: (profile: string) => void
}

const SOURCE_TABS: Array<{
  value: SourceMode
  icon: string
  label: string
  description: string
}> = [
  {
    value: 'platform',
    icon: '🗂️',
    label: '从平台选择',
    description: '选择自己已经保存的教案',
  },
  {
    value: 'file',
    icon: '📄',
    label: '上传 Word/PDF',
    description: '从本地教学文档提取文字',
  },
  {
    value: 'paste',
    icon: '📋',
    label: '粘贴文字',
    description: '粘贴教研要求、教案或评课意见',
  },
]

const INTENT_OPTIONS: Array<{
  value: StyleProfileIntent
  label: string
  hint: string
}> = [
  {
    value: 'satisfied_example',
    label: '满意范例',
    hint: '提取值得保留的教学优势',
  },
  {
    value: 'representative',
    label: '个人代表作',
    hint: '作为个人长期风格的重要证据',
  },
  {
    value: 'local_standard',
    label: '地方或学校规范',
    hint: '提取必须遵循的教研要求',
  },
  {
    value: 'needs_improvement',
    label: '旧教案，希望优化',
    hint: '识别现有基础、问题和成长方向',
  },
  {
    value: 'structure_only',
    label: '只参考结构',
    hint: '只分析栏目、环节和组织方式',
  },
  {
    value: 'language_only',
    label: '只参考表达风格',
    hint: '只分析语气、措辞和解释方式',
  },
  {
    value: 'negative_example',
    label: '反面样例',
    hint: '只提取需要避免的问题',
  },
]

function runeLength(text: string): number {
  return Array.from(text).length
}

function makeKey(): string {
  return `material_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

function sourceLabel(source: StyleProfileSourceType): string {
  switch (source) {
    case 'platform_plan':
      return '平台教案'
    case 'docx':
      return 'Word文档'
    case 'pdf':
      return 'PDF文档'
    case 'pasted':
      return '粘贴文字'
  }
}

function confidenceLabel(confidence: string): string {
  switch (confidence) {
    case 'low':
      return '初步判断'
    case 'medium':
      return '中等可信'
    case 'high':
      return '较高可信'
    default:
      return confidence
  }
}

export default function AssistantStyleProfileModal(
  props: AssistantStyleProfileModalProps,
) {
  const {
    open,
    subject,
    grade,
    currentUserID,
    onClose,
    onUseProfile,
  } = props

  const [sourceMode, setSourceMode] = useState<SourceMode>('platform')

  const [plans, setPlans] = useState<LessonPlan[]>([])
  const [plansLoading, setPlansLoading] = useState(false)
  const [plansError, setPlansError] = useState('')
  const [selectedPlanID, setSelectedPlanID] = useState('')

  const [materials, setMaterials] = useState<MaterialDraft[]>([])
  const [pasteTitle, setPasteTitle] = useState('')
  const [pasteText, setPasteText] = useState('')

  const [fileBusy, setFileBusy] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [errorMsg, setErrorMsg] = useState('')

  const [profileResult, setProfileResult] =
    useState<StyleProfileResponse | null>(null)
  const [profileText, setProfileText] = useState('')

  const fileInputRef = useRef<HTMLInputElement>(null)

  const localTotalRunes = useMemo(
    () => materials.reduce(
      (sum, material) => sum + (material.localRuneCount || 0),
      0,
    ),
    [materials],
  )

  const busy = generating || fileBusy

  const invalidateProfile = () => {
    setProfileResult(null)
    setProfileText('')
  }

  useEffect(() => {
    if (!open) return

    setSourceMode('platform')
    setMaterials([])
    setSelectedPlanID('')
    setPasteTitle('')
    setPasteText('')
    setErrorMsg('')
    setProfileResult(null)
    setProfileText('')

    if (!currentUserID) {
      setPlans([])
      setPlansError('未识别当前用户，暂时无法读取平台教案')
      return
    }

    let cancelled = false

    setPlansLoading(true)
    setPlansError('')

    getLessonPlans({
      author_id: currentUserID,
      subject: subject || undefined,
      limit: 100,
      offset: 0,
    })
      .then(response => {
        if (cancelled) return

        const items = response.lesson_plans || []
        setPlans(items)

        if (items.length > 0) {
          setSelectedPlanID(items[0].id)
        }
      })
      .catch(error => {
        if (cancelled) return

        setPlans([])
        setPlansError(
          error instanceof Error
            ? error.message
            : '读取我的教案失败',
        )
      })
      .finally(() => {
        if (!cancelled) {
          setPlansLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [open, currentUserID, subject])

  useEffect(() => {
    if (!open) return

    const handler = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !busy) {
        onClose()
      }
    }

    document.addEventListener('keydown', handler)

    return () => {
      document.removeEventListener('keydown', handler)
    }
  }, [open, busy, onClose])

  if (!open) return null

  const addMaterial = (material: MaterialDraft): boolean => {
    setErrorMsg('')

    if (materials.length >= MAX_MATERIALS) {
      setErrorMsg(`一次最多分析${MAX_MATERIALS}份材料`)
      return false
    }

    if (
      material.source_type === 'platform_plan'
      && material.source_id
      && materials.some(
        item =>
          item.source_type === 'platform_plan'
          && item.source_id === material.source_id,
      )
    ) {
      setErrorMsg('这份平台教案已经添加')
      return false
    }

    setMaterials(previous => [...previous, material])
    invalidateProfile()

    return true
  }

  const handleAddPlatformPlan = () => {
    const plan = plans.find(item => item.id === selectedPlanID)

    if (!plan) {
      setErrorMsg('请先选择一份平台教案')
      return
    }

    addMaterial({
      key: makeKey(),
      title: plan.title || plan.topic || '未命名教案',
      source_type: 'platform_plan',
      source_id: plan.id,
      intent: 'representative',
    })
  }

  const handleFiles = async (
    event: ChangeEvent<HTMLInputElement>,
  ) => {
    const selectedFiles = Array.from(event.target.files || [])
    event.target.value = ''

    if (selectedFiles.length === 0) return

    const available = MAX_MATERIALS - materials.length

    if (available <= 0) {
      setErrorMsg(`一次最多分析${MAX_MATERIALS}份材料`)
      return
    }

    const filesToRead = selectedFiles.slice(0, available)

    if (selectedFiles.length > available) {
      setErrorMsg(
        `当前最多还能添加${available}份，已只处理前${available}个文件`,
      )
    } else {
      setErrorMsg('')
    }

    setFileBusy(true)

    try {
      const newMaterials: MaterialDraft[] = []

      for (const file of filesToRead) {
        const result = await extractDocFile(file)
        const characterCount = runeLength(result.text)

        if (characterCount > MAX_LOCAL_RUNES) {
          throw new Error(
            `《${file.name}》共${characterCount}个字符，`
            + `单份材料最多${MAX_LOCAL_RUNES}个字符，请先精简`,
          )
        }

        const sourceType: StyleProfileSourceType =
          file.name.toLowerCase().endsWith('.pdf')
            ? 'pdf'
            : 'docx'

        newMaterials.push({
          key: makeKey(),
          title: file.name,
          source_type: sourceType,
          intent: 'representative',
          content: result.text,
          localRuneCount: characterCount,
        })
      }

      const addedRunes = newMaterials.reduce(
        (sum, material) => sum + (material.localRuneCount || 0),
        0,
      )

      if (localTotalRunes + addedRunes > MAX_TOTAL_RUNES) {
        throw new Error(
          `本地材料合计不能超过${MAX_TOTAL_RUNES}个字符，请减少材料`,
        )
      }

      setMaterials(previous => [...previous, ...newMaterials])
      invalidateProfile()
    } catch (error) {
      setErrorMsg(
        error instanceof Error
          ? error.message
          : '文件解析失败',
      )
    } finally {
      setFileBusy(false)
    }
  }

  const handleAddPaste = () => {
    const content = pasteText.trim()

    if (!content) {
      setErrorMsg('请先粘贴教案或教研材料')
      return
    }

    const characterCount = runeLength(content)

    if (characterCount > MAX_LOCAL_RUNES) {
      setErrorMsg(
        `粘贴内容共${characterCount}个字符，`
        + `单份材料最多${MAX_LOCAL_RUNES}个字符`,
      )
      return
    }

    if (localTotalRunes + characterCount > MAX_TOTAL_RUNES) {
      setErrorMsg(
        `本地材料合计不能超过${MAX_TOTAL_RUNES}个字符`,
      )
      return
    }

    const added = addMaterial({
      key: makeKey(),
      title: pasteTitle.trim() || '粘贴的教学材料',
      source_type: 'pasted',
      intent: 'local_standard',
      content,
      localRuneCount: characterCount,
    })

    if (added) {
      setPasteTitle('')
      setPasteText('')
    }
  }

  const updateIntent = (
    key: string,
    intent: StyleProfileIntent,
  ) => {
    setMaterials(previous => previous.map(material => (
      material.key === key
        ? { ...material, intent }
        : material
    )))
    invalidateProfile()
  }

  const removeMaterial = (key: string) => {
    setMaterials(previous => (
      previous.filter(material => material.key !== key)
    ))
    invalidateProfile()
  }

  const handleGenerate = async () => {
    if (materials.length === 0) {
      setErrorMsg('请至少添加一份教案或教研材料')
      return
    }

    setGenerating(true)
    setErrorMsg('')
    setProfileResult(null)

    try {
      const response = await analyzeAssistantStyleProfile({
        subject: subject || undefined,
        grade: grade || undefined,
        materials: materials.map(material => ({
          title: material.title,
          source_type: material.source_type,
          source_id: material.source_id,
          intent: material.intent,
          content: material.content,
        })),
      })

      setProfileResult(response)
      setProfileText(response.profile_markdown || '')
    } catch (error) {
      setErrorMsg(
        error instanceof Error
          ? error.message
          : '生成教学风格画像失败',
      )
    } finally {
      setGenerating(false)
    }
  }

  const handleUseProfile = () => {
    const profile = profileText.trim()

    if (!profile) {
      setErrorMsg('画像内容为空，无法继续生成助手')
      return
    }

    onUseProfile(profile)
  }

  return (
    <div
      onClick={busy ? undefined : onClose}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 10020,
        background: 'rgba(17,24,39,0.52)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '20px',
      }}
    >
      <div
        onClick={event => event.stopPropagation()}
        style={{
          width: '1060px',
          maxWidth: '100%',
          maxHeight: '92vh',
          background: C.card,
          borderRadius: '16px',
          boxShadow: '0 28px 72px rgba(0,0,0,0.2)',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        <header
          style={{
            padding: '16px 22px',
            borderBottom: `1px solid ${C.border}`,
            background:
              'linear-gradient(135deg,rgba(79,123,232,0.08),rgba(16,185,129,0.05))',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            flexShrink: 0,
          }}
        >
          <div>
            <div
              style={{
                fontSize: '17px',
                fontWeight: 700,
                color: C.text,
              }}
            >
              📚 从我的教案生成成长型助手
            </div>

            <div
              style={{
                marginTop: '4px',
                fontSize: '12px',
                color: C.textSec,
                lineHeight: 1.6,
              }}
            >
              AI提炼稳定优势、教研要求和成长方向，不机械复刻具体课例。
            </div>
          </div>

          <button
            type="button"
            onClick={() => {
              if (!busy) onClose()
            }}
            disabled={busy}
            style={{
              border: 'none',
              background: 'transparent',
              color: C.textMuted,
              fontSize: '22px',
              cursor: busy ? 'not-allowed' : 'pointer',
            }}
          >
            ×
          </button>
        </header>

        <main
          style={{
            flex: 1,
            minHeight: 0,
            overflow: 'auto',
            padding: '18px 22px 22px',
          }}
        >
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: profileResult
                ? 'minmax(0,0.92fr) minmax(0,1.08fr)'
                : '1fr',
              gap: '16px',
            }}
          >
            <div style={{ minWidth: 0 }}>
              <div
                style={{
                  padding: '11px 13px',
                  marginBottom: '14px',
                  borderRadius: '10px',
                  background: C.primaryLight,
                  color: C.textSec,
                  fontSize: '12px',
                  lineHeight: 1.7,
                }}
              >
                <b style={{ color: C.primary }}>任选一种方式添加材料即可。</b>
                {' '}
                不需要把平台选择、上传文件和粘贴文字全部填写。
                需要多份材料时，可以继续使用当前方式，或切换其他来源追加，
                最多共{MAX_MATERIALS}份。
                <div
                  style={{
                    marginTop: '4px',
                    color: C.textMuted,
                  }}
                >
                  当前范围：
                  {subject || '不限学科'}
                  {' · '}
                  {grade || '不限学段'}
                </div>
              </div>

              <section style={sectionStyle}>
                <div style={sectionTitleStyle}>
                  选择材料来源
                </div>

                <div
                  style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(3,minmax(0,1fr))',
                    gap: '8px',
                  }}
                >
                  {SOURCE_TABS.map(tab => {
                    const active = sourceMode === tab.value

                    return (
                      <button
                        key={tab.value}
                        type="button"
                        onClick={() => {
                          setSourceMode(tab.value)
                          setErrorMsg('')
                        }}
                        disabled={busy}
                        style={{
                          padding: '10px 9px',
                          borderRadius: '9px',
                          border: `1px solid ${
                            active ? C.primary : C.border
                          }`,
                          background: active
                            ? C.primaryLight
                            : C.card,
                          color: active
                            ? C.primary
                            : C.textSec,
                          textAlign: 'left',
                          cursor: busy
                            ? 'not-allowed'
                            : 'pointer',
                        }}
                      >
                        <div
                          style={{
                            fontSize: '12px',
                            fontWeight: 700,
                          }}
                        >
                          {tab.icon} {tab.label}
                        </div>

                        <div
                          style={{
                            marginTop: '3px',
                            fontSize: '10px',
                            lineHeight: 1.5,
                            color: C.textMuted,
                          }}
                        >
                          {tab.description}
                        </div>
                      </button>
                    )
                  })}
                </div>
              </section>

              <section style={sectionStyle}>
                {sourceMode === 'platform' && (
                  <>
                    <div style={sectionTitleStyle}>
                      从平台选择一份自己的教案
                    </div>

                    {plansLoading ? (
                      <div style={mutedStyle}>
                        正在读取我的教案…
                      </div>
                    ) : plansError ? (
                      <div
                        style={{
                          ...mutedStyle,
                          color: C.danger,
                        }}
                      >
                        ⚠️ {plansError}
                      </div>
                    ) : plans.length === 0 ? (
                      <div style={mutedStyle}>
                        当前学科下暂无可选教案，可以切换到上传或粘贴。
                      </div>
                    ) : (
                      <div
                        style={{
                          display: 'flex',
                          gap: '8px',
                        }}
                      >
                        <select
                          value={selectedPlanID}
                          onChange={event => {
                            setSelectedPlanID(event.target.value)
                          }}
                          disabled={busy}
                          style={{
                            ...inputStyle,
                            flex: 1,
                            minWidth: 0,
                          }}
                        >
                          {plans.map(plan => (
                            <option
                              key={plan.id}
                              value={plan.id}
                            >
                              {plan.title || plan.topic}
                              {plan.grade
                                ? ` · ${plan.grade}`
                                : ''}
                            </option>
                          ))}
                        </select>

                        <button
                          type="button"
                          onClick={handleAddPlatformPlan}
                          disabled={busy}
                          style={secondaryButtonStyle}
                        >
                          添加这份教案
                        </button>
                      </div>
                    )}
                  </>
                )}

                {sourceMode === 'file' && (
                  <>
                    <div style={sectionTitleStyle}>
                      上传 Word 或文字版 PDF
                    </div>

                    <button
                      type="button"
                      onClick={() => {
                        fileInputRef.current?.click()
                      }}
                      disabled={busy}
                      style={{
                        ...secondaryButtonStyle,
                        width: '100%',
                        padding: '11px',
                      }}
                    >
                      {fileBusy
                        ? '⏳ 正在提取文字…'
                        : '📄 选择 .docx 或 .pdf 文件'}
                    </button>

                    <input
                      ref={fileInputRef}
                      type="file"
                      accept=".docx,.pdf"
                      multiple
                      onChange={handleFiles}
                      style={{ display: 'none' }}
                    />

                    <div
                      style={{
                        ...mutedStyle,
                        marginTop: '7px',
                      }}
                    >
                      文件只在浏览器中提取文字，原始文件不会上传或保存。
                    </div>
                  </>
                )}

                {sourceMode === 'paste' && (
                  <>
                    <div style={sectionTitleStyle}>
                      粘贴一份教案或教研材料
                    </div>

                    <input
                      value={pasteTitle}
                      onChange={event => {
                        setPasteTitle(event.target.value)
                      }}
                      placeholder="例如：七年级学科教研要求"
                      disabled={busy}
                      style={{
                        ...inputStyle,
                        width: '100%',
                        marginBottom: '8px',
                      }}
                    />

                    <textarea
                      value={pasteText}
                      onChange={event => {
                        setPasteText(event.target.value)
                      }}
                      placeholder="粘贴教案、学校教研要求、评课意见或反面样例……"
                      rows={6}
                      disabled={busy}
                      style={{
                        ...inputStyle,
                        width: '100%',
                        resize: 'vertical',
                        lineHeight: 1.6,
                        fontFamily: 'inherit',
                      }}
                    />

                    <div
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        gap: '8px',
                        marginTop: '7px',
                      }}
                    >
                      <span style={mutedStyle}>
                        {runeLength(pasteText).toLocaleString()}
                        {' / '}
                        {MAX_LOCAL_RUNES.toLocaleString()}
                        个字符
                      </span>

                      <button
                        type="button"
                        onClick={handleAddPaste}
                        disabled={busy || !pasteText.trim()}
                        style={secondaryButtonStyle}
                      >
                        添加这份文字
                      </button>
                    </div>
                  </>
                )}
              </section>

              <section style={sectionStyle}>
                <div
                  style={{
                    ...sectionTitleStyle,
                    display: 'flex',
                    justifyContent: 'space-between',
                    gap: '8px',
                  }}
                >
                  <span>
                    已选材料（{materials.length}/{MAX_MATERIALS}）
                  </span>

                  <span
                    style={{
                      fontSize: '11px',
                      fontWeight: 400,
                      color: C.textMuted,
                    }}
                  >
                    本地文字
                    {' '}
                    {localTotalRunes.toLocaleString()}
                    {' / '}
                    {MAX_TOTAL_RUNES.toLocaleString()}
                  </span>
                </div>

                {materials.length === 0 ? (
                  <div
                    style={{
                      padding: '20px',
                      textAlign: 'center',
                      border: `1px dashed ${C.border}`,
                      borderRadius: '8px',
                      color: C.textMuted,
                      fontSize: '12px',
                      lineHeight: 1.7,
                    }}
                  >
                    尚未添加材料。
                    <br />
                    从上方三个来源中任选一种添加即可。
                  </div>
                ) : (
                  <div
                    style={{
                      display: 'flex',
                      flexDirection: 'column',
                      gap: '8px',
                    }}
                  >
                    {materials.map(material => (
                      <div
                        key={material.key}
                        style={{
                          padding: '10px',
                          border: `1px solid ${C.border}`,
                          borderRadius: '9px',
                          background: C.bg,
                        }}
                      >
                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'flex-start',
                            gap: '8px',
                          }}
                        >
                          <div
                            style={{
                              flex: 1,
                              minWidth: 0,
                            }}
                          >
                            <div
                              style={{
                                fontSize: '12px',
                                fontWeight: 650,
                                color: C.text,
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                              }}
                              title={material.title}
                            >
                              {material.title}
                            </div>

                            <div
                              style={{
                                marginTop: '3px',
                                fontSize: '10px',
                                color: C.textMuted,
                              }}
                            >
                              {sourceLabel(material.source_type)}
                              {material.localRuneCount
                                ? ` · ${material.localRuneCount.toLocaleString()}字符`
                                : ' · 正文由后端安全读取'}
                            </div>
                          </div>

                          <button
                            type="button"
                            onClick={() => {
                              removeMaterial(material.key)
                            }}
                            disabled={busy}
                            style={{
                              border: 'none',
                              background: 'transparent',
                              color: C.danger,
                              cursor: busy
                                ? 'not-allowed'
                                : 'pointer',
                              fontSize: '12px',
                            }}
                          >
                            删除
                          </button>
                        </div>

                        <select
                          value={material.intent}
                          onChange={event => {
                            updateIntent(
                              material.key,
                              event.target.value as StyleProfileIntent,
                            )
                          }}
                          disabled={busy}
                          style={{
                            ...inputStyle,
                            width: '100%',
                            marginTop: '8px',
                            fontSize: '12px',
                          }}
                        >
                          {INTENT_OPTIONS.map(option => (
                            <option
                              key={option.value}
                              value={option.value}
                            >
                              {option.label} — {option.hint}
                            </option>
                          ))}
                        </select>
                      </div>
                    ))}
                  </div>
                )}
              </section>

              {errorMsg && (
                <div
                  style={{
                    padding: '10px 12px',
                    marginBottom: '12px',
                    borderRadius: '8px',
                    background: 'rgba(239,68,68,0.06)',
                    border: '1px solid rgba(239,68,68,0.2)',
                    color: C.danger,
                    fontSize: '12px',
                    lineHeight: 1.6,
                  }}
                >
                  ⚠️ {errorMsg}
                </div>
              )}

              <button
                type="button"
                onClick={handleGenerate}
                disabled={busy || materials.length === 0}
                style={{
                  width: '100%',
                  padding: '11px',
                  border: 'none',
                  borderRadius: '9px',
                  background:
                    busy || materials.length === 0
                      ? C.border
                      : C.primary,
                  color:
                    busy || materials.length === 0
                      ? C.textMuted
                      : '#fff',
                  fontSize: '13px',
                  fontWeight: 700,
                  cursor:
                    busy || materials.length === 0
                      ? 'not-allowed'
                      : 'pointer',
                }}
              >
                {generating
                  ? '🧠 AI正在分析教学风格与成长方向…'
                  : '✨ 生成教学风格与成长画像'}
              </button>
            </div>

            {profileResult && (
              <section
                style={{
                  minWidth: 0,
                  display: 'flex',
                  flexDirection: 'column',
                  border: `1px solid ${C.border}`,
                  borderRadius: '12px',
                  overflow: 'hidden',
                  background: C.card,
                }}
              >
                <div
                  style={{
                    padding: '12px 14px',
                    borderBottom: `1px solid ${C.border}`,
                    background:
                      'linear-gradient(135deg,rgba(16,185,129,0.08),rgba(79,123,232,0.04))',
                  }}
                >
                  <div
                    style={{
                      fontSize: '14px',
                      fontWeight: 700,
                      color: C.text,
                    }}
                  >
                    🧭 教学风格与成长画像
                  </div>

                  <div
                    style={{
                      marginTop: '4px',
                      fontSize: '11px',
                      color: C.textSec,
                    }}
                  >
                    {profileResult.material_count}份材料
                    {' · '}
                    {profileResult.total_characters.toLocaleString()}
                    个字符
                    {' · '}
                    {confidenceLabel(profileResult.confidence)}
                  </div>
                </div>

                {profileResult.warnings?.length > 0 && (
                  <div
                    style={{
                      padding: '9px 12px',
                      background: 'rgba(245,158,11,0.08)',
                      borderBottom:
                        '1px solid rgba(245,158,11,0.2)',
                      color: '#92400E',
                      fontSize: '11px',
                      lineHeight: 1.6,
                    }}
                  >
                    {profileResult.warnings.map((warning, index) => (
                      <div key={index}>
                        ⚠️ {warning}
                      </div>
                    ))}
                  </div>
                )}

                <textarea
                  value={profileText}
                  onChange={event => {
                    setProfileText(event.target.value)
                  }}
                  style={{
                    flex: 1,
                    minHeight: '520px',
                    padding: '14px',
                    border: 'none',
                    outline: 'none',
                    resize: 'vertical',
                    fontSize: '12px',
                    lineHeight: 1.8,
                    fontFamily: 'inherit',
                    color: C.text,
                    background: C.bg,
                  }}
                />

                <div
                  style={{
                    padding: '11px 12px',
                    borderTop: `1px solid ${C.border}`,
                  }}
                >
                  <button
                    type="button"
                    onClick={handleUseProfile}
                    disabled={!profileText.trim()}
                    style={{
                      width: '100%',
                      padding: '10px',
                      border: 'none',
                      borderRadius: '8px',
                      background: profileText.trim()
                        ? C.success
                        : C.border,
                      color: profileText.trim()
                        ? '#fff'
                        : C.textMuted,
                      fontSize: '13px',
                      fontWeight: 700,
                      cursor: profileText.trim()
                        ? 'pointer'
                        : 'not-allowed',
                    }}
                  >
                    ✓ 确认画像，交给AI继续生成助手
                  </button>

                  <div
                    style={{
                      marginTop: '5px',
                      textAlign: 'center',
                      color: C.textMuted,
                      fontSize: '10px',
                    }}
                  >
                    画像会进入现有对话画布，仍可继续讨论和修改。
                  </div>
                </div>
              </section>
            )}
          </div>
        </main>

        <footer
          style={{
            padding: '12px 20px',
            borderTop: `1px solid ${C.border}`,
            display: 'flex',
            justifyContent: 'flex-end',
            background: C.bg,
            flexShrink: 0,
          }}
        >
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            style={{
              ...secondaryButtonStyle,
              padding: '8px 18px',
            }}
          >
            {busy ? '处理中…' : '关闭'}
          </button>
        </footer>
      </div>
    </div>
  )
}

const sectionStyle: CSSProperties = {
  padding: '12px',
  marginBottom: '12px',
  border: `1px solid ${C.border}`,
  borderRadius: '10px',
  background: C.card,
}

const sectionTitleStyle: CSSProperties = {
  marginBottom: '9px',
  fontSize: '12px',
  fontWeight: 700,
  color: C.text,
}

const mutedStyle: CSSProperties = {
  fontSize: '11px',
  color: C.textMuted,
  lineHeight: 1.6,
}

const inputStyle: CSSProperties = {
  boxSizing: 'border-box',
  padding: '8px 10px',
  border: `1px solid ${C.border}`,
  borderRadius: '7px',
  background: '#fff',
  color: C.text,
  fontSize: '13px',
  outline: 'none',
}

const secondaryButtonStyle: CSSProperties = {
  padding: '8px 13px',
  border: `1px solid ${C.border}`,
  borderRadius: '7px',
  background: '#fff',
  color: C.primary,
  fontSize: '12px',
  fontWeight: 600,
  cursor: 'pointer',
}
