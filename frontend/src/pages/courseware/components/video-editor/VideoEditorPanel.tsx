/**
 * VideoEditorPanel.tsx — 视频编辑器右侧片段属性面板
 *
 * 从 VideoEditorModal.tsx v6.1 拆分而来(v0.42.7)
 *
 * v0.42.7 新增:
 *   - 音频独立混音控件(静音切换 🔇 + 音量滑块 🎚️) — 仅当 clip.audioUrl 存在时显示
 *
 * 职责:
 *   - 显示当前激活片段的属性(名称/时长/裁剪/转场)
 *   - 滑块调整 trimStart/trimEnd
 *   - 音轨分离(无音轨时)/ 音轨展示(已分离时)/ 删除音轨 / 恢复原始
 *   - 音频音量与静音控制(v0.42.7)
 *   - 转场类型选择 + 时长调整
 *   - 预览此片段 / 连贯预览全部
 */
import type { EditorClip } from './VideoEditorTypes'
import { C, TRANS, TRANS_GROUPS } from './VideoEditorConstants'
import { fmt, fmtSize } from './VideoEditorUtils'

interface VideoEditorPanelProps {
  /** 当前激活片段(null 时显示占位) */
  ac: EditorClip | null
  /** 激活片段的索引(-1 表示无激活) */
  activeIdx: number
  /** 总片段数(决定是否显示"连贯预览全部"按钮) */
  clipsLength: number
  /** 是否有 coursewareId(决定是否显示音轨分离区) */
  hasAudio: boolean
  /** 当前正在分离音轨的片段索引(-1 表示无) */
  separatingIdx: number
  /** 更新片段字段 */
  onUpdateClip: (idx: number, u: Partial<EditorClip>) => void
  /** 分离音轨 */
  onSeparateAudio: () => void
  /** 删除已分离的音轨 */
  onDeleteAudio: () => void
  /** 恢复原始视频(含音轨) */
  onRestoreOriginal: () => void
  /** 单段预览 */
  onPlaySingle: (idx: number) => void
  /** 连贯预览全部 */
  onPlayAll: () => void
}

export default function VideoEditorPanel({
  ac, activeIdx, clipsLength, hasAudio, separatingIdx,
  onUpdateClip, onSeparateAudio, onDeleteAudio, onRestoreOriginal,
  onPlaySingle, onPlayAll,
}: VideoEditorPanelProps) {
  // 无激活片段: 显示占位提示
  if (!ac) {
    return (
      <div style={{ width: 280, borderLeft: `1px solid ${C.border}`, padding: '16px 14px', overflowY: 'auto', flexShrink: 0 }}>
        <div style={{ textAlign: 'center', padding: '40px 10px', color: C.textMuted, fontSize: 13 }}>
          <div style={{ fontSize: 32, marginBottom: 8 }}>⚙️</div>
          点击时间轴上的片段<br/>查看和编辑属性
        </div>
      </div>
    )
  }

  // v0.42.7: 音频混音控件值(默认值兜底,旧片段无字段时表现为不静音 / 满音量)
  const audioVolume = ac.audioVolume ?? 1.0
  const audioMuted = ac.audioMuted ?? false

  return (
    <div style={{ width: 280, borderLeft: `1px solid ${C.border}`, padding: '16px 14px', overflowY: 'auto', flexShrink: 0 }}>
      <div style={{ fontSize: 13, fontWeight: 600, color: C.text, marginBottom: 16 }}>⚙️ 片段属性</div>

      {/* === 基础信息 === */}
      <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 4 }}>名称</div>
      <div style={{
        fontSize: 13, color: C.text, padding: '6px 10px', borderRadius: 6,
        background: C.surfaceLight, marginBottom: 14,
        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        display: 'flex', alignItems: 'center', gap: 6,
      }}>
        {ac.muted && <span title="已静音">🔇</span>}{ac.label}
      </div>

      <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 4 }}>原始时长</div>
      <div style={{ fontSize: 13, color: C.accent, fontFamily: 'monospace', marginBottom: 14 }}>
        {fmt(ac.duration)}
      </div>

      {/* === 裁剪 === */}
      <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 4 }}>裁剪入点</div>
      <input
        type="range" min={0} max={ac.duration - 0.5} step={0.1}
        value={ac.trimStart}
        onChange={e => {
          const v = parseFloat(e.target.value)
          if (v < ac.trimEnd - 0.5) onUpdateClip(activeIdx, { trimStart: v })
        }}
        style={{ width: '100%', accentColor: C.primary, marginBottom: 2 }}
      />
      <div style={{ fontSize: 12, color: C.accent, fontFamily: 'monospace', marginBottom: 14, textAlign: 'right' }}>
        {fmt(ac.trimStart)}
      </div>

      <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 4 }}>裁剪出点</div>
      <input
        type="range" min={0.5} max={ac.duration} step={0.1}
        value={ac.trimEnd}
        onChange={e => {
          const v = parseFloat(e.target.value)
          if (v > ac.trimStart + 0.5) onUpdateClip(activeIdx, { trimEnd: v })
        }}
        style={{ width: '100%', accentColor: C.primary, marginBottom: 2 }}
      />
      <div style={{ fontSize: 12, color: C.accent, fontFamily: 'monospace', marginBottom: 14, textAlign: 'right' }}>
        {fmt(ac.trimEnd)}
      </div>

      <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 4 }}>有效时长</div>
      <div style={{ fontSize: 15, fontWeight: 700, color: C.success, fontFamily: 'monospace', marginBottom: 14 }}>
        {fmt(ac.trimEnd - ac.trimStart)}
      </div>

      {/* === 音轨分离区(v0.42.7 含独立混音控件) === */}
      {hasAudio && (<>
        <div style={{ height: 1, background: C.border, margin: '4px 0 12px' }} />
        <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 10 }}>🎧 音轨</div>

        {!ac.muted ? (
          // 未分离: 显示分离按钮
          <button
            onClick={onSeparateAudio}
            disabled={separatingIdx >= 0}
            style={{
              width: '100%', padding: '10px', borderRadius: 8,
              border: '1px solid #6366F1', background: 'transparent',
              color: '#818CF8', fontSize: 13, fontWeight: 600,
              cursor: separatingIdx >= 0 ? 'default' : 'pointer',
              opacity: separatingIdx >= 0 ? 0.5 : 1,
            }}
          >
            {separatingIdx === activeIdx ? '⏳ 分离中...' : '🔗 分离音轨'}
          </button>
        ) : (
          // 已分离: 显示音轨控制
          <div style={{ borderRadius: 10, overflow: 'hidden', border: `1px solid ${C.border}` }}>
            {/* 视频已静音标记 */}
            <div style={{ padding: '8px 10px', background: C.surfaceLight, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span style={{ fontSize: 12, color: C.text }}>🎬 视频（已静音）</span>
              <span style={{ fontSize: 10, color: C.success }}>🔇</span>
            </div>

            {ac.audioUrl ? (
              <div style={{ padding: '8px 10px', background: '#1E1B4B', borderTop: `1px solid ${C.border}` }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
                  <span style={{ fontSize: 12, color: '#A78BFA' }}>🎵 音轨</span>
                  <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
                    {ac.audioDuration && <span style={{ fontSize: 10, color: C.textMuted }}>{ac.audioDuration}</span>}
                    {ac.audioFileSize && ac.audioFileSize > 0 && <span style={{ fontSize: 10, color: C.textMuted }}>{fmtSize(ac.audioFileSize)}</span>}
                  </div>
                </div>
                <audio controls src={ac.audioUrl} preload="metadata" style={{ width: '100%', height: 28, marginBottom: 6 }} />

                {/* === v0.42.7 新增: 独立混音控件 === */}
                <div style={{ padding: '8px 0 4px', borderTop: `1px solid ${C.border}`, marginTop: 4 }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
                    <span style={{ fontSize: 11, color: '#A78BFA', fontWeight: 600 }}>
                      🎚️ 播放混音
                    </span>
                    <button
                      onClick={() => onUpdateClip(activeIdx, { audioMuted: !audioMuted })}
                      title={audioMuted ? '点击取消静音' : '点击静音(播放时此音轨不发声)'}
                      style={{
                        padding: '2px 8px', borderRadius: 4,
                        border: `1px solid ${audioMuted ? C.danger : C.success}`,
                        background: audioMuted ? `${C.danger}20` : `${C.success}20`,
                        color: audioMuted ? C.danger : C.success,
                        fontSize: 10, fontWeight: 600, cursor: 'pointer',
                      }}
                    >
                      {audioMuted ? '🔇 已静音' : '🔊 发声中'}
                    </button>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ fontSize: 10, color: C.textMuted, minWidth: 24 }}>音量</span>
                    <input
                      type="range" min={0} max={1.0} step={0.05}
                      value={audioVolume}
                      disabled={audioMuted}
                      onChange={e => onUpdateClip(activeIdx, { audioVolume: parseFloat(e.target.value) })}
                      style={{
                        flex: 1, accentColor: audioMuted ? C.textMuted : '#A78BFA',
                        opacity: audioMuted ? 0.4 : 1,
                      }}
                    />
                    <span style={{ fontSize: 10, color: '#A78BFA', fontFamily: 'monospace', minWidth: 30, textAlign: 'right' }}>
                      {Math.round(audioVolume * 100)}%
                    </span>
                  </div>
                  <div style={{ marginTop: 4, fontSize: 9, color: C.textMuted, lineHeight: 1.5 }}>
                    💡 调整此片段播放时的音频音量,不影响导出的素材文件
                  </div>
                </div>

                {/* 下载/删除音轨按钮 */}
                <div style={{ display: 'flex', gap: 6, marginTop: 6 }}>
                  <a href={ac.audioUrl} download style={{
                    flex: 1, padding: '5px 0', borderRadius: 6,
                    border: `1px solid ${C.playing}`, background: 'transparent',
                    color: C.playing, fontSize: 11, fontWeight: 600,
                    textDecoration: 'none', textAlign: 'center', display: 'block',
                  }}>⬇ 下载</a>
                  <button onClick={onDeleteAudio} style={{
                    flex: 1, padding: '5px 0', borderRadius: 6,
                    border: `1px solid ${C.danger}`, background: 'transparent',
                    color: C.danger, fontSize: 11, fontWeight: 600, cursor: 'pointer',
                  }}>🗑 删除音轨</button>
                </div>
              </div>
            ) : (
              <div style={{ padding: '8px 10px', background: '#1E1B4B', borderTop: `1px solid ${C.border}` }}>
                <span style={{ fontSize: 11, color: C.textMuted }}>音轨已删除</span>
              </div>
            )}

            {/* 恢复原始视频 */}
            {ac.originalUrl && (
              <div style={{ padding: '6px 10px', borderTop: `1px solid ${C.border}`, background: C.surface }}>
                <button onClick={onRestoreOriginal} style={{
                  width: '100%', padding: '6px 0', borderRadius: 6,
                  border: `1px solid ${C.accent}`, background: 'transparent',
                  color: C.accent, fontSize: 11, fontWeight: 600, cursor: 'pointer',
                }}>↩ 恢复原始视频（含音轨）</button>
              </div>
            )}
          </div>
        )}
      </>)}

      {/* === 转场(非最后一段才显示) === */}
      {activeIdx < clipsLength - 1 && (<>
        <div style={{ height: 1, background: C.border, margin: '12px 0' }} />
        <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 10 }}>🎞️ 与下一段的转场效果</div>

        {TRANS_GROUPS.map(g => {
          const items = TRANS.filter(t => t.group === g.key)
          if (!items.length) return null
          return (
            <div key={g.key} style={{ marginBottom: 8 }}>
              <div style={{ fontSize: 10, color: C.textMuted, marginBottom: 4, textTransform: 'uppercase', letterSpacing: 1 }}>
                {g.label}
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                {items.map(t => {
                  const sel = ac.transition === t.key
                  return (
                    <button
                      key={t.key}
                      onClick={() => onUpdateClip(activeIdx, { transition: t.key })}
                      style={{
                        padding: '4px 8px', borderRadius: 6, fontSize: 10, cursor: 'pointer',
                        border: `1px solid ${sel ? t.color : C.border}`,
                        background: sel ? t.color + '20' : 'transparent',
                        color: sel ? t.color : C.textMuted,
                        display: 'flex', alignItems: 'center', gap: 3,
                      }}
                    >
                      <span>{t.emoji}</span> <span>{t.label}</span>
                    </button>
                  )
                })}
              </div>
            </div>
          )
        })}

        {ac.transition !== 'none' && (<>
          <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 4, marginTop: 8 }}>转场时长</div>
          <input
            type="range" min={0.2} max={2.0} step={0.1}
            value={ac.transDur}
            onChange={e => onUpdateClip(activeIdx, { transDur: parseFloat(e.target.value) })}
            style={{ width: '100%', accentColor: C.accent, marginBottom: 2 }}
          />
          <div style={{ fontSize: 12, color: C.accent, fontFamily: 'monospace', textAlign: 'right' }}>
            {ac.transDur.toFixed(1)}s
          </div>
        </>)}
      </>)}

      {/* === 预览按钮 === */}
      <div style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 8 }}>
        <button onClick={() => onPlaySingle(activeIdx)} style={{
          width: '100%', padding: '10px', borderRadius: 8, border: 'none',
          background: C.primary, color: '#fff',
          fontSize: 13, fontWeight: 600, cursor: 'pointer',
        }}>▶ 预览此片段</button>

        {clipsLength >= 2 && (
          <button onClick={onPlayAll} style={{
            width: '100%', padding: '10px', borderRadius: 8,
            border: `1px solid ${C.playing}`, background: 'transparent',
            color: C.playing, fontSize: 13, fontWeight: 600, cursor: 'pointer',
          }}>▶ 连贯预览全部</button>
        )}
      </div>
    </div>
  )
}
