/**
 * TokenRatePolicyCard.tsx — 系统积分汇率与倍率设置。
 *
 * 修改后只影响后续产生的消费记录。
 * 已保存的历史消费成本、汇率、倍率及积分快照不会重新计算。
 */

import {
  useEffect,
  useState,
} from 'react'
import {
  updateSystemCreditPolicy,
  type CreditPolicy,
} from '@/api/tokens'
import {
  C,
  cardStyle,
  inputStyle,
} from './tokenDashboardParts'

interface TokenRatePolicyCardProps {
  policy: CreditPolicy | null
  onPolicyChanged: (
    policy: CreditPolicy,
  ) => void
}

export function TokenRatePolicyCard({
  policy,
  onPolicyChanged,
}: TokenRatePolicyCardProps) {
  const [editing, setEditing] =
    useState(false)

  const [rateInput, setRateInput] =
    useState(policy?.exchange_rate ?? 7)

  const [
    multiplierInput,
    setMultiplierInput,
  ] = useState(policy?.multiplier ?? 1)

  const [saving, setSaving] =
    useState(false)

  const [message, setMessage] =
    useState('')

  useEffect(() => {
    setRateInput(
      policy?.exchange_rate ?? 7,
    )
    setMultiplierInput(
      policy?.multiplier ?? 1,
    )
  }, [policy])

  const startEditing = () => {
    setRateInput(
      policy?.exchange_rate ?? 7,
    )
    setMultiplierInput(
      policy?.multiplier ?? 1,
    )
    setMessage('')
    setEditing(true)
  }

  const savePolicy = async () => {
    if (
      !Number.isFinite(rateInput) ||
      !Number.isFinite(multiplierInput) ||
      rateInput <= 0 ||
      multiplierInput <= 0
    ) {
      setMessage(
        '汇率和倍率必须为正数。',
      )
      return
    }

    setSaving(true)
    setMessage('')

    try {
      const updated =
        await updateSystemCreditPolicy({
          exchange_rate: rateInput,
          multiplier: multiplierInput,
        })

      onPolicyChanged(updated)
      setEditing(false)
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : '积分策略保存失败',
      )
    } finally {
      setSaving(false)
    }
  }

  const exchangeRate =
    policy?.exchange_rate ?? 7

  const multiplier =
    policy?.multiplier ?? 1

  return (
    <div style={cardStyle}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: '12px',
          flexWrap: 'wrap',
          marginBottom: '14px',
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
            💱 积分汇率与倍率
          </div>

          <div
            style={{
              marginTop: '3px',
              fontSize: '12px',
              color: C.textMuted,
            }}
          >
            积分 = 美元成本 × 汇率 × 倍率
          </div>
        </div>

        {!editing && (
          <button
            onClick={startEditing}
            style={{
              padding: '7px 15px',
              borderRadius: '8px',
              border: `1px solid ${C.primary}`,
              background: C.white,
              color: C.primary,
              fontSize: '13px',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            ✏️ 编辑
          </button>
        )}
      </div>

      {!editing ? (
        <ReadOnlyPolicy
          exchangeRate={exchangeRate}
          multiplier={multiplier}
          description={
            policy?.description || ''
          }
        />
      ) : (
        <div>
          <div
            style={{
              marginBottom: '14px',
              padding: '10px 14px',
              borderRadius: '8px',
              border: `1px solid ${C.orange}`,
              background:
                'rgba(245,158,11,0.08)',
              color: C.orange,
              fontSize: '12px',
              lineHeight: 1.6,
            }}
          >
            ⚠ 保存后立即影响后续AI调用的积分计算，
            历史消费记录不会重新计算。
          </div>

          <div
            style={{
              display: 'flex',
              alignItems: 'flex-end',
              gap: '16px',
              flexWrap: 'wrap',
            }}
          >
            <PolicyNumberInput
              label="汇率（美元→积分）"
              value={rateInput}
              onChange={setRateInput}
            />

            <OperationSymbol value="×" />

            <PolicyNumberInput
              label="倍率"
              value={multiplierInput}
              onChange={setMultiplierInput}
            />

            <OperationSymbol value="=" />

            <PolicyValue
              label="有效汇率预览"
              value={
                rateInput *
                multiplierInput
              }
              color={C.green}
            />
          </div>

          {message && (
            <div
              style={{
                marginTop: '10px',
                color: C.red,
                fontSize: '12px',
              }}
            >
              ⚠ {message}
            </div>
          )}

          <div
            style={{
              display: 'flex',
              gap: '8px',
              marginTop: '16px',
            }}
          >
            <button
              onClick={() =>
                void savePolicy()
              }
              disabled={saving}
              style={{
                padding: '9px 20px',
                borderRadius: '8px',
                border: 'none',
                background: saving
                  ? C.textMuted
                  : C.primary,
                color: C.white,
                fontSize: '14px',
                fontWeight: 600,
                cursor: saving
                  ? 'not-allowed'
                  : 'pointer',
              }}
            >
              {saving
                ? '保存中...'
                : '保存'}
            </button>

            <button
              onClick={() =>
                setEditing(false)
              }
              disabled={saving}
              style={{
                padding: '9px 20px',
                borderRadius: '8px',
                border: `1px solid ${C.border}`,
                background: C.white,
                color: C.textSec,
                fontSize: '14px',
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              取消
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function ReadOnlyPolicy({
  exchangeRate,
  multiplier,
  description,
}: {
  exchangeRate: number
  multiplier: number
  description: string
}) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '28px',
        flexWrap: 'wrap',
      }}
    >
      <PolicyValue
        label="汇率（美元→积分）"
        value={exchangeRate}
        color={C.primary}
      />

      <OperationSymbol value="×" />

      <PolicyValue
        label="倍率"
        value={multiplier}
        color={C.purple}
      />

      <OperationSymbol value="=" />

      <PolicyValue
        label="有效汇率"
        value={
          exchangeRate * multiplier
        }
        color={C.green}
      />

      <div
        style={{
          flex: 1,
          minWidth: '220px',
          color: C.textMuted,
          fontSize: '12px',
          lineHeight: 1.7,
        }}
      >
        {description}
        <br />
        修改只影响保存后的新消费记录。
      </div>
    </div>
  )
}

function PolicyNumberInput({
  label,
  value,
  onChange,
}: {
  label: string
  value: number
  onChange: (value: number) => void
}) {
  return (
    <div>
      <div
        style={{
          marginBottom: '5px',
          color: C.textSec,
          fontSize: '12px',
        }}
      >
        {label}
      </div>

      <input
        type="number"
        min={0}
        step="0.01"
        value={value}
        onChange={event =>
          onChange(
            Number(event.target.value),
          )
        }
        style={{
          ...inputStyle,
          width: '145px',
          fontWeight: 600,
        }}
      />
    </div>
  )
}

function PolicyValue({
  label,
  value,
  color,
}: {
  label: string
  value: number
  color: string
}) {
  return (
    <div>
      <div
        style={{
          marginBottom: '4px',
          color: C.textMuted,
          fontSize: '12px',
        }}
      >
        {label}
      </div>

      <div
        style={{
          color,
          fontSize: '27px',
          fontWeight: 700,
        }}
      >
        {Number.isFinite(value)
          ? value.toFixed(2)
          : '—'}
      </div>
    </div>
  )
}

function OperationSymbol({
  value,
}: {
  value: string
}) {
  return (
    <div
      style={{
        paddingBottom: '4px',
        color: C.textMuted,
        fontSize: '24px',
      }}
    >
      {value}
    </div>
  )
}
