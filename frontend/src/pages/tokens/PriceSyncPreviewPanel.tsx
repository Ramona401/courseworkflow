/**
 * PriceSyncPreviewPanel.tsx — 价格预览与历史区域组合组件。
 *
 * 本文件只组合：
 *   - PriceSyncPreviewCard：价格检查、明细选择和应用；
 *   - PriceSyncHistoryTable：最近同步批次。
 *
 * 具体表格展示和格式化逻辑位于各自子组件。
 */

import {
  type PriceSyncPreviewResponse,
  type PriceSyncRun,
} from '@/api/priceSync'
import { PriceSyncPreviewCard } from './PriceSyncPreviewCard'
import { PriceSyncHistoryTable } from './PriceSyncHistoryTable'

interface PriceSyncPreviewPanelProps {
  preview: PriceSyncPreviewResponse | null
  runs: PriceSyncRun[]
  selectedItemIDs: string[]
  busyAction: string

  onSelectedItemIDsChange: (
    itemIDs: string[],
  ) => void

  onPreview: () => void
  onApplySelected: () => void
  onApplyAll: () => void
  onOpenRun: (runID: string) => void
}

export function PriceSyncPreviewPanel({
  preview,
  runs,
  selectedItemIDs,
  busyAction,
  onSelectedItemIDsChange,
  onPreview,
  onApplySelected,
  onApplyAll,
  onOpenRun,
}: PriceSyncPreviewPanelProps) {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: '16px',
      }}
    >
      <PriceSyncPreviewCard
        preview={preview}
        selectedItemIDs={selectedItemIDs}
        busyAction={busyAction}
        onSelectedItemIDsChange={
          onSelectedItemIDsChange
        }
        onPreview={onPreview}
        onApplySelected={onApplySelected}
        onApplyAll={onApplyAll}
      />

      <PriceSyncHistoryTable
        runs={runs}
        busyAction={busyAction}
        onOpenRun={onOpenRun}
      />
    </div>
  )
}
