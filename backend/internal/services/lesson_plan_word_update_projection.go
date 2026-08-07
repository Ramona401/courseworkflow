package services

// lesson_plan_word_update_projection.go — Word语义正文的确定性来源投影
//
// Word结构中的语义正文并不是简单的段落数组：表格会生成标题、列标签和
// 固定分隔文本。本文件在复刻既有Markdown生成规则的同时，记录每段可编辑
// Markdown在最终正文中的精确字节区间。后续同步只允许改动这些区间，表格
// 包装文字、标题前缀和段落分隔符都属于不可修改骨架。

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type lessonPlanWordSemanticSlot struct {
	BlockIndex int
	Start      int
	End        int
}

type lessonPlanWordProjectedPart struct {
	Text  string
	Slots []lessonPlanWordSemanticSlot
}

type lessonPlanWordSemanticProjection struct {
	Text  string
	Slots []lessonPlanWordSemanticSlot
}

func projectLessonPlanWordSemanticMarkdown(
	document LessonPlanWordPreviewDocument,
) (lessonPlanWordSemanticProjection, error) {
	blockIndexes := make(map[string]int, len(document.Blocks))
	for index := range document.Blocks {
		blockID := strings.TrimSpace(document.Blocks[index].ID)
		if blockID != "" {
			blockIndexes[blockID] = index
		}
	}

	parts := make([]lessonPlanWordProjectedPart, 0, len(document.Flow))
	for _, item := range document.Flow {
		switch item.Kind {
		case "block":
			blockIndex, ok := blockIndexes[item.BlockID]
			if !ok {
				continue
			}
			part := projectLessonPlanWordBlock(document, blockIndex)
			if part.Text != "" {
				parts = append(parts, part)
			}

		case "table":
			part := projectLessonPlanWordTable(
				document,
				item.TableIndex,
				blockIndexes,
				0,
			)
			if part.Text != "" {
				parts = append(parts, part)
			}
		}
	}

	joined := joinLessonPlanWordProjectedParts(parts, "\n\n")
	sort.SliceStable(joined.Slots, func(i, j int) bool {
		return joined.Slots[i].Start < joined.Slots[j].Start
	})

	previousEnd := 0
	seenBlocks := make(map[int]bool)
	for _, slot := range joined.Slots {
		if slot.Start < previousEnd || slot.End < slot.Start || slot.End > len(joined.Text) {
			return lessonPlanWordSemanticProjection{},
				ErrLessonPlanWordStructureChangeUnsupported
		}
		if seenBlocks[slot.BlockIndex] {
			return lessonPlanWordSemanticProjection{},
				ErrLessonPlanWordStructureChangeUnsupported
		}
		seenBlocks[slot.BlockIndex] = true
		previousEnd = slot.End
	}

	return lessonPlanWordSemanticProjection{
		Text:  joined.Text,
		Slots: joined.Slots,
	}, nil
}

func projectLessonPlanWordBlock(
	document LessonPlanWordPreviewDocument,
	blockIndex int,
) lessonPlanWordProjectedPart {
	if blockIndex < 0 || blockIndex >= len(document.Blocks) {
		return lessonPlanWordProjectedPart{}
	}

	markdown := strings.TrimSpace(document.Blocks[blockIndex].Markdown)
	if markdown == "" {
		return lessonPlanWordProjectedPart{}
	}

	return lessonPlanWordProjectedPart{
		Text: markdown,
		Slots: []lessonPlanWordSemanticSlot{{
			BlockIndex: blockIndex,
			Start:      0,
			End:        len(markdown),
		}},
	}
}

func projectLessonPlanWordTable(
	document LessonPlanWordPreviewDocument,
	tableIndex int,
	blockIndexes map[string]int,
	depth int,
) lessonPlanWordProjectedPart {
	if tableIndex < 0 || tableIndex >= len(document.Tables) || depth > 6 {
		return lessonPlanWordProjectedPart{}
	}

	table := document.Tables[tableIndex]
	parts := make([]lessonPlanWordProjectedPart, 0)

	for _, row := range table.Rows {
		cells := make([]lessonPlanWordProjectedPart, len(row.Cells))

		for cellIndex, cell := range row.Cells {
			cellParts := make([]lessonPlanWordProjectedPart, 0,
				len(cell.BlockIDs)+len(cell.NestedTableIndices))

			for _, blockID := range cell.BlockIDs {
				blockIndex, ok := blockIndexes[blockID]
				if !ok {
					continue
				}
				part := projectLessonPlanWordBlock(document, blockIndex)
				if part.Text != "" {
					cellParts = append(cellParts, part)
				}
			}

			for _, nestedTableIndex := range cell.NestedTableIndices {
				part := projectLessonPlanWordTable(
					document,
					nestedTableIndex,
					blockIndexes,
					depth+1,
				)
				if part.Text != "" {
					cellParts = append(cellParts, part)
				}
			}

			cells[cellIndex] = joinLessonPlanWordProjectedParts(cellParts, "\n")
		}

		switch len(cells) {
		case 0:
			continue

		case 1:
			if cells[0].Text != "" {
				parts = append(parts, cells[0])
			}
			continue

		case 2:
			label := strings.TrimSpace(cells[0].Text)
			if label != "" && utf8.RuneCountInString(label) <= 40 {
				parts = append(parts, prefixLessonPlanWordProjectedPart("## ", cells[0]))
				if cells[1].Text != "" {
					parts = append(parts, cells[1])
				}
				continue
			}
		}

		rowLines := make([]lessonPlanWordProjectedPart, 0, len(cells))
		for cellIndex, cell := range cells {
			if strings.TrimSpace(cell.Text) == "" {
				continue
			}
			rowLines = append(
				rowLines,
				prefixLessonPlanWordProjectedPart(
					fmt.Sprintf("- 第%d列：", cellIndex+1),
					cell,
				),
			)
		}
		if len(rowLines) == 0 {
			continue
		}

		parts = append(parts, lessonPlanWordProjectedPart{
			Text: fmt.Sprintf(
				"### 表格%d · 第%d行",
				table.Index+1,
				row.Index+1,
			),
		})
		parts = append(parts, joinLessonPlanWordProjectedParts(rowLines, "\n"))
	}

	return joinLessonPlanWordProjectedParts(parts, "\n\n")
}

func prefixLessonPlanWordProjectedPart(
	prefix string,
	part lessonPlanWordProjectedPart,
) lessonPlanWordProjectedPart {
	result := lessonPlanWordProjectedPart{
		Text:  prefix + part.Text,
		Slots: make([]lessonPlanWordSemanticSlot, len(part.Slots)),
	}
	for index, slot := range part.Slots {
		result.Slots[index] = slot
		result.Slots[index].Start += len(prefix)
		result.Slots[index].End += len(prefix)
	}
	return result
}

func joinLessonPlanWordProjectedParts(
	parts []lessonPlanWordProjectedPart,
	separator string,
) lessonPlanWordProjectedPart {
	result := lessonPlanWordProjectedPart{}
	validParts := make([]lessonPlanWordProjectedPart, 0, len(parts))
	for _, part := range parts {
		if part.Text != "" {
			validParts = append(validParts, part)
		}
	}

	var builder strings.Builder
	for index, part := range validParts {
		if index > 0 {
			builder.WriteString(separator)
		}
		offset := builder.Len()
		builder.WriteString(part.Text)
		for _, slot := range part.Slots {
			slot.Start += offset
			slot.End += offset
			result.Slots = append(result.Slots, slot)
		}
	}
	result.Text = builder.String()
	return result
}

// extractLessonPlanWordRequestedBlocks 使用原语义正文中的固定骨架拆解新正文。
//
// 全局换行数量必须完全一致，避免老师或AI通过新增空行改变Word段落结构。
// 每个可编辑槽之间的表格标签和分隔符必须原样保留。
func extractLessonPlanWordRequestedBlocks(
	projection lessonPlanWordSemanticProjection,
	currentSemantic string,
	requestedSemantic string,
) (map[int]string, error) {
	current := strings.TrimSpace(strings.ReplaceAll(
		strings.ReplaceAll(currentSemantic, "\r\n", "\n"),
		"\r",
		"\n",
	))
	requested := strings.TrimSpace(strings.ReplaceAll(
		strings.ReplaceAll(requestedSemantic, "\r\n", "\n"),
		"\r",
		"\n",
	))

	if projection.Text != current ||
		strings.Count(current, "\n") != strings.Count(requested, "\n") {
		return nil, ErrLessonPlanWordStructureChangeUnsupported
	}

	result := make(map[int]string, len(projection.Slots))
	oldCursor := 0
	requestedCursor := 0

	for slotIndex, slot := range projection.Slots {
		literalBefore := projection.Text[oldCursor:slot.Start]
		if !strings.HasPrefix(requested[requestedCursor:], literalBefore) {
			return nil, ErrLessonPlanWordStructureChangeUnsupported
		}
		requestedCursor += len(literalBefore)

		nextSlotStart := len(projection.Text)
		if slotIndex+1 < len(projection.Slots) {
			nextSlotStart = projection.Slots[slotIndex+1].Start
		}
		literalAfter := projection.Text[slot.End:nextSlotStart]

		var requestedEnd int
		if slotIndex == len(projection.Slots)-1 {
			if !strings.HasSuffix(requested[requestedCursor:], literalAfter) {
				return nil, ErrLessonPlanWordStructureChangeUnsupported
			}
			requestedEnd = len(requested) - len(literalAfter)
		} else {
			if literalAfter == "" {
				return nil, ErrLessonPlanWordStructureChangeUnsupported
			}
			relativeIndex := strings.Index(requested[requestedCursor:], literalAfter)
			if relativeIndex < 0 {
				return nil, ErrLessonPlanWordStructureChangeUnsupported
			}
			requestedEnd = requestedCursor + relativeIndex
		}

		if requestedEnd < requestedCursor {
			return nil, ErrLessonPlanWordStructureChangeUnsupported
		}
		result[slot.BlockIndex] = requested[requestedCursor:requestedEnd]
		requestedCursor = requestedEnd
		oldCursor = slot.End
	}

	trailing := projection.Text[oldCursor:]
	if requested[requestedCursor:] != trailing {
		return nil, ErrLessonPlanWordStructureChangeUnsupported
	}

	return result, nil
}
