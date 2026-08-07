/**
 * CreateUserModal.tsx — 管理中心单用户创建弹窗
 *
 * 系统管理员建号：
 *   - 创建骨干教师(operator)或普通教师(viewer)时，必须先选区域，再选区域下学校；
 *   - 创建平台管理员(admin)时不显示学校归属选择；
 *   - 学校列表只展示启用且教育域为k12/vocational/adult的学校。
 *
 * 学校管理员建号：
 *   - 不显示区域、学校选择；
 *   - 后端忽略任何客户端学校参数，强制归入其真实管理学校。
 *
 * 任命制身份：
 *   - senior_operator与region_admin不能在此直接创建；
 *   - 先创建普通教师，再到组织架构管理员面板任命。
 *
 * 安全边界：
 *   前端选择只改善操作体验。后端会再次校验学校真实性、状态、教育域，
 *   并将users、school_members、personal token_accounts放入同一事务。
 */

import {
  useEffect,
  useMemo,
  useState,
} from 'react'

import {
  createAdminUserWithSchool,
  listActiveAdminRegions,
  listActiveAdminSchoolsByRegion,
  type AdminCreateOrganizationOption,
} from '@/api/adminUserCreate'
import { useAuth } from '@/store/auth'

import {
  APPOINTMENT_ONLY_ROLES,
  C,
  ROLE_OPTIONS,
} from './adminConstants'

interface CreateUserModalProps {
  onClose: () => void
  onCreated: () => void
}

interface CreateUserForm {
  username: string
  display_name: string
  password: string
  role: string
}

export function CreateUserModal({
  onClose,
  onCreated,
}: CreateUserModalProps) {
  const { user } = useAuth()

  const [form, setForm] = useState<CreateUserForm>({
    username: '',
    display_name: '',
    password: '',
    role: 'operator',
  })

  const [regions, setRegions] =
    useState<AdminCreateOrganizationOption[]>([])
  const [schools, setSchools] =
    useState<AdminCreateOrganizationOption[]>([])

  const [regionId, setRegionId] = useState('')
  const [schoolId, setSchoolId] = useState('')

  const [loadingRegions, setLoadingRegions] =
    useState(false)
  const [loadingSchools, setLoadingSchools] =
    useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const isAdmin = user?.role === 'admin'

  const isTeachingRole =
    form.role === 'operator' ||
    form.role === 'viewer'

  const needsSchoolSelection =
    isAdmin &&
    isTeachingRole

  /**
   * 可创建角色：
   *   - admin：平台管理员、骨干教师、普通教师；
   *   - senior_operator：骨干教师、普通教师；
   *   - region_admin等其它角色：无创建权限。
   */
  const availableRoles = useMemo(() => {
    const allRoles = ROLE_OPTIONS.filter(option =>
      Boolean(option.value) &&
      option.value !== 'district_inspector' &&
      !APPOINTMENT_ONLY_ROLES.includes(option.value)
    )

    if (user?.role === 'admin') {
      return allRoles
    }

    if (user?.role === 'senior_operator') {
      return allRoles.filter(option =>
        option.value === 'operator' ||
        option.value === 'viewer'
      )
    }

    return []
  }, [user?.role])

  const noCreatableRole =
    availableRoles.length === 0

  /**
   * 系统管理员打开弹窗时加载区域。
   * 不自动选中，要求管理员明确确认目标区域，避免误归属。
   */
  useEffect(() => {
    if (!isAdmin) {
      setRegions([])
      setRegionId('')
      return
    }

    let alive = true

    const loadRegions = async () => {
      setLoadingRegions(true)

      try {
        const items =
          await listActiveAdminRegions()

        if (alive) {
          setRegions(items)
        }
      } catch (caught: unknown) {
        if (alive) {
          setRegions([])
          setError(
            caught instanceof Error
              ? caught.message
              : '加载区域失败',
          )
        }
      } finally {
        if (alive) {
          setLoadingRegions(false)
        }
      }
    }

    void loadRegions()

    return () => {
      alive = false
    }
  }, [isAdmin])

  /**
   * 系统管理员选择区域后加载该区域下的启用教学学校。
   */
  useEffect(() => {
    setSchoolId('')
    setSchools([])

    if (
      !needsSchoolSelection ||
      regionId === ''
    ) {
      setLoadingSchools(false)
      return
    }

    let alive = true

    const loadSchools = async () => {
      setLoadingSchools(true)

      try {
        const items =
          await listActiveAdminSchoolsByRegion(
            regionId,
          )

        if (alive) {
          setSchools(items)
        }
      } catch (caught: unknown) {
        if (alive) {
          setSchools([])
          setError(
            caught instanceof Error
              ? caught.message
              : '加载学校失败',
          )
        }
      } finally {
        if (alive) {
          setLoadingSchools(false)
        }
      }
    }

    void loadSchools()

    return () => {
      alive = false
    }
  }, [
    needsSchoolSelection,
    regionId,
  ])

  /**
   * 从教学角色切换为平台管理员时，清空本地学校选择。
   * 后端也会拒绝平台管理账号携带普通学校校籍。
   */
  useEffect(() => {
    if (!needsSchoolSelection) {
      setRegionId('')
      setSchoolId('')
      setSchools([])
    }
  }, [needsSchoolSelection])

  const handleRoleChange = (
    role: string,
  ) => {
    setError('')
    setForm(previous => ({
      ...previous,
      role,
    }))
  }

  const handleCreate = async () => {
    if (noCreatableRole) {
      setError('当前角色无权创建用户')
      return
    }

    if (
      form.username.trim() === '' ||
      form.display_name.trim() === '' ||
      form.password.length < 6
    ) {
      setError(
        '请填写完整信息（密码至少6位）',
      )
      return
    }

    if (
      needsSchoolSelection &&
      regionId === ''
    ) {
      setError('请先选择所属区域')
      return
    }

    if (
      needsSchoolSelection &&
      schoolId === ''
    ) {
      setError('请选择所属学校')
      return
    }

    try {
      setSaving(true)
      setError('')

      await createAdminUserWithSchool({
        username: form.username.trim(),
        display_name:
          form.display_name.trim(),
        password: form.password,
        role: form.role,
        school_id:
          needsSchoolSelection
            ? schoolId
            : undefined,
      })

      onCreated()
      onClose()
    } catch (caught: unknown) {
      setError(
        caught instanceof Error
          ? caught.message
          : '创建失败',
      )
    } finally {
      setSaving(false)
    }
  }

  const fields = [
    {
      key: 'username',
      label: '登录用户名',
      placeholder: '字母、数字或下划线',
      type: 'text',
    },
    {
      key: 'display_name',
      label: '显示名称',
      placeholder: '例如：张老师',
      type: 'text',
    },
    {
      key: 'password',
      label: '初始密码',
      placeholder: '至少6位',
      type: 'password',
    },
  ]

  const commonSelectStyle:
  React.CSSProperties = {
    width: '100%',
    padding: '10px 14px',
    borderRadius: '10px',
    border: `1px solid ${C.border}`,
    fontSize: '14px',
    outline: 'none',
    boxSizing: 'border-box',
    background: C.white,
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 10000,
        background: 'rgba(0,0,0,0.4)',
        backdropFilter: 'blur(4px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '20px',
      }}
      onClick={event => {
        if (
          event.target ===
          event.currentTarget
        ) {
          onClose()
        }
      }}
    >
      <div
        style={{
          background: C.white,
          borderRadius: '20px',
          width: '520px',
          maxWidth: '100%',
          maxHeight: '90vh',
          overflowY: 'auto',
          boxShadow:
            '0 20px 60px rgba(0,0,0,0.2)',
        }}
      >
        <div
          style={{
            padding: '20px 24px',
            borderBottom:
              `1px solid ${C.border}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <div
            style={{
              fontSize: '16px',
              fontWeight: 700,
              color: C.text,
            }}
          >
            新建用户
          </div>

          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              fontSize: '20px',
              color: C.textMuted,
            }}
          >
            ×
          </button>
        </div>

        <div style={{ padding: '24px' }}>
          {error && (
            <div
              style={{
                padding: '10px 14px',
                borderRadius: '8px',
                marginBottom: '16px',
                background: C.dangerLight,
                color: C.danger,
                fontSize: '13px',
                lineHeight: 1.6,
              }}
            >
              {error}
            </div>
          )}

          {user?.role ===
            'senior_operator' && (
            <div
              style={{
                padding: '10px 14px',
                borderRadius: '8px',
                marginBottom: '16px',
                background: C.primaryLight,
                color: C.primary,
                fontSize: '12px',
                lineHeight: 1.6,
              }}
            >
              💡 新建教师将由后端自动归入您管理的学校，无需手动选择区域和学校。
            </div>
          )}

          {isAdmin && (
            <div
              style={{
                padding: '10px 14px',
                borderRadius: '8px',
                marginBottom: '16px',
                background: C.warningLight,
                color: C.warning,
                fontSize: '12px',
                lineHeight: 1.6,
              }}
            >
              💡 创建教师账号时必须选择区域和学校。学校管理员、区域管理员仍通过“组织架构 → 🛡️ 管理员”完成任命。
            </div>
          )}

          {noCreatableRole && (
            <div
              style={{
                padding: '10px 14px',
                borderRadius: '8px',
                marginBottom: '16px',
                background: C.warningLight,
                color: C.warning,
                fontSize: '12px',
              }}
            >
              ⚠️ 当前角色无权创建用户。
            </div>
          )}

          {fields.map(field => (
            <div
              key={field.key}
              style={{
                marginBottom: '14px',
              }}
            >
              <label
                style={{
                  display: 'block',
                  fontSize: '13px',
                  fontWeight: 600,
                  color: C.text,
                  marginBottom: '6px',
                }}
              >
                {field.label}
              </label>

              <input
                type={field.type}
                value={
                  (
                    form as unknown as
                    Record<string, string>
                  )[field.key]
                }
                onChange={event => {
                  setError('')
                  setForm(previous => ({
                    ...previous,
                    [field.key]:
                      event.target.value,
                  }))
                }}
                placeholder={
                  field.placeholder
                }
                style={{
                  width: '100%',
                  padding: '10px 14px',
                  borderRadius: '10px',
                  border:
                    `1px solid ${C.border}`,
                  fontSize: '14px',
                  outline: 'none',
                  boxSizing: 'border-box',
                }}
                onFocus={event => {
                  event.currentTarget.style
                    .borderColor = C.primary
                }}
                onBlur={event => {
                  event.currentTarget.style
                    .borderColor = C.border
                }}
              />
            </div>
          ))}

          <div
            style={{
              marginBottom: '14px',
            }}
          >
            <label
              style={{
                display: 'block',
                fontSize: '13px',
                fontWeight: 600,
                color: C.text,
                marginBottom: '6px',
              }}
            >
              系统角色
            </label>

            <select
              value={form.role}
              onChange={event =>
                handleRoleChange(
                  event.target.value,
                )
              }
              disabled={noCreatableRole}
              style={commonSelectStyle}
            >
              {availableRoles.map(option => (
                <option
                  key={option.value}
                  value={option.value}
                >
                  {option.label}
                </option>
              ))}
            </select>
          </div>

          {needsSchoolSelection && (
            <div
              style={{
                padding: '16px',
                borderRadius: '12px',
                background: C.bg,
                border:
                  `1px solid ${C.border}`,
                marginBottom: '20px',
              }}
            >
              <div
                style={{
                  fontSize: '13px',
                  fontWeight: 700,
                  color: C.text,
                  marginBottom: '12px',
                }}
              >
                教学归属
              </div>

              <div
                style={{
                  marginBottom: '12px',
                }}
              >
                <label
                  style={{
                    display: 'block',
                    fontSize: '13px',
                    fontWeight: 600,
                    color: C.text,
                    marginBottom: '6px',
                  }}
                >
                  所属区域
                  <span
                    style={{
                      color: C.danger,
                      marginLeft: '3px',
                    }}
                  >
                    *
                  </span>
                </label>

                <select
                  value={regionId}
                  onChange={event => {
                    setError('')
                    setRegionId(
                      event.target.value,
                    )
                  }}
                  disabled={loadingRegions}
                  style={commonSelectStyle}
                >
                  <option value="">
                    {loadingRegions
                      ? '正在加载区域…'
                      : '请选择区域'}
                  </option>

                  {regions.map(region => (
                    <option
                      key={region.id}
                      value={region.id}
                    >
                      {region.name}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label
                  style={{
                    display: 'block',
                    fontSize: '13px',
                    fontWeight: 600,
                    color: C.text,
                    marginBottom: '6px',
                  }}
                >
                  所属学校
                  <span
                    style={{
                      color: C.danger,
                      marginLeft: '3px',
                    }}
                  >
                    *
                  </span>
                </label>

                <select
                  value={schoolId}
                  onChange={event => {
                    setError('')
                    setSchoolId(
                      event.target.value,
                    )
                  }}
                  disabled={
                    regionId === '' ||
                    loadingSchools
                  }
                  style={commonSelectStyle}
                >
                  <option value="">
                    {regionId === ''
                      ? '请先选择区域'
                      : loadingSchools
                        ? '正在加载学校…'
                        : schools.length === 0
                          ? '该区域暂无可用学校'
                          : '请选择学校'}
                  </option>

                  {schools.map(school => (
                    <option
                      key={school.id}
                      value={school.id}
                    >
                      {school.name}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          )}

          <div
            style={{
              display: 'flex',
              gap: '10px',
            }}
          >
            <button
              onClick={onClose}
              style={{
                flex: 1,
                padding: '10px',
                borderRadius: '10px',
                border:
                  `1px solid ${C.border}`,
                background: C.bg,
                fontSize: '14px',
                color: C.textSec,
                cursor: 'pointer',
              }}
            >
              取消
            </button>

            <button
              onClick={() => {
                void handleCreate()
              }}
              disabled={
                saving ||
                noCreatableRole
              }
              style={{
                flex: 2,
                padding: '10px',
                borderRadius: '10px',
                border: 'none',
                background:
                  (
                    saving ||
                    noCreatableRole
                  )
                    ? C.textMuted
                    : `linear-gradient(135deg,${C.primary},#7C3AED)`,
                color: '#fff',
                fontSize: '14px',
                fontWeight: 600,
                cursor:
                  (
                    saving ||
                    noCreatableRole
                  )
                    ? 'not-allowed'
                    : 'pointer',
              }}
            >
              {saving
                ? '创建中…'
                : '创建用户'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
