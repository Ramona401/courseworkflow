package services

import (
	"strings"
	"testing"
)

// TestLessonPlanWordImageDeletionMatchAcceptsDeletedImageBlankLines
// 图片作为独立段落被删除后，前后空行会自然合并；该变化仍属于纯图片删除。
func TestLessonPlanWordImageDeletionMatchAcceptsDeletedImageBlankLines(
	t *testing.T,
) {
	stored :=
		"标题\n\n" +
			"![待删除图片](/uploads/lesson-plans/plan/delete.png)\n\n" +
			"正文"

	current := "标题\n\n正文"

	deleted, matched := matchLessonPlanWordImageDeletionOnly(
		stored,
		current,
	)

	if !matched {
		t.Fatal("图片删除后空行合并应被识别为纯图片删除")
	}

	if len(deleted) != 1 || !deleted[0] {
		t.Fatalf("空行合并场景的删除集合不正确: %#v", deleted)
	}
}

// TestLessonPlanWordImageDeletionMatchAcceptsDeletingAllImages
// 当前正文可以删除原Word中的全部图片，只要文字和其它结构完全不变。
func TestLessonPlanWordImageDeletionMatchAcceptsDeletingAllImages(
	t *testing.T,
) {
	stored := strings.Join(
		[]string{
			"教学目标",
			"![图一](/uploads/lesson-plans/plan/a.png)",
			"教学过程",
			"![图二](/uploads/lesson-plans/plan/b.png)",
			"课堂总结",
		},
		"\n\n",
	)

	current := strings.Join(
		[]string{
			"教学目标",
			"教学过程",
			"课堂总结",
		},
		"\n\n",
	)

	deleted, matched := matchLessonPlanWordImageDeletionOnly(
		stored,
		current,
	)

	if !matched {
		t.Fatal("删除全部图片且文字未变化时应通过校验")
	}

	if len(deleted) != 2 || !deleted[0] || !deleted[1] {
		t.Fatalf("删除全部图片时的删除集合不正确: %#v", deleted)
	}
}

// TestLessonPlanWordImageDeletionMatchRejectsTextMutation
// 即使同时删除了图片，只要正文文字有变化，就不得走保真派生下载。
func TestLessonPlanWordImageDeletionMatchRejectsTextMutation(
	t *testing.T,
) {
	stored := strings.Join(
		[]string{
			"教学目标",
			"![图一](/uploads/lesson-plans/plan/a.png)",
			"教学过程",
		},
		"\n\n",
	)

	current := strings.Join(
		[]string{
			"教学目标已经修改",
			"教学过程",
		},
		"\n\n",
	)

	_, matched := matchLessonPlanWordImageDeletionOnly(
		stored,
		current,
	)

	if matched {
		t.Fatal("正文文字发生变化时必须拒绝图片删除型派生下载")
	}
}

// TestLessonPlanWordImageDeletionMatchRejectsImageMutation
// 修改图片URL或alt文本属于图片替换/修改，不属于删除。
func TestLessonPlanWordImageDeletionMatchRejectsImageMutation(
	t *testing.T,
) {
	stored := strings.Join(
		[]string{
			"教学目标",
			"![原图](/uploads/lesson-plans/plan/a.png)",
			"教学过程",
			"![保留图](/uploads/lesson-plans/plan/b.png)",
		},
		"\n\n",
	)

	current := strings.Join(
		[]string{
			"教学目标",
			"![已修改](/uploads/lesson-plans/plan/changed.png)",
			"教学过程",
		},
		"\n\n",
	)

	_, matched := matchLessonPlanWordImageDeletionOnly(
		stored,
		current,
	)

	if matched {
		t.Fatal("图片Token发生修改时必须拒绝自动派生下载")
	}
}

// TestLessonPlanWordImageDeletionMatchRejectsAmbiguousDuplicateImages
// 两张完全相同且位于同一文字锚点之间的图片无法唯一确认删除位置。
func TestLessonPlanWordImageDeletionMatchRejectsAmbiguousDuplicateImages(
	t *testing.T,
) {
	imageToken := "![重复图片](/uploads/lesson-plans/plan/same.png)"

	stored := strings.Join(
		[]string{
			"标题",
			imageToken,
			imageToken,
			"正文",
		},
		"\n\n",
	)

	current := strings.Join(
		[]string{
			"标题",
			imageToken,
			"正文",
		},
		"\n\n",
	)

	_, matched := matchLessonPlanWordImageDeletionOnly(
		stored,
		current,
	)

	if matched {
		t.Fatal("无法唯一确定删除位置的重复图片必须拒绝自动处理")
	}
}
