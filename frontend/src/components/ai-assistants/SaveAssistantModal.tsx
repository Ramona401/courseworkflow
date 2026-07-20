/**
 * SaveAssistantModal.tsx — 对话式创作完成后的助手保存确认弹窗
 *
 * 表单负责确认：
 *   - 助手名称；
 *   - 发布货架；
 *   - 分享策略；
 *   - 当前教育域课程；
 *   - 当前教育域具体层级、学段或不限值；
 *   - 适用备课场景。
 *
 * 教育域层级规则：
 *   - 具体层级可以参与自动严格匹配；
 *   - K12小学/初中/高中只供手动选择；
 *   - 中职不限年级只供手动选择；
 *   - 成人不限层级只供手动选择；
 *   - 页面显示“职一”，提交值为“中职Ⅰ年级”。
 *
 * 本组件不再维护独立K12年级副本，所有层级统一消费：
 *   education-domain/options.ts
 */

import {
  useState,
  useEffect,
  useCallback,
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
  getAssistantLevelOptions,
  getEducationLevelLabel,
  isAutomaticEducationLevel,
  normalizeEducationLevelValue,
} from '@/education-domain/options'
import {
  createAssistant,
  getMyPublishGroups,
  ASSISTANT_SCENE_LABELS,
  SHARE_POLICY_LABELS,
  SHARE_POLICY_EMOJI,
  SHARE_POLICY_HINTS,
  DEFAULT_SHARE_POLICY,
  type AssistantScene,
  type AssistantSource,
  type AssistantSharePolicy,
  type CreateAIAssistantRequest,
  type PublishGroup,
} from '@/api/ai-assistants'

/* ==================== 样式 ==================== */

const C = {
  primary: '#4F7BE8',
  primaryLight:
    'rgba(79,123,232,0.08)',
  accent: '#F59E0B',
  success: '#10B981',
  danger: '#EF4444',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  bg: '#FAFBFC',
  card: '#FFFFFF',
  border: '#F3F4F6',
  borderMid: '#E5E7EB',
}

/** 工坊全部默认场景。 */
const WORKSHOP_SCENES:
  AssistantScene[] = [
    'workshop_analyze',
    'workshop_design',
    'workshop_write',
    'workshop_review',
    'workshop_revise',
  ]

/** 与后端提示词存储上限一致。 */
const MAX_PROMPT_LEN =
  128 * 1024

/** 备课工坊单助手运行时注入上限。 */
const WORKSHOP_RUNTIME_PROMPT_LEN =
  8000

/** 分享策略展示顺序。 */
const SHARE_POLICY_ORDER:
  AssistantSharePolicy[] = [
    'use_only',
    'open',
    'locked',
  ]

/* ==================== 发布货架 ==================== */

type ShelfKey =
  | 'personal'
  | 'group_teaching'
  | 'group_school'
  | 'system'

interface ShelfOption {
  key: ShelfKey
  emoji: string
  label: string
  hint: string
}

const SHELF_OPTIONS:
  Record<ShelfKey, ShelfOption> = {
    personal: {
      key: 'personal',
      emoji: '👤',
      label: '只给我自己用',
      hint: '存进我的助手，只有你能看到和使用',
    },
    group_teaching: {
      key: 'group_teaching',
      emoji: '👥',
      label: '发布到教研组',
      hint: '只推荐给所选教研组的老师',
    },
    group_school: {
      key: 'group_school',
      emoji: '🏫',
      label: '推荐给全校老师',
      hint: '本校老师备课时都能选用',
    },
    system: {
      key: 'system',
      emoji: '🏛️',
      label: '全平台通用',
      hint: '所有学校的老师都能使用',
    },
  }

function shelfToSourceAndGroup(
  shelf: ShelfKey,
  selectedGroupID: string,
): {
  source: AssistantSource
  groupID?: string
} {
  switch (shelf) {
    case 'personal':
      return {
        source: 'personal',
      }

    case 'group_teaching':
      return {
        source: 'group',
        groupID:
          selectedGroupID,
      }

    case 'group_school':
      return {
        source: 'group',
      }

    case 'system':
      return {
        source: 'system',
      }
  }
}

function isSharedShelf(
  shelf: ShelfKey,
): boolean {
  return shelf !== 'personal'
}

/* ==================== 受保护表单 ==================== */

interface SaveAssistantFormDraft {
  name: string
  subject: string
  gradeRange: string
  scenes: AssistantScene[]
  shelf: ShelfKey
  sharePolicy: AssistantSharePolicy
  selectedGroupID: string
}

function hashSaveAssistantDraftIdentity(
  value: string,
): string {
  let hash = 2166136261

  for (
    let index = 0;
    index < value.length;
    index += 1
  ) {
    hash ^= value.charCodeAt(index)
    hash = Math.imul(
      hash,
      16777619,
    )
  }

  return (hash >>> 0).toString(36)
}

function createSaveAssistantInitialForm(
  defaultSubject: string | undefined,
  defaultScene: AssistantScene | undefined,
  defaultGrade: string | undefined,
  subjectOptions: string[],
  normalizedDefaultGrade: string,
): SaveAssistantFormDraft {
  const rawSubject =
    (defaultSubject || '').trim()

  return {
    name: '',
    subject:
      subjectOptions.includes(
        rawSubject,
      )
        ? rawSubject
        : '',
    gradeRange:
      normalizedDefaultGrade ||
      (defaultGrade || '').trim(),
    scenes:
      defaultScene
        ? [defaultScene]
        : [...WORKSHOP_SCENES],
    shelf: 'personal',
    sharePolicy:
      DEFAULT_SHARE_POLICY,
    selectedGroupID: '',
  }
}

function parseSaveAssistantForm(
  raw: string,
  fallback: SaveAssistantFormDraft,
): SaveAssistantFormDraft {
  if (!raw.trim()) {
    return {
      ...fallback,
      scenes: [...fallback.scenes],
    }
  }

  try {
    const parsed =
      JSON.parse(raw) as
        Partial<SaveAssistantFormDraft>

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

    const shelf: ShelfKey =
      parsed.shelf ===
        'group_teaching' ||
      parsed.shelf ===
        'group_school' ||
      parsed.shelf === 'system'
        ? parsed.shelf
        : 'personal'

    const sharePolicy:
      AssistantSharePolicy =
      parsed.sharePolicy ===
        'open' ||
      parsed.sharePolicy ===
        'locked'
        ? parsed.sharePolicy
        : DEFAULT_SHARE_POLICY

    return {
      name:
        typeof parsed.name ===
          'string'
          ? parsed.name
          : fallback.name,
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
      shelf,
      sharePolicy,
      selectedGroupID:
        typeof parsed.selectedGroupID ===
          'string'
          ? parsed.selectedGroupID
          : fallback.selectedGroupID,
    }
  } catch {
    return {
      ...fallback,
      scenes: [...fallback.scenes],
    }
  }
}

/* ==================== Props ==================== */

export interface SaveAssistantModalProps {
  open: boolean
  draft: string
  userRole?: string
  defaultSubject?: string
  defaultScene?: AssistantScene
  defaultGrade?: string
  onClose: () => void
  onSaved: (
    id: string,
    source: AssistantSource,
  ) => void
}

/* ==================== 主组件 ==================== */

export default function SaveAssistantModal(
  props: SaveAssistantModalProps,
) {
  const {
    open,
    draft,
    defaultSubject,
    defaultScene,
    defaultGrade,
    onClose,
    onSaved,
  } = props

  const draftChars =
    Array.from(draft).length

  const { user } = useAuth()

  const {
    domain,
    profile,
  } = useEducationProfile()

  const {
    subjects: subjectOptions,
    loading: subjectsLoading,
    empty: subjectsEmpty,
  } = useSubjects()

  const levelOptions = useMemo(
    () => getAssistantLevelOptions(
      domain,
    ),
    [domain],
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

  const normalizedDefaultGrade =
    normalizeEducationLevelValue(
      domain,
      defaultGrade,
    )

  const initialForm = useMemo(
    () => createSaveAssistantInitialForm(
      defaultSubject,
      defaultScene,
      defaultGrade,
      subjectOptions,
      normalizedDefaultGrade,
    ),
    [
      defaultSubject,
      defaultScene,
      defaultGrade,
      subjectOptions,
      normalizedDefaultGrade,
    ],
  )

  /**
   * 草稿按Prompt内容和教育域双重隔离。
   *
   * 切换账号或从K12切换到职业教育时，
   * 不会错误恢复其它教育域的层级选择。
   */
  const formDraft =
    useProtectedDraft({
      userId: user?.id,
      scope: 'assistant-save',
      resourceId:
        `${domain}:` +
        hashSaveAssistantDraftIdentity(
          draft,
        ),
      field: 'form',
      initialValue:
        JSON.stringify(
          initialForm,
        ),
      maxHistory: 20,
    })

  const form =
    parseSaveAssistantForm(
      formDraft.value,
      initialForm,
    )

  const setForm: Dispatch<
    SetStateAction<
      SaveAssistantFormDraft
    >
  > = useCallback(
    next => {
      formDraft.setValue(
        previousText => {
          const previous =
            parseSaveAssistantForm(
              previousText,
              initialForm,
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
      initialForm,
    ],
  )

  const {
    name,
    subject,
    gradeRange,
    scenes,
    shelf,
    sharePolicy,
    selectedGroupID,
  } = form

  const setName = (value: string) =>
    setForm(previous => ({
      ...previous,
      name: value,
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

  const setShelf = (
    value: ShelfKey,
  ) =>
    setForm(previous => ({
      ...previous,
      shelf: value,
    }))

  const setSharePolicy = (
    value: AssistantSharePolicy,
  ) =>
    setForm(previous => ({
      ...previous,
      sharePolicy: value,
    }))

  const setSelectedGroupID = (
    value: string,
  ) =>
    setForm(previous => ({
      ...previous,
      selectedGroupID: value,
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

  /* ==================== 课程与层级修正 ==================== */

  useEffect(() => {
    if (
      !open ||
      subjectsLoading
    ) {
      return
    }

    const rawDefault =
      (defaultSubject || '').trim()

    const preferredSubject =
      subjectOptions.includes(
        rawDefault,
      )
        ? rawDefault
        : ''

    setForm(current => {
      const nextSubject =
        subjectOptions.includes(
          current.subject,
        )
          ? current.subject
          : preferredSubject

      if (
        nextSubject ===
        current.subject
      ) {
        return current
      }

      return {
        ...current,
        subject: nextSubject,
      }
    })
  }, [
    open,
    subjectsLoading,
    subjectOptions,
    defaultSubject,
    setForm,
  ])

  /**
   * 清理跨教育域草稿或历史别名。
   *
   * 当前值能规范化时使用规范值；
   * 无法规范化时回退到当前教育域第一个具体层级。
   */
  useEffect(() => {
    if (
      !open ||
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
          domain,
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
    domain,
    levelOptions,
    levelValues,
    specificLevelOptions,
    broadLevelOptions,
    setForm,
  ])

  /* ==================== 运行状态 ==================== */

  const [saving, setSaving] =
    useState(false)

  const [errorMessage, setErrorMessage] =
    useState<string | null>(null)

  const [shelfKeys, setShelfKeys] =
    useState<ShelfKey[]>([
      'personal',
    ])

  const [
    publishGroups,
    setPublishGroups,
  ] = useState<PublishGroup[]>([])

  const [
    loadingScope,
    setLoadingScope,
  ] = useState(false)

  const showShelfPicker =
    shelfKeys.length > 1

  const showPolicyPicker =
    isSharedShelf(shelf)

  /* ==================== 打开时加载发布范围 ==================== */

  useEffect(() => {
    if (!open) return

    setErrorMessage(null)
    setSaving(false)
    setShelfKeys(['personal'])
    setPublishGroups([])

    let cancelled = false

    setLoadingScope(true)

    getMyPublishGroups()
      .then(response => {
        if (cancelled) return

        const keys:
          ShelfKey[] = [
            'personal',
          ]

        if (
          response.can_publish_group &&
          response.groups.length > 0
        ) {
          keys.push(
            'group_teaching',
          )
        }

        if (
          response.can_publish_school
        ) {
          keys.push(
            'group_school',
          )
        }

        if (
          response.can_publish_system
        ) {
          keys.push('system')
        }

        setShelfKeys(keys)
        setPublishGroups(
          response.groups || [],
        )

        setForm(current => ({
          ...current,
          shelf:
            keys.includes(
              current.shelf,
            )
              ? current.shelf
              : 'personal',
          selectedGroupID:
            current.selectedGroupID &&
            response.groups.some(
              group =>
                group.id ===
                current.selectedGroupID,
            )
              ? current.selectedGroupID
              : response.groups[0]
                  ?.id || '',
        }))
      })
      .catch(() => {
        if (cancelled) return

        setShelfKeys([
          'personal',
        ])
        setPublishGroups([])

        setForm(current => ({
          ...current,
          shelf: 'personal',
          selectedGroupID: '',
        }))
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingScope(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [
    open,
    domain,
    setForm,
  ])

  useEffect(() => {
    if (!open) return

    const handler = (
      event: KeyboardEvent,
    ) => {
      if (
        event.key === 'Escape' &&
        !saving
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
    saving,
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

  /* ==================== 层级提示 ==================== */

  const gradeHint = (
    value: string,
  ): string => {
    const label =
      getEducationLevelLabel(
        domain,
        value,
      )

    if (
      isAutomaticEducationLevel(
        domain,
        value,
      )
    ) {
      return `${label}属于具体${profile.grade_label}，课程、层级和阶段严格一致时可以自动匹配。`
    }

    return `${label || '不限层级'}只供老师手动选择，不会被平台自动挂载。`
  }

  /* ==================== 保存 ==================== */

  const handleSave = async () => {
    if (saving) return

    if (subjectsLoading) {
      setErrorMessage(
        '课程目录仍在加载，请稍后再试',
      )
      return
    }

    if (subjectsEmpty) {
      setErrorMessage(
        '当前组织尚未配置可用课程，暂时不能保存助手',
      )
      return
    }

    if (!name.trim()) {
      setErrorMessage(
        '请给助手起个名字，方便以后识别',
      )
      return
    }

    if (
      !subject.trim() ||
      !subjectOptions.includes(
        subject.trim(),
      )
    ) {
      setErrorMessage(
        `请选择助手适用的具体${profile.subject_label}`,
      )
      return
    }

    if (
      !levelValues.includes(
        gradeRange.trim(),
      )
    ) {
      setErrorMessage(
        `请选择当前教育域支持的${profile.grade_label}或通用层级`,
      )
      return
    }

    if (!draft.trim()) {
      setErrorMessage(
        '草稿还是空的，请先和AI聊出一版',
      )
      return
    }

    if (scenes.length === 0) {
      setErrorMessage(
        '请至少勾选一个适用场景',
      )
      return
    }

    if (
      draft.length >
      MAX_PROMPT_LEN
    ) {
      setErrorMessage(
        `提示词过长（${draft.length}字符），上限${MAX_PROMPT_LEN}字符`,
      )
      return
    }

    const finalShelf:
      ShelfKey =
      shelfKeys.includes(shelf)
        ? shelf
        : 'personal'

    if (
      finalShelf ===
        'group_teaching' &&
      !selectedGroupID
    ) {
      setErrorMessage(
        '请选择要发布到哪个教研组',
      )
      return
    }

    const {
      source,
      groupID,
    } = shelfToSourceAndGroup(
      finalShelf,
      selectedGroupID,
    )

    setSaving(true)
    setErrorMessage(null)

    try {
      const request:
        CreateAIAssistantRequest = {
        name: name.trim(),
        avatar_emoji: '🤖',
        description: '',
        source,
        full_prompt: draft,
        subject: subject.trim(),
        grade_range:
          gradeRange.trim(),
        scenes,
        ...(groupID
          ? {
              group_id: groupID,
            }
          : {}),
        ...(isSharedShelf(
          finalShelf,
        )
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

      onSaved(
        created.id,
        created.source ||
          source,
      )
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : '保存失败，请重试',
      )
      setSaving(false)
    }
  }

  if (!open) return null

  return (
    <div
      onClick={() => {
        if (!saving) onClose()
      }}
      style={{
        position: 'fixed',
        inset: 0,
        background:
          'rgba(17,24,39,0.5)',
        zIndex: 10001,
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
          width: '540px',
          maxWidth: '100%',
          maxHeight: '90vh',
          background: C.card,
          borderRadius: '14px',
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
          background:
            'linear-gradient(135deg,rgba(16,185,129,0.06),rgba(16,185,129,0.02))',
          display: 'flex',
          alignItems: 'center',
          justifyContent:
            'space-between',
          flexShrink: 0,
        }}>
          <span style={{
            fontSize: '15px',
            fontWeight: 700,
            color: C.text,
          }}>
            💾 保存这个助手
          </span>

          <button
            type="button"
            onClick={() => {
              if (!saving) onClose()
            }}
            style={{
              background: 'none',
              border: 'none',
              cursor: saving
                ? 'not-allowed'
                : 'pointer',
              fontSize: '20px',
              color: C.textMuted,
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
          <div style={{
            marginBottom: '16px',
            padding: '10px 12px',
            borderRadius: '8px',
            background: C.primaryLight,
            border:
              '1px solid rgba(79,123,232,0.15)',
            fontSize: '12px',
            color: C.textSec,
            lineHeight: 1.6,
          }}>
            ✨ 已生成一版提示词草稿（
            <b style={{
              color: C.primary,
            }}>
              {draftChars.toLocaleString()}
            </b>
            个字符）。填写以下信息后即可保存。
          </div>

          {draftChars >
           WORKSHOP_RUNTIME_PROMPT_LEN && (
            <div style={{
              marginBottom: '16px',
              padding: '10px 12px',
              borderRadius: '8px',
              background:
                'rgba(245,158,11,0.08)',
              border:
                '1px solid rgba(245,158,11,0.28)',
              color: '#92400E',
              fontSize: '12px',
              lineHeight: 1.6,
            }}>
              ⚠️ 完整原稿会保存，但备课工坊单次最多注入前
              {' '}
              <b>
                {WORKSHOP_RUNTIME_PROMPT_LEN.toLocaleString()}
              </b>
              {' '}
              个Unicode字符。
            </div>
          )}

          {/* 名称 */}
          <div style={{
            marginBottom: '16px',
          }}>
            <label style={labelStyle}>
              助手名称
              {' '}
              <span style={{
                color: C.danger,
              }}>
                *
              </span>
            </label>

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
              placeholder="例如：我的实训任务设计助手"
              maxLength={100}
              autoFocus
              style={{
                ...inputStyle,
                width: '100%',
              }}
            />
          </div>

          {/* 货架 */}
          {showShelfPicker && (
            <div style={{
              marginBottom: '16px',
            }}>
              <label style={labelStyle}>
                保存到哪里
                {' '}
                <span style={{
                  color: C.danger,
                }}>
                  *
                </span>
              </label>

              <div style={{
                display: 'flex',
                flexDirection: 'column',
                gap: '6px',
                marginTop: '4px',
              }}>
                {shelfKeys.map(
                  shelfKey => {
                    const option =
                      SHELF_OPTIONS[
                        shelfKey
                      ]

                    const checked =
                      shelf === shelfKey

                    return (
                      <div key={shelfKey}>
                        <label style={{
                          display: 'flex',
                          alignItems:
                            'flex-start',
                          gap: '8px',
                          padding:
                            '10px 12px',
                          borderRadius:
                            '8px',
                          border:
                            `1.5px solid ${
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
                        }}>
                          <input
                            type="radio"
                            name="shelf"
                            checked={checked}
                            onChange={() =>
                              setShelf(
                                shelfKey,
                              )
                            }
                            style={{
                              cursor:
                                'pointer',
                              accentColor:
                                C.primary,
                              marginTop:
                                '2px',
                            }}
                          />

                          <div style={{
                            flex: 1,
                          }}>
                            <div style={{
                              fontSize:
                                '13px',
                              fontWeight:
                                600,
                              color: C.text,
                            }}>
                              {option.emoji}
                              {' '}
                              {option.label}
                            </div>

                            <div style={{
                              fontSize:
                                '11px',
                              color:
                                C.textSec,
                              marginTop:
                                '2px',
                              lineHeight:
                                1.5,
                            }}>
                              {option.hint}
                            </div>
                          </div>
                        </label>

                        {shelfKey ===
                          'group_teaching' &&
                         checked && (
                          <div style={{
                            margin:
                              '6px 0 2px 28px',
                          }}>
                            <select
                              value={
                                selectedGroupID
                              }
                              onChange={
                                event =>
                                  setSelectedGroupID(
                                    event
                                      .target
                                      .value,
                                  )
                              }
                              style={{
                                ...inputStyle,
                                width:
                                  '100%',
                                cursor:
                                  'pointer',
                              }}
                            >
                              {publishGroups.map(
                                group => (
                                  <option
                                    key={
                                      group.id
                                    }
                                    value={
                                      group.id
                                    }
                                  >
                                    {group.name}
                                    {group.role ===
                                      'lead'
                                      ? '（组长）'
                                      : '（骨干）'}
                                    {group.school_name
                                      ? ` · ${group.school_name}`
                                      : ''}
                                  </option>
                                ),
                              )}
                            </select>
                          </div>
                        )}
                      </div>
                    )
                  },
                )}
              </div>
            </div>
          )}

          {loadingScope &&
           !showShelfPicker && (
            <div style={{
              marginBottom: '16px',
              fontSize: '12px',
              color: C.textMuted,
            }}>
              正在确认发布范围…
            </div>
          )}

          {/* 分享策略 */}
          {showPolicyPicker && (
            <div style={{
              marginBottom: '16px',
            }}>
              <label style={labelStyle}>
                别人能怎么用
              </label>

              <div style={{
                display: 'flex',
                flexDirection: 'column',
                gap: '6px',
                marginTop: '4px',
              }}>
                {SHARE_POLICY_ORDER.map(
                  policy => {
                    const checked =
                      sharePolicy ===
                      policy

                    return (
                      <label
                        key={policy}
                        style={{
                          display: 'flex',
                          alignItems:
                            'flex-start',
                          gap: '8px',
                          padding:
                            '10px 12px',
                          borderRadius:
                            '8px',
                          border:
                            `1.5px solid ${
                              checked
                                ? C.accent
                                : C.border
                            }`,
                          background:
                            checked
                              ? 'rgba(245,158,11,0.06)'
                              : '#fff',
                          cursor:
                            'pointer',
                        }}
                      >
                        <input
                          type="radio"
                          name="share_policy"
                          checked={checked}
                          onChange={() =>
                            setSharePolicy(
                              policy,
                            )
                          }
                          style={{
                            cursor:
                              'pointer',
                            accentColor:
                              C.accent,
                            marginTop:
                              '2px',
                          }}
                        />

                        <div style={{
                          flex: 1,
                        }}>
                          <div style={{
                            fontSize:
                              '13px',
                            fontWeight:
                              600,
                            color: C.text,
                          }}>
                            {SHARE_POLICY_EMOJI[policy]}
                            {' '}
                            {SHARE_POLICY_LABELS[policy]}
                          </div>

                          <div style={{
                            fontSize:
                              '11px',
                            color: C.textSec,
                            marginTop:
                              '2px',
                            lineHeight:
                              1.5,
                          }}>
                            {SHARE_POLICY_HINTS[policy]}
                          </div>
                        </div>
                      </label>
                    )
                  },
                )}
              </div>
            </div>
          )}

          {/* 课程 */}
          <div style={{
            marginBottom: '16px',
          }}>
            <label style={labelStyle}>
              适用
              {profile.subject_label}
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
              onChange={event =>
                setSubject(
                  event.target.value,
                )
              }
              style={{
                ...inputStyle,
                width: '100%',
                cursor: 'pointer',
              }}
            >
              <option value="">
                请选择具体
                {profile.subject_label}
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
                color: C.danger,
                lineHeight: 1.5,
              }}>
                当前组织尚未配置可用
                {profile.subject_label}。
              </div>
            )}
          </div>

          {/* 学习层级 */}
          <div style={{
            marginBottom: '16px',
          }}>
            <label style={labelStyle}>
              适用
              {profile.grade_label}
              <span style={{
                color: C.textMuted,
                fontWeight: 400,
              }}>
                {' '}（影响自动或手动匹配）
              </span>
            </label>

            <select
              value={gradeRange}
              onChange={event =>
                setGradeRange(
                  event.target.value,
                )
              }
              style={{
                ...inputStyle,
                width: '100%',
                cursor: 'pointer',
              }}
            >
              <optgroup
                label={`具体${profile.grade_label}（可自动匹配）`}
              >
                {specificLevelOptions.map(
                  option => (
                    <option
                      key={option.value}
                      value={option.value}
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
              color:
                isAutomaticEducationLevel(
                  domain,
                  gradeRange,
                )
                  ? '#166534'
                  : C.textMuted,
              lineHeight: 1.55,
            }}>
              {gradeHint(gradeRange)}
            </div>
          </div>

          {/* 场景 */}
          <div style={{
            marginBottom: '8px',
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
                'repeat(2, 1fr)',
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

          {errorMessage && (
            <div style={{
              marginTop: '14px',
              padding: '10px 12px',
              borderRadius: '8px',
              background:
                'rgba(239,68,68,0.06)',
              border:
                '1px solid rgba(239,68,68,0.2)',
              color: C.danger,
              fontSize: '13px',
            }}>
              ⚠️ {errorMessage}
            </div>
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
            onClick={() => {
              if (!saving) onClose()
            }}
            disabled={saving}
            style={{
              padding: '8px 16px',
              borderRadius: '7px',
              border:
                `1px solid ${C.borderMid}`,
              background: '#fff',
              color: C.textSec,
              fontSize: '13px',
              cursor: saving
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
              void handleSave()
            }
            disabled={
              saving ||
              subjectsLoading ||
              subjectsEmpty
            }
            style={{
              padding: '8px 20px',
              borderRadius: '7px',
              border: 'none',
              background:
                saving
                  ? C.borderMid
                  : C.success,
              color:
                saving
                  ? C.textMuted
                  : '#fff',
              fontSize: '13px',
              fontWeight: 600,
              cursor:
                saving
                  ? 'not-allowed'
                  : 'pointer',
            }}
          >
            {saving
              ? '保存中...'
              : '✓ 保存助手'}
          </button>
        </div>
      </div>
    </div>
  )
}

/* ==================== 样式辅助 ==================== */

const labelStyle:
  React.CSSProperties = {
    display: 'block',
    fontSize: '12px',
    fontWeight: 600,
    color: C.textSec,
    marginBottom: '4px',
  }

const inputStyle:
  React.CSSProperties = {
    padding: '8px 10px',
    borderRadius: '6px',
    border:
      `1px solid ${C.border}`,
    fontSize: '13px',
    color: C.text,
    outline: 'none',
    boxSizing: 'border-box',
    fontFamily: 'inherit',
    background: '#fff',
  }
