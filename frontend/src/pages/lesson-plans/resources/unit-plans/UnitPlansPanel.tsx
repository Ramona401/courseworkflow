import { useState, useEffect, useRef, useCallback } from 'react'
import type { CSSProperties, ReactNode, Dispatch, SetStateAction, MouseEvent as ReactMouseEvent } from 'react'
import {
  getUnitPlans, getUnitPlan, startUnitPlan, chatUnitPlan, saveUnitPlan, deleteUnitPlan,
} from '@/api/unit-plans'
import type {
  UnitPlanListItem, UnitPlanDetail, UnitPlanMessage, UnitPlanScope, UnitPlanStatus,
} from '@/api/unit-plans'
import { getMyPublishGroups } from '@/api/ai-assistants'

const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  border: '#E5E7EB', white: '#FFFFFF',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
}

const SUBJECTS = ['语文', '数学', '英语', '人工智能', '道德与法治', '科学', '物理', '化学', '生物', '历史', '地理', '信息科技']
const GRADES = ['一年级', '二年级', '三年级', '四年级', '五年级', '六年级', '七年级', '八年级', '九年级', '高一', '高二', '高三']
const VOLUMES = ['上册', '下册', '全册']

interface ScopeOption { key: string; label: string; scope: UnitPlanScope; target: string }

function scopeBadge(s: UnitPlanScope) {
  if (s === 'system') return { t: '🌐 全局', bg: '#F3E8FF', c: '#7C3AED' }
  if (s === 'school') return { t: '🏛️ 全校', bg: '#DCFCE7', c: '#16A34A' }
  return { t: '🏫 教研组', bg: '#DBEAFE', c: '#2563EB' }
}
function statusBadge(s: UnitPlanStatus) {
  if (s === 'draft') return { t: '草稿', bg: '#FEF3C7', c: '#B45309' }
  if (s === 'active') return { t: '已发布', bg: '#DCFCE7', c: '#16A34A' }
  return { t: '已归档', bg: '#F3F4F6', c: '#6B7280' }
}

const inputStyle: CSSProperties = {
  width: '100%', padding: '8px 10px', border: '1px solid ' + C.border,
  borderRadius: 8, fontSize: 13, color: C.textPrimary, outline: 'none', boxSizing: 'border-box',
}
const labelStyle: CSSProperties = { fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 4, display: 'block' }
const btnPrimary: CSSProperties = { padding: '8px 16px', background: C.primary, color: '#fff', border: 'none', borderRadius: 8, fontSize: 13, fontWeight: 600, cursor: 'pointer' }
const btnGhost: CSSProperties = { padding: '8px 16px', background: C.white, color: C.textSecondary, border: '1px solid ' + C.border, borderRadius: 8, fontSize: 13, cursor: 'pointer' }

export default function UnitPlansPanel() {
  const [list, setList] = useState<UnitPlanListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [view, setView] = useState<'list' | 'session'>('list')

  const [scopeOptions, setScopeOptions] = useState<ScopeOption[]>([])
  const [canCreate, setCanCreate] = useState(false)

  const [showNew, setShowNew] = useState(false)
  const [nf, setNf] = useState({ scopeKey: '', subject: '语文', grade: '三年级', volume: '下册', unit: '', title: '' })
  const [starting, setStarting] = useState(false)

  const [plan, setPlan] = useState<UnitPlanDetail | null>(null)
  const [messages, setMessages] = useState<UnitPlanMessage[]>([])
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)

  const [showSave, setShowSave] = useState(false)
  const [sf, setSf] = useState({ title: '', unit_theme: '', content: '', atlas: '' })
  const [saving, setSaving] = useState(false)

  const [viewPlan, setViewPlan] = useState<UnitPlanDetail | null>(null)
  const [toast, setToast] = useState('')
  const msgEndRef = useRef<HTMLDivElement | null>(null)

  const showToast = (m: string) => { setToast(m); window.setTimeout(() => setToast(''), 2600) }
  const scrollBottom = () => { window.setTimeout(() => msgEndRef.current?.scrollIntoView({ behavior: 'smooth' }), 60) }

  const loadList = useCallback(async () => {
    setLoading(true)
    try { const r = await getUnitPlans(); setList(r.unit_plans || []) }
    catch (e: any) { showToast(e?.message || '加载失败') }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { loadList() }, [loadList])

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const r: any = await getMyPublishGroups()
        if (cancelled) return
        const opts: ScopeOption[] = (r.groups || []).map((g: any): ScopeOption => ({
          key: 'group:' + g.id,
          label: g.name + '（' + (g.role === 'lead' ? '组长' : '骨干') + '）',
          scope: 'group', target: g.id,
        }))
        if (r.can_publish_system) opts.unshift({ key: 'system:', label: '🌐 全局（所有学校通用）', scope: 'system', target: '' })
        setScopeOptions(opts)
        setCanCreate(opts.length > 0)
        setNf((s) => ({ ...s, scopeKey: opts[0]?.key || '' }))
      } catch { /* 无可发布归属 = 消费端 */ }
    })()
    return () => { cancelled = true }
  }, [])

  const doStart = async () => {
    const opt = scopeOptions.find((o) => o.key === nf.scopeKey)
    if (!opt) { showToast('请选择归属'); return }
    if (!nf.unit.trim()) { showToast('请填写单元'); return }
    setStarting(true)
    try {
      const r = await startUnitPlan({
        scope: opt.scope, scope_target_id: opt.target,
        subject: nf.subject, grade: nf.grade, volume: nf.volume,
        unit: nf.unit.trim(), title: nf.title.trim() || undefined,
      })
      setPlan(r.plan)
      setMessages([{ role: 'assistant', content: r.opening, created_at: '' }])
      setShowNew(false); setView('session'); scrollBottom()
    } catch (e: any) { showToast(e?.message || '开始失败') }
    finally { setStarting(false) }
  }

  const openExisting = async (item: UnitPlanListItem) => {
    try {
      const r = await getUnitPlan(item.id)
      if (r.plan.status === 'draft') { setPlan(r.plan); setMessages(r.messages || []); setView('session'); scrollBottom() }
      else setViewPlan(r.plan)
    } catch (e: any) { showToast(e?.message || '打开失败') }
  }

  const doSend = async () => {
    if (!plan || !input.trim() || sending) return
    const msg = input.trim(); setInput('')
    setMessages((m) => [...m, { role: 'user', content: msg, created_at: '' }])
    setSending(true); scrollBottom()
    try {
      const reply = await chatUnitPlan(plan.id, msg)
      setMessages((m) => [...m, { role: 'assistant', content: reply, created_at: '' }])
    } catch (e: any) {
      setMessages((m) => [...m, { role: 'assistant', content: '（出错了：' + (e?.message || '请重试') + '）', created_at: '' }])
    } finally { setSending(false); scrollBottom() }
  }

  const openSave = () => {
    if (!plan) return
    const lastAi = [...messages].reverse().find((m) => m.role === 'assistant')?.content || ''
    const atlas = lastAi.split('\n').filter((l) => l.trim().startsWith('|')).join('\n')
    setSf({ title: plan.title, unit_theme: plan.unit_theme || '', content: lastAi, atlas })
    setShowSave(true)
  }

  const doSave = async () => {
    if (!plan) return
    if (!sf.content.trim()) { showToast('方案正文为空'); return }
    setSaving(true)
    try {
      await saveUnitPlan(plan.id, sf)
      setShowSave(false); setView('list'); setPlan(null); setMessages([])
      showToast('已保存为正式方案'); loadList()
    } catch (e: any) { showToast(e?.message || '保存失败') }
    finally { setSaving(false) }
  }

  const doDelete = async (item: UnitPlanListItem, e: ReactMouseEvent) => {
    e.stopPropagation()
    if (!window.confirm('确认删除「' + item.title + '」？此操作不可恢复。')) return
    try { await deleteUnitPlan(item.id); showToast('已删除'); loadList() }
    catch (err: any) { showToast(err?.message || '删除失败') }
  }

  if (view === 'session' && plan) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 230px)', minHeight: 420 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, paddingBottom: 12, borderBottom: '1px solid ' + C.border, marginBottom: 12 }}>
          <button onClick={() => { setView('list'); loadList() }} style={btnGhost}>← 返回列表</button>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 15, fontWeight: 700, color: C.textPrimary, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{plan.title}</div>
            <div style={{ fontSize: 12, color: C.textMuted }}>{plan.subject} · {plan.grade}{plan.volume} · {plan.unit}　🎓 大单元架构师逐步引导</div>
          </div>
          <button onClick={openSave} style={btnPrimary}>💾 保存为正式方案</button>
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: '4px 2px' }}>
          {messages.map((m, i) => (
            <div key={i} style={{ display: 'flex', justifyContent: m.role === 'user' ? 'flex-end' : 'flex-start', marginBottom: 14 }}>
              <div style={{
                maxWidth: '82%', padding: '10px 14px', borderRadius: 12, fontSize: 13.5, lineHeight: 1.7,
                whiteSpace: 'pre-wrap', wordBreak: 'break-word',
                background: m.role === 'user' ? C.primary : C.white,
                color: m.role === 'user' ? '#fff' : C.textPrimary,
                border: m.role === 'user' ? 'none' : '1px solid ' + C.border,
              }}>{m.content}</div>
            </div>
          ))}
          {sending && (
            <div style={{ display: 'flex', justifyContent: 'flex-start', marginBottom: 14 }}>
              <div style={{ padding: '10px 14px', borderRadius: 12, fontSize: 13, color: C.textMuted, background: C.white, border: '1px solid ' + C.border }}>架构师思考中…</div>
            </div>
          )}
          <div ref={msgEndRef} />
        </div>
        <div style={{ display: 'flex', gap: 10, paddingTop: 12, borderTop: '1px solid ' + C.border, marginTop: 8 }}>
          <textarea value={input} onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); doSend() } }}
            placeholder="确认 / 补充 / 让架构师按你的意见改这一步…（Enter 发送，Shift+Enter 换行）"
            rows={2} style={{ ...inputStyle, resize: 'vertical', flex: 1 }} />
          <button onClick={doSend} disabled={sending || !input.trim()} style={{ ...btnPrimary, opacity: sending || !input.trim() ? 0.5 : 1 }}>发送</button>
        </div>
        {showSave && <SaveModal sf={sf} setSf={setSf} saving={saving} onCancel={() => setShowSave(false)} onSave={doSave} />}
        {toast && <Toast text={toast} />}
      </div>
    )
  }

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
        <div style={{ fontSize: 13, color: C.textSecondary }}>
          大单元工坊产出的整单元教学设计方案。{canCreate ? '由学科负责人逐步生成，全组（或全局）可参考。' : '由学科负责人产出，你可在此查看本组/全局已发布的方案。'}
        </div>
        {canCreate && <button onClick={() => { setNf((s) => ({ ...s, unit: '', title: '' })); setShowNew(true) }} style={btnPrimary}>＋ 新建单元方案</button>}
      </div>

      {loading ? (
        <div style={{ padding: 40, textAlign: 'center', color: C.textMuted, fontSize: 13 }}>加载中…</div>
      ) : list.length === 0 ? (
        <div style={{ padding: 48, textAlign: 'center', background: C.white, borderRadius: 12, border: '1px dashed ' + C.border }}>
          <div style={{ fontSize: 30, marginBottom: 10 }}>🗂️</div>
          <div style={{ fontSize: 14, color: C.textSecondary }}>还没有单元方案{canCreate ? '，点右上角「新建单元方案」开始。' : '。'}</div>
        </div>
      ) : (
        <div style={{ display: 'grid', gap: 12 }}>
          {list.map((it) => {
            const sb = scopeBadge(it.scope); const st = statusBadge(it.status)
            return (
              <div key={it.id} onClick={() => openExisting(it)} style={{ padding: '14px 16px', background: C.white, border: '1px solid ' + C.border, borderRadius: 12, cursor: 'pointer', transition: 'all 140ms ease' }}
                onMouseEnter={(e) => { e.currentTarget.style.borderColor = C.primary; e.currentTarget.style.boxShadow = '0 2px 10px rgba(79,123,232,0.10)' }}
                onMouseLeave={(e) => { e.currentTarget.style.borderColor = C.border; e.currentTarget.style.boxShadow = 'none' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                  <span style={{ fontSize: 11, padding: '2px 8px', borderRadius: 6, background: sb.bg, color: sb.c, fontWeight: 600 }}>{sb.t}</span>
                  <span style={{ fontSize: 11, padding: '2px 8px', borderRadius: 6, background: st.bg, color: st.c, fontWeight: 600 }}>{st.t}</span>
                  {it.scope_name && <span style={{ fontSize: 12, color: C.textMuted }}>{it.scope_name}</span>}
                  <div style={{ flex: 1 }} />
                  <button onClick={(e) => doDelete(it, e)} style={{ background: 'none', border: 'none', color: C.textMuted, cursor: 'pointer', fontSize: 13 }}>🗑</button>
                </div>
                <div style={{ fontSize: 15, fontWeight: 700, color: C.textPrimary, marginBottom: 4 }}>{it.title}</div>
                <div style={{ fontSize: 12.5, color: C.textSecondary }}>
                  {it.subject} · {it.grade}{it.volume} · {it.unit}{it.unit_theme ? '　主题：' + it.unit_theme : ''}{it.creator_name ? '　· ' + it.creator_name : ''}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {showNew && (
        <ModalShell title="新建单元方案" onCancel={() => setShowNew(false)}>
          <div style={{ display: 'grid', gap: 12 }}>
            <div>
              <label style={labelStyle}>归属（谁能看到这份方案）</label>
              <select value={nf.scopeKey} onChange={(e) => setNf((s) => ({ ...s, scopeKey: e.target.value }))} style={inputStyle}>
                {scopeOptions.map((o) => <option key={o.key} value={o.key}>{o.label}</option>)}
              </select>
              {nf.scopeKey === 'system:' && <div style={{ fontSize: 11, color: '#7C3AED', marginTop: 4 }}>🌐 全局方案对所有学校的老师可见。</div>}
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10 }}>
              <div><label style={labelStyle}>学科</label>
                <select value={nf.subject} onChange={(e) => setNf((s) => ({ ...s, subject: e.target.value }))} style={inputStyle}>{SUBJECTS.map((x) => <option key={x} value={x}>{x}</option>)}</select></div>
              <div><label style={labelStyle}>年级</label>
                <select value={nf.grade} onChange={(e) => setNf((s) => ({ ...s, grade: e.target.value }))} style={inputStyle}>{GRADES.map((x) => <option key={x} value={x}>{x}</option>)}</select></div>
              <div><label style={labelStyle}>册次</label>
                <select value={nf.volume} onChange={(e) => setNf((s) => ({ ...s, volume: e.target.value }))} style={inputStyle}>{VOLUMES.map((x) => <option key={x} value={x}>{x}</option>)}</select></div>
            </div>
            <div><label style={labelStyle}>单元（如：第二单元 / 寓言单元）</label>
              <input value={nf.unit} onChange={(e) => setNf((s) => ({ ...s, unit: e.target.value }))} style={inputStyle} placeholder="本次要做整体设计的单元" /></div>
            <div><label style={labelStyle}>标题（可选，留空自动生成）</label>
              <input value={nf.title} onChange={(e) => setNf((s) => ({ ...s, title: e.target.value }))} style={inputStyle} placeholder="如：三下第二单元 寓言故事 大单元设计" /></div>
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 18 }}>
            <button onClick={() => setShowNew(false)} style={btnGhost}>取消</button>
            <button onClick={doStart} disabled={starting} style={{ ...btnPrimary, opacity: starting ? 0.6 : 1 }}>{starting ? '正在准备…' : '开始逐步设计'}</button>
          </div>
        </ModalShell>
      )}

      {viewPlan && (
        <ModalShell title={viewPlan.title} wide onCancel={() => setViewPlan(null)}>
          <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 10 }}>{viewPlan.subject} · {viewPlan.grade}{viewPlan.volume} · {viewPlan.unit}{viewPlan.unit_theme ? '　主题：' + viewPlan.unit_theme : ''}</div>
          <div style={{ maxHeight: '60vh', overflowY: 'auto' }}>
            <div style={{ fontSize: 13, fontWeight: 700, color: C.textPrimary, margin: '6px 0' }}>方案文档</div>
            <div style={{ whiteSpace: 'pre-wrap', fontSize: 13, lineHeight: 1.7, color: C.textPrimary }}>{viewPlan.content || '（空）'}</div>
            {viewPlan.atlas && (
              <>
                <div style={{ fontSize: 13, fontWeight: 700, color: C.textPrimary, margin: '16px 0 6px' }}>单元整体设计图谱</div>
                <div style={{ whiteSpace: 'pre-wrap', fontSize: 12.5, lineHeight: 1.7, color: C.textSecondary, fontFamily: 'monospace' }}>{viewPlan.atlas}</div>
              </>
            )}
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 16 }}>
            <button onClick={() => setViewPlan(null)} style={btnGhost}>关闭</button>
          </div>
        </ModalShell>
      )}

      {toast && <Toast text={toast} />}
    </div>
  )
}

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

function SaveModal({ sf, setSf, saving, onCancel, onSave }: {
  sf: { title: string; unit_theme: string; content: string; atlas: string }
  setSf: Dispatch<SetStateAction<{ title: string; unit_theme: string; content: string; atlas: string }>>
  saving: boolean; onCancel: () => void; onSave: () => void
}) {
  return (
    <ModalShell title="保存为正式方案" wide onCancel={onCancel}>
      <div style={{ fontSize: 12, color: '#6B7280', marginBottom: 12 }}>已自动填入最后一步的方案与图谱，确认或微调后保存。保存后该方案对归属范围内的老师可见。</div>
      <div style={{ display: 'grid', gap: 12 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          <div><label style={labelStyle}>标题</label>
            <input value={sf.title} onChange={(e) => setSf((s) => ({ ...s, title: e.target.value }))} style={inputStyle} /></div>
          <div><label style={labelStyle}>单元任务主题（可选）</label>
            <input value={sf.unit_theme} onChange={(e) => setSf((s) => ({ ...s, unit_theme: e.target.value }))} style={inputStyle} /></div>
        </div>
        <div><label style={labelStyle}>方案文档</label>
          <textarea value={sf.content} onChange={(e) => setSf((s) => ({ ...s, content: e.target.value }))} rows={12} style={{ ...inputStyle, resize: 'vertical', fontFamily: 'inherit' }} /></div>
        <div><label style={labelStyle}>单元整体设计图谱（表格，可留空）</label>
          <textarea value={sf.atlas} onChange={(e) => setSf((s) => ({ ...s, atlas: e.target.value }))} rows={6} style={{ ...inputStyle, resize: 'vertical', fontFamily: 'monospace' }} /></div>
      </div>
      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 18 }}>
        <button onClick={onCancel} style={btnGhost}>取消</button>
        <button onClick={onSave} disabled={saving} style={{ ...btnPrimary, opacity: saving ? 0.6 : 1 }}>{saving ? '保存中…' : '确认保存'}</button>
      </div>
    </ModalShell>
  )
}

function Toast({ text }: { text: string }) {
  return (
    <div style={{ position: 'fixed', bottom: 28, left: '50%', transform: 'translateX(-50%)', background: 'rgba(17,24,39,0.92)', color: '#fff', padding: '10px 20px', borderRadius: 10, fontSize: 13, zIndex: 9999, boxShadow: '0 6px 24px rgba(0,0,0,0.2)' }}>{text}</div>
  )
}
