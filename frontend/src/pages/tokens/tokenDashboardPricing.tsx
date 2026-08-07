/**
 * tokenDashboardPricing.tsx — 积分策略Tab主编排组件。
 *
 * 本文件只负责加载系统积分策略和文本模型价格，并组合三个独立模块：
 *   1. PriceSyncPanel：文本、图片、视频和TTS价格同步；
 *   2. TokenPricingPolicyPanel：汇率、倍率和积分模拟；
 *   3. TokenModelPricesTable：文本模型价格手工维护。
 *
 * 具体表单和表格逻辑均拆入独立文件，避免单文件超过600行。
 */

import {
  useCallback,
  useEffect,
  useState,
} from 'react'
import {
  getModelPrices,
  getSystemCreditPolicy,
  type CreditPolicy,
  type ModelPrice,
} from '@/api/tokens'
import { C } from './tokenDashboardParts'
import { PriceSyncPanel } from './PriceSyncPanel'
import { TokenPricingPolicyPanel } from './TokenPricingPolicyPanel'
import { TokenModelPricesTable } from './TokenModelPricesTable'

// PricingTab 仅对超级管理员展示，后端接口也有超级管理员权限控制。
export function PricingTab() {
  const [policy, setPolicy] =
    useState<CreditPolicy | null>(null)

  const [prices, setPrices] =
    useState<ModelPrice[]>([])

  const [loading, setLoading] =
    useState(true)

  const [loadError, setLoadError] =
    useState('')

  // reload 是本页面唯一的数据刷新入口。
  //
  // 自动同步应用价格、手工新增价格、编辑价格和删除价格完成后，
  // 都调用该函数，保证三个子模块使用相同的最新数据。
  const reload = useCallback(async () => {
    setLoading(true)
    setLoadError('')

    try {
      const [
        policyResult,
        priceResult,
      ] = await Promise.all([
        getSystemCreditPolicy(),
        getModelPrices(),
      ])

      setPolicy(policyResult)
      setPrices(priceResult || [])
    } catch (error) {
      setLoadError(
        error instanceof Error
          ? error.message
          : '积分策略数据加载失败',
      )
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void reload()
  }, [reload])

  if (loading) {
    return (
      <div
        style={{
          padding: '40px',
          textAlign: 'center',
          color: C.textMuted,
        }}
      >
        加载积分策略和模型价格...
      </div>
    )
  }

  if (loadError) {
    return (
      <div
        style={{
          padding: '18px',
          borderRadius: '12px',
          border: `1px solid ${C.red}`,
          background: 'rgba(239,68,68,0.05)',
          color: C.red,
          fontSize: '13px',
        }}
      >
        ⚠ {loadError}
      </div>
    )
  }

  const effectiveRate =
    (policy?.exchange_rate ?? 7) *
    (policy?.multiplier ?? 1)

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: '20px',
      }}
    >
      <PriceSyncPanel
        onPricesChanged={reload}
      />

      <TokenPricingPolicyPanel
        policy={policy}
        prices={prices}
        onPolicyChanged={setPolicy}
      />

      <TokenModelPricesTable
        prices={prices}
        effectiveRate={effectiveRate}
        onChanged={reload}
      />
    </div>
  )
}
