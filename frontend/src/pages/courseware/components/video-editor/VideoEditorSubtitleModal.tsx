/**
 * VideoEditorSubtitleModal.tsx — 字幕条编辑弹窗
 *
 * v0.42.8 新增
 *
 * 功能：
 *   - 编辑字幕文本内容
 *   - 精确调整起止时间（数字输入）
 *   - 选择语言（zh-CN / en-US）
 *   - 确认保存 / 取消
 */
import { useState } from 'react'
import { C } from './VideoEditorConstants'
import type { SubtitleSegment } from './VideoEditorSubtitleTrack'

interface SubtitleModalProps {
  /** 正在编辑的字幕条（null 时不显示弹窗） */
  segment: SubtitleSegment | null
  /** 可用语言列表 */
  languages: { code: string; label: string }[]
  /** 确认保存回调 */
  onSave: (updated: SubtitleSegment) => void
  /** 关闭弹窗回调 */
  onClose: () => void
}

export default function VideoEditorSubtitleModal({ segment, languages, onSave, onClose }: SubtitleModalProps) {
  const [text, setText] = useState('')
  const [startSec, setStartSec] = useState(0)
  const [endSec, setEndSec] = useState(3)
  const [lang, setLang] = useState('zh-CN')

  // 弹窗打开时初始化表单 — 用 segment.id 变化作为重置信号
  // （避免 useEffect 内 setState 的 ESLint 警告）
  const [prevSegId, setPrevSegId] = useState('')
  if (segment && segment.id !== prevSegId) {
    setPrevSegId(segment.id)
    setText(segment.text || '')
    setStartSec(segment.start_sec)
    setEndSec(segment.end_sec)
    setLang(segment.language || 'zh-CN')
  }

  if (!segment) return null

  const handleSave = () => {
    const dur = endSec - startSec
    if (dur < 0.3) {
      alert('字幕时长至少0.3秒')
      return
    }
    onSave({
      ...segment,
      text: text.trim(),
      start_sec: Math.round(startSec * 100) / 100,
      end_sec: Math.round(endSec * 100) / 100,
      language: lang,
    })
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 10000,
      background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }} onClick={onClose}>
      <div
        onClick={e => e.stopPropagation()}
        style={{
          width: 420, background: C.surface, borderRadius: 12,
          border: `1px solid ${C.border}`, padding: 24,
          boxShadow: '0 20px 40px rgba(0,0,0,0.4)',
        }}
      >
        {/* 标题 */}
        <div style={{
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          marginBottom: 16,
        }}>
          <span style={{ fontSize: 15, fontWeight: 700, color: C.text }}>💬 编辑字幕</span>
          <button onClick={onClose} style={{
            background: 'none', border: 'none', color: C.textMuted,
            fontSize: 18, cursor: 'pointer', padding: '0 4px',
          }}>✕</button>
        </div>

        {/* 字幕文本 */}
        <label style={{ fontSize: 12, color: C.textMuted, marginBottom: 4, display: 'block' }}>字幕内容</label>
        <textarea
          value={text}
          onChange={e => setText(e.target.value)}
          placeholder="输入字幕文本..."
          rows={3}
          style={{
            width: '100%', padding: '8px 10px', fontSize: 14,
            background: C.bg, color: C.text,
            border: `1px solid ${C.border}`, borderRadius: 6,
            resize: 'vertical', boxSizing: 'border-box',
            fontFamily: 'inherit',
          }}
          autoFocus
        />

        {/* 时间设置 */}
        <div style={{ display: 'flex', gap: 12, marginTop: 12 }}>
          <div style={{ flex: 1 }}>
            <label style={{ fontSize: 12, color: C.textMuted, marginBottom: 4, display: 'block' }}>开始时间 (秒)</label>
            <input
              type="number"
              min={0}
              step={0.1}
              value={startSec}
              onChange={e => setStartSec(parseFloat(e.target.value) || 0)}
              style={{
                width: '100%', padding: '6px 8px', fontSize: 13,
                background: C.bg, color: C.text,
                border: `1px solid ${C.border}`, borderRadius: 6,
                boxSizing: 'border-box', fontFamily: 'monospace',
              }}
            />
          </div>
          <div style={{ flex: 1 }}>
            <label style={{ fontSize: 12, color: C.textMuted, marginBottom: 4, display: 'block' }}>结束时间 (秒)</label>
            <input
              type="number"
              min={0}
              step={0.1}
              value={endSec}
              onChange={e => setEndSec(parseFloat(e.target.value) || 0)}
              style={{
                width: '100%', padding: '6px 8px', fontSize: 13,
                background: C.bg, color: C.text,
                border: `1px solid ${C.border}`, borderRadius: 6,
                boxSizing: 'border-box', fontFamily: 'monospace',
              }}
            />
          </div>
          <div style={{ flex: 1 }}>
            <label style={{ fontSize: 12, color: C.textMuted, marginBottom: 4, display: 'block' }}>时长</label>
            <div style={{
              padding: '6px 8px', fontSize: 13, color: C.accent,
              background: C.bg, border: `1px solid ${C.border}`, borderRadius: 6,
              fontFamily: 'monospace', textAlign: 'center',
            }}>
              {(endSec - startSec).toFixed(1)}s
            </div>
          </div>
        </div>

        {/* 语言选择 */}
        <div style={{ marginTop: 12 }}>
          <label style={{ fontSize: 12, color: C.textMuted, marginBottom: 4, display: 'block' }}>语言</label>
          <div style={{ display: 'flex', gap: 6 }}>
            {languages.map(l => (
              <button
                key={l.code}
                onClick={() => setLang(l.code)}
                style={{
                  padding: '4px 10px', fontSize: 12, borderRadius: 6,
                  cursor: 'pointer',
                  background: lang === l.code ? C.accent + '30' : C.bg,
                  color: lang === l.code ? C.accent : C.textMuted,
                  border: `1px solid ${lang === l.code ? C.accent : C.border}`,
                  fontWeight: lang === l.code ? 700 : 400,
                }}
              >{l.label}</button>
            ))}
          </div>
        </div>

        {/* 操作按钮 */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 20 }}>
          <button onClick={onClose} style={{
            padding: '8px 16px', fontSize: 13, borderRadius: 6,
            background: C.surfaceLight, color: C.textMuted,
            border: `1px solid ${C.border}`, cursor: 'pointer',
          }}>取消</button>
          <button onClick={handleSave} style={{
            padding: '8px 16px', fontSize: 13, borderRadius: 6,
            background: C.accent, color: '#000',
            border: 'none', cursor: 'pointer', fontWeight: 700,
          }}>✓ 保存</button>
        </div>
      </div>
    </div>
  )
}
