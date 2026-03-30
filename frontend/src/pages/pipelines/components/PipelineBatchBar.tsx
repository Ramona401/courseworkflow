/**
 * PipelineBatchBar.tsx — 底部浮动批量操作栏
 * 选中Pipeline后在页面底部浮出，支持批量启动/重跑Generator/分配审核员/取消选择
 */
import { Rocket, RotateCcw, UserPlus } from 'lucide-react'

interface PipelineBatchBarProps {
  selectedCount: number          // 已选总数
  selectedPendingCount: number   // 已选中待启动的数量
  isAdmin: boolean               // 是否有管理员权限
  onBatchStart: () => void
  onBatchRestart: () => void
  onOpenAssign: () => void
  onClearSelection: () => void
}

export function PipelineBatchBar({
  selectedCount, selectedPendingCount, isAdmin,
  onBatchStart, onBatchRestart, onOpenAssign, onClearSelection,
}: PipelineBatchBarProps) {
  if (selectedCount === 0) return null

  const actionBtn = (bg: string): React.CSSProperties => ({
    padding: '8px 18px', borderRadius: 10, border: 'none',
    background: bg, color: '#fff', fontSize: 13, fontWeight: 600,
    cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6,
  })

  return (
    <div style={{
      position: 'fixed', bottom: 24, left: '50%', transform: 'translateX(-50%)',
      background: 'rgba(30,30,30,0.95)', backdropFilter: 'blur(20px)',
      borderRadius: 16, padding: '12px 24px',
      display: 'flex', alignItems: 'center', gap: 16,
      boxShadow: '0 8px 32px rgba(0,0,0,0.3)', zIndex: 900, color: '#fff',
    }}>
      {/* 选中计数 */}
      <span style={{ fontSize: 13, fontWeight: 500 }}>
        已选 <span style={{ fontWeight: 700, color: '#007aff' }}>{selectedCount}</span> 个
      </span>

      {/* 批量启动（有待启动的才显示）*/}
      {selectedPendingCount > 0 && (
        <button onClick={onBatchStart} style={actionBtn('#34c759')}>
          <Rocket size={14} /> 批量启动 ({selectedPendingCount})
        </button>
      )}

      {/* 批量重跑Generator（管理员可见）*/}
      {isAdmin && (
        <button onClick={onBatchRestart} style={actionBtn('#ff9500')}>
          <RotateCcw size={14} /> 批量重跑Generator
        </button>
      )}

      {/* 分配审核员（管理员可见）*/}
      {isAdmin && (
        <button onClick={onOpenAssign} style={actionBtn('#5856d6')}>
          <UserPlus size={14} /> 分配审核员
        </button>
      )}

      {/* 取消选择 */}
      <button
        onClick={onClearSelection}
        style={{ padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(255,255,255,0.2)', background: 'transparent', color: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer' }}>
        取消选择
      </button>
    </div>
  )
}
