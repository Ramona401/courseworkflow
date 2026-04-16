/**
 * AccountPage — 通用用户中心页面
 *
 * 功能：
 * 1) 基本信息Tab（个人信息卡片 + 编辑显示名称）
 * 2) 安全设置Tab（修改密码）
 *
 * 学校管理入口已移至两个系统顶部Header下拉菜单
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import { getProfile, updateProfile, changePassword } from '@/api/account'
import type { ProfileInfo } from '@/api/account'

const COLORS = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  success:      '#10B981',
  danger:       '#EF4444',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  border:       '#E5E7EB',
  bg:           '#F9FAFB',
  white:        '#FFFFFF',
}

const ROLE_NAMES: Record<string, string> = {
  admin:           '系统管理员',
  senior_operator: '学校管理员',
  operator:        '骨干教师',
  viewer:          '普通教师',
}

const ROLE_COLORS: Record<string, { bg: string; color: string }> = {
  admin:           { bg: 'rgba(239,68,68,0.1)',   color: '#EF4444' },
  senior_operator: { bg: 'rgba(245,158,11,0.1)',  color: '#F59E0B' },
  operator:        { bg: 'rgba(79,123,232,0.1)',  color: '#4F7BE8' },
  viewer:          { bg: 'rgba(107,114,128,0.1)', color: '#6B7280' },
}

function Toast({ message, type, onClose }: { message: string; type: 'success' | 'error'; onClose: () => void }) {
  useEffect(() => { const t = setTimeout(onClose, 3000); return () => clearTimeout(t) }, [onClose])
  return (
    <div style={{ position: 'fixed', top: '24px', right: '24px', zIndex: 9999, padding: '12px 20px', borderRadius: '12px', color: '#fff', fontSize: '14px', fontWeight: 500, background: type === 'success' ? 'linear-gradient(135deg,#10B981,#059669)' : 'linear-gradient(135deg,#EF4444,#DC2626)', boxShadow: '0 4px 16px rgba(0,0,0,0.15)' }}>
      {type === 'success' ? '✓ ' : '✕ '}{message}
    </div>
  )
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', padding: '14px 0', borderBottom: `1px solid ${COLORS.border}` }}>
      <span style={{ width: '120px', flexShrink: 0, fontSize: '14px', color: COLORS.textSec }}>{label}</span>
      <span style={{ flex: 1, fontSize: '15px', color: COLORS.text, fontWeight: 500 }}>{value}</span>
    </div>
  )
}

type TabKey = 'profile' | 'security'

export default function AccountPage() {
  const navigate  = useNavigate()
  const location  = useLocation()
  const { user, login } = useAuth()
  const fromPath: string = (location.state as { from?: string })?.from || '/'

  const [profile, setProfile]     = useState<ProfileInfo | null>(null)
  const [loading, setLoading]     = useState(true)
  const [activeTab, setActiveTab] = useState<TabKey>('profile')

  const [editingName, setEditingName] = useState(false)
  const [newName, setNewName]         = useState('')
  const [nameSaving, setNameSaving]   = useState(false)

  const [showPwdForm, setShowPwdForm] = useState(false)
  const [pwdForm, setPwdForm]         = useState({ old_password: '', new_password: '', confirm: '' })
  const [pwdSaving, setPwdSaving]     = useState(false)
  const [showOldPwd, setShowOldPwd]   = useState(false)
  const [showNewPwd, setShowNewPwd]   = useState(false)

  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)
  const showToast = (msg: string, type: 'success' | 'error') => setToast({ message: msg, type })

  const loadProfile = useCallback(async () => {
    try {
      setLoading(true)
      const data = await getProfile()
      setProfile(data)
      setNewName(data.display_name)
    } catch { showToast('获取个人信息失败', 'error') }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { loadProfile() }, [loadProfile])

  const handleSaveName = async () => {
    if (!newName.trim()) { showToast('显示名称不能为空', 'error'); return }
    try {
      setNameSaving(true)
      const result = await updateProfile({ display_name: newName.trim() })
      if (user) login(localStorage.getItem('token') || '', { ...user, display_name: result.display_name })
      setProfile(prev => prev ? { ...prev, display_name: result.display_name } : prev)
      setEditingName(false)
      showToast('显示名称更新成功', 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '更新失败', 'error')
    } finally { setNameSaving(false) }
  }

  const handleChangePassword = async () => {
    if (!pwdForm.old_password) { showToast('请输入旧密码', 'error'); return }
    if (pwdForm.new_password.length < 6) { showToast('新密码不能少于6位', 'error'); return }
    if (pwdForm.new_password !== pwdForm.confirm) { showToast('两次密码不一致', 'error'); return }
    try {
      setPwdSaving(true)
      await changePassword({ old_password: pwdForm.old_password, new_password: pwdForm.new_password })
      showToast('密码修改成功，请重新登录', 'success')
      setPwdForm({ old_password: '', new_password: '', confirm: '' })
      setShowPwdForm(false)
      setTimeout(() => navigate('/login', { replace: true }), 2500)
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '密码修改失败', 'error')
    } finally { setPwdSaving(false) }
  }

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', background: COLORS.bg, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div style={{ textAlign: 'center' }}>
          <div style={{ width: '36px', height: '36px', margin: '0 auto 16px', border: `3px solid ${COLORS.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
          <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
          <div style={{ color: COLORS.textMuted, fontSize: '14px' }}>加载个人信息...</div>
        </div>
      </div>
    )
  }

  const roleStyle = ROLE_COLORS[profile?.role || 'viewer'] || ROLE_COLORS.viewer

  return (
    <div style={{ minHeight: '100vh', background: 'linear-gradient(135deg,#F0F4FF 0%,#FAFBFC 50%,#F0FDF4 100%)' }}>
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      <header style={{ height: '64px', background: 'rgba(255,255,255,0.85)', backdropFilter: 'blur(20px)', borderBottom: `1px solid ${COLORS.border}`, display: 'flex', alignItems: 'center', padding: '0 32px', position: 'sticky', top: 0, zIndex: 100 }}>
        <button onClick={() => navigate(fromPath)}
          style={{ display: 'flex', alignItems: 'center', gap: '6px', padding: '8px 16px', borderRadius: '8px', border: `1px solid ${COLORS.border}`, background: COLORS.white, cursor: 'pointer', fontSize: '14px', color: COLORS.textSec }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = COLORS.bg }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = COLORS.white }}>
          ← 返回
        </button>
        <h1 style={{ flex: 1, textAlign: 'center', fontSize: '18px', fontWeight: 600, color: COLORS.text, margin: 0 }}>个人中心</h1>
        <div style={{ width: '80px' }} />
      </header>

      <main style={{ maxWidth: '720px', margin: '0 auto', padding: '32px 24px 80px' }}>
        {/* Tab条 */}
        <div style={{ background: COLORS.white, border: `1px solid ${COLORS.border}`, borderRadius: '14px', padding: '8px', marginBottom: '16px', display: 'flex', gap: '8px' }}>
          {(['profile', 'security'] as const).map((tab, i) => (
            <button key={tab} onClick={() => setActiveTab(tab)} style={{ border: 'none', borderRadius: '10px', padding: '9px 20px', cursor: 'pointer', fontSize: '14px', fontWeight: activeTab === tab ? 600 : 400, color: activeTab === tab ? COLORS.primary : COLORS.textSec, background: activeTab === tab ? COLORS.primaryLight : 'transparent', transition: 'all 150ms ease' }}>
              {['基本信息', '安全设置'][i]}
            </button>
          ))}
        </div>

        {/* 基本信息 Tab */}
        {activeTab === 'profile' && (
          <>
            <div style={{ background: COLORS.white, borderRadius: '20px', border: `1px solid ${COLORS.border}`, padding: '32px', marginBottom: '20px', boxShadow: '0 2px 12px rgba(0,0,0,0.06)', display: 'flex', alignItems: 'center', gap: '24px' }}>
              <div style={{ width: '80px', height: '80px', flexShrink: 0, background: 'linear-gradient(135deg,#4F7BE8,#7C3AED)', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 4px 16px rgba(79,123,232,0.3)' }}>
                <span style={{ color: '#fff', fontSize: '28px', fontWeight: 700 }}>{profile?.display_name?.charAt(0)?.toUpperCase() || 'U'}</span>
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: '22px', fontWeight: 700, color: COLORS.text, marginBottom: '6px' }}>{profile?.display_name}</div>
                <div style={{ fontSize: '14px', color: COLORS.textSec, marginBottom: '10px' }}>@{profile?.username}</div>
                <span style={{ display: 'inline-block', padding: '4px 12px', borderRadius: '20px', fontSize: '13px', fontWeight: 600, background: roleStyle.bg, color: roleStyle.color }}>
                  {ROLE_NAMES[profile?.role || 'viewer'] || profile?.role}
                </span>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '16px 20px', borderRadius: '12px', background: COLORS.bg, border: `1px solid ${COLORS.border}`, minWidth: '100px' }}>
                <div style={{ fontSize: '28px', fontWeight: 700, color: COLORS.primary, lineHeight: 1 }}>{profile?.login_count ?? 0}</div>
                <div style={{ fontSize: '12px', color: COLORS.textMuted, marginTop: '4px' }}>累计登录</div>
              </div>
            </div>

            <div style={{ background: COLORS.white, borderRadius: '16px', border: `1px solid ${COLORS.border}`, padding: '24px 28px', marginBottom: '20px', boxShadow: '0 1px 4px rgba(0,0,0,0.04)' }}>
              <h2 style={{ fontSize: '16px', fontWeight: 600, color: COLORS.text, margin: '0 0 4px 0' }}>基本信息</h2>
              <InfoRow label="登录用户名" value={profile?.username || '-'} />
              <InfoRow label="显示名称" value={
                editingName ? (
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <input value={newName} onChange={e => setNewName(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') handleSaveName() }} autoFocus style={{ flex: 1, padding: '7px 12px', borderRadius: '8px', border: `1.5px solid ${COLORS.primary}`, fontSize: '15px', outline: 'none' }} />
                    <button onClick={handleSaveName} disabled={nameSaving} style={{ padding: '7px 16px', borderRadius: '8px', border: 'none', background: COLORS.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer', opacity: nameSaving ? 0.6 : 1 }}>{nameSaving ? '保存...' : '保存'}</button>
                    <button onClick={() => { setEditingName(false); setNewName(profile?.display_name || '') }} style={{ padding: '7px 12px', borderRadius: '8px', border: `1px solid ${COLORS.border}`, background: COLORS.white, color: COLORS.textSec, fontSize: '13px', cursor: 'pointer' }}>取消</button>
                  </div>
                ) : (
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <span>{profile?.display_name}</span>
                    <button onClick={() => setEditingName(true)} style={{ padding: '4px 12px', borderRadius: '6px', border: `1px solid ${COLORS.border}`, background: COLORS.bg, color: COLORS.primary, fontSize: '12px', fontWeight: 500, cursor: 'pointer' }}>修改</button>
                  </div>
                )
              } />
              <InfoRow label="系统角色" value={<span style={{ padding: '3px 10px', borderRadius: '12px', fontSize: '13px', fontWeight: 600, background: roleStyle.bg, color: roleStyle.color }}>{ROLE_NAMES[profile?.role || 'viewer'] || profile?.role}</span>} />
              <InfoRow label="账户状态" value={<span style={{ padding: '3px 10px', borderRadius: '12px', fontSize: '13px', fontWeight: 600, background: profile?.status === 'active' ? 'rgba(16,185,129,0.1)' : 'rgba(239,68,68,0.1)', color: profile?.status === 'active' ? COLORS.success : COLORS.danger }}>{profile?.status === 'active' ? '正常' : '已禁用'}</span>} />
              <InfoRow label="最近登录" value={profile?.last_login_at || '暂无记录'} />
              <div style={{ padding: '14px 0' }}>
                <div style={{ display: 'flex', alignItems: 'center' }}>
                  <span style={{ width: '120px', flexShrink: 0, fontSize: '14px', color: COLORS.textSec }}>注册时间</span>
                  <span style={{ flex: 1, fontSize: '15px', color: COLORS.text, fontWeight: 500 }}>{profile?.created_at || '-'}</span>
                </div>
              </div>
            </div>
          </>
        )}

        {/* 安全设置 Tab */}
        {activeTab === 'security' && (
          <div style={{ background: COLORS.white, borderRadius: '16px', border: `1px solid ${COLORS.border}`, padding: '24px 28px', boxShadow: '0 1px 4px rgba(0,0,0,0.04)' }}>
            <h2 style={{ fontSize: '16px', fontWeight: 600, color: COLORS.text, margin: '0 0 20px 0' }}>安全设置</h2>
            {!showPwdForm ? (
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '16px', borderRadius: '12px', background: COLORS.bg, border: `1px solid ${COLORS.border}` }}>
                <div>
                  <div style={{ fontSize: '15px', fontWeight: 500, color: COLORS.text }}>登录密码</div>
                  <div style={{ fontSize: '13px', color: COLORS.textMuted, marginTop: '2px' }}>建议定期修改密码以保证账户安全</div>
                </div>
                <button onClick={() => setShowPwdForm(true)} style={{ padding: '8px 20px', borderRadius: '8px', border: `1px solid ${COLORS.primary}`, background: COLORS.primaryLight, color: COLORS.primary, fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>修改密码</button>
              </div>
            ) : (
              <div style={{ padding: '20px', borderRadius: '12px', background: COLORS.bg, border: `1.5px solid ${COLORS.primary}` }}>
                <div style={{ marginBottom: '16px', fontSize: '15px', fontWeight: 600, color: COLORS.text }}>修改密码</div>
                <div style={{ marginBottom: '14px' }}>
                  <label style={{ display: 'block', fontSize: '13px', color: COLORS.textSec, marginBottom: '6px' }}>旧密码</label>
                  <div style={{ position: 'relative' }}>
                    <input type={showOldPwd ? 'text' : 'password'} value={pwdForm.old_password} onChange={e => setPwdForm(p => ({ ...p, old_password: e.target.value }))} placeholder="请输入旧密码" style={{ width: '100%', padding: '10px 40px 10px 14px', borderRadius: '8px', border: `1px solid ${COLORS.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: COLORS.white }} />
                    <button onClick={() => setShowOldPwd(p => !p)} style={{ position: 'absolute', right: '12px', top: '50%', transform: 'translateY(-50%)', background: 'none', border: 'none', cursor: 'pointer', color: COLORS.textMuted, fontSize: '16px' }}>{showOldPwd ? '🙈' : '👁'}</button>
                  </div>
                </div>
                <div style={{ marginBottom: '14px' }}>
                  <label style={{ display: 'block', fontSize: '13px', color: COLORS.textSec, marginBottom: '6px' }}>新密码（至少6位）</label>
                  <div style={{ position: 'relative' }}>
                    <input type={showNewPwd ? 'text' : 'password'} value={pwdForm.new_password} onChange={e => setPwdForm(p => ({ ...p, new_password: e.target.value }))} placeholder="请输入新密码" style={{ width: '100%', padding: '10px 40px 10px 14px', borderRadius: '8px', border: `1px solid ${COLORS.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: COLORS.white }} />
                    <button onClick={() => setShowNewPwd(p => !p)} style={{ position: 'absolute', right: '12px', top: '50%', transform: 'translateY(-50%)', background: 'none', border: 'none', cursor: 'pointer', color: COLORS.textMuted, fontSize: '16px' }}>{showNewPwd ? '🙈' : '👁'}</button>
                  </div>
                </div>
                <div style={{ marginBottom: '20px' }}>
                  <label style={{ display: 'block', fontSize: '13px', color: COLORS.textSec, marginBottom: '6px' }}>确认新密码</label>
                  <input type="password" value={pwdForm.confirm} onChange={e => setPwdForm(p => ({ ...p, confirm: e.target.value }))} onKeyDown={e => { if (e.key === 'Enter') handleChangePassword() }} placeholder="再次输入新密码" style={{ width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${pwdForm.confirm && pwdForm.confirm !== pwdForm.new_password ? COLORS.danger : COLORS.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: COLORS.white }} />
                  {pwdForm.confirm && pwdForm.confirm !== pwdForm.new_password && (<div style={{ fontSize: '12px', color: COLORS.danger, marginTop: '4px' }}>两次密码不一致</div>)}
                </div>
                <div style={{ display: 'flex', gap: '10px' }}>
                  <button onClick={handleChangePassword} disabled={pwdSaving} style={{ flex: 1, padding: '10px', borderRadius: '8px', border: 'none', background: pwdSaving ? COLORS.textMuted : 'linear-gradient(135deg,#4F7BE8,#7C3AED)', color: '#fff', fontSize: '14px', fontWeight: 600, cursor: pwdSaving ? 'not-allowed' : 'pointer' }}>{pwdSaving ? '修改中...' : '确认修改'}</button>
                  <button onClick={() => { setShowPwdForm(false); setPwdForm({ old_password: '', new_password: '', confirm: '' }) }} style={{ padding: '10px 20px', borderRadius: '8px', border: `1px solid ${COLORS.border}`, background: COLORS.white, color: COLORS.textSec, fontSize: '14px', cursor: 'pointer' }}>取消</button>
                </div>
              </div>
            )}
          </div>
        )}
      </main>
    </div>
  )
}
