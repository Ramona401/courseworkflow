/**
 * SubjectsPanel.tsx — 后台「学科管理」面板（v231）
 *
 * 用途：
 *   学科字典的运营管理入口。原学科散落在前端 8+ 处硬编码、各副本不一致
 *   （备课下拉缺劳动/道法/美术/音乐/体育等），现改为数据库单一真相源，
 *   本面板即其管理界面——增删改后，全站学科下拉经 useSubjects 统一同步。
 *
 * 交互范式仿本目录 KBAuthorizedPanel：C 颜色常量 + 原生 button/input +
 *   Toast 提示 + ConfirmDialog 二次确认，无 antd。
 *
 * 后端对接（api/subjects.ts，路由 /api/v1/admin/subjects，仅 admin）：
 *   - getAllSubjects()               列出全部（含停用）
 *   - createSubject / updateSubject / deleteSubject
 *   增删改后调用 refreshSubjects() 同步全站下拉缓存。
 *
 * 说明：学科深层能力（AOCI 索引编码 code、课标库约束）仍「有就用没就降级」，
 *   新增学科能选能备课，暂无课标约束注入属预期行为，非 bug。
 *
 * 挂载：AdminPage 的「📚 学科管理」Tab（仅 admin 可见）。
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getAllSubjects, createSubject, updateSubject, deleteSubject,
  type SubjectItem,
} from '@/api/subjects'
import { refreshSubjects } from '@/hooks/useSubjects'
import { C } from './adminConstants'
import { Toast } from './adminShared'
import { ConfirmDialog } from './ConfirmDialog'

// 编辑/新增表单的字段结构
interface SubjectForm {
  name: string
  code: string
  sort_order: number
  note: string
  is_active: boolean
}

const emptyForm = (sortOrder: number): SubjectForm => ({
  name: '', code: '', sort_order: sortOrder, note: '', is_active: true,
})

export function SubjectsPanel() {
  const [list, setList] = useState<SubjectItem[]>([])
  const [loading, setLoading] = useState(true)
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)
  const showToast = useCallback((m: string, t: 'success' | 'error') => setToast({ message: m, type: t }), [])

  // 编辑弹窗：editing=null 且 modalOpen=true → 新增；editing 有值 → 编辑
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<SubjectItem | null>(null)
  const [form, setForm] = useState<SubjectForm>(emptyForm(100))
  const [saving, setSaving] = useState(false)

  // 删除二次确认
  const [confirmDel, setConfirmDel] = useState<{ open: boolean; id: string; name: string }>({
    open: false, id: '', name: '',
  })

  // ==================== 加载列表 ====================
  const load = useCallback(async () => {
    try {
      setLoading(true)
      const items = await getAllSubjects()
      setList(items)
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '加载学科列表失败', 'error')
    } finally {
      setLoading(false)
    }
  }, [showToast])

  useEffect(() => { load() }, [load])

  // ==================== 打开新增/编辑 ====================
  const openCreate = () => {
    const maxSort = list.reduce((m, s) => Math.max(m, s.sort_order), 0)
    setEditing(null)
    setForm(emptyForm(maxSort + 10))
    setModalOpen(true)
  }

  const openEdit = (row: SubjectItem) => {
    setEditing(row)
    setForm({
      name: row.name, code: row.code || '',
      sort_order: row.sort_order, note: row.note || '',
      is_active: row.is_active,
    })
    setModalOpen(true)
  }

  // ==================== 保存（新增/编辑）====================
  const handleSave = async () => {
    const name = form.name.trim()
    if (!name) { showToast('请输入学科名', 'error'); return }
    try {
      setSaving(true)
      if (editing) {
        await updateSubject(editing.id, {
          name, code: form.code.trim(),
          sort_order: form.sort_order, note: form.note.trim(),
          is_active: form.is_active,
        })
        showToast('已保存', 'success')
      } else {
        await createSubject({
          name, code: form.code.trim(),
          sort_order: form.sort_order, note: form.note.trim(),
        })
        showToast('已新增', 'success')
      }
      setModalOpen(false)
      await load()
      await refreshSubjects() // 同步全站下拉
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '保存失败', 'error')
    } finally {
      setSaving(false)
    }
  }

  // ==================== 行内启停 ====================
  const handleToggleActive = async (row: SubjectItem) => {
    try {
      await updateSubject(row.id, { is_active: !row.is_active })
      showToast(!row.is_active ? '已启用' : '已停用', 'success')
      await load()
      await refreshSubjects()
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '操作失败', 'error')
      await load()
    }
  }

  // ==================== 删除 ====================
  const doDelete = async (id: string) => {
    try {
      await deleteSubject(id)
      showToast('已删除', 'success')
      await load()
      await refreshSubjects()
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '删除失败', 'error')
    } finally {
      setConfirmDel({ open: false, id: '', name: '' })
    }
  }

  // 输入框统一样式
  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '9px 12px', borderRadius: '8px',
    border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none',
    boxSizing: 'border-box', background: C.white, color: C.text,
  }
  const labelStyle: React.CSSProperties = {
    fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '5px', display: 'block',
  }

  return (
    <div>
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      {/* 删除二次确认 */}
      {confirmDel.open && (
        <ConfirmDialog
          title="删除学科"
          message={`确认删除学科「${confirmDel.name}」？删除后不影响已有教案/课件等历史数据，但该学科将从所有下拉中移除。`}
          onConfirm={() => doDelete(confirmDel.id)}
          onCancel={() => setConfirmDel({ open: false, id: '', name: '' })}
        />
      )}

      {/* 新增/编辑弹窗 */}
      {modalOpen && (
        <div style={{
          position: 'fixed', inset: 0, zIndex: 11000,
          background: 'rgba(0,0,0,0.45)', backdropFilter: 'blur(4px)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <div style={{ background: C.white, borderRadius: '16px', width: '420px', padding: '28px', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
            <div style={{ fontSize: '16px', fontWeight: 700, color: C.text, marginBottom: '18px' }}>
              {editing ? '编辑学科' : '新增学科'}
            </div>

            <div style={{ marginBottom: '14px' }}>
              <label style={labelStyle}>学科名 <span style={{ color: C.danger }}>*</span></label>
              <input value={form.name} maxLength={50}
                onChange={e => setForm(p => ({ ...p, name: e.target.value }))}
                placeholder="如：信息科技" style={inputStyle} />
            </div>

            <div style={{ marginBottom: '14px' }}>
              <label style={labelStyle}>索引编码（可选）</label>
              <input value={form.code} maxLength={20}
                onChange={e => setForm(p => ({ ...p, code: e.target.value }))}
                placeholder="对应 AOCI 索引编码，可留空" style={inputStyle} />
            </div>

            <div style={{ marginBottom: '14px' }}>
              <label style={labelStyle}>排序（数值越小越靠前）</label>
              <input type="number" value={form.sort_order} min={1} max={9999}
                onChange={e => setForm(p => ({ ...p, sort_order: Number(e.target.value) || 0 }))}
                style={inputStyle} />
            </div>

            <div style={{ marginBottom: '14px' }}>
              <label style={labelStyle}>备注（可选）</label>
              <input value={form.note} maxLength={200}
                onChange={e => setForm(p => ({ ...p, note: e.target.value }))}
                placeholder="可选" style={inputStyle} />
            </div>

            {/* 编辑态才显示启停开关（新增默认启用）*/}
            {editing && (
              <div style={{ marginBottom: '18px', display: 'flex', alignItems: 'center', gap: '10px' }}>
                <label style={{ ...labelStyle, marginBottom: 0 }}>状态</label>
                <button
                  onClick={() => setForm(p => ({ ...p, is_active: !p.is_active }))}
                  style={{
                    padding: '5px 14px', borderRadius: '20px', border: 'none', cursor: 'pointer',
                    fontSize: '12px', fontWeight: 600,
                    background: form.is_active ? C.successLight : C.bg,
                    color: form.is_active ? C.success : C.textMuted,
                  }}>
                  {form.is_active ? '● 启用中' : '○ 已停用'}
                </button>
              </div>
            )}

            <div style={{ display: 'flex', gap: '10px' }}>
              <button onClick={() => setModalOpen(false)} disabled={saving}
                style={{ flex: 1, padding: '10px', borderRadius: '10px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>
                取消
              </button>
              <button onClick={handleSave} disabled={saving}
                style={{ flex: 1, padding: '10px', borderRadius: '10px', border: 'none', background: saving ? '#9CA3AF' : `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer' }}>
                {saving ? '保存中...' : '保存'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 标题栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '16px' }}>
        <div>
          <div style={{ fontSize: '16px', fontWeight: 700, color: C.text, display: 'flex', alignItems: 'center', gap: '8px' }}>
            📚 学科管理
            {list.length > 0 && (
              <span style={{ fontSize: '12px', fontWeight: 400, color: C.textMuted }}>共 {list.length} 个学科</span>
            )}
          </div>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px', lineHeight: 1.6, maxWidth: '640px' }}>
            全平台学科下拉的统一数据源。此处增删改后，备课工坊 / 课件工坊 / 配方 / AI助手等所有学科下拉即时同步。
            内置学科不可删除（可停用）；停用后仅从下拉隐藏，不影响历史数据。
          </div>
        </div>
        <button onClick={openCreate}
          style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: `linear-gradient(135deg,${C.primary},#7C3AED)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap' }}>
          + 新增学科
        </button>
      </div>

      {/* 列表 */}
      <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
        {/* 表头 */}
        <div style={{ display: 'grid', gridTemplateColumns: '80px 2fr 1fr 1fr 2fr 200px', padding: '12px 20px', background: C.bg, borderBottom: `1px solid ${C.border}`, fontSize: '12px', fontWeight: 600, color: C.textSec }}>
          <span>排序</span><span>学科名</span><span>编码</span><span>状态</span><span>备注</span><span>操作</span>
        </div>

        {loading ? (
          <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div>
        ) : list.length === 0 ? (
          <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>暂无学科，点击右上角「新增学科」添加</div>
        ) : (
          list.map((s, idx) => (
            <div key={s.id}
              style={{ display: 'grid', gridTemplateColumns: '80px 2fr 1fr 1fr 2fr 200px', padding: '13px 20px', alignItems: 'center', borderBottom: idx < list.length - 1 ? `1px solid ${C.border}` : 'none', fontSize: '13px' }}>
              {/* 排序 */}
              <span style={{ color: C.textMuted, fontFamily: 'monospace' }}>{s.sort_order}</span>
              {/* 学科名 + 内置标 */}
              <span style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span style={{ fontWeight: 600, color: C.text }}>{s.name}</span>
                {s.is_system && (
                  <span style={{ fontSize: '10px', padding: '1px 7px', borderRadius: '8px', background: C.primaryLight, color: C.primary, fontWeight: 600 }}>内置</span>
                )}
              </span>
              {/* 编码 */}
              <span>
                {s.code
                  ? <span style={{ fontSize: '11px', padding: '1px 8px', borderRadius: '6px', background: C.bg, color: C.textSec, border: `1px solid ${C.border}`, fontFamily: 'monospace' }}>{s.code}</span>
                  : <span style={{ color: C.textMuted }}>—</span>}
              </span>
              {/* 状态（点击切换）*/}
              <span>
                <button onClick={() => handleToggleActive(s)}
                  style={{
                    padding: '3px 12px', borderRadius: '20px', border: 'none', cursor: 'pointer',
                    fontSize: '11px', fontWeight: 600,
                    background: s.is_active ? C.successLight : C.bg,
                    color: s.is_active ? C.success : C.textMuted,
                  }}
                  title={s.is_active ? '点击停用' : '点击启用'}>
                  {s.is_active ? '● 启用' : '○ 停用'}
                </button>
              </span>
              {/* 备注 */}
              <span style={{ color: s.note ? C.textSec : C.textMuted, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {s.note || '—'}
              </span>
              {/* 操作 */}
              <span style={{ display: 'flex', gap: '6px' }}>
                <button onClick={() => openEdit(s)}
                  style={{ padding: '4px 12px', borderRadius: '6px', border: `1px solid ${C.border}`, background: C.bg, color: C.primary, fontSize: '12px', cursor: 'pointer', fontWeight: 500 }}>
                  编辑
                </button>
                {s.is_system ? (
                  <button disabled title="内置学科不可删除，如需隐藏请停用"
                    style={{ padding: '4px 12px', borderRadius: '6px', border: `1px solid ${C.border}`, background: C.bg, color: C.textMuted, fontSize: '12px', cursor: 'not-allowed', fontWeight: 500 }}>
                    删除
                  </button>
                ) : (
                  <button onClick={() => setConfirmDel({ open: true, id: s.id, name: s.name })}
                    style={{ padding: '4px 12px', borderRadius: '6px', border: '1px solid #FEE2E2', background: '#FEF2F2', color: C.danger, fontSize: '12px', cursor: 'pointer', fontWeight: 500 }}>
                    删除
                  </button>
                )}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
