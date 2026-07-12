/**
 * ShareRecipeModal.tsx — 配方共享弹窗（把个人配方设置成教研组/学校配方）
 *
 * 背景：
 *   配方创建时一律落 scope=personal（个人私有，别人搜不到）。教研组长和学校管理员
 *   需要把自己沉淀的好配方"下沉"给全组/全校老师选用，但此前前端没有共享入口，
 *   导致组长/校管反馈"只能设置个人配方，设置不了学校配方"。本弹窗即补上这个入口。
 *
 * 归属选择口径（与 SaveAssistantModal / 课件模板发布完全一致）：
 *   打开时调 getPublishTargets() 拿当前用户真实可发布范围，动态展示货架：
 *     - 👥 发布到教研组(group + 教研组ID)  : 我是某组 lead/backbone 时显示，需选具体组
 *     - 🏫 推荐给全校老师(school + 学校ID)  : 我是学校管理员时显示
 *   两档分别对应后端 ShareRecipe 接受的 group / school 两种 scope，
 *   scope_ref_id 用 getPublishTargets 返回的真实教研组ID / 学校ID（不让用户手填UUID）。
 *
 * 与 AI 助手共享的差异（为什么不照搬 SaveAssistantModal）：
 *   - 配方 ShareRecipe 后端只接受 group / school 两档（不支持 system / personal），
 *     故本弹窗无"全平台通用""只给我自己用"选项（个人是默认态，无需在此设）。
 *   - 配方 ShareRecipe 要求显式传 scope_ref_id，故必须用带真实ID的 getPublishTargets，
 *     不能用只给布尔标志的 getMyPublishGroups（那个拿不到学校ID）。
 *
 * Props 契约：
 *   open        - 是否显示
 *   recipeId    - 要共享的配方ID
 *   recipeName  - 配方名称（标题展示用）
 *   currentScope- 配方当前 scope（personal/group/school，用于展示当前状态）
 *   onClose     - 关闭回调
 *   onShared    - 共享成功回调（父组件据此刷新列表 + 提示），传回最终 scope
 */

import { useState, useEffect } from 'react'
import { shareRecipe } from '@/api/recipes'
import { getPublishTargets, type PublishTargetGroup } from '@/api/coursewares'

/* ==================== 样式常量（与配方列表页 RecipesPage 保持一致） ==================== */
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

/* ==================== 货架（shelf）定义 ==================== */
//
// 配方共享只有两档：教研组级 / 全校级。提交时映射回后端 ShareRecipe 的 scope + scope_ref_id。

/** 货架内部 key */
type ShelfKey = 'group' | 'school'

/** 单个货架选项的展示信息 */
interface ShelfOption {
  emoji: string
  label: string   // 人话标题
  hint: string    // 一句话说明给谁用
}

/** 货架展示文案 */
const SHELF_OPTIONS: Record<ShelfKey, ShelfOption> = {
  group:  { emoji: '👥', label: '发布到教研组',   hint: '只共享给所选教研组的老师，组内备课时可在配方下拉里选用' },
  school: { emoji: '🏫', label: '推荐给全校老师', hint: '作为本校共享配方，全校老师备课时都能在配方下拉里选用' },
}

/* ==================== Props 类型 ==================== */
export interface ShareRecipeModalProps {
  open: boolean
  recipeId: string
  recipeName: string
  currentScope: string
  onClose: () => void
  /** 共享成功回调，传回最终 scope（group/school） */
  onShared: (scope: ShelfKey) => void
}

/* ==================== 主组件 ==================== */
export default function ShareRecipeModal(props: ShareRecipeModalProps) {
  const { open, recipeId, recipeName, currentScope, onClose, onShared } = props

  // ==================== 表单状态 ====================
  const [shelf, setShelf]             = useState<ShelfKey>('group')   // 当前选中货架
  const [selectedGroupID, setSelectedGroupID] = useState('')         // 教研组级时选中的组
  const [saving, setSaving]           = useState(false)
  const [err, setErr]                 = useState<string | null>(null)

  // ==================== 可发布范围（来自 getPublishTargets 接口） ====================
  const [shelfKeys, setShelfKeys]         = useState<ShelfKey[]>([])           // 当前用户可选货架
  const [publishGroups, setPublishGroups] = useState<PublishTargetGroup[]>([]) // 可发布的教研组
  const [schoolID, setSchoolID]           = useState('')                       // 可发布的学校ID
  const [schoolName, setSchoolName]       = useState('')                       // 学校名（展示用）
  const [loadingScope, setLoadingScope]   = useState(false)                    // 拉取可发布范围中

  // ==================== open 时初始化 + 拉取可发布范围 ====================
  useEffect(() => {
    if (!open) return
    // 基础重置
    setShelf('group')
    setSelectedGroupID('')
    setErr(null)
    setSaving(false)
    setShelfKeys([])
    setPublishGroups([])
    setSchoolID('')
    setSchoolName('')

    let cancelled = false
    setLoadingScope(true)
    getPublishTargets()
      .then((res) => {
        if (cancelled) return
        const keys: ShelfKey[] = []
        // 教研组档：有 lead/backbone 的组才显示
        if (res.groups && res.groups.length > 0) {
          keys.push('group')
          setPublishGroups(res.groups)
          setSelectedGroupID(res.groups[0].id) // 默认选第一个组
        }
        // 全校档：是学校管理员（school.available）才显示
        if (res.school && res.school.available && res.school.school_id) {
          keys.push('school')
          setSchoolID(res.school.school_id)
          setSchoolName(res.school.name || '')
        }
        setShelfKeys(keys)
        // 默认选中第一个可用货架（优先教研组，没有则全校）
        if (keys.length > 0) setShelf(keys[0])
      })
      .catch(() => {
        if (cancelled) return
        setShelfKeys([]) // 拉取失败：无可发布范围（下方会提示）
      })
      .finally(() => {
        if (!cancelled) setLoadingScope(false)
      })

    return () => { cancelled = true }
  }, [open])

  // ==================== ESC 关闭 ====================
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !saving) onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, saving, onClose])

  // ==================== 提交共享 ====================
  const handleShare = async () => {
    if (saving) return
    setErr(null)

    // 确定 scope_ref_id
    let scopeRefID = ''
    if (shelf === 'group') {
      if (!selectedGroupID) { setErr('请选择要发布到哪个教研组'); return }
      scopeRefID = selectedGroupID
    } else {
      if (!schoolID) { setErr('未找到你管理的学校，无法发布到全校'); return }
      scopeRefID = schoolID
    }

    setSaving(true)
    try {
      await shareRecipe(recipeId, { scope: shelf, scope_ref_id: scopeRefID })
      onShared(shelf)
    } catch (e) {
      setErr(e instanceof Error ? e.message : '共享失败，请重试')
      setSaving(false)
    }
  }

  // ==================== 未 open 不渲染 ====================
  if (!open) return null

  // 当前没有任何可发布范围（既不是组长也不是校管）
  const noScope = !loadingScope && shelfKeys.length === 0

  return (
    <div
      onClick={() => { if (!saving) onClose() }}
      style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        background: 'rgba(17,24,39,0.5)', zIndex: 10001,
        display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '24px',
      }}
    >
      {/* 弹窗本体 */}
      <div
        onClick={e => e.stopPropagation()}
        style={{
          width: '500px', maxWidth: '100%', maxHeight: '90vh',
          background: C.card, borderRadius: '14px',
          boxShadow: '0 24px 64px rgba(0,0,0,0.18)',
          display: 'flex', flexDirection: 'column', overflow: 'hidden',
        }}
      >
        {/* 标题栏 */}
        <div style={{
          padding: '16px 20px', borderBottom: `1px solid ${C.border}`,
          background: 'linear-gradient(135deg,rgba(79,123,232,0.06),rgba(79,123,232,0.02))',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0,
        }}>
          <span style={{ fontSize: '15px', fontWeight: 700, color: C.text }}>🔗 共享配方</span>
          <button
            onClick={() => { if (!saving) onClose() }}
            style={{ background: 'none', border: 'none', cursor: saving ? 'not-allowed' : 'pointer', fontSize: '20px', color: C.textMuted, lineHeight: 1 }}
          >×</button>
        </div>

        {/* 表单区 */}
        <div style={{ flex: 1, overflow: 'auto', padding: '20px 24px' }}>
          {/* 配方信息 */}
          <div style={{
            marginBottom: '16px', padding: '10px 12px', borderRadius: '8px',
            background: C.primaryLight, border: '1px solid rgba(79,123,232,0.15)',
            fontSize: '13px', color: C.textSec, lineHeight: 1.6,
          }}>
            将配方 <b style={{ color: C.primary }}>📦 {recipeName}</b> 共享出去，
            被共享范围内的老师在备课时就能直接选用它。
            {currentScope === 'group' || currentScope === 'school' ? (
              <span style={{ display: 'block', marginTop: '4px', color: C.accent }}>
                ⓘ 这个配方当前已是{currentScope === 'school' ? '全校' : '教研组'}共享，重新共享会更新归属。
              </span>
            ) : null}
          </div>

          {/* 加载中 */}
          {loadingScope && (
            <div style={{ fontSize: '13px', color: C.textMuted, padding: '8px 0' }}>
              正在确认你的发布范围…
            </div>
          )}

          {/* 无可发布范围 */}
          {noScope && (
            <div style={{
              padding: '14px 16px', borderRadius: '8px',
              background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.25)',
              color: '#92400E', fontSize: '13px', lineHeight: 1.7,
            }}>
              你目前不是任何教研组的组长/骨干，也不是学校管理员，暂时无法把配方共享给教研组或全校。
              <br />如需共享，请联系管理员把你设为教研组组长或学校管理员。
            </div>
          )}

          {/* 货架选择 */}
          {!loadingScope && shelfKeys.length > 0 && (
            <div>
              <label style={labelStyle}>
                共享给谁 <span style={{ color: C.danger }}>*</span>
              </label>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', marginTop: '4px' }}>
                {shelfKeys.map(sk => {
                  const opt = SHELF_OPTIONS[sk]
                  const checked = shelf === sk
                  return (
                    <div key={sk}>
                      <label style={{
                        display: 'flex', alignItems: 'flex-start', gap: '8px',
                        padding: '10px 12px', borderRadius: '8px',
                        border: `1.5px solid ${checked ? C.primary : C.border}`,
                        background: checked ? C.primaryLight : '#fff', cursor: 'pointer',
                      }}>
                        <input
                          type="radio" name="recipe-shelf" checked={checked}
                          onChange={() => setShelf(sk)}
                          style={{ cursor: 'pointer', accentColor: C.primary, marginTop: '2px' }}
                        />
                        <div style={{ flex: 1 }}>
                          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>
                            {opt.emoji} {opt.label}
                            {sk === 'school' && schoolName ? (
                              <span style={{ fontSize: '11px', fontWeight: 400, color: C.textMuted }}> · {schoolName}</span>
                            ) : null}
                          </div>
                          <div style={{ fontSize: '11px', color: C.textSec, marginTop: '2px', lineHeight: 1.5 }}>
                            {opt.hint}
                          </div>
                        </div>
                      </label>

                      {/* 选中"发布到教研组"时展开教研组下拉 */}
                      {sk === 'group' && checked && (
                        <div style={{ margin: '6px 0 2px 28px' }}>
                          <select
                            value={selectedGroupID}
                            onChange={e => setSelectedGroupID(e.target.value)}
                            style={{ ...inputStyle, width: '100%', cursor: 'pointer' }}
                          >
                            {publishGroups.map(g => (
                              <option key={g.id} value={g.id}>
                                {g.name}
                                {g.role === 'lead' ? '（组长）' : '（骨干）'}
                                {g.school_name ? ` · ${g.school_name}` : ''}
                              </option>
                            ))}
                          </select>
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {/* 错误提示 */}
          {err && (
            <div style={{
              marginTop: '14px', padding: '10px 12px', borderRadius: '8px',
              background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.2)',
              color: C.danger, fontSize: '13px',
            }}>⚠️ {err}</div>
          )}
        </div>

        {/* 底部操作栏 */}
        <div style={{
          padding: '12px 20px', borderTop: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'flex-end', gap: '8px', background: C.bg, flexShrink: 0,
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
            onClick={handleShare}
            disabled={saving || noScope || loadingScope || shelfKeys.length === 0}
            style={{
              padding: '8px 20px', borderRadius: '7px', border: 'none',
              background: (saving || noScope || loadingScope || shelfKeys.length === 0) ? C.borderMid : C.primary,
              color: (saving || noScope || loadingScope || shelfKeys.length === 0) ? C.textMuted : '#fff',
              fontSize: '13px', fontWeight: 600,
              cursor: (saving || noScope || loadingScope || shelfKeys.length === 0) ? 'not-allowed' : 'pointer',
            }}
          >{saving ? '共享中...' : '✓ 确认共享'}</button>
        </div>
      </div>
    </div>
  )
}

/* ==================== 样式辅助 ==================== */
const labelStyle: React.CSSProperties = {
  display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '4px',
}

const inputStyle: React.CSSProperties = {
  padding: '8px 10px', borderRadius: '6px',
  border: `1px solid ${C.border}`, fontSize: '13px', color: C.text,
  outline: 'none', boxSizing: 'border-box', fontFamily: 'inherit', background: '#fff',
}
