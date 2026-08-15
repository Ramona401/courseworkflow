/**
 * coursewares.ai-review-comment-candidates.ts
 *
 * R-08 正式课件审核意见重新汇总候选API。
 *
 * 安全边界：
 *   - Generate只提交教师当前本次修改清单ID和教师自己的当前审核意见；
 *   - 当前整改要求、指令版本、R-06问题组及其version全部由后端重新读取；
 *   - Apply只提交candidate ID、replace/append动作、当前清单ID和当前教师意见；
 *   - 浏览器绝不能把candidate_text、diff、input_hash或服务端事实快照回传为事实源；
 *   - Apply成功只返回新的输入框文本，不代表正式审核已经提交。
 */

import apiClient from "./client";
import { extractData } from "./coursewares.types";

export type CWReviewCommentCandidateApplyAction = "replace" | "append";

export interface CWReviewCommentDiffAdjustment {
  before: string;
  after: string;
}

export interface CWReviewCommentDiff {
  added: string[];
  removed: string[];
  adjusted: CWReviewCommentDiffAdjustment[];
}

export interface CWReviewCommentCandidate {
  id: string;
  courseware_id: string;
  source_session_id: string;
  review_level: number;

  candidate_schema_version: number;

  original_comment: string;
  candidate_text: string;

  diff_schema_version: number;
  diff: CWReviewCommentDiff;

  created_at: string;
}

export interface GenerateCWReviewCommentCandidateRequest {
  selected_item_ids: string[];
  original_comment: string;
}

export interface ApplyCWReviewCommentCandidateRequest {
  action: CWReviewCommentCandidateApplyAction;
  selected_item_ids: string[];
  current_comment: string;
}

export interface ApplyCWReviewCommentCandidateResponse {
  candidate_id: string;
  action: CWReviewCommentCandidateApplyAction;
  next_comment: string;
}

/**
 * 根据当前服务端可信整改事实生成一份新的不可变审核意见候选。
 *
 * AI调用可能耗时，因此与R-07生成Draft一致使用5分钟请求超时。
 */
export async function generateCWReviewCommentCandidate(
  sessionId: string,
  request: GenerateCWReviewCommentCandidateRequest,
): Promise<CWReviewCommentCandidate> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/comment-candidates`,
    {
      selected_item_ids: request.selected_item_ids,
      original_comment: request.original_comment,
    },
    {
      timeout: 300000,
    },
  );

  return extractData<CWReviewCommentCandidate>(response);
}

/**
 * 教师明确选择替换或追加候选。
 *
 * candidate正文不在请求中；后端根据URL candidate ID重读冻结正文，
 * 并重新核验当前清单、指令版本、问题组及教师意见是否仍与生成时一致。
 */
export async function applyCWReviewCommentCandidate(
  sessionId: string,
  candidateId: string,
  request: ApplyCWReviewCommentCandidateRequest,
): Promise<ApplyCWReviewCommentCandidateResponse> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/comment-candidates/${candidateId}/apply`,
    {
      action: request.action,
      selected_item_ids: request.selected_item_ids,
      current_comment: request.current_comment,
    },
  );

  return extractData<ApplyCWReviewCommentCandidateResponse>(response);
}
