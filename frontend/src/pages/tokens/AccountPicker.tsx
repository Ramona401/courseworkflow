/**
 * AccountPicker.tsx — 账户选择器组件（究极彻底版·批次2）
 *
 * 一个可复用的账户选择器，解决"账户多了下拉没法选"的问题。两种模式：
 *
 *   mode="all"（充值弹窗用）：
 *     - 列出所有账户（走 getTokenAccounts，后端按 TokenScope 收窄）
 *     - 类型筛选 chips（全部/区域/学校/个人）→ 走后端 type 参数
 *     - 搜索框（防抖300ms）→ 走后端 keyword 参数（ILIKE 模糊）
 *     - 滚动到底自动加载下一页（账户上千也只按需加载）
 *     - 充值可充给任何账户，故不限制类型
 *
 *   mode="children"（分配弹窗用，究极版）：
 *     - 只列来源账户的"合法下级"（走 getAllocatableTargets）
 *     - 后端三重防越权：来源账户须在 scope 内 + 下级被白名单收窄 + allocate 父子校验兜底
 *     - 下级类型固定、数量通常不多，故无类型筛选、无搜索、无分页（一次拉完）
 *     - 从根本上防止"选到非下级账户导致分配失败"
 *
 * 设计要点：
 *   - 受控组件：选中值由父组件 value/onChange 管理
 *   - 每项显示余额，选择时即可见，避免选错
 *   - 列表区固定高度 + 内部滚动，账户再多也不撑爆弹窗
 *   - 自包含加载状态，父组件无需关心数据获取
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import {
  getTokenAccounts, getAllocatableTargets,
  type TokenAccountListItem,
} from '@/api/tokens'

// ==================== 内部样式（与 tokenDashboardParts 的 C 调色板对齐）====================
const PC = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  green: '#10B981',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  bg: '#F9FAFB',
  white: '#FFFFFF',
}

// 每页条数（mode="all" 分页用）
const PICKER_PAGE_SIZE = 20

// 账户类型筛选选项（mode="all" 用）
const TYPE_CHIPS: { value: string; label: string }[] = [
  { value: '', label: '全部' },
  { value: 'region', label: '区域' },
  { value: 'school', label: '学校' },
  { value: 'personal', label: '个人' },
]

// 账户类型 → 图标（列表项展示用）
const TYPE_ICON: Record<string, string> = {
  region: '📍',
  school: '🏫',
  personal: '👤',
}

interface AccountPickerProps {
  mode: 'all' | 'children'
  value: string                       // 当前选中的账户ID
  onChange: (accountId: string, account: TokenAccountListItem | null) => void
  fromAccountId?: string              // mode="children" 时必填：来源账户ID
  excludeAccountId?: string           // 需要从列表中排除的账户ID（如充值时一般不排除；预留）
}

export default function AccountPicker({ mode, value, onChange, fromAccountId, excludeAccountId }: AccountPickerProps) {
  const [items, setItems] = useState<TokenAccountListItem[]>([])
  const [loading, setLoading] = useState(false)
  const [typeFilter, setTypeFilter] = useState('')   // mode="all" 的类型筛选
  const [keyword, setKeyword] = useState('')         // mode="all" 的搜索词（输入框绑定）
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)

  // 防抖计时器
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  // 列表滚动容器
  const listRef = useRef<HTMLDivElement | null>(null)

  // ---------- mode="children"：拉来源账户的合法下级（一次拉完，无分页）----------
  const loadChildren = useCallback(async () => {
    if (!fromAccountId) {
      setItems([])
      return
    }
    setLoading(true)
    try {
      const data = await getAllocatableTargets(fromAccountId)
      setItems(data?.items || [])
      setTotal(data?.total || 0)
    } catch {
      setItems([])
    }
    setLoading(false)
  }, [fromAccountId])

  // ---------- mode="all"：拉账户列表（带类型筛选+搜索+分页）----------
  // page=1 时替换列表；page>1 时追加（滚动加载）
  const loadAll = useCallback(async (targetPage: number, kw: string, type: string, append: boolean) => {
    setLoading(true)
    const offset = (targetPage - 1) * PICKER_PAGE_SIZE
    try {
      const data = await getTokenAccounts({
        type: type || undefined,
        keyword: kw || undefined,
        limit: PICKER_PAGE_SIZE,
        offset,
      })
      const newItems = data?.items || []
      setTotal(data?.total || 0)
      setItems(prev => append ? [...prev, ...newItems] : newItems)
    } catch {
      if (!append) setItems([])
    }
    setLoading(false)
  }, [])

  // 初始化 + 模式/来源变化时重新加载
  useEffect(() => {
    if (mode === 'children') {
      loadChildren()
    } else {
      setPage(1)
      loadAll(1, keyword, typeFilter, false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, fromAccountId])

  // mode="all"：类型筛选变化 → 回第1页重新加载
  const handleTypeChange = (type: string) => {
    setTypeFilter(type)
    setPage(1)
    loadAll(1, keyword, type, false)
  }

  // mode="all"：搜索输入 → 防抖后回第1页加载
  const handleKeywordChange = (kw: string) => {
    setKeyword(kw)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      setPage(1)
      loadAll(1, kw, typeFilter, false)
    }, 300)
  }

  // mode="all"：滚动到底加载下一页
  const handleScroll = () => {
    if (mode !== 'all') return
    const el = listRef.current
    if (!el || loading) return
    const reachedBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 20
    const hasMore = items.length < total
    if (reachedBottom && hasMore) {
      const next = page + 1
      setPage(next)
      loadAll(next, keyword, typeFilter, true)
    }
  }

  // 清理防抖计时器
  useEffect(() => {
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current) }
  }, [])

  // 实际展示的列表（排除指定账户）
  const visibleItems = excludeAccountId
    ? items.filter(a => a.id !== excludeAccountId)
    : items

  const selectedAccount = visibleItems.find(a => a.id === value) || null

  return (
    <div>
      {/* ===== mode="all"：搜索框 + 类型筛选 chips ===== */}
      {mode === 'all' && (
        <>
          <input
            value={keyword}
            onChange={e => handleKeywordChange(e.target.value)}
            placeholder="🔍 输入账户名搜索…"
            style={{
              width: '100%', padding: '10px 12px', borderRadius: '8px',
              border: `1px solid ${PC.border}`, fontSize: '14px', outline: 'none',
              boxSizing: 'border-box', marginBottom: '10px',
            }}
          />
          <div style={{ display: 'flex', gap: '6px', marginBottom: '10px', flexWrap: 'wrap' }}>
            {TYPE_CHIPS.map(chip => (
              <button
                key={chip.value}
                onClick={() => handleTypeChange(chip.value)}
                style={{
                  padding: '5px 14px', borderRadius: '16px', cursor: 'pointer',
                  fontSize: '13px', fontWeight: typeFilter === chip.value ? 600 : 400,
                  border: `1px solid ${typeFilter === chip.value ? PC.primary : PC.border}`,
                  background: typeFilter === chip.value ? PC.primary : PC.white,
                  color: typeFilter === chip.value ? PC.white : PC.textSec,
                  transition: 'all 120ms ease',
                }}
              >
                {chip.label}
              </button>
            ))}
          </div>
        </>
      )}

      {/* ===== 列表区（固定高度 + 内部滚动）===== */}
      <div
        ref={listRef}
        onScroll={handleScroll}
        style={{
          maxHeight: '240px', overflowY: 'auto',
          border: `1px solid ${PC.border}`, borderRadius: '10px',
          background: PC.white,
        }}
      >
        {visibleItems.length === 0 && !loading && (
          <div style={{ textAlign: 'center', color: PC.textMuted, padding: '32px 16px', fontSize: '13px' }}>
            {mode === 'children' ? '该账户暂无可分配的下级账户' : '未找到匹配的账户'}
          </div>
        )}

        {visibleItems.map(a => {
          const selected = a.id === value
          return (
            <div
              key={a.id}
              onClick={() => onChange(a.id, a)}
              style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                padding: '10px 14px', cursor: 'pointer',
                borderBottom: `1px solid ${PC.bg}`,
                background: selected ? PC.primaryLight : 'transparent',
                transition: 'background 100ms ease',
              }}
              onMouseEnter={e => { if (!selected) (e.currentTarget as HTMLDivElement).style.background = PC.bg }}
              onMouseLeave={e => { if (!selected) (e.currentTarget as HTMLDivElement).style.background = 'transparent' }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0 }}>
                <span style={{
                  width: '16px', height: '16px', borderRadius: '50%', flexShrink: 0,
                  border: `2px solid ${selected ? PC.primary : PC.border}`,
                  background: selected ? PC.primary : PC.white,
                  display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
                }}>
                  {selected && <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: PC.white }} />}
                </span>
                <span style={{ fontSize: '13px' }}>{TYPE_ICON[a.account_type] || ''}</span>
                <span style={{
                  fontSize: '14px', fontWeight: selected ? 600 : 400, color: PC.text,
                  whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
                }}>
                  {a.display_name}
                </span>
                <span style={{
                  fontSize: '11px', padding: '1px 7px', borderRadius: '4px', flexShrink: 0,
                  background: PC.primaryLight, color: PC.primary,
                }}>
                  {a.account_type_name}
                </span>
              </div>
              <span style={{ fontSize: '13px', fontWeight: 600, color: PC.green, flexShrink: 0, marginLeft: '8px' }}>
                {a.available_balance.toLocaleString()} 积分
              </span>
            </div>
          )
        })}

        {loading && (
          <div style={{ textAlign: 'center', color: PC.textMuted, padding: '12px', fontSize: '12px' }}>
            加载中…
          </div>
        )}
      </div>

      {/* ===== mode="all"：列表计数提示 ===== */}
      {mode === 'all' && total > 0 && (
        <div style={{ fontSize: '11px', color: PC.textMuted, marginTop: '6px', textAlign: 'right' }}>
          已加载 {visibleItems.length} / 共 {total.toLocaleString()} 个账户
        </div>
      )}

      {/* ===== 已选确认条 ===== */}
      {selectedAccount && (
        <div style={{
          marginTop: '10px', padding: '8px 12px', borderRadius: '8px',
          background: PC.primaryLight, fontSize: '13px', color: PC.text,
        }}>
          已选：<strong>{selectedAccount.display_name}</strong>
          <span style={{ color: PC.textSec }}>（{selectedAccount.available_balance.toLocaleString()} 积分可用）</span>
        </div>
      )}
    </div>
  )
}
