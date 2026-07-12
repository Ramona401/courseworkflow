/**
 * OrgAdminsPanel.tsx — 组织多管理员管理内嵌展开面板（Phase 6.4 + B13 任命即同步身份）
 *
 * 用途：
 *   在 AdminPage「组织架构」Tab 的学校卡片/区域卡片中，展开后管理该组织的管理员
 *   （organization_admins 表，school→school_admin / region→region_admin）。
 *   交互范式仿 MemberPanel：展开式面板 + UserSearchPicker 选人 + 列表 + 移除（二次确认）。
 *
 * 后端对接（admin.ts，路由 /lesson-plans/organizations/{id}/admins）：
 *   - getOrgAdmins(orgId)                                    列出
 *   - addOrgAdmin(orgId, {user_id, role_type, sync_role})    任命（B13 返回 AddOrgAdminResult）
 *   - removeOrgAdmin(orgId, userId)                          移除
 *
 * 权限（后端 organization_admin_service 二次校验，前端不放松）：
 *   - admin        ：可对任意组织任命/移除；
 *   - region_admin ：仅其管辖区域本身或辖区学校；
 *   - 其它角色      ：后端拒绝。失败（403 等）由后端返回错误消息，本面板红字提示。
 *
 * B13 任命即同步身份（根治"有管辖无门票"静默失效）：
 *   - 选人后展示其当前系统身份（RoleBadge，来自 UserSearchPicker 第三参透传）；
 *   - 目标身份为 骨干教师(operator)/普通教师(viewer) 时，显示默认勾选的
 *     "任命后同步升级为区域管理员/学校管理员"勾选框（sync_role）；
 *   - 目标已具备其它身份时不显勾选框，灰字说明"任命仅授予管辖范围，不变更账户身份"；
 *   - 任命成功后 toast 后端拼好的 message（四种结果文案由后端统一产出），
 *     role_synced=true 时前端追加"对方重新登录后生效"——JWT 内嵌角色，
 *     目标用户的旧 token 在重新登录前仍按旧身份判定；
 *   - 移除任命永不降级账户身份（后端规则），面板底部与移除确认弹窗均有明示。
 */
import { useState, useEffect, useCallback } from 'react'
import { getOrgAdmins, addOrgAdmin, removeOrgAdmin } from '@/api/admin'
import type { OrgAdminItem } from '@/api/admin'
import { C, fmt } from './adminConstants'
import { UserSearchPicker } from './UserSearchPicker'
import { ConfirmDialog } from './ConfirmDialog'
import { RoleBadge } from './adminShared'

interface OrgAdminsPanelProps {
  /** 组织ID（学校ID或区域ID） */
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
  // B13：选中用户的当前系统身份（UserSearchPicker 第三参透传；清空选择时为空串）
  const [addUserRole, setAddUserRole] = useState('')
  // B13：同步升级勾选框状态，默认勾选（换人重选时重置回 true）
  const [syncRole, setSyncRole] = useState(true)
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState('')
  // B13：成功提示（绿色横幅，展示后端拼好的任命结果文案）
  const [notice, setNotice] = useState('')
  const [confirmRemove, setConfirmRemove] = useState<{ open: boolean; userId: string; name: string }>({
    open: false, userId: '', name: '',
  })

  // role_type 由组织类型决定：学校→school_admin，区域→region_admin
  const roleType = orgType === 'region' ? 'region_admin' : 'school_admin'
  const roleLabel = orgType === 'region' ? '区域管理员' : '学校管理员'

  // B13：同步升级白名单——仅骨干教师/普通教师起步可升级（与后端 service 白名单一致）
  const syncEligible = addUserRole === 'operator' || addUserRole === 'viewer'

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

  // ==================== 任命管理员（B13：携带 sync_role，toast 后端文案）====================
  const handleAdd = useCallback(async () => {
    if (!addUserId) { setError('请先选择要任命的用户'); return }
    try {
      setAdding(true)
      setError('')
      setNotice('')
      // sync_role 仅在"白名单内且勾选"时为 true；已具管理身份者恒 false（后端亦有兜底）
      const result = await addOrgAdmin(orgId, {
        user_id: addUserId,
        role_type: roleType,
        sync_role: syncEligible && syncRole,
      })
      // 后端 message 已按四种结果拼好；同步成功时追加重新登录提示（JWT 内嵌角色）
      setNotice(result.message + (result.role_synced ? '（对方重新登录后生效）' : ''))
      setAddUserId('')
      setAddUserName('')
      setAddUserRole('')
      setSyncRole(true)
      await load()
      onChanged?.()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '任命失败')
    } finally {
      setAdding(false)
    }
  }, [addUserId, orgId, roleType, syncEligible, syncRole, load, onChanged])

  // ==================== 移除管理员（二次确认；B13：移除永不降级身份）====================
  const doRemove = useCallback(async (userId: string) => {
    try {
      setError('')
      setNotice('')
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

      {/* 移除二次确认弹窗（B13：明示移除不降级身份） */}
      {confirmRemove.open && (
        <ConfirmDialog
          title={`移除${roleLabel}`}
          message={`确认移除「${confirmRemove.name}」的${roleLabel}身份？移除后该用户将无法再管理本${orgType === 'region' ? '区域' : '学校'}。移除不会降级其账户身份，如需降级请到用户管理手动修改。`}
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

      {/* 错误提示（红） */}
      {error && (
        <div style={{ fontSize: '12px', color: C.danger, marginBottom: '10px', padding: '8px 12px', background: C.dangerLight, borderRadius: '8px', lineHeight: 1.5 }}>
          {error}
        </div>
      )}

      {/* B13：任命结果提示（绿，展示后端文案） */}
      {notice && (
        <div style={{ fontSize: '12px', color: C.success, marginBottom: '10px', padding: '8px 12px', background: C.successLight, borderRadius: '8px', lineHeight: 1.5 }}>
          ✓ {notice}
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
          onChange={(id, n, role) => {
            setAddUserId(id)
            setAddUserName(n)
            setAddUserRole(role || '')
            setSyncRole(true)   // 换人重选时勾选框重置回默认勾选
          }}
          placeholder="输入用户名搜索要任命的用户..."
        />

        {/* B13：选中后展示当前身份 + 同步升级勾选/说明 */}
        {addUserId && addUserRole && (
          <div style={{
            marginBottom: '10px', padding: '10px 12px', borderRadius: '8px',
            background: C.bg, border: `1px solid ${C.border}`,
          }}>
            {/* 当前身份行 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: syncEligible ? '8px' : '6px' }}>
              <span style={{ fontSize: '12px', color: C.textSec }}>当前身份：</span>
              <RoleBadge role={addUserRole} />
            </div>
            {syncEligible ? (
              // 白名单内：默认勾选的同步升级勾选框
              <label style={{ display: 'flex', alignItems: 'flex-start', gap: '8px', cursor: 'pointer' }}>
                <input
                  type="checkbox"
                  checked={syncRole}
                  onChange={e => setSyncRole(e.target.checked)}
                  style={{ marginTop: '2px', cursor: 'pointer' }}
                />
                <span style={{ fontSize: '12px', color: C.text, lineHeight: 1.5 }}>
                  任命后同步升级账户身份为「{roleLabel}」
                  <span style={{ color: C.textMuted }}>
                    （推荐勾选：不升级则对方登录后没有用户管理入口）
                  </span>
                </span>
              </label>
            ) : (
              // 白名单外（已具管理身份/教研员等）：灰字说明，不显勾选框
              <div style={{ fontSize: '12px', color: C.textMuted, lineHeight: 1.5 }}>
                任命仅授予管辖范围，不会变更其账户身份。
              </div>
            )}
          </div>
        )}

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
          💡 {roleLabel}可管理本{orgType === 'region' ? '区域' : '学校'}的用户与教研组。一个{orgType === 'region' ? '区域' : '学校'}可设置多名管理员。移除任命不会自动降级账户身份。
        </div>
      </div>
    </div>
  )
}
