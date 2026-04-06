/**
 * RolesTab.tsx — 角色权限 Tab 主组件
 *
 * 布局：
 *   ① 系统内置角色区块（4张卡片，只读展示 + 展开权限折叠）
 *   ② 自定义角色区块（列表 + 新建/编辑/禁用/删除操作）
 *
 * 删除逻辑：
 *   - 有用户使用时：提示"N个用户将降级为viewer"
 *   - 无用户使用时：直接二次确认
 */
import { useState, useEffect, useCallback } from 'react'
import { C } from './adminConstants'
import {
  PERMISSION_GROUPS, SYSTEM_ROLES, getRoleColor,
  type PermissionGroup,
} from './roleConstants'
import { RoleFormModal } from './RoleFormModal'
import {
  listRoles, updateRoleStatus, deleteRole,
} from '@/api/roles'
import type { RoleListItem } from '@/api/roles'

// ==================== Props ====================

interface RolesTabProps {
  showToast: (msg: string, type: 'success' | 'error') => void
}

// ==================== 系统内置角色卡片 ====================

function SystemRoleCard({ code, displayName, description, permissionCodes }: {
  code: string
  displayName: string
  description: string
  permissionCodes: string[]
}) {
  const [expanded, setExpanded] = useState(false)
  const colors = getRoleColor(code)

  return (
    <div style={{
      background: C.white, borderRadius: '14px',
      border: `1px solid ${colors.border}`,
      overflow: 'hidden', flex: '1 1 200px',
      boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
    }}>
      {/* 卡片头部 */}
      <div style={{ padding: '16px', background: colors.bg, borderBottom: `1px solid ${colors.border}` }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: '6px' }}>
          <div>
            <div style={{ fontSize: '15px', fontWeight: 700, color: colors.color }}>{displayName}</div>
            <div style={{ fontSize: '11px', fontFamily: 'monospace', color: C.textMuted, marginTop: '2px' }}>{code}</div>
          </div>
          {/* 系统内置标签 */}
          <span style={{
            padding: '2px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 600,
            background: 'rgba(107,114,128,0.1)', color: C.textSec,
            border: '1px solid rgba(107,114,128,0.2)', whiteSpace: 'nowrap',
          }}>
            🔒 系统内置
          </span>
        </div>
        <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.5 }}>{description}</div>
        <div style={{ marginTop: '8px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span style={{ fontSize: '12px', color: colors.color, fontWeight: 600 }}>
            {permissionCodes.length} 项权限
          </span>
          <button
            onClick={() => setExpanded(p => !p)}
            style={{
              padding: '4px 12px', borderRadius: '6px', fontSize: '11px', fontWeight: 600,
              border: `1px solid ${colors.border}`, background: C.white,
              color: colors.color, cursor: 'pointer',
            }}
          >
            {expanded ? '收起 ▲' : '展开权限 ▼'}
          </button>
        </div>
      </div>

      {/* 展开的权限列表 */}
      {expanded && (
        <div style={{ padding: '12px 14px' }}>
          {PERMISSION_GROUPS.map((group: PermissionGroup) => {
            const groupPerms = group.permissions.filter(p => permissionCodes.includes(p.code))
            if (groupPerms.length === 0) return null
            return (
              <div key={group.groupKey} style={{ marginBottom: '10px' }}>
                <div style={{ fontSize: '11px', fontWeight: 700, color: C.textSec, marginBottom: '5px', display: 'flex', alignItems: 'center', gap: '4px' }}>
                  <span>{group.icon}</span>
                  <span>{group.groupName}</span>
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '3px' }}>
                  {groupPerms.map(perm => (
                    <div key={perm.code} style={{ display: 'flex', alignItems: 'center', gap: '5px', padding: '4px 6px', borderRadius: '5px', background: C.bg }}>
                      <span style={{ color: C.success, fontSize: '11px', flexShrink: 0 }}>✅</span>
                      <span style={{ fontSize: '12px', color: C.text }}>{perm.label}</span>
                    </div>
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

// ==================== 自定义角色行 ====================

function CustomRoleRow({
  role, onEdit, onToggleStatus, onDelete,
}: {
  role: RoleListItem
  onEdit: () => void
  onToggleStatus: () => void
  onDelete: () => void
}) {
  const colors = getRoleColor(role.base_role || role.role_code)
  const isActive = role.status === 'active'

  return (
    <div style={{
      display: 'grid',
      gridTemplateColumns: '2fr 1.5fr 1.5fr 0.8fr 1fr 1fr 1.6fr',
      padding: '14px 20px', alignItems: 'center',
      borderBottom: `1px solid ${C.border}`,
      transition: 'background 150ms ease',
    }}
    onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
    onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}>

      {/* 角色名 */}
      <div>
        <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{role.display_name}</div>
        {role.description && (
          <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '2px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {role.description}
          </div>
        )}
      </div>

      {/* 英文标识 */}
      <div style={{ fontSize: '12px', fontFamily: 'monospace', color: C.textSec, background: C.bg, padding: '2px 8px', borderRadius: '5px', width: 'fit-content' }}>
        {role.role_code}
      </div>

      {/* 继承自 */}
      <div>
        {role.base_role ? (
          <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, background: colors.bg, color: colors.color, border: `1px solid ${colors.border}` }}>
            继承 {role.base_role}
          </span>
        ) : (
          <span style={{ fontSize: '12px', color: C.textMuted }}>自定义</span>
        )}
      </div>

      {/* 权限数 */}
      <div style={{ fontSize: '13px', color: C.primary, fontWeight: 600, textAlign: 'center' }}>
        {role.permission_count}
      </div>

      {/* 状态 */}
      <div>
        <span style={{
          display: 'inline-flex', alignItems: 'center', gap: '4px',
          padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 600,
          background: isActive ? 'rgba(16,185,129,0.08)' : 'rgba(239,68,68,0.08)',
          color: isActive ? C.success : C.danger,
        }}>
          <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: isActive ? C.success : C.danger }} />
          {isActive ? '启用' : '禁用'}
        </span>
      </div>

      {/* 已分配用户数 */}
      <div style={{ fontSize: '13px', color: C.textSec, textAlign: 'center' }}>
        {role.user_count > 0
          ? <span style={{ color: C.warning, fontWeight: 600 }}>{role.user_count} 人</span>
          : <span style={{ color: C.textMuted }}>0</span>}
      </div>

      {/* 操作按钮 */}
      <div style={{ display: 'flex', gap: '6px', justifyContent: 'flex-end' }}>
        <button onClick={onEdit}
          style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.primary}33`, background: C.primaryLight, color: C.primary, fontSize: '11px', fontWeight: 600, cursor: 'pointer' }}>
          ✏️ 编辑
        </button>
        <button onClick={onToggleStatus}
          style={{
            padding: '4px 10px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, cursor: 'pointer',
            border: `1px solid ${isActive ? C.danger + '33' : C.success + '33'}`,
            background: isActive ? C.dangerLight : C.successLight,
            color: isActive ? C.danger : C.success,
          }}>
          {isActive ? '🚫 禁用' : '✅ 启用'}
        </button>
        <button onClick={onDelete}
          style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.danger}33`, background: C.dangerLight, color: C.danger, fontSize: '11px', fontWeight: 600, cursor: 'pointer' }}>
          🗑️ 删除
        </button>
      </div>
    </div>
  )
}

// ==================== 删除确认弹窗（支持用户降级提示）====================

function DeleteRoleDialog({ role, onConfirm, onCancel }: {
  role: RoleListItem
  onConfirm: () => void
  onCancel: () => void
}) {
  const hasUsers = role.user_count > 0
  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 12000,
      background: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(4px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}>
      <div style={{
        background: C.white, borderRadius: '16px', width: '400px',
        padding: '28px', boxShadow: '0 20px 60px rgba(0,0,0,0.22)',
      }}>
        <div style={{ fontSize: '16px', fontWeight: 700, color: C.text, marginBottom: '10px' }}>
          🗑️ 删除角色「{role.display_name}」
        </div>
        {hasUsers ? (
          <div style={{ padding: '12px 14px', borderRadius: '10px', background: C.warningLight, border: `1px solid ${C.warning}44`, marginBottom: '16px' }}>
            <div style={{ fontSize: '13px', fontWeight: 600, color: C.warning, marginBottom: '4px' }}>
              ⚠️ 该角色下有 {role.user_count} 个用户
            </div>
            <div style={{ fontSize: '13px', color: C.textSec, lineHeight: 1.5 }}>
              删除后，这些用户的角色将自动降级为 <strong>viewer（普通教师）</strong>，确认继续？
            </div>
          </div>
        ) : (
          <div style={{ fontSize: '14px', color: C.textSec, marginBottom: '16px', lineHeight: 1.6 }}>
            确认删除角色「{role.display_name}」（{role.role_code}）？此操作不可撤销。
          </div>
        )}
        <div style={{ display: 'flex', gap: '10px' }}>
          <button onClick={onCancel}
            style={{ flex: 1, padding: '10px', borderRadius: '10px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>
            取消
          </button>
          <button onClick={onConfirm}
            style={{ flex: 1, padding: '10px', borderRadius: '10px', border: 'none', background: C.danger, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>
            {hasUsers ? `确认删除（${role.user_count}人降级）` : '确认删除'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ==================== RolesTab 主组件 ====================

export function RolesTab({ showToast }: RolesTabProps) {

  // ---- 自定义角色列表 ----
  const [customRoles, setCustomRoles] = useState<RoleListItem[]>([])
  const [loading, setLoading]         = useState(false)

  // ---- 弹窗状态 ----
  const [formModal, setFormModal] = useState<{
    open: boolean; mode: 'create' | 'edit'; initial?: RoleListItem
  }>({ open: false, mode: 'create' })

  const [deleteTarget, setDeleteTarget] = useState<RoleListItem | null>(null)

  // ---- 加载自定义角色列表（过滤掉系统内置） ----
  const loadRoles = useCallback(async () => {
    try {
      setLoading(true)
      const data = await listRoles()
      // 只显示非系统内置角色（is_system=false）
      setCustomRoles((data.roles ?? []).filter(r => !r.is_system))
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '加载角色失败', 'error')
    } finally {
      setLoading(false)
    }
  }, [showToast])

  useEffect(() => { loadRoles() }, [loadRoles])

  // ---- 切换状态 ----
  const handleToggleStatus = async (role: RoleListItem) => {
    const newStatus = role.status === 'active' ? 'disabled' : 'active'
    try {
      await updateRoleStatus(role.id, { status: newStatus })
      showToast(newStatus === 'active' ? '已启用' : '已禁用', 'success')
      loadRoles()
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '操作失败', 'error')
    }
  }

  // ---- 执行删除 ----
  const handleConfirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteRole(deleteTarget.id)
      showToast('角色已删除', 'success')
      setDeleteTarget(null)
      loadRoles()
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '删除失败', 'error')
      setDeleteTarget(null)
    }
  }

  // ==================== 渲染 ====================

  return (
    <div>

      {/* ==== ① 系统内置角色区块 ==== */}
      <div style={{
        background: C.white, borderRadius: '16px',
        border: `1px solid ${C.border}`, overflow: 'hidden', marginBottom: '20px',
      }}>
        {/* 区块标题 */}
        <div style={{
          padding: '16px 20px', borderBottom: `1px solid ${C.border}`,
          background: C.bg, display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        }}>
          <div>
            <div style={{ fontSize: '15px', fontWeight: 700, color: C.text }}>🔒 系统内置角色</div>
            <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>
              内置角色权限由系统固定，不可编辑或删除，仅供查看
            </div>
          </div>
          <span style={{
            padding: '4px 12px', borderRadius: '8px', fontSize: '12px', fontWeight: 600,
            background: 'rgba(107,114,128,0.08)', color: C.textSec,
            border: `1px solid ${C.border}`,
          }}>
            共 {SYSTEM_ROLES.length} 个
          </span>
        </div>

        {/* 4张卡片并排 */}
        <div style={{ padding: '16px', display: 'flex', gap: '12px', flexWrap: 'wrap' }}>
          {SYSTEM_ROLES.map(role => (
            <SystemRoleCard
              key={role.code}
              code={role.code}
              displayName={role.displayName}
              description={role.description}
              permissionCodes={role.permissionCodes}
            />
          ))}
        </div>
      </div>

      {/* ==== ② 自定义角色区块 ==== */}
      <div style={{
        background: C.white, borderRadius: '16px',
        border: `1px solid ${C.border}`, overflow: 'hidden',
      }}>
        {/* 区块标题 + 新建按钮 */}
        <div style={{
          padding: '16px 20px', borderBottom: `1px solid ${C.border}`,
          background: C.bg, display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        }}>
          <div>
            <div style={{ fontSize: '15px', fontWeight: 700, color: C.text }}>🎭 自定义角色</div>
            <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>
              可按需创建自定义角色，灵活配置权限组合
            </div>
          </div>
          <button
            onClick={() => setFormModal({ open: true, mode: 'create' })}
            style={{
              padding: '8px 18px', borderRadius: '9px', border: 'none',
              background: `linear-gradient(135deg,${C.primary},#7C3AED)`,
              color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer',
            }}
          >
            + 新建自定义角色
          </button>
        </div>

        {/* 表头 */}
        {customRoles.length > 0 && (
          <div style={{
            display: 'grid', gridTemplateColumns: '2fr 1.5fr 1.5fr 0.8fr 1fr 1fr 1.6fr',
            padding: '10px 20px', background: C.bg, borderBottom: `1px solid ${C.border}`,
            fontSize: '12px', fontWeight: 600, color: C.textSec,
          }}>
            <span>角色名称</span>
            <span>英文标识</span>
            <span>继承自</span>
            <span style={{ textAlign: 'center' }}>权限数</span>
            <span>状态</span>
            <span style={{ textAlign: 'center' }}>已分配用户</span>
            <span style={{ textAlign: 'right' }}>操作</span>
          </div>
        )}

        {/* 列表内容 */}
        {loading ? (
          <div style={{ padding: '48px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>
            加载中...
          </div>
        ) : customRoles.length === 0 ? (
          /* 空状态 */
          <div style={{ padding: '56px 20px', textAlign: 'center' }}>
            <div style={{ fontSize: '40px', marginBottom: '12px' }}>🎭</div>
            <div style={{ fontSize: '15px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>
              尚未创建自定义角色
            </div>
            <div style={{ fontSize: '13px', color: C.textMuted, marginBottom: '20px' }}>
              点击上方「+ 新建自定义角色」按钮开始创建，可灵活组合权限
            </div>
            <button
              onClick={() => setFormModal({ open: true, mode: 'create' })}
              style={{
                padding: '9px 22px', borderRadius: '10px', border: 'none',
                background: `linear-gradient(135deg,${C.primary},#7C3AED)`,
                color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer',
              }}
            >
              + 新建自定义角色
            </button>
          </div>
        ) : (
          customRoles.map(role => (
            <CustomRoleRow
              key={role.id}
              role={role}
              onEdit={() => setFormModal({ open: true, mode: 'edit', initial: role })}
              onToggleStatus={() => handleToggleStatus(role)}
              onDelete={() => setDeleteTarget(role)}
            />
          ))
        )}
      </div>

      {/* ==== 新建/编辑弹窗 ==== */}
      {formModal.open && (
        <RoleFormModal
          mode={formModal.mode}
          initial={formModal.initial}
          onClose={() => setFormModal(p => ({ ...p, open: false }))}
          onSaved={loadRoles}
          showToast={showToast}
        />
      )}

      {/* ==== 删除确认弹窗 ==== */}
      {deleteTarget && (
        <DeleteRoleDialog
          role={deleteTarget}
          onConfirm={handleConfirmDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
