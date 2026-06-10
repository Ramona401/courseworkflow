/**
 * OrgAdminsPanel.tsx — 组织多管理员管理内嵌展开面板（Phase 6.4）
 *
 * 用途：
 *   在 AdminPage「组织架构」Tab 的学校栏中，展开某学校卡片后管理该学校的管理员
 *   （organization_admins 表，role_type=school_admin）。
 *   交互范式仿 MemberPanel：展开式面板 + UserSearchPicker 选人 + 列表 + 移除（二次确认）。
 *
 * 后端对接（admin.ts 已预埋三函数，路由 /lesson-plans/organizations/{id}/admins）：
 *   - getOrgAdmins(orgId)              列出
 *   - addOrgAdmin(orgId, {user_id, role_type})  任命
 *   - removeOrgAdmin(orgId, userId)    移除
 *
 * 权限（后端 organization_admin_service 二次校验，前端不放松）：
 *   - admin        ：可对任意组织任命/移除；
 *   - region_admin ：仅其管辖区域本身或辖区学校；
 *   - senior_operator 等：后端拒绝（本面板也只在 AdminPage 组织架构 Tab 出现，
 *     senior 没有该 Tab 的增改入口，故不会误用）。
 *   失败（403 等）由后端返回错误消息，本面板统一以红字提示。
 *
 * 本面板默认面向"学校管理员任命"场景：role_type 固定为 school_admin，
 *   挂在 school 类型组织下。region 组织的区域管理员任命由 admin 在区域栏另行处理
 *   （本期学校栏先落地，覆盖最高频的"给学校配管理员"需求）。
 */
import { useState, useEffect, useCallback } from 'react'
import { getOrgAdmins, addOrgAdmin, removeOrgAdmin } from '@/api/admin'
import type { OrgAdminItem } from '@/api/admin'
import { C, fmt } from './adminConstants'
import { UserSearchPicker } from './UserSearchPicker'
import { ConfirmDialog } from './ConfirmDialog'

interface OrgAdminsPanelProps {
  /** 组织ID（此处为学校ID） */
  orgId: string
  /** 组织类型：school（学校管理员）/ region（区域管理员），决定任命的 role_type */
  orgType: 'region' | 'school'
  /** 收起面板回调 */
  onClose: () => void
  /** 任命/移除成功后的回调（供父级刷新展示，如学校卡片上的"管理员"名） */
  onChanged?: () => void
}

export function OrgAdminsPanel({ orgId, orgType, onClose, onChanged }: OrgAdminsPanelProps) {
  const [admins, setAdmins] = useState<OrgAdminItem[]>([])
  const [loading, setLoading] = useState(true)
  const [addUserId, setAddUserId] = useState('')
  const [addUserName, setAddUserName] = useState('')
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState('')
  const [confirmRemove, setConfirmRemove] = useState<{ open: boolean; userId: string; name: string }>({
    open: false, userId: '', name: '',
  })

  // role_type 由组织类型决定：学校→school_admin，区域→region_admin
  const roleType = orgType === 'region' ? 'region_admin' : 'school_admin'
  const roleLabel = orgType === 'region' ? '区域管理员' : '学校管理员'

  // ==================== 加载管理员列表 ====================
  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError('')
      setAdmins(await getOrgAdmins(orgId))
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载管理员失败')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => { load() }, [load])

  // ==================== 任命管理员 ====================
  const handleAdd = useCallback(async () => {
    if (!addUserId) { setError('请先选择要任命的用户'); return }
    try {
      setAdding(true)
      setError('')
      await addOrgAdmin(orgId, { user_id: addUserId, role_type: roleType })
      setAddUserId('')
      setAddUserName('')
      await load()
      onChanged?.()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '任命失败')
    } finally {
      setAdding(false)
    }
  }, [addUserId, orgId, roleType, load, onChanged])

  // ==================== 移除管理员（二次确认）====================
  const doRemove = useCallback(async (userId: string) => {
    try {
      setError('')
      await removeOrgAdmin(orgId, userId)
      await load()
      onChanged?.()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '移除失败')
    } finally {
      setConfirmRemove({ open: false, userId: '', name: '' })
    }
  }, [orgId, load, onChanged])

  return (
    <div style={{ padding: '16px', background: 'rgba(124,58,237,0.05)', borderTop: `1px dashed ${C.border}` }}>

      {/* 移除二次确认弹窗 */}
      {confirmRemove.open && (
        <ConfirmDialog
          title={`移除${roleLabel}`}
          message={`确认移除「${confirmRemove.name}」的${roleLabel}身份？移除后该用户将无法再管理本${orgType === 'region' ? '区域' : '学校'}。`}
          onConfirm={() => doRemove(confirmRemove.userId)}
          onCancel={() => setConfirmRemove({ open: false, userId: '', name: '' })}
        />
      )}

      {/* 面板标题 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>🛡️ {roleLabel}管理</span>
          {admins.length > 0 && (
            <span style={{ fontSize: '11px', padding: '1px 7px', borderRadius: '10px', background: C.purpleLight, color: C.purple, fontWeight: 600 }}>
              共 {admins.length} 人
            </span>
          )}
        </div>
        <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '12px', color: C.textMuted }}>
          收起 ▲
        </button>
      </div>

      {/* 错误提示 */}
      {error && (
        <div style={{ fontSize: '12px', color: C.danger, marginBottom: '10px', padding: '8px 12px', background: C.dangerLight, borderRadius: '8px', lineHeight: 1.5 }}>
          {error}
        </div>
      )}

      {/* 管理员列表 */}
      {loading ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>加载中...</div>
      ) : admins.length === 0 ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>暂无{roleLabel}，请在下方添加</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', marginBottom: '14px' }}>
          {admins.map(a => (
            <div key={a.id} style={{
              display: 'flex', alignItems: 'center', gap: '8px',
              padding: '8px 12px', borderRadius: '8px',
              background: C.white, border: `1px solid ${C.border}`,
            }}>
              {/* 头像 */}
              <div style={{
                width: '28px', height: '28px', borderRadius: '50%', flexShrink: 0,
                background: 'linear-gradient(135deg,#7C3AED,#4F7BE8)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                color: '#fff', fontSize: '11px', fontWeight: 700,
              }}>
                {(a.display_name || a.username).charAt(0).toUpperCase()}
              </div>
              {/* 姓名 + 任命时间 */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>{a.display_name || a.username}</span>
                  <span style={{ fontSize: '10px', padding: '1px 6px', borderRadius: '8px', background: C.purpleLight, color: C.purple, border: `1px solid ${C.purple}44`, fontWeight: 700 }}>
                    {roleLabel}
                  </span>
                </div>
                <div style={{ fontSize: '11px', color: C.textMuted }}>@{a.username} · 任命于 {fmt(a.created_at)}</div>
              </div>
              {/* 移除按钮 */}
              <button
                onClick={() => setConfirmRemove({ open: true, userId: a.user_id, name: a.display_name || a.username })}
                style={{
                  padding: '4px 10px', borderRadius: '6px',
                  border: '1px solid #FEE2E2', background: '#FEF2F2',
                  color: '#EF4444', fontSize: '11px', cursor: 'pointer',
                  fontWeight: 500, whiteSpace: 'nowrap',
                }}>
                移除
              </button>
            </div>
          ))}
        </div>
      )}

      {/* 任命区域 */}
      <div style={{ background: C.white, borderRadius: '10px', border: `1px solid ${C.border}`, padding: '12px' }}>
        <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '10px' }}>任命{roleLabel}</div>
        <UserSearchPicker
          label=""
          value={addUserId} valueName={addUserName}
          onChange={(id, n) => { setAddUserId(id); setAddUserName(n) }}
          placeholder="输入用户名搜索要任命的用户..."
        />
        <button
          onClick={handleAdd} disabled={adding || !addUserId}
          style={{
            width: '100%', padding: '8px', borderRadius: '7px', border: 'none', marginTop: '4px',
            background: (!addUserId || adding) ? '#E5E7EB' : C.purple,
            color: (!addUserId || adding) ? '#9CA3AF' : '#fff',
            fontSize: '13px', fontWeight: 600,
            cursor: (!addUserId || adding) ? 'not-allowed' : 'pointer',
          }}>
          {adding ? '任命中...' : `+ 任命为${roleLabel}`}
        </button>
        <div style={{ marginTop: '8px', fontSize: '11px', color: C.textMuted, lineHeight: 1.5 }}>
          💡 {roleLabel}可管理本{orgType === 'region' ? '区域' : '学校'}的用户与教研组。一个{orgType === 'region' ? '区域' : '学校'}可设置多名管理员。
        </div>
      </div>
    </div>
  )
}
