/**
 * TokenCreditSimulator.tsx — 文本模型积分模拟计算器。
 *
 * 使用当前系统积分策略和文本模型价格，
 * 调用后端模拟接口返回与真实计费一致的计算结果。
 */

import {
  useEffect,
  useMemo,
  useState,
} from 'react'
import {
  simulateCredits,
  type CreditCalculation,
  type ModelPrice,
} from '@/api/tokens'
import {
  C,
  cardStyle,
  inputStyle,
} from './tokenDashboardParts'

interface TokenCreditSimulatorProps {
  prices: ModelPrice[]
}

export function TokenCreditSimulator({
  prices,
}: TokenCreditSimulatorProps) {
  const activePrices = useMemo(
    () => prices.filter(
      price => price.is_active,
    ),
    [prices],
  )

  const [modelName, setModelName] =
    useState('')

  const [inputTokens, setInputTokens] =
    useState(1000)

  const [outputTokens, setOutputTokens] =
    useState(500)

  const [loading, setLoading] =
    useState(false)

  const [message, setMessage] =
    useState('')

  const [result, setResult] =
    useState<CreditCalculation | null>(null)

  useEffect(() => {
    if (activePrices.length === 0) {
      setModelName('')
      return
    }

    const exists = activePrices.some(
      price =>
        price.model_name === modelName,
    )

    if (!exists) {
      setModelName(
        activePrices[0].model_name,
      )
    }
  }, [activePrices, modelName])

  const runSimulation = async () => {
    setMessage('')

    if (!modelName) {
      setMessage(
        '当前没有启用的文本模型价格。',
      )
      return
    }

    if (
      !Number.isFinite(inputTokens) ||
      !Number.isFinite(outputTokens) ||
      inputTokens < 0 ||
      outputTokens < 0
    ) {
      setMessage(
        'Token数量必须为非负数。',
      )
      return
    }

    setLoading(true)

    try {
      const calculation =
        await simulateCredits({
          model_name: modelName,
          input_tokens: inputTokens,
          output_tokens: outputTokens,
        })

      setResult(calculation)
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : '积分模拟失败',
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={cardStyle}>
      <div
        style={{
          marginBottom: '14px',
          color: C.text,
          fontSize: '15px',
          fontWeight: 700,
        }}
      >
        🧮 积分模拟计算器
      </div>

      <div
        style={{
          display: 'flex',
          alignItems: 'flex-end',
          gap: '12px',
          flexWrap: 'wrap',
        }}
      >
        <div>
          <FieldLabel value="文本模型" />

          <select
            value={modelName}
            onChange={event =>
              setModelName(
                event.target.value,
              )
            }
            disabled={
              activePrices.length === 0
            }
            style={{
              ...inputStyle,
              width: '280px',
            }}
          >
            {activePrices.map(price => (
              <option
                key={price.id}
                value={price.model_name}
              >
                {price.display_name ||
                  price.model_name}
              </option>
            ))}
          </select>
        </div>

        <TokenNumberInput
          label="输入Tokens"
          value={inputTokens}
          onChange={setInputTokens}
        />

        <TokenNumberInput
          label="输出Tokens"
          value={outputTokens}
          onChange={setOutputTokens}
        />

        <button
          onClick={() =>
            void runSimulation()
          }
          disabled={
            loading ||
            activePrices.length === 0
          }
          style={{
            padding: '9px 20px',
            borderRadius: '8px',
            border: 'none',
            background:
              loading ||
              activePrices.length === 0
                ? C.textMuted
                : C.primary,
            color: C.white,
            fontSize: '14px',
            fontWeight: 600,
            cursor:
              loading ||
              activePrices.length === 0
                ? 'not-allowed'
                : 'pointer',
          }}
        >
          {loading
            ? '计算中...'
            : '计算'}
        </button>
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

      {result && (
        <div
          style={{
            display: 'flex',
            gap: '24px',
            flexWrap: 'wrap',
            marginTop: '16px',
            padding: '16px',
            borderRadius: '12px',
            background:
              'rgba(79,123,232,0.05)',
          }}
        >
          <ResultValue
            label="美元成本"
            value={
              `$${result.cost_usd.toFixed(6)}`
            }
          />

          <ResultValue
            label="换算参数"
            value={
              `${result.exchange_rate} × ${result.multiplier}`
            }
          />

          <ResultValue
            label="积分消耗"
            value={
              `${result.credits_consumed.toFixed(4)} 积分`
            }
            highlighted
          />
        </div>
      )}
    </div>
  )
}

function FieldLabel({
  value,
}: {
  value: string
}) {
  return (
    <div
      style={{
        marginBottom: '5px',
        color: C.textSec,
        fontSize: '12px',
      }}
    >
      {value}
    </div>
  )
}

function TokenNumberInput({
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
      <FieldLabel value={label} />

      <input
        type="number"
        min={0}
        step={1}
        value={value}
        onChange={event =>
          onChange(
            Number(event.target.value),
          )
        }
        style={{
          ...inputStyle,
          width: '125px',
          fontWeight: 600,
        }}
      />
    </div>
  )
}

function ResultValue({
  label,
  value,
  highlighted = false,
}: {
  label: string
  value: string
  highlighted?: boolean
}) {
  return (
    <div>
      <div
        style={{
          color: C.textMuted,
          fontSize: '12px',
        }}
      >
        {label}
      </div>

      <div
        style={{
          marginTop: '3px',
          color: highlighted
            ? C.primary
            : C.text,
          fontSize: highlighted
            ? '23px'
            : '18px',
          fontWeight: 700,
        }}
      >
        {value}
      </div>
    </div>
  )
}
