/**
 * UnitPlansPanel.tsx — 大单元备课面板（单元方案 Tab）
 *
 * 当前能力：
 * 1. 新建大单元逐步设计会话；
 * 2. 草稿恢复和继续对话；
 * 3. 正式方案只读查看；
 * 4. 创建者重新进入已发布方案继续优化；
 * 5. AI续作结果中自动识别完整最新版正文与完整图谱；
 * 6. 正式方案再次保存，不新建重复方案；
 * 7. 非创建者仍保持只读；
 * 8. 会话级课本OCR原文选择和注入。
 *
 * 续作协议：
 * 后端在已发布方案续作时，以数据库当前 content / atlas 为唯一权威底稿，并要求AI在
 * 回复末尾输出：
 *
 *   【最新版方案正文】
 *   ...
 *   【最新版单元整体设计图谱】
 *   ...
 *
 * 本组件只负责确定性解析这两个标记，不让AI自行声明是否已更新。
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import { useAuth } from '@/store/auth'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import { DEFAULT_SUBJECTS } from '@/constants/subjects'
import type {
  CSSProperties,
  ReactNode,
  Dispatch,
  SetStateAction,
  MouseEvent as ReactMouseEvent,
  KeyboardEvent as ReactKeyboardEvent,
} from 'react'
import {
  getUnitPlans,
  getUnitPlan,
  startUnitPlan,
  chatUnitPlan,
  saveUnitPlan,
  deleteUnitPlan,
} from '@/api/unit-plans'
import type {
  UnitPlanListItem,
  UnitPlanDetail,
  UnitPlanMessage,
  UnitPlanScope,
  UnitPlanStatus,
} from '@/api/unit-plans'
import { getMyPublishGroups } from '@/api/ai-assistants'
import {
  getTextbooks,
  getTextbook,
  type TextbookListItem,
} from '@/api/textbooks'
import {
  getAvailablePublishers,
  publisherLabel,
} from '@/api/course-outlines'
import UnitPlanMaterialsModal from './UnitPlanMaterialsModal'

// ==================== 颜色常量 ====================

const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  border: '#E5E7EB',
  white: '#FFFFFF',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
}

// ==================== 下拉选项 ====================

const SUBJECTS = [...DEFAULT_SUBJECTS]

const GRADES = [
  '一年级',
  '二年级',
  '三年级',
  '四年级',
  '五年级',
  '六年级',
  '七年级',
  '八年级',
  '九年级',
  '高一',
  '高二',
  '高三',
]

const VOLUMES = ['上册', '下册', '全册']

// 空串是“通用版”的合法值，因此“不关联”必须使用独立哨兵值。
const PUBLISHER_NONE = '__NONE__'

interface NewUnitPlanForm {
  scopeKey: string
  subject: string
  grade: string
  volume: string
  unit: string
  title: string
  publisher: string
}

interface UnitPlanSaveForm {
  title: string
  unit_theme: string
  content: string
  atlas: string
}

const DEFAULT_NEW_UNIT_PLAN_FORM: NewUnitPlanForm = {
  scopeKey: '',
  subject: '语文',
  grade: '三年级',
  volume: '下册',
  unit: '',
  title: '',
  publisher: PUBLISHER_NONE,
}

const EMPTY_UNIT_PLAN_SAVE_FORM: UnitPlanSaveForm = {
  title: '',
  unit_theme: '',
  content: '',
  atlas: '',
}

/**
 * 安全解析新建单元方案表单。
 */
function parseNewUnitPlanForm(
  raw: string,
): NewUnitPlanForm {
  if (!raw.trim()) {
    return {
      ...DEFAULT_NEW_UNIT_PLAN_FORM,
    }
  }

  try {
    const parsed = JSON.parse(
      raw,
    ) as Partial<NewUnitPlanForm>

    return {
      scopeKey:
        typeof parsed.scopeKey === 'string'
          ? parsed.scopeKey
          : '',
      subject:
        typeof parsed.subject === 'string'
          ? parsed.subject
          : DEFAULT_NEW_UNIT_PLAN_FORM.subject,
      grade:
        typeof parsed.grade === 'string'
          ? parsed.grade
          : DEFAULT_NEW_UNIT_PLAN_FORM.grade,
      volume:
        typeof parsed.volume === 'string'
          ? parsed.volume
          : DEFAULT_NEW_UNIT_PLAN_FORM.volume,
      unit:
        typeof parsed.unit === 'string'
          ? parsed.unit
          : '',
      title:
        typeof parsed.title === 'string'
          ? parsed.title
          : '',
      publisher:
        typeof parsed.publisher === 'string'
          ? parsed.publisher
          : PUBLISHER_NONE,
    }
  } catch {
    return {
      ...DEFAULT_NEW_UNIT_PLAN_FORM,
    }
  }
}

/**
 * 安全解析正式方案保存表单。
 */
function parseUnitPlanSaveForm(
  raw: string,
): UnitPlanSaveForm {
  if (!raw.trim()) {
    return {
      ...EMPTY_UNIT_PLAN_SAVE_FORM,
    }
  }

  try {
    const parsed = JSON.parse(
      raw,
    ) as Partial<UnitPlanSaveForm>

    return {
      title:
        typeof parsed.title === 'string'
          ? parsed.title
          : '',
      unit_theme:
        typeof parsed.unit_theme === 'string'
          ? parsed.unit_theme
          : '',
      content:
        typeof parsed.content === 'string'
          ? parsed.content
          : '',
      atlas:
        typeof parsed.atlas === 'string'
          ? parsed.atlas
          : '',
    }
  } catch {
    return {
      ...EMPTY_UNIT_PLAN_SAVE_FORM,
    }
  }
}

// ==================== 续作标记协议 ====================

const LATEST_CONTENT_MARKER = '【最新版方案正文】'
const LATEST_ATLAS_MARKER = '【最新版单元整体设计图谱】'

interface ParsedRevision {
  content?: string
  atlas?: string
}

/**
 * 从AI回复中解析最后一组完整续作标记。
 *
 * 使用 lastIndexOf 的原因：
 * 历史内容、引用内容中也可能出现标记说明；真正需要的是AI本轮回复末尾最后一组标记。
 */
function parseLatestRevision(text: string): ParsedRevision {
  const contentPos = text.lastIndexOf(LATEST_CONTENT_MARKER)
  const atlasPos = text.lastIndexOf(LATEST_ATLAS_MARKER)

  if (contentPos < 0 && atlasPos < 0) {
    return {}
  }

  const result: ParsedRevision = {}

  if (contentPos >= 0) {
    const contentStart = contentPos + LATEST_CONTENT_MARKER.length
    const contentEnd =
      atlasPos > contentPos
        ? atlasPos
        : text.length

    const content = text.slice(contentStart, contentEnd).trim()
    if (content) {
      result.content = content
    }
  }

  if (atlasPos >= 0) {
    const atlasStart = atlasPos + LATEST_ATLAS_MARKER.length
    const atlas = text.slice(atlasStart).trim()
    if (atlas) {
      result.atlas = atlas
    }
  }

  return result
}

// ==================== 归属与状态 ====================

interface ScopeOption {
  key: string
  label: string
  scope: UnitPlanScope
  target: string
}

function scopeBadge(s: UnitPlanScope) {
  if (s === 'system') {
    return { t: '🌐 全局', bg: '#F3E8FF', c: '#7C3AED' }
  }
  if (s === 'school') {
    return { t: '🏛️ 全校', bg: '#DCFCE7', c: '#16A34A' }
  }
  return { t: '🏫 教研组', bg: '#DBEAFE', c: '#2563EB' }
}

function statusBadge(s: UnitPlanStatus) {
  if (s === 'draft') {
    return { t: '草稿', bg: '#FEF3C7', c: '#B45309' }
  }
  if (s === 'active') {
    return { t: '已发布', bg: '#DCFCE7', c: '#16A34A' }
  }
  return { t: '已归档', bg: '#F3F4F6', c: '#6B7280' }
}

// ==================== 通用样式 ====================

const inputStyle: CSSProperties = {
  width: '100%',
  padding: '8px 10px',
  border: '1px solid ' + C.border,
  borderRadius: 8,
  fontSize: 13,
  color: C.textPrimary,
  outline: 'none',
  boxSizing: 'border-box',
}

const labelStyle: CSSProperties = {
  fontSize: 12,
  fontWeight: 600,
  color: C.textSecondary,
  marginBottom: 4,
  display: 'block',
}

const btnPrimary: CSSProperties = {
  padding: '8px 16px',
  background: C.primary,
  color: '#fff',
  border: 'none',
  borderRadius: 8,
  fontSize: 13,
  fontWeight: 600,
  cursor: 'pointer',
}

const btnGhost: CSSProperties = {
  padding: '8px 16px',
  background: C.white,
  color: C.textSecondary,
  border: '1px solid ' + C.border,
  borderRadius: 8,
  fontSize: 13,
  cursor: 'pointer',
}

const btnTextbook: CSSProperties = {
  padding: '6px 12px',
  background: '#10B981',
  color: '#fff',
  border: 'none',
  borderRadius: 8,
  fontSize: 12,
  fontWeight: 600,
  cursor: 'pointer',
  whiteSpace: 'nowrap',
}

// ==================== 主组件 ====================

export default function UnitPlansPanel() {
  const { user } = useAuth()

  const [list, setList] = useState<UnitPlanListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [view, setView] = useState<'list' | 'session'>('list')

  const [scopeOptions, setScopeOptions] = useState<ScopeOption[]>([])
  const [canCreate, setCanCreate] = useState(false)

  const [showNew, setShowNew] = useState(false)

  /**
   * 新建单元方案表单统一保存为JSON草稿。
   *
   * 关闭弹窗、刷新或切换页面后仍可恢复；
   * 创建成功后只重置已经消费的单元、标题和教材版本。
   */
  const newPlanDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'unit-plan-new',
    resourceId: 'new-plan',
    field: 'form',
    initialValue: JSON.stringify(
      DEFAULT_NEW_UNIT_PLAN_FORM,
    ),
    maxHistory: 30,
  })

  const nf = parseNewUnitPlanForm(
    newPlanDraft.value,
  )

  const setNf: Dispatch<
    SetStateAction<NewUnitPlanForm>
  > = useCallback(
    (next) => {
      newPlanDraft.setValue(
        (previousText) => {
          const previous =
            parseNewUnitPlanForm(
              previousText,
            )

          const resolved =
            typeof next === 'function'
              ? next(previous)
              : next

          return JSON.stringify(
            resolved,
          )
        },
      )
    },
    [newPlanDraft.setValue],
  )

  const [starting, setStarting] = useState(false)

  const [availablePublishers, setAvailablePublishers] = useState<string[]>([])
  const [publishersLoading, setPublishersLoading] = useState(false)

  const [plan, setPlan] = useState<UnitPlanDetail | null>(null)
  const [messages, setMessages] = useState<UnitPlanMessage[]>([])
  /**
   * 单元方案AI会话输入按当前用户和方案ID隔离。
   */
  const unitChatDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'unit-plan-conversation',
    resourceId:
      plan?.id || 'no-active-plan',
    field: 'message',
    initialValue: '',
    maxHistory: 40,
  })

  const input = unitChatDraft.value
  const setInput = unitChatDraft.setValue

  const [sending, setSending] = useState(false)
  const [canEditCurrent, setCanEditCurrent] = useState(false)

  /**
   * workingDraft 是当前等待保存的完整版本。
   *
   * 重新打开正式方案时，先以数据库中的 content / atlas 初始化；
   * AI返回标准续作标记后，再用新版本覆盖；
   * 因此即使AI某一轮没有返回标准标记，保存弹窗也不会把正式方案清空。
   */
  const [workingDraft, setWorkingDraft] = useState({
    content: '',
    atlas: '',
  })

  /**
   * 是否已经从AI本轮回复中识别到完整最新版正文。
   * 仅用于界面提示，不参与后端权限和保存判定。
   */
  const [revisionReady, setRevisionReady] = useState(false)

  const [showSave, setShowSave] = useState(false)
  /**
   * 正式方案保存表单统一保存。
   *
   * 方案正文可能较长，因此历史数量限制为12份。
   */
  const saveFormDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'unit-plan-save',
    resourceId:
      plan?.id || 'no-active-plan',
    field: 'form',
    initialValue: '',
    maxHistory: 12,
  })

  const sf = parseUnitPlanSaveForm(
    saveFormDraft.value,
  )

  const setSf: Dispatch<
    SetStateAction<UnitPlanSaveForm>
  > = useCallback(
    (next) => {
      saveFormDraft.setValue(
        (previousText) => {
          const previous =
            parseUnitPlanSaveForm(
              previousText,
            )

          const resolved =
            typeof next === 'function'
              ? next(previous)
              : next

          return JSON.stringify(
            resolved,
          )
        },
      )
    },
    [saveFormDraft.setValue],
  )

  const [saving, setSaving] = useState(false)

  const [viewPlan, setViewPlan] = useState<UnitPlanDetail | null>(null)
  const [toast, setToast] = useState('')
  const msgEndRef = useRef<HTMLDivElement | null>(null)

  const [showTextbookPicker, setShowTextbookPicker] = useState(false)
  const [showMaterials, setShowMaterials] = useState(false)
  const [textbookContext, setTextbookContext] = useState('')
  const [textbookCount, setTextbookCount] = useState(0)

  const showToast = (m: string) => {
    setToast(m)
    window.setTimeout(() => setToast(''), 2600)
  }

  const scrollBottom = () => {
    window.setTimeout(
      () => msgEndRef.current?.scrollIntoView({ behavior: 'smooth' }),
      60,
    )
  }

  const loadList = useCallback(async () => {
    setLoading(true)
    try {
      const r = await getUnitPlans()
      setList(r.unit_plans || [])
    } catch (e: any) {
      showToast(e?.message || '加载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadList()
  }, [loadList])

  useEffect(() => {
    let cancelled = false

    ;(async () => {
      try {
        const r: any = await getMyPublishGroups()
        if (cancelled) return

        const opts: ScopeOption[] = (r.groups || []).map(
          (g: any): ScopeOption => ({
            key: 'group:' + g.id,
            label:
              g.name +
              '（' +
              (g.role === 'lead' ? '组长' : '骨干') +
              '）',
            scope: 'group',
            target: g.id,
          }),
        )

        if (r.can_publish_system) {
          opts.unshift({
            key: 'system:',
            label: '🌐 全局（所有学校通用）',
            scope: 'system',
            target: '',
          })
        }

        setScopeOptions(opts)
        setCanCreate(opts.length > 0)
        setNf((s) => ({
          ...s,
          scopeKey:
            s.scopeKey &&
            opts.some(
              (option) =>
                option.key === s.scopeKey,
            )
              ? s.scopeKey
              : opts[0]?.key || '',
        }))
      } catch {
        // 无可发布归属时仅作为消费端查看方案。
      }
    })()

    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!showNew) return

    let cancelled = false

    setPublishersLoading(true)
    setAvailablePublishers([])

    ;(async () => {
      try {
        const pubs = await getAvailablePublishers(nf.subject, nf.grade)
        if (!cancelled) {
          setAvailablePublishers(pubs)
          setNf((current) => ({
            ...current,
            publisher:
              current.publisher === PUBLISHER_NONE ||
              pubs.includes(current.publisher)
                ? current.publisher
                : PUBLISHER_NONE,
          }))
        }
      } catch {
        if (!cancelled) {
          setAvailablePublishers([])
          setNf((current) => ({
            ...current,
            publisher: PUBLISHER_NONE,
          }))
        }
      } finally {
        if (!cancelled) {
          setPublishersLoading(false)
        }
      }
    })()

    return () => {
      cancelled = true
    }
  }, [showNew, nf.subject, nf.grade])

  // ==================== 会话状态清理 ====================

  const clearSession = () => {
    setPlan(null)
    setMessages([])
    setCanEditCurrent(false)
    setWorkingDraft({ content: '', atlas: '' })
    setRevisionReady(false)
    setTextbookContext('')
    setTextbookCount(0)
    setShowSave(false)
    setShowTextbookPicker(false)
    setShowMaterials(false)
  }

  const leaveSession = () => {
    clearSession()
    setView('list')
    loadList()
  }

  // ==================== 新建会话 ====================

  const doStart = async () => {
    const opt = scopeOptions.find((o) => o.key === nf.scopeKey)

    if (!opt) {
      showToast('请选择归属')
      return
    }

    if (!nf.unit.trim()) {
      showToast('请填写单元')
      return
    }

    setStarting(true)

    try {
      const r = await startUnitPlan({
        scope: opt.scope,
        scope_target_id: opt.target,
        subject: nf.subject,
        grade: nf.grade,
        volume: nf.volume,
        unit: nf.unit.trim(),
        title: nf.title.trim() || undefined,
        ...(nf.publisher !== PUBLISHER_NONE
          ? { course_outline_publisher: nf.publisher }
          : {}),
      })

      setPlan(r.plan)
      // 创建成功后重置已消费字段，保留常用归属、学科、年级和册次。
      setNf((current) => ({
        ...current,
        unit: '',
        title: '',
        publisher: PUBLISHER_NONE,
      }))
      setMessages([
        {
          role: 'assistant',
          content: r.opening,
          created_at: '',
        },
      ])
      setCanEditCurrent(true)
      setWorkingDraft({
        content: r.plan.content || '',
        atlas: r.plan.atlas || '',
      })
      setRevisionReady(false)
      setTextbookContext('')
      setTextbookCount(0)
      setShowNew(false)
      setView('session')
      scrollBottom()
    } catch (e: any) {
      showToast(e?.message || '开始失败')
    } finally {
      setStarting(false)
    }
  }

  // ==================== 打开已有方案 ====================

  const openExisting = async (item: UnitPlanListItem) => {
    try {
      const r = await getUnitPlan(item.id)

      // 是否可以进入AI续作完全以后端 can_edit 为准。
      // 创建者的 draft / active 均进入会话；其他可见用户只打开只读详情。
      if (r.can_edit) {
        setPlan(r.plan)
        setMessages(r.messages || [])
        setCanEditCurrent(true)
        setWorkingDraft({
          content: r.plan.content || '',
          atlas: r.plan.atlas || '',
        })
        setRevisionReady(false)
        setTextbookContext('')
        setTextbookCount(0)
        setViewPlan(null)
        setView('session')
        scrollBottom()
        return
      }

      setCanEditCurrent(false)
      setViewPlan(r.plan)
    } catch (e: any) {
      showToast(e?.message || '打开失败')
    }
  }

  // ==================== 发送消息 ====================

  const doSend = async () => {
    if (
      !plan ||
      !input.trim() ||
      sending
    ) {
      return
    }

    const msg = input.trim()
    const localCreatedAt =
      `local_${Date.now()}`

    const displayMsg = textbookContext
      ? msg + '\n（已附课本原文参考）'
      : msg

    setMessages((current) => [
      ...current,
      {
        role: 'user',
        content: displayMsg,
        created_at: localCreatedAt,
      },
    ])

    setSending(true)
    scrollBottom()

    try {
      const aiMsg = textbookContext
        ? `【老师上传的教材原文参考】\n${textbookContext}\n\n【老师本轮说的话】\n${msg}`
        : msg

      const reply = await chatUnitPlan(
        plan.id,
        aiMsg,
      )

      setMessages((current) => [
        ...current,
        {
          role: 'assistant',
          content: reply,
          created_at: '',
        },
      ])

      /**
       * 后端成功返回后，本轮输入才算正式消费。
       * commit清空显示值，但保留Ctrl+Z恢复快照。
       */
      unitChatDraft.commit()

      const parsed =
        parseLatestRevision(reply)

      if (parsed.content || parsed.atlas) {
        setWorkingDraft((previous) => ({
          content:
            parsed.content ||
            previous.content,
          atlas:
            parsed.atlas ||
            previous.atlas,
        }))

        if (parsed.content) {
          setRevisionReady(true)
          showToast(
            '已识别AI返回的完整最新版，可保存本次优化',
          )
        }
      }
    } catch (error: any) {
      /**
       * 请求失败时撤回本地用户气泡，
       * 输入框保留原文，可直接重新发送。
       */
      setMessages((current) => [
        ...current.filter(
          (message) =>
            message.created_at !==
            localCreatedAt,
        ),
        {
          role: 'assistant',
          content:
            '（出错了：' +
            (error?.message ||
              '请重试；输入内容已经保留') +
            '）',
          created_at: '',
        },
      ])
    } finally {
      setSending(false)
      scrollBottom()
    }
  }

  // ==================== 课本选择 ====================

  const handleTextbookSelected = async (selectedIds: string[]) => {
    setShowTextbookPicker(false)

    if (selectedIds.length === 0) {
      setTextbookContext('')
      setTextbookCount(0)
      setMessages((m) => [
        ...m,
        {
          role: 'assistant',
          content: '📷 已清除课本关联，之后的对话不再参考课本原文。',
          created_at: '',
        },
      ])
      return
    }

    showToast('正在读取课本文字…')

    const texts: string[] = []

    for (const id of selectedIds) {
      try {
        const detail = await getTextbook(id)

        if (detail.ocr_text) {
          texts.push(
            `--- 课本（${detail.textbook_name || ''}·${detail.chapter || ''}·第${detail.page_number || '?'}页）---\n${detail.ocr_text}`,
          )
        } else {
          texts.push(
            `--- 课本（${detail.textbook_name || ''}·第${detail.page_number || '?'}页）---\n[此页尚未OCR识别，请先到课本管理页进行AI识别]`,
          )
        }
      } catch {
        texts.push(
          `--- 课本（ID:${id}）---\n[读取失败]`,
        )
      }
    }

    const ctx = texts.join('\n\n')

    setTextbookContext(ctx)
    setTextbookCount(selectedIds.length)
    setMessages((m) => [
      ...m,
      {
        role: 'assistant',
        content:
          `📷 已关联 ${selectedIds.length} 张课本页。` +
          '从你的下一条消息开始，我会参考这些课本原文内容。\n\n' +
          '💡 你可以直接继续提出优化意见，课本内容会自动附在消息中供AI参考。',
        created_at: '',
      },
    ])

    scrollBottom()
  }

  // ==================== 保存 ====================

  const openSave = () => {
    if (!plan) return

    const lastAi =
      [...messages]
        .reverse()
        .find((m) => m.role === 'assistant')
        ?.content || ''

    const parsed = parseLatestRevision(lastAi)

    /**
     * 内容优先级：
     * 1. workingDraft：正式方案底稿或已识别的最新版；
     * 2. 本轮AI回复中的标准标记；
     * 3. active方案数据库当前正式正文；
     * 4. draft初次生成时最后一条AI回复。
     */
    const content =
      workingDraft.content.trim() ||
      parsed.content ||
      (plan.status === 'active'
        ? plan.content
        : lastAi)

    let atlas =
      workingDraft.atlas.trim() ||
      parsed.atlas ||
      plan.atlas ||
      ''

    // 初次生成老路径没有续作标记时，继续兼容从AI回复中的Markdown表格行提取图谱。
    if (!atlas && plan.status === 'draft') {
      atlas = lastAi
        .split('\n')
        .filter((line) => line.trim().startsWith('|'))
        .join('\n')
    }

    // 已存在未保存的人工修改时优先保留，不能被新AI结果覆盖。
    if (!saveFormDraft.value.trim()) {
      setSf({
        title: plan.title,
        unit_theme: plan.unit_theme || '',
        content,
        atlas,
      })
    }

    setShowSave(true)
  }

  const doSave = async () => {
    if (!plan) return

    if (!sf.content.trim()) {
      showToast('方案正文为空')
      return
    }

    const isRevision = plan.status === 'active'

    setSaving(true)

    try {
      await saveUnitPlan(plan.id, sf)

      // 正式保存成功后清除已消费的会话和保存表单草稿。
      unitChatDraft.clear()
      saveFormDraft.clear()
      setShowSave(false)
      clearSession()
      setView('list')

      showToast(
        isRevision
          ? '本次优化已保存'
          : '已保存为正式方案',
      )

      loadList()
    } catch (e: any) {
      showToast(e?.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  // ==================== 删除 ====================

  const doDelete = async (
    item: UnitPlanListItem,
    e: ReactMouseEvent,
  ) => {
    e.stopPropagation()

    if (
      !window.confirm(
        '确认删除「' +
          item.title +
          '」？此操作不可恢复。',
      )
    ) {
      return
    }

    try {
      await deleteUnitPlan(item.id)
      showToast('已删除')
      loadList()
    } catch (err: any) {
      showToast(err?.message || '删除失败')
    }
  }

  // ==================== 对话视图 ====================

  if (view === 'session' && plan && canEditCurrent) {
    const isRevisionMode = plan.status === 'active'

    return (
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          height: 'calc(100vh - 230px)',
          minHeight: 420,
        }}
      >
        {/* 顶栏 */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            paddingBottom: 12,
            borderBottom: '1px solid ' + C.border,
            marginBottom: 12,
          }}
        >
          <button
            onClick={leaveSession}
            style={btnGhost}
          >
            ← 返回列表
          </button>

          <div style={{ flex: 1, minWidth: 0 }}>
            <div
              style={{
                fontSize: 15,
                fontWeight: 700,
                color: C.textPrimary,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {plan.title}
            </div>

            <div
              style={{
                fontSize: 12,
                color: C.textMuted,
              }}
            >
              {plan.subject} · {plan.grade}
              {plan.volume} · {plan.unit}
              {'　'}
              {isRevisionMode
                ? '✏️ 正式方案续作'
                : '🎓 大单元架构师逐步引导'}

              {plan.course_outline_publisher != null && (
                <span
                  style={{
                    marginLeft: 8,
                    color: '#7C3AED',
                    fontWeight: 600,
                  }}
                >
                  📖 {publisherLabel(plan.course_outline_publisher)}
                </span>
              )}

              {textbookCount > 0 && (
                <span
                  style={{
                    marginLeft: 8,
                    color: '#10B981',
                    fontWeight: 600,
                  }}
                >
                  📷 课本×{textbookCount}
                </span>
              )}
            </div>
          </div>

          <button
            onClick={() => setShowMaterials(true)}
            style={{
              ...btnGhost,
              color: '#7C3AED',
              fontWeight: 600,
            }}
          >
            📚 本单元资料
          </button>

          <button
            onClick={() => setShowTextbookPicker(true)}
            style={btnTextbook}
          >
            📷 上传/选择课本
          </button>

          <button
            onClick={openSave}
            style={btnPrimary}
          >
            {isRevisionMode
              ? '💾 保存本次优化'
              : '💾 保存为正式方案'}
          </button>
        </div>

        {/* 正式方案续作提示 */}
        {isRevisionMode && (
          <div
            style={{
              padding: '10px 13px',
              marginBottom: 12,
              borderRadius: 9,
              border: revisionReady
                ? '1px solid #86EFAC'
                : '1px solid #BFDBFE',
              background: revisionReady
                ? '#F0FDF4'
                : '#EFF6FF',
              color: revisionReady
                ? '#166534'
                : '#1D4ED8',
              fontSize: 12.5,
              lineHeight: 1.6,
            }}
          >
            {revisionReady ? (
              <>
                ✅ 已识别AI返回的完整最新版正文。
                点击“保存本次优化”可预览、微调并覆盖当前正式版本。
              </>
            ) : (
              <>
                ✏️ 当前为正式方案续作模式。AI会以数据库中已经保存的正文和图谱为底稿，
                不会退回早期草案。提出修改意见后，平台会自动识别AI返回的完整最新版；
                即使暂未识别，也可以打开保存窗口手工编辑。
              </>
            )}
          </div>
        )}

        {/* 消息列表 */}
        <div
          style={{
            flex: 1,
            overflowY: 'auto',
            padding: '4px 2px',
          }}
        >
          {messages.map((m, i) => (
            <div
              key={i}
              style={{
                display: 'flex',
                justifyContent:
                  m.role === 'user'
                    ? 'flex-end'
                    : 'flex-start',
                marginBottom: 14,
              }}
            >
              <div
                style={{
                  maxWidth: '82%',
                  padding: '10px 14px',
                  borderRadius: 12,
                  fontSize: 13.5,
                  lineHeight: 1.7,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  background:
                    m.role === 'user'
                      ? C.primary
                      : C.white,
                  color:
                    m.role === 'user'
                      ? '#fff'
                      : C.textPrimary,
                  border:
                    m.role === 'user'
                      ? 'none'
                      : '1px solid ' + C.border,
                }}
              >
                {m.content}
              </div>
            </div>
          ))}

          {sending && (
            <div
              style={{
                display: 'flex',
                justifyContent: 'flex-start',
                marginBottom: 14,
              }}
            >
              <div
                style={{
                  padding: '10px 14px',
                  borderRadius: 12,
                  fontSize: 13,
                  color: C.textMuted,
                  background: C.white,
                  border: '1px solid ' + C.border,
                }}
              >
                架构师思考中…
              </div>
            </div>
          )}

          <div ref={msgEndRef} />
        </div>

        {/* 课本关联提示 */}
        {textbookCount > 0 && (
          <div
            style={{
              padding: '6px 14px',
              background: '#ECFDF5',
              borderRadius: 8,
              fontSize: 12,
              color: '#059669',
              margin: '4px 0',
              display: 'flex',
              alignItems: 'center',
              gap: 8,
            }}
          >
            <span>
              📷 已关联 {textbookCount} 张课本页，
              每条消息会自动附课本原文供AI参考
            </span>

            <button
              onClick={() => {
                setTextbookContext('')
                setTextbookCount(0)
                showToast('已清除课本关联')
              }}
              style={{
                background: 'none',
                border: 'none',
                color: '#6B7280',
                cursor: 'pointer',
                fontSize: 12,
              }}
            >
              ✕ 清除
            </button>
          </div>
        )}

        {/* 输入区 */}
        <div
          style={{
            display: 'flex',
            gap: 10,
            paddingTop: 12,
            borderTop: '1px solid ' + C.border,
            marginTop: 8,
          }}
        >
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (unitChatDraft.handleKeyDown(e)) {
                return
              }

              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                doSend()
              }
            }}
            placeholder={
              isRevisionMode
                ? '说明需要优化的内容，例如：保留整体结构，只加强评价设计和课时衔接…'
                : '确认 / 补充 / 让架构师按你的意见改这一步…（Enter 发送，Shift+Enter 换行）'
            }
            rows={2}
            style={{
              ...inputStyle,
              resize: 'vertical',
              flex: 1,
            }}
          />

          <button
            onClick={doSend}
            disabled={sending || !input.trim()}
            style={{
              ...btnPrimary,
              opacity:
                sending || !input.trim()
                  ? 0.5
                  : 1,
            }}
          >
            发送
          </button>
        </div>

        {/* 保存弹窗 */}
        {showSave && (
          <SaveModal
            sf={sf}
            setSf={setSf}
            saving={saving}
            isRevision={isRevisionMode}
            revisionReady={revisionReady}
            handleDraftKeyDown={saveFormDraft.handleKeyDown}
            onCancel={() => setShowSave(false)}
            onSave={doSave}
          />
        )}

        {/* 本单元资料管理弹窗 */}
        {showMaterials && (
          <UnitPlanMaterialsModal
            unitPlanId={plan.id}
            subject={plan.subject}
            grade={plan.grade}
            onCancel={() => setShowMaterials(false)}
          />
        )}

        {/* 课本选择弹窗 */}
        {showTextbookPicker && (
          <TextbookPickerModal
            subject={plan.subject}
            grade={plan.grade}
            onConfirm={handleTextbookSelected}
            onCancel={() => setShowTextbookPicker(false)}
          />
        )}

        {toast && <Toast text={toast} />}
      </div>
    )
  }

  // ==================== 列表视图 ====================

  return (
    <div>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 14,
        }}
      >
        <div
          style={{
            fontSize: 13,
            color: C.textSecondary,
          }}
        >
          大单元工坊产出的整单元教学设计方案。
          {canCreate
            ? '由学科负责人逐步生成；创建者可随时重新进入继续优化。'
            : '你可以查看本组、学校或全局已经发布的方案。'}
        </div>

        {canCreate && (
          <button
            onClick={() => setShowNew(true)}
            style={btnPrimary}
          >
            ＋ 新建单元方案
          </button>
        )}
      </div>

      {loading ? (
        <div
          style={{
            padding: 40,
            textAlign: 'center',
            color: C.textMuted,
            fontSize: 13,
          }}
        >
          加载中…
        </div>
      ) : list.length === 0 ? (
        <div
          style={{
            padding: 48,
            textAlign: 'center',
            background: C.white,
            borderRadius: 12,
            border: '1px dashed ' + C.border,
          }}
        >
          <div
            style={{
              fontSize: 30,
              marginBottom: 10,
            }}
          >
            🗂️
          </div>

          <div
            style={{
              fontSize: 14,
              color: C.textSecondary,
            }}
          >
            还没有单元方案
            {canCreate
              ? '，点右上角「新建单元方案」开始。'
              : '。'}
          </div>
        </div>
      ) : (
        <div
          style={{
            display: 'grid',
            gap: 12,
          }}
        >
          {list.map((it) => {
            const sb = scopeBadge(it.scope)
            const st = statusBadge(it.status)

            return (
              <div
                key={it.id}
                onClick={() => openExisting(it)}
                style={{
                  padding: '14px 16px',
                  background: C.white,
                  border: '1px solid ' + C.border,
                  borderRadius: 12,
                  cursor: 'pointer',
                  transition: 'all 140ms ease',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.borderColor = C.primary
                  e.currentTarget.style.boxShadow =
                    '0 2px 10px rgba(79,123,232,0.10)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.borderColor = C.border
                  e.currentTarget.style.boxShadow = 'none'
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    marginBottom: 6,
                  }}
                >
                  <span
                    style={{
                      fontSize: 11,
                      padding: '2px 8px',
                      borderRadius: 6,
                      background: sb.bg,
                      color: sb.c,
                      fontWeight: 600,
                    }}
                  >
                    {sb.t}
                  </span>

                  <span
                    style={{
                      fontSize: 11,
                      padding: '2px 8px',
                      borderRadius: 6,
                      background: st.bg,
                      color: st.c,
                      fontWeight: 600,
                    }}
                  >
                    {st.t}
                  </span>

                  {it.scope_name && (
                    <span
                      style={{
                        fontSize: 12,
                        color: C.textMuted,
                      }}
                    >
                      {it.scope_name}
                    </span>
                  )}

                  <div style={{ flex: 1 }} />

                  <button
                    onClick={(e) => doDelete(it, e)}
                    style={{
                      background: 'none',
                      border: 'none',
                      color: C.textMuted,
                      cursor: 'pointer',
                      fontSize: 13,
                    }}
                  >
                    🗑
                  </button>
                </div>

                <div
                  style={{
                    fontSize: 15,
                    fontWeight: 700,
                    color: C.textPrimary,
                    marginBottom: 4,
                  }}
                >
                  {it.title}
                </div>

                <div
                  style={{
                    fontSize: 12.5,
                    color: C.textSecondary,
                  }}
                >
                  {it.subject} · {it.grade}
                  {it.volume} · {it.unit}
                  {it.unit_theme
                    ? '　主题：' + it.unit_theme
                    : ''}
                  {it.creator_name
                    ? '　· ' + it.creator_name
                    : ''}
                </div>

                <div
                  style={{
                    marginTop: 8,
                    fontSize: 11.5,
                    color: C.textMuted,
                  }}
                >
                  点击打开
                  {it.status === 'active'
                    ? ' · 创建者可继续优化，其他老师只读查看'
                    : ' · 继续完成逐步设计'}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* 新建弹窗 */}
      {showNew && (
        <ModalShell
          title="新建单元方案"
          onCancel={() => setShowNew(false)}
        >
          <div
            style={{
              display: 'grid',
              gap: 12,
            }}
          >
            <div>
              <label style={labelStyle}>
                归属（谁能看到这份方案）
              </label>

              <select
                value={nf.scopeKey}
                onChange={(e) =>
                  setNf((s) => ({
                    ...s,
                    scopeKey: e.target.value,
                  }))
                }
                style={inputStyle}
              >
                {scopeOptions.map((o) => (
                  <option key={o.key} value={o.key}>
                    {o.label}
                  </option>
                ))}
              </select>

              {nf.scopeKey === 'system:' && (
                <div
                  style={{
                    fontSize: 11,
                    color: '#7C3AED',
                    marginTop: 4,
                  }}
                >
                  🌐 全局方案对所有学校的老师可见。
                </div>
              )}
            </div>

            <div
              style={{
                display: 'grid',
                gridTemplateColumns: '1fr 1fr 1fr',
                gap: 10,
              }}
            >
              <div>
                <label style={labelStyle}>学科</label>
                <select
                  value={nf.subject}
                  onChange={(e) =>
                    setNf((s) => ({
                      ...s,
                      subject: e.target.value,
                    }))
                  }
                  style={inputStyle}
                >
                  {SUBJECTS.map((x) => (
                    <option key={x} value={x}>
                      {x}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label style={labelStyle}>年级</label>
                <select
                  value={nf.grade}
                  onChange={(e) =>
                    setNf((s) => ({
                      ...s,
                      grade: e.target.value,
                    }))
                  }
                  style={inputStyle}
                >
                  {GRADES.map((x) => (
                    <option key={x} value={x}>
                      {x}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label style={labelStyle}>册次</label>
                <select
                  value={nf.volume}
                  onChange={(e) =>
                    setNf((s) => ({
                      ...s,
                      volume: e.target.value,
                    }))
                  }
                  style={inputStyle}
                >
                  {VOLUMES.map((x) => (
                    <option key={x} value={x}>
                      {x}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div>
              <label style={labelStyle}>
                课程大纲教材版本（可选）
              </label>

              {publishersLoading ? (
                <div
                  style={{
                    fontSize: 12,
                    color: C.textMuted,
                    padding: '6px 0',
                  }}
                >
                  正在查询该学科年级可用的课程大纲…
                </div>
              ) : availablePublishers.length === 0 ? (
                <div
                  style={{
                    fontSize: 12,
                    color: C.textMuted,
                    padding: '6px 0',
                  }}
                >
                  该学科年级暂无课程大纲，本次备课不关联大纲；
                  如需大纲支撑请联系管理员上传。
                </div>
              ) : (
                <>
                  <select
                    value={nf.publisher}
                    onChange={(e) =>
                      setNf((s) => ({
                        ...s,
                        publisher: e.target.value,
                      }))
                    }
                    style={inputStyle}
                  >
                    <option value={PUBLISHER_NONE}>
                      不关联课程大纲
                    </option>

                    {availablePublishers.map((p) => (
                      <option
                        key={p || '__generic__'}
                        value={p}
                      >
                        {publisherLabel(p)}
                      </option>
                    ))}
                  </select>

                  <div
                    style={{
                      fontSize: 11,
                      color: C.textMuted,
                      marginTop: 4,
                    }}
                  >
                    选定后AI将严格按该版本大纲定位本单元篇目与课时。
                    会话建立时定版，换版请新建会话。
                  </div>
                </>
              )}
            </div>

            <div>
              <label style={labelStyle}>
                单元（如：第二单元 / 寓言单元）
              </label>

              <input
                value={nf.unit}
                onKeyDown={newPlanDraft.handleKeyDown}
                onChange={(e) =>
                  setNf((s) => ({
                    ...s,
                    unit: e.target.value,
                  }))
                }
                style={inputStyle}
                placeholder="本次要做整体设计的单元"
              />
            </div>

            <div>
              <label style={labelStyle}>
                标题（可选，留空自动生成）
              </label>

              <input
                value={nf.title}
                onKeyDown={newPlanDraft.handleKeyDown}
                onChange={(e) =>
                  setNf((s) => ({
                    ...s,
                    title: e.target.value,
                  }))
                }
                style={inputStyle}
                placeholder="如：三下第二单元 寓言故事 大单元设计"
              />
            </div>
          </div>

          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              gap: 10,
              marginTop: 18,
            }}
          >
            <button
              onClick={() => setShowNew(false)}
              style={btnGhost}
            >
              取消
            </button>

            <button
              onClick={doStart}
              disabled={starting}
              style={{
                ...btnPrimary,
                opacity: starting ? 0.6 : 1,
              }}
            >
              {starting
                ? '正在准备…'
                : '开始逐步设计'}
            </button>
          </div>
        </ModalShell>
      )}

      {/* 只读详情弹窗 */}
      {viewPlan && (
        <ModalShell
          title={viewPlan.title}
          wide
          onCancel={() => setViewPlan(null)}
        >
          <div
            style={{
              fontSize: 12,
              color: C.textMuted,
              marginBottom: 10,
            }}
          >
            {viewPlan.subject} · {viewPlan.grade}
            {viewPlan.volume} · {viewPlan.unit}

            {viewPlan.unit_theme
              ? '　主题：' + viewPlan.unit_theme
              : ''}

            {viewPlan.course_outline_publisher != null
              ? '　📖 大纲版本：' +
                publisherLabel(
                  viewPlan.course_outline_publisher,
                )
              : ''}
          </div>

          <div
            style={{
              maxHeight: '60vh',
              overflowY: 'auto',
            }}
          >
            <div
              style={{
                fontSize: 13,
                fontWeight: 700,
                color: C.textPrimary,
                margin: '6px 0',
              }}
            >
              方案文档
            </div>

            <div
              style={{
                whiteSpace: 'pre-wrap',
                fontSize: 13,
                lineHeight: 1.7,
                color: C.textPrimary,
              }}
            >
              {viewPlan.content || '（空）'}
            </div>

            {viewPlan.atlas && (
              <>
                <div
                  style={{
                    fontSize: 13,
                    fontWeight: 700,
                    color: C.textPrimary,
                    margin: '16px 0 6px',
                  }}
                >
                  单元整体设计图谱
                </div>

                <div
                  style={{
                    whiteSpace: 'pre-wrap',
                    fontSize: 12.5,
                    lineHeight: 1.7,
                    color: C.textSecondary,
                    fontFamily: 'monospace',
                  }}
                >
                  {viewPlan.atlas}
                </div>
              </>
            )}
          </div>

          <div
            style={{
              padding: '9px 11px',
              marginTop: 14,
              borderRadius: 8,
              background: '#F9FAFB',
              color: C.textMuted,
              fontSize: 12,
            }}
          >
            你当前以只读方式查看该方案。只有方案创建者可以重新进入AI会话继续优化。
          </div>

          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              marginTop: 16,
            }}
          >
            <button
              onClick={() => setShowMaterials(true)}
              style={{
                ...btnGhost,
                color: '#7C3AED',
                fontWeight: 600,
                marginRight: 10,
              }}
            >
              📚 查看参考资料
            </button>

            <button
              onClick={() => setViewPlan(null)}
              style={btnGhost}
            >
              关闭
            </button>
          </div>
        </ModalShell>
      )}

      {showMaterials && viewPlan && (
        <UnitPlanMaterialsModal
          unitPlanId={viewPlan.id}
          subject={viewPlan.subject}
          grade={viewPlan.grade}
          onCancel={() => setShowMaterials(false)}
        />
      )}

      {toast && <Toast text={toast} />}
    </div>
  )
}

// ==================== 课本选择弹窗 ====================

function TextbookPickerModal({
  subject,
  grade,
  onConfirm,
  onCancel,
}: {
  subject: string
  grade: string
  onConfirm: (ids: string[]) => void
  onCancel: () => void
}) {
  const [pages, setPages] = useState<TextbookListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Set<string>>(new Set())

  useEffect(() => {
    let cancelled = false

    ;(async () => {
      try {
        const resp = await getTextbooks({
          subject,
          grade_range: grade,
          limit: 200,
        })

        if (!cancelled) {
          setPages(resp.pages || [])
        }
      } catch {
        // 查询失败时显示空列表。
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    })()

    return () => {
      cancelled = true
    }
  }, [subject, grade])

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)

      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }

      return next
    })
  }

  return (
    <ModalShell
      title={`选择课本页（${subject} · ${grade}）`}
      wide
      onCancel={onCancel}
    >
      <div
        style={{
          fontSize: 12,
          color: C.textMuted,
          marginBottom: 10,
        }}
      >
        选择要参考的课本页面，AI会读取其中的文字内容。
        如需上传新课本图片，请先到“课本管理”页面上传并进行AI识别。
      </div>

      <div
        style={{
          maxHeight: '50vh',
          overflowY: 'auto',
          border: '1px solid ' + C.border,
          borderRadius: 8,
          padding: 8,
        }}
      >
        {loading ? (
          <div
            style={{
              padding: 30,
              textAlign: 'center',
              color: C.textMuted,
              fontSize: 13,
            }}
          >
            加载中…
          </div>
        ) : pages.length === 0 ? (
          <div
            style={{
              padding: 30,
              textAlign: 'center',
              color: C.textMuted,
              fontSize: 13,
            }}
          >
            暂无该学科年级的课本图片，请先到课本管理页面上传。
          </div>
        ) : (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns:
                'repeat(auto-fill, minmax(140px, 1fr))',
              gap: 8,
            }}
          >
            {pages.map((p) => {
              const isSelected = selected.has(p.id)

              return (
                <div
                  key={p.id}
                  onClick={() => toggle(p.id)}
                  style={{
                    padding: 8,
                    borderRadius: 8,
                    cursor: 'pointer',
                    textAlign: 'center',
                    border: isSelected
                      ? '2px solid #10B981'
                      : '1px solid ' + C.border,
                    background: isSelected
                      ? '#ECFDF5'
                      : C.white,
                  }}
                >
                  <img
                    src={p.image_url}
                    alt=""
                    style={{
                      width: '100%',
                      height: 80,
                      objectFit: 'cover',
                      borderRadius: 6,
                      marginBottom: 4,
                    }}
                  />

                  <div
                    style={{
                      fontSize: 11,
                      color: C.textPrimary,
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                  >
                    {p.textbook_name}
                  </div>

                  <div
                    style={{
                      fontSize: 10,
                      color: C.textMuted,
                    }}
                  >
                    第{p.page_number}页{' '}
                    {p.has_ocr
                      ? '✅已识别'
                      : '⚠️未识别'}
                  </div>

                  {isSelected && (
                    <div
                      style={{
                        fontSize: 11,
                        color: '#10B981',
                        fontWeight: 700,
                      }}
                    >
                      ✓ 已选
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>

      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginTop: 14,
        }}
      >
        <span
          style={{
            fontSize: 12,
            color: C.textSecondary,
          }}
        >
          已选 {selected.size} 页
        </span>

        <div
          style={{
            display: 'flex',
            gap: 10,
          }}
        >
          <button onClick={onCancel} style={btnGhost}>
            取消
          </button>

          <button
            onClick={() =>
              onConfirm(Array.from(selected))
            }
            style={{
              ...btnPrimary,
              background: '#10B981',
            }}
          >
            {selected.size > 0
              ? `确认关联 ${selected.size} 页`
              : '清除关联'}
          </button>
        </div>
      </div>
    </ModalShell>
  )
}

// ==================== 通用弹窗外壳 ====================

function ModalShell({
  title,
  children,
  onCancel,
  wide,
}: {
  title: string
  children: ReactNode
  onCancel: () => void
  wide?: boolean
}) {
  return (
    <div
      onClick={onCancel}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(17,24,39,0.45)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 9000,
        padding: 20,
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          background: '#fff',
          borderRadius: 14,
          padding: 22,
          width: wide ? 720 : 460,
          maxWidth: '100%',
          boxShadow: '0 12px 48px rgba(0,0,0,0.18)',
        }}
      >
        <div
          style={{
            fontSize: 16,
            fontWeight: 700,
            color: '#1F2937',
            marginBottom: 16,
          }}
        >
          {title}
        </div>

        {children}
      </div>
    </div>
  )
}

// ==================== 保存弹窗 ====================

function SaveModal({
  sf,
  setSf,
  saving,
  isRevision,
  revisionReady,
  handleDraftKeyDown,
  onCancel,
  onSave,
}: {
  sf: {
    title: string
    unit_theme: string
    content: string
    atlas: string
  }
  setSf: Dispatch<
    SetStateAction<{
      title: string
      unit_theme: string
      content: string
      atlas: string
    }>
  >
  saving: boolean
  isRevision: boolean
  revisionReady: boolean
  handleDraftKeyDown: (
    event: ReactKeyboardEvent<HTMLElement>,
  ) => boolean
  onCancel: () => void
  onSave: () => void
}) {
  return (
    <ModalShell
      title={
        isRevision
          ? '保存本次优化'
          : '保存为正式方案'
      }
      wide
      onCancel={onCancel}
    >
      <div
        style={{
          fontSize: 12,
          color: '#6B7280',
          marginBottom: 12,
          lineHeight: 1.6,
        }}
      >
        {isRevision ? (
          revisionReady ? (
            <>
              已自动填入AI返回的完整最新版正文和图谱。
              你可以继续手工微调，确认后覆盖当前正式版本。
            </>
          ) : (
            <>
              当前尚未识别到AI的完整最新版标记，因此先填入数据库中现有正式方案。
              你可以在这里直接修改后保存，也可以返回对话继续要求AI输出完整最新版。
            </>
          )
        ) : (
          <>
            已自动填入最后一步的方案与图谱。
            确认或微调后保存，保存后该方案对归属范围内的老师可见。
          </>
        )}
      </div>

      <div
        style={{
          display: 'grid',
          gap: 12,
        }}
      >
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: 10,
          }}
        >
          <div>
            <label style={labelStyle}>标题</label>
            <input
              value={sf.title}
              onKeyDown={handleDraftKeyDown}
              onChange={(e) =>
                setSf((s) => ({
                  ...s,
                  title: e.target.value,
                }))
              }
              style={inputStyle}
            />
          </div>

          <div>
            <label style={labelStyle}>
              单元任务主题（可选）
            </label>
            <input
              value={sf.unit_theme}
              onKeyDown={handleDraftKeyDown}
              onChange={(e) =>
                setSf((s) => ({
                  ...s,
                  unit_theme: e.target.value,
                }))
              }
              style={inputStyle}
            />
          </div>
        </div>

        <div>
          <label style={labelStyle}>方案文档</label>
          <textarea
            value={sf.content}
            onKeyDown={handleDraftKeyDown}
            onChange={(e) =>
              setSf((s) => ({
                ...s,
                content: e.target.value,
              }))
            }
            rows={12}
            style={{
              ...inputStyle,
              resize: 'vertical',
              fontFamily: 'inherit',
            }}
          />
        </div>

        <div>
          <label style={labelStyle}>
            单元整体设计图谱（表格，可留空）
          </label>
          <textarea
            value={sf.atlas}
            onKeyDown={handleDraftKeyDown}
            onChange={(e) =>
              setSf((s) => ({
                ...s,
                atlas: e.target.value,
              }))
            }
            rows={6}
            style={{
              ...inputStyle,
              resize: 'vertical',
              fontFamily: 'monospace',
            }}
          />
        </div>
      </div>

      <div
        style={{
          marginTop: 10,
          fontSize: 11,
          color: C.textMuted,
          lineHeight: 1.5,
        }}
      >
        已自动保存未提交内容 · Ctrl/Command+Z恢复误删
      </div>

      <div
        style={{
          display: 'flex',
          justifyContent: 'flex-end',
          gap: 10,
          marginTop: 18,
        }}
      >
        <button onClick={onCancel} style={btnGhost}>
          取消
        </button>

        <button
          onClick={onSave}
          disabled={saving}
          style={{
            ...btnPrimary,
            opacity: saving ? 0.6 : 1,
          }}
        >
          {saving
            ? '保存中…'
            : isRevision
              ? '确认保存本次优化'
              : '确认保存'}
        </button>
      </div>
    </ModalShell>
  )
}

// ==================== Toast ====================

function Toast({ text }: { text: string }) {
  return (
    <div
      style={{
        position: 'fixed',
        bottom: 28,
        left: '50%',
        transform: 'translateX(-50%)',
        background: 'rgba(17,24,39,0.92)',
        color: '#fff',
        padding: '10px 20px',
        borderRadius: 10,
        fontSize: 13,
        zIndex: 9999,
        boxShadow: '0 6px 24px rgba(0,0,0,0.2)',
      }}
    >
      {text}
    </div>
  )
}
