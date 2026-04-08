/**
 * FallbackModelsPicker — 降级模型选择器（v85新增）
 *
 * v89-3拆分：从AICenterPage.tsx中拆出
 * 支持从可用模型列表选择降级链，可排序可删除
 */
import { C } from './AICenterConstants'

export default function FallbackModelsPicker({
  value,
  onChange,
  availableModels,
  primaryModel,
}: {
  value: string[]
  onChange: (v: string[]) => void
  availableModels: string[]
  primaryModel: string | null
}) {
  // 可选模型 = 可用模型列表中排除主模型的
  const candidates = availableModels.filter(m => m !== primaryModel)

  const toggleModel = (model: string) => {
    if (value.includes(model)) {
      onChange(value.filter(m => m !== model))
    } else {
      onChange([...value, model])
    }
  }

  const moveUp = (idx: number) => {
    if (idx <= 0) return
    const arr = [...value]
    ;[arr[idx - 1], arr[idx]] = [arr[idx], arr[idx - 1]]
    onChange(arr)
  }

  const moveDown = (idx: number) => {
    if (idx >= value.length - 1) return
    const arr = [...value]
    ;[arr[idx], arr[idx + 1]] = [arr[idx + 1], arr[idx]]
    onChange(arr)
  }

  return (
    <div>
      <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>
        降级模型链（主模型失败时按顺序尝试）
      </label>

      {/* 已选择的降级模型（带排序） */}
      {value.length > 0 && (
        <div style={{
          display: 'flex', flexDirection: 'column', gap: '4px',
          marginBottom: '8px', padding: '8px',
          borderRadius: '8px', background: 'rgba(79,123,232,0.04)',
          border: `1px solid ${C.primaryBorder}`,
        }}>
          {value.map((m, idx) => (
            <div key={m} style={{
              display: 'flex', alignItems: 'center', gap: '6px',
              padding: '4px 8px', borderRadius: '6px', background: C.white,
              fontSize: '12px', fontFamily: 'monospace',
            }}>
              <span style={{
                width: '18px', height: '18px', borderRadius: '50%',
                background: C.primaryLight, color: C.primary,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: '10px', fontWeight: 700, flexShrink: 0,
              }}>{idx + 1}</span>
              <span style={{ flex: 1, color: C.text }}>{m}</span>
              <button onClick={() => moveUp(idx)} disabled={idx === 0}
                style={{ background: 'none', border: 'none', cursor: idx === 0 ? 'default' : 'pointer', color: idx === 0 ? C.textMuted : C.textSec, fontSize: '12px', padding: '2px' }}>
                ↑
              </button>
              <button onClick={() => moveDown(idx)} disabled={idx === value.length - 1}
                style={{ background: 'none', border: 'none', cursor: idx === value.length - 1 ? 'default' : 'pointer', color: idx === value.length - 1 ? C.textMuted : C.textSec, fontSize: '12px', padding: '2px' }}>
                ↓
              </button>
              <button onClick={() => toggleModel(m)}
                style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.danger, fontSize: '12px', padding: '2px', fontWeight: 700 }}>
                ✕
              </button>
            </div>
          ))}
        </div>
      )}

      {/* 可选模型标签 */}
      {candidates.length > 0 ? (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
          {candidates.map(m => {
            const selected = value.includes(m)
            return (
              <button key={m} onClick={() => toggleModel(m)} style={{
                padding: '4px 10px', borderRadius: '6px', fontSize: '11px',
                fontFamily: 'monospace', cursor: 'pointer',
                border: `1px solid ${selected ? C.primary : C.border}`,
                background: selected ? C.primaryLight : C.white,
                color: selected ? C.primary : C.textSec,
                fontWeight: selected ? 600 : 400,
              }}>
                {selected ? '✓ ' : '+ '}{m}
              </button>
            )
          })}
        </div>
      ) : (
        <div style={{ fontSize: '12px', color: C.textMuted }}>
          {availableModels.length === 0
            ? '请先在"连接配置"中查询可用模型'
            : '无其他可选模型（所有模型均为主模型）'}
        </div>
      )}

      {value.length === 0 && candidates.length > 0 && (
        <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px' }}>
          点击模型标签添加为降级备选，拖拽序号调整优先级
        </div>
      )}
    </div>
  )
}
