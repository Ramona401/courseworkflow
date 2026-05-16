/**
 * CreateUserModal.tsx — 新建用户弹窗
 * 字段:登录用户名 / 显示名称 / 初始密码 / 系统角色
 *
 * v122 改动(AdminPage 权限统一):
 *   - 角色下拉按登录者角色过滤:
 *     * admin 可创建任意角色
 *     * senior_operator 只能创建 operator / viewer(不能创建 admin 或其他学校管理员)
 *   - 默认选中 operator(最常用的骨干教师角色)
 */
import { useState, useMemo } from 'react'
import { createAdminUser } from '@/api/admin'
import { useAuth } from '@/store/auth'
import { C, ROLE_OPTIONS } from './adminConstants'

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
  // admin 可创建所有角色,senior_operator 只能创建 operator/viewer
  const availableRoles = useMemo(() => {
    const allRoles = ROLE_OPTIONS.filter(r => r.value) // 去掉"全部角色"选项
    if (user?.role === 'admin') {
      return allRoles
    }
    if (user?.role === 'senior_operator') {
      // 学校管理员只能创建低于自己级别的角色
      return allRoles.filter(r => r.value === 'operator' || r.value === 'viewer')
    }
    // 其他角色不应该走到这里(路由层已拦截),保险起见返回空
    return []
  }, [user?.role])

  const handleCreate = async () => {
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

          {/* 角色选择(v122:按登录者角色过滤) */}
          <div style={{ marginBottom: '20px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>系统角色</label>
            <select
              value={form.role}
              onChange={e => setForm(p => ({ ...p, role: e.target.value }))}
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
              onClick={handleCreate} disabled={saving}
              style={{
                flex: 2, padding: '10px', borderRadius: '10px', border: 'none',
                background: saving ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`,
                color: '#fff', fontSize: '14px', fontWeight: 600,
                cursor: saving ? 'not-allowed' : 'pointer',
              }}>
              {saving ? '创建中...' : '创建用户'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
