package main

import (
	"strings"
	"testing"
)

func TestBuildReflectionRecoveryMarkdownChangesOnlyExistingSlots(
	t *testing.T,
) {
	current := strings.Join(
		[]string{
			"正文",
			"![图](/image.png)",
			oldReflectionSuffix,
		},
		"\n",
	)

	next, err :=
		buildReflectionRecoveryMarkdown(
			current,
		)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Count(current, "\n") !=
		strings.Count(next, "\n") {
		t.Fatal("换行数量发生变化")
	}
	if !strings.HasPrefix(
		next,
		"正文\n![图](/image.png)\n",
	) {
		t.Fatal("教后反思之前的正文被改变")
	}
	if len(
		lessonPlanRecoveryImagePattern.
			FindAllString(next, -1),
	) != 1 {
		t.Fatal("图片标记数量发生变化")
	}
	if err :=
		validateRecoveredReflection(
			next,
		); err != nil {
		t.Fatal(err)
	}
}

func TestBuildReflectionRecoveryMarkdownRejectsWrongBaseline(
	t *testing.T,
) {
	_, err :=
		buildReflectionRecoveryMarkdown(
			"正文已经变化",
		)
	if err == nil {
		t.Fatal("错误基线应被拒绝")
	}
}

func TestBuildReflectionRecoveryMarkdownExcludesSuggestion(
	t *testing.T,
) {
	next, err :=
		buildReflectionRecoveryMarkdown(
			oldReflectionSuffix,
		)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(
		next,
		reflectionSuggestionMarker,
	) {
		t.Fatal("补充建议被错误写入正文")
	}
}
