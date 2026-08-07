/**
 * priceSync.ts — 模型及媒体价格同步API。
 *
 * 本文件独立于tokens.ts，避免Token主API文件继续膨胀。
 *
 * 管理能力：
 *   - 读取及更新全局同步设置；
 *   - 更新单个文本或媒体价格目标；
 *   - 生成价格预览；
 *   - 全部或选择性应用价格；
 *   - 查询同步历史及批次详情。
 */

import apiClient from './client'

export type PriceSyncTargetKind = 'text' | 'media'

export type PriceSyncSource =
  | 'main_gateway'
  | 'domestic_gateway'
  | 'media_gateway'
  | 'tts_gateway'

export type PriceSyncAction =
  | 'update'
  | 'unchanged'
  | 'skipped'
  | 'applied'
  | 'stale'

export type PriceSyncRunStatus =
  | 'previewed'
  | 'applied'
  | 'failed'

export interface PriceSyncSettings {
  enabled: boolean
  auto_apply: boolean
  group: string
  interval_hours: number
  max_change_percent: number

  main_pricing_url: string
  domestic_pricing_url: string
  media_pricing_url: string
  tts_pricing_url: string
}

export interface UpdatePriceSyncSettingsRequest {
  enabled?: boolean
  auto_apply?: boolean
  group?: string
  interval_hours?: number
  max_change_percent?: number

  main_pricing_url?: string
  domestic_pricing_url?: string
  media_pricing_url?: string
  tts_pricing_url?: string
}

export interface PriceSyncTargetConfig {
  id: string
  target_kind: PriceSyncTargetKind

  provider: string
  model_name: string
  display_name: string
  is_active: boolean

  media_type: string
  variant: string
  media_unit: string

  current_input_usd: number
  current_output_usd: number
  current_unit_cost_usd: number

  auto_sync_enabled: boolean
  sync_source: PriceSyncSource
  sync_model_name: string

  last_synced_at: string | null
  last_sync_status: string
  last_sync_message: string
}

export interface UpdatePriceSyncTargetRequest {
  auto_sync_enabled?: boolean
  sync_source?: PriceSyncSource
  sync_model_name?: string
}

export interface PriceSyncManagementState {
  settings: PriceSyncSettings
  text_targets: PriceSyncTargetConfig[]
  media_targets: PriceSyncTargetConfig[]
}

export interface PriceSyncSummary {
  total_count: number
  update_count: number
  unchanged_count: number
  skipped_count: number
  applied_count: number
  stale_count: number
}

export interface PriceSyncRun {
  id: string
  trigger_type: 'manual' | 'scheduler'
  status: PriceSyncRunStatus
  source_kind: string
  source_base_url: string
  preview_only: boolean
  summary: PriceSyncSummary
  error_message: string
  created_by: string | null
  started_at: string
  finished_at: string | null
}

export interface PriceSyncItem {
  id: string
  run_id: string
  target_kind: PriceSyncTargetKind
  target_id: string

  provider: string
  model_name: string
  sync_source: PriceSyncSource

  media_type: string
  variant: string
  media_unit: string

  old_input_usd: number
  new_input_usd: number
  old_output_usd: number
  new_output_usd: number
  old_unit_cost_usd: number
  new_unit_cost_usd: number

  action: PriceSyncAction
  reason: string
  source_payload: Record<string, unknown>
  created_at: string
}

export interface PriceSyncPreviewRequest {
  group?: string
  include_text?: boolean
  include_media?: boolean
  max_change_percent?: number
}

export interface PriceSyncPreviewResponse {
  run: PriceSyncRun
  items: PriceSyncItem[]
  summary: PriceSyncSummary
}

export interface PriceSyncApplyRequest {
  run_id: string
  apply_all: boolean
  item_ids: string[]
}

export interface PriceSyncApplyResponse {
  run: PriceSyncRun
  items: PriceSyncItem[]
  summary: PriceSyncSummary
}

function extractData<T>(
  response: {
    data?: {
      data?: T
    }
  },
): T {
  const outer = response?.data as
    | Record<string, unknown>
    | undefined

  if (outer && 'data' in outer) {
    return outer.data as T
  }

  return outer as T
}

/** 获取同步设置和全部同步目标。 */
export async function getPriceSyncManagementState() {
  const response = await apiClient.get(
    '/tokens/price-sync/settings',
  )

  return extractData<PriceSyncManagementState>(
    response,
  )
}

/** 更新全局价格同步设置。 */
export async function updatePriceSyncSettings(
  request: UpdatePriceSyncSettingsRequest,
) {
  const response = await apiClient.put(
    '/tokens/price-sync/settings',
    request,
  )

  return extractData<PriceSyncSettings>(
    response,
  )
}

/** 更新单个价格同步目标。 */
export async function updatePriceSyncTarget(
  targetKind: PriceSyncTargetKind,
  targetID: string,
  request: UpdatePriceSyncTargetRequest,
) {
  const response = await apiClient.put(
    `/tokens/price-sync/targets/${targetKind}/${targetID}`,
    request,
  )

  return extractData<PriceSyncTargetConfig>(
    response,
  )
}

/** 拉取上游价格并生成预览，不修改正式价格。 */
export async function previewPriceSync(
  request: PriceSyncPreviewRequest = {},
) {
  const response = await apiClient.post(
    '/tokens/price-sync/preview',
    request,
  )

  return extractData<PriceSyncPreviewResponse>(
    response,
  )
}

/** 应用全部可信价格变化。 */
export async function applyAllPriceSyncChanges(
  runID: string,
) {
  const request: PriceSyncApplyRequest = {
    run_id: runID,
    apply_all: true,
    item_ids: [],
  }

  const response = await apiClient.post(
    '/tokens/price-sync/apply',
    request,
  )

  return extractData<PriceSyncApplyResponse>(
    response,
  )
}

/** 只应用管理员选择的价格变化。 */
export async function applySelectedPriceSyncChanges(
  runID: string,
  itemIDs: string[],
) {
  const request: PriceSyncApplyRequest = {
    run_id: runID,
    apply_all: false,
    item_ids: itemIDs,
  }

  const response = await apiClient.post(
    '/tokens/price-sync/apply',
    request,
  )

  return extractData<PriceSyncApplyResponse>(
    response,
  )
}

/** 获取最近价格同步历史。 */
export async function getPriceSyncRuns(
  limit = 20,
) {
  const response = await apiClient.get(
    '/tokens/price-sync/runs',
    {
      params: {
        limit,
      },
    },
  )

  return extractData<PriceSyncRun[]>(
    response,
  )
}

/** 获取一个同步批次及全部价格明细。 */
export async function getPriceSyncRunDetail(
  runID: string,
) {
  const response = await apiClient.get(
    `/tokens/price-sync/runs/${runID}`,
  )

  return extractData<PriceSyncPreviewResponse>(
    response,
  )
}
