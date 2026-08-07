/**
 * coursewareReviewLessonPlanMatch.ts
 *
 * 来源教案章节匹配纯函数。
 *
 * 匹配优先级：
 *   1. 当前整改项具有明确页码，且DOCX原文存在同页Pxx锚 → 精确返回该页；
 *   2. 没有页锚或属于整课问题 → 使用既有关键词加权匹配真实章节；
 *   3. 匹配得分不足 → 返回undefined，交给界面展示完整教案供人工对照。
 *
 * “教案正文”全文虚拟节点永远不参与定位，避免伪造精确定位。
 */

import {
  FULL_DOCUMENT_HEADING,
  type LessonDocumentSection,
} from "@/pages/lesson-plans/workshop/components/lessonDocumentStructure";

import {
  parseCoursewareReviewLessonPlanPageAnchor,
} from "./coursewareReviewLessonPlanPageAnchor";

export interface CoursewareLessonPlanMatchRequest {
  pageTitle: string;
  issueTitle: string;
  issueDescription: string;
  confirmedInstruction: string;
  originalSuggestion: string;

  /**
   * 正数表示当前问题对应的课件页。
   *
   * 旧的纯匹配调用方可以省略；正式课件问题上下文会提供该字段。
   */
  pageNumber?: number;
}

interface WeightedKeyword {
  value: string;
  weight: number;
}

const CONTEXT_STOP_WORDS = new Set([
  "问题",
  "页面",
  "课件",
  "教案",
  "内容",
  "当前",
  "需要",
  "进行",
  "相关",
  "修改",
  "整改",
  "建议",
  "要求",
  "教学",
  "学生",
  "教师",
  "活动",
  "设计",
  "本页",
]);

/**
 * 在教案章节中寻找与当前问题最相关的一节。
 *
 * DOCX已经明确写出P01/P02等页映射时，页码就是比关键词更可靠的定位锚；
 * 普通教案不存在页锚时继续使用原关键词算法。
 */
export function findBestLessonPlanSection(
  sections: LessonDocumentSection[],
  request:
    | CoursewareLessonPlanMatchRequest
    | null
    | undefined,
): LessonDocumentSection | undefined {
  if (
    !request ||
    sections.length === 0
  ) {
    return undefined;
  }

  const candidateSections =
    sections.filter(
      (section) =>
        section.headingText !==
        FULL_DOCUMENT_HEADING,
    );

  if (candidateSections.length === 0) {
    return undefined;
  }

  const exactPageSection =
    findExactPageAnchorSection(
      candidateSections,
      request.pageNumber,
    );

  if (exactPageSection) {
    return exactPageSection;
  }

  const keywords =
    buildWeightedKeywords(request);

  if (keywords.length === 0) {
    return undefined;
  }

  let best:
    | {
        section: LessonDocumentSection;
        score: number;
      }
    | undefined;

  const pageTitle =
    normalizeText(request.pageTitle);

  const issueTitle =
    normalizeText(request.issueTitle);

  for (
    const section of candidateSections
  ) {
    const sectionTitle =
      normalizeText(section.title);

    const titleText =
      normalizeText(
        [
          section.title,
          ...section.headingPath,
        ].join(" "),
      );

    const bodyText =
      normalizeText(
        section.bodyMarkdown,
      );

    let score = 0;

    for (const keyword of keywords) {
      if (
        titleText.includes(
          keyword.value,
        )
      ) {
        score += keyword.weight * 10;
      }

      if (
        bodyText.includes(
          keyword.value,
        )
      ) {
        score += keyword.weight;
      }
    }

    if (
      sectionTitle.length >= 2 &&
      pageTitle.length >= 2 &&
      (
        pageTitle.includes(sectionTitle) ||
        sectionTitle.includes(pageTitle)
      )
    ) {
      score += 120;
    }

    if (
      sectionTitle.length >= 2 &&
      issueTitle.length >= 2 &&
      (
        issueTitle.includes(sectionTitle) ||
        sectionTitle.includes(issueTitle)
      )
    ) {
      score += 90;
    }

    if (
      !best ||
      score > best.score
    ) {
      best = {
        section,
        score,
      };
    }
  }

  return (
    best &&
    best.score >= 18
  )
    ? best.section
    : undefined;
}

/**
 * 优先按问题页码寻找DOCX中明确的Pxx页级锚。
 *
 * 这不是模糊推断：只有解析器明确识别到受括号保护的Pxx/总页数格式才会命中。
 * 普通Markdown教案没有这种锚点，因此完全不受影响。
 */
function findExactPageAnchorSection(
  sections: LessonDocumentSection[],
  pageNumber: number | undefined,
): LessonDocumentSection | undefined {
  if (
    !Number.isInteger(pageNumber) ||
    !pageNumber ||
    pageNumber <= 0
  ) {
    return undefined;
  }

  return sections.find((section) => {
    const anchor =
      parseCoursewareReviewLessonPlanPageAnchor(
        section.headingText,
      );

    return (
      anchor?.pageNumber === pageNumber
    );
  });
}

function buildWeightedKeywords(
  request: CoursewareLessonPlanMatchRequest,
): WeightedKeyword[] {
  const sourceValues = [
    {
      value: request.pageTitle,
      weight: 12,
    },
    {
      value: request.issueTitle,
      weight: 10,
    },
    {
      value:
        request.confirmedInstruction,
      weight: 6,
    },
    {
      value:
        request.issueDescription,
      weight: 4,
    },
    {
      value:
        request.originalSuggestion,
      weight: 3,
    },
  ];

  const keywordWeight =
    new Map<string, number>();

  for (const source of sourceValues) {
    for (
      const keyword of
        extractKeywords(source.value)
    ) {
      keywordWeight.set(
        keyword,
        Math.max(
          keywordWeight.get(keyword) ||
            0,
          source.weight,
        ),
      );
    }
  }

  return Array.from(
    keywordWeight.entries(),
  )
    .map(
      ([value, weight]) => ({
        value,
        weight,
      }),
    )
    .sort(
      (left, right) =>
        right.weight -
          left.weight ||
        right.value.length -
          left.value.length,
    )
    .slice(0, 80);
}

function extractKeywords(
  raw: string,
): string[] {
  const segments = raw
    .toLowerCase()
    .split(
      /[\s,，。；;：:、！？!?（）()【】[\]《》<>“”"'`~—_/\\|-]+/,
    )
    .map(normalizeText)
    .filter(
      (value) =>
        value.length >= 2,
    );

  const result =
    new Set<string>();

  for (const segment of segments) {
    if (
      !CONTEXT_STOP_WORDS.has(
        segment,
      )
    ) {
      result.add(
        segment.slice(0, 24),
      );
    }

    if (
      /[\u4e00-\u9fff]/.test(
        segment,
      )
    ) {
      for (
        let size = Math.min(
          6,
          segment.length,
        );
        size >= 2;
        size -= 1
      ) {
        for (
          let index = 0;
          index + size <=
          segment.length;
          index += 1
        ) {
          const chunk =
            segment.slice(
              index,
              index + size,
            );

          if (
            !CONTEXT_STOP_WORDS.has(
              chunk,
            )
          ) {
            result.add(chunk);
          }

          if (
            result.size >= 120
          ) {
            return Array.from(
              result,
            );
          }
        }
      }
    }
  }

  return Array.from(result);
}

function normalizeText(
  raw: string,
): string {
  return (raw || "")
    .toLowerCase()
    .replace(
      /[^\p{L}\p{N}]+/gu,
      "",
    )
    .trim();
}
