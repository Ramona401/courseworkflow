/**
 * OrgFormModal.tsx — 区域/学校 新建/编辑弹窗
 *
 * v172新增：编辑学校时，可勾选"门户可见板块"（备课/课件/审核三选项）。
 *   - 仅 school 类型、edit 模式 显示该区块（区域不挂用户，无需配置）
 *   - 打开时调 getAdminOrg(id) 读取该组织完整 settings（列表项不含 settings）
 *   - 保存时把 portal_modules 合并进原 settings 一并提交，不丢失其它已有配置
 *   - 缺省（从未配置）视为三板块全开
 */
import { useState, useEffect } from 'react'
import { createAdminOrg, updateAdminOrg, uploadOrgLogo, getAdminOrg } from '@/api/admin'
import type { OrgListItem, CreateOrgRequest, UpdateOrgRequest } from '@/api/admin'
import { C } from './adminConstants'
import { UserSearchPicker } from './UserSearchPicker'

interface OrgFormModalProps {
  mode: 'create' | 'edit'
  type: 'region' | 'school'
  initial?: OrgListItem
  regions: OrgListItem[]          // 新建学校时选择所属区域用
  onClose: () => void
  onSaved: () => void
}

/* 门户板块定义（key 必须与 PortalPage entries 的 key 一致） */
const PORTAL_MODULE_OPTIONS: { key: string; label: string; desc: string }[] = [
  { key: 'lesson_plan', label: '📝 备课工坊', desc: 'AI辅助教案开发' },
  { key: 'courseware',  label: '🎨 课件工坊', desc: 'AI辅助课件生成' },
  { key: 'workflow',    label: '🖥️ 课件审核', desc: '课件质量评估·审核·验收' },
]

const ALL_MODULE_KEYS = PORTAL_MODULE_OPTIONS.map(o => o.key)

/**
 * 从 settings 字符串解析 portal_modules，缺省/缺 key 一律按 true。
 * 返回一个三 key 齐全的 map，便于复选框直接绑定。
 */
function parsePortalModules(settings: string | undefined): Record<string, boolean> {
  const result: Record<string, boolean> = {}
  for (const k of ALL_MODULE_KEYS) result[k] = true // 默认全开
  if (!settings) return result
  try {
    const obj = JSON.parse(settings)
    const pm = obj?.portal_modules
    if (pm && typeof pm === 'object') {
      for (const k of ALL_MODULE_KEYS) {
        if (k in pm) result[k] = pm[k] !== false
      }
    }
  } catch {
    // 解析失败 → 保持全开
  }
  return result
}

/**
 * 把 portal_modules 合并进原 settings，序列化返回。
 * 保留原 settings 里的其它键，仅覆盖 portal_modules。
 */
function mergePortalModules(originalSettings: string | undefined, modules: Record<string, boolean>): string {
  let obj: Record<string, unknown> = {}
  if (originalSettings) {
    try {
      const parsed = JSON.parse(originalSettings)
      if (parsed && typeof parsed === 'object') obj = parsed as Record<string, unknown>
    } catch {
      obj = {}
    }
  }
  obj.portal_modules = modules
  return JSON.stringify(obj)
}

export function OrgFormModal({
  mode, type, initial, regions, onClose, onSaved,
}: OrgFormModalProps) {
  const [name, setName]           = useState(initial?.name || '')
  const [parentId, setParentId]   = useState(initial?.parent_id || '')
  const [adminId, setAdminId]     = useState(initial?.admin_user_id || '')
  const [adminName, setAdminName] = useState(initial?.admin_user_name || '')
  const [saving, setSaving]       = useState(false)
  const [logoUrl, setLogoUrl]     = useState(initial?.logo_url || '')
  const [logoUploading, setLogoUploading] = useState(false)
  const [error, setError]         = useState('')

  // v172：门户板块开关相关状态
  // 是否显示板块配置区：仅 school + edit
  const showModuleConfig = type === 'school' && mode === 'edit'
  const [originalSettings, setOriginalSettings] = useState<string>('') // 该组织原始 settings（保存时合并用）
  const [modules, setModules] = useState<Record<string, boolean>>(() => {
    const init: Record<string, boolean> = {}
    for (const k of ALL_MODULE_KEYS) init[k] = true
    return init
  })
  const [modulesLoading, setModulesLoading] = useState(false)

  // 打开时（编辑学校）拉取完整 settings，初始化复选框
  useEffect(() => {
    if (!showModuleConfig || !initial?.id) return
    let cancelled = false
    setModulesLoading(true)
    getAdminOrg(initial.id)
      .then(full => {
        if (cancelled) return
        const settings = full.settings || ''
        setOriginalSettings(settings)
        setModules(parsePortalModules(settings))
      })
      .catch(() => {
        // 读取失败时保持默认全开，不阻塞编辑
      })
      .finally(() => {
        if (!cancelled) setModulesLoading(false)
      })
    return () => { cancelled = true }
  }, [showModuleConfig, initial?.id])

  const title = mode === 'create'
    ? (type === 'region' ? '新建区域' : '新建学校')
    : (type === 'region' ? '编辑区域' : '编辑学校')

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: '8px',
    border: `1px solid ${C.border}`, fontSize: '14px',
    outline: 'none', boxSizing: 'border-box',
  }

  const toggleModule = (key: string) => {
    setModules(prev => ({ ...prev, [key]: !prev[key] }))
  }

  const handleSave = async () => {
    if (!name.trim()) { setError('请输入名称'); return }
    if (type === 'school' && !parentId) { setError('请选择所属区域'); return }
    try {
      setSaving(true); setError('')
      if (mode === 'create') {
        const req: CreateOrgRequest = {
          name: name.trim(), type,
          parent_id: type === 'school' ? parentId : null,
          admin_user_id: adminId || null,
        }
        await createAdminOrg(req)
      } else {
        const req: UpdateOrgRequest = { name: name.trim(), admin_user_id: adminId || null }
        // v172：编辑学校时，把板块开关合并进 settings 一并提交
        if (showModuleConfig) {
          req.settings = mergePortalModules(originalSettings, modules)
        }
        await updateAdminOrg(initial!.id, req)
      }
      onSaved(); onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '操作失败')
    } finally { setSaving(false) }
  }

  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 10500,
        background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{
        background: C.white, borderRadius: '20px', width: '480px',
        maxHeight: '90vh', overflowY: 'auto', boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }}>
        {/* 头部 */}
        <div style={{
          padding: '20px 24px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          position: 'sticky', top: 0, background: C.white, zIndex: 1,
        }}>
          <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>{title}</div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
        </div>

        {/* 表单 */}
        <div style={{ padding: '24px' }}>
          {error && (
            <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '14px', background: C.dangerLight, color: C.danger, fontSize: '13px' }}>
              {error}
            </div>
          )}

          {/* 名称 */}
          <div style={{ marginBottom: '14px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
              {type === 'region' ? '区域名称' : '学校名称'} <span style={{ color: C.danger }}>*</span>
            </label>
            <input
              value={name} onChange={e => setName(e.target.value)} placeholder="请输入名称"
              style={inputStyle}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border }}
            />
          </div>

          {/* 所属区域（新建学校时显示）*/}
          {type === 'school' && mode === 'create' && (
            <div style={{ marginBottom: '14px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                所属区域 <span style={{ color: C.danger }}>*</span>
              </label>
              <select
                value={parentId} onChange={e => setParentId(e.target.value)}
                style={{ ...inputStyle, background: C.white }}>
                <option value="">请选择区域</option>
                {regions.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
              </select>
            </div>
          )}

          {/* Logo上传（编辑模式 或 创建学校时显示） */}
          {(mode === 'edit' || type === 'school') && (
            <div style={{ marginBottom: '14px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                {type === 'region' ? '区域Logo' : '学校Logo'}（可选）
              </label>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                {logoUrl ? (
                  <img src={logoUrl} alt="Logo" style={{ width: 48, height: 48, objectFit: 'contain', borderRadius: '8px', border: `1px solid ${C.border}` }} />
                ) : (
                  <div style={{ width: 48, height: 48, borderRadius: '8px', border: `2px dashed ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '20px', color: C.textMuted }}>🖼️</div>
                )}
                <label style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '13px', color: C.text, cursor: logoUploading ? 'default' : 'pointer' }}>
                  {logoUploading ? '上传中...' : logoUrl ? '更换Logo' : '上传Logo'}
                  <input type="file" accept="image/jpeg,image/png,image/webp,image/svg+xml" style={{ display: 'none' }}
                    disabled={logoUploading}
                    onChange={async (e) => {
                      const file = e.target.files?.[0]
                      if (!file) return
                      if (file.size > 2 * 1024 * 1024) { setError('Logo文件不能超过2MB'); return }
                      // 编辑模式直接上传到服务器
                      if (mode === 'edit' && initial?.id) {
                        try {
                          setLogoUploading(true)
                          const result = await uploadOrgLogo(initial.id, file)
                          setLogoUrl(result.url)
                        } catch (err) { setError(err instanceof Error ? err.message : '上传失败') }
                        finally { setLogoUploading(false) }
                      } else {
                        // 创建模式：先预览，创建成功后再上传（或提示先创建再编辑上传）
                        const reader = new FileReader()
                        reader.onload = () => setLogoUrl(reader.result as string)
                        reader.readAsDataURL(file)
                        setError('提示：Logo将在创建组织后可上传，请先创建再编辑上传Logo')
                      }
                      e.target.value = ''
                    }} />
                </label>
                {logoUrl && (
                  <button onClick={() => setLogoUrl('')} style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '12px', color: C.textMuted, cursor: 'pointer' }}>移除</button>
                )}
              </div>
              <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px' }}>支持JPG/PNG/WEBP/SVG，最大2MB。上传后在课件生成时自动使用。</div>
            </div>
          )}

          {/* v172：门户可见板块配置（仅 school + edit 显示） */}
          {showModuleConfig && (
            <div style={{ marginBottom: '14px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                门户可见板块
              </label>
              <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '8px' }}>
                勾选本校老师在首页能进入的工作区。取消勾选后，本校非管理员看不到该入口、也无法直接访问。系统管理员不受此限制。
              </div>
              {modulesLoading ? (
                <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>正在读取当前配置...</div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                  {PORTAL_MODULE_OPTIONS.map(opt => {
                    const checked = modules[opt.key] !== false
                    return (
                      <label key={opt.key} style={{
                        display: 'flex', alignItems: 'center', gap: '10px',
                        padding: '10px 12px', borderRadius: '8px',
                        border: `1px solid ${checked ? C.primary : C.border}`,
                        background: checked ? C.bg : C.white,
                        cursor: 'pointer', transition: 'all 150ms ease',
                      }}>
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => toggleModule(opt.key)}
                          style={{ width: 16, height: 16, cursor: 'pointer', accentColor: C.primary }}
                        />
                        <div style={{ flex: 1 }}>
                          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>{opt.label}</div>
                          <div style={{ fontSize: '11px', color: C.textMuted }}>{opt.desc}</div>
                        </div>
                      </label>
                    )
                  })}
                </div>
              )}
            </div>
          )}

          {/* 管理员搜索选择 */}
          <UserSearchPicker
            label="管理员（可选）"
            value={adminId} valueName={adminName}
            onChange={(id, n) => { setAdminId(id); setAdminName(n) }}
            placeholder="搜索并选择管理员用户..."
          />

          {/* 操作按钮 */}
          <div style={{ display: 'flex', gap: '10px', marginTop: '4px' }}>
            <button onClick={onClose} style={{ flex: 1, padding: '10px', borderRadius: '10px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>
              取消
            </button>
            <button
              onClick={handleSave} disabled={saving}
              style={{
                flex: 2, padding: '10px', borderRadius: '10px', border: 'none',
                background: saving ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`,
                color: '#fff', fontSize: '14px', fontWeight: 600,
                cursor: saving ? 'not-allowed' : 'pointer',
              }}>
              {saving ? '保存中...' : (mode === 'create' ? '创建' : '保存')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
