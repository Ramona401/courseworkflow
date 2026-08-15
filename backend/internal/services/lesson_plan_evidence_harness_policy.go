package services

import "strings"

// lessonPlanEvidenceSurfaceIssueSignals 只覆盖表层表达/排版质量。
// 这类问题不能冒充证据事实冲突，把整份正式教案丢弃。
var lessonPlanEvidenceSurfaceIssueSignals = []string{
	"空行",
	"markdown",
	"排版",
	"格式规范",
	"段落之间",
	"标题层级",
	"标点",
	"全角",
	"半角",
	"中英文混杂",
	"中英混杂",
	"中英文误混",
	"latex",
	"数学定界符",
	"源码形式",
	"多余空格",
	"多余换行",
	"换行格式",
	"代码块",
	"错别字",
	"拼写",
	"语病",
	"语言表达",
	"表述不顺",
	"英文字母大小写",
}

// normalizeLessonPlanEvidenceVerdictPolicy 只剥离纯表层问题。
// 真正的无依据事实、来源冲突和关键遗漏仍保持 fail-closed。
func normalizeLessonPlanEvidenceVerdictPolicy(
	verdict *lessonPlanEvidenceVerdict,
) {
	if verdict == nil {
		return
	}

	var ignored []string

	verdict.UnsupportedModelAdditions, ignored =
		filterLessonPlanEvidenceIssues(
			verdict.UnsupportedModelAdditions,
			ignored,
		)

	verdict.SourceConflicts, ignored =
		filterLessonPlanEvidenceIssues(
			verdict.SourceConflicts,
			ignored,
		)

	verdict.MissingRequiredEvidence, ignored =
		filterLessonPlanEvidenceIssues(
			verdict.MissingRequiredEvidence,
			ignored,
		)

	if len(ignored) > 0 {
		lpGenLog.Info(
			"正式多证据Harness忽略非证据型表层问题",
			"count", len(ignored),
			"issues", strings.Join(
				limitOutlineGuardList(
					ignored,
					3,
				),
				"；",
			),
		)
	}

	if len(ignored) > 0 &&
		len(verdict.UnsupportedModelAdditions) == 0 &&
		len(verdict.SourceConflicts) == 0 &&
		len(verdict.MissingRequiredEvidence) == 0 {
		verdict.Pass = true
		verdict.RepairInstruction = ""
	}
}

func filterLessonPlanEvidenceIssues(
	items []string,
	ignored []string,
) ([]string, []string) {
	if len(items) == 0 {
		return items, ignored
	}

	kept := make([]string, 0, len(items))

	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}

		if isLessonPlanEvidenceSurfaceIssue(trimmed) {
			ignored = append(ignored, trimmed)
			continue
		}

		kept = append(kept, trimmed)
	}

	return kept, ignored
}

func isLessonPlanEvidenceSurfaceIssue(
	value string,
) bool {
	normalized :=
		strings.ToLower(
			strings.TrimSpace(value),
		)

	if normalized == "" {
		return false
	}

	for _, signal := range lessonPlanEvidenceSurfaceIssueSignals {
		if strings.Contains(
			normalized,
			strings.ToLower(signal),
		) {
			return true
		}
	}

	return false
}

// lessonPlanEvidenceHarnessRejectedPublicMessage 只向教师暴露可执行的业务原因。
func lessonPlanEvidenceHarnessRejectedPublicMessage(
	err error,
) string {
	const fallback = "本轮完整教案没有通过资料一致性校验，系统已阻止保存。请按已确认资料检查后重试。"

	if err == nil {
		return fallback
	}

	detail := strings.TrimSpace(err.Error())

	for _, prefix := range []string{
		ErrLessonPlanEvidenceHarnessRejected.Error() + ":",
		ErrLessonPlanEvidenceHarnessRejected.Error() + "：",
	} {
		if strings.HasPrefix(detail, prefix) {
			detail = strings.TrimSpace(
				strings.TrimPrefix(
					detail,
					prefix,
				),
			)
			break
		}
	}

	if detail == "" ||
		detail == ErrLessonPlanEvidenceHarnessRejected.Error() {
		return fallback
	}

	runes := []rune(detail)
	if len(runes) > 220 {
		detail =
			string(runes[:220]) +
				"…"
	}

	return "本轮完整教案没有通过资料一致性校验，系统已阻止保存。具体原因：" +
		detail +
		"。请按提示调整后重试。"
}
