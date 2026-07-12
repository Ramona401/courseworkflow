/**
 * SnippetInjectPicker.tsx — 【批次C新增】微调参考代码注入选择器
 *
 * 挂在 RefinePanel 工具条上：老师点「📎 注入参考代码」→ 弹层列出我的代码收藏
 * （listCodeSnippets，轻量列表）→ 点选某条 → getCodeSnippet 按 id 拉完整HTML
 * → 经 onInject 交给 RefinePanel 暂存；提交微调时由 RefinePanel 把参考代码
 * 以标记块形式拼进微调指令（前端拼接注入方案——不动后端 RefinePage 签名，
 * 老师提交前"注入了什么"完全可见，透明可控）。
 *
 * 已注入时按钮位置显示芯片「📎 已注入：xxx ✕」，点 ✕ 移除。
 * 弹层内每条收藏可「🗑」删除（二次确认）；列表懒加载（首次展开才拉取）。
 * 抽为独立组件的原因：RefinePanel 已近 600 行上限，注入选择器约 200 行，
 * 独立成文件符合模块化规则，RefinePanel 仅需约 40 行挂载代码。
 */
import { useState } from 'react'
import { listCodeSnippets, getCodeSnippet, deleteCodeSnippet } from '@/api/coursewares'
import type { CodeSnippetListItem } from '@/api/coursewares'
import { C } from './workshopConstants'

/** 已注入的参考代码（RefinePanel 暂存并在提交时拼接） */
export interface InjectedSnippet {
  id: string
  title: string
  html: string
}

interface Props {
  /** 当前已注入的收藏（null=未注入） */
  injected: InjectedSnippet | null
  /** 选中某条收藏并拉到全文后回调 */
  onInject: (s: InjectedSnippet) => void
  /** 移除已注入 */
  onRemove: () => void
  /** 微调/重生运行中禁用 */
  disabled: boolean
}

export default function SnippetInjectPicker({ injected, onInject, onRemove, disabled }: Props) {
  const [open, setOpen] = useState(false)            // 弹层开关
  const [items, setItems] = useState<CodeSnippetListItem[]>([]) // 收藏列表（轻量）
  const [loading, setLoading] = useState(false)      // 列表加载中
  const [fetchingId, setFetchingId] = useState('')   // 正在拉全文的收藏id（禁用对应行）
  const [deletingId, setDeletingId] = useState('')   // 正在删除的收藏id
  const [msg, setMsg] = useState('')                 // 弹层内提示

  /** 展开/收起弹层：展开时懒加载列表 */
  const handleToggle = async () => {
    if (disabled) return
    const next = !open
    setOpen(next)
    setMsg('')
    if (next) {
      setLoading(true)
      try {
        const r = await listCodeSnippets()
        setItems(r.snippets)
      } catch (e) {
        setMsg('❌ 加载收藏失败: ' + (e instanceof Error ? e.message : '未知错误'))
      } finally { setLoading(false) }
    }
  }

  /** 选中一条：按 id 拉完整HTML，交给父级暂存 */
  const handlePick = async (it: CodeSnippetListItem) => {
    if (fetchingId || deletingId) return
    setFetchingId(it.id)
    setMsg('')
    try {
      const detail = await getCodeSnippet(it.id)
      onInject({ id: detail.id, title: detail.title, html: detail.html_content })
      setOpen(false)
    } catch (e) {
      setMsg('❌ 获取收藏内容失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setFetchingId('') }
  }

  /** 删除一条收藏（二次确认；若删的正是当前已注入的，同步移除注入） */
  const handleDelete = async (e: React.MouseEvent, it: CodeSnippetListItem) => {
    e.stopPropagation()
    if (fetchingId || deletingId) return
    if (!window.confirm('确定删除收藏「' + it.title + '」？此操作不可撤销。')) return
    setDeletingId(it.id)
    try {
      await deleteCodeSnippet(it.id)
      setItems(prev => prev.filter(x => x.id !== it.id))
      if (injected?.id === it.id) onRemove()
    } catch (err) {
      setMsg('❌ 删除失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally { setDeletingId('') }
  }

  /** 格式化收藏时间（ISO → 月-日） */
  const fmtDate = (iso: string) => {
    const d = new Date(iso)
    if (isNaN(d.getTime())) return ''
    const pad = (n: number) => (n < 10 ? '0' + n : '' + n)
    return `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  }

  /** 格式化字节数（B/KB） */
  const fmtSize = (n: number) => n >= 1024 ? (n / 1024).toFixed(1) + 'KB' : n + 'B'

  return (
    <div style={{ position: 'relative', display: 'inline-block' }}>
      {/* 已注入：显示芯片；未注入：显示注入按钮 */}
      {injected ? (
        <span style={{
          display: 'inline-flex', alignItems: 'center', gap: 6, padding: '8px 12px',
          borderRadius: 8, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.08)',
          color: '#7C3AED', fontSize: 13, fontWeight: 600, whiteSpace: 'nowrap',
          maxWidth: 260,
        }} title={'微调将参考「' + injected.title + '」的布局/交互/视觉手法'}>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>📎 已注入：{injected.title}</span>
          <button onClick={onRemove} disabled={disabled}
            style={{ border: 'none', background: 'transparent', color: '#7C3AED', fontSize: 13, fontWeight: 700, cursor: disabled ? 'default' : 'pointer', padding: 0, lineHeight: 1 }}
            title="移除参考代码">✕</button>
        </span>
      ) : (
        <button onClick={handleToggle} disabled={disabled}
          title="从我的代码收藏中选一条，让AI微调时参照其布局骨架/交互方式/视觉手法"
          style={{
            padding: '8px 14px', borderRadius: 8, border: `1px dashed ${disabled ? C.border : '#7C3AED'}`,
            background: open ? 'rgba(124,58,237,0.08)' : 'transparent',
            color: disabled ? '#9CA3AF' : '#7C3AED', fontSize: 13, fontWeight: 600,
            cursor: disabled ? 'default' : 'pointer', whiteSpace: 'nowrap',
          }}>
          📎 注入参考代码{open ? ' ▲' : ' ▼'}
        </button>
      )}

      {/* 收藏列表弹层（绝对定位在按钮下方） */}
      {open && !injected && (
        <div style={{
          position: 'absolute', top: 'calc(100% + 6px)', right: 0, zIndex: 1000,
          width: 380, maxHeight: 320, overflowY: 'auto',
          background: '#fff', borderRadius: 10, border: `1px solid ${C.border}`,
          boxShadow: '0 8px 30px rgba(0,0,0,0.15)', padding: 10,
        }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>
            ⭐ 我的代码收藏
            <span style={{ fontWeight: 400, color: '#9CA3AF', marginLeft: 6 }}>（点一条注入本次微调）</span>
          </div>
          {msg && <div style={{ fontSize: 12, color: '#DC2626', marginBottom: 6 }}>{msg}</div>}
          {loading ? (
            <div style={{ fontSize: 13, color: '#9CA3AF', padding: '14px 0', textAlign: 'center' }}>⏳ 加载中...</div>
          ) : items.length === 0 ? (
            <div style={{ fontSize: 12, color: '#9CA3AF', padding: '14px 4px', lineHeight: 1.7 }}>
              暂无收藏。在上方预览区选中满意的页 → 点「⭐ 收藏代码」即可加入代码库，之后微调任何课件都能注入参考。
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {items.map(it => (
                <div key={it.id} onClick={() => handlePick(it)}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 8, padding: '8px 10px',
                    borderRadius: 8, border: `1px solid ${C.border}`, background: '#FAFAFA',
                    cursor: (fetchingId || deletingId) ? 'default' : 'pointer',
                    opacity: fetchingId && fetchingId !== it.id ? 0.5 : 1,
                  }}>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {fetchingId === it.id ? '⏳ ' : '⭐ '}{it.title}
                    </div>
                    <div style={{ fontSize: 11, color: '#9CA3AF', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {fmtDate(it.created_at)} · {fmtSize(it.html_len)}{it.note ? ' · ' + it.note : ''}
                    </div>
                  </div>
                  <button onClick={(e) => handleDelete(e, it)} disabled={!!fetchingId || !!deletingId}
                    title="删除此收藏"
                    style={{ border: 'none', background: 'transparent', color: '#EF4444', fontSize: 13, cursor: (fetchingId || deletingId) ? 'default' : 'pointer', padding: '2px 4px', flexShrink: 0 }}>
                    {deletingId === it.id ? '⏳' : '🗑'}
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
