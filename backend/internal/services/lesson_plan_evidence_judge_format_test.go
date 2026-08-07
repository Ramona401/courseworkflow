package services

import (
	"strings"
	"testing"
)

func TestBuildLessonPlanEvidenceJudgeJSONNormalizePrompt(t *testing.T) {
	raw := "判定通过。候选稿没有发现来源冲突、关键遗漏或无依据新增。"

	prompt :=
		buildLessonPlanEvidenceJudgeJSONNormalizePrompt(
			raw,
		)

	if !strings.Contains(
		prompt,
		"<RAW_JUDGE_VERDICT>",
	) {
		t.Fatal("格式修复提示词缺少Judge原始结论边界")
	}
	if !strings.Contains(prompt, raw) {
		t.Fatal("格式修复提示词没有保留首次Judge结论")
	}
	if strings.Contains(
		prompt,
		"<CANDIDATE_OUTPUT>",
	) {
		t.Fatal("格式修复提示词不应再次携带完整候选教案")
	}
}

func TestBuildLessonPlanEvidenceJudgeJSONNormalizePromptTruncates(t *testing.T) {
	raw := strings.Repeat(
		"判定内容",
		lessonPlanEvidenceJudgeRawNormalizeMaxRunes,
	)

	prompt :=
		buildLessonPlanEvidenceJudgeJSONNormalizePrompt(
			raw,
		)

	if !strings.Contains(
		prompt,
		"Judge原始输出已按安全预算截断",
	) {
		t.Fatal("超长Judge原始输出应按预算截断")
	}
	if len([]rune(prompt)) >
		lessonPlanEvidenceJudgeRawNormalizeMaxRunes+
			500 {
		t.Fatal("格式修复提示词超过预期安全预算")
	}
}

func TestBuildLessonPlanEvidenceJudgeJSONNormalizePromptHandlesEmpty(t *testing.T) {
	prompt :=
		buildLessonPlanEvidenceJudgeJSONNormalizePrompt(
			"   ",
		)

	if !strings.Contains(
		prompt,
		"没有返回任何可解释结论",
	) {
		t.Fatal("空Judge输出应转换成保守的不可解释结论")
	}
}
