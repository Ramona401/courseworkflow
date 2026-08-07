/**
 * tokenDashboardParts.tsx — 积分管理页子组件与共享样式
 *
 * v172 拆分：从 TokenDashboardPage.tsx 抽出表格 / 弹窗 / 共享样式。
 * v199 拆分：PricingTab（积分策略Tab）已拆至 tokenDashboardPricing.tsx（超 600 红线拆分），
 *           本文件不再含汇率倍率编辑与模型单价表逻辑。
 *
 * 共享样式 C/cardStyle/thStyle/tdStyle/inputStyle 仍在本文件 export，
 * 被 tokenDashboardPricing / tokenSummaryReport / TokenDashboardPage 多处 import（单一真相源）。
 *
 * 究极彻底版·批次2（账户选择器接入）：
 *   - PurchaseModal（充值）接入 AccountPicker(mode="all")：可搜索/类型筛选/滚动加载。
 *   - AllocateModal（分配）接入 AccountPicker(mode="children")：只显示来源账户的合法下级。
 *
 * 导出：
 *   - 共享样式常量 C / cardStyle / tabBtnStyle / thStyle / tdStyle / inputStyle 等
 *   - StatCard / Pagination
 *   - AccountsTable / AllocationsTable / PurchasesTable / ConsumptionTable
 *   - CreateAccountModal / PurchaseModal / AllocateModal
 *   - ModalOverlay / FormField
 */
import { useState, useEffect } from 'react'
import {
  createTokenAccount, purchaseTokens, allocateTokens,
  type TokenAccountListItem, type AllocationListItem,
  type PurchaseListItem, type ConsumptionListItem,
  ACCOUNT_TYPE_OPTIONS, PURCHASE_TYPE_OPTIONS, ACCOUNT_STATUS_COLORS,
  SCENE_CODE_LABELS,
} from '@/api/tokens'
import { loadAliasResolver } from '@/api/modelAliasResolve'
import AccountPicker from './AccountPicker'

// ==================== 样式常量 ====================
export const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  green: '#10B981',
  orange: '#F59E0B',
  red: '#EF4444',
  purple: '#8B5CF6',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  bg: '#F9FAFB',
  white: '#FFFFFF',
}

export const cardStyle: React.CSSProperties = {
  background: C.white, borderRadius: '16px', border: `1px solid ${C.border}`,
  padding: '20px 24px', flex: 1, minWidth: '180px',
}

export const tabBtnStyle = (active: boolean): React.CSSProperties => ({
  padding: '8px 20px', borderRadius: '8px', border: 'none', cursor: 'pointer',
  fontSize: '14px', fontWeight: active ? 600 : 400,
  color: active ? C.white : C.textSec,
  background: active ? C.primary : C.primaryLight,
  transition: 'all 150ms ease',
})

export const thStyle: React.CSSProperties = { padding: '10px 12px', textAlign: 'left', fontSize: '12px', color: C.textMuted, fontWeight: 500, borderBottom: `1px solid ${C.border}` }
export const tdStyle: React.CSSProperties = { padding: '12px', fontSize: '13px', color: C.text, borderBottom: `1px solid ${C.border}` }

export const inputStyle: React.CSSProperties = {
  width: '100%', padding: '10px 12px', borderRadius: '8px',
  border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none',
  boxSizing: 'border-box',
}
export const cancelBtnStyle: React.CSSProperties = {
  padding: '8px 20px', borderRadius: '8px', border: `1px solid ${C.border}`,
  background: C.white, color: C.textSec, cursor: 'pointer', fontSize: '14px',
}
export const submitBtnStyle: React.CSSProperties = {
  padding: '8px 20px', borderRadius: '8px', border: 'none',
  background: C.primary, color: C.white, cursor: 'pointer', fontSize: '14px', fontWeight: 600,
}

// ==================== 统计卡片 ====================
export function StatCard({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div style={cardStyle}>
      <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '8px' }}>{label}</div>
      <div style={{ fontSize: '22px', fontWeight: 700, color }}>{value}</div>
    </div>
  )
}

// ==================== 分页器（P1，四列表共用）====================
export function Pagination({ page, pageSize, total, onChange }: { page: number; pageSize: number; total: number; onChange: (p: number) => void }) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  if (total <= 0 || totalPages <= 1) return null

  const canPrev = page > 1
  const canNext = page < totalPages

  const btnStyle = (enabled: boolean): React.CSSProperties => ({
    padding: '6px 14px', borderRadius: '8px',
    border: `1px solid ${enabled ? C.primary : C.border}`,
    background: enabled ? C.white : C.bg,
    color: enabled ? C.primary : C.textMuted,
    cursor: enabled ? 'pointer' : 'not-allowed',
    fontSize: '13px', fontWeight: 500,
  })

  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '12px', marginTop: '16px', padding: '8px 0' }}>
      <button style={btnStyle(canPrev)} disabled={!canPrev} onClick={() => canPrev && onChange(page - 1)}>
        ← 上一页
      </button>
      <span style={{ fontSize: '13px', color: C.textSec }}>
        第 <strong style={{ color: C.text }}>{page}</strong> / {totalPages} 页
      </span>
      <button style={btnStyle(canNext)} disabled={!canNext} onClick={() => canNext && onChange(page + 1)}>
        下一页 →
      </button>
      <span style={{ fontSize: '12px', color: C.textMuted, marginLeft: '8px' }}>
        共 {total.toLocaleString()} 条
      </span>
    </div>
  )
}

// ==================== 账户表格 ====================
export function AccountsTable({ items, canAllocate, onAllocate }: { items: TokenAccountListItem[]; canAllocate: boolean; onAllocate: (id: string) => void }) {
  if (!items?.length) return <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px' }}>暂无账户数据</div>
  return (
    <div style={{ background: C.white, borderRadius: '12px', border: `1px solid ${C.border}`, overflow: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead><tr>
          <th style={thStyle}>账户名称</th><th style={thStyle}>类型</th><th style={thStyle}>可用余额</th>
          <th style={thStyle}>总配额</th><th style={thStyle}>已消费</th><th style={thStyle}>使用率</th>
          <th style={thStyle}>状态</th><th style={thStyle}>子账户</th>
          {canAllocate && <th style={thStyle}>操作</th>}
        </tr></thead>
        <tbody>
          {items.map(a => (
            <tr key={a.id}>
              <td style={tdStyle}><span style={{ fontWeight: 600 }}>{a.display_name}</span></td>
              <td style={tdStyle}><span style={{ fontSize: '12px', padding: '2px 8px', borderRadius: '4px', background: C.primaryLight, color: C.primary }}>{a.account_type_name}</span></td>
              <td style={tdStyle}><span style={{ fontWeight: 600, color: C.green }}>{a.available_balance.toLocaleString()}</span></td>
              <td style={tdStyle}>{a.total_quota.toLocaleString()}</td>
              <td style={tdStyle}>{a.total_consumed.toLocaleString()}</td>
              <td style={tdStyle}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <div style={{ width: '50px', height: '6px', background: '#F3F4F6', borderRadius: '3px', overflow: 'hidden' }}>
                    <div style={{ width: `${Math.min(a.usage_percent, 100)}%`, height: '100%', background: a.usage_percent > 80 ? C.red : a.usage_percent > 50 ? C.orange : C.green, borderRadius: '3px' }} />
                  </div>
                  <span style={{ fontSize: '12px', color: C.textSec }}>{a.usage_percent.toFixed(1)}%</span>
                </div>
              </td>
              <td style={tdStyle}><span style={{ color: ACCOUNT_STATUS_COLORS[a.status] || C.textMuted, fontWeight: 500 }}>{a.status_name}</span></td>
              <td style={tdStyle}>{a.child_count > 0 ? a.child_count : '-'}</td>
              {canAllocate && (
                <td style={tdStyle}>
                  <button onClick={() => onAllocate(a.id)}
                    style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.primary}`, background: 'transparent', color: C.primary, cursor: 'pointer', fontSize: '12px' }}>
                    分配
                  </button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ==================== 分配记录表格 ====================
// P3：过滤掉 allocation_type='monthly' 的月度自充值（from=to，语义混乱），
//     只展示人为分配(manual/initial)。过滤在组件内完成，主页面无感知。
export function AllocationsTable({ items }: { items: AllocationListItem[] }) {
  const visibleItems = (items || []).filter(a => a.allocation_type !== 'monthly')

  if (!visibleItems.length) return <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px' }}>暂无分配记录</div>
  return (
    <div style={{ background: C.white, borderRadius: '12px', border: `1px solid ${C.border}`, overflow: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead><tr>
          <th style={thStyle}>来源账户</th><th style={thStyle}>目标账户</th><th style={thStyle}>积分数</th>
          <th style={thStyle}>类型</th><th style={thStyle}>操作人</th><th style={thStyle}>备注</th><th style={thStyle}>时间</th>
        </tr></thead>
        <tbody>
          {visibleItems.map(a => (
            <tr key={a.id}>
              <td style={tdStyle}>{a.from_account_name}</td>
              <td style={tdStyle}>{a.to_account_name}</td>
              <td style={tdStyle}><span style={{ fontWeight: 600, color: C.primary }}>{a.amount.toLocaleString()}</span></td>
              <td style={tdStyle}>{a.allocation_type}</td>
              <td style={tdStyle}>{a.operator_name}</td>
              <td style={tdStyle}>{a.memo || '-'}</td>
              <td style={{ ...tdStyle, fontSize: '12px', color: C.textSec }}>{new Date(a.created_at).toLocaleString('zh-CN')}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ==================== 采购记录表格 ====================
export function PurchasesTable({ items }: { items: PurchaseListItem[] }) {
  if (!items?.length) return <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px' }}>暂无采购记录</div>
  return (
    <div style={{ background: C.white, borderRadius: '12px', border: `1px solid ${C.border}`, overflow: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead><tr>
          <th style={thStyle}>账户</th><th style={thStyle}>积分数</th><th style={thStyle}>类型</th>
          <th style={thStyle}>订单号</th><th style={thStyle}>操作人</th><th style={thStyle}>备注</th><th style={thStyle}>时间</th>
        </tr></thead>
        <tbody>
          {items.map(p => (
            <tr key={p.id}>
              <td style={tdStyle}>{p.account_name}</td>
              <td style={tdStyle}><span style={{ fontWeight: 600, color: C.green }}>+{p.amount.toLocaleString()}</span></td>
              <td style={tdStyle}>{p.purchase_type}</td>
              <td style={tdStyle}>{p.order_no || '-'}</td>
              <td style={tdStyle}>{p.operator_name}</td>
              <td style={tdStyle}>{p.memo || '-'}</td>
              <td style={{ ...tdStyle, fontSize: '12px', color: C.textSec }}>{new Date(p.created_at).toLocaleString('zh-CN')}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ==================== 消费流水表格 ====================

const CONSUMPTION_UNIT_LABELS: Record<string, string> = {
  request: '次',
  image: '张',
  video: '个视频',
  character: '字符',
  audio_second: '秒',
  second: '秒',
  provider_token: '供应商计量',
}

function formatConsumptionNumber(
  value: number,
): string {
  if (
    value > 0 &&
    value < 10
  ) {
    return value.toLocaleString(
      'zh-CN',
      {
        maximumFractionDigits: 3,
      },
    )
  }

  return value.toLocaleString(
    'zh-CN',
    {
      maximumFractionDigits: 2,
    },
  )
}

function resolveConsumptionBusinessName(
  item: ConsumptionListItem,
): string {
  return (
    item.business_name?.trim() ||
    item.billing_node_name?.trim() ||
    SCENE_CODE_LABELS[item.scene_code] ||
    item.scene_code ||
    'AI服务'
  )
}

function resolveConsumptionUsage(
  item: ConsumptionListItem,
): string {
  const safeUnit =
    item.usage_unit?.trim()

  if (safeUnit) {
    const quantity =
      item.usage_quantity ?? 0

    return `${formatConsumptionNumber(quantity)} ${
      CONSUMPTION_UNIT_LABELS[safeUnit] ||
      safeUnit
    }`
  }

  const internalUnit =
    item.media_unit?.trim()

  if (
    internalUnit &&
    (item.media_quantity ?? 0) > 0
  ) {
    return `${formatConsumptionNumber(
      item.media_quantity ?? 0,
    )} ${
      CONSUMPTION_UNIT_LABELS[internalUnit] ||
      internalUnit
    }`
  }

  return '1 次'
}

export function ConsumptionTable({
  items,
  isSuperAdmin = false,
}: {
  items: ConsumptionListItem[]
  isSuperAdmin?: boolean
}) {
  // 财务成本、真实模型和Token明细只对超级管理员显示。
  // 其它角色只看到积分、业务名称、安全用量、余额与时间。
  const [resolveAlias, setResolveAlias] =
    useState<((model: string) => string) | null>(null)

  useEffect(() => {
    if (!isSuperAdmin) {
      setResolveAlias(null)
      return
    }

    let alive = true

    loadAliasResolver().then(resolver => {
      if (alive) {
        setResolveAlias(
          () => resolver,
        )
      }
    })

    return () => {
      alive = false
    }
  }, [isSuperAdmin])

  if (!items?.length) {
    return (
      <div style={{
        textAlign: 'center',
        color: C.textMuted,
        padding: '40px',
      }}>
        暂无消费记录
      </div>
    )
  }

  return (
    <div style={{
      background: C.white,
      borderRadius: '12px',
      border: `1px solid ${C.border}`,
      overflow: 'auto',
    }}>
      <table style={{
        width: '100%',
        borderCollapse: 'collapse',
      }}>
        <thead>
          <tr>
            <th style={thStyle}>用户</th>
            <th style={thStyle}>消费积分</th>
            <th style={thStyle}>业务</th>
            <th style={thStyle}>用量</th>
            {isSuperAdmin && (
              <th style={thStyle}>模型</th>
            )}
            {isSuperAdmin && (
              <th style={thStyle}>Token(入/出)</th>
            )}
            {isSuperAdmin && (
              <th style={thStyle}>美元成本</th>
            )}
            <th style={thStyle}>余额变化</th>
            <th style={thStyle}>时间</th>
          </tr>
        </thead>

        <tbody>
          {items.map(item => {
            const modelUsed =
              item.model_used ||
              item.model_name ||
              ''

            const inputTokens =
              item.input_tokens ?? 0

            const outputTokens =
              item.output_tokens ?? 0

            const totalTokens =
              item.tokens_used ?? 0

            const costUSD =
              item.cost_usd ?? 0

            return (
              <tr key={item.id}>
                <td style={tdStyle}>
                  {item.user_name}
                </td>

                <td style={tdStyle}>
                  <span style={{
                    fontWeight: 600,
                    color: C.red,
                  }}>
                    -{item.amount.toLocaleString(
                      'zh-CN',
                      {
                        maximumFractionDigits: 4,
                      },
                    )}
                  </span>
                </td>

                <td style={tdStyle}>
                  <span style={{
                    display: 'inline-flex',
                    fontSize: '12px',
                    padding: '3px 8px',
                    borderRadius: '999px',
                    background: 'rgba(139,92,246,0.08)',
                    color: C.purple,
                    whiteSpace: 'nowrap',
                  }}>
                    {resolveConsumptionBusinessName(
                      item,
                    )}
                  </span>
                </td>

                <td style={{
                  ...tdStyle,
                  whiteSpace: 'nowrap',
                }}>
                  {resolveConsumptionUsage(
                    item,
                  )}
                </td>

                {isSuperAdmin && (
                  <td style={{
                    ...tdStyle,
                    fontSize: '12px',
                  }}>
                    {resolveAlias && modelUsed
                      ? resolveAlias(modelUsed)
                      : '…'}
                  </td>
                )}

                {isSuperAdmin && (
                  <td style={tdStyle}>
                    <span style={{
                      fontSize: '12px',
                    }}>
                      {inputTokens > 0
                        ? `${inputTokens.toLocaleString()} / ${outputTokens.toLocaleString()}`
                        : totalTokens.toLocaleString()}
                    </span>
                  </td>
                )}

                {isSuperAdmin && (
                  <td style={tdStyle}>
                    <span style={{
                      fontSize: '12px',
                      color: C.textSec,
                    }}>
                      {costUSD > 0
                        ? `$${costUSD.toFixed(6)}`
                        : '-'}
                    </span>
                  </td>
                )}

                <td style={{
                  ...tdStyle,
                  fontSize: '12px',
                  color: C.textSec,
                  whiteSpace: 'nowrap',
                }}>
                  {item.balance_before.toLocaleString(
                    'zh-CN',
                    {
                      maximumFractionDigits: 4,
                    },
                  )}
                  {' → '}
                  {item.balance_after.toLocaleString(
                    'zh-CN',
                    {
                      maximumFractionDigits: 4,
                    },
                  )}
                </td>

                <td style={{
                  ...tdStyle,
                  fontSize: '12px',
                  color: C.textSec,
                  whiteSpace: 'nowrap',
                }}>
                  {new Date(
                    item.created_at,
                  ).toLocaleString(
                    'zh-CN',
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

// ==================== 通用弹窗组件 ====================
export function ModalOverlay({ onClose, title, children }: { onClose: () => void; title: string; children: React.ReactNode }) {
  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 9999, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)' }} />
      <div style={{ position: 'relative', background: C.white, borderRadius: '16px', padding: '24px', width: '460px', maxHeight: '80vh', overflowY: 'auto', boxShadow: '0 20px 60px rgba(0,0,0,0.15)' }}>
        <div style={{ fontSize: '18px', fontWeight: 700, color: C.text, marginBottom: '20px' }}>{title}</div>
        {children}
      </div>
    </div>
  )
}

export function FormField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div style={{ marginBottom: '12px' }}>
      <div style={{ fontSize: '13px', color: C.textSec, marginBottom: '6px', fontWeight: 500 }}>{label}</div>
      {children}
    </div>
  )
}

// ==================== 创建账户弹窗 ====================
export function CreateAccountModal({ onClose, onSuccess }: { onClose: () => void; onSuccess: () => void }) {
  const [accountType, setAccountType] = useState('school')
  const [ownerId, setOwnerId] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [monthlyQuota, setMonthlyQuota] = useState(0)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    if (!ownerId || !displayName) { setError('请填写完整信息'); return }
    setSubmitting(true); setError('')
    try {
      await createTokenAccount({ account_type: accountType, owner_id: ownerId, display_name: displayName, monthly_quota: monthlyQuota })
      onSuccess()
    } catch (e: unknown) { setError(e instanceof Error ? e.message : '创建失败') }
    setSubmitting(false)
  }

  return (
    <ModalOverlay onClose={onClose} title="创建积分账户">
      <FormField label="账户类型">
        <select value={accountType} onChange={e => setAccountType(e.target.value)} style={inputStyle}>
          {ACCOUNT_TYPE_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
      </FormField>
      <FormField label="关联实体ID"><input value={ownerId} onChange={e => setOwnerId(e.target.value)} placeholder="组织ID或用户ID" style={inputStyle} /></FormField>
      <FormField label="账户名称"><input value={displayName} onChange={e => setDisplayName(e.target.value)} placeholder="如：PKU AI实验学校" style={inputStyle} /></FormField>
      <FormField label="月度配额"><input type="number" value={monthlyQuota} onChange={e => setMonthlyQuota(Number(e.target.value))} placeholder="0表示不自动充值" style={inputStyle} /></FormField>
      {error && <div style={{ color: C.red, fontSize: '13px', marginTop: '8px' }}>{error}</div>}
      <div style={{ display: 'flex', gap: '8px', marginTop: '16px', justifyContent: 'flex-end' }}>
        <button onClick={onClose} style={cancelBtnStyle}>取消</button>
        <button onClick={handleSubmit} disabled={submitting} style={submitBtnStyle}>{submitting ? '创建中...' : '创建'}</button>
      </div>
    </ModalOverlay>
  )
}

// ==================== 采购/充值弹窗（批次2：接入 AccountPicker mode="all"）====================
export function PurchaseModal({ onClose, onSuccess }: { onClose: () => void; onSuccess: () => void }) {
  const [accountId, setAccountId] = useState('')
  const [amount, setAmount] = useState(0)
  const [purchaseType, setPurchaseType] = useState('purchase')
  const [memo, setMemo] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    if (!accountId || amount <= 0) { setError('请选择账户并填写积分数量'); return }
    setSubmitting(true); setError('')
    try {
      await purchaseTokens({ account_id: accountId, amount, purchase_type: purchaseType, memo })
      onSuccess()
    } catch (e: unknown) { setError(e instanceof Error ? e.message : '充值失败') }
    setSubmitting(false)
  }

  return (
    <ModalOverlay onClose={onClose} title="充值积分">
      <FormField label="目标账户（可搜索/按类型筛选）">
        <AccountPicker
          mode="all"
          value={accountId}
          onChange={(id) => setAccountId(id)}
        />
      </FormField>
      <FormField label="充值积分数"><input type="number" value={amount || ''} onChange={e => setAmount(Number(e.target.value))} placeholder="请输入积分数量" style={inputStyle} /></FormField>
      <FormField label="充值类型">
        <select value={purchaseType} onChange={e => setPurchaseType(e.target.value)} style={inputStyle}>
          {PURCHASE_TYPE_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
      </FormField>
      <FormField label="备注"><input value={memo} onChange={e => setMemo(e.target.value)} placeholder="可选" style={inputStyle} /></FormField>
      {error && <div style={{ color: C.red, fontSize: '13px', marginTop: '8px' }}>{error}</div>}
      <div style={{ display: 'flex', gap: '8px', marginTop: '16px', justifyContent: 'flex-end' }}>
        <button onClick={onClose} style={cancelBtnStyle}>取消</button>
        <button onClick={handleSubmit} disabled={submitting} style={{ ...submitBtnStyle, background: C.green }}>{submitting ? '充值中...' : '充值'}</button>
      </div>
    </ModalOverlay>
  )
}

// ==================== 分配积分弹窗（批次2：接入 AccountPicker mode="children" 究极版）====================
// 目标账户只显示"来源账户的合法下级"（后端 getAllocatableTargets + 三重防越权），
// 从根本上防止选到非下级账户导致分配失败。
export function AllocateModal({ fromAccountId, fromAccountName, fromAccountBalance, onClose, onSuccess }: { fromAccountId: string; fromAccountName?: string; fromAccountBalance?: number; onClose: () => void; onSuccess: () => void }) {
  const [toAccountId, setToAccountId] = useState('')
  const [amount, setAmount] = useState(0)
  const [memo, setMemo] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    if (!toAccountId || amount <= 0) { setError('请选择目标账户并填写积分数量'); return }
    setSubmitting(true); setError('')
    try {
      await allocateTokens(fromAccountId, { to_account_id: toAccountId, amount, memo })
      onSuccess()
    } catch (e: unknown) { setError(e instanceof Error ? e.message : '分配失败') }
    setSubmitting(false)
  }

  return (
    <ModalOverlay onClose={onClose} title="分配积分">
      <div style={{ padding: '12px', background: C.primaryLight, borderRadius: '8px', marginBottom: '12px', fontSize: '13px' }}>
        来源：<strong>{fromAccountName || '当前账户'}</strong>
        {typeof fromAccountBalance === 'number' && <>（可用余额：{fromAccountBalance.toLocaleString()} 积分）</>}
      </div>
      <FormField label="目标账户（仅显示该账户的下级）">
        <AccountPicker
          mode="children"
          fromAccountId={fromAccountId}
          value={toAccountId}
          onChange={(id) => setToAccountId(id)}
        />
      </FormField>
      <FormField label="分配积分数"><input type="number" value={amount || ''} onChange={e => setAmount(Number(e.target.value))} placeholder="请输入积分数量" style={inputStyle} /></FormField>
      <FormField label="备注"><input value={memo} onChange={e => setMemo(e.target.value)} placeholder="可选" style={inputStyle} /></FormField>
      {error && <div style={{ color: C.red, fontSize: '13px', marginTop: '8px' }}>{error}</div>}
      <div style={{ display: 'flex', gap: '8px', marginTop: '16px', justifyContent: 'flex-end' }}>
        <button onClick={onClose} style={cancelBtnStyle}>取消</button>
        <button onClick={handleSubmit} disabled={submitting} style={submitBtnStyle}>{submitting ? '分配中...' : '分配'}</button>
      </div>
    </ModalOverlay>
  )
}
