import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  addSchoolGroupMember,
  createSchoolGroup,
  createSchoolUser,
  deleteSchoolGroup,
  getMySchool,
  getSchoolGroupMembers,
  getSchoolGroups,
  getSchoolUsers,
  removeSchoolGroupMember,
  resetSchoolUserPassword,
  updateSchoolGroup,
  updateSchoolGroupMemberRole,
  updateSchoolUser,
  updateSchoolUserStatus,
  type SchoolGroupListItem,
  type SchoolGroupMemberItem,
  type SchoolInfo,
  type SchoolUserItem,
} from '@/api/school-admin'

const COLORS = {
  primary: '#4F7BE8',
  success: '#10B981',
  danger: '#EF4444',
  warning: '#F59E0B',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  bg: '#F9FAFB',
  white: '#FFFFFF',
}

function SectionCard({ title, children, extra }: { title: string; children: React.ReactNode; extra?: React.ReactNode }) {
  return (
    <div style={{
      background: COLORS.white,
      borderRadius: '16px',
      border: `1px solid ${COLORS.border}`,
      padding: '20px 22px',
      marginBottom: '16px',
      boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '14px' }}>
        <h3 style={{ margin: 0, fontSize: '16px', color: COLORS.text }}>{title}</h3>
        {extra}
      </div>
      {children}
    </div>
  )
}

function Pill({ text, color = '#4F7BE8', bg = 'rgba(79,123,232,0.1)' }: { text: string; color?: string; bg?: string }) {
  return (
    <span style={{
      display: 'inline-block',
      padding: '3px 10px',
      borderRadius: '999px',
      fontSize: '12px',
      color,
      background: bg,
      fontWeight: 600,
    }}>
      {text}
    </span>
  )
}

export default function SchoolAdminPanel() {
  const [loading, setLoading] = useState(true)
  const [school, setSchool] = useState<SchoolInfo | null>(null)
  const [users, setUsers] = useState<SchoolUserItem[]>([])
  const [groups, setGroups] = useState<SchoolGroupListItem[]>([])
  const [members, setMembers] = useState<Record<string, SchoolGroupMemberItem[]>>({})
  const [error, setError] = useState('')

  // 新建教师
  const [newTeacher, setNewTeacher] = useState({
    username: '',
    display_name: '',
    password: '',
    role: 'viewer' as 'operator' | 'viewer',
  })

  // 新建教研组
  const [newGroup, setNewGroup] = useState({
    name: '',
    subject: '',
    grade_range: '',
    description: '',
  })

  // 添加成员临时表单（每组一个）
  const [addMemberForm, setAddMemberForm] = useState<Record<string, { user_id: string; role: 'member' | 'backbone' | 'lead' }>>({})

  const loadData = useCallback(async () => {
    try {
      setLoading(true)
      setError('')
      const [schoolData, userResp, groupResp] = await Promise.all([
        getMySchool(),
        getSchoolUsers(),
        getSchoolGroups(),
      ])
      setSchool(schoolData)
      setUsers(userResp.users ?? [])
      setGroups(groupResp.groups ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载学校管理数据失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { loadData() }, [loadData])

  const loadGroupMembers = useCallback(async (groupId: string) => {
    try {
      const list = await getSchoolGroupMembers(groupId)
      setMembers(prev => ({ ...prev, [groupId]: list }))
    } catch {
      // 静默处理，避免影响整体页面
    }
  }, [])

  const roleName = (r: string) => {
    if (r === 'operator') return '骨干教师'
    if (r === 'viewer') return '普通教师'
    if (r === 'backbone') return '骨干教师'
    if (r === 'lead') return '教研组长'
    if (r === 'member') return '普通成员'
    return r
  }

  const availableUsersByGroup = useMemo(() => {
    const map: Record<string, SchoolUserItem[]> = {}
    for (const g of groups) {
      const currentMemberIds = new Set((members[g.id] ?? []).map(m => m.user_id))
      map[g.id] = users.filter(u => !currentMemberIds.has(u.id))
    }
    return map
  }, [groups, members, users])

  const handleCreateTeacher = async () => {
    if (!newTeacher.username.trim() || !newTeacher.display_name.trim() || !newTeacher.password.trim()) {
      alert('请完整填写教师信息')
      return
    }
    await createSchoolUser({
      username: newTeacher.username.trim(),
      display_name: newTeacher.display_name.trim(),
      password: newTeacher.password,
      role: newTeacher.role,
    })
    setNewTeacher({ username: '', display_name: '', password: '', role: 'viewer' })
    await loadData()
    alert('教师创建成功')
  }

  const handleUpdateTeacher = async (u: SchoolUserItem) => {
    const newName = window.prompt('请输入新显示名称：', u.display_name)
    if (!newName || !newName.trim()) return
    const newRole = window.prompt('请输入角色（operator/viewer）：', u.role)?.trim() as 'operator' | 'viewer' | undefined
    if (!newRole || !['operator', 'viewer'].includes(newRole)) {
      alert('角色必须是 operator 或 viewer')
      return
    }
    await updateSchoolUser(u.id, { display_name: newName.trim(), role: newRole })
    await loadData()
    alert('教师信息更新成功')
  }

  const handleToggleTeacherStatus = async (u: SchoolUserItem) => {
    const nextStatus = u.status === 'active' ? 'disabled' : 'active'
    if (!window.confirm(`确认将用户 ${u.username} 设置为 ${nextStatus === 'active' ? '启用' : '禁用'} 吗？`)) return
    await updateSchoolUserStatus(u.id, nextStatus)
    await loadData()
  }

  const handleResetPassword = async (u: SchoolUserItem) => {
    const pwd = window.prompt(`请输入 ${u.username} 的新密码（至少6位）：`)
    if (!pwd) return
    if (pwd.length < 6) {
      alert('密码长度不能少于6位')
      return
    }
    await resetSchoolUserPassword(u.id, pwd)
    alert('密码重置成功')
  }

  const handleCreateGroup = async () => {
    if (!newGroup.name.trim() || !newGroup.subject.trim()) {
      alert('教研组名称和学科不能为空')
      return
    }
    await createSchoolGroup({
      name: newGroup.name.trim(),
      subject: newGroup.subject.trim(),
      grade_range: newGroup.grade_range.trim(),
      description: newGroup.description.trim(),
    })
    setNewGroup({ name: '', subject: '', grade_range: '', description: '' })
    await loadData()
    alert('教研组创建成功')
  }

  const handleUpdateGroup = async (g: SchoolGroupListItem) => {
    const name = window.prompt('新组名：', g.name)
    if (!name || !name.trim()) return
    const subject = window.prompt('新学科：', g.subject)
    if (!subject || !subject.trim()) return
    const grade = window.prompt('年级范围：', g.grade_range ?? '') ?? ''
    await updateSchoolGroup(g.id, {
      name: name.trim(),
      subject: subject.trim(),
      grade_range: grade.trim(),
      description: '',
    })
    await loadData()
    alert('教研组更新成功')
  }

  const handleDeleteGroup = async (g: SchoolGroupListItem) => {
    if (!window.confirm(`确认删除教研组「${g.name}」吗？`)) return
    await deleteSchoolGroup(g.id)
    await loadData()
  }

  const handleAddMember = async (groupId: string) => {
    const form = addMemberForm[groupId]
    if (!form?.user_id) {
      alert('请选择教师')
      return
    }
    await addSchoolGroupMember(groupId, {
      user_id: form.user_id,
      role: form.role,
    })
    await loadGroupMembers(groupId)
    setAddMemberForm(prev => ({ ...prev, [groupId]: { user_id: '', role: 'member' } }))
    await loadData()
  }

  const handleUpdateMemberRole = async (groupId: string, userId: string, role: 'member' | 'backbone' | 'lead') => {
    await updateSchoolGroupMemberRole(groupId, userId, role)
    await loadGroupMembers(groupId)
    await loadData()
  }

  const handleRemoveMember = async (groupId: string, userId: string) => {
    if (!window.confirm('确认移除该成员吗？')) return
    await removeSchoolGroupMember(groupId, userId)
    await loadGroupMembers(groupId)
    await loadData()
  }

  if (loading) {
    return (
      <div style={{
        background: COLORS.white, borderRadius: '16px', border: `1px solid ${COLORS.border}`,
        padding: '24px', textAlign: 'center', color: COLORS.textMuted,
      }}>
        加载学校管理数据中...
      </div>
    )
  }

  if (error) {
    return (
      <div style={{
        background: 'rgba(239,68,68,0.08)', borderRadius: '12px',
        border: '1px solid rgba(239,68,68,0.2)', padding: '14px',
        color: COLORS.danger, fontSize: '14px',
      }}>
        {error}
      </div>
    )
  }

  return (
    <div>
      {/* 学校信息 */}
      <SectionCard title="学校信息">
        <div style={{ display: 'grid', gridTemplateColumns: '140px 1fr', rowGap: '8px', fontSize: '14px' }}>
          <span style={{ color: COLORS.textSec }}>学校名称</span>
          <span style={{ color: COLORS.text, fontWeight: 600 }}>{school?.name || '-'}</span>
          <span style={{ color: COLORS.textSec }}>组织类型</span>
          <span style={{ color: COLORS.text }}>{school?.type || '-'}</span>
          <span style={{ color: COLORS.textSec }}>状态</span>
          <span>{school?.status === 'active' ? <Pill text="启用" color={COLORS.success} bg="rgba(16,185,129,0.1)" /> : <Pill text="禁用" color={COLORS.danger} bg="rgba(239,68,68,0.1)" />}</span>
        </div>
      </SectionCard>

      {/* 教师管理 */}
      <SectionCard title={`教师管理（${users.length}）`}>
        <div style={{ marginBottom: '14px', padding: '12px', border: `1px solid ${COLORS.border}`, borderRadius: '10px', background: COLORS.bg }}>
          <div style={{ fontSize: '13px', color: COLORS.textSec, marginBottom: '8px' }}>新建教师账号（手动分配到教研组）</div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 140px auto', gap: '8px' }}>
            <input placeholder="用户名" value={newTeacher.username} onChange={e => setNewTeacher(v => ({ ...v, username: e.target.value }))} />
            <input placeholder="显示名称" value={newTeacher.display_name} onChange={e => setNewTeacher(v => ({ ...v, display_name: e.target.value }))} />
            <input placeholder="初始密码(>=6位)" type="password" value={newTeacher.password} onChange={e => setNewTeacher(v => ({ ...v, password: e.target.value }))} />
            <select value={newTeacher.role} onChange={e => setNewTeacher(v => ({ ...v, role: e.target.value as 'operator' | 'viewer' }))}>
              <option value="viewer">普通教师(viewer)</option>
              <option value="operator">骨干教师(operator)</option>
            </select>
            <button onClick={handleCreateTeacher} style={{ background: COLORS.primary, color: '#fff', border: 'none', borderRadius: '8px', padding: '8px 12px', cursor: 'pointer' }}>创建</button>
          </div>
        </div>

        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
            <thead>
              <tr style={{ background: COLORS.bg }}>
                <th style={{ textAlign: 'left', padding: '8px', borderBottom: `1px solid ${COLORS.border}` }}>用户名</th>
                <th style={{ textAlign: 'left', padding: '8px', borderBottom: `1px solid ${COLORS.border}` }}>显示名</th>
                <th style={{ textAlign: 'left', padding: '8px', borderBottom: `1px solid ${COLORS.border}` }}>角色</th>
                <th style={{ textAlign: 'left', padding: '8px', borderBottom: `1px solid ${COLORS.border}` }}>状态</th>
                <th style={{ textAlign: 'left', padding: '8px', borderBottom: `1px solid ${COLORS.border}` }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {users.map(u => (
                <tr key={u.id}>
                  <td style={{ padding: '8px', borderBottom: `1px solid ${COLORS.border}` }}>{u.username}</td>
                  <td style={{ padding: '8px', borderBottom: `1px solid ${COLORS.border}` }}>{u.display_name}</td>
                  <td style={{ padding: '8px', borderBottom: `1px solid ${COLORS.border}` }}>{roleName(u.role)}</td>
                  <td style={{ padding: '8px', borderBottom: `1px solid ${COLORS.border}` }}>
                    {u.status === 'active'
                      ? <Pill text="启用" color={COLORS.success} bg="rgba(16,185,129,0.1)" />
                      : <Pill text="禁用" color={COLORS.danger} bg="rgba(239,68,68,0.1)" />}
                  </td>
                  <td style={{ padding: '8px', borderBottom: `1px solid ${COLORS.border}` }}>
                    <button onClick={() => handleUpdateTeacher(u)} style={{ marginRight: 6 }}>编辑</button>
                    <button onClick={() => handleToggleTeacherStatus(u)} style={{ marginRight: 6 }}>
                      {u.status === 'active' ? '禁用' : '启用'}
                    </button>
                    <button onClick={() => handleResetPassword(u)}>重置密码</button>
                  </td>
                </tr>
              ))}
              {users.length === 0 && (
                <tr>
                  <td colSpan={5} style={{ padding: '12px', textAlign: 'center', color: COLORS.textMuted }}>暂无教师数据</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </SectionCard>

      {/* 教研组管理 */}
      <SectionCard title={`教研组管理（${groups.length}）`}>
        <div style={{ marginBottom: '14px', padding: '12px', border: `1px solid ${COLORS.border}`, borderRadius: '10px', background: COLORS.bg }}>
          <div style={{ fontSize: '13px', color: COLORS.textSec, marginBottom: '8px' }}>新建教研组</div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr auto', gap: '8px' }}>
            <input placeholder="组名" value={newGroup.name} onChange={e => setNewGroup(v => ({ ...v, name: e.target.value }))} />
            <input placeholder="学科" value={newGroup.subject} onChange={e => setNewGroup(v => ({ ...v, subject: e.target.value }))} />
            <input placeholder="年级范围(可选)" value={newGroup.grade_range} onChange={e => setNewGroup(v => ({ ...v, grade_range: e.target.value }))} />
            <input placeholder="描述(可选)" value={newGroup.description} onChange={e => setNewGroup(v => ({ ...v, description: e.target.value }))} />
            <button onClick={handleCreateGroup} style={{ background: COLORS.primary, color: '#fff', border: 'none', borderRadius: '8px', padding: '8px 12px', cursor: 'pointer' }}>创建</button>
          </div>
        </div>

        {groups.map(g => (
          <div key={g.id} style={{ border: `1px solid ${COLORS.border}`, borderRadius: '10px', padding: '12px', marginBottom: '10px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: '10px', alignItems: 'center' }}>
              <div>
                <div style={{ fontSize: '15px', fontWeight: 600, color: COLORS.text }}>{g.name}</div>
                <div style={{ fontSize: '12px', color: COLORS.textSec }}>
                  学科：{g.subject} ｜ 年级：{g.grade_range || '-'} ｜ 成员：{g.member_count}
                </div>
              </div>
              <div>
                <button onClick={() => handleUpdateGroup(g)} style={{ marginRight: 6 }}>编辑</button>
                <button onClick={() => handleDeleteGroup(g)}>删除</button>
              </div>
            </div>

            <div style={{ marginTop: '10px', paddingTop: '10px', borderTop: `1px dashed ${COLORS.border}` }}>
              <div style={{ marginBottom: '8px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ fontSize: '13px', color: COLORS.textSec }}>成员管理</span>
                <button
                  onClick={() => loadGroupMembers(g.id)}
                  style={{ fontSize: '12px' }}
                >刷新成员</button>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 160px auto', gap: '8px', marginBottom: '8px' }}>
                <select
                  value={addMemberForm[g.id]?.user_id || ''}
                  onChange={e => setAddMemberForm(prev => ({
                    ...prev,
                    [g.id]: { user_id: e.target.value, role: prev[g.id]?.role || 'member' },
                  }))}
                >
                  <option value="">选择教师加入该组</option>
                  {(availableUsersByGroup[g.id] || []).map(u => (
                    <option key={u.id} value={u.id}>{u.display_name}（{u.username}）</option>
                  ))}
                </select>

                <select
                  value={addMemberForm[g.id]?.role || 'member'}
                  onChange={e => setAddMemberForm(prev => ({
                    ...prev,
                    [g.id]: { user_id: prev[g.id]?.user_id || '', role: e.target.value as 'member' | 'backbone' | 'lead' },
                  }))}
                >
                  <option value="member">普通成员</option>
                  <option value="backbone">骨干教师</option>
                  <option value="lead">教研组长</option>
                </select>

                <button onClick={() => handleAddMember(g.id)}>添加成员</button>
              </div>

              <div style={{ fontSize: '12px' }}>
                {(members[g.id] || []).length === 0 ? (
                  <span style={{ color: COLORS.textMuted }}>暂无成员（点击刷新成员）</span>
                ) : (
                  (members[g.id] || []).map(m => (
                    <div key={m.user_id} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '6px 0' }}>
                      <span style={{ minWidth: 180 }}>{m.display_name}（{m.username}）</span>
                      <select
                        value={m.role}
                        onChange={e => handleUpdateMemberRole(g.id, m.user_id, e.target.value as 'member' | 'backbone' | 'lead')}
                      >
                        <option value="member">普通成员</option>
                        <option value="backbone">骨干教师</option>
                        <option value="lead">教研组长</option>
                      </select>
                      <button onClick={() => handleRemoveMember(g.id, m.user_id)}>移除</button>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        ))}

        {groups.length === 0 && (
          <div style={{ textAlign: 'center', color: COLORS.textMuted, fontSize: '13px', padding: '10px 0' }}>
            暂无教研组
          </div>
        )}
      </SectionCard>
    </div>
  )
}
