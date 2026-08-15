/**
 * coursewares.ai-review-governance.ts
 *
 * 课件AI审核全局讨论结论落地、问题列表治理与R-07结构化影响方案API。
 *
 * 安全边界：
 *   1. 全局讨论人工新增问题必须绑定已经保存的可信assistant消息；
 *   2. AI建议关系的说明由后端重新读取，浏览器不能替换；
 *   3. 问题清单直接关系由用户明确选择类型、方向并填写说明；
 *   4. 关系确认、取消和AI建议忽略均为独立人工动作；
 *   5. 所有动作都不会自动确认候选指令、修改页面或提交审核决定；
 *   6. 取消关系和确认忽略必须填写可追溯原因；
 *   7. R-07最终Apply只提交version和selected_operation_ids，不回传Preview payload。
 */

import apiClient from "./client";
import { extractData } from "./coursewares.types";

import type {
  CWAIReviewGlobalDiscussion,
  CWAIReviewGlobalRelationType,
  CWAIReviewImpactPlan,
  CWAIReviewItem,
  CWAIReviewItemDiscussion,
  CWAIReviewItemRelation,
  CWAIReviewSeverity,
} from "./coursewares.ai-review";

export interface CreateCWAIReviewGlobalManualItemsRequest {
  message_id: string;

  title: string;
  description: string;
  candidate_instruction: string;

  severity: CWAIReviewSeverity;
  dimension: string;

  /**
   * 空数组表示整课问题；多个页面会由后端拆成多条页级整改项。
   */
  page_ids: string[];
}

export interface CreateCWAIReviewGlobalManualItemsResponse {
  items: CWAIReviewItem[];
  message?: string;
}

export interface CWAIReviewGlobalRelationListResponse {
  relations: CWAIReviewItemRelation[];
}

/**
 * 问题清单中直接建立关系的明确输入。
 *
 * explanation由用户人工填写，不依赖AI全局讨论消息。
 */
export interface ConfirmCWAIReviewManualRelationRequest {
  relation_type: CWAIReviewGlobalRelationType;
  source_item_id: string;
  target_item_id: string;
  explanation: string;
}

/**
 * 从可信全局讨论消息人工新增整改项。
 *
 * candidate_instruction只是候选，创建后仍需进入单条整改项独立确认。
 */
export async function createCWAIReviewGlobalManualItems(
  sessionId: string,
  request: CreateCWAIReviewGlobalManualItemsRequest,
): Promise<CreateCWAIReviewGlobalManualItemsResponse> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/global-discussion/manual-items`,
    request,
  );

  return extractData<CreateCWAIReviewGlobalManualItemsResponse>(
    response,
  );
}

/**
 * 读取当前会话创建者已经明确确认的全部关系及追加式事件历史。
 *
 * 结果同时包含从AI建议确认的关系和从问题清单直接建立的关系。
 */
export async function getCWAIReviewItemRelations(
  sessionId: string,
): Promise<CWAIReviewGlobalRelationListResponse> {
  const response = await apiClient.get(
    `/courseware-ai-reviews/${sessionId}/relations`,
  );

  return extractData<CWAIReviewGlobalRelationListResponse>(
    response,
  );
}

/**
 * 在问题清单中直接建立或重新启用一条人工关系。
 */
export async function confirmCWAIReviewManualRelation(
  sessionId: string,
  request: ConfirmCWAIReviewManualRelationRequest,
): Promise<CWAIReviewItemRelation> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/relations`,
    request,
  );

  return extractData<CWAIReviewItemRelation>(
    response,
  );
}

/**
 * 通过会话级治理接口取消一条关系。
 *
 * 适用于AI建议确认关系和问题清单直接建立关系。
 */
export async function cancelCWAIReviewItemRelation(
  sessionId: string,
  relationId: string,
  reason: string,
): Promise<CWAIReviewItemRelation> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/relations/${relationId}/cancel`,
    {
      reason,
    },
  );

  return extractData<CWAIReviewItemRelation>(
    response,
  );
}

/**
 * 读取全局讨论治理入口中的已确认关系。
 *
 * 保留该接口用于现有全局讨论组件兼容。
 */
export async function getCWAIReviewGlobalRelations(
  sessionId: string,
): Promise<CWAIReviewGlobalRelationListResponse> {
  const response = await apiClient.get(
    `/courseware-ai-reviews/${sessionId}/global-discussion/relations`,
  );

  return extractData<CWAIReviewGlobalRelationListResponse>(
    response,
  );
}

/**
 * 明确确认AI可信消息中的一条关系。
 *
 * explanation由后端从可信消息重新读取，浏览器不能替换。
 */
export async function confirmCWAIReviewGlobalRelation(
  sessionId: string,
  messageId: string,
  relationType: CWAIReviewGlobalRelationType,
  sourceItemId: string,
  targetItemId: string,
): Promise<CWAIReviewItemRelation> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/global-discussion/relations`,
    {
      message_id: messageId,
      relation_type: relationType,
      source_item_id: sourceItemId,
      target_item_id: targetItemId,
    },
  );

  return extractData<CWAIReviewItemRelation>(
    response,
  );
}

/**
 * 通过全局讨论兼容接口取消一条已确认关系。
 *
 * 新的问题清单治理界面使用会话级取消接口。
 */
export async function cancelCWAIReviewGlobalRelation(
  sessionId: string,
  relationId: string,
  reason: string,
): Promise<CWAIReviewItemRelation> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/global-discussion/relations/${relationId}`,
    {
      reason,
    },
  );

  return extractData<CWAIReviewItemRelation>(
    response,
  );
}

/**
 * 独立确认可信AI的consider_dismiss建议。
 *
 * 最终仍复用单条整改项忽略边界，返回更新后的单条讨论。
 */
export async function confirmCWAIReviewGlobalDismissal(
  sessionId: string,
  messageId: string,
  itemId: string,
  reason: string,
): Promise<CWAIReviewItemDiscussion> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/global-discussion/dismiss`,
    {
      message_id: messageId,
      item_id: itemId,
      reason,
    },
  );

  return extractData<CWAIReviewItemDiscussion>(
    response,
  );
}

/**
 * R-07：从可信全局讨论assistant消息生成结构化影响方案Draft。
 *
 * 浏览器只提交message_id；AI正文、operation payload与preconditions均由后端重建和冻结。
 */
export async function createCWAIReviewImpactPlan(
  sessionId: string,
  messageId: string,
): Promise<CWAIReviewImpactPlan> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/impact-plans`,
    {
      message_id: messageId,
    },
    {
      timeout: 300000,
    },
  );

  return extractData<CWAIReviewImpactPlan>(
    response,
  );
}

/**
 * R-07：按URL中的plan_id读取已冻结的教师Preview。
 */
export async function getCWAIReviewImpactPlan(
  sessionId: string,
  planId: string,
): Promise<CWAIReviewImpactPlan> {
  const response = await apiClient.get(
    `/courseware-ai-reviews/${sessionId}/impact-plans/${planId}`,
  );

  return extractData<CWAIReviewImpactPlan>(
    response,
  );
}

/**
 * R-07：教师一次确认并原子应用明确勾选的operation。
 *
 * 最终正文固定只有version + selected_operation_ids。
 * 不允许把Preview payload、AI正文、preconditions或身份字段回传。
 */
export async function applyCWAIReviewImpactPlan(
  sessionId: string,
  planId: string,
  version: number,
  selectedOperationIds: string[],
): Promise<CWAIReviewImpactPlan> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/impact-plans/${planId}/apply`,
    {
      version,
      selected_operation_ids: selectedOperationIds,
    },
  );

  return extractData<CWAIReviewImpactPlan>(
    response,
  );
}

/**
 * 规范后端可能返回的空数组，避免治理视图重复防御。
 */
export function normalizeCWAIReviewGlobalDiscussionGovernance(
  discussion: CWAIReviewGlobalDiscussion,
): CWAIReviewGlobalDiscussion {
  return {
    ...discussion,
    governance_relations:
      discussion.governance_relations || [],
  };
}
