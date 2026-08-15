/**
 * coursewares.page-mutation.ts
 *
 * 课件单页AI修改、重新生成、页面历史与回退API。
 *
 * 从coursewares.core.ts按页级写入职责拆出，避免核心API文件继续超过900行。
 *
 * 整改项页面应用安全边界：
 *   1. 普通页面微调不需要整改项；
 *   2. 携带review_item_id时，前端先读取后端认可的当前可执行修改要求；
 *   3. 正式整改作者读取实际交付要求，自审作者读取当前确认方案；
 *   4. 请求同时提交review_item_id和服务端当前要求身份；
 *   5. 读取后若发生并发切换，后端会拒绝旧请求；
 *   6. 当前草稿必须完整包含可信修改要求，否则不进入耗时AI页面修改。
 */

import apiClient from "./client";
import {
  getCurrentCWAIReviewInstructionVersion,
} from "./coursewares.ai-review-instruction-versions";
import {
  extractData,
} from "./coursewares.types";

import type {
  PageVersionEntry,
} from "./coursewares.types";

export type CWRefineMode =
  | "preserve"
  | "rebuild";

export interface CWRefinePageResponse {
  page_number: number;
  html_content: string;
  message: string;
  mode?: CWRefineMode;

  review_item_id?: string;
  instruction_version_id?: string;

  review_item_status?: string;
  review_item_warning?: string;
}

/**
 * 单页AI微调。
 *
 * reviewItemId非空时，必须先解析当前可执行的已确认修改要求。
 * 该读取不会替代后端事务校验，只用于形成明确的并发身份和提前反馈。
 */
export async function refinePage(
  coursewareId: string,
  pageNumber: number,
  instruction: string,
  image?: string,
  mode:
    CWRefineMode =
      "preserve",
  reviewItemId?: string,
): Promise<CWRefinePageResponse> {
  const normalizedInstruction =
    instruction.trim();

  const normalizedReviewItemID =
    reviewItemId?.trim() ||
    "";

  if (
    !normalizedInstruction &&
    !image
  ) {
    throw new Error(
      "请提供修改意见或粘贴截图",
    );
  }

  const body:
    Record<string, string> = {
      instruction,
      mode,
    };

  if (image) {
    body.image = image;
  }

  if (
    normalizedReviewItemID
  ) {
    const versionResult =
      await getCurrentCWAIReviewInstructionVersion(
        normalizedReviewItemID,
      );

    const version =
      versionResult.version;

    const instructionVersionID =
      versionResult
        .current_instruction_version_id
        ?.trim() ||
      version?.id.trim() ||
      "";

    if (
      !version ||
      !instructionVersionID
    ) {
      throw new Error(
        "这条问题还没有可执行的已确认修改要求，请刷新整改中心后重试",
      );
    }

    if (
      version.status !==
      "confirmed"
    ) {
      throw new Error(
        "当前修改要求已经不能直接执行，请刷新整改中心后重新检查",
      );
    }

    const trustedInstruction =
      version.content.trim();

    if (
      !trustedInstruction ||
      !normalizedInstruction.includes(
        trustedInstruction,
      )
    ) {
      throw new Error(
        "当前微调草稿没有完整包含已确认的修改要求，请重新从整改中心带入后再执行",
      );
    }

    body.review_item_id =
      normalizedReviewItemID;

    body.instruction_version_id =
      instructionVersionID;
  }

  const response =
    await apiClient.post(
      `/coursewares/${coursewareId}/pages/${pageNumber}/refine`,
      body,
      {
        timeout: 300000,
      },
    );

  return extractData<
    CWRefinePageResponse
  >(response);
}

/**
 * 单页从零重新生成。
 *
 * 本入口不是整改项应用入口，不自动绑定review_item_id。
 */
export async function regenerateCWPage(
  coursewareId: string,
  pageNumber: number,
): Promise<{
  page_number: number;
  html_content: string;
  message: string;
}> {
  const response =
    await apiClient.post(
      `/coursewares/${coursewareId}/pages/${pageNumber}/regenerate`,
      {},
      {
        timeout: 300000,
      },
    );

  return extractData(response);
}

/** 获取某页版本快照列表。 */
export async function listPageVersions(
  coursewareId: string,
  pageNumber: number,
): Promise<{
  page_number: number;
  versions:
    PageVersionEntry[];
  total: number;
}> {
  const response =
    await apiClient.get(
      `/coursewares/${coursewareId}/pages/${pageNumber}/versions`,
    );

  return extractData(response);
}

/**
 * 回退某页到指定历史版本。
 *
 * 后端会先保存当前页面，因此回退操作本身可逆。
 */
export async function rollbackPage(
  coursewareId: string,
  pageNumber: number,
  versionId: string,
): Promise<{
  page_number: number;
  html_content: string;
  message: string;
}> {
  const response =
    await apiClient.post(
      `/coursewares/${coursewareId}/pages/${pageNumber}/rollback`,
      {
        version_id:
          versionId,
      },
    );

  return extractData(response);
}

/** 获取指定页面历史版本的完整HTML。 */
export async function getPageVersionDetail(
  coursewareId: string,
  pageNumber: number,
  versionId: string,
): Promise<{
  page_number: number;
  version_id: string;
  version_no: number;
  source: string;
  source_label: string;
  html_content: string;
}> {
  const response =
    await apiClient.get(
      `/coursewares/${coursewareId}/pages/${pageNumber}/versions/${versionId}`,
    );

  return extractData(response);
}
