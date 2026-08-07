/**
 * TokenPricingPolicyPanel.tsx — 积分策略区域组合组件。
 *
 * 本文件只组合两个独立模块：
 *   - TokenRatePolicyCard：美元汇率与积分倍率；
 *   - TokenCreditSimulator：文本模型积分模拟。
 *
 * 具体表单状态与接口调用分别在子组件中维护。
 */

import {
  type CreditPolicy,
  type ModelPrice,
} from '@/api/tokens'
import { TokenRatePolicyCard } from './TokenRatePolicyCard'
import { TokenCreditSimulator } from './TokenCreditSimulator'

interface TokenPricingPolicyPanelProps {
  policy: CreditPolicy | null
  prices: ModelPrice[]
  onPolicyChanged: (
    policy: CreditPolicy,
  ) => void
}

export function TokenPricingPolicyPanel({
  policy,
  prices,
  onPolicyChanged,
}: TokenPricingPolicyPanelProps) {
  return (
    <>
      <TokenRatePolicyCard
        policy={policy}
        onPolicyChanged={onPolicyChanged}
      />

      <TokenCreditSimulator
        prices={prices}
      />
    </>
  )
}
