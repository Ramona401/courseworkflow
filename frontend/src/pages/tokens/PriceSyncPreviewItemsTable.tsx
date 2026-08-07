/**
 * PriceSyncPreviewItemsTable.tsx — 价格同步预览明细表。
 *
 * 本组件只展示价格对比和维护勾选状态：
 *   - 仅action=update且批次状态为previewed时允许勾选；
 *   - skipped、unchanged、applied和stale只用于审计展示；
 *   - 不调用价格应用接口，不修改正式价格。
 */

import {
  type PriceSyncItem,
  type PriceSyncPreviewResponse,
} from '@/api/priceSync'
import {
  C,
  tdStyle,
  thStyle,
} from './tokenDashboardParts'
import {
  priceSyncActionView,
  priceSyncChangeLabel,
  priceSyncItemTypeLabel,
} from './priceSyncPreviewView'

interface PriceSyncPreviewItemsTableProps {
  preview: PriceSyncPreviewResponse
  selectedSet: ReadonlySet<string>
  allSelected: boolean
  updateItemCount: number

  onToggleAll: () => void
  onToggleItem: (itemID: string) => void
}

export function PriceSyncPreviewItemsTable({
  preview,
  selectedSet,
  allSelected,
  updateItemCount,
  onToggleAll,
  onToggleItem,
}: PriceSyncPreviewItemsTableProps) {
  return (
    <div style={{ overflow: 'auto' }}>
      <table
        style={{
          width: '100%',
          minWidth: '1000px',
          borderCollapse: 'collapse',
        }}
      >
        <thead>
          <tr>
            <th
              style={{
                ...thStyle,
                width: '42px',
              }}
            >
              <input
                type="checkbox"
                checked={allSelected}
                disabled={
                  updateItemCount === 0 ||
                  preview.run.status !==
                    'previewed'
                }
                onChange={onToggleAll}
                aria-label="选择全部可应用价格变化"
              />
            </th>

            <th style={thStyle}>模型</th>
            <th style={thStyle}>类型</th>
            <th style={thStyle}>价格变化</th>
            <th style={thStyle}>来源</th>
            <th style={thStyle}>结果</th>
            <th style={thStyle}>原因</th>
          </tr>
        </thead>

        <tbody>
          {preview.items.map(item => (
            <PriceSyncPreviewItemRow
              key={item.id}
              item={item}
              selected={
                selectedSet.has(item.id)
              }
              runStatus={
                preview.run.status
              }
              onToggle={() =>
                onToggleItem(item.id)
              }
            />
          ))}
        </tbody>
      </table>
    </div>
  )
}

function PriceSyncPreviewItemRow({
  item,
  selected,
  runStatus,
  onToggle,
}: {
  item: PriceSyncItem
  selected: boolean
  runStatus:
    PriceSyncPreviewResponse['run']['status']
  onToggle: () => void
}) {
  const actionView =
    priceSyncActionView(item.action)

  const selectable =
    item.action === 'update' &&
    runStatus === 'previewed'

  return (
    <tr>
      <td style={tdStyle}>
        <input
          type="checkbox"
          checked={selected}
          disabled={!selectable}
          onChange={onToggle}
          aria-label={`选择模型${item.model_name}的价格变化`}
        />
      </td>

      <td style={tdStyle}>
        <div
          style={{
            fontWeight: 600,
            color: C.text,
          }}
        >
          {item.model_name}
        </div>

        <div
          style={{
            marginTop: '3px',
            fontSize: '11px',
            color: C.textMuted,
          }}
        >
          {item.provider}
        </div>
      </td>

      <td style={tdStyle}>
        {priceSyncItemTypeLabel(item)}
      </td>

      <td
        style={{
          ...tdStyle,
          fontSize: '12px',
          whiteSpace: 'nowrap',
        }}
      >
        {priceSyncChangeLabel(item)}
      </td>

      <td
        style={{
          ...tdStyle,
          fontSize: '11px',
          fontFamily: 'monospace',
        }}
      >
        {item.sync_source}
      </td>

      <td style={tdStyle}>
        <strong
          style={{
            color: actionView.color,
            fontSize: '12px',
          }}
        >
          {actionView.label}
        </strong>
      </td>

      <td
        style={{
          ...tdStyle,
          maxWidth: '300px',
          fontSize: '12px',
          lineHeight: 1.5,
          color: C.textSec,
        }}
      >
        {item.reason || '-'}
      </td>
    </tr>
  )
}
