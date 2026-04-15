/**
 * GroupFormModal.tsx — 教研组 新建/编辑弹窗
 *
 * v109改动：
 *   - 移除单一"教研组长"选择器（已由成员管理支持多组长）
 *   - 新增提示说明，引导通过成员面板设置组长
 */
import { useState } from 'react'
import { createAdminGroup, updateAdminGroup } from '@/api/admin'
import type { GroupListItem, CreateGroupRequest, UpdateGroupRequest } from '@/api/admin'
import { C } from './adminConstants'

interface GroupFormModalProps {
  mode: 'create' | 'edit'
  schoolId: string
  schoolName: string
  initial?: GroupListItem
  onClose: () => void
  onSaved: () => void
}

export function GroupFormModal({
  mode, schoolId, schoolName, initial, onClose, onSaved,
}: GroupFormModalProps) {
  const [name, setName]             = useState(initial?.name || '')
  const [subject, setSubject]       = useState(initial?.subject || '')
  const [gradeRange, setGradeRange] = useState(initial?.grade_range || '')
  const [desc, setDesc]             = useState('')
  const [saving, setSaving]         = useState(false)
  const [error, setError]           = useState('')

  const fieldStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: '8px',
    border: `1px solid ${C.border}`, fontSize: '14px',
    outline: 'none', boxSizing: 'border-box',
  }

  const handleSave = async () => {
    if (!name.trim())    { setError('请输入教研组名称'); return }
    if (!subject.trim()) { setError('请输入学科'); return }
    try {
      setSaving(true); setError('')
      if (mode === 'create') {
        const req: CreateGroupRequest = {
          name: name.trim(),
          school_id: schoolId,
          subject: subject.trim(),
          grade_range: gradeRange.trim() || undefined,
          description: desc.trim() || undefined,
        }
        await createAdminGroup(req)
      } else {
        const req: UpdateGroupRequest = {
          name: name.trim(),
          subject: subject.trim(),
          grade_range: gradeRange.trim() || undefined,
          description: desc.trim() || undefined,
        }
        await updateAdminGroup(initial!.id, req)
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
        background: C.white, borderRadius: '20px', width: '500px',
        maxHeight: '88vh', overflow: 'hidden',
        display: 'flex', flexDirection: 'column',
        boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }}>
        {/* 头部 */}
        <div style={{
          padding: '20px 24px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        }}>
          <div>
            <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>
              {mode === 'create' ? '新建教研组' : '编辑教研组'}
            </div>
            <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>
              所属学校：{schoolName}
            </div>
          </div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>×</button>
        </div>

        {/* 表单 */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '24px' }}>
          {error && (
            <div style={{ padding: '10px 14px', borderRadius: '8px', marginBottom: '14px', background: '#FEF2F2', color: '#EF4444', fontSize: '13px' }}>
              {error}
            </div>
          )}

          {/* 教研组名称 */}
          <div style={{ marginBottom: '14px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
              教研组名称 <span style={{ color: '#EF4444' }}>*</span>
            </label>
            <input value={name} onChange={e => setName(e.target.value)}
              placeholder="例如：七年级AI课程教研组"
              style={fieldStyle}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border }}
            />
          </div>

          {/* 学科 */}
          <div style={{ marginBottom: '14px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
              学科 <span style={{ color: '#EF4444' }}>*</span>
            </label>
            <input value={subject} onChange={e => setSubject(e.target.value)}
              placeholder="例如：AI课程"
              style={fieldStyle}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border }}
            />
          </div>

          {/* 年级范围 */}
          <div style={{ marginBottom: '14px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
              年级范围（可选）
            </label>
            <input value={gradeRange} onChange={e => setGradeRange(e.target.value)}
              placeholder="例如：G7-G9"
              style={fieldStyle}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border }}
            />
          </div>

          {/* 描述 */}
          <div style={{ marginBottom: '14px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
              描述（可选）
            </label>
            <textarea
              value={desc} onChange={e => setDesc(e.target.value)}
              placeholder="教研组简介..." rows={3}
              style={{ ...fieldStyle, resize: 'vertical', fontFamily: 'inherit' }}
              onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
              onBlur={e => { e.currentTarget.style.borderColor = C.border }}
            />
          </div>

          {/* 多组长说明提示：引导用户通过成员管理设置组长 */}
          <div style={{
            padding: '12px 14px', borderRadius: '10px', marginBottom: '16px',
            background: '#FFFBEB', border: '1px solid #FDE68A',
            fontSize: '13px', color: '#92400E', lineHeight: 1.7,
          }}>
            👑 <strong>组长设置</strong>：创建后在「成员管理」中将成员角色改为「教研组长」即可设置组长。
            支持多个组长，组长拥有教案评审权限。
          </div>

          {/* 操作按钮 */}
          <div style={{ display: 'flex', gap: '10px' }}>
            <button onClick={onClose} style={{
              flex: 1, padding: '10px', borderRadius: '10px',
              border: `1px solid ${C.border}`, background: C.bg,
              fontSize: '14px', color: C.textSec, cursor: 'pointer',
            }}>
              取消
            </button>
            <button
              onClick={handleSave} disabled={saving}
              style={{
                flex: 2, padding: '10px', borderRadius: '10px', border: 'none',
                background: saving ? '#E5E7EB' : `linear-gradient(135deg,${C.primary},#7C3AED)`,
                color: saving ? '#9CA3AF' : '#fff',
                fontSize: '14px', fontWeight: 600,
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
