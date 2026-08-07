/**
 * CWOwnerReviewItemsByPage.tsx
 *
 * 作者整改中心中的整改项展示边界。
 *
 * 本组件不再自行维护“页级问题”和“整课问题”两套展示逻辑，而是把全部问题交给
 * CWAIReviewPageChecklist 统一完成稳定页面聚合、整课分组、进度统计、完成项折叠和下一条任务引导。
 *
 * 业务边界保持不变：
 *   - 页级问题可以定位页面，并在满足状态条件时注入页面微调；
 *   - 整课问题没有唯一目标页面，仍不能注入单页微调；
 *   - 页面修改成功只进入 applied，正式整改最终仍由审核员复审确认；
 *   - 作者整改模式不参与正式审核的“本次退回”选择。
 */

import type { CWAIReviewItem } from "@/api/coursewares";

import CWAIReviewPageChecklist from "@/pages/courseware/review/CWAIReviewPageChecklist";

const C = {
  textMuted: "#94A3B8",
};

/**
 * 作者整改模式不使用审核交付选择。
 */
function ignoreDeliverySelection(): void {
  // 仅用于满足共享清单的统一接口，不产生任何业务副作用。
}

export interface CWOwnerReviewItemsByPageProps {
  items: CWAIReviewItem[];
  emptyMessage?: string;
  onSelectPage: (pageNumber: number) => void;
  onChanged: (item: CWAIReviewItem) => void;
  onInjectToRefine: (item: CWAIReviewItem) => void;
}

export default function CWOwnerReviewItemsByPage({
  items,
  emptyMessage = "暂无需要处理的整改项。",
  onSelectPage,
  onChanged,
  onInjectToRefine,
}: CWOwnerReviewItemsByPageProps) {
  if (items.length === 0) {
    return (
      <div
        style={{
          marginTop: "8px",
          padding: "18px 12px",
          color: C.textMuted,
          fontSize: "14px",
          lineHeight: 1.7,
          textAlign: "center",
        }}
      >
        {emptyMessage}
      </div>
    );
  }

  return (
    <CWAIReviewPageChecklist
      mode="remediation"
      items={items}
      allItems={items}
      selectable={false}
      selectedItemIds={[]}
      onToggleItemSelection={ignoreDeliverySelection}
      onItemChanged={onChanged}
      onSelectPage={onSelectPage}
      onInjectToRefine={onInjectToRefine}
    />
  );
}
