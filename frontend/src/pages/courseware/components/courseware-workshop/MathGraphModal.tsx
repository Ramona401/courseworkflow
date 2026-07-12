/**
 * MathGraphModal.tsx — 课件数学动态图形编辑器弹窗(批次1b首发;批次1d UI现代化;批次A AI定制,2026-07-08)
 *
 * 批次A 新增「AI 定制」双入口(48 个模板由"固定题"升维成"题族种子"):
 *   - 入口一·模板变种:模板卡片上「🔧 改编」按钮 → 以该模板当前构造代码为底稿,
 *     老师用自然语言描述变化,AI 做有底稿的最小改写(成功率高,天然继承版式与教学设计);
 *   - 入口二·从零生成:左栏搜索框下「✨ AI 描述新图形」→ 纯自然语言从零生成,兜底
 *     模板库没有的题型。
 *   AI 模式下右栏参数区替换为 MathGraphAIPanel(描述/生成/追改),尺寸/坐标轴/网格/标注
 *   等共用控件不变;预览画板执行 AI 代码(同样过 applyMathPalette 换装珊瑚粉);融入时用
 *   "合成模板"包装 AI 代码走既有 buildMathRefineInstruction,mathGraphUtils 零改动。
 *
 * 批次1d 改版要点(逻辑层零改动,纯视觉与信息架构升级):
 *   - 顶栏:紫色渐变大标题条 + 毛玻璃徽章;
 *   - 左栏:学段胶囊筛选 + 关键词搜索 + 现代卡片;
 *   - 预览区:浅紫底 + 白色圆角阴影画板卡片 + 库加载骨架屏;
 *   - 参数区:自定义紫色滑杆 + Toggle 开关 + 值胶囊输入框 + 分段式尺寸选择器。
 *
 * 单一真相源(与 mathGraphUtils.ts 文件头架构约定对应):
 *   预览与课件融入共享同一段构造代码(模板 buildConstruction 产出或 AI 生成)——
 *   预览侧:initBoard 后 new Function('board','JXG', code)(board, JXG) 直接执行;
 *   融入侧:generateMathGraphEmbed 把同一段代码包进自包含 HTML。
 *
 * 库加载:JSXGraph 1.12.2 自托管(css + js 均动态注入,轮询就绪),不进主包。
 *   参数变化重建画板前用 JXG.JSXGraph.freeBoard 释放旧板防内存泄漏。
 *
 * 交互范式对齐 FormulaEditorModal / StrokeOrderModal:
 *   props 签名 {pageNum, onInsert(instruction), onClose, inserting?} 完全一致,
 *   由 SubjectToolsPanel 的 makeInsertHandler 工厂统一驱动融入。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/MathGraphModal.tsx
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import { C } from './workshopConstants'
import {
  JSXGRAPH_JS_CDN, JSXGRAPH_CSS_CDN,
  MATH_GRAPH_DEFAULTS, MATH_GRAPH_SIZE_PRESETS, MATH_BOARD_STYLE_JS, applyMathPalette,
  buildMathRefineInstruction,
} from './mathGraphUtils'
import type { MathGraphTemplate, MathParamValue } from './mathGraphUtils'
import { MATH_GRAPH_TEMPLATES, getMathTemplateGroups } from './mathGraphTemplates'
import MathGraphAIPanel from './MathGraphAIPanel'

// JSXGraph 挂在 window.JXG(全局脚本非模块),TS 侧统一 any 处理
declare global {
  interface Window { JXG?: any }
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

// ==================== 弹窗专用 CSS(骨架屏动画 / 卡片悬浮 / 自定义滑杆) ====================
// 类名带 mg- 前缀限定作用域;<style> 随弹窗挂载卸载,不污染全局。

const MODAL_CSS = [
  '@keyframes mg-shimmer { 0% { background-position: -420px 0; } 100% { background-position: 420px 0; } }',
  '.mg-skeleton { background: linear-gradient(90deg, #F3F4F6 25%, #EDE9FE 40%, #F3F4F6 60%); background-size: 840px 100%; animation: mg-shimmer 1.3s infinite linear; }',
  '.mg-card { transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s; }',
  '.mg-card:hover { border-color: #C4B5FD !important; box-shadow: 0 4px 14px rgba(124,58,237,0.14); transform: translateY(-1px); }',
  '.mg-adapt-btn { opacity: 0; transition: opacity 0.15s; }',
  '.mg-card:hover .mg-adapt-btn { opacity: 1; }',
  '.mg-range { -webkit-appearance: none; appearance: none; width: 100%; height: 6px; border-radius: 3px; background: #EDE9FE; outline: none; cursor: pointer; }',
  '.mg-range::-webkit-slider-thumb { -webkit-appearance: none; appearance: none; width: 18px; height: 18px; border-radius: 50%; background: linear-gradient(135deg, #8B5CF6, #6D28D9); border: 2.5px solid #fff; box-shadow: 0 1px 4px rgba(109,40,217,0.45); cursor: pointer; }',
  '.mg-range::-moz-range-thumb { width: 18px; height: 18px; border-radius: 50%; background: #7C3AED; border: 2.5px solid #fff; box-shadow: 0 1px 4px rgba(109,40,217,0.45); cursor: pointer; }',
].join('\n')

// ==================== 学段胶囊筛选定义(按分组名 emoji 前缀匹配) ====================

const STAGES = [
  { key: 'all', label: '全部' },
  { key: '🧒', label: '🧒 小学' },
  { key: '📗', label: '📗 初中' },
  { key: '📘', label: '📘 高中' },
] as const

// ==================== 批次A:AI 从零生成模式的默认画板范围(与后端默认值一致) ====================

const AI_CREATE_BB: [number, number, number, number] = [-10, 8, 10, -6]

// ==================== 库加载(模块级单例,多次开弹窗不重复注入) ====================

/** 注入 JSXGraph CSS(幂等:按 href 特征去重) */
function ensureJSXGraphCSS() {
  if (document.querySelector('link[href*="jsxgraph.css"]')) return
  const link = document.createElement('link')
  link.rel = 'stylesheet'
  link.href = JSXGRAPH_CSS_CDN
  document.head.appendChild(link)
}

/** 加载 JSXGraph JS(幂等 + 轮询就绪),resolve 后 window.JXG 可用 */
function loadJSXGraphLib(): Promise<void> {
  return new Promise((resolve, reject) => {
    if (window.JXG && window.JXG.JSXGraph) { resolve(); return }
    // 已有 script 标签(可能正在加载)→ 轮询等待
    const existing = document.querySelector('script[src*="jsxgraphcore.js"]')
    if (existing) {
      let tries = 0
      const timer = window.setInterval(() => {
        tries++
        if (window.JXG && window.JXG.JSXGraph) { window.clearInterval(timer); resolve() }
        else if (tries > 100) { window.clearInterval(timer); reject(new Error('JSXGraph 加载超时')) }
      }, 100)
      return
    }
    const s = document.createElement('script')
    s.src = JSXGRAPH_JS_CDN
    s.onload = () => {
      if (window.JXG && window.JXG.JSXGraph) resolve()
      else reject(new Error('JSXGraph 加载后未就绪'))
    }
    s.onerror = () => reject(new Error('JSXGraph 脚本加载失败(网络或自托管地址问题)'))
    document.head.appendChild(s)
  })
}

// ==================== 辅助 ====================

/** 从模板参数定义构造默认参数取值表 */
function buildDefaultParams(tpl: MathGraphTemplate): Record<string, MathParamValue> {
  const out: Record<string, MathParamValue> = {}
  for (const p of tpl.params) out[p.key] = p.defaultValue
  return out
}

/** 开关组件(替代原生 checkbox 的现代 Toggle) */
function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <div
      onClick={() => onChange(!checked)}
      style={{
        width: 40, height: 22, borderRadius: 11, flexShrink: 0, cursor: 'pointer',
        background: checked ? 'linear-gradient(135deg, #8B5CF6, #6D28D9)' : '#D1D5DB',
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

export default function MathGraphModal({ pageNum, onInsert, onClose, inserting }: Props) {
  // ---- 状态 ----
  const [libReady, setLibReady] = useState(false)          // JSXGraph 库就绪
  const [libError, setLibError] = useState('')             // 库加载失败信息
  const [activeTplId, setActiveTplId] = useState(MATH_GRAPH_TEMPLATES[0].id)  // 当前模板
  const [params, setParams] = useState<Record<string, MathParamValue>>(buildDefaultParams(MATH_GRAPH_TEMPLATES[0]))
  const [sizeIdx, setSizeIdx] = useState(1)                // 尺寸方案索引(默认"标准")
  const [showGrid, setShowGrid] = useState<boolean>(MATH_GRAPH_DEFAULTS.showGrid)
  const [showAxis, setShowAxis] = useState<boolean>(MATH_GRAPH_TEMPLATES[0].showAxis !== false)  // 批次3a:坐标轴开关,模板默认值可被覆盖
  const [caption, setCaption] = useState('')               // 可选标注
  const [previewError, setPreviewError] = useState('')     // 预览渲染错误
  const [stage, setStage] = useState<string>('all')        // 学段筛选(all/🧒/📗/📘)
  const [search, setSearch] = useState('')                 // 模板关键词搜索
  // ---- 批次A:AI 定制模式状态 ----
  const [aiMode, setAiMode] = useState<'adapt' | 'create' | null>(null)  // null=普通模板模式
  const [aiCode, setAiCode] = useState('')                 // AI 生成的构造代码(空=尚未生成)
  const [aiBaseCode, setAiBaseCode] = useState('')         // adapt 入口的底稿代码快照

  // ---- refs ----
  const boardHostRef = useRef<HTMLDivElement | null>(null) // 画板容器
  const boardRef = useRef<any>(null)                       // 当前 JSXGraph board 实例(供 freeBoard)
  const debounceRef = useRef<number | null>(null)          // 参数变化防抖定时器

  const activeTpl = MATH_GRAPH_TEMPLATES.find(t => t.id === activeTplId) || MATH_GRAPH_TEMPLATES[0]
  const size = MATH_GRAPH_SIZE_PRESETS[sizeIdx] || MATH_GRAPH_SIZE_PRESETS[1]

  // ---- 批次A:AI 模式派生量(create 用默认框,adapt 沿用底稿模板的框与纵横比) ----
  const isAI = aiMode !== null
  const effBB: [number, number, number, number] = (aiMode === 'create') ? AI_CREATE_BB : activeTpl.boundingBox
  const effKeepAspect = (aiMode === 'create') ? true : activeTpl.keepAspectRatio

  // ---- 左栏可见分组:先按学段胶囊过滤,再按搜索词过滤(名称+说明),空组不渲染 ----
  const kw = search.trim().toLowerCase()
  const visibleGroups = getMathTemplateGroups()
    .filter(g => stage === 'all' || g.group.startsWith(stage))
    .map(g => ({ group: g.group, items: g.items.filter(t => !kw || (t.name + t.desc).toLowerCase().includes(kw)) }))
    .filter(g => g.items.length > 0)

  // ---- 挂载:加载库(CSS 立即注入,JS 异步就绪) ----
  useEffect(() => {
    ensureJSXGraphCSS()
    let cancelled = false
    loadJSXGraphLib()
      .then(() => { if (!cancelled) setLibReady(true) })
      .catch(e => { if (!cancelled) setLibError(e instanceof Error ? e.message : '库加载失败') })
    return () => { cancelled = true }
  }, [])

  // ---- 预览渲染核心:释放旧板 → initBoard → 执行构造代码(模板产出或 AI 生成,与融入同一份) ----
  const renderPreview = useCallback(() => {
    if (!libReady || !boardHostRef.current || !window.JXG) return
    setPreviewError('')
    // 释放旧板防内存泄漏
    if (boardRef.current) {
      try { window.JXG.JSXGraph.freeBoard(boardRef.current) } catch { /* 忽略释放异常 */ }
      boardRef.current = null
    }
    // 容器 ID 固定(弹窗内单画板)
    const host = boardHostRef.current
    host.innerHTML = ''
    host.id = 'mathgraph-modal-preview'
    try {
      // 全局画板风格(字体/坐标轴/滑杆默认值)——与课件 embed 共用同一段 JS,幂等
      // eslint-disable-next-line no-new-func
      new Function('JXG', MATH_BOARD_STYLE_JS)(window.JXG)
      const board = window.JXG.JSXGraph.initBoard('mathgraph-modal-preview', {
        boundingbox: effBB,
        axis: showAxis,
        grid: showGrid,
        keepaspectratio: effKeepAspect,
        showCopyright: false,
        showNavigation: false,
        pan: { enabled: false },
        zoom: { enabled: false, wheel: false },
      })
      boardRef.current = board
      // 单一真相源:预览执行与课件融入完全相同的构造代码(统一过调色板换装)
      // AI 模式:有 AI 代码执行 AI 代码;adapt 未生成时先展示模板底稿;create 未生成时留空板
      let rawCode = ''
      if (isAI) {
        if (aiCode.trim()) rawCode = aiCode
        else if (aiMode === 'adapt') rawCode = activeTpl.buildConstruction(params)
      } else {
        rawCode = activeTpl.buildConstruction(params)
      }
      if (rawCode) {
        const code = applyMathPalette(rawCode)
        // eslint-disable-next-line no-new-func
        new Function('board', 'JXG', code)(board, window.JXG)
      }
    } catch (e) {
      setPreviewError('预览渲染失败: ' + (e instanceof Error ? e.message : String(e)))
    }
  }, [libReady, activeTpl, params, showGrid, showAxis, isAI, aiMode, aiCode, effBB, effKeepAspect])

  // ---- 模板/网格/尺寸/AI代码变化:300ms 防抖重渲染 ----
  useEffect(() => {
    if (!libReady) return
    if (debounceRef.current) window.clearTimeout(debounceRef.current)
    debounceRef.current = window.setTimeout(() => { renderPreview() }, 300)
    return () => { if (debounceRef.current) window.clearTimeout(debounceRef.current) }
  }, [libReady, renderPreview])

  // ---- 卸载:释放画板 ----
  useEffect(() => {
    return () => {
      if (boardRef.current && window.JXG) {
        try { window.JXG.JSXGraph.freeBoard(boardRef.current) } catch { /* 忽略 */ }
        boardRef.current = null
      }
    }
  }, [])

  // ---- 切换模板:重置参数为该模板默认值(同时退出 AI 模式) ----
  const handlePickTemplate = (tpl: MathGraphTemplate) => {
    setActiveTplId(tpl.id)
    setParams(buildDefaultParams(tpl))
    setShowAxis(tpl.showAxis !== false)  // 批次3a:切换模板重置为模板默认
    setPreviewError('')
    setAiMode(null); setAiCode(''); setAiBaseCode('')
  }

  // ---- 批次A:进入「模板改编」模式(底稿 = 该模板当前构造代码;若点的正是当前模板则保留已调参数) ----
  const enterAdaptMode = (tpl: MathGraphTemplate) => {
    const p = tpl.id === activeTplId ? params : buildDefaultParams(tpl)
    setActiveTplId(tpl.id)
    setParams(p)
    setShowAxis(tpl.showAxis !== false)
    setAiBaseCode(tpl.buildConstruction(p))
    setAiCode('')
    setAiMode('adapt')
    setPreviewError('')
  }

  // ---- 批次A:进入「从零生成」模式 ----
  const enterCreateMode = () => {
    setAiBaseCode('')
    setAiCode('')
    setAiMode('create')
    setPreviewError('')
  }

  // ---- 批次A:退出 AI 模式,回到普通模板模式 ----
  const exitAIMode = () => {
    setAiMode(null); setAiCode(''); setAiBaseCode(''); setPreviewError('')
  }

  // ---- 参数修改 ----
  const setParam = (key: string, v: MathParamValue) => {
    setParams(prev => ({ ...prev, [key]: v }))
  }

  // ---- 融入当前页:拼指令交父组件(AI 模式用合成模板包装 AI 代码,走同一融入链路) ----
  const handleInsert = () => {
    if (inserting) return
    let tplForInsert: MathGraphTemplate = activeTpl
    let paramsForInsert: Record<string, MathParamValue> = params
    if (isAI) {
      if (!aiCode.trim()) return
      const frozenCode = aiCode  // 冻结当前 AI 代码,防闭包引用后续变化
      tplForInsert = {
        id: 'ai-custom',
        group: '✨ AI定制',
        name: aiMode === 'adapt' ? activeTpl.name + '·AI改编' : 'AI自定义图形',
        emoji: '✨',
        desc: 'AI 定制的交互图形',
        boundingBox: effBB,
        keepAspectRatio: effKeepAspect,
        params: [],
        buildConstruction: () => frozenCode,
      }
      paramsForInsert = {}
    }
    const instruction = buildMathRefineInstruction({
      graph: {
        template: tplForInsert,
        params: paramsForInsert,
        width: size.width,
        height: size.height,
        showGrid,
        showAxis,
        caption: caption.trim() || undefined,
      },
    })
    onInsert(instruction)
  }

  const insertDisabled = inserting || !libReady || !!libError || (isAI && !aiCode.trim())

  // ==================== 渲染 ====================

  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(17,24,39,0.62)', backdropFilter: 'blur(3px)', zIndex: 99993, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={() => { if (!inserting) onClose() }}
    >
      {/* 弹窗专用样式(骨架屏/卡片悬浮/自定义滑杆) */}
      <style>{MODAL_CSS}</style>
      <div
        style={{ width: 'min(1220px, 95vw)', height: 'min(780px, 93vh)', background: '#fff', borderRadius: 20, display: 'flex', flexDirection: 'column', overflow: 'hidden', boxShadow: '0 28px 70px rgba(0,0,0,0.35)' }}
        onClick={e => e.stopPropagation()}
      >
        {/* ===== 顶栏:紫色渐变标题条 ===== */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '15px 22px', background: 'linear-gradient(135deg, #7C3AED 0%, #6D28D9 55%, #5B21B6 100%)', flexShrink: 0 }}>
          <span style={{ width: 44, height: 44, borderRadius: 13, background: 'rgba(255,255,255,0.18)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 23, flexShrink: 0 }}>📊</span>
          <div style={{ minWidth: 0 }}>
            <div style={{ fontSize: 17, fontWeight: 800, color: '#fff', letterSpacing: 0.3 }}>数学动态图形</div>
            <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.78)', marginTop: 2 }}>模板点选 · AI 改编定制 · 融入后课堂仍可拖动交互</div>
          </div>
          {isAI && (
            <span style={{ padding: '5px 13px', borderRadius: 999, background: 'rgba(255,255,255,0.24)', border: '1px solid rgba(255,255,255,0.38)', color: '#fff', fontSize: 12, fontWeight: 700, whiteSpace: 'nowrap' }}>
              ✨ AI 定制中
            </span>
          )}
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

          {/* ---- 左栏:学段筛选 + 搜索 + AI从零入口 + 模板卡片 ---- */}
          <div style={{ width: 268, borderRight: '1px solid ' + C.border, display: 'flex', flexDirection: 'column', flexShrink: 0, background: '#FBFAFE' }}>
            {/* 学段胶囊 + 搜索 + AI入口(固定头部) */}
            <div style={{ padding: '12px 12px 8px', borderBottom: '1px solid #F1EEF9', flexShrink: 0 }}>
              <div style={{ display: 'flex', gap: 5, marginBottom: 9 }}>
                {STAGES.map(s => {
                  const on = stage === s.key
                  return (
                    <button key={s.key} onClick={() => setStage(s.key)}
                      style={{
                        flex: 1, padding: '5px 0', borderRadius: 999, fontSize: 11.5, fontWeight: on ? 700 : 500, cursor: 'pointer',
                        border: '1.5px solid ' + (on ? '#7C3AED' : '#E9E5F5'),
                        background: on ? 'linear-gradient(135deg, #8B5CF6, #6D28D9)' : '#fff',
                        color: on ? '#fff' : C.textSecondary, whiteSpace: 'nowrap',
                      }}
                    >{s.label}</button>
                  )
                })}
              </div>
              <input
                type="text" value={search} onChange={e => setSearch(e.target.value)}
                placeholder="🔍 搜索模板,如:对称 / 全等 / 函数"
                style={{ width: '100%', boxSizing: 'border-box', padding: '7px 11px', borderRadius: 10, border: '1.5px solid #E9E5F5', background: '#fff', fontSize: 12, outline: 'none' }}
              />
              {/* 批次A:AI 从零生成入口 */}
              <button
                onClick={enterCreateMode}
                style={{
                  width: '100%', marginTop: 8, padding: '8px 0', borderRadius: 10, cursor: 'pointer', fontSize: 12.5, fontWeight: 700,
                  border: '1.5px dashed ' + (aiMode === 'create' ? '#7C3AED' : '#C4B5FD'),
                  background: aiMode === 'create' ? 'linear-gradient(135deg, #F5F3FF, #EDE9FE)' : '#fff',
                  color: '#6D28D9',
                }}
              >✨ AI 描述新图形(模板里没有的)</button>
            </div>
            {/* 分组卡片列表(可滚动) */}
            <div style={{ flex: 1, overflowY: 'auto', padding: '10px 10px 14px' }}>
              {visibleGroups.length === 0 && (
                <div style={{ padding: '30px 10px', textAlign: 'center', fontSize: 12.5, color: C.textMuted }}>没有匹配的模板,换个关键词试试</div>
              )}
              {visibleGroups.map(g => (
                <div key={g.group} style={{ marginBottom: 13 }}>
                  <div style={{ fontSize: 11.5, fontWeight: 800, color: '#8B7FB8', padding: '3px 6px 6px', letterSpacing: 0.4 }}>{g.group}</div>
                  {g.items.map(tpl => {
                    const active = tpl.id === activeTplId
                    return (
                      <div key={tpl.id} className="mg-card" onClick={() => handlePickTemplate(tpl)}
                        style={{
                          display: 'flex', gap: 10, alignItems: 'flex-start', padding: '10px 11px', borderRadius: 13, cursor: 'pointer', marginBottom: 6,
                          background: active ? 'linear-gradient(135deg, #F5F3FF, #EDE9FE)' : '#fff',
                          border: '1.5px solid ' + (active ? '#7C3AED' : '#EFECf8'),
                          boxShadow: active ? '0 4px 14px rgba(124,58,237,0.16)' : '0 1px 3px rgba(0,0,0,0.03)',
                          position: 'relative',
                        }}
                      >
                        <span style={{ width: 34, height: 34, borderRadius: 10, background: active ? '#fff' : '#F5F3FF', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 18, flexShrink: 0 }}>{tpl.emoji}</span>
                        <div style={{ minWidth: 0, paddingRight: 2 }}>
                          <div style={{ fontSize: 13, fontWeight: active ? 800 : 600, color: active ? '#6D28D9' : C.textPrimary, lineHeight: 1.35 }}>{tpl.name}</div>
                          <div style={{ fontSize: 11, color: C.textMuted, marginTop: 3, lineHeight: 1.5 }}>{tpl.desc}</div>
                        </div>
                        {/* 批次A:改编按钮(悬浮显现;选中卡常显,提示可改编) */}
                        <button
                          className={active ? undefined : 'mg-adapt-btn'}
                          onClick={e => { e.stopPropagation(); enterAdaptMode(tpl) }}
                          title="以此模板为底稿,用自然语言改编成你的题目"
                          style={{
                            position: 'absolute', right: 7, bottom: 7, padding: '3px 9px', borderRadius: 999, fontSize: 10.5, fontWeight: 700, cursor: 'pointer', whiteSpace: 'nowrap',
                            border: '1px solid #D8CEF5', background: '#fff', color: '#6D28D9',
                            boxShadow: '0 1px 4px rgba(109,40,217,0.14)',
                          }}
                        >🔧 改编</button>
                      </div>
                    )
                  })}
                </div>
              ))}
            </div>
          </div>

          {/* ---- 右栏:预览 + 参数/AI面板 ---- */}
          <div style={{ flex: 1, display: 'flex', minWidth: 0 }}>

            {/* 预览区(浅紫底) */}
            <div style={{ flex: 1, overflow: 'auto', padding: '18px 20px', display: 'flex', flexDirection: 'column', alignItems: 'center', minWidth: 0, background: '#F8F7FC' }}>
              {/* 当前模板/AI 标题条 */}
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 13, flexWrap: 'wrap', justifyContent: 'center' }}>
                <span style={{ fontSize: 19 }}>{isAI ? '✨' : activeTpl.emoji}</span>
                <span style={{ fontSize: 15, fontWeight: 800, color: C.textPrimary }}>
                  {isAI ? (aiMode === 'adapt' ? activeTpl.name + ' · AI 改编' : 'AI 自定义图形') : activeTpl.name}
                </span>
                <span style={{ fontSize: 12, color: C.textMuted }}>
                  {isAI ? (aiCode ? '当前显示 AI 生成结果' : (aiMode === 'adapt' ? '当前显示模板底稿,待你描述变化' : '描述图形后点生成')) : activeTpl.desc}
                </span>
              </div>
              {libError && (
                <div style={{ padding: '12px 16px', borderRadius: 10, background: '#FEE2E2', color: '#DC2626', fontSize: 13 }}>
                  ❌ {libError}(请检查网络后重新打开弹窗)
                </div>
              )}
              {/* 库加载骨架屏(shimmer 占位,尺寸与真实画板一致防跳动) */}
              {!libReady && !libError && (
                <div className="mg-skeleton" style={{ width: size.width, height: size.height, borderRadius: 16, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <span style={{ fontSize: 13, color: '#8B7FB8', fontWeight: 600 }}>⏳ 正在加载图形引擎…</span>
                </div>
              )}
              {/* 画板卡片:白色圆角阴影包裹,尺寸即课件内实际尺寸(所见即所得) */}
              <div style={{ padding: 12, background: '#fff', borderRadius: 18, border: '1px solid #EDE9FE', boxShadow: '0 10px 30px rgba(109,40,217,0.10)', display: libReady ? 'block' : 'none', flexShrink: 0 }}>
                <div ref={boardHostRef} style={{ width: size.width, height: size.height, borderRadius: 10, overflow: 'hidden' }} />
              </div>
              {previewError && (
                <div style={{ marginTop: 10, padding: '8px 12px', borderRadius: 10, background: '#FEE2E2', color: '#DC2626', fontSize: 12 }}>
                  {previewError}{isAI && aiCode ? '(AI 代码执行出错,可在右侧输入要求让 AI 修正,或点重置)' : ''}
                </div>
              )}
              {caption.trim() && (
                <div style={{ marginTop: 10, fontSize: 14, color: '#6B7280', fontStyle: 'italic' }}>{caption.trim()}</div>
              )}
              <div style={{ marginTop: 12, fontSize: 12, color: '#A79FC9' }}>💡 预览里的滑杆和红点可以直接拖动试玩</div>
            </div>

            {/* 参数区 / AI 定制面板 */}
            <div style={{ width: 300, borderLeft: '1px solid ' + C.border, overflowY: 'auto', padding: '16px 16px 20px', flexShrink: 0, background: '#fff' }}>
              {isAI ? (
                /* 批次A:AI 模式下参数区替换为 AI 定制面板 */
                <MathGraphAIPanel
                  mode={aiMode as 'adapt' | 'create'}
                  templateName={aiMode === 'adapt' ? activeTpl.name : undefined}
                  baseCode={aiBaseCode}
                  boundingBox={'[' + effBB.join(', ') + ']'}
                  code={aiCode}
                  onCode={setAiCode}
                  onExit={exitAIMode}
                  busyExternal={inserting}
                  previewError={previewError}
                />
              ) : (
                <>
                  {/* 模板信息卡 */}
                  <div style={{ padding: '11px 13px', borderRadius: 13, background: 'linear-gradient(135deg, #F5F3FF, #EDE9FE)', border: '1px solid #E4DDFA', marginBottom: 15 }}>
                    <div style={{ fontSize: 13, fontWeight: 800, color: '#5B21B6' }}>⚙️ 参数设置</div>
                    <div style={{ fontSize: 11.5, color: '#7A6FA8', lineHeight: 1.6, marginTop: 4 }}>
                      这里设的值是课件里滑杆的初始位置,课堂上仍可现场拖动调整。
                    </div>
                  </div>

                  {/* 模板参数 */}
                  {activeTpl.params.length === 0 && (
                    <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 14 }}>本模板无预设参数,全部交互都在画板内直接拖动完成。</div>
                  )}
                  {activeTpl.params.map(pd => (
                    <div key={pd.key} style={{ marginBottom: 15 }}>
                      {pd.type === 'number' ? (
                        <>
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                            <span style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>{pd.label}</span>
                            {/* 值胶囊输入框:显示当前值,可直接键入精确值 */}
                            <input
                              type="number"
                              value={Number(params[pd.key])}
                              min={pd.min} max={pd.max} step={pd.step}
                              onChange={e => {
                                const v = parseFloat(e.target.value)
                                if (!Number.isNaN(v)) setParam(pd.key, Math.min(pd.max ?? v, Math.max(pd.min ?? v, v)))
                              }}
                              style={{ width: 72, padding: '3px 8px', borderRadius: 999, border: '1.5px solid #E4DDFA', background: '#F5F3FF', color: '#6D28D9', fontWeight: 700, fontSize: 12.5, textAlign: 'center', outline: 'none' }}
                            />
                          </div>
                          <input
                            type="range" className="mg-range"
                            value={Number(params[pd.key])}
                            min={pd.min} max={pd.max} step={pd.step}
                            onChange={e => setParam(pd.key, parseFloat(e.target.value))}
                          />
                        </>
                      ) : (
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10 }}>
                          <span style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, lineHeight: 1.4 }}>{pd.label}</span>
                          <Toggle checked={Boolean(params[pd.key])} onChange={v => setParam(pd.key, v)} />
                        </div>
                      )}
                      {pd.hint && <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>{pd.hint}</div>}
                    </div>
                  ))}
                </>
              )}

              <div style={{ borderTop: '1px dashed #E4DDFA', margin: '16px 0' }} />

              {/* 画板尺寸:分段式选择器(模板/AI 模式共用) */}
              <div style={{ marginBottom: 15 }}>
                <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 7 }}>画板尺寸</div>
                <div style={{ display: 'flex', borderRadius: 11, overflow: 'hidden', border: '1.5px solid #E4DDFA' }}>
                  {MATH_GRAPH_SIZE_PRESETS.map((s, i) => {
                    const on = i === sizeIdx
                    return (
                      <button key={s.label} onClick={() => setSizeIdx(i)}
                        style={{
                          flex: 1, padding: '7px 0', border: 'none', cursor: 'pointer',
                          borderRight: i < MATH_GRAPH_SIZE_PRESETS.length - 1 ? '1px solid #E4DDFA' : 'none',
                          background: on ? 'linear-gradient(135deg, #8B5CF6, #6D28D9)' : '#fff',
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

              {/* 坐标轴开关(批次3a:分数/角度等无坐标语义模板默认关) */}
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 15 }}>
                <span style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>显示坐标轴</span>
                <Toggle checked={showAxis} onChange={setShowAxis} />
              </div>

              {/* 网格开关 */}
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 15 }}>
                <span style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>显示坐标网格</span>
                <Toggle checked={showGrid} onChange={setShowGrid} />
              </div>

              {/* 标注 */}
              <div style={{ marginBottom: 8 }}>
                <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 7 }}>标注文字(可选)</div>
                <input
                  type="text"
                  value={caption}
                  onChange={e => setCaption(e.target.value)}
                  placeholder="如:拖动滑杆观察 k 对斜率的影响"
                  maxLength={60}
                  style={{ width: '100%', boxSizing: 'border-box', padding: '8px 11px', borderRadius: 10, border: '1.5px solid #E9E5F5', fontSize: 13, outline: 'none' }}
                />
              </div>
            </div>
          </div>
        </div>

        {/* ===== 底栏 ===== */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '13px 22px', borderTop: '1px solid ' + C.border, flexShrink: 0, background: '#FBFAFE' }}>
          <span style={{ fontSize: 12, color: C.textMuted }}>
            {isAI ? 'AI 生成的图形与模板同款视觉主题,融入后课堂同样可拖动交互' : '基于 JSXGraph 开源库(MIT/LGPL),生成的图形可离线运行、课堂可交互'}
          </span>
          <div style={{ marginLeft: 'auto', display: 'flex', gap: 10 }}>
            <button
              onClick={() => { if (!inserting) onClose() }}
              disabled={inserting}
              style={{ padding: '10px 22px', borderRadius: 12, border: '1.5px solid #E9E5F5', background: '#fff', color: C.textSecondary, fontSize: 14, fontWeight: 600, cursor: inserting ? 'not-allowed' : 'pointer' }}
            >取消</button>
            <button
              onClick={handleInsert}
              disabled={insertDisabled}
              title={isAI && !aiCode.trim() ? '先生成 AI 图形再融入' : undefined}
              style={{
                padding: '10px 26px', borderRadius: 12, border: 'none', fontSize: 14, fontWeight: 800,
                background: insertDisabled ? '#C4B5FD' : 'linear-gradient(135deg, #8B5CF6, #6D28D9)',
                boxShadow: insertDisabled ? 'none' : '0 6px 18px rgba(109,40,217,0.35)',
                color: '#fff', cursor: insertDisabled ? 'not-allowed' : 'pointer',
              }}
            >{inserting ? '⏳ AI 融入中…' : '📊 融入第 ' + pageNum + ' 页'}</button>
          </div>
        </div>
      </div>
    </div>
  )
}
