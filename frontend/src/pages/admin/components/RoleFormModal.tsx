/**
 * RoleFormModal.tsx — 新建/编辑角色弹窗
 *
 * 功能：
 *   - 新建模式：填写英文标识/中文名/描述/继承自；选择继承后自动勾选该角色权限
 *   - 编辑模式：英文标识只读，可修改中文名/描述；权限 checkbox 自由勾选
 *   - 权限配置区：按分组（Pipeline/系统配置/教案系统）展示全部18项
 *   - 每组有"全选本组"按钮；顶部有"全选所有"按钮
 *   - 保存时调用 createRole / updateRole + updateRolePermissions
 */
import { useState, useEffect, useCallback } from 'react'
import { C } from './adminConstants'
import {
  PERMISSION_GROUPS, ALL_PERMISSIONS, BASE_ROLE_OPTIONS,
  SYSTEM_ROLES, type PermissionDef,
} from './roleConstants'
import {
  createRole, updateRole, updateRolePermissions, getRolePermissions,
} from '@/api/roles'
import type { RoleListItem } from '@/api/roles'

// ==================== Props ====================

interface RoleFormModalProps {
  mode: 'create' | 'edit'
  initial?: RoleListItem          // 编辑时传入
  onClose: () => void
  onSaved: () => void
  showToast: (msg: string, type: 'success' | 'error') => void
}

// ==================== 组件 ====================

export function RoleFormModal({ mode, initial, onClose, onSaved, showToast }: RoleFormModalProps) {

  // ---- 基本信息表单状态 ----
  const [roleCode,    setRoleCode]    = useState(initial?.role_code    ?? '')
  const [displayName, setDisplayName] = useState(initial?.display_name ?? '')
  const [description, setDescription] = useState(initial?.description  ?? '')
  const [baseRole,    setBaseRole]    = useState(initial?.base_role     ?? '')

  // ---- 权限勾选状态：Set<permissionCode> ----
  const [checkedCodes, setCheckedCodes] = useState<Set<string>>(new Set())
  const [permLoading,  setPermLoading]  = useState(false)
  const [saving,       setSaving]       = useState(false)
  const [errors,       setErrors]       = useState<Record<string, string>>({})

  // ---- 编辑模式：加载已有权限 ----
  const loadPermissions = useCallback(async () => {
    if (mode !== 'edit' || !initial) return
    try {
      setPermLoading(true)
      const perms = await getRolePermissions(initial.id)
      setCheckedCodes(new Set(perms.map(p => p.permission_code)))
    } catch {
      showToast('加载权限失败', 'error')
    } finally {
      setPermLoading(false)
    }
  }, [mode, initial, showToast])

  useEffect(() => { loadPermissions() }, [loadPermissions])

  // ---- 继承角色变更时自动勾选权限 ----
  const handleBaseRoleChange = (val: string) => {
    setBaseRole(val)
    if (!val) {
      // 不继承：清空权限
      setCheckedCodes(new Set())
      return
    }
    // 找到对应内置角色的权限列表并全部勾选
    const sysRole = SYSTEM_ROLES.find(r => r.code === val)
    if (sysRole) {
      setCheckedCodes(new Set(sysRole.permissionCodes))
    }
  }

  // ---- 单个权限 checkbox 切换 ----
  const togglePerm = (code: string) => {
    setCheckedCodes(prev => {
      const next = new Set(prev)
      if (next.has(code)) next.delete(code)
      else next.add(code)
      return next
    })
  }

  // ---- 整组全选/取消全选 ----
  const toggleGroup = (perms: PermissionDef[]) => {
    const codes = perms.map(p => p.code)
    const allChecked = codes.every(c => checkedCodes.has(c))
    setCheckedCodes(prev => {
      const next = new Set(prev)
      if (allChecked) codes.forEach(c => next.delete(c))
      else codes.forEach(c => next.add(c))
      return next
    })
  }

  // ---- 全选/取消全选所有权限 ----
  const toggleAll = () => {
    const all = ALL_PERMISSIONS.map(p => p.code)
    const allChecked = all.every(c => checkedCodes.has(c))
    if (allChecked) setCheckedCodes(new Set())
    else setCheckedCodes(new Set(all))
  }

  // ---- 表单校验 ----
  const validate = (): boolean => {
    const e: Record<string, string> = {}
    if (!roleCode.trim())    e.roleCode    = '请填写英文标识'
    if (!/^[a-z][a-z0-9_]*$/.test(roleCode.trim()))
      e.roleCode = '只能包含小写字母、数字和下划线，且以字母开头'
    if (!displayName.trim()) e.displayName = '请填写中文名称'
    setErrors(e)
    return Object.keys(e).length === 0
  }

  // ---- 保存 ----
  const handleSave = async () => {
    if (!validate()) return
    setSaving(true)
    try {
      const permPayload = Array.from(checkedCodes).map(code => {
        const def = ALL_PERMISSIONS.find(p => p.code === code)!
        return { permission_code: code, resource: def.resource, action: def.action }
      })

      if (mode === 'create') {
        // 1. 创建角色
        const created = await createRole({
          role_code:    roleCode.trim(),
          display_name: displayName.trim(),
          description:  description.trim(),
          base_role:    baseRole || undefined,
        })
        // 2. 如果手动调整了权限（与继承不同），再更新权限
        //    始终调用一次保证权限与 UI 完全一致
        await updateRolePermissions(created.id, { permissions: permPayload })
        showToast('角色创建成功', 'success')
      } else {
        // 编辑：更新基本信息 + 权限
        await updateRole(initial!.id, {
          display_name: displayName.trim(),
          description:  description.trim(),
        })
        await updateRolePermissions(initial!.id, { permissions: permPayload })
        showToast('角色更新成功', 'success')
      }
      onSaved()
      onClose()
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '保存失败', 'error')
    } finally {
      setSaving(false)
    }
  }

  // ---- 计算全选状态 ----
  const allCount     = ALL_PERMISSIONS.length
  const checkedCount = checkedCodes.size
  const allChecked   = checkedCount === allCount
  const halfChecked  = checkedCount > 0 && checkedCount < allCount

  // ==================== 渲染 ====================

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '9px 12px', borderRadius: '8px',
    border: `1px solid ${C.border}`, fontSize: '14px',
    outline: 'none', background: C.white, color: C.text,
    boxSizing: 'border-box',
  }
  const labelStyle: React.CSSProperties = {
    display: 'block', fontSize: '13px', fontWeight: 600,
    color: C.textSec, marginBottom: '6px',
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 11000,
      background: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(4px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: '20px',
    }}>
      <div style={{
        background: C.white, borderRadius: '18px',
        width: '640px', maxHeight: '88vh',
        display: 'flex', flexDirection: 'column',
        boxShadow: '0 24px 64px rgba(0,0,0,0.22)',
        overflow: 'hidden',
      }}>

        {/* ---- 弹窗标题 ---- */}
        <div style={{
          padding: '20px 24px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          flexShrink: 0,
        }}>
          <div>
            <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>
              {mode === 'create' ? '🎭 新建自定义角色' : `✏️ 编辑角色：${initial?.display_name}`}
            </div>
            <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>
              {mode === 'create' ? '配置角色基本信息和权限' : '修改角色名称和权限配置'}
            </div>
          </div>
          <button onClick={onClose} style={{ width: '30px', height: '30px', borderRadius: '50%', border: 'none', background: C.bg, cursor: 'pointer', fontSize: '16px', color: C.textSec, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✕</button>
        </div>

        {/* ---- 可滚动内容区 ---- */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '20px 24px' }}>

          {/* == 基本信息区 == */}
          <div style={{ marginBottom: '20px' }}>
            <div style={{ fontSize: '14px', fontWeight: 700, color: C.text, marginBottom: '14px', paddingBottom: '8px', borderBottom: `1px solid ${C.border}` }}>
              📋 基本信息
            </div>

            {/* 英文标识 */}
            <div style={{ marginBottom: '14px' }}>
              <label style={labelStyle}>
                英文标识 <span style={{ color: C.danger }}>*</span>
                {mode === 'edit' && <span style={{ color: C.textMuted, fontWeight: 400, marginLeft: '6px' }}>（创建后不可修改）</span>}
              </label>
              <input
                value={roleCode}
                onChange={e => { setRoleCode(e.target.value); setErrors(p => ({ ...p, roleCode: '' })) }}
                disabled={mode === 'edit'}
                placeholder="如：custom_reviewer"
                style={{
                  ...inputStyle,
                  background: mode === 'edit' ? C.bg : C.white,
                  color: mode === 'edit' ? C.textSec : C.text,
                  borderColor: errors.roleCode ? C.danger : C.border,
                  cursor: mode === 'edit' ? 'not-allowed' : 'text',
                }}
              />
              {errors.roleCode && <div style={{ fontSize: '12px', color: C.danger, marginTop: '4px' }}>{errors.roleCode}</div>}
              <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>小写字母、数字和下划线，以字母开头</div>
            </div>

            {/* 中文名称 */}
            <div style={{ marginBottom: '14px' }}>
              <label style={labelStyle}>中文名称 <span style={{ color: C.danger }}>*</span></label>
              <input
                value={displayName}
                onChange={e => { setDisplayName(e.target.value); setErrors(p => ({ ...p, displayName: '' })) }}
                placeholder="如：内容审核员"
                style={{ ...inputStyle, borderColor: errors.displayName ? C.danger : C.border }}
              />
              {errors.displayName && <div style={{ fontSize: '12px', color: C.danger, marginTop: '4px' }}>{errors.displayName}</div>}
            </div>

            {/* 描述 */}
            <div style={{ marginBottom: '14px' }}>
              <label style={labelStyle}>描述</label>
              <textarea
                value={description}
                onChange={e => setDescription(e.target.value)}
                placeholder="简要描述该角色的职责范围..."
                rows={2}
                style={{ ...inputStyle, resize: 'vertical', minHeight: '64px', fontFamily: 'inherit' }}
              />
            </div>

            {/* 继承自（仅新建时显示） */}
            {mode === 'create' && (
              <div>
                <label style={labelStyle}>继承自（选填）</label>
                <select
                  value={baseRole}
                  onChange={e => handleBaseRoleChange(e.target.value)}
                  style={{ ...inputStyle }}
                >
                  {BASE_ROLE_OPTIONS.map(o => (
                    <option key={o.value} value={o.value}>{o.label}</option>
                  ))}
                </select>
                <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>
                  选择继承角色后将自动勾选对应权限，可在下方进一步调整
                </div>
              </div>
            )}
          </div>

          {/* == 权限配置区 == */}
          <div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '14px', paddingBottom: '8px', borderBottom: `1px solid ${C.border}` }}>
              <div style={{ fontSize: '14px', fontWeight: 700, color: C.text }}>
                🔑 权限配置
                <span style={{ fontSize: '12px', fontWeight: 400, color: C.textMuted, marginLeft: '8px' }}>
                  已选 {checkedCount} / {allCount} 项
                </span>
              </div>
              <button
                onClick={toggleAll}
                style={{
                  padding: '5px 14px', borderRadius: '7px',
                  border: `1px solid ${allChecked ? C.danger : C.primary}`,
                  background: allChecked ? C.dangerLight : C.primaryLight,
                  color: allChecked ? C.danger : C.primary,
                  fontSize: '12px', fontWeight: 600, cursor: 'pointer',
                }}
              >
                {halfChecked ? '全选所有' : allChecked ? '取消全选' : '全选所有'}
              </button>
            </div>

            {permLoading ? (
              <div style={{ padding: '24px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>加载权限中...</div>
            ) : (
              PERMISSION_GROUPS.map(group => {
                const groupCodes  = group.permissions.map(p => p.code)
                const groupCheckedCount = groupCodes.filter(c => checkedCodes.has(c)).length
                const groupAllChecked   = groupCheckedCount === group.permissions.length
                const groupHalf         = groupCheckedCount > 0 && !groupAllChecked

                return (
                  <div key={group.groupKey} style={{ marginBottom: '16px' }}>
                    {/* 分组标题行 */}
                    <div style={{
                      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                      padding: '8px 12px', borderRadius: '8px',
                      background: C.bg, marginBottom: '8px',
                    }}>
                      <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, display: 'flex', alignItems: 'center', gap: '6px' }}>
                        <span>{group.icon}</span>
                        <span>{group.groupName}</span>
                        <span style={{ fontSize: '11px', color: C.textMuted, fontWeight: 400 }}>
                          {groupCheckedCount}/{group.permissions.length}
                        </span>
                      </div>
                      <button
                        onClick={() => toggleGroup(group.permissions)}
                        style={{
                          padding: '3px 10px', borderRadius: '6px', fontSize: '11px', fontWeight: 600,
                          border: `1px solid ${groupAllChecked ? C.danger + '66' : C.primary + '66'}`,
                          background: groupAllChecked ? C.dangerLight : groupHalf ? C.primaryLight : C.primaryLight,
                          color: groupAllChecked ? C.danger : C.primary,
                          cursor: 'pointer',
                        }}
                      >
                        {groupAllChecked ? '取消全组' : '全选本组'}
                      </button>
                    </div>

                    {/* 权限项列表 */}
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '6px', paddingLeft: '4px' }}>
                      {group.permissions.map(perm => {
                        const checked = checkedCodes.has(perm.code)
                        return (
                          <label
                            key={perm.code}
                            style={{
                              display: 'flex', alignItems: 'center', gap: '8px',
                              padding: '8px 10px', borderRadius: '7px', cursor: 'pointer',
                              border: `1px solid ${checked ? C.primary + '33' : C.border}`,
                              background: checked ? C.primaryLight : C.white,
                              transition: 'all 150ms ease',
                            }}
                          >
                            <input
                              type="checkbox"
                              checked={checked}
                              onChange={() => togglePerm(perm.code)}
                              style={{ width: '15px', height: '15px', cursor: 'pointer', accentColor: C.primary, flexShrink: 0 }}
                            />
                            <div style={{ flex: 1, minWidth: 0 }}>
                              <div style={{ fontSize: '13px', color: checked ? C.primary : C.text, fontWeight: checked ? 600 : 400 }}>
                                {perm.label}
                              </div>
                              <div style={{ fontSize: '10px', color: C.textMuted, fontFamily: 'monospace' }}>
                                {perm.code}
                              </div>
                            </div>
                          </label>
                        )
                      })}
                    </div>
                  </div>
                )
              })
            )}
          </div>
        </div>

        {/* ---- 底部操作栏 ---- */}
        <div style={{
          padding: '16px 24px', borderTop: `1px solid ${C.border}`,
          display: 'flex', gap: '10px', justifyContent: 'flex-end',
          background: C.bg, flexShrink: 0,
        }}>
          <button onClick={onClose} style={{ padding: '10px 24px', borderRadius: '9px', border: `1px solid ${C.border}`, background: C.white, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>
            取消
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            style={{
              padding: '10px 28px', borderRadius: '9px', border: 'none',
              background: saving ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`,
              color: '#fff', fontSize: '14px', fontWeight: 600,
              cursor: saving ? 'not-allowed' : 'pointer',
            }}
          >
            {saving ? '保存中...' : mode === 'create' ? '创建角色' : '保存修改'}
          </button>
        </div>
      </div>
    </div>
  )
}
