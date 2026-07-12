/**
 * UserDetailModal.tsx — 用户详情弹窗（双Tab：基本信息 / 操作记录）
 *
 * 归属治理批B新增（2026-07-04）：
 *   - 新增「所属学校」区块（置于"教案系统归属"之前）：数据源为详情接口新返回的
 *     schools 数组（school_members = 用户归属的唯一事实源），逐校展示
 *     校名/入校来源/该校教研组数/入校时间——根治"前端显示没学校、后端说有学校"
 *     的显示源错位（此前所有归属显示都从教研组反推，退光组后校籍隐形）。
 *   - 每条校籍带红色「移出本校」按钮（canManageTarget 才显示）：
 *     确认弹窗写明将【连带退出该校 N 个教研组】（R3：退校⇒强制退光该校组，
 *     后端单事务保证）；成功后重拉整份详情（校籍与教研组两个列表都会变），
 *     toast 后端拼好的中文 message。senior 对非本校校籍点击会被后端 403，
 *     错误在弹窗内红字展示。
 *   - 归属规则提示：退组不退校（教研组区块的"移除"只退组）；想让用户彻底
 *     离开某校，用本区块的「移出本校」。
 *
 * v52任务六升级：教案归属区块支持切换角色/移除/添加到教研组。
 *
 * 历史修复（bug：管理员无法修改用户的显示名称/系统角色）：
 *   - 新增「资料与角色」编辑区块：显示名称 + 系统角色，走 updateAdminUser（PUT /admin/users/{id}）。
 *     后端接口/前端 API 封装此前均已存在，缺的只是弹窗里的编辑 UI，本次补齐。
 *   - 权限口径（与后端 service 层守卫对齐）：
 *       admin           → 可编辑任意用户，角色下拉全部 6 角色；不能修改自己的角色
 *       senior_operator → 仅可编辑本校"骨干教师/普通教师"目标，角色下拉仅 operator/viewer
 *       region_admin    → 只读提示（管理中心本就只读）
 *   - 「账户操作」（重置密码/启禁用）同步收敛到同一可管性判定 canManageTarget：
 *     堵住学校管理员对同级/上级账号（如混入 school_members 的 admin 账号）操作的 UI 入口，
 *     后端 service 层有对应守卫兜底，前端只是不给入口。
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getAdminUserDetail, getAdminAuditLogs,
  updateAdminUser, updateAdminUserStatus, resetAdminUserPassword,
  updateAdminGroupMemberRole, getAdminOrgs, getAdminGroups,
  addUserToGroup, removeUserFromGroup, removeUserFromSchool,
} from '@/api/admin'
import type {
  AdminUserDetail, AdminGroupMembership, AdminSchoolMembership, AuditLogItem,
  OrgListItem, GroupListItem,
} from '@/api/admin'
import { useAuth } from '@/store/auth'
import { C, ROLE_OPTIONS, APPOINTMENT_ONLY_ROLES, fmt, getActionStyle } from './adminConstants'
import { RoleBadge, StatusBadge } from './adminShared'

interface UserDetailModalProps {
  userId: string
  onClose: () => void
  onAction: () => void   // 操作成功后回调（刷新用户列表等）
}

export function UserDetailModal({ userId, onClose, onAction }: UserDetailModalProps) {
  const [detail, setDetail]     = useState<AdminUserDetail | null>(null)
  const [loading, setLoading]   = useState(true)
  const [resetPwd, setResetPwd] = useState('')
  const [saving, setSaving]     = useState(false)

  // 双Tab状态
  const [detailTab, setDetailTab]     = useState<'info' | 'logs'>('info')
  const [userLogs, setUserLogs]       = useState<AuditLogItem[]>([])
  const [logsLoading, setLogsLoading] = useState(false)
  const [logsLoaded, setLogsLoaded]   = useState(false)

  // 基本信息编辑（显示名称 + 系统角色）
  const [editName, setEditName]     = useState('')
  const [editRole, setEditRole]     = useState('')
  const [editSaving, setEditSaving] = useState(false)
  const [editError, setEditError]   = useState('')

  // 教案归属：移除二次确认
  const [removeTarget, setRemoveTarget] = useState<AdminGroupMembership | null>(null)
  const [removing, setRemoving]         = useState(false)
  const [removeError, setRemoveError]   = useState('')

  // 批B：移出本校二次确认（与退组确认弹窗同构，但语义是 R3：连带退光该校组）
  const [schoolRemoveTarget, setSchoolRemoveTarget] = useState<AdminSchoolMembership | null>(null)
  const [schoolRemoving, setSchoolRemoving]         = useState(false)
  const [schoolRemoveError, setSchoolRemoveError]   = useState('')
  const [schoolRemoveMsg, setSchoolRemoveMsg]       = useState('')
  // 批C2：目标是本校管理员时，'同时移除任命'勾选（默认勾选，打开弹窗时重置）
  const [schoolRemoveAdmin, setSchoolRemoveAdmin]   = useState(true)

  // 教案归属：切换角色加载态
  const [switchingGroupId, setSwitchingGroupId] = useState<string | null>(null)

  // 添加到教研组面板
  const [addPanelOpen, setAddPanelOpen]         = useState(false)
  const [addSchools, setAddSchools]             = useState<OrgListItem[]>([])
  const [addSchoolsLoaded, setAddSchoolsLoaded] = useState(false)
  const [addSchoolId, setAddSchoolId]           = useState('')
  const [addGroups, setAddGroups]               = useState<GroupListItem[]>([])
  const [addGroupsLoading, setAddGroupsLoading] = useState(false)
  const [addGroupId, setAddGroupId]             = useState('')
  const [addRole, setAddRole]                   = useState('member')
  const [addLoading, setAddLoading]             = useState(false)
  const [addError, setAddError]                 = useState('')

  // ---- 当前登录者视角与"目标是否可管理"判定 ----
  const { user: currentUser } = useAuth()
  const isAdminUser  = currentUser?.role === 'admin'
  const isSeniorUser = currentUser?.role === 'senior_operator'
  const isSelf       = currentUser?.id === userId
  // 可管理目标：admin 恒可；senior 仅可管"骨干/普通教师"级别的目标
  // （防止学校管理员对同级或更高级账号做资料/角色/密码/禁用操作——如混入本校成员表的 admin 账号）
  const canManageTarget = !!detail && (
    isAdminUser || (isSeniorUser && (detail.role === 'operator' || detail.role === 'viewer'))
  )
  // 批C（任命唯一事实源）：学校/区域管理员为任命制身份——升级走「组织架构→🛡️管理员」任命
  // （自动升身份），降级随末任命移除自动发生，故不进编辑下拉。目标当前已是任命制身份时，
  // 把现身份并入选项首位保证下拉正常显示，同时整个下拉禁用（不可改动）。
  const isAppointmentManaged = !!detail && APPOINTMENT_ONLY_ROLES.includes(detail.role)
  const editRoleOptions = isAdminUser
    ? [
        ...(detail && isAppointmentManaged ? [{ value: detail.role, label: detail.role_name }] : []),
        ...ROLE_OPTIONS.filter(r => r.value !== '' && !APPOINTMENT_ONLY_ROLES.includes(r.value)),
      ]
    : ROLE_OPTIONS.filter(r => r.value === 'operator' || r.value === 'viewer')
  // 是否有未保存的变更（无变更时保存按钮置灰）
  const basicDirty = !!detail && (editName.trim() !== detail.display_name || editRole !== detail.role)

  // 加载用户详情
  const loadDetail = useCallback(async () => {
    try {
      const d = await getAdminUserDetail(userId)
      setDetail(d)
      // 预填基本信息编辑表单
      setEditName(d.display_name)
      setEditRole(d.role)
    } catch { /* 忽略 */ } finally { setLoading(false) }
  }, [userId])

  useEffect(() => { loadDetail() }, [loadDetail])

  // 懒加载操作记录（切到logs Tab时加载一次）
  useEffect(() => {
    if (detailTab !== 'logs' || logsLoaded) return
    setLogsLoading(true)
    getAdminAuditLogs({ user_id: userId, page: 1, page_size: 20 })
      .then(d => { setUserLogs(d.logs); setLogsLoaded(true) })
      .catch(() => { setLogsLoaded(true) })
      .finally(() => setLogsLoading(false))
  }, [detailTab, userId, logsLoaded])

  // 保存基本信息（显示名称 + 系统角色）
  const handleSaveBasic = async () => {
    if (!detail) return
    const name = editName.trim()
    if (!name) { setEditError('显示名称不能为空'); return }
    setEditSaving(true); setEditError('')
    try {
      await updateAdminUser(userId, { display_name: name, role: editRole })
      const newRoleName = ROLE_OPTIONS.find(r => r.value === editRole)?.label || editRole
      // 本地同步详情，避免整弹窗重拉；父级 onAction 会刷新用户列表并弹"操作成功"
      setDetail(p => p ? { ...p, display_name: name, role: editRole, role_name: newRoleName } : p)
      setEditName(name)
      onAction()
    } catch (e: unknown) {
      setEditError(e instanceof Error ? e.message : '保存失败')
    } finally { setEditSaving(false) }
  }

  // 重置密码
  const handleReset = async () => {
    if (resetPwd.length < 6) return
    try {
      setSaving(true)
      await resetAdminUserPassword(userId, resetPwd)
      setResetPwd(''); onAction()
    } catch { /* 忽略 */ } finally { setSaving(false) }
  }

  // 启用/禁用账户
  const handleToggle = async () => {
    if (!detail) return
    const newStatus = detail.status === 'active' ? 'disabled' : 'active'
    try {
      setSaving(true)
      await updateAdminUserStatus(userId, newStatus)
      setDetail(p => p ? { ...p, status: newStatus } : p)
      onAction()
    } catch { /* 忽略 */ } finally { setSaving(false) }
  }

  // 切换教研组角色：member→backbone→lead 循环，支持多组长
  const handleSwitchRole = async (g: AdminGroupMembership) => {
    const newRole = g.role === 'lead' ? 'member' : g.role === 'backbone' ? 'lead' : 'backbone'
    setSwitchingGroupId(g.group_id)
    try {
      await updateAdminGroupMemberRole(g.group_id, userId, newRole)
      // 乐观更新，避免重新请求整个详情
      setDetail(prev => prev ? {
        ...prev,
        teaching_groups: prev.teaching_groups.map(tg =>
          tg.group_id === g.group_id
            ? { ...tg, role: newRole, role_name: newRole === 'lead' ? '教研组长' : newRole === 'backbone' ? '骨干教师' : '普通成员' }
            : tg
        ),
      } : prev)
    } catch { /* 忽略 */ } finally { setSwitchingGroupId(null) }
  }

  // 确认移出教研组（R2：只退组，不碰校籍——校籍区块的组数会随详情重拉刷新，此处乐观更新组列表即可）
  const doRemoveFromGroup = async () => {
    if (!removeTarget) return
    setRemoving(true); setRemoveError('')
    try {
      await removeUserFromGroup(userId, removeTarget.group_id)
      // 组列表乐观更新 + 重拉详情同步校籍区块的"该校组数"（轻请求，保证两区块一致）
      await loadDetail()
      setRemoveTarget(null)
      onAction()
    } catch (e: unknown) {
      setRemoveError(e instanceof Error ? e.message : '移除失败')
    } finally { setRemoving(false) }
  }

  // 批B：确认移出本校（R3：后端单事务退光该校组+删校籍；成功后重拉整份详情）
  const doRemoveFromSchool = async () => {
    if (!schoolRemoveTarget) return
    setSchoolRemoving(true); setSchoolRemoveError('')
    try {
      const msg = await removeUserFromSchool(userId, schoolRemoveTarget.school_id, schoolRemoveTarget.is_school_admin && schoolRemoveAdmin)
      await loadDetail()   // 校籍与教研组两个列表都变了，整份重拉最稳
      setSchoolRemoveTarget(null)
      setSchoolRemoveMsg(msg)
      onAction()
    } catch (e: unknown) {
      setSchoolRemoveError(e instanceof Error ? e.message : '移出失败')
    } finally { setSchoolRemoving(false) }
  }

  // 打开添加面板，懒加载学校列表
  const openAddPanel = async () => {
    setAddPanelOpen(true); setAddError('')
    if (addSchoolsLoaded) return
    try {
      const orgs = await getAdminOrgs({ type: 'school' })
      setAddSchools(orgs); setAddSchoolsLoaded(true)
    } catch { /* 忽略 */ }
  }

  // 选择学校后联动加载教研组
  const handleAddSchoolChange = async (schoolId: string) => {
    setAddSchoolId(schoolId); setAddGroupId(''); setAddGroups([])
    if (!schoolId) return
    setAddGroupsLoading(true)
    try { setAddGroups(await getAdminGroups(schoolId)) }
    catch { /* 忽略 */ } finally { setAddGroupsLoading(false) }
  }

  // 确认加入教研组（R1：后端自动入校，重拉详情后校籍区块会新增该校）
  const handleAddToGroup = async () => {
    if (!addGroupId) { setAddError('请选择教研组'); return }
    setAddLoading(true); setAddError('')
    try {
      await addUserToGroup(userId, { group_id: addGroupId, role: addRole })
      const newDetail = await getAdminUserDetail(userId)
      setDetail(newDetail)
      setAddPanelOpen(false)
      setAddSchoolId(''); setAddGroupId(''); setAddRole('member')
      onAction()
    } catch (e: unknown) {
      setAddError(e instanceof Error ? e.message : '添加失败，可能该用户已在此教研组中')
    } finally { setAddLoading(false) }
  }

  // 归属记录角色标签样式
  const getMRLabel = (role: string, isLead: boolean) => {
    if (role === 'lead' || isLead) return { text: '教研组长', bg: C.warningLight, color: C.warning }
    if (role === 'backbone') return { text: '骨干教师', bg: C.purpleLight,  color: C.purple  }
    return                          { text: '普通成员', bg: C.bg,           color: C.textSec }
  }

  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 10000,
        background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '24px',
      }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>

      {/* ---- 移出教研组二次确认内嵌弹窗（R2：只退组不退校）---- */}
      {removeTarget && (
        <div style={{
          position: 'fixed', inset: 0, zIndex: 10200,
          background: 'rgba(0,0,0,0.5)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <div style={{ background: C.white, borderRadius: '16px', width: '360px', padding: '24px', boxShadow: '0 20px 60px rgba(0,0,0,0.25)' }}>
            <div style={{ fontSize: '15px', fontWeight: 700, color: C.text, marginBottom: '10px' }}>确认移出教研组</div>
            <div style={{ fontSize: '13px', color: C.textSec, lineHeight: 1.6, marginBottom: '20px' }}>
              确认将该用户从「{removeTarget.group_name}」移出？此操作可重新添加。
              <br /><span style={{ fontSize: '12px', color: C.textMuted }}>提示：退出教研组不会移出学校（校籍保留）；如需彻底移出学校，请使用「所属学校」区块的移出功能。</span>
            </div>
            {removeError && <div style={{ fontSize: '12px', color: C.danger, marginBottom: '10px' }}>{removeError}</div>}
            <div style={{ display: 'flex', gap: '10px' }}>
              <button
                onClick={() => { setRemoveTarget(null); setRemoveError('') }} disabled={removing}
                style={{ flex: 1, padding: '9px', borderRadius: '9px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '13px', color: C.textSec, cursor: removing ? 'not-allowed' : 'pointer' }}>
                取消
              </button>
              <button
                onClick={doRemoveFromGroup} disabled={removing}
                style={{ flex: 1, padding: '9px', borderRadius: '9px', border: 'none', background: removing ? C.textMuted : C.danger, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: removing ? 'not-allowed' : 'pointer' }}>
                {removing ? '移除中...' : '确认移出'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ---- 批B：移出本校二次确认内嵌弹窗（R3：连带退光该校组）---- */}
      {schoolRemoveTarget && (
        <div style={{
          position: 'fixed', inset: 0, zIndex: 10200,
          background: 'rgba(0,0,0,0.5)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <div style={{ background: C.white, borderRadius: '16px', width: '400px', padding: '24px', boxShadow: '0 20px 60px rgba(0,0,0,0.25)', border: `2px solid ${C.danger}22` }}>
            <div style={{ fontSize: '15px', fontWeight: 700, color: C.danger, marginBottom: '10px' }}>⚠️ 确认移出本校</div>
            <div style={{ fontSize: '13px', color: C.textSec, lineHeight: 1.7, marginBottom: '20px' }}>
              确认将该用户从「<b>{schoolRemoveTarget.school_name}</b>」移出？
              {schoolRemoveTarget.group_count > 0 ? (
                <><br />将<b style={{ color: C.danger }}>同时退出该校 {schoolRemoveTarget.group_count} 个教研组</b>，并删除其校籍记录。</>
              ) : (
                <><br />将删除其校籍记录（该用户在此学校未加入任何教研组）。</>
              )}
              <br />移出后该用户将不再看到此学校的共享教案与课件；可通过重新加组或批量导入恢复归属。
            </div>
            {/* 批C2：目标是本校管理员时提供连带移除任命勾选 */}
            {schoolRemoveTarget.is_school_admin && (
              <label style={{ display: 'flex', alignItems: 'flex-start', gap: '8px', padding: '10px 12px', borderRadius: '8px', background: C.warningLight, marginBottom: '14px', cursor: 'pointer' }}>
                <input type="checkbox" checked={schoolRemoveAdmin} onChange={e => setSchoolRemoveAdmin(e.target.checked)} style={{ marginTop: '2px' }} />
                <span style={{ fontSize: '12px', color: C.warning, lineHeight: 1.6 }}>
                  该用户是<b>本校管理员</b>。勾选此项将同时移除其管理员任命（若这是其末个任命，系统身份将自动降级为骨干教师）；不勾选则保留其对本校的管理权（派驻场景）。
                </span>
              </label>
            )}
            {schoolRemoveError && <div style={{ fontSize: '12px', color: C.danger, marginBottom: '10px', padding: '8px 10px', borderRadius: '6px', background: C.dangerLight }}>{schoolRemoveError}</div>}
            <div style={{ display: 'flex', gap: '10px' }}>
              <button
                onClick={() => { setSchoolRemoveTarget(null); setSchoolRemoveError('') }} disabled={schoolRemoving}
                style={{ flex: 1, padding: '9px', borderRadius: '9px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '13px', color: C.textSec, cursor: schoolRemoving ? 'not-allowed' : 'pointer' }}>
                取消
              </button>
              <button
                onClick={doRemoveFromSchool} disabled={schoolRemoving}
                style={{ flex: 1, padding: '9px', borderRadius: '9px', border: 'none', background: schoolRemoving ? C.textMuted : C.danger, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: schoolRemoving ? 'not-allowed' : 'pointer' }}>
                {schoolRemoving ? '移出中...' : '确认移出本校'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ---- 主弹窗 ---- */}
      <div style={{
        background: C.white, borderRadius: '20px', width: '700px', maxHeight: '90vh',
        overflow: 'hidden', display: 'flex', flexDirection: 'column',
        boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }}>
        {/* 头部：用户信息 + Tab切换 */}
        <div style={{ padding: '20px 24px', borderBottom: `1px solid ${C.border}` }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: (!loading && detail) ? '16px' : 0 }}>
            <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>用户详情</div>
            <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
          </div>
          {!loading && detail && (
            <>
              <div style={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
                <div style={{ width: '48px', height: '48px', borderRadius: '50%', flexShrink: 0, background: 'linear-gradient(135deg,#4F7BE8,#7C3AED)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '18px', color: '#fff', fontWeight: 700 }}>
                  {detail.display_name?.charAt(0)?.toUpperCase() || 'U'}
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>{detail.display_name}</div>
                  <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px' }}>@{detail.username}</div>
                  <div style={{ display: 'flex', gap: '6px', marginTop: '6px' }}>
                    <RoleBadge role={detail.role} roleName={detail.role_name} />
                    <StatusBadge status={detail.status} />
                  </div>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <div style={{ fontSize: '22px', fontWeight: 700, color: C.primary }}>{detail.login_count}</div>
                  <div style={{ fontSize: '11px', color: C.textMuted }}>累计登录</div>
                </div>
              </div>
              {/* Tab切换 */}
              <div style={{ display: 'flex', gap: '4px', marginTop: '16px', background: C.bg, borderRadius: '10px', padding: '3px', border: `1px solid ${C.border}`, width: 'fit-content' }}>
                {(['info', 'logs'] as const).map(tab => (
                  <button key={tab} onClick={() => setDetailTab(tab)} style={{ padding: '6px 18px', borderRadius: '8px', border: 'none', cursor: 'pointer', fontSize: '13px', fontWeight: detailTab === tab ? 600 : 400, color: detailTab === tab ? C.primary : C.textSec, background: detailTab === tab ? C.white : 'transparent', boxShadow: detailTab === tab ? '0 1px 4px rgba(0,0,0,0.08)' : 'none', transition: 'all 150ms ease' }}>
                    {tab === 'info' ? '📋 基本信息' : '📄 操作记录'}
                  </button>
                ))}
              </div>
            </>
          )}
        </div>

        {/* 内容区 */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '20px 24px' }}>
          {loading && <div style={{ textAlign: 'center', padding: '40px', color: C.textMuted }}>加载中...</div>}
          {!loading && !detail && <div style={{ textAlign: 'center', padding: '40px', color: C.danger }}>加载失败</div>}

          {/* ===== Tab：基本信息 ===== */}
          {!loading && detail && detailTab === 'info' && (
            <>
              {/* 账户信息 */}
              <div style={{ marginBottom: '20px' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '10px', paddingBottom: '6px', borderBottom: `1px solid ${C.border}` }}>账户信息</div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
                  {[
                    { l: '注册时间', v: fmt(detail.created_at) },
                    { l: '最近登录', v: detail.last_login_at ? fmt(detail.last_login_at) : '暂无' },
                  ].map(i => (
                    <div key={i.l} style={{ padding: '10px 14px', borderRadius: '8px', background: C.bg }}>
                      <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '3px' }}>{i.l}</div>
                      <div style={{ fontSize: '13px', color: C.text, fontWeight: 500 }}>{i.v}</div>
                    </div>
                  ))}
                </div>
              </div>

              {/* 资料与角色 */}
              <div style={{ marginBottom: '20px' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '10px', paddingBottom: '6px', borderBottom: `1px solid ${C.border}` }}>资料与角色</div>
                {canManageTarget ? (
                  <div style={{ padding: '14px', borderRadius: '12px', border: `1px solid ${C.border}`, background: C.bg }}>
                    {editError && (
                      <div style={{ fontSize: '12px', color: C.danger, marginBottom: '10px', padding: '8px 10px', borderRadius: '6px', background: C.dangerLight }}>{editError}</div>
                    )}
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px', marginBottom: '10px' }}>
                      <div>
                        <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '5px' }}>显示名称</label>
                        <input
                          value={editName}
                          onChange={e => setEditName(e.target.value)}
                          placeholder="用户显示名称"
                          style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', background: C.white, boxSizing: 'border-box' }}
                          onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                          onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                        />
                      </div>
                      <div>
                        <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '5px' }}>
                          系统角色{isSelf ? <span style={{ color: C.textMuted, fontWeight: 400 }}>（不能修改自己的角色）</span> : isAppointmentManaged ? <span style={{ color: C.textMuted, fontWeight: 400 }}>（任命制身份：随任命/移除自动升降，不可直接修改）</span> : null}
                        </label>
                        <select
                          value={editRole}
                          disabled={isSelf || isAppointmentManaged}
                          onChange={e => setEditRole(e.target.value)}
                          style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', background: isSelf ? C.bg : C.white, color: C.text, cursor: isSelf ? 'not-allowed' : 'pointer', boxSizing: 'border-box' }}>
                          {editRoleOptions.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
                        </select>
                      </div>
                    </div>
                    {/* 角色变更提示：区域/学校级角色仅决定账号级别，管辖范围需另行配置 */}
                    {isAdminUser && editRole !== detail.role &&
                      ['region_admin', 'senior_operator', 'district_inspector'].includes(editRole) && (
                      <div style={{ fontSize: '12px', color: C.warning, marginBottom: '10px', padding: '8px 10px', borderRadius: '6px', background: C.warningLight, lineHeight: 1.5 }}>
                        提示：该角色仅决定账号级别；具体管辖范围（区域管辖 / 学校管理员任命 / 抽查分配）需在「组织架构」等对应管理面板中另行配置。
                      </div>
                    )}
                    <button
                      onClick={handleSaveBasic}
                      disabled={editSaving || !basicDirty || !editName.trim()}
                      style={{ padding: '8px 20px', borderRadius: '8px', border: 'none', background: (editSaving || !basicDirty || !editName.trim()) ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: (editSaving || !basicDirty || !editName.trim()) ? 'not-allowed' : 'pointer' }}>
                      {editSaving ? '保存中...' : '💾 保存修改'}
                    </button>
                  </div>
                ) : (
                  <div style={{ padding: '12px 14px', borderRadius: '10px', background: C.bg, border: `1px dashed ${C.border}`, fontSize: '12px', color: C.textMuted }}>
                    该用户的资料与角色仅系统管理员可修改
                  </div>
                )}
              </div>

              {/* 课件审核权限 */}
              <div style={{ marginBottom: '20px' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '10px', paddingBottom: '6px', borderBottom: `1px solid ${C.border}` }}>课件审核权限</div>
                {detail.course_assignments.length === 0 ? (
                  <div style={{ fontSize: '13px', color: C.textMuted }}>未分配课程</div>
                ) : (
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                    {detail.course_assignments.map(a => (
                      <span key={a.course_code} style={{ padding: '4px 10px', borderRadius: '6px', background: C.primaryLight, color: C.primary, fontSize: '12px', fontFamily: 'monospace' }}>
                        {a.course_code}{a.course_name !== a.course_code ? ` · ${a.course_name}` : ''}
                      </span>
                    ))}
                  </div>
                )}
              </div>

              {/* 所属学校（批B新增：数据源 school_members = 归属唯一事实源）*/}
              <div style={{ marginBottom: '20px' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '10px', paddingBottom: '6px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <span>所属学校</span>
                  <span style={{ fontSize: '11px', color: C.textMuted, fontWeight: 400 }}>共 {detail.schools.length} 所（决定其共享库可见范围）</span>
                </div>

                {schoolRemoveMsg && (
                  <div style={{ fontSize: '12px', color: C.success, marginBottom: '10px', padding: '8px 10px', borderRadius: '6px', background: C.successLight }}>✅ {schoolRemoveMsg}</div>
                )}

                {detail.schools.length === 0 ? (
                  <div style={{ padding: '16px', borderRadius: '10px', background: C.bg, border: `1px dashed ${C.border}`, textAlign: 'center', fontSize: '13px', color: C.textMuted, marginBottom: '4px' }}>
                    未加入任何学校（各共享库仅可见本人内容与系统级资源）
                  </div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                    {/* 表头 */}
                    <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 1.1fr 0.7fr 0.9fr auto', padding: '4px 12px', fontSize: '11px', fontWeight: 600, color: C.textMuted, gap: '8px' }}>
                      <span>学校</span><span>入校来源</span><span>教研组</span><span>入校时间</span><span style={{ minWidth: '86px' }}>操作</span>
                    </div>
                    {detail.schools.map(s => (
                      <div key={s.school_id} style={{ display: 'grid', gridTemplateColumns: '1.4fr 1.1fr 0.7fr 0.9fr auto', padding: '10px 12px', borderRadius: '10px', background: C.bg, border: `1px solid ${C.border}`, alignItems: 'center', gap: '8px' }}>
                        <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>🏫 {s.school_name}{s.is_school_admin && <span style={{ marginLeft: '6px', padding: '1px 6px', borderRadius: '5px', fontSize: '10px', fontWeight: 600, background: C.warningLight, color: C.warning }}>🛡️ 本校管理员</span>}</div>
                        <div style={{ fontSize: '11px', color: C.textSec, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{s.source_name || '—'}</div>
                        <div style={{ fontSize: '12px', color: s.group_count > 0 ? C.text : C.textMuted }}>
                          {s.group_count > 0 ? `${s.group_count} 个组` : '未入组'}
                        </div>
                        <div style={{ fontSize: '11px', color: C.textMuted, whiteSpace: 'nowrap' }}>{s.joined_at ? fmt(s.joined_at) : '—'}</div>
                        <div style={{ minWidth: '86px', flexShrink: 0 }}>
                          {canManageTarget ? (
                            <button
                              onClick={() => { setSchoolRemoveMsg(''); setSchoolRemoveAdmin(true); setSchoolRemoveTarget(s) }}
                              title="彻底移出该学校（将连带退出该校全部教研组）"
                              style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.danger}`, background: C.dangerLight, color: C.danger, fontSize: '11px', cursor: 'pointer', fontWeight: 600, whiteSpace: 'nowrap' }}>
                              🚪 移出本校
                            </button>
                          ) : (
                            <span style={{ fontSize: '11px', color: C.textMuted }}>—</span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
                <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '6px', lineHeight: 1.5 }}>
                  💡 归属规则：加入教研组会自动加入其所在学校；退出教研组<b>不会</b>退出学校；「移出本校」会连带退出该校全部教研组。
                </div>
              </div>

              {/* 教案系统归属（任务六：切换角色/移除/添加）*/}
              <div style={{ marginBottom: '20px' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '10px', paddingBottom: '6px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <span>教案系统归属</span>
                  <span style={{ fontSize: '11px', color: C.textMuted, fontWeight: 400 }}>共 {detail.teaching_groups.length} 个教研组</span>
                </div>

                {detail.teaching_groups.length === 0 ? (
                  <div style={{ padding: '16px', borderRadius: '10px', background: C.bg, border: `1px dashed ${C.border}`, textAlign: 'center', fontSize: '13px', color: C.textMuted, marginBottom: '10px' }}>
                    暂未加入任何教研组
                  </div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', marginBottom: '10px' }}>
                    {/* 表头 */}
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.1fr 0.75fr 0.85fr auto', padding: '4px 12px', fontSize: '11px', fontWeight: 600, color: C.textMuted, gap: '8px' }}>
                      <span>所属学校</span><span>教研组</span><span>成员角色</span><span>加入时间</span><span style={{ minWidth: '110px' }}>操作</span>
                    </div>
                    {/* 归属记录列表 */}
                    {detail.teaching_groups.map(g => {
                      const rl = getMRLabel(g.role, g.is_lead)
                      const isSwitching = switchingGroupId === g.group_id
                      return (
                        <div key={g.group_id} style={{ display: 'grid', gridTemplateColumns: '1fr 1.1fr 0.75fr 0.85fr auto', padding: '10px 12px', borderRadius: '10px', background: C.bg, border: `1px solid ${C.border}`, alignItems: 'center', gap: '8px' }}>
                          <div style={{ fontSize: '12px', color: C.textSec, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>🏫 {g.school_name}</div>
                          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {g.is_lead && <span style={{ color: C.warning, marginRight: '3px' }}>★</span>}{g.group_name}
                          </div>
                          <div>
                            <span style={{ display: 'inline-block', padding: '2px 7px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, background: rl.bg, color: rl.color, border: `1px solid ${rl.color}22`, whiteSpace: 'nowrap' }}>
                              {rl.text}
                            </span>
                          </div>
                          <div style={{ fontSize: '11px', color: C.textMuted, whiteSpace: 'nowrap' }}>{fmt(g.joined_at)}</div>
                          {/* 操作按钮组 */}
                          <div style={{ display: 'flex', gap: '4px', minWidth: '110px', flexShrink: 0 }}>
                            {/* 切换角色：三态循环 member→骨干→组长→member */}
                            {(
                              <button
                                onClick={() => handleSwitchRole(g)} disabled={isSwitching}
                                title={g.role === 'lead' ? '切换为普通成员' : g.role === 'backbone' ? '切换为教研组长' : '切换为骨干教师'}
                                style={{ padding: '3px 7px', borderRadius: '5px', border: `1px solid ${C.purpleLight}`, background: C.purpleLight, color: C.purple, fontSize: '10px', cursor: isSwitching ? 'not-allowed' : 'pointer', fontWeight: 500, whiteSpace: 'nowrap', opacity: isSwitching ? 0.5 : 1 }}>
                                {isSwitching ? '...' : g.role === 'lead' ? '→普通' : g.role === 'backbone' ? '→组长' : '→骨干'}
                              </button>
                            )}
                            {/* 移除（组长灰色禁用）*/}
                            <button
                              onClick={() => !g.is_lead && setRemoveTarget(g)} disabled={false}
                              title='移出该教研组（校籍保留）'
                              style={{ padding: '3px 7px', borderRadius: '5px', border: `1px solid ${g.is_lead ? C.border : C.dangerLight}`, background: C.dangerLight, color: C.danger, fontSize: '10px', cursor: 'pointer', fontWeight: 500, whiteSpace: 'nowrap' }}>
                              移除
                            </button>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )}

                {/* 添加到教研组面板 */}
                {!addPanelOpen ? (
                  <button onClick={openAddPanel} style={{ width: '100%', padding: '9px', borderRadius: '10px', border: `1px dashed ${C.primary}`, background: C.primaryLight, color: C.primary, fontSize: '13px', fontWeight: 600, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px' }}>
                    ＋ 添加到教研组
                  </button>
                ) : (
                  <div style={{ padding: '14px', borderRadius: '12px', border: `1px solid ${C.border}`, background: C.bg }}>
                    <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '12px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <span>➕ 添加到教研组</span>
                      <button onClick={() => { setAddPanelOpen(false); setAddError(''); setAddSchoolId(''); setAddGroupId(''); setAddRole('member') }}
                        style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '16px', color: C.textMuted }}>×</button>
                    </div>
                    {addError && <div style={{ fontSize: '12px', color: C.danger, marginBottom: '10px', padding: '8px 10px', borderRadius: '6px', background: C.dangerLight }}>{addError}</div>}
                    {/* 步骤一：选学校 */}
                    <div style={{ marginBottom: '10px' }}>
                      <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '5px' }}>① 选择学校</label>
                      <select value={addSchoolId} onChange={e => handleAddSchoolChange(e.target.value)} style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', background: C.white, boxSizing: 'border-box' }}>
                        <option value="">请选择学校...</option>
                        {addSchools.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                      </select>
                      {!addSchoolsLoaded && <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px' }}>加载中...</div>}
                    </div>
                    {/* 步骤二：选教研组 */}
                    {addSchoolId && (
                      <div style={{ marginBottom: '10px' }}>
                        <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '5px' }}>② 选择教研组</label>
                        <select value={addGroupId} onChange={e => setAddGroupId(e.target.value)} style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', background: C.white, boxSizing: 'border-box' }}>
                          <option value="">{addGroupsLoading ? '加载中...' : '请选择教研组...'}</option>
                          {addGroups.map(g => <option key={g.id} value={g.id}>{g.name}（{g.subject}{g.grade_range ? `·${g.grade_range}` : ''}）</option>)}
                        </select>
                        {addSchoolId && !addGroupsLoading && addGroups.length === 0 && (
                          <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px' }}>该学校暂无教研组</div>
                        )}
                      </div>
                    )}
                    {/* 步骤三：选角色 + 确认 */}
                    {addGroupId && (
                      <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                        <div style={{ flex: 1 }}>
                          <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '5px' }}>③ 选择角色</label>
                          <select value={addRole} onChange={e => setAddRole(e.target.value)} style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', background: C.white }}>
                            <option value="member">普通成员</option>
                            <option value="backbone">骨干教师</option>
                            <option value="lead">教研组长</option>
                          </select>
                        </div>
                        <div style={{ paddingTop: '18px' }}>
                          <button onClick={handleAddToGroup} disabled={addLoading}
                            style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: addLoading ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: addLoading ? 'not-allowed' : 'pointer', whiteSpace: 'nowrap' }}>
                            {addLoading ? '添加中...' : '✓ 确认添加'}
                          </button>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>

              {/* 账户操作（收敛到可管性判定：admin 恒可；senior 仅骨干/普通教师目标；region_admin 只读不显示） */}
              {canManageTarget && (
                <div style={{ marginBottom: '20px' }}>
                  <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '10px', paddingBottom: '6px', borderBottom: `1px solid ${C.border}` }}>账户操作</div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                    <div style={{ display: 'flex', gap: '8px' }}>
                      <input type="password" value={resetPwd} onChange={e => setResetPwd(e.target.value)} placeholder="输入新密码（至少6位）"
                        style={{ flex: 1, padding: '9px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none' }}
                        onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                        onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                      />
                      <button onClick={handleReset} disabled={resetPwd.length < 6 || saving}
                        style={{ padding: '9px 18px', borderRadius: '8px', border: 'none', background: resetPwd.length >= 6 ? C.primary : C.textMuted, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: resetPwd.length >= 6 ? 'pointer' : 'not-allowed' }}>
                        重置密码
                      </button>
                    </div>
                    <button onClick={handleToggle} disabled={saving}
                      style={{ padding: '9px 18px', borderRadius: '8px', border: `1px solid ${detail.status === 'active' ? C.danger : C.success}`, background: detail.status === 'active' ? C.dangerLight : C.successLight, color: detail.status === 'active' ? C.danger : C.success, fontSize: '14px', fontWeight: 600, cursor: 'pointer', width: '100%' }}>
                      {detail.status === 'active' ? '禁用该账户' : '启用该账户'}
                    </button>
                  </div>
                </div>
              )}
            </>
          )}

          {/* ===== Tab：操作记录 ===== */}
          {!loading && detail && detailTab === 'logs' && (
            <div>
              <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '14px' }}>显示该用户最近 20 条操作记录</div>
              {logsLoading ? (
                <div style={{ textAlign: 'center', padding: '40px', color: C.textMuted }}>加载中...</div>
              ) : userLogs.length === 0 ? (
                <div style={{ padding: '32px', borderRadius: '12px', background: C.bg, border: `1px dashed ${C.border}`, textAlign: 'center', fontSize: '13px', color: C.textMuted }}>暂无操作记录</div>
              ) : (
                userLogs.map((log, idx) => {
                  const s = getActionStyle(log.action)
                  let dd = ''
                  try { dd = Object.entries(JSON.parse(log.detail)).map(([k, v]) => `${k}: ${v}`).join('  ·  ') }
                  catch { dd = log.detail || '' }
                  return (
                    <div key={log.id} style={{ display: 'flex', alignItems: 'flex-start', gap: '12px', padding: '12px 0', borderBottom: idx < userLogs.length - 1 ? `1px solid ${C.border}` : 'none' }}>
                      <div style={{ width: '8px', height: '8px', borderRadius: '50%', marginTop: '5px', flexShrink: 0, background: s.color }} />
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '3px' }}>
                          <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, background: s.bg, color: s.color }}>{log.action_name}</span>
                          {log.ip && <span style={{ fontSize: '11px', color: C.textMuted, fontFamily: 'monospace' }}>{log.ip}</span>}
                        </div>
                        {dd && <div style={{ fontSize: '12px', color: C.textSec, fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{dd}</div>}
                      </div>
                      <div style={{ flexShrink: 0, fontSize: '11px', color: C.textMuted, whiteSpace: 'nowrap' }}>
                        {fmt(typeof log.created_at === 'string' ? log.created_at : new Date(log.created_at).toISOString())}
                      </div>
                    </div>
                  )
                })
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
