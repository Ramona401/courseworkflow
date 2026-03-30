/**
 * ConfirmDialog.tsx — 通用删除/操作二次确认对话框
 */
import { C } from './adminConstants'

interface ConfirmDialogProps {
  title: string
  message: string
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({ title, message, onConfirm, onCancel }: ConfirmDialogProps) {
  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 11000,
      background: 'rgba(0,0,0,0.45)', backdropFilter: 'blur(4px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}>
      <div style={{
        background: C.white, borderRadius: '16px',
        width: '380px', padding: '28px',
        boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }}>
        {/* 标题 */}
        <div style={{ fontSize: '16px', fontWeight: 700, color: C.text, marginBottom: '10px' }}>
          {title}
        </div>
        {/* 说明文字 */}
        <div style={{ fontSize: '14px', color: C.textSec, marginBottom: '24px', lineHeight: 1.6 }}>
          {message}
        </div>
        {/* 操作按钮 */}
        <div style={{ display: 'flex', gap: '10px' }}>
          <button
            onClick={onCancel}
            style={{
              flex: 1, padding: '10px', borderRadius: '10px',
              border: `1px solid ${C.border}`, background: C.bg,
              fontSize: '14px', color: C.textSec, cursor: 'pointer',
            }}>
            取消
          </button>
          <button
            onClick={onConfirm}
            style={{
              flex: 1, padding: '10px', borderRadius: '10px',
              border: 'none', background: C.danger, color: '#fff',
              fontSize: '14px', fontWeight: 600, cursor: 'pointer',
            }}>
            确认删除
          </button>
        </div>
      </div>
    </div>
  )
}
