package services

// lesson_plan_word_parser_types.go — DOCX保真解析结构与语义正文辅助

import (
	"encoding/xml"
	"fmt"
	"path"
	"sort"
	"strings"

	"tedna/internal/models"
)

type LessonPlanWordPreviewDocument struct {
	SchemaVersion int                             `json:"schema_version"`
	ParserVersion string                          `json:"parser_version"`
	SourceFormat  string                          `json:"source_format"`
	Flow          []LessonPlanWordPreviewFlowItem `json:"flow"`
	Blocks        []LessonPlanWordPreviewBlock    `json:"blocks"`
	Tables        []LessonPlanWordPreviewTable    `json:"tables"`
	Media         []LessonPlanWordPreviewMedia    `json:"media"`
	Formulas      []LessonPlanWordPreviewFormula  `json:"formulas"`
}

type LessonPlanWordPreviewFlowItem struct {
	Kind       string `json:"kind"`
	BlockID    string `json:"block_id,omitempty"`
	TableIndex int    `json:"table_index"`
}

type LessonPlanWordPreviewBlock struct {
	ID             string                     `json:"id"`
	Kind           string                     `json:"kind"`
	Text           string                     `json:"text"`
	Markdown       string                     `json:"markdown"`
	Editable       bool                       `json:"editable"`
	SourcePath     string                     `json:"source_path"`
	ParagraphIndex int                        `json:"paragraph_index"`
	TableIndex     int                        `json:"table_index"`
	RowIndex       int                        `json:"row_index"`
	CellIndex      int                        `json:"cell_index"`
	Runs           []LessonPlanWordPreviewRun `json:"runs"`
}

type LessonPlanWordPreviewRun struct {
	Kind           string `json:"kind"`
	Text           string `json:"text,omitempty"`
	Bold           bool   `json:"bold,omitempty"`
	Italic         bool   `json:"italic,omitempty"`
	Underline      bool   `json:"underline,omitempty"`
	VerticalAlign  string `json:"vertical_align,omitempty"`
	RelationshipID string `json:"relationship_id,omitempty"`
	MediaTarget    string `json:"media_target,omitempty"`
	FormulaID      string `json:"formula_id,omitempty"`
}

type LessonPlanWordPreviewTable struct {
	Index            int                             `json:"index"`
	Nested           bool                            `json:"nested"`
	ParentTableIndex int                             `json:"parent_table_index"`
	ParentRowIndex   int                             `json:"parent_row_index"`
	ParentCellIndex  int                             `json:"parent_cell_index"`
	GridWidths       []int                           `json:"grid_widths"`
	Rows             []LessonPlanWordPreviewTableRow `json:"rows"`
}

type LessonPlanWordPreviewTableRow struct {
	Index int                              `json:"index"`
	Cells []LessonPlanWordPreviewTableCell `json:"cells"`
}

type LessonPlanWordPreviewTableCell struct {
	Index              int      `json:"index"`
	GridSpan           int      `json:"grid_span"`
	VerticalMerge      string   `json:"vertical_merge,omitempty"`
	WidthTwips         int      `json:"width_twips,omitempty"`
	Text               string   `json:"text"`
	BlockIDs           []string `json:"block_ids"`
	NestedTableIndices []int    `json:"nested_table_indices,omitempty"`
}

type LessonPlanWordPreviewMedia struct {
	RelationshipID string `json:"relationship_id"`
	Target         string `json:"target"`
	ContentType    string `json:"content_type"`
	TargetMode     string `json:"target_mode,omitempty"`
	Referenced     bool   `json:"referenced"`
	Missing        bool   `json:"missing"`
}

type LessonPlanWordPreviewFormula struct {
	ID      string `json:"id"`
	BlockID string `json:"block_id"`
	Text    string `json:"text"`
}

type LessonPlanWordPreviewMetrics struct {
	BlockCount                int `json:"block_count"`
	EditableBlockCount        int `json:"editable_block_count"`
	TableCount                int `json:"table_count"`
	NestedTableCount          int `json:"nested_table_count"`
	MergedCellCount           int `json:"merged_cell_count"`
	ImageCount                int `json:"image_count"`
	UniqueImageCount          int `json:"unique_image_count"`
	FormulaCount              int `json:"formula_count"`
	SuperscriptRunCount       int `json:"superscript_run_count"`
	SubscriptRunCount         int `json:"subscript_run_count"`
	UnsupportedObjectCount    int `json:"unsupported_object_count"`
	ExternalRelationshipCount int `json:"external_relationship_count"`
	CharacterCount            int `json:"character_count"`
}

type LessonPlanWordPreviewWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type lessonPlanWordParseResult struct {
	Payload  models.LessonPlanWordParsedPayload
	Document LessonPlanWordPreviewDocument
	Metrics  LessonPlanWordPreviewMetrics
	Warnings []LessonPlanWordPreviewWarning
}

type lessonPlanWordRelationship struct {
	ID         string
	Type       string
	Target     string
	TargetMode string
}

type lessonPlanWordWarningCollector struct {
	items map[string]*LessonPlanWordPreviewWarning
}

func newLessonPlanWordWarningCollector() *lessonPlanWordWarningCollector {
	return &lessonPlanWordWarningCollector{
		items: make(map[string]*LessonPlanWordPreviewWarning),
	}
}

func (collector *lessonPlanWordWarningCollector) add(
	code string,
	message string,
	count int,
) {
	if count <= 0 {
		return
	}

	if existing, ok := collector.items[code]; ok {
		existing.Count += count
		return
	}

	collector.items[code] = &LessonPlanWordPreviewWarning{
		Code:    code,
		Message: message,
		Count:   count,
	}
}

func (collector *lessonPlanWordWarningCollector) list() []LessonPlanWordPreviewWarning {
	codes := make([]string, 0, len(collector.items))
	for code := range collector.items {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	result := make([]LessonPlanWordPreviewWarning, 0, len(codes))
	for _, code := range codes {
		result = append(result, *collector.items[code])
	}
	return result
}

// parseLessonPlanWordDOCX 解析已经安全落盘的DOCX文件。
type lessonPlanWordTableParseContext struct {
	tableIndex int
	rowIndex   int
	cellIndex  int
}

type lessonPlanWordParagraphBuilder struct {
	paragraphIndex int
	tableIndex     int
	rowIndex       int
	cellIndex      int
	runs           []LessonPlanWordPreviewRun
}

type lessonPlanWordRunBuilder struct {
	text          strings.Builder
	bold          bool
	italic        bool
	underline     bool
	verticalAlign string
}

func currentLessonPlanWordTableCell(
	document *LessonPlanWordPreviewDocument,
	stack []lessonPlanWordTableParseContext,
) *LessonPlanWordPreviewTableCell {
	if len(stack) == 0 {
		return nil
	}
	context := stack[len(stack)-1]
	if context.tableIndex < 0 ||
		context.tableIndex >= len(document.Tables) ||
		context.rowIndex < 0 ||
		context.rowIndex >= len(document.Tables[context.tableIndex].Rows) ||
		context.cellIndex < 0 ||
		context.cellIndex >= len(document.Tables[context.tableIndex].Rows[context.rowIndex].Cells) {
		return nil
	}
	return &document.Tables[context.tableIndex].Rows[context.rowIndex].Cells[context.cellIndex]
}

func finalizeLessonPlanWordParagraph(
	builder *lessonPlanWordParagraphBuilder,
	document *LessonPlanWordPreviewDocument,
	metrics *LessonPlanWordPreviewMetrics,
) *LessonPlanWordPreviewBlock {
	if builder == nil {
		return nil
	}

	textParts := make([]string, 0, len(builder.runs))
	markdownParts := make([]string, 0, len(builder.runs))
	editable := false

	for _, run := range builder.runs {
		switch run.Kind {
		case "text":
			if run.Text == "" {
				continue
			}
			editable = true
			textParts = append(textParts, run.Text)
			markdownParts = append(markdownParts, renderLessonPlanWordRunMarkdown(run))
			if run.VerticalAlign == "superscript" {
				metrics.SuperscriptRunCount++
			}
			if run.VerticalAlign == "subscript" {
				metrics.SubscriptRunCount++
			}

		case "image":
			label := "图片"
			if run.MediaTarget != "" {
				label = path.Base(run.MediaTarget)
			}
			textParts = append(textParts, "[图片："+label+"]")
			markdownParts = append(markdownParts, "[图片："+label+"]")

		case "formula":
			textParts = append(textParts, "[公式："+run.Text+"]")
			markdownParts = append(
				markdownParts,
				"{{"+strings.ToUpper(run.FormulaID)+":"+run.Text+"}}",
			)
		}
	}

	text := strings.TrimSpace(strings.Join(textParts, ""))
	markdown := strings.TrimSpace(strings.Join(markdownParts, ""))
	if text == "" && markdown == "" {
		return nil
	}

	blockID := fmt.Sprintf("p-%06d", builder.paragraphIndex)
	kind := "paragraph"
	sourcePath := fmt.Sprintf("body/p/%d", builder.paragraphIndex)

	if builder.tableIndex >= 0 {
		cell := &document.Tables[builder.tableIndex].Rows[builder.rowIndex].Cells[builder.cellIndex]
		paragraphInCell := len(cell.BlockIDs) + 1
		blockID = fmt.Sprintf(
			"t-%03d-r-%03d-c-%03d-p-%03d",
			builder.tableIndex+1,
			builder.rowIndex+1,
			builder.cellIndex+1,
			paragraphInCell,
		)
		kind = "table_cell_paragraph"
		sourcePath = fmt.Sprintf(
			"table/%d/row/%d/cell/%d/p/%d",
			builder.tableIndex+1,
			builder.rowIndex+1,
			builder.cellIndex+1,
			paragraphInCell,
		)
		cell.BlockIDs = append(cell.BlockIDs, blockID)
		if cell.Text == "" {
			cell.Text = text
		} else if text != "" {
			cell.Text += "\n" + text
		}
	}

	if editable {
		metrics.EditableBlockCount++
	}

	return &LessonPlanWordPreviewBlock{
		ID:             blockID,
		Kind:           kind,
		Text:           text,
		Markdown:       markdown,
		Editable:       editable,
		SourcePath:     sourcePath,
		ParagraphIndex: builder.paragraphIndex,
		TableIndex:     builder.tableIndex,
		RowIndex:       builder.rowIndex,
		CellIndex:      builder.cellIndex,
		Runs:           builder.runs,
	}
}

func renderLessonPlanWordRunMarkdown(run LessonPlanWordPreviewRun) string {
	text := run.Text
	if text == "" {
		return ""
	}

	switch run.VerticalAlign {
	case "superscript":
		text = "^{" + text + "}"
	case "subscript":
		text = "_{" + text + "}"
	}

	if run.Bold {
		text = "**" + text + "**"
	} else if run.Italic {
		text = "*" + text + "*"
	}

	return text
}

func buildLessonPlanWordSemanticMarkdown(
	document LessonPlanWordPreviewDocument,
) string {
	blockMap := make(map[string]LessonPlanWordPreviewBlock, len(document.Blocks))
	for _, block := range document.Blocks {
		blockMap[block.ID] = block
	}

	parts := make([]string, 0, len(document.Flow))
	for _, item := range document.Flow {
		switch item.Kind {
		case "block":
			if block, ok := blockMap[item.BlockID]; ok && strings.TrimSpace(block.Markdown) != "" {
				parts = append(parts, strings.TrimSpace(block.Markdown))
			}

		case "table":
			markdown := buildLessonPlanWordTableSemanticMarkdown(
				document,
				item.TableIndex,
				blockMap,
				0,
			)
			if markdown != "" {
				parts = append(parts, markdown)
			}
		}
	}

	return strings.Join(parts, "\n\n")
}

func buildLessonPlanWordTableSemanticMarkdown(
	document LessonPlanWordPreviewDocument,
	tableIndex int,
	blockMap map[string]LessonPlanWordPreviewBlock,
	depth int,
) string {
	if tableIndex < 0 || tableIndex >= len(document.Tables) || depth > 6 {
		return ""
	}

	table := document.Tables[tableIndex]
	parts := make([]string, 0)

	for _, row := range table.Rows {
		cellMarkdown := make([]string, len(row.Cells))
		for cellIndex, cell := range row.Cells {
			paragraphs := make([]string, 0, len(cell.BlockIDs)+len(cell.NestedTableIndices))
			for _, blockID := range cell.BlockIDs {
				if block, ok := blockMap[blockID]; ok && strings.TrimSpace(block.Markdown) != "" {
					paragraphs = append(paragraphs, strings.TrimSpace(block.Markdown))
				}
			}
			for _, nestedTableIndex := range cell.NestedTableIndices {
				nestedMarkdown := buildLessonPlanWordTableSemanticMarkdown(
					document,
					nestedTableIndex,
					blockMap,
					depth+1,
				)
				if nestedMarkdown != "" {
					paragraphs = append(paragraphs, nestedMarkdown)
				}
			}
			cellMarkdown[cellIndex] = strings.Join(paragraphs, "\n")
		}

		if len(cellMarkdown) == 1 {
			if strings.TrimSpace(cellMarkdown[0]) != "" {
				parts = append(parts, strings.TrimSpace(cellMarkdown[0]))
			}
			continue
		}

		if len(cellMarkdown) == 2 {
			label := strings.TrimSpace(cellMarkdown[0])
			content := strings.TrimSpace(cellMarkdown[1])
			if label != "" && len([]rune(label)) <= 40 {
				parts = append(parts, "## "+label)
				if content != "" {
					parts = append(parts, content)
				}
				continue
			}
		}

		rowLines := make([]string, 0)
		for cellIndex, content := range cellMarkdown {
			content = strings.TrimSpace(content)
			if content == "" {
				continue
			}
			rowLines = append(
				rowLines,
				fmt.Sprintf("- 第%d列：%s", cellIndex+1, content),
			)
		}
		if len(rowLines) > 0 {
			parts = append(
				parts,
				fmt.Sprintf("### 表格%d · 第%d行", table.Index+1, row.Index+1),
				strings.Join(rowLines, "\n"),
			)
		}
	}

	return strings.Join(parts, "\n\n")
}

func xmlAttrValue(element xml.StartElement, localName string) string {
	for _, attribute := range element.Attr {
		if attribute.Name.Local == localName {
			return attribute.Value
		}
	}
	return ""
}

func xmlOnOffValue(element xml.StartElement) bool {
	value := strings.ToLower(strings.TrimSpace(xmlAttrValue(element, "val")))
	return value != "0" && value != "false" && value != "off" && value != "no"
}

func normalizeLessonPlanWordRelationshipTarget(target string) string {
	target = strings.ReplaceAll(strings.TrimSpace(target), "\\", "/")
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(path.Clean(target), "/")
	}
	return path.Clean(path.Join("word", target))
}

func inferLessonPlanWordMediaContentType(target string) string {
	switch strings.ToLower(path.Ext(target)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	case ".wmf":
		return "image/x-wmf"
	case ".emf":
		return "image/x-emf"
	default:
		return "application/octet-stream"
	}
}
