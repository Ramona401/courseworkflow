/**
 * mathGraphTemplateShared.ts — 数学图形模板共享辅助(批次1c首发;1e sliderAttrs;1h makeLayout;2a-fix,2026-07-08)
 *
 * 各学段模板文件共用的小工具,从原单体注册表抽出,避免每个学段文件复制一份。
 *
 * 批次1h 画板版式规范(makeLayout,所有模板逐步就位):
 *   顶部控制区 = 滑杆靠左成列 + 读数靠右,垫半透明白背板(与网格分层,控件不再浮在格子上);
 *   底部提示区 = 灰字提示垫同款背板,不贴画板下缘;
 *   中部主体区 = 教学图形独占。坐标一律由 makeLayout 按 boundingBox 自动计算,不再手拍。
 *
 * 批次2a-fix(重要修复):slider(i) 原返回带外层括号的 "[[x1,y],[x2,y]]",
 *   与调用点惯用拼法 [ + slider(i) + , [min,v,max]] 组合后变成嵌套数组
 *   [[[x1,y],[x2,y]],[min,v,max]],JSXGraph 滑杆需三个平级父元素,导致
 *   "Can't create point with parent types 'object'" 渲染失败。
 *   现改为返回不带外层括号的 "[x1,y], [x2,y]",调用点拼出正确的
 *   [[x1,y],[x2,y],[min,v,max]]。调用点写法保持不变。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/mathGraphTemplateShared.ts
 * 依赖: mathGraphUtils.ts(仅类型)
 */
import type { MathParamValue } from './mathGraphUtils'

/** 数值插值辅助:限制3位小数防浮点长尾污染生成代码;布尔原样转字符串 */
export function n(v: MathParamValue): string {
  if (typeof v === 'boolean') return String(v)
  return String(Math.round(v * 1000) / 1000)
}

/** 滑杆通用外观(紫色主题;保留旧常量兼容既有模板引用) */
export const SLIDER_ATTRS = "snapWidth:0.1, strokeColor:'#7C3AED', fillColor:'#7C3AED', highline:{strokeColor:'#7C3AED'}, baseline:{strokeColor:'#C4B5FD'}, label:{fontSize:14}"

/**
 * 任意主题色滑杆外观生成器(批次1e)——底线浅灰、进度线与圆钮同色。
 * @param color 滑杆主色  @param snap 吸附步进(角度类用 1,长度类用 0.5)
 */
export function sliderAttrs(color: string, snap: number = 0.1): string {
  return 'snapWidth:' + snap
    + ", strokeColor:'" + color + "', fillColor:'" + color + "'"
    + ", highline:{strokeColor:'" + color + "', strokeWidth:3}"
    + ", baseline:{strokeColor:'#E5E7EB', strokeWidth:2}"
    + ', label:{fontSize:14}'
}

// ============================================================
// 批次1h:画板排版辅助
// ============================================================

/** makeLayout 返回的排版坐标计算器 */
export interface BoardLayout {
  /** 左侧安全 x(控制区/提示文字起点) */
  leftX: number
  /** 中部 x(读数区起点,约在横向中点) */
  midX: number
  /** 顶部第 i 行的 y 坐标(i 从 0 起,行距约 8% 画板高) */
  topY: (i: number) => number
  /** 底部提示行 y 坐标 */
  hintY: number
  /**
   * 第 i 行滑杆的两端点坐标串 "[x1,y], [x2,y]"(不带外层括号!)。
   * 调用点拼法:board.create('slider', [ + slider(i) + , [min,v,max]])
   * → 得到正确的三平级父元素 [[x1,y],[x2,y],[min,v,max]](见文件头 2a-fix)。
   */
  slider: (i: number) => string
  /** 顶部控制区背板构造代码(容纳 rows 行;半透明白+细边,layer 2 压在网格上、图形下) */
  panel: (rows: number) => string
  /** 底部提示区背板构造代码(单行高) */
  hintPanel: () => string
}

/**
 * 按 boundingBox 生成标准排版坐标(批次1h版式规范的唯一实现):
 * 边距/行距按画板宽高比例计算,任何尺寸的 boundingBox 都能得到一致观感。
 * 背板 layer=2(网格 layer1 之上、多边形 layer3/线 layer7/点 layer9 之下),
 * 控件与图形都压在背板上方,网格被背板柔和隔开——"控制面板浮于图上"的现代感。
 */
export function makeLayout(bb: [number, number, number, number]): BoardLayout {
  const [x1, y1, x2, y2] = bb
  const w = x2 - x1
  const h = y1 - y2
  const r = (v: number) => Math.round(v * 100) / 100
  const leftX = r(x1 + 0.045 * w)
  const midX = r(x1 + 0.5 * w)
  const topY = (i: number) => r(y1 - 0.085 * h - i * 0.08 * h)
  const hintY = r(y2 + 0.055 * h)
  const panelAttrs = "{fillColor:'#FFFFFF', fillOpacity:0.72, borders:{strokeColor:'#E9EDF4', strokeWidth:1, layer:2}, vertices:{visible:false}, fixed:true, highlight:false, layer:2}"
  const rect = (px1: number, py1: number, px2: number, py2: number) =>
    "board.create('polygon', [[" + px1 + ',' + py1 + '],[' + px2 + ',' + py1 + '],[' + px2 + ',' + py2 + '],[' + px1 + ',' + py2 + ']], ' + panelAttrs + ');'
  return {
    leftX,
    midX,
    topY,
    hintY,
    /* 2a-fix:不带外层括号(调用点自行包最外层数组) */
    slider: (i: number) => '[' + leftX + ', ' + topY(i) + '], [' + r(x1 + 0.3 * w) + ', ' + topY(i) + ']',
    panel: (rows: number) => rect(r(x1 + 0.02 * w), r(y1 - 0.03 * h), r(x2 - 0.02 * w), r(topY(rows - 1) - 0.05 * h)),
    hintPanel: () => rect(r(x1 + 0.02 * w), r(y2 + 0.115 * h), r(x2 - 0.02 * w), r(y2 + 0.015 * h)),
  }
}
