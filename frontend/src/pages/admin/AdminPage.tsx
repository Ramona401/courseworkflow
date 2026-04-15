/**
 * AdminPage — 统一用户管理中心（主文件）
 *
 * 路由：/admin（独立页面，不在任何Layout内）
 * 权限：仅admin（路由层RoleGuard保护）
 *
 * 5个Tab：
 *   📊 概览       — 统计卡片 + 角色分布横条图 + 最近10条日志快览
 *   👥 用户管理   — 用户列表+学校筛选+详情弹窗（双Tab：基本信息/操作记录）
 *   🏫 组织架构   — 三栏递进：区域→学校→教研组，完整CRUD + 成员管理
 *                   修复：区域列表加载完成后自动选中首个区域并展开学校
 *                         学校列表加载完成后如只有一所学校自动选中并展开教研组
 *   📋 操作日志   — 用户名搜索+日期范围+操作类型+详情展开
 *   🎭 角色权限   — 系统内置角色只读展示 + 自定义角色完整管理
 *
 * 角色名称与学校体系对齐：
 *   admin → 系统管理员 / senior_operator → 学校管理员
 *   operator → 骨干教师 / viewer → 普通教师
 *
 * FE-AD-01修复：添加searchTimer的useEffect卸载清理，防止组件卸载后触发setState
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  getAdminStats, getAdminUsers, getAdminAuditLogs,
  getAdminOrgs, updateAdminOrg, deleteAdminOrg,
  getAdminGroups,
  deleteAdminGroup,
} from '@/api/admin'
import type {
  AdminStats, AdminUserListItem, AuditLogItem,
  OrgListItem, GroupListItem,
} from '@/api/admin'

// ---- 子组件 ----
import { C, ROLE_OPTIONS, ACTION_OPTIONS, fmt, getActionStyle, rowBtn } from './components/adminConstants'
import { Toast, RoleBadge, StatusBadge, StatCard } from './components/adminShared'
import { ConfirmDialog }    from './components/ConfirmDialog'
import { OrgFormModal }     from './components/OrgFormModal'
import { GroupFormModal }   from './components/GroupFormModal'
import { MemberPanel }      from './components/MemberPanel'
import { RoleBarChart }     from './components/RoleBarChart'
import { RecentLogsCard }   from './components/RecentLogsCard'
import { UserDetailModal }  from './components/UserDetailModal'
import { CreateUserModal }  from './components/CreateUserModal'
import { RolesTab }         from './components/RolesTab'

// ==================== 三栏列卡（组织架构Tab内部组件）====================
function ColCard({ title, count, onAdd, addLabel, loading, empty, children }: {
  title: React.ReactNode; count?: number
  onAdd?: () => void; addLabel?: string
  loading?: boolean; empty?: string
  children: React.ReactNode
}) {
  return (
    <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, overflow: 'hidden', display: 'flex', flexDirection: 'column', minHeight: '400px' }}>
      <div style={{ padding: '14px 16px', borderBottom: `1px solid ${C.border}`, background: C.bg, display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0 }}>
        <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, display: 'flex', alignItems: 'center', gap: '6px' }}>
          {title}
          {count !== undefined && <span style={{ fontSize: '11px', color: C.textMuted, fontWeight: 400 }}>({count})</span>}
        </div>
        {onAdd && (
          <button onClick={onAdd} style={{ padding: '5px 12px', borderRadius: '7px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '12px', fontWeight: 600, cursor: 'pointer' }}>
            + {addLabel || '新建'}
          </button>
        )}
      </div>
      <div style={{ flex: 1, overflowY: 'auto' }}>
        {loading ? (
          <div style={{ padding: '32px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>加载中...</div>
        ) : (
          <>
            {children}
            {empty && <div style={{ padding: '32px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>{empty}</div>}
          </>
        )}
      </div>
    </div>
  )
}

// ==================== 主组件 ====================
export default function AdminPage() {
  const navigate  = useNavigate()
  const location  = useLocation()
  const fromPath: string = (location.state as { from?: string })?.from || '/'

  // ---- Tab状态 ----
  const [activeTab, setActiveTab] = useState<'overview' | 'users' | 'orgs' | 'logs' | 'roles'>('overview')

  // ---- 概览Tab ----
  const [stats, setStats]                         = useState<AdminStats | null>(null)
  const [statsLoading, setStatsLoading]           = useState(true)
  const [recentLogs, setRecentLogs]               = useState<AuditLogItem[]>([])
  const [recentLogsLoading, setRecentLogsLoading] = useState(false)

  // ---- 用户管理Tab ----
  const [users, setUsers]               = useState<AdminUserListItem[]>([])
  const [userTotal, setUserTotal]       = useState(0)
  const [userPage, setUserPage]         = useState(1)
  const [userLoading, setUserLoading]   = useState(false)
  const [roleFilter, setRoleFilter]     = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [keyword, setKeyword]           = useState('')
  const [keywordInput, setKeywordInput] = useState('')
  const [schoolFilter, setSchoolFilter] = useState('')
  const [schools, setSchools]           = useState<OrgListItem[]>([])
  const [schoolsLoaded, setSchoolsLoaded] = useState(false)
  const [detailUserId, setDetailUserId]   = useState<string | null>(null)
  const [showCreateModal, setShowCreateModal] = useState(false)

  // ---- 组织架构Tab ----
  const [regions, setRegions]         = useState<OrgListItem[]>([])
  const [schools2, setSchools2]       = useState<OrgListItem[]>([])
  const [groups2, setGroups2]         = useState<GroupListItem[]>([])
  const [selRegion, setSelRegion]     = useState<OrgListItem | null>(null)
  const [selSchool, setSelSchool]     = useState<OrgListItem | null>(null)
  const [regLoading, setRegLoading]   = useState(false)
  const [schLoading, setSchLoading]   = useState(false)
  const [grpLoading, setGrpLoading]   = useState(false)
  const [expandedGroupId, setExpandedGroupId] = useState<string | null>(null)

  const [orgModal, setOrgModal] = useState<{
    open: boolean; mode: 'create' | 'edit'; type: 'region' | 'school'; initial?: OrgListItem
  }>({ open: false, mode: 'create', type: 'region' })

  const [groupModal, setGroupModal] = useState<{
    open: boolean; mode: 'create' | 'edit'; initial?: GroupListItem
  }>({ open: false, mode: 'create' })

  const [confirmDel, setConfirmDel] = useState<{
    open: boolean; title: string; message: string; onConfirm: () => void
  }>({ open: false, title: '', message: '', onConfirm: () => {} })

  // ---- 操作日志Tab ----
  const [logs, setLogs]               = useState<AuditLogItem[]>([])
  const [logTotal, setLogTotal]       = useState(0)
  const [logPage, setLogPage]         = useState(1)
  const [logLoading, setLogLoading]   = useState(false)
  const [logFilterInput, setLogFilterInput] = useState({ username: '', action: '', startDate: '', endDate: '' })
  const [logFilters, setLogFilters]         = useState({ username: '', action: '', startDate: '', endDate: '' })
  const [expandedLogId, setExpandedLogId]   = useState<string | null>(null)

  // ---- 全局Toast ----
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)
  const showToast = useCallback((m: string, t: 'success' | 'error') => setToast({ message: m, type: t }), [])

  // ==================== 数据加载 ====================

  const loadStats = useCallback(async () => {
    try { setStatsLoading(true); setStats(await getAdminStats()) }
    catch { /* 忽略 */ } finally { setStatsLoading(false) }
  }, [])
  useEffect(() => { loadStats() }, [loadStats])

  const loadRecentLogs = useCallback(async () => {
    try { setRecentLogsLoading(true); setRecentLogs((await getAdminAuditLogs({ page: 1, page_size: 10 })).logs) }
    catch { /* 忽略 */ } finally { setRecentLogsLoading(false) }
  }, [])
  useEffect(() => { if (activeTab === 'overview') loadRecentLogs() }, [activeTab, loadRecentLogs])

  // 用户Tab：懒加载学校筛选列表
  const loadSchools = useCallback(async () => {
    if (schoolsLoaded) return
    try { const all = await getAdminOrgs(); setSchools(all.filter(o => o.type === 'school')); setSchoolsLoaded(true) }
    catch { /* 忽略 */ }
  }, [schoolsLoaded])
  useEffect(() => { if (activeTab === 'users') loadSchools() }, [activeTab, loadSchools])

  const loadUsers = useCallback(async () => {
    try {
      setUserLoading(true)
      const data = await getAdminUsers({ page: userPage, page_size: 15, role: roleFilter, status: statusFilter, keyword, school_id: schoolFilter || undefined })
      setUsers(data.users); setUserTotal(data.total)
    } catch (e: unknown) { showToast(e instanceof Error ? e.message : '加载用户失败', 'error') }
    finally { setUserLoading(false) }
  }, [userPage, roleFilter, statusFilter, keyword, schoolFilter, showToast])
  useEffect(() => { if (activeTab === 'users') loadUsers() }, [activeTab, loadUsers])

  // ---- 组织架构：加载学校（带自动选中逻辑）----
  const loadSchools2 = useCallback(async (regionId: string) => {
    try {
      setSchLoading(true)
      const list = await getAdminOrgs({ type: 'school', parent_id: regionId })
      setSchools2(list)
      // 如果只有一所学校，自动选中并加载教研组
      if (list.length === 1) {
        setSelSchool(list[0])
        setGrpLoading(true)
        try {
          const grps = await getAdminGroups(list[0].id)
          setGroups2(grps)
        } catch { /* 忽略 */ } finally { setGrpLoading(false) }
      }
    } catch { /* 忽略 */ } finally { setSchLoading(false) }
  }, [])

  const loadGroups2 = useCallback(async (schoolId: string) => {
    try { setGrpLoading(true); setGroups2(await getAdminGroups(schoolId)) }
    catch { /* 忽略 */ } finally { setGrpLoading(false) }
  }, [])

  // ---- 组织架构：加载区域（带自动选中逻辑）----
  const loadRegions = useCallback(async () => {
    try {
      setRegLoading(true)
      const list = await getAdminOrgs({ type: 'region' })
      setRegions(list)
      // 如果只有一个区域，自动选中并加载其学校
      if (list.length === 1) {
        setSelRegion(list[0])
        setSelSchool(null); setSchools2([]); setGroups2([])
        await loadSchools2(list[0].id)
      }
    } catch { /* 忽略 */ } finally { setRegLoading(false) }
  }, [loadSchools2])

  useEffect(() => {
    if (activeTab === 'orgs') {
      // 切换到组织架构Tab时重新加载（保持自动展开逻辑）
      setSelRegion(null); setSelSchool(null); setSchools2([]); setGroups2([])
      setExpandedGroupId(null)
      loadRegions()
    }
  }, [activeTab, loadRegions])

  const handleSelectRegion = (r: OrgListItem) => {
    setSelRegion(r); setSelSchool(null); setGroups2([]); setExpandedGroupId(null)
    loadSchools2(r.id)
  }
  const handleSelectSchool = (s: OrgListItem) => {
    setSelSchool(s); setExpandedGroupId(null)
    loadGroups2(s.id)
  }

  const handleDeleteOrg = (org: OrgListItem) => {
    setConfirmDel({
      open: true,
      title: `删除${org.type === 'region' ? '区域' : '学校'}`,
      message: `确认删除「${org.name}」？此操作不可撤销。如有下属组织或成员，将无法删除。`,
      onConfirm: async () => {
        try {
          await deleteAdminOrg(org.id)
          showToast('删除成功', 'success')
          if (org.type === 'region') {
            setSelRegion(null); setSchools2([]); setGroups2([])
            loadRegions()
          } else {
            setSelSchool(null); setGroups2([])
            if (selRegion) loadSchools2(selRegion.id)
          }
        } catch (e: unknown) { showToast(e instanceof Error ? e.message : '删除失败', 'error') }
        setConfirmDel(p => ({ ...p, open: false }))
      },
    })
  }

  const handleToggleOrgStatus = async (org: OrgListItem) => {
    const newStatus = org.status === 'active' ? 'disabled' : 'active'
    try {
      await updateAdminOrg(org.id, { name: org.name, admin_user_id: org.admin_user_id, status: newStatus })
      showToast(newStatus === 'active' ? '已启用' : '已禁用', 'success')
      if (org.type === 'region') loadRegions()
      else if (selRegion) loadSchools2(selRegion.id)
    } catch (e: unknown) { showToast(e instanceof Error ? e.message : '操作失败', 'error') }
  }

  const handleDeleteGroup = (g: GroupListItem) => {
    setConfirmDel({
      open: true, title: '删除教研组',
      message: `确认删除教研组「${g.name}」？此操作不可撤销。`,
      onConfirm: async () => {
        try {
          await deleteAdminGroup(g.id)
          showToast('删除成功', 'success')
          if (selSchool) loadGroups2(selSchool.id)
        } catch (e: unknown) { showToast(e instanceof Error ? e.message : '删除失败', 'error') }
        setConfirmDel(p => ({ ...p, open: false }))
      },
    })
  }

  // ---- 操作日志 ----
  const loadLogs = useCallback(async () => {
    try {
      setLogLoading(true)
      const data = await getAdminAuditLogs({
        page: logPage, page_size: 20,
        action: logFilters.action || undefined,
        username: logFilters.username || undefined,
        start_date: logFilters.startDate || undefined,
        end_date: logFilters.endDate || undefined,
      })
      setLogs(data.logs); setLogTotal(data.total)
    } catch { /* 忽略 */ } finally { setLogLoading(false) }
  }, [logPage, logFilters])
  useEffect(() => { if (activeTab === 'logs') loadLogs() }, [activeTab, loadLogs])

  const handleLogSearch = () => { setLogFilters({ ...logFilterInput }); setLogPage(1); setExpandedLogId(null) }
  const handleLogReset  = () => {
    const e = { username: '', action: '', startDate: '', endDate: '' }
    setLogFilterInput(e); setLogFilters(e); setLogPage(1); setExpandedLogId(null)
  }
  const toggleLogDetail = (id: string) => setExpandedLogId(p => p === id ? null : id)

  // 用户搜索防抖
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const handleKeywordChange = (v: string) => {
    setKeywordInput(v)
    if (searchTimer.current) clearTimeout(searchTimer.current)
    searchTimer.current = setTimeout(() => { setKeyword(v); setUserPage(1) }, 400)
  }

  // FE-AD-01修复：组件卸载时清理防抖定时器，防止卸载后触发setState导致内存泄漏
  useEffect(() => {
    return () => {
      if (searchTimer.current) clearTimeout(searchTimer.current)
    }
  }, [])

  const totalPages    = Math.ceil(userTotal / 15)
  const logTotalPages = Math.ceil(logTotal / 20)
  const inputStyle: React.CSSProperties = {
    padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`,
    fontSize: '13px', outline: 'none', background: C.white, color: C.text,
  }

  // ==================== 渲染 ====================
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
          onAction={() => { loadUsers(); loadStats(); showToast('操作成功', 'success') }}
        />
      )}

      {showCreateModal && (
        <CreateUserModal
          onClose={() => setShowCreateModal(false)}
          onCreated={() => { loadUsers(); loadStats(); showToast('用户创建成功', 'success') }}
        />
      )}

      {orgModal.open && (
        <OrgFormModal
          mode={orgModal.mode} type={orgModal.type} initial={orgModal.initial}
          regions={regions}
          onClose={() => setOrgModal(p => ({ ...p, open: false }))}
          onSaved={() => {
            showToast(orgModal.mode === 'create' ? '创建成功' : '更新成功', 'success')
            if (orgModal.type === 'region') loadRegions()
            else if (selRegion) loadSchools2(selRegion.id)
            loadStats()
          }}
        />
      )}

      {groupModal.open && selSchool && (
        <GroupFormModal
          mode={groupModal.mode} schoolId={selSchool.id} schoolName={selSchool.name}
          initial={groupModal.initial}
          onClose={() => setGroupModal(p => ({ ...p, open: false }))}
          onSaved={() => {
            showToast(groupModal.mode === 'create' ? '创建成功' : '更新成功', 'success')
            loadGroups2(selSchool.id); loadStats()
          }}
        />
      )}

      {/* ---- 顶部导航 ---- */}
      <header style={{ height: '64px', position: 'sticky', top: 0, zIndex: 100, background: 'rgba(255,255,255,0.88)', backdropFilter: 'blur(20px)', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', padding: '0 32px', gap: '16px' }}>
        <button onClick={() => navigate(fromPath)}
          style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.white }}>
          {'<- 返回'}
        </button>
        <div style={{ flex: 1, textAlign: 'center' }}>
          <h1 style={{ fontSize: '18px', fontWeight: 700, color: C.text, margin: 0 }}>👥 用户管理中心</h1>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>统一管理用户、组织架构与操作日志</div>
        </div>
        <button onClick={() => setShowCreateModal(true)}
          style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
          + 新建用户
        </button>
      </header>

      <div style={{ maxWidth: '1400px', margin: '0 auto', padding: '24px' }}>

        {/* Tab切换（5个Tab）*/}
        <div style={{ display: 'flex', gap: '4px', marginBottom: '20px', background: C.bg, borderRadius: '12px', padding: '4px', border: `1px solid ${C.border}`, width: 'fit-content' }}>
          {(['overview', 'users', 'orgs', 'logs', 'roles'] as const).map((tab, i) => {
            const labels = ['📊 概览', '👥 用户管理', '🏫 组织架构', '📋 操作日志', '🎭 角色权限']
            return (
              <button key={tab} onClick={() => setActiveTab(tab)}
                style={{ padding: '9px 22px', borderRadius: '9px', border: 'none', cursor: 'pointer', fontSize: '14px', fontWeight: activeTab === tab ? 600 : 400, color: activeTab === tab ? C.primary : C.textSec, background: activeTab === tab ? C.white : 'transparent', boxShadow: activeTab === tab ? '0 1px 4px rgba(0,0,0,0.08)' : 'none', transition: 'all 150ms ease' }}>
                {labels[i]}
              </button>
            )
          })}
        </div>

        {/* ===== Tab: 概览 ===== */}
        {activeTab === 'overview' && (
          <div>
            {statsLoading
              ? <div style={{ textAlign: 'center', padding: '60px', color: C.textMuted }}>加载统计数据...</div>
              : stats ? (
                <>
                  <div style={{ display: 'flex', gap: '16px', marginBottom: '16px' }}>
                    <StatCard label="总用户数"  value={stats.total_users}  sub={`活跃 ${stats.active_users} · 禁用 ${stats.disabled_users}`} color={C.primary} />
                    <StatCard label="组织总数"  value={stats.total_orgs}   sub={`学校 ${stats.total_schools} 所`} color={C.success} />
                    <StatCard label="教研组"    value={stats.total_groups} sub={`教研组成员 ${stats.total_members} 人`} color={C.warning} />
                  </div>
                  <RoleBarChart stats={stats} />
                  <RecentLogsCard
                    logs={recentLogs} loading={recentLogsLoading}
                    onViewAll={() => {
                      setActiveTab('logs')
                      const e = { username: '', action: '', startDate: '', endDate: '' }
                      setLogFilters(e); setLogFilterInput(e); setLogPage(1)
                    }}
                  />
                  <div style={{ background: C.primaryLight, borderRadius: '12px', border: `1px solid ${C.primary}22`, padding: '16px 20px', fontSize: '13px', color: C.primary }}>
                    💡 点击上方"🏫 组织架构"Tab管理区域、学校和教研组，支持完整的创建、编辑、删除和成员管理。
                  </div>
                </>
              ) : null}
          </div>
        )}

        {/* ===== Tab: 用户管理 ===== */}
        {activeTab === 'users' && (
          <div>
            <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, padding: '16px 20px', marginBottom: '16px', display: 'flex', gap: '10px', alignItems: 'center', flexWrap: 'wrap' }}>
              <input value={keywordInput} onChange={e => handleKeywordChange(e.target.value)} placeholder="搜索用户名或显示名..."
                style={{ flex: 1, minWidth: '180px', padding: '8px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none' }}
                onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                onBlur={e => { e.currentTarget.style.borderColor = C.border }}
              />
              <select value={roleFilter} onChange={e => { setRoleFilter(e.target.value); setUserPage(1) }}
                style={{ padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: C.white }}>
                {ROLE_OPTIONS.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
              </select>
              <select value={statusFilter} onChange={e => { setStatusFilter(e.target.value); setUserPage(1) }}
                style={{ padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: C.white }}>
                <option value="">全部状态</option>
                <option value="active">正常</option>
                <option value="disabled">已禁用</option>
              </select>
              <select value={schoolFilter} onChange={e => { setSchoolFilter(e.target.value); setUserPage(1) }}
                style={{ padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: C.white, minWidth: '120px' }}>
                <option value="">全部学校</option>
                {schools.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
              <div style={{ fontSize: '13px', color: C.textMuted }}>共 {userTotal} 个用户</div>
            </div>

            <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
              <div style={{ display: 'grid', gridTemplateColumns: '2fr 1.5fr 1.5fr 1.2fr 1.5fr 1fr', padding: '12px 20px', background: C.bg, borderBottom: `1px solid ${C.border}`, fontSize: '12px', fontWeight: 600, color: C.textSec }}>
                <span>用户</span><span>系统角色</span><span>教案系统归属</span><span>状态</span><span>最近登录</span><span>操作</span>
              </div>
              {userLoading
                ? <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div>
                : users.length === 0
                ? <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>暂无用户</div>
                : users.map((user, idx) => (
                  <div key={user.id}
                    style={{ display: 'grid', gridTemplateColumns: '2fr 1.5fr 1.5fr 1.2fr 1.5fr 1fr', padding: '14px 20px', alignItems: 'center', borderBottom: idx < users.length - 1 ? `1px solid ${C.border}` : 'none', transition: 'background 150ms ease' }}
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
                    <div><RoleBadge role={user.role} roleName={user.role_name} /></div>
                    <div>
                      {user.group_count > 0 ? (
                        <div>
                          <div style={{ fontSize: '13px', color: C.text, fontWeight: 500 }}>{user.group_name || '-'}</div>
                          <div style={{ fontSize: '11px', color: C.textMuted }}>{user.school_name}{user.group_count > 1 ? `等${user.group_count}个组` : ''}</div>
                        </div>
                      ) : (
                        <span style={{ fontSize: '12px', color: C.textMuted }}>未加入教研组</span>
                      )}
                    </div>
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
                ))
              }
            </div>

            {totalPages > 1 && (
              <div style={{ display: 'flex', justifyContent: 'center', gap: '8px', marginTop: '16px', alignItems: 'center' }}>
                <button onClick={() => setUserPage(p => Math.max(1, p - 1))} disabled={userPage === 1}
                  style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: userPage === 1 ? C.textMuted : C.text, cursor: userPage === 1 ? 'not-allowed' : 'pointer' }}>上一页</button>
                <span style={{ fontSize: '13px', color: C.textSec }}>第 {userPage} / {totalPages} 页</span>
                <button onClick={() => setUserPage(p => Math.min(totalPages, p + 1))} disabled={userPage === totalPages}
                  style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: userPage === totalPages ? C.textMuted : C.text, cursor: userPage === totalPages ? 'not-allowed' : 'pointer' }}>下一页</button>
              </div>
            )}
          </div>
        )}

        {/* ===== Tab: 组织架构（三栏递进 + 自动展开）===== */}
        {activeTab === 'orgs' && (
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '16px', alignItems: 'start' }}>

            {/* 区域栏 */}
            <ColCard title="🌍 区域" count={regions.length}
              onAdd={() => setOrgModal({ open: true, mode: 'create', type: 'region' })} addLabel="新建区域"
              loading={regLoading} empty={regions.length === 0 ? '暂无区域，点击右上角新建' : undefined}>
              {regions.map(r => (
                <div key={r.id}>
                  <div onClick={() => handleSelectRegion(r)}
                    style={{ padding: '12px 14px', cursor: 'pointer', background: selRegion?.id === r.id ? C.primaryLight : 'transparent', borderLeft: selRegion?.id === r.id ? `3px solid ${C.primary}` : '3px solid transparent', transition: 'all 150ms ease' }}
                    onMouseEnter={e => { if (selRegion?.id !== r.id) (e.currentTarget as HTMLElement).style.background = C.bg }}
                    onMouseLeave={e => { if (selRegion?.id !== r.id) (e.currentTarget as HTMLElement).style.background = 'transparent' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '4px' }}>
                      <span style={{ fontSize: '14px', fontWeight: 600, color: selRegion?.id === r.id ? C.primary : C.text, flex: 1 }}>{r.name}</span>
                      <StatusBadge status={r.status} />
                    </div>
                    <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '8px' }}>
                      {r.admin_user_name ? `管理员：${r.admin_user_name}` : '暂无管理员'}
                    </div>
                    <div style={{ display: 'flex', gap: '6px' }} onClick={e => e.stopPropagation()}>
                      <button onClick={() => setOrgModal({ open: true, mode: 'edit', type: 'region', initial: r })} style={rowBtn(C.primary, C.primaryLight)}>✏️ 编辑</button>
                      <button onClick={() => handleToggleOrgStatus(r)} style={rowBtn(r.status === 'active' ? C.danger : C.success, r.status === 'active' ? C.dangerLight : C.successLight)}>
                        {r.status === 'active' ? '🚫 禁用' : '✅ 启用'}
                      </button>
                      <button onClick={() => handleDeleteOrg(r)} style={rowBtn(C.danger, C.dangerLight)}>🗑️ 删除</button>
                    </div>
                  </div>
                  <div style={{ height: '1px', background: C.border, margin: '0 14px' }} />
                </div>
              ))}
            </ColCard>

            {/* 学校栏 */}
            <ColCard
              title={selRegion ? <span>🏫 <span style={{ color: C.primary }}>{selRegion.name}</span> 的学校</span> : '🏫 学校'}
              count={schools2.length}
              onAdd={selRegion ? () => setOrgModal({ open: true, mode: 'create', type: 'school' }) : undefined} addLabel="新建学校"
              loading={schLoading}
              empty={!selRegion ? '← 请先选择左侧区域' : schools2.length === 0 ? '暂无学校，点击右上角新建' : undefined}>
              {schools2.map(s => (
                <div key={s.id}>
                  <div onClick={() => handleSelectSchool(s)}
                    style={{ padding: '12px 14px', cursor: 'pointer', background: selSchool?.id === s.id ? C.primaryLight : 'transparent', borderLeft: selSchool?.id === s.id ? `3px solid ${C.primary}` : '3px solid transparent', transition: 'all 150ms ease' }}
                    onMouseEnter={e => { if (selSchool?.id !== s.id) (e.currentTarget as HTMLElement).style.background = C.bg }}
                    onMouseLeave={e => { if (selSchool?.id !== s.id) (e.currentTarget as HTMLElement).style.background = 'transparent' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '4px' }}>
                      <span style={{ fontSize: '14px', fontWeight: 600, color: selSchool?.id === s.id ? C.primary : C.text, flex: 1 }}>{s.name}</span>
                      <StatusBadge status={s.status} />
                    </div>
                    <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '8px' }}>
                      {s.admin_user_name ? `管理员：${s.admin_user_name}` : '暂无管理员'} · {s.group_count} 个教研组
                    </div>
                    <div style={{ display: 'flex', gap: '6px' }} onClick={e => e.stopPropagation()}>
                      <button onClick={() => setOrgModal({ open: true, mode: 'edit', type: 'school', initial: s })} style={rowBtn(C.primary, C.primaryLight)}>✏️ 编辑</button>
                      <button onClick={() => handleToggleOrgStatus(s)} style={rowBtn(s.status === 'active' ? C.danger : C.success, s.status === 'active' ? C.dangerLight : C.successLight)}>
                        {s.status === 'active' ? '🚫 禁用' : '✅ 启用'}
                      </button>
                      <button onClick={() => handleDeleteOrg(s)} style={rowBtn(C.danger, C.dangerLight)}>🗑️ 删除</button>
                    </div>
                  </div>
                  <div style={{ height: '1px', background: C.border, margin: '0 14px' }} />
                </div>
              ))}
            </ColCard>

            {/* 教研组栏 */}
            <ColCard
              title={selSchool ? <span>👨‍🏫 <span style={{ color: C.primary }}>{selSchool.name}</span> 的教研组</span> : '👨‍🏫 教研组'}
              count={groups2.length}
              onAdd={selSchool ? () => setGroupModal({ open: true, mode: 'create' }) : undefined} addLabel="新建教研组"
              loading={grpLoading}
              empty={!selSchool ? '← 请先选择中间学校' : groups2.length === 0 ? '暂无教研组，点击右上角新建' : undefined}>
              {groups2.map(g => (
                <div key={g.id}>
                  <div style={{ padding: '12px 14px' }}>
                    <div style={{ display: 'flex', alignItems: 'flex-start', gap: '6px', marginBottom: '4px' }}>
                      <div style={{ flex: 1 }}>
                        <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{g.name}</div>
                        <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '2px' }}>
                          {g.subject}{g.grade_range ? ` · ${g.grade_range}` : ''} · {g.lead_user_name ? `组长：${g.lead_user_name}` : '暂无组长'} · {g.member_count} 人
                        </div>
                      </div>
                      <StatusBadge status={g.status} />
                    </div>
                    <div style={{ display: 'flex', gap: '6px', marginTop: '8px' }}>
                      <button onClick={() => setGroupModal({ open: true, mode: 'edit', initial: g })} style={rowBtn(C.primary, C.primaryLight)}>✏️ 编辑</button>
                      <button onClick={() => handleDeleteGroup(g)} style={rowBtn(C.danger, C.dangerLight)}>🗑️ 删除</button>
                      <button onClick={() => setExpandedGroupId(p => p === g.id ? null : g.id)}
                        style={rowBtn(expandedGroupId === g.id ? C.purple : C.textSec, expandedGroupId === g.id ? C.purpleLight : C.bg)}>
                        {expandedGroupId === g.id ? '收起 ▲' : '👥 成员'}
                      </button>
                    </div>
                  </div>
                  {expandedGroupId === g.id && (
                    <MemberPanel groupId={g.id} onClose={() => setExpandedGroupId(null)} />
                  )}
                  <div style={{ height: '1px', background: C.border, margin: '0 14px' }} />
                </div>
              ))}
            </ColCard>
          </div>
        )}

        {/* ===== Tab: 操作日志 ===== */}
        {activeTab === 'logs' && (
          <div>
            <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, padding: '16px 20px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', gap: '10px', alignItems: 'center', flexWrap: 'wrap', marginBottom: '12px' }}>
                <input value={logFilterInput.username} onChange={e => setLogFilterInput(p => ({ ...p, username: e.target.value }))}
                  placeholder="搜索用户名 / 显示名..."
                  style={{ ...inputStyle, flex: '1 1 160px', minWidth: '140px' }}
                  onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                  onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                  onKeyDown={e => { if (e.key === 'Enter') handleLogSearch() }}
                />
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <span style={{ fontSize: '12px', color: C.textSec, whiteSpace: 'nowrap' }}>开始</span>
                  <input type="date" value={logFilterInput.startDate} onChange={e => setLogFilterInput(p => ({ ...p, startDate: e.target.value }))}
                    style={{ ...inputStyle, width: '140px' }}
                    onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                    onBlur={e => { e.currentTarget.style.borderColor = C.border }} />
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <span style={{ fontSize: '12px', color: C.textSec, whiteSpace: 'nowrap' }}>结束</span>
                  <input type="date" value={logFilterInput.endDate} onChange={e => setLogFilterInput(p => ({ ...p, endDate: e.target.value }))}
                    style={{ ...inputStyle, width: '140px' }}
                    onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                    onBlur={e => { e.currentTarget.style.borderColor = C.border }} />
                </div>
                <select value={logFilterInput.action} onChange={e => setLogFilterInput(p => ({ ...p, action: e.target.value }))}
                  style={{ ...inputStyle, minWidth: '130px' }}>
                  {ACTION_OPTIONS.map(a => <option key={a.value} value={a.value}>{a.label}</option>)}
                </select>
              </div>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <button onClick={handleLogSearch}
                  style={{ padding: '7px 20px', borderRadius: '8px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
                  🔍 查询
                </button>
                <button onClick={handleLogReset}
                  style={{ padding: '7px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.bg, color: C.textSec, fontSize: '13px', cursor: 'pointer' }}>
                  重置
                </button>
                {(logFilters.username || logFilters.startDate || logFilters.endDate || logFilters.action) && (
                  <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap', alignItems: 'center' }}>
                    {logFilters.username  && <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: C.primaryLight, color: C.primary }}>用户：{logFilters.username}</span>}
                    {logFilters.action    && <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: C.warningLight, color: C.warning }}>{ACTION_OPTIONS.find(a => a.value === logFilters.action)?.label || logFilters.action}</span>}
                    {logFilters.startDate && <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: C.successLight, color: C.success }}>{logFilters.startDate} 起</span>}
                    {logFilters.endDate   && <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: C.successLight, color: C.success }}>至 {logFilters.endDate}</span>}
                  </div>
                )}
                <div style={{ marginLeft: 'auto', fontSize: '13px', color: C.textMuted }}>共 {logTotal} 条记录</div>
              </div>
            </div>

            <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
              <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 2fr 1.2fr 0.9fr 1.3fr 0.6fr', padding: '12px 20px', background: C.bg, borderBottom: `1px solid ${C.border}`, fontSize: '12px', fontWeight: 600, color: C.textSec }}>
                <span>操作者</span><span>操作内容摘要</span><span>操作类型</span><span>IP地址</span><span>时间</span><span>详情</span>
              </div>
              {logLoading
                ? <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div>
                : logs.length === 0
                ? <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>暂无日志记录</div>
                : logs.map((log, idx) => {
                  const isExpanded = expandedLogId === log.id
                  const s = getActionStyle(log.action)
                  return (
                    <div key={log.id} style={{ borderBottom: idx < logs.length - 1 ? `1px solid ${C.border}` : 'none' }}>
                      <div
                        style={{ display: 'grid', gridTemplateColumns: '1.4fr 2fr 1.2fr 0.9fr 1.3fr 0.6fr', padding: '12px 20px', alignItems: 'center', fontSize: '13px', background: isExpanded ? 'rgba(79,123,232,0.03)' : 'transparent', transition: 'background 150ms ease' }}
                        onMouseEnter={e => { if (!isExpanded) (e.currentTarget as HTMLElement).style.background = C.bg }}
                        onMouseLeave={e => { if (!isExpanded) (e.currentTarget as HTMLElement).style.background = 'transparent' }}>
                        <div>
                          <div style={{ fontWeight: 500, color: C.text }}>{log.display_name || log.username}</div>
                          <div style={{ fontSize: '11px', color: C.textMuted }}>@{log.username}</div>
                        </div>
                        <div style={{ color: C.textSec, fontSize: '12px', fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {(() => { try { return Object.entries(JSON.parse(log.detail)).map(([k, v]) => `${k}: ${v}`).join('  ·  ') || '—' } catch { return log.detail || '—' } })()}
                        </div>
                        <div><span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '11px', background: s.bg, color: s.color, fontWeight: 600 }}>{log.action_name}</span></div>
                        <div style={{ color: C.textMuted, fontFamily: 'monospace', fontSize: '11px' }}>{log.ip || '—'}</div>
                        <div style={{ color: C.textSec, fontSize: '11px' }}>{fmt(typeof log.created_at === 'string' ? log.created_at : new Date(log.created_at).toISOString())}</div>
                        <div>
                          <button onClick={() => toggleLogDetail(log.id)}
                            style={{ padding: '4px 10px', borderRadius: '6px', cursor: 'pointer', border: `1px solid ${isExpanded ? C.primary : C.border}`, background: isExpanded ? C.primaryLight : C.bg, color: isExpanded ? C.primary : C.textSec, fontSize: '11px', fontWeight: 600, transition: 'all 150ms ease', whiteSpace: 'nowrap' }}>
                            {isExpanded ? '收起 ▲' : '详情 ▼'}
                          </button>
                        </div>
                      </div>
                      {isExpanded && (
                        <div style={{ padding: '12px 20px 16px', background: 'rgba(79,123,232,0.03)', borderTop: `1px dashed ${C.border}` }}>
                          <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '8px' }}>📄 完整操作详情</div>
                          <pre style={{ margin: 0, padding: '12px 16px', background: C.bg, borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '12px', color: C.text, fontFamily: '"Fira Code","Cascadia Code",Consolas,monospace', lineHeight: 1.6, overflowX: 'auto', whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                            {(() => { try { return JSON.stringify(JSON.parse(log.detail), null, 2) } catch { return log.detail || '（无详情数据）' } })()}
                          </pre>
                        </div>
                      )}
                    </div>
                  )
                })
              }
            </div>

            {logTotalPages > 1 && (
              <div style={{ display: 'flex', justifyContent: 'center', gap: '8px', marginTop: '16px', alignItems: 'center' }}>
                <button onClick={() => setLogPage(p => Math.max(1, p - 1))} disabled={logPage === 1}
                  style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: logPage === 1 ? C.textMuted : C.text, cursor: logPage === 1 ? 'not-allowed' : 'pointer' }}>上一页</button>
                <span style={{ fontSize: '13px', color: C.textSec }}>第 {logPage} / {logTotalPages} 页</span>
                <button onClick={() => setLogPage(p => Math.min(logTotalPages, p + 1))} disabled={logPage === logTotalPages}
                  style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: logPage === logTotalPages ? C.textMuted : C.text, cursor: logPage === logTotalPages ? 'not-allowed' : 'pointer' }}>下一页</button>
              </div>
            )}
          </div>
        )}

        {/* ===== Tab: 角色权限 ===== */}
        {activeTab === 'roles' && (
          <RolesTab showToast={showToast} />
        )}

      </div>
    </div>
  )
}
