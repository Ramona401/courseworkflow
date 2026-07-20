/**
 * CourseOutlinesPage.tsx — 课程大纲管理页（大单元备课能力·批次一）
 *
 * 谁能进：菜单/路由放行 admin + senior_operator + operator（组长是 operator 角色 + 教研组 lead 身份）。
 * 谁能建/改：归属下拉的可选项由 getMyPublishGroups() + 本地 admin 全局选项决定——
 *   - 担任 lead/backbone 的教研组 → 可建该组的 group 级大纲
 *   - 校管(can_publish_school) → 可建本校 school 级大纲（第一版暂关，待 my-groups 补学校ID）
 *   - admin → 可建 system 全局级大纲（所有学校通用，target 留空，后端填占位ID）
 *   普通老师(无任何可管理归属) → 看到"你没有可管理的归属"，只能查看不能建。
 * 后端 service 层 canManageScope 是最终防线（system 仅 admin）。
 *
 * 设计要点：大纲正文 content 存原文整块，录入就是一个大文本框，粘贴即可，不拆字段。
 */
import { useEffect, useState, useCallback } from 'react'
import {
  getCourseOutlines,
  getCourseOutline,
  createCourseOutline,
  updateCourseOutline,
  deleteCourseOutline,
  type CourseOutlineListItem,
  type CourseOutlineScope,
  COURSE_OUTLINE_PUBLISHERS,
  COURSE_OUTLINE_PUBLISHER_GENERIC_LABEL,
  publisherLabel,
} from '@/api/course-outlines'
import { getMyPublishGroups, type PublishGroup } from '@/api/ai-assistants'
import { useAuth } from '@/store/auth'
import { useEducationProfile } from '@/hooks/useEducationProfile'

// ---------- 配色（与 LPSidebar 蓝紫系一致） ----------
const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  border: '#E5E7EB',
  borderLight: '#F3F4F6',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  bg: '#FAFBFC',
  white: '#FFFFFF',
  danger: '#EF4444',
  success: '#10B981',
  system: '#6D28D9',
}

/** 归属可选项（统一 group/school/system 为一个下拉） */
interface ScopeOption {
  scope: CourseOutlineScope
  targetId: string
  label: string       // 展示名（如"英语教研组（XX小学）· 教研组"或"🌐 全局（所有学校通用）"）
}

/** 编辑弹窗的表单态 */
interface FormState {
  id?: string                 // 有 = 编辑，无 = 新建
  scopeKey: string            // "group:<id>" / "school:<id>" / "system:"，对应 ScopeOption 唯一键
  subject: string
  grade: string
  volume: string
  publisher: string           // 教材版本（空串=通用/不限版本）
  title: string
  content: string
}

const emptyForm: FormState = {
  scopeKey: '', subject: '', grade: '', volume: '', publisher: '', title: '', content: '',
}

/** 全局大纲选项（admin 专属，target 留空由后端填占位ID） */
const SYSTEM_OPTION: ScopeOption = { scope: 'system', targetId: '', label: '🌐 全局（所有学校通用）' }

export default function CourseOutlinesPage({ embedded = false }: { embedded?: boolean } = {}) {
  const { user } = useAuth()
  const {
    isK12,
    profile,
    ready: educationReady,
  } = useEducationProfile()

  const isAdmin = user?.role === 'admin'

  /**
   * admin保留K12全局课程大纲管理兼容能力。
   *
   * 普通用户只有在教育域完成解析且确认为K12时，
   * 才展示出版社字段。
   */
  const showPublisher =
    isAdmin ||
    (
      educationReady &&
      isK12 &&
      profile.publisher_enabled
    )

  const [list, setList] = useState<CourseOutlineListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [scopeOptions, setScopeOptions] = useState<ScopeOption[]>([])
  const [canPublishSchool, setCanPublishSchool] = useState(false)
  const [mySchoolName, setMySchoolName] = useState('')

  // 弹窗
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState<FormState>(emptyForm)
  const [saving, setSaving] = useState(false)
  const [modalError, setModalError] = useState('')

  // 全局提示
  const [toast, setToast] = useState<{ type: 'success' | 'error'; msg: string } | null>(null)
  const flash = (type: 'success' | 'error', msg: string) => {
    setToast({ type, msg })
    setTimeout(() => setToast(null), 2600)
  }

  // ---------- 加载列表 ----------
  const loadList = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getCourseOutlines()
      setList(res.outlines || [])
    } catch (e) {
      flash('error', e instanceof Error ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  // ---------- 加载归属可选项（复用 my-groups + admin 全局选项） ----------
  const loadScopeOptions = useCallback(async () => {
    try {
      const res = await getMyPublishGroups()
      const opts: ScopeOption[] = []
      // 教研组级：我担任 lead/backbone 的组
      ;(res.groups || []).forEach((g: PublishGroup) => {
        opts.push({
          scope: 'group',
          targetId: g.id,
          label: `${g.name}（${g.school_name}）· 教研组`,
        })
      })
      // 学校级：第一版暂不开放（my-groups 接口尚未返回真实学校ID，
      //   待下一小步给后端 my-groups 补学校ID字段后再开放，避免拿占位ID落错归属）。
      // 全局级：仅 admin，可录"所有学校通用"的课程大纲（置顶为默认选项）。
      if (isAdmin) {
        opts.unshift(SYSTEM_OPTION)
      }
      setCanPublishSchool(res.can_publish_school)
      setScopeOptions(opts)
    } catch (e) {
      // my-groups 失败不阻断查看；admin 即便失败也应能录全局大纲
      if (isAdmin) {
        setScopeOptions([SYSTEM_OPTION])
      }
      console.error('加载归属选项失败', e)
    }
  }, [isAdmin])

  useEffect(() => { loadList() }, [loadList])
  useEffect(() => { loadScopeOptions() }, [loadScopeOptions])

  // ---------- 打开新建 ----------
  const openCreate = () => {
    setForm({ ...emptyForm, scopeKey: scopeOptions[0] ? `${scopeOptions[0].scope}:${scopeOptions[0].targetId}` : '' })
    setModalError('')
    setShowModal(true)
  }

  // ---------- 打开编辑（拉详情回填） ----------
  const openEdit = async (id: string) => {
    setModalError('')
    try {
      const d = await getCourseOutline(id)
      setForm({
        id: d.id,
        scopeKey: `${d.scope}:${d.scope_target_id}`,  // 编辑时归属不可改（下方禁用下拉）
        subject: d.subject,
        grade: d.grade,
        volume: d.volume,
        publisher: showPublisher
          ? (d.publisher ?? '')
          : '',
        title: d.title,
        content: d.content,
      })
      setShowModal(true)
    } catch (e) {
      flash('error', e instanceof Error ? e.message : '加载详情失败')
    }
  }

  // ---------- 保存（新建/编辑） ----------
  const handleSave = async () => {
    setModalError('')
    if (!form.subject.trim() || !form.grade.trim() || !form.volume.trim() ||
        !form.title.trim() || !form.content.trim()) {
      setModalError('学科、年级、册次、标题、正文均为必填')
      return
    }
    if (!form.id && !form.scopeKey) {
      setModalError('请选择归属（教研组 / 学校 / 全局）')
      return
    }
    setSaving(true)
    try {
      if (form.id) {
        // 编辑：归属不变，只更新内容
        await updateCourseOutline(form.id, {
          subject: form.subject.trim(),
          grade: form.grade.trim(),
          volume: form.volume.trim(),
          publisher: showPublisher
            ? form.publisher.trim()
            : '',
          title: form.title.trim(),
          content: form.content,
        })
        flash('success', '更新成功')
      } else {
        // 新建：解析 scopeKey（system 的 targetId 为空，后端填占位ID）
        const [scope, targetId] = form.scopeKey.split(':') as [CourseOutlineScope, string]
        await createCourseOutline({
          publisher: showPublisher
            ? form.publisher.trim()
            : '',
          scope,
          scope_target_id: targetId,
          subject: form.subject.trim(),
          grade: form.grade.trim(),
          volume: form.volume.trim(),
          title: form.title.trim(),
          content: form.content,
        })
        flash('success', '创建成功')
      }
      setShowModal(false)
      loadList()
    } catch (e) {
      setModalError(e instanceof Error ? e.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  // ---------- 删除 ----------
  const handleDelete = async (item: CourseOutlineListItem) => {
    if (!window.confirm(`确定删除大纲「${item.title}」吗？删除后备课将不再引用它。`)) return
    try {
      await deleteCourseOutline(item.id)
      flash('success', '已删除')
      loadList()
    } catch (e) {
      flash('error', e instanceof Error ? e.message : '删除失败')
    }
  }

  const canManageAny = scopeOptions.length > 0
  const isSystemSelected = form.scopeKey.startsWith('system:')

  // ==================== 渲染 ====================
  return (
    <div style={{ padding: embedded ? 0 : '24px 28px', maxWidth: 1100, margin: '0 auto' }}>
      {/* 标题区：独立访问时显示完整标题；嵌入"我的备课资料"时由外壳给标题，这里只保留操作按钮 */}
      {!embedded ? (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
          <div>
            <h1 style={{ fontSize: 22, fontWeight: 700, color: C.textPrimary, margin: 0 }}>📖 课程大纲</h1>
            <div style={{ fontSize: 13, color: C.textSecondary, marginTop: 6 }}>
              一册书的完整课时地图。备课时系统会按「学科+年级+册次」自动把对应大纲喂给 AI，让它备某一课时也知道整册全貌。
            </div>
          </div>
          {canManageAny && (
            <button onClick={openCreate} style={btnPrimary}>＋ 新建大纲</button>
          )}
        </div>
      ) : (
        canManageAny && (
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 6 }}>
            <button onClick={openCreate} style={btnPrimary}>＋ 新建大纲</button>
          </div>
        )
      )}

      {/* 无可管理归属的提示 */}
      {!canManageAny && !loading && (
        <div style={{
          marginTop: 16, padding: '12px 16px', borderRadius: 10,
          background: '#FFF7ED', border: '1px solid #FED7AA', color: '#9A3412', fontSize: 13,
        }}>
          你目前不是任何教研组的组长/骨干，也不是学校管理员，因此只能查看课程大纲，无法新建或编辑。
          如需录入本组大纲，请联系教研组长或学校管理员。
        </div>
      )}

      {/* 列表 */}
      <div style={{ marginTop: 18 }}>
        {loading ? (
          <div style={{ padding: 40, textAlign: 'center', color: C.textMuted }}>加载中…</div>
        ) : list.length === 0 ? (
          <div style={{
            padding: 48, textAlign: 'center', color: C.textMuted,
            background: C.white, borderRadius: 12, border: `1px dashed ${C.border}`,
          }}>
            还没有课程大纲。{canManageAny ? '点击右上角「新建大纲」录入第一份。' : ''}
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {list.map((it) => (
              <div key={it.id} style={cardStyle}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                    <span style={{ fontSize: 15, fontWeight: 600, color: C.textPrimary }}>{it.title}</span>
                    <span style={scopeBadge(it.scope)}>{scopeBadgeText(it)}</span>
                  </div>
                  <div style={{ fontSize: 12, color: C.textSecondary, marginTop: 6 }}>
                    {it.subject} · {it.grade} · {it.volume}
                    {showPublisher && (
                      <span style={{
                        marginLeft: 8,
                        padding: '1px 8px',
                        borderRadius: 6,
                        fontSize: 11,
                        background: it.publisher
                          ? 'rgba(79,123,232,0.10)'
                          : C.borderLight,
                        color: it.publisher
                          ? C.primary
                          : C.textMuted,
                      }}>
                        {publisherLabel(it.publisher)}
                      </span>
                    )}
                    <span style={{ color: C.textMuted, marginLeft: 10 }}>
                      由 {it.creator_name || '—'} 维护 · 更新于 {fmtDate(it.updated_at)}
                    </span>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 8, flexShrink: 0 }}>
                  <button onClick={() => openEdit(it.id)} style={btnGhost}>查看/编辑</button>
                  <button onClick={() => handleDelete(it)} style={btnDanger}>删除</button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 新建/编辑弹窗 */}
      {showModal && (
        <div style={overlay} onClick={() => !saving && setShowModal(false)}>
          <div style={modalBox} onClick={(e) => e.stopPropagation()}>
            <div style={{ fontSize: 17, fontWeight: 700, color: C.textPrimary, marginBottom: 4 }}>
              {form.id ? '编辑课程大纲' : '新建课程大纲'}
            </div>
            <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 16 }}>
              正文直接把整份大纲粘进来即可，无需排版，系统原样保存。
            </div>

            {/* 归属（新建可选，编辑禁用） */}
            <Field label="归属">
              <select
                value={form.scopeKey}
                disabled={!!form.id}
                onChange={(e) => setForm({ ...form, scopeKey: e.target.value })}
                style={{ ...inputStyle, background: form.id ? C.borderLight : C.white }}
              >
                {form.id ? (
                  <option value={form.scopeKey}>（编辑时归属不可更改）</option>
                ) : (
                  <>
                    <option value="">请选择归属…</option>
                    {scopeOptions.map((o) => (
                      <option key={`${o.scope}:${o.targetId}`} value={`${o.scope}:${o.targetId}`}>{o.label}</option>
                    ))}
                  </>
                )}
              </select>
            </Field>

            {/* 全局大纲提示 */}
            {!form.id && isSystemSelected && (
              <div style={{
                fontSize: 12, color: C.system, marginTop: -6, marginBottom: 12,
                padding: '8px 12px', borderRadius: 8,
                background: 'rgba(139,92,246,0.08)', border: '1px solid rgba(139,92,246,0.2)',
              }}>
                🌐 全局大纲对<b>所有学校</b>的老师生效，备课时按「学科 + 年级」自动调用。请确认内容具有普适性。
              </div>
            )}

            {/* 学科 / 年级 / 册次 三个检索标签 */}
            <div style={{ display: 'flex', gap: 10 }}>
              <Field label="学科" flex>
                <input value={form.subject} onChange={(e) => setForm({ ...form, subject: e.target.value })}
                  placeholder="如：语文" style={inputStyle} />
              </Field>
              <Field label="年级" flex>
                <input value={form.grade} onChange={(e) => setForm({ ...form, grade: e.target.value })}
                  placeholder="如：三年级" style={inputStyle} />
              </Field>
              <Field label="册次" flex>
                <input value={form.volume} onChange={(e) => setForm({ ...form, volume: e.target.value })}
                  placeholder="如：下册" style={inputStyle} />
              </Field>
            </div>

            {showPublisher && (
              <>
            {/* 教材版本：预置下拉 + 可手动输入新版本（一标多本，空=通用/不限版本） */}
            <Field label="教材版本">
              <select
                value={form.publisher === '' || COURSE_OUTLINE_PUBLISHERS.includes(form.publisher) ? form.publisher : '__custom__'}
                onChange={(e) => {
                  const v = e.target.value
                  if (v === '__custom__') {
                    setForm({ ...form, publisher: COURSE_OUTLINE_PUBLISHERS.includes(form.publisher) ? ' ' : (form.publisher || ' ') })
                  } else {
                    setForm({ ...form, publisher: v })
                  }
                }}
                style={inputStyle}
              >
                <option value="">{COURSE_OUTLINE_PUBLISHER_GENERIC_LABEL}</option>
                {COURSE_OUTLINE_PUBLISHERS.map((p) => (
                  <option key={p} value={p}>{p}</option>
                ))}
                <option value="__custom__">➕ 其他版本（手动输入）</option>
              </select>
              {form.publisher !== '' && !COURSE_OUTLINE_PUBLISHERS.includes(form.publisher) && (
                <input
                  value={form.publisher === ' ' ? '' : form.publisher}
                  onChange={(e) => setForm({ ...form, publisher: e.target.value })}
                  placeholder="输入教材版本名，如：冀教版"
                  style={{ ...inputStyle, marginTop: 8 }}
                />
              )}
            </Field>

              </>
            )}

            <Field label="标题">
              <input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })}
                placeholder="如：三年级下册语文课程大纲" style={inputStyle} />
            </Field>

            <Field label="大纲正文（原文整块）">
              <textarea
                value={form.content}
                onChange={(e) => setForm({ ...form, content: e.target.value })}
                placeholder="把整份课程大纲粘贴到这里，含各单元、课文、课时安排……系统原样保存，不限格式。"
                style={{ ...inputStyle, minHeight: 240, resize: 'vertical', fontFamily: 'inherit', lineHeight: 1.6 }}
              />
            </Field>

            {modalError && (
              <div style={{ color: C.danger, fontSize: 13, marginBottom: 10 }}>{modalError}</div>
            )}

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 8 }}>
              <button onClick={() => setShowModal(false)} disabled={saving} style={btnGhost}>取消</button>
              <button onClick={handleSave} disabled={saving} style={btnPrimary}>
                {saving ? '保存中…' : (form.id ? '保存修改' : '创建')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 全局 toast */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: 28, left: '50%', transform: 'translateX(-50%)',
          padding: '10px 20px', borderRadius: 10, color: C.white, fontSize: 14, zIndex: 9999,
          background: toast.type === 'success' ? C.success : C.danger,
          boxShadow: '0 4px 16px rgba(0,0,0,0.18)',
        }}>{toast.msg}</div>
      )}
    </div>
  )
}

// ==================== 子组件 / 样式 ====================
function Field({ label, children, flex }: { label: string; children: React.ReactNode; flex?: boolean }) {
  return (
    <div style={{ marginBottom: 14, flex: flex ? 1 : undefined }}>
      <div style={{ fontSize: 13, fontWeight: 600, color: C.textSecondary, marginBottom: 6 }}>{label}</div>
      {children}
    </div>
  )
}

/** 列表徽章文案：group/school 显"类型 · 归属名"，system 显"🌐 全局（所有学校通用）" */
function scopeBadgeText(it: CourseOutlineListItem): string {
  if (it.scope === 'group') return `🏫 教研组 · ${it.scope_name}`
  if (it.scope === 'school') return `🏛️ 学校 · ${it.scope_name}`
  return `🌐 ${it.scope_name || '全局（所有学校通用）'}`
}

function fmtDate(s: string): string {
  if (!s) return '—'
  try {
    const d = new Date(s)
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  } catch { return s }
}

const cardStyle: React.CSSProperties = {
  display: 'flex', alignItems: 'center', gap: 12,
  padding: '16px 18px', background: C.white,
  borderRadius: 12, border: `1px solid ${C.borderLight}`,
}
const scopeBadge = (scope: string): React.CSSProperties => ({
  fontSize: 11, padding: '2px 8px', borderRadius: 6,
  background: scope === 'group' ? 'rgba(79,123,232,0.1)'
    : scope === 'school' ? 'rgba(16,185,129,0.1)'
    : 'rgba(139,92,246,0.12)',
  color: scope === 'group' ? '#3B5BC4'
    : scope === 'school' ? '#0F766E'
    : '#6D28D9',
})
const inputStyle: React.CSSProperties = {
  width: '100%', boxSizing: 'border-box', padding: '9px 12px',
  borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 14, color: C.textPrimary, outline: 'none',
}
const btnPrimary: React.CSSProperties = {
  padding: '9px 18px', borderRadius: 9, border: 'none',
  background: 'linear-gradient(135deg, #4F7BE8, #6366F1)', color: '#fff',
  fontSize: 14, fontWeight: 600, cursor: 'pointer',
}
const btnGhost: React.CSSProperties = {
  padding: '8px 14px', borderRadius: 9, border: `1px solid ${C.border}`,
  background: C.white, color: C.textSecondary, fontSize: 13, cursor: 'pointer',
}
const btnDanger: React.CSSProperties = {
  padding: '8px 14px', borderRadius: 9, border: `1px solid #FECACA`,
  background: '#FEF2F2', color: C.danger, fontSize: 13, cursor: 'pointer',
}
const overlay: React.CSSProperties = {
  position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)',
  display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 9000, padding: 20,
}
const modalBox: React.CSSProperties = {
  background: C.white, borderRadius: 16, padding: 24,
  width: '100%', maxWidth: 620, maxHeight: '90vh', overflowY: 'auto',
  boxShadow: '0 12px 40px rgba(0,0,0,0.2)',
}
