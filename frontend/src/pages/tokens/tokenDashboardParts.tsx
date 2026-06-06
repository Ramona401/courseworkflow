/**
 * tokenDashboardParts.tsx — 积分管理页子组件与共享样式
 *
 * v172 拆分：从 TokenDashboardPage.tsx 抽出表格 / 弹窗 / 积分策略Tab，
 * 使主页面在新增三级权限角色逻辑后仍保持 <600 行。
 *
 * 导出：
 *   - 共享样式常量 C / cardStyle / tabBtnStyle / thStyle / tdStyle / inputStyle 等
 *   - StatCard
 *   - AccountsTable / AllocationsTable / PurchasesTable / ConsumptionTable
 *   - CreateAccountModal / PurchaseModal / AllocateModal
 *   - PricingTab
 */
import { useState, useEffect } from 'react'
import {
  createTokenAccount, purchaseTokens, allocateTokens,
  getModelPrices, getSystemCreditPolicy, simulateCredits,
  type TokenAccountListItem, type AllocationListItem,
  type PurchaseListItem, type ConsumptionListItem,
  type ModelPrice, type CreditPolicy, type CreditCalculation,
  ACCOUNT_TYPE_OPTIONS, PURCHASE_TYPE_OPTIONS, ACCOUNT_STATUS_COLORS,
  SCENE_CODE_LABELS, PROVIDER_COLORS,
} from '@/api/tokens'

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
export function AllocationsTable({ items }: { items: AllocationListItem[] }) {
  if (!items?.length) return <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px' }}>暂无分配记录</div>
  return (
    <div style={{ background: C.white, borderRadius: '12px', border: `1px solid ${C.border}`, overflow: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead><tr>
          <th style={thStyle}>来源账户</th><th style={thStyle}>目标账户</th><th style={thStyle}>积分数</th>
          <th style={thStyle}>类型</th><th style={thStyle}>操作人</th><th style={thStyle}>备注</th><th style={thStyle}>时间</th>
        </tr></thead>
        <tbody>
          {items.map(a => (
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
export function ConsumptionTable({ items }: { items: ConsumptionListItem[] }) {
  if (!items?.length) return <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px' }}>暂无消费记录</div>
  return (
    <div style={{ background: C.white, borderRadius: '12px', border: `1px solid ${C.border}`, overflow: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead><tr>
          <th style={thStyle}>用户</th><th style={thStyle}>消费积分</th><th style={thStyle}>场景</th>
          <th style={thStyle}>模型</th><th style={thStyle}>Token(入/出)</th><th style={thStyle}>美元成本</th><th style={thStyle}>余额变化</th><th style={thStyle}>时间</th>
        </tr></thead>
        <tbody>
          {items.map(c => (
            <tr key={c.id}>
              <td style={tdStyle}>{c.user_name}</td>
              <td style={tdStyle}><span style={{ fontWeight: 600, color: C.red }}>-{c.amount.toLocaleString()}</span></td>
              <td style={tdStyle}><span style={{ fontSize: '12px', padding: '2px 6px', borderRadius: '4px', background: 'rgba(139,92,246,0.08)', color: C.purple }}>{SCENE_CODE_LABELS[c.scene_code] || c.scene_code}</span></td>
              <td style={{ ...tdStyle, fontSize: '12px' }}>{c.model_used}</td>
              <td style={tdStyle}><span style={{ fontSize: '12px' }}>{c.input_tokens > 0 ? `${c.input_tokens.toLocaleString()} / ${c.output_tokens.toLocaleString()}` : c.tokens_used.toLocaleString()}</span></td>
              <td style={tdStyle}><span style={{ fontSize: '12px', color: C.textSec }}>{c.cost_usd > 0 ? `$${c.cost_usd.toFixed(6)}` : '-'}</span></td>
              <td style={{ ...tdStyle, fontSize: '12px', color: C.textSec }}>{c.balance_before.toLocaleString()} → {c.balance_after.toLocaleString()}</td>
              <td style={{ ...tdStyle, fontSize: '12px', color: C.textSec }}>{new Date(c.created_at).toLocaleString('zh-CN')}</td>
            </tr>
          ))}
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

// ==================== 采购/充值弹窗 ====================
export function PurchaseModal({ accounts, onClose, onSuccess }: { accounts: TokenAccountListItem[]; onClose: () => void; onSuccess: () => void }) {
  const [accountId, setAccountId] = useState(accounts[0]?.id || '')
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
      <FormField label="目标账户">
        <select value={accountId} onChange={e => setAccountId(e.target.value)} style={inputStyle}>
          {accounts.map(a => <option key={a.id} value={a.id}>{a.display_name} ({a.account_type_name})</option>)}
        </select>
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

// ==================== 分配积分弹窗 ====================
export function AllocateModal({ fromAccountId, accounts, onClose, onSuccess }: { fromAccountId: string; accounts: TokenAccountListItem[]; onClose: () => void; onSuccess: () => void }) {
  const fromAcc = accounts.find(a => a.id === fromAccountId)
  const childAccounts = accounts.filter(a => a.id !== fromAccountId)
  const [toAccountId, setToAccountId] = useState(childAccounts[0]?.id || '')
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
        来源：<strong>{fromAcc?.display_name}</strong>（可用余额：{fromAcc?.available_balance.toLocaleString()} 积分）
      </div>
      <FormField label="目标账户">
        <select value={toAccountId} onChange={e => setToAccountId(e.target.value)} style={inputStyle}>
          {childAccounts.map(a => <option key={a.id} value={a.id}>{a.display_name} ({a.account_type_name})</option>)}
        </select>
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

// ==================== 积分策略Tab ====================
export function PricingTab() {
  const [policy, setPolicy] = useState<CreditPolicy | null>(null)
  const [prices, setPrices] = useState<ModelPrice[]>([])
  const [simResult, setSimResult] = useState<CreditCalculation | null>(null)
  const [simModel, setSimModel] = useState('')
  const [simInput, setSimInput] = useState(1000)
  const [simOutput, setSimOutput] = useState(500)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const load = async () => {
      setLoading(true)
      try {
        const [p, m] = await Promise.all([getSystemCreditPolicy(), getModelPrices()])
        setPolicy(p)
        setPrices(m || [])
        if (m && m.length > 0) setSimModel(m[0].model_name)
      } catch { /* ignore */ }
      setLoading(false)
    }
    load()
  }, [])

  const handleSimulate = async () => {
    if (!simModel) return
    try {
      const r = await simulateCredits({ model_name: simModel, input_tokens: simInput, output_tokens: simOutput })
      setSimResult(r)
    } catch { /* ignore */ }
  }

  if (loading) return <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px' }}>加载中...</div>

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
      <div style={{ ...cardStyle, display: 'flex', gap: '32px', alignItems: 'center', flexWrap: 'wrap' }}>
        <div>
          <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '4px' }}>💱 汇率（美元→积分）</div>
          <div style={{ fontSize: '28px', fontWeight: 700, color: C.primary }}>{policy?.exchange_rate ?? 7}</div>
        </div>
        <div style={{ fontSize: '24px', color: C.textMuted }}>×</div>
        <div>
          <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '4px' }}>📊 倍率</div>
          <div style={{ fontSize: '28px', fontWeight: 700, color: C.purple }}>{policy?.multiplier ?? 1}</div>
        </div>
        <div style={{ fontSize: '24px', color: C.textMuted }}>=</div>
        <div>
          <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '4px' }}>有效汇率</div>
          <div style={{ fontSize: '28px', fontWeight: 700, color: C.green }}>{((policy?.exchange_rate ?? 7) * (policy?.multiplier ?? 1)).toFixed(2)}</div>
        </div>
        <div style={{ flex: 1, fontSize: '13px', color: C.textSec, minWidth: '200px' }}>
          积分 = 美元成本 × {policy?.exchange_rate ?? 7} × {policy?.multiplier ?? 1}<br/>
          <span style={{ fontSize: '12px', color: C.textMuted }}>{policy?.description || ''}</span>
        </div>
      </div>

      <div style={{ ...cardStyle }}>
        <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '12px' }}>🧮 积分模拟计算器</div>
        <div style={{ display: 'flex', gap: '12px', alignItems: 'flex-end', flexWrap: 'wrap' }}>
          <div>
            <div style={{ fontSize: '12px', color: C.textSec, marginBottom: '4px' }}>模型</div>
            <select value={simModel} onChange={e => setSimModel(e.target.value)} style={{ ...inputStyle, width: '260px' }}>
              {prices.filter(p => p.is_active).map(p => <option key={p.id} value={p.model_name}>{p.display_name || p.model_name}</option>)}
            </select>
          </div>
          <div>
            <div style={{ fontSize: '12px', color: C.textSec, marginBottom: '4px' }}>输入Tokens</div>
            <input type="number" value={simInput} onChange={e => setSimInput(Number(e.target.value))} style={{ ...inputStyle, width: '120px' }} />
          </div>
          <div>
            <div style={{ fontSize: '12px', color: C.textSec, marginBottom: '4px' }}>输出Tokens</div>
            <input type="number" value={simOutput} onChange={e => setSimOutput(Number(e.target.value))} style={{ ...inputStyle, width: '120px' }} />
          </div>
          <button onClick={handleSimulate} style={{ padding: '10px 20px', borderRadius: '8px', border: 'none', background: C.primary, color: C.white, cursor: 'pointer', fontWeight: 600, fontSize: '14px' }}>计算</button>
        </div>
        {simResult && (
          <div style={{ marginTop: '16px', padding: '16px', background: 'rgba(79,123,232,0.04)', borderRadius: '12px', display: 'flex', gap: '24px', flexWrap: 'wrap' }}>
            <div><span style={{ fontSize: '12px', color: C.textMuted }}>美元成本</span><div style={{ fontSize: '18px', fontWeight: 600, color: C.text }}>${simResult.cost_usd.toFixed(6)}</div></div>
            <div><span style={{ fontSize: '12px', color: C.textMuted }}>× 汇率 {simResult.exchange_rate} × 倍率 {simResult.multiplier}</span><div style={{ fontSize: '18px', fontWeight: 600, color: C.text }}>=</div></div>
            <div><span style={{ fontSize: '12px', color: C.textMuted }}>积分消耗</span><div style={{ fontSize: '24px', fontWeight: 700, color: C.primary }}>{simResult.credits_consumed.toFixed(4)} 积分</div></div>
          </div>
        )}
      </div>

      <div style={{ background: C.white, borderRadius: '12px', border: `1px solid ${C.border}`, overflow: 'auto' }}>
        <div style={{ padding: '16px 20px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>📋 模型单价表（每1K Tokens）</span>
          <span style={{ fontSize: '12px', color: C.textMuted }}>{prices.length} 个模型</span>
        </div>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead><tr>
            <th style={thStyle}>模型</th><th style={thStyle}>供应商</th>
            <th style={thStyle}>输入 ($/1K)</th><th style={thStyle}>输出 ($/1K)</th>
            <th style={thStyle}>输入 (积分/1K)</th><th style={thStyle}>输出 (积分/1K)</th>
            <th style={thStyle}>状态</th>
          </tr></thead>
          <tbody>
            {prices.map(p => {
              const rate = (policy?.exchange_rate ?? 7) * (policy?.multiplier ?? 1)
              return (
                <tr key={p.id}>
                  <td style={tdStyle}><span style={{ fontWeight: 600 }}>{p.display_name || p.model_name}</span><br/><span style={{ fontSize: '11px', color: C.textMuted }}>{p.model_name}</span></td>
                  <td style={tdStyle}><span style={{ fontSize: '12px', padding: '2px 8px', borderRadius: '4px', background: `${PROVIDER_COLORS[p.provider] || '#9CA3AF'}15`, color: PROVIDER_COLORS[p.provider] || '#9CA3AF' }}>{p.provider}</span></td>
                  <td style={tdStyle}>${p.cost_per_1k_input.toFixed(4)}</td>
                  <td style={tdStyle}>${p.cost_per_1k_output.toFixed(4)}</td>
                  <td style={tdStyle}><span style={{ fontWeight: 600, color: C.primary }}>{(p.cost_per_1k_input * rate).toFixed(4)}</span></td>
                  <td style={tdStyle}><span style={{ fontWeight: 600, color: C.primary }}>{(p.cost_per_1k_output * rate).toFixed(4)}</span></td>
                  <td style={tdStyle}><span style={{ color: p.is_active ? C.green : C.textMuted }}>{p.is_active ? '✓ 启用' : '✗ 禁用'}</span></td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
