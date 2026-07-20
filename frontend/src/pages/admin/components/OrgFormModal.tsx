/**
 * OrgFormModal.tsx — 区域/学校新建与编辑弹窗
 *
 * 上下文7：
 *   - 新建学校必须主动选择教育类型；
 *   - 初始值为空，不默认K12；
 *   - 仅允许中小学、职业教育、成人教育；
 *   - 区域表单不提供可编辑教育域；
 *   - 编辑学校只读显示创建时确定的教育类型。
 *
 * 其它既有能力：
 *   - 编辑学校门户板块配置；
 *   - 组织Logo上传与移除；
 *   - 主管理员选择。
 */
import { useEffect, useState } from 'react'
import {
  createAdminOrg,
  getAdminOrg,
  updateAdminOrg,
  uploadOrgLogo,
} from '@/api/admin'
import type {
  CreateOrgRequest,
  OrgListItem,
  TeachingEducationDomain,
  UpdateOrgRequest,
} from '@/api/admin'
import { C } from './adminConstants'
import { UserSearchPicker } from './UserSearchPicker'

interface OrgFormModalProps {
  mode: 'create' | 'edit'
  type: 'region' | 'school'
  initial?: OrgListItem
  regions: OrgListItem[]
  onClose: () => void
  onSaved: () => void
}

const PORTAL_MODULE_OPTIONS: {
  key: string
  label: string
  desc: string
}[] = [
  {
    key: 'lesson_plan',
    label: '📝 备课工坊',
    desc: 'AI辅助教案开发',
  },
  {
    key: 'courseware',
    label: '🎨 课件工坊',
    desc: 'AI辅助课件生成',
  },
  {
    key: 'workflow',
    label: '🖥️ 课件审核',
    desc: '课件质量评估·审核·验收',
  },
]

const ALL_MODULE_KEYS =
  PORTAL_MODULE_OPTIONS.map(option => option.key)

const EDUCATION_DOMAIN_OPTIONS: {
  value: TeachingEducationDomain
  label: string
  desc: string
}[] = [
  {
    value: 'k12',
    label: '中小学',
    desc: '义务教育与普通高中课程体系',
  },
  {
    value: 'vocational',
    label: '职业教育',
    desc: '中职、高职及职业技能课程体系',
  },
  {
    value: 'adult',
    label: '成人教育',
    desc: '成人学习、继续教育与培训体系',
  },
]

function isTeachingEducationDomain(
  value: string | undefined,
): value is TeachingEducationDomain {
  return value === 'k12' ||
    value === 'vocational' ||
    value === 'adult'
}

function educationDomainLabel(
  value: string | undefined,
): string {
  const matched =
    EDUCATION_DOMAIN_OPTIONS.find(
      option => option.value === value,
    )

  if (matched) {
    return matched.label
  }
  if (value === 'mixed') {
    return '跨域管理'
  }

  return '未配置'
}

function parsePortalModules(
  settings: string | undefined,
): Record<string, boolean> {
  const result: Record<string, boolean> = {}

  for (const key of ALL_MODULE_KEYS) {
    result[key] = true
  }

  if (!settings) {
    return result
  }

  try {
    const parsed = JSON.parse(settings)
    const modules = parsed?.portal_modules

    if (modules && typeof modules === 'object') {
      for (const key of ALL_MODULE_KEYS) {
        if (key in modules) {
          result[key] = modules[key] !== false
        }
      }
    }
  } catch {
    // 非法历史settings保持全部开启，不阻断组织编辑。
  }

  return result
}

function mergePortalModules(
  originalSettings: string | undefined,
  modules: Record<string, boolean>,
): string {
  let parsed: Record<string, unknown> = {}

  if (originalSettings) {
    try {
      const value = JSON.parse(originalSettings)
      if (value && typeof value === 'object') {
        parsed = value as Record<string, unknown>
      }
    } catch {
      parsed = {}
    }
  }

  parsed.portal_modules = modules
  return JSON.stringify(parsed)
}

export function OrgFormModal({
  mode,
  type,
  initial,
  regions,
  onClose,
  onSaved,
}: OrgFormModalProps) {
  const [name, setName] =
    useState(initial?.name || '')
  const [parentId, setParentId] =
    useState(initial?.parent_id || '')
  const [adminId, setAdminId] =
    useState(initial?.admin_user_id || '')
  const [adminName, setAdminName] =
    useState(initial?.admin_user_name || '')

  // 新建学校必须主动选择，不设置K12默认值。
  const [educationDomain, setEducationDomain] =
    useState<TeachingEducationDomain | ''>(() => {
      if (
        mode === 'edit' &&
        isTeachingEducationDomain(
          initial?.education_domain,
        )
      ) {
        return initial.education_domain
      }

      return ''
    })

  const [saving, setSaving] = useState(false)
  const [logoUrl, setLogoUrl] =
    useState(initial?.logo_url || '')
  const [logoUploading, setLogoUploading] =
    useState(false)
  const [clearLogo, setClearLogo] =
    useState(false)
  const [error, setError] = useState('')

  const showModuleConfig =
    type === 'school' && mode === 'edit'

  const [originalSettings, setOriginalSettings] =
    useState<string>('')
  const [modules, setModules] =
    useState<Record<string, boolean>>(() => {
      const initialModules: Record<string, boolean> = {}

      for (const key of ALL_MODULE_KEYS) {
        initialModules[key] = true
      }

      return initialModules
    })
  const [modulesLoading, setModulesLoading] =
    useState(false)

  useEffect(() => {
    if (!showModuleConfig || !initial?.id) {
      return
    }

    let cancelled = false
    setModulesLoading(true)

    getAdminOrg(initial.id)
      .then(fullOrganization => {
        if (cancelled) {
          return
        }

        const settings =
          fullOrganization.settings || ''

        setOriginalSettings(settings)
        setModules(
          parsePortalModules(settings),
        )

        if (
          isTeachingEducationDomain(
            fullOrganization.education_domain,
          )
        ) {
          setEducationDomain(
            fullOrganization.education_domain,
          )
        }
      })
      .catch(() => {
        // 读取失败时保留列表传入值与默认门户配置。
      })
      .finally(() => {
        if (!cancelled) {
          setModulesLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [
    showModuleConfig,
    initial?.id,
  ])

  const title = mode === 'create'
    ? (
      type === 'region'
        ? '新建区域'
        : '新建学校'
    )
    : (
      type === 'region'
        ? '编辑区域'
        : '编辑学校'
    )

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '10px 14px',
    borderRadius: '8px',
    border: `1px solid ${C.border}`,
    fontSize: '14px',
    outline: 'none',
    boxSizing: 'border-box',
  }

  const toggleModule = (key: string) => {
    setModules(previous => ({
      ...previous,
      [key]: !previous[key],
    }))
  }

  const handleSave = async () => {
    if (!name.trim()) {
      setError('请输入名称')
      return
    }

    if (type === 'school' && !parentId) {
      setError('请选择所属区域')
      return
    }

    if (
      type === 'school' &&
      mode === 'create' &&
      !educationDomain
    ) {
      setError('请选择教育类型')
      return
    }

    try {
      setSaving(true)
      setError('')

      if (mode === 'create') {
        const request: CreateOrgRequest = {
          name: name.trim(),
          type,
          parent_id:
            type === 'school'
              ? parentId
              : null,
          admin_user_id:
            adminId || null,
        }

        // 区域请求不发送education_domain，
        // 后端和数据库统一强制mixed。
        if (
          type === 'school' &&
          educationDomain
        ) {
          request.education_domain =
            educationDomain
        }

        await createAdminOrg(request)
      } else {
        const request: UpdateOrgRequest = {
          name: name.trim(),
          admin_user_id:
            adminId || null,
        }

        if (showModuleConfig) {
          request.settings =
            mergePortalModules(
              originalSettings,
              modules,
            )
        }

        if (clearLogo) {
          request.clear_logo = true
        }

        await updateAdminOrg(
          initial!.id,
          request,
        )
      }

      onSaved()
      onClose()
    } catch (caught: unknown) {
      setError(
        caught instanceof Error
          ? caught.message
          : '操作失败',
      )
    } finally {
      setSaving(false)
    }
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 10500,
        background: 'rgba(0,0,0,0.4)',
        backdropFilter: 'blur(4px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
      onClick={event => {
        if (event.target === event.currentTarget) {
          onClose()
        }
      }}
    >
      <div
        style={{
          background: C.white,
          borderRadius: '20px',
          width: '480px',
          maxHeight: '90vh',
          overflowY: 'auto',
          boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
        }}
      >
        <div
          style={{
            padding: '20px 24px',
            borderBottom: `1px solid ${C.border}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            position: 'sticky',
            top: 0,
            background: C.white,
            zIndex: 1,
          }}
        >
          <div
            style={{
              fontSize: '16px',
              fontWeight: 700,
              color: C.text,
            }}
          >
            {title}
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
                marginBottom: '14px',
                background: C.dangerLight,
                color: C.danger,
                fontSize: '13px',
              }}
            >
              {error}
            </div>
          )}

          <div style={{ marginBottom: '14px' }}>
            <label
              style={{
                display: 'block',
                fontSize: '13px',
                fontWeight: 600,
                color: C.text,
                marginBottom: '6px',
              }}
            >
              {type === 'region'
                ? '区域名称'
                : '学校名称'}
              {' '}
              <span style={{ color: C.danger }}>
                *
              </span>
            </label>

            <input
              value={name}
              onChange={event =>
                setName(event.target.value)}
              placeholder="请输入名称"
              style={inputStyle}
              onFocus={event => {
                event.currentTarget.style.borderColor =
                  C.primary
              }}
              onBlur={event => {
                event.currentTarget.style.borderColor =
                  C.border
              }}
            />
          </div>

          {type === 'school' &&
            mode === 'create' && (
            <div style={{ marginBottom: '14px' }}>
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
                {' '}
                <span style={{ color: C.danger }}>
                  *
                </span>
              </label>

              <select
                value={parentId}
                onChange={event =>
                  setParentId(event.target.value)}
                style={{
                  ...inputStyle,
                  background: C.white,
                }}
              >
                <option value="">
                  请选择区域
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
          )}

          {type === 'school' &&
            mode === 'create' && (
            <div style={{ marginBottom: '14px' }}>
              <label
                style={{
                  display: 'block',
                  fontSize: '13px',
                  fontWeight: 600,
                  color: C.text,
                  marginBottom: '6px',
                }}
              >
                教育类型
                {' '}
                <span style={{ color: C.danger }}>
                  *
                </span>
              </label>

              <select
                value={educationDomain}
                onChange={event =>
                  setEducationDomain(
                    event.target.value as
                      TeachingEducationDomain | '',
                  )}
                style={{
                  ...inputStyle,
                  background: C.white,
                }}
              >
                <option value="">
                  请选择教育类型
                </option>

                {EDUCATION_DOMAIN_OPTIONS.map(
                  option => (
                    <option
                      key={option.value}
                      value={option.value}
                    >
                      {option.label}
                    </option>
                  ),
                )}
              </select>

              <div
                style={{
                  fontSize: '11px',
                  color: C.textMuted,
                  marginTop: '5px',
                  lineHeight: 1.5,
                }}
              >
                创建后将作为本校课程、教案和课件的教育域基础。
                当前步骤不允许选择“跨域管理”。
              </div>

              {educationDomain && (
                <div
                  style={{
                    marginTop: '7px',
                    padding: '8px 10px',
                    borderRadius: '8px',
                    background: C.primaryLight,
                    color: C.primary,
                    fontSize: '12px',
                  }}
                >
                  {
                    EDUCATION_DOMAIN_OPTIONS.find(
                      option =>
                        option.value ===
                        educationDomain,
                    )?.desc
                  }
                </div>
              )}
            </div>
          )}

          {type === 'school' &&
            mode === 'edit' && (
            <div style={{ marginBottom: '14px' }}>
              <label
                style={{
                  display: 'block',
                  fontSize: '13px',
                  fontWeight: 600,
                  color: C.text,
                  marginBottom: '6px',
                }}
              >
                教育类型
              </label>

              <div
                style={{
                  padding: '10px 14px',
                  borderRadius: '8px',
                  border: `1px solid ${C.border}`,
                  background: C.bg,
                  fontSize: '14px',
                  color: educationDomain
                    ? C.text
                    : C.danger,
                  fontWeight: 600,
                }}
              >
                {educationDomainLabel(
                  educationDomain ||
                    initial?.education_domain,
                )}
              </div>

              <div
                style={{
                  fontSize: '11px',
                  color: C.textMuted,
                  marginTop: '5px',
                }}
              >
                教育类型在创建学校时确定，本表单仅只读展示。
              </div>
            </div>
          )}

          {type === 'region' && (
            <div
              style={{
                marginBottom: '14px',
                padding: '10px 12px',
                borderRadius: '8px',
                background: C.bg,
                border: `1px solid ${C.border}`,
                fontSize: '12px',
                color: C.textSec,
                lineHeight: 1.5,
              }}
            >
              区域属于跨域管理组织，教育域由系统固定为
              <strong> mixed</strong>，不提供手动选择。
            </div>
          )}

          {(mode === 'edit' ||
            type === 'school') && (
            <div style={{ marginBottom: '14px' }}>
              <label
                style={{
                  display: 'block',
                  fontSize: '13px',
                  fontWeight: 600,
                  color: C.text,
                  marginBottom: '6px',
                }}
              >
                {type === 'region'
                  ? '区域Logo'
                  : '学校Logo'}
                （可选）
              </label>

              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '12px',
                }}
              >
                {logoUrl ? (
                  <img
                    src={logoUrl}
                    alt="Logo"
                    style={{
                      width: 48,
                      height: 48,
                      objectFit: 'contain',
                      borderRadius: '8px',
                      border: `1px solid ${C.border}`,
                    }}
                  />
                ) : (
                  <div
                    style={{
                      width: 48,
                      height: 48,
                      borderRadius: '8px',
                      border: `2px dashed ${C.border}`,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: '20px',
                      color: C.textMuted,
                    }}
                  >
                    🖼️
                  </div>
                )}

                <label
                  style={{
                    padding: '6px 14px',
                    borderRadius: '8px',
                    border: `1px solid ${C.border}`,
                    background: C.bg,
                    fontSize: '13px',
                    color: C.text,
                    cursor: logoUploading
                      ? 'default'
                      : 'pointer',
                  }}
                >
                  {logoUploading
                    ? '上传中...'
                    : logoUrl
                      ? '更换Logo'
                      : '上传Logo'}

                  <input
                    type="file"
                    accept="image/jpeg,image/png,image/webp,image/svg+xml"
                    style={{ display: 'none' }}
                    disabled={logoUploading}
                    onChange={async event => {
                      const file =
                        event.target.files?.[0]

                      if (!file) {
                        return
                      }

                      if (
                        file.size >
                        2 * 1024 * 1024
                      ) {
                        setError(
                          'Logo文件不能超过2MB',
                        )
                        return
                      }

                      if (
                        mode === 'edit' &&
                        initial?.id
                      ) {
                        try {
                          setLogoUploading(true)

                          const result =
                            await uploadOrgLogo(
                              initial.id,
                              file,
                            )

                          setLogoUrl(result.url)
                          setClearLogo(false)
                        } catch (caught) {
                          setError(
                            caught instanceof Error
                              ? caught.message
                              : '上传失败',
                          )
                        } finally {
                          setLogoUploading(false)
                        }
                      } else {
                        const reader =
                          new FileReader()

                        reader.onload = () =>
                          setLogoUrl(
                            reader.result as string,
                          )

                        reader.readAsDataURL(file)

                        setError(
                          '提示：Logo将在创建组织后可上传，请先创建再编辑上传Logo',
                        )
                      }

                      event.target.value = ''
                    }}
                  />
                </label>

                {logoUrl && (
                  <button
                    onClick={() => {
                      setLogoUrl('')

                      if (mode === 'edit') {
                        setClearLogo(true)
                      }
                    }}
                    style={{
                      padding: '4px 10px',
                      borderRadius: '6px',
                      border: `1px solid ${C.border}`,
                      background: 'transparent',
                      fontSize: '12px',
                      color: C.textMuted,
                      cursor: 'pointer',
                    }}
                  >
                    移除
                  </button>
                )}
              </div>

              {clearLogo &&
                mode === 'edit' &&
                !logoUrl && (
                <div
                  style={{
                    fontSize: '11px',
                    color: C.danger,
                    marginTop: '4px',
                  }}
                >
                  Logo将在点击“保存”后移除。
                </div>
              )}

              <div
                style={{
                  fontSize: '11px',
                  color: C.textMuted,
                  marginTop: '4px',
                }}
              >
                支持JPG/PNG/WEBP/SVG，最大2MB。
              </div>
            </div>
          )}

          {showModuleConfig && (
            <div style={{ marginBottom: '14px' }}>
              <label
                style={{
                  display: 'block',
                  fontSize: '13px',
                  fontWeight: 600,
                  color: C.text,
                  marginBottom: '6px',
                }}
              >
                门户可见板块
              </label>

              <div
                style={{
                  fontSize: '11px',
                  color: C.textMuted,
                  marginBottom: '8px',
                }}
              >
                勾选本校老师在首页能进入的工作区。
                系统管理员不受此限制。
              </div>

              {modulesLoading ? (
                <div
                  style={{
                    fontSize: '12px',
                    color: C.textMuted,
                    padding: '8px 0',
                  }}
                >
                  正在读取当前配置...
                </div>
              ) : (
                <div
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '8px',
                  }}
                >
                  {PORTAL_MODULE_OPTIONS.map(
                    option => {
                      const checked =
                        modules[option.key] !== false

                      return (
                        <label
                          key={option.key}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '10px',
                            padding: '10px 12px',
                            borderRadius: '8px',
                            border:
                              `1px solid ${
                                checked
                                  ? C.primary
                                  : C.border
                              }`,
                            background:
                              checked
                                ? C.bg
                                : C.white,
                            cursor: 'pointer',
                          }}
                        >
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={() =>
                              toggleModule(
                                option.key,
                              )}
                            style={{
                              width: 16,
                              height: 16,
                              cursor: 'pointer',
                              accentColor: C.primary,
                            }}
                          />

                          <div style={{ flex: 1 }}>
                            <div
                              style={{
                                fontSize: '13px',
                                fontWeight: 600,
                                color: C.text,
                              }}
                            >
                              {option.label}
                            </div>

                            <div
                              style={{
                                fontSize: '11px',
                                color: C.textMuted,
                              }}
                            >
                              {option.desc}
                            </div>
                          </div>
                        </label>
                      )
                    },
                  )}
                </div>
              )}
            </div>
          )}

          <UserSearchPicker
            label="管理员（可选）"
            value={adminId}
            valueName={adminName}
            onChange={(id, selectedName) => {
              setAdminId(id)
              setAdminName(selectedName)
            }}
            placeholder="搜索并选择管理员用户..."
          />

          <div
            style={{
              display: 'flex',
              gap: '10px',
              marginTop: '4px',
            }}
          >
            <button
              onClick={onClose}
              style={{
                flex: 1,
                padding: '10px',
                borderRadius: '10px',
                border: `1px solid ${C.border}`,
                background: C.bg,
                fontSize: '14px',
                color: C.textSec,
                cursor: 'pointer',
              }}
            >
              取消
            </button>

            <button
              onClick={handleSave}
              disabled={saving}
              style={{
                flex: 2,
                padding: '10px',
                borderRadius: '10px',
                border: 'none',
                background: saving
                  ? C.textMuted
                  : `linear-gradient(135deg,${C.primary},#7C3AED)`,
                color: '#fff',
                fontSize: '14px',
                fontWeight: 600,
                cursor: saving
                  ? 'not-allowed'
                  : 'pointer',
              }}
            >
              {saving
                ? '保存中...'
                : mode === 'create'
                  ? '创建'
                  : '保存'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
