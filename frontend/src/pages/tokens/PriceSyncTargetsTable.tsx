/**
 * PriceSyncTargetsTable.tsx — 单模型同步目标配置表。
 *
 * 每条记录可独立设置：
 *   - 是否参加定时同步；
 *   - 使用哪个价格来源；
 *   - 上游价格接口中的精确模型名。
 *
 * 本组件不修改正式价格。
 */

import { useState } from 'react'
import {
  type PriceSyncSource,
  type PriceSyncTargetConfig,
  type UpdatePriceSyncTargetRequest,
} from '@/api/priceSync'
import {
  C,
  inputStyle,
  tdStyle,
  thStyle,
} from './tokenDashboardParts'

interface PriceSyncTargetsTableProps {
  textTargets: PriceSyncTargetConfig[]
  mediaTargets: PriceSyncTargetConfig[]
  busyAction: string
  onSave: (
    target: PriceSyncTargetConfig,
    request: UpdatePriceSyncTargetRequest,
  ) => Promise<PriceSyncTargetConfig>
}

const SOURCE_LABELS:
  Record<PriceSyncSource, string> = {
    main_gateway: '主聚合网关',
    domestic_gateway: '境内价格源',
    media_gateway: '图片/视频价格源',
    tts_gateway: 'TTS价格源',
  }

function allowedSources(
  target: PriceSyncTargetConfig,
): PriceSyncSource[] {
  if (target.target_kind === 'text') {
    return [
      'main_gateway',
      'domestic_gateway',
    ]
  }

  if (target.media_type === 'tts') {
    return [
      'main_gateway',
      'tts_gateway',
    ]
  }

  return [
    'main_gateway',
    'media_gateway',
  ]
}

function formatUSD(value: number): string {
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

function targetPriceLabel(
  target: PriceSyncTargetConfig,
): string {
  if (target.target_kind === 'text') {
    return (
      `${formatUSD(target.current_input_usd)} 输入 / ` +
      `${formatUSD(target.current_output_usd)} 输出 · 每1K Token`
    )
  }

  return (
    `${formatUSD(target.current_unit_cost_usd)} / ` +
    `${target.media_unit || '单位'}`
  )
}

function targetTypeLabel(
  target: PriceSyncTargetConfig,
): string {
  if (target.target_kind === 'text') {
    return '文本'
  }

  switch (target.media_type) {
  case 'image':
    return '图片'
  case 'video':
    return '视频'
  case 'tts':
    return 'TTS'
  default:
    return target.media_type
  }
}

function formatSyncTime(
  value: string | null,
): string {
  if (!value) return '尚未同步'

  return new Date(value).toLocaleString(
    'zh-CN',
  )
}

export function PriceSyncTargetsTable({
  textTargets,
  mediaTargets,
  busyAction,
  onSave,
}: PriceSyncTargetsTableProps) {
  const [editingID, setEditingID] =
    useState('')

  const [draftSource, setDraftSource] =
    useState<PriceSyncSource>(
      'main_gateway',
    )

  const [draftModelName, setDraftModelName] =
    useState('')

  const [draftEnabled, setDraftEnabled] =
    useState(false)

  const startEdit = (
    target: PriceSyncTargetConfig,
  ) => {
    setEditingID(target.id)
    setDraftSource(target.sync_source)
    setDraftModelName(
      target.sync_model_name,
    )
    setDraftEnabled(
      target.auto_sync_enabled,
    )
  }

  const cancelEdit = () => {
    setEditingID('')
    setDraftModelName('')
  }

  const saveTarget = async (
    target: PriceSyncTargetConfig,
  ) => {
    if (!draftModelName.trim()) {
      return
    }

    try {
      await onSave(target, {
        auto_sync_enabled:
          draftEnabled,
        sync_source: draftSource,
        sync_model_name:
          draftModelName.trim(),
      })

      cancelEdit()
    } catch {
      // 错误由主面板统一展示。
    }
  }

  const renderRows = (
    items: PriceSyncTargetConfig[],
  ) => items.map(target => {
    const editing =
      editingID === target.id

    const saving =
      busyAction ===
      `target:${target.id}`

    return (
      <tr key={`${target.target_kind}:${target.id}`}>
        <td style={tdStyle}>
          <div
            style={{
              fontWeight: 600,
              color: C.text,
            }}
          >
            {target.display_name ||
              target.model_name}
          </div>

          <div
            style={{
              marginTop: '3px',
              fontFamily: 'monospace',
              fontSize: '11px',
              color: C.textMuted,
            }}
          >
            {target.model_name}
          </div>
        </td>

        <td style={tdStyle}>
          <span
            style={{
              display: 'inline-block',
              padding: '2px 8px',
              borderRadius: '999px',
              background: C.primaryLight,
              color: C.primary,
              fontSize: '11px',
            }}
          >
            {targetTypeLabel(target)}
          </span>

          <div
            style={{
              marginTop: '5px',
              fontSize: '11px',
              color: C.textMuted,
            }}
          >
            {target.provider}
          </div>
        </td>

        <td style={tdStyle}>
          <div
            style={{
              minWidth: '190px',
              fontSize: '12px',
              color: C.textSec,
            }}
          >
            {targetPriceLabel(target)}
          </div>
        </td>

        <td style={tdStyle}>
          {editing ? (
            <select
              value={draftSource}
              onChange={event =>
                setDraftSource(
                  event.target.value as
                    PriceSyncSource,
                )
              }
              style={{
                ...inputStyle,
                minWidth: '160px',
                padding: '7px 9px',
                fontSize: '12px',
              }}
            >
              {allowedSources(target).map(
                source => (
                  <option
                    key={source}
                    value={source}
                  >
                    {SOURCE_LABELS[source]}
                  </option>
                ),
              )}
            </select>
          ) : (
            <span
              style={{
                fontSize: '12px',
                color: C.textSec,
              }}
            >
              {SOURCE_LABELS[
                target.sync_source
              ] || target.sync_source}
            </span>
          )}
        </td>

        <td style={tdStyle}>
          {editing ? (
            <input
              value={draftModelName}
              onChange={event =>
                setDraftModelName(
                  event.target.value,
                )
              }
              style={{
                ...inputStyle,
                minWidth: '220px',
                padding: '7px 9px',
                fontFamily: 'monospace',
                fontSize: '11px',
              }}
            />
          ) : (
            <span
              style={{
                fontFamily: 'monospace',
                fontSize: '11px',
                color: C.textSec,
              }}
            >
              {target.sync_model_name}
            </span>
          )}
        </td>

        <td style={tdStyle}>
          {editing ? (
            <label
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                whiteSpace: 'nowrap',
                fontSize: '12px',
                cursor: 'pointer',
              }}
            >
              <input
                type="checkbox"
                checked={draftEnabled}
                onChange={event =>
                  setDraftEnabled(
                    event.target.checked,
                  )
                }
              />
              参加定时同步
            </label>
          ) : (
            <span
              style={{
                color:
                  target.auto_sync_enabled
                    ? C.green
                    : C.textMuted,
                fontSize: '12px',
              }}
            >
              {target.auto_sync_enabled
                ? '✓ 已开启'
                : '— 已关闭'}
            </span>
          )}
        </td>

        <td style={tdStyle}>
          <div
            style={{
              minWidth: '145px',
              fontSize: '11px',
              color: C.textMuted,
            }}
          >
            {formatSyncTime(
              target.last_synced_at,
            )}
          </div>

          {target.last_sync_message && (
            <div
              style={{
                marginTop: '4px',
                maxWidth: '180px',
                fontSize: '11px',
                color:
                  target.last_sync_status ===
                  'success'
                    ? C.green
                    : C.orange,
              }}
            >
              {target.last_sync_message}
            </div>
          )}
        </td>

        <td style={tdStyle}>
          {editing ? (
            <div
              style={{
                display: 'flex',
                gap: '5px',
              }}
            >
              <button
                onClick={() =>
                  void saveTarget(target)
                }
                disabled={
                  saving ||
                  !draftModelName.trim()
                }
                style={{
                  padding: '5px 10px',
                  borderRadius: '6px',
                  border: 'none',
                  background: saving
                    ? C.textMuted
                    : C.green,
                  color: C.white,
                  fontSize: '12px',
                  cursor: saving
                    ? 'not-allowed'
                    : 'pointer',
                }}
              >
                {saving ? '保存中' : '保存'}
              </button>

              <button
                onClick={cancelEdit}
                disabled={saving}
                style={{
                  padding: '5px 10px',
                  borderRadius: '6px',
                  border:
                    `1px solid ${C.border}`,
                  background: C.white,
                  color: C.textSec,
                  fontSize: '12px',
                  cursor: 'pointer',
                }}
              >
                取消
              </button>
            </div>
          ) : (
            <button
              onClick={() =>
                startEdit(target)
              }
              style={{
                padding: '5px 12px',
                borderRadius: '6px',
                border:
                  `1px solid ${C.border}`,
                background: C.white,
                color: C.textSec,
                fontSize: '12px',
                cursor: 'pointer',
              }}
            >
              配置
            </button>
          )}
        </td>
      </tr>
    )
  })

  const allTargets = [
    ...textTargets,
    ...mediaTargets,
  ]

  return (
    <div
      style={{
        background: C.white,
        borderRadius: '14px',
        border: `1px solid ${C.border}`,
        overflow: 'auto',
      }}
    >
      <div
        style={{
          padding: '16px 20px',
          borderBottom:
            `1px solid ${C.border}`,
        }}
      >
        <div
          style={{
            fontSize: '15px',
            fontWeight: 700,
            color: C.text,
          }}
        >
          🎯 单模型同步配置
        </div>

        <div
          style={{
            marginTop: '4px',
            fontSize: '12px',
            lineHeight: 1.6,
            color: C.textMuted,
          }}
        >
          上游模型名必须与价格接口返回值逐字一致。
          开启单模型同步后，仍需打开全局定时同步才会自动检查。
        </div>
      </div>

      <table
        style={{
          width: '100%',
          minWidth: '1180px',
          borderCollapse: 'collapse',
        }}
      >
        <thead>
          <tr>
            <th style={thStyle}>模型</th>
            <th style={thStyle}>类型</th>
            <th style={thStyle}>当前价格</th>
            <th style={thStyle}>价格来源</th>
            <th style={thStyle}>上游模型名</th>
            <th style={thStyle}>自动同步</th>
            <th style={thStyle}>最近同步</th>
            <th style={thStyle}>操作</th>
          </tr>
        </thead>

        <tbody>
          {renderRows(allTargets)}
        </tbody>
      </table>

      {allTargets.length === 0 && (
        <div
          style={{
            padding: '32px',
            textAlign: 'center',
            color: C.textMuted,
          }}
        >
          暂无可配置的价格同步目标。
        </div>
      )}
    </div>
  )
}
