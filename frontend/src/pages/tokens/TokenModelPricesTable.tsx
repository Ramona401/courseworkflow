/**
 * TokenModelPricesTable.tsx — 文本模型价格手工维护。
 *
 * 本文件维护新增、编辑、删除状态及接口调用。
 * 具体表格行展示拆入TokenModelPriceRows.tsx。
 *
 * 图片、视频和TTS价格不在本表中编辑。
 */

import { useState } from 'react'
import {
  createModelPrice,
  deleteModelPrice,
  updateModelPrice,
  type CreateModelPriceRequest,
  type ModelPrice,
} from '@/api/tokens'
import {
  C,
  thStyle,
} from './tokenDashboardParts'
import {
  CreatePriceRow,
  ModelPriceRow,
  type EditPriceForm,
} from './TokenModelPriceRows'

interface TokenModelPricesTableProps {
  prices: ModelPrice[]
  effectiveRate: number
  onChanged: () => void | Promise<void>
}

const EMPTY_CREATE_FORM:
  CreateModelPriceRequest = {
    model_name: '',
    provider: 'qwen',
    cost_per_1k_input: 0,
    cost_per_1k_output: 0,
    display_name: '',
  }

function requestErrorMessage(
  error: unknown,
): string {
  const candidate = error as {
    message?: string
    response?: {
      data?: {
        message?: string
      }
    }
  }

  return (
    candidate?.response?.data?.message ||
    candidate?.message ||
    '模型价格操作失败'
  )
}

export function TokenModelPricesTable({
  prices,
  effectiveRate,
  onChanged,
}: TokenModelPricesTableProps) {
  const [adding, setAdding] =
    useState(false)

  const [createForm, setCreateForm] =
    useState<CreateModelPriceRequest>({
      ...EMPTY_CREATE_FORM,
    })

  const [editingID, setEditingID] =
    useState('')

  const [editForm, setEditForm] =
    useState<EditPriceForm>({
      cost_per_1k_input: 0,
      cost_per_1k_output: 0,
      display_name: '',
    })

  const [busy, setBusy] =
    useState(false)

  const [message, setMessage] =
    useState('')

  const [error, setError] =
    useState('')

  const clearMessages = () => {
    setMessage('')
    setError('')
  }

  const toggleAdding = () => {
    clearMessages()
    setAdding(value => !value)
  }

  const addPrice = async () => {
    clearMessages()

    if (!createForm.model_name.trim()) {
      setError(
        '模型名不能为空，且必须与实际调用模型名逐字一致。',
      )
      return
    }

    if (
      createForm.cost_per_1k_input < 0 ||
      createForm.cost_per_1k_output < 0
    ) {
      setError('模型价格不能为负数。')
      return
    }

    setBusy(true)

    try {
      await createModelPrice({
        model_name:
          createForm.model_name.trim(),
        provider:
          createForm.provider.trim() ||
          'unknown',
        cost_per_1k_input:
          createForm.cost_per_1k_input,
        cost_per_1k_output:
          createForm.cost_per_1k_output,
        display_name:
          createForm.display_name.trim(),
      })

      setAdding(false)
      setCreateForm({
        ...EMPTY_CREATE_FORM,
      })
      setMessage('文本模型价格已创建。')

      await onChanged()
    } catch (requestError) {
      setError(
        requestErrorMessage(requestError),
      )
    } finally {
      setBusy(false)
    }
  }

  const startEdit = (
    price: ModelPrice,
  ) => {
    clearMessages()
    setEditingID(price.id)
    setEditForm({
      cost_per_1k_input:
        price.cost_per_1k_input,
      cost_per_1k_output:
        price.cost_per_1k_output,
      display_name:
        price.display_name || '',
    })
  }

  const saveEdit = async (
    priceID: string,
  ) => {
    clearMessages()

    if (
      editForm.cost_per_1k_input < 0 ||
      editForm.cost_per_1k_output < 0
    ) {
      setError('模型价格不能为负数。')
      return
    }

    setBusy(true)

    try {
      await updateModelPrice(priceID, {
        cost_per_1k_input:
          editForm.cost_per_1k_input,
        cost_per_1k_output:
          editForm.cost_per_1k_output,
        display_name:
          editForm.display_name.trim(),
      })

      setEditingID('')
      setMessage('文本模型价格已更新。')

      await onChanged()
    } catch (requestError) {
      setError(
        requestErrorMessage(requestError),
      )
    } finally {
      setBusy(false)
    }
  }

  const removePrice = async (
    price: ModelPrice,
  ) => {
    const name =
      price.display_name ||
      price.model_name

    const confirmed = window.confirm(
      `确认删除「${name}」的模型价格？删除后该模型会使用后端兜底估价。`,
    )

    if (!confirmed) return

    clearMessages()
    setBusy(true)

    try {
      await deleteModelPrice(price.id)
      setMessage('文本模型价格已删除。')

      await onChanged()
    } catch (requestError) {
      setError(
        requestErrorMessage(requestError),
      )
    } finally {
      setBusy(false)
    }
  }

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
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: '12px',
          flexWrap: 'wrap',
          padding: '16px 20px',
          borderBottom:
            `1px solid ${C.border}`,
        }}
      >
        <div>
          <div
            style={{
              color: C.text,
              fontSize: '15px',
              fontWeight: 700,
            }}
          >
            📋 文本模型单价表
          </div>

          <div
            style={{
              marginTop: '4px',
              color: C.textMuted,
              fontSize: '12px',
            }}
          >
            美元价格单位为每1K Token，共 {prices.length} 个模型。
          </div>
        </div>

        <button
          onClick={toggleAdding}
          disabled={busy}
          style={{
            padding: '7px 15px',
            borderRadius: '8px',
            border: 'none',
            background: adding
              ? C.textMuted
              : C.green,
            color: C.white,
            fontSize: '13px',
            fontWeight: 600,
            cursor: busy
              ? 'not-allowed'
              : 'pointer',
          }}
        >
          {adding
            ? '取消新增'
            : '+ 新增单价'}
        </button>
      </div>

      <StatusMessage
        message={message}
        error={error}
      />

      <table
        style={{
          width: '100%',
          minWidth: '960px',
          borderCollapse: 'collapse',
        }}
      >
        <thead>
          <tr>
            <th style={thStyle}>模型</th>
            <th style={thStyle}>供应商</th>
            <th style={thStyle}>输入 ($/1K)</th>
            <th style={thStyle}>输出 ($/1K)</th>
            <th style={thStyle}>输入积分/1K</th>
            <th style={thStyle}>输出积分/1K</th>
            <th style={thStyle}>状态</th>
            <th style={thStyle}>操作</th>
          </tr>
        </thead>

        <tbody>
          {adding && (
            <CreatePriceRow
              form={createForm}
              busy={busy}
              onChange={setCreateForm}
              onSave={() =>
                void addPrice()
              }
            />
          )}

          {prices.map(price => (
            <ModelPriceRow
              key={price.id}
              price={price}
              effectiveRate={effectiveRate}
              editing={
                editingID === price.id
              }
              editForm={editForm}
              busy={busy}
              onEdit={() =>
                startEdit(price)
              }
              onEditFormChange={
                setEditForm
              }
              onSave={() =>
                void saveEdit(price.id)
              }
              onCancel={() =>
                setEditingID('')
              }
              onDelete={() =>
                void removePrice(price)
              }
            />
          ))}
        </tbody>
      </table>

      {prices.length === 0 && !adding && (
        <div
          style={{
            padding: '34px',
            color: C.textMuted,
            textAlign: 'center',
          }}
        >
          暂无文本模型价格。
        </div>
      )}
    </div>
  )
}

function StatusMessage({
  message,
  error,
}: {
  message: string
  error: string
}) {
  if (!message && !error) {
    return null
  }

  const failed = Boolean(error)

  return (
    <div
      style={{
        margin: '12px 20px 0',
        padding: '9px 12px',
        borderRadius: '8px',
        background: failed
          ? 'rgba(239,68,68,0.06)'
          : 'rgba(16,185,129,0.08)',
        color: failed
          ? C.red
          : C.green,
        fontSize: '12px',
      }}
    >
      {failed ? '⚠' : '✓'}
      {' '}
      {error || message}
    </div>
  )
}
