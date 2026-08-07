package services

import "testing"

func TestExtractLessonPlanWordRequestedBlocksUsesStructuralPosition(t *testing.T) {
	document := LessonPlanWordPreviewDocument{
		Flow: []LessonPlanWordPreviewFlowItem{
			{Kind: "block", BlockID: "b0"},
			{Kind: "table", TableIndex: 0},
			{Kind: "block", BlockID: "b3"},
		},
		Blocks: []LessonPlanWordPreviewBlock{
			{ID: "b0", Markdown: "相同", ParagraphIndex: 1, Editable: true},
			{ID: "b1", Markdown: "目标", ParagraphIndex: 2, Editable: true},
			{ID: "b2", Markdown: "相同", ParagraphIndex: 3, Editable: true},
			{ID: "b3", Markdown: "结尾", ParagraphIndex: 4, Editable: true},
		},
		Tables: []LessonPlanWordPreviewTable{{
			Index: 0,
			Rows: []LessonPlanWordPreviewTableRow{{
				Index: 0,
				Cells: []LessonPlanWordPreviewTableCell{
					{Index: 0, BlockIDs: []string{"b1"}},
					{Index: 1, BlockIDs: []string{"b2"}},
				},
			}},
		}},
	}

	projection, err := projectLessonPlanWordSemanticMarkdown(document)
	if err != nil {
		t.Fatalf("生成结构投影失败: %v", err)
	}

	const current = "相同\n\n## 目标\n\n相同\n\n结尾"
	if projection.Text != current {
		t.Fatalf("投影正文不匹配:\n%s", projection.Text)
	}

	requested := "相同\n\n## 目标\n\n修改后\n\n结尾"
	blocks, err := extractLessonPlanWordRequestedBlocks(
		projection,
		current,
		requested,
	)
	if err != nil {
		t.Fatalf("拆解新正文失败: %v", err)
	}

	if blocks[0] != "相同" {
		t.Fatalf("第一个重复段落被错误修改: %q", blocks[0])
	}
	if blocks[2] != "修改后" {
		t.Fatalf("表格内第二个重复段落定位错误: %q", blocks[2])
	}
}

func TestExtractLessonPlanWordRequestedBlocksRejectsNewLines(t *testing.T) {
	document := LessonPlanWordPreviewDocument{
		Flow: []LessonPlanWordPreviewFlowItem{
			{Kind: "block", BlockID: "b0"},
			{Kind: "block", BlockID: "b1"},
		},
		Blocks: []LessonPlanWordPreviewBlock{
			{ID: "b0", Markdown: "第一段", ParagraphIndex: 1, Editable: true},
			{ID: "b1", Markdown: "第二段", ParagraphIndex: 2, Editable: true},
		},
	}

	projection, err := projectLessonPlanWordSemanticMarkdown(document)
	if err != nil {
		t.Fatal(err)
	}
	_, err = extractLessonPlanWordRequestedBlocks(
		projection,
		"第一段\n\n第二段",
		"第一段\n\n新增段\n\n第二段",
	)
	if err == nil {
		t.Fatal("新增Word段落应被拒绝")
	}
}

func TestReconcileLessonPlanWordBlockDeletesOnlyExplicitImage(t *testing.T) {
	block := LessonPlanWordPreviewBlock{
		ID:             "b0",
		ParagraphIndex: 1,
		Editable:       true,
		Markdown:       "前![图1](/a.png)中![图2](/b.png)后",
		Runs: []LessonPlanWordPreviewRun{
			{Kind: "text", Text: "前"},
			{Kind: "image", RelationshipID: "rId1", MediaTarget: "word/media/a.png"},
			{Kind: "text", Text: "中"},
			{Kind: "image", RelationshipID: "rId2", MediaTarget: "word/media/b.png"},
			{Kind: "text", Text: "后"},
		},
	}

	patch, changed, err := reconcileLessonPlanWordBlock(
		&block,
		"前中![图2](/b.png)后",
	)
	if err != nil {
		t.Fatalf("删除图片失败: %v", err)
	}
	if !changed {
		t.Fatal("删除图片应产生修改")
	}
	if len(patch.DeleteRelationshipIDs) != 1 ||
		patch.DeleteRelationshipIDs[0] != "rId1" {
		t.Fatalf("删除关系错误: %#v", patch.DeleteRelationshipIDs)
	}
	if block.Markdown != "前中![图2](/b.png)后" {
		t.Fatalf("保留图片顺序错误: %q", block.Markdown)
	}
}

func TestReplaceLessonPlanWordRunTextsPreservesStyledEdges(t *testing.T) {
	block := LessonPlanWordPreviewBlock{
		Runs: []LessonPlanWordPreviewRun{
			{Kind: "text", Text: "标题：", Bold: true},
			{Kind: "text", Text: "旧正文"},
			{Kind: "text", Text: "（注）", Italic: true},
		},
	}

	values := replaceLessonPlanWordRunTextsPreservingStyles(
		&block,
		"标题：新正文（注）",
	)
	if len(values) != 3 {
		t.Fatalf("文字运行数量错误: %d", len(values))
	}
	if values[0] != "标题：" || values[1] != "新正文" || values[2] != "（注）" {
		t.Fatalf("文字运行边界错误: %#v", values)
	}
	if !block.Runs[0].Bold || !block.Runs[2].Italic {
		t.Fatal("原有运行格式未保留")
	}
}
