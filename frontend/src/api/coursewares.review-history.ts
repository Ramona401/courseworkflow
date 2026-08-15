/**
 * R-03 已审核记录只读详情 API。
 *
 * 与当前审核工作台严格分离：
 * - 主键是 courseware_review_id；
 * - 只读取正式审核时冻结的历史事实；
 * - 历史页面与当前页面分别返回；
 * - 不包含任何写操作函数。
 */

import apiClient from "./client";
import { extractData } from "./coursewares.types";

import type {
  CWAIReviewDimension,
  CWAIReviewLessonReferenceMode,
} from "./coursewares.ai-review-config";

import type {
  CWAIReviewSeverity,
} from "./coursewares.ai-review.types";

export interface CWReviewHistoryCourseware {
  id: string;
  title: string;
  subject: string;
  grade: string;
}

export interface CWReviewHistoryReviewer {
  id: string;
  display_name: string;
}

export interface CWReviewHistoryDecision {
  review_level: number;
  review_round: number;
  decision: string;
  score: number | null;
  comment: string;
  reviewed_at: string | null;
}

export interface CWReviewHistoryConfig {
  available: boolean;
  schema_version: number;
  dimensions: CWAIReviewDimension[];
  custom_focus: string;
  lesson_reference_mode: CWAIReviewLessonReferenceMode;
  lesson_materials_used: boolean | null;
  unavailable_reason: string;
}

export interface CWReviewHistoryTeacherView {
  teacher_title: string;
  what_happened: string;
  teaching_impact: string;
  improvement_goal: string;
  acceptance_checks: string[];
  teacher_context: string;
  manual_check_required: boolean;
}

export interface CWReviewHistoryDeliveredInstruction {
  version_id: string;
  version_no: number;
  content: string;
  source_type: string;
  confirmed_at: string | null;
}

export interface CWReviewHistoryModificationRecord {
  content: string;
  created_at: string | null;
}

export interface CWReviewHistoryIssue {
  id: string;

  page_id: string | null;
  page_number: number;
  page_title: string;

  severity: CWAIReviewSeverity;
  dimension: CWAIReviewDimension;

  teacher_view: CWReviewHistoryTeacherView;

  delivered_instruction_available: boolean;
  delivered_instruction: CWReviewHistoryDeliveredInstruction | null;
  delivered_instruction_unavailable_reason: string;

  previous_modification_records: CWReviewHistoryModificationRecord[];
}

export interface CWReviewHistoryPage {
  page_id: string;
  page_number: number;
  page_title: string;
  html_content: string;
  page_updated_at: string | null;
  current_exists: boolean;
}

export interface CWReviewHistoryCurrentPage {
  page_id: string;
  page_number: number;
  page_title: string;
  html_content: string;
  updated_at: string | null;
}

export interface CWReviewHistoryDetail {
  review_id: string;
  record_title: string;

  courseware: CWReviewHistoryCourseware;
  reviewer: CWReviewHistoryReviewer;
  review: CWReviewHistoryDecision;

  review_config: CWReviewHistoryConfig;

  issues_available: boolean;
  issues_unavailable_reason: string;
  issues: CWReviewHistoryIssue[];

  historical_pages_available: boolean;
  historical_pages_unavailable_reason: string;
  historical_pages: CWReviewHistoryPage[];

  current_pages: CWReviewHistoryCurrentPage[];
}

/**
 * 查询一条正式已审核记录的不可变只读详情。
 */
export async function getCWReviewHistoryDetail(
  reviewId: string,
): Promise<CWReviewHistoryDetail> {
  const response = await apiClient.get(
    `/courseware-reviews/records/${reviewId}/detail`,
  );

  return extractData<CWReviewHistoryDetail>(response);
}
