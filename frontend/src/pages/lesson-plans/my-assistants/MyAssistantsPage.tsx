/**
 * MyAssistantsPage.tsx — 我的 AI 助手 / 提示词工坊主页面(阶段A · 对话画布版)
 *
 * 设计理念(Yuhan 拍板):页面是一块"对话画布"。造助手时注意力全在对话上,
 *   现成助手收进右侧侧滑抽屉(默认关,点按钮滑出),草稿可一键收起让对话独占全宽。
 *
 * 三种造助手路径(都汇到 SaveAssistantModal 落库):
 *   1. 跟 AI 聊着造(对话画布主流程)
 *   2. 挑现成助手"丢给 AI 分析"(抽屉取 full_prompt 注入对话)
 *   3. 粘贴已有提示词(PastePromptModal,可直接存或丢给 AI 润色)
 *
 * 助手管理(本次新增,配合 share_policy 全套):
 *   抽屉卡片 DrawerAssistantCard 现支持 ✏️编辑 / 🗑️删除 / ➕复制(按 can_fork 显隐):
 *     - 编辑:打开 AssistantEditModal(edit 模式),保存后刷新列表
 *     - 删除:window.confirm 二次确认后调 deleteAssistant,刷新列表
 *     - 复制:按后端下发的 can_fork 显隐(作者设 use_only/locked 则隐藏)
 *   can_edit/can_delete/can_fork 全由后端按当前用户算好,前端只按布尔显隐,最终拦截在后端。
 *
 * 重构:DrawerAssistantCard + miniBtn 抽至 ./DrawerAssistantCard.tsx,控制本文件行数。
 *
 * 职责边界:本页只造助手/管助手;"用助手"在备课工坊。故抽屉卡片不放"用这个→跳工坊"。
 */

import { useState, useEffect, useCallback } from 'react'
import { DEFAULT_SUBJECTS } from '@/constants/subjects'
import { useAuth } from '@/store/auth'
import {
  listAssistants,
  getAssistant,
  forkAssistant,
  deleteAssistant,
  ASSISTANT_SOURCE_LABELS,
  ASSISTANT_SOURCE_EMOJI,
  type AIAssistantListItem,
  type AssistantScene,
  type AssistantSource,
} from '@/api/ai-assistants'
import AssistantDesignerPanel from '@/components/ai-assistants/AssistantDesignerPanel'
import SaveAssistantModal from '@/components/ai-assistants/SaveAssistantModal'
import PastePromptModal from '@/components/ai-assistants/PastePromptModal'
import AssistantEditModal from '@/components/ai-assistants/AssistantEditModal'
import DrawerAssistantCard, { miniBtn } from './DrawerAssistantCard'

/* ==================== 样式常量(与助手系列组件保持一致) ==================== */
const C = {
  primary:        '#4F7BE8',
  primaryLight:   'rgba(79,123,232,0.08)',
  accent:         '#F59E0B',
  success:        '#10B981',
  danger:         '#EF4444',
  text:           '#1F2937',
  textSec:        '#6B7280',
  textMuted:      '#9CA3AF',
  bg:             '#FAFBFC',
  card:           '#FFFFFF',
  border:         '#F3F4F6',
  borderMid:      '#E5E7EB',
  groupAccent:    '#F59E0B',
}

/** localStorage key:记住老师的主教学科 */
const LS_KEY = 'tedna_my_assistants_subject'

/** 学科可选项(与 SaveAssistantModal/AssistantEditModal 保持一致) */
const SUBJECTS = ['', ...DEFAULT_SUBJECTS]  // 空=不限；单一真相源（方案甲，v231）

/** 工坊全部场景(透传给 designer 作默认,让 AI 按全链路推断组件) */
const WORKSHOP_SCENES: AssistantScene[] = [
  'workshop_analyze', 'workshop_design', 'workshop_write', 'workshop_review', 'workshop_revise',
]

/** 按角色返回顶部一句话身份提示 */
function roleHint(role: string | undefined): string {
  if (role === 'admin') {
    return '和 AI 聊一聊就能造助手。你可以存为自己用,也可以发布为本校推荐或全平台系统助手。'
  }
  if (role === 'senior_operator') {
    return '和 AI 聊一聊就能造助手。你可以只给自己用,也可以发布为「本校推荐」,给全校老师定标杆。'
  }
  return '和 AI 聊一聊,就能造出懂你的备课助手。想参考现成的,点右上角「现成助手」。'
}

/* ==================== 主页面 ==================== */

export default function MyAssistantsPage() {
  // 当前用户(取角色)
  const { user } = useAuth()
  const role = user?.role

  // ==================== 学科状态(localStorage 记忆) ====================
  const [subject, setSubject] = useState<string>(() => {
    try { return localStorage.getItem(LS_KEY) || '' } catch { return '' }
  })
  useEffect(() => {
    try { localStorage.setItem(LS_KEY, subject) } catch { /* 忽略写入失败 */ }
  }, [subject])

  // ==================== 现成助手列表 ====================
  const [related, setRelated]     = useState<AIAssistantListItem[]>([])
  const [loading, setLoading]     = useState(false)
  const [listErr, setListErr]     = useState<string | null>(null)
  const [forkingId, setForkingId]   = useState<string | null>(null)
  const [analyzingId, setAnalyzingId] = useState<string | null>(null)
  const [deletingId, setDeletingId]   = useState<string | null>(null)

  // ==================== 抽屉开关 ====================
  const [drawerOpen, setDrawerOpen] = useState(false)

  // ==================== 注入 designer 输入框的文字 ====================
  const [injected, setInjected] = useState('')

  // ==================== 存助手弹窗 ====================
  const [saveOpen, setSaveOpen]       = useState(false)
  const [draftToSave, setDraftToSave] = useState('')

  // ==================== 粘贴提示词弹窗 ====================
  const [pasteOpen, setPasteOpen] = useState(false)

  // ==================== 编辑助手弹窗(本次新增) ====================
  const [editId, setEditId] = useState<string | null>(null)

  // ==================== 顶部横幅 ====================
  const [banner, setBanner] = useState<string | null>(null)

  // ==================== 加载现成助手 ====================
  const loadRelated = useCallback(async () => {
    setLoading(true); setListErr(null)
    try {
      const resp = await listAssistants({ subject: subject || undefined })
      setRelated(resp.assistants || [])
    } catch (e) {
      setListErr(e instanceof Error ? e.message : '加载助手列表失败')
      setRelated([])
    } finally {
      setLoading(false)
    }
  }, [subject])

  useEffect(() => { loadRelated() }, [loadRelated])

  // 横幅 5 秒自动消失
  useEffect(() => {
    if (!banner) return
    const t = setTimeout(() => setBanner(null), 5000)
    return () => clearTimeout(t)
  }, [banner])

  // ==================== 注入文字到对话画布(两步设值,保证每次都触发 designer 的 useEffect) ====================
  const injectToCanvas = useCallback((text: string) => {
    setInjected('')
    setTimeout(() => setInjected(text), 30)
  }, [])

  // ==================== designer "存为我的助手"回调 ====================
  const handleApplyDraft = useCallback((draft: string) => {
    setDraftToSave(draft)
    setSaveOpen(true)
  }, [])

  // ==================== 存助手成功(按 source 差异化提示) ====================
  const handleSaved = useCallback((_id: string, source: AssistantSource) => {
    setSaveOpen(false)
    if (source === 'group') {
      setBanner('✅ 已发布为「本校推荐」助手,全校老师备课时都能选用它了')
    } else if (source === 'system') {
      setBanner('✅ 已发布为「系统」助手,全平台老师都能选用它了')
    } else {
      setBanner('✅ 助手已保存,备课时可在工坊里选用它')
    }
    loadRelated()
  }, [loadRelated])

  // ==================== 复制现成助手到我的 ====================
  const handleFork = useCallback(async (item: AIAssistantListItem) => {
    if (forkingId) return
    setForkingId(item.id)
    try {
      const forked = await forkAssistant(item.id)
      setBanner(`✅ 已复制为「${forked.name}」,可在工坊选用,或在左侧继续让 AI 帮你改`)
      await loadRelated()
    } catch (e) {
      setBanner(`⚠️ 复制失败:${e instanceof Error ? e.message : '请重试'}`)
    } finally {
      setForkingId(null)
    }
  }, [forkingId, loadRelated])

  // ==================== 把助手"丢给 AI 分析" ====================
  const handleAnalyze = useCallback(async (item: AIAssistantListItem) => {
    if (analyzingId) return
    setAnalyzingId(item.id)
    try {
      const full = await getAssistant(item.id)
      // 产权保护防御:若后端判定当前用户无权看原文(prompt_protected),full_prompt 已被置空。
      //   正常情况下 can_view_prompt=false 时分析按钮已隐藏,不会走到这里;此处为双保险,
      //   避免把空 prompt 丢给 AI,改为友好提示。
      if (full.prompt_protected) {
        setBanner('⚠️ 作者把「' + item.name + '」设为「仅可用」,未开放提示词原文,无法丢给 AI 分析。你仍可在备课工坊直接使用它。')
        return
      }
      const prompt = full.full_prompt || '(这个助手没有可读的设定内容)'
      const guide =
        `我想参考「${item.name}」这个助手,造一个类似的。\n\n` +
        `它的完整设定是:\n${prompt}\n\n` +
        `请帮我分析它的设计思路,并和我讨论:基于我的需求,我可以从哪些角度补充或改进?`
      setDrawerOpen(false)
      injectToCanvas(guide)
    } catch (e) {
      setBanner(`⚠️ 读取助手失败:${e instanceof Error ? e.message : '请重试'}`)
    } finally {
      setAnalyzingId(null)
    }
  }, [analyzingId, injectToCanvas])

  // ==================== 编辑助手:打开编辑弹窗 ====================
  const handleEdit = useCallback((item: AIAssistantListItem) => {
    setEditId(item.id)
  }, [])

  // ==================== 编辑保存成功:关弹窗 + 刷新 ====================
  const handleEditSaved = useCallback(() => {
    setEditId(null)
    setBanner('✅ 助手已更新')
    loadRelated()
  }, [loadRelated])

  // ==================== 删除助手:二次确认后删除 ====================
  const handleDelete = useCallback(async (item: AIAssistantListItem) => {
    if (deletingId) return
    const ok = window.confirm(`确定删除助手「${item.name}」吗?\n此操作不可恢复。`)
    if (!ok) return
    setDeletingId(item.id)
    try {
      await deleteAssistant(item.id)
      setBanner(`✅ 已删除「${item.name}」`)
      await loadRelated()
    } catch (e) {
      setBanner(`⚠️ 删除失败:${e instanceof Error ? e.message : '请重试'}`)
    } finally {
      setDeletingId(null)
    }
  }, [deletingId, loadRelated])

  // ==================== 粘贴提示词:出口1 直接保存 ====================
  const handlePasteSaveDirect = useCallback((text: string) => {
    setPasteOpen(false)
    setDraftToSave(text)
    setSaveOpen(true)
  }, [])

  // ==================== 粘贴提示词:出口2 丢给 AI 润色 ====================
  const handlePasteSendToAI = useCallback((text: string) => {
    setPasteOpen(false)
    const guide =
      `这是我已经写好的一段提示词,请帮我审阅并润色,让它更清晰、更适合做备课助手:\n\n${text}\n\n` +
      `请指出可以改进的地方,我们一起把它打磨好。`
    injectToCanvas(guide)
  }, [injectToCanvas])

  // ==================== 分组 ====================
  const grouped: Record<AssistantSource, AIAssistantListItem[]> = { system: [], group: [], personal: [] }
  for (const a of related) grouped[a.source].push(a)
  const groupCount = grouped.group.length
  const totalCount = related.length

  // 本校推荐助手里,当前用户实际能做什么(供提示条/抽屉说明按权限动态措辞):
  //   groupAnalyzable=有能"丢给 AI 分析"的(can_view_prompt,即能看原文)
  //   groupForkable  =有能"复制到我的"的(can_fork)
  //   都为 false 说明这批助手对当前用户都是"仅可用",措辞应改为"备课时直接选用"而非"复制来改"
  const groupAnalyzable = grouped.group.some(a => a.can_view_prompt)
  const groupForkable   = grouped.group.some(a => a.can_fork)

  // 抽屉内分组顺序:本校推荐(标杆)置顶 → 我的 → 系统
  const DISPLAY_ORDER: AssistantSource[] = ['group', 'personal', 'system']

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', height: '100%', position: 'relative' }}>
      {/* ==================== 顶栏 ==================== */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: '14px', flexWrap: 'wrap',
        padding: '12px 16px', background: C.card, borderRadius: '12px',
        border: `1px solid ${C.border}`, flexShrink: 0,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>我主要教</span>
          <select
            value={subject}
            onChange={e => setSubject(e.target.value)}
            style={{
              padding: '6px 11px', borderRadius: '8px', border: `1px solid ${C.borderMid}`,
              fontSize: '14px', fontWeight: 600, color: C.primary, background: C.primaryLight,
              cursor: 'pointer', outline: 'none',
            }}
          >
            {SUBJECTS.map(s => <option key={s} value={s}>{s || '(全部学科)'}</option>)}
          </select>
        </div>
        <div style={{ flex: 1, minWidth: '160px', fontSize: '12px', color: C.textSec, lineHeight: 1.6 }}>
          {roleHint(role)}
        </div>
        {/* 粘贴提示词入口 */}
        <button
          onClick={() => setPasteOpen(true)}
          style={{
            display: 'flex', alignItems: 'center', gap: '6px',
            padding: '8px 14px', borderRadius: '9px',
            border: `1px solid ${C.borderMid}`, background: '#fff', color: C.textSec,
            fontSize: '13px', fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap', flexShrink: 0,
          }}
          title="已有现成的提示词?粘进来直接存,或交给 AI 润色"
        >
          ✍️ 粘贴提示词
        </button>
        {/* 召唤抽屉按钮 */}
        <button
          onClick={() => setDrawerOpen(true)}
          style={{
            display: 'flex', alignItems: 'center', gap: '6px',
            padding: '8px 14px', borderRadius: '9px',
            border: `1px solid ${C.primary}`, background: C.primaryLight, color: C.primary,
            fontSize: '13px', fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap', flexShrink: 0,
          }}
          title="查看现成的助手,可参考、复制,或丢给 AI 分析"
        >
          📚 现成助手{totalCount > 0 ? ` (${totalCount})` : ''}
        </button>
      </div>

      {/* 标杆提示条:仅普通老师 + 本校有 group 助手时显示 */}
      {/*   文案随当前用户对这批助手的实际权限动态变化,避免"怂恿用户做现在做不了的事": */}
      {/*     有能复制/分析的 → 提"复制来改、丢给 AI 分析";都只能用 → 提"备课时直接选用" */}
      {groupCount > 0 && role !== 'senior_operator' && role !== 'admin' && (
        <div style={{
          padding: '9px 14px', borderRadius: '10px',
          background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.25)',
          color: '#92400E', fontSize: '12.5px', lineHeight: 1.6, flexShrink: 0,
        }}>
          {(groupForkable || groupAnalyzable) ? (
            <>
              💡 你们教研组已沉淀 <b>{groupCount}</b> 个「本校推荐」助手。造之前不妨点右上角「现成助手」看看——
              {groupForkable && '能直接用、复制来改,'}
              {groupAnalyzable && <>或<b>丢给 AI 帮你分析还能补充什么</b></>}。
            </>
          ) : (
            <>
              💡 你们教研组已准备了 <b>{groupCount}</b> 个「本校推荐」助手,
              <b>备课时可以直接选用</b>。点右上角「现成助手」可以先看看它们。
            </>
          )}
        </div>
      )}

      {/* 成功/提示横幅 */}
      {banner && (
        <div style={{
          padding: '9px 14px', borderRadius: '10px',
          background: banner.startsWith('⚠️') ? 'rgba(239,68,68,0.06)' : 'rgba(16,185,129,0.08)',
          border: `1px solid ${banner.startsWith('⚠️') ? 'rgba(239,68,68,0.2)' : 'rgba(16,185,129,0.25)'}`,
          color: banner.startsWith('⚠️') ? C.danger : '#047857',
          fontSize: '13px', fontWeight: 500, flexShrink: 0,
        }}>
          {banner}
        </div>
      )}

      {/* ==================== 对话画布(占满) ==================== */}
      <div style={{
        flex: 1, minHeight: 0,
        background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`,
        padding: '14px', display: 'flex', flexDirection: 'column',
      }}>
        <div style={{ flex: 1, minHeight: 0 }}>
          <AssistantDesignerPanel
            subject={subject}
            grade=""
            scenes={WORKSHOP_SCENES}
            initialDraft=""
            onApplyDraft={handleApplyDraft}
            applyButtonLabel="✓ 存为我的助手"
            applyHintText="保存后可在工坊选用,也会出现在「现成助手」里"
            injectedInput={injected}
            fillHeight
            collapsibleDraft
          />
        </div>
      </div>

      {/* ==================== 侧滑抽屉:现成助手 ==================== */}
      <div
        onClick={() => setDrawerOpen(false)}
        style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(17,24,39,0.35)',
          opacity: drawerOpen ? 1 : 0,
          pointerEvents: drawerOpen ? 'auto' : 'none',
          transition: 'opacity 200ms ease',
          zIndex: 9000,
        }}
      />
      <div style={{
        position: 'fixed', top: 0, right: 0, bottom: 0,
        width: '380px', maxWidth: '90vw',
        background: C.card,
        boxShadow: '-8px 0 32px rgba(0,0,0,0.12)',
        transform: drawerOpen ? 'translateX(0)' : 'translateX(100%)',
        transition: 'transform 260ms cubic-bezier(0.4,0,0.2,1)',
        zIndex: 9001,
        display: 'flex', flexDirection: 'column',
      }}>
        <div style={{
          padding: '16px 18px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0,
        }}>
          <span style={{ fontSize: '14px', fontWeight: 700, color: C.text }}>
            📚 现成的助手{subject ? `（${subject}）` : ''}
          </span>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <button onClick={loadRelated} title="刷新"
              style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '13px', color: C.primary }}>🔄</button>
            <button onClick={() => setDrawerOpen(false)} title="关闭"
              style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted, lineHeight: 1 }}>×</button>
          </div>
        </div>

        <div style={{
          padding: '10px 18px', fontSize: '11px', color: C.textSec, lineHeight: 1.6,
          background: C.bg, borderBottom: `1px solid ${C.border}`, flexShrink: 0,
        }}>
          这里是可以参考、选用的助手。每个助手下方会列出可用的操作,备课时也能直接在工坊里选用它们。点描述可展开细看。
        </div>

        <div style={{ flex: 1, overflow: 'auto', padding: '12px 14px' }}>
          {loading && (
            <div style={{ padding: '30px 0', textAlign: 'center', color: C.textMuted, fontSize: '12px' }}>加载中…</div>
          )}

          {listErr && !loading && (
            <div style={{
              padding: '12px', borderRadius: '8px', textAlign: 'center',
              background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.15)',
              color: C.danger, fontSize: '12px',
            }}>
              ⚠️ {listErr}
              <br />
              <button onClick={loadRelated} style={{ marginTop: '6px', ...miniBtn(C.danger) }}>重试</button>
            </div>
          )}

          {!loading && !listErr && totalCount === 0 && (
            <div style={{ padding: '36px 16px', textAlign: 'center', color: C.textMuted, fontSize: '12px', lineHeight: 1.7 }}>
              <div style={{ fontSize: '30px', marginBottom: '8px' }}>🗂️</div>
              {subject ? `「${subject}」暂无现成助手` : '暂无现成助手'}
              <br />
              关掉这里,在左侧和 AI 聊一聊,造一个属于你的吧
            </div>
          )}

          {!loading && !listErr && totalCount > 0 && (
            <>
              {DISPLAY_ORDER.map(src => {
                const items = grouped[src]
                if (items.length === 0) return null
                const isBenchmark = src === 'group'
                return (
                  <div key={src} style={{ marginBottom: '14px' }}>
                    <div style={{
                      padding: '2px 2px 6px', fontSize: '11px', fontWeight: 700,
                      color: isBenchmark ? C.groupAccent : C.textSec,
                    }}>
                      {ASSISTANT_SOURCE_EMOJI[src]} {ASSISTANT_SOURCE_LABELS[src]}
                      <span style={{ color: C.textMuted, fontWeight: 400 }}> ({items.length})</span>
                      {isBenchmark && (
                        <span style={{ color: C.textMuted, fontWeight: 400 }}> · 教研组推荐,建议优先参考</span>
                      )}
                    </div>
                    {items.map(item => (
                      <DrawerAssistantCard
                        key={item.id}
                        item={item}
                        forking={forkingId === item.id}
                        analyzing={analyzingId === item.id}
                        deleting={deletingId === item.id}
                        highlight={isBenchmark}
                        onAnalyze={() => handleAnalyze(item)}
                        onFork={() => handleFork(item)}
                        onEdit={() => handleEdit(item)}
                        onDelete={() => handleDelete(item)}
                      />
                    ))}
                  </div>
                )
              })}
            </>
          )}
        </div>
      </div>

      {/* ==================== 存为助手确认弹窗 ==================== */}
      <SaveAssistantModal
        open={saveOpen}
        draft={draftToSave}
        userRole={role}
        defaultSubject={subject}
        onClose={() => setSaveOpen(false)}
        onSaved={handleSaved}
      />

      {/* ==================== 编辑助手弹窗 ==================== */}
      <AssistantEditModal
        open={editId !== null}
        mode="edit"
        assistantId={editId || undefined}
        onClose={() => setEditId(null)}
        onSaved={handleEditSaved}
      />

      {/* ==================== 粘贴提示词弹窗 ==================== */}
      <PastePromptModal
        open={pasteOpen}
        onClose={() => setPasteOpen(false)}
        onSaveDirect={handlePasteSaveDirect}
        onSendToAI={handlePasteSendToAI}
      />
    </div>
  )
}
