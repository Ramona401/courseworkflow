/**
 * MemberPanel.tsx — 教研组成员管理内嵌展开面板
 * 功能：查看成员列表 / 切换角色 / 移除成员 / 添加成员
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getAdminGroupMembers, addAdminGroupMember,
  updateAdminGroupMemberRole, removeAdminGroupMember,
} from '@/api/admin'
import type { GroupMemberItem } from '@/api/admin'
import { C, fmt } from './adminConstants'
import { UserSearchPicker } from './UserSearchPicker'

interface MemberPanelProps {
  groupId: string
  onClose: () => void
}

export function MemberPanel({ groupId, onClose }: MemberPanelProps) {
  const [members, setMembers]         = useState<GroupMemberItem[]>([])
  const [loading, setLoading]         = useState(true)
  const [addUserId, setAddUserId]     = useState('')
  const [addUserName, setAddUserName] = useState('')
  const [addRole, setAddRole]         = useState('member')
  const [adding, setAdding]           = useState(false)
  const [addError, setAddError]       = useState('')

  // 加载成员列表
  const load = useCallback(async () => {
    try {
      setLoading(true)
      setMembers(await getAdminGroupMembers(groupId))
    } catch { } finally { setLoading(false) }
  }, [groupId])

  useEffect(() => { load() }, [load])

  // 添加成员
  const handleAdd = async () => {
    if (!addUserId) { setAddError('请先选择用户'); return }
    try {
      setAdding(true); setAddError('')
      await addAdminGroupMember(groupId, { user_id: addUserId, role: addRole })
      setAddUserId(''); setAddUserName(''); setAddRole('member')
      await load()
    } catch (e: unknown) {
      setAddError(e instanceof Error ? e.message : '添加失败')
    } finally { setAdding(false) }
  }

  // 移除成员
  const handleRemove = async (userId: string) => {
    try { await removeAdminGroupMember(groupId, userId); await load() } catch { }
  }

  // 切换成员角色
  const handleRoleChange = async (userId: string, role: string) => {
    try { await updateAdminGroupMemberRole(groupId, userId, role); await load() } catch { }
  }

  const roleLabel = (role: string) => role === 'backbone' ? '骨干教师' : '普通成员'
  const roleColor = (role: string) =>
    role === 'backbone'
      ? { bg: C.purpleLight, color: C.purple }
      : { bg: C.bg, color: C.textSec }

  return (
    <div style={{ padding: '16px', background: 'rgba(79,123,232,0.04)', borderTop: `1px dashed ${C.border}` }}>
      {/* 面板标题 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
        <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>👥 成员管理</div>
        <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '12px', color: C.textMuted }}>
          收起 ▲
        </button>
      </div>

      {/* 成员列表 */}
      {loading ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>加载中...</div>
      ) : members.length === 0 ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>暂无成员</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', marginBottom: '14px' }}>
          {members.map(m => {
            const rc = roleColor(m.role)
            return (
              <div key={m.user_id} style={{
                display: 'flex', alignItems: 'center', gap: '8px',
                padding: '8px 12px', borderRadius: '8px',
                background: C.white, border: `1px solid ${C.border}`,
              }}>
                {/* 头像 */}
                <div style={{
                  width: '28px', height: '28px', borderRadius: '50%', flexShrink: 0,
                  background: `linear-gradient(135deg,${C.primary},#7C3AED)`,
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  color: '#fff', fontSize: '11px', fontWeight: 700,
                }}>
                  {(m.display_name || m.username).charAt(0).toUpperCase()}
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: '13px', fontWeight: 500, color: C.text }}>{m.display_name}</div>
                  <div style={{ fontSize: '11px', color: C.textMuted }}>加入：{fmt(m.joined_at)}</div>
                </div>
                {/* 角色切换下拉 */}
                <select
                  value={m.role}
                  onChange={e => handleRoleChange(m.user_id, e.target.value)}
                  style={{
                    padding: '3px 8px', borderRadius: '6px',
                    border: `1px solid ${rc.color}`, background: rc.bg, color: rc.color,
                    fontSize: '11px', fontWeight: 600, cursor: 'pointer', outline: 'none',
                  }}>
                  <option value="member">普通成员</option>
                  <option value="backbone">骨干教师</option>
                </select>
                {/* 移除按钮 */}
                <button
                  onClick={() => handleRemove(m.user_id)}
                  style={{
                    padding: '4px 10px', borderRadius: '6px',
                    border: `1px solid ${C.dangerLight}`, background: C.dangerLight,
                    color: C.danger, fontSize: '11px', cursor: 'pointer',
                    fontWeight: 500, whiteSpace: 'nowrap',
                  }}>
                  移除
                </button>
              </div>
            )
          })}
        </div>
      )}

      {/* 添加成员区域 */}
      <div style={{ background: C.white, borderRadius: '10px', border: `1px solid ${C.border}`, padding: '12px' }}>
        <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '10px' }}>添加成员</div>
        {addError && <div style={{ fontSize: '12px', color: C.danger, marginBottom: '8px' }}>{addError}</div>}
        <UserSearchPicker
          label=""
          value={addUserId} valueName={addUserName}
          onChange={(id, n) => { setAddUserId(id); setAddUserName(n) }}
          placeholder="输入用户名搜索..."
        />
        <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
          <select
            value={addRole} onChange={e => setAddRole(e.target.value)}
            style={{ padding: '7px 10px', borderRadius: '7px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', background: C.white }}>
            <option value="member">普通成员</option>
            <option value="backbone">骨干教师</option>
          </select>
          <button
            onClick={handleAdd} disabled={adding || !addUserId}
            style={{
              flex: 1, padding: '7px', borderRadius: '7px', border: 'none',
              background: (!addUserId || adding) ? C.textMuted : C.primary,
              color: '#fff', fontSize: '13px', fontWeight: 600,
              cursor: (!addUserId || adding) ? 'not-allowed' : 'pointer',
            }}>
            {adding ? '添加中...' : '+ 确认添加'}
          </button>
        </div>
      </div>
    </div>
  )
}
