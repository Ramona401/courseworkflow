/**
 * AssistantEditModal.tsx — AI助手新建与编辑弹窗
 *
 * 支持三种模式：
 *   - create-personal：新建个人助手；
 *   - create-group：新建共享助手；
 *   - edit：编辑现有助手。
 *
 * 教育域适配：
 *   - 新建助手使用当前登录用户教育域；
 *   - 编辑助手优先使用助手自身education_domain资源快照；
 *   - K12、职业教育、成人教育使用同一套层级工具；
 *   - 具体层级可参与自动匹配；
 *   - 学段和不限值只供手动选择；
 *   - 职一、职二、职三保存为中职规范值。
 *
 * Prompt编辑区包含：
 *   - 手动编辑；
 *   - AI帮我写。
 */

import {
  useState,
  useEffect,
  useCallback,
  useRef,
  useMemo,
} from 'react'
import type {
  Dispatch,
  SetStateAction,
} from 'react'
import { useAuth } from '@/store/auth'
import {
  useEducationProfile,
} from '@/hooks/useEducationProfile'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import { useSubjects } from '@/hooks/useSubjects'
import {
  fallbackEducationProfile,
  type EducationDomain,
} from '@/education-domain/types'
import {
  getAssistantLevelOptions,
  getEducationLevelLabel,
  isAutomaticEducationLevel,
  normalizeEducationLevelValue,
} from '@/education-domain/options'
import {
  getAssistant,
  createAssistant,
  updateAssistant,
  parseAssistantScenes,
  ASSISTANT_SCENE_LABELS,
  DEFAULT_SHARE_POLICY,
  type AssistantScene,
  type AssistantSource,
  type AssistantSharePolicy,
  type CreateAIAssistantRequest,
  type UpdateAIAssistantRequest,
} from '@/api/ai-assistants'
import AssistantDesignerPanel from './AssistantDesignerPanel'
import SharePolicyPicker from './SharePolicyPicker'
import {
  C,
  QUICK_EMOJIS,
  MAX_PROMPT_LEN,
  labelStyle,
  inputStyle,
} from './editModalStyles'

type EditTab =
  | 'manual'
  | 'designer'

const WORKSHOP_RUNTIME_PROMPT_LEN =
  8000

const VALID_SHARE_POLICIES:
  AssistantSharePolicy[] = [
    'use_only',
    'open',
    'locked',
  ]

/**
 * 资源教育域只允许三个具体教学域参与助手表单。
 *
 * system/common/mixed等其它值使用当前管理页面教育域作展示兜底，
 * 真正的权限与资源隔离仍由后端执行。
 */
function resolveAssistantFormDomain(
  value: unknown,
  fallback: EducationDomain,
): EducationDomain {
  if (
    value === 'k12' ||
    value === 'vocational' ||
    value === 'adult'
  ) {
    return value
  }

  return fallback
}

function normalizeAssistantSubject(
  value: string | undefined,
  subjectOptions: string[],
): string {
  const trimmed =
    (value || '').trim()

  return subjectOptions.includes(
    trimmed,
  )
    ? trimmed
    : ''
}

function assistantGradeHint(
  domain: EducationDomain,
  gradeLabel: string,
  grade: string,
): string {
  const displayLabel =
    getEducationLevelLabel(
      domain,
      grade,
    )

  if (
    isAutomaticEducationLevel(
      domain,
      grade,
    )
  ) {
    return `${displayLabel}属于具体${gradeLabel}。课程、具体层级和阶段严格一致时，可被平台自动匹配。`
  }

  return `${displayLabel || '不限层级'}只供老师手动选择，不会被平台自动挂载。`
}

/* ==================== 表单草稿 ==================== */

interface AssistantEditFormDraft {
  name: string
  avatar: string
  description: string
  subject: string
  gradeRange: string
  scenes: AssistantScene[]
  fullPrompt: string
  sharePolicy: AssistantSharePolicy
}

function createAssistantEditInitialForm(
  defaultScene: AssistantScene | undefined,
  defaultSubject: string | undefined,
  defaultGrade: string | undefined,
  subjectOptions: string[],
  domain: EducationDomain,
): AssistantEditFormDraft {
  return {
    name: '',
    avatar: '🤖',
    description: '',
    subject:
      normalizeAssistantSubject(
        defaultSubject,
        subjectOptions,
      ),
    gradeRange:
      normalizeEducationLevelValue(
        domain,
        defaultGrade,
      ),
    scenes:
      defaultScene
        ? [defaultScene]
        : [],
    fullPrompt: '',
    sharePolicy:
      DEFAULT_SHARE_POLICY,
  }
}

function parseAssistantEditForm(
  raw: string,
  fallback: AssistantEditFormDraft,
): AssistantEditFormDraft {
  if (!raw.trim()) {
    return {
      ...fallback,
      scenes: [...fallback.scenes],
    }
  }

  try {
    const parsed =
      JSON.parse(raw) as
        Partial<AssistantEditFormDraft>

    const scenes =
      Array.isArray(parsed.scenes)
        ? parsed.scenes.filter(
            (
              scene,
            ): scene is AssistantScene =>
              typeof scene ===
                'string' &&
              scene in
                ASSISTANT_SCENE_LABELS,
          )
        : [...fallback.scenes]

    const sharePolicy =
      VALID_SHARE_POLICIES.includes(
        parsed.sharePolicy as
          AssistantSharePolicy,
      )
        ? parsed.sharePolicy as
            AssistantSharePolicy
        : fallback.sharePolicy

    return {
      name:
        typeof parsed.name ===
          'string'
          ? parsed.name
          : fallback.name,
      avatar:
        typeof parsed.avatar ===
          'string'
          ? parsed.avatar
          : fallback.avatar,
      description:
        typeof parsed.description ===
          'string'
          ? parsed.description
          : fallback.description,
      subject:
        typeof parsed.subject ===
          'string'
          ? parsed.subject
          : fallback.subject,
      gradeRange:
        typeof parsed.gradeRange ===
          'string'
          ? parsed.gradeRange
          : fallback.gradeRange,
      scenes,
      fullPrompt:
        typeof parsed.fullPrompt ===
          'string'
          ? parsed.fullPrompt
          : fallback.fullPrompt,
      sharePolicy,
    }
  } catch {
    return {
      ...fallback,
      scenes: [...fallback.scenes],
    }
  }
}

/* ==================== Props ==================== */

export type AssistantEditMode =
  | 'create-personal'
  | 'create-group'
  | 'edit'

export interface AssistantEditModalProps {
  open: boolean
  mode: AssistantEditMode
  assistantId?: string
  defaultScene?: AssistantScene
  defaultSubject?: string
  defaultGrade?: string
  onClose: () => void
  onSaved?: (
    id: string,
    source: AssistantSource,
  ) => void
}

/* ==================== 主组件 ==================== */

export default function AssistantEditModal(
  props: AssistantEditModalProps,
) {
  const {
    open,
    mode,
    assistantId,
    defaultScene,
    defaultSubject,
    defaultGrade,
    onClose,
    onSaved,
  } = props

  const { user } = useAuth()

  const {
    domain: currentDomain,
    profile: currentProfile,
  } = useEducationProfile()

  const {
    subjects: subjectOptions,
    loading: subjectsLoading,
    empty: subjectsEmpty,
  } = useSubjects()

  /**
   * 编辑模式优先使用资源自身教育域。
   *
   * 这样平台管理员编辑职业教育助手时，
   * 不会因为管理页面是mixed而只看到K12年级。
   */
  const [
    loadedEducationDomain,
    setLoadedEducationDomain,
  ] = useState<
    EducationDomain | null
  >(null)

  const effectiveDomain =
    loadedEducationDomain ||
    currentDomain

  const effectiveProfile =
    effectiveDomain ===
      currentDomain
      ? currentProfile
      : fallbackEducationProfile(
          effectiveDomain,
        )

  const levelOptions = useMemo(
    () => getAssistantLevelOptions(
      effectiveDomain,
    ),
    [effectiveDomain],
  )

  const specificLevelOptions =
    useMemo(
      () => levelOptions.filter(
        option => option.automatic,
      ),
      [levelOptions],
    )

  const broadLevelOptions =
    useMemo(
      () => levelOptions.filter(
        option => !option.automatic,
      ),
      [levelOptions],
    )

  const levelValues = useMemo(
    () => levelOptions.map(
      option => option.value,
    ),
    [levelOptions],
  )

  const [
    serverInitialForm,
    setServerInitialForm,
  ] = useState<
    AssistantEditFormDraft
  >(() =>
    createAssistantEditInitialForm(
      defaultScene,
      defaultSubject,
      defaultGrade,
      subjectOptions,
      currentDomain,
    ),
  )

  const formResourceID =
    mode === 'edit'
      ? `edit:${assistantId || 'missing'}`
      : [
          mode,
          currentDomain,
          defaultScene ||
            'no-scene',
          (defaultSubject || '')
            .trim() ||
            'no-subject',
          normalizeEducationLevelValue(
            currentDomain,
            defaultGrade,
          ) ||
            'no-grade',
        ].join('|')

  const formDraft =
    useProtectedDraft({
      userId: user?.id,
      scope: 'assistant-edit',
      resourceId:
        formResourceID,
      field: 'form',
      initialValue:
        JSON.stringify(
          serverInitialForm,
        ),
      maxHistory: 20,
    })

  const form =
    parseAssistantEditForm(
      formDraft.value,
      serverInitialForm,
    )

  const setForm: Dispatch<
    SetStateAction<
      AssistantEditFormDraft
    >
  > = useCallback(
    next => {
      formDraft.setValue(
        previousText => {
          const previous =
            parseAssistantEditForm(
              previousText,
              serverInitialForm,
            )

          const resolved =
            typeof next ===
              'function'
              ? next(previous)
              : next

          return JSON.stringify(
            resolved,
          )
        },
      )
    },
    [
      formDraft.setValue,
      serverInitialForm,
    ],
  )

  const {
    name,
    avatar,
    description,
    subject,
    gradeRange,
    scenes,
    fullPrompt,
    sharePolicy,
  } = form

  const setName = (
    value: string,
  ) =>
    setForm(previous => ({
      ...previous,
      name: value,
    }))

  const setAvatar = (
    value: string,
  ) =>
    setForm(previous => ({
      ...previous,
      avatar: value,
    }))

  const setDescription = (
    value: string,
  ) =>
    setForm(previous => ({
      ...previous,
      description: value,
    }))

  const setSubject = (
    value: string,
  ) =>
    setForm(previous => ({
      ...previous,
      subject: value,
    }))

  const setGradeRange = (
    value: string,
  ) =>
    setForm(previous => ({
      ...previous,
      gradeRange: value,
    }))

  const setFullPrompt = (
    value: string,
  ) =>
    setForm(previous => ({
      ...previous,
      fullPrompt: value,
    }))

  const setSharePolicy = (
    value: AssistantSharePolicy,
  ) =>
    setForm(previous => ({
      ...previous,
      sharePolicy: value,
    }))

  const setScenes: Dispatch<
    SetStateAction<
      AssistantScene[]
    >
  > = next =>
    setForm(previous => ({
      ...previous,
      scenes:
        typeof next ===
          'function'
          ? next(
              previous.scenes,
            )
          : next,
    }))

  const promptChars =
    Array.from(
      fullPrompt,
    ).length

  /* ==================== 存量异常提示 ==================== */

  const [
    legacySubjectValue,
    setLegacySubjectValue,
  ] = useState('')

  const [
    legacyGradeValue,
    setLegacyGradeValue,
  ] = useState('')

  const [
    loadedSource,
    setLoadedSource,
  ] = useState<
    AssistantSource | null
  >(null)

  /* ==================== UI状态 ==================== */

  const [loading, setLoading] =
    useState(false)

  const [saving, setSaving] =
    useState(false)

  const [loadErr, setLoadErr] =
    useState<string | null>(null)

  const [activeTab, setActiveTab] =
    useState<EditTab>('manual')

  const promptRef =
    useRef<HTMLTextAreaElement>(
      null,
    )

  /* ==================== 重置新建表单 ==================== */

  const resetForm =
    useCallback(() => {
      const rawSubject =
        (defaultSubject || '').trim()

      const normalizedSubject =
        normalizeAssistantSubject(
          rawSubject,
          subjectOptions,
        )

      const rawGrade =
        (defaultGrade || '').trim()

      const normalizedGrade =
        normalizeEducationLevelValue(
          currentDomain,
          rawGrade,
        )

      setLoadedEducationDomain(
        currentDomain,
      )

      setServerInitialForm(
        createAssistantEditInitialForm(
          defaultScene,
          defaultSubject,
          defaultGrade,
          subjectOptions,
          currentDomain,
        ),
      )

      setLegacySubjectValue(
        rawSubject &&
        !normalizedSubject
          ? rawSubject
          : '',
      )

      setLegacyGradeValue(
        rawGrade &&
        !normalizedGrade
          ? rawGrade
          : '',
      )

      setLoadedSource(null)
      setLoadErr(null)
      setActiveTab('manual')
    }, [
      defaultScene,
      defaultSubject,
      defaultGrade,
      subjectOptions,
      currentDomain,
    ])

  /* ==================== 打开与加载详情 ==================== */

  useEffect(() => {
    if (
      !open ||
      subjectsLoading
    ) {
      return
    }

    if (
      mode === 'edit' &&
      assistantId
    ) {
      setLoading(true)
      setLoadErr(null)
      setActiveTab('manual')

      getAssistant(
        assistantId,
      )
        .then(data => {
          const resourceDomain =
            resolveAssistantFormDomain(
              (
                data as
                  typeof data & {
                    education_domain?: string
                  }
              ).education_domain,
              currentDomain,
            )

          const rawSubject =
            (data.subject || '')
              .trim()

          const normalizedSubject =
            normalizeAssistantSubject(
              rawSubject,
              subjectOptions,
            )

          const rawGrade =
            (data.grade_range || '')
              .trim()

          const normalizedGrade =
            normalizeEducationLevelValue(
              resourceDomain,
              rawGrade,
            )

          setLoadedEducationDomain(
            resourceDomain,
          )

          setServerInitialForm({
            name:
              data.name || '',
            avatar:
              data.avatar_emoji ||
              '🤖',
            description:
              data.description ||
              '',
            subject:
              normalizedSubject,
            gradeRange:
              normalizedGrade,
            scenes:
              parseAssistantScenes(
                data.scenes,
              ),
            fullPrompt:
              data.full_prompt || '',
            sharePolicy:
              data.share_policy ||
              DEFAULT_SHARE_POLICY,
          })

          setLegacySubjectValue(
            rawSubject &&
            !normalizedSubject
              ? rawSubject
              : '',
          )

          setLegacyGradeValue(
            rawGrade &&
            !normalizedGrade
              ? rawGrade
              : '',
          )

          setLoadedSource(
            data.source,
          )

          setLoading(false)
        })
        .catch(error => {
          setLoadErr(
            error instanceof Error
              ? error.message
              : '加载助手详情失败',
          )
          setLoading(false)
        })
    } else {
      resetForm()
    }
  }, [
    open,
    mode,
    assistantId,
    resetForm,
    subjectsLoading,
    subjectOptions,
    currentDomain,
  ])

  /**
   * 新建模式或恢复本地草稿时，确保层级属于当前有效教育域。
   *
   * 编辑模式的服务器值已按资源教育域规范化；
   * 无法识别的存量值保留在legacyGradeValue中提示重新选择。
   */
  useEffect(() => {
    if (
      !open ||
      loading ||
      levelOptions.length === 0
    ) {
      return
    }

    setForm(current => {
      const raw =
        current.gradeRange.trim()

      if (
        raw === '' &&
        levelValues.includes('')
      ) {
        return current
      }

      const normalized =
        normalizeEducationLevelValue(
          effectiveDomain,
          raw,
        )

      const nextGrade =
        normalized &&
        levelValues.includes(
          normalized,
        )
          ? normalized
          : specificLevelOptions[0]
              ?.value ||
            broadLevelOptions[0]
              ?.value ||
            ''

      if (
        nextGrade ===
        current.gradeRange
      ) {
        return current
      }

      return {
        ...current,
        gradeRange: nextGrade,
      }
    })
  }, [
    open,
    loading,
    effectiveDomain,
    levelOptions,
    levelValues,
    specificLevelOptions,
    broadLevelOptions,
    setForm,
  ])

  useEffect(() => {
    if (!open) return

    const handler = (
      event: KeyboardEvent,
    ) => {
      if (
        event.key === 'Escape'
      ) {
        onClose()
      }
    }

    document.addEventListener(
      'keydown',
      handler,
    )

    return () =>
      document.removeEventListener(
        'keydown',
        handler,
      )
  }, [
    open,
    onClose,
  ])

  const toggleScene = (
    scene: AssistantScene,
  ) => {
    setScenes(previous =>
      previous.includes(scene)
        ? previous.filter(
            item => item !== scene,
          )
        : [...previous, scene],
    )
  }

  const handleApplyDraft =
    useCallback((draft: string) => {
      setForm(previous => ({
        ...previous,
        fullPrompt: draft,
      }))

      setActiveTab('manual')
    }, [setForm])

  const showPolicyPicker =
    mode === 'create-group' ||
    (
      mode === 'edit' &&
      (
        loadedSource === 'group' ||
        loadedSource === 'system'
      )
    )

  /* ==================== 校验 ==================== */

  const validate =
    (): string | null => {
      if (!name.trim()) {
        return '请填写助手名称'
      }

      if (!subject.trim()) {
        return `请选择助手适用的具体${effectiveProfile.subject_label}`
      }

      if (
        !subjectOptions.includes(
          subject.trim(),
        )
      ) {
        return `适用${effectiveProfile.subject_label}必须从当前课程清单中选择`
      }

      if (
        !levelValues.includes(
          gradeRange.trim(),
        )
      ) {
        return `请选择当前教育域支持的${effectiveProfile.grade_label}或通用层级`
      }

      if (!fullPrompt.trim()) {
        return '请填写系统提示词；若在AI页签生成了草稿，请先应用到编辑'
      }

      if (scenes.length === 0) {
        return '请至少勾选一个适用场景'
      }

      if (
        fullPrompt.length >
        MAX_PROMPT_LEN
      ) {
        return `系统提示词过长（${fullPrompt.length}字符），上限${MAX_PROMPT_LEN}字符`
      }

      return null
    }

  /* ==================== 提交 ==================== */

  const handleSubmit = async () => {
    const validationError =
      validate()

    if (validationError) {
      alert(validationError)
      return
    }

    if (saving) return

    setSaving(true)

    try {
      if (
        mode === 'edit' &&
        assistantId
      ) {
        const request:
          UpdateAIAssistantRequest = {
          name: name.trim(),
          avatar_emoji:
            avatar || '🤖',
          description:
            description.trim(),
          full_prompt:
            fullPrompt,
          subject:
            subject.trim(),
          grade_range:
            gradeRange.trim(),
          scenes,
          ...(showPolicyPicker
            ? {
                share_policy:
                  sharePolicy,
              }
            : {}),
        }

        await updateAssistant(
          assistantId,
          request,
        )

        formDraft.clear()

        onSaved?.(
          assistantId,
          loadedSource ||
            'personal',
        )

        onClose()
      } else {
        const source:
          AssistantSource =
          mode ===
            'create-group'
            ? 'group'
            : 'personal'

        const request:
          CreateAIAssistantRequest = {
          name: name.trim(),
          avatar_emoji:
            avatar || '🤖',
          description:
            description.trim(),
          source,
          full_prompt:
            fullPrompt,
          subject:
            subject.trim(),
          grade_range:
            gradeRange.trim(),
          scenes,
          ...(showPolicyPicker
            ? {
                share_policy:
                  sharePolicy,
              }
            : {}),
        }

        const created =
          await createAssistant(
            request,
          )

        formDraft.clear()

        onSaved?.(
          created.id,
          created.source,
        )

        onClose()
      }
    } catch (error) {
      alert(
        error instanceof Error
          ? error.message
          : '保存失败，请重试',
      )
    } finally {
      setSaving(false)
    }
  }

  if (!open) return null

  const modalTitle =
    mode === 'edit'
      ? name
        ? `✏️ 编辑 — ${name}`
        : '✏️ 编辑助手'
      : mode === 'create-group'
        ? '🏫 新建共享助手'
        : '➕ 新建我的助手'

  return (
    <div
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        background:
          'rgba(17,24,39,0.5)',
        zIndex: 10000,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '24px',
      }}
    >
      <div
        onClick={event =>
          event.stopPropagation()
        }
        style={{
          width: '960px',
          maxWidth: '100%',
          maxHeight: '92vh',
          background: C.card,
          borderRadius: '12px',
          boxShadow:
            '0 24px 64px rgba(0,0,0,0.18)',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        <div style={{
          padding: '16px 20px',
          borderBottom:
            `1px solid ${C.border}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent:
            'space-between',
          background:
            'linear-gradient(135deg,rgba(79,123,232,0.06),rgba(129,140,248,0.04))',
          flexShrink: 0,
        }}>
          <span style={{
            fontSize: '15px',
            fontWeight: 700,
            color: C.text,
          }}>
            {modalTitle}
          </span>

          <button
            type="button"
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              fontSize: '20px',
              color: C.textMuted,
              padding: '0 4px',
              lineHeight: 1,
            }}
          >
            ×
          </button>
        </div>

        <div style={{
          flex: 1,
          overflow: 'auto',
          padding: '20px 24px',
        }}>
          {(loading ||
            subjectsLoading) && (
            <div style={{
              padding: '40px 0',
              textAlign: 'center',
              color: C.textMuted,
            }}>
              加载助手详情中...
            </div>
          )}

          {loadErr &&
           !loading && (
            <div style={{
              padding: '12px',
              borderRadius: '8px',
              background:
                'rgba(239,68,68,0.06)',
              border:
                '1px solid rgba(239,68,68,0.2)',
              color: C.danger,
              fontSize: '13px',
              marginBottom: '12px',
            }}>
              ⚠️ {loadErr}
            </div>
          )}

          {!loading &&
           !subjectsLoading &&
           !loadErr && (
            <>
              {/* 名称与图标 */}
              <div style={{
                marginBottom: '16px',
              }}>
                <label style={labelStyle}>
                  名称
                  {' '}
                  <span style={{
                    color: C.danger,
                  }}>
                    *
                  </span>
                </label>

                <div style={{
                  display: 'flex',
                  gap: '8px',
                }}>
                  <input
                    type="text"
                    value={avatar}
                    onChange={event =>
                      setAvatar(
                        event.target.value,
                      )
                    }
                    onKeyDown={event =>
                      formDraft.handleKeyDown(
                        event,
                      )
                    }
                    placeholder="🤖"
                    maxLength={4}
                    style={{
                      ...inputStyle,
                      width: '50px',
                      textAlign: 'center',
                      fontSize: '18px',
                    }}
                  />

                  <input
                    type="text"
                    value={name}
                    onChange={event =>
                      setName(
                        event.target.value,
                      )
                    }
                    onKeyDown={event =>
                      formDraft.handleKeyDown(
                        event,
                      )
                    }
                    placeholder="例如：实训任务设计助手"
                    maxLength={100}
                    style={{
                      ...inputStyle,
                      flex: 1,
                    }}
                  />
                </div>

                <div style={{
                  marginTop: '6px',
                  display: 'flex',
                  gap: '4px',
                  flexWrap: 'wrap',
                }}>
                  {QUICK_EMOJIS.map(
                    emoji => (
                      <button
                        key={emoji}
                        type="button"
                        onClick={() =>
                          setAvatar(
                            emoji,
                          )
                        }
                        style={{
                          width: '28px',
                          height: '28px',
                          borderRadius:
                            '6px',
                          border:
                            `1px solid ${
                              avatar ===
                              emoji
                                ? C.primary
                                : C.border
                            }`,
                          background:
                            avatar === emoji
                              ? C.primaryLight
                              : '#fff',
                          cursor:
                            'pointer',
                          fontSize:
                            '14px',
                        }}
                      >
                        {emoji}
                      </button>
                    ),
                  )}
                </div>
              </div>

              {/* 描述 */}
              <div style={{
                marginBottom: '16px',
              }}>
                <label style={labelStyle}>
                  描述
                  <span style={{
                    color: C.textMuted,
                    fontWeight: 400,
                  }}>
                    {' '}（选填）
                  </span>
                </label>

                <input
                  type="text"
                  value={description}
                  onChange={event =>
                    setDescription(
                      event.target.value,
                    )
                  }
                  onKeyDown={event =>
                    formDraft.handleKeyDown(
                      event,
                    )
                  }
                  placeholder="一句话说明助手的定位"
                  maxLength={500}
                  style={{
                    ...inputStyle,
                    width: '100%',
                  }}
                />
              </div>

              {/* 课程和层级 */}
              <div style={{
                display: 'flex',
                gap: '12px',
                marginBottom: '16px',
              }}>
                <div style={{ flex: 1 }}>
                  <label style={labelStyle}>
                    适用
                    {effectiveProfile.subject_label}
                    {' '}
                    <span style={{
                      color: C.danger,
                    }}>
                      *
                    </span>
                  </label>

                  <select
                    value={subject}
                    disabled={
                      subjectsLoading ||
                      subjectsEmpty
                    }
                    onChange={event => {
                      setSubject(
                        event.target.value,
                      )
                      setLegacySubjectValue('')
                    }}
                    style={{
                      ...inputStyle,
                      width: '100%',
                      cursor: 'pointer',
                    }}
                  >
                    <option value="">
                      请选择具体
                      {effectiveProfile.subject_label}
                    </option>

                    {subjectOptions.map(
                      option => (
                        <option
                          key={option}
                          value={option}
                        >
                          {option}
                        </option>
                      ),
                    )}
                  </select>

                  {subjectsEmpty &&
                   !subjectsLoading && (
                    <div style={{
                      marginTop: '6px',
                      fontSize: '11px',
                      lineHeight: 1.5,
                      color: C.danger,
                    }}>
                      当前组织尚未配置可用
                      {effectiveProfile.subject_label}。
                    </div>
                  )}

                  {legacySubjectValue && (
                    <div style={{
                      marginTop: '6px',
                      fontSize: '11px',
                      lineHeight: 1.5,
                      color: '#92400E',
                    }}>
                      ⚠️ 原适用
                      {effectiveProfile.subject_label}
                      “{legacySubjectValue}”
                      不在当前课程清单中，请重新选择。
                    </div>
                  )}
                </div>

                <div style={{ flex: 1 }}>
                  <label style={labelStyle}>
                    适用
                    {effectiveProfile.grade_label}
                  </label>

                  <select
                    value={gradeRange}
                    onChange={event => {
                      setGradeRange(
                        event.target.value,
                      )
                      setLegacyGradeValue('')
                    }}
                    style={{
                      ...inputStyle,
                      width: '100%',
                      cursor: 'pointer',
                    }}
                  >
                    <optgroup
                      label={`具体${effectiveProfile.grade_label}（可自动匹配）`}
                    >
                      {specificLevelOptions.map(
                        option => (
                          <option
                            key={
                              option.value
                            }
                            value={
                              option.value
                            }
                          >
                            {option.label}
                          </option>
                        ),
                      )}
                    </optgroup>

                    {broadLevelOptions.length >
                     0 && (
                      <optgroup label="通用层级（仅手动选择）">
                        {broadLevelOptions.map(
                          option => (
                            <option
                              key={
                                option.value ||
                                '__empty__'
                              }
                              value={
                                option.value
                              }
                            >
                              {option.label}
                            </option>
                          ),
                        )}
                      </optgroup>
                    )}
                  </select>

                  <div style={{
                    marginTop: '6px',
                    fontSize: '11px',
                    lineHeight: 1.55,
                    color:
                      isAutomaticEducationLevel(
                        effectiveDomain,
                        gradeRange,
                      )
                        ? '#166534'
                        : '#64748B',
                  }}>
                    {assistantGradeHint(
                      effectiveDomain,
                      effectiveProfile.grade_label,
                      gradeRange,
                    )}
                  </div>

                  {legacyGradeValue && (
                    <div style={{
                      marginTop: '6px',
                      fontSize: '11px',
                      lineHeight: 1.5,
                      color: '#92400E',
                    }}>
                      ⚠️ 原
                      {effectiveProfile.grade_label}
                      “{legacyGradeValue}”
                      不属于当前教育域，请重新选择。
                    </div>
                  )}
                </div>
              </div>

              {/* 场景 */}
              <div style={{
                marginBottom: '16px',
              }}>
                <label style={labelStyle}>
                  适用场景
                  {' '}
                  <span style={{
                    color: C.danger,
                  }}>
                    *
                  </span>
                </label>

                <div style={{
                  display: 'grid',
                  gridTemplateColumns:
                    'repeat(3, 1fr)',
                  gap: '6px',
                  marginTop: '4px',
                }}>
                  {(
                    Object.entries(
                      ASSISTANT_SCENE_LABELS,
                    ) as [
                      AssistantScene,
                      string,
                    ][]
                  ).map(
                    ([
                      scene,
                      label,
                    ]) => {
                      const checked =
                        scenes.includes(
                          scene,
                        )

                      return (
                        <label
                          key={scene}
                          style={{
                            display:
                              'flex',
                            alignItems:
                              'center',
                            gap: '6px',
                            padding:
                              '7px 10px',
                            borderRadius:
                              '6px',
                            border:
                              `1px solid ${
                                checked
                                  ? C.primary
                                  : C.border
                              }`,
                            background:
                              checked
                                ? C.primaryLight
                                : '#fff',
                            cursor:
                              'pointer',
                            fontSize:
                              '13px',
                            color: C.text,
                          }}
                        >
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={() =>
                              toggleScene(
                                scene,
                              )
                            }
                            style={{
                              cursor:
                                'pointer',
                              accentColor:
                                C.primary,
                            }}
                          />

                          {label}
                        </label>
                      )
                    },
                  )}
                </div>
              </div>

              {showPolicyPicker && (
                <SharePolicyPicker
                  value={sharePolicy}
                  onChange={
                    setSharePolicy
                  }
                />
              )}

              {/* Prompt编辑区 */}
              <div style={{
                marginBottom: '8px',
              }}>
                <div style={{
                  display: 'flex',
                  justifyContent:
                    'space-between',
                  alignItems: 'center',
                  marginBottom: '6px',
                }}>
                  <label style={{
                    ...labelStyle,
                    marginBottom: 0,
                  }}>
                    系统提示词 Prompt
                    {' '}
                    <span style={{
                      color: C.danger,
                    }}>
                      *
                    </span>
                  </label>

                  <span style={{
                    fontSize: '11px',
                    color:
                      fullPrompt.length >
                      MAX_PROMPT_LEN
                        ? C.danger
                        : promptChars >
                            WORKSHOP_RUNTIME_PROMPT_LEN
                          ? C.accent
                          : C.textMuted,
                  }}>
                    {promptChars.toLocaleString()}
                    {' '}
                    Unicode字符
                  </span>
                </div>

                <div style={{
                  display: 'flex',
                  gap: '4px',
                  marginBottom: '8px',
                  padding: '4px',
                  background: C.bg,
                  borderRadius: '8px',
                  border:
                    `1px solid ${C.border}`,
                  width: 'fit-content',
                }}>
                  <button
                    type="button"
                    onClick={() =>
                      setActiveTab(
                        'manual',
                      )
                    }
                    style={{
                      padding:
                        '6px 14px',
                      borderRadius:
                        '6px',
                      border: 'none',
                      background:
                        activeTab ===
                        'manual'
                          ? C.card
                          : 'transparent',
                      color:
                        activeTab ===
                        'manual'
                          ? C.primary
                          : C.textSec,
                      fontSize: '12px',
                      fontWeight:
                        activeTab ===
                        'manual'
                          ? 700
                          : 500,
                      cursor: 'pointer',
                    }}
                  >
                    📝 手动编辑
                  </button>

                  <button
                    type="button"
                    onClick={() =>
                      setActiveTab(
                        'designer',
                      )
                    }
                    style={{
                      padding:
                        '6px 14px',
                      borderRadius:
                        '6px',
                      border: 'none',
                      background:
                        activeTab ===
                        'designer'
                          ? C.card
                          : 'transparent',
                      color:
                        activeTab ===
                        'designer'
                          ? C.primary
                          : C.textSec,
                      fontSize: '12px',
                      fontWeight:
                        activeTab ===
                        'designer'
                          ? 700
                          : 500,
                      cursor: 'pointer',
                    }}
                  >
                    💬 AI帮我写
                  </button>
                </div>

                {activeTab ===
                 'manual' && (
                  <>
                    <textarea
                      ref={promptRef}
                      value={fullPrompt}
                      onChange={event =>
                        setFullPrompt(
                          event.target
                            .value,
                        )
                      }
                      onKeyDown={event =>
                        formDraft.handleKeyDown(
                          event,
                        )
                      }
                      placeholder="在此编写或粘贴完整系统提示词..."
                      rows={16}
                      style={{
                        width: '100%',
                        padding:
                          '10px 12px',
                        borderRadius:
                          '8px',
                        border:
                          `1px solid ${
                            fullPrompt.length >
                            MAX_PROMPT_LEN
                              ? C.danger
                              : C.border
                          }`,
                        fontSize:
                          '12px',
                        lineHeight: 1.6,
                        fontFamily:
                          'Menlo, Monaco, Consolas, monospace',
                        color: C.text,
                        outline: 'none',
                        boxSizing:
                          'border-box',
                        resize:
                          'vertical',
                        minHeight:
                          '280px',
                        maxHeight:
                          '500px',
                        background: C.bg,
                      }}
                    />

                    {promptChars >
                     WORKSHOP_RUNTIME_PROMPT_LEN && (
                      <div style={{
                        marginTop: '8px',
                        padding:
                          '9px 12px',
                        borderRadius:
                          '8px',
                        background:
                          'rgba(245,158,11,0.08)',
                        border:
                          '1px solid rgba(245,158,11,0.28)',
                        color:
                          '#92400E',
                        fontSize:
                          '12px',
                        lineHeight:
                          1.6,
                      }}>
                        ⚠️ 完整提示词会保存，但备课工坊每轮最多注入前
                        {' '}
                        <b>
                          {WORKSHOP_RUNTIME_PROMPT_LEN.toLocaleString()}
                        </b>
                        {' '}
                        个Unicode字符。
                      </div>
                    )}
                  </>
                )}

                {activeTab ===
                 'designer' && (
                  <AssistantDesignerPanel
                    subject={subject}
                    grade={gradeRange}
                    scenes={scenes}
                    initialDraft={
                      fullPrompt
                    }
                    onApplyDraft={
                      handleApplyDraft
                    }
                  />
                )}
              </div>
            </>
          )}
        </div>

        <div style={{
          padding: '12px 20px',
          borderTop:
            `1px solid ${C.border}`,
          display: 'flex',
          justifyContent: 'flex-end',
          gap: '8px',
          background: C.bg,
          flexShrink: 0,
        }}>
          <button
            type="button"
            onClick={onClose}
            disabled={saving}
            style={{
              padding: '8px 16px',
              borderRadius: '7px',
              border:
                `1px solid ${C.borderMid}`,
              background: '#fff',
              color: C.textSec,
              fontSize: '13px',
              cursor:
                saving
                  ? 'not-allowed'
                  : 'pointer',
              opacity:
                saving ? 0.5 : 1,
            }}
          >
            取消
          </button>

          <button
            type="button"
            onClick={() =>
              void handleSubmit()
            }
            disabled={
              saving ||
              loading ||
              subjectsLoading ||
              subjectsEmpty ||
              Boolean(loadErr)
            }
            style={{
              padding: '8px 20px',
              borderRadius: '7px',
              border: 'none',
              background:
                saving ||
                loading ||
                loadErr
                  ? C.borderMid
                  : C.primary,
              color:
                saving ||
                loading ||
                loadErr
                  ? C.textMuted
                  : '#fff',
              fontSize: '13px',
              fontWeight: 600,
              cursor:
                saving ||
                loading ||
                loadErr
                  ? 'not-allowed'
                  : 'pointer',
            }}
          >
            {saving
              ? '保存中...'
              : '💾 保存'}
          </button>
        </div>
      </div>
    </div>
  )
}
