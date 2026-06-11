/**
 * 课件工坊 API —— 字体方案层 (coursewares.font.ts)（字体F2新建）
 *
 * 字体方案：一套 = 标题字体 + 正文字体的固定搭配，5套系统预设（全OFL开源协议）。
 * 课件选中后，后端把方案code写入 coursewares.font_scheme，并对已生成页做"秒换"
 * （纯字符串替换注入的 TEDNA-TPL-FONT 字体<style>块，零AI调用零token，秒级完成）。
 * 后续生成/重生/微调的页面由后端确定性注入，自动带上所选字体。
 * 经桶文件 coursewares.ts 透出，对外 import 路径不变。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'

// ==================== 类型 ====================

/** 一条 @font-face 声明的数据（前端用 base_url + file 现场加载渲染字样预览） */
export interface CWFontFace {
  family: string // CSS font-family 内部别名（如 TednaSans）
  file: string   // woff2 文件名
  weight: string // font-weight（"400"/"600"）
}

/** 字体方案（5套系统预设常量，后端硬编码不建表） */
export interface CWFontScheme {
  code: string            // 方案code（存库值，如 modern/serif/wenkai/smiley/english）
  name: string            // 方案名（如"现代清爽"）
  description: string     // 一句话描述
  heading_label: string   // 标题字体中文名
  body_label: string      // 正文字体中文名
  heading_family: string  // 标题 font-family 栈（自有字体部分）
  body_family: string     // 正文 font-family 栈
  faces: CWFontFace[]     // 需要的 @font-face 列表
}

/** 课件当前字体选择（空串=未选，跟随模板默认字体） */
export interface CWFontSelection {
  font_scheme: string
}

/** 选择/清除字体的执行结果 */
export interface CWFontResult {
  font_scheme: string    // 生效后的方案code（清除后为空串）
  swapped_pages: number  // 被秒换字体的已生成页数（零token字符串操作）
}

// ==================== API ====================

/** 字体方案列表（5套系统预设 + 字体文件公网基地址） */
export async function listFontSchemes(): Promise<{ schemes: CWFontScheme[]; total: number; base_url: string }> {
  const resp = await apiClient.get('/courseware-fonts')
  return extractData(resp)
}

/** 查询课件当前字体选择 */
export async function getCWFont(coursewareId: string): Promise<CWFontSelection> {
  const resp = await apiClient.get('/coursewares/' + coursewareId + '/font')
  return extractData(resp)
}

/** 课件选用某字体方案（写code快照 + 秒换全部已生成页）。秒换是字符串操作但页多时仍给足超时 */
export async function setCWFont(coursewareId: string, schemeCode: string): Promise<CWFontResult> {
  const resp = await apiClient.put(
    '/coursewares/' + coursewareId + '/font',
    { scheme_code: schemeCode },
    { timeout: 30000 },
  )
  return extractData(resp)
}

/** 清除课件字体选择（已生成页移除字体注入块，回退到模板自带字体） */
export async function clearCWFont(coursewareId: string): Promise<CWFontResult> {
  const resp = await apiClient.put(
    '/coursewares/' + coursewareId + '/font',
    { clear: true },
    { timeout: 30000 },
  )
  return extractData(resp)
}
