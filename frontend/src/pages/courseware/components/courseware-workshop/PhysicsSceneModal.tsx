/**
 * PhysicsSceneModal.tsx — 力学场景编辑器弹窗
 *
 * 第6批C2：
 * - 接入 PhysicsSceneAIPanel；
 * - 支持 AI 新建力学场景；
 * - 支持基于现有 Matter.js 模板 AI 改编；
 * - AI 生成的是 setup 构造代码，不是 HTML；
 * - 继续复用现有 Matter.js 预览、播放、暂停、重置和融入链路。
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import { C } from './workshopConstants'
import {
  MATTER_JS_CDN, PHYSICS_DEFAULTS, PHYSICS_SIZE_PRESETS,
  buildPhysicsRefineInstruction,
} from './physicsUtils'
import type { PhysicsTemplate, PhysicsParamValue } from './physicsUtils'
import { PHYSICS_TEMPLATES, getPhysicsTemplateGroups } from './physicsTemplates'
import PhysicsSceneAIPanel from './PhysicsSceneAIPanel'

declare global {
  interface Window { Matter?: any }
}

interface Props {
  pageNum: number
  onInsert: (instruction: string) => void
  onClose: () => void
  inserting?: boolean
}

const MODAL_CSS = [
  '@keyframes ph-shimmer { 0% { background-position: -420px 0; } 100% { background-position: 420px 0; } }',
  '.ph-skeleton { background: linear-gradient(90deg, #F3F4F6 25%, #FEE2E2 40%, #F3F4F6 60%); background-size: 840px 100%; animation: ph-shimmer 1.3s infinite linear; }',
  '.ph-card { transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s; }',
  '.ph-card:hover { border-color: #FCA5A5 !important; box-shadow: 0 8px 24px rgba(220,38,38,0.14); transform: translateY(-1px); }',
  '.ph-ai-btn { opacity: 0; transition: opacity 0.15s; }',
  '.ph-card:hover .ph-ai-btn { opacity: 1; }',
  '.ph-range { -webkit-appearance: none; appearance: none; width: 100%; height: 6px; border-radius: 3px; background: #FEE2E2; outline: none; cursor: pointer; }',
  '.ph-range::-webkit-slider-thumb { -webkit-appearance: none; appearance: none; width: 18px; height: 18px; border-radius: 50%; background: linear-gradient(135deg, #F87171, #B91C1C); border: 2.5px solid #fff; box-shadow: 0 1px 4px rgba(185,28,28,0.45); cursor: pointer; }',
  '.ph-range::-moz-range-thumb { width: 18px; height: 18px; border-radius: 50%; background: #DC2626; border: 2.5px solid #fff; box-shadow: 0 1px 4px rgba(185,28,28,0.45); cursor: pointer; }',
].join('\n')

function loadMatterLib(): Promise<void> {
  return new Promise((resolve, reject) => {
    if (window.Matter) { resolve(); return }
    const existing = document.querySelector('script[src*="matter.min.js"]')
    if (existing) {
      let tries = 0
      const timer = window.setInterval(() => {
        tries++
        if (window.Matter) { window.clearInterval(timer); resolve() }
        else if (tries > 100) { window.clearInterval(timer); reject(new Error('Matter.js 加载超时')) }
      }, 100)
      return
    }
    const s = document.createElement('script')
    s.src = MATTER_JS_CDN
    s.onload = () => { window.Matter ? resolve() : reject(new Error('Matter.js 加载后未就绪')) }
    s.onerror = () => reject(new Error('Matter.js 脚本加载失败'))
    document.head.appendChild(s)
  })
}

function buildDefaultParams(tpl: PhysicsTemplate): Record<string, PhysicsParamValue> {
  const out: Record<string, PhysicsParamValue> = {}
  for (const p of tpl.params) out[p.key] = p.defaultValue
  return out
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <div
      onClick={() => onChange(!checked)}
      style={{
        width: 40, height: 22, borderRadius: 11, flexShrink: 0, cursor: 'pointer',
        background: checked ? 'linear-gradient(135deg, #F87171, #B91C1C)' : '#D1D5DB',
        position: 'relative', transition: 'background 0.18s',
      }}
    >
      <div style={{
        position: 'absolute', top: 2, left: checked ? 20 : 2, width: 18, height: 18, borderRadius: '50%',
        background: '#fff', boxShadow: '0 1px 3px rgba(0,0,0,0.28)', transition: 'left 0.18s',
      }} />
    </div>
  )
}

export default function PhysicsSceneModal({ pageNum, onInsert, onClose, inserting }: Props) {
  const [libReady, setLibReady] = useState(false)
  const [libError, setLibError] = useState('')
  const [previewError, setPreviewError] = useState('')
  const [activeTplId, setActiveTplId] = useState(PHYSICS_TEMPLATES[0].id)
  const [params, setParams] = useState<Record<string, PhysicsParamValue>>(buildDefaultParams(PHYSICS_TEMPLATES[0]))
  const [sizeIdx, setSizeIdx] = useState(1)
  const [autoplay, setAutoplay] = useState<boolean>(PHYSICS_DEFAULTS.autoplay)
  const [showVelocity, setShowVelocity] = useState<boolean>(PHYSICS_DEFAULTS.showVelocity)
  const [playing, setPlaying] = useState(false)
  const [caption, setCaption] = useState('')

  const [aiMode, setAiMode] = useState<'adapt' | 'create' | null>(null)
  const [aiCode, setAiCode] = useState('')
  const [aiBaseCode, setAiBaseCode] = useState('')

  const stageRef = useRef<HTMLDivElement | null>(null)
  const engineRef = useRef<any>(null)
  const renderRef = useRef<any>(null)
  const runnerRef = useRef<any>(null)
  const debounceRef = useRef<number | null>(null)

  const activeTpl = PHYSICS_TEMPLATES.find(t => t.id === activeTplId) || PHYSICS_TEMPLATES[0]
  const size = PHYSICS_SIZE_PRESETS[sizeIdx] || PHYSICS_SIZE_PRESETS[1]
  const isAI = aiMode !== null

  useEffect(() => {
    let cancelled = false
    loadMatterLib()
      .then(() => { if (!cancelled) setLibReady(true) })
      .catch(e => { if (!cancelled) setLibError(e instanceof Error ? e.message : '库加载失败') })
    return () => { cancelled = true }
  }, [])

  const teardown = useCallback(() => {
    const M = window.Matter
    if (!M) return
    try {
      if (renderRef.current) {
        M.Render.stop(renderRef.current)
        renderRef.current.canvas?.remove()
      }
    } catch { /* 忽略 */ }
    try {
      if (runnerRef.current) M.Runner.stop(runnerRef.current)
    } catch { /* 忽略 */ }
    renderRef.current = null
    runnerRef.current = null
    engineRef.current = null
  }, [])

  const getSetupCode = useCallback(() => {
    if (isAI) {
      if (aiCode.trim()) return aiCode
      if (aiMode === 'adapt') return activeTpl.buildSetup(params)
      return ''
    }
    return activeTpl.buildSetup(params)
  }, [isAI, aiCode, aiMode, activeTpl, params])

  const runSetup = useCallback(() => {
    const M = window.Matter
    const engine = engineRef.current
    if (!M || !engine) return

    M.Composite.clear(engine.world, false)
    engine.gravity.x = 0
    engine.gravity.y = 1

    const code = getSetupCode()
    if (!code.trim()) return

    // eslint-disable-next-line no-new-func
    new Function('Matter', 'engine', 'world', 'W', 'H', code)(M, engine, engine.world, size.width, size.height)
  }, [getSetupCode, size.width, size.height])

  const rebuildPreview = useCallback(() => {
    if (!libReady || !stageRef.current || !window.Matter) return

    setPreviewError('')
    teardown()

    const M = window.Matter
    const host = stageRef.current
    host.innerHTML = ''

    try {
      const engine = M.Engine.create()
      engineRef.current = engine

      const render = M.Render.create({
        element: host,
        engine,
        options: {
          width: size.width,
          height: size.height,
          wireframes: false,
          background: '#FFFFFF',
          showVelocity,
          pixelRatio: window.devicePixelRatio || 1,
        },
      })
      renderRef.current = render

      const runner = M.Runner.create()
      runnerRef.current = runner

      runSetup()
      M.Render.run(render)
      M.Runner.run(runner, engine)
      runner.enabled = false
      setPlaying(false)
    } catch (e) {
      setPreviewError('预览构建失败: ' + (e instanceof Error ? e.message : String(e)))
    }
  }, [libReady, size.width, size.height, showVelocity, runSetup, teardown])

  useEffect(() => {
    if (!libReady) return
    if (debounceRef.current) window.clearTimeout(debounceRef.current)
    debounceRef.current = window.setTimeout(() => { rebuildPreview() }, 300)
    return () => { if (debounceRef.current) window.clearTimeout(debounceRef.current) }
  }, [libReady, rebuildPreview])

  useEffect(() => () => { teardown() }, [teardown])

  const handleToggle = () => {
    const runner = runnerRef.current
    if (!runner) return
    runner.enabled = !runner.enabled
    setPlaying(runner.enabled)
  }

  const handleReset = () => {
    if (!engineRef.current) return
    try {
      runSetup()
      if (runnerRef.current) runnerRef.current.enabled = false
      setPlaying(false)
      setPreviewError('')
    } catch (e) {
      setPreviewError('重置失败: ' + (e instanceof Error ? e.message : String(e)))
    }
  }

  const handlePickTemplate = (tpl: PhysicsTemplate) => {
    setActiveTplId(tpl.id)
    setParams(buildDefaultParams(tpl))
    setPreviewError('')
    setAiMode(null)
    setAiCode('')
    setAiBaseCode('')
  }

  const enterAdaptMode = (tpl: PhysicsTemplate) => {
    const p = tpl.id === activeTplId ? params : buildDefaultParams(tpl)
    setActiveTplId(tpl.id)
    setParams(p)
    setAiBaseCode(tpl.buildSetup(p))
    setAiCode('')
    setAiMode('adapt')
    setPreviewError('')
  }

  const enterCreateMode = () => {
    setAiBaseCode('')
    setAiCode('')
    setAiMode('create')
    setPreviewError('')
  }

  const exitAIMode = () => {
    setAiMode(null)
    setAiCode('')
    setAiBaseCode('')
    setPreviewError('')
  }

  const setParam = (key: string, v: PhysicsParamValue) => {
    setParams(prev => ({ ...prev, [key]: v }))
  }

  const handleInsert = () => {
    if (inserting) return

    let tplForInsert: PhysicsTemplate = activeTpl
    let paramsForInsert: Record<string, PhysicsParamValue> = params

    if (isAI) {
      if (!aiCode.trim()) return
      const frozenCode = aiCode
      tplForInsert = {
        id: 'ai-physics-scene',
        group: '✨ AI定制',
        name: aiMode === 'adapt' ? activeTpl.name + '·AI改编' : 'AI自定义力学场景',
        emoji: '✨',
        desc: 'AI 定制的 Matter.js 力学仿真',
        params: [],
        buildSetup: () => frozenCode,
      }
      paramsForInsert = {}
    }

    const instruction = buildPhysicsRefineInstruction({
      scene: {
        template: tplForInsert,
        params: paramsForInsert,
        width: size.width,
        height: size.height,
        autoplay,
        showVelocity,
        caption: caption.trim() || undefined,
      },
    })
    onInsert(instruction)
  }

  const insertDisabled = inserting || !libReady || !!libError || (isAI && !aiCode.trim())

  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(17,24,39,0.62)', backdropFilter: 'blur(5px)', zIndex: 99993, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={() => { if (!inserting) onClose() }}
    >
      <style>{MODAL_CSS}</style>
      <div
        style={{ width: 'min(1520px, 98vw)', height: 'min(900px, 96vh)', background: '#fff', borderRadius: 24, display: 'flex', flexDirection: 'column', overflow: 'hidden', boxShadow: '0 34px 88px rgba(0,0,0,0.38)' }}
        onClick={e => e.stopPropagation()}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '16px 24px', background: 'linear-gradient(135deg, #F87171 0%, #DC2626 55%, #B91C1C 100%)', flexShrink: 0 }}>
          <span style={{ width: 48, height: 48, borderRadius: 16, background: 'rgba(255,255,255,0.18)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 25, flexShrink: 0 }}>🎯</span>
          <div style={{ minWidth: 0 }}>
            <div style={{ fontSize: 19, fontWeight: 850, color: '#fff', letterSpacing: 0.3 }}>力学场景</div>
            <div style={{ fontSize: 12.5, color: 'rgba(255,255,255,0.82)', marginTop: 2 }}>Matter.js 仿真 · AI 改编/新建 · 播放/暂停/重置 · 课堂反复演示</div>
          </div>
          {isAI && (
            <span style={{ padding: '6px 15px', borderRadius: 999, background: 'rgba(255,255,255,0.24)', border: '1px solid rgba(255,255,255,0.38)', color: '#fff', fontSize: 12.5, fontWeight: 800, whiteSpace: 'nowrap' }}>
              ✨ AI 定制中
            </span>
          )}
          <span style={{ marginLeft: 8, padding: '6px 15px', borderRadius: 999, background: 'rgba(255,255,255,0.18)', border: '1px solid rgba(255,255,255,0.32)', color: '#fff', fontSize: 12.5, fontWeight: 800, whiteSpace: 'nowrap' }}>
            将融入:第 {pageNum} 页
          </span>
          <button
            onClick={() => { if (!inserting) onClose() }}
            style={{ marginLeft: 'auto', border: 'none', background: 'rgba(255,255,255,0.15)', width: 36, height: 36, borderRadius: 12, fontSize: 18, cursor: 'pointer', color: '#fff', flexShrink: 0 }}
            title="关闭"
          >✕</button>
        </div>

        <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
          <div style={{ width: 300, borderRight: '1px solid ' + C.border, display: 'flex', flexDirection: 'column', flexShrink: 0, background: '#FDFAFA' }}>
            <div style={{ padding: '14px 14px 12px', borderBottom: '1px solid #F8ECEC', flexShrink: 0 }}>
              <div style={{ fontSize: 13, fontWeight: 850, color: '#B91C1C' }}>🎬 选择力学场景</div>
              <div style={{ fontSize: 11.5, color: C.textMuted, marginTop: 6, lineHeight: 1.55 }}>
                点模板直接使用；点“改编”可让 AI 在现有 Matter.js 场景上变种。
              </div>
              <button
                onClick={enterCreateMode}
                style={{
                  width: '100%', marginTop: 10, padding: '9px 0', borderRadius: 12, cursor: 'pointer',
                  border: '1.5px dashed ' + (aiMode === 'create' ? '#B91C1C' : '#FBD5D5'),
                  background: aiMode === 'create' ? 'linear-gradient(135deg,#FEF2F2,#FEE2E2)' : '#fff',
                  color: '#B91C1C', fontSize: 12.5, fontWeight: 800,
                }}
              >✨ AI 新建力学场景</button>
            </div>

            <div style={{ flex: 1, overflowY: 'auto', padding: '12px 12px 16px' }}>
              {getPhysicsTemplateGroups().map(g => (
                <div key={g.group} style={{ marginBottom: 14 }}>
                  <div style={{ fontSize: 11.5, fontWeight: 850, color: '#C97070', padding: '3px 6px 7px', letterSpacing: 0.4 }}>{g.group}</div>
                  {g.items.map(tpl => {
                    const active = tpl.id === activeTplId && !isAI
                    return (
                      <div key={tpl.id} className="ph-card" onClick={() => handlePickTemplate(tpl)}
                        style={{
                          display: 'flex', gap: 10, alignItems: 'flex-start', padding: '10px 11px 30px', borderRadius: 15, cursor: 'pointer', marginBottom: 8,
                          background: active ? 'linear-gradient(135deg, #FEF2F2, #FEE2E2)' : '#fff',
                          border: '1.5px solid ' + (active ? '#DC2626' : '#F6EBEB'),
                          boxShadow: active ? '0 8px 22px rgba(220,38,38,0.16)' : '0 2px 8px rgba(80,18,18,0.04)',
                          position: 'relative',
                        }}
                      >
                        <span style={{ width: 36, height: 36, borderRadius: 12, background: active ? '#fff' : '#FEF2F2', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 18, flexShrink: 0 }}>{tpl.emoji}</span>
                        <div style={{ minWidth: 0 }}>
                          <div style={{ fontSize: 13.5, fontWeight: active ? 850 : 650, color: active ? '#B91C1C' : C.textPrimary, lineHeight: 1.35 }}>{tpl.name}</div>
                          <div style={{ fontSize: 11.2, color: C.textMuted, marginTop: 4, lineHeight: 1.5 }}>{tpl.desc}</div>
                        </div>
                        <button
                          className="ph-ai-btn"
                          onClick={e => { e.stopPropagation(); enterAdaptMode(tpl) }}
                          title="以此模板为底稿，用自然语言改编"
                          style={{
                            position: 'absolute', right: 9, bottom: 7, padding: '4px 10px', borderRadius: 999, fontSize: 10.5, fontWeight: 800, cursor: 'pointer',
                            border: '1px solid #FBD5D5', background: '#fff', color: '#B91C1C',
                          }}
                        >🔧 改编</button>
                      </div>
                    )
                  })}
                </div>
              ))}
            </div>
          </div>

          <div style={{ flex: 1, display: 'flex', minWidth: 0 }}>
            <div style={{ flex: 1, overflow: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column', alignItems: 'center', minWidth: 0, background: 'radial-gradient(circle at 50% 0%,#FFFFFF 0%,#FCF7F7 58%,#FEF2F2 100%)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 14, flexWrap: 'wrap', justifyContent: 'center' }}>
                <span style={{ fontSize: 20 }}>{isAI ? '✨' : activeTpl.emoji}</span>
                <span style={{ fontSize: 16, fontWeight: 850, color: C.textPrimary }}>
                  {isAI ? (aiMode === 'adapt' ? activeTpl.name + ' · AI改编' : 'AI 新力学场景') : activeTpl.name}
                </span>
                <span style={{ fontSize: 12.5, color: C.textMuted }}>
                  {isAI ? (aiCode ? '当前显示 AI 生成结果' : '右侧描述后生成') : activeTpl.desc}
                </span>
              </div>

              {libError && (
                <div style={{ padding: '12px 16px', borderRadius: 10, background: '#FEE2E2', color: '#DC2626', fontSize: 13 }}>
                  ❌ {libError}
                </div>
              )}

              {!libReady && !libError && (
                <div className="ph-skeleton" style={{ width: size.width, height: size.height, borderRadius: 16, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <span style={{ fontSize: 13, color: '#C97070', fontWeight: 650 }}>⏳ 正在加载物理引擎…</span>
                </div>
              )}

              <div style={{ padding: 14, background: '#fff', borderRadius: 22, border: '1px solid #FEE2E2', boxShadow: '0 18px 44px rgba(220,38,38,0.12)', display: libReady ? 'block' : 'none', flexShrink: 0 }}>
                <div ref={stageRef} style={{ width: size.width, height: size.height, borderRadius: 14, overflow: 'hidden', background: '#FFFFFF' }} />
              </div>

              {libReady && !libError && (
                <div style={{ display: 'flex', gap: 10, marginTop: 12 }}>
                  <button onClick={handleToggle}
                    style={{ border: 'none', borderRadius: 11, padding: '8px 24px', fontSize: 14, fontWeight: 850, cursor: 'pointer', color: '#fff', background: 'linear-gradient(135deg, #F87171, #B91C1C)', boxShadow: '0 5px 14px rgba(185,28,28,0.28)' }}
                  >{playing ? '⏸ 暂停' : '▶ 播放'}</button>
                  <button onClick={handleReset}
                    style={{ borderRadius: 11, padding: '8px 20px', fontSize: 14, fontWeight: 750, cursor: 'pointer', background: '#fff', color: '#374151', border: '1.5px solid #F3D9D9' }}
                  >↺ 重置</button>
                </div>
              )}

              {previewError && (
                <div style={{ marginTop: 10, padding: '8px 12px', borderRadius: 10, background: '#FEE2E2', color: '#DC2626', fontSize: 12 }}>{previewError}</div>
              )}

              {caption.trim() && (
                <div style={{ marginTop: 10, fontSize: 14, color: '#6B7280', fontStyle: 'italic' }}>{caption.trim()}</div>
              )}

              <div style={{ marginTop: 12, fontSize: 12.5, color: '#D19999' }}>💡 生成后请先按播放测试，确认不会飞出画布或静止不动，再融入课件。</div>
            </div>

            <div style={{ width: 340, borderLeft: '1px solid ' + C.border, overflowY: 'auto', padding: '18px 18px 22px', flexShrink: 0, background: '#fff' }}>
              <div style={{ padding: '13px 14px', borderRadius: 15, background: 'linear-gradient(135deg, #FEF2F2, #FEE2E2)', border: '1px solid #FBD5D5', marginBottom: 16 }}>
                <div style={{ fontSize: 13.5, fontWeight: 850, color: '#B91C1C' }}>{isAI ? '✨ AI 定制' : '⚙️ 参数设置'}</div>
                <div style={{ fontSize: 11.8, color: '#B07070', lineHeight: 1.65, marginTop: 5 }}>
                  {isAI ? 'AI 生成的是 Matter.js setup 代码，仍使用当前播放/暂停/重置链路。' : '调参后按播放观察变化；融入课件后按钮同样可反复播放重置。'}
                </div>
              </div>

              {isAI ? (
                <PhysicsSceneAIPanel
                  mode={aiMode as 'adapt' | 'create'}
                  templateName={aiMode === 'adapt' ? activeTpl.name : undefined}
                  baseCode={aiBaseCode}
                  code={aiCode}
                  onCode={setAiCode}
                  onExit={exitAIMode}
                  busyExternal={inserting}
                  previewError={previewError}
                />
              ) : (
                <>
                  {activeTpl.params.map(pd => (
                    <div key={pd.key} style={{ marginBottom: 15 }}>
                      {pd.type === 'number' ? (
                        <>
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                            <span style={{ fontSize: 13, fontWeight: 650, color: C.textPrimary }}>{pd.label}</span>
                            <input
                              type="number"
                              value={Number(params[pd.key])}
                              min={pd.min} max={pd.max} step={pd.step}
                              onChange={e => {
                                const v = parseFloat(e.target.value)
                                if (!Number.isNaN(v)) setParam(pd.key, Math.min(pd.max ?? v, Math.max(pd.min ?? v, v)))
                              }}
                              style={{ width: 76, padding: '4px 9px', borderRadius: 999, border: '1.5px solid #FBD5D5', background: '#FEF2F2', color: '#B91C1C', fontWeight: 800, fontSize: 12.5, textAlign: 'center', outline: 'none' }}
                            />
                          </div>
                          <input
                            type="range" className="ph-range"
                            value={Number(params[pd.key])}
                            min={pd.min} max={pd.max} step={pd.step}
                            onChange={e => setParam(pd.key, parseFloat(e.target.value))}
                          />
                        </>
                      ) : (
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10 }}>
                          <span style={{ fontSize: 13, fontWeight: 650, color: C.textPrimary, lineHeight: 1.4 }}>{pd.label}</span>
                          <Toggle checked={Boolean(params[pd.key])} onChange={v => setParam(pd.key, v)} />
                        </div>
                      )}
                      {pd.hint && <div style={{ fontSize: 11, color: C.textMuted, marginTop: 5, lineHeight: 1.5 }}>{pd.hint}</div>}
                    </div>
                  ))}
                </>
              )}

              <div style={{ borderTop: '1px dashed #FBD5D5', margin: '18px 0' }} />

              <div style={{ marginBottom: 15 }}>
                <div style={{ fontSize: 13, fontWeight: 650, color: C.textPrimary, marginBottom: 8 }}>画布尺寸</div>
                <div style={{ display: 'flex', borderRadius: 12, overflow: 'hidden', border: '1.5px solid #FBD5D5' }}>
                  {PHYSICS_SIZE_PRESETS.map((s, i) => {
                    const on = i === sizeIdx
                    return (
                      <button key={s.label} onClick={() => setSizeIdx(i)}
                        style={{
                          flex: 1, padding: '8px 0', border: 'none', cursor: 'pointer',
                          borderRight: i < PHYSICS_SIZE_PRESETS.length - 1 ? '1px solid #FBD5D5' : 'none',
                          background: on ? 'linear-gradient(135deg, #F87171, #B91C1C)' : '#fff',
                          color: on ? '#fff' : C.textSecondary,
                        }}
                      >
                        <div style={{ fontSize: 12.5, fontWeight: on ? 850 : 650 }}>{['小', '标准', '大'][i]}</div>
                        <div style={{ fontSize: 10, opacity: 0.86, marginTop: 1 }}>{s.width}×{s.height}</div>
                      </button>
                    )
                  })}
                </div>
              </div>

              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
                <span style={{ fontSize: 13, fontWeight: 650, color: C.textPrimary }}>融入后自动播放</span>
                <Toggle checked={autoplay} onChange={setAutoplay} />
              </div>
              <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 15, lineHeight: 1.5 }}>
                建议关闭：课件页打开时定格初始状态，讲到位后老师按播放。
              </div>

              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
                <span style={{ fontSize: 13, fontWeight: 650, color: C.textPrimary }}>显示速度矢量</span>
                <Toggle checked={showVelocity} onChange={setShowVelocity} />
              </div>
              <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 15, lineHeight: 1.5 }}>
                讲速度方向变化时打开，如抛体、碰撞、斜面运动。
              </div>

              <div style={{ marginBottom: 8 }}>
                <div style={{ fontSize: 13, fontWeight: 650, color: C.textPrimary, marginBottom: 8 }}>标注文字（可选）</div>
                <input
                  type="text"
                  value={caption}
                  onChange={e => setCaption(e.target.value)}
                  placeholder="如：观察轻重双球是否同时落地"
                  maxLength={60}
                  style={{ width: '100%', boxSizing: 'border-box', padding: '9px 12px', borderRadius: 12, border: '1.5px solid #F8E0E0', fontSize: 13, outline: 'none' }}
                />
              </div>
            </div>
          </div>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '14px 24px', borderTop: '1px solid ' + C.border, flexShrink: 0, background: '#FDFAFA' }}>
          <span style={{ fontSize: 12.5, color: C.textMuted }}>
            {isAI ? 'AI 力学场景使用 Matter.js setup 代码生成，融入后仍可离线运行、课堂反复播放。' : '基于 Matter.js 开源物理引擎，生成的场景可离线运行、课堂可反复播放。'}
          </span>
          <div style={{ marginLeft: 'auto', display: 'flex', gap: 10 }}>
            <button
              onClick={() => { if (!inserting) onClose() }}
              disabled={inserting}
              style={{ padding: '10px 24px', borderRadius: 13, border: '1.5px solid #F8E0E0', background: '#fff', color: C.textSecondary, fontSize: 14, fontWeight: 650, cursor: inserting ? 'not-allowed' : 'pointer' }}
            >取消</button>
            <button
              onClick={handleInsert}
              disabled={insertDisabled}
              style={{
                padding: '10px 28px', borderRadius: 13, border: 'none', fontSize: 14, fontWeight: 850,
                background: insertDisabled ? '#FCA5A5' : 'linear-gradient(135deg, #F87171, #B91C1C)',
                boxShadow: insertDisabled ? 'none' : '0 8px 22px rgba(185,28,28,0.35)',
                color: '#fff', cursor: insertDisabled ? 'not-allowed' : 'pointer',
              }}
            >{inserting ? '⏳ AI 融入中…' : '🎯 融入第 ' + pageNum + ' 页'}</button>
          </div>
        </div>
      </div>
    </div>
  )
}
