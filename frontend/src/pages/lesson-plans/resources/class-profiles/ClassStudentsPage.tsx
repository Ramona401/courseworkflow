import { useState, useEffect, useCallback } from 'react'
import type { CSSProperties, ReactNode } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  getClassProfile, updateClassProfile,
  getClassStudents, createClassStudent, updateClassStudent, deleteClassStudent,
  summarizeClassProfile, autoTierStudents,
} from '@/api/class-profiles'
import type {
  ClassProfileDetail, ClassStudentView, StudentTier, ClassSummaryResult, AutoTierResult,
} from '@/api/class-profiles'
import ScoreImportModal from './ScoreImportModal'

/**
 * ClassStudentsPage.tsx — 学生个体档案独立全屏子页（差异化教学·班级学情 批次2a + 2b + 2c）
 *
 * 路由：/lesson-plans/resources/class-profiles/:id/students
 *   从「我的备课资料 → 班级学情」面板的班级卡「👥 学生档案」按钮进入。
 *
 * 三层数据结构里，本页是第一层"学生个体档案（原料库）"的录入界面。
 *
 * ⚠ 合规红线（界面化落地）：
 *   - 顶部常驻红色提示带：本页学生信息仅保存在本班级档案内，绝不发送给 AI；请用学号代号，勿填真名。
 *   - 学生明细（含学号代号）永不注入 AI；注入 AI 的只有班级卡的匿名群体结论。
 *
 * 批次2c（AI 总结学情）：
 *   - 顶部「🧠 让 AI 总结学情」按钮：调后端 summarize 端点。后端把学生明细就地脱敏聚合成
 *     匿名统计量喂 AI，返回班级卡四大段（只生成不落库）。
 *   - 弹预览窗：展示样本量 + 默认展开的「AI 看到的数据」(脱敏统计量原文，透明可信) + 四大段。
 *   - 老师点「采用并写回班级卡」才调 updateClassProfile 落库（决策2-B）；取消则丢弃。
 *   - 决策3-B：学生 <5 人时软提示"样本偏少仅供参考"，但不硬拦（仍可继续）。
 *
 * 手动录入只填"定性判断"：学号代号 / ABC 分层 / 薄弱点 / 备注。
 * 成绩为只读列——来自批次2b 的「成绩单导入」。
 *
 * 视觉范式完全对齐 ClassProfilesPanel / UnitPlansPanel（同一套 C 配色 / inputStyle / ModalShell / Toast）。
 *
 * ⚠ 返回目标（批次2a 修复）：「← 返回备课资料」携带 ?tab=class-situation 精确落回「班级学情」Tab。
 */

const BACK_TO_CLASS_SITUATION = '/lesson-plans/resources?tab=class-situation'

const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  border: '#E5E7EB', white: '#FFFFFF', bg: '#FAFBFC',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
  danger: '#EF4444', dangerLight: '#FEF2F2', dangerBorder: '#FECACA',
  success: '#16A34A',
  // 批次2c：AI 总结按钮/弹窗用紫色系，与导入(绿)、添加(蓝)区分
  ai: '#7C3AED', aiLight: 'rgba(124,58,237,0.08)', aiBorder: '#DDD6FE',
}

// ABC 分层徽章配色
const TIER_BADGE: Record<string, { t: string; bg: string; c: string }> = {
  A: { t: 'A 拔尖', bg: '#DCFCE7', c: '#16A34A' },
  B: { t: 'B 中等', bg: '#DBEAFE', c: '#2563EB' },
  C: { t: 'C 学困', bg: '#FEF3C7', c: '#B45309' },
  '': { t: '未分层', bg: '#F3F4F6', c: '#6B7280' },
}

interface StudentForm {
  student_code: string
  tier: StudentTier
  weak_topics: string
  note: string
}

const emptyForm: StudentForm = { student_code: '', tier: '', weak_topics: '', note: '' }

// 决策3-B：学生数低于此阈值时软提示"样本偏少"
const SMALL_SAMPLE_THRESHOLD = 5

// ---------- 共享样式 ----------
const inputStyle: CSSProperties = {
  width: '100%', padding: '8px 10px', border: '1px solid ' + C.border,
  borderRadius: 8, fontSize: 13, color: C.textPrimary, outline: 'none', boxSizing: 'border-box',
}
const labelStyle: CSSProperties = { fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 4, display: 'block' }
const btnPrimary: CSSProperties = { padding: '8px 16px', background: C.primary, color: '#fff', border: 'none', borderRadius: 8, fontSize: 13, fontWeight: 600, cursor: 'pointer' }
const btnGhost: CSSProperties = { padding: '8px 16px', background: C.white, color: C.textSecondary, border: '1px solid ' + C.border, borderRadius: 8, fontSize: 13, cursor: 'pointer' }
const btnImport: CSSProperties = { padding: '8px 16px', background: C.white, color: C.success, border: '1px solid ' + C.success, borderRadius: 8, fontSize: 13, fontWeight: 600, cursor: 'pointer' }
// 批次2c：AI 总结按钮（紫色边框）
const btnAI: CSSProperties = { padding: '8px 16px', background: C.aiLight, color: C.ai, border: '1px solid ' + C.ai, borderRadius: 8, fontSize: 13, fontWeight: 600, cursor: 'pointer' }
// 批次2d：按分数线分层按钮（青色边框，与其它操作区分）
const btnTier: CSSProperties = { padding: '8px 16px', background: C.white, color: '#0891B2', border: '1px solid #0891B2', borderRadius: 8, fontSize: 13, fontWeight: 600, cursor: 'pointer' }

export default function ClassStudentsPage() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [profile, setProfile] = useState<ClassProfileDetail | null>(null)
  const [students, setStudents] = useState<ClassStudentView[]>([])
  const [loading, setLoading] = useState(true)
  const [loadErr, setLoadErr] = useState('')

  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<string>('')
  const [form, setForm] = useState<StudentForm>(emptyForm)
  const [saving, setSaving] = useState(false)

  const [showImport, setShowImport] = useState(false)

  // 批次2c：AI 总结状态
  const [summarizing, setSummarizing] = useState(false)             // 调 AI 中（按钮 loading）
  const [summaryResult, setSummaryResult] = useState<ClassSummaryResult | null>(null) // 非空=显预览窗
  const [adopting, setAdopting] = useState(false)                   // 采用写回中

  // 批次2d：按分数线自动分层弹窗
  const [showAutoTier, setShowAutoTier] = useState(false)

  const [toast, setToast] = useState('')
  const showToast = (m: string) => { setToast(m); window.setTimeout(() => setToast(''), 2600) }

  const loadAll = useCallback(async () => {
    if (!id) return
    setLoading(true); setLoadErr('')
    try {
      const [p, s] = await Promise.all([getClassProfile(id), getClassStudents(id)])
      setProfile(p)
      setStudents(s.students || [])
    } catch (e: any) {
      setLoadErr(e?.message || '加载失败（可能该班级不存在或无权访问）')
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => { loadAll() }, [loadAll])

  const set = (k: keyof StudentForm, v: string) => setForm((s) => ({ ...s, [k]: v }))

  const openCreate = () => { setEditId(''); setForm(emptyForm); setShowForm(true) }

  const openEdit = (st: ClassStudentView) => {
    setEditId(st.id)
    setForm({ student_code: st.student_code, tier: st.tier, weak_topics: st.weak_topics, note: st.note })
    setShowForm(true)
  }

  const doSave = async () => {
    if (editId && !form.student_code.trim()) { showToast('编辑时学号代号不能为空'); return }
    setSaving(true)
    try {
      if (editId) {
        await updateClassStudent(id, editId, {
          student_code: form.student_code.trim(),
          tier: form.tier,
          weak_topics: form.weak_topics.trim(),
          note: form.note.trim(),
        })
        showToast('已保存')
      } else {
        await createClassStudent(id, {
          student_code: form.student_code.trim() || undefined,
          tier: form.tier,
          weak_topics: form.weak_topics.trim(),
          note: form.note.trim(),
        })
        showToast('已添加学生')
      }
      setShowForm(false)
      loadAll()
    } catch (e: any) {
      showToast(e?.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const doDelete = async (st: ClassStudentView) => {
    if (!window.confirm('确认删除学号「' + st.student_code + '」的档案？此操作不可恢复。')) return
    try { await deleteClassStudent(id, st.id); showToast('已删除'); loadAll() }
    catch (e: any) { showToast(e?.message || '删除失败') }
  }

  // ========== 批次2c：让 AI 总结学情 ==========
  // 决策3-B：<5 人软提示但不硬拦；调后端 summarize（只生成不落库），成功则弹预览窗
  const doSummarize = async () => {
    if (students.length === 0) { showToast('还没有学生数据，请先录入或导入成绩单'); return }
    if (students.length < SMALL_SAMPLE_THRESHOLD) {
      const ok = window.confirm(
        '当前仅 ' + students.length + ' 名学生，样本偏少，AI 总结结论仅供参考。\n' +
        '建议补充更多学生数据后再分析。是否仍要继续？'
      )
      if (!ok) return
    }
    setSummarizing(true)
    try {
      const res = await summarizeClassProfile(id)
      setSummaryResult(res) // 非空触发预览窗
    } catch (e: any) {
      showToast(e?.message || 'AI 总结失败，请稍后重试')
    } finally {
      setSummarizing(false)
    }
  }

  // 采用：把 AI 四大段写回班级卡（决策2-B：采用才落库）
  // 沿用 profile 现有定位字段（学科/年级/班名/学期/人数），四大段用 AI 结果覆盖
  const doAdoptSummary = async () => {
    if (!summaryResult || !profile) return
    setAdopting(true)
    try {
      await updateClassProfile(id, {
        subject: profile.subject,
        grade: profile.grade,
        class_name: profile.class_name,
        term: profile.term,
        student_count: profile.student_count,
        overall_profile: summaryResult.overall_profile,
        tier_structure: summaryResult.tier_structure,
        weak_points: summaryResult.weak_points,
        teaching_advice: summaryResult.teaching_advice,
      })
      setSummaryResult(null)        // 关预览窗
      showToast('已更新班级学情卡')  // 决策：留在本页弹 toast，不跳转
      // 重新拉 profile（last_analyzed_from 已变 ai_summary，保持数据新鲜）
      try { const p = await getClassProfile(id); setProfile(p) } catch { /* 刷新失败不影响主流程 */ }
    } catch (e: any) {
      showToast(e?.message || '写回失败，请重试')
    } finally {
      setAdopting(false)
    }
  }

  const renderScore = (st: ClassStudentView): ReactNode => {
    if (st.latest_score === null || st.latest_score === undefined) {
      return <span style={{ color: C.textMuted }}>—</span>
    }
    return <span style={{ color: C.textPrimary, fontWeight: 600 }}>{st.latest_score}</span>
  }

  return (
    <div style={{ minHeight: '100vh', background: C.bg, padding: '24px 28px' }}>
      <div style={{ maxWidth: 1000, margin: '0 auto' }}>

        {/* 返回 + 班级信息头 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
          <button onClick={() => navigate(BACK_TO_CLASS_SITUATION)} style={btnGhost}>← 返回备课资料</button>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 18, fontWeight: 700, color: C.textPrimary }}>
              👥 {profile ? profile.class_name : '学生档案'}
            </div>
            {profile && (
              <div style={{ fontSize: 12.5, color: C.textSecondary, marginTop: 2 }}>
                {profile.subject} · {profile.grade}{profile.term ? ' · ' + profile.term : ''} · 共 {students.length} 名学生
              </div>
            )}
          </div>
          {profile && (
            <div style={{ display: 'flex', gap: 8 }}>
              {/* 批次2c：AI 总结入口 */}
              <button onClick={doSummarize} disabled={summarizing} style={{ ...btnAI, opacity: summarizing ? 0.6 : 1, cursor: summarizing ? 'wait' : 'pointer' }}>
                {summarizing ? '🧠 AI 分析中…' : '🧠 让 AI 总结学情'}
              </button>
              {/* 批次2d：按分数线自动分层入口 */}
              <button onClick={() => setShowAutoTier(true)} style={btnTier}>🪜 按分数线分层</button>
              <button onClick={() => setShowImport(true)} style={btnImport}>📥 导入成绩单</button>
              <button onClick={openCreate} style={btnPrimary}>＋ 添加学生</button>
            </div>
          )}
        </div>

        {/* 合规提示带（红色常驻）*/}
        <div style={{
          display: 'flex', alignItems: 'flex-start', gap: 10,
          background: C.dangerLight, border: '1px solid ' + C.dangerBorder, borderRadius: 10,
          padding: '12px 14px', marginBottom: 18,
        }}>
          <span style={{ fontSize: 16, lineHeight: 1.4 }}>🔒</span>
          <div style={{ fontSize: 12.5, color: '#991B1B', lineHeight: 1.7 }}>
            <strong>隐私红线：</strong>本页学生信息<strong>仅保存在本班级档案内，绝不会发送给 AI</strong>。
            请使用<strong>学号代号</strong>（如 01 / S001），<strong>不要填写学生真实姓名</strong>。
            「让 AI 总结学情」时，AI 也只会拿到全班的匿名统计数字，看不到任何一个学生的明细。
          </div>
        </div>

        {/* 主体 */}
        {loading ? (
          <div style={{ padding: 48, textAlign: 'center', color: C.textMuted, fontSize: 13 }}>加载中…</div>
        ) : loadErr ? (
          <div style={{ padding: 40, textAlign: 'center', background: C.white, borderRadius: 12, border: '1px solid ' + C.dangerBorder }}>
            <div style={{ fontSize: 28, marginBottom: 10 }}>😕</div>
            <div style={{ fontSize: 14, color: C.danger, marginBottom: 14 }}>{loadErr}</div>
            <button onClick={() => navigate(BACK_TO_CLASS_SITUATION)} style={btnGhost}>返回备课资料</button>
          </div>
        ) : students.length === 0 ? (
          <div style={{ padding: 48, textAlign: 'center', background: C.white, borderRadius: 12, border: '1px dashed ' + C.border }}>
            <div style={{ fontSize: 30, marginBottom: 10 }}>🧑‍🎓</div>
            <div style={{ fontSize: 14, color: C.textSecondary, marginBottom: 6 }}>还没有学生档案。</div>
            <div style={{ fontSize: 12.5, color: C.textMuted }}>点右上角「添加学生」逐个录入，或用「📥 导入成绩单」批量建立（学号不存在会自动建档）。</div>
          </div>
        ) : (
          <div style={{ background: C.white, borderRadius: 12, border: '1px solid ' + C.border, overflow: 'hidden' }}>
            <div style={{
              display: 'grid', gridTemplateColumns: '120px 90px 80px 1fr 120px',
              gap: 12, padding: '10px 16px', background: '#F9FAFB',
              fontSize: 12, fontWeight: 600, color: C.textSecondary, borderBottom: '1px solid ' + C.border,
            }}>
              <div>学号代号</div>
              <div>分层</div>
              <div>最新成绩</div>
              <div>薄弱知识点 / 备注</div>
              <div style={{ textAlign: 'right' }}>操作</div>
            </div>
            {students.map((st) => {
              const tb = TIER_BADGE[st.tier] || TIER_BADGE['']
              return (
                <div key={st.id} style={{
                  display: 'grid', gridTemplateColumns: '120px 90px 80px 1fr 120px',
                  gap: 12, padding: '12px 16px', alignItems: 'center',
                  borderBottom: '1px solid ' + C.border, fontSize: 13,
                }}>
                  <div style={{ fontWeight: 700, color: C.textPrimary }}>{st.student_code}</div>
                  <div>
                    <span style={{ fontSize: 11, padding: '2px 8px', borderRadius: 6, background: tb.bg, color: tb.c, fontWeight: 600 }}>{tb.t}</span>
                  </div>
                  <div>{renderScore(st)}</div>
                  <div style={{ color: C.textSecondary, lineHeight: 1.6 }}>
                    {st.weak_topics
                      ? <span>{st.weak_topics}</span>
                      : <span style={{ color: C.textMuted }}>—</span>}
                    {st.note && <span style={{ color: C.textMuted }}>　📝 {st.note}</span>}
                  </div>
                  <div style={{ textAlign: 'right' }}>
                    <button onClick={() => openEdit(st)} style={{ ...btnGhost, padding: '4px 10px', marginRight: 6 }}>编辑</button>
                    <button onClick={() => doDelete(st)} style={{ background: 'none', border: 'none', color: C.textMuted, cursor: 'pointer', fontSize: 13 }}>🗑</button>
                  </div>
                </div>
              )
            })}
          </div>
        )}

      </div>

      {/* 新建/编辑弹窗 */}
      {showForm && (
        <StudentFormModal
          editing={!!editId}
          form={form} set={set}
          saving={saving}
          onCancel={() => setShowForm(false)}
          onSave={doSave}
        />
      )}

      {/* 成绩单导入弹窗（批次2b；2c 起带导入后 AI 总结快捷入口）*/}
      {showImport && profile && (
        <ScoreImportModal
          classProfileId={id}
          className={profile.class_name}
          onClose={() => setShowImport(false)}
          onImported={() => { loadAll() }}
          onRequestSummary={async () => {
            // 2c 决策4 第二入口：导入弹窗已自行关闭，这里先刷新列表再触发 AI 总结。
            // summarizeClassProfile 后端实时拉 DB 最新学生数据，不依赖前端 state，
            // 故刷新与总结的时序无实质风险（前端 state 仅影响 <5 人软提示判断）。
            await loadAll()
            doSummarize()
          }}
        />
      )}

      {/* AI 总结预览弹窗（批次2c）*/}
      {summaryResult && (
        <ClassSummaryModal
          result={summaryResult}
          adopting={adopting}
          onAdopt={doAdoptSummary}
          onCancel={() => setSummaryResult(null)}
        />
      )}

      {/* 按分数线自动分层弹窗（批次2d）*/}
      {showAutoTier && (
        <AutoTierModal
          students={students}
          onClose={() => setShowAutoTier(false)}
          onDone={(res: AutoTierResult) => {
            setShowAutoTier(false)
            showToast(`已分层：A ${res.tier_a} 人 / B ${res.tier_b} 人 / C ${res.tier_c} 人` + (res.skipped_no_score > 0 ? `（${res.skipped_no_score} 人无成绩未参与）` : ''))
            loadAll()
          }}
          doAutoTier={async (aLine: number, cLine: number) => autoTierStudents(id, { a_line: aLine, c_line: cLine })}
        />
      )}

      {toast && <Toast text={toast} />}
    </div>
  )
}

// ---------- 新建/编辑弹窗 ----------
function StudentFormModal({
  editing, form, set, saving, onCancel, onSave,
}: {
  editing: boolean
  form: StudentForm
  set: (k: keyof StudentForm, v: string) => void
  saving: boolean
  onCancel: () => void
  onSave: () => void
}) {
  const tiers: { v: StudentTier; label: string }[] = [
    { v: '', label: '未分层' },
    { v: 'A', label: 'A 拔尖' },
    { v: 'B', label: 'B 中等' },
    { v: 'C', label: 'C 学困' },
  ]
  return (
    <ModalShell title={editing ? '编辑学生档案' : '添加学生'} onCancel={onCancel}>
      <div style={{ display: 'grid', gap: 12 }}>
        <div>
          <label style={labelStyle}>学号代号 {editing ? '*' : '（留空将自动编号 01、02…）'}</label>
          <input value={form.student_code} onChange={(e) => set('student_code', e.target.value)} style={inputStyle}
            placeholder={editing ? '' : '可留空自动编号，或手填如 S001'} />
          <div style={{ fontSize: 11, color: '#991B1B', marginTop: 4 }}>⚠ 只填学号代号，请勿填写学生真实姓名。</div>
        </div>

        <div>
          <label style={labelStyle}>分层（老师判定，可留空）</label>
          <div style={{ display: 'flex', gap: 8 }}>
            {tiers.map((t) => {
              const active = form.tier === t.v
              return (
                <button key={t.v || 'none'} onClick={() => set('tier', t.v)}
                  style={{
                    flex: 1, padding: '8px 0', borderRadius: 8, fontSize: 13, cursor: 'pointer',
                    border: '1px solid ' + (active ? C.primary : C.border),
                    background: active ? C.primaryLight : C.white,
                    color: active ? C.primary : C.textSecondary,
                    fontWeight: active ? 700 : 400,
                  }}>{t.label}</button>
              )
            })}
          </div>
        </div>

        <div>
          <label style={labelStyle}>薄弱知识点（可选）</label>
          <textarea value={form.weak_topics} onChange={(e) => set('weak_topics', e.target.value)} rows={2}
            style={{ ...inputStyle, resize: 'vertical', fontFamily: 'inherit' }}
            placeholder="如：函数应用、几何证明、阅读理解信息整合" />
        </div>

        <div>
          <label style={labelStyle}>备注（可选，老师私人记录）</label>
          <textarea value={form.note} onChange={(e) => set('note', e.target.value)} rows={2}
            style={{ ...inputStyle, resize: 'vertical', fontFamily: 'inherit' }}
            placeholder="如：需多关注、进步明显、课堂积极" />
        </div>

        <div style={{ fontSize: 11.5, color: C.textMuted, lineHeight: 1.6 }}>
          💡 成绩不在此处手动录入——用顶部「📥 导入成绩单」批量录入历次成绩，系统会自动归并到对应学号。
        </div>
      </div>

      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 18 }}>
        <button onClick={onCancel} style={btnGhost}>取消</button>
        <button onClick={onSave} disabled={saving} style={{ ...btnPrimary, opacity: saving ? 0.6 : 1 }}>
          {saving ? '保存中…' : (editing ? '保存修改' : '添加')}
        </button>
      </div>
    </ModalShell>
  )
}

// ---------- AI 总结预览弹窗（批次2c）----------
function ClassSummaryModal({
  result, adopting, onAdopt, onCancel,
}: {
  result: ClassSummaryResult
  adopting: boolean
  onAdopt: () => void
  onCancel: () => void
}) {
  // 决策：脱敏统计量默认展开（一眼看到依据，更透明）
  const [statsOpen, setStatsOpen] = useState(true)
  const small = result.student_count < SMALL_SAMPLE_THRESHOLD

  // 四大段配置（标题 + 取值）
  const sections: { title: string; icon: string; text: string }[] = [
    { title: '整体画像', icon: '🪞', text: result.overall_profile },
    { title: '分层结构', icon: '🪜', text: result.tier_structure },
    { title: '学科薄弱点', icon: '🎯', text: result.weak_points },
    { title: '分层教学建议', icon: '💡', text: result.teaching_advice },
  ]

  return (
    <div onClick={() => { if (!adopting) onCancel() }}
      style={{ position: 'fixed', inset: 0, background: 'rgba(17,24,39,0.45)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 9000, padding: 20 }}>
      <div onClick={(e) => e.stopPropagation()}
        style={{ background: '#fff', borderRadius: 16, width: 720, maxWidth: '95vw', maxHeight: '90vh', overflow: 'hidden', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>

        {/* 头部 */}
        <div style={{ padding: '18px 22px', borderBottom: '1px solid ' + C.border, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontSize: 16, fontWeight: 700, color: C.ai }}>🧠 AI 总结的班级学情</div>
          <button onClick={() => { if (!adopting) onCancel() }} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: 20, color: C.textMuted }}>×</button>
        </div>

        {/* 内容（可滚动）*/}
        <div style={{ padding: 22, overflowY: 'auto', flex: 1 }}>

          {/* 样本量提示 */}
          <div style={{ fontSize: 13, color: C.textSecondary, marginBottom: 12 }}>
            基于本班 <b style={{ color: C.textPrimary }}>{result.student_count}</b> 名学生的匿名统计生成。
            {small && <span style={{ color: '#B45309' }}>　⚠ 样本偏少，结论仅供参考。</span>}
          </div>

          {/* 合规说明条 */}
          <div style={{ display: 'flex', gap: 8, alignItems: 'flex-start', background: C.aiLight, border: '1px solid ' + C.aiBorder, borderRadius: 10, padding: '10px 14px', marginBottom: 16 }}>
            <span style={{ fontSize: 14 }}>🔒</span>
            <div style={{ fontSize: 12, color: '#5B21B6', lineHeight: 1.7 }}>
              此总结<b>仅基于全班匿名统计</b>（人数、分层占比、均分、薄弱点频次），
              <b>未使用任何学生的学号、成绩明细或个人信息</b>。下方「AI 看到的数据」即为喂给 AI 的全部内容。
            </div>
          </div>

          {/* AI 看到的数据（脱敏统计量原文，默认展开，可折叠）*/}
          <div style={{ border: '1px solid ' + C.border, borderRadius: 10, marginBottom: 16, overflow: 'hidden' }}>
            <button onClick={() => setStatsOpen(v => !v)}
              style={{ width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 14px', background: '#F9FAFB', border: 'none', cursor: 'pointer', fontSize: 13, fontWeight: 600, color: C.textSecondary }}>
              <span>📊 AI 看到的数据（已脱敏的匿名统计量）</span>
              <span style={{ fontSize: 12 }}>{statsOpen ? '收起 ▲' : '展开 ▼'}</span>
            </button>
            {statsOpen && (
              <pre style={{ margin: 0, padding: '12px 14px', background: '#FCFCFD', fontSize: 12, lineHeight: 1.7, color: C.textSecondary, whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace' }}>
                {result.stats_text}
              </pre>
            )}
          </div>

          {/* 四大段预览 */}
          <div style={{ display: 'grid', gap: 12 }}>
            {sections.map((sec) => (
              <div key={sec.title} style={{ border: '1px solid ' + C.border, borderRadius: 10, padding: '12px 14px' }}>
                <div style={{ fontSize: 13, fontWeight: 700, color: C.textPrimary, marginBottom: 6 }}>{sec.icon} {sec.title}</div>
                <div style={{ fontSize: 13, color: C.textSecondary, lineHeight: 1.8, whiteSpace: 'pre-wrap' }}>
                  {sec.text || <span style={{ color: C.textMuted }}>（AI 未给出此段内容）</span>}
                </div>
              </div>
            ))}
          </div>

          <div style={{ fontSize: 11.5, color: C.textMuted, lineHeight: 1.7, marginTop: 14 }}>
            💡 点「采用并写回班级卡」会用以上四段<b>覆盖</b>该班学情卡的对应内容（班级定位信息不变）。
            采用前可先复制保留旧内容；也可取消后手动微调。
          </div>
        </div>

        {/* 底部操作栏 */}
        <div style={{ padding: '14px 22px', borderTop: '1px solid ' + C.border, display: 'flex', gap: 10 }}>
          <button onClick={() => { if (!adopting) onCancel() }} style={{ ...btnGhost, flex: 1, textAlign: 'center' }}>取消</button>
          <button onClick={onAdopt} disabled={adopting}
            style={{ flex: 2, padding: '10px', borderRadius: 9, border: 'none', background: adopting ? C.textMuted : C.ai, color: '#fff', fontSize: 14, fontWeight: 600, cursor: adopting ? 'wait' : 'pointer' }}>
            {adopting ? '写回中…' : '采用并写回班级卡'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ---------- 按分数线自动分层弹窗（批次2d）----------
function AutoTierModal({
  students, onClose, onDone, doAutoTier,
}: {
  students: ClassStudentView[]
  onClose: () => void
  onDone: (res: AutoTierResult) => void
  doAutoTier: (aLine: number, cLine: number) => Promise<AutoTierResult>
}) {
  const [aLine, setALine] = useState('85')
  const [cLine, setCLine] = useState('65')
  const [submitting, setSubmitting] = useState(false)
  const [err, setErr] = useState('')

  const a = Number(aLine)
  const c = Number(cLine)
  const linesValid = isFinite(a) && isFinite(c) && a > c && a >= 0 && c >= 0

  // 实时预览：用当前 students 的 latest_score 按线分层算各层人数（纯前端，确认时才真正写库）
  let pa = 0, pb = 0, pc = 0, pNoScore = 0
  if (linesValid) {
    for (const st of students) {
      if (st.latest_score === null || st.latest_score === undefined) { pNoScore++; continue }
      const s = st.latest_score
      if (s >= a) pa++
      else if (s < c) pc++
      else pb++
    }
  }

  const doSubmit = async () => {
    if (!linesValid) { setErr('分数线不合法：A 线必须高于 C 线，且均不能为负'); return }
    setErr(''); setSubmitting(true)
    try {
      const res = await doAutoTier(a, c)
      onDone(res)
    } catch (e: any) {
      setErr(e?.message || '分层失败，请重试')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <ModalShell title="🪜 按分数线自动分层" onCancel={onClose}>
      <div style={{ display: 'grid', gap: 14 }}>
        <div style={{ fontSize: 12.5, color: C.textSecondary, lineHeight: 1.7 }}>
          设定两条分数线，系统按每个学生的<b>最新成绩</b>自动归入 ABC 三层。
          没有成绩的学生不参与（分层保持不变）。
        </div>

        <div style={{ display: 'flex', gap: 12 }}>
          <div style={{ flex: 1 }}>
            <label style={labelStyle}>A 层分数线（≥ 此分为 A 拔尖）</label>
            <input type="number" value={aLine} onChange={(e) => setALine(e.target.value)} style={inputStyle} placeholder="如 85" />
          </div>
          <div style={{ flex: 1 }}>
            <label style={labelStyle}>C 层分数线（&lt; 此分为 C 学困）</label>
            <input type="number" value={cLine} onChange={(e) => setCLine(e.target.value)} style={inputStyle} placeholder="如 65" />
          </div>
        </div>
        <div style={{ fontSize: 11.5, color: C.textMuted }}>介于两线之间（C 线 ≤ 分数 &lt; A 线）的学生归为 B 中等。</div>

        {/* 实时预览 */}
        {linesValid ? (
          <div style={{ background: '#ECFEFF', border: '1px solid #A5F3FC', borderRadius: 10, padding: '12px 14px', fontSize: 13, color: '#155E75', lineHeight: 1.9 }}>
            <div style={{ fontWeight: 700, marginBottom: 4 }}>预览（按当前成绩）：</div>
            A 层 <b>{pa}</b> 人　·　B 层 <b>{pb}</b> 人　·　C 层 <b>{pc}</b> 人
            {pNoScore > 0 && <div style={{ color: '#B45309', marginTop: 2 }}>⚠ {pNoScore} 人暂无成绩，不参与本次分层（分层保持不变）。</div>}
          </div>
        ) : (
          <div style={{ background: C.dangerLight, border: '1px solid ' + C.dangerBorder, borderRadius: 10, padding: '10px 14px', fontSize: 12.5, color: C.danger }}>
            请输入合法分数线：A 线必须高于 C 线，且均不能为负。
          </div>
        )}

        {/* 覆盖警告 */}
        <div style={{ fontSize: 11.5, color: '#B45309', lineHeight: 1.6 }}>
          ⚠ 确认后将<b>覆盖所有有成绩学生的现有分层</b>（包括之前手动设过的）。无成绩学生不受影响。
        </div>

        {err && <div style={{ fontSize: 12.5, color: C.danger }}>{err}</div>}
      </div>

      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 18 }}>
        <button onClick={onClose} style={btnGhost}>取消</button>
        <button onClick={doSubmit} disabled={submitting || !linesValid}
          style={{ ...btnPrimary, background: (submitting || !linesValid) ? C.textMuted : '#0891B2', opacity: 1, cursor: (submitting || !linesValid) ? 'not-allowed' : 'pointer' }}>
          {submitting ? '分层中…' : '确认分层'}
        </button>
      </div>
    </ModalShell>
  )
}

// ---------- 通用组件（对齐 ClassProfilesPanel）----------
function ModalShell({ title, children, onCancel }: { title: string; children: ReactNode; onCancel: () => void }) {
  return (
    <div onClick={onCancel} style={{ position: 'fixed', inset: 0, background: 'rgba(17,24,39,0.45)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 9000, padding: 20 }}>
      <div onClick={(e) => e.stopPropagation()} style={{ background: '#fff', borderRadius: 14, padding: 22, width: 460, maxWidth: '100%', boxShadow: '0 12px 48px rgba(0,0,0,0.18)' }}>
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
