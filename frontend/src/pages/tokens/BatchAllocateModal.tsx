/**
 * BatchAllocateModal — 一键批量分配弹窗（一键分配 batch 新增）
 *
 * 场景：区域管理员向辖区几十所学校、学校管理员向全校老师分配积分时，
 * 逐个打开单笔分配弹窗极其低效。本弹窗一次勾选多个目标 + 每户定额，一键完成。
 *
 * 交互流程：
 *   1. 打开即调 getAllocatableTargets(fromAccountId) 拉取"合法下级账户"
 *      （后端按 scope 收窄：区域账户→管辖学校；学校账户→本校成员个人账户）。
 *   2. 复选框列表勾选目标；支持关键词过滤（纯前端过滤，下级通常≤几百个）；
 *      「全选当前列表」只选中过滤后可见项、「清空」清掉全部勾选。
 *   3. 输入每户金额（每户定额模式：每个目标分同样金额，总额=每户×N）；
 *      实时显示"N 个目标 × X 积分 = 总计 Y 积分"，超出来源可用余额时红字预警并禁用提交。
 *   4. 提交调 batchAllocateTokens：
 *      - 后端预检失败（余额不足/超500上限等）→ HTTP 400，一笔未分，红字展示原因可修正重试；
 *      - 进入逐笔执行 → 恒 200，切换到结果视图：成功数/失败数/实际分出总额/失败明细
 *        （失败明细按目标ID回查列表映射成账户名，逐条展示原因）。
 *   5. 结果视图点「完成」→ onSuccess（父组件刷新列表/统计/区域卡片余额）。
 *
 * 设计说明：
 *   - Props 签名刻意与既有 AllocateModal 完全一致（fromAccountId/Name/Balance + onClose/onSuccess），
 *     父组件用同一份 allocateFrom 状态即可在"单个/批量"两个弹窗间切换，零额外接线。
 *   - 不修改 tokenDashboardParts.tsx / RegionAccountsCard.tsx（共享文件零改动，回归面最小）；
 *     样式常量 C 与 inputStyle 从 parts 复用，视觉与既有弹窗一致。
 *   - 前端上限与后端 maxBatchAllocateTargets(500) 同口径，超限直接禁用提交并提示。
 */
import { useState, useEffect, useMemo } from 'react'
import {
  getAllocatableTargets, batchAllocateTokens,
  type TokenAccountListItem, type BatchAllocateResult,
} from '@/api/tokens'
import { C, inputStyle } from './tokenDashboardParts'

/** 单次批量分配目标上限（与后端 maxBatchAllocateTargets 同口径） */
const MAX_TARGETS = 500

interface Props {
  fromAccountId: string      // 来源账户ID
  fromAccountName: string    // 来源账户名（标题展示）
  fromAccountBalance: number // 来源账户可用余额（前端预警用；后端提交时二次校验）
  onClose: () => void        // 关闭（未提交/中途放弃）
  onSuccess: () => void      // 完成（结果视图点"完成"后触发，父组件刷新数据）
}

export default function BatchAllocateModal({ fromAccountId, fromAccountName, fromAccountBalance, onClose, onSuccess }: Props) {
  // ========== 目标列表 ==========
  const [targets, setTargets] = useState<TokenAccountListItem[]>([])
  const [loadingTargets, setLoadingTargets] = useState(true)
  const [loadError, setLoadError] = useState('')

  // ========== 勾选/输入 ==========
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [keyword, setKeyword] = useState('')       // 目标名称过滤（纯前端）
  const [amountEach, setAmountEach] = useState('') // 每户金额（受控字符串，提交时 parseFloat）
  const [memo, setMemo] = useState('')             // 备注（写入每笔分配流水）

  // ========== 提交/结果 ==========
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState('')
  const [result, setResult] = useState<BatchAllocateResult | null>(null)

  // 打开即拉取合法下级账户（后端 scope 收窄 + 三重防越权）
  useEffect(() => {
    let alive = true
    ;(async () => {
      setLoadingTargets(true)
      setLoadError('')
      try {
        const data = await getAllocatableTargets(fromAccountId)
        if (alive) setTargets(data?.items || [])
      } catch {
        if (alive) setLoadError('加载可分配目标失败，请关闭后重试')
      }
      if (alive) setLoadingTargets(false)
    })()
    return () => { alive = false }
  }, [fromAccountId])

  // 关键词过滤后的可见列表
  const filtered = useMemo(() => {
    const kw = keyword.trim()
    if (!kw) return targets
    return targets.filter(t => (t.display_name || '').includes(kw))
  }, [targets, keyword])

  // ========== 派生校验 ==========
  const amount = parseFloat(amountEach)
  const amountValid = !isNaN(amount) && amount > 0
  const count = selected.size
  const totalNeeded = amountValid ? amount * count : 0
  const overBudget = amountValid && count > 0 && totalNeeded > fromAccountBalance // 超出可用余额
  const overLimit = count > MAX_TARGETS                                           // 超出单次上限
  const canSubmit = amountValid && count > 0 && !overBudget && !overLimit && !submitting

  // 数字格式化（与主页面口径一致：小数值保留4位，大数千分位）
  const fmt = (n: number) => n < 10 && n > 0 ? n.toFixed(4) : n.toLocaleString(undefined, { maximumFractionDigits: 2 })

  // ========== 勾选操作 ==========
  const toggleOne = (id: string) => {
    setSelected(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }
  // 全选"当前过滤后可见"的目标（在已选基础上并入，方便分批搜索勾选）
  const selectAllFiltered = () => {
    setSelected(prev => {
      const next = new Set(prev)
      filtered.forEach(t => next.add(t.id))
      return next
    })
  }
  const clearAll = () => setSelected(new Set())

  // ========== 提交 ==========
  const handleSubmit = async () => {
    if (!canSubmit) return
    setSubmitting(true)
    setSubmitError('')
    try {
      const res = await batchAllocateTokens(fromAccountId, {
        to_account_ids: Array.from(selected),
        amount_each: amount,
        memo: memo.trim() || undefined,
      })
      setResult(res) // 切换到结果视图（部分失败也是 200，明细在 failures）
    } catch (e) {
      // 预检失败（余额不足/超上限等）→ 400 走这里，一笔未分，可修正后重试
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
      setSubmitError(msg || '批量分配请求失败，请稍后重试')
    }
    setSubmitting(false)
  }

  // 失败明细：目标ID → 账户名（回查目标列表映射，查不到时降级显示ID）
  const nameOf = (id: string) => targets.find(t => t.id === id)?.display_name || id

  // ========== 共享小样式 ==========
  const btnStyle = (bg: string, disabled = false): React.CSSProperties => ({
    padding: '10px 20px', borderRadius: '8px', border: 'none',
    cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.5 : 1,
    fontSize: '14px', fontWeight: 600, color: C.white, background: bg,
  })
  const ghostBtnStyle: React.CSSProperties = {
    padding: '6px 14px', borderRadius: '8px', cursor: 'pointer', fontSize: '13px',
    border: `1px solid ${C.border}`, background: 'transparent', color: C.textSec,
  }

  return (
    // 遮罩层：点击遮罩不关闭（防误触丢失勾选），只能点按钮关闭
    <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.45)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '20px' }}>
      <div style={{ width: 'min(680px, 94vw)', maxHeight: '88vh', overflowY: 'auto', background: C.white, border: `1px solid ${C.border}`, borderRadius: '16px', padding: '24px', boxShadow: '0 20px 60px rgba(0,0,0,0.25)' }}>

        {/* ========== 标题 ========== */}
        <div style={{ fontSize: '17px', fontWeight: 700, marginBottom: '4px' }}>⚡ 一键批量分配</div>
        <div style={{ fontSize: '13px', color: C.textMuted, marginBottom: '16px' }}>
          来源账户：{fromAccountName}　可用余额：<span style={{ color: C.green, fontWeight: 600 }}>{fmt(fromAccountBalance)}</span> 积分
        </div>

        {result ? (
          /* ==================== 结果视图 ==================== */
          <div>
            <div style={{ display: 'flex', gap: '16px', marginBottom: '16px', flexWrap: 'wrap' }}>
              <div style={{ fontSize: '14px' }}>✅ 成功：<span style={{ color: C.green, fontWeight: 700 }}>{result.success_count}</span> 笔</div>
              <div style={{ fontSize: '14px' }}>❌ 失败：<span style={{ color: result.fail_count > 0 ? C.red : C.textMuted, fontWeight: 700 }}>{result.fail_count}</span> 笔</div>
              <div style={{ fontSize: '14px' }}>💰 实际分出：<span style={{ fontWeight: 700 }}>{fmt(result.total_allocated)}</span> 积分</div>
            </div>
            {result.failures.length > 0 && (
              <div style={{ marginBottom: '16px', padding: '12px', borderRadius: '10px', background: 'rgba(239,68,68,0.06)', border: `1px solid ${C.red}` }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.red, marginBottom: '8px' }}>失败明细（这些账户未分到积分，钱仍在来源账户，可修正后重新批量分配）：</div>
                {result.failures.map((f, i) => (
                  <div key={i} style={{ fontSize: '13px', color: C.textSec, padding: '3px 0' }}>
                    · {nameOf(f.to_account_id)} — {f.reason}
                  </div>
                ))}
              </div>
            )}
            <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
              <button onClick={onSuccess} style={btnStyle(C.primary)}>完成</button>
            </div>
          </div>
        ) : (
          /* ==================== 勾选/输入视图 ==================== */
          <div>
            {/* 目标过滤 + 全选/清空 */}
            <div style={{ display: 'flex', gap: '8px', alignItems: 'center', marginBottom: '10px', flexWrap: 'wrap' }}>
              <div style={{ flex: 1, minWidth: '180px' }}>
                <input value={keyword} onChange={e => setKeyword(e.target.value)} placeholder="🔍 按名称过滤目标…" style={inputStyle} />
              </div>
              <button onClick={selectAllFiltered} style={ghostBtnStyle}>☑️ 全选当前列表（{filtered.length}）</button>
              <button onClick={clearAll} style={ghostBtnStyle}>清空</button>
            </div>

            {/* 目标复选列表 */}
            <div style={{ maxHeight: '300px', overflowY: 'auto', border: `1px solid ${C.border}`, borderRadius: '10px', padding: '6px', marginBottom: '14px' }}>
              {loadingTargets ? (
                <div style={{ textAlign: 'center', color: C.textMuted, padding: '24px', fontSize: '13px' }}>加载可分配目标中…</div>
              ) : loadError ? (
                <div style={{ textAlign: 'center', color: C.red, padding: '24px', fontSize: '13px' }}>{loadError}</div>
              ) : filtered.length === 0 ? (
                <div style={{ textAlign: 'center', color: C.textMuted, padding: '24px', fontSize: '13px' }}>
                  {targets.length === 0 ? '该账户没有可分配的下级账户' : '没有匹配当前过滤词的目标'}
                </div>
              ) : (
                filtered.map(t => (
                  <label key={t.id} style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '7px 10px', borderRadius: '8px', cursor: 'pointer', background: selected.has(t.id) ? 'rgba(59,130,246,0.08)' : 'transparent' }}>
                    <input type="checkbox" checked={selected.has(t.id)} onChange={() => toggleOne(t.id)} />
                    <span style={{ flex: 1, fontSize: '13px' }}>{t.display_name}</span>
                    <span style={{ fontSize: '12px', color: C.textMuted }}>余额 {fmt(t.balance)}</span>
                  </label>
                ))
              )}
            </div>

            {/* 每户金额 + 备注 */}
            <div style={{ display: 'flex', gap: '10px', marginBottom: '10px', flexWrap: 'wrap' }}>
              <div style={{ flex: 1, minWidth: '160px' }}>
                <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '4px' }}>每户分配金额（每个目标分同样金额）</div>
                <input value={amountEach} onChange={e => setAmountEach(e.target.value)} placeholder="如 1000" type="number" min="0" style={inputStyle} />
              </div>
              <div style={{ flex: 2, minWidth: '200px' }}>
                <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '4px' }}>备注（可选，写入每笔分配流水）</div>
                <input value={memo} onChange={e => setMemo(e.target.value)} placeholder="如 2026秋季学期额度下发" style={inputStyle} />
              </div>
            </div>

            {/* 实时汇总与预警 */}
            <div style={{ fontSize: '13px', marginBottom: '6px', color: C.textSec }}>
              已勾选 <b>{count}</b> 个目标{amountValid && count > 0 ? <> × {fmt(amount)} 积分 = 总计 <b style={{ color: overBudget ? C.red : C.green }}>{fmt(totalNeeded)}</b> 积分</> : null}
            </div>
            {overBudget && (
              <div style={{ fontSize: '13px', color: C.red, marginBottom: '6px' }}>⚠️ 总计超出来源账户可用余额（{fmt(fromAccountBalance)}），请减少目标或降低每户金额</div>
            )}
            {overLimit && (
              <div style={{ fontSize: '13px', color: C.red, marginBottom: '6px' }}>⚠️ 单次批量分配最多 {MAX_TARGETS} 个目标，请分批操作</div>
            )}
            {submitError && (
              <div style={{ fontSize: '13px', color: C.red, marginBottom: '6px' }}>❌ {submitError}</div>
            )}

            {/* 操作按钮 */}
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '14px' }}>
              <button onClick={onClose} disabled={submitting} style={{ ...ghostBtnStyle, padding: '10px 20px', fontSize: '14px' }}>取消</button>
              <button onClick={handleSubmit} disabled={!canSubmit} style={btnStyle(C.green, !canSubmit)}>
                {submitting ? '分配中…' : count > 0 && amountValid ? `确认分配（${count} 户 × ${fmt(amount)}）` : '确认分配'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
