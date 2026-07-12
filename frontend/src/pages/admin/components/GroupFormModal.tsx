/**
 * GroupFormModal — 教研组新建/编辑弹窗（组织架构 Tab 第三栏使用）
 *
 * 两种模式：
 *   - create：在指定学校（schoolId/schoolName 由父组件传入）下新建教研组
 *   - edit  ：编辑既有教研组（initial 为列表项 GroupListItem）
 *
 * 字段：名称*、学科*、年级范围（选填）、描述（选填）
 *
 * 账户与权限修复批新增（本次，教研组描述抹除修复）：
 *   - 根因：编辑模式的初始值来自列表项 GroupListItem，该类型不含 description 字段，
 *     desc 初始恒为空串；而保存时又总是提交 description
 *     → 只要打开"编辑"再点保存（哪怕只改名称），已有描述就被静默抹成空。
 *   - 修法：编辑模式打开时调 getAdminGroupDetail(initial.id) 拉取完整详情，
 *     回填 name/subject/grade_range/description 四字段（详情为权威数据源，
 *     列表项字段仅作加载完成前的占位显示）；
 *     详情加载中禁用保存按钮并显示"加载详情中..."；
 *     加载失败红字提示并保持禁用（宁可不让保存，也不能用半截数据覆盖库里的完整数据）。
 *   - 配套说明：后端 UpdateTeachingGroup 的 settings/status 空值覆盖问题
 *     （与组织侧 B10 同根因：repo 全量覆盖 + 空值兜底）在
 *     organization_service.go 侧以"缺省保留现值"方式另行修复，见该文件注释。
 */
import { useState, useEffect } from 'react'
import { createAdminGroup, updateAdminGroup, getAdminGroupDetail } from '@/api/admin'
import type { GroupListItem } from '@/api/admin'
import { C } from './adminConstants'

export function GroupFormModal({ mode, schoolId, schoolName, initial, onClose, onSaved }: {
  mode: 'create' | 'edit'
  schoolId: string
  schoolName: string
  initial?: GroupListItem
  onClose: () => void
  onSaved: () => void
}) {
  // ---- 表单状态（编辑模式下先用列表项占位，详情返回后以详情为准整体回填）----
  const [name, setName]             = useState(initial?.name || '')
  const [subject, setSubject]       = useState(initial?.subject || '')
  const [gradeRange, setGradeRange] = useState(initial?.grade_range || '')
  const [desc, setDesc]             = useState('')

  // ---- 详情加载状态（仅编辑模式使用）----
  // detailLoading=true 或 detailError 非空时禁用保存，防止半截数据覆盖库中完整数据
  const [detailLoading, setDetailLoading] = useState(mode === 'edit')
  const [detailError, setDetailError]     = useState('')

  // ---- 提交状态 ----
  const [saving, setSaving] = useState(false)
  const [error, setError]   = useState('')

  // 编辑模式：打开弹窗即拉取教研组完整详情，回填含 description 在内的全部可编辑字段
  useEffect(() => {
    if (mode !== 'edit' || !initial) return
    let cancelled = false // 组件卸载/弹窗关闭后不再 setState
    ;(async () => {
      try {
        setDetailLoading(true)
        const d = await getAdminGroupDetail(initial.id)
        if (cancelled) return
        // 以详情为权威数据源整体回填（覆盖列表项占位值）
        setName(d.name)
        setSubject(d.subject)
        setGradeRange(d.grade_range || '')
        setDesc(d.description || '')
      } catch (e: unknown) {
        if (!cancelled) {
          setDetailError(e instanceof Error && e.message
            ? `加载教研组详情失败：${e.message}`
            : '加载教研组详情失败，请关闭弹窗后重试')
        }
      } finally {
        if (!cancelled) setDetailLoading(false)
      }
    })()
    return () => { cancelled = true }
  }, [mode, initial])

  // 保存：create 走新建接口；edit 走更新接口（提交详情回填后的完整字段）
  const handleSave = async () => {
    const trimmedName = name.trim()
    const trimmedSubject = subject.trim()
    if (!trimmedName)    { setError('请填写教研组名称'); return }
    if (!trimmedSubject) { setError('请填写学科'); return }
    setError('')
    setSaving(true)
    try {
      if (mode === 'create') {
        await createAdminGroup({
          school_id: schoolId,
          name: trimmedName,
          subject: trimmedSubject,
          grade_range: gradeRange.trim(),
          description: desc.trim(),
        })
      } else if (initial) {
        await updateAdminGroup(initial.id, {
          name: trimmedName,
          subject: trimmedSubject,
          grade_range: gradeRange.trim(),
          description: desc.trim(),
        })
      }
      onSaved()
      onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '保存失败，请重试')
    } finally {
      setSaving(false)
    }
  }

  // 保存按钮是否禁用：提交中 / 编辑模式详情未就绪（加载中或加载失败）
  const saveDisabled = saving || (mode === 'edit' && (detailLoading || !!detailError))

  const fieldLabel: React.CSSProperties = {
    fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '6px', display: 'block',
  }
  const fieldInput: React.CSSProperties = {
    width: '100%', padding: '9px 12px', borderRadius: '8px', border: `1px solid ${C.border}`,
    fontSize: '14px', outline: 'none', boxSizing: 'border-box', background: C.white, color: C.text,
  }

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 1000, background: 'rgba(15,23,42,0.45)', backdropFilter: 'blur(4px)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '20px' }}
      onClick={onClose}>
      <div style={{ width: '460px', maxWidth: '100%', maxHeight: '90vh', overflowY: 'auto', background: C.white, borderRadius: '16px', boxShadow: '0 20px 60px rgba(0,0,0,0.2)', padding: '24px' }}
        onClick={e => e.stopPropagation()}>

        {/* 标题 */}
        <div style={{ marginBottom: '18px' }}>
          <div style={{ fontSize: '17px', fontWeight: 700, color: C.text }}>
            {mode === 'create' ? '👨‍🏫 新建教研组' : '✏️ 编辑教研组'}
          </div>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>
            所属学校：{schoolName}
          </div>
        </div>

        {/* 编辑模式：详情加载状态提示 */}
        {mode === 'edit' && detailLoading && (
          <div style={{ padding: '8px 12px', borderRadius: '8px', background: C.primaryLight, color: C.primary, fontSize: '12px', marginBottom: '14px' }}>
            ⏳ 正在加载教研组详情（含描述），加载完成前暂不可保存...
          </div>
        )}
        {mode === 'edit' && detailError && (
          <div style={{ padding: '8px 12px', borderRadius: '8px', background: C.dangerLight, color: C.danger, fontSize: '12px', marginBottom: '14px' }}>
            {detailError}（为防覆盖已有数据，保存已禁用）
          </div>
        )}

        {/* 名称 */}
        <div style={{ marginBottom: '14px' }}>
          <label style={fieldLabel}>教研组名称 <span style={{ color: C.danger }}>*</span></label>
          <input value={name} onChange={e => setName(e.target.value)} placeholder="例如：三年级数学组"
            style={fieldInput}
            onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
            onBlur={e => { e.currentTarget.style.borderColor = C.border }} />
        </div>

        {/* 学科 */}
        <div style={{ marginBottom: '14px' }}>
          <label style={fieldLabel}>学科 <span style={{ color: C.danger }}>*</span></label>
          <input value={subject} onChange={e => setSubject(e.target.value)} placeholder="例如：数学"
            style={fieldInput}
            onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
            onBlur={e => { e.currentTarget.style.borderColor = C.border }} />
        </div>

        {/* 年级范围 */}
        <div style={{ marginBottom: '14px' }}>
          <label style={fieldLabel}>年级范围（选填）</label>
          <input value={gradeRange} onChange={e => setGradeRange(e.target.value)} placeholder="例如：1-3年级"
            style={fieldInput}
            onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
            onBlur={e => { e.currentTarget.style.borderColor = C.border }} />
        </div>

        {/* 描述 */}
        <div style={{ marginBottom: '16px' }}>
          <label style={fieldLabel}>描述（选填）</label>
          <textarea value={desc} onChange={e => setDesc(e.target.value)} placeholder="教研组简介、职责说明等"
            rows={3}
            style={{ ...fieldInput, resize: 'vertical', fontFamily: 'inherit', lineHeight: 1.5 }}
            onFocus={e => { e.currentTarget.style.borderColor = C.primary }}
            onBlur={e => { e.currentTarget.style.borderColor = C.border }} />
        </div>

        {/* 提交错误提示 */}
        {error && (
          <div style={{ padding: '8px 12px', borderRadius: '8px', background: C.dangerLight, color: C.danger, fontSize: '12px', marginBottom: '14px' }}>
            {error}
          </div>
        )}

        {/* 底部按钮 */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
          <button onClick={onClose} disabled={saving}
            style={{ padding: '9px 20px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.bg, color: C.textSec, fontSize: '13px', fontWeight: 500, cursor: saving ? 'not-allowed' : 'pointer' }}>
            取消
          </button>
          <button onClick={handleSave} disabled={saveDisabled}
            style={{ padding: '9px 24px', borderRadius: '8px', border: 'none', background: saveDisabled ? C.border : `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: saveDisabled ? 'not-allowed' : 'pointer' }}>
            {saving ? '保存中...' : mode === 'edit' && detailLoading ? '加载详情中...' : '保存'}
          </button>
        </div>
      </div>
    </div>
  )
}
