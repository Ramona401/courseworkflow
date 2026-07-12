/**
 * MoleculeLabModal.tsx — 课件分子实验室编辑器弹窗(批次2首发,2026-07-09)
 *
 * 双模式设计(顶部左栏分段切换):
 *   🧊 3D 立体分子(3Dmol.js)——模板点选 14 个 K12 常见分子/晶体,
 *      可选球棍/空间填充/棍状三种样式、自动旋转、元素标签、背景色;
 *      预览与课件内均可鼠标拖拽旋转、滚轮缩放(3D 观察是核心教学价值);
 *   ✏️ 2D 结构式(smiles-drawer)——16 个模板 + 开放 SMILES 输入框
 *      (SMILES 串很短,老师可直接粘贴教材/网上查到的串,渲染平面结构式)。
 *
 * 单一真相源(对齐 mathGraphUtils 架构约定):
 *   3D 模式:模板 buildXYZ() 的同一份 xyz 字符串,预览侧直接 addModel,
 *   融入侧经 generateMolecule3DEmbed 内联进自包含 HTML——零外部数据请求;
 *   2D 模式:同一 SMILES 串双侧共用。预览所见即课件所得。
 *
 * 库加载:3Dmol 2.5.5 / smiles-drawer 2.4.1 均自托管,按当前模式懒加载
 *   (幂等注入 + 轮询就绪),不进主包。
 *
 * 交互范式对齐 MathGraphModal / FormulaEditorModal:
 *   props 签名 {pageNum, onInsert(instruction), onClose, inserting?} 完全一致,
 *   由 SubjectToolsPanel 的 makeInsertHandler 工厂统一驱动融入。
 *   视觉主题:绿色渐变(学科工具宫格分子卡主题色 #059669 系)。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/MoleculeLabModal.tsx
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import { C } from './workshopConstants'
import {
  MOL3D_JS_CDN, SMILES_DRAWER_JS_CDN,
  MOLECULE_3D_DEFAULTS, MOLECULE_SIZE_PRESETS, MOLECULE_BG_PRESETS, MOLECULE_STYLE_OPTIONS,
  buildMoleculeRefineInstruction,
} from './moleculeUtils'
import type { Molecule3DTemplate, MoleculeStyleKey } from './moleculeUtils'
import { MOLECULE_3D_TEMPLATES, MOLECULE_2D_TEMPLATES, getMolecule3DGroups, getMolecule2DGroups } from './moleculeTemplates'

// 3Dmol 挂在 window.$3Dmol / smiles-drawer 挂在 window.SmilesDrawer(全局脚本非模块,TS 侧 any 处理)
declare global {
  interface Window { $3Dmol?: any; SmilesDrawer?: any }
}

// ==================== 类型 ====================

interface Props {
  /** 目标课件页号(仅展示用,融入逻辑在父组件) */
  pageNum: number
  /** 融入回调:把拼好的微调指令交给父组件调 refinePage */
  onInsert: (instruction: string) => void
  /** 关闭弹窗 */
  onClose: () => void
  /** 父组件融入运行态(禁用按钮防重复提交) */
  inserting?: boolean
}

/** 弹窗模式 */
type LabMode = '3d' | '2d'

// ==================== 弹窗专用 CSS(骨架屏动画 / 卡片悬浮;类名 ml- 前缀限定作用域) ====================

const MODAL_CSS = [
  '@keyframes ml-shimmer { 0% { background-position: -420px 0; } 100% { background-position: 420px 0; } }',
  '.ml-skeleton { background: linear-gradient(90deg, #F3F4F6 25%, #D1FAE5 40%, #F3F4F6 60%); background-size: 840px 100%; animation: ml-shimmer 1.3s infinite linear; }',
  '.ml-card { transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s; }',
  '.ml-card:hover { border-color: #6EE7B7 !important; box-shadow: 0 4px 14px rgba(5,150,105,0.14); transform: translateY(-1px); }',
].join('\n')

/**
 * 3D 样式 → 3Dmol setStyle 参数对象(预览侧直接使用)。
 * ⚠ 必须与 moleculeUtils.ts 的 MOLECULE_STYLE_JS 保持逐字段同步,
 *   否则出现"预览一个样、课件另一个样"的双实现漂移。
 */
const STYLE_OBJS: Record<MoleculeStyleKey, any> = {
  ballstick: { stick: { radius: 0.14, colorscheme: 'Jmol' }, sphere: { scale: 0.28, colorscheme: 'Jmol' } },
  spacefill: { sphere: { colorscheme: 'Jmol' } },
  stick: { stick: { radius: 0.09, colorscheme: 'Jmol' } },
}

// ==================== 库加载(模块级幂等,多次开弹窗不重复注入;范式同 MathGraphModal) ====================

/** 通用脚本加载器:按 src 特征去重注入 + 轮询全局就绪 */
function loadScriptOnce(src: string, fileKey: string, isReady: () => boolean): Promise<void> {
  return new Promise((resolve, reject) => {
    if (isReady()) { resolve(); return }
    const existing = document.querySelector('script[src*="' + fileKey + '"]')
    if (existing) {
      let tries = 0
      const timer = window.setInterval(() => {
        tries++
        if (isReady()) { window.clearInterval(timer); resolve() }
        else if (tries > 100) { window.clearInterval(timer); reject(new Error('库加载超时: ' + fileKey)) }
      }, 100)
      return
    }
    const s = document.createElement('script')
    s.src = src
    s.onload = () => { isReady() ? resolve() : reject(new Error('库加载后未就绪: ' + fileKey)) }
    s.onerror = () => reject(new Error('脚本加载失败(网络或自托管地址问题): ' + fileKey))
    document.head.appendChild(s)
  })
}

const load3DmolLib = () => loadScriptOnce(MOL3D_JS_CDN, '3Dmol-min.js', () => !!window.$3Dmol)
const loadSmilesDrawerLib = () => loadScriptOnce(SMILES_DRAWER_JS_CDN, 'smiles-drawer.min.js', () => !!window.SmilesDrawer)

// ==================== 辅助 ====================

/** 判断背景是否为深色(元素标签取反色;逻辑同 moleculeUtils.isDarkBackground) */
function isDarkBg(hex: string): boolean {
  const m = /^#?([0-9a-fA-F]{6})$/.exec(hex.trim())
  if (!m) return false
  const v = parseInt(m[1], 16)
  const r = (v >> 16) & 255, g = (v >> 8) & 255, b = v & 255
  return (r * 299 + g * 587 + b * 114) / 1000 < 128
}

/** 开关组件(绿色主题 Toggle,范式同 MathGraphModal.Toggle) */
function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <div
      onClick={() => onChange(!checked)}
      style={{
        width: 40, height: 22, borderRadius: 11, flexShrink: 0, cursor: 'pointer',
        background: checked ? 'linear-gradient(135deg, #10B981, #047857)' : '#D1D5DB',
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

// ==================== 组件 ====================

export default function MoleculeLabModal({ pageNum, onInsert, onClose, inserting }: Props) {
  // ---- 状态 ----
  const [mode, setMode] = useState<LabMode>('3d')          // 当前模式
  const [lib3dReady, setLib3dReady] = useState(false)      // 3Dmol 就绪
  const [lib2dReady, setLib2dReady] = useState(false)      // smiles-drawer 就绪
  const [libError, setLibError] = useState('')             // 库加载失败信息
  const [previewError, setPreviewError] = useState('')     // 预览渲染错误
  // 3D 模式状态
  const [active3DId, setActive3DId] = useState(MOLECULE_3D_TEMPLATES[0].id)
  const [styleKey, setStyleKey] = useState<MoleculeStyleKey>(MOLECULE_3D_DEFAULTS.styleKey)
  const [spin, setSpin] = useState<boolean>(MOLECULE_3D_DEFAULTS.spin)
  const [showLabels, setShowLabels] = useState<boolean>(MOLECULE_3D_DEFAULTS.showLabels)
  const [background, setBackground] = useState<string>(MOLECULE_3D_DEFAULTS.background)
  // 2D 模式状态(smiles 为单一真相;点模板填入,手改则显示名退化为"自定义分子")
  const [smiles, setSmiles] = useState(MOLECULE_2D_TEMPLATES[0].smiles)
  const [smilesName, setSmilesName] = useState(MOLECULE_2D_TEMPLATES[0].name + ' ' + MOLECULE_2D_TEMPLATES[0].formula)
  const [active2DId, setActive2DId] = useState(MOLECULE_2D_TEMPLATES[0].id)
  // 共用状态
  const [sizeIdx, setSizeIdx] = useState(1)                // 尺寸方案(默认"标准")
  const [caption, setCaption] = useState('')               // 可选标注

  // ---- refs ----
  const host3dRef = useRef<HTMLDivElement | null>(null)    // 3D 画布容器
  const viewerRef = useRef<any>(null)                      // 3Dmol viewer 实例
  const debounceRef = useRef<number | null>(null)          // 预览防抖定时器

  const active3D: Molecule3DTemplate = MOLECULE_3D_TEMPLATES.find(t => t.id === active3DId) || MOLECULE_3D_TEMPLATES[0]
  const size = MOLECULE_SIZE_PRESETS[sizeIdx] || MOLECULE_SIZE_PRESETS[1]
  const libReady = mode === '3d' ? lib3dReady : lib2dReady

  // ---- 按当前模式懒加载对应库 ----
  useEffect(() => {
    let cancelled = false
    setLibError('')
    const task = mode === '3d'
      ? (lib3dReady ? null : load3DmolLib().then(() => { if (!cancelled) setLib3dReady(true) }))
      : (lib2dReady ? null : loadSmilesDrawerLib().then(() => { if (!cancelled) setLib2dReady(true) }))
    if (task) task.catch(e => { if (!cancelled) setLibError(e instanceof Error ? e.message : '库加载失败') })
    return () => { cancelled = true }
  }, [mode, lib3dReady, lib2dReady])

  // ---- 3D 预览渲染:清空容器 → createViewer → addModel(同源xyz) → setStyle → render ----
  const render3D = useCallback(() => {
    if (!lib3dReady || !host3dRef.current || !window.$3Dmol) return
    setPreviewError('')
    const host = host3dRef.current
    host.innerHTML = ''      // 丢弃旧 viewer 的 canvas,直接重建(3Dmol 无显式销毁 API)
    viewerRef.current = null
    try {
      const viewer = window.$3Dmol.createViewer(host, { backgroundColor: background, antialias: true })
      viewerRef.current = viewer
      /* 单一真相源:与课件 embed 完全同一份 xyz 数据 */
      viewer.addModel(active3D.buildXYZ(), 'xyz')
      viewer.setStyle({}, STYLE_OBJS[styleKey])
      if (showLabels) {
        viewer.addPropertyLabels('elem', {}, {
          fontColor: isDarkBg(background) ? '#E5E7EB' : '#374151',
          fontSize: 12, showBackground: false, alignment: 'center',
        })
      }
      viewer.zoomTo()
      viewer.render()
      if (spin) viewer.spin('y', 0.6)
    } catch (e) {
      setPreviewError('3D 预览渲染失败: ' + (e instanceof Error ? e.message : String(e)))
    }
  }, [lib3dReady, active3D, styleKey, spin, showLabels, background])

  // ---- 2D 预览渲染:parse(SMILES) → draw 到固定 id 的 canvas ----
  const render2D = useCallback(() => {
    if (!lib2dReady || !window.SmilesDrawer) return
    const cv = document.getElementById('molecule-modal-2d-canvas') as HTMLCanvasElement | null
    if (!cv) return
    setPreviewError('')
    try {
      const drawer = new window.SmilesDrawer.Drawer({ width: size.width, height: size.height, bondThickness: 1.2 })
      window.SmilesDrawer.parse(smiles.trim(), (tree: any) => {
        drawer.draw(tree, 'molecule-modal-2d-canvas', 'light', false)
      }, () => {
        setPreviewError('SMILES 解析失败,请检查输入(如苯环是 c1ccccc1,乙酸是 CC(=O)O)')
      })
    } catch (e) {
      setPreviewError('结构式渲染失败: ' + (e instanceof Error ? e.message : String(e)))
    }
  }, [lib2dReady, smiles, size.width, size.height])

  // ---- 配置变化:300ms 防抖重渲染当前模式预览 ----
  useEffect(() => {
    if (!libReady) return
    if (debounceRef.current) window.clearTimeout(debounceRef.current)
    debounceRef.current = window.setTimeout(() => { mode === '3d' ? render3D() : render2D() }, 300)
    return () => { if (debounceRef.current) window.clearTimeout(debounceRef.current) }
  }, [libReady, mode, render3D, render2D])

  // ---- 卸载:丢弃 viewer 引用(容器随组件树销毁) ----
  useEffect(() => () => { viewerRef.current = null }, [])

  // ---- 2D:点选模板 ----
  const pick2DTemplate = (tplId: string) => {
    const t = MOLECULE_2D_TEMPLATES.find(x => x.id === tplId)
    if (!t) return
    setActive2DId(t.id)
    setSmiles(t.smiles)
    setSmilesName(t.name + ' ' + t.formula)
    setPreviewError('')
  }

  // ---- 2D:手输 SMILES(显示名退化;若恰好与某模板一致则认领回来) ----
  const handleSmilesInput = (v: string) => {
    setSmiles(v)
    const hit = MOLECULE_2D_TEMPLATES.find(t => t.smiles === v.trim())
    if (hit) { setActive2DId(hit.id); setSmilesName(hit.name + ' ' + hit.formula) }
    else { setActive2DId(''); setSmilesName('自定义分子') }
  }

  // ---- 融入当前页:按模式拼指令交父组件 ----
  const handleInsert = () => {
    if (inserting) return
    const common = { width: size.width, height: size.height, caption: caption.trim() || undefined }
    const instruction = mode === '3d'
      ? buildMoleculeRefineInstruction({ mode: '3d', mol: { template: active3D, styleKey, spin, showLabels, background, ...common } })
      : buildMoleculeRefineInstruction({ mode: '2d', mol: { smiles: smiles.trim(), displayName: smilesName, ...common } })
    onInsert(instruction)
  }

  const insertDisabled = inserting || !libReady || !!libError
    || (mode === '2d' && (!smiles.trim() || !!previewError))

  // ==================== 渲染 ====================

  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(17,24,39,0.62)', backdropFilter: 'blur(3px)', zIndex: 99993, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={() => { if (!inserting) onClose() }}
    >
      <style>{MODAL_CSS}</style>
      <div
        style={{ width: 'min(1220px, 95vw)', height: 'min(780px, 93vh)', background: '#fff', borderRadius: 20, display: 'flex', flexDirection: 'column', overflow: 'hidden', boxShadow: '0 28px 70px rgba(0,0,0,0.35)' }}
        onClick={e => e.stopPropagation()}
      >
        {/* ===== 顶栏:绿色渐变标题条 ===== */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '15px 22px', background: 'linear-gradient(135deg, #10B981 0%, #059669 55%, #047857 100%)', flexShrink: 0 }}>
          <span style={{ width: 44, height: 44, borderRadius: 13, background: 'rgba(255,255,255,0.18)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 23, flexShrink: 0 }}>⚗️</span>
          <div style={{ minWidth: 0 }}>
            <div style={{ fontSize: 17, fontWeight: 800, color: '#fff', letterSpacing: 0.3 }}>分子实验室</div>
            <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.78)', marginTop: 2 }}>3D 立体分子 · 2D 结构式 · 数据全内联,离线断网照常渲染</div>
          </div>
          <span style={{ marginLeft: 6, padding: '5px 13px', borderRadius: 999, background: 'rgba(255,255,255,0.16)', border: '1px solid rgba(255,255,255,0.28)', color: '#fff', fontSize: 12, fontWeight: 700, whiteSpace: 'nowrap' }}>
            将融入:第 {pageNum} 页
          </span>
          <button
            onClick={() => { if (!inserting) onClose() }}
            style={{ marginLeft: 'auto', border: 'none', background: 'rgba(255,255,255,0.14)', width: 32, height: 32, borderRadius: 10, fontSize: 16, cursor: 'pointer', color: '#fff', flexShrink: 0 }}
            title="关闭"
          >✕</button>
        </div>

        {/* ===== 主体:左模板画廊 + 右预览/参数 ===== */}
        <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>

          {/* ---- 左栏:模式切换 + 模板卡片 ---- */}
          <div style={{ width: 268, borderRight: '1px solid ' + C.border, display: 'flex', flexDirection: 'column', flexShrink: 0, background: '#FAFDFB' }}>
            {/* 模式分段切换(固定头部) */}
            <div style={{ padding: '12px 12px 10px', borderBottom: '1px solid #ECF5F0', flexShrink: 0 }}>
              <div style={{ display: 'flex', borderRadius: 11, overflow: 'hidden', border: '1.5px solid #D5EDE2' }}>
                {([['3d', '🧊 3D 立体分子'], ['2d', '✏️ 2D 结构式']] as [LabMode, string][]).map(([k, label], i) => {
                  const on = mode === k
                  return (
                    <button key={k} onClick={() => { setMode(k); setPreviewError('') }}
                      style={{
                        flex: 1, padding: '8px 0', border: 'none', cursor: 'pointer', fontSize: 12.5, fontWeight: on ? 800 : 600,
                        borderRight: i === 0 ? '1px solid #D5EDE2' : 'none',
                        background: on ? 'linear-gradient(135deg, #10B981, #047857)' : '#fff',
                        color: on ? '#fff' : C.textSecondary, whiteSpace: 'nowrap',
                      }}
                    >{label}</button>
                  )
                })}
              </div>
              <div style={{ fontSize: 11, color: C.textMuted, marginTop: 7, lineHeight: 1.5 }}>
                {mode === '3d' ? '可旋转缩放的立体模型,适合讲分子构型与晶体' : '平面结构式,适合讲官能团与化学式;可直接粘贴 SMILES'}
              </div>
            </div>
            {/* 分组卡片列表(可滚动) */}
            <div style={{ flex: 1, overflowY: 'auto', padding: '10px 10px 14px' }}>
              {mode === '3d' && getMolecule3DGroups().map(g => (
                <div key={g.group} style={{ marginBottom: 13 }}>
                  <div style={{ fontSize: 11.5, fontWeight: 800, color: '#5F9E85', padding: '3px 6px 6px', letterSpacing: 0.4 }}>{g.group}</div>
                  {g.items.map(tpl => {
                    const active = tpl.id === active3DId
                    return (
                      <div key={tpl.id} className="ml-card" onClick={() => { setActive3DId(tpl.id); setPreviewError('') }}
                        style={{
                          display: 'flex', gap: 10, alignItems: 'flex-start', padding: '10px 11px', borderRadius: 13, cursor: 'pointer', marginBottom: 6,
                          background: active ? 'linear-gradient(135deg, #ECFDF5, #D1FAE5)' : '#fff',
                          border: '1.5px solid ' + (active ? '#059669' : '#EBF3EE'),
                          boxShadow: active ? '0 4px 14px rgba(5,150,105,0.16)' : '0 1px 3px rgba(0,0,0,0.03)',
                        }}
                      >
                        <span style={{ width: 34, height: 34, borderRadius: 10, background: active ? '#fff' : '#ECFDF5', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 18, flexShrink: 0 }}>{tpl.emoji}</span>
                        <div style={{ minWidth: 0 }}>
                          <div style={{ fontSize: 13, fontWeight: active ? 800 : 600, color: active ? '#047857' : C.textPrimary, lineHeight: 1.35 }}>{tpl.name} <span style={{ fontWeight: 500, fontSize: 11.5, color: active ? '#059669' : C.textMuted }}>{tpl.formula}</span></div>
                          <div style={{ fontSize: 11, color: C.textMuted, marginTop: 3, lineHeight: 1.5 }}>{tpl.desc}</div>
                        </div>
                      </div>
                    )
                  })}
                </div>
              ))}
              {mode === '2d' && getMolecule2DGroups().map(g => (
                <div key={g.group} style={{ marginBottom: 13 }}>
                  <div style={{ fontSize: 11.5, fontWeight: 800, color: '#5F9E85', padding: '3px 6px 6px', letterSpacing: 0.4 }}>{g.group}</div>
                  {g.items.map(tpl => {
                    const active = tpl.id === active2DId
                    return (
                      <div key={tpl.id} className="ml-card" onClick={() => pick2DTemplate(tpl.id)}
                        style={{
                          display: 'flex', gap: 10, alignItems: 'center', padding: '9px 11px', borderRadius: 13, cursor: 'pointer', marginBottom: 6,
                          background: active ? 'linear-gradient(135deg, #ECFDF5, #D1FAE5)' : '#fff',
                          border: '1.5px solid ' + (active ? '#059669' : '#EBF3EE'),
                          boxShadow: active ? '0 4px 14px rgba(5,150,105,0.16)' : '0 1px 3px rgba(0,0,0,0.03)',
                        }}
                      >
                        <div style={{ minWidth: 0 }}>
                          <div style={{ fontSize: 13, fontWeight: active ? 800 : 600, color: active ? '#047857' : C.textPrimary }}>{tpl.name} <span style={{ fontWeight: 500, fontSize: 11.5, color: active ? '#059669' : C.textMuted }}>{tpl.formula}</span></div>
                          <div style={{ fontSize: 11, color: C.textMuted, marginTop: 2, lineHeight: 1.5 }}>{tpl.desc}</div>
                        </div>
                      </div>
                    )
                  })}
                </div>
              ))}
            </div>
          </div>

          {/* ---- 右栏:预览 + 参数 ---- */}
          <div style={{ flex: 1, display: 'flex', minWidth: 0 }}>

            {/* 预览区(浅绿底) */}
            <div style={{ flex: 1, overflow: 'auto', padding: '18px 20px', display: 'flex', flexDirection: 'column', alignItems: 'center', minWidth: 0, background: '#F6FBF8' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 13, flexWrap: 'wrap', justifyContent: 'center' }}>
                <span style={{ fontSize: 19 }}>{mode === '3d' ? active3D.emoji : '✏️'}</span>
                <span style={{ fontSize: 15, fontWeight: 800, color: C.textPrimary }}>
                  {mode === '3d' ? active3D.name + ' ' + active3D.formula : smilesName}
                </span>
                <span style={{ fontSize: 12, color: C.textMuted }}>
                  {mode === '3d' ? active3D.desc : 'SMILES: ' + (smiles.trim() || '(空)')}
                </span>
              </div>
              {libError && (
                <div style={{ padding: '12px 16px', borderRadius: 10, background: '#FEE2E2', color: '#DC2626', fontSize: 13 }}>
                  ❌ {libError}(请检查网络后重新打开弹窗)
                </div>
              )}
              {/* 库加载骨架屏 */}
              {!libReady && !libError && (
                <div className="ml-skeleton" style={{ width: size.width, height: size.height, borderRadius: 16, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <span style={{ fontSize: 13, color: '#5F9E85', fontWeight: 600 }}>⏳ 正在加载{mode === '3d' ? ' 3D 分子引擎' : '结构式引擎'}…</span>
                </div>
              )}
              {/* 画布卡片(所见即所得,尺寸即课件内实际尺寸) */}
              <div style={{ padding: 12, background: '#fff', borderRadius: 18, border: '1px solid #D1FAE5', boxShadow: '0 10px 30px rgba(5,150,105,0.10)', display: libReady ? 'block' : 'none', flexShrink: 0 }}>
                {mode === '3d' ? (
                  <div ref={host3dRef} style={{ width: size.width, height: size.height, position: 'relative', borderRadius: 10, overflow: 'hidden', background: background }} />
                ) : (
                  <canvas id="molecule-modal-2d-canvas" width={size.width} height={size.height} style={{ display: 'block', borderRadius: 10 }} />
                )}
              </div>
              {previewError && (
                <div style={{ marginTop: 10, padding: '8px 12px', borderRadius: 10, background: '#FEE2E2', color: '#DC2626', fontSize: 12 }}>{previewError}</div>
              )}
              {caption.trim() && (
                <div style={{ marginTop: 10, fontSize: 14, color: '#6B7280', fontStyle: 'italic' }}>{caption.trim()}</div>
              )}
              <div style={{ marginTop: 12, fontSize: 12, color: '#8FBBA6' }}>
                {mode === '3d' ? '💡 预览里可以直接鼠标拖拽旋转、滚轮缩放,融入课件后课堂上同样可以' : '💡 结构式为静态图,融入后即刻显示'}
              </div>
            </div>

            {/* 参数区 */}
            <div style={{ width: 300, borderLeft: '1px solid ' + C.border, overflowY: 'auto', padding: '16px 16px 20px', flexShrink: 0, background: '#fff' }}>
              <div style={{ padding: '11px 13px', borderRadius: 13, background: 'linear-gradient(135deg, #ECFDF5, #D1FAE5)', border: '1px solid #BBECD8', marginBottom: 15 }}>
                <div style={{ fontSize: 13, fontWeight: 800, color: '#047857' }}>⚙️ {mode === '3d' ? '显示设置' : '结构式输入'}</div>
                <div style={{ fontSize: 11.5, color: '#4F8F76', lineHeight: 1.6, marginTop: 4 }}>
                  {mode === '3d' ? '分子数据全部内联在生成的代码里,断网也能渲染。' : '点左侧模板或直接输入/粘贴 SMILES 串。'}
                </div>
              </div>

              {mode === '3d' ? (
                <>
                  {/* 显示样式:分段选择器 */}
                  <div style={{ marginBottom: 15 }}>
                    <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 7 }}>显示样式</div>
                    <div style={{ display: 'flex', borderRadius: 11, overflow: 'hidden', border: '1.5px solid #BBECD8' }}>
                      {MOLECULE_STYLE_OPTIONS.map((s, i) => {
                        const on = s.key === styleKey
                        return (
                          <button key={s.key} onClick={() => setStyleKey(s.key)} title={s.hint}
                            style={{
                              flex: 1, padding: '7px 0', border: 'none', cursor: 'pointer',
                              borderRight: i < MOLECULE_STYLE_OPTIONS.length - 1 ? '1px solid #BBECD8' : 'none',
                              background: on ? 'linear-gradient(135deg, #10B981, #047857)' : '#fff',
                              color: on ? '#fff' : C.textSecondary, fontSize: 12.5, fontWeight: on ? 800 : 600,
                            }}
                          >{s.label}</button>
                        )
                      })}
                    </div>
                    <div style={{ fontSize: 11, color: C.textMuted, marginTop: 5 }}>{MOLECULE_STYLE_OPTIONS.find(s => s.key === styleKey)?.hint}</div>
                  </div>

                  {/* 自动旋转 / 元素标签 */}
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 15 }}>
                    <span style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>自动旋转展示</span>
                    <Toggle checked={spin} onChange={setSpin} />
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 15 }}>
                    <span style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>显示元素标签</span>
                    <Toggle checked={showLabels} onChange={setShowLabels} />
                  </div>

                  {/* 背景色 */}
                  <div style={{ marginBottom: 15 }}>
                    <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 7 }}>画布背景</div>
                    <div style={{ display: 'flex', borderRadius: 11, overflow: 'hidden', border: '1.5px solid #BBECD8' }}>
                      {MOLECULE_BG_PRESETS.map((b, i) => {
                        const on = b.value === background
                        return (
                          <button key={b.value} onClick={() => setBackground(b.value)}
                            style={{
                              flex: 1, padding: '7px 0', border: 'none', cursor: 'pointer',
                              borderRight: i < MOLECULE_BG_PRESETS.length - 1 ? '1px solid #BBECD8' : 'none',
                              background: on ? 'linear-gradient(135deg, #10B981, #047857)' : '#fff',
                              color: on ? '#fff' : C.textSecondary, fontSize: 12.5, fontWeight: on ? 800 : 600,
                            }}
                          >{b.label}</button>
                        )
                      })}
                    </div>
                    <div style={{ fontSize: 11, color: C.textMuted, marginTop: 5 }}>深色背景下空间填充模型立体感更强</div>
                  </div>
                </>
              ) : (
                /* 2D:SMILES 输入 */
                <div style={{ marginBottom: 15 }}>
                  <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 7 }}>SMILES 串</div>
                  <textarea
                    value={smiles}
                    onChange={e => handleSmilesInput(e.target.value)}
                    placeholder="如:CC(=O)O(乙酸)、c1ccccc1(苯)"
                    rows={3}
                    style={{ width: '100%', boxSizing: 'border-box', padding: '8px 11px', borderRadius: 10, border: '1.5px solid #D5EDE2', fontSize: 13, outline: 'none', resize: 'vertical', fontFamily: 'monospace' }}
                  />
                  <div style={{ fontSize: 11, color: C.textMuted, marginTop: 5, lineHeight: 1.6 }}>
                    SMILES 是化学结构的文本记法,教材/百科上查到的串可直接粘贴,预览实时更新。
                  </div>
                </div>
              )}

              <div style={{ borderTop: '1px dashed #BBECD8', margin: '16px 0' }} />

              {/* 画布尺寸(双模式共用) */}
              <div style={{ marginBottom: 15 }}>
                <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 7 }}>画布尺寸</div>
                <div style={{ display: 'flex', borderRadius: 11, overflow: 'hidden', border: '1.5px solid #BBECD8' }}>
                  {MOLECULE_SIZE_PRESETS.map((s, i) => {
                    const on = i === sizeIdx
                    return (
                      <button key={s.label} onClick={() => setSizeIdx(i)}
                        style={{
                          flex: 1, padding: '7px 0', border: 'none', cursor: 'pointer',
                          borderRight: i < MOLECULE_SIZE_PRESETS.length - 1 ? '1px solid #BBECD8' : 'none',
                          background: on ? 'linear-gradient(135deg, #10B981, #047857)' : '#fff',
                          color: on ? '#fff' : C.textSecondary,
                        }}
                      >
                        <div style={{ fontSize: 12.5, fontWeight: on ? 800 : 600 }}>{['小', '标准', '大'][i]}</div>
                        <div style={{ fontSize: 10, opacity: 0.85, marginTop: 1 }}>{s.width}×{s.height}</div>
                      </button>
                    )
                  })}
                </div>
              </div>

              {/* 标注 */}
              <div style={{ marginBottom: 8 }}>
                <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 7 }}>标注文字(可选)</div>
                <input
                  type="text"
                  value={caption}
                  onChange={e => setCaption(e.target.value)}
                  placeholder={mode === '3d' ? '如:拖动旋转观察水分子的V形结构' : '如:乙酸分子中的羧基官能团'}
                  maxLength={60}
                  style={{ width: '100%', boxSizing: 'border-box', padding: '8px 11px', borderRadius: 10, border: '1.5px solid #D5EDE2', fontSize: 13, outline: 'none' }}
                />
              </div>
            </div>
          </div>
        </div>

        {/* ===== 底栏 ===== */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '13px 22px', borderTop: '1px solid ' + C.border, flexShrink: 0, background: '#FAFDFB' }}>
          <span style={{ fontSize: 12, color: C.textMuted }}>
            基于 3Dmol.js(BSD-3)与 smiles-drawer(MIT)开源库,分子数据全内联,离线可用
          </span>
          <div style={{ marginLeft: 'auto', display: 'flex', gap: 10 }}>
            <button
              onClick={() => { if (!inserting) onClose() }}
              disabled={inserting}
              style={{ padding: '10px 22px', borderRadius: 12, border: '1.5px solid #D5EDE2', background: '#fff', color: C.textSecondary, fontSize: 14, fontWeight: 600, cursor: inserting ? 'not-allowed' : 'pointer' }}
            >取消</button>
            <button
              onClick={handleInsert}
              disabled={insertDisabled}
              title={mode === '2d' && !smiles.trim() ? '先输入或选择一个 SMILES' : undefined}
              style={{
                padding: '10px 26px', borderRadius: 12, border: 'none', fontSize: 14, fontWeight: 800,
                background: insertDisabled ? '#A7F3D0' : 'linear-gradient(135deg, #10B981, #047857)',
                boxShadow: insertDisabled ? 'none' : '0 6px 18px rgba(4,120,87,0.35)',
                color: '#fff', cursor: insertDisabled ? 'not-allowed' : 'pointer',
              }}
            >{inserting ? '⏳ AI 融入中…' : '⚗️ 融入第 ' + pageNum + ' 页'}</button>
          </div>
        </div>
      </div>
    </div>
  )
}
