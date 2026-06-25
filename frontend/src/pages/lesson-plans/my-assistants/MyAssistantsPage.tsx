/**
 * MyAssistantsPage.tsx — 我的 AI 助手 / 提示词工坊主页面(阶段A · 对话画布版)
 *
 * ════════════════════════════════════════════════════════════════════════
 * 设计理念(对齐 Yuhan 拍板 — 顶级 Harness):
 *   这个页面就是一块"对话画布"。老师造助手时,注意力全在对话上,
 *   其他一切(现成助手列表、草稿)都是"可召唤、用完即收"的辅助,不跟对话抢地盘。
 *   - 对话区占满主体(designer fillHeight + collapsibleDraft)
 *   - 现成助手收进右侧"侧滑抽屉",默认关闭,点按钮才滑出
 *   - 草稿可一键收起,让对话独占全宽
 * ════════════════════════════════════════════════════════════════════════
 *
 * ════════════ 三种造助手的路径(都汇到 SaveAssistantModal 落库) ════════════
 *   1. 跟 AI 聊着造        —— 对话画布主流程
 *   2. 挑现成助手"丢给 AI 分析" —— 抽屉里取 full_prompt 注入对话,让 AI 帮想还能补什么
 *   3. 粘贴已有提示词      —— 「✍️ 粘贴提示词」,可直接存 或 丢给 AI 润色
 *
 * ════════════ 身份分层 + 标杆化(沿用) ════════════
 *   - 顶部按角色一句话提示(老师/教研员/admin 不同)
 *   - 存助手货架选择在 SaveAssistantModal(老师无选项保持极简)
 *   - 抽屉里"🏫 本校推荐"置顶强调;普通老师且本校有 group 助手时,顶部给温和提示条
 *
 * 复用既有资产:
 *   - AssistantDesignerPanel(injectedInput/fillHeight/collapsibleDraft)
 *   - SaveAssistantModal(userRole 货架选择)
 *   - PastePromptModal(粘贴入口,双出口)
 *   - listAssistants / getAssistant / forkAssistant
 *   - useAuth(取角色)
 *
 * 职责边界:本页只造助手/管助手;"用助手"在备课工坊。故抽屉卡片不放"用这个→跳工坊"。
 */

import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '@/store/auth'
import {
  listAssistants,
  getAssistant,
  forkAssistant,
  ASSISTANT_SOURCE_LABELS,
  ASSISTANT_SOURCE_EMOJI,
  type AIAssistantListItem,
  type AssistantScene,
  type AssistantSource,
} from '@/api/ai-assistants'
import AssistantDesignerPanel from '@/components/ai-assistants/AssistantDesignerPanel'
import SaveAssistantModal from '@/components/ai-assistants/SaveAssistantModal'
import PastePromptModal from '@/components/ai-assistants/PastePromptModal'

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
  systemAccent:   '#4F7BE8',
  groupAccent:    '#F59E0B',
  personalAccent: '#10B981',
}

/** localStorage key:记住老师的主教学科 */
const LS_KEY = 'tedna_my_assistants_subject'

/** 学科可选项(与 SaveAssistantModal/AssistantEditModal 保持一致) */
const SUBJECTS = [
  '', // 空=不限
  '人工智能', '语文', '数学', '英语', '物理', '化学', '生物',
  '历史', '地理', '政治', '科学', '信息科技', '技术', '综合实践',
]

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

/* ==================== 子组件:抽屉里单张"现成助手"紧凑卡片 ==================== */

interface DrawerCardProps {
  item: AIAssistantListItem
  forking: boolean
  analyzing: boolean
  highlight?: boolean
  onAnalyze: () => void
  onFork: () => void
}

function DrawerAssistantCard({ item, forking, analyzing, highlight, onAnalyze, onFork }: DrawerCardProps) {
  const [expanded, setExpanded] = useState(false)
  const accent =
    item.source === 'system'   ? C.systemAccent :
    item.source === 'group'    ? C.groupAccent  :
                                 C.personalAccent

  return (
    <div style={{
      padding: '9px 11px', borderRadius: '8px',
      border: `1px solid ${highlight ? `${accent}66` : C.border}`,
      borderLeft: `3px solid ${accent}`,
      background: highlight ? `${accent}0D` : '#fff',
      marginBottom: '7px',
    }}>
      {/* 标题行(紧凑:名字 + 来源徽章) */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '5px', flexWrap: 'wrap' }}>
        <span style={{ fontSize: '12.5px', fontWeight: 600, color: C.text }}>
          {item.avatar_emoji} {item.name}
        </span>
        <span style={{
          padding: '1px 5px', borderRadius: '4px', fontSize: '9.5px', fontWeight: 600,
          background: `${accent}1A`, color: accent,
        }}>
          {ASSISTANT_SOURCE_EMOJI[item.source]} {ASSISTANT_SOURCE_LABELS[item.source]}
        </span>
        {item.subject && (
          <span style={{ fontSize: '9.5px', color: C.textMuted }}>📚 {item.subject}</span>
        )}
      </div>

      {/* 描述(默认折叠成1行,点展开看全) */}
      {item.description && (
        <div
          style={{
            fontSize: '11px', color: C.textSec, marginTop: '4px', lineHeight: 1.55,
            overflow: 'hidden',
            display: '-webkit-box',
            WebkitLineClamp: expanded ? 99 : 1,
            WebkitBoxOrient: 'vertical' as const,
            cursor: 'pointer',
          }}
          onClick={() => setExpanded(v => !v)}
          title={expanded ? '点击收起' : '点击展开'}
        >
          {item.description}
        </div>
      )}

      {/* 操作行 */}
      <div style={{ display: 'flex', gap: '6px', marginTop: '7px', flexWrap: 'wrap' }}>
        <button
          onClick={onAnalyze}
          disabled={analyzing}
          style={miniBtn(C.accent, analyzing)}
          title="把这个助手的完整设定丢给左侧 AI,让它帮你分析、讨论你可以从哪些角度补充"
        >{analyzing ? '读取中…' : '🔍 丢给 AI 分析'}</button>
        {/* 系统/本校助手可复制到我的;个人助手已是自己的 */}
        {item.source !== 'personal' ? (
          <button
            onClick={onFork}
            disabled={forking}
            style={miniBtn(C.primary, forking)}
          >{forking ? '复制中…' : '➕ 复制到我的'}</button>
        ) : (
          <span style={{ fontSize: '10px', color: C.textMuted, alignSelf: 'center' }}>
            ✓ 已是你的
          </span>
        )}
      </div>
    </div>
  )
}

/** 小按钮样式 */
function miniBtn(color: string, disabled?: boolean): React.CSSProperties {
  return {
    padding: '3px 9px', borderRadius: '5px',
    border: `1px solid ${color}`, background: '#fff', color,
    fontSize: '11px', fontWeight: 500,
    cursor: disabled ? 'not-allowed' : 'pointer',
    opacity: disabled ? 0.5 : 1,
    whiteSpace: 'nowrap',
  }
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

  // ==================== 抽屉开关 ====================
  const [drawerOpen, setDrawerOpen] = useState(false)

  // ==================== 注入 designer 输入框的文字 ====================
  const [injected, setInjected] = useState('')

  // ==================== 存助手弹窗 ====================
  const [saveOpen, setSaveOpen]       = useState(false)
  const [draftToSave, setDraftToSave] = useState('')

  // ==================== 粘贴提示词弹窗 ====================
  const [pasteOpen, setPasteOpen] = useState(false)

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
      {groupCount > 0 && role !== 'senior_operator' && role !== 'admin' && (
        <div style={{
          padding: '9px 14px', borderRadius: '10px',
          background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.25)',
          color: '#92400E', fontSize: '12.5px', lineHeight: 1.6, flexShrink: 0,
        }}>
          💡 你们教研组已沉淀 <b>{groupCount}</b> 个「本校推荐」助手。造之前不妨点右上角「现成助手」看看——
          能直接用、复制来改,或<b>丢给 AI 帮你分析还能补充什么</b>。
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
          挑一个现成的助手,可<b style={{ color: C.accent }}>🔍 丢给左侧 AI 分析</b>(让它帮你想还能补什么)、
          <b style={{ color: C.primary }}>➕ 复制到我的</b>再改,或点描述展开细看。
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
                        highlight={isBenchmark}
                        onAnalyze={() => handleAnalyze(item)}
                        onFork={() => handleFork(item)}
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
