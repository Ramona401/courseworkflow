/**
 * StrokeOrderModal.tsx — 课件笔顺动画生成器弹窗
 *
 * 复用 VideoEditorModal 的全屏弹窗交互范式。
 * 老师输入汉字 → 预览笔顺动画(动画/描红两模式) → 调参(速度/粗细/颜色/尺寸/布局)
 * → 点"融入当前页" → 拼微调指令经 onInsert 回调交给父组件调 RefinePage。
 *
 * 自托管说明（批次0b 换源，2026-07-08）：
 *   Hanzi Writer 库与笔画数据均已自托管，地址常量从 strokeOrderUtils 导入
 *   （HANZI_WRITER_CDN / HANZI_DATA_BASE，单一事实来源，本文件不再内联地址）。
 *   弹窗实时预览的 HanziWriter.create 显式传 charDataLoader 指向自托管笔画数据——
 *   js 与单字数据是两条独立加载链路，只换 js 不接管数据加载的话，
 *   每个字的笔画 JSON 仍会偷偷走 jsdelivr 外网。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/StrokeOrderModal.tsx
 * 依赖: strokeOrderUtils.ts(同目录), Hanzi Writer 自托管(本服务器 libs 目录)
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import {
  extractValidChars,
  STROKE_DEFAULTS,
  STROKE_COLOR_PRESETS,
  STROKE_SPEED_LABELS,
  buildStrokeRefineInstruction,
  HANZI_WRITER_CDN,
  HANZI_DATA_BASE,
  type StrokeCharConfig,
} from './strokeOrderUtils'
import { C } from './workshopConstants'

// ==================== 类型 ====================

interface Props {
  /** 当前课件页号（显示用） */
  pageNum: number
  /** 融入回调：父组件收到微调指令后调 RefinePage */
  onInsert: (instruction: string) => void
  /** 关闭弹窗 */
  onClose: () => void
  /** 是否正在执行微调（禁用融入按钮） */
  inserting?: boolean
}

// ==================== Hanzi Writer 全局类型声明 ====================

declare const HanziWriter: {
  create: (el: string | HTMLElement, char: string, opts: Record<string, unknown>) => HanziWriterInstance
}

interface HanziWriterInstance {
  animateCharacter: (opts?: { onComplete?: () => void }) => void
  animateStroke: (idx: number, opts?: { onComplete?: () => void }) => void
  hideCharacter: () => void
  showOutline: () => void
  quiz: (opts: {
    onMistake?: (data: { strokeNum: number }) => void
    onCorrectStroke?: (data: { strokeNum: number }) => void
    onComplete?: () => void
  }) => void
}

// ==================== 常量 ====================

/** 布局选项 */
const LAYOUT_OPTIONS = [
  { key: 'horizontal' as const, label: '横排', emoji: '↔️' },
  { key: 'vertical' as const, label: '竖排', emoji: '↕️' },
  { key: 'grid' as const, label: '网格', emoji: '⊞' },
]

// ==================== 组件 ====================

export default function StrokeOrderModal({ pageNum, onInsert, onClose, inserting }: Props) {
  // ---- 输入与字符状态 ----
  const [inputText, setInputText] = useState('')
  const [chars, setChars] = useState<string[]>([])
  const [activeIdx, setActiveIdx] = useState(0)

  // ---- 参数状态 ----
  const [speed, setSpeed] = useState(STROKE_DEFAULTS.speed)
  const [strokeColor, setStrokeColor] = useState(STROKE_DEFAULTS.strokeColor)
  const [size, setSize] = useState(STROKE_DEFAULTS.size)
  const [mode, setMode] = useState<'animate' | 'quiz'>('animate')
  const [layout, setLayout] = useState<'horizontal' | 'vertical' | 'grid'>('horizontal')
  const [positionHint, setPositionHint] = useState('')

  // ---- 预览状态 ----
  const [hwLoaded, setHwLoaded] = useState(false)
  const [writerInstance, setWriterInstance] = useState<HanziWriterInstance | null>(null)
  const [totalStrokes, setTotalStrokes] = useState(0)
  const [currentStroke, setCurrentStroke] = useState(0)
  const [statusText, setStatusText] = useState('输入汉字后点"加载预览"')
  const [toast, setToast] = useState('')

  const writerTargetRef = useRef<HTMLDivElement>(null)

  // ---- 加载 Hanzi Writer（自托管） ----
  useEffect(() => {
    if (typeof (window as unknown as Record<string, unknown>).HanziWriter !== 'undefined') {
      setHwLoaded(true)
      return
    }
    const existing = document.querySelector('script[src*="hanzi-writer"]')
    if (existing) {
      const check = setInterval(() => {
        if (typeof (window as unknown as Record<string, unknown>).HanziWriter !== 'undefined') {
          setHwLoaded(true)
          clearInterval(check)
        }
      }, 200)
      return () => clearInterval(check)
    }
    const s = document.createElement('script')
    s.src = HANZI_WRITER_CDN
    s.onload = () => setHwLoaded(true)
    document.head.appendChild(s)
  }, [])

  // ---- 加载字符列表 ----
  const handleLoadChars = useCallback(() => {
    const valid = extractValidChars(inputText)
    if (valid.length === 0) {
      showToast('请输入至少一个汉字')
      return
    }
    setChars(valid)
    setActiveIdx(0)
  }, [inputText])

  // ---- 当 activeIdx/chars 变化或参数变化时重建 writer ----
  useEffect(() => {
    if (!hwLoaded || chars.length === 0 || !writerTargetRef.current) return
    const char = chars[activeIdx]
    if (!char) return

    // 清空旧 writer
    const target = writerTargetRef.current
    target.innerHTML = ''
    setCurrentStroke(0)
    setTotalStrokes(0)
    setStatusText('加载中...')

    try {
      const w = HanziWriter.create(target, char, {
        width: size,
        height: size,
        padding: 12,
        strokeAnimationSpeed: speed * 0.5,
        delayBetweenStrokes: 300,
        strokeColor: strokeColor,
        outlineColor: '#e0d6cc',
        radicalColor: strokeColor,
        drawingColor: strokeColor,
        showOutline: true,
        showCharacter: false,
        renderer: 'svg',
        // 笔画数据走自托管（否则默认仍从 jsdelivr 拉取单字 JSON）
        charDataLoader: (c: string) =>
          fetch(HANZI_DATA_BASE + encodeURIComponent(c) + '.json').then(r => {
            if (!r.ok) throw new Error('HTTP ' + r.status)
            return r.json()
          }),
        onLoadCharDataSuccess: (data: { strokes: unknown[] }) => {
          setTotalStrokes(data.strokes.length)
          if (mode === 'quiz') {
            startQuiz(w, data.strokes.length)
          } else {
            setStatusText('点击"播放动画"查看笔顺')
          }
        },
        onLoadCharDataError: () => {
          setStatusText('该字暂无笔画数据')
          setTotalStrokes(0)
        },
      })
      setWriterInstance(w)
    } catch {
      setStatusText('加载失败')
    }
  }, [hwLoaded, chars, activeIdx, size, speed, strokeColor, mode])

  // ---- 描红模式启动 ----
  const startQuiz = (w: HanziWriterInstance, total: number) => {
    setStatusText('请在田字格内按笔顺书写')
    setCurrentStroke(0)
    w.quiz({
      onMistake: (d) => setStatusText('笔顺有误，请重试第 ' + (d.strokeNum + 1) + ' 笔'),
      onCorrectStroke: (d) => {
        setCurrentStroke(d.strokeNum + 1)
        setStatusText('正确！第 ' + (d.strokeNum + 1) + ' / ' + total + ' 笔')
      },
      onComplete: () => {
        setCurrentStroke(total)
        setStatusText('恭喜！书写完全正确！')
      },
    })
  }

  // ---- 动画控制 ----
  const handleAnimate = () => {
    if (!writerInstance) return
    setCurrentStroke(0)
    setStatusText('正在播放笔顺动画...')
    writerInstance.animateCharacter({
      onComplete: () => {
        setCurrentStroke(totalStrokes)
        setStatusText('播放完成')
      },
    })
  }

  const handleAnimateNext = () => {
    if (!writerInstance || totalStrokes === 0) return
    if (currentStroke === 0) writerInstance.hideCharacter()
    if (currentStroke < totalStrokes) {
      writerInstance.animateStroke(currentStroke, {
        onComplete: () => {
          const next = currentStroke + 1
          setCurrentStroke(next)
          setStatusText(next >= totalStrokes ? '全部笔画完成' : '第 ' + next + ' / ' + totalStrokes + ' 笔')
        },
      })
    }
  }

  const handleReset = () => {
    if (!writerInstance) return
    writerInstance.hideCharacter()
    writerInstance.showOutline()
    setCurrentStroke(0)
    setStatusText('已重置')
    if (mode === 'quiz' && totalStrokes > 0) {
      startQuiz(writerInstance, totalStrokes)
    }
  }

  // ---- 融入当前页 ----
  const handleInsert = () => {
    if (chars.length === 0) { showToast('请先加载汉字'); return }
    // 构建全部字符的配置
    const charConfigs: StrokeCharConfig[] = chars.map(ch => ({
      char: ch,
      speed,
      strokeColor,
      size,
      showGrid: true,
      clickToAnimate: mode === 'animate',
      quizMode: mode === 'quiz',
    }))
    const instruction = buildStrokeRefineInstruction({
      chars: charConfigs,
      positionHint: positionHint.trim() || undefined,
      layout,
    })
    onInsert(instruction)
  }

  // ---- Toast ----
  const showToast = (msg: string) => {
    setToast(msg)
    setTimeout(() => setToast(''), 2500)
  }

  // ---- 进度百分比 ----
  const progressPct = totalStrokes > 0 ? Math.round((currentStroke / totalStrokes) * 100) : 0

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 99995,
      background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}>
      <div style={{
        width: '90vw', maxWidth: 960, maxHeight: '90vh', overflow: 'auto',
        background: '#fff', borderRadius: 16, boxShadow: '0 20px 60px rgba(0,0,0,0.3)',
        padding: 28, position: 'relative',
      }}>
        {/* 顶栏 */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
          <div>
            <div style={{ fontSize: 18, fontWeight: 700, color: C.textPrimary }}>✍️ 笔顺动画生成器</div>
            <div style={{ fontSize: 13, color: C.textMuted, marginTop: 2 }}>生成汉字笔顺动画，融入第 {pageNum} 页课件</div>
          </div>
          <button onClick={onClose} style={{
            width: 36, height: 36, borderRadius: '50%', border: '1px solid ' + C.border,
            background: '#fff', fontSize: 18, cursor: 'pointer', display: 'flex',
            alignItems: 'center', justifyContent: 'center', color: C.textMuted,
          }}>✕</button>
        </div>

        {/* 输入区 */}
        <div style={{ display: 'flex', gap: 10, marginBottom: 16 }}>
          <input
            type="text" value={inputText} onChange={e => setInputText(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') handleLoadChars() }}
            placeholder="输入要练习的汉字，如：花病医治别干"
            style={{ flex: 1, padding: '10px 14px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 15, outline: 'none' }}
          />
          <button onClick={handleLoadChars} style={{
            padding: '10px 20px', borderRadius: 8, border: 'none',
            background: 'linear-gradient(135deg, #F59E0B, #EF4444)', color: '#fff',
            fontSize: 14, fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap',
          }}>加载预览</button>
        </div>

        {/* 字符切换 Tab */}
        {chars.length > 0 && (
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16 }}>
            {chars.map((ch, i) => (
              <button key={ch + i} onClick={() => setActiveIdx(i)} style={{
                minWidth: 44, height: 44, fontSize: 22, fontWeight: 600,
                borderRadius: 10, cursor: 'pointer',
                border: '2px solid ' + (i === activeIdx ? C.primary : C.border),
                background: i === activeIdx ? C.primaryBg : '#fff',
                color: i === activeIdx ? C.primary : C.textPrimary,
                transition: 'all 0.15s',
              }}>{ch}</button>
            ))}
          </div>
        )}

        {/* 主体：左预览 + 右参数 */}
        <div style={{ display: 'grid', gridTemplateColumns: chars.length > 0 ? '340px 1fr' : '1fr', gap: 24 }}>
          {/* 左栏：田字格预览 */}
          {chars.length > 0 && (
            <div style={{
              background: '#FAFAFA', border: '1px solid ' + C.border, borderRadius: 12,
              padding: 20, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 14,
            }}>
              {/* 田字格容器 */}
              <div style={{
                width: size, height: size, maxWidth: 300, maxHeight: 300,
                border: '2px solid #d4c8bb', borderRadius: 10, position: 'relative',
                background: '#fffdf9', overflow: 'hidden',
              }}>
                {/* 米字格辅助线 */}
                <div style={{ position: 'absolute', inset: 0, pointerEvents: 'none', zIndex: 0 }}>
                  <div style={{ position: 'absolute', top: '50%', left: 0, right: 0, borderTop: '1px dashed #e0d6cc' }} />
                  <div style={{ position: 'absolute', left: '50%', top: 0, bottom: 0, borderLeft: '1px dashed #e0d6cc' }} />
                </div>
                <div ref={writerTargetRef} style={{ position: 'relative', zIndex: 1, width: '100%', height: '100%' }} />
              </div>

              {/* 状态文字 */}
              <div style={{ fontSize: 13, color: C.textSecondary, textAlign: 'center', minHeight: 20 }}>{statusText}</div>

              {/* 进度条 */}
              {totalStrokes > 0 && (
                <div style={{ width: '100%' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, color: C.textMuted, marginBottom: 4 }}>
                    <span>{chars[activeIdx]} · {totalStrokes} 画</span>
                    <span>{currentStroke} / {totalStrokes}</span>
                  </div>
                  <div style={{ height: 6, borderRadius: 3, background: '#F3F4F6', overflow: 'hidden' }}>
                    <div style={{ height: '100%', borderRadius: 3, background: C.primary, transition: 'width 0.3s', width: progressPct + '%' }} />
                  </div>
                </div>
              )}

              {/* 控制按钮 */}
              {mode === 'animate' && totalStrokes > 0 && (
                <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', justifyContent: 'center' }}>
                  <button onClick={handleAnimate} style={ctrlBtn}>▶ 播放</button>
                  <button onClick={handleAnimateNext} style={ctrlBtn}>→ 逐笔</button>
                  <button onClick={handleReset} style={ctrlBtn}>↺ 重置</button>
                </div>
              )}
              {mode === 'quiz' && totalStrokes > 0 && (
                <div style={{ fontSize: 12, color: '#0891B2', textAlign: 'center' }}>
                  用鼠标在田字格内按正确笔顺书写
                  <div style={{ marginTop: 6 }}>
                    <button onClick={handleReset} style={ctrlBtn}>↺ 重来</button>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* 右栏：参数配置 */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {/* 模式切换 */}
            <ParamSection title="模式">
              <div style={{ display: 'flex', gap: 8 }}>
                {(['animate', 'quiz'] as const).map(m => (
                  <button key={m} onClick={() => setMode(m)} style={{
                    flex: 1, padding: '10px 8px', borderRadius: 8, fontSize: 13, cursor: 'pointer',
                    border: '1.5px solid ' + (mode === m ? C.primary : C.border),
                    background: mode === m ? C.primaryBg : '#fff',
                    color: mode === m ? C.primary : C.textSecondary,
                    fontWeight: mode === m ? 600 : 400,
                  }}>{m === 'animate' ? '▶ 动画演示' : '✏️ 描红练习'}</button>
                ))}
              </div>
            </ParamSection>

            {/* 速度 */}
            <ParamSection title="笔画速度">
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <input type="range" min={1} max={5} step={1} value={speed}
                  onChange={e => setSpeed(Number(e.target.value))}
                  style={{ flex: 1 }} />
                <span style={{ fontSize: 13, fontWeight: 600, minWidth: 40, color: C.textPrimary }}>
                  {STROKE_SPEED_LABELS[speed] || speed}
                </span>
              </div>
            </ParamSection>

            {/* 尺寸 */}
            <ParamSection title="田字格尺寸">
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <input type="range" min={150} max={360} step={10} value={size}
                  onChange={e => setSize(Number(e.target.value))}
                  style={{ flex: 1 }} />
                <span style={{ fontSize: 13, fontWeight: 600, minWidth: 50, color: C.textPrimary }}>{size}px</span>
              </div>
            </ParamSection>

            {/* 笔画颜色 */}
            <ParamSection title="笔画颜色">
              <div style={{ display: 'flex', gap: 8 }}>
                {STROKE_COLOR_PRESETS.map(p => (
                  <div key={p.color} onClick={() => setStrokeColor(p.color)}
                    title={p.label}
                    style={{
                      width: 32, height: 32, borderRadius: 8, cursor: 'pointer',
                      background: p.color, transition: 'all 0.15s',
                      border: strokeColor === p.color ? '3px solid ' + C.primary : '1px solid ' + C.border,
                      transform: strokeColor === p.color ? 'scale(1.15)' : 'scale(1)',
                    }} />
                ))}
              </div>
            </ParamSection>

            {/* 多字布局 */}
            {chars.length > 1 && (
              <ParamSection title={'多字布局（共 ' + chars.length + ' 字）'}>
                <div style={{ display: 'flex', gap: 8 }}>
                  {LAYOUT_OPTIONS.map(lo => (
                    <button key={lo.key} onClick={() => setLayout(lo.key)} style={{
                      flex: 1, padding: '8px 6px', borderRadius: 8, fontSize: 13, cursor: 'pointer',
                      border: '1.5px solid ' + (layout === lo.key ? C.primary : C.border),
                      background: layout === lo.key ? C.primaryBg : '#fff',
                      color: layout === lo.key ? C.primary : C.textSecondary,
                      fontWeight: layout === lo.key ? 600 : 400,
                    }}>{lo.emoji} {lo.label}</button>
                  ))}
                </div>
              </ParamSection>
            )}

            {/* 位置偏好 */}
            <ParamSection title="融入位置偏好（可选）">
              <input type="text" value={positionHint} onChange={e => setPositionHint(e.target.value)}
                placeholder="如：放在生字卡片旁边、替换现有生字展示区"
                style={{ width: '100%', padding: '8px 12px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 13, outline: 'none', boxSizing: 'border-box' }}
              />
            </ParamSection>

            {/* 融入按钮 */}
            <button onClick={handleInsert} disabled={chars.length === 0 || inserting}
              style={{
                marginTop: 4, padding: '14px 24px', borderRadius: 10, border: 'none',
                background: chars.length > 0 && !inserting
                  ? 'linear-gradient(135deg, #F59E0B, #EF4444)' : '#E5E7EB',
                color: chars.length > 0 && !inserting ? '#fff' : '#9CA3AF',
                fontSize: 15, fontWeight: 700, cursor: chars.length > 0 && !inserting ? 'pointer' : 'default',
                width: '100%', boxShadow: chars.length > 0 && !inserting ? '0 4px 16px rgba(245,158,11,0.3)' : 'none',
              }}
            >
              {inserting ? '⏳ AI 正在融入当前页...' : '✍️ 融入第 ' + pageNum + ' 页课件'}
            </button>
            <div style={{ fontSize: 12, color: C.textMuted, lineHeight: 1.5, textAlign: 'center' }}>
              点击后 AI 会把笔顺动画自然融入当前页面的布局中
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

/** 参数区块容器 */
function ParamSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ background: '#FAFAFA', borderRadius: 10, padding: '12px 16px', border: '1px solid ' + C.border }}>
      <div style={{ fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 8 }}>{title}</div>
      {children}
    </div>
  )
}

/** 控制按钮通用样式 */
const ctrlBtn: React.CSSProperties = {
  padding: '6px 16px', borderRadius: 8, border: '1px solid ' + C.border,
  background: '#fff', color: C.textPrimary, fontSize: 13, fontWeight: 500, cursor: 'pointer',
}
