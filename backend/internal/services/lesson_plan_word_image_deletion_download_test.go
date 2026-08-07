package services

import (
	"strings"
	"testing"
)

// TestLessonPlanWordImageDeletionSelectionAcceptsImageOnlyRemoval
// 验证结构快照中的图片顺序可以正确映射到Markdown删除结果。
func TestLessonPlanWordImageDeletionSelectionAcceptsImageOnlyRemoval(
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
			"![图二](/uploads/lesson-plans/plan/b.png)",
			"课堂总结",
		},
		"\n\n",
	)

	occurrences := []lessonPlanWordImageOccurrence{
		{
			GlobalIndex:    0,
			RelationshipID: "rId1",
			MarkdownToken:  "![图一](/uploads/lesson-plans/plan/a.png)",
			ImageURL:       "/uploads/lesson-plans/plan/a.png",
		},
		{
			GlobalIndex:    1,
			RelationshipID: "rId2",
			MarkdownToken:  "![图二](/uploads/lesson-plans/plan/b.png)",
			ImageURL:       "/uploads/lesson-plans/plan/b.png",
		},
	}

	deleted, err := selectLessonPlanWordDeletedImageOccurrences(
		stored,
		current,
		occurrences,
	)
	if err != nil {
		t.Fatalf("纯图片删除应通过结构映射校验: %v", err)
	}

	if len(deleted) != 1 || !deleted[0] {
		t.Fatalf("删除的Word图片运行集合不正确: %#v", deleted)
	}
}

// TestLessonPlanWordImageDeletionXMLRemovalPreservesOtherImages
// 验证删除目标drawing时不会误删其它图片容器或正文XML。
func TestLessonPlanWordImageDeletionXMLRemovalPreservesOtherImages(
	t *testing.T,
) {
	documentXML := []byte(
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<w:document xmlns:w="w" xmlns:a="a" xmlns:r="r">` +
			`<w:body>` +
			`<w:p><w:r><w:drawing>` +
			`<a:blip r:embed="rId1"/>` +
			`</w:drawing></w:r></w:p>` +
			`<w:p><w:r><w:drawing>` +
			`<a:blip r:embed="rId2"/>` +
			`</w:drawing></w:r></w:p>` +
			`</w:body></w:document>`,
	)

	modified, err := removeLessonPlanWordImageOccurrences(
		documentXML,
		map[int]bool{0: true},
		2,
	)
	if err != nil {
		t.Fatalf("删除单张Word图片运行失败: %v", err)
	}

	result := string(modified)

	if strings.Contains(result, "rId1") {
		t.Fatal("被删除图片的关系ID仍存在于document.xml")
	}

	if !strings.Contains(result, "rId2") {
		t.Fatal("未删除图片的关系ID被错误移除")
	}

	if strings.Count(result, "<w:drawing") != 1 {
		t.Fatalf("保留的图片容器数量异常: %s", result)
	}
}
