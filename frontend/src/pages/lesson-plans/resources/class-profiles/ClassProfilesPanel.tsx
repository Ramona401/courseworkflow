import { useState, useEffect, useCallback } from 'react'
import { DEFAULT_SUBJECTS } from '@/constants/subjects'
import type { CSSProperties, ReactNode, MouseEvent as ReactMouseEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  getClassProfiles, getClassProfile, createClassProfile, updateClassProfile, deleteClassProfile,
} from '@/api/class-profiles'
import type {
  ClassProfileListItem, ClassProfileDetail,
} from '@/api/class-profiles'

/**
 * ClassProfilesPanel.tsx — 班级学情面板（差异化教学·老师私有资料），嵌于「我的备课资料」Tab
 *
 * 批次1 功能：
 *   - 班级学情卡列表（纯个人，每班一张卡）
 *   - 新建班级卡（手写入口：填学科/年级/班级/学期 + 四大段群体学情，四大段可留空慢慢补）
 *   - 编辑卡片、删除卡片
 *
 * 批次2a 新增：
 *   - 每张班级卡上加「👥 学生档案」按钮，跳转到学生个体档案独立全屏子页
 *     （/lesson-plans/resources/class-profiles/:id/students）。
 *
 * 三层数据结构里，本面板做第三层"班级卡"（群体结论，注入 AI）；点进学生档案是第一层
 * "学生个体档案"（本地明细，永不注入 AI）。AI 总结/成绩导入三个入口在后续批次接入。
 *
 * 合规提示：卡片四大段是匿名群体描述，将来唯一会注入 AI 的内容；学生个体明细不在这里。
 * 编辑弹窗内明确标注"以下内容会在备课时提供给 AI"。
 *
 * 视觉规范完全对齐 UnitPlansPanel（C 配色 / inputStyle / ModalShell / Toast / 徽章）。
 */

const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  border: '#E5E7EB', white: '#FFFFFF',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
}

const SUBJECTS = [...DEFAULT_SUBJECTS]  // 单一真相源（方案甲，v231）
const GRADES = ['一年级', '二年级', '三年级', '四年级', '五年级', '六年级', '七年级', '八年级', '九年级', '高一', '高二', '高三']

// 表单数据形状（新建/编辑共用）
interface CardForm {
  subject: string
  grade: string
  class_name: string
  term: string
  student_count: number
  overall_profile: string
  tier_structure: string
  weak_points: string
  teaching_advice: string
}

const emptyForm: CardForm = {
  subject: '语文', grade: '七年级', class_name: '', term: '',
  student_count: 0, overall_profile: '', tier_structure: '', weak_points: '', teaching_advice: '',
}

// ---------- 共享样式（对齐 UnitPlansPanel）----------
const inputStyle: CSSProperties = {
  width: '100%', padding: '8px 10px', border: '1px solid ' + C.border,
  borderRadius: 8, fontSize: 13, color: C.textPrimary, outline: 'none', boxSizing: 'border-box',
}
const labelStyle: CSSProperties = { fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 4, display: 'block' }
const btnPrimary: CSSProperties = { padding: '8px 16px', background: C.primary, color: '#fff', border: 'none', borderRadius: 8, fontSize: 13, fontWeight: 600, cursor: 'pointer' }
const btnGhost: CSSProperties = { padding: '8px 16px', background: C.white, color: C.textSecondary, border: '1px solid ' + C.border, borderRadius: 8, fontSize: 13, cursor: 'pointer' }

export default function ClassProfilesPanel() {
  const navigate = useNavigate()
  const [list, setList] = useState<ClassProfileListItem[]>([])
  const [loading, setLoading] = useState(true)

  // 新建/编辑弹窗：editId 为空=新建，非空=编辑
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<string>('')
  const [form, setForm] = useState<CardForm>(emptyForm)
  const [saving, setSaving] = useState(false)

  const [toast, setToast] = useState('')
  const showToast = (m: string) => { setToast(m); window.setTimeout(() => setToast(''), 2600) }

  const loadList = useCallback(async () => {
    setLoading(true)
    try { const r = await getClassProfiles(); setList(r.profiles || []) }
    catch (e: any) { showToast(e?.message || '加载失败') }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { loadList() }, [loadList])

  // 打开"新建"弹窗
  const openCreate = () => {
    setEditId('')
    setForm(emptyForm)
    setShowForm(true)
  }

  // 打开"编辑"弹窗（先拉详情回填四大段）
  const openEdit = async (item: ClassProfileListItem) => {
    try {
      const d: ClassProfileDetail = await getClassProfile(item.id)
      setEditId(d.id)
      setForm({
        subject: d.subject, grade: d.grade, class_name: d.class_name, term: d.term,
        student_count: d.student_count,
        overall_profile: d.overall_profile, tier_structure: d.tier_structure,
        weak_points: d.weak_points, teaching_advice: d.teaching_advice,
      })
      setShowForm(true)
    } catch (e: any) { showToast(e?.message || '打开失败') }
  }

  // 跳转到学生档案子页（批次2a）
  const goStudents = (item: ClassProfileListItem, e: ReactMouseEvent) => {
    e.stopPropagation() // 防冒泡触发卡片整体的"编辑"
    navigate('/lesson-plans/resources/class-profiles/' + item.id + '/students')
  }

  // 保存（新建 or 更新）
  const doSave = async () => {
    if (!form.subject.trim() || !form.grade.trim() || !form.class_name.trim()) {
      showToast('学科、年级、班级名为必填'); return
    }
    setSaving(true)
    try {
      if (editId) {
        await updateClassProfile(editId, {
          subject: form.subject, grade: form.grade, class_name: form.class_name, term: form.term,
          student_count: Number(form.student_count) || 0,
          overall_profile: form.overall_profile, tier_structure: form.tier_structure,
          weak_points: form.weak_points, teaching_advice: form.teaching_advice,
        })
        showToast('已保存')
      } else {
        await createClassProfile({
          subject: form.subject, grade: form.grade, class_name: form.class_name, term: form.term,
          student_count: Number(form.student_count) || 0,
          overall_profile: form.overall_profile, tier_structure: form.tier_structure,
          weak_points: form.weak_points, teaching_advice: form.teaching_advice,
        })
        showToast('已创建班级')
      }
      setShowForm(false); loadList()
    } catch (e: any) { showToast(e?.message || '保存失败') }
    finally { setSaving(false) }
  }

  const doDelete = async (item: ClassProfileListItem, e: ReactMouseEvent) => {
    e.stopPropagation()
    if (!window.confirm('确认删除「' + item.class_name + '」的学情卡？此操作不可恢复（学生明细也将一并隐藏）。')) return
    try { await deleteClassProfile(item.id); showToast('已删除'); loadList() }
    catch (err: any) { showToast(err?.message || '删除失败') }
  }

  const set = (k: keyof CardForm, v: string | number) => setForm((s) => ({ ...s, [k]: v }))

  return (
    <div>
      {/* 顶部说明 + 新建按钮 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
        <div style={{ fontSize: 13, color: C.textSecondary, maxWidth: 720 }}>
          为你带的每个班建一张学情卡，记录整体基础、分层结构与薄弱点。备课时挂载某个班，AI 就能据此做分层教学设计。
          <span style={{ color: C.textMuted }}>（卡片内容只在备课时提供给 AI，学生个体明细永远只保存在本地、不发送 AI。）</span>
        </div>
        <button onClick={openCreate} style={btnPrimary}>＋ 新建班级</button>
      </div>

      {/* 列表 */}
      {loading ? (
        <div style={{ padding: 40, textAlign: 'center', color: C.textMuted, fontSize: 13 }}>加载中…</div>
      ) : list.length === 0 ? (
        <div style={{ padding: 48, textAlign: 'center', background: C.white, borderRadius: 12, border: '1px dashed ' + C.border }}>
          <div style={{ fontSize: 30, marginBottom: 10 }}>🧑‍🎓</div>
          <div style={{ fontSize: 14, color: C.textSecondary }}>还没有班级学情卡，点右上角「新建班级」开始。</div>
        </div>
      ) : (
        <div style={{ display: 'grid', gap: 12 }}>
          {list.map((it) => (
            <div key={it.id} onClick={() => openEdit(it)} style={{ padding: '14px 16px', background: C.white, border: '1px solid ' + C.border, borderRadius: 12, cursor: 'pointer', transition: 'all 140ms ease' }}
              onMouseEnter={(e) => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = '0 2px 10px rgba(79,123,232,0.10)' }}
              onMouseLeave={(e) => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                <span style={{ fontSize: 11, padding: '2px 8px', borderRadius: 6, background: '#DBEAFE', color: '#2563EB', fontWeight: 600 }}>🏫 我的班级</span>
                {it.has_profile
                  ? <span style={{ fontSize: 11, padding: '2px 8px', borderRadius: 6, background: '#DCFCE7', color: '#16A34A', fontWeight: 600 }}>学情已填</span>
                  : <span style={{ fontSize: 11, padding: '2px 8px', borderRadius: 6, background: '#FEF3C7', color: '#B45309', fontWeight: 600 }}>待完善</span>}
                <div style={{ flex: 1 }} />
                {/* 批次2a：学生档案入口 */}
                <button onClick={(e) => goStudents(it, e)}
                  style={{ padding: '4px 10px', background: C.primaryLight, color: C.primary, border: '1px solid ' + C.border, borderRadius: 7, fontSize: 12, fontWeight: 600, cursor: 'pointer' }}>
                  👥 学生档案{it.student_count > 0 ? ' · ' + it.student_count : ''}
                </button>
                <button onClick={(e) => doDelete(it, e)} style={{ background: 'none', border: 'none', color: C.textMuted, cursor: 'pointer', fontSize: 13 }}>🗑</button>
              </div>
              <div style={{ fontSize: 15, fontWeight: 700, color: C.textPrimary, marginBottom: 4 }}>{it.class_name}</div>
              <div style={{ fontSize: 12.5, color: C.textSecondary }}>
                {it.subject} · {it.grade}{it.term ? ' · ' + it.term : ''}
                {it.student_count > 0 ? ' · ' + it.student_count + '人' : ''}
                {it.last_analyzed_at ? '　· 更新于 ' + formatDate(it.last_analyzed_at) : ''}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 新建/编辑弹窗 */}
      {showForm && (
        <CardFormModal
          editing={!!editId}
          form={form} set={set}
          saving={saving}
          onCancel={() => setShowForm(false)}
          onSave={doSave}
        />
      )}

      {toast && <Toast text={toast} />}
    </div>
  )
}

// ---------- 新建/编辑弹窗 ----------
function CardFormModal({
  editing, form, set, saving, onCancel, onSave,
}: {
  editing: boolean
  form: CardForm
  set: (k: keyof CardForm, v: string | number) => void
  saving: boolean
  onCancel: () => void
  onSave: () => void
}) {
  return (
    <ModalShell title={editing ? '编辑班级学情卡' : '新建班级'} wide onCancel={onCancel}>
      <div style={{ maxHeight: '66vh', overflowY: 'auto', paddingRight: 4 }}>
        {/* 定位字段 */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10, marginBottom: 12 }}>
          <div><label style={labelStyle}>班级名 *</label>
            <input value={form.class_name} onChange={(e) => set('class_name', e.target.value)} style={inputStyle} placeholder="如：初二3班 / 七(5)班" /></div>
          <div><label style={labelStyle}>学期（可选）</label>
            <input value={form.term} onChange={(e) => set('term', e.target.value)} style={inputStyle} placeholder="如：2025-2026 上学期" /></div>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10, marginBottom: 16 }}>
          <div><label style={labelStyle}>学科 *</label>
            <select value={form.subject} onChange={(e) => set('subject', e.target.value)} style={inputStyle}>{SUBJECTS.map((x) => <option key={x} value={x}>{x}</option>)}</select></div>
          <div><label style={labelStyle}>年级 *</label>
            <select value={form.grade} onChange={(e) => set('grade', e.target.value)} style={inputStyle}>{GRADES.map((x) => <option key={x} value={x}>{x}</option>)}</select></div>
          <div><label style={labelStyle}>人数（可选）</label>
            <input type="number" min={0} value={form.student_count || ''} onChange={(e) => set('student_count', e.target.value === '' ? 0 : Number(e.target.value))} style={inputStyle} placeholder="如：42" /></div>
        </div>

        {/* 注入 AI 的群体学情内容 */}
        <div style={{ background: C.primaryLight, border: '1px solid ' + C.border, borderRadius: 10, padding: '10px 12px', marginBottom: 12 }}>
          <div style={{ fontSize: 12.5, fontWeight: 700, color: C.primary, marginBottom: 2 }}>📋 班级学情（备课时会提供给 AI）</div>
          <div style={{ fontSize: 11.5, color: C.textSecondary, lineHeight: 1.6 }}>
            以下四项是<strong>整体群体描述</strong>，备课挂载该班时会作为背景提供给 AI 做分层设计。
            可先留空，以后用学生名单/成绩单让 AI 帮你总结（即将上线）。请勿在此填写学生姓名等个人信息。
          </div>
        </div>

        <div style={{ display: 'grid', gap: 12 }}>
          <div><label style={labelStyle}>整体画像（基础水平 / 学习风格 / 班风）</label>
            <textarea value={form.overall_profile} onChange={(e) => set('overall_profile', e.target.value)} rows={3} style={{ ...inputStyle, resize: 'vertical', fontFamily: 'inherit' }}
              placeholder="如：班级整体基础中等偏上，思维活跃但书写习惯较弱；课堂参与度高，小组合作意愿强。" /></div>
          <div><label style={labelStyle}>分层结构（A 拔尖 / B 中等 / C 学困，按占比与特征描述，匿名）</label>
            <textarea value={form.tier_structure} onChange={(e) => set('tier_structure', e.target.value)} rows={4} style={{ ...inputStyle, resize: 'vertical', fontFamily: 'inherit' }}
              placeholder={'如：\nA 层约 30%：基础扎实，可承担拓展挑战与领学任务；\nB 层约 50%：掌握主干知识，需巩固与适度提升；\nC 层约 20%：基础薄弱，需降低起点、补差与正向激励。'} /></div>
          <div><label style={labelStyle}>学科薄弱点</label>
            <textarea value={form.weak_points} onChange={(e) => set('weak_points', e.target.value)} rows={2} style={{ ...inputStyle, resize: 'vertical', fontFamily: 'inherit' }}
              placeholder="如：二次函数图像与性质理解普遍困难；阅读理解中信息整合能力两极分化。" /></div>
          <div><label style={labelStyle}>分层教学建议（可选）</label>
            <textarea value={form.teaching_advice} onChange={(e) => set('teaching_advice', e.target.value)} rows={2} style={{ ...inputStyle, resize: 'vertical', fontFamily: 'inherit' }}
              placeholder="如：导入用真实情境降低 C 层门槛；为 A 层准备开放性拓展任务；关键环节设置分层练习。" /></div>
        </div>
      </div>

      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 18 }}>
        <button onClick={onCancel} style={btnGhost}>取消</button>
        <button onClick={onSave} disabled={saving} style={{ ...btnPrimary, opacity: saving ? 0.6 : 1 }}>{saving ? '保存中…' : (editing ? '保存修改' : '创建班级')}</button>
      </div>
    </ModalShell>
  )
}

// ---------- 通用组件（对齐 UnitPlansPanel）----------
function ModalShell({ title, children, onCancel, wide }: { title: string; children: ReactNode; onCancel: () => void; wide?: boolean }) {
  return (
    <div onClick={onCancel} style={{ position: 'fixed', inset: 0, background: 'rgba(17,24,39,0.45)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 9000, padding: 20 }}>
      <div onClick={(e) => e.stopPropagation()} style={{ background: '#fff', borderRadius: 14, padding: 22, width: wide ? 720 : 460, maxWidth: '100%', boxShadow: '0 12px 48px rgba(0,0,0,0.18)' }}>
        <div style={{ fontSize: 16, fontWeight: 700, color: '#1F2937', marginBottom: 16 }}>{title}</div>
        {children}
      </div>
    </div>
  )
}

function Toast({ text }: { text: string }) {
  return (
    <div style={{ position: 'fixed', bottom: 28, left: '50%', transform: 'translateX(-50%)', background: 'rgba(17,24,39,0.92)', color: '#fff', padding: '10px 20px', borderRadius: 10, fontSize: 13, zIndex: 9999, boxShadow: '0 6px 24px rgba(0,0,0,0.2)' }}>{text}</div>
  )
}

// ---------- 纯函数辅助 ----------
function formatDate(iso: string): string {
  try {
    const d = new Date(iso)
    if (isNaN(d.getTime())) return ''
    return `${d.getMonth() + 1}月${d.getDate()}日`
  } catch { return '' }
}
