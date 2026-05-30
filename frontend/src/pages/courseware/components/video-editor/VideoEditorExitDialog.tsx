/**
 * VideoEditorExitDialog.tsx — 退出确认弹窗
 *
 * 从 VideoEditorModal.tsx 拆分(B1瘦身)
 *
 * 包含：草稿名输入 + 保存/放弃/继续编辑三按钮
 */
import { useState } from 'react'
import { C } from './VideoEditorConstants'

interface ExitDialogProps {
  clipCount: number
  onSaveDraft: (name: string) => Promise<void>
  onDiscard: () => void
  onCancel: () => void
}

export default function VideoEditorExitDialog({
  clipCount, onSaveDraft, onDiscard, onCancel,
}: ExitDialogProps) {
  const [draftName, setDraftName] = useState('')
  const [saving, setSaving] = useState(false)

  const handleSave = async () => {
    setSaving(true)
    try { await onSaveDraft(draftName) }
    catch { alert('草稿保存失败，请重试'); setSaving(false) }
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 99999,
      background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }} onClick={() => !saving && onCancel()}>
      <div onClick={e => e.stopPropagation()} style={{
        background: C.surface, borderRadius: 16, padding: '32px 28px',
        maxWidth: 400, width: '90%', textAlign: 'center',
        boxShadow: '0 16px 48px rgba(0,0,0,0.4)',
      }}>
        <div style={{ fontSize: 40, marginBottom: 12 }}>📝</div>
        <div style={{ fontSize: 16, fontWeight: 600, color: C.text, marginBottom: 6 }}>保存编辑草稿？</div>
        <div style={{ fontSize: 13, color: C.textMuted, marginBottom: 16 }}>当前有 {clipCount} 个片段，保存后可随时恢复</div>
        <input
          value={draftName} onChange={e => setDraftName(e.target.value)}
          maxLength={200} placeholder="草稿名称（可选，如：第二版剪辑）"
          disabled={saving}
          style={{
            width: '100%', padding: '8px 12px', borderRadius: 6,
            border: '1px solid rgba(255,255,255,0.15)',
            background: C.surfaceLight, color: C.text,
            fontSize: 13, marginBottom: 4, outline: 'none', boxSizing: 'border-box',
          }}
        />
        <div style={{ fontSize: 10, color: C.textMuted, textAlign: 'right', marginBottom: 16 }}>{draftName.length}/200 字符</div>
        <div style={{ display: 'flex', gap: 10, justifyContent: 'center' }}>
          <button disabled={saving} onClick={handleSave} style={{
            padding: '10px 20px', borderRadius: 8, border: 'none',
            background: saving ? '#4B5563' : C.primary,
            color: '#fff', fontSize: 14, fontWeight: 600,
            cursor: saving ? 'default' : 'pointer',
          }}>{saving ? '⏳ 保存中...' : '💾 保存草稿'}</button>
          <button disabled={saving} onClick={onDiscard} style={{
            padding: '10px 20px', borderRadius: 8,
            border: `1px solid ${C.danger}`, background: 'transparent',
            color: C.danger, fontSize: 14, fontWeight: 600,
            cursor: saving ? 'default' : 'pointer',
            opacity: saving ? 0.5 : 1,
          }}>🗑 放弃</button>
          <button disabled={saving} onClick={onCancel} style={{
            padding: '10px 20px', borderRadius: 8,
            border: `1px solid ${C.border}`, background: 'transparent',
            color: C.text, fontSize: 14,
            cursor: saving ? 'default' : 'pointer',
            opacity: saving ? 0.5 : 1,
          }}>↩ 继续编辑</button>
        </div>
      </div>
    </div>
  )
}
