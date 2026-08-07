/**
 * 教学智能体编辑草稿统一出口。
 *
 * 具体职责已拆分为：
 *   - Schema：协议、限制、默认结构和ID；
 *   - Parse：sessionStorage及服务器数据恢复；
 *   - Normalize：保存前规范化和请求构造；
 *   - Validation：后端协议镜像校验。
 */

export * from "./coursewareAssistantDraftSchema";
export * from "./coursewareAssistantDraftParse";
export * from "./coursewareAssistantDraftNormalize";
export * from "./coursewareAssistantDraftValidation";
