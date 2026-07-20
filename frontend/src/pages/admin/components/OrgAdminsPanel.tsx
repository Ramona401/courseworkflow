/**
 * OrgAdminsPanel.tsx — 组织多管理员管理面板
 *
 * 区域管理员固定教育域规则：
 *   1. 任命region_admin时必须主动选择k12、vocational或adult；
 *   2. 可选项由后端按该区域实际存在的有效学校类型返回；
 *   3. 前端不默认选择K12；
 *   4. 同一用户跨教育域任命由后端返回409拒绝；
 *   5. 存量未配置区域任命明确显示异常标记；
 *   6. school_admin不填写教育域，直接继承学校。
 */

import { useCallback, useEffect, useState } from 'react'
import {
  addOrgAdmin,
  getOrgAdminManagement,
  removeOrgAdmin,
} from '@/api/admin'
import type {
  OrgAdminItem,
  TeachingEducationDomain,
} from '@/api/admin'
import { C, fmt } from './adminConstants'
import { ConfirmDialog } from './ConfirmDialog'
import { RoleBadge } from './adminShared'
import { UserSearchPicker } from './UserSearchPicker'

interface OrgAdminsPanelProps {
  orgId: string
  orgType: 'region' | 'school'
  onClose: () => void
  onChanged?: () => void
}

const educationDomainNames: Record<
  TeachingEducationDomain,
  string
> = {
  k12: 'K12基础教育',
  vocational: '职业教育',
  adult: '成人教育',
}

function getEducationDomainName(domain: string): string {
  if (
    domain === 'k12' ||
    domain === 'vocational' ||
    domain === 'adult'
  ) {
    return educationDomainNames[domain]
  }

  return '教育域未配置'
}

export function OrgAdminsPanel({
  orgId,
  orgType,
  onClose,
  onChanged,
}: OrgAdminsPanelProps) {
  const [admins, setAdmins] = useState<OrgAdminItem[]>([])
  const [availableDomains, setAvailableDomains] = useState<
    TeachingEducationDomain[]
  >([])
  const [educationDomain, setEducationDomain] = useState<
    TeachingEducationDomain | ''
  >('')

  const [loading, setLoading] = useState(true)
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const [addUserId, setAddUserId] = useState('')
  const [addUserName, setAddUserName] = useState('')
  const [addUserRole, setAddUserRole] = useState('')
  const [syncRole, setSyncRole] = useState(true)

  const [confirmRemove, setConfirmRemove] = useState<{
    open: boolean
    userId: string
    name: string
  }>({
    open: false,
    userId: '',
    name: '',
  })

  const isRegion = orgType === 'region'
  const roleType = isRegion
    ? 'region_admin'
    : 'school_admin'
  const roleLabel = isRegion
    ? '区域管理员'
    : '学校管理员'

  const syncEligible =
    addUserRole === 'operator' ||
    addUserRole === 'viewer'

  const noAvailableRegionDomain =
    isRegion && availableDomains.length === 0

  const addDisabled =
    adding ||
    !addUserId ||
    noAvailableRegionDomain ||
    (isRegion && !educationDomain)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError('')

      const result = await getOrgAdminManagement(orgId)

      setAdmins(result.admins)
      setAvailableDomains(
        result.available_education_domains
      )

      // 不自动选择第一项。
      // 若原选择已经不在后端允许范围内，则立即清空。
      setEducationDomain(current =>
        current &&
        result.available_education_domains.includes(current)
          ? current
          : ''
      )
    } catch (e: unknown) {
      setError(
        e instanceof Error
          ? e.message
          : '加载管理员失败'
      )
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    void load()
  }, [load])

  const handleAdd = useCallback(async () => {
    if (!addUserId) {
      setError('请先选择要任命的用户')
      return
    }

    if (isRegion && !educationDomain) {
      setError('请选择该负责人固定负责的教育类型')
      return
    }

    try {
      setAdding(true)
      setError('')
      setNotice('')

      const result = await addOrgAdmin(orgId, {
        user_id: addUserId,
        role_type: roleType,
        education_domain:
          isRegion && educationDomain
            ? educationDomain
            : undefined,
        sync_role: syncEligible && syncRole,
      })

      setNotice(
        result.message +
          (result.role_synced
            ? '（对方重新登录后生效）'
            : '')
      )

      setAddUserId('')
      setAddUserName('')
      setAddUserRole('')
      setEducationDomain('')
      setSyncRole(true)

      await load()
      onChanged?.()
    } catch (e: unknown) {
      setError(
        e instanceof Error ? e.message : '任命失败'
      )
    } finally {
      setAdding(false)
    }
  }, [
    addUserId,
    educationDomain,
    isRegion,
    load,
    onChanged,
    orgId,
    roleType,
    syncEligible,
    syncRole,
  ])

  const handleRemove = useCallback(
    async (userId: string) => {
      try {
        setError('')
        setNotice('')

        await removeOrgAdmin(orgId, userId)

        setNotice('移除成功')
        await load()
        onChanged?.()
      } catch (e: unknown) {
        setError(
          e instanceof Error ? e.message : '移除失败'
        )
      } finally {
        setConfirmRemove({
          open: false,
          userId: '',
          name: '',
        })
      }
    },
    [load, onChanged, orgId]
  )

  return (
    <div
      style={{
        padding: '16px',
        background: 'rgba(124,58,237,0.05)',
        borderTop: `1px dashed ${C.border}`,
      }}
    >
      {confirmRemove.open && (
        <ConfirmDialog
          title={`移除${roleLabel}`}
          message={
            `确认移除「${confirmRemove.name}」的` +
            `${roleLabel}任命？移除后该用户将无法继续管理本` +
            `${isRegion ? '区域' : '学校'}。` +
            '若这是该用户最后一个任命制管辖，系统会自动将其账户身份调整为骨干教师；' +
            '若仍有其它任命，账户身份保持不变。'
          }
          onConfirm={() =>
            handleRemove(confirmRemove.userId)
          }
          onCancel={() =>
            setConfirmRemove({
              open: false,
              userId: '',
              name: '',
            })
          }
        />
      )}

      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: '12px',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
          }}
        >
          <span
            style={{
              fontSize: '13px',
              fontWeight: 600,
              color: C.text,
            }}
          >
            🛡️ {roleLabel}管理
          </span>

          {admins.length > 0 && (
            <span
              style={{
                padding: '1px 7px',
                borderRadius: '10px',
                background: C.purpleLight,
                color: C.purple,
                fontSize: '11px',
                fontWeight: 600,
              }}
            >
              共 {admins.length} 人
            </span>
          )}
        </div>

        <button
          onClick={onClose}
          style={{
            border: 'none',
            background: 'none',
            color: C.textMuted,
            cursor: 'pointer',
            fontSize: '12px',
          }}
        >
          收起 ▲
        </button>
      </div>

      {error && (
        <div
          style={{
            marginBottom: '10px',
            padding: '8px 12px',
            borderRadius: '8px',
            background: C.dangerLight,
            color: C.danger,
            fontSize: '12px',
            lineHeight: 1.5,
          }}
        >
          {error}
        </div>
      )}

      {notice && (
        <div
          style={{
            marginBottom: '10px',
            padding: '8px 12px',
            borderRadius: '8px',
            background: C.successLight,
            color: C.success,
            fontSize: '12px',
            lineHeight: 1.5,
          }}
        >
          ✓ {notice}
        </div>
      )}

      {loading ? (
        <div
          style={{
            padding: '8px 0',
            color: C.textMuted,
            fontSize: '12px',
          }}
        >
          加载中...
        </div>
      ) : admins.length === 0 ? (
        <div
          style={{
            padding: '8px 0',
            color: C.textMuted,
            fontSize: '12px',
          }}
        >
          暂无{roleLabel}，请在下方添加
        </div>
      ) : (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: '6px',
            marginBottom: '14px',
          }}
        >
          {admins.map(admin => {
            const configured =
              admin.education_domain === 'k12' ||
              admin.education_domain ===
                'vocational' ||
              admin.education_domain === 'adult'

            return (
              <div
                key={`${admin.org_id}:${admin.user_id}`}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  padding: '8px 12px',
                  border: `1px solid ${C.border}`,
                  borderRadius: '8px',
                  background: C.white,
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: '28px',
                    height: '28px',
                    flexShrink: 0,
                    borderRadius: '50%',
                    background:
                      'linear-gradient(135deg,#7C3AED,#4F7BE8)',
                    color: '#fff',
                    fontSize: '11px',
                    fontWeight: 700,
                  }}
                >
                  {(admin.display_name || admin.username)
                    .charAt(0)
                    .toUpperCase()}
                </div>

                <div style={{ flex: 1, minWidth: 0 }}>
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      flexWrap: 'wrap',
                      gap: '6px',
                    }}
                  >
                    <span
                      style={{
                        color: C.text,
                        fontSize: '13px',
                        fontWeight: 600,
                      }}
                    >
                      {admin.display_name ||
                        admin.username}
                    </span>

                    <span
                      style={{
                        padding: '1px 6px',
                        border: `1px solid ${C.purple}44`,
                        borderRadius: '8px',
                        background: C.purpleLight,
                        color: C.purple,
                        fontSize: '10px',
                        fontWeight: 700,
                      }}
                    >
                      {roleLabel}
                    </span>

                    {isRegion && (
                      <span
                        style={{
                          padding: '1px 6px',
                          border: configured
                            ? '1px solid #A7F3D0'
                            : '1px solid #FECACA',
                          borderRadius: '8px',
                          background: configured
                            ? '#ECFDF5'
                            : '#FEF2F2',
                          color: configured
                            ? '#047857'
                            : '#DC2626',
                          fontSize: '10px',
                          fontWeight: 700,
                        }}
                      >
                        {getEducationDomainName(
                          admin.education_domain
                        )}
                      </span>
                    )}
                  </div>

                  <div
                    style={{
                      color: C.textMuted,
                      fontSize: '11px',
                    }}
                  >
                    @{admin.username} · 任命于{' '}
                    {fmt(admin.created_at)}
                  </div>
                </div>

                <button
                  onClick={() =>
                    setConfirmRemove({
                      open: true,
                      userId: admin.user_id,
                      name:
                        admin.display_name ||
                        admin.username,
                    })
                  }
                  style={{
                    padding: '4px 10px',
                    border: '1px solid #FEE2E2',
                    borderRadius: '6px',
                    background: '#FEF2F2',
                    color: '#EF4444',
                    cursor: 'pointer',
                    fontSize: '11px',
                    fontWeight: 500,
                    whiteSpace: 'nowrap',
                  }}
                >
                  移除
                </button>
              </div>
            )
          })}
        </div>
      )}

      <div
        style={{
          padding: '12px',
          border: `1px solid ${C.border}`,
          borderRadius: '10px',
          background: C.white,
        }}
      >
        <div
          style={{
            marginBottom: '10px',
            color: C.textSec,
            fontSize: '12px',
            fontWeight: 600,
          }}
        >
          任命{roleLabel}
        </div>

        <UserSearchPicker
          label=""
          value={addUserId}
          valueName={addUserName}
          onChange={(id, name, role) => {
            setAddUserId(id)
            setAddUserName(name)
            setAddUserRole(role || '')
            setEducationDomain('')
            setSyncRole(true)
          }}
          placeholder="输入用户名搜索要任命的用户..."
        />

        {isRegion && (
          <div style={{ marginBottom: '10px' }}>
            <label
              style={{
                display: 'block',
                marginBottom: '6px',
                color: C.textSec,
                fontSize: '12px',
                fontWeight: 600,
              }}
            >
              负责教育类型
              <span style={{ color: C.danger }}>
                {' '}*
              </span>
            </label>

            <select
              value={educationDomain}
              disabled={noAvailableRegionDomain}
              onChange={event =>
                setEducationDomain(
                  event.target.value as
                    | TeachingEducationDomain
                    | ''
                )
              }
              style={{
                width: '100%',
                padding: '8px 10px',
                border: `1px solid ${C.border}`,
                borderRadius: '7px',
                background: noAvailableRegionDomain
                  ? '#F3F4F6'
                  : C.white,
                color: educationDomain
                  ? C.text
                  : C.textMuted,
                cursor: noAvailableRegionDomain
                  ? 'not-allowed'
                  : 'pointer',
                fontSize: '13px',
              }}
            >
              <option value="">
                请选择负责教育类型
              </option>

              {availableDomains.map(domain => (
                <option key={domain} value={domain}>
                  {educationDomainNames[domain]}
                </option>
              ))}
            </select>

            {noAvailableRegionDomain ? (
              <div
                style={{
                  marginTop: '6px',
                  padding: '7px 9px',
                  border: '1px solid #FDE68A',
                  borderRadius: '7px',
                  background: '#FFFBEB',
                  color: '#92400E',
                  fontSize: '11px',
                  lineHeight: 1.5,
                }}
              >
                本区域下暂无已正确配置教育类型的有效学校，
                暂不能任命区域教育负责人。
              </div>
            ) : (
              <div
                style={{
                  marginTop: '5px',
                  color: C.textMuted,
                  fontSize: '11px',
                  lineHeight: 1.5,
                }}
              >
                这里只显示本区域实际存在的学校类型。
                任命后负责人固定属于该教育域，不提供切换。
              </div>
            )}
          </div>
        )}

        {addUserId && addUserRole && (
          <div
            style={{
              marginBottom: '10px',
              padding: '10px 12px',
              border: `1px solid ${C.border}`,
              borderRadius: '8px',
              background: C.bg,
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                marginBottom: syncEligible
                  ? '8px'
                  : '6px',
              }}
            >
              <span
                style={{
                  color: C.textSec,
                  fontSize: '12px',
                }}
              >
                当前身份：
              </span>
              <RoleBadge role={addUserRole} />
            </div>

            {syncEligible ? (
              <label
                style={{
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: '8px',
                  cursor: 'pointer',
                }}
              >
                <input
                  type="checkbox"
                  checked={syncRole}
                  onChange={event =>
                    setSyncRole(event.target.checked)
                  }
                  style={{
                    marginTop: '2px',
                    cursor: 'pointer',
                  }}
                />

                <span
                  style={{
                    color: C.text,
                    fontSize: '12px',
                    lineHeight: 1.5,
                  }}
                >
                  任命后同步升级账户身份为
                  「{roleLabel}」
                  <span style={{ color: C.textMuted }}>
                    （推荐勾选：否则对方登录后没有对应管理入口）
                  </span>
                </span>
              </label>
            ) : (
              <div
                style={{
                  color: C.textMuted,
                  fontSize: '12px',
                  lineHeight: 1.5,
                }}
              >
                任命只增加管辖范围，不改变其现有账户身份。
              </div>
            )}
          </div>
        )}

        <button
          onClick={handleAdd}
          disabled={addDisabled}
          style={{
            width: '100%',
            marginTop: '4px',
            padding: '8px',
            border: 'none',
            borderRadius: '7px',
            background: addDisabled
              ? '#E5E7EB'
              : C.purple,
            color: addDisabled
              ? '#9CA3AF'
              : '#fff',
            cursor: addDisabled
              ? 'not-allowed'
              : 'pointer',
            fontSize: '13px',
            fontWeight: 600,
          }}
        >
          {adding
            ? '任命中...'
            : `+ 任命为${roleLabel}`}
        </button>

        <div
          style={{
            marginTop: '8px',
            color: C.textMuted,
            fontSize: '11px',
            lineHeight: 1.5,
          }}
        >
          💡{' '}
          {isRegion
            ? '同一用户可以负责多个区域，但所有有效区域任命必须属于同一个教育域。'
            : '学校管理员的教育域直接继承学校，不需要单独选择。'}
          移除最后一个任命制管辖时，
          系统会自动将账户身份调整为骨干教师。
        </div>
      </div>
    </div>
  )
}
