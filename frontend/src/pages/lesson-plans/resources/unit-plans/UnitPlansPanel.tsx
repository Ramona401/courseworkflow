/**
 * UnitPlansPanel.tsx — 大单元备课面板（单元方案 Tab）
 *
 * v233 新增（课程大纲教材版本绑定，对齐备课工坊）：
 *   新建会话弹窗在学科+年级选定后，调 getAvailablePublishers 拉取该学科年级
 *   真实存在大纲的教材版本列表；有版本才显示「课程大纲教材版本」下拉：
 *     - 「不关联课程大纲」（哨兵值 PUBLISHER_NONE，不传字段 → 后端落 NULL，不注入）
 *     - 「通用 / 不限版本」（空串，publisherLabel 转文案）
 *     - 各具名版本（人教版等，后端按版本精确匹配注入，零跨版本兜底）
 *   无版本则显示提示文案，本次备课不关联大纲。会话建立时定版，中途不可改。
 *   会话顶栏与只读详情弹窗回显已绑定版本（📖）。
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import { DEFAULT_SUBJECTS } from '@/constants/subjects'
import type { CSSProperties, ReactNode, Dispatch, SetStateAction, MouseEvent as ReactMouseEvent } from 'react'
import {
  getUnitPlans, getUnitPlan, startUnitPlan, chatUnitPlan, saveUnitPlan, deleteUnitPlan,
} from '@/api/unit-plans'
import type {
  UnitPlanListItem, UnitPlanDetail, UnitPlanMessage, UnitPlanScope, UnitPlanStatus,
} from '@/api/unit-plans'
import { getMyPublishGroups } from '@/api/ai-assistants'
import { getTextbooks, getTextbook, type TextbookListItem } from '@/api/textbooks'
import { getAvailablePublishers, publisherLabel } from '@/api/course-outlines'

// ==================== 颜色常量 ====================
const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  border: '#E5E7EB', white: '#FFFFFF',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
}

// ==================== 下拉选项（单一真相源） ====================
const SUBJECTS = [...DEFAULT_SUBJECTS]
const GRADES = ['一年级', '二年级', '三年级', '四年级', '五年级', '六年级', '七年级', '八年级', '九年级', '高一', '高二', '高三']
const VOLUMES = ['上册', '下册', '全册']

// 「不关联课程大纲」哨兵值（v233）——
// 注意空串('')是有效版本值（通用/不限版本），不能用空串表达"不关联"，故用独立哨兵。
// 选哨兵时 doStart 不传 course_outline_publisher 字段 → 后端落 NULL → 不注入大纲。
const PUBLISHER_NONE = '__NONE__'

// ==================== 归属类型 ====================
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

// ==================== 通用样式 ====================
const inputStyle: CSSProperties = {
  width: '100%', padding: '8px 10px', border: '1px solid ' + C.border,
  borderRadius: 8, fontSize: 13, color: C.textPrimary, outline: 'none', boxSizing: 'border-box',
}
const labelStyle: CSSProperties = { fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 4, display: 'block' }
const btnPrimary: CSSProperties = { padding: '8px 16px', background: C.primary, color: '#fff', border: 'none', borderRadius: 8, fontSize: 13, fontWeight: 600, cursor: 'pointer' }
const btnGhost: CSSProperties = { padding: '8px 16px', background: C.white, color: C.textSecondary, border: '1px solid ' + C.border, borderRadius: 8, fontSize: 13, cursor: 'pointer' }
// 课本上传按钮样式（绿色小按钮）
const btnTextbook: CSSProperties = { padding: '6px 12px', background: '#10B981', color: '#fff', border: 'none', borderRadius: 8, fontSize: 12, fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap' }

// ==================== 主组件 ====================
export default function UnitPlansPanel() {
  const [list, setList] = useState<UnitPlanListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [view, setView] = useState<'list' | 'session'>('list')

  const [scopeOptions, setScopeOptions] = useState<ScopeOption[]>([])
  const [canCreate, setCanCreate] = useState(false)

  const [showNew, setShowNew] = useState(false)
  const [nf, setNf] = useState({ scopeKey: '', subject: '语文', grade: '三年级', volume: '下册', unit: '', title: '', publisher: PUBLISHER_NONE })
  const [starting, setStarting] = useState(false)

  // v233：新建弹窗的可选教材版本列表（该学科年级真实存在大纲的版本；空数组=无大纲不显示选择器）
  const [availablePublishers, setAvailablePublishers] = useState<string[]>([])
  const [publishersLoading, setPublishersLoading] = useState(false)

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

  // 课本上传相关状态
  const [showTextbookPicker, setShowTextbookPicker] = useState(false)
  const [textbookContext, setTextbookContext] = useState('') // 当前会话关联的课本OCR文字
  const [textbookCount, setTextbookCount] = useState(0) // 关联的课本页数

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

  // v233：新建弹窗打开时 / 学科·年级变化时，拉取该学科年级可用的课程大纲教材版本列表。
  // 每次重拉都把已选版本重置回"不关联"（换了学科年级，旧选择可能不在新列表里）。
  // 带取消守卫：弹窗关闭或依赖变化时丢弃过期响应，避免竞态覆盖。
  useEffect(() => {
    if (!showNew) return
    let cancelled = false
    setPublishersLoading(true)
    setAvailablePublishers([])
    setNf((s) => ({ ...s, publisher: PUBLISHER_NONE }))
    ;(async () => {
      try {
        const pubs = await getAvailablePublishers(nf.subject, nf.grade)
        if (!cancelled) setAvailablePublishers(pubs)
      } catch {
        if (!cancelled) setAvailablePublishers([]) // 查询失败按"无大纲"处理，不阻断新建流程
      } finally {
        if (!cancelled) setPublishersLoading(false)
      }
    })()
    return () => { cancelled = true }
  }, [showNew, nf.subject, nf.grade])

  // ==================== 新建会话 ====================
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
        // v233：选了版本才传字段（含空串=通用版）；选"不关联"不传 → 后端落 NULL 不注入
        ...(nf.publisher !== PUBLISHER_NONE ? { course_outline_publisher: nf.publisher } : {}),
      })
      setPlan(r.plan)
      setMessages([{ role: 'assistant', content: r.opening, created_at: '' }])
      setShowNew(false); setView('session'); scrollBottom()
      // 进入新会话时清空课本状态
      setTextbookContext(''); setTextbookCount(0)
    } catch (e: any) { showToast(e?.message || '开始失败') }
    finally { setStarting(false) }
  }

  // ==================== 打开已有方案 ====================
  const openExisting = async (item: UnitPlanListItem) => {
    try {
      const r = await getUnitPlan(item.id)
      if (r.plan.status === 'draft') {
        setPlan(r.plan); setMessages(r.messages || []); setView('session'); scrollBottom()
        // 恢复草稿时清空课本状态（课本是会话级不落库）
        setTextbookContext(''); setTextbookCount(0)
      } else {
        setViewPlan(r.plan)
      }
    } catch (e: any) { showToast(e?.message || '打开失败') }
  }

  // ==================== 发送消息（含课本上下文注入） ====================
  const doSend = async () => {
    if (!plan || !input.trim() || sending) return
    const msg = input.trim(); setInput('')
    // 显示用户消息气泡（如有课本上下文，只在消息末尾加提示，不把OCR全文显示在气泡里）
    const displayMsg = textbookContext ? msg + '\n（已附课本原文参考）' : msg
    setMessages((m) => [...m, { role: 'user', content: displayMsg, created_at: '' }])
    setSending(true); scrollBottom()
    try {
      // 如有课本上下文，把OCR文字拼进发给AI的消息前面
      const aiMsg = textbookContext
        ? `【老师上传的教材原文参考】\n${textbookContext}\n\n【老师本轮说的话】\n${msg}`
        : msg
      const reply = await chatUnitPlan(plan.id, aiMsg)
      setMessages((m) => [...m, { role: 'assistant', content: reply, created_at: '' }])
    } catch (e: any) {
      setMessages((m) => [...m, { role: 'assistant', content: '（出错了：' + (e?.message || '请重试') + '）', created_at: '' }])
    } finally { setSending(false); scrollBottom() }
  }

  // ==================== 课本选择完成回调 ====================
  const handleTextbookSelected = async (selectedIds: string[]) => {
    setShowTextbookPicker(false)
    if (selectedIds.length === 0) {
      // 清空课本关联
      setTextbookContext(''); setTextbookCount(0)
      setMessages((m) => [...m, { role: 'assistant', content: '📷 已清除课本关联，之后的对话不再参考课本原文。', created_at: '' }])
      return
    }
    // 逐条取课本详情，拼出OCR文字
    showToast('正在读取课本文字…')
    const texts: string[] = []
    for (const id of selectedIds) {
      try {
        const detail = await getTextbook(id)
        if (detail.ocr_text) {
          texts.push(`--- 课本（${detail.textbook_name || ''}·${detail.chapter || ''}·第${detail.page_number || '?'}页）---\n${detail.ocr_text}`)
        } else {
          texts.push(`--- 课本（${detail.textbook_name || ''}·第${detail.page_number || '?'}页）---\n[此页尚未OCR识别，请先到课本管理页进行AI识别]`)
        }
      } catch {
        texts.push(`--- 课本（ID:${id}）---\n[读取失败]`)
      }
    }
    const ctx = texts.join('\n\n')
    setTextbookContext(ctx)
    setTextbookCount(selectedIds.length)
    setMessages((m) => [...m, {
      role: 'assistant',
      content: `📷 已关联 ${selectedIds.length} 张课本页。从你的下一条消息开始，我会参考这些课本原文内容。\n\n💡 提示：你可以直接说"确认"继续当前步骤，课本内容会自动附在你的消息里供我参考。`,
      created_at: '',
    }])
    scrollBottom()
  }

  // ==================== 保存 ====================
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
      setTextbookContext(''); setTextbookCount(0)
      showToast('已保存为正式方案'); loadList()
    } catch (e: any) { showToast(e?.message || '保存失败') }
    finally { setSaving(false) }
  }

  // ==================== 删除 ====================
  const doDelete = async (item: UnitPlanListItem, e: ReactMouseEvent) => {
    e.stopPropagation()
    if (!window.confirm('确认删除「' + item.title + '」？此操作不可恢复。')) return
    try { await deleteUnitPlan(item.id); showToast('已删除'); loadList() }
    catch (err: any) { showToast(err?.message || '删除失败') }
  }

  // ==================== 对话视图 ====================
  if (view === 'session' && plan) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 230px)', minHeight: 420 }}>
        {/* 顶栏 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, paddingBottom: 12, borderBottom: '1px solid ' + C.border, marginBottom: 12 }}>
          <button onClick={() => { setView('list'); loadList(); setTextbookContext(''); setTextbookCount(0) }} style={btnGhost}>← 返回列表</button>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 15, fontWeight: 700, color: C.textPrimary, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{plan.title}</div>
            <div style={{ fontSize: 12, color: C.textMuted }}>
              {plan.subject} · {plan.grade}{plan.volume} · {plan.unit}　🎓 大单元架构师逐步引导
              {plan.course_outline_publisher != null && (
                <span style={{ marginLeft: 8, color: '#7C3AED', fontWeight: 600 }}>📖 {publisherLabel(plan.course_outline_publisher)}</span>
              )}
              {textbookCount > 0 && <span style={{ marginLeft: 8, color: '#10B981', fontWeight: 600 }}>📷 课本×{textbookCount}</span>}
            </div>
          </div>
          <button onClick={() => setShowTextbookPicker(true)} style={btnTextbook}>📷 上传/选择课本</button>
          <button onClick={openSave} style={btnPrimary}>💾 保存为正式方案</button>
        </div>

        {/* 消息列表 */}
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

        {/* 课本关联提示条 */}
        {textbookCount > 0 && (
          <div style={{ padding: '6px 14px', background: '#ECFDF5', borderRadius: 8, fontSize: 12, color: '#059669', margin: '4px 0', display: 'flex', alignItems: 'center', gap: 8 }}>
            <span>📷 已关联 {textbookCount} 张课本页，每条消息会自动附课本原文供AI参考</span>
            <button onClick={() => { setTextbookContext(''); setTextbookCount(0); showToast('已清除课本关联') }}
              style={{ background: 'none', border: 'none', color: '#6B7280', cursor: 'pointer', fontSize: 12 }}>✕ 清除</button>
          </div>
        )}

        {/* 输入区 */}
        <div style={{ display: 'flex', gap: 10, paddingTop: 12, borderTop: '1px solid ' + C.border, marginTop: 8 }}>
          <textarea value={input} onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); doSend() } }}
            placeholder="确认 / 补充 / 让架构师按你的意见改这一步…（Enter 发送，Shift+Enter 换行）"
            rows={2} style={{ ...inputStyle, resize: 'vertical', flex: 1 }} />
          <button onClick={doSend} disabled={sending || !input.trim()} style={{ ...btnPrimary, opacity: sending || !input.trim() ? 0.5 : 1 }}>发送</button>
        </div>

        {/* 保存弹窗 */}
        {showSave && <SaveModal sf={sf} setSf={setSf} saving={saving} onCancel={() => setShowSave(false)} onSave={doSave} />}
        {/* 课本选择弹窗（复用课本管理已有的选择器） */}
        {showTextbookPicker && plan && (
          <TextbookPickerModal
            subject={plan.subject}
            grade={plan.grade}
            onConfirm={handleTextbookSelected}
            onCancel={() => setShowTextbookPicker(false)}
          />
        )}
        {toast && <Toast text={toast} />}
      </div>
    )
  }

  // ==================== 列表视图 ====================
  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
        <div style={{ fontSize: 13, color: C.textSecondary }}>
          大单元工坊产出的整单元教学设计方案。{canCreate ? '由学科负责人逐步生成，全组（或全局）可参考。' : '由学科负责人产出，你可在此查看本组/全局已发布的方案。'}
        </div>
        {canCreate && <button onClick={() => { setNf((s) => ({ ...s, unit: '', title: '', publisher: PUBLISHER_NONE })); setShowNew(true) }} style={btnPrimary}>＋ 新建单元方案</button>}
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

      {/* 新建弹窗 */}
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
            {/* v233：课程大纲教材版本选择（有该学科年级的大纲才显示下拉；与备课工坊首屏同款交互） */}
            <div>
              <label style={labelStyle}>课程大纲教材版本（可选）</label>
              {publishersLoading ? (
                <div style={{ fontSize: 12, color: C.textMuted, padding: '6px 0' }}>正在查询该学科年级可用的课程大纲…</div>
              ) : availablePublishers.length === 0 ? (
                <div style={{ fontSize: 12, color: C.textMuted, padding: '6px 0' }}>该学科年级暂无课程大纲，本次备课不关联大纲；如需大纲支撑请联系管理员上传。</div>
              ) : (
                <>
                  <select value={nf.publisher} onChange={(e) => setNf((s) => ({ ...s, publisher: e.target.value }))} style={inputStyle}>
                    <option value={PUBLISHER_NONE}>不关联课程大纲</option>
                    {availablePublishers.map((p) => <option key={p || '__generic__'} value={p}>{publisherLabel(p)}</option>)}
                  </select>
                  <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>
                    选定后 AI 将严格按该版本大纲定位本单元篇目与课时（会话建立时定版，中途不可改；换版请新建会话）。
                  </div>
                </>
              )}
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

      {/* 只读详情弹窗 */}
      {viewPlan && (
        <ModalShell title={viewPlan.title} wide onCancel={() => setViewPlan(null)}>
          <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 10 }}>
            {viewPlan.subject} · {viewPlan.grade}{viewPlan.volume} · {viewPlan.unit}
            {viewPlan.unit_theme ? '　主题：' + viewPlan.unit_theme : ''}
            {viewPlan.course_outline_publisher != null ? '　📖 大纲版本：' + publisherLabel(viewPlan.course_outline_publisher) : ''}
          </div>
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

// ==================== 课本选择弹窗（轻量版，复用课本API） ====================
function TextbookPickerModal({ subject, grade, onConfirm, onCancel }: {
  subject: string; grade: string; onConfirm: (ids: string[]) => void; onCancel: () => void
}) {
  const [pages, setPages] = useState<TextbookListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Set<string>>(new Set())

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const resp = await getTextbooks({ subject, grade_range: grade, limit: 200 })
        if (!cancelled) setPages(resp.pages || [])
      } catch { /* 静默 */ }
      finally { if (!cancelled) setLoading(false) }
    })()
    return () => { cancelled = true }
  }, [subject, grade])

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }

  return (
    <ModalShell title={`选择课本页（${subject} · ${grade}）`} wide onCancel={onCancel}>
      <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 10 }}>
        选择要参考的课本页面，AI会读取其中的文字内容。如需上传新课本图片，请先到「课本管理」页面上传并进行AI识别。
      </div>
      <div style={{ maxHeight: '50vh', overflowY: 'auto', border: '1px solid ' + C.border, borderRadius: 8, padding: 8 }}>
        {loading ? (
          <div style={{ padding: 30, textAlign: 'center', color: C.textMuted, fontSize: 13 }}>加载中…</div>
        ) : pages.length === 0 ? (
          <div style={{ padding: 30, textAlign: 'center', color: C.textMuted, fontSize: 13 }}>暂无该学科年级的课本图片，请先到课本管理页面上传。</div>
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: 8 }}>
            {pages.map((p) => {
              const isSelected = selected.has(p.id)
              return (
                <div key={p.id} onClick={() => toggle(p.id)} style={{
                  padding: '8px', borderRadius: 8, cursor: 'pointer', textAlign: 'center',
                  border: isSelected ? '2px solid #10B981' : '1px solid ' + C.border,
                  background: isSelected ? '#ECFDF5' : C.white,
                }}>
                  <img src={p.image_url} alt="" style={{ width: '100%', height: 80, objectFit: 'cover', borderRadius: 6, marginBottom: 4 }} />
                  <div style={{ fontSize: 11, color: C.textPrimary, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {p.textbook_name}
                  </div>
                  <div style={{ fontSize: 10, color: C.textMuted }}>
                    第{p.page_number}页 {p.has_ocr ? '✅已识别' : '⚠️未识别'}
                  </div>
                  {isSelected && <div style={{ fontSize: 11, color: '#10B981', fontWeight: 700 }}>✓ 已选</div>}
                </div>
              )
            })}
          </div>
        )}
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 14 }}>
        <span style={{ fontSize: 12, color: C.textSecondary }}>已选 {selected.size} 页</span>
        <div style={{ display: 'flex', gap: 10 }}>
          <button onClick={onCancel} style={btnGhost}>取消</button>
          <button onClick={() => onConfirm(Array.from(selected))} style={{ ...btnPrimary, background: '#10B981' }}>
            {selected.size > 0 ? `确认关联 ${selected.size} 页` : '清除关联'}
          </button>
        </div>
      </div>
    </ModalShell>
  )
}

// ==================== 通用弹窗外壳 ====================
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

// ==================== 保存弹窗 ====================
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

// ==================== Toast 提示 ====================
function Toast({ text }: { text: string }) {
  return (
    <div style={{ position: 'fixed', bottom: 28, left: '50%', transform: 'translateX(-50%)', background: 'rgba(17,24,39,0.92)', color: '#fff', padding: '10px 20px', borderRadius: 10, fontSize: 13, zIndex: 9999, boxShadow: '0 6px 24px rgba(0,0,0,0.2)' }}>{text}</div>
  )
}
