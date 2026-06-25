/**
 * VideoEditorExportDialog.tsx — 一键导出三选项弹窗（S-V2新增）
 *
 * 替换原 VideoEditorModal.handleExport 中的临时 window.confirm。
 * 三选项：
 *   ① 仅视频          — 拼接片段+转场+原声，不带字幕与配音
 *   ② 视频+字幕烧录    — FFmpeg硬字幕永久烧进画面
 *   ③ 视频+字幕+配音   — 烧录字幕后再把TTS旁白按时间轴混入成片
 *
 * 串行执行链由父级 MediaManagerPanel.onExport 完成：
 *   advancedConcat → burnIn(可选) → mixNarration(可选)
 *   每步失败自动降级保留上一步产物（设计文档5.2约定）。
 *
 * 本弹窗只在"存在非空字幕"时由 VideoEditorModal 打开；无字幕时
 * 保持旧行为直接导出纯视频，不打扰老师。
 */
import { useState } from 'react'

/** 导出模式：仅视频 / +字幕烧录 / +字幕+配音 */
export type ExportMode = 'video' | 'subtitle' | 'narration'

/** 单个导出选项的展示配置 */
interface ExportOptionDef {
  key: ExportMode
  emoji: string
  title: string
  desc: string
  enabled: boolean
  disabledHint?: string
}

interface Props {
  /** 当前字幕轨中非空文本的字幕条数 */
  subtitleCount: number
  /** 当前字幕轨中已生成TTS配音的字幕条数 */
  narratedCount: number
  /** 确认导出（按所选模式） */
  onConfirm: (mode: ExportMode) => void
  /** 取消（返回编辑器） */
  onCancel: () => void
}

export default function VideoEditorExportDialog({ subtitleCount, narratedCount, onConfirm, onCancel }: Props) {
  // 默认选中"仅视频"——最保守、最快的选项
  const [mode, setMode] = useState<ExportMode>('video')

  // 三个选项的可用性与文案（不可用项灰显并说明原因，与迭代2.6"禁用态解释"同规范）
  const options: ExportOptionDef[] = [
    {
      key: 'video',
      emoji: '🎬',
      title: '仅导出视频',
      desc: '拼接全部片段与转场，保留原声。速度最快（不重编码画面）。',
      enabled: true,
    },
    {
      key: 'subtitle',
      emoji: '💬',
      title: '视频 + 字幕烧录',
      desc: '把 ' + subtitleCount + ' 条字幕永久烧录进画面（硬字幕，任何播放器可见）。需重编码视频，约1-2分钟。',
      enabled: subtitleCount > 0,
      disabledHint: '当前没有可烧录的字幕',
    },
    {
      key: 'narration',
      emoji: '🎙',
      title: '视频 + 字幕 + 配音',
      desc: '烧录 ' + subtitleCount + ' 条字幕，并把 ' + narratedCount + ' 条TTS配音旁白按时间轴混入成片。',
      enabled: subtitleCount > 0 && narratedCount > 0,
      disabledHint: narratedCount === 0 ? '尚未生成TTS配音（请先在字幕轨点 🎙 配音）' : '当前没有可烧录的字幕',
    },
  ]

  const current = options.find(o => o.key === mode)
  const canConfirm = !!current && current.enabled

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 99998, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <div style={{ width: 520, maxWidth: '92vw', background: '#fff', borderRadius: 16, boxShadow: '0 12px 48px rgba(0,0,0,0.35)', overflow: 'hidden' }}>
        {/* 弹窗头部 */}
        <div style={{ padding: '18px 24px', borderBottom: '1px solid #E5E7EB' }}>
          <div style={{ fontSize: 16, fontWeight: 700, color: '#1F2937' }}>📦 导出成片</div>
          <div style={{ fontSize: 12, color: '#6B7280', marginTop: 3 }}>选择导出内容，处理按"拼接 → 烧录字幕 → 混入配音"顺序串行进行</div>
        </div>

        {/* 三选项列表 */}
        <div style={{ padding: '16px 20px', display: 'flex', flexDirection: 'column', gap: 10 }}>
          {options.map(opt => {
            const selected = mode === opt.key
            return (
              <div
                key={opt.key}
                onClick={() => { if (opt.enabled) setMode(opt.key) }}
                style={{
                  display: 'flex', alignItems: 'flex-start', gap: 12, padding: '12px 14px', borderRadius: 12,
                  border: '2px solid ' + (selected && opt.enabled ? '#7C3AED' : '#E5E7EB'),
                  background: opt.enabled ? (selected ? 'rgba(124,58,237,0.06)' : '#fff') : '#F9FAFB',
                  cursor: opt.enabled ? 'pointer' : 'not-allowed',
                  opacity: opt.enabled ? 1 : 0.6,
                  transition: 'all 150ms ease',
                }}
                title={opt.enabled ? '' : opt.disabledHint}
              >
                {/* 单选指示圆点 */}
                <div style={{
                  width: 18, height: 18, borderRadius: '50%', flexShrink: 0, marginTop: 2, boxSizing: 'border-box',
                  border: '2px solid ' + (selected && opt.enabled ? '#7C3AED' : '#D1D5DB'),
                  background: selected && opt.enabled ? '#7C3AED' : '#fff',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                }}>
                  {selected && opt.enabled && <div style={{ width: 6, height: 6, borderRadius: '50%', background: '#fff' }} />}
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 14, fontWeight: 600, color: opt.enabled ? '#1F2937' : '#9CA3AF' }}>
                    {opt.emoji} {opt.title}
                  </div>
                  <div style={{ fontSize: 12, color: opt.enabled ? '#6B7280' : '#9CA3AF', marginTop: 3, lineHeight: 1.5 }}>
                    {opt.enabled ? opt.desc : (opt.disabledHint || opt.desc)}
                  </div>
                </div>
              </div>
            )
          })}
        </div>

        {/* 降级说明 */}
        <div style={{ margin: '0 20px 14px', padding: '8px 12px', borderRadius: 8, background: '#FFFBEB', border: '1px solid rgba(245,158,11,0.3)', fontSize: 11, color: '#92400E', lineHeight: 1.6 }}>
          💡 多步处理时任一步失败会自动保留上一步的可用成片（如字幕烧录失败仍会得到无字幕成片），不会整体失败。
        </div>

        {/* 操作按钮 */}
        <div style={{ padding: '14px 20px', borderTop: '1px solid #E5E7EB', display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button onClick={onCancel} style={{
            padding: '9px 20px', borderRadius: 9, border: '1px solid #E5E7EB', background: '#fff',
            color: '#6B7280', fontSize: 13, fontWeight: 500, cursor: 'pointer',
          }}>返回编辑</button>
          <button onClick={() => { if (canConfirm) onConfirm(mode) }} disabled={!canConfirm} style={{
            padding: '9px 24px', borderRadius: 9, border: 'none',
            background: canConfirm ? 'linear-gradient(135deg,#7C3AED,#6D28D9)' : '#E5E7EB',
            color: canConfirm ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600,
            cursor: canConfirm ? 'pointer' : 'not-allowed',
          }}>🚀 开始导出</button>
        </div>
      </div>
    </div>
  )
}
