/**
 * cwAIReviewControllerHelpers.ts
 *
 * 课件AI审核控制器使用的纯函数。
 *
 * 本文件不持有React状态、不调用API：
 *   - 合并批次响应；
 *   - 合并整改项响应；
 *   - 从已完成批次中提取finding；
 *   - 判断正式整改项是否可以随退回决定交付。
 */

import {
  parseCWAIReviewBatchResult,
  type CWAIReviewBatch,
  type CWAIReviewFinding,
  type CWAIReviewItem,
} from "@/api/coursewares";

export function mergeCWAIReviewBatch(
  batches: CWAIReviewBatch[],
  incoming: CWAIReviewBatch | null,
): CWAIReviewBatch[] {
  if (!incoming) {
    return batches;
  }

  const exists = batches.some(
    (batch) => batch.id === incoming.id,
  );

  const next = exists
    ? batches.map((batch) =>
        batch.id === incoming.id
          ? incoming
          : batch,
      )
    : [...batches, incoming];

  return next.sort(
    (left, right) =>
      left.batch_no - right.batch_no,
  );
}

export function mergeCWAIReviewItems(
  current: CWAIReviewItem[],
  incoming: CWAIReviewItem[],
): CWAIReviewItem[] {
  const itemMap =
    new Map<string, CWAIReviewItem>();

  for (const item of current) {
    itemMap.set(item.id, item);
  }

  for (const item of incoming) {
    itemMap.set(item.id, item);
  }

  return Array.from(itemMap.values()).sort(
    (left, right) => {
      if (
        left.page_number_snapshot !==
        right.page_number_snapshot
      ) {
        return (
          left.page_number_snapshot -
          right.page_number_snapshot
        );
      }

      return (
        left.created_at || ""
      ).localeCompare(
        right.created_at || "",
      );
    },
  );
}

export function collectCWAIReviewBatchFindings(
  batches: CWAIReviewBatch[],
): CWAIReviewFinding[] {
  const findings: CWAIReviewFinding[] = [];

  for (const batch of batches) {
    const result =
      parseCWAIReviewBatchResult(batch);

    if (result?.findings) {
      findings.push(...result.findings);
    }
  }

  return findings;
}

export function isDeliverableFormalReviewItem(
  item:
    | CWAIReviewItem
    | null
    | undefined,
): boolean {
  return (
    !!item &&
    item.source_type === "formal" &&
    item.status === "confirmed" &&
    !!item.confirmed_instruction.trim() &&
    !item.courseware_review_id &&
    !item.feedback_id
  );
}
