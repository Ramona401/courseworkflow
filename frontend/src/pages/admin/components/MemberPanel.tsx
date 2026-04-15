/**
 * MemberPanel.tsx — 教研组成员管理内嵌展开面板
 *
 * v109改动：支持多组长
 *   - 角色下拉新增"教研组长"选项（role='lead'）
 *   - 组长用金色标识，排在列表最前
 *   - 添加成员时可直接指定为组长
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

/** 角色显示配置：lead=金色组长 / backbone=紫色骨干 / member=灰色普通 */
const ROLE_CONFIG: Record<string, { label: string; bg: string; color: string; border: string }> = {
  lead:     { label: '教研组长', bg: '#FFFBEB', color: '#B45309', border: '#F59E0B' },
  backbone: { label: '骨干教师', bg: '#F5F3FF', color: '#7C3AED', border: '#A78BFA' },
  member:   { label: '普通成员', bg: '#F9FAFB', color: '#6B7280', border: '#E5E7EB' },
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
    } catch { /* 忽略 */ } finally { setLoading(false) }
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

  // 移除成员（有确认弹窗）
  const handleRemove = async (userId: string, displayName: string) => {
    if (!confirm(`确定要移除「${displayName}」吗？`)) return
    try { await removeAdminGroupMember(groupId, userId); await load() } catch { /* 忽略 */ }
  }

  // 切换成员角色
  const handleRoleChange = async (userId: string, role: string) => {
    try { await updateAdminGroupMemberRole(groupId, userId, role); await load() } catch { /* 忽略 */ }
  }

  // 各角色人数统计
  const leadCount     = members.filter(m => m.role === 'lead').length
  const backboneCount = members.filter(m => m.role === 'backbone').length
  const memberCount   = members.filter(m => m.role === 'member').length

  return (
    <div style={{ padding: '16px', background: 'rgba(79,123,232,0.04)', borderTop: `1px dashed ${C.border}` }}>

      {/* 面板标题 + 角色统计徽章 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>👥 成员管理</span>
          {members.length > 0 && (
            <div style={{ display: 'flex', gap: '6px' }}>
              {leadCount > 0 && (
                <span style={{ fontSize: '11px', padding: '1px 7px', borderRadius: '10px', background: '#FFFBEB', color: '#B45309', border: '1px solid #F59E0B', fontWeight: 600 }}>
                  组长×{leadCount}
                </span>
              )}
              {backboneCount > 0 && (
                <span style={{ fontSize: '11px', padding: '1px 7px', borderRadius: '10px', background: '#F5F3FF', color: '#7C3AED', border: '1px solid #A78BFA' }}>
                  骨干×{backboneCount}
                </span>
              )}
              {memberCount > 0 && (
                <span style={{ fontSize: '11px', padding: '1px 7px', borderRadius: '10px', background: '#F9FAFB', color: '#6B7280', border: '1px solid #E5E7EB' }}>
                  成员×{memberCount}
                </span>
              )}
            </div>
          )}
        </div>
        <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '12px', color: C.textMuted }}>
          收起 ▲
        </button>
      </div>

      {/* 成员列表 */}
      {loading ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>加载中...</div>
      ) : members.length === 0 ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>暂无成员，请先添加</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', marginBottom: '14px' }}>
          {members.map(m => {
            const rc = ROLE_CONFIG[m.role] || ROLE_CONFIG.member
            const isLead = m.role === 'lead'
            return (
              <div key={m.user_id} style={{
                display: 'flex', alignItems: 'center', gap: '8px',
                padding: '8px 12px', borderRadius: '8px',
                background: isLead ? '#FFFDF0' : C.white,
                border: `1px solid ${isLead ? '#F59E0B' : C.border}`,
                boxShadow: isLead ? '0 1px 4px rgba(245,158,11,0.12)' : 'none',
              }}>
                {/* 头像：组长金色渐变，其他蓝紫渐变 */}
                <div style={{
                  width: '28px', height: '28px', borderRadius: '50%', flexShrink: 0,
                  background: isLead
                    ? 'linear-gradient(135deg,#F59E0B,#D97706)'
                    : `linear-gradient(135deg,${C.primary},#7C3AED)`,
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  color: '#fff', fontSize: '11px', fontWeight: 700,
                }}>
                  {(m.display_name || m.username).charAt(0).toUpperCase()}
                </div>

                {/* 姓名 + 组长标签 + 加入时间 */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                    <span style={{ fontSize: '13px', fontWeight: isLead ? 600 : 500, color: C.text }}>
                      {m.display_name}
                    </span>
                    {isLead && (
                      <span style={{ fontSize: '10px', padding: '1px 5px', borderRadius: '8px', background: '#FEF3C7', color: '#B45309', border: '1px solid #F59E0B', fontWeight: 700 }}>
                        组长
                      </span>
                    )}
                  </div>
                  <div style={{ fontSize: '11px', color: C.textMuted }}>加入：{fmt(m.joined_at)}</div>
                </div>

                {/* 角色切换下拉：支持组长/骨干/普通 */}
                <select
                  value={m.role}
                  onChange={e => handleRoleChange(m.user_id, e.target.value)}
                  style={{
                    padding: '3px 8px', borderRadius: '6px',
                    border: `1px solid ${rc.border}`,
                    background: rc.bg, color: rc.color,
                    fontSize: '11px', fontWeight: 600,
                    cursor: 'pointer', outline: 'none',
                  }}>
                  <option value="lead">教研组长</option>
                  <option value="backbone">骨干教师</option>
                  <option value="member">普通成员</option>
                </select>

                {/* 移除按钮 */}
                <button
                  onClick={() => handleRemove(m.user_id, m.display_name)}
                  style={{
                    padding: '4px 10px', borderRadius: '6px',
                    border: '1px solid #FEE2E2', background: '#FEF2F2',
                    color: '#EF4444', fontSize: '11px', cursor: 'pointer',
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
        {addError && <div style={{ fontSize: '12px', color: '#EF4444', marginBottom: '8px' }}>{addError}</div>}
        <UserSearchPicker
          label=""
          value={addUserId} valueName={addUserName}
          onChange={(id, n) => { setAddUserId(id); setAddUserName(n) }}
          placeholder="输入用户名搜索..."
        />
        <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
          {/* 角色选择：支持组长/骨干/普通，选中状态有颜色变化 */}
          <select
            value={addRole} onChange={e => setAddRole(e.target.value)}
            style={{
              padding: '7px 10px', borderRadius: '7px',
              border: `1px solid ${addRole === 'lead' ? '#F59E0B' : addRole === 'backbone' ? '#A78BFA' : C.border}`,
              background: addRole === 'lead' ? '#FFFBEB' : addRole === 'backbone' ? '#F5F3FF' : C.white,
              color: addRole === 'lead' ? '#B45309' : addRole === 'backbone' ? '#7C3AED' : C.text,
              fontSize: '13px', fontWeight: addRole === 'lead' ? 600 : 400,
              outline: 'none', cursor: 'pointer',
            }}>
            <option value="member">普通成员</option>
            <option value="backbone">骨干教师</option>
            <option value="lead">教研组长</option>
          </select>
          <button
            onClick={handleAdd} disabled={adding || !addUserId}
            style={{
              flex: 1, padding: '7px', borderRadius: '7px', border: 'none',
              background: (!addUserId || adding) ? '#E5E7EB' : C.primary,
              color: (!addUserId || adding) ? '#9CA3AF' : '#fff',
              fontSize: '13px', fontWeight: 600,
              cursor: (!addUserId || adding) ? 'not-allowed' : 'pointer',
            }}>
            {adding ? '添加中...' : '+ 确认添加'}
          </button>
        </div>
        {/* 多组长说明 */}
        <div style={{ marginTop: '8px', fontSize: '11px', color: C.textMuted, lineHeight: 1.5 }}>
          💡 一个教研组可以有多个组长，组长拥有教案评审权限
        </div>
      </div>
    </div>
  )
}
