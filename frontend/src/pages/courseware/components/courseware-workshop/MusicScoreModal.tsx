/**
 * MusicScoreModal.tsx — 课件五线谱/乐谱编辑器弹窗
 *
 * 复用笔顺编辑器(StrokeOrderModal)的交互范式：
 * 老师输入 ABC 记谱法 / 选模板 → 实时预览五线谱 SVG → 调参(宽度/播放/调号/拍号)
 * → 点"融入当前页" → 拼微调指令经 onInsert 回调交给父组件调 RefinePage。
 *
 * 技术方案：
 *   - 渲染预览用 abcjs（CDN 动态加载，renderAbc 输出 SVG 到预览区）
 *   - 可选 MIDI 播放（abcjs 内置 synth，课件内也可播放）
 *   - 声音音色（SoundFont）已自托管到本服务器（见 musicScoreUtils.SOUNDFONT_URL），
 *     试听按钮显式传 soundFontUrl，否则 abcjs 走默认 github.io 音源国内无声
 *   - ABC 记谱法入门门槛极低（文本格式，老师几分钟即可上手）
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/MusicScoreModal.tsx
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import {
  MUSIC_DEFAULTS,
  KEY_OPTIONS,
  METER_OPTIONS,
  MUSIC_TEMPLATES,
  ABCJS_CDN,
  SOUNDFONT_URL,
  buildMusicRefineInstruction,
  type MusicScoreConfig,
} from './musicScoreUtils'
import { C } from './workshopConstants'

// ==================== 类型 ====================

interface Props {
  pageNum: number
  onInsert: (instruction: string) => void
  onClose: () => void
  inserting?: boolean
}

// ==================== abcjs 全局类型 ====================

declare const ABCJS: {
  renderAbc: (target: string | HTMLElement, abc: string, opts?: Record<string, unknown>) => unknown[]
  synth?: {
    supportsAudio: () => boolean
    CreateSynth: new () => {
      // init 支持 options.soundFontUrl 指定音源地址（自托管，解决国内默认音源不可达无声问题）
      init: (opts: { visualObj: unknown; options?: { soundFontUrl?: string } }) => Promise<void>
      prime: () => Promise<void>
      start: () => void
      stop: () => void
      addEventListener: (event: string, cb: () => void) => void
    }
  }
}

// ==================== 组件 ====================

export default function MusicScoreModal({ pageNum, onInsert, onClose, inserting }: Props) {
  // ---- ABC 文本状态 ----
  const [abc, setAbc] = useState('X:1\nT:我的乐谱\nM:4/4\nL:1/4\nK:C\nC D E F | G A B c |]')
  const [width, setWidth] = useState(MUSIC_DEFAULTS.width)
  const [showPlayer, setShowPlayer] = useState(MUSIC_DEFAULTS.showPlayer)
  const [customTitle, setCustomTitle] = useState('')
  const [positionHint, setPositionHint] = useState('')

  // ---- abcjs 加载与预览状态 ----
  const [abcjsLoaded, setAbcjsLoaded] = useState(false)
  const [previewError, setPreviewError] = useState('')
  const [toast, setToast] = useState('')

  const previewRef = useRef<HTMLDivElement>(null)

  // ---- 快速设置：改 ABC 头部字段 ----
  const updateAbcField = (field: string, value: string) => {
    const regex = new RegExp('^' + field + ':.*$', 'm')
    if (regex.test(abc)) {
      setAbc(abc.replace(regex, field + ':' + value))
    } else {
      // 在第一行音符之前插入
      const lines = abc.split('\n')
      const insertIdx = lines.findIndex(l => /^[A-Ga-gz|[\](){}]/.test(l.trim()))
      if (insertIdx >= 0) {
        lines.splice(insertIdx, 0, field + ':' + value)
        setAbc(lines.join('\n'))
      }
    }
  }

  // ---- 从 ABC 读取当前字段值 ----
  const getAbcField = (field: string): string => {
    const m = abc.match(new RegExp('^' + field + ':(.*)$', 'm'))
    return m ? m[1].trim() : ''
  }

  // ---- 加载 abcjs CDN ----
  useEffect(() => {
    if (typeof (window as unknown as Record<string, unknown>).ABCJS !== 'undefined') {
      setAbcjsLoaded(true)
      return
    }
    const existing = document.querySelector('script[src*="abcjs"]')
    if (existing) {
      const check = setInterval(() => {
        if (typeof (window as unknown as Record<string, unknown>).ABCJS !== 'undefined') {
          setAbcjsLoaded(true)
          clearInterval(check)
        }
      }, 200)
      return () => clearInterval(check)
    }
    const s = document.createElement('script')
    s.src = ABCJS_CDN
    s.onload = () => setAbcjsLoaded(true)
    document.head.appendChild(s)
  }, [])

  // ---- 实时预览 ----
  const updatePreview = useCallback(() => {
    if (!abcjsLoaded || !abc.trim() || !previewRef.current) return
    try {
      previewRef.current.innerHTML = ''
      ABCJS.renderAbc(previewRef.current, abc, {
        responsive: 'resize',
        staffwidth: Math.min(width, 560),
      })
      setPreviewError('')
    } catch (e) {
      setPreviewError(e instanceof Error ? e.message : '渲染失败')
    }
  }, [abcjsLoaded, abc, width])

  useEffect(() => {
    const timer = setTimeout(updatePreview, 400)
    return () => clearTimeout(timer)
  }, [updatePreview])

  // ---- 模板点选 ----
  const handleTemplateClick = (tplAbc: string) => {
    setAbc(tplAbc)
  }

  // ---- 试听预览（仅弹窗内，用 abcjs synth） ----
  const [playing, setPlaying] = useState(false)
  const synthRef = useRef<{ stop: () => void } | null>(null)

  const handlePlayPreview = async () => {
    if (!abcjsLoaded || !ABCJS.synth || !ABCJS.synth.supportsAudio()) {
      showToast('当前浏览器不支持音频播放')
      return
    }
    if (playing) {
      synthRef.current?.stop()
      setPlaying(false)
      return
    }
    try {
      setPlaying(true)
      const vis = ABCJS.renderAbc('*', abc)[0]
      const synth = new ABCJS.synth.CreateSynth()
      // 关键：显式指定自托管音源，否则默认走 github.io 国内加载失败无声
      await synth.init({ visualObj: vis, options: { soundFontUrl: SOUNDFONT_URL } })
      await synth.prime()
      synth.start()
      synthRef.current = synth
      synth.addEventListener('finished', () => setPlaying(false))
    } catch {
      setPlaying(false)
      showToast('播放失败：音源加载异常或乐谱格式有误')
    }
  }

  // 卸载时停止播放
  useEffect(() => { return () => { synthRef.current?.stop() } }, [])

  // ---- 融入当前页 ----
  const handleInsert = () => {
    if (!abc.trim()) { showToast('请先输入乐谱'); return }
    const instruction = buildMusicRefineInstruction({
      scores: [{ abc, width, showPlayer, title: customTitle.trim() || undefined }],
      positionHint: positionHint.trim() || undefined,
    })
    onInsert(instruction)
  }

  const showToast = (msg: string) => {
    setToast(msg)
    setTimeout(() => setToast(''), 2500)
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 99995,
      background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}>
      <div style={{
        width: '92vw', maxWidth: 1040, maxHeight: '92vh', overflow: 'auto',
        background: '#fff', borderRadius: 16, boxShadow: '0 20px 60px rgba(0,0,0,0.3)',
        padding: 28, position: 'relative',
      }}>
        {/* 顶栏 */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
          <div>
            <div style={{ fontSize: 18, fontWeight: 700, color: C.textPrimary }}>🎼 五线谱编辑器</div>
            <div style={{ fontSize: 13, color: C.textMuted, marginTop: 2 }}>编辑乐谱（ABC 记谱法），融入第 {pageNum} 页课件</div>
          </div>
          <button onClick={onClose} style={{
            width: 36, height: 36, borderRadius: '50%', border: '1px solid ' + C.border,
            background: '#fff', fontSize: 18, cursor: 'pointer', display: 'flex',
            alignItems: 'center', justifyContent: 'center', color: C.textMuted,
          }}>✕</button>
        </div>

        {/* 主体：左输入 + 右预览/参数 */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 380px', gap: 24 }}>
          {/* 左栏：ABC 输入 + 模板 */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {/* ABC 输入框 */}
            <div>
              <label style={labelStyle}>ABC 记谱法（手动编辑或从下方模板点选）</label>
              <textarea value={abc} onChange={e => setAbc(e.target.value)}
                rows={10}
                style={{ width: '100%', padding: '10px 14px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 14, fontFamily: 'Menlo, Consolas, monospace', outline: 'none', resize: 'vertical', boxSizing: 'border-box', lineHeight: 1.6 }} />
            </div>

            {/* ABC 快速参考 */}
            <div style={{ padding: '10px 14px', borderRadius: 8, background: '#FFFBEB', border: '1px solid #FDE68A', fontSize: 12, color: '#92400E', lineHeight: 1.7 }}>
              <b>ABC 记谱法速查：</b>
              <br />音符：<code>C D E F G A B</code>（低八度）<code>c d e f g a b</code>（高八度）
              <br />升降号：<code>^C</code>（升C） <code>_B</code>（降B） <code>=E</code>（还原E）
              <br />时值：<code>C2</code>（二分） <code>C/</code>（八分） <code>C/4</code>（十六分）
              <br />休止：<code>z</code> &nbsp; 小节线：<code>|</code> &nbsp; 终止线：<code>|]</code>
              <br />和弦标记：<code>&quot;Am&quot;A</code> &nbsp; 连音线：<code>(CEG)</code>
            </div>

            {/* 模板 */}
            <div style={{ background: '#FAFAFA', borderRadius: 10, padding: '12px 14px', border: '1px solid ' + C.border }}>
              <div style={{ fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 8 }}>🎵 常用乐谱模板（点选即替换上方输入框）</div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                {MUSIC_TEMPLATES.map(tpl => (
                  <button key={tpl.label} onClick={() => handleTemplateClick(tpl.abc)}
                    style={{ padding: '6px 12px', borderRadius: 8, border: '1px solid ' + C.border, background: '#fff', color: '#374151', fontSize: 13, cursor: 'pointer' }}>
                    {tpl.label}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* 右栏：预览 + 参数 */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {/* 实时预览 */}
            <div style={{
              background: '#FFFDF7', border: '1px solid #E5E0D5', borderRadius: 12,
              padding: 20, minHeight: 160,
            }}>
              <div style={{ fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span>🎼 实时预览</span>
                {abcjsLoaded && abc.trim() && (
                  <button onClick={handlePlayPreview} style={{
                    padding: '4px 12px', borderRadius: 6, border: '1px solid #D4A574',
                    background: playing ? '#FEF3C7' : '#FFF8F0', color: '#8B6914',
                    fontSize: 12, fontWeight: 600, cursor: 'pointer',
                  }}>
                    {playing ? '⏹ 停止' : '▶ 试听'}
                  </button>
                )}
              </div>
              {!abcjsLoaded ? (
                <div style={{ color: C.textMuted, fontSize: 13, textAlign: 'center', padding: 20 }}>⏳ 加载 abcjs 中...</div>
              ) : previewError ? (
                <div style={{ color: '#DC2626', fontSize: 13 }}>❌ {previewError}</div>
              ) : (
                <div ref={previewRef} style={{ width: '100%' }} />
              )}
            </div>

            {/* 调号 */}
            <ParamSection title="调号">
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {KEY_OPTIONS.map(k => {
                  const current = getAbcField('K')
                  return (
                    <button key={k.key} onClick={() => updateAbcField('K', k.key)} style={{
                      padding: '5px 10px', borderRadius: 6, fontSize: 12, cursor: 'pointer',
                      border: '1px solid ' + (current === k.key ? '#B45309' : C.border),
                      background: current === k.key ? '#FFFBEB' : '#fff',
                      color: current === k.key ? '#B45309' : C.textSecondary,
                      fontWeight: current === k.key ? 600 : 400,
                    }}>{k.label}</button>
                  )
                })}
              </div>
            </ParamSection>

            {/* 拍号 */}
            <ParamSection title="拍号">
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {METER_OPTIONS.map(m => {
                  const current = getAbcField('M')
                  return (
                    <button key={m.meter} onClick={() => updateAbcField('M', m.meter)} style={{
                      padding: '5px 10px', borderRadius: 6, fontSize: 12, cursor: 'pointer',
                      border: '1px solid ' + (current === m.meter ? '#B45309' : C.border),
                      background: current === m.meter ? '#FFFBEB' : '#fff',
                      color: current === m.meter ? '#B45309' : C.textSecondary,
                      fontWeight: current === m.meter ? 600 : 400,
                    }}>{m.label}</button>
                  )
                })}
              </div>
            </ParamSection>

            {/* 乐谱宽度 */}
            <ParamSection title="乐谱宽度">
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <input type="range" min={300} max={800} step={50} value={width}
                  onChange={e => setWidth(Number(e.target.value))} style={{ flex: 1 }} />
                <span style={{ fontSize: 13, fontWeight: 600, minWidth: 50, color: C.textPrimary }}>{width}px</span>
              </div>
            </ParamSection>

            {/* 播放按钮 */}
            <ParamSection title="课件内播放按钮">
              <div style={{ display: 'flex', gap: 8 }}>
                {([true, false] as const).map(v => (
                  <button key={String(v)} onClick={() => setShowPlayer(v)} style={{
                    flex: 1, padding: '8px 6px', borderRadius: 8, fontSize: 13, cursor: 'pointer',
                    border: '1.5px solid ' + (showPlayer === v ? '#B45309' : C.border),
                    background: showPlayer === v ? '#FFFBEB' : '#fff',
                    color: showPlayer === v ? '#B45309' : C.textSecondary,
                    fontWeight: showPlayer === v ? 600 : 400,
                  }}>{v ? '🔊 显示播放按钮' : '🔇 不显示'}</button>
                ))}
              </div>
            </ParamSection>

            {/* 自定义标题 */}
            <ParamSection title="自定义标题（可选，覆盖 ABC 中的 T:）">
              <input type="text" value={customTitle} onChange={e => setCustomTitle(e.target.value)}
                placeholder="留空则使用 ABC 文本中的 T: 标题"
                style={{ width: '100%', padding: '8px 12px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 13, outline: 'none', boxSizing: 'border-box' }} />
            </ParamSection>

            {/* 位置偏好 */}
            <ParamSection title="融入位置偏好（可选）">
              <input type="text" value={positionHint} onChange={e => setPositionHint(e.target.value)}
                placeholder="如：放在歌词上方、替换现有乐谱区域"
                style={{ width: '100%', padding: '8px 12px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 13, outline: 'none', boxSizing: 'border-box' }} />
            </ParamSection>

            {/* 融入按钮 */}
            <button onClick={handleInsert} disabled={!abc.trim() || inserting}
              style={{
                marginTop: 4, padding: '14px 24px', borderRadius: 10, border: 'none',
                background: abc.trim() && !inserting
                  ? 'linear-gradient(135deg, #B45309, #D97706)' : '#E5E7EB',
                color: abc.trim() && !inserting ? '#fff' : '#9CA3AF',
                fontSize: 15, fontWeight: 700, cursor: abc.trim() && !inserting ? 'pointer' : 'default',
                width: '100%', boxShadow: abc.trim() && !inserting ? '0 4px 16px rgba(180,83,9,0.3)' : 'none',
              }}
            >
              {inserting ? '⏳ AI 正在融入当前页...' : '🎼 融入第 ' + pageNum + ' 页课件'}
            </button>
            <div style={{ fontSize: 12, color: C.textMuted, lineHeight: 1.5, textAlign: 'center' }}>
              点击后 AI 会把五线谱自然融入当前页面的布局中
            </div>
          </div>
        </div>

        {/* Toast */}
        {toast && (
          <div style={{
            position: 'fixed', bottom: 30, left: '50%', transform: 'translateX(-50%)',
            background: '#1F2937', color: '#fff', padding: '8px 24px', borderRadius: 8,
            fontSize: 13, zIndex: 99999, boxShadow: '0 4px 12px rgba(0,0,0,0.2)',
          }}>{toast}</div>
        )}
      </div>
    </div>
  )
}

// ==================== 子组件 ====================

function ParamSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ background: '#FAFAFA', borderRadius: 10, padding: '12px 16px', border: '1px solid ' + C.border }}>
      <div style={{ fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 8 }}>{title}</div>
      {children}
    </div>
  )
}

const labelStyle: React.CSSProperties = { fontSize: 12, fontWeight: 600, color: '#6B7280', display: 'block', marginBottom: 4 }
