/**
 * AdminPage — 统一用户管理中心
 *
 * 路由：/admin（独立页面，不在任何Layout内）
 * 权限：仅admin（路由层RoleGuard保护）
 *
 * 4个Tab：
 *   📊 概览    — 统计卡片 + 角色分布横条图 + 最近10条日志快览
 *   👥 用户管理 — 用户列表+学校筛选+详情弹窗（双Tab：基本信息/操作记录）
 *   🏫 组织架构 — 三栏递进：区域→学校→教研组，完整CRUD + 成员管理
 *   📋 操作日志 — 用户名搜索+日期范围+操作类型+详情展开
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  getAdminStats, getAdminUsers, getAdminUserDetail,
  createAdminUser, updateAdminUserStatus,
  resetAdminUserPassword, getAdminAuditLogs,
  getAdminOrgs, createAdminOrg, updateAdminOrg, deleteAdminOrg,
  getAdminGroups, getAdminGroupDetail, createAdminGroup, updateAdminGroup, deleteAdminGroup,
  getAdminGroupMembers, addAdminGroupMember, updateAdminGroupMemberRole, removeAdminGroupMember,
} from '@/api/admin'
import type {
  AdminStats, AdminUserListItem, AdminUserDetail, AuditLogItem,
  OrgListItem, GroupListItem, GroupDetail, GroupMemberItem,
  CreateOrgRequest, UpdateOrgRequest,
  CreateGroupRequest, UpdateGroupRequest,
} from '@/api/admin'

// ==================== 样式常量 ====================

const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981',
  successLight: 'rgba(16,185,129,0.08)',
  danger: '#EF4444',
  dangerLight: 'rgba(239,68,68,0.08)',
  warning: '#F59E0B',
  warningLight: 'rgba(245,158,11,0.08)',
  purple: '#7C3AED',
  purpleLight: 'rgba(124,58,237,0.08)',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  bg: '#F9FAFB',
  white: '#FFFFFF',
}

const ROLE_OPTIONS = [
  { value: '', label: '全部角色' },
  { value: 'admin', label: '管理员' },
  { value: 'senior_operator', label: '高级操作员' },
  { value: 'operator', label: '操作员' },
  { value: 'viewer', label: '查看者' },
]

const ACTION_OPTIONS = [
  { value: '', label: '全部操作' },
  { value: 'user.login', label: '用户登录' },
  { value: 'user.logout', label: '用户登出' },
  { value: 'admin.user_create', label: '创建用户' },
  { value: 'admin.user_status', label: '状态变更' },
  { value: 'admin.user_reset_password', label: '重置密码' },
  { value: 'pipeline.confirm_finalize', label: '确认定稿' },
  { value: 'pipeline.verify', label: '触发验收' },
]

// ==================== 工具函数 ====================

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr)
  const diffMin = Math.floor((Date.now() - date.getTime()) / 60000)
  if (diffMin < 1)  return '刚刚'
  if (diffMin < 60) return `${diffMin}分钟前`
  const diffH = Math.floor(diffMin / 60)
  if (diffH < 24)   return `${diffH}小时前`
  const diffD = Math.floor(diffH / 24)
  if (diffD < 7)    return `${diffD}天前`
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

function fmt(dateStr: string | null | undefined): string {
  if (!dateStr) return '—'
  return String(dateStr).replace('T', ' ').substring(0, 16)
}

// ==================== 通用小组件 ====================

function Toast({ message, type, onClose }: {
  message: string; type: 'success' | 'error'; onClose: () => void
}) {
  useEffect(() => { const t = setTimeout(onClose, 3500); return () => clearTimeout(t) }, [onClose])
  return (
    <div style={{
      position: 'fixed', top: '24px', right: '24px', zIndex: 9999,
      padding: '12px 20px', borderRadius: '12px', color: '#fff', fontSize: '14px', fontWeight: 500,
      background: type === 'success' ? 'linear-gradient(135deg,#10B981,#059669)' : 'linear-gradient(135deg,#EF4444,#DC2626)',
      boxShadow: '0 4px 16px rgba(0,0,0,0.15)',
    }}>
      {type === 'success' ? '✓ ' : '✕ '}{message}
    </div>
  )
}

function RoleBadge({ role, roleName }: { role: string; roleName?: string }) {
  const sm: Record<string, { bg: string; color: string }> = {
    admin: { bg: C.dangerLight, color: C.danger },
    senior_operator: { bg: C.warningLight, color: C.warning },
    operator: { bg: C.primaryLight, color: C.primary },
    viewer: { bg: C.bg, color: C.textSec },
  }
  const s = sm[role] || { bg: C.bg, color: C.textSec }
  const nm: Record<string, string> = { admin: '管理员', senior_operator: '高级操作员', operator: '操作员', viewer: '查看者' }
  return (
    <span style={{ display: 'inline-block', padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 600, background: s.bg, color: s.color }}>
      {roleName || nm[role] || role}
    </span>
  )
}

function StatusBadge({ status }: { status: string }) {
  const active = status === 'active'
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px', padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 600, background: active ? C.successLight : C.dangerLight, color: active ? C.success : C.danger }}>
      <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: active ? C.success : C.danger }} />
      {active ? '正常' : '已禁用'}
    </span>
  )
}

function StatCard({ label, value, sub, color }: { label: string; value: number; sub?: string; color?: string }) {
  return (
    <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, padding: '20px 24px', flex: 1, boxShadow: '0 1px 4px rgba(0,0,0,0.04)' }}>
      <div style={{ fontSize: '13px', color: C.textSec, marginBottom: '8px' }}>{label}</div>
      <div style={{ fontSize: '28px', fontWeight: 700, color: color || C.text, lineHeight: 1 }}>{value}</div>
      {sub && <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '6px' }}>{sub}</div>}
    </div>
  )
}

// ==================== 确认删除对话框 ====================

function ConfirmDialog({ title, message, onConfirm, onCancel }: {
  title: string; message: string; onConfirm: () => void; onCancel: () => void
}) {
  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 11000, background: 'rgba(0,0,0,0.45)', backdropFilter: 'blur(4px)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <div style={{ background: C.white, borderRadius: '16px', width: '380px', padding: '28px', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ fontSize: '16px', fontWeight: 700, color: C.text, marginBottom: '10px' }}>{title}</div>
        <div style={{ fontSize: '14px', color: C.textSec, marginBottom: '24px', lineHeight: 1.6 }}>{message}</div>
        <div style={{ display: 'flex', gap: '10px' }}>
          <button onClick={onCancel} style={{ flex: 1, padding: '10px', borderRadius: '10px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>取消</button>
          <button onClick={onConfirm} style={{ flex: 1, padding: '10px', borderRadius: '10px', border: 'none', background: C.danger, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>确认删除</button>
        </div>
      </div>
    </div>
  )
}

// ==================== 用户搜索选择器（新建/编辑弹窗公用）====================

/**
 * UserSearchPicker — 输入关键词实时搜索用户，选中后回调
 * 用于管理员选择、组长选择等场景
 */
function UserSearchPicker({ label, value, valueName, onChange, placeholder }: {
  label: string
  value: string
  valueName: string
  onChange: (id: string, name: string) => void
  placeholder?: string
}) {
  const [kw, setKw]             = useState('')
  const [results, setResults]   = useState<AdminUserListItem[]>([])
  const [searching, setSearching] = useState(false)
  const [open, setOpen]         = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const search = useCallback(async (q: string) => {
    if (!q.trim()) { setResults([]); return }
    try {
      setSearching(true)
      const data = await getAdminUsers({ keyword: q, page: 1, page_size: 8 })
      setResults(data.users)
    } catch { } finally { setSearching(false) }
  }, [])

  const handleKwChange = (v: string) => {
    setKw(v)
    setOpen(true)
    if (timer.current) clearTimeout(timer.current)
    timer.current = setTimeout(() => search(v), 350)
  }

  const handleSelect = (u: AdminUserListItem) => {
    onChange(u.id, u.display_name || u.username)
    setKw('')
    setResults([])
    setOpen(false)
  }

  const handleClear = () => {
    onChange('', '')
    setKw('')
    setResults([])
  }

  return (
    <div style={{ marginBottom: '14px' }}>
      <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>{label}</label>

      {/* 已选中状态 */}
      {value ? (
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.primary}`, background: C.primaryLight }}>
          <span style={{ flex: 1, fontSize: '13px', color: C.primary, fontWeight: 500 }}>✓ {valueName}</span>
          <button onClick={handleClear} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '14px', color: C.textMuted, padding: '0 4px' }}>×</button>
        </div>
      ) : (
        /* 搜索输入框 */
        <div style={{ position: 'relative' }}>
          <input
            value={kw}
            onChange={e => handleKwChange(e.target.value)}
            onFocus={() => kw && setOpen(true)}
            onBlur={() => setTimeout(() => setOpen(false), 200)}
            placeholder={placeholder || '输入用户名或显示名搜索...'}
            style={{ width: '100%', padding: '9px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }}
            onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
          />
          {/* 搜索结果下拉 */}
          {open && (results.length > 0 || searching) && (
            <div style={{ position: 'absolute', top: '100%', left: 0, right: 0, zIndex: 200, background: C.white, border: `1px solid ${C.border}`, borderRadius: '8px', boxShadow: '0 4px 16px rgba(0,0,0,0.12)', marginTop: '4px', overflow: 'hidden' }}>
              {searching ? (
                <div style={{ padding: '12px', textAlign: 'center', fontSize: '13px', color: C.textMuted }}>搜索中...</div>
              ) : (
                results.map(u => (
                  <div key={u.id} onMouseDown={() => handleSelect(u)}
                    style={{ padding: '10px 14px', cursor: 'pointer', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', gap: '10px' }}
                    onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
                    onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.white }}>
                    <div style={{ width: '28px', height: '28px', borderRadius: '50%', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: '11px', fontWeight: 700, flexShrink: 0 }}>
                      {(u.display_name || u.username).charAt(0).toUpperCase()}
                    </div>
                    <div>
                      <div style={{ fontSize: '13px', fontWeight: 500, color: C.text }}>{u.display_name}</div>
                      <div style={{ fontSize: '11px', color: C.textMuted }}>@{u.username}</div>
                    </div>
                    <RoleBadge role={u.role} />
                  </div>
                ))
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// ==================== 组织弹窗（新建/编辑区域或学校）====================

function OrgFormModal({ mode, type, initial, regions, onClose, onSaved }: {
  mode: 'create' | 'edit'
  type: 'region' | 'school'
  initial?: OrgListItem
  regions: OrgListItem[]   // 创建学校时的区域下拉选项
  onClose: () => void
  onSaved: () => void
}) {
  const [name, setName]           = useState(initial?.name || '')
  const [parentId, setParentId]   = useState(initial?.parent_id || '')
  const [adminId, setAdminId]     = useState(initial?.admin_user_id || '')
  const [adminName, setAdminName] = useState(initial?.admin_user_name || '')
  const [saving, setSaving]       = useState(false)
  const [error, setError]         = useState('')

  const title = mode === 'create'
    ? (type === 'region' ? '新建区域' : '新建学校')
    : (type === 'region' ? '编辑区域' : '编辑学校')

  const handleSave = async () => {
    if (!name.trim()) { setError('请输入名称'); return }
    if (type === 'school' && !parentId) { setError('请选择所属区域'); return }
    try {
      setSaving(true); setError('')
      if (mode === 'create') {
        const req: CreateOrgRequest = { name: name.trim(), type, parent_id: type === 'school' ? parentId : null, admin_user_id: adminId || null }
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
    <div style={{ position: 'fixed', inset: 0, zIndex: 10500, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{ background: C.white, borderRadius: '20px', width: '480px', overflow: 'hidden', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ padding: '20px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>{title}</div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
        </div>
        <div style={{ padding: '24px' }}>
          {error && <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '14px', background: C.dangerLight, color: C.danger, fontSize: '13px' }}>{error}</div>}

          {/* 名称 */}
          <div style={{ marginBottom: '14px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
              {type === 'region' ? '区域名称' : '学校名称'} <span style={{ color: C.danger }}>*</span>
            </label>
            <input value={name} onChange={e => setName(e.target.value)} placeholder="请输入名称"
              style={{ width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border }}
            />
          </div>

          {/* 所属区域（仅创建学校时必填；编辑时不可改） */}
          {type === 'school' && mode === 'create' && (
            <div style={{ marginBottom: '14px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                所属区域 <span style={{ color: C.danger }}>*</span>
              </label>
              <select value={parentId} onChange={e => setParentId(e.target.value)}
                style={{ width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: C.white, boxSizing: 'border-box' }}>
                <option value="">请选择区域</option>
                {regions.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
              </select>
            </div>
          )}

          {/* 管理员搜索选择 */}
          <UserSearchPicker
            label="管理员（可选）"
            value={adminId}
            valueName={adminName}
            onChange={(id, name) => { setAdminId(id); setAdminName(name) }}
            placeholder="搜索并选择管理员用户..."
          />

          <div style={{ display: 'flex', gap: '10px', marginTop: '4px' }}>
            <button onClick={onClose} style={{ flex: 1, padding: '10px', borderRadius: '10px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>取消</button>
            <button onClick={handleSave} disabled={saving}
              style={{ flex: 2, padding: '10px', borderRadius: '10px', border: 'none', background: saving ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer' }}>
              {saving ? '保存中...' : (mode === 'create' ? '创建' : '保存')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// ==================== 教研组弹窗（新建/编辑）====================

function GroupFormModal({ mode, schoolId, schoolName, initial, onClose, onSaved }: {
  mode: 'create' | 'edit'
  schoolId: string
  schoolName: string
  initial?: GroupListItem
  onClose: () => void
  onSaved: () => void
}) {
  const [name, setName]           = useState(initial?.name || '')
  const [subject, setSubject]     = useState(initial?.subject || '')
  const [gradeRange, setGradeRange] = useState(initial?.grade_range || '')
  const [leadId, setLeadId]       = useState(initial?.lead_user_id || '')
  const [leadName, setLeadName]   = useState(initial?.lead_user_name || '')
  const [desc, setDesc]           = useState('')
  const [saving, setSaving]       = useState(false)
  const [error, setError]         = useState('')

  const handleSave = async () => {
    if (!name.trim()) { setError('请输入教研组名称'); return }
    if (!subject.trim()) { setError('请输入学科'); return }
    try {
      setSaving(true); setError('')
      if (mode === 'create') {
        const req: CreateGroupRequest = { name: name.trim(), school_id: schoolId, subject: subject.trim(), grade_range: gradeRange.trim() || undefined, lead_user_id: leadId || null, description: desc.trim() || undefined }
        await createAdminGroup(req)
      } else {
        const req: UpdateGroupRequest = { name: name.trim(), subject: subject.trim(), grade_range: gradeRange.trim() || undefined, lead_user_id: leadId || null, description: desc.trim() || undefined }
        await updateAdminGroup(initial!.id, req)
      }
      onSaved(); onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '操作失败')
    } finally { setSaving(false) }
  }

  const fieldStyle: React.CSSProperties = { width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 10500, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{ background: C.white, borderRadius: '20px', width: '500px', maxHeight: '88vh', overflow: 'hidden', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ padding: '20px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>{mode === 'create' ? '新建教研组' : '编辑教研组'}</div>
            <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>所属学校：{schoolName}</div>
          </div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: '24px' }}>
          {error && <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '14px', background: C.dangerLight, color: C.danger, fontSize: '13px' }}>{error}</div>}

          {/* 教研组名称 */}
          <div style={{ marginBottom: '14px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>教研组名称 <span style={{ color: C.danger }}>*</span></label>
            <input value={name} onChange={e => setName(e.target.value)} placeholder="例如：七年级AI课程教研组"
              style={fieldStyle}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border }}
            />
          </div>

          {/* 学科 */}
          <div style={{ marginBottom: '14px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>学科 <span style={{ color: C.danger }}>*</span></label>
            <input value={subject} onChange={e => setSubject(e.target.value)} placeholder="例如：AI课程"
              style={fieldStyle}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border }}
            />
          </div>

          {/* 年级范围 */}
          <div style={{ marginBottom: '14px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>年级范围（可选）</label>
            <input value={gradeRange} onChange={e => setGradeRange(e.target.value)} placeholder="例如：G7-G9"
              style={fieldStyle}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border }}
            />
          </div>

          {/* 组长搜索选择 */}
          <UserSearchPicker
            label="教研组长（可选）"
            value={leadId}
            valueName={leadName}
            onChange={(id, name) => { setLeadId(id); setLeadName(name) }}
            placeholder="搜索并选择组长..."
          />

          {/* 描述 */}
          <div style={{ marginBottom: '14px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>描述（可选）</label>
            <textarea value={desc} onChange={e => setDesc(e.target.value)} placeholder="教研组简介..." rows={3}
              style={{ ...fieldStyle, resize: 'vertical', fontFamily: 'inherit' }}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border }}
            />
          </div>

          <div style={{ display: 'flex', gap: '10px' }}>
            <button onClick={onClose} style={{ flex: 1, padding: '10px', borderRadius: '10px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>取消</button>
            <button onClick={handleSave} disabled={saving}
              style={{ flex: 2, padding: '10px', borderRadius: '10px', border: 'none', background: saving ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer' }}>
              {saving ? '保存中...' : (mode === 'create' ? '创建' : '保存')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// ==================== 成员管理面板（教研组内嵌展开）====================

function MemberPanel({ groupId, onClose }: { groupId: string; onClose: () => void }) {
  const [members, setMembers]     = useState<GroupMemberItem[]>([])
  const [loading, setLoading]     = useState(true)

  // 添加成员状态
  const [addUserId, setAddUserId]     = useState('')
  const [addUserName, setAddUserName] = useState('')
  const [addRole, setAddRole]         = useState('member')
  const [adding, setAdding]           = useState(false)
  const [addError, setAddError]       = useState('')

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setMembers(await getAdminGroupMembers(groupId))
    } catch { } finally { setLoading(false) }
  }, [groupId])

  useEffect(() => { load() }, [load])

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

  const handleRemove = async (userId: string) => {
    try {
      await removeAdminGroupMember(groupId, userId)
      await load()
    } catch { }
  }

  const handleRoleChange = async (userId: string, role: string) => {
    try {
      await updateAdminGroupMemberRole(groupId, userId, role)
      await load()
    } catch { }
  }

  const roleLabel = (role: string) => role === 'backbone' ? '骨干教师' : '普通成员'
  const roleColor = (role: string) => role === 'backbone' ? { bg: C.purpleLight, color: C.purple } : { bg: C.bg, color: C.textSec }

  return (
    <div style={{ padding: '16px', background: 'rgba(79,123,232,0.04)', borderTop: `1px dashed ${C.border}` }}>
      {/* 标题行 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
        <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>👥 成员管理</div>
        <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '12px', color: C.textMuted }}>收起 ▲</button>
      </div>

      {/* 现有成员列表 */}
      {loading ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>加载中...</div>
      ) : members.length === 0 ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>暂无成员</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', marginBottom: '14px' }}>
          {members.map(m => {
            const rc = roleColor(m.role)
            return (
              <div key={m.user_id} style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 12px', borderRadius: '8px', background: C.white, border: `1px solid ${C.border}` }}>
                {/* 头像 */}
                <div style={{ width: '28px', height: '28px', borderRadius: '50%', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: '11px', fontWeight: 700, flexShrink: 0 }}>
                  {(m.display_name || m.username).charAt(0).toUpperCase()}
                </div>
                {/* 姓名 */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: '13px', fontWeight: 500, color: C.text }}>{m.display_name}</div>
                  <div style={{ fontSize: '11px', color: C.textMuted }}>加入：{fmt(m.joined_at)}</div>
                </div>
                {/* 角色选择下拉 */}
                <select value={m.role} onChange={e => handleRoleChange(m.user_id, e.target.value)}
                  style={{ padding: '3px 8px', borderRadius: '6px', border: `1px solid ${rc.color}`, background: rc.bg, color: rc.color, fontSize: '11px', fontWeight: 600, cursor: 'pointer', outline: 'none' }}>
                  <option value="member">普通成员</option>
                  <option value="backbone">骨干教师</option>
                </select>
                {/* 移除按钮 */}
                <button onClick={() => handleRemove(m.user_id)}
                  style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.dangerLight}`, background: C.dangerLight, color: C.danger, fontSize: '11px', cursor: 'pointer', fontWeight: 500, whiteSpace: 'nowrap' }}>
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
          value={addUserId}
          valueName={addUserName}
          onChange={(id, name) => { setAddUserId(id); setAddUserName(name) }}
          placeholder="输入用户名搜索..."
        />
        <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
          <select value={addRole} onChange={e => setAddRole(e.target.value)}
            style={{ padding: '7px 10px', borderRadius: '7px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', background: C.white }}>
            <option value="member">普通成员</option>
            <option value="backbone">骨干教师</option>
          </select>
          <button onClick={handleAdd} disabled={adding || !addUserId}
            style={{ flex: 1, padding: '7px', borderRadius: '7px', border: 'none', background: (!addUserId || adding) ? C.textMuted : C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: (!addUserId || adding) ? 'not-allowed' : 'pointer' }}>
            {adding ? '添加中...' : '+ 确认添加'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ==================== 角色分布横条图（纯CSS）====================

function RoleBarChart({ stats }: { stats: AdminStats }) {
  const roles = [
    { label: '管理员', count: stats.admin_count, color: C.danger },
    { label: '高级操作员', count: stats.senior_operator_count, color: C.warning },
    { label: '操作员', count: stats.operator_count, color: C.primary },
    { label: '查看者', count: stats.viewer_count, color: C.textSec },
  ]
  const total = stats.total_users || 1
  const maxCount = Math.max(...roles.map(r => r.count), 1)
  const [animated, setAnimated] = useState(false)
  useEffect(() => { const t = setTimeout(() => setAnimated(true), 50); return () => clearTimeout(t) }, [])
  return (
    <div style={{ background: C.white, borderRadius: '16px', border: `1px solid ${C.border}`, padding: '24px', marginBottom: '16px', boxShadow: '0 1px 4px rgba(0,0,0,0.04)' }}>
      <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '20px', display: 'flex', justifyContent: 'space-between' }}>
        <span>角色分布</span>
        <span style={{ fontSize: '12px', color: C.textMuted, fontWeight: 400 }}>共 {stats.total_users} 人</span>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
        {roles.map(role => {
          const barPct = (role.count / maxCount) * 100
          const totalPct = ((role.count / total) * 100).toFixed(1)
          return (
            <div key={role.label} style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <div style={{ width: '80px', flexShrink: 0, fontSize: '13px', color: C.textSec, textAlign: 'right', fontWeight: 500 }}>{role.label}</div>
              <div style={{ flex: 1, height: '24px', background: C.bg, borderRadius: '6px', overflow: 'hidden', position: 'relative' }}>
                <div style={{ position: 'absolute', left: 0, top: 0, bottom: 0, width: animated ? `${barPct}%` : '0%', background: role.color, borderRadius: '6px', opacity: 0.85, transition: 'width 500ms cubic-bezier(0.4,0,0.2,1)' }} />
                {role.count > 0 && barPct > 18 && (
                  <div style={{ position: 'absolute', left: '10px', top: 0, bottom: 0, display: 'flex', alignItems: 'center', fontSize: '11px', fontWeight: 600, color: '#fff', opacity: animated ? 1 : 0, transition: 'opacity 400ms ease 200ms' }}>{role.count} 人</div>
                )}
              </div>
              <div style={{ width: '76px', flexShrink: 0, display: 'flex', flexDirection: 'column', alignItems: 'flex-end' }}>
                <span style={{ fontSize: '14px', fontWeight: 700, color: role.color }}>{role.count}</span>
                <span style={{ fontSize: '11px', color: C.textMuted }}>{totalPct}%</span>
              </div>
            </div>
          )
        })}
      </div>
      <div style={{ marginTop: '16px', paddingTop: '12px', borderTop: `1px solid ${C.border}`, fontSize: '12px', color: C.textMuted, display: 'flex', gap: '20px', flexWrap: 'wrap' }}>
        {roles.map(r => (
          <span key={r.label} style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
            <span style={{ display: 'inline-block', width: '8px', height: '8px', borderRadius: '2px', background: r.color }} />
            {r.label}：{r.count} 人
          </span>
        ))}
      </div>
    </div>
  )
}

// ==================== 最近操作日志快览 ====================

function RecentLogsCard({ logs, loading, onViewAll }: { logs: AuditLogItem[]; loading: boolean; onViewAll: () => void }) {
  const getAS = (a: string) => a.startsWith('user.') ? { bg: C.primaryLight, color: C.primary } : a.startsWith('admin.') ? { bg: C.purpleLight, color: C.purple } : { bg: C.warningLight, color: C.warning }
  return (
    <div style={{ background: C.white, borderRadius: '16px', border: `1px solid ${C.border}`, padding: '24px', marginBottom: '16px', boxShadow: '0 1px 4px rgba(0,0,0,0.04)' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
        <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>最近操作</div>
        <button onClick={onViewAll} style={{ padding: '5px 14px', borderRadius: '8px', cursor: 'pointer', border: `1px solid ${C.border}`, background: C.bg, fontSize: '12px', color: C.primary, fontWeight: 500 }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.primaryLight }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}>查看全部 →</button>
      </div>
      {loading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {[1,2,3].map(i => (
            <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <div style={{ width: '32px', height: '32px', borderRadius: '50%', background: C.border, flexShrink: 0 }} />
              <div style={{ flex: 1 }}>
                <div style={{ height: '12px', borderRadius: '4px', background: C.border, width: `${50 + i * 10}%`, marginBottom: '6px' }} />
                <div style={{ height: '10px', borderRadius: '4px', background: C.bg, width: '40%' }} />
              </div>
            </div>
          ))}
        </div>
      ) : logs.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '24px', color: C.textMuted, fontSize: '13px' }}>暂无操作记录</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column' }}>
          {logs.map((log, idx) => {
            const s = getAS(log.action)
            let dd = ''
            try { const d = JSON.parse(log.detail); const e = Object.entries(d); if (e.length) { const [k, v] = e[0]; dd = `${k}: ${v}` } } catch { dd = log.detail || '' }
            return (
              <div key={log.id} style={{ display: 'flex', alignItems: 'center', gap: '12px', padding: '10px 0', borderBottom: idx < logs.length - 1 ? `1px solid ${C.border}` : 'none' }}>
                <div style={{ width: '32px', height: '32px', borderRadius: '50%', flexShrink: 0, background: `linear-gradient(135deg,${C.primary},#7C3AED)`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: '12px', fontWeight: 700 }}>
                  {(log.display_name || log.username || '?').charAt(0).toUpperCase()}
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                    <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>{log.display_name || log.username}</span>
                    <span style={{ padding: '1px 7px', borderRadius: '5px', fontSize: '11px', fontWeight: 600, background: s.bg, color: s.color }}>{log.action_name}</span>
                  </div>
                  {dd && <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '2px', fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{dd}</div>}
                </div>
                <div style={{ flexShrink: 0, fontSize: '11px', color: C.textMuted, whiteSpace: 'nowrap' }}>
                  {formatRelativeTime(typeof log.created_at === 'string' ? log.created_at : new Date(log.created_at).toISOString())}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

// ==================== 用户详情弹窗（双Tab） ====================

function UserDetailModal({ userId, onClose, onAction }: { userId: string; onClose: () => void; onAction: () => void }) {
  const [detail, setDetail]   = useState<AdminUserDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [resetPwd, setResetPwd] = useState('')
  const [saving, setSaving]   = useState(false)
  const [detailTab, setDetailTab] = useState<'info' | 'logs'>('info')
  const [userLogs, setUserLogs]   = useState<AuditLogItem[]>([])
  const [logsLoading, setLogsLoading] = useState(false)
  const [logsLoaded, setLogsLoaded]   = useState(false)

  useEffect(() => {
    getAdminUserDetail(userId).then(setDetail).catch(() => {}).finally(() => setLoading(false))
  }, [userId])

  useEffect(() => {
    if (detailTab !== 'logs' || logsLoaded) return
    setLogsLoading(true)
    getAdminAuditLogs({ user_id: userId, page: 1, page_size: 20 })
      .then(d => { setUserLogs(d.logs); setLogsLoaded(true) })
      .catch(() => { setLogsLoaded(true) })
      .finally(() => setLogsLoading(false))
  }, [detailTab, userId, logsLoaded])

  const handleReset = async () => {
    if (resetPwd.length < 6) return
    try { setSaving(true); await resetAdminUserPassword(userId, resetPwd); setResetPwd(''); onAction() } catch { } finally { setSaving(false) }
  }

  const handleToggle = async () => {
    if (!detail) return
    const ns = detail.status === 'active' ? 'disabled' : 'active'
    try { setSaving(true); await updateAdminUserStatus(userId, ns); setDetail(p => p ? { ...p, status: ns } : p); onAction() } catch { } finally { setSaving(false) }
  }

  const getMRLabel = (role: string, isLead: boolean) => {
    if (isLead) return { text: '组长', bg: C.warningLight, color: C.warning }
    if (role === 'backbone') return { text: '骨干教师', bg: C.purpleLight, color: C.purple }
    return { text: '普通成员', bg: C.bg, color: C.textSec }
  }

  const getAS = (a: string) => a.startsWith('user.') ? { bg: C.primaryLight, color: C.primary } : a.startsWith('admin.') ? { bg: C.purpleLight, color: C.purple } : { bg: C.warningLight, color: C.warning }

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 10000, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '24px' }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{ background: C.white, borderRadius: '20px', width: '660px', maxHeight: '88vh', overflow: 'hidden', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ padding: '20px 24px', borderBottom: `1px solid ${C.border}` }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: (!loading && detail) ? '16px' : 0 }}>
            <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>用户详情</div>
            <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
          </div>
          {!loading && detail && (
            <>
              <div style={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
                <div style={{ width: '48px', height: '48px', borderRadius: '50%', flexShrink: 0, background: 'linear-gradient(135deg,#4F7BE8,#7C3AED)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '18px', color: '#fff', fontWeight: 700 }}>
                  {detail.display_name?.charAt(0)?.toUpperCase() || 'U'}
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>{detail.display_name}</div>
                  <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px' }}>@{detail.username}</div>
                  <div style={{ display: 'flex', gap: '6px', marginTop: '6px' }}>
                    <RoleBadge role={detail.role} roleName={detail.role_name} />
                    <StatusBadge status={detail.status} />
                  </div>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <div style={{ fontSize: '22px', fontWeight: 700, color: C.primary }}>{detail.login_count}</div>
                  <div style={{ fontSize: '11px', color: C.textMuted }}>累计登录</div>
                </div>
              </div>
              <div style={{ display: 'flex', gap: '4px', marginTop: '16px', background: C.bg, borderRadius: '10px', padding: '3px', border: `1px solid ${C.border}`, width: 'fit-content' }}>
                {(['info', 'logs'] as const).map(tab => (
                  <button key={tab} onClick={() => setDetailTab(tab)} style={{ padding: '6px 18px', borderRadius: '8px', border: 'none', cursor: 'pointer', fontSize: '13px', fontWeight: detailTab === tab ? 600 : 400, color: detailTab === tab ? C.primary : C.textSec, background: detailTab === tab ? C.white : 'transparent', boxShadow: detailTab === tab ? '0 1px 4px rgba(0,0,0,0.08)' : 'none', transition: 'all 150ms ease' }}>
                    {tab === 'info' ? '📋 基本信息' : '📄 操作记录'}
                  </button>
                ))}
              </div>
            </>
          )}
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: '20px 24px' }}>
          {loading && <div style={{ textAlign: 'center', padding: '40px', color: C.textMuted }}>加载中...</div>}
          {!loading && !detail && <div style={{ textAlign: 'center', padding: '40px', color: C.danger }}>加载失败</div>}
          {!loading && detail && detailTab === 'info' && (
            <>
              <div style={{ marginBottom: '20px' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '10px', paddingBottom: '6px', borderBottom: `1px solid ${C.border}` }}>账户信息</div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
                  {[{ l: '注册时间', v: fmt(detail.created_at) }, { l: '最近登录', v: detail.last_login_at ? fmt(detail.last_login_at) : '暂无' }].map(i => (
                    <div key={i.l} style={{ padding: '10px 14px', borderRadius: '8px', background: C.bg }}>
                      <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '3px' }}>{i.l}</div>
                      <div style={{ fontSize: '13px', color: C.text, fontWeight: 500 }}>{i.v}</div>
                    </div>
                  ))}
                </div>
              </div>
              <div style={{ marginBottom: '20px' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '10px', paddingBottom: '6px', borderBottom: `1px solid ${C.border}` }}>课件审核权限</div>
                {detail.course_assignments.length === 0 ? (
                  <div style={{ fontSize: '13px', color: C.textMuted }}>未分配课程</div>
                ) : (
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                    {detail.course_assignments.map(a => (
                      <span key={a.course_code} style={{ padding: '4px 10px', borderRadius: '6px', background: C.primaryLight, color: C.primary, fontSize: '12px', fontFamily: 'monospace' }}>
                        {a.course_code}{a.course_name !== a.course_code ? ` · ${a.course_name}` : ''}
                      </span>
                    ))}
                  </div>
                )}
              </div>
              <div style={{ marginBottom: '20px' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '10px', paddingBottom: '6px', borderBottom: `1px solid ${C.border}` }}>教案系统归属</div>
                {detail.teaching_groups.length === 0 ? (
                  <div style={{ padding: '20px', borderRadius: '10px', background: C.bg, border: `1px dashed ${C.border}`, textAlign: 'center', fontSize: '13px', color: C.textMuted }}>暂未加入任何教研组</div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                    <div style={{ display: 'grid', gridTemplateColumns: '1.2fr 1.4fr 0.9fr 1fr', padding: '4px 12px', fontSize: '11px', fontWeight: 600, color: C.textMuted }}>
                      <span>所属学校</span><span>教研组</span><span>成员角色</span><span>加入时间</span>
                    </div>
                    {detail.teaching_groups.map(g => {
                      const rl = getMRLabel(g.role, g.is_lead)
                      return (
                        <div key={g.group_id} style={{ display: 'grid', gridTemplateColumns: '1.2fr 1.4fr 0.9fr 1fr', padding: '10px 12px', borderRadius: '10px', background: C.bg, border: `1px solid ${C.border}`, alignItems: 'center' }}>
                          <div style={{ fontSize: '13px', color: C.textSec }}>🏫 {g.school_name}</div>
                          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>{g.is_lead && <span style={{ color: C.warning, marginRight: '3px' }}>★</span>}{g.group_name}</div>
                          <div><span style={{ display: 'inline-block', padding: '2px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, background: rl.bg, color: rl.color, border: `1px solid ${rl.color}22` }}>{rl.text}</span></div>
                          <div style={{ fontSize: '11px', color: C.textMuted }}>{fmt(g.joined_at)}</div>
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
              <div style={{ marginBottom: '20px' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '10px', paddingBottom: '6px', borderBottom: `1px solid ${C.border}` }}>账户操作</div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  <div style={{ display: 'flex', gap: '8px' }}>
                    <input type="password" value={resetPwd} onChange={e => setResetPwd(e.target.value)} placeholder="输入新密码（至少6位）"
                      style={{ flex: 1, padding: '9px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none' }}
                      onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                      onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                    />
                    <button onClick={handleReset} disabled={resetPwd.length < 6 || saving}
                      style={{ padding: '9px 18px', borderRadius: '8px', border: 'none', background: resetPwd.length >= 6 ? C.primary : C.textMuted, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: resetPwd.length >= 6 ? 'pointer' : 'not-allowed' }}>
                      重置密码
                    </button>
                  </div>
                  <button onClick={handleToggle} disabled={saving}
                    style={{ padding: '9px 18px', borderRadius: '8px', border: `1px solid ${detail.status === 'active' ? C.danger : C.success}`, background: detail.status === 'active' ? C.dangerLight : C.successLight, color: detail.status === 'active' ? C.danger : C.success, fontSize: '14px', fontWeight: 600, cursor: 'pointer', width: '100%' }}>
                    {detail.status === 'active' ? '禁用该账户' : '启用该账户'}
                  </button>
                </div>
              </div>
            </>
          )}
          {!loading && detail && detailTab === 'logs' && (
            <div>
              <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '14px' }}>显示该用户最近 20 条操作记录</div>
              {logsLoading ? (
                <div style={{ textAlign: 'center', padding: '40px', color: C.textMuted }}>加载中...</div>
              ) : userLogs.length === 0 ? (
                <div style={{ padding: '32px', borderRadius: '12px', background: C.bg, border: `1px dashed ${C.border}`, textAlign: 'center', fontSize: '13px', color: C.textMuted }}>暂无操作记录</div>
              ) : (
                userLogs.map((log, idx) => {
                  const s = getAS(log.action)
                  let dd = ''
                  try { dd = Object.entries(JSON.parse(log.detail)).map(([k,v]) => `${k}: ${v}`).join('  ·  ') } catch { dd = log.detail || '' }
                  return (
                    <div key={log.id} style={{ display: 'flex', alignItems: 'flex-start', gap: '12px', padding: '12px 0', borderBottom: idx < userLogs.length - 1 ? `1px solid ${C.border}` : 'none' }}>
                      <div style={{ width: '8px', height: '8px', borderRadius: '50%', marginTop: '5px', flexShrink: 0, background: s.color }} />
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '3px' }}>
                          <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, background: s.bg, color: s.color }}>{log.action_name}</span>
                          {log.ip && <span style={{ fontSize: '11px', color: C.textMuted, fontFamily: 'monospace' }}>{log.ip}</span>}
                        </div>
                        {dd && <div style={{ fontSize: '12px', color: C.textSec, fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{dd}</div>}
                      </div>
                      <div style={{ flexShrink: 0, fontSize: '11px', color: C.textMuted, whiteSpace: 'nowrap' }}>{fmt(typeof log.created_at === 'string' ? log.created_at : new Date(log.created_at).toISOString())}</div>
                    </div>
                  )
                })
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

// ==================== 新建用户弹窗 ====================

function CreateUserModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [form, setForm] = useState({ username: '', display_name: '', password: '', role: 'operator' })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const { createAdminUser: create } = { createAdminUser: async (d: typeof form) => { const { createAdminUser } = await import('@/api/admin'); return createAdminUser(d) } }

  const handleCreate = async () => {
    if (!form.username.trim() || !form.display_name.trim() || form.password.length < 6) { setError('请填写完整信息（密码至少6位）'); return }
    try { setSaving(true); setError(''); await create(form); onCreated(); onClose() } catch (e: unknown) { setError(e instanceof Error ? e.message : '创建失败') } finally { setSaving(false) }
  }

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 10000, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{ background: C.white, borderRadius: '20px', width: '460px', overflow: 'hidden', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ padding: '20px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>新建用户</div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
        </div>
        <div style={{ padding: '24px' }}>
          {error && <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '16px', background: C.dangerLight, color: C.danger, fontSize: '13px' }}>{error}</div>}
          {[
            { key: 'username', label: '登录用户名', placeholder: '字母数字下划线', type: 'text' },
            { key: 'display_name', label: '显示名称', placeholder: '例如：张老师', type: 'text' },
            { key: 'password', label: '初始密码', placeholder: '至少6位', type: 'password' },
          ].map(f => (
            <div key={f.key} style={{ marginBottom: '14px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>{f.label}</label>
              <input type={f.type} value={(form as Record<string,string>)[f.key]} onChange={e => setForm(p => ({ ...p, [f.key]: e.target.value }))} placeholder={f.placeholder}
                style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }}
                onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                onBlur={e => { e.currentTarget.style.borderColor = C.border }}
              />
            </div>
          ))}
          <div style={{ marginBottom: '20px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>系统角色</label>
            <select value={form.role} onChange={e => setForm(p => ({ ...p, role: e.target.value }))} style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: C.white }}>
              {ROLE_OPTIONS.filter(r => r.value).map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
            </select>
          </div>
          <div style={{ display: 'flex', gap: '10px' }}>
            <button onClick={onClose} style={{ flex: 1, padding: '10px', borderRadius: '10px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>取消</button>
            <button onClick={handleCreate} disabled={saving} style={{ flex: 2, padding: '10px', borderRadius: '10px', border: 'none', background: saving ? C.textMuted : `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer' }}>
              {saving ? '创建中...' : '创建用户'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// ==================== 主组件 ====================

export default function AdminPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const fromPath: string = (location.state as { from?: string })?.from || '/'

  const [activeTab, setActiveTab] = useState<'overview' | 'users' | 'orgs' | 'logs'>('overview')

  // 统计
  const [stats, setStats]             = useState<AdminStats | null>(null)
  const [statsLoading, setStatsLoading] = useState(true)

  // 概览最近日志
  const [recentLogs, setRecentLogs]               = useState<AuditLogItem[]>([])
  const [recentLogsLoading, setRecentLogsLoading]   = useState(false)

  // 用户列表
  const [users, setUsers]           = useState<AdminUserListItem[]>([])
  const [userTotal, setUserTotal]     = useState(0)
  const [userPage, setUserPage]       = useState(1)
  const [userLoading, setUserLoading] = useState(false)
  const [roleFilter, setRoleFilter]   = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [keyword, setKeyword]         = useState('')
  const [keywordInput, setKeywordInput] = useState('')
  const [schoolFilter, setSchoolFilter] = useState('')
  const [schools, setSchools]         = useState<OrgListItem[]>([])
  const [schoolsLoaded, setSchoolsLoaded] = useState(false)
  const [detailUserId, setDetailUserId] = useState<string | null>(null)
  const [showCreateModal, setShowCreateModal] = useState(false)

  // ==================== 组织架构Tab状态 ====================

  // 原始数据
  const [regions, setRegions]         = useState<OrgListItem[]>([])  // 所有区域
  const [schools2, setSchools2]       = useState<OrgListItem[]>([])  // 选中区域的学校
  const [groups2, setGroups2]         = useState<GroupListItem[]>([]) // 选中学校的教研组

  // 选中项
  const [selRegion, setSelRegion]     = useState<OrgListItem | null>(null)
  const [selSchool, setSelSchool]     = useState<OrgListItem | null>(null)

  // 加载状态
  const [regLoading, setRegLoading]   = useState(false)
  const [schLoading, setSchLoading]   = useState(false)
  const [grpLoading, setGrpLoading]   = useState(false)

  // 展开成员管理的教研组ID
  const [expandedGroupId, setExpandedGroupId] = useState<string | null>(null)

  // 弹窗状态
  const [orgModal, setOrgModal] = useState<{
    open: boolean
    mode: 'create' | 'edit'
    type: 'region' | 'school'
    initial?: OrgListItem
  }>({ open: false, mode: 'create', type: 'region' })

  const [groupModal, setGroupModal] = useState<{
    open: boolean
    mode: 'create' | 'edit'
    initial?: GroupListItem
  }>({ open: false, mode: 'create' })

  // 删除确认对话框
  const [confirmDel, setConfirmDel] = useState<{
    open: boolean
    title: string
    message: string
    onConfirm: () => void
  }>({ open: false, title: '', message: '', onConfirm: () => {} })

  // 操作日志Tab
  const [logs, setLogs]           = useState<AuditLogItem[]>([])
  const [logTotal, setLogTotal]     = useState(0)
  const [logPage, setLogPage]       = useState(1)
  const [logLoading, setLogLoading] = useState(false)
  const [logFilterInput, setLogFilterInput] = useState({ username: '', action: '', startDate: '', endDate: '' })
  const [logFilters, setLogFilters]         = useState({ username: '', action: '', startDate: '', endDate: '' })
  const [expandedLogId, setExpandedLogId]   = useState<string | null>(null)

  // Toast
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)
  const showToast = (m: string, t: 'success' | 'error') => setToast({ message: m, type: t })

  // ==================== 加载函数 ====================

  const loadStats = useCallback(async () => {
    try { setStatsLoading(true); setStats(await getAdminStats()) } catch { } finally { setStatsLoading(false) }
  }, [])
  useEffect(() => { loadStats() }, [loadStats])

  const loadRecentLogs = useCallback(async () => {
    try { setRecentLogsLoading(true); setRecentLogs((await getAdminAuditLogs({ page: 1, page_size: 10 })).logs) } catch { } finally { setRecentLogsLoading(false) }
  }, [])
  useEffect(() => { if (activeTab === 'overview') loadRecentLogs() }, [activeTab, loadRecentLogs])

  // 加载学校列表（用于用户Tab学校筛选）
  const loadSchools = useCallback(async () => {
    if (schoolsLoaded) return
    try { const all = await getAdminOrgs(); setSchools(all.filter(o => o.type === 'school')); setSchoolsLoaded(true) } catch { }
  }, [schoolsLoaded])
  useEffect(() => { if (activeTab === 'users') loadSchools() }, [activeTab, loadSchools])

  const loadUsers = useCallback(async () => {
    try {
      setUserLoading(true)
      const data = await getAdminUsers({ page: userPage, page_size: 15, role: roleFilter, status: statusFilter, keyword, school_id: schoolFilter || undefined })
      setUsers(data.users); setUserTotal(data.total)
    } catch (e: unknown) { showToast(e instanceof Error ? e.message : '加载用户失败', 'error') } finally { setUserLoading(false) }
  }, [userPage, roleFilter, statusFilter, keyword, schoolFilter])
  useEffect(() => { if (activeTab === 'users') loadUsers() }, [activeTab, loadUsers])

  // 组织架构：加载所有区域
  const loadRegions = useCallback(async () => {
    try { setRegLoading(true); setRegions(await getAdminOrgs({ type: 'region' })) } catch { } finally { setRegLoading(false) }
  }, [])
  useEffect(() => { if (activeTab === 'orgs') { loadRegions(); setSelRegion(null); setSelSchool(null); setSchools2([]); setGroups2([]) } }, [activeTab, loadRegions])

  // 选中区域 → 加载该区域的学校
  const loadSchools2 = useCallback(async (regionId: string) => {
    try { setSchLoading(true); setSchools2(await getAdminOrgs({ type: 'school', parent_id: regionId })) } catch { } finally { setSchLoading(false) }
  }, [])

  // 选中学校 → 加载该学校的教研组
  const loadGroups2 = useCallback(async (schoolId: string) => {
    try { setGrpLoading(true); setGroups2(await getAdminGroups(schoolId)) } catch { } finally { setGrpLoading(false) }
  }, [])

  // ==================== 组织架构操作 ====================

  const handleSelectRegion = (r: OrgListItem) => {
    setSelRegion(r); setSelSchool(null); setGroups2([]); setExpandedGroupId(null)
    loadSchools2(r.id)
  }

  const handleSelectSchool = (s: OrgListItem) => {
    setSelSchool(s); setExpandedGroupId(null)
    loadGroups2(s.id)
  }

  // 删除区域
  const handleDeleteOrg = (org: OrgListItem) => {
    setConfirmDel({
      open: true,
      title: `删除${org.type === 'region' ? '区域' : '学校'}`,
      message: `确认删除「${org.name}」？此操作不可撤销。如有下属组织或成员，将无法删除。`,
      onConfirm: async () => {
        try {
          await deleteAdminOrg(org.id)
          showToast('删除成功', 'success')
          if (org.type === 'region') { loadRegions(); setSelRegion(null); setSchools2([]); setGroups2([]) }
          else { if (selRegion) loadSchools2(selRegion.id); setSelSchool(null); setGroups2([]) }
        } catch (e: unknown) { showToast(e instanceof Error ? e.message : '删除失败', 'error') }
        setConfirmDel(p => ({ ...p, open: false }))
      }
    })
  }

  // 禁用/启用组织
  const handleToggleOrgStatus = async (org: OrgListItem) => {
    const newStatus = org.status === 'active' ? 'disabled' : 'active'
    try {
      await updateAdminOrg(org.id, { name: org.name, admin_user_id: org.admin_user_id, status: newStatus })
      showToast(newStatus === 'active' ? '已启用' : '已禁用', 'success')
      if (org.type === 'region') loadRegions()
      else if (selRegion) loadSchools2(selRegion.id)
    } catch (e: unknown) { showToast(e instanceof Error ? e.message : '操作失败', 'error') }
  }

  // 删除教研组
  const handleDeleteGroup = (g: GroupListItem) => {
    setConfirmDel({
      open: true,
      title: '删除教研组',
      message: `确认删除教研组「${g.name}」？此操作不可撤销。`,
      onConfirm: async () => {
        try {
          await deleteAdminGroup(g.id)
          showToast('删除成功', 'success')
          if (selSchool) loadGroups2(selSchool.id)
        } catch (e: unknown) { showToast(e instanceof Error ? e.message : '删除失败', 'error') }
        setConfirmDel(p => ({ ...p, open: false }))
      }
    })
  }

  // ==================== 日志Tab ====================

  const loadLogs = useCallback(async () => {
    try {
      setLogLoading(true)
      const data = await getAdminAuditLogs({ page: logPage, page_size: 20, action: logFilters.action || undefined, username: logFilters.username || undefined, start_date: logFilters.startDate || undefined, end_date: logFilters.endDate || undefined })
      setLogs(data.logs); setLogTotal(data.total)
    } catch { } finally { setLogLoading(false) }
  }, [logPage, logFilters])
  useEffect(() => { if (activeTab === 'logs') loadLogs() }, [activeTab, loadLogs])

  const handleLogSearch = () => { setLogFilters({ ...logFilterInput }); setLogPage(1); setExpandedLogId(null) }
  const handleLogReset = () => { const e = { username: '', action: '', startDate: '', endDate: '' }; setLogFilterInput(e); setLogFilters(e); setLogPage(1); setExpandedLogId(null) }
  const toggleLogDetail = (id: string) => setExpandedLogId(p => p === id ? null : id)

  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const handleKeywordChange = (v: string) => {
    setKeywordInput(v)
    if (searchTimer.current) clearTimeout(searchTimer.current)
    searchTimer.current = setTimeout(() => { setKeyword(v); setUserPage(1) }, 400)
  }

  const totalPages    = Math.ceil(userTotal / 15)
  const logTotalPages = Math.ceil(logTotal / 20)

  const inputStyle: React.CSSProperties = { padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', background: C.white, color: C.text }
  const getActionStyle = (a: string) => a.startsWith('user.') ? { bg: C.primaryLight, color: C.primary } : a.startsWith('admin.') ? { bg: C.purpleLight, color: C.purple } : { bg: C.warningLight, color: C.warning }

  // ==================== 组织架构：行级操作按钮通用样式 ====================

  const rowBtn = (color: string, bgColor: string): React.CSSProperties => ({
    padding: '3px 8px', borderRadius: '5px', border: `1px solid ${bgColor}`,
    background: bgColor, color: color, fontSize: '11px', cursor: 'pointer', fontWeight: 500,
  })

  // ==================== 组织架构：单栏卡片容器 ====================

  const ColCard = ({ title, count, onAdd, addLabel, loading, empty, children }: {
    title: React.ReactNode; count?: number; onAdd?: () => void; addLabel?: string
    loading?: boolean; empty?: string; children: React.ReactNode
  }) => (
    <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, overflow: 'hidden', display: 'flex', flexDirection: 'column', minHeight: '400px' }}>
      {/* 栏头 */}
      <div style={{ padding: '14px 16px', borderBottom: `1px solid ${C.border}`, background: C.bg, display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0 }}>
        <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, display: 'flex', alignItems: 'center', gap: '6px' }}>
          {title}
          {count !== undefined && <span style={{ fontSize: '11px', color: C.textMuted, fontWeight: 400 }}>({count})</span>}
        </div>
        {onAdd && (
          <button onClick={onAdd} style={{ padding: '5px 12px', borderRadius: '7px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '12px', fontWeight: 600, cursor: 'pointer' }}>
            + {addLabel || '新建'}
          </button>
        )}
      </div>
      {/* 内容 */}
      <div style={{ flex: 1, overflowY: 'auto' }}>
        {loading ? (
          <div style={{ padding: '32px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>加载中...</div>
        ) : (
          <>
            {children}
            {empty && (
              <div style={{ padding: '32px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>{empty}</div>
            )}
          </>
        )}
      </div>
    </div>
  )

  // ==================== 渲染 ====================

  return (
    <div style={{ minHeight: '100vh', background: 'linear-gradient(135deg,#EEF2FF 0%,#FAFBFC 50%,#F0FDF4 100%)' }}>
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
      {confirmDel.open && (
        <ConfirmDialog
          title={confirmDel.title} message={confirmDel.message}
          onConfirm={confirmDel.onConfirm}
          onCancel={() => setConfirmDel(p => ({ ...p, open: false }))}
        />
      )}
      {detailUserId && (
        <UserDetailModal userId={detailUserId} onClose={() => setDetailUserId(null)} onAction={() => { loadUsers(); loadStats(); showToast('操作成功', 'success') }} />
      )}
      {showCreateModal && (
        <CreateUserModal onClose={() => setShowCreateModal(false)} onCreated={() => { loadUsers(); loadStats(); showToast('用户创建成功', 'success') }} />
      )}
      {orgModal.open && (
        <OrgFormModal
          mode={orgModal.mode} type={orgModal.type} initial={orgModal.initial}
          regions={regions}
          onClose={() => setOrgModal(p => ({ ...p, open: false }))}
          onSaved={() => {
            showToast(orgModal.mode === 'create' ? '创建成功' : '更新成功', 'success')
            if (orgModal.type === 'region') loadRegions()
            else if (selRegion) loadSchools2(selRegion.id)
            loadStats()
          }}
        />
      )}
      {groupModal.open && selSchool && (
        <GroupFormModal
          mode={groupModal.mode} schoolId={selSchool.id} schoolName={selSchool.name} initial={groupModal.initial}
          onClose={() => setGroupModal(p => ({ ...p, open: false }))}
          onSaved={() => {
            showToast(groupModal.mode === 'create' ? '创建成功' : '更新成功', 'success')
            loadGroups2(selSchool.id); loadStats()
          }}
        />
      )}

      {/* 顶部导航 */}
      <header style={{ height: '64px', position: 'sticky', top: 0, zIndex: 100, background: 'rgba(255,255,255,0.88)', backdropFilter: 'blur(20px)', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', padding: '0 32px', gap: '16px' }}>
        <button onClick={() => navigate(fromPath)} style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.white }}>{'<- 返回'}</button>
        <div style={{ flex: 1, textAlign: 'center' }}>
          <h1 style={{ fontSize: '18px', fontWeight: 700, color: C.text, margin: 0 }}>👥 用户管理中心</h1>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>统一管理用户、组织架构与操作日志</div>
        </div>
        <button onClick={() => setShowCreateModal(true)} style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>+ 新建用户</button>
      </header>

      <div style={{ maxWidth: '1400px', margin: '0 auto', padding: '24px' }}>
        {/* Tab 切换 */}
        <div style={{ display: 'flex', gap: '4px', marginBottom: '20px', background: C.bg, borderRadius: '12px', padding: '4px', border: `1px solid ${C.border}`, width: 'fit-content' }}>
          {(['overview','users','orgs','logs'] as const).map((tab, i) => {
            const labels = ['📊 概览','👥 用户管理','🏫 组织架构','📋 操作日志']
            return (
              <button key={tab} onClick={() => setActiveTab(tab)} style={{ padding: '9px 22px', borderRadius: '9px', border: 'none', cursor: 'pointer', fontSize: '14px', fontWeight: activeTab === tab ? 600 : 400, color: activeTab === tab ? C.primary : C.textSec, background: activeTab === tab ? C.white : 'transparent', boxShadow: activeTab === tab ? '0 1px 4px rgba(0,0,0,0.08)' : 'none', transition: 'all 150ms ease' }}>
                {labels[i]}
              </button>
            )
          })}
        </div>

        {/* ===== Tab: 概览 ===== */}
        {activeTab === 'overview' && (
          <div>
            {statsLoading ? <div style={{ textAlign: 'center', padding: '60px', color: C.textMuted }}>加载统计数据...</div> : stats ? (
              <>
                <div style={{ display: 'flex', gap: '16px', marginBottom: '16px' }}>
                  <StatCard label="总用户数" value={stats.total_users} sub={`活跃 ${stats.active_users} · 禁用 ${stats.disabled_users}`} color={C.primary} />
                  <StatCard label="组织总数" value={stats.total_orgs} sub={`学校 ${stats.total_schools} 所`} color={C.success} />
                  <StatCard label="教研组" value={stats.total_groups} sub={`教研组成员 ${stats.total_members} 人`} color={C.warning} />
                </div>
                <RoleBarChart stats={stats} />
                <RecentLogsCard logs={recentLogs} loading={recentLogsLoading} onViewAll={() => { setActiveTab('logs'); setLogFilters({ username: '', action: '', startDate: '', endDate: '' }); setLogFilterInput({ username: '', action: '', startDate: '', endDate: '' }); setLogPage(1) }} />
                <div style={{ background: C.primaryLight, borderRadius: '12px', border: `1px solid ${C.primary}22`, padding: '16px 20px', fontSize: '13px', color: C.primary }}>
                  💡 点击上方"🏫 组织架构"Tab管理区域、学校和教研组，支持完整的创建、编辑、删除和成员管理。
                </div>
              </>
            ) : null}
          </div>
        )}

        {/* ===== Tab: 用户管理 ===== */}
        {activeTab === 'users' && (
          <div>
            <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, padding: '16px 20px', marginBottom: '16px', display: 'flex', gap: '10px', alignItems: 'center', flexWrap: 'wrap' }}>
              <input value={keywordInput} onChange={e => handleKeywordChange(e.target.value)} placeholder="搜索用户名或显示名..."
                style={{ flex: 1, minWidth: '180px', padding: '8px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none' }}
                onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                onBlur={e => { e.currentTarget.style.borderColor = C.border }}
              />
              <select value={roleFilter} onChange={e => { setRoleFilter(e.target.value); setUserPage(1) }} style={{ padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: C.white }}>
                {ROLE_OPTIONS.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
              </select>
              <select value={statusFilter} onChange={e => { setStatusFilter(e.target.value); setUserPage(1) }} style={{ padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: C.white }}>
                <option value="">全部状态</option>
                <option value="active">正常</option>
                <option value="disabled">已禁用</option>
              </select>
              <select value={schoolFilter} onChange={e => { setSchoolFilter(e.target.value); setUserPage(1) }} style={{ padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: C.white, minWidth: '120px' }}>
                <option value="">全部学校</option>
                {schools.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
              <div style={{ fontSize: '13px', color: C.textMuted }}>共 {userTotal} 个用户</div>
            </div>
            <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
              <div style={{ display: 'grid', gridTemplateColumns: '2fr 1.5fr 1.5fr 1.2fr 1.5fr 1fr', padding: '12px 20px', background: C.bg, borderBottom: `1px solid ${C.border}`, fontSize: '12px', fontWeight: 600, color: C.textSec }}>
                <span>用户</span><span>课件审核角色</span><span>教案系统归属</span><span>状态</span><span>最近登录</span><span>操作</span>
              </div>
              {userLoading ? <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div> : users.length === 0 ? <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>暂无用户</div> : (
                users.map((user, idx) => (
                  <div key={user.id} style={{ display: 'grid', gridTemplateColumns: '2fr 1.5fr 1.5fr 1.2fr 1.5fr 1fr', padding: '14px 20px', alignItems: 'center', borderBottom: idx < users.length - 1 ? `1px solid ${C.border}` : 'none', transition: 'background 150ms ease' }}
                    onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
                    onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                      <div style={{ width: '34px', height: '34px', borderRadius: '50%', flexShrink: 0, background: `linear-gradient(135deg,${C.primary},#7C3AED)`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: '13px', fontWeight: 700 }}>{user.display_name?.charAt(0)?.toUpperCase() || 'U'}</div>
                      <div><div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{user.display_name}</div><div style={{ fontSize: '12px', color: C.textMuted }}>@{user.username}</div></div>
                    </div>
                    <div><RoleBadge role={user.role} roleName={user.role_name} /></div>
                    <div>{user.group_count > 0 ? <div><div style={{ fontSize: '13px', color: C.text, fontWeight: 500 }}>{user.group_name || '-'}</div><div style={{ fontSize: '11px', color: C.textMuted }}>{user.school_name}{user.group_count > 1 ? ` 等${user.group_count}个组` : ''}</div></div> : <span style={{ fontSize: '12px', color: C.textMuted }}>未加入教研组</span>}</div>
                    <div><StatusBadge status={user.status} /></div>
                    <div style={{ fontSize: '12px', color: C.textSec }}>{user.last_login_at ? user.last_login_at.replace('T',' ').substring(0,16) : '从未登录'}<div style={{ fontSize: '11px', color: C.textMuted }}>共{user.login_count}次</div></div>
                    <div><button onClick={() => setDetailUserId(user.id)} style={{ padding: '5px 14px', borderRadius: '7px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '12px', color: C.primary, cursor: 'pointer', fontWeight: 500 }}
                      onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.primaryLight }}
                      onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}>详情</button></div>
                  </div>
                ))
              )}
            </div>
            {totalPages > 1 && (
              <div style={{ display: 'flex', justifyContent: 'center', gap: '8px', marginTop: '16px', alignItems: 'center' }}>
                <button onClick={() => setUserPage(p => Math.max(1,p-1))} disabled={userPage===1} style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: userPage===1 ? C.textMuted : C.text, cursor: userPage===1 ? 'not-allowed' : 'pointer' }}>上一页</button>
                <span style={{ fontSize: '13px', color: C.textSec }}>第 {userPage} / {totalPages} 页</span>
                <button onClick={() => setUserPage(p => Math.min(totalPages,p+1))} disabled={userPage===totalPages} style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: userPage===totalPages ? C.textMuted : C.text, cursor: userPage===totalPages ? 'not-allowed' : 'pointer' }}>下一页</button>
              </div>
            )}
          </div>
        )}

        {/* ===== Tab: 组织架构（三栏递进）===== */}
        {activeTab === 'orgs' && (
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '16px', alignItems: 'start' }}>

            {/* ---- 左栏：区域列表 ---- */}
            <ColCard
              title="🌍 区域"
              count={regions.length}
              onAdd={() => setOrgModal({ open: true, mode: 'create', type: 'region' })}
              addLabel="新建区域"
              loading={regLoading}
              empty={regions.length === 0 ? '暂无区域，点击右上角新建' : undefined}
            >
              {regions.map(r => (
                <div key={r.id}>
                  {/* 区域主行 */}
                  <div
                    onClick={() => handleSelectRegion(r)}
                    style={{
                      padding: '12px 14px', cursor: 'pointer',
                      background: selRegion?.id === r.id ? C.primaryLight : 'transparent',
                      borderLeft: selRegion?.id === r.id ? `3px solid ${C.primary}` : '3px solid transparent',
                      transition: 'all 150ms ease',
                    }}
                    onMouseEnter={e => { if (selRegion?.id !== r.id) (e.currentTarget as HTMLElement).style.background = C.bg }}
                    onMouseLeave={e => { if (selRegion?.id !== r.id) (e.currentTarget as HTMLElement).style.background = 'transparent' }}
                  >
                    {/* 区域名 + 状态 */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '4px' }}>
                      <span style={{ fontSize: '14px', fontWeight: 600, color: selRegion?.id === r.id ? C.primary : C.text, flex: 1 }}>{r.name}</span>
                      <StatusBadge status={r.status} />
                    </div>
                    {/* 管理员 + 学校数 */}
                    <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '8px' }}>
                      {r.admin_user_name ? `管理员：${r.admin_user_name}` : '暂无管理员'} · {r.group_count + (r.member_count > 0 ? 0 : 0)} 下属学校
                    </div>
                    {/* 操作按钮行 */}
                    <div style={{ display: 'flex', gap: '6px' }} onClick={e => e.stopPropagation()}>
                      <button
                        onClick={() => setOrgModal({ open: true, mode: 'edit', type: 'region', initial: r })}
                        style={rowBtn(C.primary, C.primaryLight)}>✏️ 编辑</button>
                      <button
                        onClick={() => handleToggleOrgStatus(r)}
                        style={rowBtn(r.status === 'active' ? C.danger : C.success, r.status === 'active' ? C.dangerLight : C.successLight)}>
                        {r.status === 'active' ? '🚫 禁用' : '✅ 启用'}
                      </button>
                      <button onClick={() => handleDeleteOrg(r)} style={rowBtn(C.danger, C.dangerLight)}>🗑️ 删除</button>
                    </div>
                  </div>
                  {/* 分隔线 */}
                  <div style={{ height: '1px', background: C.border, margin: '0 14px' }} />
                </div>
              ))}
            </ColCard>

            {/* ---- 中栏：学校列表 ---- */}
            <ColCard
              title={selRegion ? <span>🏫 <span style={{ color: C.primary }}>{selRegion.name}</span> 的学校</span> : '🏫 学校'}
              count={schools2.length}
              onAdd={selRegion ? () => setOrgModal({ open: true, mode: 'create', type: 'school' }) : undefined}
              addLabel="新建学校"
              loading={schLoading}
              empty={!selRegion ? '← 请先选择左侧区域' : schools2.length === 0 ? '暂无学校，点击右上角新建' : undefined}
            >
              {schools2.map(s => (
                <div key={s.id}>
                  <div
                    onClick={() => handleSelectSchool(s)}
                    style={{
                      padding: '12px 14px', cursor: 'pointer',
                      background: selSchool?.id === s.id ? C.primaryLight : 'transparent',
                      borderLeft: selSchool?.id === s.id ? `3px solid ${C.primary}` : '3px solid transparent',
                      transition: 'all 150ms ease',
                    }}
                    onMouseEnter={e => { if (selSchool?.id !== s.id) (e.currentTarget as HTMLElement).style.background = C.bg }}
                    onMouseLeave={e => { if (selSchool?.id !== s.id) (e.currentTarget as HTMLElement).style.background = 'transparent' }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '4px' }}>
                      <span style={{ fontSize: '14px', fontWeight: 600, color: selSchool?.id === s.id ? C.primary : C.text, flex: 1 }}>{s.name}</span>
                      <StatusBadge status={s.status} />
                    </div>
                    <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '8px' }}>
                      {s.admin_user_name ? `管理员：${s.admin_user_name}` : '暂无管理员'} · {s.group_count} 个教研组
                    </div>
                    <div style={{ display: 'flex', gap: '6px' }} onClick={e => e.stopPropagation()}>
                      <button onClick={() => setOrgModal({ open: true, mode: 'edit', type: 'school', initial: s })} style={rowBtn(C.primary, C.primaryLight)}>✏️ 编辑</button>
                      <button onClick={() => handleToggleOrgStatus(s)} style={rowBtn(s.status === 'active' ? C.danger : C.success, s.status === 'active' ? C.dangerLight : C.successLight)}>{s.status === 'active' ? '🚫 禁用' : '✅ 启用'}</button>
                      <button onClick={() => handleDeleteOrg(s)} style={rowBtn(C.danger, C.dangerLight)}>🗑️ 删除</button>
                    </div>
                  </div>
                  <div style={{ height: '1px', background: C.border, margin: '0 14px' }} />
                </div>
              ))}
            </ColCard>

            {/* ---- 右栏：教研组列表 ---- */}
            <ColCard
              title={selSchool ? <span>👨‍🏫 <span style={{ color: C.primary }}>{selSchool.name}</span> 的教研组</span> : '👨‍🏫 教研组'}
              count={groups2.length}
              onAdd={selSchool ? () => setGroupModal({ open: true, mode: 'create' }) : undefined}
              addLabel="新建教研组"
              loading={grpLoading}
              empty={!selSchool ? '← 请先选择中间学校' : groups2.length === 0 ? '暂无教研组，点击右上角新建' : undefined}
            >
              {groups2.map(g => (
                <div key={g.id}>
                  {/* 教研组主行 */}
                  <div style={{ padding: '12px 14px' }}>
                    <div style={{ display: 'flex', alignItems: 'flex-start', gap: '6px', marginBottom: '4px' }}>
                      <div style={{ flex: 1 }}>
                        <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{g.name}</div>
                        <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '2px' }}>
                          {g.subject}{g.grade_range ? ` · ${g.grade_range}` : ''} · {g.lead_user_name ? `组长：${g.lead_user_name}` : '暂无组长'} · {g.member_count} 人
                        </div>
                      </div>
                      <StatusBadge status={g.status} />
                    </div>
                    {/* 操作按钮 */}
                    <div style={{ display: 'flex', gap: '6px', marginTop: '8px' }}>
                      <button onClick={() => setGroupModal({ open: true, mode: 'edit', initial: g })} style={rowBtn(C.primary, C.primaryLight)}>✏️ 编辑</button>
                      <button onClick={() => handleDeleteGroup(g)} style={rowBtn(C.danger, C.dangerLight)}>🗑️ 删除</button>
                      <button
                        onClick={() => setExpandedGroupId(p => p === g.id ? null : g.id)}
                        style={rowBtn(expandedGroupId === g.id ? C.purple : C.textSec, expandedGroupId === g.id ? C.purpleLight : C.bg)}>
                        {expandedGroupId === g.id ? '收起 ▲' : '👥 成员'}
                      </button>
                    </div>
                  </div>

                  {/* 成员管理展开面板 */}
                  {expandedGroupId === g.id && (
                    <MemberPanel groupId={g.id} onClose={() => setExpandedGroupId(null)} />
                  )}

                  <div style={{ height: '1px', background: C.border, margin: '0 14px' }} />
                </div>
              ))}
            </ColCard>
          </div>
        )}

        {/* ===== Tab: 操作日志 ===== */}
        {activeTab === 'logs' && (
          <div>
            <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, padding: '16px 20px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', gap: '10px', alignItems: 'center', flexWrap: 'wrap', marginBottom: '12px' }}>
                <input value={logFilterInput.username} onChange={e => setLogFilterInput(p => ({ ...p, username: e.target.value }))} placeholder="搜索用户名 / 显示名..."
                  style={{ ...inputStyle, flex: '1 1 160px', minWidth: '140px' }}
                  onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                  onBlur={e => { e.currentTarget.style.borderColor = C.border }}
                  onKeyDown={e => { if (e.key === 'Enter') handleLogSearch() }}
                />
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <span style={{ fontSize: '12px', color: C.textSec, whiteSpace: 'nowrap' }}>开始</span>
                  <input type="date" value={logFilterInput.startDate} onChange={e => setLogFilterInput(p => ({ ...p, startDate: e.target.value }))} style={{ ...inputStyle, width: '140px' }}
                    onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                    onBlur={e => { e.currentTarget.style.borderColor = C.border }} />
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <span style={{ fontSize: '12px', color: C.textSec, whiteSpace: 'nowrap' }}>结束</span>
                  <input type="date" value={logFilterInput.endDate} onChange={e => setLogFilterInput(p => ({ ...p, endDate: e.target.value }))} style={{ ...inputStyle, width: '140px' }}
                    onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
                    onBlur={e => { e.currentTarget.style.borderColor = C.border }} />
                </div>
                <select value={logFilterInput.action} onChange={e => setLogFilterInput(p => ({ ...p, action: e.target.value }))} style={{ ...inputStyle, minWidth: '130px' }}>
                  {ACTION_OPTIONS.map(a => <option key={a.value} value={a.value}>{a.label}</option>)}
                </select>
              </div>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <button onClick={handleLogSearch} style={{ padding: '7px 20px', borderRadius: '8px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>🔍 查询</button>
                <button onClick={handleLogReset} style={{ padding: '7px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.bg, color: C.textSec, fontSize: '13px', cursor: 'pointer' }}>重置</button>
                {(logFilters.username || logFilters.startDate || logFilters.endDate || logFilters.action) && (
                  <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap', alignItems: 'center' }}>
                    {logFilters.username && <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: C.primaryLight, color: C.primary }}>用户：{logFilters.username}</span>}
                    {logFilters.action && <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: C.warningLight, color: C.warning }}>{ACTION_OPTIONS.find(a => a.value === logFilters.action)?.label || logFilters.action}</span>}
                    {logFilters.startDate && <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: C.successLight, color: C.success }}>{logFilters.startDate} 起</span>}
                    {logFilters.endDate && <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '12px', background: C.successLight, color: C.success }}>至 {logFilters.endDate}</span>}
                  </div>
                )}
                <div style={{ marginLeft: 'auto', fontSize: '13px', color: C.textMuted }}>共 {logTotal} 条记录</div>
              </div>
            </div>
            <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
              <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 2fr 1.2fr 0.9fr 1.3fr 0.6fr', padding: '12px 20px', background: C.bg, borderBottom: `1px solid ${C.border}`, fontSize: '12px', fontWeight: 600, color: C.textSec }}>
                <span>操作者</span><span>操作内容摘要</span><span>操作类型</span><span>IP地址</span><span>时间</span><span>详情</span>
              </div>
              {logLoading ? <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div> : logs.length === 0 ? <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>暂无日志记录</div> : (
                logs.map((log, idx) => {
                  const isExpanded = expandedLogId === log.id
                  const s = getActionStyle(log.action)
                  return (
                    <div key={log.id} style={{ borderBottom: idx < logs.length-1 ? `1px solid ${C.border}` : 'none' }}>
                      <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 2fr 1.2fr 0.9fr 1.3fr 0.6fr', padding: '12px 20px', alignItems: 'center', fontSize: '13px', background: isExpanded ? 'rgba(79,123,232,0.03)' : 'transparent', transition: 'background 150ms ease' }}
                        onMouseEnter={e => { if (!isExpanded) (e.currentTarget as HTMLElement).style.background = C.bg }}
                        onMouseLeave={e => { if (!isExpanded) (e.currentTarget as HTMLElement).style.background = 'transparent' }}>
                        <div><div style={{ fontWeight: 500, color: C.text }}>{log.display_name || log.username}</div><div style={{ fontSize: '11px', color: C.textMuted }}>@{log.username}</div></div>
                        <div style={{ color: C.textSec, fontSize: '12px', fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {(() => { try { return Object.entries(JSON.parse(log.detail)).map(([k,v]) => `${k}: ${v}`).join('  ·  ') || '—' } catch { return log.detail || '—' } })()}
                        </div>
                        <div><span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '11px', background: s.bg, color: s.color, fontWeight: 600 }}>{log.action_name}</span></div>
                        <div style={{ color: C.textMuted, fontFamily: 'monospace', fontSize: '11px' }}>{log.ip || '—'}</div>
                        <div style={{ color: C.textSec, fontSize: '11px' }}>{fmt(typeof log.created_at === 'string' ? log.created_at : new Date(log.created_at).toISOString())}</div>
                        <div><button onClick={() => toggleLogDetail(log.id)} style={{ padding: '4px 10px', borderRadius: '6px', cursor: 'pointer', border: `1px solid ${isExpanded ? C.primary : C.border}`, background: isExpanded ? C.primaryLight : C.bg, color: isExpanded ? C.primary : C.textSec, fontSize: '11px', fontWeight: 600, transition: 'all 150ms ease', whiteSpace: 'nowrap' }}>{isExpanded ? '收起 ▲' : '详情 ▼'}</button></div>
                      </div>
                      {isExpanded && (
                        <div style={{ padding: '12px 20px 16px', background: 'rgba(79,123,232,0.03)', borderTop: `1px dashed ${C.border}` }}>
                          <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '8px' }}>📄 完整操作详情</div>
                          <pre style={{ margin: 0, padding: '12px 16px', background: C.bg, borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '12px', color: C.text, fontFamily: '"Fira Code","Cascadia Code",Consolas,monospace', lineHeight: 1.6, overflowX: 'auto', whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                            {(() => { try { return JSON.stringify(JSON.parse(log.detail), null, 2) } catch { return log.detail || '（无详情数据）' } })()}
                          </pre>
                        </div>
                      )}
                    </div>
                  )
                })
              )}
            </div>
            {logTotalPages > 1 && (
              <div style={{ display: 'flex', justifyContent: 'center', gap: '8px', marginTop: '16px', alignItems: 'center' }}>
                <button onClick={() => setLogPage(p => Math.max(1,p-1))} disabled={logPage===1} style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: logPage===1 ? C.textMuted : C.text, cursor: logPage===1 ? 'not-allowed' : 'pointer' }}>上一页</button>
                <span style={{ fontSize: '13px', color: C.textSec }}>第 {logPage} / {logTotalPages} 页</span>
                <button onClick={() => setLogPage(p => Math.min(logTotalPages,p+1))} disabled={logPage===logTotalPages} style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: logPage===logTotalPages ? C.textMuted : C.text, cursor: logPage===logTotalPages ? 'not-allowed' : 'pointer' }}>下一页</button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
