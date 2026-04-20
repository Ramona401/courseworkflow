/**
 * AssistantSelector.tsx — AI 助手选择器
 *
 * 功能概览:
 *   下拉选择器,按来源(system/group/personal)分组展示当前场景下可见的 AI 助手。
 *   支持:选中切换、查看详情、复制到我的(Fork)、编辑个人助手、新建个人助手。
 *
 * 使用场景:
 *   - 评审工作台顶部(scene=review_workbench)
 *   - 工坊各阶段顶部(scene=workshop_analyze/design/write/review/revise)
 *
 * 设计要点:
 *   - 首次加载自动选中 is_default_here=true 的第一项(默认推荐)
 *   - value=null 表示不选助手,走系统兜底默认行为
 *   - 点击复制到我的直接调用 forkAssistant,成功后切换到新助手
 *   - 编辑/新建通过 onEdit/onCreateNew 回调交给父组件处理
 *   - disabled 状态用于对话进行中时锁定选择
 *
 * v112(P0 STEP 10)关键修复:
 *   【问题】下拉面板在评审工作台和工坊阶段头中被父容器 overflow:hidden 裁切,z-index 无法解决
 *   【方案】下拉面板改用 position:fixed + 动态坐标定位,彻底脱离父容器裁切影响
 *   【实现】
 *     1. 新增 triggerRef 挂在触发按钮上,open 时 getBoundingClientRect 计算屏幕坐标
 *     2. 下拉面板从 absolute 改为 fixed,定位到按钮下方 6px、按钮右对齐
 *     3. 窗口滚动/缩放时自动关闭下拉(避免 fixed 定位漂移错位)
 *     4. 视口高度检查:若按钮下方空间不够 520px,面板向上弹出
 *
 * Props 契约:
 *   scene        - 场景代码,按此过滤列表(必传)
 *   value        - 当前选中的助手 ID(null=未选)
 *   onChange     - 选中变化回调,传入 ID 或 null
 *   subject?     - 可选:学科精准匹配
 *   grade?       - 可选:年级精准匹配
 *   onView?      - 可选:点击查看详情回调
 *   onEdit?      - 可选:点击编辑回调(个人助手)
 *   onCreateNew? - 可选:点击新建个人助手回调
 *   disabled?    - 可选:禁用态(对话进行中)
 *   compact?     - 可选:紧凑模式(按钮更小)
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import {
  listAssistants,
  forkAssistant,
  ASSISTANT_SCENE_LABELS,
  ASSISTANT_SOURCE_LABELS,
  ASSISTANT_SOURCE_EMOJI,
  type AIAssistantListItem,
  type AssistantScene,
  type AssistantSource,
} from '@/api/ai-assistants'

/* ==================== 样式常量(与 workshopConstants 保持一致) ==================== */
const C = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent:       '#F59E0B',
  success:      '#10B981',
  danger:       '#EF4444',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
  // 三种来源的强调色
  systemAccent: '#4F7BE8',   // 蓝
  groupAccent:  '#F59E0B',   // 橙
  personalAccent: '#10B981', // 绿
}

/** 下拉面板固定尺寸(用于 fixed 定位时的视口空间检查) */
const PANEL_WIDTH = 360
const PANEL_MAX_HEIGHT = 520

/* ==================== Props 类型 ==================== */

export interface AssistantSelectorProps {
  /** 当前场景代码,按此过滤助手列表 */
  scene: AssistantScene
  /** 当前选中的助手 ID(null 表示未选) */
  value: string | null
  /** 选中变化回调 */
  onChange: (id: string | null) => void
  /** 可选:学科精准匹配过滤 */
  subject?: string
  /** 可选:年级精准匹配过滤 */
  grade?: string
  /** 可选:点击查看详情回调(不传则不显示该按钮) */
  onView?: (id: string) => void
  /** 可选:点击编辑回调(不传则不显示该按钮,仅对 personal 助手显示) */
  onEdit?: (id: string) => void
  /** v121 新增:可选:点击删除回调(不传则不显示该按钮,仅对 personal 助手显示) */
  onDelete?: (id: string) => void
  /** 可选:点击新建个人助手回调(不传则不显示底部入口) */
  onCreateNew?: () => void
  /** 可选:禁用态(对话进行中等场景使用) */
  disabled?: boolean
  /** 可选:紧凑模式(触发按钮更小,用于顶部导航条) */
  compact?: boolean
}

/** 面板定位坐标(fixed 模式下存屏幕绝对坐标) */
interface PanelPosition {
  /** 面板顶部 y 坐标(正数=下方弹出)或 bottom y 坐标(向上弹出时) */
  top?: number
  bottom?: number
  /** 面板右边缘距视口右侧的距离 */
  right: number
}

/* ==================== 子组件:单条助手项 ==================== */

interface AssistantItemProps {
  item: AIAssistantListItem
  isSelected: boolean
  onSelect: () => void
  onView?: () => void
  onEdit?: () => void
  onFork: () => void
  onDelete?: () => void
  forking: boolean
}

function AssistantItem({
  item, isSelected, onSelect, onView, onEdit, onFork, onDelete, forking,
}: AssistantItemProps) {
  // 根据来源选择强调色
  const accentColor =
    item.source === 'system'   ? C.systemAccent   :
    item.source === 'group'    ? C.groupAccent    :
                                 C.personalAccent

  return (
    <div
      onClick={onSelect}
      style={{
        padding: '10px 12px',
        borderRadius: '8px',
        cursor: 'pointer',
        border: `1.5px solid ${isSelected ? C.primary : C.border}`,
        background: isSelected ? C.primaryLight : '#fff',
        borderLeft: `3px solid ${accentColor}`,
        marginBottom: '4px',
        transition: 'all 150ms ease',
        display: 'flex',
        alignItems: 'flex-start',
        gap: '8px',
      }}
      onMouseEnter={e => {
        if (!isSelected) (e.currentTarget as HTMLDivElement).style.background = '#F9FAFB'
      }}
      onMouseLeave={e => {
        if (!isSelected) (e.currentTarget as HTMLDivElement).style.background = '#fff'
      }}
    >
      {/* 单选圆点 */}
      <div style={{
        width: '14px', height: '14px', borderRadius: '50%',
        border: `2px solid ${isSelected ? C.primary : C.border}`,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        flexShrink: 0, marginTop: '2px',
      }}>
        {isSelected && <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: C.primary }} />}
      </div>

      {/* 主内容区 */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {/* 标题行 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '6px', flexWrap: 'wrap' }}>
          <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>
            {item.avatar_emoji} {item.name}
          </span>
          {item.is_default_here && (
            <span style={{
              padding: '1px 6px', borderRadius: '4px',
              background: 'rgba(79,123,232,0.12)', color: C.primary,
              fontSize: '10px', fontWeight: 600,
            }}>推荐</span>
          )}
        </div>

        {/* 描述行 */}
        {item.description && (
          <div style={{
            fontSize: '11px', color: C.textSec, marginTop: '2px',
            lineHeight: 1.5,
            overflow: 'hidden',
            display: '-webkit-box',
            WebkitLineClamp: 2,
            WebkitBoxOrient: 'vertical' as const,
          }}>
            {item.description}
          </div>
        )}

        {/* 元信息行:学科 / 年级 / 使用次数 */}
        {(item.subject || item.grade_range || item.use_count > 0) && (
          <div style={{ fontSize: '10px', color: C.textMuted, marginTop: '3px', display: 'flex', gap: '8px' }}>
            {item.subject && <span>📚 {item.subject}</span>}
            {item.grade_range && <span>🎓 {item.grade_range}</span>}
            {item.use_count > 0 && <span>用{item.use_count}次</span>}
          </div>
        )}

        {/* 操作按钮行 */}
        <div
          style={{ display: 'flex', gap: '4px', marginTop: '6px' }}
          onClick={e => e.stopPropagation()}
        >
          {onView && (
            <button
              onClick={onView}
              title="查看详情"
              style={miniBtnStyle(false)}
            >📖 看</button>
          )}
          {/* personal 助手:编辑 + 删除 */}
          {item.source === 'personal' && onEdit && item.can_edit && (
            <button
              onClick={onEdit}
              title="编辑"
              style={miniBtnStyle(false)}
            >✏️ 改</button>
          )}
          {item.source === 'personal' && onDelete && item.can_delete && (
            <button
              onClick={onDelete}
              title="删除"
              style={miniBtnStyle(false, C.danger)}
            >🗑 删</button>
          )}
          {/* system/group 助手:复制到我的 */}
          {item.source !== 'personal' && (
            <button
              onClick={onFork}
              disabled={forking}
              title="复制到我的助手(以便修改)"
              style={miniBtnStyle(false, C.primary)}
            >{forking ? '复制中…' : '➕ 复制到我的'}</button>
          )}
        </div>
      </div>
    </div>
  )
}

/** 小按钮样式(操作按钮) */
function miniBtnStyle(filled: boolean, color: string = C.textSec): React.CSSProperties {
  return {
    padding: '3px 8px',
    borderRadius: '5px',
    border: `1px solid ${filled ? color : C.border}`,
    background: filled ? color : '#fff',
    color: filled ? '#fff' : color,
    fontSize: '11px',
    fontWeight: 500,
    cursor: 'pointer',
    whiteSpace: 'nowrap',
  }
}

/* ==================== 主组件 ==================== */

export default function AssistantSelector(props: AssistantSelectorProps) {
  const { scene, value, onChange, subject, grade, onView, onEdit, onDelete, onCreateNew, disabled, compact } = props

  const [assistants, setAssistants] = useState<AIAssistantListItem[]>([])
  const [loading, setLoading]       = useState(false)
  const [error, setError]           = useState<string | null>(null)
  const [open, setOpen]             = useState(false)
  const [forkingId, setForkingId]   = useState<string | null>(null)

  // v112 新增:下拉面板的 fixed 定位坐标,open 时通过触发按钮 rect 计算
  const [panelPos, setPanelPos] = useState<PanelPosition>({ top: 0, right: 0 })

  // v112 新增:触发按钮 ref,用于 getBoundingClientRect 获取屏幕坐标
  const triggerRef   = useRef<HTMLButtonElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  // v121 Bug 1 修复:下拉面板 ref,用于 handleScroll 识别面板内滚动
  const panelRefV121 = useRef<HTMLDivElement>(null)

  // ==================== 加载助手列表 ====================
  const loadAssistants = useCallback(async () => {
    setLoading(true); setError(null)
    try {
      const list = await listAssistants({ scene, subject, grade })
      // 注意:listAssistants 返回 {assistants:[...], total:N} 结构,需解构出 assistants 数组
      setAssistants(list.assistants || [])
    } catch (e: unknown) {
      console.error('加载 AI 助手列表失败:', e)
      setError(e instanceof Error ? e.message : '加载失败')
      setAssistants([])
    } finally {
      setLoading(false)
    }
  }, [scene, subject, grade])

  useEffect(() => { loadAssistants() }, [loadAssistants])

  // ==================== 首次加载自动选中默认助手 ====================
  // 规则:若当前 value=null 且列表里存在 is_default_here=true 的项,自动选中第一个
  useEffect(() => {
    if (value !== null) return
    if (assistants.length === 0) return
    const defaultOne = assistants.find(a => a.is_default_here)
    if (defaultOne) onChange(defaultOne.id)
  // 仅依赖助手列表变化,不依赖 value 避免循环
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assistants])

  /**
   * v112 新增:计算下拉面板的 fixed 定位坐标
   *
   * 计算逻辑:
   *   1. 通过触发按钮的 rect 拿到屏幕绝对坐标
   *   2. 默认按钮下方 6px 处展开面板
   *   3. 视口高度检查:如果按钮下方剩余空间 < PANEL_MAX_HEIGHT,向上展开
   *   4. 水平方向始终右对齐(跟随按钮右边缘)
   */
  const recalcPanelPos = useCallback(() => {
    if (!triggerRef.current) return
    const rect = triggerRef.current.getBoundingClientRect()
    const viewportHeight = window.innerHeight
    const spaceBelow = viewportHeight - rect.bottom
    const spaceAbove = rect.top

    // 判断向上还是向下弹出
    const openUpward = spaceBelow < PANEL_MAX_HEIGHT && spaceAbove > spaceBelow

    if (openUpward) {
      // 向上弹出:面板底部紧贴按钮顶部
      setPanelPos({
        bottom: viewportHeight - rect.top + 6,
        right: window.innerWidth - rect.right,
      })
    } else {
      // 向下弹出:面板顶部在按钮下方 6px
      setPanelPos({
        top: rect.bottom + 6,
        right: window.innerWidth - rect.right,
      })
    }
  }, [])

  // ==================== 点击外部关闭 + 滚动关闭 + 尺寸变化重算 ====================
  useEffect(() => {
    if (!open) return

    // 打开时先计算一次位置
    recalcPanelPos()

    const handleClick = (e: MouseEvent) => {
      const target = e.target as Node
      // 两个容器都不能包含点击点才关闭:触发按钮本身 + 下拉面板
      if (containerRef.current && !containerRef.current.contains(target)) {
        setOpen(false)
      }
    }
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    // v112 新增:窗口滚动 / resize 时关闭下拉,避免 fixed 定位漂移脱节
    // v121 Bug 1 修复:区分面板内滚动(保持打开)和面板外滚动(关闭)
    const handleScroll = (e: Event) => {
      const target = e.target as Node | null
      const panelEl = panelRefV121.current
      if (panelEl && target && panelEl.contains(target)) return
      setOpen(false)
    }
    const handleResize = () => recalcPanelPos()

    document.addEventListener('mousedown', handleClick)
    document.addEventListener('keydown', handleEsc)
    // capture 阶段监听所有父级滚动(包括评审工作台中列的 overflow:auto 容器)
    window.addEventListener('scroll', handleScroll, true)
    window.addEventListener('resize', handleResize)

    return () => {
      document.removeEventListener('mousedown', handleClick)
      document.removeEventListener('keydown', handleEsc)
      window.removeEventListener('scroll', handleScroll, true)
      window.removeEventListener('resize', handleResize)
    }
  }, [open, recalcPanelPos])

  // ==================== 当前选中项 ====================
  const selected = value ? assistants.find(a => a.id === value) || null : null

  // ==================== 操作处理 ====================

  const handleSelect = (id: string) => {
    onChange(id)
    setOpen(false)
  }

  const handleFork = async (item: AIAssistantListItem) => {
    if (forkingId) return
    const confirmMsg = `将 ${item.name} 复制一份到 我的助手,复制后可自由修改。确认继续?`
    if (!window.confirm(confirmMsg)) return

    setForkingId(item.id)
    try {
      const forked = await forkAssistant(item.id)
      // 刷新列表并切换到新助手
      await loadAssistants()
      onChange(forked.id)
      // 提示用户
      alert(`已复制为 ${forked.name},可在 我的助手 中编辑。`)
    } catch (e: unknown) {
      console.error('Fork 助手失败:', e)
      alert(e instanceof Error ? e.message : '复制失败,请重试')
    } finally {
      setForkingId(null)
    }
  }

  const handleView = (item: AIAssistantListItem) => {
    if (onView) onView(item.id)
  }

  const handleEdit = (item: AIAssistantListItem) => {
    if (onEdit) onEdit(item.id)
  }

  const handleCreateNew = () => {
    setOpen(false)
    if (onCreateNew) onCreateNew()
  }

  // ==================== 按来源分组 ====================
  const grouped: Record<AssistantSource, AIAssistantListItem[]> = {
    system:   [],
    group:    [],
    personal: [],
  }
  for (const a of assistants) {
    grouped[a.source].push(a)
  }

  // ==================== 渲染:触发按钮 ====================
  const triggerLabel = selected
    ? `${selected.avatar_emoji} ${selected.name}`
    : loading
      ? '加载中…'
      : error
        ? '加载失败,点击重试'
        : '选择 AI 助手…'

  const triggerPadding = compact ? '6px 10px' : '8px 12px'
  const triggerFontSize = compact ? '12px' : '13px'
  const triggerMaxWidth = compact ? '220px' : '280px'

  /**
   * v112:面板 fixed 定位样式
   * 根据 panelPos 的 top/bottom 字段决定向下或向上展开
   */
  const panelFixedStyle: React.CSSProperties = {
    position: 'fixed',
    ...(panelPos.top !== undefined ? { top: `${panelPos.top}px` } : {}),
    ...(panelPos.bottom !== undefined ? { bottom: `${panelPos.bottom}px` } : {}),
    right: `${panelPos.right}px`,
    width: `${PANEL_WIDTH}px`,
    maxHeight: `${PANEL_MAX_HEIGHT}px`,
    overflow: 'auto',
    background: '#fff',
    borderRadius: '10px',
    border: `1px solid ${C.border}`,
    boxShadow: '0 12px 32px rgba(0,0,0,0.16)',
    zIndex: 9999,
    padding: '10px',
  }

  return (
    <div ref={containerRef} style={{ position: 'relative', display: 'inline-block' }}>
      {/* ============ 触发按钮 ============ */}
      <button
        ref={triggerRef}
        onClick={() => {
          if (disabled) return
          if (error) { loadAssistants(); return }
          setOpen(o => !o)
        }}
        disabled={disabled}
        style={{
          display: 'flex', alignItems: 'center', gap: '6px',
          padding: triggerPadding,
          borderRadius: '8px',
          border: `1px solid ${open ? C.primary : C.border}`,
          background: disabled ? '#F3F4F6' : open ? C.primaryLight : '#fff',
          color: disabled ? C.textMuted : C.text,
          fontSize: triggerFontSize,
          fontWeight: 500,
          cursor: disabled ? 'not-allowed' : 'pointer',
          outline: 'none',
          maxWidth: triggerMaxWidth,
          transition: 'all 150ms ease',
        }}
      >
        <span style={{
          overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
          color: error ? C.danger : 'inherit',
        }}>
          🤖 {triggerLabel}
        </span>
        {selected?.is_default_here && (
          <span style={{
            padding: '1px 5px', borderRadius: '3px',
            background: 'rgba(79,123,232,0.12)', color: C.primary,
            fontSize: '10px', fontWeight: 600, flexShrink: 0,
          }}>推荐</span>
        )}
        <span style={{
          fontSize: '10px', color: C.textMuted, flexShrink: 0,
          transform: open ? 'rotate(180deg)' : 'rotate(0deg)',
          transition: 'transform 150ms ease',
        }}>▼</span>
      </button>

      {/* ============ 下拉面板(fixed 定位,脱离父容器 overflow 裁切)============ */}
      {open && (
        <div ref={panelRefV121} style={panelFixedStyle}>
          {/* ---------- 头部:场景标签 ---------- */}
          <div style={{
            padding: '6px 8px', marginBottom: '8px',
            fontSize: '11px', color: C.textMuted,
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            borderBottom: `1px solid ${C.border}`,
            paddingBottom: '8px',
          }}>
            <span>场景:{ASSISTANT_SCENE_LABELS[scene] || scene}</span>
            <button
              onClick={() => loadAssistants()}
              title="刷新列表"
              style={{
                background: 'none', border: 'none', cursor: 'pointer',
                fontSize: '11px', color: C.primary, padding: '2px 6px',
              }}
            >🔄 刷新</button>
          </div>

          {/* ---------- 加载 / 错误 / 空态 ---------- */}
          {loading && (
            <div style={{ padding: '24px 0', textAlign: 'center', color: C.textMuted, fontSize: '12px' }}>
              加载助手中…
            </div>
          )}
          {error && !loading && (
            <div style={{
              padding: '12px', borderRadius: '8px',
              background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.15)',
              fontSize: '12px', color: C.danger, textAlign: 'center',
            }}>
              ⚠️ {error}
              <br />
              <button
                onClick={() => loadAssistants()}
                style={{
                  marginTop: '6px', padding: '4px 12px', borderRadius: '5px',
                  border: `1px solid ${C.danger}`, background: '#fff',
                  color: C.danger, fontSize: '11px', cursor: 'pointer',
                }}
              >重试</button>
            </div>
          )}
          {!loading && !error && assistants.length === 0 && (
            <div style={{ padding: '24px 12px', textAlign: 'center', color: C.textMuted, fontSize: '12px', lineHeight: 1.6 }}>
              <div style={{ fontSize: '28px', marginBottom: '8px' }}>🤖</div>
              该场景暂无可用助手<br />
              {onCreateNew && (
                <button
                  onClick={handleCreateNew}
                  style={{
                    marginTop: '10px', padding: '6px 14px', borderRadius: '6px',
                    border: `1px solid ${C.primary}`, background: C.primaryLight,
                    color: C.primary, fontSize: '12px', fontWeight: 600, cursor: 'pointer',
                  }}
                >+ 新建个人助手</button>
              )}
            </div>
          )}

          {/* ---------- 按来源分组展示 ---------- */}
          {!loading && !error && assistants.length > 0 && (
            <>
              {(['system', 'group', 'personal'] as AssistantSource[]).map(source => {
                const items = grouped[source]
                if (items.length === 0) return null
                return (
                  <div key={source} style={{ marginBottom: '10px' }}>
                    {/* 分组标题 */}
                    <div style={{
                      padding: '4px 8px 6px',
                      fontSize: '11px', fontWeight: 700,
                      color: C.textSec,
                      display: 'flex', alignItems: 'center', gap: '4px',
                    }}>
                      <span>{ASSISTANT_SOURCE_EMOJI[source]} {ASSISTANT_SOURCE_LABELS[source]}</span>
                      <span style={{ color: C.textMuted, fontWeight: 400 }}>({items.length})</span>
                    </div>
                    {/* 分组内条目 */}
                    {items.map(item => (
                      <AssistantItem
                        key={item.id}
                        item={item}
                        isSelected={item.id === value}
                        onSelect={() => handleSelect(item.id)}
                        onView={onView ? () => handleView(item) : undefined}
                        onEdit={onEdit ? () => handleEdit(item) : undefined}
                        onDelete={onDelete ? () => onDelete(item.id) : undefined}
                        onFork={() => handleFork(item)}
                        forking={forkingId === item.id}
                      />
                    ))}
                  </div>
                )
              })}

              {/* ---------- 底部:清除选择 + 新建 ---------- */}
              <div style={{
                display: 'flex', gap: '6px',
                marginTop: '6px', paddingTop: '8px',
                borderTop: `1px solid ${C.border}`,
              }}>
                {value !== null && (
                  <button
                    onClick={() => { onChange(null); setOpen(false) }}
                    style={{
                      flex: 1, padding: '7px',
                      borderRadius: '6px',
                      border: `1px solid ${C.border}`, background: '#fff',
                      color: C.textSec, fontSize: '12px', cursor: 'pointer',
                    }}
                  >✕ 清除选择</button>
                )}
                {onCreateNew && (
                  <button
                    onClick={handleCreateNew}
                    style={{
                      flex: 1, padding: '7px',
                      borderRadius: '6px',
                      border: `1px dashed ${C.primary}`, background: 'transparent',
                      color: C.primary, fontSize: '12px', fontWeight: 600, cursor: 'pointer',
                    }}
                  >+ 新建个人助手</button>
                )}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  )
}
