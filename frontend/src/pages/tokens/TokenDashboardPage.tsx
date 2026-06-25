/**
 * TokenDashboardPage — Token积分管理主页
 *
 * v128 新增（阶段C · Token/积分系统）
 * v172 改造（积分管理三级数据权限隔离 · 角色驱动Tab与视图）
 *
 * P1+P3 交付前完善：
 *   - P1 分页：四个列表统一分页器，offset 传后端，total 由后端返回；切 Tab 回第1页。
 *   - P3 过滤：分配记录的月度自充值在 AllocationsTable 内前端过滤。
 *
 * 究极彻底版·批次2（账户选择/筛选体验）：
 *   - 账户管理Tab 顶部加"类型筛选 chips + 搜索框"：类型走后端 type 参数、搜索走后端 keyword。
 *   - 充值弹窗/分配弹窗 改用 AccountPicker（充值 mode=all，分配 mode=children 只列合法下级）。
 *
 * 究极彻底版·A（分配记录 total 精确）：
 *   - 分配记录请求带 exclude_monthly=true：后端排除月度自充值，items 与 total 一致。
 *     AllocationsTable 内的前端过滤保留作双保险（无害）。
 *
 * region_admin 区域分配 batch（前端那一半）：
 *   后端已放开 region_admin 的区域分配能力（可从自己管辖的区域账户向管辖学校账户分配），
 *   本前端改动让区域管理员在积分页拥有对应的可见Tab与分配入口：
 *     - 角色判定新增 isRegionAdmin；canAllocate 纳入 region_admin（出现"分配"按钮与弹窗）。
 *     - 视图层级 isManager（admin/senior/region 共有的"管理者视角"）替代原 isPersonal 取反，
 *       使 region_admin 不再被误判为"个人/我的"视角（修复统计卡片显示"我的总余额"的错位）。
 *     - scopeLabel：region_admin → "本区域"（统计卡片前缀，语义对齐管辖区域）。
 *     - visibleTabs：region_admin → accounts/allocations/consumption/purchases 四个
 *       （可查账户/分配记录/消费流水/采购记录；但充值按钮、创建账户、积分策略Tab 仍仅 admin）。
 *   分配弹窗 AllocateModal 与其内部的 AccountPicker(mode=children) 完全复用，零改动——
 *   后端 GetAllocatableTargets 对 region_admin 只返回管辖学校（已通过越权测试验证），
 *   故 region_admin 打开分配弹窗时，目标账户天然只列出其管辖区域内的学校账户。
 *
 *   ⚠ 已知预期行为（非bug）：region_admin 的消费流水后端 UserIDs 为空集（本轮不开放其
 *     个人消费维度），故"消费流水"Tab 对 region_admin 显示为空列表，属设计预期。
 *     给该Tab仅为界面完整性，后续如需开放区域消费汇总再单独迭代后端。
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
  C, tabBtnStyle, inputStyle,
  StatCard, AccountsTable, AllocationsTable, PurchasesTable, ConsumptionTable,
  CreateAccountModal, PurchaseModal, AllocateModal,
  Pagination,
} from './tokenDashboardParts'
import { PricingTab } from './tokenDashboardPricing'
import TokenSummaryReport from './tokenSummaryReport'

type TabKey = 'report' | 'accounts' | 'allocations' | 'purchases' | 'consumption' | 'pricing'

// 每页条数（四个列表统一口径）
const PAGE_SIZE = 20

// Tab 中文标签
const TAB_LABELS: Record<TabKey, string> = {
  report: '📈 汇总报告',
  accounts: '💰 账户管理',
  allocations: '📤 分配记录',
  purchases: '🛒 采购记录',
  consumption: '📊 消费流水',
  pricing: '💎 积分策略',
}

// 账户类型筛选 chips（账户Tab 用，走后端 type 参数）
const ACCOUNT_TYPE_CHIPS: { value: string; label: string }[] = [
  { value: '', label: '全部' },
  { value: 'region', label: '📍 区域' },
  { value: 'school', label: '🏫 学校' },
  { value: 'personal', label: '👤 个人' },
]

export default function TokenDashboardPage() {
  const { user } = useAuth()
  const role = user?.role || ''
  const isAdmin = role === 'admin'
  const isSchoolAdmin = role === 'senior_operator'
  const isRegionAdmin = role === 'region_admin' // region_admin batch：区域管理员

  // canAllocate：能否分配积分（出现"分配"按钮与弹窗）。
  //   admin / senior_operator / region_admin 三类管理者均可，后端各自按 scope 收窄分配来源。
  const canAllocate = isAdmin || isSchoolAdmin || isRegionAdmin

  // isManager：管理者视角（区别于普通老师"我的/个人"视角）。
  //   原代码用 isPersonal=!isAdmin&&!isSchoolAdmin 导致 region_admin 被误判为个人视角，
  //   此处显式纳入 region_admin，使统计卡片显示"账户数/区域余额"等管理者维度。
  const isManager = isAdmin || isSchoolAdmin || isRegionAdmin
  const isPersonal = !isManager

  // 可见Tab：
  //   admin          → 全部五个（含采购/策略）
  //   region_admin   → accounts/allocations/consumption/purchases 四个（看采购记录，
  //                    但充值按钮、创建账户、积分策略Tab 仍仅 admin；消费流水暂为空集）
  //   senior_operator→ accounts/allocations/consumption 三个
  //   operator/viewer→ accounts/consumption 两个
  const visibleTabs: TabKey[] = isAdmin
    ? ['report', 'accounts', 'allocations', 'purchases', 'consumption', 'pricing']
    : isRegionAdmin
      ? ['report', 'accounts', 'allocations', 'consumption', 'purchases']
      : isSchoolAdmin
        ? ['report', 'accounts', 'allocations', 'consumption']
        : ['accounts', 'consumption']

  // 统计卡片范围前缀：admin=系统 / region_admin=本区域 / senior=本校 / 其它=我的
  const scopeLabel = isAdmin ? '系统' : isRegionAdmin ? '本区域' : isSchoolAdmin ? '本校' : '我的'

  const [stats, setStats] = useState<TokenOverviewStats | null>(null)
  const [tab, setTab] = useState<TabKey>(visibleTabs[0])
  const [accounts, setAccounts] = useState<TokenAccountListItem[]>([])
  const [allocations, setAllocations] = useState<AllocationListItem[]>([])
  const [purchases, setPurchases] = useState<PurchaseListItem[]>([])
  const [consumption, setConsumption] = useState<ConsumptionListItem[]>([])
  const [loading, setLoading] = useState(false)
  const [scopeMsg, setScopeMsg] = useState('')

  // P1 分页 state
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)

  // 批次2：账户Tab 的类型筛选 + 搜索（走后端）
  const [accountTypeFilter, setAccountTypeFilter] = useState('')
  const [accountKeyword, setAccountKeyword] = useState('')

  // 弹窗状态
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showPurchaseModal, setShowPurchaseModal] = useState(false)
  const [showAllocateModal, setShowAllocateModal] = useState(false)
  // 分配来源账户信息（批次2：AllocateModal 改为接收 名称/余额 展示）
  const [allocateFrom, setAllocateFrom] = useState<{ id: string; name: string; balance: number }>({ id: '', name: '', balance: 0 })

  const loadStats = useCallback(async () => {
    try {
      const data = await getTokenOverview()
      setStats(data)
    } catch { /* ignore */ }
  }, [])

  // 加载列表数据
  // P1：按 page 计算 offset；账户Tab 额外带 type/keyword（批次2 后端筛选）；
  // 分配记录带 exclude_monthly=true（A：后端排除月度自充值，total 精确）。
  const loadList = useCallback(async () => {
    setLoading(true)
    setScopeMsg('')
    const offset = (page - 1) * PAGE_SIZE
    try {
      if (tab === 'accounts') {
        const data = await getTokenAccounts({
          type: accountTypeFilter || undefined,
          keyword: accountKeyword || undefined,
          limit: PAGE_SIZE, offset,
        }) as { items?: TokenAccountListItem[]; total?: number; scope_message?: string }
        setAccounts(data?.items || [])
        setTotal(data?.total || 0)
        if (data?.scope_message) setScopeMsg(data.scope_message)
      } else if (tab === 'allocations') {
        const data = await getTokenAllocations({ exclude_monthly: true, limit: PAGE_SIZE, offset })
        setAllocations(data?.items || [])
        setTotal(data?.total || 0)
      } else if (tab === 'purchases') {
        const data = await getTokenPurchases({ limit: PAGE_SIZE, offset })
        setPurchases(data?.items || [])
        setTotal(data?.total || 0)
      } else if (tab === 'consumption') {
        const data = await getTokenConsumption({ limit: PAGE_SIZE, offset }) as { items?: ConsumptionListItem[]; total?: number; scope_message?: string }
        setConsumption(data?.items || [])
        setTotal(data?.total || 0)
        if (data?.scope_message) setScopeMsg(data.scope_message)
      }
    } catch { /* ignore */ }
    setLoading(false)
  }, [tab, page, accountTypeFilter, accountKeyword])

  useEffect(() => { loadStats() }, [loadStats])
  useEffect(() => { loadList() }, [loadList])

  // 切 Tab 时回到第1页，并清空账户筛选条件（避免筛选残留到其它Tab）
  const handleSwitchTab = (t: TabKey) => {
    setTab(t)
    setPage(1)
    setAccountTypeFilter('')
    setAccountKeyword('')
  }

  // 账户Tab：类型筛选切换 → 回第1页（loadList 依赖 accountTypeFilter 自动触发）
  const handleAccountTypeChange = (type: string) => {
    setAccountTypeFilter(type)
    setPage(1)
  }

  // 账户Tab：搜索输入 → 回第1页（受控输入，loadList 依赖 accountKeyword 自动触发）
  const handleAccountKeywordChange = (kw: string) => {
    setAccountKeyword(kw)
    setPage(1)
  }

  const formatNum = (n: number) => n < 10 && n > 0 ? n.toFixed(4) : n.toLocaleString(undefined, { maximumFractionDigits: 2 })

  const showPagination = tab !== 'pricing'

  return (
    <div>
      {/* ========== 范围提示 ========== */}
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
          <button key={t} onClick={() => handleSwitchTab(t)} style={tabBtnStyle(tab === t)}>
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

      {/* ========== 账户Tab：类型筛选 chips + 搜索框（批次2，走后端）========== */}
      {tab === 'accounts' && (
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '16px', flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
            {ACCOUNT_TYPE_CHIPS.map(chip => (
              <button
                key={chip.value}
                onClick={() => handleAccountTypeChange(chip.value)}
                style={{
                  padding: '6px 16px', borderRadius: '16px', cursor: 'pointer',
                  fontSize: '13px', fontWeight: accountTypeFilter === chip.value ? 600 : 400,
                  border: `1px solid ${accountTypeFilter === chip.value ? C.primary : C.border}`,
                  background: accountTypeFilter === chip.value ? C.primary : C.white,
                  color: accountTypeFilter === chip.value ? C.white : C.textSec,
                  transition: 'all 120ms ease',
                }}
              >
                {chip.label}
              </button>
            ))}
          </div>
          <div style={{ flex: 1, minWidth: '200px', maxWidth: '320px' }}>
            <input
              value={accountKeyword}
              onChange={e => handleAccountKeywordChange(e.target.value)}
              placeholder="🔍 搜索账户名…"
              style={inputStyle}
            />
          </div>
        </div>
      )}

      {/* ========== 列表内容 ========== */}
      {loading ? (
        <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px' }}>加载中...</div>
      ) : (
        <>
          {tab === 'report' && <TokenSummaryReport />}
          {tab === 'accounts' && <AccountsTable items={accounts} canAllocate={canAllocate} onAllocate={(id) => {
            const acc = accounts.find(a => a.id === id)
            setAllocateFrom({ id, name: acc?.display_name || '', balance: acc?.available_balance || 0 })
            setShowAllocateModal(true)
          }} />}
          {tab === 'allocations' && <AllocationsTable items={allocations} />}
          {tab === 'purchases' && <PurchasesTable items={purchases} />}
          {tab === 'consumption' && <ConsumptionTable items={consumption} isAdmin={isAdmin} />}
          {tab === 'pricing' && <PricingTab />}

          {/* P1 分页器 */}
          {showPagination && (
            <Pagination
              page={page}
              pageSize={PAGE_SIZE}
              total={total}
              onChange={setPage}
            />
          )}
        </>
      )}

      {/* ========== 弹窗（批次2：PurchaseModal/AllocateModal 对齐新 props）========== */}
      {showCreateModal && <CreateAccountModal onClose={() => setShowCreateModal(false)} onSuccess={() => { setShowCreateModal(false); loadList(); loadStats() }} />}
      {showPurchaseModal && <PurchaseModal onClose={() => setShowPurchaseModal(false)} onSuccess={() => { setShowPurchaseModal(false); loadList(); loadStats() }} />}
      {showAllocateModal && <AllocateModal fromAccountId={allocateFrom.id} fromAccountName={allocateFrom.name} fromAccountBalance={allocateFrom.balance} onClose={() => setShowAllocateModal(false)} onSuccess={() => { setShowAllocateModal(false); loadList(); loadStats() }} />}
    </div>
  )
}
