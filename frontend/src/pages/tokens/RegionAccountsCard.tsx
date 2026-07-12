/**
 * RegionAccountsCard.tsx — "我管辖的区域账户"卡片（T1 区域分配入口 batch 新增）
 *
 * 背景：
 *   v172.1 起非 admin 的账户列表在 SQL 层无条件排除一切 region 账户（防跨级泄漏），
 *   region_admin 连自己管辖的区域账户也看不见——账户不可见，就没有"从区域账户
 *   向学校分配"的入口。后端分配链路（tokenSourceAllowed + getAllocatableTargets +
 *   allocate）早已放行，本组件补齐这个"发现入口"。
 *
 * 职责（纯展示 + 回调，不持分配弹窗状态）：
 *   - 挂载时调 getMyRegionAccounts 拉"我管辖的区域账户"（后端按
 *     AllowedRegionOwnerIDs 收窄，仅 region_admin 非空，其余角色恒空列表）
 *   - 空列表且无缺失账户提示 → 整块不渲染（admin/senior 等角色天然隐藏，
 *     父组件即便误挂载也无副作用）
 *   - 每个区域账户一行：账户名 + 可用余额 + 「📤 分配给学校」按钮，
 *     点击经 onAllocate(id, name, balance) 冒泡给父组件（TokenDashboardPage）
 *     打开既有 AllocateModal（AccountPicker mode=children 只列辖区学校，零改动复用）
 *   - missing_accounts > 0 时显示橙色提示（该区域尚未创建积分账户，请联系系统管理员）
 *   - refreshSignal 变化时重拉（父组件在分配成功后自增，余额即时刷新）
 *
 * 安全边界：本组件不做任何权限判断——数据可见性与分配鉴权全在后端
 *   （ResolveTokenScope / tokenSourceAllowed 三重防护），前端只按返回渲染。
 */
import { useState, useEffect, useCallback } from 'react'
import { getMyRegionAccounts, type TokenAccountListItem } from '@/api/tokens'
import { C } from './tokenDashboardParts'

interface RegionAccountsCardProps {
  /** 刷新信号：父组件在分配成功后自增，本组件监听变化重拉余额 */
  refreshSignal?: number
  /** 点击"分配给学校"回调：父组件据此打开既有 AllocateModal */
  onAllocate: (id: string, name: string, balance: number) => void
}

export default function RegionAccountsCard({ refreshSignal = 0, onAllocate }: RegionAccountsCardProps) {
  const [items, setItems] = useState<TokenAccountListItem[]>([])
  const [missing, setMissing] = useState(0)
  const [loaded, setLoaded] = useState(false)

  // 拉取"我管辖的区域账户"（失败静默为空，不打扰主页面）
  const load = useCallback(async () => {
    try {
      const data = await getMyRegionAccounts()
      setItems(data?.items || [])
      setMissing(data?.missing_accounts || 0)
    } catch {
      setItems([])
      setMissing(0)
    }
    setLoaded(true)
  }, [])

  // 首次挂载 + refreshSignal 变化（分配成功后）重拉
  useEffect(() => { load() }, [load, refreshSignal])

  // 未加载完成 / 无管辖区域账户且无缺失提示 → 整块隐藏（非 region_admin 角色天然走这里）
  if (!loaded) return null
  if (items.length === 0 && missing === 0) return null

  return (
    <div style={{
      marginBottom: '24px', padding: '20px 24px', borderRadius: '16px',
      background: 'rgba(139,92,246,0.05)', border: `1px solid rgba(139,92,246,0.35)`,
    }}>
      {/* ===== 标题 + 说明 ===== */}
      <div style={{ display: 'flex', alignItems: 'baseline', gap: '10px', marginBottom: '4px', flexWrap: 'wrap' }}>
        <span style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>🏦 我管辖的区域账户</span>
        <span style={{ fontSize: '12px', color: C.textSec }}>
          从区域账户向辖区学校分配积分；学校管理员再继续下发给老师
        </span>
      </div>

      {/* ===== 区域账户行（通常1个，兼容多区域管辖）===== */}
      {items.map(acc => {
        const active = acc.status === 'active'
        const hasBalance = acc.available_balance > 0
        const canAllocate = active && hasBalance
        return (
          <div key={acc.id} style={{
            display: 'flex', alignItems: 'center', gap: '16px', flexWrap: 'wrap',
            padding: '14px 16px', marginTop: '10px', borderRadius: '12px',
            background: C.white, border: `1px solid ${C.border}`,
          }}>
            {/* 账户名 + 状态 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0, flex: '1 1 200px' }}>
              <span style={{ fontSize: '15px' }}>📍</span>
              <span style={{ fontSize: '14px', fontWeight: 600, color: C.text, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {acc.display_name}
              </span>
              {!active && (
                <span style={{ fontSize: '11px', padding: '1px 7px', borderRadius: '4px', background: 'rgba(239,68,68,0.08)', color: C.red, flexShrink: 0 }}>
                  {acc.status_name}
                </span>
              )}
            </div>

            {/* 可用余额 */}
            <div style={{ flexShrink: 0 }}>
              <span style={{ fontSize: '12px', color: C.textMuted, marginRight: '6px' }}>可用余额</span>
              <span style={{ fontSize: '18px', fontWeight: 700, color: hasBalance ? C.green : C.red }}>
                {acc.available_balance.toLocaleString(undefined, { maximumFractionDigits: 2 })}
              </span>
              <span style={{ fontSize: '12px', color: C.textSec, marginLeft: '4px' }}>积分</span>
            </div>

            {/* 分配按钮：冻结/余额不足时禁用并给原因提示 */}
            <button
              onClick={() => canAllocate && onAllocate(acc.id, acc.display_name, acc.available_balance)}
              disabled={!canAllocate}
              title={!active ? '账户不在活跃状态，无法分配' : !hasBalance ? '余额不足，请联系系统管理员向本区域账户充值' : ''}
              style={{
                padding: '8px 18px', borderRadius: '8px', border: 'none', flexShrink: 0,
                cursor: canAllocate ? 'pointer' : 'not-allowed',
                fontSize: '13px', fontWeight: 600,
                color: C.white, background: canAllocate ? C.purple : C.textMuted,
              }}
            >
              📤 分配给学校
            </button>
          </div>
        )
      })}

      {/* ===== 缺失账户提示（管辖区域尚未创建积分账户）===== */}
      {missing > 0 && (
        <div style={{
          marginTop: '10px', padding: '10px 14px', borderRadius: '10px',
          background: 'rgba(245,158,11,0.08)', border: `1px solid ${C.orange}`,
          fontSize: '12px', color: C.orange,
        }}>
          ⚠️ 您管辖的 {missing} 个区域尚未创建积分账户，请联系系统管理员在"账户管理"中为区域创建积分账户并充值后，方可向学校分配。
        </div>
      )}
    </div>
  )
}
