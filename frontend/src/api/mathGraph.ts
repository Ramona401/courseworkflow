/**
 * mathGraph.ts — 数学动态图形 AI 定制端点封装
 * (批次A,2026-07-08;批次A+拍题出图;单请求超时300s,2026-07-09)
 *
 * 对应后端 POST /api/v1/math-graph/generate(math_graph_handler.go)。
 * 多模式单接口:
 *   - adapt(模板变种):base_code = 模板当前 buildConstruction 产出(或上一轮 AI 代码,
 *     即"追改"),description = 老师要求的变化,AI 做有底稿的最小改写;
 *   - create(从零生成):仅 description,AI 从零产出构造代码;
 *   - 批次A+·拍题出图:任一模式可携带 image(题目照片 data URI),后端走多模态直接读题
 *     (文字+题目配图一并理解),description 变为可选补充说明。图片仅首轮携带,追改不带图。
 * 返回可直接执行的 JSXGraph 构造代码(操作 board 变量的纯 JS,前端执行前统一过
 * applyMathPalette 换装珊瑚粉主题)。结果不落库,弹窗会话级持有。
 *
 * 超时说明(2026-07-09生产实测):拍题多模态一次生成可达 3 分钟(gemini 读图 + 2万+ token),
 * 超过 client 全局 120s 超时导致前端报"请求超时"而后端实际成功(积分已扣、结果被丢)。
 * 故本请求单独覆写 timeout=300s(后端 AI 总超时 900s,留足余量)。
 * 范式先例:pipelines 大数据量 pages 请求同款单请求超时覆写。
 *
 * 封装范式对齐 lesson-plans-ref.ts(apiClient.post + resp.data.data)。
 */
import apiClient from './client'

/** 生成请求单独超时:300 秒(拍题多模态实测可达 3 分钟,全局 120s 不够) */
const GENERATE_TIMEOUT_MS = 300000

/** AI 生成/改编请求体 */
export interface MathGraphGenerateRequest {
  /** 模式:adapt 模板变种(带底稿) / create 从零生成 */
  mode: 'adapt' | 'create'
  /** 老师的自然语言描述(带图时可为空串,作为补充说明) */
  description: string
  /** 底稿构造代码(adapt 模式必填) */
  base_code?: string
  /** 底稿模板名称(adapt 模式可选,帮 AI 理解教学场景) */
  template_name?: string
  /** 画板坐标范围字符串,如 "[-10, 8, 10, -6]"(可选,后端有默认值) */
  bounding_box?: string
  /** 批次A+:题目图片 data URI(data:image/jpeg;base64,xxx,可选,拍题出图用) */
  image?: string
}

/** AI 生成/改编响应体 */
export interface MathGraphGenerateResponse {
  /** 可直接执行的 JSXGraph 构造代码(操作 board 变量的纯 JS) */
  code: string
}

/** AI 生成/改编 JSXGraph 构造代码(拍题多模态可达 3 分钟,单请求超时 300s) */
export async function generateMathGraphCode(
  data: MathGraphGenerateRequest
): Promise<MathGraphGenerateResponse> {
  const resp = await apiClient.post('/math-graph/generate', data, { timeout: GENERATE_TIMEOUT_MS })
  return resp.data.data as MathGraphGenerateResponse
}
