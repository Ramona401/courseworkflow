/**
 * SchoolAdminPage.tsx — 学校管理员管理页面
 *
 * 路由：/school-admin（独立页面，不在任何Layout内）
 * 入口：两个系统Header下拉菜单 → "学校管理"
 *
 * 关键设计：
 *   - 用户列表：调用 /api/v1/school-admin/users（senior_operator可访问）
 *   - 教研组列表：调用 /api/v1/school-admin/groups（senior_operator可访问）
 *   - 其他操作（详情/成员管理）仍复用Admin子组件（后端已有权限控制）
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  getSchoolUsers,
  getSchoolGroups,
  getMySchool,
  type SchoolInfo,
  type SchoolUserItem,
  type SchoolGroupListItem,
} from '@/api/school-admin'
import { deleteAdminGroup } from '@/api/admin'
import type { GroupListItem } from '@/api/admin'

import { C, ROLE_OPTIONS } from '../admin/components/adminConstants'
import { Toast, RoleBadge, StatusBadge } from '../admin/components/adminShared'
import { ConfirmDialog }   from '../admin/components/ConfirmDialog'
import { GroupFormModal }  from '../admin/components/GroupFormModal'
import { MemberPanel }     from '../admin/components/MemberPanel'
import { UserDetailModal } from '../admin/components/UserDetailModal'
import { CreateUserModal } from '../admin/components/CreateUserModal'

// ==================== ColCard（与AdminPage同款）====================
function ColCard({ title, count, loading, emptyText, children }: {
  title: React.ReactNode; count?: number; loading?: boolean; emptyText?: string; children?: React.ReactNode
}) {
  return (
    <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, overflow: 'hidden', display: 'flex', flexDirection: 'column', minHeight: '400px' }}>
      <div style={{ padding: '14px 16px', borderBottom: `1px solid ${C.border}`, background: C.bg, display: 'flex', alignItems: 'center', gap: '6px', flexShrink: 0 }}>
        <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{title}</div>
        {count !== undefined && <span style={{ fontSize: '11px', color: C.textMuted }}>({count})</span>}
      </div>
      <div style={{ flex: 1, overflowY: 'auto' }}>
        {loading
          ? <div style={{ padding: '32px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>加载中...</div>
          : <>{children}{emptyText && <div style={{ padding: '32px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>{emptyText}</div>}</>}
      </div>
    </div>
  )
}

// ==================== 主组件 ====================
export default function SchoolAdminPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const fromPath: string = (location.state as { from?: string })?.from || '/'

  // ---- 学校信息 ----
  const [schoolInfo, setSchoolInfo]         = useState<SchoolInfo | null>(null)
  const [schoolLoading, setSchoolLoading]   = useState(true)
  const [notSchoolAdmin, setNotSchoolAdmin] = useState(false)

  // ---- Tab ----
  const [activeTab, setActiveTab] = useState<'users' | 'groups'>('users')

  // ---- 教师管理 ----
  const [users, setUsers]               = useState<SchoolUserItem[]>([])
  const [userTotal, setUserTotal]       = useState(0)
  const [userLoading, setUserLoading]   = useState(false)
  const [roleFilter, setRoleFilter]     = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword]           = useState('')
  const [detailUserId, setDetailUserId] = useState<string | null>(null)
  const [showCreateModal, setShowCreateModal] = useState(false)

  // ---- 教研组管理 ----
  const [groups, setGroups]               = useState<SchoolGroupListItem[]>([])
  const [groupsLoading, setGroupsLoading] = useState(false)
  const [expandedGroupId, setExpandedGroupId] = useState<string | null>(null)
  const [groupModal, setGroupModal] = useState<{
    open: boolean; mode: 'create' | 'edit'; initial?: GroupListItem
  }>({ open: false, mode: 'create' })
  const [confirmDel, setConfirmDel] = useState<{
    open: boolean; title: string; message: string; onConfirm: () => void
  }>({ open: false, title: '', message: '', onConfirm: () => {} })

  // ---- Toast ----
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)
  const showToast = useCallback((m: string, t: 'success' | 'error') => setToast({ message: m, type: t }), [])

  // ---- 搜索防抖 ----
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(() => () => { if (searchTimer.current) clearTimeout(searchTimer.current) }, [])

  // ==================== 初始化：验证学校管理员身份 ====================
  useEffect(() => {
    const check = async () => {
      try {
        setSchoolLoading(true)
        const school = await getMySchool()
        setSchoolInfo(school)
      } catch {
        setNotSchoolAdmin(true)
      } finally {
        setSchoolLoading(false)
      }
    }
    check()
  }, [])

  // ==================== 教师列表 ====================
  // 调用 /api/v1/school-admin/users（后端自动按学校+教研组关联筛选）
  const loadUsers = useCallback(async () => {
    if (!schoolInfo) return
    try {
      setUserLoading(true)
      const data = await getSchoolUsers()
      // 前端本地过滤（不走后端admin权限接口）
      let filtered = data.users ?? []
      if (roleFilter)   filtered = filtered.filter(u => u.role === roleFilter)
      if (statusFilter) filtered = filtered.filter(u => u.status === statusFilter)
      if (keyword)      filtered = filtered.filter(u =>
        u.username.toLowerCase().includes(keyword.toLowerCase()) ||
        u.display_name.toLowerCase().includes(keyword.toLowerCase())
      )
      setUsers(filtered)
      setUserTotal(filtered.length)
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '加载用户失败', 'error')
    } finally {
      setUserLoading(false)
    }
  }, [schoolInfo, roleFilter, statusFilter, keyword, showToast])

  useEffect(() => {
    if (activeTab === 'users' && schoolInfo) loadUsers()
  }, [activeTab, loadUsers, schoolInfo])

  // ==================== 教研组列表 ====================
  // 调用 /api/v1/school-admin/groups（后端自动按学校筛选）
  const loadGroups = useCallback(async () => {
    if (!schoolInfo) return
    try {
      setGroupsLoading(true)
      const data = await getSchoolGroups()
      setGroups(data.groups ?? [])
    } catch {
      showToast('加载教研组失败', 'error')
    } finally {
      setGroupsLoading(false)
    }
  }, [schoolInfo, showToast])

  useEffect(() => {
    if (activeTab === 'groups' && schoolInfo) loadGroups()
  }, [activeTab, loadGroups, schoolInfo])

  // ---- 搜索防抖 ----
  const handleKeywordChange = (v: string) => {
    setKeywordInput(v)
    if (searchTimer.current) clearTimeout(searchTimer.current)
    searchTimer.current = setTimeout(() => { setKeyword(v) }, 400)
  }

  // ---- 删除教研组 ----
  const handleDeleteGroup = (g: SchoolGroupListItem) => {
    setConfirmDel({
      open: true, title: '删除教研组',
      message: `确认删除教研组「${g.name}」？此操作不可撤销。`,
      onConfirm: async () => {
        try {
          await deleteAdminGroup(g.id)
          showToast('删除成功', 'success')
          loadGroups()
        } catch (e: unknown) {
          showToast(e instanceof Error ? e.message : '删除失败', 'error')
        }
        setConfirmDel(p => ({ ...p, open: false }))
      },
    })
  }

  // ==================== 加载中 ====================
  if (schoolLoading) {
    return (
      <div style={{ minHeight: '100vh', background: C.bg, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div style={{ textAlign: 'center', color: C.textMuted }}>
          <div style={{ width: '36px', height: '36px', margin: '0 auto 12px', border: `3px solid ${C.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
          <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
          正在验证学校管理员身份...
        </div>
      </div>
    )
  }

  // ==================== 无权限 ====================
  if (notSchoolAdmin) {
    return (
      <div style={{ minHeight: '100vh', background: C.bg, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div style={{ textAlign: 'center', padding: '40px', background: C.white, borderRadius: '20px', border: `1px solid ${C.border}`, maxWidth: '400px' }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>🏫</div>
          <div style={{ fontSize: '18px', fontWeight: 700, color: C.text, marginBottom: '8px' }}>未绑定学校管理员</div>
          <div style={{ fontSize: '14px', color: C.textSec, marginBottom: '24px' }}>
            您的账号尚未绑定到任何学校的管理员，请联系系统管理员配置。
          </div>
          <button onClick={() => navigate(fromPath)} style={{ padding: '10px 24px', borderRadius: '10px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>
            返回
          </button>
        </div>
      </div>
    )
  }

  // ==================== 主渲染 ====================
  return (
    <div style={{ minHeight: '100vh', background: 'linear-gradient(135deg,#EEF2FF 0%,#FAFBFC 50%,#F0FDF4 100%)' }}>

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      {confirmDel.open && (
        <ConfirmDialog
          title={confirmDel.title} message={confirmDel.message}
          onConfirm={confirmDel.onConfirm}
          onCancel={() => setConfirmDel(p => ({ ...p, open: false }))}
        />
      )}

      {detailUserId && (
        <UserDetailModal
          userId={detailUserId}
          onClose={() => setDetailUserId(null)}
          onAction={() => { loadUsers(); showToast('操作成功', 'success') }}
        />
      )}

      {showCreateModal && (
        <CreateUserModal
          onClose={() => setShowCreateModal(false)}
          onCreated={() => { loadUsers(); showToast('用户创建成功', 'success') }}
        />
      )}

      {groupModal.open && schoolInfo && (
        <GroupFormModal
          mode={groupModal.mode}
          schoolId={schoolInfo.id}
          schoolName={schoolInfo.name}
          initial={groupModal.initial as GroupListItem | undefined}
          onClose={() => setGroupModal(p => ({ ...p, open: false }))}
          onSaved={() => {
            showToast(groupModal.mode === 'create' ? '创建成功' : '更新成功', 'success')
            loadGroups()
          }}
        />
      )}

      {/* ---- 顶部导航 ---- */}
      <header style={{ height: '64px', position: 'sticky', top: 0, zIndex: 100, background: 'rgba(255,255,255,0.88)', backdropFilter: 'blur(20px)', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', padding: '0 32px', gap: '16px' }}>
        <button
          onClick={() => navigate(fromPath)}
          style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.white }}>
          ← 返回
        </button>
        <div style={{ flex: 1, textAlign: 'center' }}>
          <h1 style={{ fontSize: '18px', fontWeight: 700, color: C.text, margin: 0 }}>🏫 {schoolInfo?.name || '学校管理中心'}</h1>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>管理本校教师账号与教研组</div>
        </div>
        {activeTab === 'users' && (
          <button onClick={() => setShowCreateModal(true)} style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>+ 新建教师</button>
        )}
        {activeTab === 'groups' && (
          <button onClick={() => setGroupModal({ open: true, mode: 'create' })} style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>+ 新建教研组</button>
        )}
      </header>

      <div style={{ maxWidth: '1400px', margin: '0 auto', padding: '24px' }}>

        {/* Tab切换 */}
        <div style={{ display: 'flex', gap: '4px', marginBottom: '20px', background: C.bg, borderRadius: '12px', padding: '4px', border: `1px solid ${C.border}`, width: 'fit-content' }}>
          {(['users', 'groups'] as const).map((tab, i) => (
            <button key={tab} onClick={() => setActiveTab(tab)} style={{ padding: '9px 22px', borderRadius: '9px', border: 'none', cursor: 'pointer', fontSize: '14px', fontWeight: activeTab === tab ? 600 : 400, color: activeTab === tab ? C.primary : C.textSec, background: activeTab === tab ? C.white : 'transparent', boxShadow: activeTab === tab ? '0 1px 4px rgba(0,0,0,0.08)' : 'none', transition: 'all 150ms ease' }}>
              {['👥 教师管理', '👨‍🏫 教研组管理'][i]}
            </button>
          ))}
        </div>

        {/* ===== 教师管理Tab ===== */}
        {activeTab === 'users' && (
          <div>
            <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, padding: '16px 20px', marginBottom: '16px', display: 'flex', gap: '10px', alignItems: 'center', flexWrap: 'wrap' }}>
              <input value={keywordInput} onChange={e => handleKeywordChange(e.target.value)} placeholder="搜索用户名或显示名..."
                style={{ flex: 1, minWidth: '180px', padding: '8px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none' }}
                onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                onBlur={e => { e.currentTarget.style.borderColor = C.border }} />
              <select value={roleFilter} onChange={e => { setRoleFilter(e.target.value) }}
                style={{ padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: C.white }}>
                {ROLE_OPTIONS.filter(r => r.value !== 'admin').map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
              </select>
              <select value={statusFilter} onChange={e => { setStatusFilter(e.target.value) }}
                style={{ padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: C.white }}>
                <option value="">全部状态</option>
                <option value="active">正常</option>
                <option value="disabled">已禁用</option>
              </select>
              <div style={{ fontSize: '13px', color: C.textMuted }}>共 {userTotal} 位教师</div>
            </div>

            <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
              <div style={{ display: 'grid', gridTemplateColumns: '2fr 1.5fr 1.2fr 1.5fr 1fr', padding: '12px 20px', background: C.bg, borderBottom: `1px solid ${C.border}`, fontSize: '12px', fontWeight: 600, color: C.textSec }}>
                <span>教师</span><span>系统角色</span><span>状态</span><span>最近登录</span><span>操作</span>
              </div>
              {userLoading ? (
                <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div>
              ) : users.length === 0 ? (
                <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>暂无教师数据</div>
              ) : users.map((user, idx) => (
                <div key={user.id}
                  style={{ display: 'grid', gridTemplateColumns: '2fr 1.5fr 1.2fr 1.5fr 1fr', padding: '14px 20px', alignItems: 'center', borderBottom: idx < users.length - 1 ? `1px solid ${C.border}` : 'none', transition: 'background 150ms ease' }}
                  onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
                  onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}>

                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                    <div style={{ width: '34px', height: '34px', borderRadius: '50%', flexShrink: 0, background: `linear-gradient(135deg,${C.primary},#7C3AED)`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: '13px', fontWeight: 700 }}>
                      {user.display_name?.charAt(0)?.toUpperCase() || 'U'}
                    </div>
                    <div>
                      <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{user.display_name}</div>
                      <div style={{ fontSize: '12px', color: C.textMuted }}>@{user.username}</div>
                    </div>
                  </div>

                  <div><RoleBadge role={user.role} /></div>
                  <div><StatusBadge status={user.status} /></div>

                  <div style={{ fontSize: '12px', color: C.textSec }}>
                    {user.last_login_at ? String(user.last_login_at).replace('T', ' ').substring(0, 16) : '从未登录'}
                    <div style={{ fontSize: '11px', color: C.textMuted }}>共{user.login_count}次</div>
                  </div>

                  <div>
                    <button onClick={() => setDetailUserId(user.id)}
                      style={{ padding: '5px 14px', borderRadius: '7px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '12px', color: C.primary, cursor: 'pointer', fontWeight: 500 }}
                      onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.primaryLight }}
                      onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}>
                      详情
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* ===== 教研组管理Tab ===== */}
        {activeTab === 'groups' && (
          <ColCard title={<span>👨‍🏫 本校教研组</span>} count={groups.length} loading={groupsLoading}
            emptyText={groups.length === 0 ? '暂无教研组，点击右上角「新建教研组」添加' : undefined}>
            {groups.map(g => (
              <div key={g.id}>
                <div style={{ padding: '12px 14px' }}>
                  <div style={{ display: 'flex', alignItems: 'flex-start', gap: '8px' }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{g.name}</div>
                      <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '2px' }}>
                        {g.subject}{g.grade_range ? ` · ${g.grade_range}` : ''}
                        {g.lead_user_names ? ` · 组长：${g.lead_user_names}` : g.lead_user_name ? ` · 组长：${g.lead_user_name}` : ' · 暂无组长'}
                        {` · ${g.member_count} 人`}
                      </div>
                    </div>
                    <StatusBadge status={g.status} />
                  </div>
                  <div style={{ display: 'flex', gap: '6px', marginTop: '8px' }}>
                    <button onClick={() => setGroupModal({ open: true, mode: 'edit', initial: g as unknown as GroupListItem })}
                      style={{ padding: '3px 8px', borderRadius: '5px', border: `1px solid ${C.primaryLight}`, background: C.primaryLight, color: C.primary, fontSize: '11px', cursor: 'pointer', fontWeight: 500 }}>✏️ 编辑</button>
                    <button onClick={() => handleDeleteGroup(g)}
                      style={{ padding: '3px 8px', borderRadius: '5px', border: `1px solid ${C.dangerLight}`, background: C.dangerLight, color: C.danger, fontSize: '11px', cursor: 'pointer', fontWeight: 500 }}>🗑️ 删除</button>
                    <button onClick={() => setExpandedGroupId(p => p === g.id ? null : g.id)}
                      style={{ padding: '3px 8px', borderRadius: '5px', fontSize: '11px', cursor: 'pointer', fontWeight: 500, border: `1px solid ${expandedGroupId === g.id ? C.purpleLight : C.border}`, background: expandedGroupId === g.id ? C.purpleLight : C.bg, color: expandedGroupId === g.id ? C.purple : C.textSec }}>
                      {expandedGroupId === g.id ? '收起 ▲' : '👥 成员'}
                    </button>
                  </div>
                </div>
                {expandedGroupId === g.id && <MemberPanel groupId={g.id} onClose={() => setExpandedGroupId(null)} />}
                <div style={{ height: '1px', background: C.border, margin: '0 14px' }} />
              </div>
            ))}
          </ColCard>
        )}

      </div>
    </div>
  )
}
