/**
 * pageNumberCalibration.ts — 课件页码校准计划构建与前端数据校验。
 *
 * 本模块只处理当前胶片条视觉顺序，不调用接口。
 * 真正校准由后端既有 pages/reorder 端点原子完成：
 *   - 按页面ID顺序重写连续页码；
 *   - 同步课件总页数；
 *   - 刷新各页导航栏中的当前页与总页数。
 */

export interface PageNumberCalibrationItem {
  id?: string
  page_number: number
  title: string
}

export interface PageNumberCalibrationPlan {
  pageIds: string[]
  selectedPageNumber: number
  orderText: string
  total: number
  alreadySequential: boolean
}

export type PageNumberCalibrationBuildResult =
  | {
      ok: true
      plan: PageNumberCalibrationPlan
    }
  | {
      ok: false
      message: string
    }

function compactPageTitle(value: string): string {
  const title = value.trim() || '未命名页面'
  return title.length > 28
    ? `${title.slice(0, 28)}…`
    : title
}

/**
 * 根据胶片条当前从左到右顺序构建校准计划。
 *
 * pageIds必须完整且唯一；任何缺失都拒绝提交，
 * 避免后端收到不完整顺序后误判或部分重排。
 */
export function buildPageNumberCalibrationPlan(
  pages: PageNumberCalibrationItem[],
  activePage: number,
): PageNumberCalibrationBuildResult {
  if (pages.length === 0) {
    return {
      ok: false,
      message: '⚠️ 当前没有页面，无法校准页码',
    }
  }

  const pageIds: string[] = []
  const seen = new Set<string>()

  for (const page of pages) {
    const pageId = typeof page.id === 'string'
      ? page.id.trim()
      : ''

    if (!pageId) {
      return {
        ok: false,
        message: '⚠️ 部分页面缺少数据库ID，请刷新页面列表后再校准',
      }
    }

    if (seen.has(pageId)) {
      return {
        ok: false,
        message: '⚠️ 页面列表包含重复ID，请刷新后再校准',
      }
    }

    seen.add(pageId)
    pageIds.push(pageId)
  }

  const selectedIndex = pages.findIndex(
    page => page.page_number === activePage,
  )

  const orderText = pages
    .map((page, index) => (
      `新P${index + 1} ← 原P${page.page_number}《${compactPageTitle(page.title)}》`
    ))
    .join('\n')

  return {
    ok: true,
    plan: {
      pageIds,
      selectedPageNumber: selectedIndex >= 0
        ? selectedIndex + 1
        : 1,
      orderText,
      total: pages.length,
      alreadySequential: pages.every(
        (page, index) => page.page_number === index + 1,
      ),
    },
  }
}

