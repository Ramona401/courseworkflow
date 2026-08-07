package services

import (
	"strings"
	"testing"
)

func TestApplyLessonPlanWordImportedImageURLs(
	t *testing.T,
) {
	document := LessonPlanWordPreviewDocument{
		Flow: []LessonPlanWordPreviewFlowItem{
			{
				Kind:    "block",
				BlockID: "p-000001",
			},
		},
		Blocks: []LessonPlanWordPreviewBlock{
			{
				ID:       "p-000001",
				Markdown: "旧占位内容",
				Runs: []LessonPlanWordPreviewRun{
					{
						Kind: "text",
						Text: "原子结构示意图\n",
					},
					{
						Kind:           "image",
						RelationshipID: "rId5",
						MediaTarget:    "word/media/image1.png",
					},
				},
			},
		},
	}

	count := applyLessonPlanWordImportedImageURLs(
		&document,
		map[string]string{
			"rId5": "/uploads/lesson-plans/plan-1/image.png",
		},
	)

	if count != 1 {
		t.Fatalf(
			"替换数量错误：actual=%d expected=1",
			count,
		)
	}

	markdown := buildLessonPlanWordSemanticMarkdown(
		document,
	)

	expected := "![image1.png](/uploads/lesson-plans/plan-1/image.png)"

	if !strings.Contains(
		markdown,
		expected,
	) {
		t.Fatalf(
			"语义正文未生成Markdown图片：%s",
			markdown,
		)
	}

	if strings.Contains(
		markdown,
		"[图片：image1.png]",
	) {
		t.Fatalf(
			"语义正文仍残留已处理图片占位符：%s",
			markdown,
		)
	}
}

func TestApplyLessonPlanWordImportedImageURLsKeepsMissingPlaceholder(
	t *testing.T,
) {
	document := LessonPlanWordPreviewDocument{
		Blocks: []LessonPlanWordPreviewBlock{
			{
				ID: "p-000001",
				Runs: []LessonPlanWordPreviewRun{
					{
						Kind:           "image",
						RelationshipID: "rIdMissing",
						MediaTarget:    "word/media/image9.wmf",
					},
				},
			},
		},
	}

	count := applyLessonPlanWordImportedImageURLs(
		&document,
		map[string]string{
			"other": "/uploads/unused.png",
		},
	)

	if count != 0 {
		t.Fatalf(
			"不支持图片不应被替换：%d",
			count,
		)
	}

	if document.Blocks[0].Markdown !=
		"[图片：image9.wmf]" {
		t.Fatalf(
			"不支持图片占位符错误：%s",
			document.Blocks[0].Markdown,
		)
	}
}

func TestApplyLessonPlanWordImportedImageURLsKeepsRelationshipOrder(
	t *testing.T,
) {
	document := LessonPlanWordPreviewDocument{
		Flow: []LessonPlanWordPreviewFlowItem{
			{
				Kind:    "block",
				BlockID: "p-000001",
			},
		},
		Blocks: []LessonPlanWordPreviewBlock{
			{
				ID: "p-000001",
				Runs: []LessonPlanWordPreviewRun{
					{
						Kind:           "image",
						RelationshipID: "rId1",
						MediaTarget:    "word/media/image1.png",
					},
					{
						Kind: "text",
						Text: "\n",
					},
					{
						Kind:           "image",
						RelationshipID: "rId2",
						MediaTarget:    "word/media/image1.png",
					},
				},
			},
		},
	}

	count := applyLessonPlanWordImportedImageURLs(
		&document,
		map[string]string{
			"rId1": "/uploads/lesson-plans/plan-1/first.png",
			"rId2": "/uploads/lesson-plans/plan-1/second.png",
		},
	)

	if count != 2 {
		t.Fatalf(
			"同名图片关系替换数量错误：%d",
			count,
		)
	}

	markdown := buildLessonPlanWordSemanticMarkdown(
		document,
	)

	firstPosition := strings.Index(
		markdown,
		"/first.png",
	)

	secondPosition := strings.Index(
		markdown,
		"/second.png",
	)

	if firstPosition < 0 ||
		secondPosition < 0 ||
		firstPosition >= secondPosition {
		t.Fatalf(
			"两个图片关系没有按Word运行顺序独立保留：%s",
			markdown,
		)
	}
}

func TestDetectLessonPlanWordImportedImage(
	t *testing.T,
) {
	pngData := []byte{
		0x89,
		0x50,
		0x4e,
		0x47,
		0x0d,
		0x0a,
		0x1a,
		0x0a,
	}

	mimeType,
		extension,
		supported :=
		detectLessonPlanWordImportedImage(
			pngData,
		)

	if !supported ||
		mimeType != "image/png" ||
		extension != ".png" {
		t.Fatalf(
			"PNG识别错误：supported=%v mime=%q ext=%q",
			supported,
			mimeType,
			extension,
		)
	}

	svgData := []byte(
		`<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
	)

	_, _, supported =
		detectLessonPlanWordImportedImage(
			svgData,
		)

	if supported {
		t.Fatal(
			"SVG不应作为公开Word图片资产直接提取",
		)
	}
}

func TestSanitizeLessonPlanWordImageLabel(
	t *testing.T,
) {
	actual := sanitizeLessonPlanWordImageLabel(
		"图[1]\n.png",
	)

	if actual != "图_1_ .png" {
		t.Fatalf(
			"图片alt安全化结果错误：%q",
			actual,
		)
	}
}
