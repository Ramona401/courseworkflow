/**
 * CWAIReviewGlobalGovernanceLimits.ts
 *
 * 全局讨论结论落地与问题治理前端输入边界。
 *
 * 设计原则：
 *   1. 标题、描述、问题维度、页面数量和治理原因与后端硬边界保持一致；
 *   2. 所有提交操作在浏览器端按Unicode字符数复核，后端仍是最终安全边界；
 *   3. 候选指令保留现有表单4000字输入边界，不改变既有业务容量；
 *   4. 本文件只提供常量和纯函数，不访问接口、不修改状态。
 */

export const CW_GLOBAL_MANUAL_ITEM_TITLE_MAX_RUNES = 300;

export const CW_GLOBAL_MANUAL_ITEM_DESCRIPTION_MAX_RUNES = 6000;

export const CW_GLOBAL_MANUAL_ITEM_DIMENSION_MAX_RUNES = 64;

export const CW_GLOBAL_MANUAL_ITEM_INSTRUCTION_INPUT_MAX_RUNES = 4000;

export const CW_GLOBAL_MANUAL_ITEM_MAX_PAGES = 100;

export const CW_GLOBAL_GOVERNANCE_REASON_MAX_RUNES = 500;

/**
 * 按Unicode字符统计长度。
 *
 * 使用Array.from避免中文、普通Emoji等输入被JavaScript UTF-16长度误判。
 * 后端继续使用utf8.RuneCountInString执行最终复核。
 */
export function countCWGlobalRunes(value: string): number {
  return Array.from(value).length;
}
