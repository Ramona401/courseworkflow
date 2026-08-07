/**
 * priceSyncPreviewView.ts — 价格同步预览展示工具。
 *
 * 本文件只负责把后端价格同步数据转换为前端展示文本，
 * 不维护React状态，不发起接口请求，也不修改价格。
 */

import {
  type PriceSyncAction,
  type PriceSyncItem,
} from '@/api/priceSync'
import { C } from './tokenDashboardParts'

export interface PriceSyncActionView {
  label: string
  color: string
}

const ACTION_VIEWS:
  Record<PriceSyncAction, PriceSyncActionView> = {
    update: {
      label: '待应用',
      color: C.orange,
    },
    unchanged: {
      label: '无变化',
      color: C.textMuted,
    },
    skipped: {
      label: '已跳过',
      color: C.red,
    },
    applied: {
      label: '已应用',
      color: C.green,
    },
    stale: {
      label: '本地已变化',
      color: C.purple,
    },
  }

const MEDIA_TYPE_LABELS:
  Record<string, string> = {
    image: '图片',
    video: '视频',
    tts: 'TTS',
  }

/** 格式化美元价格，极小单价保留更多小数位。 */
export function formatPriceSyncUSD(
  value: number,
): string {
  if (!Number.isFinite(value)) {
    return '-'
  }

  if (value === 0) {
    return '$0'
  }

  if (Math.abs(value) < 0.001) {
    return `$${value.toFixed(8)}`
  }

  return `$${value.toFixed(6)}`
}

/** 使用中文本地时间展示同步批次时间。 */
export function formatPriceSyncTime(
  value: string | null,
): string {
  if (!value) {
    return '-'
  }

  return new Date(value).toLocaleString(
    'zh-CN',
  )
}

/** 返回文本、图片、视频或TTS类型名称。 */
export function priceSyncItemTypeLabel(
  item: PriceSyncItem,
): string {
  if (item.target_kind === 'text') {
    return '文本'
  }

  return (
    MEDIA_TYPE_LABELS[item.media_type] ||
    item.media_type ||
    '媒体'
  )
}

/** 返回一次价格变化的旧值和新值展示文本。 */
export function priceSyncChangeLabel(
  item: PriceSyncItem,
): string {
  if (item.target_kind === 'text') {
    return (
      `${formatPriceSyncUSD(item.old_input_usd)} / ` +
      `${formatPriceSyncUSD(item.old_output_usd)} → ` +
      `${formatPriceSyncUSD(item.new_input_usd)} / ` +
      `${formatPriceSyncUSD(item.new_output_usd)} 每1K Token`
    )
  }

  return (
    `${formatPriceSyncUSD(item.old_unit_cost_usd)} → ` +
    `${formatPriceSyncUSD(item.new_unit_cost_usd)} / ` +
    `${item.media_unit || '单位'}`
  )
}

/** 返回同步明细动作的中文名称和展示颜色。 */
export function priceSyncActionView(
  action: PriceSyncAction,
): PriceSyncActionView {
  return ACTION_VIEWS[action]
}
