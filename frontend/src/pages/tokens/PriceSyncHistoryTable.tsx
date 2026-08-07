/**
 * PriceSyncHistoryTable.tsx — 最近价格同步批次列表。
 *
 * 本组件只展示同步批次摘要并触发明细加载，
 * 不修改价格和同步配置。
 */

import {
  type PriceSyncRun,
} from '@/api/priceSync'
import {
  C,
  tdStyle,
  thStyle,
} from './tokenDashboardParts'

interface PriceSyncHistoryTableProps {
  runs: PriceSyncRun[]
  busyAction: string
  onOpenRun: (runID: string) => void
}

export function PriceSyncHistoryTable({
  runs,
  busyAction,
  onOpenRun,
}: PriceSyncHistoryTableProps) {
  return (
    <section
      style={{
        background: C.white,
        borderRadius: '14px',
        border: `1px solid ${C.border}`,
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          padding: '16px 20px',
          borderBottom:
            `1px solid ${C.border}`,
          color: C.text,
          fontSize: '15px',
          fontWeight: 700,
        }}
      >
        🕘 最近同步历史
      </div>

      {runs.length === 0 ? (
        <div
          style={{
            padding: '28px',
            textAlign: 'center',
            color: C.textMuted,
          }}
        >
          暂无价格同步记录。
        </div>
      ) : (
        <div style={{ overflow: 'auto' }}>
          <table
            style={{
              width: '100%',
              minWidth: '820px',
              borderCollapse: 'collapse',
            }}
          >
            <thead>
              <tr>
                <th style={thStyle}>时间</th>
                <th style={thStyle}>触发方式</th>
                <th style={thStyle}>状态</th>
                <th style={thStyle}>待更新</th>
                <th style={thStyle}>已应用</th>
                <th style={thStyle}>跳过</th>
                <th style={thStyle}>操作</th>
              </tr>
            </thead>

            <tbody>
              {runs.map(run => {
                const loading =
                  busyAction ===
                  `run:${run.id}`

                return (
                  <tr key={run.id}>
                    <td style={tdStyle}>
                      {formatTime(
                        run.started_at,
                      )}
                    </td>

                    <td style={tdStyle}>
                      {run.trigger_type ===
                      'scheduler'
                        ? '定时任务'
                        : '人工检查'}
                    </td>

                    <td style={tdStyle}>
                      {run.status}
                    </td>

                    <td style={tdStyle}>
                      {run.summary
                        ?.update_count ?? 0}
                    </td>

                    <td style={tdStyle}>
                      {run.summary
                        ?.applied_count ?? 0}
                    </td>

                    <td style={tdStyle}>
                      {run.summary
                        ?.skipped_count ?? 0}
                    </td>

                    <td style={tdStyle}>
                      <button
                        onClick={() =>
                          onOpenRun(run.id)
                        }
                        disabled={loading}
                        style={{
                          padding: '6px 11px',
                          borderRadius: '6px',
                          border:
                            `1px solid ${C.textSec}`,
                          background: C.white,
                          color: C.textSec,
                          fontSize: '12px',
                          cursor: loading
                            ? 'not-allowed'
                            : 'pointer',
                          opacity: loading
                            ? 0.65
                            : 1,
                        }}
                      >
                        {loading
                          ? '加载中'
                          : '查看明细'}
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}

function formatTime(
  value: string | null,
): string {
  if (!value) {
    return '-'
  }

  return new Date(value).toLocaleString(
    'zh-CN',
  )
}
