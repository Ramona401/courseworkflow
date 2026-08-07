/**
 * PriceSyncSettingsCard.tsx — 价格同步全局设置。
 *
 * 配置保存不直接修改任何模型价格。
 * 价格变更仍须经过预览及应用流程。
 */

import { useEffect, useState } from 'react'
import {
  type PriceSyncSettings,
  type UpdatePriceSyncSettingsRequest,
} from '@/api/priceSync'
import {
  C,
  cardStyle,
  inputStyle,
} from './tokenDashboardParts'

interface PriceSyncSettingsCardProps {
  settings: PriceSyncSettings
  saving: boolean
  onSave: (
    request: UpdatePriceSyncSettingsRequest,
  ) => Promise<PriceSyncSettings>
}

interface URLFieldProps {
  label: string
  value: string
  placeholder: string
  help: string
  onChange: (value: string) => void
}

function URLField({
  label,
  value,
  placeholder,
  help,
  onChange,
}: URLFieldProps) {
  return (
    <div>
      <div
        style={{
          marginBottom: '5px',
          fontSize: '12px',
          fontWeight: 600,
          color: C.textSec,
        }}
      >
        {label}
      </div>

      <input
        value={value}
        onChange={event => onChange(
          event.target.value,
        )}
        placeholder={placeholder}
        style={{
          ...inputStyle,
          fontFamily: 'monospace',
          fontSize: '12px',
        }}
      />

      <div
        style={{
          marginTop: '4px',
          fontSize: '11px',
          lineHeight: 1.5,
          color: C.textMuted,
        }}
      >
        {help}
      </div>
    </div>
  )
}

export function PriceSyncSettingsCard({
  settings,
  saving,
  onSave,
}: PriceSyncSettingsCardProps) {
  const [draft, setDraft] =
    useState<PriceSyncSettings>(settings)

  const [localError, setLocalError] =
    useState('')

  useEffect(() => {
    setDraft(settings)
    setLocalError('')
  }, [settings])

  const updateDraft = (
    patch: Partial<PriceSyncSettings>,
  ) => {
    setDraft(current => ({
      ...current,
      ...patch,
    }))
  }

  const handleSave = async () => {
    setLocalError('')

    if (!draft.group.trim()) {
      setLocalError('价格同步分组不能为空。')
      return
    }

    if (
      draft.interval_hours < 1 ||
      draft.interval_hours > 720
    ) {
      setLocalError(
        '同步间隔必须在1至720小时之间。',
      )
      return
    }

    if (
      draft.max_change_percent < 1 ||
      draft.max_change_percent > 1000
    ) {
      setLocalError(
        '价格变化安全阈值必须在1%至1000%之间。',
      )
      return
    }

    try {
      const saved = await onSave({
        enabled: draft.enabled,
        auto_apply: draft.auto_apply,
        group: draft.group.trim(),
        interval_hours:
          draft.interval_hours,
        max_change_percent:
          draft.max_change_percent,

        main_pricing_url:
          draft.main_pricing_url.trim(),
        domestic_pricing_url:
          draft.domestic_pricing_url.trim(),
        media_pricing_url:
          draft.media_pricing_url.trim(),
        tts_pricing_url:
          draft.tts_pricing_url.trim(),
      })

      setDraft(saved)
    } catch {
      // 错误信息由主面板统一展示。
    }
  }

  return (
    <div style={cardStyle}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: '12px',
          flexWrap: 'wrap',
          marginBottom: '16px',
        }}
      >
        <div>
          <div
            style={{
              fontSize: '15px',
              fontWeight: 700,
              color: C.text,
            }}
          >
            ⚙️ 自动同步全局设置
          </div>

          <div
            style={{
              marginTop: '3px',
              fontSize: '12px',
              color: C.textMuted,
            }}
          >
            保存配置不会立即改价。
          </div>
        </div>

        <button
          onClick={handleSave}
          disabled={saving}
          style={{
            padding: '8px 18px',
            borderRadius: '8px',
            border: 'none',
            background: saving
              ? C.textMuted
              : C.primary,
            color: C.white,
            fontSize: '13px',
            fontWeight: 600,
            cursor: saving
              ? 'not-allowed'
              : 'pointer',
          }}
        >
          {saving ? '保存中...' : '保存同步设置'}
        </button>
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns:
            'repeat(auto-fit, minmax(220px, 1fr))',
          gap: '14px',
          marginBottom: '16px',
        }}
      >
        <label
          style={{
            padding: '12px',
            borderRadius: '10px',
            border: `1px solid ${C.border}`,
            cursor: 'pointer',
          }}
        >
          <input
            type="checkbox"
            checked={draft.enabled}
            onChange={event => updateDraft({
              enabled: event.target.checked,
            })}
            style={{ marginRight: '8px' }}
          />

          <span
            style={{
              fontSize: '13px',
              fontWeight: 600,
              color: C.text,
            }}
          >
            启用定时价格同步
          </span>

          <div
            style={{
              marginTop: '5px',
              marginLeft: '22px',
              fontSize: '11px',
              lineHeight: 1.5,
              color: C.textMuted,
            }}
          >
            仅检查开启了单模型自动同步的记录。
          </div>
        </label>

        <label
          style={{
            padding: '12px',
            borderRadius: '10px',
            border: `1px solid ${
              draft.auto_apply
                ? C.orange
                : C.border
            }`,
            cursor: 'pointer',
          }}
        >
          <input
            type="checkbox"
            checked={draft.auto_apply}
            onChange={event => updateDraft({
              auto_apply:
                event.target.checked,
            })}
            style={{ marginRight: '8px' }}
          />

          <span
            style={{
              fontSize: '13px',
              fontWeight: 600,
              color: draft.auto_apply
                ? C.orange
                : C.text,
            }}
          >
            自动应用可信变化
          </span>

          <div
            style={{
              marginTop: '5px',
              marginLeft: '22px',
              fontSize: '11px',
              lineHeight: 1.5,
              color: C.textMuted,
            }}
          >
            关闭时定时任务只保存预览，等待人工确认。
          </div>
        </label>
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns:
            'repeat(auto-fit, minmax(180px, 1fr))',
          gap: '14px',
          marginBottom: '18px',
        }}
      >
        <div>
          <div
            style={{
              marginBottom: '5px',
              fontSize: '12px',
              fontWeight: 600,
              color: C.textSec,
            }}
          >
            聚合网关分组
          </div>

          <input
            value={draft.group}
            onChange={event => updateDraft({
              group: event.target.value,
            })}
            style={inputStyle}
          />
        </div>

        <div>
          <div
            style={{
              marginBottom: '5px',
              fontSize: '12px',
              fontWeight: 600,
              color: C.textSec,
            }}
          >
            同步间隔（小时）
          </div>

          <input
            type="number"
            min={1}
            max={720}
            value={draft.interval_hours}
            onChange={event => updateDraft({
              interval_hours:
                Number(event.target.value),
            })}
            style={inputStyle}
          />
        </div>

        <div>
          <div
            style={{
              marginBottom: '5px',
              fontSize: '12px',
              fontWeight: 600,
              color: C.textSec,
            }}
          >
            单次变化安全阈值（%）
          </div>

          <input
            type="number"
            min={1}
            max={1000}
            step="1"
            value={
              draft.max_change_percent
            }
            onChange={event => updateDraft({
              max_change_percent:
                Number(event.target.value),
            })}
            style={inputStyle}
          />
        </div>
      </div>

      <div
        style={{
          padding: '14px',
          borderRadius: '12px',
          background: C.bg,
          border: `1px solid ${C.border}`,
        }}
      >
        <div
          style={{
            marginBottom: '12px',
            fontSize: '13px',
            fontWeight: 700,
            color: C.text,
          }}
        >
          价格接口地址
        </div>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns:
              'repeat(auto-fit, minmax(320px, 1fr))',
            gap: '14px',
          }}
        >
          <URLField
            label="主聚合网关价格接口"
            value={draft.main_pricing_url}
            placeholder="留空自动从 api_base_url 推导"
            help="留空时自动使用主聚合网关根地址的 /api/pricing。"
            onChange={value => updateDraft({
              main_pricing_url: value,
            })}
          />

          <URLField
            label="境内文本价格接口"
            value={
              draft.domestic_pricing_url
            }
            placeholder="例如 https://.../api/pricing"
            help="未配置时 qwen3.7-max 等境内模型会安全跳过。"
            onChange={value => updateDraft({
              domestic_pricing_url: value,
            })}
          />

          <URLField
            label="图片和视频价格接口"
            value={draft.media_pricing_url}
            placeholder="例如 https://.../api/pricing"
            help="必须是价格JSON接口，不能填写图片或视频推理地址。"
            onChange={value => updateDraft({
              media_pricing_url: value,
            })}
          />

          <URLField
            label="TTS价格接口"
            value={draft.tts_pricing_url}
            placeholder="例如 https://.../api/pricing"
            help="必须明确返回字符、音频秒或供应商Token单价。"
            onChange={value => updateDraft({
              tts_pricing_url: value,
            })}
          />
        </div>
      </div>

      {localError && (
        <div
          style={{
            marginTop: '12px',
            fontSize: '12px',
            color: C.red,
          }}
        >
          ⚠ {localError}
        </div>
      )}
    </div>
  )
}
