/**
 * 课件方案编辑器 — IndexEditor.tsx v3.1
 *
 * 本次改造（拖拽排序）：
 *   1. 每张方案卡片增加独立拖拽手柄，拖到目标页面后即可调整顺序。
 *   2. 保留上移/下移按钮，兼顾键盘操作和不方便拖拽的使用环境。
 *   3. 排序先进行前端乐观更新，再提交完整页面ID顺序给后端。
 *   4. 后端保存失败时自动恢复排序前的完整页面数组，不再静默留下错误顺序。
 *   5. 保存过程中锁定全部排序入口，防止重复拖拽和并发排序。
 *   6. 编辑页面、方案生成或AI修改过程中禁止排序，避免页码身份与异步页面流发生竞争。
 *
 * 既有改造（内容丰富度 + 降认知负担）：
 *   1. 把不直观的「复杂度 (1-5) 数字框」改为「内容丰富度」三档大按钮：
 *      🌱 精简 / 📖 适中 / 🎯 充实，并配一句引导文案。
 *      底层仍写 estimated_complexity（精简=2 / 适中=3 / 充实=5），后端无感、无需改接口。
 *   2. 把专业字段「交互类型 / 视觉形式」折叠进「⚙ 高级（可不改）」区，默认收起。
 *   3. 内容概要文本框上方增加教师可理解的引导文案。
 *   4. 卡片展示态突出内容丰富度，其余技术维度弱化为一行小灰字。
 *
 * 两层架构展示：
 *   - 普通用户：看到翻译后的方案（目的、概要、内容丰富度等人话信息）。
 *   - admin：额外可展开查看层1 AOCI技术索引原文。
 */

import { useState } from 'react'
import type { DragEvent } from 'react'
import {
  updateCWPageIndex,
  addCWPage,
  deleteCWPage,
  reorderCWPages,
  CW_INTERACTION_TYPES,
  CW_VISUAL_FORMATS,
  CW_COGNITIVE_LEVELS,
} from '@/api/coursewares'
import type { CoursewarePage } from '@/api/coursewares'
import {
  buildCoursewarePageReorderPlan,
  findCoursewarePageIndex,
} from './indexEditorReorder'

// ==================== 颜色常量 ====================

const C = {
  primary: '#F59E0B',
  primaryBg: 'rgba(245,158,11,0.08)',
  primaryBorder: 'rgba(245,158,11,0.3)',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  danger: '#EF4444',
  success: '#059669',
  white: '#fff',
}

// ==================== 内容丰富度三档（老师可见的人话） ====================

/**
 * value即落库的estimated_complexity：
 * - 精简=2
 * - 适中=3
 * - 充实=5
 *
 * 三个值均在后端合法范围1至5内，并与后端内容丰富度归档逻辑对齐。
 */
interface RichnessOption {
  value: number
  emoji: string
  label: string
  desc: string
  color: string
  bg: string
}

const RICHNESS_OPTIONS: RichnessOption[] = [
  {
    value: 2,
    emoji: '🌱',
    label: '精简',
    desc: '要点为主，简洁留白',
    color: '#059669',
    bg: '#D1FAE5',
  },
  {
    value: 3,
    emoji: '📖',
    label: '适中',
    desc: '标准图文讲解',
    color: '#0891B2',
    bg: '#CFFAFE',
  },
  {
    value: 5,
    emoji: '🎯',
    label: '充实',
    desc: '详尽展开，多举例',
    color: '#DC2626',
    bg: '#FEE2E2',
  },
]

/**
 * 把任意estimated_complexity归并到最近的教师可见档位。
 *
 * 规则与后端一致：
 * - 大于等于4：充实
 * - 等于3：适中
 * - 小于等于2及异常值：精简
 */
function richnessOf(
  complexity: number,
): RichnessOption {
  if (complexity >= 4) {
    return RICHNESS_OPTIONS[2]
  }

  if (complexity === 3) {
    return RICHNESS_OPTIONS[1]
  }

  return RICHNESS_OPTIONS[0]
}

// ==================== Props ====================

interface IndexEditorProps {
  coursewareId: string
  pages: CoursewarePage[]
  onPagesChange: (pages: CoursewarePage[]) => void
  loading?: boolean
  isAdmin?: boolean
  indexOverview?: string

  /**
   * 外部业务正在生成、AI修改或确认方案时禁止页面排序。
   *
   * 该字段只负责前端交互收窄；
   * 最终写权限和课件状态仍由后端重新校验。
   */
  reorderDisabled?: boolean
}

type ReorderNotice = {
  type: 'success' | 'error'
  text: string
}

export default function IndexEditor({
  coursewareId,
  pages,
  onPagesChange,
  loading,
  isAdmin,
  indexOverview,
  reorderDisabled = false,
}: IndexEditorProps) {
  const [editingPage, setEditingPage] =
    useState<number | null>(null)

  const [editForm, setEditForm] =
    useState<Record<string, string | number>>({})

  const [saving, setSaving] =
    useState(false)

  const [addingPage, setAddingPage] =
    useState(false)

  const [expandedIndex, setExpandedIndex] =
    useState<Set<number>>(new Set())

  const [showAdvanced, setShowAdvanced] =
    useState(false)

  /**
   * 页面排序状态。
   */
  const [reordering, setReordering] =
    useState(false)

  const [draggedPageId, setDraggedPageId] =
    useState<string | null>(null)

  const [dragOverPageId, setDragOverPageId] =
    useState<string | null>(null)

  const [reorderNotice, setReorderNotice] =
    useState<ReorderNotice | null>(null)

  /**
   * 编辑期间禁止排序。
   *
   * 编辑状态仍以page_number定位当前表单；
   * 若编辑期间同时重排，page_number变化会导致保存目标不明确。
   */
  const reorderLocked =
    reorderDisabled ||
    reordering ||
    editingPage !== null ||
    saving ||
    addingPage

  const reorderLockReason = (() => {
    if (reorderDisabled) {
      return '方案正在生成、修改或确认，暂不可调整顺序'
    }

    if (reordering) {
      return '正在保存页面顺序'
    }

    if (editingPage !== null || saving) {
      return '请先保存或取消当前页面编辑'
    }

    if (addingPage) {
      return '正在添加页面'
    }

    return '按住手柄拖到目标页面'
  })()

  // ==================== admin展开/折叠层1索引 ====================

  const toggleIndexExpand = (
    pageNum: number,
  ) => {
    setExpandedIndex(previous => {
      const next = new Set(previous)

      if (next.has(pageNum)) {
        next.delete(pageNum)
      } else {
        next.add(pageNum)
      }

      return next
    })
  }

  // ==================== 开始编辑 ====================

  const startEdit = (
    page: CoursewarePage,
  ) => {
    if (reordering) {
      return
    }

    setEditingPage(page.page_number)
    setShowAdvanced(false)
    setEditForm({
      title: page.title,
      purpose: page.purpose,
      content_summary: page.content_summary,
      interaction_type: page.interaction_type,
      visual_format: page.visual_format,
      media_requirements: page.media_requirements,
      estimated_complexity: page.estimated_complexity,
    })
  }

  // ==================== 保存编辑 ====================

  const saveEdit = async () => {
    if (
      editingPage === null ||
      reordering
    ) {
      return
    }

    setSaving(true)

    try {
      await updateCWPageIndex(
        coursewareId,
        editingPage,
        {
          title: String(editForm.title || ''),
          purpose: String(editForm.purpose || ''),
          content_summary: String(
            editForm.content_summary || '',
          ),
          interaction_type: String(
            editForm.interaction_type || '',
          ),
          visual_format: String(
            editForm.visual_format || '',
          ),
          media_requirements: String(
            editForm.media_requirements || '',
          ),
          estimated_complexity:
            Number(editForm.estimated_complexity) || 3,
        },
      )

      const updated = pages.map(page =>
        page.page_number === editingPage
          ? {
              ...page,
              title: String(editForm.title || ''),
              purpose: String(editForm.purpose || ''),
              content_summary: String(
                editForm.content_summary || '',
              ),
              interaction_type: String(
                editForm.interaction_type || '',
              ),
              visual_format: String(
                editForm.visual_format || '',
              ),
              media_requirements: String(
                editForm.media_requirements || '',
              ),
              estimated_complexity:
                Number(
                  editForm.estimated_complexity,
                ) || 3,
            }
          : page,
      )

      onPagesChange(updated)
      setEditingPage(null)
    } catch {
      alert('保存失败')
    } finally {
      setSaving(false)
    }
  }

  // ==================== 删除页面 ====================

  const handleDelete = async (
    pageNum: number,
  ) => {
    if (reordering) {
      return
    }

    if (
      !window.confirm(
        `确定删除第 ${pageNum} 页？`,
      )
    ) {
      return
    }

    try {
      await deleteCWPage(
        coursewareId,
        pageNum,
      )

      const remaining = pages.filter(
        page => page.page_number !== pageNum,
      )

      const renumbered = remaining.map(
        (page, index) => ({
          ...page,
          page_number: index + 1,
        }),
      )

      onPagesChange(renumbered)
    } catch {
      alert('删除失败')
    }
  }

  // ==================== 添加页面 ====================

  const handleAdd = async () => {
    if (reordering) {
      return
    }

    setAddingPage(true)

    try {
      const newPage = await addCWPage(
        coursewareId,
        {
          title: `第 ${pages.length + 1} 页`,
          purpose: '',
          content_summary: '',
          interaction_type: 'static',
          visual_format: 'text_heavy',
        },
      )

      onPagesChange([
        ...pages,
        newPage,
      ])
    } catch {
      alert('添加失败')
    } finally {
      setAddingPage(false)
    }
  }

  // ==================== 页面排序统一保存入口 ====================

  /**
   * 执行一次页面排序。
   *
   * 流程：
   * 1. 基于当前完整页面数组构建确定性目标顺序。
   * 2. 保存排序前快照。
   * 3. 乐观更新界面，让拖拽反馈立即可见。
   * 4. 把完整页面ID顺序提交给后端事务接口。
   * 5. 请求失败时恢复排序前快照并明确提示。
   */
  const persistPageOrder = async (
    fromIndex: number,
    toIndex: number,
  ) => {
    if (reorderLocked) {
      return
    }

    let plan

    try {
      plan = buildCoursewarePageReorderPlan(
        pages,
        fromIndex,
        toIndex,
      )
    } catch (error) {
      setReorderNotice({
        type: 'error',
        text:
          error instanceof Error
            ? error.message
            : '页面顺序数据异常，请刷新后重试',
      })
      return
    }

    if (!plan) {
      return
    }

    /**
     * 页面对象做浅复制，确保失败回滚使用独立快照。
     */
    const previousPages = pages.map(
      page => ({ ...page }),
    )

    setReordering(true)
    setReorderNotice(null)

    /**
     * 先乐观更新，拖拽松手后页面立即落位。
     */
    onPagesChange(plan.pages)

    try {
      await reorderCWPages(
        coursewareId,
        plan.pageIds,
      )

      setReorderNotice({
        type: 'success',
        text: `已将“${
          plan.movedPage.title || '未命名页面'
        }”调整到第 ${plan.toIndex + 1} 页`,
      })
    } catch {
      /**
       * 后端事务失败时恢复原数组。
       *
       * 后端本身也会整体回滚数据库修改，
       * 前后端重新保持同一顺序。
       */
      onPagesChange(previousPages)

      setReorderNotice({
        type: 'error',
        text: '页面顺序保存失败，已自动恢复原顺序，请稍后重试',
      })
    } finally {
      setReordering(false)
    }
  }

  // ==================== 上移/下移 ====================

  const movePage = (
    index: number,
    direction: 'up' | 'down',
  ) => {
    const target =
      direction === 'up'
        ? index - 1
        : index + 1

    void persistPageOrder(
      index,
      target,
    )
  }

  // ==================== 原生拖拽排序 ====================

  const handleDragStart = (
    event: DragEvent<HTMLElement>,
    pageId: string,
  ) => {
    if (reorderLocked) {
      event.preventDefault()
      return
    }

    const normalizedPageId = pageId.trim()

    if (!normalizedPageId) {
      event.preventDefault()
      setReorderNotice({
        type: 'error',
        text: '当前页面缺少稳定ID，请刷新课件后重试',
      })
      return
    }

    setReorderNotice(null)
    setDraggedPageId(normalizedPageId)
    setDragOverPageId(normalizedPageId)

    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData(
      'text/plain',
      normalizedPageId,
    )
  }

  const handleDragOver = (
    event: DragEvent<HTMLDivElement>,
    targetPageId: string,
  ) => {
    if (reorderLocked) {
      return
    }

    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'

    const normalizedTargetPageId =
      targetPageId.trim()

    if (normalizedTargetPageId) {
      setDragOverPageId(
        normalizedTargetPageId,
      )
    }
  }

  const handleDrop = (
    event: DragEvent<HTMLDivElement>,
    targetPageId: string,
  ) => {
    event.preventDefault()

    if (reorderLocked) {
      setDraggedPageId(null)
      setDragOverPageId(null)
      return
    }

    const sourcePageId = (
      draggedPageId ||
      event.dataTransfer.getData('text/plain')
    ).trim()

    const normalizedTargetPageId =
      targetPageId.trim()

    setDraggedPageId(null)
    setDragOverPageId(null)

    const fromIndex =
      findCoursewarePageIndex(
        pages,
        sourcePageId,
      )

    const toIndex =
      findCoursewarePageIndex(
        pages,
        normalizedTargetPageId,
      )

    if (
      fromIndex < 0 ||
      toIndex < 0 ||
      fromIndex === toIndex
    ) {
      return
    }

    void persistPageOrder(
      fromIndex,
      toIndex,
    )
  }

  const handleDragEnd = () => {
    setDraggedPageId(null)
    setDragOverPageId(null)
  }

  if (loading) {
    return (
      <div
        style={{
          textAlign: 'center',
          padding: '60px 0',
          color: C.textMuted,
        }}
      >
        加载中...
      </div>
    )
  }

  return (
    <div>
      {/* 课件脉络概述 */}
      {indexOverview && (
        <div
          style={{
            padding: '14px 18px',
            borderRadius: '10px',
            marginBottom: '16px',
            background:
              'linear-gradient(135deg, rgba(245,158,11,0.06), rgba(239,68,68,0.04))',
            border:
              '1px solid rgba(245,158,11,0.2)',
          }}
        >
          <div
            style={{
              fontSize: '13px',
              fontWeight: 600,
              color: '#D97706',
              marginBottom: '6px',
            }}
          >
            📋 课件脉络
          </div>
          <div
            style={{
              fontSize: '13px',
              color: '#4B5563',
              lineHeight: '1.6',
            }}
          >
            {indexOverview}
          </div>
        </div>
      )}

      {/* 页面数量统计和添加入口 */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: '12px',
          marginBottom: '10px',
          flexWrap: 'wrap',
        }}
      >
        <div>
          <div
            style={{
              fontSize: '14px',
              color: C.textSecondary,
            }}
          >
            共{' '}
            <strong
              style={{
                color: C.primary,
              }}
            >
              {pages.length}
            </strong>{' '}
            页
          </div>

          <div
            style={{
              marginTop: '3px',
              fontSize: '12px',
              color: reorderLocked
                ? C.textMuted
                : C.textSecondary,
            }}
          >
            {reordering
              ? '⏳ 正在保存页面顺序...'
              : `⠿ ${reorderLockReason}`}
          </div>
        </div>

        <button
          onClick={handleAdd}
          disabled={
            addingPage ||
            reordering
          }
          style={{
            padding: '6px 16px',
            borderRadius: '8px',
            border: `1px dashed ${C.primary}`,
            background: C.primaryBg,
            color: C.primary,
            fontSize: '13px',
            fontWeight: 600,
            cursor:
              addingPage || reordering
                ? 'default'
                : 'pointer',
            opacity:
              addingPage || reordering
                ? 0.6
                : 1,
          }}
        >
          {addingPage
            ? '添加中...'
            : '+ 添加页面'}
        </button>
      </div>

      {/* 排序结果提示 */}
      {reorderNotice && (
        <div
          role={
            reorderNotice.type === 'error'
              ? 'alert'
              : 'status'
          }
          style={{
            marginBottom: '12px',
            padding: '9px 12px',
            borderRadius: '8px',
            fontSize: '13px',
            lineHeight: 1.5,
            color:
              reorderNotice.type === 'error'
                ? '#B91C1C'
                : '#047857',
            background:
              reorderNotice.type === 'error'
                ? '#FEF2F2'
                : '#ECFDF5',
            border: `1px solid ${
              reorderNotice.type === 'error'
                ? '#FECACA'
                : '#A7F3D0'
            }`,
          }}
        >
          {reorderNotice.type === 'error'
            ? '❌ '
            : '✅ '}
          {reorderNotice.text}
        </div>
      )}

      {/* 拖拽中的操作提示 */}
      {draggedPageId && (
        <div
          style={{
            marginBottom: '12px',
            padding: '8px 12px',
            borderRadius: '8px',
            fontSize: '12px',
            color: '#92400E',
            background: '#FFFBEB',
            border: '1px solid #FDE68A',
          }}
        >
          正在调整顺序：把页面拖到目标卡片后松开即可。
        </div>
      )}

      {/* 卡片列表 */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: '12px',
        }}
      >
        {pages.map((page, idx) => {
          const isEditing =
            editingPage === page.page_number

          const rich = richnessOf(
            page.estimated_complexity,
          )

          const interaction =
            CW_INTERACTION_TYPES[
              page.interaction_type
            ] || {
              label: page.interaction_type,
              emoji: '📄',
            }

          const visual =
            CW_VISUAL_FORMATS[
              page.visual_format
            ] || {
              label: page.visual_format,
              emoji: '📝',
            }

          const cognitive =
            CW_COGNITIVE_LEVELS[
              page.idx_cognitive_level
            ] || null

          const isIndexExpanded =
            expandedIndex.has(
              page.page_number,
            )

          const normalizedPageId =
            page.id.trim()

          const isDragged =
            Boolean(normalizedPageId) &&
            draggedPageId === normalizedPageId

          const isDragTarget =
            Boolean(
              draggedPageId &&
                normalizedPageId &&
                dragOverPageId ===
                  normalizedPageId &&
                draggedPageId !==
                  normalizedPageId,
            )

          const moveUpDisabled =
            reorderLocked || idx === 0

          const moveDownDisabled =
            reorderLocked ||
            idx === pages.length - 1

          return (
            <div
              key={page.id || idx}
              onDragOver={event =>
                handleDragOver(
                  event,
                  page.id,
                )
              }
              onDrop={event =>
                handleDrop(
                  event,
                  page.id,
                )
              }
              style={{
                position: 'relative',
                background: C.white,
                borderRadius: '12px',
                padding: '16px 20px',
                border: isDragTarget
                  ? `2px dashed ${C.primary}`
                  : `1px solid ${
                      isEditing
                        ? C.primaryBorder
                        : C.border
                    }`,
                boxShadow: isDragTarget
                  ? '0 8px 24px rgba(245,158,11,0.18)'
                  : isEditing
                    ? '0 2px 12px rgba(245,158,11,0.15)'
                    : '0 1px 3px rgba(0,0,0,0.04)',
                opacity: isDragged
                  ? 0.52
                  : 1,
                transform: isDragTarget
                  ? 'translateY(2px)'
                  : 'none',
                transition:
                  'border-color 120ms, box-shadow 120ms, opacity 120ms, transform 120ms',
              }}
            >
              {/* 卡片头部 */}
              <div
                style={{
                  display: 'flex',
                  justifyContent:
                    'space-between',
                  alignItems: 'center',
                  gap: '12px',
                  marginBottom: '10px',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '10px',
                    minWidth: 0,
                    flex: 1,
                  }}
                >
                  {/* 独立拖拽手柄 */}
                  <span
                    role="button"
                    tabIndex={
                      reorderLocked
                        ? -1
                        : 0
                    }
                    aria-label={`拖动第${page.page_number}页调整顺序`}
                    aria-disabled={
                      reorderLocked
                    }
                    title={reorderLockReason}
                    draggable={
                      !reorderLocked &&
                      Boolean(normalizedPageId)
                    }
                    onDragStart={event =>
                      handleDragStart(
                        event,
                        page.id,
                      )
                    }
                    onDragEnd={
                      handleDragEnd
                    }
                    style={{
                      width: '26px',
                      height: '32px',
                      borderRadius: '7px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent:
                        'center',
                      flexShrink: 0,
                      userSelect: 'none',
                      fontSize: '20px',
                      lineHeight: 1,
                      letterSpacing: '-5px',
                      color: reorderLocked
                        ? '#D1D5DB'
                        : C.textMuted,
                      background:
                        draggedPageId ===
                        normalizedPageId
                          ? C.primaryBg
                          : '#F9FAFB',
                      border: `1px solid ${
                        draggedPageId ===
                        normalizedPageId
                          ? C.primaryBorder
                          : C.border
                      }`,
                      cursor: reorderLocked
                        ? 'not-allowed'
                        : draggedPageId ===
                            normalizedPageId
                          ? 'grabbing'
                          : 'grab',
                    }}
                  >
                    ⠿
                  </span>

                  <span
                    style={{
                      width: '28px',
                      height: '28px',
                      borderRadius: '50%',
                      background:
                        'linear-gradient(135deg, #F59E0B, #EF4444)',
                      color: '#fff',
                      fontSize: '13px',
                      fontWeight: 700,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent:
                        'center',
                      flexShrink: 0,
                    }}
                  >
                    {page.page_number}
                  </span>

                  {isEditing ? (
                    <input
                      value={
                        editForm.title || ''
                      }
                      onChange={event =>
                        setEditForm({
                          ...editForm,
                          title:
                            event.target.value,
                        })
                      }
                      style={{
                        fontSize: '15px',
                        fontWeight: 600,
                        border: `1px solid ${C.border}`,
                        borderRadius: '6px',
                        padding: '4px 8px',
                        flex: 1,
                        minWidth: '200px',
                      }}
                    />
                  ) : (
                    <span
                      style={{
                        fontSize: '15px',
                        fontWeight: 600,
                        color: C.textPrimary,
                        minWidth: 0,
                        overflowWrap:
                          'anywhere',
                      }}
                    >
                      {page.title ||
                        '(未命名)'}
                    </span>
                  )}
                </div>

                <div
                  style={{
                    display: 'flex',
                    gap: '4px',
                    alignItems: 'center',
                    flexShrink: 0,
                    flexWrap: 'wrap',
                    justifyContent:
                      'flex-end',
                  }}
                >
                  <button
                    onClick={() =>
                      movePage(idx, 'up')
                    }
                    disabled={
                      moveUpDisabled
                    }
                    title={
                      moveUpDisabled
                        ? reorderLockReason
                        : '上移一页'
                    }
                    style={{
                      background:
                        'transparent',
                      border: 'none',
                      fontSize: '16px',
                      cursor:
                        moveUpDisabled
                          ? 'default'
                          : 'pointer',
                      opacity:
                        moveUpDisabled
                          ? 0.3
                          : 1,
                      padding: '2px 6px',
                    }}
                  >
                    ⬆
                  </button>

                  <button
                    onClick={() =>
                      movePage(idx, 'down')
                    }
                    disabled={
                      moveDownDisabled
                    }
                    title={
                      moveDownDisabled
                        ? reorderLockReason
                        : '下移一页'
                    }
                    style={{
                      background:
                        'transparent',
                      border: 'none',
                      fontSize: '16px',
                      cursor:
                        moveDownDisabled
                          ? 'default'
                          : 'pointer',
                      opacity:
                        moveDownDisabled
                          ? 0.3
                          : 1,
                      padding: '2px 6px',
                    }}
                  >
                    ⬇
                  </button>

                  {isEditing ? (
                    <>
                      <button
                        onClick={saveEdit}
                        disabled={saving}
                        style={{
                          padding:
                            '3px 10px',
                          borderRadius:
                            '6px',
                          border: 'none',
                          background:
                            C.primary,
                          color: '#fff',
                          fontSize: '12px',
                          cursor: saving
                            ? 'default'
                            : 'pointer',
                          opacity: saving
                            ? 0.7
                            : 1,
                        }}
                      >
                        {saving
                          ? '...'
                          : '保存'}
                      </button>

                      <button
                        onClick={() =>
                          setEditingPage(
                            null,
                          )
                        }
                        disabled={saving}
                        style={{
                          padding:
                            '3px 10px',
                          borderRadius:
                            '6px',
                          border: `1px solid ${C.border}`,
                          background:
                            'transparent',
                          color:
                            C.textSecondary,
                          fontSize: '12px',
                          cursor: saving
                            ? 'default'
                            : 'pointer',
                          opacity: saving
                            ? 0.6
                            : 1,
                        }}
                      >
                        取消
                      </button>
                    </>
                  ) : (
                    <button
                      onClick={() =>
                        startEdit(page)
                      }
                      disabled={reordering}
                      style={{
                        padding: '3px 10px',
                        borderRadius: '6px',
                        border: `1px solid ${C.border}`,
                        background:
                          'transparent',
                        color:
                          C.textSecondary,
                        fontSize: '12px',
                        cursor: reordering
                          ? 'default'
                          : 'pointer',
                        opacity: reordering
                          ? 0.6
                          : 1,
                      }}
                    >
                      编辑
                    </button>
                  )}

                  <button
                    onClick={() =>
                      handleDelete(
                        page.page_number,
                      )
                    }
                    disabled={
                      reordering ||
                      saving
                    }
                    style={{
                      padding: '3px 10px',
                      borderRadius: '6px',
                      border: `1px solid ${C.border}`,
                      background:
                        'transparent',
                      color: C.danger,
                      fontSize: '12px',
                      cursor:
                        reordering || saving
                          ? 'default'
                          : 'pointer',
                      opacity:
                        reordering || saving
                          ? 0.6
                          : 1,
                    }}
                  >
                    删除
                  </button>
                </div>
              </div>

              {/* 卡片内容 */}
              {isEditing ? (
                <div
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '10px',
                    fontSize: '13px',
                  }}
                >
                  {/* 教学目的 */}
                  <label
                    style={{
                      color:
                        C.textSecondary,
                    }}
                  >
                    教学目的
                    <textarea
                      value={String(
                        editForm.purpose ||
                          '',
                      )}
                      onChange={event =>
                        setEditForm({
                          ...editForm,
                          purpose:
                            event.target
                              .value,
                        })
                      }
                      rows={2}
                      style={{
                        width: '100%',
                        border: `1px solid ${C.border}`,
                        borderRadius: '6px',
                        padding: '6px 8px',
                        resize: 'vertical',
                        marginTop: '4px',
                      }}
                    />
                  </label>

                  {/* 内容概要 */}
                  <label
                    style={{
                      color:
                        C.textSecondary,
                    }}
                  >
                    内容概要
                    <span
                      style={{
                        color:
                          C.textMuted,
                        fontSize: '12px',
                        marginLeft:
                          '6px',
                      }}
                    >
                      （写得越详细，AI
                      越会逐点展开这一页）
                    </span>
                    <textarea
                      value={String(
                        editForm.content_summary ||
                          '',
                      )}
                      onChange={event =>
                        setEditForm({
                          ...editForm,
                          content_summary:
                            event.target
                              .value,
                        })
                      }
                      rows={4}
                      style={{
                        width: '100%',
                        border: `1px solid ${C.border}`,
                        borderRadius: '6px',
                        padding: '6px 8px',
                        resize: 'vertical',
                        marginTop: '4px',
                      }}
                    />
                  </label>

                  {/* 内容丰富度 */}
                  <div>
                    <div
                      style={{
                        color:
                          C.textSecondary,
                        marginBottom:
                          '6px',
                      }}
                    >
                      内容丰富度
                      <span
                        style={{
                          color:
                            C.textMuted,
                          fontSize: '12px',
                          marginLeft:
                            '6px',
                        }}
                      >
                        （想让这一页内容更多、举例更丰富，就选「充实」）
                      </span>
                    </div>

                    <div
                      style={{
                        display: 'flex',
                        gap: '8px',
                      }}
                    >
                      {RICHNESS_OPTIONS.map(
                        option => {
                          const selected =
                            richnessOf(
                              Number(
                                editForm.estimated_complexity,
                              ) || 3,
                            ).value ===
                            option.value

                          return (
                            <button
                              key={
                                option.value
                              }
                              type="button"
                              onClick={() =>
                                setEditForm({
                                  ...editForm,
                                  estimated_complexity:
                                    option.value,
                                })
                              }
                              style={{
                                flex: 1,
                                padding:
                                  '10px 8px',
                                borderRadius:
                                  '10px',
                                cursor:
                                  'pointer',
                                border: `2px solid ${
                                  selected
                                    ? option.color
                                    : C.border
                                }`,
                                background:
                                  selected
                                    ? option.bg
                                    : C.white,
                                display:
                                  'flex',
                                flexDirection:
                                  'column',
                                alignItems:
                                  'center',
                                gap: '2px',
                                transition:
                                  'all 0.15s',
                              }}
                            >
                              <span
                                style={{
                                  fontSize:
                                    '20px',
                                }}
                              >
                                {
                                  option.emoji
                                }
                              </span>
                              <span
                                style={{
                                  fontSize:
                                    '13px',
                                  fontWeight: 700,
                                  color: selected
                                    ? option.color
                                    : C.textPrimary,
                                }}
                              >
                                {
                                  option.label
                                }
                              </span>
                              <span
                                style={{
                                  fontSize:
                                    '11px',
                                  color: selected
                                    ? option.color
                                    : C.textMuted,
                                  textAlign:
                                    'center',
                                  lineHeight:
                                    '1.3',
                                }}
                              >
                                {option.desc}
                              </span>
                            </button>
                          )
                        },
                      )}
                    </div>
                  </div>

                  {/* 高级选项 */}
                  <div
                    style={{
                      marginTop: '2px',
                    }}
                  >
                    <button
                      type="button"
                      onClick={() =>
                        setShowAdvanced(
                          value => !value,
                        )
                      }
                      style={{
                        background:
                          'transparent',
                        border: 'none',
                        color: C.textMuted,
                        fontSize: '12px',
                        cursor: 'pointer',
                        padding: '4px 0',
                      }}
                    >
                      {showAdvanced
                        ? '▼'
                        : '▶'}{' '}
                      ⚙
                      高级选项（可不改，不确定就保持默认）
                    </button>

                    {showAdvanced && (
                      <div
                        style={{
                          display: 'flex',
                          flexDirection:
                            'column',
                          gap: '10px',
                          marginTop: '6px',
                          padding: '12px',
                          borderRadius:
                            '8px',
                          background:
                            '#F9FAFB',
                          border: `1px solid ${C.border}`,
                        }}
                      >
                        <div
                          style={{
                            display: 'flex',
                            gap: '12px',
                          }}
                        >
                          <label
                            style={{
                              flex: 1,
                              color:
                                C.textSecondary,
                            }}
                          >
                            交互类型
                            <select
                              value={String(
                                editForm.interaction_type ||
                                  'static',
                              )}
                              onChange={event =>
                                setEditForm({
                                  ...editForm,
                                  interaction_type:
                                    event
                                      .target
                                      .value,
                                })
                              }
                              style={{
                                width:
                                  '100%',
                                border: `1px solid ${C.border}`,
                                borderRadius:
                                  '6px',
                                padding:
                                  '6px 8px',
                                marginTop:
                                  '4px',
                              }}
                            >
                              {Object.entries(
                                CW_INTERACTION_TYPES,
                              ).map(
                                ([
                                  key,
                                  value,
                                ]) => (
                                  <option
                                    key={
                                      key
                                    }
                                    value={
                                      key
                                    }
                                  >
                                    {
                                      value.emoji
                                    }{' '}
                                    {
                                      value.label
                                    }
                                  </option>
                                ),
                              )}
                            </select>
                          </label>

                          <label
                            style={{
                              flex: 1,
                              color:
                                C.textSecondary,
                            }}
                          >
                            视觉形式
                            <select
                              value={String(
                                editForm.visual_format ||
                                  'text_heavy',
                              )}
                              onChange={event =>
                                setEditForm({
                                  ...editForm,
                                  visual_format:
                                    event
                                      .target
                                      .value,
                                })
                              }
                              style={{
                                width:
                                  '100%',
                                border: `1px solid ${C.border}`,
                                borderRadius:
                                  '6px',
                                padding:
                                  '6px 8px',
                                marginTop:
                                  '4px',
                              }}
                            >
                              {Object.entries(
                                CW_VISUAL_FORMATS,
                              ).map(
                                ([
                                  key,
                                  value,
                                ]) => (
                                  <option
                                    key={
                                      key
                                    }
                                    value={
                                      key
                                    }
                                  >
                                    {
                                      value.emoji
                                    }{' '}
                                    {
                                      value.label
                                    }
                                  </option>
                                ),
                              )}
                            </select>
                          </label>
                        </div>

                        <label
                          style={{
                            color:
                              C.textSecondary,
                          }}
                        >
                          多媒体需求
                          <input
                            value={String(
                              editForm.media_requirements ||
                                '',
                            )}
                            onChange={event =>
                              setEditForm({
                                ...editForm,
                                media_requirements:
                                  event
                                    .target
                                    .value,
                              })
                            }
                            style={{
                              width: '100%',
                              border: `1px solid ${C.border}`,
                              borderRadius:
                                '6px',
                              padding:
                                '6px 8px',
                              marginTop:
                                '4px',
                            }}
                          />
                        </label>
                      </div>
                    )}
                  </div>
                </div>
              ) : (
                <div>
                  {/* 方案展示 */}
                  {page.purpose && (
                    <div
                      style={{
                        fontSize: '13px',
                        color:
                          C.textSecondary,
                        marginBottom:
                          '6px',
                      }}
                    >
                      <strong>目的：</strong>
                      {page.purpose}
                    </div>
                  )}

                  {page.content_summary && (
                    <div
                      style={{
                        fontSize: '13px',
                        color:
                          C.textSecondary,
                        marginBottom:
                          '8px',
                      }}
                    >
                      <strong>概要：</strong>
                      {page.content_summary}
                    </div>
                  )}

                  {/* 内容丰富度 */}
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '8px',
                      marginBottom: '6px',
                    }}
                  >
                    <span
                      style={{
                        padding:
                          '3px 12px',
                        borderRadius:
                          '10px',
                        fontSize: '12px',
                        fontWeight: 700,
                        background:
                          rich.bg,
                        color: rich.color,
                      }}
                    >
                      {rich.emoji}{' '}
                      内容{rich.label}
                    </span>
                  </div>

                  {/* 技术维度 */}
                  <div
                    style={{
                      fontSize: '12px',
                      color: C.textMuted,
                      display: 'flex',
                      gap: '12px',
                      flexWrap: 'wrap',
                    }}
                  >
                    <span>
                      {interaction.emoji}{' '}
                      {interaction.label}
                    </span>

                    <span>
                      {visual.emoji}{' '}
                      {visual.label}
                    </span>

                    {cognitive && (
                      <span>
                        🧠 {cognitive.label}
                      </span>
                    )}

                    {page.media_requirements && (
                      <span>
                        🖼️{' '}
                        {page
                          .media_requirements
                          .length > 16
                          ? page.media_requirements.slice(
                              0,
                              16,
                            ) + '...'
                          : page.media_requirements}
                      </span>
                    )}
                  </div>

                  {/* admin可见AOCI技术索引 */}
                  {isAdmin &&
                    page.page_index && (
                      <div
                        style={{
                          marginTop: '8px',
                        }}
                      >
                        <button
                          onClick={() =>
                            toggleIndexExpand(
                              page.page_number,
                            )
                          }
                          style={{
                            background:
                              'transparent',
                            border: 'none',
                            fontSize:
                              '12px',
                            color:
                              C.textMuted,
                            cursor:
                              'pointer',
                            padding:
                              '2px 0',
                          }}
                        >
                          {isIndexExpanded
                            ? '▼'
                            : '▶'}{' '}
                          AOCI索引
                        </button>

                        {isIndexExpanded && (
                          <pre
                            style={{
                              marginTop:
                                '4px',
                              padding:
                                '8px 12px',
                              borderRadius:
                                '6px',
                              background:
                                '#F9FAFB',
                              border: `1px solid ${C.border}`,
                              fontSize:
                                '11px',
                              color:
                                C.textSecondary,
                              whiteSpace:
                                'pre-wrap',
                              fontFamily:
                                'monospace',
                              lineHeight:
                                '1.5',
                              maxHeight:
                                '200px',
                              overflow:
                                'auto',
                            }}
                          >
                            {page.page_index}
                          </pre>
                        )}
                      </div>
                    )}
                </div>
              )}
            </div>
          )
        })}
      </div>

      {pages.length === 0 && (
        <div
          style={{
            textAlign: 'center',
            padding: '60px 0',
          }}
        >
          <div
            style={{
              fontSize: '40px',
              marginBottom: '12px',
            }}
          >
            📋
          </div>
          <div
            style={{
              fontSize: '15px',
              color: C.textSecondary,
            }}
          >
            还没有课件方案
          </div>
          <div
            style={{
              fontSize: '13px',
              color: C.textMuted,
              marginTop: '4px',
            }}
          >
            点击“AI生成方案”，AI将自动分析教案内容
          </div>
        </div>
      )}
    </div>
  )
}
