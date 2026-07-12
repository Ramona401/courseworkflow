/**
 * CreateUserModal.tsx — 新建用户弹窗
 * 字段:登录用户名 / 显示名称 / 初始密码 / 系统角色
 *
 * 归属治理批C（任命唯一事实源）改动:
 *   - admin 的可创建角色去掉 senior_operator（region_admin/district_inspector 原本就不在）——
 *     学校管理员/区域管理员为任命制身份,不允许建号直接授予:先建教师账号(或选已有账号),
 *     再到「组织架构→对应卡片→🛡️ 管理员」任命,B13 会自动同步升级其系统身份。
 *     后端 user_service.validateCreateUserReq 有同口径校验兜底(防绕过前端直连 API)。
 *   - admin 视角新增引导提示框说明上述流程。
 *
 * v122 改动(AdminPage 权限统一):
 *   - 角色下拉按登录者角色过滤:
 *     * admin 可创建 系统管理员/骨干教师/普通教师(批C后)
 *     * senior_operator 只能创建 operator / viewer(不能创建 admin 或其他学校管理员)
 *   - 默认选中 operator(最常用的骨干教师角色)
 *
 * Phase6.2 说明(区域管理员):
 *   - region_admin(区域管理员)按后端 permission_matrix 仅有 user:view 不含 create,
 *     因此不进入 availableRoles 任何分支(返回空数组),且 AdminPage 已对其隐藏“新建用户”按钮,
 *     正常情况下 region_admin 不会打开本弹窗。此处保险起见:空选项时禁用创建按钮并提示无权限。
 */
import { useState, useMemo } from 'react'
import { createAdminUser } from '@/api/admin'
import { useAuth } from '@/store/auth'
import { C, ROLE_OPTIONS, APPOINTMENT_ONLY_ROLES } from './adminConstants'

interface CreateUserModalProps {
  onClose: () => void
  onCreated: () => void
}

export function CreateUserModal({ onClose, onCreated }: CreateUserModalProps) {
  const { user } = useAuth()
  const [form, setForm] = useState({
    username: '', display_name: '', password: '', role: 'operator',
  })
  const [saving, setSaving] = useState(false)
  const [error, setError]   = useState('')

  // v122:按登录者角色过滤可创建的角色选项
  // 批C:任命制身份(senior_operator/region_admin)一律不可建号直接授予,district_inspector 沿旧规则由 admin 统一开通渠道之外也不提供
  const availableRoles = useMemo(() => {
    // 去掉“全部角色”占位项 + 区域教研员 + 任命制身份(学校管理员/区域管理员)
    const allRoles = ROLE_OPTIONS.filter(
      r => r.value && r.value !== 'district_inspector' && !APPOINTMENT_ONLY_ROLES.includes(r.value)
    )
    if (user?.role === 'admin') {
      return allRoles
    }
    if (user?.role === 'senior_operator') {
      // 学校管理员只能创建低于自己级别的角色
      return allRoles.filter(r => r.value === 'operator' || r.value === 'viewer')
    }
    // 其他角色(含 region_admin)不应该走到这里(按钮已隐藏 + 路由层已拦截),保险起见返回空
    return []
  }, [user?.role])

  // 无可创建角色时禁止创建(防御:region_admin 等误入)
  const noCreatableRole = availableRoles.length === 0

  const handleCreate = async () => {
    if (noCreatableRole) {
      setError('当前角色无权创建用户'); return
    }
    if (!form.username.trim() || !form.display_name.trim() || form.password.length < 6) {
      setError('请填写完整信息(密码至少6位)'); return
    }
    try {
      setSaving(true); setError('')
      await createAdminUser(form)
      onCreated(); onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally { setSaving(false) }
  }

  const fields = [
    { key: 'username',     label: '登录用户名', placeholder: '字母数字下划线', type: 'text'     },
    { key: 'display_name', label: '显示名称',   placeholder: '例如:张老师',   type: 'text'     },
    { key: 'password',     label: '初始密码',   placeholder: '至少6位',        type: 'password' },
  ]

  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 10000,
        background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{ background: C.white, borderRadius: '20px', width: '460px', overflow: 'hidden', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        {/* 头部 */}
        <div style={{ padding: '20px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>新建用户</div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
        </div>

        {/* 表单 */}
        <div style={{ padding: '24px' }}>
          {error && (
            <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '16px', background: C.dangerLight, color: C.danger, fontSize: '13px' }}>
              {error}
            </div>
          )}

          {/* v122:学校管理员的温馨提示 */}
          {user?.role === 'senior_operator' && (
            <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '16px', background: C.primaryLight, color: C.primary, fontSize: '12px', lineHeight: 1.6 }}>
              💡 您是学校管理员,新建的用户将属于您管理的学校。可创建角色:骨干教师、普通教师
            </div>
          )}

          {/* 批C:admin 的任命制身份引导提示 */}
          {user?.role === 'admin' && (
            <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '16px', background: C.warningLight, color: C.warning, fontSize: '12px', lineHeight: 1.6 }}>
              💡 学校管理员 / 区域管理员为<b>任命制身份</b>,不在此创建:请先建教师账号(或使用已有账号),再到「组织架构」对应学校/区域卡片的「🛡️ 管理员」面板任命,系统将自动升级其身份。
            </div>
          )}

          {/* 防御提示:无可创建角色 */}
          {noCreatableRole && (
            <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '16px', background: C.warningLight, color: C.warning, fontSize: '12px', lineHeight: 1.6 }}>
              ⚠️ 当前角色无权创建用户。
            </div>
          )}

          {/* 文本字段 */}
          {fields.map(f => (
            <div key={f.key} style={{ marginBottom: '14px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>{f.label}</label>
              <input
                type={f.type}
                value={(form as Record<string, string>)[f.key]}
                onChange={e => setForm(p => ({ ...p, [f.key]: e.target.value }))}
                placeholder={f.placeholder}
                style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }}
                onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                onBlur={e => { e.currentTarget.style.borderColor = C.border }}
              />
            </div>
          ))}

          {/* 角色选择(v122:按登录者角色过滤;批C:任命制身份已剔除) */}
          <div style={{ marginBottom: '20px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>系统角色</label>
            <select
              value={form.role}
              onChange={e => setForm(p => ({ ...p, role: e.target.value }))}
              disabled={noCreatableRole}
              style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: C.white }}>
              {availableRoles.map(r => (
                <option key={r.value} value={r.value}>{r.label}</option>
              ))}
            </select>
          </div>

          {/* 操作按钮 */}
          <div style={{ display: 'flex', gap: '10px' }}>
            <button onClick={onClose} style={{ flex: 1, padding: '10px', borderRadius: '10px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>
              取消
            </button>
            <button
              onClick={handleCreate} disabled={saving || noCreatableRole}
              style={{
                flex: 2, padding: '10px', borderRadius: '10px', border: 'none',
                background: (saving || noCreatableRole) ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`,
                color: '#fff', fontSize: '14px', fontWeight: 600,
                cursor: (saving || noCreatableRole) ? 'not-allowed' : 'pointer',
              }}>
              {saving ? '创建中...' : '创建用户'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
