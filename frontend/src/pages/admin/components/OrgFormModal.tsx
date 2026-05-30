/**
 * OrgFormModal.tsx — 区域/学校 新建/编辑弹窗
 */
import { useState } from 'react'
import { createAdminOrg, updateAdminOrg, uploadOrgLogo } from '@/api/admin'
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

  const title = mode === 'create'
    ? (type === 'region' ? '新建区域' : '新建学校')
    : (type === 'region' ? '编辑区域' : '编辑学校')

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: '8px',
    border: `1px solid ${C.border}`, fontSize: '14px',
    outline: 'none', boxSizing: 'border-box',
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
        overflow: 'hidden', boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }}>
        {/* 头部 */}
        <div style={{
          padding: '20px 24px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
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
