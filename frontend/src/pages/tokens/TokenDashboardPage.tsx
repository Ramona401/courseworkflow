/**
 * TokenDashboardPage — Token积分管理主页
 *
 * v128 新增（阶段C · Token/积分系统）
 * v172 改造（积分管理三级数据权限隔离 · 角色驱动Tab与视图）
 *
 * 排版修复（本次·纯样式）：
 *   原最外层 <div> 无任何 padding/maxWidth/margin，内容直接顶着 LPLayout 的
 *   <main padding:28px 32px> 铺满整宽，四周显得局促、表格被横向拉得过宽。
 *   现对齐 MyTeachingResourcesPage 的容器范式，给最外层加内边距 + 最大宽度 + 居中：
 *     - PAGE_PAD='4px 4px'：在 <main> 已有 28/32 外边距基础上再补一点呼吸空间；
 *     - maxWidth=1280：比资料页(1100)更宽，容纳区域账户卡片与多列宽表格；
 *     - margin:'0 auto'：内容居中，超宽屏不至于两侧留白失衡。
 *   仅改最外层容器一个 <div>，其余逻辑/结构逐字保留，零回归。
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
 * T1 区域分配入口 batch 新增：
 *   原有局面——账户列表 SQL 对非 admin 排除一切 region 账户（v172.1 防泄漏），
 *   region_admin 连自己管辖的区域账户也看不见，导致"从区域账户分配"没有前端入口，
 *   admin 只能替区域管理员逐校分配。本次补齐该入口：
 *     - 页面顶部（统计卡片之上）挂 <RegionAccountsCard/>：
 *       调 GET /tokens/my-region-accounts 显示"我管辖的区域账户"（余额 + 分配按钮），
 *       后端数据由 AllowedRegionOwnerIDs 驱动（fail-closed，无区域管辖者恒空）。
 *     - 点"📤 分配给学校" → 复用既有 AllocateModal + AccountPicker(mode=children)：
 *       getAllocatableTargets 对区域账户只返回其管辖学校（三重防越权已验证），分配链路零改动。
 *     - regionRefresh 刷新信号：分配成功后自增，卡片重拉余额即时更新。
 *   自此链路打通：admin 充值到区域账户 → 区域管理员自助分配到学校 → 学校再下发老师。
 *
 * 双重身份并集 batch（前端半边）：
 *   后端 ResolveTokenScope 已支持「学校管理员兼任区域管理员」的并集范围——
 *   users.role=senior_operator 但在 organization_admins 被任命为某区域 region_admin 的账号
 *  （如 lubin/lichao），主链路解析本校后 best-effort 并入辖区学校与区域分配来源
 *  （详见后端 token_scope.go）。前端相应地把 RegionAccountsCard 的挂载条件从
 *   "仅 isRegionAdmin"放宽为（isRegionAdmin || isSchoolAdmin）：
 *     - 兼任区域管辖的校管 → 后端 AllowedRegionOwnerIDs 非空 → 卡片显示其管辖的
 *       区域账户与"分配给学校"入口；账户Tab 亦能看到辖区学校账户（后端 OwnerIDs 并集）。
 *     - 普通校管（无区域任命）→ 后端 my-region-accounts 返回空列表 → 卡片自隐藏，
 *       界面与旧版完全一致（fail-closed，零回归）。
 *   分配链路（AllocateModal + AccountPicker mode=children + 三重防越权）零改动复用。
 *   注：兼任者的统计卡片前缀仍显示"本校"，但数据为"本校∪辖区"并集口径（后端收窄），
 *   属已知的文案轻微不贴切，不影响数据正确性。
 *
 * 一键分配 batch（前端半边，本次）：
 *   后端已上线 POST /tokens/accounts/{id}/batch-allocate（每户定额+预检余额+逐笔收集成败）。
 *   前端改动：
 *     - 新增 BatchAllocateModal（同目录新文件）：勾选目标（搜索过滤+全选）+ 每户金额 +
 *       实时总额与余额预警 + 提交后逐条成败结果展示。Props 与 AllocateModal 完全同签名。
 *     - 分配入口统一改为"分配方式选择"两步：点任何"分配"按钮（区域卡片/账户列表）先弹
 *       轻量选择层——「👤 单个分配」走原 AllocateModal，「⚡ 一键批量分配」走新弹窗。
 *       选择层直接内联在本文件（几十行），刻意不改 tokenDashboardParts.tsx 与
 *       RegionAccountsCard.tsx 两个共享文件（回归面最小；单笔链路逐字保留）。
 *     - 批量成功后同样触发 loadList + loadStats + regionRefresh（区域卡片余额即时刷新）。
 *   角色覆盖：admin / senior_operator / region_admin / 双重身份账号走同一入口，
 *   目标列表由后端 getAllocatableTargets 按 scope 收窄，无需前端区分。
 */
import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '@/store/auth'
import { useNavigate } from 'react-router-dom'
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
import RegionAccountsCard from './RegionAccountsCard'
import BatchAllocateModal from './BatchAllocateModal'

type TabKey = 'report' | 'accounts' | 'allocations' | 'purchases' | 'consumption' | 'pricing'

// 每页条数（四个列表统一口径）
const PAGE_SIZE = 20

// 排版修复：最外层容器统一样式（对齐 MyTeachingResourcesPage 的容器范式）。
//   LPLayout 的 <main> 已有 padding:28px 32px，此处再补少量内边距增加呼吸感；
//   maxWidth 1280 容纳区域卡片与多列宽表格；margin:0 auto 使内容在超宽屏居中。
const PAGE_CONTAINER_STYLE: React.CSSProperties = {
  padding: '4px 4px',
  maxWidth: 1280,
  margin: '0 auto',
}

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
  const navigate = useNavigate()
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
  // 一键分配 batch：分配方式选择层 + 批量分配弹窗
  const [showAllocateChooser, setShowAllocateChooser] = useState(false)
  const [showBatchModal, setShowBatchModal] = useState(false)
  // 分配来源账户信息（单个/批量两个弹窗共用同一份状态）
  const [allocateFrom, setAllocateFrom] = useState<{ id: string; name: string; balance: number }>({ id: '', name: '', balance: 0 })

  // T1：区域账户卡片刷新信号（分配成功后自增 → RegionAccountsCard 重拉余额）
  const [regionRefresh, setRegionRefresh] = useState(0)

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

  // 一键分配 batch：所有分配入口（区域卡片/账户列表）统一先打开"分配方式选择"层，
  // 用户再选走单个 AllocateModal 还是批量 BatchAllocateModal（两弹窗共用 allocateFrom）。
  const openAllocateChooser = (id: string, name: string, balance: number) => {
    setAllocateFrom({ id, name, balance })
    setShowAllocateChooser(true)
  }

  // T1：区域账户卡片"分配给学校"入口 → 进分配方式选择层
  const handleRegionAllocate = (id: string, name: string, balance: number) => {
    openAllocateChooser(id, name, balance)
  }

  // 分配成功后的统一刷新（单个/批量共用）：列表 + 统计 + 区域卡片余额
  const afterAllocateSuccess = () => {
    setShowAllocateModal(false)
    setShowBatchModal(false)
    loadList()
    loadStats()
    setRegionRefresh(v => v + 1)
  }

  const formatNum = (n: number) => n < 10 && n > 0 ? n.toFixed(4) : n.toLocaleString(undefined, { maximumFractionDigits: 2 })

  const showPagination = tab !== 'pricing'

  return (
    <div style={PAGE_CONTAINER_STYLE}>
      {/* ========== 独立页顶栏（脱离 LPLayout 后自带标题与返回入口） ========== */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '14px', marginBottom: '24px', paddingBottom: '16px', borderBottom: `1px solid ${C.border}` }}>
        <button
          onClick={() => navigate('/')}
          style={{ display: 'flex', alignItems: 'center', gap: '6px', padding: '8px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, background: C.white, cursor: 'pointer', fontSize: '13px', color: C.textSec, fontWeight: 500 }}
        >
          <span style={{ fontSize: '15px' }}>←</span> 返回首页
        </button>
        <div>
          <div style={{ fontSize: '20px', fontWeight: 700, color: C.text }}>💎 积分管理</div>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>账户 · 分配 · 消费 · 策略</div>
        </div>
      </div>

      {/* ========== T1+双重身份并集:我管辖的区域账户 ==========
          挂载条件（isRegionAdmin || isSchoolAdmin）：
            - region_admin：原 T1 行为不变；
            - senior_operator：兼任区域管辖者（后端并集）能看到区域账户与分配入口，
              普通校管后端返回空列表 → 组件内空列表自隐藏，界面零变化。 */}
      {(isRegionAdmin || isSchoolAdmin) && (
        <RegionAccountsCard refreshSignal={regionRefresh} onAllocate={handleRegionAllocate} />
      )}

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

      {/* ========== 账户Tab:类型筛选 chips + 搜索框(批次2,走后端) ========== */}
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
            openAllocateChooser(id, acc?.display_name || '', acc?.available_balance || 0)
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

      {/* ========== 一键分配 batch:分配方式选择层(内联轻量弹窗) ==========
          点任何"分配"按钮先到这里选方式,单笔链路(AllocateModal)与批量链路
          (BatchAllocateModal)共用 allocateFrom,共享文件零改动。 */}
      {showAllocateChooser && (
        <div onClick={() => setShowAllocateChooser(false)}
          style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.45)', zIndex: 999, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '20px' }}>
          <div onClick={e => e.stopPropagation()}
            style={{ width: 'min(420px, 92vw)', background: C.white, border: `1px solid ${C.border}`, borderRadius: '16px', padding: '24px', boxShadow: '0 20px 60px rgba(0,0,0,0.25)' }}>
            <div style={{ fontSize: '16px', fontWeight: 700, marginBottom: '4px' }}>选择分配方式</div>
            <div style={{ fontSize: '13px', color: C.textMuted, marginBottom: '18px' }}>
              来源:{allocateFrom.name}　可用余额:{formatNum(allocateFrom.balance)} 积分
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              <button
                onClick={() => { setShowAllocateChooser(false); setShowAllocateModal(true) }}
                style={{ padding: '14px', borderRadius: '10px', border: `1px solid ${C.border}`, background: 'transparent', cursor: 'pointer', fontSize: '14px', fontWeight: 600, color: C.textSec, textAlign: 'left' }}>
                👤 单个分配<span style={{ display: 'block', fontSize: '12px', fontWeight: 400, color: C.textMuted, marginTop: '2px' }}>选择一个目标账户,分配指定金额</span>
              </button>
              <button
                onClick={() => { setShowAllocateChooser(false); setShowBatchModal(true) }}
                style={{ padding: '14px', borderRadius: '10px', border: 'none', background: C.green, cursor: 'pointer', fontSize: '14px', fontWeight: 600, color: C.white, textAlign: 'left' }}>
                ⚡ 一键批量分配<span style={{ display: 'block', fontSize: '12px', fontWeight: 400, opacity: 0.85, marginTop: '2px' }}>勾选多个目标(支持全选),每户分配同样金额</span>
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ========== 弹窗(批次2:PurchaseModal/AllocateModal 对齐新 props) ==========
          单个/批量分配成功共用 afterAllocateSuccess:刷新列表+统计+区域卡片余额 */}
      {showCreateModal && <CreateAccountModal onClose={() => setShowCreateModal(false)} onSuccess={() => { setShowCreateModal(false); loadList(); loadStats() }} />}
      {showPurchaseModal && <PurchaseModal onClose={() => setShowPurchaseModal(false)} onSuccess={() => { setShowPurchaseModal(false); loadList(); loadStats() }} />}
      {showAllocateModal && <AllocateModal fromAccountId={allocateFrom.id} fromAccountName={allocateFrom.name} fromAccountBalance={allocateFrom.balance} onClose={() => setShowAllocateModal(false)} onSuccess={afterAllocateSuccess} />}
      {showBatchModal && <BatchAllocateModal fromAccountId={allocateFrom.id} fromAccountName={allocateFrom.name} fromAccountBalance={allocateFrom.balance} onClose={() => setShowBatchModal(false)} onSuccess={afterAllocateSuccess} />}
    </div>
  )
}
