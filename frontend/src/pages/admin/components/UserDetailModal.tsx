/**
 * UserDetailModal.tsx — 用户详情弹窗（双Tab：基本信息 / 操作记录）
 *
 * v52任务六升级：教案归属区块支持：
 *   - 切换角色（member ↔ backbone）
 *   - 移除（组长保护+二次确认）
 *   - 添加到教研组（三步：选学校→选组→选角色→确认）
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getAdminUserDetail, getAdminAuditLogs,
  updateAdminUserStatus, resetAdminUserPassword,
  updateAdminGroupMemberRole, getAdminOrgs, getAdminGroups,
  addUserToGroup, removeUserFromGroup,
} from '@/api/admin'
import type {
  AdminUserDetail, AdminGroupMembership, AuditLogItem,
  OrgListItem, GroupListItem,
} from '@/api/admin'
import { C, fmt, getActionStyle } from './adminConstants'
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
  const [detailTab, setDetailTab]   = useState<'info' | 'logs'>('info')
  const [userLogs, setUserLogs]     = useState<AuditLogItem[]>([])
  const [logsLoading, setLogsLoading] = useState(false)
  const [logsLoaded, setLogsLoaded]   = useState(false)

  // 教案归属：移除二次确认
  const [removeTarget, setRemoveTarget] = useState<AdminGroupMembership | null>(null)
  const [removing, setRemoving]         = useState(false)
  const [removeError, setRemoveError]   = useState('')

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

  // 加载用户详情
  const loadDetail = useCallback(async () => {
    try {
      const d = await getAdminUserDetail(userId)
      setDetail(d)
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

  // 确认移出教研组
  const doRemoveFromGroup = async () => {
    if (!removeTarget) return
    setRemoving(true); setRemoveError('')
    try {
      await removeUserFromGroup(userId, removeTarget.group_id)
      setDetail(prev => prev ? {
        ...prev,
        teaching_groups: prev.teaching_groups.filter(tg => tg.group_id !== removeTarget.group_id),
      } : prev)
      setRemoveTarget(null)
      onAction()
    } catch (e: unknown) {
      setRemoveError(e instanceof Error ? e.message : '移除失败')
    } finally { setRemoving(false) }
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

  // 确认加入教研组
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

      {/* ---- 移除二次确认内嵌弹窗 ---- */}
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
                              title='移出该教研组'
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

              {/* 账户操作 */}
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
