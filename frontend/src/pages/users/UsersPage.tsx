/**
 * 用户管理页面（P1-4）- Apple 风格
 * - 用户列表（表格展示）
 * - 创建用户弹窗
 * - 编辑用户弹窗
 * - 重置密码弹窗
 * - 启用/禁用状态切换
 * - 仅admin可访问
 */
import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '@/store/auth'
import type { UserInfo } from '@/api/auth'
import {
  getUsers,
  createUser,
  updateUser,
  resetPassword,
  updateUserStatus,
} from '@/api/users'
import type { CreateUserRequest, UpdateUserRequest } from '@/api/users'
import { Plus, Edit3, Key, UserCheck, UserX, X, AlertCircle, CheckCircle } from 'lucide-react'

// ==================== 角色/状态映射 ====================

const roleMap: Record<string, string> = {
  admin: '管理员',
  operator: '操作员',
  viewer: '查看者',
}

const roleColors: Record<string, { bg: string; color: string }> = {
  admin: { bg: 'rgba(255,59,48,0.1)', color: '#ff3b30' },
  operator: { bg: 'rgba(0,122,255,0.1)', color: '#007aff' },
  viewer: { bg: 'rgba(142,142,147,0.1)', color: '#8e8e93' },
}

const statusMap: Record<string, string> = {
  active: '正常',
  disabled: '已禁用',
}

// ==================== 通用样式 ====================

const cardStyle: React.CSSProperties = {
  background: '#fff',
  borderRadius: '16px',
  boxShadow: '0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04)',
  border: '1px solid rgba(0,0,0,0.04)',
}

const btnPrimary: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: '6px',
  padding: '8px 18px',
  background: 'linear-gradient(135deg, #007aff, #5856d6)',
  color: '#fff',
  border: 'none',
  borderRadius: '10px',
  fontSize: '14px',
  fontWeight: 500,
  cursor: 'pointer',
  transition: 'all 0.2s ease',
}

const btnSecondary: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: '4px',
  padding: '6px 12px',
  background: 'rgba(0,122,255,0.08)',
  color: '#007aff',
  border: 'none',
  borderRadius: '8px',
  fontSize: '13px',
  fontWeight: 500,
  cursor: 'pointer',
  transition: 'all 0.15s ease',
}

const btnDanger: React.CSSProperties = {
  ...btnSecondary,
  background: 'rgba(255,59,48,0.08)',
  color: '#ff3b30',
}

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '10px 14px',
  border: '1px solid #d1d1d6',
  borderRadius: '10px',
  fontSize: '14px',
  outline: 'none',
  transition: 'border-color 0.2s ease, box-shadow 0.2s ease',
  boxSizing: 'border-box',
}

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: '13px',
  fontWeight: 500,
  color: '#1d1d1f',
  marginBottom: '6px',
}

const selectStyle: React.CSSProperties = {
  ...inputStyle,
  appearance: 'none',
  backgroundImage: 'url("data:image/svg+xml,%3Csvg xmlns=\'http://www.w3.org/2000/svg\' width=\'12\' height=\'12\' viewBox=\'0 0 12 12\'%3E%3Cpath fill=\'%238e8e93\' d=\'M6 8L1 3h10z\'/%3E%3C/svg%3E")',
  backgroundRepeat: 'no-repeat',
  backgroundPosition: 'right 12px center',
  paddingRight: '32px',
}

// ==================== 弹窗遮罩样式 ====================

const overlayStyle: React.CSSProperties = {
  position: 'fixed',
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  background: 'rgba(0,0,0,0.4)',
  backdropFilter: 'blur(8px)',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  zIndex: 1000,
}

const modalStyle: React.CSSProperties = {
  background: '#fff',
  borderRadius: '20px',
  padding: '28px',
  width: '440px',
  maxWidth: '90vw',
  boxShadow: '0 20px 60px rgba(0,0,0,0.15)',
}

// ==================== Toast 提示组件 ====================

interface ToastProps {
  message: string
  type: 'success' | 'error'
  onClose: () => void
}

function Toast({ message, type, onClose }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(onClose, 3000)
    return () => clearTimeout(timer)
  }, [onClose])

  return (
    <div style={{
      position: 'fixed',
      top: '20px',
      right: '20px',
      zIndex: 2000,
      display: 'flex',
      alignItems: 'center',
      gap: '10px',
      padding: '14px 20px',
      borderRadius: '14px',
      background: type === 'success' ? '#f0fdf4' : '#fef2f2',
      border: `1px solid ${type === 'success' ? '#bbf7d0' : '#fecaca'}`,
      boxShadow: '0 8px 30px rgba(0,0,0,0.12)',
      fontSize: '14px',
      color: type === 'success' ? '#166534' : '#991b1b',
      animation: 'slideIn 0.3s ease',
    }}>
      {type === 'success' ? <CheckCircle size={18} /> : <AlertCircle size={18} />}
      <span>{message}</span>
    </div>
  )
}

// ==================== 主组件 ====================

export default function UsersPage() {
  const { user: currentUser } = useAuth()

  // 用户列表状态
  const [users, setUsers] = useState<UserInfo[]>([])
  const [loading, setLoading] = useState(true)

  // 弹窗状态
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [showPasswordModal, setShowPasswordModal] = useState(false)
  const [editingUser, setEditingUser] = useState<UserInfo | null>(null)

  // 表单状态
  const [createForm, setCreateForm] = useState<CreateUserRequest>({
    username: '', display_name: '', password: '', role: 'operator',
  })
  const [editForm, setEditForm] = useState<UpdateUserRequest>({
    display_name: '', role: 'operator',
  })
  const [newPassword, setNewPassword] = useState('')

  // 操作状态
  const [submitting, setSubmitting] = useState(false)
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)

  // ==================== 数据加载 ====================

  const loadUsers = useCallback(async () => {
    try {
      setLoading(true)
      const result = await getUsers()
      setUsers(result.users || [])
    } catch (err: any) {
      showToast(err.message || '加载用户列表失败', 'error')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadUsers()
  }, [loadUsers])

  // ==================== Toast ====================

  const showToast = (message: string, type: 'success' | 'error') => {
    setToast({ message, type })
  }

  // ==================== 创建用户 ====================

  const handleCreate = async () => {
    if (!createForm.username.trim()) { showToast('请输入用户名', 'error'); return }
    if (!createForm.display_name.trim()) { showToast('请输入显示名称', 'error'); return }
    if (createForm.password.length < 6) { showToast('密码长度不能少于6位', 'error'); return }

    try {
      setSubmitting(true)
      await createUser(createForm)
      showToast('用户创建成功', 'success')
      setShowCreateModal(false)
      setCreateForm({ username: '', display_name: '', password: '', role: 'operator' })
      loadUsers()
    } catch (err: any) {
      showToast(err.message || '创建失败', 'error')
    } finally {
      setSubmitting(false)
    }
  }

  // ==================== 编辑用户 ====================

  const openEditModal = (u: UserInfo) => {
    setEditingUser(u)
    setEditForm({ display_name: u.display_name, role: u.role })
    setShowEditModal(true)
  }

  const handleEdit = async () => {
    if (!editingUser) return
    if (!editForm.display_name.trim()) { showToast('请输入显示名称', 'error'); return }

    try {
      setSubmitting(true)
      await updateUser(editingUser.id, editForm)
      showToast('用户信息更新成功', 'success')
      setShowEditModal(false)
      setEditingUser(null)
      loadUsers()
    } catch (err: any) {
      showToast(err.message || '更新失败', 'error')
    } finally {
      setSubmitting(false)
    }
  }

  // ==================== 重置密码 ====================

  const openPasswordModal = (u: UserInfo) => {
    setEditingUser(u)
    setNewPassword('')
    setShowPasswordModal(true)
  }

  const handleResetPassword = async () => {
    if (!editingUser) return
    if (newPassword.length < 6) { showToast('密码长度不能少于6位', 'error'); return }

    try {
      setSubmitting(true)
      await resetPassword(editingUser.id, { new_password: newPassword })
      showToast(`已重置 ${editingUser.display_name} 的密码`, 'success')
      setShowPasswordModal(false)
      setEditingUser(null)
    } catch (err: any) {
      showToast(err.message || '重置失败', 'error')
    } finally {
      setSubmitting(false)
    }
  }

  // ==================== 状态切换 ====================

  const handleToggleStatus = async (u: UserInfo) => {
    const newStatus = u.status === 'active' ? 'disabled' : 'active'
    const actionText = newStatus === 'active' ? '启用' : '禁用'

    if (u.id === currentUser?.id) {
      showToast('不能禁用自己的账户', 'error')
      return
    }

    if (!window.confirm(`确定要${actionText}用户「${u.display_name}」吗？`)) return

    try {
      await updateUserStatus(u.id, { status: newStatus })
      showToast(`已${actionText}用户 ${u.display_name}`, 'success')
      loadUsers()
    } catch (err: any) {
      showToast(err.message || '操作失败', 'error')
    }
  }

  // ==================== 输入框聚焦效果 ====================

  const handleFocus = (e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) => {
    e.target.style.borderColor = '#007aff'
    e.target.style.boxShadow = '0 0 0 3px rgba(0,122,255,0.15)'
  }

  const handleBlur = (e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) => {
    e.target.style.borderColor = '#d1d1d6'
    e.target.style.boxShadow = 'none'
  }

  // ==================== 时间格式化 ====================

  const formatTime = (t: string | null) => {
    if (!t) return '从未登录'
    const d = new Date(t)
    return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  }

  // ==================== 渲染 ====================

  return (
    <div style={{ padding: '0' }}>
      {/* Toast 提示 */}
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      {/* 页面头部（标题由MainLayout header提供） */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <p style={{ fontSize: '14px', color: '#8e8e93', margin: 0 }}>
          管理系统用户账户、角色和课程分配
        </p>
        <button style={btnPrimary} onClick={() => setShowCreateModal(true)}
          onMouseEnter={e => { e.currentTarget.style.transform = 'translateY(-1px)'; e.currentTarget.style.boxShadow = '0 4px 12px rgba(0,122,255,0.3)' }}
          onMouseLeave={e => { e.currentTarget.style.transform = 'none'; e.currentTarget.style.boxShadow = 'none' }}>
          <Plus size={16} />
          <span>新建用户</span>
        </button>
      </div>

      {/* 统计卡片 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px', marginBottom: '24px' }}>
        {[
          { label: '用户总数', value: users.length, color: '#007aff' },
          { label: '管理员', value: users.filter(u => u.role === 'admin').length, color: '#ff3b30' },
          { label: '操作员', value: users.filter(u => u.role === 'operator').length, color: '#5856d6' },
          { label: '已禁用', value: users.filter(u => u.status === 'disabled').length, color: '#8e8e93' },
        ].map((s, i) => (
          <div key={i} style={{ ...cardStyle, padding: '20px' }}>
            <div style={{ fontSize: '13px', color: '#8e8e93', marginBottom: '8px' }}>{s.label}</div>
            <div style={{ fontSize: '28px', fontWeight: 700, color: s.color, letterSpacing: '-1px' }}>{s.value}</div>
          </div>
        ))}
      </div>

      {/* 用户表格 */}
      <div style={{ ...cardStyle, overflow: 'hidden' }}>
        {loading ? (
          <div style={{ padding: '60px', textAlign: 'center' }}>
            <div style={{
              width: '32px', height: '32px',
              border: '2px solid #007aff', borderTopColor: 'transparent',
              borderRadius: '50%', animation: 'spin 0.8s linear infinite',
              margin: '0 auto 12px',
            }} />
            <div style={{ color: '#8e8e93', fontSize: '14px' }}>加载中...</div>
          </div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr style={{ borderBottom: '1px solid #f2f2f7' }}>
                {['用户名', '显示名称', '角色', '状态', '最后登录', '登录次数', '操作'].map(h => (
                  <th key={h} style={{
                    padding: '14px 16px',
                    fontSize: '12px',
                    fontWeight: 600,
                    color: '#8e8e93',
                    textAlign: 'left',
                    textTransform: 'uppercase',
                    letterSpacing: '0.5px',
                  }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {users.map(u => (
                <tr key={u.id} style={{ borderBottom: '1px solid #f2f2f7', transition: 'background 0.15s ease' }}
                  onMouseEnter={e => e.currentTarget.style.background = '#fafafa'}
                  onMouseLeave={e => e.currentTarget.style.background = 'transparent'}>
                  <td style={{ padding: '14px 16px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                      <div style={{
                        width: '32px', height: '32px',
                        background: `linear-gradient(135deg, ${roleColors[u.role]?.color || '#8e8e93'}22, ${roleColors[u.role]?.color || '#8e8e93'}44)`,
                        borderRadius: '50%',
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        fontSize: '13px', fontWeight: 600, color: roleColors[u.role]?.color || '#8e8e93',
                      }}>
                        {u.display_name.charAt(0)}
                      </div>
                      <span style={{ fontSize: '14px', fontWeight: 500, color: '#1d1d1f' }}>{u.username}</span>
                    </div>
                  </td>
                  <td style={{ padding: '14px 16px', fontSize: '14px', color: '#3a3a3c' }}>{u.display_name}</td>
                  <td style={{ padding: '14px 16px' }}>
                    <span style={{
                      display: 'inline-block',
                      padding: '3px 10px',
                      borderRadius: '6px',
                      fontSize: '12px',
                      fontWeight: 500,
                      background: roleColors[u.role]?.bg || '#f2f2f7',
                      color: roleColors[u.role]?.color || '#8e8e93',
                    }}>{roleMap[u.role] || u.role}</span>
                  </td>
                  <td style={{ padding: '14px 16px' }}>
                    <span style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: '4px',
                      fontSize: '13px',
                      color: u.status === 'active' ? '#34c759' : '#ff3b30',
                    }}>
                      <span style={{
                        width: '6px', height: '6px', borderRadius: '50%',
                        background: u.status === 'active' ? '#34c759' : '#ff3b30',
                      }} />
                      {statusMap[u.status] || u.status}
                    </span>
                  </td>
                  <td style={{ padding: '14px 16px', fontSize: '13px', color: '#8e8e93' }}>
                    {formatTime(u.last_login_at)}
                  </td>
                  <td style={{ padding: '14px 16px', fontSize: '14px', color: '#3a3a3c', textAlign: 'center' }}>
                    {u.login_count}
                  </td>
                  <td style={{ padding: '14px 16px' }}>
                    <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
                      <button style={btnSecondary} onClick={() => openEditModal(u)} title="编辑">
                        <Edit3 size={13} /> 编辑
                      </button>
                      <button style={btnSecondary} onClick={() => openPasswordModal(u)} title="重置密码">
                        <Key size={13} /> 密码
                      </button>
                      {u.id !== currentUser?.id && (
                        <button
                          style={u.status === 'active' ? btnDanger : { ...btnSecondary, background: 'rgba(52,199,89,0.08)', color: '#34c759' }}
                          onClick={() => handleToggleStatus(u)}
                          title={u.status === 'active' ? '禁用' : '启用'}>
                          {u.status === 'active'
                            ? <><UserX size={13} /> 禁用</>
                            : <><UserCheck size={13} /> 启用</>}
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
              {users.length === 0 && (
                <tr>
                  <td colSpan={7} style={{ padding: '60px', textAlign: 'center', color: '#8e8e93', fontSize: '14px' }}>
                    暂无用户数据
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>

      {/* ==================== 创建用户弹窗 ==================== */}
      {showCreateModal && (
        <div style={overlayStyle} onClick={() => setShowCreateModal(false)}>
          <div style={modalStyle} onClick={e => e.stopPropagation()}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
              <h3 style={{ fontSize: '18px', fontWeight: 700, color: '#1d1d1f', margin: 0 }}>新建用户</h3>
              <button onClick={() => setShowCreateModal(false)}
                style={{ background: '#f2f2f7', border: 'none', borderRadius: '50%', width: '30px', height: '30px',
                  display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}>
                <X size={16} color="#8e8e93" />
              </button>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <div>
                <label style={labelStyle}>用户名</label>
                <input style={inputStyle} placeholder="请输入用户名"
                  value={createForm.username}
                  onChange={e => setCreateForm({ ...createForm, username: e.target.value })}
                  onFocus={handleFocus} onBlur={handleBlur} />
              </div>
              <div>
                <label style={labelStyle}>显示名称</label>
                <input style={inputStyle} placeholder="请输入显示名称"
                  value={createForm.display_name}
                  onChange={e => setCreateForm({ ...createForm, display_name: e.target.value })}
                  onFocus={handleFocus} onBlur={handleBlur} />
              </div>
              <div>
                <label style={labelStyle}>初始密码</label>
                <input style={inputStyle} type="password" placeholder="至少6位"
                  value={createForm.password}
                  onChange={e => setCreateForm({ ...createForm, password: e.target.value })}
                  onFocus={handleFocus} onBlur={handleBlur} />
              </div>
              <div>
                <label style={labelStyle}>角色</label>
                <select style={selectStyle} value={createForm.role}
                  onChange={e => setCreateForm({ ...createForm, role: e.target.value as any })}
                  onFocus={handleFocus} onBlur={handleBlur}>
                  <option value="operator">操作员</option>
                  <option value="viewer">查看者</option>
                  <option value="admin">管理员</option>
                </select>
              </div>
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '28px' }}>
              <button onClick={() => setShowCreateModal(false)}
                style={{ padding: '9px 20px', background: '#f2f2f7', color: '#3a3a3c', border: 'none',
                  borderRadius: '10px', fontSize: '14px', fontWeight: 500, cursor: 'pointer' }}>
                取消
              </button>
              <button onClick={handleCreate} disabled={submitting}
                style={{ ...btnPrimary, opacity: submitting ? 0.6 : 1, padding: '9px 24px' }}>
                {submitting ? '创建中...' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ==================== 编辑用户弹窗 ==================== */}
      {showEditModal && editingUser && (
        <div style={overlayStyle} onClick={() => setShowEditModal(false)}>
          <div style={modalStyle} onClick={e => e.stopPropagation()}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
              <h3 style={{ fontSize: '18px', fontWeight: 700, color: '#1d1d1f', margin: 0 }}>
                编辑用户 - {editingUser.username}
              </h3>
              <button onClick={() => setShowEditModal(false)}
                style={{ background: '#f2f2f7', border: 'none', borderRadius: '50%', width: '30px', height: '30px',
                  display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}>
                <X size={16} color="#8e8e93" />
              </button>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <div>
                <label style={labelStyle}>显示名称</label>
                <input style={inputStyle} placeholder="请输入显示名称"
                  value={editForm.display_name}
                  onChange={e => setEditForm({ ...editForm, display_name: e.target.value })}
                  onFocus={handleFocus} onBlur={handleBlur} />
              </div>
              <div>
                <label style={labelStyle}>角色</label>
                <select style={selectStyle} value={editForm.role}
                  onChange={e => setEditForm({ ...editForm, role: e.target.value as any })}
                  onFocus={handleFocus} onBlur={handleBlur}
                  disabled={editingUser.id === currentUser?.id}>
                  <option value="operator">操作员</option>
                  <option value="viewer">查看者</option>
                  <option value="admin">管理员</option>
                </select>
                {editingUser.id === currentUser?.id && (
                  <div style={{ fontSize: '12px', color: '#8e8e93', marginTop: '4px' }}>不能修改自己的角色</div>
                )}
              </div>
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '28px' }}>
              <button onClick={() => setShowEditModal(false)}
                style={{ padding: '9px 20px', background: '#f2f2f7', color: '#3a3a3c', border: 'none',
                  borderRadius: '10px', fontSize: '14px', fontWeight: 500, cursor: 'pointer' }}>
                取消
              </button>
              <button onClick={handleEdit} disabled={submitting}
                style={{ ...btnPrimary, opacity: submitting ? 0.6 : 1, padding: '9px 24px' }}>
                {submitting ? '保存中...' : '保存'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ==================== 重置密码弹窗 ==================== */}
      {showPasswordModal && editingUser && (
        <div style={overlayStyle} onClick={() => setShowPasswordModal(false)}>
          <div style={modalStyle} onClick={e => e.stopPropagation()}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
              <h3 style={{ fontSize: '18px', fontWeight: 700, color: '#1d1d1f', margin: 0 }}>
                重置密码 - {editingUser.display_name}
              </h3>
              <button onClick={() => setShowPasswordModal(false)}
                style={{ background: '#f2f2f7', border: 'none', borderRadius: '50%', width: '30px', height: '30px',
                  display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}>
                <X size={16} color="#8e8e93" />
              </button>
            </div>

            <div>
              <label style={labelStyle}>新密码</label>
              <input style={inputStyle} type="password" placeholder="至少6位"
                value={newPassword}
                onChange={e => setNewPassword(e.target.value)}
                onFocus={handleFocus} onBlur={handleBlur} />
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '28px' }}>
              <button onClick={() => setShowPasswordModal(false)}
                style={{ padding: '9px 20px', background: '#f2f2f7', color: '#3a3a3c', border: 'none',
                  borderRadius: '10px', fontSize: '14px', fontWeight: 500, cursor: 'pointer' }}>
                取消
              </button>
              <button onClick={handleResetPassword} disabled={submitting}
                style={{ ...btnPrimary, background: 'linear-gradient(135deg, #ff9500, #ff3b30)',
                  opacity: submitting ? 0.6 : 1, padding: '9px 24px' }}>
                {submitting ? '重置中...' : '确认重置'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 动画样式 */}
      <style>{`
        @keyframes slideIn {
          from { transform: translateX(30px); opacity: 0; }
          to { transform: translateX(0); opacity: 1; }
        }
      `}</style>
    </div>
  )
}
