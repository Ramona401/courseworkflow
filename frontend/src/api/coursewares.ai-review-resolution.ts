/**
 * coursewares.ai-review-resolution.ts
 *
 * 课件整改问题的人工确认与作者执行补充接口。
 *
 * 当前包含三类人工动作：
 *
 * 一、作者自审问题确认解决
 *   - 问题必须已经完成页面修改；
 *   - 作者必须实际检查当前页面效果；
 *   - 后端会再次比较当前页面和修改完成时的内容指纹；
 *   - 正式审核问题不能通过该动作关闭。
 *
 * 二、页面变化问题重新检查
 *   - 适用于自审或已交付作者的正式整改问题；
 *   - 问题必须处于stale；
 *   - 后端重新读取当前页面并保存当前内容指纹；
 *   - 状态回到applied；
 *   - 自审问题继续等待作者最终确认；
 *   - 正式问题继续等待审核员复审；
 *   - 本动作不会自动把问题标记为resolved。
 *
 * 三、作者正式整改执行补充
 *   - 只用于已经正式交付作者的formal整改项；
 *   - 只保存教师填写的本次执行说明；
 *   - 不调用AI、不改确认要求、不改页面、不改变整改状态。
 */

import apiClient from "./client";
import { extractData } from "./coursewares.types";

import type {
  CWAIReviewItem,
  CWAIReviewItemDiscussion,
  CWAIReviewItemMessage,
} from "./coursewares.ai-review";

/** 整改问题人工状态操作响应。 */
export interface CWAIReviewItemManualStateResponse {
  item: CWAIReviewItem;
  messages: CWAIReviewItemMessage[];
  message: string;
}

/**
 * 作者检查当前课件后，明确确认自己的自审问题已经解决。
 */
export async function resolveSelfCWAIReviewItem(
  itemId: string,
): Promise<CWAIReviewItemManualStateResponse> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/items/${itemId}/resolve`,
    {},
  );

  return extractData<CWAIReviewItemManualStateResponse>(response);
}

/**
 * 作者重新检查发生变化的页面。
 *
 * 当前页面由后端重新读取，浏览器不提供HTML或内容指纹。
 */
export async function recheckCWAIReviewItem(
  itemId: string,
): Promise<CWAIReviewItemManualStateResponse> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/items/${itemId}/recheck`,
    {},
  );

  return extractData<CWAIReviewItemManualStateResponse>(response);
}

/**
 * 作者为已经正式交付的整改项补充本次实际执行情况。
 *
 * 该文字只是过程记录，不会调用AI，也不会改变正式审核员确认的修改要求或问题状态。
 */
export async function addCWAIReviewItemExecutionNote(
  itemId: string,
  content: string,
): Promise<CWAIReviewItemDiscussion> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/items/${itemId}/execution-note`,
    {
      content,
    },
  );

  return extractData<CWAIReviewItemDiscussion>(response);
}
