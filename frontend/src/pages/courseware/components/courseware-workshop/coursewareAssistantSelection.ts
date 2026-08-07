/**
 * 教学智能体当前页面定位。
 *
 * 课件工坊继续以buildPreviewNum作为唯一页面选择真相源，
 * 本模块只在渲染边界把当前页码映射为稳定page_id。
 *
 * 不新增第二套selectedPageId状态，避免页面排序、删除、
 * 全屏退出回写和工作台Tab之间出现双真相源。
 */

import type {
  PageItem,
} from "./PagePreviewBlock";

export interface CoursewareAssistantSelectedPage {
  pageId: string;
  pageNumber: number;
  pageTitle: string;
}

/**
 * 根据当前选中页码解析稳定页面。
 *
 * selectedPageNumber无效时沿用PagePreviewBlock的首页面兜底；
 * 目标页缺少稳定ID时返回null，绝不退化为可变页码调用后端。
 */
export function resolveCoursewareAssistantSelectedPage(
  pages: PageItem[],
  selectedPageNumber: number,
): CoursewareAssistantSelectedPage | null {
  const selectedPage =
    pages.find(
      (page) =>
        page.page_number ===
        selectedPageNumber,
    ) ||
    pages[0];

  if (!selectedPage) {
    return null;
  }

  const pageId =
    selectedPage.id?.trim() || "";

  if (!pageId) {
    return null;
  }

  return {
    pageId,
    pageNumber:
      selectedPage.page_number,
    pageTitle:
      selectedPage.title,
  };
}
