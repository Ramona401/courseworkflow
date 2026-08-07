/**
 * PriceSyncPreviewCard.tsx — 价格检查、摘要和应用操作。
 *
 * 本组件负责：
 *   - 触发人工价格检查；
 *   - 维护预览明细勾选集合；
 *   - 触发选择性应用或全部应用；
 *   - 展示同步批次摘要及来源信息。
 *
 * 价格明细表已拆至PriceSyncPreviewItemsTable.tsx。
 */

import {
  type PriceSyncPreviewResponse,
} from '@/api/priceSync'
import { C } from './tokenDashboardParts'
import { PriceSyncPreviewItemsTable } from './PriceSyncPreviewItemsTable'
import {
  formatPriceSyncTime,
} from './priceSyncPreviewView'

interface PriceSyncPreviewCardProps {
  preview: PriceSyncPreviewResponse | null
  selectedItemIDs: string[]
  busyAction: string

  onSelectedItemIDsChange: (
    itemIDs: string[],
  ) => void

  onPreview: () => void
  onApplySelected: () => void
  onApplyAll: () => void
}

const previewCardStyle = {
  background: C.white,
  borderRadius: '14px',
  border: `1px solid ${C.border}`,
  overflow: 'hidden',
} as const

export function PriceSyncPreviewCard({
  preview,
  selectedItemIDs,
  busyAction,
  onSelectedItemIDsChange,
  onPreview,
  onApplySelected,
  onApplyAll,
}: PriceSyncPreviewCardProps) {
  const updateItems =
    preview?.items.filter(
      item => item.action === 'update',
    ) || []

  const selectedSet =
    new Set(selectedItemIDs)

  const allSelected =
    updateItems.length > 0 &&
    updateItems.every(
      item => selectedSet.has(item.id),
    )

  const toggleItem = (
    itemID: string,
  ) => {
    const next = new Set(selectedSet)

    if (next.has(itemID)) {
      next.delete(itemID)
    } else {
      next.add(itemID)
    }

    onSelectedItemIDsChange(
      Array.from(next),
    )
  }

  const toggleAll = () => {
    onSelectedItemIDsChange(
      allSelected
        ? []
        : updateItems.map(
          item => item.id,
        ),
    )
  }

  const previewing =
    busyAction === 'preview'

  const applying =
    busyAction === 'apply-selected' ||
    busyAction === 'apply-all'

  const canApply =
    preview?.run.status === 'previewed' &&
    updateItems.length > 0

  return (
    <section style={previewCardStyle}>
      <PriceSyncPreviewToolbar
        previewing={previewing}
        applying={applying}
        canApply={canApply}
        selectedCount={
          selectedItemIDs.length
        }
        onPreview={onPreview}
        onApplySelected={
          onApplySelected
        }
        onApplyAll={onApplyAll}
      />

      {!preview ? (
        <div
          style={{
            padding: '36px',
            textAlign: 'center',
            color: C.textMuted,
            fontSize: '13px',
          }}
        >
          点击“检查最新价格”后，
          这里会展示文本、图片、视频和TTS价格对比。
        </div>
      ) : (
        <>
          <PriceSyncSummaryStrip
            preview={preview}
          />

          <PriceSyncRunMetadata
            preview={preview}
          />

          <PriceSyncPreviewItemsTable
            preview={preview}
            selectedSet={selectedSet}
            allSelected={allSelected}
            updateItemCount={
              updateItems.length
            }
            onToggleAll={toggleAll}
            onToggleItem={toggleItem}
          />
        </>
      )}
    </section>
  )
}

function PriceSyncPreviewToolbar({
  previewing,
  applying,
  canApply,
  selectedCount,
  onPreview,
  onApplySelected,
  onApplyAll,
}: {
  previewing: boolean
  applying: boolean
  canApply: boolean
  selectedCount: number
  onPreview: () => void
  onApplySelected: () => void
  onApplyAll: () => void
}) {
  return (
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
            fontSize: '15px',
            fontWeight: 700,
            color: C.text,
          }}
        >
          🔍 价格检查与应用
        </div>

        <div
          style={{
            marginTop: '4px',
            fontSize: '12px',
            color: C.textMuted,
          }}
        >
          检查只生成预览，应用只影响后续调用。
        </div>
      </div>

      <div
        style={{
          display: 'flex',
          gap: '8px',
          flexWrap: 'wrap',
        }}
      >
        <PreviewToolbarButton
          label={
            previewing
              ? '检查中...'
              : '检查最新价格'
          }
          color={C.primary}
          outlined
          disabled={
            previewing || applying
          }
          onClick={onPreview}
        />

        <PreviewToolbarButton
          label={
            selectedCount > 0
              ? `应用已选（${selectedCount}）`
              : '应用已选'
          }
          color={C.green}
          disabled={
            !canApply ||
            selectedCount === 0 ||
            applying
          }
          onClick={onApplySelected}
        />

        <PreviewToolbarButton
          label="应用全部变化"
          color={C.orange}
          disabled={
            !canApply || applying
          }
          onClick={onApplyAll}
        />
      </div>
    </div>
  )
}

function PriceSyncSummaryStrip({
  preview,
}: {
  preview: PriceSyncPreviewResponse
}) {
  const values = [
    [
      '总数',
      preview.summary.total_count,
      C.text,
    ],
    [
      '待更新',
      preview.summary.update_count,
      C.orange,
    ],
    [
      '无变化',
      preview.summary.unchanged_count,
      C.green,
    ],
    [
      '已跳过',
      preview.summary.skipped_count,
      C.red,
    ],
    [
      '已应用',
      preview.summary.applied_count,
      C.primary,
    ],
    [
      '冲突',
      preview.summary.stale_count,
      C.purple,
    ],
  ] as const

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns:
          'repeat(auto-fit, minmax(115px, 1fr))',
        gap: '10px',
        padding: '14px 20px',
        background: C.bg,
        borderBottom:
          `1px solid ${C.border}`,
      }}
    >
      {values.map(
        ([label, value, color]) => (
          <div key={label}>
            <div
              style={{
                fontSize: '11px',
                color: C.textMuted,
              }}
            >
              {label}
            </div>

            <div
              style={{
                marginTop: '2px',
                fontSize: '20px',
                fontWeight: 700,
                color,
              }}
            >
              {value}
            </div>
          </div>
        ),
      )}
    </div>
  )
}

function PriceSyncRunMetadata({
  preview,
}: {
  preview: PriceSyncPreviewResponse
}) {
  return (
    <div
      style={{
        padding: '10px 20px',
        borderBottom:
          `1px solid ${C.border}`,
        fontSize: '11px',
        lineHeight: 1.6,
        color: C.textMuted,
      }}
    >
      批次：{preview.run.id}
      {' · '}
      状态：{preview.run.status}
      {' · '}
      时间：
      {formatPriceSyncTime(
        preview.run.started_at,
      )}

      {preview.run.source_base_url && (
        <>
          <br />
          价格来源：
          <span
            style={{
              wordBreak: 'break-all',
            }}
          >
            {preview.run.source_base_url}
          </span>
        </>
      )}
    </div>
  )
}

function PreviewToolbarButton({
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
        padding: '7px 14px',
        borderRadius: '8px',
        border: outlined
          ? `1px solid ${color}`
          : 'none',
        background: outlined
          ? C.white
          : disabled
            ? C.textMuted
            : color,
        color: outlined
          ? color
          : C.white,
        fontSize: '12px',
        fontWeight: 600,
        cursor: disabled
          ? 'not-allowed'
          : 'pointer',
        opacity: disabled
          ? 0.65
          : 1,
      }}
    >
      {label}
    </button>
  )
}
