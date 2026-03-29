/**
 * AdminPage — 统一用户管理中心
 *
 * 路由：/admin（独立页面，不在任何Layout内）
 * 权限：仅admin（路由层RoleGuard保护）
 * 入口：两个系统顶部Header下拉菜单 → "用户管理"
 *
 * 4个Tab：
 *   📊 概览    — 统计卡片（用户数/组织数/教研组数，按角色分布）
 *   👥 用户管理 — 用户列表+筛选+新建+编辑+禁用+重置密码+详情弹窗
 *   🏫 组织架构 — 区域/学校/教研组树形结构+成员管理
 *   📋 操作日志 — 分页日志+按用户/操作筛选
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  getAdminStats, getAdminUsers, getAdminUserDetail,
  createAdminUser, updateAdminUser, updateAdminUserStatus,
  resetAdminUserPassword, getAdminOrgs, getAdminGroups,
  getAdminAuditLogs,
} from '@/api/admin'
import type { OrgListItem, GroupListItem } from '@/api/admin'
import type {
  AdminStats, AdminUserListItem, AdminUserDetail,
  AuditLogItem,
} from '@/api/admin'

// ==================== 样式常量 ====================

const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981',
  successLight: 'rgba(16,185,129,0.08)',
  danger: '#EF4444',
  dangerLight: 'rgba(239,68,68,0.08)',
  warning: '#F59E0B',
  warningLight: 'rgba(245,158,11,0.08)',
  purple: '#7C3AED',
  purpleLight: 'rgba(124,58,237,0.08)',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  bg: '#F9FAFB',
  white: '#FFFFFF',
}

const ROLE_OPTIONS = [
  { value: '', label: '全部角色' },
  { value: 'admin', label: '管理员', color: C.danger },
  { value: 'senior_operator', label: '高级操作员', color: C.warning },
  { value: 'operator', label: '操作员', color: C.primary },
  { value: 'viewer', label: '查看者', color: C.textSec },
]

const ACTION_OPTIONS = [
  { value: '', label: '全部操作' },
  { value: 'user.login', label: '用户登录' },
  { value: 'user.logout', label: '用户登出' },
  { value: 'admin.user_create', label: '创建用户' },
  { value: 'admin.user_status', label: '状态变更' },
  { value: 'admin.user_reset_password', label: '重置密码' },
  { value: 'pipeline.confirm_finalize', label: '确认定稿' },
  { value: 'pipeline.verify', label: '触发验收' },
]

// ==================== Toast ====================

function Toast({ message, type, onClose }: {
  message: string; type: 'success' | 'error'; onClose: () => void
}) {
  useEffect(() => {
    const t = setTimeout(onClose, 3500)
    return () => clearTimeout(t)
  }, [onClose])
  return (
    <div style={{
      position: 'fixed', top: '24px', right: '24px', zIndex: 9999,
      padding: '12px 20px', borderRadius: '12px', color: '#fff',
      fontSize: '14px', fontWeight: 500,
      background: type === 'success'
        ? 'linear-gradient(135deg,#10B981,#059669)'
        : 'linear-gradient(135deg,#EF4444,#DC2626)',
      boxShadow: '0 4px 16px rgba(0,0,0,0.15)',
    }}>
      {type === 'success' ? '✓ ' : '✕ '}{message}
    </div>
  )
}

// ==================== 角色徽标 ====================

function RoleBadge({ role, roleName }: { role: string; roleName?: string }) {
  const styleMap: Record<string, { bg: string; color: string }> = {
    admin:           { bg: C.dangerLight,  color: C.danger },
    senior_operator: { bg: C.warningLight, color: C.warning },
    operator:        { bg: C.primaryLight, color: C.primary },
    viewer:          { bg: C.bg,           color: C.textSec },
  }
  const s = styleMap[role] || { bg: C.bg, color: C.textSec }
  const nameMap: Record<string, string> = {
    admin: '管理员', senior_operator: '高级操作员', operator: '操作员', viewer: '查看者',
  }
  return (
    <span style={{
      display: 'inline-block', padding: '2px 10px', borderRadius: '12px',
      fontSize: '12px', fontWeight: 600, background: s.bg, color: s.color,
    }}>
      {roleName || nameMap[role] || role}
    </span>
  )
}

// ==================== 状态徽标 ====================

function StatusBadge({ status }: { status: string }) {
  const active = status === 'active'
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: '4px',
      padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 600,
      background: active ? C.successLight : C.dangerLight,
      color: active ? C.success : C.danger,
    }}>
      <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: active ? C.success : C.danger }} />
      {active ? '正常' : '已禁用'}
    </span>
  )
}

// ==================== 统计卡片 ====================

function StatCard({ label, value, sub, color }: {
  label: string; value: number; sub?: string; color?: string
}) {
  return (
    <div style={{
      background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`,
      padding: '20px 24px', flex: 1,
      boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
    }}>
      <div style={{ fontSize: '13px', color: C.textSec, marginBottom: '8px' }}>{label}</div>
      <div style={{ fontSize: '28px', fontWeight: 700, color: color || C.text, lineHeight: 1 }}>{value}</div>
      {sub && <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '6px' }}>{sub}</div>}
    </div>
  )
}

// ==================== 用户详情弹窗 ====================

function UserDetailModal({ userId, onClose, onAction }: {
  userId: string
  onClose: () => void
  onAction: () => void
}) {
  const [detail, setDetail] = useState<AdminUserDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [resetPwd, setResetPwd] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    getAdminUserDetail(userId)
      .then(setDetail)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [userId])

  const handleReset = async () => {
    if (resetPwd.length < 6) return
    try {
      setSaving(true)
      await resetAdminUserPassword(userId, resetPwd)
      setResetPwd('')
      onAction()
    } catch {
    } finally {
      setSaving(false)
    }
  }

  const handleToggleStatus = async () => {
    if (!detail) return
    const newStatus = detail.status === 'active' ? 'disabled' : 'active'
    try {
      setSaving(true)
      await updateAdminUserStatus(userId, newStatus)
      setDetail(prev => prev ? { ...prev, status: newStatus } : prev)
      onAction()
    } catch {
    } finally {
      setSaving(false)
    }
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 10000,
      background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: '24px',
    }} onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{
        background: C.white, borderRadius: '20px', width: '620px', maxHeight: '85vh',
        overflow: 'hidden', display: 'flex', flexDirection: 'column',
        boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }}>
        {/* 弹窗头部 */}
        <div style={{
          padding: '20px 24px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        }}>
          <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>用户详情</div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
        </div>

        {/* 弹窗内容 */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '24px' }}>
          {loading ? (
            <div style={{ textAlign: 'center', padding: '40px', color: C.textMuted }}>加载中...</div>
          ) : detail ? (
            <>
              {/* 基础信息 */}
              <div style={{ display: 'flex', alignItems: 'center', gap: '16px', marginBottom: '24px' }}>
                <div style={{
                  width: '56px', height: '56px', borderRadius: '50%',
                  background: 'linear-gradient(135deg,#4F7BE8,#7C3AED)',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  fontSize: '22px', color: '#fff', fontWeight: 700, flexShrink: 0,
                }}>
                  {detail.display_name?.charAt(0)?.toUpperCase() || 'U'}
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: '18px', fontWeight: 700, color: C.text }}>{detail.display_name}</div>
                  <div style={{ fontSize: '13px', color: C.textSec, marginTop: '3px' }}>@{detail.username}</div>
                  <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
                    <RoleBadge role={detail.role} roleName={detail.role_name} />
                    <StatusBadge status={detail.status} />
                  </div>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <div style={{ fontSize: '24px', fontWeight: 700, color: C.primary }}>{detail.login_count}</div>
                  <div style={{ fontSize: '11px', color: C.textMuted }}>累计登录</div>
                </div>
              </div>

              {/* 账户信息 */}
              <Section title="账户信息">
                <InfoGrid items={[
                  { label: '注册时间', value: detail.created_at },
                  { label: '最近登录', value: detail.last_login_at || '暂无' },
                ]} />
              </Section>

              {/* 课件审核权限 */}
              <Section title="课件审核权限">
                {detail.course_assignments.length === 0 ? (
                  <div style={{ fontSize: '13px', color: C.textMuted, padding: '8px 0' }}>未分配课程</div>
                ) : (
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                    {detail.course_assignments.map(a => (
                      <span key={a.course_code} style={{
                        padding: '4px 10px', borderRadius: '6px',
                        background: C.primaryLight, color: C.primary,
                        fontSize: '12px', fontFamily: 'monospace',
                      }}>
                        {a.course_code}{a.course_name !== a.course_code ? ` · ${a.course_name}` : ''}
                      </span>
                    ))}
                  </div>
                )}
              </Section>

              {/* 教案系统权限 */}
              <Section title="教案系统归属">
                {detail.teaching_groups.length === 0 ? (
                  <div style={{ fontSize: '13px', color: C.textMuted, padding: '8px 0' }}>未加入任何教研组</div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                    {detail.teaching_groups.map(g => (
                      <div key={g.group_id} style={{
                        padding: '10px 14px', borderRadius: '10px',
                        background: C.bg, border: `1px solid ${C.border}`,
                        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                      }}>
                        <div>
                          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>
                            {g.is_lead && <span style={{ color: C.warning, marginRight: '4px' }}>★</span>}
                            {g.group_name}
                          </div>
                          <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px' }}>{g.school_name}</div>
                        </div>
                        <span style={{
                          padding: '2px 8px', borderRadius: '8px', fontSize: '11px',
                          background: g.role === 'backbone' ? C.warningLight : C.bg,
                          color: g.role === 'backbone' ? C.warning : C.textSec,
                          border: `1px solid ${g.role === 'backbone' ? 'rgba(245,158,11,0.3)' : C.border}`,
                        }}>
                          {g.is_lead ? '组长' : g.role_name}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </Section>

              {/* 操作区 */}
              <Section title="账户操作">
                <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  {/* 重置密码 */}
                  <div style={{ display: 'flex', gap: '8px' }}>
                    <input
                      type="password"
                      value={resetPwd}
                      onChange={e => setResetPwd(e.target.value)}
                      placeholder="输入新密码（至少6位）"
                      style={{
                        flex: 1, padding: '9px 14px', borderRadius: '8px',
                        border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none',
                      }}
                      onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                      onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                    />
                    <button
                      onClick={handleReset}
                      disabled={resetPwd.length < 6 || saving}
                      style={{
                        padding: '9px 18px', borderRadius: '8px', border: 'none',
                        background: resetPwd.length >= 6 ? C.primary : C.textMuted,
                        color: '#fff', fontSize: '13px', fontWeight: 600,
                        cursor: resetPwd.length >= 6 ? 'pointer' : 'not-allowed',
                      }}
                    >重置密码</button>
                  </div>

                  {/* 启用/禁用 */}
                  <button
                    onClick={handleToggleStatus}
                    disabled={saving}
                    style={{
                      padding: '9px 18px', borderRadius: '8px',
                      border: `1px solid ${detail.status === 'active' ? C.danger : C.success}`,
                      background: detail.status === 'active' ? C.dangerLight : C.successLight,
                      color: detail.status === 'active' ? C.danger : C.success,
                      fontSize: '14px', fontWeight: 600, cursor: 'pointer',
                      width: '100%',
                    }}
                  >
                    {detail.status === 'active' ? '禁用该账户' : '启用该账户'}
                  </button>
                </div>
              </Section>
            </>
          ) : (
            <div style={{ textAlign: 'center', padding: '40px', color: C.danger }}>加载失败</div>
          )}
        </div>
      </div>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ marginBottom: '20px' }}>
      <div style={{
        fontSize: '13px', fontWeight: 600, color: C.textSec,
        marginBottom: '10px', paddingBottom: '6px',
        borderBottom: `1px solid ${C.border}`,
      }}>{title}</div>
      {children}
    </div>
  )
}

function InfoGrid({ items }: { items: { label: string; value: string }[] }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
      {items.map(item => (
        <div key={item.label} style={{ padding: '10px 14px', borderRadius: '8px', background: C.bg }}>
          <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '3px' }}>{item.label}</div>
          <div style={{ fontSize: '13px', color: C.text, fontWeight: 500 }}>{item.value}</div>
        </div>
      ))}
    </div>
  )
}

// ==================== 新建用户表单弹窗 ====================

function CreateUserModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [form, setForm] = useState({ username: '', display_name: '', password: '', role: 'operator' })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const handleCreate = async () => {
    if (!form.username.trim() || !form.display_name.trim() || form.password.length < 6) {
      setError('请填写完整信息（密码至少6位）')
      return
    }
    try {
      setSaving(true)
      setError('')
      await createAdminUser(form)
      onCreated()
      onClose()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '创建失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 10000,
      background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }} onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{
        background: C.white, borderRadius: '20px', width: '460px',
        overflow: 'hidden', boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }}>
        <div style={{
          padding: '20px 24px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        }}>
          <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>新建用户</div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
        </div>
        <div style={{ padding: '24px' }}>
          {error && (
            <div style={{
              padding: '10px 14px', borderRadius: '8px', marginBottom: '16px',
              background: C.dangerLight, color: C.danger, fontSize: '13px',
            }}>{error}</div>
          )}

          {[
            { key: 'username', label: '登录用户名', placeholder: '字母数字下划线', type: 'text' },
            { key: 'display_name', label: '显示名称', placeholder: '例如：张老师', type: 'text' },
            { key: 'password', label: '初始密码', placeholder: '至少6位', type: 'password' },
          ].map(f => (
            <div key={f.key} style={{ marginBottom: '14px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                {f.label}
              </label>
              <input
                type={f.type}
                value={(form as Record<string,string>)[f.key]}
                onChange={e => setForm(p => ({ ...p, [f.key]: e.target.value }))}
                placeholder={f.placeholder}
                style={{
                  width: '100%', padding: '10px 14px', borderRadius: '10px',
                  border: `1px solid ${C.border}`, fontSize: '14px',
                  outline: 'none', boxSizing: 'border-box',
                }}
                onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                onBlur={e => { e.currentTarget.style.borderColor = C.border }}
              />
            </div>
          ))}

          <div style={{ marginBottom: '20px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
              系统角色
            </label>
            <select
              value={form.role}
              onChange={e => setForm(p => ({ ...p, role: e.target.value }))}
              style={{
                width: '100%', padding: '10px 14px', borderRadius: '10px',
                border: `1px solid ${C.border}`, fontSize: '14px',
                outline: 'none', boxSizing: 'border-box', background: C.white,
              }}
            >
              {ROLE_OPTIONS.filter(r => r.value).map(r => (
                <option key={r.value} value={r.value}>{r.label}</option>
              ))}
            </select>
            <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>
              课件审核系统角色。教案系统权限通过加入教研组来配置。
            </div>
          </div>

          <div style={{ display: 'flex', gap: '10px' }}>
            <button
              onClick={onClose}
              style={{
                flex: 1, padding: '10px', borderRadius: '10px',
                border: `1px solid ${C.border}`, background: C.bg,
                fontSize: '14px', color: C.textSec, cursor: 'pointer',
              }}
            >取消</button>
            <button
              onClick={handleCreate}
              disabled={saving}
              style={{
                flex: 2, padding: '10px', borderRadius: '10px', border: 'none',
                background: saving ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`,
                color: '#fff', fontSize: '14px', fontWeight: 600,
                cursor: saving ? 'not-allowed' : 'pointer',
              }}
            >{saving ? '创建中...' : '创建用户'}</button>
          </div>
        </div>
      </div>
    </div>
  )
}

// ==================== 主组件 ====================

export default function AdminPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const fromPath: string = (location.state as { from?: string })?.from || '/'

  const [activeTab, setActiveTab] = useState<'overview' | 'users' | 'orgs' | 'logs'>('overview')

  // 统计
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [statsLoading, setStatsLoading] = useState(true)

  // 用户列表
  const [users, setUsers] = useState<AdminUserListItem[]>([])
  const [userTotal, setUserTotal] = useState(0)
  const [userPage, setUserPage] = useState(1)
  const [userPageSize] = useState(15)
  const [userLoading, setUserLoading] = useState(false)
  const [roleFilter, setRoleFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [keyword, setKeyword] = useState('')
  const [keywordInput, setKeywordInput] = useState('')

  // 用户详情弹窗
  const [detailUserId, setDetailUserId] = useState<string | null>(null)
  // 新建用户弹窗
  const [showCreateModal, setShowCreateModal] = useState(false)

  // 组织架构
  const [orgs, setOrgs] = useState<OrgListItem[]>([])
  const [groups, setGroups] = useState<GroupListItem[]>([])
  const [orgsLoading, setOrgsLoading] = useState(false)

  // 操作日志
  const [logs, setLogs] = useState<AuditLogItem[]>([])
  const [logTotal, setLogTotal] = useState(0)
  const [logPage, setLogPage] = useState(1)
  const [logLoading, setLogLoading] = useState(false)
  const [logAction, setLogAction] = useState('')

  // Toast
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)
  const showToast = (m: string, t: 'success' | 'error') => setToast({ message: m, type: t })

  // ==================== 加载统计 ====================

  const loadStats = useCallback(async () => {
    try {
      setStatsLoading(true)
      const data = await getAdminStats()
      setStats(data)
    } catch {
    } finally {
      setStatsLoading(false)
    }
  }, [])

  useEffect(() => { loadStats() }, [loadStats])

  // ==================== 加载用户列表 ====================

  const loadUsers = useCallback(async () => {
    try {
      setUserLoading(true)
      const data = await getAdminUsers({
        page: userPage, page_size: userPageSize,
        role: roleFilter, status: statusFilter, keyword,
      })
      setUsers(data.users)
      setUserTotal(data.total)
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '加载用户失败', 'error')
    } finally {
      setUserLoading(false)
    }
  }, [userPage, userPageSize, roleFilter, statusFilter, keyword])

  useEffect(() => {
    if (activeTab === 'users') loadUsers()
  }, [activeTab, loadUsers])

  // ==================== 加载组织 ====================

  const loadOrgs = useCallback(async () => {
    try {
      setOrgsLoading(true)
      const [o, g] = await Promise.all([getAdminOrgs(), getAdminGroups()])
      setOrgs(o)
      setGroups(g)
    } catch {
    } finally {
      setOrgsLoading(false)
    }
  }, [])

  useEffect(() => {
    if (activeTab === 'orgs') loadOrgs()
  }, [activeTab, loadOrgs])

  // ==================== 加载日志 ====================

  const loadLogs = useCallback(async () => {
    try {
      setLogLoading(true)
      const data = await getAdminAuditLogs({ page: logPage, page_size: 20, action: logAction })
      setLogs(data.logs)
      setLogTotal(data.total)
    } catch {
    } finally {
      setLogLoading(false)
    }
  }, [logPage, logAction])

  useEffect(() => {
    if (activeTab === 'logs') loadLogs()
  }, [activeTab, loadLogs])

  // 搜索防抖
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const handleKeywordChange = (v: string) => {
    setKeywordInput(v)
    if (searchTimer.current) clearTimeout(searchTimer.current)
    searchTimer.current = setTimeout(() => {
      setKeyword(v)
      setUserPage(1)
    }, 400)
  }

  const handleFilterChange = (role: string, status: string) => {
    setRoleFilter(role)
    setStatusFilter(status)
    setUserPage(1)
  }

  const totalPages = Math.ceil(userTotal / userPageSize)
  const logTotalPages = Math.ceil(logTotal / 20)

  // ==================== 渲染 ====================

  return (
    <div style={{ minHeight: '100vh', background: 'linear-gradient(135deg,#EEF2FF 0%,#FAFBFC 50%,#F0FDF4 100%)' }}>
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
      {detailUserId && (
        <UserDetailModal
          userId={detailUserId}
          onClose={() => setDetailUserId(null)}
          onAction={() => { loadUsers(); loadStats(); showToast('操作成功', 'success') }}
        />
      )}
      {showCreateModal && (
        <CreateUserModal
          onClose={() => setShowCreateModal(false)}
          onCreated={() => { loadUsers(); loadStats(); showToast('用户创建成功', 'success') }}
        />
      )}

      {/* 顶部导航 */}
      <header style={{
        height: '64px', position: 'sticky', top: 0, zIndex: 100,
        background: 'rgba(255,255,255,0.88)', backdropFilter: 'blur(20px)',
        borderBottom: `1px solid ${C.border}`,
        display: 'flex', alignItems: 'center', padding: '0 32px', gap: '16px',
      }}>
        <button
          onClick={() => navigate(fromPath)}
          style={{
            display: 'flex', alignItems: 'center', gap: '6px',
            padding: '8px 16px', borderRadius: '8px',
            border: `1px solid ${C.border}`, background: C.white,
            fontSize: '14px', color: C.textSec, cursor: 'pointer',
          }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.white }}
        >
          {'<- 返回'}
        </button>

        <div style={{ flex: 1, textAlign: 'center' }}>
          <h1 style={{ fontSize: '18px', fontWeight: 700, color: C.text, margin: 0 }}>
            👥 用户管理中心
          </h1>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>
            统一管理用户、组织架构与操作日志
          </div>
        </div>

        {/* 新建用户按钮 */}
        <button
          onClick={() => setShowCreateModal(true)}
          style={{
            padding: '8px 18px', borderRadius: '8px', border: 'none',
            background: `linear-gradient(135deg,${C.primary},#7C3AED)`,
            color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer',
          }}
        >
          + 新建用户
        </button>
      </header>

      <div style={{ maxWidth: '1400px', margin: '0 auto', padding: '24px' }}>

        {/* Tab 切换 */}
        <div style={{
          display: 'flex', gap: '4px', marginBottom: '20px',
          background: C.bg, borderRadius: '12px', padding: '4px',
          border: `1px solid ${C.border}`, width: 'fit-content',
        }}>
          {([
            ['overview', '📊 概览'],
            ['users',    '👥 用户管理'],
            ['orgs',     '🏫 组织架构'],
            ['logs',     '📋 操作日志'],
          ] as const).map(([tab, label]) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              style={{
                padding: '9px 22px', borderRadius: '9px', border: 'none', cursor: 'pointer',
                fontSize: '14px', fontWeight: activeTab === tab ? 600 : 400,
                color: activeTab === tab ? C.primary : C.textSec,
                background: activeTab === tab ? C.white : 'transparent',
                boxShadow: activeTab === tab ? '0 1px 4px rgba(0,0,0,0.08)' : 'none',
                transition: 'all 150ms ease',
              }}
            >
              {label}
            </button>
          ))}
        </div>

        {/* ===== Tab: 概览 ===== */}
        {activeTab === 'overview' && (
          <div>
            {statsLoading ? (
              <div style={{ textAlign: 'center', padding: '60px', color: C.textMuted }}>加载统计数据...</div>
            ) : stats ? (
              <>
                {/* 第一行：总体统计 */}
                <div style={{ display: 'flex', gap: '16px', marginBottom: '16px' }}>
                  <StatCard label="总用户数" value={stats.total_users} sub={`活跃 ${stats.active_users} · 禁用 ${stats.disabled_users}`} color={C.primary} />
                  <StatCard label="组织总数" value={stats.total_orgs} sub={`学校 ${stats.total_schools} 所`} color={C.success} />
                  <StatCard label="教研组" value={stats.total_groups} sub={`教研组成员 ${stats.total_members} 人`} color={C.warning} />
                </div>

                {/* 第二行：角色分布 */}
                <div style={{
                  background: C.white, borderRadius: '16px', border: `1px solid ${C.border}`,
                  padding: '24px', marginBottom: '16px',
                  boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
                }}>
                  <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '16px' }}>
                    课件审核系统角色分布
                  </div>
                  <div style={{ display: 'flex', gap: '12px' }}>
                    {[
                      { label: '管理员', count: stats.admin_count, color: C.danger },
                      { label: '高级操作员', count: stats.senior_operator_count, color: C.warning },
                      { label: '操作员', count: stats.operator_count, color: C.primary },
                      { label: '查看者', count: stats.viewer_count, color: C.textSec },
                    ].map(item => (
                      <div key={item.label} style={{
                        flex: 1, padding: '16px', borderRadius: '12px',
                        background: C.bg, border: `1px solid ${C.border}`,
                        textAlign: 'center',
                      }}>
                        <div style={{ fontSize: '24px', fontWeight: 700, color: item.color }}>{item.count}</div>
                        <div style={{ fontSize: '12px', color: C.textSec, marginTop: '4px' }}>{item.label}</div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* 快捷操作提示 */}
                <div style={{
                  background: C.primaryLight, borderRadius: '12px',
                  border: `1px solid ${C.primary}22`, padding: '16px 20px',
                  fontSize: '13px', color: C.primary,
                }}>
                  💡 点击上方"👥 用户管理"Tab查看完整用户列表，支持按角色/状态筛选、搜索、新建用户、查看用户权限详情。
                </div>
              </>
            ) : null}
          </div>
        )}

        {/* ===== Tab: 用户管理 ===== */}
        {activeTab === 'users' && (
          <div>
            {/* 筛选栏 */}
            <div style={{
              background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`,
              padding: '16px 20px', marginBottom: '16px',
              display: 'flex', gap: '12px', alignItems: 'center', flexWrap: 'wrap',
            }}>
              {/* 搜索 */}
              <input
                value={keywordInput}
                onChange={e => handleKeywordChange(e.target.value)}
                placeholder="搜索用户名或显示名..."
                style={{
                  flex: 1, minWidth: '200px', padding: '8px 14px', borderRadius: '8px',
                  border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none',
                }}
                onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                onBlur={e => { e.currentTarget.style.borderColor = C.border }}
              />

              {/* 角色筛选 */}
              <select
                value={roleFilter}
                onChange={e => handleFilterChange(e.target.value, statusFilter)}
                style={{
                  padding: '8px 14px', borderRadius: '8px',
                  border: `1px solid ${C.border}`, fontSize: '14px',
                  outline: 'none', background: C.white,
                }}
              >
                {ROLE_OPTIONS.map(r => (
                  <option key={r.value} value={r.value}>{r.label}</option>
                ))}
              </select>

              {/* 状态筛选 */}
              <select
                value={statusFilter}
                onChange={e => handleFilterChange(roleFilter, e.target.value)}
                style={{
                  padding: '8px 14px', borderRadius: '8px',
                  border: `1px solid ${C.border}`, fontSize: '14px',
                  outline: 'none', background: C.white,
                }}
              >
                <option value="">全部状态</option>
                <option value="active">正常</option>
                <option value="disabled">已禁用</option>
              </select>

              <div style={{ fontSize: '13px', color: C.textMuted }}>
                共 {userTotal} 个用户
              </div>
            </div>

            {/* 用户表格 */}
            <div style={{
              background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`,
              overflow: 'hidden',
            }}>
              {/* 表头 */}
              <div style={{
                display: 'grid', gridTemplateColumns: '2fr 1.5fr 1.5fr 1.2fr 1.5fr 1fr',
                padding: '12px 20px', background: C.bg,
                borderBottom: `1px solid ${C.border}`,
                fontSize: '12px', fontWeight: 600, color: C.textSec,
              }}>
                <span>用户</span>
                <span>课件审核角色</span>
                <span>教案系统归属</span>
                <span>状态</span>
                <span>最近登录</span>
                <span>操作</span>
              </div>

              {/* 表格内容 */}
              {userLoading ? (
                <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div>
              ) : users.length === 0 ? (
                <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>暂无用户</div>
              ) : (
                users.map((user, idx) => (
                  <div
                    key={user.id}
                    style={{
                      display: 'grid', gridTemplateColumns: '2fr 1.5fr 1.5fr 1.2fr 1.5fr 1fr',
                      padding: '14px 20px', alignItems: 'center',
                      borderBottom: idx < users.length - 1 ? `1px solid ${C.border}` : 'none',
                      transition: 'background 150ms ease',
                    }}
                    onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
                    onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
                  >
                    {/* 用户信息 */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                      <div style={{
                        width: '34px', height: '34px', borderRadius: '50%', flexShrink: 0,
                        background: `linear-gradient(135deg,${C.primary},#7C3AED)`,
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        color: '#fff', fontSize: '13px', fontWeight: 700,
                      }}>
                        {user.display_name?.charAt(0)?.toUpperCase() || 'U'}
                      </div>
                      <div>
                        <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{user.display_name}</div>
                        <div style={{ fontSize: '12px', color: C.textMuted }}>@{user.username}</div>
                      </div>
                    </div>

                    {/* 课件审核角色 */}
                    <div>
                      <RoleBadge role={user.role} roleName={user.role_name} />
                    </div>

                    {/* 教案归属 */}
                    <div>
                      {user.group_count > 0 ? (
                        <div>
                          <div style={{ fontSize: '13px', color: C.text, fontWeight: 500 }}>
                            {user.group_name || '-'}
                          </div>
                          <div style={{ fontSize: '11px', color: C.textMuted }}>
                            {user.school_name}{user.group_count > 1 ? ` 等${user.group_count}个组` : ''}
                          </div>
                        </div>
                      ) : (
                        <span style={{ fontSize: '12px', color: C.textMuted }}>未加入教研组</span>
                      )}
                    </div>

                    {/* 状态 */}
                    <div><StatusBadge status={user.status} /></div>

                    {/* 最近登录 */}
                    <div style={{ fontSize: '12px', color: C.textSec }}>
                      {user.last_login_at
                        ? user.last_login_at.replace('T', ' ').substring(0, 16)
                        : '从未登录'}
                      <div style={{ fontSize: '11px', color: C.textMuted }}>
                        共{user.login_count}次
                      </div>
                    </div>

                    {/* 操作 */}
                    <div>
                      <button
                        onClick={() => setDetailUserId(user.id)}
                        style={{
                          padding: '5px 14px', borderRadius: '7px',
                          border: `1px solid ${C.border}`, background: C.bg,
                          fontSize: '12px', color: C.primary, cursor: 'pointer',
                          fontWeight: 500,
                        }}
                        onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.primaryLight }}
                        onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
                      >
                        详情
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>

            {/* 分页 */}
            {totalPages > 1 && (
              <div style={{
                display: 'flex', justifyContent: 'center', gap: '8px',
                marginTop: '16px', alignItems: 'center',
              }}>
                <button
                  onClick={() => setUserPage(p => Math.max(1, p - 1))}
                  disabled={userPage === 1}
                  style={{
                    padding: '6px 14px', borderRadius: '8px',
                    border: `1px solid ${C.border}`, background: C.white,
                    fontSize: '13px', color: userPage === 1 ? C.textMuted : C.text,
                    cursor: userPage === 1 ? 'not-allowed' : 'pointer',
                  }}
                >上一页</button>
                <span style={{ fontSize: '13px', color: C.textSec }}>
                  第 {userPage} / {totalPages} 页
                </span>
                <button
                  onClick={() => setUserPage(p => Math.min(totalPages, p + 1))}
                  disabled={userPage === totalPages}
                  style={{
                    padding: '6px 14px', borderRadius: '8px',
                    border: `1px solid ${C.border}`, background: C.white,
                    fontSize: '13px', color: userPage === totalPages ? C.textMuted : C.text,
                    cursor: userPage === totalPages ? 'not-allowed' : 'pointer',
                  }}
                >下一页</button>
              </div>
            )}
          </div>
        )}

        {/* ===== Tab: 组织架构 ===== */}
        {activeTab === 'orgs' && (
          <div>
            {orgsLoading ? (
              <div style={{ textAlign: 'center', padding: '60px', color: C.textMuted }}>加载组织数据...</div>
            ) : (
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                {/* 组织列表 */}
                <div style={{
                  background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`,
                  overflow: 'hidden',
                }}>
                  <div style={{
                    padding: '16px 20px', borderBottom: `1px solid ${C.border}`,
                    fontSize: '14px', fontWeight: 600, color: C.text,
                    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                  }}>
                    <span>🏢 区域与学校</span>
                    <span style={{ fontSize: '12px', color: C.textMuted }}>{orgs.length} 个</span>
                  </div>
                  <div style={{ padding: '8px' }}>
                    {orgs.map((org) => (
                      <div key={org.id} style={{
                        padding: '10px 12px', borderRadius: '8px', marginBottom: '4px',
                        background: C.bg,
                        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                      }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                          <span style={{ fontSize: '16px' }}>
                            {org.type === 'region' ? '🌍' : '🏫'}
                          </span>
                          <div>
                            <div style={{ fontSize: '14px', fontWeight: 500, color: C.text }}>{org.name}</div>
                            {org.parent_name && (
                              <div style={{ fontSize: '11px', color: C.textMuted }}>{org.parent_name}</div>
                            )}
                          </div>
                        </div>
                        <div style={{ textAlign: 'right' }}>
                          <div style={{ fontSize: '12px', color: C.textSec }}>
                            {org.group_count} 个教研组
                          </div>
                          <div style={{ fontSize: '11px', color: C.textMuted }}>
                            {org.member_count} 名教师
                          </div>
                        </div>
                      </div>
                    ))}
                    {orgs.length === 0 && (
                      <div style={{ padding: '20px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>
                        暂无组织数据
                      </div>
                    )}
                  </div>
                </div>

                {/* 教研组列表 */}
                <div style={{
                  background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`,
                  overflow: 'hidden',
                }}>
                  <div style={{
                    padding: '16px 20px', borderBottom: `1px solid ${C.border}`,
                    fontSize: '14px', fontWeight: 600, color: C.text,
                    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                  }}>
                    <span>👨‍🏫 教研组</span>
                    <span style={{ fontSize: '12px', color: C.textMuted }}>{groups.length} 个</span>
                  </div>
                  <div style={{ padding: '8px' }}>
                    {groups.map((g) => (
                      <div key={g.id} style={{
                        padding: '10px 12px', borderRadius: '8px', marginBottom: '4px',
                        background: C.bg,
                        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                      }}>
                        <div>
                          <div style={{ fontSize: '14px', fontWeight: 500, color: C.text }}>{g.name}</div>
                          <div style={{ fontSize: '11px', color: C.textSec }}>
                            {g.school_name} · {g.subject}
                          </div>
                          {g.lead_user_name && (
                            <div style={{ fontSize: '11px', color: C.textMuted }}>
                              组长：{g.lead_user_name}
                            </div>
                          )}
                        </div>
                        <div style={{
                          padding: '4px 10px', borderRadius: '8px',
                          background: C.primaryLight, color: C.primary,
                          fontSize: '12px', fontWeight: 600,
                        }}>
                          {g.member_count} 人
                        </div>
                      </div>
                    ))}
                    {groups.length === 0 && (
                      <div style={{ padding: '20px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>
                        暂无教研组数据
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* ===== Tab: 操作日志 ===== */}
        {activeTab === 'logs' && (
          <div>
            {/* 筛选栏 */}
            <div style={{
              background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`,
              padding: '14px 20px', marginBottom: '16px',
              display: 'flex', gap: '12px', alignItems: 'center',
            }}>
              <select
                value={logAction}
                onChange={e => { setLogAction(e.target.value); setLogPage(1) }}
                style={{
                  padding: '8px 14px', borderRadius: '8px',
                  border: `1px solid ${C.border}`, fontSize: '14px',
                  outline: 'none', background: C.white,
                }}
              >
                {ACTION_OPTIONS.map(a => (
                  <option key={a.value} value={a.value}>{a.label}</option>
                ))}
              </select>
              <div style={{ fontSize: '13px', color: C.textMuted }}>
                共 {logTotal} 条记录
              </div>
            </div>

            {/* 日志列表 */}
            <div style={{
              background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`,
              overflow: 'hidden',
            }}>
              {/* 表头 */}
              <div style={{
                display: 'grid', gridTemplateColumns: '1.5fr 2fr 1.5fr 1fr 1.5fr',
                padding: '12px 20px', background: C.bg,
                borderBottom: `1px solid ${C.border}`,
                fontSize: '12px', fontWeight: 600, color: C.textSec,
              }}>
                <span>操作者</span>
                <span>操作内容</span>
                <span>操作类型</span>
                <span>IP地址</span>
                <span>时间</span>
              </div>

              {logLoading ? (
                <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div>
              ) : logs.length === 0 ? (
                <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>暂无日志</div>
              ) : (
                logs.map((log, idx) => (
                  <div
                    key={log.id}
                    style={{
                      display: 'grid', gridTemplateColumns: '1.5fr 2fr 1.5fr 1fr 1.5fr',
                      padding: '12px 20px', alignItems: 'center',
                      borderBottom: idx < logs.length - 1 ? `1px solid ${C.border}` : 'none',
                      fontSize: '13px',
                    }}
                  >
                    <div>
                      <div style={{ fontWeight: 500, color: C.text }}>{log.display_name || log.username}</div>
                      <div style={{ fontSize: '11px', color: C.textMuted }}>@{log.username}</div>
                    </div>
                    <div style={{ color: C.textSec, fontSize: '12px', fontFamily: 'monospace' }}>
                      {(() => {
                        try {
                          const d = JSON.parse(log.detail)
                          return Object.entries(d).map(([k, v]) => `${k}: ${v}`).join(' · ')
                        } catch {
                          return log.detail
                        }
                      })()}
                    </div>
                    <div>
                      <span style={{
                        padding: '2px 8px', borderRadius: '6px', fontSize: '11px',
                        background: log.action.startsWith('user') ? C.primaryLight : C.warningLight,
                        color: log.action.startsWith('user') ? C.primary : C.warning,
                        fontWeight: 600,
                      }}>
                        {log.action_name}
                      </span>
                    </div>
                    <div style={{ color: C.textMuted, fontFamily: 'monospace', fontSize: '11px' }}>
                      {log.ip}
                    </div>
                    <div style={{ color: C.textSec, fontSize: '11px' }}>
                      {typeof log.created_at === 'string'
                        ? log.created_at.replace('T', ' ').substring(0, 16)
                        : new Date(log.created_at).toLocaleString('zh-CN')}
                    </div>
                  </div>
                ))
              )}
            </div>

            {/* 分页 */}
            {logTotalPages > 1 && (
              <div style={{
                display: 'flex', justifyContent: 'center', gap: '8px',
                marginTop: '16px', alignItems: 'center',
              }}>
                <button
                  onClick={() => setLogPage(p => Math.max(1, p - 1))}
                  disabled={logPage === 1}
                  style={{
                    padding: '6px 14px', borderRadius: '8px',
                    border: `1px solid ${C.border}`, background: C.white,
                    fontSize: '13px', color: logPage === 1 ? C.textMuted : C.text,
                    cursor: logPage === 1 ? 'not-allowed' : 'pointer',
                  }}
                >上一页</button>
                <span style={{ fontSize: '13px', color: C.textSec }}>
                  第 {logPage} / {logTotalPages} 页
                </span>
                <button
                  onClick={() => setLogPage(p => Math.min(logTotalPages, p + 1))}
                  disabled={logPage === logTotalPages}
                  style={{
                    padding: '6px 14px', borderRadius: '8px',
                    border: `1px solid ${C.border}`, background: C.white,
                    fontSize: '13px', color: logPage === logTotalPages ? C.textMuted : C.text,
                    cursor: logPage === logTotalPages ? 'not-allowed' : 'pointer',
                  }}
                >下一页</button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
