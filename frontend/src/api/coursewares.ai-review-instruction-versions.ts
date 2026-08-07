/**
 * coursewares.ai-review-instruction-versions.ts
 *
 * 课件审核整改指令不可变版本API。
 *
 * 浏览器安全边界：
 *   1. 浏览器只读取后端公开的版本业务字段；
 *   2. 保存并确认只提交正文和最近读取到的当前版本ID；
 *   3. 创建人、确认人、来源、哈希、页面快照和版本号均由后端确定；
 *   4. 并发窗口使用同一个旧版本ID确认时，后端只允许一个请求成功；
 *   5. 正式整改作者只能读取审核提交时实际交付的版本。
 */

import apiClient from "./client";
import {
  extractData,
} from "./coursewares.types";
import type {
  CWAIReviewItemDiscussion,
} from "./coursewares.ai-review";

export type CWAIReviewInstructionVersionStatus =
  | "draft"
  | "confirmed"
  | "superseded"
  | "invalid_for_page";

export type CWAIReviewInstructionVersionSource =
  | "legacy_backfill"
  | "legacy_direct_update"
  | "manual"
  | "ai_candidate"
  | "global_discussion";

export interface CWAIReviewInstructionVersion {
  id: string;
  item_id: string;
  version_no: number;

  content: string;
  content_hash: string;
  source_type:
    CWAIReviewInstructionVersionSource;

  created_at: string | null;
  confirmed_at: string | null;

  page_snapshot_hash: string;
  status:
    CWAIReviewInstructionVersionStatus;

  is_current: boolean;
}

export interface CWAIReviewInstructionVersionListResponse {
  current_instruction_version_id:
    string | null;

  versions:
    CWAIReviewInstructionVersion[];

  total: number;
}

export interface CWAIReviewCurrentInstructionVersionResponse {
  current_instruction_version_id:
    string | null;

  version:
    CWAIReviewInstructionVersion | null;
}

export interface CWAIReviewInstructionVersionDetailResponse {
  version:
    CWAIReviewInstructionVersion;
}

/**
 * 读取当前参与者有权看到的全部版本。
 *
 * 正式整改作者只能得到交付版本；
 * 审核员和自审作者可以得到完整版本历史。
 */
export async function listCWAIReviewInstructionVersions(
  itemId: string,
): Promise<CWAIReviewInstructionVersionListResponse> {
  const response = await apiClient.get(
    `/courseware-ai-reviews/items/${itemId}/instruction-versions`,
  );

  return extractData<CWAIReviewInstructionVersionListResponse>(
    response,
  );
}

/**
 * 读取当前可见版本。
 *
 * 对正式整改作者，后端返回实际交付版本；
 * 对审核员或自审作者，后端返回当前确认版本。
 */
export async function getCurrentCWAIReviewInstructionVersion(
  itemId: string,
): Promise<CWAIReviewCurrentInstructionVersionResponse> {
  const response = await apiClient.get(
    `/courseware-ai-reviews/items/${itemId}/instruction-versions/current`,
  );

  return extractData<CWAIReviewCurrentInstructionVersionResponse>(
    response,
  );
}

/**
 * 读取指定历史版本。
 */
export async function getCWAIReviewInstructionVersion(
  itemId: string,
  versionId: string,
): Promise<CWAIReviewInstructionVersionDetailResponse> {
  const response = await apiClient.get(
    `/courseware-ai-reviews/items/${itemId}/instruction-versions/${versionId}`,
  );

  return extractData<CWAIReviewInstructionVersionDetailResponse>(
    response,
  );
}

/**
 * 原子创建并确认连续的新版本。
 *
 * expectedCurrentVersionId必须来自最近一次版本读取。
 * 首次确认时传空字符串。
 */
export async function confirmCWAIReviewInstructionVersion(
  itemId: string,
  instruction: string,
  expectedCurrentVersionId: string,
): Promise<CWAIReviewItemDiscussion> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/items/${itemId}/instruction-versions/confirm`,
    {
      instruction,
      expected_current_version_id:
        expectedCurrentVersionId,
    },
  );

  return extractData<CWAIReviewItemDiscussion>(
    response,
  );
}
