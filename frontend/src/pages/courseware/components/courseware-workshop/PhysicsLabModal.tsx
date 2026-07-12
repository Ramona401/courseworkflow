/**
 * PhysicsLabModal.tsx — 物理实验室编辑器弹窗
 *
 * 第6批B：
 *   - 接入 ExperimentAIPanel；
 *   - 支持 AI 新建物理实验；
 *   - 支持基于现有模板 AI 改编；
 *   - AI 结果仍走原有预览与融入链路，保留底部课堂控制条。
 */

import { useEffect, useMemo, useState } from 'react'
import { C } from './workshopConstants'
import {
  PHYSICS_LAB_DEFAULT_SIZE_INDEX,
  PHYSICS_LAB_SIZE_PRESETS,
  buildDefaultPhysicsLabParams,
  buildPhysicsLabLayoutOverride,
  buildPhysicsLabRefineInstruction,
} from './physicsLabUtils'
import type { PhysicsLabParamValue, PhysicsLabTemplate } from './physicsLabUtils'
import { PHYSICS_LAB_TEMPLATES, getPhysicsLabGroups } from './physicsLabTemplates'
import ExperimentAIPanel from './ExperimentAIPanel'

interface Props {
  pageNum: number
  onInsert: (instruction: string) => void
  onClose: () => void
  inserting?: boolean
}

const MODAL_CSS = [
  '.plab-card { transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s; }',
  '.plab-card:hover { border-color:#38BDF8 !important; box-shadow:0 8px 24px rgba(14,165,233,0.16); transform:translateY(-1px); }',
  '.plab-ai-btn { opacity:0; transition: opacity 0.15s; }',
  '.plab-card:hover .plab-ai-btn { opacity:1; }',
  '.plab-range { -webkit-appearance:none; appearance:none; width:100%; height:6px; border-radius:3px; background:#E0F2FE; outline:none; cursor:pointer; }',
  '.plab-range::-webkit-slider-thumb { -webkit-appearance:none; appearance:none; width:18px; height:18px; border-radius:50%; background:linear-gradient(135deg,#38BDF8,#0284C7); border:2.5px solid #fff; box-shadow:0 1px 4px rgba(2,132,199,0.45); cursor:pointer; }',
  '.plab-range::-moz-range-thumb { width:18px; height:18px; border-radius:50%; background:#0284C7; border:2.5px solid #fff; box-shadow:0 1px 4px rgba(2,132,199,0.45); cursor:pointer; }',
].join('\n')

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

function replaceRootId(html: string, rootId: string): string {
  return html.split('__ROOT_ID__').join(rootId)
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <div
      onClick={() => onChange(!checked)}
      style={{
        width: 40, height: 22, borderRadius: 11, flexShrink: 0, cursor: 'pointer',
        background: checked ? 'linear-gradient(135deg, #38BDF8, #0284C7)' : '#D1D5DB',
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

export default function PhysicsLabModal({ pageNum, onInsert, onClose, inserting }: Props) {
  const [activeTplId, setActiveTplId] = useState(PHYSICS_LAB_TEMPLATES[0].id)
  const activeTpl = PHYSICS_LAB_TEMPLATES.find(t => t.id === activeTplId) || PHYSICS_LAB_TEMPLATES[0]

  const [params, setParams] = useState<Record<string, PhysicsLabParamValue>>(buildDefaultPhysicsLabParams(activeTpl))
  const [sizeIdx, setSizeIdx] = useState(PHYSICS_LAB_DEFAULT_SIZE_INDEX)
  const [caption, setCaption] = useState('')
  const [positionHint, setPositionHint] = useState('')

  const [aiMode, setAiMode] = useState<'adapt' | 'create' | null>(null)
  const [aiCode, setAiCode] = useState('')
  const [aiBaseCode, setAiBaseCode] = useState('')

  const size = PHYSICS_LAB_SIZE_PRESETS[sizeIdx] || PHYSICS_LAB_SIZE_PRESETS[PHYSICS_LAB_DEFAULT_SIZE_INDEX]
  const isAI = aiMode !== null

  // 右侧栏参数用于设置初始状态；预览参数做轻量防抖，避免 iframe srcDoc 每一格滑动都重载闪烁。
  const [previewParams, setPreviewParams] = useState<Record<string, PhysicsLabParamValue>>(params)
  useEffect(() => {
    const timer = window.setTimeout(() => setPreviewParams(params), 220)
    return () => window.clearTimeout(timer)
  }, [params])

  const previewDoc = useMemo(() => {
    const rootId = 'physics-lab-preview-root'
    let html = ''

    if (isAI) {
      if (aiCode.trim()) {
        html = replaceRootId(aiCode, rootId) + buildPhysicsLabLayoutOverride(rootId)
      } else if (aiMode === 'adapt') {
        html = activeTpl.buildHTML(previewParams, rootId) + buildPhysicsLabLayoutOverride(rootId)
      } else {
        html = '<div style="width:100%;height:100%;box-sizing:border-box;border:1px dashed #BAE6FD;border-radius:16px;background:linear-gradient(135deg,#F0F9FF,#FFFFFF);display:flex;flex-direction:column;align-items:center;justify-content:center;font-family:-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;color:#075985;">'
          + '<div style="font-size:38px;margin-bottom:12px;">✨</div>'
          + '<div style="font-size:18px;font-weight:800;">AI 新建物理实验</div>'
          + '<div style="font-size:13px;color:#64748B;margin-top:8px;">在右侧描述实验，或上传题目/装置图片</div>'
          + '</div>'
      }
    } else {
      html = activeTpl.buildHTML(previewParams, rootId) + buildPhysicsLabLayoutOverride(rootId)
    }

    const cap = caption.trim()
      ? '<div style="text-align:center;font-size:14px;color:#6B7280;margin-top:8px;font-style:italic;">' + escapeHtml(caption.trim()) + '</div>'
      : ''

    return '<!doctype html><html><head><meta charset="utf-8"><style>html,body{margin:0;padding:0;background:transparent;overflow:hidden;}body{display:flex;flex-direction:column;align-items:center;justify-content:flex-start;font-family:-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;}</style></head><body>'
      + '<div style="width:' + size.width + 'px;height:' + size.height + 'px;">'
      + html
      + '</div>'
      + cap
      + '</body></html>'
  }, [isAI, aiMode, aiCode, activeTpl, previewParams, size.width, size.height, caption])

  const handlePickTemplate = (tpl: PhysicsLabTemplate) => {
    const nextParams = buildDefaultPhysicsLabParams(tpl)
    setActiveTplId(tpl.id)
    setParams(nextParams)
    setPreviewParams(nextParams)
    setAiMode(null)
    setAiCode('')
    setAiBaseCode('')
  }

  const enterAdaptMode = (tpl: PhysicsLabTemplate) => {
    const p = tpl.id === activeTplId ? params : buildDefaultPhysicsLabParams(tpl)
    setActiveTplId(tpl.id)
    setParams(p)
    setPreviewParams(p)
    setAiBaseCode(tpl.buildHTML(p, '__ROOT_ID__'))
    setAiCode('')
    setAiMode('adapt')
  }

  const enterCreateMode = () => {
    setAiBaseCode('')
    setAiCode('')
    setAiMode('create')
  }

  const exitAIMode = () => {
    setAiMode(null)
    setAiCode('')
    setAiBaseCode('')
  }

  const setParam = (key: string, v: PhysicsLabParamValue) => {
    setParams(prev => Object.assign({}, prev, { [key]: v }))
  }

  const handleInsert = () => {
    if (inserting) return

    let tplForInsert: PhysicsLabTemplate = activeTpl
    let paramsForInsert: Record<string, PhysicsLabParamValue> = params

    if (isAI) {
      if (!aiCode.trim()) return
      const frozenHTML = aiCode
      tplForInsert = {
        id: 'ai-physics-lab',
        group: '✨ AI定制',
        name: aiMode === 'adapt' ? activeTpl.name + '·AI改编' : 'AI自定义物理实验',
        emoji: '✨',
        desc: 'AI 定制的互动物理实验',
        params: [],
        buildHTML: (_params, rootId) => replaceRootId(frozenHTML, rootId),
      }
      paramsForInsert = {}
    }

    const instruction = buildPhysicsLabRefineInstruction({
      lab: {
        template: tplForInsert,
        params: paramsForInsert,
        width: size.width,
        height: size.height,
        caption: caption.trim() || undefined,
      },
      positionHint: positionHint.trim() || undefined,
    })
    onInsert(instruction)
  }

  const insertDisabled = inserting || (isAI && !aiCode.trim())

  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(8,20,32,0.62)', backdropFilter: 'blur(5px)', zIndex: 99993, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={() => { if (!inserting) onClose() }}
    >
      <style>{MODAL_CSS}</style>
      <div
        style={{ width: 'min(1520px, 98vw)', height: 'min(900px, 96vh)', background: '#fff', borderRadius: 24, display: 'flex', flexDirection: 'column', overflow: 'hidden', boxShadow: '0 34px 88px rgba(0,0,0,0.38)' }}
        onClick={e => e.stopPropagation()}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '16px 24px', background: 'linear-gradient(135deg, #38BDF8 0%, #0284C7 52%, #075985 100%)', flexShrink: 0 }}>
          <span style={{ width: 48, height: 48, borderRadius: 16, background: 'rgba(255,255,255,0.18)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 25, flexShrink: 0 }}>🧪</span>
          <div style={{ minWidth: 0 }}>
            <div style={{ fontSize: 19, fontWeight: 850, color: '#fff', letterSpacing: 0.3 }}>物理实验室</div>
            <div style={{ fontSize: 12.5, color: 'rgba(255,255,255,0.82)', marginTop: 2 }}>模板点选 · AI 改编/新建 · 底部课堂控制条 · 纯HTML离线运行</div>
          </div>
          {isAI && (
            <span style={{ padding: '6px 15px', borderRadius: 999, background: 'rgba(255,255,255,0.24)', border: '1px solid rgba(255,255,255,0.38)', color: '#fff', fontSize: 12.5, fontWeight: 800, whiteSpace: 'nowrap' }}>
              ✨ AI 定制中
            </span>
          )}
          <span style={{ marginLeft: 8, padding: '6px 15px', borderRadius: 999, background: 'rgba(255,255,255,0.18)', border: '1px solid rgba(255,255,255,0.32)', color: '#fff', fontSize: 12.5, fontWeight: 800, whiteSpace: 'nowrap' }}>
            将融入:第 {pageNum} 页
          </span>
          <button onClick={() => { if (!inserting) onClose() }}
            style={{ marginLeft: 'auto', border: 'none', background: 'rgba(255,255,255,0.15)', width: 36, height: 36, borderRadius: 12, fontSize: 18, cursor: 'pointer', color: '#fff', flexShrink: 0 }}
            title="关闭"
          >✕</button>
        </div>

        <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
          <div style={{ width: 300, borderRight: '1px solid ' + C.border, display: 'flex', flexDirection: 'column', flexShrink: 0, background: 'linear-gradient(180deg,#F8FCFF,#F3FAFE)' }}>
            <div style={{ padding: '14px 14px 12px', borderBottom: '1px solid #D9EEF8', flexShrink: 0 }}>
              <div style={{ fontSize: 13, fontWeight: 850, color: '#075985' }}>🧪 选择物理实验</div>
              <div style={{ fontSize: 11.5, color: C.textMuted, marginTop: 6, lineHeight: 1.55 }}>
                点模板直接使用；点“改编”可让 AI 在现有结构上变种。
              </div>
              <button
                onClick={enterCreateMode}
                style={{
                  width: '100%', marginTop: 10, padding: '9px 0', borderRadius: 12, cursor: 'pointer',
                  border: '1.5px dashed ' + (aiMode === 'create' ? '#0284C7' : '#BAE6FD'),
                  background: aiMode === 'create' ? 'linear-gradient(135deg,#E0F2FE,#F8FCFF)' : '#fff',
                  color: '#0284C7', fontSize: 12.5, fontWeight: 800,
                }}
              >✨ AI 新建物理实验</button>
            </div>

            <div style={{ flex: 1, overflowY: 'auto', padding: '12px 12px 16px' }}>
              {getPhysicsLabGroups().map(g => (
                <div key={g.group} style={{ marginBottom: 14 }}>
                  <div style={{ fontSize: 11.5, fontWeight: 850, color: '#4688A6', padding: '3px 6px 7px', letterSpacing: 0.4 }}>{g.group}</div>
                  {g.items.map(tpl => {
                    const active = tpl.id === activeTplId && !isAI
                    return (
                      <div key={tpl.id} className="plab-card" onClick={() => handlePickTemplate(tpl)}
                        style={{
                          display: 'flex', gap: 11, alignItems: 'flex-start', padding: '11px 12px 30px', borderRadius: 15, cursor: 'pointer', marginBottom: 8,
                          background: active ? 'linear-gradient(135deg, #E0F2FE, #F8FCFF)' : 'rgba(255,255,255,0.92)',
                          border: '1.5px solid ' + (active ? '#0284C7' : '#E3F2F8'),
                          boxShadow: active ? '0 8px 22px rgba(14,165,233,0.18)' : '0 2px 8px rgba(14,54,78,0.04)',
                          position: 'relative',
                        }}
                      >
                        <span style={{ width: 36, height: 36, borderRadius: 12, background: active ? '#fff' : '#F0F9FF', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 18, flexShrink: 0 }}>{tpl.emoji}</span>
                        <div style={{ minWidth: 0 }}>
                          <div style={{ fontSize: 13.5, fontWeight: active ? 850 : 650, color: active ? '#075985' : C.textPrimary, lineHeight: 1.35 }}>{tpl.name}</div>
                          <div style={{ fontSize: 11.2, color: C.textMuted, marginTop: 4, lineHeight: 1.5 }}>{tpl.desc}</div>
                        </div>
                        <button
                          className="plab-ai-btn"
                          onClick={e => { e.stopPropagation(); enterAdaptMode(tpl) }}
                          title="以此模板为底稿，用自然语言改编"
                          style={{
                            position: 'absolute', right: 9, bottom: 7, padding: '4px 10px', borderRadius: 999, fontSize: 10.5, fontWeight: 800, cursor: 'pointer',
                            border: '1px solid #BAE6FD', background: '#fff', color: '#0284C7',
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
            <div style={{ flex: 1, overflow: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column', alignItems: 'center', minWidth: 0, background: 'radial-gradient(circle at 50% 0%,#FFFFFF 0%,#F7FBFF 58%,#EFF8FE 100%)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 9, marginBottom: 14, flexWrap: 'wrap', justifyContent: 'center' }}>
                <span style={{ fontSize: 20 }}>{isAI ? '✨' : activeTpl.emoji}</span>
                <span style={{ fontSize: 16, fontWeight: 850, color: C.textPrimary }}>{isAI ? (aiMode === 'adapt' ? activeTpl.name + ' · AI改编' : 'AI 新物理实验') : activeTpl.name}</span>
                <span style={{ fontSize: 12.5, color: C.textMuted }}>{isAI ? (aiCode ? '当前显示 AI 生成结果' : '右侧描述后生成') : activeTpl.desc}</span>
              </div>
              <div style={{ padding: 14, background: '#fff', borderRadius: 22, border: '1px solid #D9EEF8', boxShadow: '0 18px 44px rgba(14,165,233,0.12)', flexShrink: 0 }}>
                <iframe
                  title="物理实验预览"
                  sandbox="allow-scripts"
                  srcDoc={previewDoc}
                  style={{ width: size.width, height: size.height + (caption.trim() ? 34 : 0), border: 'none', display: 'block', borderRadius: 14, background: 'transparent' }}
                />
              </div>
              <div style={{ marginTop: 12, fontSize: 12.5, color: '#6EAAC6' }}>💡 生成后请先在这里测试滑杆、按钮和读数，再融入课件。</div>
            </div>

            <div style={{ width: 340, borderLeft: '1px solid ' + C.border, overflowY: 'auto', padding: '18px 18px 22px', flexShrink: 0, background: '#fff' }}>
              <div style={{ padding: '13px 14px', borderRadius: 15, background: 'linear-gradient(135deg, #E0F2FE, #F8FCFF)', border: '1px solid #BAE6FD', marginBottom: 16 }}>
                <div style={{ fontSize: 13.5, fontWeight: 850, color: '#075985' }}>{isAI ? '✨ AI 定制' : '⚙️ 初始参数 / 融入设置'}</div>
                <div style={{ fontSize: 11.8, color: '#4F839B', lineHeight: 1.65, marginTop: 5 }}>
                  {isAI ? 'AI 生成的是完整互动实验组件，仍会保留底部课堂控制条。' : '这里设置的是课件中的初始状态；融入后继续使用组件底部课堂控制条。'}
                </div>
              </div>

              {isAI ? (
                <ExperimentAIPanel
                  target="physics_lab"
                  mode={aiMode as 'adapt' | 'create'}
                  templateName={aiMode === 'adapt' ? activeTpl.name : undefined}
                  baseCode={aiBaseCode}
                  code={aiCode}
                  onCode={setAiCode}
                  onExit={exitAIMode}
                  busyExternal={inserting}
                />
              ) : (
                <>
                  {activeTpl.params.map(pd => (
                    <div key={pd.key} style={{ marginBottom: 16 }}>
                      {pd.type === 'number' ? (
                        <>
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 7 }}>
                            <span style={{ fontSize: 13, fontWeight: 650, color: C.textPrimary }}>{pd.label}</span>
                            <input
                              type="number"
                              value={Number(params[pd.key])}
                              min={pd.min} max={pd.max} step={pd.step}
                              onChange={e => {
                                const v = parseFloat(e.target.value)
                                if (!Number.isNaN(v)) setParam(pd.key, Math.min(pd.max ?? v, Math.max(pd.min ?? v, v)))
                              }}
                              style={{ width: 76, padding: '4px 9px', borderRadius: 999, border: '1.5px solid #BAE6FD', background: '#F0F9FF', color: '#075985', fontWeight: 800, fontSize: 12.5, textAlign: 'center', outline: 'none' }}
                            />
                          </div>
                          <input type="range" className="plab-range" value={Number(params[pd.key])} min={pd.min} max={pd.max} step={pd.step} onChange={e => setParam(pd.key, parseFloat(e.target.value))} />
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

              <div style={{ borderTop: '1px dashed #BAE6FD', margin: '18px 0' }} />

              <div style={{ marginBottom: 16 }}>
                <div style={{ fontSize: 13, fontWeight: 650, color: C.textPrimary, marginBottom: 8 }}>实验尺寸</div>
                <div style={{ display: 'flex', borderRadius: 12, overflow: 'hidden', border: '1.5px solid #BAE6FD' }}>
                  {PHYSICS_LAB_SIZE_PRESETS.map((s, i) => {
                    const on = i === sizeIdx
                    return (
                      <button key={s.label} onClick={() => setSizeIdx(i)}
                        style={{
                          flex: 1, padding: '8px 0', border: 'none', cursor: 'pointer',
                          borderRight: i < PHYSICS_LAB_SIZE_PRESETS.length - 1 ? '1px solid #BAE6FD' : 'none',
                          background: on ? 'linear-gradient(135deg, #38BDF8, #0284C7)' : '#fff',
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

              <div style={{ marginBottom: 16 }}>
                <div style={{ fontSize: 13, fontWeight: 650, color: C.textPrimary, marginBottom: 8 }}>标注文字（可选）</div>
                <input
                  type="text"
                  value={caption}
                  onChange={e => setCaption(e.target.value)}
                  placeholder="如：拖动电阻观察电流如何变化"
                  maxLength={60}
                  style={{ width: '100%', boxSizing: 'border-box', padding: '9px 12px', borderRadius: 12, border: '1.5px solid #D5EDF8', fontSize: 13, outline: 'none' }}
                />
              </div>

              <div style={{ marginBottom: 8 }}>
                <div style={{ fontSize: 13, fontWeight: 650, color: C.textPrimary, marginBottom: 8 }}>融入位置偏好（可选）</div>
                <input
                  type="text"
                  value={positionHint}
                  onChange={e => setPositionHint(e.target.value)}
                  placeholder="如：放在右侧、替换原来的实验图"
                  maxLength={80}
                  style={{ width: '100%', boxSizing: 'border-box', padding: '9px 12px', borderRadius: 12, border: '1.5px solid #D5EDF8', fontSize: 13, outline: 'none' }}
                />
              </div>
            </div>
          </div>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '14px 24px', borderTop: '1px solid ' + C.border, flexShrink: 0, background: '#F8FCFF' }}>
          <span style={{ fontSize: 12.5, color: C.textMuted }}>
            {isAI ? 'AI 实验组件生成后仍是纯 HTML/SVG/Canvas，可离线运行。' : '纯 HTML/SVG/Canvas 实验组件，无外部依赖；融入后底部课堂控制条会保留。'}
          </span>
          <div style={{ marginLeft: 'auto', display: 'flex', gap: 10 }}>
            <button onClick={() => { if (!inserting) onClose() }} disabled={inserting}
              style={{ padding: '10px 24px', borderRadius: 13, border: '1.5px solid #D5EDF8', background: '#fff', color: C.textSecondary, fontSize: 14, fontWeight: 650, cursor: inserting ? 'not-allowed' : 'pointer' }}
            >取消</button>
            <button onClick={handleInsert} disabled={insertDisabled}
              style={{
                padding: '10px 28px', borderRadius: 13, border: 'none', fontSize: 14, fontWeight: 850,
                background: insertDisabled ? '#7DD3FC' : 'linear-gradient(135deg, #38BDF8, #0284C7)',
                boxShadow: insertDisabled ? 'none' : '0 8px 22px rgba(2,132,199,0.35)',
                color: '#fff', cursor: insertDisabled ? 'not-allowed' : 'pointer',
              }}
            >{inserting ? '⏳ AI 融入中…' : '🧪 融入第 ' + pageNum + ' 页'}</button>
          </div>
        </div>
      </div>
    </div>
  )
}
