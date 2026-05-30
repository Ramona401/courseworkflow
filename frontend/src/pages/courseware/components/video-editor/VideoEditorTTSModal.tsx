/**
 * VideoEditorTTSModal.tsx — TTS 批量配音弹窗
 *
 * v0.42.9 新增
 * v0.42.9.1 优化：新增实时计时器 + 预估进度条 + 逐条动画提示
 *
 * 功能：
 *   - 选择音色（按语言筛选）
 *   - 调整语速（0.75x ~ 1.5x）
 *   - 选择处理范围（全部 / 仅未配音 / 手动选择）
 *   - 批量生成 TTS 配音（含实时进度反馈）
 *   - 显示生成结果
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { C } from './VideoEditorConstants'
import { listTTSVoices, generateSubtitleTTS } from '../../../../api/coursewares'
import type { TTSVoice } from '../../../../api/coursewares'
import type { SubtitleSegment } from './VideoEditorSubtitleTrack'

interface TTSModalProps {
  /** 课件 ID */
  coursewareId: string
  /** 字幕轨 ID（数据库中的 courseware_subtitles.id） */
  subtitleId: string
  /** 当前字幕片段列表 */
  segments: SubtitleSegment[]
  /** 当前语言 */
  language: string
  /** TTS 完成后回调（传回更新后的 segments） */
  onComplete: (updatedSegments: SubtitleSegment[]) => void
  /** 关闭弹窗 */
  onClose: () => void
}

/** 语速预设 */
const SPEED_OPTIONS = [
  { value: 0.75, label: '0.75x 慢速' },
  { value: 1.0, label: '1.0x 正常' },
  { value: 1.25, label: '1.25x 稍快' },
  { value: 1.5, label: '1.5x 快速' },
]

/** 处理范围选项 */
type ScopeOption = 'all' | 'no_tts' | 'selected'

/** 每条字幕预估 TTS 处理时间（秒） */
const EST_SEC_PER_SEGMENT = 3

export default function VideoEditorTTSModal({
  coursewareId, subtitleId, segments, language, onComplete, onClose,
}: TTSModalProps) {
  // 音色列表
  const [voices, setVoices] = useState<TTSVoice[]>([])
  const [loadingVoices, setLoadingVoices] = useState(true)
  const [selectedVoice, setSelectedVoice] = useState('')

  // 配置
  const [speed, setSpeed] = useState(1.0)
  const [scope, setScope] = useState<ScopeOption>('all')
  const [selectedSegIds, setSelectedSegIds] = useState<Set<string>>(new Set())

  // 生成状态
  const [generating, setGenerating] = useState(false)
  const [elapsedSec, setElapsedSec] = useState(0)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const [result, setResult] = useState<{ success: number; fail: number; errors: string[] } | null>(null)

  // 加载音色列表
  useEffect(() => {
    setLoadingVoices(true)
    listTTSVoices(language).then(res => {
      setVoices(res.voices || [])
      if (res.voices?.length > 0) setSelectedVoice(res.voices[0].code)
    }).catch(err => {
      console.error('加载音色列表失败:', err)
    }).finally(() => setLoadingVoices(false))
  }, [language])

  // 生成期间计时器
  useEffect(() => {
    if (generating) {
      setElapsedSec(0)
      timerRef.current = setInterval(() => setElapsedSec(prev => prev + 1), 1000)
    } else {
      if (timerRef.current) { clearInterval(timerRef.current); timerRef.current = null }
    }
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [generating])

  // 计算待处理的字幕条
  const getTargetSegments = useCallback((): SubtitleSegment[] => {
    const nonEmpty = segments.filter(s => s.text.trim().length > 0)
    if (scope === 'all') return nonEmpty
    if (scope === 'no_tts') return nonEmpty.filter(s => !s.tts_audio_url)
    if (scope === 'selected') return nonEmpty.filter(s => selectedSegIds.has(s.id))
    return nonEmpty
  }, [segments, scope, selectedSegIds])

  const targetCount = getTargetSegments().length
  const noTTSCount = segments.filter(s => s.text.trim() && !s.tts_audio_url).length
  const hasTTSCount = segments.filter(s => !!s.tts_audio_url).length

  // 预估总时间
  const estimatedTotalSec = targetCount * EST_SEC_PER_SEGMENT
  // 进度百分比（基于时间预估）
  const progressPct = generating && estimatedTotalSec > 0
    ? Math.min(95, Math.round((elapsedSec / estimatedTotalSec) * 100))
    : 0
  // 预估剩余秒数
  const remainingSec = Math.max(0, estimatedTotalSec - elapsedSec)

  // 格式化时间 MM:SS
  const fmtTime = (sec: number): string => {
    const m = Math.floor(sec / 60)
    const s = sec % 60
    return `${m}:${String(s).padStart(2, '0')}`
  }

  // 切换选中
  const toggleSeg = (id: string) => {
    setSelectedSegIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }

  // 全选/取消全选
  const toggleAll = () => {
    if (selectedSegIds.size === segments.length) {
      setSelectedSegIds(new Set())
    } else {
      setSelectedSegIds(new Set(segments.map(s => s.id)))
    }
  }

  // 批量生成 TTS
  const handleGenerate = async () => {
    if (!selectedVoice || targetCount === 0 || generating) return
    setGenerating(true)
    setResult(null)

    try {
      const segIds = scope === 'all' ? undefined : getTargetSegments().map(s => s.id)
      const resp = await generateSubtitleTTS(coursewareId, subtitleId, selectedVoice, speed, segIds)

      setResult({
        success: resp.success_count,
        fail: resp.fail_count,
        errors: resp.errors || [],
      })

      // 解析更新后的 segments 并回调
      if (resp.segments) {
        try {
          const updated = JSON.parse(resp.segments)
          if (Array.isArray(updated)) onComplete(updated)
        } catch { /* 解析失败不影响 */ }
      }
    } catch (err) {
      setResult({
        success: 0,
        fail: targetCount,
        errors: [err instanceof Error ? err.message : '生成失败'],
      })
    } finally {
      setGenerating(false)
    }
  }

  // 进度动画文案（循环切换）
  const progressTexts = [
    '🎤 正在录制配音...',
    '🔊 AI 正在朗读文本...',
    '✨ 合成高品质音频...',
    '📦 保存音频文件...',
    '🎵 处理中，请耐心等待...',
  ]
  const currentProgressText = generating
    ? progressTexts[Math.floor(elapsedSec / 3) % progressTexts.length]
    : ''

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 10001,
      background: 'rgba(0,0,0,0.7)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }} onClick={generating ? undefined : onClose}>
      <div
        onClick={e => e.stopPropagation()}
        style={{
          width: 540, maxHeight: '85vh', background: C.surface, borderRadius: 12,
          border: `1px solid ${C.border}`, padding: 24,
          boxShadow: '0 20px 40px rgba(0,0,0,0.5)',
          display: 'flex', flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        {/* 标题 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, flexShrink: 0 }}>
          <span style={{ fontSize: 16, fontWeight: 700, color: C.text }}>🎙️ TTS 批量配音</span>
          {!generating && (
            <button onClick={onClose} style={{
              background: 'none', border: 'none', color: C.textMuted,
              fontSize: 18, cursor: 'pointer', padding: '0 4px',
            }}>✕</button>
          )}
        </div>

        {/* ==================== 生成进度区域 ==================== */}
        {generating && (
          <div style={{
            padding: 16, borderRadius: 8, marginBottom: 16, flexShrink: 0,
            background: 'linear-gradient(135deg, rgba(124,58,237,0.1) 0%, rgba(245,158,11,0.1) 100%)',
            border: '1px solid rgba(124,58,237,0.3)',
          }}>
            {/* 进度条 */}
            <div style={{
              height: 6, borderRadius: 3, background: 'rgba(255,255,255,0.1)',
              marginBottom: 12, overflow: 'hidden',
            }}>
              <div style={{
                height: '100%', borderRadius: 3,
                background: 'linear-gradient(90deg, #7C3AED, #F59E0B)',
                width: `${progressPct}%`,
                transition: 'width 1s linear',
              }} />
            </div>

            {/* 进度文案 + 时间 */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span style={{ fontSize: 13, color: '#C4B5FD', fontWeight: 600 }}>
                {currentProgressText}
              </span>
              <span style={{ fontSize: 12, color: C.textMuted, fontFamily: 'monospace' }}>
                {fmtTime(elapsedSec)} / ~{fmtTime(estimatedTotalSec)}
              </span>
            </div>

            {/* 详细信息 */}
            <div style={{ marginTop: 8, fontSize: 11, color: C.textMuted }}>
              正在处理 {targetCount} 条字幕 · 预计剩余 ~{remainingSec > 60 ? `${Math.ceil(remainingSec / 60)} 分钟` : `${remainingSec} 秒`}
              {elapsedSec > estimatedTotalSec && (
                <span style={{ color: '#F59E0B', marginLeft: 8 }}>（处理时间超出预估，仍在继续...）</span>
              )}
            </div>
          </div>
        )}

        {/* 统计信息 */}
        {!generating && (
          <div style={{ display: 'flex', gap: 12, marginBottom: 16, flexShrink: 0 }}>
            <div style={{ padding: '6px 12px', background: C.bg, borderRadius: 6, fontSize: 12, color: C.textMuted }}>
              📝 共 <span style={{ color: C.accent, fontWeight: 700 }}>{segments.filter(s => s.text.trim()).length}</span> 条有文本字幕
            </div>
            <div style={{ padding: '6px 12px', background: C.bg, borderRadius: 6, fontSize: 12, color: C.textMuted }}>
              🎵 已配音 <span style={{ color: C.success, fontWeight: 700 }}>{hasTTSCount}</span> 条
            </div>
            <div style={{ padding: '6px 12px', background: C.bg, borderRadius: 6, fontSize: 12, color: C.textMuted }}>
              ⏳ 待配音 <span style={{ color: '#F59E0B', fontWeight: 700 }}>{noTTSCount}</span> 条
            </div>
          </div>
        )}

        <div style={{ flex: 1, overflowY: 'auto', minHeight: 0, opacity: generating ? 0.4 : 1, pointerEvents: generating ? 'none' : 'auto' }}>
          {/* 音色选择 */}
          <div style={{ marginBottom: 16 }}>
            <label style={{ fontSize: 12, color: C.textMuted, marginBottom: 6, display: 'block', fontWeight: 600 }}>选择音色</label>
            {loadingVoices ? (
              <div style={{ padding: 12, color: C.textMuted, fontSize: 13 }}>加载音色列表中...</div>
            ) : (
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 6 }}>
                {voices.map(v => (
                  <button
                    key={v.code}
                    onClick={() => setSelectedVoice(v.code)}
                    style={{
                      padding: '8px 10px', borderRadius: 6, cursor: 'pointer',
                      textAlign: 'left', fontSize: 12,
                      background: selectedVoice === v.code ? C.accent + '20' : C.bg,
                      border: `1px solid ${selectedVoice === v.code ? C.accent : C.border}`,
                      color: selectedVoice === v.code ? C.accent : C.text,
                    }}
                  >
                    <div style={{ fontWeight: 600, marginBottom: 2 }}>
                      {v.gender === 'female' ? '👩' : '👨'} {v.name}
                    </div>
                    <div style={{ fontSize: 10, color: C.textMuted }}>{v.style}</div>
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* 语速选择 */}
          <div style={{ marginBottom: 16 }}>
            <label style={{ fontSize: 12, color: C.textMuted, marginBottom: 6, display: 'block', fontWeight: 600 }}>语速</label>
            <div style={{ display: 'flex', gap: 6 }}>
              {SPEED_OPTIONS.map(opt => (
                <button
                  key={opt.value}
                  onClick={() => setSpeed(opt.value)}
                  style={{
                    flex: 1, padding: '6px 8px', borderRadius: 6, cursor: 'pointer',
                    fontSize: 12, fontWeight: speed === opt.value ? 700 : 400,
                    background: speed === opt.value ? C.accent + '20' : C.bg,
                    border: `1px solid ${speed === opt.value ? C.accent : C.border}`,
                    color: speed === opt.value ? C.accent : C.textMuted,
                  }}
                >{opt.label}</button>
              ))}
            </div>
          </div>

          {/* 处理范围 */}
          <div style={{ marginBottom: 16 }}>
            <label style={{ fontSize: 12, color: C.textMuted, marginBottom: 6, display: 'block', fontWeight: 600 }}>处理范围</label>
            <div style={{ display: 'flex', gap: 6 }}>
              {([
                { key: 'all' as ScopeOption, label: `全部 (${segments.filter(s => s.text.trim()).length})`, emoji: '📋' },
                { key: 'no_tts' as ScopeOption, label: `仅未配音 (${noTTSCount})`, emoji: '⏳' },
                { key: 'selected' as ScopeOption, label: '手动选择', emoji: '✅' },
              ]).map(opt => (
                <button
                  key={opt.key}
                  onClick={() => setScope(opt.key)}
                  style={{
                    flex: 1, padding: '6px 8px', borderRadius: 6, cursor: 'pointer',
                    fontSize: 11, fontWeight: scope === opt.key ? 700 : 400,
                    background: scope === opt.key ? C.primary + '20' : C.bg,
                    border: `1px solid ${scope === opt.key ? C.primary : C.border}`,
                    color: scope === opt.key ? C.primary : C.textMuted,
                  }}
                >{opt.emoji} {opt.label}</button>
              ))}
            </div>
          </div>

          {/* 手动选择列表 */}
          {scope === 'selected' && (
            <div style={{ marginBottom: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                <span style={{ fontSize: 11, color: C.textMuted }}>已选 {selectedSegIds.size} 条</span>
                <button onClick={toggleAll} style={{
                  background: 'none', border: 'none', color: C.accent, fontSize: 11,
                  cursor: 'pointer', textDecoration: 'underline',
                }}>{selectedSegIds.size === segments.length ? '取消全选' : '全选'}</button>
              </div>
              <div style={{ maxHeight: 150, overflowY: 'auto', borderRadius: 6, border: `1px solid ${C.border}` }}>
                {segments.filter(s => s.text.trim()).map((seg, i) => (
                  <label key={seg.id} style={{
                    display: 'flex', alignItems: 'center', gap: 8,
                    padding: '6px 10px', cursor: 'pointer',
                    borderBottom: `1px solid ${C.border}`,
                    background: selectedSegIds.has(seg.id) ? C.primary + '10' : 'transparent',
                    fontSize: 12, color: C.text,
                  }}>
                    <input
                      type="checkbox"
                      checked={selectedSegIds.has(seg.id)}
                      onChange={() => toggleSeg(seg.id)}
                      style={{ accentColor: C.primary }}
                    />
                    <span style={{ color: C.textMuted, fontFamily: 'monospace', fontSize: 10, width: 24, flexShrink: 0 }}>#{i + 1}</span>
                    <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{seg.text}</span>
                    {seg.tts_audio_url && <span style={{ fontSize: 10, color: C.success }}>🎵</span>}
                  </label>
                ))}
              </div>
            </div>
          )}

          {/* 生成结果 */}
          {result && (
            <div style={{
              padding: 12, borderRadius: 8, marginBottom: 16,
              background: result.fail === 0 ? 'rgba(16,185,129,0.1)' : 'rgba(239,68,68,0.1)',
              border: `1px solid ${result.fail === 0 ? 'rgba(16,185,129,0.3)' : 'rgba(239,68,68,0.3)'}`,
            }}>
              <div style={{ fontSize: 13, fontWeight: 600, color: result.fail === 0 ? C.success : C.danger, marginBottom: 4 }}>
                {result.fail === 0 ? '✅ 配音完成' : '⚠️ 部分失败'}
              </div>
              <div style={{ fontSize: 12, color: C.textMuted }}>
                成功 {result.success} 条{result.fail > 0 && `，失败 ${result.fail} 条`}
                {elapsedSec > 0 && ` · 耗时 ${fmtTime(elapsedSec)}`}
              </div>
              {result.errors.length > 0 && (
                <div style={{ marginTop: 6, fontSize: 11, color: C.danger, maxHeight: 60, overflowY: 'auto' }}>
                  {result.errors.map((e, i) => <div key={i}>• {e}</div>)}
                </div>
              )}
            </div>
          )}
        </div>

        {/* 底部按钮 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 16, flexShrink: 0 }}>
          <span style={{ fontSize: 11, color: C.textMuted }}>
            {generating
              ? `处理中... ${progressPct}%`
              : `将为 ${targetCount} 条字幕生成配音` + (targetCount > 0 ? ` · 预计 ~${estimatedTotalSec > 60 ? Math.ceil(estimatedTotalSec / 60) + '分钟' : estimatedTotalSec + '秒'}` : '')
            }
          </span>
          <div style={{ display: 'flex', gap: 8 }}>
            {!generating && (
              <button onClick={onClose} style={{
                padding: '8px 16px', fontSize: 13, borderRadius: 6,
                background: C.surfaceLight, color: C.textMuted,
                border: `1px solid ${C.border}`, cursor: 'pointer',
              }}>{result ? '关闭' : '取消'}</button>
            )}
            {!result && (
              <button
                onClick={handleGenerate}
                disabled={generating || targetCount === 0 || !selectedVoice}
                style={{
                  padding: '8px 20px', fontSize: 13, borderRadius: 6,
                  background: generating || targetCount === 0 ? C.surfaceLight : C.accent,
                  color: generating || targetCount === 0 ? C.textMuted : '#000',
                  border: 'none', cursor: generating || targetCount === 0 ? 'not-allowed' : 'pointer',
                  fontWeight: 700,
                }}
              >
                {generating ? `⏳ 生成中 ${fmtTime(elapsedSec)}...` : `🎙️ 生成配音 (${targetCount}条)`}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
