/**
 * coursewareReviewLessonPlanStructure.ts
 *
 * 课件审核“来源教案对照”专用的只读文档结构解析器。
 *
 * 设计边界：
 *   - 正常Markdown教案完全复用通用lessonDocumentStructure解析结果；
 *   - 仅当通用解析器退化为唯一“教案正文”虚拟节点时，才启用纯文本增强；
 *   - 增强只服务于审核对照展示和滚动定位，不产生可信的后端章节修改语义；
 *   - DOCX课件设计中明确存在P01/P02等页锚时，将其作为真实的页级展示节点；
 *   - 不修改通用教案编辑解析器，避免影响章节AI修改的前后端同源规则。
 */

import {
  FULL_DOCUMENT_HEADING,
  parseLessonDocumentStructure,
  type LessonDocumentSection,
  type LessonDocumentStructure,
} from "@/pages/lesson-plans/workshop/components/lessonDocumentStructure";

import {
  formatCoursewareReviewLessonPlanPageAnchorTitle,
  parseCoursewareReviewLessonPlanPageAnchor,
} from "./coursewareReviewLessonPlanPageAnchor";

interface ReviewTextLine {
  start: number;
  next: number;
  trim: string;
}

interface ReviewPlainTextHeading {
  line: ReviewTextLine;
  title: string;
  level: number;
}

const circledHeadingPattern =
  /^([①②③④⑤⑥⑦⑧⑨⑩⑪⑫⑬⑭⑮⑯⑰⑱⑲⑳])[ \t]*[、.．：:]?[ \t]*(.+?)\s*$/;

const bracketHeadingPattern =
  /^(?:【([^】]{1,80})】|〔([^〕]{1,80})〕|〖([^〗]{1,80})〗)[：:]?\s*$/;

const structuralHeadingPattern =
  /^(模块|单元|阶段|部分|板块|专题)[一二三四五六七八九十百0-9]*[ \t、.．：:-]*(.+?)\s*$/;

/**
 * 解析课件审核对照使用的教案结构。
 *
 * 普通教案先走正式通用解析器；只有唯一虚拟全文节点时，才尝试从DOCX纯文本中
 * 恢复圈码、中文括号栏目以及课件P01/P02等页级设计锚。
 */
export function parseCoursewareReviewLessonDocumentStructure(
  content: string,
): LessonDocumentStructure {
  const standardStructure =
    parseLessonDocumentStructure(content);

  if (
    !isFullDocumentFallback(
      standardStructure.sections,
    )
  ) {
    return standardStructure;
  }

  const enhancedStructure =
    parseReviewPlainTextStructure(content);

  return enhancedStructure || standardStructure;
}

/** 判断当前节点是否只是通用解析器生成的“全文虚拟节点”。 */
export function isCoursewareReviewFullDocumentFallback(
  section: LessonDocumentSection | undefined,
): boolean {
  return section?.headingText === FULL_DOCUMENT_HEADING;
}

function isFullDocumentFallback(
  sections: LessonDocumentSection[],
): boolean {
  return (
    sections.length === 1 &&
    isCoursewareReviewFullDocumentFallback(
      sections[0],
    )
  );
}

/**
 * 从DOCX纯文本常见格式中恢复展示用章节。
 *
 * 高置信结构包括：
 *   - ① 课程设计说明
 *   - 【课程基本信息】
 *   - 〔教学过程〕
 *   - 模块一：课堂导入
 *   - 【P04 / 21 · 神秘助手：智能手环】区域：...
 *
 * 至少两个标题才认为文档具备可用结构。
 */
function parseReviewPlainTextStructure(
  content: string,
): LessonDocumentStructure | null {
  const lines = splitReviewTextLines(content);
  const headings: ReviewPlainTextHeading[] = [];

  for (const line of lines) {
    const parsed =
      parseReviewPlainTextHeading(line.trim);

    if (!parsed) continue;

    headings.push({
      line,
      title: parsed.title,
      level: parsed.level,
    });
  }

  if (headings.length < 2) {
    return null;
  }

  const preambleMarkdown = content.slice(
    0,
    headings[0].line.start,
  );

  const occurrenceByHeading =
    new Map<string, number>();

  const pathStack: Array<{
    level: number;
    title: string;
  }> = [];

  const sections: LessonDocumentSection[] = [];

  headings.forEach((heading, index) => {
    while (
      pathStack.length > 0 &&
      pathStack[pathStack.length - 1].level >=
        heading.level
    ) {
      pathStack.pop();
    }

    const headingPath = [
      ...pathStack.map((entry) => entry.title),
      heading.title,
    ];

    pathStack.push({
      level: heading.level,
      title: heading.title,
    });

    const headingText = heading.line.trim;

    const occurrence =
      (occurrenceByHeading.get(headingText) || 0) +
      1;

    occurrenceByHeading.set(
      headingText,
      occurrence,
    );

    const endOffset =
      index + 1 < headings.length
        ? headings[index + 1].line.start
        : content.length;

    const contentStartOffset = Math.min(
      heading.line.next,
      endOffset,
    );

    sections.push({
      id: buildReviewSectionID(
        headingText,
        occurrence,
      ),
      title: heading.title,
      headingText,
      level: heading.level,
      headingPath,
      occurrence,
      startOffset: heading.line.start,
      contentStartOffset,
      endOffset,
      bodyMarkdown: content.slice(
        contentStartOffset,
        endOffset,
      ),
      locator: {
        heading_text: headingText,
        occurrence,
      },
    });
  });

  return {
    preambleMarkdown,
    sections,
  };
}

function splitReviewTextLines(
  content: string,
): ReviewTextLine[] {
  if (!content) {
    return [{
      start: 0,
      next: 0,
      trim: "",
    }];
  }

  const lines: ReviewTextLine[] = [];
  let start = 0;

  while (start < content.length) {
    const newlineIndex =
      content.indexOf("\n", start);

    const end =
      newlineIndex >= 0
        ? newlineIndex
        : content.length;

    const next =
      newlineIndex >= 0
        ? newlineIndex + 1
        : content.length;

    const text = content
      .slice(start, end)
      .replace(/\r$/, "");

    lines.push({
      start,
      next,
      trim: text.trim(),
    });

    start = next;
  }

  return lines;
}

function parseReviewPlainTextHeading(
  raw: string,
): {
  title: string;
  level: number;
} | null {
  const line = raw.trim();
  if (!line) return null;

  /*
   * 页级锚优先解析。
   *
   * 它通常位于【课件设计】栏目内部，所以使用level=4，
   * 能自然成为该栏目下面的子节点。
   */
  const pageAnchor =
    parseCoursewareReviewLessonPlanPageAnchor(
      line,
    );

  if (pageAnchor) {
    return {
      title:
        formatCoursewareReviewLessonPlanPageAnchorTitle(
          pageAnchor,
        ),
      level: 4,
    };
  }

  const circledMatch =
    line.match(circledHeadingPattern);

  if (circledMatch) {
    const title =
      cleanReviewHeadingTitle(
        circledMatch[2],
      );

    if (isSafeReviewHeadingTitle(title)) {
      return {
        title,
        level: 2,
      };
    }
  }

  const bracketMatch =
    line.match(bracketHeadingPattern);

  if (bracketMatch) {
    const title =
      cleanReviewHeadingTitle(
        bracketMatch[1] ||
        bracketMatch[2] ||
        bracketMatch[3] ||
        "",
      );

    if (isSafeReviewHeadingTitle(title)) {
      return {
        title,
        level: 3,
      };
    }
  }

  const structuralMatch =
    line.match(structuralHeadingPattern);

  if (structuralMatch) {
    const prefix =
      cleanReviewHeadingTitle(
        structuralMatch[1],
      );

    const title =
      cleanReviewHeadingTitle(
        structuralMatch[2],
      );

    const combined =
      `${prefix}${title ? `：${title}` : ""}`;

    if (
      isSafeReviewHeadingTitle(combined)
    ) {
      return {
        title: combined,
        level: 2,
      };
    }
  }

  return null;
}

function cleanReviewHeadingTitle(
  value: string,
): string {
  return [
    ...value
      .trim()
      .replace(/[：:]$/, "")
      .trim(),
  ]
    .slice(0, 80)
    .join("");
}

function isSafeReviewHeadingTitle(
  value: string,
): boolean {
  const title = value.trim();
  if (!title) return false;

  const length = [...title].length;
  if (length > 80) return false;

  return !/[。！？；]/.test(title);
}

/** 生成仅供审核抽屉DOM、React键与滚动定位使用的确定性ID。 */
function buildReviewSectionID(
  headingText: string,
  occurrence: number,
): string {
  const input =
    `review\u001f${headingText}\u001f${occurrence}`;

  let hash = 2166136261;

  for (
    let index = 0;
    index < input.length;
    index += 1
  ) {
    hash ^= input.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }

  return `cw-review-section-${(
    hash >>> 0
  )
    .toString(16)
    .padStart(8, "0")}`;
}
