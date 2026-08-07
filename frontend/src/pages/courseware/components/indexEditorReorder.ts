/**
 * 课件方案页面排序纯函数。
 *
 * 设计目标：
 * 1. 所有排序均以稳定的页面ID为依据，不以会变化的page_number作为身份。
 * 2. 不直接修改React传入的页面数组或页面对象。
 * 3. 排序完成后，在前端即时把page_number规范化为1至N。
 * 4. 请求发送前严格检查页面ID非空且无重复，避免把不完整顺序提交给后端。
 *
 * 后端仍是最终事实源：
 * - 后端会重新校验页面集合完整性；
 * - 后端会在事务中重排页码并同步导航栏页码和课件总页数；
 * - 本模块只负责构建前端乐观更新所需的新数组。
 */

import type { CoursewarePage } from '@/api/coursewares'

/**
 * 一次页面重排的确定性计划。
 */
export interface CoursewarePageReorderPlan {
  /**
   * 已按目标顺序排列且页码重新编号后的页面数组。
   */
  pages: CoursewarePage[]

  /**
   * 需要提交给后端的完整页面ID顺序。
   */
  pageIds: string[]

  /**
   * 本次被移动的页面。
   */
  movedPage: CoursewarePage

  /**
   * 移动前的数组下标。
   */
  fromIndex: number

  /**
   * 移动后的数组下标。
   */
  toIndex: number
}

/**
 * 根据稳定页面ID查找页面下标。
 *
 * 页面ID为空或没有匹配页面时返回-1。
 */
export function findCoursewarePageIndex(
  pages: CoursewarePage[],
  pageId: string,
): number {
  const normalizedPageId = pageId.trim()

  if (!normalizedPageId) {
    return -1
  }

  return pages.findIndex(
    page => page.id.trim() === normalizedPageId,
  )
}

/**
 * 校验并返回完整页面ID顺序。
 *
 * 任何空ID或重复ID都说明前端页面数据不是一个可安全提交的完整快照，
 * 此时必须拒绝排序，而不是把异常数据交给后端。
 */
function collectValidatedPageIds(
  pages: CoursewarePage[],
): string[] {
  const pageIds = pages.map(
    page => page.id.trim(),
  )

  if (pageIds.some(pageId => !pageId)) {
    throw new Error(
      '页面数据缺少稳定ID，请刷新课件后重试',
    )
  }

  if (new Set(pageIds).size !== pageIds.length) {
    throw new Error(
      '页面数据包含重复ID，请刷新课件后重试',
    )
  }

  return pageIds
}

/**
 * 构建一次完整页面重排计划。
 *
 * 返回null表示：
 * - 起始下标或目标下标越界；
 * - 页面被拖回原位置；
 * - 当前没有页面可排序。
 */
export function buildCoursewarePageReorderPlan(
  pages: CoursewarePage[],
  fromIndex: number,
  toIndex: number,
): CoursewarePageReorderPlan | null {
  if (
    pages.length === 0 ||
    fromIndex < 0 ||
    fromIndex >= pages.length ||
    toIndex < 0 ||
    toIndex >= pages.length ||
    fromIndex === toIndex
  ) {
    return null
  }

  /**
   * 先校验当前页面快照。
   *
   * 即使拖拽下标合法，只要页面身份数据异常，也不能执行乐观排序。
   */
  collectValidatedPageIds(pages)

  const reorderedPages = [...pages]
  const removedPages = reorderedPages.splice(
    fromIndex,
    1,
  )
  const movedPage = removedPages[0]

  if (!movedPage) {
    return null
  }

  reorderedPages.splice(
    toIndex,
    0,
    movedPage,
  )

  /**
   * 对页面对象做浅复制，避免直接修改父组件持有的对象。
   */
  const renumberedPages = reorderedPages.map(
    (page, index) => ({
      ...page,
      page_number: index + 1,
    }),
  )

  const pageIds = collectValidatedPageIds(
    renumberedPages,
  )

  return {
    pages: renumberedPages,
    pageIds,
    movedPage: {
      ...movedPage,
      page_number: toIndex + 1,
    },
    fromIndex,
    toIndex,
  }
}
