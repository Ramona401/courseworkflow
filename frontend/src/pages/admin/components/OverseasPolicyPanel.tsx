/**
 * OverseasPolicyPanel.tsx — 学校境外模型授权管理面板（批二-A 新增）
 *
 * 用途：
 *   在 AdminPage「概览」Tab 底部展开式卡片中，由 admin 管理「哪些学校被授权走境外模型」。
 *   对应 school_model_policies 表。默认所有学校只能用境内模型（qwen-max）；
 *   仅被显式授权（overseas_enabled=true）的学校，AI 文本调用才放行境外模型（claude/gemini 等）。
 *
 * 交互（仿 KBAuthorizedPanel / OrgAdminsPanel）：
 *   - 顶部「授权新学校」：学校下拉选择（getAdminOrgs type=school）+ 可选备注 + 授权按钮。
 *   - 下方「已登记学校列表」：每行显示学校名 / 授权状态徽章 / 备注 / 授权人 / 更新时间；
 *     行内可一键开关（境外⇄境内）、移除（二次确认，移除=回到默认境内）。
 *
 * 后端对接（admin.ts）：
 *   getSchoolModelPolicies()  列出已登记策略
 *   setSchoolModelPolicy(id, {overseas_enabled, note})  授权/取消/改备注（UPSERT）
 *   deleteSchoolModelPolicy(id)  删除记录（=回到默认境内）
 *
 * 权限：仅 admin（路由 adminOnly + 概览 Tab 本就仅 admin 可见，双重限定）。
 * 即时生效：分流模块对「学校是否授权」每次实时查库无缓存，保存即生效。
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getSchoolModelPolicies, setSchoolModelPolicy, deleteSchoolModelPolicy,
  getAdminOrgs,
} from '@/api/admin'
import type { SchoolModelPolicyItem, OrgListItem } from '@/api/admin'
import { C, fmt } from './adminConstants'
import { ConfirmDialog } from './ConfirmDialog'

export function OverseasPolicyPanel() {
  // ==================== 状态 ====================

  /** 已登记策略列表 */
  const [policies, setPolicies] = useState<SchoolModelPolicyItem[]>([])
  /** 全部学校（供下拉选择，type=school） */
  const [schools, setSchools] = useState<OrgListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  /** 授权表单：选中学校 + 可选备注 */
  const [pickSchoolId, setPickSchoolId] = useState('')
  const [note, setNote] = useState('')
  const [granting, setGranting] = useState(false)

  /** 行内开关切换中的学校ID（防重复点击） */
  const [togglingId, setTogglingId] = useState('')

  /** 移除二次确认 */
  const [confirmRemove, setConfirmRemove] = useState<{ open: boolean; schoolId: string; name: string }>({
    open: false, schoolId: '', name: '',
  })

  // ==================== 加载 ====================

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError('')
      // 并行拉：已登记策略 + 全部学校
      const [pols, orgs] = await Promise.all([
        getSchoolModelPolicies(),
        getAdminOrgs({ type: 'school' }),
      ])
      setPolicies(pols)
      setSchools(orgs)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载学校授权数据失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  // 已登记的学校ID集合（下拉里把已登记的标注出来，避免重复添加困惑）
  const registeredIds = new Set(policies.map(p => p.school_id))

  // ==================== 授权新学校 ====================

  const handleGrant = useCallback(async () => {
    if (!pickSchoolId) { setError('请先选择要授权的学校'); return }
    try {
      setGranting(true)
      setError('')
      // 新授权默认 overseas_enabled=true（从下拉添加即表示要放行境外）
      await setSchoolModelPolicy(pickSchoolId, { overseas_enabled: true, note: note.trim() })
      setPickSchoolId('')
      setNote('')
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '授权失败')
    } finally {
      setGranting(false)
    }
  }, [pickSchoolId, note, load])

  // ==================== 行内开关：境外⇄境内 ====================

  const handleToggle = useCallback(async (p: SchoolModelPolicyItem) => {
    try {
      setTogglingId(p.school_id)
      setError('')
      await setSchoolModelPolicy(p.school_id, {
        overseas_enabled: !p.overseas_enabled,
        note: p.note, // 保留原备注
      })
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '切换失败')
    } finally {
      setTogglingId('')
    }
  }, [load])

  // ==================== 移除（二次确认）====================

  const doRemove = useCallback(async (schoolId: string) => {
    try {
      setError('')
      await deleteSchoolModelPolicy(schoolId)
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '移除失败')
    } finally {
      setConfirmRemove({ open: false, schoolId: '', name: '' })
    }
  }, [load])

  // ==================== 渲染 ====================

  return (
    <div style={{ padding: '16px 20px', background: 'rgba(245,158,11,0.04)', borderTop: `1px dashed ${C.border}` }}>

      {/* 移除二次确认弹窗 */}
      {confirmRemove.open && (
        <ConfirmDialog
          title="移除境外授权记录"
          message={`确认移除「${confirmRemove.name}」的境外授权记录？移除后该校回到默认境内模型（qwen-max），其老师的境外文本调用将被降级。`}
          onConfirm={() => doRemove(confirmRemove.schoolId)}
          onCancel={() => setConfirmRemove({ open: false, schoolId: '', name: '' })}
        />
      )}

      {/* 错误提示 */}
      {error && (
        <div style={{ fontSize: '12px', color: C.danger, marginBottom: '12px', padding: '8px 12px', background: C.dangerLight, borderRadius: '8px', lineHeight: 1.5 }}>
          ⚠ {error}
        </div>
      )}

      {/* 说明条 */}
      <div style={{
        padding: '10px 14px', borderRadius: '10px', marginBottom: '14px',
        background: C.warningLight, border: `1px solid ${C.warning}33`,
        fontSize: '12px', color: C.textSec, lineHeight: 1.6,
      }}>
        默认所有学校只能用<b>境内模型</b>（qwen-max）。仅在此<b>显式授权</b>的学校，其老师的 AI 文本调用才放行<b>境外模型</b>（claude/gemini 等）。
        授权<b>即时生效</b>（分流实时查库，无缓存）。
      </div>

      {/* 授权新学校 */}
      <div style={{ background: C.white, borderRadius: '10px', border: `1px solid ${C.border}`, padding: '14px', marginBottom: '16px' }}>
        <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '10px' }}>授权新学校走境外</div>
        <div style={{ display: 'flex', gap: '10px', alignItems: 'flex-start', flexWrap: 'wrap' }}>
          {/* 学校下拉 */}
          <select
            value={pickSchoolId}
            onChange={e => setPickSchoolId(e.target.value)}
            style={{
              flex: '1 1 240px', minWidth: '200px', padding: '9px 12px', borderRadius: '8px',
              border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none',
              background: C.white, cursor: 'pointer', color: C.text,
            }}
          >
            <option value="">— 选择学校 —</option>
            {schools.map(s => (
              <option key={s.id} value={s.id}>
                {s.name}{registeredIds.has(s.id) ? '（已登记）' : ''}
              </option>
            ))}
          </select>
          {/* 备注（可选） */}
          <input
            value={note}
            onChange={e => setNote(e.target.value)}
            placeholder="备注（可选，授权原因/用途）"
            style={{
              flex: '2 1 280px', minWidth: '200px', padding: '9px 12px', borderRadius: '8px',
              border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none',
              background: C.white, color: C.text, boxSizing: 'border-box',
            }}
          />
          {/* 授权按钮 */}
          <button
            onClick={handleGrant}
            disabled={granting || !pickSchoolId}
            style={{
              padding: '9px 18px', borderRadius: '8px', border: 'none',
              background: (!pickSchoolId || granting) ? '#E5E7EB' : C.warning,
              color: (!pickSchoolId || granting) ? '#9CA3AF' : '#fff',
              fontSize: '13px', fontWeight: 600, whiteSpace: 'nowrap',
              cursor: (!pickSchoolId || granting) ? 'not-allowed' : 'pointer',
            }}>
            {granting ? '授权中...' : '+ 授权境外'}
          </button>
        </div>
      </div>

      {/* 已登记学校列表 */}
      <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '10px' }}>
        已登记学校
        {policies.length > 0 && (
          <span style={{ fontSize: '11px', padding: '1px 7px', borderRadius: '10px', background: C.warningLight, color: C.warning, fontWeight: 600, marginLeft: '8px' }}>
            共 {policies.length} 所
          </span>
        )}
      </div>

      {loading ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>加载中...</div>
      ) : policies.length === 0 ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>
          暂无登记学校（全部学校默认走境内模型）。在上方选择学校授权境外。
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {policies.map(p => {
            const isToggling = togglingId === p.school_id
            return (
              <div key={p.school_id} style={{
                display: 'flex', alignItems: 'center', gap: '10px',
                padding: '10px 14px', borderRadius: '10px',
                background: C.white, border: `1px solid ${C.border}`,
              }}>
                {/* 学校名 + 状态徽章 + 备注 + 授权人 */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                    <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>
                      {p.school_name || p.school_id}
                    </span>
                    <span style={{
                      fontSize: '11px', padding: '2px 9px', borderRadius: '10px', fontWeight: 700,
                      background: p.overseas_enabled ? C.successLight : C.bg,
                      color: p.overseas_enabled ? C.success : C.textMuted,
                      border: `1px solid ${p.overseas_enabled ? C.success + '44' : C.border}`,
                    }}>
                      {p.overseas_enabled ? '🌐 境外已授权' : '🇨🇳 已关闭(境内)'}
                    </span>
                  </div>
                  <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '3px' }}>
                    {p.note ? `备注：${p.note} · ` : ''}
                    {p.granted_by_name ? `授权人：${p.granted_by_name} · ` : ''}
                    更新于 {fmt(p.updated_at)}
                  </div>
                </div>
                {/* 行内开关 */}
                <button
                  onClick={() => handleToggle(p)}
                  disabled={isToggling}
                  style={{
                    padding: '5px 12px', borderRadius: '7px', whiteSpace: 'nowrap',
                    border: `1px solid ${p.overseas_enabled ? C.border : C.success + '66'}`,
                    background: isToggling ? '#E5E7EB' : (p.overseas_enabled ? C.white : C.successLight),
                    color: isToggling ? '#9CA3AF' : (p.overseas_enabled ? C.textSec : C.success),
                    fontSize: '12px', fontWeight: 600,
                    cursor: isToggling ? 'not-allowed' : 'pointer',
                  }}>
                  {isToggling ? '处理中...' : (p.overseas_enabled ? '切到境内' : '开启境外')}
                </button>
                {/* 移除 */}
                <button
                  onClick={() => setConfirmRemove({ open: true, schoolId: p.school_id, name: p.school_name || p.school_id })}
                  style={{
                    padding: '5px 10px', borderRadius: '7px',
                    border: '1px solid #FEE2E2', background: '#FEF2F2',
                    color: '#EF4444', fontSize: '12px', cursor: 'pointer', fontWeight: 500, whiteSpace: 'nowrap',
                  }}>
                  移除
                </button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
