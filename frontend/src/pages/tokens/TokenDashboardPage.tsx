/**
 * TokenDashboardPage — Token积分管理主页
 *
 * v128 新增（阶段C · Token/积分系统）
 * v172 改造（积分管理三级数据权限隔离 · 角色驱动Tab与视图）：
 *   - 表格/弹窗/积分策略Tab 拆到 ./tokenDashboardParts（保持本文件 <600 行）
 *   - 按角色决定可见 Tab 与操作按钮：
 *       admin           → 账户/分配/采购/消费/策略 五Tab + 创建/充值/分配按钮 + 全系统统计
 *       senior_operator → 账户/分配/消费 三Tab + 分配按钮 + 本校统计（未绑定学校显示提示）
 *       operator/viewer → 账户(本人只读)/消费(本人) 两Tab + 本人统计，无写操作
 *   - 后端按 JWT 自动按角色收窄数据，前端只做 UI 显隐与文案区分
 */
import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '@/store/auth'
import {
  getTokenOverview, getTokenAccounts, getTokenAllocations,
  getTokenPurchases, getTokenConsumption,
  type TokenOverviewStats, type TokenAccountListItem,
  type AllocationListItem, type PurchaseListItem, type ConsumptionListItem,
} from '@/api/tokens'
import {
  C, tabBtnStyle,
  StatCard, AccountsTable, AllocationsTable, PurchasesTable, ConsumptionTable,
  CreateAccountModal, PurchaseModal, AllocateModal, PricingTab,
} from './tokenDashboardParts'

type TabKey = 'accounts' | 'allocations' | 'purchases' | 'consumption' | 'pricing'

// Tab 中文标签
const TAB_LABELS: Record<TabKey, string> = {
  accounts: '💰 账户管理',
  allocations: '📤 分配记录',
  purchases: '🛒 采购记录',
  consumption: '📊 消费流水',
  pricing: '💎 积分策略',
}

export default function TokenDashboardPage() {
  const { user } = useAuth()
  const role = user?.role || ''
  const isAdmin = role === 'admin'
  const isSchoolAdmin = role === 'senior_operator'
  const canAllocate = isAdmin || isSchoolAdmin
  // 个人视角：非 admin 且非学校管理员（普通/骨干教师）
  const isPersonal = !isAdmin && !isSchoolAdmin

  // 按角色决定可见 Tab
  const visibleTabs: TabKey[] = isAdmin
    ? ['accounts', 'allocations', 'purchases', 'consumption', 'pricing']
    : isSchoolAdmin
      ? ['accounts', 'allocations', 'consumption']
      : ['accounts', 'consumption'] // operator/viewer：本人账户 + 本人消费

  // 统计卡片文案前缀（区分范围）
  const scopeLabel = isAdmin ? '系统' : isSchoolAdmin ? '本校' : '我的'

  const [stats, setStats] = useState<TokenOverviewStats | null>(null)
  const [tab, setTab] = useState<TabKey>(visibleTabs[0])
  const [accounts, setAccounts] = useState<TokenAccountListItem[]>([])
  const [allocations, setAllocations] = useState<AllocationListItem[]>([])
  const [purchases, setPurchases] = useState<PurchaseListItem[]>([])
  const [consumption, setConsumption] = useState<ConsumptionListItem[]>([])
  const [loading, setLoading] = useState(false)
  // 范围被收窄提示（如 senior 未绑定学校），由后端 scope_message 下发
  const [scopeMsg, setScopeMsg] = useState('')

  // 弹窗状态
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showPurchaseModal, setShowPurchaseModal] = useState(false)
  const [showAllocateModal, setShowAllocateModal] = useState(false)
  const [allocateFromId, setAllocateFromId] = useState('')

  // 加载概览统计（后端按角色自动收窄）
  const loadStats = useCallback(async () => {
    try {
      const data = await getTokenOverview()
      setStats(data)
    } catch { /* ignore */ }
  }, [])

  // 加载列表数据（后端按角色自动收窄；账户/消费接口可能回传 scope_message）
  const loadList = useCallback(async () => {
    setLoading(true)
    setScopeMsg('')
    try {
      if (tab === 'accounts') {
        const data = await getTokenAccounts({ limit: 100 }) as { items?: TokenAccountListItem[]; scope_message?: string }
        setAccounts(data?.items || [])
        if (data?.scope_message) setScopeMsg(data.scope_message)
      } else if (tab === 'allocations') {
        const data = await getTokenAllocations({ limit: 100 })
        setAllocations(data?.items || [])
      } else if (tab === 'purchases') {
        const data = await getTokenPurchases({ limit: 100 })
        setPurchases(data?.items || [])
      } else if (tab === 'consumption') {
        const data = await getTokenConsumption({ limit: 100 }) as { items?: ConsumptionListItem[]; scope_message?: string }
        setConsumption(data?.items || [])
        if (data?.scope_message) setScopeMsg(data.scope_message)
      }
    } catch { /* ignore */ }
    setLoading(false)
  }, [tab])

  useEffect(() => { loadStats() }, [loadStats])
  useEffect(() => { loadList() }, [loadList])

  const formatNum = (n: number) => n < 10 && n > 0 ? n.toFixed(4) : n.toLocaleString(undefined, { maximumFractionDigits: 2 })

  return (
    <div>
      {/* ========== 范围提示（未绑定学校等）========== */}
      {scopeMsg && (
        <div style={{ marginBottom: '16px', padding: '12px 16px', borderRadius: '12px', background: 'rgba(245,158,11,0.08)', border: `1px solid ${C.orange}`, color: C.orange, fontSize: '13px' }}>
          ⚠️ {scopeMsg}
        </div>
      )}

      {/* ========== 概览统计 ========== */}
      {stats && (
        <div style={{ display: 'flex', gap: '16px', marginBottom: '24px', flexWrap: 'wrap' }}>
          {!isPersonal && <StatCard label={`${scopeLabel}账户数`} value={formatNum(stats.total_accounts)} color={C.primary} />}
          <StatCard label={`${scopeLabel}总余额`} value={`${formatNum(stats.total_balance)} 积分`} color={C.green} />
          <StatCard label={`${scopeLabel}本月消费`} value={`${formatNum(stats.month_consumed)} 积分`} color={C.orange} />
          <StatCard label={`${scopeLabel}今日消费`} value={`${formatNum(stats.today_consumed)} 积分`} color={C.purple} />
          {!isPersonal && stats.low_balance_count > 0 && (
            <StatCard label="余额预警" value={`${stats.low_balance_count} 个`} color={C.red} />
          )}
        </div>
      )}

      {/* ========== Tab切换 + 操作按钮 ========== */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '20px', flexWrap: 'wrap' }}>
        {visibleTabs.map(t => (
          <button key={t} onClick={() => setTab(t)} style={tabBtnStyle(tab === t)}>
            {TAB_LABELS[t]}
          </button>
        ))}
        <div style={{ flex: 1 }} />
        {isAdmin && tab === 'accounts' && (
          <button onClick={() => setShowCreateModal(true)}
            style={{ padding: '8px 16px', borderRadius: '8px', border: 'none', cursor: 'pointer', fontSize: '13px', fontWeight: 600, color: C.white, background: C.primary }}>
            + 创建账户
          </button>
        )}
        {isAdmin && tab === 'purchases' && (
          <button onClick={() => setShowPurchaseModal(true)}
            style={{ padding: '8px 16px', borderRadius: '8px', border: 'none', cursor: 'pointer', fontSize: '13px', fontWeight: 600, color: C.white, background: C.green }}>
            + 充值积分
          </button>
        )}
      </div>

      {/* ========== 列表内容 ========== */}
      {loading ? (
        <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px' }}>加载中...</div>
      ) : (
        <>
          {tab === 'accounts' && <AccountsTable items={accounts} canAllocate={canAllocate} onAllocate={(id) => { setAllocateFromId(id); setShowAllocateModal(true) }} />}
          {tab === 'allocations' && <AllocationsTable items={allocations} />}
          {tab === 'purchases' && <PurchasesTable items={purchases} />}
          {tab === 'consumption' && <ConsumptionTable items={consumption} />}
          {tab === 'pricing' && <PricingTab />}
        </>
      )}

      {/* ========== 弹窗 ========== */}
      {showCreateModal && <CreateAccountModal onClose={() => setShowCreateModal(false)} onSuccess={() => { setShowCreateModal(false); loadList(); loadStats() }} />}
      {showPurchaseModal && <PurchaseModal accounts={accounts} onClose={() => setShowPurchaseModal(false)} onSuccess={() => { setShowPurchaseModal(false); loadList(); loadStats() }} />}
      {showAllocateModal && <AllocateModal fromAccountId={allocateFromId} accounts={accounts} onClose={() => setShowAllocateModal(false)} onSuccess={() => { setShowAllocateModal(false); loadList(); loadStats() }} />}
    </div>
  )
}
