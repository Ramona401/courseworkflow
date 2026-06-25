/**
 * tokenDashboardPricing.tsx — 积分策略 Tab（PricingTab）
 *
 * 从 tokenDashboardParts.tsx 拆出（超 600 行红线拆分）：
 *   原 tokenDashboardParts 因聚合了「汇率倍率编辑 + 模型单价表增删改」两大块逼近 763 行，
 *   PricingTab 是其中最大且最独立的一块，整体搬出。
 *
 * 依赖：
 *   - 共享样式常量 C/cardStyle/thStyle/tdStyle/inputStyle 仍从 ./tokenDashboardParts import（单一真相源）
 *   - 自管 API：getModelPrices/getSystemCreditPolicy/simulateCredits/updateSystemCreditPolicy
 *     /createModelPrice/updateModelPrice/deleteModelPrice + PROVIDER_COLORS
 *
 * 仅被 TokenDashboardPage 在 tab==='pricing' 时渲染；admin 专属（路由+后端 adminOnly 双控）。
 * 纯位置搬迁，逻辑零变更。
 */
import { useState, useEffect } from 'react'
import {
  getModelPrices, getSystemCreditPolicy, simulateCredits, updateSystemCreditPolicy,
  createModelPrice, updateModelPrice, deleteModelPrice,
  type ModelPrice, type CreditPolicy, type CreditCalculation, type CreateModelPriceRequest,
  PROVIDER_COLORS,
} from '@/api/tokens'
import { C, cardStyle, thStyle, tdStyle, inputStyle } from './tokenDashboardParts'

// ==================== 积分策略Tab ====================
export function PricingTab() {
  const [policy, setPolicy] = useState<CreditPolicy | null>(null)
  const [prices, setPrices] = useState<ModelPrice[]>([])
  const [simResult, setSimResult] = useState<CreditCalculation | null>(null)
  const [simModel, setSimModel] = useState('')
  const [simInput, setSimInput] = useState(1000)
  const [simOutput, setSimOutput] = useState(500)
  const [loading, setLoading] = useState(true)

  // 汇率/倍率编辑态（仅 admin，PricingTab 本就 admin 专属）
  const [rateEditing, setRateEditing] = useState(false)
  const [rateInput, setRateInput] = useState(7)
  const [multiplierInput, setMultiplierInput] = useState(1)
  const [rateSaving, setRateSaving] = useState(false)
  const [rateMsg, setRateMsg] = useState('')

  const reload = async () => {
    setLoading(true)
    try {
      const [p, m] = await Promise.all([getSystemCreditPolicy(), getModelPrices()])
      setPolicy(p)
      setPrices(m || [])
      if (m && m.length > 0) setSimModel(m[0].model_name)
    } catch { /* ignore */ }
    setLoading(false)
  }

  useEffect(() => { reload() }, [])

  // ===== 模型单价表 编辑/新增/删除 state（批·国内模型积分：可编辑不写死）=====
  const [priceAdding, setPriceAdding] = useState(false)
  const [priceForm, setPriceForm] = useState<CreateModelPriceRequest>({
    model_name: '', provider: 'qwen', cost_per_1k_input: 0, cost_per_1k_output: 0, display_name: '',
  })
  const [priceSaving, setPriceSaving] = useState(false)
  const [priceMsg, setPriceMsg] = useState('')
  const [editPriceId, setEditPriceId] = useState<string | null>(null)
  const [editPriceForm, setEditPriceForm] = useState<{ cost_per_1k_input: number; cost_per_1k_output: number; display_name: string }>({
    cost_per_1k_input: 0, cost_per_1k_output: 0, display_name: '',
  })
  const [confirmDelPrice, setConfirmDelPrice] = useState<{ id: string; name: string } | null>(null)

  // 新增单价
  const handleAddPrice = async () => {
    if (!priceForm.model_name.trim()) { setPriceMsg('模型名(model_name)不能为空，须与实际调用模型名一致'); return }
    if (!(priceForm.cost_per_1k_input >= 0) || !(priceForm.cost_per_1k_output >= 0)) { setPriceMsg('单价不能为负'); return }
    try {
      setPriceSaving(true); setPriceMsg('')
      await createModelPrice({
        model_name: priceForm.model_name.trim(),
        provider: (priceForm.provider || '').trim() || 'unknown',
        cost_per_1k_input: priceForm.cost_per_1k_input,
        cost_per_1k_output: priceForm.cost_per_1k_output,
        display_name: (priceForm.display_name || '').trim(),
      })
      setPriceAdding(false)
      setPriceForm({ model_name: '', provider: 'qwen', cost_per_1k_input: 0, cost_per_1k_output: 0, display_name: '' })
      await reload()
    } catch (e: unknown) {
      setPriceMsg(e instanceof Error ? e.message : '新增失败')
    } finally {
      setPriceSaving(false)
    }
  }

  // 进入行内编辑
  const startEditPrice = (p: ModelPrice) => {
    setEditPriceId(p.id)
    setEditPriceForm({ cost_per_1k_input: p.cost_per_1k_input, cost_per_1k_output: p.cost_per_1k_output, display_name: p.display_name || '' })
  }
  // 保存行内编辑
  const handleSavePrice = async (id: string) => {
    try {
      setPriceSaving(true)
      await updateModelPrice(id, {
        cost_per_1k_input: editPriceForm.cost_per_1k_input,
        cost_per_1k_output: editPriceForm.cost_per_1k_output,
        display_name: editPriceForm.display_name.trim(),
      })
      setEditPriceId(null)
      await reload()
    } catch { /* ignore */ }
    finally { setPriceSaving(false) }
  }
  // 删除
  const handleDeletePrice = async (id: string) => {
    try {
      await deleteModelPrice(id)
      setConfirmDelPrice(null)
      await reload()
    } catch { setConfirmDelPrice(null) }
  }

  // 进入编辑态时用当前值预填输入框
  const startRateEdit = () => {
    setRateInput(policy?.exchange_rate ?? 7)
    setMultiplierInput(policy?.multiplier ?? 1)
    setRateMsg('')
    setRateEditing(true)
  }

  const handleRateSave = async () => {
    // 基本校验：必须为正数
    if (!(rateInput > 0) || !(multiplierInput > 0)) {
      setRateMsg('汇率与倍率必须为正数')
      return
    }
    try {
      setRateSaving(true)
      setRateMsg('')
      const updated = await updateSystemCreditPolicy({ exchange_rate: rateInput, multiplier: multiplierInput })
      setPolicy(updated)
      setRateEditing(false)
    } catch (e: unknown) {
      setRateMsg(e instanceof Error ? e.message : '保存失败')
    } finally {
      setRateSaving(false)
    }
  }

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
      <div style={{ ...cardStyle }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
          <span style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>💱 积分汇率与倍率</span>
          {!rateEditing && (
            <button onClick={startRateEdit}
              style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.primary}`, background: C.white, color: C.primary, fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
              ✏️ 编辑
            </button>
          )}
        </div>

        {!rateEditing ? (
          /* 只读态 */
          <div style={{ display: 'flex', gap: '32px', alignItems: 'center', flexWrap: 'wrap' }}>
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
        ) : (
          /* 编辑态 */
          <div>
            <div style={{ padding: '10px 14px', borderRadius: '8px', background: 'rgba(245,158,11,0.08)', border: `1px solid ${C.orange}`, color: C.orange, fontSize: '12px', marginBottom: '14px', lineHeight: 1.6 }}>
              ⚠️ 修改后<b>立即生效</b>，影响此后所有 AI 调用的积分计算（积分 = 美元成本 × 汇率 × 倍率）。历史消费记录不受影响。
            </div>
            <div style={{ display: 'flex', gap: '20px', alignItems: 'flex-end', flexWrap: 'wrap' }}>
              <div>
                <div style={{ fontSize: '12px', color: C.textSec, marginBottom: '4px' }}>💱 汇率（美元→积分）</div>
                <input type="number" step="0.01" min="0" value={rateInput}
                  onChange={e => setRateInput(Number(e.target.value))}
                  style={{ ...inputStyle, width: '140px', fontSize: '18px', fontWeight: 700 }} />
              </div>
              <div style={{ fontSize: '24px', color: C.textMuted, paddingBottom: '6px' }}>×</div>
              <div>
                <div style={{ fontSize: '12px', color: C.textSec, marginBottom: '4px' }}>📊 倍率</div>
                <input type="number" step="0.01" min="0" value={multiplierInput}
                  onChange={e => setMultiplierInput(Number(e.target.value))}
                  style={{ ...inputStyle, width: '140px', fontSize: '18px', fontWeight: 700 }} />
              </div>
              <div style={{ fontSize: '24px', color: C.textMuted, paddingBottom: '6px' }}>=</div>
              <div style={{ paddingBottom: '2px' }}>
                <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '4px' }}>有效汇率（预览）</div>
                <div style={{ fontSize: '26px', fontWeight: 700, color: C.green }}>{(rateInput * multiplierInput).toFixed(2)}</div>
              </div>
            </div>
            {rateMsg && (
              <div style={{ fontSize: '12px', color: C.red, marginTop: '10px' }}>⚠ {rateMsg}</div>
            )}
            <div style={{ display: 'flex', gap: '10px', marginTop: '16px' }}>
              <button onClick={handleRateSave} disabled={rateSaving}
                style={{ padding: '9px 20px', borderRadius: '8px', border: 'none', background: rateSaving ? C.textMuted : C.primary, color: C.white, fontSize: '14px', fontWeight: 600, cursor: rateSaving ? 'not-allowed' : 'pointer' }}>
                {rateSaving ? '保存中...' : '💾 保存'}
              </button>
              <button onClick={() => setRateEditing(false)} disabled={rateSaving}
                style={{ padding: '9px 20px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, color: C.textSec, fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>
                取消
              </button>
            </div>
          </div>
        )}
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
          <span style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ fontSize: '12px', color: C.textMuted }}>{prices.length} 个模型</span>
            <button onClick={() => { setPriceAdding(v => !v); setPriceMsg('') }}
              style={{ padding: '6px 14px', borderRadius: '8px', border: 'none', background: priceAdding ? C.textMuted : C.green, color: C.white, fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
              {priceAdding ? '取消' : '+ 新增单价'}
            </button>
          </span>
        </div>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead><tr>
            <th style={thStyle}>模型</th><th style={thStyle}>供应商</th>
            <th style={thStyle}>输入 ($/1K)</th><th style={thStyle}>输出 ($/1K)</th>
            <th style={thStyle}>输入 (积分/1K)</th><th style={thStyle}>输出 (积分/1K)</th>
            <th style={thStyle}>状态</th><th style={thStyle}>操作</th>
          </tr></thead>
          <tbody>
            {priceAdding && (
              <tr style={{ background: 'rgba(16,185,129,0.05)' }}>
                <td style={tdStyle} colSpan={7}>
                  <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', alignItems: 'center' }}>
                    <input value={priceForm.model_name} onChange={e => setPriceForm(f => ({ ...f, model_name: e.target.value }))}
                      placeholder="model_name 如 qwen-max" style={{ ...inputStyle, width: '180px', fontFamily: 'monospace' }} />
                    <input value={priceForm.provider} onChange={e => setPriceForm(f => ({ ...f, provider: e.target.value }))}
                      placeholder="供应商 如 qwen" style={{ ...inputStyle, width: '100px' }} />
                    <input type="number" step="0.0001" min="0" value={priceForm.cost_per_1k_input} onChange={e => setPriceForm(f => ({ ...f, cost_per_1k_input: Number(e.target.value) }))}
                      placeholder="输入$/1K" title="输入单价 美元/1K" style={{ ...inputStyle, width: '110px' }} />
                    <input type="number" step="0.0001" min="0" value={priceForm.cost_per_1k_output} onChange={e => setPriceForm(f => ({ ...f, cost_per_1k_output: Number(e.target.value) }))}
                      placeholder="输出$/1K" title="输出单价 美元/1K" style={{ ...inputStyle, width: '110px' }} />
                    <input value={priceForm.display_name} onChange={e => setPriceForm(f => ({ ...f, display_name: e.target.value }))}
                      placeholder="显示名(可选)" style={{ ...inputStyle, width: '140px' }} />
                    <button onClick={handleAddPrice} disabled={priceSaving}
                      style={{ padding: '8px 16px', borderRadius: '8px', border: 'none', background: priceSaving ? C.textMuted : C.green, color: C.white, fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
                      {priceSaving ? '...' : '保存'}
                    </button>
                  </div>
                  {priceMsg && <div style={{ fontSize: '12px', color: C.red, marginTop: '8px' }}>⚠ {priceMsg}</div>}
                  <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '6px' }}>
                    ⚠ model_name 必须与实际调用的模型名<b>逐字一致</b>（如境内降级模型填 <code>qwen-max</code>），否则计费时匹配不到。
                  </div>
                </td>
              </tr>
            )}
            {prices.map(p => {
              const rate = (policy?.exchange_rate ?? 7) * (policy?.multiplier ?? 1)
              return (
                <tr key={p.id}>
                  <td style={tdStyle}><span style={{ fontWeight: 600 }}>{p.display_name || p.model_name}</span><br/><span style={{ fontSize: '11px', color: C.textMuted }}>{p.model_name}</span></td>
                  <td style={tdStyle}><span style={{ fontSize: '12px', padding: '2px 8px', borderRadius: '4px', background: `${PROVIDER_COLORS[p.provider] || '#9CA3AF'}15`, color: PROVIDER_COLORS[p.provider] || '#9CA3AF' }}>{p.provider}</span></td>
                  <td style={tdStyle}>{editPriceId === p.id
                    ? <input type="number" step="0.0001" min="0" value={editPriceForm.cost_per_1k_input} onChange={e => setEditPriceForm(f => ({ ...f, cost_per_1k_input: Number(e.target.value) }))} style={{ ...inputStyle, width: '90px' }} />
                    : `$${p.cost_per_1k_input.toFixed(4)}`}</td>
                  <td style={tdStyle}>{editPriceId === p.id
                    ? <input type="number" step="0.0001" min="0" value={editPriceForm.cost_per_1k_output} onChange={e => setEditPriceForm(f => ({ ...f, cost_per_1k_output: Number(e.target.value) }))} style={{ ...inputStyle, width: '90px' }} />
                    : `$${p.cost_per_1k_output.toFixed(4)}`}</td>
                  <td style={tdStyle}><span style={{ fontWeight: 600, color: C.primary }}>{(p.cost_per_1k_input * rate).toFixed(4)}</span></td>
                  <td style={tdStyle}><span style={{ fontWeight: 600, color: C.primary }}>{(p.cost_per_1k_output * rate).toFixed(4)}</span></td>
                  <td style={tdStyle}><span style={{ color: p.is_active ? C.green : C.textMuted }}>{p.is_active ? '✓ 启用' : '✗ 禁用'}</span></td>
                  <td style={tdStyle}>
                    {editPriceId === p.id ? (
                      <span style={{ display: 'flex', gap: '4px' }}>
                        <button onClick={() => handleSavePrice(p.id)} disabled={priceSaving} style={{ padding: '4px 10px', borderRadius: '6px', border: 'none', background: C.green, color: C.white, fontSize: '12px', cursor: 'pointer' }}>保存</button>
                        <button onClick={() => setEditPriceId(null)} style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: C.white, color: C.textSec, fontSize: '12px', cursor: 'pointer' }}>取消</button>
                      </span>
                    ) : (
                      <span style={{ display: 'flex', gap: '4px' }}>
                        <button onClick={() => startEditPrice(p)} style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: C.white, color: C.textSec, fontSize: '12px', cursor: 'pointer' }}>编辑</button>
                        <button onClick={() => setConfirmDelPrice({ id: p.id, name: p.display_name || p.model_name })} style={{ padding: '4px 10px', borderRadius: '6px', border: '1px solid #FEE2E2', background: '#FEF2F2', color: '#EF4444', fontSize: '12px', cursor: 'pointer' }}>删除</button>
                      </span>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {confirmDelPrice && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 9999 }}>
          <div style={{ background: C.white, borderRadius: '12px', padding: '24px', maxWidth: '380px', width: '90%' }}>
            <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '10px' }}>删除模型单价</div>
            <div style={{ fontSize: '13px', color: C.textSec, lineHeight: 1.6, marginBottom: '20px' }}>
              确认删除「{confirmDelPrice.name}」的单价？删除后该模型调用将走兜底计费（每1K token 计 1 积分，无美元成本）。
            </div>
            <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
              <button onClick={() => setConfirmDelPrice(null)} style={{ padding: '8px 18px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, color: C.textSec, fontSize: '14px', cursor: 'pointer' }}>取消</button>
              <button onClick={() => handleDeletePrice(confirmDelPrice.id)} style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: C.red, color: C.white, fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>删除</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
