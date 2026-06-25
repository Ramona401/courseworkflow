/**
 * SaveAssistantModal.tsx — 「存为我的助手」确认弹窗(提示词工坊 阶段A)
 *
 * 角色:
 *   老师/教研员/管理员在 MyAssistantsPage 里和 AI 聊出一版 full_prompt 草稿后,
 *   点"存为我的助手",弹出本弹窗做最后确认——填名称 + 选「存到谁的货架」+
 *   勾选适用场景 + 确认学科,然后落库。
 *
 * ════════════ 身份分层(里程碑一:教研组级分享打通) ════════════
 *   同一份草稿,不同身份能把它"摆到不同的货架上"。货架不再按 userRole 静态判断,
 *   而是打开弹窗时调 getMyPublishGroups() 拿真实可发布范围,动态展示:
 *     - 👤 只给我自己用(personal)        : 所有人恒有
 *     - 👥 发布到教研组(group + group_id) : 我是某组 lead/backbone 时显示,需选具体组
 *     - 🏫 推荐给全校老师(group,无 group_id): 学校管理员(senior_operator/admin)显示
 *     - 🏛️ 全平台通用(system)             : 仅 admin 显示
 *   关键:教研组级助手只对该组成员可见;全校级对全校可见。两档都走 source='group',
 *   靠 group_id 是否携带区分(后端 service 据此落库 + 判可见)。
 *   组员想改组助手 → fork 成自己的 personal 再改,原版不动(本弹窗不涉及编辑)。
 *
 * 为什么单独成文件:
 *   MyAssistantsPage 主体已含「对话组合器 + 相关助手侧栏」两大块,逼近 600 行红线。
 *   把这个确认弹窗抽出来,主页面与本弹窗都稳在红线内,且弹窗逻辑内聚便于单独维护。
 *
 * 设计要点(对齐 Yuhan 拍板):
 *   - 不让老师面对"一堆选项框"。普通老师只问三件事:名字、场景、学科(货架对其隐藏)。
 *   - 货架文案说人话:不写 personal/group/system,写"只给我自己用 / 发布到教研组 / 推荐给全校老师 / 全平台通用"。
 *   - 学科默认 = 老师在页面顶部选的"我主要教"学科(defaultSubject),可改但通常不用动。
 *   - emoji/description 不在这里问——给默认值(emoji 🤖, description 空),想精修去助手编辑弹窗。
 *   - 草稿(full_prompt)只读(只展示字符数),要改正文回对话区继续聊。
 *
 * Props 契约:
 *   open           - 是否显示
 *   draft          - 已聊出的完整 full_prompt 草稿(只读,落库用)
 *   userRole       - 当前用户角色(保留以兼容父组件,货架展示已改为接口驱动,不再依赖它)
 *   defaultSubject - 默认学科(取自页面顶部"我主要教",可空)
 *   defaultScene   - 默认勾选的场景(取自 designer 当前场景,可空;空则默认勾全部工坊场景)
 *   onClose        - 关闭回调
 *   onSaved        - 保存成功回调,传回新助手 id 和最终 source(父组件据此刷新/提示)
 */

import { useState, useEffect } from 'react'
import {
  createAssistant,
  getMyPublishGroups,
  ASSISTANT_SCENE_LABELS,
  type AssistantScene,
  type AssistantSource,
  type CreateAIAssistantRequest,
  type PublishGroup,
} from '@/api/ai-assistants'

/* ==================== 样式常量(与 EditModal/Selector/DesignerPanel 保持一致) ==================== */
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
  borderMid:    '#E5E7EB',
}

/** 学科可选项(与 AssistantEditModal 的 SUBJECTS 保持一致,避免循环依赖写死) */
const SUBJECTS = [
  '', // 空=不限
  '人工智能', '语文', '数学', '英语', '物理', '化学', '生物',
  '历史', '地理', '政治', '科学', '信息科技', '技术', '综合实践',
]

/** 工坊全部场景(defaultScene 为空时的兜底默认勾选——覆盖备课全链路 5 阶段) */
const WORKSHOP_SCENES: AssistantScene[] = [
  'workshop_analyze', 'workshop_design', 'workshop_write', 'workshop_review', 'workshop_revise',
]

/** prompt 长度上限(与后端 maxAssistantPromptLen 对齐) */
const MAX_PROMPT_LEN = 128 * 1024

/* ==================== 货架(shelf)定义 ==================== */
//
// 里程碑一:货架内部 key 扩为 4 档(group 来源细分教研组级/全校级两档),
// 提交时再映射回后端的 source + group_id。

/** 货架内部 key(不直接等于后端 source,group 拆两档) */
type ShelfKey = 'personal' | 'group_teaching' | 'group_school' | 'system'

/** 单个货架选项的展示信息 */
interface ShelfOption {
  key: ShelfKey
  emoji: string
  label: string      // 人话标题
  hint: string       // 一句话说明给谁用
}

/** 全部货架选项的展示文案(按 key 取用) */
const SHELF_OPTIONS: Record<ShelfKey, ShelfOption> = {
  personal:       { key: 'personal',       emoji: '👤', label: '只给我自己用',   hint: '存进「我的助手」,只有你能看到和使用' },
  group_teaching: { key: 'group_teaching', emoji: '👥', label: '发布到教研组',   hint: '只推荐给所选教研组的老师,组内备课时可选用' },
  group_school:   { key: 'group_school',   emoji: '🏫', label: '推荐给全校老师', hint: '作为本校推荐助手,全校老师备课时都能选用' },
  system:         { key: 'system',         emoji: '🏛️', label: '全平台通用',     hint: '作为系统助手,所有学校所有老师都能用' },
}

/**
 * 把货架 key 映射为后端创建请求的 source + group_id
 *   personal       → source=personal
 *   group_teaching → source=group + group_id=选中教研组
 *   group_school   → source=group(不带 group_id,全校级)
 *   system         → source=system
 */
function shelfToSourceAndGroup(
  shelf: ShelfKey,
  selectedGroupID: string,
): { source: AssistantSource; groupID?: string } {
  switch (shelf) {
    case 'personal':       return { source: 'personal' }
    case 'group_teaching': return { source: 'group', groupID: selectedGroupID }
    case 'group_school':   return { source: 'group' }
    case 'system':         return { source: 'system' }
  }
}

/* ==================== Props 类型 ==================== */

export interface SaveAssistantModalProps {
  open: boolean
  /** 已聊出的完整 full_prompt 草稿(只读) */
  draft: string
  /** 当前用户角色(保留以兼容父组件;货架展示已改为接口驱动,不再依赖此值) */
  userRole?: string
  /** 默认学科(取自页面顶部"我主要教") */
  defaultSubject?: string
  /** 默认勾选的场景(取自 designer 当前场景) */
  defaultScene?: AssistantScene
  /** 关闭回调 */
  onClose: () => void
  /** 保存成功回调,传回新助手 ID 和最终 source */
  onSaved: (id: string, source: AssistantSource) => void
}

/* ==================== 主组件 ==================== */

export default function SaveAssistantModal(props: SaveAssistantModalProps) {
  const { open, draft, defaultSubject, defaultScene, onClose, onSaved } = props

  // ==================== 表单状态 ====================
  const [name, setName]       = useState('')
  const [subject, setSubject] = useState('')
  const [scenes, setScenes]   = useState<AssistantScene[]>([])
  const [shelf, setShelf]     = useState<ShelfKey>('personal') // 货架:默认存为我的
  const [saving, setSaving]   = useState(false)
  const [err, setErr]         = useState<string | null>(null)

  // ==================== 可发布范围(来自 /my-groups 接口) ====================
  const [shelfKeys, setShelfKeys]           = useState<ShelfKey[]>(['personal']) // 当前用户可选货架
  const [publishGroups, setPublishGroups]   = useState<PublishGroup[]>([])       // 可发布的教研组
  const [selectedGroupID, setSelectedGroupID] = useState('')                     // 教研组级时选中的组
  const [loadingScope, setLoadingScope]     = useState(false)                    // 拉取可发布范围中

  // 是否需要显示货架选择(只有可选 >1 个时才显示;普通老师只有 personal 不显示)
  const showShelfPicker = shelfKeys.length > 1

  // ==================== open 时初始化表单 + 拉取可发布范围 ====================
  useEffect(() => {
    if (!open) return
    // 基础表单初始化
    setName('')
    setSubject(defaultSubject || '')
    setScenes(defaultScene ? [defaultScene] : [...WORKSHOP_SCENES])
    setShelf('personal')          // 货架默认 personal(无论什么身份,默认"先存给自己"最安全)
    setSelectedGroupID('')
    setErr(null)
    setSaving(false)

    // 拉取可发布范围:据返回标志位 + groups 动态拼货架
    let cancelled = false
    setLoadingScope(true)
    getMyPublishGroups()
      .then((res) => {
        if (cancelled) return
        const keys: ShelfKey[] = ['personal'] // personal 恒在最前
        if (res.can_publish_group && res.groups.length > 0) keys.push('group_teaching')
        if (res.can_publish_school) keys.push('group_school')
        if (res.can_publish_system) keys.push('system')
        setShelfKeys(keys)
        setPublishGroups(res.groups || [])
        // 教研组下拉默认选第一个组,供老师选了"发布到教研组"时直接可用
        if (res.groups && res.groups.length > 0) {
          setSelectedGroupID(res.groups[0].id)
        }
      })
      .catch(() => {
        // 拉取失败:保守只给 personal(不阻断保存,普通场景本就只有 personal)
        if (cancelled) return
        setShelfKeys(['personal'])
        setPublishGroups([])
      })
      .finally(() => {
        if (!cancelled) setLoadingScope(false)
      })

    return () => { cancelled = true }
  }, [open, defaultSubject, defaultScene])

  // ==================== ESC 关闭 ====================
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !saving) onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, saving, onClose])

  // ==================== 场景多选切换 ====================
  const toggleScene = (scene: AssistantScene) => {
    setScenes(prev => (prev.includes(scene) ? prev.filter(s => s !== scene) : [...prev, scene]))
  }

  // ==================== 提交保存 ====================
  const handleSave = async () => {
    if (saving) return
    // 校验(与后端校验口径一致,前置拦截给友好提示)
    if (!name.trim()) { setErr('请给助手起个名字,方便以后认出它'); return }
    if (!draft.trim()) { setErr('草稿还是空的,请先在左侧和 AI 聊出一版'); return }
    if (scenes.length === 0) { setErr('请至少勾选一个适用场景'); return }
    if (draft.length > MAX_PROMPT_LEN) {
      setErr(`提示词过长(${draft.length} 字符),上限 ${MAX_PROMPT_LEN} 字符`)
      return
    }

    // 货架兜底:若当前选中的货架不在允许范围内(理论不会),强制回退 personal
    const finalShelf: ShelfKey = shelfKeys.includes(shelf) ? shelf : 'personal'

    // 教研组级必须选中一个组
    if (finalShelf === 'group_teaching' && !selectedGroupID) {
      setErr('请选择要发布到哪个教研组')
      return
    }

    const { source, groupID } = shelfToSourceAndGroup(finalShelf, selectedGroupID)

    setSaving(true)
    setErr(null)
    try {
      const req: CreateAIAssistantRequest = {
        name: name.trim(),
        avatar_emoji: '🤖',
        description: '',
        source,                 // 后端会按角色+教研组身份再校验
        full_prompt: draft,
        subject: subject.trim(),
        grade_range: '',
        scenes,
        ...(groupID ? { group_id: groupID } : {}), // 仅教研组级携带 group_id
      }
      const created = await createAssistant(req)
      onSaved(created.id, (created.source as AssistantSource) || source)
    } catch (e) {
      setErr(e instanceof Error ? e.message : '保存失败,请重试')
      setSaving(false)
    }
  }

  // ==================== 未 open 不渲染 ====================
  if (!open) return null

  return (
    <div
      onClick={() => { if (!saving) onClose() }}
      style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        background: 'rgba(17,24,39,0.5)',
        zIndex: 10001, // 高于对话区可能的浮层
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: '24px',
      }}
    >
      {/* 弹窗本体(阻止冒泡) */}
      <div
        onClick={e => e.stopPropagation()}
        style={{
          width: '540px', maxWidth: '100%', maxHeight: '90vh',
          background: C.card, borderRadius: '14px',
          boxShadow: '0 24px 64px rgba(0,0,0,0.18)',
          display: 'flex', flexDirection: 'column', overflow: 'hidden',
        }}
      >
        {/* 标题栏 */}
        <div style={{
          padding: '16px 20px', borderBottom: `1px solid ${C.border}`,
          background: 'linear-gradient(135deg,rgba(16,185,129,0.06),rgba(16,185,129,0.02))',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0,
        }}>
          <span style={{ fontSize: '15px', fontWeight: 700, color: C.text }}>
            💾 保存这个助手
          </span>
          <button
            onClick={() => { if (!saving) onClose() }}
            style={{ background: 'none', border: 'none', cursor: saving ? 'not-allowed' : 'pointer', fontSize: '20px', color: C.textMuted, lineHeight: 1 }}
          >×</button>
        </div>

        {/* 表单区 */}
        <div style={{ flex: 1, overflow: 'auto', padding: '20px 24px' }}>
          {/* 草稿信息提示 */}
          <div style={{
            marginBottom: '16px', padding: '10px 12px', borderRadius: '8px',
            background: C.primaryLight, border: '1px solid rgba(79,123,232,0.15)',
            fontSize: '12px', color: C.textSec, lineHeight: 1.6,
          }}>
            ✨ 已聊出一版提示词草稿(<b style={{ color: C.primary }}>{draft.length.toLocaleString()}</b> 字符)。
            填好下面的信息就能保存,之后备课时随时能选用。
          </div>

          {/* 名称 */}
          <div style={{ marginBottom: '16px' }}>
            <label style={labelStyle}>
              助手名称 <span style={{ color: C.danger }}>*</span>
            </label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="例如:我的初中AI严苛审核员"
              maxLength={100}
              autoFocus
              style={{ ...inputStyle, width: '100%' }}
            />
          </div>

          {/* 货架选择(可选货架 >1 时显示;普通老师只有 personal 不显示,保持极简) */}
          {showShelfPicker && (
            <div style={{ marginBottom: '16px' }}>
              <label style={labelStyle}>
                保存到哪里 <span style={{ color: C.danger }}>*</span>
                <span style={{ color: C.textMuted, fontWeight: 400 }}> (这决定谁能用到这个助手)</span>
              </label>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', marginTop: '4px' }}>
                {shelfKeys.map(sk => {
                  const opt = SHELF_OPTIONS[sk]
                  const checked = shelf === sk
                  return (
                    <div key={sk}>
                      <label
                        style={{
                          display: 'flex', alignItems: 'flex-start', gap: '8px',
                          padding: '10px 12px', borderRadius: '8px',
                          border: `1.5px solid ${checked ? C.primary : C.border}`,
                          background: checked ? C.primaryLight : '#fff',
                          cursor: 'pointer',
                        }}
                      >
                        <input
                          type="radio"
                          name="shelf"
                          checked={checked}
                          onChange={() => setShelf(sk)}
                          style={{ cursor: 'pointer', accentColor: C.primary, marginTop: '2px' }}
                        />
                        <div style={{ flex: 1 }}>
                          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>
                            {opt.emoji} {opt.label}
                          </div>
                          <div style={{ fontSize: '11px', color: C.textSec, marginTop: '2px', lineHeight: 1.5 }}>
                            {opt.hint}
                          </div>
                        </div>
                      </label>

                      {/* 选中"发布到教研组"时,在该项下方展开教研组下拉 */}
                      {sk === 'group_teaching' && checked && (
                        <div style={{ margin: '6px 0 2px 28px' }}>
                          <select
                            value={selectedGroupID}
                            onChange={e => setSelectedGroupID(e.target.value)}
                            style={{ ...inputStyle, width: '100%', cursor: 'pointer' }}
                          >
                            {publishGroups.map(g => (
                              <option key={g.id} value={g.id}>
                                {g.name}
                                {g.role === 'lead' ? '(组长)' : '(骨干)'}
                                {g.school_name ? ` · ${g.school_name}` : ''}
                              </option>
                            ))}
                          </select>
                          <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px', lineHeight: 1.5 }}>
                            发布后,该教研组的老师在备课时都能选到这个助手。
                          </div>
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {/* 可发布范围加载中提示(仅在尚未确定货架且加载中时短暂出现) */}
          {loadingScope && !showShelfPicker && (
            <div style={{ marginBottom: '16px', fontSize: '12px', color: C.textMuted }}>
              正在确认你的发布范围…
            </div>
          )}

          {/* 学科 */}
          <div style={{ marginBottom: '16px' }}>
            <label style={labelStyle}>
              适用学科 <span style={{ color: C.textMuted, fontWeight: 400 }}>(默认取你主要教的科目,可改)</span>
            </label>
            <select
              value={subject}
              onChange={e => setSubject(e.target.value)}
              style={{ ...inputStyle, width: '100%', cursor: 'pointer' }}
            >
              {SUBJECTS.map(s => (
                <option key={s} value={s}>{s || '(不限学科)'}</option>
              ))}
            </select>
          </div>

          {/* 场景多选 */}
          <div style={{ marginBottom: '8px' }}>
            <label style={labelStyle}>
              适用场景 <span style={{ color: C.danger }}>*</span>
              <span style={{ color: C.textMuted, fontWeight: 400 }}> (这个助手在哪些备课环节出现,至少选 1 个)</span>
            </label>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '6px', marginTop: '4px' }}>
              {(Object.entries(ASSISTANT_SCENE_LABELS) as [AssistantScene, string][]).map(([scene, label]) => {
                const checked = scenes.includes(scene)
                return (
                  <label
                    key={scene}
                    style={{
                      display: 'flex', alignItems: 'center', gap: '6px',
                      padding: '7px 10px', borderRadius: '6px',
                      border: `1px solid ${checked ? C.primary : C.border}`,
                      background: checked ? C.primaryLight : '#fff',
                      cursor: 'pointer', fontSize: '13px', color: C.text,
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={() => toggleScene(scene)}
                      style={{ cursor: 'pointer', accentColor: C.primary }}
                    />
                    {label}
                  </label>
                )
              })}
            </div>
          </div>

          {/* 错误提示 */}
          {err && (
            <div style={{
              marginTop: '14px', padding: '10px 12px', borderRadius: '8px',
              background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.2)',
              color: C.danger, fontSize: '13px',
            }}>
              ⚠️ {err}
            </div>
          )}
        </div>

        {/* 底部操作栏 */}
        <div style={{
          padding: '12px 20px', borderTop: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'flex-end', gap: '8px',
          background: C.bg, flexShrink: 0,
        }}>
          <button
            onClick={() => { if (!saving) onClose() }}
            disabled={saving}
            style={{
              padding: '8px 16px', borderRadius: '7px',
              border: `1px solid ${C.borderMid}`, background: '#fff',
              color: C.textSec, fontSize: '13px',
              cursor: saving ? 'not-allowed' : 'pointer', opacity: saving ? 0.5 : 1,
            }}
          >取消</button>
          <button
            onClick={handleSave}
            disabled={saving}
            style={{
              padding: '8px 20px', borderRadius: '7px', border: 'none',
              background: saving ? C.borderMid : C.success,
              color: saving ? C.textMuted : '#fff',
              fontSize: '13px', fontWeight: 600,
              cursor: saving ? 'not-allowed' : 'pointer',
            }}
          >
            {saving ? '保存中...' : '✓ 保存助手'}
          </button>
        </div>
      </div>
    </div>
  )
}

/* ==================== 样式辅助 ==================== */

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: '12px', fontWeight: 600, color: C.textSec,
  marginBottom: '4px',
}

const inputStyle: React.CSSProperties = {
  padding: '8px 10px', borderRadius: '6px',
  border: `1px solid ${C.border}`,
  fontSize: '13px', color: C.text,
  outline: 'none', boxSizing: 'border-box',
  fontFamily: 'inherit', background: '#fff',
}
