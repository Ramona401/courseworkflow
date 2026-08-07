/**
 * 课件工坊 API —— 多级审核层。
 *
 * 正式审核提交时可以携带：
 *
 *   - 已完成的课件AI审核会话ID；
 *   - 本轮新确认、需要交付作者的问题ID；
 *   - 审核员明确确认已经解决的上轮问题ID。
 *
 * 业务约束：
 *
 *   - 只有“退回修改”可以交付本轮新问题；
 *   - 通过审核时，本轮旧问题必须全部确认解决；
 *   - 继续退回时，可以只确认部分旧问题已经解决；
 *   - 首次审核没有历史整改问题时，可以不提交复审问题字段；
 *   - 前端不提交AI报告正文或问题状态；
 *   - 后端重新校验并在同一事务中保存全部结果。
 */

import apiClient from "./client";
import {
  extractData,
} from "./coursewares.types";

import type {
  CoursewareDetail,
} from "./coursewares.types";

import type {
  CoursewareAnnotation,
} from "./coursewares.collab";

// ==================== 类型定义 ====================

/** 审核决策请求（审核员L1/L2操作）。 */
export interface CWReviewDecisionRequest {
  decision:
    | "approved"
    | "revision";

  score?: number;
  comment: string;
  dimensions?: string;

  /**
   * 已完成的本轮课件AI审核会话。
   *
   * 仅退回修改时用于生成正式整体反馈快照。
   */
  ai_review_session_id?: string;

  /**
   * 本轮新确认、随退回结果交付作者的问题。
   *
   * 必须属于ai_review_session_id对应的会话。
   */
  review_item_ids?: string[];

  /**
   * 审核员本轮明确确认已经解决的上轮问题。
   *
   * 通过审核时必须包含本轮全部旧问题；
   * 继续退回时允许只包含其中一部分；
   * 当前轮次没有历史问题或没有确认任何问题时可以省略。
   */
  resolved_review_item_ids?: string[];
}

/** 审核记录列表项。 */
export interface CWReviewListItem {
  id: string;
  courseware_id: string;
  review_level: number;
  level_name: string;
  reviewer_id: string;
  reviewer_name: string;
  decision: string;
  score: number | null;
  comment: string;
  review_round: number;
  created_at: string;
}

/** 审核历史响应。 */
export interface CWReviewHistoryResponse {
  reviews: CWReviewListItem[];
  total: number;
  current_level: number;
}

/** 待审核列表项。 */
export interface CWPendingReviewItem {
  courseware_id: string;
  title: string;
  subject: string;
  grade: string;
  page_count: number;
  source_type: string;
  source_name: string;
  author_id: string;
  author_name: string;
  school_name: string;
  review_level: number;
  level_name: string;
  submitted_at: string;
}

/** 待审核列表响应。 */
export interface CWPendingReviewListResponse {
  items: CWPendingReviewItem[];
  total: number;
}

/** 审核统计响应。 */
export interface CWReviewStatsResponse {
  total_pending: number;
  total_reviewed: number;
  total_approved: number;
  total_revision: number;
}

/** 已审核记录列表项。 */
export interface CWReviewedListItem {
  id: string;
  courseware_id: string;
  courseware_title: string;
  subject: string;
  grade: string;
  author_name: string;
  review_level: number;
  level_name: string;
  reviewer_name: string;
  decision: string;
  score: number | null;
  comment: string;
  created_at: string;
}

/** 已审核记录响应。 */
export interface CWReviewedListResponse {
  items: CWReviewedListItem[];
  total: number;
}

/**
 * 当前级别、当前轮次需要审核员复查的历史正式问题。
 *
 * 不包含原审核员、作者和内部AI会话身份。
 */
export interface CWReviewCarryoverItem {
  id: string;

  original_review_level: number;
  original_review_round: number;

  pending_review_level: number;
  pending_review_round: number;

  page_id: string | null;
  page_number_snapshot: number;
  page_title_snapshot: string;
  page_html_hash: string;
  page_updated_at_snapshot: string | null;

  severity: string;
  dimension: string;

  title: string;
  description: string;
  confirmed_instruction: string;

  status: string;
  applied_page_hash: string;

  confirmed_at: string | null;
  applied_at: string | null;
  resubmitted_at: string | null;
}

/** 审核详情响应。 */
export interface CWReviewDetailResponse {
  courseware: CoursewareDetail;
  annotations: CoursewareAnnotation[];
  reviews: CWReviewListItem[];

  pending_review_round: number;
  carryover_items: CWReviewCarryoverItem[];
}

// ==================== API函数 ====================

/** 作者首次提交或退回后重新提交课件审核。 */
export async function submitCoursewareForReview(
  coursewareId: string,
): Promise<{
  message: string;
}> {
  const response =
    await apiClient.post(
      `/coursewares/${coursewareId}/submit-review`,
      {},
    );

  return extractData(response);
}

/** L1教研组审核。 */
export async function reviewCWL1(
  coursewareId: string,
  request: CWReviewDecisionRequest,
): Promise<{
  message: string;
}> {
  const response =
    await apiClient.post(
      `/courseware-reviews/${coursewareId}/l1`,
      request,
    );

  return extractData<{
    message: string;
  }>(response);
}

/** L2学校审核。 */
export async function reviewCWL2(
  coursewareId: string,
  request: CWReviewDecisionRequest,
): Promise<{
  message: string;
}> {
  const response =
    await apiClient.post(
      `/courseware-reviews/${coursewareId}/l2`,
      request,
    );

  return extractData<{
    message: string;
  }>(response);
}

/** 查询审核历史。 */
export async function getCWReviewHistory(
  coursewareId: string,
): Promise<CWReviewHistoryResponse> {
  const response =
    await apiClient.get(
      `/courseware-reviews/${coursewareId}/history`,
    );

  return extractData<CWReviewHistoryResponse>(
    response,
  );
}

/** 查询待审课件、历史审核和本轮旧问题。 */
export async function getCWReviewDetail(
  coursewareId: string,
): Promise<CWReviewDetailResponse> {
  const response =
    await apiClient.get(
      `/courseware-reviews/${coursewareId}/detail`,
    );

  return extractData<CWReviewDetailResponse>(
    response,
  );
}

/** 查询当前用户可审核的待审课件。 */
export async function getCWPendingReviews(
  params?: {
    limit?: number;
    offset?: number;
  },
): Promise<CWPendingReviewListResponse> {
  const response =
    await apiClient.get(
      "/courseware-reviews/pending",
      {
        params,
      },
    );

  return extractData<CWPendingReviewListResponse>(
    response,
  );
}

/** 查询已审核记录。 */
export async function getCWReviewedRecords(
  params: {
    level: number;
    decision?: string;
    limit?: number;
    offset?: number;
  },
): Promise<CWReviewedListResponse> {
  const response =
    await apiClient.get(
      "/courseware-reviews/reviewed",
      {
        params,
      },
    );

  return extractData<CWReviewedListResponse>(
    response,
  );
}

/** 查询审核统计。 */
export async function getCWReviewStats(
  level?: number,
): Promise<CWReviewStatsResponse> {
  const response =
    await apiClient.get(
      "/courseware-reviews/stats",
      {
        params: level
          ? {
              level,
            }
          : {},
      },
    );

  return extractData<CWReviewStatsResponse>(
    response,
  );
}

// ==================== 审核级别常量 ====================

export const CW_REVIEW_LEVEL_NAMES:
  Record<number, string> = {
  0: "未提交",
  1: "教研组审核",
  2: "学校审核",
};

export const CW_REVIEW_LEVEL_COLORS:
  Record<number, string> = {
  0: "#9CA3AF",
  1: "#F59E0B",
  2: "#EF4444",
};
