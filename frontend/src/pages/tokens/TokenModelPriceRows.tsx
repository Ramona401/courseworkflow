/**
 * TokenModelPriceRows.tsx — 文本模型价格表的纯展示行组件。
 *
 * 本文件不发起接口请求、不保存业务状态。
 * 新增、编辑和删除动作由父组件TokenModelPricesTable管理。
 */

import {
  type CreateModelPriceRequest,
  type ModelPrice,
  PROVIDER_COLORS,
} from '@/api/tokens'
import {
  C,
  inputStyle,
  tdStyle,
} from './tokenDashboardParts'

export interface EditPriceForm {
  cost_per_1k_input: number
  cost_per_1k_output: number
  display_name: string
}

interface CreatePriceRowProps {
  form: CreateModelPriceRequest
  busy: boolean
  onChange: (
    form: CreateModelPriceRequest,
  ) => void
  onSave: () => void
}

interface ModelPriceRowProps {
  price: ModelPrice
  effectiveRate: number
  editing: boolean
  editForm: EditPriceForm
  busy: boolean

  onEdit: () => void
  onEditFormChange: (
    form: EditPriceForm,
  ) => void
  onSave: () => void
  onCancel: () => void
  onDelete: () => void
}

export function CreatePriceRow({
  form,
  busy,
  onChange,
  onSave,
}: CreatePriceRowProps) {
  return (
    <tr
      style={{
        background:
          'rgba(16,185,129,0.05)',
      }}
    >
      <td
        style={tdStyle}
        colSpan={8}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            flexWrap: 'wrap',
          }}
        >
          <input
            value={form.model_name}
            onChange={event =>
              onChange({
                ...form,
                model_name:
                  event.target.value,
              })
            }
            placeholder="实际 model_name"
            style={{
              ...inputStyle,
              width: '210px',
              fontFamily: 'monospace',
            }}
          />

          <input
            value={form.provider}
            onChange={event =>
              onChange({
                ...form,
                provider:
                  event.target.value,
              })
            }
            placeholder="供应商"
            style={{
              ...inputStyle,
              width: '110px',
            }}
          />

          <PriceNumberInput
            value={
              form.cost_per_1k_input
            }
            placeholder="输入$/1K"
            onChange={value =>
              onChange({
                ...form,
                cost_per_1k_input: value,
              })
            }
          />

          <PriceNumberInput
            value={
              form.cost_per_1k_output
            }
            placeholder="输出$/1K"
            onChange={value =>
              onChange({
                ...form,
                cost_per_1k_output: value,
              })
            }
          />

          <input
            value={form.display_name}
            onChange={event =>
              onChange({
                ...form,
                display_name:
                  event.target.value,
              })
            }
            placeholder="显示名称"
            style={{
              ...inputStyle,
              width: '150px',
            }}
          />

          <ActionButton
            label={
              busy ? '保存中' : '保存'
            }
            color={C.green}
            disabled={busy}
            onClick={onSave}
          />
        </div>
      </td>
    </tr>
  )
}

export function ModelPriceRow({
  price,
  effectiveRate,
  editing,
  editForm,
  busy,
  onEdit,
  onEditFormChange,
  onSave,
  onCancel,
  onDelete,
}: ModelPriceRowProps) {
  const providerColor =
    PROVIDER_COLORS[price.provider] ||
    C.textMuted

  return (
    <tr>
      <td style={tdStyle}>
        {editing ? (
          <input
            value={editForm.display_name}
            onChange={event =>
              onEditFormChange({
                ...editForm,
                display_name:
                  event.target.value,
              })
            }
            placeholder="显示名称"
            style={{
              ...inputStyle,
              width: '170px',
            }}
          />
        ) : (
          <div
            style={{
              fontWeight: 600,
            }}
          >
            {price.display_name ||
              price.model_name}
          </div>
        )}

        <div
          style={{
            marginTop: '3px',
            color: C.textMuted,
            fontFamily: 'monospace',
            fontSize: '11px',
          }}
        >
          {price.model_name}
        </div>
      </td>

      <td style={tdStyle}>
        <span
          style={{
            padding: '2px 8px',
            borderRadius: '999px',
            background:
              `${providerColor}15`,
            color: providerColor,
            fontSize: '12px',
          }}
        >
          {price.provider}
        </span>
      </td>

      <td style={tdStyle}>
        {editing ? (
          <PriceNumberInput
            value={
              editForm.cost_per_1k_input
            }
            onChange={value =>
              onEditFormChange({
                ...editForm,
                cost_per_1k_input: value,
              })
            }
          />
        ) : (
          formatPrice(
            price.cost_per_1k_input,
          )
        )}
      </td>

      <td style={tdStyle}>
        {editing ? (
          <PriceNumberInput
            value={
              editForm.cost_per_1k_output
            }
            onChange={value =>
              onEditFormChange({
                ...editForm,
                cost_per_1k_output: value,
              })
            }
          />
        ) : (
          formatPrice(
            price.cost_per_1k_output,
          )
        )}
      </td>

      <td style={tdStyle}>
        <strong
          style={{ color: C.primary }}
        >
          {(
            price.cost_per_1k_input *
            effectiveRate
          ).toFixed(4)}
        </strong>
      </td>

      <td style={tdStyle}>
        <strong
          style={{ color: C.primary }}
        >
          {(
            price.cost_per_1k_output *
            effectiveRate
          ).toFixed(4)}
        </strong>
      </td>

      <td style={tdStyle}>
        <span
          style={{
            color: price.is_active
              ? C.green
              : C.textMuted,
          }}
        >
          {price.is_active
            ? '✓ 启用'
            : '✗ 禁用'}
        </span>
      </td>

      <td style={tdStyle}>
        {editing ? (
          <div
            style={{
              display: 'flex',
              gap: '5px',
            }}
          >
            <ActionButton
              label="保存"
              color={C.green}
              disabled={busy}
              onClick={onSave}
            />

            <ActionButton
              label="取消"
              color={C.textSec}
              outlined
              disabled={busy}
              onClick={onCancel}
            />
          </div>
        ) : (
          <div
            style={{
              display: 'flex',
              gap: '5px',
            }}
          >
            <ActionButton
              label="编辑"
              color={C.textSec}
              outlined
              disabled={busy}
              onClick={onEdit}
            />

            <ActionButton
              label="删除"
              color={C.red}
              outlined
              disabled={busy}
              onClick={onDelete}
            />
          </div>
        )}
      </td>
    </tr>
  )
}

function PriceNumberInput({
  value,
  placeholder,
  onChange,
}: {
  value: number
  placeholder?: string
  onChange: (value: number) => void
}) {
  return (
    <input
      type="number"
      min={0}
      step="0.000001"
      value={value}
      placeholder={placeholder}
      onChange={event =>
        onChange(
          Number(event.target.value),
        )
      }
      style={{
        ...inputStyle,
        width: '115px',
      }}
    />
  )
}

function ActionButton({
  label,
  color,
  outlined = false,
  disabled,
  onClick,
}: {
  label: string
  color: string
  outlined?: boolean
  disabled: boolean
  onClick: () => void
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      style={{
        padding: '6px 11px',
        borderRadius: '6px',
        border: outlined
          ? `1px solid ${color}`
          : 'none',
        background: outlined
          ? C.white
          : color,
        color: outlined
          ? color
          : C.white,
        fontSize: '12px',
        cursor: disabled
          ? 'not-allowed'
          : 'pointer',
        opacity: disabled ? 0.6 : 1,
      }}
    >
      {label}
    </button>
  )
}

function formatPrice(
  value: number,
): string {
  if (!Number.isFinite(value)) {
    return '-'
  }

  return `$${value.toFixed(6)}`
}
