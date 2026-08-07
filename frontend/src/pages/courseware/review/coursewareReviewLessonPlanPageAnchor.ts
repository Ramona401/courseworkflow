/**
 * coursewareReviewLessonPlanPageAnchor.ts
 *
 * 解析上传DOCX原文中由课件生成链写出的页级设计锚点。
 *
 * 常见形式：
 *   【P04 / 21 · 神秘助手：智能手环】 区域：全屏 | ...
 *   [P04/21 · 神秘助手：智能手环] ...
 *
 * 该锚点只用于来源教案只读对照中的精确滚动，不作为课件页身份、
 * 权限、审核决定或页面写入依据。正式课件身份仍以服务端稳定page_id为准。
 */

export interface CoursewareReviewLessonPlanPageAnchor {
  pageNumber: number;
  totalPages: number;
  title: string;
}

const pageAnchorPattern =
  /^[【[]\s*P\s*0*([1-9][0-9]*)\s*\/\s*([1-9][0-9]*)\s*[·•・]\s*([^】\]]{1,120})[】\]]/i;

/**
 * 从一行DOCX纯文本中读取课件页锚。
 *
 * 行尾允许继续跟“区域 / 画面内容 / 交互 / 衔接 / 资源”等设计信息；
 * 只解析开头受括号保护的页码、总页数和页标题。
 */
export function parseCoursewareReviewLessonPlanPageAnchor(
  raw: string,
): CoursewareReviewLessonPlanPageAnchor | null {
  const line = raw.trim();
  if (!line) return null;

  const match = line.match(pageAnchorPattern);
  if (!match) return null;

  const pageNumber = Number.parseInt(match[1], 10);
  const totalPages = Number.parseInt(match[2], 10);
  const title = cleanPageAnchorTitle(match[3]);

  if (
    !Number.isFinite(pageNumber) ||
    !Number.isFinite(totalPages) ||
    pageNumber <= 0 ||
    totalPages <= 0 ||
    pageNumber > totalPages ||
    !title
  ) {
    return null;
  }

  return {
    pageNumber,
    totalPages,
    title,
  };
}

/** 给审核抽屉展示一个不包含内部设计字段的页锚标题。 */
export function formatCoursewareReviewLessonPlanPageAnchorTitle(
  anchor: CoursewareReviewLessonPlanPageAnchor,
): string {
  const width = Math.max(
    2,
    String(anchor.totalPages).length,
  );

  return `P${String(anchor.pageNumber).padStart(width, "0")} · ${anchor.title}`;
}

function cleanPageAnchorTitle(
  value: string,
): string {
  return [...value.trim()]
    .slice(0, 120)
    .join("");
}
